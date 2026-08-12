# Monster Catch Items Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make catch (bridle) consumables work end to end — the client's `USE_CATCH_ITEM` request decodes, resolves against the targeted monster, removes it without kill rewards, grants the reward item, plays the capture effect, and always unlocks the client — while correcting every coverage-matrix cell this work proves wrong.

**Architecture:** `atlas-channel` decodes the request and forwards a `REQUEST_CATCH_MONSTER` command to `atlas-consumables`, which validates the item and reserves it, then emits a `CATCH` command on `COMMAND_TOPIC_MONSTER`. `atlas-monsters` owns the species/HP/roll checks and the atomic claim-and-remove, then publishes the outcome twice: `CATCH_RESOLVED` on a new dedicated low-volume topic (consumables commits or cancels the reservation) and `CAUGHT`/`CATCH_FAILED` on the existing monster status topic (the channel renders effects and unlocks). Codec work adds the serverbound `UseCatchItem` struct and repairs two clientbound catch codecs whose `uniqueId` handling the design-phase IDA pass proved wrong.

**Tech Stack:** Go 1.x, Kafka (segmentio), Redis (`libs/atlas-redis`), JSON:API REST (`libs/atlas-rest`), `libs/atlas-packet` codecs, `tools/packet-audit`, tenant seed templates (`services/atlas-configurations/seed-data/templates/`).

## Global Constraints

- **Source of truth:** [`prd.md`](prd.md) and [`design.md`](design.md). Every IDA address quoted below is copied from `design.md` §2/§3 — do not re-derive them, and do not invent new ones.
- **Never hard-code an opcode in Go** (DOM-25). Opcodes live only in `services/atlas-configurations/seed-data/templates/*.json` and `docs/packets/registry/*.yaml`.
- **Never hard-code catch item ids, mob ids, `create` ids, `mobHP`, or `bridleProp` in Go.** All are resolved per tenant at runtime from atlas-data (PRD §6, FR-6).
- **No new domain constants** without first checking `libs/atlas-constants/` (DOM-21).
- **Multi-tenancy:** every hop uses `tenant.MustFromContext(ctx)`; the new atlas-monsters consumable client does **not** cache (design §7).
- **Buff-duration guard:** nothing in this task touches `COMMAND_TOPIC_CHARACTER_BUFF`; do not add a `duration` field anywhere.
- **Goroutine guard:** any goroutine must go through `routine.Go`.
- **Redis guard:** keyed Redis commands only inside `libs/atlas-redis`.
- **Test helpers:** use the project Builder pattern. No `*_testhelpers.go` files.
- **No TODOs, stubs, or 501s** in landed commits.
- Every commit runs in the worktree `.worktrees/task-212-monster-catch-items` on branch `task-212-monster-catch-items`.

## Design corrections carried into this plan

Three points where `design.md` names something that does not exist as written. The plan implements the corrected form; the design intent is unchanged.

1. **§4.5 names `TenantRegistry.RemoveExisting`.** `atlas-monsters` stores monsters in `atlasredis.Registry[string, storedMonster]` (`monster/registry.go:284`), **not** `TenantRegistry` — the tenant is baked into the key suffix (`monsterSuffix`). The atomic claim primitive therefore goes on `Registry[K, V]` in `libs/atlas-redis/registry.go`. (Task 8.)
2. **§4.4 step 1/6 "silent drop" leaves a dangling reservation.** A `CATCH` command whose monster is gone (or that loses the claim) must still emit `CATCH_RESOLVED(success=false, cause=UNRESOLVED)` so consumables cancels the reservation, and `CATCH_FAILED(cause=UNRESOLVED)` so the channel unlocks the client. Per design §6.4 the `UNRESOLVED` cause renders **no** `BridleMobCatchFail` packet — EnableActions only. Idempotency still holds: the consumables once-handler has already deregistered by the time a redelivery arrives, so the second event is a no-op. (Tasks 10, 11c, 12b.)
3. **§8 understates the v92 writer gap.** `template_gms_92_1.json` routes **none** of `BridleMobCatchFail` (0x053), `CatchMonster` (0x123), `CatchMonsterWithItem` (0x124) — verified by reading the template. All three routes are added. (Task 6.)

---

## File Structure

**`libs/atlas-packet`**
- Create `monster/serverbound/use_catch_item.go` — the `UseCatchItem` request struct (Encode + Decode).
- Create `monster/serverbound/use_catch_item_test.go` — round-trip + ten per-version byte fixtures.
- Modify `monster/clientbound/catch_monster.go` — delete `legacyMobPoolPrefix`, write `uniqueId` unconditionally.
- Modify `monster/clientbound/monster_special_effect_by_skill.go`, `monster/clientbound/inc_mob_charge_count.go` — same one-line fix (design SD-3).
- Modify `monster/clientbound/catch_monster_with_item.go` — add leading `uniqueId`, gate the trailing `result` byte off on gms_v48.
- Modify the three existing `*_test.go` fixtures whose expected bytes change.

**`libs/atlas-redis`**
- Modify `registry.go` — add `Registry.RemoveExisting` (atomic delete-and-claim).
- Modify `registry_test.go` — concurrency test proving exactly one `true`.

**`libs/atlas-constants`**
- Modify `item/constants.go` — add `ClassificationConsumableCatchItem = Classification(227)`.

**`tools/packet-audit`**
- Modify `cmd/run.go` — add the `CWvsContext::SendBridleItemUseRequest` case to `candidatesFromFName`.

**`docs/packets`**
- Modify `registry/gms_v48.yaml`, `gms_v61.yaml`, `gms_v72.yaml`, `gms_v79.yaml`, `gms_v92.yaml`.
- Modify `feature-na-evidence.yaml` — the v48 `BRIDLE_MOB_CATCH_FAIL` affirmation.
- Regenerate `audits/STATUS.md`, `audits/status.json`; add `evidence/**` and `audits/<version>/**` artifacts.

**`services/atlas-configurations/seed-data/templates`**
- Modify all ten matrix templates — `USE_CATCH_ITEM` handler route; v48 and v92 writer routes.

**`services/atlas-monsters`**
- Create `monster/consumable/{requests.go,rest.go,model.go,processor.go,mock/processor.go}` — the atlas-data catch-item client.
- Create `monster/catch.go` — `Processor.Catch` and the resolution ladder.
- Create `monster/catch_test.go`.
- Modify `monster/registry.go` — `Registry.ClaimMonster`.
- Modify `monster/processor.go` — `Catch` on the `Processor` interface, `testConsumableLookup` seam.
- Modify `monster/kafka.go`, `monster/producer.go` — new event types, bodies, providers.
- Modify `kafka/consumer/monster/{kafka.go,consumer.go}` — `CATCH` command.

**`services/atlas-consumables`**
- Modify `data/consumable/model.go` — the seven missing getters.
- Create `catchdelay/registry.go` — the `useDelay` Redis gate.
- Create `consumable/catch.go` — `RequestCatchMonster` + the reserve/commit/cancel handlers.
- Create `consumable/catch_test.go`.
- Create `kafka/message/monster/kafka.go`, `kafka/consumer/monster/consumer.go` — the `EVENT_TOPIC_MONSTER_CATCH` consumer.
- Modify `consumable/processor.go` (interface), `kafka/message/consumable/kafka.go`, `kafka/consumer/consumable/consumer.go`, `monster/*` (catch command producer), `main.go`.

**`services/atlas-channel`**
- Create `socket/handler/monster_catch_item_use.go`, `socket/handler/monster_catch_item_use_test.go`.
- Modify `consumable/{processor.go,producer.go}`, `kafka/message/consumable/kafka.go`, `kafka/message/monster/kafka.go`, `kafka/consumer/monster/consumer.go`, `kafka/consumer/consumable/consumer.go`, `socket/writer/catch_monster_with_item.go`, `main.go`.

**`deploy/k8s`**
- Modify `base/env-configmap.yaml`, `overlays/main/kustomization.yaml`, `overlays/pr/kustomization.yaml`.

---

## Task 1: `UseCatchItem` serverbound codec

**Files:**
- Create: `libs/atlas-packet/monster/serverbound/use_catch_item.go`
- Test: `libs/atlas-packet/monster/serverbound/use_catch_item_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `serverbound.UseCatchItemHandle` (const `string` = `"MonsterCatchItemUseHandle"`), `serverbound.UseCatchItem` struct with getters `UpdateTime() uint32`, `Slot() int16`, `ItemId() uint32`, `MonsterUniqueId() uint32`, plus `Operation() string`, `String() string`, `Encode`, `Decode`.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/monster/serverbound/use_catch_item_test.go`:

```go
package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestUseCatchItemBytes pins the wire layout of the serverbound USE_CATCH_ITEM
// request (CWvsContext::SendBridleItemUseRequest). The body is identical on
// every version inspected — Encode4 updateTime, Encode2 nPOS, Encode4 nItemID,
// Encode4 hit-mob object id — so there is deliberately NO version gate here
// (design.md §5.1; PRD FR-1.1).
//
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=gms_v48 ida=0x70e0c5
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=gms_v61 ida=0x832005
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=gms_v72 ida=0x90457d
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=gms_v79 ida=0x9558e5
// packet-audit:verify packet=monster/serverbound/MonsterUseCatchItem version=gms_v95 ida=0x9e08c0
func TestUseCatchItemBytes(t *testing.T) {
	input := NewUseCatchItem(0x11223344, 0x0005, 2270008, 0x07654321)

	want := []byte{
		0x44, 0x33, 0x22, 0x11, // updateTime      uint32 LE (Encode4 get_update_time)
		0x05, 0x00, // slot            int16  LE (Encode2 nPOS)
		0x38, 0xa4, 0x22, 0x00, // itemId          uint32 LE (Encode4 nItemID)
		0x21, 0x43, 0x65, 0x07, // monsterUniqueId uint32 LE (Encode4 hit-mob id)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := input.Encode(nil, ctx)(nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("UseCatchItem %s layout mismatch\n got %% x\nwant %% x", v.Name, got, want)
			}
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestUseCatchItemDecode proves Decode recovers every field (the handler path).
func TestUseCatchItemDecode(t *testing.T) {
	input := NewUseCatchItem(0x11223344, 0x0005, 2270008, 0x07654321)
	ctx := pt.CreateContext("GMS", 83, 1)

	var out UseCatchItem
	pt.RoundTrip(t, ctx, input.Encode, out.Decode, nil)

	if out.UpdateTime() != 0x11223344 || out.Slot() != 5 ||
		out.ItemId() != 2270008 || out.MonsterUniqueId() != 0x07654321 {
		t.Fatalf("decoded %+v, want updateTime=0x11223344 slot=5 itemId=2270008 uniqueId=0x07654321", out)
	}
}
```

Note the `%% x` escapes are for this markdown block only — write plain `% x` in the Go file.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./monster/serverbound/ -run UseCatchItem -v`
Expected: FAIL — `undefined: NewUseCatchItem`, `undefined: UseCatchItem`.

- [ ] **Step 3: Write the implementation**

Create `libs/atlas-packet/monster/serverbound/use_catch_item.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const UseCatchItemHandle = "MonsterCatchItemUseHandle"

// UseCatchItem is the serverbound USE_CATCH_ITEM packet
// (CWvsContext::SendBridleItemUseRequest): the client asks the server to use a
// bridle/catch consumable (item class 227) on a specific field monster.
//
// Byte layout (IDA-verified, identical on every version inspected):
//   - updateTime      : uint32 — client get_update_time()
//   - slot            : int16  — inventory position (nPOS)
//   - itemId          : uint32 — the catch item (nItemID, always /10000 == 227)
//   - monsterUniqueId : uint32 — the hit mob's field object id
//
// IDA basis: CWvsContext::SendBridleItemUseRequest — gms_v48 @0x70e0c5,
// gms_v61 @0x832005, gms_v72 @0x90457d, gms_v79 @0x9558e5, gms_v95 @0x9e08c0.
// No version-gated divergence was observed, so this codec carries NO
// MajorAtLeast gate; introduce one only if a remaining IDB proves otherwise.
//
// The client sets ExclRequest immediately after the COutPacket ctor on every
// version inspected, so EVERY terminal server outcome must send an unlocking
// packet (design.md §4.6).
//
// packet-audit:fname CWvsContext::SendBridleItemUseRequest
type UseCatchItem struct {
	updateTime      uint32
	slot            int16
	itemId          uint32
	monsterUniqueId uint32
}

func NewUseCatchItem(updateTime uint32, slot int16, itemId uint32, monsterUniqueId uint32) UseCatchItem {
	return UseCatchItem{updateTime: updateTime, slot: slot, itemId: itemId, monsterUniqueId: monsterUniqueId}
}

func (m UseCatchItem) UpdateTime() uint32      { return m.updateTime }
func (m UseCatchItem) Slot() int16             { return m.slot }
func (m UseCatchItem) ItemId() uint32          { return m.itemId }
func (m UseCatchItem) MonsterUniqueId() uint32 { return m.monsterUniqueId }
func (m UseCatchItem) Operation() string       { return UseCatchItemHandle }
func (m UseCatchItem) String() string {
	return fmt.Sprintf("updateTime [%d], slot [%d], itemId [%d], monsterUniqueId [%d]", m.updateTime, m.slot, m.itemId, m.monsterUniqueId)
}

func (m UseCatchItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.slot)
		w.WriteInt(m.itemId)
		w.WriteInt(m.monsterUniqueId)
		return w.Bytes()
	}
}

