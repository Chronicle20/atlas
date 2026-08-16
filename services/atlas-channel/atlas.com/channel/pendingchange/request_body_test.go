package pendingchange

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestRequestOmitsAssetId pins the seam invariant atlas-character documents on
// the receiving side (pending_change/entity.go: "AssetId is null on the
// cash-shop purchase path, which carries an entitlement reference correlated by
// TransactionId instead of an asset").
//
// These two methods are the ONLY producers of pending-change records in the
// repo, and both are reached exclusively from the cash-shop purchase handlers
// (cash_shop_operation.go handleBuyNameChange / handleBuyWorldTransfer). So
// every record they create is a purchase-path record and none of them may carry
// an assetId.
//
// Sending one is what produced the live double-award on atlas-pr-1370: a
// non-nil assetId makes atlas-character emit a destroy_asset saga for a coupon
// the player does not hold (it fails silently), and then makes the cancel path's
// Resolve branch refund a coupon that was never consumed. See
// docs/tasks/task-227-cash-name-change-world-transfer/bug-purchase-path-sets-assetid.md.
func TestRequestOmitsAssetId(t *testing.T) {
	tests := []struct {
		name string
		call func(p Processor) error
	}{
		{
			name: "name change",
			call: func(p Processor) error {
				_, err := p.RequestNameChange(1, "newname")
				return err
			},
		},
		{
			name: "world transfer",
			call: func(p Processor) error {
				_, err := p.RequestWorldTransfer(1, world.Id(2))
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotBody []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotBody, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"type":       "pending-changes",
						"id":         "1",
						"attributes": map[string]any{"characterId": 1, "status": "PENDING"},
					},
				})
			}))
			defer server.Close()

			withCharactersServiceURL(t, server.URL)

			if err := tc.call(newTestProcessor()); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			var payload struct {
				Data struct {
					Attributes map[string]any `json:"attributes"`
				} `json:"data"`
			}
			if err := json.Unmarshal(gotBody, &payload); err != nil {
				t.Fatalf("unmarshalling captured request body %q: %v", gotBody, err)
			}
			if v, present := payload.Data.Attributes["assetId"]; present {
				t.Fatalf("purchase-path request must not carry an assetId, got assetId=%v in %q", v, gotBody)
			}
		})
	}
}
