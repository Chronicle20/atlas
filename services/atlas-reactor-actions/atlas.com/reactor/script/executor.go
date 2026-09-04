package script

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	reactorsaga "atlas-reactor-actions/saga"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/operation"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/ops"
)

// ReactorContext holds context information for reactor operation execution
type ReactorContext struct {
	Field          field.Model
	ReactorId      uint32
	Classification string
	ReactorName    string
	X              int16
	Y              int16
}

// OperationExecutor executes reactor script operations
type OperationExecutor struct {
	l     logrus.FieldLogger
	ctx   context.Context
	sagaP reactorsaga.Processor
}

// NewOperationExecutor creates a new operation executor
func NewOperationExecutor(l logrus.FieldLogger, ctx context.Context) *OperationExecutor {
	return &OperationExecutor{
		l:     l,
		ctx:   ctx,
		sagaP: reactorsaga.NewProcessor(l, ctx),
	}
}

// ExecuteOperation executes a single operation
func (e *OperationExecutor) ExecuteOperation(rc ReactorContext, characterId uint32, op operation.Model) error {
	e.l.Debugf("Executing operation [%s] for character [%d] on reactor [%s]", op.Type(), characterId, rc.Classification)

	switch op.Type() {
	case "drop_items":
		return e.executeDropItems(rc, characterId, op)

	case "spawn_monster":
		return e.executeSpawnMonster(rc, characterId, op)

	case "spray_items":
		return e.executeSprayItems(rc, characterId, op)

	case "weaken_area_boss":
		return e.executeWeakenAreaBoss(rc, characterId, op)

	case "move_environment":
		return e.executeMoveEnvironment(rc, characterId, op)

	case "reset_environment":
		return e.executeResetEnvironment(rc, characterId, op)

	case "kill_all_monsters":
		return e.executeKillAllMonsters(rc, characterId, op)

	case "drop_message":
		return e.executeDropMessage(rc, characterId, op)

	case "update_pq_state":
		return e.executeUpdatePqState(rc, characterId, op)

	case "hit_reactor":
		return e.executeHitReactor(rc, characterId, op)

	case "broadcast_pq_message":
		return e.executeBroadcastPqMessage(rc, characterId, op)

	case "stage_clear_attempt":
		return e.executeStageClearAttempt(rc, characterId, op)

	default:
		e.l.Warnf("Unknown operation type [%s] for character [%d]", op.Type(), characterId)
		return nil
	}
}

// ExecuteOperations executes multiple operations
func (e *OperationExecutor) ExecuteOperations(rc ReactorContext, characterId uint32, ops []operation.Model) error {
	for _, op := range ops {
		if err := e.ExecuteOperation(rc, characterId, op); err != nil {
			return err
		}
	}
	return nil
}

