package conversation

import (
	"atlas-npc-conversations/conversation/quest/progress"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

func TestSuppressAwardAssetByStartQuest(t *testing.T) {
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
	startQuest := func(rewards ...saga.QuestRewardItem) builtStep {
		return builtStep{
			stepId: "start",
			status: saga.Pending,
			action: saga.StartQuest,
			payload: saga.StartQuestPayload{
				CharacterId: 1,
				QuestId:     1000,
				Rewards:     rewards,
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
				QuestId:     2000,
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
			name:     "no start_quest leaves AwardAsset visible",
			input:    []builtStep{awardAsset(2000000, 1, true)},
			expected: []bool{true},
		},
		{
			name: "matching reward suppresses preceding AwardAsset",
			input: []builtStep{
				awardAsset(2000000, 1, true),
				startQuest(saga.QuestRewardItem{ItemId: 2000000, Amount: 1}),
			},
			expected: []bool{false},
		},
		{
			name: "partial-quantity mismatch leaves AwardAsset visible",
			input: []builtStep{
				awardAsset(2000000, 5, true),
				startQuest(saga.QuestRewardItem{ItemId: 2000000, Amount: 1}),
			},
			expected: []bool{true},
		},
		{
			name: "AwardAsset after StartQuest is not suppressed",
			input: []builtStep{
				startQuest(saga.QuestRewardItem{ItemId: 2000000, Amount: 1}),
				awardAsset(2000000, 1, true),
			},
			expected: []bool{true},
		},
		{
			name: "batch with both StartQuest and CompleteQuest suppresses independently",
			input: []builtStep{
				awardAsset(2000000, 1, true),
				awardAsset(2000001, 1, true),
				startQuest(saga.QuestRewardItem{ItemId: 2000000, Amount: 1}),
				completeQuest(saga.QuestRewardItem{ItemId: 2000001, Amount: 1}),
			},
			expected: []bool{false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := append([]builtStep(nil), tt.input...)
			// Apply both suppressors to mirror production behavior; the two
			// helpers are independent and each only inspects its own action.
			suppressAwardAssetByCompleteQuest(steps)
			suppressAwardAssetByStartQuest(steps)

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

// TestCreateSagaForOperations_SiblingRewardsWriteIntoStartQuest asserts that
// when a conversation batch contains both AwardAsset steps and a StartQuest
// step, the sibling item rewards are propagated into the StartQuest payload's
// Rewards field so downstream services can display them on quest start.
func TestCreateSagaForOperations_SiblingRewardsWriteIntoStartQuest(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)

	characterId := uint32(22)

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
		mustOp(t, "award_item", map[string]string{"itemId": "2000000", "quantity": "2"}),
		mustOp(t, "start_quest", map[string]string{"questId": "1000", "npcId": "9001"}),
	}

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

	s, err := executor.createSagaForOperations(f, characterId, ops)
	if err != nil {
		t.Fatalf("createSagaForOperations returned error: %v", err)
	}

	var sawStart bool
	for _, step := range s.Steps {
		if step.Action != saga.StartQuest {
			continue
		}
		sawStart = true
		sqp, ok := step.Payload.(saga.StartQuestPayload)
		if !ok {
			t.Fatalf("start_quest payload has unexpected type %T", step.Payload)
		}
		if len(sqp.Rewards) != 1 {
			t.Fatalf("expected 1 reward on StartQuestPayload, got %d", len(sqp.Rewards))
		}
		if sqp.Rewards[0].ItemId != 2000000 || sqp.Rewards[0].Amount != 2 {
			t.Errorf("unexpected reward: %+v", sqp.Rewards[0])
		}
	}
	if !sawStart {
		t.Fatalf("no start_quest step found in saga (steps=%+v)", stepIds(s.Steps))
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

func stepIds(steps []saga.Step[any]) []string {
	out := make([]string, len(steps))
	for i, s := range steps {
		out[i] = s.StepId
	}
	return out
}

func TestExecuteOpenDuey(t *testing.T) {
	t.Run("emits show_parcel", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
		InitRegistry(rc)
		l, _ := test.NewNullLogger()
		var tm tenant.Model
		tctx := tenant.WithContext(context.Background(), tm)
		characterId := uint32(100)
		GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().
			SetCharacterId(characterId).
			SetNpcId(9010009).
			Build())
		defer GetRegistry().ClearContext(tctx, characterId)

		executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}

		op, err := NewOperationBuilder().SetType("open_duey").Build()
		if err != nil {
			t.Fatalf("build op: %v", err)
		}

		f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
		_, status, action, payload, err := executor.createStepForOperation(f, characterId, op)
		if err != nil {
			t.Fatalf("createStepForOperation returned error: %v", err)
		}
		if action != saga.ShowParcel {
			t.Fatalf("expected action ShowParcel, got %q", action)
		}
		if status != saga.Pending {
			t.Fatalf("expected status Pending, got %q", status)
		}
		p, ok := payload.(saga.ShowParcelPayload)
		if !ok {
			t.Fatalf("expected ShowParcelPayload, got %T", payload)
		}
		if p.CharacterId != characterId {
			t.Errorf("expected CharacterId %d, got %d", characterId, p.CharacterId)
		}
		if p.NpcId != 9010009 {
			t.Errorf("expected NpcId 9010009, got %d", p.NpcId)
		}
		if p.WorldId != world.Id(0) {
			t.Errorf("expected WorldId 0, got %d", p.WorldId)
		}
		if p.ChannelId != channel.Id(1) {
			t.Errorf("expected ChannelId 1, got %d", p.ChannelId)
		}
		if p.Quick {
			t.Error("expected Quick to be false")
		}
	})

	t.Run("missing context", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
		InitRegistry(rc)
		l, _ := test.NewNullLogger()
		var tm tenant.Model
		tctx := tenant.WithContext(context.Background(), tm)
		characterId := uint32(101)

		executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}

		op, err := NewOperationBuilder().SetType("open_duey").Build()
		if err != nil {
			t.Fatalf("build op: %v", err)
		}

		f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
		_, _, _, payload, err := executor.createStepForOperation(f, characterId, op)
		if err == nil {
			t.Fatal("expected error when no conversation context exists")
		}
		if !strings.Contains(err.Error(), "conversation context") {
			t.Errorf("expected error to mention conversation context, got %q", err.Error())
		}
		if payload != nil {
			t.Errorf("expected nil payload, got %v", payload)
		}
	})
}

func TestCreateStepForOperation_RebalanceAP_SingleTarget(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(77)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}

	op, err := NewOperationBuilder().
		SetType("rebalance_ap").
		AddParamValue("targets", `[{"stat":"dexterity","floor":"20"}]`).
		Build()
	if err != nil {
		t.Fatalf("build op: %v", err)
	}

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	stepId, status, action, payload, err := executor.createStepForOperation(f, characterId, op)
	if err != nil {
		t.Fatalf("createStepForOperation returned error: %v", err)
	}
	if action != saga.RebalanceAP {
		t.Fatalf("expected action RebalanceAP, got %q", action)
	}
	if status != saga.Pending {
		t.Fatalf("expected status Pending, got %q", status)
	}
	if stepId == "" {
		t.Fatal("expected non-empty stepId")
	}
	p, ok := payload.(saga.RebalanceAPPayload)
	if !ok {
		t.Fatalf("expected RebalanceAPPayload, got %T", payload)
	}
	if p.CharacterId != characterId {
		t.Errorf("characterId: expected %d, got %d", characterId, p.CharacterId)
	}
	if len(p.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(p.Targets))
	}
	if p.Targets[0].Stat != saga.RebalanceStatDexterity || p.Targets[0].Floor != 20 {
		t.Errorf("unexpected target: %+v", p.Targets[0])
	}
}

