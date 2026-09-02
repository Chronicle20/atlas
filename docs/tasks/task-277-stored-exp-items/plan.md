# Stored-EXP Items (Writ of Solomon) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the two halves of the stored-EXP feature — `USE_SOLOMON_ITEM` credits a character's `gachapon_experience` counter from a Writ of Solomon item (classification 237), and `USE_GACHA_EXP` redeems the whole counter into real EXP and zeroes it — across the eight client versions that send them.

**Architecture:** Wire decode lands in `libs/atlas-packet` (no version gate; the divergence is opcode-only, owned by the seed templates). `atlas-channel` decodes and dispatches. `atlas-consumables` owns item ownership, the reserve/consume saga, and eligibility (`info/maxLevel`, non-zero-balance interlock). `atlas-character` is the sole writer of `characters` and owns two new Kafka commands: `CREDIT_STORED_EXPERIENCE` and `REDEEM_STORED_EXPERIENCE`. `atlas-data` exposes the two WZ fields the feature needs (`info/maxLevel`, `spec/exp`).

**Tech Stack:** Go 1.27, GORM, Kafka (segmentio), JSON:API REST between services, `packet-audit` coverage tooling.

**Spec:** `docs/tasks/task-277-stored-exp-items/design.md` (PRD: `docs/tasks/task-277-stored-exp-items/prd.md`)

## Global Constraints

- **Worktree:** all work happens in `.worktrees/task-277-stored-exp-items/`. Never edit the main repo.
- **In-scope versions (eight):** `gms_v72`, `gms_v79`, `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, `jms_v185`. `gms_v12`, `gms_v48`, `gms_v61` are OUT of scope and get positive-absence evidence instead.
- **Opcodes (design §1.2, read off the `COutPacket` ctor at each send site — do not re-derive, do not guess):**

  | Column | `USE_SOLOMON_ITEM` | `USE_GACHA_EXP` |
  |---|---|---|
  | gms_v72 | `0x9C` (156) | `0x9D` (157) |
  | gms_v79 | `0x9B` (155) | `0x9C` (156) |
  | gms_v83 | `0x9D` (157) | `0x9E` (158) |
  | gms_v84 | `0xA1` (161) | `0xA2` (162) |
  | gms_v87 | `0xA5` (165) | `0xA6` (166) |
  | gms_v92 | `0xB2` (178) | `0xB3` (179) |
  | gms_v95 | `0xB5` (181) | `0xB6` (182) |
  | jms_v185 | `0x71` (113) | `0x72` (114) |

- **IDA send-site addresses (design §1.1), for `packet-audit:verify` markers and registry `ida.address`:**

  | Column | `CWvsContext::SendExpUpItemUseRequest` | `CWvsContext::SendTempExpUseRequest` |
  |---|---|---|
  | gms_v72 | `0x90cb20` | `0x90cd28` |
  | gms_v79 | `0x95dee8` | `0x95e0f0` |
  | gms_v83 | `0xa12685` | `0xa1288f` |
  | gms_v84 | `0xa5cac2` | `0xa5cccc` |
  | gms_v87 | `0xaa80ac` | `0xaa82b6` |
  | gms_v92 | `0x9b0ab0` | `0x9b0d20` |
  | gms_v95 | `0x9db1c0` | `0x9db430` |
  | jms_v185 | `0xaf883d` | `0xaf8a40` |

- **Wire bodies (invariant on every column — NO `MajorAtLeast` gate on either codec):**
  - `USE_SOLOMON_ITEM`: `updateTime:uint32, slot:int16, itemId:uint32` — byte-identical to the existing `inventory/serverbound.ItemUse`.
  - `USE_GACHA_EXP`: `updateTime:uint32`. Nothing else.
- **Item family:** classification **237** (`itemId / 10000 == 237`, i.e. `2370000`–`2379999`), from `is_exp_up_item` (gms_v95 `@0x5078be`).
- **Level bounds:** upper bound only. Solomon use rejects when `characterLevel > info/maxLevel`; a missing/zero `maxLevel` means NO upper bound. Redeem rejects when `characterLevel > 50`.
- **Naming:** the DB column and stat stay `gachapon_experience` / `stat.TypeGachaponExperience` (no migration). New Kafka commands and Go symbols use "stored experience". This mismatch is deliberate — add a comment at each new command noting `gachapon_experience` is the community misnomer for the client's `GW_CharacterStat::nTempEXP`.
- **Never** land a placeholder comment, a stubbed handler, or an unimplemented status response. Every branch this plan describes is fully specified; if one is not, stop and ask rather than filling it with a marker comment.
- Module roots (`go build` / `go test` cwd) are named per task.

---

## Task 1: `libs/atlas-packet` — `SolomonItemUse` audit-only wrapper

### Files

- `libs/atlas-packet/inventory/serverbound/item_use.go` — add `CharacterItemUseSolomonHandle` to the const block at lines 13-17
- `libs/atlas-packet/inventory/serverbound/solomon_item_use.go` — **new file**
- `libs/atlas-packet/inventory/serverbound/solomon_item_use_test.go` — **new file**

Patterns to copy: `libs/atlas-packet/inventory/serverbound/summon_bag_item_use.go:1-21` (the wrapper, verbatim shape) and `libs/atlas-packet/character/serverbound/item_cancel_test.go:1-61` (marker block + round-trip + byte fixture).

Module root: `libs/atlas-packet` (`github.com/Chronicle20/atlas/libs/atlas-packet`).

**Interfaces:**
- Produces: `serverbound.CharacterItemUseSolomonHandle` (string const), `serverbound.SolomonItemUse` struct, `serverbound.NewSolomonItemUse() SolomonItemUse`.
- Consumes: nothing.

**Why a wrapper and not a reuse of `ItemUse`:** `USE_SOLOMON_ITEM`'s body is byte-identical to `USE_ITEM`, but its client send site is a *different function*. Collapsing them onto one packet id would pin this op's evidence to the potion sender's decompile — a manufactured ✅. The repo has settled precedent (`SummonBagItemUse`, `ReturnScrollItemUse`): one audit-only wrapper per op. Nothing calls it; `atlas-channel` keeps decoding the shared `ItemUse`.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/inventory/serverbound/solomon_item_use_test.go`, package `serverbound`, imports `bytes`, `testing`, and `pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"`.

`TestSolomonItemUseRoundTrip` — iterate `pt.Variants`, `t.Run(v.Name, ...)`, `ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)`, build `input := SolomonItemUse{ItemUse: ItemUse{operation: CharacterItemUseSolomonHandle, updateTime: 0x0A0B0C0D, source: 0x0203, itemId: 2370000}}`, `output := NewSolomonItemUse()`, `pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)`, then assert `output.UpdateTime()`, `output.Source()`, `output.ItemId()` each equal the input's, and `output.Operation() == CharacterItemUseSolomonHandle`.

The function carries all eight `packet-audit:verify` markers directly above it (exact form, one per line, copied from `item_cancel_test.go:10-17`):

```
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=gms_v72 ida=0x90cb20
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=gms_v79 ida=0x95dee8
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=gms_v83 ida=0xa12685
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=gms_v84 ida=0xa5cac2
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=gms_v87 ida=0xaa80ac
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=gms_v92 ida=0x9b0ab0
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=gms_v95 ida=0x9db1c0
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=jms_v185 ida=0xaf883d
```

`TestSolomonItemUseByteFixtureV95` — pins the wire against the v95 send site. Doc comment must record the three encode calls (`Encode4(get_update_time())`, `Encode2(nPOS)`, `Encode4(nItemID)`) from `CWvsContext::SendExpUpItemUseRequest` @`0x9db1c0`, and that the body is invariant across all eight columns.

```go
func TestSolomonItemUseByteFixtureV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	in := SolomonItemUse{ItemUse: ItemUse{operation: CharacterItemUseSolomonHandle, updateTime: 0x0A0B0C0D, source: 0x0203, itemId: 2370000}}
	got := pt.Encode(t, ctx, in.Encode, nil)
	want := []byte{
		0x0D, 0x0C, 0x0B, 0x0A, // updateTime  Encode4 (LE)
		0x03, 0x02, // slot        Encode2 (LE)
		0xD0, 0x29, 0x42, 0x00, // itemId 2370000 = 0x002429D0 Encode4 (LE)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v95 bytes:\n got %x\nwant %x", got, want)
	}
}
```

`TestSolomonItemUseByteFixtureV72` — identical body, `pt.CreateContext("GMS", 72, 1)`, same `want`, error message `"v72 bytes:..."`. Doc comment records the v72 send site `0x90cb20` and that no version gate applies.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./inventory/serverbound/ -run 'SolomonItemUse' -v`
Expected: FAIL — `undefined: SolomonItemUse`, `undefined: CharacterItemUseSolomonHandle`.

- [ ] **Step 3: Add the const**

In `libs/atlas-packet/inventory/serverbound/item_use.go`, extend the const block:

```go
const (
	CharacterItemUseHandle           = "CharacterItemUseHandle"
	CharacterItemUseTownScrollHandle = "CharacterItemUseTownScrollHandle"
	CharacterItemUseSummonBagHandle  = "CharacterItemUseSummonBagHandle"
	CharacterItemUseSolomonHandle    = "CharacterItemUseSolomonHandle"
)
```

- [ ] **Step 4: Write the wrapper**

Create `libs/atlas-packet/inventory/serverbound/solomon_item_use.go`:

```go
package serverbound

