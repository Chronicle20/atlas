package service

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Manager struct {
	termChan  chan os.Signal
	doneChan  chan struct{}
	waitGroup *sync.WaitGroup
	context   context.Context
	cancel    context.CancelFunc
}

var manager *Manager
var once sync.Once

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
	go func() {
		m.waitGroup.Add(1)
		defer m.waitGroup.Done()
		<-m.doneChan
		f()
	}()
}

func (m *Manager) Wait() {
	<-m.termChan
	close(m.doneChan)
	m.cancel()
	m.waitGroup.Wait()
}

func (m *Manager) WaitGroup() *sync.WaitGroup {
	return m.waitGroup
}

func (m *Manager) Context() context.Context {
	return m.context
}