func (m *UseCatchItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.slot = r.ReadInt16()
		m.itemId = r.ReadUint32()
		m.monsterUniqueId = r.ReadUint32()
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-packet && go test -race ./monster/serverbound/ -run UseCatchItem -v`
Expected: PASS for every `pt.Variants` sub-test.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/monster/serverbound/use_catch_item.go libs/atlas-packet/monster/serverbound/use_catch_item_test.go
git commit -m "feat(task-212): add USE_CATCH_ITEM serverbound codec"
```

---

## Task 2: Registry entries and packet-audit fname linkage

**Files:**
- Modify: `docs/packets/registry/gms_v48.yaml`, `gms_v61.yaml`, `gms_v72.yaml`, `gms_v79.yaml`
- Modify: `tools/packet-audit/cmd/run.go`

**Interfaces:**
- Consumes: `libs/atlas-packet/monster/serverbound/UseCatchItem` (Task 1).
- Produces: the `candidatesFromFName` mapping `CWvsContext::SendBridleItemUseRequest → {name: "UseCatchItem", pkg: "monster", dir: DirServerbound}`, which makes the matrix path `monster/serverbound/MonsterUseCatchItem` and the report filename `MonsterUseCatchItem.json`.

- [ ] **Step 1: Add the registry entries**

Each of the four legacy registries gains one entry, inserted at its **sorted position within the serverbound block** (match the surrounding file's ordering — entries are ordered by opcode within direction). The `opcode` field is decimal.

`docs/packets/registry/gms_v48.yaml` — opcode `63` (`0x03F`):

```yaml
- op: USE_CATCH_ITEM
  direction: serverbound
  opcode: 63
  fname: CWvsContext::SendBridleItemUseRequest
  provenance: ida-derived
```

`gms_v61.yaml` — opcode `74` (`0x04A`).
`gms_v72.yaml` — opcode `80` (`0x050`).
`gms_v79.yaml` — opcode `79` (`0x04F`).

All four use the identical `op`, `direction`, `fname`, and `provenance` values shown above; only `opcode` differs.

- [ ] **Step 2: Add the packet-audit fname case**

In `tools/packet-audit/cmd/run.go`, inside `candidatesFromFName`, next to the other `CMob::`/monster serverbound cases (near `case "CMob::SendDropPickUpRequest":`), add:

```go
	case "CWvsContext::SendBridleItemUseRequest":
		// task-212: USE_CATCH_ITEM — atlas UseCatchItem (handle =
		// "MonsterCatchItemUseHandle"), in monster/serverbound because the
		// request targets a field monster and carries its object id.
		// Encode4 updateTime + Encode2 nPOS + Encode4 nItemID + Encode4 mobId,
		// identical on gms_v48 @0x70e0c5 / v61 @0x832005 / v72 @0x90457d /
		// v79 @0x9558e5 / v95 @0x9e08c0.
		return []candidate{{name: "UseCatchItem", pkg: "monster", dir: csvpkg.DirServerbound}}
```

- [ ] **Step 3: Verify the linkage resolves**

Run from the worktree root:

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  -ida-source docs/packets/ida-exports/gms_83.json \
  -output /tmp/claude-1000/rpt-212
ls /tmp/claude-1000/rpt-212/gms_v83/MonsterUseCatchItem.json
```

Expected: the report file exists. If the `-ida-source` filename differs, list `docs/packets/ida-exports/` and use the gms_v83 export actually present — do not guess.

- [ ] **Step 4: Commit**

```bash
git add docs/packets/registry tools/packet-audit/cmd/run.go
git commit -m "feat(task-212): register USE_CATCH_ITEM on v48/v61/v72/v79 and link its fname"
```

---

## Task 3: Route `USE_CATCH_ITEM` in all ten seed templates

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_{48,61,72,79,83,84,87,92,95}_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`

**Interfaces:**
- Consumes: `serverbound.UseCatchItemHandle` = `"MonsterCatchItemUseHandle"` (Task 1); the opcodes from Task 2 / PRD FR-1.2.
- Produces: the tenant-config route the channel handler (Task 12a) binds to.

- [ ] **Step 1: Insert one `handlers` entry per template**

Each entry goes into `socket.handlers` at its **sorted `opCode` position** — never appended, never placed next to a semantically related entry. `template_gms_12_1.json` is explicitly out of scope (design §6.6).

Entry shape (the v83 example; only `opCode` changes per file):

```json
{
  "opCode": "0x51",
  "validator": "LoggedInValidator",
  "handler": "MonsterCatchItemUseHandle",
  "fname": "CWvsContext::SendBridleItemUseRequest",
  "services": [
    "channel"
  ]
}
```

Per-file `opCode` values:

| File | `opCode` |
|---|---|
| `template_gms_48_1.json` | `"0x3F"` |
| `template_gms_61_1.json` | `"0x4A"` |
| `template_gms_72_1.json` | `"0x50"` |
| `template_gms_79_1.json` | `"0x4F"` |
| `template_gms_83_1.json` | `"0x51"` |
| `template_gms_84_1.json` | `"0x51"` |
| `template_gms_87_1.json` | `"0x54"` |
| `template_gms_92_1.json` | `"0x58"` |
| `template_gms_95_1.json` | `"0x57"` |
| `template_jms_185_1.json` | `"0x49"` |

Match each file's existing hex casing and zero-padding convention for the neighbouring entries (read the two entries either side before inserting). An empty `validator` makes the handler silently disappear at load — `LoggedInValidator` is mandatory on all ten.

- [ ] **Step 2: Run the template guards**

```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
```

Expected: all three exit 0. If the order guard reports a misplaced entry, move it — do not relax the guard.

- [ ] **Step 3: Verify each template parses and carries exactly one route**

```bash
for f in services/atlas-configurations/seed-data/templates/template_gms_{48,61,72,79,83,84,87,92,95}_1.json services/atlas-configurations/seed-data/templates/template_jms_185_1.json; do
  python3 -c "
import json,sys
d=json.load(open('$f'))
e=[h for h in d['socket']['handlers'] if h['handler']=='MonsterCatchItemUseHandle']
assert len(e)==1, ('$f', e)
assert e[0]['validator'], ('$f','empty validator')
print('$f', e[0]['opCode'])
"
done
```

Expected: ten lines, each printing the opCode from the table.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates
git commit -m "feat(task-212): route USE_CATCH_ITEM in all ten tenant templates"
```

---

## Task 4: Make the `CMobPool::OnMobPacket` uniqueId prefix unconditional

**Files:**
- Modify: `libs/atlas-packet/monster/clientbound/catch_monster.go`
- Modify: `libs/atlas-packet/monster/clientbound/monster_special_effect_by_skill.go`
- Modify: `libs/atlas-packet/monster/clientbound/inc_mob_charge_count.go`
- Test: `libs/atlas-packet/monster/clientbound/catch_monster_test.go`, `monster_special_effect_by_skill_test.go`, `inc_mob_charge_count_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `CatchMonster`, `MonsterSpecialEffectBySkill`, `IncMobChargeCount` all write their leading `uniqueId` on every version. `legacyMobPoolPrefix` no longer exists.

This is design SD-3's **include them** recommendation. All three codecs are dispatched through the same `CMobPool::OnMobPacket` switch, which consumes a leading `Decode4` → `GetMob` on every version in the matrix (design §2 F-1 table); gating the prefix to pre-v83 makes v83+ unreadable by the client.

- [ ] **Step 1: Write the failing test**

In `libs/atlas-packet/monster/clientbound/catch_monster_test.go`, replace the v83/v95 golden-byte blocks inside `TestCatchMonster` with prefixed expectations:

```go
	// Golden bytes (v83 baseline). CMobPool::OnMobPacket @0x67936d reads the
	// mob object id (Decode4) -> GetMob BEFORE dispatching; CMob::OnCatchEffect
	// @0x66d6b9 then reads v3 = Decode1(a1) -> ShowCatchEffect(this, v3).
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x67936d)
		0x42, // result byte (Decode1 @0x66d6b9)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("CatchMonster v83 layout mismatch\n got % x\nwant % x", got, want)
	}

	// Golden bytes (v95). CMobPool::OnMobPacket @0x6570b0 Decode4 -> GetMob,
	// then CMob::OnCatchEffect @0x63cd00: v3 = Decode1; v4 = Decode1.
	gotV95 := input.Encode(nil, pt.CreateContext("GMS", 95, 0))(nil)
	wantV95 := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x6570b0)
		0x42, // result  byte (Decode1 @0x63cd00)
		0x01, // success byte (Decode1 @0x63cd00)
	}
	if !bytes.Equal(gotV95, wantV95) {
		t.Fatalf("CatchMonster v95 layout mismatch\n got % x\nwant % x", gotV95, wantV95)
	}
```

Add a v92 fixture to the same file:

```go
// TestCatchMonsterBytesV92 pins the v92 wire against CMobPool::OnMobPacket
// @0x64a6c0 (Decode4 -> GetMob) dispatching case 291 to sub_630C30 @0x630c30,
// which reads a single Decode1 — the v83 shape, NOT the two-byte v95 one.
//
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonster version=gms_v92 ida=0x630c30
func TestCatchMonsterBytesV92(t *testing.T) {
	input := NewCatchMonster(0x07654321, 0x42, 0x01)
	ctx := pt.CreateContext("GMS", 92, 1)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x64a6c0)
		0x42, // result byte (Decode1 @0x630c30)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v92 catchMonster bytes:\n got % x\nwant % x", got, want)
	}
}
```

Add a v48 fixture to the same file:

```go
// TestCatchMonsterBytesV48 pins the v48 wire. CMobPool::OnMobPacket @0x559390
// reads the mob object id (Decode4) -> GetMob, then dispatches case 172 to
// sub_5511F4, which reads one Decode1 (the result byte). No success byte.
//
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonster version=gms_v48 ida=0x5511f4
func TestCatchMonsterBytesV48(t *testing.T) {
	input := NewCatchMonster(0x07654321, 0x42, 0x01)
	ctx := pt.CreateContext("GMS", 48, 1)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x559390)
		0x42, // result byte (Decode1, sub_5511F4)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v48 catchMonster bytes:\n got % x\nwant % x", got, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./monster/clientbound/ -run 'CatchMonster' -v`
Expected: FAIL — v83/v95/v92/v48 byte mismatches (4 fewer leading bytes than expected).

- [ ] **Step 3: Implement — delete the gate**

In `catch_monster.go`, delete the entire `legacyMobPoolPrefix` function and its doc comment, and replace both call sites with an unconditional write/read:

```go
func (m CatchMonster) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteByte(m.result)
		if v95CatchLayout(t) {
			w.WriteByte(m.success)
		}
		return w.Bytes()
	}
}

func (m *CatchMonster) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.result = r.ReadByte()
		if v95CatchLayout(t) {
			m.success = r.ReadByte()
		}
	}
}
```

Replace the struct's "Legacy (pre-v83) wire note" paragraph with:

```go
// Wire note: CATCH_MONSTER is a per-mob OnMobPacket case, so the client consumes
// a leading uniqueId via CMobPool::OnMobPacket (Decode4 -> GetMob) BEFORE
// CMob::OnCatchEffect reads the result byte. This is universal, not legacy —
// confirmed on v48 @0x559390, v61 @0x5d48f3, v79 @0x646d46, v83 @0x67936d,
// v92 @0x64a6c0, v95 @0x6570b0, jms @0x6f8732, and by symbol on v84/v87
// (task-212 design.md §2 F-1). It was previously gated to pre-v83 by
// legacyMobPoolPrefix, which made every v83+ catch packet undecodable by the
// client; the gate is deleted.
```

In `monster_special_effect_by_skill.go` and `inc_mob_charge_count.go`, replace each `if legacyMobPoolPrefix(t) {` block with the unconditional statement (`w.WriteInt(m.uniqueId)` in Encode, `m.uniqueId = r.ReadUint32()` in Decode) and drop the now-unused `t := tenant.MustFromContext(ctx)` where it becomes unused. Update each file's prefix comment to cite the same F-1 addresses.

- [ ] **Step 4: Fix the two sibling fixtures**

`monster_special_effect_by_skill_test.go` and `inc_mob_charge_count_test.go` contain v83+ golden bytes that now gain a 4-byte prefix. Prepend `0x21, 0x43, 0x65, 0x07,` (or whatever uniqueId that test's input carries — read it, do not assume) to each affected `want` slice, with the comment `// uniqueId int32 LE (pool Decode4, universal — task-212 F-1)`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test -race ./monster/clientbound/ -v`
Expected: PASS, whole package.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/monster/clientbound
git commit -m "fix(task-212): OnMobPacket uniqueId prefix is universal, not pre-v83"
```

---

## Task 5: `CatchMonsterWithItem` gains `uniqueId`; v48 drops the result byte

**Files:**
- Modify: `libs/atlas-packet/monster/clientbound/catch_monster_with_item.go`
- Test: `libs/atlas-packet/monster/clientbound/catch_monster_with_item_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/catch_monster_with_item.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `NewCatchMonsterWithItem(uniqueId uint32, itemId int32, result byte) CatchMonsterWithItem` (**signature change** — a leading parameter is added), getter `UniqueId() uint32`; and `writer.CatchMonsterWithItemBody(uniqueId uint32, itemId int32, result byte) packet.Encode`.

- [ ] **Step 1: Write the failing test**

Rewrite `libs/atlas-packet/monster/clientbound/catch_monster_with_item_test.go`:

```go
package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v83 ida=0x66d997
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v84 ida=0x683c9f
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v87 ida=0x6a886e
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v92 ida=0x630c50
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v95 ida=0x63cd40
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=jms_v185 ida=0x6eb148
func TestCatchMonsterWithItem(t *testing.T) {
	input := NewCatchMonsterWithItem(0x07654321, 2270008, 0x01)

	// v83 baseline: CMobPool::OnMobPacket @0x67936d Decode4 -> GetMob, then
	// CMob::OnEffectByItem @0x66d997 reads Decode4 itemId + Decode1 result.
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x67936d)
		0x38, 0xa4, 0x22, 0x00, // itemId   int32 LE (Decode4 @0x66d997)
		0x01, // result   byte  (Decode1 @0x66d997)
	}
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	if !bytes.Equal(got, want) {
		t.Fatalf("CatchMonsterWithItem v83 layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestCatchMonsterWithItemBytesV48 pins the v48 SHORT body. CMobPool::OnMobPacket
// @0x559390 Decode4 -> GetMob dispatches case 173 to sub_551481, which reads
// ONLY Decode4 (the item id) and passes it to sub_54E82D — there is no trailing
// result byte on v48. Every later version reads Decode4 + Decode1 (v61
// @0x5cc793, v79 @0x63c937, v92 @0x630c50). design.md §2 F-2.
//
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v48 ida=0x551481
func TestCatchMonsterWithItemBytesV48(t *testing.T) {
	input := NewCatchMonsterWithItem(0x07654321, 2270008, 0x01)
	ctx := pt.CreateContext("GMS", 48, 1)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x559390)
		0x38, 0xa4, 0x22, 0x00, // itemId   int32 LE (Decode4, sub_551481)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v48 catchMonsterWithItem bytes:\n got % x\nwant % x", got, want)
	}
}

// TestCatchMonsterWithItemBytesV61 pins the first version that DOES carry the
// trailing result byte — the boundary the v48CatchByItemNoResult gate encodes.
//
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v61 ida=0x5cc793
func TestCatchMonsterWithItemBytesV61(t *testing.T) {
	input := NewCatchMonsterWithItem(0x07654321, 2270008, 0x01)
	ctx := pt.CreateContext("GMS", 61, 1)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x5d48f3)
		0x38, 0xa4, 0x22, 0x00, // itemId   int32 LE (Decode4 @0x5cc793)
		0x01, // result   byte  (Decode1 @0x5cc793)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 catchMonsterWithItem bytes:\n got % x\nwant % x", got, want)
	}
}

// TestCatchMonsterWithItemBytesV79 pins the v79 cell (fixture promotion).
//
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v79 ida=0x63c937
func TestCatchMonsterWithItemBytesV79(t *testing.T) {
	input := NewCatchMonsterWithItem(0x07654321, 2270008, 0x01)
	ctx := pt.CreateContext("GMS", 79, 1)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x646d46)
		0x38, 0xa4, 0x22, 0x00, // itemId   int32 LE (Decode4 @0x63c937)
		0x01, // result   byte  (Decode1 @0x63c937)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 catchMonsterWithItem bytes:\n got % x\nwant % x", got, want)
	}
}
```

The gms_v72 cell has no independently derived address in `design.md`; do **not** invent one. It is promoted in Task 7 only if a v72 `CMob::OnEffectByItem` address can be read from the checked-in export or a live IDB — otherwise its 🟡ᶠ stands and Task 7 records why.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./monster/clientbound/ -run CatchMonsterWithItem -v`
Expected: FAIL — `too many arguments in call to NewCatchMonsterWithItem`.

- [ ] **Step 3: Implement**

Rewrite `catch_monster_with_item.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CatchMonsterWithItemWriter = "CatchMonsterWithItem"

// CatchMonsterWithItem is the clientbound CATCH_MONSTER_WITH_ITEM packet
// (CMob::OnEffectByItem): the server tells the client to play a capture-by-item
// effect on the targeted mob.
//
// Byte layout (IDA-verified):
//   - uniqueId : int32 — the mob object id, consumed by CMobPool::OnMobPacket
//     (Decode4 -> GetMob) BEFORE dispatch. Universal, every version.
//   - itemId   : int32 — the catch item id (Decode4 -> ShowEffectByItem 1st arg)
//   - result   : byte  — the effect result code (Decode1 -> 2nd arg).
//     ABSENT on gms_v48 (see v48CatchByItemNoResult).
//
// IDA basis: CMobPool::OnMobPacket — v48 @0x559390, v61 @0x5d48f3, v79 @0x646d46,
// v83 @0x67936d, v92 @0x64a6c0, v95 @0x6570b0, jms @0x6f8732 (task-212 §2 F-1).
// CMob::OnEffectByItem — v61 @0x5cc793, v79 @0x63c937, v83 @0x66d997,
// v84 @0x683c9f, v87 @0x6a886e, v92 @0x630c50, v95 @0x63cd40, jms @0x6eb148;
// v48's arm is sub_551481, which reads Decode4 and nothing else (§2 F-2).
//
// packet-audit:fname CMob::OnEffectByItem
type CatchMonsterWithItem struct {
	uniqueId uint32
	itemId   int32
	result   byte
}

func NewCatchMonsterWithItem(uniqueId uint32, itemId int32, result byte) CatchMonsterWithItem {
	return CatchMonsterWithItem{uniqueId: uniqueId, itemId: itemId, result: result}
}

func (m CatchMonsterWithItem) UniqueId() uint32   { return m.uniqueId }
func (m CatchMonsterWithItem) ItemId() int32      { return m.itemId }
func (m CatchMonsterWithItem) Result() byte       { return m.result }
func (m CatchMonsterWithItem) Operation() string  { return CatchMonsterWithItemWriter }
func (m CatchMonsterWithItem) String() string {
	return fmt.Sprintf("uniqueId [%d], itemId [%d], result [%d]", m.uniqueId, m.itemId, m.result)
}

// v48CatchByItemNoResult reports whether the tenant omits OnEffectByItem's
// trailing result byte. VERIFIED: v48's arm sub_551481 @0x551481 reads Decode4
// and nothing else; every later version reads Decode4 + Decode1 (v61 @0x5cc793,
// v79 @0x63c937, v92 @0x630c50, and v83/v84/v87/v95/jms per the addresses above).
func v48CatchByItemNoResult(t tenant.Model) bool {
	return t.IsRegion("GMS") && !t.MajorAtLeast(61)
}

func (m CatchMonsterWithItem) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteInt32(m.itemId)
		if !v48CatchByItemNoResult(t) {
			w.WriteByte(m.result)
		}
		return w.Bytes()
	}
}

func (m *CatchMonsterWithItem) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.itemId = r.ReadInt32()
		if !v48CatchByItemNoResult(t) {
			m.result = r.ReadByte()
		}
	}
}
```

- [ ] **Step 4: Update the channel writer**

`services/atlas-channel/atlas.com/channel/socket/writer/catch_monster_with_item.go`:

```go
// CatchMonsterWithItemBody encodes the clientbound CATCH_MONSTER_WITH_ITEM
// packet, which plays a capture-by-item effect on a targeted mob. The leading
// uniqueId is consumed by CMobPool::OnMobPacket before dispatch, so the mob
// must still exist client-side when this arrives (task-212 design §4.2).
func CatchMonsterWithItemBody(uniqueId uint32, itemId int32, result byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewCatchMonsterWithItem(uniqueId, itemId, result).Encode(l, ctx)(options)
		}
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd libs/atlas-packet && go test -race ./monster/... && cd - >/dev/null
cd services/atlas-channel/atlas.com/channel && go build ./... && cd - >/dev/null
```
Expected: both clean. `RoundTrip` on the `GMS v48` variant must survive the short body (it will — Decode mirrors the gate).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/monster/clientbound services/atlas-channel/atlas.com/channel/socket/writer/catch_monster_with_item.go
git commit -m "fix(task-212): CatchMonsterWithItem carries uniqueId; v48 omits the result byte"
```

