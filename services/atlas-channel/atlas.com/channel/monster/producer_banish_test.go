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
	msgs, err := BanishCommandProvider(magnetTestField(), 4242, 9500324)()
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
	if c.Body.CharacterId != 4242 {
		t.Fatalf("Body.CharacterId = %d, want 4242", c.Body.CharacterId)
	}
	if c.Body.MonsterTemplateId != 9500324 {
		t.Fatalf("Body.MonsterTemplateId = %d, want 9500324", c.Body.MonsterTemplateId)
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
	banish, err := BanishCommandProvider(magnetTestField(), 4242, 9500324)()
	if err != nil {
		t.Fatalf("banish provider returned error: %v", err)
	}
	force, err := ForceControlCommandProvider(magnetTestField(), 4242, 777)()
	if err != nil {
		t.Fatalf("force provider returned error: %v", err)
	}
	if string(banish[0].Key) != string(force[0].Key) {
		t.Fatalf("keys differ (%x vs %x); BanishCommandProvider(f, 4242, ...) must key on character id 4242, same as ForceControlCommandProvider(f, 4242, ...) keying on monster id 4242",
			banish[0].Key, force[0].Key)
	}
}
