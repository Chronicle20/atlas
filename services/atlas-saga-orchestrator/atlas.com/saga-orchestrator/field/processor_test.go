package field_test

import (
	"atlas-saga-orchestrator/field"
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

// resetFieldDoc is the JSON:API request document shape produced by
// requests.PostRequest for a ResetFieldInputRestModel.
type resetFieldDoc struct {
	Data struct {
		Attributes struct {
			Difficulty int `json:"difficulty"`
		} `json:"attributes"`
	} `json:"data"`
}

// TestResetFieldIssuesPostWithDifficulty proves Processor.ResetField posts
// to the field-scoped reset resource path (world, channel, map, instance)
// atlas-maps exposes, carrying the requested difficulty -- Cosmic's
// MapleMap.resetPQ(difficulty) (task-290 G5).
func TestResetFieldIssuesPostWithDifficulty(t *testing.T) {
	tests := []struct {
		name       string
		difficulty int
	}{
		{name: "difficulty zero", difficulty: 0},
		{name: "difficulty non-zero", difficulty: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath string
			var capturedMethod string
			var capturedBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				capturedMethod = r.Method
				b, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				capturedBody = b
				w.WriteHeader(http.StatusNoContent)
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

			p := field.NewProcessor(l, ctx)
			if err := p.ResetField(f, tt.difficulty); err != nil {
				t.Fatalf("ResetField returned error: %v", err)
			}

			if capturedMethod != http.MethodPost {
				t.Fatalf("expected method POST, got %q", capturedMethod)
			}
			wantPath := fmt.Sprintf("/worlds/%d/channels/%d/maps/%d/instances/%s/reset",
				f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
			if capturedPath != wantPath {
				t.Fatalf("expected path %q, got %q", wantPath, capturedPath)
			}

			var got resetFieldDoc
			if err := json.Unmarshal(capturedBody, &got); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}
			if got.Data.Attributes.Difficulty != tt.difficulty {
				t.Fatalf("expected difficulty=%d on the wire, got %d", tt.difficulty, got.Data.Attributes.Difficulty)
			}
		})
	}
}

// TestResetFieldPropagatesUpstreamFailure proves a non-2xx response from
// atlas-maps surfaces as an error rather than being swallowed.
func TestResetFieldPropagatesUpstreamFailure(t *testing.T) {
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

	p := field.NewProcessor(l, ctx)
	if err := p.ResetField(f, 0); err == nil {
		t.Fatal("expected an error from a failing upstream response, got nil")
	}
}
