package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestMonsterMovement(t *testing.T) {
	p := MovementRequest{}
	p.uniqueId = 1001
	p.moveId = 55
	p.dwFlag = 1
	p.nActionAndDir = -3
	p.skillData = 0x0305 // skillId=5, skillLevel=3
	p.moveFlags = 0
	p.hackedCode = 0
	p.flyCtxTargetX = 100
	p.flyCtxTargetY = 200
	p.hackedCodeCRC = 999
	p.bChasing = 1
	p.hasTarget = 0
	p.bChasing2 = 1
	p.bChasingHack = 0
	p.tChaseDuration = 500

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, p.Encode, p.Decode, nil)

			if p.UniqueId() != 1001 {
				t.Errorf("expected uniqueId 1001, got %d", p.UniqueId())
			}
			if p.MoveId() != 55 {
				t.Errorf("expected moveId 55, got %d", p.MoveId())
			}
			if p.DwFlag() != 1 {
				t.Errorf("expected dwFlag 1, got %d", p.DwFlag())
			}
			if !p.MonsterMoveStartResult() {
				t.Error("expected monsterMoveStartResult true")
			}
			if p.ActionAndDir() != -3 {
				t.Errorf("expected nActionAndDir -3, got %d", p.ActionAndDir())
			}
			if p.SkillId() != 5 {
				t.Errorf("expected skillId 5, got %d", p.SkillId())
			}
			if p.SkillLevel() != 3 {
				t.Errorf("expected skillLevel 3, got %d", p.SkillLevel())
			}
		})
	}
}

func TestMonsterMovementGMS28(t *testing.T) {
	// GMS v28 does not have multiTargetForBall, randTimeForAreaAttack, hackedCodeCRC, or chasing fields.
	p := MovementRequest{}
	p.uniqueId = 2002
	p.moveId = 10
	p.dwFlag = 0
	p.nActionAndDir = 1
	p.skillData = 0
	p.moveFlags = 0
	p.hackedCode = 0
	p.flyCtxTargetX = 0
	p.flyCtxTargetY = 0

	ctx := test.CreateContext("GMS", 28, 1)
	test.RoundTrip(t, ctx, p.Encode, p.Decode, nil)

	if p.UniqueId() != 2002 {
		t.Errorf("expected uniqueId 2002, got %d", p.UniqueId())
	}
	if p.MonsterMoveStartResult() {
		t.Error("expected monsterMoveStartResult false")
	}
}

func TestMonsterMovementOperationString(t *testing.T) {
	p := MovementRequest{}
	if p.Operation() != MonsterMovementHandle {
		t.Errorf("expected operation %s, got %s", MonsterMovementHandle, p.Operation())
	}
	if p.String() == "" {
		t.Error("expected non-empty string")
	}
}
