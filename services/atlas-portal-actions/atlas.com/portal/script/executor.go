package script

import (
	"context"
	"fmt"
	"strconv"
	"time"

	portalsaga "atlas-portal-actions/saga"

	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-script-core/operation"
	"github.com/Chronicle20/atlas-script-core/saga"
	"github.com/sirupsen/logrus"
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

// ExecuteOperation executes a single operation
// portalId is the numeric ID of the current portal (for operations like block_portal)
func (e *OperationExecutor) ExecuteOperation(f field.Model, characterId uint32, portalId uint32, op operation.Model) error {
	e.l.Debugf("Executing operation [%s] for character [%d]", op.Type(), characterId)

	switch op.Type() {
	case "play_portal_sound":
		return e.executePlayPortalSound(f, characterId, op)

	case "warp":
		return e.executeWarp(f, characterId, op)

	case "drop_message":
		return e.executeDropMessage(f, characterId, op)

	case "show_hint":
		return e.executeShowHint(f, characterId, op)

	case "block_portal":
		return e.executeBlockPortal(f, characterId, portalId, op)

	case "create_skill":
		return e.executeCreateSkill(characterId, op)

	case "update_skill":
		return e.executeUpdateSkill(characterId, op)

	default:
		e.l.Warnf("Unknown operation type [%s] for character [%d]", op.Type(), characterId)
		return nil
	}
}

// ExecuteOperations executes multiple operations
// portalId is the numeric ID of the current portal (for operations like block_portal)
func (e *OperationExecutor) ExecuteOperations(f field.Model, characterId uint32, portalId uint32, ops []operation.Model) error {
	for _, op := range ops {
		if err := e.ExecuteOperation(f, characterId, portalId, op); err != nil {
			return err
		}
	}
	return nil
}

// executePlayPortalSound sends a saga to play portal sound effect
func (e *OperationExecutor) executePlayPortalSound(f field.Model, characterId uint32, op operation.Model) error {
	e.l.Debugf("Play portal sound for character [%d]", characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("portal-action-sound").
		AddStep(
			fmt.Sprintf("sound-%d", characterId),
			saga.Pending,
			saga.PlayPortalSound,
			saga.PlayPortalSoundPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeWarp warps the character to a new location
func (e *OperationExecutor) executeWarp(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	mapIdStr, ok := params["mapId"]
	if !ok {
		return fmt.Errorf("warp operation missing mapId parameter")
	}

	mapId, err := strconv.ParseUint(mapIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid mapId [%s]: %w", mapIdStr, err)
	}

	var portalId uint32 = 0
	if portalIdStr, hasPortalId := params["portalId"]; hasPortalId {
		pId, err := strconv.ParseUint(portalIdStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid portalId [%s]: %w", portalIdStr, err)
		}
		portalId = uint32(pId)
	}

	portalName := params["portalName"]

	e.l.Debugf("Warping character [%d] to map [%d] portal [%d/%s]", characterId, mapId, portalId, portalName)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("portal-action-warp").
		AddStep(
			fmt.Sprintf("warp-%d", characterId),
			saga.Pending,
			saga.WarpToPortal,
			saga.WarpToPortalPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				MapId:       _map.Id(mapId),
				PortalId:    portalId,
				PortalName:  portalName,
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeDropMessage sends a message to the player
func (e *OperationExecutor) executeDropMessage(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	message, ok := params["message"]
	if !ok {
		return fmt.Errorf("drop_message operation missing message parameter")
	}

	messageType := "PINK_TEXT"
	if mt, hasType := params["messageType"]; hasType {
		messageType = mt
	}

	e.l.Debugf("Sending message to character [%d]: %s", characterId, message)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("portal-action-message").
		AddStep(
			fmt.Sprintf("message-%d", characterId),
			saga.Pending,
			saga.SendMessage,
			saga.SendMessagePayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				MessageType: messageType,
				Message:     message,
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeShowHint sends a hint message to the player
func (e *OperationExecutor) executeShowHint(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	hint, ok := params["hint"]
	if !ok {
		return fmt.Errorf("show_hint operation missing hint parameter")
	}

	var width uint16 = 0
	if widthStr, hasWidth := params["width"]; hasWidth {
		w, err := strconv.ParseUint(widthStr, 10, 16)
		if err != nil {
			return fmt.Errorf("invalid width [%s]: %w", widthStr, err)
		}
		width = uint16(w)
	}

	var height uint16 = 0
	if heightStr, hasHeight := params["height"]; hasHeight {
		h, err := strconv.ParseUint(heightStr, 10, 16)
		if err != nil {
			return fmt.Errorf("invalid height [%s]: %w", heightStr, err)
		}
		height = uint16(h)
	}

	e.l.Debugf("Showing hint to character [%d]: %s (width=%d, height=%d)", characterId, hint, width, height)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("portal-action-hint").
		AddStep(
			fmt.Sprintf("hint-%d", characterId),
			saga.Pending,
			saga.ShowHint,
			saga.ShowHintPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				Hint:        hint,
				Width:       width,
				Height:      height,
			},
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
				MapId:       mapId,
				PortalId:    portalId,
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeCreateSkill creates a new skill for the character
func (e *OperationExecutor) executeCreateSkill(characterId uint32, op operation.Model) error {
	params := op.Params()

	skillIdStr, ok := params["skillId"]
	if !ok {
		return fmt.Errorf("create_skill operation missing skillId parameter")
	}

	skillId, err := strconv.ParseUint(skillIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid skillId [%s]: %w", skillIdStr, err)
	}

	var level byte = 1
	if levelStr, hasLevel := params["level"]; hasLevel {
		l, err := strconv.ParseInt(levelStr, 10, 8)
		if err != nil {
			return fmt.Errorf("invalid level [%s]: %w", levelStr, err)
		}
		level = byte(l)
	}

	var masterLevel byte = 1
	if masterLevelStr, hasMasterLevel := params["masterLevel"]; hasMasterLevel {
		ml, err := strconv.ParseInt(masterLevelStr, 10, 8)
		if err != nil {
			return fmt.Errorf("invalid masterLevel [%s]: %w", masterLevelStr, err)
		}
		masterLevel = byte(ml)
	}

	expiration := time.Now().Add(365 * 24 * time.Hour) // Default to 1 year from now
	if expirationStr, hasExpiration := params["expiration"]; hasExpiration {
		if expirationStr == "-1" {
			// -1 means no expiration, use a far future date
			expiration = time.Now().Add(100 * 365 * 24 * time.Hour)
		} else {
			expMs, err := strconv.ParseInt(expirationStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid expiration [%s]: %w", expirationStr, err)
			}
			if expMs > 0 {
				expiration = time.UnixMilli(expMs)
			}
		}
	}

	e.l.Debugf("Creating skill [%d] for character [%d] (level=%d, masterLevel=%d)", skillId, characterId, level, masterLevel)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("portal-action-create-skill").
		AddStep(
			fmt.Sprintf("create-skill-%d-%d", characterId, skillId),
			saga.Pending,
			saga.CreateSkill,
			saga.CreateSkillPayload{
				CharacterId: characterId,
				SkillId:     uint32(skillId),
				Level:       level,
				MasterLevel: masterLevel,
				Expiration:  expiration,
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeUpdateSkill updates an existing skill for the character
func (e *OperationExecutor) executeUpdateSkill(characterId uint32, op operation.Model) error {
	params := op.Params()

	skillIdStr, ok := params["skillId"]
	if !ok {
		return fmt.Errorf("update_skill operation missing skillId parameter")
	}

	skillId, err := strconv.ParseUint(skillIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid skillId [%s]: %w", skillIdStr, err)
	}

	var level byte = 1
	if levelStr, hasLevel := params["level"]; hasLevel {
		l, err := strconv.ParseInt(levelStr, 10, 8)
		if err != nil {
			return fmt.Errorf("invalid level [%s]: %w", levelStr, err)
		}
		level = byte(l)
	}

	var masterLevel byte = 1
	if masterLevelStr, hasMasterLevel := params["masterLevel"]; hasMasterLevel {
		ml, err := strconv.ParseInt(masterLevelStr, 10, 8)
		if err != nil {
			return fmt.Errorf("invalid masterLevel [%s]: %w", masterLevelStr, err)
		}
		masterLevel = byte(ml)
	}

	expiration := time.Now().Add(365 * 24 * time.Hour) // Default to 1 year from now
	if expirationStr, hasExpiration := params["expiration"]; hasExpiration {
		if expirationStr == "-1" {
			// -1 means no expiration, use a far future date
			expiration = time.Now().Add(100 * 365 * 24 * time.Hour)
		} else {
			expMs, err := strconv.ParseInt(expirationStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid expiration [%s]: %w", expirationStr, err)
			}
			if expMs > 0 {
				expiration = time.UnixMilli(expMs)
			}
		}
	}

	e.l.Debugf("Updating skill [%d] for character [%d] (level=%d, masterLevel=%d)", skillId, characterId, level, masterLevel)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("portal-action-update-skill").
		AddStep(
			fmt.Sprintf("update-skill-%d-%d", characterId, skillId),
			saga.Pending,
			saga.UpdateSkill,
			saga.UpdateSkillPayload{
				CharacterId: characterId,
				SkillId:     uint32(skillId),
				Level:       level,
				MasterLevel: masterLevel,
				Expiration:  expiration,
			},
		).Build()

	return e.sagaP.Create(s)
}
