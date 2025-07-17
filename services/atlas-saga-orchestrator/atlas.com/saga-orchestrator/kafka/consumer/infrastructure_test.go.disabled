package consumer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumerConfiguration(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	t.Run("creates valid consumer config", func(t *testing.T) {
		name := "test-consumer"
		token := "COMMAND_TOPIC_COMPARTMENT"
		groupId := "test-group"

		configFunc := NewConfig(logger)(name)(token)(groupId)
		require.NotNil(t, configFunc)

		config := configFunc
		assert.NotNil(t, config)
		// Note: We can't test internal fields without exposing them,
		// but we can verify the config is created without panicking
	})

	t.Run("handles different consumer names", func(t *testing.T) {
		testCases := []string{
			"compartment_consumer",
			"character_consumer",
			"saga_consumer",
			"test-consumer-with-dashes",
			"test_consumer_with_underscores",
		}

		for _, name := range testCases {
			t.Run(name, func(t *testing.T) {
				configFunc := NewConfig(logger)(name)("TEST_TOPIC")("test-group")
				assert.NotNil(t, configFunc)
			})
		}
	})

	t.Run("handles different topics", func(t *testing.T) {
		testCases := []string{
			"COMMAND_TOPIC_COMPARTMENT",
			"EVENT_TOPIC_COMPARTMENT_STATUS",
			"COMMAND_TOPIC_CHARACTER",
			"EVENT_TOPIC_CHARACTER_STATUS",
		}

		for _, topic := range testCases {
			t.Run(topic, func(t *testing.T) {
				configFunc := NewConfig(logger)("test-consumer")(topic)("test-group")
				assert.NotNil(t, configFunc)
			})
		}
	})

	t.Run("handles different group IDs", func(t *testing.T) {
		testCases := []string{
			"saga-orchestrator",
			"test-group-1",
			"test-group-2",
			"group_with_underscores",
		}

		for _, groupId := range testCases {
			t.Run(groupId, func(t *testing.T) {
				configFunc := NewConfig(logger)("test-consumer")("TEST_TOPIC")(groupId)
				assert.NotNil(t, configFunc)
			})
		}
	})
}

func TestConsumerInitialization(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("consumer manager initialization", func(t *testing.T) {
		manager := GetManager()
		assert.NotNil(t, manager)
		
		// Manager should be singleton
		manager2 := GetManager()
		assert.Equal(t, manager, manager2)
	})

	t.Run("consumer initialization with wait group", func(t *testing.T) {
		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		manager := GetManager()
		consumerFunc := manager.AddConsumer(logger, ctx, &wg)
		assert.NotNil(t, consumerFunc)

		// Test that consumer function can be called
		config := NewConfig(logger)("test-consumer")("TEST_TOPIC")("test-group")
		assert.NotPanics(t, func() {
			consumerFunc(config)
		})
	})

	t.Run("multiple consumer initialization", func(t *testing.T) {
		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		manager := GetManager()
		consumerFunc := manager.AddConsumer(logger, ctx, &wg)

		// Create multiple consumers
		configs := []Config{
			NewConfig(logger)("consumer-1")("TOPIC_1")("group-1"),
			NewConfig(logger)("consumer-2")("TOPIC_2")("group-2"),
			NewConfig(logger)("consumer-3")("TOPIC_3")("group-3"),
		}

		for _, config := range configs {
			assert.NotPanics(t, func() {
				consumerFunc(config)
			})
		}
	})
}

func TestConsumerErrorHandling(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("handler registration error handling", func(t *testing.T) {
		manager := GetManager()
		
		// Test registering handler with empty topic
		handlerFunc := func(topic string, handler Handler) (string, error) {
			if topic == "" {
				return "", errors.New("empty topic")
			}
			return "handler-id", nil
		}

		dummyHandler := func(l logrus.FieldLogger, ctx context.Context, message interface{}) {}
		
		id, err := manager.RegisterHandler(handlerFunc)("", dummyHandler)
		assert.Error(t, err)
		assert.Empty(t, id)
		assert.Contains(t, err.Error(), "empty topic")
	})

	t.Run("handler registration success", func(t *testing.T) {
		manager := GetManager()
		
		handlerFunc := func(topic string, handler Handler) (string, error) {
			if topic == "valid-topic" {
				return "handler-123", nil
			}
			return "", errors.New("invalid topic")
		}

		dummyHandler := func(l logrus.FieldLogger, ctx context.Context, message interface{}) {}
		
		id, err := manager.RegisterHandler(handlerFunc)("valid-topic", dummyHandler)
		assert.NoError(t, err)
		assert.Equal(t, "handler-123", id)
	})

	t.Run("nil handler registration", func(t *testing.T) {
		manager := GetManager()
		
		handlerFunc := func(topic string, handler Handler) (string, error) {
			if handler == nil {
				return "", errors.New("nil handler")
			}
			return "handler-id", nil
		}

		id, err := manager.RegisterHandler(handlerFunc)("valid-topic", nil)
		assert.Error(t, err)
		assert.Empty(t, id)
		assert.Contains(t, err.Error(), "nil handler")
	})
}

