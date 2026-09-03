package npc_spawn_test

import (
	"atlas-saga-orchestrator/npc_spawn"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	fieldconst "github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// spawnInputDoc is the JSON:API request document shape produced by
// requests.PostRequest for a SpawnInputRestModel.
type spawnInputDoc struct {
	Data struct {
		Attributes struct {
			NpcId         uint32 `json:"npcId"`
			SpawnIfAbsent bool   `json:"spawnIfAbsent"`
		} `json:"attributes"`
	} `json:"data"`
}

// spawnResponseDoc renders a JSON:API "npcs" single-resource response
// mirroring the wire shape atlas-maps returns from POST .../npcs.
func spawnResponseDoc() string {
	return `{"data":{"id":"42","type":"npcs","attributes":{"uniqueId":42,"worldId":1,"channelId":2,"mapId":100000000,"npcId":9010000,"x":100,"y":200,"fh":1}}}`
}

// TestSpawnNpcCarriesSpawnIfAbsent proves requestSpawnNpc (invoked via
// Processor.SpawnNpc) puts the SpawnIfAbsent wire field on the outbound
// JSON:API request body in both the true and false cases.
func TestSpawnNpcCarriesSpawnIfAbsent(t *testing.T) {
	tests := []struct {
		name          string
		spawnIfAbsent bool
	}{
		{name: "spawn if absent true", spawnIfAbsent: true},
		{name: "spawn if absent false", spawnIfAbsent: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody []byte
			var capturedPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				b, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				capturedBody = b
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(spawnResponseDoc()))
			}))
			defer srv.Close()
			t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")

			ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
			if err != nil {
				t.Fatal(err)
			}
			ctx := tenant.WithContext(context.Background(), ten)
			l, _ := test.NewNullLogger()

			f := fieldconst.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).Build()
			req := npc_spawn.SpawnRequest{
				WorldId:       world.Id(1),
				ChannelId:     channel.Id(2),
				MapId:         _map.Id(100000000),
				NpcId:         9010000,
				X:             100,
				Y:             200,
				Fh:            1,
				SpawnIfAbsent: tt.spawnIfAbsent,
			}

			p := npc_spawn.NewProcessor(l, ctx)
			if err := p.SpawnNpc(f, req); err != nil {
				t.Fatalf("SpawnNpc returned error: %v", err)
			}

			wantPath := fmt.Sprintf("/worlds/%d/channels/%d/maps/%d/instances/%s/npcs",
				f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
			if capturedPath != wantPath {
				t.Fatalf("expected path %q, got %q", wantPath, capturedPath)
			}

			var got spawnInputDoc
			if err := json.Unmarshal(capturedBody, &got); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}
			if got.Data.Attributes.SpawnIfAbsent != tt.spawnIfAbsent {
				t.Fatalf("expected SpawnIfAbsent=%v on the wire, got %v", tt.spawnIfAbsent, got.Data.Attributes.SpawnIfAbsent)
			}
			if got.Data.Attributes.NpcId != req.NpcId {
				t.Fatalf("expected NpcId=%d on the wire, got %d", req.NpcId, got.Data.Attributes.NpcId)
			}
		})
	}
}

// TestSpawnNpcPropagatesUpstreamFailure proves a non-2xx/201 response from
// atlas-maps surfaces as an error rather than being swallowed.
func TestSpawnNpcPropagatesUpstreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	f := fieldconst.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).Build()
	req := npc_spawn.SpawnRequest{
		WorldId:   world.Id(1),
		ChannelId: channel.Id(2),
		MapId:     _map.Id(100000000),
		NpcId:     9010000,
	}

	p := npc_spawn.NewProcessor(l, ctx)
	if err := p.SpawnNpc(f, req); err == nil {
		t.Fatal("expected an error from a failing upstream response, got nil")
	}
}
