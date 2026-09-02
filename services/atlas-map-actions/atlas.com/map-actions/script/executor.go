package script

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	mapactionsaga "atlas-map-actions/saga"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/operation"
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
	case "lock_ui":
		return e.executeUiLock(f, characterId, true)
	case "unlock_ui":
		return e.executeUiLock(f, characterId, false)
	case "set_quest_progress":
		return e.executeSetQuestProgress(f, characterId, op)
	case "start_quest":
		return e.executeStartQuest(f, characterId, op)
	case "open_npc":
		return e.executeOpenNpc(f, characterId, op)
	default:
		// FR-3.0 / design D3: an unknown operation is a seed defect, not a
		// no-op. The schema's operation enum is generated from this switch
		// (tools/gen-map-action-schema.sh), so a document cannot name an
		// operation this switch lacks without failing catalog-lint first.
		return fmt.Errorf("unknown operation type [%s]", op.Type())
	}
}

// ExecuteOperations runs a rule's operations in document order and stops at the
// first error. An unknown operation therefore suppresses every operation after
// it in the same rule (design D3). That is deliberate: a half-applied cutscene
// or a spawn without its announcement is worse than a loud failure at map entry.
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

	monsterIdStr, hasSingle := params["monsterId"]
	monsterIdsStr, hasList := params["monsterIds"]
	if hasList && strings.TrimSpace(monsterIdsStr) == "" {
		hasList = false
	}
	if hasSingle == hasList {
		return fmt.Errorf("spawn_monster operation requires exactly one of monsterId or monsterIds")
	}

	var monsterId uint64
	if hasSingle {
		parsed, err := strconv.ParseUint(monsterIdStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid monsterId [%s]: %w", monsterIdStr, err)
		}
		monsterId = parsed
	} else {
		// Design D9 (G7): pepeking_effect picks one of three monsters
		// uniformly. Randomizing here keeps the rule engine stateless — a
		// `random` rule selector would need a non-deterministic condition
		// type the aggregator has no concept of.
		candidates := strings.Split(monsterIdsStr, ",")
		ids := make([]uint64, 0, len(candidates))
		for _, c := range candidates {
			c = strings.TrimSpace(c)
			parsed, err := strconv.ParseUint(c, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid monsterIds entry [%s]: %w", c, err)
			}
			ids = append(ids, parsed)
		}
		monsterId = ids[rand.Intn(len(ids))]
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

	// FR-2.1: Cosmic guards every map spawn with getMonsterById(id) != null.
	// The guard itself is decided in atlas-monsters against its own registry
	// (design D5/F6) — a read-then-write here would be a cross-service TOCTOU
	// two simultaneous map entries would both pass.
	var spawnIfAbsent bool
	if s, has := params["spawnIfAbsent"]; has {
		parsed, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("invalid spawnIfAbsent [%s]: %w", s, err)
		}
		spawnIfAbsent = parsed
	}

	// Use event mapId by default, allow override
	mapId := f.MapId()
	instance := f.Instance()
	if mapIdStr, hasMapId := params["mapId"]; hasMapId {
		mId, err := strconv.ParseUint(mapIdStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid mapId [%s]: %w", mapIdStr, err)
		}
		mapId = _map.Id(mId)
		// A foreign map id cannot be scoped by this field's instance UUID, so the
		// spawn is sent unscoped, matching pre-F3 behavior for the override path.
		instance = uuid.Nil
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
				CharacterId:   characterId,
				WorldId:       f.WorldId(),
				ChannelId:     f.ChannelId(),
				MapId:         mapId,
				Instance:      instance,
				MonsterId:     uint32(monsterId),
				X:             x,
				Y:             y,
				Count:         count,
				SpawnIfAbsent: spawnIfAbsent,
			},
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

func (e *OperationExecutor) executeSetQuestProgress(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	questIdStr, ok := params["questId"]
	if !ok {
		return fmt.Errorf("set_quest_progress operation missing questId parameter")
	}
	questId, err := strconv.ParseUint(questIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid questId [%s]: %w", questIdStr, err)
	}

	infoNumberStr, ok := params["infoNumber"]
	if !ok {
		return fmt.Errorf("set_quest_progress operation missing infoNumber parameter")
	}
	infoNumber, err := strconv.ParseUint(infoNumberStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid infoNumber [%s]: %w", infoNumberStr, err)
	}

	progress, ok := params["progress"]
	if !ok {
		return fmt.Errorf("set_quest_progress operation missing progress parameter")
	}

	e.l.Debugf("Setting quest [%d] progress [%d]=[%s] for character [%d].", questId, infoNumber, progress, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-quest-progress").
		AddStep(
			fmt.Sprintf("quest-progress-%d-%d", characterId, questId),
			saga.Pending,
			saga.SetQuestProgress,
			saga.SetQuestProgressPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				QuestId:     uint32(questId),
				InfoNumber:  uint32(infoNumber),
				Progress:    progress,
			},
		).Build()

	return e.sagaP.Create(s)
}

func (e *OperationExecutor) executeStartQuest(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	questIdStr, ok := params["questId"]
	if !ok {
		return fmt.Errorf("start_quest operation missing questId parameter")
	}
	questId, err := strconv.ParseUint(questIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid questId [%s]: %w", questIdStr, err)
	}

	var npcId uint64
	if npcIdStr, has := params["npcId"]; has {
		npcId, err = strconv.ParseUint(npcIdStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid npcId [%s]: %w", npcIdStr, err)
		}
	}

	e.l.Debugf("Force-starting quest [%d] for character [%d].", questId, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-start-quest").
		AddStep(
			fmt.Sprintf("start-quest-%d-%d", characterId, questId),
			saga.Pending,
			saga.StartQuest,
			saga.StartQuestPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				QuestId:     uint32(questId),
				NpcId:       uint32(npcId),
			},
		).Build()

	return e.sagaP.Create(s)
}

func (e *OperationExecutor) executeOpenNpc(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	npcIdStr, ok := params["npcId"]
	if !ok {
		return fmt.Errorf("open_npc operation missing npcId parameter")
	}
	npcId, err := strconv.ParseUint(npcIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid npcId [%s]: %w", npcIdStr, err)
	}

	e.l.Debugf("Opening NPC [%d] conversation for character [%d].", npcId, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-open-npc").
		AddStep(
			fmt.Sprintf("open-npc-%d-%d", characterId, npcId),
			saga.Pending,
			saga.StartNpcConversation,
			saga.StartNpcConversationPayload{
				CharacterId: characterId,
				// AccountId is deliberately 0: handleStartNpcConversation does
				// read it (saga-orchestrator/saga/producer.go, propagated into
				// npc.Command.Body.AccountId), but the map-enter command this
				// service consumes (script/consumer.go's enterBody) carries no
				// account id at all -- the enter command is published by
				// atlas-channel without one. Threading a real value would mean
				// changing that cross-service kafka contract, which is outside
				// this task's scope; see task-C2 report.
				AccountId:     0,
				NpcTemplateId: uint32(npcId),
				WorldId:       f.WorldId(),
				ChannelId:     f.ChannelId(),
				MapId:         f.MapId(),
				Instance:      f.Instance(),
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
