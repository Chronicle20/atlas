package buff

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// berserkEventJSON is a golden fixture of what atlas-buffs'
// berserkStatusEventProvider puts on EVENT_TOPIC_CHARACTER_BUFF_STATUS
// (see atlas-buffs berserk/producer_test.go — the emit-side twin of this
// test). If either side's struct drifts, one of the two tests breaks.
const berserkEventJSON = `{"worldId":1,"characterId":42,"type":"BERSERK","body":{"transactionId":"11111111-2222-3333-4444-555555555555","channelId":3,"skillId":1320006,"characterLevel":135,"skillLevel":20,"active":true}}`

// canonicalUpdateStatValueBody is the exact JSON the UPDATE_STAT_VALUE command
// body must serialize to. The identical literal is asserted in the atlas-buffs
// owner contract (services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka_test.go)
// so the two re-declared contracts stay byte-identical on the wire.
const canonicalUpdateStatValueBody = `{"sourceId":1111002,"statType":"COMBO","operation":"INCREMENT","amount":2,"cap":6}`

func TestBerserkStatusEventDecode(t *testing.T) {
	var e StatusEvent[BerserkStatusEventBody]
	assert.NoError(t, json.Unmarshal([]byte(berserkEventJSON), &e))

	assert.Equal(t, world.Id(1), e.WorldId)
	assert.Equal(t, uint32(42), e.CharacterId)
	assert.Equal(t, EventStatusTypeBerserk, e.Type)
	assert.Equal(t, uuid.MustParse("11111111-2222-3333-4444-555555555555"), e.Body.TransactionId)
	assert.Equal(t, channel.Id(3), e.Body.ChannelId)
	assert.Equal(t, uint32(skill.DarkKnightBerserkId), e.Body.SkillId)
	assert.Equal(t, byte(135), e.Body.CharacterLevel)
	assert.Equal(t, byte(20), e.Body.SkillLevel)
	assert.True(t, e.Body.Active)
}

func TestBerserkStatusEventDecodeInactive(t *testing.T) {
	inactive := `{"worldId":0,"characterId":7,"type":"BERSERK","body":{"transactionId":"11111111-2222-3333-4444-555555555555","channelId":1,"skillId":1320006,"characterLevel":200,"skillLevel":30,"active":false}}`
	var e StatusEvent[BerserkStatusEventBody]
	assert.NoError(t, json.Unmarshal([]byte(inactive), &e))
	assert.False(t, e.Body.Active, "inactive ticks clear the aura — they are broadcast too")
}

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

// canonicalPeriodicEffectBody is the exact JSON atlas-buffs emits for a
// PERIODIC_EFFECT status event body (task-214). The identical literal is
// asserted in the owner contract
// (services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka_test.go);
// the two live in separate Go modules, so a field or tag renamed on one side
// and not the other fails no build — it decodes into a zero-valued body at
// runtime and the pulse silently stops.
const canonicalPeriodicEffectBody = `{"channelId":3,"skillId":1311008,"statType":"DRAGON_BLOOD"}`

func TestPeriodicEffectStatusEventBody_CanonicalJSON(t *testing.T) {
	b, err := json.Marshal(PeriodicEffectStatusEventBody{
		ChannelId: 3,
		SkillId:   1311008,
		StatType:  "DRAGON_BLOOD",
	})
	assert.NoError(t, err)
	assert.Equal(t, canonicalPeriodicEffectBody, string(b))
}

func TestPeriodicEffectStatusEventDecode(t *testing.T) {
	const periodicEventJSON = `{"worldId":1,"characterId":42,"type":"PERIODIC_EFFECT","body":{"channelId":3,"skillId":1311008,"statType":"DRAGON_BLOOD"}}`
	var e StatusEvent[PeriodicEffectStatusEventBody]
	assert.NoError(t, json.Unmarshal([]byte(periodicEventJSON), &e))

	assert.Equal(t, world.Id(1), e.WorldId)
	assert.Equal(t, uint32(42), e.CharacterId)
	assert.Equal(t, EventStatusTypePeriodicEffect, e.Type)
	assert.Equal(t, channel.Id(3), e.Body.ChannelId)
	assert.Equal(t, uint32(1311008), e.Body.SkillId)
	assert.Equal(t, "DRAGON_BLOOD", e.Body.StatType)
}
