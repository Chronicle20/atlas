# Tenant Packet Trace Logging — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-27
---

## 1. Overview

When a MapleStory client crashes, disconnects, or renders garbage, the root cause is almost always a
byte-level mismatch between what the server wrote and what the client's read routine expected — a
field written as a short that the client reads as an int, an optional block gated on the wrong
version, a string length that disagrees with its payload. Today Atlas has no way to see those bytes.
The socket layer (`libs/atlas-socket`) decrypts an inbound frame and hands it straight to a handler,
and `session.Announce` encrypts an outbound buffer and writes it to the connection; neither logs the
payload. Diagnosis currently requires attaching an external packet sniffer with the tenant's AES-OFB
IVs, which is impractical against a deployed environment.

This feature adds a per-tenant diagnostic switch — default off — that makes both socket-serving
services (`atlas-channel` and `atlas-login`) emit a debug-level hex dump of every packet crossing a
client connection, in both directions, annotated with everything the server already knows about that
packet: direction, opcode, the configured handler or writer name, byte length, and session id. The
switch lives in the tenant configuration owned by `atlas-configurations`, is edited from the tenant
detail page in `atlas-ui`, and reaches the socket services over the Kafka configuration projection
those services already consume.

The intended workflow is: a tester reproduces a client crash, an operator flips the tenant's trace
flag on in the web UI and sets the affected pod's `LOG_LEVEL` to `Debug`, the crash is reproduced
once more, and the resulting log is read backwards from the disconnect to find the last packet the
client accepted and the first one it did not. The flag is a debugging instrument, not a monitoring
feature: it is expected to be on for minutes, not days.

## 2. Goals

Primary goals:

- Give a tenant-scoped, runtime-togglable switch that produces a complete byte-level record of client
  traffic without redeploying a service or rebuilding an image.
- Annotate each dump with the server-side identity of the packet — handler name for inbound, writer
  name for outbound, opcode, and length — so a dump can be matched to the code that produced or
  consumed it.
- Make the trace safe by construction when off: zero measurable cost on the hot send/receive path
  when the flag is false, and no behavioral change to the wire protocol whether it is on or off.
- Cover both socket-serving services, since a meaningful share of client crashes occur during the
  login/version handshake rather than in-channel.
- Default to off for every tenant, including tenants whose stored configuration predates this
  feature.

Non-goals:

- A packet viewer, capture browser, or replay tool in `atlas-ui`. The output is log text.
- Persisting captures to a database, object store, or pcap file.
- Tracing service-to-service traffic (Kafka events, REST calls between Atlas services).
- Per-opcode, per-session, or per-account filtering. The switch is a single tenant-wide boolean
  (see §9 for the deferred alternative).
- Redaction of credentials or other sensitive payloads (see §8 — this is an explicit, accepted
  decision, not an oversight).
- Changing, replacing, or duplicating the existing OpenTelemetry span instrumentation on the
  handler and announce paths.

## 3. User Stories

- As a server developer debugging a client crash, I want to enable byte-level packet logging for a
  single tenant so that I can see the exact bytes the client received immediately before it died.
- As a server developer, I want each dump labelled with the writer name that produced it so that I
  can go straight to the `socket/writer/*.go` file responsible rather than reverse-mapping an opcode.
- As a server developer, I want inbound dumps labelled with the handler name and opcode so that I can
  tell whether a malformed request was misrouted or mis-parsed.
- As a server developer, I want to see inbound packets whose opcode has no registered handler so that
  I can identify protocol operations the server is silently dropping.
- As an operator, I want the flag defaulted off and settable from the web UI so that turning tracing
  on for a reproduction does not require a config file edit, a redeploy, or a pod restart.
- As an operator, I want the flag scoped to one tenant so that enabling it for a test tenant does not
  flood the logs with another tenant's traffic on a shared pod.

## 4. Functional Requirements

### 4.1 Configuration

- **FR-1.1** Tenant configuration gains a boolean diagnostic field controlling packet trace logging.
  It is stored inside the tenant's JSON configuration blob (`tenants.Entity.Data`), so no relational
  schema migration is required.
