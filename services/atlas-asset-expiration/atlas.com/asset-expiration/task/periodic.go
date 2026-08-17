package task

import (
	"atlas-asset-expiration/character"
	"atlas-asset-expiration/session"
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	service "github.com/Chronicle20/atlas/libs/atlas-service"

	"github.com/sirupsen/logrus"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	defaultInterval = 60 * time.Second

	// serviceName is the environment-registry owner name for
	// atlas-asset-expiration, matching the const declared in package main
	// (main.go:18). Duplicated here because the two packages cannot share
	// an unexported const.
	serviceName = "atlas-asset-expiration"
)

// PeriodicTask runs expiration checks at regular intervals for all online sessions
type PeriodicTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
	stopCh   chan struct{}
	wg       *sync.WaitGroup
}

// NewPeriodicTask creates a new periodic expiration check task
func NewPeriodicTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *PeriodicTask {
	if interval <= 0 {
		interval = defaultInterval
	}
	return &PeriodicTask{
		l:        l,
		ctx:      ctx,
		interval: interval,
		stopCh:   make(chan struct{}),
		wg:       &sync.WaitGroup{},
	}
}

// Start starts the periodic task
func (t *PeriodicTask) Start() {
	t.wg.Add(1)
	routine.Go(t.l, t.ctx, func(_ context.Context) {
		t.run()
	})
	t.l.Infof("Periodic expiration task started with interval [%v].", t.interval)
}

// Stop stops the periodic task
func (t *PeriodicTask) Stop() {
	close(t.stopCh)
	t.wg.Wait()
	t.l.Infoln("Periodic expiration task stopped.")
}

func (t *PeriodicTask) run() {
	defer t.wg.Done()

	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.checkAllSessions()
		case <-t.stopCh:
			return
		}
	}
}

// checkAllSessions runs the expiration check for every tenant of every
// environment this deployment owns for atlas-asset-expiration. Both the
// environment set and each environment's tenant set are resolved fresh on
// every call (FR-6.4) via service.ForEachOwnedEnvironment: the session
// snapshot is taken once per tick, but which tenant a session belongs to is
// only decided inside the per-environment body.
func (t *PeriodicTask) checkAllSessions() {
	sessions := session.GetTracker().GetAll()
	if len(sessions) == 0 {
		t.l.Debugln("No active sessions to check.")
		return
	}

	t.l.Infof("Running periodic expiration check for [%d] sessions.", len(sessions))

	listTenants := func(_ context.Context) ([]tenant.Model, error) {
		seen := make(map[uuid.UUID]bool)
		ts := make([]tenant.Model, 0, len(sessions))
		for _, s := range sessions {
			if seen[s.TenantId] {
				continue
			}
			tm, err := tenant.Create(s.TenantId, s.Region, s.MajorVersion, s.MinorVersion)
			if err != nil {
				t.l.WithError(err).Warnf("Failed to create tenant model for tenant [%s].", s.TenantId)
				continue
			}
			seen[s.TenantId] = true
			ts = append(ts, tm)
		}
		return ts, nil
	}

	service.ForEachOwnedEnvironment(t.l, t.ctx, serviceName, listTenants, func(ctx context.Context) {
		tm := tenant.MustFromContext(ctx)
		pp := producer.ProviderImpl(t.l)(ctx)
		for _, s := range sessions {
			if s.TenantId != tm.Id() {
				continue
			}
			character.NewProcessor(t.l, ctx).CheckAndExpire(pp)(s.CharacterId, s.AccountId, s.Channel.WorldId())
		}
	})

	t.l.Infof("Completed periodic expiration check for [%d] sessions.", len(sessions))
}
