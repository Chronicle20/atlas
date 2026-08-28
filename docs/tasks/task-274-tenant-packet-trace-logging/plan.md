# Tenant Packet Trace Logging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a per-tenant `diagnostics.tracePackets` switch that, together with a pod running at `LOG_LEVEL=Debug`, emits a classic hex+ASCII dump of every inbound and outbound packet on `atlas-channel` and `atlas-login`.

**Architecture:** A new boolean lives inside the tenant configuration JSON document and is carried to both socket services by the existing Kafka configuration projection. A new pure package `libs/atlas-socket/trace` owns the formatter and the gate, so both services emit byte-identical output. `libs/atlas-socket` gains one optional `SetPacketTracer` configurator so the library stays free of tenant knowledge; the service-side closure it receives is bound to exactly one tenant. Outbound traces are emitted in `session.Announce` and in `Model.WriteHello`. A prerequisite defect is fixed inside this task: `configuration.PublishSnapshot` is never called from the projection apply loop in either service, so the package-level snapshot is dead in `atlas-channel` and frozen at startup in `atlas-login`.

**Tech Stack:** Go 1.27 (five modules), logrus, `github.com/google/uuid`, GORM + SQLite in-memory for `atlas-configurations` tests, testify; atlas-ui is **Vite + React Router + TypeScript + React Query + react-hook-form + Zod + shadcn/ui**, tested with vitest + @testing-library/react.

**Spec:** `docs/tasks/task-274-tenant-packet-trace-logging/design.md` (PRD at `docs/tasks/task-274-tenant-packet-trace-logging/prd.md`)

## Global Constraints

- The flag's absent value MUST deserialize to `false`. No backfill, no migration. (FR-1.2)
- Two independent conditions are required for any output: the tenant flag AND the pod logger at Debug. (FR-2.2)
- A trace lookup MUST NOT block and MUST NOT return an error into the packet path. `configuration.GetTenantConfig` is unusable here — it calls `waitReady()` with a 60 s `readyTimeout`. (FR-2.4)
- Trace emission MUST NOT alter, reorder, delay, or drop any packet. A panic inside the trace path MUST be recovered and swallowed. (FR-2.5)
- Traced bytes are always **plaintext**: inbound after decryption including the opcode bytes, outbound as produced by the writer *before* the 4-byte header prepend and AES-OFB encryption. (FR-3.2, FR-4.2)
- The header line and the whole body are ONE log call, one multi-line string. (FR-5.7)
- No truncation at any packet size. (FR-5.5)
- Tracing is scoped to the tenant owning the session; tenant B's flag must never trace tenant A's packets. (FR-2.6)
- Direction prefixes are exactly `[PKT IN ]` and `[PKT OUT]` (fixed width, note the space in the inbound form).
- Never land a placeholder comment, a stubbed handler, or an unimplemented status response.
- Check `libs/atlas-constants/` before defining any new domain type, alias, or numeric constant.
- Preserve existing line endings.

---

## Task 1: The shared trace formatter package

### Files

- `libs/atlas-socket/trace/trace.go` — new file; the whole package
- `libs/atlas-socket/trace/trace_test.go` — new file; table tests
- `libs/atlas-socket/go.mod` — read-only; already requires `github.com/google/uuid v1.6.0` and `github.com/sirupsen/logrus v1.10.1`, so no dependency change is needed

Module root for `go build`/`go test`: `libs/atlas-socket`.

Patterns to copy: none needed — this is a self-contained pure package. `libs/atlas-socket/server_test.go:37-73` shows the repo's `logrus.New()` + `nopWriter{}` idiom if a silent logger is wanted; for the `Enabled` tests use `github.com/sirupsen/logrus/hooks/test` as `libs/atlas-opcodes/producer_test.go:24-51` does.

### Interfaces

- Consumes: nothing.
- Produces:
  - `trace.Direction` (`int`) with constants `trace.Inbound` (0) and `trace.Outbound` (1)
  - `type trace.Header struct { Direction Direction; Name string; Op *uint16; OpSize int; Length int; SessionId uuid.UUID }`
  - `func trace.Format(h Header, b []byte) string`
  - `func trace.Dump(b []byte) string`
  - `func trace.Enabled(l logrus.FieldLogger, flag bool) bool`

### Exact rendering rules

**Header line.** Space-separated, in this order:

1. `[PKT IN ]` when `Direction == Inbound`, `[PKT OUT]` when `Outbound`.
2. `handler=<Name>` when `Inbound`, `writer=<Name>` when `Outbound`.
3. `op=n/a` when `Op == nil`; otherwise `op=0x%02x` when `OpSize == 1`, `op=0x%04x` for every other `OpSize`.
4. `len=%d` from `Header.Length` (decimal).
5. `session=%s` — the full `SessionId.String()`, never abbreviated.

**Body.** For each 16-byte chunk at offset `o`:

- `fmt.Sprintf("%04x  ", o)` — 4 lowercase hex digits then two spaces.
- Then, for `i` in `0..15`: if `i == 8`, write one extra space first; then write `fmt.Sprintf("%02x ", chunk[i])` if `i < len(chunk)`, else write three spaces. This region is always exactly 49 characters wide, so a short final line stays column-aligned (FR-5.4).
- Then one space, then `|`, then the ASCII gutter (one char per byte actually present: the byte itself when `0x20 <= b <= 0x7e`, otherwise `.`), then `|`.

Lines are joined with `\n`. There is no trailing newline. `Dump(nil)` and `Dump([]byte{})` return `""`.

`Format` returns the header line alone when `len(b) == 0` (FR-5.6); otherwise the header line, `"\n"`, then `Dump(b)`. One string, no trailing newline.

Build the body with a single `strings.Builder`, pre-sized via `b.Grow((len(bs)/16+1) * 78)`, rather than concatenation.

**`Enabled`.** Check `flag` first and return `false` immediately when it is false — this is the hot path and must not touch the logger (FR-2.1). Then attempt `l.(interface{ IsLevelEnabled(logrus.Level) bool })`; when the assertion succeeds return `lc.IsLevelEnabled(logrus.DebugLevel)`; when it fails return `true` (best effort — logrus discards the entry itself).

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-socket/trace/trace_test.go`, `package trace`. Table-driven. The expected strings below are exact and MUST be asserted byte for byte.

`TestDump` — `cases := []struct{ name string; in []byte; want string }`:

| subtest name | input | want |
|---|---|---|
| `empty` | `[]byte{}` | `""` |
| `nil` | `nil` | `""` |
| `single byte` | `[]byte{0x41}` | `"0000  41                                                |A|"` |
| `non printable` | `[]byte{0x00, 0x1f, 0x7f, 0xff}` | `"0000  00 1f 7f ff                                       |....|"` |
| `fifteen bytes` | `[]byte{0x41,0x42,0x43,0x44,0x45,0x46,0x47,0x48,0x49,0x4a,0x4b,0x4c,0x4d,0x4e,0x4f}` | `"0000  41 42 43 44 45 46 47 48  49 4a 4b 4c 4d 4e 4f     |ABCDEFGHIJKLMNO|"` |
| `sixteen aligned` | `[]byte{0x00,0x01,0x02,0x03,0x04,0x05,0x06,0x07,0x08,0x09,0x0a,0x0b,0x0c,0x0d,0x0e,0x0f}` | `"0000  00 01 02 03 04 05 06 07  08 09 0a 0b 0c 0d 0e 0f  |................|"` |
| `two lines with short tail` | `[]byte{0x7d,0x00,0x01,0x00,0x00,0x00,0xff,0xff,0xff,0xff,0x01,0x05,0x00,0x4d,0x61,0x70,0x6c,0x65,0x00}` | `"0000  7d 00 01 00 00 00 ff ff  ff ff 01 05 00 4d 61 70  |}............Map|\n0010  6c 65 00                                          |le.|"` |

`TestDump_LargePayloadIsNotTruncated` — build `bs := bytes.Repeat([]byte{0x41}, 4100)`; assert `strings.Count(Dump(bs), "\n") == 256` (257 lines), assert the last line begins with `"1000  41 41 41 41"`, and assert `!strings.Contains(Dump(bs), "truncated")`.

`TestFormat` — `op := uint16(0x007d)`, `sid := uuid.MustParse("3f2a1c88-0000-4000-8000-000000000001")`:

| subtest name | header | body | want |
|---|---|---|---|
| `outbound with body` | `Header{Outbound, "CHARACTER_DATA", &op, 2, 19, sid}` | the 19-byte `two lines with short tail` input above | `"[PKT OUT] writer=CHARACTER_DATA op=0x007d len=19 session=3f2a1c88-0000-4000-8000-000000000001\n"` + the two-line dump above |
| `inbound byte opcode` | `Header{Inbound, "USER_CHAT", &op, 1, 3, sid}` | `[]byte{0x7d,0x00,0x01}` | header is `"[PKT IN ] handler=USER_CHAT op=0x7d len=3 session=3f2a1c88-0000-4000-8000-000000000001"`, followed by `"\n0000  7d 00 01                                          |}..|"` |
| `nil opcode` | `Header{Outbound, "<hello>", nil, 2, 0, sid}` | `[]byte{}` | `"[PKT OUT] writer=<hello> op=n/a len=0 session=3f2a1c88-0000-4000-8000-000000000001"` — exactly, with no `\n` |
| `unresolved handler name` | `Header{Inbound, "<none>", &op, 2, 0, sid}` | `nil` | `"[PKT IN ] handler=<none> op=0x007d len=0 session=3f2a1c88-0000-4000-8000-000000000001"` |

`TestFormat_IsASingleString` — for the `outbound with body` case assert `strings.Contains(got, "\n")` and that `got` does not end in `"\n"`.

`TestEnabled` — four subtests:

| subtest name | logger | flag | want |
|---|---|---|---|
| `flag off short circuits` | a `*logrus.Logger` at `logrus.DebugLevel` | `false` | `false` |
| `flag on at info level` | `logrus.New()` with `SetLevel(logrus.InfoLevel)` | `true` | `false` |
| `flag on at debug level` | `logrus.New()` with `SetLevel(logrus.DebugLevel)` | `true` | `true` |
| `logger without level probe` | a local `type levellessLogger struct{ logrus.FieldLogger }` wrapping a null logger (so the concrete type no longer exposes `IsLevelEnabled`) | `true` | `true` |

- [ ] **Step 2: Run the tests to verify they fail**

Run from `libs/atlas-socket`: `go test ./trace/...`
Expected: FAIL — `no Go files` / undefined `Dump`, `Format`, `Enabled`, `Header`.

- [ ] **Step 3: Write `libs/atlas-socket/trace/trace.go`**

Implement exactly the rules stated above. Doc comments must record *why*: `Format` returns one string because concurrent sessions on a pod would otherwise interleave bodies (FR-5.7); `Op` is a pointer because the `WriteHello` handshake has no opcode (FR-4.3); `Enabled` checks the flag before touching the logger because the off path must not allocate (FR-2.1).

- [ ] **Step 4: Run the tests to verify they pass**

Run from `libs/atlas-socket`: `go test ./trace/... -run 'TestDump|TestFormat|TestEnabled' -v`
Expected: PASS, all subtests.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-socket/trace/trace.go libs/atlas-socket/trace/trace_test.go
git commit -m "feat(atlas-socket): add shared packet trace formatter"
```

