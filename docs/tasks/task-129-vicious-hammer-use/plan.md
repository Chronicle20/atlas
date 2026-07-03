# Vicious Hammer Use Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the Vicious Hammer (item 5570000) end-to-end: the retail two-phase gauge protocol (cash-item-use → open-arm → `ITEM_UPGRADE_UPDATE` confirm → terminal result), atomic consume + `slots += 1` / `hammersApplied += 1` in atlas-consumables, and per-version packet verification.

**Architecture:** Design B (two-phase, retail-faithful) per `design.md` §4. atlas-channel handles both serverbound packets and writes the mode-prefixed `VICIOUS_HAMMER` clientbound dispatcher (discrete per-mode structs, config-resolved mode bytes). atlas-consumables owns validation + the reserve→consume→ChangeStat atomic flow (scroll-flow precedent). atlas-inventory is untouched (`MODIFY_EQUIPMENT` already persists both fields). State between the two packets is kept in the client via a round-trip token that packs both slots — no server-side pending state.

**Tech Stack:** Go, Kafka (segmentio), atlas-socket request/response readers/writers, packet-audit tooling, ida-pro-mcp (verification tasks only).

## Global Constraints

- Mode bytes are NEVER hard-coded in struct constructors or body funcs — resolved via `WithResolvedCode("operations", KEY, func(mode byte)…)` (dispatcher-lint INV-2/INV-3; `docs/packets/DISPATCHER_FAMILY.md`). Literal mode bytes are allowed ONLY in tests.
- `VICIOUS_HAMMER` operations table (all GMS versions, IDA-verified version-stable v83 `sub_82B2C3` / v95 `CUIItemUpgrade::ShowResult 0x7bec20`): `OPEN = 0`, `SUCCESS = 61`, `FAILURE = 62`. OPEN is server-chosen (client accepts any byte ∉ {61, 62} and echoes it) — design OQ-1 resolved as 0.
- Round-trip token packing (server-chosen, opaque to client): `token = uint32(uint16(hammerSlot))<<16 | uint32(uint16(equipSlot))`. Slots are signed int16 (negative = equipped).
- Hammer cap: `hammersApplied < 2` (IDA-verified twice: error code 2 + the `2 - count` display).
- Client failure codes (IDA, v83 `sub_82B2C3`): `1` = not upgradable, `2` = cap reached, `3` = Horntail Necklace, other = "Unknown error %d". Server uses `0` for internal errors.
- Horntail Necklace item id = **1122000** (WZ-verified: `String.wz/Eqp.img.xml`, GMS 83.1 dump under the repo `tmp/<uuid>/GMS/83.1/` tree). It has `tuc=3`, so the exclusion must be an explicit id check, not derived from slots.
- Eligibility predicate (error 1): equip-data `slots == 0` OR `cash == true` (atlas-data `equipment` reader fields `tuc` / `cash`).
- Serverbound opcodes (registry-verified): v83 `0x104`, v84 `0x104`, v87 `0x112`, v95 `0x128`, jms `0x114`. Clientbound `VICIOUS_HAMMER`: v83 `0x162`, v84 `0x169`, v87 `0x177`, v95 `0x1A9`, jms absent.
- Every new handler entry in seed templates carries `"validator": "LoggedInValidator"` (validator-less entries are silently dropped).
- jms is OUT OF SCOPE (design OQ-2, resolved during planning): no `VICIOUS_HAMMER` clientbound registry row AND zero `CUIItemUpgrade` functions in `docs/packets/ida-exports/gms_jms_185.json` — the result packet cannot be sent, so the serverbound op is not routed either. Document, don't route.
- v92 has NO registry, NO IDB, NO export, and its seed template (`template_gms_92_1.json`) is a login-only stub (37 handlers — `CharacterCashItemUseHandle` is not routed; no `ViciousHammer` writer). No hammer entries can attach to it. Document the disposition; add nothing.
- Never overwrite a committed IDA export — surgical splice only (`VERIFYING_A_PACKET.md` §10).
- `ExecuteTransaction` is a no-op project-wide — atomicity comes from the reserve→consume-callback ordering + compensating failure event, exactly like `ConsumeScroll`.
- Tests use the project Builder pattern; no `*_testhelpers.go`.
- Gates before "done": `go test -race ./...`, `go vet ./...`, `go build ./...` per changed module; `docker buildx bake atlas-channel atlas-consumables`; `tools/redis-key-guard.sh`; `go run ./tools/packet-audit dispatcher-lint`, `matrix --check` (no NEW problems), `fname-doc --check`, `operations --check`.
- All `go` commands run from the module directory (e.g. `cd libs/atlas-packet`); all `packet-audit` commands run from the worktree root.

---

### Task 1: atlas-constants — Vicious Hammer classification constant

**Files:**
- Modify: `libs/atlas-constants/item/constants.go`
- Test: `libs/atlas-constants/item/constants_test.go`

**Interfaces:**
- Produces: `item.ClassificationViciousHammer = item.Classification(557)` — used by Tasks 8 and 10 to identify the hammer by classification (covers all `0557.img` ids) instead of a raw `557` literal.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-constants/item/constants_test.go` (match the file's existing test style):

```go
func TestViciousHammerClassification(t *testing.T) {
	if GetClassification(Id(5570000)) != ClassificationViciousHammer {
		t.Errorf("GetClassification(5570000) = %d, want ClassificationViciousHammer (557)", GetClassification(Id(5570000)))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./item/ -run TestViciousHammerClassification -v`
Expected: FAIL — `undefined: ClassificationViciousHammer`

- [ ] **Step 3: Add the constant**

In `libs/atlas-constants/item/constants.go`, inside the existing classification const block (near the other 5xx cash classifications — keep numeric order):

```go
	ClassificationViciousHammer = Classification(557)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-constants && go test -race ./item/ -v -run TestViciousHammerClassification`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-constants/item/constants.go libs/atlas-constants/item/constants_test.go
git commit -m "feat(constants): add ClassificationViciousHammer (557)"
```

---

### Task 2: atlas-packet — serverbound `ItemUseViciousHammer` tail codec (Packet A)

The hammer arm of `CASH_ITEM_USE`: after the shared `ItemUse` prefix, the `CUIItemUpgrade` dialog appends three int32s (IDA v83 `CUIItemUpgrade::OnButtonClicked sub_82AED3` / v95 `0x7c0ca0`): `Encode4(m_nItemTI)`, `Encode4(m_nSlotPosition)`, `Encode4(update_time)`. No version gate — the append sequence is identical in v83 and v95 (design §2.1 step 3). This also resolves the stale `// TODO for v83 there is a trailing updateTime` in the channel handler (removed in Task 10).

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/item_use_vicious_hammer.go`
- Test: `libs/atlas-packet/cash/serverbound/item_use_vicious_hammer_test.go`

**Interfaces:**
- Produces: `cashsb.NewItemUseViciousHammer() *ItemUseViciousHammer` with getters `ItemTI() uint32`, `SlotPosition() int32`, `UpdateTime() uint32`. Consumed by Task 10 (channel hammer arm).

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/cash/serverbound/item_use_vicious_hammer_test.go`:

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte layout (IDA, design §2.1 step 3 — identical appends in v83
// CUIItemUpgrade::OnButtonClicked sub_82AED3 and v95 0x7c0ca0):
//   Encode4(itemTI) + Encode4(slotPosition) + Encode4(updateTime) = 12 bytes,
// appended AFTER the shared ItemUse prefix. No version gate.
func TestItemUseViciousHammerByteOutput(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseViciousHammer{itemTI: 1, slotPosition: -5, updateTime: 0xDEADBEEF}
			got := input.Encode(nil, ctx)(nil)
			if len(got) != 12 {
				t.Errorf("byte count: got %d, want 12", len(got))
			}
		})
	}
}

func TestItemUseViciousHammerRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseViciousHammer{itemTI: 1, slotPosition: -5, updateTime: 12345}
			output := ItemUseViciousHammer{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ItemTI() != input.ItemTI() {
				t.Errorf("itemTI: got %d, want %d", output.ItemTI(), input.ItemTI())
			}
			if output.SlotPosition() != input.SlotPosition() {
				t.Errorf("slotPosition: got %d, want %d", output.SlotPosition(), input.SlotPosition())
			}
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %d, want %d", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -run TestItemUseViciousHammer -v`
Expected: FAIL — `undefined: ItemUseViciousHammer`

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/cash/serverbound/item_use_vicious_hammer.go` (mirror `item_use_field_effect.go`):

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseViciousHammer is the trailing body the CUIItemUpgrade dialog appends
// to the pre-built CASH_ITEM_USE packet when the Upgrade button is clicked
// (v83 CUIItemUpgrade::OnButtonClicked sub_82AED3 / v95 0x7c0ca0):
// Encode4(m_nItemTI) + Encode4(m_nSlotPosition) + Encode4(update_time).
// slotPosition is signed: negative = an equipped item, positive = a slot in
// the equip inventory. Layout is version-invariant (IDA v83 + v95).
type ItemUseViciousHammer struct {
	itemTI       uint32
	slotPosition int32
	updateTime   uint32
}

func NewItemUseViciousHammer() *ItemUseViciousHammer {
	return &ItemUseViciousHammer{}
}

func (m ItemUseViciousHammer) ItemTI() uint32       { return m.itemTI }
func (m ItemUseViciousHammer) SlotPosition() int32  { return m.slotPosition }
func (m ItemUseViciousHammer) UpdateTime() uint32   { return m.updateTime }

func (m ItemUseViciousHammer) Operation() string { return "ItemUseViciousHammer" }

func (m ItemUseViciousHammer) String() string {
	return fmt.Sprintf("itemTI [%d] slotPosition [%d] updateTime [%d]", m.itemTI, m.slotPosition, m.updateTime)
}

func (m ItemUseViciousHammer) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.itemTI)
		w.WriteInt32(m.slotPosition)
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *ItemUseViciousHammer) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.itemTI = r.ReadUint32()
		m.slotPosition = r.ReadInt32()
		m.updateTime = r.ReadUint32()
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test -race ./cash/serverbound/ -run TestItemUseViciousHammer -v`
Expected: PASS (all variants)

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/item_use_vicious_hammer.go libs/atlas-packet/cash/serverbound/item_use_vicious_hammer_test.go
git commit -m "feat(packet): ItemUseViciousHammer serverbound tail codec (CASH_ITEM_USE hammer arm)"
```

---

### Task 3: atlas-packet — serverbound `ItemUpgradeUpdate` codec (Packet B)

`ITEM_UPGRADE_UPDATE` / `CUIItemUpgrade::Update` (v83 `0x82ae28` / v95 `0x7bef50`): `Encode4(m_nReturnResult)` (echo of the open-arm mode byte, widened to int32) + `Encode4(m_nResult)` (echo of the server's round-trip token). Currently ❌ in every matrix version.

**Files:**
- Create: `libs/atlas-packet/field/serverbound/item_upgrade_update.go`
- Test: `libs/atlas-packet/field/serverbound/item_upgrade_update_test.go`

**Interfaces:**
- Produces: `fieldsb.ItemUpgradeUpdateHandle = "ItemUpgradeUpdateHandle"` (handler-map / template key) and `fieldsb.ItemUpgradeUpdate` with getters `ReturnResult() uint32`, `Result() uint32`. Consumed by Task 11 (channel handler) and Task 13 (templates).

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/field/serverbound/item_upgrade_update_test.go`:

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte layout (IDA v83 CUIItemUpgrade::Update 0x82ae28 / v95 0x7bef50):
//   Encode4(m_nReturnResult) + Encode4(m_nResult) = 8 bytes. No version gate.
// m_nReturnResult echoes the open-arm mode byte; m_nResult echoes the
// server-chosen round-trip token.
func TestItemUpgradeUpdateByteOutput(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUpgradeUpdate{returnResult: 0, result: 0x0001FFFB}
			got := input.Encode(nil, ctx)(nil)
			if len(got) != 8 {
				t.Errorf("byte count: got %d, want 8", len(got))
			}
		})
	}
}

func TestItemUpgradeUpdateRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUpgradeUpdate{returnResult: 0, result: 0x0001FFFB}
			output := ItemUpgradeUpdate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ReturnResult() != input.ReturnResult() {
				t.Errorf("returnResult: got %d, want %d", output.ReturnResult(), input.ReturnResult())
			}
			if output.Result() != input.Result() {
				t.Errorf("result: got %d, want %d", output.Result(), input.Result())
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./field/serverbound/ -run TestItemUpgradeUpdate -v`
Expected: FAIL — `undefined: ItemUpgradeUpdate`

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/field/serverbound/item_upgrade_update.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ItemUpgradeUpdateHandle = "ItemUpgradeUpdateHandle"

// ItemUpgradeUpdate — the CUIItemUpgrade gauge-confirm packet, sent once the
// dialog's gauge fills after the server armed it with the VICIOUS_HAMMER
// open-arm response. Reads (IDA v83 CUIItemUpgrade::Update 0x82ae28 /
// v95 0x7bef50): Encode4(m_nReturnResult) — the open-arm mode byte widened to
// int32 — then Encode4(m_nResult) — the server-chosen round-trip token, which
// packs hammerSlot(high int16) | equipSlot(low int16). Version-invariant.
// packet-audit:fname CUIItemUpgrade::Update
type ItemUpgradeUpdate struct {
	returnResult uint32
	result       uint32
}

func (m ItemUpgradeUpdate) ReturnResult() uint32 { return m.returnResult }
func (m ItemUpgradeUpdate) Result() uint32       { return m.result }

func (m ItemUpgradeUpdate) Operation() string { return ItemUpgradeUpdateHandle }

func (m ItemUpgradeUpdate) String() string {
	return fmt.Sprintf("returnResult [%d] result [%d]", m.returnResult, m.result)
}

func (m ItemUpgradeUpdate) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.returnResult)
		w.WriteInt(m.result)
		return w.Bytes()
	}
}

func (m *ItemUpgradeUpdate) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.returnResult = r.ReadUint32()
		m.result = r.ReadUint32()
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test -race ./field/serverbound/ -run TestItemUpgradeUpdate -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/field/serverbound/item_upgrade_update.go libs/atlas-packet/field/serverbound/item_upgrade_update_test.go
git commit -m "feat(packet): ItemUpgradeUpdate serverbound codec (CUIItemUpgrade::Update)"
```

---

### Task 4: atlas-packet — `ViciousHammer` clientbound dispatcher (discrete per-mode structs + body funcs)

Replace the empty `ViciousHammer` struct with the real mode-prefixed dispatcher per `docs/packets/DISPATCHER_FAMILY.md`: three discrete structs in ONE consolidated file, constructors taking `mode byte` first, plus fixed-key body funcs resolving the mode from the tenant `operations` table. Arm bodies (IDA v83 `sub_82B2C3` reached via `sub_82B2AD`, v95 `CUIItemUpgrade::ShowResult 0x7bec20`):

| Arm | Mode | Body after mode byte |
|---|---|---|
| Open | any ∉ {61,62} — server uses `OPEN=0` | `int32 token` + `int32 hammerCount` |
| Success | 61 | `int32 flag` (0 = success) |
| Failure | 62 | `int32 errorCode` (1/2/3, see Global Constraints) |

**Files:**
- Modify: `libs/atlas-packet/field/clientbound/vicious_hammer.go` (full rewrite — struct `ViciousHammer` and `NewViciousHammer` are retired; `ViciousHammerWriter` const survives, it is referenced by `services/atlas-channel/atlas.com/channel/main.go:765`)
- Create: `libs/atlas-packet/field/vicious_hammer_body.go`
- Test: `libs/atlas-packet/field/clientbound/vicious_hammer_test.go` (full rewrite — the old golden/roundtrip tests and their 4 `packet-audit:verify` markers for `FieldViciousHammer` are deleted; new markers are added in Tasks 14/15)

**Interfaces:**
- Produces (clientbound pkg): `NewViciousHammerOpen(mode byte, token uint32, hammerCount uint32)`, `NewViciousHammerSuccess(mode byte, flag uint32)`, `NewViciousHammerFailure(mode byte, errorCode uint32)`; all `Operation() == ViciousHammerWriter`.
- Produces (field pkg): `field.ViciousHammerOpenBody(token uint32, hammerCount uint32)`, `field.ViciousHammerSuccessBody()`, `field.ViciousHammerFailureBody(errorCode uint32)` — each returns `func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`, for `session.Announce(...)(fieldcb.ViciousHammerWriter)(body)`. Consumed by Tasks 10 and 12.

- [ ] **Step 1: Rewrite the test file (failing)**

Replace the entire contents of `libs/atlas-packet/field/clientbound/vicious_hammer_test.go`:

```go
package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Arm bodies IDA-verified in v83 (CUIItemUpgrade::OnPacket sub_82B2C3, reached
// via CField::OnItemUpgrade 0x537f8c) and v95 (CUIItemUpgrade::ShowResult
// 0x7bec20). Wire shapes are version-invariant; the mode byte is
// config-resolved in production (literal bytes below are test-only).
//
// Open  — mode(1) + token(4) + hammerCount(4) = 9 bytes
// Success — mode(1)=61 + flag(4) = 5 bytes
// Failure — mode(1)=62 + errorCode(4) = 5 bytes
func TestViciousHammerOpenByteOutput(t *testing.T) {
	// token packs hammerSlot=1 (high int16), equipSlot=-5/0xFFFB (low int16).
	input := NewViciousHammerOpen(0, 0x0001FFFB, 1)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := input.Encode(nil, ctx)(nil)
			want := []byte{0x00, 0xFB, 0xFF, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00}
			if len(got) != len(want) {
				t.Fatalf("byte count: got %d, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("byte %d: got 0x%02X, want 0x%02X", i, got[i], want[i])
				}
			}
		})
	}
}

func TestViciousHammerSuccessByteOutput(t *testing.T) {
	input := NewViciousHammerSuccess(61, 0)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := input.Encode(nil, ctx)(nil)
			want := []byte{0x3D, 0x00, 0x00, 0x00, 0x00}
			if len(got) != len(want) {
				t.Fatalf("byte count: got %d, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("byte %d: got 0x%02X, want 0x%02X", i, got[i], want[i])
				}
			}
		})
	}
}

func TestViciousHammerFailureByteOutput(t *testing.T) {
	input := NewViciousHammerFailure(62, 2)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := input.Encode(nil, ctx)(nil)
			want := []byte{0x3E, 0x02, 0x00, 0x00, 0x00}
			if len(got) != len(want) {
				t.Fatalf("byte count: got %d, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("byte %d: got 0x%02X, want 0x%02X", i, got[i], want[i])
				}
			}
		})
	}
}

func TestViciousHammerRoundTrips(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			open := NewViciousHammerOpen(0, 0x0001FFFB, 1)
			openOut := ViciousHammerOpen{}
			pt.RoundTrip(t, ctx, open.Encode, openOut.Decode, nil)
			if openOut.Token() != open.Token() || openOut.HammerCount() != open.HammerCount() {
				t.Errorf("open: got token %d count %d, want %d %d", openOut.Token(), openOut.HammerCount(), open.Token(), open.HammerCount())
			}

			success := NewViciousHammerSuccess(61, 0)
			successOut := ViciousHammerSuccess{}
			pt.RoundTrip(t, ctx, success.Encode, successOut.Decode, nil)
			if successOut.Flag() != success.Flag() {
				t.Errorf("success: got flag %d, want %d", successOut.Flag(), success.Flag())
			}

			failure := NewViciousHammerFailure(62, 3)
			failureOut := ViciousHammerFailure{}
			pt.RoundTrip(t, ctx, failure.Encode, failureOut.Decode, nil)
			if failureOut.ErrorCode() != failure.ErrorCode() {
				t.Errorf("failure: got code %d, want %d", failureOut.ErrorCode(), failure.ErrorCode())
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./field/clientbound/ -run TestViciousHammer -v`
Expected: FAIL — `undefined: NewViciousHammerOpen` (etc.)

- [ ] **Step 3: Rewrite the clientbound structs**

Replace the entire contents of `libs/atlas-packet/field/clientbound/vicious_hammer.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Discrete per-mode body codecs for the CField::OnItemUpgrade dispatcher
// (VICIOUS_HAMMER). The forwarder delegates to CUIItemUpgrade::OnPacket
// (v83 sub_82B2C3 via sub_82B2AD; v95 CUIItemUpgrade::ShowResult 0x7bec20),
// which reads Decode1(mode) and branches: 61 = success (closes the dialog,
// "Increased available upgrade by 1"), 62 = failure (closes with a notice
// keyed by the error code), any other byte = the non-terminal open/arm result
// (arms the gauge: m_nReturnResult = mode, m_nResult = token,
// m_nResultState = 1). Mode values are version-stable across v83/v95 but are
// still config-resolved from the tenant "operations" table by the body funcs
// in field/vicious_hammer_body.go — never hard-coded (DISPATCHER_FAMILY.md).
// The op is absent from the jms registry (jms VERSION-ABSENT).

// ViciousHammerWriter is the registry writer name (Operation()) shared by
// every per-mode VICIOUS_HAMMER body codec in this file.
const ViciousHammerWriter = "ViciousHammer"

// ViciousHammerOpen — the non-terminal open/arm result. Body after the mode
// byte (v83 sub_82B2C3 else-branch: Decode4 + Decode4): token (echoed back by
// the client in ITEM_UPGRADE_UPDATE) and hammerCount (the target's current
// hammersApplied; the client renders "N upgrades are left" as 2 - count).
// packet-audit:fname CField::OnItemUpgrade#Open
type ViciousHammerOpen struct {
	mode        byte
	token       uint32
	hammerCount uint32
}

func NewViciousHammerOpen(mode byte, token uint32, hammerCount uint32) ViciousHammerOpen {
	return ViciousHammerOpen{mode: mode, token: token, hammerCount: hammerCount}
}

func (m ViciousHammerOpen) Mode() byte          { return m.mode }
func (m ViciousHammerOpen) Token() uint32       { return m.token }
func (m ViciousHammerOpen) HammerCount() uint32 { return m.hammerCount }
func (m ViciousHammerOpen) Operation() string   { return ViciousHammerWriter }
func (m ViciousHammerOpen) String() string {
	return fmt.Sprintf("vicious hammer open mode [%d] token [%d] hammerCount [%d]", m.mode, m.token, m.hammerCount)
}

func (m ViciousHammerOpen) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)       // dispatcher mode byte (server-chosen, != 61/62)
		w.WriteInt(m.token)       // Decode4 -> m_nResult (round-trip token)
		w.WriteInt(m.hammerCount) // Decode4 -> current hammersApplied
		return w.Bytes()
	}
}

func (m *ViciousHammerOpen) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.token = r.ReadUint32()
		m.hammerCount = r.ReadUint32()
	}
}

// ViciousHammerSuccess — terminal success (mode 61). Body after the mode byte:
// Decode4(flag); 0 = success, non-0 renders "Unknown error %d". The server
// only ever sends 0.
// packet-audit:fname CField::OnItemUpgrade#Success
type ViciousHammerSuccess struct {
	mode byte
	flag uint32
}

func NewViciousHammerSuccess(mode byte, flag uint32) ViciousHammerSuccess {
	return ViciousHammerSuccess{mode: mode, flag: flag}
}

func (m ViciousHammerSuccess) Mode() byte        { return m.mode }
func (m ViciousHammerSuccess) Flag() uint32      { return m.flag }
func (m ViciousHammerSuccess) Operation() string { return ViciousHammerWriter }
func (m ViciousHammerSuccess) String() string {
	return fmt.Sprintf("vicious hammer success mode [%d] flag [%d]", m.mode, m.flag)
}

func (m ViciousHammerSuccess) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (61)
		w.WriteInt(m.flag)  // Decode4; 0 = success
		return w.Bytes()
	}
}

func (m *ViciousHammerSuccess) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.flag = r.ReadUint32()
	}
}

// ViciousHammerFailure — terminal failure (mode 62). Body after the mode byte:
// Decode4(errorCode); client notices: 1 = "The item is not upgradable",
// 2 = "2 upgrade increases have been used already", 3 = "You can't use
// Vicious Hammer on Horntail Necklace", default = "Unknown error %d".
// packet-audit:fname CField::OnItemUpgrade#Failure
type ViciousHammerFailure struct {
	mode      byte
	errorCode uint32
}

func NewViciousHammerFailure(mode byte, errorCode uint32) ViciousHammerFailure {
	return ViciousHammerFailure{mode: mode, errorCode: errorCode}
}

func (m ViciousHammerFailure) Mode() byte        { return m.mode }
func (m ViciousHammerFailure) ErrorCode() uint32 { return m.errorCode }
func (m ViciousHammerFailure) Operation() string { return ViciousHammerWriter }
func (m ViciousHammerFailure) String() string {
	return fmt.Sprintf("vicious hammer failure mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m ViciousHammerFailure) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)      // dispatcher mode byte (62)
		w.WriteInt(m.errorCode)  // Decode4 -> notice selector
		return w.Bytes()
	}
}

func (m *ViciousHammerFailure) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadUint32()
	}
}
```

- [ ] **Step 4: Write the body functions**

Create `libs/atlas-packet/field/vicious_hammer_body.go` (mirror `field_effect_body.go`):

```go
package field

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

type ViciousHammerMode string

const (
	ViciousHammerModeOpen    ViciousHammerMode = "OPEN"
	ViciousHammerModeSuccess ViciousHammerMode = "SUCCESS"
	ViciousHammerModeFailure ViciousHammerMode = "FAILURE"
)

// ViciousHammerOpenBody arms the CUIItemUpgrade gauge. token is the
// server-chosen round-trip value the client echoes in ITEM_UPGRADE_UPDATE;
// hammerCount is the target equip's current hammersApplied.
func ViciousHammerOpenBody(token uint32, hammerCount uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(ViciousHammerModeOpen), func(mode byte) packet.Encoder {
		return clientbound.NewViciousHammerOpen(mode, token, hammerCount)
	})
}

// ViciousHammerSuccessBody closes the dialog with the success notice. The
// client treats any non-zero flag as "Unknown error %d", so the flag is
// fixed to 0.
func ViciousHammerSuccessBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(ViciousHammerModeSuccess), func(mode byte) packet.Encoder {
		return clientbound.NewViciousHammerSuccess(mode, 0)
	})
}

// ViciousHammerFailureBody closes the dialog with the notice selected by
// errorCode (1 = not upgradable, 2 = cap reached, 3 = Horntail Necklace,
// other = unknown error).
func ViciousHammerFailureBody(errorCode uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(ViciousHammerModeFailure), func(mode byte) packet.Encoder {
		return clientbound.NewViciousHammerFailure(mode, errorCode)
	})
}
```

- [ ] **Step 5: Run tests and the full packet module**

Run: `cd libs/atlas-packet && go test -race ./field/... -v -run TestViciousHammer && go build ./... && go vet ./...`
Expected: PASS, clean build/vet. (Nothing outside the deleted test referenced `NewViciousHammer` — the channel only uses the `ViciousHammerWriter` const, which survives.)

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/field/clientbound/vicious_hammer.go libs/atlas-packet/field/clientbound/vicious_hammer_test.go libs/atlas-packet/field/vicious_hammer_body.go
git commit -m "feat(packet): ViciousHammer clientbound dispatcher — discrete Open/Success/Failure arms + config-resolved body funcs"
```

---

### Task 5: packet-audit — dispatcher family wiring (yaml, run.go candidates, retire stale artifacts)

Enroll the family in the audit tooling: dispatcher yaml, `#`-suffixed `candidatesFromFName` entries replacing the retired single-candidate case, a serverbound case for `CUIItemUpgrade::Update`, and removal of the now-stale `FieldViciousHammer` evidence/reports (the retired empty-body struct). After this task the VICIOUS_HAMMER cells honestly degrade until Tasks 14/15 re-verify them with the real body — that is expected and correct.

**Files:**
- Create: `docs/packets/dispatchers/vicious_hammer.yaml`
- Modify: `tools/packet-audit/cmd/run.go` (the `case "CField::OnItemUpgrade":` block at ~line 2421)
- Delete: `docs/packets/evidence/gms_v83/field.clientbound.FieldViciousHammer.yaml`, same file under `gms_v84/`, `gms_v87/`, `gms_v95/`
- Delete: `docs/packets/audits/<version>/FieldViciousHammer.json` + `.md` for each version where they exist (check with `ls docs/packets/audits/*/FieldViciousHammer.*`)
- Modify: `docs/packets/audits/STATUS.md` + `status.json` (regenerated by `matrix`)

**Interfaces:**
- Produces: run.go cases `CField::OnItemUpgrade#Open|#Success|#Failure` → `ViciousHammerOpen|Success|Failure` (pkg `field`, clientbound) and `CUIItemUpgrade::Update` → `ItemUpgradeUpdate` (pkg `field`, serverbound). Report names: `FieldViciousHammerOpen` etc., `FieldItemUpgradeUpdate`. Consumed by Tasks 14/15.

- [ ] **Step 1: Write the dispatcher yaml**

Create `docs/packets/dispatchers/vicious_hammer.yaml`:

```yaml
# ViciousHammer - CField::OnItemUpgrade per-version mode table.
# The forwarder delegates to CUIItemUpgrade::OnPacket (v83 sub_82B2C3 via
# sub_82B2AD; v95 CUIItemUpgrade::ShowResult 0x7bec20). Terminal modes are
# hard-coded in the client and VERSION-STABLE: 61 = success, 62 = failure
# (IDA-verified v83 + v95). OPEN is SERVER-CHOSEN: the client accepts any
# byte not in {61, 62}, stores it as m_nReturnResult, and echoes it back in
# ITEM_UPGRADE_UPDATE. Atlas fixes OPEN = 0 (task-129, design OQ-1).
# jms_v185 has no VICIOUS_HAMMER op (registry-absent) - omitted.
# gms_v92 has no registry/IDB/template routing - omitted.
writer: ViciousHammer
fname: CField::OnItemUpgrade
op: VICIOUS_HAMMER
direction: clientbound
operations:
  - { key: OPEN,    modes: { gms_v83: 0,  gms_v84: 0,  gms_v87: 0,  gms_v95: 0 } }
  - { key: SUCCESS, modes: { gms_v83: 61, gms_v84: 61, gms_v87: 61, gms_v95: 61 } }
  - { key: FAILURE, modes: { gms_v83: 62, gms_v84: 62, gms_v87: 62, gms_v95: 62 } }
```

- [ ] **Step 2: Update run.go candidates**

In `tools/packet-audit/cmd/run.go`, replace:

```go
	case "CField::OnItemUpgrade":
		return []candidate{{name: "ViciousHammer", pkg: "field", dir: csvpkg.DirClientbound}}
```

with:

```go
	// VICIOUS_HAMMER (task-129, OP-MODE-PREFIX). CField::OnItemUpgrade is a
	// vtable forwarder into CUIItemUpgrade::OnPacket (v83 sub_82B2C3 via
	// sub_82B2AD; v95 CUIItemUpgrade::ShowResult 0x7bec20), which reads
	// Decode1(mode) and branches: 61 success, 62 failure, any other byte the
	// non-terminal open/arm result. One discrete struct per arm (the
	// FIELD_EFFECT model); the retired empty-body ViciousHammer stub was a
	// false pass once the dialog body was decompiled. jms VERSION-ABSENT.
	case "CField::OnItemUpgrade#Open":
		// else-branch: Decode1(mode) + Decode4(token) + Decode4(hammerCount).
		return []candidate{{name: "ViciousHammerOpen", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnItemUpgrade#Success":
		// case 61: Decode1(mode) + Decode4(flag).
		return []candidate{{name: "ViciousHammerSuccess", pkg: "field", dir: csvpkg.DirClientbound}}
	case "CField::OnItemUpgrade#Failure":
		// case 62: Decode1(mode) + Decode4(errorCode).
		return []candidate{{name: "ViciousHammerFailure", pkg: "field", dir: csvpkg.DirClientbound}}

	// ITEM_UPGRADE_UPDATE (task-129). The CUIItemUpgrade gauge-confirm sender:
	// Encode4(m_nReturnResult) + Encode4(m_nResult) (v83 0x82ae28 /
	// v95 0x7bef50).
	case "CUIItemUpgrade::Update":
		return []candidate{{name: "ItemUpgradeUpdate", pkg: "field", dir: csvpkg.DirServerbound}}
```

- [ ] **Step 3: Retire the stale artifacts**

```bash
git rm docs/packets/evidence/gms_v83/field.clientbound.FieldViciousHammer.yaml \
       docs/packets/evidence/gms_v84/field.clientbound.FieldViciousHammer.yaml \
       docs/packets/evidence/gms_v87/field.clientbound.FieldViciousHammer.yaml \
       docs/packets/evidence/gms_v95/field.clientbound.FieldViciousHammer.yaml
ls docs/packets/audits/*/FieldViciousHammer.* 2>/dev/null   # git rm whatever this lists
```

- [ ] **Step 4: Run the lint + regen gates**

Run from the worktree root:

```bash
go build ./tools/packet-audit/... && go test ./tools/packet-audit/...
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Expected: dispatcher-lint exit 0 (INV-1..5 satisfied: three `#`-entries → three distinct structs, every struct constructed by a body func, no literals, no caller-selected keys). fname-doc exit 0 (the `packet-audit:fname` comments were written in Tasks 3–4). `matrix` regenerates STATUS.md/status.json — the VICIOUS_HAMMER row degrades from ✅ (it graded the retired empty stub) and ITEM_UPGRADE_UPDATE stays ❌; `matrix --check` must introduce no NEW orphan/dangling/stale lines mentioning these packets (the pre-existing conflict backlog is not your bar).

- [ ] **Step 5: Commit**

```bash
git add docs/packets/dispatchers/vicious_hammer.yaml tools/packet-audit/cmd/run.go docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "feat(packet-audit): enroll ViciousHammer dispatcher family + ItemUpgradeUpdate serverbound; retire empty-body stub artifacts"
```

---

### Task 6: atlas-consumables — `Cash()` on equip data + `AddHammersApplied` change

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/data/equipable/model.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/data/equipable/rest.go` (the `Extract` struct literal at ~line 148 that already maps `slots: m.Slots`)
- Modify: `services/atlas-consumables/atlas.com/consumables/equipable/processor.go` (next to `AddSlots` at line 124)
- Test: `services/atlas-consumables/atlas.com/consumables/equipable/processor_test.go` (create if absent)

**Interfaces:**
- Consumes: `asset.ModelBuilder.AddHammersApplied(delta int32)` (exists, `asset/builder.go:275`); `RestModel.Cash bool` (exists, `data/equipable/rest.go:26`).
- Produces: `data/equipable.Model.Cash() bool`; `equipable.AddHammersApplied(amount int32) Change`. Consumed by Task 8.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-consumables/atlas.com/consumables/equipable/processor_test.go`:

```go
package equipable

import (
	"testing"

	"atlas-consumables/asset"

	"github.com/google/uuid"
)

func TestAddHammersAppliedChange(t *testing.T) {
	b := asset.NewBuilder(uuid.New(), 1302000).SetHammersApplied(1)
	AddHammersApplied(1)(b)
	if got := b.Build().HammersApplied(); got != 2 {
		t.Errorf("hammersApplied: got %d, want 2", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./equipable/ -run TestAddHammersApplied -v`
Expected: FAIL — `undefined: AddHammersApplied`

- [ ] **Step 3: Implement**

Append to `services/atlas-consumables/atlas.com/consumables/equipable/processor.go` (after `AddLevel`):

```go
func AddHammersApplied(amount int32) Change {
	return func(m *asset.ModelBuilder) {
		m.AddHammersApplied(amount)
	}
}
```

In `services/atlas-consumables/atlas.com/consumables/data/equipable/model.go`, add the field and getter (the struct has unexported fields set by `Extract`):

```go
	cash          bool
```

```go
func (m Model) Cash() bool {
	return m.cash
}
```

In `services/atlas-consumables/atlas.com/consumables/data/equipable/rest.go` `Extract`, add to the `Model{...}` literal next to `slots: m.Slots,`:

```go
		cash:          m.Cash,
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test -race ./equipable/ ./data/equipable/ && go build ./...`
Expected: PASS, clean build

- [ ] **Step 5: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/equipable/processor.go services/atlas-consumables/atlas.com/consumables/equipable/processor_test.go services/atlas-consumables/atlas.com/consumables/data/equipable/model.go services/atlas-consumables/atlas.com/consumables/data/equipable/rest.go
git commit -m "feat(consumables): AddHammersApplied equip change + Cash flag on equip data model"
```

---

### Task 7: atlas-consumables — Kafka message types and event producer

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/producer.go`

**Interfaces:**
- Produces:
  - `consumable.CommandRequestViciousHammer = "REQUEST_VICIOUS_HAMMER"`, `consumable.RequestViciousHammerBody{HammerSlot, EquipSlot slot.Position}`
  - `consumable.EventTypeViciousHammer = "VICIOUS_HAMMER"`, `consumable.ViciousHammerBody{Success bool; ErrorCode uint32}`
  - `ViciousHammerEventProvider(characterId character.Id, success bool, errorCode uint32) model.Provider[[]kafka.Message]`
- Consumed by Tasks 8, 9, 12 (the channel-side copies in Task 9 must match these JSON shapes exactly).

- [ ] **Step 1: Add the message types**

In `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go`, extend the command const block:

```go
	CommandRequestViciousHammer   = "REQUEST_VICIOUS_HAMMER"
```

Add after `RequestScrollBody`:

```go
// RequestViciousHammerBody carries the two slots the CUIItemUpgrade dialog
// round-trip token packs: the hammer's cash-compartment slot and the target
// equip slot (negative = equipped, positive = equip inventory).
type RequestViciousHammerBody struct {
	HammerSlot slot.Position `json:"hammerSlot"`
	EquipSlot  slot.Position `json:"equipSlot"`
}
```

Extend the event const block:

```go
	EventTypeViciousHammer = "VICIOUS_HAMMER"
```

Add after `ScrollBody`:

```go
// ViciousHammerBody reports the terminal result of a hammer use. ErrorCode is
// the client notice selector (1 = not upgradable, 2 = cap reached,
// 3 = Horntail Necklace, 0 = unknown/internal); meaningful when !Success.
type ViciousHammerBody struct {
	Success   bool   `json:"success"`
	ErrorCode uint32 `json:"errorCode"`
}
```

- [ ] **Step 2: Add the event provider**

In `services/atlas-consumables/atlas.com/consumables/consumable/producer.go`, after `ScrollEventProvider`:

```go
func ViciousHammerEventProvider(characterId character.Id, success bool, errorCode uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Event[consumable.ViciousHammerBody]{
		CharacterId: characterId,
		Type:        consumable.EventTypeViciousHammer,
		Body: consumable.ViciousHammerBody{
			Success:   success,
			ErrorCode: errorCode,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 3: Build**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./... && go vet ./...`
Expected: clean

- [ ] **Step 4: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go services/atlas-consumables/atlas.com/consumables/consumable/producer.go
git commit -m "feat(consumables): REQUEST_VICIOUS_HAMMER command + VICIOUS_HAMMER event types and producer"
```

---

### Task 8: atlas-consumables — hammer request/consume flow (validation + atomic apply)

The scroll-flow structural precedent (`RequestScroll` `consumable/processor.go:515` / `ConsumeScroll` `:606`): read-validate, register a OneTime reservation-status handler, reserve the hammer in the CASH compartment; the callback re-validates against fresh state (idempotency/anti-replay), applies `ChangeStat(AddSlots(1), AddHammersApplied(1))`, consumes the hammer, and emits the terminal event. Failures cancel the reservation and emit the failure event (never the generic `ERROR` event — the hammer dialog needs mode-62, not enable-actions).

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go`
- Test: `services/atlas-consumables/atlas.com/consumables/consumable/vicious_hammer_test.go`

**Interfaces:**
- Consumes: Task 6 (`equipable.AddHammersApplied`, `data/equipable.Model.Cash()`), Task 7 (message types + provider), Task 1 (`item.ClassificationViciousHammer`), existing `compartment.Reserves`/`RequestReserve`/`ConsumeItem`/`CancelItemReservation`, `once.ReservationValidator`, `compartment.Consume`.
- Produces: `(p *Processor) RequestViciousHammer(characterId uint32, hammerSlot int16, equipSlot int16) error`; `ConsumeViciousHammer(transactionId uuid.UUID, characterId uint32, hammerItem *asset.Model, equipSlot int16) ItemConsumer`; `(p *Processor) ViciousHammerError(characterId uint32, transactionId uuid.UUID, hammerSlot int16, errorCode uint32, err error) error`; pure helpers `resolveViciousHammerTarget(c character.Model, equipSlot int16) (*asset.Model, bool)` and `viciousHammerErrorCode(target asset.Model, dataSlots uint16, dataCash bool) uint32`. Consumed by the consumer handler in this task.

- [ ] **Step 1: Write the failing tests (pure validation logic)**

Create `services/atlas-consumables/atlas.com/consumables/consumable/vicious_hammer_test.go`:

```go
package consumable

import (
	"testing"

	"atlas-consumables/asset"

	"github.com/google/uuid"
)

func equipAsset(templateId uint32, hammersApplied uint32) asset.Model {
	return asset.NewBuilder(uuid.New(), templateId).
		SetHammersApplied(hammersApplied).
		Build()
}

func TestViciousHammerErrorCodeEligible(t *testing.T) {
	if code := viciousHammerErrorCode(equipAsset(1302000, 0), 7, false); code != 0 {
		t.Errorf("eligible target: got code %d, want 0", code)
	}
	if code := viciousHammerErrorCode(equipAsset(1302000, 1), 7, false); code != 0 {
		t.Errorf("one hammer applied: got code %d, want 0", code)
	}
}

func TestViciousHammerErrorCodeCapReached(t *testing.T) {
	// IDA-verified cap: error 2 = "2 upgrade increases have been used already".
	if code := viciousHammerErrorCode(equipAsset(1302000, 2), 7, false); code != ViciousHammerErrorCapReached {
		t.Errorf("cap reached: got code %d, want %d", code, ViciousHammerErrorCapReached)
	}
	if code := viciousHammerErrorCode(equipAsset(1302000, 3), 7, false); code != ViciousHammerErrorCapReached {
		t.Errorf("above cap: got code %d, want %d", code, ViciousHammerErrorCapReached)
	}
}

func TestViciousHammerErrorCodeNotUpgradable(t *testing.T) {
	// WZ tuc == 0 -> client notice 1 "The item is not upgradable".
	if code := viciousHammerErrorCode(equipAsset(1302000, 0), 0, false); code != ViciousHammerErrorNotUpgradable {
		t.Errorf("zero-slot equip: got code %d, want %d", code, ViciousHammerErrorNotUpgradable)
	}
	// Cash equips are excluded.
	if code := viciousHammerErrorCode(equipAsset(1302000, 0), 7, true); code != ViciousHammerErrorNotUpgradable {
		t.Errorf("cash equip: got code %d, want %d", code, ViciousHammerErrorNotUpgradable)
	}
}

func TestViciousHammerErrorCodeHorntail(t *testing.T) {
	// 1122000 = Horntail Necklace (WZ String.wz/Eqp.img.xml, GMS 83.1). It has
	// tuc=3, so the exclusion must fire on the id, not the slot count.
	if code := viciousHammerErrorCode(equipAsset(1122000, 0), 3, false); code != ViciousHammerErrorHorntail {
		t.Errorf("horntail necklace: got code %d, want %d", code, ViciousHammerErrorHorntail)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run TestViciousHammer -v`
Expected: FAIL — `undefined: viciousHammerErrorCode` (etc.)

- [ ] **Step 3: Implement the flow**

Append to `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` (after the scroll block, ~line 738). The imports already cover everything used below (`equipable2` = `atlas-consumables/data/equipable`, `once`, `consumer`, `message`, `topic`, `compartment2`):

```go
// Vicious Hammer (item classification 557, task-129). Client failure codes
// (IDA v83 sub_82B2C3 / v95 CUIItemUpgrade::ShowResult 0x7bec20):
const (
	ViciousHammerErrorUnknown       uint32 = 0 // client default arm: "Unknown error %d"
	ViciousHammerErrorNotUpgradable uint32 = 1 // "The item is not upgradable"
	ViciousHammerErrorCapReached    uint32 = 2 // "2 upgrade increases have been used already"
	ViciousHammerErrorHorntail      uint32 = 3 // "You can't use Vicious Hammer on Horntail Necklace"
)

// maxHammersApplied is IDA-verified two ways: the error-2 notice and the
// client's "N upgrades are left" rendering of 2 - hammerCount.
const maxHammersApplied uint32 = 2

// horntailNecklaceItemId is WZ-verified (String.wz/Eqp.img.xml, GMS 83.1).
// The necklace has tuc=3, so the client's dedicated error 3 must be an
// explicit id exclusion.
const horntailNecklaceItemId uint32 = 1122000

// resolveViciousHammerTarget locates the target equip: negative slot = an
// equipped item, positive = a slot in the equip inventory (design §7).
func resolveViciousHammerTarget(c character.Model, equipSlot int16) (*asset.Model, bool) {
	if equipSlot < 0 {
		s, err := slot.GetSlotByPosition(slot.Position(equipSlot))
		if err != nil {
			return nil, false
		}
		sm, ok := c.Equipment().Get(s.Type)
		if !ok || sm.Equipable == nil {
			return nil, false
		}
		return sm.Equipable, true
	}
	return c.Inventory().Equipable().FindBySlot(equipSlot)
}

// viciousHammerErrorCode returns 0 when the target is hammer-eligible, else
// the client notice code. dataSlots/dataCash come from the equip's WZ-derived
// data (atlas-data equipment reader: tuc / cash).
func viciousHammerErrorCode(target asset.Model, dataSlots uint16, dataCash bool) uint32 {
	if target.TemplateId() == horntailNecklaceItemId {
		return ViciousHammerErrorHorntail
	}
	if dataSlots == 0 || dataCash {
		return ViciousHammerErrorNotUpgradable
	}
	if target.HammersApplied() >= maxHammersApplied {
		return ViciousHammerErrorCapReached
	}
	return 0
}

// validateViciousHammerUse fetches the target's WZ-derived equip data and
// applies viciousHammerErrorCode.
func (p *Processor) validateViciousHammerUse(target asset.Model) uint32 {
	ed, err := equipable2.NewProcessor(p.l, p.ctx).GetById(target.TemplateId())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to fetch equip data [%d] for hammer validation.", target.TemplateId())
		return ViciousHammerErrorNotUpgradable // fail closed: an unverifiable equip is rejected, not approved
	}
	return viciousHammerErrorCode(target, ed.Slots(), ed.Cash())
}

// ViciousHammerError cancels the (possible) hammer reservation and emits the
// terminal failure event. Mirrors ConsumeError but on the VICIOUS_HAMMER
// event type — the hammer dialog needs the mode-62 notice, not the generic
// enable-actions ERROR event.
func (p *Processor) ViciousHammerError(characterId uint32, transactionId uuid.UUID, hammerSlot int16, errorCode uint32, err error) error {
	p.l.WithError(err).Debugf("Character [%d] vicious hammer rejected with code [%d].", characterId, errorCode)
	cErr := p.cpp.CancelItemReservation(characterId, inventory2.TypeValueCash, transactionId, hammerSlot)
	if cErr != nil {
		p.l.WithError(cErr).Errorf("Unable to cancel hammer reservation in slot [%d] for character [%d] transaction [%s].", hammerSlot, characterId, transactionId)
	}
	cErr = producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvEventTopic)(ViciousHammerEventProvider(ts.Id(characterId), false, errorCode))
	if cErr != nil {
		p.l.WithError(cErr).Errorf("Unable to issue vicious hammer failure event for character [%d]; dialog likely stuck.", characterId)
	}
	return err
}

// RequestViciousHammer validates a hammer use and reserves the hammer; the
// reservation callback (ConsumeViciousHammer) performs the atomic
// consume + mutate. Packet A performed only a cheap pre-check in the channel —
// this is the authoritative validation (design §4.1).
func (p *Processor) RequestViciousHammer(characterId uint32, hammerSlot int16, equipSlot int16) error {
	cp := character.NewProcessor(p.l, p.ctx)
	transactionId := uuid.New()

	c, err := cp.GetById(cp.InventoryDecorator)(characterId)
	if err != nil {
		return p.ViciousHammerError(characterId, transactionId, hammerSlot, ViciousHammerErrorUnknown, err)
	}

	hammer, ok := c.Inventory().Cash().FindBySlot(hammerSlot)
	if !ok || item2.GetClassification(item2.Id(hammer.TemplateId())) != item2.ClassificationViciousHammer {
		return p.ViciousHammerError(characterId, transactionId, hammerSlot, ViciousHammerErrorUnknown, errors.New("hammer not found at claimed slot"))
	}

	target, ok := resolveViciousHammerTarget(c, equipSlot)
	if !ok {
		return p.ViciousHammerError(characterId, transactionId, hammerSlot, ViciousHammerErrorNotUpgradable, errors.New("target equip not found"))
	}
	if code := p.validateViciousHammerUse(*target); code != 0 {
		return p.ViciousHammerError(characterId, transactionId, hammerSlot, code, errors.New("hammer validation failed"))
	}

	p.l.Debugf("Creating OneTime topic consumer to await hammer transaction [%s] completion.", transactionId.String())
	t, _ := topic.EnvProvider(p.l)(compartment2.EnvEventTopicStatus)()
	validator := once.ReservationValidator(transactionId, hammer.TemplateId())
	handler := compartment.Consume(ConsumeViciousHammer(transactionId, characterId, hammer, equipSlot))
	_, err = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(validator, handler)))

	err = p.cpp.RequestReserve(transactionId, characterId, inventory2.TypeValueCash, []compartment.Reserves{{
		Slot:     hammerSlot,
		ItemId:   hammer.TemplateId(),
		Quantity: 1,
	}})
	if err != nil {
		return p.ViciousHammerError(characterId, transactionId, hammerSlot, ViciousHammerErrorUnknown, err)
	}
	return nil
}

// ConsumeViciousHammer runs when the hammer reservation is confirmed. It
// re-validates against fresh state (a replayed confirm re-checks the cap at
// execution time — design §4.1), applies slots+1 / hammersApplied+1 in one
// MODIFY_EQUIPMENT command, consumes the hammer, and emits the terminal event.
func ConsumeViciousHammer(transactionId uuid.UUID, characterId uint32, hammerItem *asset.Model, equipSlot int16) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			cp := character.NewProcessor(l, ctx)
			ep := equipable.NewProcessor(l, ctx)
			cpp := compartment.NewProcessor(l, ctx)

			c, err := cp.GetById(cp.InventoryDecorator)(characterId)
			if err != nil {
				return p.ViciousHammerError(characterId, transactionId, hammerItem.Slot(), ViciousHammerErrorUnknown, err)
			}
			target, ok := resolveViciousHammerTarget(c, equipSlot)
			if !ok {
				return p.ViciousHammerError(characterId, transactionId, hammerItem.Slot(), ViciousHammerErrorNotUpgradable, errors.New("target equip not found"))
			}
			if code := p.validateViciousHammerUse(*target); code != 0 {
				return p.ViciousHammerError(characterId, transactionId, hammerItem.Slot(), code, errors.New("hammer validation failed at execution time"))
			}

			err = ep.ChangeStat(characterId, transactionId, *target, equipable.AddSlots(1), equipable.AddHammersApplied(1))
			if err != nil {
				return p.ViciousHammerError(characterId, transactionId, hammerItem.Slot(), ViciousHammerErrorUnknown, err)
			}

			err = cpp.ConsumeItem(characterId, inventory2.TypeValueCash, transactionId, hammerItem.Slot())
			if err != nil {
				l.WithError(err).Errorf("Unable to consume hammer [%d] for character [%d]; equip already mutated.", hammerItem.TemplateId(), characterId)
			}

			err = producer.ProviderImpl(l)(ctx)(consumable.EnvEventTopic)(ViciousHammerEventProvider(ts.Id(characterId), true, 0))
			if err != nil {
				l.WithError(err).Errorf("Unable to issue vicious hammer success event for character [%d]; dialog likely stuck.", characterId)
			}
			return nil
		}
	}
}
```

- [ ] **Step 4: Register the command handler**

In `services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go`, add after `handleRequestScroll`:

```go
func handleRequestViciousHammer(l logrus.FieldLogger, ctx context.Context, c consumable2.Command[consumable2.RequestViciousHammerBody]) {
	if c.Type != consumable2.CommandRequestViciousHammer {
		return
	}
	err := consumable.NewProcessor(l, ctx).RequestViciousHammer(uint32(c.CharacterId), int16(c.Body.HammerSlot), int16(c.Body.EquipSlot))
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to use vicious hammer in slot [%d] as expected.", c.CharacterId, c.Body.HammerSlot)
	}
}
```

And register it in `InitHandlers` after the `handleRequestScroll` registration:

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestViciousHammer))); err != nil {
			return err
		}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test -race ./... && go vet ./... && go build ./...`
Expected: PASS, clean

- [ ] **Step 6: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/consumable/processor.go services/atlas-consumables/atlas.com/consumables/consumable/vicious_hammer_test.go services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go
git commit -m "feat(consumables): vicious hammer request/consume flow — authoritative validation + atomic reserve/consume/mutate"
```