- **FR-1.2** The field's absent value MUST deserialize to `false`. Every tenant whose stored
  configuration predates this change therefore has tracing off with no backfill.
- **FR-1.3** The field MUST be settable to `true` and back to `false` through the existing tenant
  configuration update endpoint, and MUST be included in the tenant configuration read response.
- **FR-1.4** Updating the field MUST follow the tenant processor's existing update semantics,
  including writing a `tenant_history` record before modification.
- **FR-1.5** The field MUST be carried on the Kafka tenant-configuration status event so that
  `atlas-channel` and `atlas-login` observe the change through their existing configuration
  projection, without a pod restart.
- **FR-1.6** A change to the field alone MUST NOT cause the listener registry to tear down and
  rebuild listeners, drop sessions, or otherwise disturb connected clients. Flipping the flag on a
  live tenant is a non-disruptive operation.
- **FR-1.7** The template-level configuration (the schema tenants derive from) is out of scope for
  this field. Tracing is an operational, per-tenant concern, and a template value would seed new
  tenants with a debugging switch. If a template field is later wanted it is a follow-up.

### 4.2 Emission — general

- **FR-2.1** When the tenant's flag is `false`, no trace output is produced and no dump is formatted.
  The check MUST short-circuit before any byte formatting work is done.
- **FR-2.2** When the tenant's flag is `true`, the trace is emitted at the logger's **Debug** level.
  Output therefore additionally requires the emitting pod's `LOG_LEVEL` to be `Debug` or `Trace`.
  Both conditions are required; neither alone produces output. (Decision recorded in §9.)
- **FR-2.3** The flag is resolved per packet from the current configuration snapshot, so a change
  takes effect on the next packet without restarting a session or a pod.
- **FR-2.4** A configuration snapshot that is not yet ready, or a tenant absent from the snapshot,
  MUST be treated as "tracing off". A trace lookup MUST NOT block the send or receive path, and MUST
  NOT propagate an error into packet handling.
- **FR-2.5** Trace emission MUST NOT alter, reorder, delay, or drop any packet. A failure inside the
  trace path (a formatting panic, a nil logger) MUST NOT fail the packet it was describing.
- **FR-2.6** Tracing is scoped to the tenant that owns the session. On a pod serving multiple
  tenants, a packet belonging to tenant A MUST NOT be traced because tenant B has the flag on.

### 4.3 Emission — inbound (client → server)

- **FR-3.1** Inbound packets are traced in `atlas-channel` and `atlas-login` at the point where the
  handler name is known — the per-service `socket/handler.AdaptHandler` seam
  (`services/atlas-channel/atlas.com/channel/socket/handler/handle.go:65`,
  and the `atlas-login` equivalent) — so the configured handler name from
  `opcodes.HandlerConfig.Handler` can be included.
- **FR-3.2** The traced bytes are the **decrypted plaintext** frame as delivered to the handler,
  including the opcode bytes. Ciphertext is not traced.
- **FR-3.3** The trace MUST be emitted before the handler runs, so a packet that crashes or hangs a
  handler still leaves a record.
- **FR-3.4** An inbound packet whose opcode has no registered handler MUST still be traced, with the
  handler name reported as unknown. This is emitted from `libs/atlas-socket`'s dispatch
  (`server.go` `handle`), which is where the "no handler" case is observable.
- **FR-3.5** The trace MUST be emitted regardless of whether the handler's validator passes, so a
  packet rejected by `LoggedInValidator` is still visible.

### 4.4 Emission — outbound (server → client)

- **FR-4.1** Outbound packets are traced in `session.Announce`
  (`services/atlas-channel/atlas.com/channel/session/processor.go:249`,
  `services/atlas-login/atlas.com/login/session/processor.go:219`), which is the single choke point
  where both the resolved `writerName` and the encoded byte buffer are available.
- **FR-4.2** The traced bytes are the **plaintext** encoded packet as produced by the writer, before
  the 4-byte header is prepended and before AES-OFB encryption. Ciphertext is not traced.