---

## Task 6: Writer routes for the v48 and v92 clientbound catch cells

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_48_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
- Modify: `docs/packets/registry/gms_v48.yaml`, `docs/packets/registry/gms_v92.yaml`

**Interfaces:**
- Consumes: `clientbound.CatchMonsterWriter` = `"CatchMonster"`, `clientbound.CatchMonsterWithItemWriter` = `"CatchMonsterWithItem"`, `charpkt.BridleMobCatchFailWriter` = `"BridleMobCatchFail"`.
- Produces: the writer routes without which the channel's announce silently reports "unconfigured".

- [ ] **Step 1: Add v48 writer routes**

`template_gms_48_1.json` currently routes none of the three (verified). Add two entries at their sorted `opCode` positions in `socket.writers`:

```json
{ "opCode": "0xAC", "writer": "CatchMonster", "fname": "CMob::OnCatchEffect", "services": ["channel"] }
```
```json
{ "opCode": "0xAD", "writer": "CatchMonsterWithItem", "fname": "CMob::OnEffectByItem", "services": ["channel"] }
```

**Do not add `BridleMobCatchFail` to v48** — design §2 F-3 proves the handler does not exist (`CWvsContext::OnPacket` @0x70d215 is a complete switch over cases 25–70 with no bridle-fail arm; the four unnamed arms `sub_72025D`/`sub_713202`/`sub_721481`/`sub_7215EA` were decompiled and ruled out).

- [ ] **Step 2: Add v92 writer routes**

`template_gms_92_1.json` routes none of the three either. Add all three at their sorted positions:

```json
{ "opCode": "0x53", "writer": "BridleMobCatchFail", "fname": "CWvsContext::OnBridleMobCatchFail", "services": ["channel"] }
```
```json
{ "opCode": "0x123", "writer": "CatchMonster", "fname": "CMob::OnCatchEffect", "services": ["channel"] }
```
```json
{ "opCode": "0x124", "writer": "CatchMonsterWithItem", "fname": "CMob::OnEffectByItem", "services": ["channel"] }
```

(`0x053` for the fail packet is the opcode STATUS.md already records for gms_v92; `0x123` likewise; `0x124` is derived in design §3 from `CMobPool::OnMobPacket` @0x64a6c0 case 292 → `CMob::OnEffectByItem` @0x630c50.)

- [ ] **Step 3: Add the matching registry entries**

`docs/packets/registry/gms_v48.yaml` — two clientbound entries at their sorted positions:

```yaml
- op: CATCH_MONSTER
  direction: clientbound
  opcode: 172
  fname: CMob::OnCatchEffect
  provenance: ida-derived
```
```yaml
- op: CATCH_MONSTER_WITH_ITEM
  direction: clientbound
  opcode: 173
  fname: CMob::OnEffectByItem
  provenance: ida-derived
```

`docs/packets/registry/gms_v92.yaml` — one clientbound entry (opcode `292`, `0x124`) with `op: CATCH_MONSTER_WITH_ITEM`, `fname: CMob::OnEffectByItem`, `provenance: ida-derived`. If `CATCH_MONSTER` is already present in the v92 registry (STATUS.md shows it at 0x123), leave it alone.

- [ ] **Step 4: Run the guards**

