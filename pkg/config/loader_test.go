package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bdkmv/vanish/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ValidConfig(t *testing.T) {
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

	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "1.0", cfg.Version)
	assert.Len(t, cfg.SyncJobs, 1)
	assert.Equal(t, "test-job", cfg.SyncJobs[0].Name)
	assert.Equal(t, "/source", cfg.SyncJobs[0].Source)
	assert.Equal(t, "/dest", cfg.SyncJobs[0].Destination)
	assert.True(t, cfg.SyncJobs[0].Enabled)
}

func TestLoad_NonExistentFile(t *testing.T) {
	_, err := config.Load("/nonexistent/config.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidContent := `this is not: valid: yaml: content`
	err := os.WriteFile(configPath, []byte(invalidContent), 0o644)
	require.NoError(t, err)

	_, err = config.Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestLoad_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty.yaml")

	err := os.WriteFile(configPath, []byte(""), 0o644)
	require.NoError(t, err)

	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Empty(t, cfg.Version)
	assert.Len(t, cfg.SyncJobs, 0)
}

func TestLoad_MultipleJobs(t *testing.T) {
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

	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Len(t, cfg.SyncJobs, 3)
	assert.Equal(t, "job1", cfg.SyncJobs[0].Name)
	assert.True(t, cfg.SyncJobs[0].Enabled)
	assert.Equal(t, "job2", cfg.SyncJobs[1].Name)
	assert.False(t, cfg.SyncJobs[1].Enabled)
	assert.Equal(t, "job3", cfg.SyncJobs[2].Name)
	assert.True(t, cfg.SyncJobs[2].Enabled)
}

func TestSave_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "save.yaml")

	cfg := &config.Config{
		Version: "1.0",
		SyncJobs: []config.SyncJob{
			{
				Name:        "test-job",
				Source:      "/source",
				Destination: "/dest",
				Enabled:     true,
			},
		},
	}

	err := config.Save(configPath, cfg)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Load it back and verify
	loaded, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, cfg.Version, loaded.Version)
	assert.Len(t, loaded.SyncJobs, 1)
	assert.Equal(t, cfg.SyncJobs[0].Name, loaded.SyncJobs[0].Name)
}

func TestSave_InvalidPath(t *testing.T) {
	cfg := &config.Config{
		Version: "1.0",
	}

	err := config.Save("/nonexistent/impossible/path/config.yaml", cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write config file")
}
