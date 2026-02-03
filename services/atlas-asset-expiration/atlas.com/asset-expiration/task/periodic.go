package task

import (
	"atlas-asset-expiration/character"
	"atlas-asset-expiration/kafka/producer"
	"atlas-asset-expiration/session"
	"context"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const (
	defaultInterval = 60 * time.Second
)

// PeriodicTask runs expiration checks at regular intervals for all online sessions
type PeriodicTask struct {
	l        logrus.FieldLogger
	interval time.Duration
	stopCh   chan struct{}
	wg       *sync.WaitGroup
}

// NewPeriodicTask creates a new periodic expiration check task
func NewPeriodicTask(l logrus.FieldLogger, interval time.Duration) *PeriodicTask {
	if interval <= 0 {
		interval = defaultInterval
	}
	return &PeriodicTask{
		l:        l,
		interval: interval,
		stopCh:   make(chan struct{}),
		wg:       &sync.WaitGroup{},
	}
}

// Start starts the periodic task
func (t *PeriodicTask) Start() {
	t.wg.Add(1)
	go t.run()
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

func (t *PeriodicTask) checkAllSessions() {
	sessions := session.GetTracker().GetAll()
	if len(sessions) == 0 {
		t.l.Debugln("No active sessions to check.")
		return
	}

	t.l.Infof("Running periodic expiration check for [%d] sessions.", len(sessions))

	for _, s := range sessions {
		// Create a context with tenant info from the stored session
		tm, err := tenant.Create(s.TenantId, s.Region, s.MajorVersion, s.MinorVersion)
		if err != nil {
			t.l.WithError(err).Warnf("Failed to create tenant model for character [%d].", s.CharacterId)
			continue
		}
		ctx := tenant.WithContext(context.Background(), tm)

		pp := producer.ProviderImpl(t.l)(ctx)
		character.CheckAndExpire(t.l)(pp)(ctx)(s.CharacterId, s.AccountId, s.WorldId)
	}

	t.l.Infof("Completed periodic expiration check for [%d] sessions.", len(sessions))
}
