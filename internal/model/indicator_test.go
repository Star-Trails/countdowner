package model

import (
	"testing"
	"time"
)

func TestCountdownIndicatorBoundaries(t *testing.T) {
	now := time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name     string
		duration time.Duration
		mode     IndicatorMode
		days     int
		fraction float64
		urgency  Urgency
	}{
		{"25 days", 25 * Day, IndicatorGrid, 25, 0, UrgencyGreen},
		{"15 days", 15 * Day, IndicatorGrid, 15, 0, UrgencyYellow},
		{"5 days", 5 * Day, IndicatorGrid, 5, 0, UrgencyRed},
		{"31 days", 31 * Day, IndicatorGrid, 31, 0, UrgencyGreen},
		{"beyond window", 31*Day + time.Nanosecond, IndicatorEmpty, 0, 0, 0},
		{"45 days", 45 * Day, IndicatorEmpty, 0, 0, 0},
		{"green boundary", 20*Day + time.Nanosecond, IndicatorGrid, 21, 0, UrgencyGreen},
		{"yellow boundary", 20 * Day, IndicatorGrid, 20, 0, UrgencyYellow},
		{"yellow lower boundary", 10*Day + time.Nanosecond, IndicatorGrid, 11, 0, UrgencyYellow},
		{"red boundary", 10 * Day, IndicatorGrid, 10, 0, UrgencyRed},
		{"above 24 hours", Day + time.Nanosecond, IndicatorGrid, 2, 0, UrgencyRed},
		{"exactly 24 hours", Day, IndicatorCircle, 0, 1, UrgencyOrange},
		{"12 hours", 12 * time.Hour, IndicatorCircle, 0, 0.5, UrgencyOrange},
		{"6 hours", 6 * time.Hour, IndicatorCircle, 0, 0.25, UrgencyOrangeRed},
		{"3 hours", 3 * time.Hour, IndicatorCircle, 0, 0.125, UrgencyRed},
		{"one minute", time.Minute, IndicatorCircle, 0, 1.0 / 1440, UrgencyRed},
		{"expired", 0, IndicatorExpired, 0, 0, 0},
		{"past", -time.Hour, IndicatorExpired, 0, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := CountdownIndicator(now.Add(tc.duration), now)
			want := Indicator{Mode: tc.mode, Days: tc.days, Fraction: tc.fraction, Urgency: tc.urgency}
			if got != want {
				t.Fatalf("indicator = %+v; want %+v", got, want)
			}
		})
	}
	// Ceil the exact duration; partial days do not use calendar boundaries.
	for days := 2; days <= 31; days++ {
		got := CountdownIndicator(now.Add(time.Duration(days-1)*Day+time.Nanosecond), now)
		if got.Mode != IndicatorGrid || got.Days != days {
			t.Fatalf("partial-day grid count: %+v; want %d", got, days)
		}
	}
}

func TestCountdownIndicatorCalendarAndDST(t *testing.T) {
	for _, date := range []time.Time{
		time.Date(2025, 2, 28, 12, 0, 0, 0, time.UTC),
		time.Date(2024, 2, 28, 12, 0, 0, 0, time.UTC),
		time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC),
	} {
		got := CountdownIndicator(date.AddDate(0, 0, 2), date)
		if got.Mode != IndicatorGrid || got.Days != 2 {
			t.Fatalf("calendar boundary %v: %+v", date, got)
		}
	}
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	spring := time.Date(2026, 3, 7, 12, 0, 0, 0, loc)
	got := CountdownIndicator(spring.AddDate(0, 0, 1), spring)
	if got.Mode != IndicatorCircle || got.Fraction != 23.0/24 {
		t.Fatalf("23-hour DST day must be circular: %+v", got)
	}
	autumn := time.Date(2026, 10, 31, 12, 0, 0, 0, loc)
	got = CountdownIndicator(autumn.AddDate(0, 0, 1), autumn)
	if got.Mode != IndicatorGrid || got.Days != 2 {
		t.Fatalf("25-hour DST day must ceil to two cells: %+v", got)
	}
}

func TestNextCountdownRefresh(t *testing.T) {
	now := time.Now()
	config := Default()
	for _, tc := range []struct {
		duration, want time.Duration
	}{
		{45 * Day, time.Minute},
		{31*Day + 5*time.Second, 5 * time.Second},
		{Day + 10*time.Second, 10 * time.Second},
		{Day, time.Millisecond},
		{12*time.Hour + 10*time.Second, 10 * time.Second},
		{3*time.Hour + 10*time.Second, 10 * time.Second},
		{2*time.Hour + 30*time.Second, 30 * time.Second},
		{15 * time.Second, 15 * time.Second},
		{0, 0},
	} {
		config.Countdowns = []Countdown{{Target: now.Add(tc.duration)}}
		if got := config.NextRefresh(now); got != tc.want {
			t.Errorf("refresh for %v = %v; want %v", tc.duration, got, tc.want)
		}
	}
}
