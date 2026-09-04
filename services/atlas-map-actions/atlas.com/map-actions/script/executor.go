package script

import (
	"context"
	"fmt"

	mapactionsaga "atlas-map-actions/saga"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/operation"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/ops"
)

type OperationExecutor struct {
	l     logrus.FieldLogger
	ctx   context.Context
	sagaP mapactionsaga.Processor
}

func NewOperationExecutor(l logrus.FieldLogger, ctx context.Context) *OperationExecutor {
	return &OperationExecutor{
		l:     l,
		ctx:   ctx,
		sagaP: mapactionsaga.NewProcessor(l, ctx),
	}
}

func (e *OperationExecutor) ExecuteOperation(f field.Model, characterId uint32, op operation.Model) error {
	e.l.Debugf("Executing operation [%s] for character [%d].", op.Type(), characterId)

	switch op.Type() {
	case "field_effect":
		return e.executeFieldEffect(f, characterId, op)
	case "show_intro":
		return e.executeShowIntro(f, characterId, op)
	case "spawn_monster":
		return e.executeSpawnMonster(f, characterId, op)
	case "drop_message":
		return e.executeDropMessage(f, characterId, op)
	case "move_environment":
		return e.executeMoveEnvironment(f, characterId, op)
	case "reset_environment":
		return e.executeResetEnvironment(f, characterId, op)
	case "lock_ui":
		return e.executeUiLock(f, characterId, true)
	case "unlock_ui":
		return e.executeUiLock(f, characterId, false)
	default:
		e.l.Warnf("Unknown operation type [%s] for character [%d].", op.Type(), characterId)
		return nil
	}
}

func (e *OperationExecutor) ExecuteOperations(f field.Model, characterId uint32, ops []operation.Model) error {
	for _, op := range ops {
		if err := e.ExecuteOperation(f, characterId, op); err != nil {
			return err
		}
	}
	return nil
}

func (e *OperationExecutor) executeFieldEffect(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	path, ok := params["path"]
	if !ok {
		return fmt.Errorf("field_effect operation missing path parameter")
	}

	e.l.Debugf("Showing field effect [%s] for character [%d].", path, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("field-effect").
		AddStep(
			fmt.Sprintf("effect-%d", characterId),
			saga.Pending,
			saga.FieldEffect,
			saga.FieldEffectPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				Path:        path,
			},
		).Build()

	return e.sagaP.Create(s)
}

func (e *OperationExecutor) executeShowIntro(f field.Model, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).Build()
	st, err := ops.ShowIntro(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}

	e.l.Debugf("Showing intro for character [%d].", characterId)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("map-action-intro"),
		fmt.Sprintf("intro-%d", characterId),
	).Build()

	return e.sagaP.Create(s)
}

func (e *OperationExecutor) executeSpawnMonster(f field.Model, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).Build()
	st, err := ops.SpawnMonster(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}
	p, err := ops.PayloadOf[saga.SpawnMonsterPayload](st)
	if err != nil {
		return err
	}

	e.l.Debugf("Spawning monster [%d] at (%d,%d) count [%d] for character [%d].", p.MonsterId, p.X, p.Y, p.Count, characterId)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("map-action-spawn"),
		fmt.Sprintf("spawn-%d-%d", characterId, p.MonsterId),
	).Build()

	return e.sagaP.Create(s)
}

func (e *OperationExecutor) executeUiLock(f field.Model, characterId uint32, enable bool) error {
	e.l.Debugf("Setting UI lock [%t] for character [%d].", enable, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("ui-lock").
		AddStep(
			fmt.Sprintf("ui-lock-%d", characterId),
			saga.Pending,
			saga.UiLock,
			saga.UiLockPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				Enable:      enable,
			},
		).Build()

	return e.sagaP.Create(s)
}

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
			SetInitiatedBy("map-action-message"),
		fmt.Sprintf("message-%d", characterId),
	).Build()

	return e.sagaP.Create(s)
}

// executeMoveEnvironment sets the state of one named field object via a saga.
// Parameters: name (required, non-blank), value (required, uint32), kind
// (optional, ENVIRONMENT or OBSTACLE; blank defaults to ENVIRONMENT).
func (e *OperationExecutor) executeMoveEnvironment(f field.Model, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).Build()
	st, err := ops.MoveEnvironment(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}

	e.l.Debugf("Moving environment object in map [%d].", f.MapId())

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("map-action-move-environment"),
		fmt.Sprintf("move-environment-%s", op.Params()["name"]),
	).Build()

	return e.sagaP.Create(s)
}

// executeResetEnvironment clears every tracked field object and restores the
// field's objects to their default state. Takes no parameters.
func (e *OperationExecutor) executeResetEnvironment(f field.Model, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(f).Build()
	st, err := ops.ResetEnvironment(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}

	e.l.Debugf("Resetting environment objects in map [%d].", f.MapId())

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("map-action-reset-environment"),
		fmt.Sprintf("reset-environment-%d", f.MapId()),
	).Build()

	return e.sagaP.Create(s)
}
