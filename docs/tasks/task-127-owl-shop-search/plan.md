# Owl of Minerva (Shop Scanner) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A player can use an Owl of Minerva (5230000, or the USE-inventory owl 231xxxx) inside a Free Market map to search player-shop/hired-merchant listings in their world, see the most-searched top-10, click a result to warp to the shop on the same channel, and auto-enter it as a visitor — with all six packet surfaces byte-fixture verified on gms_v83 and gms_v95.

**Architecture:** atlas-channel gains three serverbound handlers (OwlAction, OwlWarp, ShopScannerItemUse), a new 523 arm in the cash-item-use handler, two clientbound writers (ShopScannerResult, ShopLinkResult), a `shopscanner` package (processor + tenant-scoped in-memory registry), and consumer hooks for warp-then-auto-enter. atlas-merchant's listing search gains world scoping/order/cap/owner-state columns and a new `searchcount` package (persisted per-tenant+world search counts, atomic upsert, top-10 REST). Packet codecs live in `libs/atlas-packet/merchant/{serverbound,clientbound}` plus one cash arm-tail codec. Seed templates route all versions; live-tenant patch is documented.

**Tech Stack:** Go, GORM (Postgres prod / sqlite tests), Kafka (segmentio), JSON:API (api2go), atlas-socket packet framework, packet-audit tooling, IDA (v83 port 13342, v95 port 13341).

## Global Constraints

Copied from `design.md` / PRD — every task implicitly includes these:

- All work happens in the worktree `.worktrees/task-127-owl-shop-search` on branch `task-127-owl-shop-search`. Never edit the main checkout.
- Free Market map range is `910000000 ≤ mapId ≤ 910000022` (IDA-verified in both v83 `RunShopScanner` 0xa0a2dc and v95 0x9deb50). Every owl op validates FM scope server-side.
- Search result cap is **200**, price-ordered; `descending` honored; truncation drops the far end of the chosen ordering.
- Consumption: **1 owl consumed only when the search returns ≥1 listing**; empty/failed search consumes nothing. Enforced by conditional emission of `REQUEST_ITEM_CONSUME` — no saga.
- Cross-channel warp is **not supported** (client renders warp link only for same-channel rows). Same-channel only; violations get SHOP_LINK code CLOSED.
- Every clientbound mode byte / SHOP_LINK code is config-resolved via `atlas_packet.ResolveCode(l, options, "operations", key)` from the tenant template — never hard-coded (this includes the serverbound OwlAction mode check, resolved from `readerOptions`).
- Every seed-template handler entry MUST carry `"validator": "LoggedInValidator"` — a validator-less entry is silently dropped.
- The counts table uses the tenant-safe PK pattern: uuid surrogate PK + unique index on `(tenant_id, world_id, item_id)`.
- Test setup uses the project's Builder/constructor patterns; no `*_testhelpers.go` files.
- No `// TODO`, stubs, or 501s in landed commits.
- Committed files use repo-relative paths only (no `/home/<name>/...`).
- `dwMiniRoomSN` wire field carries the **owner characterId** (Cosmic parity); OWL_WARP echoes it back.
- Channel byte on the wire is **0-based** (`channel.Id - 1`), matching `server_list_entry.go:76`.
- Verification gates before calling the branch done: `go test -race ./...`, `go vet ./...`, `go build ./...` in every changed module; `tools/redis-key-guard.sh` from the worktree root; `docker buildx bake atlas-channel atlas-merchant atlas-configurations`; `packet-audit matrix --check` clean; code review (plan-adherence + backend-guidelines) before PR.

**Opcode matrix (from design §1.1, authoritative):**

| Op | Dir | gms_83 | gms_84 | gms_87 | gms_92 | gms_95 | jms_185 |
|---|---|---|---|---|---|---|---|
| OWL_ACTION | sb | 0x42 | 0x42 | 0x45 | 0x49 | 0x48 | 0x3A |
| OWL_WARP | sb | 0x43 | 0x43 | 0x46 | 0x4A | 0x49 | 0x3B |
| USE_SHOP_SCANNER_ITEM | sb | 0x53 | 0x53 | unknown | unknown | 0x5A | unknown |
| SHOP_SCANNER_RESULT | cb | 0x46 | 0x48 | 0x48 | 0x4A | 0x49 | 0x40 |
| SHOP_LINK_RESULT | cb | 0x47 | 0x49 | 0x49 | 0x4B | 0x4A | 0x41 |

IDA anchors (design §1): v83 — `CUIShopScanner::OnCreate` 0x8a0e9a, OwlWarp sender `sub_8A4423` 0x8a4423, `CWvsContext::SendShopScannerItemUseRequest` 0xa0a25e, `CWvsContext::OnShopScannerResult` 0xa28c29, `CWvsContext::OnShopLinkResult` 0x8a4e7a. v95 — 0x848b90, 0x848e80 (`CUIShopScanResult::OnButtonClicked`), 0x9e10e0, 0xa076c0, 0x847d60.

---

### Task 1: Free Market map helper in libs/atlas-constants

**Files:**
- Modify: `libs/atlas-constants/map/constants.go`
- Modify: `libs/atlas-constants/map/model.go`
- Test: `libs/atlas-constants/map/model_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `_map.IsFreeMarketRoom(id Id) bool` and consts `FreeMarketEntranceId`/`FreeMarketRoomLastId` — used by Tasks 10, 11, 12.

DOM-21 check already done: no FM-range helper exists in `libs/atlas-constants/map` (only individual Henesys FM-entrance map consts around `constants.go:55`); the package is `package _map`, `type Id uint32`, and its only method today is `IsSentinel()` (`model.go:41`).

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-constants/map/model_test.go`:

```go
func TestIsFreeMarketRoom(t *testing.T) {
	cases := []struct {
		name string
		id   Id
		want bool
	}{
		{"below range", Id(909999999), false},
		{"FM entrance", Id(910000000), true},
		{"FM room 1", Id(910000001), true},
		{"FM room 22 (last)", Id(910000022), true},
		{"above range", Id(910000023), false},
		{"henesys (unrelated)", Id(100000000), false},
		{"zero", Id(0), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsFreeMarketRoom(c.id); got != c.want {
				t.Errorf("IsFreeMarketRoom(%d) = %v, want %v", c.id, got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./map/ -run TestIsFreeMarketRoom -v`
Expected: FAIL — `undefined: IsFreeMarketRoom`

- [ ] **Step 3: Write minimal implementation**

In `libs/atlas-constants/map/constants.go`, add to the existing const block (near the other FM-related ids around line 55):

```go
	// Free Market interior maps (entrance + rooms 1-22). Range verified against
	// RunShopScanner in GMS v83 (0xa0a2dc) and v95 (0x9deb50) — the client
	// hard-blocks the shop scanner outside this range (task-127).
	FreeMarketEntranceId = Id(910000000)
	FreeMarketRoomLastId = Id(910000022)
```

In `libs/atlas-constants/map/model.go`, add below `IsSentinel`:

```go
// IsFreeMarketRoom reports whether id is inside the Free Market
// (entrance 910000000 through room 22 at 910000022).
func IsFreeMarketRoom(id Id) bool {
	return id >= FreeMarketEntranceId && id <= FreeMarketRoomLastId
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-constants && go test -race ./map/ -v && go vet ./map/`
Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-constants/map/constants.go libs/atlas-constants/map/model.go libs/atlas-constants/map/model_test.go
git commit -m "feat(task-127): add Free Market room range helper to atlas-constants/map"
```

---

### Task 2: Serverbound owl codecs in libs/atlas-packet

**Files:**
- Create: `libs/atlas-packet/merchant/serverbound/owl_action.go`
- Create: `libs/atlas-packet/merchant/serverbound/owl_warp.go`
- Create: `libs/atlas-packet/merchant/serverbound/shop_scanner_item_use.go`
- Create: `libs/atlas-packet/cash/serverbound/item_use_store_search.go`
- Test: `libs/atlas-packet/merchant/serverbound/owl_action_test.go`
- Test: `libs/atlas-packet/merchant/serverbound/owl_warp_test.go`
- Test: `libs/atlas-packet/merchant/serverbound/shop_scanner_item_use_test.go`
- Test: `libs/atlas-packet/cash/serverbound/item_use_store_search_test.go`

**Interfaces:**
- Consumes: `response.Writer` / `request.Reader` from atlas-socket (`WriteByte/WriteInt/WriteInt16/WriteBool`, `ReadByte/ReadUint32/ReadInt16/ReadBool`).
- Produces (used by Tasks 12, 14, 15):
  - `const OwlActionHandle = "OwlActionHandle"`; `type OwlAction` with `Mode() byte`
  - `const OwlWarpHandle = "OwlWarpHandle"`; `type OwlWarp` with `OwnerId() uint32`, `MapId() uint32`
  - `const ShopScannerItemUseHandle = "ShopScannerItemUseHandle"`; `type ShopScannerItemUse` with `Source() int16`, `ItemId() uint32`, `SearchItemId() uint32`, `Descending() bool`, `UpdateTime() uint32`
  - `cashsb.NewItemUseStoreSearch() *ItemUseStoreSearch` with `SearchItemId() uint32`, `Descending() bool`, `UpdateTime() uint32`

Wire layouts (design §1.2/§1.3/§1.5, IDA-verified): OWL_ACTION = `[byte mode]` (client only ever sends 5). OWL_WARP = `[int dwMiniRoomSN(=ownerId)][int dwFieldID]`. USE_SHOP_SCANNER_ITEM = `[short nPOS][int nItemID][int searchItemId][byte bDescendingOrder][int updateTime]` — **no** leading updateTime even on v95 (verified 0x9e10e0). The cash 523 arm tail = `[int searchItemId][byte bDescendingOrder][int updateTime]` appended by `CUIShopScanner::SendScanPacket` unconditionally in both v83 and v95 (the GMS≥95 leading-updateTime gate applies only to the `ItemUse` prefix, which the existing codec already handles).

- [ ] **Step 1: Write the failing tests**

`libs/atlas-packet/merchant/serverbound/owl_action_test.go`:

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=merchant/serverbound/OwlAction version=gms_v83 ida=0x8a0e9a
// packet-audit:verify packet=merchant/serverbound/OwlAction version=gms_v95 ida=0x848b90
func TestOwlActionRoundTrip(t *testing.T) {
	input := NewOwlAction(5)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &OwlAction{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != 5 {
				t.Errorf("mode = %d, want 5", output.Mode())
			}
		})
	}
}

// TestOwlActionWireShape pins the exact layout: a single mode byte.
// CUIShopScanner::OnCreate builds [opcode][byte 5] (v83 0x8a0e9a, v95 0x848b90).
func TestOwlActionWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewOwlAction(5)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			if len(b) != 1 {
				t.Fatalf("wire size = %d bytes, want 1: % x", len(b), b)
			}
			if b[0] != 0x05 {
				t.Errorf("byte[0] mode = 0x%02x, want 0x05", b[0])
			}
		})
	}
}
```

`libs/atlas-packet/merchant/serverbound/owl_warp_test.go`:

```go
package serverbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=merchant/serverbound/OwlWarp version=gms_v83 ida=0x8a4423
// packet-audit:verify packet=merchant/serverbound/OwlWarp version=gms_v95 ida=0x848e80
func TestOwlWarpRoundTrip(t *testing.T) {
	input := NewOwlWarp(30001, 910000005)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &OwlWarp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.OwnerId() != 30001 {
				t.Errorf("ownerId = %d, want 30001", output.OwnerId())
			}
			if output.MapId() != 910000005 {
				t.Errorf("mapId = %d, want 910000005", output.MapId())
			}
		})
	}
}

// TestOwlWarpWireShape pins [int dwMiniRoomSN][int dwFieldID] — the client
// echoes both ints from the clicked record verbatim (v83 sub_8A4423, v95
// CUIShopScanResult::OnButtonClicked 0x848e80).
func TestOwlWarpWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewOwlWarp(30001, 910000005)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			if len(b) != 8 {
				t.Fatalf("wire size = %d bytes, want 8: % x", len(b), b)
			}
			if binary.LittleEndian.Uint32(b[0:4]) != 30001 {
				t.Errorf("dwMiniRoomSN = %d, want 30001", binary.LittleEndian.Uint32(b[0:4]))
			}
			if binary.LittleEndian.Uint32(b[4:8]) != 910000005 {
				t.Errorf("dwFieldID = %d, want 910000005", binary.LittleEndian.Uint32(b[4:8]))
			}
		})
	}
}
```

`libs/atlas-packet/merchant/serverbound/shop_scanner_item_use_test.go`:

```go
package serverbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=merchant/serverbound/ShopScannerItemUse version=gms_v83 ida=0xa0a25e
// packet-audit:verify packet=merchant/serverbound/ShopScannerItemUse version=gms_v95 ida=0x9e10e0
func TestShopScannerItemUseRoundTrip(t *testing.T) {
	input := NewShopScannerItemUse(3, 2310000, 1302000, true, 12345678)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &ShopScannerItemUse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Source() != 3 {
				t.Errorf("source = %d, want 3", output.Source())
			}
			if output.ItemId() != 2310000 {
				t.Errorf("itemId = %d, want 2310000", output.ItemId())
			}
			if output.SearchItemId() != 1302000 {
				t.Errorf("searchItemId = %d, want 1302000", output.SearchItemId())
			}
			if !output.Descending() {
				t.Errorf("descending = false, want true")
			}
			if output.UpdateTime() != 12345678 {
				t.Errorf("updateTime = %d, want 12345678", output.UpdateTime())
			}
		})
	}
}

// TestShopScannerItemUseWireShape pins
// [short nPOS][int nItemID][int searchItemId][byte bDescending][int updateTime]
// — NO leading updateTime on any version, v95 included (verified 0x9e10e0).
func TestShopScannerItemUseWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewShopScannerItemUse(3, 2310000, 1302000, false, 12345678)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			if len(b) != 15 {
				t.Fatalf("wire size = %d bytes, want 15: % x", len(b), b)
			}
			if binary.LittleEndian.Uint16(b[0:2]) != 3 {
				t.Errorf("nPOS = %d, want 3", binary.LittleEndian.Uint16(b[0:2]))
			}
			if binary.LittleEndian.Uint32(b[2:6]) != 2310000 {
				t.Errorf("nItemID = %d, want 2310000", binary.LittleEndian.Uint32(b[2:6]))
			}
			if binary.LittleEndian.Uint32(b[6:10]) != 1302000 {
				t.Errorf("searchItemId = %d, want 1302000", binary.LittleEndian.Uint32(b[6:10]))
			}
			if b[10] != 0x00 {
				t.Errorf("bDescending = 0x%02x, want 0x00", b[10])
			}
			if binary.LittleEndian.Uint32(b[11:15]) != 12345678 {
				t.Errorf("updateTime = %d, want 12345678", binary.LittleEndian.Uint32(b[11:15]))
			}
		})
	}
}
```

`libs/atlas-packet/cash/serverbound/item_use_store_search_test.go`:

```go
package serverbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// The 523 arm tail of CWvsContext::SendConsumeCashItemUseRequest case 29
// (v83 jumptable case 0xa0cd0b): CUIShopScanner::SendScanPacket appends
// [int searchItemId][byte bDescending][int updateTime] to the stashed use
// packet unconditionally in both v83 and v95 — the GMS>=95 leading-updateTime
// gate applies only to the ItemUse prefix, not this tail.
func TestItemUseStoreSearchRoundTrip(t *testing.T) {
	input := NewItemUseStoreSearch()
	input.SetForTest(1302000, true, 12345678)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := NewItemUseStoreSearch()
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SearchItemId() != 1302000 {
				t.Errorf("searchItemId = %d, want 1302000", output.SearchItemId())
			}
			if !output.Descending() {
				t.Errorf("descending = false, want true")
			}
			if output.UpdateTime() != 12345678 {
				t.Errorf("updateTime = %d, want 12345678", output.UpdateTime())
			}
		})
	}
}

func TestItemUseStoreSearchWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewItemUseStoreSearch()
	in.SetForTest(1302000, false, 12345678)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			if len(b) != 9 {
				t.Fatalf("wire size = %d bytes, want 9: % x", len(b), b)
			}
			if binary.LittleEndian.Uint32(b[0:4]) != 1302000 {
				t.Errorf("searchItemId = %d, want 1302000", binary.LittleEndian.Uint32(b[0:4]))
			}
			if b[4] != 0x00 {
				t.Errorf("bDescending = 0x%02x, want 0x00", b[4])
			}
			if binary.LittleEndian.Uint32(b[5:9]) != 12345678 {
				t.Errorf("updateTime = %d, want 12345678", binary.LittleEndian.Uint32(b[5:9]))
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./merchant/serverbound/ ./cash/serverbound/ -run 'OwlAction|OwlWarp|ShopScannerItemUse|ItemUseStoreSearch' -v`
Expected: FAIL — `undefined: NewOwlAction`, `undefined: NewOwlWarp`, `undefined: NewShopScannerItemUse`, `undefined: NewItemUseStoreSearch`

- [ ] **Step 3: Write the codecs**

`libs/atlas-packet/merchant/serverbound/owl_action.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const OwlActionHandle = "OwlActionHandle"

// packet-audit:fname CUIShopScanner::OnCreate
// OwlAction is sent once when the shop-scanner UI opens (mode 5) to request
// the most-searched hot list. A full construction-site scan of every
// COutPacket(0x42) (v83) / COutPacket(0x48) (v95) found exactly one sender:
// CUIShopScanner::OnCreate with mode 5 (task-127 design §1.3).
type OwlAction struct {
	mode byte
}

func NewOwlAction(mode byte) OwlAction {
	return OwlAction{mode: mode}
}

func (m OwlAction) Mode() byte {
	return m.mode
}

func (m OwlAction) Operation() string {
	return OwlActionHandle
}

func (m OwlAction) String() string {
	return fmt.Sprintf("mode [%d]", m.mode)
}

func (m OwlAction) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *OwlAction) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
```

