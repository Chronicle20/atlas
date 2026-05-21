package channel_test

import (
	"atlas-world/channel"
	"atlas-world/test"
	"net/http"
	"net/http/httptest"
	"testing"

	channelConstant "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/gorilla/mux"
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
	setupTestRegistry(t)
	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	// Register some test channels
	processor := channel.NewProcessor(logger, ctx)
	_, _ = processor.Register(channelConstant.NewModel(1, 0), "192.168.1.1", 8080, 0, 100)
	_, _ = processor.Register(channelConstant.NewModel(1, 1), "192.168.1.2", 8081, 50, 100)

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
	setupTestRegistry(t)
	logger, _ := logtest.NewNullLogger()

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
	setupTestRegistry(t)
	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	// Register a test channel
	processor := channel.NewProcessor(logger, ctx)
	_, _ = processor.Register(channelConstant.NewModel(1, 2), "192.168.1.1", 8080, 50, 100)

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
	setupTestRegistry(t)
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

func TestHandleGetChannelServers_InvalidWorldId(t *testing.T) {
	setupTestRegistry(t)
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
