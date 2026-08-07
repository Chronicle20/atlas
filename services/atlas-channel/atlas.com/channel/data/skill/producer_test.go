package skill

import (
	"atlas-channel/kafka/message/skill"
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

func TestResetCooldownsCommandProvider_EncodesEnvelopeAndBody(t *testing.T) {
	transactionId := uuid.New()
	worldId := world.Id(1)
	characterId := uint32(100)

	msgs, err := ResetCooldownsCommandProvider(transactionId, worldId, characterId, []uint32{5121010}, 5121010)()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("provider returned %d messages, want 1", len(msgs))
	}
	if !bytes.Equal(msgs[0].Key, producer.CreateKey(int(characterId))) {
		t.Fatalf("message key = %v, want CreateKey(%d)", msgs[0].Key, characterId)
	}

	var c skill.Command[skill.ResetCooldownsBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("failed to decode command: %v", err)
	}
	if c.Type != skill.CommandTypeResetCooldowns {
		t.Errorf("type = %s, want RESET_COOLDOWNS", c.Type)
	}
	if c.TransactionId != transactionId {
		t.Errorf("transactionId = %s, want %s", c.TransactionId, transactionId)
	}
	if c.WorldId != worldId {
		t.Errorf("worldId = %d, want %d", c.WorldId, worldId)
	}
	if c.CharacterId != characterId {
		t.Errorf("characterId = %d, want %d", c.CharacterId, characterId)
	}
	if len(c.Body.ExceptSkillIds) != 1 || c.Body.ExceptSkillIds[0] != 5121010 {
		t.Errorf("exceptSkillIds = %v, want [5121010]", c.Body.ExceptSkillIds)
	}
	if c.Body.SourceSkillId != 5121010 {
		t.Errorf("sourceSkillId = %d, want 5121010", c.Body.SourceSkillId)
	}
}