`libs/atlas-packet/merchant/serverbound/owl_warp.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const OwlWarpHandle = "OwlWarpHandle"

// packet-audit:fname CUIShopScanResult::OnButtonClicked
// OwlWarp is sent when the player clicks a shop-scanner result row. The two
// ints echo the record's dwMiniRoomSN (Atlas sends the shop-owner characterId
// there) and dwFieldID verbatim (v83 sub_8A4423, v95 0x848e80).
type OwlWarp struct {
	ownerId uint32
	mapId   uint32
}

func NewOwlWarp(ownerId uint32, mapId uint32) OwlWarp {
	return OwlWarp{ownerId: ownerId, mapId: mapId}
}

func (m OwlWarp) OwnerId() uint32 {
	return m.ownerId
}

func (m OwlWarp) MapId() uint32 {
	return m.mapId
}

func (m OwlWarp) Operation() string {
	return OwlWarpHandle
}

func (m OwlWarp) String() string {
	return fmt.Sprintf("ownerId [%d] mapId [%d]", m.ownerId, m.mapId)
}

func (m OwlWarp) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		w.WriteInt(m.mapId)
		return w.Bytes()
	}
}

func (m *OwlWarp) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.mapId = r.ReadUint32()
	}
}
```

`libs/atlas-packet/merchant/serverbound/shop_scanner_item_use.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ShopScannerItemUseHandle = "ShopScannerItemUseHandle"

// packet-audit:fname CWvsContext::SendShopScannerItemUseRequest
// ShopScannerItemUse is the dedicated use-route for the USE-inventory owl
// (231xxxx family), double-clicked from the inventory. Gated client-side on
// itemId/10000 == 231 (v95 is_shopscanner_item 0x4ff5c0). No leading
// updateTime on any version, v95 included (verified 0x9e10e0; v83 0xa0a25e).
type ShopScannerItemUse struct {
	source       int16
	itemId       uint32
	searchItemId uint32
	descending   bool
	updateTime   uint32
}

func NewShopScannerItemUse(source int16, itemId uint32, searchItemId uint32, descending bool, updateTime uint32) ShopScannerItemUse {
	return ShopScannerItemUse{source: source, itemId: itemId, searchItemId: searchItemId, descending: descending, updateTime: updateTime}
}

func (m ShopScannerItemUse) Source() int16 {
	return m.source
}

func (m ShopScannerItemUse) ItemId() uint32 {
	return m.itemId
}

func (m ShopScannerItemUse) SearchItemId() uint32 {
	return m.searchItemId
}

func (m ShopScannerItemUse) Descending() bool {
	return m.descending
}

func (m ShopScannerItemUse) UpdateTime() uint32 {
	return m.updateTime
}

func (m ShopScannerItemUse) Operation() string {
	return ShopScannerItemUseHandle
}

func (m ShopScannerItemUse) String() string {
	return fmt.Sprintf("source [%d] itemId [%d] searchItemId [%d] descending [%t] updateTime [%d]", m.source, m.itemId, m.searchItemId, m.descending, m.updateTime)
}

func (m ShopScannerItemUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		w.WriteInt(m.searchItemId)
		w.WriteBool(m.descending)
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *ShopScannerItemUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
		m.searchItemId = r.ReadUint32()
		m.descending = r.ReadBool()
		m.updateTime = r.ReadUint32()
	}
}
```

`libs/atlas-packet/cash/serverbound/item_use_store_search.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseStoreSearch is the arm tail for USE_CASH_ITEM itemType 523 (Owl of
// Minerva, cash-slot type 29). CUIShopScanner::SendScanPacket appends
// [int searchItemId][byte bDescendingOrder][int updateTime] to the stashed
// use packet unconditionally in both v83 (sub_8A2407) and v95 (0x83f6b0);
// the GMS>=95 leading-updateTime gate lives in the ItemUse prefix codec.
type ItemUseStoreSearch struct {
	searchItemId uint32
	descending   bool
	updateTime   uint32
}

func NewItemUseStoreSearch() *ItemUseStoreSearch {
	return &ItemUseStoreSearch{}
}

// SetForTest populates the codec for encode-side tests.
func (m *ItemUseStoreSearch) SetForTest(searchItemId uint32, descending bool, updateTime uint32) {
	m.searchItemId = searchItemId
	m.descending = descending
	m.updateTime = updateTime
}

func (m ItemUseStoreSearch) SearchItemId() uint32 {
	return m.searchItemId
}

func (m ItemUseStoreSearch) Descending() bool {
	return m.descending
}

func (m ItemUseStoreSearch) UpdateTime() uint32 {
	return m.updateTime
}

func (m ItemUseStoreSearch) Operation() string {
	return "ItemUseStoreSearch"
}

func (m ItemUseStoreSearch) String() string {
	return fmt.Sprintf("searchItemId [%d] descending [%t] updateTime [%d]", m.searchItemId, m.descending, m.updateTime)
}

func (m ItemUseStoreSearch) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.searchItemId)
		w.WriteBool(m.descending)
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *ItemUseStoreSearch) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.searchItemId = r.ReadUint32()
		m.descending = r.ReadBool()
		m.updateTime = r.ReadUint32()
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test -race ./merchant/serverbound/ ./cash/serverbound/ -v && go vet ./merchant/... ./cash/...`
Expected: all new tests PASS (existing tests untouched), vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/merchant/serverbound/ libs/atlas-packet/cash/serverbound/item_use_store_search.go libs/atlas-packet/cash/serverbound/item_use_store_search_test.go
git commit -m "feat(task-127): serverbound owl/shop-scanner codecs (OwlAction, OwlWarp, ShopScannerItemUse, ItemUseStoreSearch)"
```

---

### Task 3: Clientbound shop-scanner codecs + body factories in libs/atlas-packet

**Files:**
- Create: `libs/atlas-packet/merchant/clientbound/shop_scanner_result.go`
- Create: `libs/atlas-packet/merchant/clientbound/shop_link_result.go`
- Create: `libs/atlas-packet/merchant/shop_scanner_body.go`
- Test: `libs/atlas-packet/merchant/clientbound/shop_scanner_result_test.go`
- Test: `libs/atlas-packet/merchant/clientbound/shop_link_result_test.go`

**Interfaces:**
- Consumes: `model.Asset` (`pktmodel.NewAsset(zeroPosition bool, slot int16, templateId uint32, expiration time.Time)`, `.SetEquipmentStats(15×uint16)`, `.SetEquipmentMeta(slots uint16, levelType, level byte, experience, hammersApplied uint32, flag uint16)`, `.Encode`, `.Decode`); `atlas_packet.WithResolvedCode(codeProperty, key string, factory func(byte) packet.Encoder)`.
- Produces (used by Tasks 11, 13, 14, 15):
  - `const ShopScannerResultWriter = "ShopScannerResult"` and `const ShopLinkResultWriter = "ShopLinkResult"`
  - `clientbound.NewShopScannerRecord(ownerName string, mapId uint32, title string, bundles uint32, bundleSize uint32, price uint32, ownerId uint32, channelId byte, inventoryType byte, asset *model.Asset) ShopScannerRecord`
  - `clientbound.NewShopScannerResult(mode byte, itemId uint32, records []ShopScannerRecord) ShopScannerResult`
  - `clientbound.NewShopScannerHotList(mode byte, itemIds []uint32) ShopScannerHotList`
  - `clientbound.NewShopLinkResult(code byte) ShopLinkResult`
  - Package-level factories: `merchant.ShopScannerResultBody(itemId uint32, records []clientbound.ShopScannerRecord)`, `merchant.ShopScannerHotListBody(itemIds []uint32)`, `merchant.ShopLinkResultBody(code ShopLinkResultCode)` (each returns `func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`)
  - Mode/code name constants: `ShopScannerResultModeResult = "RESULT"`, `ShopScannerResultModeHotList = "HOT_LIST"`, and `ShopLinkResultCode` constants `SUCCESS/CLOSED/FULL/BUSY/DEAD/NO_TRADE/DENIED/MAINTENANCE/FM_ONLY`

Wire layout (design §1.4, v83 0xa28c29 / v95 0xa076c0, identical structure; Cosmic `PacketCreator.owlOfMinerva` corroborates field-for-field):

```
mode 6 (RESULT):  [byte 6][int nNpcShopPrice=0][int nItemID][int nCount]
                  nCount × { [str sCharacterName][int dwFieldID][str sTitle]
                             [int nNumber(bundles)][int nSet(qty/bundle)][int nPrice]
                             [int dwMiniRoomSN(=ownerId)][byte nChannelID(0-based)]
                             [byte nTI][if nTI==1: GW_ItemSlotBase] }
mode 7 (HOT_LIST): [byte 7][byte count][count × int itemId]
SHOP_LINK_RESULT:  [byte code]
```

- [ ] **Step 1: Write the failing tests**

`libs/atlas-packet/merchant/clientbound/shop_scanner_result_test.go`:

```go
package clientbound

import (
	"encoding/binary"
	"testing"
	"time"

	pktmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=merchant/clientbound/ShopScannerResult version=gms_v83 ida=0xa28c29
// packet-audit:verify packet=merchant/clientbound/ShopScannerResult version=gms_v95 ida=0xa076c0
// packet-audit:verify packet=merchant/clientbound/ShopScannerHotList version=gms_v83 ida=0xa28c29
// packet-audit:verify packet=merchant/clientbound/ShopScannerHotList version=gms_v95 ida=0xa076c0
func TestShopScannerResultRoundTrip(t *testing.T) {
	records := []ShopScannerRecord{
		NewShopScannerRecord("OwnerA", 910000004, "cheap stuff", 3, 100, 5000, 30001, 0, 2, nil),
		NewShopScannerRecord("OwnerB", 910000010, "arrows", 1, 1000, 9000, 30002, 1, 2, nil),
	}
	input := NewShopScannerResult(6, 2060000, records)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &ShopScannerResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != 6 {
				t.Errorf("mode = %d, want 6", output.Mode())
			}
			if output.ItemId() != 2060000 {
				t.Errorf("itemId = %d, want 2060000", output.ItemId())
			}
			if len(output.Records()) != 2 {
				t.Fatalf("record count = %d, want 2", len(output.Records()))
			}
			r0 := output.Records()[0]
			if r0.OwnerName() != "OwnerA" || r0.MapId() != 910000004 || r0.Title() != "cheap stuff" ||
				r0.Bundles() != 3 || r0.BundleSize() != 100 || r0.Price() != 5000 ||
				r0.OwnerId() != 30001 || r0.ChannelId() != 0 || r0.InventoryType() != 2 {
				t.Errorf("record 0 mismatch: %+v", r0)
			}
		})
	}
}

// TestShopScannerResultEmpty pins the faithful no-results shape:
// nCount==0 && nNpcShopPrice==0 makes the client show SP_3637
// ("Unable to find the item you have entered").
func TestShopScannerResultEmpty(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewShopScannerResult(6, 2060000, nil)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// [byte mode][int npcShopPrice][int itemId][int count] = 13 bytes
			if len(b) != 13 {
				t.Fatalf("wire size = %d bytes, want 13: % x", len(b), b)
			}
			if b[0] != 0x06 {
				t.Errorf("mode = 0x%02x, want 0x06", b[0])
			}
			if binary.LittleEndian.Uint32(b[1:5]) != 0 {
				t.Errorf("nNpcShopPrice = %d, want 0", binary.LittleEndian.Uint32(b[1:5]))
			}
			if binary.LittleEndian.Uint32(b[5:9]) != 2060000 {
				t.Errorf("nItemID = %d, want 2060000", binary.LittleEndian.Uint32(b[5:9]))
			}
			if binary.LittleEndian.Uint32(b[9:13]) != 0 {
				t.Errorf("nCount = %d, want 0", binary.LittleEndian.Uint32(b[9:13]))
			}
		})
	}
}

// TestShopScannerResultEquipRow exercises the nTI==1 branch: a full
// GW_ItemSlotBase (slotless, zeroPosition=true) follows the record header.
func TestShopScannerResultEquipRow(t *testing.T) {
	asset := pktmodel.NewAsset(true, 0, 1302000, time.Time{}).
		SetEquipmentStats(5, 3, 0, 0, 0, 0, 17, 0, 0, 0, 0, 0, 0, 0, 0).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0)
	records := []ShopScannerRecord{
		NewShopScannerRecord("OwnerA", 910000004, "swords", 1, 1, 150000, 30001, 0, 1, &asset),
	}
	input := NewShopScannerResult(6, 1302000, records)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &ShopScannerResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Records()) != 1 {
				t.Fatalf("record count = %d, want 1", len(output.Records()))
			}
			r0 := output.Records()[0]
			if r0.InventoryType() != 1 {
				t.Fatalf("inventoryType = %d, want 1", r0.InventoryType())
			}
			if r0.Asset() == nil {
				t.Fatalf("asset = nil, want decoded GW_ItemSlotBase")
			}
		})
	}
}

func TestShopScannerHotListRoundTrip(t *testing.T) {
	input := NewShopScannerHotList(7, []uint32{2060000, 1302000, 4000000})
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &ShopScannerHotList{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != 7 {
				t.Errorf("mode = %d, want 7", output.Mode())
			}
			if len(output.ItemIds()) != 3 || output.ItemIds()[0] != 2060000 || output.ItemIds()[2] != 4000000 {
				t.Errorf("itemIds = %v, want [2060000 1302000 4000000]", output.ItemIds())
			}
		})
	}
}

// TestShopScannerHotListShortCount: fewer than 10 ever-searched items sends a
// short list — count byte reflects the actual length, no filler.
func TestShopScannerHotListShortCount(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewShopScannerHotList(7, []uint32{2060000})
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// [byte mode][byte count][1 × int] = 6 bytes
			if len(b) != 6 {
				t.Fatalf("wire size = %d bytes, want 6: % x", len(b), b)
			}
			if b[0] != 0x07 {
				t.Errorf("mode = 0x%02x, want 0x07", b[0])
			}
			if b[1] != 0x01 {
				t.Errorf("count = 0x%02x, want 0x01", b[1])
			}
			if binary.LittleEndian.Uint32(b[2:6]) != 2060000 {
				t.Errorf("itemId = %d, want 2060000", binary.LittleEndian.Uint32(b[2:6]))
			}
		})
	}
}
```

`libs/atlas-packet/merchant/clientbound/shop_link_result_test.go`:

```go
package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=merchant/clientbound/ShopLinkResult version=gms_v83 ida=0x8a4e7a
// packet-audit:verify packet=merchant/clientbound/ShopLinkResult version=gms_v95 ida=0x847d60
func TestShopLinkResultRoundTrip(t *testing.T) {
	input := NewShopLinkResult(18)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &ShopLinkResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != 18 {
				t.Errorf("code = %d, want 18", output.Code())
			}
		})
	}
}

// TestShopLinkResultWireShape pins the single-code-byte body. Code set is
// identical in v83 (0x8a4e7a) and v95 (0x847d60) — task-127 design §1.5.
func TestShopLinkResultWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, code := range []byte{0, 1, 2, 3, 4, 7, 17, 18, 23} {
		in := NewShopLinkResult(code)
		b := in.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
		if len(b) != 1 || b[0] != code {
			t.Errorf("code %d: wire = % x, want single byte 0x%02x", code, b, code)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./merchant/clientbound/ -run 'ShopScanner|ShopLink' -v`
Expected: FAIL — `undefined: NewShopScannerRecord`, `undefined: NewShopScannerResult`, `undefined: NewShopScannerHotList`, `undefined: NewShopLinkResult`

- [ ] **Step 3: Write the codecs**

`libs/atlas-packet/merchant/clientbound/shop_scanner_result.go`:

```go
package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ShopScannerResultWriter = "ShopScannerResult"

// ShopScannerRecord is one row of the shop-scanner result list
// (CWvsContext::OnShopScannerResult mode 6, ITEMDATA in the v95 typed struct).
// dwMiniRoomSN carries the shop-owner characterId (Cosmic parity, task-127
// design §4.4); channelId is 0-based on the wire; asset must be a
// zeroPosition (slotless) model.Asset and is encoded only when
// inventoryType == 1 (equip).
type ShopScannerRecord struct {
	ownerName     string
	mapId         uint32
	title         string
	bundles       uint32
	bundleSize    uint32
	price         uint32
	ownerId       uint32
	channelId     byte
	inventoryType byte
	asset         *model.Asset
}

func NewShopScannerRecord(ownerName string, mapId uint32, title string, bundles uint32, bundleSize uint32, price uint32, ownerId uint32, channelId byte, inventoryType byte, asset *model.Asset) ShopScannerRecord {
	return ShopScannerRecord{
		ownerName:     ownerName,
		mapId:         mapId,
		title:         title,
		bundles:       bundles,
		bundleSize:    bundleSize,
		price:         price,
		ownerId:       ownerId,
		channelId:     channelId,
		inventoryType: inventoryType,
		asset:         asset,
	}
}

func (r ShopScannerRecord) OwnerName() string   { return r.ownerName }
func (r ShopScannerRecord) MapId() uint32       { return r.mapId }
func (r ShopScannerRecord) Title() string       { return r.title }
func (r ShopScannerRecord) Bundles() uint32     { return r.bundles }
func (r ShopScannerRecord) BundleSize() uint32  { return r.bundleSize }
func (r ShopScannerRecord) Price() uint32       { return r.price }
func (r ShopScannerRecord) OwnerId() uint32     { return r.ownerId }
func (r ShopScannerRecord) ChannelId() byte     { return r.channelId }
func (r ShopScannerRecord) InventoryType() byte { return r.inventoryType }
func (r ShopScannerRecord) Asset() *model.Asset { return r.asset }

// packet-audit:fname CWvsContext::OnShopScannerResult#Result
// ShopScannerResult is mode 6 of CWvsContext::OnShopScannerResult (v83
// 0xa28c29, v95 0xa076c0). nNpcShopPrice > 0 makes the client insert a
// synthetic "sold in regular stores" first row — Atlas always sends 0.
// nCount==0 && nNpcShopPrice==0 shows the faithful no-results message.
type ShopScannerResult struct {
	mode         byte
	npcShopPrice uint32
	itemId       uint32
	records      []ShopScannerRecord
}

func NewShopScannerResult(mode byte, itemId uint32, records []ShopScannerRecord) ShopScannerResult {
	return ShopScannerResult{mode: mode, npcShopPrice: 0, itemId: itemId, records: records}
}

func (m ShopScannerResult) Mode() byte                   { return m.mode }
func (m ShopScannerResult) ItemId() uint32               { return m.itemId }
func (m ShopScannerResult) Records() []ShopScannerRecord { return m.records }

func (m ShopScannerResult) Operation() string {
	return ShopScannerResultWriter
}

func (m ShopScannerResult) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.npcShopPrice)
		w.WriteInt(m.itemId)
		w.WriteInt(uint32(len(m.records)))
		for _, rec := range m.records {
			w.WriteAsciiString(rec.ownerName)
			w.WriteInt(rec.mapId)
			w.WriteAsciiString(rec.title)
			w.WriteInt(rec.bundles)
			w.WriteInt(rec.bundleSize)
			w.WriteInt(rec.price)
			w.WriteInt(rec.ownerId)
			w.WriteByte(rec.channelId)
			w.WriteByte(rec.inventoryType)
			if rec.inventoryType == 1 && rec.asset != nil {
				w.WriteByteArray(rec.asset.Encode(l, ctx)(options))
			}
		}
		return w.Bytes()
	}
}

