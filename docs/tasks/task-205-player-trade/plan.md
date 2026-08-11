# Player-to-Player Trade Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship 1:1 player trading — a new `atlas-trades` service owning room lifecycle and settlement, the missing clientbound codecs across ten client versions, and an atomic saga-backed swap with a tenant-configurable meso tax and a durable ledger.

**Architecture:** atlas-channel decodes the wire and forwards typed commands on `COMMAND_TOPIC_TRADE`; atlas-trades owns an in-memory tenant-partitioned room registry plus a durable ledger, and drives the swap through one `trade_settlement` saga composite that the orchestrator expands into `release_from_character` / `accept_to_character` / `award_mesos` steps. Staged items are **reserved** in atlas-inventory (not escrowed) and only move at settlement. atlas-trades never writes a packet; atlas-channel never mutates inventory or meso.

> **Amended by Slice 7 (Tasks 25-31).** Reserve-at-staging is wire-incompatible with the reference client and is replaced by **escrow at staging**: a staged item genuinely leaves the owner's compartment into a durable `trade_escrow_items` store in atlas-trades via new `accept_to_trade` / `release_from_trade` saga actions, and staged meso is genuinely debited. See design §5A. Everything else in this paragraph stands — atlas-trades still writes no packet, atlas-channel still mutates nothing.

**Tech Stack:** Go 1.24 (multi-module `go.work`), GORM + Postgres, segmentio kafka via `libs/atlas-kafka`, JSON:API via api2go, `libs/atlas-packet` codecs, `libs/atlas-saga` orchestration, `libs/atlas-routine` goroutines.

**Read first:** [`context.md`](context.md) — every "what already exists" fact this plan builds on, with `file:line` citations. [`design.md`](design.md) §§1-15 is the authority for every behavioural decision; do not re-derive them.

## Global Constraints

- **Worktree.** All work happens in `.worktrees/task-205-player-trade` on branch `task-205-player-trade`. Never edit the main repo.
- **No hard-coded client-facing bytes (DOM-25).** Every trade mode byte, enter-error code and leave status resolves through `atlas_packet.WithResolvedCode("operations"|"enterError"|"leaveReason", KEY, …)`. No numeric literal for a wire value in Go.
- **Version gates use `MajorAtLeast`.** `t.MajorAtLeast(83)`, never `t.MajorVersion() > 82`. The one exception is the existing `tradeCrcPresent` (`libs/atlas-packet/interaction/serverbound/version_gate.go`), which is reused unchanged.
- **Multi-tenancy.** `tenant.MustFromContext(ctx)` everywhere; every registry map tenant-keyed; every DB query `tenant_id`-scoped.
- **Goroutines.** Every goroutine via `routine.Go` (`tools/goroutine-guard.sh`). Redis, if any, through `libs/atlas-redis` (`tools/redis-key-guard.sh`).
- **Immutable domain models.** Private fields + getters + Builder. No `*_testhelpers.go` — test setup uses the Builder.
- **Shared types first.** Before defining any id/type/constant, check `libs/atlas-constants/` (DOM-21). Use `character.Id`, `world.Id` (byte), `channel.Id` (byte), `_map.Id` (uint32), `item.Id`, `inventory.Type`, `miniroom.*`.
- **No TODOs, stubs or 501s in landed commits.**
- **Matrix promotion is machine-checked.** A cell that does not promote in `docs/packets/audits/STATUS.md` is a failure, never a prose claim. Round-trip-only fixtures do not count as verification.
- **Ten versions in scope:** gms_v48, gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, jms_v185. `template_gms_12_1.json` is explicitly out of scope.
- **Default tax tiers (design §8):** `>=100,000,000` 6.0%; `>=25,000,000` 5.0%; `>=10,000,000` 4.0%; `>=5,000,000` 3.0%; `>=1,000,000` 1.8%; `>=100,000` 0.8%; below 100,000 0%. `delivered = m - floor(m * rate(m))`; the difference is destroyed.
- **Other config defaults:** `maxStagedItems` 9, `minTradeLevel` 0, `reservationTtlSeconds` 300, `attestationTimeoutSeconds` 5, `taxEnabled` true. *(Slice 7 removes `reservationTtlSeconds` and adds `stageTimeoutSeconds` 5.)*
- **Never leave the client's action lock armed (Slice 7).** `CTradingRoomDlg::PutItem` and `::PutMoney` set `m_bExclRequestSent` on send and gate on `CWvsContext::CanSendExclRequest`. Every stage outcome — success, restriction refusal, freeze-rule drop, meso refusal, saga failure, timeout — must result in a packet whose leading `exclRequestSent` bool is true reaching the acting client. A stage that emits nothing is a permanently wedged dialog, not a silent drop.
- **Leave statuses (design §1.4):** 2 cancelled, 7 success, 8 failed, 9 cannot-carry, 12 different-map, 13 CRC-failed.
- **Commit after every task.** Conventional-commit subjects prefixed `task-205`.

---

# Slice 1 — Packet layer (Tasks 1-6)

Independent of every service change. Land this slice first; nothing downstream compiles against it except atlas-channel (Slice 5).

### Task 1: `NewTradeRoom` constructor

**Files:**
- Modify: `libs/atlas-packet/interaction/room.go:52-58` (doc comment), append constructor after `NewMerchantShopRoom` (`room.go:117`)
- Test: `libs/atlas-packet/interaction/room_trade_test.go` (create)

**Interfaces:**
- Consumes: `interaction.RoomType` (`room.go:15-28`), `interaction.Visitor` + `NewBaseVisitor(slot byte, avatar model.Avatar, name string) Visitor` (`visitor.go:43`).
- Produces: `interaction.NewTradeRoom(roomType RoomType, position byte, visitors []Visitor) Room` — consumed by Task 21 (atlas-channel) via `clientbound.CharacterInteractionEnterResultSuccessBody(room)`.

Design §1.3 and §4.1: the trade room's enter-result body is **exactly** the base
frame. `Room.Encode` already falls through for `TradeRoomType`/`CashTradeRoomType`
writing nothing after the `0xFF`, and `decodeVisitorForRoom`'s `default` arm
already handles trade visitors. So this task adds a constructor and a pinning
test — no encoder change.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/interaction/room_trade_test.go`:

```go
package interaction

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestNewTradeRoomIsBaseFrameOnly pins design §1.3: CTradingRoomDlg's
// enter-result tail virtual (v83 vtable off_B37448 slot +72 -> nullsub_94
// @0x48314D) is EMPTY, so the trade room body is exactly the
// CMiniRoomBaseDlg::OnEnterResultBase frame — roomType, capacity(2), position,
// {slot, avatar, name} visitors, 0xFF — and nothing follows.
func TestNewTradeRoomIsBaseFrameOnly(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)

	rm := NewTradeRoom(TradeRoomType, 1, nil)
	if rm.RoomType() != TradeRoomType {
		t.Fatalf("roomType: got %v, want %v", rm.RoomType(), TradeRoomType)
	}
	if rm.Capacity() != 2 {
		t.Fatalf("capacity: got %d, want 2", rm.Capacity())
	}
	if rm.Position() != 1 {
		t.Fatalf("position: got %d, want 1", rm.Position())
	}

	got := hex.EncodeToString(rm.Encode(l, ctx)(nil))
	// roomType(03) capacity(02) position(01) terminator(FF), nothing after.
	if got != "030201ff" {
		t.Fatalf("empty trade room bytes: got %s, want 030201ff", got)
	}
}

func TestNewTradeRoomCashVariant(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)

	rm := NewTradeRoom(CashTradeRoomType, 0, nil)
	got := hex.EncodeToString(rm.Encode(l, ctx)(nil))
	if got != "060200ff" {
		t.Fatalf("empty cash trade room bytes: got %s, want 060200ff", got)
	}
}

// TestNewTradeRoomRoundTripsVisitors proves the visitor list survives the
// base-frame encode/decode via decodeVisitorForRoom's default arm
// (visitor.go:106), which already covers Trade and CashTrade.
func TestNewTradeRoomRoundTripsVisitors(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)

	visitors := []Visitor{NewBaseVisitor(1, model.Avatar{}, "Partner")}
	in := NewTradeRoom(TradeRoomType, 1, visitors)

	out := Room{}
	pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd libs/atlas-packet && go test ./interaction/ -run TestNewTradeRoom -v`
Expected: FAIL — `undefined: NewTradeRoom`.

- [ ] **Step 3: Add the constructor**

Append after `NewMerchantShopRoom` in `libs/atlas-packet/interaction/room.go`:

```go
// NewTradeRoom builds a trade (roomType 3) or cash-trade (roomType 6)
// enter-result room. position is the recipient's position in the room —
// 0 = owner, 1 = the invited character — landing in the
// CMiniRoomBaseDlg::OnEnterResultBase second header byte (v83 @0x65ec6b ->
// *(this+0xC8)).
//
// CTradingRoomDlg's enter-result tail virtual (vtable+72; v83 off_B37448+0x48
// -> nullsub_94 @0x48314D) is EMPTY, so the trade room's body is exactly the
// base frame — roomType, capacity(2), position, {slot, avatar, name} visitors,
// 0xFF. Nothing follows. Room.Encode's switch therefore has no trade arm by
// design, not by omission (task-205 design.md §1.3, §4.1).
func NewTradeRoom(roomType RoomType, position byte, visitors []Visitor) Room {
	return Room{
		roomType: roomType,
		capacity: 2,
		position: position,
		visitors: visitors,
	}
}
```

- [ ] **Step 4: Update the `Room` doc comment**

Replace `libs/atlas-packet/interaction/room.go:52-58` with:

```go
// Room models the shop-family (personal shop / hired merchant) and trade-family
// (trade / cash trade) EnterResultSuccess bodies. The trade family adds no tail
// after the base frame (see NewTradeRoom). Game rooms (Omok / Match Cards) are
// NOT modelled here: their room-enter blob has a different layout (yourSlot byte
// after capacity; avatars and 20-byte records in two SEPARATE 0xFF-terminated
// lists) and lives in clientbound.InteractionMiniGameRoom (IDA-derived;
// ida-notes.md §G5 "Room-enter blob — FULL RESOLUTION").
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./interaction/... -v`
Expected: PASS, including the pre-existing shop room tests.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/interaction/room.go libs/atlas-packet/interaction/room_trade_test.go
git commit -m "feat(task-205): add interaction.NewTradeRoom (base-frame-only enter result)"
```

---

### Task 2: `TRADE_PUT_ITEM` clientbound codec

**Files:**
- Create: `libs/atlas-packet/interaction/clientbound/interaction_trade.go`
- Modify: `libs/atlas-packet/interaction/clientbound/interaction_body.go` (mode key const + body constructor)
- Test: `libs/atlas-packet/interaction/clientbound/interaction_trade_test.go` (create)

**Interfaces:**
- Consumes: `model.Asset` (`libs/atlas-packet/model/asset.go:63`, pointer-receiver `Encode`/`Decode`), `atlas_packet.WithResolvedCode` (`libs/atlas-packet/resolve.go:15`).
- Produces:
  - `clientbound.CharacterInteractionModeTradePutItem CharacterInteractionMode = "TRADE_PUT_ITEM"`
  - `clientbound.NewInteractionTradePutItem(mode byte, side byte, tradeSlot byte, a model.Asset) InteractionTradePutItem`
  - `clientbound.CharacterInteractionTradePutItemBody(side byte, tradeSlot byte, a model.Asset) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte` — consumed by Task 22.

Design §4.2: mode 15 on v83, body `side:byte, tradeSlot:byte, Asset`, authority
`sub_7C1FB7` @`0x7c1fb7`.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/interaction/clientbound/interaction_trade_test.go`:

```go
package clientbound

import (
	"encoding/hex"
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestInteractionTradePutItemRoundTrip pins the mode-15 arm: Decode1 side,
// Decode1 trade slot, then GW_ItemSlotBase (v83 sub_7C1FB7 @0x7c1fb7).
func TestInteractionTradePutItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			a := model.NewAsset(true, 0, 2000000, time.Time{}).SetStackableInfo(50, 0, 0)
			input := NewInteractionTradePutItem(15, 1, 3, a)

			l, _ := testlog.NewNullLogger()
			raw := input.Encode(l, ctx)(nil)
			if len(raw) < 3 {
				t.Fatalf("body too short: %d bytes", len(raw))
			}
			if raw[0] != 15 {
				t.Errorf("mode: got %d, want 15", raw[0])
			}
			if raw[1] != 1 {
				t.Errorf("side: got %d, want 1", raw[1])
			}
			if raw[2] != 3 {
				t.Errorf("tradeSlot: got %d, want 3", raw[2])
			}
		})
	}
}

// TestInteractionTradePutItemHeaderBytes pins the fixed three-byte header ahead
// of the asset blob so a field reorder is caught independently of asset codec
// churn.
func TestInteractionTradePutItemHeaderBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	a := model.NewAsset(true, 0, 2000000, time.Time{}).SetStackableInfo(50, 0, 0)
	raw := NewInteractionTradePutItem(15, 0, 1, a).Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if got := hex.EncodeToString(raw[:3]); got != "0f0001" {
		t.Errorf("header bytes: got %s, want 0f0001", got)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd libs/atlas-packet && go test ./interaction/clientbound/ -run TestInteractionTradePutItem -v`
Expected: FAIL — `undefined: NewInteractionTradePutItem`.

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/interaction/clientbound/interaction_trade.go`:

```go
// Package clientbound — trade-family (CTradingRoomDlg / CCashTradingRoomDlg)
// mode arms. Kept out of interaction_body.go, which is already at the size
// limit for a single family file (task-205 design.md §4.2).
package clientbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// InteractionTradePutItem is the mode-15 arm of CTradingRoomDlg's dispatcher
// (v83 sub_7C1F6D @0x7c1f6d -> sub_7C1FB7 @0x7c1fb7): Decode1 side, Decode1
// trade slot, then GW_ItemSlotBase. side is the recipient-relative room side
// (0 = the receiving client's own side, 1 = the counterparty); tradeSlot is the
// 1..9 slot within that side's grid.
//
// packet-audit:fname CTradingRoomDlg::OnPutItem
type InteractionTradePutItem struct {
	mode      byte
	side      byte
	tradeSlot byte
	asset     model.Asset
}

func NewInteractionTradePutItem(mode byte, side byte, tradeSlot byte, a model.Asset) InteractionTradePutItem {
	return InteractionTradePutItem{mode: mode, side: side, tradeSlot: tradeSlot, asset: a}
}

func (m InteractionTradePutItem) Mode() byte        { return m.mode }
func (m InteractionTradePutItem) Side() byte        { return m.side }
func (m InteractionTradePutItem) TradeSlot() byte   { return m.tradeSlot }
func (m InteractionTradePutItem) Asset() model.Asset { return m.asset }

func (m InteractionTradePutItem) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.side)
		w.WriteByte(m.tradeSlot)
		w.WriteByteArray(m.asset.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *InteractionTradePutItem) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.side = r.ReadByte()
		m.tradeSlot = r.ReadByte()
		m.asset = model.NewAsset(true, 0, 0, time.Time{})
		m.asset.Decode(l, ctx)(r, options)
	}
}
```

(Add `"time"` to the import block.)

- [ ] **Step 4: Add the mode key and body constructor**

In `libs/atlas-packet/interaction/clientbound/interaction_body.go`, inside the
existing operations const block (`:33-51`), after
`CharacterInteractionModeLeave`:

```go
	// Trade-family arms, CTradingRoomDlg's mode dispatcher (v83 sub_7C1F6D
	// @0x7c1f6d). Bytes are per-version and resolved from the tenant
	// "operations" table — the v83 values in these comments are documentation,
	// never a default (DOM-25).
	CharacterInteractionModeTradePutItem   CharacterInteractionMode = "TRADE_PUT_ITEM"   // 15
	CharacterInteractionModeTradeAddMeso   CharacterInteractionMode = "TRADE_ADD_MESO"   // 16
	CharacterInteractionModeTradeConfirm   CharacterInteractionMode = "TRADE_CONFIRM"    // 17
	CharacterInteractionModeTradeMesoLimit CharacterInteractionMode = "TRADE_MESO_LIMIT" // 21
```

Then append the body constructor near the other family constructors:

```go
// CharacterInteractionTradePutItemBody announces one staged item to a trade
// room occupant. side is recipient-relative (0 = the receiving client's own
// side, 1 = the counterparty); tradeSlot is 1..9.
func CharacterInteractionTradePutItemBody(side byte, tradeSlot byte, a model.Asset) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeTradePutItem, func(mode byte) packet.Encoder {
		return NewInteractionTradePutItem(mode, side, tradeSlot, a)
	})
}
```

(Import `"github.com/Chronicle20/atlas/libs/atlas-packet/model"` if not already
present in that file.)

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./interaction/... -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/interaction/clientbound/
git commit -m "feat(task-205): add TRADE_PUT_ITEM clientbound codec"
```

---

### Task 3: `TRADE_ADD_MESO` clientbound codec

**Files:**
- Modify: `libs/atlas-packet/interaction/clientbound/interaction_trade.go`
- Modify: `libs/atlas-packet/interaction/clientbound/interaction_body.go` (body constructor)
- Test: `libs/atlas-packet/interaction/clientbound/interaction_trade_test.go`

**Interfaces:**
- Consumes: mode key `CharacterInteractionModeTradeAddMeso` (added in Task 2).
- Produces:
  - `clientbound.NewInteractionTradeAddMeso(mode byte, side byte, amount uint32) InteractionTradeAddMeso`
  - `clientbound.CharacterInteractionTradeAddMesoBody(side byte, amount uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte` — consumed by Task 22.

Design §1.2 and §4.2: mode 16, body `side:byte, amount:uint32`, authority
`sub_7C208E` @`0x7c208e`. **The client performs an absolute assignment**
(`this[v3+115] = Decode4`), not an accumulation — which is what makes the
authoritative re-echo the meso-rejection mechanism (design §4.2).

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-packet/interaction/clientbound/interaction_trade_test.go`:

```go
// TestInteractionTradeAddMesoBytes pins the mode-16 arm: Decode1 side, Decode4
// amount (v83 sub_7C208E @0x7c208e). The client ASSIGNS the amount
// (this[v3+115] = Decode4), so an authoritative re-echo of the last valid
// amount is how the server corrects an out-of-range stage (design §4.2).
func TestInteractionTradeAddMesoBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			raw := NewInteractionTradeAddMeso(16, 1, 1000000).Encode(l, ctx)(nil)
			if got := hex.EncodeToString(raw); got != "100140420f00" {
				t.Errorf("bytes: got %s, want 100140420f00", got)
			}
		})
	}
}

func TestInteractionTradeAddMesoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewInteractionTradeAddMeso(16, 0, 987654321)
			output := InteractionTradeAddMeso{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Side() != input.Side() {
				t.Errorf("side: got %v, want %v", output.Side(), input.Side())
			}
			if output.Amount() != input.Amount() {
				t.Errorf("amount: got %v, want %v", output.Amount(), input.Amount())
			}
		})
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd libs/atlas-packet && go test ./interaction/clientbound/ -run TestInteractionTradeAddMeso -v`
Expected: FAIL — `undefined: NewInteractionTradeAddMeso`.

- [ ] **Step 3: Write the codec**

Append to `libs/atlas-packet/interaction/clientbound/interaction_trade.go`:

```go
// InteractionTradeAddMeso is the mode-16 arm of CTradingRoomDlg's dispatcher
// (v83 sub_7C208E @0x7c208e): Decode1 side, Decode4 amount. The client ASSIGNS
// the value (this[v3+115] = Decode4) rather than accumulating, so re-sending
// the last valid amount is an authoritative correction the client's view snaps
// back to.
//
// packet-audit:fname CTradingRoomDlg::OnSetMoney
type InteractionTradeAddMeso struct {
	mode   byte
	side   byte
	amount uint32
}

func NewInteractionTradeAddMeso(mode byte, side byte, amount uint32) InteractionTradeAddMeso {
	return InteractionTradeAddMeso{mode: mode, side: side, amount: amount}
}

func (m InteractionTradeAddMeso) Mode() byte     { return m.mode }
func (m InteractionTradeAddMeso) Side() byte     { return m.side }
func (m InteractionTradeAddMeso) Amount() uint32 { return m.amount }

func (m InteractionTradeAddMeso) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.side)
		w.WriteInt(m.amount)
		return w.Bytes()
	}
}

func (m *InteractionTradeAddMeso) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.side = r.ReadByte()
		m.amount = r.ReadUint32()
	}
}
```

- [ ] **Step 4: Add the body constructor**

Append to `libs/atlas-packet/interaction/clientbound/interaction_body.go`:

```go
// CharacterInteractionTradeAddMesoBody announces a side's staged meso total.
// The amount is ABSOLUTE, not a delta — see InteractionTradeAddMeso.
func CharacterInteractionTradeAddMesoBody(side byte, amount uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeTradeAddMeso, func(mode byte) packet.Encoder {
		return NewInteractionTradeAddMeso(mode, side, amount)
	})
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./interaction/... -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/interaction/clientbound/
git commit -m "feat(task-205): add TRADE_ADD_MESO clientbound codec"
```

---

### Task 4: `TRADE_CONFIRM` and `TRADE_MESO_LIMIT` bodyless codecs

**Files:**
- Modify: `libs/atlas-packet/interaction/clientbound/interaction_trade.go`
- Modify: `libs/atlas-packet/interaction/clientbound/interaction_body.go`
- Test: `libs/atlas-packet/interaction/clientbound/interaction_trade_test.go`

**Interfaces:**
- Consumes: mode keys added in Task 2.
- Produces:
  - `clientbound.NewInteractionTradeConfirm(mode byte) InteractionTradeConfirm`
  - `clientbound.NewInteractionTradeMesoLimit(mode byte) InteractionTradeMesoLimit`
  - `clientbound.CharacterInteractionTradeConfirmBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`
  - `clientbound.CharacterInteractionTradeMesoLimitBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`

Design §1.2: mode 17 (`CTradingRoomDlg::OnTrade` @`0x7c20bc`) reads **nothing**
and immediately auto-sends serverbound `0x14`. Mode 21 (`sub_7C21BD`
@`0x7c21bd`) also reads nothing; it shows `SP_3977` and clears the local confirm
flag. Both are mode-byte-only bodies.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-packet/interaction/clientbound/interaction_trade_test.go`:

```go
// TestInteractionTradeConfirmIsBodyless pins design §1.2: mode 17
// (CTradingRoomDlg::OnTrade @0x7c20bc) reads NO body — it sets this[112]=1,
// redraws, and immediately auto-sends serverbound 0x14 with the client's own
// CRC list. A stray trailing byte here would be read as the next packet.
func TestInteractionTradeConfirmIsBodyless(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			if got := hex.EncodeToString(NewInteractionTradeConfirm(17).Encode(l, ctx)(nil)); got != "11" {
				t.Errorf("bytes: got %s, want 11", got)
			}
		})
	}
}

// TestInteractionTradeMesoLimitIsBodyless pins design §1.2/§11.2: mode 21
// (sub_7C21BD @0x7c21bd) reads no body; it shows SP_3977 ("Players that are
// level 15 and below may only trade 1 million mesos per day"), clears
// this[111] and re-enables both confirm buttons. CCashTradingRoomDlg::OnPacket
// @0x4833b4 has NO mode-21 arm.
func TestInteractionTradeMesoLimitIsBodyless(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			if got := hex.EncodeToString(NewInteractionTradeMesoLimit(21).Encode(l, ctx)(nil)); got != "15" {
				t.Errorf("bytes: got %s, want 15", got)
			}
		})
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd libs/atlas-packet && go test ./interaction/clientbound/ -run 'TestInteractionTrade(Confirm|MesoLimit)' -v`
Expected: FAIL — `undefined: NewInteractionTradeConfirm`.

- [ ] **Step 3: Write both codecs**

Append to `libs/atlas-packet/interaction/clientbound/interaction_trade.go`:

```go
// InteractionTradeConfirm is the mode-17 arm (v83 CTradingRoomDlg::OnTrade
// @0x7c20bc). It carries NO body: the client sets its confirmed flag, redraws,
// and immediately sends serverbound PLAYER_INTERACTION mode 0x14 (TRANSACTION)
// with its own {itemId, itemCRC} attestation list.
//
// Because receipt auto-replies, this arm is broadcast ONLY after BOTH sides
// have confirmed — sending it on the first confirm would drive the
// counterparty's attestation without its owner ever pressing Trade
// (design §6.2).
//
// packet-audit:fname CTradingRoomDlg::OnTrade
type InteractionTradeConfirm struct {
	mode byte
}

func NewInteractionTradeConfirm(mode byte) InteractionTradeConfirm {
	return InteractionTradeConfirm{mode: mode}
}

func (m InteractionTradeConfirm) Mode() byte { return m.mode }

func (m InteractionTradeConfirm) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *InteractionTradeConfirm) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// InteractionTradeMesoLimit is the mode-21 arm (v83 sub_7C21BD @0x7c21bd), the
// server-side twin of CTradingRoomDlg::PutMoney's client-side daily-meso guard.
// Bodyless: the client shows SP_3977, clears its local confirm flag
// (this[111] = 0) and re-enables both confirm buttons.
//
// CCashTradingRoomDlg::OnPacket (@0x4833b4) dispatches 15/16/17 only — there is
// no mode-21 arm in the cash room. Where the arm is absent, meso rejection
// degrades to the authoritative TRADE_ADD_MESO re-echo alone (design §4.2).
//
// packet-audit:fname CTradingRoomDlg::OnMesoLimit
type InteractionTradeMesoLimit struct {
	mode byte
}

func NewInteractionTradeMesoLimit(mode byte) InteractionTradeMesoLimit {
	return InteractionTradeMesoLimit{mode: mode}
}

func (m InteractionTradeMesoLimit) Mode() byte { return m.mode }

func (m InteractionTradeMesoLimit) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *InteractionTradeMesoLimit) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
```

- [ ] **Step 4: Add both body constructors**

Append to `libs/atlas-packet/interaction/clientbound/interaction_body.go`:

```go
// CharacterInteractionTradeConfirmBody prompts a client for its CRC
// attestation. Send only after BOTH sides have confirmed (design §6.2).
func CharacterInteractionTradeConfirmBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeTradeConfirm, func(mode byte) packet.Encoder {
		return NewInteractionTradeConfirm(mode)
	})
}

// CharacterInteractionTradeMesoLimitBody tells a client its meso stage was
// refused. Absent from the cash trade room on every version; pair it with an
// authoritative CharacterInteractionTradeAddMesoBody re-echo so the correction
// lands even where this arm does not exist.
func CharacterInteractionTradeMesoLimitBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeTradeMesoLimit, func(mode byte) packet.Encoder {
		return NewInteractionTradeMesoLimit(mode)
	})
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./interaction/... -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/interaction/clientbound/
git commit -m "feat(task-205): add TRADE_CONFIRM and TRADE_MESO_LIMIT clientbound codecs"
```

---

### Task 5: Trade leave-reason keys and the `OperationTransaction` fname correction

**Files:**
- Modify: `libs/atlas-packet/interaction/clientbound/interaction_body.go:163-179` (leaveReason const block)
- Modify: `libs/atlas-packet/interaction/serverbound/operation_transaction.go` (fname comment)
- Test: `libs/atlas-packet/interaction/clientbound/interaction_trade_test.go`

**Interfaces:**
- Produces: six `CharacterInteractionLeaveReasonTrade*` string constants consumed by Task 22 via the existing `CharacterInteractionLeaveReasonBody(slot byte, reason string)` (`interaction_body.go:186-193`). **No new codec** — design §1.4 established that completion is `LEAVE` + slot + status.

> **Ordering warning (design §11.1):** editing a `packet-audit:fname` re-keys
> every evidence record that references it, staling those matrix cells. The
> fname edit MUST be followed by a matrix regeneration and re-pin in the same
> commit — never as a drive-by comment change.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-packet/interaction/clientbound/interaction_trade_test.go`:

```go
// TestTradeLeaveReasonKeysAreDistinct pins design §4.3: the trade leave path
// gets its OWN leaveReason keys and never borrows the shop or mini-game keys'
// numeric values (DOM-25 / the precedent at interaction_body.go:167-179).
func TestTradeLeaveReasonKeysAreDistinct(t *testing.T) {
	tradeKeys := []string{
		CharacterInteractionLeaveReasonTradeCancelled,
		CharacterInteractionLeaveReasonTradeSuccess,
		CharacterInteractionLeaveReasonTradeFailed,
		CharacterInteractionLeaveReasonTradeCannotCarry,
		CharacterInteractionLeaveReasonTradeDifferentMap,
		CharacterInteractionLeaveReasonTradeCrcFailed,
	}
	otherKeys := []string{
		CharacterInteractionLeaveReasonShopClosed,
		CharacterInteractionLeaveReasonUserBanned,
		CharacterInteractionLeaveReasonOutOfStock,
		CharacterInteractionLeaveReasonMiniGameClosed,
		CharacterInteractionLeaveReasonMiniGameLeft,
		CharacterInteractionLeaveReasonMiniGameExpelled,
	}

	seen := make(map[string]bool)
	for _, k := range tradeKeys {
		if k == "" {
			t.Fatal("empty trade leave reason key")
		}
		if seen[k] {
			t.Errorf("duplicate trade leave reason key %q", k)
		}
		seen[k] = true
	}
	for _, k := range otherKeys {
		if seen[k] {
			t.Errorf("trade leave reason key %q collides with a non-trade family key", k)
		}
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd libs/atlas-packet && go test ./interaction/clientbound/ -run TestTradeLeaveReasonKeys -v`
Expected: FAIL — `undefined: CharacterInteractionLeaveReasonTradeCancelled`.

- [ ] **Step 3: Add the keys**

Inside the existing `leaveReason` const block at
`libs/atlas-packet/interaction/clientbound/interaction_body.go:167-179`, after
the mini-game keys:

```go
	// Trade leave-status keys, resolved via the same "leaveReason" tenant
	// table. CTradingRoomDlg::OnLeave (v83 vtable off_B37448 slot +76 ->
	// 0x7C221D) reads one status byte and branches: 2 SP_406 "Trade cancelled
	// by the other character"; 7 success (SP_408 with the meso figure the
	// client computes from its OWN CharacterData, else SP_407); 8 SP_409
	// "Trade unsuccessful"; 9 SP_410 "...items which you cannot carry";
	// 12 SP_411 "...the other person's on a different map"; 13 SP_5566 CRC
	// mismatch. Distinct keys so the trade path never depends on another
	// family's numeric values (DOM-25).
	CharacterInteractionLeaveReasonTradeCancelled    = "TRADE_CANCELLED"     // 2
	CharacterInteractionLeaveReasonTradeSuccess      = "TRADE_SUCCESS"       // 7
	CharacterInteractionLeaveReasonTradeFailed       = "TRADE_FAILED"        // 8
	CharacterInteractionLeaveReasonTradeCannotCarry  = "TRADE_CANNOT_CARRY"  // 9
	CharacterInteractionLeaveReasonTradeDifferentMap = "TRADE_DIFFERENT_MAP" // 12
	CharacterInteractionLeaveReasonTradeCrcFailed    = "TRADE_CRC_FAILED"    // 13
```

- [ ] **Step 4: Correct the `OperationTransaction` fname**

In `libs/atlas-packet/interaction/serverbound/operation_transaction.go`, replace
`// packet-audit:fname CCashTradingRoomDlg::Trade` with:

```go
// packet-audit:fname CTradingRoomDlg::OnTrade
```

Add above the struct:

```go
// OperationTransaction is the client's CRC attestation reply, NOT a user
// action. The only sender on v83 is CTradingRoomDlg::OnTrade @0x7c20bc, which
// fires automatically on receipt of clientbound mode 17. The former fname
// (CCashTradingRoomDlg::Trade @0x485dcd) is wrong: that function sends
// Encode1(0x11) — TRADE_CONFIRM — like CTradingRoomDlg::Trade @0x7c39a0
// (task-205 design.md §1.5, §11.1).
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./interaction/... -v`
Expected: PASS.

- [ ] **Step 6: Regenerate and re-pin the matrix**

Run the packet-audit toolchain per
[`docs/packets/PROCESS.md`](../../packets/PROCESS.md) to regenerate
`docs/packets/audits/status.json` and `STATUS.md` and re-pin every evidence
record whose key referenced the old fname:

```bash
packet-audit fname-doc --check
packet-audit matrix
packet-audit operations --check
```

Expected: `fname-doc --check` and `operations --check` exit 0; `git diff` shows
the re-keyed evidence records and the regenerated matrix. If any cell degrades
to unverified, re-pin it before committing — a degraded cell is a failure.

- [ ] **Step 7: Commit fname + matrix together**

```bash
git add libs/atlas-packet/interaction/ docs/packets/audits/
git commit -m "fix(task-205): correct OperationTransaction fname, add trade leaveReason keys, re-pin matrix"
```

---

### Task 6: Per-version derivation, fixtures and matrix promotion

**Files:**
- Modify: `libs/atlas-packet/interaction/clientbound/interaction_trade.go` (version gates, only where a version diverges)
- Create: `libs/atlas-packet/interaction/clientbound/version_gate.go` (only if a divergence is found)
- Modify: `libs/atlas-packet/interaction/clientbound/interaction_trade_test.go` (per-version `packet-audit:verify` markers + hex pins)
- Modify: `docs/packets/audits/` evidence records + `status.json` + `STATUS.md`
- Create: `docs/tasks/task-205-player-trade/coverage-manifest.yaml`

**Interfaces:**
- Consumes: the four codecs from Tasks 2-4 and `NewTradeRoom` from Task 1.
- Produces: promoted matrix cells; the per-version mode-byte and arm-presence table that Task 23 (templates) consumes.

This task does **not** invent a procedure. It drives the project's documented
one — `/implement-packet` +
[`docs/packets/IMPLEMENTING_A_PACKET.md`](../../packets/IMPLEMENTING_A_PACKET.md)
for the new codecs, with the leaf step
`/verify-packet` +
[`docs/packets/audits/VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md)
per cell. Dispatch the `packet-implementer` agent once for the trade family, and
`packet-verifier` per op × version cell (batched per IDB).

- [ ] **Step 1: Record the coverage manifest before touching anything**

Create `docs/tasks/task-205-player-trade/coverage-manifest.yaml` declaring every
op × version cell this task claims, so `packet-completeness-critic` can diff it
against the branch later:

```yaml
task: task-205-player-trade
ops:
  - interaction/clientbound/InteractionTradePutItem
  - interaction/clientbound/InteractionTradeAddMeso
  - interaction/clientbound/InteractionTradeConfirm
  - interaction/clientbound/InteractionTradeMesoLimit
  - interaction/clientbound/InteractionEnterResultSuccess   # trade room variant
  - interaction/clientbound/InteractionLeave                # trade statuses
  - interaction/serverbound/InteractionOperationTransaction # fname correction
versions:
  - gms_v48
  - gms_v61
  - gms_v72
  - gms_v79
  - gms_v83
  - gms_v84
  - gms_v87
  - gms_v92
  - gms_v95
  - jms_v185
```

- [ ] **Step 2: Derive each version from its IDB**

For each of the ten versions, follow design §10.1 exactly. Resolve the IDA
session from `idb_list` **by binary name** and pass it as the `database`
parameter (`select_instance`/port selection is dead). Use `func_query` with
`name_regex`.

1. Locate the trade dialog ctor (search `TradingRoom` / the
   `UI/UIWindow.img/TradingRoom` StringPool id); read its vtable pointer.
2. `vtable+72` → enter-result tail. A nullsub means the version matches §1.3 and
   `NewTradeRoom` needs no gate. **Anything else is a divergence** — model the
   tail and add a `MajorAtLeast` gate.
3. `vtable+76` → `OnLeave`; enumerate the status-byte branches. This is the
   per-version `leaveReason` mapping Task 23 writes into the templates.
4. Find the mode dispatcher reached from `CMiniRoomBaseDlg::OnPacketBase` and
   enumerate its cases. This yields the clientbound mode bytes for
   `TRADE_PUT_ITEM` / `TRADE_ADD_MESO` / `TRADE_CONFIRM` and whether the
   mode-21 arm exists.
5. Search for a second trading-room class with its own dispatcher (v83's
   `CCashTradingRoomDlg::OnPacket` @`0x4833b4` is the shape). Absent dispatcher →
   ⬜ n-a for every cash-trade cell on that version, and `CASH_TRADE_OPEN` is
   **not** added to that template.

**Absence is asserted only from a decompiled dispatcher that lacks the case.**
An unnamed symbol is not evidence of absence — name it and re-read. Name every
symbol you reverse as you go.

- [ ] **Step 3: Record the derivation table**

Write the results into `docs/tasks/task-205-player-trade/version-matrix.md` with
one row per version and one column per op, each cell carrying the mode byte and
the IDA address it came from, plus an `n-a` marker with the dispatcher address
that lacks the case. Task 23 consumes this table verbatim; do not let it live
only in the agent transcript.

Resolve here, with evidence, the two open questions design §10.2/§10.3 left as
hypotheses:

- Is `TRANSACTION` absent below v83 (the CRC-boundary prediction)? jms_v185 is
  the stated exception — check it directly, do not infer from the GMS boundary.
- Does `CCashTradingRoomDlg` exist on gms_v48/61/72 and jms_v185?

- [ ] **Step 4: Add version gates only where the derivation found divergence**

If and only if step 2 found a version whose layout differs, create
`libs/atlas-packet/interaction/clientbound/version_gate.go` mirroring the
serverbound file's shape, using `MajorAtLeast`:

```go
package clientbound

import (
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// tradeRoomHasEnterResultTail reports whether CTradingRoomDlg's enter-result
// tail virtual (vtable+72) reads a body. On GMS v83 it is nullsub_94
// @0x48314D — empty — so the room body is the base frame alone. Replace the
// predicate below with the boundary the per-version derivation actually found;
// do NOT ship this file if every version is a nullsub.
func tradeRoomHasEnterResultTail(t tenant.Model) bool {
	return false
}
```

If every version is a nullsub, **do not create the file** — an always-false gate
is dead code.

- [ ] **Step 5: Add per-version fixtures with verify markers**

For each op × version cell, add a byte-fixture test carrying a
`// packet-audit:verify packet=<op> version=<version> ida=0x<addr>` marker and a
pinned hex expectation, following the shape of
`libs/atlas-packet/interaction/serverbound/operation_trade_put_item_test.go`
exactly. Versions inside `pt.Variants` use the `pt.Variants` loop; versions
outside it (v48, v61, v72, v79, v92) get their own hex-pinning test with
`pt.CreateContext(region, major, minor)`.

A round-trip assertion alone does **not** promote a cell — it proves the codec
is self-consistent, not client-correct. Every promoted cell needs the derived
read order and a pinned byte expectation.

- [ ] **Step 6: Run the fixtures**

Run: `cd libs/atlas-packet && go test ./interaction/... -v`
Expected: PASS for every version.

- [ ] **Step 7: Regenerate and verify the matrix**

```bash
packet-audit matrix
packet-audit fname-doc --check
packet-audit operations --check
packet-audit dispatcher-lint
```

Expected: all exit 0, and `docs/packets/audits/STATUS.md` shows every claimed
cell as ✅ or ⬜ n-a (never ❌, never blank). Confirm serverbound
`PLAYER_INTERACTION` (v83 0x07B) is no longer ❌ (PRD FR-8.9). Note the matrix
`toolSha` reads git HEAD — commit code before regenerating so the recorded sha
matches.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-packet/ docs/packets/audits/ docs/tasks/task-205-player-trade/
git commit -m "feat(task-205): derive and verify trade codecs across ten client versions"
```

---

# Slice 2 — Prerequisites (Tasks 7-8)

Two small isolated changes everything downstream depends on. Independent of
Slice 1; can run in parallel with it.

### Task 7: Parameterise the inventory reservation TTL

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:61-64` (interface), `:752-795` (impl)
- Modify: `services/atlas-inventory/atlas.com/inventory/compartment/mock/processor.go:42-43`
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go:72-81` (command body)
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go:149-170`
- Modify (mirror the wire body): `services/atlas-channel/atlas.com/channel/kafka/message/compartment/kafka.go:62-70`, `services/atlas-consumables/atlas.com/consumables/kafka/message/compartment/kafka.go:27`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/compartment/kafka.go`
- Modify (pass the existing 30 s explicitly): `services/atlas-channel/atlas.com/channel/compartment/processor.go:27,79-81`, `compartment/producer.go:88-96`, `socket/handler/character_attack_projectile.go:172`, `skill/handler/shadow_stars.go:155`; `services/atlas-consumables/atlas.com/consumables/compartment/processor.go:25,47`, `compartment/producer.go:26-30`, `compartment/mock/processor.go:13,22-24`, and callers `consumable/processor.go:309,343,713,1078,1358`, `consumable/vega.go:154,176`
- Test: `services/atlas-inventory/atlas.com/inventory/compartment/processor_test.go`

**Interfaces:**
- Produces:
  - `compartment.Processor.RequestReserve(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, expiry time.Duration, reservationRequests []ReservationRequest) error`
  - `compartment.Processor.RequestReserveAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, expiry time.Duration, reservationRequests []ReservationRequest) error`
  - Wire body gains `ExpirySeconds uint32 \`json:"expirySeconds"\`` — consumed by Task 18.

Design §5.3 / §11.5: the hard-coded 30 s at `processor.go:782` is the
drop-reservation lifetime and far too short for a trade window. Every existing
call site passes 30 s explicitly, so nothing outside trade changes behaviour.

This task also fixes the loop bug found during planning (see
[`context.md`](context.md) §1.11): the `for _, request := range
reservationRequests` body unconditionally `return`s `mb.Put(...)` on its first
iteration, so only the first item of a multi-item request is ever reserved.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-inventory/atlas.com/inventory/compartment/processor_test.go`:

```go
// TestRequestReserveHonoursExpiry pins task-205 design §5.3: the reservation
// TTL is caller-supplied, not the hard-coded 30s drop lifetime. A trade room
// holds reservations for the whole trade window (default 300s) and refreshes
// them on a ticker.
func TestRequestReserveHonoursExpiry(t *testing.T) {
	// Scaffolding: reuse this file's existing helpers (see processor_test.go:434,
	// :492, :614) to build a tenant context, a db handle, and a character whose
	// USE compartment holds one stack of 100 of item 2000000 in slot 1.
	ctx, db, characterId, tenantModel := testCompartmentWithStack(t, inventory.TypeValueUse, 1, 2000000, 100)
	p := NewProcessor(logger, ctx, db)
	err := p.RequestReserveAndEmit(uuid.New(), characterId, inventory.TypeValueUse, 300*time.Second, []ReservationRequest{
		{Slot: 1, ItemId: 2000000, Quantity: 5},
	})
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	res, ok := GetReservationRegistry().GetReservation(tenantModel, characterId, inventory.TypeValueUse, 1)
	if !ok {
		t.Fatal("expected a live reservation")
	}
	if remaining := time.Until(res.Expiration()); remaining <= 60*time.Second {
		t.Errorf("expiry: got %v remaining, want > 60s (a 300s TTL was requested)", remaining)
	}
}

// TestRequestReserveProcessesEveryRequest pins that a multi-item request
// reserves EVERY item, not just the first. The pre-task-205 loop returned
// mb.Put(...) on its first iteration.
func TestRequestReserveProcessesEveryRequest(t *testing.T) {
	p := NewProcessor(logger, ctx, db)
	err := p.RequestReserveAndEmit(uuid.New(), characterId, inventory.TypeValueUse, 30*time.Second, []ReservationRequest{
		{Slot: 1, ItemId: 2000000, Quantity: 5},
		{Slot: 2, ItemId: 2000001, Quantity: 3},
	})
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	for _, slot := range []int16{1, 2} {
		if q := GetReservationRegistry().GetReservedQuantity(tenantModel, characterId, inventory.TypeValueUse, slot); q == 0 {
			t.Errorf("slot %d: expected a reservation, got 0 reserved", slot)
		}
	}
}
```

Match the surrounding file's existing scaffolding style for the elided setup —
`processor_test.go:434,492,614` already build a processor and a stocked
compartment; reuse those helpers rather than inventing new ones.

- [ ] **Step 2: Run them and confirm they fail**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test ./compartment/ -run 'TestRequestReserve(HonoursExpiry|ProcessesEveryRequest)' -v`
Expected: FAIL — the first with a compile error (`too many arguments`), the
second with `slot 2: expected a reservation, got 0 reserved`.

- [ ] **Step 3: Change the signature and fix the loop**

In `services/atlas-inventory/atlas.com/inventory/compartment/processor.go`,
update the interface at `:61-64`:

```go
	// RequestReserve holds `expiry` worth of claim on the requested slots.
	// The caller owns the TTL: 30s for a drop/attack reservation, a full trade
	// window (default 300s, refreshed on a ticker) for atlas-trades. It was
	// hard-coded at 30s before task-205.
	RequestReserveAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, expiry time.Duration, reservationRequests []ReservationRequest) error
	RequestReserve(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, expiry time.Duration, reservationRequests []ReservationRequest) error
```

In the impl, thread `expiry` through and correct the loop so it accumulates
instead of returning early:

```go
			for _, request := range reservationRequests {
				var a asset.Model
				a, err = p.assetProcessor.WithTransaction(tx).GetBySlot(c.Id(), request.Slot)
				if err != nil {
					return err
				}
				if a.TemplateId() != request.ItemId {
					return errors.New("cannot reserve non-existent item")
				}
				currentReservedQty := GetReservationRegistry().GetReservedQuantity(p.t, characterId, inventoryType, request.Slot)
				if a.Quantity()-currentReservedQty < uint32(request.Quantity) {
					return errors.New("cannot reserve more than what is owned")
				}
				_, err = GetReservationRegistry().AddReservation(p.t, transactionId, characterId, inventoryType, request.Slot, request.ItemId, uint32(request.Quantity), expiry)
				if err != nil {
					return err
				}
				// Emit per request. Before task-205 this was `return mb.Put(...)`,
				// which silently dropped every request after the first.
				if err = mb.Put(compartment.EnvEventTopicStatus, ReservedEventStatusProvider(transactionId, c.Id(), characterId, request.ItemId, request.Slot, uint32(request.Quantity))); err != nil {
					return err
				}
			}
			return nil
```

Update `mock/processor.go:42-43` to the new signature.

- [ ] **Step 4: Extend the wire body**

In `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go`:

```go
type RequestReserveCommandBody struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	// ExpirySeconds is the reservation TTL. Zero means the historical 30s
	// default, so pre-task-205 producers keep working unchanged.
	ExpirySeconds uint32     `json:"expirySeconds"`
	Items         []ItemBody `json:"items"`
}
```

Mirror the identical field into the three copies of this struct listed under
**Files** — they are byte-for-byte mirrors and must stay so.

In the consumer (`kafka/consumer/compartment/consumer.go:149-170`), derive the
duration:

```go
		expiry := 30 * time.Second
		if c.Body.ExpirySeconds > 0 {
			expiry = time.Duration(c.Body.ExpirySeconds) * time.Second
		}
		_ = compartment.NewProcessor(l, ctx, db).RequestReserveAndEmit(transactionId, c.CharacterId, inventory.Type(c.InventoryType), expiry, reserves)
```

- [ ] **Step 5: Pass 30 s explicitly at every existing call site**

Update each caller listed under **Files** to pass `30*time.Second` (channel,
consumables) or `ExpirySeconds: 30` (producers building the command body). This
is a mechanical pass; behaviour is unchanged everywhere outside trade.

- [ ] **Step 6: Run the tests**

```bash
cd services/atlas-inventory/atlas.com/inventory && go test -race ./... && go vet ./...
cd ../../../atlas-channel/atlas.com/channel && go build ./... && go test -race ./...
cd ../../../atlas-consumables/atlas.com/consumables && go build ./... && go test -race ./...
cd ../../../atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./...
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-inventory services/atlas-channel services/atlas-consumables services/atlas-saga-orchestrator
git commit -m "feat(task-205): parameterise inventory reservation TTL; reserve every requested slot"
```

---

### Task 8: Surface `tradeBlock` from the equipment, etc and cash readers

**Files:**
- Modify: `services/atlas-data/atlas.com/data/equipment/rest.go:17-49`, `equipment/reader.go:86-116`
- Modify: `services/atlas-data/atlas.com/data/etc/rest.go:7-15`, `etc/reader.go:40-53`
- Modify: `services/atlas-data/atlas.com/data/cash/rest.go:37-48`, `cash/reader.go:53+`
- Test: `services/atlas-data/atlas.com/data/equipment/reader_test.go`, `etc/reader_test.go`, `cash/reader_test.go`

**Interfaces:**
- Produces: `TradeBlock bool \`json:"tradeBlock"\`` on the equipment, etc and cash `RestModel`s — consumed by Task 18 (atlas-trades restriction checks) over REST.

PRD FR-4.2 is explicit: a missing flag must **not** be read as "tradeable".
Consumable (`consumable/reader.go:49`) and setup (`setup/reader.go:47`) already
expose it; these three do not.

- [ ] **Step 1: Write the failing tests**

`equipment/reader_test.go` already has a fixture with
`<int name="tradeBlock" value="1"/>` at `:40` that is currently unasserted. Add
the assertion, and add a fixture + assertion to the etc and cash reader tests:

```go
// TestEquipmentReaderSurfacesTradeBlock pins PRD FR-4.2: tradeBlock must be
// readable for every item family a trade can stage, not just consumable and
// setup. A missing flag must never be read as "tradeable".
func TestEquipmentReaderSurfacesTradeBlock(t *testing.T) {
	m := readEquipmentFixture(t) // existing helper in this file
	if !m.TradeBlock {
		t.Error("tradeBlock: got false, want true (fixture sets tradeBlock=1)")
	}
}

func TestEquipmentReaderTradeBlockDefaultsFalse(t *testing.T) {
	m := readEquipmentFixtureWithout(t, "tradeBlock")
	if m.TradeBlock {
		t.Error("tradeBlock: got true, want false when the WZ node is absent")
	}
}
```

Write the analogous pair for `etc` and `cash`, adding
`<int name="tradeBlock" value="1"/>` to their `info` fixture nodes.

- [ ] **Step 2: Run them and confirm they fail**

Run: `cd services/atlas-data/atlas.com/data && go test ./equipment/ ./etc/ ./cash/ -run TradeBlock -v`
Expected: FAIL — `m.TradeBlock undefined`.

- [ ] **Step 3: Add the fields and reader lines**

`equipment/rest.go` — add to `RestModel` beside `Cash`/`Price`/`TimeLimited`:

```go
	TradeBlock bool `json:"tradeBlock"`
```

`equipment/reader.go:86-116` uses a single composite literal with an `info.`
receiver — add, next to `TimeLimited`:

```go
			TradeBlock:     info.GetBool("tradeBlock", false),
```

`etc/rest.go` — add the same field. `etc/reader.go:40-53` uses post-assignment,
so add next to the other `m.` lines:

```go
			m.TradeBlock = i.GetBool("tradeBlock", false)
```

`cash/rest.go` — add the same field. `cash/reader.go` resolves the info node as
`i, err := cxml.ChildByName("info")`; add the same `m.TradeBlock =
i.GetBool("tradeBlock", false)` line.

- [ ] **Step 4: Run the tests**

```bash
cd services/atlas-data/atlas.com/data && go test -race ./equipment/ ./etc/ ./cash/ -v && go vet ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data
git commit -m "feat(task-205): surface tradeBlock from the equipment, etc and cash WZ readers"
```

> **Operational note for Task 24's acceptance run:** existing ingested data was
> parsed without these fields. A re-ingest is required before the flag reads
> true on live data — effects are ingested, not re-parsed on read.

---

# Slice 3 — atlas-trades skeleton (Tasks 9-13)

Service registration, domain model, ledger, REST and configuration. No trade
behaviour yet — Slice 5 adds it.

### Task 9: Register and bootstrap atlas-trades

**Files:**
- Create: `services/atlas-trades/atlas.com/trades/main.go`, `go.mod`, `README.md`, `rest/handler.go`
- Modify: `.github/config/services.json`, `docker-bake.hcl` (`go_services`), `go.work`, `tools/db-bootstrap.sh`
- Create: `deploy/k8s/base/atlas-trades.yaml`
- Modify: `deploy/k8s/base/kustomization.yaml`, `deploy/k8s/base/env-configmap.yaml`
- Modify: `deploy/k8s/overlays/main/kustomization.yaml`, `overlays/main/patches/db-name-suffix.yaml`, `overlays/main/patches/atlas-env-env.yaml`
- Modify: `deploy/k8s/overlays/pr/kustomization.yaml` (+ regenerate the three script-owned files)
- Modify: `deploy/shared/routes.conf` (+ regenerate `deploy/k8s/ingress.yaml`)

**Interfaces:**
- Produces: a buildable, deployable `atlas-trades` with a `/readyz` endpoint and no domain logic. Consumed by every later task in Slices 3-5.

Follow [`docs/adding-a-new-service.md`](../../adding-a-new-service.md) **in
full** — it enumerates every hand-maintained list and the four silent-failure
traps. Copy `services/atlas-mini-games` as the structural template.

- [ ] **Step 1: Scaffold the module**

```bash
mkdir -p services/atlas-trades/atlas.com/trades/rest
cd services/atlas-trades/atlas.com/trades
go mod init atlas-trades
```

Note the short module-name convention: the module is `atlas-trades`, and
internal packages import as `atlas-trades/<pkg>`.

- [ ] **Step 2: Write `main.go`**

Mirror `services/atlas-mini-games/atlas.com/mini-games/main.go:1-114`. At this
task the consumer blocks are absent (added in Tasks 16-20) and only the outbox
migration is registered:

```go
package main

import (
	"context"
	"os"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-trades"

var consumerGroupId = consumergroup.Resolve("Trade Service")

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string { return s.baseUrl }
func (s Server) GetPrefix() string  { return s.prefix }

func GetServer() Server { return Server{baseUrl: "", prefix: "/api/"} }

func main() {
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	// Live room state is the process-local in-memory trade.Registry, not the
	// DB — atlas-trades runs replicas: 1 for that reason (design §9). The DB
	// backs only the completed-trade ledger and the transactional outbox.
	db := database.Connect(l, database.SetMigrations(outboxlib.Migration))

	publisher := outboxlib.NewTopicWriterPool()
	drainer := outboxlib.NewDrainer(l, db, publisher, outboxlib.WithDSN(database.DSN()))
	routine.Go(l, rt.Context(), func(_ context.Context) {
		drainer.Run(rt.Context())
	})
	rt.TeardownFunc(func() {
		drainer.Stop()
		publisher.Close()
	})

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
```

- [ ] **Step 3: Write `rest/handler.go`**

Copy `services/atlas-mini-games/atlas.com/mini-games/rest/handler.go` (51 lines)
verbatim, adding `ParseRoomId` and `ParseEntryId` as `server.ParseUUIDId`
wrappers:

```go
func ParseRoomId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "roomId", next)
}

func ParseEntryId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "entryId", next)
}
```

- [ ] **Step 4: Register in build & CI (doc §1)**

- `.github/config/services.json`: add to `services[]` —
  `{"name": "atlas-trades", "type": "go-service", "path": "services/atlas-trades/atlas.com/trades", "module_path": "services/atlas-trades/atlas.com/trades", "docker_image": "atlas-trades", "docker_context": "."}` (match the exact key set of a neighbouring entry).
- `docker-bake.hcl`: add `"atlas-trades"` to the hardcoded `go_services` list.
  This is **hand-synced** with services.json — adding to one does not add to the other.
- `go.work`: add `./services/atlas-trades/atlas.com/trades` to `use()`.

- [ ] **Step 5: Register in k8s base (doc §2)**

- `deploy/k8s/base/atlas-trades.yaml`: Deployment + Service, copied from a
  DB-backed neighbour. Container `name: trades`. `DB_NAME: "atlas-trades"`
  (unsuffixed — overlays patch it). **`replicas: 1` and no HPA** (design §9);
  add a comment saying the room registry is process-local so this is a
  correctness constraint, not a capacity choice.
- `deploy/k8s/base/kustomization.yaml`: add `atlas-trades.yaml` to `resources:`.
- `deploy/k8s/base/env-configmap.yaml`: add the new topic vars as identity
  entries — `COMMAND_TOPIC_TRADE: "COMMAND_TOPIC_TRADE"` and
  `EVENT_TOPIC_TRADE_STATUS: "EVENT_TOPIC_TRADE_STATUS"`.

- [ ] **Step 6: Register in the main overlay (doc §3)**

- `patches/db-name-suffix.yaml`: new document, `DB_NAME: "atlas-trades-main"`,
  targeting container `trades`.
- `patches/atlas-env-env.yaml`: new document, `ATLAS_ENV: "main"`.
- `kustomization.yaml` → `images:`: add
  `- name: ghcr.io/chronicle20/atlas-trades/atlas-trades` with `newTag:` set to
  the current fleet tag. **A missing entry means the service runs `:latest`
  forever** — the bump workflow only rewrites entries already present.
- `kustomization.yaml` → `configMapGenerator` literals: add
  `COMMAND_TOPIC_TRADE=COMMAND_TOPIC_TRADE-main` and
  `EVENT_TOPIC_TRADE_STATUS=EVENT_TOPIC_TRADE_STATUS-main`. The generator uses
  `behavior: replace`, so any base key not re-listed here is **absent** on main.
- Do **not** add `KAFKA_CONSUMER_GROUP` to main.

- [ ] **Step 7: Register in the PR overlay (doc §4)**

- `kustomization.yaml` → `ATLAS_DB_NAMES`: add `atlas-trades`.
- `kustomization.yaml` → `images:`: same entry shape as step 6.
- Regenerate the three script-owned files — never hand-edit them:

```bash
deploy/k8s/overlays/pr/scripts/gen-topic-config.sh
deploy/k8s/overlays/pr/scripts/gen-db-name-suffix.sh
deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh
```

- [ ] **Step 8: Ingress and databases (doc §§5-6)**

- `deploy/shared/routes.conf`: add the `atlas-trades` location block,
  alphabetically placed, `http://atlas-trades:8080`.
- `./deploy/scripts/sync-k8s-ingress-routes.sh` and commit both files.
- `tools/db-bootstrap.sh`: add the **unsuffixed** `atlas-trades` to the `DBS`
  list.
- Note in the PR description that `atlas-trades-main` must be created manually
  on postgres.home — main has no wave-0 create job (doc §6.1).

- [ ] **Step 9: Verify registration**

```bash
tools/service-registration-guard.sh
kubectl kustomize deploy/k8s/overlays/main | grep -B2 -A8 "name: atlas-trades$"
kubectl kustomize deploy/k8s/overlays/pr > /dev/null
docker buildx bake atlas-trades
```

Expected: guard exits 0; the main render shows `DB_NAME=atlas-trades-main`,
`ATLAS_ENV=main`, and a pinned image; the PR render is clean; the image builds.
Confirm by eye that both new topic vars are present in each overlay's
`atlas-env` generator — the guard checks key *parity*, not that the right new
keys exist.

- [ ] **Step 10: Commit**

```bash
git add services/atlas-trades .github/config/services.json docker-bake.hcl go.work tools/db-bootstrap.sh deploy/
git commit -m "feat(task-205): register and bootstrap the atlas-trades service"
```

---

### Task 10: Trade domain model, builder and registry

**Files:**
- Create: `services/atlas-trades/atlas.com/trades/trade/model.go`, `trade/builder.go`, `trade/registry.go`
- Test: `services/atlas-trades/atlas.com/trades/trade/model_test.go`, `trade/registry_test.go`

**Interfaces:**
- Consumes: `field.Model` (`field.NewBuilder(w,c,m).SetInstance(uuid).Build()`), `tenant.Model`, `miniroom.Trade`/`miniroom.CashTrade`.
- Produces (consumed by Tasks 11-22):
  - `trade.State` string enum: `StateOpenSolo`, `StatePendingInvite`, `StateOpen`, `StateAwaitingAttestation`, `StateSettling`
  - `trade.Room` with getters `Id() uuid.UUID`, `Handle() uint32`, `RoomType() byte`, `Field() field.Model`, `State() State`, `Participants() []Participant`, `CreatedAt() time.Time`
  - `trade.Participant` with `CharacterId() uint32`, `Name() string`, `Position() byte`, `Confirmed() bool`, `Attested() bool`, `MesoStaged() uint32`, `Items() []StagedItem`
  - `trade.StagedItem` with `TradeSlot() byte`, `AssetId() uint32`, `TemplateId() uint32`, `Quantity() uint32`, `InventoryType() byte`, `SourceSlot() int16`
  - `trade.NewBuilder(roomType byte, ownerId uint32, ownerName string, f field.Model) *Builder` with `SetHandle`, `SetState`, `SetVisitor`, `Build() Room`
  - `trade.Room.WithState(State) Room`, `Room.WithParticipant(position byte, fn func(Participant) Participant) Room` — immutable transforms
  - `trade.GetRegistry() *Registry` with `Create(t, Room) error`, `Get(t, uuid.UUID) (Room, bool)`, `GetByMember(t, characterId uint32) (Room, bool)`, `GetByHandle(t, handle uint32) (Room, bool)`, `All(t) []Room`, `Update(t, id uuid.UUID, fn func(Room) (Room, error)) (Room, error)`, `Remove(t, id uuid.UUID)`
  - `trade.ErrOwnerHasRoom`, `trade.ErrRoomNotFound`, `trade.ErrRoomFull`, `trade.ErrRoomFrozen`

Design §2.3 (two ids: `uuid` for REST/registry, `uint32` handle = the owner's
character id for the wire serial), §3.1 (states), §12 (concurrency: one RWMutex,
member index maintained only inside Create/Update/Remove under the write lock;
state transitions are compare-and-set so two simultaneous confirms cannot both
trigger settlement).

- [ ] **Step 1: Write the failing registry tests**

Create `services/atlas-trades/atlas.com/trades/trade/registry_test.go`:

```go
package trade

import (
	"testing"

	"github.com/google/uuid"
)

// TestRegistryRejectsSecondRoomForOwner pins the authoritative single-room
// invariant (design §2.1): atlas-channel's cross-family check is best effort,
// but a character may hold at most one TRADE room, enforced here.
func TestRegistryRejectsSecondRoomForOwner(t *testing.T) {
	reg := &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	tm := testTenant(t)

	first := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).Build()
	if err := reg.Create(tm, first); err != nil {
		t.Fatalf("first create: %v", err)
	}

	second := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).Build()
	if err := reg.Create(tm, second); err != ErrOwnerHasRoom {
		t.Fatalf("second create: got %v, want ErrOwnerHasRoom", err)
	}
}

// TestRegistryIndexesBothParticipants pins that GetByMember resolves for the
// invited character too, not just the owner — the EXIT/logout teardown path
// looks a room up by whichever side acted.
func TestRegistryIndexesBothParticipants(t *testing.T) {
	reg := &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	tm := testTenant(t)

	room := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).SetVisitor(200, "Guest").Build()
	if err := reg.Create(tm, room); err != nil {
		t.Fatalf("create: %v", err)
	}

	for _, id := range []uint32{100, 200} {
		if _, ok := reg.GetByMember(tm, id); !ok {
			t.Errorf("GetByMember(%d): not found", id)
		}
	}
}

// TestRegistryRemoveClearsEveryIndex pins that teardown drops the room, both
// member entries and the wire handle — a stale index leaves the character
// permanently unable to open another trade.
func TestRegistryRemoveClearsEveryIndex(t *testing.T) {
	reg := &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	tm := testTenant(t)

	room := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).SetVisitor(200, "Guest").Build()
	_ = reg.Create(tm, room)
	reg.Remove(tm, room.Id())

	if _, ok := reg.Get(tm, room.Id()); ok {
		t.Error("room still present after Remove")
	}
	for _, id := range []uint32{100, 200} {
		if _, ok := reg.GetByMember(tm, id); ok {
			t.Errorf("member index still holds character %d", id)
		}
	}
	if _, ok := reg.GetByHandle(tm, 100); ok {
		t.Error("handle index still holds 100")
	}
}

// TestRegistryUpdateIsCompareAndSetOnState pins design §12: two simultaneous
// confirms must not both drive the room into SETTLING.
func TestRegistryUpdateIsCompareAndSetOnState(t *testing.T) {
	reg := &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	tm := testTenant(t)
	room := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).SetVisitor(200, "Guest").SetState(StateOpen).Build()
	_ = reg.Create(tm, room)

	transition := func(r Room) (Room, error) {
		if r.State() != StateOpen {
			return Room{}, ErrRoomFrozen
		}
		return r.WithState(StateSettling), nil
	}

	if _, err := reg.Update(tm, room.Id(), transition); err != nil {
		t.Fatalf("first transition: %v", err)
	}
	if _, err := reg.Update(tm, room.Id(), transition); err != ErrRoomFrozen {
		t.Fatalf("second transition: got %v, want ErrRoomFrozen", err)
	}
}

// TestRegistryTenantIsolation pins that tenant A cannot see tenant B's room.
func TestRegistryTenantIsolation(t *testing.T) {
	reg := &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	a, b := testTenant(t), testOtherTenant(t)

	room := NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).Build()
	_ = reg.Create(a, room)

	if _, ok := reg.Get(b, room.Id()); ok {
		t.Error("tenant B can see tenant A's room")
	}
	if _, ok := reg.GetByMember(b, 100); ok {
		t.Error("tenant B can resolve tenant A's member")
	}
	if err := reg.Create(b, NewBuilder(3, 100, "Owner", testField(t)).SetHandle(100).Build()); err != nil {
		t.Errorf("tenant B blocked by tenant A's occupancy: %v", err)
	}
}
```

Add `testTenant`, `testOtherTenant` and `testField` as small helpers at the
bottom of the test file (not a `*_testhelpers.go`), building a `tenant.Model`
and a `field.Model` from `field.NewBuilder(1, 1, 100000000).Build()`.

- [ ] **Step 2: Run them and confirm they fail**

Run: `cd services/atlas-trades/atlas.com/trades && go test ./trade/ -v`
Expected: FAIL — `undefined: Registry`.

- [ ] **Step 3: Write the model**

Create `services/atlas-trades/atlas.com/trades/trade/model.go`:

```go
package trade

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// State is the trade room lifecycle (design §3.1).
//
//	CREATE            -> OpenSolo
//	INVITE            -> PendingInvite
//	invite accepted   -> Open                  (staging happens here)
//	both confirmed    -> AwaitingAttestation   (mode 17 broadcast, 5s deadline)
//	both attested     -> Settling              (saga in flight; cancels lose)
type State string

const (
	StateOpenSolo            State = "OPEN_SOLO"
	StatePendingInvite       State = "PENDING_INVITE"
	StateOpen                State = "OPEN"
	StateAwaitingAttestation State = "AWAITING_ATTESTATION"
	StateSettling            State = "SETTLING"
)

// StagedItem is one item claimed for trade. Under the reserve-at-staging model
// (design §5.3) the asset is STILL IN the owner's inventory, held by an
// atlas-inventory reservation; only settlement moves it.
type StagedItem struct {
	tradeSlot     byte
	assetId       uint32
	templateId    uint32
	quantity      uint32
	inventoryType byte
	sourceSlot    int16
}

func (s StagedItem) TradeSlot() byte     { return s.tradeSlot }
func (s StagedItem) AssetId() uint32     { return s.assetId }
func (s StagedItem) TemplateId() uint32  { return s.templateId }
func (s StagedItem) Quantity() uint32    { return s.quantity }
func (s StagedItem) InventoryType() byte { return s.inventoryType }
func (s StagedItem) SourceSlot() int16   { return s.sourceSlot }

// Participant is one side of the trade. position 0 is the room owner, 1 the
// invited character; position drives which side of the client dialog receives
// which update (FR-1.5).
type Participant struct {
	characterId uint32
	name        string
	position    byte
	confirmed   bool
	attested    bool
	mesoStaged  uint32
	items       []StagedItem
}

func (p Participant) CharacterId() uint32  { return p.characterId }
func (p Participant) Name() string         { return p.name }
func (p Participant) Position() byte       { return p.position }
func (p Participant) Confirmed() bool      { return p.confirmed }
func (p Participant) Attested() bool       { return p.attested }
func (p Participant) MesoStaged() uint32   { return p.mesoStaged }
func (p Participant) Items() []StagedItem  { return p.items }

// Room is a live trade room. It carries two ids (design §2.3): Id is the REST
// identity and registry key, Handle is the uint32 wire serial the client's
// invite carries (invite.CreateCommandBody.ReferenceId is invite.Id = uint32,
// so a uuid does not fit). Handle is set to the owner's character id, matching
// the existing mini-room convention in atlas-channel.
type Room struct {
	id           uuid.UUID
	handle       uint32
	roomType     byte
	f            field.Model
	state        State
	participants []Participant
	createdAt    time.Time
}

func (r Room) Id() uuid.UUID              { return r.id }
func (r Room) Handle() uint32             { return r.handle }
func (r Room) RoomType() byte             { return r.roomType }
func (r Room) Field() field.Model         { return r.f }
func (r Room) State() State               { return r.state }
func (r Room) Participants() []Participant { return r.participants }
func (r Room) CreatedAt() time.Time       { return r.createdAt }

// OwnerId returns position 0's character id.
func (r Room) OwnerId() uint32 {
	for _, p := range r.participants {
		if p.position == 0 {
			return p.characterId
		}
	}
	return 0
}

// VisitorId returns position 1's character id, or 0 when the room is solo.
func (r Room) VisitorId() uint32 {
	for _, p := range r.participants {
		if p.position == 1 {
			return p.characterId
		}
	}
	return 0
}

// ParticipantFor returns the participant acting as characterId.
func (r Room) ParticipantFor(characterId uint32) (Participant, bool) {
	for _, p := range r.participants {
		if p.characterId == characterId {
			return p, true
		}
	}
	return Participant{}, false
}

// Frozen reports whether staging is closed. From the moment the FIRST side
// confirms, the room rejects PUT_ITEM, ADD_MESO and any further CONFIRM from
// either side (FR-3.6, design §3.2).
func (r Room) Frozen() bool {
	if r.state != StateOpen {
		return true
	}
	for _, p := range r.participants {
		if p.confirmed {
			return true
		}
	}
	return false
}

// WithState returns a copy of r in the given state.
func (r Room) WithState(s State) Room {
	c := r
	c.state = s
	return c
}

// WithParticipant returns a copy of r with the participant at `position`
// replaced by fn's result. Participants are value types, so fn receives and
// returns a copy — this is the only way room state mutates.
func (r Room) WithParticipant(position byte, fn func(Participant) Participant) Room {
	c := r
	c.participants = make([]Participant, len(r.participants))
	copy(c.participants, r.participants)
	for i := range c.participants {
		if c.participants[i].position == position {
			c.participants[i] = fn(c.participants[i])
		}
	}
	return c
}
```

Add the `Participant` transforms used by Tasks 18-19 in the same file:

```go
func (p Participant) WithConfirmed(v bool) Participant  { c := p; c.confirmed = v; return c }
func (p Participant) WithAttested(v bool) Participant   { c := p; c.attested = v; return c }
func (p Participant) WithMesoStaged(v uint32) Participant { c := p; c.mesoStaged = v; return c }

// WithItem appends a staged item. Callers must have already rejected a
// duplicate trade slot and enforced the maxStagedItems cap.
func (p Participant) WithItem(i StagedItem) Participant {
	c := p
	c.items = make([]StagedItem, len(p.items), len(p.items)+1)
	copy(c.items, p.items)
	c.items = append(c.items, i)
	return c
}

// HasTradeSlot reports whether the given 1..9 trade slot is already occupied
// (FR-3.3).
func (p Participant) HasTradeSlot(slot byte) bool {
	for _, i := range p.items {
		if i.tradeSlot == slot {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Write the builder**

Create `services/atlas-trades/atlas.com/trades/trade/builder.go`:

```go
package trade

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Builder constructs a Room. Use it in tests too — this project does not use
// *_testhelpers.go constructors.
type Builder struct {
	id           uuid.UUID
	handle       uint32
	roomType     byte
	f            field.Model
	state        State
	participants []Participant
	createdAt    time.Time
}

// NewBuilder starts a solo room owned by ownerId at position 0. handle defaults
// to ownerId (design §2.3); override with SetHandle only in tests.
func NewBuilder(roomType byte, ownerId uint32, ownerName string, f field.Model) *Builder {
	return &Builder{
		id:       uuid.New(),
		handle:   ownerId,
		roomType: roomType,
		f:        f,
		state:    StateOpenSolo,
		participants: []Participant{
			{characterId: ownerId, name: ownerName, position: 0, items: []StagedItem{}},
		},
		createdAt: time.Now(),
	}
}

func (b *Builder) SetId(id uuid.UUID) *Builder    { b.id = id; return b }
func (b *Builder) SetHandle(h uint32) *Builder    { b.handle = h; return b }
func (b *Builder) SetState(s State) *Builder      { b.state = s; return b }
func (b *Builder) SetCreatedAt(t time.Time) *Builder { b.createdAt = t; return b }

// SetVisitor seats the invited character at position 1.
func (b *Builder) SetVisitor(characterId uint32, name string) *Builder {
	b.participants = append(b.participants, Participant{
		characterId: characterId, name: name, position: 1, items: []StagedItem{},
	})
	return b
}

func (b *Builder) Build() Room {
	return Room{
		id:           b.id,
		handle:       b.handle,
		roomType:     b.roomType,
		f:            b.f,
		state:        b.state,
		participants: b.participants,
		createdAt:    b.createdAt,
	}
}
```

- [ ] **Step 5: Write the registry**

Create `services/atlas-trades/atlas.com/trades/trade/registry.go`, mirroring
`services/atlas-mini-games/atlas.com/mini-games/game/registry.go:1-164` with a
third index (`handles`) because the wire serial and the registry key differ:

```go
package trade

import (
	"errors"
	"sync"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var (
	// ErrOwnerHasRoom is returned by Create when a participant already occupies
	// another trade room for the tenant.
	ErrOwnerHasRoom = errors.New("trade: character already has a room")
	// ErrRoomNotFound is returned by Update when the room id is unknown.
	ErrRoomNotFound = errors.New("trade: room not found")
	// ErrRoomFull is returned when a second character tries to enter an
	// already-paired room.
	ErrRoomFull = errors.New("trade: room is full")
	// ErrRoomFrozen is returned when a staging or transition action arrives
	// after the room left the state that permits it (FR-3.6, design §3.2).
	ErrRoomFrozen = errors.New("trade: room is frozen")
)

// Registry is the tenant-partitioned in-memory store of trade rooms. One
// RWMutex guards all three maps; the member and handle indexes are maintained
// ONLY inside Create/Update/Remove, always alongside the room mutation, under
// the write lock. The registry is process-local, which is why atlas-trades runs
// replicas: 1 (design §9).
type Registry struct {
	mutex   sync.RWMutex
	rooms   map[tenant.Model]map[uuid.UUID]Room
	members map[tenant.Model]map[uint32]uuid.UUID
	handles map[tenant.Model]map[uint32]uuid.UUID
}

var (
	registry *Registry
	once     sync.Once
)

func newRoomMap() map[tenant.Model]map[uuid.UUID]Room     { return make(map[tenant.Model]map[uuid.UUID]Room) }
func newMemberMap() map[tenant.Model]map[uint32]uuid.UUID { return make(map[tenant.Model]map[uint32]uuid.UUID) }
func newHandleMap() map[tenant.Model]map[uint32]uuid.UUID { return make(map[tenant.Model]map[uint32]uuid.UUID) }

// GetRegistry returns the process-wide Registry singleton.
func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{rooms: newRoomMap(), members: newMemberMap(), handles: newHandleMap()}
	})
	return registry
}