// SolomonItemUse is an AUDIT-ONLY codec. USE_SOLOMON_ITEM shares its wire body
// with USE_ITEM (Encode4 updateTime + Encode2 slot + Encode4 itemId), but its
// client send site is CWvsContext::SendExpUpItemUseRequest — a different
// function, gated on is_exp_up_item(nItemID) (itemId/10000 == 237) rather than
// the potion families. Collapsing it onto InventoryItemUse's packet id would
// pin every cell's evidence to the potion sender's decompile — a manufactured
// ✅. One wrapper per op = one packet id, one audit report and one evidence key
// per op, exactly as docs/packets/audits/VERIFYING_A_PACKET.md "Shared-model
// ops" prescribes.
//
// Nothing calls this: atlas-channel's CharacterItemUseSolomonHandleFunc keeps
// decoding the shared ItemUse. Do not "simplify" it away (see task-229).
//
// packet-audit:fname CWvsContext::SendExpUpItemUseRequest
type SolomonItemUse struct {
	ItemUse
}

func NewSolomonItemUse() SolomonItemUse {
	return SolomonItemUse{ItemUse: NewItemUse(CharacterItemUseSolomonHandle)}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./inventory/serverbound/ -v`
Expected: PASS, all three new tests plus the existing `ItemUse` tests.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/inventory/serverbound/item_use.go libs/atlas-packet/inventory/serverbound/solomon_item_use.go libs/atlas-packet/inventory/serverbound/solomon_item_use_test.go
git commit -m "feat(packet): add USE_SOLOMON_ITEM audit-only codec"
```

---

## Task 2: `libs/atlas-packet` — `StoredExperienceUse` codec

### Files

- `libs/atlas-packet/character/serverbound/stored_experience_use.go` — **new file**
- `libs/atlas-packet/character/serverbound/stored_experience_use_test.go` — **new file**

Patterns to copy: `libs/atlas-packet/character/serverbound/item_cancel.go:1-41` (whole-file shape: const, struct, accessors, `Operation`, `String`, `Encode`, `Decode`) and `libs/atlas-packet/character/serverbound/distribute_ap.go:32-46` (the `WriteInt(m.updateTime)` / `r.ReadUint32()` idiom for a `uint32` updateTime).

Module root: `libs/atlas-packet`.

**Interfaces:**
- Produces: `serverbound.CharacterUseStoredExperienceHandle` (string const), `serverbound.StoredExperienceUse` struct with `UpdateTime() uint32`, `Operation() string`, `String() string`, `Encode`, `Decode`.
- Consumes: nothing.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/character/serverbound/stored_experience_use_test.go`, package `serverbound`, imports `bytes`, `testing`, `pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"`.

`TestStoredExperienceUseRoundTrip` — iterate `pt.Variants`, `input := StoredExperienceUse{updateTime: 0x0A0B0C0D}`, `output := StoredExperienceUse{}`, `pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)`, assert `output.UpdateTime() == input.UpdateTime()`. Carries all eight markers:

```
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=gms_v72 ida=0x90cd28
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=gms_v79 ida=0x95e0f0
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=gms_v83 ida=0xa1288f
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=gms_v84 ida=0xa5cccc
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=gms_v87 ida=0xaa82b6
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=gms_v92 ida=0x9b0d20
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=gms_v95 ida=0x9db430
// packet-audit:verify packet=character/serverbound/StoredExperienceUse version=jms_v185 ida=0xaf8a40
```

`TestStoredExperienceUseByteFixtureV95` — doc comment records that `CWvsContext::SendTempExpUseRequest` @`0x9db430` builds `COutPacket(182)` then a single `Encode4(get_update_time())` and nothing else, then `SendPacket` + `SetExclRequestSent(1)`; the body is invariant across all eight columns.

```go
func TestStoredExperienceUseByteFixtureV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	got := pt.Encode(t, ctx, StoredExperienceUse{updateTime: 0x0A0B0C0D}.Encode, nil)
	want := []byte{0x0D, 0x0C, 0x0B, 0x0A} // updateTime Encode4 (LE) /*0x9db430*/
	if !bytes.Equal(got, want) {
		t.Errorf("v95 bytes:\n got %x\nwant %x", got, want)
	}
}
```

`TestStoredExperienceUseByteFixtureV72` — same shape, `pt.CreateContext("GMS", 72, 1)`, same `want`, doc comment naming `0x90cd28`. This one also asserts total length is exactly 4 (`if len(got) != 4 { t.Fatalf(...) }`) — the op's defining property is that it carries nothing but the tick.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./character/serverbound/ -run 'StoredExperienceUse' -v`
Expected: FAIL — `undefined: StoredExperienceUse`.

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/character/serverbound/stored_experience_use.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CharacterUseStoredExperienceHandle = "CharacterUseStoredExperienceHandle"

// StoredExperienceUse - USE_GACHA_EXP. Redeems the whole stored-EXP counter
// (GW_CharacterStat::nTempEXP, persisted by Atlas as the misnamed
// `gachapon_experience` column) into real character EXP.
//
// CUIStatusBar::TryUseTempExp is the sole caller of the sender; it gates on the
// EXP-bar click rect, characterLevel <= 50, nTempEXP > 0 and a YesNo confirm.
// The sender itself re-checks the job guard and characterLevel <= 50. None of
// that is on the wire — the body is the tick and nothing else, on every version
// that carries the op, so there is no version gate here.
//
// packet-audit:fname CWvsContext::SendTempExpUseRequest
type StoredExperienceUse struct {
	updateTime uint32
}

func (m StoredExperienceUse) UpdateTime() uint32 { return m.updateTime }

func (m StoredExperienceUse) Operation() string {
	return CharacterUseStoredExperienceHandle
}

func (m StoredExperienceUse) String() string {
	return fmt.Sprintf("updateTime [%d]", m.updateTime)
}

func (m StoredExperienceUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *StoredExperienceUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./character/serverbound/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/character/serverbound/stored_experience_use.go libs/atlas-packet/character/serverbound/stored_experience_use_test.go
git commit -m "feat(packet): add USE_GACHA_EXP StoredExperienceUse codec"
```

---

## Task 3: Packet registry rows and positive-absence evidence

### Files

- `docs/packets/registry/gms_v72.yaml` — add two serverbound rows (opcodes 156, 157) in the serverbound opcode gap after `MOB_CRC_KEY_CHANGED_REPLY` (opcode 155, around line 2804-2812) and before `MOVE_PET` (161)
- `docs/packets/registry/gms_v79.yaml` — add two serverbound rows (opcodes 155, 156) after `MOB_CRC_KEY_CHANGED_REPLY` (154, around line 2522-2531) and before `CASH_ITEM_GACHAPON_BUTTON` (159)
- `docs/packets/registry/gms_v83.yaml` — add `packet:` to the two existing rows at lines 2914-2923
- `docs/packets/registry/gms_v84.yaml` — add `packet:` to the two existing rows near line 3685
- `docs/packets/registry/gms_v87.yaml` — add `packet:` to the two existing rows near line 3058
- `docs/packets/registry/gms_v92.yaml` — add `packet:` to the two existing rows near line 3291
- `docs/packets/registry/gms_v95.yaml` — add `packet:` to the two existing rows near line 3404
- `docs/packets/registry/jms_v185.yaml` — add `packet:` to the two existing rows near line 2637
- `docs/packets/feature-na-evidence.yaml` — add four entries

This task is a documented exception to the ≤6-file rule: all ten files receive the same two-line mechanical edit. Registry files are declarative YAML with no build step. See `context.md`.

Patterns to copy: `docs/packets/registry/gms_v72.yaml:2804-2811` (an `ida-discovered` row with `ida.address` and `note`); `docs/packets/registry/gms_v83.yaml:2296` (the `packet:` key placement — it sits between `fname:` and `provenance:`); `docs/packets/feature-na-evidence.yaml:8-20` (na entry shape).

**Interfaces:**
- Consumes: the packet names declared in Tasks 1-2 — `inventory/serverbound/InventorySolomonItemUse` and `character/serverbound/StoredExperienceUse`. These MUST match the `packet=` values in the `packet-audit:verify` markers byte-for-byte.
- Produces: registry rows that `packet-audit matrix` reads.

- [ ] **Step 1: Add the gms_v72 rows**

Insert into `docs/packets/registry/gms_v72.yaml`, in serverbound opcode order between opcode 155 and 161:

```yaml
- op: USE_SOLOMON_ITEM
  direction: serverbound
  opcode: 156
  fname: CWvsContext::SendExpUpItemUseRequest
  packet: inventory/serverbound/InventorySolomonItemUse
  provenance: ida-discovered
  ida:
    address: 9489184
  note: 'task-277: v72 opcode READ off the COutPacket ctor at CWvsContext::SendExpUpItemUseRequest @0x90cb20 (=9489184) — COutPacket(156)/0x9C, then Encode4(get_update_time)/Encode2(nPOS)/Encode4(nItemID). Body invariant vs v83..v95. Closes the disputed ⬜ from FR-3: v72 does send this op; the blank was a registry gap, not an absence.'
- op: USE_GACHA_EXP
  direction: serverbound
  opcode: 157
  fname: CWvsContext::SendTempExpUseRequest
  packet: character/serverbound/StoredExperienceUse
  provenance: ida-discovered
  ida:
    address: 9489704
  note: 'task-277: v72 opcode READ off the COutPacket ctor at CWvsContext::SendTempExpUseRequest @0x90cd28 (=9489704) — COutPacket(157)/0x9D then a single Encode4(get_update_time). Body invariant vs v83..v95. Closes the disputed ⬜ from FR-3.'
```

- [ ] **Step 2: Add the gms_v79 rows**

Insert into `docs/packets/registry/gms_v79.yaml`, in serverbound opcode order between opcode 154 and 159:

```yaml
- op: USE_SOLOMON_ITEM
  direction: serverbound
  opcode: 155
  fname: CWvsContext::SendExpUpItemUseRequest
  packet: inventory/serverbound/InventorySolomonItemUse
  provenance: ida-discovered
  ida:
    address: 9822440
  note: 'task-277: v79 opcode READ off the COutPacket ctor at CWvsContext::SendExpUpItemUseRequest @0x95dee8 (=9822440) — COutPacket(155)/0x9B, then Encode4/Encode2/Encode4. Body invariant vs v83..v95. Closes the disputed ⬜ from FR-3.'
- op: USE_GACHA_EXP
  direction: serverbound
  opcode: 156
  fname: CWvsContext::SendTempExpUseRequest
  packet: character/serverbound/StoredExperienceUse
  provenance: ida-discovered
  ida:
    address: 9822960
  note: 'task-277: v79 opcode READ off the COutPacket ctor at CWvsContext::SendTempExpUseRequest @0x95e0f0 (=9822960) — COutPacket(156)/0x9C then a single Encode4(get_update_time). Body invariant vs v83..v95. Closes the disputed ⬜ from FR-3.'
```

- [ ] **Step 3: Add `packet:` to the six existing columns**

For each of `gms_v83.yaml`, `gms_v84.yaml`, `gms_v87.yaml`, `gms_v92.yaml`, `gms_v95.yaml`, `jms_v185.yaml`: locate the existing `- op: USE_SOLOMON_ITEM` and `- op: USE_GACHA_EXP` serverbound rows and insert one line after each row's `fname:`:

```yaml
  packet: inventory/serverbound/InventorySolomonItemUse
```
```yaml
  packet: character/serverbound/StoredExperienceUse
```

Do NOT change the `opcode:`, `provenance:`, `ida:` or `note:` of these rows — design §1.2 confirmed the existing opcodes are correct on all six.

- [ ] **Step 4: Add the four na-evidence entries**

Append to the `entries:` list in `docs/packets/feature-na-evidence.yaml` — four entries, `USE_SOLOMON_ITEM` and `USE_GACHA_EXP` × `gms_v48` and `gms_v61`. Each carries this evidence text (substituting the version):

> `<version>` does not carry the stored-EXP subsystem at all. Both IDBs resolve full MSVC-mangled symbols (`?SendPacket@CClientSocket@@QAEXABVCOutPacket@@@Z` and `?GetItemInfo@CItemInfo@@...` both resolve), and in both, `?SendExpUpItemUseRequest@CWvsContext@@QAEXJJ@Z`, `?SendTempExpUseRequest@CWvsContext@@QAEXXZ` **and** `?GetMaxLEV@CItemInfo@@QAEJJ@Z` are all absent — sender, redeemer, and the WZ accessor the sender gates on. Positive absence, not a search miss. (task-277)

- [ ] **Step 5: Verify the registry parses and the ops resolve**

Run: `go run ./tools/packet-audit operations --check`
Expected: exit 0.

Run: `go run ./tools/packet-audit fname-doc --check`
Expected: exit 0.

If `fname-doc --check` reports the two fnames are undocumented, add them to the fname doc it names in its own error output — do not suppress the check.

- [ ] **Step 6: Commit**

```bash
git add docs/packets/registry/ docs/packets/feature-na-evidence.yaml
git commit -m "docs(packets): register USE_SOLOMON_ITEM/USE_GACHA_EXP on v72+v79, pin packet names, add v48/v61 na-evidence"
```

---

## Task 4: Evidence records and coverage-matrix promotion

### Files

- `docs/packets/evidence/gms_v72/inventory.serverbound.InventorySolomonItemUse.yaml` — **new file** (and the same for v79, v83, v84, v87, v92, v95, jms_v185)
- `docs/packets/evidence/gms_v72/character.serverbound.StoredExperienceUse.yaml` — **new file** (and the same for the other seven columns)
- `docs/packets/audits/STATUS.md` — regenerated, do not hand-edit
- `docs/packets/audits/status.json` — regenerated, do not hand-edit

Sixteen evidence files, all the same four-field shape. Read-only reference: `docs/packets/IMPLEMENTING_A_PACKET.md` §0-4.

Patterns to copy: `docs/packets/evidence/gms_v83/character.serverbound.ItemCancel.yaml` (whole file — `packet`, `direction`, `version`, `category`, `ida.function`, `ida.address`, `ida.decompile_sha256`).

**Interfaces:**
- Consumes: the marker/registry packet names from Tasks 1-3, and the per-column IDA addresses in Global Constraints.
- Produces: sixteen promoted cells in the coverage matrix.

- [ ] **Step 1: Check what `matrix` currently reports**

Run: `go run ./tools/packet-audit matrix --check`
Expected: FAIL, naming the sixteen unpinned cells (the markers from Tasks 1-2 exist but have no evidence record).

- [ ] **Step 2: Write the sixteen evidence records**

Each file follows this shape exactly (shown for the v95 Solomon cell; substitute `packet`, `version`, `ida.function` and `ida.address` per the Global Constraints tables):

```yaml
packet: inventory/serverbound/InventorySolomonItemUse
direction: serverbound
version: gms_v95
category: TIER1-FIXTURE
ida:
    function: CWvsContext::SendExpUpItemUseRequest
    address: "0x9db1c0"
    decompile_sha256: <sha256 of the decompile, produced by the pinning step below>
```

`decompile_sha256` must NOT be invented. Produce it the way `IMPLEMENTING_A_PACKET.md` §4 prescribes — the `packet-audit` pinning path — from the live decompile of the named function at the named address. If a decompile for a column is unobtainable, STOP and report the column; do not fabricate a hash and do not silently drop the cell.

- [ ] **Step 3: Regenerate the matrix**

Run: `go run ./tools/packet-audit matrix`
Then: `go run ./tools/packet-audit matrix --check`
Expected: exit 0.

- [ ] **Step 4: Confirm the sixteen cells promoted and v48/v61 stayed ⬜**

Run: `grep -n 'USE_SOLOMON_ITEM\|USE_GACHA_EXP' docs/packets/audits/STATUS.md`
Expected: both rows show ✅ in the gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v92, gms_v95 and jms_v185 columns, and ⬜ (justified by the Task 3 na-evidence) in gms_v48 and gms_v61. A cell that did not promote is a failure to fix here, not a prose claim.

- [ ] **Step 5: Run the remaining packet-audit gates**

Run: `go run ./tools/packet-audit fname-doc --check && go run ./tools/packet-audit operations --check && go run ./tools/packet-audit gate-check --check && go run ./tools/packet-audit doc-freshness --check`
Expected: all exit 0.

- [ ] **Step 6: Commit**

```bash
git add docs/packets/evidence/ docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "docs(packets): pin stored-EXP evidence and promote 16 matrix cells"
```

---

## Task 5: `atlas-data` — expose `info/maxLevel` and `spec/exp`

### Files

- `services/atlas-data/atlas.com/data/consumable/rest.go` — add `SpecTypeExperience` to the const block (lines 10-36) and `MaxLevel` to `RestModel` (near `ReqLevel`, line 52)
- `services/atlas-data/atlas.com/data/consumable/reader.go` — parse both fields
- `services/atlas-data/atlas.com/data/consumable/reader_test.go` — new cases

Module root: `services/atlas-data/atlas.com/data`.

Patterns to copy: `services/atlas-data/atlas.com/data/consumable/reader.go:49-95` (the `i.GetIntegerWithDefault("<name>", 0)` info-node idiom) and `reader.go:124-133` (the `s.GetIntegerWithDefault(string(SpecTypeXxx), 0)` spec-node idiom).

**Interfaces:**
- Produces: JSON fields `maxLevel` (on the consumable resource root) and `exp` (inside the `spec` map).
- Consumes: nothing.

**Scope guard:** these two fields only. The broader unparsed-`spec` gap is explicitly a separate task per the PRD's non-goals — do not sweep other fields in.

- [ ] **Step 1: Write the failing test**

Add to `services/atlas-data/atlas.com/data/consumable/reader_test.go`, following the existing file's fixture-XML shape (read it first; copy its node-construction helper rather than inventing one).

`TestReadMaxLevelAndExperienceSpec` — table-driven over a Writ-of-Solomon-shaped consumable node:

| case | `info/maxLevel` | `spec/exp` | expect `RestModel.MaxLevel` | expect `Spec[SpecTypeExperience]` |
|---|---|---|---|---|
| both present | `200` | `3000` | `200` | `3000` |
| maxLevel absent | (node omitted) | `3000` | `0` | `3000` |
| exp absent | `200` | (node omitted) | `200` | `0` |
| both absent | (omitted) | (omitted) | `0` | `0` |

The "absent ⇒ 0" cases are load-bearing, not filler: a tenant whose `Item.wz` was ingested before this change serves exactly that shape, and downstream (Task 10) treats `maxLevel == 0` as "no upper bound" and `exp <= 0` as "reject, never destroy the Writ."

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-data/atlas.com/data && go test ./consumable/ -run 'MaxLevelAndExperience' -v`
Expected: FAIL — `RestModel` has no field `MaxLevel`; `undefined: SpecTypeExperience`.

- [ ] **Step 3: Add the const and the field**

In `rest.go`, add to the `SpecType` const block:

```go
	SpecTypeExperience           = SpecType("exp")
```

and to `RestModel`, immediately after `ReqLevel`:

```go
	MaxLevel        uint32             `json:"maxLevel"`
```

- [ ] **Step 4: Parse both fields**

In `reader.go`, alongside the other `info` reads (near `m.MasterLevel`):

```go
			m.MaxLevel = uint32(i.GetIntegerWithDefault("maxLevel", 0))
```

and inside the `spec` block, alongside the other spec reads:

```go
				m.Spec[SpecTypeExperience] = s.GetIntegerWithDefault(string(SpecTypeExperience), 0)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-data/atlas.com/data && go build ./... && go test ./consumable/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-data/atlas.com/data/consumable/
git commit -m "feat(data): expose consumable info/maxLevel and spec/exp"
```

---

## Task 6: `atlas-consumables` — consume the two new data fields

### Files

- `services/atlas-consumables/atlas.com/consumables/data/consumable/model.go` — add `SpecTypeExperience` to the const block (lines 5-33), `maxLevel uint32` to `Model` (near `reqLevel`, line 56), and a `MaxLevel()` accessor
- `services/atlas-consumables/atlas.com/consumables/data/consumable/rest.go` — add `MaxLevel` to `RestModel` (near `ReqLevel`) and map it in `Extract`
- `services/atlas-consumables/atlas.com/consumables/data/consumable/rest_test.go` — new case

Module root: `services/atlas-consumables/atlas.com/consumables`.

Patterns to copy: `services/atlas-consumables/atlas.com/consumables/data/consumable/model.go:113-116` (the `GetSpec` accessor already in place — no change needed there) and the existing `reqLevel` field/`Extract` mapping in the same package (read `rest.go`'s `Extract` before editing; mirror whatever it does for `ReqLevel`).

**Interfaces:**
- Consumes: the `maxLevel` and `spec.exp` JSON fields produced by Task 5.
- Produces: `consumable.SpecTypeExperience` (SpecType const) and `consumable.Model.MaxLevel() uint32`, both used by Task 10.

- [ ] **Step 1: Write the failing test**

Add to `services/atlas-consumables/atlas.com/consumables/data/consumable/rest_test.go` (read the file first and copy its existing extract-round-trip shape):

`TestExtractMaxLevelAndExperienceSpec` — build a `RestModel` with `MaxLevel: 200` and `Spec: map[SpecType]int32{SpecTypeExperience: 3000}`, run it through `Extract`, then assert:

- `m.MaxLevel() == 200`
- `val, ok := m.GetSpec(SpecTypeExperience)` gives `ok == true`, `val == 3000`

Second case: `RestModel{}` (zero value, empty spec map) extracts to `m.MaxLevel() == 0` and `GetSpec(SpecTypeExperience)` returning `ok == false`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./data/consumable/ -run 'MaxLevelAndExperience' -v`
Expected: FAIL — `undefined: SpecTypeExperience`; `m.MaxLevel` undefined.

- [ ] **Step 3: Add the const, field and accessor**

In `model.go`, add to the `SpecType` const block:

```go
	SpecTypeExperience           = SpecType("exp")
```

add `maxLevel uint32` to `Model` immediately after `reqLevel`, and add the accessor next to the other single-field accessors:

```go
// MaxLevel is the item's INCLUSIVE upper level bound (WZ info/maxLevel) — the
// only level gate the client applies to a Writ of Solomon
// (CItemInfo::GetMaxLEV, read by CWvsContext::SendExpUpItemUseRequest). There
// is no lower bound. Zero means the field is absent, which callers MUST treat
// as "no upper bound", never as "reject": a tenant whose Item.wz predates the
// maxLevel parse serves zero here.
func (m Model) MaxLevel() uint32 {
	return m.maxLevel
}
```

- [ ] **Step 4: Add the REST field and map it**

In `rest.go`, add `MaxLevel uint32 \`json:"maxLevel"\`` to `RestModel` immediately after `ReqLevel`, and map it in `Extract` exactly as `ReqLevel` is mapped.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./... && go test ./data/consumable/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/data/consumable/
git commit -m "feat(consumables): read consumable maxLevel and spec/exp"
```

---

## Task 7: Classification 237 constant and the credit command seam

### Files

- `libs/atlas-constants/item/constants.go` — add `ClassificationConsumableExpUpItem` to the consumable classification block (between `ClassificationBullet = 233` at line 45 and `ClassificationConsumableMonsterCard = 238` at line 46)
- `services/atlas-consumables/atlas.com/consumables/kafka/message/character/kafka.go` — add the command const and body struct
- `services/atlas-consumables/atlas.com/consumables/character/producer.go` — add the provider
- `services/atlas-consumables/atlas.com/consumables/character/processor.go` — add the method to `Processor` and `ProcessorImpl`

Module roots: `libs/atlas-constants` and `services/atlas-consumables/atlas.com/consumables` (build/test both).

Patterns to copy: `services/atlas-consumables/atlas.com/consumables/character/producer.go:13-25` (`changeHPCommandProvider`, verbatim shape) and `services/atlas-consumables/atlas.com/consumables/character/processor.go:62-64` (`ChangeHP`, verbatim shape).

**Interfaces:**
- Produces: `item.ClassificationConsumableExpUpItem` (used by Task 10); `character.Processor.CreditStoredExperience(f field.Model, characterId uint32, amount uint32, reason string) error` (used by Task 10); the wire command `CREDIT_STORED_EXPERIENCE` with body `{channelId, amount, reason}` (consumed by Task 8).
- Consumes: nothing.

The command name, body field names and JSON tags here MUST match Task 8's `atlas-character` side byte-for-byte — they are the same Kafka message.

- [ ] **Step 1: Add the classification constant**

In `libs/atlas-constants/item/constants.go`, inside the consumable classification block:

```go
	// ClassificationConsumableExpUpItem is the Writ of Solomon family
	// (2370000-2379999). Derived from is_exp_up_item, the sole gate on
	// CDraggableItem::OnDoubleClicked's path to
	// CWvsContext::SendExpUpItemUseRequest (gms_v95 @0x5078be):
	//     BOOL __cdecl is_exp_up_item(int nItemID) { return nItemID / 10000 == 237; }
	// A Writ CREDITS the stored-EXP counter (GW_CharacterStat::nTempEXP,
	// persisted by Atlas as the misnamed `gachapon_experience` column); it does
	// not award character EXP directly.
	ClassificationConsumableExpUpItem = Classification(237)
```

- [ ] **Step 2: Write the failing test**

Add to `services/atlas-consumables/atlas.com/consumables/character/producer_test.go` (create the file if it does not exist; if it does, copy its existing provider-assertion shape).

`TestCreditStoredExperienceCommandProvider` — build `f` for world 1 / channel 2, call `creditStoredExperienceCommandProvider(f, 1234, 3000, "SOLOMON_ITEM")`, take the single message from the provider, unmarshal its value into `character2.Command[character2.CreditStoredExperienceCommandBody]`, and assert:

| field | expected |
|---|---|
| `CharacterId` | `1234` |
| `WorldId` | `1` |
| `Type` | `"CREDIT_STORED_EXPERIENCE"` |
| `Body.ChannelId` | `2` |
| `Body.Amount` | `3000` |
| `Body.Reason` | `"SOLOMON_ITEM"` |

Also assert the message key equals `producer.CreateKey(1234)`.

- [ ] **Step 3: Run test to verify it fails**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./character/ -run 'CreditStoredExperience' -v`
Expected: FAIL — `undefined: creditStoredExperienceCommandProvider`.

- [ ] **Step 4: Add the command const and body**

In `services/atlas-consumables/atlas.com/consumables/kafka/message/character/kafka.go`, extend the const block and add the body struct:

```go
const (
	EnvCommandTopic  = "COMMAND_TOPIC_CHARACTER"
	CommandChangeMap = "CHANGE_MAP"
	CommandChangeHP  = "CHANGE_HP"
	CommandChangeMP  = "CHANGE_MP"
	// CommandCreditStoredExperience banks EXP into the character's stored-EXP
	// counter. The counter is the client's GW_CharacterStat::nTempEXP; Atlas
	// persists it in the `gachapon_experience` column, a community misnomer
	// kept to avoid a migration across atlas-login/-cashshop/-npc-shops.
	CommandCreditStoredExperience = "CREDIT_STORED_EXPERIENCE"
)

type CreditStoredExperienceCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    uint32     `json:"amount"`
	Reason    string     `json:"reason"`
}
```

- [ ] **Step 5: Add the provider**

In `services/atlas-consumables/atlas.com/consumables/character/producer.go`:

```go
func creditStoredExperienceCommandProvider(f field.Model, characterId uint32, amount uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.CreditStoredExperienceCommandBody]{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		Type:        character2.CommandCreditStoredExperience,
		Body: character2.CreditStoredExperienceCommandBody{
			ChannelId: f.ChannelId(),
			Amount:    amount,
			Reason:    reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 6: Add the processor method**

In `services/atlas-consumables/atlas.com/consumables/character/processor.go`, add to the `Processor` interface:

```go
	CreditStoredExperience(f field.Model, characterId uint32, amount uint32, reason string) error
```

and the implementation next to `ChangeMP`:

```go
func (p *ProcessorImpl) CreditStoredExperience(f field.Model, characterId uint32, amount uint32, reason string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(creditStoredExperienceCommandProvider(f, characterId, amount, reason))
}
```

If the package carries a generated or hand-written `ProcessorMock` used by `morph_coupon_test.go`, add the new method to it too — `var _ Processor = (*ProcessorImpl)(nil)` and the mock must both still compile.

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd libs/atlas-constants && go build ./... && go test ./...`
Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./... && go test ./character/... -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-constants/item/constants.go services/atlas-consumables/atlas.com/consumables/character/ services/atlas-consumables/atlas.com/consumables/kafka/message/character/kafka.go
git commit -m "feat(consumables): add classification 237 and the CREDIT_STORED_EXPERIENCE seam"
```

---

## Task 8: `atlas-character` — `CREDIT_STORED_EXPERIENCE`

### Files

- `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go` — command const (const block lines 13-37) and body struct
- `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go` — register and handle
- `services/atlas-character/atlas.com/character/character/administrator.go` — `SetGachaponExperience` `EntityUpdateFunction`
- `services/atlas-character/atlas.com/character/character/processor.go` — `Processor` interface entries plus the `...AndEmit` / `(mb *message.Buffer)` pair
- `services/atlas-character/atlas.com/character/character/processor_stored_experience_test.go` — **new file**

Module root: `services/atlas-character/atlas.com/character`.

Patterns to copy:
- Command chain: const `kafka.go:28` → body `kafka.go:164-167` → registration `consumer.go:73` → handler `consumer.go:264-273` → processor pair `processor.go:1418-1461`.
- `EntityUpdateFunction`: `administrator.go:276-282` (`SetExperience`).
- Test setup: `services/atlas-character/atlas.com/character/character/processor_test.go:24-66` (`testDatabase`, `testTenant`, `testLogger`).

Read-only reference: `services/atlas-character/atlas.com/character/character/entity.go:40` (`GachaponExperience uint32` column) and `administrator.go:101-102` (the dynamic-update reflection case, already present — no change needed).

**Interfaces:**
- Consumes: the `CREDIT_STORED_EXPERIENCE` wire command produced by Task 7 — `Type == "CREDIT_STORED_EXPERIENCE"`, body `{channelId, amount uint32, reason string}`.
- Produces: `character.SetGachaponExperience(v uint32) EntityUpdateFunction`; `Processor.CreditStoredExperienceAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint32, reason string) error` and `Processor.CreditStoredExperience(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint32, reason string) error`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-character/atlas.com/character/character/processor_stored_experience_test.go`, package `character`, reusing `testDatabase`/`testTenant`/`testLogger` from `processor_test.go`.

`TestCreditStoredExperience` — table-driven. Each case seeds a character with the given starting `GachaponExperience` via the existing builder+create path used in `processor_test.go`, runs `CreditStoredExperienceAndEmit` with a fresh `uuid.New()` transaction id and `channel.NewModel(1, 2)`, then re-reads the row and inspects the outbox messages the emit produced.

| case | starting balance | amount | expect column after | expect emitted |
|---|---|---|---|---|
| `credits from zero` | `0` | `3000` | `3000` | one `STAT_CHANGED` with `[]stat.Type{stat.TypeGachaponExperience}` and values `{"gachapon_experience": uint32(3000)}` |
| `accumulates` | `1000` | `500` | `1500` | `STAT_CHANGED` with `{"gachapon_experience": uint32(1500)}` |
| `clamps at MaxUint32 instead of wrapping` | `4294967290` | `100` | `4294967295` | `STAT_CHANGED` with `{"gachapon_experience": uint32(4294967295)}` |
| `exactly saturates` | `4294967295` | `1` | `4294967295` | `STAT_CHANGED` with `{"gachapon_experience": uint32(4294967295)}` |
| `zero amount is a total no-op` | `1000` | `0` | `1000` | no messages at all |

The clamp case is the FR-17 assertion: `4294967290 + 100` wraps to `94` on a naive `uint32` add, so the test fails loudly on the wrong implementation rather than silently.

`TestCreditStoredExperienceUnknownCharacter` — call `CreditStoredExperienceAndEmit` for a character id that was never created; assert it returns a non-nil error and emits nothing.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run 'CreditStoredExperience' -v`
Expected: FAIL — `undefined: CreditStoredExperienceAndEmit`.

- [ ] **Step 3: Add the entity update function**

In `administrator.go`, next to `SetExperience`:

```go
// SetGachaponExperience writes the stored-EXP counter. The column name is the
// community misnomer for the client's GW_CharacterStat::nTempEXP; it is kept
// so no migration is needed across atlas-login/-cashshop/-npc-shops.
func SetGachaponExperience(amount uint32) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"GachaponExperience"}, func(e *entity) {
			e.GachaponExperience = amount
		}
	}
}
```

- [ ] **Step 4: Add the command const and body**

In `kafka/message/character/kafka.go`, add to the const block:

```go
	CommandCreditStoredExperience = "CREDIT_STORED_EXPERIENCE"