// executeDropItems handles reactor item drops via saga orchestration
// Supports both legacy params (minMeso, maxMeso, mesoRange) and new params (mesoChance, mesoMin, mesoMax, minItems)
func (e *OperationExecutor) executeDropItems(rc ReactorContext, characterId uint32, op operation.Model) error {
	params := op.Params()

	// Parse drop type - defaults to "drop" (simultaneous), can be "spray" (200ms intervals)
	dropType := "drop"
	if v, ok := params["dropType"]; ok {
		dropType = v
	}

	// Parse meso enabled
	mesoEnabled := params["meso"] == "true"

	// Parse meso configuration with backward compatibility
	var mesoChance, mesoMin, mesoMax, minItems uint32 = 1, 1, 1, 0

	// New format: mesoChance
	if v, ok := params["mesoChance"]; ok {
		if parsed, err := strconv.ParseUint(v, 10, 32); err == nil {
			mesoChance = uint32(parsed)
		}
	}

	// New format: mesoMin, fallback to legacy minMeso
	if v, ok := params["mesoMin"]; ok {
		if parsed, err := strconv.ParseUint(v, 10, 32); err == nil {
			mesoMin = uint32(parsed)
		}
	} else if v, ok := params["minMeso"]; ok {
		if parsed, err := strconv.ParseUint(v, 10, 32); err == nil {
			mesoMin = uint32(parsed)
		}
	}

	// New format: mesoMax, fallback to legacy maxMeso
	if v, ok := params["mesoMax"]; ok {
		if parsed, err := strconv.ParseUint(v, 10, 32); err == nil {
			mesoMax = uint32(parsed)
		}
	} else if v, ok := params["maxMeso"]; ok {
		if parsed, err := strconv.ParseUint(v, 10, 32); err == nil {
			mesoMax = uint32(parsed)
		}
	}

	// New format: minItems (minimum guaranteed drops, padded with meso)
	if v, ok := params["minItems"]; ok {
		if parsed, err := strconv.ParseUint(v, 10, 32); err == nil {
			minItems = uint32(parsed)
		}
	}

	e.l.Debugf("Spawning reactor drops: reactor=%s (objectId=%d), map=%d, pos=(%d,%d), char=%d, type=%s, meso=%t, mesoChance=%d, mesoMin=%d, mesoMax=%d, minItems=%d",
		rc.Classification, rc.ReactorId, rc.Field.MapId(), rc.X, rc.Y, characterId, dropType, mesoEnabled, mesoChance, mesoMin, mesoMax, minItems)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("reactor-action-drop").
		AddStep(
			fmt.Sprintf("drop-%s-%d", rc.Classification, characterId),
			saga.Pending,
			saga.SpawnReactorDrops,
			saga.SpawnReactorDropsPayload{
				CharacterId:    characterId,
				WorldId:        rc.Field.WorldId(),
				ChannelId:      rc.Field.ChannelId(),
				MapId:          rc.Field.MapId(),
				Instance:       rc.Field.Instance(),
				ReactorId:      rc.ReactorId,
				Classification: rc.Classification,
				X:              rc.X,
				Y:              rc.Y,
				DropType:       dropType,
				Meso:           mesoEnabled,
				MesoChance:     mesoChance,
				MesoMin:        mesoMin,
				MesoMax:        mesoMax,
				MinItems:       minItems,
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeSpawnMonster spawns monsters at the reactor location
func (e *OperationExecutor) executeSpawnMonster(rc ReactorContext, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(rc.Field).SetPosition(rc.X, rc.Y).Build()
	st, err := ops.SpawnMonster(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}
	p, err := ops.PayloadOf[saga.SpawnMonsterPayload](st)
	if err != nil {
		return err
	}

	e.l.Debugf("Spawning [%d] monster(s) [%d] at reactor [%s] location (%d, %d)", p.Count, p.MonsterId, rc.Classification, p.X, p.Y)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("reactor-action-spawn"),
		fmt.Sprintf("spawn-%s-%d", rc.Classification, p.MonsterId),
	).Build()

	return e.sagaP.Create(s)
}

// executeSprayItems sprays items around the reactor with 200ms delay between drops
// This delegates to executeDropItems with dropType="spray"
func (e *OperationExecutor) executeSprayItems(rc ReactorContext, characterId uint32, op operation.Model) error {
	// Create a modified operation with dropType=spray
	// We inject the spray type into params before delegating
	params := op.Params()
	params["dropType"] = "spray"

	e.l.Debugf("SPRAY_ITEMS: delegating to drop_items with spray type for reactor=%s", rc.Classification)
	return e.executeDropItems(rc, characterId, op)
}

// executeWeakenAreaBoss weakens a boss monster in the area
// TODO: This needs a new saga action for boss weakening
func (e *OperationExecutor) executeWeakenAreaBoss(rc ReactorContext, characterId uint32, op operation.Model) error {
	params := op.Params()

	monsterIdStr, ok := params["monsterId"]
	if !ok {
		return fmt.Errorf("weaken_area_boss operation missing monsterId parameter")
	}

	message := params["message"]

	e.l.Infof("WEAKEN_AREA_BOSS: reactor=%s, map=%d, monsterId=%s, message=%s",
		rc.Classification, rc.Field.MapId(), monsterIdStr, message)

	// TODO: Create saga command for weakening boss
	// This will need to interface with atlas-monsters service

	return nil
}

// executeMoveEnvironment sets the state of one named field object via a saga.
// Parameters: name (required, non-blank), value (required, uint32), kind
// (optional, ENVIRONMENT or OBSTACLE; blank defaults to ENVIRONMENT).
func (e *OperationExecutor) executeMoveEnvironment(rc ReactorContext, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(rc.Field).SetPosition(rc.X, rc.Y).Build()
	st, err := ops.MoveEnvironment(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}

	e.l.Debugf("Moving environment object [%s] at reactor [%s].", op.Params()["name"], rc.Classification)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("reactor-action-move-environment"),
		fmt.Sprintf("move-environment-%s-%s", rc.Classification, op.Params()["name"]),
	).Build()

	return e.sagaP.Create(s)
}

// executeResetEnvironment clears every tracked field object and restores the
// field's objects to their default state. Takes no parameters.
func (e *OperationExecutor) executeResetEnvironment(rc ReactorContext, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(rc.Field).SetPosition(rc.X, rc.Y).Build()
	st, err := ops.ResetEnvironment(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}

	e.l.Debugf("Resetting environment objects in map [%d] at reactor [%s].", rc.Field.MapId(), rc.Classification)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("reactor-action-reset-environment"),
		fmt.Sprintf("reset-environment-%s", rc.Classification),
	).Build()

	return e.sagaP.Create(s)
}

