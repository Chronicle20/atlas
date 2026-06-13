package mount

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestGetByCharacterId_UnmarshalsMount stands up a JSON:API mount endpoint and
// verifies the channel client resolves and Extracts level/exp/tiredness. Guards
// the RestModel type name ("mounts") and field tags against atlas-mounts drift.
func TestGetByCharacterId_UnmarshalsMount(t *testing.T) {
	const characterId = 7
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != fmt.Sprintf("/api/characters/%d/mount", characterId) {
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"mounts","id":"3b0b-uuid","attributes":{"characterId":7,"level":3,"exp":1234,"tiredness":42}}}`))
	}))
	defer srv.Close()
	t.Setenv("MOUNTS_SERVICE_URL", srv.URL+"/api/")

	m, err := NewProcessor(logrus.New(), context.Background()).GetByCharacterId(characterId)
	if err != nil {
		t.Fatalf("GetByCharacterId: %v", err)
	}
	if m.CharacterId() != characterId || m.Level() != 3 || m.Exp() != 1234 || m.Tiredness() != 42 {
		t.Fatalf("extracted mount wrong: %+v", m)
	}
}

// TestGetByCharacterId_PropagatesError verifies a 5xx surfaces as an error (so the
// char-info handler logs it rather than masking it as "no mount").
func TestGetByCharacterId_PropagatesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("MOUNTS_SERVICE_URL", srv.URL+"/api/")

	if _, err := NewProcessor(logrus.New(), context.Background()).GetByCharacterId(1); err == nil {
		t.Fatal("expected error on 5xx, got nil")
	}
}