// Create registers r for tenant t, failing with ErrOwnerHasRoom if any of r's
// participants already occupies a room.
func (reg *Registry) Create(t tenant.Model, r Room) error {
	reg.mutex.Lock()
	defer reg.mutex.Unlock()

	for _, p := range r.Participants() {
		if _, ok := reg.members[t][p.CharacterId()]; ok {
			return ErrOwnerHasRoom
		}
	}

	if reg.rooms[t] == nil {
		reg.rooms[t] = make(map[uuid.UUID]Room)
	}
	if reg.members[t] == nil {
		reg.members[t] = make(map[uint32]uuid.UUID)
	}
	if reg.handles[t] == nil {
		reg.handles[t] = make(map[uint32]uuid.UUID)
	}

	reg.rooms[t][r.Id()] = r
	reg.index(t, r)
	return nil
}

func (reg *Registry) Get(t tenant.Model, id uuid.UUID) (Room, bool) {
	reg.mutex.RLock()
	defer reg.mutex.RUnlock()
	r, ok := reg.rooms[t][id]
	return r, ok
}

// GetByMember resolves the room characterId occupies, as either side.
func (reg *Registry) GetByMember(t tenant.Model, characterId uint32) (Room, bool) {
	reg.mutex.RLock()
	defer reg.mutex.RUnlock()
	id, ok := reg.members[t][characterId]
	if !ok {
		return Room{}, false
	}
	r, ok := reg.rooms[t][id]
	return r, ok
}

// GetByHandle resolves a room from the uint32 wire serial an invite carries.
func (reg *Registry) GetByHandle(t tenant.Model, handle uint32) (Room, bool) {
	reg.mutex.RLock()
	defer reg.mutex.RUnlock()
	id, ok := reg.handles[t][handle]
	if !ok {
		return Room{}, false
	}
	r, ok := reg.rooms[t][id]
	return r, ok
}

// All returns every live room for tenant t (the REST list read).
func (reg *Registry) All(t tenant.Model) []Room {
	reg.mutex.RLock()
	defer reg.mutex.RUnlock()
	out := make([]Room, 0, len(reg.rooms[t]))
	for _, r := range reg.rooms[t] {
		out = append(out, r)
	}
	return out
}

// Update mutates the room under a single write lock: fn receives the current
// Room and returns its replacement. A non-nil error from fn leaves the room
// untouched and is returned as-is — this is how state transitions are made
// compare-and-set (design §12), so two simultaneous confirms cannot both
// trigger settlement.
func (reg *Registry) Update(t tenant.Model, id uuid.UUID, fn func(Room) (Room, error)) (Room, error) {
	reg.mutex.Lock()
	defer reg.mutex.Unlock()

	cur, ok := reg.rooms[t][id]
	if !ok {
		return Room{}, ErrRoomNotFound
	}

	updated, err := fn(cur)
	if err != nil {
		return Room{}, err
	}

	reg.deindex(t, cur)
	reg.rooms[t][updated.Id()] = updated
	reg.index(t, updated)
	return updated, nil
}

// Remove deletes the room and clears every index entry it owns. A missing id is
// a no-op.
func (reg *Registry) Remove(t tenant.Model, id uuid.UUID) {
	reg.mutex.Lock()
	defer reg.mutex.Unlock()

	r, ok := reg.rooms[t][id]
	if !ok {
		return
	}
	delete(reg.rooms[t], id)
	reg.deindex(t, r)
}

// index records every participant and the wire handle. Callers hold the write
// lock and have ensured the per-tenant maps are non-nil.
func (reg *Registry) index(t tenant.Model, r Room) {
	for _, p := range r.Participants() {
		reg.members[t][p.CharacterId()] = r.Id()
	}
	reg.handles[t][r.Handle()] = r.Id()
}

// deindex removes every participant and the wire handle. Callers hold the write
// lock.
func (reg *Registry) deindex(t tenant.Model, r Room) {
	for _, p := range r.Participants() {
		delete(reg.members[t], p.CharacterId())
	}
	delete(reg.handles[t], r.Handle())
}
```

- [ ] **Step 6: Run the tests**

```bash
cd services/atlas-trades/atlas.com/trades && go test -race ./trade/ -v && go vet ./...
```

Expected: PASS, including under `-race`.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-trades
git commit -m "feat(task-205): add the trade domain model, builder and room registry"
```

---

### Task 11: Completed-trade ledger (entity, model, provider, administrator)

**Files:**
- Create: `services/atlas-trades/atlas.com/trades/ledger/entity.go`, `ledger/model.go`, `ledger/builder.go`, `ledger/provider.go`, `ledger/administrator.go`, `ledger/processor.go`, `ledger/mock/processor.go`
- Modify: `services/atlas-trades/atlas.com/trades/main.go` (register `ledger.Migration`)
- Test: `services/atlas-trades/atlas.com/trades/ledger/administrator_test.go`, `ledger/provider_test.go`

**Interfaces:**
- Consumes: `trade.Room`, `trade.Participant`, `trade.StagedItem` (Task 10).
- Produces (consumed by Tasks 12 and 19):
  - `ledger.Migration(db *gorm.DB) error`
  - `ledger.Processor` interface with `Record(mb *message.Buffer) func(e Model) error`, `GetById(id uuid.UUID) (Model, error)`, `GetByCharacterId(characterId uint32, from, to time.Time) ([]Model, error)`
  - `ledger.NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor`
  - `ledger.NewBuilder(transactionId uuid.UUID, f field.Model, roomType byte) *Builder` with `AddSide(characterId uint32, name string, mesoStaged, mesoTax, mesoDelivered uint32, items []Item) *Builder`, `Build() Model`
  - `ledger.ErrDuplicateTransaction`

PRD §6 three-table model, plus design §9's two additions: a unique index on
`(tenant_id, transaction_id)` as the write-side idempotency guard (FR-5.7), and
`room_type` on the entry only (both sides always share it).

- [ ] **Step 1: Write the failing administrator test**

Create `services/atlas-trades/atlas.com/trades/ledger/administrator_test.go`:

```go
package ledger

import (
	"testing"

	"github.com/google/uuid"
)

// TestCreateIsIdempotentPerTransaction pins FR-5.7 / design §9: a duplicate
// settle hits the unique (tenant_id, transaction_id) index and returns success
// without writing a second entry, so a retried saga never double-records.
func TestCreateIsIdempotentPerTransaction(t *testing.T) {
	db := testDb(t)
	txId := uuid.New()

	entry := NewBuilder(txId, testField(t), 3).
		AddSide(100, "Alice", 10_000_000, 400_000, 0, nil).
		AddSide(200, "Bob", 0, 0, 9_600_000, nil).
		Build()

	first, err := create(db, testTenantId(t))(entry)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	second, err := create(db, testTenantId(t))(entry)
	if err != nil {
		t.Fatalf("duplicate create must succeed, got: %v", err)
	}
	if second.Id() != first.Id() {
		t.Errorf("duplicate create wrote a new row: got %s, want %s", second.Id(), first.Id())
	}

	all, err := allEntries(db, testTenantId(t))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("entry count: got %d, want 1", len(all))
	}
}

// TestCreateWritesExactlyTwoSides pins PRD §6: an entry always has exactly two
// sides, with their items attached.
func TestCreateWritesExactlyTwoSides(t *testing.T) {
	db := testDb(t)
	entry := NewBuilder(uuid.New(), testField(t), 3).
		AddSide(100, "Alice", 0, 0, 0, []Item{{ItemId: 2000000, Quantity: 5}}).
		AddSide(200, "Bob", 0, 0, 0, nil).
		Build()

	stored, err := create(db, testTenantId(t))(entry)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(stored.Sides()) != 2 {
		t.Fatalf("sides: got %d, want 2", len(stored.Sides()))
	}
	if len(stored.Sides()[0].Items()) != 1 {
		t.Errorf("side 0 items: got %d, want 1", len(stored.Sides()[0].Items()))
	}
}

// TestGetByCharacterIdMatchesEitherSide pins PRD FR-7.2: the GM lookup finds a
// trade whether the character gave or received.
func TestGetByCharacterIdMatchesEitherSide(t *testing.T) {
	db := testDb(t)
	entry := NewBuilder(uuid.New(), testField(t), 3).
		AddSide(100, "Alice", 0, 0, 0, nil).
		AddSide(200, "Bob", 0, 0, 0, nil).
		Build()
	_, _ = create(db, testTenantId(t))(entry)

	for _, id := range []uint32{100, 200} {
		found, err := byCharacter(db, testTenantId(t))(id, time.Time{}, time.Now().Add(time.Hour))
		if err != nil {
			t.Fatalf("lookup %d: %v", id, err)
		}
		if len(found) != 1 {
			t.Errorf("lookup %d: got %d entries, want 1", id, len(found))
		}
	}
}
```

Add `testDb`, `testTenantId` and `testField` helpers at the bottom of the file,
using the in-memory/sqlite or dockertest scaffolding the sibling services'
administrator tests already use — copy the shape from
`services/atlas-mini-games/atlas.com/mini-games/record/administrator_test.go`.

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd services/atlas-trades/atlas.com/trades && go test ./ledger/ -v`
Expected: FAIL — `undefined: NewBuilder`.

- [ ] **Step 3: Write the entities**

Create `services/atlas-trades/atlas.com/trades/ledger/entity.go`:

```go
package ledger

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entry is one settled trade. The unique (tenant_id, transaction_id) index is
// the write-side idempotency guard for FR-5.7: a duplicate settle hits the
// constraint and returns the existing row rather than double-recording.
//
// room_type lives here and is deliberately NOT denormalised onto Side — both
// sides of a trade always share it (design §9).
type Entry struct {
	Id            uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId      uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_trade_entry_tenant_tx"`
	TransactionId uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_trade_entry_tenant_tx"`
	WorldId       byte      `gorm:"not null"`
	ChannelId     byte      `gorm:"not null"`
	MapId         uint32    `gorm:"not null"`
	RoomType      byte      `gorm:"not null"`
	SettledAt     time.Time `gorm:"not null;index"`
	Sides         []Side    `gorm:"foreignKey:EntryId"`
}

func (Entry) TableName() string { return "trade_ledger_entries" }

// Side is one participant's contribution. Exactly two rows per Entry.
// CharacterName is denormalised because names change and the ledger is a
// point-in-time record.
type Side struct {
	Id             uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId       uuid.UUID `gorm:"type:uuid;not null;index"`
	EntryId        uuid.UUID `gorm:"type:uuid;not null;index"`
	CharacterId    uint32    `gorm:"not null;index"`
	CharacterName  string    `gorm:"not null"`
	MesoStaged     uint32    `gorm:"not null"`
	MesoTax        uint32    `gorm:"not null"`
	MesoDelivered  uint32    `gorm:"not null"`
	Items          []ItemRow `gorm:"foreignKey:SideId"`
}

func (Side) TableName() string { return "trade_ledger_sides" }

// ItemRow is one asset a side gave. AssetId and ReferenceId are nullable
// because only identity-bearing assets (equips, pets, cash) have them.
type ItemRow struct {
	Id          uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId    uuid.UUID `gorm:"type:uuid;not null;index"`
	SideId      uuid.UUID `gorm:"type:uuid;not null;index"`
	ItemId      uint32    `gorm:"not null"`
	Quantity    uint32    `gorm:"not null"`
	AssetId     *uint32
	ReferenceId *uint32
}

func (ItemRow) TableName() string { return "trade_ledger_items" }

// Migration creates the three ledger tables. Fresh tables, no backfill.
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entry{}, &Side{}, &ItemRow{})
}
```

- [ ] **Step 4: Write the model, builder, provider, administrator and processor**

`ledger/model.go` — immutable `Model`, `SideModel`, `Item` with private fields +
getters, plus `Make(e Entry) (Model, error)` mapping entity → model. Follow the
shape of `services/atlas-mini-games/atlas.com/mini-games/record/model.go`.

`ledger/builder.go` — `NewBuilder(transactionId uuid.UUID, f field.Model,
roomType byte) *Builder`, `AddSide(...)`, `SetSettledAt(time.Time)`, `Build()
Model`. `SettledAt` defaults to `time.Now()`.

`ledger/provider.go` — `byIdProvider(db, tenantId, id)`,
`byCharacterProvider(db, tenantId, characterId, from, to)` using
`model.SliceProvider` and GORM `Preload("Sides.Items")`. Every query
`tenant_id`-scoped.

`ledger/administrator.go` — `create(db *gorm.DB, tenantId uuid.UUID) func(m
Model) (Model, error)`, wrapping the insert in `database.ExecuteTransaction` and
translating a unique-violation on `idx_trade_entry_tenant_tx` into a read of the
existing row:

```go
// create writes the entry, its two sides and their items in one transaction.
// A unique-constraint violation on (tenant_id, transaction_id) is NOT an error:
// it means this settlement already recorded (a retried saga), so we read the
// existing row back and return it. This is the FR-5.7 idempotency guard.
func create(db *gorm.DB, tenantId uuid.UUID) func(m Model) (Model, error) {
	return func(m Model) (Model, error) {
		var out Model
		err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			e := toEntity(tenantId, m)
			if err := tx.Create(&e).Error; err != nil {
				if !database.IsUniqueViolation(err) {
					return err
				}
				existing, rerr := byTransactionId(tx, tenantId, m.TransactionId())
				if rerr != nil {
					return rerr
				}
				out = existing
				return nil
			}
			stored, merr := Make(e)
			if merr != nil {
				return merr
			}
			out = stored
			return nil
		})
		return out, err
	}
}
```

If `database.IsUniqueViolation` does not exist in `libs/atlas-database`, match on
the Postgres `23505` SQLSTATE the same way the library's existing error
classifiers do, in a small unexported helper in this file.

`ledger/processor.go` — `Processor` interface + `ProcessorImpl` +
`NewProcessor(l, ctx, db)` + `var _ Processor = (*ProcessorImpl)(nil)`, resolving
the tenant with `tenant.MustFromContext(ctx)` and delegating to the
provider/administrator functions.

`ledger/mock/processor.go` — a struct of function fields implementing
`Processor`, following `services/atlas-mini-games/atlas.com/mini-games/record/mock/processor.go`.

- [ ] **Step 5: Register the migration**

In `services/atlas-trades/atlas.com/trades/main.go`:

```go
	db := database.Connect(l, database.SetMigrations(ledger.Migration, outboxlib.Migration))
