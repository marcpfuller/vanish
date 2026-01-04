// Package config defines configuration structures for vanish sync jobs.
package config

// Config represents the vanish configuration file structure
type Config struct {
	Version  string    `yaml:"version"`
	SyncJobs []SyncJob `yaml:"sync_jobs"`
}

// SyncJob defines a single sync operation
type SyncJob struct {
	Name        string `yaml:"name"`
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
	Enabled     bool   `yaml:"enabled"`
}
