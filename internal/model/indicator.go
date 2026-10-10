package model

import "time"

const Day = 24 * time.Hour

type IndicatorMode uint8

const (
	IndicatorEmpty IndicatorMode = iota
	IndicatorGrid
	IndicatorCircle
	IndicatorExpired
)

type Urgency uint8

const (
	UrgencyGreen Urgency = iota
	UrgencyYellow
	UrgencyRed
	UrgencyOrange
	UrgencyOrangeRed
)

type Indicator struct {
	Mode     IndicatorMode
	Days     int
	Fraction float64
	Urgency  Urgency
}

func CountdownIndicator(target, now time.Time) Indicator {
	d := target.Sub(now)
	switch {
	case d <= 0:
		return Indicator{Mode: IndicatorExpired}
	case d > 31*Day:
		return Indicator{Mode: IndicatorEmpty}
	case d > Day:
		days := int((d-1)/Day + 1)
		urgency := UrgencyRed
		if days >= 21 {
			urgency = UrgencyGreen
		} else if days >= 11 {
			urgency = UrgencyYellow
		}
		return Indicator{Mode: IndicatorGrid, Days: days, Urgency: urgency}
	default:
		urgency := UrgencyRed
		if d >= 12*time.Hour {
			urgency = UrgencyOrange
		} else if d > 3*time.Hour {
			urgency = UrgencyOrangeRed
		}
		return Indicator{Mode: IndicatorCircle, Fraction: float64(d) / float64(Day), Urgency: urgency}
	}
}

// NextRefresh keeps minute-level progress fresh and catches earlier text,
// mode, and urgency transitions. Expired-only lists need no clock refresh.
func (c Config) NextRefresh(now time.Time) time.Duration {
	var next time.Duration
	for _, item := range c.Countdowns {
		d := item.Target.Sub(now)
		if d <= 0 {
			continue
		}
		unit := time.Minute
		if d > Day {
			unit = Day
		}
		delay := min(time.Minute, (d-1)%unit+1)
		for _, boundary := range []time.Duration{31 * Day, 20 * Day, 10 * Day, Day, 12 * time.Hour, 3 * time.Hour} {
			if d > boundary {
				delay = min(delay, d-boundary)
			}
		}
		// At these exact instants the next representable duration changes
		// the time unit or color; don't retain that frame for a full minute.
		if d == Day || d == 12*time.Hour {
			delay = min(delay, time.Millisecond)
		}
		if next == 0 || delay < next {
			next = delay
		}
	}
	return next
}