```

and the body struct next to `DeductExperienceCommandBody`:

```go
// CreditStoredExperienceCommandBody banks EXP into the stored-EXP counter
// (GW_CharacterStat::nTempEXP, persisted as `gachapon_experience`). It does NOT
// award character EXP — REDEEM_STORED_EXPERIENCE does that.
type CreditStoredExperienceCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    uint32     `json:"amount"`
	Reason    string     `json:"reason"`
}
```

- [ ] **Step 5: Add the processor pair**

In `character/processor.go`, add to the `Processor` interface next to the `ChangeHP` pair (lines 135-136):

```go
	CreditStoredExperienceAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint32, reason string) error
	CreditStoredExperience(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint32, reason string) error
```

and the implementations:

```go
func (p *ProcessorImpl) CreditStoredExperienceAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint32, reason string) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).CreditStoredExperience(buf)(transactionId, channel, characterId, amount, reason)
		})
	})
}

func (p *ProcessorImpl) CreditStoredExperience(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint32, reason string) error {
	return func(transactionId uuid.UUID, channel channel.Model, characterId uint32, amount uint32, reason string) error {
		var adjusted uint32
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			current := c.GachaponExperience()
			// Saturating add. The counter is a uint32 column; a naive add
			// wraps a near-full counter back to near-zero and silently eats
			// the player's banked EXP (FR-17).
			if amount > math.MaxUint32-current {
				adjusted = math.MaxUint32
			} else {
				adjusted = current + amount
			}
			if adjusted == current {
				return nil
			}
			p.l.Debugf("Crediting character [%d] stored experience by [%d] to [%d], reason [%s].", characterId, amount, adjusted, reason)
			return dynamicUpdate(tx)(SetGachaponExperience(adjusted))(c)
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not credit character [%d] stored experience by [%d].", characterId, amount)
			return txErr
		}
		if amount == 0 {
			return nil
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeGachaponExperience}, map[string]interface{}{"gachapon_experience": adjusted}))
		return nil
	}
}
```

`math` is already imported in this file (used by `resolveEffectiveMax`); confirm before adding an import.

- [ ] **Step 6: Register and handle the command**

In `kafka/consumer/character/consumer.go`, add a registration alongside the others:

```go
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreditStoredExperience(db)))); err != nil {
				return err
			}
