package pending_change

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mtsMux stands up an httptest server serving the two atlas-mts endpoints
// gate 11 reads, with the response body for each controlled independently.
// It mirrors the real JSON:API shape (a top-level "data" array) so the test
// drives the same api2go unmarshal path the live client uses.
func mtsMux(t *testing.T, holdingsBody, listingsBody string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/characters/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/mts/holding"):
			_, _ = w.Write([]byte(holdingsBody))
		case strings.HasSuffix(r.URL.Path, "/mts/listings"):
			_, _ = w.Write([]byte(listingsBody))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	return httptest.NewServer(mux)
}

const emptyDataDoc = `{"data":[]}`

const oneHoldingDoc = `{"data":[{"type":"holdings","id":"11111111-1111-1111-1111-111111111111","attributes":{}}]}`

const oneListingDoc = `{"data":[{"type":"listings","id":"22222222-2222-2222-2222-222222222222","attributes":{}}]}`

// TestMtsGate_ActiveListingNoHoldingsBlocks is THE case fix-round-1 found: a
// character with a live, un-settled auction and NO take-home holdings must
// still trip gate 11. Before this fix, mtsHoldingOpen only queried the
// holdings endpoint, so this case passed (character reported eligible) — a
// listing becomes a holding only on cancel/expiry, never while active.
func TestMtsGate_ActiveListingNoHoldingsBlocks(t *testing.T) {
	srv := mtsMux(t, emptyDataDoc, oneListingDoc)
	defer srv.Close()
	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	open, err := mtsHoldingOpen(testLogger(t), context.Background(), 300300)
	if err != nil {
		t.Fatalf("mtsHoldingOpen: %v", err)
	}
	if !open {
		t.Fatal("expected an active listing with no holdings to report open=true (gate 11 must reject)")
	}
}

// TestMtsGate_HoldingNoActiveListingsBlocks proves the pre-existing behavior
// (a holding alone) still blocks — the fix must not regress it.
func TestMtsGate_HoldingNoActiveListingsBlocks(t *testing.T) {
	srv := mtsMux(t, oneHoldingDoc, emptyDataDoc)
	defer srv.Close()
	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	open, err := mtsHoldingOpen(testLogger(t), context.Background(), 300301)
	if err != nil {
		t.Fatalf("mtsHoldingOpen: %v", err)
	}
	if !open {
		t.Fatal("expected a holding with no active listings to report open=true")
	}
}

// TestMtsGate_NeitherPasses proves a character with no holdings and no active
// listings is not blocked by gate 11.
func TestMtsGate_NeitherPasses(t *testing.T) {
	srv := mtsMux(t, emptyDataDoc, emptyDataDoc)
	defer srv.Close()
	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	open, err := mtsHoldingOpen(testLogger(t), context.Background(), 300302)
	if err != nil {
		t.Fatalf("mtsHoldingOpen: %v", err)
	}
	if open {
		t.Fatal("expected neither holdings nor active listings to report open=false")
	}
}

// familyTreeDoc builds a JSON:API "data" array of the given number of
// family-tree-members resources, matching the shape getFamilyTreeHandler
// (services/atlas-families/atlas.com/family/family/resource.go) returns.
func familyTreeDoc(ids ...string) string {
	doc := `{"data":[`
	for i, id := range ids {
		if i > 0 {
			doc += ","
		}
		doc += `{"type":"family-tree-members","id":"` + id + `","attributes":{}}`
	}
	doc += `]}`
	return doc
}

// TestInFamily covers the two bugs fixed in inFamily: a non-404 error must
// never be reported as an affirmative family membership, and a tree
// containing only the requesting character is not a family.
func TestInFamily(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantIn     bool
		wantErr    bool
	}{
		{
			name:       "404 -> no family",
			statusCode: http.StatusNotFound,
			wantIn:     false,
			wantErr:    false,
		},
		{
			name:       "200 with self-only tree -> no family",
			statusCode: http.StatusOK,
			body:       familyTreeDoc("500500"),
			wantIn:     false,
			wantErr:    false,
		},
		{
			name:       "200 with a relative -> in family",
			statusCode: http.StatusOK,
			body:       familyTreeDoc("500500", "500501"),
			wantIn:     true,
			wantErr:    false,
		},
		{
			name:       "500 (transport/decode failure) -> error, not in family",
			statusCode: http.StatusInternalServerError,
			body:       "",
			wantIn:     false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/families/tree/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(tt.statusCode)
				if tt.body != "" {
					_, _ = w.Write([]byte(tt.body))
				}
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()
			t.Setenv("FAMILIES_SERVICE_URL", srv.URL+"/")

			in, err := inFamily(testLogger(t), context.Background(), 500500)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error to propagate, got nil")
				}
			} else if err != nil {
				t.Fatalf("inFamily: unexpected error: %v", err)
			}
			if in != tt.wantIn {
				t.Fatalf("inFamily = %v, want %v", in, tt.wantIn)
			}
		})
	}
}