func TestCreateStepForOperation_RebalanceAP_MultiTarget(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(78)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}

	op, err := NewOperationBuilder().
		SetType("rebalance_ap").
		AddParamValue("targets", `[{"stat":"strength","floor":"20"},{"stat":"dexterity","floor":"20"}]`).
		Build()
	if err != nil {
		t.Fatalf("build op: %v", err)
	}
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	_, _, action, payload, err := executor.createStepForOperation(f, characterId, op)
	if err != nil {
		t.Fatalf("createStepForOperation returned error: %v", err)
	}
	if action != saga.RebalanceAP {
		t.Fatalf("expected RebalanceAP action")
	}
	p := payload.(saga.RebalanceAPPayload)
	if len(p.Targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(p.Targets))
	}
	if p.Targets[0].Stat != saga.RebalanceStatStrength || p.Targets[0].Floor != 20 {
		t.Errorf("target[0] unexpected: %+v", p.Targets[0])
	}
	if p.Targets[1].Stat != saga.RebalanceStatDexterity || p.Targets[1].Floor != 20 {
		t.Errorf("target[1] unexpected: %+v", p.Targets[1])
	}
}

func TestCreateStepForOperation_RebalanceAP_RejectsEmpty(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(79)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}

	op, _ := NewOperationBuilder().
		SetType("rebalance_ap").
		AddParamValue("targets", `[]`).
		Build()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	if _, _, _, _, err := executor.createStepForOperation(f, characterId, op); err == nil {
		t.Fatal("expected error on empty targets")
	}
}