---

### Task 9: atlas-channel — Kafka message types, command producer, processor method

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/consumable/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/consumable/processor.go`

**Interfaces:**
- Consumes: JSON shapes from Task 7 (field names must match exactly: `hammerSlot`, `equipSlot`, `success`, `errorCode`).
- Produces: `consumable.NewProcessor(l, ctx).RequestViciousHammerUse(f field.Model, characterId character.Id, hammerSlot slot.Position, equipSlot slot.Position) error`; channel-side `consumable2.EventTypeViciousHammer` + `ViciousHammerBody`. Consumed by Tasks 11 and 12.

- [ ] **Step 1: Mirror the message types**

In `services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go`, extend the command consts:

```go
	CommandRequestViciousHammer = "REQUEST_VICIOUS_HAMMER"
```

after `RequestScrollBody` add:

```go
type RequestViciousHammerBody struct {
	HammerSlot slot.Position `json:"hammerSlot"`
	EquipSlot  slot.Position `json:"equipSlot"`
}
```

extend the event consts:

```go
	EventTypeViciousHammer = "VICIOUS_HAMMER"
```

after `ScrollBody` add:

```go
type ViciousHammerBody struct {
	Success   bool   `json:"success"`
	ErrorCode uint32 `json:"errorCode"`
}
```

- [ ] **Step 2: Add the command provider**

In `services/atlas-channel/atlas.com/channel/consumable/producer.go`, after `RequestScrollCommandProvider` (mirror its shape exactly — same `Command` envelope fields from `f field.Model`):

```go
func RequestViciousHammerCommandProvider(f field.Model, characterId character.Id, hammerSlot slot.Position, equipSlot slot.Position) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.RequestViciousHammerBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        consumable.CommandRequestViciousHammer,
		Body: consumable.RequestViciousHammerBody{
			HammerSlot: hammerSlot,
			EquipSlot:  equipSlot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

(Copy the envelope-field style from `RequestScrollCommandProvider` at `producer.go:33` verbatim — if it sets the fields differently, match it.)

- [ ] **Step 3: Add the processor method**

In `services/atlas-channel/atlas.com/channel/consumable/processor.go`, after `RequestScrollUse`:

```go
func (p *Processor) RequestViciousHammerUse(f field.Model, characterId character.Id, hammerSlot slot.Position, equipSlot slot.Position) error {
	p.l.Debugf("Character [%d] attempting to use vicious hammer in slot [%d] on equip slot [%d].", characterId, hammerSlot, equipSlot)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestViciousHammerCommandProvider(f, characterId, hammerSlot, equipSlot))
}
```

- [ ] **Step 4: Build**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...`
Expected: clean

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go services/atlas-channel/atlas.com/channel/consumable/producer.go services/atlas-channel/atlas.com/channel/consumable/processor.go
git commit -m "feat(channel): REQUEST_VICIOUS_HAMMER command plumbing"
```

---

### Task 10: atlas-channel — token helpers + hammer arm in `CharacterCashItemUseHandle` (Packet A)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/vicious_hammer_token.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/vicious_hammer_token_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`

**Interfaces:**
- Consumes: Task 2 (`cashsb.NewItemUseViciousHammer`), Task 4 (`fieldpkt.ViciousHammerOpenBody` / `ViciousHammerFailureBody`), existing `character2.NewProcessor(l, ctx).GetEquipableInSlot(characterId, slot)` (`character/processor.go:197`; the equip compartment holds equipped items at negative slots), `fieldcb.ViciousHammerWriter`.
- Produces: `packViciousHammerToken(hammerSlot int16, equipSlot int16) uint32`, `unpackViciousHammerToken(token uint32) (hammerSlot int16, equipSlot int16)`; named constants `CashSlotItemTypeViciousHammer` (66) / `CashSlotItemTypeViciousHammerV95` (67) and `viciousHammerCashSlotItemType(t tenant.Model) CashSlotItemType`. Consumed by Task 11.

- [ ] **Step 1: Write the failing token tests**

Create `services/atlas-channel/atlas.com/channel/socket/handler/vicious_hammer_token_test.go`:

```go
package handler

import "testing"

func TestViciousHammerTokenRoundTrip(t *testing.T) {
	cases := []struct {
		name       string
		hammerSlot int16
		equipSlot  int16
	}{
		{"inventory target", 5, 3},
		{"equipped target (negative slot)", 1, -5},
		{"high slots", 96, 24},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := packViciousHammerToken(tc.hammerSlot, tc.equipSlot)
			h, e := unpackViciousHammerToken(token)
			if h != tc.hammerSlot || e != tc.equipSlot {
				t.Errorf("got (%d, %d), want (%d, %d)", h, e, tc.hammerSlot, tc.equipSlot)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestViciousHammerToken -v`
Expected: FAIL — `undefined: packViciousHammerToken`

- [ ] **Step 3: Implement the token helpers**

Create `services/atlas-channel/atlas.com/channel/socket/handler/vicious_hammer_token.go`:

```go
package handler

// The CUIItemUpgrade round-trip token: the server chooses an arbitrary int32
// the client stores (m_nResult) and echoes verbatim in ITEM_UPGRADE_UPDATE.
// Atlas packs both slots into it so the confirm handler is stateless
// (design §4): high int16 = the hammer's cash-compartment slot, low int16 =
// the target equip slot (negative = equipped).

func packViciousHammerToken(hammerSlot int16, equipSlot int16) uint32 {
	return uint32(uint16(hammerSlot))<<16 | uint32(uint16(equipSlot))
}

func unpackViciousHammerToken(token uint32) (int16, int16) {
	return int16(uint16(token >> 16)), int16(uint16(token))
}
```

- [ ] **Step 4: Run token tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/ -run TestViciousHammerToken -v`
Expected: PASS

- [ ] **Step 5: Add the hammer arm to the cash-item-use handler**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`:

1. Change the handler signature to keep the writer producer (line 25): `func CharacterCashItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(...)`.
2. Name the constants (extend the const block at line 116):

```go
	CashSlotItemTypeViciousHammer    = CashSlotItemType(66) // GMS < 95
	CashSlotItemTypeViciousHammerV95 = CashSlotItemType(67) // GMS >= 95
```

3. Add the tenant-aware selector (near `GetCashSlotItemType`) — note plain 66 means CharacterCreation on v95, so the check must be version-scoped:

```go
func viciousHammerCashSlotItemType(t tenant.Model) CashSlotItemType {
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		return CashSlotItemTypeViciousHammerV95
	}
	return CashSlotItemTypeViciousHammer
}
```

4. Update the `category == 557` branch in `GetCashSlotItemType` (line ~469) to return the named constants instead of raw `CashSlotItemType(67)` / `CashSlotItemType(66)`.
5. Add the arm after the `CashSlotItemTypeFieldEffect` block (before the fall-through warn), and delete the stale `// TODO for v83 there is a trailing updateTime.` comment (the trailing updateTime is the hammer tail, now decoded):

```go
		if it == viciousHammerCashSlotItemType(t) {
			sp := cashsb.NewItemUseViciousHammer()
			sp.Decode(l, ctx)(r, readerOptions)
			handleViciousHammerOpen(l, ctx, wp)(s, source, *sp)
			return
		}
```

6. Add the pre-check helper at the bottom of the file. Packet A performs NO mutation — it either arms the gauge (open-arm) or rejects immediately; authoritative validation re-runs in atlas-consumables on Packet B (design §4.1). The cheap pre-check covers existence + cap; WZ eligibility (codes 1/3 from equip data) is left to the authoritative pass — a gauge that later fails with mode 62 is correct UX:

```go
func handleViciousHammerOpen(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, hammerSlot slot.Position, sp cashsb.ItemUseViciousHammer) {
	return func(s session.Model, hammerSlot slot.Position, sp cashsb.ItemUseViciousHammer) {
		announce := func(body func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte) {
			err := session.Announce(l)(ctx)(wp)(fieldcb.ViciousHammerWriter)(body)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write vicious hammer response to character [%d].", s.CharacterId())
			}
		}

		equipSlot := int16(sp.SlotPosition())
		target, err := character2.NewProcessor(l, ctx).GetEquipableInSlot(s.CharacterId(), equipSlot)()
		if err != nil {
			l.Warnf("Character [%d] attempted vicious hammer on missing equip slot [%d].", s.CharacterId(), equipSlot)
			announce(fieldpkt.ViciousHammerFailureBody(1)) // "The item is not upgradable"
			return
		}
		if target.HammersApplied() >= 2 {
			announce(fieldpkt.ViciousHammerFailureBody(2)) // "2 upgrade increases have been used already"
			return
		}

		token := packViciousHammerToken(int16(hammerSlot), equipSlot)
		announce(fieldpkt.ViciousHammerOpenBody(token, target.HammersApplied()))
	}
}
```

7. Add the imports: `fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"` and `fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"` (match alias style used elsewhere in the handler package — check a neighboring handler file and reuse its aliases if they differ).

- [ ] **Step 6: Build + test the handler package**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./socket/handler/... && go vet ./...`
Expected: clean. (Note: the raw `2` cap literals in the pre-check mirror the IDA-verified cap; the authoritative constant lives in atlas-consumables. If you prefer, hoist `const viciousHammerCap = 2` next to the token helpers — do NOT import it across services.)

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/vicious_hammer_token.go services/atlas-channel/atlas.com/channel/socket/handler/vicious_hammer_token_test.go services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go
git commit -m "feat(channel): vicious hammer arm in cash-item-use — pre-check + open-arm gauge response"
```

---

### Task 11: atlas-channel — `ItemUpgradeUpdateHandle` (Packet B) + registration

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/item_upgrade_update.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (the `produceHandlers()` map, ~line 867)

**Interfaces:**
- Consumes: Task 3 (`fieldsb.ItemUpgradeUpdate`, `fieldsb.ItemUpgradeUpdateHandle`), Task 9 (`RequestViciousHammerUse`), Task 10 (`unpackViciousHammerToken`).
- Produces: `handler.ItemUpgradeUpdateHandleFunc` registered under `fieldsb.ItemUpgradeUpdateHandle`.

- [ ] **Step 1: Write the handler**

Create `services/atlas-channel/atlas.com/channel/socket/handler/item_upgrade_update.go` (import aliases: match `character_cash_item_use.go` — `fieldsb` is the alias main.go uses for `field/serverbound`; in the handler package use the same import path with a local alias):

```go
package handler

import (
	"atlas-channel/consumable"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// ItemUpgradeUpdateHandleFunc handles the CUIItemUpgrade gauge-confirm packet.
// The client echoes the open-arm mode byte (returnResult) and the server's
// round-trip token (result), which packs hammerSlot|equipSlot. All
// authoritative validation happens in atlas-consumables against fresh state —
// a forged or replayed confirm is rejected there (design §4.1).
func ItemUpgradeUpdateHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := fieldsb.ItemUpgradeUpdate{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		hammerSlot, equipSlot := unpackViciousHammerToken(p.Result())
		err := consumable.NewProcessor(l, ctx).RequestViciousHammerUse(s.Field(), character.Id(s.CharacterId()), slot.Position(hammerSlot), slot.Position(equipSlot))
		if err != nil {
			l.WithError(err).Errorf("Character [%d] unable to request vicious hammer application.", s.CharacterId())
		}
	}
}
```

- [ ] **Step 2: Register in main.go**

In `services/atlas-channel/atlas.com/channel/main.go` `produceHandlers()`, next to the cash-item-use registration (line 867):

```go
	handlerMap[fieldsb.ItemUpgradeUpdateHandle] = handler.ItemUpgradeUpdateHandleFunc
```

(`fieldsb` is already imported in main.go — it provides `fieldsb.MapChangeHandle` at line 802.)

- [ ] **Step 3: Build**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...`
Expected: clean

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/item_upgrade_update.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(channel): ItemUpgradeUpdate handler — gauge confirm dispatches REQUEST_VICIOUS_HAMMER"
```

---

### Task 12: atlas-channel — hammer-result Kafka consumer

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go`

**Interfaces:**
- Consumes: Task 9 (`consumable2.EventTypeViciousHammer`, `ViciousHammerBody`), Task 4 (`fieldpkt.ViciousHammerSuccessBody` / `ViciousHammerFailureBody`), `fieldcb.ViciousHammerWriter`.
- Produces: the terminal mode-61/62 write to the initiating character's session.

- [ ] **Step 1: Add the handler**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go`, add imports `fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"` and `fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"`, then after `handleScrollConsumableEvent`:

```go
func handleViciousHammerConsumableEvent(sc server.Model, wp writer.Producer) message.Handler[consumable2.Event[consumable2.ViciousHammerBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e consumable2.Event[consumable2.ViciousHammerBody]) {
		if e.Type != consumable2.EventTypeViciousHammer {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		body := fieldpkt.ViciousHammerSuccessBody()
		if !e.Body.Success {
			body = fieldpkt.ViciousHammerFailureBody(e.Body.ErrorCode)
		}
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), session.Announce(l)(ctx)(wp)(fieldcb.ViciousHammerWriter)(body))
		if err != nil {
			l.WithError(err).Errorf("Unable to process vicious hammer event for character [%d].", e.CharacterId)
		}
	}
}
```

- [ ] **Step 2: Register it**

In the same file's `InitHandlers`, after the `handleScrollConsumableEvent` registration block:

```go
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleViciousHammerConsumableEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
```

- [ ] **Step 3: Build + full channel tests**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./... && go vet ./...`
Expected: clean

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go
git commit -m "feat(channel): consume VICIOUS_HAMMER result events — terminal success/failure dialog write"
```

---

### Task 13: tenant seed templates — handler entries + operations tables

Seed templates apply only at tenant creation; the live-tenant patch procedure is documented in Task 16. jms and v92 get NO entries (Global Constraints — documented dispositions).

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`

**Interfaces:**
- Consumes: `ItemUpgradeUpdateHandle` (Task 3), the `OPEN/SUCCESS/FAILURE` operations keys (Tasks 4–5).
- Produces: routed opcodes the verification tasks (14/15) need — a serverbound cell only promotes when the version's template routes the op.

- [ ] **Step 1: Add the serverbound handler entries**

In each template's `socket.handlers` array, insert (keep the array's ascending-opcode ordering convention; every entry MUST carry the validator — validator-less entries are silently dropped):

- `template_gms_83_1.json` and `template_gms_84_1.json`:

```json
      {
        "opCode": "0x104",
        "validator": "LoggedInValidator",
        "handler": "ItemUpgradeUpdateHandle"
      },
```

- `template_gms_87_1.json`:

```json
      {
        "opCode": "0x112",
        "validator": "LoggedInValidator",
        "handler": "ItemUpgradeUpdateHandle"
      },
```

- `template_gms_95_1.json`:

```json
      {
        "opCode": "0x128",
        "validator": "LoggedInValidator",
        "handler": "ItemUpgradeUpdateHandle"
      },
```

- [ ] **Step 2: Add the operations tables to the ViciousHammer writer entries**

Each of the four templates already registers the writer (`grep -n '"writer": "ViciousHammer"'` — v83 `0x162`, v84 `0x169`, v87 `0x177`, v95 `0x1A9`). Extend each entry with the mode table (identical in all four — the FieldEffect entry at v83 line ~1736 shows the shape):

```json
      {
        "opCode": "0x162",
        "writer": "ViciousHammer",
        "options": {
          "operations": {
            "OPEN": 0,
            "SUCCESS": 61,
            "FAILURE": 62
          }
        }
      },
```

(Adjust `opCode` per version; keys/values identical.)

- [ ] **Step 3: Validate JSON + operations check**

```bash
for v in 83 84 87 95; do python3 -m json.tool "services/atlas-configurations/seed-data/templates/template_gms_${v}_1.json" > /dev/null && echo "gms_${v} OK"; done
go run ./tools/packet-audit operations --check
```

Expected: all four `OK`; `operations --check` exit 0 (template tables match `docs/packets/dispatchers/vicious_hammer.yaml`).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_83_1.json services/atlas-configurations/seed-data/templates/template_gms_84_1.json services/atlas-configurations/seed-data/templates/template_gms_87_1.json services/atlas-configurations/seed-data/templates/template_gms_95_1.json
git commit -m "feat(config): route ItemUpgradeUpdateHandle + ViciousHammer operations tables (gms 83/84/87/95)"
```

---

### Task 14: packet verification campaign — gms_v83 + gms_v95 (live IDBs)

Promote the cells for the two versions whose protocol was byte-verified live during design. Follow `docs/packets/audits/VERIFYING_A_PACKET.md` §§3–10 exactly. **Known addresses** (design §2): v83 — forwarder `CField::OnItemUpgrade 0x537f8c`, dialog OnPacket `sub_82B2C3` (via `sub_82B2AD`), sender `CUIItemUpgrade::Update 0x82ae28`; v95 — forwarder `0x52a430`, `CUIItemUpgrade::ShowResult 0x7bec20`, sender `CUIItemUpgrade::Update 0x7bef50`.

**Files:**
- Modify: `docs/packets/ida-exports/gms_v83.json`, `docs/packets/ida-exports/gms_v95.json` (surgical splices ONLY)
- Modify: `libs/atlas-packet/field/serverbound/item_upgrade_update_test.go`, `libs/atlas-packet/field/clientbound/vicious_hammer_test.go` (add markers)
- Create: `docs/packets/audits/gms_v83/FieldItemUpgradeUpdate.{json,md}`, `FieldViciousHammerOpen.{json,md}`, `FieldViciousHammerSuccess.{json,md}`, `FieldViciousHammerFailure.{json,md}`; same set under `gms_v95/`
- Create: evidence records under `docs/packets/evidence/gms_v83/` and `gms_v95/` (via `evidence pin`)
- Modify: `docs/packets/audits/STATUS.md`, `status.json` (regenerated)

**Interfaces:**
- Consumes: everything from Tasks 3–5 and 13.
- Produces: `ITEM_UPGRADE_UPDATE` serverbound cells ✅ for v83/v95; `VICIOUS_HAMMER` clientbound op-row re-promoted for v83/v95 (worst-of the three arms).

- [ ] **Step 1: Select the IDA instances**

Use `mcp__ida-pro__list_instances` and match the binary NAME (never assume ports): the v83 dump (`MapleStory_dump.exe`-style name) and `GMS_v95.0_U_DEVM.exe`. If either is missing, STOP and report BLOCKED asking the user to load it — do not substitute or fake.

- [ ] **Step 2 (per version): name the sender and confirm reads**

`select_instance` the version. Decompile the `CUIItemUpgrade::Update` sender (v83 `0x82ae28` / v95 `0x7bef50`) and confirm the read/write order: `COutPacket(op)` + `Encode4(m_nReturnResult)` + `Encode4(m_nResult)`, and that the `COutPacket` ctor opcode matches the registry (v83 260/0x104, v95 296/0x128 — the ctor integer is ground truth, not the symbol). If the function is unnamed (`sub_82AE28`), rename it to `CUIItemUpgrade::Update` via `mcp__ida-pro__rename` first. Decompile the dialog receive path (v83 `sub_82B2C3` via `sub_82B2AD`; v95 `0x7bec20`) and re-confirm the three arm bodies against the Task 4 codecs. Quote actual decompile lines in your working notes — every fixture byte must cite one.

- [ ] **Step 3 (per version): splice the export**

Harvest ONLY the needed entries to a temp output (`-prior-export "" -pending <roster.md> -descent-depth 12` per §10), then splice into the committed export:
- `CUIItemUpgrade::Update` — the real harvested sender entry (strip a `{op: Delegate, ref: COutPacket}` artifact if present).
- Synthetic `CField::OnItemUpgrade#Open`, `#Success`, `#Failure` clientbound entries keyed at the forwarder's address, each leading with the `Decode1` mode then that arm's reads (the FIELD_EFFECT model — mirror how the existing `#`-suffixed entries are shaped in the export; copy an existing synthetic entry's JSON structure such as `CField::OnFieldEffect#Summon`).

NEVER regenerate the whole export. Diff before committing: `git diff --stat docs/packets/ida-exports/` must show only insertions in the two files.

- [ ] **Step 4: Add the verify markers**

Above `TestItemUpgradeUpdateByteOutput` in `item_upgrade_update_test.go`:

```go
// packet-audit:verify packet=field/serverbound/FieldItemUpgradeUpdate version=gms_v83 ida=0x82ae28
// packet-audit:verify packet=field/serverbound/FieldItemUpgradeUpdate version=gms_v95 ida=0x7bef50
```

Above the arm tests in `vicious_hammer_test.go` (forwarder addresses, the FIELD_EFFECT convention):

```go
// packet-audit:verify packet=field/clientbound/FieldViciousHammerOpen version=gms_v83 ida=0x537f8c
// packet-audit:verify packet=field/clientbound/FieldViciousHammerSuccess version=gms_v83 ida=0x537f8c
// packet-audit:verify packet=field/clientbound/FieldViciousHammerFailure version=gms_v83 ida=0x537f8c
// packet-audit:verify packet=field/clientbound/FieldViciousHammerOpen version=gms_v95 ida=0x52a430
// packet-audit:verify packet=field/clientbound/FieldViciousHammerSuccess version=gms_v95 ida=0x52a430
// packet-audit:verify packet=field/clientbound/FieldViciousHammerFailure version=gms_v95 ida=0x52a430
```

(The exact `ida=` values must match the spliced export entries' addresses — adjust if the export uses a different canonical address.)

- [ ] **Step 5: Generate reports**

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  -ida-source docs/packets/ida-exports/gms_v83.json \
  -output /tmp/claude/rpt83
cp /tmp/claude/rpt83/gms_v83/FieldItemUpgradeUpdate.* /tmp/claude/rpt83/gms_v83/FieldViciousHammer{Open,Success,Failure}.* docs/packets/audits/gms_v83/
```

Repeat with the v95 template/export → `docs/packets/audits/gms_v95/`.

- [ ] **Step 6: Pin evidence**

For each version × packet (serverbound is tier-1 — three artifacts mandatory; pin the clientbound arms as well since the family was tier-1):

```bash
go run ./tools/packet-audit evidence pin --packet field/serverbound/FieldItemUpgradeUpdate --version gms_v83 --ida "CUIItemUpgrade::Update" --category TIER1-FIXTURE
go run ./tools/packet-audit evidence pin --packet field/clientbound/FieldViciousHammerOpen --version gms_v83 --ida "CField::OnItemUpgrade#Open" --category TIER1-FIXTURE
# ... Success, Failure; then the same four for gms_v95
```

Then open each written YAML and add `verifies:` manually, e.g.:

```yaml
verifies:
  - libs/atlas-packet/field/serverbound/item_upgrade_update_test.go#TestItemUpgradeUpdateByteOutput
```

- [ ] **Step 7: Regenerate + check**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit fname-doc --check
```

Expected: `ITEM_UPGRADE_UPDATE` v83 + v95 cells ✅; `VICIOUS_HAMMER` v83 + v95 ✅ (v84/v87 pending Task 15); zero NEW orphan/dangling/stale/conflict lines mentioning these packets; dispatcher-lint + fname-doc exit 0.

- [ ] **Step 8: Commit**

```bash
git add docs/packets/ida-exports/gms_v83.json docs/packets/ida-exports/gms_v95.json \
        libs/atlas-packet/field/serverbound/item_upgrade_update_test.go libs/atlas-packet/field/clientbound/vicious_hammer_test.go \
        docs/packets/audits/gms_v83/ docs/packets/audits/gms_v95/ docs/packets/evidence/gms_v83/ docs/packets/evidence/gms_v95/ \
        docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "verify(packet): ITEM_UPGRADE_UPDATE + ViciousHammer dispatcher arms — gms_v83 + gms_v95 promoted"
```

---

### Task 15: packet verification campaign — gms_v84 + gms_v87; jms disposition

Same procedure as Task 14 for v84 and v87. **Gate:** these IDBs were NOT in the loaded instance set as of 2026-06-30 — `list_instances` first; if an IDB is not loaded, STOP and report BLOCKED asking the user to load it (a missing IDB is a genuine blocker per `VERIFYING_A_PACKET.md`; do not interpolate, do not fake addresses). The `CUIItemUpgrade::Update` senders in v84/v87 are likely unnamed — locate them via the §10 byte signature `6A <op> 8D 8D ?? ?? ?? ?? E8` with the version's registry opcode (v84 `0x104`, v87 `0x112`), structure-match to the v83/v95 twins, rename, then harvest/splice.

**Files:**
- Modify: `docs/packets/ida-exports/gms_v84.json`, `gms_v87.json` (splices)
- Modify: the two test files (add v84/v87 markers — forwarder addresses: v84 `0x544395`, v87 `0x55fa12` from the retired evidence; re-confirm live)
- Create: reports under `docs/packets/audits/gms_v84/` and `gms_v87/`; evidence under `docs/packets/evidence/gms_v84/` and `gms_v87/`
- Modify: `docs/packets/audits/STATUS.md`, `status.json`

**Steps (per version, v84 then v87 — spelled out so this task stands alone):**

- [ ] **Step 1: Instance gate.** `mcp__ida-pro__list_instances`; match the binary NAME for the version (never assume ports). If the IDB is not loaded → STOP, report BLOCKED naming exactly which IDB to load. Never interpolate addresses, substitute another version, or fake evidence.

- [ ] **Step 2: Locate + name the sender.** `select_instance` the version. The `CUIItemUpgrade::Update` sender is likely unnamed. Locate it with the §10 byte signature `6A <op> 8D 8D ?? ?? ?? ?? E8` using the version's registry opcode (v84 `0x104`, v87 `0x112`), structure-match against the v83 (`0x82ae28`) / v95 (`0x7bef50`) twins (two `Encode4` calls after the `COutPacket` ctor), confirm the `COutPacket` ctor integer equals the registry opcode (ground truth over symbols), then `mcp__ida-pro__rename` it to `CUIItemUpgrade::Update`. Also decompile the forwarder (`CField::OnItemUpgrade` — retired-evidence addresses to re-confirm: v84 `0x544395`, v87 `0x55fa12`) and its dialog OnPacket callee; confirm the 61/62/else mode switch and per-arm reads match the Task 4 codecs. Quote actual decompile lines.

- [ ] **Step 3: Splice the exports.** Harvest only the needed entries to a temp output (`-prior-export "" -pending <roster.md> -descent-depth 12`), then surgically splice into `docs/packets/ida-exports/gms_v84.json` / `gms_v87.json`: the real `CUIItemUpgrade::Update` entry (strip any `{op: Delegate, ref: COutPacket}` artifact) plus synthetic `CField::OnItemUpgrade#Open`, `#Success`, `#Failure` entries keyed at the forwarder address (copy the JSON shape of the Task 14 v83 splices). NEVER regenerate a whole export; `git diff --stat docs/packets/ida-exports/` must show only insertions.

- [ ] **Step 4: Markers.** Add to `libs/atlas-packet/field/serverbound/item_upgrade_update_test.go`:

```go
// packet-audit:verify packet=field/serverbound/FieldItemUpgradeUpdate version=gms_v84 ida=<v84 sender addr>
// packet-audit:verify packet=field/serverbound/FieldItemUpgradeUpdate version=gms_v87 ida=<v87 sender addr>
```

and to `libs/atlas-packet/field/clientbound/vicious_hammer_test.go` (forwarder addresses):

```go
// packet-audit:verify packet=field/clientbound/FieldViciousHammerOpen version=gms_v84 ida=<v84 forwarder addr>
// packet-audit:verify packet=field/clientbound/FieldViciousHammerSuccess version=gms_v84 ida=<v84 forwarder addr>
// packet-audit:verify packet=field/clientbound/FieldViciousHammerFailure version=gms_v84 ida=<v84 forwarder addr>
// ... same three for gms_v87
```

Fill `<addr>` with the addresses actually confirmed in Step 2 — they must match the spliced export entries.

- [ ] **Step 5: Reports.** Generate to a temp dir and copy only the four reports per version:

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_gms_84_1.json \
  -ida-source docs/packets/ida-exports/gms_v84.json \
  -output /tmp/claude/rpt84
cp /tmp/claude/rpt84/gms_v84/FieldItemUpgradeUpdate.* /tmp/claude/rpt84/gms_v84/FieldViciousHammer{Open,Success,Failure}.* docs/packets/audits/gms_v84/
```

(Repeat with the v87 template/export → `docs/packets/audits/gms_v87/`.)

- [ ] **Step 6: Evidence.** Pin per version × packet and add `verifies:` manually to each written YAML:

```bash
go run ./tools/packet-audit evidence pin --packet field/serverbound/FieldItemUpgradeUpdate --version gms_v84 --ida "CUIItemUpgrade::Update" --category TIER1-FIXTURE
go run ./tools/packet-audit evidence pin --packet field/clientbound/FieldViciousHammerOpen --version gms_v84 --ida "CField::OnItemUpgrade#Open" --category TIER1-FIXTURE
# ... Success, Failure; then the same four for gms_v87
```

- [ ] **Step 7: jms disposition — document only (no code).** Confirm `docs/packets/registry/jms_v185.yaml` has no `VICIOUS_HAMMER` clientbound row and `gms_jms_185.json` has no `CUIItemUpgrade` entries (both already confirmed during planning). Record the disposition where the audit conventions put deferrals (check `docs/packets/ida-exports/_pending.md` / `docs/packets/audits/README*`); at minimum state it in the commit message: jms serverbound `ITEM_UPGRADE_UPDATE` (0x114) stays ❌ unrouted because the result op does not exist in the jms client registry — the flow cannot complete.

- [ ] **Step 8: Gates.**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
```

Expected: v84/v87 cells ✅ for both ops; no NEW problems mentioning these packets; lint/doc/operations exit 0.

- [ ] **Step 9: Commit**

```bash
git add docs/packets/ida-exports/gms_v84.json docs/packets/ida-exports/gms_v87.json \
        libs/atlas-packet/field/serverbound/item_upgrade_update_test.go libs/atlas-packet/field/clientbound/vicious_hammer_test.go \
        docs/packets/audits/gms_v84/ docs/packets/audits/gms_v87/ docs/packets/evidence/gms_v84/ docs/packets/evidence/gms_v87/ \
        docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "verify(packet): ITEM_UPGRADE_UPDATE + ViciousHammer arms — gms_v84 + gms_v87; jms documented version-absent"
```

---

### Task 16: full verification gates + rollout documentation

**Files:**
- Create: `docs/tasks/task-129-vicious-hammer-use/rollout.md`

**Steps:**

- [ ] **Step 1: Module gates**

```bash
cd libs/atlas-constants && go test -race ./... && go vet ./... && go build ./...
cd ../atlas-packet && go test -race ./... && go vet ./... && go build ./...
cd ../../services/atlas-consumables/atlas.com/consumables && go test -race ./... && go vet ./... && go build ./...
cd ../../../atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...
```

Expected: all clean.

- [ ] **Step 2: Docker bakes (mandatory — catches Dockerfile COPY gaps go.work hides)**

From the worktree root:

```bash
docker buildx bake atlas-channel
docker buildx bake atlas-consumables
```

Expected: both build clean. (No new lib was added, so no Dockerfile/go.work edits are expected — the bake proves it.)

- [ ] **Step 3: Repo guards**

```bash
tools/redis-key-guard.sh
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
```

Expected: clean / no new problems.

- [ ] **Step 4: Write the rollout doc**

Create `docs/tasks/task-129-vicious-hammer-use/rollout.md` (repo-relative paths only):

```markdown
# task-129 rollout — live-tenant configuration patch

Seed templates apply only at tenant creation. Existing tenants need a config
patch + channel restart (handlers/writers do not hot-reload).

For EACH live GMS tenant (v83 / v84 / v87 / v95):

1. PATCH the tenant's socket configuration (via atlas-tenants / the UI):
   - Add to `socket.handlers`:
     `{ "opCode": "<per-version>", "validator": "LoggedInValidator", "handler": "ItemUpgradeUpdateHandle" }`
     opcodes: v83/v84 `0x104`, v87 `0x112`, v95 `0x128`.
   - Extend the existing `ViciousHammer` writer entry (v83 `0x162`, v84 `0x169`,
     v87 `0x177`, v95 `0x1A9`) with:
     `"options": { "operations": { "OPEN": 0, "SUCCESS": 61, "FAILURE": 62 } }`
2. Restart the tenant's atlas-channel pods.
3. Smoke test per version (in-game): double-click hammer → drop equip →
   Upgrade → gauge fills → success notice; equip window shows slots+1 without
   relog; third hammer on the same equip → "2 upgrade increases have been used
   already" and the hammer is NOT consumed; Horntail Necklace target →
   dedicated refusal notice.

jms tenants: NOT patched — VICIOUS_HAMMER does not exist in the jms client
registry; the flow is version-absent (see plan Task 15).
gms_v92 tenants: NOT patched — the v92 template is a login-only stub without
CASH_ITEM_USE routing; there is nothing for the hammer flow to attach to.
```

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-129-vicious-hammer-use/rollout.md
git commit -m "docs(task-129): rollout procedure — live-tenant config patch + smoke tests"
```

- [ ] **Step 6: Request code review**

Run `superpowers:requesting-code-review` (plan-adherence + backend-guidelines) BEFORE any PR. Findings go to `docs/tasks/task-129-vicious-hammer-use/audit.md`.

---

## Self-review notes (spec coverage)

- design §2.1 two-phase flow → Tasks 2, 3, 10, 11 (Design B; Packet A arms, Packet B applies).
- design §2.2 mode-prefixed dispatcher → Tasks 4, 5 (discrete structs, config-resolved modes, yaml).
- design §2.3 opcode map → Task 13 (templates) + Global Constraints (registry-verified values).
- design §3.1 packet layer → Tasks 2–4; the cash-use TODO resolution → Task 10 step 5.
- design §3.2 channel → Tasks 9–12.
- design §3.3 consumables → Tasks 6–8 (reserve→consume callback, hammer-specific error path).
- design §3.4 inventory no-change → verified by rollout smoke tests (Task 16 step 4).
- design §3.5 config → Tasks 13, 16.
- design §4.1 atomicity/idempotency → Task 8 (re-validation in `ConsumeViciousHammer`; no `ExecuteTransaction`).
- design §6 error taxonomy + cap → Task 8 constants + tests; Horntail id WZ-verified during planning (1122000).
- design §7 target addressing → `resolveViciousHammerTarget` (Task 8) + signed-slot token tests (Task 10).
- design §8 verification plan → Tasks 14, 15 (v92: no registry/IDB/template — documented, nothing claimable).
- design §9 open questions → OQ-1 resolved (OPEN=0, Task 5 yaml); OQ-2 resolved (jms out, evidence in Global Constraints, documented Task 15); OQ-3 resolved (WZ-verified predicate + 1122000, Task 8); OQ-4 encoded (signed slotPosition; itemTI decoded and available on `ItemUseViciousHammer` if a future version needs it).
- design §10 testing → per-task TDD steps + Task 16 gates.