```

- [ ] **Step 6: Run the tests**

```bash
cd services/atlas-trades/atlas.com/trades && go test -race ./ledger/ -v && go vet ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-trades
git commit -m "feat(task-205): add the completed-trade ledger with per-transaction idempotency"
```

---

### Task 12: REST surface — rooms and ledger

**Files:**
- Create: `services/atlas-trades/atlas.com/trades/trade/rest.go`, `trade/resource.go`, `trade/processor.go` (read-only methods only at this task)
- Create: `services/atlas-trades/atlas.com/trades/ledger/rest.go`, `ledger/resource.go`
- Modify: `services/atlas-trades/atlas.com/trades/main.go` (route initializers)
- Test: `services/atlas-trades/atlas.com/trades/trade/resource_test.go`, `ledger/resource_test.go`

**Interfaces:**
- Consumes: `trade.GetRegistry()` (Task 10), `ledger.Processor` (Task 11).
- Produces:
  - `trade.RestModel` with `GetName() string { return "rooms" }`
  - `ledger.RestModel` with `GetName() string { return "ledgerEntries" }`
  - `trade.InitResource(si jsonapi.ServerInformation) server.RouteInitializer`
  - `ledger.InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer`
  - `trade.Processor.RoomsForTenant() []Room`, `trade.Processor.RoomById(id uuid.UUID) (Room, bool)` — the REST reads go through the processor, never the registry directly (DOM-14).

PRD §5: four read-only endpoints, JSON:API via api2go, tenant-scoped, page size
capped at 100. No POST/PATCH/DELETE — rooms are Kafka-driven and ledger rows are
immutable (FR-7.4).

- [ ] **Step 1: Write the failing resource tests**

Create `services/atlas-trades/atlas.com/trades/trade/resource_test.go`:

```go
package trade

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetRoomByIdReturns404WhenGone pins PRD §5: a settled or cancelled room is
// gone, and the endpoint must 404 rather than serve a stale snapshot.
func TestGetRoomByIdReturns404WhenGone(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/trades/rooms/"+uuid.New().String(), nil)
	req.Header.Set("TENANT_ID", testTenantId(t).String())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", rec.Code)
	}
}

// TestGetRoomsIsTenantScoped pins the NFR: a room in tenant A must be invisible
// from tenant B.
func TestGetRoomsIsTenantScoped(t *testing.T) {
	router := testRouter(t)
	room := NewBuilder(3, 100, "Owner", testField(t)).Build()
	_ = GetRegistry().Create(testTenant(t), room)
	t.Cleanup(func() { GetRegistry().Remove(testTenant(t), room.Id()) })

	req := httptest.NewRequest(http.MethodGet, "/api/trades/rooms", nil)
	req.Header.Set("TENANT_ID", testOtherTenantId(t).String())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); strings.Contains(body, room.Id().String()) {
		t.Errorf("tenant B sees tenant A's room: %s", body)
	}
}

// TestGetRoomsRejectsOversizePage pins PRD §5's error table: a page size above
// the cap is a 400, not a silent clamp.
func TestGetRoomsRejectsOversizePage(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/trades/rooms?page[size]=500", nil)
	req.Header.Set("TENANT_ID", testTenantId(t).String())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rec.Code)
	}
}
```

Write the analogous `ledger/resource_test.go` covering: `GET /trades/ledger`
filtered by `filter[characterId]` matching either side, `filter[from]`/
`filter[to]` RFC3339 parsing, a malformed filter producing 400, and
`GET /trades/ledger/{entryId}` 404ing an unknown id.

- [ ] **Step 2: Run them and confirm they fail**

Run: `cd services/atlas-trades/atlas.com/trades && go test ./trade/ ./ledger/ -run Resource -v`
Expected: FAIL — `undefined: testRouter` / `undefined: InitResource`.

- [ ] **Step 3: Write the REST models**

`trade/rest.go`:

```go
package trade

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// RestModel is the JSON:API projection of a live trade room. Resource type
// "rooms" (PRD §5). Read-only: rooms are driven exclusively by Kafka commands.
type RestModel struct {
	Id           string                  `json:"-"`
	RoomType     byte                    `json:"roomType"`
	WorldId      byte                    `json:"worldId"`
	ChannelId    byte                    `json:"channelId"`
	MapId        uint32                  `json:"mapId"`
	State        string                  `json:"state"`
	Participants []ParticipantRestModel  `json:"participants"`
	CreatedAt    time.Time               `json:"createdAt"`
}

type ParticipantRestModel struct {
	CharacterId uint32                `json:"characterId"`
	Position    byte                  `json:"position"`
	Confirmed   bool                  `json:"confirmed"`
	MesoStaged  uint32                `json:"mesoStaged"`
	Items       []StagedItemRestModel `json:"items"`
}

type StagedItemRestModel struct {
	TradeSlot  byte   `json:"tradeSlot"`
	ItemId     uint32 `json:"itemId"`
	Quantity   uint32 `json:"quantity"`
	AssetId    uint32 `json:"assetId"`
}

func (r RestModel) GetID() string          { return r.Id }
func (r *RestModel) SetID(id string) error { r.Id = id; return nil }
func (r RestModel) GetName() string        { return "rooms" }

// Transform projects a Room into its REST shape.
func Transform(m Room) (RestModel, error) {
	participants := make([]ParticipantRestModel, 0, len(m.Participants()))
	for _, p := range m.Participants() {
		items := make([]StagedItemRestModel, 0, len(p.Items()))
		for _, i := range p.Items() {
			items = append(items, StagedItemRestModel{
				TradeSlot: i.TradeSlot(),
				ItemId:    i.TemplateId(),
				Quantity:  i.Quantity(),
				AssetId:   i.AssetId(),
			})
		}
		participants = append(participants, ParticipantRestModel{
			CharacterId: p.CharacterId(),
			Position:    p.Position(),
			Confirmed:   p.Confirmed(),
			MesoStaged:  p.MesoStaged(),
			Items:       items,
		})
	}
	return RestModel{
		Id:           m.Id().String(),
		RoomType:     m.RoomType(),
		WorldId:      byte(m.Field().WorldId()),
		ChannelId:    byte(m.Field().ChannelId()),
		MapId:        uint32(m.Field().MapId()),
		State:        string(m.State()),
		Participants: participants,
		CreatedAt:    m.CreatedAt(),
	}, nil
}
```

`ledger/rest.go` — the same shape for resource type `ledgerEntries`, matching
PRD §5's attribute table (`transactionId`, `worldId`, `channelId`, `mapId`,
`roomType`, `settledAt`, `sides[]` with `{characterId, characterName,
mesoStaged, mesoTax, mesoDelivered, items[]}`, items being
`{itemId, quantity, assetId, referenceId}`).

- [ ] **Step 4: Write the resources**

`trade/resource.go`, mirroring
`services/atlas-mini-games/atlas.com/mini-games/game/resource.go:36-122`:

```go
const (
	GetRooms  = "get_trade_rooms"
	GetRoomById = "get_trade_room_by_id"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		r := router.PathPrefix("/trades/rooms").Subrouter()
		r.HandleFunc("", registerGet(GetRooms, handleGetRooms())).Methods(http.MethodGet)
		r.HandleFunc("/{roomId}", registerGet(GetRoomById, handleGetRoomById())).Methods(http.MethodGet)
	}
}
```

`handleGetRooms` applies the PRD's filters (`filter[characterId]`,
`filter[worldId]`, `filter[channelId]`, `filter[mapId]`), sorts deterministically
by `Id()` before paginating (the registry map iteration order is random), then
`paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, 100)`,
`paginate.Slice`, `server.MarshalPaginatedResponse`. A page size above 100 must
produce 400 — pass 100 as the cap, not `paginate.MaxPageSize`.

`handleGetRoomById` wraps `rest.ParseRoomId` and 404s a miss.

`ledger/resource.go` — same shape, `InitResource(si)(db)`, with
`GET /trades/ledger` and `GET /trades/ledger/{entryId}`.

- [ ] **Step 5: Add the read-only processor methods**

Create `services/atlas-trades/atlas.com/trades/trade/processor.go` with only the
reads for now (Tasks 17-19 add the commands to the same interface):

```go
// Processor owns every trade-room operation. REST handlers go through it rather
// than reaching into the registry directly (DOM-14).
type Processor interface {
	RoomsForTenant() []Room
	RoomById(id uuid.UUID) (Room, bool)
	RoomForCharacter(characterId uint32) (Room, bool)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	reg *Registry
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, t: tenant.MustFromContext(ctx), reg: GetRegistry()}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) RoomsForTenant() []Room { return p.reg.All(p.t) }

func (p *ProcessorImpl) RoomById(id uuid.UUID) (Room, bool) { return p.reg.Get(p.t, id) }

func (p *ProcessorImpl) RoomForCharacter(characterId uint32) (Room, bool) {
	return p.reg.GetByMember(p.t, characterId)
}
```

- [ ] **Step 6: Mount the routes**

In `main.go`'s `server.New(l)` chain, before the readiness mount:

```go
		AddRouteInitializer(trade.InitResource(GetServer())).
		AddRouteInitializer(ledger.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
```

- [ ] **Step 7: Run the tests**

```bash
cd services/atlas-trades/atlas.com/trades && go test -race ./... -v && go vet ./... && go build ./...
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-trades
git commit -m "feat(task-205): add the read-only rooms and ledger REST surface"
```

---

### Task 13: Tenant `trade-configs` resource and the tax calculator

**Files:**
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/resource.go` (six handlers + route registration), `configuration/rest.go` (REST model + Transform + Extract), `configuration/processor.go` (twelve methods), `configuration/provider.go`, `configuration/kafka.go` (three event types + provider), `configuration/seed.go` (path + loader)
- Modify: `services/atlas-tenants/atlas.com/tenants/rest/handler.go` (`ParseTradeConfigId`)
- Create: `services/atlas-tenants/configurations/trade-configs/default.json`
- Create: `services/atlas-trades/atlas.com/trades/configuration/model.go`, `configuration/registry.go`, `configuration/requests.go`, `configuration/rest.go`, `configuration/tax.go`
- Test: `services/atlas-trades/atlas.com/trades/configuration/tax_test.go`, `configuration/model_test.go`

**Interfaces:**
- Produces (consumed by Tasks 18-19):
  - `configuration.Model` with `TaxEnabled() bool`, `TaxTiers() []Tier`, `MaxStagedItems() int`, `MinTradeLevel() int`, `ReservationTtl() time.Duration`, `AttestationTimeout() time.Duration`
  - `configuration.Tier{Threshold uint32; Rate float64}`
  - `configuration.DefaultConfig() Model`
  - `configuration.GetRegistry().Get(l, ctx) Model` — fetches once per tenant, falls back to defaults
  - `configuration.Tax(m Model, amount uint32) (tax uint32, delivered uint32)`

Design §8. Mirror the `mts-configs` precedent in atlas-tenants
(`configuration/resource.go:817-1026`, `rest.go:247-379`,
`processor.go:112-155`, `provider.go:157,191`, `kafka.go:25-27,65-66`,
`seed.go`). No migration is needed — configuration is one polymorphic JSONB table
keyed by a `ResourceName` string discriminator.

> **Trap (context.md §1.12):** `services/atlas-tenants/configurations/` currently
> ships only `rps-rewards/`, so `SeedMtsConfigs` fails at runtime for want of a
> seed directory. Ship `trade-configs/default.json` **and** make the no-config
> fallback the tested path, per design §8.

- [ ] **Step 1: Write the failing tax tests**

Create `services/atlas-trades/atlas.com/trades/configuration/tax_test.go`:

```go
package configuration

import "testing"

// TestTaxWorkedExample pins the PRD's acceptance criterion: 10,000,000 staged
// delivers 9,600,000 at the 4% tier, and the 400,000 difference is destroyed —
// it is credited to nobody.
func TestTaxWorkedExample(t *testing.T) {
	tax, delivered := Tax(DefaultConfig(), 10_000_000)
	if tax != 400_000 {
		t.Errorf("tax: got %d, want 400000", tax)
	}
	if delivered != 9_600_000 {
		t.Errorf("delivered: got %d, want 9600000", delivered)
	}
	if tax+delivered != 10_000_000 {
		t.Errorf("tax+delivered: got %d, want 10000000", tax+delivered)
	}
}

// TestTaxTierBoundariesFromBothSides pins that each threshold is inclusive and
// the tier below it applies one meso lower.
func TestTaxTierBoundariesFromBothSides(t *testing.T) {
	cases := []struct {
		amount   uint32
		wantRate float64
	}{
		{100_000_000, 0.060}, {99_999_999, 0.050},
		{25_000_000, 0.050}, {24_999_999, 0.040},
		{10_000_000, 0.040}, {9_999_999, 0.030},
		{5_000_000, 0.030}, {4_999_999, 0.018},
		{1_000_000, 0.018}, {999_999, 0.008},
		{100_000, 0.008}, {99_999, 0.0},
		{0, 0.0},
	}
	for _, c := range cases {
		tax, delivered := Tax(DefaultConfig(), c.amount)
		want := uint32(float64(c.amount) * c.wantRate)
		if tax != want {
			t.Errorf("amount %d: tax got %d, want %d (rate %.3f)", c.amount, tax, want, c.wantRate)
		}
		if tax+delivered != c.amount {
			t.Errorf("amount %d: tax+delivered got %d, want %d", c.amount, tax+delivered, c.amount)
		}
	}
}

// TestTaxDisabledDeductsNothing pins FR-9.1's master switch.
func TestTaxDisabledDeductsNothing(t *testing.T) {
	m := DefaultConfig().WithTaxEnabled(false)
	tax, delivered := Tax(m, 100_000_000)
	if tax != 0 {
		t.Errorf("tax: got %d, want 0", tax)
	}
	if delivered != 100_000_000 {
		t.Errorf("delivered: got %d, want 100000000", delivered)
	}
}

// TestInvalidTierTableFallsBackToDefaults pins FR-9.3 / design §6.5: a table
// whose thresholds are not strictly descending, or whose rate is outside [0,1],
// is rejected LOUDLY and the shipped defaults are used — never a silent accept.
func TestInvalidTierTableFallsBackToDefaults(t *testing.T) {
	cases := map[string][]Tier{
		"ascending thresholds": {{Threshold: 100_000, Rate: 0.008}, {Threshold: 1_000_000, Rate: 0.018}},
		"duplicate threshold":  {{Threshold: 100_000, Rate: 0.008}, {Threshold: 100_000, Rate: 0.018}},
		"rate above one":       {{Threshold: 100_000, Rate: 1.5}},
		"negative rate":        {{Threshold: 100_000, Rate: -0.1}},
	}
	for name, tiers := range cases {
		t.Run(name, func(t *testing.T) {
			if err := ValidateTiers(tiers); err == nil {
				t.Fatal("ValidateTiers accepted an invalid table")
			}
			m := FromTiers(tiers) // applies the fallback
			if len(m.TaxTiers()) != len(DefaultConfig().TaxTiers()) {
				t.Errorf("tier count: got %d, want the default table's %d", len(m.TaxTiers()), len(DefaultConfig().TaxTiers()))
			}
		})
	}
}
```

Add `configuration/model_test.go` asserting `DefaultConfig()` returns
`maxStagedItems 9`, `minTradeLevel 0`, `reservationTtl 300s`,
`attestationTimeout 5s`, `taxEnabled true`.

- [ ] **Step 2: Run them and confirm they fail**

Run: `cd services/atlas-trades/atlas.com/trades && go test ./configuration/ -v`
Expected: FAIL — `undefined: DefaultConfig`.

- [ ] **Step 3: Write the atlas-trades-side configuration package**

`configuration/model.go` — immutable `Model` with private fields, getters, a
`With*` transform per knob, and:

```go
// Tier is one meso-tax band. Rate applies to amounts >= Threshold, with the
// FIRST matching (highest) tier winning — so the table must be strictly
// descending by Threshold.
type Tier struct {
	Threshold uint32  `json:"threshold"`
	Rate      float64 `json:"rate"`
}

// DefaultConfig is the shipped fallback used whenever a tenant has no
// trade-configs resource or its table fails validation. The service never
// hard-fails on a missing configuration resource (FR-9.2) and never silently
// disables trading.
func DefaultConfig() Model {
	return Model{
		taxEnabled: true,
		taxTiers: []Tier{
			{Threshold: 100_000_000, Rate: 0.060},
			{Threshold: 25_000_000, Rate: 0.050},
			{Threshold: 10_000_000, Rate: 0.040},
			{Threshold: 5_000_000, Rate: 0.030},
			{Threshold: 1_000_000, Rate: 0.018},
			{Threshold: 100_000, Rate: 0.008},
		},
		maxStagedItems:     9,
		minTradeLevel:      0,
		reservationTtl:     300 * time.Second,
		attestationTimeout: 5 * time.Second,
	}
}
```

`configuration/tax.go`:

```go
package configuration

import "math"

// ValidateTiers enforces FR-9.3: strictly descending thresholds and rates in
// [0, 1]. An invalid table is a loud error, never a silent partial accept.
func ValidateTiers(tiers []Tier) error {
	for i, t := range tiers {
		if t.Rate < 0 || t.Rate > 1 {
			return fmt.Errorf("trade tax tier %d: rate %.4f outside [0, 1]", i, t.Rate)
		}
		if i > 0 && t.Threshold >= tiers[i-1].Threshold {
			return fmt.Errorf("trade tax tier %d: threshold %d is not strictly below the previous tier's %d", i, t.Threshold, tiers[i-1].Threshold)
		}
	}
	return nil
}

// Tax computes the meso deduction for one side's staged amount:
// delivered = m - floor(m * rate(m)). The difference is DESTROYED — the giver's
// negative award_mesos is the full m, the receiver's positive award is
// delivered, and no third party is credited (design §6.5).
func Tax(m Model, amount uint32) (tax uint32, delivered uint32) {
	if !m.TaxEnabled() || amount == 0 {
		return 0, amount
	}
	for _, tier := range m.TaxTiers() {
		if amount >= tier.Threshold {
			tax = uint32(math.Floor(float64(amount) * tier.Rate))
			return tax, amount - tax
		}
	}
	return 0, amount
}
```

`configuration/registry.go` — a `sync.Once` singleton + `sync.RWMutex` keyed by
`tenant.Model`, fetching from atlas-tenants on first use, logging once at INFO
when a tenant has no resource and the defaults apply (FR-9.2), and running the
fetched tiers through `ValidateTiers` before accepting them.

`configuration/requests.go` + `configuration/rest.go` — the outbound
`GET /tenants/{id}/configurations/trade-configs` read, mirroring
`services/atlas-mts/atlas.com/mts/configuration/`.

- [ ] **Step 4: Add the atlas-tenants resource**

Mirror every `mts-configs` piece listed under **Files**, substituting
`trade-configs`. Concretely:

- `configuration/rest.go`: `TradeConfigRestModel` with
  `GetName() string { return "trade-configs" }` and attributes `taxEnabled`,
  `taxTiers`, `maxStagedItems`, `minTradeLevel`, `reservationTtlSeconds`,
  `attestationTimeoutSeconds`; plus `TransformTradeConfig` and
  `ExtractTradeConfig`. Note `taxTiers` is an array of objects, not a scalar —
  it decodes as `[]interface{}` of `map[string]interface{}` with `float64`
  numbers, so its Transform arm differs from the scalar `if val, ok :=
  attributes["x"].(float64)` pattern the other fields use.
- `configuration/processor.go`: the twelve-method block
  (`CreateTradeConfig`/`AndEmit`, `Update…`, `Delete…`, `GetTradeConfigById`,
  `GetAllTradeConfigs`, the two providers, `SeedTradeConfigs`).
- `configuration/provider.go`: `GetByTenantIdAndResourceNameProvider(tenantID, "trade-configs")(db)`.
- `configuration/kafka.go`: `EventTypeTradeConfigCreated/Updated/Deleted` +
  `CreateTradeConfigStatusEventProvider`.
- `configuration/seed.go`: `defaultTradeConfigsPath = "/configurations/trade-configs"`,
  `getTradeConfigsPath()` honouring `TRADE_CONFIGS_SEED_PATH`, and
  `LoadTradeConfigFiles()`.
- `configuration/resource.go`: the six handlers plus the six route registrations,
  with `/seed` registered **before** `{tradeConfigId}` so it is not shadowed.
- `rest/handler.go`: `ParseTradeConfigId` as `server.ParseStringId(l, "tradeConfigId", next)`.

- [ ] **Step 5: Ship the seed file**

Create `services/atlas-tenants/configurations/trade-configs/default.json` with
exactly the design §8 body:

```json
{
  "taxEnabled": true,
  "taxTiers": [
    { "threshold": 100000000, "rate": 0.060 },
    { "threshold": 25000000, "rate": 0.050 },
    { "threshold": 10000000, "rate": 0.040 },
    { "threshold": 5000000, "rate": 0.030 },
    { "threshold": 1000000, "rate": 0.018 },
    { "threshold": 100000, "rate": 0.008 }
  ],
  "maxStagedItems": 9,
  "minTradeLevel": 0,
  "reservationTtlSeconds": 300,
  "attestationTimeoutSeconds": 5
}
```

- [ ] **Step 6: Run the tests**

```bash
cd services/atlas-trades/atlas.com/trades && go test -race ./configuration/ -v
cd ../../../atlas-tenants/atlas.com/tenants && go test -race ./... && go vet ./... && go build ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-tenants services/atlas-trades
git commit -m "feat(task-205): add the tenant trade-configs resource and the meso tax calculator"
```

---

# Slice 4 — Saga settlement composite (Tasks 14-15)

### Task 14: `trade_settlement` action and payload

**Files:**
- Modify: `libs/atlas-saga/model.go` (Action constant), `libs/atlas-saga/payloads.go` (payload types), `libs/atlas-saga/unmarshal.go` (case)
- Test: `libs/atlas-saga/unmarshal_test.go`

**Interfaces:**
- Produces (consumed by Tasks 15 and 19):
  - `saga.TradeSettlement Action = "trade_settlement"`
  - `saga.TradeSettlementPayload{TransactionId uuid.UUID; WorldId world.Id; ChannelId channel.Id; RoomType byte; Sides [2]TradeSettlementSide}`
  - `saga.TradeSettlementSide{CharacterId uint32; Items []TradeSettlementItem; MesoStaged uint32; MesoTax uint32; MesoDelivered uint32}`
  - `saga.TradeSettlementItem{InventoryType byte; SourceSlot int16; AssetId uint32; TemplateId uint32; Quantity uint32}`

Design §6.3: one composite carrying the two participants, their staged item
references and meso, and the **resolved** tax amounts — the tax is computed in
atlas-trades (it needs the tenant config) and passed in as integers so the
orchestrator stays config-free. `saga.TradeTransaction Type =
"trade_transaction"` already exists at `model.go:16` and is reused as the saga
type.

- [ ] **Step 1: Write the failing unmarshal test**

Append to `libs/atlas-saga/unmarshal_test.go`:

```go
// TestUnmarshalTradeSettlement pins that a trade_settlement step round-trips
// through the shared lib's payload unmarshaller — an unregistered action
// deserialises to nil and fails at runtime with "unknown action type".
func TestUnmarshalTradeSettlement(t *testing.T) {
	raw := []byte(`{
	  "transactionId": "11111111-1111-1111-1111-111111111111",
	  "sagaType": "trade_transaction",
	  "initiatedBy": "atlas-trades",
	  "steps": [{
	    "stepId": "trade_settlement",
	    "status": "pending",
	    "action": "trade_settlement",
	    "payload": {
	      "transactionId": "11111111-1111-1111-1111-111111111111",
	      "worldId": 1,
	      "channelId": 1,
	      "roomType": 3,
	      "sides": [
	        {"characterId": 100, "mesoStaged": 10000000, "mesoTax": 400000, "mesoDelivered": 9600000,
	         "items": [{"inventoryType": 2, "sourceSlot": 1, "assetId": 55, "templateId": 2000000, "quantity": 5}]},
	        {"characterId": 200, "mesoStaged": 0, "mesoTax": 0, "mesoDelivered": 0, "items": []}
	      ]
	    }
	  }]
	}`)

	var s Saga
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(s.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(s.Steps))
	}
	p, ok := s.Steps[0].Payload.(TradeSettlementPayload)
	if !ok {
		t.Fatalf("payload type: got %T, want TradeSettlementPayload", s.Steps[0].Payload)
	}
	if p.Sides[0].MesoDelivered != 9_600_000 {
		t.Errorf("side 0 mesoDelivered: got %d, want 9600000", p.Sides[0].MesoDelivered)
	}
	if len(p.Sides[0].Items) != 1 || p.Sides[0].Items[0].AssetId != 55 {
		t.Errorf("side 0 items: got %+v", p.Sides[0].Items)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd libs/atlas-saga && go test ./ -run TestUnmarshalTradeSettlement -v`
Expected: FAIL — `undefined: TradeSettlementPayload`.

- [ ] **Step 3: Add the action constant**

In `libs/atlas-saga/model.go`, after the Storage action block (`:122-131`):

```go
	// Trade actions (task-205). trade_settlement is a COMPOSITE: the
	// orchestrator expands it into release_from_character / accept_to_character /
	// award_mesos steps (see expandTradeSettlement). atlas-trades never
	// enumerates concrete saga steps itself. The saga type is the pre-existing
	// TradeTransaction.
	TradeSettlement Action = "trade_settlement"
```

- [ ] **Step 4: Add the payload types**

In `libs/atlas-saga/payloads.go`:

```go
// TradeSettlementItem references one staged asset. Under the
// reserve-at-staging model the asset is still in the owner's inventory at
// expansion time, so the orchestrator can look it up by slot exactly as
// expandTransferToStorage does.
type TradeSettlementItem struct {
	InventoryType byte   `json:"inventoryType"`
	SourceSlot    int16  `json:"sourceSlot"`
	AssetId       uint32 `json:"assetId"`
	TemplateId    uint32 `json:"templateId"`
	Quantity      uint32 `json:"quantity"`
}

// TradeSettlementSide is one participant's contribution. The tax figures are
// RESOLVED INTEGERS computed by atlas-trades from the tenant config — the
// orchestrator stays config-free (design §6.3). MesoStaged is deducted from
// this side; MesoDelivered is credited to the other; the difference
// (MesoTax) is destroyed.
type TradeSettlementSide struct {
	CharacterId   uint32                `json:"characterId"`
	Items         []TradeSettlementItem `json:"items"`
	MesoStaged    uint32                `json:"mesoStaged"`
	MesoTax       uint32                `json:"mesoTax"`
	MesoDelivered uint32                `json:"mesoDelivered"`
}

// TradeSettlementPayload is the whole two-party swap as one compensatable unit.
type TradeSettlementPayload struct {
	TransactionId uuid.UUID              `json:"transactionId"`
	WorldId       world.Id               `json:"worldId"`
	ChannelId     channel.Id             `json:"channelId"`
	RoomType      byte                   `json:"roomType"`
	Sides         [2]TradeSettlementSide `json:"sides"`
}
```

- [ ] **Step 5: Register the unmarshal case**

In `libs/atlas-saga/unmarshal.go`, alongside `case ReleaseFromCharacter` (`:342`):

```go
	case TradeSettlement:
		var p TradeSettlementPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		return p, nil
```

- [ ] **Step 6: Run the tests**

```bash
cd libs/atlas-saga && go test -race ./... -v && go vet ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-saga
git commit -m "feat(task-205): add the trade_settlement saga action and payload"
```

---

### Task 15: Orchestrator expansion of `trade_settlement`

> **Amended by Task 29.** `expandTradeSettlement` keeps its ordering rule and its tax arithmetic, but releases from escrow instead of from characters and drops the negative `award_mesos` legs.

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go` (`isExpandableAction` `:1167-1184`, `expandAndProcessStep` switch `:1186-1264`, new `expandTradeSettlement`)
- Modify: `saga/event_acceptance.go:170`, `saga/error_mapper.go:25`, `saga/character_extractor.go:65,73`, `saga/model.go:1267,1309`
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/trade_expansion_test.go` (create)

**Interfaces:**
- Consumes: `saga.TradeSettlementPayload` (Task 14), the orchestrator-local
  `AcceptToCharacterPayload` (`saga/model.go:914-922`),
  `assetDataFromCompartmentAsset`, `compartment.RequestCompartment`.
- Produces: runtime expansion of one `trade_settlement` step into the concrete
  step list, consumed by Task 19 at execution time.

Design §6.3 step order — **all releases precede all accepts**, so a slot freed by
an outgoing item is available to an incoming one, and a failure in either side's
release compensates before anything has been created:

```
for each staged item of A:  release_from_character(A, assetId, qty)
for each staged item of B:  release_from_character(B, assetId, qty)
for each staged item of A:  accept_to_character(B, snapshot)
for each staged item of B:  accept_to_character(A, snapshot)
if A.meso > 0:              award_mesos(A, -A.mesoStaged); award_mesos(B, +A.mesoDelivered)
if B.meso > 0:              award_mesos(B, -B.mesoStaged); award_mesos(A, +B.mesoDelivered)
```

- [ ] **Step 1: Write the failing expansion test**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/trade_expansion_test.go`:

```go
package saga

import "testing"

// TestExpandTradeSettlementOrdersReleasesBeforeAccepts pins design §6.3: every
// release precedes every accept, so an outgoing item's slot is free for an
// incoming one and a release failure compensates before anything is created.
func TestExpandTradeSettlementOrdersReleasesBeforeAccepts(t *testing.T) {
	p := testProcessorWithCompartments(t, tradeCompartments())

	step := NewStep[any]("trade_settlement", Pending, TradeSettlement, tradeSettlementFixture())
	steps, err := p.expandTradeSettlement(step)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}

	lastRelease, firstAccept := -1, -1
	for i, s := range steps {
		switch s.Action() {
		case ReleaseFromCharacter:
			lastRelease = i
		case AcceptToCharacter:
			if firstAccept == -1 {
				firstAccept = i
			}
		}
	}
	if lastRelease == -1 || firstAccept == -1 {
		t.Fatalf("expected both releases and accepts, got %d steps", len(steps))
	}
	if lastRelease > firstAccept {
		t.Errorf("release at %d follows accept at %d; every release must precede every accept", lastRelease, firstAccept)
	}
}

