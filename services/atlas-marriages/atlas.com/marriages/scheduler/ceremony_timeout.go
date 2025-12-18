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
	
	go s.run()
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

// processActiveCeremonies processes active ceremonies for all tenants
func (s *CeremonyTimeoutScheduler) processActiveCeremonies() {
	s.log.Debug("Processing active ceremonies for timeout monitoring")
	
	// Get all tenants that have active ceremonies
	tenantIds, err := s.getTenantsWithActiveCeremonies()
	if err != nil {
		s.log.WithError(err).Error("Failed to get tenants with active ceremonies")
		return
	}
	
	if len(tenantIds) == 0 {
		s.log.Debug("No tenants with active ceremonies found")
		return
	}
	
	s.log.WithField("tenantCount", len(tenantIds)).Debug("Processing active ceremonies for tenants")
	
	// Process each tenant
	for _, tenantId := range tenantIds {
		s.processActiveCeremoniesForTenant(tenantId)
	}
}

// getTenantsWithActiveCeremonies retrieves all tenant IDs that have active ceremonies
func (s *CeremonyTimeoutScheduler) getTenantsWithActiveCeremonies() ([]uuid.UUID, error) {
	var tenantIds []uuid.UUID
	
	retryConfig := retry.DefaultRetryConfig().
		WithLogger(s.log.WithField("operation", "get-tenants-with-active-ceremonies")).
		WithContext(s.ctx).
		WithMaxRetries(2).
		WithInitialDelay(500 * time.Millisecond)
	
	err := retry.ExecuteWithRetry(retryConfig, func() error {
		return s.db.Model(&marriage.CeremonyEntity{}).
			Where("status = ?", marriage.CeremonyStatusActive).
			Distinct("tenant_id").
			Pluck("tenant_id", &tenantIds).Error
	})
	
	return tenantIds, err
}

// processActiveCeremoniesForTenant processes active ceremonies for a specific tenant
func (s *CeremonyTimeoutScheduler) processActiveCeremoniesForTenant(tenantId uuid.UUID) {
	retryConfig := retry.DefaultRetryConfig().
		WithLogger(s.log.WithFields(logrus.Fields{
			"operation": "process-ceremony-timeouts",
			"tenantId":  tenantId,
		})).
		WithContext(s.ctx).
		WithMaxRetries(3).
		WithInitialDelay(1 * time.Second).
		WithMaxDelay(10 * time.Second)
	
	err := retry.ExecuteWithRetry(retryConfig, func() error {
		// Create a tenant model
		tenantModel, err := tenant.Create(tenantId, "ceremony-timeout-scheduler", 1, 0)
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
		
		// Process ceremony timeouts for this tenant
		return processor.ProcessCeremonyTimeouts()
	})
	
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"tenantId": tenantId,
			"error":    err,
		}).Error("Failed to process ceremony timeouts for tenant after retries")
		return
	}
	
	s.log.WithField("tenantId", tenantId).Debug("Successfully processed ceremony timeouts for tenant")
}