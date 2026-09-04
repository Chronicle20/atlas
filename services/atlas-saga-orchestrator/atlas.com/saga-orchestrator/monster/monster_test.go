package monster_test

import (
	"atlas-saga-orchestrator/monster"
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
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// spawnInputDoc is the JSON:API request document shape produced by
// requests.PostRequest for a SpawnInputRestModel: {"data":{"type":...,
// "id":..., "attributes":{...}}}. Decoding only the fields under test keeps
// this test independent of SpawnInputRestModel's own JSON:API relationship
// contract, which is out of scope for this fix.
type spawnInputDoc struct {
	Data struct {
		Attributes struct {
			MonsterId     uint32 `json:"monsterId"`
			SpawnIfAbsent bool   `json:"spawnIfAbsent"`
		} `json:"attributes"`
	} `json:"data"`
}

// spawnResponseDoc renders a JSON:API "monsters" single-resource response
// carrying the fixed uniqueId 42, mirroring the wire shape atlas-monsters
// returns from POST .../monsters.
func spawnResponseDoc() string {
	return `{"data":{"id":"42","type":"monsters","attributes":{"uniqueId":42,"worldId":1,"channelId":2,"mapId":100000000,"monsterId":9300000,"controlCharacterId":0,"x":100,"y":200,"fh":1,"stance":0,"team":-1,"maxHp":1000,"hp":1000,"maxMp":0,"mp":0}}}`
}

// TestSpawnMonsterCarriesSpawnIfAbsent proves requestSpawnMonster (invoked via
// Processor.SpawnMonster) puts the SpawnIfAbsent wire field on the outbound
// JSON:API request body in both the true and false cases, rather than
// silently dropping it as a zero value.
func TestSpawnMonsterCarriesSpawnIfAbsent(t *testing.T) {
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
			t.Setenv("MONSTERS_SERVICE_URL", srv.URL+"/")

			ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
			if err != nil {
				t.Fatal(err)
			}
			ctx := tenant.WithContext(context.Background(), ten)
			l, _ := test.NewNullLogger()

			f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).Build()
			req := monster.SpawnRequest{
				WorldId:       world.Id(1),
				ChannelId:     channel.Id(2),
				MapId:         _map.Id(100000000),
				MonsterId:     9300000,
				X:             100,
				Y:             200,
				Fh:            1,
				Team:          -1,
				SpawnIfAbsent: tt.spawnIfAbsent,
			}

			p := monster.NewProcessor(l, ctx)
			if err := p.SpawnMonster(f, req); err != nil {
				t.Fatalf("SpawnMonster returned error: %v", err)
			}

			wantPath := fmt.Sprintf("/worlds/%d/channels/%d/maps/%d/instances/%s/monsters",
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
			if got.Data.Attributes.MonsterId != req.MonsterId {
				t.Fatalf("expected MonsterId=%d on the wire, got %d", req.MonsterId, got.Data.Attributes.MonsterId)
			}
		})
	}
}
