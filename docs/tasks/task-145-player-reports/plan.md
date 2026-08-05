# Player Reports (Sue / Claim System) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the client's two player-reporting mechanisms (sue `/`-command and the CUIClaim harassment window) end-to-end: packet codecs, atlas-channel handlers/writers, a persistent `report` domain in atlas-ban, server-side chat capture in atlas-messages, and an atlas-ui admin review page.

**Architecture:** atlas-channel decodes SUE_CHARACTER / CLAIM_REQUEST and emits a `REPORT` Kafka command; atlas-ban resolves the accused via atlas-character REST, fetches a chat transcript from atlas-messages REST, persists the report, and emits a status event; atlas-channel consumes the status event and sends the reporter the IDA-verified result packet. Chat capture rides the existing atlas-messages chat path into a bounded per-character Redis sorted set.

**Tech Stack:** Go (atlas-channel, atlas-ban, atlas-messages, libs/atlas-packet, libs/atlas-redis), Kafka, GORM/Postgres, Redis, React 19 + TanStack Query 5 (atlas-ui), IDA Pro via ida-pro-mcp, packet-audit tooling.

**Authoritative inputs:** `docs/tasks/task-145-player-reports/design.md` (design), `prd.md` (requirements), `packet-findings.md` (IDA-verified packet semantics).

## Global Constraints

- **DOM-25 / config-drive-all-modes:** every client-interpreted result/mode byte resolves from the tenant writer `options.operations` table via `atlas_packet.WithResolvedCode`/`ResolveCode` — never a literal in domain logic, even for version-stable values.
- **Version scope (PRD §2.1) — the two mechanisms have different spans:** sue is supported on **v61, v72, v79, v83, v84, v87, v95**; claim on **v72, v79, v83, v84, v87, v95**. v48 supports neither (both send-sites verified absent), v61 has no claim send-site. Mode values are version-stable across the whole span; **opcodes are not**.
- **Sue operations table (version-stable across the span):** `SUCCESS: 0x00, UNABLE_TO_LOCATE: 0x01, DAILY_LIMIT: 0x02, REPORTED_NOTICE: 0x03, GENERIC_FAILURE: 0x04`.
- **Claim operations table (version-stable across the span):** `SUCCESS: 0x02, REPORTED_NOTICE: 0x03, TRY_AGAIN: 0x41, RECHECK_NAME: 0x42, NOT_ENOUGH_MESOS: 0x43, CANNOT_CONNECT: 0x44, EXCEEDED: 0x45, TIME_WINDOW: 0x47, FALSE_REPORT_CITED: 0x48`.
- **Opcodes** (from `docs/packets/audits/STATUS.md`, except the two v72/v79 claim-request values which this task adds to the registry — Task 23b):

  | packet | v61 | v72 | v79 | v83 | v84 | v87 | v95 |
  |---|---|---|---|---|---|---|---|
  | SUE_CHARACTER_RESULT | 0x34 | 0x34 | 0x34 | 0x37 | 0x37 | 0x37 | 0x37 |
  | CLAIM_RESULT | — | 0x2A | 0x2A | 0x2D | 0x2D | 0x2D | 0x2C |
  | CLAIM_AVAILABLE_TIME | — | 0x2B | 0x2B | 0x2E | 0x2E | 0x2E | 0x2D |
  | CLAIM_STATUS_CHANGED | — | 0x2C | 0x2C | 0x2F | 0x2F | 0x2F | 0x2E |
  | CLAIM_REQUEST (sb) | — | **0x69** | **0x68** | 0x6A | 0x6A | 0x6D | 0x76 |

  **Never port a v83 opcode down to v61/v72/v79** — the clientbound trio sits a full region lower there.
- **v1 result policy:** sue → SUCCESS / UNABLE_TO_LOCATE / GENERIC_FAILURE; claim → SUCCESS (hasRemaining=1, remaining=100, named constant) / RECHECK_NAME / TRY_AGAIN. All other table keys exist but are unemitted.
- **Size caps (server-side, truncate-and-log, never reject):** description 2000 chars, chat log 16384 bytes.
- **Chat capture defaults:** `CHAT_CAPTURE_RETENTION_SECONDS=900`, `CHAT_CAPTURE_MAX_LINES=200`. Captured types: GENERAL, BUDDY, PARTY, GUILD, ALLIANCE, WHISPER, MESSENGER. Not captured: PET, PINK_TEXT, slash commands.
- **Scope exclusions:** jms, gms_12, gms_48 and gms_92 get NO seed-template entries; gms_61 gets sue entries only (feature stays config-disabled otherwise). No quota/mesos enforcement, no accused-notification codes (deferred to docs/TODO.md, never code TODOs).
- **Template entries are inserted at their sorted `opCode` position** in both the `handlers` and `writers` arrays — `tools/template-opcode-order-guard.sh` (CI-gated) rejects appending them beside semantically-related entries.
- **Multi-tenancy:** GORM tenant scoping is automatic via `database.RegisterTenantCallbacks` + `db.WithContext(ctx)` — providers do NOT add explicit `tenant_id` WHERE clauses (mirror the `ban` domain). Kafka via standard header parsers/decorators.
- **Redis:** all keyed commands live inside `libs/atlas-redis` (redis-key-guard). Extend the existing lib, never a new one.
- **Every new `socket.handlers` seed entry carries a validator** (`LoggedInValidator`) — validator-less entries are silently dropped.
- **No test-only constructors / `*_testhelpers.go`.** Dependency injection via exported `NewProcessorWithClients` constructors living in production files.
- **Verification gates before "done":** `go test -race ./...`, `go vet ./...`, `go build ./...` per changed module; `docker buildx bake atlas-channel atlas-ban atlas-messages`; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` and `tools/template-opcode-order-guard.sh` from repo root; `go run ./tools/packet-audit operations --check`, `matrix --check`, `gate-check --check`.
- **Commit style:** `type(scope): summary`, committed from the worktree root; every commit lists exact files.

## Module Roots (all paths repo-relative from the worktree root)

| Alias | Path |
|---|---|
| **PKT** | `libs/atlas-packet` (module `github.com/Chronicle20/atlas/libs/atlas-packet`) |
| **RED** | `libs/atlas-redis` (module `github.com/Chronicle20/atlas/libs/atlas-redis`, package `redis`) |
| **BAN** | `services/atlas-ban/atlas.com/ban` (module `atlas-ban`) |
| **MSG** | `services/atlas-messages/atlas.com/messages` (module `atlas-messages`) |
| **CH**  | `services/atlas-channel/atlas.com/channel` (module `atlas-channel`) |
| **UI**  | `services/atlas-ui` |

---

### Task 1: Packet codecs — SueCharacterResult, ClaimAvailableTime, ClaimSvrStatusChanged

**Files:**
- Create: `libs/atlas-packet/report/clientbound/sue_character_result.go`
- Create: `libs/atlas-packet/report/clientbound/sue_character_result_test.go`
- Create: `libs/atlas-packet/report/clientbound/claim_available_time.go`
- Create: `libs/atlas-packet/report/clientbound/claim_available_time_test.go`
- Create: `libs/atlas-packet/report/clientbound/claim_status_changed.go`
- Create: `libs/atlas-packet/report/clientbound/claim_status_changed_test.go`

**Interfaces:**
- Consumes: `atlas-socket` request/response, `libs/atlas-packet/test` helpers (`pt.CreateContext`, `pt.Encode`, `pt.RoundTrip`, `pt.Variants`).
- Produces: `clientbound.SueCharacterResultWriter = "SueCharacterResult"`, `NewSueCharacterResult(result byte) SueCharacterResult` (getter `Result() byte`); `ClaimAvailableTimeWriter = "ClaimAvailableTime"`, `NewClaimAvailableTime(openHour, closeHour byte) ClaimAvailableTime` (getters `OpenHour()`, `CloseHour()`); `ClaimSvrStatusChangedWriter = "ClaimSvrStatusChanged"`, `NewClaimSvrStatusChanged(connected bool) ClaimSvrStatusChanged` (getter `Connected() bool`). All three have standard `Operation()/String()/Encode(l, ctx)/Decode(l, ctx)` (no version branches — bodies IDA-verified identical v83↔v95, `packet-findings.md` §1/§4/§5).

- [ ] **Step 1: Write the failing tests**

`libs/atlas-packet/report/clientbound/sue_character_result_test.go`:

```go
package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestSueCharacterResultGolden(t *testing.T) {
	// 1 byte result code; 1 = "Unable to locate the user" (packet-findings.md §1).
	input := NewSueCharacterResult(0x01)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSueCharacterResultRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewSueCharacterResult(0x04)
			output := SueCharacterResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Result() != input.Result() {
				t.Errorf("round-trip mismatch: got %d want %d", output.Result(), input.Result())
			}
		})
	}
}
```

`libs/atlas-packet/report/clientbound/claim_available_time_test.go`:

```go
package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestClaimAvailableTimeGolden(t *testing.T) {
	// openHour, closeHour. 0/0 = always available (verified client branch).
	input := NewClaimAvailableTime(8, 22)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x08, 0x16}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimAvailableTimeAlwaysOpenGolden(t *testing.T) {
	input := NewClaimAvailableTime(0, 0)
	ctx := pt.CreateContext("GMS", 95, 1)
	expected := []byte{0x00, 0x00}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimAvailableTimeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewClaimAvailableTime(9, 21)
			output := ClaimAvailableTime{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.OpenHour() != input.OpenHour() || output.CloseHour() != input.CloseHour() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
```

`libs/atlas-packet/report/clientbound/claim_status_changed_test.go`:

```go
package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestClaimSvrStatusChangedGolden(t *testing.T) {
	// 1 byte connected flag; nonzero sets m_bClaimSvrConnected.
	input := NewClaimSvrStatusChanged(true)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimSvrStatusChangedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewClaimSvrStatusChanged(true)
			output := ClaimSvrStatusChanged{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Connected() != input.Connected() {
				t.Errorf("round-trip mismatch: got %v want %v", output.Connected(), input.Connected())
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./report/... 2>&1 | head -20`
Expected: FAIL to build — `undefined: NewSueCharacterResult` (package does not exist yet).

- [ ] **Step 3: Write the codecs**

`libs/atlas-packet/report/clientbound/sue_character_result.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SueCharacterResultWriter = "SueCharacterResult"

// SueCharacterResult - CWvsContext::OnSueCharacterResult. Single result byte
// rendered as a chat-log line (CHATLOG_ADD, not a modal): 0 success,
// 1 unable to locate, 2 daily limit, 3 reported-notice to the accused,
// any other value = generic failure. Byte-identical v61..v95 (only the
// opcode moves: 0x34 on v61/v72/v79, 0x37 on v83+). Version-absent on v48.
// packet-audit:fname CWvsContext::OnSueCharacterResult
type SueCharacterResult struct {
	result byte
}

func NewSueCharacterResult(result byte) SueCharacterResult {
	return SueCharacterResult{result: result}
}

func (m SueCharacterResult) Result() byte { return m.result }

func (m SueCharacterResult) Operation() string { return SueCharacterResultWriter }

func (m SueCharacterResult) String() string {
	return fmt.Sprintf("result [%d]", m.result)
}

func (m SueCharacterResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.result)
		return w.Bytes()
	}
}

func (m *SueCharacterResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.result = r.ReadByte()
	}
}
```

`libs/atlas-packet/report/clientbound/claim_available_time.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ClaimAvailableTimeWriter = "ClaimAvailableTime"

// ClaimAvailableTime - CWvsContext::OnSetClaimSvrAvailableTime. Two bytes:
// open hour -> m_nClaimSvrOpenTime, close hour -> m_nClaimSvrCloseTime.
// open == close == 0 is treated by the client as always-available.
// Byte-identical v72..v95 (opcode 0x2B on v72/v79, 0x2E on v83-v87, 0x2D on v95).
// packet-audit:fname CWvsContext::OnSetClaimSvrAvailableTime
type ClaimAvailableTime struct {
	openHour  byte
	closeHour byte
}

func NewClaimAvailableTime(openHour byte, closeHour byte) ClaimAvailableTime {
	return ClaimAvailableTime{openHour: openHour, closeHour: closeHour}
}

func (m ClaimAvailableTime) OpenHour() byte  { return m.openHour }
func (m ClaimAvailableTime) CloseHour() byte { return m.closeHour }

func (m ClaimAvailableTime) Operation() string { return ClaimAvailableTimeWriter }

func (m ClaimAvailableTime) String() string {
	return fmt.Sprintf("open [%d] close [%d]", m.openHour, m.closeHour)
}

func (m ClaimAvailableTime) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.openHour)
		w.WriteByte(m.closeHour)
		return w.Bytes()
	}
}

func (m *ClaimAvailableTime) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.openHour = r.ReadByte()
		m.closeHour = r.ReadByte()
	}
}
```

`libs/atlas-packet/report/clientbound/claim_status_changed.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ClaimSvrStatusChangedWriter = "ClaimSvrStatusChanged"

// ClaimSvrStatusChanged - CWvsContext::OnClaimSvrStatusChanged. One byte
// connected flag -> m_bClaimSvrConnected. Without connected = 1 the client
// refuses to open the claim dialog. Byte-identical v72..v95.
// packet-audit:fname CWvsContext::OnClaimSvrStatusChanged
type ClaimSvrStatusChanged struct {
	connected bool
}

func NewClaimSvrStatusChanged(connected bool) ClaimSvrStatusChanged {
	return ClaimSvrStatusChanged{connected: connected}
}

func (m ClaimSvrStatusChanged) Connected() bool { return m.connected }

func (m ClaimSvrStatusChanged) Operation() string { return ClaimSvrStatusChangedWriter }

func (m ClaimSvrStatusChanged) String() string {
	return fmt.Sprintf("connected [%t]", m.connected)
}

func (m ClaimSvrStatusChanged) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.connected)
		return w.Bytes()
	}
}

func (m *ClaimSvrStatusChanged) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.connected = r.ReadBool()
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./report/... -run 'SueCharacterResult|ClaimAvailableTime|ClaimSvrStatusChanged' -v 2>&1 | tail -20`
Expected: PASS (all golden + round-trip subtests).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/report/clientbound/
git commit -m "feat(packet): SueCharacterResult, ClaimAvailableTime, ClaimSvrStatusChanged clientbound codecs"
```

---

### Task 2: Packet codecs — ClaimResult (Success + Notice)

**Files:**
- Create: `libs/atlas-packet/report/clientbound/claim_result.go`
- Create: `libs/atlas-packet/report/clientbound/claim_result_test.go`

**Interfaces:**
- Consumes: same as Task 1.
- Produces: `ClaimResultWriter = "ClaimResult"`; `NewClaimResultSuccess(mode byte, hasRemaining bool, remaining int32) ClaimResultSuccess` (getters `Mode()`, `HasRemaining()`, `Remaining()`); `NewClaimResultNotice(mode byte) ClaimResultNotice` (getter `Mode()`). Discrete structs sharing one writer const — the note/clientbound `SendSuccess`/`SendError` pattern. This is NOT a dispatcher family (only mode 2 carries payload; every other arm reads nothing — design §3.1); no `families.yaml` entry.

- [ ] **Step 1: Write the failing tests**

`libs/atlas-packet/report/clientbound/claim_result_test.go`:

```go
package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestClaimResultSuccessGolden(t *testing.T) {
	// mode 2 = success: byte hasRemaining, int32 remaining ("D reports left this week").
	input := NewClaimResultSuccess(0x02, true, 100)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x02, 0x01, 0x64, 0x00, 0x00, 0x00}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimResultNoticeGolden(t *testing.T) {
	// mode 0x42 = "Please re-check the character name then try again" — bare mode byte.
	input := NewClaimResultNotice(0x42)
	ctx := pt.CreateContext("GMS", 95, 1)
	expected := []byte{0x42}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimResultSuccessRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewClaimResultSuccess(0x02, true, 42)
			output := ClaimResultSuccess{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() || output.HasRemaining() != input.HasRemaining() || output.Remaining() != input.Remaining() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}

func TestClaimResultNoticeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewClaimResultNotice(0x41)
			output := ClaimResultNotice{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("round-trip mismatch: got %d want %d", output.Mode(), input.Mode())
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./report/... 2>&1 | head -10`
Expected: FAIL to build — `undefined: NewClaimResultSuccess`.

- [ ] **Step 3: Write the codec**

