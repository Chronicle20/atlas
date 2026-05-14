package monsterbook

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// TestExtract is a unit-level guard on the wire→domain transformation that
// runs without touching the HTTP layer. The httptest cases below exercise the
// full GetRequest decode path (which depends on the JSON:API stubs); this
// test pins the field-by-field mapping so a future refactor of either side
// can't silently drop a field.
func TestExtract(t *testing.T) {
	rm := CollectionRestModel{
		Id:               42,
		BookLevel:        7,
		NormalCount:      120,
		SpecialCount:     30,
		TotalUniqueCards: 150,
		CoverCardId:      2380000,
		ExpBonusPercent:  10,
	}
	c, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract returned err: %v", err)
	}
	if c.BookLevel() != 7 {
		t.Errorf("BookLevel = %d, want 7", c.BookLevel())
	}
	if c.NormalCount() != 120 {
		t.Errorf("NormalCount = %d, want 120", c.NormalCount())
	}
	if c.SpecialCount() != 30 {
		t.Errorf("SpecialCount = %d, want 30", c.SpecialCount())
	}
	if c.TotalUniqueCards() != 150 {
		t.Errorf("TotalUniqueCards = %d, want 150", c.TotalUniqueCards())
	}
	if c.CoverCardId() != 2380000 {
		t.Errorf("CoverCardId = %d, want 2380000", c.CoverCardId())
	}
	if c.ExpBonusPercent() != 10 {
		t.Errorf("ExpBonusPercent = %d, want 10", c.ExpBonusPercent())
	}
}

// TestGetTotalUniqueCards_HTTP exercises the full client path against an
// httptest server. The decode path goes through api2go/jsonapi.Unmarshal,
// which validates that CollectionRestModel implements the relationship
// stubs (SetToOneReferenceID, SetToManyReferenceIDs) — see
// libs/atlas-rest/CLAUDE.md. If those stubs were missing, the success
// case would fail with "does not implement Unmarshal*Relations".
func TestGetTotalUniqueCards_HTTP(t *testing.T) {
	tests := []struct {
		name        string
		handler     http.HandlerFunc
		wantTotal   uint16
		wantErr     bool
		wantNotFnd  bool
		wantContain string // substring of err.Error() when wantErr && !wantNotFnd
	}{
		{
			name: "success returns totalUniqueCards",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/characters/42/monster-book") {
					t.Errorf("unexpected request path: %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/vnd.api+json")
				_, _ = io.WriteString(w, `{"data":{"type":"monster-book","id":"42","attributes":{"bookLevel":3,"normalCount":4,"specialCount":3,"totalUniqueCards":7,"coverCardId":0,"expBonusPercent":0}}}`)
			},
			wantTotal: 7,
		},
		{
			name: "404 surfaces ErrNotFound so the validation context can distinguish it",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not found", http.StatusNotFound)
			},
			wantErr:    true,
			wantNotFnd: true,
		},
		{
			name: "5xx surfaces a non-NotFound error so the validation context fails closed visibly",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			},
			wantErr:     true,
			wantContain: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			// requests.RootUrl reads MONSTER_BOOK_SERVICE_URL — point it at
			// the test server. Trailing slash matters: requests.go formats
			// "<base>characters/%d/monster-book".
			t.Setenv("MONSTER_BOOK_SERVICE_URL", srv.URL+"/")

			l := logrus.New()
			l.SetOutput(io.Discard)
			p := NewProcessor(l, context.Background())

			total, err := p.GetTotalUniqueCards(42)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (total=%d)", total)
				}
				if tt.wantNotFnd && !errors.Is(err, requests.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
				if !tt.wantNotFnd && errors.Is(err, requests.ErrNotFound) {
					t.Errorf("did not expect ErrNotFound, got %v", err)
				}
				if tt.wantContain != "" && !strings.Contains(err.Error(), tt.wantContain) {
					t.Errorf("err = %q, want substring %q", err.Error(), tt.wantContain)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
		})
	}
}
