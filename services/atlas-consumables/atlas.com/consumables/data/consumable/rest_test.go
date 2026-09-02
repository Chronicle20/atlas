package consumable

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	t.Run("Transform", func(t *testing.T) {
		m := Model{
			id:              2022309,
			tradeBlock:      true,
			price:           150,
			unitPrice:       3,
			slotMax:         100,
			timeLimited:     true,
			notSale:         true,
			reqLevel:        10,
			quest:           true,
			only:            true,
			consumeOnPickup: true,
			success:         70,
			cursed:          5,
			create:          2000000,
			masterLevel:     3,
			reqSkillLevel:   4,
			tradeAvailable:  1,
			noCancelMouse:   true,
			pquest:          true,
			left:            -1,
			right:           1,
			top:             -2,
			bottom:          2,
			bridleMsgType:   1,
			bridleProp:      55,
			bridlePropChg:   1.5,
			useDelay:        1000,
			delayMsg:        "delay-msg",
			incFatigue:      -3,
			npc:             9000,
			script:          "some_script",
			runOnPickup:     true,
			monsterBook:     true,
			monsterId:       100100,
			bigSize:         true,
			tragetBlock:     true,
			effect:          "some_effect",
			monsterHp:       80,
			worldMsg:        "/name caught /monster",
			incPDD:          1,
			incMDD:          2,
			incACC:          3,
			incMHP:          4,
			incMMP:          5,
			incPAD:          6,
			incMAD:          7,
			incEVA:          8,
			incLUK:          9,
			incDEX:          10,
			incINT:          11,
			incSTR:          12,
			incSpeed:        13,
			incJump:         14,
			spec:            map[SpecType]int32{SpecTypeHP: 100, SpecTypeMP: 50},
			monsterSummons: []SummonModel{
				{templateId: 100100, probability: 30},
				{templateId: 100200, probability: 70},
			},
			morphs:  map[uint32]uint32{1: 10, 2: 20},
			skills:  []uint32{1000, 2000},
			rewards: []RewardModel{{itemId: 1, count: 1, prob: 10, effect: "e1", worldMsg: "w1", period: -1}},
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
			t.Fatalf("round trip mismatch:\n got  = %+v\n want = %+v", got, m)
		}
	})

	t.Run("Reward", func(t *testing.T) {
		rw := RewardModel{
			itemId:   1132010,
			count:    2,
			prob:     100,
			effect:   "Effect/BasicEff/Event1/Good",
			worldMsg: "/name got /item",
			period:   7200,
		}

		rrm, err := TransformReward(rw)
		if err != nil {
			t.Fatalf("TransformReward failed: %v", err)
		}

		got, err := ExtractReward(rrm)
		if err != nil {
			t.Fatalf("ExtractReward failed: %v", err)
		}

		if !reflect.DeepEqual(got, rw) {
			t.Fatalf("round trip mismatch:\n got  = %+v\n want = %+v", got, rw)
		}
	})
}

func TestExtractRewardFields(t *testing.T) {
	rm := RewardRestModel{ItemId: 1132010, Count: 1, Prob: 100, Effect: "Effect/BasicEff/Event1/Good", WorldMsg: "/name got /item", Period: 7200}
	got, err := ExtractReward(rm)
	if err != nil {
		t.Fatal(err)
	}
	if got.ItemId() != 1132010 || got.Count() != 1 || got.Prob() != 100 {
		t.Fatalf("base = {%d,%d,%d}", got.ItemId(), got.Count(), got.Prob())
	}
	if got.Effect() != "Effect/BasicEff/Event1/Good" {
		t.Errorf("Effect() = %q", got.Effect())
	}
	if got.WorldMsg() != "/name got /item" {
		t.Errorf("WorldMsg() = %q", got.WorldMsg())
	}
	if got.Period() != 7200 {
		t.Errorf("Period() = %d", got.Period())
	}
}

func TestExtractMaxLevelAndExperienceSpec(t *testing.T) {
	t.Run("populated", func(t *testing.T) {
		rm := RestModel{MaxLevel: 200, Spec: map[SpecType]int32{SpecTypeExperience: 3000}}
		m, err := Extract(rm)
		if err != nil {
			t.Fatal(err)
		}
		if m.MaxLevel() != 200 {
			t.Errorf("MaxLevel() = %d, want 200", m.MaxLevel())
		}
		val, ok := m.GetSpec(SpecTypeExperience)
		if !ok || val != 3000 {
			t.Errorf("GetSpec(SpecTypeExperience) = (%d, %t), want (3000, true)", val, ok)
		}
	})

	t.Run("zero value", func(t *testing.T) {
		rm := RestModel{}
		m, err := Extract(rm)
		if err != nil {
			t.Fatal(err)
		}
		if m.MaxLevel() != 0 {
			t.Errorf("MaxLevel() = %d, want 0", m.MaxLevel())
		}
		if _, ok := m.GetSpec(SpecTypeExperience); ok {
			t.Errorf("GetSpec(SpecTypeExperience) ok = true, want false")
		}
	})
}

func TestExtractPropagatesRewardsToModel(t *testing.T) {
	rm := RestModel{Id: 2022309, Rewards: []RewardRestModel{{ItemId: 1, Count: 1, Prob: 10, Period: -1}}}
	m, err := Extract(rm)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Rewards()) != 1 {
		t.Fatalf("len(m.Rewards()) = %d, want 1", len(m.Rewards()))
	}
	if m.Rewards()[0].Prob() != 10 || m.Rewards()[0].Period() != -1 {
		t.Errorf("reward = prob %d period %d", m.Rewards()[0].Prob(), m.Rewards()[0].Period())
	}
}
