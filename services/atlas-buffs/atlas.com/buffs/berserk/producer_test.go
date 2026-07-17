package berserk

import (
	"encoding/json"
	"testing"
	"time"

	character2 "atlas-buffs/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Providers are pure message builders, so the on-the-wire JSON contract is
// asserted directly — this is the emit-side half of the golden contract test
// (atlas-channel's mirror decode is the consume-side half, Task 9).
func TestBerserkStatusEventProvider(t *testing.T) {
	txId := uuid.New()
	m := NewBuilder(world.Id(1), 42, 20).
		SetChannel(channel.Id(3)).
		SetCharacterLevel(135).
		Build().
		evaluated(true, 135, time.Time{})

	msgs, err := berserkStatusEventProvider(txId, m)()
	assert.NoError(t, err)
	assert.Len(t, msgs, 1)

	var e character2.StatusEvent[character2.BerserkStatusEventBody]
	assert.NoError(t, json.Unmarshal(msgs[0].Value, &e))
	assert.Equal(t, world.Id(1), e.WorldId)
	assert.Equal(t, uint32(42), e.CharacterId)
	assert.Equal(t, character2.EventStatusTypeBerserk, e.Type)
	assert.Equal(t, txId, e.Body.TransactionId)
	assert.Equal(t, channel.Id(3), e.Body.ChannelId)
	assert.Equal(t, uint32(skill.DarkKnightBerserkId), e.Body.SkillId)
	assert.Equal(t, byte(135), e.Body.CharacterLevel)
	assert.Equal(t, byte(20), e.Body.SkillLevel)
	assert.True(t, e.Body.Active)

	// Key must be the character id (per-character ordering on the topic).
	assert.NotEmpty(t, msgs[0].Key)
}

func TestBerserkStatusEventJSONFieldNames(t *testing.T) {
	body := character2.BerserkStatusEventBody{}
	data, err := json.Marshal(body)
	assert.NoError(t, err)
	for _, field := range []string{"transactionId", "channelId", "skillId", "characterLevel", "skillLevel", "active"} {
		assert.Contains(t, string(data), `"`+field+`"`, "JSON field names are the cross-service contract with atlas-channel")
	}
}