```

and the handler next to `handleChangeHP`:

```go
func handleCreditStoredExperience(db *gorm.DB) message.Handler[character2.Command[character2.CreditStoredExperienceCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.CreditStoredExperienceCommandBody]) {
		if c.Type != character2.CommandCreditStoredExperience {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).CreditStoredExperienceAndEmit(c.TransactionId, cha, c.CharacterId, c.Body.Amount, c.Body.Reason)
	}
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd services/atlas-character/atlas.com/character && go build ./... && go test ./character/... ./kafka/... -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-character/atlas.com/character/
git commit -m "feat(character): add CREDIT_STORED_EXPERIENCE command with saturating credit"
```

---

## Task 9: `atlas-character` — `REDEEM_STORED_EXPERIENCE`

### Files

- `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go` — command const and body struct
- `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go` — register and handle
- `services/atlas-character/atlas.com/character/character/processor.go` — `Processor` interface entries plus the `...AndEmit` / `(mb *message.Buffer)` pair
- `services/atlas-character/atlas.com/character/character/processor_stored_experience_test.go` — extend the file created in Task 8

Module root: `services/atlas-character/atlas.com/character`.

Patterns to copy: the same command chain as Task 8; `processor.go:736-742` (`AwardExperienceAndEmit`) and `processor.go:744-795` (`AwardExperience`) — read both before writing.

Read-only reference: `libs/atlas-database/transaction.go:9-26`.

**Interfaces:**
- Consumes: the `REDEEM_STORED_EXPERIENCE` wire command produced by Task 11 — `Type == "REDEEM_STORED_EXPERIENCE"`, body `{channelId}`. Also consumes `SetGachaponExperience` from Task 8.
- Produces: `Processor.RedeemStoredExperienceAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32) error` and its `(mb *message.Buffer)` partner.

**Transaction nesting — resolved, do not re-litigate.** `database.ExecuteTransaction` (`libs/atlas-database/transaction.go:9-14`) short-circuits to `fn(db)` when `isTransaction(db)` is true, and `isTransaction` (`:20-26`) tests whether `db.Statement.ConnPool` is a `gorm.TxCommitter`. So `p.WithTransaction(tx).AwardExperience(buf)(...)` genuinely JOINS the outer transaction; its internal `ExecuteTransaction` does not open or commit a second one. Design Risk #1 is closed — build on `WithTransaction` as designed, no hoisting of the EXP arithmetic.

**Two `STAT_CHANGED` events, deliberately.** `AwardExperience` already emits `EXPERIENCE_CHANGED` and a `STAT_CHANGED{EXPERIENCE}`. Redeem adds its own `STAT_CHANGED{GACHAPON_EXPERIENCE}` after it. Both reach the client and the channel snapshot registry applies each; FR-12's requirement is that the gachapon stat is in the emitted set, which it is. Do not try to merge them — `AwardExperience` is reused verbatim so FR-14 (shared level-cap and overflow behaviour, including its `AWARD_LEVEL` command emission) falls out for free.

- [ ] **Step 1: Write the failing test**

Extend `processor_stored_experience_test.go`.

`TestRedeemStoredExperience` — table-driven; each case seeds a character with the given balance, level and starting experience, calls `RedeemStoredExperienceAndEmit(uuid.New(), channel.NewModel(1, 2), characterId)`, then re-reads the row and inspects the emitted messages.

| case | balance | level | expect experience delta | expect balance after | expect emitted |
|---|---|---|---|---|---|
| `redeems the whole balance` | `5000` | `30` | `+5000` | `0` | `EXPERIENCE_CHANGED`, `STAT_CHANGED{EXPERIENCE}`, `STAT_CHANGED{GACHAPON_EXPERIENCE: uint32(0)}` |
| `at the level bound` | `5000` | `50` | `+5000` | `0` | same as above |
| `above the level bound is a no-op` | `5000` | `51` | `0` | `5000` | no messages |
| `zero balance is a no-op` | `0` | `30` | `0` | `0` | no messages |

The two no-op cases assert FR-11 literally: no write, no event, and (necessarily) no error returned — the caller must not treat them as failures.

`TestRedeemStoredExperienceIsExactlyOnce` — seed balance `5000`, level `30`, then call `RedeemStoredExperienceAndEmit` twice with different transaction ids. Assert the character's experience rose by exactly `5000` in total (not `10000`), the balance is `0`, and the second call emitted no messages. This is FR-13 / the duplicate-replay assertion: the second call reads `0` inside its own transaction and short-circuits.

`TestRedeemStoredExperienceDistributionIsWhite` — seed balance `5000`, level `30`, redeem, then unmarshal the `EXPERIENCE_CHANGED` message and assert it carries a `character2.ExperienceDistributionTypeWhite` (`"WHITE"`) distribution with `Amount == 5000` (plus the `showEffect` White+Chat display pair `AwardExperience` appends), no `ExperienceDistributionTypeItem` distribution, and that persisted experience rose by exactly `5000`, not `10000`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run 'RedeemStoredExperience' -v`
Expected: FAIL — `undefined: RedeemStoredExperienceAndEmit`.

