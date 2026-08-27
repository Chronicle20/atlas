package effect

import (
	"atlas-messages/data/skill/effect/statup"
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	su, err := statup.Extract(statup.RestModel{Type: "buff", Amount: 5})
	if err != nil {
		t.Fatalf("statup.Extract failed: %v", err)
	}

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
		hpr:                  11.5,
		mpr:                  12.5,
		mhprRate:             13,
		mmprRate:             14,
		mobSkill:             15,
		mobSkillLevel:        16,
		mhpR:                 17,
		mmpR:                 18,
		hpCon:                19,
		mpCon:                20,
		duration:             21,
		target:               22,
		barrier:              23,
		mob:                  24,
		overtime:             true,
		repeatEffect:         true,
		moveTo:               25,
		cp:                   26,
		nuffSkill:            27,
		skill:                true,
		x:                    28,
		y:                    29,
		mobCount:             30,
		moneyCon:             31,
		cooldown:             32,
		morphId:              33,
		ghost:                34,
		fatigue:              35,
		berserk:              36,
		booster:              37,
		prop:                 38.5,
		itemCon:              39,
		itemConNo:            40,
		damage:               41,
		attackCount:          42,
		fixDamage:            43,
		bulletCount:          44,
		bulletConsume:        45,
		mapProtection:        46,
		cureAbnormalStatuses: []string{"weaken", "curse"},
		statups:              []statup.Model{su},
		monsterStatus:        map[string]uint32{"speed": 47},
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch.\nExpected %+v\nGot %+v", m, got)
	}
}
