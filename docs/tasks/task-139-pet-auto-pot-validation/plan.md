# Pet Auto-Pot Validation & Pet Skill Pouches — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Validate `PET_AUTO_POT` requests in atlas-channel (specific pet, alive, version-family skill gate) and implement the pet-skill pouch system (0519 items → persisted pet flag → client sync) that makes the JMS gate satisfiable.

**Architecture:** The socket handler `PetItemUseHandle` becomes a validation pipeline (parallel REST fetches → cheapest-first checks → unstick+warn on reject, unchanged pass-through on success). The FR-3 gate is config-selected per tenant (`skillGate: equipAbility` for GMS = worn pet-ability equips; `skillGate: petSkillFlag` for JMS = pouch-taught flag). The pouch system flows: channel cash-item-use type-28 arm → consumables `ConsumePetSkillPouch` → pets `SET_SKILL` command → `FLAG_CHANGED` event → channel re-announces the pet asset with a config-resolved `usPetSkill` wire mask (DOM-25).

**Tech Stack:** Go microservices (atlas-channel, atlas-consumables, atlas-pets, atlas-data, atlas-configurations seed templates), shared libs (atlas-constants, atlas-packet), Kafka commands/events, JSON:API REST, IDA Pro MCP for the two remaining byte-level verifications.

## Global Constraints

> **Rebase revision (branch updated onto main).** Main now routes
> `PetItemUseHandle` on **eight** versions — gms_61 (0x8E), gms_72 (0xA5),
> gms_79 (0xA7), gms_83 (0xAB), gms_84 (0xB0), gms_87 (0xB7), gms_95 (0xCB),
> jms_185 (0xAE) — not the five this plan was written against, and **gms_48 is a
> ninth** whose client speaks the packet (opcode 0x75) but whose template never
> routed it (design §1.7). All legacy GMS clients use the **same** worn-equip
> ability gate as v83 (design §1.1, IDA-verified per version), so no new gate
> mode is needed. What changes is coverage: Task 3 (per-version pet trailer),
> Task 12 (petId-less pet resolution on v48), Task 13 (nine templates, one of
> them a new entry; gms_95 validator step dropped — already fixed on main) and
> the new Task 15 (codec version gate, fixtures, matrix). Only the partial
> gms_12/gms_92 templates are out of scope, on evidence (design §3.6).

- **Verification gates (CLAUDE.md):** `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake atlas-<svc>` for every service whose `go.mod` was touched; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` and `tools/lint.sh --check` clean from repo root; `tools/template-opcode-order-guard.sh` clean once Task 13 touches the seed templates. (The last three gates postdate the original plan — see the updated CLAUDE.md build section.)
- **Every routed version gets configured.** The FR-3 gate fails closed, so a template left without `skillGate` rejects all auto-pot on that version. "Five versions" anywhere in this plan means eight; treat any per-version list below as authoritative only where it was revised in this pass.
- **DOM-25:** client-interpreted wire values (usPetSkill bits) resolve from tenant writer-options tables; domain services store only Atlas-canonical semantic bits. Never hardcode client bit values in Go.
- **No inventing:** the two IDA-dependent items (jms type-28 sub-body layout, per-version usPetSkill bits) are verified in their tasks before any value is written down. A bit with no IDA confirmation is omitted from the template table, never guessed.
- **No TODOs / stubs** in any commit. Builder pattern for test setup (no `*_testhelpers.go`).
- **Worktree:** all work happens in `.worktrees/task-139-pet-auto-pot-validation` on branch `task-139-pet-auto-pot-validation`. Verify with `git branch --show-current` after each commit.
- **Immutable models:** private fields + getters + Clone/Builder, matching each package's existing style.
- Log rejections at **warn** with characterId, petId, itemId, slot, buffSkill, and a machine-readable reason. Success paths stay silent beyond existing debug lines.

**Design deviations discovered during planning** (all verified against source/WZ; see `context.md` for detail):
1. The 0519 skill keys and `add` flag live under the item's **`info`** node (design said spec/info; it is info only — verified in v83 `Item.wz/Cash/0519.img.xml`).
2. Pet-equip ability attributes in `Character.wz/PetEquip` are **string-typed** `"0"/"1"` (use the xml `GetBool` string-fallback helpers).
3. atlas-channel has **no** `data/consumable` or `data/equipment` client packages — Task 10 creates them.
4. `ResolveCode` is byte-only; usPetSkill bits reach 0x100 → Task 2 adds a uint16 variant.
5. `SetPetInfo` keeps its signature; the flag travels via a new additive `SetPetFlag` (avoids a cross-module breaking change).
6. In the raw equip compartment, worn **cash** equips sit at `position - 100` (e.g. petHP −24 is stored as −124) — the gate normalizes positions (see `SetInventory`, `services/atlas-channel/atlas.com/channel/character/model.go:284`).
7. gms_95_1 has **eight** validator-less pet handlers (0x52, 0x6E, 0xC7–0xCC), not just `PetItemUseHandle` — Task 13 fixes the whole block.

---

### Task 1: atlas-constants — Classification 519 + `pet/skill` package

**Files:**
- Modify: `libs/atlas-constants/item/constants.go` (cash classification block, after `ClassificationPetImprints`)
- Create: `libs/atlas-constants/pet/skill/constants.go`
- Test: `libs/atlas-constants/pet/skill/constants_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `item.ClassificationPetSkill` (`Classification(519)`); package `github.com/Chronicle20/atlas/libs/atlas-constants/pet/skill` with `type Key string`, `type Flag uint16`, the nine keys, `All() []Key`, `BitFor(Key) (Flag, bool)`, `Has(uint16, Key) bool`, `Apply(uint16, Key, bool) uint16`. Later tasks reference these exact names.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-constants/pet/skill/constants_test.go`:

```go
package skill_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/pet/skill"
)

func TestAllOrderAndBits(t *testing.T) {
	// Order and bit assignment are Atlas-canonical storage semantics (design §3.5):
	// the nine 0519 WZ spec keys, bits 1<<0 .. 1<<8 in that order.
	want := []struct {
		key skill.Key
		bit skill.Flag
	}{
		{skill.PickupItem, 1 << 0},
		{skill.ConsumeHP, 1 << 1},
		{skill.LongRange, 1 << 2},
		{skill.DropSweep, 1 << 3},
		{skill.PickupAll, 1 << 4},
		{skill.IgnorePickup, 1 << 5},
		{skill.ConsumeMP, 1 << 6},
		{skill.Recall, 1 << 7},
		{skill.AutoSpeaking, 1 << 8},
	}
	all := skill.All()
	if len(all) != len(want) {
		t.Fatalf("All() len = %d, want %d", len(all), len(want))
	}
	for i, w := range want {
		if all[i] != w.key {
			t.Errorf("All()[%d] = %q, want %q", i, all[i], w.key)
		}
		bit, ok := skill.BitFor(w.key)
		if !ok || bit != w.bit {
			t.Errorf("BitFor(%q) = %v,%v, want %v,true", w.key, bit, ok, w.bit)
		}
	}
}

func TestBitForUnknown(t *testing.T) {
	if _, ok := skill.BitFor(skill.Key("bogus")); ok {
		t.Error("BitFor(bogus) ok = true, want false")
	}
}

