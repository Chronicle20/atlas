package character

import (
	"encoding/json"
	"testing"
)

// canonicalApplyBody is the exact JSON the APPLY command body must serialize to.
// The identical literal is asserted in the atlas-summons mirror
// (services/atlas-summons/atlas.com/summons/buff/producer_test.go) so the two
// re-declared contracts stay byte-identical on the wire.
const (
	canonicalApplyWithAccumulate    = `{"fromId":2,"sourceId":1320009,"level":25,"duration":99000,"changes":[{"type":"WEAPON_DEFENSE","amount":100}],"accumulate":true}`
	canonicalApplyWithoutAccumulate = `{"fromId":2,"sourceId":1320009,"level":25,"duration":99000,"changes":[{"type":"WEAPON_DEFENSE","amount":100}]}`
)

func sampleApplyBody(accumulate bool) ApplyCommandBody {
	return ApplyCommandBody{
		FromId:     2,
		SourceId:   1320009,
		Level:      25,
		Duration:   99000,
		Changes:    []StatChange{{Type: "WEAPON_DEFENSE", Amount: 100}},
		Accumulate: accumulate,
	}
}

func TestApplyCommandBody_AccumulateOmittedWhenFalse(t *testing.T) {
	b, err := json.Marshal(sampleApplyBody(false))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != canonicalApplyWithoutAccumulate {
		t.Fatalf("accumulate=false must omit the field.\n got: %s\nwant: %s", b, canonicalApplyWithoutAccumulate)
	}
}

func TestApplyCommandBody_AccumulatePresentWhenTrue(t *testing.T) {
	b, err := json.Marshal(sampleApplyBody(true))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != canonicalApplyWithAccumulate {
		t.Fatalf("accumulate=true must serialize the field.\n got: %s\nwant: %s", b, canonicalApplyWithAccumulate)
	}
}

func TestApplyCommandBody_RoundTrip(t *testing.T) {
	for _, acc := range []bool{false, true} {
		raw, _ := json.Marshal(sampleApplyBody(acc))
		var got ApplyCommandBody
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.Accumulate != acc {
			t.Fatalf("round-trip accumulate = %v, want %v", got.Accumulate, acc)
		}
	}
}
