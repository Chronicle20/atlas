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
	"github.com/stretchr/testify/require"
)

// setupRecordingDataServer creates an httptest server that mocks the DATA
// service, same as setupMockDataServer, except it also records every
// requested fullPath so tests can assert on which data-service paths were
// asked for (WarpByName's Kafka publish is otherwise unobservable in tests).
//
// The random-spawn drain (portal/requests.go: inMapUrl + DrainProvider)
// appends page[number]/page[size] query params that a literal response key
// can't predict, so responses are matched against the full path with query
// first and, failing that, against the bare path (query stripped) — letting
// callers register the paginated drain endpoint by its bare path.
func setupRecordingDataServer(t *testing.T, responses map[string]interface{}) (*[]string, func()) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fullPath := r.URL.Path
		if r.URL.RawQuery != "" {
			fullPath = r.URL.Path + "?" + r.URL.RawQuery
		}
		paths = append(paths, fullPath)
		t.Logf("Mock server received request: %s %s", r.Method, fullPath)

		response, ok := responses[fullPath]
		if !ok {
			response, ok = responses[r.URL.Path]
		}
		if ok {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
			return
		}

		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]string{{"detail": "not found"}},
		}); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))

	originalURL := os.Getenv("DATA_SERVICE_URL")
	require.NoError(t, os.Setenv("DATA_SERVICE_URL", server.URL+"/api/"))

	cleanup := func() {
		server.Close()
		if originalURL != "" {
			require.NoError(t, os.Setenv("DATA_SERVICE_URL", originalURL))
		} else {
			require.NoError(t, os.Unsetenv("DATA_SERVICE_URL"))
		}
	}

	return &paths, cleanup
}

func pathsContain(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}

// pathsContainPrefix reports whether any recorded path starts with prefix.
// The random-spawn drain (portal/requests.go: inMapUrl + DrainProvider)
// appends page[number]/page[size] query params, so the bare map-portals
// path is never requested exactly — only as a prefix.
func pathsContainPrefix(paths []string, prefix string) bool {
	for _, p := range paths {
		if len(p) >= len(prefix) && p[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func TestWarpByName_Hit(t *testing.T) {
	targetPortal := createPortalResource("7", "st00", "", 999999999, "")

	paths, cleanup := setupRecordingDataServer(t, map[string]interface{}{
		"/api/data/maps/200000000/portals?name=st00": jsonAPIResponse{Data: []jsonAPIResource{targetPortal}},
	})
	defer cleanup()

	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := test.CreateTestContext()

	f := createTestField(100000000)
	portal.NewProcessor(logger, ctx).WarpByName(f, 12345, 200000000, "st00")

	if !pathsContain(*paths, "/api/data/maps/200000000/portals?name=st00") {
		t.Errorf("expected recorded paths to contain the name lookup, got %v", *paths)
	}
	if pathsContainPrefix(*paths, "/api/data/maps/200000000/portals?page") {
		t.Errorf("expected no random-spawn drain when name resolves, got %v", *paths)
	}

	// A resolvable name must not trigger the "unable to locate portal"
	// fallback warning. (Unrelated WARN noise from the Kafka producer, which
	// has no broker configured in this test, is expected and ignored here.)
	for _, entry := range hook.Entries {
		if entry.Level == logrus.WarnLevel && containsAllForTest(entry.Message, "Unable to locate portal") {
			t.Errorf("expected no portal-resolution warning, got %q", entry.Message)
		}
	}
}

func TestWarpByName_MissFallsBackToRandomSpawn(t *testing.T) {
	fallbackPortal := createPortalResource("0", "sp", "", 999999999, "")

	paths, cleanup := setupRecordingDataServer(t, map[string]interface{}{
		"/api/data/maps/200000000/portals?name=nope": jsonAPIResponse{Data: []jsonAPIResource{}},
		"/api/data/maps/200000000/portals":           jsonAPIResponse{Data: []jsonAPIResource{fallbackPortal}},
	})
	defer cleanup()

	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := test.CreateTestContext()

	f := createTestField(100000000)
	portal.NewProcessor(logger, ctx).WarpByName(f, 12345, 200000000, "nope")

	// Only the "unable to locate portal" fallback warning is asserted here;
	// unrelated WARN noise from the Kafka producer (no broker configured in
	// this test) is expected and ignored.
	warnCount := 0
	for _, entry := range hook.Entries {
		if entry.Level == logrus.WarnLevel && containsAllForTest(entry.Message, "Unable to locate portal") {
			warnCount++
			if !containsAllForTest(entry.Message, "nope", "200000000", "12345") {
				t.Errorf("expected warning to mention nope, 200000000, and 12345, got %q", entry.Message)
			}
		}
	}
	if warnCount != 1 {
		t.Errorf("expected exactly one portal-resolution warning, got %d", warnCount)
	}

	if !pathsContain(*paths, "/api/data/maps/200000000/portals?name=nope") {
		t.Errorf("expected recorded paths to contain the name lookup, got %v", *paths)
	}
	if !pathsContainPrefix(*paths, "/api/data/maps/200000000/portals?page") {
		t.Errorf("expected recorded paths to contain the random-spawn drain, got %v", *paths)
	}
}

// containsAllForTest mirrors the unexported containsAll helper in
// consumer_test.go (package portal), which this external test package cannot
// call directly.
func containsAllForTest(s string, substrings ...string) bool {
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
