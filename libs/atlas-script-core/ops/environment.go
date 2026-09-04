package ops

import (
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

const (
	opMoveEnvironment  = "move_environment"
	opResetEnvironment = "reset_environment"
)

// MoveEnvironment builds a MoveEnvironment step, setting the state of one named
// field object.
//
// Parameters:
//   - name  (required) opaque object name; not validated against WZ data.
//     Whitespace-only is treated as absent. The raw (untrimmed) value is
//     sent in the payload — only the blank-check trims — matching both
//     source executors (map-actions executor.go:257-266, reactor-actions
//     executor.go:280-289).
//   - value (required) the new object state, uint32.
//   - kind  (optional) "ENVIRONMENT" or "OBSTACLE"; blank defaults to
//     ENVIRONMENT (see field.ParseObjectKind). Read raw, not through the
//     Resolver: both current call sites
//     (map-actions script/executor.go, reactor-actions script/executor.go)
//     pass params["kind"] straight to field.ParseObjectKind, and a blank
//     value legitimately defaults to ENVIRONMENT.
func MoveEnvironment(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	name, err := requiredString(p, r, characterId, opMoveEnvironment, "name")
	if err != nil {
		return Step{}, err
	}
	if strings.TrimSpace(name) == "" {
		return Step{}, missingParam(opMoveEnvironment, "name")
	}

	if strings.TrimSpace(p["value"]) == "" {
		return Step{}, missingParam(opMoveEnvironment, "value")
	}
	valueInt, err := requiredInt(p, r, characterId, opMoveEnvironment, "value")
	if err != nil {
		return Step{}, err
	}
	state, err := rangedUint32(opMoveEnvironment, "value", valueInt)
	if err != nil {
		return Step{}, err
	}

	kind, err := field.ParseObjectKind(p["kind"])
	if err != nil {
		return Step{}, invalidParam(opMoveEnvironment, "kind", p["kind"], err)
	}

	return newStep(saga.MoveEnvironment, saga.MoveEnvironmentPayload{
		WorldId:   t.Field().WorldId(),
		ChannelId: t.Field().ChannelId(),
		MapId:     t.Field().MapId(),
		Instance:  t.Field().Instance(),
		Kind:      kind,
		Name:      name,
		State:     state,
	}), nil
}

// ResetEnvironment builds a ResetEnvironment step, clearing every tracked field
// object and restoring the field's objects to their default state. Takes no
// parameters.
func ResetEnvironment(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	return newStep(saga.ResetEnvironment, saga.ResetEnvironmentPayload{
		WorldId:   t.Field().WorldId(),
		ChannelId: t.Field().ChannelId(),
		MapId:     t.Field().MapId(),
		Instance:  t.Field().Instance(),
	}), nil
}