`libs/atlas-packet/report/clientbound/claim_result.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ClaimResultWriter = "ClaimResult"

// ClaimResult (CWvsContext::OnClaimResult) is a mode byte followed by a
// mode-dependent payload where ONLY mode 2 (success) carries data; every
// other verified mode (3, 0x41-0x45, 0x47, 0x48) is the bare mode byte
// rendered as a CUtilDlg::Notice modal. Mode sets identical v72..v95.
// This is the SetSkillResponse-style "mode + conditional payload" shape,
// not a dispatcher family (design.md §3.1).

// ClaimResultSuccess - mode, byte hasRemaining, int32 remaining.
// packet-audit:fname CWvsContext::OnClaimResult#Success
type ClaimResultSuccess struct {
	mode         byte
	hasRemaining bool
	remaining    int32
}

func NewClaimResultSuccess(mode byte, hasRemaining bool, remaining int32) ClaimResultSuccess {
	return ClaimResultSuccess{mode: mode, hasRemaining: hasRemaining, remaining: remaining}
}

func (m ClaimResultSuccess) Mode() byte         { return m.mode }
func (m ClaimResultSuccess) HasRemaining() bool { return m.hasRemaining }
func (m ClaimResultSuccess) Remaining() int32   { return m.remaining }

func (m ClaimResultSuccess) Operation() string { return ClaimResultWriter }

func (m ClaimResultSuccess) String() string {
	return fmt.Sprintf("mode [%d] hasRemaining [%t] remaining [%d]", m.mode, m.hasRemaining, m.remaining)
}

func (m ClaimResultSuccess) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(m.hasRemaining)
		w.WriteInt32(m.remaining)
		return w.Bytes()
	}
}

func (m *ClaimResultSuccess) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.hasRemaining = r.ReadBool()
		m.remaining = r.ReadInt32()
	}
}

// ClaimResultNotice - bare mode byte (modes 3, 0x41-0x45, 0x47, 0x48).
// packet-audit:fname CWvsContext::OnClaimResult#Notice
type ClaimResultNotice struct {
	mode byte
}

func NewClaimResultNotice(mode byte) ClaimResultNotice {
	return ClaimResultNotice{mode: mode}
}

func (m ClaimResultNotice) Mode() byte { return m.mode }

func (m ClaimResultNotice) Operation() string { return ClaimResultWriter }

func (m ClaimResultNotice) String() string {
	return fmt.Sprintf("mode [%d]", m.mode)
}

func (m ClaimResultNotice) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ClaimResultNotice) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./report/... -run ClaimResult -v 2>&1 | tail -15`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/report/clientbound/claim_result.go libs/atlas-packet/report/clientbound/claim_result_test.go
git commit -m "feat(packet): ClaimResult success/notice clientbound codecs"
```

---

### Task 3: Packet codec — ClaimRequest (serverbound)

**Files:**
- Create: `libs/atlas-packet/report/serverbound/claim_request.go`
- Create: `libs/atlas-packet/report/serverbound/claim_request_test.go`

**Interfaces:**
- Consumes: same as Task 1.
- Produces: `serverbound.ClaimRequestHandle = "ClaimRequest"`; `NewClaimRequest(chatClaim byte, targetName string, reasonType byte, description string, chatLog string) ClaimRequest`; getters `IsChatClaim() bool`, `TargetName() string`, `ReasonType() byte`, `Description() string`, `ChatLog() string`.

- [ ] **Step 1: Write the failing tests**

`libs/atlas-packet/report/serverbound/claim_request_test.go`:

```go
package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestClaimRequestChatClaimGolden(t *testing.T) {
	// bChatClaim=1 appends the client-supplied chat log string.
	input := NewClaimRequest(1, "bob", 0x02, "hi", "yo")
	ctx := pt.CreateContext("GMS", 95, 1)
	expected := []byte{
		0x01,                         // bChatClaim
		0x03, 0x00, 0x62, 0x6F, 0x62, // "bob"
		0x02,                   // nType
		0x02, 0x00, 0x68, 0x69, // "hi"
		0x02, 0x00, 0x79, 0x6F, // "yo"
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimRequestRegularGolden(t *testing.T) {
	// bChatClaim=0: no chat log trailer.
	input := NewClaimRequest(0, "bob", 0x05, "hi", "")
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{
		0x00,
		0x03, 0x00, 0x62, 0x6F, 0x62,
		0x05,
		0x02, 0x00, 0x68, 0x69,
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewClaimRequest(1, "alice", 0x03, "harassment in fm1", "alice: mean words")
			output := ClaimRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.IsChatClaim() != input.IsChatClaim() || output.TargetName() != input.TargetName() ||
				output.ReasonType() != input.ReasonType() || output.Description() != input.Description() ||
				output.ChatLog() != input.ChatLog() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./report/serverbound/... 2>&1 | head -10`
Expected: FAIL to build — `undefined: NewClaimRequest`.

- [ ] **Step 3: Write the codec**

`libs/atlas-packet/report/serverbound/claim_request.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ClaimRequestHandle = "ClaimRequest"

// ClaimRequest - CWvsContext::SendClaimRequest. Sent by the CUIClaim report
// window. Body (v95-verified; v72 @0x91f2b4 and v79 @0x9711ff verified
// 2026-08-04; v83 send-site named and byte-verified as part of this task's
// IDA work — packet-findings.md §2):
//
//	byte   bChatClaim   1 = chat/harassment claim, 0 = regular claim
//	string sTargetCharacterName
//	byte   nType        reason type from CUIClaim::GetResult
//	string sContext     free-text description
//	if bChatClaim == 1: string chatLog (client-supplied two-character log)
//
// No version branch: body identical across v72..v95 per findings. The packet
// does not exist below v72 (verified absent on v48/v61). If the v83
// byte-verification ever disagrees, add an inline t.MajorVersion() guard
// here at that point.
// packet-audit:fname CWvsContext::SendClaimRequest
type ClaimRequest struct {
	chatClaim   byte
	targetName  string
	reasonType  byte
	description string
	chatLog     string
}

func NewClaimRequest(chatClaim byte, targetName string, reasonType byte, description string, chatLog string) ClaimRequest {
	return ClaimRequest{chatClaim: chatClaim, targetName: targetName, reasonType: reasonType, description: description, chatLog: chatLog}
}

func (m ClaimRequest) IsChatClaim() bool   { return m.chatClaim == 1 }
func (m ClaimRequest) TargetName() string  { return m.targetName }
func (m ClaimRequest) ReasonType() byte    { return m.reasonType }
func (m ClaimRequest) Description() string { return m.description }
func (m ClaimRequest) ChatLog() string     { return m.chatLog }

func (m ClaimRequest) Operation() string { return ClaimRequestHandle }

func (m ClaimRequest) String() string {
	return fmt.Sprintf("chatClaim [%d], target [%s], type [%d], description [%s]", m.chatClaim, m.targetName, m.reasonType, m.description)
}

func (m ClaimRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.chatClaim)
		w.WriteAsciiString(m.targetName)
		w.WriteByte(m.reasonType)
		w.WriteAsciiString(m.description)
		if m.chatClaim == 1 {
			w.WriteAsciiString(m.chatLog)
		}
		return w.Bytes()
	}
}

func (m *ClaimRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.chatClaim = r.ReadByte()
		m.targetName = r.ReadAsciiString()
		m.reasonType = r.ReadByte()
		m.description = r.ReadAsciiString()
		if m.chatClaim == 1 {
			m.chatLog = r.ReadAsciiString()
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./report/... -v 2>&1 | tail -15`
Expected: PASS (all report package tests).

- [ ] **Step 5: Run module gates and commit**

Run: `cd libs/atlas-packet && go vet ./... && go test -race ./report/...`
Expected: clean.

```bash
git add libs/atlas-packet/report/serverbound/
git commit -m "feat(packet): ClaimRequest serverbound codec"
```

---

### Task 4: atlas-ban — Kafka contract + report entity/model/builder

**Files:**
- Create: `services/atlas-ban/atlas.com/ban/kafka/message/report/kafka.go`
- Create: `services/atlas-ban/atlas.com/ban/report/entity.go`
- Create: `services/atlas-ban/atlas.com/ban/report/model.go`
- Create: `services/atlas-ban/atlas.com/ban/report/builder.go`
- Create: `services/atlas-ban/atlas.com/ban/report/builder_test.go`

**Interfaces:**
- Consumes: `github.com/Chronicle20/atlas/libs/atlas-constants/world`, `.../channel`, `github.com/google/uuid`, gorm.
- Produces (Kafka contract, mirrored verbatim on the channel side in Task 13): `report.EnvCommandTopic = "COMMAND_TOPIC_REPORT"`, `CommandTypeCreate = "CREATE"`, `EnvEventTopicStatus = "EVENT_TOPIC_REPORT_STATUS"`, `EventStatusCreated = "CREATED"`, `EventStatusError = "ERROR"`, `ErrorCodeNotFound = "NOT_FOUND"`, `ErrorCodeInternal = "INTERNAL"`, `KindSue = "sue"`, `KindClaim = "claim"`, `Command[E]`, `CreateCommandBody`, `StatusEvent`.
- Produces (domain): `report.Kind` / `report.Status` string types with `Valid()`; `report.TranscriptLine`; caps `MaxDescriptionLength = 2000`, `MaxChatLogBytes = 16384`; `ErrInvalidStatus`, `ErrInvalidKind`; immutable `Model` with getters `Id() uuid.UUID`, `TenantId()`, `Kind() Kind`, `ReporterId() uint32`, `ReporterName() string`, `AccusedId() uint32`, `AccusedName() string`, `ReasonType() byte`, `Description() string`, `ChatLog() *string`, `ServerTranscript() []TranscriptLine`, `Status() Status`, `CreatedAt()`, `UpdatedAt()`; `NewBuilder(tenantId uuid.UUID, kind Kind, reporterId uint32) *Builder` with `SetId/SetReporterName/SetAccusedId/SetAccusedName/SetReasonType/SetDescription/SetChatLog/SetServerTranscript/SetStatus/SetCreatedAt/SetUpdatedAt` and validating `Build() (Model, error)`; `report.Migration`; `Entity` with `TableName() "reports"`.

- [ ] **Step 1: Write the Kafka contract file (no test — constants and structs only)**

`services/atlas-ban/atlas.com/ban/kafka/message/report/kafka.go`:

```go
package report

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic   = "COMMAND_TOPIC_REPORT"
	CommandTypeCreate = "CREATE"

	EnvEventTopicStatus = "EVENT_TOPIC_REPORT_STATUS"
	EventStatusCreated  = "CREATED"
	EventStatusError    = "ERROR"

	ErrorCodeNotFound = "NOT_FOUND"
	ErrorCodeInternal = "INTERNAL"

	KindSue   = "sue"
	KindClaim = "claim"
)

type Command[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

// CreateCommandBody carries the report exactly as supplied on the wire.
// Accused identity is mechanism-dependent: claim and v95 sue supply
// AccusedName (v95 sue's sub-command string is treated as the target name);
// legacy sue (v83/v84/v87) supplies AccusedId. The consumer resolves the
// missing half via atlas-character and rejects unresolvable targets.
type CreateCommandBody struct {
	Kind        string     `json:"kind"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	ReporterId  uint32     `json:"reporterId"`
	AccusedId   uint32     `json:"accusedId"`
	AccusedName string     `json:"accusedName"`
	ReasonType  byte       `json:"reasonType"`
	Description string     `json:"description"`
	ChatClaim   bool       `json:"chatClaim"`
	ChatLog     string     `json:"chatLog"`
}

type StatusEvent struct {
	ReportId   uuid.UUID `json:"reportId"` // uuid.Nil on ERROR
	Kind       string    `json:"kind"`
	WorldId    world.Id  `json:"worldId"`
	ReporterId uint32    `json:"reporterId"`
	Status     string    `json:"status"`    // CREATED | ERROR
	ErrorCode  string    `json:"errorCode"` // NOT_FOUND | INTERNAL; empty on CREATED
}
```

- [ ] **Step 2: Write the failing builder test**

`services/atlas-ban/atlas.com/ban/report/builder_test.go`:

```go
package report

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuilderBuildsValidReport(t *testing.T) {
	tenantId := uuid.New()
	chatLog := "alice: hi"
	m, err := NewBuilder(tenantId, KindClaim, 1).
		SetReporterName("Reporter").
		SetAccusedId(2).
		SetAccusedName("Accused").
		SetReasonType(3).
		SetDescription("harassment").
		SetChatLog(&chatLog).
		SetServerTranscript([]TranscriptLine{{Timestamp: 1, SenderId: 1, SenderName: "Reporter", ChatType: "GENERAL", Text: "hi"}}).
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if m.TenantId() != tenantId || m.Kind() != KindClaim || m.ReporterId() != 1 {
		t.Errorf("core fields mismatch: %+v", m)
	}
	if m.Status() != StatusOpen {
		t.Errorf("default status: got %s want %s", m.Status(), StatusOpen)
	}
	if m.ChatLog() == nil || *m.ChatLog() != chatLog {
		t.Errorf("chat log mismatch")
	}
	if len(m.ServerTranscript()) != 1 || m.ServerTranscript()[0].Text != "hi" {
		t.Errorf("transcript mismatch: %+v", m.ServerTranscript())
	}
}

func TestBuilderRejectsInvalidKind(t *testing.T) {
	_, err := NewBuilder(uuid.New(), Kind("bogus"), 1).SetAccusedName("x").Build()
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

func TestBuilderRejectsInvalidStatus(t *testing.T) {
	_, err := NewBuilder(uuid.New(), KindSue, 1).SetAccusedName("x").SetStatus(Status("bogus")).Build()
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestStatusValid(t *testing.T) {
	for _, s := range []Status{StatusOpen, StatusReviewed, StatusActioned} {
		if !s.Valid() {
			t.Errorf("expected %s valid", s)
		}
	}
	if Status("closed").Valid() {
		t.Error("expected 'closed' invalid")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd services/atlas-ban/atlas.com/ban && go test ./report/... 2>&1 | head -5`
Expected: FAIL to build — package `report` does not exist.

- [ ] **Step 4: Write entity, model, builder**

`services/atlas-ban/atlas.com/ban/report/entity.go`:

```go
package report

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Entity is the durable report row. Id is a surrogate uuid generated in Go
// at create time (never a business-value PK; works across Postgres and the
// sqlite test driver). ServerTranscript is a marshaled []TranscriptLine
// snapshot taken at creation; nil when atlas-messages was unreachable.
type Entity struct {
	Id               uuid.UUID `gorm:"primaryKey;type:uuid"`
	TenantId         uuid.UUID `gorm:"not null;index:idx_reports_tenant_status,priority:1"`
	Kind             string    `gorm:"not null"`
	ReporterId       uint32    `gorm:"not null"`
	ReporterName     string    `gorm:"not null"`
	AccusedId        uint32    `gorm:"not null"`
	AccusedName      string    `gorm:"not null"`
	ReasonType       byte      `gorm:"not null"`
	Description      string    `gorm:"type:text;not null"`
	ChatLog          *string   `gorm:"type:text"`
	ServerTranscript []byte    `gorm:"type:jsonb"`
	Status           string    `gorm:"not null;default:open;index:idx_reports_tenant_status,priority:2"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (e Entity) TableName() string {
	return "reports"
}
```

`services/atlas-ban/atlas.com/ban/report/model.go`:

```go
package report

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidStatus = errors.New("invalid report status")
	ErrInvalidKind   = errors.New("invalid report kind")
)

// Server-side caps on client-supplied strings (NFR): truncate-and-log,
// never reject — a truncated report is more useful to a GM than a vanished
// one. Description is capped in characters (bytes), chat log in bytes.
const (
	MaxDescriptionLength = 2000
	MaxChatLogBytes      = 16384
)

type Kind string

const (
	KindSue   Kind = "sue"
	KindClaim Kind = "claim"
)

func (k Kind) Valid() bool {
	return k == KindSue || k == KindClaim
}

type Status string

const (
	StatusOpen     Status = "open"
	StatusReviewed Status = "reviewed"
	StatusActioned Status = "actioned"
)

func (s Status) Valid() bool {
	return s == StatusOpen || s == StatusReviewed || s == StatusActioned
}

// TranscriptLine is one server-captured chat line attached to a report.
type TranscriptLine struct {
	Timestamp  int64  `json:"timestamp"`
	SenderId   uint32 `json:"senderId"`
	SenderName string `json:"senderName"`
	ChatType   string `json:"chatType"`
	Text       string `json:"text"`
}

type Model struct {
	id               uuid.UUID
	tenantId         uuid.UUID
	kind             Kind
	reporterId       uint32
	reporterName     string
	accusedId        uint32
	accusedName      string
	reasonType       byte
	description      string
	chatLog          *string
	serverTranscript []TranscriptLine
	status           Status
	createdAt        time.Time
	updatedAt        time.Time
}

func (m Model) Id() uuid.UUID                       { return m.id }
func (m Model) TenantId() uuid.UUID                 { return m.tenantId }
func (m Model) Kind() Kind                          { return m.kind }
func (m Model) ReporterId() uint32                  { return m.reporterId }
func (m Model) ReporterName() string                { return m.reporterName }
func (m Model) AccusedId() uint32                   { return m.accusedId }
func (m Model) AccusedName() string                 { return m.accusedName }
func (m Model) ReasonType() byte                    { return m.reasonType }
func (m Model) Description() string                 { return m.description }
func (m Model) ChatLog() *string                    { return m.chatLog }
func (m Model) ServerTranscript() []TranscriptLine  { return m.serverTranscript }
func (m Model) Status() Status                      { return m.status }
func (m Model) CreatedAt() time.Time                { return m.createdAt }
func (m Model) UpdatedAt() time.Time                { return m.updatedAt }
```

`services/atlas-ban/atlas.com/ban/report/builder.go`:

```go
package report

import (
	"time"

	"github.com/google/uuid"
)

type Builder struct {
	id               uuid.UUID
	tenantId         uuid.UUID
	kind             Kind
	reporterId       uint32
	reporterName     string
	accusedId        uint32
	accusedName      string
	reasonType       byte
	description      string
	chatLog          *string
	serverTranscript []TranscriptLine
	status           Status
	createdAt        time.Time
	updatedAt        time.Time
}

func NewBuilder(tenantId uuid.UUID, kind Kind, reporterId uint32) *Builder {
	return &Builder{
		tenantId:   tenantId,
		kind:       kind,
		reporterId: reporterId,
		status:     StatusOpen,
	}
}

func (b *Builder) SetId(id uuid.UUID) *Builder                            { b.id = id; return b }
func (b *Builder) SetReporterName(name string) *Builder                   { b.reporterName = name; return b }
func (b *Builder) SetAccusedId(id uint32) *Builder                        { b.accusedId = id; return b }
func (b *Builder) SetAccusedName(name string) *Builder                    { b.accusedName = name; return b }
func (b *Builder) SetReasonType(reasonType byte) *Builder                 { b.reasonType = reasonType; return b }
func (b *Builder) SetDescription(description string) *Builder             { b.description = description; return b }
func (b *Builder) SetChatLog(chatLog *string) *Builder                    { b.chatLog = chatLog; return b }
func (b *Builder) SetServerTranscript(lines []TranscriptLine) *Builder    { b.serverTranscript = lines; return b }
func (b *Builder) SetStatus(status Status) *Builder                       { b.status = status; return b }
func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder              { b.createdAt = createdAt; return b }
func (b *Builder) SetUpdatedAt(updatedAt time.Time) *Builder              { b.updatedAt = updatedAt; return b }

func (b *Builder) Build() (Model, error) {
	if !b.kind.Valid() {
		return Model{}, ErrInvalidKind
	}
	if !b.status.Valid() {
		return Model{}, ErrInvalidStatus
	}
	return Model{
		id:               b.id,
		tenantId:         b.tenantId,
		kind:             b.kind,
		reporterId:       b.reporterId,
		reporterName:     b.reporterName,
		accusedId:        b.accusedId,
		accusedName:      b.accusedName,
		reasonType:       b.reasonType,
		description:      b.description,
		chatLog:          b.chatLog,
		serverTranscript: b.serverTranscript,
		status:           b.status,
		createdAt:        b.createdAt,
		updatedAt:        b.updatedAt,
	}, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-ban/atlas.com/ban && go test ./report/... -v 2>&1 | tail -12`
Expected: PASS (4 tests).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ban/atlas.com/ban/kafka/message/report/ services/atlas-ban/atlas.com/ban/report/
git commit -m "feat(ban): report kafka contract, entity, model, builder"
```

---

### Task 5: atlas-ban — report provider + administrator + migration wiring

**Files:**
- Create: `services/atlas-ban/atlas.com/ban/report/provider.go`
- Create: `services/atlas-ban/atlas.com/ban/report/administrator.go`
- Create: `services/atlas-ban/atlas.com/ban/report/administrator_test.go`
- Modify: `services/atlas-ban/atlas.com/ban/main.go` (line 57: add `report.Migration`)

**Interfaces:**
- Consumes: Task 4 types; `database.EntityProvider`, `model.Provider` (as in `ban/provider.go`).
- Produces: `create(db *gorm.DB) func(tenantId uuid.UUID, kind Kind, reporterId uint32, reporterName string, accusedId uint32, accusedName string, reasonType byte, description string, chatLog *string, transcript []TranscriptLine) (Model, error)`; `updateStatus(db *gorm.DB) func(id uuid.UUID, status Status) error` (returns `gorm.ErrRecordNotFound` when no row matched); `Make(e Entity) (Model, error)`; `entityById(id uuid.UUID) database.EntityProvider[Entity]`; `entitiesByTenant() database.EntityProvider[[]Entity]`; `entitiesByStatus(status Status) database.EntityProvider[[]Entity]`.

- [ ] **Step 1: Write the failing tests**

`services/atlas-ban/atlas.com/ban/report/administrator_test.go` (mirrors `ban/processor_test.go` setup — sqlite + `database.RegisterTenantCallbacks`):

```go
package report

import (
	"context"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDatabase(t *testing.T) *gorm.DB {
	l, _ := test.NewNullLogger()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err = db.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
	}
	return db
}

func sampleTenant() tenant.Model {
	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tm
}

func testContext(tm tenant.Model) context.Context {
	return tenant.WithContext(context.Background(), tm)
}

func TestCreateAndFetchReport(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	tdb := db.WithContext(testContext(tm))

	chatLog := "bob: rude things"
	m, err := create(tdb)(tm.Id(), KindClaim, 1, "Reporter", 2, "Accused", 3, "harassment", &chatLog,
		[]TranscriptLine{{Timestamp: 10, SenderId: 2, SenderName: "Accused", ChatType: "GENERAL", Text: "rude things"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if m.Id() == uuid.Nil {
		t.Fatal("expected generated id")
	}

	e, err := entityById(m.Id())(tdb)()
	if err != nil {
		t.Fatalf("entityById: %v", err)
	}
	got, err := Make(e)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if got.Kind() != KindClaim || got.Status() != StatusOpen || got.AccusedName() != "Accused" {
		t.Errorf("fetched mismatch: %+v", got)
	}
	if got.ChatLog() == nil || *got.ChatLog() != chatLog {
		t.Error("chat log not persisted")
	}
	if len(got.ServerTranscript()) != 1 || got.ServerTranscript()[0].Text != "rude things" {
		t.Errorf("transcript not round-tripped: %+v", got.ServerTranscript())
	}
}

func TestCreateNilTranscriptStaysNil(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	tdb := db.WithContext(testContext(tm))

	m, err := create(tdb)(tm.Id(), KindSue, 1, "Reporter", 2, "Accused", 0, "spamming", nil, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	e, err := entityById(m.Id())(tdb)()
	if err != nil {
		t.Fatalf("entityById: %v", err)
	}
	got, _ := Make(e)
	if got.ChatLog() != nil || got.ServerTranscript() != nil {
		t.Errorf("expected nil chat log and transcript, got %+v / %+v", got.ChatLog(), got.ServerTranscript())
	}
}

func TestUpdateStatus(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	tdb := db.WithContext(testContext(tm))

	m, _ := create(tdb)(tm.Id(), KindSue, 1, "Reporter", 2, "Accused", 0, "spamming", nil, nil)
	if err := updateStatus(tdb)(m.Id(), StatusReviewed); err != nil {
		t.Fatalf("updateStatus: %v", err)
	}
	e, _ := entityById(m.Id())(tdb)()
	if e.Status != string(StatusReviewed) {
		t.Errorf("status: got %s want %s", e.Status, StatusReviewed)
	}
	if err := updateStatus(tdb)(uuid.New(), StatusReviewed); err == nil {
		t.Error("expected error for unknown id")
	}
}

func TestEntitiesByStatusFilters(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	tdb := db.WithContext(testContext(tm))

	m1, _ := create(tdb)(tm.Id(), KindSue, 1, "R", 2, "A", 0, "one", nil, nil)
	_, _ = create(tdb)(tm.Id(), KindSue, 1, "R", 3, "B", 0, "two", nil, nil)
	_ = updateStatus(tdb)(m1.Id(), StatusActioned)

	open, err := entitiesByStatus(StatusOpen)(tdb)()
	if err != nil {
		t.Fatalf("entitiesByStatus: %v", err)
	}
	if len(open) != 1 || open[0].Description != "two" {
		t.Errorf("open filter mismatch: %+v", open)
	}
	all, _ := entitiesByTenant()(tdb)()
	if len(all) != 2 {
		t.Errorf("expected 2 rows, got %d", len(all))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-ban/atlas.com/ban && go test ./report/... 2>&1 | head -5`
Expected: FAIL to build — `undefined: create`.

- [ ] **Step 3: Write provider and administrator**

`services/atlas-ban/atlas.com/ban/report/provider.go`:

```go
package report

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func entityById(id uuid.UUID) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("id = ?", id).First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider[Entity](result)
	}
}

func entitiesByTenant() database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Order("created_at DESC").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}

func entitiesByStatus(status Status) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("status = ?", string(status)).Order("created_at DESC").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}
```

`services/atlas-ban/atlas.com/ban/report/administrator.go`:

```go
package report

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB) func(tenantId uuid.UUID, kind Kind, reporterId uint32, reporterName string, accusedId uint32, accusedName string, reasonType byte, description string, chatLog *string, transcript []TranscriptLine) (Model, error) {
	return func(tenantId uuid.UUID, kind Kind, reporterId uint32, reporterName string, accusedId uint32, accusedName string, reasonType byte, description string, chatLog *string, transcript []TranscriptLine) (Model, error) {
		var transcriptJSON []byte
		if transcript != nil {
			var err error
			transcriptJSON, err = json.Marshal(transcript)
			if err != nil {
				return Model{}, err
			}
		}

		e := &Entity{
			Id:               uuid.New(),
			TenantId:         tenantId,
			Kind:             string(kind),
			ReporterId:       reporterId,
			ReporterName:     reporterName,
			AccusedId:        accusedId,
			AccusedName:      accusedName,
			ReasonType:       reasonType,
			Description:      description,
			ChatLog:          chatLog,
			ServerTranscript: transcriptJSON,
			Status:           string(StatusOpen),
		}

		err := db.Create(e).Error
		if err != nil {
			return Model{}, err
		}
		return Make(*e)
	}
}

func updateStatus(db *gorm.DB) func(id uuid.UUID, status Status) error {
	return func(id uuid.UUID, status Status) error {
		result := db.Model(&Entity{}).Where("id = ?", id).Update("status", string(status))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	}
}

func Make(e Entity) (Model, error) {
	var transcript []TranscriptLine
	if len(e.ServerTranscript) > 0 {
		if err := json.Unmarshal(e.ServerTranscript, &transcript); err != nil {
			return Model{}, err
		}
	}
	return NewBuilder(e.TenantId, Kind(e.Kind), e.ReporterId).
		SetId(e.Id).
		SetReporterName(e.ReporterName).
		SetAccusedId(e.AccusedId).
		SetAccusedName(e.AccusedName).
		SetReasonType(e.ReasonType).
		SetDescription(e.Description).
		SetChatLog(e.ChatLog).
		SetServerTranscript(transcript).
		SetStatus(Status(e.Status)).
		SetCreatedAt(e.CreatedAt).
		SetUpdatedAt(e.UpdatedAt).
		Build()
}
```

- [ ] **Step 4: Wire the migration**

In `services/atlas-ban/atlas.com/ban/main.go`, add the import `"atlas-ban/report"` and change line 57:

```go
db := database.Connect(l, database.SetMigrations(ban.Migration, history.Migration, report.Migration))
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-ban/atlas.com/ban && go test ./report/... -v 2>&1 | tail -12 && go build ./...`
Expected: PASS; build clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ban/atlas.com/ban/report/ services/atlas-ban/atlas.com/ban/main.go
git commit -m "feat(ban): report provider, administrator, migration wiring"
```

---

### Task 6: atlas-ban — character & chat REST client packages

**Files:**
- Create: `services/atlas-ban/atlas.com/ban/character/requests.go`
- Create: `services/atlas-ban/atlas.com/ban/character/rest.go`
- Create: `services/atlas-ban/atlas.com/ban/character/processor.go`
- Create: `services/atlas-ban/atlas.com/ban/chat/requests.go`
- Create: `services/atlas-ban/atlas.com/ban/chat/rest.go`
- Create: `services/atlas-ban/atlas.com/ban/chat/rest_test.go`
- Create: `services/atlas-ban/atlas.com/ban/chat/processor.go`

**Interfaces:**
- Consumes: `github.com/Chronicle20/atlas/libs/atlas-rest/requests` (`RootUrl`, `GetRequest`, `Provider`, `SliceProvider`), `libs/atlas-model/model` (`First`, `Filters`, `ErrNoResultFound`).
- Produces: `character.Processor` with `GetById(characterId uint32) (Model, error)` and `GetByName(name string) (Model, error)`; `character.Model` getters `Id() uint32`, `Name() string`. `chat.Processor` with `RecentInvolving(characterIds []uint32) ([]Model, error)`; `chat.Model` getters `Timestamp() int64`, `SenderId() uint32`, `SenderName() string`, `ChatType() string`, `Text() string`. Service URLs resolve via `requests.RootUrl("CHARACTERS")` / `requests.RootUrl("MESSAGES")` (env `CHARACTERS_SERVICE_URL` / `MESSAGES_SERVICE_URL`, `BASE_SERVICE_URL` fallback — never hard-coded).
- The chat RestModel field set matches the atlas-messages resource created in Task 12 (resource type `chat-messages`).

- [ ] **Step 1: Write the failing chat Extract test**

`services/atlas-ban/atlas.com/ban/chat/rest_test.go`:

```go
package chat

import "testing"

func TestExtract(t *testing.T) {
	rm := RestModel{
		Id:         "0",
		Timestamp:  1720540800123,
		SenderId:   7,
		SenderName: "Alice",
		ChatType:   "GENERAL",
		Text:       "hello",
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Timestamp() != 1720540800123 || m.SenderId() != 7 || m.SenderName() != "Alice" ||
		m.ChatType() != "GENERAL" || m.Text() != "hello" {
		t.Errorf("mismatch: %+v", m)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ban/atlas.com/ban && go test ./chat/... 2>&1 | head -5`
Expected: FAIL to build — package does not exist.

- [ ] **Step 3: Write the character client**

`services/atlas-ban/atlas.com/ban/character/requests.go`:

```go
package character

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters"
	ById     = Resource + "/%d"
	ByName   = Resource + "?name=%s"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}

func requestByName(name string) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByName, name))
}
```

`services/atlas-ban/atlas.com/ban/character/rest.go`:

```go
package character

import "strconv"

type RestModel struct {
	Id   uint32 `json:"-"`
	Name string `json:"name"`
}

func (r RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{id: rm.Id, name: rm.Name}, nil
}
```

`services/atlas-ban/atlas.com/ban/character/processor.go`:

```go
package character

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Model struct {
	id   uint32
	name string
}

func (m Model) Id() uint32   { return m.id }
func (m Model) Name() string { return m.name }

type Processor interface {
	GetById(characterId uint32) (Model, error)
	GetByName(name string) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)()
}

func (p *ProcessorImpl) GetByName(name string) (Model, error) {
	ps := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByName(name), Extract, model.Filters[Model]())
	return model.First(ps, model.Filters[Model]())
}
```

- [ ] **Step 4: Write the chat client**

`services/atlas-ban/atlas.com/ban/chat/requests.go`:

```go
package chat

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	HistoryByCharacterIds = "chat/history?characterIds=%s"
)

func getBaseRequest() string {
	return requests.RootUrl("MESSAGES")
}

func requestHistory(characterIds []uint32) requests.Request[[]RestModel] {
	ids := make([]string, 0, len(characterIds))
	for _, id := range characterIds {
		ids = append(ids, strconv.FormatUint(uint64(id), 10))
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+HistoryByCharacterIds, strings.Join(ids, ",")))
}
```

`services/atlas-ban/atlas.com/ban/chat/rest.go`:

```go
package chat

