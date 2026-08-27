package summon

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestTransform(t *testing.T) {
	instance := uuid.New()
	expiresAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(3)).SetInstance(instance).Build()
	m := NewBuilder().
		SetId(50).
		SetOwnerCharacterId(100).
		SetSkillId(200).
		SetSkillLevel(3).
		SetSummonType(SummonTypeAttacker).
		SetMovementType(MovementFollow).
		SetField(f).
		SetX(10).
		SetY(20).
		SetHp(30).
		SetMaxHp(40).
		SetExpiresAt(expiresAt).
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.Id != "50" {
		t.Errorf("Id mismatch. Expected 50, got %v", rm.Id)
	}

	if rm.OwnerCharacterId != 100 {
		t.Errorf("OwnerCharacterId mismatch. Expected 100, got %v", rm.OwnerCharacterId)
	}

	if rm.SkillId != 200 {
		t.Errorf("SkillId mismatch. Expected 200, got %v", rm.SkillId)
	}

	if rm.SkillLevel != 3 {
		t.Errorf("SkillLevel mismatch. Expected 3, got %v", rm.SkillLevel)
	}

	if rm.SummonType != string(SummonTypeAttacker) {
		t.Errorf("SummonType mismatch. Expected %v, got %v", string(SummonTypeAttacker), rm.SummonType)
	}

	if rm.MovementType != byte(MovementFollow) {
		t.Errorf("MovementType mismatch. Expected %v, got %v", byte(MovementFollow), rm.MovementType)
	}

	if rm.X != 10 {
		t.Errorf("X mismatch. Expected 10, got %v", rm.X)
	}

	if rm.Y != 20 {
		t.Errorf("Y mismatch. Expected 20, got %v", rm.Y)
	}

	if rm.Hp != 30 {
		t.Errorf("Hp mismatch. Expected 30, got %v", rm.Hp)
	}

	if rm.MaxHp != 40 {
		t.Errorf("MaxHp mismatch. Expected 40, got %v", rm.MaxHp)
	}

	if rm.ExpiresAt != expiresAt.UnixMilli() {
		t.Errorf("ExpiresAt mismatch. Expected %v, got %v", expiresAt.UnixMilli(), rm.ExpiresAt)
	}

	if rm.WorldId != world.Id(1) {
		t.Errorf("WorldId mismatch. Expected %v, got %v", world.Id(1), rm.WorldId)
	}

	if rm.ChannelId != channel.Id(2) {
		t.Errorf("ChannelId mismatch. Expected %v, got %v", channel.Id(2), rm.ChannelId)
	}

	if rm.MapId != _map.Id(3) {
		t.Errorf("MapId mismatch. Expected %v, got %v", _map.Id(3), rm.MapId)
	}

	if rm.Instance != instance {
		t.Errorf("Instance mismatch. Expected %v, got %v", instance, rm.Instance)
	}
}
