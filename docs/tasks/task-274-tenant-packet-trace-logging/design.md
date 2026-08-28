# Tenant Packet Trace Logging — Design

Version: v1
Status: Approved for planning
Created: 2026-08-27
Consumes: `prd.md` (v1)

---

## 1. Summary

The feature decomposes into four independent pieces:

1. **A configuration field** — `diagnostics.tracePackets`, a nested boolean inside the tenant
   configuration JSON document, carried end to end by machinery that already exists.
2. **A live, non-blocking read path** for that field inside `atlas-channel` and `atlas-login`.
   This is the only part that needs new plumbing, because *neither service currently refreshes its
   `configuration` package snapshot after startup* (§4 — a latent defect this task must fix to meet
   FR-2.3).
3. **A shared formatter and a tracer seam** in `libs/atlas-socket`, so both services emit
   byte-identical dumps and the library stays free of tenant knowledge.
4. **Two emission sites per service** — one inbound (in the library's dispatch), one outbound (in
   `session.Announce`, plus the `WriteHello` handshake).

Everything else — persistence, history, the Kafka projection, listener stability, JSON:API
round-tripping — falls out of existing behaviour with no new code.

---

## 2. Decisions on the PRD's open questions

### OQ-1 — Field name and shape: **nested `diagnostics.tracePackets`**

```json
"diagnostics": { "tracePackets": false }
```

Chosen over a flat `tracePackets` for three reasons:

- It matches the established shape of the tenant document. `cashShop` and `mapleLife` are already
  nested sub-objects backed by their own small Go packages
  (`services/atlas-configurations/atlas.com/configurations/tenants/cashshop`,
  `.../maplelife`), each declared as one field on `tenants.RestModel`. A nested `diagnostics`
  object is the convention, not an exception.
- It namespaces an operationally dangerous, non-gameplay switch away from gameplay settings. The
  UI tab in OQ-4 then maps 1:1 onto a configuration subtree.
- The deferred filters in PRD §9 (opcode allowlist, direction filter, per-session scope) land as
  siblings inside `diagnostics` with no JSON migration. Promoting a flat `tracePackets` into an
  object later would require backfilling every stored tenant document.

The cost is three small mirror structs (one per Go module that deserializes the tenant document) and
one TypeScript interface — the same cost `cashShop` already pays.

Go shape, in `tenants/diagnostics/rest.go` (and mirrored in the two consumers):

```go
package diagnostics

// RestModel carries per-tenant operational diagnostics switches. Every field
// must be a zero-valued-safe boolean: a tenant document written before this
// object existed unmarshals to the zero value, which is "off" (FR-1.2).
type RestModel struct {
    TracePackets bool `json:"tracePackets"`
}
```

Absent-key handling requires no code: `encoding/json` leaves both the missing `diagnostics` object
and the missing `tracePackets` key at their zero values, satisfying FR-1.2 with no backfill.

### OQ-2 — Where the library seam lives: **a single optional tracer configurator, and inbound is traced *only* in the library**

The PRD (FR-3.1) proposed tracing inbound packets in each service's `socket/handler.AdaptHandler`
so the handler name is available, with `libs/atlas-socket` covering only the unhandled-opcode case
(FR-3.4). This design **deviates**: it emits every inbound trace from a single site inside
`libs/atlas-socket`'s `handle`, and supplies the handler name to the library as an
`op → name` lookup map.

Rationale — the library site is strictly better on every requirement the PRD states:

| Requirement | `AdaptHandler` site | library `handle` site |
|---|---|---|
| FR-5.2 opcode in header | Not available. `opcodes.HandlerAdapter` receives the handler *name* but the opcode is the map key (`libs/atlas-opcodes/producer.go:84`); it would have to be re-derived from the payload, or the adapter signature widened across two libraries and two services. | Already parsed: `op := config.rw.Read(&reader)` (`libs/atlas-socket/server.go`). |
| FR-3.3 emitted before the handler runs | Yes. | Yes, and strictly earlier. |
| FR-3.4 unhandled opcode traced | Needs a second, differently-shaped emission site in the library. | Same site, same format, `handler=<none>`. |
| FR-3.5 validator-rejected packets traced | Yes. | Yes. |
| Session not present in the registry | **Not traced** — `AdaptHandler` wraps its body in `sp.IfPresentById`, which silently returns when no session exists. A packet arriving during teardown would vanish. | Traced. |
| Format identity across services (FR-5.8) | Two service copies of the call. | One call. |

