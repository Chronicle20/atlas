package monster

// cjson empty-array audit (PRD FR-4.10): every slice / map field on a
// status-event body must marshal to `[]` / `{}` when nil, never `null`.
//
// Inventory of body types in kafka.go containing slice or map fields:
//   - statusEventDamagedBody.DamageEntries        ([]damageEntry)
//   - statusEventKilledBody.DamageEntries         ([]damageEntry)
//   - statusEffectAppliedBody.Statuses            (map[string]int32)
//   - statusEffectExpiredBody.Statuses            (map[string]int32)
//   - statusEffectCancelledBody.Statuses          (map[string]int32)
//
// All other body types in kafka.go (statusEventCreatedBody,
// statusEventDestroyedBody, statusEventStartControlBody,
// statusEventAggroChangedBody, statusEventStopControlBody,
// statusEventDamageReflectedBody, statusEventFriendlyDropBody,
// statusEventNextSkillDecidedBody) contain only scalar fields and need
// no MarshalJSON override.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatusEventDamagedBody_EmptyDamageEntries_MarshalsAsArray(t *testing.T) {
	b := statusEventDamagedBody{
		DamageEntries: nil,
	}
	out, err := json.Marshal(b)
	require.NoError(t, err)
	require.Contains(t, string(out), `"damageEntries":[]`, "got: %s", out)
	require.NotContains(t, string(out), `"damageEntries":null`)
}

func TestStatusEventKilledBody_EmptyDamageEntries_MarshalsAsArray(t *testing.T) {
	b := statusEventKilledBody{
		DamageEntries: nil,
	}
	out, err := json.Marshal(b)
	require.NoError(t, err)
	require.Contains(t, string(out), `"damageEntries":[]`, "got: %s", out)
}

func TestStatusEffectAppliedBody_EmptyStatuses_MarshalsAsObject(t *testing.T) {
	b := statusEffectAppliedBody{
		EffectId: "test-effect-id",
		Statuses: nil,
	}
	out, err := json.Marshal(b)
	require.NoError(t, err)
	require.Contains(t, string(out), `"statuses":{}`, "got: %s", out)
	require.NotContains(t, string(out), `"statuses":null`)
}

func TestStatusEffectExpiredBody_EmptyStatuses_MarshalsAsObject(t *testing.T) {
	b := statusEffectExpiredBody{Statuses: nil}
	out, err := json.Marshal(b)
	require.NoError(t, err)
	require.Contains(t, string(out), `"statuses":{}`)
}

func TestStatusEffectCancelledBody_EmptyStatuses_MarshalsAsObject(t *testing.T) {
	b := statusEffectCancelledBody{Statuses: nil}
	out, err := json.Marshal(b)
	require.NoError(t, err)
	require.Contains(t, string(out), `"statuses":{}`)
}

func TestStatusEventDamagedBody_RoundTripPreservesEmpty(t *testing.T) {
	in := statusEventDamagedBody{DamageEntries: nil}
	out, err := json.Marshal(in)
	require.NoError(t, err)
	require.True(t, strings.Contains(string(out), `"damageEntries":[]`))
	var back statusEventDamagedBody
	require.NoError(t, json.Unmarshal(out, &back))
}

func TestStatusEffectAppliedBody_NonReflect_SerializesEmptyReflectFields(t *testing.T) {
	b := statusEffectAppliedBody{
		EffectId: "test-effect-id",
		Statuses: map[string]int32{"FREEZE": 1},
	}
	out, err := json.Marshal(b)
	require.NoError(t, err)
	s := string(out)
	require.Contains(t, s, `"reflectKind":""`, "got: %s", s)
	require.Contains(t, s, `"reflectPercent":0`)
	require.Contains(t, s, `"reflectLtX":0`)
	require.Contains(t, s, `"reflectLtY":0`)
	require.Contains(t, s, `"reflectRbX":0`)
	require.Contains(t, s, `"reflectRbY":0`)
	require.Contains(t, s, `"reflectMaxDamage":0`)
	require.NotContains(t, s, `"reflectKind":null`)
}

func TestStatusEffectAppliedBody_Reflect_SerializesAllReflectFields(t *testing.T) {
	b := statusEffectAppliedBody{
		EffectId:         "test-effect-id",
		Statuses:         map[string]int32{"WEAPON_COUNTER": 30},
		ReflectKind:      "PHYSICAL",
		ReflectPercent:   30,
		ReflectLtX:       -50,
		ReflectLtY:       -30,
		ReflectRbX:       50,
		ReflectRbY:       30,
		ReflectMaxDamage: 32767,
	}
	out, err := json.Marshal(b)
	require.NoError(t, err)
	s := string(out)
	require.Contains(t, s, `"reflectKind":"PHYSICAL"`)
	require.Contains(t, s, `"reflectPercent":30`)
	require.Contains(t, s, `"reflectMaxDamage":32767`)
}