func (m *ShopScannerResult) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.npcShopPrice = r.ReadUint32()
		m.itemId = r.ReadUint32()
		count := r.ReadUint32()
		m.records = make([]ShopScannerRecord, 0, count)
		for i := uint32(0); i < count; i++ {
			var rec ShopScannerRecord
			rec.ownerName = r.ReadAsciiString()
			rec.mapId = r.ReadUint32()
			rec.title = r.ReadAsciiString()
			rec.bundles = r.ReadUint32()
			rec.bundleSize = r.ReadUint32()
			rec.price = r.ReadUint32()
			rec.ownerId = r.ReadUint32()
			rec.channelId = r.ReadByte()
			rec.inventoryType = r.ReadByte()
			if rec.inventoryType == 1 {
				a := &model.Asset{}
				a.Decode(l, ctx)(r, options)
				rec.asset = a
			}
			m.records = append(m.records, rec)
		}
	}
}

// packet-audit:fname CWvsContext::OnShopScannerResult#HotList
// ShopScannerHotList is mode 7: the most-searched item list shown when the
// scanner UI opens. Short lists (fewer than 10 ever-searched items) send the
// actual count — no filler.
type ShopScannerHotList struct {
	mode    byte
	itemIds []uint32
}

func NewShopScannerHotList(mode byte, itemIds []uint32) ShopScannerHotList {
	return ShopScannerHotList{mode: mode, itemIds: itemIds}
}

func (m ShopScannerHotList) Mode() byte        { return m.mode }
func (m ShopScannerHotList) ItemIds() []uint32 { return m.itemIds }

func (m ShopScannerHotList) Operation() string {
	return ShopScannerResultWriter
}

func (m ShopScannerHotList) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(byte(len(m.itemIds)))
		for _, id := range m.itemIds {
			w.WriteInt(id)
		}
		return w.Bytes()
	}
}

func (m *ShopScannerHotList) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := r.ReadByte()
		m.itemIds = make([]uint32, 0, count)
		for i := byte(0); i < count; i++ {
			m.itemIds = append(m.itemIds, r.ReadUint32())
		}
	}
}
```

`libs/atlas-packet/merchant/clientbound/shop_link_result.go`:

```go
package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ShopLinkResultWriter = "ShopLinkResult"

// packet-audit:fname CWvsContext::OnShopLinkResult
// ShopLinkResult carries the owl-warp/enter outcome as a single code byte.
// Code set identical in v83 (0x8a4e7a) and v95 (0x847d60): 0 success,
// 1 closed, 2 full, 3 busy, 4 dead, 7 no-trade, 17 denied, 18 maintenance,
// 23 FM-only; anything else = "This character is unable to do it".
type ShopLinkResult struct {
	code byte
}

func NewShopLinkResult(code byte) ShopLinkResult {
	return ShopLinkResult{code: code}
}

func (m ShopLinkResult) Code() byte {
	return m.code
}

func (m ShopLinkResult) Operation() string {
	return ShopLinkResultWriter
}

func (m ShopLinkResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.code)
		return w.Bytes()
	}
}

func (m *ShopLinkResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.code = r.ReadByte()
	}
}
```

`libs/atlas-packet/merchant/shop_scanner_body.go` (mirrors `operation_body.go`; check that file's package clause and imports and match them exactly):

```go
package merchant

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// ShopScannerResult CWvsContext::OnShopScannerResult
type ShopScannerResultMode = string

const (
	ShopScannerResultModeResult  = ShopScannerResultMode("RESULT")
	ShopScannerResultModeHotList = ShopScannerResultMode("HOT_LIST")
)

// ShopLinkResult CWvsContext::OnShopLinkResult
type ShopLinkResultCode = string

const (
	ShopLinkResultCodeSuccess     = ShopLinkResultCode("SUCCESS")
	ShopLinkResultCodeClosed      = ShopLinkResultCode("CLOSED")
	ShopLinkResultCodeFull        = ShopLinkResultCode("FULL")
	ShopLinkResultCodeBusy        = ShopLinkResultCode("BUSY")
	ShopLinkResultCodeDead        = ShopLinkResultCode("DEAD")
	ShopLinkResultCodeNoTrade     = ShopLinkResultCode("NO_TRADE")
	ShopLinkResultCodeDenied      = ShopLinkResultCode("DENIED")
	ShopLinkResultCodeMaintenance = ShopLinkResultCode("MAINTENANCE")
	ShopLinkResultCodeFMOnly      = ShopLinkResultCode("FM_ONLY")
)

func ShopScannerResultBody(itemId uint32, records []clientbound.ShopScannerRecord) func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ShopScannerResultModeResult, func(mode byte) packet.Encoder {
		return clientbound.NewShopScannerResult(mode, itemId, records)
	})
}

func ShopScannerHotListBody(itemIds []uint32) func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ShopScannerResultModeHotList, func(mode byte) packet.Encoder {
		return clientbound.NewShopScannerHotList(mode, itemIds)
	})
}

func ShopLinkResultBody(code ShopLinkResultCode) func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", code, func(c byte) packet.Encoder {
		return clientbound.NewShopLinkResult(c)
	})
}
```

Note: if `WithResolvedCode`'s exact return type in `libs/atlas-packet/resolve.go:13` differs from the signature above, match the existing `operation_body.go` factory signatures verbatim.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test -race ./merchant/... -v && go vet ./merchant/...`
Expected: all new tests PASS, existing merchant tests still PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/merchant/clientbound/shop_scanner_result.go libs/atlas-packet/merchant/clientbound/shop_scanner_result_test.go libs/atlas-packet/merchant/clientbound/shop_link_result.go libs/atlas-packet/merchant/clientbound/shop_link_result_test.go libs/atlas-packet/merchant/shop_scanner_body.go
git commit -m "feat(task-127): clientbound ShopScannerResult/HotList/ShopLinkResult codecs and body factories"
```

---

### Task 4: atlas-merchant — world-scoped, ordered, capped, enriched listing search

**Files:**
- Modify: `services/atlas-merchant/atlas.com/merchant/shop/provider.go:87-125` (`listingSearchRow`, `searchListingsByItemId`)
- Modify: `services/atlas-merchant/atlas.com/merchant/shop/processor.go` (interface line ~42, `ListingSearchResult` ~90, method ~151)
- Modify: `services/atlas-merchant/atlas.com/merchant/shop/rest.go:133-176` (`ListingSearchRestModel`, `TransformSearchResult`)
- Modify: `services/atlas-merchant/atlas.com/merchant/shop/resource.go:154-189` (`handleSearchListings`)
- Modify: `services/atlas-merchant/atlas.com/merchant/shop/mock/processor.go:24,100-103`
- Modify: `services/atlas-merchant/atlas.com/merchant/README.md` (REST endpoints table)
- Test: `services/atlas-merchant/atlas.com/merchant/shop/provider_search_test.go` (new file)

**Interfaces:**
- Consumes: existing `listing.Entity`/`listing.Make`, `shop.Entity` columns (`character_id`, `shop_type`, `state`, `world_id` all exist — no schema change), `databasetest.NewInMemoryTenantDB` / `databasetest.TenantContext`.
- Produces (used by Tasks 8, 9):
  - `type ListingSearchCriteria struct { ItemId uint32; WorldId *world.Id; Descending bool }`
  - `Processor.SearchListingsByItemId(criteria ListingSearchCriteria) ([]ListingSearchResult, error)` (interface + impl + mock)
  - `ListingSearchResult` gains `ShopOwnerId uint32`, `ShopType ShopType`, `State State`
  - `ListingSearchRestModel` gains `OwnerId uint32 json:"ownerId"`, `ShopType byte json:"shopType"`, `State byte json:"state"`, `ItemSnapshot asset.AssetData json:"itemSnapshot"`
  - `const MaxSearchResults = 200`
  - REST: `GET /merchants/search/listings?itemId={id}&worldId={id}&order=asc|desc` — `worldId`/`order` optional (absent ⇒ old tenant-wide asc behavior, backward compatible; the owl path always passes both)

Key facts from research: the current query uses `db.Table("listings")` + explicit JOIN, which **bypasses** the automatic tenant callback (`hasTenantColumn` reads `db.Statement.Schema`, unset for `.Table()` queries) — so this task also adds the explicit `tenant_id` predicates the query has silently lacked. `listings.item_id` is already indexed (`listing/entity.go:16`).

- [ ] **Step 1: Write the failing provider tests**

Create `services/atlas-merchant/atlas.com/merchant/shop/provider_search_test.go`. Follow the style of `shop/provider_tenant_test.go` (uses `databasetest.NewInMemoryTenantDB(t, Migration)`; here we need both migrations):

```go
package shop

import (
	"testing"
	"time"

	"atlas-merchant/kafka/message/asset"
	"atlas-merchant/listing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedSearchData creates, for one tenant: an Open shop in world 0, an Open
// shop in world 1, and a Maintenance shop in world 0 — each with one listing
// for item 2060000 at ascending prices — plus a second tenant's world-0 shop
// with the same item.
func seedSearchData(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration, listing.Migration)
	tidA, tidB := uuid.New(), uuid.New()
	now := time.Now()

	mkShop := func(tid uuid.UUID, characterId uint32, worldId world.Id, state State, title string) uuid.UUID {
		id := uuid.New()
		require.NoError(t, db.Create(&Entity{
			Model:        gorm.Model{CreatedAt: now, UpdatedAt: now},
			Id:           id,
			TenantId:     tid,
			CharacterId:  characterId,
			ShopType:     byte(CharacterShop),
			State:        byte(state),
			Title:        title,
			WorldId:      worldId,
			ChannelId:    1,
			MapId:        910000004,
			InstanceId:   uuid.Nil,
			PermitItemId: 5140000,
		}).Error)
		return id
	}
	mkListing := func(tid uuid.UUID, shopId uuid.UUID, itemId uint32, price uint32) {
		require.NoError(t, db.Create(&listing.Entity{
			Model:            gorm.Model{CreatedAt: now, UpdatedAt: now},
			Id:               uuid.New(),
			TenantId:         tid,
			ShopId:           shopId,
			ItemId:           itemId,
			ItemType:         2,
			Quantity:         100,
			BundleSize:       100,
			BundlesRemaining: 1,
			PricePerBundle:   price,
			ItemSnapshot:     asset.AssetData{Quantity: 100},
			ListedAt:         now,
		}).Error)
	}

	sA0 := mkShop(tidA, 1001, 0, Open, "w0 open")
	sA1 := mkShop(tidA, 1002, 1, Open, "w1 open")
	sAM := mkShop(tidA, 1003, 0, Maintenance, "w0 maint")
	sB0 := mkShop(tidB, 2001, 0, Open, "tenantB w0")

	mkListing(tidA, sA0, 2060000, 1000)
	mkListing(tidA, sA1, 2060000, 2000)
	mkListing(tidA, sAM, 2060000, 3000)
	mkListing(tidB, sB0, 2060000, 500)

	return db, tidA, tidB
}

func TestSearchListings_WorldScopedAndTenantScoped(t *testing.T) {
	db, tidA, _ := seedSearchData(t)
	w0 := world.Id(0)
	results, err := searchListingsByItemId(tidA, ListingSearchCriteria{ItemId: 2060000, WorldId: &w0})(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, results, 2) // w0 open + w0 maintenance; w1 and tenantB excluded
	require.Equal(t, uint32(1000), results[0].Listing.PricePerBundle())
	require.Equal(t, uint32(3000), results[1].Listing.PricePerBundle())
	require.Equal(t, uint32(1001), results[0].ShopOwnerId)
	require.Equal(t, CharacterShop, results[0].ShopType)
	require.Equal(t, Open, results[0].State)
	require.Equal(t, Maintenance, results[1].State)
}

func TestSearchListings_NoWorldFilterKeepsOldBehavior(t *testing.T) {
	db, tidA, _ := seedSearchData(t)
	results, err := searchListingsByItemId(tidA, ListingSearchCriteria{ItemId: 2060000})(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, results, 3) // all tenant-A worlds, tenant B still excluded
}

func TestSearchListings_DescendingOrder(t *testing.T) {
	db, tidA, _ := seedSearchData(t)
	w0 := world.Id(0)
	results, err := searchListingsByItemId(tidA, ListingSearchCriteria{ItemId: 2060000, WorldId: &w0, Descending: true})(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, uint32(3000), results[0].Listing.PricePerBundle())
}

func TestSearchListings_CapAt200(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, listing.Migration)
	tid := uuid.New()
	now := time.Now()
	shopId := uuid.New()
	require.NoError(t, db.Create(&Entity{
		Model: gorm.Model{CreatedAt: now, UpdatedAt: now}, Id: shopId, TenantId: tid,
		CharacterId: 1001, ShopType: byte(CharacterShop), State: byte(Open),
		Title: "bulk", WorldId: 0, ChannelId: 1, MapId: 910000004,
		InstanceId: uuid.Nil, PermitItemId: 5140000,
	}).Error)
	for i := 0; i < 205; i++ {
		require.NoError(t, db.Create(&listing.Entity{
			Model: gorm.Model{CreatedAt: now, UpdatedAt: now}, Id: uuid.New(), TenantId: tid,
			ShopId: shopId, ItemId: 2060000, ItemType: 2, Quantity: 1, BundleSize: 1,
			BundlesRemaining: 1, PricePerBundle: uint32(1000 + i),
			ItemSnapshot: asset.AssetData{Quantity: 1}, ListedAt: now,
		}).Error)
	}
	w0 := world.Id(0)
	results, err := searchListingsByItemId(tid, ListingSearchCriteria{ItemId: 2060000, WorldId: &w0})(db.WithContext(databasetest.TenantContext(tid)))()
	require.NoError(t, err)
	require.Len(t, results, MaxSearchResults)
	// ascending truncates the most expensive tail
	require.Equal(t, uint32(1000), results[0].Listing.PricePerBundle())
	require.Equal(t, uint32(1000+MaxSearchResults-1), results[len(results)-1].Listing.PricePerBundle())
}
```


- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-merchant/atlas.com/merchant && go test ./shop/ -run TestSearchListings -v`
Expected: FAIL — `undefined: ListingSearchCriteria` (and signature mismatch on `searchListingsByItemId`)

- [ ] **Step 3: Implement provider + processor + REST + mock**

In `shop/processor.go`:

1. Add near `MaxListings = 16` / `MaxVisitors = 3` (line ~32):

```go
	// MaxSearchResults caps the shop-scanner search (client renders at most
	// 200 rows — SP_3630/3631, task-127 design §1.4).
	MaxSearchResults = 200
```

2. Add above `ListingSearchResult`:

```go
// ListingSearchCriteria narrows a listing search. WorldId nil means
// tenant-wide (pre-task-127 behavior); the owl path always sets it.
type ListingSearchCriteria struct {
	ItemId     uint32
	WorldId    *world.Id
	Descending bool
}
```

3. Extend `ListingSearchResult` (keep existing fields, append):

```go
type ListingSearchResult struct {
	Listing     listing.Model
	ShopId      uuid.UUID
	Title       string
	WorldId     world.Id
	ChannelId   channel.Id
	MapId       uint32
	ShopOwnerId uint32
	ShopType    ShopType
	State       State
}
```

4. Change the interface method (line ~42) and impl (line ~151):

```go
	SearchListingsByItemId(criteria ListingSearchCriteria) ([]ListingSearchResult, error)
```

```go
func (p *ProcessorImpl) SearchListingsByItemId(criteria ListingSearchCriteria) ([]ListingSearchResult, error) {
	return searchListingsByItemId(p.t.Id(), criteria)(p.db.WithContext(p.ctx))()
}
```

In `shop/provider.go`, replace `listingSearchRow` and `searchListingsByItemId`:

```go
type listingSearchRow struct {
	listing.Entity
	ShopTitle       string     `gorm:"column:shop_title"`
	ShopWorldId     world.Id   `gorm:"column:shop_world_id"`
	ShopChannelId   channel.Id `gorm:"column:shop_channel_id"`
	ShopMapId       uint32     `gorm:"column:shop_map_id"`
	ShopCharacterId uint32     `gorm:"column:shop_character_id"`
	ShopShopType    byte       `gorm:"column:shop_shop_type"`
	ShopState       byte       `gorm:"column:shop_state"`
}

// searchListingsByItemId joins listings to shops for the shop-scanner search.
// The .Table() form bypasses the automatic tenant callback (no bound schema),
// so tenant_id predicates are explicit here.
func searchListingsByItemId(tenantId uuid.UUID, criteria ListingSearchCriteria) database.EntityProvider[[]ListingSearchResult] {
	return func(db *gorm.DB) model.Provider[[]ListingSearchResult] {
		order := "listings.price_per_bundle ASC"
		if criteria.Descending {
			order = "listings.price_per_bundle DESC"
		}
		q := db.Table("listings").
			Select("listings.*, shops.title AS shop_title, shops.world_id AS shop_world_id, shops.channel_id AS shop_channel_id, shops.map_id AS shop_map_id, shops.character_id AS shop_character_id, shops.shop_type AS shop_shop_type, shops.state AS shop_state").
			Joins("JOIN shops ON shops.id = listings.shop_id").
			Where("listings.item_id = ? AND listings.tenant_id = ? AND shops.tenant_id = ? AND shops.state IN (?, ?)", criteria.ItemId, tenantId, tenantId, byte(Open), byte(Maintenance))
		if criteria.WorldId != nil {
			q = q.Where("shops.world_id = ?", *criteria.WorldId)
		}
		var rows []listingSearchRow
		err := q.Order(order).Limit(MaxSearchResults).Find(&rows).Error
		if err != nil {
			return model.ErrorProvider[[]ListingSearchResult](err)
		}

		results := make([]ListingSearchResult, 0, len(rows))
		for _, r := range rows {
			lm, err := listing.Make(r.Entity)
			if err != nil {
				return model.ErrorProvider[[]ListingSearchResult](err)
			}
			results = append(results, ListingSearchResult{
				Listing:     lm,
				ShopId:      r.Entity.ShopId,
				Title:       r.ShopTitle,
				WorldId:     r.ShopWorldId,
				ChannelId:   r.ShopChannelId,
				MapId:       r.ShopMapId,
				ShopOwnerId: r.ShopCharacterId,
				ShopType:    ShopType(r.ShopShopType),
				State:       State(r.ShopState),
			})
		}
		return model.FixedProvider(results)
	}
}
```

