package main

import (
	"fmt"
	"log"
	"path/filepath"
	"slices"
	"time"

	"countdowner/internal/desktop"
	"countdowner/internal/model"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type application struct {
	config                               model.Config
	store                                model.Store
	settings, widget                     *mygo.Window
	controller                           *desktop.Controller
	tray                                 *mygo.Tray
	quitting, readOnly                   bool
	status, desktopStatus                string
	statusError                          bool
	draftTitle, editingID, pendingDelete string
	draftTime                            time.Time
	draftDateOnly                        bool
	draftInput, deleteConfirm            ui.Handle
	settingsButton                       ui.Handle
	widgetDeleteID                       string
}

func (a *application) start() {
	mygo.Theme.SetSource(mygo.ThemeSource(a.config.Theme))
	work := mygo.Screen.PrimaryDisplay().WorkArea
	a.settings = mygo.NewWindow(mygo.WindowOptions{
		Title: "Countdowner — " + a.t("Settings"),
		Width: min(760, max(520, work.Width-80)), Height: min(820, max(420, work.Height-80)),
		MinWidth: 520, MinHeight: 420, Hidden: true, StateKey: "settings",
		Content: ui.View(a.settingsView),
	})
	a.settings.OnClose(func(e *mygo.CloseEvent) {
		if !a.quitting {
			e.PreventDefault()
			a.settings.Hide()
		}
	})
	var err error
	a.tray, err = mygo.NewTray(mygo.TrayOptions{Icon: makeTrayIcon(mygo.Theme.IsDark()), ToolTip: "Countdowner"})
	if err != nil {
		a.fail(err)
		a.showSettings()
	} else {
		a.tray.OnClick(a.showSettings)
		a.tray.OnRightClick(func() {
			a.tray.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
				{Label: a.t("Settings"), Click: func(*mygo.MenuItem, *mygo.Window) { a.showSettings() }},
				mygo.Separator(),
				{Label: a.t("Quit"), Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
			}))
			a.tray.PopUpMenu()
			a.tray.SetMenu(nil)
		})
	}
	mygo.Theme.OnUpdated(func() {
		if a.tray != nil {
			a.tray.SetIcon(makeTrayIcon(mygo.Theme.IsDark()), false)
		}
		a.redraw()
	})
	mygo.Screen.OnDisplaysChanged(a.maintainDesktop)
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { a.quitting = true })
	// Explorer may destroy our cross-process child; this is not an app quit.
	mygo.App.OnWindowAllClosed(func() {})
	a.maintainDesktop()
	if len(a.config.Countdowns) == 0 || a.readOnly || a.statusError {
		a.showSettings()
	}
	done := make(chan struct{})
	mygo.App.OnQuit(func() { close(done) })
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mygo.RunOnMain(a.maintainDesktop)
			case <-done:
				return
			}
		}
	}()
}

func (a *application) showSettings() {
	a.cancelWidgetDelete()
	if a.settings != nil {
		a.settings.Show()
		a.settings.Focus()
	}
}

func widgetHeight(count int) int { return min(620, 100+max(1, count)*96) }

func (a *application) widgetSize() model.Size {
	if a.config.WidgetSize.Width > 0 {
		return model.Size{
			Width:  max(desktop.MinWidth, a.config.WidgetSize.Width),
			Height: max(desktop.MinHeight, a.config.WidgetSize.Height),
		}
	}
	return model.Size{Width: 340, Height: widgetHeight(len(a.config.Countdowns))}
}

func (a *application) saveWidgetBounds(position model.Position, size model.Size) {
	if a.quitting || a.readOnly {
		return
	}
	next := a.config
	next.Position, next.WidgetSize = position, size
	a.commit(next)
}

func (a *application) maintainDesktop() {
	if a.quitting {
		return
	}
	var err error
	if a.widget == nil {
		if !desktop.Available() {
			a.setDesktopStatus("Waiting for Explorer desktop")
			return
		}
		size := a.widgetSize()
		w := mygo.NewWindow(mygo.WindowOptions{
			Title: "Countdowner Widget", Width: size.Width, Height: size.Height,
			Hidden: true, Frameless: true, SkipTaskbar: true, DisableShadow: true,
			DisableResize: true, DisableMinimize: true, DisableMaximize: true, DisableClose: true,
			Content: ui.View(a.widgetView),
		})
		a.widget = w
		w.OnClosed(func() {
			if a.widget == w {
				a.widget, a.controller = nil, nil
			}
		})
		controller := &desktop.Controller{
			OnManipulate: a.cancelWidgetDelete,
			OnResize:     a.saveWidgetBounds,
			OnPosition: func(position model.Position) {
				next := a.config
				next.Position = position
				if next.WidgetSize.Width > 0 {
					next.WidgetSize = a.controller.Size()
				}
				if (next.Position != a.config.Position || next.WidgetSize != a.config.WidgetSize) && !a.quitting && !a.readOnly {
					a.commit(next)
				}
			},
			OnRedraw: a.redraw,
		}
		a.controller = controller
		err = controller.Attach(w.NativeHandle(), a.config.Position, size.Width, size.Height)
		if err != nil {
			w.Destroy() // Never leave a failed desktop widget as a floating window.
		}
	} else {
		err = a.controller.Ensure()
	}
	if err != nil {
		a.setDesktopStatus(err.Error())
	} else {
		a.setDesktopStatus("Desktop connected")
	}
}