```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
```
Expected: exit 0 for both.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/seed-data/templates docs/packets/registry
git commit -m "feat(task-212): route the catch clientbound writers on v48 and v92"
```

---

## Task 7: Promote the coverage-matrix cells

**Files:**
- Modify: `docs/packets/feature-na-evidence.yaml`
- Create: `docs/packets/evidence/<version>/*.yaml` (via the tool)
- Create/modify: `docs/packets/audits/<version>/{MonsterUseCatchItem,MonsterCatchMonster,MonsterCatchMonsterWithItem,CharacterBridleMobCatchFail}.{json,md}`
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

**Interfaces:**
- Consumes: Tasks 1–6 (codecs, markers, registry entries, template routes).
- Produces: a clean `go run ./tools/packet-audit matrix --check` (exit 0).

This task is the single-cell verify procedure from [`VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md), applied to the cells design §3 enumerates. Do not restate that playbook — follow it.

- [ ] **Step 1: Affirm the v48 `BRIDLE_MOB_CATCH_FAIL` n-a**

Append to `entries:` in `docs/packets/feature-na-evidence.yaml`:

```yaml
  - op: BRIDLE_MOB_CATCH_FAIL
    version: gms_v48
    evidence: >
      gms_v48 has no BRIDLE_MOB_CATCH_FAIL receive handler. CWvsContext::OnPacket
      @0x70d215 is a complete compiled switch over cases 25-70 with no bridle-fail
      arm; the four unnamed arms were individually decompiled and ruled out —
      sub_72025D (case 58, a delegating stub), sub_713202 (case 62), sub_721481
      (case 68, a two-byte + optional-string notice) and sub_7215EA (case 70, the
      same shape). The v48 IDB is symbol-rich across CWvsContext (mangled MSVC
      names throughout) and CWvsContext::SendBridleItemUseRequest @0x70e0c5 IS
      named, while OnBridleMobCatchFail is absent from the class entirely. v61
      has it (@0x8307f3, named), so this is a v48-only absence: v48 sends catch
      requests but has no server-driven failure notice. The sibling clientbound
      catch packets DO exist on v48 (CATCH_MONSTER 0x0AC, CATCH_MONSTER_WITH_ITEM
      0x0AD via CMobPool::OnMobPacket @0x559390 cases 172/173). (task-212)
```

- [ ] **Step 2: Generate the audit reports**

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_<v>.json \
  -ida-source docs/packets/ida-exports/<export>.json \
  -output /tmp/claude-1000/rpt-212
```

Run once per version key touched, then copy only the four report basenames listed above into `docs/packets/audits/<version>/`. Evidence without a report is a `matrix --check` "dangling evidence" failure.

- [ ] **Step 3: Pin evidence for the tier-1 cells**

```bash
go run ./tools/packet-audit evidence pin \
  --packet monster/serverbound/MonsterUseCatchItem --version gms_v48 \
  --ida "CWvsContext::SendBridleItemUseRequest" --category TIER1-FIXTURE
```

Repeat per (packet, version) cell being promoted. After each command, open the written `docs/packets/evidence/<version>/<packet dots>.yaml` and add the `verifies:` field by hand, e.g.:

```yaml
verifies:
  - libs/atlas-packet/monster/serverbound/use_catch_item_test.go#TestUseCatchItemBytes
```

Cells in scope (design §3): `USE_CATCH_ITEM` on all ten; `CATCH_MONSTER` on v48 and v92 plus re-verification of v61/v72/v79/v83/v84/v87/v95/jms after Task 4; `CATCH_MONSTER_WITH_ITEM` on v48, v61, v79, v92 plus re-verification of the previously-✅ cells after Task 5; `BRIDLE_MOB_CATCH_FAIL` on v61/v72/v79/v87.

- [ ] **Step 4: Regenerate and check**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Expected: exit 0, and `grep USE_CATCH_ITEM docs/packets/audits/STATUS.md` shows ✅ in all ten columns with no ⬜ remaining on that row.

- [ ] **Step 5: Record what did not promote**

If any cell in Step 3's list cannot be promoted from evidence actually in hand — the gms_v72 `CMob::OnEffectByItem` address is the known candidate, and gms_v72 `BRIDLE_MOB_CATCH_FAIL` may be another — first attempt the unblock: read the address out of `docs/packets/ida-exports/` or a live IDB via `func_query`/`decompile` (resolve the session from `idb_list` by binary **name**, pass it as `database`). Only if the IDB is genuinely unavailable, leave the cell at its current grade and append a short "not promoted, and why" note to this task's section of `context.md`. Never write a fixture whose bytes are not traceable to a decompile line.

- [ ] **Step 6: Commit**

```bash
git add docs/packets
git commit -m "docs(task-212): promote the catch-family coverage-matrix cells"
```

---

## Task 8: Atomic delete-and-claim in `libs/atlas-redis`, and `ClaimMonster`

**Files:**
- Modify: `libs/atlas-redis/registry.go`
- Test: `libs/atlas-redis/registry_test.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/registry.go`
- Test: `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `func (r *Registry[K, V]) RemoveExisting(ctx context.Context, key K) (bool, error)` in package `redis`.
  - `func (r *Registry) ClaimMonster(ctx context.Context, t tenant.Model, uniqueId uint32) (Model, bool, error)` in `atlas-monsters`' `monster` package — returns `(model, true, nil)` for the single winning caller, `(Model{}, false, nil)` for every loser and for a missing monster.

- [ ] **Step 1: Write the failing tests**

Append to `libs/atlas-redis/registry_test.go`:

```go
// TestRemoveExisting_ExactlyOneWinner is the exclusivity contract: N goroutines
// racing to delete the same key must see exactly one true. Redis DEL returns the
// number of keys removed, so the winner is decided server-side.
func TestRemoveExisting_ExactlyOneWinner(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	r := NewRegistry[string, string](client, "claim-test", func(s string) string { return s })
	ctx := context.Background()
	if err := r.Put(ctx, "k", "v"); err != nil {
		t.Fatalf("put: %v", err)
	}

	const racers = 16
	var wg sync.WaitGroup
	results := make([]bool, racers)
	wg.Add(racers)
	for i := 0; i < racers; i++ {
		go func(i int) {
			defer wg.Done()
			ok, rerr := r.RemoveExisting(ctx, "k")
			if rerr != nil {
				t.Errorf("racer %d: %v", i, rerr)
			}
			results[i] = ok
		}(i)
	}
	wg.Wait()

	won := 0
	for _, ok := range results {
		if ok {
			won++
		}
	}
	if won != 1 {
		t.Fatalf("RemoveExisting winners = %d, want exactly 1", won)
	}
}

// TestRemoveExisting_MissingKey reports false without error.
func TestRemoveExisting_MissingKey(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	r := NewRegistry[string, string](client, "claim-test", func(s string) string { return s })
	ok, err := r.RemoveExisting(context.Background(), "absent")
	if err != nil {
		t.Fatalf("RemoveExisting: %v", err)
	}
	if ok {
		t.Fatal("RemoveExisting on a missing key = true, want false")
	}
}
```

Add `"sync"` to the test file's imports if absent. Match the file's existing miniredis/goredis import aliases rather than introducing new ones.

Append to `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go`:

```go
// TestClaimMonster_ExactlyOneWinner — two concurrent catches on one monster
// must produce exactly one claim, and the loser must not see a model. This is
// the NFR-Race-safety guarantee that RemoveMonster (Get-then-Del, reply
// discarded) cannot provide.
func TestClaimMonster_ExactlyOneWinner(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300101, 0, 0, 0, 5, 0, 500, 100)

	const racers = 8
	var wg sync.WaitGroup
	claimed := make([]bool, racers)
	wg.Add(racers)
	for i := 0; i < racers; i++ {
		go func(i int) {
			defer wg.Done()
			_, ok, err := r.ClaimMonster(ctx, ten, m.UniqueId())
			if err != nil {
				t.Errorf("racer %d: %v", i, err)
			}
			claimed[i] = ok
		}(i)
	}
	wg.Wait()

	won := 0
	for _, ok := range claimed {
		if ok {
			won++
		}
	}
	if won != 1 {
		t.Fatalf("ClaimMonster winners = %d, want exactly 1", won)
	}
	if _, err := r.GetMonster(ten, m.UniqueId()); err == nil {
		t.Error("monster still present after a successful claim")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd libs/atlas-redis && go test ./... -run RemoveExisting -v
cd ../../services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run ClaimMonster -v
```
Expected: FAIL — `RemoveExisting`/`ClaimMonster` undefined.

- [ ] **Step 3: Implement `RemoveExisting`**

In `libs/atlas-redis/registry.go`, directly after `Remove`:

```go
// RemoveExisting deletes the key and reports whether it existed. Redis DEL is
// atomic and returns the number of keys removed, so under concurrency exactly
// one caller observes true — the primitive callers need when a removal must
// also be an exclusive claim (e.g. one monster, one catcher). Remove is
// deliberately left alone: its callers do not need the verdict and changing its
// signature would churn every one of them.
func (r *Registry[K, V]) RemoveExisting(ctx context.Context, key K) (bool, error) {
	rk := namespacedKey(r.namespace, r.keyFn(key))
	n, err := r.client.Del(ctx, rk).Result()
	if err != nil {
		return false, fmt.Errorf("redis del: %w", err)
	}
	return n == 1, nil
}
```

Confirm `fmt` is already imported in that file; add it if not.

- [ ] **Step 4: Implement `ClaimMonster`**

In `services/atlas-monsters/atlas.com/monsters/monster/registry.go`, directly after `RemoveMonster`:

```go
// ClaimMonster removes a monster and reports whether THIS caller was the one
// that removed it. Unlike RemoveMonster (Get-then-Del with the Del reply
// discarded), the delete is the claim: exactly one of N concurrent callers can
// observe ok=true, so exactly one catch attempt on a given monster can succeed.
// A missing monster and a lost race are both (Model{}, false, nil) — neither is
// an error, and neither should emit anything.
//
// The map-index removal and id release run ONLY for the winner, so a loser
// cannot recycle an id the winner still owns.
func (r *Registry) ClaimMonster(ctx context.Context, t tenant.Model, uniqueId uint32) (Model, bool, error) {
	sm, err := r.reg.Get(ctx, monsterSuffix(t, uniqueId))
	if errors.Is(err, atlasredis.ErrNotFound) {
		return Model{}, false, nil
	}
	if err != nil {
		return Model{}, false, err
	}
	_, m, err := fromStored(sm)
	if err != nil {
		return Model{}, false, err
	}

	claimed, err := r.reg.RemoveExisting(ctx, monsterSuffix(t, uniqueId))
	if err != nil {
		return Model{}, false, err
	}
	if !claimed {
		return Model{}, false, nil
	}

	_ = r.mapIdx.Remove(ctx, mapIndexSuffixFromModel(t, m), strconv.FormatUint(uint64(uniqueId), 10))
	GetIdAllocator().Release(ctx, t, uniqueId)
	return m, true, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd libs/atlas-redis && go test -race ./... && cd - >/dev/null
cd services/atlas-monsters/atlas.com/monsters && go test -race ./monster/ -run ClaimMonster -v && cd - >/dev/null
tools/redis-key-guard.sh
```
Expected: all clean, guard exit 0.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-redis services/atlas-monsters/atlas.com/monsters/monster/registry.go services/atlas-monsters/atlas.com/monsters/monster/registry_test.go
git commit -m "feat(task-212): atomic delete-and-claim for exactly-once monster removal"
```

---

## Task 9: atlas-monsters catch-item data client

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/monster/consumable/{requests.go,rest.go,model.go,processor.go}`
- Create: `services/atlas-monsters/atlas.com/monsters/monster/consumable/mock/processor.go`
- Test: `services/atlas-monsters/atlas.com/monsters/monster/consumable/rest_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: package `consumable` (import path `atlas-monsters/monster/consumable`) exposing
  `Processor` with `GetById(itemId uint32) (Model, error)`, `NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`,
  and `Model` with getters `Id() uint32`, `MonsterId() uint32`, `MonsterHp() uint32`, `Create() uint32`, `BridleProp() uint32`, `BridlePropChg() float64`,
  plus `NewModelBuilder()` returning `*ModelBuilder` with `SetId`/`SetMonsterId`/`SetMonsterHp`/`SetCreate`/`SetBridleProp`/`SetBridlePropChg`/`Build`.

Deliberately narrow: only the five fields the resolution ladder reads. Design §7 forbids caching, so this client has no cache layer (unlike `monster/information`).

- [ ] **Step 1: Write the failing test**

Create `services/atlas-monsters/atlas.com/monsters/monster/consumable/rest_test.go`:

```go
package consumable

import "testing"

// TestExtract maps the atlas-data consumable resource onto the five fields the
// catch ladder reads. The upstream resource is much wider; this client is
// deliberately narrow.
func TestExtract(t *testing.T) {
	rm := RestModel{
		Id:            2270002,
		Create:        4031868,
		MonsterId:     9300157,
		MonsterHP:     40,
		BridleProp:    50,
		BridlePropChg: 1.2,
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Id() != 2270002 || m.Create() != 4031868 || m.MonsterId() != 9300157 ||
		m.MonsterHp() != 40 || m.BridleProp() != 50 || m.BridlePropChg() != 1.2 {
		t.Fatalf("Extract produced %+v", m)
	}
}

// TestBuilder is the Builder-pattern seam the catch tests use for setup.
func TestBuilder(t *testing.T) {
	m := NewModelBuilder().SetId(2270000).SetMonsterId(9300101).SetCreate(1902000).Build()
	if m.MonsterHp() != 0 || m.BridleProp() != 0 {
		t.Fatalf("unset fields must be zero: %+v", m)
	}
	if m.Id() != 2270000 || m.MonsterId() != 9300101 || m.Create() != 1902000 {
		t.Fatalf("builder produced %+v", m)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/consumable/ -v`
Expected: FAIL — the package does not exist.

- [ ] **Step 3: Implement the package**

`monster/consumable/rest.go`:

```go
package consumable

import "strconv"

// RestModel is the narrow projection of atlas-data's consumable resource that
// the catch ladder needs. atlas-data returns many more fields; unmarshalling
// ignores the rest.
type RestModel struct {
	Id            uint32  `json:"-"`
	Create        uint32  `json:"create"`
	MonsterId     uint32  `json:"monsterId"`
	MonsterHP     uint32  `json:"monsterHP"`
	BridleProp    uint32  `json:"bridleProp"`
	BridlePropChg float64 `json:"bridlePropChg"`
}

func (r RestModel) GetName() string { return "consumables" }

func (r RestModel) GetID() string { return strconv.Itoa(int(r.Id)) }

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
		id:            rm.Id,
		create:        rm.Create,
		monsterId:     rm.MonsterId,
		monsterHp:     rm.MonsterHP,
		bridleProp:    rm.BridleProp,
		bridlePropChg: rm.BridlePropChg,
	}, nil
}
```

`monster/consumable/model.go`:

```go
package consumable

// Model is the immutable catch-item view. Construct it from REST via Extract or
// in tests via NewModelBuilder.
type Model struct {
	id            uint32
	create        uint32
	monsterId     uint32
	monsterHp     uint32
	bridleProp    uint32
	bridlePropChg float64
}

func (m Model) Id() uint32             { return m.id }
func (m Model) Create() uint32         { return m.create }
func (m Model) MonsterId() uint32      { return m.monsterId }
func (m Model) MonsterHp() uint32      { return m.monsterHp }
func (m Model) BridleProp() uint32     { return m.bridleProp }
func (m Model) BridlePropChg() float64 { return m.bridlePropChg }

type ModelBuilder struct {
	m Model
}

func NewModelBuilder() *ModelBuilder { return &ModelBuilder{} }

func (b *ModelBuilder) SetId(v uint32) *ModelBuilder             { b.m.id = v; return b }
func (b *ModelBuilder) SetCreate(v uint32) *ModelBuilder         { b.m.create = v; return b }
func (b *ModelBuilder) SetMonsterId(v uint32) *ModelBuilder      { b.m.monsterId = v; return b }
func (b *ModelBuilder) SetMonsterHp(v uint32) *ModelBuilder      { b.m.monsterHp = v; return b }
func (b *ModelBuilder) SetBridleProp(v uint32) *ModelBuilder     { b.m.bridleProp = v; return b }
func (b *ModelBuilder) SetBridlePropChg(v float64) *ModelBuilder { b.m.bridlePropChg = v; return b }
func (b *ModelBuilder) Build() Model                             { return b.m }
```

`monster/consumable/requests.go`:

```go
package consumable

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	consumablesResource = "data/consumables"
	consumableResource  = consumablesResource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(itemId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+consumableResource, itemId))
}
```

`monster/consumable/processor.go`:

```go
package consumable

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetById(itemId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetById resolves a catch item's WZ data for the request's tenant. There is
// deliberately NO cache: 0227.img differs by region and version and the reward
// ids differ with it, catch attempts are rare, and a tenant-keyed cache would
// buy nothing for the risk of serving one tenant another's reward id
// (task-212 design §7).
func (p *ProcessorImpl) GetById(itemId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(itemId), Extract)()
}
```

`monster/consumable/mock/processor.go` — mirror the shape of `monster/information/mock/processor.go` (read it first and match its style exactly):

```go
package mock

import (
	"atlas-monsters/monster/consumable"
)

type ProcessorImpl struct {
	GetByIdFunc func(itemId uint32) (consumable.Model, error)
}

func (p *ProcessorImpl) GetById(itemId uint32) (consumable.Model, error) {
	if p.GetByIdFunc != nil {
		return p.GetByIdFunc(itemId)
	}
	return consumable.Model{}, nil
}

var _ consumable.Processor = (*ProcessorImpl)(nil)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test -race ./monster/consumable/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/consumable
git commit -m "feat(task-212): add the atlas-monsters catch-item data client"
```

---

## Task 10: atlas-monsters `CATCH` command and resolution ladder

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/kafka.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/producer.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go`
- Create: `services/atlas-monsters/atlas.com/monsters/monster/catch.go`
- Create: `services/atlas-monsters/atlas.com/monsters/monster/catch_test.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go`

**Interfaces:**
- Consumes: `Registry.ClaimMonster` (Task 8), `atlas-monsters/monster/consumable` (Task 9).
- Produces:
  - Command: `EnvCommandTopic` type `"CATCH"` with body `{"characterId": uint32, "itemId": uint32}` and the shared `command[E]` envelope (`worldId`, `channelId`, `mapId`, `instance`, `monsterId` = the mob's **uniqueId**).
  - `Processor.Catch(uniqueId uint32, characterId uint32, itemId uint32)` on the `monster.Processor` interface.
  - Events on `EVENT_TOPIC_MONSTER_CATCH`: `CATCH_RESOLVED` body `{"characterId","itemId","success","cause"}`.
  - Events on `EVENT_TOPIC_MONSTER_STATUS`: `CAUGHT` body `{"characterId","itemId"}`, `CATCH_FAILED` body `{"characterId","itemId","cause"}`.
  - Cause constants: `CatchCauseSpeciesMismatch` = `"SPECIES_MISMATCH"`, `CatchCauseHpTooHigh` = `"HP_TOO_HIGH"`, `CatchCauseRollFailed` = `"ROLL_FAILED"`, `CatchCauseUnresolved` = `"UNRESOLVED"`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-monsters/atlas.com/monsters/monster/catch_test.go`:

```go
package monster

import (
	"atlas-monsters/monster/consumable"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// withCatchItem installs the test-only consumable lookup and returns a cleanup.
func withCatchItem(t *testing.T, m consumable.Model, err error) func() {
	t.Helper()
	prev := testConsumableLookup
	testConsumableLookup = func(_ uint32) (consumable.Model, error) { return m, err }
	return func() { testConsumableLookup = prev }
}

func spawnCatchable(t *testing.T, ten tenant.Model, monsterId uint32, hp uint32, maxHp uint32) uint32 {
	t.Helper()
	r := GetMonsterRegistry()
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(context.Background(), ten, f, monsterId, 0, 0, 0, 5, 0, maxHp, 100)
	if hp != maxHp {
		if _, err := r.ApplyDamage(ten, m.UniqueId(), 1, maxHp-hp); err != nil {
			t.Fatalf("apply damage: %v", err)
		}
	}
	return m.UniqueId()
}

func eventTypes(events *[]emittedBody) []string {
	var out []string
	for _, e := range *events {
		out = append(out, e.Type)
	}
	return out
}

// TestCatch_Success — species matches, HP under the gate, no bridleProp (so the
// roll is a deterministic pass): the monster is claimed and removed, and the
// three events fire in the order CATCH_RESOLVED, CAUGHT, DESTROYED.
func TestCatch_Success(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	GetMonsterRegistry().Clear(context.Background())
	defer withCatchItem(t, consumable.NewModelBuilder().
		SetId(2270000).SetMonsterId(9300101).SetCreate(1902000).Build(), nil)()

	uniqueId := spawnCatchable(t, ten, 9300101, 100, 100)
	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Catch(uniqueId, 42, 2270000)

	if got := eventTypes(events); len(got) != 3 ||
		got[0] != EventMonsterCatchResolved || got[1] != EventMonsterStatusCaught || got[2] != EventMonsterStatusDestroyed {
		t.Fatalf("event order = %v, want [CATCH_RESOLVED CAUGHT DESTROYED]", eventTypes(events))
	}
	if (*events)[0].Topic != EnvEventTopicMonsterCatch {
		t.Errorf("CATCH_RESOLVED topic = %q, want %q", (*events)[0].Topic, EnvEventTopicMonsterCatch)
	}
	var body catchResolvedBody
	if err := json.Unmarshal((*events)[0].Body, &body); err != nil {
		t.Fatalf("decode CATCH_RESOLVED: %v", err)
	}
	if !body.Success || body.CharacterId != 42 || body.ItemId != 2270000 || body.Cause != "" {
		t.Fatalf("CATCH_RESOLVED body = %+v", body)
	}
	if _, err := GetMonsterRegistry().GetMonster(ten, uniqueId); err == nil {
		t.Error("monster still present after a successful catch")
	}
}

// TestCatch_SpeciesMismatch — the item names a different mob: failure, monster
// untouched, no KILLED and no DESTROYED (no experience, no drops, no death).
func TestCatch_SpeciesMismatch(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	GetMonsterRegistry().Clear(context.Background())
	defer withCatchItem(t, consumable.NewModelBuilder().
		SetId(2270000).SetMonsterId(9300101).SetCreate(1902000).Build(), nil)()

	uniqueId := spawnCatchable(t, ten, 9500197, 100, 100)
	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Catch(uniqueId, 42, 2270000)

	if got := eventTypes(events); len(got) != 2 ||
		got[0] != EventMonsterCatchResolved || got[1] != EventMonsterStatusCatchFailed {
		t.Fatalf("event order = %v, want [CATCH_RESOLVED CATCH_FAILED]", eventTypes(events))
	}
	var body catchResolvedBody
	_ = json.Unmarshal((*events)[0].Body, &body)
	if body.Success || body.Cause != CatchCauseSpeciesMismatch {
		t.Fatalf("CATCH_RESOLVED body = %+v, want success=false cause=SPECIES_MISMATCH", body)
	}
	if _, err := GetMonsterRegistry().GetMonster(ten, uniqueId); err != nil {
		t.Error("monster removed on a species mismatch")
	}
}

// TestCatch_HpGate — mobHP is a PERCENTAGE of max HP (design A-1). The
// cross-multiplied comparison must admit hp exactly at the boundary and reject
// one point above it, and mobHP=100 must admit a FULL-HP monster (the case
// integer truncation would break).
func TestCatch_HpGate(t *testing.T) {
	cases := []struct {
		name   string
		mobHP  uint32
		hp     uint32
		maxHp  uint32
		expect bool
	}{
		{"at the 40% boundary", 40, 400, 1000, true},
		{"one point above 40%", 40, 401, 1000, false},
		{"well under 30%", 30, 100, 1000, true},
		{"full HP at mobHP=100", 100, 1000, 1000, true},
		{"no gate when mobHP is zero", 0, 1000, 1000, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			GetMonsterRegistry().Clear(context.Background())
			defer withCatchItem(t, consumable.NewModelBuilder().
				SetId(2270005).SetMonsterId(9300187).SetCreate(2109001).SetMonsterHp(tc.mobHP).Build(), nil)()

			uniqueId := spawnCatchable(t, ten, 9300187, tc.hp, tc.maxHp)
			p, events := newRecordingProcessorWithBodies(t, ten)
			p.Catch(uniqueId, 42, 2270005)

			var body catchResolvedBody
			_ = json.Unmarshal((*events)[0].Body, &body)
			if body.Success != tc.expect {
				t.Fatalf("success = %t, want %t (cause %q)", body.Success, tc.expect, body.Cause)
			}
			if !tc.expect && body.Cause != CatchCauseHpTooHigh {
				t.Fatalf("cause = %q, want HP_TOO_HIGH", body.Cause)
			}
		})
	}
}

// TestCatch_RollFailed — a seeded roller that always loses produces ROLL_FAILED
// and leaves the monster alive.
func TestCatch_RollFailed(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	GetMonsterRegistry().Clear(context.Background())
	defer withCatchItem(t, consumable.NewModelBuilder().
		SetId(2270002).SetMonsterId(9300157).SetCreate(4031868).SetBridleProp(50).SetBridlePropChg(1.2).Build(), nil)()

	prevRoll := testCatchRoll
	testCatchRoll = func(chance uint32) (bool, error) {
		if chance != 60 {
			t.Errorf("effective chance = %d, want 60 (50 * 1.2, rounded)", chance)
		}
		return false, nil
	}
	defer func() { testCatchRoll = prevRoll }()

	uniqueId := spawnCatchable(t, ten, 9300157, 100, 100)
	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Catch(uniqueId, 42, 2270002)

	var body catchResolvedBody
	_ = json.Unmarshal((*events)[0].Body, &body)
	if body.Success || body.Cause != CatchCauseRollFailed {
		t.Fatalf("CATCH_RESOLVED body = %+v, want ROLL_FAILED", body)
	}
	if _, err := GetMonsterRegistry().GetMonster(ten, uniqueId); err != nil {
		t.Error("monster removed on a failed roll")
	}
}

// TestCatch_MonsterGone — a redelivered command whose monster is already gone
// emits CATCH_RESOLVED(false, UNRESOLVED) so the reservation is cancelled, and
// CATCH_FAILED(UNRESOLVED) so the client unlocks. It grants nothing.
func TestCatch_MonsterGone(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	GetMonsterRegistry().Clear(context.Background())
	defer withCatchItem(t, consumable.NewModelBuilder().
		SetId(2270000).SetMonsterId(9300101).SetCreate(1902000).Build(), nil)()

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Catch(999999, 42, 2270000)

	if got := eventTypes(events); len(got) != 2 ||
		got[0] != EventMonsterCatchResolved || got[1] != EventMonsterStatusCatchFailed {
		t.Fatalf("event order = %v, want [CATCH_RESOLVED CATCH_FAILED]", eventTypes(events))
	}
	var body catchResolvedBody
	_ = json.Unmarshal((*events)[0].Body, &body)
	if body.Success || body.Cause != CatchCauseUnresolved {
		t.Fatalf("CATCH_RESOLVED body = %+v, want UNRESOLVED", body)
	}
}

// TestCatch_ConcurrentAttempts_OneCaught — two players catching the same monster
// concurrently must produce exactly one CAUGHT (NFR-Race-safety). The loser
// reports UNRESOLVED, which cancels its reservation and unlocks its client
// without a failure notice. Each processor records into its own event slice, so
// the recorder itself is not shared state.
func TestCatch_ConcurrentAttempts_OneCaught(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	GetMonsterRegistry().Clear(context.Background())
	defer withCatchItem(t, consumable.NewModelBuilder().
		SetId(2270000).SetMonsterId(9300101).SetCreate(1902000).Build(), nil)()

	uniqueId := spawnCatchable(t, ten, 9300101, 100, 100)

	const racers = 8
	recorders := make([]*[]emittedBody, racers)
	var wg sync.WaitGroup
	wg.Add(racers)
	for i := 0; i < racers; i++ {
		p, events := newRecordingProcessorWithBodies(t, ten)
		recorders[i] = events
		go func(p *ProcessorImpl, characterId uint32) {
			defer wg.Done()
			p.Catch(uniqueId, characterId, 2270000)
		}(p, uint32(100+i))
	}
	wg.Wait()

	caught := 0
	for _, events := range recorders {
		for _, e := range *events {
			if e.Type == EventMonsterStatusCaught {
				caught++
			}
		}
	}
	if caught != 1 {
		t.Fatalf("CAUGHT events = %d, want exactly 1", caught)
	}
}

// TestCatch_LookupFailure — fail-closed, exactly like Kill's boss lookup: a data
// error drops the command entirely rather than reporting a catch failure.
func TestCatch_LookupFailure(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	GetMonsterRegistry().Clear(context.Background())
	defer withCatchItem(t, consumable.Model{}, errors.New("upstream down"))()

	uniqueId := spawnCatchable(t, ten, 9300101, 100, 100)
	p, events := newRecordingProcessorWithBodies(t, ten)
	p.Catch(uniqueId, 42, 2270000)

	if len(*events) != 0 {
		t.Fatalf("expected no events on a data lookup failure, got %v", eventTypes(events))
	}
	if _, err := GetMonsterRegistry().GetMonster(ten, uniqueId); err != nil {
		t.Error("monster removed despite a data lookup failure")
	}
}
```

If `Registry.ApplyDamage`'s signature differs from what `spawnCatchable` assumes, read `registry.go` and use the real one — the goal is only "spawn a monster at a chosen HP".

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run TestCatch -v`
Expected: FAIL — `p.Catch` undefined, cause constants undefined.

- [ ] **Step 3: Add the event contract**

In `monster/kafka.go`, extend the const block:

```go
	EnvEventTopicMonsterCatch = "EVENT_TOPIC_MONSTER_CATCH"

	EventMonsterCatchResolved = "CATCH_RESOLVED"

	EventMonsterStatusCaught      = "CAUGHT"
	EventMonsterStatusCatchFailed = "CATCH_FAILED"

	// Catch failure causes. The wire collapses all of them to a single byte
	// (design §6.4), so these survive only in logs and in the channel's
	// cause -> wire-reason mapping. UNRESOLVED means the attempt lost a race
	// (monster gone, or another catcher claimed it) — the channel renders no
	// failure packet for it, only the unlock.
	CatchCauseSpeciesMismatch = "SPECIES_MISMATCH"
	CatchCauseHpTooHigh       = "HP_TOO_HIGH"
	CatchCauseRollFailed      = "ROLL_FAILED"
	CatchCauseUnresolved      = "UNRESOLVED"
```

and add the three bodies next to the other `statusEvent*Body` types:

```go
// catchResolvedBody is the economic outcome, published on the dedicated
// low-volume EVENT_TOPIC_MONSTER_CATCH. atlas-consumables consumes it to commit
// or cancel the item reservation; it must NOT be published on the status topic,
// which carries a DAMAGED event per hit and whose every handler unmarshals every
// message (design §4.2).
type catchResolvedBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
	Success     bool   `json:"success"`
	Cause       string `json:"cause"`
}

// statusEventCaughtBody is the presentation outcome, published on the status
// topic immediately BEFORE DESTROYED. The status topic is keyed by MapId, so
// that ordering is a partition guarantee — which matters because
// CMobPool::OnMobPacket resolves the mob via GetMob and silently drops the
// effect packet if the mob is already gone.
type statusEventCaughtBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
}

type statusEventCatchFailedBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
	Cause       string `json:"cause"`
}
```

In `monster/producer.go`, add the three providers:

```go
// catchResolvedEventProvider keys on the character, not the map: the dedicated
// catch topic exists for atlas-consumables, whose ordering concern is
// per-character reservation handling.
func catchResolvedEventProvider(m Model, characterId uint32, itemId uint32, success bool, cause string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := statusEventFromField(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterCatchResolved, catchResolvedBody{
		CharacterId: characterId,
		ItemId:      itemId,
		Success:     success,
		Cause:       cause,
	})
	return producer.SingleMessageProvider(key, &value)
}

func caughtStatusEventProvider(m Model, characterId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusCaught, statusEventCaughtBody{
		CharacterId: characterId,
		ItemId:      itemId,
	})
}

func catchFailedStatusEventProvider(m Model, characterId uint32, itemId uint32, cause string) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusCatchFailed, statusEventCatchFailedBody{
		CharacterId: characterId,
		ItemId:      itemId,
		Cause:       cause,
	})
}
```

- [ ] **Step 4: Implement the ladder**

Create `services/atlas-monsters/atlas.com/monsters/monster/catch.go`:

```go
package monster

import (
	"atlas-monsters/monster/consumable"
	"crypto/rand"
	"math"
	"math/big"
)

// testConsumableLookup is a test-only override for the catch-item data lookup,
// mirroring testInformationLookup. Nil in production.
var testConsumableLookup func(itemId uint32) (consumable.Model, error)

// testCatchRoll is a test-only override for the probability roll. Nil in
// production, where rollCatch uses crypto/rand.
var testCatchRoll func(chance uint32) (bool, error)

// effectiveCatchChance applies bridlePropChg as a ONE-SHOT multiplier on
// bridleProp, clamped to 100 (design assumption A-2). Both values are
// server-side WZ data the client never reads, so no IDB can settle this; a
// per-attempt escalation was rejected because it would need per-(character,
// monster) state nothing else in the codebase keeps. A zero bridleProp means
// the item is deterministic once species and HP pass (FR-3.5).
func effectiveCatchChance(prop uint32, chg float64) uint32 {
	if prop == 0 {
		return 0
	}
	if chg <= 0 {
		return minChance(prop)
	}
	return minChance(uint32(math.Round(float64(prop) * chg)))
}

func minChance(v uint32) uint32 {
	if v > 100 {
		return 100
	}
	return v
}

// rollCatch draws [0,100) from a CSPRNG and reports whether it beat the chance,
// the same shape rollReward uses in atlas-consumables.
func rollCatch(chance uint32) (bool, error) {
	if testCatchRoll != nil {
		return testCatchRoll(chance)
	}
	if chance >= 100 {
		return true, nil
	}
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		return false, err
	}
	return uint32(n.Int64()) < chance, nil
}

// catchHpGatePasses reports whether the monster is weak enough. mobHP is a
// PERCENTAGE of max HP (design assumption A-1 — the client never performs this
// check, so it cannot be read from any IDB). The comparison is cross-multiplied
// precisely so integer truncation cannot let a full-HP monster through at
// mobHP < 100: hp <= maxHp * mobHP / 100 becomes hp * 100 <= maxHp * mobHP.
// A zero mobHP means no gate.
func catchHpGatePasses(hp uint32, maxHp uint32, mobHP uint32) bool {
	if mobHP == 0 {
		return true
	}
	return uint64(hp)*100 <= uint64(maxHp)*uint64(mobHP)
}

// Catch resolves a bridle (catch-item) capture attempt. It is fail-closed and
// authoritative: atlas-consumables validated the ITEM, but every monster-state
// check happens here, exactly as Kill re-checks alive+boss rather than trusting
// the caller.
//
// Emission contract:
//   - success: CATCH_RESOLVED(true) -> CAUGHT -> DESTROYED. The economic
//     outcome goes first so it is the first thing attempted after the claim.
//   - a check failed: CATCH_RESOLVED(false, cause) + CATCH_FAILED(cause).
//     Nothing is removed and no KILLED/DESTROYED fires — a catch awards no
//     experience, rolls no drops, and emits no death events (FR-3.6).
//   - monster gone or claim lost: CATCH_RESOLVED(false, UNRESOLVED) +
//     CATCH_FAILED(UNRESOLVED). The resolved event is what cancels the caller's
//     reservation; the channel renders no failure packet for UNRESOLVED, only
//     the unlock. A redelivery is harmless because the caller's once-handler has
//     already deregistered.
//   - data lookup failed: nothing at all (fail-closed).
func (p *ProcessorImpl) Catch(uniqueId uint32, characterId uint32, itemId uint32) {
	m, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)
	if err != nil || !m.Alive() {
		p.l.Debugf("CATCH: monster [%d] not found or already dead; reporting unresolved for character [%d].", uniqueId, characterId)
		p.emitCatchUnresolved(uniqueId, characterId, itemId)
		return
	}

	var ci consumable.Model
	var ciErr error
	if testConsumableLookup != nil {
		ci, ciErr = testConsumableLookup(itemId)
	} else {
		ci, ciErr = consumable.NewProcessor(p.l, p.ctx).GetById(itemId)
	}
	if ciErr != nil {
		p.l.WithError(ciErr).Errorf("CATCH: catch-item [%d] lookup failed; dropping (fail-closed).", itemId)
		return
	}

	if m.MonsterId() != ci.MonsterId() {
		p.l.Debugf("CATCH: item [%d] targets mob [%d] but monster [%d] is mob [%d].", itemId, ci.MonsterId(), uniqueId, m.MonsterId())
		p.emitCatchFailure(m, characterId, itemId, CatchCauseSpeciesMismatch)
		return
	}
	if !catchHpGatePasses(m.Hp(), m.MaxHp(), ci.MonsterHp()) {
		p.l.Debugf("CATCH: monster [%d] hp [%d]/[%d] above the [%d]%% gate for item [%d].", uniqueId, m.Hp(), m.MaxHp(), ci.MonsterHp(), itemId)
		p.emitCatchFailure(m, characterId, itemId, CatchCauseHpTooHigh)
		return
	}
	if chance := effectiveCatchChance(ci.BridleProp(), ci.BridlePropChg()); chance > 0 {
		won, rerr := rollCatch(chance)
		if rerr != nil {
			p.l.WithError(rerr).Errorf("CATCH: roll failed for item [%d]; dropping (fail-closed).", itemId)
			return
		}
		if !won {
			p.l.Debugf("CATCH: monster [%d] roll failed at [%d]%% for item [%d].", uniqueId, chance, itemId)
			p.emitCatchFailure(m, characterId, itemId, CatchCauseRollFailed)
			return
		}
	}

	claimed, ok, cerr := GetMonsterRegistry().ClaimMonster(p.ctx, p.t, uniqueId)
	if cerr != nil {
		p.l.WithError(cerr).Errorf("CATCH: claim failed for monster [%d]; dropping (fail-closed).", uniqueId)
		return
	}
	if !ok {
		p.l.Debugf("CATCH: monster [%d] claim lost by character [%d].", uniqueId, characterId)
		p.emitCatchUnresolved(uniqueId, characterId, itemId)
		return
	}

	GetDropTimerRegistry().Unregister(p.ctx, p.t, uniqueId)
	GetAttackCooldownRegistry().ClearCooldowns(p.ctx, p.t, uniqueId)

	_ = p.emit(EnvEventTopicMonsterCatch, catchResolvedEventProvider(claimed, characterId, itemId, true, ""))
	_ = p.emit(EnvEventTopicMonsterStatus, caughtStatusEventProvider(claimed, characterId, itemId))
	_ = p.emit(EnvEventTopicMonsterStatus, destroyedStatusEventProvider(claimed))
}

func (p *ProcessorImpl) emitCatchFailure(m Model, characterId uint32, itemId uint32, cause string) {
	_ = p.emit(EnvEventTopicMonsterCatch, catchResolvedEventProvider(m, characterId, itemId, false, cause))
	_ = p.emit(EnvEventTopicMonsterStatus, catchFailedStatusEventProvider(m, characterId, itemId, cause))
}

// emitCatchUnresolved reports a lost race. The monster model is gone, so the
// events carry a bare field-less model built from the uniqueId alone; the
// consumers key on characterId and itemId, not on the field.
func (p *ProcessorImpl) emitCatchUnresolved(uniqueId uint32, characterId uint32, itemId uint32) {
	m := Model{uniqueId: uniqueId}
	p.emitCatchFailure(m, characterId, itemId, CatchCauseUnresolved)
}
```

If `Model`'s `uniqueId` field is not settable from within the package that way, use the package's own Builder instead — read `monster/model.go` and use whatever the package already provides. Do not export a new setter.

Add `Catch(uniqueId uint32, characterId uint32, itemId uint32)` to the `Processor` interface in `monster/processor.go`, immediately after `Kill`.

- [ ] **Step 5: Wire the command**

In `kafka/consumer/monster/kafka.go`, add `CommandTypeCatch = "CATCH"` to the const block and the body:

```go
// catchCommandBody asks the processor to remove a monster as a bridle
// (catch-item) capture. Deliberately minimal: every handler on this shared
// command topic json-unmarshals every message, so a field name whose type
// disagrees with a sibling body produces one spurious unmarshal error per
// message. characterId and itemId are both uint32 and both already appear with
// that type in sibling bodies (damageCommandBody.CharacterId,
// drainMpCommandBody.SkillId).
type catchCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
}
```

In `kafka/consumer/monster/consumer.go`, add the handler next to `handleKillCommand`:

```go
func handleCatchCommand(l logrus.FieldLogger, ctx context.Context, c command[catchCommandBody]) {
	if c.Type != CommandTypeCatch {
		return
	}

	p := monster.NewProcessor(l, ctx)
	p.Catch(c.MonsterId, c.Body.CharacterId, c.Body.ItemId)
}
```

and register it in `InitHandlers` immediately after the `handleKillCommand` registration:

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCatchCommand))); err != nil {
			return err
		}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd services/atlas-monsters/atlas.com/monsters
go test -race ./... && go vet ./... && go build ./...
```
Expected: all clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-monsters
git commit -m "feat(task-212): CATCH command and reward-free catch resolution in atlas-monsters"
```

---

## Task 11a: atlas-consumables data getters and the catch-item classification

**Files:**
- Modify: `libs/atlas-constants/item/constants.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/data/consumable/model.go`
- Test: `services/atlas-consumables/atlas.com/consumables/data/consumable/model_test.go` (create if absent)

**Interfaces:**
- Consumes: nothing.
- Produces: `item.ClassificationConsumableCatchItem` (= `Classification(227)`); and on `consumable.Model`: `Create() uint32`, `MonsterId() uint32`, `MonsterHp() uint32`, `BridleMsgType() uint32`, `BridleProp() uint32`, `BridlePropChg() float64`, `UseDelay() uint32`, `DelayMsg() string`.

The struct fields already exist and REST already populates them (`data/consumable/rest.go:132-145`); only the accessors are missing.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-consumables/atlas.com/consumables/data/consumable/model_test.go`:

```go
package consumable

import "testing"

// TestCatchFieldAccessors covers the bridle fields the catch flow reads. They
// were already parsed and carried over REST; only the getters were missing.
func TestCatchFieldAccessors(t *testing.T) {
	rm := RestModel{
		Id:            2270008,
		Create:        2022323,
		MonsterId:     9500336,
		MonsterHP:     0,
		BridleMsgType: 4,
		BridleProp:    50,
		BridlePropChg: 1.2,
		UseDelay:      3000,
		DelayMsg:      "You cannot use the Fishing Net yet.",
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Create() != 2022323 || m.MonsterId() != 9500336 || m.MonsterHp() != 0 ||
		m.BridleMsgType() != 4 || m.BridleProp() != 50 || m.BridlePropChg() != 1.2 ||
		m.UseDelay() != 3000 || m.DelayMsg() != "You cannot use the Fishing Net yet." {
		t.Fatalf("accessors returned %+v", m)
	}
}
```

If `Extract`'s name or signature in this package differs, read `rest.go` and call the real one.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./data/consumable/ -v`
Expected: FAIL — `m.Create undefined` etc.

- [ ] **Step 3: Implement**

In `libs/atlas-constants/item/constants.go`, add to the consumable classification block, in numeric order (between 226 and 228):

```go
	ClassificationConsumableCatchItem      = Classification(227)
```

In `services/atlas-consumables/atlas.com/consumables/data/consumable/model.go`, append the accessors next to the existing ones:

```go
// Create is the item id a successful use produces — for a catch item, the
// reward granted in the caught monster's place.
func (m Model) Create() uint32 {
	return m.create
}

// MonsterId is the mob template a catch item targets (WZ info/mob).
func (m Model) MonsterId() uint32 {
	return m.monsterId
}

// MonsterHp is the catch HP gate (WZ info/mobHP), interpreted as a PERCENTAGE
// of the target's max HP (task-212 assumption A-1). Zero means no gate.
func (m Model) MonsterHp() uint32 {
	return m.monsterHp
}

// BridleMsgType selects the CLIENT-side "no monster found" message and is never
// read off the wire by either catch response packet (task-212 design §6.3).
func (m Model) BridleMsgType() uint32 {
	return m.bridleMsgType
}

// BridleProp is the base catch success percentage. Zero means deterministic
// once the species and HP gates pass.
func (m Model) BridleProp() uint32 {
	return m.bridleProp
}

// BridlePropChg multiplies BridleProp once, statelessly (assumption A-2).
func (m Model) BridlePropChg() float64 {
	return m.bridlePropChg
}

// UseDelay is the per-item cooldown in milliseconds (WZ info/useDelay).
func (m Model) UseDelay() uint32 {
	return m.useDelay
}

// DelayMsg is what BRIDLE_MOB_CATCH_FAIL reason 1 renders.
func (m Model) DelayMsg() string {
	return m.delayMsg
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd libs/atlas-constants && go build ./... && cd - >/dev/null
cd services/atlas-consumables/atlas.com/consumables && go test -race ./data/consumable/ -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-constants/item/constants.go services/atlas-consumables/atlas.com/consumables/data/consumable
git commit -m "feat(task-212): expose the bridle consumable fields and the 227 classification"
```

---

## Task 11b: atlas-consumables request path — validation, useDelay, reservation

**Files:**
- Create: `services/atlas-consumables/atlas.com/consumables/catchdelay/registry.go`
- Create: `services/atlas-consumables/atlas.com/consumables/consumable/catch.go`
- Create: `services/atlas-consumables/atlas.com/consumables/consumable/catch_test.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/monster/{processor.go,producer.go}` (new file `producer.go`)
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/message/monster/kafka.go` (new)
- Modify: `services/atlas-consumables/atlas.com/consumables/main.go`

**Interfaces:**
- Consumes: `consumable.Model` getters (Task 11a); `item.ClassificationConsumableCatchItem` (Task 11a); the atlas-monsters `CATCH` command contract (Task 10).
- Produces:
  - On the `consumable.Processor` interface: `RequestCatchMonster(f field.Model, characterId uint32, slot int16, itemId item2.Id, monsterUniqueId uint32) error`.
  - Command constant `CommandRequestCatchMonster = "REQUEST_CATCH_MONSTER"` and body `RequestCatchMonsterBody{Source slot.Position, ItemId item.Id, MonsterUniqueId uint32}`.
  - Event constant `EventTypeCatchFailed = "CATCH_FAILED"` and body `CatchFailedBody{ItemId uint32, Cause string}` on `EVENT_TOPIC_CONSUMABLE_STATUS`.
  - Cause constants `CatchCauseUseDelay = "USE_DELAY"`, `CatchCauseInventoryFull = "INVENTORY_FULL"`, `CatchCauseInvalidItem = "INVALID_ITEM"`.
  - `catchdelay.InitRegistry(client *goredis.Client)`, `catchdelay.GetRegistry() *Registry`, and on it `Allow(ctx context.Context, characterId uint32, itemId uint32, delay time.Duration) (bool, error)`.
  - `monster.Processor.RequestCatch(f field.Model, monsterUniqueId uint32, characterId uint32, itemId uint32) error`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-consumables/atlas.com/consumables/consumable/catch_test.go`:

```go
package consumable

import (
	consumable3 "atlas-consumables/data/consumable"
	"testing"
)

// TestValidateCatchItem is the pre-reserve gate: only class-227 items with a
// non-zero create id may proceed. Everything else is rejected before the
// inventory is touched (FR-3.2).
func TestValidateCatchItem(t *testing.T) {
	cases := []struct {
		name   string
		itemId uint32
		create uint32
		wantOk bool
	}{
		{"a catch item with a reward", 2270000, 1902000, true},
		{"a catch item with no create id", 2270000, 0, false},
		{"a red potion", 2000000, 1902000, false},
		{"a revitalizer", 2260000, 1902000, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ci, _ := consumable3.Extract(consumable3.RestModel{Id: tc.itemId, Create: tc.create})
			if got := validateCatchItem(tc.itemId, ci); got != tc.wantOk {
				t.Fatalf("validateCatchItem = %t, want %t", got, tc.wantOk)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run ValidateCatchItem -v`
Expected: FAIL — `undefined: validateCatchItem`.

- [ ] **Step 3: Add the useDelay registry**

Create `services/atlas-consumables/atlas.com/consumables/catchdelay/registry.go`:

```go
// Package catchdelay enforces a catch item's WZ useDelay server-side. The delay
// is server-enforced because BRIDLE_MOB_CATCH_FAIL reason 1 renders the item's
// delayMsg (design §6.4) — without enforcement, reason 1 would be unreachable
// and delayMsg dead data.
package catchdelay

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type key struct {
	characterId uint32
	itemId      uint32
}

type Registry struct {
	reg *atlas.TenantRegistry[key, bool]
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTenantRegistry[key, bool](client, "consumable-catch-delay", func(k key) string {
			return fmt.Sprintf("%d:%d", k.characterId, k.itemId)
		}),
	}
}

func GetRegistry() *Registry { return registry }

// Allow reports whether a catch attempt may proceed and, when it may, arms the
// cooldown. The window is armed on EVERY admitted attempt, success or failure —
// the client's own 200ms ExclRequest floor is a separate concern and is not
// replicated here. A zero delay always admits and arms nothing.
func (r *Registry) Allow(ctx context.Context, characterId uint32, itemId uint32, delay time.Duration) (bool, error) {
	if r == nil || delay <= 0 {
		return true, nil
	}
	t := tenant.MustFromContext(ctx)
	k := key{characterId: characterId, itemId: itemId}

	exists, err := r.reg.Exists(ctx, t, k)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	if err := r.reg.PutWithTTL(ctx, t, k, true, delay); err != nil {
		return false, err
	}
	return true, nil
}
```

Call `catchdelay.InitRegistry(rc)` in `main.go` alongside the existing `map/character` registry init (read `main.go` for the redis client variable name and mirror it exactly).

- [ ] **Step 4: Add the Kafka contracts**

In `kafka/message/consumable/kafka.go`, add to the command block:

```go
	CommandRequestCatchMonster = "REQUEST_CATCH_MONSTER"
```

```go
// RequestCatchMonsterBody carries a bridle (catch-item) use. monsterUniqueId is
// the field object id the client's FindHitMobInRect selected — the server
// revalidates species, HP and the roll in atlas-monsters regardless.
type RequestCatchMonsterBody struct {
	Source          slot.Position `json:"source"`
	ItemId          item.Id       `json:"itemId"`
	MonsterUniqueId uint32        `json:"monsterUniqueId"`
}
```

and to the event block:

```go
	EventTypeCatchFailed = "CATCH_FAILED"

	// Catch failure causes reported by atlas-consumables' pre-reserve gates.
	// The wire byte is NOT chosen here — atlas-channel maps cause to reason
	// (DOM-25), because 0/1 is a client-interpreted value.
	CatchCauseUseDelay      = "USE_DELAY"
	CatchCauseInventoryFull = "INVENTORY_FULL"
	CatchCauseInvalidItem   = "INVALID_ITEM"
```

```go
type CatchFailedBody struct {
	ItemId uint32 `json:"itemId"`
	Cause  string `json:"cause"`
}
```

Create `kafka/message/monster/kafka.go` in atlas-consumables — the outbound `CATCH` command mirror:

```go
package monster

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic  = "COMMAND_TOPIC_MONSTER"
	CommandTypeCatch = "CATCH"
)

// Command mirrors atlas-monsters' shared command envelope. MonsterId carries
// the mob's unique (field object) id, matching every sibling command.
type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// CatchCommandBody is deliberately minimal: every handler on the shared monster
// command topic unmarshals every message, so a field whose type disagrees with a
// sibling body logs one spurious error per message.
type CatchCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
}
```

Create `monster/producer.go` in atlas-consumables:

```go
package monster

import (
	monsterMsg "atlas-consumables/kafka/message/monster"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// catchCommandProvider keys on the monster's unique id so concurrent catch
// attempts on one mob land on a single partition and are resolved in order.
func catchCommandProvider(f field.Model, monsterUniqueId uint32, characterId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterUniqueId))
	value := &monsterMsg.Command[monsterMsg.CatchCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterUniqueId,
		Type:      monsterMsg.CommandTypeCatch,
		Body: monsterMsg.CatchCommandBody{
			CharacterId: characterId,
			ItemId:      itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

and add to `monster/processor.go` in atlas-consumables (interface + impl):

```go
	RequestCatch(f field.Model, monsterUniqueId uint32, characterId uint32, itemId uint32) error
```

```go
func (p *ProcessorImpl) RequestCatch(f field.Model, monsterUniqueId uint32, characterId uint32, itemId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(monsterMsg.EnvCommandTopic)(catchCommandProvider(f, monsterUniqueId, characterId, itemId))
}
```

- [ ] **Step 5: Implement the request path**

Create `services/atlas-consumables/atlas.com/consumables/consumable/catch.go`:

```go
package consumable

import (
	consumable3 "atlas-consumables/data/consumable"
	"time"

	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// validateCatchItem is the pre-reserve item gate (FR-3.2): the item must be a
// class-227 bridle consumable AND name a reward to grant. Classification comes
// from libs/atlas-constants rather than an ad-hoc itemId/10000 == 227 (DOM-21).
func validateCatchItem(itemId uint32, ci consumable3.Model) bool {
	if item2.GetClassification(item2.Id(itemId)) != item2.ClassificationConsumableCatchItem {
		return false
	}
	return ci.Create() != 0
}

func catchUseDelay(ci consumable3.Model) time.Duration {
	return time.Duration(ci.UseDelay()) * time.Millisecond
}
```

Add `RequestCatchMonster(f field.Model, characterId uint32, slot int16, itemId item2.Id, monsterUniqueId uint32) error` to the `Processor` interface in `consumable/processor.go`, beside `RequestItemReward`, and implement it in `consumable/catch.go`:

```go
// RequestCatchMonster begins a bridle (catch-item) attempt: validate the item,
// arm the useDelay window, confirm the reward can be placed, then reserve the
// item and hand the monster-state decision to atlas-monsters. Modelled on
// RequestItemReward (processor.go:1079) — one transactionId spans
// reserve -> resolve -> commit.
//
// The item is NOT consumed here. Commit happens only on
// CATCH_RESOLVED(success=true), which satisfies "a failed catch does not consume
// the item" (FR-3.9) by construction rather than by compensation.
func (p *ProcessorImpl) RequestCatchMonster(f field.Model, characterId uint32, slot int16, itemId item2.Id, monsterUniqueId uint32) error {
	transactionId := uuid.New()

	ci, err := p.cdp.GetById(uint32(itemId))
	if err != nil {
		return p.catchError(characterId, itemId, consumable.CatchCauseInvalidItem, err)
	}
	if !validateCatchItem(uint32(itemId), ci) {
		return p.catchError(characterId, itemId, consumable.CatchCauseInvalidItem, errors.New("not a usable catch item"))
	}

	allowed, err := catchdelay.GetRegistry().Allow(p.ctx, characterId, uint32(itemId), catchUseDelay(ci))
	if err != nil {
		return p.catchError(characterId, itemId, consumable.CatchCauseInvalidItem, err)
	}
	if !allowed {
		return p.catchError(characterId, itemId, consumable.CatchCauseUseDelay, nil)
	}

	// atlas-inventory owns the merge-aware verdict. A full inventory fails here,
	// before anything is reserved and before a monster is removed.
	ok, err := p.ip.CanAccommodate(characterId, []inventory.AccommodationRequest{{ItemId: ci.Create(), Quantity: 1}})
	if err != nil {
		return p.catchError(characterId, itemId, consumable.CatchCauseInvalidItem, err)
	}
	if !ok {
		return p.catchError(characterId, itemId, consumable.CatchCauseInventoryFull, errors.New("inventory full"))
	}

	// Register the outcome handler BEFORE the reserve handler, and both BEFORE
	// RequestReserve, so no terminal event can race ahead of its handler.
	catchTopic, _ := topic.EnvProvider(p.l)(monsterMsg.EnvEventTopicCatch)()
	if _, err = consumer.GetManager().RegisterHandler(catchTopic, message.AdaptHandler(message.OneTimeConfig(catchResolvedValidator(characterId, itemId), catchResolutionHandler(transactionId, characterId, slot, itemId, ci.Create())))); err != nil {
		return p.catchError(characterId, itemId, consumable.CatchCauseInvalidItem, err)
	}

	compTopic, _ := topic.EnvProvider(p.l)(compartment2.EnvEventTopicStatus)()
	validator := once.ReservationValidator(transactionId, uint32(itemId))
	handler := compartment.Consume(ConsumeCatch(f, monsterUniqueId, characterId, itemId))
	if _, err = consumer.GetManager().RegisterHandler(compTopic, message.AdaptHandler(message.OneTimeConfig(validator, handler))); err != nil {
		return p.catchError(characterId, itemId, consumable.CatchCauseInvalidItem, err)
	}

	if err = p.cpp.RequestReserve(transactionId, characterId, inventory2.TypeValueUse, []compartment.Reserves{{Slot: slot, ItemId: uint32(itemId), Quantity: 1}}); err != nil {
		return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
	}
	return nil
}

// ConsumeCatch fires on RESERVED: the item is now held, so the monster-state
// decision can be handed to atlas-monsters, which is authoritative.
func ConsumeCatch(f field.Model, monsterUniqueId uint32, characterId uint32, itemId item2.Id) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			return monster.NewProcessor(l, ctx).RequestCatch(f, monsterUniqueId, characterId, uint32(itemId))
		}
	}
}
```

Read `RequestItemReward` (`processor.go:1079`) and `RequestFeed` (`processor.go:325`) before writing this and match their exact `RegisterHandler` / `OneTimeConfig` / import-alias shapes rather than the aliases guessed above.

Plus the failure emitter:

```go
// catchError unsticks the client on a pre-reserve rejection: nothing is
// reserved, so this only reports the semantic cause. atlas-channel maps cause
// to the client's wire reason byte — this service never picks 0 or 1 (DOM-25).
func (p *ProcessorImpl) catchError(characterId uint32, itemId item2.Id, cause string, err error) error {
	p.l.Debugf("Character [%d] catch request with item [%d] rejected pre-reserve: cause [%s], err [%v].", characterId, itemId, cause, err)
	if cErr := producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvEventTopic)(CatchFailedEventProvider(ts.Id(characterId), uint32(itemId), cause)); cErr != nil {
		p.l.WithError(cErr).Errorf("Unable to emit catch failure for character [%d]; client may be stuck.", characterId)
	}
	if err != nil {
		return err
	}
	return errors.New(cause)
}
```

Add `CatchFailedEventProvider` to the same producer file that holds `ErrorEventProvider` (find it with `grep -rn "func ErrorEventProvider" services/atlas-consumables`), following that function's exact shape with `Type: consumable.EventTypeCatchFailed` and `Body: consumable.CatchFailedBody{ItemId: itemId, Cause: cause}`.

Finally add the command handler in `kafka/consumer/consumable/consumer.go`, beside `handleRequestItemReward`:

```go
func handleRequestCatchMonster(l logrus.FieldLogger, ctx context.Context, c consumable2.Command[consumable2.RequestCatchMonsterBody]) {
	if c.Type != consumable2.CommandRequestCatchMonster {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	err := consumable.NewProcessor(l, ctx).RequestCatchMonster(f, uint32(c.CharacterId), int16(c.Body.Source), c.Body.ItemId, c.Body.MonsterUniqueId)
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to use catch item in slot [%d].", c.CharacterId, c.Body.Source)
	}
}
```

and register it in `InitHandlers` after `handleRequestItemReward`.

- [ ] **Step 6: Add the useDelay test and run the package tests**

Append to `services/atlas-consumables/atlas.com/consumables/catchdelay/registry_test.go` (create the file):

```go
package catchdelay

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestAllow_WindowRejectsThenAdmits — the first attempt is admitted and arms the
// window, a second inside the window is rejected, and once the TTL lapses the
// item is usable again. This is what makes BRIDLE_MOB_CATCH_FAIL reason 1 (the
// item's delayMsg) reachable at all.
func TestAllow_WindowRejectsThenAdmits(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	InitRegistry(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}))

	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	r := GetRegistry()

	ok, err := r.Allow(ctx, 42, 2270008, 3*time.Second)
	if err != nil || !ok {
		t.Fatalf("first attempt: ok=%t err=%v, want admitted", ok, err)
	}

	ok, err = r.Allow(ctx, 42, 2270008, 3*time.Second)
	if err != nil || ok {
		t.Fatalf("second attempt inside the window: ok=%t err=%v, want rejected", ok, err)
	}

	// A different item is a different window.
	ok, err = r.Allow(ctx, 42, 2270002, 3*time.Second)
	if err != nil || !ok {
		t.Fatalf("a different item: ok=%t err=%v, want admitted", ok, err)
	}

	mr.FastForward(4 * time.Second)
	ok, err = r.Allow(ctx, 42, 2270008, 3*time.Second)
	if err != nil || !ok {
		t.Fatalf("after the window lapsed: ok=%t err=%v, want admitted", ok, err)
	}
}

// TestAllow_ZeroDelayAlwaysAdmits — most catch items carry no useDelay.
func TestAllow_ZeroDelayAlwaysAdmits(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	InitRegistry(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}))

	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	for i := 0; i < 3; i++ {
		ok, err := GetRegistry().Allow(ctx, 42, 2270000, 0)
		if err != nil || !ok {
			t.Fatalf("attempt %d: ok=%t err=%v, want admitted", i, ok, err)
		}
	}
}
```

Match the miniredis import path already used elsewhere in the repo (`grep -rn "miniredis" services/atlas-monsters` shows it) rather than the one guessed above.

Run: `cd services/atlas-consumables/atlas.com/consumables && go test -race ./catchdelay/ ./consumable/ -run 'Allow|ValidateCatchItem' -v`
Expected: PASS.

The module-wide `go build ./...` is deliberately deferred to Task 11c Step 6 — this task's `RequestCatchMonster` references `catchResolvedValidator` and `catchResolutionHandler`, which Task 11c supplies. **Tasks 11b and 11c are one compile unit; do not commit 11b alone.**

---

## Task 11c: atlas-consumables resolution — commit, grant, cancel

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/catch.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/catch_test.go`
- Create: `services/atlas-consumables/atlas.com/consumables/kafka/consumer/monster/consumer.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/message/monster/kafka.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/main.go`

**Interfaces:**
- Consumes: `CATCH_RESOLVED` on `EVENT_TOPIC_MONSTER_CATCH` (Task 10).
- Produces: `catchResolutionHandler(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id, createItemId uint32) message.Handler[monsterMsg.Event[monsterMsg.CatchResolvedBody]]`.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-consumables/atlas.com/consumables/consumable/catch_test.go`:

```go
// TestCatchOutcomeDecision pins the two-way branch the resolution handler takes,
// separated from its Kafka plumbing so it is testable without a broker:
// success commits the reservation and grants the create item; failure cancels
// the reservation and grants nothing (FR-3.8, FR-3.9).
func TestCatchOutcomeDecision(t *testing.T) {
	cases := []struct {
		name       string
		success    bool
		wantCommit bool
		wantGrant  bool
		wantCancel bool
	}{
		{"a successful catch", true, true, true, false},
		{"a failed catch", false, false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := catchOutcome(tc.success)
			if d.commit != tc.wantCommit || d.grant != tc.wantGrant || d.cancel != tc.wantCancel {
				t.Fatalf("catchOutcome(%t) = %+v", tc.success, d)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run CatchOutcome -v`
Expected: FAIL — `undefined: catchOutcome`.

- [ ] **Step 3: Add the event contract**

Append to atlas-consumables' `kafka/message/monster/kafka.go`:

```go
const (
	EnvEventTopicCatch        = "EVENT_TOPIC_MONSTER_CATCH"
	EventMonsterCatchResolved = "CATCH_RESOLVED"
)

// Event mirrors atlas-monsters' status-event envelope on the dedicated catch
// topic. This service deliberately does NOT subscribe to
// EVENT_TOPIC_MONSTER_STATUS: that topic carries a DAMAGED event per hit and
// every registered handler unmarshals every message (design §4.2).
type Event[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	UniqueId  uint32     `json:"uniqueId"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type CatchResolvedBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
	Success     bool   `json:"success"`
	Cause       string `json:"cause"`
}
```

- [ ] **Step 4: Implement the resolution handler**

Append to `consumable/catch.go`:

```go
// catchDecision is the pure branch the resolution handler takes. Keeping it
// pure is what makes the commit/cancel contract testable without Kafka.
type catchDecision struct {
	commit bool
	grant  bool
	cancel bool
}

func catchOutcome(success bool) catchDecision {
	if success {
		return catchDecision{commit: true, grant: true}
	}
	return catchDecision{cancel: true}
}

// catchResolvedValidator matches only this attempt's CATCH_RESOLVED event.
// Correlation is by (characterId, itemId) captured at reserve time rather than a
// transaction id on the wire, so the CATCH command body stays minimal (FR-3.7).
func catchResolvedValidator(characterId uint32, itemId item2.Id) func(e monsterMsg.Event[monsterMsg.CatchResolvedBody]) bool {
	return func(e monsterMsg.Event[monsterMsg.CatchResolvedBody]) bool {
		return e.Type == monsterMsg.EventMonsterCatchResolved &&
			e.Body.CharacterId == characterId &&
			e.Body.ItemId == uint32(itemId)
	}
}

// catchResolutionHandler commits or cancels the reservation opened by
// RequestCatchMonster. On success it grants the create item through the same
// once-handler pair the reward-box flow uses, so a post-reserve creation failure
// cancels correctly. On failure it cancels and the item is untouched (FR-3.9) —
// and it emits NO consumable error event, because atlas-channel already renders
// the failure from the monster CATCH_FAILED event and two unlock packets would
// be sent otherwise.
func catchResolutionHandler(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id, createItemId uint32) message.Handler[monsterMsg.Event[monsterMsg.CatchResolvedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monsterMsg.Event[monsterMsg.CatchResolvedBody]) {
		p := NewProcessor(l, ctx).(*ProcessorImpl)
		d := catchOutcome(e.Body.Success)

		if d.cancel {
			if cErr := p.cpp.CancelItemReservation(characterId, inventory2.TypeValueUse, transactionId, slot); cErr != nil {
				l.WithError(cErr).Errorf("Unable to cancel catch reservation for character [%d] (transaction [%s]).", characterId, transactionId.String())
			}
			l.Debugf("Character [%d] catch failed (cause [%s]); item [%d] preserved.", characterId, e.Body.Cause, itemId)
			return
		}

		if d.commit {
			if cErr := p.cpp.ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, slot); cErr != nil {
				l.WithError(cErr).Errorf("Catch succeeded but the item consume failed for character [%d] (transaction [%s]); needs ops intervention.", characterId, transactionId.String())
			}
		}
		if d.grant {
			if cErr := p.cpp.RequestCreateItem(transactionId, characterId, createItemId, 1, time.Time{}); cErr != nil {
				l.WithError(cErr).Errorf("Catch succeeded but granting reward item [%d] failed for character [%d].", createItemId, characterId)
			}
		}
	}
}
```

`RequestCatchMonster` (Task 11b) already registers this handler — it calls `catchResolvedValidator` and `catchResolutionHandler` before reserving. Task 11b's build therefore fails until this task lands; that is the intended ordering, and Task 11b's Step 6 is the gate that proves both halves compile together. If you are executing 11b and 11c as one unit, write this file first.

- [ ] **Step 5: Add the consumer**

Create `services/atlas-consumables/atlas.com/consumables/kafka/consumer/monster/consumer.go`, mirroring `kafka/consumer/consumable/consumer.go`'s `InitConsumers` shape:

```go
package monster

import (
	consumer2 "atlas-consumables/kafka/consumer"
	monsterMsg "atlas-consumables/kafka/message/monster"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// InitConsumers subscribes to the dedicated catch topic only. There are no
// persistent handlers here: catch outcomes are consumed by the per-attempt
// once-handler RequestCatchMonster registers.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("monster_catch_event")(monsterMsg.EnvEventTopicCatch)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}
```

Register it in `main.go` beside the other `InitConsumers` calls.

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd services/atlas-consumables/atlas.com/consumables
go test -race ./... && go vet ./... && go build ./...
```
Expected: all clean. This is the gate for Tasks 11b **and** 11c together.

- [ ] **Step 7: Commit both halves**

```bash
git add services/atlas-consumables libs/atlas-constants
git commit -m "feat(task-212): catch-item request, useDelay gate, and reservation commit/cancel"
```

---

## Task 12a: atlas-channel handler and command emitter

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/monster_catch_item_use.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/monster_catch_item_use_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/consumable/{processor.go,producer.go}`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go`

**Interfaces:**
- Consumes: `serverbound.UseCatchItem` / `serverbound.UseCatchItemHandle` (Task 1); the consumables command contract (Task 11b).
- Produces: `handler.MonsterCatchItemUseHandleFunc` matching the `handlerMap` signature `func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{})`; and `consumable.Processor.RequestCatchMonster(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, monsterUniqueId uint32) error`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/monster_catch_item_use_test.go`:

```go
package handler

import (
	"testing"

	monstersb "github.com/Chronicle20/atlas/libs/atlas-packet/monster/serverbound"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// TestMonsterCatchItemUseDecode proves the handler's decode step recovers every
// field from the wire. The handler itself performs no validation — item checks
// live in atlas-consumables and monster checks in atlas-monsters (design §4.6).
func TestMonsterCatchItemUseDecode(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	encoded := monstersb.NewUseCatchItem(0x11223344, 7, 2270008, 0x07654321).Encode(nil, ctx)(nil)

	req := request.Request(encoded)
	reader := request.NewRequestReader(&req, 0)

	var p monstersb.UseCatchItem
	p.Decode(nil, ctx)(&reader, nil)

	if p.Slot() != 7 || p.ItemId() != 2270008 || p.MonsterUniqueId() != 0x07654321 {
		t.Fatalf("decoded %s", p.String())
	}
	if p.Operation() != monstersb.UseCatchItemHandle {
		t.Fatalf("Operation() = %q, want %q", p.Operation(), monstersb.UseCatchItemHandle)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run MonsterCatchItemUse -v`
Expected: FAIL until Task 1's codec is on the workspace path — if Task 1 is already committed it will PASS immediately; in that case proceed and let Step 4 be the real gate.

- [ ] **Step 3: Implement the handler**

Create `services/atlas-channel/atlas.com/channel/socket/handler/monster_catch_item_use.go`:

```go
package handler

import (
	"atlas-channel/consumable"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	monstersb "github.com/Chronicle20/atlas/libs/atlas-packet/monster/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// MonsterCatchItemUseHandleFunc decodes USE_CATCH_ITEM and forwards it. It
// performs no validation and holds no state: the item checks belong to
// atlas-consumables and the monster checks to atlas-monsters, which is
// authoritative and fail-closed. The opcode arrives from tenant configuration
// via the template route — never a constant here (DOM-25).
func MonsterCatchItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		var p monstersb.UseCatchItem
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = consumable.NewProcessor(l, ctx).RequestCatchMonster(s.Field(), character.Id(s.CharacterId()), item.Id(p.ItemId()), slot.Position(p.Slot()), p.MonsterUniqueId())
	}
}
```

- [ ] **Step 4: Add the command emitter and register the handler**

In `kafka/message/consumable/kafka.go` add `CommandRequestCatchMonster = "REQUEST_CATCH_MONSTER"` and:

```go
type RequestCatchMonsterBody struct {
	Source          slot.Position `json:"source"`
	ItemId          item.Id       `json:"itemId"`
	MonsterUniqueId uint32        `json:"monsterUniqueId"`
}
```

In `consumable/producer.go`:

```go
func RequestCatchMonsterCommandProvider(f field.Model, characterId character.Id, source slot.Position, itemId item.Id, monsterUniqueId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.RequestCatchMonsterBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        consumable.CommandRequestCatchMonster,
		Body: consumable.RequestCatchMonsterBody{
			Source:          source,
			ItemId:          itemId,
			MonsterUniqueId: monsterUniqueId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

In `consumable/processor.go` add to the interface and impl:

```go
	RequestCatchMonster(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, monsterUniqueId uint32) error
```

```go
func (p *ProcessorImpl) RequestCatchMonster(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, monsterUniqueId uint32) error {
	p.l.Debugf("Character [%d] using catch item [%d] from slot [%d] on monster [%d].", characterId, itemId, source, monsterUniqueId)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestCatchMonsterCommandProvider(f, characterId, source, itemId, monsterUniqueId))
}
```

Add the corresponding method to `consumable/mock/` so the mock still satisfies the interface.

In `main.go`, beside `handlerMap[monstersb.MobDropPickupRequestHandle]`:

```go
	handlerMap[monstersb.UseCatchItemHandle] = handler.MonsterCatchItemUseHandleFunc
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel
go test -race ./socket/handler/ ./consumable/ && go build ./...
```
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel
git commit -m "feat(task-212): decode USE_CATCH_ITEM and forward it to atlas-consumables"
```

---

## Task 12b: atlas-channel rendering — effects, failure, and the unlock

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go`
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go`

**Interfaces:**
- Consumes: `CAUGHT` / `CATCH_FAILED` on `EVENT_TOPIC_MONSTER_STATUS` (Task 10); `CATCH_FAILED` on `EVENT_TOPIC_CONSUMABLE_STATUS` (Task 11b); `writer.CatchMonsterWithItemBody(uniqueId, itemId, result)` (Task 5).
- Produces: `bridleFailReason(cause string) (byte, bool)` — the wire reason and whether to send a packet at all.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go`:

```go
// TestBridleFailReason maps internal causes onto the only two values the client
// understands. CWvsContext::OnBridleMobCatchFail @0x9d9a80 branches on exactly
// two: 0 renders string 0x110E, 1 renders the item's delayMsg (falling back to
// 0x110F), and ANY other value renders nothing at all. Reason 1 is reserved for
// the not-yet/try-again case, which is why useDelay is server-enforced.
// UNRESOLVED sends no packet: the request was legitimate and lost a race, so the
// client should simply unlock.
func TestBridleFailReason(t *testing.T) {
	cases := []struct {
		cause      string
		wantReason byte
		wantSend   bool
	}{
		{monster2.CatchCauseSpeciesMismatch, 0, true},
		{monster2.CatchCauseHpTooHigh, 0, true},
		{monster2.CatchCauseRollFailed, 0, true},
		{consumable2.CatchCauseInventoryFull, 0, true},
		{consumable2.CatchCauseInvalidItem, 0, true},
		{consumable2.CatchCauseUseDelay, 1, true},
		{monster2.CatchCauseUnresolved, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.cause, func(t *testing.T) {
			reason, send := bridleFailReason(tc.cause)
			if reason != tc.wantReason || send != tc.wantSend {
				t.Fatalf("bridleFailReason(%q) = (%d, %t), want (%d, %t)", tc.cause, reason, send, tc.wantReason, tc.wantSend)
			}
		})
	}
}
```

Add the imports the test needs, matching the file's existing aliases (`monster2` for `atlas-channel/kafka/message/monster`, `consumable2` for `atlas-channel/kafka/message/consumable`).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/monster/ -run BridleFailReason -v`
Expected: FAIL — `undefined: bridleFailReason`.

- [ ] **Step 3: Mirror the event contracts**

In `kafka/message/monster/kafka.go` (atlas-channel), add to the event const block:

```go
	EventStatusCaught      = "CAUGHT"
	EventStatusCatchFailed = "CATCH_FAILED"

	CatchCauseSpeciesMismatch = "SPECIES_MISMATCH"
	CatchCauseHpTooHigh       = "HP_TOO_HIGH"
	CatchCauseRollFailed      = "ROLL_FAILED"
	CatchCauseUnresolved      = "UNRESOLVED"
```

```go
type StatusEventCaughtBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
}

type StatusEventCatchFailedBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
	Cause       string `json:"cause"`
}
```

In `kafka/message/consumable/kafka.go` (atlas-channel), add `EventTypeCatchFailed = "CATCH_FAILED"`, the three consumables cause constants (`CatchCauseUseDelay`, `CatchCauseInventoryFull`, `CatchCauseInvalidItem`) and `CatchFailedBody{ItemId uint32, Cause string}`, matching Task 11b's JSON tags exactly.

- [ ] **Step 4: Implement the renderers**

In `kafka/consumer/monster/consumer.go`:

```go
// bridleFailReason maps an internal catch-failure cause onto the client's wire
// reason byte and reports whether to send the packet at all. The wire value is
// resolved HERE, in the rendering service — the domain services emit semantic
// causes only (DOM-25).
func bridleFailReason(cause string) (byte, bool) {
	switch cause {
	case consumable2.CatchCauseUseDelay:
		return 1, true
	case monster2.CatchCauseUnresolved:
		return 0, false
	default:
		return 0, true
	}
}

// handleStatusEventCaught renders a successful capture: the two effect packets
// go to everyone in the map (both always fire — bridleMsgType selects neither;
// neither CMob::OnCatchEffect nor CMob::OnEffectByItem reads it off the wire),
// then the acting character alone is unlocked. result = 1 selects the "captured"
// animation (CAnimationDisplayer::Effect_Catch @0x438eb6 loads StringPool 3687
// for non-zero, 3688 for zero).
//
// These MUST reach the client before the sibling DESTROYED event: the client
// resolves the mob via CMobPool::OnMobPacket -> GetMob and silently drops the
// packet once the mob is gone. Both events are keyed by MapId on the same topic,
// so the ordering is a partition guarantee.
func handleStatusEventCaught(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventCaughtBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventCaughtBody]) {
		if e.Type != monster2.EventStatusCaught {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		f := sc.Field(e.MapId, e.Instance)
		if err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
			if aerr := session.Announce(l)(ctx)(wp)(monsterpkt.CatchMonsterWriter)(writer.CatchMonsterBody(e.UniqueId, 1, 1))(s); aerr != nil {
				return aerr
			}
			return session.Announce(l)(ctx)(wp)(monsterpkt.CatchMonsterWithItemWriter)(writer.CatchMonsterWithItemBody(e.UniqueId, int32(e.Body.ItemId), 1))(s)
		}); err != nil {
			l.WithError(err).Errorf("Unable to announce the capture of monster [%d] in map [%d].", e.UniqueId, e.MapId)
		}

		// Emitted from its own statement so a failed effect broadcast can never
		// leave the client wedged.
		if err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)); err != nil {
			l.WithError(err).Errorf("Unable to unlock character [%d] after a successful catch.", e.Body.CharacterId)
		}
	}
}

// handleStatusEventCatchFailed renders a failed capture to the acting character
// only. The fail packet is optional — gms_v48 has no OnBridleMobCatchFail
// handler at all (its writer is simply not routed, and the writer registry
// reports it unconfigured) and UNRESOLVED deliberately renders nothing — but the
// unlock is not, so it is emitted from its own statement.
func handleStatusEventCatchFailed(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventCatchFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventCatchFailedBody]) {
		if e.Type != monster2.EventStatusCatchFailed {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		announceCatchFailure(l, ctx, sc, wp, e.Body.CharacterId, e.Body.ItemId, e.Body.Cause)
	}
}

// announceCatchFailure is shared by the monster-side and consumable-side failure
// paths so both render identically and both always unlock.
func announceCatchFailure(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId uint32, itemId uint32, cause string) {
	sp := session.NewProcessor(l, ctx)
	if reason, send := bridleFailReason(cause); send {
		if err := sp.IfPresentByCharacterId(sc.Channel())(characterId, session.Announce(l)(ctx)(wp)(charpkt.BridleMobCatchFailWriter)(writer.BridleMobCatchFailBody(reason, int32(itemId), 0))); err != nil {
			l.WithError(err).Debugf("Unable to write [%s] for character [%d]; continuing to the unlock.", charpkt.BridleMobCatchFailWriter, characterId)
		}
	}
	if err := sp.IfPresentByCharacterId(sc.Channel())(characterId, session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)); err != nil {
		l.WithError(err).Errorf("Unable to unlock character [%d] after a failed catch.", characterId)
	}
}
```

Register both handlers in this file's `InitHandlers`, following the existing `id, err = rf(...)` / `handles = append(...)` pattern exactly.

In `kafka/consumer/consumable/consumer.go`, add the pre-reserve failure renderer and register it:

```go
func handleCatchFailedEvent(sc server.Model, wp writer.Producer) message.Handler[consumable2.Event[consumable2.CatchFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e consumable2.Event[consumable2.CatchFailedBody]) {
		if e.Type != consumable2.EventTypeCatchFailed {
			return
		}
		monsterconsumer.AnnounceCatchFailure(l, ctx, sc, wp, uint32(e.CharacterId), e.Body.ItemId, e.Body.Cause)
	}
}
```

To avoid an import cycle, export `announceCatchFailure` as `AnnounceCatchFailure` (and `bridleFailReason` stays unexported) if the consumable consumer cannot otherwise reach it; if the two packages already share a helper location, put it there instead. Whichever placement you choose, there must be exactly ONE implementation of the mapping — two copies is how the 0/1 contract drifts.

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel
go test -race ./... && go vet ./... && go build ./...
```
Expected: all clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel
git commit -m "feat(task-212): render the catch effects, the failure notice, and always unlock"
```

---

## Task 13: Topic configuration, documentation, and the full verification sweep

**Files:**
- Modify: `deploy/k8s/base/env-configmap.yaml`
- Modify: `deploy/k8s/overlays/main/kustomization.yaml`
- Modify: `deploy/k8s/overlays/pr/kustomization.yaml`
- Modify: `services/atlas-monsters/docs/kafka.md`, `services/atlas-consumables/docs/kafka.md`, and the two services' `README.md` topic tables
- Modify: `docs/tasks/task-212-monster-catch-items/context.md` (the "not promoted, and why" note, if Task 7 produced one)

**Interfaces:**
- Consumes: everything above.
- Produces: a branch that passes the full CLAUDE.md build-and-verify list.

- [ ] **Step 1: Register the new topic**

In `deploy/k8s/base/env-configmap.yaml`, beside `EVENT_TOPIC_MONSTER_STATUS`:

```yaml
  EVENT_TOPIC_MONSTER_CATCH: "EVENT_TOPIC_MONSTER_CATCH"
```

In `deploy/k8s/overlays/main/kustomization.yaml`, beside the `EVENT_TOPIC_MONSTER_STATUS` literal:

```yaml
      - EVENT_TOPIC_MONSTER_CATCH=EVENT_TOPIC_MONSTER_CATCH-main
```

In `deploy/k8s/overlays/pr/kustomization.yaml`:

```yaml
      - EVENT_TOPIC_MONSTER_CATCH=EVENT_TOPIC_MONSTER_CATCH-PLACEHOLDER_ATLAS_ENV
```

An unsuffixed fallback in an overlay is a known silent-failure trap — both overlays must carry the suffixed form. Confirm the surrounding block's indentation by reading the neighbouring `EVENT_TOPIC_MONSTER_STATUS` line rather than assuming.

- [ ] **Step 2: Update the service docs**

Add `EVENT_TOPIC_MONSTER_CATCH` (produced by atlas-monsters, consumed by atlas-consumables) and `COMMAND_TOPIC_MONSTER` type `CATCH` to the relevant tables in `services/atlas-monsters/docs/kafka.md` and `services/atlas-consumables/docs/kafka.md`, plus the `README.md` topic tables in both services. Follow the surrounding table format exactly.

- [ ] **Step 3: Run the full verification list**

From the worktree root:

```bash
for m in libs/atlas-packet libs/atlas-redis libs/atlas-constants \
         services/atlas-monsters/atlas.com/monsters \
         services/atlas-consumables/atlas.com/consumables \
         services/atlas-channel/atlas.com/channel; do
  echo "== $m"
  (cd "$m" && go test -race ./... && go vet ./... && go build ./...) || exit 1
done

tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/skill-job-id-guard.sh
tools/buff-duration-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/service-registration-guard.sh
go run ./tools/packet-audit matrix --check
tools/lint.sh --check
```

Expected: every command exits 0. `tools/lint.sh --check` needs nvm on PATH for its atlas-ui half — if it false-fails without it, source nvm and re-run rather than declaring it passed. Run `tools/lint.sh` (no flags) first to fix formatting in place, then re-run `--check`.

- [ ] **Step 4: Run the mandatory bake**

No `go.mod` gained a new require in this task, but three services changed and the shared Dockerfile is the only thing that proves their `COPY libs/...` lines are complete:

```bash
docker buildx bake atlas-channel atlas-consumables atlas-monsters
```

Expected: all three targets succeed. Do not skip this — `go build` against the workspace cannot catch a missing `COPY`.

- [ ] **Step 5: Commit**

```bash
git add deploy/k8s services docs
git commit -m "chore(task-212): register EVENT_TOPIC_MONSTER_CATCH and document the catch flow"
```

- [ ] **Step 6: Code review before PR**

Run `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (no atlas-ui files changed, so no frontend reviewer). Findings go to `docs/tasks/task-212-monster-catch-items/audit.md`. Do not open the PR until the review has run and its findings are addressed.

---

## Manual acceptance (post-merge, live tenant)

These are the PRD acceptance criteria that no automated test in this plan covers. Run them against a live tenant on at least one legacy version (gms_v48 or gms_v79) and one modern one (gms_v83 or gms_v95):

- [ ] Using the correct catch item on a qualifying monster removes the monster, grants the `create` item, consumes the catch item, and plays the capture effect.
- [ ] Using a catch item on the wrong species sends `BRIDLE_MOB_CATCH_FAIL`, leaves the item in inventory, and unlocks the client.
- [ ] A failed probability roll leaves the item in inventory (item `02270002`, the only one carrying `bridleProp`).
- [ ] A caught monster yields no experience, no drops, and no death events.
- [ ] Every terminal path unlocks — act again immediately after each of: success, wrong species, HP too high, failed roll, inventory full, `useDelay` window.
- [ ] On gms_v48 specifically, a failed catch still unlocks despite `BridleMobCatchFail` being unrouted.

If the HP boundary behaves wrongly, assumption **A-1** (`mobHP` as a percentage) is the first thing to revisit. If item `02270002` feels wrong, revisit **A-2** (`bridlePropChg` as a one-shot multiplier). If the wrong animation plays, revisit **A-3**. If spam-catching is possible or the delay message never appears, revisit **A-4**.
