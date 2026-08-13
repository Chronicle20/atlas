# Cash Shop Surprise Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a player double-click a Cash Shop Surprise box (`5222000`) in their cash locker and receive a randomly-rolled cash item into the same locker, atomically, on every client version whose binary supports it.

**Architecture:** Client sends `CUICashItemGachapon::OnButtonClicked` (8-byte cash SN) → `atlas-channel` decodes and produces `COMMAND_TOPIC_CASH_SHOP` / `OPEN_SURPRISE` → `atlas-cashshop` resolves and validates the box, checks capacity, rolls against `atlas-reward-pools` over REST, resolves the commodity, then in ONE database transaction inserts an idempotency ledger row, decrements/releases the box, and creates the reward asset → emits `EVENT_TOPIC_CASH_SHOP_STATUS` / `SURPRISE_OPENED` or `SURPRISE_FAILED` → `atlas-channel` writes the standalone `CCashShop::OnCashItemGachaponResult` packet. No saga: the roll mutates nothing, so ordering roll → (consume + grant) makes partial application structurally impossible.

**Tech Stack:** Go 1.x microservices (immutable models + Builder, Processor `Interface`+`Impl`, `Method(mb)` / `MethodAndEmit()` pairing, GORM entities, JSON:API via api2go, Kafka via `message.Buffer` + outbox), `libs/atlas-packet` codecs, tenant socket-config JSON templates, `packet-audit` coverage matrix, TypeScript/React 19 + shadcn/ui + Zod for `atlas-ui`.

## Global Constraints

- **Design supersedes PRD where they conflict.** The RE pass in `design.md` §0 invalidated PRD FR-5.1 (no `GachaponOpenDone` reuse), FR-6.1/6.2/6.3 (no error code exists on this wire), and FR-3.5/3.6 + §6 (no `box_template_id` column, no new endpoint). Implement the design.
- **The mode byte is NEVER hard-coded.** It differs on every version: v83 SUCCESS `0xE5`/FAILED `0xE4`, v84 `0xEE`/`0xED`, v87 `0xF4`/`0xF3`, v92 `0xBE`/`0xBD`, v95 `0xC1`/`0xC0`, jms_v185 `0xEB`/`0xEA`. It is resolved at encode time from the tenant template's `options.operations` map via `atlas_packet.WithResolvedCode` (DOM-25).
- **Send opcodes** (serverbound `CUICashItemGachapon::OnButtonClicked`): v79 `0x9F`, v83 `0xA1`, v84 `0xA5`, v87 `0xA9`, v92 `0xB6`, v95 `0xB9`, jms_v185 `0xA7`.
- **Result opcodes** (clientbound `CCashShop::OnCashItemGachaponResult`): v83 `0x14D`, v84 `0x154`, v87 `0x15E`, v92 `0x180`, v95 `0x188`, jms_v185 `0x16D`. v79 has NO result handler.
- **v79 decision (user-confirmed, overrides design §5's recommendation):** route v79 **serverbound only**. The grant is silent — the item lands in the locker and is visible after the player reopens the Cash Shop. The v79 clientbound cell stays `n-a` with the design §5 proof.
- **`n-a` with proof (3 columns): v48, v61, v72.** No `CUICashItemGachapon` in the binary; the standalone opcode's handler is a 1-byte flag store; item `5222000` returns 404 from `atlas-data` on those tenants.
- **Do not modify** `shop_operation_result_gachapon.go`, `shop_operation_result_failed.go`, or any already-✅ codec. This task adds files.
- **No wire regressions.** `packet-audit matrix` regeneration must show promotions only.
- **Multi-tenancy:** every read and write is tenant-scoped via `tenant.MustFromContext(ctx)`.
- **No `// TODO`, no stubs, no 501s in landed commits.**
- **Commit style:** conventional commits, one per task step-group as shown.
- **Repo-relative paths only** in any committed file. Never write a literal home directory.

---

## File Structure

**Create:**

| Path | Responsibility |
|---|---|
| `libs/atlas-packet/cash/serverbound/item_gachapon_button.go` | `CashItemGachaponButton` codec (Encode + Decode), 8-byte cash SN |
| `libs/atlas-packet/cash/serverbound/item_gachapon_button_test.go` | Round-trip + per-version byte fixtures |
| `libs/atlas-packet/cash/clientbound/item_gachapon_result.go` | `CashItemGachaponSuccess` / `CashItemGachaponFailed` codecs + body providers |
| `libs/atlas-packet/cash/clientbound/item_gachapon_result_test.go` | Per-version encode fixtures driven by an `operations` map |
| `services/atlas-cashshop/atlas.com/cashshop/surprise/processor.go` | `OpenSurprise` orchestration: resolve → capacity → roll → commodity → one tx |
| `services/atlas-cashshop/atlas.com/cashshop/surprise/processor_test.go` | Capacity, ownership, atomicity, idempotency tests |
| `services/atlas-cashshop/atlas.com/cashshop/surprise/capacity.go` | `HasRoomForSwap` pure helper |
| `services/atlas-cashshop/atlas.com/cashshop/surprise/capacity_test.go` | Both capacity branches |
| `services/atlas-cashshop/atlas.com/cashshop/surprise/opening/entity.go` | `cash_surprise_openings` idempotency ledger entity + migration |
| `services/atlas-cashshop/atlas.com/cashshop/surprise/opening/administrator.go` | `insert` — the transaction's first statement |
| `services/atlas-cashshop/atlas.com/cashshop/rewardpool/requests.go` | REST client to atlas-reward-pools `POST /gachapons/{id}/rewards/select` |
| `services/atlas-cashshop/atlas.com/cashshop/rewardpool/rest.go` | `RewardRestModel` |
| `services/atlas-cashshop/atlas.com/cashshop/rewardpool/processor.go` | `SelectReward(boxTemplateId)` + `ErrPoolMissing` / `ErrPoolEmpty` mapping |
| `services/atlas-channel/atlas.com/channel/socket/handler/cash_item_gachapon.go` | `CashItemGachaponHandleFunc` |
| `services/atlas-ui/src/lib/schemas/…` (edits only) | — |

**Modify:** listed per task below.

---

### Task 1: Serverbound `CashItemGachaponButton` codec

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/item_gachapon_button.go`
- Test: `libs/atlas-packet/cash/serverbound/item_gachapon_button_test.go`

**Interfaces:**
- Consumes: nothing (first task).
- Produces: `const CashItemGachaponHandle = "CashItemGachaponHandle"`; `type CashItemGachaponButton struct{ cashId int64 }`; `func NewCashItemGachaponButton(cashId int64) CashItemGachaponButton`; method `CashId() int64`; `Operation() string`; `String() string`; `Encode(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`; `(*CashItemGachaponButton) Decode(logrus.FieldLogger, context.Context) func(*request.Reader, map[string]interface{})`.

**Background:** The client body is identical on every version that has it (`design.md` §1.2):
`COutPacket(<send opcode>)` then `EncodeBuffer(&m_liItemSN, 8)`. `EncodeBuffer` of a `LARGE_INTEGER` is byte-identical to a little-endian int64, so there is no version gate and no `WriteByteArray` special case. The codec pattern to copy is `libs/atlas-packet/cash/serverbound/check_wallet.go`.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/cash/serverbound/item_gachapon_button_test.go`:

```go
package serverbound

import (
	"context"
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// The client sends only EncodeBuffer(&m_liItemSN, 8) — a little-endian
// int64 cash serial. Identical on v79/v83/v84/v87/v92/v95/jms_v185
// (design.md §1.2), so one fixture covers every version.
// packet-audit:verify cash/serverbound/CashItemGachaponButton
func TestCashItemGachaponButtonDecode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// 1234567890 as a little-endian int64.
	raw, err := hex.DecodeString("d202964900000000")
	if err != nil {
		t.Fatalf("bad fixture: %v", err)
	}
	r := request.NewReader(&raw)
	m := CashItemGachaponButton{}
	m.Decode(l, context.Background())(r, map[string]interface{}{})
	if m.CashId() != 1234567890 {
		t.Fatalf("cashId = %d, want 1234567890", m.CashId())
	}
}

func TestCashItemGachaponButtonRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	out := NewCashItemGachaponButton(1234567890).Encode(l, context.Background())(map[string]interface{}{})
	if got := hex.EncodeToString(out); got != "d202964900000000" {
		t.Fatalf("encoded = %s", got)
	}
	r := request.NewReader(&out)
	var back CashItemGachaponButton
	back.Decode(l, context.Background())(r, map[string]interface{}{})
	if back.CashId() != 1234567890 {
		t.Fatalf("round-trip cashId = %d", back.CashId())
	}
}

func TestCashItemGachaponButtonOperation(t *testing.T) {
	if NewCashItemGachaponButton(1).Operation() != CashItemGachaponHandle {
		t.Fatal("Operation must return CashItemGachaponHandle")
	}
}
```

Before running, sanity-check the fixture yourself: `1234567890` = `0x499602D2`, little-endian over 8 bytes = `d2 02 96 49 00 00 00 00`. If `response.Writer.WriteInt64` turns out to write big-endian, the fixture — not the codec — is what changes.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-packet && go test ./cash/serverbound/ -run CashItemGachapon -v
```

Expected: FAIL — `undefined: CashItemGachaponButton`.

- [ ] **Step 3: Write the implementation**

Create `libs/atlas-packet/cash/serverbound/item_gachapon_button.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CashItemGachaponHandle = "CashItemGachaponHandle"

// CashItemGachaponButton - the Cash Shop Surprise "Open" button.
// CUICashItemGachapon::OnButtonClicked(nId == 2000) emits
// COutPacket(<send opcode>) + EncodeBuffer(&m_liItemSN, 8) and nothing else.
// EncodeBuffer of a LARGE_INTEGER is byte-identical to a little-endian
// int64, so the body needs no version gate: v79 0x9F, v83 0xA1, v84 0xA5,
// v87 0xA9, v92 0xB6, v95 0xB9, jms_v185 0xA7 all carry the same 8 bytes.
// The client self-gates re-clicks with `if (m_nState < 1)`; only v79 also
// calls CWvsContext::SetExclRequestSent, so on every version in scope the
// send does NOT arm the excl-request gate and no EnableActions is owed.
// packet-audit:fname CUICashItemGachapon::OnButtonClicked
type CashItemGachaponButton struct {
	cashId int64
}

func NewCashItemGachaponButton(cashId int64) CashItemGachaponButton {
	return CashItemGachaponButton{cashId: cashId}
}

func (m CashItemGachaponButton) CashId() int64 { return m.cashId }

func (m CashItemGachaponButton) Operation() string { return CashItemGachaponHandle }

func (m CashItemGachaponButton) String() string {
	return fmt.Sprintf("cashId [%d]", m.cashId)
}

func (m CashItemGachaponButton) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt64(m.cashId)
		return w.Bytes()
	}
}

func (m *CashItemGachaponButton) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.cashId = r.ReadInt64()
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd libs/atlas-packet && go test ./cash/serverbound/ -run CashItemGachapon -v
```

Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/item_gachapon_button.go libs/atlas-packet/cash/serverbound/item_gachapon_button_test.go
git commit -m "feat(task-207): add CashItemGachaponButton serverbound codec"
```

---

### Task 2: Clientbound `CashItemGachaponResult` codecs

**Files:**
- Create: `libs/atlas-packet/cash/clientbound/item_gachapon_result.go`
- Test: `libs/atlas-packet/cash/clientbound/item_gachapon_result_test.go`

**Interfaces:**
- Consumes: `CashInventoryItem` and its `EncodeBytes(l)` / `decodeCashInventoryItemSkipPadding(r)` from `libs/atlas-packet/cash/clientbound/shop_inventory.go` (unchanged).
- Produces:
  - `const CashItemGachaponResultWriter = "CashItemGachaponResult"`
  - `const CashItemGachaponModeSuccess = "SUCCESS"`, `const CashItemGachaponModeFailed = "FAILED"`
  - `func NewCashItemGachaponSuccess(mode byte, sn int64, remain int32, newItem CashInventoryItem, itemId int32, count byte, jackpot byte) CashItemGachaponSuccess`
  - `func NewCashItemGachaponFailed(mode byte) CashItemGachaponFailed`
  - `func CashItemGachaponSuccessBody(sn int64, remain int32, newItem CashInventoryItem, itemId int32, count byte, jackpot byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`
  - `func CashItemGachaponFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`

**Background:** Read order from `design.md` §1.3, identical on every version that has the handler:

```
mode = Decode1()
if mode == <SUCCESS>:
    DecodeBuffer(liSN, 8)      // int64, SN of the consumed box
    remain = Decode4()         // int32, box's new quantity; 0 removes the locker row
    DecodeBuffer(newItem, 0x37) // 55-byte GW_CashItemInfo == CashInventoryItem.EncodeBytes
    itemId  = Decode4()        // int32, rewarded template id (read by the UI object)
    count   = Decode1()
    jackpot = Decode1()        // selects CashGachaponJackpot vs CashGachaponNormal sfx
elif mode == <FAILED>:
    (no body read)
```

The trailing `itemId`/`count`/`jackpot` are read by `CUICashItemGachapon`, not `CCashShop`. On GMS they are simply left in the buffer when no dialog is open — harmless. **The server always writes them.** The blob is unconditional here — unlike `GachaponOpenDone`, there is no `isCashItem` gate byte.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/cash/clientbound/item_gachapon_result_test.go`:

```go
package clientbound

import (
	"context"
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Per-version operations tables, IDA-derived per design.md §1.1
// (CCashShop::OnCashItemGachaponResult dispatch). Template JSON numbers
// decode as float64, so the fixtures use float64 exactly as the runtime does.
func gachaponOps(success byte, failed byte) map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			CashItemGachaponModeSuccess: float64(success),
			CashItemGachaponModeFailed:  float64(failed),
		},
	}
}

func gachaponOpsByVersion() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"gms_v83":  gachaponOps(0xE5, 0xE4),
		"gms_v84":  gachaponOps(0xEE, 0xED),
		"gms_v87":  gachaponOps(0xF4, 0xF3),
		"gms_v92":  gachaponOps(0xBE, 0xBD),
		"gms_v95":  gachaponOps(0xC1, 0xC0),
		"jms_v185": gachaponOps(0xEB, 0xEA),
	}
}

func sampleGachaponItem() CashInventoryItem {
	return CashInventoryItem{
		CashId:      1234567890,
		AccountId:   10,
		CharacterId: 20,
		TemplateId:  5222000,
		CommodityId: 40000,
		Quantity:    1,
		GiftFrom:    "",
		Expiration:  0,
	}
}

// The 55-byte GW_CashItemInfo blob is unconditional on this packet — unlike
// GachaponOpenDone there is no isCashItem gate byte (design.md §1.3).
// packet-audit:verify cash/clientbound/CashItemGachaponResult
func TestCashItemGachaponSuccessEncodePerVersion(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for version, opts := range gachaponOpsByVersion() {
		out := CashItemGachaponSuccessBody(1234567890, 2, sampleGachaponItem(), 5222001, 1, 0)(l, context.Background())(opts)
		wantMode := opts["operations"].(map[string]interface{})[CashItemGachaponModeSuccess].(float64)
		if out[0] != byte(wantMode) {
			t.Fatalf("%s: mode byte = %#x, want %#x", version, out[0], byte(wantMode))
		}
		// 1 mode + 8 sn + 4 remain + 55 blob + 4 itemId + 1 count + 1 jackpot
		if len(out) != 74 {
			t.Fatalf("%s: length = %d, want 74 (hex %s)", version, len(out), hex.EncodeToString(out))
		}
	}
}

func TestCashItemGachaponFailedEncodePerVersion(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for version, opts := range gachaponOpsByVersion() {
		out := CashItemGachaponFailedBody()(l, context.Background())(opts)
		wantMode := opts["operations"].(map[string]interface{})[CashItemGachaponModeFailed].(float64)
		if len(out) != 1 {
			t.Fatalf("%s: FAILED arm must be mode-only, got %d bytes", version, len(out))
		}
		if out[0] != byte(wantMode) {
			t.Fatalf("%s: mode byte = %#x, want %#x", version, out[0], byte(wantMode))
		}
	}
}

// A missing operations table must NOT silently encode a plausible byte —
// ResolveCode returns the loud 99 sentinel. This is the DOM-25 guard.
func TestCashItemGachaponModeIsNotHardCoded(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	out := CashItemGachaponFailedBody()(l, context.Background())(map[string]interface{}{})
	if out[0] != 99 {
		t.Fatalf("unconfigured mode = %d, want the 99 sentinel — the byte is hard-coded", out[0])
	}
}

func TestCashItemGachaponSuccessRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	original := NewCashItemGachaponSuccess(0xE5, 1234567890, 2, sampleGachaponItem(), 5222001, 3, 1)
	out := original.Encode(l, context.Background())(map[string]interface{}{})
	r := newReaderForTest(out)
	var back CashItemGachaponSuccess
	back.Decode(l, context.Background())(r, map[string]interface{}{})
	if back.Mode() != 0xE5 || back.SN() != 1234567890 || back.Remain() != 2 {
		t.Fatalf("header round-trip failed: %+v", back)
	}
	if back.ItemId() != 5222001 || back.Count() != 3 || back.Jackpot() != 1 {
		t.Fatalf("trailing UI fields round-trip failed: %+v", back)
	}
	if back.NewItem().TemplateId != 5222000 {
		t.Fatalf("blob round-trip failed: %+v", back.NewItem())
	}
}
```

`newReaderForTest` may not exist — check the package first:

```bash
grep -rn "request.NewReader" libs/atlas-packet/cash/clientbound/*_test.go | head -3
```

Use whatever helper the sibling tests already use (likely `request.NewReader(&out)` directly, with `"github.com/Chronicle20/atlas/libs/atlas-socket/request"` imported). Replace `newReaderForTest(out)` accordingly rather than adding a new helper.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-packet && go test ./cash/clientbound/ -run CashItemGachapon -v
```

Expected: FAIL — `undefined: CashItemGachaponSuccessBody`.

- [ ] **Step 3: Write the implementation**

Create `libs/atlas-packet/cash/clientbound/item_gachapon_result.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// Cash Shop Surprise result — the STANDALONE opcode
// CASHSHOP_CASH_ITEM_GACHAPON_RESULT, not a CASHSHOP_OPERATION arm. The
// CASHSHOP_OPERATION GACHAPON_OPEN_* arms belong to CUICashGachapon (the
// Cash Gachapon UI), a different feature — see design.md §1.4. This packet's
// handler is CCashShop::OnCashItemGachaponResult; the trailing itemId/count/
// jackpot fields are read by CUICashItemGachapon, not CCashShop, and the
// server always writes them.
//
// Opcodes: v83 0x14D, v84 0x154, v87 0x15E, v92 0x180, v95 0x188,
// jms_v185 0x16D. v79 has NO result handler (n-a) and v48/v61/v72 have no
// CUICashItemGachapon at all (n-a).
const CashItemGachaponResultWriter = "CashItemGachaponResult"

const (
	CashItemGachaponModeSuccess = "SUCCESS"
	CashItemGachaponModeFailed  = "FAILED"
)

// CashItemGachaponSuccess - the SUCCESS arm: mode + sn:DecodeBuffer(8)
// (int64, SN of the consumed box, matched against m_aCashItemInfo[i].liSN) +
// remain:Decode4 (int32, the box's new quantity; 0 removes the locker row) +
// newItem:DecodeBuffer(0x37=55) (GW_CashItemInfo, UNCONDITIONAL — there is no
// isCashItem gate on this packet) + itemId:Decode4 (int32, rewarded template
// id, UI icon + chat log) + count:Decode1 + jackpot:Decode1 (selects the
// CashGachaponJackpot vs CashGachaponNormal sfx).
// packet-audit:fname CCashShop::OnCashItemGachaponResult#SUCCESS
type CashItemGachaponSuccess struct {
	mode    byte
	sn      int64
	remain  int32
	newItem CashInventoryItem
	itemId  int32
	count   byte
	jackpot byte
}

func NewCashItemGachaponSuccess(mode byte, sn int64, remain int32, newItem CashInventoryItem, itemId int32, count byte, jackpot byte) CashItemGachaponSuccess {
	return CashItemGachaponSuccess{mode: mode, sn: sn, remain: remain, newItem: newItem, itemId: itemId, count: count, jackpot: jackpot}
}

func (m CashItemGachaponSuccess) Mode() byte                 { return m.mode }
func (m CashItemGachaponSuccess) SN() int64                  { return m.sn }
func (m CashItemGachaponSuccess) Remain() int32              { return m.remain }
func (m CashItemGachaponSuccess) NewItem() CashInventoryItem { return m.newItem }
func (m CashItemGachaponSuccess) ItemId() int32              { return m.itemId }
func (m CashItemGachaponSuccess) Count() byte                { return m.count }
func (m CashItemGachaponSuccess) Jackpot() byte              { return m.jackpot }
func (m CashItemGachaponSuccess) Operation() string          { return CashItemGachaponResultWriter }

func (m CashItemGachaponSuccess) String() string {
	return fmt.Sprintf("cash-item-gachapon success mode [%d] sn [%d] remain [%d] itemId [%d] count [%d] jackpot [%d]", m.mode, m.sn, m.remain, m.itemId, m.count, m.jackpot)
}

func (m CashItemGachaponSuccess) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt64(m.sn)
		w.WriteInt32(m.remain)
		w.WriteByteArray(m.newItem.EncodeBytes(l))
		w.WriteInt32(m.itemId)
		w.WriteByte(m.count)
		w.WriteByte(m.jackpot)
		return w.Bytes()
	}
}

func (m *CashItemGachaponSuccess) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.sn = r.ReadInt64()
		m.remain = r.ReadInt32()
		m.newItem = decodeCashInventoryItemSkipPadding(r)
		m.itemId = r.ReadInt32()
		m.count = r.ReadByte()
		m.jackpot = r.ReadByte()
	}
}

// CashItemGachaponFailed - the FAILED arm. The client reads NOTHING after
// the mode byte: it calls StringPool::GetString(<fixed id>) and
// CUtilDlg::Notice. There is no error-code field on this wire (design.md
// §2.3), so the distinct failure reasons are logged server-side and carried
// on the status event, never sent to the client. The client also does not
// re-enable the dialog's Open button on failure — that is native behaviour,
// and we replicate it rather than inventing a recovery packet.
// packet-audit:fname CCashShop::OnCashItemGachaponResult#FAILED
type CashItemGachaponFailed struct {
	mode byte
}

func NewCashItemGachaponFailed(mode byte) CashItemGachaponFailed {
	return CashItemGachaponFailed{mode: mode}
}

func (m CashItemGachaponFailed) Mode() byte        { return m.mode }
func (m CashItemGachaponFailed) Operation() string { return CashItemGachaponResultWriter }

func (m CashItemGachaponFailed) String() string {
	return fmt.Sprintf("cash-item-gachapon failed mode [%d]", m.mode)
}

func (m CashItemGachaponFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *CashItemGachaponFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// CashItemGachaponSuccessBody resolves the SUCCESS mode byte from the tenant
// operations table at encode time. The byte differs on EVERY version
// (v83 0xE5, v84 0xEE, v87 0xF4, v92 0xBE, v95 0xC1, jms 0xEB), which is
// exactly the DOM-25 failure mode the rule exists for — never hard-code it.
func CashItemGachaponSuccessBody(sn int64, remain int32, newItem CashInventoryItem, itemId int32, count byte, jackpot byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashItemGachaponModeSuccess, func(mode byte) packet.Encoder {
		return NewCashItemGachaponSuccess(mode, sn, remain, newItem, itemId, count, jackpot)
	})
}

// CashItemGachaponFailedBody resolves the FAILED mode byte the same way
// (v83 0xE4, v84 0xED, v87 0xF3, v92 0xBD, v95 0xC0, jms 0xEA).
func CashItemGachaponFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashItemGachaponModeFailed, func(mode byte) packet.Encoder {
		return NewCashItemGachaponFailed(mode)
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd libs/atlas-packet && go test ./cash/clientbound/ -run CashItemGachapon -v
```

Expected: PASS (4 tests). If `WriteInt32` does not exist on the writer, check `libs/atlas-socket/response` for the actual name (`shop_operation_result_gachapon.go:67` uses `w.WriteInt32`, so it does).

- [ ] **Step 5: Confirm no already-verified codec changed**

```bash
cd libs/atlas-packet && go test ./cash/... 2>&1 | tail -20
git diff --stat libs/atlas-packet/
```

Expected: all cash tests PASS; `git diff --stat` lists ONLY the two new files from Tasks 1–2. If any pre-existing file appears, revert that change — the design forbids editing shared codecs.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/cash/clientbound/item_gachapon_result.go libs/atlas-packet/cash/clientbound/item_gachapon_result_test.go
git commit -m "feat(task-207): add CashItemGachaponResult clientbound codecs"
```

---

### Task 3: Registry entries and the jms `0xA7` attribution correction

**Files:**
- Modify: `docs/packets/registry/gms_v83.yaml:2856-2860` (provenance)
- Modify: `docs/packets/registry/gms_v79.yaml` (new serverbound entry)
- Modify: `docs/packets/registry/jms_v185.yaml` (new serverbound entry)
- Modify: `docs/tasks/task-183-cashshop-result-family/arm-catalog.md:251`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: registry entries the coverage matrix reads in Task 18. No Go symbols.

**Background:** `design.md` §1.1 verified every opcode against its IDB. Two registry entries are missing entirely (v79 `0x9F` = 159, jms_v185 `0xA7` = 167) and the v83 entry is marked `provenance: csv-import` — meaning never IDA-verified — which is now false.

- [ ] **Step 1: Upgrade the v83 provenance**

The current entry at `docs/packets/registry/gms_v83.yaml:2856-2860` reads:

```yaml
- op: CASH_ITEM_GACHAPON_BUTTON
  direction: serverbound
  opcode: 161
  fname: CUICashItemGachapon::OnButtonClicked
  provenance: csv-import
```

Before editing, confirm what `provenance` value other IDA-derived entries use:

```bash
grep -h 'provenance:' docs/packets/registry/gms_v83.yaml | sort | uniq -c
```

Set `provenance:` to whichever value that survey shows is used for IDA-derived entries (do **not** invent a new value). Add a comment line above the entry recording the derivation address: `# IDA-derived: CUICashItemGachapon::OnButtonClicked send site 0x99a9a7 (task-207)` — but only if the file already carries comments; check with `grep -n '^\s*#' docs/packets/registry/gms_v83.yaml | head`. If it carries none, skip the comment (the evidence lives in `design.md` §1.1).

- [ ] **Step 2: Add the v79 entry**

Insert into `docs/packets/registry/gms_v79.yaml` at its **sorted opcode position** for serverbound entries (opcode 159 = `0x9F`):

```yaml
- op: CASH_ITEM_GACHAPON_BUTTON
  direction: serverbound
  opcode: 159
  fname: CUICashItemGachapon::OnButtonClicked
  provenance: <same value used in Step 1>
```

Find the position with:

```bash
grep -n 'direction: serverbound' -A1 docs/packets/registry/gms_v79.yaml | grep -n 'opcode: 15[5-9]\|opcode: 16[0-2]'
```

- [ ] **Step 3: Add the jms_v185 entry**

Same shape in `docs/packets/registry/jms_v185.yaml`, opcode 167 (`0xA7`), at its sorted serverbound position.

- [ ] **Step 4: Correct the arm-catalog note**

`docs/tasks/task-183-cashshop-result-family/arm-catalog.md:251` attributes jms `COutPacket(0xA7)` — reached by double-clicking item `5222002` in the Cash Locker — to `SendChangeMaplePoint`. Read the surrounding paragraph first:

```bash
sed -n '240,262p' docs/tasks/task-183-cashshop-result-family/arm-catalog.md
```

Amend it in place to record the task-207 finding, preserving the original claim rather than deleting it:

> **Corrected by task-207.** In this jms build the only sender of `0xA7` within the UI code segment (`0xA00000`–`0xB00000`) is `CUICashItemGachapon::OnButtonClicked` (`0xa6e39c`), and `5222002` carries classification `522` (gachapon coupon) — so the double-click path described here is the Cash Shop Surprise open request, not `SendChangeMaplePoint`. Scope caveat: the opcode search was bounded to the UI segment, not the whole image, so this corrects the attribution without proving `SendChangeMaplePoint` never sends `0xA7` from elsewhere.

- [ ] **Step 5: Verify the registry still parses**

```bash
cd tools/packet-audit && go run . matrix --check 2>&1 | tail -30
```

Expected: no YAML parse error and no *new* failure attributable to these three entries. The matrix will still report the target cells as `❌` — that is correct until Task 18. Record any pre-existing failures now so Task 18 can tell new breakage from old:

```bash
cd tools/packet-audit && go run . matrix --check 2>&1 | tail -40 > "${SCRATCH:?set SCRATCH to your session scratchpad dir}/matrix-baseline.txt"
```

- [ ] **Step 6: Commit**

```bash
git add docs/packets/registry/gms_v83.yaml docs/packets/registry/gms_v79.yaml docs/packets/registry/jms_v185.yaml docs/tasks/task-183-cashshop-result-family/arm-catalog.md
git commit -m "docs(task-207): add v79/jms CASH_ITEM_GACHAPON_BUTTON registry entries, upgrade v83 provenance, correct jms 0xA7 attribution"
```

---

### Task 4: `atlas-reward-pools` — add the `cash-surprise` kind

**Files:**
- Modify: `services/atlas-reward-pools/atlas.com/reward-pools/gachapon/builder.go:9-22` (const block), `:77` (validation)
- Test: `services/atlas-reward-pools/atlas.com/reward-pools/gachapon/builder_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `const gachapon.KindCashSurprise = "cash-surprise"`. `gachapon.DefaultKind` stays `KindGachapon`.

**Background:** The closed union today is `KindGachapon = "gachapon"` and `KindIncubator = "incubator"`, validated at `builder.go:77` with `if b.kind != KindGachapon && b.kind != KindIncubator`. `DefaultKind = KindGachapon` must not change — existing rows and every caller that never sets `Kind` depend on it.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-reward-pools/atlas.com/reward-pools/gachapon/builder_test.go`:

```go
func TestBuilderAcceptsCashSurpriseKind(t *testing.T) {
	m, err := NewBuilder(uuid.New(), "5222000").
		SetName("Cash Shop Surprise").
		SetKind(KindCashSurprise).
		Build()
	if err != nil {
		t.Fatalf("cash-surprise kind rejected: %v", err)
	}
	if m.Kind() != KindCashSurprise {
		t.Fatalf("kind = %q, want %q", m.Kind(), KindCashSurprise)
	}
}

func TestBuilderStillRejectsUnknownKind(t *testing.T) {
	_, err := NewBuilder(uuid.New(), "1").SetKind("mystery-box").Build()
	if err == nil {
		t.Fatal("unknown kind must be rejected — the union stays closed")
	}
}

func TestDefaultKindUnchanged(t *testing.T) {
	m, err := NewBuilder(uuid.New(), "9000000").Build()
	if err != nil {
		t.Fatalf("default build failed: %v", err)
	}
	if m.Kind() != KindGachapon {
		t.Fatalf("DefaultKind regressed to %q — existing rows read this value", m.Kind())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-reward-pools && go test ./gachapon/ -run 'CashSurprise|UnknownKind|DefaultKind' -v
```

Expected: FAIL — `undefined: KindCashSurprise`.

- [ ] **Step 3: Widen the union**

In `services/atlas-reward-pools/atlas.com/reward-pools/gachapon/builder.go`, replace the const block's doc comment and body:

```go
// KindGachapon, KindIncubator and KindCashSurprise are the closed union of
// valid Kind values for a reward pool: the classic tiered reward pool, the
// Pigmy Egg incubator pool, and the Cash Shop Surprise box pool (task-207).
// Every comparison against a pool's Kind must reference one of these
// constants rather than a bare string literal.
const (
	KindGachapon     = "gachapon"
	KindIncubator    = "incubator"
	KindCashSurprise = "cash-surprise"
)
```

And replace the validation at `builder.go:77`:

```go
	if !isValidKind(b.kind) {
		return Model{}, fmt.Errorf("gachapon: invalid kind %q", b.kind)
	}
```

Add below `Build()`:

```go
// isValidKind enforces the closed union. Adding a fourth kind means adding
// it here and deciding, in reward/processor.go, whether it rolls flat or
// tiered — see usesFlatWeights.
func isValidKind(kind string) bool {
	return kind == KindGachapon || kind == KindIncubator || kind == KindCashSurprise
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd services/atlas-reward-pools && go test ./gachapon/ -v
```

Expected: PASS, including every pre-existing gachapon test.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-reward-pools/atlas.com/reward-pools/gachapon/builder.go services/atlas-reward-pools/atlas.com/reward-pools/gachapon/builder_test.go
git commit -m "feat(task-207): add cash-surprise pool kind to reward-pools"
```

---

### Task 5: `atlas-reward-pools` — `commodity_id` on pool items

**Files:**
- Modify: `services/atlas-reward-pools/atlas.com/reward-pools/item/entity.go` (new column)
- Modify: `services/atlas-reward-pools/atlas.com/reward-pools/item/model.go` (field + getter)
- Modify: `services/atlas-reward-pools/atlas.com/reward-pools/item/builder.go` (setter + validation)
- Modify: `services/atlas-reward-pools/atlas.com/reward-pools/item/rest.go` (`RestModel`, `JSONModel`, `Transform`)
- Modify: `services/atlas-reward-pools/atlas.com/reward-pools/item/processor.go:30-32,71-76` (`Update` signature)
- Modify: `services/atlas-reward-pools/atlas.com/reward-pools/item/administrator.go` (`UpdateItem`), `provider.go` / `Make` if it maps columns explicitly
- Modify: `services/atlas-reward-pools/atlas.com/reward-pools/item/resource.go` (create/patch wiring)
- Test: `services/atlas-reward-pools/atlas.com/reward-pools/item/builder_test.go`, `item/processor_test.go`

**Interfaces:**
- Consumes: `gachapon.KindCashSurprise` (Task 4).
- Produces:
  - `item.Model.CommodityId() uint32`
  - `(*item.Builder).SetCommodityId(commodityId uint32) *item.Builder`
  - `(*item.Builder).SetKind(kind string) *item.Builder` — the pool kind the entry belongs to, used only for validation; NOT persisted
  - `item.Processor.Update(id uint32, itemId uint32, quantity uint32, tier string, weight uint32, commodityId uint32) error`
  - `item.RestModel.CommodityId uint32` with json tag `commodityId`

**Background:** For a `cash-surprise` entry the reward is identified by **cash shop commodity id (serial number)**, not raw item id (`design.md` §4.1, PRD FR-3.3). The commodity catalog owns `itemId`/`count`/`period`, so rolling a commodity guarantees a self-consistent locker entry. `item_id` stays on the row as an advisory display value.

The precedent for this exact migration is `Weight`, added as `gorm:"not null;default:0"` — AutoMigrate backfills existing rows and no data migration is needed. Note `Build()` currently requires a valid tier (`common|uncommon|rare`); `cash-surprise` entries reuse `"common"` as a placeholder exactly as incubator entries do (`PoolItemDialog.tsx` sends `tier: "common"` for weighted kinds).

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-reward-pools/atlas.com/reward-pools/item/builder_test.go`:

```go
func TestBuilderCashSurpriseRequiresCommodityId(t *testing.T) {
	_, err := NewBuilder(uuid.New(), 1).
		SetGachaponId("5222000").
		SetKind(gachapon.KindCashSurprise).
		SetItemId(5222001).
		SetQuantity(1).
		SetTier("common").
		SetWeight(10).
		Build()
	if !errors.Is(err, ErrCommodityIdRequired) {
		t.Fatalf("err = %v, want ErrCommodityIdRequired — a cash-surprise entry without a commodity cannot be granted", err)
	}
}

func TestBuilderCashSurpriseAcceptsCommodityId(t *testing.T) {
	m, err := NewBuilder(uuid.New(), 1).
		SetGachaponId("5222000").
		SetKind(gachapon.KindCashSurprise).
		SetItemId(5222001).
		SetQuantity(1).
		SetTier("common").
		SetWeight(10).
		SetCommodityId(40000).
		Build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if m.CommodityId() != 40000 {
		t.Fatalf("commodityId = %d, want 40000", m.CommodityId())
	}
}

// Existing kinds must be untouched: a gachapon or incubator entry with no
// commodity id still builds, and reads 0.
func TestBuilderOtherKindsDoNotRequireCommodityId(t *testing.T) {
	for _, kind := range []string{gachapon.KindGachapon, gachapon.KindIncubator, ""} {
		m, err := NewBuilder(uuid.New(), 1).
			SetGachaponId("9000000").
			SetKind(kind).
			SetItemId(2000000).
			SetQuantity(1).
			SetTier("common").
			Build()
		if err != nil {
			t.Fatalf("kind %q: build failed: %v", kind, err)
		}
		if m.CommodityId() != 0 {
			t.Fatalf("kind %q: commodityId = %d, want 0", kind, m.CommodityId())
		}
	}
}
```

Add `"atlas-reward-pools/gachapon"` and `"errors"` to that file's imports. If importing `gachapon` from `item` creates an import cycle, check first:

```bash
cd services/atlas-reward-pools && grep -rn '"atlas-reward-pools/item"' gachapon/
```

If `gachapon` imports `item`, do **not** import `gachapon` from `item`. Instead declare the kind strings the builder compares against as a local unexported constant in `item/builder.go` (`const kindCashSurprise = "cash-surprise"`) with a comment pointing at `gachapon.KindCashSurprise` as the authority, and use the literal `"cash-surprise"` in the tests. Decide this before writing Step 3.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-reward-pools && go test ./item/ -run CashSurprise -v
```

Expected: FAIL — `undefined: ErrCommodityIdRequired`.

- [ ] **Step 3: Implement the column, model, builder, and REST changes**

`item/entity.go` — add below `Weight`:

```go
	// CommodityId is the cash shop commodity (serial number) awarded by a
	// cash-surprise pool entry. The commodity catalog owns the reward's
	// itemId/count/period, so rolling a commodity guarantees a
	// self-consistent locker entry (task-207). `default:0` backfills
	// pre-existing rows when AutoMigrate adds this column; gachapon and
	// incubator entries leave it 0 and keep using ItemId.
	CommodityId uint32 `gorm:"not null;default:0"`
```

`item/model.go` — add the `commodityId uint32` field and:

```go
// CommodityId is the cash shop commodity (serial number) awarded by a
// cash-surprise pool entry; 0 for gachapon and incubator entries, which use
// ItemId instead.
func (m Model) CommodityId() uint32 {
	return m.commodityId
}
```

`item/builder.go` — add `commodityId uint32` and `kind string` fields, the two setters, the sentinel, and the validation inside `Build()` after the tier check:

```go
// SetCommodityId sets the cash shop commodity (serial number) this entry
// awards. Required for cash-surprise entries; other kinds leave it 0.
func (b *Builder) SetCommodityId(commodityId uint32) *Builder {
	b.commodityId = commodityId
	return b
}

// SetKind records the owning pool's kind so Build can apply kind-specific
// validation. It is NOT persisted — the kind lives on the pool row.
func (b *Builder) SetKind(kind string) *Builder {
	b.kind = kind
	return b
}

// ErrCommodityIdRequired is returned when a cash-surprise pool entry omits
// its commodity id. Such an entry can be rolled but never granted, so it is
// rejected at write time rather than failing silently at open time.
var ErrCommodityIdRequired = errors.New("commodityId is required for cash-surprise pool entries")
```

In `Build()`, after `if !isValidTier(b.tier)`:

```go
	if b.kind == kindCashSurprise && b.commodityId == 0 {
		return Model{}, ErrCommodityIdRequired
	}
```

(or `gachapon.KindCashSurprise` if Step 1 established there is no cycle), and add `commodityId: b.commodityId` to the returned `Model`.

`item/rest.go` — add `CommodityId uint32 \`json:"commodityId"\`` to both `RestModel` and `JSONModel`, and `CommodityId: m.CommodityId(),` to `Transform`.

`item/processor.go` — widen `Update`:

```go
	// Update rewrites an item's itemId/quantity/tier/weight/commodityId in
	// place. commodityId is 0 for gachapon and incubator entries.
	Update(id uint32, itemId uint32, quantity uint32, tier string, weight uint32, commodityId uint32) error
```

and thread `commodityId` through to `UpdateItem` in `administrator.go`. Update `Make` (wherever entity→model mapping lives — `grep -rn "func Make" services/atlas-reward-pools/atlas.com/reward-pools/item/`) to carry `commodityId`.

`item/resource.go` — thread `CommodityId` from the request body into the builder on create and into `Update` on patch, and pass the owning pool's kind to `SetKind`. The resource already loads the pool for other checks or can fetch it via `gachapon.NewProcessor(...).GetById(gachaponId)`; read the file and follow whichever pattern is there.

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-reward-pools && go build ./... && go test ./item/ ./gachapon/ -v 2>&1 | tail -30
```

Expected: PASS. Fix every `Update(...)` call site the widened signature breaks (`go build ./...` names them).

- [ ] **Step 5: Run the full module test suite**

```bash
cd services/atlas-reward-pools && go test -race ./... 2>&1 | tail -20
```

Expected: PASS. Pre-existing gachapon/incubator behaviour must be unchanged.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-reward-pools/
git commit -m "feat(task-207): add commodityId to reward-pool items"
```

---

### Task 6: `atlas-reward-pools` — flat-weight selection, `ErrEmptyPool`, and the commodity on the reward

**Files:**
- Modify: `services/atlas-reward-pools/atlas.com/reward-pools/reward/processor.go:43-99` (`SelectReward`), `:101-146` (`GetPrizePool`)
- Modify: `services/atlas-reward-pools/atlas.com/reward-pools/reward/model.go` (add `commodityId`)
- Modify: `services/atlas-reward-pools/atlas.com/reward-pools/reward/builder.go` (`SetCommodityId`)
- Modify: `services/atlas-reward-pools/atlas.com/reward-pools/reward/rest.go` (`RestModel` + `Transform`)
- Modify: `services/atlas-reward-pools/atlas.com/reward-pools/reward/resource.go:29-51` (409 mapping)
- Test: `services/atlas-reward-pools/atlas.com/reward-pools/reward/processor_test.go`, `reward/resource_test.go`

**Interfaces:**
- Consumes: `gachapon.KindCashSurprise` (Task 4); `item.Model.CommodityId()` (Task 5).
- Produces:
  - `var reward.ErrEmptyPool = errors.New("reward pool has no eligible entries")`
  - `reward.Model.CommodityId() uint32`
  - `(*reward.Builder).SetCommodityId(commodityId uint32) *reward.Builder`
  - `reward.RestModel.CommodityId uint32` with json tag `commodityId`
  - `POST /gachapons/{gachaponId}/rewards/select` returns **404** when no pool matches the id and **409** when the pool exists but has no eligible entries.

**Background:** `SelectReward` today branches `if g.Kind() == gachapon.KindIncubator` → whole-machine, flat `item.Weight`, no global merge; `else` → tiered roll + global merge. `cash-surprise` takes the **same branch as incubator** (`design.md` §4.1, resolving PRD Q2 to flat weights): flat weights satisfy FR-3.4, and skipping `getMergedPool` satisfies FR-3.2 — the shared global pool holds regular item ids that would be invalid as cash rewards.

Rewrite the branch condition as a named predicate so the third kind does not accrete a copy of the body. `GetPrizePool` has the identical branch at `:107` and needs the same treatment.

The empty-pool error today is an anonymous `errors.New("no items available in pool for tier: " + tier)` (`:77`), which the resource maps to whatever `server.WriteErrorResponse` defaults to. Replacing it with a sentinel lets the caller distinguish 404 (`POOL_MISSING`) from 409 (`POOL_EMPTY`) — PRD FR-3.7.

Determinism (`design.md` §4.1): `crypto/rand` stays inside `selectItem`; the weighting boundary is tested through the already-extracted pure `selectWeightedIndex(pool, roll)`. **No RNG injection is added.**

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-reward-pools/atlas.com/reward-pools/reward/processor_test.go` (mirror the setup used by the existing `TestSelectReward…` tests in that file — read them first for the fixture/database helpers):

```go
// A cash-surprise pool rolls flat-weighted over the whole pool and NEVER
// merges the shared global pool: the global pool holds regular item ids that
// would be invalid as cash rewards (PRD FR-3.2).
func TestSelectRewardCashSurpriseExcludesGlobalPool(t *testing.T) {
	// setup: one cash-surprise pool "5222000" with a single entry
	// (itemId 5222001, commodityId 40000, weight 10, tier "common"),
	// plus at least one GLOBAL item in tier "common".
	// assert: 100 rolls all return itemId 5222001 and commodityId 40000.
}

// Flat weights, not the three-tier roll: a pool whose entries span tiers
// must still be selectable purely by weight.
func TestSelectRewardCashSurpriseUsesFlatWeights(t *testing.T) {
	// setup: pool "5222000" with entries in tiers common/uncommon/rare,
	// weights 1 / 0 / 0, all with distinct commodityIds.
	// assert: 50 rolls all return the weight-1 entry; Tier() is "".
}

func TestSelectRewardCashSurpriseEmptyPoolReturnsSentinel(t *testing.T) {
	// setup: pool "5222000" of kind cash-surprise with NO items.
	_, err := /* processor */.SelectReward("5222000")
	if !errors.Is(err, ErrEmptyPool) {
		t.Fatalf("err = %v, want ErrEmptyPool", err)
	}
}

