package location

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// TestGetField_HappyPath verifies that GetField correctly decodes a JSON:API
// response from atlas-maps and returns a fully-populated field.Model.
func TestGetField_HappyPath(t *testing.T) {
	instanceUUID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/characters/1234/location") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "character-locations",
				"id": "1234",
				"attributes": {
					"worldId": 2,
					"channelId": 3,
					"mapId": 100000000,
					"instance": "` + instanceUUID.String() + `"
				}
			}
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	f, err := GetField(logrus.New(), context.Background(), 1234)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uint8(f.WorldId()) != 2 {
		t.Errorf("WorldId = %d, want 2", f.WorldId())
	}
	if uint8(f.ChannelId()) != 3 {
		t.Errorf("ChannelId = %d, want 3", f.ChannelId())
	}
	if uint32(f.MapId()) != 100000000 {
		t.Errorf("MapId = %d, want 100000000", f.MapId())
	}
	if f.Instance() != instanceUUID {
		t.Errorf("Instance = %s, want %s", f.Instance(), instanceUUID)
	}
}

// TestGetField_NotFound verifies that a 404 from atlas-maps is mapped to
// the package-level ErrNotFound sentinel so callers can distinguish
// first-login conditions from infrastructure errors.
func TestGetField_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	_, err := GetField(logrus.New(), context.Background(), 1234)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestGetField_InfrastructureError verifies that a 5xx from atlas-maps
// returns a non-nil error that is NOT ErrNotFound — callers should treat
// these as infrastructure failures.
func TestGetField_InfrastructureError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	_, err := GetField(logrus.New(), context.Background(), 1234)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("expected non-ErrNotFound, got ErrNotFound")
	}
}
