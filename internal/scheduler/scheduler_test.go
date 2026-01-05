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

func TestScheduler_Lifecycle(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "new scheduler",
			run: func(t *testing.T) {
				store, err := vault.NewFileStore(t.TempDir())
				require.NoError(t, err)
				vaultSvc := vault.NewVaultService(store)
				tsProvider := network.NewTsnetProvider()
				syncSvc := sync.NewRcloneService(tsProvider.Dial)

				sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")

				assert.NotNil(t, sched)
				assert.Equal(t, "./config.yaml", sched.configPath)
				assert.NotNil(t, sched.cron)
			},
		},
		{
			name: "next run after start",
			run: func(t *testing.T) {
				store, err := vault.NewFileStore(t.TempDir())
				require.NoError(t, err)
				vaultSvc := vault.NewVaultService(store)
				tsProvider := network.NewTsnetProvider()
				syncSvc := sync.NewRcloneService(tsProvider.Dial)

				sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")

				nextRun := sched.NextRun()
				assert.True(t, nextRun.IsZero())

				err = sched.Start(context.Background(), "0 0 * * *")
				require.NoError(t, err)
				defer sched.Stop()

				nextRun = sched.NextRun()
				assert.False(t, nextRun.IsZero())
				assert.True(t, nextRun.After(time.Now()))
			},
		},
		{
			name: "stop after start",
			run: func(t *testing.T) {
				store, err := vault.NewFileStore(t.TempDir())
				require.NoError(t, err)
				vaultSvc := vault.NewVaultService(store)
				tsProvider := network.NewTsnetProvider()
				syncSvc := sync.NewRcloneService(tsProvider.Dial)

				sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")

				err = sched.Start(context.Background(), "0 0 * * *")
				require.NoError(t, err)

				nextRun := sched.NextRun()
				assert.False(t, nextRun.IsZero())

				sched.Stop()
			},
		},
		{
			name: "next run before start",
			run: func(t *testing.T) {
				store, err := vault.NewFileStore(t.TempDir())
				require.NoError(t, err)
				vaultSvc := vault.NewVaultService(store)
				tsProvider := network.NewTsnetProvider()
				syncSvc := sync.NewRcloneService(tsProvider.Dial)

				sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")

				nextRun := sched.NextRun()
				assert.True(t, nextRun.IsZero(), "NextRun should be zero before Start")
			},
		},
		{
			name: "stop without start",
			run: func(t *testing.T) {
				store, err := vault.NewFileStore(t.TempDir())
				require.NoError(t, err)
				vaultSvc := vault.NewVaultService(store)
				tsProvider := network.NewTsnetProvider()
				syncSvc := sync.NewRcloneService(tsProvider.Dial)

				sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")

				sched.Stop()
			},
		},
		{
			name: "multiple starts",
			run: func(t *testing.T) {
				store, err := vault.NewFileStore(t.TempDir())
				require.NoError(t, err)
				vaultSvc := vault.NewVaultService(store)
				tsProvider := network.NewTsnetProvider()
				syncSvc := sync.NewRcloneService(tsProvider.Dial)

				sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")

				err = sched.Start(context.Background(), "0 0 * * *")
				require.NoError(t, err)
				defer sched.Stop()

				err = sched.Start(context.Background(), "0 1 * * *")
				require.NoError(t, err)

				entries := sched.cron.Entries()
				assert.NotEmpty(t, entries)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestScheduler_Start(t *testing.T) {
	tests := []struct {
		name     string
		cronExpr string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "empty cron expression",
			cronExpr: "",
			wantErr:  true,
			errMsg:   "cron schedule is empty",
		},
		{
			name:     "invalid cron expression",
			cronExpr: "invalid cron",
			wantErr:  true,
			errMsg:   "failed to add cron job",
		},
		{
			name:     "valid daily cron",
			cronExpr: "0 0 * * *",
			wantErr:  false,
		},
		{
			name:     "valid hourly cron",
			cronExpr: "0 * * * *",
			wantErr:  false,
		},
		{
			name:     "valid every minute cron",
			cronExpr: "* * * * *",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := vault.NewFileStore(t.TempDir())
			require.NoError(t, err)
			vaultSvc := vault.NewVaultService(store)
			tsProvider := network.NewTsnetProvider()
			syncSvc := sync.NewRcloneService(tsProvider.Dial)

			sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")
			defer sched.Stop()

			err = sched.Start(context.Background(), tt.cronExpr)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				// Verify NextRun is set for valid cron
				nextRun := sched.NextRun()
				assert.False(t, nextRun.IsZero())
				assert.True(t, nextRun.After(time.Now()))
			}
		})
	}
}

func TestScheduler_ValidCronExpressions(t *testing.T) {
	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := vault.NewFileStore(t.TempDir())
			require.NoError(t, err)
			vaultSvc := vault.NewVaultService(store)
			tsProvider := network.NewTsnetProvider()
			syncSvc := sync.NewRcloneService(tsProvider.Dial)

			sched := NewScheduler(vaultSvc, tsProvider, syncSvc, "./config.yaml")
			defer sched.Stop()

			err = sched.Start(context.Background(), tt.cron)
			require.NoError(t, err)

			nextRun := sched.NextRun()
			assert.False(t, nextRun.IsZero(), "Expected non-zero NextRun for cron %q", tt.cron)
		})
	}
}
