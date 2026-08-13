package script

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/operation"
)

// FR-2.1/FR-2.2: the three moving operations are the ones whose outcome the
// client is unlocked by a SET_FIELD, not by EnableActions.
func TestOpTable_MovingOperations(t *testing.T) {
	for _, op := range []string{"warp", "warp_to_saved_location", "start_instance_transport"} {
		assert.True(t, IsMovingOperation(op), "[%s] must be classified opClassMoving", op)
	}
}

func TestOpTable_StaticOperations(t *testing.T) {
	for _, op := range []string{
		"play_portal_sound", "drop_message", "show_hint", "block_portal",
		"create_skill", "update_skill", "apply_consumable_effect",
		"cancel_consumable_effect", "save_location", "start_quest",
	} {
		assert.False(t, IsMovingOperation(op), "[%s] must be classified opClassStatic", op)
	}
}

func TestOpTable_UnknownOperationIsNotMoving(t *testing.T) {
	assert.False(t, IsMovingOperation("no_such_operation"),
		"an unknown type is not in the table and is not moving")
}

// FR-2.2: omitting the class is a failure, not a silent default.
func TestValidateOpTable_RejectsUnsetClass(t *testing.T) {
	bad := map[string]opDef{
		"forgot_to_classify": {run: func(e *OperationExecutor, f field.Model, characterId, portalId uint32, op operation.Model) error {
			return nil
		}},
	}
	err := validateOpTable(bad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forgot_to_classify")
	assert.Contains(t, err.Error(), "opClass")
}

func TestValidateOpTable_RejectsNilRun(t *testing.T) {
	bad := map[string]opDef{
		"no_body": {class: opClassStatic},
	}
	err := validateOpTable(bad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no_body")
}

// The production table itself must be valid — this is what init() enforces.
func TestValidateOpTable_ProductionTableIsValid(t *testing.T) {
	assert.NoError(t, validateOpTable(opTable))
}

// Every operation the previous switch handled is still dispatchable.
func TestOpTable_CoversEveryKnownOperation(t *testing.T) {
	want := []string{
		"play_portal_sound", "warp", "drop_message", "show_hint", "block_portal",
		"create_skill", "update_skill", "start_instance_transport",
		"apply_consumable_effect", "cancel_consumable_effect", "save_location",
		"warp_to_saved_location", "start_quest",
	}
	assert.Len(t, opTable, len(want))
	for _, op := range want {
		_, ok := opTable[op]
		assert.True(t, ok, "[%s] must be in opTable", op)
	}
}
