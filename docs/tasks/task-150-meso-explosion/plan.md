# Meso Explosion — Exploded-Meso Destruction + Damage — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Chief Bandit's Meso Explosion (skill 4211006) functional: decode the meso-explosion attack packet variant, validate and destroy the listed meso drops, and apply the client-reported damage.

**Architecture:** A variant flag threaded through the existing `AttackInfo`/`DamageInfo` codec in `libs/atlas-packet` (three version-invariant wire deltas, no new version gates), plus a validation + CONSUME-emission branch in atlas-channel's `processAttack`. atlas-drops is untouched — its existing `CONSUME` path already removes the drop and atlas-channel's drop consumer already broadcasts the explode animation.

**Tech Stack:** Go, atlas-packet codec + `pt` round-trip test harness, atlas-channel processors, Kafka (`libs/atlas-kafka/producer`), packet-audit tooling.

**Design:** `docs/tasks/task-150-meso-explosion/design.md` (PRD: `prd.md`). All byte layouts below are IDA-verified per design §2; do not re-derive from Cosmic or memory.

## Global Constraints

- Skill id 4211006 is `skill.ChiefBanditMesoExplosionId` (`libs/atlas-constants/skill/constants.go:3179`). Always compare via `skill.Id(...) == skill.ChiefBanditMesoExplosionId` or `skill.Is(...)` — never a bare numeric literal.
- The three variant deltas (per-mob count byte replaces the 2-byte delay; trailing drop list; trailing int16 delay) are **byte-identical across all eight IDA-verified GMS versions — gms_v48/v61/v72/v79/v83/v84/v87/v95** (design §2.1, v2). Add **no new `Region()`/`MajorVersion()` gates** — all surrounding fields keep their existing gates, which already model every base-layout difference across that range (design §2.1a).
- **Per-mob CRC stays on the existing shared `MajorVersion() >= 61` gate** (`damage_info.go:57,83`). The meso `DamageInfo` mode branches ONLY the delay→count+damages part; the CRC read/write is untouched and shared. Do NOT re-gate it to `>= 83` — that snippet in an earlier draft was written against the pre-legacy codec and would drop the per-mob CRC for v61/v72/v79. v48 (< 61) correctly skips the CRC for free because the gate is shared (design §2.1a).
- The decoder MUST NOT size meso damage arrays from the `hits` nibble: the client encodes `nMaxAttackCount & 0xF` there, which wraps at 16 (design §2.2). Only the per-mob count byte sizes the array.
- FR-6: rejection of a meso-explosion attack must produce **zero side effects** — no HP/MP cost, no damage, no broadcast, no CONSUME. Validation therefore runs before the cost block in `processAttack`.
- `services/atlas-drops` must not be modified (FR-9/design §5.3).
- Wire fidelity: the per-drop `hitMask` byte and trailing `mesoDelay` are decoded, retained, and re-encoded even though server logic only uses drop ids.
- `pt.Variants` now spans **v28, v48, v61, v72, v79, v83, v84, v86, v87, v95, jms_v185**; every `for _, v := range pt.Variants` round-trip test runs against all of them. v28/v86 are test-harness boundary variants (round-trip symmetry only, no IDA claim).
- jms_v185's serverbound variant tail is **implemented, not verified** (sender `0xa3aab1` is SCY-virtualized — design §2.3); gms_v92 and gms_12 have no IDB and are not matrix columns (unverified-follows-family — design §2.4). Never fabricate a `packet-audit:verify` marker or evidence hash for any of these three.
- **No new `packet-audit:verify` markers anywhere in this task** (see Task 3 rationale): the melee cells stay pinned to the registry-primary fname; a second marker per cell would orphan under `matrix --check`.
- Tests use the project's Builder pattern (e.g. `drop.NewModelBuilder()`); do not create `*_testhelpers.go` files.
- No literal home/absolute paths in committed files.
- Commands below run from the worktree root (`.worktrees/task-150-meso-explosion`) unless a `cd` is shown.

---

### Task 1: `DamageInfo` meso-explosion mode (libs/atlas-packet)

The meso variant replaces the standard 2-byte `delay` in each damage entry with a 1-byte damage-line count followed by that many 4-byte damages (IDA: v83 `0x96b3fb` — `Encode1(&v132, v71[20])` count byte, then damage loop from offset 24, then `Encode4(CMob::GetCrc(...))`). Verified identical across the legacy senders: v48 `0x6ae4d7` (count `v61[20]`, **no** trailing mob CRC — v48 < 61), v61 `0x7b8a39` (count @`0x7b92f7`, CRC `sub_5CF2AF` @`0x7b932b`), v72 `0x875828` (count @`0x876128`, CRC `sub_61F8A5` @`0x876155`), v79 `0x8c22fd` (count @`0x8c2c2a`, CRC `sub_640131` @`0x8c2c57`). The mob CRC read/write is unchanged and stays on its shared `>= 61` gate.

**Files:**
- Modify: `libs/atlas-packet/model/damage_info.go`
- Test: `libs/atlas-packet/model/damage_info_test.go` (new file)

**Interfaces:**
- Consumes: existing `DamageInfo` struct, `pt` harness (`libs/atlas-packet/test`).
- Produces: `NewMesoExplosionDamageInfo() *DamageInfo` — constructor setting a private `mesoExplosion bool`; Decode/Encode branch on it. Task 2's `AttackInfo` decode loop and fixtures call this.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/model/damage_info_test.go`:

```go
package model

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// The meso-explosion DamageInfo entry (task-150) replaces the standard 2-byte
// delay with a 1-byte damage-line count followed by that many 4-byte damages
// (design §2.1; IDA v83 0x96b3fb: Encode1 count byte, then the damage loop,
// then the mob CRC as usual). hits is unused in this mode. Standard-mode
// entries are covered by the AttackInfo round-trip fixtures.
func TestMesoExplosionDamageInfoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			di := NewMesoExplosionDamageInfo()
			di.SetMonsterId(9001).SetHitAction(0x07).SetDamages([]uint32{111, 222, 333})

			out := NewMesoExplosionDamageInfo()
			pt.RoundTrip(t, ctx, di.Encode, out.Decode, nil)

			if len(out.Damages()) != 3 {
				t.Fatalf("decoded %d damage lines, want 3 (the count byte must size the array)", len(out.Damages()))
			}
			enc1 := pt.Encode(t, ctx, di.Encode, nil)
			enc2 := pt.Encode(t, ctx, out.Encode, nil)
			if !bytes.Equal(enc1, enc2) {
				t.Errorf("re-encode mismatch:\n got % x\nwant % x", enc2, enc1)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./model/ -run TestMesoExplosionDamageInfoRoundTrip -v`