func (a *application) setDesktopStatus(value string) {
	if value != a.desktopStatus {
		log.Print(value)
		a.desktopStatus = value
		a.redraw()
	}
}

func (a *application) fail(err error) {
	log.Print(err)
	a.status, a.statusError = err.Error(), true
	a.redraw()
}

func (a *application) redraw() {
	if a.settings != nil {
		a.settings.Invalidate()
	}
	if a.widget != nil {
		a.widget.Invalidate()
	}
}

func (a *application) commit(next model.Config) bool {
	if a.readOnly {
		a.fail(fmt.Errorf("Configuration is read-only; the original file is preserved"))
		return false
	}
	if err := a.store.Save(next); err != nil {
		a.fail(err)
		return false
	}
	previous := a.config
	a.config = next
	a.status, a.statusError = "Saved", false
	if previous.Theme != next.Theme {
		mygo.Theme.SetSource(mygo.ThemeSource(next.Theme))
	}
	if a.settings != nil && previous.Language != next.Language {
		a.settings.SetTitle("Countdowner — " + a.t("Settings"))
	}
	if a.controller != nil && (previous.WidgetSize != next.WidgetSize ||
		(next.WidgetSize.Width == 0 && len(previous.Countdowns) != len(next.Countdowns))) {
		size := a.widgetSize()
		if size != a.controller.Size() {
			if err := a.controller.SetSize(size.Width, size.Height); err != nil {
				a.setDesktopStatus(err.Error())
			}
		}
	}
	a.redraw()
	return true
}

func (a *application) saveDraft() {
	layout := "2006-01-02 15:04"
	if a.draftDateOnly {
		layout = "2006-01-02"
	}
	item, err := model.NewCountdown(a.draftTitle, a.draftTime.Format(layout), time.Local)
	if err != nil {
		a.fail(err)
		return
	}
	next := a.config
	next.Countdowns = slices.Clone(next.Countdowns)
	if a.editingID == "" {
		next.Countdowns = append(next.Countdowns, item)
	} else {
		index := slices.IndexFunc(next.Countdowns, func(item model.Countdown) bool { return item.ID == a.editingID })
		if index < 0 {
			return
		}
		item.ID = a.editingID
		// Editing just the title must preserve the exact stored instant,
		// including a repeated local time around a DST transition.
		old := next.Countdowns[index]
		midnight := a.draftTime.Hour() == 0 && a.draftTime.Minute() == 0 &&
			a.draftTime.Second() == 0 && a.draftTime.Nanosecond() == 0
		if a.draftTime.Equal(old.Target) && (!a.draftDateOnly || midnight) {
			item.Target, item.TimeZone = old.Target, old.TimeZone
		}
		next.Countdowns[index] = item
	}
	if a.commit(next) {
		a.clearDraft()
	}
}

func (a *application) clearDraft() {
	a.draftTitle, a.editingID = "", ""
	a.draftTime = time.Time{}
	a.draftDateOnly = false
}

func (a *application) edit(item model.Countdown) {
	a.draftInput.Focus()
	a.editingID, a.draftTitle = item.ID, item.Title
	a.draftTime = item.Target.In(time.Local)
	a.draftDateOnly = a.draftTime.Hour() == 0 && a.draftTime.Minute() == 0 &&
		a.draftTime.Second() == 0 && a.draftTime.Nanosecond() == 0
}

func (a *application) remove(id string) {
	next := a.config
	next.Countdowns = slices.DeleteFunc(slices.Clone(next.Countdowns), func(item model.Countdown) bool { return item.ID == id })
	if a.commit(next) {
		a.pendingDelete = ""
		if a.widgetDeleteID == id {
			a.widgetDeleteID = ""
		}
		if a.editingID == id {
			a.clearDraft()
		}
	}
}