---

## Task 2: The `libs/atlas-socket` tracer seam

### Files

- `libs/atlas-socket/server.go` — add `tracer PacketTracer` to the unexported `config` struct (declared at lines 87-97) and the tracer call inside `handle` (lines 325-334)
- `libs/atlas-socket/opts.go` — add the `PacketTracer` type and the `SetPacketTracer` configurator
- `libs/atlas-socket/server_test.go` — add the tracer tests

Module root: `libs/atlas-socket`.

Patterns to copy: `libs/atlas-socket/opts.go:55-59` (`SetHandlers` — the exact configurator shape). For the test, `libs/atlas-socket/server_test.go:37-73` builds `cfg := &config{...}` as a struct literal in an internal (`package socket`) test; the new tests call `handle(l)(cfg, sessionId, p)` directly with that same literal and need none of the `net.Pipe` machinery.

### Interfaces

- Consumes: nothing from Task 1.
- Produces:
  - `type socket.PacketTracer func(sessionId uuid.UUID, op uint16, payload []byte)`
  - `func socket.SetPacketTracer(tracer PacketTracer) Configurator`
  - `config.tracer` — unexported; only Task 7a/8a set it, and only via the configurator.

### Current code being changed

`libs/atlas-socket/server.go:325-334` today:

```go
func handle(l logrus.FieldLogger) func(config *config, sessionId uuid.UUID, p request.Request) {
	return func(config *config, sessionId uuid.UUID, p request.Request) {
		reader := request.NewRequestReader(&p, time.Now().Unix())
		op := config.rw.Read(&reader)
		if h, ok := config.handlers[op]; ok {
			h(sessionId, reader)
		} else {
			l.Infof("Read a unhandled message with op 0x%02X.", op&0xFF)
		}
	}
}
```

`request.Request` is `type Request []byte` (`libs/atlas-socket/request/request.go:13`), so `p` is already the full decrypted plaintext frame including the opcode bytes and can be handed to the tracer directly (FR-3.2).

- [ ] **Step 1: Write the failing tests**

Append to `libs/atlas-socket/server_test.go` (it is `package socket`, so `config` and `handle` are reachable):

`TestHandle_NilTracerIsSafe` — build `cfg := &config{rw: ShortReadWriter{}, handlers: map[uint16]request.Handler{0x0001: func(uuid.UUID, request.Reader) { called = true }}}` with `cfg.tracer` left nil; call `handle(l)(cfg, uuid.New(), request.Request{0x01, 0x00, 0xaa})`; assert no panic and `called == true`.

`TestHandle_TracerSeesFullFrameBeforeHandler` — table with two cases:

| case | handlers map | frame | expect tracer op | expect tracer payload | expect handler ran |
|---|---|---|---|---|---|
| registered handler | `{0x0001: h}` | `request.Request{0x01, 0x00, 0xaa, 0xbb}` | `0x0001` | `[]byte{0x01, 0x00, 0xaa, 0xbb}` | true |
| unregistered opcode (FR-3.4) | `{}` (empty) | `request.Request{0xff, 0x00, 0x11}` | `0x00ff` | `[]byte{0xff, 0x00, 0x11}` | false |

Record ordering with a shared `var order []string`: the tracer appends `"trace"`, the handler appends `"handle"`. For the registered case assert `order == []string{"trace", "handle"}` — the trace must precede the handler (FR-3.3). Also assert the tracer received the same `sessionId` that was passed to `handle`.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `libs/atlas-socket`: `go test ./... -run TestHandle`
Expected: FAIL to compile — `cfg.tracer` undefined.

- [ ] **Step 3: Implement the seam**

In `libs/atlas-socket/opts.go`, after `SetHandlers`:

```go
// PacketTracer receives every inbound frame after decryption and after the
// opcode is parsed, but before dispatch. Optional: nil means no tracing,
// and the nil check is the entire cost when tracing is off (FR-2.1). The
// library deliberately knows nothing about tenants -- the closure the
// service installs owns the tenant, the handler-name map, and the flag.
type PacketTracer func(sessionId uuid.UUID, op uint16, payload []byte)

func SetPacketTracer(tracer PacketTracer) Configurator {
	return func(s *config) {
		s.tracer = tracer
	}
}
```

In `libs/atlas-socket/server.go`, add `tracer PacketTracer` as the last field of the `config` struct, and insert the call into `handle` immediately after `op := config.rw.Read(&reader)`:

```go
		if config.tracer != nil {
			config.tracer(sessionId, op, p)
		}
```

The existing unhandled-opcode `l.Infof` stays exactly as it is; the trace is additive.

- [ ] **Step 4: Run the tests to verify they pass**

Run from `libs/atlas-socket`: `go test ./... -race`
Expected: PASS, including the pre-existing `Serve`/`run` tests.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-socket/opts.go libs/atlas-socket/server.go libs/atlas-socket/server_test.go
git commit -m "feat(atlas-socket): add optional SetPacketTracer inbound seam"
```

---

## Task 3: `libs/atlas-opcodes` helpers

### Files

- `libs/atlas-opcodes/producer.go` — add `BuildHandlerNames` next to `BuildHandlerMap` (which lives at lines 58-87)
- `libs/atlas-opcodes/width.go` — new file; `OpCodeSize` and `OpReadWriterFor`
- `libs/atlas-opcodes/producer_test.go` — add `BuildHandlerNames` tests
- `libs/atlas-opcodes/width_test.go` — new file
- `libs/atlas-opcodes/config.go` — read-only; `ServiceLogin = "login"` and `ServiceChannel = "channel"` at lines 6-7, `HandlerConfig` at lines 11-21, `appliesToService` at lines 36-46
- `libs/atlas-opcodes/go.mod` — read-only; already requires and `replace`s `github.com/Chronicle20/atlas/libs/atlas-socket` (lines 6 and 17), so `OpReadWriterFor` needs no new dependency

Module root: `libs/atlas-opcodes`.

Patterns to copy: `libs/atlas-opcodes/producer.go` lines 58-87 (`BuildHandlerMap` — same service filter, same `strconv.ParseUint(hc.OpCode, 0, 16)` parse at line 77, same warn-and-continue on a bad opcode). For tests, `libs/atlas-opcodes/producer_test.go:55-69` and `:89-102` (`test.NewNullLogger()` + `[]HandlerConfig{...}` + assert on map and log entries), with helpers `warnContaining` at `:24-41` and `warnCount` at `:43-51`.

### Interfaces

- Consumes: nothing.
- Produces:
  - `func opcodes.BuildHandlerNames(l logrus.FieldLogger, service string, handlers []HandlerConfig) map[uint16]string`
  - `func opcodes.OpCodeSize(region string, majorVersion uint16) int`
  - `func opcodes.OpReadWriterFor(region string, majorVersion uint16) socket.OpReadWriter`

`OpCodeSize` returns `1` when `region == "GMS" && majorVersion <= 28`, otherwise `2`. `OpReadWriterFor` returns `socket.ByteReadWriter{}` for size 1 and `socket.ShortReadWriter{}` otherwise. These two functions are the single source of a rule currently duplicated verbatim at `services/atlas-channel/atlas.com/channel/main.go:429-431` and `services/atlas-login/atlas.com/login/main.go:272-274`; Tasks 7a and 8a delete those copies.

- [ ] **Step 1: Write the failing tests**

In `libs/atlas-opcodes/producer_test.go` (package matches the existing file):

`TestBuildHandlerNames` — table over `[]HandlerConfig`, calling `BuildHandlerNames(l, ServiceChannel, cfgs)`:

| subtest name | input handlers | want map |
|---|---|---|
| `maps opcode to configured name` | `{OpCode: "0x0001", Handler: "USER_CHAT"}`, `{OpCode: "0x00FF", Handler: "PONG"}` | `{0x0001: "USER_CHAT", 0x00ff: "PONG"}` |
| `includes entries with no registered handler` | `{OpCode: "0x0002", Handler: "NOT_REGISTERED"}` | `{0x0002: "NOT_REGISTERED"}` — this is the point of the function: the trace path must name an opcode even when nothing handles it (FR-3.4) |
| `skips other service entries` | `{OpCode: "0x0003", Handler: "LOGIN_ONLY", Services: []string{ServiceLogin}}` | `{}` (empty, non-nil) |
| `includes untagged entries` | `{OpCode: "0x0004", Handler: "SHARED"}` (no `Services`) | `{0x0004: "SHARED"}` |
| `skips unparsable opcode` | `{OpCode: "not-hex", Handler: "BROKEN"}` | `{}` (empty, non-nil) |

For the last case additionally assert `warnContaining(h, "BROKEN", "not-hex")` is true. For the `skips other service entries` case assert `warnCount(h) == 0` — a cross-service entry is normal, not a defect.

Create `libs/atlas-opcodes/width_test.go`:

`TestOpCodeSize` and `TestOpReadWriterFor` over one shared table:

| subtest name | region | majorVersion | want size | want type |
|---|---|---|---|---|
| `gms v28 is byte` | `"GMS"` | `28` | `1` | `socket.ByteReadWriter{}` |
| `gms v27 is byte` | `"GMS"` | `27` | `1` | `socket.ByteReadWriter{}` |
| `gms v29 is short` | `"GMS"` | `29` | `2` | `socket.ShortReadWriter{}` |
| `gms v83 is short` | `"GMS"` | `83` | `2` | `socket.ShortReadWriter{}` |
| `jms v185 is short` | `"JMS"` | `185` | `2` | `socket.ShortReadWriter{}` |
| `jms v28 is short` | `"JMS"` | `28` | `2` | `socket.ShortReadWriter{}` — the byte rule is GMS-only |

Assert the read-writer type with `require.IsType`.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `libs/atlas-opcodes`: `go test ./...`
Expected: FAIL to compile — undefined `BuildHandlerNames`, `OpCodeSize`, `OpReadWriterFor`.

- [ ] **Step 3: Implement**

Append to `libs/atlas-opcodes/producer.go`:

```go
// BuildHandlerNames returns the op -> configured handler name map for this
// service's slice of the tenant socket config. Same service filtering and
// opcode parsing as BuildHandlerMap, but deliberately separate: the packet
// trace must be able to name an opcode whose handler is not registered on
// this pod, which is exactly the case BuildHandlerMap drops (FR-3.4).
func BuildHandlerNames(l logrus.FieldLogger, service string, handlers []HandlerConfig) map[uint16]string {
	result := make(map[uint16]string)
	for _, hc := range handlers {
		if !appliesToService(hc.Services, service) {
			continue
		}
		op, err := strconv.ParseUint(hc.OpCode, 0, 16)
		if err != nil {
			l.WithError(err).Warnf("Unable to configure handler [%s] for opcode [%s].", hc.Handler, hc.OpCode)
			continue
		}
		result[uint16(op)] = hc.Handler
	}
	return result
}
```

Create `libs/atlas-opcodes/width.go`:

```go
package opcodes