Expected: FAIL (compile error: `undefined: NewMesoExplosionDamageInfo`)

- [ ] **Step 3: Implement the meso mode in `damage_info.go`**

Add the field and constructor (after `NewDamageInfo`):

```go
// NewMesoExplosionDamageInfo constructs a DamageInfo for the meso-explosion
// attack variant (skill 4211006): the wire entry carries a 1-byte damage-line
// count in place of the standard 2-byte delay, so hits is unused (task-150
// design §2.1/§2.2).
func NewMesoExplosionDamageInfo() *DamageInfo {
	return &DamageInfo{mesoExplosion: true}
}
```

Add `mesoExplosion bool` to the `DamageInfo` struct (after `hits byte`).

In `Decode`, branch **only** the delay+damages block. **Leave the existing per-mob
CRC read exactly where it is** — it is already gated `MajorVersion() >= 61` on
main (`damage_info.go:57`), shared between both modes. Do NOT retype it as `>= 83`:

```go
		m.previousPositionY = r.ReadUint16()
		if m.mesoExplosion {
			count := r.ReadByte()
			for range count {
				m.damages = append(m.damages, r.ReadUint32())
			}
		} else {
			m.delay = r.ReadUint16()
			for range m.hits {
				m.damages = append(m.damages, r.ReadUint32())
			}
		}
		// UNCHANGED shared block — already present on main at this gate. v48 (< 61)
		// skips the per-mob CRC in both modes, matching its sender (design §2.1a).
		if t.Region() == "GMS" && t.MajorVersion() >= 61 {
			m.crc = r.ReadUint32()
		}
```

In `Encode`, mirror it — again leaving the shared `>= 61` CRC block untouched:

```go
		w.WriteShort(m.previousPositionY)
		if m.mesoExplosion {
			w.WriteByte(byte(len(m.damages)))
		} else {
			w.WriteShort(m.delay)
		}
		for _, d := range m.damages {
			w.WriteInt(d)
		}
		// UNCHANGED shared block (damage_info.go:83).
		if t.Region() == "GMS" && t.MajorVersion() >= 61 {
			w.WriteInt(m.crc)
		}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./model/ -run 'TestMesoExplosionDamageInfoRoundTrip|TestAttackInfo' -v`
Expected: PASS (new test AND all existing `TestAttackInfoRoundTrip`/`TestAttackInfoVersionBoundary` cases — standard mode unchanged, FR-4)

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/model/damage_info.go libs/atlas-packet/model/damage_info_test.go
git commit -m "feat(task-150): meso-explosion DamageInfo mode in atlas-packet"
```

---

### Task 2: `AttackInfo` meso variant — detection, drop list, trailing delay (libs/atlas-packet)

Variant detection keys off the skill id, which is decoded before the damage entries, so the variant is self-detectable mid-decode. The trailing section (drop count byte → per-drop `{uint32 dropId, byte hitMask}` → int16 delay) sits directly after `characterX`/`characterY` (IDA: v83 `0x96b3fb` — `Encode1(dropListSize)`, loop `Encode4(*(drop + 32)); Encode1(v108[m])`, then `Encode2(a2)`; identical in v84 `0x9aa379`, v87 `0x9eee04`, v95 `0x942200`, and in the four legacy senders — v48 drop list @`0x6aedb1`/`0x6aedcd`/`0x6aeddf` + tail `Encode2` @`0x6aeded`, v61 @`0x7b9378`/`0x7b9394`/`0x7b93a6` + `0x7b93b4`, v72 @`0x8761a9`/`0x8761c5`/`0x8761d7` + `0x8761e5`, v79 @`0x8c2ca7`/`0x8c2cc3`/`0x8c2cd5` + `0x8c2ce3`).

**Files:**
- Modify: `libs/atlas-packet/model/attack_info.go`
- Test: `libs/atlas-packet/model/attack_info_test.go`

**Interfaces:**
- Consumes: `NewMesoExplosionDamageInfo()` from Task 1; existing `skill.ChiefBanditMesoExplosionId`.
- Produces (used by Tasks 3 and 6):
  - `type ExplodedMesoDrop struct{ dropId uint32; hitMask byte }` with `NewExplodedMesoDrop(dropId uint32, hitMask byte) ExplodedMesoDrop`, `DropId() uint32`, `HitMask() byte`.
  - `(m *AttackInfo) ExplodedMesoDrops() []uint32` — ids only, empty for non-meso attacks (FR-3).
  - `(m *AttackInfo) ExplodedMesoDropEntries() []ExplodedMesoDrop`, `(m *AttackInfo) MesoDelay() uint16` — wire-fidelity accessors.
  - `(m *AttackInfo) SetExplodedMesoDrops([]ExplodedMesoDrop) *AttackInfo`, `(m *AttackInfo) SetMesoDelay(uint16) *AttackInfo` — fixture builders.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-packet/model/attack_info_test.go` (add `"bytes"` to the imports):