The handler-name map is trivial to build: `libs/atlas-opcodes` already parses
`HandlerConfig.OpCode` and `HandlerConfig.Handler` in `BuildHandlerMap`. It gains a sibling:

```go
// BuildHandlerNames returns the op -> configured handler name map for this
// service's slice of the tenant socket config. Same filtering and opcode
// parsing as BuildHandlerMap; separate so the trace path does not depend on
// a handler actually being registered.
func BuildHandlerNames(l logrus.FieldLogger, service string, handlers []HandlerConfig) map[uint16]string
```

This confirms that PRD §7's "`libs/atlas-opcodes` likely unchanged" is not quite right: the library
gains one additive exported function and no signature changes.

The library seam itself is a configurator, matching `SetHandlers` / `SetCreator` /
`SetMessageDecryptor`:

```go
// libs/atlas-socket/opts.go
type PacketTracer func(sessionId uuid.UUID, op uint16, payload []byte)

func SetPacketTracer(tracer PacketTracer) Configurator { ... }
```

`config.tracer` defaults to nil. `handle` becomes:

```go
func handle(l logrus.FieldLogger) func(config *config, sessionId uuid.UUID, p request.Request) {
    return func(config *config, sessionId uuid.UUID, p request.Request) {
        reader := request.NewRequestReader(&p, time.Now().Unix())
        op := config.rw.Read(&reader)
        if config.tracer != nil {
            config.tracer(sessionId, op, p)
        }
        if h, ok := config.handlers[op]; ok {
            h(sessionId, reader)
        } else {
            l.Infof("Read a unhandled message with op 0x%02X.", op&0xFF)
        }
    }
}
```

`p` is the full decrypted plaintext frame including the opcode bytes (FR-3.2). The existing
unhandled-opcode `Infof` stays; the trace is additive. The library holds no tenant knowledge — the
closure the service installs owns the tenant, the name map, and the flag check.

### OQ-3 — Seed data: **no change**

`services/atlas-configurations/seed-data/` contains only `templates/`, and FR-1.7 puts templates
out of scope. There is no tenant seed file to edit. The zero value covers every existing tenant row.

### OQ-4 — UI placement: **a new "Diagnostics" sidebar entry on the tenant detail page**

`TenantDetailLayout` (`services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx:22`)
already builds `sidebarNavItems` as a plain array, and already conditionally inserts an entry
(`Maple Life`). Adding `{ title: "Diagnostics", href: \`/tenants/${id}/diagnostics\` }` is a
one-line change plus one route and one page.

Chosen over the Global Properties tab because:

- The credential-exposure warning (§8 of the PRD) needs room to be prominent. On the properties tab
  it competes with Region/Major/Minor/Uses PIN and reads as boilerplate.
- Global Properties is the routine-settings page. A switch that writes plaintext passwords into the
  log stream should not sit one tab-stop from the fields an operator edits during normal setup.
- The tab maps 1:1 onto the `diagnostics` config subtree, so future deferred filters extend the
  page rather than bloating properties.

The new page reuses the properties form's exact structure (react-hook-form + Zod + `Switch` +
`useUpdateTenantConfiguration`). Note that `tenantsService.updateTenantConfiguration`
(`services/atlas-ui/src/services/api/tenants.service.ts:305`) already spreads a `Partial<TenantConfigAttributes>`
over the full cached attributes and PATCHes the whole document, so the diagnostics page submits only
`{ diagnostics: { tracePackets } }` and the rest of the tenant document round-trips untouched.

---

## 3. Data flow

