package producer

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func TestProviderImpl(t *testing.T) {
	logger := logrus.New()
	
	providerFunc := ProviderImpl(logger)
	
	if providerFunc == nil {
		t.Error("ProviderImpl() returned nil")
	}
	
	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	if err != nil {
		t.Fatalf("Failed to create tenant model: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tenantModel)
	
	contextFunc := providerFunc(ctx)
	
	if contextFunc == nil {
		t.Error("ProviderImpl()(ctx) returned nil")
	}
	
	token := "test-token"
	messageProducer := contextFunc(token)
	
	if messageProducer == nil {
		t.Error("ProviderImpl()(ctx)(token) returned nil")
	}
}

func TestProviderImpl_CurriedFunctionStructure(t *testing.T) {
	logger := logrus.New()
	
	// Test the curried function structure
	providerFunc := ProviderImpl(logger)
	
	// First level: accepts logger, returns function
	if providerFunc == nil {
		t.Error("First level function returned nil")
	}
	
	// Second level: accepts context, returns function
	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	if err != nil {
		t.Fatalf("Failed to create tenant model: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tenantModel)
	contextFunc := providerFunc(ctx)
	
	if contextFunc == nil {
		t.Error("Second level function returned nil")
	}
	
	// Third level: accepts token, returns MessageProducer
	token := "test-token"
	messageProducer := contextFunc(token)
	
	if messageProducer == nil {
		t.Error("Third level function returned nil")
	}
}

func TestProviderImpl_DifferentTokens(t *testing.T) {
	logger := logrus.New()
	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	if err != nil {
		t.Fatalf("Failed to create tenant model: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tenantModel)
	
	contextFunc := ProviderImpl(logger)(ctx)
	
	// Test with different tokens
	token1 := "token1"
	token2 := "token2"
	
	producer1 := contextFunc(token1)
	producer2 := contextFunc(token2)
	
	if producer1 == nil {
		t.Error("Producer for token1 is nil")
	}
	
	if producer2 == nil {
		t.Error("Producer for token2 is nil")
	}
	
	// Both should be non-nil
	// Note: We can't compare function instances directly in Go
}

func TestProviderImpl_DifferentContexts(t *testing.T) {
	logger := logrus.New()
	
	providerFunc := ProviderImpl(logger)
	
	// Test with different contexts
	tenantId1 := uuid.New()
	tenantId2 := uuid.New()
	tenantModel1, err := tenant.Create(tenantId1, "test-region", 1, 0)
	if err != nil {
		t.Fatalf("Failed to create tenant model 1: %v", err)
	}
	tenantModel2, err := tenant.Create(tenantId2, "test-region", 1, 0)
	if err != nil {
		t.Fatalf("Failed to create tenant model 2: %v", err)
	}
	ctx1 := tenant.WithContext(context.Background(), tenantModel1)
	ctx2 := tenant.WithContext(context.Background(), tenantModel2)
	
	contextFunc1 := providerFunc(ctx1)
	contextFunc2 := providerFunc(ctx2)
	
	if contextFunc1 == nil {
		t.Error("Context function for ctx1 is nil")
	}
	
	if contextFunc2 == nil {
		t.Error("Context function for ctx2 is nil")
	}
	
	// Test that both can create producers
	token := "test-token"
	producer1 := contextFunc1(token)
	producer2 := contextFunc2(token)
	
	if producer1 == nil {
		t.Error("Producer from ctx1 is nil")
	}
	
	if producer2 == nil {
		t.Error("Producer from ctx2 is nil")
	}
}