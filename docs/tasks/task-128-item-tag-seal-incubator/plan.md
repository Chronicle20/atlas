# Item Tag, Sealing Locks, and Incubator (Cash 506 Family) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the three cash 506-family item behaviors (Item Tag 5060000, Sealing Lock 5060001/5061000–5061003, Incubator 5060002) end-to-end for all supported tenant versions, including a new asset `owner` field through the whole stack and the `INCUBATOR_RESULT` clientbound packet.

**Architecture:** The channel handler decodes per-version sub-bodies, validates server-side, and drives atomic sagas (destroy-first ordering). atlas-inventory owns all asset mutation via two new compartment commands (`SET_OWNER`, `APPLY_LOCK`) plus a lock-aware expire branch. The orchestrator gains three saga actions; the incubator result reaches the client via a dedicated fire-and-forget event consumed by a new channel consumer. Incubator reward pools live in atlas-tenants generic-JSONB configuration.

**Tech Stack:** Go workspaces, GORM/Postgres (sqlite in tests), Kafka (message.Buffer/Emit), JSON:API REST, atlas-packet codecs, packet-audit verification tooling.

**Design:** `docs/tasks/task-128-item-tag-seal-incubator/design.md` (PRD: `prd.md`). Design corrections discovered during planning are listed in `context.md` §Corrections — the plan below already incorporates them.

## Global Constraints

- Multi-tenancy: all behavior tenant-scoped via `tenant.MustFromContext(ctx)`; packet opcodes come from tenant config, never hardcoded (PRD §8).
- Atomicity: every consume+mutate pair is a saga; destroy-first step ordering; no fire-and-forget multi-step mutations (PRD §8). Saga `Timeout` stays 0 (orchestrator default 30 s) — all three sagas have fixed 2–4 steps, no data-driven step counts.
- Validated no-ops log at `Warnf` with character/item/slot context, consume nothing (PRD §4).
- All target-slot inputs are client-controlled: validate compartment type, slot occupancy, and classification server-side before mutating (PRD §8).
- Never invent opcodes/read-orders: v83 sub-body orders and `INCUBATOR_RESULT` bodies are IDA-verified in design.md §1; anything else must be verified against checked-in IDA exports or live IDA (v83 port 13342, v95 port 13341). An unresolvable fname is a stop-and-ask, never a guess.
- Per changed module before claiming done: `go test -race ./...`, `go vet ./...`, `go build ./...` clean; `docker buildx bake atlas-<svc>` for every service whose `go.mod` was touched; `tools/redis-key-guard.sh` clean from repo root (run WITHOUT a `GOWORK=off` prefix).
- No `// TODO`, stubs, or 501s in landed commits.
- Interface changes require immediate mock updates (`var _ Processor` compile checks enforce this in atlas-tenants).
- Update the affected service README/kafka docs when commands/events/endpoints change.
- Worktree: all work happens in this worktree on branch `task-128-item-tag-seal-incubator`. All paths below are worktree-relative.

**Verification shorthand used below:** "MODULE-VERIFY `<dir>`" means run `go test -race ./... -count=1 && go vet ./... && go build ./...` inside `<dir>` and expect all clean.

---

### Task 1: Named item id constants (libs/atlas-constants)

**Files:**
- Modify: `libs/atlas-constants/item/constants.go` (named-id const block, near line 234)

**Interfaces:**
- Consumes: existing `type Id uint32`, `ClassificationItemImprints = Classification(506)` (constants.go:75).
- Produces: `item.ItemTag`, `item.SealingLock`, `item.Incubator`, `item.SealingLock7Day`, `item.SealingLock30Day`, `item.SealingLock90Day`, `item.SealingLock365Day` — used by Tasks 16 and referenced in tests.

- [ ] **Step 1: Add the constants**

In the named-id const block of `libs/atlas-constants/item/constants.go` (the block containing `WhiteScroll = Id(2340000)` at :242), add:

```go
	// Cash imprint family (Item.wz/Cash/0506.img, task-128)
	ItemTag           = Id(5060000)
	SealingLock       = Id(5060001)
	Incubator         = Id(5060002)
	SealingLock7Day   = Id(5061000)
	SealingLock30Day  = Id(5061001)
	SealingLock90Day  = Id(5061002)
	SealingLock365Day = Id(5061003)
```

- [ ] **Step 2: Verify**

MODULE-VERIFY `libs/atlas-constants`. Expected: clean (constants only, no behavior).

- [ ] **Step 3: Commit**

```bash
git add libs/atlas-constants/item/constants.go
git commit -m "feat(constants): named ids for cash 506 imprint family"
```

---

### Task 2: Asset owner field in the packet codec (libs/atlas-packet)

**Files:**
- Modify: `libs/atlas-packet/model/asset.go`
- Test: `libs/atlas-packet/model/asset_test.go`

**Interfaces:**
- Consumes: existing `Asset` value-type with immutable `Set*` setters (asset.go:56-167).
- Produces: `Asset.SetOwner(owner string) Asset`, `Asset.Owner() string`. The four encoders write `m.owner` where they previously wrote `""`.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-packet/model/asset_test.go`:

```go
func TestAssetOwnerEncoded(t *testing.T) {
	exp := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	base := NewAsset(false, -5, 1302000, exp).
		SetEquipmentStats(10, 11, 12, 13, 100, 50, 80, 70, 30, 25, 15, 20, 10, 5, 3).
		SetEquipmentMeta(7, 1, 2, 500, 3, 0x0001)
	named := base.SetOwner("Tumi")
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			plain := base.Encode(l, ctx)(nil)
			withOwner := named.Encode(l, ctx)(nil)
			if len(withOwner) != len(plain)+len("Tumi") {
				t.Fatalf("owner bytes not encoded: len(withOwner)=%d len(plain)=%d", len(withOwner), len(plain))
			}
			// empty owner must be byte-identical to the pre-change encoding
			empty := base.SetOwner("").Encode(l, ctx)(nil)
			if !bytes.Equal(empty, plain) {
				t.Fatal("empty owner changed the wire bytes")
			}
		})
	}
}

func TestAssetOwnerEncodedStackable(t *testing.T) {
	exp := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	base := NewAsset(false, 3, 2000000, exp).SetStackableInfo(50, 0, 0)
	named := base.SetOwner("Tumi")
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	plain := base.Encode(l, ctx)(nil)
	withOwner := named.Encode(l, ctx)(nil)
	if len(withOwner) != len(plain)+len("Tumi") {
		t.Fatalf("owner bytes not encoded on stackable: %d vs %d", len(withOwner), len(plain))
	}
}
```

Match the file's existing imports (`bytes`, `pt`, `testlog` aliases — mirror `TestAssetDeterministicEncode` at asset_test.go:217).

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-packet && go test ./model/ -run TestAssetOwner -v`
Expected: FAIL — `named.SetOwner` undefined.

- [ ] **Step 3: Implement**

In `libs/atlas-packet/model/asset.go`:

1. Add field to the `Asset` struct (with the stackable fields, near `quantity`): `owner string`.
2. Add setter/getter alongside the other setters (:119-167):

```go
func (m Asset) SetOwner(owner string) Asset {
	m.owner = owner
	return m
}

func (m Asset) Owner() string {
	return m.owner
}
```

3. Replace the four hardcoded writes — at `encodeEquipableInfo` (:209), `encodeCashEquipableInfo` (:261), `encodeStackableInfo` (:287), `encodeCashItemInfo` (:332) — change `w.WriteAsciiString("")` to `w.WriteAsciiString(m.owner)`. Do NOT touch the decode mirrors (:416, :472) — decode continues to discard the name (channel is write-only for this field).

- [ ] **Step 4: Run tests**

Run: `cd libs/atlas-packet && go test -race ./model/ -count=1`
Expected: PASS, including all pre-existing asset tests (they assert length/determinism, empty owner is byte-identical).

- [ ] **Step 5: Verify module + commit**

MODULE-VERIFY `libs/atlas-packet`, then:

```bash
git add libs/atlas-packet/model/asset.go libs/atlas-packet/model/asset_test.go
git commit -m "feat(packet): asset owner name in equip/stackable/cash encoders"
```

---

### Task 3: Serverbound sub-body codecs (libs/atlas-packet)

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/item_use_item_tag.go`
- Create: `libs/atlas-packet/cash/serverbound/item_use_seal.go`
- Create: `libs/atlas-packet/cash/serverbound/item_use_incubator.go`
- Test: matching `*_test.go` next to each

**Interfaces:**
- Consumes: `request.Reader`, `response.NewWriter` — mirror `item_use_chalkboard.go:12-49` exactly.
- Produces: `NewItemUseItemTag(updateTimeFirst bool) *ItemUseItemTag` with `Slot() int16`, `UpdateTime() uint32`; `NewItemUseSeal(updateTimeFirst bool) *ItemUseSeal` and `NewItemUseIncubator(updateTimeFirst bool) *ItemUseIncubator`, each with `InventoryType() int32`, `Slot() int32`, `UpdateTime() uint32`. Used by Task 16.

v83 read orders are IDA-verified (design.md §1): tag = `short slot` (+ trailing `int updateTime` when not updateTimeFirst); seal and incubator = `int inventoryType`, `int slot` (+ trailing updateTime). Task 19 re-verifies v87/v95/jms from the IDA exports.

- [ ] **Step 1: Write the failing tests**

`item_use_item_tag_test.go`:

```go
package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestItemUseItemTagRoundTrip(t *testing.T) {
	for _, first := range []bool{true, false} {
		for _, v := range pt.Variants {
			t.Run(v.Name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				input := ItemUseItemTag{slot: -1, updateTime: 1000, updateTimeFirst: first}
				output := *NewItemUseItemTag(first)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Slot() != input.Slot() {
					t.Errorf("slot = %d, want %d", output.Slot(), input.Slot())
				}
				if !first && output.UpdateTime() != input.UpdateTime() {
					t.Errorf("updateTime = %d, want %d", output.UpdateTime(), input.UpdateTime())
				}
			})
		}
	}
}

// v83 golden bytes: short slot (-1 = FF FF) + trailing int updateTime (1000 = E8 03 00 00)
func TestItemUseItemTagV83Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := ItemUseItemTag{slot: -1, updateTime: 1000, updateTimeFirst: false}
	got := m.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{0xFF, 0xFF, 0xE8, 0x03, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}
```

`item_use_seal_test.go` and `item_use_incubator_test.go` — same shape for `ItemUseSeal`/`ItemUseIncubator` with `input := ItemUseSeal{inventoryType: 1, slot: -5, updateTime: 1000, updateTimeFirst: first}`, getter assertions for `InventoryType()`/`Slot()`, and v83 golden bytes:

```go
	want := []byte{0x01, 0x00, 0x00, 0x00, 0xFB, 0xFF, 0xFF, 0xFF, 0xE8, 0x03, 0x00, 0x00}
```

(int32 1 LE, int32 -5 LE, uint32 1000 LE.)

- [ ] **Step 2: Run to verify failure**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -run 'ItemUseItemTag|ItemUseSeal|ItemUseIncubator' -v`
Expected: FAIL — types undefined.

- [ ] **Step 3: Implement the three codecs**

`item_use_item_tag.go`:

```go
package serverbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-packet/request"
	"github.com/Chronicle20/atlas/libs/atlas-packet/response"
	"github.com/sirupsen/logrus"
)

// ItemUseItemTag is the type-25 sub-body of the cash ItemUse packet (Item Tag 5060000).
type ItemUseItemTag struct {
	slot            int16
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseItemTag(updateTimeFirst bool) *ItemUseItemTag {
	return &ItemUseItemTag{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseItemTag) Slot() int16        { return m.slot }
func (m ItemUseItemTag) UpdateTime() uint32 { return m.updateTime }
func (m ItemUseItemTag) Operation() string  { return "ItemUseItemTag" }

func (m ItemUseItemTag) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.slot)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseItemTag) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.slot = r.ReadInt16()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
```

`item_use_seal.go` — same skeleton with:

