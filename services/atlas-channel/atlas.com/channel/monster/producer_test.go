package monster

import (
	"encoding/json"
	"testing"

	monster2 "atlas-channel/kafka/message/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

func TestDamageCommandProvider_EncodesDamagesSlice(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	provider := DamageCommandProvider(f, 12345, 67, []uint32{40, 80, 120}, 1)

	msgs, err := provider()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}

	var cmd monster2.Command[monster2.DamageCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal command: %v", err)
	}
	if cmd.Type != monster2.CommandTypeDamage {
		t.Fatalf("Type = %s, want %s", cmd.Type, monster2.CommandTypeDamage)
	}
	if cmd.MonsterId != 12345 {
		t.Fatalf("MonsterId = %d, want 12345", cmd.MonsterId)
	}
	if cmd.Body.CharacterId != 67 {
		t.Fatalf("Body.CharacterId = %d, want 67", cmd.Body.CharacterId)
	}
	if len(cmd.Body.Damages) != 3 || cmd.Body.Damages[0] != 40 || cmd.Body.Damages[1] != 80 || cmd.Body.Damages[2] != 120 {
		t.Fatalf("Body.Damages = %v, want [40 80 120]", cmd.Body.Damages)
	}
	if cmd.Body.AttackType != 1 {
		t.Fatalf("Body.AttackType = %d, want 1", cmd.Body.AttackType)
	}
}

func TestUseBasicAttackCommandProvider(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).SetInstance(uuid.Nil).Build()
	prov := UseBasicAttackCommandProvider(f, uint32(5001), uint8(1))
	msgs, err := prov()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1", len(msgs))
	}
	var cmd monster2.Command[monster2.UseBasicAttackCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != monster2.CommandTypeUseBasicAttack {
		t.Errorf("Type = %q, want %q", cmd.Type, monster2.CommandTypeUseBasicAttack)
	}
	if cmd.MonsterId != 5001 {
		t.Errorf("MonsterId = %d, want 5001", cmd.MonsterId)
	}
	if cmd.Body.AttackPos != 1 {
		t.Errorf("AttackPos = %d, want 1", cmd.Body.AttackPos)
	}
}