- [ ] **Step 3: Add the command const and body**

In `kafka/message/character/kafka.go`:

```go
	CommandRedeemStoredExperience = "REDEEM_STORED_EXPERIENCE"
```

```go
// RedeemStoredExperienceCommandBody redeems the WHOLE stored-EXP counter
// (`gachapon_experience`, the client's GW_CharacterStat::nTempEXP) into real
// character EXP and zeroes it. There is no amount: the client's
// CUIStatusBar::TryUseTempExp charges the entire balance or nothing.
type RedeemStoredExperienceCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
}
```

- [ ] **Step 4: Add the processor pair**

In `character/processor.go`, add to the `Processor` interface:

```go
	RedeemStoredExperienceAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32) error
	RedeemStoredExperience(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32) error
```

and the implementations:

```go
// storedExperienceMaxLevel is the inclusive level bound the client applies to a
// stored-EXP charge. CUIStatusBar::TryUseTempExp gates on characterLevel <= 0x32
// and CWvsContext::SendTempExpUseRequest re-checks it, emitting StringPool
// 0xC97 ("Your level is too high to use the selected item.") above it.
const storedExperienceMaxLevel = byte(50)

func (p *ProcessorImpl) RedeemStoredExperienceAndEmit(transactionId uuid.UUID, channel channel.Model, characterId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).RedeemStoredExperience(buf)(transactionId, channel, characterId)
		})
	})
}

func (p *ProcessorImpl) RedeemStoredExperience(mb *message.Buffer) func(transactionId uuid.UUID, channel channel.Model, characterId uint32) error {
	return func(transactionId uuid.UUID, channel channel.Model, characterId uint32) error {
		var redeemed uint32
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			ip := p.WithTransaction(tx)
			c, err := ip.GetById()(characterId)
			if err != nil {
				return err
			}
			if c.GachaponExperience() == 0 {
				// FR-11: a zero balance is a total no-op — no write, no event,
				// no error, no disconnect. This is also what makes a replayed
				// redeem safe: the second one reads 0 inside its own
				// transaction and stops here (FR-13).
				return nil
			}
			if c.Level() > storedExperienceMaxLevel {
				p.l.Debugf("Character [%d] is level [%d]; refusing to redeem stored experience above level [%d].", characterId, c.Level(), storedExperienceMaxLevel)
				return nil
			}
			redeemed = c.GachaponExperience()

			// Zero the counter and award the EXP on the SAME tx. Ordering is
			// zero-then-award so a failure in the award rolls the zero back
			// with it. database.ExecuteTransaction short-circuits to fn(tx)
			// when already inside a transaction, so AwardExperience's internal
			// ExecuteTransaction joins this one rather than committing
			// independently (libs/atlas-database/transaction.go:9-26).
			if err := dynamicUpdate(tx)(SetGachaponExperience(0))(c); err != nil {
				return err
			}
			experience := []ExperienceModel{NewExperienceModel(character2.ExperienceDistributionTypeWhite, redeemed, 0)}
			return ip.AwardExperience(mb)(transactionId, characterId, channel, experience, true)
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not redeem stored experience for character [%d].", characterId)
			return txErr
		}
		if redeemed == 0 {
			return nil
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeGachaponExperience}, map[string]interface{}{"gachapon_experience": uint32(0)}))
		return nil
	}
}
```

