package script

import (
	"atlas-portal-actions/action"
	"context"
	"fmt"
	"strconv"
	"time"

	portalsaga "atlas-portal-actions/saga"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/operation"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/ops"
)

const (
	// warpSagaTimeout bounds a portal warp saga. The default is 30s
	// (orchestrator DefaultSagaTimeout); against a ~300ms observed end-to-end
	// warp that is 100x, and it is how long a player whose warp did not land
	// would stay frozen now that the outcome no longer unlocks them
	// eagerly (task-184 FR-2.6). start_instance_transport deliberately keeps
	// the 30s default — it does strictly more work.
	warpSagaTimeout = 5 * time.Second

	// pendingActionTTL bounds the registry entry backing a suppressed unlock.
	// It must exceed warpSagaTimeout by a wide margin so handleStatusEventFailed
	// can still find the entry when the timeout fires.
	pendingActionTTL = 60 * time.Second
)

// OperationExecutor executes portal script operations
type OperationExecutor struct {
	l     logrus.FieldLogger
	ctx   context.Context
	sagaP portalsaga.Processor
}

// NewOperationExecutor creates a new operation executor
func NewOperationExecutor(l logrus.FieldLogger, ctx context.Context) *OperationExecutor {
	return &OperationExecutor{
		l:     l,
		ctx:   ctx,
		sagaP: portalsaga.NewProcessor(l, ctx),
	}
}

// newOperationExecutorWithSaga builds an executor over an injected saga
// processor. Used by tests to observe dispatched sagas without touching Kafka.
func newOperationExecutorWithSaga(l logrus.FieldLogger, ctx context.Context, sagaP portalsaga.Processor) *OperationExecutor {
	return &OperationExecutor{l: l, ctx: ctx, sagaP: sagaP}
}

// ExecuteOperation executes a single operation.
// portalId is the numeric ID of the current portal (for operations like block_portal).
// Dispatch goes through opTable, which is also the classification authority for
// whether an operation moves the character — see optable.go.
func (e *OperationExecutor) ExecuteOperation(f field.Model, characterId uint32, portalId uint32, op operation.Model) error {
	e.l.Debugf("Executing operation [%s] for character [%d]", op.Type(), characterId)

	def, ok := opTable[op.Type()]
	if !ok {
		e.l.Warnf("Unknown operation type [%s] for character [%d]", op.Type(), characterId)
		return nil
	}
	return def.run(e, f, characterId, portalId, op)
}

// ExecuteOperations executes multiple operations in order, stopping at the
// first error.
//
// portalId is the numeric ID of the current portal (for operations like
// block_portal).
//
// movedCharacter reports whether at least one MOVING operation was
// SUCCESSFULLY dispatched (its saga was created). The caller uses this to
// decide whether the client is already going to be unlocked by the resulting
// SET_FIELD — see consumer.go and task-184 prd.md §1.1.
//
// The distinction between "declared" and "successfully dispatched" is
// load-bearing: a warp that failed before creating its saga has no saga to
// fail and release the player, so the caller MUST still unlock them.
func (e *OperationExecutor) ExecuteOperations(f field.Model, characterId uint32, portalId uint32, ops []operation.Model) (bool, error) {
	movedCharacter := false
	for _, op := range ops {
		if err := e.ExecuteOperation(f, characterId, portalId, op); err != nil {
			return movedCharacter, err
		}
		if IsMovingOperation(op.Type()) {
			movedCharacter = true
		}
	}
	return movedCharacter, nil
}

// executePlayPortalSound sends a saga to play portal sound effect
func (e *OperationExecutor) executePlayPortalSound(f field.Model, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).Build()
	st, err := ops.PlayPortalSound(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}

	e.l.Debugf("Play portal sound for character [%d]", characterId)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("portal-action-sound"),
		fmt.Sprintf("sound-%d", characterId),
	).Build()

	return e.sagaP.Create(s)
}

// executeWarp warps the character to a new location
func (e *OperationExecutor) executeWarp(f field.Model, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).Build()
	st, err := ops.WarpToPortal(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}
	p, err := ops.PayloadOf[saga.WarpToPortalPayload](st)
	if err != nil {
		return err
	}

	e.l.Debugf("Warping character [%d] to map [%d] portal [%d/%s]", characterId, p.MapId, p.PortalId, p.PortalName)

	// The transaction id is minted here so the pending action can be registered
	// under it. If this warp is suppressed from unlocking the client
	// (consumer.go, task-184 FR-2.3), this registration is what lets
	// handleStatusEventFailed release the player when the warp does not land.
	sagaId := uuid.New()
	action.GetRegistry().AddWithTTL(e.l, e.ctx, sagaId, action.PendingAction{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		Kind:        action.KindWarp,
	}, pendingActionTTL)

	s := st.AppendTo(
		saga.NewBuilder().
			SetTransactionId(sagaId).
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("portal-action-warp").
			SetTimeout(warpSagaTimeout),
		fmt.Sprintf("warp-%d", characterId),
	).Build()

	return e.sagaP.Create(s)
}

