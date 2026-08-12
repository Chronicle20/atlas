package character

import (
	"encoding/json"
	"testing"
	"time"
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

// canonicalUpdateStatValueBody is the exact JSON the UPDATE_STAT_VALUE command
// body must serialize to. The identical literal is asserted in the
// atlas-channel mirror (services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka_test.go)
// so the two re-declared contracts stay byte-identical on the wire.
const canonicalUpdateStatValueBody = `{"sourceId":1111002,"statType":"COMBO","operation":"INCREMENT","amount":2,"cap":6}`

func TestUpdateStatValueCommandBody_CanonicalJSON(t *testing.T) {
	b, err := json.Marshal(UpdateStatValueCommandBody{
		SourceId:  1111002,
		StatType:  "COMBO",
		Operation: StatOperationIncrement,
		Amount:    2,
		Cap:       6,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != canonicalUpdateStatValueBody {
		t.Fatalf("canonical mismatch.\n got: %s\nwant: %s", b, canonicalUpdateStatValueBody)
	}
}

func TestStatUpdatedStatusEventBody_RoundTrip(t *testing.T) {
	created := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	expires := created.Add(150 * time.Second)
	in := StatUpdatedStatusEventBody{
		SourceId:  1111002,
		Level:     20,
		Duration:  150000,
		Changes:   []StatChange{{Type: "COMBO", Amount: 3}},
		CreatedAt: created,
		ExpiresAt: expires,
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got StatUpdatedStatusEventBody
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.SourceId != in.SourceId || got.Level != in.Level || got.Duration != in.Duration ||
		len(got.Changes) != 1 || got.Changes[0] != in.Changes[0] ||
		!got.CreatedAt.Equal(in.CreatedAt) || !got.ExpiresAt.Equal(in.ExpiresAt) {
		t.Fatalf("round-trip mismatch: %+v vs %+v", got, in)
	}
}

// canonicalPeriodicEffectBody is the exact JSON the PERIODIC_EFFECT status
// event body must serialize to. The identical literal is asserted in the
// atlas-channel mirror (services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka_test.go)
// so the two re-declared contracts stay byte-identical on the wire. atlas-buffs
// owns this contract; atlas-channel only consumes it.
const canonicalPeriodicEffectBody = `{"channelId":3,"skillId":1311008,"statType":"DRAGON_BLOOD"}`

func TestPeriodicEffectStatusEventBody_CanonicalJSON(t *testing.T) {
	b, err := json.Marshal(PeriodicEffectStatusEventBody{
		ChannelId: 3,
		SkillId:   1311008,
		StatType:  "DRAGON_BLOOD",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != canonicalPeriodicEffectBody {
		t.Fatalf("canonical mismatch.\n got: %s\nwant: %s", b, canonicalPeriodicEffectBody)
	}
}