// OpCodeSize returns the wire width in bytes of this tenant's opcodes.
// This rule was duplicated verbatim in atlas-channel/main.go and
// atlas-login/main.go; the packet tracer would have been a third copy, so
// it lives here now and both main.go files call OpReadWriterFor.
func OpCodeSize(region string, majorVersion uint16) int {
	if region == "GMS" && majorVersion <= 28 {
		return 1
	}
	return 2
}

// OpReadWriterFor returns the OpReadWriter matching OpCodeSize.
func OpReadWriterFor(region string, majorVersion uint16) socket.OpReadWriter {
	if OpCodeSize(region, majorVersion) == 1 {
		return socket.ByteReadWriter{}
	}
	return socket.ShortReadWriter{}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run from `libs/atlas-opcodes`: `go test ./... -race -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-opcodes/producer.go libs/atlas-opcodes/producer_test.go libs/atlas-opcodes/width.go libs/atlas-opcodes/width_test.go
git commit -m "feat(atlas-opcodes): add BuildHandlerNames and the opcode-width helpers"
```

---

## Task 4: The `diagnostics` field in `atlas-configurations`

### Files

- `services/atlas-configurations/atlas.com/configurations/tenants/diagnostics/rest.go` — new file
- `services/atlas-configurations/atlas.com/configurations/tenants/rest.go` — add the field to `RestModel` (declared at lines 12-31)
- `services/atlas-configurations/atlas.com/configurations/tenants/rest_test.go` — add the round-trip test
- `services/atlas-configurations/atlas.com/configurations/tenants/processor_test.go` — add the outbox-payload test
- `services/atlas-configurations/docs/rest.md` — add `diagnostics (object)` to both tenant field lists (the existing `cashShop (object)` lines are at 98 and 233)
- `services/atlas-configurations/atlas.com/configurations/tenants/processor.go` — read-only; `UpdateById` (`:124-176`) marshals the whole `RestModel` at `:147` and enqueues `sanitized := input` at `:172-174`, and `Create` (`:187-238`) does the same at `:193`/`:230-232`. **No change is needed here** — the new field is carried by construction.

Module root: `services/atlas-configurations/atlas.com/configurations`.

Patterns to copy:
- `services/atlas-configurations/atlas.com/configurations/tenants/cashshop/rest.go:1-11` (a small sibling package holding one `RestModel`)
- `services/atlas-configurations/atlas.com/configurations/tenants/rest_test.go:14-64` (`TestTenantRestModelCarriesMapleLife` — two subtests, "document with the key" and "document without the key", asserting a nested sub-object round-trips and an absent key yields the zero value). This is the exact template.
- `services/atlas-configurations/atlas.com/configurations/tenants/processor_test.go:757-786` (`TestProcessor_UpdateById_OutboxMessageNeverCarriesClientSuppliedEnvironment`) plus its helpers `outboxTenantEnvelope` at `:696-703` and `latestOutboxTenantEnvelope` at `:707-718`, `setupTestDB` at `:53-79`, `testLogger` at `:81-85`, `createTestRestModel` at `:90-96`.

### Interfaces

- Consumes: nothing.
- Produces: `diagnostics.RestModel` with the single field `TracePackets bool` and json tag `tracePackets`; `tenants.RestModel.Diagnostics` with json tag `diagnostics`.

### Current shape

`services/atlas-configurations/atlas.com/configurations/tenants/rest.go:12-31` today ends its sub-object block with:

```go
	CashShop     cashshop.RestModel   `json:"cashShop"`
	MapleLife    maplelife.RestModel  `json:"mapleLife"`
```

The new field follows the same non-pointer, no-`omitempty` convention.

- [ ] **Step 1: Write the failing tests**

In `services/atlas-configurations/atlas.com/configurations/tenants/rest_test.go`, add `TestTenantRestModelCarriesDiagnostics` with three subtests, copying the structure of `TestTenantRestModelCarriesMapleLife` at `:14-64`:

| subtest name | input JSON `attributes` | assert |
|---|---|---|
| `document with diagnostics tracePackets true` | `{"region":"GMS","majorVersion":83,"minorVersion":1,"diagnostics":{"tracePackets":true}}` | after `json.Unmarshal`, `rm.Diagnostics.TracePackets == true`; after re-`json.Marshal`, the output contains `"diagnostics":{"tracePackets":true}` |
| `document with no diagnostics key` | `{"region":"GMS","majorVersion":83,"minorVersion":1}` | `rm.Diagnostics.TracePackets == false` (FR-1.2) |
| `document with empty diagnostics object` | `{"region":"GMS","majorVersion":83,"minorVersion":1,"diagnostics":{}}` | `rm.Diagnostics.TracePackets == false` |

In `services/atlas-configurations/atlas.com/configurations/tenants/processor_test.go`, add `TestProcessor_UpdateById_OutboxMessageCarriesDiagnostics`, copying `:757-786` exactly for the DB setup, the `UpdateById` call, and the `latestOutboxTenantEnvelope(t, db, topic)` decode. Difference: seed the tenant via `createTestRestModel(...)`, then call `UpdateById` with an input whose `Diagnostics.TracePackets` is `true`, and assert the decoded envelope's tenant attributes carry `Diagnostics.TracePackets == true` (FR-1.5). Extend the local `outboxTenantEnvelope` decode struct at `:696-703` with the `diagnostics` field if it decodes into a narrowed shape rather than `tenants.RestModel`.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-configurations/atlas.com/configurations`: `go test ./tenants/... -run 'Diagnostics' -v`
Expected: FAIL to compile — `rm.Diagnostics` undefined.

- [ ] **Step 3: Implement**

Create `services/atlas-configurations/atlas.com/configurations/tenants/diagnostics/rest.go`:

```go
package diagnostics

// RestModel carries per-tenant operational diagnostics switches. Every
// field must be zero-value safe: a tenant document written before this
// object existed unmarshals to the zero value, which is "off" (FR-1.2),
// so no backfill and no migration is required.
//
// TracePackets is deliberately dangerous -- with it on, and the serving
// pod at LOG_LEVEL=Debug, login-family packets put account passwords,
// PICs/PINs and HWIDs into the log stream in plaintext. Logs captured
// while it is on are credential-bearing material.
type RestModel struct {
	TracePackets bool `json:"tracePackets"`
}
```

In `tenants/rest.go`, import `atlas-configurations/tenants/diagnostics` and add, after `MapleLife`:

```go
	Diagnostics  diagnostics.RestModel `json:"diagnostics"`
```

Add `- \`diagnostics\` (object)` to `services/atlas-configurations/docs/rest.md` immediately after each of the two `- \`cashShop\` (object)` lines (98 and 233), with a following line noting it currently holds only `tracePackets` (boolean, default `false`) and that enabling it writes plaintext credentials to the serving pod's logs.

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-configurations/atlas.com/configurations`: `go test ./tenants/... -race`
Expected: PASS, including the pre-existing tenant processor and rest tests.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/tenants/diagnostics/rest.go \
        services/atlas-configurations/atlas.com/configurations/tenants/rest.go \
        services/atlas-configurations/atlas.com/configurations/tenants/rest_test.go \
        services/atlas-configurations/atlas.com/configurations/tenants/processor_test.go \
        services/atlas-configurations/docs/rest.md
git commit -m "feat(configurations): add tenant diagnostics.tracePackets"
```

---

## Task 5: atlas-channel — mirror, live snapshot, non-blocking accessor

### Files

- `services/atlas-channel/atlas.com/channel/configuration/tenant/diagnostics/rest.go` — new file; the mirror
- `services/atlas-channel/atlas.com/channel/configuration/tenant/rest.go` — add the field to `RestModel` (lines 10-20)
- `services/atlas-channel/atlas.com/channel/configuration/registry.go` — add `TracePacketsEnabled`
- `services/atlas-channel/atlas.com/channel/configuration/projection/loop.go` — add the `PublishSnapshot` call in the tick body (the snapshot is taken at line 86)
- `services/atlas-channel/atlas.com/channel/configuration/registry_test.go` — add the accessor tests
- `services/atlas-channel/atlas.com/channel/configuration/projection/projection_test.go` — add the no-ops test

Module root: `services/atlas-channel/atlas.com/channel`.

Patterns to copy:
- The `diagnostics` package Task 4 creates under `atlas-configurations` (new file there) — this mirror is the same struct with the same json tag.
- `services/atlas-channel/atlas.com/channel/configuration/registry.go:89-107` (`PublishSnapshot` — shows `configMu`, `tenantConfig`, `serviceConfig`, `readyOnce`), and `:68-79` (`GetTenantConfig` — the blocking version the accessor must NOT imitate).
- `services/atlas-channel/atlas.com/channel/configuration/registry_test.go:14-43` (`TestGetServiceConfig_BlocksUntilPublishSnapshot` — the `done chan result` + `select`/`time.After` idiom).
- `services/atlas-channel/atlas.com/channel/configuration/projection/projection_test.go:92-149` (`TestComputeOps_AddRemovePortChangeUnchanged` — builds a `map[uuid.UUID]tenant.RestModel` literal and a `mk(port int) *configuration.RestModel` closure, then asserts op counts and kinds).

### Interfaces

- Consumes: nothing from Tasks 1-3. Mirrors the JSON shape produced by Task 4.
- Produces: `func configuration.TracePacketsEnabled(tenantId uuid.UUID) bool` — consumed by Tasks 7a and 7b.

### Why the loop change is required

`services/atlas-channel/atlas.com/channel/configuration/projection/loop.go` already imports `"atlas-channel/configuration"` (line 4) but **only for the `*configuration.RestModel` type** — `PublishSnapshot` has no production caller anywhere in `atlas-channel`. `serviceConfig` is therefore permanently nil, `readyCh` is never closed, and every existing `GetServiceConfig()` caller (e.g. `session/task.go`) blocks the full 60 s `readyTimeout` and then returns `ErrNotReady`. `registry.go:81-88`'s own doc comment already states the intended contract ("and again from the projection apply loop on each observed change"); this task writes the call site that was never written. Without it FR-2.3 cannot hold.

- [ ] **Step 1: Write the failing tests**

In `services/atlas-channel/atlas.com/channel/configuration/registry_test.go` (`package configuration_test`), add `TestTracePacketsEnabled`. Because `PublishSnapshot` mutates package state, write it as ONE test function with sequential phases rather than parallel subtests:

| phase | action | assert |
|---|---|---|
| 1 — before any snapshot | call `configuration.TracePacketsEnabled(uuid.New())` inside a goroutine writing to a buffered channel | the value arrives within `50 * time.Millisecond` and is `false`; a `time.After(50*time.Millisecond)` branch must `t.Fatal("TracePacketsEnabled blocked")` (FR-2.4) |
| 2 — tenant absent from snapshot | `configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{tenantA: {}})` then query `uuid.New()` | `false` |
| 3 — flag on for A, off for B | publish `{tenantA: {Diagnostics: diagnostics.RestModel{TracePackets: true}}, tenantB: {}}` | `TracePacketsEnabled(tenantA) == true` and `TracePacketsEnabled(tenantB) == false` (FR-2.6) |
| 4 — live change | publish again with `tenantA` set to `TracePackets: false` | `TracePacketsEnabled(tenantA) == false` (FR-2.3) |

Note in a comment that phase 1 must run first because `PublishSnapshot` closes `readyCh` irreversibly, and that this file already contains `TestGetServiceConfig_BlocksUntilPublishSnapshot`, which also publishes — so the new test must tolerate running in either order by never asserting that `TracePacketsEnabled` blocks, only that it returns promptly.

In `services/atlas-channel/atlas.com/channel/configuration/projection/projection_test.go`, add `TestComputeOps_DiagnosticsOnlyChangeEmitsNoOps` (FR-1.6): build `prevTenants` and `nextTenants` as two `map[uuid.UUID]tenant.RestModel` literals for the same tenant id, identical in `Region`, `MajorVersion`, `MinorVersion`, `Socket` and `Worlds`, differing **only** in `Diagnostics.TracePackets` (`false` → `true`); use the same `*configuration.RestModel` value for both `prevSvc` and `nextSvc`. Assert `require.Empty(t, ComputeOps(svc, prevTenants, svc, nextTenants))` — flipping the switch on a live tenant must not drain or re-add a single listener.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-channel/atlas.com/channel`: `go test ./configuration/... -run 'TracePackets|DiagnosticsOnly' -v`
Expected: FAIL to compile — undefined `configuration.TracePacketsEnabled` and `tenant.RestModel.Diagnostics`.

- [ ] **Step 3: Implement**

Create `services/atlas-channel/atlas.com/channel/configuration/tenant/diagnostics/rest.go` — same package name, same struct and json tag as Task 4's, with a doc comment noting it is the projection's deserialization mirror of `atlas-configurations`' tenant document.

In `services/atlas-channel/atlas.com/channel/configuration/tenant/rest.go`, import `atlas-channel/configuration/tenant/diagnostics` and add after `Worlds`:

```go
	Diagnostics  diagnostics.RestModel `json:"diagnostics"`
```

In `services/atlas-channel/atlas.com/channel/configuration/registry.go`, after `GetTenantConfig`:

```go
// TracePacketsEnabled reports whether the tenant has packet trace logging
// switched on. Unlike GetTenantConfig it NEVER blocks and never returns an
// error: this runs on the socket send and receive paths, where a 60-second
// waitReady would be a hang, not a diagnostic (FR-2.4). A snapshot that has
// not been published yet, and a tenant absent from the snapshot, both mean
// "off".
func TracePacketsEnabled(tenantId uuid.UUID) bool {
	configMu.RLock()
	defer configMu.RUnlock()
	tc, ok := tenantConfig[tenantId]
	if !ok {
		return false
	}
	return tc.Diagnostics.TracePackets
}
```

In `services/atlas-channel/atlas.com/channel/configuration/projection/loop.go`, insert immediately after the `nextSvc, nextTenants := a.State.Snapshot()` line (currently line 86) and before `ops := ComputeOps(...)`:

```go
			// Keep the package-level configuration snapshot in step with the
			// projection on every tick. Without this atlas-channel never
			// populates it at all: PublishSnapshot has no other production
			// caller here, so serviceConfig stays nil and readyCh is never
			// closed. registry.go's own doc comment already promises this
			// call site; it was never written. TracePacketsEnabled reads
			// from here, so FR-2.3's "takes effect on the next packet" is
			// bounded by one tick (250ms) plus Kafka propagation.
			configuration.PublishSnapshot(nextSvc, nextTenants)
```

Ordering is safe: the loop runs only after `WaitCaughtUp`, and no listener exists — so no packet can arrive — until the same tick's first `OpAdd` executes, which is after this publish.

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-channel/atlas.com/channel`: `go test ./configuration/... -race`
Expected: PASS, including the pre-existing `TestGetServiceConfig_BlocksUntilPublishSnapshot` and `TestComputeOps_AddRemovePortChangeUnchanged`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/configuration/
git commit -m "feat(channel): mirror diagnostics config and publish the projection snapshot each tick"
```

---

## Task 6: atlas-login — mirror, live snapshot, non-blocking accessor

This is Task 5 applied to `atlas-login`. The code is repeated in full below because tasks may be read out of order and the two services' files differ in line numbers and, for `loop.go`, in surrounding structure.

### Files

- `services/atlas-login/atlas.com/login/configuration/tenant/diagnostics/rest.go` — new file; the mirror
- `services/atlas-login/atlas.com/login/configuration/tenant/rest.go` — add the field to `RestModel` (lines 10-20)
- `services/atlas-login/atlas.com/login/configuration/registry.go` — add `TracePacketsEnabled` (after `GetTenantConfig` at `:67-78`)
- `services/atlas-login/atlas.com/login/configuration/projection/loop.go` — add the `PublishSnapshot` call (the snapshot is taken at line 66)
- `services/atlas-login/atlas.com/login/configuration/registry_test.go` — add the accessor tests
- `services/atlas-login/atlas.com/login/configuration/projection/projection_test.go` — add the no-ops test

Module root: `services/atlas-login/atlas.com/login`.

Patterns to copy:
- `services/atlas-login/atlas.com/login/configuration/registry.go:88-106` (`PublishSnapshot`) and `:67-78` (`GetTenantConfig`).
- `services/atlas-login/atlas.com/login/configuration/registry_test.go:1-45` (`TestGetServiceConfig_BlocksUntilPublishSnapshot`).
- `services/atlas-login/atlas.com/login/configuration/projection/projection_test.go:84-135` (`TestComputeOps_AddRemovePortChangeUnchanged`).

Note: login's `ListenerConfig` (`services/atlas-login/atlas.com/login/configuration/projection/apply.go:25-30`) has `Port, Region, MajorVersion, MinorVersion` and **no `IPAddress`**, unlike channel's — the ComputeOps test fixture differs accordingly.

### Interfaces

- Consumes: nothing.
- Produces: `func configuration.TracePacketsEnabled(tenantId uuid.UUID) bool` in the `atlas-login` module — consumed by Tasks 8a and 8b.

### Why the loop change is required here too

`atlas-login`'s `PublishSnapshot` is called exactly once, from `main.go:100`, immediately after catch-up. The snapshot is therefore **frozen at startup**: a later configuration change never reaches `GetTenantConfig` callers. Adding the apply-loop call makes it live. `main.go:100`'s one-shot call must stay — it runs before the apply loop starts and is what the account-session consumer and the `accept_tos` handler rely on today.

- [ ] **Step 1: Write the failing tests**

In `services/atlas-login/atlas.com/login/configuration/registry_test.go`, add `TestTracePacketsEnabled` with the same four sequential phases as Task 5 Step 1:

| phase | action | assert |
|---|---|---|
| 1 — before any snapshot | call `configuration.TracePacketsEnabled(uuid.New())` in a goroutine writing to a buffered channel | returns within `50 * time.Millisecond` with `false`; a `time.After` branch calls `t.Fatal("TracePacketsEnabled blocked")` (FR-2.4) |
| 2 — tenant absent | publish `{tenantA: {}}`, query `uuid.New()` | `false` |
| 3 — per-tenant scoping | publish `{tenantA: {Diagnostics: diagnostics.RestModel{TracePackets: true}}, tenantB: {}}` | `true` for A, `false` for B (FR-2.6) |
| 4 — live change | republish with A's flag `false` | `false` (FR-2.3) |

In `services/atlas-login/atlas.com/login/configuration/projection/projection_test.go`, add `TestComputeOps_DiagnosticsOnlyChangeEmitsNoOps`: two `map[uuid.UUID]tenant.RestModel` literals for the same tenant id, identical in every field except `Diagnostics.TracePackets` (`false` → `true`), the same `*configuration.RestModel` for both service snapshots, and `require.Empty(t, ComputeOps(svc, prevTenants, svc, nextTenants))` (FR-1.6).

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-login/atlas.com/login`: `go test ./configuration/... -run 'TracePackets|DiagnosticsOnly' -v`
Expected: FAIL to compile — undefined `configuration.TracePacketsEnabled` and `tenant.RestModel.Diagnostics`.

- [ ] **Step 3: Implement**

Create `services/atlas-login/atlas.com/login/configuration/tenant/diagnostics/rest.go`:

```go
package diagnostics

// RestModel mirrors the diagnostics sub-object of the tenant configuration
// document owned by atlas-configurations. Zero value is "off", so a tenant
// document written before the object existed traces nothing (FR-1.2).
type RestModel struct {
	TracePackets bool `json:"tracePackets"`
}
```

In `services/atlas-login/atlas.com/login/configuration/tenant/rest.go`, import `atlas-login/configuration/tenant/diagnostics` and add after `Worlds`:

```go
	Diagnostics  diagnostics.RestModel `json:"diagnostics"`
```

In `services/atlas-login/atlas.com/login/configuration/registry.go`, after `GetTenantConfig`:

```go
// TracePacketsEnabled reports whether the tenant has packet trace logging
// switched on. Unlike GetTenantConfig it NEVER blocks and never returns an
// error: this runs on the socket send and receive paths, where a 60-second
// waitReady would be a hang, not a diagnostic (FR-2.4). A snapshot that has
// not been published yet, and a tenant absent from the snapshot, both mean
// "off".
func TracePacketsEnabled(tenantId uuid.UUID) bool {
	configMu.RLock()
	defer configMu.RUnlock()
	tc, ok := tenantConfig[tenantId]
	if !ok {
		return false
	}
	return tc.Diagnostics.TracePackets
}
```

In `services/atlas-login/atlas.com/login/configuration/projection/loop.go`, insert immediately after `nextSvc, nextTenants := a.State.Snapshot()` (currently line 66) and before `ops := ComputeOps(...)`:

```go
			// Keep the package-level configuration snapshot in step with the
			// projection on every tick. main.go publishes once after
			// catch-up, which freezes the snapshot at startup -- a later
			// config change is invisible to GetTenantConfig callers and to
			// TracePacketsEnabled. registry.go's doc comment already
			// promises this call site; it was never written.
			configuration.PublishSnapshot(nextSvc, nextTenants)
```

Leave `main.go:100`'s existing one-shot `publishSnapshot()` in place.

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-login/atlas.com/login`: `go test ./configuration/... -race`
Expected: PASS, including the pre-existing tests.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-login/atlas.com/login/configuration/
git commit -m "feat(login): mirror diagnostics config and publish the projection snapshot each tick"
```

---

## Task 7a: atlas-channel inbound emission

### Files

- `services/atlas-channel/atlas.com/channel/socket/trace.go` — new file; the tenant-bound tracer closure
- `services/atlas-channel/atlas.com/channel/socket/trace_test.go` — new file
- `services/atlas-channel/atlas.com/channel/socket/init.go` — widen `CreateSocketService` with a `names map[uint16]string` parameter and install `socket.SetPacketTracer`
- `services/atlas-channel/atlas.com/channel/main.go` — build the names map, pass it, and replace the duplicated opcode-width block with `opcodes.OpReadWriterFor`

Module root: `services/atlas-channel/atlas.com/channel`.

Patterns to copy: `services/atlas-channel/atlas.com/channel/socket/init.go:114-140` (the existing `socket.Serve(...)` configurator list — `SetHandlers`, `SetCreator`, `SetMessageDecryptor`, `SetDestroyer`, `SetReadWriter`, `SetIdleNotifier`). The new `socket.SetPacketTracer(...)` goes in the same list.

### Interfaces

- Consumes: `trace.Format`, `trace.Header`, `trace.Inbound`, `trace.Enabled` (Task 1); `socket.PacketTracer`, `socket.SetPacketTracer` (Task 2); `opcodes.BuildHandlerNames`, `opcodes.OpCodeSize`, `opcodes.OpReadWriterFor` (Task 3); `configuration.TracePacketsEnabled` (Task 5).
- Produces: `func socket.NewPacketTracer(l logrus.FieldLogger, t tenant.Model, names map[uint16]string) socket2.PacketTracer` in `atlas-channel`'s `socket` package. `CreateSocketService`'s inner function signature becomes `func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, sc server.Model, ipAddress string, port int, names map[uint16]string) (net.Listener, error)`.

### Current call sites

`services/atlas-channel/atlas.com/channel/main.go:646` and `:651` today:

```go
		hp := handlerProducer(fl)(handler.AdaptHandler(fl)(t, wp))(tenantCfg.Socket.Handlers, validatorMap, handlerMap)
		...
		lis, err := socket.CreateSocketService(fl, tctx, tdm.WaitGroup(), h.Wg)(hp, rw, wp, sc, cfg.IPAddress, cfg.Port)
```

`services/atlas-channel/atlas.com/channel/main.go:429-431` today:

```go
		var rw socket2.OpReadWriter = socket2.ShortReadWriter{}
		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			rw = socket2.ByteReadWriter{}
		}
```

Both `t` (the tenant model) and `tenantCfg.Socket.Handlers` are in scope at both points — they are inside the same `buildListener` closure.

### There is no import cycle

`atlas-channel/configuration` imports only `atlas-channel/configuration/tenant` and `atlas-channel/configuration/task` (`configuration/registry.go:4`, `configuration/rest.go:4`). It does not import `socket`, so `socket` importing `configuration` is safe.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/trace_test.go`. Use `github.com/sirupsen/logrus/hooks/test` (`l, h := test.NewNullLogger(); l.SetLevel(logrus.DebugLevel)`) and register a tenant with `tenant.Register(uuid.New(), "GMS", 83, 1)`.

`TestNewPacketTracer` — one test function with sequential phases, because `configuration.PublishSnapshot` is package-global state:

| phase | setup | call | assert |
|---|---|---|---|
| flag off | no `PublishSnapshot` for this tenant | tracer with `names = {0x0001: "USER_CHAT"}`, invoked with op `0x0001`, payload `{0x01,0x00,0xaa}` | `len(h.AllEntries()) == 0` — nothing formatted (FR-2.1) |
| flag on, known handler | `configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{t.Id(): {Diagnostics: diagnostics.RestModel{TracePackets: true}}})` | same call | exactly one entry, at `logrus.DebugLevel`, whose `Message` starts with `"[PKT IN ] handler=USER_CHAT op=0x0001 len=3 session="` and contains `"\n0000  01 00 aa"` |
| flag on, unknown opcode | same snapshot | tracer invoked with op `0x00ff`, payload `{0xff,0x00}` | one further entry whose `Message` contains `"handler=<none> op=0x00ff len=2"` (FR-3.4) |
| flag on, pod at Info | `l.SetLevel(logrus.InfoLevel)` | same call as phase 2 | no new entry (FR-2.2) |
| other tenant's flag | publish with a **different** tenant id holding `TracePackets: true` and this tenant absent | same call | no new entry (FR-2.6) |

Reset the hook between phases with `h.Reset()`.

- [ ] **Step 2: Run the test to verify it fails**

Run from `services/atlas-channel/atlas.com/channel`: `go test ./socket/... -run TestNewPacketTracer -v`
Expected: FAIL to compile — undefined `NewPacketTracer`.

- [ ] **Step 3: Implement**

Create `services/atlas-channel/atlas.com/channel/socket/trace.go`:

```go
package socket

