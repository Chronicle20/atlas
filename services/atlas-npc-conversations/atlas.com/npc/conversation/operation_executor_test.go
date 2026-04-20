package conversation

import (
	"context"
	"testing"

	"atlas-npc-conversations/saga"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestEvaluateContextValueAsInt_EmbeddedNegation(t *testing.T) {
	// This test covers the bug where "-{context.cost}" was passed to arithmetic
	// evaluation without resolving the {context.cost} placeholder first.
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(1)

	// Seed the registry with a conversation context containing the "cost" key
	ctx := NewConversationContextBuilder().
		SetCharacterId(characterId).
		AddContextValue("cost", "1000").
		Build()
	GetRegistry().SetContext(tctx, characterId, ctx)
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{
		l:   l,
		ctx: tctx,
		t:   tm,
	}

	tests := []struct {
		name        string
		value       string
		expected    int
		expectError bool
	}{
		{
			name:     "direct context reference",
			value:    "{context.cost}",
			expected: 1000,
		},
		{
			name:     "negated context reference",
			value:    "-{context.cost}",
			expected: -1000,
		},
		{
			name:     "literal negative number",
			value:    "-500",
			expected: -500,
		},
		{
			name:     "literal positive number",
			value:    "200",
			expected: 200,
		},
		{
			name:        "missing context key",
			value:       "-{context.missing}",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.evaluateContextValueAsInt(characterId, "amount", tt.value)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none, result: %d", result)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestSuppressAwardAssetByCompleteQuest(t *testing.T) {
	awardAsset := func(itemId, qty uint32, show bool) builtStep {
		return builtStep{
			stepId: "award",
			status: saga.Pending,
			action: saga.AwardAsset,
			payload: saga.AwardItemActionPayload{
				CharacterId: 1,
				Item:        saga.ItemPayload{TemplateId: itemId, Quantity: qty},
				ShowEffect:  show,
			},
		}
	}
	completeQuest := func(rewards ...saga.QuestRewardItem) builtStep {
		return builtStep{
			stepId: "complete",
			status: saga.Pending,
			action: saga.CompleteQuest,
			payload: saga.CompleteQuestPayload{
				CharacterId: 1,
				QuestId:     1000,
				Rewards:     rewards,
			},
		}
	}

	tests := []struct {
		name     string
		input    []builtStep
		expected []bool // ShowEffect for each AwardAsset step in order
	}{
		{
			name:     "no complete_quest leaves AwardAsset visible",
			input:    []builtStep{awardAsset(2000000, 1, true), awardAsset(2000001, 1, true)},
			expected: []bool{true, true},
		},
		{
			name: "matching reward suppresses preceding AwardAsset",
			input: []builtStep{
				awardAsset(2000000, 1, true),
				completeQuest(saga.QuestRewardItem{ItemId: 2000000, Amount: 1}),
			},
			expected: []bool{false},
		},
		{
			name: "partial-quantity mismatch leaves AwardAsset visible",
			input: []builtStep{
				awardAsset(2000000, 5, true),
				completeQuest(saga.QuestRewardItem{ItemId: 2000000, Amount: 1}),
			},
			expected: []bool{true},
		},
		{
			name: "silent (ShowEffect=false) AwardAsset stays unchanged",
			input: []builtStep{
				awardAsset(2000000, 1, false),
				completeQuest(saga.QuestRewardItem{ItemId: 2000000, Amount: 1}),
			},
			expected: []bool{false},
		},
		{
			name: "AwardAsset after CompleteQuest is not suppressed",
			input: []builtStep{
				completeQuest(saga.QuestRewardItem{ItemId: 2000000, Amount: 1}),
				awardAsset(2000000, 1, true),
			},
			expected: []bool{true},
		},
		{
			name: "two AwardAssets sharing one reward only suppress the first",
			input: []builtStep{
				awardAsset(2000000, 1, true),
				awardAsset(2000000, 1, true),
				completeQuest(saga.QuestRewardItem{ItemId: 2000000, Amount: 1}),
			},
			expected: []bool{false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := append([]builtStep(nil), tt.input...)
			suppressAwardAssetByCompleteQuest(steps)

			idx := 0
			for _, st := range steps {
				if st.action != saga.AwardAsset {
					continue
				}
				pl := st.payload.(saga.AwardItemActionPayload)
				if pl.ShowEffect != tt.expected[idx] {
					t.Errorf("award step %d: expected ShowEffect=%v, got %v", idx, tt.expected[idx], pl.ShowEffect)
				}
				idx++
			}
		})
	}
}
