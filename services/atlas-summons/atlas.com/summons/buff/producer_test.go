package buff

import (
	"encoding/json"
	"testing"
)

// These canonical JSON literals MUST match the atlas-buffs consumer mirror
// (services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka_test.go).
// Keeping both tests asserting the same exact strings guarantees the two
// re-declared ApplyCommandBody contracts stay byte-identical on the wire — a
// drift in either field name, order, or omitempty tag breaks one of the two.
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
