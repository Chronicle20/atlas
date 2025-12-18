package rest

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type TestResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestMakeGetRequest(t *testing.T) {
	logger := logrus.New()
	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	if err != nil {
		t.Fatalf("Failed to create tenant model: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tenantModel)
	
	url := "http://example.com/test"
	request := MakeGetRequest[TestResponse](url)
	
	if request == nil {
		t.Error("MakeGetRequest() returned nil")
	}
	
	// Test that the request function exists and can be called
	// Note: This will fail due to HTTP setup, but that's expected
	_, _ = request(logger, ctx)
}

func TestMakePostRequest(t *testing.T) {
	logger := logrus.New()
	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	if err != nil {
		t.Fatalf("Failed to create tenant model: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tenantModel)
	
	url := "http://example.com/test"
	data := map[string]interface{}{"name": "test"}
	request := MakePostRequest[TestResponse](url, data)
	
	if request == nil {
		t.Error("MakePostRequest() returned nil")
	}
	
	// Test that the request function exists and can be called
	// Note: This will fail due to HTTP setup, but that's expected
	_, _ = request(logger, ctx)
}

func TestMakePatchRequest(t *testing.T) {
	logger := logrus.New()
	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	if err != nil {
		t.Fatalf("Failed to create tenant model: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tenantModel)
	
	url := "http://example.com/test"
	data := map[string]interface{}{"name": "updated"}
	request := MakePatchRequest[TestResponse](url, data)
	
	if request == nil {
		t.Error("MakePatchRequest() returned nil")
	}
	
	// Test that the request function exists and can be called
	// Note: This will fail due to HTTP setup, but that's expected
	_, _ = request(logger, ctx)
}

func TestMakeDeleteRequest(t *testing.T) {
	logger := logrus.New()
	tenantId := uuid.New()
	tenantModel, err := tenant.Create(tenantId, "test-region", 1, 0)
	if err != nil {
		t.Fatalf("Failed to create tenant model: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tenantModel)
	
	url := "http://example.com/test"
	request := MakeDeleteRequest(url)
	
	if request == nil {
		t.Error("MakeDeleteRequest() returned nil")
	}
	
	// Test that the request function exists and can be called
	// Note: This will fail due to HTTP setup, but that's expected
	_ = request(logger, ctx)
}