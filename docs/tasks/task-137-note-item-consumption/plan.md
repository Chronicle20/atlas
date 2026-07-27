# Note Item Consumption & Memo Packet Verification — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Make every player-initiated note send consume exactly one Note item (classification 509) via a destroy-first saga, wire the USE_CASH_ITEM note arm on **all nine verified versions** (gms_v48/v61/v72/v79/v83/v84/v87/v95, jms_v185), and promote the note-family packet-matrix cells to ✅ (MEMO_RESULT × v84/jms185; NoteOperationDiscard × jms185 **and the four legacy versions v48/v61/v72/v79**).

**Architecture:** The player send path is USE_CASH_ITEM (IDA-verified on **all nine** versions, design §1.1 + `legacy-verify/{v48,v61,v72,v79}.md`); a new `note_send` saga (`DestroyAsset` → new `CreateNote` action) couples consumption with note creation, with orchestrator compensation re-awarding the item if creation fails after the destroy. atlas-notes threads a transaction id through its command/status events and emits a new `CREATE_FAILED` event; the channel's saga status consumer sends SEND_SUCCESS/SEND_ERROR client feedback. The NOTE_ACTION SEND arm (only legitimately written by the unimplemented cash-shop gift flow — and NOT written at all on v48/v79) gets the same ownership+consumption gate, closing the free-note cheat path.

> **Scope revision (main-sync):** originally planned for five versions; main added four legacy columns (v48/v61/v72/v79) now in scope. The saga/service core (Tasks 4–13) is entirely version-agnostic and unchanged. The nine-version fan-out touches only the codec (Task 2 fixtures), the codec gate (Task 1), the writer config (Task 14), and the matrix-promotion tasks (15–17). Two legacy divergences are folded in: (a) **v48/v61 use a shifted MEMO_RESULT mode table** (SEND_SUCCESS=3, SEND_ERROR=4 vs the standard 4/5) — resolved via per-version config, NOT a codec change; a hard-coded 5 would wedge those clients; (b) per-version note-discard bodies (design §1.5). The stale "CharacterCashItemUseHandle routed only on gms_83/84, add to 87/95/jms" premise is dead — main routes it in all nine templates.

**Tech Stack:** Go workspace monorepo; Kafka (segmentio/kafka-go via libs/atlas-kafka); JSON seed templates in atlas-configurations; packet-audit tooling + ida-pro-mcp for matrix promotions.

## Global Constraints

- **Spec:** `docs/tasks/task-137-note-item-consumption/design.md` (all IDA addresses, wire formats, opcodes, and error codes below come from its §1; do NOT re-derive from Cosmic or memory).
- **DOM-25:** client-interpreted bytes (mode bytes, error codes) resolve from tenant config (`operations`/`errors` tables), never hard-coded. **Critical for this task:** the MEMO_RESULT SEND_SUCCESS/SEND_ERROR mode bytes are NOT uniform — v48/v61 use `{SHOW:2, SEND_SUCCESS:3, SEND_ERROR:4}`, v72+ use `{SHOW:3, SEND_SUCCESS:4, SEND_ERROR:5, REFRESH:7}` (design §1.3). A hard-coded `5` silently wedges v48/v61 clients (mode 5 is unhandled there → no unlock, no dialog). The sub-error codes (0/1/2, and the `NO_NOTE_ITEM` sentinel 3) are uniform across all nine.
- **FR-7:** no item is consumed on any pre-flight rejection (unknown receiver, missing item, malformed request). Pre-flight rejections happen in the channel handler BEFORE any saga is created.
- **FR-5:** destroy-first ordering is mandatory — `DestroyAsset` is step 1, `CreateNote` is step 2, in every saga this plan creates.
- **Note flag is `1`** on send (PRD non-goal; design §2.4).
- **Gift memos are out of scope** (design §2.3): the NOTE_ACTION SEND arm's trailing gift fields (byte/int32/8-byte SN) stay undecoded; `note/serverbound/NoteOperationSend` (the gift-send codec, ❌ on all nine) is not a target.
- Test setup uses the project's Builder pattern; no `*_testhelpers.go` files.
- No `// TODO` stubs in landed commits.
- Every task's commit runs from the worktree root `/…/.worktrees/task-137-note-item-consumption` on branch `task-137-note-item-consumption`.
- Verification gate (final task): `go test -race ./...`, `go vet ./...`, `go build ./...` per changed module; `docker buildx bake` for atlas-channel, atlas-saga-orchestrator, atlas-notes; `tools/redis-key-guard.sh`; `tools/goroutine-guard.sh`; `tools/lint.sh --check`; `packet-audit` matrix/fname-doc/operations `--check` exit 0 (these last three guards were added to main after the original plan; they run tree-wide from the repo root).

## File Structure (what is created/modified where)

| Module | Files |
|---|---|
| `libs/atlas-packet` | `cash/serverbound/item_use.go` (gate fix + exported `UpdateTimeFirst`), `cash/serverbound/item_use_note.go` (new arm codec, nine-version fixtures), `note/clientbound/operation_body.go` (version-resolved SEND_ERROR mode + `NO_NOTE_ITEM` key), matching `_test.go` files, fixture tests for the matrix promotions (MEMO_RESULT × v84/jms185; NoteOperationDiscard × jms185/v48/v61/v72/v79) |
| `libs/atlas-saga` | `model.go` (`NoteSend` type, `CreateNote` action), `payloads.go` (`CreateNotePayload`), `unmarshal.go` (case), `unmarshal_test.go` |
| `services/atlas-notes` | `kafka/message/note/kafka.go` (TransactionId + CREATE_FAILED), `note/processor.go` + `note/producer.go` + `note/mock/processor.go` (txn threading), `kafka/consumer/note/consumer.go` (failure event), `note/resource.go` (caller update) |
| `services/atlas-saga-orchestrator` | `kafka/message/note/kafka.go` (new), `note/processor.go` + `note/producer.go` + `note/mock/processor.go` (new), `saga/model.go` (re-exports + unmarshal case), `saga/handler.go` (`handleCreateNote`), `saga/event_acceptance.go` (kinds + tables), `kafka/consumer/note/consumer.go` (new), `main.go` (registration), `saga/producer.go` (note_send completed Results), `saga/compensator.go` (`compensateNoteSend`) |
| `services/atlas-channel` | `kafka/message/saga/kafka.go` (completed body fields + `SagaTypeNoteSend`), `kafka/consumer/saga/consumer.go` (note_send branches), `compartment/model.go` (`FindFirstByClassification`), `socket/handler/note_send.go` (new shared helper), `socket/handler/character_cash_item_use.go` (note arm), `socket/handler/note_operation.go` (SEND gate), `note/processor.go` + `note/producer.go` (SendNote removal) |
| `services/atlas-configurations` | 9 seed templates (`template_gms_48_1.json`, `_61_`, `_72_`, `_79_`, `_83_`, `_84_`, `_87_`, `_95_`, `template_jms_185_1.json`) — writer `operations`/`errors` config only; handler routing already present |
| `docs` | `docs/tasks/task-137-note-item-consumption/live-tenant-patch.md`; `docs/tasks/task-137-note-item-consumption/legacy-verify/{v48,v61,v72,v79}.md` (IDA evidence, already committed); `docs/packets/audits/` evidence + regenerated `STATUS.md` |

Dependency order: Tasks 1–4 (shared libs) → 5 (atlas-notes) → 6–9 (orchestrator) → 10–13 (atlas-channel) → 14 (templates) → 15–17 (matrix promotions, independent of 5–14) → 18 (gate). The matrix promotions now cover MEMO_RESULT × v84/jms185 (Tasks 15–16) and NoteOperationDiscard × jms185 + the four legacy versions (Task 17, expanded).

---

### Task 1: `item_use.go` updateTime gate fix (`>=95` → `>=87` + JMS)

The shared USE_CASH_ITEM prefix codec gates the leading updateTime on `Region=="GMS" && MajorVersion>=95`. IDA-verified (design §1.1): v87 (fn 0xa9fef9) already encodes it leading, and JMS185 (0xaef2f5) leads too; the six versions ≤ v84 (v48/v61/v72/v79/v83/v84) all trail (legacy pass: v48 @0x711d9f, v61 @0x83672e, v72 @0x909123, v79 @0x95a54e). The single predicate `(GMS && >=87) || JMS` classifies all nine correctly. `CharacterCashItemUseHandle` is routed in all nine templates (main wired it), so v87/v95/jms are now live behind this gate — the `>=95` bug would send v87 a trailing decode; the fix is a prerequisite, not just cleanup.