// TestExpandTradeSettlementCrossesTheAssets pins that A's items are accepted by
// B and vice versa — a same-side accept would be a no-op swap.
func TestExpandTradeSettlementCrossesTheAssets(t *testing.T) {
	p := testProcessorWithCompartments(t, tradeCompartments())
	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, tradeSettlementFixture()))
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	for _, s := range steps {
		if s.Action() != AcceptToCharacter {
			continue
		}
		pl := s.Payload().(AcceptToCharacterPayload)
		switch pl.TemplateId {
		case 2000000: // staged by character 100
			if pl.CharacterId != 200 {
				t.Errorf("item 2000000 accepted by %d, want 200", pl.CharacterId)
			}
		case 1302000: // staged by character 200
			if pl.CharacterId != 100 {
				t.Errorf("item 1302000 accepted by %d, want 100", pl.CharacterId)
			}
		}
	}
}

// TestExpandTradeSettlementMesoIsAsymmetric pins design §6.5: the giver is
// deducted the FULL staged amount and the receiver credited the POST-TAX
// amount, so the tax is destroyed rather than moved.
func TestExpandTradeSettlementMesoIsAsymmetric(t *testing.T) {
	p := testProcessorWithCompartments(t, tradeCompartments())
	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, tradeSettlementFixture()))
	if err != nil {
		t.Fatalf("expand: %v", err)
	}

	var deducted, credited int32
	for _, s := range steps {
		if s.Action() != AwardMesos {
			continue
		}
		pl := s.Payload().(AwardMesosPayload)
		if pl.Amount < 0 {
			deducted += -pl.Amount
		} else {
			credited += pl.Amount
		}
	}
	// Fixture stages 10,000,000 from side A at the 4% tier.
	if deducted != 10_000_000 {
		t.Errorf("deducted: got %d, want 10000000", deducted)
	}
	if credited != 9_600_000 {
		t.Errorf("credited: got %d, want 9600000", credited)
	}
	if deducted-credited != 400_000 {
		t.Errorf("destroyed: got %d, want 400000", deducted-credited)
	}
}

// TestExpandTradeSettlementEmitsNoMesoStepsWhenNothingStaged pins that an
// item-only trade produces no award_mesos steps at all.
func TestExpandTradeSettlementEmitsNoMesoStepsWhenNothingStaged(t *testing.T) {
	p := testProcessorWithCompartments(t, tradeCompartments())
	fixture := tradeSettlementFixture()
	fixture.Sides[0].MesoStaged, fixture.Sides[0].MesoTax, fixture.Sides[0].MesoDelivered = 0, 0, 0
	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, fixture))
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	for _, s := range steps {
		if s.Action() == AwardMesos {
			t.Fatalf("unexpected award_mesos step for an item-only trade: %+v", s.Payload())
		}
	}
}

// tradeCompartments is the shared fixture inventory: character 100 holds five
// of item 2000000 in USE slot 1, character 200 holds one 1302000 in EQUIP
// slot 3. testProcessorWithCompartments stubs compartment.RequestCompartment
// to serve it.
func tradeCompartments() map[uint32][]testAsset {
	return map[uint32][]testAsset{
		100: {{Slot: 1, TemplateId: 2000000, Quantity: 5, Id: "55", InventoryType: 2}},
		200: {{Slot: 3, TemplateId: 1302000, Quantity: 1, Id: "77", InventoryType: 1}},
	}
}

// tradeSettlementFixture stages the compartments above plus 10,000,000 meso
// from side A, already taxed at the default 4% tier (design §6.3 requires the
// tax to arrive RESOLVED — the orchestrator does no rate arithmetic).
func tradeSettlementFixture() TradeSettlementPayload {
	return TradeSettlementPayload{
		TransactionId: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		WorldId:       1,
		ChannelId:     1,
		RoomType:      3,
		Sides: [2]TradeSettlementSide{
			{
				CharacterId:   100,
				Items:         []TradeSettlementItem{{InventoryType: 2, SourceSlot: 1, AssetId: 55, TemplateId: 2000000, Quantity: 5}},
				MesoStaged:    10_000_000,
				MesoTax:       400_000,
				MesoDelivered: 9_600_000,
			},
			{
				CharacterId: 200,
				Items:       []TradeSettlementItem{{InventoryType: 1, SourceSlot: 3, AssetId: 77, TemplateId: 1302000, Quantity: 1}},
			},
		},
	}
}
```

- [ ] **Step 2: Run them and confirm they fail**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run TestExpandTradeSettlement -v`
Expected: FAIL — `p.expandTradeSettlement undefined`.

- [ ] **Step 3: Write the expander**

In `saga/processor.go`, next to `expandWithdrawFromStorage`:

```go
// expandTradeSettlement expands the task-205 trade_settlement composite into
// the concrete two-party swap. Step order matters (design §6.3): ALL releases
// precede ALL accepts, so a slot freed by an outgoing item is available to an
// incoming one, and a failure in either side's release compensates before
// anything has been created.
//
// The tax figures arrive already resolved from atlas-trades (which owns the
// tenant config); this expander does no rate arithmetic. The giver is deducted
// MesoStaged and the receiver credited MesoDelivered — the difference is
// destroyed, credited to nobody.
func (p *ProcessorImpl) expandTradeSettlement(st Step[any]) ([]Step[any], error) {
	payload, ok := st.Payload().(TradeSettlementPayload)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for TradeSettlement")
	}

	// Snapshot both sides' assets first: every accept needs an AssetData built
	// from the asset as it exists BEFORE any release soft-deletes it. The
	// snapshot is the whole AssetData, so cash-item ownership and expiry
	// (atlas-asset-expiration) survive the transfer unchanged — FR-10.3 needs
	// no special cash path.
	type resolved struct {
		item     TradeSettlementItem
		assetId  uint32
		snapshot asset2.AssetData
	}
	sides := [2][]resolved{}
	for si, side := range payload.Sides {
		for _, item := range side.Items {
			comp, err := compartment.RequestCompartment(p.l, p.ctx)(side.CharacterId, item.InventoryType)
			if err != nil {
				return nil, fmt.Errorf("unable to lookup character [%d] inventory compartment: %w", side.CharacterId, err)
			}
			var found *compartment.AssetRestModel
			for i := range comp.Assets {
				if comp.Assets[i].Slot == item.SourceSlot {
					found = &comp.Assets[i]
					break
				}
			}
			if found == nil {
				return nil, fmt.Errorf("no asset found at slot [%d] in character [%d] inventory [%d]", item.SourceSlot, side.CharacterId, item.InventoryType)
			}
			if found.TemplateId != item.TemplateId {
				return nil, fmt.Errorf("asset at slot [%d] for character [%d] is template [%d], expected [%d]", item.SourceSlot, side.CharacterId, found.TemplateId, item.TemplateId)
			}
			var assetId uint32
			fmt.Sscanf(found.Id, "%d", &assetId)
			sides[si] = append(sides[si], resolved{item: item, assetId: assetId, snapshot: assetDataFromCompartmentAsset(found)})
		}
	}

	steps := make([]Step[any], 0)

	// 1. Every release, both sides.
	for si, side := range payload.Sides {
		for _, r := range sides[si] {
			steps = append(steps, NewStep[any](
				fmt.Sprintf("release_from_character_%d_%d", side.CharacterId, r.assetId),
				Pending,
				ReleaseFromCharacter,
				ReleaseFromCharacterPayload{
					TransactionId: payload.TransactionId,
					CharacterId:   side.CharacterId,
					InventoryType: r.item.InventoryType,
					AssetId:       r.assetId,
					Quantity:      r.item.Quantity,
				},
			))
		}
	}

	// 2. Every accept, crossed: side 0's items go to side 1 and vice versa.
	for si, side := range payload.Sides {
		recipient := payload.Sides[1-si].CharacterId
		for _, r := range sides[si] {
			steps = append(steps, NewStep[any](
				fmt.Sprintf("accept_to_character_%d_%d", recipient, r.assetId),
				Pending,
				AcceptToCharacter,
				AcceptToCharacterPayload{
					TransactionId: payload.TransactionId,
					CharacterId:   recipient,
					InventoryType: r.item.InventoryType,
					TemplateId:    r.item.TemplateId,
					AssetData:     r.snapshot,
				},
			))
			_ = side
		}
	}

	// 3. Meso, per side that staged any. Deduct the full staged amount from the
	//    giver, credit the post-tax amount to the receiver.
	for si, side := range payload.Sides {
		if side.MesoStaged == 0 {
			continue
		}
		receiver := payload.Sides[1-si].CharacterId
		steps = append(steps,
			NewStep[any](
				fmt.Sprintf("award_mesos_deduct_%d", side.CharacterId),
				Pending,
				AwardMesos,
				AwardMesosPayload{
					CharacterId: side.CharacterId,
					WorldId:     payload.WorldId,
					ChannelId:   payload.ChannelId,
					ActorId:     receiver,
					ActorType:   "CHARACTER",
					Amount:      -int32(side.MesoStaged),
					ShowEffect:  false,
				},
			),
			NewStep[any](
				fmt.Sprintf("award_mesos_credit_%d", receiver),
				Pending,
				AwardMesos,
				AwardMesosPayload{
					CharacterId: receiver,
					WorldId:     payload.WorldId,
					ChannelId:   payload.ChannelId,
					ActorId:     side.CharacterId,
					ActorType:   "CHARACTER",
					Amount:      int32(side.MesoDelivered),
					ShowEffect:  true,
				},
			),
		)
	}

	return steps, nil
}
```

- [ ] **Step 4: Register the action in all six registries**

Miss any one of these and it fails at runtime, not at compile time:

1. `saga/processor.go:1167-1184` — add `TradeSettlement` to `isExpandableAction`'s
   case list. `TestIsExpandableActionCoversExpansionSwitch`
   (`saga/mts_expansion_test.go:262-280`) fails if you skip this.
2. `saga/processor.go:1186-1264` — add
   `case TradeSettlement: newSteps, err = p.expandTradeSettlement(st)` to
   `expandAndProcessStep`.
3. `saga/event_acceptance.go:170` — the expanded actions
   (`ReleaseFromCharacter`, `AcceptToCharacter`, `AwardMesos`) already have
   entries; confirm no new entry is needed for the composite itself, and add one
   only if the gate rejects an unlisted action.
4. `saga/error_mapper.go:25` — map a `trade_settlement` failure to the error
   class atlas-trades translates into `LEAVE 8`.
5. `saga/character_extractor.go:65,73` — extract **both** participants so
   per-character saga tracking sees the trade.
6. `saga/model.go:1267,1309` — the orchestrator-side unmarshal case for
   `TradeSettlementPayload`.

- [ ] **Step 5: Run the tests**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... -v && go vet ./... && go build ./...
```

Expected: PASS, including `TestIsExpandableActionCoversExpansionSwitch`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-saga-orchestrator
git commit -m "feat(task-205): expand trade_settlement into the two-party swap in the orchestrator"
```

---

# Slice 5 — Behaviour and wiring (Tasks 16-24)

### Task 16: Trade Kafka contracts

**Files:**
- Create: `services/atlas-trades/atlas.com/trades/kafka/message/message.go`, `kafka/message/trade/kafka.go`, `kafka/message/character/kafka.go`, `kafka/message/invite/kafka.go`, `kafka/message/saga/kafka.go`, `kafka/message/compartment/kafka.go`
- Create: `services/atlas-trades/atlas.com/trades/kafka/consumer/consumer.go`
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/trade/kafka.go`
- Test: `services/atlas-trades/atlas.com/trades/kafka/message/trade/kafka_test.go`

**Interfaces:**
- Produces (consumed by Tasks 17-22): the `COMMAND_TOPIC_TRADE` /
  `EVENT_TOPIC_TRADE_STATUS` envelopes, mirrored byte-for-byte in atlas-channel.

PRD §5's Kafka contracts. The atlas-channel copy must match struct names, field
names and json tags **exactly** — this is the task-17 mirroring convention noted
at `services/atlas-mini-games/.../kafka/message/minigame/kafka.go:11`.

- [ ] **Step 1: Write the failing contract-parity test**

Create `services/atlas-trades/atlas.com/trades/kafka/message/trade/kafka_test.go`:

```go
package trade

import (
	"encoding/json"
	"testing"
)

// TestCommandEnvelopeJsonShape pins the wire contract atlas-channel mirrors.
// A field-name or tag drift here silently breaks the channel's decode into a
// zero-valued body.
func TestCommandEnvelopeJsonShape(t *testing.T) {
	c := Command[PutItemCommandBody]{
		TransactionId: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		WorldId:       1,
		ChannelId:     2,
		MapId:         100000000,
		CharacterId:   100,
		Type:          CommandTypePutItem,
		Body:          PutItemCommandBody{InventoryType: 2, Slot: 1, Quantity: 5, TargetSlot: 3},
	}
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round map[string]interface{}
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, k := range []string{"transactionId", "worldId", "channelId", "mapId", "instance", "characterId", "type", "body"} {
		if _, ok := round[k]; !ok {
			t.Errorf("command envelope missing key %q", k)
		}
	}
	body := round["body"].(map[string]interface{})
	for _, k := range []string{"inventoryType", "slot", "quantity", "targetSlot"} {
		if _, ok := body[k]; !ok {
			t.Errorf("put-item body missing key %q", k)
		}
	}
}

// TestEveryCommandTypeIsDistinct guards against a copy-paste collision in the
// const block — two commands sharing a type string means one handler silently
// swallows the other's messages.
func TestEveryCommandTypeIsDistinct(t *testing.T) {
	all := []string{
		CommandTypeCreateRoom, CommandTypeInvite, CommandTypeDeclineInvite,
		CommandTypeEnterRoom, CommandTypePutItem, CommandTypeAddMeso,
		CommandTypeConfirm, CommandTypeTransaction, CommandTypeCancel, CommandTypeChat,
	}
	seen := make(map[string]bool, len(all))
	for _, v := range all {
		if v == "" {
			t.Fatal("empty command type constant")
		}
		if seen[v] {
			t.Errorf("duplicate command type %q", v)
		}
		seen[v] = true
	}
}
```

Write the mirror-image `TestEveryStatusTypeIsDistinct` over the eleven status
types.

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd services/atlas-trades/atlas.com/trades && go test ./kafka/message/trade/ -v`
Expected: FAIL — `undefined: Command`.

- [ ] **Step 3: Write the contract**

Create `services/atlas-trades/atlas.com/trades/kafka/message/trade/kafka.go`:

```go
// Package trade carries the COMMAND_TOPIC_TRADE / EVENT_TOPIC_TRADE_STATUS
// envelopes. Mirrored byte-for-byte by atlas-channel
// (services/atlas-channel/atlas.com/channel/kafka/message/trade/kafka.go);
// struct names, field names and json tags must match that file exactly.
package trade

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic     = "COMMAND_TOPIC_TRADE"
	EnvEventTopicStatus = "EVENT_TOPIC_TRADE_STATUS"

	CommandTypeCreateRoom    = "CREATE_ROOM"
	CommandTypeInvite        = "INVITE"
	CommandTypeDeclineInvite = "DECLINE_INVITE"
	CommandTypeEnterRoom     = "ENTER_ROOM"
	CommandTypePutItem       = "PUT_ITEM"
	CommandTypeAddMeso       = "ADD_MESO"
	CommandTypeConfirm       = "CONFIRM"
	CommandTypeTransaction   = "TRANSACTION"
	CommandTypeCancel        = "CANCEL"
	CommandTypeChat          = "CHAT"

	StatusTypeRoomCreated          = "ROOM_CREATED"
	StatusTypeInviteSent           = "INVITE_SENT"
	StatusTypeInviteRejected       = "INVITE_REJECTED"
	StatusTypeParticipantEntered   = "PARTICIPANT_ENTERED"
	StatusTypeItemStaged           = "ITEM_STAGED"
	StatusTypeMesoStaged           = "MESO_STAGED"
	StatusTypeMesoRefused          = "MESO_REFUSED"
	StatusTypeParticipantConfirmed = "PARTICIPANT_CONFIRMED"
	StatusTypeAttestationRequested = "ATTESTATION_REQUESTED"
	StatusTypeSettled              = "SETTLED"
	StatusTypeCancelled            = "CANCELLED"
	StatusTypeError                = "ERROR"
	StatusTypeChat                 = "CHAT"
)

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

type CreateRoomCommandBody struct {
	RoomType byte `json:"roomType"`
}

type InviteCommandBody struct {
	TargetCharacterId uint32 `json:"targetCharacterId"`
}

type DeclineInviteCommandBody struct {
	SerialNumber uint32 `json:"serialNumber"`
	ErrorCode    byte   `json:"errorCode"`
}

type EnterRoomCommandBody struct {
	Handle uint32 `json:"handle"`
}

type PutItemCommandBody struct {
	InventoryType byte   `json:"inventoryType"`
	Slot          int16  `json:"slot"`
	Quantity      uint16 `json:"quantity"`
	TargetSlot    byte   `json:"targetSlot"`
}

// AddMesoCommandBody carries the ABSOLUTE total from the client's input box,
// not a delta (CTradingRoomDlg::PutMoney, design §1.6). Signed because the
// serverbound codec is Encode4 of a signed int32 and a hostile client can send
// a negative.
type AddMesoCommandBody struct {
	Amount int32 `json:"amount"`
}

// CrcEntry is one {data, crc} pair from a TRADE_CONFIRM or TRANSACTION
// payload. Absent on GMS <= v79 (tradeCrcPresent), where the lists are empty.
type CrcEntry struct {
	Data uint32 `json:"data"`
	Crc  uint32 `json:"crc"`
}

type ConfirmCommandBody struct {
	Entries []CrcEntry `json:"entries"`
}

type TransactionCommandBody struct {
	Entries []CrcEntry `json:"entries"`
}

type CancelCommandBody struct{}

type ChatCommandBody struct {
	Message string `json:"message"`
}

// StatusEvent is the atlas-trades -> atlas-channel envelope. It always carries
// both participants so the channel can address the room without a lookup.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	RoomId        uuid.UUID  `json:"roomId"`
	Handle        uint32     `json:"handle"`
	RoomType      byte       `json:"roomType"`
	OwnerId       uint32     `json:"ownerId"`
	VisitorId     uint32     `json:"visitorId"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type RoomCreatedEventBody struct {
	// Position is the recipient's seat: 0 owner, 1 visitor (FR-1.5).
	Position byte `json:"position"`
}

type ParticipantEnteredEventBody struct {
	CharacterId uint32 `json:"characterId"`
	Name        string `json:"name"`
	Position    byte   `json:"position"`
}

type InviteSentEventBody struct {
	TargetCharacterId uint32 `json:"targetCharacterId"`
	InviterName       string `json:"inviterName"`
}

// ItemStagedEventBody names the staging SIDE by position, not by character.
// atlas-channel converts it to each recipient's own recipient-relative side
// byte before writing the packet.
type ItemStagedEventBody struct {
	Position      byte   `json:"position"`
	TradeSlot     byte   `json:"tradeSlot"`
	InventoryType byte   `json:"inventoryType"`
	SourceSlot    int16  `json:"sourceSlot"`
	AssetId       uint32 `json:"assetId"`
	TemplateId    uint32 `json:"templateId"`
	Quantity      uint32 `json:"quantity"`
}

type MesoStagedEventBody struct {
	Position byte   `json:"position"`
	Amount   uint32 `json:"amount"`
}

// MesoRefusedEventBody drives the authoritative re-echo: atlas-channel sends
// TRADE_ADD_MESO with LastValidAmount so the client's view snaps back, plus
// TRADE_MESO_LIMIT where that arm exists (design §4.2).
type MesoRefusedEventBody struct {
	Position        byte   `json:"position"`
	LastValidAmount uint32 `json:"lastValidAmount"`
}

type ParticipantConfirmedEventBody struct {
	Position byte `json:"position"`
}

type AttestationRequestedEventBody struct{}

type SettledEventBody struct {
	LedgerEntryId uuid.UUID `json:"ledgerEntryId"`
}

// CancelledEventBody carries the semantic leaveReason KEY string, which
// atlas-channel resolves to a per-version status byte via the tenant
// leaveReason table (DOM-25). Never a numeric status.
type CancelledEventBody struct {
	Reason string `json:"reason"`
}

// ErrorEventBody carries an enterError KEY string, resolved the same way.
type ErrorEventBody struct {
	Code string `json:"code"`
}

type ChatEventBody struct {
	Position byte   `json:"position"`
	Message  string `json:"message"`
}
```

- [ ] **Step 4: Mirror the file into atlas-channel**

Copy it verbatim to
`services/atlas-channel/atlas.com/channel/kafka/message/trade/kafka.go`, changing
only the package doc comment's direction ("Mirrors
services/atlas-trades/…/kafka/message/trade/kafka.go").

- [ ] **Step 5: Add the remaining atlas-trades message packages**

`kafka/message/message.go` — the buffer/emit aliases, copied from
`services/atlas-mini-games/.../kafka/message/message.go`.

`kafka/message/character/kafka.go`, `kafka/message/invite/kafka.go`,
`kafka/message/saga/kafka.go`, `kafka/message/compartment/kafka.go` — the
consumed/produced envelopes for character status, invites, saga commands and
inventory reservations, each a byte-for-byte mirror of the owning service's
file. For invites, mirror
`services/atlas-invites/.../kafka/message/invite/kafka.go` exactly.

`kafka/consumer/consumer.go` — the shared `NewConfig(l)(name)(token)(groupId)`
helper + `LookupBrokers()`, copied from
`services/atlas-mini-games/.../kafka/consumer/consumer.go:12-25`.

- [ ] **Step 6: Run the tests**

```bash
cd services/atlas-trades/atlas.com/trades && go test -race ./... && go build ./...
cd ../../../atlas-channel/atlas.com/channel && go build ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-trades services/atlas-channel
git commit -m "feat(task-205): add the COMMAND_TOPIC_TRADE and EVENT_TOPIC_TRADE_STATUS contracts"
```

---

### Task 17: Room lifecycle — create, invite, decline, enter

**Files:**
- Modify: `services/atlas-trades/atlas.com/trades/trade/processor.go` (lifecycle methods)
- Create: `services/atlas-trades/atlas.com/trades/trade/producer.go`
- Create: `services/atlas-trades/atlas.com/trades/data/character/`, `data/map/` (REST readers)
- Create: `services/atlas-trades/atlas.com/trades/kafka/consumer/trade/consumer.go`, `kafka/consumer/invite/consumer.go`
- Modify: `services/atlas-trades/atlas.com/trades/main.go`
- Test: `services/atlas-trades/atlas.com/trades/trade/processor_lifecycle_test.go`

**Interfaces:**
- Consumes: `trade.Registry` (Task 10), `configuration.GetRegistry()` (Task 13), the contracts (Task 16).
- Produces (consumed by Tasks 18-22):
  - `trade.Processor.CreateRoom(txId uuid.UUID, f field.Model, characterId uint32, roomType byte) error`
  - `trade.Processor.Invite(txId uuid.UUID, f field.Model, characterId uint32, targetCharacterId uint32) error`
  - `trade.Processor.DeclineInvite(txId uuid.UUID, characterId uint32, originatorId uint32) error`
  - `trade.Processor.EnterRoom(txId uuid.UUID, f field.Model, characterId uint32, handle uint32) error`
  - `trade.Processor.TeardownCharacter(txId uuid.UUID, characterId uint32, reason string) error`

Design §3.1 states and §3.3 teardown triggers; FR-2.1-2.6; FR-4.5-4.7
restrictions at create/accept time.

- [ ] **Step 1: Write the failing lifecycle tests**

Create `services/atlas-trades/atlas.com/trades/trade/processor_lifecycle_test.go`:

```go
package trade

import "testing"

// TestCreateRoomStartsSoloWithCapacityTwo pins FR-1.1: the room starts with
// exactly one occupant at position 0.
func TestCreateRoomStartsSoloWithCapacityTwo(t *testing.T) {
	p, _ := testProcessor(t)
	if err := p.CreateRoom(uuid.New(), testField(t), 100, 3); err != nil {
		t.Fatalf("create: %v", err)
	}
	room, ok := p.RoomForCharacter(100)
	if !ok {
		t.Fatal("room not registered")
	}
	if room.State() != StateOpenSolo {
		t.Errorf("state: got %s, want %s", room.State(), StateOpenSolo)
	}
	if len(room.Participants()) != 1 {
		t.Errorf("participants: got %d, want 1", len(room.Participants()))
	}
	if room.Handle() != 100 {
		t.Errorf("handle: got %d, want the owner's character id 100", room.Handle())
	}
}

// TestCreateRoomRejectsDeadCharacter pins FR-4.7: a dead character gets
// NOT_WHEN_DEAD, not a room.
func TestCreateRoomRejectsDeadCharacter(t *testing.T) {
	p, emitted := testProcessorWithCharacter(t, testCharacter{Id: 100, Hp: 0, Level: 30})
	if err := p.CreateRoom(uuid.New(), testField(t), 100, 3); err != nil {
		t.Fatalf("create must buffer an error event, not return: %v", err)
	}
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("a dead character got a room")
	}
	assertErrorEvent(t, emitted, "NOT_WHEN_DEAD")
}

// TestCreateRoomRejectsTradeDisallowedMap pins FR-4.6.
func TestCreateRoomRejectsTradeDisallowedMap(t *testing.T) {
	p, emitted := testProcessorWithMap(t, testMap{Id: 100000000, TradeDisallowed: true})
	_ = p.CreateRoom(uuid.New(), testField(t), 100, 3)
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("a trade-disallowed map allowed a room")
	}
	assertErrorEvent(t, emitted, "TRADE_NOT_ALLOWED")
}

// TestCreateRoomRejectsBelowMinLevel pins FR-4.5 against a tenant config with a
// non-default minimum.
func TestCreateRoomRejectsBelowMinLevel(t *testing.T) {
	p, emitted := testProcessorWithConfig(t, configuration.DefaultConfig().WithMinTradeLevel(20), testCharacter{Id: 100, Hp: 100, Level: 10})
	_ = p.CreateRoom(uuid.New(), testField(t), 100, 3)
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("a below-minimum-level character got a room")
	}
	assertErrorEvent(t, emitted, "UNABLE")
}

// TestCreateRoomRejectsSecondRoom pins FR-1.2's authoritative half.
func TestCreateRoomRejectsSecondRoom(t *testing.T) {
	p, emitted := testProcessor(t)
	_ = p.CreateRoom(uuid.New(), testField(t), 100, 3)
	_ = p.CreateRoom(uuid.New(), testField(t), 100, 3)
	assertErrorEvent(t, emitted, "OTHER_REQUESTS")
}

// TestInviteMovesRoomToPendingAndProducesTradeInvite pins FR-2.1: atlas-trades
// is the first production producer of invite.TypeTrade, and the referenceId is
// the room's uint32 handle (a uuid does not fit).
func TestInviteMovesRoomToPendingAndProducesTradeInvite(t *testing.T) {
	p, emitted := testProcessor(t)
	_ = p.CreateRoom(uuid.New(), testField(t), 100, 3)
	if err := p.Invite(uuid.New(), testField(t), 100, 200); err != nil {
		t.Fatalf("invite: %v", err)
	}
	room, _ := p.RoomForCharacter(100)
	if room.State() != StatePendingInvite {
		t.Errorf("state: got %s, want %s", room.State(), StatePendingInvite)
	}
	cmd := assertInviteCommand(t, emitted)
	if cmd.InviteType != invite.TypeTrade {
		t.Errorf("inviteType: got %s, want TRADE", cmd.InviteType)
	}
	if cmd.Body.ReferenceId != invite.Id(room.Handle()) {
		t.Errorf("referenceId: got %d, want the room handle %d", cmd.Body.ReferenceId, room.Handle())
	}
}

// TestDeclineDestroysTheRoom pins FR-2.5 / design §3.1: a decline tears the
// pending room down (the reference client closes the inviter's dialog).
func TestDeclineDestroysTheRoom(t *testing.T) {
	p, _ := testProcessor(t)
	_ = p.CreateRoom(uuid.New(), testField(t), 100, 3)
	_ = p.Invite(uuid.New(), testField(t), 100, 200)
	if err := p.DeclineInvite(uuid.New(), 200, 100); err != nil {
		t.Fatalf("decline: %v", err)
	}
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("room survived a declined invite")
	}
}

// TestEnterRoomSeatsVisitorAtPositionOne pins FR-1.5 / FR-2.4.
func TestEnterRoomSeatsVisitorAtPositionOne(t *testing.T) {
	p, _ := testProcessor(t)
	_ = p.CreateRoom(uuid.New(), testField(t), 100, 3)
	_ = p.Invite(uuid.New(), testField(t), 100, 200)
	room, _ := p.RoomForCharacter(100)
	if err := p.EnterRoom(uuid.New(), testField(t), 200, room.Handle()); err != nil {
		t.Fatalf("enter: %v", err)
	}
	room, _ = p.RoomForCharacter(200)
	if room.State() != StateOpen {
		t.Errorf("state: got %s, want %s", room.State(), StateOpen)
	}
	pt, ok := room.ParticipantFor(200)
	if !ok || pt.Position() != 1 {
		t.Errorf("visitor position: got %+v", pt)
	}
}

