package _map_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	_map "atlas-monsters/map"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// characterLocationDoc renders a JSON:API single-resource "character-locations"
// document, mirroring atlas-maps' GET /characters/{characterId}/location
// response shape (map/rest.go's LocationRestModel).
func characterLocationDoc(id string, worldId world.Id, channelId channel.Id, mapId mapconst.Id, instance uuid.UUID) string {
	return fmt.Sprintf(
		`{"data":{"id":"%s","type":"character-locations","attributes":{"worldId":%d,"channelId":%d,"mapId":%d,"instance":"%s"}}}`,
		id, worldId, channelId, mapId, instance.String(),
	)
}

// TestGetCharacterField_HTTPRoundTrip exercises GetCharacterField's real
// unmarshal path (requests.Provider -> JSON:API decode -> ExtractLocation),
// not an injected seam (RelinquishControlOnHide/RestoreCandidacyOnReveal's
// tests inject locationFn directly). This proves the LocationRestModel
// struct tags and ExtractLocation actually decode a live atlas-maps
// response into the expected field.Model.
func TestGetCharacterField_HTTPRoundTrip(t *testing.T) {
	wantWorld := world.Id(1)
	wantChannel := channel.Id(2)
	wantMap := mapconst.Id(100000000)
	wantInstance := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters/42/location" {
			http.Error(w, "unexpected path: "+r.URL.Path, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(characterLocationDoc("42", wantWorld, wantChannel, wantMap, wantInstance)))
	}))
	defer srv.Close()
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	f, err := _map.NewProcessor(l, ctx).GetCharacterField(42)
	if err != nil {
		t.Fatal(err)
	}
	if f.WorldId() != wantWorld {
		t.Fatalf("WorldId() = %d, want %d (worldId attribute decode failed)", f.WorldId(), wantWorld)
	}
	if f.ChannelId() != wantChannel {
		t.Fatalf("ChannelId() = %d, want %d (channelId attribute decode failed)", f.ChannelId(), wantChannel)
	}
	if f.MapId() != wantMap {
		t.Fatalf("MapId() = %d, want %d (mapId attribute decode failed)", f.MapId(), wantMap)
	}
	if f.Instance() != wantInstance {
		t.Fatalf("Instance() = %s, want %s (instance attribute decode failed)", f.Instance(), wantInstance)
	}
}

// TestGetCharacterField_HTTPRoundTrip_NotFound proves a real upstream 404
// maps to requests.ErrNotFound through GetCharacterField -- the same
// sentinel monster/processor.go's RelinquishControlOnHide/
// RestoreCandidacyOnReveal branch on to distinguish "offline/absent"
// (Debugf) from a transient error (Warnf).
func TestGetCharacterField_HTTPRoundTrip_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	if _, err := _map.NewProcessor(l, ctx).GetCharacterField(404); !errors.Is(err, requests.ErrNotFound) {
		t.Fatalf("err = %v, want errors.Is(_, requests.ErrNotFound)", err)
	}
}