func TestCreateStepForOperation_RebalanceAP_RejectsDuplicateStat(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(80)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}

	op, _ := NewOperationBuilder().
		SetType("rebalance_ap").
		AddParamValue("targets", `[{"stat":"dexterity","floor":"20"},{"stat":"dexterity","floor":"25"}]`).
		Build()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	if _, _, _, _, err := executor.createStepForOperation(f, characterId, op); err == nil {
		t.Fatal("expected error on duplicate stat")
	}
}

func TestCreateStepForOperation_RebalanceAP_RejectsInvalidStat(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(81)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}

	op, _ := NewOperationBuilder().
		SetType("rebalance_ap").
		AddParamValue("targets", `[{"stat":"banana","floor":"20"}]`).
		Build()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	if _, _, _, _, err := executor.createStepForOperation(f, characterId, op); err == nil {
		t.Fatal("expected error on invalid stat")
	}
}

func TestCreateStepForOperation_RebalanceAP_RejectsFloorBelow4(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(82)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}

	op, _ := NewOperationBuilder().
		SetType("rebalance_ap").
		AddParamValue("targets", `[{"stat":"dexterity","floor":"3"}]`).
		Build()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	if _, _, _, _, err := executor.createStepForOperation(f, characterId, op); err == nil {
		t.Fatal("expected error on floor below 4")
	}
}

func TestCreateStepForOperation_RebalanceAP_RejectsMalformedJSON(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(83)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}

	op, _ := NewOperationBuilder().
		SetType("rebalance_ap").
		AddParamValue("targets", `[not json`).
		Build()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	if _, _, _, _, err := executor.createStepForOperation(f, characterId, op); err == nil {
		t.Fatal("expected error on malformed JSON")
	}
}

// TestCreateSagaForOperation_ChangeJobAppendsCancelAllBuffs verifies that the
// single-operation saga for a change_job operation also emits a cancel_all_buffs
// step, so a job change clears all active buffs (and dismounts any
// MONSTER_RIDING mount, FR-4.2).
func TestCreateSagaForOperation_ChangeJobAppendsCancelAllBuffs(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)

	characterId := uint32(33)

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

	op, err := NewOperationBuilder().
		SetType("change_job").
		AddParamValue("jobId", "200").
		Build()
	if err != nil {
		t.Fatalf("failed to build change_job op: %v", err)
	}

	worldId := world.Id(0)
	channelId := channel.Id(1)
	f := field.NewBuilder(worldId, channelId, _map.Id(100000000)).Build()

	s, err := executor.createSagaForOperation(f, characterId, op)
	if err != nil {
		t.Fatalf("createSagaForOperation returned error: %v", err)
	}

	if len(s.Steps) != 2 {
		t.Fatalf("expected 2 steps (change_job + cancel_all_buffs), got %d: %+v", len(s.Steps), stepIds(s.Steps))
	}
	if s.Steps[0].Action != saga.ChangeJob {
		t.Errorf("step 0 action = %v, want ChangeJob", s.Steps[0].Action)
	}
	if s.Steps[1].Action != saga.CancelAllBuffs {
		t.Errorf("step 1 action = %v, want CancelAllBuffs", s.Steps[1].Action)
	}

	pl, ok := s.Steps[1].Payload.(saga.CancelAllBuffsPayload)
	if !ok {
		t.Fatalf("cancel_all_buffs payload type = %T, want CancelAllBuffsPayload", s.Steps[1].Payload)
	}
	if pl.CharacterId != characterId {
		t.Errorf("cancel_all_buffs CharacterId = %d, want %d", pl.CharacterId, characterId)
	}
	if pl.WorldId != worldId {
		t.Errorf("cancel_all_buffs WorldId = %d, want %d", pl.WorldId, worldId)
	}
	if pl.ChannelId != channelId {
		t.Errorf("cancel_all_buffs ChannelId = %d, want %d", pl.ChannelId, channelId)
	}
}