- [ ] **Step 5: Register and handle the command**

In `kafka/consumer/character/consumer.go`:

```go
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRedeemStoredExperience(db)))); err != nil {
				return err
			}
```

```go
func handleRedeemStoredExperience(db *gorm.DB) message.Handler[character2.Command[character2.RedeemStoredExperienceCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.RedeemStoredExperienceCommandBody]) {
		if c.Type != character2.CommandRedeemStoredExperience {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		_ = character.NewProcessor(l, ctx, db).RedeemStoredExperienceAndEmit(c.TransactionId, cha, c.CharacterId)
	}
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-character/atlas.com/character && go build ./... && go test ./character/... ./kafka/... -v`
Expected: PASS, including all four `TestRedeemStoredExperience` cases and the exactly-once test.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-character/atlas.com/character/
git commit -m "feat(character): add REDEEM_STORED_EXPERIENCE single-transaction redeem"
```

---

## Task 10: `atlas-consumables` — the `ConsumeSolomon` consumer

### Files

- `services/atlas-consumables/atlas.com/consumables/consumable/solomon.go` — **new file**
- `services/atlas-consumables/atlas.com/consumables/consumable/solomon_test.go` — **new file**
- `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` — add the routing branch in `RequestItemConsume` (lines 310-336)

Module root: `services/atlas-consumables/atlas.com/consumables`.

Patterns to copy: `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon.go:71-167` (the whole `routesToX` + `xDeps` + pure-body + exported-`ItemConsumer` structure, copy it wholesale and substitute) and `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon_test.go:257-292` (`newMorphCouponHarness` — the mock-wiring block to copy for the Solomon harness).

Read-only reference: `consumable/processor.go:418-433` (`ConsumeError`) and `services/atlas-consumables/atlas.com/consumables/character/model.go:41,209-211` (`gachaponExperience` field and `GachaponExperience()` accessor — **already present**, along with its REST decode at `character/rest.go:19,123,157`; consume the existing accessor, add nothing).

**Interfaces:**
- Consumes: `item2.ClassificationConsumableExpUpItem` and `character.Processor.CreditStoredExperience` (Task 7); `consumable.Model.MaxLevel()` and `consumable.SpecTypeExperience` (Task 6).
- Produces: `ConsumeSolomon(transactionId uuid.UUID, f field.Model, characterId uint32, slot int16, itemId item2.Id) ItemConsumer` and `routesToSolomon(itemId item2.Id) bool`.

**Rejection semantics — read this before writing the error path.** `atlas-consumables` cannot call `session.EnableActions`: that helper lives in `atlas-channel`'s `session` package and needs a `session.Model` and a `writer.Producer`. What `ConsumeError` does is cancel the reservation and emit an `ERROR` event on `EVENT_TOPIC_CONSUMABLE_STATUS`; `atlas-channel`'s `handleErrorConsumableEvent` fallback arm (`services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go:141-148`) turns any unrecognized error type into exactly the empty-`STAT_CHANGED` unstick. So routing every Solomon rejection through `d.onError` gives the design's "cancel + EnableActions + log" for free, with **no new channel-side wiring**. Do not add a new `ErrorType` const — a new recognized type would fall out of that fallback arm and wedge the client.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-consumables/atlas.com/consumables/consumable/solomon_test.go`.

Build `newSolomonHarness` by copying `morph_coupon_test.go:257-292` and substituting the collaborator set: the consumable-data processor mock (for `GetById`), the map-character processor mock (for `GetMap`), the compartment mock (recording `ConsumeItem` calls), and the character-processor mock recording `GetById` and `CreditStoredExperience` calls. `onError` captures into `h.errors`.

`TestConsumeSolomon` — table-driven over `consumeSolomon`:

| case | `spec/exp` | `maxLevel` | char level | char stored balance | expect `ConsumeItem` | expect `CreditStoredExperience` | expect `onError` |
|---|---|---|---|---|---|---|---|
| `eligible` | `3000` | `200` | `30` | `0` | once, `inventory2.TypeValueUse`, the reserved slot | once, amount `3000` | never |
| `maxLevel absent means no upper bound` | `3000` | `0` | `200` | `0` | once | once, amount `3000` | never |
| `level above maxLevel` | `3000` | `20` | `30` | `0` | never | never | once |
| `level exactly at maxLevel` | `3000` | `30` | `30` | `0` | once | once, amount `3000` | never |
| `balance already non-zero` | `3000` | `200` | `30` | `1200` | never | never | once |
| `spec/exp absent` | `0` | `200` | `30` | `0` | never | never | once |
| `spec/exp negative` | `-5` | `200` | `30` | `0` | never | never | once |

Every "never `ConsumeItem`" row is the FR-6 assertion in concrete form: the Writ survives a rejection. The `spec/exp absent` row is specifically the un-reingested-tenant case — the item must bounce, never be destroyed for zero EXP.

`TestConsumeSolomonDataReadFailure` — make the consumable-data `GetById` mock return an error; assert `onError` fired once and `ConsumeItem` was never called.