func TestHasApply(t *testing.T) {
	var f uint16
	f = skill.Apply(f, skill.ConsumeHP, true)
	if !skill.Has(f, skill.ConsumeHP) {
		t.Error("Has(consumeHP) = false after Apply(true)")
	}
	if skill.Has(f, skill.ConsumeMP) {
		t.Error("Has(consumeMP) = true, want false")
	}
	// idempotent set
	if skill.Apply(f, skill.ConsumeHP, true) != f {
		t.Error("Apply(true) not idempotent")
	}
	f = skill.Apply(f, skill.ConsumeHP, false)
	if skill.Has(f, skill.ConsumeHP) {
		t.Error("Has(consumeHP) = true after Apply(false)")
	}
	// Apply on unknown key is a no-op
	if skill.Apply(f, skill.Key("bogus"), true) != f {
		t.Error("Apply(unknown) mutated the mask")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./pet/skill/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write the implementation**

Create `libs/atlas-constants/pet/skill/constants.go`:

```go
// Package skill defines the semantic pet skills taught/removed by 0519 pet
// skill pouch items, and the Atlas-canonical flag bits used to persist them on
// the pet model. These bits are Atlas-internal storage semantics, deliberately
// decoupled from client wire bits (usPetSkill), which are version-dependent
// and resolve from tenant configuration (DOM-25).
package skill

// Key is the semantic identifier of a pet skill, spelled exactly as the 0519
// item WZ keys. The pet-equip family spells DropSweep as "sweepForDrop"; the
// 0519 pouch family calls the same ability "dropSweep".
type Key string

const (
	PickupItem   = Key("pickupItem")
	ConsumeHP    = Key("consumeHP")
	LongRange    = Key("longRange")
	DropSweep    = Key("dropSweep")
	PickupAll    = Key("pickupAll")
	IgnorePickup = Key("ignorePickup")
	ConsumeMP    = Key("consumeMP")
	Recall       = Key("recall")
	AutoSpeaking = Key("autoSpeaking")
)

// Flag is the Atlas-canonical pet skill bitmask persisted on the pet model.
type Flag uint16

const (
	FlagPickupItem   = Flag(1 << 0)
	FlagConsumeHP    = Flag(1 << 1)
	FlagLongRange    = Flag(1 << 2)
	FlagDropSweep    = Flag(1 << 3)
	FlagPickupAll    = Flag(1 << 4)
	FlagIgnorePickup = Flag(1 << 5)
	FlagConsumeMP    = Flag(1 << 6)
	FlagRecall       = Flag(1 << 7)
	FlagAutoSpeaking = Flag(1 << 8)
)

var ordered = []Key{PickupItem, ConsumeHP, LongRange, DropSweep, PickupAll, IgnorePickup, ConsumeMP, Recall, AutoSpeaking}

var bits = map[Key]Flag{
	PickupItem:   FlagPickupItem,
	ConsumeHP:    FlagConsumeHP,
	LongRange:    FlagLongRange,
	DropSweep:    FlagDropSweep,
	PickupAll:    FlagPickupAll,
	IgnorePickup: FlagIgnorePickup,
	ConsumeMP:    FlagConsumeMP,
	Recall:       FlagRecall,
	AutoSpeaking: FlagAutoSpeaking,
}

// All returns the nine skills in canonical bit order.
func All() []Key {
	res := make([]Key, len(ordered))
	copy(res, ordered)
	return res
}

// BitFor returns the canonical flag bit for a semantic key.
func BitFor(k Key) (Flag, bool) {
	b, ok := bits[k]
	return b, ok
}

// Has reports whether the mask contains the skill.
func Has(mask uint16, k Key) bool {
	b, ok := bits[k]
	return ok && mask&uint16(b) != 0
}

// Apply sets or clears the skill on the mask. Unknown keys are a no-op.
func Apply(mask uint16, k Key, enabled bool) uint16 {
	b, ok := bits[k]
	if !ok {
		return mask
	}
	if enabled {
		return mask | uint16(b)
	}
	return mask &^ uint16(b)
}
```

Modify `libs/atlas-constants/item/constants.go` — insert after the `ClassificationPetImprints = Classification(517)` line (keep alignment with the block):

```go
	ClassificationPetSkill                 = Classification(519)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-constants && go test -race ./... && go vet ./...`
Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-constants/item/constants.go libs/atlas-constants/pet/skill/
git commit -m "feat(constants): add ClassificationPetSkill(519) and pet/skill semantic keys+flags"
```

---

### Task 2: atlas-packet — `ResolveCode16` (uint16 option codes)

**Files:**
- Modify: `libs/atlas-packet/resolve.go`
- Test: `libs/atlas-packet/resolve_test.go` (create if absent; check first with `ls libs/atlas-packet/*_test.go`)

**Interfaces:**
- Consumes: nothing new.
- Produces: `atlas_packet.ResolveCode16(l logrus.FieldLogger, options map[string]interface{}, property string, key string) (uint16, bool)` — soft-miss lookup (returns `(0,false)` + debug log when property/key absent). Task 3 uses it.

- [ ] **Step 1: Write the failing test**

Add to `libs/atlas-packet/resolve_test.go` (create the file with `package atlas_packet` if it does not exist; if it exists, append):

```go
package atlas_packet

import (
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

func TestResolveCode16(t *testing.T) {
	l, _ := test.NewNullLogger()
	options := map[string]interface{}{
		"petSkill": map[string]interface{}{
			"consumeHP":    "0x20",
			"autoSpeaking": "0x100",
			"asNumber":     float64(64),
			"bad":          "zzz",
		},
	}

	if v, ok := ResolveCode16(l, options, "petSkill", "consumeHP"); !ok || v != 0x20 {
		t.Errorf("consumeHP = %#x,%v; want 0x20,true", v, ok)
	}
	if v, ok := ResolveCode16(l, options, "petSkill", "autoSpeaking"); !ok || v != 0x100 {
		t.Errorf("autoSpeaking = %#x,%v; want 0x100,true", v, ok)
	}
	if v, ok := ResolveCode16(l, options, "petSkill", "asNumber"); !ok || v != 64 {
		t.Errorf("asNumber = %d,%v; want 64,true", v, ok)
	}
	// soft misses: absent key, absent property, unparseable value
	if _, ok := ResolveCode16(l, options, "petSkill", "recall"); ok {
		t.Error("absent key resolved ok=true, want false")
	}
	if _, ok := ResolveCode16(l, options, "nope", "consumeHP"); ok {
		t.Error("absent property resolved ok=true, want false")
	}
	if _, ok := ResolveCode16(l, options, "petSkill", "bad"); ok {
		t.Error("unparseable value resolved ok=true, want false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test -run TestResolveCode16 ./... -v`
Expected: FAIL — `ResolveCode16` undefined.

- [ ] **Step 3: Write the implementation**

Append to `libs/atlas-packet/resolve.go`:

```go
// ResolveCode16 looks up an optional uint16 code from the runtime options map.
// Unlike ResolveCode — which returns a loud 99 default because a missing mode
// byte is a fatal misconfiguration — a miss here is soft: the caller decides
// what absence means. Used for sparse bit tables (e.g. the petSkill usPetSkill
// table) where an unverified bit must encode as absent, never a guessed value.
func ResolveCode16(l logrus.FieldLogger, options map[string]interface{}, property string, key string) (uint16, bool) {
	genericCodes, ok := options[property]
	if !ok {
		l.Debugf("Property [%s] missing from options when resolving code [%s].", property, key)
		return 0, false
	}

	codes, ok := genericCodes.(map[string]interface{})
	if !ok {
		l.Debugf("Property [%s] is not a map when resolving code [%s].", property, key)
		return 0, false
	}

	raw, ok := codes[key]
	if !ok {
		return 0, false
	}

	switch v := raw.(type) {
	case float64:
		return uint16(v), true
	case string:
		n, err := strconv.ParseUint(v, 0, 16)
		if err != nil {
			l.WithError(err).Debugf("Code [%s] in property [%s] has unparseable value [%q].", key, property, v)
			return 0, false
		}
		return uint16(n), true
	default:
		l.Debugf("Code [%s] in property [%s] has unsupported type %T.", key, property, raw)
		return 0, false
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test -race -run TestResolveCode16 ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/resolve.go libs/atlas-packet/resolve_test.go
git commit -m "feat(atlas-packet): ResolveCode16 for sparse uint16 option tables"
```

---

### Task 3: atlas-packet — pet asset `petFlag` + config-resolved `usPetSkill` encode

**Files:**
- Modify: `libs/atlas-packet/model/asset.go` (field, getter, setter, `encodePetCashItemInfo`)
- Test: `libs/atlas-packet/model/asset_test.go` (append)

**Interfaces:**
- Consumes: `atlas_packet.ResolveCode16` (Task 2), `pet/skill` constants (Task 1).
- Produces: `Asset.SetPetFlag(flag uint16) Asset`, `Asset.PetFlag() uint16`. The pet cash-item encode replaces `w.WriteShort(0) // skill` with the wire mask resolved from writer options property `"petSkill"` (semantic key → uint16 wire bit). `SetPetInfo`'s signature is unchanged.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-packet/model/asset_test.go`:

```go
func TestAssetPetCashItemSkillMask(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx := context.Background()
	expiration := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	base := NewAsset(true, 0, 5000017, expiration).
		SetCashId(123).
		SetPetInfo(42, "Mr. Roboto", 3, 100, 50)

	// Layout of encodePetCashItemInfo with zeroPosition=true:
	// type(1) templateId(4) bool(1) petId(8) time(8) name(13) level(1)
	// closeness(2) fullness(1) expiration(8) attribute(2) => skill short at offset 49.
	const skillOffset = 49

	zeroFlag := base.Encode(l, ctx)(map[string]interface{}{})

	// 1. petFlag set but no petSkill table configured -> byte-identical to zero-flag encode.
	flagged := base.SetPetFlag(2) // FlagConsumeHP (1<<1), Atlas-canonical
	noTable := flagged.Encode(l, ctx)(map[string]interface{}{})
	if !bytes.Equal(zeroFlag, noTable) {
		t.Fatal("petFlag with no petSkill table must encode byte-identical to zero flag")
	}

	// 2. petFlag set with a configured table -> wire bit at the skill short.
	withTable := flagged.Encode(l, ctx)(map[string]interface{}{
		"petSkill": map[string]interface{}{"consumeHP": "0x20"},
	})
	if len(withTable) != len(zeroFlag) {
		t.Fatalf("length changed: got %d, want %d", len(withTable), len(zeroFlag))
	}
	if withTable[skillOffset] != 0x20 || withTable[skillOffset+1] != 0x00 {
		t.Errorf("skill short = %#x %#x, want 0x20 0x00", withTable[skillOffset], withTable[skillOffset+1])
	}
	// everything else unchanged
	for i := range zeroFlag {
		if i == skillOffset || i == skillOffset+1 {
			continue
		}
		if withTable[i] != zeroFlag[i] {
			t.Fatalf("byte %d changed: got %#x, want %#x", i, withTable[i], zeroFlag[i])
		}
	}

	// 3. multiple flags OR together (autoSpeaking 1<<8 canonical -> 0x100 wire).
	multi := base.SetPetFlag(2 | 256).Encode(l, ctx)(map[string]interface{}{
		"petSkill": map[string]interface{}{"consumeHP": "0x20", "autoSpeaking": "0x100"},
	})
	if multi[skillOffset] != 0x20 || multi[skillOffset+1] != 0x01 {
		t.Errorf("multi skill short = %#x %#x, want 0x20 0x01", multi[skillOffset], multi[skillOffset+1])
	}

	if got := flagged.PetFlag(); got != 2 {
		t.Errorf("PetFlag() = %d, want 2", got)
	}
}
```

Add the imports the test file needs if not already present (`bytes` is the likely addition).

**Revised for step 3b:** `encodePetCashItemInfo` now reads the tenant from the context, so `ctx := context.Background()` above must become a real tenant context (`pt.CreateContext("GMS", 83, 1)` — the helper the packet tests already use). Build the same test body per version and assert the trailer lengths listed in step 3b; any pre-existing pet-asset test that passes a bare `context.Background()` needs the same treatment or it will panic in `tenant.MustFromContext`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test -run TestAssetPetCashItemSkillMask ./model/... -v`
Expected: FAIL — `SetPetFlag` undefined.

- [ ] **Step 3: Write the implementation**

In `libs/atlas-packet/model/asset.go`:

1. Add the field next to the other pet fields (`petId`, `petName`, `petLevel`, `closeness`, `fullness` around line 49):

```go
	petFlag uint16
```

2. Add getter/setter next to `SetPetInfo` (line ~160):

```go
func (m Asset) PetFlag() uint16 { return m.petFlag }

// SetPetFlag sets the Atlas-canonical pet skill mask (see atlas-constants
// pet/skill). The wire encoding translates it through the tenant's petSkill
// options table at encode time — canonical bits never hit the wire directly.
func (m Asset) SetPetFlag(flag uint16) Asset {
	m.petFlag = flag
	return m
}
```

3. In `encodePetCashItemInfo` (`libs/atlas-packet/model/asset.go:337-359` on current main) replace:

```go
		w.WriteShort(0)   // skill
```

with:

```go
		w.WriteShort(resolvePetSkillWireMask(l, options, m.petFlag))
```

Note: the closure signature is `func(options map[string]interface{}) []byte` and `l` is captured from the method args — the method is currently `l logrus.FieldLogger, _ context.Context`; keep `l` **and** name the context parameter, because step 3b needs the tenant.

3b. **Version-gate the trailer (design §1.6 — pre-existing main defect, fixed here because this task edits the function).** `GW_ItemSlotPet::RawDecode` reads fewer trailing fields on the legacy clients: v61 stops after `usPetSkill`; v72 adds `remainLife` but not the trailing `attribute`; v79 and later (and JMS) read all four. Today Atlas always writes all four — 6 bytes of overrun against v61, 2 against v72. Replace the tail with:

```go
		w.WriteShort(0) // petAttribute
		w.WriteShort(resolvePetSkillWireMask(l, options, m.petFlag)) // usPetSkill
		// GW_ItemSlotPet::RawDecode gained remainLife in the v72 revision and the
		// trailing attribute short in the v79 revision: v61 reads neither
		// (@0x4b52f2), v72 reads remainLife only (@0x4d06dd), v79 (@0x4d84c4) and
		// v83 (@0x4e4219) read both. IDA-verified.
		if (t.IsRegion("GMS") && t.MajorAtLeast(72)) || t.Region() == "JMS" {
			w.WriteInt(18000) // remaining life
		}
		if (t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS" {
			w.WriteShort(0) // attribute
		}
```

with `t := tenant.MustFromContext(ctx)` at the top of the method, mirroring `encodeEquipableInfo` (`asset.go:200-260`) — same file, same idiom, including the `MajorAtLeast` rule (never a raw `> N`; see `bug_majorversion_gt83_is_off_by_one_v87`). JMS keeps today's full trailer: jms_185 is not a legacy client and its verified fixtures must not move.

Add to the Step-1 test file a per-version length assertion driven by the existing `pt.CreateContext` helper (the pattern `libs/atlas-packet/pet/serverbound/item_use_test.go` uses): v61 encode is 6 bytes shorter than v83, v72 is 2 shorter, v79/v83/v84/v87/v95/jms are unchanged from today's bytes. The unchanged-versions assertion is the regression guard — if any of those move, the change is wrong.

4. Add the helper at the bottom of the file:

```go
// resolvePetSkillWireMask translates the Atlas-canonical pet skill mask into
// the tenant's client usPetSkill bits via the writer options "petSkill" table
// (DOM-25: the wire bit values are client-interpreted and version-dependent).
// Semantic bits with no table entry encode as absent — never a guessed value.
func resolvePetSkillWireMask(l logrus.FieldLogger, options map[string]interface{}, petFlag uint16) uint16 {
	if petFlag == 0 {
		return 0
	}
	var wire uint16
	for _, k := range petskill.All() {
		if !petskill.Has(petFlag, k) {
			continue
		}
		if v, ok := atlas_packet.ResolveCode16(l, options, "petSkill", string(k)); ok {
			wire |= v
		} else {
			l.Debugf("Pet skill [%s] set on pet but no petSkill wire bit configured; encoding as absent.", k)
		}
	}
	return wire
}
```

5. Add imports:

```go
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	petskill "github.com/Chronicle20/atlas/libs/atlas-constants/pet/skill"
```

(The root `atlas-packet` package does not import `model`, so this creates no cycle — verified.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test -race ./model/... && go vet ./...`
Expected: PASS (including the pre-existing `TestAssetPetCashItem`, which proves the zero-flag regression), vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/model/asset.go libs/atlas-packet/model/asset_test.go
git commit -m "feat(atlas-packet): encode pet usPetSkill from config-resolved petSkill table (DOM-25)"
```

---

### Task 4: atlas-data — parse 0519 pet-skill keys on cash items

**Files:**
- Modify: `services/atlas-data/atlas.com/data/cash/rest.go` (RestModel fields)
- Modify: `services/atlas-data/atlas.com/data/cash/reader.go` (info-node parsing)
- Test: `services/atlas-data/atlas.com/data/cash/reader_test.go` (append)

**Interfaces:**
- Consumes: `pet/skill.All()` (Task 1).
- Produces: cash item RestModel gains `PetSkills []string` (JSON `petSkills`, only keys present with truthy value) and `PetSkillAdd bool` (JSON `petSkillAdd`; `true` = grant, `false` = remove; only meaningful when `PetSkills` is non-empty). Task 7's consumables mirror matches these JSON names exactly.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-data/atlas.com/data/cash/reader_test.go` (fixture content is verbatim from v83 `Item.wz/Cash/0519.img.xml`, canvas nodes trimmed):

```go
const testPetSkillXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0519.img">
  <imgdir name="05190001">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="consumeHP" value="1"/>
      <int name="add" value="1"/>
    </imgdir>
  </imgdir>
  <imgdir name="05190006">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="consumeMP" value="1"/>
      <int name="add" value="1"/>
    </imgdir>
  </imgdir>
  <imgdir name="05191001">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="consumeHP" value="1"/>
      <int name="add" value="0"/>
    </imgdir>
  </imgdir>
</imgdir>
`

func TestReaderPetSkills(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testPetSkillXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	if len(rmm) != 3 {
		t.Fatalf("len(rmm) = %d, want 3", len(rmm))
	}

	cases := []struct {
		id     string
		skills []string
		add    bool
	}{
		{"5190001", []string{"consumeHP"}, true},
		{"5190006", []string{"consumeMP"}, true},
		{"5191001", []string{"consumeHP"}, false},
	}
	for _, c := range cases {
		rm, ok := rmm[c.id]
		if !ok {
			t.Fatalf("rmm[%s] does not exist", c.id)
		}
		if len(rm.PetSkills) != len(c.skills) || rm.PetSkills[0] != c.skills[0] {
			t.Errorf("[%s] PetSkills = %v, want %v", c.id, rm.PetSkills, c.skills)
		}
		if rm.PetSkillAdd != c.add {
			t.Errorf("[%s] PetSkillAdd = %t, want %t", c.id, rm.PetSkillAdd, c.add)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-data/atlas.com/data && go test -run TestReaderPetSkills ./cash/... -v`
Expected: FAIL — `PetSkills` undefined on RestModel.

- [ ] **Step 3: Write the implementation**

In `services/atlas-data/atlas.com/data/cash/rest.go`, add to `RestModel` after the `Spec` field:

```go
	PetSkills   []string `json:"petSkills,omitempty"`
	PetSkillAdd bool     `json:"petSkillAdd,omitempty"`
```

In `services/atlas-data/atlas.com/data/cash/reader.go`, inside the per-item loop after the time-windows block and before `s, err := cxml.ChildByName("spec")` (the `info` node variable in scope is `i`):

```go
			// 0519 pet skill pouches: the skill key(s) and add flag live under
			// info (not spec) — add=1 grants the skill, add=0 removes it.
			for _, k := range petskill.All() {
				if i.GetBool(string(k), false) {
					m.PetSkills = append(m.PetSkills, string(k))
				}
			}
			if len(m.PetSkills) > 0 {
				m.PetSkillAdd = i.GetBool("add", false)
			}
```

Add the import:

```go
	petskill "github.com/Chronicle20/atlas/libs/atlas-constants/pet/skill"
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-data/atlas.com/data && go test -race ./cash/... && go vet ./cash/...`
Expected: PASS (existing `TestReader*` untouched), vet clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/cash/
git commit -m "feat(atlas-data): expose 0519 pet skill keys and add flag on cash items"
```

---

### Task 5: atlas-data — parse pet-ability attributes on equipment

**Files:**
- Modify: `services/atlas-data/atlas.com/data/equipment/rest.go` (RestModel field)
- Modify: `services/atlas-data/atlas.com/data/equipment/reader.go` (info parsing)
- Test: `services/atlas-data/atlas.com/data/equipment/reader_test.go` (append)

**Interfaces:**
- Consumes: nothing new.
- Produces: equipment RestModel gains `PetAbilities []string` (JSON `petAbilities`, omitempty) listing the truthy pet-equip ability attributes out of the eight equip-family keys: `pickupMeso, pickupItem, pickupOthers, sweepForDrop, longRange, consumeHP, consumeMP, ignorePickup`. Task 10's channel client mirrors the JSON name.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-data/atlas.com/data/equipment/reader_test.go` (fixture verbatim from v83 `Character.wz/PetEquip/01812002.img.xml` info block, canvas trimmed — note the **string-typed** values):

```go
const testPetEquipXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="01812002.img">
  <imgdir name="info">
    <int name="reqLevel" value="0"/>
    <int name="cash" value="1"/>
    <string name="pickupMeso" value="0"/>
    <string name="pickupItem" value="0"/>
    <string name="pickupOthers" value="0"/>
    <string name="sweepForDrop" value="0"/>
    <string name="longRange" value="0"/>
    <string name="consumeHP" value="1"/>
  </imgdir>
</imgdir>
`

func TestReaderPetAbilities(t *testing.T) {
	l, _ := test.NewNullLogger()

	rm, err := Read(l)(xml.FromByteArrayProvider([]byte(testPetEquipXML)))()
	if err != nil {
		t.Fatal(err)
	}
	if len(rm.PetAbilities) != 1 || rm.PetAbilities[0] != "consumeHP" {
		t.Errorf("PetAbilities = %v, want [consumeHP]", rm.PetAbilities)
	}
}
```

(If the equipment `Read` provider signature differs — check `TestReader` at `reader_test.go:509` — mirror its invocation exactly; the assertion stays the same.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-data/atlas.com/data && go test -run TestReaderPetAbilities ./equipment/... -v`
Expected: FAIL — `PetAbilities` undefined.

- [ ] **Step 3: Write the implementation**

In `services/atlas-data/atlas.com/data/equipment/rest.go`, add to `RestModel` after `BonusExp`:

```go
	PetAbilities []string `json:"petAbilities,omitempty"`
```

In `services/atlas-data/atlas.com/data/equipment/reader.go`, add to the `RestModel{...}` literal (after `BonusExp: bonusExpTiers,`):

```go
			PetAbilities:   readPetAbilities(info),
```

and add at the bottom of the file:

```go
// petAbilityKeys are the worn-pet-equip ability attributes the client ORs into
// dwPetAbilityFlag (CPet::UpdatePetAbility). The equip family spells sweep as
// sweepForDrop; the 0519 pouch family calls the same ability dropSweep.
var petAbilityKeys = []string{"pickupMeso", "pickupItem", "pickupOthers", "sweepForDrop", "longRange", "consumeHP", "consumeMP", "ignorePickup"}

// readPetAbilities extracts the truthy pet-ability attributes. They are
// string-typed "0"/"1" in Character.wz/PetEquip; GetBool handles the fallback.
func readPetAbilities(info *xml.Node) []string {
	var res []string
	for _, k := range petAbilityKeys {
		if info.GetBool(k, false) {
			res = append(res, k)
		}
	}
	return res
}
```

(Match `readPetAbilities`'s parameter type to whatever `info` is in that scope — the same node type `GetShort` is called on.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-data/atlas.com/data && go test -race ./equipment/... && go vet ./equipment/...`
Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/equipment/
git commit -m "feat(atlas-data): expose pet-ability equip attributes (consumeHP/consumeMP et al)"
```

---

### Task 6: atlas-pets — `SET_SKILL` command, flag persistence, `FLAG_CHANGED` event

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/kafka/message/pet/kafka.go`
- Modify: `services/atlas-pets/atlas.com/pets/pet/administrator.go`
- Modify: `services/atlas-pets/atlas.com/pets/pet/producer.go`
- Modify: `services/atlas-pets/atlas.com/pets/pet/processor.go` (interface + impl)
- Modify: `services/atlas-pets/atlas.com/pets/kafka/consumer/pet/consumer.go`
- Test: `services/atlas-pets/atlas.com/pets/pet/processor_test.go` (append)

**Interfaces:**
- Consumes: `pet/skill.BitFor/Apply` (Task 1). The pet entity `Flag` column already exists (`pet/entity.go`, `gorm:"not null;default:0"`) — no schema change.
- Produces: Kafka command type `SET_SKILL` on `COMMAND_TOPIC_PET` with body `{skill string, enabled bool}`; status event `FLAG_CHANGED` on `EVENT_TOPIC_PET_STATUS` with body `{slot int8, flag uint16}`; processor methods `SetSkillAndEmit(petId uint32, skillKey string, enabled bool) error` and `SetSkill(mb *message.Buffer) func(petId uint32) func(skillKey string) func(enabled bool) error`.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-pets/atlas.com/pets/pet/processor_test.go` (uses the file's existing `testLogger/testContext/testDatabase/mustBuild` helpers):

```go
func TestProcessor_SetSkill(t *testing.T) {
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t))

	mb := message.NewBuffer()
	i := mustBuild(t, pet.NewModelBuilder(0, 7000001, 5000017, "Skiller", 1))
	o, err := p.Create(mb)(i)
	if err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	// enable consumeHP (canonical bit 1<<1)
	if err := p.SetSkill(message.NewBuffer())(o.Id())("consumeHP")(true); err != nil {
		t.Fatalf("SetSkill enable: %v", err)
	}
	m, err := p.GetById(o.Id())
	if err != nil {
		t.Fatal(err)
	}
	if m.Flag() != 2 {
		t.Errorf("Flag = %d, want 2", m.Flag())
	}

	// idempotent re-enable: no error, unchanged
	if err := p.SetSkill(message.NewBuffer())(o.Id())("consumeHP")(true); err != nil {
		t.Fatalf("SetSkill idempotent enable: %v", err)
	}
	m, _ = p.GetById(o.Id())
	if m.Flag() != 2 {
		t.Errorf("Flag after idempotent enable = %d, want 2", m.Flag())
	}

	// enable a second skill; both bits present
	if err := p.SetSkill(message.NewBuffer())(o.Id())("autoSpeaking")(true); err != nil {
		t.Fatalf("SetSkill autoSpeaking: %v", err)
	}
	m, _ = p.GetById(o.Id())
	if m.Flag() != 2|256 {
		t.Errorf("Flag = %d, want %d", m.Flag(), 2|256)
	}

	// disable consumeHP
	if err := p.SetSkill(message.NewBuffer())(o.Id())("consumeHP")(false); err != nil {
		t.Fatalf("SetSkill disable: %v", err)
	}
	m, _ = p.GetById(o.Id())
	if m.Flag() != 256 {
		t.Errorf("Flag after disable = %d, want 256", m.Flag())
	}

	// unknown key: warn + drop, no error, no change
	if err := p.SetSkill(message.NewBuffer())(o.Id())("bogus")(true); err != nil {
		t.Fatalf("SetSkill unknown key returned error: %v", err)
	}
	m, _ = p.GetById(o.Id())
	if m.Flag() != 256 {
		t.Errorf("Flag after unknown key = %d, want 256", m.Flag())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-pets/atlas.com/pets && go test -run TestProcessor_SetSkill ./pet/... -v`
Expected: FAIL — `SetSkill` undefined.

- [ ] **Step 3: Write the implementation**

`kafka/message/pet/kafka.go` — add to the command-type const block:

```go
	CommandSetSkill          = "SET_SKILL"
```

add after `EvolveCommandBody`:

```go
// SetSkillCommandBody carries a semantic pet skill key (atlas-constants
// pet/skill spelling) — never a client wire bit.
type SetSkillCommandBody struct {
	Skill   string `json:"skill"`
	Enabled bool   `json:"enabled"`
}
```

add to the status-event const block:

```go
	StatusEventTypeFlagChanged      = "FLAG_CHANGED"
```

add after `ExcludeChangedStatusEventBody`:

```go
type FlagChangedStatusEventBody struct {
	Slot int8   `json:"slot"`
	Flag uint16 `json:"flag"`
}
```

`pet/administrator.go` — add after `updateSlot` (same shape):

```go
func updateFlag(db *gorm.DB) func(petId uint32, flag uint16) error {
	return func(petId uint32, flag uint16) error {
		result := db.Model(&Entity{}).
			Where("id = ?", petId).
			Update("flag", flag)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("no entity found or flag is already set to the given value")
		}

		return nil
	}
}
```

`pet/producer.go` — add:

```go
func flagChangedEventProvider(m Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet.StatusEvent[pet.FlagChangedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet.StatusEventTypeFlagChanged,
		Body: pet.FlagChangedStatusEventBody{
			Slot: m.Slot(),
			Flag: m.Flag(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

`pet/processor.go` — add to the `Processor` interface (next to the `SetExclude` entries):

```go
	SetSkillAndEmit(petId uint32, skillKey string, enabled bool) error
	SetSkill(mb *message.Buffer) func(petId uint32) func(skillKey string) func(enabled bool) error
```

add the implementation (near `AwardFullness`, same transactional shape):

```go
func (p *ProcessorImpl) SetSkillAndEmit(petId uint32, skillKey string, enabled bool) error {
	return message.Emit(p.kp)(func(mb *message.Buffer) error {
		return p.SetSkill(mb)(petId)(skillKey)(enabled)
	})
}

func (p *ProcessorImpl) SetSkill(mb *message.Buffer) func(petId uint32) func(skillKey string) func(enabled bool) error {
	return func(petId uint32) func(skillKey string) func(enabled bool) error {
		return func(skillKey string) func(enabled bool) error {
			return func(enabled bool) error {
				if _, ok := petskill.BitFor(petskill.Key(skillKey)); !ok {
					p.l.Warnf("Received SET_SKILL for pet [%d] with unknown skill key [%s]. Dropping.", petId, skillKey)
					return nil
				}
				txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
					pe, err := p.With(WithTransaction(tx)).GetById(petId)
					if err != nil {
						return err
					}
					newFlag := petskill.Apply(pe.Flag(), petskill.Key(skillKey), enabled)
					if newFlag == pe.Flag() {
						return nil
					}
					if err := updateFlag(tx)(petId, newFlag); err != nil {
						return err
					}
					pe, err = Clone(pe).SetFlag(newFlag).Build()
					if err != nil {
						return err
					}
					return mb.Put(pet.EnvStatusEventTopic, flagChangedEventProvider(pe))
				})
				if txErr != nil {
					p.l.WithError(txErr).Errorf("Unable to set skill [%s] to [%t] for pet [%d].", skillKey, enabled, petId)
					return txErr
				}
				p.l.Infof("Set skill [%s] to [%t] for pet [%d].", skillKey, enabled, petId)
				return nil
			}
		}
	}
}
```

with import `petskill "github.com/Chronicle20/atlas/libs/atlas-constants/pet/skill"`.

`kafka/consumer/pet/consumer.go` — register in `InitHandlers` after the `handleSetExcludeCommand` registration:

```go
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSetSkillCommand(db)))); err != nil {
				return err
			}
```

and add the handler:

```go
func handleSetSkillCommand(db *gorm.DB) message.Handler[pet2.Command[pet2.SetSkillCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pet2.Command[pet2.SetSkillCommandBody]) {
		if c.Type != pet2.CommandSetSkill {
			return
		}
		err := pet.NewProcessor(l, ctx, db).SetSkillAndEmit(c.PetId, c.Body.Skill, c.Body.Enabled)
		if err != nil {
			l.WithError(err).Errorf("Unable to set skill [%s] for pet [%d].", c.Body.Skill, c.PetId)
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-pets/atlas.com/pets && go test -race ./... && go vet ./... && go build ./...`
Expected: PASS, clean. (If a `pet` Processor mock exists beyond `pet/mock/temporal.go` and fails to compile against the widened interface, add the two methods there with no-op bodies matching the mock's existing style.)

- [ ] **Step 5: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/
git commit -m "feat(atlas-pets): SET_SKILL command persists pet skill flag, emits FLAG_CHANGED"
```

---

### Task 7: atlas-consumables — 0519 pouch consume branch

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/message/pet/kafka.go` (SET_SKILL mirror + PetId type alignment)
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go` (`RequestItemConsumeBody.PetId`)
- Modify: `services/atlas-consumables/atlas.com/consumables/pet/processor.go`, `pet/producer.go` (SetSkill producer; AwardFullness petId type)
- Modify: `services/atlas-consumables/atlas.com/consumables/cash/rest.go`, `cash/model.go` (petSkills mirror)
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` (branch + `ConsumePetSkillPouch` + `ErrPetCannotLearn`)
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go` (pass PetId)
- Test: `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go` (append)

**Interfaces:**
- Consumes: `item.ClassificationPetSkill` (Task 1); cash REST `petSkills`/`petSkillAdd` (Task 4 JSON names); pets `SET_SKILL` contract (Task 6).
- Produces: `RequestItemConsumeBody.PetId uint64` (JSON `petId,omitempty`) — Task 9's channel mirror must match; `RequestItemConsume` gains a trailing `petId uint64` parameter; consumables pet client `SetSkill(actorId uint32, petId uint32, skill string, enabled bool) error`; error type string `PET_CANNOT_LEARN`.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go`:

```go
func TestPetSkillPouchClassification(t *testing.T) {
	// 0519 items route to the pet-skill branch, not the standard consumer.
	for _, id := range []item.Id{5190001, 5190006, 5191001} {
		if item.GetClassification(id) != item.ClassificationPetSkill {
			t.Errorf("GetClassification(%d) = %d, want 519", id, item.GetClassification(id))
		}
		if usesStandardConsumer(id) {
			t.Errorf("usesStandardConsumer(%d) = true, want false", id)
		}
	}
}
```

(`usesStandardConsumer` is package-internal; this test file is already `package consumable`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test -run TestPetSkillPouchClassification ./consumable/... -v`
Expected: FAIL — `ClassificationPetSkill` exists (Task 1) so this may PASS immediately if `usesStandardConsumer` already excludes cash items; if it passes, keep it as a pin and continue. The compile failures in later steps are the real red gate for this task.

- [ ] **Step 3: Message mirrors**

`kafka/message/pet/kafka.go` — align `PetId` with atlas-pets (`uint32`) and add the command (full file after change):

```go
package pet

const (
	EnvCommandTopic      = "COMMAND_TOPIC_PET"
	CommandAwardFullness = "AWARD_FULLNESS"
	CommandSetSkill      = "SET_SKILL"
)

type Command[E any] struct {
	ActorId uint32 `json:"actorId"`
	PetId   uint32 `json:"petId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type AwardFullnessCommandBody struct {
	Amount byte `json:"amount"`
}

// SetSkillCommandBody carries a semantic pet skill key (atlas-constants
// pet/skill spelling) — never a client wire bit.
type SetSkillCommandBody struct {
	Skill   string `json:"skill"`
	Enabled bool   `json:"enabled"`
}
```

`kafka/message/consumable/kafka.go` — add to `RequestItemConsumeBody`:

```go
	PetId    uint64        `json:"petId,omitempty"`
```

- [ ] **Step 4: Pet client SetSkill + AwardFullness type alignment**

`pet/producer.go`:

```go
func awardFullnessCommandProvider(actorId uint32, petId uint32, amount byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &message.Command[message.AwardFullnessCommandBody]{
		ActorId: actorId,
		PetId:   petId,
		Type:    message.CommandAwardFullness,
		Body: message.AwardFullnessCommandBody{
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func setSkillCommandProvider(actorId uint32, petId uint32, skill string, enabled bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &message.Command[message.SetSkillCommandBody]{
		ActorId: actorId,
		PetId:   petId,
		Type:    message.CommandSetSkill,
		Body: message.SetSkillCommandBody{
			Skill:   skill,
			Enabled: enabled,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

`pet/processor.go` — change `AwardFullness` and add `SetSkill`:

```go
func (p *Processor) AwardFullness(actorId uint32, petId uint32, amount byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvCommandTopic)(awardFullnessCommandProvider(actorId, petId, amount))
}

func (p *Processor) SetSkill(actorId uint32, petId uint32, skill string, enabled bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvCommandTopic)(setSkillCommandProvider(actorId, petId, skill, enabled))
}
```

Update the two existing `AwardFullness` call sites in `consumable/processor.go` (`ConsumePetFood`, `ConsumeCashPetFood`): `pp.AwardFullness(characterId, uint32(pe.Id()), inc)` (consumables' pet `Model.Id()` is `uint64`; pet ids are `uint32` — the narrowing cast is safe and matches atlas-pets).

- [ ] **Step 5: Cash client mirror**

`cash/rest.go` — add to `RestModel` after `Spec`:

```go
	PetSkills   []string `json:"petSkills,omitempty"`
	PetSkillAdd bool     `json:"petSkillAdd,omitempty"`
```

and thread both through `Extract` into the model. `cash/model.go` — add fields + accessors:

```go
	petSkills   []string
	petSkillAdd bool
```

```go
// PetSkills returns the semantic skill keys this 0519 item grants or removes.
func (m Model) PetSkills() []string { return m.petSkills }

// PetSkillAdd reports grant (true) vs removal (false); only meaningful when
// PetSkills is non-empty.
func (m Model) PetSkillAdd() bool { return m.petSkillAdd }
```

- [ ] **Step 6: Consume branch**

`consumable/processor.go`:

1. Add next to `ErrPetCannotConsume` (line ~47):

```go
var ErrPetCannotLearn = errors.New("pet cannot learn")
```

2. In `ConsumeError` (line ~280), extend the error-type mapping:

```go
	if errors.Is(err, ErrPetCannotLearn) {
		errorType = consumable.ErrorTypePetCannotLearn
	}
```

and add to `kafka/message/consumable/kafka.go` const block:

```go
	ErrorTypePetCannotLearn   = "PET_CANNOT_LEARN"
```

3. Change `RequestItemConsume`'s signature to accept the pet:

```go
func (p *Processor) RequestItemConsume(c channel.Model, characterId uint32, slot int16, itemId item2.Id, quantity int16, petId uint64) error {
```

and add the branch before the `else` fallback:

```go
	} else if item2.GetClassification(itemId) == item2.ClassificationPetSkill {
		itemConsumer = ConsumePetSkillPouch(transactionId, characterId, slot, itemId, petId)
	}
```

4. Add the consumer function (near `ConsumeCashPetFood`):

```go
// ConsumePetSkillPouch applies a 0519 pet skill pouch: validates the wire
// petId against ownership and spawn state, emits SET_SKILL for each skill key
// the item's data carries (add=grant, add=0=remove), then commits the cash
// reservation. Skill keys are data-driven from item WZ attributes — never
// hardcoded per item id.
func ConsumePetSkillPouch(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id, petId uint64) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			pp := pet.NewProcessor(l, ctx)
			cpp := compartment.NewProcessor(l, ctx)

			if petId == 0 || petId > math.MaxUint32 {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueCash, slot, ErrPetCannotLearn)
			}

			ci, err := cash.NewProcessor(l, ctx).GetById(uint32(itemId))
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueCash, slot, err)
			}
			skills := ci.PetSkills()
			if len(skills) == 0 {
				l.Warnf("Cash item [%d] is classification 519 but carries no pet skill keys; data missing or stale ingest.", itemId)
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueCash, slot, errors.New("pet skill data missing"))
			}

			pe, err := pp.GetById(petId)
			if err != nil || pe.OwnerId() != characterId || !pet.Spawned(pe) {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueCash, slot, ErrPetCannotLearn)
			}

			for _, sk := range skills {
				if err := pp.SetSkill(characterId, uint32(petId), sk, ci.PetSkillAdd()); err != nil {
					return p.ConsumeError(characterId, transactionId, inventory2.TypeValueCash, slot, err)
				}
			}

			if err := cpp.ConsumeItem(characterId, inventory2.TypeValueCash, transactionId, slot); err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueCash, slot, err)
			}
			return nil
		}
	}
}
```

(`TypeValueCash` throughout — deliberately not copying `ConsumeCashPetFood`'s Use/Cash mix-up at `processor.go:438-460`; that pre-existing inconsistency stays out of scope.)

5. `kafka/consumer/consumable/consumer.go` — pass the new field:

```go
	err := consumable.NewProcessor(l, ctx).RequestItemConsume(ch, uint32(c.CharacterId), int16(c.Body.Source), c.Body.ItemId, c.Body.Quantity, c.Body.PetId)
```

(Fix any other `RequestItemConsume` caller the compiler surfaces the same way, passing `0` where no pet is involved.)

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test -race ./... && go vet ./... && go build ./...`
Expected: PASS, clean.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/
git commit -m "feat(atlas-consumables): ConsumePetSkillPouch branch for 0519 items emits SET_SKILL"
```

---

### Task 8: atlas-packet — jms type-28 sub-packet `ItemUsePetSkill` (IDA-verified)

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/item_use_pet_skill.go`
- Test: `libs/atlas-packet/cash/serverbound/item_use_pet_skill_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `cashsb.NewItemUsePetSkill(updateTimeFirst bool) *ItemUsePetSkill` with `PetId() uint64` and `UpdateTime() uint32`, decoding the bytes that follow the common `cashsb.ItemUse` prefix for cash-slot-item type 28. Task 9 consumes it.

- [ ] **Step 1: IDA verification (blocking prerequisite — do NOT guess the layout)**

1. `mcp__ida-pro__select_instance` for the **jms** instance (confirm the loaded IDB is the jms v185 `_U_DEVM` build before reading).
2. `mcp__ida-pro__func_query` with `name_regex` for `SendConsumeCashItemUseRequest` (known at `0xaef2f5`), then `mcp__ida-pro__decompile` it.
3. Locate jump-table **case 28** (the design pinned the pet-SN encode `EncodeBuffer(pet+0x18, 8)` at `0xaf1a42`). Record, in order, every encode between the common prefix (`source i16`, `itemId u32`, plus wherever jms puts `updateTime`) and the end of the case-28 arm.
4. Write the confirmed byte order into the test fixture below, adjusting if the client encodes anything besides the 8-byte pet SN (e.g. leading/trailing updateTime). If the function cannot be located or the case-28 arm does not match the design's description, STOP and escalate (unresolved-fname rule) — do not substitute a layout.

- [ ] **Step 2: Write the failing byte-fixture test**

Create `libs/atlas-packet/cash/serverbound/item_use_pet_skill_test.go`. Template assuming the verified layout is `petId u64` with the same trailing-updateTime convention as `ItemUsePetConsumable` — **adjust to the Step 1 finding and note the verified order in the test comment**:

```go
package serverbound

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// packet-audit:verify CWvsContext::SendConsumeCashItemUseRequest (case 28, jms_v185)
// Verified via IDA jms_v185: <record the actual encode order and addresses here>.
func TestItemUsePetSkillDecode(t *testing.T) {
	raw := []byte{
		0x2A, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // petId = 42 (pet locker SN, u64 LE)
		// <updateTime bytes here if and only if Step 1 shows the client sends one in this position>
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := NewItemUsePetSkill(true)
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if p.PetId() != 42 {
		t.Errorf("petId = %d, want 42", p.PetId())
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test -run TestItemUsePetSkill ./cash/serverbound/... -v`
Expected: FAIL — type undefined.

- [ ] **Step 4: Write the implementation**

Create `libs/atlas-packet/cash/serverbound/item_use_pet_skill.go`, mirroring `item_use_pet_consumable.go`'s shape and encoding exactly the Step 1 order:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUsePetSkill is the type-28 sub-body of CUser::SendCashItemUseRequest /
// CWvsContext::SendConsumeCashItemUseRequest: a 0519 pet skill pouch use. Only
// the jms client can emit it (GMS builds have no case-28 arm); the petId is
// the chosen pet's 8-byte locker SN, which round-trips as the Atlas pet id.
type ItemUsePetSkill struct {
	petId           uint64
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUsePetSkill(updateTimeFirst bool) *ItemUsePetSkill {
	return &ItemUsePetSkill{updateTimeFirst: updateTimeFirst}
}

func (m ItemUsePetSkill) PetId() uint64      { return m.petId }
func (m ItemUsePetSkill) UpdateTime() uint32 { return m.updateTime }

func (m ItemUsePetSkill) Operation() string { return "ItemUsePetSkill" }

func (m ItemUsePetSkill) String() string {
	return fmt.Sprintf("petId [%d] updateTime [%d]", m.petId, m.updateTime)
}

func (m ItemUsePetSkill) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteLong(m.petId)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUsePetSkill) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.petId = r.ReadUint64()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
```

(If Step 1 shows a different order — e.g. no trailing updateTime on jms — implement the verified order and drop the divergent branch. The struct comment must cite the IDA evidence either way.)

- [ ] **Step 5: Run tests, commit**

Run: `cd libs/atlas-packet && go test -race ./cash/serverbound/... && go vet ./...`
Expected: PASS.

```bash
git add libs/atlas-packet/cash/serverbound/item_use_pet_skill.go libs/atlas-packet/cash/serverbound/item_use_pet_skill_test.go
git commit -m "feat(atlas-packet): jms type-28 ItemUsePetSkill sub-packet (IDA-verified)"
```

---

### Task 9: atlas-channel — cash-item-use type-28 arm + petId command plumbing

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go` (`PetId` field)
- Modify: `services/atlas-channel/atlas.com/channel/consumable/processor.go` + `consumable/producer.go` (`RequestItemConsumeWithPet`)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` (arm + constant + classification swap)

**Interfaces:**
- Consumes: `cashsb.NewItemUsePetSkill` (Task 8), `item.ClassificationPetSkill` (Task 1). JSON field `petId,omitempty` must match Task 7's consumables mirror.
- Produces: `consumable.Processor.RequestItemConsumeWithPet(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, updateTime uint32, petId uint64) error`; `CashSlotItemTypePetSkill = CashSlotItemType(28)`.

- [ ] **Step 1: Message + producer + processor**

`kafka/message/consumable/kafka.go` — add to `RequestItemConsumeBody`:

```go
	PetId    uint64        `json:"petId,omitempty"`
```

`consumable/producer.go` — extend the provider (keep the existing exported name and add the pet variant so existing callers stay untouched):

```go
func RequestItemConsumeWithPetCommandProvider(f field.Model, characterId character.Id, source slot.Position, itemId item.Id, quantity int16, petId uint64) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.RequestItemConsumeBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        consumable.CommandRequestItemConsume,
		Body: consumable.RequestItemConsumeBody{
			Source:   source,
			ItemId:   itemId,
			Quantity: quantity,
			PetId:    petId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

`consumable/processor.go` — add:

```go
// RequestItemConsumeWithPet is RequestItemConsume for consume paths that carry
// a target pet (0519 pet skill pouches). The auto-pot path deliberately does
// NOT use it: its pet validation happens at the socket handler and nothing
// downstream needs the pet.
func (p *Processor) RequestItemConsumeWithPet(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, updateTime uint32, petId uint64) error {
	p.l.Debugf("Character [%d] using pet skill item [%d] from slot [%d] on pet [%d]. updateTime [%d]", characterId, itemId, source, petId, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestItemConsumeWithPetCommandProvider(f, characterId, source, itemId, 1, petId))
}
```

- [ ] **Step 2: Handler arm**

`socket/handler/character_cash_item_use.go`:

1. Add the named constant next to the others:

```go
	CashSlotItemTypePetSkill      = CashSlotItemType(28)
```

2. In `GetCashSlotItemType`, replace the magic literal:

```go
	if category == 519 {
		return CashSlotItemType(28)
	}
```

with:

```go
	if category == item.ClassificationPetSkill {
		return CashSlotItemTypePetSkill
	}
```

3. Add the arm in `CharacterCashItemUseHandleFunc` after the `CashSlotItemTypePetConsumable` block:

```go
		if it == CashSlotItemTypePetSkill {
			sp := cashsb.NewItemUsePetSkill(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			if !updateTimeFirst {
				updateTime = sp.UpdateTime()
			}
			_ = consumable.NewProcessor(l, ctx).RequestItemConsumeWithPet(s.Field(), character.Id(s.CharacterId()), itemId, source, updateTime, sp.PetId())
			return
		}
```

(If Task 8's verified layout dropped the trailing updateTime, drop the `if !updateTimeFirst` block here too. A 0519 use arriving from a client family that cannot send one decodes garbage — downstream owner/spawned validation in consumables rejects it; no version-specific rejection code, per design §3.3.)

- [ ] **Step 3: Build, vet, test, commit**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./consumable/... ./socket/handler/...`
Expected: clean.

```bash
git add services/atlas-channel/atlas.com/channel/
git commit -m "feat(atlas-channel): route 0519 pet skill pouch use with target petId"
```

---

### Task 10: atlas-channel — `data/consumable` and `data/equipment` client packages

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/data/consumable/{rest.go,requests.go,processor.go}`
- Create: `services/atlas-channel/atlas.com/channel/data/equipment/{rest.go,requests.go,processor.go}`

**Interfaces:**
- Consumes: atlas-data REST — `GET {DATA}/data/consumables/{id}` (JSON:API type `consumables`) and `GET {DATA}/data/equipment/{id}` (JSON:API type `statistics`, now carrying `petAbilities` from Task 5).
- Produces: `data/consumable`: `NewProcessor(l, ctx).GetById(itemId uint32) (Model, error)` with `Model.GetSpec(SpecType) (int32, bool)` and keys `SpecTypeHP("hp")`, `SpecTypeHPRecovery("hpR")`, `SpecTypeMP("mp")`, `SpecTypeMPRecovery("mpR")`. `data/equipment`: `NewProcessor(l, ctx).GetById(itemId uint32) (Model, error)` with `Model.PetAbilities() []string`. Task 12 consumes both.

- [ ] **Step 1: data/consumable package**

`data/consumable/rest.go` (spec keys copied from `atlas-consumables/atlas.com/consumables/data/consumable`, JSON:API name from atlas-data's consumable RestModel — `"consumables"`):

```go
package consumable

import "strconv"

type SpecType string

const (
	SpecTypeHP         = SpecType("hp")
	SpecTypeMP         = SpecType("mp")
	SpecTypeHPRecovery = SpecType("hpR")
	SpecTypeMPRecovery = SpecType("mpR")
)

type RestModel struct {
	Id   uint32             `json:"-"`
	Spec map[SpecType]int32 `json:"spec"`
}

func (r RestModel) GetName() string {
	return "consumables"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:   rm.Id,
		spec: rm.Spec,
	}, nil
}

type Model struct {
	id   uint32
	spec map[SpecType]int32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) GetSpec(specType SpecType) (int32, bool) {
	val, ok := m.spec[specType]
	return val, ok
}
```

`data/consumable/requests.go` (URL from `atlas-consumables`' client — `data/consumables/%d`):

```go
package consumable

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	consumableResource = "data/consumables/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(itemId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+consumableResource, itemId))
}
```

`data/consumable/processor.go` (mirror `data/cash/processor.go`'s shape):

```go
package consumable

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

func (p *Processor) GetById(itemId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(itemId), Extract)()
}
```

(Before finalizing, compare with `data/cash/processor.go` + `data/cash/rest.go` — if that package's `GetById` returns the RestModel directly or its `Extract` lives elsewhere, mirror the local house style rather than this sketch, keeping the produced signatures above.)

- [ ] **Step 2: data/equipment package**

Same shape; `rest.go`:

```go
package equipment

import "strconv"

type RestModel struct {
	Id           uint32   `json:"-"`
	PetAbilities []string `json:"petAbilities,omitempty"`
}

func (r RestModel) GetName() string {
	return "statistics"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:           rm.Id,
		petAbilities: rm.PetAbilities,
	}, nil
}

type Model struct {
	id           uint32
	petAbilities []string
}

func (m Model) Id() uint32 {
	return m.id
}

// PetAbilities lists the equip's truthy pet-ability attributes (equip-family
// spelling: consumeHP, consumeMP, sweepForDrop, ...).
func (m Model) PetAbilities() []string {
	return m.petAbilities
}
```

`requests.go` with `equipmentResource = "data/equipment/%d"`, `processor.go` identical in shape to Step 1.

- [ ] **Step 3: Build, vet, commit**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./data/...`
Expected: clean.

```bash
git add services/atlas-channel/atlas.com/channel/data/
git commit -m "feat(atlas-channel): data clients for consumable specs and equipment pet abilities"
```

---

### Task 11: atlas-channel — pet flag sync (FLAG_CHANGED consumer, asset enrichment, encode wiring)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/pet/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/pet/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/asset/builder.go`, `asset/model.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/processor.go` (enrichment)
- Modify: `services/atlas-channel/atlas.com/channel/socket/model/asset.go`

**Interfaces:**
- Consumes: `FLAG_CHANGED` event (Task 6 body `{slot,flag}`), `packetmodel.Asset.SetPetFlag` (Task 3), channel pet `Model.Flag()` (already exists).
- Produces: pet cash assets encoded to clients now carry the pet's flag; a flag change re-announces the pet asset (same `announcePetStatUpdate` used by closeness changes).

- [ ] **Step 1: Message mirror + consumer**

`kafka/message/pet/kafka.go` — add to the status-event const block:

```go
	StatusEventTypeFlagChanged      = "FLAG_CHANGED"
```

and the body type:

```go
type FlagChangedStatusEventBody struct {
	Slot int8   `json:"slot"`
	Flag uint16 `json:"flag"`
}
```

`kafka/consumer/pet/consumer.go` — register after the `handleExcludeChanged` registration (same pattern):

```go
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleFlagChanged(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
```

and add the handler (modeled on `handleClosenessChanged` at line 257):

```go
func handleFlagChanged(sc server.Model, wp writer.Producer) message.Handler[pet2.StatusEvent[pet2.FlagChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e pet2.StatusEvent[pet2.FlagChangedStatusEventBody]) {
		if e.Type != pet2.StatusEventTypeFlagChanged {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		// Re-announce the pet's cash asset so the client's GW_ItemSlotPet
		// usPetSkill short refreshes — there is no dedicated skill packet.
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.OwnerId, announcePetStatUpdate(l)(ctx)(wp)(e.PetId, e.OwnerId))
	}
}
```

- [ ] **Step 2: Asset flag plumbing**

`asset/builder.go` — add `petFlag uint16` to the builder struct (next to `petLevel`), thread it in `Clone` (the `petLevel: m.petLevel,` block) and `Build` (the `petLevel: b.petLevel,` block), and add the setter next to `SetPetLevel`:

```go
func (b *ModelBuilder) SetPetFlag(v uint16) *ModelBuilder             { b.petFlag = v; return b }
```

`asset/model.go` — add `petFlag uint16` next to `petLevel` and the getter next to `PetLevel()`:

```go
func (m Model) PetFlag() uint16          { return m.petFlag }
```

`character/processor.go` — in `PetAssetEnrichmentDecorator`'s enrichment chain (line ~127), add:

```go
					SetPetFlag(pm.Flag()).
```

between `SetPetLevel(...)` and `SetCloseness(...)`.

`socket/model/asset.go` — in the `if a.IsPet()` branch:

```go
	if a.IsPet() {
		base = base.SetPetInfo(a.PetId(), a.PetName(), a.PetLevel(), a.Fullness(), a.Closeness())
		base = base.SetPetFlag(a.PetFlag())
	}
```

- [ ] **Step 3: Build, vet, test, commit**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./asset/... ./socket/...`
Expected: clean.

```bash
git add services/atlas-channel/atlas.com/channel/
git commit -m "feat(atlas-channel): sync pet skill flag to clients via FLAG_CHANGED re-announce"
```

---

### Task 12: atlas-channel — `PetItemUseHandle` validation chain

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/enable_actions.go` (moved helper)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go` (remove the moved helper)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/pet_item_use.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/pet_item_use_test.go`

**Interfaces:**
- Consumes: channel `pet.NewProcessor(l,ctx).GetById(uint32)` (`Model.OwnerId()/Slot()/Flag()`), `character.NewProcessor(l,ctx).GetById()(id)` (`Model.Hp()`), `compartment.NewProcessor(l,ctx).GetByType(characterId, inventory.TypeValueEquip)`, `data/consumable` + `data/equipment` (Task 10), `pet/skill.Has` (Task 1), `slot` constants.
- Produces: the validated handler. Rejection = shared `enableActions` unstick + one structured warn. Reasons: `pet_not_found`, `pet_not_owned`, `pet_not_spawned`, `character_dead`, `missing_pet_skill`, `not_consumable`, `skill_gate_unconfigured`, `equip_data_missing`, `fetch_failed` (character/transport fetch error — fail closed).

**gms_48 pet resolution (rebase revision).** The v48 packet has no `petId`
(design §1.7), so `evaluateAutoPot` keeps its signature but the *lookup* in front
of it branches on whether the decoded packet carried one:

```go
// petId on the wire (GMS >= 61, JMS) -> resolve that pet, narrowing-guard it.
// No petId (gms_48, single-pet client) -> resolve the character's spawned pet.
// Never fall back from the first to the second: that reopens the FR-1 hole.
```

Add `pet.NewProcessor(l, ctx).GetByOwner(characterId)`-style spawned lookup (the
channel pet processor already exposes an owner-scoped fetch; filter `Slot() >= 0`
— check the exact method name before writing) and two table cases:
`{"v48 no petId, spawned pet", …, wantOk: true}` and
`{"v48 no petId, no spawned pet", …, wantReason: "pet_not_found"}`. The decoded
`ItemUse` needs a way to say "absent" — have the Task 15 codec gate leave
`petId` zero when the version does not read it, and treat `petId == 0` as the
absent case (a real Atlas pet id is never 0).

> **Revision (post-merge review).** Treating `petId == 0` as "absent" is not
> sufficient on its own: it conflates "this version has no petId field"
> (gms_48) with "this version has the field and the client sent literal 0"
> (malformed or forged on v61+). The second case was falling through into the
> spawned-pet branch, contradicting this section's own "never fall back from
> one to the other" invariant. It was not exploitable — the fallback only ever
> resolves the caller's *own* spawned pet and `evaluateAutoPot`'s ownership
> check still gates the result — but the code no longer matches the comment.
> Resolution: `classifyPetIdInput(hasWirePetId, petId)`, a pure helper gated on
> the now-exported `pet2.HasLeadingPetId(t)`, returns
> (usePetId, reason, ok) and rejects the third case with the existing
> `pet_not_found` reason. Table-tested in `pet_item_use_test.go`
> (`TestClassifyPetIdInput`); no new rejection reason was introduced, so the
> documented reason vocabulary above is unchanged.

**Version note (rebase revision).** Otherwise the handler stays version-uniform:
every GMS client uses the same worn-equip gate and the same pet-ability slot list
(design §1.1, verified in `CPet__UpdatePetAbility` v61 `0x614b60`), so
`petAbilityPositions` needs no per-version branch and no new gate mode is added.
The only version-dependent input remains the config-resolved `skillGate` option.
`buffSkill` is decoded as a **bool** by the current codec (`ItemUse.BuffSkill()`),
while the wire byte carries the mitigation code `{0,1,2,4,8}` (design §1.2) — log
it as the bool it is, or widen the codec to a byte in its own change with the
five verified cells re-fixtured. Do **not** widen it silently inside this task.

- [ ] **Step 1: Move `enableActions` to a shared file**

Create `socket/handler/enable_actions.go`:

```go
package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	"github.com/sirupsen/logrus"
)

// enableActions releases the client's pending item/skill-use state with an
// empty StatChanged (exclRequestSent=true) — the canonical unstick response.
func enableActions(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
		return func(wp writer.Producer) func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)
		}
	}
}
```

Delete the identical function from `character_skill_use.go:131-137` (same package — all existing callers keep working). Run `cd services/atlas-channel/atlas.com/channel && go build ./...` to confirm.

- [ ] **Step 2: Write the failing tests (pure decision logic)**

Create `socket/handler/pet_item_use_test.go`:

```go
package handler

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
)

func TestEvaluateAutoPot(t *testing.T) {
	cases := []struct {
		name                     string
		characterHp              uint16
		petOwnerId               uint32
		petSlot                  int8
		recoversHP, recoversMP   bool
		hasHPSource, hasMPSource bool
		wantReason               string
		wantOk                   bool
	}{
		{"happy hp", 100, 1, 0, true, false, true, false, "", true},
		{"happy mp", 100, 1, 0, false, true, false, true, "", true},
		{"dual either hp source", 100, 1, 0, true, true, true, false, "", true},
		{"dual either mp source", 100, 1, 0, true, true, false, true, "", true},
		{"not owned", 100, 2, 0, true, false, true, false, "pet_not_owned", false},
		{"not spawned", 100, 1, -1, true, false, true, false, "pet_not_spawned", false},
		{"dead", 0, 1, 0, true, false, true, false, "character_dead", false},
		{"missing hp skill", 100, 1, 0, true, false, false, true, "missing_pet_skill", false},
		{"missing mp skill", 100, 1, 0, false, true, true, false, "missing_pet_skill", false},
		{"not a potion", 100, 1, 0, false, false, true, true, "not_consumable", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reason, ok := evaluateAutoPot(1, c.characterHp, c.petOwnerId, c.petSlot, c.recoversHP, c.recoversMP, c.hasHPSource, c.hasMPSource)
			if ok != c.wantOk || reason != c.wantReason {
				t.Errorf("got (%q,%v), want (%q,%v)", reason, ok, c.wantReason, c.wantOk)
			}
		})
	}
}

func TestPetAbilityPositions(t *testing.T) {
	cases := []struct {
		petSlot int8
		in      slot.Position
		want    bool
	}{
		{0, -24, true},  // petHP, always honored
		{0, -25, true},  // petMP, always honored
		{0, -21, true},  // pet-0 ability range
		{0, -46, true},  // petItemIgnore (pet 0)
		{0, -31, false}, // pet-1 range does not apply to pet 0
		{1, -31, true},
		{1, -24, true}, // shared slots apply to every pet index
		{1, -21, false},
		{2, -39, true},
		{2, -48, true},
		{2, -46, false},
		{0, -7, false}, // ordinary equip slot never matches
	}
	for _, c := range cases {
		got := petAbilityPositions(c.petSlot)[c.in]
		if got != c.want {
			t.Errorf("petAbilityPositions(%d)[%d] = %v, want %v", c.petSlot, c.in, got, c.want)
		}
	}
}

func TestNormalizeWornPosition(t *testing.T) {
	// Worn cash equips are stored at position-100 in the raw equip compartment
	// (see character.Model.SetInventory); pet equips are cash items.
	if got := normalizeWornPosition(-124); got != -24 {
		t.Errorf("normalizeWornPosition(-124) = %d, want -24", got)
	}
	if got := normalizeWornPosition(-24); got != -24 {
		t.Errorf("normalizeWornPosition(-24) = %d, want -24", got)
	}
	if got := normalizeWornPosition(5); got != 5 {
		t.Errorf("normalizeWornPosition(5) = %d, want 5", got)
	}
}

func TestMatchPetAbilityEquips(t *testing.T) {
	worn := []wornEquip{
		{position: -124, abilities: []string{"consumeHP"}},
		{position: -31, abilities: []string{"consumeMP"}},
	}
	// pet 0: -124 normalizes to -24 (shared) -> HP yes; -31 is pet-1 range -> MP no.
	hasHP, hasMP, sawData := matchPetAbilityEquips(worn, 0)
	if !hasHP || hasMP || !sawData {
		t.Errorf("pet 0: got (%v,%v,%v), want (true,false,true)", hasHP, hasMP, sawData)
	}
	// pet 1: shared -24 HP yes; -31 in range -> MP yes.
	hasHP, hasMP, _ = matchPetAbilityEquips(worn, 1)
	if !hasHP || !hasMP {
		t.Errorf("pet 1: got (%v,%v), want (true,true)", hasHP, hasMP)
	}
	// no attribute data at all -> sawData false (drives equip_data_missing).
	_, _, sawData = matchPetAbilityEquips([]wornEquip{{position: -124, abilities: nil}}, 0)
	if sawData {
		t.Error("sawData = true with no ability data, want false")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test -run 'TestEvaluateAutoPot|TestPetAbilityPositions|TestNormalizeWornPosition|TestMatchPetAbilityEquips' ./socket/handler/... -v`
Expected: FAIL — functions undefined.

- [ ] **Step 4: Implement the handler**

Rewrite `socket/handler/pet_item_use.go`:

```go
package handler

import (
	"atlas-channel/character"
	"atlas-channel/compartment"
	"atlas-channel/consumable"
	consumabledata "atlas-channel/data/consumable"
	equipmentdata "atlas-channel/data/equipment"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"math"

	character2 "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	petskill "github.com/Chronicle20/atlas/libs/atlas-constants/pet/skill"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	pet2 "github.com/Chronicle20/atlas/libs/atlas-packet/pet/serverbound"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// skillGate option values (tenant handler config, design §3.2): the FR-3 gate
// mirrors exactly the gate the tenant's client family enforces.
const (
	skillGateEquipAbility = "equipAbility" // GMS: worn pet-ability equip
	skillGatePetSkillFlag = "petSkillFlag" // JMS: pouch-taught pet flag
)

func PetItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := pet2.ItemUse{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		reject := func(reason string) {
			l.WithFields(logrus.Fields{
				"characterId": s.CharacterId(),
				"petId":       p.PetId(),
				"itemId":      p.ItemId(),
				"slot":        p.Source(),
				"buffSkill":   p.BuffSkill(),
				"reason":      reason,
			}).Warnf("Rejecting pet auto-pot request from character [%d]: [%s].", s.CharacterId(), reason)
			if err := enableActions(l)(ctx)(wp)(s); err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", statpkt.StatChangedWriter, s.CharacterId())
			}
		}

		// Atlas pet ids are uint32; the wire petId (pet locker SN) round-trips
		// as the pet id, so anything wider is forged.
		if p.PetId() > math.MaxUint32 {
			reject("pet_not_found")
			return
		}

		gate, _ := readerOptions["skillGate"].(string)
		if gate != skillGateEquipAbility && gate != skillGatePetSkillFlag {
			// Fail closed and loud: a template gap must never be permissive.
			reject("skill_gate_unconfigured")
			return
		}

		// Parallel fetch — one round-trip of latency, not three (design §3.1).
		// model.Future carries no per-future error (Group.Wait returns only the
		// first), so each provider captures its own error and never fails the
		// group; Wait() is the happens-before barrier for the captured values.
		pg, _ := model.NewGroup(ctx)
		var (
			pm    pet.Model
			pmErr error
			c     character.Model
			cErr  error
			ci    consumabledata.Model
			ciErr error
		)
		model.Submit(pg, func() (any, error) {
			pm, pmErr = pet.NewProcessor(l, ctx).GetById(uint32(p.PetId()))
			return nil, nil
		})
		model.Submit(pg, func() (any, error) {
			c, cErr = character.NewProcessor(l, ctx).GetById()(s.CharacterId())
			return nil, nil
		})
		model.Submit(pg, func() (any, error) {
			ci, ciErr = consumabledata.NewProcessor(l, ctx).GetById(p.ItemId())
			return nil, nil
		})
		_ = pg.Wait()

		// Fail closed on any fetch failure (design §5) — never forward unvalidated.
		if pmErr != nil {
			reject("pet_not_found")
			return
		}
		if ciErr != nil {
			reject("not_consumable")
			return
		}
		if cErr != nil {
			l.WithError(cErr).Warnf("Unable to resolve character [%d] during pet auto-pot validation.", s.CharacterId())
			reject("fetch_failed")
			return
		}

		recoversHP, recoversMP := classifyRecovery(ci)

		hasHP, hasMP, ok := resolveSkillSources(l, ctx)(gate, s.CharacterId(), pm)
		if !ok {
			reject("equip_data_missing")
			return
		}

		if reason, pass := evaluateAutoPot(s.CharacterId(), c.Hp(), pm.OwnerId(), pm.Slot(), recoversHP, recoversMP, hasHP, hasMP); !pass {
			reject(reason)
			return
		}

		_ = consumable.NewProcessor(l, ctx).RequestItemConsume(s.Field(), character2.Id(s.CharacterId()), item.Id(p.ItemId()), slot.Position(p.Source()), p.UpdateTime())
	}
}

// classifyRecovery reports whether the consumed item's spec recovers HP and/or
// MP. HP vs MP intent is not on the wire (TryConsumePetHP/MP encode identical
// packets) — the server derives it from the item, and for dual items either
// matching skill source passes.
func classifyRecovery(ci consumabledata.Model) (bool, bool) {
	hp := false
	if v, ok := ci.GetSpec(consumabledata.SpecTypeHP); ok && v > 0 {
		hp = true
	}
	if v, ok := ci.GetSpec(consumabledata.SpecTypeHPRecovery); ok && v > 0 {
		hp = true
	}
	mp := false
	if v, ok := ci.GetSpec(consumabledata.SpecTypeMP); ok && v > 0 {
		mp = true
	}
	if v, ok := ci.GetSpec(consumabledata.SpecTypeMPRecovery); ok && v > 0 {
		mp = true
	}
	return hp, mp
}

// evaluateAutoPot runs the FR-1/FR-2/FR-3 decision on already-resolved inputs.
// Ordered cheapest-first; all failures are externally identical (unstick+warn).
func evaluateAutoPot(characterId uint32, characterHp uint16, petOwnerId uint32, petSlot int8, recoversHP, recoversMP, hasHPSource, hasMPSource bool) (string, bool) {
	if petOwnerId != characterId {
		return "pet_not_owned", false
	}
	if petSlot < 0 {
		return "pet_not_spawned", false
	}
	if characterHp == 0 {
		return "character_dead", false
	}
	if !recoversHP && !recoversMP {
		return "not_consumable", false
	}
	if recoversHP && hasHPSource {
		return "", true
	}
	if recoversMP && hasMPSource {
		return "", true
	}
	return "missing_pet_skill", false
}

// resolveSkillSources returns (hasHPSource, hasMPSource, ok). ok=false means
// the equip gate found worn candidates but no ability data at all — the
// deploy-ordering signal (atlas-data not re-ingested), distinct from a plain
// missing skill so operators can tell the two apart in logs.
func resolveSkillSources(l logrus.FieldLogger, ctx context.Context) func(gate string, characterId uint32, pm pet.Model) (bool, bool, bool) {
	return func(gate string, characterId uint32, pm pet.Model) (bool, bool, bool) {
		if gate == skillGatePetSkillFlag {
			return petskill.Has(pm.Flag(), petskill.ConsumeHP), petskill.Has(pm.Flag(), petskill.ConsumeMP), true
		}

		cm, err := compartment.NewProcessor(l, ctx).GetByType(characterId, inventory2.TypeValueEquip)
		if err != nil {
			return false, false, true // no worn equips resolvable -> plain missing_pet_skill
		}
		positions := petAbilityPositions(pm.Slot())
		var worn []wornEquip
		ep := equipmentdata.NewProcessor(l, ctx)
		for _, a := range cm.Assets() {
			pos := normalizeWornPosition(slot.Position(a.Slot()))
			if !positions[pos] {
				continue
			}
			em, err := ep.GetById(a.TemplateId())
			if err != nil {
				worn = append(worn, wornEquip{position: slot.Position(a.Slot())})
				continue
			}
			worn = append(worn, wornEquip{position: slot.Position(a.Slot()), abilities: em.PetAbilities()})
		}
		if len(worn) == 0 {
			return false, false, true
		}
		hasHP, hasMP, sawData := matchPetAbilityEquips(worn, pm.Slot())
		return hasHP, hasMP, sawData
	}
}

type wornEquip struct {
	position  slot.Position
	abilities []string
}

// petAbilityPositions mirrors the client's CPet::UpdatePetAbility slot list:
// petHP(-24)/petMP(-25) apply to every pet; each pet index additionally has
// its own ability range (pet 0: -21..-29,-46; pet 1: -31..-37,-47;
// pet 2: -39..-45,-48). Positions are the canonical (non-cash-offset) values
// from libs/atlas-constants/inventory/slot.
func petAbilityPositions(petSlot int8) map[slot.Position]bool {
	res := map[slot.Position]bool{-24: true, -25: true}
	var lo, hi, ignore slot.Position
	switch petSlot {
	case 0:
		lo, hi, ignore = -29, -21, -46
	case 1:
		lo, hi, ignore = -37, -31, -47
	case 2:
		lo, hi, ignore = -45, -39, -48
	default:
		return res
	}
	for p := lo; p <= hi; p++ {
		res[p] = true
	}
	res[ignore] = true
	return res
}

// normalizeWornPosition maps the raw equip-compartment position to the
// canonical slot position: worn cash equips are stored at position-100
// (see character.Model.SetInventory), and pet equips are cash items.
func normalizeWornPosition(p slot.Position) slot.Position {
	if p < -100 {
		return p + 100
	}
	return p
}

// matchPetAbilityEquips reports (hasHP, hasMP, sawData) across the worn
// candidates already filtered to the pet's ability positions. sawData=false
// when no candidate carried any ability attributes (missing equip data).
func matchPetAbilityEquips(worn []wornEquip, petSlot int8) (bool, bool, bool) {
	positions := petAbilityPositions(petSlot)
	hasHP, hasMP, sawData := false, false, false
	for _, w := range worn {
		if !positions[normalizeWornPosition(w.position)] {
			continue
		}
		if len(w.abilities) > 0 {
			sawData = true
		}
		for _, ab := range w.abilities {
			if ab == string(petskill.ConsumeHP) {
				hasHP = true
			}
			if ab == string(petskill.ConsumeMP) {
				hasMP = true
			}
		}
	}
	return hasHP, hasMP, sawData
}
```

Implementation notes for the subagent:
- `p.BuffSkill()` is decoded and logged, never control flow: the wire byte is damage-mitigation context from `CUserLocal::SetDamaged` (0=normal, 1=Power Guard, 2=Meso Guard, 4/8=other mitigation arms), not a skill id — keep that comment on the packet struct usage (FR-6).
- `model.Future[T].Get()` is the only accessor (`libs/atlas-model/model/parallel_group.go:26`) and `Group.Wait()` returns only the first error — hence the error-capture closures above; do not "simplify" back to error-returning providers or reason attribution is lost.
- The forward call on success is **byte-for-byte the pre-task call** (FR-5): `RequestItemConsume(s.Field(), character.Id(s.CharacterId()), item.Id(p.ItemId()), slot.Position(p.Source()), p.UpdateTime())`. The downstream consumable ERROR-event unstick fires only for forwarded requests, so rejection cannot double-unstick (FR-13).
- Check `compartment.NewProcessor(l, ctx).GetByType` exists with that name (it is what `GetItemInSlot` uses via `p.cp`); if the compartment processor constructor differs, adapt the call, not the logic.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/... && go vet ./... && go build ./...`
Expected: PASS, clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/
git commit -m "feat(atlas-channel): validate pet auto-pot (ownership, spawn, alive, version-family skill gate)"
```

---

### Task 13: atlas-configurations — seed template wiring (+ per-version IDA bit verification)

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_48_1.json` (**new entry**, not just options)
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_61_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_72_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_79_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`
- Not modified (documented exclusion, design §3.6): `template_gms_12_1.json` / `template_gms_92_1.json` (partial bring-up templates, no pet handlers at all).
  - **Correction (post-merge):** `template_gms_92_1.json` IS now modified. Merging main added a `PetItemUseHandle` entry to it (opCode `0xC8`), invalidating the exclusion premise and leaving the fail-closed gate rejecting every v92 auto-pot. It now carries `options.skillGate: "equipAbility"` (IDA-verified on the v92 IDB) and the `petSkill` writer table on `CharacterInventoryChange` `0x1E` / `SetField` `0x8C`. See the correction note in design §3.6. `template_gms_12_1.json` remains genuinely unmodified.

**Interfaces:**
- Consumes: handler option key `skillGate` (Task 12), writer options property `petSkill` (Task 3).
- Produces: live handler/writer wiring for new tenants. (Existing tenants need the live-config PATCH — rollout note in `context.md`, known pattern `bug_new_opcodes_not_in_live_tenant_config`.)

- [ ] **Step 1: skillGate handler options**

Update the `PetItemUseHandle` entry in each template (opCodes below are the ones on main; the eight existing entries already carry `"validator": "LoggedInValidator"` — only `options` is added):

- `template_gms_48_1.json`: **create** the entry — `{"opCode": "0x75", "validator": "LoggedInValidator", "handler": "PetItemUseHandle", "services": ["channel"], "options": {"skillGate": "equipAbility"}}`. Insert at its sorted position, immediately after `PetDropPickUpHandle` 0x74 (which is also where it belongs semantically — the guard enforces ascending opCode, so sorted position and semantic position agree here). Mirror the `services` key from the sibling pet entries in that file. Verified: 0x75 is unused in this template, and v48's `CPet::SendDropPickUpRequest` (`0x58ed98`) emits 116 = 0x74, confirming this IDB's raw opcode values are the template opCodes.
- `template_gms_61_1.json` (`:877-882`): `{"opCode": "0x8E", "validator": "LoggedInValidator", "handler": "PetItemUseHandle", "options": {"skillGate": "equipAbility"}}`
- `template_gms_72_1.json` (`:925-930`): `{"opCode": "0xA5", ... "options": {"skillGate": "equipAbility"}}`
- `template_gms_79_1.json` (`:936-941`): `{"opCode": "0xA7", ... "options": {"skillGate": "equipAbility"}}`
- `template_gms_83_1.json`: `{"opCode": "0xAB", "validator": "LoggedInValidator", "handler": "PetItemUseHandle", "options": {"skillGate": "equipAbility"}}`
- `template_gms_84_1.json`: `{"opCode": "0xB0", "validator": "LoggedInValidator", "handler": "PetItemUseHandle", "options": {"skillGate": "equipAbility"}}`
- `template_gms_87_1.json`: `{"opCode": "0xB7", "validator": "LoggedInValidator", "handler": "PetItemUseHandle", "options": {"skillGate": "equipAbility"}}`
- `template_gms_95_1.json`: `{"opCode": "0xCB", "validator": "LoggedInValidator", "handler": "PetItemUseHandle", "options": {"skillGate": "equipAbility"}}` (validator added — see Step 2)
- `template_jms_185_1.json`: `{"opCode": "0xAE", "validator": "LoggedInValidator", "handler": "PetItemUseHandle", "options": {"skillGate": "petSkillFlag"}}`

- [ ] **Step 2: ~~gms_95 dead pet-handler block~~ — DROPPED (fixed on main)**

The eight gms_95 pet handlers (0x52, 0x6E, 0xC7–0xCC) now all carry
`"validator": "LoggedInValidator"`; `PetItemUseHandle` is at
`template_gms_95_1.json:636-640`. Re-confirm with
`grep -B2 '"handler": "Pet' services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
before assuming — if a validator is missing again, restore this step. The
underlying silent-skip in `BuildHandlerMap`
(`libs/atlas-opcodes/producer.go:65-69`) is unchanged, so every entry this task
edits must keep its validator key.

- [ ] **Step 3: IDA verification of per-version usPetSkill bits (blocking for Step 4 values)**

For each entry below, use `select_instance(port)` to the matching IDB, `func_query` with `name_regex`, and record the evidence (address + expression) in the commit message body. **A bit that fails verification is omitted, not guessed:**

1. **jms consumeHP = 0x20** — already verified this task (`GetUpgradePetSkill(pet) & 0x20` in jms `TryConsumePetHP` `0xa26d8a`). Re-cite.
2. **jms consumeMP (expected 0x40)** — decompile jms `CUserLocal::TryConsumePetMP` (find via `func_query name_regex=TryConsumePetMP`); confirm the `GetUpgradePetSkill & mask` constant.
3. **jms autoSpeaking** — decompile jms `CPet::AutoSpeakingByEvent`; record the usPetSkill mask it tests.
4. **v83 autoSpeaking = 0x100** — already verified (`CPet::AutoSpeakingByEvent 0x70761f`). Re-cite.
5. **v84/v87/v95 autoSpeaking** — decompile each version's `AutoSpeakingByEvent` (v84/v87 functions may need locating via the v83 pattern) and record the mask. GMS clients read usPetSkill **only** for autoSpeaking (design §1.1), so autoSpeaking is the only GMS table entry unless new evidence appears.
6. **v61/v72/v79 autoSpeaking** — `CPet::AutoSpeakingByEvent` is named in both the v61 (`0x6162d1`) and v72 (`0x66eea7`) IDBs; locate the v79 equivalent via the same pattern. Record each version's usPetSkill mask, or omit the table entry if the function does not test one. Omission is the safe outcome: a legacy pet with no configured bit encodes 0, i.e. byte-identical to today.

- [ ] **Step 4: petSkill writer tables**

Add an `options.petSkill` map to the `"CharacterInventoryChange"` and `"SetField"` writer entries in every template (both writers encode pet cash assets — the inventory-change re-announce and the full character load respectively). Populate **only Step-3-verified bits**, e.g. (values illustrative until Step 3 confirms):

- gms_61/72/79/83/84/87/95 (per-version, as verified): `{"petSkill": {"autoSpeaking": "0x100"}}`
- jms_185 (as verified): `{"petSkill": {"consumeHP": "0x20", "consumeMP": "0x40", "autoSpeaking": "0x100"}}`

A pet flag bit with no table entry encodes as absent (Task 3) — sparse tables are behavior-complete because only the JMS auto-pot gate and autoSpeaking read these bits in supported clients.

- [ ] **Step 5: Validate + commit**

Run, from the worktree root:

```bash
python3 -c "import json,glob; [json.load(open(f)) for f in glob.glob('services/atlas-configurations/seed-data/templates/template_*.json')]"
tools/template-opcode-order-guard.sh
grep -c '"skillGate"' services/atlas-configurations/seed-data/templates/template_*.json   # expect 1 for the nine routed templates (gms_48 included), 0 for gms_12/92
(cd services/atlas-configurations && go test -race ./... && go build ./...)
```

Expected: JSON parses; opcode-order guard clean; exactly nine templates carry `skillGate`; configurations tests/build clean.

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(configurations): skillGate options, petSkill writer tables, revive gms_95 pet handlers"
```

(Commit body: the Step 3 evidence table — version, function, address, mask.)

---

### Task 15: legacy version coverage (v48/v61/v72/v79) — codec gate, fixtures, matrix, data check

*(Added by the rebase revision. Runs after Task 13 and before Task 14's sweep.)*

**Files:**
- Modify: `libs/atlas-packet/pet/serverbound/item_use.go` (version-gate the leading `petId`)
- Modify: `libs/atlas-packet/pet/serverbound/item_use_test.go`
- Modify: `docs/packets/audits/gms_v48/PetItemUse.json`, `gms_v61/*.json`, `gms_v72/*.json`, `gms_v79/*.json` (evidence records, written by the tooling — never hand-edited)
- Regenerate: `docs/packets/audits/STATUS.md` + `status.json`
- Create: `docs/tasks/task-139-pet-auto-pot-validation/coverage-manifest.yaml`

**Interfaces:**
- Consumes: the `pet/serverbound.ItemUse` codec.
- Produces: `PET_AUTO_POT` ✅ on gms_v48/v61/v72/v79; a codec whose `PetId()` is zero exactly when the client sent no pet id (Task 12 depends on this).

- [ ] **Step 1: Version-gate the leading `petId` (v48 only)**

v61/v72/v79 match the current codec byte-for-byte; **v48 does not** — it omits the 8-byte locker SN entirely (design §1.7, encoder `0x70dc8d`). Gate both directions:

```go
// The pet locker SN leads this packet from the v61 revision onward; the v48
// client is single-pet and sends none (encoder @0x70dc8d writes only the
// mitigation byte, updateTime, slot and itemId). IDA-verified. Boundary pinned
// at 61 — the verified edge of the available IDB set.
if (t.IsRegion("GMS") && t.MajorAtLeast(61)) || t.Region() == "JMS" {
	m.petId = r.ReadUint64()
}
```

`Decode`/`Encode` currently ignore the context — thread `ctx` through both and take `t := tenant.MustFromContext(ctx)`. Leaving `petId` at zero on v48 is what Task 12 reads as "absent" (a real Atlas pet id is never 0). **The five already-verified cells must not move**: assert their fixtures byte-for-byte.

- [ ] **Step 2: Byte fixtures for the four legacy cells**

Follow `docs/packets/audits/VERIFYING_A_PACKET.md` (or dispatch `/verify-packet` once per cell). Re-derive each read order in-tool rather than trusting this plan, then confirm it matches:

| Version | Encoder | Opcode | Layout |
|---|---|---|---|
| gms_v48 | `0x70dc8d` | 117 (0x75) | `Encode1(byte) · Encode4(updateTime) · Encode2(slot) · Encode4(itemId)` — **no petSN** |
| gms_v61 | `0x831ab9` | 142 (0x8E) | `buffer(petSN,8) · Encode1(byte) · Encode4(updateTime) · Encode2(slot) · Encode4(itemId)` |
| gms_v72 | `0x903f8b` | 165 (0xA5) | identical to v61 |
| gms_v79 | `0x9552d0` | 167 (0xA7) | identical to v61 |

Add four `// packet-audit:verify packet=pet/serverbound/PetItemUse version=gms_vNN ida=0x…` markers alongside the five existing ones and extend the round-trip table with the legacy contexts. For v61/v72/v79 **no codec change is expected** — if a fixture forces one there, stop: the layout claim is wrong and design §1.2 needs revisiting, not the test relaxing.

- [ ] **Step 2a: Re-harvest the gms_48 evidence record**

`docs/packets/audits/gms_v48/PetItemUse.json` currently records
`"IDAComment": "function not found in IDB"` and a ⬜ verdict — an artifact of the
encoder being unnamed at harvest time. It is named now
(`CWvsContext__SendStatChangeItemUseRequestByPetQ` @ `0x70dc8d`). Re-run the
harvest for that cell against the v48 IDB so the record carries a real address
and read order; do not hand-edit the JSON.

Target the IDB with **`-ida-database <session-id>`** (from `idb_list`), not
`-ida-port` — task-138 (#1087, on main) made the session id the preferred
selector when many IDBs are open on one server, which is the case here.

- [ ] **Step 2: Regenerate and check the matrix**

```bash
packet-audit matrix
packet-audit matrix --check
packet-audit fname-doc --check
packet-audit operations --check
```

Expected: the three legacy cells move 🟡ᶠ → ✅ and gms_48 moves ⬜ → ✅; all four commands exit 0. Note `matrix --check` bans a stale n-a while the feature family is present, so the v48 cell must actually promote — a lingering ⬜ fails the gate. Regenerate the matrix **after** any merge from main (`bug_packet_matrix_toolsha_reads_git_head`).

- [ ] **Step 3: Legacy equip-attribute data check (design §7 note 2a — the one open unknown)**

The `equipAbility` gate reads pet-ability attributes that atlas-data must expose (Task 5). Only v83 WZ is available locally, so confirm against a live legacy tenant before calling this done: query atlas-data for a pet-ability equip on a v61/v72/v79 tenant (`/api/data/equipment/{itemId}` with that tenant's `REGION`/`MAJOR_VERSION`/`MINOR_VERSION` headers — see the WZ-inspection notes) and record whether `consumeHP`/`consumeMP` are present after re-ingest.

- If present: nothing further; the gate is satisfiable on legacy.
- If the legacy WZ has no such attributes (or no `1812002`-equivalent item): **do not** guess a fallback. Record the finding in `context.md`, keep `equip_data_missing` as a distinct rejection reason so it is diagnosable in logs, and raise the gate choice for that version with the user — a version where no player can satisfy the gate needs an explicit decision, not a silent permissive default.

- [ ] **Step 4: ~~Confirm the gms_48 n-a~~ — DONE during the rebase revision; v48 is in scope**

Resolved, not deferred: v48 has the feature. Encoder `0x70dc8d` (opcode 0x75, no
petSN), senders `TryConsumePetHP` `0x6a840c` / `TryConsumePetMP` `0x6a8596`,
keymap ids at `CFuncKeyMappedMan+896/+900` via `OnPetConsumeItemInit`
`0x4e5eb7`, same worn-equip gate (`CPet+264/+272`). All six symbols plus the
three `GW_ItemSlot*::RawDecode` functions are named in the v48 IDB now. The
`?TryConsumePetMP@CUserLocal@@` label at `0x7204db` was a mis-port (job 132 /
`DarkKnightBerserkId` 1320006 / bodyless opcode 86) and is renamed
`CUserLocal__TryDoBerserk_job132_skill1320006`. See design §1.7. This step
survives only as the record of *why* the matrix disagreed.

- [ ] **Step 5: Coverage manifest + commit**

Write `coverage-manifest.yaml` declaring every op × version this task touches (the eight `PET_AUTO_POT` cells, the pet cash-asset encode per version, the jms type-28 arm) so `packet-completeness-critic` can diff claims against the branch delta at review time.

```bash
git add libs/atlas-packet/pet/serverbound/item_use_test.go docs/packets/audits/ docs/tasks/task-139-pet-auto-pot-validation/coverage-manifest.yaml
git commit -m "test(atlas-packet): byte-fixture PET_AUTO_POT for gms_v61/v72/v79; promote matrix cells"
```

---

### Task 14: Full verification sweep + context.md

**Files:**
- Create: `docs/tasks/task-139-pet-auto-pot-validation/context.md` (written alongside this plan; update if implementation deviated)

- [ ] **Step 1: Per-module gates**

For each changed module, from its directory:

```bash
for m in libs/atlas-constants libs/atlas-packet \
         services/atlas-data/atlas.com/data \
         services/atlas-pets/atlas.com/pets \
         services/atlas-consumables/atlas.com/consumables \
         services/atlas-channel/atlas.com/channel \
         services/atlas-configurations/atlas.com/configurations; do
  (cd "$m" && go test -race ./... && go vet ./... && go build ./...) || echo "FAILED: $m"
done
```

Expected: no `FAILED:` lines.

- [ ] **Step 2: Docker bake (mandatory — go.work will not catch Dockerfile COPY gaps)**

From the worktree root:

```bash
docker buildx bake atlas-data atlas-pets atlas-consumables atlas-channel atlas-configurations
```

Expected: all targets build. (No new shared lib was added, so no Dockerfile edits are expected — but bake proves it.)

- [ ] **Step 3: Redis key guard**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
tools/template-opcode-order-guard.sh
```

Expected: all clean. (No new redis or goroutine usage in this task, but the guards are unconditional per CLAUDE.md; `tools/lint.sh` with no flags fixes formatting in place — run that first, then `--check`. The opcode-order guard applies because Task 13 touches the seed templates.)

- [ ] **Step 4: Acceptance-criteria walkthrough**

Check each PRD acceptance item against the code and tests; for the two runtime-only items (pouch use end-to-end, auto-pot pass-after-pouch), note in `context.md` that they require a deployed env with re-ingested data — do not claim them verified from unit tests alone.

- [ ] **Step 5: Commit any stragglers and verify branch**

```bash
git status --short   # expect clean or only intended files
git rev-parse --show-toplevel   # must end with .worktrees/task-139-pet-auto-pot-validation
git branch --show-current       # must be task-139-pet-auto-pot-validation
```

Then run the code-review step (`superpowers:requesting-code-review`) before any PR.

---

## Self-Review Notes (spec coverage)

- FR-1 specific pet (exists/owned/spawned) → Task 12 (`evaluateAutoPot`, petId narrowing, fetch-miss reject).
- FR-2 alive → Task 12 (`character_dead`).
- FR-3 version-family gate → Task 12 (`resolveSkillSources`) + Task 13 (`skillGate` config); fail-closed `skill_gate_unconfigured`; `equip_data_missing` deploy-ordering reason (design §7).
- FR-4 unstick + warn → Task 12 (`reject`, shared `enableActions`); no double-unstick (reject *instead of* forward).
- FR-5 pass-through unchanged → Task 12 (identical forward call).
- FR-6 buffSkill log-only → Task 12 (logged in warn fields + debug line; semantics comment).
- FR-7 classification 519 → Task 1.
- FR-8 data-driven pouch apply/remove → Tasks 4, 7 (keys from WZ `info`, `add` flag; no per-item hardcoding).
- FR-9 flag persistence → column pre-exists; updater + tx in Task 6.
- FR-10 flag projection → Task 6 event + Task 11 channel consumer (REST `flag` already exposed both sides).
- FR-11 target pet → Tasks 8, 9 (jms pet-selection petId on the wire, threaded via `PetId` command field).
- FR-12 client sync, DOM-25 → Tasks 3, 11, 13 (usPetSkill via config-resolved sparse tables; verified bits only).
- FR-13 non-regression → Task 3 zero-flag byte-identical test; Task 12 identical forward; server enforces exactly the client family's own gate.
- FR-1a single-pet (v48) pet resolution → Task 12 (spawned-pet lookup when the decoded `petId` is 0, never as a fallback for petId-bearing versions).
- FR-14 all nine versions configured → Task 13 Step 1 (nine templates, gms_48 a new entry) + Step 5 grep assertion; exclusions evidenced in design §3.6.
- FR-15 codec gate + legacy fixtures → Task 15 Steps 1–2a (matrix promotion is the machine check; a prose claim is not acceptance).
- FR-16 pet-item trailer version gates → Task 3 Step 3b + per-version length assertions.
- ~~gms_95 dead handler fix~~ → dropped, already fixed on main (Task 13 Step 2 records the re-check).
- Consumables `PetId` type alignment (design §6) → Task 7 Step 3.
- Design §9 deferred verifications → Task 8 Step 1 (jms type-28 layout), Task 13 Step 3 (usPetSkill bits), Task 12 position normalization (worn-cash −100 offset resolved during planning).
- **Deliberate deviation from design §8**: consumables-side unit coverage is a classification pin, not a full `ConsumePetSkillPouch` failure-path harness — that function is REST+Kafka side-effecting and the consumables test convention (`consumable/processor_test.go`) is pure-function-only; there is no mock/httptest precedent to follow, and inventing one is out of scope. Its failure paths are structurally identical to the reviewed `ConsumeCashPetFood` and are exercised at runtime via the standard ConsumeError machinery.