```go
// mesoExplosionAttackInfo builds a meso-explosion (4211006) melee attack.
// Fixture values are chosen to catch nibble-misuse (design §2.2): hits is 0 —
// the client writes nMaxAttackCount & 0xF there, which wraps to 0 at
// attackCount 16 — while the per-mob damage-line counts are 3 and 1. A decoder
// that sizes damage arrays from the nibble reads zero damages and fails the
// round-trip with unconsumed bytes.
func mesoExplosionAttackInfo() *AttackInfo {
	ai := NewAttackInfo(AttackTypeMelee)
	ai.SetDamage(2) // mob count (high nibble)
	ai.SetHits(0)   // nMaxAttackCount & 0xF for attackCount=16 → wrapped to 0
	ai.SetSkillId(4211006)
	ai.SetOption(0)
	ai.SetLeft(false)
	ai.SetAttackAction(0x05)
	ai.SetActionSpeed(4)
	di := NewMesoExplosionDamageInfo()
	di.SetMonsterId(9001).SetHitAction(0x07).SetDamages([]uint32{100, 200, 300})
	ai.AddDamageInfo(*di)
	di2 := NewMesoExplosionDamageInfo()
	di2.SetMonsterId(9002).SetHitAction(0x06).SetDamages([]uint32{400})
	ai.AddDamageInfo(*di2)
	ai.SetExplodedMesoDrops([]ExplodedMesoDrop{
		NewExplodedMesoDrop(501001, 0x01),
		NewExplodedMesoDrop(501002, 0x03),
	})
	ai.SetMesoDelay(120)
	return ai
}

// TestAttackInfoMesoExplosionRoundTrip pins the meso-explosion variant
// (task-150 design §2.1): per-mob count byte instead of the int16 delay,
// trailing exploded-drop list, trailing int16 delay. The deltas are
// version-invariant; surrounding fields keep their existing gates, which is
// what running across all pt.Variants proves.
func TestAttackInfoMesoExplosionRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			ai := mesoExplosionAttackInfo()

			out := NewAttackInfo(AttackTypeMelee)
			pt.RoundTrip(t, ctx, ai.Encode, out.Decode, nil)

			ids := out.ExplodedMesoDrops()
			if len(ids) != 2 || ids[0] != 501001 || ids[1] != 501002 {
				t.Errorf("exploded meso drop ids = %v, want [501001 501002]", ids)
			}
			if out.MesoDelay() != 120 {
				t.Errorf("meso delay = %d, want 120", out.MesoDelay())
			}
			dis := out.DamageInfo()
			if len(dis) != 2 || len(dis[0].Damages()) != 3 || len(dis[1].Damages()) != 1 {
				t.Fatalf("per-mob damage counts wrong: got %d entries", len(dis))
			}
			entries := out.ExplodedMesoDropEntries()
			if len(entries) != 2 || entries[0].HitMask() != 0x01 || entries[1].HitMask() != 0x03 {
				t.Errorf("hit masks did not round-trip: %+v", entries)
			}
			enc1 := pt.Encode(t, ctx, ai.Encode, nil)
			enc2 := pt.Encode(t, ctx, out.Encode, nil)
			if !bytes.Equal(enc1, enc2) {
				t.Errorf("re-encode mismatch:\n got % x\nwant % x", enc2, enc1)
			}
		})
	}
}

// TestAttackInfoNonMesoHasNoDropList pins FR-3's "empty for non-meso attacks"
// contract and that the variant tail is never written for other skills.
func TestAttackInfoNonMesoHasNoDropList(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	ai := sampleAttackInfo(AttackTypeMelee)
	out := NewAttackInfo(AttackTypeMelee)
	pt.RoundTrip(t, ctx, ai.Encode, out.Decode, nil)
	if len(out.ExplodedMesoDrops()) != 0 {
		t.Errorf("non-meso attack decoded %d exploded drops, want 0", len(out.ExplodedMesoDrops()))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./model/ -run TestAttackInfoMesoExplosion -v`
Expected: FAIL (compile error: `undefined: ExplodedMesoDrop`)

- [ ] **Step 3: Implement the variant in `attack_info.go`**

Add the entry type (above `NewAttackInfo`):

```go
// ExplodedMesoDrop is one entry of the meso-explosion trailing drop list: the
// detonated meso drop's object id (CDrop field +32) plus the client's bitmask
// of which attacked-mob indices that drop's explosion damaged. The hit mask is
// retained for wire fidelity only; server logic consumes just the ids.
type ExplodedMesoDrop struct {
	dropId  uint32
	hitMask byte
}

func NewExplodedMesoDrop(dropId uint32, hitMask byte) ExplodedMesoDrop {
	return ExplodedMesoDrop{dropId: dropId, hitMask: hitMask}
}

func (e ExplodedMesoDrop) DropId() uint32 { return e.dropId }
func (e ExplodedMesoDrop) HitMask() byte  { return e.hitMask }
```

Add two fields to the `AttackInfo` struct (after `bulletY`):

```go
	explodedMesoDrops    []ExplodedMesoDrop
	mesoDelay            uint16
```

In `Decode`, right after `m.skillId = r.ReadUint32()`, add:

```go
		// Meso Explosion (4211006) is a CLOSE_RANGE_ATTACK variant written by a
		// dedicated client sender. Its three deltas (per-mob count byte, trailing
		// drop list, trailing delay) are byte-identical across every IDA-verified
		// version (task-150 design §2.1), so one flag and no new version gates.
		isMesoExplosion := skill.Id(m.skillId) == skill.ChiefBanditMesoExplosionId
```

Change the damage-entry loop:

```go
		for range m.damage {
			var di *DamageInfo
			if isMesoExplosion {
				di = NewMesoExplosionDamageInfo()
			} else {
				di = NewDamageInfo(m.hits)
			}
			di.Decode(l, ctx)(r, options)
			m.damageInfo = append(m.damageInfo, *di)
		}
```

Right after `m.characterY = r.ReadUint16()`, add:

```go
		if isMesoExplosion {
			dropCount := r.ReadByte()
			for range dropCount {
				m.explodedMesoDrops = append(m.explodedMesoDrops, ExplodedMesoDrop{
					dropId:  r.ReadUint32(),
					hitMask: r.ReadByte(),
				})
			}
			m.mesoDelay = r.ReadUint16()
		}
```

(The variant is melee-only and skill-id keyed, so it cannot co-occur with the ranged bullet block or the grenade/spark/dragon specials that follow.)

In `Encode`, mirror both: compute `isMesoExplosion := skill.Id(m.skillId) == skill.ChiefBanditMesoExplosionId` at the top of the returned closure, and after `w.WriteShort(m.characterY)` add:

```go
			if isMesoExplosion {
				w.WriteByte(byte(len(m.explodedMesoDrops)))
				for _, e := range m.explodedMesoDrops {
					w.WriteInt(e.dropId)
					w.WriteByte(e.hitMask)
				}
				w.WriteShort(m.mesoDelay)
			}
```