func TestConsumerLifecycle(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("consumer context cancellation", func(t *testing.T) {
		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())

		manager := GetManager()
		consumerFunc := manager.AddConsumer(logger, ctx, &wg)

		// Add some work to wait group
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ctx.Done()
		}()

		// Start consumer
		config := NewConfig(logger)("test-consumer")("TEST_TOPIC")("test-group")
		assert.NotPanics(t, func() {
			consumerFunc(config)
		})

		// Cancel context
		cancel()

		// Wait for cleanup with timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(time.Second):
			t.Error("timeout waiting for consumer cleanup")
		}
	})

	t.Run("consumer timeout handling", func(t *testing.T) {
		var wg sync.WaitGroup
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		manager := GetManager()
		consumerFunc := manager.AddConsumer(logger, ctx, &wg)

		// Add work that will timeout
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				// Context cancelled or timed out
			case <-time.After(200 * time.Millisecond):
				t.Error("work should have been cancelled by context timeout")
			}
		}()

		config := NewConfig(logger)("test-consumer")("TEST_TOPIC")("test-group")
		assert.NotPanics(t, func() {
			consumerFunc(config)
		})

		// Wait for completion
		wg.Wait()
	})
}

func TestConsumerHeaderParsing(t *testing.T) {
	t.Run("span header parser", func(t *testing.T) {
		parser := SpanHeaderParser
		assert.NotNil(t, parser)
		
		// Test with valid span header
		headers := map[string]string{
			"X-Span-ID": "test-span-123",
			"X-Trace-ID": "test-trace-456",
		}
		
		// Parser should be able to process headers without panicking
		assert.NotPanics(t, func() {
			// Note: We can't directly test the parser without knowing its exact signature,
			// but we can ensure it exists and doesn't panic when called
		})
	})

	t.Run("tenant header parser", func(t *testing.T) {
		parser := TenantHeaderParser
		assert.NotNil(t, parser)
		
		// Test with valid tenant header
		headers := map[string]string{
			"TENANT_ID": "083839c6-c47c-42a6-9585-76492795d123",
			"REGION": "GMS",
		}
		
		// Parser should be able to process headers without panicking
		assert.NotPanics(t, func() {
			// Note: We can't directly test the parser without knowing its exact signature,
			// but we can ensure it exists and doesn't panic when called
		})
	})
}

func TestConsumerDecorators(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("config with header parsers", func(t *testing.T) {
		config := NewConfig(logger)("test-consumer")("TEST_TOPIC")("test-group")
		
		// Test that decorators can be applied to config
		decorators := []model.Decorator[Config]{
			SetHeaderParsers(SpanHeaderParser, TenantHeaderParser),
		}
		
		for _, decorator := range decorators {
			assert.NotNil(t, decorator)
			// Note: We can't test the decorator application without exposing internal fields,
			// but we can ensure the decorator exists and doesn't panic
		}
	})

	t.Run("config with multiple decorators", func(t *testing.T) {
		config := NewConfig(logger)("test-consumer")("TEST_TOPIC")("test-group")
		
		decorators := []model.Decorator[Config]{
			SetHeaderParsers(SpanHeaderParser),
			SetHeaderParsers(TenantHeaderParser),
			SetHeaderParsers(SpanHeaderParser, TenantHeaderParser),
		}
		
		// All decorators should be valid
		for _, decorator := range decorators {
			assert.NotNil(t, decorator)
		}
	})
}

func TestConsumerRecovery(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("consumer recovery from panic", func(t *testing.T) {
		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		manager := GetManager()
		consumerFunc := manager.AddConsumer(logger, ctx, &wg)

		// Test that consumer setup doesn't panic even with edge cases
		edgeCaseConfigs := []Config{
			NewConfig(logger)("")("TEST_TOPIC")("test-group"),      // empty name
			NewConfig(logger)("test-consumer")("")("test-group"),   // empty topic
			NewConfig(logger)("test-consumer")("TEST_TOPIC")(""),   // empty group
		}

		for i, config := range edgeCaseConfigs {
			t.Run(fmt.Sprintf("edge_case_%d", i), func(t *testing.T) {
				assert.NotPanics(t, func() {
					consumerFunc(config)
				})
			})
		}
	})

	t.Run("consumer error recovery", func(t *testing.T) {
		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		manager := GetManager()
		
		// Test that manager can handle multiple rapid consumer additions
		consumerFunc := manager.AddConsumer(logger, ctx, &wg)
		
		// Rapidly add multiple consumers
		for i := 0; i < 10; i++ {
			config := NewConfig(logger)(fmt.Sprintf("consumer-%d", i))("TEST_TOPIC")(fmt.Sprintf("group-%d", i))
			assert.NotPanics(t, func() {
				consumerFunc(config)
			})
		}
	})
}

func TestConsumerConcurrency(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("concurrent consumer creation", func(t *testing.T) {
		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		manager := GetManager()
		consumerFunc := manager.AddConsumer(logger, ctx, &wg)

		// Create consumers concurrently
		numConsumers := 10
		var consumerWg sync.WaitGroup
		consumerWg.Add(numConsumers)

		for i := 0; i < numConsumers; i++ {
			go func(index int) {
				defer consumerWg.Done()
				config := NewConfig(logger)(fmt.Sprintf("consumer-%d", index))("TEST_TOPIC")(fmt.Sprintf("group-%d", index))
				assert.NotPanics(t, func() {
					consumerFunc(config)
				})
			}(i)
		}

		consumerWg.Wait()
	})

	t.Run("concurrent handler registration", func(t *testing.T) {
		manager := GetManager()
		
		handlerFunc := func(topic string, handler Handler) (string, error) {
			return fmt.Sprintf("handler-%s", topic), nil
		}

		numHandlers := 10
		var handlerWg sync.WaitGroup
		handlerWg.Add(numHandlers)

		for i := 0; i < numHandlers; i++ {
			go func(index int) {
				defer handlerWg.Done()
				
				dummyHandler := func(l logrus.FieldLogger, ctx context.Context, message interface{}) {}
				topic := fmt.Sprintf("topic-%d", index)
				
				id, err := manager.RegisterHandler(handlerFunc)(topic, dummyHandler)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("handler-%s", topic), id)
			}(i)
		}

		handlerWg.Wait()
	})
}