// TestCreateSagaForOperations_ChangeJobAppendsCancelAllBuffs verifies that the
// batch saga path appends a single cancel_all_buffs step when a change_job
// operation is present alongside other operations (FR-4.2).
func TestCreateSagaForOperations_ChangeJobAppendsCancelAllBuffs(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)

	characterId := uint32(44)

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
		mustOp(t, "award_item", map[string]string{"itemId": "2000000", "quantity": "1"}),
		mustOp(t, "change_job", map[string]string{"jobId": "200"}),
	}

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

	s, err := executor.createSagaForOperations(f, characterId, ops)
	if err != nil {
		t.Fatalf("createSagaForOperations returned error: %v", err)
	}

	// 2 source ops + 1 appended cancel_all_buffs step.
	if len(s.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d: %+v", len(s.Steps), stepIds(s.Steps))
	}

	var cancelCount int
	for _, step := range s.Steps {
		if step.Action == saga.CancelAllBuffs {
			cancelCount++
		}
	}
	if cancelCount != 1 {
		t.Fatalf("expected exactly 1 cancel_all_buffs step, got %d: %+v", cancelCount, stepIds(s.Steps))
	}

	// The cancel step is appended after the change_job step.
	last := s.Steps[len(s.Steps)-1]
	if last.Action != saga.CancelAllBuffs {
		t.Errorf("last step action = %v, want CancelAllBuffs", last.Action)
	}
}

// fakeProgressProcessor is an inline test double for progress.Processor.
type fakeProgressProcessor struct {
	entries []progress.Model
	err     error
	called  *bool
}

func (f fakeProgressProcessor) GetByCharacterAndQuest(characterId uint32, questId uint32) model.Provider[[]progress.Model] {
	return func() ([]progress.Model, error) {
		if f.called != nil {
			*f.called = true
		}
		return f.entries, f.err
	}
}

func mustProgressEntry(t *testing.T, infoNumber uint32, progressValue string) progress.Model {
	t.Helper()
	m, err := progress.Extract(progress.RestModel{InfoNumber: infoNumber, Progress: progressValue})
	if err != nil {
		t.Fatalf("failed to build progress entry: %v", err)
	}
	return m
}