// RestModel mirrors the atlas-messages "chat-messages" resource.
type RestModel struct {
	Id         string `json:"-"`
	Timestamp  int64  `json:"timestamp"`
	SenderId   uint32 `json:"senderId"`
	SenderName string `json:"senderName"`
	ChatType   string `json:"chatType"`
	Text       string `json:"text"`
}

func (r RestModel) GetName() string {
	return "chat-messages"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		timestamp:  rm.Timestamp,
		senderId:   rm.SenderId,
		senderName: rm.SenderName,
		chatType:   rm.ChatType,
		text:       rm.Text,
	}, nil
}
```

`services/atlas-ban/atlas.com/ban/chat/processor.go`:

```go
package chat

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Model struct {
	timestamp  int64
	senderId   uint32
	senderName string
	chatType   string
	text       string
}

func (m Model) Timestamp() int64   { return m.timestamp }
func (m Model) SenderId() uint32   { return m.senderId }
func (m Model) SenderName() string { return m.senderName }
func (m Model) ChatType() string   { return m.chatType }
func (m Model) Text() string       { return m.text }

type Processor interface {
	// RecentInvolving returns the buffered chat lines authored by any of the
	// listed characters, merged and sorted ascending by timestamp.
	RecentInvolving(characterIds []uint32) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) RecentInvolving(characterIds []uint32) ([]Model, error) {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestHistory(characterIds), Extract, model.Filters[Model]())()
}
```

(`requests.SliceProvider` already applies `Extract` — do not double-map; this matches `character.ByNameProvider` usage in atlas-messages.)

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-ban/atlas.com/ban && go test ./chat/... ./character/... -v 2>&1 | tail -8 && go build ./...`
Expected: PASS (chat Extract test); build clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ban/atlas.com/ban/character/ services/atlas-ban/atlas.com/ban/chat/
git commit -m "feat(ban): character and chat REST client packages"
```

---

### Task 7: atlas-ban — report processor + status-event producer

**Files:**
- Create: `services/atlas-ban/atlas.com/ban/report/producer.go`
- Create: `services/atlas-ban/atlas.com/ban/report/processor.go`
- Create: `services/atlas-ban/atlas.com/ban/report/processor_test.go`

**Interfaces:**
- Consumes: Tasks 4–6 (`report2 "atlas-ban/kafka/message/report"`, `"atlas-ban/character"`, `"atlas-ban/chat"`), `"atlas-ban/kafka/message"` buffer/Emit, `"atlas-ban/kafka/producer"`.
- Produces:

```go
type Processor interface {
	CreateFromCommand(buf *message.Buffer) func(c report2.CreateCommandBody) error
	CreateFromCommandAndEmit(c report2.CreateCommandBody) error
	UpdateStatus(reportId uuid.UUID, status Status) (Model, error)
	GetById(reportId uuid.UUID) (Model, error)
	ByIdProvider(reportId uuid.UUID) model.Provider[Model]
	GetByTenant() ([]Model, error)
	GetByStatus(status Status) ([]Model, error)
}
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor
func NewProcessorWithClients(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, charP character.Processor, chatP chat.Processor) Processor
```

`NewProcessorWithClients` is a production constructor (dependency injection), not a test helper; `NewProcessor` delegates to it with the real REST clients.

- [ ] **Step 1: Write local mock clients + failing processor tests**

Character/chat fakes are defined inline in the test file (they implement the two-method interfaces from Task 6; no `_testhelpers.go` files):

`services/atlas-ban/atlas.com/ban/report/processor_test.go`:

```go
package report

import (
	"errors"
	"strings"
	"testing"

	"atlas-ban/character"
	"atlas-ban/chat"
	"atlas-ban/kafka/message"
	report2 "atlas-ban/kafka/message/report"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"
)

type fakeCharacterProcessor struct {
	byId   map[uint32]character.Model
	byName map[string]character.Model
	err    error
}

func (f *fakeCharacterProcessor) GetById(characterId uint32) (character.Model, error) {
	if f.err != nil {
		return character.Model{}, f.err
	}
	m, ok := f.byId[characterId]
	if !ok {
		return character.Model{}, requests.ErrNotFound
	}
	return m, nil
}

func (f *fakeCharacterProcessor) GetByName(name string) (character.Model, error) {
	if f.err != nil {
		return character.Model{}, f.err
	}
	m, ok := f.byName[name]
	if !ok {
		return character.Model{}, model.ErrNoResultFound
	}
	return m, nil
}

type fakeChatProcessor struct {
	lines []chat.Model
	err   error
}

func (f *fakeChatProcessor) RecentInvolving(_ []uint32) ([]chat.Model, error) {
	return f.lines, f.err
}

func makeCharacter(t *testing.T, id uint32, name string) character.Model {
	t.Helper()
	m, err := character.Extract(character.RestModel{Id: id, Name: name})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return m
}

func TestCreateFromCommandHappyPathClaim(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	l, _ := test.NewNullLogger()
	charP := &fakeCharacterProcessor{
		byId:   map[uint32]character.Model{1: makeCharacter(t, 1, "Reporter")},
		byName: map[string]character.Model{"Accused": makeCharacter(t, 2, "Accused")},
	}
	chatP := &fakeChatProcessor{lines: []chat.Model{}}
	p := NewProcessorWithClients(l, testContext(tm), db, charP, chatP)

	buf := message.NewBuffer()
	err := p.CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindClaim, ReporterId: 1, AccusedName: "Accused",
		ReasonType: 3, Description: "harassment", ChatClaim: true, ChatLog: "log",
	})
	if err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}

	reports, err := p.GetByTenant()
	if err != nil {
		t.Fatalf("GetByTenant: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	m := reports[0]
	if m.AccusedId() != 2 || m.AccusedName() != "Accused" || m.ReporterName() != "Reporter" {
		t.Errorf("resolution mismatch: %+v", m)
	}
	if m.ChatLog() == nil || *m.ChatLog() != "log" {
		t.Error("chat log not stored verbatim")
	}
	msgs := buf.GetAll()
	if len(msgs[report2.EnvEventTopicStatus]) != 1 {
		t.Fatalf("expected 1 status event, got %+v", msgs)
	}
}

func TestCreateFromCommandNotFoundPersistsNothing(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	l, _ := test.NewNullLogger()
	charP := &fakeCharacterProcessor{
		byId:   map[uint32]character.Model{1: makeCharacter(t, 1, "Reporter")},
		byName: map[string]character.Model{},
	}
	p := NewProcessorWithClients(l, testContext(tm), db, charP, &fakeChatProcessor{})

	buf := message.NewBuffer()
	err := p.CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindClaim, ReporterId: 1, AccusedName: "Ghost",
		ReasonType: 3, Description: "x",
	})
	if err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	reports, _ := p.GetByTenant()
	if len(reports) != 0 {
		t.Fatalf("expected no persisted report, got %d", len(reports))
	}
	// The error status event must still be buffered so the reporter gets
	// the not-found result packet.
	msgs := buf.GetAll()
	if len(msgs[report2.EnvEventTopicStatus]) != 1 {
		t.Fatalf("expected 1 status event, got %+v", msgs)
	}
}

func TestCreateFromCommandCharacterServiceDownIsInternal(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	l, _ := test.NewNullLogger()
	charP := &fakeCharacterProcessor{err: errors.New("connection refused")}
	p := NewProcessorWithClients(l, testContext(tm), db, charP, &fakeChatProcessor{})

	buf := message.NewBuffer()
	if err := p.CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindSue, ReporterId: 1, AccusedId: 2, Description: "x",
	}); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	reports, _ := p.GetByTenant()
	if len(reports) != 0 {
		t.Fatal("expected no persisted report")
	}
}

func TestCreateFromCommandTruncatesOversizedInputs(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	l, _ := test.NewNullLogger()
	charP := &fakeCharacterProcessor{
		byId: map[uint32]character.Model{
			1: makeCharacter(t, 1, "Reporter"),
			2: makeCharacter(t, 2, "Accused"),
		},
	}
	p := NewProcessorWithClients(l, testContext(tm), db, charP, &fakeChatProcessor{})

	buf := message.NewBuffer()
	longDescription := strings.Repeat("d", MaxDescriptionLength+500)
	longLog := strings.Repeat("c", MaxChatLogBytes+500)
	if err := p.CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindClaim, ReporterId: 1, AccusedId: 2,
		Description: longDescription, ChatClaim: true, ChatLog: longLog,
	}); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	reports, _ := p.GetByTenant()
	if len(reports) != 1 {
		t.Fatal("expected persisted report")
	}
	if len(reports[0].Description()) != MaxDescriptionLength {
		t.Errorf("description not capped: %d", len(reports[0].Description()))
	}
	if reports[0].ChatLog() == nil || len(*reports[0].ChatLog()) != MaxChatLogBytes {
		t.Error("chat log not capped")
	}
}

func TestCreateFromCommandTranscriptFailureTolerated(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	l, _ := test.NewNullLogger()
	charP := &fakeCharacterProcessor{
		byId: map[uint32]character.Model{
			1: makeCharacter(t, 1, "Reporter"),
			2: makeCharacter(t, 2, "Accused"),
		},
	}
	p := NewProcessorWithClients(l, testContext(tm), db, charP, &fakeChatProcessor{err: errors.New("messages down")})

	buf := message.NewBuffer()
	if err := p.CreateFromCommand(buf)(report2.CreateCommandBody{
		Kind: report2.KindSue, ReporterId: 1, AccusedId: 2, Description: "x",
	}); err != nil {
		t.Fatalf("CreateFromCommand: %v", err)
	}
	reports, _ := p.GetByTenant()
	if len(reports) != 1 {
		t.Fatal("expected persisted report despite transcript failure")
	}
	if reports[0].ServerTranscript() != nil {
		t.Error("expected nil transcript")
	}
}

func TestUpdateStatusValidationAndNotFound(t *testing.T) {
	db := setupTestDatabase(t)
	tm := sampleTenant()
	l, _ := test.NewNullLogger()
	charP := &fakeCharacterProcessor{
		byId: map[uint32]character.Model{
			1: makeCharacter(t, 1, "Reporter"),
			2: makeCharacter(t, 2, "Accused"),
		},
	}
	p := NewProcessorWithClients(l, testContext(tm), db, charP, &fakeChatProcessor{})

	buf := message.NewBuffer()
	_ = p.CreateFromCommand(buf)(report2.CreateCommandBody{Kind: report2.KindSue, ReporterId: 1, AccusedId: 2, Description: "x"})
	reports, _ := p.GetByTenant()

	m, err := p.UpdateStatus(reports[0].Id(), StatusActioned)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if m.Status() != StatusActioned {
		t.Errorf("status: got %s", m.Status())
	}

	if _, err = p.UpdateStatus(reports[0].Id(), Status("bogus")); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
	if _, err = p.UpdateStatus(uuid.New(), StatusReviewed); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}
```

NOTE for the implementer: `makeCharacter` constructs `character.Model` through the exported `RestModel`/`Extract` API (RestModel.Id is an exported field). Verify `message.NewBuffer()`/`buf.GetAll()` names against `BAN/kafka/message/message.go` before use and adjust to the actual buffer API (mirror how `ban` domain tests inspect buffered messages, if any do).

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-ban/atlas.com/ban && go test ./report/... 2>&1 | head -5`
Expected: FAIL to build — `undefined: NewProcessorWithClients`.

- [ ] **Step 3: Write producer and processor**

`services/atlas-ban/atlas.com/ban/report/producer.go`:

```go
package report

import (
	report2 "atlas-ban/kafka/message/report"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kafkago "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func statusEventProvider(reportId uuid.UUID, kind Kind, worldId world.Id, reporterId uint32, status string, errorCode string) model.Provider[[]kafka.Message] {
	key := kafkago.CreateKey(int(reporterId))
	value := &report2.StatusEvent{
		ReportId:   reportId,
		Kind:       string(kind),
		WorldId:    worldId,
		ReporterId: reporterId,
		Status:     status,
		ErrorCode:  errorCode,
	}
	return kafkago.SingleMessageProvider(key, value)
}
```

`services/atlas-ban/atlas.com/ban/report/processor.go`:

```go
package report

import (
	"atlas-ban/character"
	"atlas-ban/chat"
	"atlas-ban/kafka/message"
	report2 "atlas-ban/kafka/message/report"
	"atlas-ban/kafka/producer"
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	CreateFromCommand(buf *message.Buffer) func(c report2.CreateCommandBody) error
	CreateFromCommandAndEmit(c report2.CreateCommandBody) error
	UpdateStatus(reportId uuid.UUID, status Status) (Model, error)
	GetById(reportId uuid.UUID) (Model, error)
	ByIdProvider(reportId uuid.UUID) model.Provider[Model]
	GetByTenant() ([]Model, error)
	GetByStatus(status Status) ([]Model, error)
}

type ProcessorImpl struct {
	l     logrus.FieldLogger
	ctx   context.Context
	db    *gorm.DB
	t     tenant.Model
	p     producer.Provider
	charP character.Processor
	chatP chat.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return NewProcessorWithClients(l, ctx, db, character.NewProcessor(l, ctx), chat.NewProcessor(l, ctx))
}

// NewProcessorWithClients constructs a Processor with explicit REST client
// implementations. Production callers use NewProcessor; callers that already
// hold client instances (or substitutes) inject them here.
func NewProcessorWithClients(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, charP character.Processor, chatP chat.Processor) Processor {
	return &ProcessorImpl{
		l:     l,
		ctx:   ctx,
		db:    db,
		t:     tenant.MustFromContext(ctx),
		p:     producer.ProviderImpl(l)(ctx),
		charP: charP,
		chatP: chatP,
	}
}

func (p *ProcessorImpl) CreateFromCommandAndEmit(c report2.CreateCommandBody) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.CreateFromCommand(buf)(c)
	})
}

// CreateFromCommand resolves the accused, snapshots the chat transcript,
// persists the report, and buffers exactly one status event (CREATED or
// ERROR). Business rejections (unresolvable accused, DB failure) return nil
// so the ERROR event still emits — the reporter must always get a result
// packet.
func (p *ProcessorImpl) CreateFromCommand(buf *message.Buffer) func(c report2.CreateCommandBody) error {
	return func(c report2.CreateCommandBody) error {
		fail := func(code string) error {
			return buf.Put(report2.EnvEventTopicStatus, statusEventProvider(uuid.Nil, Kind(c.Kind), c.WorldId, c.ReporterId, report2.EventStatusError, code))
		}

		reporter, err := p.charP.GetById(c.ReporterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to resolve reporter [%d] for report.", c.ReporterId)
			return fail(report2.ErrorCodeInternal)
		}

		var accused character.Model
		switch {
		case c.AccusedName != "":
			accused, err = p.charP.GetByName(c.AccusedName)
		case c.AccusedId != 0:
			accused, err = p.charP.GetById(c.AccusedId)
		default:
			err = model.ErrNoResultFound
		}
		if err != nil {
			if errors.Is(err, requests.ErrNotFound) || errors.Is(err, model.ErrNoResultFound) {
				p.l.Infof("Rejecting [%s] report from [%d]: accused [%s/%d] not found in tenant.", c.Kind, c.ReporterId, c.AccusedName, c.AccusedId)
				return fail(report2.ErrorCodeNotFound)
			}
			p.l.WithError(err).Errorf("Unable to resolve accused [%s/%d] for report from [%d].", c.AccusedName, c.AccusedId, c.ReporterId)
			return fail(report2.ErrorCodeInternal)
		}

		description := c.Description
		if len(description) > MaxDescriptionLength {
			p.l.Warnf("Truncating report description from [%d] to [%d] chars for reporter [%d].", len(description), MaxDescriptionLength, c.ReporterId)
			description = description[:MaxDescriptionLength]
		}
		var chatLog *string
		if c.ChatClaim {
			cl := c.ChatLog
			if len(cl) > MaxChatLogBytes {
				p.l.Warnf("Truncating report chat log from [%d] to [%d] bytes for reporter [%d].", len(cl), MaxChatLogBytes, c.ReporterId)
				cl = cl[:MaxChatLogBytes]
			}
			chatLog = &cl
		}

		// Transcript is corroboration, best-effort by design: a messages
		// outage degrades to a null transcript, never a failed report.
		var transcript []TranscriptLine
		lines, terr := p.chatP.RecentInvolving([]uint32{c.ReporterId, accused.Id()})
		if terr != nil {
			p.l.WithError(terr).Warnf("Unable to fetch chat transcript for report from [%d]; persisting without.", c.ReporterId)
		} else {
			transcript = make([]TranscriptLine, 0, len(lines))
			for _, line := range lines {
				transcript = append(transcript, TranscriptLine{
					Timestamp:  line.Timestamp(),
					SenderId:   line.SenderId(),
					SenderName: line.SenderName(),
					ChatType:   line.ChatType(),
					Text:       line.Text(),
				})
			}
		}

		m, err := create(p.db.WithContext(p.ctx))(p.t.Id(), Kind(c.Kind), c.ReporterId, reporter.Name(), accused.Id(), accused.Name(), c.ReasonType, description, chatLog, transcript)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to persist [%s] report from [%d].", c.Kind, c.ReporterId)
			return fail(report2.ErrorCodeInternal)
		}
		p.l.Infof("Created [%s] report [%s]: reporter [%d/%s] accused [%d/%s] reason [%d].", m.Kind(), m.Id(), m.ReporterId(), m.ReporterName(), m.AccusedId(), m.AccusedName(), m.ReasonType())
		return buf.Put(report2.EnvEventTopicStatus, statusEventProvider(m.Id(), m.Kind(), c.WorldId, c.ReporterId, report2.EventStatusCreated, ""))
	}
}

func (p *ProcessorImpl) UpdateStatus(reportId uuid.UUID, status Status) (Model, error) {
	if !status.Valid() {
		return Model{}, ErrInvalidStatus
	}
	err := updateStatus(p.db.WithContext(p.ctx))(reportId, status)
	if err != nil {
		return Model{}, err
	}
	m, err := p.GetById(reportId)
	if err != nil {
		return Model{}, err
	}
	p.l.Infof("Report [%s] status updated to [%s].", reportId, status)
	return m, nil
}

func (p *ProcessorImpl) GetById(reportId uuid.UUID) (Model, error) {
	return p.ByIdProvider(reportId)()
}

func (p *ProcessorImpl) ByIdProvider(reportId uuid.UUID) model.Provider[Model] {
	return model.Map(Make)(entityById(reportId)(p.db.WithContext(p.ctx)))
}

func (p *ProcessorImpl) GetByTenant() ([]Model, error) {
	return model.SliceMap(Make)(entitiesByTenant()(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetByStatus(status Status) ([]Model, error) {
	return model.SliceMap(Make)(entitiesByStatus(status)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-ban/atlas.com/ban && go test -race ./report/... -v 2>&1 | tail -20`