func (a *application) move(id string, direction int) {
	if a.config.Sort != "manual" {
		return
	}
	next := a.config
	next.Countdowns = slices.Clone(next.Countdowns)
	index := slices.IndexFunc(next.Countdowns, func(item model.Countdown) bool { return item.ID == id })
	other := index + direction
	if index < 0 || other < 0 || other >= len(next.Countdowns) {
		return
	}
	next.Countdowns[index], next.Countdowns[other] = next.Countdowns[other], next.Countdowns[index]
	a.commit(next)
}

func (a *application) setAutostart(enabled bool) {
	previous := mygo.App.OpenAtLogin()
	if err := mygo.App.SetOpenAtLogin(enabled); err != nil {
		a.fail(err)
		return
	}
	next := a.config
	next.Autostart = enabled
	if !a.commit(next) {
		if err := mygo.App.SetOpenAtLogin(previous); err != nil {
			a.fail(err)
		}
	}
}

func (a *application) configureTheme(c *ui.Context, scale float32) *ui.Theme {
	t := *c.Theme()
	t.Font = a.fontFamily()
	t.FontSize *= scale
	t.Spacing *= scale
	t.Radius = 4 * scale
	// Neutral monochrome controls, with high-contrast active states.
	t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = t.Text, t.TextMuted, t.Text, t.Background
	c.SetTheme(&t)
	return &t
}

func (a *application) remaining(item model.Countdown, now time.Time) string {
	return a.t(model.Remaining(item.Target, now))
}

func (a *application) widgetView(c *ui.Context) {
	scale := float32(1)
	if a.controller != nil {
		scale = a.controller.VisualScale()
	}
	t := a.configureTheme(c, scale)
	width, height := c.Size()
	width, height = width/scale, height/scale
	// Keep text readable at small sizes; overflow remains scrollable.
	contentScale := min(width/340, (height-84)/float32(max(1, len(a.config.Countdowns))*86))
	contentScale = min(float32(1.5), max(float32(0.8), contentScale))
	padding := min(float32(24), max(float32(12), width*18/340)) * scale
	c.Root().Background(t.Background)
	a.refreshCountdowns(c)
	now := c.Now()
	ui.Column(c).Fill().Padding(padding).Gap(10 * scale).Children(func() {
		a.widgetHeader(c, scale)
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Gap(16 * scale * contentScale).Children(func() {
			items := a.config.Ordered()
			if len(items) == 0 {
				ui.Text(c, a.t("No countdowns yet")).FontSize(17 * scale)
				ui.Text(c, a.t("Open settings from the tray to add a deadline")).FontSize(11 * scale).TextColor(t.TextMuted)
			}
			a.widgetItems(c, scale, contentScale, now)
		})
	})
}

func (a *application) settingsView(c *ui.Context) {
	t := a.configureTheme(c, 1)
	c.Root().Background(t.Background)
	a.refreshCountdowns(c)
	ui.Column(c).Fill().Children(func() {
		ui.Column(c).Padding(24, 28, 16).Gap(4).Children(func() {
			ui.Text(c, "Countdowner").FontSize(28).Bold()
			ui.Text(c, a.t("Your deadlines, on your desktop")).TextColor(t.TextMuted)
		})
		ui.Scroll(c).Grow(1).Padding(0, 28, 16).Gap(18).Children(func() {
			a.preferencesView(c)
			ui.Divider(c)
			a.draftView(c)
			ui.Divider(c)
			a.countdownsView(c)
			ui.Divider(c)
			ui.Text(c, a.t(a.desktopStatus)).FontSize(12).TextColor(t.TextMuted)
			ui.Text(c, a.t("Data folder")+": "+filepath.Dir(a.store.Path)).FontSize(11).TextColor(t.TextMuted).Selectable()
			ui.Button(c, a.t("Reset desktop position")).Disabled(a.readOnly).OnClick(func() {
				if a.controller != nil {
					if err := a.controller.Restore(model.Position{X: 40, Y: 40}); err != nil {
						a.fail(err)
					}
				}
			})
			ui.Divider(c)
			a.aboutView(c)
		})
		ui.Column(c).Padding(12, 28).Gap(4).Background(t.Surface).Children(func() {
			ink := t.TextMuted
			if a.statusError {
				ink = t.Danger
			}
			if a.status != "" {
				ui.Text(c, a.t(a.status)).TextColor(ink).FontSize(12)
			}
			ui.Text(c, a.t("Automatically saved. Closing this window keeps the widget running.")).FontSize(11).TextColor(t.TextMuted)
		})
	})
}

