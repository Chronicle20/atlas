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
			rewards: []RewardModel{{itemId: 1132010, count: 2, prob: 100}},
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
			itemId: 1132010,
			count:  2,
			prob:   100,
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
