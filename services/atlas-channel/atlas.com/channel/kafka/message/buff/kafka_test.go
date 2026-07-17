package buff

import (
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// berserkEventJSON is a golden fixture of what atlas-buffs'
// berserkStatusEventProvider puts on EVENT_TOPIC_CHARACTER_BUFF_STATUS
// (see atlas-buffs berserk/producer_test.go — the emit-side twin of this
// test). If either side's struct drifts, one of the two tests breaks.
const berserkEventJSON = `{"worldId":1,"characterId":42,"type":"BERSERK","body":{"transactionId":"11111111-2222-3333-4444-555555555555","channelId":3,"skillId":1320006,"characterLevel":135,"skillLevel":20,"active":true}}`

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