// NewPacketTracer builds the inbound tracer this tenant's listener installs
// on libs/atlas-socket. The closure captures t, so per-tenant scoping is
// structural: each listener gets its own tracer bound to its own tenant and
// there is no path by which tenant B's flag reaches tenant A's socket
// (FR-2.6).
//
// names is the op -> configured handler name map for this service's slice
// of the tenant socket config. An opcode with no entry renders "<none>"
// rather than being dropped, because an unhandled opcode is exactly the
// case the trace exists to diagnose (FR-3.4).
func NewPacketTracer(l logrus.FieldLogger, t tenant.Model, names map[uint16]string) socket.PacketTracer {
	opSize := opcodes.OpCodeSize(t.Region(), t.MajorVersion())
	return func(sessionId uuid.UUID, op uint16, payload []byte) {
		// A trace failure must never fail the packet it describes (FR-2.5).
		defer func() {
			if r := recover(); r != nil {
				l.Warnf("Packet trace panicked and was suppressed: %v", r)
			}
		}()
		if !trace.Enabled(l, configuration.TracePacketsEnabled(t.Id())) {
			return
		}
		name, ok := names[op]
		if !ok {
			name = "<none>"
		}
		l.Debug(trace.Format(trace.Header{
			Direction: trace.Inbound,
			Name:      name,
			Op:        &op,
			OpSize:    opSize,
			Length:    len(payload),
			SessionId: sessionId,
		}, payload))
	}
}
```

Imports: `atlas-channel/configuration`, `github.com/google/uuid`, `github.com/sirupsen/logrus`, `opcodes "github.com/Chronicle20/atlas/libs/atlas-opcodes"`, `socket "github.com/Chronicle20/atlas/libs/atlas-socket"`, `"github.com/Chronicle20/atlas/libs/atlas-socket/trace"`, `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`. Note the package is itself named `socket`, and `init.go` already aliases the library as `socket` — keep that alias and reference the library type as `socket.PacketTracer`.

In `services/atlas-channel/atlas.com/channel/socket/init.go`, add `names map[uint16]string` as the final parameter of both the outer return type and the inner function of `CreateSocketService` (line 83-84), and add to the `socket.Serve(...)` configurator list, after `socket.SetReadWriter(rw)`:

```go
				socket.SetPacketTracer(NewPacketTracer(l, t, names)),