// executeKillAllMonsters kills all monsters in the map
// TODO: This needs a new saga action for mass monster killing
func (e *OperationExecutor) executeKillAllMonsters(rc ReactorContext, characterId uint32, op operation.Model) error {
	e.l.Infof("KILL_ALL_MONSTERS: reactor=%s, map=%d",
		rc.Classification, rc.Field.MapId())

	// TODO: Create saga command for killing all monsters
	// This will need to interface with atlas-monsters service

	return nil
}

// executeDropMessage sends a message to the player
func (e *OperationExecutor) executeDropMessage(rc ReactorContext, characterId uint32, op operation.Model) error {
	t := ops.NewTargetBuilder(rc.Field).SetPosition(rc.X, rc.Y).Build()
	st, err := ops.SendMessage(op.Params(), ops.DirectResolver{}, t, characterId)
	if err != nil {
		return err
	}

	e.l.Debugf("Sending message to character [%d].", characterId)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("reactor-action-message"),
		fmt.Sprintf("message-%d", characterId),
	).Build()

	return e.sagaP.Create(s)
}

// executeUpdatePqState updates party quest custom data via saga orchestration
func (e *OperationExecutor) executeUpdatePqState(rc ReactorContext, characterId uint32, op operation.Model) error {
	params := op.Params()

	// Look up the PQ instance for this character
	pqInstance, err := e.getPqInstanceByCharacter(characterId)
	if err != nil {
		return fmt.Errorf("failed to get PQ instance for character %d: %w", characterId, err)
	}

	// Parse updates (key=value pairs)
	updates := make(map[string]string)
	if v, ok := params["updates"]; ok && v != "" {
		for _, pair := range strings.Split(v, ",") {
			parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(parts) == 2 {
				updates[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Parse increments (comma-separated key names)
	var increments []string
	if v, ok := params["increments"]; ok && v != "" {
		for _, key := range strings.Split(v, ",") {
			if trimmed := strings.TrimSpace(key); trimmed != "" {
				increments = append(increments, trimmed)
			}
		}
	}

	e.l.Debugf("Updating PQ state: instance=%s, updates=%v, increments=%v", pqInstance.Id, updates, increments)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("reactor-action-pq-state").
		AddStep(
			fmt.Sprintf("pq-state-%s-%d", rc.Classification, characterId),
			saga.Pending,
			saga.UpdatePqCustomData,
			saga.UpdatePqCustomDataPayload{
				InstanceId: pqInstance.Id,
				Updates:    updates,
				Increments: increments,
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeHitReactor hits another reactor by name via saga orchestration
func (e *OperationExecutor) executeHitReactor(rc ReactorContext, characterId uint32, op operation.Model) error {
	params := op.Params()

	reactorName, ok := params["reactorName"]
	if !ok {
		return fmt.Errorf("hit_reactor operation missing reactorName parameter")
	}

	e.l.Debugf("Hitting reactor [%s] in map [%d] for character [%d]", reactorName, rc.Field.MapId(), characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("reactor-action-hit-reactor").
		AddStep(
			fmt.Sprintf("hit-%s-%d", reactorName, characterId),
			saga.Pending,
			saga.HitReactor,
			saga.HitReactorPayload{
				WorldId:     rc.Field.WorldId(),
				ChannelId:   rc.Field.ChannelId(),
				MapId:       rc.Field.MapId(),
				Instance:    rc.Field.Instance(),
				CharacterId: characterId,
				ReactorName: reactorName,
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeBroadcastPqMessage broadcasts a message to all PQ members via saga orchestration
func (e *OperationExecutor) executeBroadcastPqMessage(rc ReactorContext, characterId uint32, op operation.Model) error {
	params := op.Params()

	message, ok := params["message"]
	if !ok {
		return fmt.Errorf("broadcast_pq_message operation missing message parameter")
	}

	messageType := "PINK_TEXT"
	if mt, ok := params["type"]; ok {
		switch mt {
		case "5":
			messageType = "PINK_TEXT"
		case "6":
			messageType = "BLUE_TEXT"
		default:
			messageType = mt
		}
	}

	// Look up the PQ instance for this character
	pqInstance, err := e.getPqInstanceByCharacter(characterId)
	if err != nil {
		return fmt.Errorf("failed to get PQ instance for character %d: %w", characterId, err)
	}

	e.l.Debugf("Broadcasting PQ message: instance=%s, type=%s, message=%s", pqInstance.Id, messageType, message)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("reactor-action-pq-broadcast").
		AddStep(
			fmt.Sprintf("pq-broadcast-%s-%d", rc.Classification, characterId),
			saga.Pending,
			saga.BroadcastPqMessage,
			saga.BroadcastPqMessagePayload{
				InstanceId:  pqInstance.Id,
				MessageType: messageType,
				Message:     message,
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeStageClearAttempt triggers a stage clear attempt on the character's party quest instance via saga orchestration
func (e *OperationExecutor) executeStageClearAttempt(rc ReactorContext, characterId uint32, op operation.Model) error {
	pqInstance, err := e.getPqInstanceByCharacter(characterId)
	if err != nil {
		return fmt.Errorf("failed to get PQ instance for character %d: %w", characterId, err)
	}

	t := ops.NewTargetBuilder(rc.Field).SetPosition(rc.X, rc.Y).Build()
	st, err := ops.StageClearAttemptPq(t, characterId, pqInstance.Id)
	if err != nil {
		return err
	}

	e.l.Debugf("Attempting stage clear: instance=%s, reactor=%s, character=%d", pqInstance.Id, rc.Classification, characterId)

	s := st.AppendTo(
		saga.NewBuilder().
			SetSagaType(saga.InventoryTransaction).
			SetInitiatedBy("reactor-action-stage-clear"),
		fmt.Sprintf("stage-clear-%s-%d", rc.Classification, characterId),
	).Build()

	return e.sagaP.Create(s)
}

// getPqInstanceByCharacter queries the party-quests service for the character's PQ instance
func (e *OperationExecutor) getPqInstanceByCharacter(characterId uint32) (pqInstanceRestModel, error) {
	root, err := requests.RootUrlFor(e.ctx, "PARTY_QUESTS")
	if err != nil {
		return pqInstanceRestModel{}, err
	}
	url := fmt.Sprintf(root+"party-quests/instances/character/%d", characterId)
	return requests.GetRequest[pqInstanceRestModel](url)(e.l, e.ctx)
}
