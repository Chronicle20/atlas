package script

import (
	"context"
	"fmt"
	"strconv"

	mapactionsaga "atlas-map-actions/saga"

	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-script-core/operation"
	"github.com/Chronicle20/atlas-script-core/saga"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
	case "unlock_ui":
		// unlock_ui is handled implicitly by enabling character actions after processing
		e.l.Debugf("Unlock UI for character [%d] (handled by consumer).", characterId)
		return nil
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
	params := op.Params()

	path, ok := params["path"]
	if !ok {
		return fmt.Errorf("show_intro operation missing path parameter")
	}

	e.l.Debugf("Showing intro [%s] for character [%d].", path, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-intro").
		AddStep(
			fmt.Sprintf("intro-%d", characterId),
			saga.Pending,
			saga.ShowIntro,
			saga.ShowIntroPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				Path:        path,
			},
		).Build()

	return e.sagaP.Create(s)
}

func (e *OperationExecutor) executeSpawnMonster(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	monsterIdStr, ok := params["monsterId"]
	if !ok {
		return fmt.Errorf("spawn_monster operation missing monsterId parameter")
	}
	monsterId, err := strconv.ParseUint(monsterIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid monsterId [%s]: %w", monsterIdStr, err)
	}

	var x int16 = 0
	if xStr, hasX := params["x"]; hasX {
		xVal, err := strconv.ParseInt(xStr, 10, 16)
		if err != nil {
			return fmt.Errorf("invalid x [%s]: %w", xStr, err)
		}
		x = int16(xVal)
	}

	var y int16 = 0
	if yStr, hasY := params["y"]; hasY {
		yVal, err := strconv.ParseInt(yStr, 10, 16)
		if err != nil {
			return fmt.Errorf("invalid y [%s]: %w", yStr, err)
		}
		y = int16(yVal)
	}

	var count int = 1
	if countStr, hasCount := params["count"]; hasCount {
		countVal, err := strconv.Atoi(countStr)
		if err != nil {
			return fmt.Errorf("invalid count [%s]: %w", countStr, err)
		}
		count = countVal
	}

	// Use event mapId by default, allow override
	mapId := f.MapId()
	if mapIdStr, hasMapId := params["mapId"]; hasMapId {
		mId, err := strconv.ParseUint(mapIdStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid mapId [%s]: %w", mapIdStr, err)
		}
		mapId = _map.Id(mId)
	}

	e.l.Debugf("Spawning monster [%d] at (%d,%d) count [%d] for character [%d].", monsterId, x, y, count, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-spawn").
		AddStep(
			fmt.Sprintf("spawn-%d-%d", characterId, monsterId),
			saga.Pending,
			saga.SpawnMonster,
			saga.SpawnMonsterPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				MapId:       mapId,
				Instance:    uuid.Nil,
				MonsterId:   uint32(monsterId),
				X:           x,
				Y:           y,
				Count:       count,
			},
		).Build()

	return e.sagaP.Create(s)
}

func (e *OperationExecutor) executeDropMessage(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	msg, ok := params["message"]
	if !ok {
		return fmt.Errorf("drop_message operation missing message parameter")
	}

	messageType := "PINK_TEXT"
	if mt, hasType := params["messageType"]; hasType {
		messageType = mt
	}

	e.l.Debugf("Sending message to character [%d]: %s", characterId, msg)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-message").
		AddStep(
			fmt.Sprintf("message-%d", characterId),
			saga.Pending,
			saga.SendMessage,
			saga.SendMessagePayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				MessageType: messageType,
				Message:     msg,
			},
		).Build()

	return e.sagaP.Create(s)
}
