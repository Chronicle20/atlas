package effect

import (
	"atlas-channel/data/skill/effect/statup"
	"reflect"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field Extract reads from RestModel (including the nested
// LT/RB points and stat-up slice) survives a Transform -> Extract round
// trip. RestModel.CardStats is not read by Extract (no Model field carries
// it) and is not asserted here.
func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		weaponAttack:         1,
		magicAttack:          2,
		weaponDefense:        3,
		magicDefense:         4,
		accuracy:             5,
		avoidability:         6,
		speed:                7,
		jump:                 8,
		hp:                   9,
		mp:                   10,
		hpr:                  1.5,
		mpr:                  2.5,
		mhprRate:             11,
		mmprRate:             12,
		mobSkill:             13,
		mobSkillLevel:        14,
		mhpR:                 15,
		mmpR:                 16,
		hpCon:                17,
		mpCon:                18,
		duration:             19,
		target:               20,
		barrier:              21,
		mob:                  22,
		overtime:             true,
		repeatEffect:         true,
		moveTo:               23,
		cp:                   24,
		nuffSkill:            25,
		skill:                true,
		x:                    26,
		y:                    27,
		mobCount:             28,
		rangeValue:           29,
		moneyCon:             30,
		cooldown:             31,
		morphId:              32,
		ghost:                33,
		fatigue:              34,
		berserk:              35,
		booster:              36,
		prop:                 3.5,
		itemCon:              37,
		itemConNo:            38,
		damage:               39,
		attackCount:          40,
		fixDamage:            41,
		dot:                  42,
		dotInterval:          43,
		dotTime:              44,
		lt:                   point.NewModel(point.X(45), point.Y(46)),
		rb:                   point.NewModel(point.X(47), point.Y(48)),
		bulletCount:          49,
		bulletConsume:        50,
		mapProtection:        51,
		cureAbnormalStatuses: []string{"seal", "darkness"},
		statups: []statup.Model{
			statup.NewModel("STR", 10),
			statup.NewModel("DEX", 5),
		},
		monsterStatus: map[string]uint32{"weakness": 1},
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	m2, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}
