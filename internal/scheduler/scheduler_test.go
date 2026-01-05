package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/bdkmv/vanish/internal/network"
	"github.com/bdkmv/vanish/internal/sync"
	"github.com/bdkmv/vanish/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduler(t *testing.T) {
	// Create dependencies
	store, err := vault.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	vaultSvc := vault.NewVaultService(store)
	tsProvider := network.NewTsnetProvider()
	syncSvc := sync.NewRcloneService(tsProvider.Dial)

	// Create scheduler
	sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")

	if sched == nil {
		t.Fatal("NewScheduler returned nil")
	}

	if sched.configPath != "./config.yaml" {
		t.Errorf("Expected configPath './config.yaml', got %s", sched.configPath)
	}

	if sched.cron == nil {
		t.Error("Expected cron to be initialized")
	}
}

func TestScheduler_StartWithEmptyCron(t *testing.T) {
	store, err := vault.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	vaultSvc := vault.NewVaultService(store)
	tsProvider := network.NewTsnetProvider()
	syncSvc := sync.NewRcloneService(tsProvider.Dial)

	sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")

	// Test with empty cron expression
	err = sched.Start(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty cron expression, got nil")
	}
}

func TestScheduler_StartWithInvalidCron(t *testing.T) {
	store, err := vault.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	vaultSvc := vault.NewVaultService(store)
	tsProvider := network.NewTsnetProvider()
	syncSvc := sync.NewRcloneService(tsProvider.Dial)

	sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")

	// Test with invalid cron expression
	err = sched.Start(context.Background(), "invalid cron")
	if err == nil {
		t.Error("Expected error for invalid cron expression, got nil")
	}
	defer sched.Stop()
}

func TestScheduler_NextRun(t *testing.T) {
	store, err := vault.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	vaultSvc := vault.NewVaultService(store)
	tsProvider := network.NewTsnetProvider()
	syncSvc := sync.NewRcloneService(tsProvider.Dial)

	sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")

	// Before starting, NextRun should return zero time
	nextRun := sched.NextRun()
	if !nextRun.IsZero() {
		t.Errorf("Expected zero time before start, got %v", nextRun)
	}

	// Start with a valid cron expression
	err = sched.Start(context.Background(), "0 0 * * *")
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer sched.Stop()

	// After starting, NextRun should return a future time
	nextRun = sched.NextRun()
	if nextRun.IsZero() {
		t.Error("Expected non-zero time after start")
	}

	if !nextRun.After(time.Now()) {
		t.Error("Expected NextRun to be in the future")
	}
}

func TestScheduler_Stop(t *testing.T) {
	store, err := vault.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	vaultSvc := vault.NewVaultService(store)
	tsProvider := network.NewTsnetProvider()
	syncSvc := sync.NewRcloneService(tsProvider.Dial)

	sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")

	// Start scheduler
	err = sched.Start(context.Background(), "0 0 * * *")
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	// Verify it's running
	nextRun := sched.NextRun()
	if nextRun.IsZero() {
		t.Error("Expected non-zero NextRun before stop")
	}

	// Stop should not panic
	sched.Stop()

	// After stop, cron context should be done (entries may still exist in memory)
	// Just verify Stop doesn't panic - that's the main point of this test
}

func TestScheduler_NextRun_BeforeStart(t *testing.T) {
	store, err := vault.NewFileStore(t.TempDir())
	require.NoError(t, err)
	vaultSvc := vault.NewVaultService(store)
	tsProvider := network.NewTsnetProvider()
	syncSvc := sync.NewRcloneService(tsProvider.Dial)

	sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")

	// NextRun before start should return zero time
	nextRun := sched.NextRun()
	assert.True(t, nextRun.IsZero(), "NextRun should be zero before Start")
}

func TestScheduler_Stop_WithoutStart(t *testing.T) {
	store, err := vault.NewFileStore(t.TempDir())
	require.NoError(t, err)
	vaultSvc := vault.NewVaultService(store)
	tsProvider := network.NewTsnetProvider()
	syncSvc := sync.NewRcloneService(tsProvider.Dial)

	sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")

	// Calling Stop without Start should not panic
	sched.Stop()
}

func TestScheduler_MultipleStarts(t *testing.T) {
	store, err := vault.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	vaultSvc := vault.NewVaultService(store)
	tsProvider := network.NewTsnetProvider()
	syncSvc := sync.NewRcloneService(tsProvider.Dial)

	sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")

	// Start scheduler
	err = sched.Start(context.Background(), "0 0 * * *")
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer sched.Stop()

	// Starting again should add another job
	err = sched.Start(context.Background(), "0 1 * * *")
	if err != nil {
		t.Fatalf("Failed to start scheduler second time: %v", err)
	}

	// Should have entries
	entries := sched.cron.Entries()
	if len(entries) == 0 {
		t.Error("Expected cron entries after start")
	}
}

func TestScheduler_ValidCronExpressions(t *testing.T) {
	store, err := vault.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	vaultSvc := vault.NewVaultService(store)
	tsProvider := network.NewTsnetProvider()
	syncSvc := sync.NewRcloneService(tsProvider.Dial)

	testCases := []struct {
		name string
		cron string
	}{
		{"every minute", "* * * * *"},
		{"hourly", "0 * * * *"},
		{"daily at 2am", "0 2 * * *"},
		{"every 6 hours", "0 */6 * * *"},
		{"weekdays at 3am", "0 3 * * 1-5"},
		{"sunday midnight", "0 0 * * 0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")
			err := sched.Start(context.Background(), tc.cron)
			if err != nil {
				t.Errorf("Failed to start with cron %q: %v", tc.cron, err)
			}
			defer sched.Stop()

			nextRun := sched.NextRun()
			if nextRun.IsZero() {
				t.Errorf("Expected non-zero NextRun for cron %q", tc.cron)
			}
		})
	}
}
