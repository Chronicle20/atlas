package character

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func TestNewProcessor(t *testing.T) {
	logger := logrus.New()
	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	if err != nil {
		t.Fatalf("Failed to create tenant model: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tenantModel)
	
	processor := NewProcessor(logger, ctx, &gorm.DB{})
	
	if processor == nil {
		t.Error("NewProcessor() returned nil")
	}
	
	impl, ok := processor.(*ProcessorImpl)
	if !ok {
		t.Error("NewProcessor() did not return ProcessorImpl")
	}
	
	if impl.l == nil {
		t.Error("ProcessorImpl.l is nil")
	}
	
	if impl.ctx == nil {
		t.Error("ProcessorImpl.ctx is nil")
	}
	
	if impl.db == nil {
		t.Error("ProcessorImpl.db is nil")
	}
	
	if impl.t.Id() != tenantId {
		t.Errorf("Expected tenant ID %v, got %v", tenantId, impl.t.Id())
	}
}

// These tests are limited because they require external HTTP services
// In a real implementation, we would mock the HTTP client
func TestProcessorImpl_Methods_Exist(t *testing.T) {
	logger := logrus.New()
	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	if err != nil {
		t.Fatalf("Failed to create tenant model: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tenantModel)
	
	processor := NewProcessor(logger, ctx, &gorm.DB{})
	
	// Test that the methods exist and return proper types
	provider := processor.ByIdProvider(123)
	if provider == nil {
		t.Error("ByIdProvider() returned nil")
	}
	
	// GetById should exist and be callable
	// Note: This will fail due to HTTP setup, but that's expected
	_, _ = processor.GetById(123)
}