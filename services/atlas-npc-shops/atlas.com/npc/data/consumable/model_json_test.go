package consumable

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelJSONRoundTrip(t *testing.T) {
	original := Model{
		id:              2070000,
		tradeBlock:      true,
		price:           500,
		unitPrice:       3.5,
		slotMax:         200,
		timeLimited:     true,
		notSale:         false,
		reqLevel:        10,
		quest:           true,
		only:            true,
		consumeOnPickup: true,
		success:         100,
		cursed:          5,
		create:          42,
		masterLevel:     3,
		reqSkillLevel:   1,
		tradeAvailable:  true,
		noCancelMouse:   true,
		pquest:          true,
		left:            -100,
		right:           100,
		top:             -50,
		bottom:          50,
		bridleMsgType:   2,
		bridleProp:      10,
		bridlePropChg:   0.5,
		useDelay:        1000,
		delayMsg:        "wait",
		incFatigue:      -5,
		npc:             9010000,
		script:          "test_script",
		runOnPickup:     true,
		monsterBook:     true,
		monsterId:       100100,
		bigSize:         true,
		tragetBlock:     true,
		effect:          "heal",
		monsterHp:       5000,
		worldMsg:        "hello",
		incPDD:          10,
		incMDD:          20,
		incACC:          30,
		incMHP:          100,
		incMMP:          200,
		incPAD:          15,
		incMAD:          25,
		incEVA:          5,
		incLUK:          3,
		incDEX:          4,
		incINT:          5,
		incSTR:          6,
		incSpeed:        8,
		incJump:         7,
		spec: map[SpecType]int32{
			SpecTypeHP:   100,
			SpecTypeMP:   50,
			SpecTypeTime: 300,
		},
		monsterSummons: []SummonModel{
			{templateId: 100100, probability: 80},
			{templateId: 200200, probability: 20},
		},
		morphs: map[uint32]uint32{
			1: 100,
			2: 200,
		},
		skills: []uint32{1001, 2001, 3001},
		rewards: []RewardModel{
			{itemId: 4000000, count: 10, prob: 50},
			{itemId: 4000001, count: 5, prob: 30},
		},
	}

	data, err := json.Marshal(original)
	assert.NoError(t, err)

	var restored Model
	err = json.Unmarshal(data, &restored)
	assert.NoError(t, err)

	assert.Equal(t, original.id, restored.id)
	assert.Equal(t, original.tradeBlock, restored.tradeBlock)
	assert.Equal(t, original.price, restored.price)
	assert.Equal(t, original.unitPrice, restored.unitPrice)
	assert.Equal(t, original.slotMax, restored.slotMax)
	assert.Equal(t, original.timeLimited, restored.timeLimited)
	assert.Equal(t, original.notSale, restored.notSale)
	assert.Equal(t, original.reqLevel, restored.reqLevel)
	assert.Equal(t, original.quest, restored.quest)
	assert.Equal(t, original.only, restored.only)
	assert.Equal(t, original.consumeOnPickup, restored.consumeOnPickup)
	assert.Equal(t, original.success, restored.success)
	assert.Equal(t, original.cursed, restored.cursed)
	assert.Equal(t, original.create, restored.create)
	assert.Equal(t, original.masterLevel, restored.masterLevel)
	assert.Equal(t, original.reqSkillLevel, restored.reqSkillLevel)
	assert.Equal(t, original.tradeAvailable, restored.tradeAvailable)
	assert.Equal(t, original.noCancelMouse, restored.noCancelMouse)
	assert.Equal(t, original.pquest, restored.pquest)
	assert.Equal(t, original.left, restored.left)
	assert.Equal(t, original.right, restored.right)
	assert.Equal(t, original.top, restored.top)
	assert.Equal(t, original.bottom, restored.bottom)
	assert.Equal(t, original.bridleMsgType, restored.bridleMsgType)
	assert.Equal(t, original.bridleProp, restored.bridleProp)
	assert.Equal(t, original.bridlePropChg, restored.bridlePropChg)
	assert.Equal(t, original.useDelay, restored.useDelay)
	assert.Equal(t, original.delayMsg, restored.delayMsg)
	assert.Equal(t, original.incFatigue, restored.incFatigue)
	assert.Equal(t, original.npc, restored.npc)
	assert.Equal(t, original.script, restored.script)
	assert.Equal(t, original.runOnPickup, restored.runOnPickup)
	assert.Equal(t, original.monsterBook, restored.monsterBook)
	assert.Equal(t, original.monsterId, restored.monsterId)
	assert.Equal(t, original.bigSize, restored.bigSize)
	assert.Equal(t, original.tragetBlock, restored.tragetBlock)
	assert.Equal(t, original.effect, restored.effect)
	assert.Equal(t, original.monsterHp, restored.monsterHp)
	assert.Equal(t, original.worldMsg, restored.worldMsg)
	assert.Equal(t, original.incPDD, restored.incPDD)
	assert.Equal(t, original.incMDD, restored.incMDD)
	assert.Equal(t, original.incACC, restored.incACC)
	assert.Equal(t, original.incMHP, restored.incMHP)
	assert.Equal(t, original.incMMP, restored.incMMP)
	assert.Equal(t, original.incPAD, restored.incPAD)
	assert.Equal(t, original.incMAD, restored.incMAD)
	assert.Equal(t, original.incEVA, restored.incEVA)
	assert.Equal(t, original.incLUK, restored.incLUK)
	assert.Equal(t, original.incDEX, restored.incDEX)
	assert.Equal(t, original.incINT, restored.incINT)
	assert.Equal(t, original.incSTR, restored.incSTR)
	assert.Equal(t, original.incSpeed, restored.incSpeed)
	assert.Equal(t, original.incJump, restored.incJump)
	assert.Equal(t, original.spec, restored.spec)
	assert.Len(t, restored.monsterSummons, 2)
	assert.Equal(t, original.monsterSummons[0].templateId, restored.monsterSummons[0].templateId)
	assert.Equal(t, original.monsterSummons[0].probability, restored.monsterSummons[0].probability)
	assert.Equal(t, original.monsterSummons[1].templateId, restored.monsterSummons[1].templateId)
	assert.Equal(t, original.monsterSummons[1].probability, restored.monsterSummons[1].probability)
	assert.Equal(t, original.morphs, restored.morphs)
	assert.Equal(t, original.skills, restored.skills)
	assert.Len(t, restored.rewards, 2)
	assert.Equal(t, original.rewards[0].itemId, restored.rewards[0].itemId)
	assert.Equal(t, original.rewards[0].count, restored.rewards[0].count)
	assert.Equal(t, original.rewards[0].prob, restored.rewards[0].prob)
}

func TestModelJSONEmpty(t *testing.T) {
	original := Model{}

	data, err := json.Marshal(original)
	assert.NoError(t, err)

	var restored Model
	err = json.Unmarshal(data, &restored)
	assert.NoError(t, err)

	assert.Equal(t, uint32(0), restored.id)
}

func TestModelJSONSlice(t *testing.T) {
	models := []Model{
		{id: 1, price: 100, slotMax: 50},
		{id: 2, price: 200, slotMax: 100},
	}

	data, err := json.Marshal(models)
	assert.NoError(t, err)

	var restored []Model
	err = json.Unmarshal(data, &restored)
	assert.NoError(t, err)

	assert.Len(t, restored, 2)
	assert.Equal(t, uint32(1), restored[0].id)
	assert.Equal(t, uint32(100), restored[0].price)
	assert.Equal(t, uint32(2), restored[1].id)
	assert.Equal(t, uint32(200), restored[1].price)
}