// TestEnterRoomRejectsAThirdCharacter pins that a paired room is closed.
func TestEnterRoomRejectsAThirdCharacter(t *testing.T) {
	p, emitted := testProcessor(t)
	_ = p.CreateRoom(uuid.New(), testField(t), 100, 3)
	_ = p.Invite(uuid.New(), testField(t), 100, 200)
	room, _ := p.RoomForCharacter(100)
	_ = p.EnterRoom(uuid.New(), testField(t), 200, room.Handle())
	_ = p.EnterRoom(uuid.New(), testField(t), 300, room.Handle())
	assertErrorEvent(t, emitted, "OTHER_REQUESTS")
}
```

- [ ] **Step 2: Run them and confirm they fail**

Run: `cd services/atlas-trades/atlas.com/trades && go test ./trade/ -run 'TestCreateRoom|TestInvite|TestDecline|TestEnterRoom' -v`
Expected: FAIL — `p.CreateRoom undefined`.

- [ ] **Step 3: Add the producer**

Create `services/atlas-trades/atlas.com/trades/trade/producer.go`, mirroring
`services/atlas-mini-games/.../game/producer.go:60-101`:

```go
// statusEventProvider builds one EVENT_TOPIC_TRADE_STATUS message. The key is
// the map id so a room's events stay ordered relative to each other.
func statusEventProvider[E any](txId uuid.UUID, r Room, characterId uint32, eventType string, body E) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(r.Field().MapId()))
	value := &trademsg.StatusEvent[E]{
		TransactionId: txId,
		WorldId:       r.Field().WorldId(),
		ChannelId:     r.Field().ChannelId(),
		MapId:         r.Field().MapId(),
		Instance:      r.Field().Instance(),
		RoomId:        r.Id(),
		Handle:        r.Handle(),
		RoomType:      r.RoomType(),
		OwnerId:       r.OwnerId(),
		VisitorId:     r.VisitorId(),
		CharacterId:   characterId,
		Type:          eventType,
		Body:          body,
	}
	return producer.SingleMessageProvider(key, value)
}

// errorProvider announces a mini-room enter error by its semantic KEY string.
// atlas-channel resolves it to the per-version numeric code (DOM-25).
func errorProvider(txId uuid.UUID, r Room, characterId uint32, code string) model.Provider[[]kafka.Message] {
	return statusEventProvider(txId, r, characterId, trademsg.StatusTypeError, trademsg.ErrorEventBody{Code: code})
}
```

Plus one provider per status type, and an `inviteCommandProvider` producing the
`COMMAND_TOPIC_INVITE` `CREATE` with `InviteType: invite.TypeTrade` and
`ReferenceId: invite.Id(room.Handle())`.

- [ ] **Step 4: Add the data readers**

`data/character/` — `Hp(characterId) (uint32, error)`, `Level(characterId)
(uint16, error)`, `Name(characterId) (string, error)`, `Meso(characterId)
(uint32, error)`, each with `processor.go` + `requests.go` + `rest.go` +
`mock/processor.go`, copied from
`services/atlas-mini-games/atlas.com/mini-games/data/character/`.

`data/map/` — `FieldLimit(mapId) (uint32, error)` for the trade-disallowed check
(FR-4.6). Mirror `services/atlas-mini-games/.../data/map/`; the existing
mini-game create path already reads `fieldLimit&0x80` for
"cannot start a mini-room here", so read the trade-disallow bit the same way and
name the helper `tradeDisallowed(fieldLimit uint32) bool` with the bit and its
provenance in a comment. **Derive the bit from map WZ data or the client, not
from memory** — if it is not verifiable, treat an unreadable field limit as a
refusal and log at ERROR.

- [ ] **Step 5: Write the lifecycle methods**

Extend the `Processor` interface and add the impl, using the `emit`/`withTx`
outbox wrapper pattern from
`services/atlas-mini-games/.../game/processor.go:307-322`. Each command
validates in order and buffers an error event (returning `nil`) rather than
returning an error, so the buffer still flushes:

```go
// CreateRoom opens a solo trade room for characterId (FR-1.1). The validation
// ladder runs in order — dead -> map -> level -> already-in-room — and each
// failure buffers the faithful mini-room enter error rather than returning.
func (p *ProcessorImpl) CreateRoom(txId uuid.UUID, f field.Model, characterId uint32, roomType byte) error {
	return p.emit(func(p *ProcessorImpl, mb *message.Buffer) error {
		return p.createRoom(mb, txId, f, characterId, roomType)
	})
}
```

`createRoom` checks, in order: HP == 0 → `NOT_WHEN_DEAD`; field limit disallows
trade → `TRADE_NOT_ALLOWED`; level below `cfg.MinTradeLevel()` → `UNABLE`;
`reg.GetByMember` hit → `OTHER_REQUESTS`. On success it builds the room with
`NewBuilder(roomType, characterId, name, f)`, registers it, and buffers
`ROOM_CREATED` with `Position: 0`.

`Invite` requires the caller to own an `OPEN_SOLO` or `PENDING_INVITE` room,
validates the target (exists, same map, alive, not already in a room →
`INVITE_REJECTED` with the faithful result code), moves the room to
`PENDING_INVITE`, and emits the `COMMAND_TOPIC_INVITE` `CREATE`.

`DeclineInvite` removes the room and buffers `CANCELLED` with reason
`TRADE_CANCELLED` to the originator.

`EnterRoom` resolves by handle, rejects a full room with `OTHER_REQUESTS`,
re-runs the dead/map/level checks for the entering character, seats them at
position 1 via `reg.Update`, moves the room to `OPEN`, and buffers
`PARTICIPANT_ENTERED`.

`TeardownCharacter` resolves the character's room and, unless it is already
`SETTLING` (design §3.3, "cancel loses to settlement"), drops the reservations,
removes the room and buffers `CANCELLED` with the caller-supplied reason key.

- [ ] **Step 6: Wire the consumers**

`kafka/consumer/trade/consumer.go` — `InitConsumers`/`InitHandlers` over
`trademsg.EnvCommandTopic`, one handler per command type, each type-guarded
(`if c.Type != trademsg.CommandTypeCreateRoom { return }`) and rebuilding the
field with `field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()`.
Copy the shape from `services/atlas-mini-games/.../kafka/consumer/minigame/consumer.go:20-88`.

`kafka/consumer/invite/consumer.go` — consumes `EVENT_TOPIC_INVITE_STATUS`,
filtering `e.InviteType == invite.TypeTrade`, mapping `ACCEPTED` →
`EnterRoom(txId, f, e.Body.TargetId, uint32(e.ReferenceId))` and `REJECTED` →
`DeclineInvite`. Handle expiry (FR-2.6) on whichever status the invites service
emits for a timeout; if it emits `REJECTED`, that arm already covers it — verify
against `services/atlas-invites` rather than assuming.

Register both in `main.go` alongside the existing blocks.

- [ ] **Step 7: Run the tests**

```bash
cd services/atlas-trades/atlas.com/trades && go test -race ./... -v && go vet ./... && go build ./...
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-trades
git commit -m "feat(task-205): implement trade room create, invite, decline and enter"
```

---

### Task 18: Staging — items, meso, restrictions and reservations

> **Superseded by Task 27.** The restriction engine, the slot bookkeeping and the meso-refusal re-echo survive unchanged; the reservation half is replaced by escrow. Do not re-implement this task — Task 27 rewrites it in place.

**Files:**
- Modify: `services/atlas-trades/atlas.com/trades/trade/processor.go`
- Create: `services/atlas-trades/atlas.com/trades/trade/restriction.go`
- Create: `services/atlas-trades/atlas.com/trades/data/inventory/`, `data/item/` (REST readers)
- Create: `services/atlas-trades/atlas.com/trades/compartment/processor.go`, `compartment/producer.go` (reserve/cancel commands)
- Test: `services/atlas-trades/atlas.com/trades/trade/processor_staging_test.go`, `trade/restriction_test.go`

**Interfaces:**
- Consumes: Task 7's `ExpirySeconds` reserve body, Task 8's `tradeBlock`, Task 13's config.
- Produces (consumed by Task 19):
  - `trade.Processor.PutItem(txId uuid.UUID, characterId uint32, inventoryType byte, slot int16, quantity uint16, targetSlot byte) error`
  - `trade.Processor.AddMeso(txId uuid.UUID, characterId uint32, amount int32) error`
  - `trade.RefreshReservations(l, ctx) error` — the ticker entry point
  - `restriction.Check(ctx, a AssetView, d ItemDataView, source byte) error` returning a named rule error

FR-3, FR-4.1-4.4, FR-4.8; design §5.3 (reserve, don't escrow), §7 (the
restriction table), §4.2 (meso rejection = authoritative re-echo).

- [ ] **Step 1: Write the failing restriction tests**

Create `services/atlas-trades/atlas.com/trades/trade/restriction_test.go`:

```go
package trade

import "testing"

// TestRestrictionsRejectUntradeableFlags pins FR-4.1 against both flags.
func TestRestrictionsRejectUntradeableFlags(t *testing.T) {
	for name, flags := range map[string]uint16{
		"untradeable":      uint16(asset.FlagUntradeable),
		"mergeUntradeable": uint16(asset.FlagMergeUntradeable),
	} {
		t.Run(name, func(t *testing.T) {
			err := checkRestrictions(assetView{Flags: flags}, itemDataView{}, inventoryTypeUse)
			if err == nil {
				t.Fatal("staging accepted an untradeable asset")
			}
		})
	}
}

// TestRestrictionsRejectTradeBlock pins FR-4.2.
func TestRestrictionsRejectTradeBlock(t *testing.T) {
	if err := checkRestrictions(assetView{}, itemDataView{TradeBlock: true}, inventoryTypeUse); err == nil {
		t.Fatal("staging accepted a tradeBlock item")
	}
}

// TestRestrictionsRejectUnreadableItemData pins the PRD's explicit rule: a
// missing flag must NOT be read as "tradeable". An atlas-data LOOKUP FAILURE
// (not a false value) is a refusal, logged at ERROR (design §7).
func TestRestrictionsRejectUnreadableItemData(t *testing.T) {
	if err := checkRestrictions(assetView{}, itemDataView{Unreadable: true}, inventoryTypeUse); err == nil {
		t.Fatal("staging accepted an item whose data could not be read")
	}
}

// TestRestrictionsRejectQuestAndEquippedSources pins FR-4.3 and FR-4.4.
func TestRestrictionsRejectQuestAndEquippedSources(t *testing.T) {
	for name, source := range map[string]byte{
		"quest":    inventoryTypeQuest,
		"equipped": inventoryTypeEquipped,
	} {
		t.Run(name, func(t *testing.T) {
			if err := checkRestrictions(assetView{}, itemDataView{}, source); err == nil {
				t.Fatalf("staging accepted a %s-sourced asset", name)
			}
		})
	}
}

// TestRestrictionsAcceptAPlainTradeableItem guards against an over-broad rule.
func TestRestrictionsAcceptAPlainTradeableItem(t *testing.T) {
	if err := checkRestrictions(assetView{}, itemDataView{}, inventoryTypeUse); err != nil {
		t.Fatalf("staging refused a plain tradeable item: %v", err)
	}
}
```

- [ ] **Step 2: Write the failing staging tests**

Create `services/atlas-trades/atlas.com/trades/trade/processor_staging_test.go`:

```go
package trade

import "testing"

// TestPutItemReservesRatherThanEscrows pins design §5.3 Option A: staging is a
// reservation, NOT a release. Nothing leaves the owner's inventory until
// settlement, so a crash orphans nothing.
func TestPutItemReservesRatherThanEscrows(t *testing.T) {
	p, emitted := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, 2, 1, 5, 3); err != nil {
		t.Fatalf("put item: %v", err)
	}
	assertReserveCommand(t, emitted, 100, 2, 1, 5)
	assertNoCommandOfType(t, emitted, "RELEASE_FROM_CHARACTER")
	assertNoSagaSubmitted(t, emitted)
}

// TestPutItemReserveUsesTheConfiguredTtl pins design §5.3: 300s by default, not
// the 30s drop-reservation lifetime.
func TestPutItemReserveUsesTheConfiguredTtl(t *testing.T) {
	p, emitted := testOpenRoom(t)
	_ = p.PutItem(uuid.New(), 100, 2, 1, 5, 3)
	cmd := assertReserveCommand(t, emitted, 100, 2, 1, 5)
	if cmd.Body.ExpirySeconds != 300 {
		t.Errorf("expirySeconds: got %d, want 300", cmd.Body.ExpirySeconds)
	}
}

// TestPutItemRejectsOccupiedOrOutOfRangeSlot pins FR-3.3.
func TestPutItemRejectsOccupiedOrOutOfRangeSlot(t *testing.T) {
	p, _ := testOpenRoom(t)
	_ = p.PutItem(uuid.New(), 100, 2, 1, 5, 3)

	for name, slot := range map[string]byte{"occupied": 3, "zero": 0, "above nine": 10} {
		t.Run(name, func(t *testing.T) {
			_ = p.PutItem(uuid.New(), 100, 2, 2, 1, slot)
			room, _ := p.RoomForCharacter(100)
			pt, _ := room.ParticipantFor(100)
			if len(pt.Items()) != 1 {
				t.Errorf("items: got %d, want the original 1", len(pt.Items()))
			}
		})
	}
}

// TestPutItemHonoursMaxStagedItems pins FR-9.1's configurable cap.
func TestPutItemHonoursMaxStagedItems(t *testing.T) {
	p, _ := testOpenRoomWithConfig(t, configuration.DefaultConfig().WithMaxStagedItems(2))
	_ = p.PutItem(uuid.New(), 100, 2, 1, 1, 1)
	_ = p.PutItem(uuid.New(), 100, 2, 2, 1, 2)
	_ = p.PutItem(uuid.New(), 100, 2, 3, 1, 3)

	room, _ := p.RoomForCharacter(100)
	pt, _ := room.ParticipantFor(100)
	if len(pt.Items()) != 2 {
		t.Errorf("items: got %d, want the configured cap of 2", len(pt.Items()))
	}
}

// TestPutItemDropsSilentlyOnRestrictionFailure pins design §7: the reference
// client has no put-item-time error for "untradeable", so the empty slot IS the
// feedback. No clientbound update, no error event — but a server-side log.
func TestPutItemDropsSilentlyOnRestrictionFailure(t *testing.T) {
	p, emitted := testOpenRoomWithUntradeableAsset(t)
	_ = p.PutItem(uuid.New(), 100, 2, 1, 1, 1)

	assertNoEventOfType(t, emitted, "ITEM_STAGED")
	assertNoEventOfType(t, emitted, "ERROR")
	room, _ := p.RoomForCharacter(100)
	pt, _ := room.ParticipantFor(100)
	if len(pt.Items()) != 0 {
		t.Errorf("items: got %d, want 0", len(pt.Items()))
	}
}

// TestStagingIsFrozenAfterFirstConfirm pins FR-3.6 / design §3.2: from the
// moment EITHER side confirms, both PUT_ITEM and ADD_MESO are rejected. The
// reference client enforces this locally too, so a server-side rejection here
// means a modified client — log at WARN, no clientbound response.
func TestStagingIsFrozenAfterFirstConfirm(t *testing.T) {
	p, emitted := testOpenRoom(t)
	_ = p.Confirm(uuid.New(), 100, nil)

	_ = p.PutItem(uuid.New(), 200, 2, 1, 1, 1)
	_ = p.AddMeso(uuid.New(), 200, 1000)

	assertNoEventOfType(t, emitted, "ITEM_STAGED")
	assertNoEventOfType(t, emitted, "MESO_STAGED")
}

// TestAddMesoIsAbsoluteNotADelta pins design §1.6: PutMoney sends the absolute
// total from the input box, and the clientbound echo is an assignment.
func TestAddMesoIsAbsoluteNotADelta(t *testing.T) {
	p, _ := testOpenRoomWithMeso(t, 100, 5_000_000)
	_ = p.AddMeso(uuid.New(), 100, 1_000_000)
	_ = p.AddMeso(uuid.New(), 100, 2_000_000)

	room, _ := p.RoomForCharacter(100)
	pt, _ := room.ParticipantFor(100)
	if pt.MesoStaged() != 2_000_000 {
		t.Errorf("mesoStaged: got %d, want 2000000 (absolute, not 3000000 accumulated)", pt.MesoStaged())
	}
}

// TestAddMesoRefusedReEchoesTheLastValidAmount pins FR-4.8 / design §4.2: an
// out-of-range stage is corrected by an AUTHORITATIVE re-echo so the client's
// view snaps back, plus the meso-limit arm where it exists.
func TestAddMesoRefusedReEchoesTheLastValidAmount(t *testing.T) {
	p, emitted := testOpenRoomWithMeso(t, 100, 5_000_000)
	_ = p.AddMeso(uuid.New(), 100, 1_000_000)
	_ = p.AddMeso(uuid.New(), 100, 9_999_999) // more than the character holds

	body := assertEventBody[MesoRefusedEventBody](t, emitted, "MESO_REFUSED")
	if body.LastValidAmount != 1_000_000 {
		t.Errorf("lastValidAmount: got %d, want 1000000", body.LastValidAmount)
	}

	room, _ := p.RoomForCharacter(100)
	pt, _ := room.ParticipantFor(100)
	if pt.MesoStaged() != 1_000_000 {
		t.Errorf("mesoStaged: got %d, want the last valid 1000000", pt.MesoStaged())
	}
}

// TestAddMesoRejectsNegative pins the NFR: every client value is untrusted.
func TestAddMesoRejectsNegative(t *testing.T) {
	p, emitted := testOpenRoomWithMeso(t, 100, 5_000_000)
	_ = p.AddMeso(uuid.New(), 100, -1)
	assertNoEventOfType(t, emitted, "MESO_STAGED")
}
```

- [ ] **Step 3: Run both suites and confirm they fail**

Run: `cd services/atlas-trades/atlas.com/trades && go test ./trade/ -run 'TestRestrictions|TestPutItem|TestAddMeso|TestStagingIsFrozen' -v`
Expected: FAIL — `undefined: checkRestrictions`.

- [ ] **Step 4: Write the restriction checks**

Create `services/atlas-trades/atlas.com/trades/trade/restriction.go`:

```go
package trade

import (
	"errors"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

// Named rules so the rejection log says which one fired (design §7).
var (
	errUntradeableFlag = errors.New("trade: asset carries an untradeable flag")
	errTradeBlock      = errors.New("trade: item data sets tradeBlock")
	errQuestItem       = errors.New("trade: quest items cannot be staged")
	errEquipped        = errors.New("trade: equipped items cannot be staged")
	errItemDataUnknown = errors.New("trade: item data could not be read")
)

// assetView and itemDataView are the two inputs a restriction check needs,
// decoupled from the REST models so the rules are testable without a server.
type assetView struct {
	Flags uint16
}

type itemDataView struct {
	TradeBlock bool
	// Unreadable is true when the atlas-data lookup FAILED. A failure is a
	// refusal, not a "tradeable" default (PRD FR-4.2 is explicit about this).
	Unreadable bool
}

// checkRestrictions evaluates FR-4.1..FR-4.4 at stage time. A non-nil error
// means the stage is dropped: no clientbound update, empty client slot, and a
// server-side log naming the item and the failing rule.
func checkRestrictions(a assetView, d itemDataView, source byte) error {
	if source == byte(inventory.TypeValueEquip) && isEquippedSource(source) {
		return errEquipped
	}
	if source == byte(inventory.TypeValueQuest) {
		return errQuestItem
	}
	if a.Flags&uint16(asset.FlagUntradeable) != 0 || a.Flags&uint16(asset.FlagMergeUntradeable) != 0 {
		return errUntradeableFlag
	}
	if d.Unreadable {
		return errItemDataUnknown
	}
	if d.TradeBlock {
		return errTradeBlock
	}
	return nil
}
```

Resolve the exact `asset.FlagUntradeable` / `asset.FlagMergeUntradeable`
constants and the equipped-compartment discriminator from
`libs/atlas-constants/asset/flag.go` and
`libs/atlas-constants/inventory/` — do **not** hard-code 0x08 / 0x200 or a
compartment number here. The `isEquippedSource` helper distinguishes the
EQUIPPED compartment from EQUIP inventory; read the constants rather than
inferring them.

- [ ] **Step 5: Write the staging methods**

`PutItem`:
1. Resolve the room via `reg.GetByMember`; drop and log at DEBUG if none.
2. If `room.Frozen()`, log at WARN ("staging after confirm indicates a modified
   client") and return `nil` with no clientbound response.
3. Validate `targetSlot` in `1..cfg.MaxStagedItems()` and unoccupied; drop
   otherwise.
4. Read the asset at `(inventoryType, slot)` and its atlas-data record; run
   `checkRestrictions`; on failure log at INFO with item id and rule, and drop.
5. Validate `quantity <= asset.Quantity() - alreadyReserved`; drop otherwise.
6. Issue the reserve command with `ExpirySeconds:
   uint32(cfg.ReservationTtl().Seconds())`.
7. `reg.Update` appending the `StagedItem`, then buffer `ITEM_STAGED`.

`AddMeso`:
1. Same room/frozen guards.
2. Reject `amount < 0` outright (untrusted client value) — drop, log at WARN.
3. Read the character's **current** meso fresh; if `uint32(amount)` exceeds it,
   or the counterparty's post-trade meso would exceed the cap, buffer
   `MESO_REFUSED` carrying the participant's **existing** `MesoStaged()` as
   `LastValidAmount` and leave the staged amount unchanged.
4. Otherwise `reg.Update` **assigning** (not adding) `WithMesoStaged`, and
   buffer `MESO_STAGED`.

`RefreshReservations` re-issues the reserve command for every staged item of
every live room, so the TTL never expires under a live trade (design §5.3). Task
20 drives it from the ticker.

Add `compartment/processor.go` + `compartment/producer.go` producing
`REQUEST_RESERVE` and the cancel command, mirroring
`services/atlas-channel/atlas.com/channel/compartment/producer.go:88-96` with
the new `ExpirySeconds` field.

Add `data/inventory/` (compartment + asset reads) and `data/item/` (the
atlas-data `tradeBlock` read from Task 8), each with the standard
`processor.go`/`requests.go`/`rest.go`/`mock/processor.go` quartet.

- [ ] **Step 6: Run the tests**

```bash
cd services/atlas-trades/atlas.com/trades && go test -race ./... -v && go vet ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-trades
git commit -m "feat(task-205): implement trade staging with reservations and restriction checks"
```

---

### Task 19: Confirm, attestation, settlement and the ledger write

**Files:**
- Modify: `services/atlas-trades/atlas.com/trades/trade/processor.go`
- Create: `services/atlas-trades/atlas.com/trades/trade/settlement.go`
- Create: `services/atlas-trades/atlas.com/trades/saga/processor.go`, `saga/producer.go`
- Create: `services/atlas-trades/atlas.com/trades/kafka/consumer/saga/consumer.go`
- Modify: `services/atlas-trades/atlas.com/trades/main.go`
- Test: `services/atlas-trades/atlas.com/trades/trade/processor_settlement_test.go`

**Interfaces:**
- Consumes: `saga.TradeSettlementPayload` (Task 14), `ledger.Processor` (Task 11), `configuration.Tax` (Task 13).
- Produces:
  - `trade.Processor.Confirm(txId uuid.UUID, characterId uint32, entries []trademsg.CrcEntry) error`
  - `trade.Processor.Attest(txId uuid.UUID, characterId uint32, entries []trademsg.CrcEntry) error`
  - `trade.Processor.SettlementSucceeded(txId uuid.UUID, roomId uuid.UUID) error`
  - `trade.Processor.SettlementFailed(txId uuid.UUID, roomId uuid.UUID, reason string) error`
  - `trade.Processor.ExpireAttestation(txId uuid.UUID, roomId uuid.UUID) error`

Design §6 in full: pre-checks (§6.1), the attestation round trip (§6.2), the
saga shape (§6.3), the completion-packet ordering (§6.4).

- [ ] **Step 1: Write the failing settlement tests**

Create `services/atlas-trades/atlas.com/trades/trade/processor_settlement_test.go`:

```go
package trade

import "testing"

// TestConfirmDoesNotBroadcastModeSeventeenOnFirstConfirm pins design §6.2:
// mode 17 auto-replies with an attestation, so sending it on the first confirm
// would let one side drive the other's attestation without its owner ever
// pressing Trade.
func TestConfirmDoesNotBroadcastModeSeventeenOnFirstConfirm(t *testing.T) {
	p, emitted := testStagedRoom(t)
	_ = p.Confirm(uuid.New(), 100, nil)

	assertEventOfType(t, emitted, "PARTICIPANT_CONFIRMED")
	assertNoEventOfType(t, emitted, "ATTESTATION_REQUESTED")

	room, _ := p.RoomForCharacter(100)
	if room.State() != StateOpen {
		t.Errorf("state after one confirm: got %s, want %s", room.State(), StateOpen)
	}
}

// TestBothConfirmsEnterAwaitingAttestation pins design §3.1/§6.2.
func TestBothConfirmsEnterAwaitingAttestation(t *testing.T) {
	p, emitted := testStagedRoom(t)
	_ = p.Confirm(uuid.New(), 100, nil)
	_ = p.Confirm(uuid.New(), 200, nil)

	assertEventOfType(t, emitted, "ATTESTATION_REQUESTED")
	room, _ := p.RoomForCharacter(100)
	if room.State() != StateAwaitingAttestation {
		t.Errorf("state: got %s, want %s", room.State(), StateAwaitingAttestation)
	}
	assertNoSagaSubmitted(t, emitted)
}

// TestDoubleConfirmFromOneSideIsIgnored pins that a repeated confirm cannot
// stand in for the counterparty's.
func TestDoubleConfirmFromOneSideIsIgnored(t *testing.T) {
	p, emitted := testStagedRoom(t)
	_ = p.Confirm(uuid.New(), 100, nil)
	_ = p.Confirm(uuid.New(), 100, nil)

	assertNoEventOfType(t, emitted, "ATTESTATION_REQUESTED")
}

// TestBothAttestationsSubmitOneSaga pins design §6.3: exactly one saga, whose
// transactionId is the ledger idempotency key.
func TestBothAttestationsSubmitOneSaga(t *testing.T) {
	p, emitted := testConfirmedRoom(t)
	_ = p.Attest(uuid.New(), 100, nil)
	_ = p.Attest(uuid.New(), 200, nil)

	sagas := collectSagas(t, emitted)
	if len(sagas) != 1 {
		t.Fatalf("sagas submitted: got %d, want 1", len(sagas))
	}
	if sagas[0].SagaType != "trade_transaction" {
		t.Errorf("sagaType: got %s, want trade_transaction", sagas[0].SagaType)
	}
	if len(sagas[0].Steps) != 1 || sagas[0].Steps[0].Action != "trade_settlement" {
		t.Errorf("steps: want one trade_settlement composite, got %+v", sagas[0].Steps)
	}
	room, _ := p.RoomForCharacter(100)
	if room.State() != StateSettling {
		t.Errorf("state: got %s, want %s", room.State(), StateSettling)
	}
}

// TestAttestationTimeoutSettlesAnyway pins design §3.1: the attestation is
// defence in depth, not a liveness dependency. A client that never replies must
// not be able to wedge a trade.
func TestAttestationTimeoutSettlesAnyway(t *testing.T) {
	p, emitted := testConfirmedRoom(t)
	_ = p.Attest(uuid.New(), 100, nil)

	room, _ := p.RoomForCharacter(100)
	if err := p.ExpireAttestation(uuid.New(), room.Id()); err != nil {
		t.Fatalf("expire: %v", err)
	}

	if len(collectSagas(t, emitted)) != 1 {
		t.Error("attestation timeout did not settle the trade")
	}
}

// TestCrcMismatchTearsDownWithStatusThirteen pins design §6.1 check 4.
func TestCrcMismatchTearsDownWithStatusThirteen(t *testing.T) {
	p, emitted := testConfirmedRoomWithCrc(t)
	_ = p.Attest(uuid.New(), 100, []trademsg.CrcEntry{{Data: 2000000, Crc: 0xDEADBEEF}})
	_ = p.Attest(uuid.New(), 200, nil)

	assertCancelledWithReason(t, emitted, "TRADE_CRC_FAILED")
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("room survived a CRC mismatch")
	}
}

// TestPreCheckFailureTearsDownRatherThanKeepingTheRoom pins design §6.1's
// correction of PRD FR-4.9: CTradingRoomDlg::OnLeave closes the dialog before
// showing the notice, so there is no client state in which the room survives a
// status 8/9/13.
func TestPreCheckFailureTearsDownRatherThanKeepingTheRoom(t *testing.T) {
	p, emitted := testConfirmedRoomWithFullInventory(t, 200)
	_ = p.Attest(uuid.New(), 100, nil)
	_ = p.Attest(uuid.New(), 200, nil)

	assertCancelledWithReason(t, emitted, "TRADE_CANNOT_CARRY")
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("room survived a settlement pre-check failure")
	}
	assertNoSagaSubmitted(t, emitted)
}

// TestCancelLosesToSettlement pins FR-6.5 / design §3.3: once the room is
// SETTLING, a cancel is recorded and ignored; the saga's terminal status
// produces the client's LEAVE.
func TestCancelLosesToSettlement(t *testing.T) {
	p, emitted := testSettlingRoom(t)
	_ = p.TeardownCharacter(uuid.New(), 100, "TRADE_CANCELLED")

	assertNoEventOfType(t, emitted, "CANCELLED")
	if _, ok := p.RoomForCharacter(100); !ok {
		t.Error("a cancel tore down a settling room")
	}
}

// TestSettlementSuccessWritesTheLedgerAndEmitsSettled pins FR-5.6 / FR-7.1 and
// design §6.4's ordering: SETTLED is emitted only after the saga reports
// terminal success.
func TestSettlementSuccessWritesTheLedgerAndEmitsSettled(t *testing.T) {
	p, emitted, ledgerDb := testSettlingRoomWithLedger(t)
	room, _ := p.RoomForCharacter(100)

	if err := p.SettlementSucceeded(uuid.New(), room.Id()); err != nil {
		t.Fatalf("settle: %v", err)
	}

	assertEventOfType(t, emitted, "SETTLED")
	entries := readLedger(t, ledgerDb)
	if len(entries) != 1 {
		t.Fatalf("ledger entries: got %d, want 1", len(entries))
	}
	if len(entries[0].Sides()) != 2 {
		t.Errorf("ledger sides: got %d, want 2", len(entries[0].Sides()))
	}
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("room survived settlement")
	}
}

