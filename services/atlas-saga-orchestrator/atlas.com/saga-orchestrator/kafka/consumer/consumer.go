package consumer

import (
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
)

func NewConfig(l logrus.FieldLogger) func(name string) func(token string) func(groupId string) consumer.Config {
	return func(name string) func(token string) func(groupId string) consumer.Config {
		return func(token string) func(groupId string) consumer.Config {
			t, _ := topic.EnvProvider(l)(token)()
			return func(groupId string) consumer.Config {
				return consumer.NewConfig(LookupBrokers(), name, t, groupId)
			}
		}
	}
}

func LookupBrokers() []string {
	return []string{os.Getenv("BOOTSTRAP_SERVERS")}
}

// Config is an alias for consumer.Config
type Config = consumer.Config

// Handler is an alias for handler.Handler  
type Handler = handler.Handler

// Manager interface for consumer management
type Manager interface {
	AddConsumer(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(config Config, decorators ...model.Decorator[Config])
	RegisterHandler(topic string, handler Handler) (string, error)
}

// ManagerImpl is the implementation of Manager
type ManagerImpl struct {
	consumers map[string]Config
	handlers  map[string][]Handler
	mutex     sync.RWMutex
}

// Global manager instance
var manager Manager

// GetManager returns the global consumer manager instance
func GetManager() Manager {
	if manager == nil {
		manager = &ManagerImpl{
			consumers: make(map[string]Config),
			handlers:  make(map[string][]Handler),
		}
	}
	return manager
}

// AddConsumer adds a consumer configuration
func (m *ManagerImpl) AddConsumer(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(config Config, decorators ...model.Decorator[Config]) {
	return func(config Config, decorators ...model.Decorator[Config]) {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		
		// Apply decorators to the config
		for _, decorator := range decorators {
			config = decorator(config)
		}
		
		// Store configuration using string representation as key
		configKey := fmt.Sprintf("%p", &config)
		m.consumers[configKey] = config
		
		l.Debug("Consumer configuration added")
	}
}

// RegisterHandler registers a handler for a specific topic
func (m *ManagerImpl) RegisterHandler(topic string, handler Handler) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.handlers[topic] == nil {
		m.handlers[topic] = make([]Handler, 0)
	}
	
	m.handlers[topic] = append(m.handlers[topic], handler)
	
	return topic, nil
}
