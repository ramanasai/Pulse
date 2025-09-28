package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type ReminderConfig struct {
	Enabled  bool     `mapstructure:"enabled"`
	Time     string   `mapstructure:"time"`      // "17:00"
	Workdays []string `mapstructure:"workdays"`  // ["Mon","Tue","Wed","Thu","Fri"]
	Holidays []string `mapstructure:"holidays"`  // ["2025-01-26", "2025-08-15"]
	Timezone string   `mapstructure:"timezone"`  // e.g. "Asia/Kolkata" (optional)
}

type Config struct {
	Theme    string         `mapstructure:"theme"`
	Reminder ReminderConfig `mapstructure:"reminder"`
}

func Default() Config {
	return Config{
		Theme: "default",
		Reminder: ReminderConfig{
			Enabled:  true,
			Time:     "17:00",
			Workdays: []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
			Holidays: []string{},
			Timezone: "",
		},
	}
}

func xdgConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil { return "", err }
	dir := filepath.Join(home, ".config", "pulse")
	if err := os.MkdirAll(dir, 0o755); err != nil { return "", err }
	return filepath.Join(dir, "config.yaml"), nil
}

func Load() (Config, error) {
	cfg := Default()

	path, err := xdgConfigPath()
	if err != nil {
		return cfg, err
	}

	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(path)

	// defaults
	v.SetDefault("theme", cfg.Theme)
	v.SetDefault("reminder.enabled", cfg.Reminder.Enabled)
	v.SetDefault("reminder.time", cfg.Reminder.Time)
	v.SetDefault("reminder.workdays", cfg.Reminder.Workdays)
	v.SetDefault("reminder.holidays", cfg.Reminder.Holidays)
	v.SetDefault("reminder.timezone", cfg.Reminder.Timezone)

	_ = v.ReadInConfig() // ok if missing
	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("config unmarshal: %w", err)
	}

	// normalize workdays
	for i, d := range cfg.Reminder.Workdays {
		cfg.Reminder.Workdays[i] = strings.Title(strings.ToLower(strings.TrimSpace(d[:3])))
	}
	return cfg, nil
}

func (c Config) Location() *time.Location {
	if tz := strings.TrimSpace(c.Reminder.Timezone); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc
		}
	}
	return time.Local
}
