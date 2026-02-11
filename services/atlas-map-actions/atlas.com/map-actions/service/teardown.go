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

var once sync.Once
var tdm *Manager

func GetTeardownManager() *Manager {
	once.Do(func() {
		tdm = &Manager{}
		tdm.termChan = make(chan os.Signal, 1)
		tdm.doneChan = make(chan struct{})
		signal.Notify(tdm.termChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		tdm.waitGroup = &sync.WaitGroup{}
		tdm.context, tdm.cancel = context.WithCancel(context.Background())
	})
	return tdm
}

func (m *Manager) Context() context.Context {
	return m.context
}

func (m *Manager) WaitGroup() *sync.WaitGroup {
	return m.waitGroup
}

func (m *Manager) TeardownFunc(f func()) {
	go func() {
		<-m.termChan
		m.cancel()
		m.waitGroup.Wait()
		f()
		close(m.doneChan)
	}()
}

func (m *Manager) Wait() {
	<-m.doneChan
}