- **FR-4.3** The unencrypted handshake packet sent by `Model.WriteHello` MUST also be traced, with a
  writer name identifying it as the hello/handshake packet. It does not flow through `Announce`.
- **FR-4.4** The trace MUST be emitted before the write to the connection, so a packet that the
  client rejects fatally is recorded even though the connection dies.
- **FR-4.5** If the writer producer fails to resolve a writer name to an opcode, the resulting error
  path is unchanged; no partial dump is emitted for a packet that was never encoded.

### 4.5 Output format

- **FR-5.1** Each traced packet produces a header line followed by a classic hex+ASCII dump:

  ```
  [PKT OUT] writer=CHARACTER_DATA op=0x7D len=4212 session=3f2a1c88-...
  0000  7d 00 01 00 00 00 ff ff  ff ff 01 05 00 4d 61 70  |}............Map|
  0010  6c 65 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |le..............|
  ```

- **FR-5.2** The header line MUST carry: direction (inbound/outbound, distinguishable at a glance),
  the handler name (inbound) or writer name (outbound), the opcode in hex, the packet length in
  decimal bytes, and the session id.
- **FR-5.3** Dump body lines MUST use a 4-hex-digit offset, 16 bytes per line rendered as
  lowercase two-digit hex, a wider gap between the 8th and 9th byte, and a trailing ASCII gutter
  delimited by `|` in which bytes outside the printable ASCII range `0x20`–`0x7E` render as `.`.
- **FR-5.4** A final partial line MUST be padded so the ASCII gutter stays column-aligned.
- **FR-5.5** The dump MUST NOT be truncated. Large packets (e.g. `CHARACTER_DATA`, several KB) are
  dumped in full, because an invalid read frequently occurs in the tail. (Decision recorded in §9.)
- **FR-5.6** A packet of zero body length MUST produce the header line and no body lines.
- **FR-5.7** The header line and body MUST be emitted as a single log entry (one multi-line message),
  not as N independent log calls, so concurrent sessions on the same pod cannot interleave one
  packet's body into another's.
- **FR-5.8** The dump formatter MUST live in a shared location usable by both services and by
  `libs/atlas-socket` — the natural home is `libs/atlas-socket` — so channel and login produce
  byte-identical formatting.

## 5. API Surface

No new endpoints. The existing tenant configuration resource in `atlas-configurations` gains one
field.

**`GET /api/tenants/{id}`** and **`GET /api/tenants`** — response `attributes` gains the boolean
field. Example fragment:

```json
{
  "data": {
    "type": "tenants",
    "id": "0e9b...",
    "attributes": {
      "region": "GMS",
      "majorVersion": 83,
      "minorVersion": 1,
      "usesPin": true,
      "tracePackets": false,
      "socket": { "handlers": [], "writers": [] }
    }
  }
}
```

**`PATCH`/`PUT /api/tenants/{id}`** — accepts the same field. Omitting it in a request body follows
the endpoint's existing whole-document update semantics; the UI form submits the full attribute set,
as `tenants-properties-form.tsx` already does for `usesPin`.

Error cases: none new. The field is a plain boolean with no validation rules beyond JSON type, so it
contributes no entries to the tenant `validation_error` collection.

The exact JSON field name (`tracePackets` above is illustrative) is a design-phase decision; it must
be chosen consistently across the Go REST models in `atlas-configurations`, `atlas-channel`,
`atlas-login`, and the TypeScript attribute interfaces in `atlas-ui`.

## 6. Data Model

- No relational migration. Tenant configuration is persisted as a JSON document in
  `tenants.Entity.Data` (`services/atlas-configurations/atlas.com/configurations/tenants/entity.go`),
  and the new field is a key inside that document. Existing rows omit the key and unmarshal to
  `false`.
- `tenant_history` records are written by the existing update path and will capture the field like
  any other configuration change.
- The Go `RestModel` in `atlas-configurations/tenants` gains the field; the mirror `RestModel` in
  `atlas-channel/configuration/tenant` and `atlas-login/configuration/tenant` gains it as well, since
  those are the shapes the configuration projection deserializes into.