// TestSettlementFailureEmitsStatusEightAndWritesNoLedgerRow pins FR-5.3 and
// FR-7.3: a failed trade is observable via logs and metrics only.
func TestSettlementFailureEmitsStatusEightAndWritesNoLedgerRow(t *testing.T) {
	p, emitted, ledgerDb := testSettlingRoomWithLedger(t)
	room, _ := p.RoomForCharacter(100)

	_ = p.SettlementFailed(uuid.New(), room.Id(), "TRADE_FAILED")

	assertCancelledWithReason(t, emitted, "TRADE_FAILED")
	if entries := readLedger(t, ledgerDb); len(entries) != 0 {
		t.Errorf("ledger entries: got %d, want 0 for a failed trade", len(entries))
	}
}

// TestSettlementTaxIsResolvedBeforeTheSagaLeaves pins design §6.3: the tax is
// computed in atlas-trades (it needs the tenant config) and passed to the
// orchestrator as resolved integers.
func TestSettlementTaxIsResolvedBeforeTheSagaLeaves(t *testing.T) {
	p, emitted := testConfirmedRoomWithMeso(t, 100, 10_000_000)
	_ = p.Attest(uuid.New(), 100, nil)
	_ = p.Attest(uuid.New(), 200, nil)

	sagas := collectSagas(t, emitted)
	payload := sagas[0].Steps[0].Payload.(saga.TradeSettlementPayload)
	if payload.Sides[0].MesoStaged != 10_000_000 {
		t.Errorf("mesoStaged: got %d, want 10000000", payload.Sides[0].MesoStaged)
	}
	if payload.Sides[0].MesoTax != 400_000 {
		t.Errorf("mesoTax: got %d, want 400000", payload.Sides[0].MesoTax)
	}
	if payload.Sides[0].MesoDelivered != 9_600_000 {
		t.Errorf("mesoDelivered: got %d, want 9600000", payload.Sides[0].MesoDelivered)
	}
}
```

- [ ] **Step 2: Run them and confirm they fail**

Run: `cd services/atlas-trades/atlas.com/trades && go test ./trade/ -run 'TestConfirm|TestBoth|TestDouble|TestAttestation|TestCrc|TestPreCheck|TestCancelLoses|TestSettlement' -v`
Expected: FAIL — `p.Confirm undefined`.

- [ ] **Step 3: Write the confirm and attestation flow**

`Confirm(txId, characterId, entries)`:
1. Room + `Frozen()`-aware guard: reject if the room is not `OPEN`, or if this
   participant has already confirmed.
2. Store the participant's CRC entries alongside `WithConfirmed(true)` — they are
   the fallback attestation on timeout (design §3.1).
3. Buffer `PARTICIPANT_CONFIRMED` to the counterparty.
4. If **both** are now confirmed, transition to `AWAITING_ATTESTATION` and buffer
   `ATTESTATION_REQUESTED` to **both** — never on the first confirm.
5. Arm the attestation deadline via `routine.Go` + a `time.Timer` of
   `cfg.AttestationTimeout()` calling `ExpireAttestation(txId, roomId)`. The
   timer must be cancelled when the room leaves `AWAITING_ATTESTATION`.

`Attest(txId, characterId, entries)`:
1. Reject unless the room is `AWAITING_ATTESTATION`.
2. Record `WithAttested(true)` and the entries.
3. When both have attested (or on `ExpireAttestation`), run the settlement
   pre-checks and either tear down or submit the saga.

`ExpireAttestation(txId, roomId)` proceeds with whatever attestation arrived,
falling back to the `TRADE_CONFIRM` CRC lists for the missing side.

- [ ] **Step 4: Write the settlement pre-checks and saga submission**

Create `services/atlas-trades/atlas.com/trades/trade/settlement.go`:

```go
// settle runs the design §6.1 pre-checks against FRESH reads, then submits one
// trade_settlement saga whose transactionId is also the ledger's idempotency
// key (FR-5.7).
//
// A refusal TEARS THE ROOM DOWN (design §6.1's correction of PRD FR-4.9):
// CTradingRoomDlg::OnLeave closes the dialog before showing any of these
// notices, so there is no client state in which the room survives a status
// 8/9/13. Under the reserve-at-staging model nothing needs reverting anyway.
//
// Failure -> LEAVE status mapping:
//   (1) free slots            -> TRADE_CANNOT_CARRY  (9)
//   (2) meso cap              -> TRADE_CANNOT_CARRY  (9)
//   (3) reservation lost      -> TRADE_FAILED        (8)
//   (4) CRC mismatch          -> TRADE_CRC_FAILED    (13)
func (p *ProcessorImpl) settle(mb *message.Buffer, txId uuid.UUID, room Room) error {
	cfg := configuration.GetRegistry().Get(p.l, p.ctx)

	if reason, ok := p.preCheck(room, cfg); !ok {
		return p.teardown(mb, txId, room, reason)
	}

	payload := saga.TradeSettlementPayload{
		TransactionId: txId,
		WorldId:       room.Field().WorldId(),
		ChannelId:     room.Field().ChannelId(),
		RoomType:      room.RoomType(),
	}
	for i, participant := range orderedParticipants(room) {
		tax, delivered := configuration.Tax(cfg, participant.MesoStaged())
		side := saga.TradeSettlementSide{
			CharacterId:   participant.CharacterId(),
			MesoStaged:    participant.MesoStaged(),
			MesoTax:       tax,
			MesoDelivered: delivered,
		}
		for _, item := range participant.Items() {
			side.Items = append(side.Items, saga.TradeSettlementItem{
				InventoryType: item.InventoryType(),
				SourceSlot:    item.SourceSlot(),
				AssetId:       item.AssetId(),
				TemplateId:    item.TemplateId(),
				Quantity:      item.Quantity(),
			})
		}
		payload.Sides[i] = side
	}

	if _, err := p.reg.Update(p.t, room.Id(), func(r Room) (Room, error) {
		if r.State() != StateAwaitingAttestation {
			return Room{}, ErrRoomFrozen
		}
		return r.WithState(StateSettling), nil
	}); err != nil {
		// Another goroutine already drove settlement — compare-and-set lost, so
		// this attempt is a no-op rather than a second saga (design §12).
		return nil
	}

	return mb.Put(sagamsg.EnvCommandTopic, settlementSagaProvider(txId, room, payload))
}
```

`preCheck` implements design §6.1's four checks against fresh reads: free slots
per compartment for each side's incoming items (counting stackable merges), each
side's meso within `[0, mesoCap]` after `+incoming_post_tax − outgoing`, every
reservation still live at the staged quantity, and — where the version carries
CRC entries — every attested `{itemId, crc}` matching the staged assets.

`SettlementSucceeded` writes the ledger row via `ledger.Processor.Record`, buffers
`SETTLED`, drops the reservations, and removes the room. Per design §6.4 it runs
**only** on terminal saga success, so the client's meso figure is already
updated; a race degrades the message from `SP_408` to `SP_407`, which is
cosmetic and accepted.

`SettlementFailed` buffers `CANCELLED` with `TRADE_FAILED`, drops the
reservations, and removes the room. No ledger row (FR-7.3).

`saga/producer.go` builds the saga command with `SagaType:
saga.TradeTransaction`, `InitiatedBy: "atlas-trades"`, and the single
`trade_settlement` step. `kafka/consumer/saga/consumer.go` consumes
`EVENT_TOPIC_SAGA_STATUS`, filters by the transaction ids atlas-trades submitted,
and routes terminal success/failure into the two methods above.

- [ ] **Step 5: Add the metrics**

Register the design §12 counters in the processor:
`trade_rooms_opened`, `trade_settled`, `trade_cancelled`,
`trade_settlement_failed{reason}`, `trade_meso_taxed_total`,
`trade_reservation_expired`. Increment each at its state transition, alongside a
structured log carrying tenant, room id, both character ids and the transaction
id.

- [ ] **Step 6: Run the tests**

```bash
cd services/atlas-trades/atlas.com/trades && go test -race ./... -v && go vet ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-trades
git commit -m "feat(task-205): implement confirm, attestation, settlement and the ledger write"
```

---

### Task 20: Teardown consumers and the reservation-refresh ticker

> **Superseded by Task 28.** The four teardown arms survive; each now unwinds escrow first, and the reservation-refresh ticker is replaced by the stuck-escrow retry ticker.

**Files:**
- Create: `services/atlas-trades/atlas.com/trades/kafka/consumer/character/consumer.go`, `kafka/consumer/session/consumer.go`
- Modify: `services/atlas-trades/atlas.com/trades/main.go`
- Test: `services/atlas-trades/atlas.com/trades/kafka/consumer/character/consumer_test.go`

**Interfaces:**
- Consumes: `trade.Processor.TeardownCharacter` (Task 17), `trade.RefreshReservations` (Task 18).
- Produces: the four teardown arms and the refresh ticker.

Design §3.3's trigger table and §12's crash-recovery posture.

- [ ] **Step 1: Write the failing teardown tests**

Create `services/atlas-trades/atlas.com/trades/kafka/consumer/character/consumer_test.go`:

```go
package character

import "testing"

// TestLogoutTearsDownWithCancelledReason pins design §3.3: a disconnect sends
// the survivor LEAVE 2 (TRADE_CANCELLED), and escrow recovery must not depend
// on the disconnecting client being reachable (FR-6.4). Under the
// reserve-at-staging model there is nothing to recover — the reservations
// simply drop.
func TestLogoutTearsDownWithCancelledReason(t *testing.T) {
	p, emitted := testRoomWithBothSides(t)
	handleStatusEventLogout(testDb(t))(logger, ctx, logoutEvent(100))
	assertCancelledWithReason(t, emitted, "TRADE_CANCELLED")
	if _, ok := p.RoomForCharacter(200); ok {
		t.Error("survivor still in a room after the counterparty logged out")
	}
}

// TestMapChangeTearsDownWithDifferentMapReason pins design §3.3's row: a map
// change is LEAVE 12, not LEAVE 2 — the client has a distinct string for it
// (SP_411 "the other person's on a different map").
func TestMapChangeTearsDownWithDifferentMapReason(t *testing.T) {
	_, emitted := testRoomWithBothSides(t)
	handleStatusEventMapChanged(testDb(t))(logger, ctx, mapChangedEvent(100))
	assertCancelledWithReason(t, emitted, "TRADE_DIFFERENT_MAP")
}

// TestChannelChangeTearsDownWithDifferentMapReason pins the same row for a
// channel change, which emits neither LOGOUT nor MAP_CHANGED — without this arm
// the member index keeps the character bound to a dead room forever.
func TestChannelChangeTearsDownWithDifferentMapReason(t *testing.T) {
	_, emitted := testRoomWithBothSides(t)
	handleStatusEventChannelChanged(testDb(t))(logger, ctx, channelChangedEvent(100))
	assertCancelledWithReason(t, emitted, "TRADE_DIFFERENT_MAP")
}

// TestTeardownIsIgnoredWhileSettling pins FR-6.5 at the consumer boundary too.
func TestTeardownIsIgnoredWhileSettling(t *testing.T) {
	p, emitted := testSettlingRoom(t)
	handleStatusEventLogout(testDb(t))(logger, ctx, logoutEvent(100))
	assertNoEventOfType(t, emitted, "CANCELLED")
	if _, ok := p.RoomForCharacter(100); !ok {
		t.Error("a logout tore down a settling room")
	}
}
```

- [ ] **Step 2: Run them and confirm they fail**

Run: `cd services/atlas-trades/atlas.com/trades && go test ./kafka/consumer/character/ -v`
Expected: FAIL — `undefined: handleStatusEventLogout`.

- [ ] **Step 3: Write the consumers**

`kafka/consumer/character/consumer.go` — copy
`services/atlas-mini-games/.../kafka/consumer/character/consumer.go:1-94`
verbatim in shape, with three handlers calling
`trade.NewProcessor(l, ctx).TeardownCharacter(...)` and the design §3.3 reason
per trigger:

| Handler | Event type | Reason key |
|---|---|---|
| `handleStatusEventLogout` | `LOGOUT` | `TRADE_CANCELLED` |
| `handleStatusEventMapChanged` | `MAP_CHANGED` | `TRADE_DIFFERENT_MAP` |
| `handleStatusEventChannelChanged` | `CHANNEL_CHANGED` | `TRADE_DIFFERENT_MAP` |

`kafka/consumer/session/consumer.go` — the `SESSION_DESTROYED` arm, guarded by
`if e.CharacterId == 0 { return }`, reason `TRADE_CANCELLED`.

- [ ] **Step 4: Add the refresh ticker**

In `main.go`, after the outbox drainer:

```go
	// Refresh the inventory reservations of every live room at TTL/3 so a trade
	// window never outlives its reservations (design §5.3). Expiry is a backstop
	// for a DEAD room, not a normal-path event: if it does fire mid-trade,
	// settlement fails cleanly with LEAVE 8.
	routine.Go(l, rt.Context(), func(ctx context.Context) {
		ticker := time.NewTicker(trade.ReservationRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := trade.RefreshAllReservations(l, ctx); err != nil {
					l.WithError(err).Error("Unable to refresh trade reservations.")
				}
			}
		}
	})
```

`trade.ReservationRefreshInterval` is derived from the configured TTL divided by
three; export it as a function of the config rather than a package constant if
the TTL is per-tenant.

Register both consumers in `main.go` alongside the trade and invite ones.

- [ ] **Step 5: Run the tests and the goroutine guard**

```bash
cd services/atlas-trades/atlas.com/trades && go test -race ./... -v && go vet ./...
cd "$(git rev-parse --show-toplevel)"
tools/goroutine-guard.sh
```

Expected: PASS; the guard exits 0 (every goroutine goes through `routine.Go`).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-trades
git commit -m "feat(task-205): add trade teardown consumers and the reservation-refresh ticker"
```

---

### Task 21: atlas-channel trade processor and handler arms

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/trade/processor.go`, `trade/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_interaction.go` (twelve arms)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_interaction_trade_test.go`

**Interfaces:**
- Consumes: the Task 16 contract mirror.
- Produces:
  - `trade.NewProcessor(l, ctx) *Processor` with `CreateRoom`, `Invite`, `DeclineInvite`, `PutItem`, `AddMeso`, `Confirm`, `Transaction`, `Cancel`, `Chat`, `InGame(characterId) (bool, error)` — a thin fire-and-forget Kafka producer, mirroring `services/atlas-channel/atlas.com/channel/minigame/processor.go:16-68`.

PRD §7's atlas-channel row; design §2.1 (the cross-family occupancy check lives
here because atlas-channel is the only component holding all three views).

- [ ] **Step 1: Write the failing handler tests**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_interaction_trade_test.go`:

```go
package handler

import "testing"

// TestCreateTradeRoomProducesACommand pins that the CREATE arm no longer
// l.Debugf-and-returns.
func TestCreateTradeRoomProducesACommand(t *testing.T) {
	produced := testHandleInteraction(t, createTradePacket(t, 3))
	cmd := assertTradeCommand(t, produced, trade.CommandTypeCreateRoom)
	if cmd.Body.RoomType != 3 {
		t.Errorf("roomType: got %d, want 3", cmd.Body.RoomType)
	}
}

// TestCreateTradeRoomBlockedByAnExistingMiniGameRoom pins design §2.1: the
// cross-family occupancy check lives in atlas-channel because it is the only
// component that already holds all three views. A collision replies
// OTHER_REQUESTS (mini-room enter error 3), per FR-1.2.
func TestCreateTradeRoomBlockedByAnExistingMiniGameRoom(t *testing.T) {
	announced, produced := testHandleInteractionWithMiniGameMembership(t, createTradePacket(t, 3))
	assertNoTradeCommand(t, produced)
	assertEnterResultError(t, announced, interactioncb.CharacterInteractionEnterErrorModeOtherRequests)
}

// TestCreateTradeRoomBlockedByAnExistingShop pins the same for atlas-merchant.
func TestCreateTradeRoomBlockedByAnExistingShop(t *testing.T) {
	announced, produced := testHandleInteractionWithShopMembership(t, createTradePacket(t, 3))
	assertNoTradeCommand(t, produced)
	assertEnterResultError(t, announced, interactioncb.CharacterInteractionEnterErrorModeOtherRequests)
}

// TestTradePutItemForwardsEveryDecodedField pins that no field is dropped
// between the codec and the command.
func TestTradePutItemForwardsEveryDecodedField(t *testing.T) {
	produced := testHandleInteraction(t, tradePutItemPacket(t, 2, 5, 100, 3))
	cmd := assertTradeCommandBody[trade.PutItemCommandBody](t, produced, trade.CommandTypePutItem)
	if cmd.InventoryType != 2 || cmd.Slot != 5 || cmd.Quantity != 100 || cmd.TargetSlot != 3 {
		t.Errorf("body: got %+v", cmd)
	}
}

// TestTradeConfirmForwardsTheCrcEntries pins that the attestation payload
// survives — on GMS <= v79 the list is legitimately empty (tradeCrcPresent).
func TestTradeConfirmForwardsTheCrcEntries(t *testing.T) {
	produced := testHandleInteraction(t, tradeConfirmPacket(t, []crc{{100, 200}, {300, 400}}))
	cmd := assertTradeCommandBody[trade.ConfirmCommandBody](t, produced, trade.CommandTypeConfirm)
	if len(cmd.Entries) != 2 {
		t.Fatalf("entries: got %d, want 2", len(cmd.Entries))
	}
}

// TestExitNotifiesTheTradeProcessorAlongsideTheOthers pins that the EXIT arm
// fans out to all THREE families now, not two.
func TestExitNotifiesTheTradeProcessorAlongsideTheOthers(t *testing.T) {
	produced := testHandleInteraction(t, exitPacket(t))
	assertTradeCommand(t, produced, trade.CommandTypeCancel)
}
```

- [ ] **Step 2: Run them and confirm they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run Trade -v`
Expected: FAIL — `undefined: trade.CommandTypeCreateRoom` / no command produced.

- [ ] **Step 3: Write the channel-side processor**

Create `services/atlas-channel/atlas.com/channel/trade/processor.go`, mirroring
`minigame/processor.go:16-68`:

```go
// Processor is a thin fire-and-forget producer onto COMMAND_TOPIC_TRADE.
// atlas-channel never mutates inventory or meso for trade — all trade state
// lives in atlas-trades (task-205 design §2.2).
type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

func (p *Processor) CreateRoom(f field.Model, characterId uint32, roomType byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(trade2.EnvCommandTopic)(CreateRoomCommandProvider(uuid.New(), f, characterId, roomType))
}

func (p *Processor) PutItem(f field.Model, characterId uint32, inventoryType byte, slot int16, quantity uint16, targetSlot byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(trade2.EnvCommandTopic)(PutItemCommandProvider(uuid.New(), f, characterId, inventoryType, slot, quantity, targetSlot))
}
```

…and one method per command type. Add `InGame(characterId) (bool, error)` as a
REST read against `GET /trades/rooms?filter[characterId]=N` for the cross-family
check, mirroring `minigame/processor.go:28-52`.

`trade/producer.go` holds one `…CommandProvider` per command, keyed by map id.

- [ ] **Step 4: Replace the handler arms**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_interaction.go`:

- `CREATE` roomType 3 (`:87-90`) and roomType 6 (`:132-134`): run the
  cross-family occupancy check, then call `trade.NewProcessor(l, ctx).CreateRoom`.
  The check reads all three views the handler already has access to:

```go
			if roomType == model.TradeMiniRoomType || roomType == model.CashTradeMiniRoomType {
				// Cross-family occupancy (FR-1.2). The three registries live in
				// three services with no shared occupancy store, so this check is
				// BEST EFFORT and races are resolved by whichever room the client
				// actually ends up in; atlas-trades still enforces its own
				// single-room invariant authoritatively (design §2.1).
				if occupied, err := characterOccupiesAnyMiniRoom(l, ctx, s.CharacterId()); err != nil {
					l.WithError(err).Warnf("Unable to check mini-room occupancy for character [%d]; proceeding.", s.CharacterId())
				} else if occupied {
					_ = session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(interactioncb.CharacterInteractionEnterResultErrorBody(interactioncb.CharacterInteractionEnterErrorModeOtherRequests))(s)
					return
				}
				_ = trade.NewProcessor(l, ctx).CreateRoom(s.Field(), s.CharacterId(), byte(roomType))
				return
			}
```

  `characterOccupiesAnyMiniRoom` queries `minigame.NewProcessor(l, ctx).InGame`,
  `merchant.NewProcessor(l, ctx).GetVisitingShop` and
  `trade.NewProcessor(l, ctx).InGame`, returning true on the first hit.

- `INVITE` (`:134-140`) → `trade…Invite(s.Field(), s.CharacterId(), sp.TargetCharacterId())`.
- `INVITE_DECLINE` (`:141-146`) → `trade…DeclineInvite(...)`.
- `CASH_TRADE_OPEN` nProc 0 / nProc 4 with roomType 6 (`:255-262`) → `CreateRoom`
  / `EnterRoom` for the cash room. Gate this on the per-version resolution from
  Task 6: on versions where `CCashTradingRoomDlg` does not exist, the handler key
  is not in that template, so the arm is unreachable there by construction.
- `TRADE_PUT_ITEM` (`:292-297`), `TRADE_ADD_MESO` (`:298-303`),
  `TRADE_CONFIRM` (`:304-310`), `TRANSACTION` (`:311-317`) → the matching
  processor call, forwarding every decoded field.
- `VISIT` (`:147-183`), `CHAT` (`:184-200`), `EXIT` (`:201-236`): add the trade
  processor to the existing minigame/merchant fan-out. Follow the established
  comment convention — "each service drops X from characters that are not
  members of one of its rooms".

Keep every `l.Debugf` line: they are the existing observability and the arms
should log **and** act, not log **or** act.

- [ ] **Step 5: Run the tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./... -v && go vet ./... && go build ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel
git commit -m "feat(task-205): wire the atlas-channel trade handler arms to atlas-trades"
```

---

### Task 22: atlas-channel trade status consumer

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/trade/consumer.go`
- Modify: the channel's consumer registration site (alongside the minigame consumer's registration)
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/trade/consumer_test.go`

**Interfaces:**
- Consumes: the Task 16 status events and every clientbound body from Tasks 1-5.
- Produces: the clientbound writes that make the trade window work.

Copy the shape of
`services/atlas-channel/atlas.com/channel/kafka/consumer/minigame/consumer.go`
(490 lines) exactly: `consumer.SetStartOffset(kafka.LastOffset)` so stale events
are not replayed into live sessions, the `sc server.Model` + `wp writer.Producer`
currying, the `[]listener.HandlerHandle` return, the `guard` /`announceTo` /
`announceToRoom` helpers, and one type-guarded handler per status type.

- [ ] **Step 1: Write the failing consumer tests**

Create `services/atlas-channel/atlas.com/channel/kafka/consumer/trade/consumer_test.go`:

```go
package trade

import "testing"

// TestItemStagedIsSideRelativePerRecipient pins the one thing this consumer
// must get right: the wire `side` byte is RECIPIENT-relative (0 = your own
// side, 1 = the counterparty), while the event carries an absolute room
// position. Sending the same byte to both clients puts the item on the wrong
// side of one of the two dialogs.
func TestItemStagedIsSideRelativePerRecipient(t *testing.T) {
	announced := handleAndCapture(t, itemStagedEvent(ownerId(100), visitorId(200), position(0)))

	if got := sideByteSentTo(t, announced, 100); got != 0 {
		t.Errorf("stager's own side byte: got %d, want 0", got)
	}
	if got := sideByteSentTo(t, announced, 200); got != 1 {
		t.Errorf("counterparty's side byte: got %d, want 1", got)
	}
}

// TestMesoStagedIsSideRelativePerRecipient pins the same for mode 16.
func TestMesoStagedIsSideRelativePerRecipient(t *testing.T) {
	announced := handleAndCapture(t, mesoStagedEvent(ownerId(100), visitorId(200), position(1)))
	if got := sideByteSentTo(t, announced, 200); got != 0 {
		t.Errorf("stager's own side byte: got %d, want 0", got)
	}
	if got := sideByteSentTo(t, announced, 100); got != 1 {
		t.Errorf("counterparty's side byte: got %d, want 1", got)
	}
}

// TestAttestationRequestedGoesToBothSides pins design §6.2: mode 17 is
// broadcast to BOTH clients at once, only after both confirmed.
func TestAttestationRequestedGoesToBothSides(t *testing.T) {
	announced := handleAndCapture(t, attestationRequestedEvent(ownerId(100), visitorId(200)))
	for _, id := range []uint32{100, 200} {
		if !receivedMode(t, announced, id, "TRADE_CONFIRM") {
			t.Errorf("character %d did not receive the attestation prompt", id)
		}
	}
}

// TestMesoRefusedSendsBothTheReEchoAndTheLimitArm pins design §4.2: the
// authoritative TRADE_ADD_MESO re-echo is what actually corrects the client
// (mode 16 is an ASSIGNMENT), and TRADE_MESO_LIMIT only supplies the reason.
func TestMesoRefusedSendsBothTheReEchoAndTheLimitArm(t *testing.T) {
	announced := handleAndCapture(t, mesoRefusedEvent(characterId(100), lastValid(1_000_000)))
	if !receivedMode(t, announced, 100, "TRADE_ADD_MESO") {
		t.Error("no authoritative re-echo sent")
	}
	if !receivedMode(t, announced, 100, "TRADE_MESO_LIMIT") {
		t.Error("no meso-limit reason sent")
	}
}

// TestSettledSendsLeaveSuccessToBothSides pins design §1.4: completion is
// LEAVE + slot + status 7, not a distinct mode.
func TestSettledSendsLeaveSuccessToBothSides(t *testing.T) {
	announced := handleAndCapture(t, settledEvent(ownerId(100), visitorId(200)))
	for _, id := range []uint32{100, 200} {
		assertLeaveReason(t, announced, id, interactioncb.CharacterInteractionLeaveReasonTradeSuccess)
	}
}

// TestCancelledMapsTheReasonKeyStraightThrough pins DOM-25: the event carries a
// semantic KEY and the channel resolves it via the tenant leaveReason table —
// the channel never invents a numeric status.
func TestCancelledMapsTheReasonKeyStraightThrough(t *testing.T) {
	for _, reason := range []string{"TRADE_CANCELLED", "TRADE_FAILED", "TRADE_CANNOT_CARRY", "TRADE_DIFFERENT_MAP", "TRADE_CRC_FAILED"} {
		t.Run(reason, func(t *testing.T) {
			announced := handleAndCapture(t, cancelledEvent(ownerId(100), visitorId(200), reason))
			assertLeaveReason(t, announced, 100, reason)
		})
	}
}

// TestEventsForAnotherChannelAreIgnored pins the guard every handler runs.
func TestEventsForAnotherChannelAreIgnored(t *testing.T) {
	announced := handleAndCapture(t, settledEventOnChannel(9))
	if len(announced) != 0 {
		t.Errorf("handled an event for another channel: %d writes", len(announced))
	}
}
```

- [ ] **Step 2: Run them and confirm they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/trade/ -v`
Expected: FAIL — `undefined: handleAndCapture` / package has no handlers.

- [ ] **Step 3: Write the consumer**

Create `services/atlas-channel/atlas.com/channel/kafka/consumer/trade/consumer.go`
with `InitConsumers`, `InitHandlers`, the `guard`/`fieldOf`/`announceTo`/
`announceToRoom` helpers, and this side-resolution helper — the single most
error-prone piece:

```go
// sideFor converts the event's ABSOLUTE room position (0 owner, 1 visitor) into
// the RECIPIENT-RELATIVE side byte the client reads: 0 means "my own side of
// the dialog", 1 means "the counterparty's". Sending one byte to both clients
// puts the item on the wrong side of one of the two windows.
func sideFor(stagerPosition byte, recipientPosition byte) byte {
	if stagerPosition == recipientPosition {
		return 0
	}
	return 1
}

// positionOf resolves a character's absolute position in the room from the
// event's owner/visitor ids.
func positionOf[E any](e trade2.StatusEvent[E], characterId uint32) byte {
	if characterId == e.VisitorId {
		return 1
	}
	return 0
}
```

Handler map:

| Status type | Clientbound write |
|---|---|
| `ROOM_CREATED` | `CharacterInteractionEnterResultSuccessBody(interaction.NewTradeRoom(roomType, 0, nil))` to the owner |
| `PARTICIPANT_ENTERED` | `EnterResultSuccess` with both visitors to the entrant; `CharacterInteractionEnterBody(visitor)` to the owner |
| `INVITE_SENT` | `CharacterInteractionInviteBody(roomType, inviterName, handle)` to the target |
| `INVITE_REJECTED` | `CharacterInteractionInviteResultBody(code)` to the inviter |
| `ITEM_STAGED` | `CharacterInteractionTradePutItemBody(sideFor(...), tradeSlot, asset)` to **both**, each with its own side byte |
| `MESO_STAGED` | `CharacterInteractionTradeAddMesoBody(sideFor(...), amount)` to **both** |
| `MESO_REFUSED` | `CharacterInteractionTradeAddMesoBody(0, lastValidAmount)` **plus** `CharacterInteractionTradeMesoLimitBody()` to the refused character only |
| `PARTICIPANT_CONFIRMED` | no clientbound write — the reference client shows the counterparty's confirm state locally; log only |
| `ATTESTATION_REQUESTED` | `CharacterInteractionTradeConfirmBody()` to **both** |
| `SETTLED` | `CharacterInteractionLeaveReasonBody(slot, TRADE_SUCCESS)` to both |
| `CANCELLED` | `CharacterInteractionLeaveReasonBody(slot, e.Body.Reason)` to both |
| `ERROR` | `CharacterInteractionEnterResultErrorBody(e.Body.Code)` to the acting character |
| `CHAT` | `CharacterInteractionChatBody(slot, name+" : "+message)` to both — the miniroom chat wire carries no separate name field, matching the minigame consumer's handling |

The `ITEM_STAGED` handler needs a `model.Asset` to encode. Build it from the
event body's `TemplateId`/`Quantity`/`AssetId` via the same asset-resolution the
channel already uses for inventory writes; if the channel must read the asset
back, do it once per event and reuse it for both recipients.

Register the consumer alongside the minigame one at the channel's registration
site, matching its `handles` bookkeeping so it deregisters on channel teardown.

- [ ] **Step 4: Run the tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./... -v && go vet ./... && go build ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel
git commit -m "feat(task-205): add the atlas-channel trade status consumer"
```

