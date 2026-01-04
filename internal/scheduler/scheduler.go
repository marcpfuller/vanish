// Package scheduler provides scheduled execution of sync operations using cron expressions.
package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bdkmv/vanish/internal/cmd"
	"github.com/bdkmv/vanish/internal/network"
	"github.com/bdkmv/vanish/internal/sync"
	"github.com/bdkmv/vanish/internal/vault"
	"github.com/robfig/cron/v3"
)

// Scheduler manages scheduled sync operations
type Scheduler struct {
	cron        *cron.Cron
	vaultSvc    *vault.VaultService
	tsProvider  network.TailscaleProvider
	syncService sync.SyncService
	configPath  string
}

// NewScheduler creates a new scheduler
func NewScheduler(
	vaultSvc *vault.VaultService,
	tsProvider network.TailscaleProvider,
	syncService sync.SyncService,
	configPath string,
) *Scheduler {
	return &Scheduler{
		cron:        cron.New(),
		vaultSvc:    vaultSvc,
		tsProvider:  tsProvider,
		syncService: syncService,
		configPath:  configPath,
	}
}

// Start starts the scheduler with the given cron expression
func (s *Scheduler) Start(ctx context.Context, cronExpr string) error {
	if cronExpr == "" {
		return fmt.Errorf("cron schedule is empty")
	}

	log.Printf("📅 Scheduling sync jobs with cron: %s", cronExpr)

	// Add the sync job to the cron scheduler
	_, err := s.cron.AddFunc(cronExpr, func() {
		log.Println("⏰ Scheduled sync triggered")
		if err := s.runSync(); err != nil {
			log.Printf("❌ Scheduled sync failed: %v", err)
		} else {
			log.Println("✅ Scheduled sync completed successfully")
		}
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	// Start the cron scheduler
	s.cron.Start()
	log.Println("✅ Scheduler started successfully")

	// Log next scheduled run
	entries := s.cron.Entries()
	if len(entries) > 0 {
		log.Printf("⏭️  Next sync scheduled for: %s", entries[0].Next.Format(time.RFC3339))
	}

	return nil
}

// runSync executes the sync operation
func (s *Scheduler) runSync() error {
	opts := cmd.SyncOptions{
		ConfigPath: s.configPath,
		Timeout:    10 * time.Minute,
	}
	return cmd.Sync(s.vaultSvc, s.tsProvider, s.syncService, opts)
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	log.Println("🛑 Stopping scheduler...")
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("✅ Scheduler stopped")
}

// NextRun returns the time of the next scheduled run
func (s *Scheduler) NextRun() time.Time {
	entries := s.cron.Entries()
	if len(entries) > 0 {
		return entries[0].Next
	}
	return time.Time{}
}
