// Package config defines configuration structures for vanish sync jobs.
package config

// Config represents the vanish configuration file structure
type Config struct {
	Version  string         `yaml:"version"`
	Schedule ScheduleConfig `yaml:"schedule"`
	SyncJobs []SyncJob      `yaml:"sync_jobs"`
}

// ScheduleConfig represents the scheduling configuration for service mode
type ScheduleConfig struct {
	Enabled bool   `yaml:"enabled"`
	Cron    string `yaml:"cron"` // Cron expression: "minute hour day month day-of-week"
}

// SyncJob defines a single sync operation
type SyncJob struct {
	Name        string `yaml:"name"`
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
	Enabled     bool   `yaml:"enabled"`
	Mode        string `yaml:"mode"` // "copy" (default) or "sync" (destructive)
}