func TestGetQuestProgressOperation(t *testing.T) {
	transportErr := errors.New("connection refused")

	tests := []struct {
		name          string
		params        map[string]string
		seedContext   map[string]string
		entries       []func(t *testing.T) progress.Model
		procErr       error
		wantErr       bool
		wantErrSubstr string
		wantContext   *string
	}{
		{
			name:   "default info number",
			params: map[string]string{"questId": "3360", "contextKey": "magatiaPassword"},
			entries: []func(t *testing.T) progress.Model{
				func(t *testing.T) progress.Model { return mustProgressEntry(t, 0, "Open Sesame") },
			},
			wantContext: strPtr("Open Sesame"),
		},
		{
			name:   "named info number",
			params: map[string]string{"questId": "20730", "infoNumber": "9300285", "contextKey": "gate"},
			entries: []func(t *testing.T) progress.Model{
				func(t *testing.T) progress.Model { return mustProgressEntry(t, 0, "x") },
				func(t *testing.T) progress.Model { return mustProgressEntry(t, 9300285, "0") },
			},
			wantContext: strPtr("0"),
		},
		{
			name:   "step alias",
			params: map[string]string{"questId": "20730", "step": "9300285", "contextKey": "gate"},
			entries: []func(t *testing.T) progress.Model{
				func(t *testing.T) progress.Model { return mustProgressEntry(t, 0, "x") },
				func(t *testing.T) progress.Model { return mustProgressEntry(t, 9300285, "0") },
			},
			wantContext: strPtr("0"),
		},
		{
			name:   "infoNumber wins over step",
			params: map[string]string{"questId": "20730", "infoNumber": "9300285", "step": "1", "contextKey": "gate"},
			entries: []func(t *testing.T) progress.Model{
				func(t *testing.T) progress.Model { return mustProgressEntry(t, 0, "x") },
				func(t *testing.T) progress.Model { return mustProgressEntry(t, 9300285, "0") },
			},
			wantContext: strPtr("0"),
		},
		{
			name:   "numeric-looking progress stays a string",
			params: map[string]string{"questId": "6400", "contextKey": "seagull"},
			entries: []func(t *testing.T) progress.Model{
				func(t *testing.T) progress.Model { return mustProgressEntry(t, 0, "72") },
			},
			wantContext: strPtr("72"),
		},
		{
			name:        "unstarted quest",
			params:      map[string]string{"questId": "3360", "contextKey": "pw"},
			procErr:     progress.ErrNotFound,
			wantContext: strPtr(""),
		},
		{
			name:   "info number absent from collection",
			params: map[string]string{"questId": "3360", "infoNumber": "5", "contextKey": "pw"},
			entries: []func(t *testing.T) progress.Model{
				func(t *testing.T) progress.Model { return mustProgressEntry(t, 0, "x") },
			},
			wantContext: strPtr(""),
		},
		{
			name:        "empty collection",
			params:      map[string]string{"questId": "3360", "contextKey": "pw"},
			entries:     []func(t *testing.T) progress.Model{},
			wantContext: strPtr(""),
		},
		{
			name:    "transport error",
			params:  map[string]string{"questId": "3360", "contextKey": "pw"},
			procErr: transportErr,
			wantErr: true,
		},
		{
			name:          "missing questId",
			params:        map[string]string{"contextKey": "pw"},
			wantErr:       true,
			wantErrSubstr: "questId",
		},
		{
			name:          "missing contextKey",
			params:        map[string]string{"questId": "3360"},
			wantErr:       true,
			wantErrSubstr: "contextKey",
		},
		{
			name:        "questId from context",
			params:      map[string]string{"questId": "{context.qid}", "contextKey": "pw"},
			seedContext: map[string]string{"qid": "3360"},
			entries: []func(t *testing.T) progress.Model{
				func(t *testing.T) progress.Model { return mustProgressEntry(t, 0, "z") },
			},
			wantContext: strPtr("z"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mr := miniredis.RunT(t)
			rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
			InitRegistry(rc)

			l, _ := test.NewNullLogger()
			l.SetLevel(logrus.DebugLevel)

			var tm tenant.Model
			tctx := tenant.WithContext(context.Background(), tm)
			characterId := uint32(77)

			builder := NewConversationContextBuilder().
				SetCharacterId(characterId).
				AddContextValue("_seed", "1")
			for k, v := range tc.seedContext {
				builder = builder.AddContextValue(k, v)
			}
			convCtx := builder.Build()
			GetRegistry().SetContext(tctx, characterId, convCtx)
			defer GetRegistry().ClearContext(tctx, characterId)

			var entries []progress.Model
			for _, f := range tc.entries {
				entries = append(entries, f(t))
			}

			var called bool
			progressExpectedNotCalled := tc.name == "missing questId" || tc.name == "missing contextKey"
			procP := fakeProgressProcessor{entries: entries, err: tc.procErr, called: &called}

			executor := &OperationExecutorImpl{
				l:              l,
				ctx:            tctx,
				t:              tm,
				questProgressP: procP,
			}

			op, err := func() (OperationModel, error) {
				b := NewOperationBuilder().SetType("local:get_quest_progress")
				for k, v := range tc.params {
					b.AddParamValue(k, v)
				}
				return b.Build()
			}()
			if err != nil {
				t.Fatalf("failed to build op: %v", err)
			}

			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

			err = executor.ExecuteOperation(f, characterId, op)

			if progressExpectedNotCalled && called {
				t.Errorf("quest progress processor was called for case %q, expected it not to be", tc.name)
			}

			if tc.wantErr {
				if err == nil {
					t.Fatalf("ExecuteOperation returned nil error, want non-nil")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ExecuteOperation returned error: %v", err)
			}

			if tc.wantContext != nil {
				got, err := executor.getContextValue(characterId, tc.params["contextKey"])
				if err != nil {
					t.Fatalf("failed to read context key %q: %v", tc.params["contextKey"], err)
				}
				if got != *tc.wantContext {
					t.Errorf("context[%q] = %q, want %q", tc.params["contextKey"], got, *tc.wantContext)
				}
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

// TestCreateStepSendMessageDefaultsMessageType verifies that send_message no
// longer requires messageType (FR-13; previously errors.New("missing
// messageType parameter for send_message operation")), accepts the `type`
// alias, and maps the numeric "6" to BLUE_TEXT.
func TestCreateStepSendMessageDefaultsMessageType(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(80)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

	tests := []struct {
		name            string
		params          map[string]string
		wantMessageType string
	}{
		{
			name:            "no messageType defaults to PINK_TEXT",
			params:          map[string]string{"message": "hi"},
			wantMessageType: "PINK_TEXT",
		},
		{
			name:            "type alias",
			params:          map[string]string{"message": "hi", "type": "NOTICE"},
			wantMessageType: "NOTICE",
		},
		{
			name:            "numeric type alias maps to BLUE_TEXT",
			params:          map[string]string{"message": "hi", "type": "6"},
			wantMessageType: "BLUE_TEXT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, err := NewOperationBuilder().SetType("send_message").SetParams(tt.params).Build()
			if err != nil {
				t.Fatalf("build op: %v", err)
			}
			_, status, action, payload, err := executor.createStepForOperation(f, characterId, op)
			if err != nil {
				t.Fatalf("createStepForOperation returned error: %v", err)
			}
			if status != saga.Pending {
				t.Fatalf("expected status Pending, got %q", status)
			}
			if action != saga.SendMessage {
				t.Fatalf("expected action SendMessage, got %q", action)
			}
			p, ok := payload.(saga.SendMessagePayload)
			if !ok {
				t.Fatalf("expected SendMessagePayload, got %T", payload)
			}
			if p.MessageType != tt.wantMessageType {
				t.Errorf("MessageType = %q, want %q", p.MessageType, tt.wantMessageType)
			}
			if p.Message != "hi" {
				t.Errorf("Message = %q, want %q", p.Message, "hi")
			}
		})
	}
}

// TestCreateStepSpawnMonsterOptionalPosition verifies that spawn_monster no
// longer requires x/y (FR-15/FR-16; previously errors.New("missing x
// parameter for spawn_monster operation")) and that Instance carries the
// target field's instance only when the spawn targets the same map (OQ-3).
func TestCreateStepSpawnMonsterOptionalPosition(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(81)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}
	instID := uuid.New()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910010000)).SetInstance(instID).Build()

	t.Run("no x/y defaults to 0,0 and inherits the target's instance", func(t *testing.T) {
		op, err := NewOperationBuilder().
			SetType("spawn_monster").
			AddParamValue("monsterId", "100100").
			Build()
		if err != nil {
			t.Fatalf("build op: %v", err)
		}
		_, status, action, payload, err := executor.createStepForOperation(f, characterId, op)
		if err != nil {
			t.Fatalf("createStepForOperation returned error: %v", err)
		}
		if status != saga.Pending {
			t.Fatalf("expected status Pending, got %q", status)
		}
		if action != saga.SpawnMonster {
			t.Fatalf("expected action SpawnMonster, got %q", action)
		}
		p, ok := payload.(saga.SpawnMonsterPayload)
		if !ok {
			t.Fatalf("expected SpawnMonsterPayload, got %T", payload)
		}
		if p.X != 0 || p.Y != 0 {
			t.Errorf("X,Y = %d,%d, want 0,0", p.X, p.Y)
		}
		if p.Instance != instID {
			t.Errorf("Instance = %s, want %s", p.Instance, instID)
		}
	})

	t.Run("mapId targeting a different map carries uuid.Nil instance", func(t *testing.T) {
		op, err := NewOperationBuilder().
			SetType("spawn_monster").
			AddParamValue("monsterId", "100100").
			AddParamValue("mapId", "910510202").
			Build()
		if err != nil {
			t.Fatalf("build op: %v", err)
		}
		_, _, _, payload, err := executor.createStepForOperation(f, characterId, op)
		if err != nil {
			t.Fatalf("createStepForOperation returned error: %v", err)
		}
		p, ok := payload.(saga.SpawnMonsterPayload)
		if !ok {
			t.Fatalf("expected SpawnMonsterPayload, got %T", payload)
		}
		if p.Instance != uuid.Nil {
			t.Errorf("Instance = %s, want uuid.Nil", p.Instance)
		}
	})
}

// TestCreateStepCreateSkillHonoursExpiration verifies that create_skill now
// reads the expiration parameter (design §5.4; operation_executor.go
// previously hard-coded a ~1 year expiration regardless of input) and that
// the "-1" sentinel resolves to an expiration far beyond the old 1-year
// default.
func TestCreateStepCreateSkillHonoursExpiration(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(82)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

	op, err := NewOperationBuilder().
		SetType("create_skill").
		AddParamValue("skillId", "1001003").
		AddParamValue("expiration", "-1").
		Build()
	if err != nil {
		t.Fatalf("build op: %v", err)
	}
	_, status, action, payload, err := executor.createStepForOperation(f, characterId, op)
	if err != nil {
		t.Fatalf("createStepForOperation returned error: %v", err)
	}
	if status != saga.Pending {
		t.Fatalf("expected status Pending, got %q", status)
	}
	if action != saga.CreateSkill {
		t.Fatalf("expected action CreateSkill, got %q", action)
	}
	p, ok := payload.(saga.CreateSkillPayload)
	if !ok {
		t.Fatalf("expected CreateSkillPayload, got %T", payload)
	}
	minExpiration := time.Now().Add(50 * 365 * 24 * time.Hour)
	if !p.Expiration.After(minExpiration) {
		t.Errorf("Expiration = %s, want after %s", p.Expiration, minExpiration)
	}
}

// TestCreateStepWarpToMapRequiresMapId verifies that warp_to_map now delegates
// to ops.WarpToPortal (FR-18), which requires mapId — operation_executor.go
// previously defaulted a missing mapId to 0 (saga.WarpToPortalPayload{MapId: 0})
// instead of erroring.
func TestCreateStepWarpToMapRequiresMapId(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(83)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

	t.Run("missing mapId errors", func(t *testing.T) {
		op, err := NewOperationBuilder().SetType("warp_to_map").Build()
		if err != nil {
			t.Fatalf("build op: %v", err)
		}
		_, _, _, _, err = executor.createStepForOperation(f, characterId, op)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "warp_to_portal") || !strings.Contains(err.Error(), "mapId") {
			t.Errorf("error = %q, want it to name warp_to_portal and mapId", err.Error())
		}
	})

	t.Run("portalId without mapId errors", func(t *testing.T) {
		op, err := NewOperationBuilder().SetType("warp_to_map").AddParamValue("portalId", "3").Build()
		if err != nil {
			t.Fatalf("build op: %v", err)
		}
		_, _, _, _, err = executor.createStepForOperation(f, characterId, op)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "warp_to_portal") || !strings.Contains(err.Error(), "mapId") {
			t.Errorf("error = %q, want it to name warp_to_portal and mapId", err.Error())
		}
	})

	t.Run("mapId and portalName still produce a valid step", func(t *testing.T) {
		op, err := NewOperationBuilder().
			SetType("warp_to_map").
			AddParamValue("mapId", "104000000").
			AddParamValue("portalName", "west00").
			Build()
		if err != nil {
			t.Fatalf("build op: %v", err)
		}
		_, status, action, payload, err := executor.createStepForOperation(f, characterId, op)
		if err != nil {
			t.Fatalf("createStepForOperation returned error: %v", err)
		}
		if status != saga.Pending {
			t.Fatalf("expected status Pending, got %q", status)
		}
		if action != saga.WarpToPortal {
			t.Fatalf("expected action WarpToPortal, got %q", action)
		}
		p, ok := payload.(saga.WarpToPortalPayload)
		if !ok {
			t.Fatalf("expected WarpToPortalPayload, got %T", payload)
		}
		if p.MapId != _map.Id(104000000) {
			t.Errorf("MapId = %d, want %d", p.MapId, _map.Id(104000000))
		}
		if p.PortalName != "west00" {
			t.Errorf("PortalName = %q, want %q", p.PortalName, "west00")
		}
	})
}