---

### Task 23: Route the trade keys in all ten seed templates

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_48_1.json`, `_61_`, `_72_`, `_79_`, `_83_`, `_84_`, `_87_`, `_92_`, `_95_`, `template_jms_185_1.json`
- Do **not** touch: `template_gms_12_1.json`

**Interfaces:**
- Consumes: the per-version derivation table from Task 6
  (`docs/tasks/task-205-player-trade/version-matrix.md`).
- Produces: the tenant `operations`, `leaveReason` and handler entries every
  runtime lookup depends on.

FR-11 and design §10.6. `atlas_packet.ResolveCode` returns **99** on a lookup
miss (`libs/atlas-packet/resolve.go:29`, documented as "will likely cause a
client crash") — a missing key here is a live-crash bug, not a compile error.

- [ ] **Step 1: Add the writer `operations` keys**

For each of the ten templates, add to the `CharacterInteraction` **writer**
`operations` table the four new keys at that version's byte, taken verbatim from
the Task 6 derivation table:

```
TRADE_PUT_ITEM
TRADE_ADD_MESO
TRADE_CONFIRM
TRADE_MESO_LIMIT   (omit on versions whose dispatcher has no mode-21 arm)
```

Every entry goes at its **sorted position** by `opCode`, never appended next to a
semantically-related entry.

- [ ] **Step 2: Add the `leaveReason` keys**

Add the six `TRADE_*` leave reasons at the status bytes the version's
`CTradingRoomDlg::OnLeave` branches on (Task 6 step 2.3):

```
TRADE_CANCELLED  TRADE_SUCCESS  TRADE_FAILED
TRADE_CANNOT_CARRY  TRADE_DIFFERENT_MAP  TRADE_CRC_FAILED
```

Where a version's `OnLeave` lacks a branch, omit that key for that version
rather than guessing a byte.

- [ ] **Step 3: Fill the handler-table gaps**

- `template_gms_92_1.json` (FR-11.2): add the missing **trade** handler keys —
  `INVITE`, `INVITE_DECLINE`, `TRADE_PUT_ITEM`, `TRADE_ADD_MESO`,
  `TRADE_CONFIRM`, `TRANSACTION`, `CASH_TRADE_OPEN` — at the bytes Task 6
  derived. **Do not** add the non-trade gaps (`CHAT`,
  `PERSONAL_STORE_ITEM_SOLD`, `MEMORY_GAME_PUT_STONE_ERROR`, both
  `MERCHANT_VIEW_*`, the merchant/personal-store handler keys) — they belong to
  families this task cannot verify, and adding them blind is exactly the
  "new opcodes missing from live config" bug in reverse (design §10.4).
- `TRANSACTION` (FR-11.3): add it to gms_48/61/72/79 and jms_185 **only where
  Task 6 confirmed the arm exists**. Where it does not, the matrix cell is
  ⬜ n-a with the dispatcher evidence and no key is added.
- `CASH_TRADE_OPEN` (FR-11.4, design §10.3): same rule — add only where
  `CCashTradingRoomDlg` was found.
- `TRADE_NOT_ALLOWED` enter-error key: add to `template_jms_185_1.json` (FR-11.4)
  and anywhere else the derivation shows the client has it.

- [ ] **Step 4: Verify every socket-handler entry has a validator**

A handler bound without a validator is **silently dropped** at dispatch. Check
each new handler entry carries the validator its neighbours use, and that no new
entry has an empty validator list.

- [ ] **Step 5: Run the template guards**

```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
```

Expected: all exit 0. `template-opcode-order-guard.sh` enforces strictly
ascending `opCode` in both `handlers` and `writers`;
`template-duplicate-binding-guard.sh` bans binding the same
`(implementation name, numeric opCode)` pair twice — the leading-zero-padding
duplicate (`0xB8` vs `0x0B8`) whose last-write-wins behaviour decided which
entry's options survived in task-194.

- [ ] **Step 6: Cross-check the templates against the registry**

```bash
packet-audit operations --check
```

Expected: exit 0 — every key a codec resolves is present at the version's
correct byte, and every template key maps to a real operation.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(task-205): route the trade handler and writer keys in all ten seed templates"
```

---

### Task 24: Full verification gate and review

**Files:**
- Modify: `docs/tasks/task-205-player-trade/version-matrix.md` (finalise), `docs/packets/audits/STATUS.md` (regenerate)
- Create: `services/atlas-trades/atlas.com/trades/README.md`

**Interfaces:**
- Consumes: everything.
- Produces: a green branch ready for PR.

- [ ] **Step 1: Run every project gate**

From the worktree root:

```bash
for m in libs/atlas-packet libs/atlas-saga \
         services/atlas-trades/atlas.com/trades \
         services/atlas-channel/atlas.com/channel \
         services/atlas-inventory/atlas.com/inventory \
         services/atlas-data/atlas.com/data \
         services/atlas-tenants/atlas.com/tenants \
         services/atlas-saga-orchestrator/atlas.com/saga-orchestrator; do
  (cd "$m" && echo "== $m ==" && go build ./... && go vet ./... && go test -race ./...) || exit 1
done
```

Expected: every module clean. Do not proceed past a failure.

- [ ] **Step 2: Bake every service whose `go.mod` was touched**

```bash
docker buildx bake atlas-trades atlas-channel atlas-inventory atlas-data atlas-tenants atlas-saga-orchestrator
```

Expected: all build. `go build` against `go.work` will **not** catch a missing
`COPY libs/...` line in the shared Dockerfile — only the bake will. This step is
mandatory, not optional.

- [ ] **Step 3: Run every guard**

```bash
tools/service-registration-guard.sh
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/skill-job-id-guard.sh
tools/buff-duration-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/lint.sh --check
```

Expected: all exit 0. If `tools/lint.sh --check` fails on the atlas-ui leg,
`nvm use 22` first — the guard false-fails without nvm on PATH. Run
`tools/lint.sh` (no flags) to fix formatting in place before re-checking.

- [ ] **Step 4: Confirm the matrix**

```bash
packet-audit matrix
packet-audit fname-doc --check
packet-audit operations --check
packet-audit dispatcher-lint
git diff --stat docs/packets/audits/
```

Expected: exit 0 across the board; every cell claimed in
`coverage-manifest.yaml` is ✅ or ⬜ n-a; serverbound `PLAYER_INTERACTION`
(v83 0x07B) is no longer ❌.

- [ ] **Step 5: Write the service README**

`services/atlas-trades/atlas.com/trades/README.md` documenting: the room state
machine, the two ids (`uuid` + `uint32` handle) and why, the reserve-not-escrow
decision with a pointer to design §5, the `replicas: 1` constraint, the REST
surface, the Kafka contracts, and the `trade-configs` knobs with their defaults.
Code is the source of truth — describe what is there, make no forward promises.

- [ ] **Step 6: Run the completeness critic and the code review**

Dispatch, in parallel:

- `packet-completeness-critic` on `task-205-player-trade` — diffs
  `coverage-manifest.yaml` against the branch's git and matrix delta, catching
  CHANGED-BUT-UNCLAIMED codecs and CLAIMED-BUT-UNVERIFIED cells.
- `superpowers:requesting-code-review`, which dispatches
  `plan-adherence-reviewer` and `backend-guidelines-reviewer` (no atlas-ui files
  changed, so the frontend reviewer is not needed).

Pin the review subagents to Sonnet — review workflows use the cheaper model.
Ensure every subagent operates inside this worktree and writes its artifacts to
`docs/tasks/task-205-player-trade/`, never into the main repo. Verify
`git status` is clean of stray main-repo edits after the runs.

- [ ] **Step 7: Address the findings, then commit**

Fix every confirmed finding on this branch — do not split them into a follow-up
task. Re-run steps 1-4 after any fix.

```bash
git add -A
git commit -m "docs(task-205): finalise the version matrix, service README and review artifacts"
```

- [ ] **Step 8: Record the deployment prerequisites in the PR body**

State explicitly, because neither is derivable from the diff:

1. `atlas-trades-main` must be created manually on postgres.home before the main
   deploy — main has no wave-0 create job.
2. atlas-data must be **re-ingested** before `tradeBlock` reads true on live
   data; existing records were parsed without the field (effects are ingested,
   not re-parsed on read).

---

# Slice 7 — Amendment: escrow at staging (Tasks 25-31)

Written after the first live test of Slices 1-6. Design §5A is the authority for
every decision in this slice; §5.2-§5.4 are superseded and must not be used to
justify a choice here.

**Why:** the reference client arms `m_bExclRequestSent` when it sends
`PUT_ITEM` / `PUT_MONEY` (`CTradingRoomDlg::PutItem` @`0x7c359f`,
`::PutMoney` @`0x7c37ca`) and refuses both until a server packet clears it
(`CWvsContext::CanSendExclRequest` @`0x485bf7`; cleared by the leading bool in
`OnInventoryOperation` @`0xa1ead9` / `OnStatChanged`, or by `SET_FIELD`).
Reserve-at-staging mutates nothing, so it sends none of them: the dialog latches
after the first stage. Making the mutation real emits the unlock through the
pipelines that already exist — this slice adds **no new clientbound codec**.

**Order:** 25 → 26 → 27 → 28 → 29 → 30 → 31. Tasks 25 and 26 are independent of
each other and may be parallelised; everything from 27 on is sequential.

### Task 25: `accept_to_trade` / `release_from_trade` saga actions

**Files:**
- Modify: `libs/atlas-saga/model.go` (two `Action` constants), `libs/atlas-saga/payloads.go`, `libs/atlas-saga/unmarshal.go`
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/trade/custody/kafka.go`
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/trade/processor.go`, `trade/producer.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go`, `saga/processor.go` (the `transfer_to_trade` expander), `saga/character_extractor.go`
- Test: `libs/atlas-saga/unmarshal_test.go`, `saga-orchestrator/saga/processor_trade_escrow_test.go`

**Interfaces:**
- Produces (consumed by Tasks 26, 27, 29): actions `accept_to_trade` /
  `release_from_trade`; composite `transfer_to_trade`;
  `AcceptToTradePayload` / `ReleaseFromTradePayload`; command topic
  `COMMAND_TOPIC_TRADE_CUSTODY`.

Mirror the MTS custody family exactly — it is the closest sibling and was
written for the same reason. `AcceptToMtsListingPayload`
(`libs/atlas-saga/payloads.go:673`) is the shape to copy for the item snapshot;
`kafka/message/mts/custody/kafka.go` is the shape to copy for the orchestrator's
own copy of the wire contract, including the comment explaining why the copy
exists (the orchestrator cannot import the destination service's module).

- [x] **Step 1: Write the failing unmarshal tests**

Extend `libs/atlas-saga/unmarshal_test.go` with `TestUnmarshalAcceptToTradeStep`
and `TestUnmarshalReleaseFromTradeStep`, following
`TestUnmarshalMtsBidEscrowStep` (line 699). Each asserts the `Action` constant
and that `step.Payload` type-asserts to the concrete payload — the guard against
a payload registered under the wrong action string in `unmarshal.go`.

- [x] **Step 2: Write the failing expander test**

`saga/processor_trade_escrow_test.go`: `TestExpandTransferToTradeReleasesThenAccepts`
asserts the composite expands to exactly `[release_from_character, accept_to_trade]`
in that order, with the accept payload carrying the full stat snapshot read from
the source compartment. Model it on the existing `expandTransferToStorage`
coverage.

- [x] **Step 3: Add the actions, payloads and unmarshal arms**

`AcceptToTrade Action = "accept_to_trade"`, `ReleaseFromTrade Action = "release_from_trade"`,
`TransferToTrade Action = "transfer_to_trade"`. `AcceptToTradePayload` carries
`TransactionId`, `EscrowId`, `RoomId`, `OwnerId`, `TradeSlot`,
`SourceInventoryType`, `SourceSlot` and the item snapshot block.
`ReleaseFromTradePayload` carries `TransactionId` and `EscrowId` only — the row
holds everything else, exactly as `ReleaseFromMtsHoldingPayload` does.

- [x] **Step 4: Add the orchestrator's custody producer/processor and the expander**

`trade/processor.go` exposes `AcceptToTradeAndEmit` / `AcceptToTrade(mb)` /
`ReleaseFromTradeAndEmit` / `ReleaseFromTrade(mb)`, copying `mts/processor.go`.
`expandTransferToTrade` copies `expandTransferToStorage` (`saga/processor.go:1270`):
look up the source asset, build the snapshot, emit the two steps.

- [x] **Step 5: Register the actions in the character extractor and re-run**

`saga/character_extractor.go` must resolve a character id for both new actions,
or event routing silently drops their completions. `go test -race ./...` in both
modules; `go vet ./...`; `tools/lint.sh --check`.

- [x] **Step 6: Commit** — `feat(task-205): add the accept_to_trade / release_from_trade custody actions`

---

### Task 26: The atlas-trades escrow store and custody consumer

**Files:**
- Create: `services/atlas-trades/atlas.com/trades/escrow/entity.go`, `model.go`, `builder.go`, `administrator.go`, `provider.go`, `processor.go`
- Create: `services/atlas-trades/atlas.com/trades/kafka/consumer/custody/consumer.go`
- Create: `services/atlas-trades/atlas.com/trades/kafka/message/custody/kafka.go`
- Modify: `services/atlas-trades/atlas.com/trades/main.go` (migration + consumer registration)
- Test: `escrow/administrator_test.go`, `escrow/provider_test.go`, `kafka/consumer/custody/consumer_test.go`

**Interfaces:**
- Consumes: Task 25's `COMMAND_TOPIC_TRADE_CUSTODY` contract.
- Produces (consumed by Tasks 27, 28, 29, 31):
  - `escrow.Processor.CreateItem(mb)(model) error`, `.DeleteItem(mb)(id) error`
  - `escrow.Processor.UpsertMeso(mb)(roomId, ownerId, amount) error`, `.DeleteMeso(mb)(roomId, ownerId) error`
  - `escrow.Processor.ByRoomProvider(roomId)`, `.OrphanedProvider()` — every row whose room is not in the live registry

Design §5A.3. Two tables: `trade_escrow_items` and `trade_escrow_mesos`.

- [x] **Step 1: Write the failing administrator tests**

`TestCreateItemIsTenantScoped`, `TestDeleteItemIsIdempotent` (a second delete
affects zero rows and returns success — the unwind retries), and
`TestUpsertMesoReplacesRatherThanAccumulates` (mode 16 is an assignment; an
upsert that added would double-debit a re-stage).

- [x] **Step 2: Write the failing recovery-shape test**

`TestOrphanedProviderReadsAcrossTenants` — the reconciler runs with no tenant in
context and must rebuild each row's tenant from its stored quad. This is the
test that fails loudly if `TenantRegion` / `TenantMajor` / `TenantMinor` are
omitted from the entity.

- [x] **Step 3: Write the entity**

Copy the column discipline of `services/atlas-mts/atlas.com/mts/holding/entity.go`:
surrogate UUID PK, `(tenant_id, id)` unique, **explicit name-keyed stat columns,
no JSON blob** (COPY/restore column-order safety). Add `room_id`, `owner_id`,
`trade_slot`, `source_inventory_type`, `source_slot`, and the tenant quad from
`settlement/entity.go`. Indexes `(tenant_id, room_id)` and `(tenant_id, owner_id)`.
`trade_escrow_mesos` is `(id, tenant quad, room_id, owner_id, amount, created_at)`
with `(tenant_id, room_id, owner_id)` unique.

- [x] **Step 4: Write the model, builder, provider, administrator and processor**

Immutable model, private fields + getters + Builder (no `*_testhelpers.go`).
Every administrator write takes the caller's `*message.Buffer` so the row and
the status event land in one outbox batch — the discipline `settlement` already
follows.

- [x] **Step 5: Wire the custody consumer and the migration**

The consumer handles `ACCEPT_TO_TRADE` (write the row, emit the saga step's
completion event) and `RELEASE_FROM_TRADE` (soft-delete, emit completion).
Register `escrow.Migration` in `main.go` alongside the existing ones.

- [x] **Step 6: Verify and commit** — `feat(task-205): add the trade escrow store and custody consumer`

`go test -race ./...`, `go vet ./...`, `tools/lint.sh --check`, and
`docker buildx bake atlas-trades atlas-saga-orchestrator` (both `go.mod`s moved).

---

### Task 27: Rewrite staging onto escrow (supersedes Task 18)

**Files:**
- Modify: `services/atlas-trades/atlas.com/trades/trade/processor.go`
- Delete: `services/atlas-trades/atlas.com/trades/compartment/processor.go`, `compartment/producer.go`
- Modify: `services/atlas-trades/atlas.com/trades/kafka/message/trade/kafka.go` and the atlas-channel mirror (add `ITEM_REFUSED`)
- Test: `trade/processor_staging_test.go` (rewrite), `trade/processor_meso_delta_test.go` (new)

**Interfaces:**
- Consumes: Task 25's `transfer_to_trade`, Task 26's `escrow.Processor`, Task 13's config.
- Produces (consumed by Tasks 28, 29, 30): `ITEM_STAGED` unchanged in shape;
  new `ITEM_REFUSED` status event symmetric with `MESO_REFUSED`.

Design §5A.4, §5A.5, §5A.6. Task 18's restriction engine, slot bookkeeping and
meso re-echo survive **unchanged** — only the custody half is rewritten.

**The Kafka contract mirror is guarded.** `ITEM_REFUSED` must be added to *both*
copies in the same commit or `tools/trade-contract-mirror-guard.sh` fails.

- [x] **Step 1: Write the failing meso-delta tests**

`TestAddMesoDebitsOnlyTheDelta`: escrow 1,000, client sends 1,500 → exactly one
`award_mesos(−500)`, escrow row becomes 1,500. `TestAddMesoRefundsWhenTheTotalDrops`:
escrow 1,500, client sends 400 → `award_mesos(+1,100)`. `TestAddMesoZeroDeltaEmitsNoAward`.
These pin §5A.5; a naive implementation that debits the absolute total
double-charges on every re-stage.

- [x] **Step 2: Write the failing refusal-unlock tests**

`TestRestrictionRefusalEmitsItemRefused` and `TestFreezeRuleRefusalEmitsItemRefused` —
every refusal path emits the event that Task 30 turns into the client's unlock.
A refusal that emits nothing is the wedge this whole slice exists to fix, so
these are the load-bearing tests of the slice.

- [x] **Step 3: Write the failing stage-timeout test**

`TestStageNotResolvedWithinTimeoutRefuses` — a `transfer_to_trade` whose
completion never arrives within `stageTimeoutSeconds` drops the stage and emits
`ITEM_REFUSED` (design §5A.11).

- [x] **Step 4: Replace the reservation call with the escrow saga**

`PutItem` keeps its restriction and slot checks, then submits `transfer_to_trade`
instead of `RequestReserve`. The room slot is marked staged when the custody
consumer confirms the escrow row, **not** at submission — the mode-15 broadcast
must follow the escrow write (§5A.4 step 4), or a failed saga leaves the
counterparty's dialog showing an item that was never escrowed.

- [x] **Step 5: Delete the `compartment` package and its wiring**

Nothing reserves any more. Removing the package is the check that nothing else
still depends on it.

- [x] **Step 6: Verify and commit** — `feat(task-205): escrow staged items and meso instead of reserving`

---

### Task 28: Teardown unwind and the stuck-escrow retry ticker (supersedes Task 20)

**Files:**
- Modify: `services/atlas-trades/atlas.com/trades/trade/processor.go` (teardown), `kafka/consumer/character/consumer.go`, `kafka/consumer/session/consumer.go`, `main.go`
- Create: `services/atlas-trades/atlas.com/trades/escrow/retry.go`
- Test: `trade/processor_teardown_escrow_test.go`, `escrow/retry_test.go`

**Interfaces:**
- Consumes: Task 26's `escrow.Processor.ByRoomProvider` / `.OrphanedProvider`.
- Produces: unwind on every §3.3 trigger; the retry ticker entry point.

Design §5A.8.

- [x] **Step 1: Write the failing unwind tests**

One per §3.3 trigger: each returns every escrow item to its owner and refunds
every escrowed meso before the `LEAVE` is emitted.
`TestUnwindIsIdempotentUnderRepeatedTeardown` pins the compare-and-set —
a logout racing a map change must not refund twice.

- [x] **Step 2: Write the failing retry test**

`TestRetryTickerReattemptsAnEscrowRowWhoseReturnFailed` — an
`accept_to_character` that failed on a full inventory leaves the row in escrow;
the ticker re-attempts it and the row clears when the return succeeds. This is
the failure mode §5A.8 accepts and must not silently leak.

- [x] **Step 3: Implement the unwind and the ticker**

The ticker replaces the reservation-refresh ticker at the same cadence; it is
registered in `main.go` via `routine.Go` (`tools/goroutine-guard.sh`).

- [x] **Step 4: Verify and commit** — `feat(task-205): unwind trade escrow on teardown and retry stuck returns`

---

### Task 29: Settlement over escrow (amends Tasks 15 and 19)

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go` (`expandTradeSettlement`)
- Modify: `libs/atlas-saga/payloads.go` (`TradeSettlementPayload` carries escrow ids, not source slots)
- Modify: `services/atlas-trades/atlas.com/trades/trade/settlement.go` (pre-check 3)
- Test: `saga/processor_trade_settlement_test.go` (rewrite), `trade/settlement_test.go`

**Interfaces:**
- Consumes: Task 26's escrow rows.
- Produces: the amended expansion.

Design §5A.7.

- [x] **Step 1: Write the failing expansion test**

`TestExpandTradeSettlementReleasesFromEscrowBeforeAccepting` asserts the exact
step order of §5A.7 and — critically — that **no negative `award_mesos` step is
produced**. The debit already happened at stage time; a settlement that debits
again charges the giver twice. Keep the existing coverage of the
all-releases-before-all-accepts rule; it is unchanged.

- [x] **Step 2: Write the failing tax test**

`TestSettlementCreditsTheTaxedAmountOnly` — the receiver is credited
`m − floor(m × rate(m))` and nobody is credited the difference (§6.5 unchanged).

- [x] **Step 3: Amend the payload and the expander**

`TradeSettlementSide` carries escrow ids in place of `(inventoryType, slot)`
source references. The expander no longer performs a compartment lookup for the
giver's items — the escrow row is the snapshot.

- [x] **Step 4: Amend pre-check 3**

"every reservation is still live" becomes "every escrow row is still present and
matches". Delete the reservation-liveness read; it has no source any more.

- [x] **Step 5: Verify and commit** — `feat(task-205): settle trades out of escrow`

---

### Task 30: The refusal unlock in atlas-channel

**Files:**
- Move: `services/atlas-channel/atlas.com/channel/socket/handler/enable_actions.go` → `services/atlas-channel/atlas.com/channel/session/enable_actions.go`, exported as `session.EnableActions`. Package `session` already imports `socket/writer` (that is where `session.Announce` lives), so there is no cycle; update the existing call sites in `socket/handler/` (`pet_item_use.go`, `character_skill_use.go`, `character_cash_item_use.go`, `character_cash_item_use_point_reset.go`) and `teleportrock/use.go`'s `enableActionsFunc` seam.
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/trade/consumer.go`
- Test: `kafka/consumer/trade/consumer_test.go`

**Interfaces:**
- Consumes: Task 27's `ITEM_REFUSED`, the existing `MESO_REFUSED`.
- Produces: the bare unlock packet.

Design §5A.6. atlas-trades still writes no packet; this emission belongs to
atlas-channel (§2.2).

- [x] **Step 1: Write the failing consumer tests**

`TestItemRefusedEmitsTheBareUnlock` and `TestMesoRefusedEmitsTheBareUnlock`
assert a `STAT_CHANGED` with `exclRequestSent = true` and an **empty** update
list, addressed to the acting character only. `TestItemRefusedEmitsNoInteractionFrame`
pins that the slot stays empty — the unlock must not become visible feedback.

- [x] **Step 2: Implement**

Reuse the existing `enableActions` shape rather than writing a second one. The
move out of package `handler` is the minimum needed to share it; do not
duplicate it.

- [x] **Step 3: Verify and commit** — `fix(task-205): release the client action lock on every refused stage`

---

### Task 31: Config swap, cleanup and the full verification gate

**Files:**
- Modify: `services/atlas-tenants/.../configuration/` (`trade-configs`: drop `reservationTtlSeconds`, add `stageTimeoutSeconds`), `services/atlas-trades/atlas.com/trades/configuration/`
- Modify: `docs/tasks/task-205-player-trade/pr-notes.md`
- Test: the existing configuration tests

**Interfaces:** none new.

Design §5A.10, §5A.11, §5A.12.

- [x] **Step 1: Write the failing config tests**

`stageTimeoutSeconds` defaults to 5 when absent and a partial PATCH cannot wipe
it — the same optional-knob discipline the earlier config fixes established.

- [x] **Step 2: Swap the keys and confirm nothing still reads the old one**

`grep -rn reservationTtlSeconds` must return only the seed-data migration note.

- [x] **Step 3: Run the full gate**

`go test -race ./...` and `go vet ./...` in every changed module;
`go build ./...`; `docker buildx bake atlas-trades atlas-saga-orchestrator atlas-channel atlas-tenants`;
`tools/redis-key-guard.sh`; `tools/goroutine-guard.sh`;
`tools/trade-contract-mirror-guard.sh`; `tools/lint.sh --check`.

- [x] **Step 4: Live re-test of the reported defect**

Stage an item, then stage mesos, then stage a second item, then cancel. All four
must work, the staged item must leave the inventory window, and the cancelled
trade must return everything. This is the acceptance test for the whole slice;
the original defect was invisible to every automated test because it lives
inside the client.

- [x] **Step 5: Code review before PR, then commit** — `docs(task-205): record the escrow amendment in the PR notes`

---

### Slice 7 as built — where the implementation diverged from this plan

Four decisions were made during implementation that this plan did not
anticipate. Each is amended in `design.md` at the section named; they are
collected here so a reviewer diffing plan against branch is not left guessing.

| Planned | Built | Why |
|---|---|---|
| Task 27 step 4: the **custody consumer** marks the slot staged and announces `ITEM_STAGED` | The **saga's terminal status** does, via `StageSucceeded` / `StageFailed` (§5A.4) | Confirmation and refusal become one signal on one topic in one order, so they cannot disagree — and the custody consumer stays at its own boundary, knowing nothing about rooms or dialog slots |
| Task 27 step 3: a `stageTimeoutSeconds` config key bounds an unresolved stage | No key, and no local timer | The orchestrator already bounds its own saga and emits `SAGA_FAILED`, which this branch turns into `ITEM_REFUSED`. A second timeout would be a second unmaintained number racing the first |
| Task 28: a **stuck-escrow retry ticker** sweeps orphaned rows | No ticker; the orphan is returned inline by `returnOrphanedStage` / `refundOrphanedStake` (§5A.8) | The row a ticker was meant to find announces itself — its own saga is completing. What remains is the return that fails on a full inventory, which startup recovery already sweeps. Net: one *fewer* background loop than the service started with |
| §5A.5: the escrow meso row is upserted alongside the `award_mesos` | The row is **armed pending first and committed on completion**, via a compare-and-set on the stake id (§5A.5) | The naive ordering loses money: a teardown between the debit and the record destroys the only record of meso the player has already been charged. The compare-and-set also makes a superseded stake inert, which matters because a player can retype the meso box faster than a saga round trip |

One defect was found and fixed while wiring this up, and is worth a reviewer's
attention because it is invisible in normal operation: `escrowStore` reads must
be rebound onto `emit`'s transaction (`ProcessorImpl.withTx`). A reader left on
the root handle takes a second pooled connection to answer a question asked from
inside a transaction — a deterministic deadlock at pool size 1, and a latent one
in production whenever the pool is exhausted. It also read outside the
transaction, so a command could miss its own earlier write.

---

## Deferred and explicitly out of scope

Recorded here so a reviewer can see these were decided, not forgotten:

| Item | Why it is not in this plan |
|---|---|
| `template_gms_12_1.json` | Outside the PRD's version set; has no interaction keys at all (design §11.4). |
| gms_92's non-trade interaction keys | Belong to families this task cannot verify; adding them blind is the "new opcodes missing from live config" bug in reverse (design §10.4, §11.3). Warrants its own template task. |
| Ledger retention policy | Unbounded growth accepted at current scale; the `settled_at` index makes a retention job a later additive change (design §10.5). |
| `services/atlas-tenants/configurations/mts-configs/` seed directory | A pre-existing runtime failure in another resource, recorded in `context.md` §1.12. trade-configs ships its own seed dir. |
| Cross-family occupancy as an enforced invariant | No shared occupancy store exists across atlas-trades / atlas-mini-games / atlas-merchant; the check is best-effort in atlas-channel by design (§2.1). |
| atlas-ui surface for the ledger | Explicit PRD non-goal. |
| The trade window's report button | Inert in the reference client itself: `UI/Basic.img/BtClaim` is control 1005 and the dialog's only button-notify sink (`sub_7C2A71` @`0x7c2a71`) has no `case 1005`. Reported during testing; no server change can affect it (design §11.6). |
