package schedule

import (
	"context"
	"strings"
	"time"

	"github.com/ramanasai/pulse/internal/config"
)

// NextAt computes the next occurrence of reminder time that is on a configured workday and not a holiday.
func NextAt(now time.Time, cfg config.Config) time.Time {
	loc := cfg.Location()
	now = now.In(loc)

	// parse "HH:MM"
	hour, min := 17, 0
	if len(cfg.Reminder.Time) >= 4 {
		if t, err := time.ParseInLocation("15:04", cfg.Reminder.Time, loc); err == nil {
			hour = t.Hour()
			min = t.Minute()
		}
	}
	workdays := map[string]bool{}
	for _, d := range cfg.Reminder.Workdays {
		abbr := strings.Title(strings.ToLower(strings.TrimSpace(d[:3])))
		workdays[abbr] = true
	}
	isWorkday := func(t time.Time) bool {
		abbr := t.Weekday().String()[:3]
		return workdays[abbr]
	}
	holidays := map[string]bool{}
	for _, h := range cfg.Reminder.Holidays {
		holidays[strings.TrimSpace(h)] = true
	}
	isHoliday := func(t time.Time) bool {
		key := t.Format("2006-01-02")
		return holidays[key]
	}

	// candidate today at hh:mm
	cand := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, loc)
	if !now.Before(cand) {
		cand = cand.Add(24 * time.Hour)
	}
	for {
		if isWorkday(cand) && !isHoliday(cand) {
			return cand
		}
		cand = cand.Add(24 * time.Hour)
	}
}

// RunConfigured runs the reminder callback at the configured schedule until ctx is canceled.
func RunConfigured(ctx context.Context, cfg config.Config, f func()) {
	next := NextAt(time.Now(), cfg)
	t := time.NewTimer(time.Until(next))
	for {
		select {
		case <-ctx.Done():
			if !t.Stop() {
				select {
				case <-t.C:
				default:
				}
			}
			return
		case <-t.C:
			f()
			next = NextAt(time.Now(), cfg)
			t.Reset(time.Until(next))
		}
	}
}
