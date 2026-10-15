package model

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Countdown stores an absolute instant; changing the system timezone does not
// change the deadline. TimeZone records the timezone used at input.
type Countdown struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Target   time.Time `json:"target"`
	TimeZone string    `json:"timeZone"`
}

type Position struct {
	Monitor string  `json:"monitor"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
}

type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Config struct {
	Version    int         `json:"version"`
	Countdowns []Countdown `json:"countdowns"`
	Sort       string      `json:"sort"`
	Theme      string      `json:"theme"`
	Language   string      `json:"language"`
	Autostart  bool        `json:"autostart"`
	Position   Position    `json:"position"`
	WidgetSize Size        `json:"widgetSize,omitzero"`
}

func Default() Config {
	return Config{
		Version: 1, Countdowns: []Countdown{}, Sort: "deadline", Theme: "system", Language: "en",
		Position: Position{X: 40, Y: 40},
	}
}

func (c Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("不支持的配置版本 %d；原文件不会被覆盖", c.Version)
	}
	if !slices.Contains([]string{"manual", "deadline"}, c.Sort) {
		return errors.New("未知排序模式")
	}
	if !slices.Contains([]string{"system", "light", "dark"}, c.Theme) {
		return errors.New("未知主题")
	}
	if !slices.Contains([]string{"en", "zh-Hant", "zh-Hans", "ja", "ko"}, c.Language) {
		return errors.New("unknown interface language")
	}
	if c.WidgetSize.Width < 0 || c.WidgetSize.Height < 0 || (c.WidgetSize.Width == 0) != (c.WidgetSize.Height == 0) {
		return errors.New("invalid widget size")
	}
	seen := make(map[string]bool)
	for _, item := range c.Countdowns {
		if item.ID == "" || seen[item.ID] || strings.TrimSpace(item.Title) == "" || item.Target.IsZero() {
			return errors.New("倒计时配置无效")
		}
		seen[item.ID] = true
	}
	return nil
}

func NewCountdown(title, value string, loc *time.Location) (Countdown, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Countdown{}, errors.New("Enter a title")
	}
	target, err := ParseTarget(value, loc)
	if err != nil {
		return Countdown{}, err
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return Countdown{}, err
	}
	return Countdown{ID: hex.EncodeToString(id[:]), Title: title, Target: target, TimeZone: loc.String()}, nil
}

// ParseTarget accepts a date alone (midnight) or a local time with minutes.
func ParseTarget(value string, loc *time.Location) (time.Time, error) {
	value = strings.TrimSpace(value)
	layout := "2006-01-02 15:04"
	if len(value) == 10 {
		layout = "2006-01-02"
	}
	target, err := time.ParseInLocation(layout, value, loc)
	if err != nil || target.Format(layout) != value {
		return time.Time{}, errors.New("Enter a valid local date or time")
	}
	return target, nil
}

func (c Config) Ordered() []Countdown {
	items := slices.Clone(c.Countdowns)
	if c.Sort == "deadline" {
		slices.SortStableFunc(items, func(a, b Countdown) int {
			return a.Target.Compare(b.Target)
		})
	}
	return items
}

// RemainingTime holds rounded countdown units, independently of their language.
type RemainingTime struct {
	Days, Hours, Minutes int
	Expired              bool
}

func Remaining(target, now time.Time) RemainingTime {
	d := target.Sub(now)
	if d <= 0 {
		return RemainingTime{Expired: true}
	}
	if d >= 24*time.Hour {
		return RemainingTime{Days: int((d-1)/(24*time.Hour) + 1)}
	}
	minutes := (d-1)/time.Minute + 1
	return RemainingTime{Hours: int(minutes / 60), Minutes: int(minutes % 60)}
}