```
atlas-ui  Diagnostics page
   └─ PATCH /api/tenants/{id}  { attributes: { …, diagnostics: { tracePackets: true } } }
        └─ atlas-configurations tenants.ProcessorImpl.UpdateById
             ├─ json.Marshal(input) → tenants.Entity.Data          (persist, FR-1.1)
             ├─ update(...) → tenant_history row                    (FR-1.4)
             └─ enqueueTenantStatus → outbox → EVENT_TOPIC_CONFIGURATION_TENANT_STATUS  (FR-1.5)
                  └─ atlas-channel / atlas-login projection.Subscriber
                       └─ projection.State.ApplyTenant  (tenant.RestModel incl. Diagnostics)
                            └─ projection.ApplyLoop tick (250 ms)
                                 ├─ ComputeOps → NO ops (Diagnostics is not in ListenerConfig) (FR-1.6)
                                 └─ configuration.PublishSnapshot(svc, tenants)   ← NEW (§4)
                                      └─ configuration.TracePacketsEnabled(tenantId) ← NEW, non-blocking
                                           ├─ socket.SetPacketTracer closure   (inbound)
                                           └─ session.Announce / WriteHello    (outbound)
```

Nothing on the write side is new. `UpdateById`
(`services/atlas-configurations/atlas.com/configurations/tenants/processor.go:124`) marshals the
whole `RestModel` into `Entity.Data` and enqueues the same sanitized model onto the outbox, so a new
field is carried by construction. `projection.State.ApplyTenant` unmarshals the whole
`tenant.RestModel`, so the mirror struct is the only thing that has to know the field exists.

**FR-1.6 is satisfied for free.** `projection.ComputeOps` diffs only `desired{Key, Cfg}`, and
`ListenerConfig` holds `IPAddress, Port, Region, MajorVersion, MinorVersion`
(`configuration/projection/apply.go`). A change confined to `Diagnostics` produces an identical
`desired` map, so no `OpDrain`/`OpAdd` is emitted and no session is disturbed. A test must pin this
(§8).

---

## 4. The blocking problem: the snapshot is never refreshed

**This is the one part of the PRD that cannot be implemented as written.** PRD §7 says
`atlas-channel` should "resolve it per packet from `configuration.GetTenantConfig`". Two defects
make that wrong:

1. **`GetTenantConfig` blocks.** It calls `waitReady()`, which waits up to `readyTimeout` (60 s) on
   `readyCh` (`services/atlas-channel/atlas.com/channel/configuration/registry.go:68`). FR-2.4
   forbids a trace lookup that blocks the send or receive path.
2. **In `atlas-channel`, nothing ever populates it.** `configuration.PublishSnapshot` has **no
   production caller** in `atlas-channel` — `main.go` imports only
   `atlas-channel/configuration/projection`. `serviceConfig` is therefore permanently nil and
   `readyCh` is never closed. In `atlas-login`, `PublishSnapshot` is called exactly once, at
   `main.go:100`, immediately after catch-up — so its snapshot is frozen at startup and a later
   config change is invisible to `GetTenantConfig` callers.

Registry.go's own doc comment states the intended contract:

> "Called by main.go after CaughtUp fires (**and again from the projection apply loop on each
> observed change**) so legacy callers of GetServiceConfig / GetTenantConfig … see the same data the
> listener registry was built from."

The apply-loop call site was never written. Completing it is a prerequisite for FR-2.3, not scope
creep — and it also repairs `atlas-channel`'s `session/task.go:27` `GetServiceConfig()` call, which
today blocks 60 s and returns `ErrNotReady` on every invocation.

### 4.1 Republish from the apply loop

In both services, `configuration/projection/loop.go` already imports the `configuration` package and
already takes a snapshot every tick. Add the publish immediately after the snapshot:

```go
case <-t.C:
    nextSvc, nextTenants := a.State.Snapshot()
    // Keep the legacy configuration package vars in step with the
    // projection on every tick. Without this, atlas-channel never
    // populates them at all and atlas-login freezes them at startup.
    configuration.PublishSnapshot(nextSvc, nextTenants)
    ops := ComputeOps(prevSvc, prevTenants, nextSvc, nextTenants)
```

