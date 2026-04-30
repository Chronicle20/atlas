package saga

// TestPresetCompensation verifies that when a preset application fails mid-saga
// (specifically on the second create_and_equip_asset step), the compensation
// reverse-walk correctly dispatches rollback commands for all previously-
// completed steps.
//
// Per task-037 PRD acceptance §10 #6.
//
// Saga shape mirrors buildPresetCharacterCreationSaga:
//
//	step 0: create_character               ← Completed
//	step 1: award_asset_0    (item 0)      ← Completed
//	step 2: award_asset_1    (item 1)      ← Completed
//	step 3: create_and_equip_asset_0       ← Completed (first equip succeeded)
//	step 4: create_and_equip_asset_1       ← Failed    (forced failure here)
//	step 5: create_skill_0                 ← Pending   (never reached)
//
// Expected compensation dispatches from DispatchCharacterCreationRollbacks:
//   - DestroyItem(TemplateId=1302000)   for create_and_equip_asset_0 (Completed)
//   - DestroyItem(TemplateId=2000002)   for award_asset_1 (Completed)
//   - DestroyItem(TemplateId=2000001)   for award_asset_0 (Completed)
//   - DeleteCharacter(characterId=99001) for create_character (Completed)
//   - NO DeleteSkill                    — create_skill_0 was Pending, not Completed
//
// The test exercises DispatchCharacterCreationRollbacks directly to avoid
// triggering the EmitSagaFailed Kafka path (no broker in the test environment).
// The lifecycle Compensating → Failed transition is verified separately via
// TryTransition to confirm cache state progresses correctly.

