package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

var hourChoices = clockChoices(24)
var minuteChoices = clockChoices(60)

func clockChoices(count int) []string {
	choices := make([]string, count)
	for i := range choices {
		choices[i] = fmt.Sprintf("%02d", i)
	}
	return choices
}

func (a *application) dateTimeView(c *ui.Context) {
	if a.draftTime.IsZero() {
		// Start at the next whole local hour rather than an already-past time.
		now := time.Now()
		a.draftTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, time.Local)
	}
	ui.Column(c.Key("datetime-" + a.editingID)).Gap(8).Children(func() {
		ui.Text(c, a.t("Target date and time")).FontSize(12).TextColor(c.Theme().TextMuted)
		ui.Row(c).Gap(8).Children(func() {
			ui.DateInput(c, &a.draftTime).Label(a.t("Date")).Grow(1).Disabled(a.readOnly)
			ui.Button(c, a.t("Today")).Disabled(a.readOnly).OnClick(func() { a.setDraftDay(time.Now()) })
			ui.Button(c, a.t("Tomorrow")).Disabled(a.readOnly).OnClick(func() { a.setDraftDay(time.Now().AddDate(0, 0, 1)) })
		})
		ui.Checkbox(c, &a.draftDateOnly, a.t("Date only (00:00)")).Disabled(a.readOnly)
		if !a.draftDateOnly {
			ui.Row(c).Gap(12).Wrap().Children(func() {
				ui.Row(c).Gap(6).Children(func() {
					ui.Text(c, a.t("Hour"))
					hour := float64(a.draftTime.Hour())
					ui.NumberInput(c.Key("hour-input"), &hour, 0, 23, 1).Width(140).Label(a.t("Hour input")).Disabled(a.readOnly).OnChange(func() {
						a.setDraftClock(int(hour), a.draftTime.Minute())
					})
					choice := hourChoices[a.draftTime.Hour()]
					clockPicker(c.Key("hour-picker"), &choice, hourChoices, a.t("Hour"), a.readOnly).Label(a.t("Choose hour")).OnChange(func() {
						value, _ := strconv.Atoi(choice)
						a.setDraftClock(value, a.draftTime.Minute())
					})
				})
				ui.Row(c).Gap(6).Children(func() {
					ui.Text(c, a.t("Minute"))
					minute := float64(a.draftTime.Minute())
					ui.NumberInput(c.Key("minute-input"), &minute, 0, 59, 1).Width(140).Label(a.t("Minute input")).Disabled(a.readOnly).OnChange(func() {
						a.setDraftClock(a.draftTime.Hour(), int(minute))
					})
					choice := minuteChoices[a.draftTime.Minute()]
					clockPicker(c.Key("minute-picker"), &choice, minuteChoices, a.t("Minute"), a.readOnly).Label(a.t("Choose minute")).OnChange(func() {
						value, _ := strconv.Atoi(choice)
						a.setDraftClock(a.draftTime.Hour(), value)
					})
				})
			})
		}
	})
}

// A bounded grid avoids a 60-row dropdown and lets the user pick any minute
// with one click, including values not divisible by five.
func clockPicker(c *ui.Context, selected *string, choices []string, label string, disabled bool) ui.Element {
	t := c.Theme()
	picker := ui.SelectBase(c, selected)
	picker.Trigger.Width(28).Padding(6).Radius(t.Radius).
		Background(t.Surface).Border(1, t.Border).Label(label).Disabled(disabled).
		Center().Children(func() {
		ui.Text(c, "⌄").TextColor(t.TextMuted)
	})
	picker.Popup(func(panel ui.Element) {
		panel.Width(260).Padding(8).Gap(6).Radius(6).Background(t.Background).Border(1, t.Border)
		ui.Text(c, label).FontSize(12).TextColor(t.TextMuted)
		ui.Grid(c).Columns(6).Gap(4).Children(func() {
			for _, choice := range choices {
				item := picker.Item(choice).Height(28).Shrink(0).Center().Radius(4)
				ink := t.Text
				if choice == *selected || item.Highlighted() {
					item.Background(t.Accent)
					ink = t.AccentText
				}
				item.Children(func() { ui.Text(c, choice).TextColor(ink).FontFeatures("tnum") })
			}
		})
	})
	return picker.Trigger
}

func (a *application) setDraftDay(day time.Time) {
	year, month, date := day.Date()
	a.draftTime = time.Date(year, month, date, a.draftTime.Hour(), a.draftTime.Minute(), 0, 0, time.Local)
}

func (a *application) setDraftClock(hour, minute int) {
	year, month, day := a.draftTime.Date()
	a.draftTime = time.Date(year, month, day, hour, minute, 0, 0, time.Local)
}