Add `"github.com/google/uuid"` to provider.go imports if not present.

In `shop/rest.go`, extend `ListingSearchRestModel` and `TransformSearchResult`:

```go
type ListingSearchRestModel struct {
	Id               string          `json:"-"`
	ShopId           string          `json:"shopId"`
	ShopTitle        string          `json:"shopTitle"`
	WorldId          byte            `json:"worldId"`
	ChannelId        byte            `json:"channelId"`
	MapId            uint32          `json:"mapId"`
	OwnerId          uint32          `json:"ownerId"`
	ShopType         byte            `json:"shopType"`
	State            byte            `json:"state"`
	ItemId           uint32          `json:"itemId"`
	ItemType         byte            `json:"itemType"`
	Quantity         uint16          `json:"quantity"`
	BundleSize       uint16          `json:"bundleSize"`
	BundlesRemaining uint16          `json:"bundlesRemaining"`
	PricePerBundle   uint32          `json:"pricePerBundle"`
	ItemSnapshot     asset.AssetData `json:"itemSnapshot"`
}
```

```go
func TransformSearchResult(sr ListingSearchResult) (ListingSearchRestModel, error) {
	return ListingSearchRestModel{
		Id:               sr.Listing.Id().String(),
		ShopId:           sr.ShopId.String(),
		ShopTitle:        sr.Title,
		WorldId:          byte(sr.WorldId),
		ChannelId:        byte(sr.ChannelId),
		MapId:            sr.MapId,
		OwnerId:          sr.ShopOwnerId,
		ShopType:         byte(sr.ShopType),
		State:            byte(sr.State),
		ItemId:           sr.Listing.ItemId(),
		ItemType:         sr.Listing.ItemType(),
		Quantity:         sr.Listing.Quantity(),
		BundleSize:       sr.Listing.BundleSize(),
		BundlesRemaining: sr.Listing.BundlesRemaining(),
		PricePerBundle:   sr.Listing.PricePerBundle(),
		ItemSnapshot:     sr.Listing.ItemSnapshot(),
	}, nil
}
```

Add `"atlas-merchant/kafka/message/asset"` to rest.go imports if not present.

In `shop/resource.go` `handleSearchListings`, after the existing `itemId` parse, add the optional params and switch to the criteria call:

```go
			criteria := ListingSearchCriteria{ItemId: uint32(v)}
			if ws := r.URL.Query().Get("worldId"); ws != "" {
				wv, err := strconv.ParseUint(ws, 10, 8)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				wid := world.Id(wv)
				criteria.WorldId = &wid
			}
			criteria.Descending = r.URL.Query().Get("order") == "desc"

			p := NewProcessor(d.Logger(), d.Context(), db)
			results, err := p.SearchListingsByItemId(criteria)
```

Add `"github.com/Chronicle20/atlas/libs/atlas-constants/world"` to resource.go imports if not present.

In `shop/mock/processor.go`, update the function field and method:

```go
	SearchListingsByItemIdFunc func(criteria shop.ListingSearchCriteria) ([]shop.ListingSearchResult, error)
```

```go
func (m *ProcessorMock) SearchListingsByItemId(criteria shop.ListingSearchCriteria) ([]shop.ListingSearchResult, error) {
	if m.SearchListingsByItemIdFunc != nil {
		return m.SearchListingsByItemIdFunc(criteria)
	}
	return nil, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-merchant/atlas.com/merchant && go test -race ./... -count=1 && go vet ./... && go build ./...`
Expected: all PASS (new search tests + the whole existing suite — any other `SearchListingsByItemId` caller the compiler finds must be updated to the criteria form), vet/build clean.

- [ ] **Step 5: Update the service README**

In `services/atlas-merchant/atlas.com/merchant/README.md`, update the REST endpoints row for `GET /merchants/search/listings` to document `itemId` (required), `worldId` (optional), `order=asc|desc` (optional, default asc), the 200-row cap, and the new response fields (`ownerId`, `shopType`, `state`, `itemSnapshot`).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-merchant/atlas.com/merchant/shop/ services/atlas-merchant/atlas.com/merchant/README.md
git commit -m "feat(task-127): world-scoped, ordered, capped, owner-enriched listing search in atlas-merchant"
```

---

### Task 5: atlas-merchant — searchcount package (persisted most-searched counts)

**Files:**
- Create: `services/atlas-merchant/atlas.com/merchant/searchcount/entity.go`
- Create: `services/atlas-merchant/atlas.com/merchant/searchcount/model.go`
- Create: `services/atlas-merchant/atlas.com/merchant/searchcount/administrator.go`
- Create: `services/atlas-merchant/atlas.com/merchant/searchcount/provider.go`
- Create: `services/atlas-merchant/atlas.com/merchant/searchcount/processor.go`
- Create: `services/atlas-merchant/atlas.com/merchant/searchcount/rest.go`
- Modify: `services/atlas-merchant/atlas.com/merchant/main.go:65` (add migration)
- Test: `services/atlas-merchant/atlas.com/merchant/searchcount/processor_test.go`

**Interfaces:**
- Consumes: `databasetest`, `tenant.MustFromContext`, `gorm.io/gorm/clause`.
- Produces (used by Tasks 6, 8):
  - `searchcount.NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor`
  - `Processor.RecordSearch(worldId world.Id, itemId uint32) error` (atomic upsert increment)
  - `Processor.GetTop(worldId world.Id, limit int) ([]Model, error)` (count DESC)
  - `Model.ItemId() uint32`, `Model.Count() uint64`
  - `searchcount.Migration`, `searchcount.RestModel` (`GetName() = "shop-search-counts"`), `searchcount.Transform`

- [ ] **Step 1: Write the failing tests**

`services/atlas-merchant/atlas.com/merchant/searchcount/processor_test.go`:

```go
package searchcount

import (
	"sync"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func newTestProcessor(t *testing.T) (Processor, Processor) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	l := logrus.New()
	tidA, tidB := uuid.New(), uuid.New()
	pA := NewProcessor(l, databasetest.TenantContext(tidA), db)
	pB := NewProcessor(l, databasetest.TenantContext(tidB), db)
	return pA, pB
}

func TestRecordSearch_IncrementsAndIsolatesTenants(t *testing.T) {
	pA, pB := newTestProcessor(t)

	require.NoError(t, pA.RecordSearch(0, 2060000))
	require.NoError(t, pA.RecordSearch(0, 2060000))
	require.NoError(t, pA.RecordSearch(0, 1302000))
	require.NoError(t, pB.RecordSearch(0, 2060000))

	top, err := pA.GetTop(0, 10)
	require.NoError(t, err)
	require.Len(t, top, 2)
	require.Equal(t, uint32(2060000), top[0].ItemId())
	require.Equal(t, uint64(2), top[0].Count())
	require.Equal(t, uint32(1302000), top[1].ItemId())
	require.Equal(t, uint64(1), top[1].Count())

	topB, err := pB.GetTop(0, 10)
	require.NoError(t, err)
	require.Len(t, topB, 1)
	require.Equal(t, uint64(1), topB[0].Count())
}

func TestRecordSearch_WorldScoped(t *testing.T) {
	pA, _ := newTestProcessor(t)
	require.NoError(t, pA.RecordSearch(0, 2060000))
	require.NoError(t, pA.RecordSearch(1, 2060000))
	require.NoError(t, pA.RecordSearch(1, 2060000))

	top0, err := pA.GetTop(0, 10)
	require.NoError(t, err)
	require.Len(t, top0, 1)
	require.Equal(t, uint64(1), top0[0].Count())

	top1, err := pA.GetTop(1, 10)
	require.NoError(t, err)
	require.Len(t, top1, 1)
	require.Equal(t, uint64(2), top1[0].Count())
}

func TestGetTop_LimitsToTen(t *testing.T) {
	pA, _ := newTestProcessor(t)
	for i := uint32(0); i < 15; i++ {
		itemId := 2060000 + i
		for j := uint32(0); j <= i; j++ {
			require.NoError(t, pA.RecordSearch(0, itemId))
		}
	}
	top, err := pA.GetTop(0, 10)
	require.NoError(t, err)
	require.Len(t, top, 10)
	// highest count first
	require.Equal(t, uint64(15), top[0].Count())
	require.Equal(t, uint32(2060014), top[0].ItemId())
}

// TestRecordSearch_ConcurrentIncrements: parallel increments must sum
// correctly (atomic upsert, no lost updates). sqlite serializes writers;
// if the driver returns SQLITE_BUSY under -race, bound the concurrency
// but keep total increments at 20.
func TestRecordSearch_ConcurrentIncrements(t *testing.T) {
	pA, _ := newTestProcessor(t)
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- pA.RecordSearch(world.Id(0), 2060000)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	top, err := pA.GetTop(0, 10)
	require.NoError(t, err)
	require.Len(t, top, 1)
	require.Equal(t, uint64(20), top[0].Count())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-merchant/atlas.com/merchant && go test ./searchcount/ -v`
Expected: FAIL — package doesn't exist yet (`no Go files` / undefined symbols).

- [ ] **Step 3: Implement the package**

`searchcount/entity.go`:

```go
package searchcount

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity is a per-tenant, per-world search counter for one item id.
// Tenant-safe PK pattern (FR-12): uuid surrogate PK + unique index on
// (tenant_id, world_id, item_id) — never a bare business-key PK.
type Entity struct {
	gorm.Model
	Id       uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_listing_search_counts_tenant_world_item"`
	WorldId  world.Id  `gorm:"not null;uniqueIndex:idx_listing_search_counts_tenant_world_item"`
	ItemId   uint32    `gorm:"not null;uniqueIndex:idx_listing_search_counts_tenant_world_item"`
	Count    uint64    `gorm:"not null;default:0"`
}

func (e *Entity) TableName() string {
	return "listing_search_counts"
}

func Make(e Entity) (Model, error) {
	return Model{itemId: e.ItemId, count: e.Count}, nil
}

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
```

`searchcount/model.go`:

```go
package searchcount

type Model struct {
	itemId uint32
	count  uint64
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Count() uint64 {
	return m.count
}
```

`searchcount/administrator.go`:

```go
package searchcount

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// incrementSearchCount is an atomic upsert: first search inserts count=1,
// subsequent searches increment in-place. Conflict target is the unique
// (tenant_id, world_id, item_id) index.
func incrementSearchCount(tenantId uuid.UUID, worldId world.Id, itemId uint32) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		e := &Entity{
			Id:       uuid.New(),
			TenantId: tenantId,
			WorldId:  worldId,
			ItemId:   itemId,
			Count:    1,
		}
		return db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "tenant_id"}, {Name: "world_id"}, {Name: "item_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"count": gorm.Expr("listing_search_counts.count + 1"),
			}),
		}).Create(e).Error
	}
}
```

`searchcount/provider.go`:

```go
package searchcount

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
)

// getTopByWorld returns the highest-count entities for a world. Uses a
// schema-bound Find so the automatic tenant callback scopes the query.
func getTopByWorld(worldId world.Id, limit int) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("world_id = ?", worldId).Order("count DESC").Limit(limit).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}
```

(Match the exact `database.EntityProvider` import path/alias used by `shop/provider.go`.)

`searchcount/processor.go`:

```go
package searchcount

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	RecordSearch(worldId world.Id, itemId uint32) error
	GetTop(worldId world.Id, limit int) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) RecordSearch(worldId world.Id, itemId uint32) error {
	return incrementSearchCount(p.t.Id(), worldId, itemId)(p.db.WithContext(p.ctx))
}

func (p *ProcessorImpl) GetTop(worldId world.Id, limit int) ([]Model, error) {
	return model.SliceMap(Make)(getTopByWorld(worldId, limit)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}
```

`searchcount/rest.go`:

```go
package searchcount

import "strconv"

type RestModel struct {
	Id     string `json:"-"`
	ItemId uint32 `json:"itemId"`
	Count  uint64 `json:"count"`
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r RestModel) GetName() string {
	return "shop-search-counts"
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:     strconv.FormatUint(uint64(m.ItemId()), 10),
		ItemId: m.ItemId(),
		Count:  m.Count(),
	}, nil
}
```

In `main.go:65`, add the migration:

```go
	db := database.Connect(l, database.SetMigrations(shop.Migration, listing.Migration, message.Migration, frederick.Migration, searchcount.Migration))
```

(and add `"atlas-merchant/searchcount"` to main.go imports).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-merchant/atlas.com/merchant && go test -race ./searchcount/ -v -count=1 && go vet ./... && go build ./...`
Expected: PASS (including the concurrent-increment test), vet/build clean. If sqlite returns `database is locked` under `-race`, keep total increments at 20 but run them from 4 goroutines × 5 sequential increments each — the assertion (total = 20) must not weaken.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-merchant/atlas.com/merchant/searchcount/ services/atlas-merchant/atlas.com/merchant/main.go
git commit -m "feat(task-127): persisted per-tenant+world listing search counts with atomic upsert"
```

---

### Task 6: atlas-merchant — RECORD_ITEM_SEARCH command + top-10 REST route

**Files:**
- Modify: `services/atlas-merchant/atlas.com/merchant/kafka/message/merchant/kafka.go` (const block ~line 11-27, bodies ~line 52+)
- Modify: `services/atlas-merchant/atlas.com/merchant/kafka/consumer/merchant/consumer.go` (`InitHandlers` ~line 29-48, new handler)
- Modify: `services/atlas-merchant/atlas.com/merchant/shop/resource.go` (wr subrouter, line ~37; new handler)
- Modify: `services/atlas-merchant/atlas.com/merchant/README.md` (Kafka commands + REST tables)
- Test: extends `services/atlas-merchant/atlas.com/merchant/searchcount/processor_test.go` coverage via consumer-level compile check (the handler is thin glue; logic already tested in Task 5)

**Interfaces:**
- Consumes: `searchcount.NewProcessor` (Task 5), existing `merchant2.Command[E]` envelope (carries `WorldId`, `CharacterId`), `message.AdaptHandler(message.PersistentConfig(...))` registration idiom.
- Produces (used by Task 9): command contract on `COMMAND_TOPIC_MERCHANT`:
  - `CommandRecordItemSearch = "RECORD_ITEM_SEARCH"`
  - `type CommandRecordItemSearchBody struct { ItemId uint32 json:"itemId" }`
  - REST: `GET /worlds/{worldId}/shop-searches/top` → JSON:API `[{itemId, count}]`, top 10 by count.

- [ ] **Step 1: Add the command type**

In `kafka/message/merchant/kafka.go`, add to the command const block:

```go
	CommandRecordItemSearch = "RECORD_ITEM_SEARCH"
```

and with the other body structs:

```go
type CommandRecordItemSearchBody struct {
	ItemId uint32 `json:"itemId"`
}
```

- [ ] **Step 2: Add the consumer handler**

In `kafka/consumer/merchant/consumer.go`, add (modeled exactly on `handleEnterShopCommand` at line ~210):

```go
func handleRecordItemSearchCommand(db *gorm.DB) message.Handler[merchant2.Command[merchant2.CommandRecordItemSearchBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e merchant2.Command[merchant2.CommandRecordItemSearchBody]) {
		if e.Type != merchant2.CommandRecordItemSearch {
			return
		}
		if err := searchcount.NewProcessor(l, ctx, db).RecordSearch(e.WorldId, e.Body.ItemId); err != nil {
			l.WithError(err).Errorf("Error recording item search for item [%d] in world [%d].", e.Body.ItemId, e.WorldId)
		}
	}
}
```

Register it inside `InitHandlers` alongside the existing `rf(...)` lines:

```go
		rf(t, message.AdaptHandler(message.PersistentConfig(handleRecordItemSearchCommand(db))))
```

Add `"atlas-merchant/searchcount"` to the consumer file's imports. No `InitConsumers` change — the topic subscription already exists.

- [ ] **Step 3: Add the REST route**

In `shop/resource.go`, on the existing `wr` subrouter (line ~37):

```go
			wr.HandleFunc("/shop-searches/top", registerHandler("get_top_shop_searches", handleGetTopShopSearches(db))).Methods(http.MethodGet)