// executeDropMessage sends a message to the player
func (e *OperationExecutor) executeDropMessage(f field.Model, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).Build()
	st, err := ops.SendMessage(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}

	e.l.Debugf("Sending message to character [%d].", characterId)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("portal-action-message"),
		fmt.Sprintf("message-%d", characterId),
	).Build()

	return e.sagaP.Create(s)
}

// executeShowHint sends a hint message to the player
func (e *OperationExecutor) executeShowHint(f field.Model, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).Build()
	st, err := ops.ShowHint(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}
	p, err := ops.PayloadOf[saga.ShowHintPayload](st)
	if err != nil {
		return err
	}

	e.l.Debugf("Showing hint to character [%d]: %s (width=%d, height=%d)", characterId, p.Hint, p.Width, p.Height)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("portal-action-hint"),
		fmt.Sprintf("hint-%d", characterId),
	).Build()

	return e.sagaP.Create(s)
}

// executeBlockPortal sends a saga to block a portal for a character
// Uses the current portal's mapId and portalId by default, but can be overridden via params
func (e *OperationExecutor) executeBlockPortal(f field.Model, characterId uint32, currentPortalId uint32, op operation.Model) error {
	params := op.Params()

	// Use current map by default, allow override via params
	mapId := uint32(f.MapId())
	if mapIdStr, ok := params["mapId"]; ok {
		parsed, err := strconv.ParseUint(mapIdStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid mapId [%s]: %w", mapIdStr, err)
		}
		mapId = uint32(parsed)
	}

	// Use current portal by default, allow override via params
	portalId := currentPortalId
	if portalIdStr, ok := params["portalId"]; ok {
		parsed, err := strconv.ParseUint(portalIdStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid portalId [%s]: %w", portalIdStr, err)
		}
		portalId = uint32(parsed)
	}

	e.l.Debugf("Blocking portal [%d] in map [%d] for character [%d]", portalId, mapId, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("portal-action-block").
		AddStep(
			fmt.Sprintf("block-%d-%d-%d", characterId, mapId, portalId),
			saga.Pending,
			saga.BlockPortal,
			saga.BlockPortalPayload{
				CharacterId: characterId,
				MapId:       _map.Id(mapId),
				PortalId:    portalId,
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeCreateSkill creates a new skill for the character
func (e *OperationExecutor) executeCreateSkill(characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(field.Model{}).Build()
	st, err := ops.CreateSkill(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}
	p, err := ops.PayloadOf[saga.CreateSkillPayload](st)
	if err != nil {
		return err
	}

	e.l.Debugf("Creating skill [%d] for character [%d] (level=%d, masterLevel=%d)", p.SkillId, characterId, p.Level, p.MasterLevel)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("portal-action-create-skill"),
		fmt.Sprintf("create-skill-%d-%d", characterId, p.SkillId),
	).Build()

	return e.sagaP.Create(s)
}

// executeUpdateSkill updates an existing skill for the character
func (e *OperationExecutor) executeUpdateSkill(characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(field.Model{}).Build()
	st, err := ops.UpdateSkill(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}
	p, err := ops.PayloadOf[saga.UpdateSkillPayload](st)
	if err != nil {
		return err
	}

	e.l.Debugf("Updating skill [%d] for character [%d] (level=%d, masterLevel=%d)", p.SkillId, characterId, p.Level, p.MasterLevel)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("portal-action-update-skill"),
		fmt.Sprintf("update-skill-%d-%d", characterId, p.SkillId),
	).Build()

	return e.sagaP.Create(s)
}

// executeStartInstanceTransport starts an instance-based transport for the character
func (e *OperationExecutor) executeStartInstanceTransport(f field.Model, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).Build()
	st, err := ops.StartInstanceTransport(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}
	p, err := ops.PayloadOf[saga.StartInstanceTransportPayload](st)
	if err != nil {
		return err
	}

	// Get optional failure message for when transport fails. The shared op
	// does not consume this — it feeds only the registry entry below.
	failureMessage := op.Params()["failureMessage"]

	e.l.Debugf("Starting instance transport [%s] for character [%d]", p.RouteName, characterId)

	// Generate saga ID upfront so we can track it
	sagaId := uuid.New()

	// Register pending action for saga failure handling
	action.GetRegistry().Add(e.l, e.ctx, sagaId, action.PendingAction{
		CharacterId:    characterId,
		WorldId:        f.WorldId(),
		ChannelId:      f.ChannelId(),
		FailureMessage: failureMessage,
		Kind:           action.KindTransport,
	})

	s := st.AppendTo(
		saga.NewBuilder().
			SetTransactionId(sagaId).
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("portal-action-transport"),
		fmt.Sprintf("transport-%d-%s", characterId, p.RouteName),
	).Build()

	return e.sagaP.Create(s)
}

// executeApplyConsumableEffect applies consumable effects (buffs) to the character
func (e *OperationExecutor) executeApplyConsumableEffect(f field.Model, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).Build()
	st, err := ops.ApplyConsumableEffect(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}
	p, err := ops.PayloadOf[saga.ApplyConsumableEffectPayload](st)
	if err != nil {
		return err
	}

	e.l.Debugf("Applying consumable effect [%d] for character [%d]", p.ItemId, characterId)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("portal-action-apply-effect"),
		fmt.Sprintf("apply-effect-%d-%d", characterId, p.ItemId),
	).Build()

	return e.sagaP.Create(s)
}

