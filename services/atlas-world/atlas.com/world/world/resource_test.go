package world_test

import (
	"atlas-world/channel"
	"atlas-world/test"
	"atlas-world/world"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestHandleGetWorlds_NoChannelsRegistered(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	tenant := test.CreateDefaultMockTenant()

	// Ensure no channels exist for this tenant
	servers := channel.GetChannelRegistry().ChannelServers(tenant)
	for _, s := range servers {
		_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, s.WorldId(), s.ChannelId())
	}

	// Create router with the world resource
	router := mux.NewRouter()
	world.InitResource(testServerInformation{})(router, logger)

	// Create request
	req, err := http.NewRequest("GET", "/worlds/", nil)
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

	// When no channels are registered, should return OK with empty array
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestHandleGetWorld_NotFound(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	tenant := test.CreateDefaultMockTenant()

	// Ensure no channels exist for this tenant
	servers := channel.GetChannelRegistry().ChannelServers(tenant)
	for _, s := range servers {
		_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, s.WorldId(), s.ChannelId())
	}

	// Create router with the world resource
	router := mux.NewRouter()
	world.InitResource(testServerInformation{})(router, logger)

	// Create request for a non-existent world
	req, err := http.NewRequest("GET", "/worlds/99", nil)
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

	// World doesn't exist because no channels are registered for it
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestHandleGetWorld_InvalidWorldId(t *testing.T) {
	logger, _ := logtest.NewNullLogger()

	// Create router with the world resource
	router := mux.NewRouter()
	world.InitResource(testServerInformation{})(router, logger)

	// Create request with invalid world ID
	req, err := http.NewRequest("GET", "/worlds/invalid", nil)
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

	// Should return 400 Bad Request for invalid world ID
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

// Note: Testing TestHandleGetWorld_Success and TestHandleGetWorlds_Success
// requires the configuration service to be available with proper tenant config.
// These are integration tests that would need the full service infrastructure.
// The tests above verify the error handling paths work correctly.