Cost: one map copy of N tenants at 4 Hz. Negligible at Atlas's tenant counts, and the copy already
happens for `Snapshot()` regardless.

Ordering is safe: the loop runs only after `WaitCaughtUp`, and no listener exists (so no packet can
arrive) until the loop's first `OpAdd` executes on that same tick, after the publish.

`atlas-login` keeps its existing one-shot `publishSnapshot()` at `main.go:100` — it runs before the
apply loop starts and is what the account-session consumer and `accept_tos` handler rely on today.

### 4.2 A non-blocking accessor

Added to both services' `configuration/registry.go`:

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

FR-2.3 (a change takes effect on the next packet, no restart) is met with a worst-case latency of
one apply-loop tick, 250 ms, plus Kafka propagation.

**Alternative considered and rejected:** threading `*projection.State` into the emission sites.
It works for `AdaptHandler` (main.go's `AddBody` closure has the state) but not for
`session.Announce`, which is a package-level curried function invoked from several hundred call
sites with a fixed `(l)(ctx)(wp)(name)(encoder)` shape. Widening it is a repo-scale mechanical
change for no benefit over a package-level accessor.

---

## 5. `libs/atlas-socket/trace` — the shared formatter

New package, so `libs/atlas-socket` (root) and both services can import it without a cycle.

```go
package trace

type Direction int
const (
    Inbound Direction = iota
    Outbound
)

// Header is everything the header line carries (FR-5.2). Op is a pointer
// because the WriteHello handshake has no opcode (FR-4.3) — nil renders
// "op=n/a". OpSize (1 or 2) fixes the hex width so a byte-opcode tenant
// does not print a misleading four-digit opcode.
type Header struct {
    Direction Direction
    Name      string    // configured handler name (in) or writer name (out); "<none>" when unresolved
    Op        *uint16
    OpSize    int
    Length    int
    SessionId uuid.UUID
}

// Format renders the complete trace as ONE multi-line string: the header
// line followed by the hex+ASCII body. Single string, single log call, so
// concurrent sessions on a pod cannot interleave one packet's body into
// another's (FR-5.7).
func Format(h Header, b []byte) string

// Dump renders only the hex+ASCII body. Exported for direct testing.
func Dump(b []byte) string
```

Rendering rules (FR-5.3 / 5.4 / 5.5 / 5.6):

- 16 bytes per line, `%04x` offset, lowercase `%02x` bytes, two spaces between the 8th and 9th
  byte, ASCII gutter delimited by `|`, bytes outside `0x20`–`0x7E` rendered `.`.
- A short final line pads the hex columns with spaces so the gutter stays aligned.
- No truncation at any length.
- Zero-length body: header line only, no trailing newline artifacts.
- Built with a single `strings.Builder`, pre-sized to
  `len(b)/16*(~78 bytes)` so a 4 KB packet is one allocation-amortised build rather than 256
  concatenations.

Direction renders as the literal prefixes `[PKT IN ]` and `[PKT OUT]` — fixed width, so a column of
dumps aligns and `grep 'PKT OUT'` is exact.

```
[PKT OUT] writer=CHARACTER_DATA op=0x007d len=4212 session=3f2a1c88-…
0000  7d 00 01 00 00 00 ff ff  ff ff 01 05 00 4d 61 70  |}............Map|
0010  6c 65 00                                          |le.|
[PKT IN ] handler=<none> op=0x00ff len=6 session=3f2a1c88-…
```

### 5.1 The gate helper

Both conditions of FR-2.2 (tenant flag AND pod at Debug) are checked before any formatting. The
level check is best-effort via an interface assertion, because `logrus.FieldLogger` does not expose
the level but both `*logrus.Logger` and `*logrus.Entry` implement `IsLevelEnabled`:

```go
// Enabled reports whether a dump should be built at all. The tenant flag is
// checked first and is the cheap path (one RLock + map read + bool). The
// level probe is best-effort: a FieldLogger implementation that does not
// expose IsLevelEnabled falls through to "enabled" and lets logrus discard
// the entry, which costs a formatted string but never loses a trace.
func Enabled(l logrus.FieldLogger, flag bool) bool {
    if !flag {
        return false
    }
    if lc, ok := l.(interface{ IsLevelEnabled(logrus.Level) bool }); ok {
        return lc.IsLevelEnabled(logrus.DebugLevel)
    }
    return true
}
```

FR-2.1's "short-circuit before any byte formatting" is met: with the flag off the entire cost is the
nil check on `config.tracer` (inbound) plus this function's first branch.

---

## 6. Emission sites

### 6.1 Inbound — one site, in the library, one closure per listener

The closure is built where the tenant, the logger and the tenant socket config all exist. In
`atlas-channel` that is `socket.CreateSocketService`; the `op → name` map is passed in from
`main.go`'s `AddBody`, which already holds `tenantCfg.Socket.Handlers`.

```go
// socket/init.go, inside CreateSocketService
names := opcodes.BuildHandlerNames(l, opcodes.ServiceChannel, handlerConfigs)
socket.Serve(l, ctx, wg, fanOut, lis,
    …,
    socket.SetPacketTracer(tracePacketIn(l, t, names)),
)

func tracePacketIn(l logrus.FieldLogger, t tenant.Model, names map[uint16]string) socket.PacketTracer {
    return func(sessionId uuid.UUID, op uint16, payload []byte) {
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
            OpSize:    opcodes.OpCodeSize(t.Region(), t.MajorVersion()),
            Length:    len(payload),
            SessionId: sessionId,
        }, payload))
    }
}
```

The closure captures `t`, so FR-2.6 (per-tenant scoping on a multi-tenant pod) holds structurally:
each listener installs its own tracer bound to its own tenant, and there is no path by which tenant
B's flag can reach tenant A's socket. `atlas-login` gets the identical function with
`opcodes.ServiceLogin`.

FR-2.5 (a trace failure never fails the packet) — the closure returns nothing, `l.Debug` cannot
error, and `handle` ignores it. A `defer recover()` inside the closure guards a formatter panic:

```go
defer func() {
    if r := recover(); r != nil {
        l.Warnf("Packet trace panicked and was suppressed: %v", r)
    }
}()
```

### 6.2 Outbound — `session.Announce`

```go
w, err := writerProducer(writerName)
if err != nil {
    …                            // FR-4.5: unchanged error path, no partial dump
}
b := w(l, spanCtx)(encoder)
tracePacketOut(l, t, writerName, s.SessionId(), b)   // before the write (FR-4.4)
if err := s.announceEncrypted(b); err != nil { … }
```

`t` is already in scope in `atlas-channel`'s `Announce` (`t := tenant.MustFromContext(ctx)`);
`atlas-login`'s `Announce` adds the same line. The bytes handed to the tracer are the writer's
plaintext output *before* the 4-byte header prepend and AES-OFB encryption inside
`announceEncrypted` (FR-4.2).

**Resolving the outbound opcode.** `writer.MessageGetter` writes the opcode as the first 1–2 bytes
of the buffer (`libs/atlas-socket/writer/writer.go`), little-endian for the short form
(`response.Writer.WriteShort` → `binary.LittleEndian`). So the opcode is read straight off the
payload; the only unknown is its width. Today that width rule is duplicated verbatim in both
services:

```go
var rw socket2.OpReadWriter = socket2.ShortReadWriter{}
if t.Region() == "GMS" && t.MajorVersion() <= 28 {
    rw = socket2.ByteReadWriter{}
}
```
(`services/atlas-channel/atlas.com/channel/main.go:429`, `services/atlas-login/atlas.com/login/main.go:272`)

Lift it into `libs/atlas-opcodes` as the single source of truth, and have both `main.go` files use
it:

```go
// OpCodeSize returns the wire width in bytes of this tenant's opcodes.
func OpCodeSize(region string, majorVersion uint16) int

// OpReadWriterFor returns the OpReadWriter matching OpCodeSize.
func OpReadWriterFor(region string, majorVersion uint16) socket.OpReadWriter
```

This is a targeted improvement to code the change is already touching, not unrelated refactoring:
without it the tracer would become a third copy of the rule.

**Alternative considered and rejected:** a per-tenant `writerName → opcode` registry, mirroring
`writer.RegisterTenantWriterOptions`. It avoids reading the opcode off the payload, but requires a
new registry in `atlas-login` (which has no equivalent today) plus registration and eviction
lifecycle in both services — strictly more machinery than one pure function, for a value that is
already sitting in byte 0 of the buffer being dumped.

### 6.3 The handshake packet

`Model.WriteHello` builds its bytes internally from the session's own IVs
(`session/model.go:143` in channel, `:110` in login) and writes them via `announce` — unencrypted,
never through `Announce`. It has neither a logger nor a tenant.

Rather than exporting the IVs, `WriteHello` takes a nil-safe callback:

```go
// WriteHello sends the unencrypted handshake. tracer, when non-nil, is
// invoked with the plaintext hello bytes before they reach the connection
// (FR-4.3/FR-4.4). Passed in rather than resolved here so Model keeps no
// dependency on the configuration package and the IVs stay unexported.
func (s *Model) WriteHello(majorVersion, minorVersion uint16, tracer func([]byte)) error {
    b := WriteHello(nil)(majorVersion, minorVersion, s.send.IV(), s.recv.IV(), s.locale)
    if tracer != nil {
        tracer(b)
    }
    return s.announce(b)
}
```

The sole caller in each service is the session processor's create path
(`session/processor.go:375` in channel, `:167` in login), which has `p.l` and `p.t` and builds the
closure. The header renders `writer=<hello> op=n/a`.

---

## 7. Component inventory

| Unit | Location | Purpose | Depends on |
|---|---|---|---|
| `diagnostics.RestModel` | `atlas-configurations/…/tenants/diagnostics` | The persisted field | — |
| `diagnostics.RestModel` (mirror) | `atlas-channel/configuration/tenant/diagnostics`, `atlas-login/…` | Projection deserialization target | — |
| `trace.Format` / `trace.Dump` / `trace.Enabled` | `libs/atlas-socket/trace` | Pure formatting + gating | logrus |
| `socket.PacketTracer` / `SetPacketTracer` | `libs/atlas-socket` | Optional inbound seam | — |
| `opcodes.BuildHandlerNames` | `libs/atlas-opcodes` | op → configured handler name | `HandlerConfig` |
| `opcodes.OpCodeSize` / `OpReadWriterFor` | `libs/atlas-opcodes` | Single source of the width rule | `atlas-socket` |
| `configuration.TracePacketsEnabled` | both services | Non-blocking live flag read | projection republish |
| apply-loop `PublishSnapshot` | both services' `configuration/projection/loop.go` | Keeps the snapshot live | `configuration` |
| `tracePacketIn` / `tracePacketOut` | both services' `socket/` and `session/` | Tenant-bound closures | all of the above |
| Diagnostics page + route + nav entry | `atlas-ui` | Operator control | `useUpdateTenantConfiguration` |

Every unit above is independently testable: the formatter is pure, the gate takes its inputs, the
accessor is a map read, and the closures are small enough to exercise with a fake logger.

---

## 8. Testing

**`libs/atlas-socket/trace`** — table tests over `Dump`:
16-byte-aligned input; a 1-byte final line and a 15-byte final line (gutter alignment, FR-5.4);
an all-non-printable payload (`.` substitution, FR-5.3); an empty payload (header only, FR-5.6);
a >4 KB payload asserting the exact line count and the absence of any truncation marker (FR-5.5).
`Format` tests cover both directions, a nil `Op` (`op=n/a`), `OpSize` 1 vs 2 hex widths, and that
the returned value is exactly one string containing `\n` (FR-5.7). `Enabled` tests cover
flag-off, flag-on-with-Info-logger, flag-on-with-Debug-logger, and a `FieldLogger` that does not
implement `IsLevelEnabled`.

**`libs/atlas-socket`** — `handle` with a nil tracer (no panic, unchanged behaviour); with a tracer
and a registered handler (tracer called before the handler, payload includes the opcode bytes);
with a tracer and an unregistered opcode (tracer still called, FR-3.4).

**`configuration`, both services** — `TracePacketsEnabled` returns false before any
`PublishSnapshot`, without blocking (assert the call returns well inside `readyTimeout`); returns
false for a tenant absent from the snapshot; returns the published value; reflects a second
`PublishSnapshot` (FR-2.3); returns tenant A's value for A and tenant B's for B (FR-2.6).

**`configuration/projection`, both services** — `ComputeOps` with two snapshots differing *only* in
`Diagnostics.TracePackets` emits zero ops (FR-1.6). This is the test that protects connected
sessions from an operator toggling the switch.

**`atlas-configurations/tenants`** — `RestModel` round-trips `diagnostics.tracePackets` through
marshal/unmarshal; a document with no `diagnostics` key unmarshals to `false` (FR-1.2); the
existing update-path test set is extended to assert the field reaches the outbox payload (FR-1.5).

**`atlas-ui`** — the Diagnostics page renders the switch with the persisted value, submits
`{ diagnostics: { tracePackets: true } }`, and renders the warning copy. `TenantDetailLayout` test
gains an assertion for the new nav entry.

**Gates** — flagless `tools/verify.sh` exits 0; `backend-guidelines-reviewer` and
`frontend-guidelines-reviewer` both run before the PR (`tools/change-surfaces.sh` will report
`go_changed=true` and `frontend_review=true`).

---

## 9. Error handling

| Condition | Behaviour |
|---|---|
| Snapshot never published / tenant absent | `TracePacketsEnabled` → `false`. No block, no error, no log. (FR-2.4) |
| Pod below Debug level | `trace.Enabled` → `false` before formatting. (FR-2.2) |
| Opcode has no configured handler | Traced with `handler=<none>`; the pre-existing `Infof` is unchanged. (FR-3.4) |
| Validator rejects the packet | Already traced — the trace fired in the library, upstream of the validator. (FR-3.5) |
| Session absent from the registry | Already traced, for the same reason. |
| `writerProducer` cannot resolve the name | Error returns exactly as today; no bytes were produced, so nothing is dumped. (FR-4.5) |
| Formatter panics | `recover` inside the tracer closure logs a Warn and swallows it; the packet is unaffected. (FR-2.5) |
| Payload shorter than the opcode width | The tracer renders `op=n/a` rather than indexing out of range. |

---

## 10. Security and operability

The trace is deliberately unredacted; PRD §8 records that decision and the mitigations it obligates.
This design delivers them:

- Default off, by the zero value, with no backfill (FR-1.2).
- Two independent conditions required for any output (FR-2.2), so a config edit alone cannot start
  leaking credentials.
- The UI control lives on its own Diagnostics page (OQ-4) whose copy must state both consequences
  explicitly — log volume, and that login-family packets carry account passwords, PICs/PINs and
  HWIDs in plaintext.
- Service documentation must record that any log captured with tracing enabled is
  credential-bearing material.

The runbook consequence of FR-2.2 to document: flipping the tenant flag on a pod running at `Info`
produces nothing. The operator must also raise `LOG_LEVEL`.

---

## 11. Deviations from the PRD

Two, both recorded above with their reasoning:

1. **FR-3.1** — inbound tracing happens in `libs/atlas-socket`'s `handle`, not in each service's
   `socket/handler.AdaptHandler`. The handler name is supplied to the library as an `op → name`
   map. This makes FR-3.4 the same code path instead of a second one, gives FR-5.2 the opcode for
   free, and additionally traces packets arriving for a session that is no longer in the registry.
   (§2, OQ-2)
2. **§7, `libs/atlas-opcodes`** — the PRD expected it unchanged. It gains two additive exported
   helpers (`BuildHandlerNames`, `OpCodeSize`/`OpReadWriterFor`) and no signature changes. (§2, §6.2)

One prerequisite defect must be fixed inside this task, because FR-2.3 cannot hold without it:
`configuration.PublishSnapshot` is never called in `atlas-channel` and is called only once at
startup in `atlas-login`. Wiring it into the projection apply loop completes the contract
`registry.go` already documents. (§4)