Expected: PASS (all 7 processor + 4 administrator + 4 builder tests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ban/atlas.com/ban/report/
git commit -m "feat(ban): report processor with accused resolution, caps, transcript snapshot"
```

---

### Task 8: atlas-ban — REPORT command consumer + main wiring

**Files:**
- Create: `services/atlas-ban/atlas.com/ban/kafka/consumer/report/consumer.go`
- Create: `services/atlas-ban/atlas.com/ban/kafka/consumer/report/consumer_test.go`
- Modify: `services/atlas-ban/atlas.com/ban/main.go` (consumer registration, after the `account2` block at lines 64-67)

**Interfaces:**
- Consumes: Task 7 `report.NewProcessor(l, ctx, db).CreateFromCommandAndEmit(c.Body)`; `consumer2 "atlas-ban/kafka/consumer"` NewConfig; contract from Task 4.
- Produces: `report.InitConsumers(l)(cmf)(consumerGroupId)` and `report.InitHandlers(l)(db)(rf) error` (exact `ban2` consumer shape from `kafka/consumer/ban/consumer.go`).

- [ ] **Step 1: Write the failing handler test**

`services/atlas-ban/atlas.com/ban/kafka/consumer/report/consumer_test.go` — validate the type-guard: a command with the wrong `Type` must be ignored (no processor invocation, hence no rows):

```go
package report

import (
	"testing"

	report3 "atlas-ban/report"
	report2 "atlas-ban/kafka/message/report"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"context"
)

func setupDb(t *testing.T) *gorm.DB {
	l, _ := test.NewNullLogger()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err := db.AutoMigrate(&report3.Entity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestHandleCreateCommandIgnoresOtherTypes(t *testing.T) {
	db := setupDb(t)
	l, _ := test.NewNullLogger()
	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tm)

	h := handleCreateReportCommand(db)
	h(l, ctx, report2.Command[report2.CreateCommandBody]{
		Type: "DELETE",
		Body: report2.CreateCommandBody{Kind: report2.KindSue, ReporterId: 1, AccusedId: 2},
	})

	var count int64
	db.WithContext(ctx).Model(&report3.Entity{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected no rows for ignored command type, got %d", count)
	}
}
```

(The happy path exercises Kafka producer plumbing inside `CreateFromCommandAndEmit`; it is covered by Task 7's buffer-level tests plus live acceptance. The consumer test pins the type-guard contract only.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ban/atlas.com/ban && go test ./kafka/consumer/report/... 2>&1 | head -5`
Expected: FAIL to build — package does not exist.

- [ ] **Step 3: Write the consumer**

`services/atlas-ban/atlas.com/ban/kafka/consumer/report/consumer.go`:

```go
package report

import (
	consumer2 "atlas-ban/kafka/consumer"
	report2 "atlas-ban/kafka/message/report"
	report3 "atlas-ban/report"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("report_command")(report2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(report2.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateReportCommand(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleCreateReportCommand(db *gorm.DB) message.Handler[report2.Command[report2.CreateCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c report2.Command[report2.CreateCommandBody]) {
		if c.Type != report2.CommandTypeCreate {
			return
		}
		l.Debugf("Received create report command kind [%s] reporter [%d].", c.Body.Kind, c.Body.ReporterId)
		if err := report3.NewProcessor(l, ctx, db).CreateFromCommandAndEmit(c.Body); err != nil {
			l.WithError(err).Errorf("Error processing create report command from reporter [%d].", c.Body.ReporterId)
		}
	}
}
```

- [ ] **Step 4: Wire main.go**

In `services/atlas-ban/atlas.com/ban/main.go`, add import `report4 "atlas-ban/kafka/consumer/report"` and, after the `account2` handler block (line 67):

```go
	report4.InitConsumers(l)(cmf)(consumerGroupId)
	if err := report4.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-ban/atlas.com/ban && go test ./kafka/... -v 2>&1 | tail -6 && go build ./...`
Expected: PASS; build clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ban/atlas.com/ban/kafka/consumer/report/ services/atlas-ban/atlas.com/ban/main.go
git commit -m "feat(ban): REPORT command consumer"
```

---

### Task 9: atlas-ban — report REST resource, routes, mock, README

**Files:**
- Create: `services/atlas-ban/atlas.com/ban/report/resource.go`
- Create: `services/atlas-ban/atlas.com/ban/report/rest.go`
- Create: `services/atlas-ban/atlas.com/ban/report/rest_test.go`
- Create: `services/atlas-ban/atlas.com/ban/report/mock/processor.go`
- Modify: `services/atlas-ban/atlas.com/ban/rest/handler.go` (add `ParseReportId`)
- Modify: `services/atlas-ban/atlas.com/ban/main.go` (line 76-77 area: add `AddRouteInitializer(report.InitResource(GetServer())(db))`)
- Modify: `services/atlas-ban/docs/README.md` if present, else the service README located via `Glob services/atlas-ban/**/README.md` (REST endpoints + Kafka tables)

**Interfaces:**
- Consumes: Task 7 Processor; `rest.RegisterHandler` / `rest.RegisterInputHandler` from `BAN/rest/handler.go`.
- Produces: JSON:API resource type `reports` at `GET /api/reports` (`?status=` filter), `GET /api/reports/{reportId}`, `PATCH /api/reports/{reportId}`; `rest.ParseReportId(l, next func(reportId uuid.UUID) http.HandlerFunc) http.HandlerFunc`; `mock.ProcessorMock` with `XxxFunc` fields for every Processor method.

- [ ] **Step 1: Write the failing RestModel test**

`services/atlas-ban/atlas.com/ban/report/rest_test.go`:

```go
package report

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTransformRoundTrip(t *testing.T) {
	chatLog := "log"
	m, err := NewBuilder(uuid.New(), KindClaim, 1).
		SetId(uuid.New()).
		SetReporterName("R").
		SetAccusedId(2).
		SetAccusedName("A").
		SetReasonType(3).
		SetDescription("d").
		SetChatLog(&chatLog).
		SetServerTranscript([]TranscriptLine{{Timestamp: 5, SenderId: 1, SenderName: "R", ChatType: "GENERAL", Text: "hi"}}).
		SetStatus(StatusReviewed).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if rm.GetName() != "reports" {
		t.Errorf("resource name: %s", rm.GetName())
	}
	if rm.GetID() != m.Id().String() {
		t.Errorf("id: %s", rm.GetID())
	}
	if rm.Kind != "claim" || rm.Status != "reviewed" || rm.ChatLog == nil || len(rm.ServerTranscript) != 1 {
		t.Errorf("attributes mismatch: %+v", rm)
	}
}

func TestRestModelSetIDRejectsGarbage(t *testing.T) {
	rm := &RestModel{}
	if err := rm.SetID("not-a-uuid"); err == nil {
		t.Fatal("expected error")
	}
	id := uuid.New()
	if err := rm.SetID(id.String()); err != nil || rm.Id != id {
		t.Fatalf("SetID: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ban/atlas.com/ban && go test ./report/... 2>&1 | head -5`
Expected: FAIL to build — `undefined: Transform`.

- [ ] **Step 3: Write resource.go**

`services/atlas-ban/atlas.com/ban/report/resource.go`:

```go
package report

import (
	"time"

	"github.com/google/uuid"
)

type RestModel struct {
	Id               uuid.UUID        `json:"-"`
	Kind             string           `json:"kind"`
	ReporterId       uint32           `json:"reporterId"`
	ReporterName     string           `json:"reporterName"`
	AccusedId        uint32           `json:"accusedId"`
	AccusedName      string           `json:"accusedName"`
	ReasonType       byte             `json:"reasonType"`
	Description      string           `json:"description"`
	ChatLog          *string          `json:"chatLog"`
	ServerTranscript []TranscriptLine `json:"serverTranscript"`
	Status           string           `json:"status"`
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}

func (r RestModel) GetName() string {
	return "reports"
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

func (r *RestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:               m.Id(),
		Kind:             string(m.Kind()),
		ReporterId:       m.ReporterId(),
		ReporterName:     m.ReporterName(),
		AccusedId:        m.AccusedId(),
		AccusedName:      m.AccusedName(),
		ReasonType:       m.ReasonType(),
		Description:      m.Description(),
		ChatLog:          m.ChatLog(),
		ServerTranscript: m.ServerTranscript(),
		Status:           string(m.Status()),
		CreatedAt:        m.CreatedAt(),
		UpdatedAt:        m.UpdatedAt(),
	}, nil
}
```

- [ ] **Step 4: Write rest.go and ParseReportId**

Append to `services/atlas-ban/atlas.com/ban/rest/handler.go`:

```go
type ReportIdHandler func(id uuid.UUID) http.HandlerFunc

func ParseReportId(l logrus.FieldLogger, next ReportIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.Parse(vars["reportId"])
		if err != nil {
			l.WithError(err).Errorln("Error parsing reportId as uuid")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(id)(w, r)
	}
}
```

(add `"github.com/google/uuid"` to that file's imports.)

`services/atlas-ban/atlas.com/ban/report/rest.go`:

```go
package report

import (
	"atlas-ban/rest"
	"errors"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			register := rest.RegisterHandler(l)(db)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(db)(si)

			r := router.PathPrefix("/reports").Subrouter()
			r.HandleFunc("/", register("get_reports", handleGetReports)).Methods(http.MethodGet)
			r.HandleFunc("/{reportId}", register("get_report", handleGetReportById)).Methods(http.MethodGet)
			r.HandleFunc("/{reportId}", registerInput("update_report_status", handleUpdateReportStatus)).Methods(http.MethodPatch)
		}
	}
}

func handleGetReports(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statusStr := r.URL.Query().Get("status")

		var reports []Model
		var err error
		if statusStr != "" {
			s := Status(statusStr)
			if !s.Valid() {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			reports, err = NewProcessor(d.Logger(), d.Context(), d.DB()).GetByStatus(s)
		} else {
			reports, err = NewProcessor(d.Logger(), d.Context(), d.DB()).GetByTenant()
		}
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to locate reports.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(reports))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}

func handleGetReportById(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseReportId(d.Logger(), func(reportId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetById(reportId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve report [%s].", reportId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := Transform(m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleUpdateReportStatus(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return rest.ParseReportId(d.Logger(), func(reportId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if input.Id != uuid.Nil && input.Id != reportId {
				d.Logger().Errorln("Report ID does not match URL")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).UpdateStatus(reportId, Status(input.Status))
			if err != nil {
				if errors.Is(err, ErrInvalidStatus) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				d.Logger().WithError(err).Errorf("Unable to update report [%s].", reportId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := Transform(m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}
```

- [ ] **Step 5: Write the mock**

`services/atlas-ban/atlas.com/ban/report/mock/processor.go`:

```go
package mock

import (
	"atlas-ban/kafka/message"
	report2 "atlas-ban/kafka/message/report"
	"atlas-ban/report"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	CreateFromCommandFunc        func(buf *message.Buffer) func(c report2.CreateCommandBody) error
	CreateFromCommandAndEmitFunc func(c report2.CreateCommandBody) error
	UpdateStatusFunc             func(reportId uuid.UUID, status report.Status) (report.Model, error)
	GetByIdFunc                  func(reportId uuid.UUID) (report.Model, error)
	ByIdProviderFunc             func(reportId uuid.UUID) model.Provider[report.Model]
	GetByTenantFunc              func() ([]report.Model, error)
	GetByStatusFunc              func(status report.Status) ([]report.Model, error)
}

func (m *ProcessorMock) CreateFromCommand(buf *message.Buffer) func(c report2.CreateCommandBody) error {
	if m.CreateFromCommandFunc != nil {
		return m.CreateFromCommandFunc(buf)
	}
	return func(report2.CreateCommandBody) error { return nil }
}

func (m *ProcessorMock) CreateFromCommandAndEmit(c report2.CreateCommandBody) error {
	if m.CreateFromCommandAndEmitFunc != nil {
		return m.CreateFromCommandAndEmitFunc(c)
	}
	return nil
}

func (m *ProcessorMock) UpdateStatus(reportId uuid.UUID, status report.Status) (report.Model, error) {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(reportId, status)
	}
	return report.Model{}, nil
}

func (m *ProcessorMock) GetById(reportId uuid.UUID) (report.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(reportId)
	}
	return report.Model{}, nil
}

func (m *ProcessorMock) ByIdProvider(reportId uuid.UUID) model.Provider[report.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(reportId)
	}
	return model.FixedProvider(report.Model{})
}

func (m *ProcessorMock) GetByTenant() ([]report.Model, error) {
	if m.GetByTenantFunc != nil {
		return m.GetByTenantFunc()
	}
	return nil, nil
}

func (m *ProcessorMock) GetByStatus(status report.Status) ([]report.Model, error) {
	if m.GetByStatusFunc != nil {
		return m.GetByStatusFunc(status)
	}
	return nil, nil
}
```

- [ ] **Step 6: Wire main.go route and update README**

In `services/atlas-ban/atlas.com/ban/main.go` add after the `history.InitResource` line:

```go
		AddRouteInitializer(report.InitResource(GetServer())(db)).
```

Locate the service README with `Glob services/atlas-ban/**/*.md` and add: the three `/api/reports` endpoints (methods, query params, status codes 400/404), the `COMMAND_TOPIC_REPORT` consumer and `EVENT_TOPIC_REPORT_STATUS` producer rows, and the new `CHARACTERS_SERVICE_URL` / `MESSAGES_SERVICE_URL` env vars (optional, `BASE_SERVICE_URL` fallback).

- [ ] **Step 7: Run tests + gates, verify mock satisfies the interface**

Run: `cd services/atlas-ban/atlas.com/ban && go test -race ./... && go vet ./... && go build ./...`
Expected: all clean. Add this compile-time assertion at the top of `mock/processor.go` if not already present: `var _ report.Processor = (*ProcessorMock)(nil)`.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-ban/
git commit -m "feat(ban): report REST resource (list/detail/status PATCH), mock, docs"
```

---

### Task 10: libs/atlas-redis — bounded append on TenantKeyedSortedSet

**Files:**
- Modify: `libs/atlas-redis/keyed_sorted_set.go`
- Modify: `libs/atlas-redis/keyed_sorted_set_test.go`

**Interfaces:**
- Consumes: existing `TenantKeyedSortedSet[K]`, miniredis test scaffolding already in the lib (`setupTestRedis(t)`, `makeTenant(...)`, `keyPrefix`/`computeKeyPrefix` reset pattern, `TenantKey`).
- Produces: `func (s *TenantKeyedSortedSet[K]) AddBounded(ctx context.Context, t tenant.Model, k K, member string, score float64, minScore float64, maxCount int64, ttl time.Duration) error` — one pipelined ZADD + ZREMRANGEBYSCORE (drop scores < minScore) + ZREMRANGEBYRANK (keep newest maxCount) + EXPIRE (key TTL so idle buffers evaporate without a sweeper).

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-redis/keyed_sorted_set_test.go`:

```go
func TestTenantKeyedSortedSet_AddBounded(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, mr := setupTestRedis(t)
	ctx := context.Background()

	s := NewTenantKeyedSortedSet[uint32](client, "chat:recent", func(id uint32) string {
		return strconv.FormatUint(uint64(id), 10)
	})
	tm := makeTenant("00000000-0000-0000-0000-000000000002", "GMS", 83, 1)
	const characterId uint32 = 7
	ttl := 900 * time.Second

	// Age-based pruning: a member whose score falls below minScore is dropped.
	if err := s.AddBounded(ctx, tm, characterId, "old", 1000, 0, 10, ttl); err != nil {
		t.Fatalf("AddBounded old: %v", err)
	}
	if err := s.AddBounded(ctx, tm, characterId, "new", 2000, 1500, 10, ttl); err != nil {
		t.Fatalf("AddBounded new: %v", err)
	}
	members, err := s.Range(ctx, tm, characterId)
	if err != nil {
		t.Fatalf("Range: %v", err)
	}
	if len(members) != 1 || members[0] != "new" {
		t.Fatalf("age prune: got %v want [new]", members)
	}

	// Count-based trimming: only the newest maxCount members survive.
	for i := 0; i < 5; i++ {
		member := "m" + strconv.Itoa(i)
		if err := s.AddBounded(ctx, tm, characterId, member, float64(3000+i), 0, 3, ttl); err != nil {
			t.Fatalf("AddBounded %s: %v", member, err)
		}
	}
	members, _ = s.Range(ctx, tm, characterId)
	if len(members) != 3 {
		t.Fatalf("count trim: got %d members (%v), want 3", len(members), members)
	}
	if members[0] != "m2" || members[2] != "m4" {
		t.Fatalf("count trim kept wrong members: %v", members)
	}

	// TTL refresh: the key expires after the window.
	wantKey := "atlas:chat:recent:" + TenantKey(tm) + ":7"
	if !mr.Exists(wantKey) {
		t.Fatalf("expected key %q; keys=%v", wantKey, mr.Keys())
	}
	mr.FastForward(ttl + time.Second)
	if mr.Exists(wantKey) {
		t.Fatal("expected key to expire after ttl")
	}
}
```

(add `"time"` to the test file imports if missing.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-redis && go test -run AddBounded ./... 2>&1 | head -5`
Expected: FAIL to build — `s.AddBounded undefined`.

- [ ] **Step 3: Implement AddBounded**

Append to `libs/atlas-redis/keyed_sorted_set.go` (add `"strconv"` and `"time"` imports):

```go
// AddBounded inserts member with the given score and enforces the buffer
// bounds in the same pipeline: members with score < minScore are pruned
// (age window), the set is trimmed to the newest maxCount members, and the
// key TTL is refreshed to ttl so idle keys evaporate without a sweeper.
// maxCount <= 0 skips the count trim; ttl <= 0 skips the TTL refresh.
func (s *TenantKeyedSortedSet[K]) AddBounded(ctx context.Context, t tenant.Model, k K, member string, score float64, minScore float64, maxCount int64, ttl time.Duration) error {
	key := s.key(t, k)
	pipe := s.client.TxPipeline()
	pipe.ZAdd(ctx, key, goredis.Z{Score: score, Member: member})
	pipe.ZRemRangeByScore(ctx, key, "-inf", "("+strconv.FormatFloat(minScore, 'f', -1, 64))
	if maxCount > 0 {
		pipe.ZRemRangeByRank(ctx, key, 0, -(maxCount + 1))
	}
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-redis && go test -race ./... 2>&1 | tail -3 && go vet ./...`
Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-redis/keyed_sorted_set.go libs/atlas-redis/keyed_sorted_set_test.go
git commit -m "feat(redis): AddBounded pipelined bounded append on TenantKeyedSortedSet"
```

---

### Task 11: atlas-messages — chat capture buffer + capture wiring

**Files:**
- Create: `services/atlas-messages/atlas.com/messages/chat/model.go`
- Create: `services/atlas-messages/atlas.com/messages/chat/config.go`
- Create: `services/atlas-messages/atlas.com/messages/chat/registry.go`
- Create: `services/atlas-messages/atlas.com/messages/chat/processor.go`
- Create: `services/atlas-messages/atlas.com/messages/chat/processor_test.go`
- Modify: `services/atlas-messages/atlas.com/messages/message/processor.go` (capture calls in HandleGeneral/HandleMulti/HandleWhisper/HandleMessenger)
- Modify: `services/atlas-messages/atlas.com/messages/main.go` (Redis connect + registry init)
- Modify: `services/atlas-messages/atlas.com/messages/go.mod` via `go mod tidy` (new deps: atlas-redis, go-redis, miniredis test dep)

**Interfaces:**
- Consumes: Task 10 `AddBounded`; `atlas "github.com/Chronicle20/atlas/libs/atlas-redis"`; existing chat-type constants in `MSG/kafka/message/message/kafka.go` (`ChatTypeGeneral`, `ChatTypeWhisper`, `ChatTypeMessenger`; HandleMulti already receives the multi chatType string).
- Produces: `chat.Line` (json-tagged struct: `Timestamp int64 "ts"`, `SenderId uint32 "senderId"`, `SenderName string "senderName"`, `ChatType string "type"`, `Text string "text"`, `WorldId byte "worldId"`, `ChannelId byte "channelId"`, `MapId uint32 "mapId"`); `chat.InitRegistry(client *goredis.Client)` / `chat.GetRegistry() *Registry`; `Registry.Append(ctx, t, line Line) error`; `Registry.RecentBySender(ctx, t, characterId uint32) ([]Line, error)`; `chat.NewProcessor(l, ctx) Processor` with `Capture(f field.Model, senderId uint32, senderName string, chatType string, text string) error` and `RecentInvolving(characterIds []uint32) ([]Line, error)` (merged, sorted ascending by Timestamp). Env: `CHAT_CAPTURE_RETENTION_SECONDS` (default 900), `CHAT_CAPTURE_MAX_LINES` (default 200).

- [ ] **Step 1: Write the failing tests**

`services/atlas-messages/atlas.com/messages/chat/processor_test.go` (miniredis; add `github.com/alicebob/miniredis/v2` as a test dep):

```go
package chat

import (
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus/hooks/test"
)

func setupRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func testTenantContext() context.Context {
	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), tm)
}

func testField() field.Model {
	return field.NewBuilder(0, 1, 100000000).Build()
}

func TestCaptureAndRecentInvolving(t *testing.T) {
	setupRegistry(t)
	l, _ := test.NewNullLogger()
	ctx := testTenantContext()
	p := NewProcessor(l, ctx)

	if err := p.Capture(testField(), 1, "Alice", "GENERAL", "first"); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	time.Sleep(2 * time.Millisecond) // distinct unix-milli timestamps
	if err := p.Capture(testField(), 2, "Bob", "WHISPER", "second"); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	if err := p.Capture(testField(), 3, "Carol", "GENERAL", "uninvolved"); err != nil {
		t.Fatalf("Capture: %v", err)
	}

	lines, err := p.RecentInvolving([]uint32{1, 2})
	if err != nil {
		t.Fatalf("RecentInvolving: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %+v", len(lines), lines)
	}
	if lines[0].Text != "first" || lines[1].Text != "second" {
		t.Errorf("expected timestamp-ascending merge, got %+v", lines)
	}
	if lines[0].SenderName != "Alice" || lines[0].ChatType != "GENERAL" || lines[0].MapId != 100000000 {
		t.Errorf("line fields mismatch: %+v", lines[0])
	}
}

func TestRecentInvolvingEmptyBuffer(t *testing.T) {
	setupRegistry(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, testTenantContext())

	lines, err := p.RecentInvolving([]uint32{99})
	if err != nil {
		t.Fatalf("RecentInvolving: %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected empty, got %+v", lines)
	}
}

func TestConfigDefaults(t *testing.T) {
	if got := envInt("CHAT_CAPTURE_TEST_UNSET_VAR", 900); got != 900 {
		t.Errorf("default: got %d", got)
	}
	t.Setenv("CHAT_CAPTURE_TEST_SET_VAR", "42")
	if got := envInt("CHAT_CAPTURE_TEST_SET_VAR", 900); got != 42 {
		t.Errorf("env override: got %d", got)
	}
	t.Setenv("CHAT_CAPTURE_TEST_BAD_VAR", "notanint")
	if got := envInt("CHAT_CAPTURE_TEST_BAD_VAR", 900); got != 900 {
		t.Errorf("bad value falls back to default: got %d", got)
	}
}
```

NOTE for the implementer: verify `field.NewBuilder(worldId, channelId, mapId).Build()` argument types against `libs/atlas-constants/field` before use (world.Id/channel.Id/_map.Id) and cast literals accordingly.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-messages/atlas.com/messages && go test ./chat/... 2>&1 | head -5`
Expected: FAIL to build — package does not exist.

- [ ] **Step 3: Implement the chat package**

`services/atlas-messages/atlas.com/messages/chat/model.go`:

```go
package chat

// Line is one captured chat line in the bounded per-character buffer.
// Stored as JSON in a timestamp-scored Redis sorted set; short-retention
// working state, not an archive.
type Line struct {
	Timestamp  int64  `json:"ts"` // unix-milli, stamped at capture (the wire carries no timestamp)
	SenderId   uint32 `json:"senderId"`
	SenderName string `json:"senderName"`
	ChatType   string `json:"type"`
	Text       string `json:"text"`
	WorldId    byte   `json:"worldId"`
	ChannelId  byte   `json:"channelId"`
	MapId      uint32 `json:"mapId"`
}
```

`services/atlas-messages/atlas.com/messages/chat/config.go`:

```go
package chat

import (
	"os"
	"strconv"
	"sync"
)

const (
	envRetentionSeconds = "CHAT_CAPTURE_RETENTION_SECONDS"
	envMaxLines         = "CHAT_CAPTURE_MAX_LINES"

	defaultRetentionSeconds = 900
	defaultMaxLines         = 200
)

var (
	configOnce       sync.Once
	retentionSeconds int
	maxLines         int
)

func envInt(name string, def int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return def
	}
	return v
}

func loadConfig() {
	configOnce.Do(func() {
		retentionSeconds = envInt(envRetentionSeconds, defaultRetentionSeconds)
		maxLines = envInt(envMaxLines, defaultMaxLines)
	})
}
```

`services/atlas-messages/atlas.com/messages/chat/registry.go`:

```go
package chat

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	lines *atlas.TenantKeyedSortedSet[uint32]
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		lines: atlas.NewTenantKeyedSortedSet[uint32](client, "chat:recent", func(characterId uint32) string {
			return strconv.FormatUint(uint64(characterId), 10)
		}),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) Append(ctx context.Context, t tenant.Model, line Line) error {
	loadConfig()
	member, err := json.Marshal(line)
	if err != nil {
		return err
	}
	score := float64(line.Timestamp)
	minScore := score - float64(retentionSeconds)*1000
	ttl := time.Duration(retentionSeconds) * time.Second
	return r.lines.AddBounded(ctx, t, line.SenderId, string(member), score, minScore, int64(maxLines), ttl)
}

func (r *Registry) RecentBySender(ctx context.Context, t tenant.Model, characterId uint32) ([]Line, error) {
	members, err := r.lines.Range(ctx, t, characterId)
	if err != nil {
		return nil, err
	}
	result := make([]Line, 0, len(members))
	for _, m := range members {
		var line Line
		if err := json.Unmarshal([]byte(m), &line); err != nil {
			continue
		}
		result = append(result, line)
	}
	return result, nil
}
```

`services/atlas-messages/atlas.com/messages/chat/processor.go`:

```go
package chat

import (
	"context"
	"sort"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	// Capture appends one player-authored chat line to the sender's bounded
	// buffer. Failures are the caller's to log; capture must never block or
	// fail the chat flow.
	Capture(f field.Model, senderId uint32, senderName string, chatType string, text string) error
	// RecentInvolving returns lines authored by any of the listed characters,
	// merged and sorted ascending by timestamp.
	RecentInvolving(characterIds []uint32) ([]Line, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, t: tenant.MustFromContext(ctx)}
}

func (p *ProcessorImpl) Capture(f field.Model, senderId uint32, senderName string, chatType string, text string) error {
	r := GetRegistry()
	if r == nil {
		return nil
	}
	return r.Append(p.ctx, p.t, Line{
		Timestamp:  time.Now().UnixMilli(),
		SenderId:   senderId,
		SenderName: senderName,
		ChatType:   chatType,
		Text:       text,
		WorldId:    byte(f.WorldId()),
		ChannelId:  byte(f.ChannelId()),
		MapId:      uint32(f.MapId()),
	})
}

func (p *ProcessorImpl) RecentInvolving(characterIds []uint32) ([]Line, error) {
	r := GetRegistry()
	if r == nil {
		return nil, nil
	}
	all := make([]Line, 0)
	for _, id := range characterIds {
		lines, err := r.RecentBySender(p.ctx, p.t, id)
		if err != nil {
			return nil, err
		}
		all = append(all, lines...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Timestamp < all[j].Timestamp })
	return all, nil
}
```

- [ ] **Step 4: Wire capture into the message processor**

In `services/atlas-messages/atlas.com/messages/message/processor.go`:

Add import `"atlas-messages/chat"` and a helper at the bottom of the file:

```go
// captureLine records a player-authored chat line for report corroboration.
// Best-effort: a Redis outage logs a warning and never blocks the chat flow.
func (p *ProcessorImpl) captureLine(f field.Model, senderId uint32, senderName string, chatType string, text string) {
	if err := chat.NewProcessor(p.l, p.ctx).Capture(f, senderId, senderName, chatType, text); err != nil {
		p.l.WithError(err).Warnf("Unable to capture chat line for character [%d].", senderId)
	}
}
```

Insert calls AFTER each command-registry check (so slash commands are never captured), immediately before the chat-event emit:

- `HandleGeneral` (before the `producer.ProviderImpl(...)` emit at line 55): `p.captureLine(f, actorId, c.Name(), message2.ChatTypeGeneral, message)`
- `HandleMulti` (before the emit at line 78): `p.captureLine(f, actorId, c.Name(), chatType, message)`
- `HandleWhisper` (before the emit at line 111, after the recipient lookup): `p.captureLine(f, actorId, c.Name(), message2.ChatTypeWhisper, message)`
- `HandleMessenger` (before the emit at line 125): `p.captureLine(f, actorId, c.Name(), message2.ChatTypeMessenger, message)`

`HandlePet` and `IssuePinkText` get NO capture calls (PET echoes owner input under the pet's actor id; PINK_TEXT is system-issued).

Verify the chat-type constant names against `MSG/kafka/message/message/kafka.go:13-21` (`ChatTypeGeneral`, `ChatTypeWhisper`, `ChatTypeMessenger`) — use those constants, not literals.

- [ ] **Step 5: Wire Redis in main.go**

In `services/atlas-messages/atlas.com/messages/main.go` add imports `"atlas-messages/chat"` and `atlas "github.com/Chronicle20/atlas/libs/atlas-redis"`, then after the tracer init (line 41):

```go
	rc := atlas.Connect(l)
	chat.InitRegistry(rc)
```

`REDIS_URL` already flows from the shared `atlas-env` configmap (`deploy/k8s/base/env-configmap.yaml:151`); no manifest change needed for it.

- [ ] **Step 6: Tidy and run tests**

Run: `cd services/atlas-messages/atlas.com/messages && go mod tidy && go test -race ./chat/... ./message/... 2>&1 | tail -6 && go build ./...`
Expected: PASS; build clean. (`go mod tidy` AFTER the imports exist, never before — workspace footgun.)

- [ ] **Step 7: Commit**

```bash
git add services/atlas-messages/
git commit -m "feat(messages): bounded per-character chat capture buffer in Redis"
```

---

### Task 12: atlas-messages — chat history REST endpoint

**Files:**
- Create: `services/atlas-messages/atlas.com/messages/rest/handler.go`
- Create: `services/atlas-messages/atlas.com/messages/chat/resource.go`
- Create: `services/atlas-messages/atlas.com/messages/chat/resource_test.go`
- Modify: `services/atlas-messages/atlas.com/messages/main.go` (Server info struct + route initializer)
- Modify: service README (locate via `Glob services/atlas-messages/**/*.md`) and `docs/storage.md` (atlas-messages is no longer storage-free)

**Interfaces:**
- Consumes: Task 11 `chat.Processor.RecentInvolving`; `atlas-rest/server` helpers.
- Produces: `GET /api/chat/history?characterIds={a},{b}` returning JSON:API list of resource type `chat-messages` with attributes `{timestamp, senderId, senderName, chatType, text}` — the exact contract Task 6's ban-side `chat.RestModel` consumes. Server-to-server only: NO nginx/ingress entry. `rest.RegisterHandler` (db-less variant) + `rest.HandlerDependency{Logger(), Context()}`.

- [ ] **Step 1: Write the failing resource test**

`services/atlas-messages/atlas.com/messages/chat/resource_test.go`:

```go
package chat

import "testing"

func TestTransform(t *testing.T) {
	line := Line{Timestamp: 123, SenderId: 7, SenderName: "Alice", ChatType: "GENERAL", Text: "hi", WorldId: 0, ChannelId: 1, MapId: 100000000}
	rm := Transform(3, line)
	if rm.GetName() != "chat-messages" {
		t.Errorf("resource name: %s", rm.GetName())
	}
	if rm.GetID() != "3" {
		t.Errorf("id: %s", rm.GetID())
	}
	if rm.Timestamp != 123 || rm.SenderId != 7 || rm.SenderName != "Alice" || rm.ChatType != "GENERAL" || rm.Text != "hi" {
		t.Errorf("attributes mismatch: %+v", rm)
	}
}

func TestParseCharacterIds(t *testing.T) {
	ids, err := parseCharacterIds("1,42")
	if err != nil || len(ids) != 2 || ids[0] != 1 || ids[1] != 42 {
		t.Errorf("parse: %v %v", ids, err)
	}
	if _, err := parseCharacterIds(""); err == nil {
		t.Error("expected error for empty")
	}
	if _, err := parseCharacterIds("1,abc"); err == nil {
		t.Error("expected error for garbage")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-messages/atlas.com/messages && go test ./chat/... 2>&1 | head -5`
Expected: FAIL to build — `undefined: Transform`.

- [ ] **Step 3: Write the db-less rest helper**

`services/atlas-messages/atlas.com/messages/rest/handler.go` (ban's `rest/handler.go` with the db field removed and only the GET path kept):

```go
package rest

import (
	"context"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

type HandlerDependency struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func (h HandlerDependency) Logger() logrus.FieldLogger {
	return h.l
}

func (h HandlerDependency) Context() context.Context {
	return h.ctx
}

type HandlerContext struct {
	si jsonapi.ServerInformation
}

func (h HandlerContext) ServerInformation() jsonapi.ServerInformation {
	return h.si
}

type GetHandler func(d *HandlerDependency, c *HandlerContext) http.HandlerFunc

func RegisterHandler(l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
		return func(handlerName string, handler GetHandler) http.HandlerFunc {
			return server.RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return server.ParseTenant(fl, sctx, func(tl logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
					return handler(&HandlerDependency{l: tl, ctx: tctx}, &HandlerContext{si: si})
				})
			})
		}
	}
}
```

- [ ] **Step 4: Write resource.go**

`services/atlas-messages/atlas.com/messages/chat/resource.go`:

```go
package chat

import (
	"atlas-messages/rest"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

// RestModel is the "chat-messages" resource consumed server-to-server by
// atlas-ban's report pipeline. Not exposed through nginx/ingress.
type RestModel struct {
	Id         string `json:"-"`
	Timestamp  int64  `json:"timestamp"`
	SenderId   uint32 `json:"senderId"`
	SenderName string `json:"senderName"`
	ChatType   string `json:"chatType"`
	Text       string `json:"text"`
}

func (r RestModel) GetName() string {
	return "chat-messages"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Transform(index int, line Line) RestModel {
	return RestModel{
		Id:         strconv.Itoa(index),
		Timestamp:  line.Timestamp,
		SenderId:   line.SenderId,
		SenderName: line.SenderName,
		ChatType:   line.ChatType,
		Text:       line.Text,
	}
}

func parseCharacterIds(raw string) ([]uint32, error) {
	if raw == "" {
		return nil, errors.New("characterIds is required")
	}
	parts := strings.Split(raw, ",")
	ids := make([]uint32, 0, len(parts))
	for _, part := range parts {
		v, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32)
		if err != nil {
			return nil, err
		}
		ids = append(ids, uint32(v))
	}
	return ids, nil
}

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		register := rest.RegisterHandler(l)(si)
		r := router.PathPrefix("/chat").Subrouter()
		r.HandleFunc("/history", register("get_chat_history", handleGetChatHistory)).Methods(http.MethodGet)
	}
}

func handleGetChatHistory(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, err := parseCharacterIds(r.URL.Query().Get("characterIds"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		lines, err := NewProcessor(d.Logger(), d.Context()).RecentInvolving(ids)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to retrieve chat history.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res := make([]RestModel, 0, len(lines))
		for i, line := range lines {
			res = append(res, Transform(i, line))
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}
```

- [ ] **Step 5: Wire main.go**

In `services/atlas-messages/atlas.com/messages/main.go` add a server-info type (above `main`) and the route initializer:

```go
type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string {
	return s.baseUrl
}

func (s Server) GetPrefix() string {
	return s.prefix
}

func GetServer() Server {
	return Server{
		baseUrl: "",
		prefix:  "/api/",
	}
}
```

and change the server block (line 77-83) to include:

```go
		AddRouteInitializer(chat.InitResource(GetServer())).
```

- [ ] **Step 6: Docs**

- Service README: document the capture behavior (types captured/excluded, env tunables, retention semantics) and the `GET /api/chat/history` endpoint (server-to-server only).
- `docs/storage.md`: locate the atlas-messages row/entry via Grep and update it — the service now holds short-retention Redis working state (namespace `chat:recent`).

- [ ] **Step 7: Run tests + gates**

Run: `cd services/atlas-messages/atlas.com/messages && go test -race ./... 2>&1 | tail -5 && go vet ./... && go build ./...`
Expected: all clean.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-messages/ docs/storage.md
git commit -m "feat(messages): chat history REST endpoint for report corroboration"
```

---

### Task 13: atlas-channel — report Kafka contract + domain package

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/report/kafka.go`
- Create: `services/atlas-channel/atlas.com/channel/report/producer.go`
- Create: `services/atlas-channel/atlas.com/channel/report/producer_test.go`
- Create: `services/atlas-channel/atlas.com/channel/report/processor.go`

**Interfaces:**
- Consumes: nothing new — mirrors Task 4's contract byte-for-byte (same constant names both sides, like the chat topics); `"atlas-channel/kafka/producer"` `ProviderImpl`.
- Produces: `report.NewProcessor(l, ctx) Processor` with:

```go
type Processor interface {
	Sue(reporterId uint32, worldId world.Id, channelId channel.Id, accusedId uint32, subCommand string, flag byte, reason string) error
	Claim(reporterId uint32, worldId world.Id, channelId channel.Id, targetName string, reasonType byte, description string, chatClaim bool, chatLog string) error
}
```

Pure emit — no local state. Legacy sue passes `accusedId` + empty `subCommand`; v95 sue passes `accusedId=0` + `subCommand` (mapped to `AccusedName`); which field was populated is the wire→command mapping and lives here.

- [ ] **Step 1: Write the contract file**

`services/atlas-channel/atlas.com/channel/kafka/message/report/kafka.go`: identical content to Task 4's `services/atlas-ban/atlas.com/ban/kafka/message/report/kafka.go` (package `report`; copy the file verbatim — same consts `EnvCommandTopic`, `CommandTypeCreate`, `EnvEventTopicStatus`, `EventStatusCreated`, `EventStatusError`, `ErrorCodeNotFound`, `ErrorCodeInternal`, `KindSue`, `KindClaim`; same `Command[E]`, `CreateCommandBody`, `StatusEvent` structs).

- [ ] **Step 2: Write the failing producer test**

`services/atlas-channel/atlas.com/channel/report/producer_test.go`:

```go
package report

import (
	"encoding/json"
	"testing"

	report2 "atlas-channel/kafka/message/report"
)

func TestSueCommandProviderLegacy(t *testing.T) {
	msgs, err := sueCommandProvider(1, 0, 2, 12345, "", 0x05, "spamming")()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	var cmd report2.Command[report2.CreateCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != report2.CommandTypeCreate {
		t.Errorf("type: %s", cmd.Type)
	}
	b := cmd.Body
	if b.Kind != report2.KindSue || b.ReporterId != 1 || b.AccusedId != 12345 || b.AccusedName != "" ||
		b.ReasonType != 0x05 || b.Description != "spamming" || b.ChatClaim {
		t.Errorf("body mismatch: %+v", b)
	}
}

func TestSueCommandProviderV95SubCommand(t *testing.T) {
	msgs, _ := sueCommandProvider(1, 0, 2, 0, "alice", 0x05, "spamming")()
	var cmd report2.Command[report2.CreateCommandBody]
	_ = json.Unmarshal(msgs[0].Value, &cmd)
	if cmd.Body.AccusedId != 0 || cmd.Body.AccusedName != "alice" {
		t.Errorf("v95 mapping mismatch: %+v", cmd.Body)
	}
}

func TestClaimCommandProvider(t *testing.T) {
	msgs, err := claimCommandProvider(7, 0, 1, "bob", 0x03, "harassment", true, "bob: mean")()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	var cmd report2.Command[report2.CreateCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	b := cmd.Body
	if b.Kind != report2.KindClaim || b.ReporterId != 7 || b.AccusedName != "bob" ||
		b.ReasonType != 0x03 || b.Description != "harassment" || !b.ChatClaim || b.ChatLog != "bob: mean" {
		t.Errorf("body mismatch: %+v", b)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./report/... 2>&1 | head -5`
Expected: FAIL to build — package does not exist.

- [ ] **Step 4: Write producer and processor**

`services/atlas-channel/atlas.com/channel/report/producer.go`:

```go
package report

import (
	report2 "atlas-channel/kafka/message/report"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func sueCommandProvider(reporterId uint32, worldId world.Id, channelId channel.Id, accusedId uint32, subCommand string, flag byte, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(reporterId))
	value := &report2.Command[report2.CreateCommandBody]{
		Type: report2.CommandTypeCreate,
		Body: report2.CreateCommandBody{
			Kind:        report2.KindSue,
			WorldId:     worldId,
			ChannelId:   channelId,
			ReporterId:  reporterId,
			AccusedId:   accusedId,
			AccusedName: subCommand,
			ReasonType:  flag,
			Description: reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func claimCommandProvider(reporterId uint32, worldId world.Id, channelId channel.Id, targetName string, reasonType byte, description string, chatClaim bool, chatLog string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(reporterId))
	value := &report2.Command[report2.CreateCommandBody]{
		Type: report2.CommandTypeCreate,
		Body: report2.CreateCommandBody{
			Kind:        report2.KindClaim,
			WorldId:     worldId,
			ChannelId:   channelId,
			ReporterId:  reporterId,
			AccusedName: targetName,
			ReasonType:  reasonType,
			Description: description,
			ChatClaim:   chatClaim,
			ChatLog:     chatLog,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

`services/atlas-channel/atlas.com/channel/report/processor.go`:

```go
package report

import (
	report2 "atlas-channel/kafka/message/report"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	// Sue submits a /-command report. Legacy versions supply accusedId;
	// v95 supplies subCommand (forwarded as the accused name). The ban
	// service resolves whichever half is missing.
	Sue(reporterId uint32, worldId world.Id, channelId channel.Id, accusedId uint32, subCommand string, flag byte, reason string) error
	// Claim submits a CUIClaim report window submission.
	Claim(reporterId uint32, worldId world.Id, channelId channel.Id, targetName string, reasonType byte, description string, chatClaim bool, chatLog string) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) Sue(reporterId uint32, worldId world.Id, channelId channel.Id, accusedId uint32, subCommand string, flag byte, reason string) error {
	p.l.Debugf("Character [%d] sues [%d/%s] flag [%d].", reporterId, accusedId, subCommand, flag)
	return producer.ProviderImpl(p.l)(p.ctx)(report2.EnvCommandTopic)(sueCommandProvider(reporterId, worldId, channelId, accusedId, subCommand, flag, reason))
}

func (p *ProcessorImpl) Claim(reporterId uint32, worldId world.Id, channelId channel.Id, targetName string, reasonType byte, description string, chatClaim bool, chatLog string) error {
	p.l.Debugf("Character [%d] claims against [%s] type [%d] chatClaim [%t].", reporterId, targetName, reasonType, chatClaim)
	return producer.ProviderImpl(p.l)(p.ctx)(report2.EnvCommandTopic)(claimCommandProvider(reporterId, worldId, channelId, targetName, reasonType, description, chatClaim, chatLog))
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./report/... -v 2>&1 | tail -8 && go build ./...`
Expected: PASS; build clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/report/ services/atlas-channel/atlas.com/channel/report/
git commit -m "feat(channel): report command domain package"
```

---

### Task 14: atlas-channel — sue handler rewrite + claim handler

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/sue_character.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/claim_request.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (`produceHandlers()`, next to the `fieldsb.SueCharacterHandle` entry at line 836)

**Interfaces:**
- Consumes: Task 3 `reportsb "github.com/Chronicle20/atlas/libs/atlas-packet/report/serverbound"`; Task 13 `report.NewProcessor`; session getters `s.CharacterId() uint32`, `s.WorldId() world.Id`, `s.ChannelId() channel.Id`.
- Produces: `handler.ClaimRequestHandleFunc` registered as `handlerMap[reportsb.ClaimRequestHandle]`. No packet is written by either handler — the result packet comes from the Task 16 status consumer.

- [ ] **Step 1: Rewrite the sue handler**

`services/atlas-channel/atlas.com/channel/socket/handler/sue_character.go`:

```go
package handler

import (
	"atlas-channel/report"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func SueCharacterHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, ro map[string]interface{}) {
	return func(s session.Model, r *request.Reader, ro map[string]interface{}) {
		p := fieldsb.SueCharacter{}
		p.Decode(l, ctx)(r, ro)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		err := report.NewProcessor(l, ctx).Sue(s.CharacterId(), s.WorldId(), s.ChannelId(), p.CharacterId(), p.SubCommand(), p.Flag(), p.Reason())
		if err != nil {
			l.WithError(err).Errorf("Unable to submit sue report from character [%d].", s.CharacterId())
		}
	}
}
```

- [ ] **Step 2: Write the claim handler**

`services/atlas-channel/atlas.com/channel/socket/handler/claim_request.go`:

```go
package handler

import (
	"atlas-channel/report"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	reportsb "github.com/Chronicle20/atlas/libs/atlas-packet/report/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func ClaimRequestHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, ro map[string]interface{}) {
	return func(s session.Model, r *request.Reader, ro map[string]interface{}) {
		p := reportsb.ClaimRequest{}
		p.Decode(l, ctx)(r, ro)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		err := report.NewProcessor(l, ctx).Claim(s.CharacterId(), s.WorldId(), s.ChannelId(), p.TargetName(), p.ReasonType(), p.Description(), p.IsChatClaim(), p.ChatLog())
		if err != nil {
			l.WithError(err).Errorf("Unable to submit claim report from character [%d].", s.CharacterId())
		}
	}
}
```

- [ ] **Step 3: Register in produceHandlers**

In `services/atlas-channel/atlas.com/channel/main.go`, add import `reportsb "github.com/Chronicle20/atlas/libs/atlas-packet/report/serverbound"` and, next to the `fieldsb.SueCharacterHandle` line (836):

```go
	handlerMap[reportsb.ClaimRequestHandle] = handler.ClaimRequestHandleFunc
```

- [ ] **Step 4: Build and run handler-adjacent tests**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/... 2>&1 | tail -3`
Expected: build + existing tests clean. (Handler behavior is command emission; the producer bodies are pinned by Task 13's tests. If `CH/socket/handler` has an established captured-producer test pattern — check with `Glob services/atlas-channel/atlas.com/channel/socket/handler/*_test.go` — add an equivalent decode→emit test for `ClaimRequestHandleFunc` following it; if no such pattern exists, do not invent one.)

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/ services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(channel): sue handler forwards to report processor; new claim request handler"
```

---

### Task 15: atlas-channel — result/enable writers

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/writer/sue_character_result.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/writer/claim_result.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/writer/claim_available_time.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/writer/claim_status_changed.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/writer/claim_result_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (`produceWriters()`: append the four writer name constants)

**Interfaces:**
- Consumes: Task 1/2 codecs (`reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"`), `atlas_packet.WithResolvedCode` (DOM-25).
- Produces:

```go
type SueResultCode string
const (
	SueResultSuccess        SueResultCode = "SUCCESS"
	SueResultUnableToLocate SueResultCode = "UNABLE_TO_LOCATE"
	SueResultGenericFailure SueResultCode = "GENERIC_FAILURE"
)
type ClaimResultCode string
const (
	ClaimResultSuccessCode ClaimResultCode = "SUCCESS"
	ClaimResultTryAgain    ClaimResultCode = "TRY_AGAIN"
	ClaimResultRecheckName ClaimResultCode = "RECHECK_NAME"
)
func SueCharacterResultBody(key SueResultCode) packet.Encode
func ClaimResultSuccessBody(hasRemaining bool, remaining int32) packet.Encode
func ClaimResultNoticeBody(key ClaimResultCode) packet.Encode
func ClaimAvailableTimeBody(openHour byte, closeHour byte) packet.Encode
func ClaimSvrStatusChangedBody(connected bool) packet.Encode
```

- [ ] **Step 1: Write the failing writer-body test**

`services/atlas-channel/atlas.com/channel/socket/writer/claim_result_test.go`:

```go
package writer

import (
	"bytes"
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

func reportTestContext(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

var reportTestOptions = map[string]interface{}{
	"operations": map[string]interface{}{
		"SUCCESS":          "0x02",
		"TRY_AGAIN":        "0x41",
		"RECHECK_NAME":     "0x42",
		"UNABLE_TO_LOCATE": "0x01",
	},
}

func TestClaimResultSuccessBodyResolvesMode(t *testing.T) {
	l, _ := test.NewNullLogger()
	actual := ClaimResultSuccessBody(true, 100)(l, reportTestContext(t))(reportTestOptions)
	expected := []byte{0x02, 0x01, 0x64, 0x00, 0x00, 0x00}
	if !bytes.Equal(actual, expected) {
		t.Errorf("got %v want %v", actual, expected)
	}
}

func TestClaimResultNoticeBodyResolvesMode(t *testing.T) {
	l, _ := test.NewNullLogger()
	actual := ClaimResultNoticeBody(ClaimResultRecheckName)(l, reportTestContext(t))(reportTestOptions)
	expected := []byte{0x42}
	if !bytes.Equal(actual, expected) {
		t.Errorf("got %v want %v", actual, expected)
	}
}

func TestSueCharacterResultBodyResolvesCode(t *testing.T) {
	l, _ := test.NewNullLogger()
	sueOptions := map[string]interface{}{
		"operations": map[string]interface{}{"UNABLE_TO_LOCATE": "0x01"},
	}
	actual := SueCharacterResultBody(SueResultUnableToLocate)(l, reportTestContext(t))(sueOptions)
	expected := []byte{0x01}
	if !bytes.Equal(actual, expected) {
		t.Errorf("got %v want %v", actual, expected)
	}
}

func TestClaimEnableBodies(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx := reportTestContext(t)
	if got := ClaimAvailableTimeBody(0, 0)(l, ctx)(nil); !bytes.Equal(got, []byte{0x00, 0x00}) {
		t.Errorf("available time: %v", got)
	}
	if got := ClaimSvrStatusChangedBody(true)(l, ctx)(nil); !bytes.Equal(got, []byte{0x01}) {
		t.Errorf("status changed: %v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/writer/... -run 'ClaimResult|SueCharacterResult|ClaimEnable' 2>&1 | head -5`
Expected: FAIL to build — `undefined: ClaimResultSuccessBody`.

- [ ] **Step 3: Write the writers**

`services/atlas-channel/atlas.com/channel/socket/writer/sue_character_result.go`:

```go
package writer

import (
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// SueResultCode keys the SueCharacterResult writer's tenant operations
// table (DOM-25). DAILY_LIMIT and REPORTED_NOTICE exist in the table but
// have no v1 emitter (quota enforcement and accused notification deferred).
type SueResultCode string

const (
	SueResultSuccess        SueResultCode = "SUCCESS"
	SueResultUnableToLocate SueResultCode = "UNABLE_TO_LOCATE"
	SueResultGenericFailure SueResultCode = "GENERIC_FAILURE"
)

func SueCharacterResultBody(key SueResultCode) packet.Encode {
	return atlas_packet.WithResolvedCode("operations", string(key), func(code byte) packet.Encoder {
		return reportcb.NewSueCharacterResult(code)
	})
}
```

`services/atlas-channel/atlas.com/channel/socket/writer/claim_result.go`:

```go
package writer

import (
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// ClaimResultCode keys the ClaimResult writer's tenant operations table
// (DOM-25). The full verified mode set lives in the table; v1 emits only
// SUCCESS / TRY_AGAIN / RECHECK_NAME.
type ClaimResultCode string

const (
	ClaimResultSuccessCode ClaimResultCode = "SUCCESS"
	ClaimResultTryAgain    ClaimResultCode = "TRY_AGAIN"
	ClaimResultRecheckName ClaimResultCode = "RECHECK_NAME"
)

func ClaimResultSuccessBody(hasRemaining bool, remaining int32) packet.Encode {
	return atlas_packet.WithResolvedCode("operations", string(ClaimResultSuccessCode), func(code byte) packet.Encoder {
		return reportcb.NewClaimResultSuccess(code, hasRemaining, remaining)
	})
}

func ClaimResultNoticeBody(key ClaimResultCode) packet.Encode {
	return atlas_packet.WithResolvedCode("operations", string(key), func(code byte) packet.Encoder {
		return reportcb.NewClaimResultNotice(code)
	})
}
```

`services/atlas-channel/atlas.com/channel/socket/writer/claim_available_time.go`:

```go
package writer

import (
	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func ClaimAvailableTimeBody(openHour byte, closeHour byte) packet.Encode {
	return reportcb.NewClaimAvailableTime(openHour, closeHour).Encode
}
```

`services/atlas-channel/atlas.com/channel/socket/writer/claim_status_changed.go`:

```go
package writer

import (
	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func ClaimSvrStatusChangedBody(connected bool) packet.Encode {
	return reportcb.NewClaimSvrStatusChanged(connected).Encode
}
```

NOTE for the implementer: check the actual signature of `atlas_packet.WithResolvedCode` — it returns `func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`, which must satisfy `packet.Encode`; if `packet.Encode` is a distinct named type, wrap with a conversion `packet.Encode(...)`. Verify `packet.Encoder` is the interface WithResolvedCode's factory expects (it is in `libs/atlas-packet/resolve.go:13`).

- [ ] **Step 4: Register writer names**

In `services/atlas-channel/atlas.com/channel/main.go`, add import `reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"` and append to `produceWriters()` (near the `fieldcb.BlockedServerWriter` entry, line 752):

```go
		reportcb.SueCharacterResultWriter,
		reportcb.ClaimResultWriter,
		reportcb.ClaimAvailableTimeWriter,
		reportcb.ClaimSvrStatusChangedWriter,
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/writer/... -run 'ClaimResult|SueCharacterResult|ClaimEnable' -v 2>&1 | tail -8 && go build ./...`
Expected: PASS; build clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/writer/ services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(channel): sue/claim result and claim-enable writer bodies (config-resolved codes)"
```

---

### Task 16: atlas-channel — report status consumer

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/report/consumer.go`
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/report/consumer_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (InitConsumers at the line-174 block; InitHandlers registration at the line-422 block)

**Interfaces:**
- Consumes: Task 13 contract (`report2 "atlas-channel/kafka/message/report"`), Task 15 writer bodies, `session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId, op)`, `sc.IsWorld(...)` — exact shape of `CH/kafka/consumer/buddylist/consumer.go`.
- Produces: `report.InitConsumers(l)(cmf)(consumerGroupId)`; `report.InitHandlers(l)(sc)(wp)(rf) ([]listener.HandlerHandle, error)`; pure mapping function `resultPacket(e report2.StatusEvent) (string, packet.Encode, bool)` (exported for nothing — unexported, tested in-package). `remainingClaimsDisplay = 100` named constant (display text only; quotas untracked in v1).

**Mapping table (design §4.5):**

| kind | status | errorCode | packet |
|---|---|---|---|
| sue | CREATED | — | SueCharacterResult SUCCESS |
| sue | ERROR | NOT_FOUND | SueCharacterResult UNABLE_TO_LOCATE |
| sue | ERROR | anything else | SueCharacterResult GENERIC_FAILURE |
| claim | CREATED | — | ClaimResultSuccess(hasRemaining=1, remaining=100) |
| claim | ERROR | NOT_FOUND | ClaimResultNotice RECHECK_NAME |
| claim | ERROR | anything else | ClaimResultNotice TRY_AGAIN |

Reporter offline → `IfPresentByCharacterId` no-ops (feedback is best-effort; the report is already persisted).

- [ ] **Step 1: Write the failing mapping test**

`services/atlas-channel/atlas.com/channel/kafka/consumer/report/consumer_test.go`:

```go
package report

import (
	"testing"

	report2 "atlas-channel/kafka/message/report"

	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
)

func TestResultPacketMapping(t *testing.T) {
	cases := []struct {
		name       string
		event      report2.StatusEvent
		wantWriter string
		wantOk     bool
	}{
		{"sue created", report2.StatusEvent{Kind: report2.KindSue, Status: report2.EventStatusCreated}, reportcb.SueCharacterResultWriter, true},
		{"sue not found", report2.StatusEvent{Kind: report2.KindSue, Status: report2.EventStatusError, ErrorCode: report2.ErrorCodeNotFound}, reportcb.SueCharacterResultWriter, true},
		{"sue internal", report2.StatusEvent{Kind: report2.KindSue, Status: report2.EventStatusError, ErrorCode: report2.ErrorCodeInternal}, reportcb.SueCharacterResultWriter, true},
		{"claim created", report2.StatusEvent{Kind: report2.KindClaim, Status: report2.EventStatusCreated}, reportcb.ClaimResultWriter, true},
		{"claim not found", report2.StatusEvent{Kind: report2.KindClaim, Status: report2.EventStatusError, ErrorCode: report2.ErrorCodeNotFound}, reportcb.ClaimResultWriter, true},
		{"claim internal", report2.StatusEvent{Kind: report2.KindClaim, Status: report2.EventStatusError, ErrorCode: report2.ErrorCodeInternal}, reportcb.ClaimResultWriter, true},
		{"unknown kind dropped", report2.StatusEvent{Kind: "bogus", Status: report2.EventStatusCreated}, "", false},
		{"unknown status dropped", report2.StatusEvent{Kind: report2.KindSue, Status: "PENDING"}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writerName, body, ok := resultPacket(tc.event)
			if ok != tc.wantOk {
				t.Fatalf("ok: got %v want %v", ok, tc.wantOk)
			}
			if !ok {
				return
			}
			if writerName != tc.wantWriter {
				t.Errorf("writer: got %s want %s", writerName, tc.wantWriter)
			}
			if body == nil {
				t.Error("expected non-nil body")
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/report/... 2>&1 | head -5`
Expected: FAIL to build — package does not exist.

- [ ] **Step 3: Write the consumer**

`services/atlas-channel/atlas.com/channel/kafka/consumer/report/consumer.go`:

```go
package report

import (
	consumer2 "atlas-channel/kafka/consumer"
	report2 "atlas-channel/kafka/message/report"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// remainingClaimsDisplay feeds the client's "D reports left this week"
// success text. Quotas are untracked in v1; the value is display-only.
const remainingClaimsDisplay = 100

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("report_status_event")(report2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var handles []listener.HandlerHandle
				t, _ := topic.EnvProvider(l)(report2.EnvEventTopicStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleStatusEvent(sc server.Model, wp writer.Producer) message.Handler[report2.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e report2.StatusEvent) {
		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}
		writerName, body, ok := resultPacket(e)
		if !ok {
			l.Warnf("Dropping unmapped report status event kind [%s] status [%s] code [%s].", e.Kind, e.Status, e.ErrorCode)
			return
		}
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.ReporterId, session.Announce(l)(ctx)(wp)(writerName)(body))
		if err != nil {
			l.WithError(err).Errorf("Unable to deliver report result to character [%d].", e.ReporterId)
		}
	}
}

// resultPacket maps a report status event to the result packet the
// reporter sees (design.md §4.5). Unknown combinations are dropped, never
// sent as a guessed mode.
func resultPacket(e report2.StatusEvent) (string, packet.Encode, bool) {
	switch e.Kind {
	case report2.KindSue:
		switch {
		case e.Status == report2.EventStatusCreated:
			return reportcb.SueCharacterResultWriter, writer.SueCharacterResultBody(writer.SueResultSuccess), true
		case e.Status == report2.EventStatusError && e.ErrorCode == report2.ErrorCodeNotFound:
			return reportcb.SueCharacterResultWriter, writer.SueCharacterResultBody(writer.SueResultUnableToLocate), true
		case e.Status == report2.EventStatusError:
			return reportcb.SueCharacterResultWriter, writer.SueCharacterResultBody(writer.SueResultGenericFailure), true
		}
	case report2.KindClaim:
		switch {
		case e.Status == report2.EventStatusCreated:
			return reportcb.ClaimResultWriter, writer.ClaimResultSuccessBody(true, remainingClaimsDisplay), true
		case e.Status == report2.EventStatusError && e.ErrorCode == report2.ErrorCodeNotFound:
			return reportcb.ClaimResultWriter, writer.ClaimResultNoticeBody(writer.ClaimResultRecheckName), true
		case e.Status == report2.EventStatusError:
			return reportcb.ClaimResultWriter, writer.ClaimResultNoticeBody(writer.ClaimResultTryAgain), true
		}
	}
	return "", nil, false
}
```

- [ ] **Step 4: Wire main.go**

In `services/atlas-channel/atlas.com/channel/main.go` add import `reportstatus "atlas-channel/kafka/consumer/report"`, then:

- In the InitConsumers block (after `note3.InitConsumers` at line 209): `reportstatus.InitConsumers(l)(cmf)(consumerGroupId)`
- In the register block (after the `buddylist.InitHandlers` registration at line 422-424):

```go
		if err := register(reportstatus.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return err
		}
```

(match the exact error-handling shape of the surrounding `register(...)` calls at main.go:416-437.)

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/report/... -v 2>&1 | tail -12 && go build ./...`
Expected: PASS (8 mapping subtests); build clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/report/ services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(channel): report status consumer delivers result packets to the reporter"
```

---

### Task 17: atlas-channel — claim-enable emission on session bootstrap

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go` (inside `processStateReturn`, alongside the existing goroutine blocks at lines 207-260)

**Interfaces:**
- Consumes: Task 15 `writer.ClaimSvrStatusChangedBody` / `writer.ClaimAvailableTimeBody`; Task 1 writer name constants.
- Produces: after login, every session receives `ClaimSvrStatusChanged(true)` then `ClaimAvailableTime(0, 0)` (always-open — the client treats `open == close == 0` as always-available). **Gating is config-presence, not code:** if the tenant config lacks the `ClaimSvrStatusChanged` writer, `session.Announce`'s writer lookup errors — log at debug and skip the second send. No region/version conditionals in Go; jms/gms-92 tenants are disabled purely by having no template entries.

- [ ] **Step 1: Add the goroutine block**

In `processStateReturn` in `services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go`, add import `reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"` and insert after the buddy-list goroutine (after line 215):

```go
					go func() {
						// Claim UI enable: the client keeps CUIClaim disabled until
						// m_bClaimSvrConnected is set and an availability window
						// arrives (0,0 = always open). Tenants whose config lacks
						// these writers (jms, gms-92) skip both sends here — the
						// writer lookup failing IS the feature gate.
						err := session.Announce(l)(ctx)(wp)(reportcb.ClaimSvrStatusChangedWriter)(writer.ClaimSvrStatusChangedBody(true))(s)
						if err != nil {
							l.WithError(err).Debugf("Claim status writer unavailable for tenant; claim UI stays disabled for character [%d].", c.Id())
							return
						}
						err = session.Announce(l)(ctx)(wp)(reportcb.ClaimAvailableTimeWriter)(writer.ClaimAvailableTimeBody(0, 0))(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to write claim availability for character [%d].", c.Id())
						}
					}()
```

- [ ] **Step 2: Build and test**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./kafka/consumer/session/... 2>&1 | tail -3`
Expected: build + existing session tests clean. (Runtime behavior is exercised in Task 25's live acceptance: claim UI opens on a v83 client.)

- [ ] **Step 3: Run full channel gates**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./... 2>&1 | tail -5 && go vet ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go
git commit -m "feat(channel): send claim-enable packets on session bootstrap (config-gated)"
```

---

### Task 18: Seed templates, dispatcher-doc operations tables, topic env vars, live-config patch doc

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_61_1.json` (sue pair only)
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_72_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_79_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Create: `docs/packets/dispatchers/sue_character_result.yaml`
- Create: `docs/packets/dispatchers/claim_result.yaml`
- Modify: `deploy/k8s/base/env-configmap.yaml` (two topic entries)
- Regenerate: `deploy/k8s/overlays/pr/kustomization.yaml` + `deploy/k8s/overlays/main/kustomization.yaml` via `deploy/k8s/overlays/pr/scripts/gen-topic-config.sh`
- Create: `docs/tasks/task-145-player-reports/live-config-patch.md`

**gms_12, gms_48, gms_92 and jms_185 templates get NO entries** (PRD §2.1 / non-goal). jms and gms_12/gms_92 as before — no IDB or registry to verify opcodes against, and inventing them is forbidden. **gms_48 is excluded for a different reason**: its client has both clientbound receivers but neither send-site (verified — PRD FR-6.5), so writers there would answer requests that can never arrive. All stay feature-disabled via the Task 17 config gate.

**gms_61 is sue-only**: it gets the `SueCharacterResult` writer and nothing else. It has no `CLAIM_REQUEST` send-site, so adding claim writers would enable a UI that cannot submit.

- [ ] **Step 1: Add handler entries**

In the `socket.handlers` array of the six claim-capable templates add one entry (every entry carries a validator — the silent-drop gotcha). **gms_61 gets no handler entry.** The sue handler is already routed in every template — do not re-add it.

| template | entry |
|---|---|
| gms_72 | `{ "opCode": "0x69", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }` |
| gms_79 | `{ "opCode": "0x68", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }` |
| gms_83 | `{ "opCode": "0x6A", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }` |
| gms_84 | `{ "opCode": "0x6A", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }` |
| gms_87 | `{ "opCode": "0x6D", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }` |
| gms_95 | `{ "opCode": "0x76", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }` |

`"services": ["channel"]` matches the existing `SueCharacter` entries — omitting it is not the house style. Insert each at its **sorted position** in the array; `tools/template-opcode-order-guard.sh` enforces strictly ascending `opCode`.

The existing `SueCharacter` handler entries are already present and correct in every in-scope template — verified: gms_61 `0x68`, gms_72 `0x71`, gms_79 `0x70`, gms_83 `0x72`, gms_95 `0x7E`. Do not re-add or renumber them.

- [ ] **Step 2: Add writer entries**

Per-template opcodes (the operations tables are identical everywhere; only the `opCode` values move):

| template | SueCharacterResult | ClaimResult | ClaimAvailableTime | ClaimSvrStatusChanged |
|---|---|---|---|---|
| gms_61 | 0x34 | — | — | — |
| gms_72 | 0x34 | 0x2A | 0x2B | 0x2C |
| gms_79 | 0x34 | 0x2A | 0x2B | 0x2C |
| gms_83 | 0x37 | 0x2D | 0x2E | 0x2F |
| gms_84 | 0x37 | 0x2D | 0x2E | 0x2F |
| gms_87 | 0x37 | 0x2D | 0x2E | 0x2F |
| gms_95 | 0x37 | 0x2C | 0x2D | 0x2E |

Entry bodies (substitute the `opCode` from the table above; sorted insertion applies here too):

```json
{ "opCode": "<SueCharacterResult>", "writer": "SueCharacterResult", "services": ["channel"], "options": { "operations": {
  "SUCCESS": "0x00", "UNABLE_TO_LOCATE": "0x01", "DAILY_LIMIT": "0x02",
  "REPORTED_NOTICE": "0x03", "GENERIC_FAILURE": "0x04" } } },
{ "opCode": "<ClaimResult>", "writer": "ClaimResult", "services": ["channel"], "options": { "operations": {
  "SUCCESS": "0x02", "REPORTED_NOTICE": "0x03", "TRY_AGAIN": "0x41",
  "RECHECK_NAME": "0x42", "NOT_ENOUGH_MESOS": "0x43", "CANNOT_CONNECT": "0x44",
  "EXCEEDED": "0x45", "TIME_WINDOW": "0x47", "FALSE_REPORT_CITED": "0x48" } } },
{ "opCode": "<ClaimAvailableTime>", "writer": "ClaimAvailableTime", "services": ["channel"] },
{ "opCode": "<ClaimSvrStatusChanged>", "writer": "ClaimSvrStatusChanged", "services": ["channel"] }
```

(`services` placement matches the existing entries: after `writer`, before `options`.)

**Do not port v83 opcodes down to v61/v72/v79** — the clientbound trio sits a full region lower there, and a wrong opcode is a silent mis-send, not an error.

- [ ] **Step 3: Author the dispatcher-doc operations YAMLs**

Model: `docs/packets/dispatchers/note_operation.yaml`. These make `packet-audit operations --check` the permanent guard that template tables match the enumerations.

> **The mode values on the newly-supported columns are NOT yet verified.** The
> `v83 ≡ v95` equivalence in `packet-findings.md` was established for those two
> versions only. Before filling the `gms_v61`/`gms_v72`/`gms_v79` mode columns
> below, **read each table out of its own IDB** — do not copy v83's row across.
> There is positive reason for caution: `CWvsContext::OnClaimResult` is `0x2a7`
> bytes on v72/v79/v83 but `0x25d` on v48/v61, so the handler demonstrably
> changed shape somewhere in that range. Fill each cell from a decompile, and if
> a version's mode set genuinely differs, that is a real divergence to encode,
> not a mistake to smooth over. The `<verify>` placeholders below must not
> survive into the committed file.

`docs/packets/dispatchers/sue_character_result.yaml`:

```yaml
# SueCharacterResult — CWvsContext::OnSueCharacterResult result-code table.
# Codes 0-3 byte-identical in v83 (0xa29739) and v95 (0x9fae10), IDA-verified
# (task-145 packet-findings.md §1). v61/v72/v79 columns read from their own
# IDBs during this task. Any code outside 0-3 renders the generic-failure
# line; 4 is the deliberate "other" bucket key.
writer: SueCharacterResult
fname: CWvsContext::OnSueCharacterResult
op: SUE_CHARACTER_RESULT
direction: clientbound
opcodes:
  gms_v61: "0x34"
  gms_v72: "0x34"
  gms_v79: "0x34"
  gms_v83: "0x37"
  gms_v84: "0x37"
  gms_v87: "0x37"
  gms_v95: "0x37"
operations:
  - { key: SUCCESS,          modes: { gms_v61: <verify>, gms_v72: <verify>, gms_v79: <verify>, gms_v83: 0, gms_v84: 0, gms_v87: 0, gms_v95: 0 } }
  - { key: UNABLE_TO_LOCATE, modes: { gms_v61: <verify>, gms_v72: <verify>, gms_v79: <verify>, gms_v83: 1, gms_v84: 1, gms_v87: 1, gms_v95: 1 } }
  - { key: DAILY_LIMIT,      modes: { gms_v61: <verify>, gms_v72: <verify>, gms_v79: <verify>, gms_v83: 2, gms_v84: 2, gms_v87: 2, gms_v95: 2 } }
  - { key: REPORTED_NOTICE,  modes: { gms_v61: <verify>, gms_v72: <verify>, gms_v79: <verify>, gms_v83: 3, gms_v84: 3, gms_v87: 3, gms_v95: 3 } }
  - { key: GENERIC_FAILURE,  modes: { gms_v61: <verify>, gms_v72: <verify>, gms_v79: <verify>, gms_v83: 4, gms_v84: 4, gms_v87: 4, gms_v95: 4 } }
```

`docs/packets/dispatchers/claim_result.yaml` (no `gms_v61` column — claim is version-absent there):

```yaml
# ClaimResult — CWvsContext::OnClaimResult mode table. Mode sets identical in
# v83 (0xa27891) and v95 (0x9fa7d0), IDA-verified (task-145
# packet-findings.md §3). v72 (0x91f9a9) / v79 (0x9718f4) columns read from
# their own IDBs during this task. Only SUCCESS (2) carries payload
# (byte hasRemaining, int32 remaining); all other modes are bare notices.
writer: ClaimResult
fname: CWvsContext::OnClaimResult
op: CLAIM_RESULT
direction: clientbound
opcodes:
  gms_v72: "0x2A"
  gms_v79: "0x2A"
  gms_v83: "0x2D"
  gms_v84: "0x2D"
  gms_v87: "0x2D"
  gms_v95: "0x2C"
operations:
  - { key: SUCCESS,            modes: { gms_v72: <verify>, gms_v79: <verify>, gms_v83: 2, gms_v84: 2, gms_v87: 2, gms_v95: 2 } }
  - { key: REPORTED_NOTICE,    modes: { gms_v72: <verify>, gms_v79: <verify>, gms_v83: 3, gms_v84: 3, gms_v87: 3, gms_v95: 3 } }
  - { key: TRY_AGAIN,          modes: { gms_v72: <verify>, gms_v79: <verify>, gms_v83: 65, gms_v84: 65, gms_v87: 65, gms_v95: 65 } }
  - { key: RECHECK_NAME,       modes: { gms_v72: <verify>, gms_v79: <verify>, gms_v83: 66, gms_v84: 66, gms_v87: 66, gms_v95: 66 } }
  - { key: NOT_ENOUGH_MESOS,   modes: { gms_v72: <verify>, gms_v79: <verify>, gms_v83: 67, gms_v84: 67, gms_v87: 67, gms_v95: 67 } }
  - { key: CANNOT_CONNECT,     modes: { gms_v72: <verify>, gms_v79: <verify>, gms_v83: 68, gms_v84: 68, gms_v87: 68, gms_v95: 68 } }
  - { key: EXCEEDED,           modes: { gms_v72: <verify>, gms_v79: <verify>, gms_v83: 69, gms_v84: 69, gms_v87: 69, gms_v95: 69 } }
  - { key: TIME_WINDOW,        modes: { gms_v72: <verify>, gms_v79: <verify>, gms_v83: 71, gms_v84: 71, gms_v87: 71, gms_v95: 71 } }
  - { key: FALSE_REPORT_CITED, modes: { gms_v72: <verify>, gms_v79: <verify>, gms_v83: 72, gms_v84: 72, gms_v87: 72, gms_v95: 72 } }
```

Before finalizing, open `docs/packets/dispatchers/note_operation.yaml` and match its exact field set (if the schema has extra required keys, mirror them; if `op:` values must match registry op names, confirm `SUE_CHARACTER_RESULT`/`CLAIM_RESULT` against `docs/packets/registry/gms_v83.yaml`).

- [ ] **Step 4: Validate templates against the docs**

Run: `go run ./tools/packet-audit operations --check`
Expected: `operations check OK` (exit 0). If it reports drift, fix the template tables (the YAML docs are source of truth). Also sanity-check JSON:

Run: `for v in 61 72 79 83 84 87 95; do f=services/atlas-configurations/seed-data/templates/template_gms_${v}_1.json; python3 -c "import json;json.load(open('$f'))" && echo "$f OK"; done`
Expected: all OK.

Then run: `tools/template-opcode-order-guard.sh`
Expected: exit 0. A failure means an entry was appended rather than inserted at its sorted `opCode` position.

- [ ] **Step 5: Add topic env vars**

In `deploy/k8s/base/env-configmap.yaml`, add alphabetically:

```yaml
  COMMAND_TOPIC_REPORT: "COMMAND_TOPIC_REPORT"
  EVENT_TOPIC_REPORT_STATUS: "EVENT_TOPIC_REPORT_STATUS"
```

Then regenerate the overlay topic lists:

Run: `deploy/k8s/overlays/pr/scripts/gen-topic-config.sh`
Expected: `COMMAND_TOPIC_REPORT` / `EVENT_TOPIC_REPORT_STATUS` rows appear in both `deploy/k8s/overlays/pr/kustomization.yaml` and `deploy/k8s/overlays/main/kustomization.yaml` (verify with grep; if the script only regenerates pr, apply the same edit pattern to main by hand following the existing suffix convention).

- [ ] **Step 6: Write the live-config patch doc**

`docs/tasks/task-145-player-reports/live-config-patch.md` — seed templates apply only at tenant creation; existing tenants need a live PATCH. Document, for each existing gms-83/84/87/95 tenant:

1. The `socket.handlers` entry and four `socket.writers` entries (exact JSON from Steps 1-2, per version) to merge into the tenant's socket configuration via the atlas-tenants configurations API (JSON:API envelope — bare bodies 400).
2. atlas-channel restart requirement (the projection does not hot-reload handlers/writers).
3. New env vars for existing deployments: `CHARACTERS_SERVICE_URL` / `MESSAGES_SERVICE_URL` on atlas-ban (optional — `BASE_SERVICE_URL` fallback) and `CHAT_CAPTURE_RETENTION_SECONDS` / `CHAT_CAPTURE_MAX_LINES` on atlas-messages (optional — defaults 900/200).
4. Explicit note: gms-92 and jms tenants get NO patch (feature stays disabled).

Use repo-relative paths and placeholder tenant ids (never real hostnames/homes).

- [ ] **Step 7: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/ docs/packets/dispatchers/sue_character_result.yaml docs/packets/dispatchers/claim_result.yaml deploy/k8s/ docs/tasks/task-145-player-reports/live-config-patch.md
git commit -m "feat(config): report handler/writer seed entries, operations docs, topic env vars"
```

---

### Task 19: Ingress route for /api/reports

**Files:**
- Modify: `deploy/shared/routes.conf`
- Modify: `deploy/compose/routes.conf`
- Regenerate: k8s ingress via `deploy/scripts/sync-k8s-ingress-routes.sh`

- [ ] **Step 1: Add the route to both routes.conf files**

Alphabetical placement (immediately after the existing `/api/bans` block at `deploy/shared/routes.conf:312`, and its counterpart in `deploy/compose/routes.conf`):

```nginx
location ~ ^/api/reports(/.*)?$ {
  set $u "atlas-ban:8080";
  proxy_pass http://$u$request_uri;
}
```

- [ ] **Step 2: Sync k8s ingress**

Run: `deploy/scripts/sync-k8s-ingress-routes.sh`
Expected: exits 0; `git status` shows the generated k8s ingress artifacts updated. (If the script lives elsewhere, locate with `Glob deploy/**/sync-k8s-ingress-routes.sh`.)

- [ ] **Step 3: Commit**

```bash
git add deploy/
git commit -m "feat(deploy): route /api/reports to atlas-ban"
```

---

### Task 20: atlas-ui — report types, service, hooks

**Files:**
- Create: `services/atlas-ui/src/types/models/report.ts`
- Create: `services/atlas-ui/src/services/api/reports.service.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/useReports.ts`

**Interfaces:**
- Consumes: `api` client (`@/lib/api/client`), `ServiceOptions`/`QueryOptions` (`@/lib/api/query-params`), `Tenant` type — exactly as `bans.service.ts`/`useBans.ts`.
- Produces: `Report`, `ReportAttributes`, `TranscriptLine`, `ReportStatus`, `ReportKind` (const-object enums — `erasableSyntaxOnly`); `reportsService.getAllReports/getReportById/updateReportStatus`; `reportKeys`, `useReports(tenant, options?)`, `useReport(tenant, id)`, `useUpdateReportStatus()`.

- [ ] **Step 1: Write the types**

`services/atlas-ui/src/types/models/report.ts`:

```ts
// Report domain model types (player sue/claim reports reviewed by GMs).

/**
 * Const-object enums rather than TS `enum` so the type-only strip the
 * Vite/ESBuild toolchain performs is lossless (see `erasableSyntaxOnly`
 * in tsconfig.app.json).
 */
export const ReportStatus = {
    Open: 'open',
    Reviewed: 'reviewed',
    Actioned: 'actioned',
} as const;
export type ReportStatus = typeof ReportStatus[keyof typeof ReportStatus];

export const ReportStatusLabels: Record<ReportStatus, string> = {
    [ReportStatus.Open]: 'Open',
    [ReportStatus.Reviewed]: 'Reviewed',
    [ReportStatus.Actioned]: 'Actioned',
};

export const ReportKind = {
    Sue: 'sue',
    Claim: 'claim',
} as const;
export type ReportKind = typeof ReportKind[keyof typeof ReportKind];

export const ReportKindLabels: Record<ReportKind, string> = {
    [ReportKind.Sue]: 'Sue',
    [ReportKind.Claim]: 'Claim',
};

export interface TranscriptLine {
    timestamp: number;
    senderId: number;
    senderName: string;
    chatType: string;
    text: string;
}

export interface ReportAttributes {
    kind: ReportKind;
    reporterId: number;
    reporterName: string;
    accusedId: number;
    accusedName: string;
    reasonType: number;
    description: string;
    chatLog: string | null;
    serverTranscript: TranscriptLine[] | null;
    status: ReportStatus;
    createdAt: string;
    updatedAt: string;
}

export interface Report {
    id: string;
    type: 'reports';
    attributes: ReportAttributes;
}
```

- [ ] **Step 2: Write the service**

`services/atlas-ui/src/services/api/reports.service.ts`:

```ts
import { api } from "@/lib/api/client";
import { type ServiceOptions, type QueryOptions } from "@/lib/api/query-params";
import type { Report, ReportStatus } from "@/types/models/report";

const BASE_PATH = "/api/reports";

export interface ReportQueryOptions extends QueryOptions {
  status?: ReportStatus;
}

function sortReports(reports: Report[]): Report[] {
  return reports.sort(
    (a, b) => new Date(b.attributes.createdAt).getTime() - new Date(a.attributes.createdAt).getTime(),
  );
}

export const reportsService = {
  async getAllReports(options?: ReportQueryOptions): Promise<Report[]> {
    let url = BASE_PATH;
    if (options?.status !== undefined) {
      url += `?status=${options.status}`;
    }
    const reports = await api.getList<Report>(url, options);
    return sortReports(reports);
  },

  async getReportById(id: string, options?: ServiceOptions): Promise<Report> {
    return api.getOne<Report>(`${BASE_PATH}/${id}`, options);
  },

  async updateReportStatus(id: string, status: ReportStatus, options?: ServiceOptions): Promise<void> {
    // JSON:API envelope — bare attribute bodies are rejected with 400 by
    // RegisterInputHandler endpoints.
    await api.patch<void>(
      `${BASE_PATH}/${id}`,
      { data: { type: "reports", id, attributes: { status } } },
      options,
    );
  },
};
```

- [ ] **Step 3: Write the hooks**

`services/atlas-ui/src/lib/hooks/api/useReports.ts`:

```ts
/**
 * React Query hooks for player-report review.
 */

import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from "@tanstack/react-query";
import { reportsService, type ReportQueryOptions } from "@/services/api/reports.service";
import type { Report, ReportStatus } from "@/types/models/report";
import type { Tenant } from "@/types/models/tenant";
import type { ServiceOptions } from "@/lib/api/query-params";

export const reportKeys = {
  all: ["reports"] as const,
  lists: () => [...reportKeys.all, "list"] as const,
  list: (tenant: Tenant | null, options?: ReportQueryOptions) =>
    [...reportKeys.lists(), tenant?.id ?? "no-tenant", options] as const,
  details: () => [...reportKeys.all, "detail"] as const,
  detail: (tenant: Tenant | null, id: string) =>
    [...reportKeys.details(), tenant?.id ?? "no-tenant", id] as const,
};

export function useReports(
  tenant: Tenant | null,
  options?: ReportQueryOptions,
): UseQueryResult<Report[], Error> {
  return useQuery({
    queryKey: reportKeys.list(tenant, options),
    queryFn: () => reportsService.getAllReports(options),
    enabled: !!tenant?.id,
    gcTime: 5 * 60 * 1000,
  });
}

export function useReport(
  tenant: Tenant | null,
  id: string,
  options?: ServiceOptions,
): UseQueryResult<Report, Error> {
  return useQuery({
    queryKey: reportKeys.detail(tenant, id),
    queryFn: () => reportsService.getReportById(id, options),
    enabled: !!tenant?.id && !!id,
    gcTime: 5 * 60 * 1000,
  });
}

export function useUpdateReportStatus(): UseMutationResult<
  void,
  Error,
  { id: string; status: ReportStatus }
> {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }) => reportsService.updateReportStatus(id, status),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: reportKeys.all });
    },
  });
}

export function useInvalidateReports() {
  const queryClient = useQueryClient();
  return {
    invalidateAll: () => queryClient.invalidateQueries({ queryKey: reportKeys.all }),
  };
}
```

- [ ] **Step 4: Typecheck**

Run: `cd services/atlas-ui && npx tsc --noEmit 2>&1 | head -10`
Expected: no new errors. Before writing, compare against `bans.service.ts`/`useBans.ts` and match any project-specific detail this plan missed (e.g. `api.patch` argument order, `QueryOptions` shape).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/types/models/report.ts services/atlas-ui/src/services/api/reports.service.ts services/atlas-ui/src/lib/hooks/api/useReports.ts
git commit -m "feat(ui): report types, API service, React Query hooks"
```

---

### Task 21: atlas-ui — ReportsPage (list) + registration

**Files:**
- Create: `services/atlas-ui/src/components/features/reports/ReportStatusBadge.tsx`
- Create: `services/atlas-ui/src/pages/reports-columns.tsx`
- Create: `services/atlas-ui/src/pages/ReportsPage.tsx`
- Modify: `services/atlas-ui/src/App.tsx` (lazy routes `/reports`, `/reports/:reportId` next to the bans routes at lines 79-80)
- Modify: `services/atlas-ui/src/components/app-sidebar.tsx` (item `{ title: "Reports", url: "/reports" }` under the Security group at line 64-68)
- Modify: `services/atlas-ui/src/lib/breadcrumbs/routes.ts` (entries for `/reports` and `/reports/:reportId`)

**Interfaces:**
- Consumes: Task 20 hooks/types; `DataTableWrapper`, `useGridRefresh`, shadcn `Select`/`Badge`/`Skeleton` — mirror `BansPage.tsx` structurally (list page skeleton, filter select, refresh, row click navigates to detail).
- Produces: `ReportStatusBadge({ status })`; `getColumns(...)`/`hiddenColumns` for the reports table; routed pages.

- [ ] **Step 1: Write the status badge**

`services/atlas-ui/src/components/features/reports/ReportStatusBadge.tsx`:

```tsx
import { Badge } from "@/components/ui/badge";
import { ReportStatus, ReportStatusLabels } from "@/types/models/report";

const statusVariant: Record<ReportStatus, "default" | "secondary" | "outline"> = {
  [ReportStatus.Open]: "default",
  [ReportStatus.Reviewed]: "secondary",
  [ReportStatus.Actioned]: "outline",
};

export function ReportStatusBadge({ status }: { status: ReportStatus }) {
  return <Badge variant={statusVariant[status] ?? "default"}>{ReportStatusLabels[status] ?? status}</Badge>;
}
```

- [ ] **Step 2: Write the columns**

`services/atlas-ui/src/pages/reports-columns.tsx` — mirror `bans-columns.tsx` conventions (open it and copy its `ColumnDef` scaffolding, header helpers, and any actions-menu pattern) with these data columns:

| column | accessor | rendering |
|---|---|---|
| Kind | `attributes.kind` | `ReportKindLabels[kind]` |
| Reporter | `attributes.reporterName` | text (fallback `#reporterId`) |
| Accused | `attributes.accusedName` | text (fallback `#accusedId`) |
| Reason | `attributes.reasonType` | numeric wire byte, plain text |
| Status | `attributes.status` | `<ReportStatusBadge status={...} />` |
| Created | `attributes.createdAt` | locale date-time string |

- [ ] **Step 3: Write the page**

`services/atlas-ui/src/pages/ReportsPage.tsx` — structural copy of `BansPage.tsx` (skeleton component, `useTenant`, `useGridRefresh`) with: status filter `Select` over `all | open | reviewed | actioned` feeding `useReports(activeTenant, statusFilter !== "all" ? { status } : undefined)`; `DataTableWrapper` with the Task 21 columns; row click → `navigate(\`/reports/${report.id}\`)`. No create/delete dialogs (reports are created in-game only).

- [ ] **Step 4: Register routes, sidebar, breadcrumbs**

- `App.tsx`: lazy-import ReportsPage and add inside the AppShell layout route, next to bans:

```tsx
<Route path="/reports" element={<ReportsPage />} />
```

(The `/reports/:reportId` route is registered in Task 22 alongside ReportDetailPage; until then, do not add row-click navigation dead ends — the row click handler can land here but 404s harmlessly under the SPA router for one task.)

- `app-sidebar.tsx`: add `{ title: "Reports", url: "/reports" }` to the Security group's items after Bans.
- `lib/breadcrumbs/routes.ts`: append to `ROUTE_CONFIGS`:

```ts
  {
    pattern: '/reports',
    label: 'Reports',
    parent: '/',
  },
  {
    pattern: '/reports/[reportId]',
    label: 'Report Detail',
    parent: '/reports',
  },
```

(match the dynamic-segment `pattern` syntax used by existing detail entries in that file — copy an `/accounts/[id]`-style entry's exact shape.)

- [ ] **Step 5: Typecheck + lint + tests**

Run: `cd services/atlas-ui && npx tsc --noEmit && npm run lint 2>&1 | tail -5 && npm test 2>&1 | tail -5`
Expected: clean (match whatever test script the package.json defines; if none, tsc+lint suffice — the Bans feature's testing level is the bar).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/
git commit -m "feat(ui): reports list page with status filter"
```

---

### Task 22: atlas-ui — ReportDetailPage + status update dialog

**Files:**
- Create: `services/atlas-ui/src/components/features/reports/UpdateReportStatusDialog.tsx`
- Create: `services/atlas-ui/src/pages/ReportDetailPage.tsx`
- Modify: `services/atlas-ui/src/App.tsx` (add the `/reports/:reportId` route)

**Interfaces:**
- Consumes: Task 20 `useReport`, `useUpdateReportStatus`; shadcn `Card`, `Dialog`, `Select`, `Table`; `BanDetailPage.tsx` as the structural model.
- Produces: detail view with description | client chat log | server transcript side by side; status PATCH from the detail view.

- [ ] **Step 1: Write the dialog**

`services/atlas-ui/src/components/features/reports/UpdateReportStatusDialog.tsx` — mirror the Bans feature's dialog conventions (`components/features/bans/ExpireBanDialog.tsx` is the model for props/open-state/toast handling):

```tsx
import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import { useUpdateReportStatus } from "@/lib/hooks/api/useReports";
import { ReportStatus, ReportStatusLabels } from "@/types/models/report";
import type { Report } from "@/types/models/report";

interface UpdateReportStatusDialogProps {
  report: Report;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function UpdateReportStatusDialog({ report, open, onOpenChange }: UpdateReportStatusDialogProps) {
  const [status, setStatus] = useState<ReportStatus>(report.attributes.status);
  const updateStatus = useUpdateReportStatus();

  const handleSubmit = () => {
    updateStatus.mutate(
      { id: report.id, status },
      {
        onSuccess: () => {
          toast.success(`Report marked ${ReportStatusLabels[status]}`);
          onOpenChange(false);
        },
        onError: (error) => {
          toast.error(`Failed to update report status: ${error.message}`);
        },
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Update Report Status</DialogTitle>
          <DialogDescription>
            Set the review state for the report against {report.attributes.accusedName}.
          </DialogDescription>
        </DialogHeader>
        <Select value={status} onValueChange={(v) => setStatus(v as ReportStatus)}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {Object.values(ReportStatus).map((s) => (
              <SelectItem key={s} value={s}>
                {ReportStatusLabels[s]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={updateStatus.isPending || status === report.attributes.status}>
            {updateStatus.isPending ? "Saving..." : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

- [ ] **Step 2: Write the detail page**

`services/atlas-ui/src/pages/ReportDetailPage.tsx` — structural copy of `BanDetailPage.tsx` (param parsing, loading skeleton, not-found state, `Toaster`). Content layout:

- Header row: kind label + `ReportStatusBadge` + "Update Status" button opening the dialog; reporter → accused line; created/updated timestamps.
- Three-column grid (`grid gap-4 lg:grid-cols-3`, stacking on small screens), one `Card` each:
  1. **Description** — `<p className="whitespace-pre-wrap text-sm">{attributes.description}</p>`
  2. **Client Chat Log** — `attributes.chatLog` rendered as `<pre className="whitespace-pre-wrap text-sm">{chatLog}</pre>`, or a muted "No chat log submitted." — ALWAYS a text node, never `dangerouslySetInnerHTML` (user-generated content).
  3. **Server Transcript** — when `attributes.serverTranscript?.length`, a simple `Table` with Time (locale time from `timestamp`), Sender (`senderName`), Type (`chatType`), Message (`text`, `whitespace-pre-wrap`); else muted "No server transcript captured."

- [ ] **Step 3: Register the detail route**

`App.tsx`: `<Route path="/reports/:reportId" element={<ReportDetailPage />} />` next to the `/reports` route (lazy-imported like its neighbors).

- [ ] **Step 4: Typecheck + lint + tests**

Run: `cd services/atlas-ui && npx tsc --noEmit && npm run lint 2>&1 | tail -5`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/
git commit -m "feat(ui): report detail page with transcript view and status updates"
```

---

### Task 23: IDA — name the v83 CLAIM_REQUEST send-site and byte-verify its body (FR-6.2)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v83.json` (surgical splice — the export is NON-idempotent; never regenerate/overwrite)
- Possibly modify: `libs/atlas-packet/report/serverbound/claim_request.go` (only if the v83 body disagrees with v95 — then add the inline `t.MajorVersion()` guard and update Task 3's tests)

**This is in-scope implementation work, not a deferrable gap.** The v83 registry row for `CWvsContext::SendClaimRequest` is `csv-import` provenance — the function is unnamed in the v83_Me IDB.

- [ ] **Step 1: Select the v83 instance and confirm identity**

Use ida-pro-mcp: `select_instance(13342)` (v83_Me IDB), then `server_health`/`survey_binary` to confirm the loaded binary is the v83 dump before reading anything.

- [ ] **Step 2: Locate the send-site**

Anchors (from `packet-findings.md` §2):
- `CUIClaim::OnCreate` @ `0x7db17d` — walk its xrefs/callees (`func_query` with `name_regex` for known names; `xrefs_to` / `callees` for the graph).
- The send-site constructs `COutPacket::COutPacket(&oPacket, 106)` (106 = 0x6A, the v83 CLAIM_REQUEST opcode). `find_xref_signatures` / `find_bytes` against the COutPacket-with-106 pattern narrows candidates.

Decompile the candidate and confirm it encodes: byte, string, byte, string, conditional string — matching v95's `CWvsContext::SendClaimRequest` @ `0xa05fb0` (decompile that on instance 13341 for side-by-side comparison).

- [ ] **Step 3: Name it and save**

`rename` the function to `CWvsContext::SendClaimRequest`, `idb_save`. If no candidate decompiles to the expected shape, STOP AND ASK — an unresolvable fname is a stop-and-ask, never a substituted name or hash.

- [ ] **Step 4: Byte-verify the v83 body**

Quote the decompiled v83 encode order in the task notes. If it differs from the Task 3 codec (unexpected per findings), update the codec with an inline `t.MajorVersion()` version guard + new golden fixtures FIRST, then re-run `cd libs/atlas-packet && go test ./report/...`.

- [ ] **Step 5: Splice the export**

Splice ONLY the newly named function's record into `docs/packets/ida-exports/gms_v83.json` following `docs/packets/audits/VERIFYING_A_PACKET.md` §9-10 (surgical splice; strip any COutPacket-delegate harvest artifact). Verify with `git diff --stat` that only the intended record changed.

- [ ] **Step 6: Commit**

```bash
git add docs/packets/ida-exports/gms_v83.json
git commit -m "chore(packets): name v83 CWvsContext::SendClaimRequest and splice export"
```

---

### Task 23b: Registry correction — CLAIM_REQUEST exists on v72 and v79 (PRD FR-6.4)

**Files:**
- Modify: `docs/packets/registry/gms_v72.yaml` (add the `CLAIM_REQUEST` serverbound row)
- Modify: `docs/packets/registry/gms_v79.yaml` (same)
- Modify: `docs/packets/ida-exports/gms_v72.json`, `docs/packets/ida-exports/gms_v79.json` (surgical splice — NON-idempotent, never regenerate)
- Modify: `docs/packets/audits/STATUS.md` + `status.json` (regenerated, never hand-edited)

`STATUS.md` currently renders v72 and v79 `CLAIM_REQUEST` as `⬜` (n-a). **That is a false negative, not a real absence** — the registry simply has no row. Unlike Task 23's v83 case, the send-sites are **already named** in both IDBs, so there is no naming step here.

- [ ] **Step 1: Confirm the send-sites (already named — just re-read them)**

`idb_open` the v72 (`GMS_v72.1_U_DEVM.exe.i64`) and v79 (`GMS_v79_1_DEVM.exe.i64`) IDBs, then `func_query` `name_regex: "(?i)SendClaimRequest"`.

Expected: `?SendClaimRequest@CWvsContext@@QAEXAAV?$ZArray@V?$ZXString@D@@@@V?$ZXString@D@@@Z` at `0x91f2b4` (v72) and `0x9711ff` (v79), size `0x6bf` each.

- [ ] **Step 2: Re-read the opcode and body from the decompile**

Decompile each and quote the packet-build block. Expected (verified 2026-08-04):

```
v72 @0x91f749: COutPacket(105)   // 0x69
v79           : COutPacket(104)   // 0x68
then, both:  Encode1(bChatClaim)
             EncodeStr(targetName)
             Encode1(reasonType)
             EncodeStr(description)
             if (bChatClaim) EncodeStr(chatLog)
             CClientSocket::SendPacket
```

This is byte-for-byte the v95 shape, so the Task 3 codec needs no version branch. If a re-read disagrees with the above, the decompile wins — update the codec and STOP to re-scope.

- [ ] **Step 3: Add the registry rows**

Add a `CLAIM_REQUEST` entry to each registry mirroring the v83 row's field set (`op`, `opcode`, `fname: CWvsContext::SendClaimRequest`, `direction: serverbound`), with a `note:` recording that the row was added by task-145 from a live IDB read (address + opcode + date) and that it corrects a prior n-a false negative. Provenance is an IDA read, **not** `csv-import`.

- [ ] **Step 4: Splice the exports**

Splice ONLY the new function records into the two export JSONs per `docs/packets/audits/VERIFYING_A_PACKET.md` §9-10. Confirm with `git diff --stat` that only the intended records changed.

- [ ] **Step 5: Regenerate and check the matrix**

Run: `go run ./tools/packet-audit matrix` then `go run ./tools/packet-audit matrix --check`
Expected: exit 0, and the v72/v79 `CLAIM_REQUEST` cells now render `❌` (present, unimplemented) instead of `⬜`. They become `✅` in Task 24.

Also run `go run ./tools/packet-audit doc-freshness --check` — the version-set facts block is unchanged, but the run must stay clean.

- [ ] **Step 6: Commit**

```bash
git add docs/packets/registry/gms_v72.yaml docs/packets/registry/gms_v79.yaml \
        docs/packets/ida-exports/gms_v72.json docs/packets/ida-exports/gms_v79.json \
        docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "fix(packets): register CLAIM_REQUEST on gms v72/v79 (corrects n-a false negative)"
```

---

### Task 23c: Verified absences (PRD FR-6.5) — evidence already recorded

**Files:** none to author — `packet-findings.md` §7 was written during the spec update and is already on the branch.

§7 records, with addresses and decompile evidence, the three `⬜` cells this task's scoping *depends on*: v48 sue (§7.1) and v48/v61 claim (§7.2). They exist so a future reader neither re-litigates them nor "corrects" them the way Task 23b corrects v72/v79.

This task is therefore a **confirmation step, not authoring work**:

- [ ] **Step 1: Re-read §7 before relying on it**

Confirm the three checks in §7.2 still hold against the current IDBs (they are cheap: one `func_query`, one `search_text`, one neighbour lookup per version). If any check now returns a hit, **stop** — a claim send-site on v48/v61 would move those versions into scope and invalidate the Task 18 template split.

- [ ] **Step 2: Confirm nothing was filed as a deferral**

`grep -n "gms-48\|gms_48\|v61 claim" docs/TODO.md` must return nothing. These are permanent absences, not blocked work; listing them would send someone hunting for packets that do not exist (see Task 25 Step 1).

---

### Task 24: Packet verification campaign — 31 matrix cells

**Files (per cell, produced by the packet-verifier flow):**
- Modify: the five codec test files (add `packet-audit:verify` markers with per-version IDA addresses)
- Create/modify: evidence records under `docs/packets/evidence/gms_v{61,72,79,83,84,87,95}/`
- Regenerate: `docs/packets/audits/STATUS.md` + `status.json`

**Scope: 31 cells promoted to ✅** — not a flat product, because the two mechanisms have different version spans (PRD §2.1):

| packet | versions | cells |
|---|---|---|
| `SUE_CHARACTER_RESULT` (cb) | v61, v72, v79, v83, v84, v87, v95 | 7 |
| `CLAIM_RESULT` (cb) | v72, v79, v83, v84, v87, v95 | 6 |
| `CLAIM_AVAILABLE_TIME` (cb) | v72, v79, v83, v84, v87, v95 | 6 |
| `CLAIM_STATUS_CHANGED` (cb) | v72, v79, v83, v84, v87, v95 | 6 |
| `CLAIM_REQUEST` (sb) | v72, v79, v83, v84, v87, v95 | 6 |

jms cells stay ❌/⬜ (out of scope). **v48 cells stay ⬜ and are documented, not promoted** (Task 23c) — there is nothing to implement there. Static claims are NOT verification (FR-6.1) — every cell needs the decompile-derived read/write order, a byte-fixture with marker + IDA address, the pinned evidence record, and matrix regeneration, committed together.

- [ ] **Step 1: Dispatch packet-verifier agents, batched per IDB**

Follow `docs/packets/audits/VERIFYING_A_PACKET.md` exactly. Dispatch the `packet-verifier` agent once per packet × version, batched by IDB instance (all v83 cells together, then v95, v84, v87, v61, v72, v79 per their IDBs — resolve current ports via `idb_list`, do not assume) — never two agents against the same IDA instance concurrently. Tasks 23 and 23b must both be complete first: v83's `CLAIM_REQUEST` fname must resolve, and v72/v79 must have registry rows to promote.

- [ ] **Step 2: Handle known gotchas**

- v84 clientbound opcodes sit in the post-0x3D shifted region for the claim trio — the STATUS.md rows used in Task 18 already reflect the shift, but each v84 cell must still be verified from the v84 IDB, not ported from v83.
- **v61/v72/v79 clientbound opcodes sit a full region BELOW v83's** (0x34 vs 0x37; 0x2A–0x2C vs 0x2D–0x2F). Verify each from its own IDB. A ported v83 opcode here is a silent mis-send, not a build error.
- The v61↔v72 boundary is where the claim mechanism appears and the v79↔v83 boundary is where the opcodes shift, so `packet-audit gate-check --check` will want fixtures on **both** sides of each gated divergence. Budget for that rather than discovering it at CI.
- Serverbound CLAIM_REQUEST cells need marker + evidence + REPORT (report generated via the root `-ida-source` flag per the serverbound-verification memory).
- An op whose fname doesn't resolve in an IDA export is a stop-and-ask; never auto-re-export or substitute.

- [ ] **Step 3: Verify the matrix**

Run: `go run ./tools/packet-audit matrix --check && go run ./tools/packet-audit fname-doc --check && go run ./tools/packet-audit operations --check && go run ./tools/packet-audit gate-check --check`
Expected: exit 0 for all four; `grep -E 'SUE_CHARACTER_RESULT|CLAIM_' docs/packets/audits/STATUS.md` shows ✅ in the v61 column for `SUE_CHARACTER_RESULT` and in the v72–v95 columns for all five rows, with v48 still ⬜ and jms unchanged.

- [ ] **Step 4: Commit** (the verifier flow commits per-cell artifacts together; finish with the regenerated matrix)

```bash
git add docs/packets/ libs/atlas-packet/
git commit -m "test(packets): verify sue/claim packet cells across gms v61-v95"
```

---

### Task 25: Deferrals, docs, and final verification gates

**Files:**
- Modify: `docs/TODO.md` (locate via Glob first)
- All changed modules (verification only)

- [ ] **Step 1: Record explicit deferrals in docs/TODO.md**

Add entries (not code TODOs) under the appropriate section:

- Report quota / mesos-cost enforcement (sue result code 2 `DAILY_LIMIT`; claim modes 0x43/0x45/0x47/0x48 + real `remaining` counts) — result-code plumbing already expressive; wire counting + config when prioritized.
- Accused-notification codes (sue 3 / claim mode 3 `REPORTED_NOTICE`) — writers accept the keys already.
- gms-12 / gms-92 report enablement — blocked on registry files + IDBs (opcodes unverifiable); config entries only, no code.
- jms claim support — claim opcodes exist in the jms matrix; template entries + verification when jms is in scope.

Do **not** file gms-48 or gms-61-claim as deferrals: those are verified permanent absences (Task 23c), not blocked work. Recording them as "TODO" would invite someone to go looking for packets that do not exist.

- [ ] **Step 2: Full Go gates on every changed module**

Run each; ALL must be clean:

```bash
(cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./...)
(cd libs/atlas-redis && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-ban/atlas.com/ban && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-messages/atlas.com/messages && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...)
```

- [ ] **Step 3: Docker bakes (mandatory — go build will NOT catch missing Dockerfile COPY lines)**

Run from the worktree root:

```bash
docker buildx bake atlas-ban atlas-messages atlas-channel
```

Expected: all three build. (`report/` packages live inside existing service dirs and `libs/atlas-packet`/`libs/atlas-redis` already have COPY lines — no Dockerfile edits anticipated; the bake proves it.)

- [ ] **Step 4: Repo-root guards**

```bash
tools/redis-key-guard.sh
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit operations --check
```

Expected: all exit 0. (Run guard scripts from the repo root WITHOUT a global GOWORK=off prefix.)

- [ ] **Step 5: UI gate**

```bash
cd services/atlas-ui && npx tsc --noEmit && npm run lint && npm run build
```

Expected: clean.

- [ ] **Step 6: Live acceptance (PRD §10) — three tenants**

After deploy (per `live-config-patch.md`). Run 1–4 against a **v83** tenant and again against a **v72** tenant (the lowest full-feature column, and the one sharing no opcode values with v83 — the case a ported-opcode bug would surface in):

1. `/sue <name> <reason>` from a live client → `sue` report row in atlas-ban; reporter sees the success chat line. Sue against a nonexistent id → result 1, nothing persisted.
2. Claim UI opens (status/availability packets); chat-claim submission persists a `claim` report with verbatim client log + server transcript; success notice shows remaining-count text.
3. Claim against an unresolvable name → mode 0x42 notice, nothing persisted.
4. atlas-ui `/reports`: list, filter by status, open detail, PATCH status.

Then against a **v61** tenant (sue-only), confirm the split scope holds:

5. Sue round-trips exactly as in (1) — including the 0x34 result opcode, not 0x37.
6. The claim UI stays disabled (no status packet sent), and the channel logs the writer-not-found skip at debug rather than erroring.

Record observed results in the task folder before invoking `superpowers:finishing-a-development-branch`.

- [ ] **Step 7: Commit**

```bash
git add docs/TODO.md docs/tasks/task-145-player-reports/
git commit -m "docs(task-145): deferrals and acceptance notes"
```

---

## Execution Notes

- **Task order:** 1→2→3 (packet lib) unblock everything; 4→9 (ban) and 10→12 (redis/messages) are independent of each other; 13→17 (channel) need Tasks 1-3 and the Task 4/13 contract; 18-19 (config) need writer/handler names from 1-3; 20→22 (UI) need only Task 9's REST contract; 23→24 (verification) need 1-3 and are serialized on IDA instances; 25 is last.
- **Contract discipline:** the Kafka contract file exists twice (ban + channel modules) with identical constant names and struct shapes — if one side changes, change both in the same commit.
- **Grounding:** every opcode/mode byte in this plan traces to `packet-findings.md` (IDA-verified 2026-07-09) or `docs/packets/audits/STATUS.md`. Do not substitute values from general MapleStory knowledge.
- **Where this plan says "NOTE for the implementer: verify X"**, that verification is part of the task, not optional — the referenced API shapes were read from the worktree on 2026-07-09 but signatures drift.
