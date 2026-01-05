package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bdkmv/vanish/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, cfg *config.Config)
	}{
		{
			name: "valid config",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				configContent := `version: "1.0"
sync_jobs:
  - name: "test-job"
    source: "/source"
    destination: "/dest"
    enabled: true
`
				err := os.WriteFile(configPath, []byte(configContent), 0o644)
				require.NoError(t, err)
				return configPath
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *config.Config) {
				assert.Equal(t, "1.0", cfg.Version)
				assert.Len(t, cfg.SyncJobs, 1)
				assert.Equal(t, "test-job", cfg.SyncJobs[0].Name)
				assert.Equal(t, "/source", cfg.SyncJobs[0].Source)
				assert.Equal(t, "/dest", cfg.SyncJobs[0].Destination)
				assert.True(t, cfg.SyncJobs[0].Enabled)
			},
		},
		{
			name: "non-existent file",
			setup: func(t *testing.T) string {
				return "/nonexistent/config.yaml"
			},
			wantErr:     true,
			errContains: "failed to read config file",
		},
		{
			name: "invalid YAML",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "invalid.yaml")
				invalidContent := `this is not: valid: yaml: content`
				err := os.WriteFile(configPath, []byte(invalidContent), 0o644)
				require.NoError(t, err)
				return configPath
			},
			wantErr:     true,
			errContains: "failed to parse config file",
		},
		{
			name: "empty config",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "empty.yaml")
				err := os.WriteFile(configPath, []byte(""), 0o644)
				require.NoError(t, err)
				return configPath
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *config.Config) {
				assert.Empty(t, cfg.Version)
				assert.Len(t, cfg.SyncJobs, 0)
			},
		},
		{
			name: "multiple jobs",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "multi.yaml")
				configContent := `version: "1.0"
sync_jobs:
  - name: "job1"
    source: "/source1"
    destination: "/dest1"
    enabled: true
  - name: "job2"
    source: "/source2"
    destination: "/dest2"
    enabled: false
  - name: "job3"
    source: "/source3"
    destination: "/dest3"
    enabled: true
`
				err := os.WriteFile(configPath, []byte(configContent), 0o644)
				require.NoError(t, err)
				return configPath
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *config.Config) {
				assert.Len(t, cfg.SyncJobs, 3)
				assert.Equal(t, "job1", cfg.SyncJobs[0].Name)
				assert.True(t, cfg.SyncJobs[0].Enabled)
				assert.Equal(t, "job2", cfg.SyncJobs[1].Name)
				assert.False(t, cfg.SyncJobs[1].Enabled)
				assert.Equal(t, "job3", cfg.SyncJobs[2].Name)
				assert.True(t, cfg.SyncJobs[2].Enabled)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setup(t)
			cfg, err := config.Load(configPath)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestSave(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		setupPath   func(t *testing.T) string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: &config.Config{
				Version: "1.0",
				SyncJobs: []config.SyncJob{
					{
						Name:        "test-job",
						Source:      "/source",
						Destination: "/dest",
						Enabled:     true,
					},
				},
			},
			setupPath: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "save.yaml")
			},
			wantErr: false,
		},
		{
			name: "invalid path",
			config: &config.Config{
				Version: "1.0",
			},
			setupPath: func(t *testing.T) string {
				return "/nonexistent/impossible/path/config.yaml"
			},
			wantErr:     true,
			errContains: "failed to write config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupPath(t)
			err := config.Save(configPath, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				// Verify file exists
				_, err = os.Stat(configPath)
				require.NoError(t, err)

				// Load it back and verify
				loaded, err := config.Load(configPath)
				require.NoError(t, err)
				assert.Equal(t, tt.config.Version, loaded.Version)
				assert.Len(t, loaded.SyncJobs, len(tt.config.SyncJobs))
			}
		})
	}
}