// TestCreateStepStartQuestUsesContextDefaults verifies that start_quest reads
// the conversation context at most once (design OQ-4), passing questId/npcId
// context defaults into ops.StartQuest rather than the old two separate
// GetPreviousContext lookups.
func TestCreateStepStartQuestUsesContextDefaults(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(84)
	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

	t.Run("no questId param uses context questId and npcId", func(t *testing.T) {
		GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().
			SetCharacterId(characterId).
			SetNpcId(9010000).
			AddContextValue("questId", "2000").
			Build())
		defer GetRegistry().ClearContext(tctx, characterId)

		op, err := NewOperationBuilder().SetType("start_quest").Build()
		if err != nil {
			t.Fatalf("build op: %v", err)
		}
		_, status, action, payload, err := executor.createStepForOperation(f, characterId, op)
		if err != nil {
			t.Fatalf("createStepForOperation returned error: %v", err)
		}
		if status != saga.Pending {
			t.Fatalf("expected status Pending, got %q", status)
		}
		if action != saga.StartQuest {
			t.Fatalf("expected action StartQuest, got %q", action)
		}
		p, ok := payload.(saga.StartQuestPayload)
		if !ok {
			t.Fatalf("expected StartQuestPayload, got %T", payload)
		}
		if p.QuestId != 2000 || p.NpcId != 9010000 {
			t.Errorf("QuestId/NpcId = %d/%d, want 2000/9010000", p.QuestId, p.NpcId)
		}
	})

	t.Run("explicit questId overrides context but npcId still defaults", func(t *testing.T) {
		GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().
			SetCharacterId(characterId).
			SetNpcId(9010000).
			AddContextValue("questId", "2000").
			Build())
		defer GetRegistry().ClearContext(tctx, characterId)

		op, err := NewOperationBuilder().SetType("start_quest").AddParamValue("questId", "3000").Build()
		if err != nil {
			t.Fatalf("build op: %v", err)
		}
		_, _, _, payload, err := executor.createStepForOperation(f, characterId, op)
		if err != nil {
			t.Fatalf("createStepForOperation returned error: %v", err)
		}
		p, ok := payload.(saga.StartQuestPayload)
		if !ok {
			t.Fatalf("expected StartQuestPayload, got %T", payload)
		}
		if p.QuestId != 3000 || p.NpcId != 9010000 {
			t.Errorf("QuestId/NpcId = %d/%d, want 3000/9010000", p.QuestId, p.NpcId)
		}
	})

	t.Run("empty context and no questId param errors", func(t *testing.T) {
		GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
		defer GetRegistry().ClearContext(tctx, characterId)

		op, err := NewOperationBuilder().SetType("start_quest").Build()
		if err != nil {
			t.Fatalf("build op: %v", err)
		}
		_, _, _, _, err = executor.createStepForOperation(f, characterId, op)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "start_quest") || !strings.Contains(err.Error(), "questId") {
			t.Errorf("error = %q, want it to name start_quest and questId", err.Error())
		}
	})
}