// executeSaveLocation saves the character's current location for later retrieval
func (e *OperationExecutor) executeSaveLocation(f field.Model, characterId uint32, portalId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).SetPortalId(portalId).Build()
	st, err := ops.SaveLocation(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}
	p, err := ops.PayloadOf[saga.SaveLocationPayload](st)
	if err != nil {
		return err
	}

	e.l.Debugf("Saving location [%s] for character [%d] (map=%d, portal=%d)", p.LocationType, characterId, p.MapId, p.PortalId)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("portal-action-save-location"),
		fmt.Sprintf("save-location-%d-%s", characterId, p.LocationType),
	).Build()

	return e.sagaP.Create(s)
}

// executeWarpToSavedLocation warps the character back to a previously saved location
func (e *OperationExecutor) executeWarpToSavedLocation(f field.Model, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).Build()
	st, err := ops.WarpToSavedLocation(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}
	p, err := ops.PayloadOf[saga.WarpToSavedLocationPayload](st)
	if err != nil {
		return err
	}

	e.l.Debugf("Warping character [%d] to saved location [%s]", characterId, p.LocationType)

	sagaId := uuid.New()
	action.GetRegistry().AddWithTTL(e.l, e.ctx, sagaId, action.PendingAction{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		Kind:        action.KindWarp,
	}, pendingActionTTL)

	s := st.AppendTo(
		saga.NewBuilder().
			SetTransactionId(sagaId).
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("portal-action-warp-saved-location").
			SetTimeout(warpSagaTimeout),
		fmt.Sprintf("warp-saved-%d-%s", characterId, p.LocationType),
	).Build()

	return e.sagaP.Create(s)
}

// executeCancelConsumableEffect cancels consumable effects (buffs) on the character
func (e *OperationExecutor) executeCancelConsumableEffect(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	itemIdStr, ok := params["itemId"]
	if !ok {
		return fmt.Errorf("cancel_consumable_effect operation missing itemId parameter")
	}

	itemId, err := strconv.ParseUint(itemIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid itemId [%s]: %w", itemIdStr, err)
	}

	e.l.Debugf("Cancelling consumable effect [%d] for character [%d]", itemId, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("portal-action-cancel-effect").
		AddStep(
			fmt.Sprintf("cancel-effect-%d-%d", characterId, itemId),
			saga.Pending,
			saga.CancelConsumableEffect,
			saga.CancelConsumableEffectPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				ItemId:      uint32(itemId),
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeStartQuest dispatches a saga to start a quest for the character.
// questId is required. npcId is optional and defaults to 0 since portals have no NPC context.
func (e *OperationExecutor) executeStartQuest(f field.Model, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).Build()
	st, err := ops.StartQuest(op.Params(), ops.DirectResolver{}, t, characterId, ops.QuestDefaults{})
	if err != nil {
		return err
	}
	p, err := ops.PayloadOf[saga.StartQuestPayload](st)
	if err != nil {
		return err
	}

	e.l.Debugf("Starting quest [%d] for character [%d] (npcId=%d)", p.QuestId, characterId, p.NpcId)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("portal-action-start-quest"),
		fmt.Sprintf("start-quest-%d-%d", characterId, p.QuestId),
	).Build()

	return e.sagaP.Create(s)
}