```

`t` is already in scope there (`t := sc.Tenant()`, `init.go:98`).

In `services/atlas-channel/atlas.com/channel/main.go`, replace lines 429-431 with:

```go
		rw := opcodes.OpReadWriterFor(t.Region(), t.MajorVersion())
```

and change line 651 to pass the names map built from the same config the handler map is built from:

```go
		handlerNames := opcodes.BuildHandlerNames(fl, opcodes.ServiceChannel, tenantCfg.Socket.Handlers)
		lis, err := socket.CreateSocketService(fl, tctx, tdm.WaitGroup(), h.Wg)(hp, rw, wp, sc, cfg.IPAddress, cfg.Port, handlerNames)
```

Remove the now-unused `socket2` import from `main.go` only if nothing else in the file uses it — check with `grep -n 'socket2\.' services/atlas-channel/atlas.com/channel/main.go` before deleting.

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-channel/atlas.com/channel`:

```
go build ./...
go test ./socket/... ./configuration/... -race
```

Expected: build succeeds and tests PASS, including the pre-existing `wiring_test.go` and `writers_test.go`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/trace.go \
        services/atlas-channel/atlas.com/channel/socket/trace_test.go \
        services/atlas-channel/atlas.com/channel/socket/init.go \
        services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(channel): trace inbound packets via the atlas-socket tracer seam"