// TestCreateStepStageClearAttemptPq verifies that stage_clear_attempt_pq now
// delegates to ops.StageClearAttemptPq (FR-17), passing uuid.Nil for the
// instance to route the orchestrator to the character-lookup branch.
func TestCreateStepStageClearAttemptPq(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(85)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

	op, err := NewOperationBuilder().SetType("stage_clear_attempt_pq").Build()
	if err != nil {
		t.Fatalf("build op: %v", err)
	}
	_, status, action, payload, err := executor.createStepForOperation(f, characterId, op)
	if err != nil {
		t.Fatalf("createStepForOperation returned error: %v", err)
	}
	if status != saga.Pending {
		t.Fatalf("expected status Pending, got %q", status)
	}
	if action != saga.StageClearAttemptPq {
		t.Fatalf("expected action StageClearAttemptPq, got %q", action)
	}
	p, ok := payload.(saga.StageClearAttemptPqPayload)
	if !ok {
		t.Fatalf("expected StageClearAttemptPqPayload, got %T", payload)
	}
	if p.CharacterId != characterId {
		t.Errorf("CharacterId = %d, want %d", p.CharacterId, characterId)
	}
	if p.InstanceId != uuid.Nil {
		t.Errorf("InstanceId = %s, want uuid.Nil", p.InstanceId)
	}
}