// Regression control: the two pre-existing kinds must behave identically
// after the refactor.
func TestSelectRewardGachaponStillMergesGlobalPool(t *testing.T) {
	// setup: a gachapon pool with ZERO machine items but a global item in
	// every tier. assert: SelectReward succeeds and returns the global item.
}

func TestSelectRewardIncubatorStillExcludesGlobalPool(t *testing.T) {
	// setup: an incubator pool with one weighted item plus a global item.
	// assert: every roll returns the machine item.
}
```

Fill each body out completely using the existing tests' fixture helpers (`services/atlas-reward-pools/atlas.com/reward-pools/test/fixtures.go`, `test/database.go`) — do not leave the comment-only skeletons above in the committed file.

Append to `reward/resource_test.go`:

```go
func TestSelectRewardEmptyPoolReturns409(t *testing.T) {
	// setup: cash-surprise pool with no items; POST /gachapons/5222000/rewards/select
	// assert: status 409
}

func TestSelectRewardMissingPoolReturns404(t *testing.T) {
	// POST /gachapons/9999999/rewards/select against an empty database
	// assert: status 404
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-reward-pools && go test ./reward/ -run 'CashSurprise|EmptyPool|MissingPool' -v
```

Expected: FAIL — `undefined: ErrEmptyPool`.

- [ ] **Step 3: Implement the predicate, sentinel, and commodity plumbing**

In `reward/processor.go`, add near the top:

```go
// ErrEmptyPool is returned when a pool exists but has no eligible entries.
// It is distinct from the not-found error a missing pool produces so the
// resource can answer 409 vs 404, and so atlas-cashshop can log POOL_EMPTY
// vs POOL_MISSING (PRD FR-3.7).
var ErrEmptyPool = errors.New("reward pool has no eligible entries")

// usesFlatWeights reports whether a pool kind rolls the whole pool weighted
// by item.Weight — never the tiered selectTier roll, and never the shared
// global pool. Incubator machines and cash-surprise boxes both do; the
// classic gachapon does not. Adding a fourth kind means deciding here.
func usesFlatWeights(kind string) bool {
	return kind == gachapon.KindIncubator || kind == gachapon.KindCashSurprise
}
```

Replace the `if g.Kind() == gachapon.KindIncubator {` condition at `:52` with `if usesFlatWeights(g.Kind()) {`, and update its comment to name both kinds. Inside that branch also carry the commodity id:

```go
		for _, mi := range machineItems {
			pool = append(pool, poolItem{
				ItemId:      mi.ItemId(),
				Quantity:    mi.Quantity(),
				Weight:      mi.Weight(),
				CommodityId: mi.CommodityId(),
			})
		}
```

Add `CommodityId uint32` to the `poolItem` struct (`:16-24`) with a comment noting it is 0 for every kind except `cash-surprise`. `getMergedPool` leaves it 0 for both machine and global items — add `CommodityId: mi.CommodityId(),` to the machine-item append there too so a mis-kinded row is not silently zeroed, and leave the global append at 0 with the existing "global-pool items have no weight concept" comment extended to mention commodity.

Replace `:76-78` with:

```go
	if len(pool) == 0 {
		return Model{}, ErrEmptyPool
	}
```

Add `SetCommodityId(selected.CommodityId).` to the `NewBuilder(...)` chain at `:85-89`, and add `"commodity_id": selected.CommodityId,` to the log fields at `:91-96`.

Apply the same `usesFlatWeights` swap at `GetPrizePool` `:107`, and add `SetCommodityId(mi.CommodityId()).` to that branch's builder chain.

`reward/model.go` — add `commodityId uint32` and:

```go
// CommodityId is the cash shop commodity (serial number) this reward grants.
// Non-zero only for cash-surprise pools; other kinds identify the reward by
// ItemId alone.
func (m Model) CommodityId() uint32 {
	return m.commodityId
}
```

`reward/builder.go` — add the field and `SetCommodityId`, mirroring `SetWeight`.

`reward/rest.go` — add `CommodityId uint32 \`json:"commodityId"\`` and `CommodityId: m.CommodityId(),` in `Transform`.

`reward/resource.go` `handleSelectReward` — map the sentinel before the generic writer:

```go
			result, err := NewProcessor(d.Logger(), d.Context(), d.DB()).SelectReward(gachaponId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Selecting reward for gachapon [%s].", gachaponId)
				if errors.Is(err, ErrEmptyPool) {
					// The pool exists but has nothing to give. 409, not 500:
					// the caller (atlas-cashshop) distinguishes this from a
					// missing pool to log POOL_EMPTY vs POOL_MISSING.
					w.WriteHeader(http.StatusConflict)
					return
				}
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
```

Confirm the missing-pool path already yields 404: `gachapon.Processor.GetById` returns `gorm.ErrRecordNotFound`, and `server.WriteErrorResponse` should map it. Verify with the resource test rather than assuming; if it does not, add an explicit `errors.Is(err, gorm.ErrRecordNotFound)` → `http.StatusNotFound` branch with the same comment style.

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-reward-pools && go test -race ./... 2>&1 | tail -25
```

Expected: PASS, including the two regression-control tests.

- [ ] **Step 5: Update the service docs**

`services/atlas-reward-pools/docs/domain.md` and `docs/rest.md` describe the `kind` union, the `item` entity, and the `POST /gachapons/{gachaponId}/rewards/select` responses. Locate the exact sections first:

```bash
grep -n 'incubator\|kind\|rewards/select\|Weight' services/atlas-reward-pools/docs/domain.md services/atlas-reward-pools/docs/rest.md
```

Add: the third `kind` value; the `commodityId` field on pool items and on the reward resource; the 409 empty-pool response; and a note that a `cash-surprise` pool's **id is the box template id** (`5222000`), exactly as an incubator pool's id is the egg item id.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-reward-pools/
git commit -m "feat(task-207): roll cash-surprise pools flat-weighted with commodity ids and a 409 empty-pool outcome"
```

---

### Task 7: `atlas-cashshop` — tenant-configured Surprise box template ids

**Files:**
- Modify: `services/atlas-cashshop/atlas.com/cashshop/configuration/tenant/cashshop/rest.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/configuration/tenant/cashshop/surprise/rest.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/configuration/registry.go` (new accessor)
- Test: `services/atlas-cashshop/atlas.com/cashshop/configuration/registry_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `func configuration.GetSurpriseBoxTemplateIds(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) []uint32` — returns the configured list, or `[]uint32{5222000}` when unconfigured.

**Background:** PRD FR-2.2 and DOM-25: the Surprise item id is config-resolved, not a code constant. A **list** rather than a scalar so a tenant can designate additional boxes (`design.md` §4.2). `5222000` is the default.

The existing shape to copy is `configuration/tenant/cashshop/commodities` — a sub-package holding a `RestModel` embedded in `cashshop.RestModel`, read through the memoized `configuration.GetTenantConfig`. Note `GetTenantConfig` swallows fetch failures and returns a zero `RestModel`, so the default must come from the accessor, not from the config fetch.

- [ ] **Step 1: Write the failing test**

Create or append to `services/atlas-cashshop/atlas.com/cashshop/configuration/registry_test.go`:

```go
// An unconfigured tenant must still be able to open the stock box: the
// tenant-config fetch failing returns a zero RestModel (see GetTenantConfig),
// so the 5222000 default has to live in the accessor.
func TestGetSurpriseBoxTemplateIdsDefaultsTo5222000(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ids := GetSurpriseBoxTemplateIds(l, context.Background(), uuid.New())
	if len(ids) != 1 || ids[0] != 5222000 {
		t.Fatalf("ids = %v, want [5222000]", ids)
	}
}

func TestGetSurpriseBoxTemplateIdsUsesConfiguredList(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	tenantId := uuid.New()
	// Seed the memoized cache directly, the way the other registry tests do.
	setTenantConfigForTest(tenantId, tenant.RestModel{
		CashShop: cashshop.RestModel{
			Surprise: surprise.RestModel{BoxTemplateIds: []uint32{5222000, 5222002}},
		},
	})
	ids := GetSurpriseBoxTemplateIds(l, context.Background(), tenantId)
	if len(ids) != 2 || ids[0] != 5222000 || ids[1] != 5222002 {
		t.Fatalf("ids = %v, want [5222000 5222002]", ids)
	}
}
```

`setTenantConfigForTest` does not exist. Check whether `registry_test.go` exists at all and how any sibling test seeds the cache:

```bash
ls services/atlas-cashshop/atlas.com/cashshop/configuration/
grep -rn 'tenantConfig\[' services/atlas-cashshop/atlas.com/cashshop/configuration/
```

If there is no existing seam, add one in `registry.go` — an exported-for-test helper is wrong here; instead export a narrow `SetTenantConfig(tenantId uuid.UUID, cfg tenant.RestModel)` used only by tests **or** keep the test in-package (`package configuration`) and write `tenantConfig[tenantId] = cfg` under `mu.Lock()` directly. Prefer the in-package write — no production surface added.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-cashshop && go test ./configuration/ -run SurpriseBox -v
```

Expected: FAIL — `undefined: GetSurpriseBoxTemplateIds`.

- [ ] **Step 3: Implement**

Create `services/atlas-cashshop/atlas.com/cashshop/configuration/tenant/cashshop/surprise/rest.go`:

```go
package surprise

// RestModel carries the Cash Shop Surprise box configuration. BoxTemplateIds
// is the set of cash item template ids that open as a Surprise box; the
// pool a box draws from is the atlas-reward-pools pool whose id equals the
// box's template id. A list rather than a scalar so a tenant can designate
// additional boxes beyond the stock 5222000 (task-207 FR-2.2 / DOM-25).
type RestModel struct {
	BoxTemplateIds []uint32 `json:"boxTemplateIds,omitempty"`
}
```

Modify `configuration/tenant/cashshop/rest.go`:

```go
package cashshop

import (
	"atlas-cashshop/configuration/tenant/cashshop/commodities"
	"atlas-cashshop/configuration/tenant/cashshop/surprise"
)

type RestModel struct {
	Commodities commodities.RestModel `json:"commodities"`
	Surprise    surprise.RestModel    `json:"surprise"`
}
```

Add to `configuration/registry.go`:

```go
// DefaultSurpriseBoxTemplateId is the stock Cash Shop Surprise box. It is
// the fallback, not a constant the open path compares against directly:
// GetSurpriseBoxTemplateIds is the only reader, and a tenant may override
// or extend the list.
const DefaultSurpriseBoxTemplateId = uint32(5222000)

// GetSurpriseBoxTemplateIds returns the cash item template ids that open as
// a Cash Shop Surprise box for this tenant. GetTenantConfig returns a zero
// RestModel when the fetch fails, so an empty list means "unconfigured" and
// falls back to the stock box rather than disabling the feature.
func GetSurpriseBoxTemplateIds(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) []uint32 {
	cfg, _ := GetTenantConfig(l, ctx, tenantId)
	if len(cfg.CashShop.Surprise.BoxTemplateIds) == 0 {
		return []uint32{DefaultSurpriseBoxTemplateId}
	}
	return cfg.CashShop.Surprise.BoxTemplateIds
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd services/atlas-cashshop && go test ./configuration/... -v
```

Expected: PASS.

- [ ] **Step 5: Add the key to the tenant configuration templates**

The tenant configuration resource is served by `atlas-tenants`/`atlas-configurations`, and a config key absent from every version's template is a recurring silent failure. Find where the cashshop tenant config is seeded:

```bash
grep -rln 'hourlyExpirations' services/atlas-configurations/ services/atlas-tenants/ 2>/dev/null
```

Add a `"surprise": {"boxTemplateIds": [5222000]}` sibling to `"commodities"` in **every** file that survey returns — the config-table-in-all-versions rule. If the survey returns nothing (the key is purely runtime-optional), record that in the commit message and rely on the accessor's default.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-cashshop/ services/atlas-configurations/ services/atlas-tenants/ 2>/dev/null
git commit -m "feat(task-207): add tenant-configured surprise box template ids"
```

---

### Task 8: `atlas-cashshop` — the idempotency ledger

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/surprise/opening/entity.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/surprise/opening/administrator.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/surprise/opening/administrator_test.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/main.go` (register the migration)

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces:
  - `func opening.Migration(db *gorm.DB) error`
  - `func opening.Insert(db *gorm.DB, tenantId uuid.UUID, transactionId uuid.UUID, accountId uint32, assetId uint32) error` — returns `opening.ErrAlreadyOpened` on primary-key collision
  - `var opening.ErrAlreadyOpened = errors.New("surprise box already opened for this transaction")`

**Background:** PRD FR-4.4. `transactionId` is minted by `atlas-channel` per click and travels on the command, so a Kafka redelivery replays the same id while a genuine second click gets a new one. The insert is the **first** statement in the open transaction: a redelivery hits the PK violation, the transaction aborts, and nothing is granted.

This is deliberately a real ledger row rather than an optimistic compare-and-set on the box quantity — a CAS would still consume a *second* box on redelivery when the player holds a stack (`design.md` §4.3).

- [ ] **Step 1: Write the failing test**

Create `services/atlas-cashshop/atlas.com/cashshop/surprise/opening/administrator_test.go`. Use the service's existing test-database helper — find it first:

```bash
grep -rn 'func .*testDatabase\|sqlite\|gorm.Open' services/atlas-cashshop/atlas.com/cashshop/*/*_test.go | head -5
```

```go
func TestInsertIsIdempotentOnTransactionId(t *testing.T) {
	db := testDatabase(t) // whatever the survey above named
	if err := Migration(db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	tenantId, txId := uuid.New(), uuid.New()

	if err := Insert(db, tenantId, txId, 10, 100); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	err := Insert(db, tenantId, txId, 10, 100)
	if !errors.Is(err, ErrAlreadyOpened) {
		t.Fatalf("second insert err = %v, want ErrAlreadyOpened", err)
	}
}

// The primary key is (tenant_id, transaction_id): the SAME transaction id
// belonging to a different tenant is a different opening.
func TestInsertIsScopedByTenant(t *testing.T) {
	db := testDatabase(t)
	if err := Migration(db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	txId := uuid.New()
	if err := Insert(db, uuid.New(), txId, 10, 100); err != nil {
		t.Fatalf("tenant A: %v", err)
	}
	if err := Insert(db, uuid.New(), txId, 10, 100); err != nil {
		t.Fatalf("tenant B must not collide with tenant A: %v", err)
	}
}

func TestInsertAllowsDistinctTransactions(t *testing.T) {
	db := testDatabase(t)
	if err := Migration(db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	tenantId := uuid.New()
	if err := Insert(db, tenantId, uuid.New(), 10, 100); err != nil {
		t.Fatalf("first click: %v", err)
	}
	// A genuine second click mints a new transaction id and must succeed.
	if err := Insert(db, tenantId, uuid.New(), 10, 100); err != nil {
		t.Fatalf("second click: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-cashshop && go test ./surprise/opening/ -v
```

Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement the entity and administrator**

Create `services/atlas-cashshop/atlas.com/cashshop/surprise/opening/entity.go`:

```go
package opening

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

// entity is the Cash Shop Surprise open ledger. One row per successfully
// committed open, keyed by the transaction id atlas-channel mints per click.
// Its insert is the FIRST statement in the open transaction, so a Kafka
// redelivery of the same command hits the primary-key violation and the
// whole transaction aborts without granting anything (task-207 FR-4.4).
//
// A real ledger row rather than a compare-and-set on the box quantity: a CAS
// would still consume a SECOND box on redelivery when the player holds a
// stack.
type entity struct {
	TenantId      uuid.UUID `gorm:"primaryKey;not null"`
	TransactionId uuid.UUID `gorm:"primaryKey;not null"`
	AccountId     uint32    `gorm:"not null"`
	AssetId       uint32    `gorm:"not null"`
	CreatedAt     time.Time `gorm:"not null"`
}

func (e entity) TableName() string {
	return "cash_surprise_openings"
}
```

Create `services/atlas-cashshop/atlas.com/cashshop/surprise/opening/administrator.go`:

```go
package opening

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrAlreadyOpened means this (tenant, transaction) pair has already been
// committed — a Kafka redelivery, not a new click. Callers treat it as
// success-without-effect rather than as a failure to report to the client.
var ErrAlreadyOpened = errors.New("surprise box already opened for this transaction")

// Insert writes the ledger row. It MUST be the first statement in the open
// transaction so a duplicate aborts before any state changes.
func Insert(db *gorm.DB, tenantId uuid.UUID, transactionId uuid.UUID, accountId uint32, assetId uint32) error {
	e := entity{
		TenantId:      tenantId,
		TransactionId: transactionId,
		AccountId:     accountId,
		AssetId:       assetId,
		CreatedAt:     time.Now(),
	}
	err := db.Create(&e).Error
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrAlreadyOpened
	}
	return err
}
```

`gorm.ErrDuplicatedKey` requires the dialector's `TranslateError` option. Verify how the service opens its database:

```bash
grep -rn 'TranslateError\|postgres.Open\|gorm.Open' services/atlas-cashshop/atlas.com/cashshop/main.go services/atlas-cashshop/atlas.com/cashshop/*/*.go | head
grep -rn 'ErrDuplicatedKey\|23505\|UniqueViolation' services/ libs/ --include='*.go' | head
```

If `TranslateError` is not enabled, follow whatever duplicate-key detection the repo already uses (the survey's second command shows it). Do not add a bare string match on the driver error text if a typed check already exists elsewhere.

- [ ] **Step 4: Register the migration**

Find where `atlas-cashshop` runs its AutoMigrations:

```bash
grep -n 'Migration' services/atlas-cashshop/atlas.com/cashshop/main.go
```

Add `opening.Migration` alongside the existing ones, in the same style.

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-cashshop && go build ./... && go test ./surprise/... -v
```

Expected: PASS (3 tests).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-cashshop/
git commit -m "feat(task-207): add cash_surprise_openings idempotency ledger"
```

---

### Task 9: `atlas-cashshop` — reward-pools REST client

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/rewardpool/rest.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/rewardpool/requests.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/rewardpool/processor.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/rewardpool/processor_test.go`
- Modify: the service's k8s manifests / env wiring if `GACHAPONS` is not already a known service URL for this service

**Interfaces:**
- Consumes: nothing from earlier tasks (the wire contract comes from Task 6's `reward.RestModel`).
- Produces:
  - `type rewardpool.Model struct` with getters `ItemId() uint32`, `Quantity() uint32`, `CommodityId() uint32`
  - `type rewardpool.Processor interface { SelectReward(boxTemplateId uint32) (Model, error) }`
  - `func rewardpool.NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`
  - `var rewardpool.ErrPoolMissing`, `var rewardpool.ErrPoolEmpty`

**Background:** The precedent to copy verbatim is `services/atlas-channel/atlas.com/channel/incubator/requests.go` — same endpoint, same nil-body POST:

```go
func getBaseRequest() string { return requests.RootUrl("GACHAPONS") }

func requestSelectReward(eggId uint32) requests.Request[RewardRestModel] {
	url := fmt.Sprintf("%sgachapons/%d/rewards/select", getBaseRequest(), eggId)
	return requests.PostRequest[RewardRestModel](url, nil)
}
```

The nil body matters: `jsonapi.Marshal` panics on a body value that does not implement `MarshalIdentifier`, and the server reads no request body.

A `cash-surprise` pool's id **is** the box template id (`design.md` §4.1), so `POST /gachapons/5222000/rewards/select` already resolves the right pool with no new endpoint.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-cashshop/atlas.com/cashshop/rewardpool/processor_test.go` with an `httptest.Server` standing in for atlas-reward-pools. Check how sibling packages point the request layer at a test server:

```bash
grep -rn 'httptest.NewServer' services/atlas-cashshop/atlas.com/cashshop/ services/atlas-channel/atlas.com/channel/ | head -5
```

```go
func TestSelectRewardReturnsCommodity(t *testing.T) {
	// httptest server answering 200 with a JSON:API gachapon-rewards
	// document carrying itemId 5222001, quantity 1, commodityId 40000.
	// assert: Model carries all three.
}

// 404 = no cash-surprise pool is configured for this box template id.
func TestSelectRewardMissingPoolMapsToErrPoolMissing(t *testing.T) {
	// httptest server answering 404
	// assert: errors.Is(err, ErrPoolMissing)
}

// 409 = the pool exists but has no eligible entries (Task 6).
func TestSelectRewardEmptyPoolMapsToErrPoolEmpty(t *testing.T) {
	// httptest server answering 409
	// assert: errors.Is(err, ErrPoolEmpty)
}

func TestSelectRewardTransportFailureIsNotSwallowed(t *testing.T) {
	// httptest server answering 500
	// assert: err != nil and is NEITHER ErrPoolMissing NOR ErrPoolEmpty —
	// an infrastructure fault must not be reported as a misconfigured pool.
}
```

Fill each body out completely against whichever seam the survey found.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-cashshop && go test ./rewardpool/ -v
```

Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement**

`rewardpool/rest.go`:

```go
package rewardpool

// RewardRestModel mirrors atlas-reward-pools reward/rest.go. CommodityId is
// the cash shop commodity (serial number) a cash-surprise pool awards; the
// commodity catalog owns the reward's itemId/count/period, so the grant path
// resolves the commodity rather than trusting ItemId here.
type RewardRestModel struct {
	Id          string `json:"-"`
	ItemId      uint32 `json:"itemId"`
	Quantity    uint32 `json:"quantity"`
	Tier        string `json:"tier"`
	Weight      uint32 `json:"weight"`
	CommodityId uint32 `json:"commodityId"`
	GachaponId  string `json:"gachaponId"`
}

func (r RewardRestModel) GetName() string { return "gachapon-rewards" }
func (r RewardRestModel) GetID() string   { return r.Id }
func (r *RewardRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}
```

`rewardpool/requests.go`: copy the incubator file's two functions, renaming `requestSelectReward(eggId uint32)` to take `boxTemplateId uint32`, and drop the atlas-data NPC request (not needed here).

`rewardpool/processor.go`:

```go
package rewardpool

import (
	"context"
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// ErrPoolMissing means no cash-surprise pool is configured for this box
// template id (404). ErrPoolEmpty means the pool exists but has no eligible
// entries (409, task-207 FR-3.7). They are distinct so the open path can log
// POOL_MISSING vs POOL_EMPTY — the client sees the same bare FAILED arm
// either way, so the log is the only place the difference survives.
var (
	ErrPoolMissing = errors.New("no cash-surprise pool configured for box template")
	ErrPoolEmpty   = errors.New("cash-surprise pool has no eligible entries")
)

type Model struct {
	itemId      uint32
	quantity    uint32
	commodityId uint32
}

func (m Model) ItemId() uint32      { return m.itemId }
func (m Model) Quantity() uint32    { return m.quantity }
func (m Model) CommodityId() uint32 { return m.commodityId }

type Processor interface {
	SelectReward(boxTemplateId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) SelectReward(boxTemplateId uint32) (Model, error) {
	rm, err := requestSelectReward(boxTemplateId)(p.l, p.ctx)
	if err != nil {
		return Model{}, classifySelectError(err)
	}
	return Model{itemId: rm.ItemId, quantity: rm.Quantity, commodityId: rm.CommodityId}, nil
}
```

`classifySelectError` maps the request layer's error to the two sentinels. The request layer already maps 404 to `requests.ErrNotFound` (see the incubator client's comment), so:

```go
// classifySelectError distinguishes the two *configuration* faults from
// everything else. An infrastructure fault must NOT be reported as a
// misconfigured pool — the operator would go looking in the wrong place.
func classifySelectError(err error) error {
	if errors.Is(err, requests.ErrNotFound) {
		return errors.Join(ErrPoolMissing, err)
	}
	if isStatus(err, http.StatusConflict) {
		return errors.Join(ErrPoolEmpty, err)
	}
	return err
}
```

`isStatus` depends on what the request layer exposes for non-404 status codes. Determine it before writing:

```bash
grep -rn 'ErrNotFound\|StatusCode\|type .*Error' libs/atlas-rest/requests/*.go | head -20
```

If the layer exposes a typed status error, match on it. If it exposes **only** `ErrNotFound` and an opaque error, add a 409-carrying error type to `libs/atlas-rest/requests` following that package's existing conventions rather than string-matching the message — and note the lib change in the commit message, since it means `docker buildx bake` must cover every service that imports it.

- [ ] **Step 4: Confirm the `GACHAPONS` service URL reaches atlas-cashshop**

```bash
grep -rn 'GACHAPONS' deploy/ services/atlas-cashshop/ .github/ 2>/dev/null | head
```

`requests.RootUrl("GACHAPONS")` reads a `GACHAPONS_SERVICE_URL`-style env var. If `atlas-cashshop`'s deployment does not set it, add it to the k8s base **and both kustomize overlays** — a value set in base but missing from an overlay is a known silent failure. Never hard-code the base namespace into the URL.

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-cashshop && go build ./... && go test ./rewardpool/ -v
```

Expected: PASS (4 tests).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-cashshop/ deploy/ libs/atlas-rest/ 2>/dev/null
git commit -m "feat(task-207): add atlas-reward-pools client to atlas-cashshop"
```

---

### Task 10: `atlas-cashshop` — the capacity rule

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/surprise/capacity.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/surprise/capacity_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `func surprise.HasRoomForSwap(assetCount uint32, capacity uint32, boxQuantity uint32) bool`

**Background:** PRD FR-2.3 / `design.md` §2.4. The reward is created while the box is consumed, so the **peak** slot count matters, not the net:

- box quantity > 1 → the box row survives, so the grant needs one free slot: `assetCount < capacity`.
- box quantity == 1 → the box row is released, so the grant is slot-neutral: `assetCount <= capacity`.

`DefaultCapacity` is 55 (`cashshop/inventory/compartment/processor.go:21`). Note the existing purchase path uses the stricter `ccm.Capacity() <= uint32(len(ccm.Assets()))` rejection — that is correct for a purchase (pure addition) and must not be changed.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-cashshop/atlas.com/cashshop/surprise/capacity_test.go`:

```go
package surprise

import "testing"

func TestHasRoomForSwap(t *testing.T) {
	tests := []struct {
		name        string
		assetCount  uint32
		capacity    uint32
		boxQuantity uint32
		want        bool
	}{
		// Stack of boxes: the box row survives the consume, so the reward
		// genuinely needs a spare slot.
		{"stacked box, spare slot", 54, 55, 3, true},
		{"stacked box, locker exactly full", 55, 55, 3, false},
		{"stacked box, one short of full", 54, 55, 2, true},

		// Last box: the row is released, so the grant is slot-neutral and a
		// completely full locker is still fine.
		{"last box, locker exactly full", 55, 55, 1, true},
		{"last box, spare slot", 40, 55, 1, true},

		// Over-capacity lockers (data drift) must still be rejected for the
		// stacked case and permitted for the neutral case only at equality.
		{"over capacity, stacked box", 56, 55, 2, false},
		{"over capacity, last box", 56, 55, 1, false},
	}
	for _, tt := range tests {
		if got := HasRoomForSwap(tt.assetCount, tt.capacity, tt.boxQuantity); got != tt.want {
			t.Errorf("%s: HasRoomForSwap(%d, %d, %d) = %v, want %v",
				tt.name, tt.assetCount, tt.capacity, tt.boxQuantity, got, tt.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-cashshop && go test ./surprise/ -run HasRoomForSwap -v
```

Expected: FAIL — `undefined: HasRoomForSwap`.

- [ ] **Step 3: Implement**

Create `services/atlas-cashshop/atlas.com/cashshop/surprise/capacity.go`:

```go
package surprise

// HasRoomForSwap reports whether a compartment holding assetCount assets at
// the given capacity can absorb a Surprise open.
//
// The reward is created while the box is consumed, so the PEAK slot count
// decides, not the net:
//   - boxQuantity > 1: the box row survives the decrement, so the reward
//     needs its own free slot.
//   - boxQuantity == 1: the box row is released, so the grant is
//     slot-neutral and an exactly-full locker is fine.
//
// An over-capacity locker (assetCount > capacity, possible through data
// drift) is rejected in both branches: the neutral case permits equality,
// not excess.
func HasRoomForSwap(assetCount uint32, capacity uint32, boxQuantity uint32) bool {
	if boxQuantity == 1 {
		return assetCount <= capacity
	}
	return assetCount < capacity
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd services/atlas-cashshop && go test ./surprise/ -run HasRoomForSwap -v
```

Expected: PASS (7 sub-cases).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/surprise/capacity.go services/atlas-cashshop/atlas.com/cashshop/surprise/capacity_test.go
git commit -m "feat(task-207): add surprise-open capacity rule"
```

---

### Task 11: `atlas-cashshop` — Kafka command and status-event contracts

**Files:**
- Modify: `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/cashshop/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go` (mirror)

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces (identical json shapes on both sides of the topic):
  - `const cashshop.CommandTypeOpenSurprise = "OPEN_SURPRISE"`
  - `type OpenSurpriseCommandBody struct { TransactionId uuid.UUID \`json:"transactionId"\`; AccountId uint32 \`json:"accountId"\`; CashId int64 \`json:"cashId"\` }`
  - `const cashshop.StatusEventTypeSurpriseOpened = "SURPRISE_OPENED"`
  - `const cashshop.StatusEventTypeSurpriseFailed = "SURPRISE_FAILED"`
  - `type SurpriseOpenedEventBody struct { CompartmentId uuid.UUID \`json:"compartmentId"\`; BoxCashId int64 \`json:"boxCashId"\`; BoxRemaining uint32 \`json:"boxRemaining"\`; RewardAssetId uint32 \`json:"rewardAssetId"\`; RewardTemplateId uint32 \`json:"rewardTemplateId"\`; RewardCount uint32 \`json:"rewardCount"\` }`
  - `type SurpriseFailedEventBody struct { Reason string \`json:"reason"\` }`
  - producers `cashshop2.SurpriseOpenedStatusEventProvider(...)`, `cashshop2.SurpriseFailedStatusEventProvider(characterId uint32, reason string)`

**Background:** The two message packages are hand-mirrored between `atlas-cashshop` (producer) and `atlas-channel` (consumer) — they are separate Go modules and the json tags are the only contract. `atlas-channel`'s copy currently carries `CommandTypeMoveFromCashInventory` which `atlas-cashshop`'s does not, so they are already allowed to diverge; add the new constants and bodies to **both**.

The failure reasons are server-side only. There is no error-code field on this wire (`design.md` §2.3), so `Reason` never reaches the client — it exists for the log and for operators. The closed set: `BOX_NOT_FOUND`, `NOT_OWNED`, `NOT_A_SURPRISE_BOX`, `LOCKER_FULL`, `POOL_EMPTY`, `POOL_MISSING`, `COMMODITY_MISSING`, `INTERNAL`.

`BoxRemaining` is the box's quantity *after* the decrement — the client removes the locker row when it is 0. `RewardCount` comes from the commodity's `Count()`, not from the pool entry's quantity.

- [ ] **Step 1: Add the constants and bodies to atlas-cashshop**

In `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go`, add to the command const block:

```go
	CommandTypeOpenSurprise = "OPEN_SURPRISE"
```

and below `RequestCharacterSlotIncreaseByItemCommandBody`:

```go
// OpenSurpriseCommandBody opens one Cash Shop Surprise box. TransactionId is
// minted by atlas-channel per click and is the idempotency key: a Kafka
// redelivery replays the same id (and is rejected by the openings ledger)
// while a genuine second click gets a new one. CashId identifies the box in
// the account's cash locker — the server resolves and re-validates it, since
// the edge does not own the locker.
type OpenSurpriseCommandBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	AccountId     uint32    `json:"accountId"`
	CashId        int64     `json:"cashId"`
}
```

Add to the status-event const block:

```go
	StatusEventTypeSurpriseOpened = "SURPRISE_OPENED"
	StatusEventTypeSurpriseFailed = "SURPRISE_FAILED"
```

and the two bodies:

```go
// SurpriseOpenedEventBody carries everything the channel writer needs for
// the CCashShop::OnCashItemGachaponResult SUCCESS arm. BoxRemaining is the
// box's quantity AFTER the decrement — the client removes the locker row
// when it is 0. RewardCount comes from the commodity catalog, not from the
// pool entry.
type SurpriseOpenedEventBody struct {
	CompartmentId    uuid.UUID `json:"compartmentId"`
	BoxCashId        int64     `json:"boxCashId"`
	BoxRemaining     uint32    `json:"boxRemaining"`
	RewardAssetId    uint32    `json:"rewardAssetId"`
	RewardTemplateId uint32    `json:"rewardTemplateId"`
	RewardCount      uint32    `json:"rewardCount"`
}

// SurpriseFailedEventBody's Reason NEVER reaches the client: the FAILED arm
// of this packet has an empty body and no error-code field (design.md §2.3).
// It exists for the log and for operators. Closed set: BOX_NOT_FOUND,
// NOT_OWNED, NOT_A_SURPRISE_BOX, LOCKER_FULL, POOL_EMPTY, POOL_MISSING,
// COMMODITY_MISSING, INTERNAL.
type SurpriseFailedEventBody struct {
	Reason string `json:"reason"`
}
```

- [ ] **Step 2: Add the producers**

In `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/cashshop/producer.go`, add `SurpriseOpenedStatusEventProvider` and `SurpriseFailedStatusEventProvider` following `PurchaseStatusEventProvider`'s exact shape (same key derivation, same `StatusEvent[...]` wrapper). Read that function first and mirror it.

- [ ] **Step 3: Mirror the contract into atlas-channel**

Add the same three constants and three body structs (byte-identical json tags) to `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go`. `atlas-channel` also needs a command producer for `OPEN_SURPRISE` — add `OpenSurpriseCommandProvider(characterId uint32, transactionId uuid.UUID, accountId uint32, cashId int64)` to `services/atlas-channel/atlas.com/channel/kafka/producer/…` following `RequestPurchaseCommandProvider`'s shape (locate it with `grep -rn 'RequestPurchaseCommandProvider' services/atlas-channel/`).

- [ ] **Step 4: Write a contract test pinning the json tags**

The two modules cannot import each other, so the only guard against drift is a test that pins the wire shape. Add to `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka_test.go` (create if absent):

```go
// The atlas-channel mirror of these bodies is a hand-maintained copy in a
// separate Go module — the json tags are the ONLY contract. Pin them so a
// rename here fails loudly instead of silently dropping fields at runtime.
func TestOpenSurpriseCommandBodyWireShape(t *testing.T) {
	b, err := json.Marshal(OpenSurpriseCommandBody{
		TransactionId: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		AccountId:     10,
		CashId:        1234567890,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"transactionId":"00000000-0000-0000-0000-000000000001","accountId":10,"cashId":1234567890}`
	if string(b) != want {
		t.Fatalf("wire shape drifted:\n got %s\nwant %s", b, want)
	}
}

func TestSurpriseOpenedEventBodyWireShape(t *testing.T) {
	b, err := json.Marshal(SurpriseOpenedEventBody{
		CompartmentId:    uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		BoxCashId:        1234567890,
		BoxRemaining:     2,
		RewardAssetId:    77,
		RewardTemplateId: 5222001,
		RewardCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"compartmentId":"00000000-0000-0000-0000-000000000002","boxCashId":1234567890,"boxRemaining":2,"rewardAssetId":77,"rewardTemplateId":5222001,"rewardCount":1}`
	if string(b) != want {
		t.Fatalf("wire shape drifted:\n got %s\nwant %s", b, want)
	}
}

func TestSurpriseFailedEventBodyWireShape(t *testing.T) {
	b, err := json.Marshal(SurpriseFailedEventBody{Reason: "LOCKER_FULL"})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"reason":"LOCKER_FULL"}` {
		t.Fatalf("wire shape drifted: %s", b)
	}
}
```

Add the identical three tests to the `atlas-channel` mirror package.

- [ ] **Step 5: Run tests**

```bash
cd services/atlas-cashshop && go build ./... && go test ./kafka/... -v 2>&1 | tail -20
cd ../atlas-channel && go build ./... && go test ./kafka/... -v 2>&1 | tail -20
```

Expected: PASS in both modules.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-cashshop/ services/atlas-channel/
git commit -m "feat(task-207): add OPEN_SURPRISE command and SURPRISE_OPENED/FAILED status events"
```

---

### Task 12: `atlas-cashshop` — the open orchestration

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/surprise/processor.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/surprise/processor_test.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/cashshop/commodity/model.go` (add an `Id()` getter for logging)

**Interfaces:**
- Consumes: `surprise.HasRoomForSwap` (Task 10); `opening.Insert` / `opening.ErrAlreadyOpened` (Task 8); `rewardpool.Processor` / `ErrPoolMissing` / `ErrPoolEmpty` (Task 9); `configuration.GetSurpriseBoxTemplateIds` (Task 7); the Kafka contracts (Task 11).
- Produces:
  - `type surprise.Processor interface { OpenAndEmit(transactionId uuid.UUID, accountId uint32, characterId uint32, cashId int64) error; Open(mb *message.Buffer) func(transactionId uuid.UUID, accountId uint32, characterId uint32, cashId int64) error }`
  - `func surprise.NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor`

**Background — the sequence (`design.md` §4.2):**

1. **Resolve.** Pick the compartment for the character's job type (Explorer/Cygnus/Legend — the same three-way branch `Purchase` uses at `cashshop/processor.go:126-133`), fetch it with `cicP.GetByAccountIdAndType(accountId, compartmentType)` (its `Assets()` are decorated), and scan for the asset whose `CashId()` matches. Reject when absent (`BOX_NOT_FOUND`) or when its `TemplateId()` is not in the configured Surprise list (`NOT_A_SURPRISE_BOX`). Ownership (FR-2.1) is enforced **structurally**: the compartment is looked up by `accountId`, so an asset belonging to another account is simply not in the scanned set — which is why `NOT_OWNED` and `BOX_NOT_FOUND` collapse to the same observable outcome. Log both possibilities.
2. **Capacity.** `HasRoomForSwap(uint32(len(ccm.Assets())), ccm.Capacity(), box.Quantity())`. Failure ⇒ `LOCKER_FULL`.
3. **Roll.** `rewardpool.NewProcessor(l, ctx).SelectReward(box.TemplateId())`. `ErrPoolMissing` ⇒ `POOL_MISSING`, `ErrPoolEmpty` ⇒ `POOL_EMPTY`, anything else ⇒ `INTERNAL`.
4. **Resolve the commodity.** `commodity.NewProcessor(l, ctx).GetById(reward.CommodityId())` → `ItemId()`, `Count()`, `Period()`. Missing ⇒ `COMMODITY_MISSING`.
5. **One transaction.** `database.ExecuteTransaction` + `message.Emit(outbox.EmitProvider(...))`, the `PurchaseAndEmit` shape:
   - `opening.Insert(tx, tenantId, transactionId, accountId, box.Id())` **first**; `ErrAlreadyOpened` ⇒ abort the transaction and return nil (success-without-effect, no event).
   - decrement the box: `astP.UpdateQuantity(box.Id(), box.Quantity()-1)`, or `astP.Release(mb)(box.Id())` when the resulting quantity is 0.
   - create the reward: `astP.Create(mb)(ccm.Id(), ci.ItemId(), reward.CommodityId(), ci.Count(), 0, characterId)`.
   - `mb.Put(cashshop.EnvEventTopicStatus, cashshop2.SurpriseOpenedStatusEventProvider(...))`.

**Deviation from PRD FR-4.2, deliberate:** the PRD says "via the existing compartment `Accept` path". `compartment.Accept` is the **saga-facing** inbound path and emits `ACCEPTED` status events for a saga to correlate. The in-service creation path is `asset.Create`, which is what `Purchase` uses and which produces the identical flattened row (`cashId`, `commodityId`, `templateId`, `quantity`, `expiration`, `purchasedBy`). `asset.Create` already derives `expiration` from the commodity's `period` (`asset/processor.go:82-95`), satisfying FR-4.2's intent. Its named mechanism is not the right one here.

**FR-4.5 recursion:** a pool configured to award a Surprise box creates an infinite box. This is honoured-by-configuration, **not** blocked in code. Add a `WARN` log when the rolled commodity's `itemId` is itself in the configured Surprise list, so the operator sees it.

**Important:** `astP.UpdateQuantity` and `astP.Release` both run on `p.db`, so the processor must be reconstructed against `tx` inside the closure (`NewProcessor(p.l, p.ctx, tx)`) exactly as `PurchaseAndEmit` does, or the writes escape the transaction and FR-4.1 is violated. Verify this by reading `asset.ProcessorImpl.UpdateQuantity` — it calls `updateQuantity(p.db.WithContext(p.ctx), …)`, i.e. whatever `db` the processor was built with.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-cashshop/atlas.com/cashshop/surprise/processor_test.go`. Mirror the fixture style the service's other integration tests use (`grep -rn 'func Test' services/atlas-cashshop/atlas.com/cashshop/cashshop/*_test.go | head`). Cover, each fully written out:

```go
// Happy path: box quantity 3 → decremented to 2, reward created in the SAME
// compartment, SURPRISE_OPENED emitted carrying boxRemaining 2.
func TestOpenGrantsRewardAndDecrementsBox(t *testing.T)

// Last box: quantity 1 → the locker row is released, SURPRISE_OPENED carries
// boxRemaining 0 (the client removes the row on 0).
func TestOpenReleasesBoxAtZeroQuantity(t *testing.T)

// The reward carries commodityId, templateId, quantity and an expiration
// derived from the commodity's period — all read back off the created asset.
func TestOpenRewardCarriesCommodityDerivedFields(t *testing.T)

// FR-2.1: an asset belonging to another account is not in this account's
// compartment, so the open is rejected with NO state change.
func TestOpenRejectsAssetOwnedByAnotherAccount(t *testing.T)

// FR-2.2: an asset that exists and is owned but whose templateId is not a
// configured Surprise box is rejected.
func TestOpenRejectsNonSurpriseTemplate(t *testing.T)

// FR-2.3 through the real path, both branches of HasRoomForSwap.
func TestOpenRejectsWhenLockerFullAndBoxIsStacked(t *testing.T)
func TestOpenSucceedsWhenLockerFullAndBoxIsLast(t *testing.T)

// FR-6.4 / FR-4.1: an empty pool leaves the box untouched.
func TestOpenWithEmptyPoolLeavesBoxIntact(t *testing.T)

// FR-4.1: forcing the reward creation to fail must roll the decrement back.
func TestOpenIsAtomicWhenGrantFails(t *testing.T)

// FR-4.4: the same transactionId twice grants exactly once and consumes
// exactly one box.
func TestOpenIsIdempotentOnTransactionId(t *testing.T)

// A genuine second click (new transactionId) opens a second box.
func TestOpenTwiceWithDistinctTransactionIdsGrantsTwice(t *testing.T)
```

For `TestOpenIsAtomicWhenGrantFails`, force the failure by pointing the commodity processor at a stub that returns a valid commodity for the roll step but making `asset.Create` fail — e.g. a compartment id that violates the foreign key, or a commodity whose `itemId` triggers an error path. Whichever seam the existing tests already provide; do not add production-code test hooks.

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-cashshop && go test ./surprise/ -v 2>&1 | tail -20
```

Expected: FAIL — `undefined: NewProcessor`.

- [ ] **Step 3: Add the missing commodity getter**

`services/atlas-cashshop/atlas.com/cashshop/cashshop/commodity/model.go` has `itemId`/`count`/`price`/`period` getters but **no** `Id()`. The observability requirement (log the rolled commodity id) needs it:

```go
// Id is the cash shop commodity serial number — the value a cash-surprise
// reward pool entry names and the value GW_CashItemInfo carries as
// CommodityId.
func (m Model) Id() uint32 {
	return m.id
}
```

- [ ] **Step 4: Implement the processor**

Create `services/atlas-cashshop/atlas.com/cashshop/surprise/processor.go` following the sequence above. Structural requirements:

- `Processor` interface + `ProcessorImpl` struct + `NewProcessor(l, ctx, db)` + `var _ Processor = (*ProcessorImpl)(nil)`, matching `cashshop/processor.go`.
- The pure `Open(mb *message.Buffer)` / `OpenAndEmit(...)` pairing.
- Steps 1–4 run **outside** the transaction (nothing is mutated, so a failure there cannot partially apply). Only step 5 is transactional.
- Failure emission follows `Purchase`'s `rejectEmit` precedent: a rejection that commits **no** state change must fire on the **direct producer path**, not the outbox, or it leaks into the outbox as though it were part of a committed transaction. Since steps 1–4 are outside the transaction entirely, emit `SURPRISE_FAILED` there via `producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(...)` directly.
- Every path logs, with fields: `tenant`, `account_id`, `character_id`, `transaction_id`, `box_asset_id`, `box_template_id`, `pool_id` (= box template id), `commodity_id`, `reward_template_id`, `outcome`. This is a currency-adjacent path — NFR "Observability" requires an unexplained grant to be reconstructable from logs alone.
- The recursion WARN from the Background section.

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-cashshop && go test -race ./... 2>&1 | tail -25
```

Expected: PASS, all 11 surprise tests plus every pre-existing test.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-cashshop/
git commit -m "feat(task-207): implement transactional cash shop surprise open"
```

---

### Task 13: `atlas-cashshop` — consume the `OPEN_SURPRISE` command

**Files:**
- Modify: `services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go`

**Interfaces:**
- Consumes: `surprise.Processor` (Task 12); `cashshop.CommandTypeOpenSurprise` and `OpenSurpriseCommandBody` (Task 11).
- Produces: a registered handler on `COMMAND_TOPIC_CASH_SHOP` for `OPEN_SURPRISE`.

**Background:** A shared command topic fans every message to every registered handler, so each handler **must** guard on `c.Type != <its own type>` and return early — otherwise a body meant for another handler unmarshals into the wrong struct and produces garbage. Read the existing `handleRequestPurchase` (or equivalently-named) handler in that file and copy its guard, registration, and tenant handling exactly.

- [ ] **Step 1: Write the failing test**

Add to the consumer's test file (create if absent, mirroring a sibling consumer test):

```go
// The command topic is shared: every handler sees every message, so the
// type guard is what keeps an OPEN_SURPRISE body from being unmarshalled by
// the purchase handler and vice versa.
func TestHandleOpenSurpriseIgnoresOtherCommandTypes(t *testing.T) {
	// dispatch a REQUEST_PURCHASE command into handleOpenSurprise
	// assert: no surprise processor call, no error
}

func TestHandleOpenSurpriseInvokesProcessor(t *testing.T) {
	// dispatch an OPEN_SURPRISE command with a known transactionId/accountId/cashId
	// assert: the processor receives exactly those values
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-cashshop && go test ./kafka/consumer/cashshop/ -run OpenSurprise -v
```

Expected: FAIL.

- [ ] **Step 3: Implement the handler and register it**

```go
func handleOpenSurprise(db *gorm.DB) message.Handler[cashshop.Command[cashshop.OpenSurpriseCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.OpenSurpriseCommandBody]) {
		// COMMAND_TOPIC_CASH_SHOP is shared across handlers: without this
		// guard another command's body unmarshals into OpenSurpriseCommandBody
		// and produces a garbage open request.
		if c.Type != cashshop.CommandTypeOpenSurprise {
			return
		}
		err := surprise.NewProcessor(l, ctx, db).OpenAndEmit(c.Body.TransactionId, c.Body.AccountId, c.CharacterId, c.Body.CashId)
		if err != nil {
			l.WithError(err).Errorf("Unable to open surprise box [%d] for character [%d].", c.Body.CashId, c.CharacterId)
		}
	}
}
```

Register it alongside the other command handlers in the same `InitHandlers` function, matching the existing `rf(t, message.AdaptHandler(message.PersistentConfig(...)))` shape (read the file — the exact signature differs from `atlas-channel`'s).

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-cashshop && go build ./... && go test -race ./... 2>&1 | tail -15
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-cashshop/
git commit -m "feat(task-207): consume OPEN_SURPRISE command in atlas-cashshop"
```

---

### Task 14: `atlas-channel` — handler, command producer, and writer registration

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/cash_item_gachapon.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/cash_item_gachapon_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/cashshop/processor.go` (new `OpenSurprise` method)
- Modify: `services/atlas-channel/atlas.com/channel/main.go:625` (writer list), `:925` area (handler map)

**Interfaces:**
- Consumes: `cashsb.CashItemGachaponHandle` and `CashItemGachaponButton` (Task 1); `cashcb.CashItemGachaponResultWriter` (Task 2); the `OPEN_SURPRISE` producer (Task 11).
- Produces:
  - `func handler.CashItemGachaponHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{})`
  - `cashshop.Processor.OpenSurprise(accountId uint32, characterId uint32, cashId int64) error` on the channel-side processor

**Background:** The edge does **not** own the locker, so **no validation lives in the handler** (`design.md` §4.4): it decodes, mints a `transactionId`, and produces the command. The pattern to copy is `socket/handler/cash_shop_check_wallet.go` for the handler shell and `cashshop/processor.go`'s `RequestPurchase` for the producer method.

Registration is in two hand-maintained lists in `main.go`: `produceWriters()` (the writer name list, around line 625) and the `handlerMap` assignments (around line 925). A writer missing from `produceWriters()` cannot be announced at runtime even though the codec compiles.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/cash_item_gachapon_test.go`:

```go
// The edge does not own the locker: the handler must decode and forward,
// performing NO ownership, template, or capacity validation of its own.
// Every check lives in atlas-cashshop, which is the only service that can
// make them atomically with the grant.
func TestCashItemGachaponHandleProducesCommand(t *testing.T) {
	// build a reader over the 8-byte little-endian cash id d202964900000000
	// dispatch through CashItemGachaponHandleFunc with a stub producer
	// assert: exactly one OPEN_SURPRISE command produced, carrying
	//   accountId = session account, characterId = session character,
	//   cashId = 1234567890, and a NON-NIL transactionId
}

// Two clicks must mint two distinct transaction ids, or the openings ledger
// would reject the second as a redelivery.
func TestCashItemGachaponHandleMintsDistinctTransactionIds(t *testing.T) {
	// dispatch twice; assert the two transactionIds differ
}
```

Mirror whatever producer-stubbing seam the sibling handler tests use (`grep -rn 'func Test' services/atlas-channel/atlas.com/channel/socket/handler/*_test.go | head`); `character_cash_item_use_test.go` exists and is the nearest neighbour.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel && go test ./socket/handler/ -run CashItemGachapon -v
```

Expected: FAIL — `undefined: CashItemGachaponHandleFunc`.

- [ ] **Step 3: Add the processor method**

In `services/atlas-channel/atlas.com/channel/cashshop/processor.go`, add to the `Processor` interface and implement:

```go
	OpenSurprise(accountId uint32, characterId uint32, cashId int64) error
```

```go
// OpenSurprise forwards a Cash Shop Surprise open request. The transaction
// id is minted here, once per click: atlas-cashshop's openings ledger keys
// idempotency on it, so a Kafka redelivery replays this id and is rejected
// while a genuine second click gets a fresh one.
func (p *ProcessorImpl) OpenSurprise(accountId uint32, characterId uint32, cashId int64) error {
	transactionId := uuid.New()
	p.l.Debugf("Character [%d] opening surprise box [%d]. Transaction [%s].", characterId, cashId, transactionId)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(OpenSurpriseCommandProvider(characterId, transactionId, accountId, cashId))
}
```

- [ ] **Step 4: Write the handler**

Create `services/atlas-channel/atlas.com/channel/socket/handler/cash_item_gachapon.go`:

```go
package handler

import (
	"atlas-channel/cashshop"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// CashItemGachaponHandleFunc handles the Cash Shop Surprise "Open" button
// (CUICashItemGachapon::OnButtonClicked). It performs NO validation: the
// edge does not own the cash locker, and every check — ownership, template,
// capacity — has to happen atomically with the grant, which only
// atlas-cashshop can do. The client self-gates re-clicks via m_nState and
// does not arm the excl-request gate, so nothing is unlocked here.
func CashItemGachaponHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := cashsb.CashItemGachaponButton{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		err := cashshop.NewProcessor(l, ctx).OpenSurprise(s.AccountId(), s.CharacterId(), p.CashId())
		if err != nil {
			l.WithError(err).Errorf("Unable to request surprise box open for character [%d].", s.CharacterId())
		}
	}
}
```

- [ ] **Step 5: Register the handler and the writer**

In `services/atlas-channel/atlas.com/channel/main.go`:
- add `cashcb.CashItemGachaponResultWriter,` to the `produceWriters()` slice, next to `cashcb.CashQueryResultWriter` (around line 625);
- add `handlerMap[cashsb.CashItemGachaponHandle] = handler.CashItemGachaponHandleFunc` next to `handlerMap[cashsb.CashShopCheckWalletHandle]` (around line 925).

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd services/atlas-channel && go build ./... && go test ./socket/handler/ -run CashItemGachapon -v
```

Expected: PASS (2 tests).

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/
git commit -m "feat(task-207): add CashItemGachapon handler and writer registration to atlas-channel"
```

---

### Task 15: `atlas-channel` — announce the result

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`

**Interfaces:**
- Consumes: `SurpriseOpenedEventBody` / `SurpriseFailedEventBody` (Task 11); `cashcb.CashItemGachaponSuccessBody` / `CashItemGachaponFailedBody` (Task 2).
- Produces: two registered handlers on `EVENT_TOPIC_CASH_SHOP_STATUS`.

**Background:** Both handlers follow `handleStatusEventPurchase`'s exact shape — type guard, `tenant.MustFromContext(ctx)` + `t.Is(sc.Tenant())` guard, then `session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error { … })`.

The success handler reads the reward asset back to build the `CashInventoryItem` blob — exactly as `handleStatusEventPurchase` does at `consumer.go:105-125` — so one place stays responsible for the asset→blob mapping.

Field mapping for the SUCCESS arm:
- `sn` ← `e.Body.BoxCashId` (the **box's** SN, not the reward's — the client matches it against `m_aCashItemInfo[i].liSN` to find the row to decrement)
- `remain` ← `int32(e.Body.BoxRemaining)`
- `newItem` ← the reward asset, built the `handleStatusEventPurchase` way
- `itemId` ← `int32(e.Body.RewardTemplateId)`
- `count` ← `byte(e.Body.RewardCount)`
- `jackpot` ← `0`. The jackpot byte only selects `CashGachaponJackpot` vs `CashGachaponNormal` sfx; nothing in the pool model expresses "this was a jackpot", so it is always the normal sfx. Comment this so it does not read as an oversight.

- [ ] **Step 1: Write the failing test**

Add to the consumer's test file:

```go
// The SUCCESS arm's `sn` is the BOX's cash serial, not the reward's: the
// client uses it to locate the locker row to decrement or remove.
func TestHandleSurpriseOpenedAnnouncesBoxSn(t *testing.T)

// remain 0 is how the client is told to remove the row entirely.
func TestHandleSurpriseOpenedCarriesZeroRemainOnLastBox(t *testing.T)

// The FAILED arm is mode-only; the event's Reason is server-side and must
// NOT appear on the wire.
func TestHandleSurpriseFailedAnnouncesModeOnly(t *testing.T)

// A status event for another tenant must be ignored.
func TestHandleSurpriseOpenedIgnoresOtherTenants(t *testing.T)
```

Write each out fully against the seam the file's existing tests use.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel && go test ./kafka/consumer/cashshop/ -run Surprise -v
```

Expected: FAIL.

- [ ] **Step 3: Implement both handlers and register them**

Add `handleStatusEventSurpriseOpened` and `handleStatusEventSurpriseFailed` to `consumer.go`, and append both to the `handles` chain in `InitHandlers` following the three existing `id, err = rf(t, message.AdaptHandler(message.PersistentConfig(...)))` blocks.

```go
func handleStatusEventSurpriseOpened(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.SurpriseOpenedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.SurpriseOpenedEventBody]) {
		if e.Type != cashshop2.StatusEventTypeSurpriseOpened {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			a, err := asset.NewProcessor(l, ctx).GetById(s.AccountId(), e.Body.CompartmentId, e.Body.RewardAssetId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve surprise reward asset [%d] for character [%d].", e.Body.RewardAssetId, e.CharacterId)
				return err
			}

			item := cashpkt.CashInventoryItem{
				CashId:      a.Item().CashId(),
				AccountId:   s.AccountId(),
				CharacterId: e.CharacterId,
				TemplateId:  a.Item().TemplateId(),
				CommodityId: a.CommodityId(),
				Quantity:    int16(a.Item().Quantity()),
				GiftFrom:    "",
				Expiration:  packetmodel.MsTime(a.Expiration()),
			}

			// sn is the BOX's serial — the client matches it against
			// m_aCashItemInfo[i].liSN to find the row to decrement, and
			// removes that row when remain is 0. jackpot is always 0: the
			// byte only picks CashGachaponJackpot vs CashGachaponNormal sfx
			// and the pool model has no notion of a jackpot tier.
			err = session.Announce(l)(ctx)(wp)(cashpkt.CashItemGachaponResultWriter)(
				cashpkt.CashItemGachaponSuccessBody(
					e.Body.BoxCashId,
					int32(e.Body.BoxRemaining),
					item,
					int32(e.Body.RewardTemplateId),
					byte(e.Body.RewardCount),
					0,
				))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce surprise open result to character [%d].", e.CharacterId)
				return err
			}
			return nil
		})
	}
}

func handleStatusEventSurpriseFailed(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.SurpriseFailedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.SurpriseFailedEventBody]) {
		if e.Type != cashshop2.StatusEventTypeSurpriseFailed {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		// The FAILED arm has an empty body — the client reads only the mode
		// byte, calls StringPool::GetString(<fixed id>) and shows a notice.
		// e.Body.Reason has no field to travel in and stays server-side.
		l.Infof("Surprise open failed for character [%d]: %s.", e.CharacterId, e.Body.Reason)

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			err := session.Announce(l)(ctx)(wp)(cashpkt.CashItemGachaponResultWriter)(cashpkt.CashItemGachaponFailedBody())(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce surprise open failure to character [%d].", e.CharacterId)
				return err
			}
			return nil
		})
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-channel && go build ./... && go test -race ./... 2>&1 | tail -20
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/
git commit -m "feat(task-207): announce surprise open result from atlas-channel"
```

---

### Task 16: Tenant socket-config template routing

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_79_1.json` (handler only)
- Modify: `template_gms_83_1.json`, `template_gms_84_1.json`, `template_gms_87_1.json`, `template_gms_92_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json` (handler + writer)

**Interfaces:**
- Consumes: `CashItemGachaponHandle` (Task 1) and `CashItemGachaponResult` (Task 2) — the names in the JSON must match those Go constants **exactly**.
- Produces: runtime routing. No Go symbols.

**Background:** Seven handler entries and six writer entries. v79 gets the **handler only** — the user-confirmed decision routes v79 serverbound so the grant lands in the locker, but v79's binary has no result handler, so there is no writer to route (`design.md` §5).

| template | handler `opCode` | writer `opCode` | SUCCESS | FAILED |
|---|---|---|---|---|
| `template_gms_79_1.json` | `0x9F` | — | — | — |
| `template_gms_83_1.json` | `0xA1` | `0x14D` | 229 | 228 |
| `template_gms_84_1.json` | `0xA5` | `0x154` | 238 | 237 |
| `template_gms_87_1.json` | `0xA9` | `0x15E` | 244 | 243 |
| `template_gms_92_1.json` | `0xB6` | `0x180` | 190 | 189 |
| `template_gms_95_1.json` | `0xB9` | `0x188` | 193 | 192 |
| `template_jms_185_1.json` | `0xA7` | `0x16D` | 235 | 234 |

`template_gms_12_1.json`, `template_gms_48_1.json`, `template_gms_61_1.json`, `template_gms_72_1.json` get **nothing** — those versions are `n-a`.

Three failure modes this task must avoid, all previously observed in this repo:
1. A handler whose `validator` is empty is **silently dropped** at load. `LoggedInValidator` matches every sibling cash-shop handler — confirm with `grep -n 'CashShopCheckWalletHandle' -A2 -B2` in one template before writing.
2. A writer without an `fname` is rejected by the seed loader.
3. Entries must go at their **sorted `opCode` position**, never adjacent to a semantically-related entry — `tools/template-opcode-order-guard.sh` enforces strictly ascending order for both arrays.

- [ ] **Step 1: Confirm the entry shapes against an existing sibling**

```bash
python3 -c "
import json
d=json.load(open('services/atlas-configurations/seed-data/templates/template_gms_83_1.json'))
def walk(o,p=''):
    if isinstance(o,dict):
        for k,v in o.items(): yield from walk(v,p+'/'+k)
    elif isinstance(o,list):
        for i,v in enumerate(o): yield from walk(v,p+'/[]')
    else: yield p,o
import itertools
print([e for e in json.dumps(d).split('},') if 'CashShopCheckWallet' in e][:1])
"
```

Or more simply:

```bash
grep -n 'CashShopCheckWalletHandle' -B3 -A4 services/atlas-configurations/seed-data/templates/template_gms_83_1.json
grep -n 'CashQueryResult' -B3 -A8 services/atlas-configurations/seed-data/templates/template_gms_83_1.json
```

Copy the exact key set and ordering those show. The shapes below are the design's; **the sibling entries are the authority** if they differ.

Handler:

```json
{ "opCode": "0xA1", "validator": "LoggedInValidator",
  "handler": "CashItemGachaponHandle",
  "fname": "CUICashItemGachapon::OnButtonClicked", "services": ["channel"] }
```

Writer:

```json
{ "opCode": "0x14D", "writer": "CashItemGachaponResult",
  "fname": "CCashShop::OnCashItemGachaponResult",
  "options": { "operations": { "SUCCESS": 229, "FAILED": 228 } },
  "services": ["channel"] }
```

- [ ] **Step 2: Insert all thirteen entries at their sorted positions**

Edit each of the seven templates with the Edit tool, one file at a time — **not** a shell patch loop, which has produced garbled output in this repo. For each file, locate the correct insertion point by finding the adjacent opcodes:

```bash
grep -n '"opCode": "0xA0"\|"opCode": "0xA1"\|"opCode": "0xA2"' services/atlas-configurations/seed-data/templates/template_gms_83_1.json
```

Preserve the file's existing line endings and indentation exactly.

- [ ] **Step 3: Run the template guards**

```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
```

Expected: all three exit 0. If the order guard reports a violation, the entry went in the wrong place — move it, do not renumber anything.

- [ ] **Step 4: Verify the JSON still parses**

```bash
for f in services/atlas-configurations/seed-data/templates/template_gms_79_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_84_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_87_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_92_1.json \
         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
         services/atlas-configurations/seed-data/templates/template_jms_185_1.json; do
  python3 -m json.tool "$f" > /dev/null && echo "OK $f" || echo "BAD $f"
done
```

Expected: seven `OK` lines.

- [ ] **Step 5: Confirm the counts**

```bash
grep -c 'CashItemGachaponHandle' services/atlas-configurations/seed-data/templates/*.json | grep -v ':0'
grep -c '"CashItemGachaponResult"' services/atlas-configurations/seed-data/templates/*.json | grep -v ':0'
```

Expected: exactly 7 files with a handler (79, 83, 84, 87, 92, 95, jms) and exactly 6 with a writer (all but 79). If `template_gms_12_1.json`, `48`, `61`, or `72` appears in either list, remove that entry — those columns are `n-a`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(task-207): route CashItemGachapon handler and result writer in seven templates"
```

- [ ] **Step 7: Note the live-tenant follow-up**

New opcodes present in the seed templates but absent from a live tenant's socket configuration mean the handler exists in code and never fires at runtime — a recurring failure in this repo. Add a line to `docs/tasks/task-207-cash-shop-surprise/context.md` under "Rollout" recording that live tenant socket configs must be reseeded or PATCHed with these entries before the feature can be exercised on a running environment. This is a deployment step, not a code change.

---

### Task 17: `atlas-ui` — widen the pool kind

**Files:**
- Modify: `services/atlas-ui/src/types/models/reward-pool.ts`
- Modify: `services/atlas-ui/src/types/models/reward-pool-item.ts`
- Modify: `services/atlas-ui/src/components/features/reward-pools/KindBadge.tsx`
- Modify: `services/atlas-ui/src/lib/schemas/reward-pools.schema.ts`
- Test: `services/atlas-ui/src/components/features/reward-pools/__tests__/` (extend the existing suites)

**Interfaces:**
- Consumes: the widened `kind` union and the `commodityId` field from Tasks 4–6 (server side).
- Produces:
  - `type RewardPoolKind = "gachapon" | "incubator" | "cash-surprise"`
  - `RewardPoolItemAttributes.commodityId: number`
  - `export const cashSurprisePoolSchema` / `CashSurprisePoolFormData`
  - `export const cashSurpriseItemSchema` / `CashSurpriseItemFormData`

**Background:** `KindBadge` is currently a binary ternary with **no default branch** — a third kind would silently render as "Gachapon". Turn it into a lookup so an unhandled kind is a type error, not a wrong label.

- [ ] **Step 1: Write the failing test**

Extend the existing badge test in `services/atlas-ui/src/components/features/reward-pools/__tests__/` (find it: `ls services/atlas-ui/src/components/features/reward-pools/__tests__/`):

```tsx
it("renders a distinct badge for cash-surprise pools", () => {
  render(<KindBadge kind="cash-surprise" />);
  expect(screen.getByText("Cash Surprise")).toBeInTheDocument();
});

it("still renders the existing kinds unchanged", () => {
  const { rerender } = render(<KindBadge kind="gachapon" />);
  expect(screen.getByText("Gachapon")).toBeInTheDocument();
  rerender(<KindBadge kind="incubator" />);
  expect(screen.getByText("Incubator")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui && npx vitest run src/components/features/reward-pools/__tests__ 2>&1 | tail -20
```

(If `npm`/`npx` fails, the repo needs nvm 22 — run `nvm use 22` first.)

Expected: FAIL — "Cash Surprise" not found (the ternary falls through to "Gachapon").

- [ ] **Step 3: Widen the types**

`src/types/models/reward-pool.ts`:

```ts
export type RewardPoolKind = "gachapon" | "incubator" | "cash-surprise";
```

`src/types/models/reward-pool-item.ts` — add to `RewardPoolItemAttributes`:

```ts
  commodityId: number; // cash shop commodity (serial number); 0 on gachapon/incubator items
```

- [ ] **Step 4: Rewrite `KindBadge` as a total lookup**

```tsx
import { Badge } from "@/components/ui/badge";
import type { RewardPoolKind } from "@/types/models/reward-pool";

/**
 * Pool-kind badge. A Record keyed by RewardPoolKind rather than a ternary:
 * the previous binary ternary had no default branch, so a new kind rendered
 * silently as "Gachapon". With the Record, adding a kind without a badge is
 * a type error.
 *
 * The amber utility classes match the existing amber-badge convention used
 * across the codebase — keep them here rather than inventing a new token.
 */
const BADGES: Record<RewardPoolKind, React.ReactElement> = {
  incubator: (
    <Badge className="bg-amber-500/15 text-amber-600 dark:text-amber-400 border-transparent">
      Incubator
    </Badge>
  ),
  "cash-surprise": (
    <Badge className="bg-violet-500/15 text-violet-600 dark:text-violet-400 border-transparent">
      Cash Surprise
    </Badge>
  ),
  gachapon: <Badge variant="secondary">Gachapon</Badge>,
};

export function KindBadge({ kind }: { kind: RewardPoolKind }) {
  return BADGES[kind];
}
```

Check that `violet-*` is already used somewhere in the codebase before adopting it:

```bash
grep -rn 'violet-500/15\|violet-600' services/atlas-ui/src/ | head -3
```

If it is not, use another colour that **is** already in use (`grep -rn '500/15' services/atlas-ui/src/components/ | head`) rather than introducing a new one.

- [ ] **Step 5: Add the two schemas**

Append to `src/lib/schemas/reward-pools.schema.ts`:

```ts
// A cash-surprise pool's id IS the box template id, exactly as an incubator
// pool's id is the egg item id — there is no separate column. npcIds is
// unused for this kind (the box is opened from the Cash Shop, not from an
// NPC), so the form omits the field entirely rather than hiding it.
export const cashSurprisePoolSchema = z.object({
  boxItemId: z.number().int().positive("Box item id is required"),
  name: z.string().min(1, "Name is required"),
});
export type CashSurprisePoolFormData = z.infer<typeof cashSurprisePoolSchema>;

// A cash-surprise entry awards a cash shop COMMODITY (serial number), not a
// raw item id: the commodity catalog owns the reward's itemId, count and
// period, so rolling a commodity guarantees a self-consistent locker entry.
// itemId stays on the entry for operator display only.
export const cashSurpriseItemSchema = z.object({
  itemId: z.number().int().positive("Item id is required"),
  quantity: z.number().int().positive(),
  weight: z.number().int().positive("Weight must be at least 1"),
  commodityId: z.number().int().positive("Commodity id is required"),
});
export type CashSurpriseItemFormData = z.infer<typeof cashSurpriseItemSchema>;
```

Also update the file's header comment, which currently reads "for the gachapon and incubator reward-pool dialogs".

- [ ] **Step 6: Run tests and the build**

```bash
cd services/atlas-ui && npx vitest run src/components/features/reward-pools 2>&1 | tail -20 && npm run build 2>&1 | tail -20
```

Expected: PASS and a clean build. The build type-checks the tests too, so a type error in a spec file fails here.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/
git commit -m "feat(task-207): widen reward-pool kind to cash-surprise in atlas-ui"
```

---

### Task 18: `atlas-ui` — the cash-surprise pool and item forms

**Files:**
- Modify: `services/atlas-ui/src/components/features/reward-pools/PoolFormDialog.tsx`
- Modify: `services/atlas-ui/src/components/features/reward-pools/PoolItemDialog.tsx`
- Modify: `services/atlas-ui/src/components/features/reward-pools/PoolItemsTable.tsx` (show the commodity column)
- Test: the existing `__tests__` suites for both dialogs

**Interfaces:**
- Consumes: `cashSurprisePoolSchema`, `cashSurpriseItemSchema`, the widened `RewardPoolKind` (Task 17).
- Produces: no new exports; behaviour only.

**Background:** `PoolFormDialog` already splits per kind — a `RadioGroup` selects the kind and each kind renders its **own** form with its own `useForm`. Add a third radio option and a third form mirroring the incubator branch exactly: `boxItemId` (→ pool slug, as `eggItemId` already is via `createPool.mutateAsync({ id: String(values.eggItemId), … })`) and `name`. **No `npcIds` field** — FR-7.2 is satisfied by the per-kind form split, not by conditionally hiding a field.

The submit payload mirrors `submitIncubator`, with `npcIds: []`:

```ts
const attributes = {
  name: values.name,
  kind: "cash-surprise" as const,
  npcIds: [],
  commonWeight: 0,
  uncommonWeight: 0,
  rareWeight: 0,
};
```

`PoolItemDialog` currently computes `const weighted = kind === "incubator"`. `cash-surprise` is **also** weighted and additionally requires `commodityId`. Its `kind` prop type is the literal union `"gachapon" | "incubator" | "global"` and must gain `"cash-surprise"`.

- [ ] **Step 1: Write the failing tests**

Extend the two dialog suites:

```tsx
// PoolFormDialog
it("offers a cash-surprise kind and submits the box item id as the pool id", async () => {
  // render in create mode, select the "Cash Surprise" radio,
  // fill boxItemId 5222000 and name "Surprise Box", submit
  // assert createPool called with { id: "5222000", attributes: { kind: "cash-surprise", npcIds: [], ... } }
});

it("does not render an NPC Ids field for cash-surprise pools", () => {
  // render, select "Cash Surprise"
  // assert queryByLabelText(/NPC Ids/i) is null
});

// PoolItemDialog
it("requires a commodity id for cash-surprise entries", async () => {
  // render with kind="cash-surprise", fill itemId/quantity/weight but not commodityId, submit
  // assert the "Commodity id is required" message appears and no mutation fired
});

it("submits commodityId alongside weight for cash-surprise entries", async () => {
  // fill all four fields, submit
  // assert createItem called with { itemId, quantity, tier: "common", weight, commodityId }
});

it("does not send a commodityId for incubator entries", async () => {
  // regression control
});
```

Write each out fully against the render/mocking helpers the existing specs use.

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-ui && npx vitest run src/components/features/reward-pools 2>&1 | tail -25
```

Expected: FAIL.

- [ ] **Step 3: Add the third branch to `PoolFormDialog`**

- add `const cashSurpriseForm = useForm<CashSurprisePoolFormData>({ resolver: zodResolver(cashSurprisePoolSchema), defaultValues: cashSurpriseDefaults })`, with `cashSurpriseDefaults` built the same way `incubatorDefaults` is (`DefaultValues<T>` so create mode can leave the numeric field undefined under `exactOptionalPropertyTypes`; edit mode reads `boxItemId: Number(pool.id)`);
- add `cashSurpriseForm.reset(cashSurpriseDefaults)` to the existing `useEffect`;
- add `submitCashSurprise` mirroring `submitIncubator` with the payload above;
- add a third `RadioGroupItem value="cash-surprise" id="kind-cash-surprise"` with label `Cash Surprise`;
- convert the two-way `kind === "gachapon" ? … : …` render into a three-way branch. Since the existing code is a ternary, the cleanest edit is to extract each form into a local variable and select with a `switch`, or nest a second ternary — match whichever reads better against the surrounding style; do not restructure the component beyond what the third branch needs.

Add, next to the box item id input, a short helper line — FR-4.5's recursion risk is honoured by configuration, not blocked in code, so the operator has to be told:

```tsx
<p className="text-sm text-muted-foreground">
  A pool that awards a Surprise box will produce an endless box. This is
  allowed; check your entries.
</p>
```

- [ ] **Step 4: Extend `PoolItemDialog`**

- widen the `kind` prop to `"gachapon" | "incubator" | "cash-surprise" | "global"`;
- `const weighted = kind === "incubator" || kind === "cash-surprise";`
- `const needsCommodity = kind === "cash-surprise";`
- `const schema = needsCommodity ? cashSurpriseItemSchema : weighted ? weightItemSchema : tierItemSchema;`
- widen the `useForm` generic and `defaultValues` union to include `CashSurpriseItemFormData`;
- render a `commodityId` number input when `needsCommodity`, with the same error-message pattern the weight input uses;
- in `onSubmit`, add `commodityId` to the weighted payload when `needsCommodity` (and **only** then — the incubator regression test pins this).

Locate every `PoolItemDialog` call site and pass the real pool kind through:

```bash
grep -rn '<PoolItemDialog' services/atlas-ui/src/
```

- [ ] **Step 5: Show the commodity in `PoolItemsTable`**

Add a `Commodity` column rendered only for `cash-surprise` pools (FR-7.3, "display the resolved item for operator sanity-checking"). The table already receives the pool or its kind — read the file and follow its existing conditional-column pattern if it has one; if it has none, gate the column with a simple `kind === "cash-surprise" &&` on both the header and the cell.

- [ ] **Step 6: Run tests and the build**

```bash
cd services/atlas-ui && npx vitest run 2>&1 | tail -20 && npm run build 2>&1 | tail -20
```

Expected: PASS and a clean build.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/
git commit -m "feat(task-207): add cash-surprise pool and item forms to atlas-ui"
```

---

### Task 19: Coverage matrix — verify and prove every cell

**Files:**
- Modify: `docs/packets/audits/status.json`, `docs/packets/audits/STATUS.md` (regenerated, never hand-edited)
- Modify: `docs/packets/feature-families.yaml`, `docs/packets/feature-na-evidence.yaml`
- Create: evidence records and fixtures under `docs/packets/audits/gms_v*/` and `jms_v185/` as the playbook directs

**Interfaces:**
- Consumes: the codecs (Tasks 1–2), registry entries (Task 3), template routing (Task 16).
- Produces: promoted matrix cells. No Go symbols.

**Background:** This task is **driven by the playbook, not re-specified here.** The single-cell procedure is `docs/packets/audits/VERIFYING_A_PACKET.md`, entered via the `/verify-packet` command and the `packet-verifier` agent (pin those subagents to Sonnet or Haiku per the model-cost rule — do not use an expensive model for verification fan-out).

The three matrix rows in play (`docs/packets/audits/STATUS.md`):
- `:723` `CASH_ITEM_GACHAPON_BUTTON` / `CUICashItemGachapon::OnButtonClicked` (serverbound)
- `:409` `CASHSHOP_CASH_ITEM_GACHAPON_RESULT` / `CCashShop::OnCashItemGachaponResult` (clientbound)
- `:486` `CASHSHOP_CASH_GACHAPON_OPEN_RESULT` — the **alias** row, same fname, the second of the two consecutive opcodes routed to the same handler on v84 (`0x155`), v87 (`0x15F`), v92 (`0x181`), v95 (`0x189`). It is **not** aliased on v83 or jms.

Target end-state per column:

| Version | Serverbound (`:723`) | Clientbound (`:409`) | Alias row (`:486`) |
|---|---|---|---|
| gms_v48 | `n-a` + proof | `n-a` + proof | already `⬜` |
| gms_v61 | `n-a` + proof | `n-a` + proof | already `⬜` |
| gms_v72 | `n-a` + proof | `n-a` + proof | already `⬜` |
| gms_v79 | **verified** (routed) | `n-a` + proof | already `⬜` |
| gms_v83 | verified | verified | **`n-a`** — v83's dispatcher has no `0x14E` case (`0x478e2b`); currently mis-listed as `0x14E ❌` |
| gms_v84 | verified | verified | verify or leave — see Step 5 |
| gms_v87 | verified | verified | verify or leave |
| gms_v92 | verified | verified | verify or leave |
| gms_v95 | verified | verified | verify or leave |
| jms_v185 | verified | verified | already `⬜` |

- [ ] **Step 1: Read the playbook**

```bash
sed -n '1,80p' docs/packets/audits/VERIFYING_A_PACKET.md
sed -n '1,60p' docs/packets/PROCESS.md
```

Do not proceed from this plan's summary — the playbook is the source of truth for the artifact set (fixture + evidence record + audit report) each cell needs.

- [ ] **Step 2: Verify the seven serverbound cells**

One `/verify-packet` run per cell: `cash/serverbound/CashItemGachaponButton` × {gms_v79, gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, jms_v185}. The IDB addresses are already derived (`design.md` §1.1) — v79 `0x8efda6`, v83 `0x99a9a7`, v84 `0x9db82d`, v87 `0xa215f6`, v92 `0x944110`, v95 `0x96aa40`, jms `0xa6e309`. Two functions were already renamed during the design pass (v83 `sub_99A9A7`, v92 `sub_944110`); name any others you touch.

Batch per IDB, not per version-order, so each IDA session is opened once.

- [ ] **Step 3: Verify the six clientbound cells**

`cash/clientbound/CashItemGachaponResult` × {gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, jms_v185}. Handler addresses: v83 `0x478e2b`, v84 `0x47bf59`, v87 `0x4844a4`, v92 `0x495770`, v95 `0x4997e0`, jms `0x48b21d`.

Note the fname carries a `#SUCCESS` / `#FAILED` suffix in the codec markers. Follow the playbook's convention for mode-suffixed fnames — check how `shop_operation_result_gachapon.go`'s `#GACHAPON_OPEN_SUCCESS` markers are recorded and mirror that.

- [ ] **Step 4: Record the seven `n-a` proofs**

- **v48/v61/v72, both directions (6 cells).** Three independent proofs from `design.md` §1.1/§1.6: no `CUICashItemGachapon` anywhere in the binary; the standalone opcode's handler is a 3-line "read one byte into a `CWvsContext` flag" stub (v48 `0x4536b9`, v61 `0x46128a`, v72 `0x470d24`) and not a gachapon result at all; item `5222000` returns 404 from `atlas-data` on those tenants. Include the **mandatory sibling cross-check** the playbook requires — the clientbound side being a flag stub is exactly that check, and it must be written into the evidence text, not just asserted.
- **v79 clientbound (1 cell).** `CCashShop::OnPacket` (`0x471da6`) enumerates 301–309 with no gachapon case; `CCashShop::OnCashItemResult` (`0x4720ed`) enumerates 47 modes with no gachapon arm; `CUICashItemGachapon::Update` (`0x8ef7ce`) renders the reward from `m_nSelectedItemID` and nothing in the binary writes it. **Record the scope caveat verbatim:** the two cash-shop dispatch tables are the complete set of `CCashShop` clientbound entry points, but a result arriving through a wholly unrelated router was not exhaustively excluded.

  This cell is `n-a` while its same-family serverbound sibling is `verified` on the same version — precisely the family-inconsistency the `matrix --check` gate catches. It **requires** an entry in `docs/packets/feature-na-evidence.yaml` with non-empty evidence text, and the family must be declared in `docs/packets/feature-families.yaml` if it is not already:

  ```bash
  grep -n 'CASH_ITEM_GACHAPON\|CASHSHOP_CASH_ITEM' docs/packets/feature-families.yaml
  ```

  If absent, add a family grouping `CASH_ITEM_GACHAPON_BUTTON` and `CASHSHOP_CASH_ITEM_GACHAPON_RESULT`, following the file's existing entry shape.

- [ ] **Step 5: Fix the v83 alias row and decide the rest of row `:486`**

`CASHSHOP_CASH_GACHAPON_OPEN_RESULT` × gms_v83 currently reads `0x14E ❌`. v83's dispatcher has **no** `0x14E` case (`0x478e2b`) — the cell is `n-a`, not merely unverified. Correct the registry entry that produces that `0x14E` (find it: `grep -n '0x14E\|opcode: 334' docs/packets/registry/gms_v83.yaml`) and record the `n-a`.

For v84/v87/v92/v95 the alias row is a genuine second opcode routed to the same handler, decoding identically. Verifying those four cells is in scope if the fixtures transfer trivially (same codec, same read order, different opcode). If the playbook's per-cell artifact requirements make them a materially separate effort, leave them at `❌` — but say so explicitly in the commit message and in `context.md`, since "left at ❌" is a coverage decision, not an omission. Do **not** silently skip them.

- [ ] **Step 6: Regenerate and check the matrix**

```bash
cd tools/packet-audit && go run . matrix && go run . matrix --check 2>&1 | tail -40
```

Compare against the baseline captured in Task 3 Step 5:

```bash
diff <(cd tools/packet-audit && go run . matrix --check 2>&1 | tail -40) \
     "$SCRATCH/matrix-baseline.txt"
```

Expected: promotions only. **No cell may degrade.** Also run whichever of `dispatcher-lint`, `fname-doc`, and `operations --check` the playbook names as gates, and show exit 0 for each.

- [ ] **Step 7: Commit**

```bash
git add docs/packets/
git commit -m "docs(task-207): promote CashItemGachapon matrix cells and record n-a proofs"
```

---

### Task 20: Full verification gauntlet and documentation

**Files:**
- Modify: `services/atlas-cashshop/docs/` (domain + any kafka/rest doc), `services/atlas-channel/docs/` if it documents handlers, `docs/research/missing-features/economy-and-trade.md:29`
- Create: `docs/tasks/task-207-cash-shop-surprise/audit.md` (written by the reviewer agents)

**Interfaces:**
- Consumes: everything.
- Produces: a branch that is genuinely ready for a PR.

- [ ] **Step 1: Per-module Go verification**

```bash
for m in libs/atlas-packet services/atlas-cashshop services/atlas-channel services/atlas-reward-pools; do
  echo "=== $m ==="
  (cd "$m" && go vet ./... && go test -race ./... 2>&1 | tail -5)
done
```

Expected: clean vet and PASS in all four. Add `libs/atlas-rest` to the list if Task 9 Step 3 modified it.

- [ ] **Step 2: Repo-root guards**

```bash
tools/redis-key-guard.sh && echo "redis OK"
tools/goroutine-guard.sh && echo "goroutine OK"
tools/skill-job-id-guard.sh && echo "skill-job OK"
tools/buff-duration-guard.sh && echo "buff OK"
tools/template-opcode-order-guard.sh && echo "opcode-order OK"
tools/template-duplicate-binding-guard.sh && echo "dup-binding OK"
tools/template-movement-types-guard.sh && echo "movement OK"
```

Expected: seven `OK` lines. `tools/service-registration-guard.sh` is only required if `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work`, or `tools/db-bootstrap.sh` changed — run it if Task 9 Step 4 touched `deploy/`.

- [ ] **Step 3: Lint**

```bash
tools/lint.sh          # fix mode — rewrites files in place
tools/lint.sh --check  # must exit 0
```

`--check` false-fails without nvm; if it does, `nvm use 22` and re-run. Commit any formatting the fix pass rewrote.

- [ ] **Step 4: Docker bake — mandatory, not optional**

Every service whose `go.mod` was touched. `go build` against the workspace will **not** catch a missing `COPY libs/...` line in the shared Dockerfile:

```bash
docker buildx bake atlas-cashshop atlas-channel atlas-reward-pools 2>&1 | tail -20
```

If Task 9 Step 3 added anything to `libs/atlas-rest`, bake **every** service that imports it:

```bash
grep -rln 'libs/atlas-rest' services/*/go.mod
```

Expected: all targets build.

- [ ] **Step 5: atlas-ui**

```bash
cd services/atlas-ui && npx vitest run 2>&1 | tail -10 && npm run build 2>&1 | tail -10
```

Expected: PASS and clean build.

- [ ] **Step 6: Update the service documentation**

- `services/atlas-cashshop/docs/` — the new `surprise` domain (open sequence, the openings ledger, the capacity rule), the `OPEN_SURPRISE` command, the two status events, and the reward-pools dependency. Find the files first: `ls services/atlas-cashshop/docs/`.
- `services/atlas-reward-pools/docs/` — already updated in Task 6 Step 5; confirm it is still accurate.
- `docs/research/missing-features/economy-and-trade.md:29` — the backlog entry that was the only mention of `5222000` in the repo. Mark it delivered by task-207 rather than deleting the line.

- [ ] **Step 7: Code review before the PR**

Mandatory — do not skip even though the plan is complete. Dispatch in parallel, pinned to Sonnet or Haiku:

- `plan-adherence-reviewer` — every task in this plan actually implemented
- `backend-guidelines-reviewer` — DOM-* over the four Go modules
- `frontend-guidelines-reviewer` — FE-* over the changed `atlas-ui` TS
- `packet-completeness-critic` — CHANGED-BUT-UNCLAIMED / CLAIMED-BUT-UNVERIFIED against the task's coverage manifest

All four must write into `docs/tasks/task-207-cash-shop-surprise/` inside **this worktree**, never the main repo. Verify the tree is clean after they run:

```bash
git rev-parse --show-toplevel   # must end with /.worktrees/task-207-cash-shop-surprise
git status --short
```

- [ ] **Step 8: Live acceptance**

PRD §10's behaviour checklist, run against a live tenant. Note that live tenant socket configs must carry the new opcodes first (Task 16 Step 7) or the handler never fires.

- v83 tenant (primary target): open a box with a stack of 3, then the last one; confirm the reward appears with no relog and the box row disappears at 0.
- One v84+ tenant, exercising the **standalone opcode** — `design.md` §1.4 removed the dispatcher-arm path PRD §10's second bullet assumed.
- Full locker → client-visible error, box intact.
- Empty pool → client-visible error, box intact.
- Forged request naming another account's asset → rejected and logged, no state change.

Record the results in `docs/tasks/task-207-cash-shop-surprise/context.md`. A check that could not be run is recorded as not run, never as passed.

- [ ] **Step 9: Commit**

```bash
git add docs/ services/
git commit -m "docs(task-207): document cash shop surprise and record verification results"
```