```

---

## Task 7b: atlas-channel outbound emission

### Files

- `services/atlas-channel/atlas.com/channel/session/trace.go` — new file; the outbound helper
- `services/atlas-channel/atlas.com/channel/session/processor.go` — trace inside `Announce` (`:249-281`) and pass the hello tracer from `Create` (`:375`)
- `services/atlas-channel/atlas.com/channel/session/model.go` — `WriteHello` gains a nil-safe tracer callback (`:143-145`)
- `services/atlas-channel/atlas.com/channel/session/processor_test.go` — add the Announce trace test

Module root: `services/atlas-channel/atlas.com/channel`.

Patterns to copy: `services/atlas-channel/atlas.com/channel/session/processor_test.go:661-712` (`TestAnnounce_StartsSpan` — `test.CreateTestContext()`, `createTestSession(sessionId)`, a stub `writer.Producer`, a no-op `packet.Encode`, and the OTel `announceMockTracer`/`announceMockSpan`/`announceMockProvider` stubs at `:6xx-657`).

### Interfaces

- Consumes: `trace.Format`, `trace.Header`, `trace.Outbound`, `trace.Enabled` (Task 1); `opcodes.OpCodeSize` (Task 3); `configuration.TracePacketsEnabled` (Task 5).
- Produces:
  - `func tracePacketOut(l logrus.FieldLogger, t tenant.Model, name string, sessionId uuid.UUID, b []byte)` — unexported, `session` package.
  - `func (s *Model) WriteHello(majorVersion uint16, minorVersion uint16, tracer func([]byte)) error` — signature change; the sole caller is `session/processor.go:375`.

### Outbound opcode resolution

`writer.MessageGetter` (`libs/atlas-socket/writer/writer.go:15-24`) writes the opcode as the first 1-2 bytes of the buffer, little-endian for the short form. So the opcode is read straight off the payload:

- `opcodes.OpCodeSize(t.Region(), t.MajorVersion()) == 1` and `len(b) >= 1` → `op = uint16(b[0])`
- size `2` and `len(b) >= 2` → `op = binary.LittleEndian.Uint16(b[0:2])`
- otherwise → `Op: nil`, rendering `op=n/a` rather than indexing out of range

### Current code being changed

`services/atlas-channel/atlas.com/channel/session/processor.go:264-275` today:

```go
						w, err := writerProducer(writerName)
						if err != nil {
							span.RecordError(err)
							span.SetStatus(codes.Error, err.Error())
							return err
						}
						if err := s.announceEncrypted(w(l, spanCtx)(encoder)); err != nil {
```

`t := tenant.MustFromContext(ctx)` is already present at `:257`.

`services/atlas-channel/atlas.com/channel/session/model.go:143-145` today:

```go
func (s *Model) WriteHello(majorVersion uint16, minorVersion uint16) error {
	return s.announce(WriteHello(nil)(majorVersion, minorVersion, s.send.IV(), s.recv.IV(), s.locale))
}
```

- [ ] **Step 1: Write the failing tests**

Add to `services/atlas-channel/atlas.com/channel/session/processor_test.go`:

`TestAnnounce_TracesOutboundPacket` — set up as `TestAnnounce_StartsSpan` at `:661` does (tenant on the context via `test.CreateTestContext()`, a `createTestSession(sessionId)` whose connection is a `net.Pipe` half that is drained in a goroutine so `announceEncrypted` does not block), with `l, h := test.NewNullLogger()` at `logrus.DebugLevel`. The stub `writer.Producer` returns a `writer.BodyFunc` yielding the fixed buffer `[]byte{0x7d, 0x00, 0xaa, 0xbb}`.

| phase | setup | assert |
|---|---|---|
| flag off | no `PublishSnapshot` carrying this tenant | `h.AllEntries()` contains no entry whose message starts with `"[PKT OUT]"` |
| flag on | `configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{tenantId: {Diagnostics: diagnostics.RestModel{TracePackets: true}}})` | exactly one Debug entry whose message starts with `"[PKT OUT] writer=CHARACTER_DATA op=0x007d len=4 session=" + sessionId.String()` and whose body line is `"0000  7d 00 aa bb                                       |}...|"` |
| writer resolution fails | producer returns `errors.New("writer not found")` | `Announce` returns that error unchanged AND no `[PKT OUT]` entry is emitted (FR-4.5) |

`TestTracePacketOut_ShortPayloadRendersNoOpcode` — call `tracePacketOut` directly with a 1-byte buffer on a `GMS`/`83` tenant (opcode width 2) and assert the message contains `"op=n/a len=1"`.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-channel/atlas.com/channel`: `go test ./session/... -run 'TracesOutbound|TracePacketOut' -v`
Expected: FAIL to compile — undefined `tracePacketOut`.

- [ ] **Step 3: Implement**

Create `services/atlas-channel/atlas.com/channel/session/trace.go` with `tracePacketOut`, applying the opcode-resolution rules above, the `trace.Enabled(l, configuration.TracePacketsEnabled(t.Id()))` gate, and a `defer recover()` that logs a Warn and swallows (FR-2.5). Emit with `l.Debug(trace.Format(...))` using `Direction: trace.Outbound`.

In `session/processor.go`'s `Announce`, insert between the writer resolution and the send — after `w, err := writerProducer(writerName)` succeeds:

```go
						b := w(l, spanCtx)(encoder)
						// Before the write, so a packet the client rejects
						// fatally is still recorded (FR-4.4). These are the
						// writer's plaintext bytes, before announceEncrypted
						// prepends the 4-byte header and applies AES-OFB
						// (FR-4.2).
						tracePacketOut(l, t, writerName, s.SessionId(), b)
						if err := s.announceEncrypted(b); err != nil {
```

Leave the `span.RecordError`/`span.SetStatus` error paths exactly as they are.

In `session/model.go`:

```go
// WriteHello sends the unencrypted handshake. tracer, when non-nil, is
// invoked with the plaintext hello bytes before they reach the connection
// (FR-4.3, FR-4.4). Passed in rather than resolved here so Model keeps no
// dependency on the configuration package and the IVs stay unexported.
func (s *Model) WriteHello(majorVersion uint16, minorVersion uint16, tracer func([]byte)) error {
	b := WriteHello(nil)(majorVersion, minorVersion, s.send.IV(), s.recv.IV(), s.locale)
	if tracer != nil {
		tracer(b)
	}
	return s.announce(b)
}
```

In `session/processor.go`'s `Create` (`:375`), pass the closure — the handshake has no opcode, so the header renders `writer=<hello> op=n/a`:

```go
		err := s.WriteHello(p.t.MajorVersion(), p.t.MinorVersion(), func(b []byte) {
			tracePacketOut(fl, p.t, "<hello>", sessionId, b)
		})
```

`tracePacketOut` must render `Op: nil` when `name == "<hello>"` — the hello frame does not begin with an opcode. Implement that as an explicit `if name == "<hello>"` branch in `tracePacketOut`, with a comment saying why, rather than relying on a length check.

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-channel/atlas.com/channel`:

```
go build ./...
go test ./session/... -race
```

Expected: build succeeds and tests PASS, including `TestAnnounce_StartsSpan`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/session/
git commit -m "feat(channel): trace outbound packets in Announce and WriteHello"
```

---

## Task 8a: atlas-login inbound emission

### Files

- `services/atlas-login/atlas.com/login/socket/trace.go` — new file
- `services/atlas-login/atlas.com/login/socket/trace_test.go` — new file
- `services/atlas-login/atlas.com/login/socket/init.go` — widen `CreateSocketService` with `names map[uint16]string` and install `socket.SetPacketTracer`
- `services/atlas-login/atlas.com/login/main.go` — build the names map, pass it, and replace the duplicated opcode-width block

Module root: `services/atlas-login/atlas.com/login`.

Patterns to copy: `services/atlas-channel/atlas.com/channel/socket/trace.go` from Task 7a (the same closure with `opcodes.ServiceLogin` supplied at the call site instead of `ServiceChannel`), and `services/atlas-login/atlas.com/login/socket/init.go:62-70` (the existing `socket.Run(...)` configurator list).

### Interfaces

- Consumes: Tasks 1, 2, 3, and `configuration.TracePacketsEnabled` from Task 6.
- Produces: `func socket.NewPacketTracer(l logrus.FieldLogger, t tenant.Model, names map[uint16]string) socket.PacketTracer` in `atlas-login`'s `socket` package. `CreateSocketService`'s inner signature becomes `func(hp socket.HandlerProducer, rw socket.OpReadWriter, wp writer.Producer, port int, names map[uint16]string)`.

### Current call sites

`services/atlas-login/atlas.com/login/main.go:272-274`:

```go
		var rw socket2.OpReadWriter = socket2.ShortReadWriter{}
		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			rw = socket2.ByteReadWriter{}
		}
```

`services/atlas-login/atlas.com/login/main.go:277` and `:300`:

```go
		hp := handlerProducer(fl)(handler.AdaptHandler(fl)(t, wp))(tenantCfg.Socket.Handlers, validatorMap, handlerMap)
		...
		socket.CreateSocketService(fl, tctx, tdm.WaitGroup())(hp, rw, wp, cfg.Port)
```

Both `t` and `tenantCfg.Socket.Handlers` are in scope inside the same `buildListener` closure (`main.go:242-303`).

`atlas-login/configuration` imports only `atlas-login/configuration/tenant` and `atlas-login/configuration/task`, so `socket` importing `configuration` introduces no cycle.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-login/atlas.com/login/socket/trace_test.go` with `TestNewPacketTracer`, phases identical to Task 7a Step 1:

| phase | setup | call | assert |
|---|---|---|---|
| flag off | no snapshot for this tenant | op `0x0001`, payload `{0x01,0x00,0xaa}`, `names = {0x0001: "LOGIN"}` | no log entries |
| flag on, known handler | publish with `{t.Id(): {Diagnostics: diagnostics.RestModel{TracePackets: true}}}` | same | one Debug entry starting `"[PKT IN ] handler=LOGIN op=0x0001 len=3 session="`, containing `"\n0000  01 00 aa"` |
| flag on, unknown opcode | same snapshot | op `0x00ff`, payload `{0xff,0x00}` | one entry containing `"handler=<none> op=0x00ff len=2"` |
| flag on, pod at Info | `l.SetLevel(logrus.InfoLevel)` | same as phase 2 | no new entry |
| other tenant's flag | publish with a different tenant id flagged on, this tenant absent | same as phase 2 | no new entry |

- [ ] **Step 2: Run the test to verify it fails**

Run from `services/atlas-login/atlas.com/login`: `go test ./socket/... -run TestNewPacketTracer -v`
Expected: FAIL to compile — undefined `NewPacketTracer`.

- [ ] **Step 3: Implement**

Create `services/atlas-login/atlas.com/login/socket/trace.go` — the same `NewPacketTracer` body as Task 7a, importing `atlas-login/configuration` instead of `atlas-channel/configuration`. Repeat the doc comments; do not cross-import the channel package.

In `services/atlas-login/atlas.com/login/socket/init.go`, add `names map[uint16]string` as the final parameter of the returned function (line 37), and add after `socket.SetReadWriter(rw)` in the `socket.Run(...)` list:

```go
					socket.SetPacketTracer(NewPacketTracer(l, t, names)),
```

`t` is already in scope (`t := tenant.MustFromContext(ctx)`, `init.go:38`).

In `services/atlas-login/atlas.com/login/main.go`, replace lines 272-274 with:

```go
		rw := opcodes.OpReadWriterFor(t.Region(), t.MajorVersion())
```

and change line 300 to:

```go
		handlerNames := opcodes.BuildHandlerNames(fl, opcodes.ServiceLogin, tenantCfg.Socket.Handlers)
		socket.CreateSocketService(fl, tctx, tdm.WaitGroup())(hp, rw, wp, cfg.Port, handlerNames)
```

Check `grep -n 'socket2\.' services/atlas-login/atlas.com/login/main.go` before removing the `socket2` import.

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-login/atlas.com/login`:

```
go build ./...
go test ./socket/... ./configuration/... -race
```

Expected: build succeeds and tests PASS, including `wiring_test.go`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-login/atlas.com/login/socket/trace.go \
        services/atlas-login/atlas.com/login/socket/trace_test.go \
        services/atlas-login/atlas.com/login/socket/init.go \
        services/atlas-login/atlas.com/login/main.go
git commit -m "feat(login): trace inbound packets via the atlas-socket tracer seam"
```

---

## Task 8b: atlas-login outbound emission

### Files

- `services/atlas-login/atlas.com/login/session/trace.go` — new file
- `services/atlas-login/atlas.com/login/session/processor.go` — trace inside `Announce` (`:209-225`) and pass the hello tracer from `Create` (`:167`)
- `services/atlas-login/atlas.com/login/session/model.go` — `WriteHello` gains the tracer callback (`:110-112`)
- `services/atlas-login/atlas.com/login/session/processor_test.go` — **new file**; this package has no test file today

Module root: `services/atlas-login/atlas.com/login`.

Patterns to copy: `services/atlas-channel/atlas.com/channel/session/processor_test.go:661-712` (`TestAnnounce_StartsSpan`) is the only existing Announce test in the repo. Adapt it: login's `Announce` has **no OTel span and no `tenant.MustFromContext` call today**, so drop the `announceMockTracer`/`announceMockSpan`/`announceMockProvider` stubs entirely and add the tenant lookup as part of this task.

### Interfaces

- Consumes: Task 1's `trace` package, Task 3's `opcodes.OpCodeSize`, Task 6's `configuration.TracePacketsEnabled`.
- Produces:
  - `func tracePacketOut(l logrus.FieldLogger, t tenant.Model, name string, sessionId uuid.UUID, b []byte)` — unexported, `atlas-login`'s `session` package.
  - `func (s *Model) WriteHello(majorVersion uint16, minorVersion uint16, tracer func([]byte)) error`.

### Current code being changed

`services/atlas-login/atlas.com/login/session/processor.go:214-219` today — note there is no span and no tenant:

```go
					return func(s Model) error {
						w, err := writerProducer(writerName)
						if err != nil {
							return err
						}
						return s.announceEncrypted(w(l, ctx)(encoder))
					}
```

`services/atlas-login/atlas.com/login/session/model.go:110-112`:

```go
func (s *Model) WriteHello(majorVersion uint16, minorVersion uint16) error {
	return s.announce(WriteHello(nil)(majorVersion, minorVersion, s.send.IV(), s.recv.IV(), s.locale))
}
```

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-login/atlas.com/login/session/processor_test.go`.

`TestAnnounce_TracesOutboundPacket` — build a context carrying a tenant (`tenant.WithContext(context.Background(), t)` with `t, _ := tenant.Register(uuid.New(), "GMS", 83, 1)`), a session created through the package's own `NewSession(sessionId, t, 8, conn)` over one half of a `net.Pipe()` whose other half is drained in a goroutine, `l, h := test.NewNullLogger()` at `logrus.DebugLevel`, and a stub `writer.Producer` returning a `writer.BodyFunc` yielding `[]byte{0x7d, 0x00, 0xaa, 0xbb}`.

| phase | setup | assert |
|---|---|---|
| flag off | no snapshot carrying this tenant | no entry starting `"[PKT OUT]"` |
| flag on | `configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{t.Id(): {Diagnostics: diagnostics.RestModel{TracePackets: true}}})` | one Debug entry starting `"[PKT OUT] writer=LOGIN_RESULT op=0x007d len=4 session=" + sessionId.String()`, body line `"0000  7d 00 aa bb                                       |}...|"` |
| writer resolution fails | producer returns `errors.New("writer not found")` | `Announce` returns that error and no `[PKT OUT]` entry is emitted (FR-4.5) |

`TestTracePacketOut_HelloRendersNoOpcode` — call `tracePacketOut(l, t, "<hello>", sessionId, []byte{0x0e, 0x00, 0x53, 0x00})` with the flag on and assert the message starts with `"[PKT OUT] writer=<hello> op=n/a len=4"`.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-login/atlas.com/login`: `go test ./session/... -v`
Expected: FAIL to compile — undefined `tracePacketOut`.

- [ ] **Step 3: Implement**

Create `services/atlas-login/atlas.com/login/session/trace.go` with the same `tracePacketOut` as Task 7b: opcode read off byte 0 (width 1) or bytes 0-1 little-endian (width 2), `Op: nil` when `name == "<hello>"` or the buffer is shorter than the opcode width, `trace.Enabled` gate, `defer recover()` Warn-and-swallow.

Rewrite `Announce`'s innermost function in `session/processor.go`:

```go
					return func(s Model) error {
						t := tenant.MustFromContext(ctx)
						w, err := writerProducer(writerName)
						if err != nil {
							return err
						}
						b := w(l, ctx)(encoder)
						// Before the write, so a packet the client rejects
						// fatally is still recorded (FR-4.4). Plaintext, before
						// announceEncrypted's header prepend and AES-OFB (FR-4.2).
						tracePacketOut(l, t, writerName, s.SessionId(), b)
						return s.announceEncrypted(b)
					}
```

Add the `tenant` import to `session/processor.go` if it is not already there.

In `session/model.go`, change `WriteHello` to the three-argument form exactly as in Task 7b Step 3, with the same doc comment.

In `session/processor.go`'s `Create` (`:167`):

```go
		err := s.WriteHello(p.t.MajorVersion(), p.t.MinorVersion(), func(b []byte) {
			tracePacketOut(fl, p.t, "<hello>", sessionId, b)
		})
```

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-login/atlas.com/login`:

```
go build ./...
go test ./session/... ./socket/... -race
```

