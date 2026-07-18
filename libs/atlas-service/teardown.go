package service

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

type Manager struct {
	termChan  chan os.Signal
	doneChan  chan struct{}
	waitGroup *sync.WaitGroup
	context   context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

var (
	manager *Manager
	once    sync.Once
)

func GetTeardownManager() *Manager {
	once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())

		manager = &Manager{
			termChan:  make(chan os.Signal),
			doneChan:  make(chan struct{}),
			waitGroup: &sync.WaitGroup{},
			context:   ctx,
			cancel:    cancel,
		}

		signal.Notify(manager.termChan, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGHUP)
	})
	return manager
}

func (m *Manager) TeardownFunc(f func()) {
	m.waitGroup.Add(1)
	routine.Go(logrus.StandardLogger(), m.context, func(_ context.Context) {
		defer m.waitGroup.Done()
		<-m.doneChan
		f()
	})
}

func (m *Manager) Wait() {
	<-m.termChan
	// Idempotent: the teardown singleton cannot be re-armed, so guard the
	// close so a repeated Wait (e.g. go test -count>1) does not panic with
	// "close of closed channel".
	m.closeOnce.Do(func() {
		close(m.doneChan)
	})
	m.cancel()
	m.waitGroup.Wait()
}

func (m *Manager) WaitGroup() *sync.WaitGroup {
	return m.waitGroup
}

func (m *Manager) Context() context.Context {
	return m.context
}
