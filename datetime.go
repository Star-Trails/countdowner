package main

import (
	"time"

	"github.com/egoist/mygo/ui"
)

func (a *application) dateTimeView(c *ui.Context) {
	if a.draftTime.IsZero() {
		// Start at the next whole local hour rather than an already-past time.
		now := time.Now()
		a.draftTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, time.Local)
	}
	ui.Column(c.Key("datetime-" + a.editingID)).Gap(8).Children(func() {
		ui.Text(c, a.text().TargetDateAndTime()).FontSize(12).TextColor(c.Theme().TextMuted)
		ui.Row(c).Gap(8).Children(func() {
			ui.DateInput(c, &a.draftTime).Label(a.text().Date()).Grow(1).Disabled(a.readOnly)
			ui.Button(c, a.text().Today()).Disabled(a.readOnly).OnClick(func() { a.setDraftDay(time.Now()) })
			ui.Button(c, a.text().Tomorrow()).Disabled(a.readOnly).OnClick(func() { a.setDraftDay(time.Now().AddDate(0, 0, 1)) })
		})
		ui.Checkbox(c, &a.draftDateOnly, a.text().DateOnly()).Disabled(a.readOnly)
		if !a.draftDateOnly {
			ui.Row(c).Gap(12).Wrap().Children(func() {
				ui.Row(c).Gap(6).Children(func() {
					ui.Text(c, a.text().Hour())
					hour := float64(a.draftTime.Hour())
					ui.NumberInput(c.Key("hour-input"), &hour, 0, 23, 1).Width(140).Label(a.text().HourInput()).Disabled(a.readOnly).OnChange(func() {
						a.setDraftClock(int(hour), a.draftTime.Minute())
					})
				})
				ui.Row(c).Gap(6).Children(func() {
					ui.Text(c, a.text().Minute())
					minute := float64(a.draftTime.Minute())
					ui.NumberInput(c.Key("minute-input"), &minute, 0, 59, 1).Width(140).Label(a.text().MinuteInput()).Disabled(a.readOnly).OnChange(func() {
						a.setDraftClock(a.draftTime.Hour(), int(minute))
					})
				})
			})
		}
	})
}

func (a *application) setDraftDay(day time.Time) {
	year, month, date := day.Date()
	a.draftTime = time.Date(year, month, date, a.draftTime.Hour(), a.draftTime.Minute(), 0, 0, time.Local)
}

func (a *application) setDraftClock(hour, minute int) {
	year, month, day := a.draftTime.Date()
	a.draftTime = time.Date(year, month, day, hour, minute, 0, 0, time.Local)
}
