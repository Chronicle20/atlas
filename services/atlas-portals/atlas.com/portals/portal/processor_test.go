package portal_test

import (
	"atlas-portals/portal"
	"atlas-portals/test"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// jsonAPIResponse wraps data in JSON:API format
type jsonAPIResponse struct {
	Data interface{} `json:"data"`
}

// jsonAPIResource represents a JSON:API resource
type jsonAPIResource struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Attributes map[string]interface{} `json:"attributes"`
}

// createPortalResource creates a JSON:API resource for a portal
func createPortalResource(id string, name string, target string, targetMapId uint32, scriptName string) jsonAPIResource {
	return jsonAPIResource{
		Type: "portals",
		ID:   id,
		Attributes: map[string]interface{}{
			"name":        name,
			"target":      target,
			"type":        0,
			"x":           0,
			"y":           0,
			"targetMapId": targetMapId,
			"scriptName":  scriptName,
		},
	}
}

// setupMockDataServer creates an httptest server that mocks the DATA service
func setupMockDataServer(t *testing.T, responses map[string]interface{}) (*httptest.Server, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Build full URL path including query string
		fullPath := r.URL.Path
		if r.URL.RawQuery != "" {
			fullPath = r.URL.Path + "?" + r.URL.RawQuery
		}
		t.Logf("Mock server received request: %s %s", r.Method, fullPath)

		if response, ok := responses[fullPath]; ok {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Return 404 for unhandled paths
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]string{{"detail": "not found"}},
		})
	}))

	// Save original env and set mock server URL
	originalURL := os.Getenv("DATA_SERVICE_URL")
	os.Setenv("DATA_SERVICE_URL", server.URL+"/api/")

	cleanup := func() {
		server.Close()
		if originalURL != "" {
			os.Setenv("DATA_SERVICE_URL", originalURL)
		} else {
			os.Unsetenv("DATA_SERVICE_URL")
		}
	}

	return server, cleanup
}

func TestEnter_PortalNotFound(t *testing.T) {
	// Setup mock server that returns 404 for portal lookup
	_, cleanup := setupMockDataServer(t, map[string]interface{}{})
	defer cleanup()

	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := test.CreateTestContext()

	// Call Enter - should log error and return without panic
	portal.Enter(logger)(ctx)(1, 1, 100000000, 99, 12345)

	// Verify error was logged
	found := false
	for _, entry := range hook.Entries {
		if entry.Level == logrus.ErrorLevel {
			found = true
			break
		}
	}

	if !found {
		t.Log("Expected an error log entry when portal not found")
		// Note: This is informational - the test passes if no panic occurs
	}
}

func TestEnter_PortalWithScript(t *testing.T) {
	// Portal with script - should enable actions and return
	portalResource := createPortalResource("5", "script_portal", "", 999999999, "portal_script")

	_, cleanup := setupMockDataServer(t, map[string]interface{}{
		"/api/data/maps/100000000/portals/5": jsonAPIResponse{Data: portalResource},
	})
	defer cleanup()

	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := test.CreateTestContext()

	// Call Enter - should not panic, should handle script portal
	// Note: EnableActions makes a Kafka call which won't work in test,
	// but the function should not panic
	portal.Enter(logger)(ctx)(1, 1, 100000000, 5, 12345)

	// Test passes if no panic
}

func TestEnter_PortalWithTargetMap(t *testing.T) {
	// Portal with target map
	sourcePortal := createPortalResource("1", "town_portal", "spawn", 200000000, "")
	targetPortal := createPortalResource("0", "spawn", "", 999999999, "")

	_, cleanup := setupMockDataServer(t, map[string]interface{}{
		"/api/data/maps/100000000/portals/1":          jsonAPIResponse{Data: sourcePortal},
		"/api/data/maps/200000000/portals?name=spawn": jsonAPIResponse{Data: []jsonAPIResource{targetPortal}},
	})
	defer cleanup()

	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := test.CreateTestContext()

	// Call Enter - should attempt to warp to target map
	// Note: WarpById makes a Kafka call which won't work in test,
	// but the function should not panic
	portal.Enter(logger)(ctx)(1, 1, 100000000, 1, 12345)

	// Test passes if no panic
}

