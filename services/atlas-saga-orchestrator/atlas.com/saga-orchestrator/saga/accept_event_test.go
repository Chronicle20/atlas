package saga

import (
	charactermock "atlas-saga-orchestrator/character/mock"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func acceptEventTestCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func newAcceptEventTestProcessor(t *testing.T) (Processor, *logtest.Hook, context.Context) {
	t.Helper()
	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := acceptEventTestCtx(t)
	return NewProcessor(logger, ctx), hook, ctx
}

func putAcceptEventSaga(t *testing.T, ctx context.Context, s Saga) {
	t.Helper()
	require.NoError(t, GetCache().Put(ctx, s))
}

func TestAcceptEvent_SagaNotFound(t *testing.T) {
	p, hook, _ := newAcceptEventTestProcessor(t)
	_, ok := p.AcceptEvent(uuid.New(), EventKindAssetCreated)
	assert.False(t, ok, "AcceptEvent returns false when saga is not found")

	entries := hook.AllEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, logrus.DebugLevel, entries[0].Level)
	assert.Equal(t, SkipReasonSagaNotFound, entries[0].Data["reason"])
}

func TestAcceptEvent_NoPendingStep(t *testing.T) {
	p, hook, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", Completed, SendMessage, SendMessagePayload{CharacterId: 1, Message: "hi"}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	_, ok := p.AcceptEvent(tx, EventKindAssetCreated)
	assert.False(t, ok)

	var debugEntry *logrus.Entry
	for i := range hook.AllEntries() {
		e := hook.AllEntries()[i]
		if e.Level == logrus.DebugLevel {
			debugEntry = e
		}
	}
	require.NotNil(t, debugEntry, "expected a debug-level skip log")
	assert.Equal(t, SkipReasonNoPendingStep, debugEntry.Data["reason"])
}

func TestAcceptEvent_ActionMismatch(t *testing.T) {
	p, hook, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", Pending, AwardAsset, AwardItemActionPayload{
			CharacterId: 1,
			Item:        ItemPayload{TemplateId: 2070015, Quantity: 1},
			ShowEffect:  true,
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	_, ok := p.AcceptEvent(tx, EventKindCharacterStatChanged)
	assert.False(t, ok, "STAT_CHANGED must not match AwardAsset (§9.1 bug)")

	var debugEntry *logrus.Entry
	for i := range hook.AllEntries() {
		e := hook.AllEntries()[i]
		if e.Level == logrus.DebugLevel {
			debugEntry = e
		}
	}
	require.NotNil(t, debugEntry, "expected a debug-level skip log")
	assert.Equal(t, SkipReasonActionMismatch, debugEntry.Data["reason"])
	assert.Equal(t, AwardAsset, debugEntry.Data["step_action"])
	assert.Equal(t, "s1", debugEntry.Data["step_id"])
	assert.Equal(t, EventKindCharacterStatChanged, debugEntry.Data["event_kind"])
	assert.Equal(t, tx.String(), debugEntry.Data["transaction_id"])
}

func TestAcceptEvent_Match(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", Pending, AwardAsset, AwardItemActionPayload{
			CharacterId: 1,
			Item:        ItemPayload{TemplateId: 2070015, Quantity: 1},
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	decision, ok := p.AcceptEvent(tx, EventKindAssetCreated)
	require.True(t, ok)
	assert.Equal(t, "s1", decision.Step.StepId())
	assert.Equal(t, AwardAsset, decision.Step.Action())
	assert.Equal(t, tx, decision.Saga.TransactionId())
}

