package scheduler

import (
	"context"
	"time"

	"atlas-marriages/marriage"
	"atlas-marriages/retry"

	"github.com/Chronicle20/atlas-tenant"
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

	go s.run()
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

// processExpiredProposals processes expired proposals for all tenants
func (s *ProposalExpiryScheduler) processExpiredProposals() {
	s.log.Debug("Processing expired proposals for all tenants")

	// Get all tenants that have proposals
	tenantIds, err := s.getTenantsWithProposals()
	if err != nil {
		s.log.WithError(err).Error("Failed to get tenants with proposals")
		return
	}

	if len(tenantIds) == 0 {
		s.log.Debug("No tenants with proposals found")
		return
	}

	s.log.WithField("tenantCount", len(tenantIds)).Debug("Processing expired proposals for tenants")

	// Process each tenant
	for _, tenantId := range tenantIds {
		s.processExpiredProposalsForTenant(tenantId)
	}
}

// getTenantsWithProposals retrieves all tenant IDs that have pending proposals
func (s *ProposalExpiryScheduler) getTenantsWithProposals() ([]uuid.UUID, error) {
	var tenantIds []uuid.UUID

	retryConfig := retry.DefaultRetryConfig().
		WithLogger(s.log.WithField("operation", "get-tenants-with-proposals")).
		WithContext(s.ctx).
		WithMaxRetries(2).
		WithInitialDelay(500 * time.Millisecond)

	err := retry.ExecuteWithRetry(retryConfig, func() error {
		return s.db.Model(&marriage.ProposalEntity{}).
			Where("status = ?", marriage.ProposalStatusPending).
			Distinct("tenant_id").
			Pluck("tenant_id", &tenantIds).Error
	})

	return tenantIds, err
}

// processExpiredProposalsForTenant processes expired proposals for a specific tenant
func (s *ProposalExpiryScheduler) processExpiredProposalsForTenant(tenantId uuid.UUID) {
	retryConfig := retry.DefaultRetryConfig().
		WithLogger(s.log.WithFields(logrus.Fields{
			"operation": "process-expired-proposals",
			"tenantId":  tenantId,
		})).
		WithContext(s.ctx).
		WithMaxRetries(3).
		WithInitialDelay(1 * time.Second).
		WithMaxDelay(10 * time.Second)

	err := retry.ExecuteWithRetry(retryConfig, func() error {
		// Create a tenant model
		tenantModel, err := tenant.Create(tenantId, "background-scheduler", 1, 0)
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"tenantId": tenantId,
				"error":    err,
			}).Error("Failed to create tenant model")
			return err
		}

		// Create a context with tenant information
		tenantCtx := tenant.WithContext(s.ctx, tenantModel)

		// Create a processor with tenant context
		processor := marriage.NewProcessor(s.log, tenantCtx, s.db)

		// Process expired proposals for this tenant
		return processor.ProcessExpiredProposals()
	})

	if err != nil {
		s.log.WithFields(logrus.Fields{
			"tenantId": tenantId,
			"error":    err,
		}).Error("Failed to process expired proposals for tenant after retries")
		return
	}

	s.log.WithField("tenantId", tenantId).Debug("Successfully processed expired proposals for tenant")
}
