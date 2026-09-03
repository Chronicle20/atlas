package reactor_test

import (
	"atlas-saga-orchestrator/reactor"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	fieldconst "github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var emitted *producertest.Capture

// TestMain installs a capturing producer so HitReactorByName's HIT command
// emit (a real Kafka write via producer.Produce) succeeds instantly instead
// of retrying against an unreachable broker for ~42s (see producertest
// package doc).
func TestMain(m *testing.M) {
	if err := os.Setenv(string(reactor.EnvCommandTopic), string(reactor.EnvCommandTopic)); err != nil {
		panic(err)
	}
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
}

func newTestContext(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return tenant.WithContext(context.Background(), ten)
}

// TestHitReactorByNameResolvesAndProducesHit proves HitReactorByName looks
// up a reactor by name via atlas-reactors, then produces a HIT command
// carrying the resolved reactor's id and the acting character.
func TestHitReactorByNameResolvesAndProducesHit(t *testing.T) {
	emitted.Reset()

	const characterId = uint32(100100)
	const reactorId = uint32(9000101)
	const reactorName = "boss_switch"

	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"data":[{"id":"%d","type":"reactors","attributes":{"name":"%s"}}]}`, reactorId, reactorName)
	}))
	defer srv.Close()
	t.Setenv("REACTORS_SERVICE_URL", srv.URL+"/")

	ctx := newTestContext(t)
	l, _ := test.NewNullLogger()

	f := fieldconst.NewBuilder(world.Id(1), channel.Id(2), _map.Id(922000000)).Build()

	p := reactor.NewProcessor(l, ctx)
	if err := p.HitReactorByName(f, characterId, reactorName); err != nil {
		t.Fatalf("HitReactorByName returned error: %v", err)
	}

	wantPath := fmt.Sprintf("/worlds/%d/channels/%d/maps/%d/instances/%s/reactors",
		f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
	if capturedPath != wantPath {
		t.Fatalf("expected path %q, got %q", wantPath, capturedPath)
	}

	msgs := emitted.Messages(string(reactor.EnvCommandTopic))
	if len(msgs) != 1 {
		t.Fatalf("expected 1 HIT command, got %d", len(msgs))
	}
	var cmd reactor.Command[reactor.HitCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unable to unmarshal HIT command: %v", err)
	}
	if cmd.Type != reactor.CommandTypeHit {
		t.Fatalf("expected type %s, got %s", reactor.CommandTypeHit, cmd.Type)
	}
	if cmd.Body.ReactorId != reactorId {
		t.Fatalf("expected reactorId %d, got %d", reactorId, cmd.Body.ReactorId)
	}
	if cmd.Body.CharacterId != characterId {
		t.Fatalf("expected characterId %d, got %d", characterId, cmd.Body.CharacterId)
	}
}

// TestHitReactorByNameNoMatchIsAnError proves an empty atlas-reactors result
// surfaces as an error rather than silently producing a HIT for a zero id.
func TestHitReactorByNameNoMatchIsAnError(t *testing.T) {
	emitted.Reset()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	t.Setenv("REACTORS_SERVICE_URL", srv.URL+"/")

	ctx := newTestContext(t)
	l, _ := test.NewNullLogger()

	f := fieldconst.NewBuilder(world.Id(1), channel.Id(2), _map.Id(922000000)).Build()

	p := reactor.NewProcessor(l, ctx)
	if err := p.HitReactorByName(f, 100100, "missing_switch"); err == nil {
		t.Fatal("expected an error when no reactor matches the name, got nil")
	}

	if msgs := emitted.Messages(string(reactor.EnvCommandTopic)); len(msgs) != 0 {
		t.Fatalf("expected no HIT command to be produced, got %d", len(msgs))
	}
}

// TestResetReactorsCarriesMinState proves ResetReactors posts to the
// field-scoped reset resource, carrying an optional minState -- nil resets
// every reactor, a pointer resets only reactors at or above that state
// (task-290 G5).
func TestResetReactorsCarriesMinState(t *testing.T) {
	tests := []struct {
		name         string
		minState     *int8
		wantHasField bool
	}{
		{name: "reset all (nil minState)", minState: nil, wantHasField: false},
		{name: "reset from state 0", minState: int8Ptr(0), wantHasField: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath string
			var capturedBody map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				var doc struct {
					Data struct {
						Attributes map[string]any `json:"attributes"`
					} `json:"data"`
				}
				if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
					t.Fatal(err)
				}
				capturedBody = doc.Data.Attributes
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()
			t.Setenv("REACTORS_SERVICE_URL", srv.URL+"/")

			ctx := newTestContext(t)
			l, _ := test.NewNullLogger()
			f := fieldconst.NewBuilder(world.Id(1), channel.Id(2), _map.Id(922000000)).Build()

			p := reactor.NewProcessor(l, ctx)
			if err := p.ResetReactors(f, tt.minState); err != nil {
				t.Fatalf("ResetReactors returned error: %v", err)
			}

			wantPath := fmt.Sprintf("/worlds/%d/channels/%d/maps/%d/instances/%s/reactors/reset",
				f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
			if capturedPath != wantPath {
				t.Fatalf("expected path %q, got %q", wantPath, capturedPath)
			}

			_, hasField := capturedBody["minState"]
			if hasField != tt.wantHasField {
				t.Fatalf("expected minState field present=%v, got %v (body=%v)", tt.wantHasField, hasField, capturedBody)
			}
		})
	}
}

// TestShuffleReactorsIssuesPost proves ShuffleReactors posts to the
// field-scoped shuffle resource (task-290 G5).
func TestShuffleReactorsIssuesPost(t *testing.T) {
	var capturedPath string
	var capturedMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	t.Setenv("REACTORS_SERVICE_URL", srv.URL+"/")

	ctx := newTestContext(t)
	l, _ := test.NewNullLogger()
	f := fieldconst.NewBuilder(world.Id(1), channel.Id(2), _map.Id(922000000)).Build()

	p := reactor.NewProcessor(l, ctx)
	if err := p.ShuffleReactors(f); err != nil {
		t.Fatalf("ShuffleReactors returned error: %v", err)
	}

	if capturedMethod != http.MethodPost {
		t.Fatalf("expected method POST, got %q", capturedMethod)
	}
	wantPath := fmt.Sprintf("/worlds/%d/channels/%d/maps/%d/instances/%s/reactors/shuffle",
		f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
	if capturedPath != wantPath {
		t.Fatalf("expected path %q, got %q", wantPath, capturedPath)
	}
}

func int8Ptr(v int8) *int8 { return &v }