- Tenant configuration is already environment-scoped (`Entity.Environment`); the new field inherits
  that scoping with no additional work.

## 7. Service Impact

| Service / library | Change |
|---|---|
| `atlas-configurations` | Add the field to the tenant `RestModel`; ensure it round-trips through create/update/read and the outbox Kafka event. No entity or migration change. |
| `atlas-channel` | Add the field to `configuration/tenant.RestModel`; resolve it per packet from `configuration.GetTenantConfig`; emit inbound traces at `socket/handler.AdaptHandler` and outbound traces at `session.Announce` and `Model.WriteHello`. |
| `atlas-login` | Same as `atlas-channel`, at the corresponding `socket/handler` adapter, `session.Announce` (`processor.go:219`), and `Model.WriteHello` (`session/model.go:111`). |
| `libs/atlas-socket` | Host the shared hex+ASCII dump formatter (FR-5.8); emit the unhandled-opcode inbound trace from `handle` (FR-3.4) via an injected, optional tracer so the library keeps no tenant knowledge of its own. |
| `libs/atlas-opcodes` | Likely unchanged — the flag is a tenant-level property, not part of the socket handler/writer config. Design phase confirms. |
| `atlas-ui` | Add the boolean to `TenantConfigAttributes` in `src/services/api/tenants.service.ts`; add a labelled `Switch` to `src/pages/tenants-properties-form.tsx` (Global Properties tab) with its Zod schema entry, form reset wiring, and submit payload entry. |

Services not affected: every other Atlas service. Only `atlas-channel` and `atlas-login` serve client
sockets (verified: they are the only two importing `atlas-socket/handler` / calling `socket.Serve`).

## 8. Non-Functional Requirements

**Performance.** When the flag is off the cost per packet MUST be a snapshot read plus a boolean
test — no allocation, no string building. When on, the cost is accepted to be significant: a
multi-kilobyte packet produces several hundred formatted lines. The flag is a debugging instrument
and is not expected to be enabled under load.

**Log volume.** An active channel session produces on the order of tens of packets per second, and a
single `CHARACTER_DATA` packet can exceed 250 dump lines. Operators must be warned in the UI that
this generates very large volumes of log output and is intended for short reproduction windows. The
UI control's helper text MUST state this.

**Security.** The trace is deliberately unredacted (recorded decision, §9): login-family packets
carry account passwords, PICs/PINs, and HWIDs in plaintext, and those bytes will appear in the log
stream when the flag is on for a tenant. This is accepted because a partially-masked dump defeats the
offset-level debugging the feature exists for. Mitigations that MUST be delivered:

- The flag defaults to off and requires two independent conditions (tenant flag *and* pod
  `LOG_LEVEL=Debug`) to produce any output, so it cannot be enabled accidentally by a config edit
  alone.
- The UI control MUST carry an explicit warning that enabling it writes credentials to the log
  stream, and MUST be visually distinguished from ordinary configuration switches.
- Documentation MUST state that logs captured with tracing enabled are to be treated as
  credential-bearing material.

**Multi-tenancy.** Trace decisions are per-tenant and resolved from the tenant on the session's
context. A pod serving several tenants MUST trace only the tenants with the flag set.

**Observability.** The trace complements, and does not replace, the existing OTel spans on the
handler and announce paths. The session id on the header line is the correlation key back to those
spans and to the rest of the session's log lines.

**Concurrency.** Inbound handling is per-packet goroutine-based and outbound sends are serialized by
`Model.sendLock`. The trace MUST be a single log call per packet (FR-5.7) and MUST NOT introduce new
locking on the send path.

## 9. Open Questions

Resolved during the spec interview (recorded here as decisions, not open items):

- **Log level.** Both conditions are required: the tenant flag *and* the pod running at `Debug`.
  The alternative — emitting regardless of `LOG_LEVEL` so the flag alone suffices — was considered
  and rejected. Consequence to document in the runbook: flipping the flag on a pod running at `Info`
  produces nothing, and the operator must also raise the pod's log level.
