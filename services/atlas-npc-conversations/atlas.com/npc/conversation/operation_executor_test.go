package conversation

import (
	"context"
	"testing"

	"atlas-npc-conversations/saga"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
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

// TestCreateSagaForOperations_DeduplicatesStepIds guards against a regression
// where a quest-completion conversation that batches multiple ops of the same
// type (e.g., two `award_item` ops alongside `complete_quest`) produced saga
// steps with colliding stepIds, which the orchestrator rejects with
// "duplicate step ID". Observed in atlas-saga-orchestrator logs as
// transaction_id=70419e40-… on 2026-04-21 with `award_item-11` appearing twice.
func TestCreateSagaForOperations_DeduplicatesStepIds(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)

	characterId := uint32(11)

	// Seed a conversation context so evaluateContextValue can resolve literals.
	convCtx := NewConversationContextBuilder().
		SetCharacterId(characterId).
		Build()
	GetRegistry().SetContext(tctx, characterId, convCtx)
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{
		l:   l,
		ctx: tctx,
		t:   tm,
	}

	mustOp := func(t *testing.T, opType string, params map[string]string) OperationModel {
		t.Helper()
		b := NewOperationBuilder().SetType(opType)
		for k, v := range params {
			b.AddParamValue(k, v)
		}
		op, err := b.Build()
		if err != nil {
			t.Fatalf("failed to build op %s: %v", opType, err)
		}
		return op
	}

	ops := []OperationModel{
		mustOp(t, "award_exp", map[string]string{"amount": "10"}),
		mustOp(t, "award_item", map[string]string{"itemId": "2010000", "quantity": "3"}),
		mustOp(t, "award_item", map[string]string{"itemId": "2010009", "quantity": "3"}),
	}

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

	s, err := executor.createSagaForOperations(f, characterId, ops)
	if err != nil {
		t.Fatalf("createSagaForOperations returned error: %v", err)
	}

	if len(s.Steps) != len(ops) {
		t.Fatalf("expected %d steps, got %d", len(ops), len(s.Steps))
	}

	seen := make(map[string]int, len(s.Steps))
	for i, step := range s.Steps {
		seen[step.StepId]++
		if seen[step.StepId] > 1 {
			t.Errorf("duplicate stepId %q at index %d (steps: %+v)", step.StepId, i, stepIds(s.Steps))
		}
	}

	// First occurrence of each type keeps the bare "<type>-<characterId>"
	// stepId; subsequent occurrences are suffixed with the loop index.
	wantPrefix := []string{"award_exp-11", "award_item-11", "award_item-11-"}
	for i, want := range wantPrefix {
		got := s.Steps[i].StepId
		if i < 2 {
			if got != want {
				t.Errorf("step %d stepId = %q, want %q", i, got, want)
			}
			continue
		}
		if len(got) <= len(want) || got[:len(want)] != want {
			t.Errorf("step %d stepId = %q, want prefix %q", i, got, want)
		}
	}

	// Validate that the underlying actions/payloads survived dedup unchanged.
	if s.Steps[0].Action != saga.AwardExperience {
		t.Errorf("step 0 action = %v, want AwardExperience", s.Steps[0].Action)
	}
	for i := 1; i <= 2; i++ {
		if s.Steps[i].Action != saga.AwardAsset {
			t.Errorf("step %d action = %v, want AwardAsset", i, s.Steps[i].Action)
		}
	}
}

func stepIds(steps []sharedsaga.Step[any]) []string {
	out := make([]string, len(steps))
	for i, s := range steps {
		out[i] = s.StepId
	}
	return out
}
