package scheduler

import (
	"atlas-marriages/marriage"
	"context"
	"time"

	retry "github.com/Chronicle20/atlas/libs/atlas-retry"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ProposalExpiryScheduler handles periodic checking and expiry of proposals
type ProposalExpiryScheduler struct {
	log      logrus.FieldLogger
	ctx      context.Context
	db       *gorm.DB
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
}

// NewProposalExpiryScheduler creates a new proposal expiry scheduler
func NewProposalExpiryScheduler(log logrus.FieldLogger, ctx context.Context, db *gorm.DB) *ProposalExpiryScheduler {
	return &ProposalExpiryScheduler{
		log:      log.WithField("component", "proposal-expiry-scheduler"),
		ctx:      ctx,
		db:       db,
		interval: 5 * time.Minute, // Check every 5 minutes
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// WithInterval sets the check interval
func (s *ProposalExpiryScheduler) WithInterval(interval time.Duration) *ProposalExpiryScheduler {
	s.interval = interval
	return s
}

// Start begins the background proposal expiry checking
func (s *ProposalExpiryScheduler) Start() {
	s.log.WithField("interval", s.interval).Info("Starting proposal expiry scheduler")

	routine.Go(s.log, s.ctx, func(_ context.Context) {
		s.run()
	})
}

// Stop gracefully stops the scheduler
func (s *ProposalExpiryScheduler) Stop() {
	s.log.Info("Stopping proposal expiry scheduler")
	close(s.stop)
	<-s.done
	s.log.Info("Proposal expiry scheduler stopped")
}

// run is the main loop for the scheduler
func (s *ProposalExpiryScheduler) run() {
	defer close(s.done)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Process immediately on start
	s.processExpiredProposals()

	for {
		select {
		case <-ticker.C:
			s.processExpiredProposals()
		case <-s.stop:
			return
		case <-s.ctx.Done():
			s.log.Info("Context cancelled, stopping proposal expiry scheduler")
			return
		}
	}
}

// processExpiredProposals processes expired proposals for every tenant of
// every environment this deployment owns for atlas-marriages. Both the
// environment set and each environment's tenant set are resolved fresh on
// every call (FR-6.4) via service.ForEachOwnedEnvironment.
func (s *ProposalExpiryScheduler) processExpiredProposals() {
	s.log.Debug("Processing expired proposals for all tenants")

	service.ForEachOwnedEnvironment(s.log, s.ctx, serviceName, s.listTenantsWithProposals,
		func(ctx context.Context) {
			s.processExpiredProposalsForTenant(ctx)
		})
}

// listTenantsWithProposals is the service.TenantLister for this scheduler:
// it retrieves every tenant ID that has a pending proposal and builds the
// corresponding tenant models.
func (s *ProposalExpiryScheduler) listTenantsWithProposals(ctx context.Context) ([]tenant.Model, error) {
	tenantIds, err := s.getTenantsWithProposals(ctx)
	if err != nil {
		return nil, err
	}
	if len(tenantIds) == 0 {
		s.log.Debug("No tenants with proposals found")
		return nil, nil
	}

	s.log.WithField("tenantCount", len(tenantIds)).Debug("Processing expired proposals for tenants")

	ts := make([]tenant.Model, 0, len(tenantIds))
	for _, tenantId := range tenantIds {
		tm, err := tenant.Create(tenantId, "background-scheduler", 1, 0)
		if err != nil {
			s.log.WithError(err).WithField("tenantId", tenantId).Error("Unable to build tenant model; skipping.")
			continue
		}
		ts = append(ts, tm)
	}
	return ts, nil
}

// getTenantsWithProposals retrieves all tenant IDs that have pending proposals
func (s *ProposalExpiryScheduler) getTenantsWithProposals(ctx context.Context) ([]uuid.UUID, error) {
	var tenantIds []uuid.UUID

	cfg := retry.DefaultConfig().WithMaxRetries(3).WithInitialDelay(500 * time.Millisecond).WithMaxDelay(5 * time.Second)
	err := retry.Try(ctx, cfg, func(attempt int) (bool, error) {
		err := s.db.Model(&marriage.ProposalEntity{}).
			Where("status = ?", marriage.ProposalStatusPending).
			Distinct("tenant_id").
			Pluck("tenant_id", &tenantIds).Error
		return err != nil, err
	})
	if err != nil {
		s.log.WithError(err).Error("Failed to get tenants with proposals")
	}

	return tenantIds, err
}

// processExpiredProposalsForTenant processes expired proposals for the
// tenant already carried on ctx (set by service.ForEachOwnedEnvironment).
func (s *ProposalExpiryScheduler) processExpiredProposalsForTenant(ctx context.Context) {
	t := tenant.MustFromContext(ctx)
	cfg2 := retry.DefaultConfig().WithMaxRetries(3).WithInitialDelay(1 * time.Second).WithMaxDelay(10 * time.Second)
	err := retry.Try(ctx, cfg2, func(attempt int) (bool, error) {
		processor := marriage.NewProcessor(s.log, ctx, s.db)
		err := processor.ProcessExpiredProposals()
		return err != nil, err
	})
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"tenantId": t.Id(),
			"error":    err,
		}).Error("Failed to process expired proposals for tenant after retries")
		return
	}

	s.log.WithField("tenantId", t.Id()).Debug("Successfully processed expired proposals for tenant")
}
