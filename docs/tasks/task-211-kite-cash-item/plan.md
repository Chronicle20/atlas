# Kites (Cash Item Category 508) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make cash item category 508 (kites / message boxes) work end to end — a player uses one, a banner with their message hangs at their position for everyone on the map, late joiners see it, and it comes down when the owner leaves.

**Architecture:** A new Redis-only `atlas-kites` service owns authoritative kite state and enforces placement policy; `atlas-channel` decodes the type-18 sub-body, issues a Kafka command, renders the three already-verified clientbound writers, and replays kites on map entry. Two byte-neutral packet renames correct names the IDB disproved. Placement knobs come from a new `kite-configs` tenant configuration resource.

**Tech Stack:** Go 1.25.5, Redis (`libs/atlas-redis`), Kafka (`libs/atlas-kafka`), JSON:API via api2go, `libs/atlas-packet`, kustomize/k8s, `tools/packet-audit`.

## Global Constraints

- **Version scope is TEN, not eleven.** gms 48/61/72/79/83/84/87/92/95 + jms 185. **gms v12 is excluded** and gets no binding and no `n-a` matrix entry (design Q5). This amends PRD FR-9.1/§10.
- **No encoded byte may change on any version.** Both packet renames are semantic only (design ADR-7b, ADR-7c).
- **No new `FieldLimit` bit.** The client has no such gate; the rule is a map-id prefix denylist, default `[91]` (Free Market) (design Q1).
- **Kite items are NOT consumed** on use — no `saga.DestroyAsset`, no inventory mutation (PRD FR-4.1).
- **Message bound is 182 bytes**, enforced in `atlas-kites`, never in the socket decoder (design Q4).
- **Destroy animation byte is `0`** for both reasons — it is a suppress-animation flag, not a selector (design Q2).
- **Every keyed Redis access goes through `libs/atlas-redis`** (`tools/redis-key-guard.sh`).
- **Every goroutine goes through `routine.Go`** (`tools/goroutine-guard.sh`).
- **No client wire value hard-coded** (DOM-25). Opcodes resolve via tenant config.
- **Before defining any new type/constant, check `libs/atlas-constants/`** (DOM-21). `item.ClassificationMessageBanner = Classification(508)` already exists at `libs/atlas-constants/item/constants.go:78`.
- **Immutable models everywhere**: private fields + getters + Builder. Processors: `Interface` + `Impl`, `NewProcessor(l, ctx)`, pure `Method(mb *message.Buffer)` plus `MethodAndEmit()` wrappers.
- **No `// TODO`, no stubs, no 501s** in any landed commit.
- **Test setup uses the project Builder pattern.** No `*_testhelpers.go` files.
- All work happens in the worktree `.worktrees/task-211-kite-cash-item` on branch `task-211-kite-cash-item`.

Read [`context.md`](context.md) before starting. It lists the reference implementations to copy, the seven silent-failure traps, and the decisions that must not be re-litigated.

---

## Task 1: Rename `FieldKiteSpawn.kiteType` → `y`

The sixth field is a Y coordinate, not a type discriminator. Both int16s feed one `IWzVector2D::RelMove`; appearance comes from `templateId` via `CItemInfo::GetItemProp`. Confirmed on GMS v95 (`0x6369c0`) and GMS v83 (`0x65acdf`). This is a pure identifier rename — **zero encoded bytes change**.

**Files:**
- Modify: `libs/atlas-packet/field/clientbound/kite_spawn.go:16-55`
- Modify: `libs/atlas-packet/field/clientbound/kite_v48_test.go:19,31`
- Modify: `docs/packets/audits/gms_v83/FieldKiteSpawn.json` (row 5 `IDAComment`)
- Modify: `docs/packets/audits/gms_v87/FieldKiteSpawn.json` (row 5 `IDAComment`)
- Modify: `docs/packets/audits/gms_v95/FieldKiteSpawn.json` (row 5 `IDAComment`)
- Modify: `docs/packets/audits/jms_v185/FieldKiteSpawn.json` (row 5 `IDAComment`)

**Interfaces:**
- Produces: `clientbound.NewKiteSpawn(id uint32, templateId uint32, message string, name string, x int16, y int16) KiteSpawn` — the 6th parameter is now named `y`. Task 16 calls it.

- [ ] **Step 1: Capture the pre-change byte output as the guard**

Run from the worktree root:

```bash
cd libs/atlas-packet && go test ./field/clientbound/ -run 'TestKiteSpawn' -v 2>&1 | tail -20
```

Expected: PASS for `TestKiteSpawn` and `TestKiteSpawnBytesV48`. These are the tests that must still pass byte-identically after the rename — that is the whole proof of FR-2.3.

- [ ] **Step 2: Rename the field, constructor parameter, and both codec bodies**

Replace the struct/constructor/codec block in `libs/atlas-packet/field/clientbound/kite_spawn.go` (lines 15-55) with:

```go
// packet-audit:fname CMessageBoxPool::OnMessageBoxEnterField
//
// The sixth field is the spawn Y coordinate, NOT a kite-type discriminator.
// CMessageBoxPool::OnMessageBoxEnterField (gms_v95 @0x6369c0, gms_v83
// @0x65acdf) decodes it into MESSAGEBOX+32, then computes
// renderX = (+28) - 3 and renderY = (+32) - 100 and feeds BOTH to a single
// IWzVector2D::RelMove. The -3/-100 are sprite-anchor offsets, not flags.
// The banner's appearance is selected by templateId alone, which is the sole
// argument to CItemInfo::GetItemProp further down the same function. There is
// no kite-type field on the wire.
type KiteSpawn struct {
	id         uint32
	templateId uint32
	message    string
	name       string
	x          int16
	y          int16
}

func NewKiteSpawn(id uint32, templateId uint32, message string, name string, x int16, y int16) KiteSpawn {
	return KiteSpawn{id: id, templateId: templateId, message: message, name: name, x: x, y: y}
}

func (m KiteSpawn) Operation() string { return KiteSpawnWriter }
func (m KiteSpawn) String() string {
	return fmt.Sprintf("id [%d], templateId [%d]", m.id, m.templateId)
}

func (m KiteSpawn) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.id)
		w.WriteInt(m.templateId)
		w.WriteAsciiString(m.message)
		w.WriteAsciiString(m.name)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		return w.Bytes()
	}
}

func (m *KiteSpawn) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.id = r.ReadUint32()
		m.templateId = r.ReadUint32()
		m.message = r.ReadAsciiString()
		m.name = r.ReadAsciiString()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
	}
}
```

- [ ] **Step 3: Fix the two stale comments in the v48 fixture**

In `libs/atlas-packet/field/clientbound/kite_v48_test.go`, change line 19 from

```
// Decode2(x), Decode2(kiteType).
```

to

```
// Decode2(x), Decode2(y).
```

and line 31 from

```go
		0x02, 0x00, // kiteType   — Decode2
```

to

```go
		0x02, 0x00, // y          — Decode2
```

The `NewKiteSpawn(0x01020304, 5000, "hi", "bob", 300, 2)` call is positional and needs no change.

- [ ] **Step 4: Run the packet tests — bytes must be unchanged**

```bash
cd libs/atlas-packet && go test ./field/clientbound/ -run 'TestKiteSpawn' -v
```

Expected: PASS, same as Step 1. If `TestKiteSpawnBytesV48` fails the rename was not byte-neutral — stop and revert.

- [ ] **Step 5: Correct the four stale audit-JSON comments**

These are the four records that mislabel row 5. Edit each file's `Rows[5].IDAComment` string, changing nothing else (no reformatting — these files are compared byte-wise by the audit tooling).

| File | Replace | With |
|---|---|---|
| `docs/packets/audits/gms_v83/FieldKiteSpawn.json` | `"nType/y (spawn y or kite type, +32)"` | `"ptMessageBox.y (spawn y, +32)"` |
| `docs/packets/audits/gms_v87/FieldKiteSpawn.json` | `"nType (kite type, +32 @0x694f30)"` | `"ptMessageBox.y (spawn y, +32 @0x694f30)"` |
| `docs/packets/audits/gms_v95/FieldKiteSpawn.json` | `"nType (kite type)"` | `"ptMessageBox.y (spawn y, +32)"` |
| `docs/packets/audits/jms_v185/FieldKiteSpawn.json` | `"y / kiteType (@line94)"` | `"ptMessageBox.y (spawn y, @line94)"` |

The v48/v61/v72/v79/v84/v92 records have an empty row-5 comment and need no edit.