```go
// ItemUseSeal is the type-26/64/65 sub-body of the cash ItemUse packet (Sealing Locks).
type ItemUseSeal struct {
	inventoryType   int32
	slot            int32
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseSeal(updateTimeFirst bool) *ItemUseSeal {
	return &ItemUseSeal{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseSeal) InventoryType() int32 { return m.inventoryType }
func (m ItemUseSeal) Slot() int32          { return m.slot }
func (m ItemUseSeal) UpdateTime() uint32   { return m.updateTime }
func (m ItemUseSeal) Operation() string    { return "ItemUseSeal" }
```

Encode body: `w.WriteInt32(m.inventoryType); w.WriteInt32(m.slot); if !m.updateTimeFirst { w.WriteInt(m.updateTime) }`. Decode mirrors with `r.ReadInt32()` / `r.ReadUint32()`.

`item_use_incubator.go` — identical fields/body with type name `ItemUseIncubator` and `Operation() string { return "ItemUseIncubator" }` (kept as a distinct struct: the two packets are distinct client ops that only coincidentally share a shape at v83; per-version verification is per-op).

- [ ] **Step 4: Run tests**

Run: `cd libs/atlas-packet && go test -race ./cash/serverbound/ -count=1`
Expected: PASS.

- [ ] **Step 5: Verify module + commit**

MODULE-VERIFY `libs/atlas-packet`, then:

```bash
git add libs/atlas-packet/cash/serverbound/item_use_item_tag.go libs/atlas-packet/cash/serverbound/item_use_item_tag_test.go \
        libs/atlas-packet/cash/serverbound/item_use_seal.go libs/atlas-packet/cash/serverbound/item_use_seal_test.go \
        libs/atlas-packet/cash/serverbound/item_use_incubator.go libs/atlas-packet/cash/serverbound/item_use_incubator_test.go
git commit -m "feat(packet): item tag / seal / incubator cash ItemUse sub-bodies"
```

---

### Task 4: INCUBATOR_RESULT clientbound writer (libs/atlas-packet)

**Files:**
- Create: `libs/atlas-packet/incubator/clientbound/result.go`
- Test: `libs/atlas-packet/incubator/clientbound/result_test.go`

**Interfaces:**
- Consumes: `tenant.MustFromContext` version switch idiom (asset.go), `RemoveDoor` writer shape (door/clientbound/remove.go:11-52).
- Produces: `const IncubatorResultWriter = "IncubatorResult"`, `NewIncubatorResult(itemId uint32, count uint16) IncubatorResult` with `Encode`. Used by Tasks 16, 17.

IDA-verified bodies (design.md §1): v83 `CWvsContext::OnIncubatorResult` @0xa28298 reads `int itemId`, `short count`; v95 @0xa00380 reads those plus `int gachaponItemId`, `int bonusItemId`, `int bonusCount`. `itemId <= 0` → client shows the "inventory is full, try again" dialog. v83/v84 use the short body; v87/v95/JMS the extended one. Atlas rolls a single reward, so the extended tail is zeros (client skips the bonus branch). The version delta is a tenant switch inside `Encode` (same idiom as `model/asset.go`), not a constructor flag — deviation from design §7.2 noted in context.md.

- [ ] **Step 1: Write the failing test**

`result_test.go`:

```go
package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Task 19 adds the packet-audit:verify markers + evidence for all five versions.
func TestIncubatorResult(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := NewIncubatorResult(2000000, 1)

	// v83/v84: int itemId + short count (2000000 = 0x001E8480)
	short := []byte{0x80, 0x84, 0x1E, 0x00, 0x01, 0x00}
	// v87/v95/jms: + int gachaponItemId, int bonusItemId, int bonusCount (all zero)
	extended := append(append([]byte{}, short...), make([]byte, 12)...)

	cases := []struct {
		region string
		major  uint16
		want   []byte
	}{
		{"GMS", 83, short},
		{"GMS", 84, short},
		{"GMS", 87, extended},
		{"GMS", 95, extended},
		{"JMS", 185, extended},
	}
	for _, c := range cases {
		got := m.Encode(l, pt.CreateContext(c.region, c.major, 1))(nil)
		if !bytes.Equal(got, c.want) {
			t.Errorf("%s v%d: got % X, want % X", c.region, c.major, got, c.want)
		}
	}
}

func TestIncubatorResultFailureBody(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := NewIncubatorResult(0, 0).Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd libs/atlas-packet && go test ./incubator/... -v`
Expected: FAIL — package does not exist yet (build error).

- [ ] **Step 3: Implement**

`result.go`:

```go
package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-packet/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const IncubatorResultWriter = "IncubatorResult"

// IncubatorResult is CWvsContext::OnIncubatorResult. itemId <= 0 renders the
// client's "inventory is full, try again later" dialog.
type IncubatorResult struct {
	itemId uint32
	count  uint16
}

func NewIncubatorResult(itemId uint32, count uint16) IncubatorResult {
	return IncubatorResult{itemId: itemId, count: count}
}

func (m IncubatorResult) ItemId() uint32    { return m.itemId }
func (m IncubatorResult) Count() uint16     { return m.count }
func (m IncubatorResult) Operation() string { return IncubatorResultWriter }

func (m IncubatorResult) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.itemId)
		w.WriteShort(m.count)
		if (t.Region() == "GMS" && t.MajorVersion() >= 87) || t.Region() == "JMS" {
			// Atlas rolls a single reward; the gachapon/bonus tail is unused.
			w.WriteInt(0)
			w.WriteInt(0)
			w.WriteInt(0)
		}
		return w.Bytes()
	}
}
```

Check the exact import path/aliases against `door/clientbound/remove.go` and match them.

- [ ] **Step 4: Run tests**

Run: `cd libs/atlas-packet && go test -race ./incubator/... -count=1`
Expected: PASS.

- [ ] **Step 5: Verify module + commit**

MODULE-VERIFY `libs/atlas-packet`, then:

```bash
git add libs/atlas-packet/incubator/
git commit -m "feat(packet): IncubatorResult clientbound writer (2-field v83/v84, 5-field v87+)"
```

---

### Task 5: Saga types, actions, payloads (libs/atlas-saga)

**Files:**
- Modify: `libs/atlas-saga/model.go`, `libs/atlas-saga/payloads.go`, `libs/atlas-saga/unmarshal.go`
- Test: `libs/atlas-saga/unmarshal_test.go`

**Interfaces:**
- Produces (used by Tasks 9, 10, 16, 17):
  - `Type` consts: `ItemTagUse Type = "item_tag_use"`, `SealingLockUse Type = "sealing_lock_use"`, `IncubatorUse Type = "incubator_use"`.
  - `Action` consts: `SetAssetOwner Action = "set_asset_owner"`, `ApplyAssetLock Action = "apply_asset_lock"`, `IncubatorResult Action = "incubator_result"`.
  - `SetAssetOwnerPayload{CharacterId uint32, InventoryType byte, Slot int16, Owner string}`
  - `ApplyAssetLockPayload{CharacterId uint32, InventoryType byte, Slot int16, Expiration time.Time}`
  - `IncubatorResultPayload{CharacterId uint32, WorldId byte, ChannelId byte, ItemId uint32, Count uint32}`
  - `DestroyAssetFromSlotPayload` gains `TemplateId uint32` (compensator re-create needs it; today the payload has none — payloads.go:101-108).

- [ ] **Step 1: Write the failing tests**

Add to `libs/atlas-saga/unmarshal_test.go`, mirroring an existing per-action test in that file (read one first and copy its shape exactly — construct a JSON step with `"action": "set_asset_owner"` etc., unmarshal into `Step[any]`, assert the payload type and fields):

```go
func TestUnmarshalSetAssetOwnerStep(t *testing.T) {
	data := []byte(`{"stepId":"s1","status":"pending","action":"set_asset_owner","payload":{"characterId":7,"inventoryType":1,"slot":-5,"owner":"Tumi"},"createdAt":"2026-07-02T00:00:00Z","updatedAt":"2026-07-02T00:00:00Z"}`)
	var s Step[any]
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	p, ok := s.Payload.(SetAssetOwnerPayload)
	if !ok {
		t.Fatalf("payload type = %T", s.Payload)
	}
	if p.Owner != "Tumi" || p.Slot != -5 || p.InventoryType != 1 || p.CharacterId != 7 {
		t.Fatalf("payload = %+v", p)
	}
}
```

Add equivalent `TestUnmarshalApplyAssetLockStep` (assert `Expiration` round-trips an RFC3339 value), `TestUnmarshalIncubatorResultStep` (assert ItemId/Count/WorldId/ChannelId), and `TestUnmarshalDestroyAssetFromSlotTemplateId` (existing action, JSON payload includes `"templateId":4001126`, assert it lands).

- [ ] **Step 2: Run to verify failure**

Run: `cd libs/atlas-saga && go test ./... -run TestUnmarshal -v`
Expected: FAIL — payload types undefined.

- [ ] **Step 3: Implement**

1. `model.go` `Type` const block (:13-27): append the three `Type` consts.
2. `model.go` `Action` const block (:42-161): append the three `Action` consts.
3. `payloads.go`: append the three payload structs exactly as in **Interfaces** above, each field with the lowerCamel JSON tag shown; add `TemplateId uint32 \`json:"templateId,omitempty"\`` to `DestroyAssetFromSlotPayload` (:101-108) with the comment `// TemplateId lets the compensator re-create a slot-destroyed asset`.
4. `unmarshal.go`: add three cases in the `switch s.Action` (before `default`), each following the `DestroyAsset` case shape (:72-77) with the matching payload type.

- [ ] **Step 4: Run tests**

Run: `cd libs/atlas-saga && go test -race ./... -count=1`
Expected: PASS (any lib-level unmarshal-completeness test must also pass; if it enumerates actions, add the three).

- [ ] **Step 5: Verify + commit**

MODULE-VERIFY `libs/atlas-saga`. Note: the orchestrator's own completeness test (`services/atlas-saga-orchestrator/.../saga/unmarshal_completeness_test.go`) will now FAIL until Task 9 — that is expected and is why Task 9 must land in the same PR; do not run the orchestrator module gate until Task 9.

```bash
git add libs/atlas-saga/model.go libs/atlas-saga/payloads.go libs/atlas-saga/unmarshal.go libs/atlas-saga/unmarshal_test.go
git commit -m "feat(saga): SetAssetOwner/ApplyAssetLock/IncubatorResult actions and payloads"
```

---

### Task 6: Asset owner field end-to-end in atlas-inventory

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/asset/entity.go` (struct + `Make`)
- Modify: `services/atlas-inventory/atlas.com/inventory/asset/model.go`, `asset/builder.go`, `asset/rest.go`, `asset/administrator.go`, `asset/producer.go`
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/message/asset/kafka.go` (`AssetData`)
- Modify: the ACCEPT-command model construction in `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go`
- Test: `asset/builder_test.go`, `asset/rest_test.go`, `asset/producer_test.go`

**Interfaces:**
- Produces: `asset.Model.Owner() string`, `ModelBuilder.SetOwner(string)`, `RestModel.Owner`, `AssetData.Owner`, `updateOwner(db, id, owner) error`. Used by Tasks 7, 13, 14.

- [ ] **Step 1: Write the failing tests**

In `asset/builder_test.go`, add (mirror the existing test style in that file):

```go
func TestBuilderOwnerCarriedByClone(t *testing.T) {
	m := NewBuilder(uuid.New(), 1302000).SetOwner("Tumi").Build()
	if m.Owner() != "Tumi" {
		t.Fatalf("Owner() = %q, want Tumi", m.Owner())
	}
	c := Clone(m).Build()
	if c.Owner() != "Tumi" {
		t.Fatalf("Clone dropped owner: %q", c.Owner())
	}
}
```

In `asset/rest_test.go`, extend the existing Transform/Extract round-trip test (or add one) asserting `Owner` survives `Transform` → `Extract`. In `asset/producer_test.go`, assert `makeAssetData` populates `Owner` (mirror how the existing test checks `OwnerId`/`Flag`).

