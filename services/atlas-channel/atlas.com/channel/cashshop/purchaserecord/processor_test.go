package purchaserecord

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// purchaseRecordResponse is a JSON:API purchaseRecords response, mirroring
// what atlas-cashshop's purchaserecord resource marshals (resource type
// "purchaseRecords", id = serial number).
const purchaseRecordResponse = `{
  "data": { "type": "purchaseRecords", "id": "5555", "attributes": { "purchased": true, "count": 2 } }
}`

// TestGetForAccount_DecodesRecord stands up an httptest server emulating
// atlas-cashshop's purchase-record lookup and asserts the processor decodes
// the purchased/count fields and hits the expected account/serial path.
func TestGetForAccount_DecodesRecord(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(purchaseRecordResponse))
	}))
	defer srv.Close()

	t.Setenv("CASHSHOP_SERVICE_URL", srv.URL+"/")

	m, err := NewProcessor(logrus.New(), context.Background()).GetForAccount(9000, 5555)
	if err != nil {
		t.Fatalf("GetForAccount returned error: %v", err)
	}

	if gotPath != "/accounts/9000/purchaseRecords/5555" {
		t.Errorf("request path: want /accounts/9000/purchaseRecords/5555, got %q", gotPath)
	}
	if m.SerialNumber() != 5555 {
		t.Errorf("SerialNumber: got %d, want 5555", m.SerialNumber())
	}
	if !m.Purchased() {
		t.Error("Purchased: got false, want true")
	}
	if m.Count() != 2 {
		t.Errorf("Count: got %d, want 2", m.Count())
	}
}