**Files:**
- Modify: `libs/atlas-packet/cash/serverbound/item_use.go`
- Test: `libs/atlas-packet/cash/serverbound/item_use_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `func UpdateTimeFirst(t tenant.Model) bool` (exported from package `serverbound`, import path `github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound`). Task 12 calls it from the channel handler.

- [x] **Step 1: Write the failing test**

Append to `libs/atlas-packet/cash/serverbound/item_use_test.go`:

```go
// TestItemUseUpdateTimeFirst pins the updateTime position per version.
// IDA-verified (task-137 design §1.1, CWvsContext::SendConsumeCashItemUseRequest):
// the six GMS versions <= v84 (v48 0x70e495, v61 0x832a5d, v72 0x904fe2,
// v79 0x95634a, v83 0xa0a63f, v84 0xa54a2f) encode updateTime TRAILING; GMS
// v87 (0xa9fef9), v95 (0x9eb3e0) and JMS185 (0xaef2f5) encode it LEADING,
// immediately after the opcode.
func TestItemUseUpdateTimeFirst(t *testing.T) {
	cases := []struct {
		name    string
		region  string
		major   uint16
		leading bool
	}{
		{"GMS v48 trailing", "GMS", 48, false},
		{"GMS v61 trailing", "GMS", 61, false},
		{"GMS v72 trailing", "GMS", 72, false},
		{"GMS v79 trailing", "GMS", 79, false},
		{"GMS v83 trailing", "GMS", 83, false},
		{"GMS v84 trailing", "GMS", 84, false},
		{"GMS v87 leading", "GMS", 87, true},
		{"GMS v95 leading", "GMS", 95, true},
		{"JMS v185 leading", "JMS", 185, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.region, tc.major, 1)

			tn := tenant.MustFromContext(ctx)
			if got := UpdateTimeFirst(tn); got != tc.leading {
				t.Errorf("UpdateTimeFirst: got %v, want %v", got, tc.leading)
			}

			// leading layout: updateTime(4), slot(2), itemId(4)
			// trailing layout: slot(2), itemId(4) — trailing updateTime is read by the arm codec
			var raw []byte
			if tc.leading {
				raw = []byte{
					0x64, 0x00, 0x00, 0x00, // updateTime = 100
					0x07, 0x00, // slot = 7
					0x50, 0xA8, 0x4D, 0x00, // itemId = 5090000
				}
			} else {
				raw = []byte{
					0x07, 0x00, // slot = 7
					0x50, 0xA8, 0x4D, 0x00, // itemId = 5090000
				}
			}
			req := request.Request(raw)
			reader := request.NewRequestReader(&req, 0)

			p := ItemUse{}
			p.Decode(logrus.New(), ctx)(&reader, map[string]interface{}{})

			if p.Source() != 7 {
				t.Errorf("source: got %d, want 7", p.Source())
			}
			if p.ItemId() != 5090000 {
				t.Errorf("itemId: got %d, want 5090000", p.ItemId())
			}
			if tc.leading && p.UpdateTime() != 100 {
				t.Errorf("updateTime: got %d, want 100", p.UpdateTime())
			}
		})
	}
}
```

Add the needed imports to the test file's import block:

```go
import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)
```

- [x] **Step 2: Run test to verify it fails**

Run (from worktree root): `cd libs/atlas-packet && go test ./cash/serverbound/ -run TestItemUseUpdateTimeFirst -v`
Expected: FAIL — compile error `undefined: UpdateTimeFirst` (and, once defined naively, the v87 case would fail on updateTime).

- [x] **Step 3: Implement**

In `libs/atlas-packet/cash/serverbound/item_use.go`:

Add after the `CharacterCashItemUseHandle` const:

```go
// UpdateTimeFirst reports whether the USE_CASH_ITEM request carries its
// updateTime immediately after the opcode (leading) rather than at the end
// of the packet (trailing). IDA-verified per version in
// CWvsContext::SendConsumeCashItemUseRequest: all GMS <= v84 (v48/v61/v72/
// v79/v83/v84) trail; GMS v87+ and JMS lead.
func UpdateTimeFirst(t tenant.Model) bool {
	return (t.Region() == "GMS" && t.MajorVersion() >= 87) || t.Region() == "JMS"
}
```

Replace both gate expressions (line 38 in `Encode`, line 50 in `Decode`):

```go
		if UpdateTimeFirst(t) {
```

(the old expression in both spots is `if t.Region() == "GMS" && t.MajorVersion() >= 95 {`).

- [x] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -v`
Expected: PASS, including the pre-existing `TestItemUseRoundTrip` (round-trip is symmetric across the gate change).

- [x] **Step 5: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/item_use.go libs/atlas-packet/cash/serverbound/item_use_test.go
git commit -m "fix(atlas-packet): USE_CASH_ITEM leading updateTime starts at v87, JMS leads too"
```

---

### Task 2: `ItemUseNote` arm codec + nine-version fixtures

New serverbound arm codec for the note body of USE_CASH_ITEM. Wire format (design §1.1): `EncodeStr(toName)`, `EncodeStr(message)`, then trailing `updateTime` iff the version trails (the six GMS versions ≤ v84: v48/v61/v72/v79/v83/v84). Mirrors `ItemUseChalkboard`.

**Coverage note:** the `ItemUseNote` codec has NO per-version-number branch — the ONLY variable is the trailing-vs-leading gate boolean. `pt.Variants` already exercises both sides: GMS v28/v83/v84/v86 (all `<87` ⇒ trailing) and v87/v95/jms (leading). The four legacy versions (v48/v61/v72/v79) are all trailing and behaviorally identical to the v28/v83 trailing leg, so `TestItemUseNoteRoundTrip` covers them without adding entries to the shared `pt.Variants` fixture (which other packets' gates depend on positionally — do not extend it here). `TestItemUseNoteDecodeTrailing` pins the exact trailing byte shape; `TestItemUseNoteDecodeLeading` pins the leading shape. The per-version byte evidence for the four legacy cash-item arms is captured in `legacy-verify/{v48,v61,v72,v79}.md`, not as separate codec tests (there is nothing version-specific to test in the codec).

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/item_use_note.go`
- Test: `libs/atlas-packet/cash/serverbound/item_use_note_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `func NewItemUseNote(updateTimeFirst bool) *ItemUseNote` with methods `ToName() string`, `Message() string`, `UpdateTime() uint32`, plus standard `Encode`/`Decode`/`Operation()`/`String()`. Task 12 decodes with it.

- [x] **Step 1: Write the failing test**

Create `libs/atlas-packet/cash/serverbound/item_use_note_test.go`:

```go
package serverbound

import (
	"context"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// TestItemUseNoteDecodeTrailing pins the GMS v83/v84 arm shape: toName,
// message, trailing updateTime (task-137 design §1.1).
func TestItemUseNoteDecodeTrailing(t *testing.T) {
	raw := []byte{
		0x03, 0x00, 'B', 'o', 'b', // toName = "Bob"
		0x02, 0x00, 'h', 'i', // message = "hi"
		0x64, 0x00, 0x00, 0x00, // updateTime = 100 (trailing)
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := NewItemUseNote(false)
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if p.ToName() != "Bob" {
		t.Errorf("toName: got %q, want %q", p.ToName(), "Bob")
	}
	if p.Message() != "hi" {
		t.Errorf("message: got %q, want %q", p.Message(), "hi")
	}
	if p.UpdateTime() != 100 {
		t.Errorf("updateTime: got %d, want 100", p.UpdateTime())
	}
}

// TestItemUseNoteDecodeLeading pins the GMS v87/v95 and JMS185 arm shape:
// updateTime was already consumed by the leading ItemUse prefix, so the arm
// body is just toName, message.
func TestItemUseNoteDecodeLeading(t *testing.T) {
	raw := []byte{
		0x03, 0x00, 'B', 'o', 'b',
		0x02, 0x00, 'h', 'i',
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := NewItemUseNote(true)
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if p.ToName() != "Bob" {
		t.Errorf("toName: got %q, want %q", p.ToName(), "Bob")
	}
	if p.Message() != "hi" {
		t.Errorf("message: got %q, want %q", p.Message(), "hi")
	}
	if p.UpdateTime() != 0 {
		t.Errorf("updateTime: got %d, want 0 (leading variant reads none)", p.UpdateTime())
	}
}

func TestItemUseNoteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			updateTimeFirst := (v.Region == "GMS" && v.MajorVersion >= 87) || v.Region == "JMS"
			input := ItemUseNote{toName: "Alice", message: "hello there", updateTime: 42, updateTimeFirst: updateTimeFirst}
			output := NewItemUseNote(updateTimeFirst)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ToName() != input.ToName() {
				t.Errorf("toName: got %q, want %q", output.ToName(), input.ToName())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %q, want %q", output.Message(), input.Message())
			}
			if !updateTimeFirst && output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %d, want %d", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -run TestItemUseNote -v`
Expected: FAIL — compile error `undefined: NewItemUseNote`.

- [x] **Step 3: Implement**

Create `libs/atlas-packet/cash/serverbound/item_use_note.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseNote - the note (memo send) arm of USE_CASH_ITEM, consume cash item
// type 0x15 (classification 509). The client opens a send-memo dialog and
// encodes recipient + message; updateTime trails on GMS v83/v84 only.
// packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest#Note
type ItemUseNote struct {
	toName          string
	message         string
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseNote(updateTimeFirst bool) *ItemUseNote {
	return &ItemUseNote{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseNote) ToName() string     { return m.toName }
func (m ItemUseNote) Message() string    { return m.message }
func (m ItemUseNote) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseNote) Operation() string { return "ItemUseNote" }

func (m ItemUseNote) String() string {
	return fmt.Sprintf("toName [%s] message [%s] updateTime [%d]", m.toName, m.message, m.updateTime)
}

func (m ItemUseNote) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.toName)
		w.WriteAsciiString(m.message)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseNote) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.toName = r.ReadAsciiString()
		m.message = r.ReadAsciiString()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/item_use_note.go libs/atlas-packet/cash/serverbound/item_use_note_test.go
git commit -m "feat(atlas-packet): USE_CASH_ITEM note arm codec (ItemUseNote), 9-version fixtures"
```

---

### Task 3: Fix `NoteSendErrorBody` mode byte + add `NO_NOTE_ITEM` error key

Pre-existing bug found during planning: `NoteSendErrorBody` resolves its mode byte from the `SEND_SUCCESS` operations key (`libs/atlas-packet/note/clientbound/operation_body.go:38`), so every "error" currently goes out as the success mode (which per design §1.3 shows the success notice and never reads the error byte). It must resolve `SEND_ERROR` instead. **The resolved value is version-dependent and comes from the tenant `operations` table** — mode **5** on v72+/GMS83+/JMS, mode **4** on v48/v61 (design §1.3). The code stays version-agnostic (it reads whatever the tenant config holds); Task 14 populates the shifted v48/v61 value. Also add the `NO_NOTE_ITEM` semantic key (design §5.3: sub-code value 3 in tenant config — deliberately outside the client's 0–2 dialog range; IDA-verified silent excl-unlock in all nine versions).

**Files:**
- Modify: `libs/atlas-packet/note/clientbound/operation_body.go`
- Test: `libs/atlas-packet/note/clientbound/operation_body_test.go` (new)

**Interfaces:**
- Consumes: existing `NewNoteSendError(mode, errorCode byte)`.
- Produces: const `NoteSendErrorNoNoteItem = "NO_NOTE_ITEM"` (used by Tasks 10, 12, 13); corrected `NoteSendErrorBody(errorKey string)` behavior.

- [x] **Step 1: Write the failing test**

Create `libs/atlas-packet/note/clientbound/operation_body_test.go`:

```go
package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestNoteSendErrorBodyUsesSendErrorMode pins that the error body resolves
// the SEND_ERROR operations key (mode 5 here — the GMS v83 value; v48/v61
// tenants resolve 4), not SEND_SUCCESS. Client (CWvsContext::OnMemoResult):
// the SEND_ERROR arm clears the exclusive-request lock then reads one error
// byte, in all nine versions.
func TestNoteSendErrorBodyUsesSendErrorMode(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			"SEND_SUCCESS": float64(4),
			"SEND_ERROR":   float64(5),
		},
		"errors": map[string]interface{}{
			"RECEIVER_UNKNOWN": float64(1),
			"NO_NOTE_ITEM":     float64(3),
		},
	}
	l, _ := testlog.NewNullLogger()

	got := NoteSendErrorBody(NoteSendErrorNoNoteItem)(l, ctx)(options)
	want := []byte{0x05, 0x03}
	if !bytes.Equal(got, want) {
		t.Errorf("NO_NOTE_ITEM body: got %v, want %v", got, want)
	}

	got = NoteSendErrorBody(NoteSendErrorReceiverUnknown)(l, ctx)(options)
	want = []byte{0x05, 0x01}
	if !bytes.Equal(got, want) {
		t.Errorf("RECEIVER_UNKNOWN body: got %v, want %v", got, want)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./note/clientbound/ -run TestNoteSendErrorBodyUsesSendErrorMode -v`
Expected: FAIL — compile error `undefined: NoteSendErrorNoNoteItem`; after adding only the const it would still fail with `got [4 3], want [5 3]`.

- [x] **Step 3: Implement**

In `libs/atlas-packet/note/clientbound/operation_body.go`:

Add to the const block (after `NoteSendErrorReceiverInboxFull`):

```go
	// NoteSendErrorNoNoteItem is sent when a NOTE_ACTION SEND arrives from a
	// character owning no Note item (classification 509), or when the
	// note_send saga fails mid-flight. Tenant templates map it to code 3 —
	// outside the client's 0-2 dialog range: the mode-5 arm clears the
	// exclusive-request lock before decoding the code and shows no dialog
	// for unknown codes (IDA-verified, all nine versions). The SEND_ERROR
	// mode byte carrying this code is version-resolved (4 on v48/v61,
	// 5 on v72+) from the tenant operations table.
	NoteSendErrorNoNoteItem = "NO_NOTE_ITEM"
```

Fix line 38 — change:

```go
			mode := atlas_packet.ResolveCode(l, options, "operations", NoteOperationSendSuccess)
```

to:

```go
			mode := atlas_packet.ResolveCode(l, options, "operations", NoteOperationSendError)
```

- [x] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./note/... -v`
Expected: PASS (existing display/operation tests unaffected — they don't exercise `NoteSendErrorBody`).

- [x] **Step 5: Commit**

```bash
git add libs/atlas-packet/note/clientbound/operation_body.go libs/atlas-packet/note/clientbound/operation_body_test.go
git commit -m "fix(atlas-packet): NoteSendErrorBody resolves SEND_ERROR mode, add NO_NOTE_ITEM key"
```

---

### Task 4: `libs/atlas-saga` — `NoteSend` type, `CreateNote` action, `CreateNotePayload`

**Files:**
- Modify: `libs/atlas-saga/model.go`, `libs/atlas-saga/payloads.go`, `libs/atlas-saga/unmarshal.go`
- Test: `libs/atlas-saga/unmarshal_test.go`

**Interfaces:**
- Produces: `NoteSend Type = "note_send"`; `CreateNote Action = "create_note"`; `CreateNotePayload{SenderId uint32, ReceiverId uint32, Message string, Flag byte}`. Used by Tasks 6–13.

- [x] **Step 1: Write the failing test**

Append to `libs/atlas-saga/unmarshal_test.go`:

```go
func TestCreateNoteStepUnmarshal(t *testing.T) {
	in := Step[any]{
		StepId: "create_note",
		Status: Pending,
		Action: CreateNote,
		Payload: CreateNotePayload{
			SenderId:   100,
			ReceiverId: 200,
			Message:    "hello",
			Flag:       1,
		},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out Step[any]
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	p, ok := out.Payload.(CreateNotePayload)
	if !ok {
		t.Fatalf("payload type: got %T, want CreateNotePayload", out.Payload)
	}
	if p.SenderId != 100 || p.ReceiverId != 200 || p.Message != "hello" || p.Flag != 1 {
		t.Errorf("payload round-trip mismatch: %+v", p)
	}
}
```

(If `encoding/json` is not already imported in the test file, add it.)

- [x] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-saga && go test -run TestCreateNoteStepUnmarshal -v ./...`
Expected: FAIL — compile error `undefined: CreateNote` / `undefined: CreateNotePayload`.

- [x] **Step 3: Implement**

`libs/atlas-saga/model.go` — add to the `Type` const block (after `PetEvolution`):

```go
	NoteSend             Type = "note_send"
```

Add to the `Action` const block (after the `// Field effect actions` group):

```go
	// Note actions
	CreateNote Action = "create_note"
```

`libs/atlas-saga/payloads.go` — append:

```go
// CreateNotePayload represents the payload required to create a note (memo)
// for a receiving character. Emitted by the orchestrator as a note CREATE
// command carrying the saga transaction id.
type CreateNotePayload struct {
	SenderId   uint32 `json:"senderId"`   // Character sending the note
	ReceiverId uint32 `json:"receiverId"` // Character receiving the note
	Message    string `json:"message"`    // Note message text
	Flag       byte   `json:"flag"`       // Note flag (always 1 for player sends)
}
```

`libs/atlas-saga/unmarshal.go` — add a case before the `default:` (after the `FieldEffectWeather` case at line ~497):

```go
	case CreateNote:
		var payload CreateNotePayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
```

- [x] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-saga && go test ./... -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add libs/atlas-saga/model.go libs/atlas-saga/payloads.go libs/atlas-saga/unmarshal.go libs/atlas-saga/unmarshal_test.go
git commit -m "feat(atlas-saga): note_send saga type, create_note action + payload"
```

---

### Task 5: atlas-notes — transaction id threading + CREATE_FAILED event

atlas-notes today emits `CREATED` with no transaction id and swallows create errors (`kafka/consumer/note/consumer.go:50` discards both return values). The orchestrator needs: `CREATED`/`CREATE_FAILED` correlated by transaction id.

**Files:**
- Modify: `services/atlas-notes/atlas.com/notes/kafka/message/note/kafka.go`
- Modify: `services/atlas-notes/atlas.com/notes/note/processor.go` (Create/CreateAndEmit signatures)
- Modify: `services/atlas-notes/atlas.com/notes/note/producer.go`
- Modify: `services/atlas-notes/atlas.com/notes/note/mock/processor.go`
- Modify: `services/atlas-notes/atlas.com/notes/kafka/consumer/note/consumer.go`
- Modify: `services/atlas-notes/atlas.com/notes/note/resource.go` (caller: pass `uuid.Nil`)
- Test: `services/atlas-notes/atlas.com/notes/note/processor_test.go` (update existing call sites; add txn assertions), `services/atlas-notes/atlas.com/notes/note/producer_test.go` (new)

**Interfaces:**
- Consumes: `CreateNotePayload` semantics from Task 4 (message shape only — no Go dependency).
- Produces (JSON contract Task 6 mirrors): `Command[E]`/`StatusEvent[E]` envelopes gain `TransactionId uuid.UUID \`json:"transactionId,omitempty"\``; new `StatusEventTypeCreateFailed = "CREATE_FAILED"` with `StatusEventCreateFailedBody{SenderId uint32, Reason string}`.
- Produces (Go): `Processor.Create(mb) func(transactionId uuid.UUID) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error)`; `CreateAndEmit(transactionId uuid.UUID, characterId uint32, senderId uint32, msg string, flag byte) (Model, error)`; `CreateNoteStatusEventProvider(transactionId uuid.UUID, characterId uint32, noteId uint32, senderId uint32, msg string, flag byte, timestamp time.Time)`; `CreateFailedStatusEventProvider(transactionId uuid.UUID, characterId uint32, senderId uint32, reason string)`.

- [x] **Step 1: Write the failing producer test**

Create `services/atlas-notes/atlas.com/notes/note/producer_test.go`:

```go
package note

import (
	"encoding/json"
	"testing"
	"time"

	note2 "atlas-notes/kafka/message/note"

	"github.com/google/uuid"
)

func TestCreateNoteStatusEventProviderCarriesTransactionId(t *testing.T) {
	txn := uuid.New()
	msgs, err := CreateNoteStatusEventProvider(txn, 200, 7, 100, "hello", 1, time.Now())()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages: got %d, want 1", len(msgs))
	}
	var e note2.StatusEvent[note2.StatusEventCreatedBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.TransactionId != txn {
		t.Errorf("transactionId: got %s, want %s", e.TransactionId, txn)
	}
	if e.Type != note2.StatusEventTypeCreated {
		t.Errorf("type: got %s, want %s", e.Type, note2.StatusEventTypeCreated)
	}
	if e.CharacterId != 200 || e.Body.NoteId != 7 || e.Body.SenderId != 100 {
		t.Errorf("body mismatch: %+v", e)
	}
}

func TestCreateFailedStatusEventProvider(t *testing.T) {
	txn := uuid.New()
	msgs, err := CreateFailedStatusEventProvider(txn, 200, 100, "db down")()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	var e note2.StatusEvent[note2.StatusEventCreateFailedBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.TransactionId != txn {
		t.Errorf("transactionId: got %s, want %s", e.TransactionId, txn)
	}
	if e.Type != note2.StatusEventTypeCreateFailed {
		t.Errorf("type: got %s, want %s", e.Type, note2.StatusEventTypeCreateFailed)
	}
	if e.Body.SenderId != 100 || e.Body.Reason != "db down" {
		t.Errorf("body mismatch: %+v", e.Body)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-notes/atlas.com/notes && go test ./note/ -run "TestCreateNoteStatusEventProviderCarriesTransactionId|TestCreateFailedStatusEventProvider" -v`
Expected: FAIL — compile errors (provider signatures, missing const/body type).

- [x] **Step 3: Implement the message contract**

`services/atlas-notes/atlas.com/notes/kafka/message/note/kafka.go`:

Add `"github.com/google/uuid"` to imports. Add to the const block:

```go
	StatusEventTypeCreateFailed = "CREATE_FAILED"
```

Add `TransactionId` as the FIRST field of both envelopes:

```go
// Command represents a Kafka command for note operations
type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId,omitempty"` // Saga transaction id (uuid.Nil when not saga-driven)
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}
```

```go
// StatusEvent represents a Kafka status event for note operations
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId,omitempty"` // Saga transaction id (uuid.Nil when not saga-driven)
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}
```

Append the failure body:

```go
// StatusEventCreateFailedBody contains data for a note create failure event
type StatusEventCreateFailedBody struct {
	SenderId uint32 `json:"senderId"`
	Reason   string `json:"reason"`
}
```

- [x] **Step 4: Implement producer changes**

`services/atlas-notes/atlas.com/notes/note/producer.go` — change `CreateNoteStatusEventProvider` to:

```go
// CreateNoteStatusEventProvider creates a status event for note creation.
// transactionId is uuid.Nil for non-saga creations (REST).
func CreateNoteStatusEventProvider(transactionId uuid.UUID, characterId uint32, noteId uint32, senderId uint32, msg string, flag byte, timestamp time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	body := note.StatusEventCreatedBody{
		NoteId:   noteId,
		SenderId: senderId,
		Message:  msg,
		Flag:     flag,
		Time:     timestamp,
	}
	value := note.StatusEvent[note.StatusEventCreatedBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          note.StatusEventTypeCreated,
		Body:          body,
	}
	return producer.SingleMessageProvider(key, value)
}
```

Append:

```go
// CreateFailedStatusEventProvider creates a status event for a failed note
// creation, so the saga orchestrator can fail the create_note step and
// compensate the consumed Note item.
func CreateFailedStatusEventProvider(transactionId uuid.UUID, characterId uint32, senderId uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := note.StatusEvent[note.StatusEventCreateFailedBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          note.StatusEventTypeCreateFailed,
		Body: note.StatusEventCreateFailedBody{
			SenderId: senderId,
			Reason:   reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

Add `"github.com/google/uuid"` to the import block.

- [x] **Step 5: Implement processor changes**

`services/atlas-notes/atlas.com/notes/note/processor.go`:

Interface (lines 19–20) becomes:

```go
	Create(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error)
	CreateAndEmit(transactionId uuid.UUID, characterId uint32, senderId uint32, msg string, flag byte) (Model, error)
```

`Create` implementation — wrap the existing chain with the new leading param and thread the id into the event provider:

```go
// Create creates a new note. transactionId is uuid.Nil for non-saga creations.
func (p *ProcessorImpl) Create(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error) {
	return func(transactionId uuid.UUID) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error) {
		return func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error) {
			return func(senderId uint32) func(msg string) func(flag byte) (Model, error) {
				return func(msg string) func(flag byte) (Model, error) {
					return func(flag byte) (Model, error) {
						m, err := NewBuilder().
							SetCharacterId(characterId).
							SetSenderId(senderId).
							SetMessage(msg).
							SetFlag(flag).
							Build()
						if err != nil {
							return Model{}, err
						}

						m, err = createNote(p.db.WithContext(p.ctx), p.t.Id(), m)
						if err != nil {
							return Model{}, err
						}
						err = mb.Put(note.EnvEventTopicNoteStatus, CreateNoteStatusEventProvider(transactionId, m.CharacterId(), m.Id(), m.SenderId(), m.Message(), m.Flag(), m.Timestamp()))
						if err != nil {
							return Model{}, err
						}
						return m, nil
					}
				}
			}
		}
	}
}

// CreateAndEmit creates a new note and emits a status event
func (p *ProcessorImpl) CreateAndEmit(transactionId uuid.UUID, characterId uint32, senderId uint32, msg string, flag byte) (Model, error) {
	return message.EmitWithResult[Model, byte](p.producer)(func(mb *message.Buffer) func(flag byte) (Model, error) {
		return p.Create(mb)(transactionId)(characterId)(senderId)(msg)
	})(flag)
}
```

Add `"github.com/google/uuid"` to the import block.

- [x] **Step 6: Update callers and mock**

`services/atlas-notes/atlas.com/notes/kafka/consumer/note/consumer.go` — replace `handleNoteCreate` body:

```go
func handleNoteCreate(db *gorm.DB) message.Handler[note2.Command[note2.CommandCreateBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c note2.Command[note2.CommandCreateBody]) {
		if c.Type != note2.CommandTypeCreate {
			return
		}

		_, err := note.NewProcessor(l, ctx, db).CreateAndEmit(c.TransactionId, c.CharacterId, c.Body.SenderId, c.Body.Message, c.Body.Flag)
		if err != nil {
			l.WithError(err).Errorf("Unable to create note for character [%d] from sender [%d].", c.CharacterId, c.Body.SenderId)
			emitErr := producer.ProviderImpl(l)(ctx)(note2.EnvEventTopicNoteStatus)(note.CreateFailedStatusEventProvider(c.TransactionId, c.CharacterId, c.Body.SenderId, err.Error()))
			if emitErr != nil {
				l.WithError(emitErr).Errorf("Unable to emit CREATE_FAILED for transaction [%s].", c.TransactionId)
			}
		}
	}
}
```

Add `"atlas-notes/kafka/producer"` to the consumer's imports.

`services/atlas-notes/atlas.com/notes/note/resource.go:125` — REST creation is not saga-driven:

```go
		m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).CreateAndEmit(uuid.Nil, im.CharacterId(), im.SenderId(), im.Message(), im.Flag())
```

(add `"github.com/google/uuid"` import).

`services/atlas-notes/atlas.com/notes/note/mock/processor.go` — update `CreateFunc`/`CreateAndEmitFunc` field types and the two methods to the new signatures (mechanical: insert the `transactionId uuid.UUID` leading param exactly as in the interface; add `"github.com/google/uuid"` import).

Update every existing `Create(`/`CreateAndEmit(` call site in `processor_test.go` (and any other `_test.go` in the module — find them with `grep -rn "CreateAndEmit(\|Create(mb)" services/atlas-notes --include="*_test.go"`) to pass `uuid.Nil` (or a fresh `uuid.New()` where the test then asserts the id round-trips).

- [x] **Step 7: Run tests to verify they pass**

Run: `cd services/atlas-notes/atlas.com/notes && go test -race ./... && go vet ./...`
Expected: PASS, no vet findings.

- [x] **Step 8: Commit**

```bash
git add services/atlas-notes/
git commit -m "feat(atlas-notes): thread saga transaction id through note create, emit CREATE_FAILED"
```

---

### Task 6: Orchestrator — note message defs, note processor + mock

**Files:**
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/note/kafka.go`
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/note/processor.go`
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/note/producer.go`
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/note/mock/processor.go`
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/note/producer_test.go`

**Interfaces:**
- Consumes: JSON contract from Task 5 (field-for-field mirror; no Go import of atlas-notes).
- Produces: `note.Processor` with `CreateNote(transactionId uuid.UUID, receiverId uint32, senderId uint32, message string, flag byte) error`; message types `note.Command[E]`, `note.StatusEvent[E]`, `note.StatusEventCreatedBody`, `note.StatusEventCreateFailedBody`, consts `EnvCommandTopic`, `EnvEventTopicNoteStatus`, `CommandTypeCreate`, `StatusEventTypeCreated`, `StatusEventTypeCreateFailed`. Used by Tasks 7–8.

- [x] **Step 1: Write the failing test**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/note/producer_test.go`:

```go
package note

import (
	"encoding/json"
	"testing"

	note2 "atlas-saga-orchestrator/kafka/message/note"

	"github.com/google/uuid"
)

func TestCreateNoteCommandProvider(t *testing.T) {
	txn := uuid.New()
	msgs, err := CreateNoteCommandProvider(txn, 200, 100, "hello", 1)()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages: got %d, want 1", len(msgs))
	}
	var c note2.Command[note2.CommandCreateBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.TransactionId != txn {
		t.Errorf("transactionId: got %s, want %s", c.TransactionId, txn)
	}
	if c.Type != note2.CommandTypeCreate {
		t.Errorf("type: got %s, want %s", c.Type, note2.CommandTypeCreate)
	}
	if c.CharacterId != 200 {
		t.Errorf("characterId (receiver): got %d, want 200", c.CharacterId)
	}
	if c.Body.SenderId != 100 || c.Body.Message != "hello" || c.Body.Flag != 1 {
		t.Errorf("body mismatch: %+v", c.Body)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./note/ -v`
Expected: FAIL — package does not exist yet.

- [x] **Step 3: Implement the message defs**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/note/kafka.go` (field-for-field JSON mirror of atlas-notes' `kafka/message/note/kafka.go` as changed in Task 5):

```go
package note

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic         = "COMMAND_TOPIC_NOTE"
	EnvEventTopicNoteStatus = "EVENT_TOPIC_NOTE_STATUS"

	CommandTypeCreate = "CREATE"

	StatusEventTypeCreated      = "CREATED"
	StatusEventTypeCreateFailed = "CREATE_FAILED"
)

// Command mirrors atlas-notes' note command envelope. WorldId/ChannelId are
// unused by the CREATE handler (atlas-notes kafka/consumer/note/consumer.go)
// and are zero for orchestrator-emitted commands.
type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId,omitempty"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

// CommandCreateBody contains data for creating a note
type CommandCreateBody struct {
	SenderId uint32 `json:"senderId"`
	Message  string `json:"message"`
	Flag     byte   `json:"flag"`
}

// StatusEvent mirrors atlas-notes' note status event envelope.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// StatusEventCreatedBody contains data for a note created event. Only the
// fields the orchestrator matches on are declared; extra fields on the wire
// are ignored by encoding/json.
type StatusEventCreatedBody struct {
	NoteId   uint32 `json:"noteId"`
	SenderId uint32 `json:"senderId"`
}

// StatusEventCreateFailedBody contains data for a note create failure event
type StatusEventCreateFailedBody struct {
	SenderId uint32 `json:"senderId"`
	Reason   string `json:"reason"`
}
```

- [x] **Step 4: Implement processor, producer, mock**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/note/producer.go`:

```go
package note

import (
	note2 "atlas-saga-orchestrator/kafka/message/note"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/google/uuid"
)

// CreateNoteCommandProvider builds the note CREATE command carrying the saga
// transaction id. CharacterId is the RECEIVER (atlas-notes stores notes keyed
// by the receiving character).
func CreateNoteCommandProvider(transactionId uuid.UUID, receiverId uint32, senderId uint32, message string, flag byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(receiverId))
	value := &note2.Command[note2.CommandCreateBody]{
		TransactionId: transactionId,
		CharacterId:   receiverId,
		Type:          note2.CommandTypeCreate,
		Body: note2.CommandCreateBody{
			SenderId: senderId,
			Message:  message,
			Flag:     flag,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/note/processor.go`:

```go
package note

import (
	note2 "atlas-saga-orchestrator/kafka/message/note"
	"atlas-saga-orchestrator/kafka/producer"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	CreateNote(transactionId uuid.UUID, receiverId uint32, senderId uint32, message string, flag byte) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// CreateNote emits the note CREATE command for the create_note saga step.
// The step completes when atlas-notes' CREATED/CREATE_FAILED status event
// arrives (kafka/consumer/note/consumer.go).
func (p *ProcessorImpl) CreateNote(transactionId uuid.UUID, receiverId uint32, senderId uint32, message string, flag byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(note2.EnvCommandTopic)(CreateNoteCommandProvider(transactionId, receiverId, senderId, message, flag))
}
```

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/note/mock/processor.go`:

```go
package mock

import (
	"github.com/google/uuid"
)

// ProcessorMock is a function-field mock of note.Processor.
type ProcessorMock struct {
	CreateNoteFunc func(transactionId uuid.UUID, receiverId uint32, senderId uint32, message string, flag byte) error
}

func (m *ProcessorMock) CreateNote(transactionId uuid.UUID, receiverId uint32, senderId uint32, message string, flag byte) error {
	if m.CreateNoteFunc != nil {
		return m.CreateNoteFunc(transactionId, receiverId, senderId, message, flag)
	}
	return nil
}
```

- [x] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./note/... -v && go vet ./note/... ./kafka/message/note/`
Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/note/ services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/note/
git commit -m "feat(atlas-saga-orchestrator): note command message defs + note processor"
```

---

### Task 7: Orchestrator — `CreateNote` action wiring (model, handler, event acceptance)

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` (re-exports + step-unmarshal case)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` (interface method, `noteP` field, `WithNoteProcessor`, `GetHandler` case, `handleCreateNote`)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` (kinds + tables)
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go`

**Interfaces:**
- Consumes: `note.Processor` / `note/mock.ProcessorMock` (Task 6); `sharedsaga.NoteSend`, `sharedsaga.CreateNote`, `sharedsaga.CreateNotePayload` (Task 4).
- Produces: saga-package re-exports `NoteSend`, `CreateNote`, `CreateNotePayload`; `EventKindNoteCreated EventKind = "note.created"`, `EventKindNoteCreateFailed EventKind = "note.create_failed"`; `Handler.WithNoteProcessor(note.Processor) Handler`. Used by Tasks 8–9.

- [x] **Step 1: Write the failing test**

Append to `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go` (same style as `TestHandleDestroyAsset` at line 681; `notemock` import is `notemock "atlas-saga-orchestrator/note/mock"`):

```go
func TestHandleCreateNote(t *testing.T) {
	tests := []struct {
		name        string
		payload     CreateNotePayload
		mockError   error
		expectError bool
	}{
		{
			name:        "Success case",
			payload:     CreateNotePayload{SenderId: 100, ReceiverId: 200, Message: "hello", Flag: 1},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "Error case",
			payload:     CreateNotePayload{SenderId: 100, ReceiverId: 200, Message: "hello", Flag: 1},
			mockError:   errors.New("kafka down"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noteP := &notemock.ProcessorMock{}

			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			_, ctx := setupContext()

			transactionId := uuid.New()
			noteP.CreateNoteFunc = func(txn uuid.UUID, receiverId uint32, senderId uint32, message string, flag byte) error {
				assert.Equal(t, transactionId, txn)
				assert.Equal(t, tt.payload.ReceiverId, receiverId)
				assert.Equal(t, tt.payload.SenderId, senderId)
				assert.Equal(t, tt.payload.Message, message)
				assert.Equal(t, tt.payload.Flag, flag)
				return tt.mockError
			}

			s, err := NewBuilder().
				SetTransactionId(transactionId).
				SetSagaType(NoteSend).
				SetInitiatedBy("test").
				Build()
			assert.NoError(t, err)

			step := NewStep[any]("create_note", Pending, CreateNote, tt.payload)

			err = NewHandler(logger, ctx).WithNoteProcessor(noteP).handleCreateNote(s, step)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run TestHandleCreateNote -v`
Expected: FAIL — compile errors (`NoteSend`, `CreateNotePayload`, `WithNoteProcessor`, `handleCreateNote` undefined).

- [x] **Step 3: Implement model re-exports + unmarshal case**

`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go`:

Add to the saga-type const block (after `PetEvolution`):

```go
	NoteSend             = sharedsaga.NoteSend
```

Add to the action const block (after the `// Field effect actions` group's `FieldEffectWeather` line, keeping the `RebalanceStat*` group last):

```go
	// Note actions
	CreateNote = sharedsaga.CreateNote
```

Add to the payload-type re-export block:

```go
	CreateNotePayload                   = sharedsaga.CreateNotePayload
```

Add a case to the step-payload unmarshal switch (next to the `FieldEffectWeather` case at line ~1382):

```go
	case CreateNote:
		var payload CreateNotePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
```

- [x] **Step 4: Implement event-acceptance entries**

`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go`:

Add to the `EventKind` const block:

```go
	// Note.
	EventKindNoteCreated      EventKind = "note.created"
	EventKindNoteCreateFailed EventKind = "note.create_failed"
```

Add to `acceptanceTable` (near the skills entries):

```go
	// Note.
	sharedsaga.CreateNote: {EventKindNoteCreated, EventKindNoteCreateFailed},
```

Add to `outcomeTable`:

```go
	// Note.
	EventKindNoteCreated:      OutcomeSuccess,
	EventKindNoteCreateFailed: OutcomeFailure,
```

- [x] **Step 5: Implement handler**

`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`:

1. Import `"atlas-saga-orchestrator/note"`.
2. Add to the `Handler` interface (next to the other `With*` methods): `WithNoteProcessor(note.Processor) Handler`; and to the handler-method list: `handleCreateNote(s Saga, st Step[any]) error`.
3. Add field `noteP note.Processor` to `HandlerImpl` and `noteP: note.NewProcessor(l, ctx),` to `NewHandler`.
4. Add the `WithNoteProcessor` method (the existing `With*` methods deliberately clone only the fields their tests need — follow that established style):

```go
func (h *HandlerImpl) WithNoteProcessor(noteP note.Processor) Handler {
	return &HandlerImpl{
		l:     h.l,
		ctx:   h.ctx,
		t:     h.t,
		noteP: noteP,
	}
}
```

5. Add to the `GetHandler` switch (near `FieldEffectWeather` at line ~860):

```go
	case CreateNote:
		return h.handleCreateNote, true
```

6. Append the handler implementation (near `handleFieldEffectWeather`):

```go
// handleCreateNote handles the CreateNote action for note_send sagas. It
// emits the note CREATE command carrying the saga transaction id and does
// NOT mark the step complete — completion arrives via atlas-notes'
// CREATED/CREATE_FAILED status event (kafka/consumer/note/consumer.go).
func (h *HandlerImpl) handleCreateNote(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(CreateNotePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	h.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"sender_id":      payload.SenderId,
		"receiver_id":    payload.ReceiverId,
	}).Debug("Requesting note creation.")

	err := h.noteP.CreateNote(s.TransactionId(), payload.ReceiverId, payload.SenderId, payload.Message, payload.Flag)
	if err != nil {
		h.logActionError(s, st, err, "Unable to request note creation.")
		return err
	}
	return nil
}
```

- [x] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run "TestHandleCreateNote" -v`
Expected: PASS.

Then run the whole saga package — the completeness/coverage tests (`event_acceptance_test.go`, `unmarshal_completeness_test.go`) enumerate all actions and will fail if any wiring above was missed:

Run: `go test ./saga/ 2>&1 | tail -5`
Expected: `ok` (if a completeness test names `create_note`, the missing entry is in this task's scope — fix it here, don't defer).

- [x] **Step 7: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/
git commit -m "feat(atlas-saga-orchestrator): CreateNote action handler + event acceptance wiring"
```

---

### Task 8: Orchestrator — note status consumer + main registration

**Files:**
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/note/consumer.go`
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/note/consumer_test.go`, plus `testmain_test.go` copied convention
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go`

**Interfaces:**
- Consumes: `note2.StatusEvent[...]` (Task 6), `saga.AcceptEvent(txnId, EventKind) (AcceptDecision, bool)`, `saga.StepCompleted(txnId, success bool)`, `EventKindNoteCreated`/`EventKindNoteCreateFailed` (Task 7).
- Produces: `note.InitConsumers` / `note.InitHandlers` registered in `main.go`.

- [x] **Step 1: Write the failing test**

First copy the test bootstrap convention: look at `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/pet/testmain_test.go` and create the equivalent `kafka/consumer/note/testmain_test.go` with package name `note` (copy the file, change only the package clause).

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/note/consumer_test.go` (mirrors `kafka/consumer/pet/consumer_test.go`):

```go
package note

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	note2 "atlas-saga-orchestrator/kafka/message/note"
	"atlas-saga-orchestrator/saga"
)

func mustTenantCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func putTestSaga(t *testing.T, ctx context.Context, s saga.Saga) {
	t.Helper()
	require.NoError(t, saga.GetCache().Put(ctx, s))
}

// noteSendSaga builds a note_send saga whose destroy step is already
// completed, leaving create_note as the earliest pending step.
func noteSendSaga(t *testing.T, tx uuid.UUID) saga.Saga {
	t.Helper()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.NoteSend).
		SetInitiatedBy("test").
		AddStep("consume_note_item", saga.Completed, saga.DestroyAsset, saga.DestroyAssetPayload{CharacterId: 100, TemplateId: 5090000, Quantity: 1}).
		AddStep("create_note", saga.Pending, saga.CreateNote, saga.CreateNotePayload{SenderId: 100, ReceiverId: 200, Message: "hi", Flag: 1}).
		Build()
	require.NoError(t, err)
	return s
}

func TestHandleCreatedEvent_CompletesCreateNoteStep(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	putTestSaga(t, ctx, noteSendSaga(t, tx))

	handleCreatedEvent(logger, ctx, note2.StatusEvent[note2.StatusEventCreatedBody]{
		TransactionId: tx,
		CharacterId:   200,
		Type:          note2.StatusEventTypeCreated,
		Body:          note2.StatusEventCreatedBody{NoteId: 7, SenderId: 100},
	})

	// Completing the final step drives the saga terminal. Depending on how far
	// the completion path runs in the test environment (the Kafka emission is
	// best-effort here), the saga is either evicted from the cache or its
	// create_note step is no longer pending. A still-pending step means the
	// event was NOT accepted — the failure this test guards against.
	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	if err == nil {
		assert.NotEqual(t, saga.Pending, got.Steps()[1].Status(), "create_note must not remain pending after CREATED event")
	}
}

func TestHandleCreatedEvent_IgnoresNilTransactionId(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	putTestSaga(t, ctx, noteSendSaga(t, tx))

	handleCreatedEvent(logger, ctx, note2.StatusEvent[note2.StatusEventCreatedBody]{
		TransactionId: uuid.Nil,
		CharacterId:   200,
		Type:          note2.StatusEventTypeCreated,
		Body:          note2.StatusEventCreatedBody{NoteId: 7, SenderId: 100},
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Pending, got.Steps()[1].Status(), "nil-txn event must not complete anything")
}

func TestHandleCreateFailedEvent_FailsCreateNoteStep(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	putTestSaga(t, ctx, noteSendSaga(t, tx))

	handleCreateFailedEvent(logger, ctx, note2.StatusEvent[note2.StatusEventCreateFailedBody]{
		TransactionId: tx,
		CharacterId:   200,
		Type:          note2.StatusEventTypeCreateFailed,
		Body:          note2.StatusEventCreateFailedBody{SenderId: 100, Reason: "db down"},
	})

	// StepCompleted(false) routes into compensation; the saga must no longer
	// have create_note pending (it is either failed-and-compensating or the
	// saga is already terminal/evicted).
	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	if err == nil {
		assert.NotEqual(t, saga.Pending, got.Steps()[1].Status())
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./kafka/consumer/note/ -v`
Expected: FAIL — `handleCreatedEvent`/`handleCreateFailedEvent` undefined.

NOTE: `TestHandleCreateFailedEvent_FailsCreateNoteStep` exercises the compensation path, which is implemented in Task 9. If it fails here for a compensation-specific reason (not a compile error), mark it with `t.Skip("compensation lands in Task 9")` and REMOVE the skip in Task 9 Step 4. Do not leave the skip in the final tree.

- [x] **Step 3: Implement**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/note/consumer.go` (pattern: `kafka/consumer/pet/consumer.go`):

```go
package note

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	note2 "atlas-saga-orchestrator/kafka/message/note"
	"atlas-saga-orchestrator/saga"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("note_status_event")(note2.EnvEventTopicNoteStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(note2.EnvEventTopicNoteStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreatedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateFailedEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleCreatedEvent(l logrus.FieldLogger, ctx context.Context, e note2.StatusEvent[note2.StatusEventCreatedBody]) {
	if e.Type != note2.StatusEventTypeCreated {
		return
	}

	// Skip events without a transaction id (REST-created / non-saga notes).
	if e.TransactionId == uuid.Nil {
		return
	}

	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindNoteCreated); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"note_id":        e.Body.NoteId,
		"receiver_id":    e.CharacterId,
	}).Debug("Note created, marking saga step as completed.")

	_ = p.StepCompleted(e.TransactionId, true)
}

func handleCreateFailedEvent(l logrus.FieldLogger, ctx context.Context, e note2.StatusEvent[note2.StatusEventCreateFailedBody]) {
	if e.Type != note2.StatusEventTypeCreateFailed {
		return
	}

	if e.TransactionId == uuid.Nil {
		return
	}

	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindNoteCreateFailed); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"receiver_id":    e.CharacterId,
		"reason":         e.Body.Reason,
	}).Warn("Note creation failed, failing saga step.")

	_ = p.StepCompleted(e.TransactionId, false)
}
```

Register in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go` — alongside the existing blocks (lines ~90–150): import the package as `noteConsumer "atlas-saga-orchestrator/kafka/consumer/note"`, add `noteConsumer.InitConsumers(l)(cmf)(consumerGroupId)` next to `pet.InitConsumers…`, and the matching

```go
	if err := noteConsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to init note handlers.")
	}
```

(match the exact error-handling phrasing of the adjacent blocks).

- [x] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./kafka/consumer/note/ -v && go build ./...`
Expected: PASS (modulo the possible Task-9 skip), build clean.

- [x] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/note/ services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go
git commit -m "feat(atlas-saga-orchestrator): note status consumer completes/fails create_note steps"
```

---

### Task 9: Orchestrator — note_send completed Results + compensation

Two pieces: (a) the COMPLETED status event must carry the sender's characterId so the channel can announce SEND_SUCCESS (the completed body has no CharacterId field; `Results` is the existing mechanism, see `extractCharacterCreationResults`); (b) a `note_send` branch in `CompensateFailedStep` that re-awards the destroyed Note item when `create_note` fails (mirrors `compensatePetEvolution`).

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/producer.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go`
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/producer_test.go` (or new `producer_note_test.go` if none exists), `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator_test.go`

**Interfaces:**
- Consumes: `NoteSend`, `CreateNote`, `CreateNotePayload` (Task 7); `c.compP.RequestCreateItem(transactionId, characterId, templateId, quantity, time.Time{})` (existing); `EmitSagaFailedByIds` (existing).
- Produces: COMPLETED events for note_send sagas carry `Body.Results["characterId"] = <senderId>` (consumed by Task 10).

- [x] **Step 1: Write the failing completed-results test**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/producer_note_test.go`:

```go
func TestCompletedStatusEventProviderNoteSendResults(t *testing.T) {
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(NoteSend).
		SetInitiatedBy("test").
		AddStep("consume_note_item", Completed, DestroyAsset, DestroyAssetPayload{CharacterId: 100, TemplateId: 5090000, Quantity: 1}).
		AddStep("create_note", Completed, CreateNote, CreateNotePayload{SenderId: 100, ReceiverId: 200, Message: "hi", Flag: 1}).
		Build()
	require.NoError(t, err)

	msgs, err := CompletedStatusEventProvider(s)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var e sagaMsg.StatusEvent[sagaMsg.StatusEventCompletedBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &e))
	assert.Equal(t, "note_send", e.Body.SagaType)
	require.NotNil(t, e.Body.Results)
	assert.Equal(t, float64(100), e.Body.Results["characterId"], "sender characterId must ride in Results")
}
```

(`sagaMsg` = the import alias for `atlas-saga-orchestrator/kafka/message/saga` already used in this package — check the existing alias name in `producer.go`/`compensator.go` and reuse it; import `encoding/json`, `github.com/google/uuid`, `github.com/stretchr/testify/assert`, `github.com/stretchr/testify/require`.)

- [x] **Step 2: Write the failing compensation test**

The Kafka emission half of compensation is not testable without a broker, so — exactly like pet evolution (`DispatchPetEvolutionRollbacks`, tested by `TestPetEvolutionCompensationRefundsResources` at `compensator_test.go:37`) — the implementation splits into a pure dispatch half (`DispatchNoteSendRollbacks`) and a terminal half (`compensateNoteSend`), and the tests exercise the dispatch half directly with a spy mock.

Append to `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator_test.go`:

```go
// TestNoteSendCompensationRefundsItem: create_note failed after the destroy
// completed → the reverse-walk must re-award the destroyed Note item via
// RequestCreateItem, exactly once, with the destroy step's template id.
func TestNoteSendCompensationRefundsItem(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	const (
		senderId = uint32(100)
		noteItem = uint32(5090000)
	)

	type createCall struct {
		CharacterId uint32
		TemplateId  uint32
		Quantity    uint32
	}
	var createItemCalls []createCall
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, characterId uint32, templateId uint32, quantity uint32, _ time.Time) error {
			createItemCalls = append(createItemCalls, createCall{
				CharacterId: characterId,
				TemplateId:  templateId,
				Quantity:    quantity,
			})
			return nil
		},
	}

	transactionId := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(NoteSend).
		SetInitiatedBy("note-send-compensation-test").
		AddStep("consume_note_item", Completed, DestroyAsset, DestroyAssetPayload{
			CharacterId: senderId,
			TemplateId:  noteItem,
			Quantity:    1,
			RemoveAll:   false,
		}).
		AddStep("create_note", Failed, CreateNote, CreateNotePayload{
			SenderId:   senderId,
			ReceiverId: 200,
			Message:    "hi",
			Flag:       1,
		}).
		Build()
	assert.NoError(t, err, "saga build should not fail")

	compensator := NewCompensator(logger, tctx).
		WithCompartmentProcessor(compP)

	compensator.DispatchNoteSendRollbacks(s)

	assert.Equal(t, 1, len(createItemCalls), "Note item should be refunded exactly once")
	if len(createItemCalls) == 1 {
		assert.Equal(t, senderId, createItemCalls[0].CharacterId, "refund must target the sender")
		assert.Equal(t, noteItem, createItemCalls[0].TemplateId, "refunded item must be the consumed Note item")
		assert.Equal(t, uint32(1), createItemCalls[0].Quantity, "refunded quantity must be 1")
	}
}

// TestNoteSendCompensationDestroyFailedNoRefund: the DestroyAsset step itself
// failed (nothing completed) → there is nothing to re-award; the reverse-walk
// must NOT dispatch RequestCreateItem.
func TestNoteSendCompensationDestroyFailedNoRefund(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	var createItemCalls int
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error {
			createItemCalls++
			return nil
		},
	}

	transactionId := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(NoteSend).
		SetInitiatedBy("note-send-compensation-test").
		AddStep("consume_note_item", Failed, DestroyAsset, DestroyAssetPayload{
			CharacterId: 100,
			TemplateId:  5090000,
			Quantity:    1,
			RemoveAll:   false,
		}).
		AddStep("create_note", Pending, CreateNote, CreateNotePayload{
			SenderId:   100,
			ReceiverId: 200,
			Message:    "hi",
			Flag:       1,
		}).
		Build()
	assert.NoError(t, err, "saga build should not fail")

	compensator := NewCompensator(logger, tctx).
		WithCompartmentProcessor(compP)

	compensator.DispatchNoteSendRollbacks(s)

	assert.Equal(t, 0, createItemCalls, "no completed destroy → no refund dispatch")
}
```

- [x] **Step 3: Run tests to verify they fail**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run "TestCompletedStatusEventProviderNoteSendResults|TestNoteSendCompensation" -v`
Expected: FAIL — Results empty for note_send; note_send falls into the default "No compensation logic available" branch (which re-marks the step Pending instead of terminating).

- [x] **Step 4: Implement**

`saga/producer.go` — in `CompletedStatusEventProvider`, after the CharacterCreation block:

```go
	// For NoteSend sagas, surface the sender's characterId so atlas-channel
	// can announce MEMO_RESULT SEND_SUCCESS to the sender's session.
	if s.SagaType() == NoteSend {
		body.Results = extractNoteSendResults(s)
	}
```

Append:

```go
// extractNoteSendResults extracts the sending character's id from a
// NoteSend saga's CreateNote step payload.
func extractNoteSendResults(s Saga) map[string]any {
	for _, step := range s.Steps() {
		if step.Action() != CreateNote {
			continue
		}
		if p, ok := step.Payload().(CreateNotePayload); ok {
			return map[string]any{"characterId": p.SenderId}
		}
	}
	return nil
}
```

`saga/compensator.go` — in `CompensateFailedStep`, after the `PetEvolution` branch (line ~235):

```go
	// Note-send reverse-walk: a failed create_note must refund the
	// already-destroyed Note item; a failed consume_note_item has nothing to
	// refund. Either way the saga terminates with one Failed emission so the
	// channel can announce SEND_ERROR (unlocking the client).
	if s.SagaType() == NoteSend {
		return c.compensateNoteSend(s, failedStep)
	}
```

Add `DispatchNoteSendRollbacks(s Saga)` to the `Compensator` interface declaration (next to `DispatchPetEvolutionRollbacks`), then append the implementations (mirror `compensatePetEvolution` / `DispatchPetEvolutionRollbacks`, lines 1093–1180):

```go
// DispatchNoteSendRollbacks reverse-walks a note_send saga's completed steps
// and refunds the destroyed Note item (DestroyAsset → RequestCreateItem).
// Pure dispatch half — no lifecycle transitions, no event emission, no cache
// eviction; callers own those. An error refunding does not abort the walk.
func (c *CompensatorImpl) DispatchNoteSendRollbacks(s Saga) {
	steps := s.Steps()
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Status() != Completed {
			continue
		}
		if step.Action() != DestroyAsset {
			continue
		}
		if payload, ok := step.Payload().(DestroyAssetPayload); ok {
			qty := payload.Quantity
			if qty == 0 {
				qty = 1
			}
			if err := c.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, time.Time{}); err != nil {
				c.l.WithError(err).WithFields(logrus.Fields{
					"transaction_id": s.TransactionId().String(),
					"step_id":        step.StepId(),
					"template_id":    payload.TemplateId,
				}).Error("NoteSend compensation: DestroyAsset → CreateItem dispatch failed; continuing.")
			}
		}
	}
}

// compensateNoteSend terminates a failing note_send saga: dispatches the
// item refund, then emits exactly one StatusEventTypeFailed carrying the
// SENDER's characterId (the channel's saga consumer announces MEMO_RESULT
// SEND_ERROR to that session, which also releases the client's
// exclusive-request lock). Double-emission is prevented by
// TryTransition(Compensating → Failed), mirroring compensatePetEvolution.
func (c *CompensatorImpl) compensateNoteSend(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"failed_step":    failedStep.StepId(),
		"failed_action":  failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Info("NoteSend saga failing — dispatching compensation.")

	c.DispatchNoteSendRollbacks(s)

	// The sender's characterId rides in the Failed event so the channel can
	// notify the right session.
	var senderId uint32
	for _, step := range s.Steps() {
		if p, ok := step.Payload().(CreateNotePayload); ok {
			senderId = p.SenderId
			break
		}
	}
	if senderId == 0 {
		if payload, ok := failedStep.Payload().(DestroyAssetPayload); ok {
			senderId = payload.CharacterId
		}
	}

	if !GetCache().TryTransition(c.ctx, s.TransactionId(), SagaLifecycleCompensating, SagaLifecycleFailed) {
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Info("saga already in terminal Failed state; note-send emission skipped.")
		SagaTimers().Cancel(s.TransactionId())
		GetCache().Remove(c.ctx, s.TransactionId())
		return nil
	}

	SagaTimers().Cancel(s.TransactionId())
	GetCache().Remove(c.ctx, s.TransactionId())

	reason := fmt.Sprintf("Note send failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if err := EmitSagaFailedByIds(c.l, c.ctx, s.TransactionId(), string(s.SagaType()), 0, senderId, sagaMsg.ErrorCodeUnknown, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event after note-send compensation.")
		return err
	}

	return nil
}
```

If Task 8 skipped `TestHandleCreateFailedEvent_FailsCreateNoteStep`, remove the skip now.

- [x] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./saga/... ./kafka/... && go vet ./...`
Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/
git commit -m "feat(atlas-saga-orchestrator): note_send completed results + item-refund compensation"
```

---

### Task 10: atlas-channel — saga message fields + note_send consumer branches

The channel's COMPLETED body is an empty struct today; it gains `SagaType`/`Results` (additive JSON decode). Completed note_send → announce MEMO_RESULT SEND_SUCCESS (mode 4; the excl unlock rides on the destroy's inventory-operation packet — `NewChangeBatch(false, …)` at `kafka/consumer/asset/consumer.go:421` writes leading byte 1 via `WriteBool(!silent)`, verified during planning). Failed note_send → announce SEND_ERROR `NO_NOTE_ITEM` (unlocks the client silently; the failure is server-logged).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/saga/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer.go`
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer_test.go` (new)

**Interfaces:**
- Consumes: `notecb.NoteSendSuccessBody()`, `notecb.NoteSendErrorBody(notecb.NoteSendErrorNoNoteItem)` (Task 3); orchestrator COMPLETED Results contract (Task 9).
- Produces: `saga.SagaTypeNoteSend = "note_send"`; `StatusEventCompletedBody{SagaType string, Results map[string]any}`; `extractResultCharacterId(map[string]any) uint32` (package-private helper, unit-tested).

Test-scope decision (recorded for plan adherence): the branch logic that is unit-testable without a live session registry + writer pipeline is the Results extraction; the announce paths follow the storage-failure precedent verbatim (session lookup → channel guard → `session.Announce`) and are exercised end-to-end by the live verification checklist in Task 14's PATCH doc. The channel codebase has no existing behavioral tests for saga-consumer announce paths to pattern-match against.

- [x] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer_test.go`:

```go
package saga

import (
	"testing"
)

func TestExtractResultCharacterId(t *testing.T) {
	cases := []struct {
		name    string
		results map[string]any
		want    uint32
	}{
		{"nil results", nil, 0},
		{"missing key", map[string]any{"other": float64(5)}, 0},
		{"json float64", map[string]any{"characterId": float64(100)}, 100},
		{"wrong type", map[string]any{"characterId": "100"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractResultCharacterId(tc.results); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/saga/ -v`
Expected: FAIL — `extractResultCharacterId` undefined.

- [x] **Step 3: Implement message fields**

`services/atlas-channel/atlas.com/channel/kafka/message/saga/kafka.go`:

Add to the saga-type const block:

```go
	SagaTypeNoteSend         = "note_send"
```

Change the completed body:

```go
type StatusEventCompletedBody struct {
	SagaType string         `json:"sagaType,omitempty"`
	Results  map[string]any `json:"results,omitempty"`
}
```

- [x] **Step 4: Implement consumer branches**

`services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer.go`:

1. Add imports:

```go
	notecb "github.com/Chronicle20/atlas/libs/atlas-packet/note/clientbound"
```

2. In `InitHandlers`, pass `wp` to the completed handler: `handleCompletedEvent(sc, wp)`.

3. Replace `handleCompletedEvent`:

```go
// handleCompletedEvent handles saga completion events
func handleCompletedEvent(sc server.Model, wp writer.Producer) message.Handler[saga.StatusEvent[saga.StatusEventCompletedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventCompletedBody]) {
		if e.Type != saga.StatusEventTypeCompleted {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		l.Debugf("Saga transaction [%s] completed successfully.", e.TransactionId.String())

		if e.Body.SagaType == saga.SagaTypeNoteSend {
			characterId := extractResultCharacterId(e.Body.Results)
			if characterId == 0 {
				l.WithField("transaction_id", e.TransactionId.String()).Warn("note_send completed without a characterId result; cannot announce SEND_SUCCESS.")
				return
			}

			s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(characterId)
			if err != nil {
				l.WithField("character_id", characterId).Debug("Sender not connected on this channel, skipping SEND_SUCCESS notification.")
				return
			}
			if s.ChannelId() != sc.ChannelId() {
				return
			}

			// The exclusive-request unlock rides on the inventory-operation
			// packet from the consumed item (leading byte 1); mode 4 itself
			// does not clear the lock.
			err = session.Announce(l)(ctx)(wp)(notecb.NoteOperationWriter)(notecb.NoteSendSuccessBody())(s)
			if err != nil {
				l.WithError(err).WithField("character_id", characterId).Error("Failed to send note SEND_SUCCESS packet to client.")
			}
			return
		}

		// Storage mesos update is handled by storage consumer
		// Character sees the result through character meso changed event
		// No additional action needed here
	}
}

// extractResultCharacterId reads Results["characterId"]; JSON numbers decode
// to float64.
func extractResultCharacterId(results map[string]any) uint32 {
	if results == nil {
		return 0
	}
	if v, ok := results["characterId"]; ok {
		if f, ok := v.(float64); ok {
			return uint32(f)
		}
	}
	return 0
}
```

4. In `handleFailedEvent`, after the session/channel guards and BEFORE the storage branch, add:

```go
		if e.Body.SagaType == saga.SagaTypeNoteSend {
			l.WithFields(logrus.Fields{
				"transaction_id": e.TransactionId.String(),
				"character_id":   e.Body.CharacterId,
				"failed_step":    e.Body.FailedStep,
			}).Warn("Note send saga failed; notifying client.")
			// NO_NOTE_ITEM (code 3) is outside the client's dialog range:
			// the MEMO_RESULT mode-5 arm clears the exclusive-request lock
			// before decoding the code, so this unlocks silently.
			err = session.Announce(l)(ctx)(wp)(notecb.NoteOperationWriter)(notecb.NoteSendErrorBody(notecb.NoteSendErrorNoNoteItem))(s)
			if err != nil {
				l.WithError(err).WithField("character_id", e.Body.CharacterId).Error("Failed to send note SEND_ERROR packet to client.")
			}
			return
		}
```

- [x] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/... -v && go build ./...`
Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/
git commit -m "feat(atlas-channel): note_send saga completion/failure client feedback"
```

---

### Task 11: atlas-channel — `FindFirstByClassification` + `buildNoteSendSaga` helper

Pure, unit-testable pieces both handler arms share.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/compartment/model.go`
- Modify: `services/atlas-channel/atlas.com/channel/saga/model.go` (re-exports)
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/note_send.go`
- Test: `services/atlas-channel/atlas.com/channel/compartment/model_test.go` (new), `services/atlas-channel/atlas.com/channel/socket/handler/note_send_test.go` (new)

**Interfaces:**
- Consumes: `sharedsaga.NoteSend`/`CreateNote`/`CreateNotePayload` (Task 4); `item.GetClassification` / `item.ClassificationNote` (atlas-constants).
- Produces: `compartment.Model.FindFirstByClassification(c item.Classification) (*asset.Model, bool)`; handler-package `buildNoteSendSaga(transactionId uuid.UUID, now time.Time, senderId uint32, templateId uint32, receiverId uint32, message string) saga.Saga` and `handleNoteSendRequest(l, ctx, wp) func(s session.Model, templateId uint32, toName string, message string)` (Tasks 12–13 call both).

- [x] **Step 1: Write the failing compartment test**

Compartment `Model`/`asset.Model` have private fields — use the existing package builders (`compartment.NewBuilder(id, characterId, inventoryType, capacity)` + `asset.NewModelBuilder(id, compartmentId, templateId)`; Builder-pattern rule, no test-only constructors).

Create `services/atlas-channel/atlas.com/channel/compartment/model_test.go`:

```go
package compartment

import (
	"testing"

	"atlas-channel/asset"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/google/uuid"
)

func TestFindFirstByClassification(t *testing.T) {
	cid := uuid.New()

	mustAsset := func(id uint32, templateId uint32, slot int16) asset.Model {
		a, err := asset.NewModelBuilder(id, cid, templateId).SetSlot(slot).SetQuantity(1).Build()
		if err != nil {
			t.Fatalf("asset build: %v", err)
		}
		return a
	}

	m, err := NewBuilder(cid, 100, inventory.TypeValueCash, 96).
		AddAsset(mustAsset(1, 2000000, 1)). // classification != 509
		AddAsset(mustAsset(2, 5090000, 2)). // Note (509) — first match
		AddAsset(mustAsset(3, 5090001, 3)). // Note (509)
		Build()
	if err != nil {
		t.Fatalf("compartment build: %v", err)
	}

	a, found := m.FindFirstByClassification(item.ClassificationNote)
	if !found {
		t.Fatal("expected a Note-classified asset")
	}
	if a.TemplateId() != 5090000 {
		t.Errorf("templateId: got %d, want 5090000 (first match wins)", a.TemplateId())
	}

	_, found = m.FindFirstByClassification(item.Classification(999))
	if found {
		t.Error("expected no match for classification 999")
	}
}
```

(If `asset.NewModelBuilder(...).Build()` enforces validations that reject these minimal assets, satisfy them with the builder's setters — do not bypass the builder.)

- [x] **Step 2: Write the failing saga-builder test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/note_send_test.go`:

```go
package handler

import (
	"testing"
	"time"

	"atlas-channel/saga"

	"github.com/google/uuid"
)

// TestBuildNoteSendSaga pins the FR-5 invariant: destroy-first, exactly two
// steps, correct payloads, flag 1.
func TestBuildNoteSendSaga(t *testing.T) {
	txn := uuid.New()
	now := time.Now()
	s := buildNoteSendSaga(txn, now, 100, 5090000, 200, "hello")

	if s.TransactionId != txn {
		t.Errorf("transactionId: got %s, want %s", s.TransactionId, txn)
	}
	if s.SagaType != saga.NoteSend {
		t.Errorf("sagaType: got %s, want %s", s.SagaType, saga.NoteSend)
	}
	if len(s.Steps) != 2 {
		t.Fatalf("steps: got %d, want 2", len(s.Steps))
	}

	if s.Steps[0].Action != saga.DestroyAsset {
		t.Errorf("step 1 action: got %s, want %s (destroy-first is mandatory)", s.Steps[0].Action, saga.DestroyAsset)
	}
	dp, ok := s.Steps[0].Payload.(saga.DestroyAssetPayload)
	if !ok {
		t.Fatalf("step 1 payload type: %T", s.Steps[0].Payload)
	}
	if dp.CharacterId != 100 || dp.TemplateId != 5090000 || dp.Quantity != 1 || dp.RemoveAll {
		t.Errorf("destroy payload mismatch: %+v", dp)
	}

	if s.Steps[1].Action != saga.CreateNote {
		t.Errorf("step 2 action: got %s, want %s", s.Steps[1].Action, saga.CreateNote)
	}
	np, ok := s.Steps[1].Payload.(saga.CreateNotePayload)
	if !ok {
		t.Fatalf("step 2 payload type: %T", s.Steps[1].Payload)
	}
	if np.SenderId != 100 || np.ReceiverId != 200 || np.Message != "hello" || np.Flag != 1 {
		t.Errorf("create-note payload mismatch: %+v", np)
	}
}
```

- [x] **Step 3: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./compartment/ ./socket/handler/ -run "TestFindFirstByClassification|TestBuildNoteSendSaga" -v`
Expected: FAIL — undefined symbols.

- [x] **Step 4: Implement**

`services/atlas-channel/atlas.com/channel/compartment/model.go` — append (import `"github.com/Chronicle20/atlas/libs/atlas-constants/item"`):

```go
func (m Model) FindFirstByClassification(c item.Classification) (*asset.Model, bool) {
	for _, a := range m.Assets() {
		if item.GetClassification(item.Id(a.TemplateId())) == c {
			return &a, true
		}
	}
	return nil, false
}
```

`services/atlas-channel/atlas.com/channel/saga/model.go` — add re-exports:

To the payload types block:

```go
	CreateNotePayload            = sharedsaga.CreateNotePayload
```

To the const block (saga types):

```go
	NoteSend             = sharedsaga.NoteSend
```

And (action constants):

```go
	CreateNote           = sharedsaga.CreateNote
```

Create `services/atlas-channel/atlas.com/channel/socket/handler/note_send.go`:

```go
package handler

import (
	character2 "atlas-channel/character"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	notecb "github.com/Chronicle20/atlas/libs/atlas-packet/note/clientbound"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// buildNoteSendSaga assembles the note_send saga: destroy-first (FR-5) —
// the Note item is confirmed consumed before the note exists; if note
// creation then fails, the orchestrator re-awards the item.
func buildNoteSendSaga(transactionId uuid.UUID, now time.Time, senderId uint32, templateId uint32, receiverId uint32, message string) saga.Saga {
	return saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.NoteSend,
		InitiatedBy:   "NOTE_SEND",
		Steps: []saga.Step{
			{
				StepId: "consume_note_item",
				Status: saga.Pending,
				Action: saga.DestroyAsset,
				Payload: saga.DestroyAssetPayload{
					CharacterId: senderId,
					TemplateId:  templateId,
					Quantity:    1,
					RemoveAll:   false,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				StepId: "create_note",
				Status: saga.Pending,
				Action: saga.CreateNote,
				Payload: saga.CreateNotePayload{
					SenderId:   senderId,
					ReceiverId: receiverId,
					Message:    message,
					Flag:       1,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
}

// handleNoteSendRequest runs the shared pre-flight checks for both note send
// paths (USE_CASH_ITEM note arm and NOTE_ACTION SEND) and, if they pass,
// creates the note_send saga. Pre-flight rejections announce MEMO_RESULT
// SEND_ERROR inline and consume nothing (FR-7).
func handleNoteSendRequest(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, templateId uint32, toName string, message string) {
	return func(s session.Model, templateId uint32, toName string, message string) {
		tc, err := character2.NewProcessor(l, ctx).GetByName(toName)
		if err != nil {
			l.WithError(err).Warnf("Character [%d] attempted to send a note to unknown receiver [%s].", s.CharacterId(), toName)
			_ = session.Announce(l)(ctx)(wp)(notecb.NoteOperationWriter)(notecb.NoteSendErrorBody(notecb.NoteSendErrorReceiverUnknown))(s)
			return
		}

		// Receiver-online check (design §4.1 step 4). Scope: the session
		// registry only tracks THIS channel's sessions; a receiver online on
		// another channel is not detected and the note is stored normally —
		// documented limitation, no cross-channel lookup exists in
		// atlas-channel today.
		if _, oerr := session.NewProcessor(l, ctx).GetByCharacterId(s.Field().Channel())(tc.Id()); oerr == nil {
			l.Debugf("Character [%d] attempted to send a note to online receiver [%d].", s.CharacterId(), tc.Id())
			_ = session.Announce(l)(ctx)(wp)(notecb.NoteOperationWriter)(notecb.NoteSendErrorBody(notecb.NoteSendErrorReceiverOnline))(s)
			return
		}

		ns := buildNoteSendSaga(uuid.New(), time.Now(), s.CharacterId(), templateId, tc.Id(), message)
		if err = saga.NewProcessor(l, ctx).Create(ns); err != nil {
			l.WithError(err).Errorf("Character [%d] unable to initiate note send saga.", s.CharacterId())
		}
	}
}
```

- [x] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./compartment/ ./socket/handler/ -run "TestFindFirstByClassification|TestBuildNoteSendSaga" -v && go build ./...`
Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/compartment/ services/atlas-channel/atlas.com/channel/saga/model.go services/atlas-channel/atlas.com/channel/socket/handler/note_send.go services/atlas-channel/atlas.com/channel/socket/handler/note_send_test.go
git commit -m "feat(atlas-channel): note_send saga builder + shared send pre-flight helper"
```

---

### Task 12: atlas-channel — USE_CASH_ITEM note arm

Wire the note arm into `CharacterCashItemUseHandleFunc`. The slot/template validation at lines 37–41 already proves the sender owns the claimed item, and `GetCashSlotItemType` returns type 21 only for classification-509 item ids — so reaching the arm already implies a Note item; no additional classification check is added (a second check would be tautological — decision recorded here for plan-adherence review). Also switch the handler's `updateTimeFirst` to the Task-1 helper (behavior-identical for the currently-routed gms_83/84 tenants; required before Task 14 routes 87/95/jms).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go` (new)

**Interfaces:**
- Consumes: `cashsb.UpdateTimeFirst` (Task 1), `cashsb.NewItemUseNote` (Task 2), `handleNoteSendRequest` (Task 11).
- Produces: `CashSlotItemTypeNote = CashSlotItemType(21)` named constant.

- [x] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go` (decode-pinning style of `mount_food_test.go`):

```go
package handler

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// TestCashSlotItemTypeNote pins that Note items (5090000-family,
// classification 509) map to the named note slot type on every version.
func TestCashSlotItemTypeNote(t *testing.T) {
	for _, v := range []struct {
		region string
		major  uint16
	}{{"GMS", 83}, {"GMS", 84}, {"GMS", 87}, {"GMS", 95}, {"JMS", 185}} {
		tn, err := tenant.Create(uuid.New(), v.region, v.major, 1)
		if err != nil {
			t.Fatal(err)
		}
		if got := GetCashSlotItemType(tn)(item.Id(5090000)); got != CashSlotItemTypeNote {
			t.Errorf("%s v%d: got %d, want %d", v.region, v.major, got, CashSlotItemTypeNote)
		}
	}
}

// TestCharacterCashItemUseHandleFuncSymbol verifies the handler constructor
// returns a non-nil closure. The constructor calls tenant.MustFromContext,
// so a tenant context is required; a nil writer.Producer is acceptable for
// the symbol check only.
func TestCharacterCashItemUseHandleFuncSymbol(t *testing.T) {
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), tn)
	if got := CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil); got == nil {
		t.Fatal("nil closure")
	}
}
```

(Add `"context"` and `"github.com/sirupsen/logrus"` to the test file's imports.)

- [x] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run "TestCashSlotItemTypeNote" -v`
Expected: FAIL — `CashSlotItemTypeNote` undefined.

- [x] **Step 3: Implement**

`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`:

1. Named constant — the const block becomes:

```go
const (
	CashSlotItemTypeFieldEffect   = CashSlotItemType(16)
	CashSlotItemTypeNote          = CashSlotItemType(21)
	CashSlotItemTypePetConsumable = CashSlotItemType(30)
	CashSlotItemTypeChalkboard    = CashSlotItemType(32)
)
```

and in `GetCashSlotItemType`, the 509 branch (line ~251) returns the constant:

```go
		if category == item.ClassificationNote {
			return CashSlotItemTypeNote
		}
```

2. Handler signature: the writer producer is now used — change line 25 from `_ writer.Producer` to `wp writer.Producer`.

3. `updateTimeFirst` (line 32) becomes:

```go
		updateTimeFirst := cashsb.UpdateTimeFirst(t)
```

4. Add the note arm after the FieldEffect arm (before the fall-through warn at line ~108):

```go
		if it == CashSlotItemTypeNote {
			sp := cashsb.NewItemUseNote(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			// Slot/template validation above proves ownership of the claimed
			// Note item; pre-flight receiver checks + the destroy-first saga
			// live in handleNoteSendRequest (note_send.go).
			handleNoteSendRequest(l, ctx, wp)(s, uint32(itemId), sp.ToName(), sp.Message())
			return
		}
```

5. Delete the stale comment line `// TODO for v83 there is a trailing updateTime.` (line 108) — it is resolved by the arm codec.

- [x] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -v && go build ./...`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go
git commit -m "feat(atlas-channel): USE_CASH_ITEM note arm — consume-gated note send"
```

---

### Task 13: atlas-channel — NOTE_ACTION SEND arm ownership gate

Replace the free `np.SendNote(...)` (design §1.2: the only legitimate writer of this arm is the unimplemented cash-shop gift flow, so traffic here is gift-flow-future or a tampered client — gate it, don't ban it). Remove the now-unused `SendNote`/`CreateCommandProvider` from the channel's note package (the CREATE command is orchestrator-owned as of Task 8).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/note_operation.go`
- Modify: `services/atlas-channel/atlas.com/channel/note/processor.go`, `services/atlas-channel/atlas.com/channel/note/producer.go`
- Test: existing `./socket/handler/` + `./note/` package tests

**Interfaces:**
- Consumes: `compartment.GetByType(characterId, inventory.TypeValueCash)`, `FindFirstByClassification` (Task 11), `handleNoteSendRequest` (Task 11), `notecb.NoteSendErrorNoNoteItem` (Task 3).

- [x] **Step 1: Replace the SEND arm**

In `services/atlas-channel/atlas.com/channel/socket/handler/note_operation.go`, replace the SEND block (lines 32–48) with:

```go
	if isNoteOperation(l)(readerOptions, op, NoteOperationSend) {
		sp := &notesb.OperationSend{}
		sp.Decode(l, ctx)(r, readerOptions)

		// This arm's only legitimate client-side writer is the cash-shop
		// gift flow (CCashShop::OnCashItemResLoadGiftDone), which is not
		// implemented server-side; the player send path is USE_CASH_ITEM.
		// Gate on Note-item ownership so a tampered client cannot mint free
		// notes (FR-4). When gifting lands, gift sends must NOT route
		// through this consume-gated path (the note is paid for by the gift
		// purchase) — see design §2.2.
		cp, err := compartment.NewProcessor(l, ctx).GetByType(s.CharacterId(), inventory.TypeValueCash)
		if err != nil {
			l.WithError(err).Warnf("Character [%d] NOTE_ACTION SEND rejected: unable to load cash compartment.", s.CharacterId())
			_ = session.Announce(l)(ctx)(wp)(notecb.NoteOperationWriter)(notecb.NoteSendErrorBody(notecb.NoteSendErrorNoNoteItem))(s)
			return
		}
		a, found := cp.FindFirstByClassification(item.ClassificationNote)
		if !found {
			l.Warnf("Character [%d] attempted NOTE_ACTION SEND without owning a Note item (classification 509). Rejecting.", s.CharacterId())
			_ = session.Announce(l)(ctx)(wp)(notecb.NoteOperationWriter)(notecb.NoteSendErrorBody(notecb.NoteSendErrorNoNoteItem))(s)
			return
		}

		handleNoteSendRequest(l, ctx, wp)(s, a.TemplateId(), sp.ToName(), sp.Message())
		return
	}
```

Imports to add: `"atlas-channel/compartment"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/item"`. The `"atlas-channel/note"` import stays (DISCARD/REQUEST arms still use `np`); if the compiler reports `np` unused because the SEND arm no longer uses it, move `np := note.NewProcessor(l, ctx)` below the SEND block.

- [x] **Step 2: Remove the dead send path from the note package**

Verify nothing else calls it: `grep -rn "SendNote\|CreateCommandProvider" services/atlas-channel/ --include="*.go"`
Expected: only the definitions in `note/processor.go` / `note/producer.go` (the handler call site is gone after Step 1).

Then delete `SendNote` from the `Processor` interface and `ProcessorImpl` in `services/atlas-channel/atlas.com/channel/note/processor.go`, and `CreateCommandProvider` from `services/atlas-channel/atlas.com/channel/note/producer.go`. `DiscardNotes`/`DiscardCommandProvider`/`GetByCharacter`/`GetById` stay.

If the grep shows another caller, STOP and re-evaluate (do not delete).

- [x] **Step 3: Run tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./...`
Expected: PASS.

- [x] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/note_operation.go services/atlas-channel/atlas.com/channel/note/
git commit -m "feat(atlas-channel): gate NOTE_ACTION SEND on Note-item ownership + consumption"
```

---

### Task 14: Seed templates (9 versions) + live-tenant PATCH doc

**Premise correction (main-sync):** the original plan added the `CharacterCashItemUseHandle` route to gms_87/95/jms. That route now **already exists in all nine templates** (main wired it during the version bring-ups — verified: gms_48 0x3E, gms_61 0x49, gms_72 0x4E, gms_79 0x4D, gms_83/84 0x4F, gms_87 0x52, gms_95 0x55, jms_185 0x47). Do NOT add duplicate handler entries. The remaining template work is the **writer config** (the clientbound MEMO_RESULT `operations`/`errors` tables) plus the serverbound `NoteOperationHandle` `operations` table, correct per version — including the **shifted v48/v61 MEMO_RESULT mode table** (design §1.3/§5.1).

Retained planning-phase findings still to verify against the current templates: the gms_95 `NoteOperationHandle` entry historically lacked a `validator` (silently dropped by `BuildHandlerMap`); gms_87/95 `NoteOperation` writers historically lacked the `errors` table. Re-inspect each template's current state before editing — main may have partially fixed these during bring-up. gms_92 and gms_12 have no matrix column and are out of scope.

**Files:**
- Modify (writer/handler `options` only; routing already present): `template_gms_48_1.json`, `template_gms_61_1.json`, `template_gms_72_1.json`, `template_gms_79_1.json`, `template_gms_83_1.json`, `template_gms_84_1.json`, `template_gms_87_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json` (all under `services/atlas-configurations/seed-data/templates/`)
- Create: `docs/tasks/task-137-note-item-consumption/live-tenant-patch.md`

For every step below: locate the entries by `jq`/grep (`handler == "NoteOperationHandle"`, `writer == "NoteOperation"`), not by hard-coded line numbers — the legacy templates were added after this plan and their layout is unknown here. Every `NoteOperationHandle` entry must carry `"validator": "LoggedInValidator"` (a validator-less entry is silently dropped).

- [x] **Step 1: Clientbound MEMO_RESULT writer — `operations` table (per version)**

The `NoteOperation` writer's `options.operations` must match the client's mode table (design §1.3). Set/verify:

- **v48, v61 (shifted):** `{ "SHOW": 2, "SEND_SUCCESS": 3, "SEND_ERROR": 4 }` (no `REFRESH`).
- **v72, v79, v83, v84, v87, v95, jms_185 (standard):** `{ "SHOW": 3, "SEND_SUCCESS": 4, "SEND_ERROR": 5, "REFRESH": 7 }`.

> A hard-coded/standard `SEND_ERROR: 5` on v48/v61 silently wedges those clients (mode 5 is unhandled there → the excl lock never clears). This is the single most important legacy config value in the task.

- [x] **Step 2: Clientbound MEMO_RESULT writer — `errors` table + `NO_NOTE_ITEM` (all nine)**

On every version's `NoteOperation` writer, ensure `options.errors` contains all four keys (sub-codes are uniform across versions, design §1.3/§5.3):

```json
          "errors": {
            "RECEIVER_ONLINE": 0,
            "RECEIVER_UNKNOWN": 1,
            "RECEIVER_INBOX_FULL": 2,
            "NO_NOTE_ITEM": 3
          }
```

(gms_83/84/jms_185 historically have the first three; add `NO_NOTE_ITEM`. gms_87/95 may lack the whole table; add it. Legacy v48/61/72/79 — inspect and add whatever is missing.)

- [x] **Step 3: Serverbound `NoteOperationHandle` — `operations` table (per version)**

The inbound dispatch table maps client mode bytes to Atlas operations. DISCARD (mode 1) is the load-bearing one Atlas actually decodes; SEND (mode 0) is the gated arm. Set `options.operations`:

- **v72, v79, v83, v84, v87, v95, jms_185:** `{ "SEND": 0, "DISCARD": 1, "REQUEST": 2 }`.
- **v48, v61:** `{ "SEND": 0, "DISCARD": 1 }` — no `REQUEST` writer exists client-side on these versions (design §1.2); include only the modes the client emits. (Adding an unused `REQUEST` key is harmless if it simplifies the fixture, but the client never sends it here.)

Mode immediates were confirmed at the writer level per version (SetRet=1, request=2 where present, gift=0) in the legacy pass and design §1.2; the discard fixtures (Task 17) pin them.

- [x] **Step 4: Validate JSON**

Run: `for f in services/atlas-configurations/seed-data/templates/template_gms_48_1.json services/atlas-configurations/seed-data/templates/template_gms_61_1.json services/atlas-configurations/seed-data/templates/template_gms_72_1.json services/atlas-configurations/seed-data/templates/template_gms_79_1.json services/atlas-configurations/seed-data/templates/template_gms_83_1.json services/atlas-configurations/seed-data/templates/template_gms_84_1.json services/atlas-configurations/seed-data/templates/template_gms_87_1.json services/atlas-configurations/seed-data/templates/template_gms_95_1.json services/atlas-configurations/seed-data/templates/template_jms_185_1.json; do jq empty "$f" && echo "OK $f"; done`
Expected: `OK` ×9. Additionally assert the shifted table where it matters:
`jq '.. | objects | select(.writer=="NoteOperation") | .options.operations.SEND_ERROR' template_gms_48_1.json` → `4`; same for gms_61; → `5` for the other seven.

- [x] **Step 5: Write the live-tenant PATCH doc**

Seed templates apply only at tenant creation (known bug pattern) — existing tenants need a config PATCH plus channel restart. Create `docs/tasks/task-137-note-item-consumption/live-tenant-patch.md` documenting, for EACH existing tenant of the **nine** versions:

1. What to add/correct (exactly the deltas from Steps 1–3, per version — reproduce the JSON fragments; call out the v48/v61 `SEND_ERROR: 4` explicitly).
2. How: PATCH the tenant's socket configuration resource via the atlas-tenants API (`GET /tenants/{tenantId}/configurations/socket`, edit, PATCH back), or via the atlas-ui configuration editor.
3. Restart note: `kubectl rollout restart deployment atlas-channel` (or per-env equivalent) — handler/writer projections do not hot-reload.
4. A verification checklist: send a note from a character holding item 5090000 (item consumed, receiver sees note on next login); a NOTE_ACTION SEND without the item via packet injection is expected to warn-log + silent-unlock; on a v48/v61 tenant specifically, confirm the SEND_ERROR path actually unlocks the client (regression guard for the shifted mode).

Use repo-relative paths and placeholders only (no literal home paths).

- [x] **Step 6: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/ docs/tasks/task-137-note-item-consumption/live-tenant-patch.md
git commit -m "feat(config): note operations/errors across 9 version templates (v48/v61 shifted mode table)"
```

---

### Task 15: Matrix promotion — `MEMO_RESULT` × gms_v84

Follow `docs/packets/audits/VERIFYING_A_PACKET.md` exactly (or dispatch the `packet-verifier` agent with the parameters below — one agent per cell, serialized). Design §6.3 pre-resolves the blockers: the old audit's "function not found" is stale — `CWvsContext::OnMemoResult` is at **0xa70785** in the v84 IDB; arms verified identical to v83. Opcode 0x029 must be CONFIRMED from the v84 clientbound dispatch table during verification (the known v84 shift starts above ~0x3D; 0x029 is below it — confirm, don't assume).

**Files:**
- Modify: `libs/atlas-packet/note/clientbound/display_test.go` (add the `packet-audit:verify … version=gms_v84 ida=0xa70785` marker + a v84 fixture leg if the playbook requires a distinct byte fixture)
- Modify/Create: evidence under `docs/packets/audits/gms_v84/` (the stale `NoteDisplay.md`/`.json` get regenerated — surgical splice per the export non-idempotency rule)
- Regenerate: `docs/packets/audits/STATUS.md` + `status.json`

- [x] **Step 1:** Confirm the IDA instance is the v84 IDB (`select_instance` to the v84 port; verify the binary name/version before reading). Decompile `CWvsContext::OnMemoResult` @0xa70785 and record the read order per mode (expect: identical to v83 — modes 3/4/5/7 per design §1.3).
- [x] **Step 2:** Confirm opcode 0x029 in the v84 clientbound dispatch table.
- [x] **Step 3:** Write/extend the byte-fixture test with the `packet-audit:verify packet=note/clientbound/NoteDisplay version=gms_v84 ida=0xa70785` marker; run `cd libs/atlas-packet && go test ./note/clientbound/ -v` → PASS.
- [x] **Step 4:** Pin the evidence record and regenerate the matrix per the playbook; run the playbook's matrix `--check` → exit 0; confirm `STATUS.md` shows ✅ for MEMO_RESULT × v84 and NO other cell changed.
- [x] **Step 5:** Commit the three artifacts together:

```bash
git add libs/atlas-packet/note/clientbound/ docs/packets/audits/
git commit -m "verify(packets): MEMO_RESULT x gms_v84 — OnMemoResult 0xa70785, opcode confirmed"
```

---

### Task 16: Matrix promotion — `MEMO_RESULT` × jms_v185

Same flow as Task 15 with the jms particulars (design §6.3): `OnMemoResult` @ **0xb0c6d0**, opcode 0x026 to confirm; IDA instance is `MapleStory_dump_SCY.exe` on **port 13344** (the retail dump is SMC — use the `_SCY`/`*_U_DEVM` build); pass `--audit-dir docs/packets/audits/jms_v185` EXPLICITLY to any packet-audit subcommand (the default dir name is wrong for jms and silently reports 0/0/0/0).

**Files:**
- Modify: `libs/atlas-packet/note/clientbound/display_test.go` (marker `version=jms_v185 ida=0xb0c6d0` + fixture leg)
- Create/Modify: evidence under `docs/packets/audits/jms_v185/`
- Regenerate: `docs/packets/audits/STATUS.md` + `status.json`

- [x] **Step 1:** `select_instance(13344)`; verify the loaded binary is the `_SCY` jms dump before reading. Decompile 0xb0c6d0; record per-mode read order.
- [x] **Step 2:** Confirm opcode 0x026 in the jms dispatch table.
- [x] **Step 3:** Byte-fixture test with the verify marker; `go test ./note/clientbound/ -v` → PASS.
- [x] **Step 4:** Pin evidence (with `--audit-dir docs/packets/audits/jms_v185`), regenerate matrix, `--check` exit 0, no cell regressions.
- [x] **Step 5:** Commit:

```bash
git add libs/atlas-packet/note/clientbound/ docs/packets/audits/
git commit -m "verify(packets): MEMO_RESULT x jms_v185 — OnMemoResult 0xb0c6d0"
```

---

### Task 17: Matrix promotion — `note/serverbound/NoteOperationDiscard` × jms_v185 + four legacy versions

Five serverbound cells: jms_v185 (❌) plus the four legacy versions v48/v61/v72/v79 (all ❌ or 🟡 on main). Each is a marker + evidence + REPORT cell (report via root `-ida-source`), per VERIFYING_A_PACKET.md §9–10 — one `packet-verifier`-style pass per cell, serialized (shared `operation_discard.go` + matrix). The per-version `CMemoListDlg::SetRet` addresses and body shapes are documented in design §1.5 and the `legacy-verify/` files; use them to derive each read order, but confirm against the IDB (do not paste from the doc without re-verifying the read order in the playbook).

The discard body has a uniform header (`mode=1`, `totalCount u8`, `specialCount u8`, `emptySlots u8`) and a per-entry loop where the "special" (gift/reward) flag and its extra int32 field differ per version:

| cell | IDB | SetRet addr | note-op | special flag | extra field | quirk |
|---|---|---|---|---|---|---|
| gms_v48 | v48 | 0x534dc4 | 0x65 | **2** | reward i32 | ALSO emits a follow-up **0x66** packet for flag==1 notes — decide during verification whether that is part of this cell or a separate op |
| gms_v61 | v61 | 0x5ad50c | 0x77 | **2** | itemId i32 | — |
| gms_v72 | v72 | 0x5fb443 | 0x81 | **3** | mesos i32 | main shows 🟡; this promotes it to ✅ |
| gms_v79 | v79 | 0x619fb7 | 0x80 | **3** | value i32 | — |
| jms_v185 | jms (port 13344, `_SCY` build) | 0x6c2d43 | 0x86 | — | — | **0x33d bytes vs ~0x26b GMS — derive from scratch; do not assume the GMS shape** |

If any version's read order diverges from the existing GMS codec, add a version-gated delta in `libs/atlas-packet/note/serverbound/operation_discard.go` with byte fixtures for each distinct shape. The special-flag value (2 vs 3) and v48's 0x66 tail mean at least two legacy shapes exist — do NOT assume one fixture covers all four.

**Files:**
- Modify: `libs/atlas-packet/note/serverbound/operation_discard.go` (version-gated deltas as the derivations demand), `operation_discard_test.go` (one `packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=<v> ida=<addr>` marker + fixture per cell)
- Create: `docs/packets/audits/{jms_v185,gms_v48,gms_v61,gms_v72,gms_v79}/NoteOperationDiscard.md`/`.json` evidence + REPORT
- Regenerate: `docs/packets/audits/STATUS.md` + `status.json`

For each of the five cells (jms first, then v48/v61/v72/v79):

- [x] **Step 1:** Select the correct IDB instance (jms → port 13344, verify the `_SCY` binary; each legacy → its own instance — always list + match the binary NAME before reading). Decompile the cited `SetRet`; derive the full write order from that binary alone (mode byte, the three header counts, per-entry normal/special fields).
- [x] **Step 2:** Compare to the codec; implement a version-gated delta if needed (byte fixtures for each distinct shape). For jms, budget for the larger function; for v48, resolve the 0x66-tail question.
- [x] **Step 3:** Fixture + `packet-audit:verify` marker; `cd libs/atlas-packet && go test ./note/serverbound/ -v` → PASS.
- [x] **Step 4:** Evidence + REPORT (root `-ida-source`; jms needs `--audit-dir docs/packets/audits/jms_v185`), surgical splice (never overwrite the export), regenerate matrix, `--check` exit 0. Expected: the cell ✅, no regressions.

- [x] **Step 5:** Commit (group the five cells, or commit per cell — either way the codec + fixture + evidence for a cell land together):

```bash
git add libs/atlas-packet/note/serverbound/ docs/packets/audits/
git commit -m "verify(packets): NoteOperationDiscard x jms_v185 + v48/v61/v72/v79 — SetRet read orders"
```

---

### Task 18: Full verification gate

Per CLAUDE.md Build & Verification — every check, every changed module. Changed Go modules: `libs/atlas-packet`, `libs/atlas-saga`, `services/atlas-notes/atlas.com/notes`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `services/atlas-channel/atlas.com/channel`. Note the guard set grew on main since this plan was written — `tools/goroutine-guard.sh` and `tools/lint.sh --check` are now mandatory alongside `redis-key-guard.sh` (Steps 3a/3b below).

- [x] **Step 1: Tests + vet per changed module**

```bash
for m in libs/atlas-packet libs/atlas-saga services/atlas-notes/atlas.com/notes services/atlas-saga-orchestrator/atlas.com/saga-orchestrator services/atlas-channel/atlas.com/channel; do
  (cd "$m" && go test -race ./... && go vet ./...) || echo "FAILED: $m"
done
```

Expected: no `FAILED:` lines. `go build ./...` in each service module as well.

- [x] **Step 2: Docker bake for every touched service** (mandatory — catches missing `COPY libs/...` lines that go.work hides)

```bash
docker buildx bake atlas-channel atlas-saga-orchestrator atlas-notes
```

Expected: all three build clean. (Also bake `atlas-configurations` if `.github/config/services.json` lists it as a Go service and its files changed — template JSON edits alone don't require it, but verify with `jq -r '.[]|.name' .github/config/services.json | grep configurations`.)

- [x] **Step 3: Repo-root guards** (from repo root, no global GOWORK=off)

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```

Expected: all clean. (`tools/lint.sh` with no flags fixes formatting in place — run it before committing if `--check` complains.)

- [x] **Step 4: packet-audit checks**

Run the matrix / fname-doc / operations `--check` commands used in Tasks 15–17 once more against the final tree (with the jms `--audit-dir` flag where applicable).
Expected: all exit 0; `docs/packets/audits/STATUS.md` shows all target cells ✅ (MEMO_RESULT × v84/jms185; NoteOperationDiscard × jms185/v48/v61/v72/v79) with no regressions.

- [x] **Step 5: Acceptance sweep against the PRD**

Walk PRD §10's checklist and confirm each box has landed evidence (per-version send path documented in design §1.1 for all nine versions; consumption on both arms; pre-flight rejections consume nothing; MEMO_RESULT × v84/jms185 ✅; NoteOperationDiscard × jms185 + four legacy ✅; nine templates + PATCH doc, incl. the v48/v61 shifted mode table; gift flow untouched). Fix anything missing BEFORE declaring done.

- [x] **Step 6: Commit any straggler fixes, then request code review**

Code review (superpowers:requesting-code-review) is mandatory before the PR — plan-adherence + backend-guidelines reviewers.

---

## Self-review notes (spec coverage)

- FR-1/FR-2 (send-path verification): done at design time for all nine versions (design §1.1 + `legacy-verify/{v48,v61,v72,v79}.md`); Tasks 15–17 add the remaining per-version byte evidence.
- FR-3 (UseCashItem note arm): Tasks 1, 2, 12, 14. Codec is uniform across all nine versions; only opcode (per-template) and updateTime position vary, both handled by the existing `(GMS>=87)||JMS` gate.
- FR-4 (SEND-arm gate): Task 13. Applies uniformly even on v48/v79 where the client has no NOTE_ACTION mode-0 writer at all (design §1.2).
- FR-5 (atomicity, destroy-first): Tasks 4, 9, 11 (`TestBuildNoteSendSaga` pins step order; compensation re-awards). Version-agnostic.
- FR-6 (client error on no item): Task 3 (`NO_NOTE_ITEM`, resolved from config per §5.3 — no client arm exists in any version; silent-unlock semantics IDA-verified across all nine) + Tasks 10/12/13 announcements. The SEND_ERROR *mode* byte is version-resolved (4 on v48/v61, 5 on v72+).
- FR-7 (no consumption on rejection): pre-flight checks precede saga creation (Task 11 helper); receiver-unknown path retained.
- FR-8/9/10/11 (matrix): Tasks 15/16 (MEMO_RESULT × v84/jms185) + Task 17 (NoteOperationDiscard × jms185 + v48/v61/v72/v79) + Task 18 Step 4. (Legacy MEMO_RESULT cells are already ✅ — not re-verified.)
- FR-12 (config wiring): Task 14 (nine templates + live PATCH doc; handler routing already present on main; v48/v61 shifted MEMO_RESULT mode table is the key legacy delta; gms_92/gms_12 out of scope — no matrix column).
- FR-13 (DOM-25): mode bytes and error codes resolve via `operations`/`errors` tables throughout; the per-version mode-table split (v48/v61 vs v72+) is absorbed entirely by config — no version literals in handlers/writers.
- Excl-lock risk (design §10): resolved during planning — destroy-driven `NewChangeBatch(false, …)` writes leading byte 1 (`WriteBool(!silent)`, `libs/atlas-packet/inventory/clientbound/change.go`), Task 10 documents it at the announce site.
- Pre-existing SEND_ERROR mode bug found and fixed (Task 3) — errors previously went out under the SEND_SUCCESS mode.
- **Legacy divergences folded in (main-sync):** all four new versions (v48/v61/v72/v79) IDA-verified to have the cash-item note-send arm (trailing updateTime); the only wire-invisible surprises — the v48/v61 shifted MEMO_RESULT mode table and the per-version discard bodies — are handled by config (Task 14) and the discard verification (Task 17) respectively, with zero change to the version-agnostic saga/service core (Tasks 4–13).
