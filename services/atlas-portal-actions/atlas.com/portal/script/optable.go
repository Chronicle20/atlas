package script

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/operation"
)

// opClass classifies a portal operation by whether it moves the character.
//
// This exists because of task-184: a portal outcome that moves the character
// must NOT clear the client's exclusive-request flag (EnableActions) — the
// SET_FIELD produced by the warp clears it, and clearing it early while the
// player still stands inside the portal's collision rect makes the GMS v83
// client legitimately re-fire the ENTER request.
//
// The zero value is deliberately invalid so a new table entry that omits the
// class cannot default to "does not move". validateOpTable rejects it and
// init() panics.
type opClass int

const (
	opClassUnset  opClass = iota // invalid — see validateOpTable
	opClassStatic                // leaves the character where they are
	opClassMoving                // dispatches a warp / field change
)

// opDef is one row of the portal operation dispatch table.
type opDef struct {
	class opClass
	run   func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error
}

// opTable is the single source of truth for both dispatch and classification.
// There is deliberately no second list of "moving" operations anywhere.
var opTable = map[string]opDef{
	"play_portal_sound": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executePlayPortalSound(f, characterId, op)
	}},
	"warp": {class: opClassMoving, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeWarp(f, characterId, op)
	}},
	"drop_message": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeDropMessage(f, characterId, op)
	}},
	"show_hint": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeShowHint(f, characterId, op)
	}},
	"block_portal": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeBlockPortal(f, characterId, portalId, op)
	}},
	"create_skill": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeCreateSkill(characterId, op)
	}},
	"update_skill": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeUpdateSkill(characterId, op)
	}},
	"start_instance_transport": {class: opClassMoving, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeStartInstanceTransport(f, characterId, op)
	}},
	"apply_consumable_effect": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeApplyConsumableEffect(f, characterId, op)
	}},
	"cancel_consumable_effect": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeCancelConsumableEffect(f, characterId, op)
	}},
	"save_location": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeSaveLocation(f, characterId, portalId, op)
	}},
	"warp_to_saved_location": {class: opClassMoving, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeWarpToSavedLocation(f, characterId, op)
	}},
	"start_quest": {class: opClassStatic, run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
		return e.executeStartQuest(f, characterId, op)
	}},
}

// validateOpTable reports the first structural defect in tbl. Extracted from
// init() so a test can exercise it against a deliberately malformed table.
func validateOpTable(tbl map[string]opDef) error {
	for name, def := range tbl {
		if def.class == opClassUnset {
			return fmt.Errorf("portal operation [%s] has no opClass; classify it as opClassStatic or opClassMoving", name)
		}
		if def.run == nil {
			return fmt.Errorf("portal operation [%s] has no run function", name)
		}
	}
	return nil
}

func init() {
	if err := validateOpTable(opTable); err != nil {
		panic(err.Error())
	}
}

// IsMovingOperation reports whether an operation type dispatches a warp or
// field change. An unknown type is not moving — it is never dispatched either
// (ExecuteOperation warns and returns nil), so no character is moved by it.
func IsMovingOperation(opType string) bool {
	return opTable[opType].class == opClassMoving
}
