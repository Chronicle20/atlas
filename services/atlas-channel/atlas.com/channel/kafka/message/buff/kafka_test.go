package buff

import (
	"encoding/json"
	"testing"
	"time"
)

// canonicalUpdateStatValueBody is the exact JSON the UPDATE_STAT_VALUE command
// body must serialize to. The identical literal is asserted in the atlas-buffs
// owner contract (services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka_test.go)
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
