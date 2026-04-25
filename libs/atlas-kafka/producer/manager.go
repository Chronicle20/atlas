package producer

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// WriterFactory builds a Writer for a resolved topic name. Tests inject
// a stub via ConfigWriterFactory; production uses defaultWriterFactory.
type WriterFactory func(topicName string) Writer

type ManagerConfig func(m *Manager)

//goland:noinspection GoUnusedExportedFunction
func ConfigWriterFactory(wf WriterFactory) ManagerConfig {
	return func(m *Manager) { m.wf = wf }
}

type Manager struct {
	mu      sync.RWMutex
	writers map[string]Writer
	wf      WriterFactory
	closed  bool
}

var (
	manager     *Manager
	managerOnce sync.Once
)

// ResetInstance clears the singleton. Test-only.
//
//goland:noinspection GoUnusedExportedFunction
func ResetInstance() {
	manager = nil
	managerOnce = sync.Once{}
}

//goland:noinspection GoUnusedExportedFunction
func GetManager(configurators ...ManagerConfig) *Manager {
	managerOnce.Do(func() {
		manager = &Manager{
			writers: make(map[string]Writer),
			wf:      defaultWriterFactory,
		}
		for _, c := range configurators {
			c(manager)
		}
	})
	return manager
}

var ErrManagerClosed = errors.New("producer manager is closed")

// Writer returns the long-lived Writer for the topic resolved from token,
// constructing it on first request. Concurrent first-touches return the
// same instance.
func (m *Manager) Writer(l logrus.FieldLogger, token string) (Writer, error) {
	t, err := topic.EnvProvider(l)(token)()
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return nil, ErrManagerClosed
	}
	if w, ok := m.writers[t]; ok {
		m.mu.RUnlock()
		return w, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, ErrManagerClosed
	}
	if w, ok := m.writers[t]; ok { // double-check after acquiring write lock
		return w, nil
	}
	w := m.wf(t)
	m.writers[t] = w
	l.Infof("Created kafka writer for topic [%s].", t)
	return w, nil
}

// Close closes every registered Writer and marks the manager closed.
// Idempotent: subsequent calls are no-ops. Errors from individual
// Writer.Close calls are logged but do not short-circuit the loop.
func (m *Manager) Close(l logrus.FieldLogger) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true

	var errCount int
	for t, w := range m.writers {
		if err := w.Close(); err != nil {
			errCount++
			l.WithError(err).Warnf("Error closing kafka writer for topic [%s].", t)
		}
	}
	l.Infof("Producer manager shut down %d writers (errors=%d).", len(m.writers), errCount)
	return nil
}

func defaultWriterFactory(topicName string) Writer {
	return WriterImpl{w: &kafka.Writer{
		Addr:                   kafka.TCP(os.Getenv("BOOTSTRAP_SERVERS")),
		Topic:                  topicName,
		Balancer:               &kafka.LeastBytes{},
		BatchTimeout:           50 * time.Millisecond,
		AllowAutoTopicCreation: true,
	}}
}

// ManagerWriterProvider returns a model.Provider[Writer] backed by the
// process-wide manager. Replaces the deleted WriterProvider helper.
//
//goland:noinspection GoUnusedExportedFunction
func ManagerWriterProvider(l logrus.FieldLogger) func(token string) model.Provider[Writer] {
	return func(token string) model.Provider[Writer] {
		return func() (Writer, error) {
			return GetManager().Writer(l, token)
		}
	}
}
