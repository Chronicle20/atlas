package instance

import (
	"atlas-party-quests/condition"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareValues(t *testing.T) {
	tests := []struct {
		name     string
		actual   uint32
		operator string
		expected uint32
		result   bool
	}{
		{"gte_true", 10, ">=", 10, true},
		{"gte_greater", 15, ">=", 10, true},
		{"gte_false", 5, ">=", 10, false},
		{"lte_true", 10, "<=", 10, true},
		{"lte_less", 5, "<=", 10, true},
		{"lte_false", 15, "<=", 10, false},
		{"eq_true", 10, "=", 10, true},
		{"eq_false", 5, "=", 10, false},
		{"gt_true", 15, ">", 10, true},
		{"gt_false", 10, ">", 10, false},
		{"lt_true", 5, "<", 10, true},
		{"lt_false", 10, "<", 10, false},
		{"unknown_operator", 10, "!=", 10, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.result, compareValues(tc.actual, tc.operator, tc.expected))
		})
	}
}

func testStageState(itemCounts map[uint32]uint32, monsterKills map[uint32]uint32) StageState {
	ss := NewStageState()
	for k, v := range itemCounts {
		ss = ss.WithItemCount(k, v)
	}
	for k, v := range monsterKills {
		ss = ss.WithMonsterKill(k, v)
	}
	return ss
}

func TestEvaluateCondition(t *testing.T) {
	itemCondition, _ := condition.NewBuilder().
		SetType("item").
		SetOperator(">=").
		SetValue(5).
		SetReferenceId(1001).
		Build()

	monsterCondition, _ := condition.NewBuilder().
		SetType("monster_kill").
		SetOperator(">=").
		SetValue(10).
		SetReferenceId(2001).
		Build()

	unknownCondition, _ := condition.NewBuilder().
		SetType("unknown").
		SetOperator(">=").
		SetValue(1).
		Build()

	tests := []struct {
		name   string
		cond   condition.Model
		state  StageState
		result bool
	}{
		{
			"item_met",
			itemCondition,
			testStageState(map[uint32]uint32{1001: 5}, nil),
			true,
		},
		{
			"item_not_met",
			itemCondition,
			testStageState(map[uint32]uint32{1001: 3}, nil),
			false,
		},
		{
			"item_missing",
			itemCondition,
			NewStageState(),
			false,
		},
		{
			"monster_met",
			monsterCondition,
			testStageState(nil, map[uint32]uint32{2001: 10}),
			true,
		},
		{
			"monster_not_met",
			monsterCondition,
			testStageState(nil, map[uint32]uint32{2001: 7}),
			false,
		},
		{
			"unknown_type_passes",
			unknownCondition,
			NewStageState(),
			true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.result, evaluateCondition(tc.cond, tc.state))
		})
	}
}

func TestEvaluateClearConditions(t *testing.T) {
	t.Run("empty_conditions_pass", func(t *testing.T) {
		assert.True(t, evaluateClearConditions(nil, NewStageState()))
		assert.True(t, evaluateClearConditions([]condition.Model{}, NewStageState()))
	})

	t.Run("all_conditions_met", func(t *testing.T) {
		c1, _ := condition.NewBuilder().SetType("item").SetOperator(">=").SetValue(5).SetReferenceId(1001).Build()
		c2, _ := condition.NewBuilder().SetType("monster_kill").SetOperator(">=").SetValue(3).SetReferenceId(2001).Build()

		ss := testStageState(map[uint32]uint32{1001: 10}, map[uint32]uint32{2001: 5})
		assert.True(t, evaluateClearConditions([]condition.Model{c1, c2}, ss))
	})

	t.Run("one_condition_not_met", func(t *testing.T) {
		c1, _ := condition.NewBuilder().SetType("item").SetOperator(">=").SetValue(5).SetReferenceId(1001).Build()
		c2, _ := condition.NewBuilder().SetType("monster_kill").SetOperator(">=").SetValue(10).SetReferenceId(2001).Build()

		ss := testStageState(map[uint32]uint32{1001: 10}, map[uint32]uint32{2001: 5})
		assert.False(t, evaluateClearConditions([]condition.Model{c1, c2}, ss))
	})
}

func TestGenerateCombination(t *testing.T) {
	t.Run("default_properties", func(t *testing.T) {
		combo := generateCombination(map[string]any{})
		assert.Len(t, combo, 3)
		for _, v := range combo {
			assert.Less(t, v, uint32(3))
		}
	})

	t.Run("custom_positions_and_digits", func(t *testing.T) {
		combo := generateCombination(map[string]any{
			"digits":    float64(5),
			"positions": float64(4),
		})
		assert.Len(t, combo, 4)
		for _, v := range combo {
			assert.Less(t, v, uint32(5))
		}
	})
}