`TestRoutesToSolomon` — table: `2370000` → true, `2370012` → true, `2379999` → true, `2369999` → false, `2380000` → false, `2000000` → false.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run 'Solomon' -v`
Expected: FAIL — `undefined: consumeSolomon`, `undefined: routesToSolomon`.

- [ ] **Step 3: Write the consumer**

Create `services/atlas-consumables/atlas.com/consumables/consumable/solomon.go`, mirroring `morph_coupon.go`'s structure:

- `routesToSolomon(itemId item2.Id) bool` → `item2.GetClassification(itemId) == item2.ClassificationConsumableExpUpItem`, with a doc comment recording the `is_exp_up_item` derivation.
- `solomonDeps` struct: `data consumable3.Processor`, `fields character2.Processor`, `compartment compartment.Processor`, `character character.Processor`, `onError func(err error) error`.
- `consumeSolomon(l, ctx, d, transactionId, characterId, slot, itemId) error`:
  1. Parallel `model.NewGroup` / `model.Submit` for the field, the consumable data, and the character — **every fallible read before the commit**, exactly as `consumeMorphCoupon` does, so a data failure returns the Writ. `pg.Wait()` error → `d.onError(err)`.
  2. `amount, ok := ci.GetSpec(consumable3.SpecTypeExperience)`; reject via `d.onError` when `!ok || amount <= 0`, logging at Warn and naming the character id and item id, with the message stating the tenant's Item.wz likely predates the `spec/exp` parse.
  3. Reject via `d.onError` when `ci.MaxLevel() > 0 && uint32(c.Level()) > ci.MaxLevel()`, logging the rule name, the character's level and the bound.
  4. Reject via `d.onError` when `c.GachaponExperience() != 0`, logging the rule name and the current balance — this mirrors the client's own interlock (StringPool `0x130E`: the player must spend the banked balance before charging another Writ).
  5. `d.compartment.ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, slot)`; error → `d.onError(err)`.
  6. `d.character.CreditStoredExperience(f, characterId, uint32(amount), "SOLOMON_ITEM")`; error is logged at Error and NOT rolled back, matching the post-commit convention in `consumeMorphCoupon:121-133`.
- `ConsumeSolomon(transactionId uuid.UUID, f field.Model, characterId uint32, slot int16, itemId item2.Id) ItemConsumer` binding the real processors, with `onError` bound to `p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)`.

The `field.Model` parameter is threaded in from `RequestItemConsume` rather than re-fetched only if the caller already holds one; if it does not, resolve it through `d.fields.GetMap(characterId)` inside the pre-commit group as `consumeMorphCoupon` does, and drop the parameter. Pick whichever matches the call site in Step 4 and keep the signature in `Interfaces` above consistent with what you build.

- [ ] **Step 4: Add the routing branch**

In `consumable/processor.go`'s `RequestItemConsume`, add a branch **before** the reward-table fallback, for the same reason the morph-coupon branch is there:

```go
	} else if routesToSolomon(itemId) {
		// Writ of Solomon (classification 237). Must precede the reward-table
		// fallback: the Writ carries no reward table, so it would otherwise
		// fall through to ConsumeBare and be destroyed with nothing banked.
		// Deliberately NOT in usesStandardConsumer — the Writ's spec/exp is not
		// a stat-up and its eligibility rules are its own.
		itemConsumer = ConsumeSolomon(transactionId, characterId, slot, itemId)
```

Note `RequestItemConsume`'s real signature is `(c channel.Model, characterId uint32, slot int16, itemId item2.Id, quantity int16, petId uint64)` — slot precedes itemId here, unlike the channel-side `RequestItemConsume`. Match the local order.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./... && go test ./consumable/... -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/consumable/
git commit -m "feat(consumables): add ConsumeSolomon with maxLevel and balance interlock"
```

---

## Task 11: `atlas-channel` — the two socket handlers

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/character_item_use.go` — add `CharacterItemUseSolomonHandleFunc`
- `services/atlas-channel/atlas.com/channel/socket/handler/character_stored_experience_use.go` — **new file**
- `services/atlas-channel/atlas.com/channel/socket/handler/character_stored_experience_use_test.go` — **new file**
- `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go` — add the redeem command const and body (const block lines 13-30)
- `services/atlas-channel/atlas.com/channel/character/producer.go` — add `RedeemStoredExperienceCommandProvider`
- `services/atlas-channel/atlas.com/channel/character/processor.go` — add `RedeemStoredExperience` to the `Processor` interface (lines 30-53) and `ProcessorImpl`
- `services/atlas-channel/atlas.com/channel/main.go` — two `handlerMap` registrations (near lines 1031-1050)

Seven files, one service; the last four are the four halves of one Kafka producer seam. Noted in `context.md`.

Module root: `services/atlas-channel/atlas.com/channel`.

Patterns to copy: `socket/handler/character_item_use.go:45-52` (`CharacterItemUseSummonBagHandleFunc`, verbatim shape); `character/producer.go:77-90` (`ChangeHPCommandProvider`); `character/processor.go:315-317` (`ChangeHP`); `socket/handler/npc_item_use_test.go:1-50` (the nearest sibling handler-test shape — `character_item_use.go` has no test file today).

Read-only reference: `services/atlas-channel/atlas.com/channel/character/snapshot/registry.go:264,307` — `stat.TypeGachaponExperience` is **already** mapped to `"gachapon_experience"` and `applyStat` already handles it. Nothing changes there. Likewise `libs/atlas-packet/stat/clientbound/changed.go:86,137` already encodes the stat, and `GACHAPON_EXPERIENCE` is already in every seed template's stat-index table.

**Interfaces:**
- Consumes: `invsb.CharacterItemUseSolomonHandle` and `invsb.NewItemUse` (Task 1); `charsb.CharacterUseStoredExperienceHandle` and `charsb.StoredExperienceUse` (Task 2); the `REDEEM_STORED_EXPERIENCE` command contract (Task 9).
- Produces: `handler.CharacterItemUseSolomonHandleFunc`, `handler.CharacterUseStoredExperienceHandleFunc`, `character.Processor.RedeemStoredExperience(f field.Model, characterId uint32) error`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_stored_experience_use_test.go`, copying the seam-installer/imports shape from `npc_item_use_test.go:1-50`.

`TestCharacterUseStoredExperienceHandleFunc` — feed the handler a `request.Reader` over the four bytes `{0x0D, 0x0C, 0x0B, 0x0A}` (the fixture tick from Task 2) and a session for character `1234` in world 1 / channel 2, with the `character.Processor` seam replaced by a recorder. Assert:

| assertion | expected |
|---|---|
| recorded calls to `RedeemStoredExperience` | exactly 1 |
| recorded `characterId` | `1234` |
| recorded field world / channel | `1` / `2` |
| bytes consumed from the reader | 4 (the body carries nothing but the tick) |

`TestRedeemStoredExperienceCommandProvider` — in `character/producer_test.go` (extend it, or create it copying an existing provider test in the package): call `RedeemStoredExperienceCommandProvider(f, 1234)` for world 1 / channel 2 and assert the marshalled message has `Type == "REDEEM_STORED_EXPERIENCE"`, `CharacterId == 1234`, `WorldId == 1`, `Body.ChannelId == 2`, and key `producer.CreateKey(1234)`. These values must match Task 9's `RedeemStoredExperienceCommandBody` exactly — they are the same wire message.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'StoredExperience' -v`
Expected: FAIL — `undefined: CharacterUseStoredExperienceHandleFunc`.

- [ ] **Step 3: Add the Solomon handler**

Append to `socket/handler/character_item_use.go` — one line different from the summon-bag handler:

```go
func CharacterItemUseSolomonHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := inventory2.NewItemUse(inventory2.CharacterItemUseSolomonHandle)
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = consumable.NewProcessor(l, ctx).RequestItemConsume(s.Field(), character.Id(s.CharacterId()), item.Id(p.ItemId()), slot.Position(p.Source()), 1, p.UpdateTime())
	}
}
```

It decodes the shared `ItemUse`, not `SolomonItemUse` — the wrapper from Task 1 exists solely to give the op its own packet id and evidence key.

- [ ] **Step 4: Add the redeem command seam**

In `kafka/message/character/kafka.go`, add `CommandRedeemStoredExperience = "REDEEM_STORED_EXPERIENCE"` to the const block and:

```go
type RedeemStoredExperienceCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
}
```

In `character/producer.go`, add `RedeemStoredExperienceCommandProvider(f field.Model, characterId uint32) model.Provider[[]kafka.Message]` following `ChangeHPCommandProvider`'s shape exactly (fresh `uuid.New()` transaction id, key from `producer.CreateKey(int(characterId))`).

In `character/processor.go`, add `RedeemStoredExperience(f field.Model, characterId uint32) error` to the `Processor` interface and:

```go
func (p *ProcessorImpl) RedeemStoredExperience(f field.Model, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(RedeemStoredExperienceCommandProvider(f, characterId))
}
```

- [ ] **Step 5: Add the redeem handler**

Create `socket/handler/character_stored_experience_use.go`:

```go
package handler

import (
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// CharacterUseStoredExperienceHandleFunc handles USE_GACHA_EXP: the player
// clicked the EXP bar and confirmed charging the EXP banked by their Writs of
// Solomon. The request carries nothing but the tick — the client always
// redeems the whole balance — so every rule (zero balance, level > 50) is
// evaluated server-side in atlas-character.
func CharacterUseStoredExperienceHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := charsb.StoredExperienceUse{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = character.NewProcessor(l, ctx).RedeemStoredExperience(s.Field(), s.CharacterId())
	}
}
```

Match the exact import alias and `character.Id(...)` conversions the neighbouring handlers use; adjust if `RedeemStoredExperience` takes a typed character id.

- [ ] **Step 6: Register both handlers**

In `main.go`, next to the existing item-use registrations:

```go
	handlerMap[invsb.CharacterItemUseSolomonHandle] = handler.CharacterItemUseSolomonHandleFunc
```
```go
	handlerMap[charsb.CharacterUseStoredExperienceHandle] = handler.CharacterUseStoredExperienceHandleFunc
```

Use whatever alias `main.go` already imports `libs/atlas-packet/character/serverbound` under (it is `charsb` at line 992).

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/... ./character/... -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/
git commit -m "feat(channel): handle USE_SOLOMON_ITEM and USE_GACHA_EXP"
```

---

## Task 12: Seed templates — route both handlers on the eight in-scope versions

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_72_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_79_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`

Read-only references (must NOT be edited — these three versions do not carry the ops): `template_gms_12_1.json`, `template_gms_48_1.json`, `template_gms_61_1.json`.

Eight files, one identical two-entry insertion each. Deliberately one task: it is the same mechanical edit repeated, and splitting it would fragment a single grep-verified acceptance criterion. Noted in `context.md`.

Patterns to copy: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json:682-690` (a serverbound handler entry).

**Interfaces:**
- Consumes: the handler-name constants from Tasks 1-2 (`CharacterItemUseSolomonHandle`, `CharacterUseStoredExperienceHandle`) and the per-version opcodes in Global Constraints.
- Produces: routing, without which both handlers are dead code.

- [ ] **Step 1: Write the failing check**

Run: `for f in 72 79 83 84 87 92 95; do printf '%s: ' "$f"; grep -c 'CharacterItemUseSolomonHandle\|CharacterUseStoredExperienceHandle' services/atlas-configurations/seed-data/templates/template_gms_${f}_1.json; done; printf 'jms185: '; grep -c 'CharacterItemUseSolomonHandle\|CharacterUseStoredExperienceHandle' services/atlas-configurations/seed-data/templates/template_jms_185_1.json`
Expected: `0` for all eight.

- [ ] **Step 2: Add both entries to each of the eight templates**

Insert into each template's serverbound handler array, in opcode order. For `template_gms_83_1.json` the two entries are:

```json
      {
        "opCode": "0x9D",
        "validator": "LoggedInValidator",
        "handler": "CharacterItemUseSolomonHandle",
        "fname": "CWvsContext::SendExpUpItemUseRequest",
        "services": [
          "channel"
        ]
      },
      {
        "opCode": "0x9E",
        "validator": "LoggedInValidator",
        "handler": "CharacterUseStoredExperienceHandle",
        "fname": "CWvsContext::SendTempExpUseRequest",
        "services": [
          "channel"
        ]
      },
```

The other seven are identical apart from `opCode`, taken from the Global Constraints table:

| template | Solomon `opCode` | redeem `opCode` |
|---|---|---|
| `template_gms_72_1.json` | `"0x9C"` | `"0x9D"` |
| `template_gms_79_1.json` | `"0x9B"` | `"0x9C"` |
| `template_gms_83_1.json` | `"0x9D"` | `"0x9E"` |
| `template_gms_84_1.json` | `"0xA1"` | `"0xA2"` |
| `template_gms_87_1.json` | `"0xA5"` | `"0xA6"` |
| `template_gms_92_1.json` | `"0xB2"` | `"0xB3"` |
| `template_gms_95_1.json` | `"0xB5"` | `"0xB6"` |
| `template_jms_185_1.json` | `"0x71"` | `"0x72"` |

Preserve each file's existing indentation and line endings exactly.

- [ ] **Step 3: Verify presence on the eight and absence on the three**

Run: `for f in 72 79 83 84 87 92 95; do printf '%s: ' "$f"; grep -c 'CharacterItemUseSolomonHandle\|CharacterUseStoredExperienceHandle' services/atlas-configurations/seed-data/templates/template_gms_${f}_1.json; done; printf 'jms185: '; grep -c 'CharacterItemUseSolomonHandle\|CharacterUseStoredExperienceHandle' services/atlas-configurations/seed-data/templates/template_jms_185_1.json`
Expected: `2` for all eight.

Run: `grep -l 'CharacterItemUseSolomonHandle\|CharacterUseStoredExperienceHandle' services/atlas-configurations/seed-data/templates/template_gms_12_1.json services/atlas-configurations/seed-data/templates/template_gms_48_1.json services/atlas-configurations/seed-data/templates/template_gms_61_1.json`
Expected: no output (exit 1) — the three out-of-scope templates are untouched.

- [ ] **Step 4: Verify every template is still valid JSON with no duplicate opcode**

Run: `for f in services/atlas-configurations/seed-data/templates/*.json; do python3 -c "import json,sys; json.load(open(sys.argv[1]))" "$f" || echo "INVALID: $f"; done`
Expected: no output.

Run: `go run ./tools/packet-audit operations --check`
Expected: exit 0. This is the check that catches an opcode colliding with an existing serverbound entry on any column.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(configurations): route stored-EXP handlers on the eight in-scope templates"
```

---

## Task 13: Documentation and the re-ingest follow-up

### Files

- `docs/research/missing-features/items-and-consumables.md` — update the "Wholly missing" §3 entry for stored-EXP items to reflect the implemented state
- `docs/TODO.md` — record the tenant re-ingest follow-up

**Interfaces:**
- Consumes: the implemented state from Tasks 1-12.
- Produces: nothing code depends on.

- [ ] **Step 1: Update the research doc**

Read `docs/research/missing-features/items-and-consumables.md` and locate the §3 "Wholly missing" entry covering the Writ of Solomon / stored ("gachapon") EXP. Rewrite it to state what now exists: both ops implemented and verified on the eight in-scope columns; classification 237 credits `gachapon_experience` via `CREDIT_STORED_EXPERIENCE`; `USE_GACHA_EXP` redeems it via `REDEEM_STORED_EXPERIENCE`; `gms_v48`/`gms_v61` carry positive-absence evidence. Correct the §5 line that describes the `2370000`–`2370012` `exp` spec as "EXP potions … no effect" — that field is now the amount a Writ banks.

Do not restate the whole design here; link to `docs/tasks/task-277-stored-exp-items/design.md`.

- [ ] **Step 2: Add the re-ingest follow-up**

Add an entry to `docs/TODO.md` in whatever section the file uses for operational follow-ups (read it first; do not invent a section). Content:

> **Tenant Item.wz re-ingest for `spec/exp` and `info/maxLevel` (task-277).** `atlas-data` now parses the consumable `info/maxLevel` field and the `spec/exp` spec. Tenants whose `Item.wz` was ingested before task-277 will serve neither, so every Writ of Solomon use is rejected (the item is returned, never destroyed) until re-ingest. Same class as task-219's morph coupon. Owner: operations.

Use repo-relative paths only — no literal home or absolute paths.

- [ ] **Step 3: Verify no absolute paths leaked in**

Run: `grep -n '/home/\|/Users/' docs/research/missing-features/items-and-consumables.md docs/TODO.md`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add docs/research/missing-features/items-and-consumables.md docs/TODO.md
git commit -m "docs: record stored-EXP implementation and the tenant re-ingest follow-up"
```

---

## Task 14: Full verification gate

### Files

- No files are edited by this task unless the gate fails.

Read-only reference: `tools/verify.sh` and `docs/verification.md`.

**Interfaces:**
- Consumes: everything from Tasks 1-13.
- Produces: the green gate that lets the branch go to code review.

- [ ] **Step 1: Run the flagless verification gate**

Run: `tools/verify.sh`
Expected: exit 0.

`--quick` and `--no-docker` skip the bake and `-race`; they do NOT count as the gate. If the gate fails, fix the cause in the owning task's files and re-run — do not narrow the invocation.

- [ ] **Step 2: Re-run every packet-audit gate**

Run: `go run ./tools/packet-audit matrix --check && go run ./tools/packet-audit fname-doc --check && go run ./tools/packet-audit operations --check && go run ./tools/packet-audit dispatcher-lint && go run ./tools/packet-audit gate-check --check && go run ./tools/packet-audit doc-freshness --check`
Expected: all exit 0.

- [ ] **Step 3: Confirm the tree is clean**

Run: `git status --porcelain`
Expected: no output.

- [ ] **Step 4: Report**

Report PASS with the gate's own exit status quoted, or the first failing block verbatim. Do not claim verified from a flagged or partial run. Code review runs after this gate and before any PR.
