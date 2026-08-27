package configuration

import (
	"reflect"
	"testing"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
)

// tradeConfigWireDocument is the exact JSON:API document atlas-tenants emits for
// GET /tenants/{id}/configurations/trade-configs. The identical literal is
// pinned on the producing side in
// services/atlas-tenants/atlas.com/tenants/configuration/trade_config_test.go
// (TestTradeConfigWireShape). The two tests together are the only thing keeping
// the nested taxTiers object array symmetric across the module boundary — the
// modules cannot import each other, so neither half alone proves the contract.
const tradeConfigWireDocument = `{"data":{"type":"trade-configs","id":"trade-configs","attributes":{"taxEnabled":true,"taxTiers":[{"threshold":100000000,"rate":0.06},{"threshold":100000,"rate":0.008}],"maxStagedItems":9,"minTradeLevel":0,"attestationTimeoutSeconds":5}}}`

// TestRestModelDecodesTheTenantsWireDocument pins that RestModel decodes the
// real serialized form through the same jsonapi.Unmarshal the outbound request
// uses (libs/atlas-rest/requests/response.go), with the taxTiers object array
// arriving intact rather than as an empty slice.
func TestRestModelDecodesTheTenantsWireDocument(t *testing.T) {
	var rm RestModel
	if err := jsonapi.Unmarshal([]byte(tradeConfigWireDocument), &rm); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}

	if rm.Id != "trade-configs" {
		t.Errorf("Id: got %q, want \"trade-configs\"", rm.Id)
	}
	if !rm.TaxEnabled {
		t.Error("TaxEnabled: got false, want true")
	}
	if rm.MaxStagedItems != 9 {
		t.Errorf("MaxStagedItems: got %d, want 9", rm.MaxStagedItems)
	}
	if rm.MinTradeLevel != 0 {
		t.Errorf("MinTradeLevel: got %d, want 0", rm.MinTradeLevel)
	}
	if rm.AttestationTimeoutSeconds != 5 {
		t.Errorf("AttestationTimeoutSeconds: got %d, want 5", rm.AttestationTimeoutSeconds)
	}

	want := []TierRestModel{
		{Threshold: 100000000, Rate: 0.060},
		{Threshold: 100000, Rate: 0.008},
	}
	if len(rm.TaxTiers) != len(want) {
		t.Fatalf("TaxTiers: got %d tiers, want %d — the nested object array did not survive the wire", len(rm.TaxTiers), len(want))
	}
	for i := range want {
		if rm.TaxTiers[i] != want[i] {
			t.Errorf("TaxTiers[%d]: got %+v, want %+v", i, rm.TaxTiers[i], want[i])
		}
	}
}

// TestWireDocumentFoldsIntoTheDomainModel pins the whole inbound path end to
// end: the tenants wire document becomes a Model whose tax table is the one the
// document carried, not the shipped default.
func TestWireDocumentFoldsIntoTheDomainModel(t *testing.T) {
	var rm RestModel
	if err := jsonapi.Unmarshal([]byte(tradeConfigWireDocument), &rm); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}

	m := Extract(rm)

	if !m.TaxEnabled() {
		t.Error("TaxEnabled: got false, want true")
	}
	if m.MaxStagedItems() != 9 {
		t.Errorf("MaxStagedItems: got %d, want 9", m.MaxStagedItems())
	}
	if m.AttestationTimeout() != 5*time.Second {
		t.Errorf("AttestationTimeout: got %s, want 5s", m.AttestationTimeout())
	}
	if len(m.TaxTiers()) != 2 {
		t.Fatalf("TaxTiers: got %d tiers, want the document's 2", len(m.TaxTiers()))
	}

	tax, delivered := Tax(m, 100_000_000)
	if tax != 6_000_000 || delivered != 94_000_000 {
		t.Errorf("Tax(100000000) under the wire table: got tax %d delivered %d, want 6000000/94000000", tax, delivered)
	}
}

// TestTransformRoundTrip pins that Extract(Transform(m)) reproduces m for a
// Model whose knobs are all non-default and non-zero, so Extract's
// zero-folding on MaxStagedItems/AttestationTimeoutSeconds never triggers.
func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		taxEnabled: false,
		taxTiers: []Tier{
			{Threshold: 50_000_000, Rate: 0.05},
			{Threshold: 10_000, Rate: 0.01},
		},
		maxStagedItems:     7,
		minTradeLevel:      15,
		attestationTimeout: 3 * time.Second,
	}

	got := Extract(Transform(m))

	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}
