package channel_test

import (
	"atlas-world/channel"
	"atlas-world/test"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

type testServerInformation struct{}

func (t testServerInformation) GetBaseURL() string {
	return "http://localhost:8080"
}

func (t testServerInformation) GetPrefix() string {
	return ""
}

func TestHandleGetChannelServers_Success(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()
	tenant := test.CreateDefaultMockTenant()

	// Register some test channels
	processor := channel.NewProcessor(logger, ctx)
	_, _ = processor.Register(1, 0, "192.168.1.1", 8080, 0, 100)
	_, _ = processor.Register(1, 1, "192.168.1.2", 8081, 50, 100)

	// Clean up after test
	defer func() {
		_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, 1, 0)
		_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, 1, 1)
	}()

	// Create router with the channel resource
	router := mux.NewRouter()
	channel.InitResource(testServerInformation{})(router, logger)

	// Create request
	req, err := http.NewRequest("GET", "/worlds/1/channels", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("TENANT_ID", test.DefaultTenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")

	// Record response
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify response contains data
	if rr.Body.Len() == 0 {
		t.Error("Expected response body to contain data")
	}
}

func TestHandleGetChannelServers_Empty(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	tenant := test.CreateDefaultMockTenant()

	// Ensure no channels exist for this tenant/world
	servers := channel.GetChannelRegistry().ChannelServers(tenant)
	for _, s := range servers {
		if s.WorldId() == 99 {
			_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, s.WorldId(), s.ChannelId())
		}
	}

	// Create router with the channel resource
	router := mux.NewRouter()
	channel.InitResource(testServerInformation{})(router, logger)

	// Create request for a world with no channels
	req, err := http.NewRequest("GET", "/worlds/99/channels", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("TENANT_ID", test.DefaultTenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")

	// Record response
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Check status code - should still be OK with empty array
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestHandleGetChannel_Success(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()
	tenant := test.CreateDefaultMockTenant()

	// Register a test channel
	processor := channel.NewProcessor(logger, ctx)
	_, _ = processor.Register(1, 2, "192.168.1.1", 8080, 50, 100)

	// Clean up after test
	defer func() {
		_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, 1, 2)
	}()

	// Create router with the channel resource
	router := mux.NewRouter()
	channel.InitResource(testServerInformation{})(router, logger)

	// Create request
	req, err := http.NewRequest("GET", "/worlds/1/channels/2", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("TENANT_ID", test.DefaultTenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")

	// Record response
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify response contains data
	if rr.Body.Len() == 0 {
		t.Error("Expected response body to contain data")
	}
}

func TestHandleGetChannel_NotFound(t *testing.T) {
	logger, _ := logtest.NewNullLogger()

	// Create router with the channel resource
	router := mux.NewRouter()
	channel.InitResource(testServerInformation{})(router, logger)

	// Create request for non-existent channel
	req, err := http.NewRequest("GET", "/worlds/99/channels/99", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("TENANT_ID", test.DefaultTenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")

	// Record response
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Check status code - should be 404
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestHandleRegisterChannelServer_ReturnsErrorWithoutKafka(t *testing.T) {
	// Note: This test verifies the handler correctly returns 500 when Kafka is unavailable.
	// In production with Kafka, it would return 202.
	logger, _ := logtest.NewNullLogger()

	// Create router with the channel resource
	router := mux.NewRouter()
	channel.InitResource(testServerInformation{})(router, logger)

	// Create JSON:API formatted request body
	input := channel.RestModel{
		ChannelId:       5,
		IpAddress:       "10.0.0.1",
		Port:            9090,
		CurrentCapacity: 0,
		MaxCapacity:     200,
	}
	body, err := jsonapi.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}

	// Create request
	req, err := http.NewRequest("POST", "/worlds/1/channels", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("TENANT_ID", test.DefaultTenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")

	// Record response
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Without Kafka infrastructure, the handler should return 500
	// This verifies the error handling path works correctly
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v (without Kafka)", status, http.StatusInternalServerError)
	}
}

func TestHandleGetChannelServers_InvalidWorldId(t *testing.T) {
	logger, _ := logtest.NewNullLogger()

	// Create router with the channel resource
	router := mux.NewRouter()
	channel.InitResource(testServerInformation{})(router, logger)

	// Create request with invalid world ID
	req, err := http.NewRequest("GET", "/worlds/invalid/channels", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("TENANT_ID", test.DefaultTenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")

	// Record response
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Check status code - should be 400 Bad Request
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}