func TestAcceptEvent_EvolvePetMatch(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("evolve", Pending, EvolvePet, EvolvePetPayload{
			CharacterId: 1,
			PetId:       2,
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	decision, ok := p.AcceptEvent(tx, EventKindPetEvolved)
	require.True(t, ok, "EVOLVED event must complete a pending evolve_pet step")
	assert.Equal(t, "evolve", decision.Step.StepId())
	assert.Equal(t, EvolvePet, decision.Step.Action())
	assert.Equal(t, tx, decision.Saga.TransactionId())
}

func TestAcceptEvent_EvolvePetRejectsClosenessChanged(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("evolve", Pending, EvolvePet, EvolvePetPayload{
			CharacterId: 1,
			PetId:       2,
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	_, ok := p.AcceptEvent(tx, EventKindPetClosenessChanged)
	assert.False(t, ok, "CLOSENESS_CHANGED must not complete an evolve_pet step")
}

func newRevivePetSaga(t *testing.T, ctx context.Context, tx uuid.UUID) {
	t.Helper()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(PetRevive).
		SetInitiatedBy("test").
		AddStep("revive", Pending, RevivePet, RevivePetPayload{
			CharacterId:      1,
			PetId:            2,
			SourceTemplateId: 5180000,
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)
}

func TestAcceptEvent_RevivePetMatchesRevived(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	newRevivePetSaga(t, ctx, tx)

	decision, ok := p.AcceptEvent(tx, EventKindPetRevived)
	require.True(t, ok, "REVIVED event must complete a pending revive_pet step")
	assert.Equal(t, "revive", decision.Step.StepId())
	assert.Equal(t, RevivePet, decision.Step.Action())
	assert.Equal(t, tx, decision.Saga.TransactionId())
}

func TestAcceptEvent_RevivePetMatchesReviveFailed(t *testing.T) {
	// REVIVE_FAILED must be accepted too, so the saga compensates the
	// already-completed destroy step immediately rather than waiting out the
	// flat timeout with the player's Water of Life gone.
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	newRevivePetSaga(t, ctx, tx)

	decision, ok := p.AcceptEvent(tx, EventKindPetReviveFailed)
	require.True(t, ok, "REVIVE_FAILED event must be accepted by a pending revive_pet step")
	assert.Equal(t, "revive", decision.Step.StepId())
	assert.Equal(t, RevivePet, decision.Step.Action())
}

func TestAcceptEvent_RevivePetRejectsClosenessChanged(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	newRevivePetSaga(t, ctx, tx)

	_, ok := p.AcceptEvent(tx, EventKindPetClosenessChanged)
	assert.False(t, ok, "CLOSENESS_CHANGED must not complete a revive_pet step")
}

func TestAcceptEvent_WarnOnceForUnmatchedEvent(t *testing.T) {
	logger, hook := logtest.NewNullLogger()
	ctx := acceptEventTestCtx(t)
	p := NewProcessor(logger, ctx)

	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", Pending, RebalanceAP, RebalanceAPPayload{CharacterId: 1}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	for i := 0; i < 3; i++ {
		_, _ = p.AcceptEvent(tx, EventKindAssetCreated)
	}

	warnCount := 0
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel && e.Data["reason"] == SkipReasonUnmatchedEvent {
			warnCount++
		}
	}
	assert.Equal(t, 1, warnCount, "warn log must fire exactly once per (tx, kind) even with repeated events")
}

func TestAcceptEvent_NoWarnWhenLaterStepAcceptsKind(t *testing.T) {
	logger, hook := logtest.NewNullLogger()
	ctx := acceptEventTestCtx(t)
	p := NewProcessor(logger, ctx)

	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", Pending, ChangeJob, ChangeJobPayload{CharacterId: 1, JobId: 400}).
		AddStep("s2", Pending, AwardAsset, AwardItemActionPayload{CharacterId: 1, Item: ItemPayload{TemplateId: 1, Quantity: 1}}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	_, _ = p.AcceptEvent(tx, EventKindAssetCreated)

	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel {
			t.Fatalf("no warn expected — some later step accepts AssetCreated, got %+v", e)
		}
	}
}

func TestAcceptEvent_NilTransactionId(t *testing.T) {
	p, hook, _ := newAcceptEventTestProcessor(t)

	_, ok := p.AcceptEvent(uuid.Nil, EventKindAssetCreated)
	assert.False(t, ok, "AcceptEvent must return false for uuid.Nil")

	require.Len(t, hook.AllEntries(), 1, "exactly one debug log expected")
	entry := hook.AllEntries()[0]
	assert.Equal(t, logrus.DebugLevel, entry.Level)
	assert.Equal(t, SkipReasonNilTransactionId, entry.Data["reason"])
	assert.NotEqual(t, SkipReasonSagaNotFound, entry.Data["reason"], "must NOT log saga_not_found for nil-UUID events")

	// transaction_id field must NOT be on the log payload — there is no
	// meaningful UUID to log.
	_, hasTxId := entry.Data["transaction_id"]
	assert.False(t, hasTxId, "transaction_id should be omitted from nil-UUID skip logs")
}

// terminalLifecycle drives the saga to the requested terminal state via the
// legal transition chain.
func terminalLifecycle(t *testing.T, ctx context.Context, tx uuid.UUID, target SagaLifecycleState) {
	t.Helper()
	switch target {
	case SagaLifecycleCompensating:
		require.True(t, GetCache().TryTransition(ctx, tx, SagaLifecyclePending, SagaLifecycleCompensating))
	case SagaLifecycleFailed:
		require.True(t, GetCache().TryTransition(ctx, tx, SagaLifecyclePending, SagaLifecycleCompensating))
		require.True(t, GetCache().TryTransition(ctx, tx, SagaLifecycleCompensating, SagaLifecycleFailed))
	case SagaLifecycleCompleted:
		require.True(t, GetCache().TryTransition(ctx, tx, SagaLifecyclePending, SagaLifecycleCompleted))
	default:
		t.Fatalf("not a terminal state: %s", target)
	}
}

func TestAcceptEvent_TerminalLifecycleAbsorbs(t *testing.T) {
	for _, terminal := range []SagaLifecycleState{SagaLifecycleCompensating, SagaLifecycleFailed, SagaLifecycleCompleted} {
		t.Run(string(terminal), func(t *testing.T) {
			ResetCache()
			p, hook, ctx := newAcceptEventTestProcessor(t)
			tx := uuid.New()
			s, err := NewBuilder().
				SetTransactionId(tx).
				SetSagaType(InventoryTransaction).
				SetInitiatedBy("test").
				AddStep("award_currency_seller", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 2, CurrencyType: 2, Amount: 100}).
				Build()
			require.NoError(t, err)
			putAcceptEventSaga(t, ctx, s)
			terminalLifecycle(t, ctx, tx, terminal)

			// Pending, action-matching step present — pre-fix this advanced the saga.
			_, ok := p.AcceptEvent(tx, EventKindCashShopWalletUpdated)
			assert.False(t, ok, "terminal lifecycle must absorb the event")

			var entry *logrus.Entry
			for _, e := range hook.AllEntries() {
				if e.Data["reason"] == SkipReasonSagaTerminal {
					entry = e
				}
			}
			require.NotNil(t, entry, "expected saga_terminal skip log")
			assert.Equal(t, tx.String(), entry.Data["transaction_id"])
			assert.Equal(t, string(EventKindCashShopWalletUpdated), entry.Data["event_kind"])
			assert.Equal(t, string(terminal), entry.Data["lifecycle_state"])
			assert.Equal(t, "award_currency_seller", entry.Data["step_id"])
		})
	}
}

func TestAcceptEvent_PendingLifecycleStillAccepts(t *testing.T) {
	ResetCache()
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("award_currency_seller", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 2, CurrencyType: 2, Amount: 100}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	decision, ok := p.AcceptEvent(tx, EventKindCashShopWalletUpdated)
	assert.True(t, ok, "pending lifecycle is unchanged happy path")
	assert.Equal(t, "award_currency_seller", decision.Step.StepId())
}

// A late SUCCESS event for a compensable step routes into CompensateLateStep;
// a late FAILURE event absorbs without compensation (PRD §4.3).
func TestAcceptEvent_TerminalRoutesLateSuccessOnly(t *testing.T) {
	ResetCache()
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := acceptEventTestCtx(t)

	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("award_mesos", Pending, AwardMesos, AwardMesosPayload{CharacterId: 1, Amount: 500}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))
	terminalLifecycle(t, ctx, tx, SagaLifecycleFailed)

	refunds := 0
	charMock := &charactermock.ProcessorMock{
		AwardMesosAndEmitFunc: func(txId uuid.UUID, ch channel.Model, characterId uint32, actorId uint32, actorType string, amount int32, showEffect bool) error {
			refunds++
			assert.Equal(t, int32(-500), amount)
			return nil
		},
	}
	p := NewProcessor(logger, ctx).WithCharacterProcessor(charMock)

	// Failure-outcome kind for the same action: absorb-only.
	_, ok := p.AcceptEvent(tx, EventKindCharacterMesoError)
	assert.False(t, ok)
	assert.Equal(t, 0, refunds, "failure outcome must not compensate")

	// Success-outcome kind: absorb + route into compensation.
	_, ok = p.AcceptEvent(tx, EventKindCharacterMesoChanged)
	assert.False(t, ok)
	assert.Equal(t, 1, refunds, "success outcome must dispatch exactly one inverse")
}

// FR-1.3/FR-1.5: a MAP_CHANGED for character A must not complete a step whose
// payload names character B. WarpToRandomPortal is used here because it
// already accepts EventKindCharacterMapChanged before Task 3 lands.
func TestAcceptEvent_CharacterIdMismatchSkips(t *testing.T) {
	p, hook, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("warp-1", Pending, WarpToRandomPortal, WarpToRandomPortalPayload{
			CharacterId: 100,
			FieldId:     field.Id("0:1:0:00000000-0000-0000-0000-000000000000"),
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	_, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged, ForCharacter(999))
	assert.False(t, ok, "an event for a different character must not complete the step")

	var found bool
	for _, e := range hook.AllEntries() {
		if e.Data["reason"] == SkipReasonCharacterIdMismatch {
			found = true
			assert.Equal(t, uint32(999), e.Data["event_character_id"])
			assert.Equal(t, uint32(100), e.Data["step_character_id"])
		}
	}
	assert.True(t, found, "the skip must be logged with reason character_id_mismatch")
}

// The matching character completes normally.
func TestAcceptEvent_CharacterIdMatchAccepts(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("warp-1", Pending, WarpToRandomPortal, WarpToRandomPortalPayload{
			CharacterId: 100,
			FieldId:     field.Id("0:1:0:00000000-0000-0000-0000-000000000000"),
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	decision, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged, ForCharacter(100))
	require.True(t, ok)
	assert.Equal(t, "warp-1", decision.Step.StepId())
}

// A payload with no character id is unconstrained — every pre-existing
// action's behaviour is preserved.
func TestAcceptEvent_NoCharacterIdInPayloadIsUnconstrained(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("warp-1", Pending, WarpToRandomPortal, struct{ Nothing string }{}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	_, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged, ForCharacter(999))
	assert.True(t, ok, "ExtractCharacterId returns 0 -> unconstrained -> accept")
}

// Calling without the option is unchanged behaviour for the other ~60 sites.
func TestAcceptEvent_NoOptionIsUnconstrained(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("warp-1", Pending, WarpToRandomPortal, WarpToRandomPortalPayload{
			CharacterId: 100,
			FieldId:     field.Id("0:1:0:00000000-0000-0000-0000-000000000000"),
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	_, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged)
	assert.True(t, ok)
}

// FR-4.3: WarpToPortal completes on a MAP_CHANGED carrying the saga tx id.
func TestAcceptEvent_WarpToPortalCompletesOnMapChanged(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("portal-action-warp").
		AddStep("warp-100", Pending, WarpToPortal, WarpToPortalPayload{
			CharacterId: 100,
			WorldId:     0,
			ChannelId:   1,
			MapId:       _map.Id(200090510),
			PortalId:    3,
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	decision, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged, ForCharacter(100))
	require.True(t, ok, "WarpToPortal must accept map_changed (FR-1.1)")
	assert.Equal(t, "warp-100", decision.Step.StepId())
}

// FR-4.3: same for WarpToSavedLocation.
func TestAcceptEvent_WarpToSavedLocationCompletesOnMapChanged(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("portal-action-warp-saved-location").
		AddStep("warp-saved-100-FREE_MARKET", Pending, WarpToSavedLocation, WarpToSavedLocationPayload{
			CharacterId:  100,
			WorldId:      0,
			ChannelId:    1,
			LocationType: "FREE_MARKET",
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	decision, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged, ForCharacter(100))
	require.True(t, ok, "WarpToSavedLocation must accept map_changed (FR-1.1)")
	assert.Equal(t, "warp-saved-100-FREE_MARKET", decision.Step.StepId())
}

// FR-4.5: the same-map case. The old comment claimed no map_changed fires for
// a portal-to-portal warp within one map; atlas-maps
// warp.ProcessorImpl.ChangeMap emits it unconditionally. The acceptance path
// is map-agnostic, so this asserts no same-map special case was introduced.
func TestAcceptEvent_WarpToPortalSameMapCompletes(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	const sameMap = 200090510
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("portal-action-warp").
		AddStep("warp-100", Pending, WarpToPortal, WarpToPortalPayload{
			CharacterId: 100,
			WorldId:     0,
			ChannelId:   1,
			MapId:       _map.Id(sameMap), // target == origin
			PortalId:    3,
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	_, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged, ForCharacter(100))
	assert.True(t, ok, "a same-map warp acknowledges exactly like a cross-map one (FR-4.5)")
}

// FR-4.4 on the newly-accepting action: the party-quest fan-out case this
// whole guard exists for.
func TestAcceptEvent_WarpToPortalRejectsOtherCharacter(t *testing.T) {
	p, hook, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("portal-action-warp").
		AddStep("warp-B", Pending, WarpToPortal, WarpToPortalPayload{
			CharacterId: 200, // character B
			WorldId:     0,
			ChannelId:   1,
			MapId:       _map.Id(200090510),
			PortalId:    3,
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	// character A's arrival must not complete B's step
	_, ok := p.AcceptEvent(tx, EventKindCharacterMapChanged, ForCharacter(100))
	assert.False(t, ok)

	var found bool
	for _, e := range hook.AllEntries() {
		if e.Data["reason"] == SkipReasonCharacterIdMismatch {
			found = true
		}
	}
	assert.True(t, found, "reason must be character_id_mismatch (FR-1.5)")
}
