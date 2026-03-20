package model

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestDamageTakenInfoPhysicalRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := DamageTakenInfo{
				characterId:       100,
				updateTime:        12345,
				nAttackIdx:        DamageTypePhysical,
				nMagicElemAttr:    DamageElementTypeFire,
				damage:            500,
				monsterTemplateId: 200100,
				monsterId:         42,
				left:              true,
				nX:                3,
				bGuard:            true,
				relativeDir:       1,
				bPowerGuard:       false,
				monsterId2:        43,
				powerGuard:        true,
				hitX:              100,
				hitY:              200,
				characterX:        110,
				characterY:        210,
				expression:        5,
			}
			output := DamageTakenInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.AttackIdx() != input.AttackIdx() {
				t.Errorf("nAttackIdx: got %v, want %v", output.AttackIdx(), input.AttackIdx())
			}
			if output.MagicElemAttr() != input.MagicElemAttr() {
				t.Errorf("nMagicElemAttr: got %v, want %v", output.MagicElemAttr(), input.MagicElemAttr())
			}
			if output.Damage() != input.Damage() {
				t.Errorf("damage: got %v, want %v", output.Damage(), input.Damage())
			}
			if output.MonsterTemplateId() != input.MonsterTemplateId() {
				t.Errorf("monsterTemplateId: got %v, want %v", output.MonsterTemplateId(), input.MonsterTemplateId())
			}
			if output.MonsterId() != input.MonsterId() {
				t.Errorf("monsterId: got %v, want %v", output.MonsterId(), input.MonsterId())
			}
			if output.Left() != input.Left() {
				t.Errorf("left: got %v, want %v", output.Left(), input.Left())
			}
			if output.NX() != input.NX() {
				t.Errorf("nX: got %v, want %v", output.NX(), input.NX())
			}
			if v.Region == "GMS" && v.MajorVersion >= 95 {
				if output.Guard() != input.Guard() {
					t.Errorf("bGuard: got %v, want %v", output.Guard(), input.Guard())
				}
			}
			if output.RelativeDir() != input.RelativeDir() {
				t.Errorf("relativeDir: got %v, want %v", output.RelativeDir(), input.RelativeDir())
			}
			if output.PowerGuard() != input.PowerGuard() {
				t.Errorf("bPowerGuard: got %v, want %v", output.PowerGuard(), input.PowerGuard())
			}
			if output.MonsterId2() != input.MonsterId2() {
				t.Errorf("monsterId2: got %v, want %v", output.MonsterId2(), input.MonsterId2())
			}
			if output.PowerGuard2() != input.PowerGuard2() {
				t.Errorf("powerGuard: got %v, want %v", output.PowerGuard2(), input.PowerGuard2())
			}
			if output.HitX() != input.HitX() {
				t.Errorf("hitX: got %v, want %v", output.HitX(), input.HitX())
			}
			if output.HitY() != input.HitY() {
				t.Errorf("hitY: got %v, want %v", output.HitY(), input.HitY())
			}
			if output.CharacterX() != input.CharacterX() {
				t.Errorf("characterX: got %v, want %v", output.CharacterX(), input.CharacterX())
			}
			if output.CharacterY() != input.CharacterY() {
				t.Errorf("characterY: got %v, want %v", output.CharacterY(), input.CharacterY())
			}
			if output.Expression() != input.Expression() {
				t.Errorf("expression: got %v, want %v", output.Expression(), input.Expression())
			}
		})
	}
}

func TestDamageTakenInfoObstacleRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := DamageTakenInfo{
				characterId:    100,
				updateTime:     54321,
				nAttackIdx:     DamageTypeObstacle,
				nMagicElemAttr: DamageElementTypeNone,
				damage:         250,
				obstacleData:   10,
				expression:     2,
			}
			output := DamageTakenInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.AttackIdx() != input.AttackIdx() {
				t.Errorf("nAttackIdx: got %v, want %v", output.AttackIdx(), input.AttackIdx())
			}
			if output.Damage() != input.Damage() {
				t.Errorf("damage: got %v, want %v", output.Damage(), input.Damage())
			}
			if output.ObstacleData() != input.ObstacleData() {
				t.Errorf("obstacleData: got %v, want %v", output.ObstacleData(), input.ObstacleData())
			}
			if output.Expression() != input.Expression() {
				t.Errorf("expression: got %v, want %v", output.Expression(), input.Expression())
			}
		})
	}
}
