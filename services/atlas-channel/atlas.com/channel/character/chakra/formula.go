// Package chakra holds the server-side state and math for Chief Bandit
// Chakra (4211001).
//
// It deliberately depends on nothing inside atlas-channel. The recovery
// window is read from socket/handler (the damage, move, map-change and
// skill-prepare/use paths) and written from skill/handler/chakra, and
// skill/handler/* packages already import socket/handler — so a registry
// living under skill/handler would close an import cycle. This mirrors
// character/statreset, which is in-process, tenant-keyed, and consumed the
// same way.
package chakra

import "math"

// CanActivate reports whether Chakra may begin its recovery window.
//
// The client's gate (CUserLocal::DoActiveSkill_Prepare, design §3.2) is
//
//	if (nHP * 100 / nMHP >= 50) return 0;
//
// which is exactly 2*HP >= MaxHP in integer arithmetic — no float, and the
// exactly-50% boundary rejects. A zero MaxHP is treated as never castable
// rather than dividing by zero.
func CanActivate(hp uint16, maxHp uint16) bool {
	if maxHp == 0 {
		return false
	}
	return 2*int32(hp) < int32(maxHp)
}

// Base returns Chakra's base recovery term from the caster's effective LUK.
//
// UNVERIFIED, community-sourced. IDA on all ten available client IDBs
// (design §1, §3.4) proved the client never computes Chakra's HP restore:
// it sends a prepare packet, then a plain USE_SKILL at animation end, and
// renders whatever HP the server reports. No base term exists in any client
// binary or in WZ. 2.9 is the deterministic midpoint of the 2.3-3.5 range
// used by the long-lived open-source server lineage; the derivation and its
// provenance are recorded in docs/tasks/task-213-chakra-hp-restore/design.md
// §3.4. It is kept as a separate function from Recovery so a better-grounded
// term can replace it with a one-function, one-test-file edit.
func Base(luck uint32) int32 {
	v := int64(luck) * 29 / 10
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(v)
}

// Recovery returns the HP Chakra restores: the base term scaled by the
// level's WZ `y` (recovery rate percent). Deterministic — no RNG parameter,
// deliberately, so re-introducing randomisation is a signature change that
// forces these tests to be revisited (design §6.4).
func Recovery(base int32, y int16) int32 {
	if base <= 0 || y <= 0 {
		return 0
	}
	v := int64(base) * int64(y) / 100
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(v)
}

// Applied clamps a computed heal to the recipient's missing HP and to the
// int16 CHANGE_HP wire contract. Chakra never raises HP above maximum and
// never applies a negative delta (which would land as damage).
func Applied(heal int32, hp uint16, maxHp uint16) int16 {
	if heal <= 0 {
		return 0
	}
	missing := int32(maxHp) - int32(hp)
	if missing <= 0 {
		return 0
	}
	if heal > missing {
		heal = missing
	}
	if heal > math.MaxInt16 {
		return math.MaxInt16
	}
	return int16(heal)
}

// EffectiveMaxHpOrBase narrows the effective MaxHp from
// atlas-effective-stats into the uint16 range, falling back to the
// character record's base MaxHp when the upstream returned zero or an
// out-of-range value. Same defensive strategy as the Heal handler's
// unexported effectiveMaxHpOrBase and atlas-character's resolveEffectiveMax.
func EffectiveMaxHpOrBase(effective uint32, base uint16) uint16 {
	if effective == 0 {
		return base
	}
	if effective > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(effective)
}