(Encode's damage-entry loop needs no change — each `DamageInfo` carries its own `mesoExplosion` flag from Task 1.)

Add accessors and builders (with the existing accessor/builder blocks):

```go
// ExplodedMesoDrops returns the drop object ids listed by a meso-explosion
// attack. Empty for every other attack (FR-3).
func (m *AttackInfo) ExplodedMesoDrops() []uint32 {
	ids := make([]uint32, 0, len(m.explodedMesoDrops))
	for _, e := range m.explodedMesoDrops {
		ids = append(ids, e.dropId)
	}
	return ids
}

// ExplodedMesoDropEntries returns the full wire entries (id + hit mask).
func (m *AttackInfo) ExplodedMesoDropEntries() []ExplodedMesoDrop {
	return m.explodedMesoDrops
}

func (m *AttackInfo) MesoDelay() uint16 {
	return m.mesoDelay
}

func (m *AttackInfo) SetExplodedMesoDrops(entries []ExplodedMesoDrop) *AttackInfo {
	m.explodedMesoDrops = entries
	return m
}

func (m *AttackInfo) SetMesoDelay(delay uint16) *AttackInfo {
	m.mesoDelay = delay
	return m
}
```

- [ ] **Step 4: Run the full model package tests**

Run: `cd libs/atlas-packet && go test ./model/ -v`
Expected: PASS — the two new tests AND every pre-existing test (`TestAttackInfoRoundTrip`, `TestAttackInfoVersionBoundary`, movement, asset, etc.). Any failure in an existing attack test means a standard-layout regression (FR-4) — fix before proceeding.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/model/attack_info.go libs/atlas-packet/model/attack_info_test.go
git commit -m "feat(task-150): meso-explosion AttackInfo variant decode/encode"
```

---

### Task 3: Serverbound wrapper fixture + variant verification documentation (libs/atlas-packet)

The audit-linkage wrapper `AttackMeleeRequest` (CLOSE_RANGE_ATTACK) delegates to `AttackInfo`, so it inherits the variant; this task pins that via a fixture and documents the per-version IDA evidence.

**Marker decision (resolved at plan time — do not "fix" this during execution):** No new `packet-audit:verify` markers are added. The five melee cells are already pinned (`attack_request_test.go:41-45`) to the registry-primary fname `CUserLocal::TryDoingNormalAttack`; evidence records are one-per-cell and pin that same fname. The meso senders are already registered as `fname_alts` on every version's CLOSE_RANGE_ATTACK serverbound registry row, but they are absent from the IDA exports, and `matrix --check` flags any marker whose `ida=` address matches neither the evidence record nor the audit report as an **orphan**. A second marker per cell therefore cannot pass the machine check. The variant evidence lives as a documentation comment instead (mirrors design §2.3's "no verification marker for a read order we could not read" posture; the clientbound meso round-trip `TestAttackMeleeWithMesoExplosionRoundTrip` in `libs/atlas-packet/character/clientbound/attack_test.go` already exists and satisfies FR-11's test requirement).

**Files:**
- Modify: `libs/atlas-packet/character/serverbound/attack_request_test.go`

**Interfaces:**
- Consumes: `model.NewMesoExplosionDamageInfo`, `model.NewExplodedMesoDrop`, `SetExplodedMesoDrops`, `SetMesoDelay` (Tasks 1–2); existing `AttackMeleeRequest` wrapper.
- Produces: `TestAttackMeleeRequestMesoExplosion` — the wrapper-level fixture the audit trail cites.

- [ ] **Step 1: Write the test (fails until Tasks 1–2 are merged; passes immediately if run after)**

Append to `libs/atlas-packet/character/serverbound/attack_request_test.go`:

```go
// Meso Explosion (skill 4211006) — CLOSE_RANGE_ATTACK variant written by a
// dedicated client sender, NOT TryDoingNormalAttack. Variant deltas (per-mob
// count byte replaces the int16 delay; trailing {dropId int32, hitMask byte}
// list; trailing int16 delay) verified byte-identical across every readable
// IDB (task-150 design §2.1–§2.2, v2):
//
//	gms_v48  CUserLocal::DoActiveSkill_MesoExplosion  0x6ae4d7  (no per-mob CRC, <61)
//	gms_v61  meso sender sub_7B8A39                   0x7b8a39
//	gms_v72  meso sender sub_875828                   0x875828  (head skill-data CRC, v72+)
//	gms_v79  meso sender 0x8c22fd (TryDoingMeleeAttack overload; short action, 2 CRCs)
//	gms_v83  CUserLocal::DoActiveSkill_MesoExplosion  0x96b3fb
//	gms_v84  meso sender (IDB label wrong)            0x9aa379
//	gms_v87  CUserLocal::DoActiveSkill_MesoExplosion  0x9eee04
//	gms_v95  CUserLocal::DoActiveSkill_MesoExplosion  0x942200
//	jms_v185 sub_A3AAB1 @0xa3aab1 — encode tail SCY-virtualized: the jms
//	         serverbound variant tail is implemented from the GMS invariants
//	         plus jms clientbound symmetry, NOT statically verified (§2.3).
//	gms_v92  no IDB — follows the GMS >= 87 family branch, unverified (§2.4).
//	gms_12   no IDB — follows the very-legacy GMS < 48 branch, unverified (§2.4).
//
// The senders are registered as fname_alts on the CLOSE_RANGE_ATTACK
// serverbound registry rows. No new packet-audit:verify markers here: the
// melee cells stay pinned to the registry-primary fname (markers above), and
// a second marker per cell would orphan under `matrix --check` (its ida=
// address matches neither the evidence record nor the audit report).
func TestAttackMeleeRequestMesoExplosion(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			ai := model.NewAttackInfo(model.AttackTypeMelee)
			ai.SetDamage(2) // mob count nibble
			ai.SetHits(0)   // nMaxAttackCount & 0xF wraps to 0 at attackCount 16 (§2.2)
			ai.SetSkillId(4211006)
			ai.SetOption(0)
			ai.SetLeft(true)
			ai.SetAttackAction(0x05)
			ai.SetActionSpeed(4)
			di := model.NewMesoExplosionDamageInfo()
			di.SetMonsterId(9001).SetHitAction(0x07).SetDamages([]uint32{100, 200, 300})
			ai.AddDamageInfo(*di)
			di2 := model.NewMesoExplosionDamageInfo()
			di2.SetMonsterId(9002).SetHitAction(0x06).SetDamages([]uint32{400})
			ai.AddDamageInfo(*di2)
			ai.SetExplodedMesoDrops([]model.ExplodedMesoDrop{
				model.NewExplodedMesoDrop(501001, 0x01),
				model.NewExplodedMesoDrop(501002, 0x03),
			})
			ai.SetMesoDelay(120)

			m := AttackMeleeRequest{attackInfo: *ai}
			pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `cd libs/atlas-packet && go test ./character/serverbound/ -run TestAttackMeleeRequest -v`
Expected: PASS (both the existing `TestAttackMeleeRequest` and the new meso test)

- [ ] **Step 3: Confirm the clientbound meso round-trip still passes (FR-11)**

Run: `cd libs/atlas-packet && go test ./character/clientbound/ -run TestAttackMeleeWithMesoExplosionRoundTrip -v`
Expected: PASS (pre-existing test; per-mob counts 3 and 1 with hits=1 — already exercises the variable-count encode)

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-packet/character/serverbound/attack_request_test.go
git commit -m "test(task-150): serverbound meso-explosion wrapper fixture + IDA evidence docs"
```

---

### Task 4: Validation helper + `AttackCount` accessor (atlas-channel)

`attackCount` on skill 4211006 is the max detonatable drops (10–20 by level; WZ-verified, design §3). The plumbing exists end-to-end but `effect.Model` lacks the accessor. The meso-drop predicate is `Meso() > 0` (same predicate as the pickup consumer).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_meso_explosion.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_meso_explosion_test.go`

**Interfaces:**
- Consumes: `drop.Model` (`atlas-channel/drop`), `effect.Model`.
- Produces (used by Task 6):
  - `(m Model) AttackCount() uint32` on `effect.Model`.
  - `validateMesoExplosion(dropIds []uint32, fieldDrops map[uint32]drop.Model, maxCount uint32) (uint32, bool)` — returns `(offendingDropId, false)` on rejection (`0` when the failure is the over-max count, which has no single offending drop), `(0, true)` when the attack is valid. An empty `dropIds` validates trivially.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_meso_explosion_test.go`:

```go
package handler

import (
	"atlas-channel/drop"
	"testing"
)

func mesoDrop(t *testing.T, id uint32, meso uint32) drop.Model {
	t.Helper()
	return drop.NewModelBuilder().SetId(id).SetMeso(meso).MustBuild()
}

func itemDrop(t *testing.T, id uint32) drop.Model {
	t.Helper()
	return drop.NewModelBuilder().SetId(id).SetItem(2000000, 1).MustBuild()
}

func TestValidateMesoExplosion(t *testing.T) {
	fieldDrops := map[uint32]drop.Model{
		11: mesoDrop(t, 11, 500),
		22: mesoDrop(t, 22, 120),
		33: itemDrop(t, 33),
	}

	tests := []struct {
		name      string
		dropIds   []uint32
		maxCount  uint32
		wantOk    bool
		wantBadId uint32
	}{
		{name: "happy path", dropIds: []uint32{11, 22}, maxCount: 10, wantOk: true},
		{name: "empty list is legal", dropIds: nil, maxCount: 10, wantOk: true},
		{name: "over skill max", dropIds: []uint32{11, 22}, maxCount: 1, wantOk: false, wantBadId: 0},
		{name: "duplicate id", dropIds: []uint32{11, 11}, maxCount: 10, wantOk: false, wantBadId: 11},
		{name: "unknown drop", dropIds: []uint32{11, 99}, maxCount: 10, wantOk: false, wantBadId: 99},
		{name: "non-meso drop", dropIds: []uint32{11, 33}, maxCount: 10, wantOk: false, wantBadId: 33},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			badId, ok := validateMesoExplosion(tc.dropIds, fieldDrops, tc.maxCount)
			if ok != tc.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOk)
			}
			if !ok && badId != tc.wantBadId {
				t.Errorf("offending drop id = %d, want %d", badId, tc.wantBadId)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestValidateMesoExplosion -v`
Expected: FAIL (compile error: `undefined: validateMesoExplosion`)

- [ ] **Step 3: Implement accessor + helper**

Append to `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go`:

```go
// AttackCount returns the skill's attackCount attribute. For Meso Explosion
// (4211006) it is the maximum number of drops one attack may detonate
// (10–20 by level; task-150 design §3).
func (m Model) AttackCount() uint32 {
	return m.attackCount
}
```

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_meso_explosion.go`:

```go
package handler

import (
	"atlas-channel/drop"
)

// validateMesoExplosion checks the exploded-drop list of a Meso Explosion
// attack against the drops in the attacker's field (task-150 FR-5/FR-6/FR-7):
// the listed count must not exceed the skill's attackCount, ids must be
// unique, every id must exist in the field, and every drop must be a meso
// drop (Meso() > 0 — same predicate as the pickup consumer). The field-scoped
// fieldDrops map structurally enforces the same-field/instance check.
//
// Returns (offendingDropId, false) when the attack must be rejected — 0 when
// the failure is the over-max count, which has no single offending drop — and
// (0, true) when valid. An empty list validates trivially: the player can
// swing with nothing to detonate.
func validateMesoExplosion(dropIds []uint32, fieldDrops map[uint32]drop.Model, maxCount uint32) (uint32, bool) {
	if uint32(len(dropIds)) > maxCount {
		return 0, false
	}
	seen := make(map[uint32]struct{}, len(dropIds))
	for _, id := range dropIds {
		if _, dup := seen[id]; dup {
			return id, false
		}
		seen[id] = struct{}{}
		d, ok := fieldDrops[id]
		if !ok {
			return id, false
		}
		if d.Meso() == 0 {
			return id, false
		}
	}
	return 0, true
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestValidateMesoExplosion -v`
Expected: PASS (all six cases)

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/data/skill/effect/model.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_meso_explosion.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_meso_explosion_test.go
git commit -m "feat(task-150): meso-explosion validation helper + AttackCount accessor"
```

---

### Task 5: Drop CONSUME command — message types, batched producer, processor method (atlas-channel)

One buffered emission carrying N messages (design §4.3-A): one `CONSUME` per drop, keyed by dropId, all in a single produce call. The channel's `Command` envelope gains the `TransactionId` field atlas-drops already unmarshals (wire-compatible either way; the existing reservation provider keeps sending the zero UUID).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/drop/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/drop/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/drop/processor.go`
- Test: `services/atlas-channel/atlas.com/channel/drop/producer_test.go` (new file)

**Interfaces:**
- Consumes: `drop2.Command[E]` envelope, `producer.RawMessage`/`MessageProvider`/`CreateKey` (`libs/atlas-kafka/producer`), `model.FixedProvider`.
- Produces (used by Task 6):
  - `drop2.CommandTypeConsume = "CONSUME"`, `drop2.ConsumeCommandBody{DropId uint32}`, `TransactionId uuid.UUID` on `drop2.Command[E]` (matches atlas-drops' `CommandConsumeBody`/envelope in `services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go:66-75,180-183`).
  - `ConsumeAllCommandProvider(transactionId uuid.UUID, f field.Model, dropIds []uint32) model.Provider[[]kafka.Message]`.
  - `(p *Processor) ConsumeAll(f field.Model, dropIds []uint32) error` — no-op on empty slice; one `uuid.New()` transaction id spans the batch.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/drop/producer_test.go`:

```go
package drop

import (
	drop2 "atlas-channel/kafka/message/drop"
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/google/uuid"
)

// One CONSUME command per exploded drop, keyed by dropId, all in a single
// buffered provider (task-150 design §4.3-A / FR-8).
func TestConsumeAllCommandProvider(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	txId := uuid.New()
	dropIds := []uint32{11, 22, 33}

	msgs, err := ConsumeAllCommandProvider(txId, f, dropIds)()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != len(dropIds) {
		t.Fatalf("got %d messages, want %d", len(msgs), len(dropIds))
	}
	for i, want := range dropIds {
		if !bytes.Equal(msgs[i].Key, producer.CreateKey(int(want))) {
			t.Errorf("message %d key mismatch", i)
		}
		var cmd drop2.Command[drop2.ConsumeCommandBody]
		if err := json.Unmarshal(msgs[i].Value, &cmd); err != nil {
			t.Fatalf("message %d unmarshal: %v", i, err)
		}
		if cmd.Type != drop2.CommandTypeConsume {
			t.Errorf("message %d type = %q, want %q", i, cmd.Type, drop2.CommandTypeConsume)
		}
		if cmd.Body.DropId != want {
			t.Errorf("message %d dropId = %d, want %d", i, cmd.Body.DropId, want)
		}
		if cmd.TransactionId != txId {
			t.Errorf("message %d transactionId = %s, want %s", i, cmd.TransactionId, txId)
		}
		if cmd.WorldId != f.WorldId() || cmd.ChannelId != f.ChannelId() || cmd.MapId != f.MapId() || cmd.Instance != f.Instance() {
			t.Errorf("message %d field envelope mismatch: %+v", i, cmd)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./drop/ -run TestConsumeAllCommandProvider -v`
Expected: FAIL (compile error: `undefined: ConsumeAllCommandProvider`)

- [ ] **Step 3: Implement message types, provider, processor method**

In `services/atlas-channel/atlas.com/channel/kafka/message/drop/kafka.go`:

Add `CommandTypeConsume` to the existing command const block — **keep every
existing entry**. Main now has `CommandTypeSpawn` too (added by task-149's Pick
Pocket meso spawn); do not drop it:

```go
const (
	EnvCommandTopic               = "COMMAND_TOPIC_DROP"
	CommandTypeRequestReservation = "REQUEST_RESERVATION"
	CommandTypeSpawn              = "SPAWN"
	CommandTypeConsume            = "CONSUME"
)
```

Add `TransactionId` to the envelope (first field, matching atlas-drops — the
channel `Command[E]` on main still omits it; atlas-drops already unmarshals it):

```go
type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}
```

Add the body type (after `RequestReservationCommandBody`):

```go
// ConsumeCommandBody is the body for CONSUME commands (matches atlas-drops'
// CommandConsumeBody). Consume removes the drop without crediting anyone —
// used to destroy Meso Explosion's detonated meso drops (task-150).
type ConsumeCommandBody struct {
	DropId uint32 `json:"dropId"`
}
```

Append to `services/atlas-channel/atlas.com/channel/drop/producer.go`:

```go
// ConsumeAllCommandProvider yields one CONSUME command per exploded drop,
// keyed by dropId, in a single buffered provider so the batch goes out in one
// produce call (task-150 design §4.3-A / FR-8). The shared transaction id
// correlates the batch in atlas-drops logs.
func ConsumeAllCommandProvider(transactionId uuid.UUID, f field.Model, dropIds []uint32) model.Provider[[]kafka.Message] {
	raws := make([]producer.RawMessage, 0, len(dropIds))
	for _, dropId := range dropIds {
		raws = append(raws, producer.RawMessage{
			Key: producer.CreateKey(int(dropId)),
			Value: &drop2.Command[drop2.ConsumeCommandBody]{
				TransactionId: transactionId,
				WorldId:       f.WorldId(),
				ChannelId:     f.ChannelId(),
				MapId:         f.MapId(),
				Instance:      f.Instance(),
				Type:          drop2.CommandTypeConsume,
				Body:          drop2.ConsumeCommandBody{DropId: dropId},
			},
		})
	}
	return producer.MessageProvider(model.FixedProvider(raws))
}
```

(Add `"github.com/google/uuid"` to `producer.go`'s imports.)

Append to `services/atlas-channel/atlas.com/channel/drop/processor.go`:

```go
// ConsumeAll emits one drop CONSUME command per exploded meso drop in a
// single produce call, carrying the attacker's field in the envelope
// (task-150 FR-8). atlas-drops removes each drop and emits CONSUMED; the
// drop consumer then announces the explode animation to the field.
func (p *Processor) ConsumeAll(f field.Model, dropIds []uint32) error {
	if len(dropIds) == 0 {
		return nil
	}
	return producer.ProviderImpl(p.l)(p.ctx)(drop2.EnvCommandTopic)(ConsumeAllCommandProvider(uuid.New(), f, dropIds))
}
```

(Add `"github.com/google/uuid"` to `processor.go`'s imports.)

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./drop/... ./kafka/... -count=1`
Expected: PASS (new provider test + all existing drop/kafka tests — envelope change must not break the reservation path)

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/drop/kafka.go services/atlas-channel/atlas.com/channel/drop/producer.go services/atlas-channel/atlas.com/channel/drop/processor.go services/atlas-channel/atlas.com/channel/drop/producer_test.go
git commit -m "feat(task-150): drop CONSUME batch producer in atlas-channel"
```

---

### Task 6: Wire meso explosion into `processAttack` (atlas-channel)

Validation runs inside the `ai.SkillId() > 0` block, after the skill effect is loaded and **before** the HP/MP cost block (design §4.2-A — FR-6 requires zero side effects on rejection, and the cost deduction is a side effect). The consume emission runs post-broadcast, replacing the TODO (FR-12). Damage lines flow through `processDamageInfoEntry` untouched — variable-length `di.Damages()` needs no pipeline change (FR-10) — and the melee broadcast already handles `isMesoExplosion` (`socket/writer/character_attack_melee.go:19`; FR-11).

**Current-main anchors (verified 2026-07-26; the handler grew after task-149's Pick
Pocket + task-148's Sacrifice landed — re-`grep`, do not trust old line numbers):**
`if ai.SkillId() > 0 {` at ~L516; the effect load `se, err = skill2.NewProcessor(l, ctx).GetEffect(ai.SkillId(), sk.Level())` at ~L528; the cost-gate comment `// Skip the generic cost block …` at ~L533; the `if hasProjectilePlan {` block at ~L653; the `// TODO destroy Chief Bandit exploded mesos` line at ~L671 (now sitting next to an unrelated `// TODO decrease HP from DragonKnight Sacrifice` from task-148 — leave that one alone). The constants alias in this file is `skill3` (constants) and `skill2` (processor) — match the file's existing imports.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`

**Interfaces:**
- Consumes: `validateMesoExplosion` + `se.AttackCount()` (Task 4), `ai.ExplodedMesoDrops()` (Task 2), `drop.NewProcessor(l, ctx).InMapModelProvider(f)` / `.ConsumeAll(f, ids)` (Task 5), `skill3.Is` (`libs/atlas-constants/skill`).
- Produces: the complete meso-explosion attack flow; the `// TODO destroy Chief Bandit exploded mesos` line is gone.

- [ ] **Step 1: Add the validation gate**

In `processAttack` (`character_attack_common.go`), add `"atlas-channel/drop"` to the imports. Declare the stash variable right before the `if ai.SkillId() > 0 {` line:

```go
					var explodedMesoDropIds []uint32
```

Inside the `ai.SkillId() > 0` block, immediately after the `se, err = ...` error check and BEFORE the cost-gate comment (`// Skip the generic cost block ...`), insert:

```go
						// Meso Explosion (task-150): validate the exploded-drop list
						// against the field's drops BEFORE any side effect (FR-6 —
						// rejection must skip cost, damage, broadcast, and destruction).
						// One field-scoped fetch; the map keys structurally enforce the
						// same-field/instance check.
						if skill3.Is(skill3.Id(ai.SkillId()), skill3.ChiefBanditMesoExplosionId) {
							ds, dErr := drop.NewProcessor(l, ctx).InMapModelProvider(s.Field())()
							if dErr != nil {
								return dErr
							}
							fieldDrops := make(map[uint32]drop.Model, len(ds))
							for _, d := range ds {
								fieldDrops[d.Id()] = d
							}
							if badId, ok := validateMesoExplosion(ai.ExplodedMesoDrops(), fieldDrops, se.AttackCount()); !ok {
								l.Warnf("Character [%d] meso-explosion attack with skill [%d] rejected: drop [%d] failed validation.", s.CharacterId(), ai.SkillId(), badId)
								return nil
							}
							explodedMesoDropIds = ai.ExplodedMesoDrops()
						}
```

- [ ] **Step 2: Replace the TODO with the consume emission**

Delete the line `// TODO destroy Chief Bandit exploded mesos` (~L671 on current main; was L407 pre-Pick-Pocket) and insert after the projectile-emission block (after the `if hasProjectilePlan { ... }` closing brace):

```go
					// Destroy the validated exploded meso drops (Chief Bandit Meso
					// Explosion, task-150). CONSUME removes each drop in atlas-drops
					// without crediting anyone; the drop consumer then announces
					// DropDestroy with the explode animation to the whole field.
					// Same at-least-once posture as the projectile emission above:
					// damage is already applied, so failures log and continue.
					if len(explodedMesoDropIds) > 0 {
						if cErr := drop.NewProcessor(l, ctx).ConsumeAll(s.Field(), explodedMesoDropIds); cErr != nil {
							l.WithError(cErr).Errorf("Unable to emit CONSUME for [%d] exploded meso drops for character [%d].", len(explodedMesoDropIds), s.CharacterId())
						} else {
							l.Debugf("Destroyed [%d] exploded meso drops for character [%d].", len(explodedMesoDropIds), s.CharacterId())
						}
					}
```

- [ ] **Step 3: Verify the build and the whole handler package**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/... -count=1`
Expected: build clean; all handler tests PASS (cost-gate, MP Eater, projectile, meso validation).

- [ ] **Step 4: Verify the TODO is gone**

Run: `grep -n "destroy Chief Bandit" services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`
Expected: no output (exit 1) — FR-12 satisfied. (Note: Pick Pocket (4211003) is already fully implemented on main — task-149, `pickPocketResolveState` et al. — so there is no longer a Pick Pocket TODO to preserve; the only neighboring TODO is `// TODO decrease HP from DragonKnight Sacrifice`, task-148, which is out of scope and must remain.)

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
git commit -m "feat(task-150): wire meso-explosion validation + drop destruction into processAttack"
```

---

### Task 7: Packet-audit artifacts (design §5.4)

No evidence records or markers change (Task 3 rationale), so the matrix content should be unchanged — this task documents the meso variant at each cell and proves the machine checks stay green. The eight GMS cells (v48–v95) are IDA-verified but carry no NEW verify marker (fname_alt-absent-from-export orphan rule, Task 3); the four legacy audit MDs record their verified sender + deltas, and the jms MD records why its tail is unverifiable.

**Files:**
- Modify: `docs/packets/audits/gms_v48/CharacterAttackMeleeRequest.md`
- Modify: `docs/packets/audits/gms_v61/CharacterAttackMeleeRequest.md`
- Modify: `docs/packets/audits/gms_v72/CharacterAttackMeleeRequest.md`
- Modify: `docs/packets/audits/gms_v79/CharacterAttackMeleeRequest.md`
- Modify: `docs/packets/audits/jms_v185/CharacterAttackMeleeRequest.md`
- Possibly regenerated: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json` (commit only if the tool changes them)

**Interfaces:**
- Consumes: `go run ./tools/packet-audit matrix` / `matrix --check` (documented in `docs/packets/audits/VERIFYING_A_PACKET.md` §8).
- Produces: the audit notes recording each version's meso sender/deltas and why no new verify pin was added.

- [ ] **Step 1: Append the verified-variant note to each of the four legacy audit MDs**

To each of `docs/packets/audits/gms_v{48,61,72,79}/CharacterAttackMeleeRequest.md`, append a note recording the version's meso sender address and the three verified deltas. Use the per-version facts from design §2.1's evidence excerpts (v48 `0x6ae4d7` — no per-mob CRC; v61 `0x7b8a39`; v72 `0x875828` — head skill-data CRC; v79 `0x8c22fd` — short action + two head CRCs). Template (fill in the version-specific address/notes):

```markdown

---

## task-150 note — Meso Explosion variant (hand-added; keep on regeneration)

CLOSE_RANGE_ATTACK carries a Meso Explosion (4211006) variant written by a
dedicated sender (this version: `<addr>`), dispatched from `DoActiveSkill`
case 4211006. IDA-verified deltas vs. the standard melee attack (design §2.1):
per-mob `Encode1(damageLineCount)` replaces the int16 delay; trailing
`{dropId int32, hitMask byte}` list after characterX/Y; trailing int16 delay.
All base-layout fields (per-mob CRC, action width, head CRCs) follow this
version's existing standard-melee gates (design §2.1a) — the variant adds no
new gate. No new packet-audit:verify marker is pinned: the meso sender is an
fname_alt absent from the IDA export, so a second marker would orphan under
`matrix --check` (Task 3 rationale). Fixture:
`libs/atlas-packet/character/serverbound/attack_request_test.go#TestAttackMeleeRequestMesoExplosion`.
```

- [ ] **Step 2: Append the unverifiable-tail note to the jms audit MD**

Append to the end of `docs/packets/audits/jms_v185/CharacterAttackMeleeRequest.md`:

```markdown

---

## task-150 note — Meso Explosion variant (hand-added; keep on regeneration)

CLOSE_RANGE_ATTACK carries a Meso Explosion (4211006) variant written by a
dedicated sender, `sub_A3AAB1` @ `0xa3aab1` in the jms IDB. The sender's
packet-encode tail is SCY code-flow-virtualized (`JUMPOUT(0xD29D2D)`), so the
jms serverbound variant read order is **not statically verifiable** in the
available dump. Atlas implements the jms variant from the deltas verified
byte-identical across the eight GMS versions (gms_v48–v95) plus the jms
**clientbound** meso branch (`CUserRemote::OnAttack` @ `0xa53999` region),
which was IDA-verified to match. No verify marker or evidence record was added
for the unreadable tail (task-150 design §2.3). gms_v92 (no IDB) follows the
GMS >= 87 family branch and gms_12 the very-legacy GMS < 48 branch; both are
template-only and unverified (design §2.4). Fixture:
`libs/atlas-packet/character/serverbound/attack_request_test.go#TestAttackMeleeRequestMesoExplosion`.
```

- [ ] **Step 3: Regenerate the matrix and run the machine check**

Run (from the worktree root):

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Expected: `matrix` leaves STATUS.md/status.json unchanged (or trivially regenerated); `matrix --check` introduces **no new problems** — zero orphan/dangling/stale/drift lines mentioning `CharacterAttackMeleeRequest` or `character/serverbound`, and the pre-existing conflict count does not increase (per VERIFYING_A_PACKET.md §8, pre-existing 🟥 conflicts may keep the exit code at 1 — the bar is no new lines).

- [ ] **Step 4: Commit**

```bash
git add docs/packets/audits/gms_v48/CharacterAttackMeleeRequest.md \
        docs/packets/audits/gms_v61/CharacterAttackMeleeRequest.md \
        docs/packets/audits/gms_v72/CharacterAttackMeleeRequest.md \
        docs/packets/audits/gms_v79/CharacterAttackMeleeRequest.md \
        docs/packets/audits/jms_v185/CharacterAttackMeleeRequest.md \
        docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "docs(task-150): meso-variant audit notes (4 legacy verified + jms/v92/v12 unverified)"
```

(If `matrix` changed nothing, drop STATUS.md/status.json and commit only the MDs.)

---

### Task 8: Full verification sweep (PRD AC-6)

**Files:** none (verification only; fix-and-recommit anything that fails).

- [ ] **Step 1: Test/vet/build both changed modules**

```bash
cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./...
cd ../../services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...
```

Expected: all clean.

- [ ] **Step 2: Redis key guard**

Run from the worktree root (no `GOWORK=off` prefix — see CLAUDE.md):

```bash
tools/redis-key-guard.sh
```

Expected: clean exit 0.

- [ ] **Step 3: Docker bake**

Run from the worktree root:

```bash
docker buildx bake atlas-channel
```

Expected: builds green. (atlas-drops is untouched by design — bake it only if a Task forced a change there, which this plan forbids.)

- [ ] **Step 4: Confirm atlas-drops is untouched**

Run: `git diff main --stat -- services/atlas-drops`
Expected: no output (FR-9/design §5.3).

- [ ] **Step 5: Request code review before PR**

Invoke `superpowers:requesting-code-review` (dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer`); findings land in `docs/tasks/task-150-meso-explosion/audit.md`. Address findings, then proceed to `superpowers:finishing-a-development-branch`.

**Note on PRD AC-2 (integration):** the in-game pass — drop mesos on a v83 tenant, detonate, observe the explode animation from a second session and monster damage; then a negative test with a forged drop id confirmed via the validation warn log — is a manual step performed by the owner on a deployed environment (design §7). It is not automatable in this plan; flag it in the PR description as the remaining acceptance evidence.

---

## Out of scope (do not implement)

- Pick Pocket (4211003) — separate TODO, stays.
- Server-side damage recomputation from meso amounts (owner decision: client-trusted).
- Staggered/delayed drop removal (owner decision: immediate; the client animation covers presentation).
- Meso credit to the attacker — exploded mesos are purely destroyed (CONSUME never credits).
- Any atlas-drops change, including a meso-only consume guard (would break the reactor caller — design §5.3).
- Any other TODO in the `character_attack_common.go` block.