- [ ] **Step 6: Regenerate the matrix and prove it is a no-op**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check; echo "exit=$?"
git diff --exit-code docs/packets/audits/status.json docs/packets/audits/STATUS.md; echo "diff-exit=$?"
```

Expected: `exit=0` and `diff-exit=0`. A rename changes no decompile hash, so the evidence records under `docs/packets/evidence/*/field.clientbound.FieldKiteSpawn.yaml` need **no** re-pin and the row must stay `✅ 🟡ᶠ 🟡ᶠ 🟡ᶠ ✅ ✅ ✅ 🟡ᶠ ✅ ✅` (`STATUS.md:332`). A non-empty diff means something other than the rename moved — investigate before continuing.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/field/clientbound/kite_spawn.go \
        libs/atlas-packet/field/clientbound/kite_v48_test.go \
        docs/packets/audits/gms_v83/FieldKiteSpawn.json \
        docs/packets/audits/gms_v87/FieldKiteSpawn.json \
        docs/packets/audits/gms_v95/FieldKiteSpawn.json \
        docs/packets/audits/jms_v185/FieldKiteSpawn.json
git commit -m "fix(atlas-packet): FieldKiteSpawn sixth field is y, not kiteType"
```

---

## Task 2: Rename the kite destroy animation constants

The leading byte is a **suppress-animation flag**, not a selector between two animations. `CMessageBoxPool::OnMessageBoxLeaveField` (gms_v95 `0x635d60`) always removes the box, then plays `CAnimationDisplayer::RegisterOneTimeAnimation` **only when the byte is 0**. Byte-neutral rename.

**Files:**
- Modify: `libs/atlas-packet/field/clientbound/kite_destroy.go:15-20`
- Modify: `libs/atlas-packet/field/clientbound/kite_destroy_test.go:15`

**Interfaces:**
- Produces: `clientbound.KiteDestroyAnimated KiteDestroyAnimationType = 0` and `clientbound.KiteDestroySilent KiteDestroyAnimationType = 1`. Task 15 uses `KiteDestroyAnimated`.

- [ ] **Step 1: Replace the constant block**

In `libs/atlas-packet/field/clientbound/kite_destroy.go`, replace lines 15-20:

```go
type KiteDestroyAnimationType byte

const (
	KiteDestroyAnimationType1 KiteDestroyAnimationType = 0
	KiteDestroyAnimationType2 KiteDestroyAnimationType = 1
)
```

with:

```go
// KiteDestroyAnimationType is the leading byte of REMOVE_KITE. It is a
// SUPPRESS-ANIMATION FLAG, not a selector between two animations.
// CMessageBoxPool::OnMessageBoxLeaveField (gms_v95 @0x635d60) always calls
// RemoveMessageBox, then gates the RelMove / canvas swap /
// CAnimationDisplayer::RegisterOneTimeAnimation despawn sequence on the byte
// being zero. Any non-zero value removes the banner instantly with no visual.
type KiteDestroyAnimationType byte

const (
	// KiteDestroyAnimated plays the one-shot despawn animation.
	KiteDestroyAnimated KiteDestroyAnimationType = 0
	// KiteDestroySilent removes the banner with no visual.
	KiteDestroySilent KiteDestroyAnimationType = 1
)
```

- [ ] **Step 2: Update the one reference in the round-trip test**

In `libs/atlas-packet/field/clientbound/kite_destroy_test.go` line 15, change

```go
	input := NewKiteDestroy(1, KiteDestroyAnimationType2)
```

to

```go
	input := NewKiteDestroy(1, KiteDestroySilent)
```

- [ ] **Step 3: Run the tests**

```bash
cd libs/atlas-packet && go test ./field/clientbound/ -run 'TestKiteDestroy' -v
```

Expected: PASS. `TestKiteDestroyBytesV48` (which passes a literal `1`) is untouched and must still produce `01 04 03 02 01`.

- [ ] **Step 4: Confirm no other references exist**

```bash
grep -rn "KiteDestroyAnimationType1\|KiteDestroyAnimationType2" --include='*.go' .
```

Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/field/clientbound/kite_destroy.go \
        libs/atlas-packet/field/clientbound/kite_destroy_test.go
git commit -m "refactor(atlas-packet): name kite destroy byte as animated/silent flag"
```

---

## Task 3: New serverbound codec `ItemUseKite`

The type-18 arm of `CWvsContext::SendConsumeCashItemUseRequest` (gms_v95 `0x9eb3e0`, arm entry `0x9ecfa2`) joins the three `CUIHope` edit lines with `'\n'`, runs `CCurseProcess::ProcessString`, and performs exactly one encode — `COutPacket::EncodeStr(oPacket, s4)` at `0x9ed271`. The sub-body is one length-prefixed ASCII string. GMS ≤ 84 trails `updateTime` after it, exactly like `ItemUseChalkboard`.

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/item_use_kite.go`
- Create: `libs/atlas-packet/cash/serverbound/item_use_kite_test.go`

**Interfaces:**
- Consumes: `serverbound.UpdateTimeFirst(t tenant.Model) bool` (`item_use.go:21`).
- Produces: `serverbound.NewItemUseKite(updateTimeFirst bool) *ItemUseKite`; methods `Message() string`, `UpdateTime() uint32`, `Operation() string`, `String() string`, `Encode`, `Decode`. Task 14 calls `NewItemUseKite` and `Message()`.

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-packet/cash/serverbound/item_use_kite_test.go`.

Note: these fixtures carry **no** `packet-audit:verify` markers. There is no coverage-matrix row for a cash sub-body — its sibling `item_use_chalkboard_test.go` has none either, and adding a marker for an op the registry does not know would break `packet-audit matrix --check`.

```go
package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseKiteUpdateTimeFirstRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseKite{message: "congrats!", updateTimeFirst: true}
			output := *NewItemUseKite(true)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}

func TestItemUseKiteNoUpdateTimeFirstRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseKite{message: "congrats!", updateTime: 99999, updateTimeFirst: false}
			output := *NewItemUseKite(false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}

// TestItemUseKiteBytesTrailingUpdateTime pins the wire shape for every GMS
// build at or below v84 (v48/v61/v72/v79/v83/v84), where
// CWvsContext::SendConsumeCashItemUseRequest appends update_time in the shared
// send tail rather than the header (see ItemUse.UpdateTimeFirst).
func TestItemUseKiteBytesTrailingUpdateTime(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, major := range []uint16{48, 61, 72, 79, 83, 84} {
		in := ItemUseKite{message: "hi", updateTime: 0x01020304, updateTimeFirst: false}
		got := in.Encode(l, pt.CreateContext("GMS", major, 1))(nil)
		want := []byte{
			0x02, 0x00, 'h', 'i', // message    — EncodeStr
			0x04, 0x03, 0x02, 0x01, // updateTime — trailing Encode4
		}
		if !bytes.Equal(got, want) {
			t.Errorf("GMS v%d kite sub-body:\n got % x\nwant % x", major, got, want)
		}
	}
}

// TestItemUseKiteBytesLeadingUpdateTime pins the wire shape for GMS v87+ and
// JMS v185, where update_time is a leading header int32 already consumed by
// the common ItemUse prefix, so the sub-body is the message alone.
func TestItemUseKiteBytesLeadingUpdateTime(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := ItemUseKite{message: "hi", updateTime: 0x01020304, updateTimeFirst: true}
	want := []byte{0x02, 0x00, 'h', 'i'}
	for _, c := range []struct {
		name   string
		region string
		major  uint16
	}{
		{"gms_v87", "GMS", 87},
		{"gms_v92", "GMS", 92},
		{"gms_v95", "GMS", 95},
		{"jms_v185", "JMS", 185},
	} {
		got := in.Encode(l, pt.CreateContext(c.region, c.major, 1))(nil)
		if !bytes.Equal(got, want) {
			t.Errorf("%s kite sub-body:\n got % x\nwant % x", c.name, got, want)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd libs/atlas-packet && go test ./cash/serverbound/ -run 'ItemUseKite' 2>&1 | head -20
```

Expected: FAIL — `undefined: ItemUseKite`, `undefined: NewItemUseKite`.

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/cash/serverbound/item_use_kite.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUseKite is the CashSlotItemType(18) sub-body of USE_CASH_ITEM — the
// message a player pins to a map with a category-508 kite (message box).
//
// Derived from the case-18 arm of CWvsContext::SendConsumeCashItemUseRequest
// (gms_v95 @0x9eb3e0; arm entry @0x9ecfa2). The arm builds a CUIHope dialog,
// reads its three edit controls via CUIHope::GetText @0x9ed0f8, joins them
// with '\n', screens the result through CCurseProcess::ProcessString @0x9ed1b7,
// and then performs its ONLY encode: COutPacket::EncodeStr @0x9ed271. So the
// sub-body is exactly one length-prefixed string.
//
// Placement coordinates are NOT on the wire — the server takes them from the
// character's own position. Nor is a kite type: the banner's appearance comes
// from the item id (see FieldKiteSpawn).
//
// updateTimeFirst mirrors ItemUse.UpdateTimeFirst: GMS <= v84 trails
// update_time after the sub-body, GMS v87+ and JMS lead it in the header.
type ItemUseKite struct {
	message         string
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseKite(updateTimeFirst bool) *ItemUseKite {
	return &ItemUseKite{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseKite) Message() string    { return m.message }
func (m ItemUseKite) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseKite) Operation() string { return "ItemUseKite" }

func (m ItemUseKite) String() string {
	return fmt.Sprintf("message [%s] updateTime [%d]", m.message, m.updateTime)
}

func (m ItemUseKite) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.message)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseKite) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadAsciiString()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd libs/atlas-packet && go test ./cash/serverbound/ -run 'ItemUseKite' -v
```

Expected: PASS on all four tests.

- [ ] **Step 5: Full module gate**

```bash
cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./...
```

Expected: all clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/item_use_kite.go \
        libs/atlas-packet/cash/serverbound/item_use_kite_test.go
git commit -m "feat(atlas-packet): add ItemUseKite serverbound cash sub-body"
```

---

## Task 4: Bind the three kite writers in all ten tenant templates

`grep -i kite` over the template set currently returns **zero hits** — a correct emit is silently dropped for want of a writer opcode binding. Entries go at their **sorted `opCode` position**, each with an `fname`.

**Critical formatting note:** these templates write opcodes **unpadded** — `"0xC5"`, not `"0x0C5"`; `"0x10E"`, not `"0x010E"`. The PRD's §FR-9.1 table shows a zero-padded form; **use the unpadded form below**, which matches each file's existing entries. A padded duplicate is exactly what `tools/template-duplicate-binding-guard.sh` was written to catch (task-194).

All thirty target opcodes were verified free during planning — no existing writer occupies any of them.

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_48_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_61_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_72_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_79_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`
- **Not** modified: `template_gms_12_1.json` (design Q5 — no matrix column, no `CharacterCashItemUseHandle` binding, cannot receive the request)

**Interfaces:**
- Consumes: writer names `SpawnKiteError`, `SpawnKite`, `DestroyKite` — the values of `clientbound.KiteErrorWriter`, `KiteSpawnWriter`, `KiteDestroyWriter` (`kite_error.go:12`, `kite_spawn.go:13`, `kite_destroy.go:13`). They are already listed in `produceWriters()` at `services/atlas-channel/atlas.com/channel/main.go:724-726`.

- [ ] **Step 1: Confirm the starting state is empty**

```bash
grep -ric kite services/atlas-configurations/seed-data/templates/
```

Expected: `0` for every file.

- [ ] **Step 2: Insert three writer entries per template**

For each template, insert the three objects into the `writers` array at the position given below, preserving the file's exact 6-space object indent / 8-space key indent (copy the shape of the neighbouring `DropDestroy` entry). Use per-file `Edit` operations — do **not** round-trip the JSON through a formatter, which would reflow the whole file.

The entry shape (illustrated for gms_v83):

```json
      {
        "opCode": "0x10E",
        "writer": "SpawnKiteError",
        "fname": "CMessageBoxPool::OnCreateFailed",
        "services": [
          "channel"
        ]
      },
      {
        "opCode": "0x10F",
        "writer": "SpawnKite",
        "fname": "CMessageBoxPool::OnMessageBoxEnterField",
        "services": [
          "channel"
        ]
      },
      {
        "opCode": "0x110",
        "writer": "DestroyKite",
        "fname": "CMessageBoxPool::OnMessageBoxLeaveField",
        "services": [
          "channel"
        ]
      },
```

Per-template opcodes and insertion anchors:

| Template | `SpawnKiteError` | `SpawnKite` | `DestroyKite` | Insert after | Insert before |
|---|---|---|---|---|---|
| `template_gms_48_1.json` | `"0xC5"` | `"0xC6"` | `"0xC7"` | `0xC2` DropDestroy | `0xCA` AffectedAreaCreated |
| `template_gms_61_1.json` | `"0xCF"` | `"0xD0"` | `"0xD1"` | `0xCE` DropDestroy | `0xD2` AffectedAreaCreated |
| `template_gms_72_1.json` | `"0xF0"` | `"0xF1"` | `"0xF2"` | `0xEF` DropDestroy | `0xF3` AffectedAreaCreated |
| `template_gms_79_1.json` | `"0xF8"` | `"0xF9"` | `"0xFA"` | `0xF7` DropDestroy | `0xFB` AffectedAreaCreated |
| `template_gms_83_1.json` | `"0x10E"` | `"0x10F"` | `"0x110"` | `0x10D` DropDestroy | `0x111` AffectedAreaCreated |
| `template_gms_87_1.json` | `"0x11F"` | `"0x120"` | `"0x121"` | `0x11E` DropDestroy | `0x122` AffectedAreaCreated |
| `template_gms_92_1.json` | `"0x13D"` | `"0x13E"` | `"0x13F"` | `0x13C` DropDestroy | `0x140` AffectedAreaCreated |
| `template_gms_95_1.json` | `"0x145"` | `"0x146"` | `"0x147"` | `0x144` DropDestroy | `0x148` AffectedAreaCreated |
| `template_jms_185_1.json` | `"0x123"` | `"0x124"` | `"0x125"` | `0x122` DropDestroy | `0x126` AffectedAreaCreated |

**`template_gms_84_1.json` is the exception** — its `DestroyKite` opcode is not adjacent to the other two (the v84 clientbound table is shifted relative to v83 from ≥0x3D). It needs **two** separate insertions:

| Entries | Insert after | Insert before |
|---|---|---|
| `SpawnKiteError` `"0x10E"` + `SpawnKite` `"0x10F"` | `0x10B` NPCAction | `0x110` SpawnHiredMerchant |
| `DestroyKite` `"0x117"` | `0x114` DropDestroy | `0x118` AffectedAreaCreated |

- [ ] **Step 3: Verify all ten templates parse and carry exactly three bindings**

```bash
python3 - <<'EOF'
import json, glob
expect = {
 'template_gms_48_1.json':  {'SpawnKiteError':0xC5,'SpawnKite':0xC6,'DestroyKite':0xC7},
 'template_gms_61_1.json':  {'SpawnKiteError':0xCF,'SpawnKite':0xD0,'DestroyKite':0xD1},
 'template_gms_72_1.json':  {'SpawnKiteError':0xF0,'SpawnKite':0xF1,'DestroyKite':0xF2},
 'template_gms_79_1.json':  {'SpawnKiteError':0xF8,'SpawnKite':0xF9,'DestroyKite':0xFA},
 'template_gms_83_1.json':  {'SpawnKiteError':0x10E,'SpawnKite':0x10F,'DestroyKite':0x110},
 'template_gms_84_1.json':  {'SpawnKiteError':0x10E,'SpawnKite':0x10F,'DestroyKite':0x117},
 'template_gms_87_1.json':  {'SpawnKiteError':0x11F,'SpawnKite':0x120,'DestroyKite':0x121},
 'template_gms_92_1.json':  {'SpawnKiteError':0x13D,'SpawnKite':0x13E,'DestroyKite':0x13F},
 'template_gms_95_1.json':  {'SpawnKiteError':0x145,'SpawnKite':0x146,'DestroyKite':0x147},
 'template_jms_185_1.json': {'SpawnKiteError':0x123,'SpawnKite':0x124,'DestroyKite':0x125},
}
def find(o):
    if isinstance(o, dict):
        if isinstance(o.get('writers'), list): return o['writers']
        for v in o.values():
            r = find(v)
            if r is not None: return r
    if isinstance(o, list):
        for v in o:
            r = find(v)
            if r is not None: return r
    return None
bad = 0
for name, want in expect.items():
    p = 'services/atlas-configurations/seed-data/templates/' + name
    w = find(json.load(open(p)))
    got = {e['writer']: int(e['opCode'], 16) for e in w if e['writer'] in want}
    nums = [int(e['opCode'], 16) for e in w]
    if got != want:
        print('FAIL bindings', name, got, '!=', want); bad = 1
    if nums != sorted(nums):
        print('FAIL not ascending', name); bad = 1
    for e in w:
        if e['writer'] in want and not e.get('fname'):
            print('FAIL missing fname', name, e['writer']); bad = 1
print('OK' if not bad else 'FAILURES ABOVE')
EOF
```

Expected: `OK`.

- [ ] **Step 4: Run the three template guards**

```bash
tools/template-opcode-order-guard.sh; echo "order=$?"
tools/template-duplicate-binding-guard.sh; echo "dup=$?"
tools/template-movement-types-guard.sh; echo "move=$?"
```

Expected: `order=0 dup=0 move=0`.

- [ ] **Step 5: Confirm no new handler entry was needed**

```bash
grep -c CharacterCashItemUseHandle services/atlas-configurations/seed-data/templates/template_gms_{48,61,72,79,83,84,87,92,95}_1.json \
                                    services/atlas-configurations/seed-data/templates/template_jms_185_1.json
```

Expected: `1` for each of the ten. The type-18 sub-body rides this existing binding; no handler entry is added (design ADR-8).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(atlas-configurations): bind kite writers in all ten templates"
```

---

## Task 5: `kite-configs` tenant configuration resource in `atlas-tenants`

PRD FR-8.1 requires the placement knobs to be tenant-configurable. `atlas-tenants` has **no generic `/configurations/{resource}` route** — every resource is hand-registered at `configuration/resource.go:1205-1252`. Without this task the consumer's fetch always misses and the knobs are permanently pinned to compiled defaults.

Shape: the **`rankings` pattern** — one config per tenant, `GET`/`POST`/`PATCH`/`DELETE` on a single path, **no `/seed` endpoint and no `/{id}` sub-routes**. See [`context.md`](context.md) §5 for why this scope was chosen; no `atlas-ui` page is built.

**Files:**
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/rest.go` (add `KiteConfigRestModel`, `TransformKiteConfig`, `ExtractKiteConfig`, `CreateSingleKiteConfigJsonData` beside the `Rankings*` equivalents at `:532-600`)
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/provider.go` (add `GetKiteConfigProvider`, modelled on `GetRankingsProvider` at `:286`)
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/processor.go` (add `CreateKiteConfig`/`CreateKiteConfigAndEmit`/`UpdateKiteConfig`/`UpdateKiteConfigAndEmit`/`DeleteKiteConfig`/`DeleteKiteConfigAndEmit`/`GetKiteConfig`/`KiteConfigProvider`, modelled on the `Rankings` set at `:1553-1710`)
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/resource.go` (add the four handlers beside `GetRankingsHandler` at `:1028-1180`, and four routes after the rankings block at `:1252`)
- Modify: `services/atlas-tenants/docs/rest.md` (document the new resource)

**Interfaces:**
- Produces: `GET /tenants/{tenantId}/configurations/kite-configs` returning a JSON:API document of type `kite-configs` with attributes `maxPerMap` (int), `maxMessageLength` (int), `blockedMapPrefixes` (array of uint32). Task 8's `atlas-kites/configuration` package consumes exactly these three attribute names.

- [ ] **Step 1: Read the rankings resource end to end**

Read these four spans before writing anything — the plan below deliberately mirrors them rather than inventing a shape:

```bash
sed -n '532,600p' services/atlas-tenants/atlas.com/tenants/configuration/rest.go
sed -n '286,320p' services/atlas-tenants/atlas.com/tenants/configuration/provider.go
sed -n '1553,1715p' services/atlas-tenants/atlas.com/tenants/configuration/processor.go
sed -n '1028,1180p' services/atlas-tenants/atlas.com/tenants/configuration/resource.go
```

- [ ] **Step 2: Write the failing REST-model test**

Append to `services/atlas-tenants/atlas.com/tenants/configuration/rest_test.go`:

```go
func TestKiteConfigTransformExtractRoundTrip(t *testing.T) {
	data := map[string]interface{}{
		"id":                 "kite-configs",
		"maxPerMap":          float64(10),
		"maxMessageLength":   float64(182),
		"blockedMapPrefixes": []interface{}{float64(91)},
	}
	rm, err := TransformKiteConfig(data)
	if err != nil {
		t.Fatalf("TransformKiteConfig: %v", err)
	}
	if rm.MaxPerMap != 10 {
		t.Errorf("MaxPerMap = %d, want 10", rm.MaxPerMap)
	}
	if rm.MaxMessageLength != 182 {
		t.Errorf("MaxMessageLength = %d, want 182", rm.MaxMessageLength)
	}
	if len(rm.BlockedMapPrefixes) != 1 || rm.BlockedMapPrefixes[0] != 91 {
		t.Errorf("BlockedMapPrefixes = %v, want [91]", rm.BlockedMapPrefixes)
	}
	if rm.GetName() != "kite-configs" {
		t.Errorf("GetName() = %s, want kite-configs", rm.GetName())
	}

	out, err := ExtractKiteConfig(rm)
	if err != nil {
		t.Fatalf("ExtractKiteConfig: %v", err)
	}
	if out["maxPerMap"] != 10 {
		t.Errorf("round-trip maxPerMap = %v, want 10", out["maxPerMap"])
	}
}
```

- [ ] **Step 3: Run it to verify it fails**

```bash
cd services/atlas-tenants/atlas.com/tenants && go test ./configuration/ -run TestKiteConfig -v 2>&1 | head -20
```

Expected: FAIL — `undefined: TransformKiteConfig`.

- [ ] **Step 4: Add the REST model and transforms**

Append to `services/atlas-tenants/atlas.com/tenants/configuration/rest.go`:

```go
// KiteConfigRestModel is the JSON:API resource for the per-tenant kite
// (cash category 508 message-box) placement policy. One row per tenant, like
// rankings — there is no id-addressed sub-resource.
type KiteConfigRestModel struct {
	Id                 string   `json:"-"`
	MaxPerMap          int      `json:"maxPerMap"`
	MaxMessageLength   int      `json:"maxMessageLength"`
	BlockedMapPrefixes []uint32 `json:"blockedMapPrefixes"`
}

func (r KiteConfigRestModel) GetID() string {
	return r.Id
}

func (r *KiteConfigRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r KiteConfigRestModel) GetName() string {
	return "kite-configs"
}

// TransformKiteConfig converts the stored JSONB map into a KiteConfigRestModel.
func TransformKiteConfig(data map[string]interface{}) (KiteConfigRestModel, error) {
	id, _ := data["id"].(string)

	readInt := func(key string) int {
		if v, ok := data[key].(float64); ok {
			return int(v)
		}
		if v, ok := data[key].(int); ok {
			return v
		}
		return 0
	}

	var prefixes []uint32
	if raw, ok := data["blockedMapPrefixes"].([]interface{}); ok {
		for _, e := range raw {
			switch v := e.(type) {
			case float64:
				prefixes = append(prefixes, uint32(v))
			case int:
				prefixes = append(prefixes, uint32(v))
			}
		}
	}

	return KiteConfigRestModel{
		Id:                 id,
		MaxPerMap:          readInt("maxPerMap"),
		MaxMessageLength:   readInt("maxMessageLength"),
		BlockedMapPrefixes: prefixes,
	}, nil
}

// ExtractKiteConfig converts a KiteConfigRestModel back into the stored JSONB map.
func ExtractKiteConfig(r KiteConfigRestModel) (map[string]interface{}, error) {
	prefixes := make([]interface{}, 0, len(r.BlockedMapPrefixes))
	for _, p := range r.BlockedMapPrefixes {
		prefixes = append(prefixes, p)
	}
	return map[string]interface{}{
		"id":                 r.Id,
		"maxPerMap":          r.MaxPerMap,
		"maxMessageLength":   r.MaxMessageLength,
		"blockedMapPrefixes": prefixes,
	}, nil
}

// CreateSingleKiteConfigJsonData wraps one kite config in a JSON:API document,
// mirroring CreateSingleRankingsJsonData.
func CreateSingleKiteConfigJsonData(cfg map[string]interface{}) (json.RawMessage, error) {
	id, _ := cfg["id"].(string)
	attrs := make(map[string]interface{}, len(cfg))
	for k, v := range cfg {
		if k == "id" {
			continue
		}
		attrs[k] = v
	}
	return json.Marshal(map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "kite-configs",
			"id":         id,
			"attributes": attrs,
		},
	})
}
```

- [ ] **Step 5: Run the model test to verify it passes**

```bash
cd services/atlas-tenants/atlas.com/tenants && go test ./configuration/ -run TestKiteConfig -v
```

Expected: PASS.

- [ ] **Step 6: Add the provider, processor methods, handlers, and routes**

Mirror the rankings implementations read in Step 1, substituting the resource name `"kite-configs"` for `"rankings"` and `KiteConfig` for `Rankings` throughout:

- `provider.go`: `GetKiteConfigProvider(tenantID uuid.UUID) func(db *gorm.DB) model.Provider[map[string]interface{}]`, delegating to `GetByTenantIdAndResourceNameProvider(tenantID, "kite-configs")`.
- `processor.go`: `CreateKiteConfig(mb *message.Buffer) func(tenantId uuid.UUID) func(cfg map[string]interface{}) (Model, error)` plus `CreateKiteConfigAndEmit`, and the matching `UpdateKiteConfig`/`UpdateKiteConfigAndEmit`, `DeleteKiteConfig`/`DeleteKiteConfigAndEmit`, `GetKiteConfig(tenantId uuid.UUID) (map[string]interface{}, error)`, `KiteConfigProvider(tenantId uuid.UUID) model.Provider[map[string]interface{}]`. Add each new method to the `Processor` interface alongside its `Rankings` sibling.
- `resource.go`: `GetKiteConfigHandler`, `CreateKiteConfigHandler`, `UpdateKiteConfigHandler`, `DeleteKiteConfigHandler` and a `registerKiteConfigInputHandler` alias mirroring `registerRankingsInputHandler`.
- `resource.go` routes — append immediately after the rankings block (`:1252`):

```go
			// Kite config endpoints — one config per tenant (rankings shape:
			// no /seed endpoint, no id-addressed sub-resource).
			r.HandleFunc("/tenants/{tenantId}/configurations/kite-configs", registerHandler("get_kite_config", GetKiteConfigHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/kite-configs", registerKiteConfigInputHandler("create_kite_config", CreateKiteConfigHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/kite-configs", registerKiteConfigInputHandler("update_kite_config", UpdateKiteConfigHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/kite-configs", registerHandler("delete_kite_config", DeleteKiteConfigHandler(db))).Methods(http.MethodDelete)
```

- [ ] **Step 7: Write and run a handler test**

Append to `services/atlas-tenants/atlas.com/tenants/configuration/rankings_handler_test.go` (which already has the httptest scaffolding for a singleton config resource) a `TestGetKiteConfigHandlerNotFound` and `TestCreateThenGetKiteConfigHandler` pair, mirroring the existing rankings handler tests exactly and asserting the JSON:API `type` is `kite-configs`.

```bash
cd services/atlas-tenants/atlas.com/tenants && go test ./configuration/ -run 'KiteConfig' -v
```

Expected: PASS.

- [ ] **Step 8: Module gate**

```bash
cd services/atlas-tenants/atlas.com/tenants && go test -race ./... && go vet ./... && go build ./...
```

Expected: all clean.

- [ ] **Step 9: Document the resource**

Add a `kite-configs` section to `services/atlas-tenants/docs/rest.md` beside the `mts-configs`/`rankings` entries, listing the four routes and the three attributes with their meanings and defaults (`maxPerMap` 10, `maxMessageLength` 182, `blockedMapPrefixes` `[91]`).

- [ ] **Step 10: Commit**

```bash
git add services/atlas-tenants/
git commit -m "feat(atlas-tenants): add kite-configs configuration resource"
```

---

## Task 6: Scaffold the `atlas-kites` module

A buildable skeleton: module, Kafka message contracts, buffer, consumer config, REST helpers, and a `main.go` that boots. No domain logic yet — that lands in Tasks 7–11 on top of a compiling base.

**Files:**
- Create: `services/atlas-kites/atlas.com/kites/go.mod`
- Create: `services/atlas-kites/atlas.com/kites/main.go`
- Create: `services/atlas-kites/atlas.com/kites/rest/handler.go`
- Create: `services/atlas-kites/atlas.com/kites/kafka/message/message.go`
- Create: `services/atlas-kites/atlas.com/kites/kafka/message/kite/kafka.go`
- Create: `services/atlas-kites/atlas.com/kites/kafka/message/character/kafka.go`
- Create: `services/atlas-kites/atlas.com/kites/kafka/consumer/consumer.go`
- Modify: `go.work`

**Interfaces:**
- Produces: `kite.EnvCommandTopic = "COMMAND_TOPIC_KITE"`, `kite.EnvEventTopicStatus = "EVENT_TOPIC_KITE_STATUS"`, the `Command[E]`/`StatusEvent[E]` envelopes and every body type. Tasks 9, 11, 13, 15 all reference these names; the `atlas-channel` mirror in Task 13 must match field-for-field.

- [ ] **Step 1: Copy the chalkboards module skeleton**

```bash
mkdir -p services/atlas-kites/atlas.com/kites/{kite,character,configuration,rest,kafka/message/kite,kafka/message/character,kafka/consumer/kite,kafka/consumer/character}
cp services/atlas-chalkboards/atlas.com/chalkboards/go.mod services/atlas-kites/atlas.com/kites/go.mod
cp services/atlas-chalkboards/atlas.com/chalkboards/rest/handler.go services/atlas-kites/atlas.com/kites/rest/handler.go
cp services/atlas-chalkboards/atlas.com/chalkboards/kafka/message/message.go services/atlas-kites/atlas.com/kites/kafka/message/message.go
cp services/atlas-chalkboards/atlas.com/chalkboards/kafka/consumer/consumer.go services/atlas-kites/atlas.com/kites/kafka/consumer/consumer.go
```

Then edit `services/atlas-kites/atlas.com/kites/go.mod` line 1 to `module atlas-kites` and change every `replace` target's relative depth check — the path depth is identical to chalkboards (`services/<svc>/atlas.com/<name>` → `../../../../libs/...`), so the `replace` block copies unchanged. `rest/handler.go`, `kafka/message/message.go`, and `kafka/consumer/consumer.go` have no `atlas-chalkboards` imports and copy verbatim.

- [ ] **Step 2: Write the kite Kafka contracts**

Create `services/atlas-kites/atlas.com/kites/kafka/message/kite/kafka.go`:

```go
package kite

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic     = "COMMAND_TOPIC_KITE"
	CommandKiteCreate   = "CREATE"
	CommandKiteDestroy  = "DESTROY"
)

// Command is produced by atlas-channel and keyed on characterId, so one
// character's placements are totally ordered within a partition. That ordering
// is what makes the one-kite-per-character invariant safe without a lock; the
// per-map cap is NOT covered by it (two characters land on two partitions) and
// takes the per-field lock in the processor instead.
type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

// CreateCommandBody carries the owner's name and position from atlas-channel.
// Both are server-side state: the client never sends coordinates for a kite
// (the sub-body is the message alone) and the name is read from the character
// record, never from the packet.
type CreateCommandBody struct {
	Name       string `json:"name"`
	TemplateId uint32 `json:"templateId"`
	Message    string `json:"message"`
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
}

type DestroyCommandBody struct {
	KiteId uint32 `json:"kiteId"`
}

const (
	EnvEventTopicStatus                 = "EVENT_TOPIC_KITE_STATUS"
	EventTopicStatusTypeCreated         = "CREATED"
	EventTopicStatusTypeDestroyed       = "DESTROYED"
	EventTopicStatusTypeCreationFailed  = "CREATION_FAILED"
)

// Destroy reasons.
const (
	DestroyReasonOwnerLeft       = "OWNER_LEFT"
	DestroyReasonOwnerLoggedOut  = "OWNER_LOGGED_OUT"
)

// Creation-failure reasons. FieldKiteError has an EMPTY body, so the client
// only ever renders a generic failure — these values exist so the refusal is
// diagnosable in logs.
const (
	FailureReasonMapFull        = "MAP_FULL"
	FailureReasonAlreadyPlaced  = "ALREADY_PLACED"
	FailureReasonMapForbidden   = "MAP_FORBIDDEN"
	FailureReasonMessageTooLong = "MESSAGE_TOO_LONG"
)

// StatusEvent is produced by atlas-kites and keyed on mapId for per-map
// ordering, matching the chalkboard/mist producers.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type CreatedStatusEventBody struct {
	KiteId     uint32 `json:"kiteId"`
	Name       string `json:"name"`
	TemplateId uint32 `json:"templateId"`
	Message    string `json:"message"`
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
}

type DestroyedStatusEventBody struct {
	KiteId uint32 `json:"kiteId"`
	Reason string `json:"reason"`
}

// CreationFailedStatusEventBody targets a single character: CANNOT_SPAWN_KITE
// goes to the requester only, never to the map. The envelope's CharacterId is
// the addressee.
type CreationFailedStatusEventBody struct {
	Reason string `json:"reason"`
}
```

- [ ] **Step 3: Copy the character status contract**

```bash
cp services/atlas-chalkboards/atlas.com/chalkboards/kafka/message/character/kafka.go \
   services/atlas-kites/atlas.com/kites/kafka/message/character/kafka.go
```

This file has no service-local imports and copies verbatim. It already carries `Instance` on all four bodies (`StatusEventLoginBody.Instance`, `StatusEventLogoutBody.Instance`, `StatusEventMapChangedBody.OldInstance`/`TargetInstance`, `ChangeChannelEventLoginBody.Instance`) — Task 11 must actually use them, unlike the chalkboards consumer.

- [ ] **Step 4: Write a `main.go` that boots**

Create `services/atlas-kites/atlas.com/kites/main.go` modelled on
`services/atlas-chalkboards/atlas.com/chalkboards/main.go`, with `serviceName = "atlas-kites"`, `consumerGroupId = consumergroup.Resolve("Kite Service")`, and — for now — no registry init, no consumers, and no route initializer beyond the debug/readiness mounts. Tasks 7–11 fill these in.

- [ ] **Step 5: Register the module in the workspace**

Add to `go.work` in the `use()` block, keeping the list's existing ordering:

```
	./services/atlas-kites/atlas.com/kites
```

- [ ] **Step 6: Build**

```bash
go work sync
cd services/atlas-kites/atlas.com/kites && go build ./... && go vet ./...
```

Expected: clean. Prune any `go.mod` requirement the skeleton does not yet use with `go mod tidy` if the build complains.

- [ ] **Step 7: Commit**

```bash
git add go.work go.work.sum services/atlas-kites/
git commit -m "feat(atlas-kites): scaffold service module and kafka contracts"
```

---

## Task 7: `atlas-kites` domain — model, builder, and the two registries

State layout per design ADR-2: the kite registry is keyed on `characterId` (safe because FR-5.2 makes it one kite per character), and the field→kites lookup is the intersection of *characters in that field* with *characters owning a kite* — the same trick `chalkboard/resource.go:76-84` uses, with no second index to keep consistent.

**Files:**
- Create: `services/atlas-kites/atlas.com/kites/kite/model.go`
- Create: `services/atlas-kites/atlas.com/kites/kite/builder.go`
- Create: `services/atlas-kites/atlas.com/kites/kite/registry.go`
- Create: `services/atlas-kites/atlas.com/kites/kite/registry_test.go`
- Create: `services/atlas-kites/atlas.com/kites/character/model.go`
- Create: `services/atlas-kites/atlas.com/kites/character/registry.go`
- Create: `services/atlas-kites/atlas.com/kites/character/processor.go`
- Create: `services/atlas-kites/atlas.com/kites/character/registry_test.go`

**Interfaces:**
- Produces:
  - `kite.Model` with getters `Id() uint32`, `Field() field.Model`, `CharacterId() uint32`, `Name() string`, `TemplateId() uint32`, `Message() string`, `X() int16`, `Y() int16`, `CreatedAt() time.Time`
  - `kite.NewBuilder(id uint32, f field.Model, characterId uint32) *Builder` with `SetName`, `SetTemplateId`, `SetMessage`, `SetPosition(x, y int16)`, `SetCreatedAt(time.Time)`, `Build() Model`
  - `kite.InitRegistry(client *goredis.Client)`; registry methods `Get(ctx, characterId) (Model, bool)`, `Put(ctx, Model) error`, `Remove(ctx, characterId) error`, `Exists(ctx, characterId) (bool, error)`, `NextId(ctx) (uint32, error)`, `AcquireFieldLock(ctx, f field.Model) (bool, error)`, `ReleaseFieldLock(ctx, f field.Model) error`
  - `character.InitRegistry(client *goredis.Client)`; `character.NewProcessor(l, ctx)` with `InMapProvider(f field.Model) model.Provider[[]uint32]`, `Enter`, `Exit`, `TransitionMap`, `TransitionChannel`

- [ ] **Step 1: Write the failing registry test**

Create `services/atlas-kites/atlas.com/kites/kite/registry_test.go`, following the miniredis pattern already used at `services/atlas-chalkboards/atlas.com/chalkboards/chalkboard/registry_test.go`:

```go
package kite

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testContext(t *testing.T) (context.Context, tenant.Model) {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tm), tm
}

func testRegistry(t *testing.T) {
	t.Helper()
	s := miniredis.RunT(t)
	InitRegistry(goredis.NewClient(&goredis.Options{Addr: s.Addr()}))
}

func testField() field.Model {
	return field.NewBuilder(0, 1, 104040000).SetInstance(uuid.Nil).Build()
}

func TestRegistryPutGetRemove(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)

	m := NewBuilder(1000000001, testField(), 42).
		SetName("Player").
		SetTemplateId(5080000).
		SetMessage("congrats!").
		SetPosition(320, -140).
		SetCreatedAt(time.Unix(0, 0).UTC()).
		Build()

	if err := getRegistry().Put(ctx, m); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, ok := getRegistry().Get(ctx, 42)
	if !ok {
		t.Fatal("Get: kite not found after Put")
	}
	if got.Id() != 1000000001 || got.Message() != "congrats!" || got.X() != 320 || got.Y() != -140 {
		t.Errorf("Get returned %+v", got)
	}
	if got.Field().Instance() != uuid.Nil || got.Field().MapId() != 104040000 {
		t.Errorf("field did not round-trip: %+v", got.Field())
	}

	if err := getRegistry().Remove(ctx, 42); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, ok := getRegistry().Get(ctx, 42); ok {
		t.Error("Get: kite still present after Remove")
	}
}

func TestRegistryNextIdIsMonotonic(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)

	a, err := getRegistry().NextId(ctx)
	if err != nil {
		t.Fatalf("NextId: %v", err)
	}
	b, err := getRegistry().NextId(ctx)
	if err != nil {
		t.Fatalf("NextId: %v", err)
	}
	if b <= a {
		t.Errorf("NextId not monotonic: %d then %d", a, b)
	}
}

func TestRegistryFieldLockIsExclusive(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)
	f := testField()

	ok, err := getRegistry().AcquireFieldLock(ctx, f)
	if err != nil || !ok {
		t.Fatalf("first AcquireFieldLock: ok=%v err=%v", ok, err)
	}
	ok, err = getRegistry().AcquireFieldLock(ctx, f)
	if err != nil {
		t.Fatalf("second AcquireFieldLock: %v", err)
	}
	if ok {
		t.Error("second AcquireFieldLock succeeded while the first was held")
	}
	if err := getRegistry().ReleaseFieldLock(ctx, f); err != nil {
		t.Fatalf("ReleaseFieldLock: %v", err)
	}
	ok, _ = getRegistry().AcquireFieldLock(ctx, f)
	if !ok {
		t.Error("AcquireFieldLock failed after release")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

```bash
cd services/atlas-kites/atlas.com/kites && go test ./kite/ 2>&1 | head -20
```

Expected: FAIL — `undefined: InitRegistry`, `undefined: NewBuilder`.

- [ ] **Step 3: Write the model and builder**

Create `services/atlas-kites/atlas.com/kites/kite/model.go`:

```go
package kite

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Model is one kite (cash category 508 message box) hanging in one field.
// Immutable: construct via Builder.
type Model struct {
	id          uint32
	f           field.Model
	characterId uint32
	name        string
	templateId  uint32
	message     string
	x           int16
	y           int16
	createdAt   time.Time
}

func (m Model) Id() uint32            { return m.id }
func (m Model) Field() field.Model    { return m.f }
func (m Model) CharacterId() uint32   { return m.characterId }
func (m Model) Name() string          { return m.name }
func (m Model) TemplateId() uint32    { return m.templateId }
func (m Model) Message() string       { return m.message }
func (m Model) X() int16              { return m.x }
func (m Model) Y() int16              { return m.y }
func (m Model) CreatedAt() time.Time  { return m.createdAt }
```

Create `services/atlas-kites/atlas.com/kites/kite/builder.go`:

```go
package kite

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type Builder struct {
	id          uint32
	f           field.Model
	characterId uint32
	name        string
	templateId  uint32
	message     string
	x           int16
	y           int16
	createdAt   time.Time
}

func NewBuilder(id uint32, f field.Model, characterId uint32) *Builder {
	return &Builder{id: id, f: f, characterId: characterId}
}

func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

func (b *Builder) SetTemplateId(templateId uint32) *Builder {
	b.templateId = templateId
	return b
}

func (b *Builder) SetMessage(message string) *Builder {
	b.message = message
	return b
}

func (b *Builder) SetPosition(x int16, y int16) *Builder {
	b.x = x
	b.y = y
	return b
}

func (b *Builder) SetCreatedAt(t time.Time) *Builder {
	b.createdAt = t
	return b
}

func (b *Builder) Build() Model {
	return Model{
		id:          b.id,
		f:           b.f,
		characterId: b.characterId,
		name:        b.name,
		templateId:  b.templateId,
		message:     b.message,
		x:           b.x,
		y:           b.y,
		createdAt:   b.createdAt,
	}
}
```

- [ ] **Step 4: Write the registry**

Create `services/atlas-kites/atlas.com/kites/kite/registry.go`. `TenantRegistry` JSON-serialises its value, so the stored type is an exported-field DTO, not the private-field `Model`.

```go
package kite

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// entry is the serialised form of Model. TenantRegistry marshals its value to
// JSON, so the stored shape needs exported fields; Model stays immutable.
type entry struct {
	Id          uint32     `json:"id"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
	Instance    uuid.UUID  `json:"instance"`
	CharacterId uint32     `json:"characterId"`
	Name        string     `json:"name"`
	TemplateId  uint32     `json:"templateId"`
	Message     string     `json:"message"`
	X           int16      `json:"x"`
	Y           int16      `json:"y"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type Registry struct {
	reg  *atlas.TenantRegistry[uint32, entry]
	ids  *atlas.IDGenerator
	lock *atlas.Lock
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[uint32, entry](client, "kite", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		ids:  atlas.NewIDGenerator(client, "kite"),
		lock: atlas.NewLock(client, "kite-cap"),
	}
}

func getRegistry() *Registry {
	return registry
}

func toEntry(m Model) entry {
	return entry{
		Id:          m.Id(),
		WorldId:     m.Field().WorldId(),
		ChannelId:   m.Field().ChannelId(),
		MapId:       m.Field().MapId(),
		Instance:    m.Field().Instance(),
		CharacterId: m.CharacterId(),
		Name:        m.Name(),
		TemplateId:  m.TemplateId(),
		Message:     m.Message(),
		X:           m.X(),
		Y:           m.Y(),
		CreatedAt:   m.CreatedAt(),
	}
}

func fromEntry(e entry) Model {
	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
	return NewBuilder(e.Id, f, e.CharacterId).
		SetName(e.Name).
		SetTemplateId(e.TemplateId).
		SetMessage(e.Message).
		SetPosition(e.X, e.Y).
		SetCreatedAt(e.CreatedAt).
		Build()
}

func (r *Registry) Get(ctx context.Context, characterId uint32) (Model, bool) {
	t := tenant.MustFromContext(ctx)
	e, err := r.reg.Get(ctx, t, characterId)
	if err != nil {
		return Model{}, false
	}
	return fromEntry(e), true
}

func (r *Registry) Put(ctx context.Context, m Model) error {
	t := tenant.MustFromContext(ctx)
	return r.reg.Put(ctx, t, m.CharacterId(), toEntry(m))
}

func (r *Registry) Remove(ctx context.Context, characterId uint32) error {
	t := tenant.MustFromContext(ctx)
	return r.reg.Remove(ctx, t, characterId)
}

func (r *Registry) Exists(ctx context.Context, characterId uint32) (bool, error) {
	t := tenant.MustFromContext(ctx)
	return r.reg.Exists(ctx, t, characterId)
}

// NextId allocates the wire id. It is a tenant-scoped Redis INCR, not a
// process-local counter, because REMOVE_KITE addresses a kite by this id alone
// and any atlas-kites replica may be the one that allocated it.
func (r *Registry) NextId(ctx context.Context) (uint32, error) {
	return r.ids.NextID(ctx, tenant.MustFromContext(ctx))
}

// fieldLockKey includes the tenant because atlas.Lock is NOT tenant-scoped —
// it namespaces by the constructor's namespace only.
func fieldLockKey(t tenant.Model, f field.Model) string {
	return fmt.Sprintf("%s:%d:%d:%d:%s", t.Id().String(), f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}

// AcquireFieldLock serialises {count -> validate -> allocate -> insert} for one
// field. The per-character invariant is already safe (the command topic is
// keyed on characterId, so one character's commands share a partition), but the
// per-map cap is not: two different characters placing on the same
// full-but-for-one map land on different partitions.
func (r *Registry) AcquireFieldLock(ctx context.Context, f field.Model) (bool, error) {
	return r.lock.Acquire(ctx, fieldLockKey(tenant.MustFromContext(ctx), f))
}

func (r *Registry) ReleaseFieldLock(ctx context.Context, f field.Model) error {
	return r.lock.Release(ctx, fieldLockKey(tenant.MustFromContext(ctx), f))
}
```

- [ ] **Step 5: Run the kite registry tests**

```bash
cd services/atlas-kites/atlas.com/kites && go test ./kite/ -v
```

Expected: PASS on all three.

- [ ] **Step 6: Write the character-in-field index**

Copy the three chalkboards files and apply the **instance fix**:

```bash
cp services/atlas-chalkboards/atlas.com/chalkboards/character/model.go     services/atlas-kites/atlas.com/kites/character/model.go
cp services/atlas-chalkboards/atlas.com/chalkboards/character/registry.go  services/atlas-kites/atlas.com/kites/character/registry.go
cp services/atlas-chalkboards/atlas.com/chalkboards/character/processor.go services/atlas-kites/atlas.com/kites/character/processor.go
cp services/atlas-chalkboards/atlas.com/chalkboards/character/registry_test.go services/atlas-kites/atlas.com/kites/character/registry_test.go
```

In `character/registry.go`, change the namespace from `"chalk-char"` to `"kite-char"`. The key function already includes `f.Instance().String()` and is correct as-is — the chalkboards bug is on the *consumer* side (Task 11), not here.

Add a test to `character/registry_test.go` proving the instance participates in the key:

```go
func TestInMapIsInstanceScoped(t *testing.T) {
	s := miniredis.RunT(t)
	InitRegistry(goredis.NewClient(&goredis.Options{Addr: s.Addr()}))

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)

	inst := uuid.New()
	base := field.NewBuilder(0, 1, 104040000).SetInstance(uuid.Nil).Build()
	instanced := field.NewBuilder(0, 1, 104040000).SetInstance(inst).Build()

	NewProcessor(nil, ctx).Enter(instanced, 42)

	if got := NewProcessor(nil, ctx).GetCharactersInMapOrNil(base); len(got) != 0 {
		t.Errorf("base field saw the instanced character: %v", got)
	}
	got, err := NewProcessor(nil, ctx).GetCharactersInMap(instanced)
	if err != nil {
		t.Fatalf("GetCharactersInMap: %v", err)
	}
	if len(got) != 1 || got[0] != 42 {
		t.Errorf("instanced field = %v, want [42]", got)
	}
}
```

If `GetCharactersInMapOrNil` does not exist on the copied processor, use `GetCharactersInMap` and ignore its error — do not add a helper that only a test uses.

- [ ] **Step 7: Run the character tests**

```bash
cd services/atlas-kites/atlas.com/kites && go test -race ./... && go vet ./...
```

Expected: clean.

- [ ] **Step 8: Redis guard**

```bash
tools/redis-key-guard.sh; echo "exit=$?"
```

Expected: `exit=0`. Every keyed command above goes through `libs/atlas-redis`; a raw `client.Get`/`client.Set` in this package is a guard failure.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-kites/
git commit -m "feat(atlas-kites): kite model, builder, redis registry, character index"
```

---

## Task 8: `atlas-kites` placement-policy configuration

Consumes the `kite-configs` resource built in Task 5, on the `mts-configs` consumer pattern: `Extract` folds every zero-valued knob back to its compiled default, and the registry falls back to `DefaultConfig()` on a fetch miss — so the feature works un-provisioned.

**Files:**
- Create: `services/atlas-kites/atlas.com/kites/configuration/model.go`
- Create: `services/atlas-kites/atlas.com/kites/configuration/rest.go`
- Create: `services/atlas-kites/atlas.com/kites/configuration/requests.go`
- Create: `services/atlas-kites/atlas.com/kites/configuration/registry.go`
- Create: `services/atlas-kites/atlas.com/kites/configuration/rest_test.go`

**Interfaces:**
- Consumes: `GET {TENANTS}/tenants/{tenantId}/configurations/kite-configs` with attributes `maxPerMap`, `maxMessageLength`, `blockedMapPrefixes` (Task 5).
- Produces: `configuration.Model` with `MaxPerMap() int`, `MaxMessageLength() int`, `BlockedMapPrefixes() []uint32`, `IsMapBlocked(mapId _map.Id) bool`; `configuration.DefaultConfig() Model`; `configuration.GetRegistry().GetTenantConfig(l, ctx, tenantId uuid.UUID) Model`. Task 9 calls `GetTenantConfig` and `IsMapBlocked`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-kites/atlas.com/kites/configuration/rest_test.go`:

```go
package configuration

import "testing"

func TestExtractFoldsZeroKnobsToDefaults(t *testing.T) {
	m := Extract(RestModel{})
	d := DefaultConfig()
	if m.MaxPerMap() != d.MaxPerMap() {
		t.Errorf("MaxPerMap = %d, want %d", m.MaxPerMap(), d.MaxPerMap())
	}
	if m.MaxMessageLength() != d.MaxMessageLength() {
		t.Errorf("MaxMessageLength = %d, want %d", m.MaxMessageLength(), d.MaxMessageLength())
	}
	if len(m.BlockedMapPrefixes()) != len(d.BlockedMapPrefixes()) {
		t.Errorf("BlockedMapPrefixes = %v, want %v", m.BlockedMapPrefixes(), d.BlockedMapPrefixes())
	}
}

func TestExtractKeepsProvidedKnobs(t *testing.T) {
	m := Extract(RestModel{MaxPerMap: 3, MaxMessageLength: 40, BlockedMapPrefixes: []uint32{91, 92}})
	if m.MaxPerMap() != 3 || m.MaxMessageLength() != 40 {
		t.Errorf("got maxPerMap=%d maxMessageLength=%d", m.MaxPerMap(), m.MaxMessageLength())
	}
	if len(m.BlockedMapPrefixes()) != 2 {
		t.Errorf("BlockedMapPrefixes = %v", m.BlockedMapPrefixes())
	}
}

// The client's own rule is GetCurFieldID() / 10000000 == 91 -> refuse
// (CWvsContext::SendConsumeCashItemUseRequest case 18, gms_v95 @0x9ed017).
// IsMapBlocked mirrors that arithmetic exactly.
func TestIsMapBlockedMirrorsClientArithmetic(t *testing.T) {
	m := DefaultConfig()
	if !m.IsMapBlocked(910000000) {
		t.Error("910000000 (Free Market entrance) should be blocked")
	}
	if !m.IsMapBlocked(919999999) {
		t.Error("919999999 (top of the FM range) should be blocked")
	}
	if m.IsMapBlocked(909999999) {
		t.Error("909999999 is below the FM range and must not be blocked")
	}
	if m.IsMapBlocked(920000000) {
		t.Error("920000000 is above the FM range and must not be blocked")
	}
	if m.IsMapBlocked(104040000) {
		t.Error("an ordinary field must not be blocked")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

```bash
cd services/atlas-kites/atlas.com/kites && go test ./configuration/ 2>&1 | head -20
```

Expected: FAIL — `undefined: Extract`, `undefined: RestModel`, `undefined: DefaultConfig`.

- [ ] **Step 3: Write the model**

Create `services/atlas-kites/atlas.com/kites/configuration/model.go`:

```go
package configuration

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// mapPrefixDivisor is the client's own field-id bucket size. The case-18 arm of
// CWvsContext::SendConsumeCashItemUseRequest (gms_v95 @0x9ed01e) divides
// GetCurFieldID() by 10,000,000 with the MSVC magic 0x6B5FCA6B / sar 22 and
// refuses when the quotient is 91 — the Free Market range. Atlas expresses that
// as a configurable prefix denylist rather than hard-coding one map id.
const mapPrefixDivisor = 10000000

// Model is the immutable per-tenant kite placement policy.
type Model struct {
	maxPerMap          int
	maxMessageLength   int
	blockedMapPrefixes []uint32
}

func (m Model) MaxPerMap() int              { return m.maxPerMap }
func (m Model) MaxMessageLength() int       { return m.maxMessageLength }
func (m Model) BlockedMapPrefixes() []uint32 { return m.blockedMapPrefixes }

// IsMapBlocked reports whether the field's map id falls in a denied prefix
// bucket, mirroring the client's arithmetic.
func (m Model) IsMapBlocked(mapId _map.Id) bool {
	prefix := uint32(mapId) / mapPrefixDivisor
	for _, p := range m.blockedMapPrefixes {
		if p == prefix {
			return true
		}
	}
	return false
}

// DefaultConfig is used whenever a tenant has not provisioned kite-configs, and
// for any individual knob left at its zero value.
func DefaultConfig() Model {
	return Model{
		maxPerMap:        10,  // PRD FR-5.1
		maxMessageLength: 182, // 3 x 60-byte CUIHope edit controls + 2 '\n' joiners (CUIHope::OnCreate @0x7824f0)
		// 91 == the Free Market range (910000000-919999999), the client's own ban.
		blockedMapPrefixes: []uint32{91},
	}
}
```

- [ ] **Step 4: Write the REST model, request, and registry**

Create `configuration/rest.go` (`RestModel` with `Id string json:"-"`, `MaxPerMap int json:"maxPerMap"`, `MaxMessageLength int json:"maxMessageLength"`, `BlockedMapPrefixes []uint32 json:"blockedMapPrefixes"`; `GetName()` returns `"kite-configs"`; `Extract(RestModel) Model` folding each zero/empty knob to `DefaultConfig()`'s value), `configuration/requests.go` (`requestForTenant(tenantId uuid.UUID) requests.Request[RestModel]` building `%stenants/%s/configurations/kite-configs` off `requests.RootUrl("TENANTS")`), and `configuration/registry.go` — all three copied structurally from `services/atlas-channel/atlas.com/channel/mts/configuration/`, substituting the resource name and knob set.

- [ ] **Step 5: Run the tests**

```bash
cd services/atlas-kites/atlas.com/kites && go test ./configuration/ -v
```

Expected: PASS on all three tests.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-kites/atlas.com/kites/configuration/
git commit -m "feat(atlas-kites): tenant-configurable placement policy"
```

---

## Task 9: `atlas-kites` processor and producer

The authoritative lifecycle. All five FR-5 refusals are evaluated **here**, not in `atlas-channel`, so the registry enforces its own invariants and two concurrent requests cannot both pass a channel-side check.

**Files:**
- Create: `services/atlas-kites/atlas.com/kites/kite/producer.go`
- Create: `services/atlas-kites/atlas.com/kites/kite/processor.go`
- Create: `services/atlas-kites/atlas.com/kites/kite/processor_test.go`

**Interfaces:**
- Consumes: `kite.Registry` (Task 7), `character.Processor.InMapProvider` (Task 7), `configuration.GetRegistry().GetTenantConfig` (Task 8), `kiteMsg.*` contracts (Task 6).
- Produces:
  ```go
  type Processor interface {
      Create(mb *message.Buffer) func(f field.Model, characterId uint32, cmd kiteMsg.CreateCommandBody) (Model, error)
      CreateAndEmit(f field.Model, characterId uint32, cmd kiteMsg.CreateCommandBody) (Model, error)
      Destroy(mb *message.Buffer) func(characterId uint32, reason string) (Model, error)
      DestroyAndEmit(characterId uint32, reason string) (Model, error)
      GetByCharacterId(characterId uint32) (Model, error)
      InMapModelProvider(f field.Model) model.Provider[[]Model]
      GetInMap(f field.Model) ([]Model, error)
  }
  func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor
  func NewProcessorWithProvider(l logrus.FieldLogger, ctx context.Context, p producer.Provider) Processor
  ```
  Tasks 10 and 11 call these.

Errors are typed so the consumer can log the reason without string-matching:

```go
var (
    ErrMapForbidden   = errors.New("map forbidden")
    ErrMessageTooLong = errors.New("message too long")
    ErrAlreadyPlaced  = errors.New("already placed")
    ErrMapFull        = errors.New("map full")
)
```

- [ ] **Step 1: Write the failing processor tests**

Create `services/atlas-kites/atlas.com/kites/kite/processor_test.go`. Use a recording `producer.Provider` seam (the `NewProcessorWithProvider` constructor exists for exactly this) and miniredis, per the mist-processor precedent.

```go
package kite

import (
	kiteMsg "atlas-kites/kafka/message/kite"
	"errors"
	"sync"
	"testing"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// recorder is a producer.Provider that captures emitted messages per topic.
type recorder struct {
	mu   sync.Mutex
	msgs map[string][]kafka.Message
	fail bool
}

func newRecorder() *recorder { return &recorder{msgs: make(map[string][]kafka.Message)} }

func (r *recorder) provider() producer.Provider {
	return func(t string) func(model.Provider[[]kafka.Message]) error {
		return func(p model.Provider[[]kafka.Message]) error {
			if r.fail {
				return errors.New("emit failed")
			}
			ms, err := p()
			if err != nil {
				return err
			}
			r.mu.Lock()
			defer r.mu.Unlock()
			r.msgs[t] = append(r.msgs[t], ms...)
			return nil
		}
	}
}

func (r *recorder) count(topic string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.msgs[topic])
}

func body() kiteMsg.CreateCommandBody {
	return kiteMsg.CreateCommandBody{Name: "Player", TemplateId: 5080000, Message: "congrats!", X: 320, Y: -140}
}

func TestCreateSucceedsAndEmitsCreated(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)
	rec := newRecorder()

	m, err := NewProcessorWithProvider(nil, ctx, rec.provider()).CreateAndEmit(testField(), 42, body())
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}
	if m.Id() == 0 || m.X() != 320 || m.Y() != -140 || m.Name() != "Player" {
		t.Errorf("unexpected model: %+v", m)
	}
	if rec.count(kiteMsg.EnvEventTopicStatus) != 1 {
		t.Errorf("emitted %d status events, want 1", rec.count(kiteMsg.EnvEventTopicStatus))
	}
	if ok, _ := getRegistry().Exists(ctx, 42); !ok {
		t.Error("kite not in registry after create")
	}
}

func TestCreateRejectsSecondKiteForSameCharacter(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)
	rec := newRecorder()
	p := NewProcessorWithProvider(nil, ctx, rec.provider())

	if _, err := p.CreateAndEmit(testField(), 42, body()); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := p.CreateAndEmit(testField(), 42, body())
	if !errors.Is(err, ErrAlreadyPlaced) {
		t.Fatalf("second create err = %v, want ErrAlreadyPlaced", err)
	}
	// One CREATED + one CREATION_FAILED.
	if rec.count(kiteMsg.EnvEventTopicStatus) != 2 {
		t.Errorf("emitted %d status events, want 2", rec.count(kiteMsg.EnvEventTopicStatus))
	}
}

func TestCreateRejectsBlockedMap(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)
	rec := newRecorder()

	fm := fieldWithMap(910000000)
	_, err := NewProcessorWithProvider(nil, ctx, rec.provider()).CreateAndEmit(fm, 42, body())
	if !errors.Is(err, ErrMapForbidden) {
		t.Fatalf("err = %v, want ErrMapForbidden", err)
	}
	if ok, _ := getRegistry().Exists(ctx, 42); ok {
		t.Error("a refused create must not insert into the registry")
	}
}

func TestCreateRejectsOverlongMessage(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)
	rec := newRecorder()

	b := body()
	b.Message = string(make([]byte, 183))
	_, err := NewProcessorWithProvider(nil, ctx, rec.provider()).CreateAndEmit(testField(), 42, b)
	if !errors.Is(err, ErrMessageTooLong) {
		t.Fatalf("err = %v, want ErrMessageTooLong", err)
	}
}

func TestCreateRollsBackRegistryWhenEmitFails(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)
	rec := newRecorder()
	rec.fail = true

	if _, err := NewProcessorWithProvider(nil, ctx, rec.provider()).CreateAndEmit(testField(), 42, body()); err == nil {
		t.Fatal("CreateAndEmit should fail when the emit fails")
	}
	if ok, _ := getRegistry().Exists(ctx, 42); ok {
		t.Error("registry must not retain a kite whose CREATED event was never emitted")
	}
}

func TestDestroyRemovesAndEmits(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)
	rec := newRecorder()
	p := NewProcessorWithProvider(nil, ctx, rec.provider())

	if _, err := p.CreateAndEmit(testField(), 42, body()); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := p.DestroyAndEmit(42, kiteMsg.DestroyReasonOwnerLeft); err != nil {
		t.Fatalf("DestroyAndEmit: %v", err)
	}
	if ok, _ := getRegistry().Exists(ctx, 42); ok {
		t.Error("kite still present after destroy")
	}
	if rec.count(kiteMsg.EnvEventTopicStatus) != 2 {
		t.Errorf("emitted %d status events, want 2 (created + destroyed)", rec.count(kiteMsg.EnvEventTopicStatus))
	}
}
```

Add the small helper beside `testField()` in `registry_test.go`:

```go
func fieldWithMap(mapId _map.Id) field.Model {
	return field.NewBuilder(0, 1, mapId).SetInstance(uuid.Nil).Build()
}
```

- [ ] **Step 2: Run to verify they fail**

```bash
cd services/atlas-kites/atlas.com/kites && go test ./kite/ -run 'TestCreate|TestDestroy' 2>&1 | head -20
```

Expected: FAIL — `undefined: NewProcessorWithProvider`, `undefined: ErrAlreadyPlaced`.

- [ ] **Step 3: Write the producer**

Create `services/atlas-kites/atlas.com/kites/kite/producer.go` with three providers, all keyed on `producer.CreateKey(int(f.MapId()))` for per-map ordering:

```go
func createdStatusEventProvider(transactionId uuid.UUID, m Model) model.Provider[[]kafka.Message]
func destroyedStatusEventProvider(transactionId uuid.UUID, m Model, reason string) model.Provider[[]kafka.Message]
func creationFailedStatusEventProvider(transactionId uuid.UUID, f field.Model, characterId uint32, reason string) model.Provider[[]kafka.Message]
```

Each fills `kiteMsg.StatusEvent[...]{TransactionId, WorldId, ChannelId, MapId, Instance, CharacterId, Type, Body}` from the field and model, mirroring `services/atlas-chalkboards/atlas.com/chalkboards/chalkboard/producer.go:14-42`.

- [ ] **Step 4: Write the processor**

Create `services/atlas-kites/atlas.com/kites/kite/processor.go`. `Create`'s ordering is fixed by design §4.2 and must not be reordered — the cheap, character-local checks run before the field lock is taken:

```go
// Create validates placement, allocates the wire id, inserts, and buffers
// KITE_CREATED. Every refusal buffers KITE_CREATION_FAILED instead and returns
// a typed error; none of them emits KITE_CREATED (FR-3.5).
//
// Order matters. The map-policy, message-length and one-per-character checks
// are character-local and run BEFORE the field lock, so a refusal never
// contends. Only the per-map cap needs the lock: the command topic is keyed on
// characterId, so one character's commands are totally ordered within a
// partition and FR-5.2 is safe by construction, but two DIFFERENT characters
// placing on the same full-but-for-one map land on different partitions.
func (p *ProcessorImpl) Create(mb *message.Buffer) func(f field.Model, characterId uint32, cmd kiteMsg.CreateCommandBody) (Model, error) {
	return func(f field.Model, characterId uint32, cmd kiteMsg.CreateCommandBody) (Model, error) {
		cfg := configuration.GetRegistry().GetTenantConfig(p.l, p.ctx, p.t.Id())

		refuse := func(reason string, err error) (Model, error) {
			p.l.WithFields(logrus.Fields{
				"tenant": p.t.Id().String(), "character": characterId,
				"world": f.WorldId(), "channel": f.ChannelId(),
				"map": f.MapId(), "instance": f.Instance().String(),
				"reason": reason,
			}).Infof("Refusing kite placement.")
			if bufErr := mb.Put(kiteMsg.EnvEventTopicStatus,
				creationFailedStatusEventProvider(uuid.New(), f, characterId, reason)); bufErr != nil {
				return Model{}, bufErr
			}
			return Model{}, err
		}

		if cfg.IsMapBlocked(f.MapId()) {
			return refuse(kiteMsg.FailureReasonMapForbidden, ErrMapForbidden)
		}
		if len(cmd.Message) > cfg.MaxMessageLength() {
			return refuse(kiteMsg.FailureReasonMessageTooLong, ErrMessageTooLong)
		}
		if exists, err := p.r.Exists(p.ctx, characterId); err != nil {
			return Model{}, err
		} else if exists {
			return refuse(kiteMsg.FailureReasonAlreadyPlaced, ErrAlreadyPlaced)
		}

		locked, err := p.r.AcquireFieldLock(p.ctx, f)
		if err != nil {
			return Model{}, err
		}
		if !locked {
			// Logged distinctly so contention is separable from a genuinely
			// full map; a lost race on a full map refuses either way.
			p.l.Debugf("Kite field lock contended for character [%d]; refusing as MAP_FULL.", characterId)
			return refuse(kiteMsg.FailureReasonMapFull, ErrMapFull)
		}
		defer func() {
			if relErr := p.r.ReleaseFieldLock(p.ctx, f); relErr != nil {
				p.l.WithError(relErr).Warnf("Unable to release kite field lock for map [%d].", f.MapId())
			}
		}()

		inField, err := p.GetInMap(f)
		if err != nil {
			return Model{}, err
		}
		if len(inField) >= cfg.MaxPerMap() {
			return refuse(kiteMsg.FailureReasonMapFull, ErrMapFull)
		}

		id, err := p.r.NextId(p.ctx)
		if err != nil {
			return Model{}, err
		}
		m := NewBuilder(id, f, characterId).
			SetName(cmd.Name).
			SetTemplateId(cmd.TemplateId).
			SetMessage(cmd.Message).
			SetPosition(cmd.X, cmd.Y).
			SetCreatedAt(time.Now()).
			Build()

		if err = p.r.Put(p.ctx, m); err != nil {
			return Model{}, err
		}
		if err = mb.Put(kiteMsg.EnvEventTopicStatus, createdStatusEventProvider(uuid.New(), m)); err != nil {
			return Model{}, err
		}
		return m, nil
	}
}

// isRefusal reports whether err is one of the four typed FR-5 policy
// refusals. A refusal is NOT an emit failure: its CREATION_FAILED event is
// already in the buffer and must still be flushed, so CreateAndEmit lets the
// flush complete and re-raises the refusal afterwards.
func isRefusal(err error) bool {
	return errors.Is(err, ErrMapForbidden) ||
		errors.Is(err, ErrMessageTooLong) ||
		errors.Is(err, ErrAlreadyPlaced) ||
		errors.Is(err, ErrMapFull)
}

// CreateAndEmit rolls the registry insert back when the emit fails, so the
// registry never holds a kite downstream consumers will not see
// (services/atlas-maps/atlas.com/maps/mist/processor.go:94-106).
func (p *ProcessorImpl) CreateAndEmit(f field.Model, characterId uint32, cmd kiteMsg.CreateCommandBody) (Model, error) {
	var m Model
	var refusal error

	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var innerErr error
		m, innerErr = p.Create(buf)(f, characterId, cmd)
		if isRefusal(innerErr) {
			refusal = innerErr
			return nil
		}
		return innerErr
	})
	if err != nil {
		// Only reached when the insert succeeded and the flush failed; Remove
		// on a character with no kite is a no-op, so this is safe for the
		// pre-insert error paths too.
		_ = p.r.Remove(p.ctx, characterId)
		return Model{}, err
	}
	if refusal != nil {
		return Model{}, refusal
	}
	return m, nil
}
```

`Destroy` removes first and treats the removal as authoritative — emit failures are logged, not fatal (mist `Destroy` precedent). `DestroyForCharacter` from PRD FR-3.4 collapses into `Destroy(characterId, reason)` because of the one-per-character invariant; there is no bulk case.

`InMapModelProvider(f)` composes `character.NewProcessor(p.l, p.ctx).InMapProvider(f)` with a filter that keeps only character ids owning a kite, then maps them through `GetByCharacterId` — exactly `chalkboard/resource.go:71-92`.

- [ ] **Step 5: Run the processor tests**

```bash
cd services/atlas-kites/atlas.com/kites && go test ./kite/ -race -v
```

Expected: PASS on all six new tests plus the three from Task 7.

- [ ] **Step 6: Add the concurrency test**

Append to `processor_test.go` a test that runs two goroutines creating kites for two *different* characters against a config with `maxPerMap: 1`, and asserts exactly one succeeds and one returns `ErrMapFull`. Spawn them with `routine.Go` — a bare `go` statement fails `tools/goroutine-guard.sh`.

```bash
cd services/atlas-kites/atlas.com/kites && go test ./kite/ -race -run TestConcurrent -count=20
```

Expected: PASS on every one of the 20 runs. A flake here means the lock is not covering the count→insert window.

- [ ] **Step 7: Guards**

```bash
tools/redis-key-guard.sh; echo "redis=$?"
tools/goroutine-guard.sh; echo "routine=$?"
```

Expected: both `0`.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-kites/atlas.com/kites/kite/
git commit -m "feat(atlas-kites): placement lifecycle with policy, locking, rollback"
```

---

## Task 10: `atlas-kites` REST resource

Serves the map-entry replay. Path-segment form (not the PRD's flat `?worldId=` form), because that is what every sibling in-map registry uses and what `requests.DrainProvider` on the channel side expects.

**Files:**
- Create: `services/atlas-kites/atlas.com/kites/kite/rest.go`
- Create: `services/atlas-kites/atlas.com/kites/kite/resource.go`
- Create: `services/atlas-kites/atlas.com/kites/kite/resource_paginate_test.go`
- Modify: `services/atlas-kites/atlas.com/kites/main.go` (add `AddRouteInitializer(kite.InitResource(GetServer()))`)

**Interfaces:**
- Produces:
  - `GET /kites/{characterId}` → the owner's kite, `404` if none
  - `GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/kites` → paginated list
  - `kite.RestModel` with `GetName() == "kites"` and JSON fields `characterId`, `name`, `templateId`, `message`, `x`, `y`, `worldId`, `channelId`, `mapId`, `instanceId`, `createdAt`. Task 13's `atlas-channel` `kite.RestModel` must mirror these names exactly.
  - `kite.InitResource(si jsonapi.ServerInformation) server.RouteInitializer`

- [ ] **Step 1: Write the failing pagination test**

Create `services/atlas-kites/atlas.com/kites/kite/resource_paginate_test.go`, modelled on `services/atlas-chalkboards/atlas.com/chalkboards/chalkboard/resource_paginate_test.go`: seed 5 kites for 5 characters all in one field, request `page[size]=2`, and assert three pages of 2/2/1 with **stable ordering by kite id** and no duplicates across pages.

- [ ] **Step 2: Run it to verify it fails**

```bash
cd services/atlas-kites/atlas.com/kites && go test ./kite/ -run Paginate 2>&1 | head -20
```

Expected: FAIL — `undefined: InitResource`.

- [ ] **Step 3: Write the REST model**

Create `services/atlas-kites/atlas.com/kites/kite/rest.go` with `RestModel` (`Id uint32 json:"-"` holding the **kite wire id**, not the character id — design ADR-3 is explicit that chalkboards conflates the two and kites must not), `GetName()`/`GetID()`/`SetID()`, and `Transform(m Model) (RestModel, error)`.

- [ ] **Step 4: Write the resource**

Create `services/atlas-kites/atlas.com/kites/kite/resource.go`, copying the structure of `chalkboard/resource.go:23-108` with two changes: sort by **kite id** (via `GetByCharacterId` on the paged character ids, then sort the resulting models) before `paginate.Slice`, so the page boundary is stable; and register the routes under `/kites`.

- [ ] **Step 5: Run the test**

```bash
cd services/atlas-kites/atlas.com/kites && go test ./kite/ -race -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-kites/atlas.com/kites/
git commit -m "feat(atlas-kites): REST resource for kite lookup and in-map replay"
```

---

## Task 11: `atlas-kites` consumers and service wiring

Two consumers: the kite command topic, and the existing character status topic for the FR-6 teardown. **The instance must be threaded on every transition** — this is the chalkboards bug the design calls out at Q6, and copying it would make instanced maps silently kite-less.

**Files:**
- Create: `services/atlas-kites/atlas.com/kites/kafka/consumer/kite/consumer.go`
- Create: `services/atlas-kites/atlas.com/kites/kafka/consumer/character/consumer.go`
- Create: `services/atlas-kites/atlas.com/kites/kafka/consumer/character/consumer_test.go`
- Modify: `services/atlas-kites/atlas.com/kites/main.go`

**Interfaces:**
- Consumes: `kite.NewProcessor` (Task 9), `character.NewProcessor` (Task 7), the contracts from Task 6.
- Produces: `kiteconsumer.InitConsumers(l)(cmf)(groupId)` / `InitHandlers(l)(rf) error`, and the same pair in the character consumer package. `main.go` calls both.

- [ ] **Step 1: Write the failing teardown test**

Create `services/atlas-kites/atlas.com/kites/kafka/consumer/character/consumer_test.go` asserting the two behaviours the chalkboards consumer gets wrong or omits:

```go
package character

import (
	"atlas-kites/character"
	"atlas-kites/kite"
	kiteMsg "atlas-kites/kafka/message/kite"
	character2 "atlas-kites/kafka/message/character"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setup(t *testing.T) context.Context {
	t.Helper()
	s := miniredis.RunT(t)
	c := goredis.NewClient(&goredis.Options{Addr: s.Addr()})
	kite.InitRegistry(c)
	character.InitRegistry(c)
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

func nullLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

// A map change must destroy the kite against the OLD field, instance included.
// Capturing `of` before the index transition is what keeps the DESTROYED event
// from fanning out to the map the character just walked into.
func TestMapChangedDestroysAgainstOldFieldWithInstance(t *testing.T) {
	ctx := setup(t)
	l := nullLogger()
	rec := newRecorder() // same recording producer.Provider as kite/processor_test.go

	oldInst := uuid.New()
	newInst := uuid.New()
	of := field.NewBuilder(0, 1, 104040000).SetInstance(oldInst).Build()

	if _, err := kite.NewProcessorWithProvider(l, ctx, rec.provider()).
		CreateAndEmit(of, 42, kiteMsg.CreateCommandBody{Name: "Player", TemplateId: 5080000, Message: "hi", X: 320, Y: -140}); err != nil {
		t.Fatalf("seed kite: %v", err)
	}
	rec.reset()

	handleStatusEventMapChanged(l, ctx, character2.StatusEvent[character2.StatusEventMapChangedBody]{
		WorldId:     0,
		CharacterId: 42,
		Type:        character2.EventCharacterStatusTypeMapChanged,
		Body: character2.StatusEventMapChangedBody{
			ChannelId:      1,
			OldMapId:       104040000,
			OldInstance:    oldInst,
			TargetMapId:    104040001,
			TargetInstance: newInst,
		},
	})

	if _, err := kite.NewProcessor(l, ctx).GetByCharacterId(42); err == nil {
		t.Error("kite survived the owner's map change")
	}

	msgs := rec.messages(kiteMsg.EnvEventTopicStatus)
	if len(msgs) != 1 {
		t.Fatalf("emitted %d status events, want 1 (DESTROYED)", len(msgs))
	}
	var ev kiteMsg.StatusEvent[kiteMsg.DestroyedStatusEventBody]
	if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.Type != kiteMsg.EventTopicStatusTypeDestroyed {
		t.Errorf("event type = %s, want DESTROYED", ev.Type)
	}
	if ev.MapId != 104040000 {
		t.Errorf("DESTROYED fanned out to map %d, want the OLD map 104040000", ev.MapId)
	}
	if ev.Instance != oldInst {
		t.Errorf("DESTROYED instance = %s, want the OLD instance %s", ev.Instance, oldInst)
	}
	if ev.Body.Reason != kiteMsg.DestroyReasonOwnerLeft {
		t.Errorf("reason = %s, want OWNER_LEFT", ev.Body.Reason)
	}
}

// The character-in-field index must key on the instance, so a character in an
// instanced copy of a map is not visible in the base copy's set. The chalkboards
// consumer this is modelled on drops the instance here and its instanced maps
// have therefore never replayed.
func TestLoginIndexesWithInstance(t *testing.T) {
	ctx := setup(t)
	l := nullLogger()

	inst := uuid.New()
	handleStatusEventLogin(l, ctx, character2.StatusEvent[character2.StatusEventLoginBody]{
		WorldId:     0,
		CharacterId: 42,
		Type:        character2.EventCharacterStatusTypeLogin,
		Body:        character2.StatusEventLoginBody{ChannelId: 1, MapId: 104040000, Instance: inst},
	})

	instanced := field.NewBuilder(0, 1, 104040000).SetInstance(inst).Build()
	base := field.NewBuilder(0, 1, 104040000).SetInstance(uuid.Nil).Build()

	got, err := character.NewProcessor(l, ctx).GetCharactersInMap(instanced)
	if err != nil {
		t.Fatalf("GetCharactersInMap(instanced): %v", err)
	}
	if len(got) != 1 || got[0] != 42 {
		t.Errorf("instanced field = %v, want [42]", got)
	}

	got, err = character.NewProcessor(l, ctx).GetCharactersInMap(base)
	if err != nil {
		t.Fatalf("GetCharactersInMap(base): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("base field saw the instanced character: %v", got)
	}
}
```

The `recorder` helper is the one written in Task 9's `kite/processor_test.go`. Promote it to a small shared test helper in the `kite` package (exported as `kite.NewTestRecorder()`) or duplicate the ~25 lines here — do **not** create a `*_testhelpers.go` file, which this project bans. Add `reset()` and `messages(topic string) []kafka.Message` to it while promoting; Task 9's tests keep using `count()`.

- [ ] **Step 2: Run it to verify it fails**

```bash
cd services/atlas-kites/atlas.com/kites && go test ./kafka/consumer/character/ 2>&1 | head -20
```

Expected: FAIL — the package has no handlers yet.

- [ ] **Step 3: Write the character status consumer**

Create `services/atlas-kites/atlas.com/kites/kafka/consumer/character/consumer.go` on the shape of `services/atlas-chalkboards/atlas.com/chalkboards/kafka/consumer/character/consumer.go`, with the instance threaded through every field construction:

```go
func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLoginBody]) {
	if e.Type != character2.EventCharacterStatusTypeLogin {
		return
	}
	// SetInstance is the difference from the chalkboards consumer this is
	// modelled on: it builds its key without the instance while its resource
	// reads with one, so instanced maps never replay. The status events have
	// always carried the instance.
	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
	character.NewProcessor(l, ctx).Enter(f, e.CharacterId)
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLogoutBody]) {
	if e.Type != character2.EventCharacterStatusTypeLogout {
		return
	}
	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
	character.NewProcessor(l, ctx).Exit(f, e.CharacterId)
	if _, err := kite.NewProcessor(l, ctx).DestroyAndEmit(e.CharacterId, kiteMsg.DestroyReasonOwnerLoggedOut); err != nil {
		l.WithError(err).Debugf("Unable to destroy kite for logged-out character [%d].", e.CharacterId)
	}
}

func handleStatusEventMapChanged(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventMapChangedBody]) {
	if e.Type != character2.EventCharacterStatusTypeMapChanged {
		return
	}
	// `of` is captured BEFORE the index transition: the DESTROYED event is
	// keyed and fanned out on the kite's own field, and the kite is on the map
	// the character is leaving.
	of := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.OldMapId).SetInstance(e.Body.OldInstance).Build()
	nf := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.TargetMapId).SetInstance(e.Body.TargetInstance).Build()
	character.NewProcessor(l, ctx).TransitionMap(of, nf, e.CharacterId)
	if _, err := kite.NewProcessor(l, ctx).DestroyAndEmit(e.CharacterId, kiteMsg.DestroyReasonOwnerLeft); err != nil {
		l.WithError(err).Debugf("Unable to destroy kite for departing character [%d].", e.CharacterId)
	}
}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.ChangeChannelEventLoginBody]) {
	if e.Type != character2.EventCharacterStatusTypeChannelChanged {
		return
	}
	of := field.NewBuilder(e.WorldId, e.Body.OldChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
	nf := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
	character.NewProcessor(l, ctx).TransitionChannel(of, nf, e.CharacterId)
	if _, err := kite.NewProcessor(l, ctx).DestroyAndEmit(e.CharacterId, kiteMsg.DestroyReasonOwnerLeft); err != nil {
		l.WithError(err).Debugf("Unable to destroy kite for channel-changing character [%d].", e.CharacterId)
	}
}
```

`DestroyAndEmit` on a character with no kite returns a not-found error; that is the common case and is logged at `Debug`, never `Error`.

- [ ] **Step 4: Write the kite command consumer**

Create `services/atlas-kites/atlas.com/kites/kafka/consumer/kite/consumer.go` on the shape of `services/atlas-chalkboards/.../kafka/consumer/chalkboard/consumer.go`: `InitConsumers` registers `consumer2.NewConfig(l)("kite_command")(kiteMsg.EnvCommandTopic)(consumerGroupId)` with the span + tenant header parsers; `InitHandlers` registers `handleCreateCommand` and `handleDestroyCommand`. Each rebuilds the field **with** `SetInstance(c.Instance)` and delegates to the processor.

- [ ] **Step 5: Wire `main.go`**

Fill in the `main.go` stub from Task 6: `kite.InitRegistry(rc)` and `character2.InitRegistry(rc)` after `atlas.Connect(l)`; `character.InitConsumers` and `kiteconsumer.InitConsumers`; both `InitHandlers` calls guarded by `l.Fatal` on error; the `kite.InitResource(GetServer())` route initializer from Task 10.

- [ ] **Step 6: Run the full module gate**

```bash
cd services/atlas-kites/atlas.com/kites && go test -race ./... && go vet ./... && go build ./...
```

Expected: all clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-kites/
git commit -m "feat(atlas-kites): kafka consumers and service wiring"
```

---

## Task 12: Register `atlas-kites` in every hand-maintained list

None of these lists is derived from another and several fail *silently*. Work
[`docs/adding-a-new-service.md`](../../adding-a-new-service.md) §1–§5 in full.
**§6 Databases is skipped entirely** — `atlas-kites` is Redis-only, exactly like
`atlas-chalkboards`, which appears in `services.json:65` with no database and no
`tools/db-bootstrap.sh` entry. §7 is N/A (not socket-exposing). This resolves the
PRD §7 "confirm during design" item.

**Files:**
- Modify: `.github/config/services.json`
- Modify: `docker-bake.hcl` (the `go_services` list — hand-synced with services.json)
- Create: `deploy/k8s/base/atlas-kites.yaml`
- Modify: `deploy/k8s/base/kustomization.yaml`
- Modify: `deploy/k8s/base/env-configmap.yaml`
- Modify: `deploy/k8s/overlays/main/kustomization.yaml`
- Modify: `deploy/k8s/overlays/main/patches/atlas-env-env.yaml`
- Modify: `deploy/k8s/overlays/pr/kustomization.yaml`
- Modify: `deploy/shared/routes.conf`
- Modify: `deploy/k8s/ingress.yaml` (regenerated, not hand-edited)

(`go.work` was already updated in Task 6.)

- [ ] **Step 1: Build & CI**

Add to `.github/config/services.json` `services[]`, matching the chalkboards entry shape:

```json
    {
      "name": "atlas-kites",
      "type": "go-service",
      "path": "services/atlas-kites",
      "module_path": "services/atlas-kites/atlas.com/kites",
      "docker_image": "ghcr.io/chronicle20/atlas-kites/atlas-kites",
      "docker_context": "."
    },
```

Add `"atlas-kites",` to the `go_services` list in `docker-bake.hcl`, keeping the existing ordering.

- [ ] **Step 2: Kubernetes base**

Create `deploy/k8s/base/atlas-kites.yaml` by copying `deploy/k8s/base/atlas-chalkboards.yaml` and substituting `atlas-kites` / container name `kites`. No `namespace:` (overlays set it), no `DB_NAME` (Redis-only), `replicas: 2`.

Add `- atlas-kites.yaml` to `resources:` in `deploy/k8s/base/kustomization.yaml`, alphabetically placed.

Add the two new topic keys to `deploy/k8s/base/env-configmap.yaml`, each at its alphabetical position among the `COMMAND_TOPIC_*` and `EVENT_TOPIC_*` blocks:

```yaml
  COMMAND_TOPIC_KITE: "COMMAND_TOPIC_KITE"
  EVENT_TOPIC_KITE_STATUS: "EVENT_TOPIC_KITE_STATUS"
```

- [ ] **Step 3: Main overlay**

In `deploy/k8s/overlays/main/kustomization.yaml`:

- add the image pin to `images:` (the bump workflow only rewrites entries **already present** — a missing entry means `:latest` forever, silently). Use the same `newTag:` value the neighbouring entries currently carry:

Read the current fleet tag off an existing entry first — every service in the
block carries the same `main-<sha>`:

```bash
grep -A 1 'atlas-chalkboards/atlas-chalkboards' deploy/k8s/overlays/main/kustomization.yaml
```

Then add the entry with that exact `newTag` value:

```yaml
  - name: ghcr.io/chronicle20/atlas-kites/atlas-kites
    newTag: main-<the sha printed above>
```

Confirm the tag exists on ghcr before committing, or the deployment pulls a
nonexistent image:

```bash
docker manifest inspect ghcr.io/chronicle20/atlas-kites/atlas-kites:main-<sha> > /dev/null; echo "exists=$?"
```

A first-time service will **not** have that tag until the main-publish workflow
runs once. If `exists` is non-zero, still pin the entry to the fleet tag — the
point is that the entry exists so the bump workflow can rewrite it — and note in
the PR that the first main deploy lands only after the publish job builds it.

- add both topic literals to the `configMapGenerator` (`behavior: replace` — a base key not re-listed here is **absent** on main):

```yaml
      - COMMAND_TOPIC_KITE=COMMAND_TOPIC_KITE-main
      - EVENT_TOPIC_KITE_STATUS=EVENT_TOPIC_KITE_STATUS-main
```

Do **not** add `KAFKA_CONSUMER_GROUP` on main (see the comment at the top of that file).

In `deploy/k8s/overlays/main/patches/atlas-env-env.yaml`, append an `ATLAS_ENV: "main"` patch document targeting deployment `atlas-kites` / container `kites`, copying the chalkboards document at `:96-101`.

- [ ] **Step 4: PR overlay**

In `deploy/k8s/overlays/pr/kustomization.yaml`, add the `images:` entry (same shape as Step 3). Do **not** add an `ATLAS_DB_NAMES` entry — `atlas-kites` has no database.

Regenerate the three generator-owned pieces rather than hand-editing them:

```bash
deploy/k8s/overlays/pr/scripts/gen-topic-config.sh
deploy/k8s/overlays/pr/scripts/gen-db-name-suffix.sh
deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh
```

Paste `gen-topic-config.sh`'s output into the PR overlay's `atlas-env` generator block; the other two write their files directly. Editing these by hand works until the next generator run silently reverts you.

- [ ] **Step 5: Ingress**

Add two nginx location blocks to `deploy/shared/routes.conf`, alphabetically placed, mirroring the chalkboards pair at `:170-177`:

```
location ~ ^/api/worlds/[^/]+/channels/[^/]+/maps/[^/]+/instances/[^/]+/kites(/.*)?$ {
  set $u "atlas-kites:8080";
  ...
}

location ~ ^/api/kites(/.*)?$ {
  set $u "atlas-kites:8080";
  ...
}
```

Copy the body of the chalkboards blocks verbatim apart from the upstream. Then regenerate and commit both files:

```bash
./deploy/scripts/sync-k8s-ingress-routes.sh
```

- [ ] **Step 6: Run the registration guard**

```bash
tools/service-registration-guard.sh; echo "exit=$?"
```

Expected: `exit=0`.

- [ ] **Step 7: Verify what the guard cannot check**

The guard enforces *parity* of keys already present — not that the correct *new* topic keys exist. Check by hand:

```bash
kubectl kustomize deploy/k8s/overlays/main | grep -E 'COMMAND_TOPIC_KITE|EVENT_TOPIC_KITE_STATUS'
#   expect: COMMAND_TOPIC_KITE: COMMAND_TOPIC_KITE-main
#           EVENT_TOPIC_KITE_STATUS: EVENT_TOPIC_KITE_STATUS-main
kubectl kustomize deploy/k8s/overlays/main | grep -A 12 'name: atlas-kites$'
#   expect: ATLAS_ENV=main, image pinned to main-<sha> (NOT :latest), no DB_NAME
kubectl kustomize deploy/k8s/overlays/pr > /dev/null; echo "pr-render=$?"
```

A missing topic var does not crash — `libs/atlas-kafka/topic/provider.go` falls back to the bare token with only a warn log, and the two sides then talk on different topics.

- [ ] **Step 8: Build the image**

```bash
docker buildx bake atlas-kites
```

Expected: success. `go build` against `go.work` cannot catch a missing `COPY libs/...` in the shared Dockerfile — only the bake can. `atlas-kites` introduces no new shared lib, so no `Dockerfile` edit is expected; if the bake fails on a missing lib, that is the signal.

- [ ] **Step 9: Commit**

```bash
git add .github/config/services.json docker-bake.hcl deploy/
git commit -m "chore(atlas-kites): register service in build, k8s, and ingress"
```

---

## Task 13: `atlas-channel` kite client package

Replaces the dead `channel/kite/model.go` — zero importers (`grep -rn "channel/kite"` → no hits), and its `ft`/`Type()` field does not exist on the wire. The package name stays; the contents are rewritten.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kite/model.go` (rewrite)
- Create: `services/atlas-channel/atlas.com/channel/kite/builder.go`
- Create: `services/atlas-channel/atlas.com/channel/kite/rest.go`
- Create: `services/atlas-channel/atlas.com/channel/kite/requests.go`
- Create: `services/atlas-channel/atlas.com/channel/kite/processor.go`
- Create: `services/atlas-channel/atlas.com/channel/kite/producer.go`
- Create: `services/atlas-channel/atlas.com/channel/kite/processor_drain_test.go`
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/kite/kafka.go`
- Modify: `services/atlas-channel/docs/domain.md:793-803`

**Interfaces:**
- Consumes: the REST contract from Task 10 and the Kafka contract from Task 6.
- Produces:
  ```go
  func kite.NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor
  type Processor interface {
      InMapModelProvider(f field.Model) model.Provider[[]Model]
      ForEachInMap(f field.Model, o model.Operator[Model]) error
      AttemptUse(f field.Model, characterId uint32, name string, templateId uint32, message string, x int16, y int16) error
  }
  // Model getters: Id() uint32, CharacterId() uint32, Name() string,
  //                TemplateId() uint32, Message() string, X() int16, Y() int16
  ```
  Task 14 calls `AttemptUse`; Task 16 calls `ForEachInMap`.

- [ ] **Step 1: Mirror the Kafka contract**

Create `services/atlas-channel/atlas.com/channel/kafka/message/kite/kafka.go` as a **field-for-field copy** of `services/atlas-kites/atlas.com/kites/kafka/message/kite/kafka.go` (Task 6), with `package kite`. This duplication is the established convention across the repo (`kafka/message/chalkboard` exists on both sides); the two files must not drift.

- [ ] **Step 2: Write the failing drain test**

Create `services/atlas-channel/atlas.com/channel/kite/processor_drain_test.go`, modelled directly on `services/atlas-channel/atlas.com/channel/chalkboard/processor_drain_test.go`. It must assert two things:

1. `InMapModelProvider` drains **every page**, not just the first (serve 3 pages from `httptest` and assert all items arrive).
2. The requested path contains the `/instances/{instanceId}/` segment. This is the bug `chalkboard/requests.go:9-17` documents having shipped without — every request built from the segment-less format string 404s against the real route, so that consumer likely never worked. Assert the exact path.

```go
func TestInMapModelProviderRequestsInstanceScopedPath(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	t.Setenv("BASE_SERVICE_URL", srv.URL+"/api/")

	inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	f := field.NewBuilder(0, 1, 104040000).SetInstance(inst).Build()

	if _, err := NewProcessor(testLogger(t), testContext(t)).InMapModelProvider(f)(); err != nil {
		t.Fatalf("InMapModelProvider: %v", err)
	}
	want := "/api/worlds/0/channels/1/maps/104040000/instances/11111111-2222-3333-4444-555555555555/kites"
	if seen != want {
		t.Errorf("requested %q, want %q", seen, want)
	}
}
```

- [ ] **Step 3: Run to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./kite/ 2>&1 | head -20
```

Expected: FAIL — `undefined: NewProcessor`.

- [ ] **Step 4: Rewrite the model and add the builder**

Replace the whole of `services/atlas-channel/atlas.com/channel/kite/model.go`:

```go
package kite

// Model is one kite (cash category 508 message box) as atlas-channel sees it.
// This replaces the previous model-only scaffold, whose `ft`/Type() field does
// not exist on the wire: FieldKiteSpawn's sixth int16 is the spawn Y, and the
// banner's appearance comes from templateId (CItemInfo::GetItemProp).
type Model struct {
	id          uint32
	characterId uint32
	name        string
	templateId  uint32
	message     string
	x           int16
	y           int16
}

func (m Model) Id() uint32          { return m.id }
func (m Model) CharacterId() uint32 { return m.characterId }
func (m Model) Name() string        { return m.name }
func (m Model) TemplateId() uint32  { return m.templateId }
func (m Model) Message() string     { return m.message }
func (m Model) X() int16            { return m.x }
func (m Model) Y() int16            { return m.y }
```

Add `builder.go` with `NewBuilder(id uint32, characterId uint32) *Builder` and `SetName`/`SetTemplateId`/`SetMessage`/`SetPosition`/`Build`, so tests construct models through the Builder pattern rather than a test-only constructor.

- [ ] **Step 5: Write rest.go, requests.go, processor.go, producer.go**

- `rest.go`: `RestModel` mirroring Task 10's field names exactly, plus `Extract(rm RestModel) (Model, error)`.
- `requests.go`:

```go
// Resource carries the /instances/{instanceId} segment from day one — the
// chalkboard sibling shipped without it and 404'd against the real route for
// its entire life (chalkboard/requests.go:9-17).
const Resource = "worlds/%d/channels/%d/maps/%d/instances/%s/kites"

func getBaseRequest() string { return requests.RootUrl("KITES") }

func inMapUrl(f field.Model) string {
	return fmt.Sprintf(getBaseRequest()+Resource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}
```

- `processor.go`: `InMapModelProvider` via `requests.DrainProvider[RestModel, Model](p.l, p.ctx)(inMapUrl(f), 250, Extract, model.Filters[Model]())`; `ForEachInMap` via `model.ForEachSlice(p.InMapModelProvider(f), o, model.ParallelExecute())`; `AttemptUse` producing the `CREATE` command.
- `producer.go`: `CreateCommandProvider` keyed `producer.CreateKey(int(characterId))` — per-character ordering, matching `chalkboard/producer.go:14`.

- [ ] **Step 6: Run the drain tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./kite/ -race -v
```

Expected: PASS on both.

- [ ] **Step 7: Rewrite the domain doc entry**

Replace `services/atlas-channel/docs/domain.md:793-803` — the "Model-only domain" Kite section — with the real package:

```markdown
## Kite

### Responsibility
Client-side view of kites (cash item category 508 message boxes) owned by
atlas-kites: drains the in-map list for map-entry replay and issues placement
commands.

### Core Models
- `Model` - id (uint32, the wire id), characterId (uint32), name (string),
  templateId (uint32), message (string), x (int16), y (int16)

### Processors
- `Processor` - InMapModelProvider(f) / ForEachInMap(f, o) drain the paginated
  `.../maps/{mapId}/instances/{instanceId}/kites` list (KITES service);
  AttemptUse(...) emits COMMAND_TOPIC_KITE CREATE keyed on characterId
```

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kite/ \
        services/atlas-channel/atlas.com/channel/kafka/message/kite/ \
        services/atlas-channel/docs/domain.md
git commit -m "feat(atlas-channel): kite client package replacing dead scaffold"
```

---

## Task 14: `atlas-channel` type-18 handler arm

The arm that has never existed: `GetCashSlotItemType` already maps classification 508 → `CashSlotItemType(18)` (`character_cash_item_use.go:810-812`), but no arm consumes it, so a kite use falls through to the terminal `l.Warnf` at `:611` and nothing happens.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` (add the arm after the chalkboard arm at `:93-98`; add `CashSlotItemTypeKite` to the const block at `:617-648`)
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_kite_test.go`

**Interfaces:**
- Consumes: `cashsb.NewItemUseKite` (Task 3), `kite.NewProcessor(...).AttemptUse` (Task 13), `character2.NewProcessor(...).GetById` (existing).

- [ ] **Step 1: Write the failing handler test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_kite_test.go`. Inject the `cashItemInSlotFunc` package var (the established test seam, per its comment at `:653-655`) and a recording producer, then assert that a type-18 use emits **exactly one** `CREATE` command whose `x`, `y`, and `name` come from the character record and **not** from the packet (the packet does not carry them):

```go
package handler

import (
	kiteMsg "atlas-channel/kafka/message/kite"
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"

	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// kiteUseRequest builds the serverbound USE_CASH_ITEM payload for a category-508
// item: the common ItemUse prefix followed by the type-18 sub-body. GMS v83
// trails updateTime, so updateTimeFirst is false on both halves.
func kiteUseRequest(t *testing.T, l logrus.FieldLogger, ctx context.Context, source int16, itemId uint32, message string) *request.Reader {
	t.Helper()
	prefix := cashsb.ItemUse{}
	// ItemUse's fields are private; encode through the same context the handler
	// decodes with so the leading/trailing updateTime gate matches.
	body := append(encodeItemUsePrefix(t, l, ctx, source, itemId),
		(&cashsb.ItemUseKite{}).Encode(l, ctx)(nil)...)
	_ = prefix
	_ = message
	return request.NewReader(body)
}

func TestKiteUseEmitsCreateWithServerSidePosition(t *testing.T) {
	l, ctx := testChannelContext(t, "GMS", 83, 1)

	// The ownership check that gates the whole handler. Injecting the package
	// var is the established seam (see cashItemInSlotFunc's own comment).
	orig := cashItemInSlotFunc
	cashItemInSlotFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32, _ int16) (uint32, error) {
		return 5080000, nil
	}
	t.Cleanup(func() { cashItemInSlotFunc = orig })

	// Character state is the ONLY source of position and name — the packet
	// carries neither.
	s := seedSession(t, ctx, 42, "Player", 320, -140)
	rec := newCommandRecorder(t)

	r := kiteUseRequest(t, l, ctx, 1, 5080000, "congrats!")
	CharacterCashItemUseHandleFunc(l, ctx, rec.WriterProducer())(s, r, nil)

	msgs := rec.messages(kiteMsg.EnvCommandTopic)
	if len(msgs) != 1 {
		t.Fatalf("emitted %d kite commands, want exactly 1", len(msgs))
	}
	var cmd kiteMsg.Command[kiteMsg.CreateCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != kiteMsg.CommandKiteCreate {
		t.Errorf("type = %s, want CREATE", cmd.Type)
	}
	if cmd.CharacterId != 42 {
		t.Errorf("characterId = %d, want 42", cmd.CharacterId)
	}
	if cmd.Body.X != 320 || cmd.Body.Y != -140 {
		t.Errorf("position = (%d,%d), want the character's (320,-140)", cmd.Body.X, cmd.Body.Y)
	}
	if cmd.Body.Name != "Player" {
		t.Errorf("name = %q, want the character's %q", cmd.Body.Name, "Player")
	}
	if cmd.Body.Message != "congrats!" {
		t.Errorf("message = %q, want %q", cmd.Body.Message, "congrats!")
	}
	if cmd.Body.TemplateId != 5080000 {
		t.Errorf("templateId = %d, want 5080000", cmd.Body.TemplateId)
	}
}

// FR-4.1: the kite item is deliberately NOT consumed. Placement is gated by the
// per-character cap in atlas-kites instead, so the arm must issue no saga and
// no inventory mutation.
func TestKiteUseDoesNotConsumeTheItem(t *testing.T) {
	l, ctx := testChannelContext(t, "GMS", 83, 1)

	orig := cashItemInSlotFunc
	cashItemInSlotFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32, _ int16) (uint32, error) {
		return 5080000, nil
	}
	t.Cleanup(func() { cashItemInSlotFunc = orig })

	s := seedSession(t, ctx, 42, "Player", 320, -140)
	rec := newCommandRecorder(t)

	r := kiteUseRequest(t, l, ctx, 1, 5080000, "congrats!")
	CharacterCashItemUseHandleFunc(l, ctx, rec.WriterProducer())(s, r, nil)

	for _, topic := range rec.topics() {
		if topic != kiteMsg.EnvCommandTopic {
			t.Errorf("kite use emitted on unexpected topic %q — no saga or inventory command may be issued", topic)
		}
	}
}
```

`testChannelContext`, `seedSession`, `newCommandRecorder`, and `encodeItemUsePrefix` are the fixtures the neighbouring handler tests in this package already use — read `services/atlas-channel/atlas.com/channel/socket/handler/` for the existing names and reuse them rather than adding parallel helpers. If a needed fixture genuinely does not exist, build the session through the project Builder pattern; do **not** add a `*_testhelpers.go` file.

- [ ] **Step 2: Run to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run Kite 2>&1 | head -20
```

Expected: FAIL — no `CREATE` command emitted (the use currently falls through to the warn).

- [ ] **Step 3: Add the type constant**

In the `CashSlotItemType` const block (`character_cash_item_use.go:617-648`), add beside `CashSlotItemTypeChalkboard`:

```go
	CashSlotItemTypeKite          = CashSlotItemType(18)
```

- [ ] **Step 4: Add the arm**

Insert immediately after the `CashSlotItemTypeChalkboard` arm (`:93-98`):

```go
		if it == CashSlotItemTypeKite {
			sp := cashsb.NewItemUseKite(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)

			// The sub-body is the message alone — the client sends no
			// coordinates for a kite (case-18 arm of
			// CWvsContext::SendConsumeCashItemUseRequest performs exactly one
			// EncodeStr). Position and owner name therefore come from
			// server-side character state, the same source
			// skill/handler/mysticdoor uses.
			c, err := character2.NewProcessor(l, ctx).GetById()(s.CharacterId())
			if err != nil {
				l.WithError(err).Debugf("Unable to resolve character [%d] for kite placement.", s.CharacterId())
				return
			}

			// No item is consumed (FR-4.1): no saga.DestroyAsset step and no
			// inventory mutation, so this is a direct command. Placement is
			// gated by the per-character cap in atlas-kites instead.
			//
			// No EnableActions either: the client's kite dialog is modal
			// (CDialog::DoModal @0x9ed0d9) and unlocks itself, and the sibling
			// chalkboard use arm sends none. Unlocking here would only widen
			// the client's duplicate-request gate.
			if err = kite.NewProcessor(l, ctx).AttemptUse(s.Field(), s.CharacterId(), c.Name(), uint32(itemId), sp.Message(), c.X(), c.Y()); err != nil {
				l.WithError(err).Debugf("Unable to request kite placement for character [%d].", s.CharacterId())
			}
			return
		}
```

Add `"atlas-channel/kite"` to the import block. Ownership is already verified before this point by the shared `cashItemInSlotFunc` check at `:53-57` that gates the whole handler (FR-4.2).

- [ ] **Step 5: Run the tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -race -run Kite -v
```

Expected: PASS on both.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/
git commit -m "feat(atlas-channel): handle cash slot item type 18 (kite)"
```

---

## Task 15: `atlas-channel` kite status consumer

`CREATION_FAILED` is deliberately **not** a map broadcast — it targets the requesting character only, which is why the event body carries the addressee.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/kite/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (`InitConsumers` near `:236`; `InitHandlers` registration near `:504`)

**Interfaces:**
- Consumes: `kiteMsg.EnvEventTopicStatus` and the three bodies (Task 13's mirror), `fieldcb.KiteSpawnWriter`/`KiteDestroyWriter`/`KiteErrorWriter`, `clientbound.KiteDestroyAnimated` (Task 2).
- Produces: `kiteconsumer.InitConsumers(l)(cmf)(groupId)` and `kiteconsumer.InitHandlers(l)(sc)(wp)(rh)`.

- [ ] **Step 1: Write the consumer**

Create `services/atlas-channel/atlas.com/channel/kafka/consumer/kite/consumer.go`, structured exactly like `kafka/consumer/chalkboard/consumer.go` — `consumer.SetStartOffset(kafka.LastOffset)`, and an `sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId)` guard as the first thing in every handler.

```go
func handleCreatedEvent(sc server.Model, wp writer.Producer) message.Handler[kiteMsg.StatusEvent[kiteMsg.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e kiteMsg.StatusEvent[kiteMsg.CreatedStatusEventBody]) {
		if e.Type != kiteMsg.EventTopicStatusTypeCreated {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance), func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(fieldcb.KiteSpawnWriter)(
				fieldcb.NewKiteSpawn(e.Body.KiteId, e.Body.TemplateId, e.Body.Message, e.Body.Name, e.Body.X, e.Body.Y).Encode)(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn kite [%d] for character [%d].", e.Body.KiteId, e.CharacterId)
		}
	}
}

func handleDestroyedEvent(sc server.Model, wp writer.Producer) message.Handler[kiteMsg.StatusEvent[kiteMsg.DestroyedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e kiteMsg.StatusEvent[kiteMsg.DestroyedStatusEventBody]) {
		if e.Type != kiteMsg.EventTopicStatusTypeDestroyed {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		// KiteDestroyAnimated (0) plays the one-shot despawn animation. The
		// byte is a suppress-animation flag, not a selector; both destroy
		// reasons are the same class of event and want the animation.
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance), func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(fieldcb.KiteDestroyWriter)(
				fieldcb.NewKiteDestroy(e.Body.KiteId, fieldcb.KiteDestroyAnimated).Encode)(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to destroy kite [%d].", e.Body.KiteId)
		}
	}
}

func handleCreationFailedEvent(sc server.Model, wp writer.Producer) message.Handler[kiteMsg.StatusEvent[kiteMsg.CreationFailedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e kiteMsg.StatusEvent[kiteMsg.CreationFailedStatusEventBody]) {
		if e.Type != kiteMsg.EventTopicStatusTypeCreationFailed {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		// Targeted, NOT a map broadcast. FieldKiteError has an empty body, so
		// the client shows a generic failure and the reason survives only here.
		l.Infof("Kite placement refused for character [%d]: [%s].", e.CharacterId, e.Body.Reason)
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(fieldcb.KiteErrorWriter)(fieldcb.NewKiteError().Encode)(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to notify character [%d] of kite failure.", e.CharacterId)
		}
	}
}
```

- [ ] **Step 2: Wire it into `main.go`**

Add `kite.InitConsumers(l)(cmf)(consumerGroupId)` beside the chalkboard call at `:236`, and the handler registration beside the chalkboard one at `:504`:

```go
		if err := register(kite.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return nil, err
		}
```

Import the package as `kiteconsumer "atlas-channel/kafka/consumer/kite"` to avoid colliding with the `atlas-channel/kite` client package imported by the handler.

- [ ] **Step 3: Verify the writers are already registered**

```bash
sed -n '718,730p' services/atlas-channel/atlas.com/channel/main.go
```

Expected: `fieldcb.KiteSpawnWriter`, `fieldcb.KiteErrorWriter`, `fieldcb.KiteDestroyWriter` already present at `:724-726`. FR-7.6 is verify-only — **make no change here**.

- [ ] **Step 4: Build**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...
```

Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/kite/ \
        services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(atlas-channel): render kite spawn, destroy, and error events"
```

---

## Task 16: `atlas-channel` map-entry replay

The other half of the feature: a character entering a map must see every kite already hanging there.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` (a `routine.Go` block beside the chalkboard pass at `:264-268`; a `spawnKitesForSession` helper beside `spawnChalkboardsForSession` at `:800-810`)

**Interfaces:**
- Consumes: `kite.NewProcessor(...).ForEachInMap` (Task 13), `fieldcb.NewKiteSpawn` (Task 1).

- [ ] **Step 1: Add the replay pass**

Insert after the chalkboard block (`:264-268`), keeping it in its own `routine.Go` so a failure cannot affect the rest of the map-enter fan-out:

```go
		routine.Go(l, ctx, func(_ context.Context) {
			if err := kite.NewProcessor(l, ctx).ForEachInMap(f, spawnKitesForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn kites for character [%d].", s.CharacterId())
			}
		})
```

- [ ] **Step 2: Add the operator**

Insert beside `spawnChalkboardsForSession` (`:800-810`):

```go
// spawnKitesForSession renders one already-placed kite to a character entering
// the map. ForEachInMap runs under model.ParallelExecute(), so this operator
// must hold no shared mutable state: it closes over only s and wp and builds a
// fresh KiteSpawn per model.
func spawnKitesForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[kite.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[kite.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[kite.Model] {
			return func(s session.Model) model.Operator[kite.Model] {
				return func(k kite.Model) error {
					return session.Announce(l)(ctx)(wp)(fieldcb.KiteSpawnWriter)(
						fieldcb.NewKiteSpawn(k.Id(), k.TemplateId(), k.Message(), k.Name(), k.X(), k.Y()).Encode)(s)
				}
			}
		}
	}
}
```

**Review point, not an assumption:** confirm by reading the final operator that it captures nothing mutable. Parallel `ForEachInMap` operators sharing state is a known hazard on this codebase.

- [ ] **Step 3: Build and run the channel module gate**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./...
```

Expected: all clean.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go
git commit -m "feat(atlas-channel): replay kites to characters entering a map"
```

---

## Task 17: Full verification sweep

Nothing here is optional and nothing here may be reported from memory. Run each command, read its actual output, and paste the real result into the completion report. A skipped gate reported as passing is a false "verified".

- [ ] **Step 1: Per-module Go gates**

```bash
for m in libs/atlas-packet services/atlas-kites/atlas.com/kites \
         services/atlas-channel/atlas.com/channel services/atlas-tenants/atlas.com/tenants; do
  echo "=== $m"
  ( cd "$m" && go build ./... && go vet ./... && go test -race ./... ) || echo "FAILED: $m"
done
```

Expected: no `FAILED:` lines.

- [ ] **Step 2: Docker bakes for every service whose `go.mod` was touched**

```bash
docker buildx bake atlas-kites
docker buildx bake atlas-channel
docker buildx bake atlas-tenants
```

Expected: all three succeed. `go build` against the workspace will **not** catch a missing `COPY libs/...` in the shared root `Dockerfile` — only the bake will, and each CI round-trip to discover it wastes a cycle.

- [ ] **Step 3: Repo guards**

```bash
tools/service-registration-guard.sh;      echo "svc-reg=$?"
tools/redis-key-guard.sh;                 echo "redis=$?"
tools/goroutine-guard.sh;                 echo "routine=$?"
tools/template-opcode-order-guard.sh;     echo "tpl-order=$?"
tools/template-duplicate-binding-guard.sh; echo "tpl-dup=$?"
tools/template-movement-types-guard.sh;   echo "tpl-move=$?"
tools/buff-duration-guard.sh;             echo "buff=$?"
tools/skill-job-id-guard.sh;              echo "skill-job=$?"
```

Expected: every one `=0`.

- [ ] **Step 4: Lint**

```bash
tools/lint.sh            # fix mode — rewrites in place
tools/lint.sh --check; echo "lint=$?"
```

Expected: `lint=0`. This needs nvm 22 on PATH or the atlas-ui leg false-fails; if another worktree is running golangci-lint concurrently you may hit lock contention — rerun rather than treating it as a finding.

- [ ] **Step 5: Packet matrix is unchanged**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check; echo "matrix=$?"
git diff --exit-code docs/packets/audits/status.json docs/packets/audits/STATUS.md; echo "diff=$?"
grep -n 'SPAWN_KITE ' docs/packets/audits/STATUS.md
```

Expected: `matrix=0`, `diff=0`, and the `SPAWN_KITE` row still reading `✅ 🟡ᶠ 🟡ᶠ 🟡ᶠ ✅ ✅ ✅ 🟡ᶠ ✅ ✅`. Both renames are byte-neutral and change no decompile hash, so no evidence re-pin is warranted and any status movement means something else shifted.

- [ ] **Step 6: Acceptance-criteria spot checks**

```bash
# Kite bindings present in exactly ten templates, absent from v12.
grep -ril kite services/atlas-configurations/seed-data/templates/ | sort
#   expect: the ten non-v12 templates, and NOT template_gms_12_1.json

# The dead channel model is gone (rewritten, not resurrected).
grep -n 'ft \|func (m Model) Type()' services/atlas-channel/atlas.com/channel/kite/model.go
#   expect: no output

# The domain doc no longer claims a model-only domain.
grep -n 'Model-only domain' services/atlas-channel/docs/domain.md
#   expect: no output in the Kite section

# atlas-kites is registered everywhere the guard checks plus ingress.
grep -rn atlas-kites .github/config/services.json docker-bake.hcl go.work \
        deploy/k8s/base/kustomization.yaml deploy/shared/routes.conf | wc -l
#   expect: >= 5
```

- [ ] **Step 7: Code review before PR**

Run `superpowers:requesting-code-review`. Go files changed in four modules, so it dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer`; no atlas-ui TypeScript changed, so the frontend reviewer is not needed. Pin the reviewer subagents to Sonnet. Findings land in `docs/tasks/task-211-kite-cash-item/audit.md`. Do not open a PR before this step — it is not optional even when the plan looks complete.

- [ ] **Step 8: Commit any fixes and report**

```bash
git add -A && git commit -m "chore(task-211): verification sweep fixes"
git log --oneline main..HEAD
```

Report the actual output of Steps 1–6 — real exit codes, real matrix row — not a summary from memory.

---

## Spec coverage

Every PRD functional requirement and design decision maps to a task:

| Requirement | Task |
|---|---|
| FR-1.1/1.2/1.3 serverbound decode (design Q3) | 3 |
| FR-2.1/2.2/2.3 `kiteType`→`y` + audit JSON + matrix | 1 |
| FR-3.1 domain model · FR-3.2 Redis registry · FR-3.3 id counter | 7 |
| FR-3.4 processor · FR-3.5 no CREATED on refusal | 9 |
| FR-4.1 not consumed · FR-4.2 ownership check | 14 |
| FR-5.1 per-map cap · FR-5.2 one-per-character · FR-5.3 map policy · FR-5.4 enforced in atlas-kites | 9 (policy model in 8) |
| FR-6.1 owner leaves · FR-6.2 logout · FR-6.3 no TTL (nothing added) | 11 |
| FR-6.4 destroy animation (design Q2) | 2, 15 |
| FR-7.1 handler arm | 14 |
| FR-7.2 channel kite package · FR-7.3 dead model + domain doc | 13 |
| FR-7.4 three consumers | 15 |
| FR-7.5 map-entry replay | 16 |
| FR-7.6 `produceWriters()` — verify only | 15 Step 3 |
| FR-8.1 tenant config (producer side) | 5 |
| FR-8.1 tenant config (consumer side) · FR-8.2 DOM-25 | 8 |
| FR-9.1/9.2/9.3 template writers, v12 excluded · FR-9.4 no new handler entry | 4 |
| §5 REST surface | 10 |
| §6 data model, no Postgres | 7 |
| §7 service registration | 12 |
| §8 non-functional gates | 17 |
| Q4 message bound 182 | 8, 9 |
| Q6 per-instance cap | 7, 11 |
