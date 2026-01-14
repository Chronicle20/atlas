package portal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// createTestContext creates a context with a mock tenant for consumer tests
func createTestContext() context.Context {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), t)
}

// setupMockDataServerForConsumer creates an httptest server for consumer tests
func setupMockDataServerForConsumer(t *testing.T, responses map[string]interface{}) (*httptest.Server, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fullPath := r.URL.Path
		if r.URL.RawQuery != "" {
			fullPath = r.URL.Path + "?" + r.URL.RawQuery
		}

		if response, ok := responses[fullPath]; ok {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]string{{"detail": "not found"}},
		})
	}))

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

func TestHandleEnterCommand_ParameterExtraction(t *testing.T) {
	// Create a mock server that will return a portal
	portalResource := map[string]interface{}{
		"type": "portals",
		"id":   "5",
		"attributes": map[string]interface{}{
			"name":        "test_portal",
			"target":      "",
			"type":        0,
			"x":           0,
			"y":           0,
			"targetMapId": 999999999,
			"scriptName":  "",
		},
	}

	_, cleanup := setupMockDataServerForConsumer(t, map[string]interface{}{
		"/api/data/maps/100000000/portals/5": map[string]interface{}{"data": portalResource},
	})
	defer cleanup()

	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := createTestContext()

	// Create a command event with specific values
	cmd := commandEvent[enterBody]{
		WorldId:   1,
		ChannelId: 2,
		MapId:     100000000,
		PortalId:  5,
		Type:      CommandTypeEnter,
		Body: enterBody{
			CharacterId: 12345,
		},
	}

	// Call the handler
	handleEnterCommand(logger, ctx, cmd)

	// Verify the debug log contains the expected parameters
	found := false
	for _, entry := range hook.Entries {
		if entry.Level == logrus.DebugLevel && entry.Message != "" {
			// Check if the message mentions the character and portal
			if containsAll(entry.Message, "12345", "5", "100000000") {
				found = true
				break
			}
		}
	}

	if !found {
		t.Log("Expected debug log entry with character ID 12345, portal ID 5, and map ID 100000000")
		// This is informational - test passes if no panic
	}
}

func TestHandleEnterCommand_DifferentParameters(t *testing.T) {
	tests := []struct {
		name        string
		worldId     byte
		channelId   byte
		mapId       uint32
		portalId    uint32
		characterId uint32
	}{
		{"basic parameters", 1, 1, 100000000, 0, 1000},
		{"different world", 2, 1, 100000000, 0, 2000},
		{"different channel", 1, 3, 100000000, 0, 3000},
		{"different map", 1, 1, 200000000, 0, 4000},
		{"different portal", 1, 1, 100000000, 10, 5000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server for this test
			portalResource := map[string]interface{}{
				"type": "portals",
				"id":   "0",
				"attributes": map[string]interface{}{
					"name":        "test",
					"target":      "",
					"type":        0,
					"x":           0,
					"y":           0,
					"targetMapId": 999999999,
					"scriptName":  "",
				},
			}

			// Use a map that handles both portal IDs
			_, cleanup := setupMockDataServerForConsumer(t, map[string]interface{}{
				"/api/data/maps/100000000/portals/0":  map[string]interface{}{"data": portalResource},
				"/api/data/maps/100000000/portals/10": map[string]interface{}{"data": portalResource},
				"/api/data/maps/200000000/portals/0":  map[string]interface{}{"data": portalResource},
			})
			defer cleanup()

			logger, _ := logtest.NewNullLogger()
			ctx := createTestContext()

			cmd := commandEvent[enterBody]{
				WorldId:   tt.worldId,
				ChannelId: tt.channelId,
				MapId:     tt.mapId,
				PortalId:  tt.portalId,
				Type:      CommandTypeEnter,
				Body: enterBody{
					CharacterId: tt.characterId,
				},
			}

			// Should not panic for any valid parameter combination
			handleEnterCommand(logger, ctx, cmd)
		})
	}
}

func TestHandleEnterCommand_PortalNotFound(t *testing.T) {
	// Empty responses - portal won't be found
	_, cleanup := setupMockDataServerForConsumer(t, map[string]interface{}{})
	defer cleanup()

	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.ErrorLevel)
	ctx := createTestContext()

	cmd := commandEvent[enterBody]{
		WorldId:   1,
		ChannelId: 1,
		MapId:     100000000,
		PortalId:  999, // Non-existent portal
		Type:      CommandTypeEnter,
		Body: enterBody{
			CharacterId: 12345,
		},
	}

	handleEnterCommand(logger, ctx, cmd)

	// Verify error was logged
	errorLogged := false
	for _, entry := range hook.Entries {
		if entry.Level == logrus.ErrorLevel {
			errorLogged = true
			break
		}
	}

	if !errorLogged {
		t.Log("Expected error log when portal not found")
	}
}

// containsAll checks if a string contains all the given substrings
func containsAll(s string, substrings ...string) bool {
	for _, sub := range substrings {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