func (a *application) preferencesView(c *ui.Context) {
	ui.Column(c).Gap(10).Children(func() {
		ui.Row(c).Gap(12).Children(func() {
			ui.Text(c, a.t("Language")).Width(100)
			index := slices.Index(languageCodes, a.config.Language)
			language := languageNames[max(0, index)]
			ui.Select(c, &language, languageNames).Grow(1).Disabled(a.readOnly).OnChange(func() {
				next := a.config
				next.Language = languageCodes[slices.Index(languageNames, language)]
				a.commit(next)
			})
		})
		ui.Row(c).Gap(12).Children(func() {
			ui.Text(c, a.t("Theme")).Width(100)
			codes := []string{"system", "light", "dark"}
			labels := []string{a.t("Follow system"), a.t("Light"), a.t("Dark")}
			choice := labels[slices.Index(codes, a.config.Theme)]
			ui.Select(c, &choice, labels).Grow(1).Disabled(a.readOnly).OnChange(func() {
				next := a.config
				next.Theme = codes[slices.Index(labels, choice)]
				a.commit(next)
			})
		})
		autostart := a.config.Autostart
		ui.Checkbox(c, &autostart, a.t("Start at login")).Disabled(a.readOnly).OnChange(func() { a.setAutostart(autostart) })
	})
}

func (a *application) draftView(c *ui.Context) {
	ui.Column(c).Gap(10).Children(func() {
		heading, action := "New countdown", "Add countdown"
		if a.editingID != "" {
			heading, action = "Edit countdown", "Save changes"
		}
		ui.Text(c, a.t(heading)).FontSize(18).Bold()
		ui.TextInput(c.Key("title-"+a.editingID), &a.draftTitle).Bind(&a.draftInput).Placeholder(a.t("Title")).Label(a.t("Title")).Disabled(a.readOnly)
		a.dateTimeView(c)
		ui.Text(c, a.t("Date only means 00:00. Times use your local timezone.")).FontSize(11).TextColor(c.Theme().TextMuted)
		ui.Row(c).Gap(8).Children(func() {
			ui.PrimaryButton(c, a.t(action)).Disabled(a.readOnly).OnClick(a.saveDraft)
			if a.editingID != "" {
				ui.Button(c, a.t("Cancel")).OnClick(a.clearDraft)
			}
		})
	})
}

func (a *application) countdownsView(c *ui.Context) {
	ui.Column(c).Gap(10).Children(func() {
		ui.Row(c).Gap(12).Children(func() {
			ui.Text(c, a.t("Countdowns")).FontSize(18).Bold().Grow(1)
			codes := []string{"manual", "deadline"}
			labels := []string{a.t("Manual order"), a.t("Deadline, earliest first")}
			choice := labels[slices.Index(codes, a.config.Sort)]
			ui.Select(c, &choice, labels).Width(230).Disabled(a.readOnly).OnChange(func() {
				next := a.config
				next.Sort = codes[slices.Index(labels, choice)]
				a.commit(next)
			})
		})
		items := a.config.Ordered()
		if len(items) == 0 {
			ui.Text(c, a.t("No countdowns yet")).TextColor(c.Theme().TextMuted)
		}
		for index, item := range items {
			ui.Column(c.Key(item.ID)).Padding(12).Gap(8).Radius(4).Background(c.Theme().Surface).Children(func() {
				ui.Row(c).Gap(12).Children(func() {
					ui.Text(c, item.Title).Grow(1).SingleLine()
					ui.Text(c, a.remaining(item, c.Now())).Bold().FontFeatures("tnum")
				})
				ui.Text(c, item.Target.In(time.Local).Format("2006-01-02 15:04 MST")).FontSize(12).TextColor(c.Theme().TextMuted)
				ui.Row(c).Gap(6).Wrap().Children(func() {
					ui.Row(c).Gap(6).Shrink(0).Children(func() {
						fullLabelButton(c, a.t("Edit")).Disabled(a.readOnly).OnClick(func() { a.edit(item) })
						ui.Button(c, "↑").Shrink(0).Label(a.t("Move up")).Disabled(a.readOnly || a.config.Sort != "manual" || index == 0).OnClick(func() { a.move(item.ID, -1) })
						ui.Button(c, "↓").Shrink(0).Label(a.t("Move down")).Disabled(a.readOnly || a.config.Sort != "manual" || index == len(items)-1).OnClick(func() { a.move(item.ID, 1) })
					})
					ui.Row(c).Gap(6).Shrink(0).Margin(0, 0, 0, ui.Auto).Children(func() {
						if a.pendingDelete == item.ID {
							fullLabelButton(c, a.t("Confirm delete")).Bind(&a.deleteConfirm).Disabled(a.readOnly).OnClick(func() { a.remove(item.ID) })
							fullLabelButton(c, a.t("Cancel")).OnClick(func() { a.pendingDelete = "" })
						} else {
							fullLabelButton(c, a.t("Delete")).Disabled(a.readOnly).OnClick(func() { a.pendingDelete = item.ID })
						}
					})
				})
			})
		}
	})
}