Expected: build succeeds and the new tests PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-login/atlas.com/login/session/
git commit -m "feat(login): trace outbound packets in Announce and WriteHello"
```

---

## Task 9: atlas-ui Diagnostics page

### Files

- `services/atlas-ui/src/services/api/tenants.service.ts` — add `diagnostics` to `TenantConfigAttributes` (declared at lines 76-134)
- `services/atlas-ui/src/pages/TenantsDiagnosticsPage.tsx` — new file
- `services/atlas-ui/src/App.tsx` — add the route (the tenant subtree is at lines 441-471)
- `services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx` — add the nav entry to `sidebarNavItems` (lines 22-36)
- `services/atlas-ui/src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx` — add the nav-entry assertion
- `services/atlas-ui/src/pages/__tests__/TenantsDiagnosticsPage.test.tsx` — new file

Working directory for `npm run`: `services/atlas-ui`.

Patterns to copy:
- `services/atlas-ui/src/pages/TenantsMapleLifePage.tsx:1-72` — the current page shape: `useParams`, `useTenantConfiguration(id ?? "")`, `useUpdateTenantConfiguration()`, `updateTenantConfig.mutate({ tenant, updates: { ... } }, { onSuccess/onError with sonner toast })`, wrapped in `<TenantDetailLayout>`. This is a better template than the older `tenants-properties-form.tsx`.
- `services/atlas-ui/src/pages/tenants-properties-form.tsx:165-184` — the `<Switch checked={field.value} onCheckedChange={field.onChange} />` inside a react-hook-form `FormField`, and the Zod schema at `:24-38`, `useForm` at `:48-57`.
- `services/atlas-ui/src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx:1-36` — `vi.mock` of `ConfigExportButton` and `useTenantConfiguration`, plus the `renderAt(id, path)` `MemoryRouter`/`Routes` helper. Tests at `:51-63` and `:65-88` are the nav-item template.
- `services/atlas-ui/src/pages/__tests__/tenants-mts-config-form.test.tsx:1-94` — the newer `vi.hoisted` + mocked-hook + `userEvent` form-page test shape.

shadcn components available: `@/components/ui/switch`, `@/components/ui/alert`, `@/components/ui/card`, `@/components/ui/label`, `@/components/ui/button`.

### Interfaces

- Consumes: the JSON shape from Task 4 (`diagnostics.tracePackets`).
- Produces: `export function TenantsDiagnosticsPage()` from `@/pages/TenantsDiagnosticsPage`, routed at `/tenants/:id/diagnostics`.

### Note on the routing model

atlas-ui is **Vite + React Router**, not Next.js App Router — there is no `app/tenants/[id]/…/page.tsx` tree. Routes are `<Route>` elements listed centrally in `services/atlas-ui/src/App.tsx`, and pages are named-export components under `src/pages/`. Add one `<Route>` and one page file; create no directories.

`updateTenantConfiguration` (`services/atlas-ui/src/services/api/tenants.service.ts:305-322`) shallow-merges a `Partial<TenantConfigAttributes>` over the full cached attributes and PATCHes the whole document, so submitting `{ diagnostics: { tracePackets } }` round-trips the rest of the tenant document untouched.

- [ ] **Step 1: Write the failing tests**

In `services/atlas-ui/src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx`, add:

```tsx
it("shows the Diagnostics nav item", () => {
  vi.mocked(useTenantConfiguration).mockReturnValue({
    data: { id: "tnt-1", attributes: { socket: { handlers: [], writers: [] } } },
  } as never);
  renderAt("tnt-1");

  const link = screen.getByRole("link", { name: "Diagnostics" });
  expect(link).toHaveAttribute("href", "/tenants/tnt-1/diagnostics");
});
```

The entry is unconditional — unlike Maple Life there is no `supportsX` gate, so there is no hide-case counterpart.

Create `services/atlas-ui/src/pages/__tests__/TenantsDiagnosticsPage.test.tsx`, mocking `@/lib/hooks/api/useTenants` (both `useTenantConfiguration` and `useUpdateTenantConfiguration`) and `@/components/features/tenants/TenantDetailLayout` (render `children` directly). Four tests:

| test name | setup | assert |
|---|---|---|
| `renders the switch off for a tenant with no diagnostics object` | `data.attributes` has no `diagnostics` key | `screen.getByRole("switch")` is not checked |
| `renders the switch on for a tenant with tracePackets true` | `attributes.diagnostics = { tracePackets: true }` | the switch is checked |
| `renders the credential warning` | any tenant | the rendered text contains the exact strings `"account passwords"` and `"very large volumes of log output"`, and mentions that the pod must also run at `LOG_LEVEL=Debug` |
| `submits only the diagnostics object` | mock `useUpdateTenantConfiguration` to return `{ mutate: mutateMock, isPending: false }`; `await userEvent.click(screen.getByRole("switch"))`, then click the save control | `mutateMock` was called once, and its first argument's `updates` deep-equals `{ diagnostics: { tracePackets: true } }` — no other key |

- [ ] **Step 2: Run the tests to verify they fail**

Run from `services/atlas-ui`: `npm test -- TenantsDiagnosticsPage TenantDetailLayout`
Expected: FAIL — cannot resolve `@/pages/TenantsDiagnosticsPage`; the Diagnostics link is not found.

- [ ] **Step 3: Implement**

In `services/atlas-ui/src/services/api/tenants.service.ts`, add to `TenantConfigAttributes` after `mapleLife`:

```ts
  diagnostics?: { tracePackets: boolean };
```

Optional, because a tenant document written before this change has no `diagnostics` key.

In `services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx`, add as the last entry of `sidebarNavItems`:

```tsx
    { title: "Diagnostics", href: `/tenants/${id}/diagnostics` },
```

Create `services/atlas-ui/src/pages/TenantsDiagnosticsPage.tsx` following the `TenantsMapleLifePage` shape: `useParams`, `useTenantConfiguration(id ?? "")`, `useUpdateTenantConfiguration()`, a react-hook-form + Zod form over `z.object({ tracePackets: z.boolean() })`, reset from `tenant?.attributes.diagnostics?.tracePackets ?? false` when the query resolves, a `<Switch>` bound to it, and on submit:

```tsx
    updateTenantConfig.mutate(
      { tenant, updates: { diagnostics: { tracePackets: values.tracePackets } } },
      {
        onSuccess: () => toast.success("Successfully saved diagnostics configuration."),
        onError: (error) => toast.error(`Failed to update diagnostics configuration: ${error.message}`),
      },
    );
```

Above the switch, render a `destructive`-variant `<Alert>` whose copy states, in the operator's words, all three consequences:

- Packet tracing writes every inbound and outbound packet to this tenant's serving pods' logs, which generates **very large volumes of log output**; it is intended for short reproduction windows only.
- The dump is unredacted. Login packets carry **account passwords**, PICs/PINs and HWIDs in plaintext, so any log captured while this is on must be treated as credential-bearing material.
- Nothing is emitted unless the serving pod also runs at `LOG_LEVEL=Debug`. Turning this on alone produces no output.

In `services/atlas-ui/src/App.tsx`, import `TenantsDiagnosticsPage` alongside the other tenant pages and add after the `/tenants/:id/mts-config` route (currently lines 470-473):

```tsx
                    <Route
                      path="/tenants/:id/diagnostics"
                      element={<TenantsDiagnosticsPage />}
                    />
```

- [ ] **Step 4: Run the tests to verify they pass**

Run from `services/atlas-ui`:

```
npm test -- TenantsDiagnosticsPage TenantDetailLayout
npm run lint
npx tsc --noEmit
```

Expected: tests PASS, lint clean, no type errors.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/services/api/tenants.service.ts \
        services/atlas-ui/src/pages/TenantsDiagnosticsPage.tsx \
        services/atlas-ui/src/pages/__tests__/TenantsDiagnosticsPage.test.tsx \
        services/atlas-ui/src/App.tsx \
        services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx \
        services/atlas-ui/src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx
git commit -m "feat(atlas-ui): add the tenant Diagnostics page for packet tracing"
```

---

## Task 10: Service documentation

Documentation only — no code, no tests. PRD §8 makes this a delivered mitigation, not an optional nicety: "Documentation MUST state that logs captured with tracing enabled are to be treated as credential-bearing material."

### Files

- `services/atlas-channel/docs/domain.md` — add a "Packet trace logging" section
- `services/atlas-login/docs/domain.md` — add the same section, adapted to login

Patterns to copy: match the surrounding heading depth and prose style of each file. Use `tools/doc-slice.sh` or `grep -n '^## '` to find the section list before choosing an insertion point rather than reading either file whole — `services/atlas-channel/docs/domain.md` is 72.6K.

### Content each section must state

1. The switch is `diagnostics.tracePackets` on the tenant configuration document, owned by `atlas-configurations` and edited from the atlas-ui tenant **Diagnostics** page.
2. It reaches this service through the configuration projection; the apply loop republishes the package-level snapshot every tick (250 ms default), and `configuration.TracePacketsEnabled` reads it without blocking. A change takes effect on the next packet — no pod or session restart.
3. **Both** conditions are required: the tenant flag AND the pod at `LOG_LEVEL=Debug` or `Trace`. Flipping the flag on a pod running at `Info` produces nothing; the operator must also raise the pod's log level.
4. Output is a header line (`[PKT IN ]` / `[PKT OUT]`, name, opcode, length, session id) plus a full hex+ASCII dump, never truncated, emitted as one log entry.
5. **Security:** the dump is deliberately unredacted. Login-family packets carry account passwords, PICs/PINs and HWIDs in plaintext. Any log captured while tracing is on is credential-bearing material and must be handled as such.
6. **Volume:** an active session produces tens of packets per second and a single `CHARACTER_DATA` packet can exceed 250 dump lines. Intended for short reproduction windows only.

- [ ] **Step 1: Locate the insertion points**

Run: `grep -n '^## ' services/atlas-channel/docs/domain.md services/atlas-login/docs/domain.md`
Pick the section after which a "Packet trace logging" heading reads naturally (the socket/session area, not the gameplay domains).

- [ ] **Step 2: Write both sections**

Write the six points above into each file at the chosen heading level. Use repo-relative paths only — never a literal home or absolute path.

- [ ] **Step 3: Verify no absolute paths leaked**

Run: `grep -n '/home/\|/Users/' services/atlas-channel/docs/domain.md services/atlas-login/docs/domain.md`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/docs/domain.md services/atlas-login/docs/domain.md
git commit -m "docs(channel,login): document per-tenant packet trace logging"
```

---

## Final verification

- [ ] **Run the full gate**

Run from the worktree root: `tools/verify.sh`
Expected: exit 0. `--quick` / `--no-docker` do NOT count — they skip the bake and `-race`.

- [ ] **Run code review before opening a PR**

`tools/change-surfaces.sh` will report `go_changed=true` and `frontend_review=true`, so both `backend-guidelines-reviewer` and `frontend-guidelines-reviewer` must run, alongside `plan-adherence-reviewer`.

- [ ] **Manually trace the cross-service seam**

The projection is the seam this change crosses. Confirm by hand that `atlas-configurations`' outbox payload shape (Task 4) and both consumers' mirror `RestModel` (Tasks 5 and 6) agree on the json tag `diagnostics.tracePackets`, and that a test asserts the new contract on both sides — `TestProcessor_UpdateById_OutboxMessageCarriesDiagnostics` on the producer side, `TestTracePacketsEnabled` phase 3 on each consumer side.
