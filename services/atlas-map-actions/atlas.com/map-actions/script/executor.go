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
	case "update_area_info":
		return e.executeUpdateAreaInfo(f, characterId, op)
	case "show_info":
		return e.executeShowInfo(f, characterId, op)
	case "play_sound":
		return e.executePlaySound(f, characterId, op)
	case "change_music":
		return e.executeChangeMusic(f, characterId, op)
	case "boat_effect":
		return e.executeBoatEffect(f, characterId, op)
	case "clear_skill":
		return e.executeClearSkill(f, characterId, op)
	case "warp_to_map":
		return e.executeWarpToMap(f, characterId, op)
	case "spawn_npc":
		return e.executeSpawnNpc(f, characterId, op)
	case "clear_drops":
		return e.executeClearDrops(f, characterId, op)
	case "reset_reactors":
		return e.executeResetReactors(f, characterId, op)
	case "shuffle_reactors":
		return e.executeShuffleReactors(f, characterId, op)
	case "reset_field":
		return e.executeResetField(f, characterId, op)
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

// executeClearSkill removes a skill from the character outright. Cosmic's
// teachSkill(id, -1, 0, -1) reaches Character.changeSkillLevel's newLevel <= -1
// branch, which deletes the skill row rather than changing its level
// (task-290 G13).
func (e *OperationExecutor) executeClearSkill(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	skillIdStr, ok := params["skillId"]
	if !ok {
		return fmt.Errorf("clear_skill operation missing skillId parameter")
	}
	skillId, err := strconv.ParseUint(skillIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid skillId [%s]: %w", skillIdStr, err)
	}

	e.l.Debugf("Clearing skill [%d] for character [%d].", skillId, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-clear-skill").
		AddStep(
			fmt.Sprintf("clear-skill-%d-%d", characterId, skillId),
			saga.Pending,
			saga.ClearSkill,
			saga.ClearSkillPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				SkillId:     uint32(skillId),
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

// executeSpawnNpc places a scripted NPC on the current field, mirroring
// Cosmic's AbstractPlayerInteraction.spawnNpc. Unlike spawn_monster, x/y are
// required rather than defaulting to 0: every G2 script that places an NPC
// gives it an explicit Point (task-290 G2).
func (e *OperationExecutor) executeSpawnNpc(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	npcIdStr, ok := params["npcId"]
	if !ok {
		return fmt.Errorf("spawn_npc operation requires npcId parameter")
	}
	npcId, err := strconv.ParseUint(npcIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid npcId [%s]: %w", npcIdStr, err)
	}

	xStr, hasX := params["x"]
	if !hasX {
		return fmt.Errorf("spawn_npc operation requires x parameter")
	}
	xVal, err := strconv.ParseInt(xStr, 10, 16)
	if err != nil {
		return fmt.Errorf("invalid x [%s]: %w", xStr, err)
	}
	x := int16(xVal)

	yStr, hasY := params["y"]
	if !hasY {
		return fmt.Errorf("spawn_npc operation requires y parameter")
	}
	yVal, err := strconv.ParseInt(yStr, 10, 16)
	if err != nil {
		return fmt.Errorf("invalid y [%s]: %w", yStr, err)
	}
	y := int16(yVal)

	var spawnIfAbsent bool
	if s, has := params["spawnIfAbsent"]; has {
		parsed, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("invalid spawnIfAbsent [%s]: %w", s, err)
		}
		spawnIfAbsent = parsed
	}

	e.l.Debugf("Spawning npc [%d] at (%d,%d) for character [%d].", npcId, x, y, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-spawn-npc").
		AddStep(
			fmt.Sprintf("spawn-npc-%d-%d", characterId, npcId),
			saga.Pending,
			saga.SpawnNpc,
			saga.SpawnNpcPayload{
				CharacterId:   characterId,
				WorldId:       f.WorldId(),
				ChannelId:     f.ChannelId(),
				MapId:         f.MapId(),
				Instance:      f.Instance(),
				NpcId:         uint32(npcId),
				X:             x,
				Y:             y,
				SpawnIfAbsent: spawnIfAbsent,
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeClearDrops removes every drop from the current field, mirroring
// Cosmic's no-arg MapleMap.clearDrops(): whole-map, not owner-filtered
// (task-290 G5). It takes no params.
func (e *OperationExecutor) executeClearDrops(f field.Model, characterId uint32, op operation.Model) error {
	e.l.Debugf("Clearing drops on field for character [%d].", characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-clear-drops").
		AddStep(
			fmt.Sprintf("clear-drops-%d", characterId),
			saga.Pending,
			saga.ClearDrops,
			saga.ClearDropsPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				MapId:       f.MapId(),
				Instance:    f.Instance(),
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeResetReactors resets reactors on the current field to state 0,
// mirroring Cosmic's MapleMap.resetReactors(List<Reactor>) (MapleMap.java:1563).
// The optional minState param mirrors 926120300.js's getInactiveReactors
// filter (state >= 7) computed in script -- there is no state-filtered Java
// overload, so this is one operation with an optional minimum-state filter,
// not two.
func (e *OperationExecutor) executeResetReactors(f field.Model, characterId uint32, op operation.Model) error {
	e.l.Debugf("Resetting reactors on field for character [%d].", characterId)

	params := op.Params()
	var minState *int8
	if s, has := params["minState"]; has {
		parsed, err := strconv.ParseInt(s, 10, 8)
		if err != nil {
			return fmt.Errorf("invalid minState [%s]: %w", s, err)
		}
		v := int8(parsed)
		minState = &v
	}

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-reset-reactors").
		AddStep(
			fmt.Sprintf("reset-reactors-%d", characterId),
			saga.Pending,
			saga.ResetReactors,
			saga.ResetReactorsPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				MapId:       f.MapId(),
				Instance:    f.Instance(),
				MinState:    minState,
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeShuffleReactors randomly permutes the positions of every reactor on
// the current field, mirroring Cosmic's MapleMap.shuffleReactors()
// (MapleMap.java:1580). It takes no params.
func (e *OperationExecutor) executeShuffleReactors(f field.Model, characterId uint32, op operation.Model) error {
	e.l.Debugf("Shuffling reactors on field for character [%d].", characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-shuffle-reactors").
		AddStep(
			fmt.Sprintf("shuffle-reactors-%d", characterId),
			saga.Pending,
			saga.ShuffleReactors,
			saga.ShuffleReactorsPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				MapId:       f.MapId(),
				Instance:    f.Instance(),
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeResetField clears the current field's monsters and restores its
// spawn points, mirroring Cosmic's MapleMap.resetPQ(difficulty)
// (MapleMap.java:3962-3975). The optional difficulty param defaults to 1
// (every G5 script passes 1 today).
func (e *OperationExecutor) executeResetField(f field.Model, characterId uint32, op operation.Model) error {
	e.l.Debugf("Resetting field for character [%d].", characterId)

	difficulty := 1
	params := op.Params()
	if s, has := params["difficulty"]; has {
		parsed, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("invalid difficulty [%s]: %w", s, err)
		}
		difficulty = parsed
	}

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-reset-field").
		AddStep(
			fmt.Sprintf("reset-field-%d", characterId),
			saga.Pending,
			saga.ResetField,
			saga.ResetFieldPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				MapId:       f.MapId(),
				Instance:    f.Instance(),
				Difficulty:  difficulty,
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

func (e *OperationExecutor) executeUpdateAreaInfo(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	areaStr, ok := params["area"]
	if !ok {
		return fmt.Errorf("update_area_info operation missing area parameter")
	}
	area, err := strconv.ParseUint(areaStr, 10, 16)
	if err != nil {
		return fmt.Errorf("invalid area [%s]: %w", areaStr, err)
	}

	info, ok := params["info"]
	if !ok {
		return fmt.Errorf("update_area_info operation missing info parameter")
	}

	e.l.Debugf("Updating area info [%d] for character [%d].", area, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-area-info").
		AddStep(
			fmt.Sprintf("area-info-%d-%d", characterId, area),
			saga.Pending,
			saga.UpdateAreaInfo,
			saga.UpdateAreaInfoPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				Area:        uint16(area),
				Info:        info,
			},
		).Build()

	return e.sagaP.Create(s)
}

func (e *OperationExecutor) executeShowInfo(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	path, ok := params["path"]
	if !ok {
		return fmt.Errorf("show_info operation missing path parameter")
	}

	e.l.Debugf("Showing info [%s] for character [%d].", path, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-show-info").
		AddStep(
			fmt.Sprintf("show-info-%d", characterId),
			saga.Pending,
			saga.ShowInfo,
			saga.ShowInfoPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				Path:        path,
			},
		).Build()

	return e.sagaP.Create(s)
}

func (e *OperationExecutor) executePlaySound(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	path, ok := params["path"]
	if !ok {
		return fmt.Errorf("play_sound operation missing path parameter")
	}

	e.l.Debugf("Playing sound [%s] for character [%d].", path, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-play-sound").
		AddStep(
			fmt.Sprintf("play-sound-%d", characterId),
			saga.Pending,
			saga.PlaySound,
			saga.PlaySoundPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				Path:        path,
			},
		).Build()

	return e.sagaP.Create(s)
}

func (e *OperationExecutor) executeChangeMusic(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	path, ok := params["path"]
	if !ok {
		return fmt.Errorf("change_music operation missing path parameter")
	}

	e.l.Debugf("Changing music to [%s] for character [%d].", path, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-change-music").
		AddStep(
			fmt.Sprintf("change-music-%d", characterId),
			saga.Pending,
			saga.ChangeMusic,
			saga.ChangeMusicPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				Path:        path,
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeWarpToMap warps a character to a map without naming a portal.
// Cosmic's warpAhead(mapId) resolves getRandomPlayerSpawnpoint() on the
// destination map — a random player spawn point, not a portal — so this
// action names only the destination map and lets the saga/character layer
// pick the spawn point (task-290 G1a).
func (e *OperationExecutor) executeWarpToMap(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	mapIdStr, ok := params["mapId"]
	if !ok {
		return fmt.Errorf("warp_to_map operation missing mapId parameter")
	}
	mapId, err := strconv.ParseUint(mapIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid mapId [%s]: %w", mapIdStr, err)
	}

	e.l.Debugf("Warping character [%d] to map [%d].", characterId, mapId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-warp").
		AddStep(
			fmt.Sprintf("warp-%d-%d", characterId, mapId),
			saga.Pending,
			saga.WarpToMap,
			saga.WarpToMapPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				MapId:       _map.Id(mapId),
			},
		).Build()

	return e.sagaP.Create(s)
}

func (e *OperationExecutor) executeBoatEffect(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	showStr, ok := params["show"]
	if !ok {
		return fmt.Errorf("boat_effect operation missing show parameter")
	}

	show, err := strconv.ParseBool(showStr)
	if err != nil {
		return fmt.Errorf("invalid show [%s]: %w", showStr, err)
	}

	e.l.Debugf("Setting boat effect show [%t] for character [%d].", show, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-boat-effect").
		AddStep(
			fmt.Sprintf("boat-effect-%d", characterId),
			saga.Pending,
			saga.BoatEffect,
			saga.BoatEffectPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				Show:        show,
			},
		).Build()

	return e.sagaP.Create(s)
}