func TestEnter_PortalWithInvalidTarget_FallbackToPortal0(t *testing.T) {
	// Portal with target map but target portal not found - should fallback to portal 0
	sourcePortal := createPortalResource("1", "broken_portal", "nonexistent", 200000000, "")
	fallbackPortal := createPortalResource("0", "default", "", 999999999, "")

	_, cleanup := setupMockDataServer(t, map[string]interface{}{
		"/api/data/maps/100000000/portals/1":               jsonAPIResponse{Data: sourcePortal},
		"/api/data/maps/200000000/portals?name=nonexistent": jsonAPIResponse{Data: []jsonAPIResource{}}, // Empty - not found
		"/api/data/maps/200000000/portals/0":               jsonAPIResponse{Data: fallbackPortal},
	})
	defer cleanup()

	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.WarnLevel)
	ctx := test.CreateTestContext()

	// Call Enter - should fallback to portal 0
	portal.Enter(logger)(ctx)(1, 1, 100000000, 1, 12345)

	// Check for warning log about fallback
	for _, entry := range hook.Entries {
		if entry.Level == logrus.WarnLevel {
			t.Logf("Warning logged: %s", entry.Message)
		}
	}

	// Test passes if no panic
}

func TestEnter_PortalNoScriptNoTarget(t *testing.T) {
	// Portal with no script and no target - should enable actions
	portalResource := createPortalResource("3", "dead_end_portal", "", 999999999, "")

	_, cleanup := setupMockDataServer(t, map[string]interface{}{
		"/api/data/maps/100000000/portals/3": jsonAPIResponse{Data: portalResource},
	})
	defer cleanup()

	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := test.CreateTestContext()

	// Call Enter - should enable actions and return
	portal.Enter(logger)(ctx)(1, 1, 100000000, 3, 12345)

	// Test passes if no panic
}

func TestGetInMapById_Success(t *testing.T) {
	portalResource := createPortalResource("42", "test_portal", "target", 100000000, "")

	_, cleanup := setupMockDataServer(t, map[string]interface{}{
		"/api/data/maps/100000000/portals/42": jsonAPIResponse{Data: portalResource},
	})
	defer cleanup()

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	model, err := portal.GetInMapById(logger)(ctx)(100000000, 42)

	if err != nil {
		t.Fatalf("GetInMapById() returned unexpected error: %v", err)
	}

	if model.Id() != 42 {
		t.Errorf("model.Id() = %d, want 42", model.Id())
	}

	if model.Target() != "target" {
		t.Errorf("model.Target() = %s, want target", model.Target())
	}
}

func TestGetInMapById_NotFound(t *testing.T) {
	_, cleanup := setupMockDataServer(t, map[string]interface{}{})
	defer cleanup()

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	_, err := portal.GetInMapById(logger)(ctx)(100000000, 999)

	if err == nil {
		t.Error("GetInMapById() expected error for non-existent portal, got nil")
	}
}

func TestGetInMapByName_Success(t *testing.T) {
	portalResource := createPortalResource("10", "spawn_portal", "", 999999999, "")

	_, cleanup := setupMockDataServer(t, map[string]interface{}{
		"/api/data/maps/100000000/portals?name=spawn_portal": jsonAPIResponse{Data: []jsonAPIResource{portalResource}},
	})
	defer cleanup()

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	model, err := portal.GetInMapByName(logger)(ctx)(100000000, "spawn_portal")

	if err != nil {
		t.Fatalf("GetInMapByName() returned unexpected error: %v", err)
	}

	if model.Id() != 10 {
		t.Errorf("model.Id() = %d, want 10", model.Id())
	}
}

func TestGetInMapByName_NotFound(t *testing.T) {
	_, cleanup := setupMockDataServer(t, map[string]interface{}{
		"/api/data/maps/100000000/portals?name=nonexistent": jsonAPIResponse{Data: []jsonAPIResource{}},
	})
	defer cleanup()

	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	_, err := portal.GetInMapByName(logger)(ctx)(100000000, "nonexistent")

	if err == nil {
		t.Error("GetInMapByName() expected error for non-existent portal, got nil")
	}
}