```

and the handler (modeled on `handleSearchListings`; world id from the path via `rest.ParseWorldId`):

```go
func handleGetTopShopSearches(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				results, err := searchcount.NewProcessor(d.Logger(), d.Context(), db).GetTop(worldId, 10)
				if err != nil {
					d.Logger().WithError(err).Errorf("Getting top shop searches.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.SliceMap(searchcount.Transform)(model.FixedProvider(results))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST models.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]searchcount.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
```

Add `"atlas-merchant/searchcount"` to resource.go imports. No ingress (`deploy/shared/routes.conf`) change: the endpoint is service-internal (called by atlas-channel via the `MERCHANT` base URL), same as the existing `/worlds/{worldId}/channels/...` field-merchants route which is likewise not exposed.

- [ ] **Step 4: Verify**

Run: `cd services/atlas-merchant/atlas.com/merchant && go test -race ./... -count=1 && go vet ./... && go build ./...`
Expected: PASS/clean.

- [ ] **Step 5: Update README + commit**

Update `README.md`: add `RECORD_ITEM_SEARCH` to the Kafka commands table (`{worldId, characterId, type, body:{itemId}}`, increments the per-tenant+world search counter) and `GET /worlds/{worldId}/shop-searches/top` to the REST table (top-10 `shop-search-counts`).

```bash
git add services/atlas-merchant/atlas.com/merchant/kafka/ services/atlas-merchant/atlas.com/merchant/shop/resource.go services/atlas-merchant/atlas.com/merchant/README.md
git commit -m "feat(task-127): RECORD_ITEM_SEARCH command and top-10 shop-searches REST endpoint"
```

---

### Task 7: atlas-channel — merchant client extension (search, top-10, record-search command)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/merchant/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/merchant/rest.go`
- Modify: `services/atlas-channel/atlas.com/channel/merchant/requests.go`
- Modify: `services/atlas-channel/atlas.com/channel/merchant/model.go`
- Modify: `services/atlas-channel/atlas.com/channel/merchant/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/merchant/processor.go`
- Test: `services/atlas-channel/atlas.com/channel/merchant/rest_test.go` (new — Extract mapping)

**Interfaces:**
- Consumes: Task 4's REST response shape (`listing-search-results` with `ownerId/shopType/state/itemSnapshot`), Task 6's REST (`shop-search-counts`) and command contract.
- Produces (used by Tasks 10, 11, 12):
  - `merchant.SearchListing` model: `ShopId() uuid.UUID`, `Title() string`, `WorldId() world.Id`, `ChannelId() channel.Id`, `MapId() uint32`, `OwnerId() uint32`, `ShopType() byte`, `State() byte`, `ItemId() uint32`, `ItemType() byte`, `Quantity() uint16`, `BundleSize() uint16`, `BundlesRemaining() uint16`, `PricePerBundle() uint32`, `ItemSnapshot() SnapshotRestModel`
  - `merchant.TopSearch` model: `ItemId() uint32`, `Count() uint64`
  - `Processor.SearchListings(worldId world.Id, itemId uint32, descending bool) ([]SearchListing, error)`
  - `Processor.GetTopSearches(worldId world.Id) ([]TopSearch, error)`
  - `Processor.RecordItemSearch(f field.Model, characterId uint32, itemId uint32) error`
  - Channel-side shop-state constants `StateOpen byte = 2`, `StateMaintenance byte = 3` (mirroring `atlas-merchant/shop/state.go`)

- [ ] **Step 1: Write the failing Extract test**

`services/atlas-channel/atlas.com/channel/merchant/rest_test.go`:

```go
package merchant

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestExtractSearchListing(t *testing.T) {
	shopId := uuid.New()
	rm := ListingSearchRestModel{
		Id:               uuid.New().String(),
		ShopId:           shopId.String(),
		ShopTitle:        "cheap stuff",
		WorldId:          0,
		ChannelId:        2,
		MapId:            910000004,
		OwnerId:          30001,
		ShopType:         1,
		State:            StateOpen,
		ItemId:           2060000,
		ItemType:         2,
		Quantity:         100,
		BundleSize:       100,
		BundlesRemaining: 3,
		PricePerBundle:   5000,
		ItemSnapshot:     SnapshotRestModel{Quantity: 100},
	}
	m, err := ExtractSearchListing(rm)
	require.NoError(t, err)
	require.Equal(t, shopId, m.ShopId())
	require.Equal(t, "cheap stuff", m.Title())
	require.Equal(t, uint32(30001), m.OwnerId())
	require.Equal(t, byte(1), m.ShopType())
	require.Equal(t, StateOpen, m.State())
	require.Equal(t, uint32(910000004), m.MapId())
	require.Equal(t, uint16(3), m.BundlesRemaining())
	require.Equal(t, uint32(5000), m.PricePerBundle())
	require.Equal(t, uint32(100), m.ItemSnapshot().Quantity)
}

func TestExtractTopSearch(t *testing.T) {
	m, err := ExtractTopSearch(TopSearchRestModel{Id: "2060000", ItemId: 2060000, Count: 42})
	require.NoError(t, err)
	require.Equal(t, uint32(2060000), m.ItemId())
	require.Equal(t, uint64(42), m.Count())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./merchant/ -run TestExtract -v`
Expected: FAIL — `undefined: ListingSearchRestModel`, `undefined: ExtractSearchListing`, etc.

- [ ] **Step 3: Implement**

In `merchant/rest.go`, add (SnapshotRestModel mirrors atlas-merchant's `asset.AssetData` JSON tags exactly — verified against `services/atlas-merchant/atlas.com/merchant/kafka/message/asset/kafka.go:10-42`):

```go
// SnapshotRestModel mirrors atlas-merchant's asset.AssetData JSON shape —
// the listing's point-in-sale item snapshot, needed to encode the
// GW_ItemSlotBase block for equip rows in the shop-scanner result.
type SnapshotRestModel struct {
	Expiration     time.Time  `json:"expiration"`
	CreatedAt      time.Time  `json:"createdAt"`
	Quantity       uint32     `json:"quantity"`
	OwnerId        uint32     `json:"ownerId"`
	Flag           uint16     `json:"flag"`
	Rechargeable   uint64     `json:"rechargeable"`
	Strength       uint16     `json:"strength"`
	Dexterity      uint16     `json:"dexterity"`
	Intelligence   uint16     `json:"intelligence"`
	Luck           uint16     `json:"luck"`
	Hp             uint16     `json:"hp"`
	Mp             uint16     `json:"mp"`
	WeaponAttack   uint16     `json:"weaponAttack"`
	MagicAttack    uint16     `json:"magicAttack"`
	WeaponDefense  uint16     `json:"weaponDefense"`
	MagicDefense   uint16     `json:"magicDefense"`
	Accuracy       uint16     `json:"accuracy"`
	Avoidability   uint16     `json:"avoidability"`
	Hands          uint16     `json:"hands"`
	Speed          uint16     `json:"speed"`
	Jump           uint16     `json:"jump"`
	Slots          uint16     `json:"slots"`
	LevelType      byte       `json:"levelType"`
	Level          byte       `json:"level"`
	Experience     uint32     `json:"experience"`
	HammersApplied uint32     `json:"hammersApplied"`
	EquippedSince  *time.Time `json:"equippedSince"`
	CashId         int64      `json:"cashId,string"`
	CommodityId    uint32     `json:"commodityId"`
	PurchaseBy     uint32     `json:"purchaseBy"`
	PetId          uint32     `json:"petId"`
}

type ListingSearchRestModel struct {
	Id               string            `json:"-"`
	ShopId           string            `json:"shopId"`
	ShopTitle        string            `json:"shopTitle"`
	WorldId          byte              `json:"worldId"`
	ChannelId        byte              `json:"channelId"`
	MapId            uint32            `json:"mapId"`
	OwnerId          uint32            `json:"ownerId"`
	ShopType         byte              `json:"shopType"`
	State            byte              `json:"state"`
	ItemId           uint32            `json:"itemId"`
	ItemType         byte              `json:"itemType"`
	Quantity         uint16            `json:"quantity"`
	BundleSize       uint16            `json:"bundleSize"`
	BundlesRemaining uint16            `json:"bundlesRemaining"`
	PricePerBundle   uint32            `json:"pricePerBundle"`
	ItemSnapshot     SnapshotRestModel `json:"itemSnapshot"`
}

func (r ListingSearchRestModel) GetID() string {
	return r.Id
}

func (r *ListingSearchRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r ListingSearchRestModel) GetName() string {
	return "listing-search-results"
}

type TopSearchRestModel struct {
	Id     string `json:"-"`
	ItemId uint32 `json:"itemId"`
	Count  uint64 `json:"count"`
}

func (r TopSearchRestModel) GetID() string {
	return r.Id
}

func (r *TopSearchRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r TopSearchRestModel) GetName() string {
	return "shop-search-counts"
}
```

In `merchant/model.go`, add the domain models, extract functions, and the state constants:

```go
// Shop states mirroring atlas-merchant shop/state.go (byte on the wire).
const (
	StateOpen        byte = 2
	StateMaintenance byte = 3
)

type SearchListing struct {
	shopId           uuid.UUID
	title            string
	worldId          world.Id
	channelId        channel.Id
	mapId            uint32
	ownerId          uint32
	shopType         byte
	state            byte
	itemId           uint32
	itemType         byte
	quantity         uint16
	bundleSize       uint16
	bundlesRemaining uint16
	pricePerBundle   uint32
	itemSnapshot     SnapshotRestModel
}

func (m SearchListing) ShopId() uuid.UUID               { return m.shopId }
func (m SearchListing) Title() string                   { return m.title }
func (m SearchListing) WorldId() world.Id               { return m.worldId }
func (m SearchListing) ChannelId() channel.Id           { return m.channelId }
func (m SearchListing) MapId() uint32                   { return m.mapId }
func (m SearchListing) OwnerId() uint32                 { return m.ownerId }
func (m SearchListing) ShopType() byte                  { return m.shopType }
func (m SearchListing) State() byte                     { return m.state }
func (m SearchListing) ItemId() uint32                  { return m.itemId }
func (m SearchListing) ItemType() byte                  { return m.itemType }
func (m SearchListing) Quantity() uint16                { return m.quantity }
func (m SearchListing) BundleSize() uint16              { return m.bundleSize }
func (m SearchListing) BundlesRemaining() uint16        { return m.bundlesRemaining }
func (m SearchListing) PricePerBundle() uint32          { return m.pricePerBundle }
func (m SearchListing) ItemSnapshot() SnapshotRestModel { return m.itemSnapshot }

func ExtractSearchListing(rm ListingSearchRestModel) (SearchListing, error) {
	shopId, err := uuid.Parse(rm.ShopId)
	if err != nil {
		return SearchListing{}, err
	}
	return SearchListing{
		shopId:           shopId,
		title:            rm.ShopTitle,
		worldId:          world.Id(rm.WorldId),
		channelId:        channel.Id(rm.ChannelId),
		mapId:            rm.MapId,
		ownerId:          rm.OwnerId,
		shopType:         rm.ShopType,
		state:            rm.State,
		itemId:           rm.ItemId,
		itemType:         rm.ItemType,
		quantity:         rm.Quantity,
		bundleSize:       rm.BundleSize,
		bundlesRemaining: rm.BundlesRemaining,
		pricePerBundle:   rm.PricePerBundle,
		itemSnapshot:     rm.ItemSnapshot,
	}, nil
}

type TopSearch struct {
	itemId uint32
	count  uint64
}

func (m TopSearch) ItemId() uint32 { return m.itemId }
func (m TopSearch) Count() uint64  { return m.count }

func ExtractTopSearch(rm TopSearchRestModel) (TopSearch, error) {
	return TopSearch{itemId: rm.ItemId, count: rm.Count}, nil
}
```

(Add `world`/`channel` constants imports to model.go if absent.)

In `merchant/requests.go`, add resource strings + requests:

```go
const (
	SearchListingsResource = "merchants/search/listings?itemId=%d&worldId=%d&order=%s"
	TopSearchesResource    = "worlds/%d/shop-searches/top"
)

func requestSearchListings(itemId uint32, worldId world.Id, descending bool) requests.Request[[]ListingSearchRestModel] {
	order := "asc"
	if descending {
		order = "desc"
	}
	return requests.GetRequest[[]ListingSearchRestModel](fmt.Sprintf(getBaseRequest()+SearchListingsResource, itemId, worldId, order))
}

func requestTopSearches(worldId world.Id) requests.Request[[]TopSearchRestModel] {
	return requests.GetRequest[[]TopSearchRestModel](fmt.Sprintf(getBaseRequest()+TopSearchesResource, worldId))
}
```

(Merge the consts into the existing const block; add `fmt`/`world` imports if absent.)

In `services/atlas-channel/atlas.com/channel/kafka/message/merchant/kafka.go`, add:

```go
	CommandRecordItemSearch = "RECORD_ITEM_SEARCH"
```

```go
type CommandRecordItemSearchBody struct {
	ItemId uint32 `json:"itemId"`
}
```

In `merchant/producer.go`, add (modeled on `EnterShopCommandProvider` at line ~111):

```go
func RecordItemSearchCommandProvider(f field.Model, characterId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant2.Command[merchant2.CommandRecordItemSearchBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		CharacterId: characterId,
		Type:        merchant2.CommandRecordItemSearch,
		Body: merchant2.CommandRecordItemSearchBody{
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

In `merchant/processor.go`, add:

```go
func (p *Processor) SearchListings(worldId world.Id, itemId uint32, descending bool) ([]SearchListing, error) {
	return requests.SliceProvider[ListingSearchRestModel, SearchListing](p.l, p.ctx)(requestSearchListings(itemId, worldId, descending), ExtractSearchListing, model.Filters[SearchListing]())()
}

func (p *Processor) GetTopSearches(worldId world.Id) ([]TopSearch, error) {
	return requests.SliceProvider[TopSearchRestModel, TopSearch](p.l, p.ctx)(requestTopSearches(worldId), ExtractTopSearch, model.Filters[TopSearch]())()
}

func (p *Processor) RecordItemSearch(f field.Model, characterId uint32, itemId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(merchant2.EnvCommandTopic)(RecordItemSearchCommandProvider(f, characterId, itemId))
}
```

(Add `world` import to processor.go if absent.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./merchant/ -v -count=1 && go vet ./merchant/ ./kafka/...`
Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/merchant/ services/atlas-channel/atlas.com/channel/kafka/message/merchant/kafka.go
git commit -m "feat(task-127): atlas-channel merchant client — search listings, top searches, record-search command"
```

---

### Task 8: atlas-channel — shopscanner registry

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/shopscanner/registry.go`
- Test: `services/atlas-channel/atlas.com/channel/shopscanner/registry_test.go`

**Interfaces:**
- Consumes: `tenant.Model`, `uuid`, `_map.Id`. This file must NOT import `atlas-channel/session` (the socket bootstrap imports both this package and session — keep the registry dependency-free to avoid cycles).
- Produces (used by Tasks 10, 11, 12):
  - `shopscanner.GetRegistry() *Registry`
  - `Registry.SetLastSearch(t tenant.Model, characterId uint32, itemId uint32)`
  - `Registry.GetLastSearch(t tenant.Model, characterId uint32) (SearchEntry, bool)`
  - `Registry.SetPending(t tenant.Model, characterId uint32, e PendingEntry)`
  - `Registry.GetPending(t tenant.Model, characterId uint32) (PendingEntry, bool)`
  - `Registry.RemovePending(t tenant.Model, characterId uint32)`
  - `Registry.ClearCharacter(t tenant.Model, characterId uint32)`
  - `type SearchEntry struct { ItemId uint32 }`
  - `type PendingEntry struct { ShopId uuid.UUID; OwnerId uint32; MapId _map.Id }`

- [ ] **Step 1: Write the failing test**

`services/atlas-channel/atlas.com/channel/shopscanner/registry_test.go`:

```go
package shopscanner

import (
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return m
}

func TestRegistry_LastSearchLifecycle(t *testing.T) {
	r := GetRegistry()
	ta, tb := testTenant(t), testTenant(t)

	_, ok := r.GetLastSearch(ta, 1)
	require.False(t, ok)

	r.SetLastSearch(ta, 1, 2060000)
	e, ok := r.GetLastSearch(ta, 1)
	require.True(t, ok)
	require.Equal(t, uint32(2060000), e.ItemId)

	// overwrite on reuse
	r.SetLastSearch(ta, 1, 1302000)
	e, _ = r.GetLastSearch(ta, 1)
	require.Equal(t, uint32(1302000), e.ItemId)

	// tenant isolation
	_, ok = r.GetLastSearch(tb, 1)
	require.False(t, ok)

	r.ClearCharacter(ta, 1)
	_, ok = r.GetLastSearch(ta, 1)
	require.False(t, ok)
}

func TestRegistry_PendingEntryLifecycle(t *testing.T) {
	r := GetRegistry()
	ta := testTenant(t)
	shopId := uuid.New()

	r.SetPending(ta, 2, PendingEntry{ShopId: shopId, OwnerId: 30001, MapId: _map.Id(910000004)})
	pe, ok := r.GetPending(ta, 2)
	require.True(t, ok)
	require.Equal(t, shopId, pe.ShopId)
	require.Equal(t, uint32(30001), pe.OwnerId)
	require.Equal(t, _map.Id(910000004), pe.MapId)

	r.RemovePending(ta, 2)
	_, ok = r.GetPending(ta, 2)
	require.False(t, ok)
}

func TestRegistry_ClearCharacterClearsBoth(t *testing.T) {
	r := GetRegistry()
	ta := testTenant(t)
	r.SetLastSearch(ta, 3, 2060000)
	r.SetPending(ta, 3, PendingEntry{ShopId: uuid.New(), OwnerId: 30001, MapId: _map.Id(910000004)})
	r.ClearCharacter(ta, 3)
	_, ok1 := r.GetLastSearch(ta, 3)
	_, ok2 := r.GetPending(ta, 3)
	require.False(t, ok1)
	require.False(t, ok2)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./shopscanner/ -v`
Expected: FAIL — package doesn't exist.

- [ ] **Step 3: Implement the registry**

`services/atlas-channel/atlas.com/channel/shopscanner/registry.go` (singleton `sync.Once` + `sync.RWMutex` + tenant-scoped keys — the `account/registry.go` pattern):

```go
package shopscanner

import (
	"sync"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

type Key struct {
	Tenant      tenant.Model
	CharacterId uint32
}

// SearchEntry remembers a character's most recent executed owl search — the
// OWL_WARP handler validates the clicked result against it.
type SearchEntry struct {
	ItemId uint32
}

// PendingEntry marks a warp-then-enter in flight: set when OWL_WARP passes
// validation, consumed on VisitorEntered/CapacityFull, dropped on
// arrival-map mismatch and session destroy.
type PendingEntry struct {
	ShopId  uuid.UUID
	OwnerId uint32
	MapId   _map.Id
}

type Registry struct {
	mutex      sync.RWMutex
	lastSearch map[Key]SearchEntry
	pending    map[Key]PendingEntry
}

var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.lastSearch = make(map[Key]SearchEntry)
		registry.pending = make(map[Key]PendingEntry)
	})
	return registry
}

func (r *Registry) SetLastSearch(t tenant.Model, characterId uint32, itemId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.lastSearch[Key{Tenant: t, CharacterId: characterId}] = SearchEntry{ItemId: itemId}
}

func (r *Registry) GetLastSearch(t tenant.Model, characterId uint32) (SearchEntry, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	e, ok := r.lastSearch[Key{Tenant: t, CharacterId: characterId}]
	return e, ok
}

func (r *Registry) SetPending(t tenant.Model, characterId uint32, e PendingEntry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.pending[Key{Tenant: t, CharacterId: characterId}] = e
}

func (r *Registry) GetPending(t tenant.Model, characterId uint32) (PendingEntry, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	e, ok := r.pending[Key{Tenant: t, CharacterId: characterId}]
	return e, ok
}

func (r *Registry) RemovePending(t tenant.Model, characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.pending, Key{Tenant: t, CharacterId: characterId})
}

// ClearCharacter drops all scanner state for a character (session destroy).
func (r *Registry) ClearCharacter(t tenant.Model, characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	k := Key{Tenant: t, CharacterId: characterId}
	delete(r.lastSearch, k)
	delete(r.pending, k)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./shopscanner/ -v -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/shopscanner/
git commit -m "feat(task-127): shopscanner registry (last search + pending shop entry, tenant-scoped)"
```

---

### Task 9: atlas-channel — shop-scanner writer bodies + record conversion

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/writer/shop_scanner.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/writer/shop_scanner_test.go`

**Interfaces:**
- Consumes: Task 3 lib factories (`merchantpkt.ShopScannerResultBody/ShopScannerHotListBody/ShopLinkResultBody`), Task 7 `merchant.SearchListing`/`SnapshotRestModel`, `pktmodel.NewAsset` chain.
- Produces (used by Tasks 10, 11, 12):
  - `writer.ShopScannerResultBody(itemId uint32, records []merchantcb.ShopScannerRecord) packet.Encode`
  - `writer.ShopScannerHotListBody(itemIds []uint32) packet.Encode`
  - `writer.ShopLinkResultBody(code merchantpkt.ShopLinkResultCode) packet.Encode`
  - `writer.ShopScannerRecords(listings []merchant.SearchListing, names map[uint32]string) []merchantcb.ShopScannerRecord` — pure conversion: 0-based channel byte, equip rows get a slotless `model.Asset` built from the snapshot.

- [ ] **Step 1: Write the failing conversion test**

`services/atlas-channel/atlas.com/channel/socket/writer/shop_scanner_test.go`:

```go
package writer

import (
	"testing"

	"atlas-channel/merchant"

	"github.com/stretchr/testify/require"
)

func TestShopScannerRecords_Conversion(t *testing.T) {
	listings := []merchant.SearchListing{
		merchant.NewSearchListing(merchant.SearchListingSeed{
			Title: "cheap stuff", WorldId: 0, ChannelId: 3, MapId: 910000004,
			OwnerId: 30001, ShopType: 1, State: merchant.StateOpen,
			ItemId: 2060000, ItemType: 2, BundleSize: 100, BundlesRemaining: 3,
			PricePerBundle: 5000,
		}),
	}
	records := ShopScannerRecords(listings, map[uint32]string{30001: "OwnerA"})
	require.Len(t, records, 1)
	r := records[0]
	require.Equal(t, "OwnerA", r.OwnerName())
	require.Equal(t, uint32(910000004), r.MapId())
	require.Equal(t, "cheap stuff", r.Title())
	require.Equal(t, uint32(3), r.Bundles())      // nNumber = bundles available
	require.Equal(t, uint32(100), r.BundleSize()) // nSet = quantity per bundle
	require.Equal(t, uint32(5000), r.Price())
	require.Equal(t, uint32(30001), r.OwnerId()) // dwMiniRoomSN echo
	require.Equal(t, byte(2), r.ChannelId())     // 0-based: channel 3 -> 2
	require.Equal(t, byte(2), r.InventoryType())
	require.Nil(t, r.Asset())
}

func TestShopScannerRecords_EquipRowGetsAsset(t *testing.T) {
	listings := []merchant.SearchListing{
		merchant.NewSearchListing(merchant.SearchListingSeed{
			Title: "swords", WorldId: 0, ChannelId: 1, MapId: 910000004,
			OwnerId: 30002, ShopType: 2, State: merchant.StateOpen,
			ItemId: 1302000, ItemType: 1, BundleSize: 1, BundlesRemaining: 1,
			PricePerBundle: 150000,
			Snapshot: merchant.SnapshotRestModel{
				Strength: 5, Dexterity: 3, WeaponAttack: 17, Slots: 7,
			},
		}),
	}
	records := ShopScannerRecords(listings, map[uint32]string{})
	require.Len(t, records, 1)
	require.Equal(t, byte(1), records[0].InventoryType())
	require.NotNil(t, records[0].Asset())
	require.Equal(t, "", records[0].OwnerName()) // missing name -> empty string
}
```

This test needs a seed constructor on the channel merchant model. Add to `services/atlas-channel/atlas.com/channel/merchant/model.go` (a plain exported constructor — NOT a `*_testhelpers.go` file; it is the model's builder-style entry point for locally-constructed values):

```go
// SearchListingSeed carries the constructor arguments for SearchListing.
type SearchListingSeed struct {
	ShopId           uuid.UUID
	Title            string
	WorldId          world.Id
	ChannelId        channel.Id
	MapId            uint32
	OwnerId          uint32
	ShopType         byte
	State            byte
	ItemId           uint32
	ItemType         byte
	Quantity         uint16
	BundleSize       uint16
	BundlesRemaining uint16
	PricePerBundle   uint32
	Snapshot         SnapshotRestModel
}

// NewSearchListing builds a SearchListing from explicit values (the model's
// constructor for locally-built values; Extract remains the REST path).
func NewSearchListing(s SearchListingSeed) SearchListing {
	return SearchListing{
		shopId:           s.ShopId,
		title:            s.Title,
		worldId:          s.WorldId,
		channelId:        s.ChannelId,
		mapId:            s.MapId,
		ownerId:          s.OwnerId,
		shopType:         s.ShopType,
		state:            s.State,
		itemId:           s.ItemId,
		itemType:         s.ItemType,
		quantity:         s.Quantity,
		bundleSize:       s.BundleSize,
		bundlesRemaining: s.BundlesRemaining,
		pricePerBundle:   s.PricePerBundle,
		itemSnapshot:     s.Snapshot,
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/writer/ -run TestShopScannerRecords -v`
Expected: FAIL — `undefined: ShopScannerRecords` (and `NewSearchListing` until added).

- [ ] **Step 3: Implement**

`services/atlas-channel/atlas.com/channel/socket/writer/shop_scanner.go`:

```go
package writer

import (
	"atlas-channel/merchant"

	merchantpkt "github.com/Chronicle20/atlas/libs/atlas-packet/merchant"
	merchantcb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound"
	pktmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// ShopScannerResult CWvsContext::OnShopScannerResult
// ShopLinkResult CWvsContext::OnShopLinkResult
// Mode bytes and link codes are config-resolved from the tenant template's
// options.operations tables (never hard-coded), matching world_message.go.

func ShopScannerResultBody(itemId uint32, records []merchantcb.ShopScannerRecord) packet.Encode {
	return merchantpkt.ShopScannerResultBody(itemId, records)
}

func ShopScannerHotListBody(itemIds []uint32) packet.Encode {
	return merchantpkt.ShopScannerHotListBody(itemIds)
}

func ShopLinkResultBody(code merchantpkt.ShopLinkResultCode) packet.Encode {
	return merchantpkt.ShopLinkResultBody(code)
}

// ShopScannerRecords converts merchant search listings plus resolved owner
// names into wire records: channel encoded 0-based (channel.Id - 1, matching
// server_list_entry.go), equip rows (itemType 1) get a slotless
// GW_ItemSlotBase built from the listing's point-in-sale snapshot, and a
// missing owner name degrades to empty string rather than failing the search.
func ShopScannerRecords(listings []merchant.SearchListing, names map[uint32]string) []merchantcb.ShopScannerRecord {
	records := make([]merchantcb.ShopScannerRecord, 0, len(listings))
	for _, sl := range listings {
		var assetPtr *pktmodel.Asset
		if sl.ItemType() == 1 {
			snap := sl.ItemSnapshot()
			asset := pktmodel.NewAsset(true, 0, sl.ItemId(), snap.Expiration).
				SetEquipmentStats(snap.Strength, snap.Dexterity, snap.Intelligence, snap.Luck,
					snap.Hp, snap.Mp, snap.WeaponAttack, snap.MagicAttack, snap.WeaponDefense,
					snap.MagicDefense, snap.Accuracy, snap.Avoidability, snap.Hands, snap.Speed, snap.Jump).
				SetEquipmentMeta(snap.Slots, snap.LevelType, snap.Level, snap.Experience, snap.HammersApplied, snap.Flag)
			assetPtr = &asset
		}
		records = append(records, merchantcb.NewShopScannerRecord(
			names[sl.OwnerId()],
			sl.MapId(),
			sl.Title(),
			uint32(sl.BundlesRemaining()),
			uint32(sl.BundleSize()),
			sl.PricePerBundle(),
			sl.OwnerId(),
			byte(sl.ChannelId())-1,
			sl.ItemType(),
			assetPtr,
		))
	}
	return records
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/writer/ ./merchant/ -v -count=1 && go vet ./socket/writer/ ./merchant/`
Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/writer/shop_scanner.go services/atlas-channel/atlas.com/channel/socket/writer/shop_scanner_test.go services/atlas-channel/atlas.com/channel/merchant/model.go
git commit -m "feat(task-127): channel shop-scanner writer bodies and record conversion"
```

---

### Task 10: atlas-channel — shopscanner processor (search flow, hot list, warp evaluation)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/shopscanner/processor.go`
- Create: `services/atlas-channel/atlas.com/channel/shopscanner/warp.go`
- Test: `services/atlas-channel/atlas.com/channel/shopscanner/warp_test.go`

**Interfaces:**
- Consumes: Tasks 1, 3, 7, 8, 9 outputs; `session.Announce`, `consumable.RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, updateTime uint32)`, `character.NewProcessor(l, ctx).GetById()(id)`.
- Produces (used by Tasks 11, 12):
  - `shopscanner.NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor`
  - `Processor.Search(wp writer.Producer) func(s session.Model, searchItemId uint32, descending bool, owlItemId item.Id, source slot.Position, updateTime uint32) error`
  - `Processor.SendHotList(wp writer.Producer) func(s session.Model) error`
  - `shopscanner.WarpCheck` struct + `shopscanner.EvaluateWarp(c WarpCheck) (merchantpkt.ShopLinkResultCode, bool)` — the pure validation ladder (design §4.2); `(code, false)` = announce that SHOP_LINK code, `("", true)` = proceed with warp.

- [ ] **Step 1: Write the failing ladder test**

`services/atlas-channel/atlas.com/channel/shopscanner/warp_test.go`:

```go
package shopscanner

import (
	"testing"

	merchantpkt "github.com/Chronicle20/atlas/libs/atlas-packet/merchant"
	"github.com/stretchr/testify/require"
)

// valid returns a WarpCheck that passes every rung — each case below breaks
// exactly one rung and expects the design §4.2 code.
func valid() WarpCheck {
	return WarpCheck{
		HasSearch:        true,
		OwnerId:          30001,
		CharacterId:      1,
		CharacterHp:      50,
		CurrentMapFM:     true,
		ShopFound:        true,
		ShopWorldId:      0,
		SessionWorldId:   0,
		ShopChannelId:    1,
		SessionChannelId: 1,
		ShopMapId:        910000004,
		EchoedMapId:      910000004,
		ShopState:        2, // Open
		ListingPresent:   true,
	}
}

func TestEvaluateWarp(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*WarpCheck)
		code   merchantpkt.ShopLinkResultCode
		ok     bool
	}{
		{"all valid", func(c *WarpCheck) {}, "", true},
		{"outside FM", func(c *WarpCheck) { c.CurrentMapFM = false }, merchantpkt.ShopLinkResultCodeFMOnly, false},
		{"no prior search", func(c *WarpCheck) { c.HasSearch = false }, merchantpkt.ShopLinkResultCodeClosed, false},
		{"own shop", func(c *WarpCheck) { c.OwnerId = 1 }, merchantpkt.ShopLinkResultCodeDenied, false},
		{"dead", func(c *WarpCheck) { c.CharacterHp = 0 }, merchantpkt.ShopLinkResultCodeDead, false},
		{"shop missing", func(c *WarpCheck) { c.ShopFound = false }, merchantpkt.ShopLinkResultCodeClosed, false},
		{"wrong world", func(c *WarpCheck) { c.ShopWorldId = 1 }, merchantpkt.ShopLinkResultCodeClosed, false},
		{"echo tamper (map mismatch)", func(c *WarpCheck) { c.EchoedMapId = 910000005 }, merchantpkt.ShopLinkResultCodeClosed, false},
		{"shop outside FM", func(c *WarpCheck) { c.ShopMapId = 100000000; c.EchoedMapId = 100000000 }, merchantpkt.ShopLinkResultCodeFMOnly, false},
		{"cross channel", func(c *WarpCheck) { c.ShopChannelId = 2 }, merchantpkt.ShopLinkResultCodeClosed, false},
		{"maintenance", func(c *WarpCheck) { c.ShopState = 3 }, merchantpkt.ShopLinkResultCodeMaintenance, false},
		{"closed state", func(c *WarpCheck) { c.ShopState = 4 }, merchantpkt.ShopLinkResultCodeClosed, false},
		{"listing gone", func(c *WarpCheck) { c.ListingPresent = false }, merchantpkt.ShopLinkResultCodeBusy, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := valid()
			tc.mutate(&c)
			code, ok := EvaluateWarp(c)
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.code, code)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./shopscanner/ -run TestEvaluateWarp -v`
Expected: FAIL — `undefined: WarpCheck`, `undefined: EvaluateWarp`

- [ ] **Step 3: Implement warp.go**

`services/atlas-channel/atlas.com/channel/shopscanner/warp.go`:

```go
package shopscanner

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	merchantpkt "github.com/Chronicle20/atlas/libs/atlas-packet/merchant"
)

// WarpCheck is the pre-fetched state the OWL_WARP validation ladder
// (design §4.2) evaluates. The handler gathers; this decides.
type WarpCheck struct {
	HasSearch        bool
	OwnerId          uint32
	CharacterId      uint32
	CharacterHp      uint16
	CurrentMapFM     bool
	ShopFound        bool
	ShopWorldId      world.Id
	SessionWorldId   world.Id
	ShopChannelId    channel.Id
	SessionChannelId channel.Id
	ShopMapId        uint32
	EchoedMapId      uint32
	ShopState        byte
	ListingPresent   bool
}

// Shop states mirroring atlas-merchant shop/state.go.
const (
	shopStateOpen        byte = 2
	shopStateMaintenance byte = 3
)

// EvaluateWarp walks the ladder in order; the first failing rung yields its
// SHOP_LINK code. ("", true) means the warp may proceed.
func EvaluateWarp(c WarpCheck) (merchantpkt.ShopLinkResultCode, bool) {
	if !c.CurrentMapFM {
		return merchantpkt.ShopLinkResultCodeFMOnly, false
	}
	if !c.HasSearch {
		return merchantpkt.ShopLinkResultCodeClosed, false
	}
	if c.OwnerId == c.CharacterId {
		return merchantpkt.ShopLinkResultCodeDenied, false
	}
	if c.CharacterHp == 0 {
		return merchantpkt.ShopLinkResultCodeDead, false
	}
	if !c.ShopFound {
		return merchantpkt.ShopLinkResultCodeClosed, false
	}
	if c.ShopWorldId != c.SessionWorldId {
		return merchantpkt.ShopLinkResultCodeClosed, false
	}
	if c.ShopMapId != c.EchoedMapId {
		return merchantpkt.ShopLinkResultCodeClosed, false
	}
	if !_map.IsFreeMarketRoom(_map.Id(c.ShopMapId)) {
		return merchantpkt.ShopLinkResultCodeFMOnly, false
	}
	if c.ShopChannelId != c.SessionChannelId {
		return merchantpkt.ShopLinkResultCodeClosed, false
	}
	if c.ShopState == shopStateMaintenance {
		return merchantpkt.ShopLinkResultCodeMaintenance, false
	}
	if c.ShopState != shopStateOpen {
		return merchantpkt.ShopLinkResultCodeClosed, false
	}
	if !c.ListingPresent {
		return merchantpkt.ShopLinkResultCodeBusy, false
	}
	return "", true
}
```

- [ ] **Step 4: Run ladder test**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./shopscanner/ -run TestEvaluateWarp -v`
Expected: PASS.

- [ ] **Step 5: Implement processor.go**

`services/atlas-channel/atlas.com/channel/shopscanner/processor.go`:

```go
package shopscanner

import (
	"atlas-channel/character"
	"atlas-channel/consumable"
	"atlas-channel/merchant"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	merchantcb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx, t: tenant.MustFromContext(ctx)}
}

// Search executes an owl search: world-scoped merchant lookup, owner-name
// resolution, fire-and-forget count increment, mode-6 result write, and —
// only when at least one listing came back — owl consumption (design §2 Q3:
// consume 1 per search with >=1 result; empty search consumes nothing).
func (p *Processor) Search(wp writer.Producer) func(s session.Model, searchItemId uint32, descending bool, owlItemId item.Id, source slot.Position, updateTime uint32) error {
	return func(s session.Model, searchItemId uint32, descending bool, owlItemId item.Id, source slot.Position, updateTime uint32) error {
		if !_map.IsFreeMarketRoom(s.MapId()) {
			// The client cannot send this honestly (RunShopScanner hard-blocks
			// outside FM) — packet injection; drop.
			p.l.Warnf("Character [%d] attempted an owl search outside the Free Market (map [%d]).", s.CharacterId(), s.MapId())
			return nil
		}

		mp := merchant.NewProcessor(p.l, p.ctx)

		// Count increment is result-independent (Cosmic parity) and must never
		// block or fail the search.
		if err := mp.RecordItemSearch(s.Field(), s.CharacterId(), searchItemId); err != nil {
			p.l.WithError(err).Warnf("Unable to record item search for character [%d], item [%d].", s.CharacterId(), searchItemId)
		}

		listings, err := mp.SearchListings(s.WorldId(), searchItemId, descending)
		if err != nil {
			p.l.WithError(err).Errorf("Owl search failed for character [%d], item [%d]; sending empty result.", s.CharacterId(), searchItemId)
			listings = nil
		}

		names := p.resolveOwnerNames(listings)
		records := writer.ShopScannerRecords(listings, names)

		p.l.Debugf("Character [%d] owl search for item [%d]: [%d] results.", s.CharacterId(), searchItemId, len(records))
		if err := session.Announce(p.l)(p.ctx)(wp)(merchantcb.ShopScannerResultWriter)(writer.ShopScannerResultBody(searchItemId, records))(s); err != nil {
			p.l.WithError(err).Errorf("Unable to announce shop scanner result to character [%d].", s.CharacterId())
			return err
		}

		if len(listings) > 0 {
			if err := consumable.NewProcessor(p.l, p.ctx).RequestItemConsume(s.Field(), characterconst.Id(s.CharacterId()), owlItemId, source, updateTime); err != nil {
				p.l.WithError(err).Errorf("Unable to consume owl [%d] for character [%d].", owlItemId, s.CharacterId())
			}
		}

		GetRegistry().SetLastSearch(p.t, s.CharacterId(), searchItemId)
		return nil
	}
}

// resolveOwnerNames resolves distinct owner ids to names, deduplicated per
// request; a failed lookup degrades to empty string for that row.
func (p *Processor) resolveOwnerNames(listings []merchant.SearchListing) map[uint32]string {
	names := make(map[uint32]string)
	cp := character.NewProcessor(p.l, p.ctx)
	for _, sl := range listings {
		if _, ok := names[sl.OwnerId()]; ok {
			continue
		}
		c, err := cp.GetById()(sl.OwnerId())
		if err != nil {
			p.l.WithError(err).Warnf("Unable to resolve owner name for character [%d].", sl.OwnerId())
			names[sl.OwnerId()] = ""
			continue
		}
		names[sl.OwnerId()] = c.Name()
	}
	return names
}

// SendHotList answers OWL_ACTION mode OPEN with the mode-7 most-searched list.
func (p *Processor) SendHotList(wp writer.Producer) func(s session.Model) error {
	return func(s session.Model) error {
		top, err := merchant.NewProcessor(p.l, p.ctx).GetTopSearches(s.WorldId())
		if err != nil {
			p.l.WithError(err).Errorf("Unable to fetch top searches for world [%d]; sending empty hot list.", s.WorldId())
			top = nil
		}
		itemIds := make([]uint32, 0, len(top))
		for _, ts := range top {
			itemIds = append(itemIds, ts.ItemId())
		}
		return session.Announce(p.l)(p.ctx)(wp)(merchantcb.ShopScannerResultWriter)(writer.ShopScannerHotListBody(itemIds))(s)
	}
}
```

Note: confirm the exact package/type for `characterconst.Id` against the existing call `consumable.NewProcessor(l, ctx).RequestItemConsume(s.Field(), character.Id(s.CharacterId()), itemId, source, updateTime)` in `socket/handler/character_cash_item_use.go:51` (there `character` is the alias for `github.com/Chronicle20/atlas/libs/atlas-constants/character`) — use the same import and conversion.

- [ ] **Step 6: Run tests + vet**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./shopscanner/ -v -count=1 && go vet ./shopscanner/`
Expected: PASS, vet clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/shopscanner/
git commit -m "feat(task-127): shopscanner processor — search flow, hot list, warp validation ladder"
```

---

### Task 11: atlas-channel — socket handlers + main.go registration

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/owl_action.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/owl_warp.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/shop_scanner_item_use.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` (const block ~114-120, 523 branch ~310-312, new arm before fallthrough ~108, handler signature line 25)
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (`produceHandlers()` ~line 867, `produceWriters()` ~line 608-793)

**Interfaces:**
- Consumes: Tasks 2, 3, 8, 9, 10 outputs; existing `merchantsb`/`merchantcb` import aliases in main.go (lines 99-100); `portal.NewProcessor(l, ctx).Warp(f field.Model, characterId uint32, targetMapId _map.Id) error`; `character2.NewProcessor(l, ctx).GetItemInSlot(characterId, inventoryType, slot) model.Provider[asset.Model]`; `atlas_packet.ResolveCode(l, readerOptions, "operations", key)`.
- Produces: three registered handlers + one new cash arm + two registered writers. Handler name strings (`OwlActionHandle`, `OwlWarpHandle`, `ShopScannerItemUseHandle`) and writer names (`ShopScannerResult`, `ShopLinkResult`) become referencable from tenant config (Task 13).

- [ ] **Step 1: Add the cash 523 arm**

In `socket/handler/character_cash_item_use.go`:

1. Change the handler factory signature (line 25) from `_ writer.Producer` to `wp writer.Producer` (the new arm announces via the shopscanner processor).
2. Add the named constant to the const block (lines 114-120):

```go
const (
	CashSlotItemTypeFieldEffect   = CashSlotItemType(16)
	CashSlotItemTypeStoreSearch   = CashSlotItemType(29)
	CashSlotItemTypePetConsumable = CashSlotItemType(30)
	CashSlotItemTypeChalkboard    = CashSlotItemType(32)
)
```

3. In `GetCashSlotItemType`, replace the raw literal in the 523 branch (lines 310-312):

```go
		if category == item.ClassificationStoreSearch {
			return CashSlotItemTypeStoreSearch
		}
```

4. Add the arm directly after the chalkboard arm (before the field-effect arm is fine too — keep it adjacent to the other early-return arms, before the warn-and-drop fallthrough at line ~108):

```go
		if it == CashSlotItemTypeStoreSearch {
			sp := cashsb.NewItemUseStoreSearch()
			sp.Decode(l, ctx)(r, readerOptions)
			_ = shopscanner.NewProcessor(l, ctx).Search(wp)(s, sp.SearchItemId(), sp.Descending(), itemId, source, sp.UpdateTime())
			return
		}
```

Add `"atlas-channel/shopscanner"` to the file's imports. Note the tail codec always carries its own trailing updateTime (design §1.2), so the prefix-vs-tail updateTime juggling the pet arm does is not needed here — `sp.UpdateTime()` is authoritative.

- [ ] **Step 2: Create the three handlers**

`services/atlas-channel/atlas.com/channel/socket/handler/owl_action.go`:

```go
package handler

import (
	"atlas-channel/session"
	"atlas-channel/shopscanner"
	"atlas-channel/socket/writer"
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	merchantsb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// OwlActionHandleFunc answers CUIShopScanner::OnCreate (mode OPEN, the only
// mode the client ever sends — task-127 design §1.3) with the most-searched
// hot list. The expected mode byte is config-resolved from the handler
// entry's options.operations table, never hard-coded.
func OwlActionHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := merchantsb.OwlAction{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		expected := atlas_packet.ResolveCode(l, readerOptions, "operations", "OPEN")
		if p.Mode() != expected {
			l.Warnf("Character [%d] sent owl action with unexpected mode [%d], expected [%d].", s.CharacterId(), p.Mode(), expected)
			return
		}
		if !_map.IsFreeMarketRoom(s.MapId()) {
			l.Warnf("Character [%d] sent owl action outside the Free Market (map [%d]).", s.CharacterId(), s.MapId())
			return
		}
		_ = shopscanner.NewProcessor(l, ctx).SendHotList(wp)(s)
	}
}
```

`services/atlas-channel/atlas.com/channel/socket/handler/owl_warp.go`:

```go
package handler

import (
	"atlas-channel/character"
	"atlas-channel/merchant"
	"atlas-channel/portal"
	"atlas-channel/session"
	"atlas-channel/shopscanner"
	"atlas-channel/socket/writer"
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	merchantpkt "github.com/Chronicle20/atlas/libs/atlas-packet/merchant"
	merchantcb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound"
	merchantsb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// OwlWarpHandleFunc handles CUIShopScanResult::OnButtonClicked: re-validates
// the clicked result against current shop state (design §4.2 ladder), then
// warps same-channel and stages the pending auto-enter. Every failure rung
// answers with the faithful SHOP_LINK code; success sends no packet (the
// client tears the scanner windows down on field change).
func OwlWarpHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := merchantsb.OwlWarp{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		announceLink := func(code merchantpkt.ShopLinkResultCode) {
			_ = session.Announce(l)(ctx)(wp)(merchantcb.ShopLinkResultWriter)(writer.ShopLinkResultBody(code))(s)
		}

		check := shopscanner.WarpCheck{
			OwnerId:          p.OwnerId(),
			CharacterId:      s.CharacterId(),
			CurrentMapFM:     _map.IsFreeMarketRoom(s.MapId()),
			SessionWorldId:   s.WorldId(),
			SessionChannelId: s.ChannelId(),
			EchoedMapId:      p.MapId(),
		}

		reg := shopscanner.GetRegistry()
		last, hasSearch := reg.GetLastSearch(t, s.CharacterId())
		check.HasSearch = hasSearch

		c, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get character [%d] for owl warp.", s.CharacterId())
			announceLink(merchantpkt.ShopLinkResultCodeClosed)
			return
		}
		check.CharacterHp = c.Hp()

		mp := merchant.NewProcessor(l, ctx)
		var shopId uuid.UUID
		shops, err := mp.GetByCharacterId(p.OwnerId())
		if err == nil && len(shops) > 0 {
			shop := shops[0]
			check.ShopFound = true
			check.ShopWorldId = shop.WorldId()
			check.ShopChannelId = shop.ChannelId()
			check.ShopMapId = shop.MapId()
			check.ShopState = shop.State()
			shopId = shop.Id()
		}

		// Listing-still-present check: re-query the world-scoped search for the
		// remembered item and look for this shop with bundles remaining.
		if check.ShopFound && hasSearch {
			listings, err := mp.SearchListings(s.WorldId(), last.ItemId, false)
			if err != nil {
				l.WithError(err).Warnf("Unable to re-validate listing for owl warp of character [%d].", s.CharacterId())
			} else {
				for _, sl := range listings {
					if sl.ShopId() == shopId && sl.BundlesRemaining() > 0 {
						check.ListingPresent = true
						break
					}
				}
			}
		}

		if code, ok := shopscanner.EvaluateWarp(check); !ok {
			l.Infof("Owl warp rejected for character [%d] to owner [%d]: code [%s].", s.CharacterId(), p.OwnerId(), code)
			announceLink(code)
			return
		}

		reg.SetPending(t, s.CharacterId(), shopscanner.PendingEntry{
			ShopId:  shopId,
			OwnerId: p.OwnerId(),
			MapId:   _map.Id(p.MapId()),
		})
		l.Debugf("Character [%d] owl-warping to shop of owner [%d] in map [%d].", s.CharacterId(), p.OwnerId(), p.MapId())
		if err := portal.NewProcessor(l, ctx).Warp(s.Field(), s.CharacterId(), _map.Id(p.MapId())); err != nil {
			l.WithError(err).Errorf("Unable to warp character [%d] for owl warp.", s.CharacterId())
			reg.RemovePending(t, s.CharacterId())
			announceLink(merchantpkt.ShopLinkResultCodeClosed)
		}
	}
}
```

(Import `"github.com/google/uuid"` in owl_warp.go for the `shopId` declaration.)

`services/atlas-channel/atlas.com/channel/socket/handler/shop_scanner_item_use.go`:

```go
package handler

import (
	"atlas-channel/session"
	"atlas-channel/shopscanner"
	"atlas-channel/socket/writer"
	"context"

	character2 "atlas-channel/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	merchantsb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// ShopScannerItemUseHandleFunc handles the dedicated USE-inventory owl route
// (CWvsContext::SendShopScannerItemUseRequest, 231xxxx family double-clicked
// from the USE inventory). Validates the claimed item is a 231-family item
// actually present at the claimed slot before searching.
func ShopScannerItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := merchantsb.ShopScannerItemUse{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		itemId := item.Id(p.ItemId())
		if uint32(itemId)/10000 != 231 {
			l.Warnf("Character [%d] attempted shop scanner item use with non-scanner item [%d].", s.CharacterId(), itemId)
			return
		}

		a, err := character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), inventory.TypeValueUse, p.Source())()
		if err != nil || item.Id(a.TemplateId()) != itemId {
			l.Warnf("Character [%d] attempted to use scanner item [%d] in slot [%d], but item not found or mismatched.", s.CharacterId(), itemId, p.Source())
			return
		}

		_ = shopscanner.NewProcessor(l, ctx).Search(wp)(s, p.SearchItemId(), p.Descending(), itemId, slot.Position(p.Source()), p.UpdateTime())
	}
}
```

(Match the import alias for the character processor to whatever `character_cash_item_use.go` uses — it aliases `atlas-channel/character` as `character2`.)

- [ ] **Step 3: Register in main.go**

In `produceHandlers()` (next to `handlerMap[merchantsb.HiredMerchantOperationHandle]` at line ~899):

```go
	handlerMap[merchantsb.OwlActionHandle] = handler.OwlActionHandleFunc
	handlerMap[merchantsb.OwlWarpHandle] = handler.OwlWarpHandleFunc
	handlerMap[merchantsb.ShopScannerItemUseHandle] = handler.ShopScannerItemUseHandleFunc
```

In `produceWriters()` (next to `merchantcb.HiredMerchantOperationWriter` at line ~781):

```go
		merchantcb.ShopScannerResultWriter,
		merchantcb.ShopLinkResultWriter,
```

- [ ] **Step 4: Build + test the whole service**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./... -count=1 && go vet ./...`
Expected: clean build, all tests PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/ services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(task-127): owl socket handlers (action, warp, dedicated use) and cash 523 arm"
```

---

### Task 12: atlas-channel — warp arrival auto-enter, capacity-full owl branch, session cleanup

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go` (`warpCharacter`, lines ~237-272)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go` (`handleCapacityFullEvent` ~283-298; `handleVisitorEvent` VisitorEntered case ~189)
- Modify: `services/atlas-channel/atlas.com/channel/socket/init.go` (destroyer, line ~46)

**Interfaces:**
- Consumes: Tasks 8, 9 outputs; `merchant.NewProcessor(l, ctx).EnterShop(characterId uint32, shopId uuid.UUID) error` (existing); `session.Announce`.
- Produces: the complete warp→enter→outcome loop. Registry entry lifecycle: `SetPending` (Task 11, warp accepted) → kept through EnterShop emission → removed on VisitorEntered (success), CapacityFull (code 2 announced), arrival-map mismatch, or session destroy.

- [ ] **Step 1: Auto-enter on map arrival**

In `kafka/consumer/character/consumer.go`, inside `warpCharacter`'s returned operator, after the `SpawnForSelf` call and before `return nil` (line ~268):

```go
				// Owl warp auto-enter (task-127): if this arrival completes a
				// pending shop-scanner warp, enter the shop as a visitor. The
				// entry stays pending until VisitorEntered or CapacityFull.
				reg := shopscanner.GetRegistry()
				if pe, ok := reg.GetPending(tenant.MustFromContext(ctx), s.CharacterId()); ok {
					if pe.MapId == event.Body.TargetMapId {
						if err := merchant.NewProcessor(l, ctx).EnterShop(s.CharacterId(), pe.ShopId); err != nil {
							l.WithError(err).Errorf("Unable to auto-enter shop [%s] for character [%d] after owl warp.", pe.ShopId, s.CharacterId())
							reg.RemovePending(tenant.MustFromContext(ctx), s.CharacterId())
						}
					} else {
						reg.RemovePending(tenant.MustFromContext(ctx), s.CharacterId())
					}
				}
```

Add imports `"atlas-channel/shopscanner"`, `"atlas-channel/merchant"`, and the tenant lib if not already present in the file (check the existing import block — `tenant` is already imported for `sc.Is(tenant.MustFromContext(ctx), ...)`).

- [ ] **Step 2: Capacity-full owl branch**

In `kafka/consumer/merchant/consumer.go`, `handleCapacityFullEvent` — after the tenant gate and debug log, before the existing interaction-error announce:

```go
		reg := shopscanner.GetRegistry()
		if _, ok := reg.GetPending(t, e.CharacterId); ok {
			// Owl warp arrival hit a full shop: answer with the faithful
			// SHOP_LINK code 2 instead of the mini-room error (task-127).
			reg.RemovePending(t, e.CharacterId)
			_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(merchantcb.ShopLinkResultWriter)(writer.ShopLinkResultBody(merchantpkt.ShopLinkResultCodeFull)))
			return
		}
```

The existing `CharacterInteractionEnterResultErrorBody(...ModeFull)` announce stays as the non-owl fallthrough. Add imports: `"atlas-channel/shopscanner"`, `"atlas-channel/socket/writer"` (check alias — the file already imports the writer package for `wp writer.Producer`), `merchantpkt "github.com/Chronicle20/atlas/libs/atlas-packet/merchant"`, `merchantcb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound"`.

- [ ] **Step 3: Consume the pending entry on successful visit**

In `handleVisitorEvent`, at the top of the `case merchant2.StatusEventVisitorEntered:` branch (line ~189):

```go
			// A completed visit consumes any pending owl-warp entry (task-127).
			shopscanner.GetRegistry().RemovePending(t, e.Body.CharacterId)
```

- [ ] **Step 4: Session-destroy cleanup**

In `socket/init.go`, replace `socket.SetDestroyer(sp.DestroyByIdWithSpan),` (line ~46) with a wrapper that clears scanner state first (the session package must not import shopscanner — this bootstrap file may import both):

```go
					socket.SetDestroyer(func(sessionId uuid.UUID) {
						_ = sp.IfPresentById(sessionId, func(s session.Model) error {
							shopscanner.GetRegistry().ClearCharacter(t, s.CharacterId())
							return nil
						})
						sp.DestroyByIdWithSpan(sessionId)
					}),
```

`t` is already in scope (`t := sc.Tenant()`, line ~27). Add imports `"atlas-channel/shopscanner"` and `"github.com/google/uuid"`.

- [ ] **Step 5: Build + test + commit**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./... -count=1 && go vet ./...`
Expected: clean.

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/ services/atlas-channel/atlas.com/channel/socket/init.go
git commit -m "feat(task-127): owl warp auto-enter on arrival, capacity-full SHOP_LINK branch, session cleanup"
```

---

### Task 13: Seed templates — handlers, writers, operations tables for all versions

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`

(`template_gms_12_1.json` is intentionally excluded — no owl support planned for v12.)

**Interfaces:**
- Consumes: handler/writer name strings from Tasks 2, 3, 11; opcodes from the Global Constraints matrix.
- Produces: per-version socket routing. **Every handler entry carries `"validator": "LoggedInValidator"`** — a validator-less entry is silently dropped by `BuildHandlerMap`.

- [ ] **Step 1: Add handler entries**

Append to each template's `socket.handlers` array (keep the file's existing entry formatting; opCode values per version below):

gms_83 (`OwlAction 0x42`, `OwlWarp 0x43`, `ShopScannerItemUse 0x53`):

```json
    {"opCode": "0x42", "validator": "LoggedInValidator", "handler": "OwlActionHandle", "options": {"operations": {"OPEN": 5}}},
    {"opCode": "0x43", "validator": "LoggedInValidator", "handler": "OwlWarpHandle"},
    {"opCode": "0x53", "validator": "LoggedInValidator", "handler": "ShopScannerItemUseHandle"}
```

gms_84: identical to gms_83 (serverbound v84 ≡ v83): `0x42` / `0x43` / `0x53`.

gms_87 (`USE_SHOP_SCANNER_ITEM` opcode unknown — not routed):

```json
    {"opCode": "0x45", "validator": "LoggedInValidator", "handler": "OwlActionHandle", "options": {"operations": {"OPEN": 5}}},
    {"opCode": "0x46", "validator": "LoggedInValidator", "handler": "OwlWarpHandle"}
```

gms_92 (dedicated route unknown — not routed):

```json
    {"opCode": "0x49", "validator": "LoggedInValidator", "handler": "OwlActionHandle", "options": {"operations": {"OPEN": 5}}},
    {"opCode": "0x4A", "validator": "LoggedInValidator", "handler": "OwlWarpHandle"}
```

gms_95:

```json
    {"opCode": "0x48", "validator": "LoggedInValidator", "handler": "OwlActionHandle", "options": {"operations": {"OPEN": 5}}},
    {"opCode": "0x49", "validator": "LoggedInValidator", "handler": "OwlWarpHandle"},
    {"opCode": "0x5A", "validator": "LoggedInValidator", "handler": "ShopScannerItemUseHandle"}
```

jms_185 (dedicated route unknown — not routed):

```json
    {"opCode": "0x3A", "validator": "LoggedInValidator", "handler": "OwlActionHandle", "options": {"operations": {"OPEN": 5}}},
    {"opCode": "0x3B", "validator": "LoggedInValidator", "handler": "OwlWarpHandle"}
```

Before adding each entry, grep the template for the opCode being claimed (e.g. `"opCode": "0x42"` in the handlers array) — if another handler already occupies it, STOP and report the conflict rather than double-routing (registry says these serverbound slots are the owl ops, so a collision means a template bug to surface, not overwrite).

- [ ] **Step 2: Add writer entries**

Append to each template's `socket.writers` array (shape modeled on the `WorldMessage` entry at `template_gms_83_1.json:1635-1661`). The `operations` tables are version-stable today but config-driven regardless (dispatcher-config-drive-all-modes rule):

gms_83:

```json
    {
      "opCode": "0x46",
      "writer": "ShopScannerResult",
      "options": {"operations": {"RESULT": 6, "HOT_LIST": 7}}
    },
    {
      "opCode": "0x47",
      "writer": "ShopLinkResult",
      "options": {"operations": {"SUCCESS": 0, "CLOSED": 1, "FULL": 2, "BUSY": 3, "DEAD": 4, "NO_TRADE": 7, "DENIED": 17, "MAINTENANCE": 18, "FM_ONLY": 23}}
    }
```

Same two entries in the other templates with only `opCode` changing:

| template | ShopScannerResult | ShopLinkResult |
|---|---|---|
| gms_84 | `0x48` | `0x49` |
| gms_87 | `0x48` | `0x49` |
| gms_92 | `0x4A` | `0x4B` |
| gms_95 | `0x49` | `0x4A` |
| jms_185 | `0x40` | `0x41` |

Apply the same collision check on clientbound opCodes within each template's writers array before adding.

- [ ] **Step 3: Validate JSON + commit**

Run: `for f in services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_92,gms_95,jms_185}_1.json; do python3 -m json.tool "$f" > /dev/null && echo "OK $f"; done`
Expected: `OK` for all six files.

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(task-127): seed-template routing for owl ops across gms_83/84/87/92/95 and jms_185"
```

---

### Task 14: Packet registry corrections + packet-audit fname mapping

**Files:**
- Modify: `docs/packets/registry/gms_v83.yaml` (delete USE_SKILL_RESET_BOOK row ~line 2217; add USE_SHOP_SCANNER_ITEM row)
- Modify: `docs/packets/registry/gms_v84.yaml` (delete USE_SKILL_RESET_BOOK row ~line 2886; add USE_SHOP_SCANNER_ITEM row)
- Modify: `tools/packet-audit/cmd/run.go` (`candidatesFromFName`, switch starting ~line 278)

**Interfaces:**
- Consumes: design §1.6 (IDA-proven registry conflict resolution).
- Produces: corrected registry rows; fname→codec candidates the audit/verify pipeline (Task 15) links through.

**Coordination note:** the USE_SKILL_RESET_BOOK removal touches rows in-flight task-125 (skill-mastery-books) may be reading. The correction is IDA-proven either way (no skill-reset-book sender exists in the v83 binary; the only `COutPacket(0x53)` construction site is `CWvsContext::SendShopScannerItemUseRequest` @0xa0a25e), but flag it in the PR description and check `.worktrees/` for a task-125 worktree before landing.

- [ ] **Step 1: Registry — gms_v83.yaml**

Delete this row block (at ~line 2217):

```yaml
- op: USE_SKILL_RESET_BOOK
  direction: serverbound
  opcode: 83
  fname: CWvsContext::SendSkillResetItemUseRequest
  provenance: csv-import
```

Add, in opcode order among the serverbound rows:

```yaml
- op: USE_SHOP_SCANNER_ITEM
  direction: serverbound
  opcode: 83
  fname: CWvsContext::SendShopScannerItemUseRequest
  provenance: ida-discovered
  note: task-127 — full COutPacket(0x53) construction-site scan of the v83 binary found exactly one sender, SendShopScannerItemUseRequest (0xa0a25e); no skill-reset-book sender exists in v83 (func_query for SkillReset/resetbook = zero hits). Supersedes the csv-import USE_SKILL_RESET_BOOK row, which also wrongly recorded USE_SHOP_SCANNER_ITEM as 0x000.
```

- [ ] **Step 2: Registry — gms_v84.yaml**

Delete the corresponding row block (at ~line 2886, includes its existing `note:` line):

```yaml
- op: USE_SKILL_RESET_BOOK
  direction: serverbound
  opcode: 83
  fname: CWvsContext::SendSkillResetItemUseRequest
  provenance: csv-import
  note: seeded from the v83 CSV column — the CSVs have no v84 column; task-083 found v84 byte-identical to v83. Corrected by discover-ops against the v84 IDB.
```

Add:

```yaml
- op: USE_SHOP_SCANNER_ITEM
  direction: serverbound
  opcode: 83
  fname: CWvsContext::SendShopScannerItemUseRequest
  provenance: ida-discovered
  note: task-127 — inherited from the v83 IDA finding (0xa0a25e) via the established v84-serverbound≡v83 rule; UNVERIFIED against a v84 IDB (none loaded). See gms_v83.yaml for the primary evidence.
```

- [ ] **Step 3: candidatesFromFName**

In `tools/packet-audit/cmd/run.go`, add a new case group to the `candidatesFromFName` switch (near the existing merchant bucket at ~line 602):

```go
	// --- shop scanner / owl (task-127) ---
	case "CUIShopScanner::OnCreate":
		return []candidate{{name: "OwlAction", pkg: "merchant", dir: csvpkg.DirServerbound}}
	case "CUIShopScanResult::OnButtonClicked":
		return []candidate{{name: "OwlWarp", pkg: "merchant", dir: csvpkg.DirServerbound}}
	case "CWvsContext::SendShopScannerItemUseRequest":
		return []candidate{{name: "ShopScannerItemUse", pkg: "merchant", dir: csvpkg.DirServerbound}}
	case "CWvsContext::OnShopScannerResult#Result":
		return []candidate{{name: "ShopScannerResult", pkg: "merchant", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnShopScannerResult#HotList":
		return []candidate{{name: "ShopScannerHotList", pkg: "merchant", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnShopLinkResult":
		return []candidate{{name: "ShopLinkResult", pkg: "merchant", dir: csvpkg.DirClientbound}}
```

- [ ] **Step 4: Verify the tool builds and commit**

Run: `cd tools/packet-audit && go build ./... && go test ./... -count=1`
Expected: clean. (Full matrix regeneration happens in Task 15 after the exports/evidence exist — a lone regen here would report the owl rows as missing evidence.)

```bash
git add docs/packets/registry/gms_v83.yaml docs/packets/registry/gms_v84.yaml tools/packet-audit/cmd/run.go
git commit -m "fix(task-127): v83/v84 serverbound 0x53 is USE_SHOP_SCANNER_ITEM (IDA-proven), map owl fnames in packet-audit"
```

---

### Task 15: Packet verification campaign — gms_v83 + gms_v95 tier-1 cells

**Files:**
- Modify: `docs/packets/ida-exports/gms_v83.json` (surgical splice — never overwrite)
- Modify: `docs/packets/ida-exports/gms_v95.json` (surgical splice)
- Create: `docs/packets/evidence/gms_v83/merchant.serverbound.OwlAction.yaml` (+ 5 siblings per packet)
- Create: `docs/packets/evidence/gms_v95/merchant.serverbound.OwlAction.yaml` (+ 5 siblings per packet)
- Modify: `docs/packets/audits/STATUS.md` + `status.json` (regenerated)

**Interfaces:**
- Consumes: Task 2/3 fixture tests (markers already in place), Task 14 fname mapping, live IDA instances.
- Produces: promoted coverage-matrix cells for the six packets × {gms_v83, gms_v95}; other versions stay seed-routed-but-unverified (the accepted state for v84/v87/v92/jms gaps, design §5).

This task follows `docs/packets/audits/VERIFYING_A_PACKET.md` — read it first and treat it as authoritative over this summary. Prefer dispatching the `packet-verifier` agent per packet × version cell (batch per IDB; serialize within one IDB).

**Hard rules (from project memory):**
- ALWAYS run `list_instances` first and match the binary NAME — the instance set rotates. Expected: v83 dump on port 13342 (`MapleStory_dump.exe.i64`), v95 on 13341 (`GMS_v95.0_U_DEVM.exe.i64`). If either is missing, STOP and report BLOCKED — do not substitute.
- The export splice is NON-idempotent: add only the new function entries; never regenerate/overwrite the file; strip any COutPacket-delegate harvest artifacts.
- An op whose fname does not resolve is a stop-and-ask — never fake a hash or substitute an fname.

**Fname/address worklist (from design §1, all pre-verified in the live IDBs during design):**

| fname | v83 addr | v95 addr | direction |
|---|---|---|---|
| `CUIShopScanner::OnCreate` | 0x8a0e9a | 0x848b90 | serverbound |
| `CUIShopScanResult::OnButtonClicked` | 0x8a4423 (v83 IDB may still name it `sub_8A4423` — name it first, per the no-deferring rule) | 0x848e80 | serverbound |
| `CWvsContext::SendShopScannerItemUseRequest` | 0xa0a25e | 0x9e10e0 | serverbound |
| `CWvsContext::OnShopScannerResult` (arms `#Result`, `#HotList`) | 0xa28c29 | 0xa076c0 | clientbound |
| `CWvsContext::OnShopLinkResult` | 0x8a4e7a | 0x847d60 | clientbound |

- [ ] **Step 1:** For each version (v83 then v95): `list_instances`, `select_instance(port)`, decompile each fname at the listed address, derive the ordered Decode/Encode call list, and splice the function entries (with `direction`, `calls[]`, and `dispatch[]` for the two `#`-arm entries of OnShopScannerResult) into the matching `docs/packets/ida-exports/<version>.json`.
- [ ] **Step 2:** Cross-check each decompiled read order against the Task 2/3 codec layouts. Any mismatch is a blocker: fix the codec + fixture FIRST (amending the Task 2/3 work), then continue. Do not pin evidence over a known mismatch.
- [ ] **Step 3:** Write the evidence records — one YAML per packet × version at `docs/packets/evidence/<version>/merchant.<direction>.<PacketName>.yaml`, `category: TIER1-FIXTURE`, with `ida.function`, `ida.address`, and `ida.decompile_sha256` computed per the playbook. Packets: `OwlAction`, `OwlWarp`, `ShopScannerItemUse` (serverbound), `ShopScannerResult`, `ShopScannerHotList`, `ShopLinkResult` (clientbound) — 12 records total.
- [ ] **Step 4:** Generate the serverbound audit REPORTs per the playbook (report generation via the root `-ida-source` flag; the Task 14 `candidatesFromFName` cases link them by primary fname).
- [ ] **Step 5:** Regenerate the matrix and check:

Run: `cd tools/packet-audit && go run . matrix --check` (exact invocation per the tool's README — `packet-audit matrix` with the default `--audits-dir docs/packets/audits`; run from the repo root if the README says so).
Expected: exit 0, no conflicts; the six owl ops show verified cells for gms_v83/gms_v95.

- [ ] **Step 6:** Commit the three artifact groups together (fixtures landed in Tasks 2-3; exports + evidence + regenerated matrix land here):

```bash
git add docs/packets/ida-exports/gms_v83.json docs/packets/ida-exports/gms_v95.json docs/packets/evidence/gms_v83/ docs/packets/evidence/gms_v95/ docs/packets/audits/
git commit -m "verify(task-127): tier-1 evidence for owl packets on gms_v83 and gms_v95, matrix regen"
```

---

### Task 16: Full verification gates + deployment notes

**Files:**
- Create: `docs/tasks/task-127-owl-shop-search/deployment.md`

- [ ] **Step 1: Run every gate, in order, from the worktree root**

```bash
# per-module tests/vet/build
(cd libs/atlas-constants && go test -race ./... -count=1 && go vet ./...)
(cd libs/atlas-packet && go test -race ./... -count=1 && go vet ./...)
(cd services/atlas-merchant/atlas.com/merchant && go test -race ./... -count=1 && go vet ./... && go build ./...)
(cd services/atlas-channel/atlas.com/channel && go test -race ./... -count=1 && go vet ./... && go build ./...)
(cd tools/packet-audit && go test ./... -count=1 && go build ./...)

# repo guards (no GOWORK=off prefix — false FAILs)
tools/redis-key-guard.sh

# docker bake — mandatory, catches Dockerfile COPY gaps go build cannot
docker buildx bake atlas-channel atlas-merchant atlas-configurations

# packet matrix
(cd tools/packet-audit && go run . matrix --check)
```

Expected: every command exits 0. Report actual output, not assumptions. If bake fails on a missing lib COPY line: the shared `Dockerfile` needs no change for this task (no new lib was created — atlas-constants and atlas-packet already have COPY lines), so a failure means a real code issue.

- [ ] **Step 2: Verify the 231-family owl exists in v83 WZ data (design §9 risk)**

Ground the dedicated route's player-reachability against local data, not memory: locate the local WZ extract's `Item.wz/Consume/0231.img.xml` (or the atlas-data ingested equivalent) and check whether item 2310000 exists for the v83 data set. Record the outcome in deployment.md: if present, the dedicated route is player-reachable on v83; if absent, note that the route stays implemented + fixture-verified at packet level and the cash owl (5230000) is the player path. Either outcome is acceptable — an unrecorded assumption is not.

- [ ] **Step 3: Write deployment.md**

Create `docs/tasks/task-127-owl-shop-search/deployment.md`:

```markdown
# task-127 deployment notes — Owl of Minerva

## Live-tenant config patch (REQUIRED)

Seed templates apply only at tenant creation. Existing tenants MUST be
patched or the owl ops are silently dropped (unhandled op / missing writer):

1. For each live tenant, PATCH its socket configuration with the same
   entries Task 13 added to its version's template:
   - handlers: OwlActionHandle (with options.operations {OPEN:5}),
     OwlWarpHandle, ShopScannerItemUseHandle (gms_83/84/95 only) — each with
     LoggedInValidator. Opcodes per version: see plan.md Global Constraints
     matrix.
   - writers: ShopScannerResult (operations {RESULT:6, HOT_LIST:7}),
     ShopLinkResult (operations {SUCCESS:0, CLOSED:1, FULL:2, BUSY:3, DEAD:4,
     NO_TRADE:7, DENIED:17, MAINTENANCE:18, FM_ONLY:23}).
2. Restart atlas-channel after patching — the handler/writer projection does
   not hot-reload.

## Rollout order

1. atlas-merchant (schema migration `listing_search_counts` is additive;
   new command type is ignored by old channel pods).
2. atlas-channel.
3. Tenant config patch + atlas-channel restart (step above).

## In-game acceptance pass (v83 tenant, per PRD §10)

- Search with results (owner/title/price/quantity/channel correct; owl
  consumed exactly 1).
- Empty search ("Unable to find..." message; owl NOT consumed).
- Hot list on scanner open (top-10 by count; survives service restart).
- Warp to open shop -> auto-enter as visitor.
- Full shop -> "full capacity" (SHOP_LINK code 2).
- Maintenance shop -> code 18. Sold-out race -> code 3. Own shop -> code 17.
- Cross-channel row shows channel number, no warp link (client-side).
```

- [ ] **Step 4: Code review before PR (mandatory)**

Invoke `superpowers:requesting-code-review` — dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer` (Go-only change set). Address findings before opening the PR. Flag the task-125 registry-row coordination (Task 14) in the PR description.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-127-owl-shop-search/deployment.md
git commit -m "docs(task-127): deployment notes (live-tenant patch, rollout order, acceptance pass)"
```

---

## Execution order & dependencies

```
Task 1 (constants) ──┐
Task 2 (sb codecs) ──┼─→ Task 11 (handlers) ─→ Task 12 (consumers) ─→ Task 13 (templates)
Task 3 (cb codecs) ──┤        ↑                                             │
Task 4 (search)    ──┼─→ Task 7 (channel client) ─→ Task 9 (writers) ──────┤
Task 5 (counts)    ──┤        │                          ↑                  │
Task 6 (command)   ──┘        └─→ Task 10 (processor) ───┘                  │
Task 8 (registry) ────────────────↑                                         │
Task 14 (registry/audit fnames) ─→ Task 15 (verify campaign) ←──────────────┘
                                        │
                                        └─→ Task 16 (gates + deployment + review)
```

Tasks 1-6 and 8 are independent of each other. Task 15 requires live IDA instances (ports may rotate — `list_instances` first) and is the only task with an external dependency; if the IDBs are unavailable, finish everything else and report Task 15 BLOCKED rather than substituting evidence.




