package scheduler

import (
	"atlas-marriages/marriage"
	"context"
	"time"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	service "github.com/Chronicle20/atlas/libs/atlas-service"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	retry "github.com/Chronicle20/atlas/libs/atlas-retry"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// serviceName is the environment-registry owner name for atlas-marriages,
// matching the const declared in package main (main.go:19). Duplicated here
// because the two packages cannot share an unexported const.
const serviceName = "atlas-marriages"

// CeremonyTimeoutScheduler handles periodic checking and timeout of active ceremonies
type CeremonyTimeoutScheduler struct {
	log      logrus.FieldLogger
	ctx      context.Context
	db       *gorm.DB
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
}

// NewCeremonyTimeoutScheduler creates a new ceremony timeout scheduler
func NewCeremonyTimeoutScheduler(log logrus.FieldLogger, ctx context.Context, db *gorm.DB) *CeremonyTimeoutScheduler {
	return &CeremonyTimeoutScheduler{
		log:      log.WithField("component", "ceremony-timeout-scheduler"),
		ctx:      ctx,
		db:       db,
		interval: 1 * time.Minute, // Check every minute for responsiveness
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// WithInterval sets the check interval
func (s *CeremonyTimeoutScheduler) WithInterval(interval time.Duration) *CeremonyTimeoutScheduler {
	s.interval = interval
	return s
}

// Start begins the background ceremony timeout checking
func (s *CeremonyTimeoutScheduler) Start() {
	s.log.WithField("interval", s.interval).Info("Starting ceremony timeout scheduler")

	routine.Go(s.log, s.ctx, func(_ context.Context) {
		s.run()
	})
}

// Stop gracefully stops the scheduler
func (s *CeremonyTimeoutScheduler) Stop() {
	s.log.Info("Stopping ceremony timeout scheduler")
	close(s.stop)
	<-s.done
	s.log.Info("Ceremony timeout scheduler stopped")
}

// run is the main loop for the scheduler
func (s *CeremonyTimeoutScheduler) run() {
	defer close(s.done)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Process immediately on start
	s.processActiveCeremonies()

	for {
		select {
		case <-ticker.C:
			s.processActiveCeremonies()
		case <-s.stop:
			return
		case <-s.ctx.Done():
			s.log.Info("Context cancelled, stopping ceremony timeout scheduler")
			return
		}
	}
}

// processActiveCeremonies processes active ceremonies for every tenant of
// every environment this deployment owns for atlas-marriages. Both the
// environment set and each environment's tenant set are resolved fresh on
// every call (FR-6.4) via service.ForEachOwnedEnvironment.
func (s *CeremonyTimeoutScheduler) processActiveCeremonies() {
	s.log.Debug("Processing active ceremonies for timeout monitoring")

	service.ForEachOwnedEnvironment(s.log, s.ctx, serviceName, s.listTenantsWithActiveCeremonies,
		func(ctx context.Context) {
			s.processActiveCeremoniesForTenant(ctx)
		})
}

// listTenantsWithActiveCeremonies is the service.TenantLister for this
// scheduler: it retrieves every tenant ID that has an active ceremony and
// builds the corresponding tenant models.
func (s *CeremonyTimeoutScheduler) listTenantsWithActiveCeremonies(ctx context.Context) ([]tenant.Model, error) {
	tenantIds, err := s.getTenantsWithActiveCeremonies(ctx)
	if err != nil {
		return nil, err
	}
	if len(tenantIds) == 0 {
		s.log.Debug("No tenants with active ceremonies found")
		return nil, nil
	}

	s.log.WithField("tenantCount", len(tenantIds)).Debug("Processing active ceremonies for tenants")

	ts := make([]tenant.Model, 0, len(tenantIds))
	for _, tenantId := range tenantIds {
		tm, err := tenant.Create(tenantId, "ceremony-timeout-scheduler", 1, 0)
		if err != nil {
			s.log.WithError(err).WithField("tenantId", tenantId).Error("Unable to build tenant model; skipping.")
			continue
		}
		ts = append(ts, tm)
	}
	return ts, nil
}

// getTenantsWithActiveCeremonies retrieves all tenant IDs that have active ceremonies
func (s *CeremonyTimeoutScheduler) getTenantsWithActiveCeremonies(ctx context.Context) ([]uuid.UUID, error) {
	var tenantIds []uuid.UUID

	cfg := retry.DefaultConfig().WithMaxRetries(3).WithInitialDelay(500 * time.Millisecond).WithMaxDelay(5 * time.Second)
	err := retry.Try(ctx, cfg, func(attempt int) (bool, error) {
		err := s.db.Model(&marriage.CeremonyEntity{}).
			Where("status = ?", marriage.CeremonyStatusActive).
			Distinct("tenant_id").
			Pluck("tenant_id", &tenantIds).Error
		return err != nil, err
	})
	if err != nil {
		s.log.WithError(err).Error("Failed to get tenants with active ceremonies")
	}

	return tenantIds, err
}

// processActiveCeremoniesForTenant processes active ceremonies for the
// tenant already carried on ctx (set by service.ForEachOwnedEnvironment).
func (s *CeremonyTimeoutScheduler) processActiveCeremoniesForTenant(ctx context.Context) {
	t := tenant.MustFromContext(ctx)
	cfg2 := retry.DefaultConfig().WithMaxRetries(3).WithInitialDelay(1 * time.Second).WithMaxDelay(10 * time.Second)
	err := retry.Try(ctx, cfg2, func(attempt int) (bool, error) {
		processor := marriage.NewProcessor(s.log, ctx, s.db)
		err := processor.ProcessCeremonyTimeouts()
		return err != nil, err
	})
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"tenantId": t.Id(),
			"error":    err,
		}).Error("Failed to process ceremony timeouts for tenant after retries")
		return
	}

	s.log.WithField("tenantId", t.Id()).Debug("Successfully processed ceremony timeouts for tenant")
}