- **Granularity.** A single tenant-wide boolean. Opcode/name allowlists, direction filters, and
  per-session scoping are deferred; if log volume proves unmanageable in practice they are the
  natural follow-up.
- **Redaction.** None. Raw bytes always, including login credentials. See §8 for the mitigations
  this obligates.
- **Format.** Classic hex+ASCII, no truncation.

Genuinely open, for the design phase:

- **OQ-1** Exact JSON/Go/TS field name and whether it sits at the top level of the tenant attributes
  or under a nested diagnostics object. A nested object leaves room for the deferred filters without
  a second flat field later.
- **OQ-2** Whether `libs/atlas-socket` should own an optional tracer hook (a configurator, matching
  the existing `SetHandlers`/`SetCreator`/`SetMessageDecryptor` pattern) or whether the
  unhandled-opcode case is better surfaced by returning that condition to the service. The former
  keeps the library free of tenant knowledge; the latter avoids adding a configurator.
- **OQ-3** Whether the seed data under `services/atlas-configurations/seed-data/` needs the field
  written explicitly as `false` or should rely on the zero value.
- **OQ-4** Whether the UI switch belongs on the existing Global Properties tab or on a new
  "Diagnostics" tab in `TenantDetailLayout`'s sidebar. A separate tab makes the warning easier to
  present and keeps a dangerous switch out of the routine-settings page.

## 10. Acceptance Criteria

Configuration:

- [ ] A tenant configuration read response includes the trace flag; a tenant created before this
      change reports `false`.
- [ ] The flag can be set to `true` and back to `false` via the tenant update endpoint, and the value
      persists across a service restart.
- [ ] Updating the flag writes a `tenant_history` record.
- [ ] The updated value reaches `atlas-channel` and `atlas-login` through the configuration
      projection without restarting either pod, and is observable within the projection's normal
      propagation window.
- [ ] Changing only the flag does not tear down or rebuild any listener and does not disconnect any
      connected session.

Emission:

- [ ] With the flag off, no packet trace output appears at any pod log level.
- [ ] With the flag on but the pod at `LOG_LEVEL=Info`, no packet trace output appears.
- [ ] With the flag on and the pod at `LOG_LEVEL=Debug`, every inbound packet on a session belonging
      to that tenant produces a dump carrying direction, handler name, opcode, length, and session id.
- [ ] Under the same conditions, every outbound packet produces a dump carrying direction, writer
      name, opcode, length, and session id.
- [ ] The `WriteHello` handshake packet is traced.
- [ ] An inbound packet with no registered handler is traced, marked as having no handler.
- [ ] An inbound packet rejected by its validator is still traced.
- [ ] On a pod serving two tenants where only one has the flag on, only that tenant's packets are
      traced.

Format:

- [ ] The dump renders 16 bytes per line with a 4-digit offset, a gap after the 8th byte, and a
      `|`-delimited ASCII gutter with non-printables as `.`.
- [ ] A packet larger than 4 KB is dumped in full, with no truncation marker.
- [ ] A final partial line is padded so the ASCII gutter remains aligned.
- [ ] A zero-length packet body produces a header line and no body lines.
- [ ] The header and body of one packet arrive as a single log entry and are never interleaved with
      another packet's dump under concurrent sessions.
- [ ] Channel and login produce byte-identical formatting for the same input bytes.

UI:

- [ ] The tenant detail page exposes a switch for the flag, defaulting to the persisted value.
- [ ] Toggling the switch and saving persists the change and surfaces the existing success/error
      toasts.
- [ ] The control carries helper text warning about log volume and about credentials appearing in
      the log stream.

Gates:

- [ ] Unit tests cover the dump formatter (alignment, non-printables, partial final line, empty
      body), the off/on gating including the snapshot-not-ready and tenant-absent cases, and the
      per-tenant scoping.
- [ ] Frontend tests cover the new form field's render, default, and submit payload.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Backend and frontend guideline reviews pass before the PR is opened.