import (
	"atlas-saga-orchestrator/character/mock"
	mock2 "atlas-saga-orchestrator/compartment/mock"
	"atlas-saga-orchestrator/kafka/message"
	"atlas-saga-orchestrator/skill"
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// presetTestSkillMock is a minimal inline implementation of skill.Processor for testing.
// It captures RequestDeleteSkill calls so the test can assert the right skills were deleted.
type presetTestSkillMock struct {
	calls []presetSkillDeleteCall
}

type presetSkillDeleteCall struct {
	TransactionId uuid.UUID
	WorldId       world.Id
	CharacterId   uint32
	SkillId       uint32
}

func (s *presetTestSkillMock) RequestCreateAndEmit(_ uuid.UUID, _ world.Id, _ uint32, _ uint32, _ byte, _ byte, _ time.Time) error {
	return nil
}

func (s *presetTestSkillMock) RequestCreate(_ *message.Buffer) func(uuid.UUID, world.Id, uint32, uint32, byte, byte, time.Time) error {
	return func(_ uuid.UUID, _ world.Id, _ uint32, _ uint32, _ byte, _ byte, _ time.Time) error {
		return nil
	}
}

func (s *presetTestSkillMock) RequestUpdateAndEmit(_ uuid.UUID, _ world.Id, _ uint32, _ uint32, _ byte, _ byte, _ time.Time) error {
	return nil
}

func (s *presetTestSkillMock) RequestUpdate(_ *message.Buffer) func(uuid.UUID, world.Id, uint32, uint32, byte, byte, time.Time) error {
	return func(_ uuid.UUID, _ world.Id, _ uint32, _ uint32, _ byte, _ byte, _ time.Time) error {
		return nil
	}
}

func (s *presetTestSkillMock) RequestDeleteSkill(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32) error {
	s.calls = append(s.calls, presetSkillDeleteCall{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		SkillId:       skillId,
	})
	return nil
}

// Ensure presetTestSkillMock satisfies skill.Processor at compile time.
var _ skill.Processor = (*presetTestSkillMock)(nil)

func TestPresetCompensation(t *testing.T) {
	// ------------------------------------------------------------------ setup
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Track DestroyItem calls so we can assert each Completed item/equip step
	// has its item rolled back.
	type destroyCall struct {
		CharacterId uint32
		TemplateId  uint32
		Quantity    uint32
		RemoveAll   bool
	}
	var destroyItemCalls []destroyCall

	// Track DeleteCharacter call.
	var deleteCharacterCalled bool
	var deleteCharacterCharId uint32

	compP := &mock2.ProcessorMock{
		RequestDestroyItemFunc: func(_ uuid.UUID, characterId uint32, templateId uint32, quantity uint32, removeAll bool) error {
			destroyItemCalls = append(destroyItemCalls, destroyCall{
				CharacterId: characterId,
				TemplateId:  templateId,
				Quantity:    quantity,
				RemoveAll:   removeAll,
			})
			return nil
		},
	}

	charP := &mock.ProcessorMock{
		RequestDeleteCharacterFunc: func(_ uuid.UUID, characterId uint32, _ world.Id) error {
			deleteCharacterCalled = true
			deleteCharacterCharId = characterId
			return nil
		},
	}

	sm := &presetTestSkillMock{}

	// ------------------------------------------------------------------ build saga
	//
	// Preset shape: 1 create_character + 2 award_asset + 2 create_and_equip_asset + 1 create_skill.
	// Steps 0–3 are Completed; step 4 (second equip) is Failed (forced failure);
	// step 5 (create_skill) is Pending (never reached when failure occurred).
	//
	// Character ID is pre-populated in payloads (normally forwarded after the
	// CharacterCreated Kafka event triggers forwardCharacterCreationResult).
	const (
		testCharId  = uint32(99001)
		testWorldId = world.Id(0)
	)

	transactionId := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(CharacterCreation).
		SetInitiatedBy("preset-compensation-test").
		// step 0: create_character — Completed
		AddStep("create_character", Completed, CreateCharacter, CharacterCreatePayload{
			AccountId:    10001,
			Name:         "TestHero",
			WorldId:      testWorldId,
			Level:        1,
			Strength:     13,
			Dexterity:    4,
			Intelligence: 4,
			Luck:         4,
			Hp:           50,
			Mp:           5,
			JobId:        job.Id(0),
			Gender:       0,
			Face:         20000,
			Hair:         30000,
			Skin:         0,
			Top:          1040002,
			Bottom:       1060002,
			Shoes:        1072001,
			Weapon:       1302000,
			MapId:        _map.Id(100000000),
		}).
		// step 1: award_asset_0 — Completed
		AddStep("award_asset_0", Completed, AwardAsset, AwardItemActionPayload{
			CharacterId: testCharId,
			Item:        ItemPayload{TemplateId: 2000001, Quantity: 10},
		}).
		// step 2: award_asset_1 — Completed
		AddStep("award_asset_1", Completed, AwardAsset, AwardItemActionPayload{
			CharacterId: testCharId,
			Item:        ItemPayload{TemplateId: 2000002, Quantity: 5},
		}).
		// step 3: create_and_equip_asset_0 — Completed (first equipment succeeded)
		AddStep("create_and_equip_asset_0", Completed, CreateAndEquipAsset, CreateAndEquipAssetPayload{
			CharacterId: testCharId,
			Item:        ItemPayload{TemplateId: 1302000, Quantity: 1},
		}).
		// step 4: create_and_equip_asset_1 — Failed (forced failure; second equip fails)
		AddStep("create_and_equip_asset_1", Failed, CreateAndEquipAsset, CreateAndEquipAssetPayload{
			CharacterId: testCharId,
			Item:        ItemPayload{TemplateId: 1442079, Quantity: 1},
		}).
		// step 5: create_skill_0 — Pending (never reached when step 4 failed)
		AddStep("create_skill_0", Pending, CreateSkill, CreateSkillPayload{
			CharacterId: testCharId,
			WorldId:     testWorldId,
			SkillId:     1000,
			Level:       1,
			MasterLevel: 1,
		}).
		Build()
	assert.NoError(t, err, "saga build should not fail")

	// Attach a characterId result to the CreateCharacter step so
	// ExtractCharacterCreationIds correctly resolves the character ID during rollback.
	s, err = s.WithStepResult(0, map[string]any{"characterId": testCharId})
	assert.NoError(t, err, "WithStepResult should not fail")

	// Store the saga in cache so lifecycle transitions can operate.
	assert.NoError(t, GetCache().Put(tctx, s))

	// ------------------------------------------------------------------ lifecycle: Pending → Compensating
	//
	// Mirrors what emitFailedFromStepSyncError / StepCompleted(false) does before
	// the compensator runs. We replicate that step here to keep the test
	// self-contained without requiring a running Kafka broker.
	ok := GetCache().TryTransition(tctx, transactionId, SagaLifecyclePending, SagaLifecycleCompensating)
	assert.True(t, ok, "lifecycle should transition Pending → Compensating")

	// ------------------------------------------------------------------ dispatch rollbacks
	//
	// Call DispatchCharacterCreationRollbacks directly. This is the exported
	// dispatch primitive used by both CompensateFailedStep (step-triggered path)
	// and handleSagaTimeout (timer path). Calling it directly avoids the
	// EmitSagaFailed Kafka dependency, while still exercising the complete
	// reverse-walk compensation logic that PRD §10 #6 requires.
	compensator := NewCompensator(logger, tctx).
		WithCharacterProcessor(charP).
		WithCompartmentProcessor(compP).
		WithSkillProcessor(sm)

	compensator.DispatchCharacterCreationRollbacks(s)

	// ------------------------------------------------------------------ lifecycle: Compensating → Failed
	//
	// After rollbacks are dispatched, finalize the lifecycle as CompensateFailedStep
	// would — but without calling EmitSagaFailed (no broker in test environment).
	finalizedOk := GetCache().TryTransition(tctx, transactionId, SagaLifecycleCompensating, SagaLifecycleFailed)
	assert.True(t, finalizedOk, "lifecycle should transition Compensating → Failed after compensation")

	// Clean up cache (mirrors c.Remove call in compensateCharacterCreation).
	GetCache().Remove(tctx, transactionId)

	// ------------------------------------------------------------------ assertions

	// 1. Cache entry should be gone after eviction.
	_, lifecycleOk := GetCache().GetLifecycle(tctx, transactionId)
	assert.False(t, lifecycleOk, "saga should be evicted from cache after compensation")

	// 2. DestroyItem should have been dispatched for every Completed item/equip step.
	//    Expected: create_and_equip_asset_0 (templateId=1302000), award_asset_1 (2000002),
	//    award_asset_0 (2000001).  create_and_equip_asset_1 is Failed (not Completed),
	//    so its item was never created — no rollback needed.
	assert.Equal(t, 3, len(destroyItemCalls),
		"expected 3 DestroyItem dispatches (1 completed equip + 2 completed award items)")

	destroyedTemplates := make(map[uint32]bool)
	for _, call := range destroyItemCalls {
		assert.Equal(t, testCharId, call.CharacterId, "DestroyItem must target the test character")
		destroyedTemplates[call.TemplateId] = true
	}
	assert.True(t, destroyedTemplates[1302000],
		"should destroy equip item 1302000 (create_and_equip_asset_0, Completed)")
	assert.True(t, destroyedTemplates[2000001],
		"should destroy inventory item 2000001 (award_asset_0, Completed)")
	assert.True(t, destroyedTemplates[2000002],
		"should destroy inventory item 2000002 (award_asset_1, Completed)")

	// create_and_equip_asset_1 (Failed, not Completed) — item was never created,
	// so NO DestroyItem should be dispatched for templateId=1442079.
	assert.False(t, destroyedTemplates[1442079],
		"must NOT destroy item 1442079 (create_and_equip_asset_1 was Failed, not Completed)")

	// 3. DeleteCharacter must have been dispatched last (deferred by reverse-walk).
	assert.True(t, deleteCharacterCalled,
		"RequestDeleteCharacter should have been dispatched")
	assert.Equal(t, testCharId, deleteCharacterCharId,
		"DeleteCharacter must target the test character")

	// 4. create_skill_0 was Pending (never completed) — no skill delete should fire.
	assert.Equal(t, 0, len(sm.calls),
		"no DeleteSkill should fire for a Pending (never-completed) skill step")
}