- [ ] **Step 2: Run to verify failure**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test ./asset/ -run 'Owner' -v`
Expected: FAIL — `SetOwner`/`Owner` undefined.

- [ ] **Step 3: Implement the domain chain**

1. `entity.go`: in the stackable field group next to `OwnerId`, add `Owner string \`gorm:"not null;default:''"\``. (`assets` is NOT in atlas-data's baseline `DumpTables` — verified — so no baseline column-list work.) In `Make` (:69-107) add `owner: e.Owner`.
2. `model.go`: add `owner string` field + `func (m Model) Owner() string { return m.owner }`.
3. `builder.go`: add `owner string` to the builder, `SetOwner(o string) *ModelBuilder`, copy in `Clone` (:10-48), emit in `Build` (:183-221).
4. `rest.go`: add `Owner string \`json:"owner"\`` to `RestModel`; map in `Transform` (`Owner: m.owner`) and `Extract` (`owner: rm.Owner`).
5. `kafka/message/asset/kafka.go`: add `Owner string \`json:"owner"\`` to `AssetData` (next to `OwnerId` at :36).
6. `asset/producer.go` `makeAssetData` (:13-47): add `Owner: a.owner`.
7. `asset/administrator.go`: in `create` (:8-52) add `Owner: m.owner` to the inserted Entity; add a scoped updater following `updateSlot`'s exact style (:54-56):

```go
func updateOwner(db *gorm.DB, id uint32, owner string) error {
	return db.Model(&Entity{}).Where("id = ?", id).Update("owner", owner).Error
}
```

(If `updateSlot` uses a different GORM idiom, match it verbatim.)

8. ACCEPT path: in `kafka/consumer/compartment/consumer.go`, find where `AcceptCommandBody.AssetData` is turned into an `asset.Model` (the handler for `CommandAccept`); add `.SetOwner(<body>.Owner)` to that builder chain so storage round-trips preserve the tag.

- [ ] **Step 4: Run tests**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test -race ./... -count=1`
Expected: PASS.

- [ ] **Step 5: Update docs + commit**

Update the asset attribute table in `services/atlas-inventory`'s README/docs if one documents the REST model. Then:

```bash
git add services/atlas-inventory/
git commit -m "feat(inventory): asset owner name column, domain field, REST/Kafka attribute"
```

---

### Task 7: SET_OWNER and APPLY_LOCK compartment commands (atlas-inventory)

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go`
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go`
- Modify: `services/atlas-inventory/atlas.com/inventory/compartment/processor.go`
- Modify: `services/atlas-inventory/atlas.com/inventory/asset/processor.go`, `asset/administrator.go`
- Test: `compartment/processor_test.go` (sqlite in-memory, existing style at :47)

**Interfaces:**
- Consumes: Task 6 (`SetOwner`, `updateOwner`), `af.FlagLock`, `UpdatedEventStatusProvider` (asset/producer.go:116).
- Produces (consumed by Task 9's producers — shapes must match exactly):

```go
CommandSetOwner  = "SET_OWNER"
CommandApplyLock = "APPLY_LOCK"

type SetOwnerCommandBody struct {
	Slot  int16  `json:"slot"`
	Owner string `json:"owner"`
}

type ApplyLockCommandBody struct {
	Slot       int16     `json:"slot"`
	Expiration time.Time `json:"expiration"` // zero time = permanent lock
}
```

- [ ] **Step 1: Write the failing processor tests**

In `compartment/processor_test.go`, following the existing sqlite setup (:47) and existing test structure, add:

```go
// SetAssetOwner stamps the owner and emits an asset UPDATED event.
func TestSetAssetOwner(t *testing.T) { /* create compartment + equip asset in slot -5;
	call NewProcessor(l, ctx, db).SetAssetOwnerAndEmit(uuid.New(), characterId, inventory.TypeValueEquip, -5, "Tumi");
	reload asset via asset processor GetBySlot; assert Owner() == "Tumi" */ }

// ApplyAssetLock sets FlagLock and the expiration.
func TestApplyAssetLock(t *testing.T) { /* equip asset, zero expiration;
	exp := time.Now().AddDate(0,0,7).Truncate(time.Second);
	ApplyAssetLockAndEmit(...); reload; assert Locked() && Expiration().Equal(exp) */ }

// ApplyAssetLock on an unlocked asset that already has an expiration is rejected
// (prevents laundering a genuinely time-limited item into a permanent one).
func TestApplyAssetLockRejectsTimeLimitedItem(t *testing.T) { /* equip asset with
	non-zero expiration, no FlagLock; expect error from ApplyAssetLockAndEmit;
	reload; assert !Locked() */ }
```

Fill in the setup using the same helpers the file's existing tests use (create tenant context, compartment row, asset row) — copy the arrange section of the nearest existing mutation test verbatim and adapt.

- [ ] **Step 2: Run to verify failure**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test ./compartment/ -run 'SetAssetOwner|ApplyAssetLock' -v`
Expected: FAIL — methods undefined.

- [ ] **Step 3: Implement**

1. `kafka/message/compartment/kafka.go`: add the two consts to the command block (:13-33) and the two bodies from **Interfaces** above.
2. `asset/administrator.go`: add

```go
func updateFlagAndExpiration(db *gorm.DB, id uint32, flag uint16, expiration time.Time) error {
	return db.Model(&Entity{}).Where("id = ?", id).Updates(map[string]interface{}{
		"flag":       flag,
		"expiration": expiration,
	}).Error
}
```

3. `asset/processor.go`: add two methods following the `UpdateEquipmentStats` shape (:227-266 — Clone/mutate/persist/emit-UPDATED):

```go
func (p *Processor) UpdateOwner(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a Model, owner string) error {
	return func(transactionId uuid.UUID, characterId uint32) func(a Model, owner string) error {
		return func(a Model, owner string) error {
			updated := Clone(a).SetOwner(owner).Build()
			if err := updateOwner(p.db.WithContext(p.ctx), a.Id(), owner); err != nil {
				return err
			}
			return mb.Put(asset.EnvEventTopicStatus, UpdatedEventStatusProvider(transactionId, characterId, updated))
		}
	}
}

func (p *Processor) ApplyLock(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a Model, expiration time.Time) error {
	return func(transactionId uuid.UUID, characterId uint32) func(a Model, expiration time.Time) error {
		return func(a Model, expiration time.Time) error {
			if !a.Locked() && !a.Expiration().IsZero() {
				return errors.New("asset has a non-lock expiration")
			}
			updated := Clone(a).AddFlag(af.FlagLock).SetExpiration(expiration).Build()
			if err := updateFlagAndExpiration(p.db.WithContext(p.ctx), a.Id(), updated.Flag(), expiration); err != nil {
				return err
			}
			return mb.Put(asset.EnvEventTopicStatus, UpdatedEventStatusProvider(transactionId, characterId, updated))
		}
	}
}
```

(Getter names per the actual builder: `AddFlag(af.Flag)`, `SetExpiration(time.Time)` exist — builder.go:103,165. `UpdatedEventStatusProvider(transactionId, characterId, a Model)` — producer.go:116.)

4. `compartment/processor.go`: add `SetAssetOwner`/`SetAssetOwnerAndEmit` and `ApplyAssetLock`/`ApplyAssetLockAndEmit` following `ExpireAsset`'s exact skeleton (:914-970): per-character inventory lock, `database.ExecuteTransaction`, `GetByCharacterAndType`, `assetProcessor.WithTransaction(tx).GetBySlot(c.Id(), slot)`, then delegate to `UpdateOwner`/`ApplyLock` above. Signatures:

```go
SetAssetOwnerAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, owner string) error
ApplyAssetLockAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, expiration time.Time) error
```

5. `kafka/consumer/compartment/consumer.go`: register two handlers in `InitHandlers` (:31-93) and implement them following `handleExpireCommand` (:309-334) — guard `c.Type != compartment2.CommandSetOwner` (resp. `CommandApplyLock`), then call the matching `AndEmit`.

- [ ] **Step 4: Run tests**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test -race ./... -count=1`
Expected: PASS including the three new tests.

- [ ] **Step 5: Update kafka docs + commit**

Add `SET_OWNER`/`APPLY_LOCK` rows to the inventory service's Kafka commands doc (locate with `grep -rl "MODIFY_EQUIPMENT" services/atlas-inventory --include="*.md"`).

```bash
git add services/atlas-inventory/
git commit -m "feat(inventory): SET_OWNER and APPLY_LOCK compartment commands"
```

---

### Task 8: Lock-aware expiration (atlas-inventory)

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/compartment/processor.go` (`ExpireAsset`, :920-970)
- Modify: `services/atlas-inventory/atlas.com/inventory/asset/processor.go`
- Test: `compartment/processor_test.go`

**Interfaces:**
- Consumes: Task 7's `updateFlagAndExpiration`; `af.ClearFlag` semantics via builder `RemoveFlag` (builder.go:169).
- Produces: expired **locked** assets are unlocked (flag cleared, expiration zeroed, `UPDATED` emitted) instead of destroyed; unlocked assets keep today's destroy/replace behavior. atlas-asset-expiration needs no change — it only compares `expiration` vs `now` and emits `EXPIRE` (verified: `asset-expiration/expiration/checker.go:11-19`, `character/processor.go:57`).

- [ ] **Step 1: Write the failing tests**

In `compartment/processor_test.go`:

```go
// A locked asset whose (lock) expiration passes is unlocked, not destroyed.
func TestExpireAssetLockedClearsLock(t *testing.T) { /* arrange equip asset with
	FlagLock set and expiration in the past; call ExpireAssetAndEmit(...);
	assert the asset row still exists, Locked() == false, Expiration().IsZero() */ }

// An unlocked expiring asset still destroys (existing behavior preserved).
func TestExpireAssetUnlockedStillDestroys(t *testing.T) { /* arrange unlocked asset
	with past expiration; ExpireAssetAndEmit; assert GetBySlot returns not-found */ }
```

Arrange sections copy the sqlite setup used by the file's existing expire/destroy tests.

- [ ] **Step 2: Run to verify failure**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test ./compartment/ -run TestExpireAsset -v`
Expected: `TestExpireAssetLockedClearsLock` FAILs (asset destroyed); the unlocked test passes.

- [ ] **Step 3: Implement**

1. `asset/processor.go`: add `ClearLock`, mirror of Task 7's `ApplyLock`:

```go
func (p *Processor) ClearLock(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a Model) error {
	return func(transactionId uuid.UUID, characterId uint32) func(a Model) error {
		return func(a Model) error {
			updated := Clone(a).RemoveFlag(af.FlagLock).SetExpiration(time.Time{}).Build()
			if err := updateFlagAndExpiration(p.db.WithContext(p.ctx), a.Id(), updated.Flag(), time.Time{}); err != nil {
				return err
			}
			return mb.Put(asset.EnvEventTopicStatus, UpdatedEventStatusProvider(transactionId, characterId, updated))
		}
	}
}
```

2. `compartment/processor.go` `ExpireAsset` (:920-970): immediately after `GetBySlot` succeeds, branch:

```go
			if a.Locked() {
				// Lock expiration passed: clear the lock and keep the asset (PRD 4.2.5).
				return p.assetProcessor.WithTransaction(tx).ClearLock(mb)(transactionId, characterId)(a)
			}
```

The `return` inside the transaction closure skips both the destroy (`Expire`) and the replacement-item branch.

- [ ] **Step 4: Run tests**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test -race ./... -count=1`
Expected: PASS.

- [ ] **Step 5: Verify module + commit**

MODULE-VERIFY `services/atlas-inventory/atlas.com/inventory`, then:

```bash
git add services/atlas-inventory/
git commit -m "feat(inventory): expired locked assets unlock instead of destroy"
```

---

### Task 9: Orchestrator saga actions + asset UPDATED acceptance (atlas-saga-orchestrator)

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` (aliases + local unmarshal switch, :862-940)
- Modify: `saga/handler.go` (`GetHandler` :704-864 + two handlers)
- Modify: `saga/event_acceptance.go`
- Modify: `compartment/processor.go`, `compartment/producer.go`
- Modify: `kafka/message/compartment/kafka.go` (orchestrator's mirror of the inventory command defs)
- Modify: `kafka/consumer/asset/consumer.go`
- Test: existing completeness tests + `saga/event_acceptance_test.go` style additions

**Interfaces:**
- Consumes: Task 5 consts/payloads; Task 7 command shapes (mirror the body structs byte-for-byte — same JSON tags).
- Produces: `RequestSetOwner(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, owner string) error` and `RequestApplyLock(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, expiration time.Time) error` on the compartment Processor; `EventKindAssetUpdated`; the actions wired end-to-end so a `SET_OWNER`/`APPLY_LOCK` step completes when the asset `UPDATED` event arrives.

- [ ] **Step 1: Run the completeness tests to see the expected failures**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run 'Completeness|Acceptance' -v`
Expected: FAIL — the new shared-lib actions (Task 5) lack local unmarshal cases and acceptance entries. These failing tests are this task's primary "failing test" gate; note exactly which tests fail.

- [ ] **Step 2: Implement**

1. `saga/model.go` alias blocks (:25-44 for types, and the action/payload alias blocks near them): add

```go
	ItemTagUse     = sharedsaga.ItemTagUse
	SealingLockUse = sharedsaga.SealingLockUse
	IncubatorUse   = sharedsaga.IncubatorUse

	SetAssetOwner   = sharedsaga.SetAssetOwner
	ApplyAssetLock  = sharedsaga.ApplyAssetLock
	IncubatorResult = sharedsaga.IncubatorResult
```

and payload aliases:

```go
	SetAssetOwnerPayload   = sharedsaga.SetAssetOwnerPayload
	ApplyAssetLockPayload  = sharedsaga.ApplyAssetLockPayload
	IncubatorResultPayload = sharedsaga.IncubatorResultPayload
```

2. `saga/model.go` local `UnmarshalJSON` switch (:886-940): add the three cases following the `DestroyAsset` case shape (:929-934).
3. `kafka/message/compartment/kafka.go` (orchestrator side): add `CommandSetOwner`/`CommandApplyLock` consts and `SetOwnerCommandBody`/`ApplyLockCommandBody` structs — identical to Task 7's definitions.
4. `compartment/producer.go`: add two providers following `RequestDestroyAssetCommandProvider` (:35-49):

```go
func RequestSetOwnerCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, owner string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.SetOwnerCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          compartment.CommandSetOwner,
		Body:          compartment.SetOwnerCommandBody{Slot: slot, Owner: owner},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestApplyLockCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, expiration time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.ApplyLockCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          compartment.CommandApplyLock,
		Body:          compartment.ApplyLockCommandBody{Slot: slot, Expiration: expiration},
	}
	return producer.SingleMessageProvider(key, value)
}
```

5. `compartment/processor.go`: add `RequestSetOwner`/`RequestApplyLock` to the `Processor` interface (:31-41) and impl — each is a one-liner emitting the provider on `compartment.EnvCommandTopic` (mirror the tail of `RequestDestroyItem` :68-98; no slot lookup needed, the payload carries the slot). Update any compartment processor mock in the orchestrator if one exists (`grep -rn "RequestDestroyItem" services/atlas-saga-orchestrator --include="*mock*"`).
6. `saga/handler.go`: add to `GetHandler` (:704-864):

```go
	case SetAssetOwner:
		return h.handleSetAssetOwner, true
	case ApplyAssetLock:
		return h.handleApplyAssetLock, true
```

and the handlers, following `handleDestroyAsset` (:1018-1033) — async, NO `StepCompleted`:

```go
func (h *HandlerImpl) handleSetAssetOwner(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(SetAssetOwnerPayload)
	if !ok {
		return errors.New("invalid payload")
	}
	err := h.compP.RequestSetOwner(s.TransactionId(), payload.CharacterId, payload.InventoryType, payload.Slot, payload.Owner)
	if err != nil {
		h.logActionError(s, st, err, "Unable to set asset owner.")
		return err
	}
	return nil
}

func (h *HandlerImpl) handleApplyAssetLock(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ApplyAssetLockPayload)
	if !ok {
		return errors.New("invalid payload")
	}
	err := h.compP.RequestApplyLock(s.TransactionId(), payload.CharacterId, payload.InventoryType, payload.Slot, payload.Expiration)
	if err != nil {
		h.logActionError(s, st, err, "Unable to apply asset lock.")
		return err
	}
	return nil
}
```

7. `saga/event_acceptance.go`: add `EventKindAssetUpdated` to the asset EventKind consts (:28-31, match the naming/value style of `EventKindAssetCreated` exactly) and acceptance entries:

```go
	sharedsaga.SetAssetOwner:  {EventKindAssetUpdated},
	sharedsaga.ApplyAssetLock: {EventKindAssetUpdated},
```

(`IncubatorResult: {}` is added in Task 10 with its handler.)
8. `kafka/consumer/asset/consumer.go`: register one more handler in `InitHandlers` (:75-101) alongside the existing four, and implement it following `handleAssetDeletedEvent` (:220-261):

```go
func handleAssetUpdatedEvent(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.UpdatedStatusEventBody]) {
	if e.Type != asset2.StatusEventTypeUpdated {
		return
	}
	p := saga.NewProcessor(l, ctx)
	_, ok := p.AcceptEvent(e.TransactionId, saga.EventKindAssetUpdated)
	if !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}
```

(The `AcceptEvent` gate means unrelated `UPDATED` traffic — e.g. `MODIFY_EQUIPMENT` — is ignored unless the saga's current step accepts `asset_updated`.)

- [ ] **Step 3: Run tests**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... -count=1`
Expected: PASS — completeness/acceptance tests green again. If an acceptance-coverage test demands an entry for `IncubatorResult` already, add `sharedsaga.IncubatorResult: {}` here instead of Task 10.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-saga-orchestrator/
git commit -m "feat(saga-orchestrator): SetAssetOwner/ApplyAssetLock actions complete on asset UPDATED"
```

---

### Task 10: Incubator result event + compensation (atlas-saga-orchestrator)

**Files:**
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/incubator/kafka.go`
- Modify: `saga/handler.go`, `saga/producer.go`, `saga/event_acceptance.go`, `saga/compensator.go`
- Test: compensator/acceptance test files in `saga/` (match existing style)

**Interfaces:**
- Consumes: Task 5/9; `handleSendMessage` fire-and-forget precedent (handler.go:1651-1668); `compensatePetEvolution`/`DispatchPetEvolutionRollbacks` (compensator.go:1053-1140).
- Produces: `EVENT_TOPIC_INCUBATOR_RESULT` event (consumed by Task 17 — struct must match):

```go
const EnvEventTopicIncubatorResult = "EVENT_TOPIC_INCUBATOR_RESULT"

type ResultEvent struct {
	CharacterId uint32 `json:"characterId"`
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	ItemId      uint32 `json:"itemId"`
	Count       uint32 `json:"count"`
}
```

- [ ] **Step 1: Write the failing test**

Add a handler test near the existing handler/compensator tests (find one with `grep -n "handleSendMessage\|GetHandler" services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/*_test.go` and mirror it): assert `GetHandler(IncubatorResult)` returns a handler, and (if the test seams allow producer injection) that the handler marks the step complete. Minimum viable failing test:

```go
func TestGetHandlerIncubatorResult(t *testing.T) {
	h := NewHandler(l, ctx) // mirror the construction used by existing GetHandler tests
	_, ok := h.GetHandler(IncubatorResult)
	if !ok {
		t.Fatal("IncubatorResult handler not registered")
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run IncubatorResult -v`
Expected: FAIL.

- [ ] **Step 3: Implement**

1. Create `kafka/message/incubator/kafka.go` with the const + struct from **Interfaces**.
2. `saga/producer.go`: add, mirroring `GachaponRewardWonEventProvider`:

```go
func IncubatorResultEventProvider(payload IncubatorResultPayload) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(payload.CharacterId))
	value := &incubator2.ResultEvent{
		CharacterId: payload.CharacterId,
		WorldId:     payload.WorldId,
		ChannelId:   payload.ChannelId,
		ItemId:      payload.ItemId,
		Count:       payload.Count,
	}
	return producer.SingleMessageProvider(key, value)
}
```

3. `saga/handler.go`: `GetHandler` case + fire-and-forget handler (mirrors `handleSendMessage` :1651-1668):

```go
func (h *HandlerImpl) handleIncubatorResult(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(IncubatorResultPayload)
	if !ok {
		return errors.New("invalid payload")
	}
	err := producer.ProviderImpl(h.l)(h.ctx)(incubator2.EnvEventTopicIncubatorResult)(IncubatorResultEventProvider(payload))
	if err != nil {
		h.logActionError(s, st, err, "Unable to emit incubator result event.")
		return err
	}
	// Fire-and-forget: the channel consumer only announces a packet, no response event.
	_ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)
	return nil
}
```

4. `saga/event_acceptance.go`: add `sharedsaga.IncubatorResult: {},` to the self-completing block (:171-200, next to `EmitGachaponWin: {}`), if not already added in Task 9.
5. `saga/compensator.go`: in `CompensateFailedStep` (:170-294), add a saga-type special case next to the `PetEvolution` one (:195-197):

```go
	if s.SagaType() == ItemTagUse || s.SagaType() == SealingLockUse || s.SagaType() == IncubatorUse {
		return c.compensateCashItemUse(s, failedStep)
	}
```

and implement `compensateCashItemUse` + `DispatchCashItemUseRollbacks` mirroring `compensatePetEvolution` (:1053-1090) and `DispatchPetEvolutionRollbacks` (:1105-1140): reverse-walk completed steps —

```go
		case DestroyAsset:
			// re-create the consumed cash item
			if payload, ok := step.Payload().(DestroyAssetPayload); ok {
				qty := payload.Quantity
				if qty == 0 {
					qty = 1
				}
				if err := c.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, time.Time{}); err != nil {
					c.l.WithError(err).Error("Reverse-walk: DestroyAsset -> CreateItem dispatch failed; continuing chain.")
				}
			}
		case DestroyAssetFromSlot:
			// re-create the sacrificed target (TemplateId added to the payload in this task's lib change)
			if payload, ok := step.Payload().(DestroyAssetFromSlotPayload); ok {
				if payload.TemplateId == 0 {
					c.l.Error("Reverse-walk: DestroyAssetFromSlot payload has no templateId; cannot re-create.")
					continue
				}
				qty := payload.Quantity
				if qty == 0 {
					qty = 1
				}
				if err := c.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, time.Time{}); err != nil {
					c.l.WithError(err).Error("Reverse-walk: DestroyAssetFromSlot -> CreateItem dispatch failed; continuing chain.")
				}
			}
		case AwardAsset:
			// invert a granted reward (mirrors DispatchCharacterCreationRollbacks)
			if payload, ok := step.Payload().(AwardItemActionPayload); ok {
				if err := c.compP.RequestDestroyItem(s.TransactionId(), payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity, false); err != nil {
					c.l.WithError(err).Error("Reverse-walk: AwardAsset -> DestroyItem dispatch failed; continuing chain.")
				}
			}
```

The terminal wrapper copies `compensatePetEvolution`'s lifecycle exactly: `TryTransition(SagaLifecycleCompensating, SagaLifecycleFailed)` guard → `SagaTimers().Cancel` → `GetCache().Remove` → `EmitSagaFailed(...)` (the FAILED event is what triggers the channel's `INCUBATOR_RESULT(0)` in Task 17). Add the new method(s) to the `Compensator` interface (:23-57). If `DestroyAssetFromSlotPayload` needs a local alias in the orchestrator saga package, add it next to the others.

- [ ] **Step 4: Run tests**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... -count=1`
Expected: PASS.

- [ ] **Step 5: MODULE-VERIFY + kafka docs + commit**

MODULE-VERIFY `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`; document `EVENT_TOPIC_INCUBATOR_RESULT` in the orchestrator's kafka doc if one exists.

```bash
git add services/atlas-saga-orchestrator/
git commit -m "feat(saga-orchestrator): incubator result event and cash-item-use compensation"
```

---

### Task 11: incubator-rewards configuration resource (atlas-tenants)

**Files:**
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/rest.go`, `provider.go`, `processor.go`, `resource.go`, `kafka.go`, `seed.go`
- Modify: `services/atlas-tenants/atlas.com/tenants/rest/handler.go`
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/mock/processor.go`
- Create: `services/atlas-tenants/configurations/incubator-rewards/*.json` (6 files)
- Test: extend the existing configuration tests (`grep -l "Vessel" services/atlas-tenants/atlas.com/tenants/configuration/*_test.go`)

**Interfaces:**
- Produces: `GET/POST /tenants/{tenantId}/configurations/incubator-rewards`, `GET/PATCH/DELETE .../incubator-rewards/{incubatorRewardId}`, `POST .../incubator-rewards/seed`. Entry attributes `itemId`, `quantity`, `weight` (all uint32). Resource name `"incubator-rewards"`, JSON:API type `"incubator-rewards"`. Consumed by Task 15's channel client.
- No Dockerfile change: the root Dockerfile copies the whole `configurations/` dir in a loop (Dockerfile:135-142, :161) — the new subdir lands at `/configurations/incubator-rewards` automatically. (Design §8's "COPY line" instruction is stale — verified.)
- No ingress change: routes live under the already-proxied atlas-tenants path.

This is a mechanical mirror of the **vessels** resource. For every step, `vessels`→`incubator-rewards`, `Vessel`→`IncubatorReward`, `vesselId`→`incubatorRewardId`.

- [ ] **Step 1: Write the failing test**

Mirror an existing vessel processor test (create/get-all round-trip through a sqlite/mock DB per the file's existing style):

```go
func TestIncubatorRewardCreateAndGetAll(t *testing.T) {
	// arrange per the existing vessel test's setup
	reward := map[string]interface{}{
		"type": "incubator-rewards",
		"id":   "red-potion",
		"attributes": map[string]interface{}{
			"itemId": float64(2000000), "quantity": float64(50), "weight": float64(40),
		},
	}
	_, err := processor.CreateIncubatorRewardAndEmit(tenantId, reward)
	// assert GetAllIncubatorRewards returns 1 entry with itemId 2000000
}
```

Also a `TransformIncubatorReward`/`ExtractIncubatorReward` round-trip test asserting the three uint32 attributes survive.

- [ ] **Step 2: Run to verify failure**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test ./configuration/... -run IncubatorReward -v`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Implement (mirror vessels at every site)**

1. `configuration/rest.go` (vessels block at :146-225) — add:

```go
// IncubatorRewardRestModel is the JSON:API resource for incubator reward pool entries
type IncubatorRewardRestModel struct {
	Id       string `json:"-"`
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
	Weight   uint32 `json:"weight"`
}

func (v IncubatorRewardRestModel) GetID() string { return v.Id }

func (v *IncubatorRewardRestModel) SetID(id string) error {
	v.Id = id
	return nil
}

func (v IncubatorRewardRestModel) GetName() string { return "incubator-rewards" }

func TransformIncubatorReward(data map[string]interface{}) (IncubatorRewardRestModel, error) {
	id, _ := data["id"].(string)
	attributes, ok := data["attributes"].(map[string]interface{})
	if !ok {
		attributes = make(map[string]interface{})
	}
	itemId := uint32(0)
	if val, ok := attributes["itemId"].(float64); ok {
		itemId = uint32(val)
	}
	quantity := uint32(0)
	if val, ok := attributes["quantity"].(float64); ok {
		quantity = uint32(val)
	}
	weight := uint32(0)
	if val, ok := attributes["weight"].(float64); ok {
		weight = uint32(val)
	}
	return IncubatorRewardRestModel{Id: id, ItemId: itemId, Quantity: quantity, Weight: weight}, nil
}

func ExtractIncubatorReward(v IncubatorRewardRestModel) (map[string]interface{}, error) {
	return map[string]interface{}{
		"type": "incubator-rewards",
		"id":   v.Id,
		"attributes": map[string]interface{}{
			"itemId":   v.ItemId,
			"quantity": v.Quantity,
			"weight":   v.Weight,
		},
	}, nil
}

func CreateIncubatorRewardJsonData(rewards []map[string]interface{}) (json.RawMessage, error) {
	return json.Marshal(map[string]interface{}{"data": rewards})
}

func CreateSingleIncubatorRewardJsonData(reward map[string]interface{}) (json.RawMessage, error) {
	return CreateIncubatorRewardJsonData([]map[string]interface{}{reward})
}
```

2. `configuration/provider.go`: `GetIncubatorRewardByIdProvider` + `GetAllIncubatorRewardsProvider` — copy `GetVesselByIdProvider`/`GetAllVesselsProvider` (:87-150) verbatim with resource name `"incubator-rewards"`.
3. `configuration/processor.go`: add the 11 interface methods (`CreateIncubatorReward(+AndEmit)`, `UpdateIncubatorReward(+AndEmit)`, `DeleteIncubatorReward(+AndEmit)`, `GetIncubatorRewardById`, `GetAllIncubatorRewards`, `IncubatorRewardByIdProvider`, `AllIncubatorRewardsProvider`, `SeedIncubatorRewards`) and impls — copy the vessel impls (`CreateVessel` :344-437, `CreateVesselAndEmit` :439-446, `UpdateVessel` :448-529, `DeleteVessel` :531-554, getters :556-573, `SeedVessels` :870-906) substituting names, resource string, event provider, and `LoadIncubatorRewardFiles()`.
4. `configuration/resource.go`: 6 handlers (copy `GetAllVesselsHandler` :203-240, `GetVesselByIdHandler` :242-271, `CreateVesselHandler` :273-321, `UpdateVesselHandler` :323-366, `DeleteVesselHandler` :368-387, `SeedVesselsHandler` :615-635) + in `RegisterRoutes` (:637-660) a `registerIncubatorRewardInputHandler := rest.RegisterInputHandler[IncubatorRewardRestModel](l)(si)` and six routes — `/seed` and the collection routes BEFORE the `/{incubatorRewardId}` routes (mux matches in registration order):

```go
			r.HandleFunc("/tenants/{tenantId}/configurations/incubator-rewards/seed", registerHandler("seed_incubator_rewards", SeedIncubatorRewardsHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/incubator-rewards", registerHandler("get_all_incubator_rewards", GetAllIncubatorRewardsHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/incubator-rewards/{incubatorRewardId}", registerHandler("get_incubator_reward_by_id", GetIncubatorRewardByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/incubator-rewards", registerIncubatorRewardInputHandler("create_incubator_reward", CreateIncubatorRewardHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/incubator-rewards/{incubatorRewardId}", registerIncubatorRewardInputHandler("update_incubator_reward", UpdateIncubatorRewardHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/incubator-rewards/{incubatorRewardId}", registerHandler("delete_incubator_reward", DeleteIncubatorRewardHandler(db))).Methods(http.MethodDelete)
```

5. `rest/handler.go` (:40-42 pattern):

```go
func ParseIncubatorRewardId(l logrus.FieldLogger, next func(string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "incubatorRewardId", next)
}
```

6. `configuration/kafka.go`: consts `EventTypeIncubatorRewardCreated = "INCUBATOR_REWARD_CREATED"` / `...Updated` / `...Deleted` + `CreateIncubatorRewardStatusEventProvider` (copy `CreateVesselStatusEventProvider` :32-53, `ResourceType: "incubator-reward"`).
7. `configuration/seed.go`: `const defaultIncubatorRewardsPath = "/configurations/incubator-rewards"`, `getIncubatorRewardsPath()` (env `INCUBATOR_REWARDS_SEED_PATH`), `LoadIncubatorRewardFiles()` — copy :51-63.
8. `configuration/mock/processor.go`: add all 11 func fields + methods (copy the vessel mocks :46-55, :156-174, :368-372). The `var _ configuration.Processor` check at :12 will not compile until complete.
9. Seed data — create `services/atlas-tenants/configurations/incubator-rewards/` with six files (WZ-verified v83 ids, design §8):

`red-potion.json`:

```json
{
  "id": "red-potion",
  "type": "incubator-rewards",
  "attributes": {
    "itemId": 2000000,
    "quantity": 50,
    "weight": 40
  }
}
```

and equally-shaped `orange-potion.json` (2000001, 50, 30), `white-potion.json` (2000003, 50, 15), `scroll-2040000.json` (2040000, 1, 10), `cap-1002000.json` (1002000, 1, 4), `sword-1302000.json` (1302000, 1, 1).

- [ ] **Step 4: Run tests**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test -race ./... -count=1`
Expected: PASS.

- [ ] **Step 5: Docs + MODULE-VERIFY + commit**

Add the resource to `services/atlas-tenants/docs/kafka.md` (event types) and the service README's endpoint table.

```bash
git add services/atlas-tenants/
git commit -m "feat(tenants): incubator-rewards configuration resource with seed pool"
```

---

### Task 12: protectTime in atlas-data cash reader (+ channel client field)

**Files:**
- Modify: `services/atlas-data/atlas.com/data/cash/reader.go` (:75 area), `cash/rest.go` (:37-44)
- Modify: `services/atlas-channel/atlas.com/channel/data/cash/rest.go`
- Test: `services/atlas-data/atlas.com/data/cash/reader_test.go`

**Interfaces:**
- Produces: atlas-data cash item REST payload gains `protectTime` (days, uint32, 0 = absent); channel `data/cash.RestModel.ProtectTime uint32`. Consumed by Task 16's seal arm.
- WZ ground truth (verified against `Item.wz/Cash/0506.img.xml`, design §1): `protectTime` is an `info`-block int — 5061000=7, 5061001=30, 5061002=90, 5061003=365; the unit is days.

- [ ] **Step 1: Write the failing test**

In `reader_test.go`, add a fixture + test mirroring `testExpCouponXML`/`TestReaderExpCoupons` (:413-487, :574-691):

```go
const testSealingLockXML = `<?xml version="1.0" encoding="UTF-8"?>
<imgdir name="0506.img">
  <imgdir name="05061000">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="protectTime" value="7"/>
    </imgdir>
  </imgdir>
  <imgdir name="05061001">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="protectTime" value="30"/>
    </imgdir>
  </imgdir>
</imgdir>`

func TestReaderProtectTime(t *testing.T) {
	l, _ := test.NewNullLogger()
	rms := Read(l)(xml.FromByteArrayProvider([]byte(testSealingLockXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	if got := rmm[strconv.Itoa(5061000)].ProtectTime; got != 7 {
		t.Fatalf("ProtectTime(5061000) = %d, want 7", got)
	}
	if got := rmm[strconv.Itoa(5061001)].ProtectTime; got != 30 {
		t.Fatalf("ProtectTime(5061001) = %d, want 30", got)
	}
}
```

Match the outer XML wrapper structure of the existing fixtures exactly (copy `testExpCouponXML`'s root/entry shape).

- [ ] **Step 2: Run to verify failure**

Run: `cd services/atlas-data/atlas.com/data && go test ./cash/ -run ProtectTime -v`
Expected: FAIL — `ProtectTime` undefined.

- [ ] **Step 3: Implement**

1. `cash/rest.go` `RestModel` (:37-44): add `ProtectTime uint32 \`json:"protectTime,omitempty"\``.
2. `cash/reader.go`: next to `m.SlotMax` (:75) add `m.ProtectTime = uint32(i.GetIntegerWithDefault("protectTime", 0))`.
3. Channel client `services/atlas-channel/atlas.com/channel/data/cash/rest.go`: add `ProtectTime uint32 \`json:"protectTime"\`` to its `RestModel`.

- [ ] **Step 4: Run tests + verify + commit**

Run: `cd services/atlas-data/atlas.com/data && go test -race ./cash/ -count=1` → PASS. MODULE-VERIFY `services/atlas-data/atlas.com/data`.

```bash
git add services/atlas-data/ services/atlas-channel/atlas.com/channel/data/cash/rest.go
git commit -m "feat(data): expose cash item protectTime (seal durations)"
```

---

### Task 13: Owner in the atlas-channel asset projection

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/asset/model.go`, `asset/builder.go`, `asset/rest.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/asset/kafka.go` (`AssetData` at :36 area — and the second struct at :70 if it repeats the field block)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go` (`buildAssetFromCreatedBody` :119, `buildAssetFromUpdatedBody` :157, `buildAssetFromAcceptedBody` :195)
- Modify: `services/atlas-channel/atlas.com/channel/socket/model/asset.go` (`NewAsset` :13)
- Test: `services/atlas-channel/atlas.com/channel/asset/builder_test.go`

**Interfaces:**
- Consumes: Task 2 (`packetmodel.Asset.SetOwner`), Task 6 (event `AssetData.Owner`, REST `owner`).
- Produces: channel `asset.Model.Owner() string`; equip packets encode the owner on inventory load (REST path) and on live updates (Kafka path).

- [ ] **Step 1: Write the failing test**

In `asset/builder_test.go` (mirror existing tests):

```go
func TestChannelAssetOwner(t *testing.T) {
	m := NewModelBuilder(1, uuid.New(), 1302000).SetOwner("Tumi").MustBuild()
	if m.Owner() != "Tumi" {
		t.Fatalf("Owner() = %q", m.Owner())
	}
	if c := Clone(m).MustBuild(); c.Owner() != "Tumi" {
		t.Fatalf("Clone dropped owner: %q", c.Owner())
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./asset/ -run TestChannelAssetOwner -v`
Expected: FAIL.

- [ ] **Step 3: Implement**

1. `asset/model.go`: `owner string` field + `Owner() string` getter (place with `ownerId`).
2. `asset/builder.go`: `SetOwner(string)` setter, `Clone` copy (:15), `Build`/`MustBuild` emit.
3. `asset/rest.go`: `Owner string \`json:"owner"\`` in `RestModel` (:8-48); `Owner: m.owner` in `Transform`; `owner: rm.Owner` in `Extract`. (This is the relog path: character REST inventory → equip packets.)
4. `kafka/message/asset/kafka.go`: add `Owner string \`json:"owner"\`` beside `OwnerId` in `AssetData`; check line :70's struct and mirror there if it duplicates the field block.
5. `kafka/consumer/asset/consumer.go`: add `.SetOwner(e.Body.Owner)` to the builder chains in `buildAssetFromCreatedBody`, `buildAssetFromUpdatedBody`, `buildAssetFromAcceptedBody`.
6. `socket/model/asset.go` `NewAsset` (:13): after `base := packetmodel.NewAsset(...)` add `base = base.SetOwner(a.Owner())` (unconditional — empty for non-tagged/stackables).

- [ ] **Step 4: Run tests + verify + commit**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./... -count=1` → PASS.

```bash
git add services/atlas-channel/
git commit -m "feat(channel): project asset owner through model, events, and equip packets"
```

---

### Task 14: Owner in the remaining AssetData mirrors (storage, merchant)

**Files:**
- Modify: `services/atlas-storage/atlas.com/storage/kafka/message/kafka.go` (`AssetData` :50), plus `services/atlas-storage/atlas.com/storage/asset/{entity,model,builder,rest}.go` and its administrator/producer (storage persists per-field — entity has `OwnerId`/`Flag` at entity.go:23-24)
- Modify: `services/atlas-merchant/atlas.com/merchant/kafka/message/asset/kafka.go` (`AssetData`) and, if the merchant service persists assets per-field, its `asset/` entity/model/builder chain (locate with `ls services/atlas-merchant/atlas.com/merchant/asset/`)
- Modify: `services/atlas-channel/atlas.com/channel/merchant/asset_data.go` (display mirror)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/asset/kafka.go` (`AssetData` pass-through mirror)
- Test: storage asset builder/rest tests (mirror the existing ones in `services/atlas-storage/atlas.com/storage/asset/`)

**Interfaces:**
- Consumes: Task 6's `AssetData.Owner` JSON contract (`json:"owner"`).
- Produces: a tagged equip deposited to storage (or listed with a merchant) keeps its owner through the round-trip. Without this, the mirrors silently drop the field.

Rationale: `grep -rln "type AssetData struct"` shows exactly five mirrors — inventory (done in Task 6), channel (done in Task 13), storage, atlas-merchant, channel merchant display, plus the orchestrator's message mirror. This task finishes the set.

- [ ] **Step 1: Write the failing test (storage)**

Mirror the existing storage asset builder/rest test style: build a storage asset model with owner, assert `Owner()` and REST round-trip. If storage's `Make(entity)` has a test, extend it with `Owner`.

- [ ] **Step 2: Run to verify failure**

Run: `cd services/atlas-storage/atlas.com/storage && go test ./asset/ -v -run Owner`
Expected: FAIL.

- [ ] **Step 3: Implement**

For **storage** (persists per-field): add `Owner string \`json:"owner"\`` to `kafka/message/kafka.go` `AssetData` (:50, beside `OwnerId` at :53); add `Owner string \`gorm:"not null;default:''"\`` to `asset/entity.go` (:23 area) + `Make`; `owner` field/getter/builder-setter in `asset/model.go`/`builder.go`; REST attribute in `asset/rest.go`; thread through wherever storage builds a model from `AssetData` (grep `OwnerId` in `services/atlas-storage/atlas.com/storage/` and add `Owner` at every construction site the grep surfaces) and wherever it emits `AssetData` back out (release path).

For **atlas-merchant**: same recipe — add the field to its `kafka/message/asset/kafka.go` `AssetData`, then `grep -rn "OwnerId" services/atlas-merchant/atlas.com/merchant/` and mirror `Owner` at every site (entity/model/builder/rest if the service persists assets; message-only if it does not).

For **channel merchant display** (`services/atlas-channel/atlas.com/channel/merchant/asset_data.go`:8 area): add `Owner string \`json:"owner"\``.

For **orchestrator mirror** (`kafka/message/asset/kafka.go`): add `Owner string \`json:"owner"\`` to its `AssetData`.

- [ ] **Step 4: Run tests + verify + commit**

MODULE-VERIFY `services/atlas-storage/atlas.com/storage` and `services/atlas-merchant/atlas.com/merchant` (and re-run channel + orchestrator module tests).

```bash
git add services/atlas-storage/ services/atlas-merchant/ services/atlas-channel/atlas.com/channel/merchant/ services/atlas-saga-orchestrator/
git commit -m "feat: thread asset owner through storage/merchant AssetData mirrors"
```

---

### Task 15: Incubator rewards client + weighted roll (atlas-channel)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/incubator/rest.go`, `incubator/requests.go`, `incubator/processor.go`, `incubator/roll.go`
- Test: `services/atlas-channel/atlas.com/channel/incubator/roll_test.go`

**Interfaces:**
- Consumes: Task 11's REST resource; `requests.RootUrl("TENANTS")` + `requests.SliceProvider` pattern (atlas-transports `transport/config/{requests,rest,processor}.go` is the canonical template).
- Produces (used by Task 16):

```go
type Reward struct { itemId, quantity, weight uint32 } // with ItemId()/Quantity()/Weight() getters
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor // Processor.GetRewards() ([]Reward, error)
func PickWeighted(rewards []Reward, rollFn func(total uint32) uint32) (Reward, bool)
```

- [ ] **Step 1: Write the failing roll test**

`roll_test.go`:

```go
package incubator

import "testing"

func rewards() []Reward {
	return []Reward{
		{itemId: 1, quantity: 1, weight: 40},
		{itemId: 2, quantity: 1, weight: 10},
		{itemId: 3, quantity: 1, weight: 50},
	}
}

func TestPickWeightedBoundaries(t *testing.T) {
	cases := []struct {
		roll uint32
		want uint32
	}{
		{0, 1}, {39, 1}, // first bucket [0,40)
		{40, 2}, {49, 2}, // second bucket [40,50)
		{50, 3}, {99, 3}, // third bucket [50,100)
	}
	for _, c := range cases {
		r, ok := PickWeighted(rewards(), func(total uint32) uint32 {
			if total != 100 {
				t.Fatalf("total = %d, want 100", total)
			}
			return c.roll
		})
		if !ok || r.ItemId() != c.want {
			t.Errorf("roll %d -> item %d, want %d", c.roll, r.ItemId(), c.want)
		}
	}
}

func TestPickWeightedEmptyAndZeroWeight(t *testing.T) {
	if _, ok := PickWeighted(nil, func(uint32) uint32 { return 0 }); ok {
		t.Error("empty pool must not pick")
	}
	if _, ok := PickWeighted([]Reward{{itemId: 1, weight: 0}}, func(uint32) uint32 { return 0 }); ok {
		t.Error("zero-weight pool must not pick")
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./incubator/ -v`
Expected: FAIL (package missing).

- [ ] **Step 3: Implement**

`rest.go`:

```go
package incubator

type RewardRestModel struct {
	Id       string `json:"-"`
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
	Weight   uint32 `json:"weight"`
}

func (r RewardRestModel) GetName() string { return "incubator-rewards" }
func (r RewardRestModel) GetID() string   { return r.Id }
func (r *RewardRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

type Reward struct {
	itemId   uint32
	quantity uint32
	weight   uint32
}

func (r Reward) ItemId() uint32   { return r.itemId }
func (r Reward) Quantity() uint32 { return r.quantity }
func (r Reward) Weight() uint32   { return r.weight }

func Extract(rm RewardRestModel) (Reward, error) {
	q := rm.Quantity
	if q == 0 {
		q = 1
	}
	return Reward{itemId: rm.ItemId, quantity: q, weight: rm.Weight}, nil
}
```

`requests.go` (mirror `transport/config/requests.go`):

```go
package incubator

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func requestRewards(tenantId string) requests.Request[[]RewardRestModel] {
	return requests.GetRequest[[]RewardRestModel](fmt.Sprintf("%stenants/%s/configurations/incubator-rewards", requests.RootUrl("TENANTS"), tenantId))
}
```

(Verify the exact `requests` import path against `transport/config/requests.go` in atlas-transports or an existing channel `requests.go` and match it.)

`processor.go`:

```go
package incubator

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetRewards() ([]Reward, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) GetRewards() ([]Reward, error) {
	t := tenant.MustFromContext(p.ctx)
	return requests.SliceProvider[RewardRestModel, Reward](p.l, p.ctx)(requestRewards(t.Id().String()), Extract, model.Filters[Reward]())()
}
```

`roll.go`:

```go
package incubator

// PickWeighted selects a reward proportional to weight. rollFn receives the total
// weight and must return a value in [0, total). Returns false for an empty or
// zero-weight pool.
func PickWeighted(rewards []Reward, rollFn func(total uint32) uint32) (Reward, bool) {
	var total uint32
	for _, r := range rewards {
		total += r.Weight()
	}
	if total == 0 {
		return Reward{}, false
	}
	roll := rollFn(total)
	var acc uint32
	for _, r := range rewards {
		acc += r.Weight()
		if roll < acc {
			return r, true
		}
	}
	return rewards[len(rewards)-1], true
}
```

- [ ] **Step 4: Run tests + commit**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./incubator/ -count=1` → PASS.

```bash
git add services/atlas-channel/atlas.com/channel/incubator/
git commit -m "feat(channel): incubator-rewards config client and weighted roll"
```

---

### Task 16: Handler arms for tag / seal / incubator (atlas-channel)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`
- Modify: `services/atlas-channel/atlas.com/channel/saga/model.go` (aliases)
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (`produceWriters` :608)

**Interfaces:**
- Consumes: Tasks 1, 3, 4, 5, 12, 13, 15; the FieldEffect arm as the saga-building template (character_cash_item_use.go:60-106); `character2.NewProcessor(l, ctx).GetItemInSlot(characterId, invType, slot)` (:37 call-site precedent); `character2.NewProcessor(l, ctx).GetById()(characterId)` → `.Name()`; channel compartment processor `GetByType` (the same method `character.GetItemInSlot` uses internally — character/processor.go:208 `p.cp.GetByType`) with `Capacity()`/`Assets()` on the returned model (compartment/model.go:26,30).
- Produces: types 25/26/27/64/65 handled; 74 an explicit documented no-op; `IncubatorResultWriter` registered; the `// TODO for v83 there is a trailing updateTime.` at :108 removed (the sub-body codecs own the trailing read now).

- [ ] **Step 1: Add saga aliases**

In `services/atlas-channel/atlas.com/channel/saga/model.go`, extend the alias blocks (:9-60):

```go
	SetAssetOwnerPayload        = sharedsaga.SetAssetOwnerPayload
	ApplyAssetLockPayload       = sharedsaga.ApplyAssetLockPayload
	IncubatorResultPayload      = sharedsaga.IncubatorResultPayload
	DestroyAssetFromSlotPayload = sharedsaga.DestroyAssetFromSlotPayload
```

(skip any that already exist) and consts:

```go
	SetAssetOwner        = sharedsaga.SetAssetOwner
	ApplyAssetLock       = sharedsaga.ApplyAssetLock
	IncubatorResult      = sharedsaga.IncubatorResult
	DestroyAssetFromSlot = sharedsaga.DestroyAssetFromSlot
	ItemTagUse           = sharedsaga.ItemTagUse
	SealingLockUse       = sharedsaga.SealingLockUse
	IncubatorUse         = sharedsaga.IncubatorUse
```

- [ ] **Step 2: Register the writer**

In `main.go` `produceWriters()` (:608), append `incubatorcb.IncubatorResultWriter` (import `incubatorcb "github.com/Chronicle20/atlas/libs/atlas-packet/incubator/clientbound"`).

- [ ] **Step 3: Implement the arms**

In `character_cash_item_use.go`:

1. Change the discarded writer producer: `func CharacterCashItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) ...`.
2. Add named consts next to the existing three (:116-120):

```go
	CashSlotItemTypeItemTag      = CashSlotItemType(25)
	CashSlotItemTypeSeal         = CashSlotItemType(26)
	CashSlotItemTypeIncubator    = CashSlotItemType(27)
	CashSlotItemTypeSealTimed    = CashSlotItemType(64)
	CashSlotItemTypeSealTimedV95 = CashSlotItemType(65)
	CashSlotItemTypeCube         = CashSlotItemType(74)
```

3. Insert the arms before the fall-through warn, after the FieldEffect arm. Item Tag:

```go
		if it == CashSlotItemTypeItemTag {
			sp := cashsb.NewItemUseItemTag(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			targetSlot := sp.Slot()
			if targetSlot >= 0 {
				l.Warnf("Character [%d] attempted to use item tag [%d] on non-equipped slot [%d].", s.CharacterId(), itemId, targetSlot)
				return
			}
			target, err := character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), inventory.TypeValueEquip, targetSlot)()
			if err != nil {
				l.Warnf("Character [%d] attempted to use item tag [%d] on empty slot [%d].", s.CharacterId(), itemId, targetSlot)
				return
			}
			if tt, ok := inventory.TypeFromItemId(item.Id(target.TemplateId())); !ok || tt != inventory.TypeValueEquip {
				l.Warnf("Character [%d] attempted to use item tag [%d] on non-equip item [%d].", s.CharacterId(), itemId, target.TemplateId())
				return
			}
			c, err := character2.NewProcessor(l, ctx).GetById()(s.CharacterId())
			if err != nil {
				l.WithError(err).Warnf("Unable to resolve character [%d] name for item tag.", s.CharacterId())
				return
			}
			transactionId := uuid.New()
			now := time.Now()
			_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
				TransactionId: transactionId,
				SagaType:      saga.ItemTagUse,
				InitiatedBy:   "CASH_ITEM_USE",
				Steps: []saga.Step{
					{
						StepId: "consume_item_tag",
						Status: saga.Pending,
						Action: saga.DestroyAsset,
						Payload: saga.DestroyAssetPayload{
							CharacterId: s.CharacterId(),
							TemplateId:  uint32(itemId),
							Quantity:    1,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						StepId: "set_asset_owner",
						Status: saga.Pending,
						Action: saga.SetAssetOwner,
						Payload: saga.SetAssetOwnerPayload{
							CharacterId:   s.CharacterId(),
							InventoryType: byte(inventory.TypeValueEquip),
							Slot:          targetSlot,
							Owner:         c.Name(),
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			})
			return
		}
```

Sealing Lock:

```go
		if it == CashSlotItemTypeSeal || it == CashSlotItemTypeSealTimed || it == CashSlotItemTypeSealTimedV95 {
			sp := cashsb.NewItemUseSeal(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			invType := inventory.Type(sp.InventoryType())
			targetSlot := int16(sp.Slot())
			if invType != inventory.TypeValueEquip {
				l.Warnf("Character [%d] attempted to use sealing lock [%d] on non-equip inventory [%d].", s.CharacterId(), itemId, invType)
				return
			}
			target, err := character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), invType, targetSlot)()
			if err != nil {
				l.Warnf("Character [%d] attempted to use sealing lock [%d] on empty slot [%d].", s.CharacterId(), itemId, targetSlot)
				return
			}
			if !target.Expiration().IsZero() && !target.Locked() {
				// A genuinely time-limited item must not be laundered into a permanent one.
				l.Warnf("Character [%d] attempted to seal time-limited item [%d] in slot [%d].", s.CharacterId(), target.TemplateId(), targetSlot)
				return
			}
			expiration := time.Time{}
			cd, err := cashdata.NewProcessor(l, ctx).GetById(uint32(itemId))
			if err != nil {
				l.WithError(err).Warnf("Unable to resolve cash item data for sealing lock [%d].", itemId)
				return
			}
			if cd.ProtectTime > 0 {
				base := time.Now()
				if target.Locked() && !target.Expiration().IsZero() {
					base = target.Expiration()
				}
				expiration = base.AddDate(0, 0, int(cd.ProtectTime))
			}
			transactionId := uuid.New()
			now := time.Now()
			_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
				TransactionId: transactionId,
				SagaType:      saga.SealingLockUse,
				InitiatedBy:   "CASH_ITEM_USE",
				Steps: []saga.Step{
					{
						StepId: "consume_sealing_lock",
						Status: saga.Pending,
						Action: saga.DestroyAsset,
						Payload: saga.DestroyAssetPayload{
							CharacterId: s.CharacterId(),
							TemplateId:  uint32(itemId),
							Quantity:    1,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						StepId: "apply_asset_lock",
						Status: saga.Pending,
						Action: saga.ApplyAssetLock,
						Payload: saga.ApplyAssetLockPayload{
							CharacterId:   s.CharacterId(),
							InventoryType: byte(invType),
							Slot:          targetSlot,
							Expiration:    expiration,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			})
			return
		}
```

(`cashdata` = the existing channel client package `atlas-channel/data/cash`; check its actual import alias at other call sites — `grep -rn "data/cash" services/atlas-channel/atlas.com/channel --include="*.go" | grep import -i` — and match.)

Incubator:

```go
		if it == CashSlotItemTypeIncubator {
			sp := cashsb.NewItemUseIncubator(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			invType := inventory.Type(sp.InventoryType())
			targetSlot := int16(sp.Slot())
			announceFailure := func() {
				_ = session.Announce(l)(ctx)(wp)(incubatorcb.IncubatorResultWriter)(incubatorcb.NewIncubatorResult(0, 0).Encode)(s)
			}
			target, err := character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), invType, targetSlot)()
			if err != nil {
				l.Warnf("Character [%d] attempted to incubate empty slot [%d] of inventory [%d].", s.CharacterId(), targetSlot, invType)
				announceFailure()
				return
			}
			rewards, err := incubator.NewProcessor(l, ctx).GetRewards()
			if err != nil || len(rewards) == 0 {
				l.Warnf("Character [%d] used incubator but tenant has no reward pool.", s.CharacterId())
				announceFailure()
				return
			}
			reward, ok := incubator.PickWeighted(rewards, func(total uint32) uint32 {
				return uint32(rand.Intn(int(total)))
			})
			if !ok {
				l.Warnf("Character [%d] used incubator but reward pool has zero weight.", s.CharacterId())
				announceFailure()
				return
			}
			rewardInvType, ok := inventory.TypeFromItemId(item.Id(reward.ItemId()))
			if !ok {
				l.Warnf("Incubator reward [%d] has no inventory type.", reward.ItemId())
				announceFailure()
				return
			}
			cm, err := compartment.NewProcessor(l, ctx).GetByType(s.CharacterId(), rewardInvType)
			if err != nil || len(cm.Assets()) >= int(cm.Capacity()) {
				l.Warnf("Character [%d] used incubator with full [%d] inventory.", s.CharacterId(), rewardInvType)
				announceFailure()
				return
			}
			f := s.Field()
			transactionId := uuid.New()
			now := time.Now()
			_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
				TransactionId: transactionId,
				SagaType:      saga.IncubatorUse,
				InitiatedBy:   "CASH_ITEM_USE",
				Steps: []saga.Step{
					{
						StepId: "consume_sacrifice",
						Status: saga.Pending,
						Action: saga.DestroyAssetFromSlot,
						Payload: saga.DestroyAssetFromSlotPayload{
							CharacterId:   s.CharacterId(),
							InventoryType: byte(invType),
							Slot:          targetSlot,
							Quantity:      1,
							TemplateId:    target.TemplateId(),
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						StepId: "consume_incubator",
						Status: saga.Pending,
						Action: saga.DestroyAsset,
						Payload: saga.DestroyAssetPayload{
							CharacterId: s.CharacterId(),
							TemplateId:  uint32(itemId),
							Quantity:    1,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						StepId: "award_reward",
						Status: saga.Pending,
						Action: saga.AwardAsset,
						Payload: saga.AwardAssetPayload{
							CharacterId: s.CharacterId(),
							Item: saga.ItemPayload{
								TemplateId: reward.ItemId(),
								Quantity:   reward.Quantity(),
							},
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						StepId: "announce_result",
						Status: saga.Pending,
						Action: saga.IncubatorResult,
						Payload: saga.IncubatorResultPayload{
							CharacterId: s.CharacterId(),
							WorldId:     byte(f.WorldId()),
							ChannelId:   byte(f.ChannelId()),
							ItemId:      reward.ItemId(),
							Count:       reward.Quantity(),
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			})
			return
		}
```

(`compartment.NewProcessor(l, ctx).GetByType` — verify the channel compartment processor's actual constructor/method names against `character/processor.go:208`'s `p.cp` field type and use exactly those; if the processor is only reachable through the character processor, add a small public passthrough there instead.)

Cube (74) documented no-op:

```go
		if it == CashSlotItemTypeCube {
			// 5062xxx (GMS >= 95) is the Miracle Cube / potential re-roll family — a
			// separate feature, deliberately not part of task-128 (design.md §11).
			l.Warnf("Character [%d] attempted to use cube-family item [%d]; not implemented.", s.CharacterId(), itemId)
			return
		}
```

4. Delete the stale `// TODO for v83 there is a trailing updateTime.` comment at :108.
5. Imports: add `cashsb` sub-body types come from the already-imported cash serverbound package; add `incubator "atlas-channel/incubator"`, `incubatorcb ".../atlas-packet/incubator/clientbound"`, `math/rand`, `time`, `uuid`, `item`, `session` as needed (most already imported).

- [ ] **Step 4: Build + test + commit**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./... -count=1`
Expected: clean. (The arms are validated end-to-end in Task 20's acceptance; unit coverage lives in the codecs, roll, and inventory processors.)

```bash
git add services/atlas-channel/
git commit -m "feat(channel): item tag, sealing lock, and incubator cash item arms"
```

---

### Task 17: Incubator result consumer + saga-failure packet (atlas-channel)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/incubator/kafka.go`
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/incubator/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/saga/kafka.go`, `kafka/consumer/saga/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (:215 consumers, :545 handlers)

**Interfaces:**
- Consumes: Task 10's `EVENT_TOPIC_INCUBATOR_RESULT` / `ResultEvent` (struct mirrored verbatim); gachapon consumer as template (`kafka/consumer/gachapon/consumer.go`); `handleFailedEvent`'s saga-type branch (saga/consumer.go:78-130).
- Produces: success path announces `INCUBATOR_RESULT(itemId, count)` to the using client; failed `incubator_use` sagas announce `INCUBATOR_RESULT(0, 0)`.

- [ ] **Step 1: Create the message mirror**

`kafka/message/incubator/kafka.go` — identical to Task 10's definition:

```go
package incubator

const (
	EnvEventTopicIncubatorResult = "EVENT_TOPIC_INCUBATOR_RESULT"
)

type ResultEvent struct {
	CharacterId uint32 `json:"characterId"`
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	ItemId      uint32 `json:"itemId"`
	Count       uint32 `json:"count"`
}
```

- [ ] **Step 2: Create the consumer**

`kafka/consumer/incubator/consumer.go`, copying the gachapon consumer's `InitConsumers`/`InitHandlers` skeleton (consumer name `"incubator_result"`, topic env `incubator2.EnvEventTopicIncubatorResult`, `LastOffset`) with handler:

```go
func handleResult(sc server.Model, wp writer.Producer) message.Handler[incubator2.ResultEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, event incubator2.ResultEvent) {
		t := tenant.MustFromContext(ctx)
		if !sc.IsWorld(t, world.Id(event.WorldId)) {
			return
		}
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(event.CharacterId,
			session.Announce(l)(ctx)(wp)(incubatorcb.IncubatorResultWriter)(incubatorcb.NewIncubatorResult(event.ItemId, uint16(event.Count)).Encode))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce incubator result to character [%d].", event.CharacterId)
		}
	}
}
```

(Verify `sc.IsWorld` and `IfPresentByCharacterId` signatures against the gachapon consumer / asset consumer and match exactly.)

- [ ] **Step 3: Register in main.go**

Next to `gachapon.InitConsumers(l)(cmf)(consumerGroupId)` (main.go:215) add `incubator.InitConsumers(l)(cmf)(consumerGroupId)`; next to the gachapon `register(...)` line (:545) add `if err := register(incubator.InitHandlers(fl)(sc)(wp)(rh)); err != nil { return nil, err }` (import `"atlas-channel/kafka/consumer/incubator"`).

- [ ] **Step 4: Saga-failure branch**

1. `kafka/message/saga/kafka.go`: add `const SagaTypeIncubatorUse = "incubator_use"` next to `SagaTypeStorageOperation` (find it with grep; if the existing comparison uses a shared-lib const instead, follow that pattern).
2. `kafka/consumer/saga/consumer.go` `handleFailedEvent` (:78-130): after the storage branch add:

```go
		if e.Body.SagaType == saga.SagaTypeIncubatorUse {
			err = session.Announce(l)(ctx)(wp)(incubatorcb.IncubatorResultWriter)(incubatorcb.NewIncubatorResult(0, 0).Encode)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce incubator failure to character [%d].", e.Body.CharacterId)
			}
			return
		}
```

- [ ] **Step 5: Build + test + commit**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./... -count=1` → clean.

```bash
git add services/atlas-channel/
git commit -m "feat(channel): incubator result consumer and failed-saga result packet"
```

---

### Task 18: Seed template wiring

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`, `template_gms_84_1.json`, `template_gms_87_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json`

**Interfaces:**
- Consumes: STATUS.md row 89 opcodes (`docs/packets/audits/STATUS.md:89`): gms_v83 `0x045`, gms_v84 `0x047`, gms_v87 `0x047`, gms_v95 `0x048`, jms_v185 `0x03F`. Registry `USE_CASH_ITEM` serverbound opcodes (verified in `docs/packets/registry/`): v87 `82` (0x52), v95 `85` (0x55), jms `71` (0x47); v83/v84 templates already carry the handler at `0x4F`.
- Produces: every supported version resolves the `IncubatorResult` writer and routes `CharacterCashItemUseHandle`.

**gms_92 is deliberately omitted**: STATUS.md has no v92 column, there is no v92 IDB, and no verifiable opcode source exists for either row. An absent writer/handler row is a safe no-op (announce fails with a log line; the inbound op stays unrouted); a guessed opcode could crash the client. Documented in `context.md` §Open items.

- [ ] **Step 1: Add the writer rows**

In each template's `socket.writers` array (gms_83's starts at line 1154), append (using each version's opcode from the table above; two-hex-digit style matches existing rows, so `0x45`, `0x47`, `0x47`, `0x48`, `0x3F`):

```json
        {
          "opCode": "0x45",
          "writer": "IncubatorResult"
        }
```

- [ ] **Step 2: Add the missing handler rows**

`CharacterCashItemUseHandle` exists only in gms_83 (line 412, `0x4F`) and gms_84 (`0x4F`). Append to `socket.handlers` in gms_87 / gms_95 / jms_185 respectively (a validator-less entry is silently dropped, so `validator` is mandatory):

```json
        {
          "opCode": "0x52",
          "validator": "LoggedInValidator",
          "handler": "CharacterCashItemUseHandle"
        }
```

(gms_95: `"0x55"`; jms_185: `"0x47"`.)

- [ ] **Step 3: Validate JSON + check for duplicates**

```bash
for f in services/atlas-configurations/seed-data/templates/template_gms_8*_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_9*_1.json \
         services/atlas-configurations/seed-data/templates/template_jms_185_1.json; do
  python3 -m json.tool "$f" > /dev/null && echo "OK $f"
  grep -c '"IncubatorResult"' "$f"
done
grep -rn '"opCode": "0x52"' services/atlas-configurations/seed-data/templates/template_gms_87_1.json
```

Expected: every file parses; exactly one `IncubatorResult` per edited template (zero in gms_92/gms_12); no pre-existing handler at the added opcodes (if one exists at the same opCode, STOP — that is a template conflict to escalate, not overwrite).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(templates): IncubatorResult writer + cash item use handler rows"
```

---

### Task 19: Packet verification campaign (markers, evidence, matrix)

**Files:**
- Modify: `libs/atlas-packet/incubator/clientbound/result_test.go` (markers)
- Create: `docs/packets/evidence/<version>/incubator.clientbound.IncubatorResult.yaml` ×5
- Modify (if needed): `tools/packet-audit/cmd/run.go` (`candidatesFromFName`)
- Regenerate: `docs/packets/audits/STATUS.md`, `status.json`

Follow `docs/packets/audits/VERIFYING_A_PACKET.md` exactly. Grounding rule: if `CWvsContext::OnIncubatorResult` does not resolve in a version's IDA export, STOP and escalate — never substitute an fname or fake an address (this includes the sub-body checks below).

- [ ] **Step 1: Verify the read orders one last time**

For each of gms_v83, gms_v84, gms_v87, gms_v95, jms_v185: locate `CWvsContext::OnIncubatorResult` in `docs/packets/ida-exports/` (or live IDA for v83 port 13342 / v95 port 13341 — `list_instances` first and match the binary name) and confirm the decode order matches Task 4's writer (2-field ≤84, 5-field ≥87). Also spot-check the serverbound sub-body encode orders for v87/v95/jms against each export's `CItemSpeakerDlg::_SendConsumeCashItemUseRequest` (registry fname) — the type-25/26/27 sub-bodies must match Task 3's codecs. Record what was checked in the commit message. Any divergence: fix the codec FIRST, then continue.

- [ ] **Step 2: Add the markers**

Above `TestIncubatorResult` add one line per version (v83/v95 addresses are design-verified; take v84/v87/jms addresses from the exports read in Step 1):

```go
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v83 ida=0xa28298
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v84 ida=0x<from export>
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v87 ida=0x<from export>
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v95 ida=0xa00380
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=jms_v185 ida=0x<from export>
```

(The `0x<from export>` placeholders MUST be replaced with real addresses in this step — a committed placeholder is a task failure.)

- [ ] **Step 3: Pin evidence**

For each version:

```bash
go run ./tools/packet-audit evidence pin --packet incubator/clientbound/IncubatorResult \
  --version gms_v83 --ida "CWvsContext::OnIncubatorResult" --category TIER1-FIXTURE
```

then edit each generated `docs/packets/evidence/<version>/incubator.clientbound.IncubatorResult.yaml` to add:

```yaml
verifies:
    - libs/atlas-packet/incubator/clientbound/result_test.go#TestIncubatorResult
```

- [ ] **Step 4: Regenerate the matrix**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Expected: STATUS.md row 89 cells promote to ✅ for all five versions. `--check` may exit 1 from pre-existing conflicts — the bar is **no new problems**; diff its output against a pre-change run. If the row does not link the new writer, add a `candidatesFromFName` case for `CWvsContext::OnIncubatorResult` → `incubator/clientbound/IncubatorResult` in `tools/packet-audit/cmd/run.go` (mirror an existing single-candidate case) and regenerate.

- [ ] **Step 5: Commit (test + evidence + matrix together)**

```bash
git add libs/atlas-packet/incubator/clientbound/result_test.go docs/packets/evidence/ docs/packets/audits/ tools/packet-audit/
git commit -m "verify(packet): INCUBATOR_RESULT byte fixtures + evidence for all versions"
```

---

### Task 20: Full verification suite + deploy runbook

**Files:**
- Create: `docs/tasks/task-128-item-tag-seal-incubator/deploy-runbook.md`

- [ ] **Step 1: Write the deploy runbook**

`deploy-runbook.md` content (live tenants do NOT pick up seed-template changes — applied at creation only):

```markdown
# task-128 deploy runbook — live tenant config patch

Seed templates only apply at tenant creation. For every EXISTING tenant:

1. PATCH the tenant's socket configuration:
   - Append to `socket.writers`: `{"opCode": "<version's opcode>", "writer": "IncubatorResult"}`
     (gms_83: 0x45, gms_84: 0x47, gms_87: 0x47, gms_95: 0x48, jms_185: 0x3F; SKIP gms_92 — no verified opcode, see context.md).
   - Where missing, append to `socket.handlers`:
     `{"opCode": "<version's opcode>", "validator": "LoggedInValidator", "handler": "CharacterCashItemUseHandle"}`
     (gms_87: 0x52, gms_95: 0x55, jms_185: 0x47; gms_83/84 already have it at 0x4F).
2. Seed the reward pool per tenant: `POST /api/tenants/{tenantId}/configurations/incubator-rewards/seed`.
3. Restart atlas-channel (writers/handlers do not hot-reload).
4. Smoke-test on a v83 tenant: tag an equip (name appears in tooltip, survives relog),
   seal an equip (lock icon + timer for timed variants), incubate an item (hatch dialog).
```

- [ ] **Step 2: Run the full verification suite**

From the worktree root, for every changed module:

```bash
for m in libs/atlas-constants libs/atlas-packet libs/atlas-saga \
         services/atlas-inventory/atlas.com/inventory \
         services/atlas-saga-orchestrator/atlas.com/saga-orchestrator \
         services/atlas-tenants/atlas.com/tenants \
         services/atlas-data/atlas.com/data \
         services/atlas-channel/atlas.com/channel \
         services/atlas-storage/atlas.com/storage \
         services/atlas-merchant/atlas.com/merchant; do
  (cd "$m" && go test -race ./... -count=1 && go vet ./... && go build ./...) || { echo "FAIL $m"; break; }
done
```

Expected: all clean. Fix anything that fails before proceeding.

- [ ] **Step 3: Bake every touched service**

```bash
docker buildx bake atlas-inventory atlas-saga-orchestrator atlas-tenants atlas-data atlas-channel atlas-storage atlas-merchant
```

Expected: all images build. A failure here usually means a missing lib `COPY` in the shared Dockerfile — fix and re-run (no new libs were added, so none is expected).

- [ ] **Step 4: Redis key guard + matrix check**

```bash
tools/redis-key-guard.sh
go run ./tools/packet-audit matrix --check
```

Expected: guard clean; matrix check introduces no new problems vs the pre-task baseline.

- [ ] **Step 5: Commit + code review gate**

```bash
git add docs/tasks/task-128-item-tag-seal-incubator/deploy-runbook.md
git commit -m "docs(task-128): live-tenant deploy runbook"
```

Then run `superpowers:requesting-code-review` (plan-adherence + backend reviewers) BEFORE any PR — per CLAUDE.md this is not skippable.

---

## Self-review notes

- Spec coverage: PRD §4.1 → Tasks 3/5/6/7/9/16; §4.2 → Tasks 3/5/7/8/9/12/16; §4.3 → Tasks 3/4/5/10/11/15/16/17; §4.4 → Tasks 2/6/13/14; §4.5 → Task 16 (+§11 cube no-op); §5 → Tasks 7/9/10/11; §6 → Tasks 6/11 (no baseline work — `assets` not in `DumpTables`, verified); §7 → all; §8 → Global Constraints + Task 20; PRD acceptance items map to Tasks 16–20.
- Known intentional deviations from design.md and open items are listed in `context.md` — read it before executing.
