package monster

import (
	monster2 "atlas-channel/kafka/message/monster"
	"encoding/json"
	"testing"
)

// TestBanishCommandProviderShape pins the BANISH envelope: MonsterId is 0
// (the client supplies a template id, not a unique monster id) and the body
// carries the character id and the client-supplied template id.
func TestBanishCommandProviderShape(t *testing.T) {
	tests := []struct {
		name              string
		characterId       uint32
		monsterTemplateId uint32
	}{
		{
			name:              "client-supplied template id",
			characterId:       4242,
			monsterTemplateId: 9500324,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgs, err := BanishCommandProvider(magnetTestField(), tc.characterId, tc.monsterTemplateId)()
			if err != nil {
				t.Fatalf("provider returned error: %v", err)
			}
			if len(msgs) != 1 {
				t.Fatalf("provider produced %d messages, want 1", len(msgs))
			}

			var c monster2.Command[monster2.BanishCommandBody]
			if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if c.Type != monster2.CommandTypeBanish {
				t.Fatalf("Type = %q, want %q", c.Type, monster2.CommandTypeBanish)
			}
			if c.MonsterId != 0 {
				t.Fatalf("MonsterId = %d, want 0 — the client supplies a template id, not a unique id", c.MonsterId)
			}
			if c.Body.CharacterId != tc.characterId {
				t.Fatalf("Body.CharacterId = %d, want %d", c.Body.CharacterId, tc.characterId)
			}
			if c.Body.MonsterTemplateId != tc.monsterTemplateId {
				t.Fatalf("Body.MonsterTemplateId = %d, want %d", c.Body.MonsterTemplateId, tc.monsterTemplateId)
			}
		})
	}
}

// TestBanishCommandKeysOnCharacter pins the ordering contract: unlike every
// other provider in this file, BANISH keys on the character id rather than
// the monster id, so a character's banish requests stay ordered against each
// other. ForceControlCommandProvider(f, 4242, 777) keys on monster id 4242
// and BanishCommandProvider(f, 4242, 9500324) keys on character id 4242, so
// identical keys prove BanishCommandProvider keyed on its first uint32
// argument (the character id) rather than on the template id 9500324 — NOT
// that both providers key on the monster id.
func TestBanishCommandKeysOnCharacter(t *testing.T) {
	tests := []struct {
		name              string
		characterId       uint32
		monsterTemplateId uint32
		monsterId         uint32
	}{
		{
			name:              "banish keys on character id, same as force-control keys on monster id",
			characterId:       4242,
			monsterTemplateId: 9500324,
			monsterId:         777,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			banish, err := BanishCommandProvider(magnetTestField(), tc.characterId, tc.monsterTemplateId)()
			if err != nil {
				t.Fatalf("banish provider returned error: %v", err)
			}
			force, err := ForceControlCommandProvider(magnetTestField(), tc.characterId, tc.monsterId)()
			if err != nil {
				t.Fatalf("force provider returned error: %v", err)
			}
			if string(banish[0].Key) != string(force[0].Key) {
				t.Fatalf("keys differ (%x vs %x); BanishCommandProvider(f, %d, ...) must key on character id %d, same as ForceControlCommandProvider(f, %d, ...) keying on monster id %d",
					banish[0].Key, force[0].Key, tc.characterId, tc.characterId, tc.characterId, tc.characterId)
			}
		})
	}
}
