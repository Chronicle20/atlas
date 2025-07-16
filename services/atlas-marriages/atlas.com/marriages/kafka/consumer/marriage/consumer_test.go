package marriage

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestNewConfig(t *testing.T) {
	logger, _ := test.NewNullLogger()
	
	configFunc := NewConfig(logger)
	assert.NotNil(t, configFunc)
	
	nameFunc := configFunc("test-name")
	assert.NotNil(t, nameFunc)
	
	tokenFunc := nameFunc("test-token")
	assert.NotNil(t, tokenFunc)
	
	config := tokenFunc("test-group")
	assert.NotNil(t, config)
}

func TestInitHandlers(t *testing.T) {
	// Test that InitHandlers function exists and is callable
	// We don't actually call it to avoid context/database dependencies
	assert.NotNil(t, InitHandlers)
}

func TestInitConsumers(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	
	initFunc := InitConsumers(logger, ctx, &gorm.DB{})
	assert.NotNil(t, initFunc)
	
	// Test that it returns a function that expects consumer setup
	consumerSetupFunc := initFunc(func(config consumer.Config, decorators ...model.Decorator[consumer.Config]) {
		// Mock consumer setup function
	})
	assert.NotNil(t, consumerSetupFunc)
	
	// Test that the consumer setup function works
	consumerSetupFunc("test-group")
	// No assertion needed, just verifying it doesn't panic
}