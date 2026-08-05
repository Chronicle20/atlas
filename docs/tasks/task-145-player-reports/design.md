# Player Reports (Sue / Claim System) — Design

Task: task-145-player-reports
Status: Approved-pending-review
Inputs: `prd.md` (approved), `packet-findings.md` (IDA-verified 2026-07-09)

This document records the architecture, the alternatives considered, and every
design-phase decision the PRD deferred. All code references were verified against
the worktree on 2026-07-09.

---

## 1. Architecture Overview

Five services participate. Flows, end to end:

```
Sue:    client ──SUE_CHARACTER──▶ atlas-channel handler
                                      │ REPORT command (Kafka)
                                      ▼
Claim:  client ──CLAIM_REQUEST──▶ atlas-channel handler ──▶ atlas-ban report processor
                                                              │ resolve accused (atlas-character REST)
                                                              │ fetch transcript (atlas-messages REST)
                                                              │ persist `reports` row
                                                              │ REPORT_STATUS event (Kafka)
                                                              ▼
        client ◀─SUE_CHARACTER_RESULT / CLAIM_RESULT── atlas-channel status consumer

Enable: atlas-channel session bootstrap ──CLAIM_STATUS_CHANGED(1) + CLAIM_AVAILABLE_TIME(0,0)──▶ client

Capture: chat command ──▶ atlas-messages processor ──▶ chat event (existing)
                                   └──▶ bounded per-character Redis buffer (new)

Review: GM ──▶ atlas-ui /reports ──▶ nginx /api/reports ──▶ atlas-ban REST (list/detail/PATCH)
```

Result delivery is asynchronous and identity-correlated (reporter character id +
world/tenant match), exactly like the existing buddy round trip
(`services/atlas-channel/atlas.com/channel/kafka/consumer/buddylist/consumer.go`).
No correlation token — consistent with every other command→status→packet flow in
the codebase.

## 2. Alternatives Considered

### 2.1 Report store ownership
- **atlas-ban (chosen, per PRD)** — already owns the moral neighborhood (`ban`,
  `history`), has DB + Kafka + REST scaffolding, and the `ban` domain is a
  complete template to mirror. Reports and bans will eventually cross-reference.
- New `atlas-reports` service — cleanest isolation, but a whole new service
  (deploy, bake target, services.json + docker-bake.hcl, ingress) for one small
  domain. Rejected as overhead without a boundary benefit.
- atlas-messages — owns chat but nothing durable (no DB today); reports are not
  messages. Rejected.

### 2.2 Chat-capture storage
- **Redis via `libs/atlas-redis` `TenantKeyedSortedSet` (chosen)** — atlas-messages
  is currently stateless; if it ever runs >1 replica, an in-process buffer is
  wrong twice over (REST queries load-balance to a replica that didn't consume
  that actor's partition). A timestamp-scored sorted set per character mirrors
  the existing atlas-merchant visitor registry
  (`services/atlas-merchant/atlas.com/merchant/visitor/registry.go`), passes the
  redis-key-guard invariant, and survives restarts within the retention window.
- In-memory registry — fails multi-replica, loses buffer on restart. Rejected.
- Postgres table — introduces atlas-messages' first DB for short-retention
  working state; per-line INSERT+prune churn for data that is 99% never read.
  Rejected.

### 2.3 Transcript attachment point
- **atlas-ban pulls from atlas-messages REST at report creation (chosen)** — the
  snapshot is taken exactly once, by the service that persists it, and a
  messages outage degrades to `server_transcript = null` without failing the
  report.
- atlas-channel embeds the transcript in the REPORT command — channel doesn't own
  the buffer; it would have to make the same REST call on the packet-handling
  path. Worse placement of the same work. Rejected.
- atlas-ui queries the live buffer at review time — retention (minutes) is far
  shorter than review latency (hours/days). Rejected.

### 2.4 Result delivery
- **Async status event consumed by atlas-channel (chosen)** — matches buddy /
  whisper / note / invite round trips; session located via
  `IfPresentByCharacterId`.
- Synchronous REST from channel to ban inside the packet handler — simpler
  correlation but blocks the socket-handler goroutine on a cross-service call
  and breaks the codebase's command/event convention. Rejected.

## 3. Packet Layer — `libs/atlas-packet/report/`

One new feature package holding all five packets (subpackage of the existing
`libs/atlas-packet` module — **no Dockerfile or go.work changes needed**):

```
libs/atlas-packet/report/
  clientbound/
    sue_character_result.go    // CWvsContext::OnSueCharacterResult, 1 byte
    claim_result.go            // CWvsContext::OnClaimResult, mode byte (+ payload for SUCCESS)
    claim_available_time.go    // CWvsContext::OnSetClaimSvrAvailableTime, 2 bytes
    claim_status_changed.go    // CWvsContext::OnClaimSvrStatusChanged, 1 byte
  serverbound/
    claim_request.go           // CWvsContext::SendClaimRequest
```

Rationale for a single `report` package rather than `claim/` + `sue/`: the repo
organizes by game feature (fame, note, party each hold their `CWvsContext::On*Result`);
sue and claim are two wire mechanisms of the same reporting feature and share one
domain downstream. The existing serverbound `SUE_CHARACTER` stays where it is
(`field/serverbound/sue_character.go` — it is `CField::SendChatMsgSlash`, already
T1-verified; we do not move it).

### 3.1 Codec shapes (from `packet-findings.md`)

All follow the standard shape: unexported fields, constructor, `Operation()`
returning the writer/handler name constant, `Encode(l, ctx)` +`Decode(l, ctx)`
mirrors, inline `t.MajorVersion()` guards written for the packet-audit analyzer
(model: `field/serverbound/sue_character.go:49-85`).

- `SueCharacterResult{result byte}` — `WriteByte`. Writer const
  `SueCharacterResultWriter = "SueCharacterResult"`.
- `ClaimResult` — discrete structs sharing `ClaimResultWriter = "ClaimResult"`
  (note/guild pattern, `note/clientbound/operation.go:16-76`):
  - `ClaimResultSuccess{mode byte, hasRemaining bool, remaining int32}` —
    mode, `WriteBool`, `WriteInt`.
  - `ClaimResultNotice{mode byte}` — bare mode byte (covers 3, 0x41–0x45, 0x47,
    0x48).
  This is **not** a dispatcher family per `docs/packets/DISPATCHER_FAMILY.md`'s
  narrow criteria: the IDA decompile (both v83 and v95) shows one switch where
  only mode 2 reads payload and every other arm reads nothing — the
  `SetSkillResponse`-style "mode + conditional payload" shape, not N
  distinct-body sub-handlers. No `families.yaml` entry, no dispatcher-lint
  involvement. The packet-verifier pass re-confirms this from the decompile; if
  that ever contradicts (it should not, given the findings), the implementer
  escalates before coding.
- `ClaimAvailableTime{openHour, closeHour byte}` — two `WriteByte`.
  `ClaimAvailableTimeWriter = "ClaimAvailableTime"`.
- `ClaimSvrStatusChanged{connected bool}` — one byte.
  `ClaimSvrStatusChangedWriter = "ClaimSvrStatusChanged"`.
- `ClaimRequest` (serverbound), `ClaimRequestHandle = "ClaimRequest"`:

  ```
  bChatClaim  byte
  targetName  string      (ReadAsciiString)
  reasonType  byte
  description string
  chatLog     string      only when bChatClaim == 1
  ```

  Body identical across v72–v95 per findings (v95 verified; v72 @ `0x91f2b4`
  and v79 @ `0x9711ff` verified 2026-08-04 per PRD FR-6.4; v83 confirmed during
  implementation per FR-6.2 — see §9.2). No version branch expected; if the v83
  byte-verification disagrees, the codec gains an inline `MajorVersion()` guard
  at that point. `CLAIM_REQUEST` does not exist on v48/v61 (PRD §2.1), so the
  codec's version span starts at v72.

All five packets carry `packet-audit:fname` comments with the CWvsContext names
above, and golden/round-trip tests per `field/serverbound/sue_character_test.go`
conventions (`pt.CreateContext`, `pt.Variants`, `pt.RoundTrip`, hard-coded byte
fixtures with `packet-audit:verify` markers).

### 3.2 Result-code resolution (DOM-25)

Both result-carrying writers resolve their client-interpreted bytes from the
tenant writer `options.operations` table via `ResolveCode`/`WithResolvedCode`
(`libs/atlas-packet/resolve.go`) — never literals in domain logic, even though
the values are verified identical across the supported span (config-drive-all-modes
rule, task-103 precedent). The *opcodes* differ per version (PRD §2.1) while the
*mode values* do not; config resolution absorbs both. Tables (values from
`packet-findings.md` §1/§3):

```
SueCharacterResult operations:
  SUCCESS: 0, UNABLE_TO_LOCATE: 1, DAILY_LIMIT: 2, REPORTED_NOTICE: 3, GENERIC_FAILURE: 4
  (client renders any value outside 0–3 as the generic-failure line; 4 is the
   deliberate "other" bucket)

ClaimResult operations:
  SUCCESS: 2, REPORTED_NOTICE: 3, TRY_AGAIN: 65, RECHECK_NAME: 66,
  NOT_ENOUGH_MESOS: 67, CANNOT_CONNECT: 68, EXCEEDED: 69,
  TIME_WINDOW: 71, FALSE_REPORT_CITED: 72
```

v1 emits only SUCCESS / UNABLE_TO_LOCATE / GENERIC_FAILURE (sue) and SUCCESS /
RECHECK_NAME / TRY_AGAIN (claim); every other key exists in the table and the
writers accept any key, satisfying the PRD's "expressible but unused" requirement.

## 4. atlas-channel

Service root: `services/atlas-channel/atlas.com/channel` (**CH**).

### 4.1 Handlers

- **Sue** — replace the stub body of `CH/socket/handler/sue_character.go`:
  decode as today, then call `report.NewProcessor(l, ctx).SueAndEmit(...)` with
  the session's character id as reporter, the wire identity
  (legacy `characterId` / v95 `subCommand` string), `flag`, `reason`. No
  packet is written here — the result packet comes from the status consumer.
- **Claim** — new `CH/socket/handler/claim_request.go`
  (`ClaimRequestHandleFunc`), registered in `produceHandlers()`
  (`CH/main.go:795-902`) as `handlerMap[reportsb.ClaimRequestHandle]`. Decodes
  `ClaimRequest`, calls `report.NewProcessor(l, ctx).ClaimAndEmit(...)`.

### 4.2 New channel domain package `CH/report/`

`processor.go` + `producer.go` mirroring `CH/buddylist/`: builds the REPORT
command (§6) and publishes to `EnvCommandTopicReport`. Pure emit — no local
state. The wire→command mapping (which identity field was supplied) lives here.

### 4.3 Writers

Four thin `*Body` wrappers in `CH/socket/writer/` (model:
`CH/socket/writer/blocked_server.go`), constants appended to `produceWriters()`
(`CH/main.go:608-793`):

- `SueCharacterResultBody(l)(options, key)` — resolves the operations key via
  `WithResolvedCode("operations", key, ...)`.
- `ClaimResultSuccessBody(l)(options, hasRemaining, remaining)` /
  `ClaimResultNoticeBody(l)(options, key)` — same resolution.
- `ClaimAvailableTimeBody(open, close byte)`, `ClaimSvrStatusChangedBody(connected bool)` — plain.

### 4.4 Claim-enable emission

In `processStateReturn` (`CH/kafka/consumer/session/consumer.go:155-338`), in the
existing goroutine block alongside the keymap/buddy sends: announce
`ClaimSvrStatusChanged(true)` then `ClaimAvailableTime(0, 0)` (always-open —
verified client behavior for `open == close == 0`).

**Gating is config-presence, not code:** attempt the writer lookup via the
writer producer; if the tenant config has no `ClaimSvrStatusChanged` writer
entry the producer returns "writer not found" — log at debug and skip both
sends. jms, gms-12, gms-48, gms-61 and gms-92 tenants (§9.2) are thereby
claim-disabled purely by omitting template entries; no region/version
conditionals in Go. This is what lets gms-61 run sue-only for free: it gets the
sue entries and no claim entries, and the same config-presence gate keeps its
claim UI dark.

### 4.5 Status consumer

New `CH/kafka/consumer/report/consumer.go` subscribed to
`EnvEventTopicReportStatus` (group `"report_status_event"`,
`kafka.LastOffset`), mirroring the buddy consumer:

1. Guard `sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId)`.
2. Locate the reporter's session:
   `session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.ReporterId, op)`.
   Reporter offline → drop silently (feedback is best-effort; the report is
   already persisted).
3. Map `(kind, status, errorCode)` → writer + operations key:

   | kind  | status  | errorCode  | packet sent |
   |-------|---------|------------|-------------|
   | sue   | CREATED | —          | SueCharacterResult SUCCESS |
   | sue   | ERROR   | NOT_FOUND  | SueCharacterResult UNABLE_TO_LOCATE |
   | sue   | ERROR   | INTERNAL   | SueCharacterResult GENERIC_FAILURE |
   | claim | CREATED | —          | ClaimResultSuccess (hasRemaining=1, remaining=100) |
   | claim | ERROR   | NOT_FOUND  | ClaimResultNotice RECHECK_NAME |
   | claim | ERROR   | INTERNAL   | ClaimResultNotice TRY_AGAIN |

   `remaining = 100` is a named constant (quotas untracked in v1; the value is
   only display text — "100 reports left this week").

## 5. atlas-ban — `report` domain

Service root: `services/atlas-ban/atlas.com/ban`. New top-level `report/`
package mirroring the `ban` domain's file naming (`resource.go` = RestModel +
Transform/Extract, `rest.go` = routes + handlers — the fuller of the two
in-service conventions).

### 5.1 Files

```
report/
  entity.go         // GORM Entity + Migration; TableName "reports"
  model.go          // immutable Model + getters; Kind + Status enums + domain errors
  builder.go        // NewBuilder(tenantId, kind, reporterId) …validation… Build()
  provider.go       // entityById, entitiesByTenant, entitiesByStatus (curried EntityProvider)
  administrator.go  // create, updateStatus, Make(Entity) (Model, error)
  processor.go      // Processor iface + Impl; Create/CreateAndEmit, UpdateStatus, providers
  resource.go       // RestModel, GetName() "reports", Transform/Extract
  rest.go           // InitResource: PathPrefix("/reports"); GET list/one, PATCH
  producer.go       // status-event message providers
  task.go           // (none needed — omit; no background task in v1)
  mock/processor.go // ProcessorMock with XxxFunc fields (atlas-notes pattern)
```

Plus:

- `kafka/message/report/kafka.go` — topics + envelope (§6).
- `kafka/consumer/report/consumer.go` — `InitConsumers` / `InitHandlers` pair,
  wired in `main.go` next to the ban consumer (`main.go:59-67`).
- `main.go:57` — add `report.Migration` to `database.SetMigrations(...)`;
  `main.go:76-77` — add `report.InitResource`.

### 5.2 Entity (per PRD §6, uuid surrogate PK)

```go
Id               uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
TenantId         uuid.UUID `gorm:"not null;index:idx_reports_tenant_status,priority:1"`
Kind             string    `gorm:"not null"`            // sue | claim
ReporterId       uint32    `gorm:"not null"`
ReporterName     string    `gorm:"not null"`
AccusedId        uint32    `gorm:"not null"`
AccusedName      string    `gorm:"not null"`
ReasonType       byte      `gorm:"not null"`            // wire byte: sue flag / claim nType
Description      string    `gorm:"type:text;not null"`  // sue reason / claim sContext
ChatLog          *string   `gorm:"type:text"`           // claim chat-claims only, verbatim
ServerTranscript datatypes.JSON `gorm:""`               // nullable snapshot
Status           string    `gorm:"not null;default:open;index:idx_reports_tenant_status,priority:2"`
CreatedAt, UpdatedAt time.Time
```

`AccusedId` is non-null: unresolved targets are rejected before persistence
(FR-3.4), so no half-resolved rows exist.

### 5.3 Create flow (Kafka consumer → processor)

`handleCreateCommand`:

1. **Resolve accused** via a new `character/` REST-client package
   (requests.go/processor.go/model.go copied from
   `services/atlas-channel/atlas.com/channel/character/`, minus inventory
   include): claim + v95 sue supply a name → `ByNameProvider`; legacy sue
   supplies an id → by-id lookup for the name. `requests.ErrNotFound` /
   empty result → emit `ERROR/NOT_FOUND`, persist nothing. The v95 sue
   sub-command string is treated as the target name for resolution; if the live
   client sends something else there, resolution fails safe to NOT_FOUND.
   Also resolve the **reporter's** name (by id) for `reporter_name`.
2. **Cap inputs** (NFR): `description` truncated to 2,000 chars, `chat_log` to
   16,384 bytes — truncate-and-log, never reject (a truncated report is more
   useful to a GM than a vanished one). Constants in `report/model.go`.
3. **Fetch transcript** (claim and sue alike): new `chat/` REST-client package
   calling atlas-messages (§7.3) for recent lines involving reporter + accused.
   Any error → log warning, `server_transcript = null`, continue. This call
   must never fail a report.
4. **Persist + emit** `CreateAndEmit` (ban-domain AndEmit shape,
   `ban/processor.go:60-71`): create row, `buf.Put(EnvEventTopicReportStatus,
   createdEventProvider(...))`. Internal failure → emit `ERROR/INTERNAL`.

atlas-ban gains two service-URL envs: `CHARACTERS_SERVICE_URL`,
`MESSAGES_SERVICE_URL` — both optional with `BASE_SERVICE_URL` fallback via
`requests.RootUrl` (do **not** hard-code overlay-namespace literals in the
kustomize base; rely on the fallback like npc-shops).

### 5.4 REST

Flat `/api/` base (atlas-ban has no service segment):

- `GET /api/reports?status=open` — list, tenant from header context; `status`
  as a plain query param (matching `bans?type=` convention, not
  `filter[status]` — the PRD's §5 route sketch is superseded here and by
  `/api/reports` replacing `/api/rpts/...`).
- `GET /api/reports/{reportId}` — full detail including `chatLog` and
  `serverTranscript`.
- `PATCH /api/reports/{reportId}` — `registerInputHandler` +
  `rest.ParseReportId` (added to `services/atlas-ban/atlas.com/ban/rest/handler.go`),
  mirroring atlas-notes `UpdateNoteHandler`: URL-id == body-id check → 400,
  unknown id → 404, invalid status value → 400 via a sentinel domain error
  (`ErrInvalidStatus`) mapped in the handler. Transitions unrestricted among
  `open|reviewed|actioned` (PRD FR-3.5). Response echoes the updated resource.

`serverTranscript` attribute shape (PRD §5, plus chat type):
`[{timestamp, senderId, senderName, chatType, text}]`.

README: update atlas-ban's REST-endpoints and Kafka tables.

## 6. Kafka Contract

New message package (mirrored in channel and ban, same constant names both
sides, like the chat topics):

```go
EnvCommandTopicReport     = "COMMAND_TOPIC_REPORT"
EnvEventTopicReportStatus = "EVENT_TOPIC_REPORT_STATUS"

// Command[E]{Type string, Body E} — CommandTypeCreate = "CREATE"
CreateCommandBody {
  Kind        string   // sue | claim
  WorldId     world.Id
  ChannelId   channel.Id
  ReporterId  uint32
  AccusedId   uint32   // legacy sue: wire value; else 0
  AccusedName string   // claim: wire value; v95 sue: subCommand; else ""
  ReasonType  byte
  Description string
  ChatClaim   bool     // claim only
  ChatLog     string   // claim chat-claims only
}

StatusEvent {
  ReportId   uuid.UUID // zero uuid on ERROR
  Kind       string
  WorldId    world.Id
  ReporterId uint32
  Status     string    // CREATED | ERROR
  ErrorCode  string    // NOT_FOUND | INTERNAL (empty on CREATED)
}
```

Both topics keyed by reporter id (`producer.CreateKey`); tenant + span headers
via the standard producer decorators; consumers use
`SetHeaderParsers(SpanHeaderParser, TenantHeaderParser)`.

## 7. atlas-messages — Chat Capture

### 7.1 Capture point

The six handler methods in
`services/atlas-messages/atlas.com/messages/message/processor.go` are the single
choke point where every non-slash-command chat line passes with actor, text,
type, and field context in hand. After the command-registry check (so slash
commands are never captured), each of `HandleGeneral`, `HandleMulti`,
`HandleWhisper`, `HandleMessenger` appends to the buffer. **Captured types:
GENERAL, BUDDY, PARTY, GUILD, ALLIANCE, WHISPER, MESSENGER** (all
player-authored text — exceeds the PRD's general-chat floor at near-zero
marginal cost since they share one code path). **Not captured: PET**
(actor id is the pet, echo of owner input), **PINK_TEXT** (system-issued).
Capture failure logs a warning and never blocks the chat flow.

The wire carries no timestamp — lines are stamped at capture time.

### 7.2 Buffer: new `chat/registry.go` (or `capture/`) in atlas-messages

Redis via `libs/atlas-redis`, namespace `"chat:recent"`, one
timestamp-scored sorted set per character (key = sender character id, member =
JSON line record, score = unix-milli). Line record:

```json
{"ts": 1720540800123, "senderId": 1, "senderName": "Foo", "type": "GENERAL",
 "text": "...", "worldId": 0, "channelId": 1, "mapId": 100000000}
```

`TenantKeyedSortedSet` lacks trimming, so **extend `libs/atlas-redis`** (extend
the existing lib, never a new one) with a bounded-append operation on
`TenantKeyedSortedSet`: one pipelined `ZADD` + `ZREMRANGEBYSCORE (-inf,
now-window)` + `ZREMRANGEBYRANK` (count cap) + `EXPIRE key <window>` — the key
TTL makes idle characters' buffers evaporate without a sweeper. Query side uses
the existing `Range` (sets are small by construction). Unit-testable with the
lib's existing miniredis-style test approach; guard stays clean because all
commands live inside `libs/atlas-redis`.

Config (env, read at startup):

- `CHAT_CAPTURE_RETENTION_SECONDS` — default **900** (15 min; comfortably
  covers the client's own claim-log ring buffer, which holds only the current
  session's recent lines).
- `CHAT_CAPTURE_MAX_LINES` — default **200** per character.

atlas-messages gains its first Redis connection — copy the standard wiring from
any peer service (e.g. atlas-expressions `main.go`).

### 7.3 Query surface

atlas-messages' REST server is already running with zero domain routes
(`main.go:71-79`), so this is one `AddRouteInitializer`:

- `GET /api/chat/history?characterIds={a},{b}` — JSON:API resource
  `"chat-messages"`: union of each listed character's buffered lines, merged
  and sorted by timestamp ascending. Server-to-server only (atlas-ban calls it
  via `MESSAGES_SERVICE_URL`/`BASE_SERVICE_URL`); **no nginx/ingress entry** —
  it is not exposed to the UI. Tenant from header context as everywhere.

"Involving A and B" = lines *authored by* A plus lines *authored by* B. A
whisper from an uninvolved third party to the accused is out of scope, matching
the client's own two-character chat log semantics.

README: document the new endpoint and the capture behavior; update
`docs/storage.md` (no longer storage-free).

## 8. atlas-ui — Reports Admin Page

Mirror the Bans feature file-for-file (Vite + React 19 SPA, react-router 7,
TanStack Query 5):

```
src/pages/ReportsPage.tsx            // list; status filter (Select), DataTableWrapper
src/pages/reports-columns.tsx        // kind, reporter, accused, reason, status badge, createdAt
src/pages/ReportDetailPage.tsx       // description | chatLog | serverTranscript side by side (Cards)
src/services/api/reports.service.ts  // BASE_PATH "/api/reports"; getList/getOne/updateStatus
src/lib/hooks/api/useReports.ts      // reportKeys factory keyed on tenant id; useReports/useReport/useUpdateReportStatus
src/types/models/report.ts           // {id, attributes}; const-object enums (erasableSyntaxOnly)
src/components/features/reports/     // ReportStatusBadge, UpdateReportStatusDialog
```

- Status PATCH uses the tenants-service envelope pattern:
  `api.patch('/api/reports/'+id, {data: {type: 'reports', id, attributes: {status}}})`
  — satisfying the JSON:API-envelope gotcha; mutation invalidates
  `reportKeys.all`.
- Chat log and transcript render as plain text (`whitespace-pre-wrap` text
  nodes — never `dangerouslySetInnerHTML`); transcript rows as a simple table.
- Registration: routes `/reports` + `/reports/:reportId` in `App.tsx`
  (lazy-imported, inside the AppShell layout route); sidebar item under the
  **Security** group in `src/components/app-sidebar.tsx`; breadcrumb entries in
  `src/lib/breadcrumbs/routes.ts`.

## 9. Configuration & Deployment

### 9.1 Ingress

Add to `deploy/shared/routes.conf` **and** `deploy/compose/routes.conf`
(alphabetical placement), then run `./deploy/scripts/sync-k8s-ingress-routes.sh`:

```nginx
location ~ ^/api/reports(/.*)?$ {
  set $u "atlas-ban:8080";
  proxy_pass http://$u$request_uri;
}
```

### 9.2 Seed templates

In `services/atlas-configurations/seed-data/templates/`, per PRD §2.1. Two
tiers: **gms_72, gms_79, gms_83, gms_84, gms_87, gms_95** get the full set;
**gms_61** gets the sue pair only.

Handlers (opcodes from `docs/packets/audits/STATUS.md`) — the sue handler is
already routed in every template; only `ClaimRequest` is new:

| template | new handler entry |
|---|---|
| gms_72 | `{ "opCode": "0x69", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }` |
| gms_79 | `{ "opCode": "0x68", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }` |
| gms_83 | `{ "opCode": "0x6A", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }` |
| gms_84 | `{ "opCode": "0x6A", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }` |
| gms_87 | `{ "opCode": "0x6D", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }` |
| gms_95 | `{ "opCode": "0x76", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }` |

(every entry carries a validator — the silent-drop gotcha. gms_61 gets **no**
`ClaimRequest` entry: the packet does not exist on that client.)

Writers, each with its §3.2 operations table where applicable:

| writer | v61 | v72 | v79 | v83 | v84 | v87 | v95 | options |
|---|---|---|---|---|---|---|---|---|
| SueCharacterResult | 0x34 | 0x34 | 0x34 | 0x37 | 0x37 | 0x37 | 0x37 | operations (sue table) |
| ClaimResult | — | 0x2A | 0x2A | 0x2D | 0x2D | 0x2D | 0x2C | operations (claim table) |
| ClaimAvailableTime | — | 0x2B | 0x2B | 0x2E | 0x2E | 0x2E | 0x2D | — |
| ClaimSvrStatusChanged | — | 0x2C | 0x2C | 0x2F | 0x2F | 0x2F | 0x2E | — |

Note the v61/v72/v79 clientbound opcodes sit a full region lower than the
v83+ ones (0x34 vs 0x37, 0x2A–0x2C vs 0x2D–0x2F) — do not port v83 values
across. Entries are inserted at their **sorted `opCode` position** in both
arrays; `tools/template-opcode-order-guard.sh` (CI-gated) rejects appending
them next to semantically-related entries.

**gms_12, gms_48, gms_92 and jms_185 templates get no entries.** jms is out of
scope per PRD (`SUE_CHARACTER` version-absent; claim UI stays dark via the §4.4
config gate). gms_48 is newly in this bucket for a *different* reason — it has
both clientbound receivers but neither send-site (PRD §2.1, verified), so
writers there would answer requests the client cannot make. gms_12 and gms_92
have **no registry file and no IDB** to verify opcodes against — inventing them
is forbidden, so they keep the feature disabled the same config-driven way.
Recorded as follow-ups (unblock when those IDBs exist).

A dispatcher-doc-style reference for the two operations tables is added under
`docs/packets/dispatchers/` only if `packet-audit operations --check` requires
it; otherwise the seed templates are the single source (verified during
implementation, not assumed).

### 9.3 Live-tenant config patch

Seed templates apply only at tenant creation. The task folder gets
`live-config-patch.md` documenting the PATCH of the handler + writer entries
into each existing gms-61 (sue pair only) and gms-72/79/83/84/87/95 tenant's socket config and the
atlas-channel restart (projection does not hot-reload handlers/writers).
New env vars for existing deployments: the two service-URLs on atlas-ban
(§5.3, fallback-safe) and the two capture tunables on atlas-messages (§7.2,
default-safe) — deployment manifests updated in `deploy/k8s/base/` for
atlas-messages' Redis connection env if peers require explicit wiring.

## 10. Packet Verification Plan (FR-6)

Per packet × version via the packet-verifier flow (decompile-derived read/write
order, byte-fixture test with `packet-audit:verify` marker + IDA address,
evidence record, matrix regeneration; artifacts committed together):

- **31 cells** targeted ✅ (registry rows currently all `csv-import` ❌), not a
  flat packets × versions product — the two mechanisms have different version
  spans (PRD §2.1):
  - `SueCharacterResult` × 7 (v61, v72, v79, v83, v84, v87, v95) = 7
  - `ClaimRequest`, `ClaimResult`, `ClaimAvailableTime`, `ClaimSvrStatusChanged`
    × 6 each (v72, v79, v83, v84, v87, v95) = 24
  jms cells stay ❌/⬜ (out of scope). v48 cells stay ⬜ and are *documented* as
  verified absences (PRD FR-6.5), not promoted.
- **v72/v79 `CLAIM_REQUEST` registry rows must be created before their cells can
  be verified** (PRD FR-6.4). The matrix currently renders both `⬜` because the
  registry has no row at all — a false negative, not a real absence. Both
  send-sites are already named in their IDBs (`CWvsContext::SendClaimRequest`
  @ `0x91f2b4` v72, @ `0x9711ff` v79), so this is registry-row + export-splice
  work only, with no IDB naming step. Opcodes read from the decompile:
  `COutPacket(105)` on v72, `COutPacket(104)` on v79.
- **v83 `CLAIM_REQUEST` send-site naming is in-scope implementation work**
  (FR-6.2): locate the function in the v83_Me IDB (port 13342; anchor via
  `CUIClaim::OnCreate` @ 0x7db17d xrefs and the `COutPacket::COutPacket(&p, 106)`
  constant), name it `CWvsContext::SendClaimRequest`, verify the encode order
  against the v95 shape, and splice the export surgically (never overwrite; the
  export is non-idempotent). An unresolvable fname is a stop-and-ask, never a
  substituted hash.
- v84/v87 derive from their IDBs per the standard flow; the v84 clientbound
  opcode-shift gotcha is already reflected in the STATUS.md rows used above.
- v61/v72/v79 cells derive from their own IDBs (ports per `idb_list`); their
  clientbound opcodes are a region lower than v83's (§9.2) so nothing may be
  ported across that boundary.
- `packet-audit gate-check --check` must stay clean: adding v72/v79 claim
  columns creates new adjacent-version pairs around the v79↔v83 opcode shift,
  and every gated divergence needs a verified fixture on **both** sides.

## 11. Error Handling Summary

| Failure | Behavior |
|---|---|
| Accused unresolvable (either mechanism) | No row persisted; `ERROR/NOT_FOUND` → sue result 1 / claim mode 0x42 |
| atlas-character unreachable | Treated as INTERNAL (not NOT_FOUND — do not misreport a working name as missing); sue generic / claim 0x41 |
| atlas-messages unreachable at create | Report persists with `server_transcript = null`; still CREATED |
| DB error on create | `ERROR/INTERNAL` → sue generic / claim 0x41 |
| Reporter offline when status arrives | Packet dropped silently; report already persisted |
| Oversized description/chat log | Truncated to caps + warning log; report persists |
| Redis unavailable (capture) | Warning log; chat flow unaffected; later reports get null transcript |
| PATCH invalid status / id mismatch / unknown id | 400 / 400 / 404 |
| Tenant config missing claim writers | Enable packets skipped (debug log); claim UI stays disabled client-side |

## 12. Testing

- **libs/atlas-packet**: golden byte-fixture + `pt.RoundTrip` per version for
  all five codecs (these double as the verifier fixtures).
- **atlas-ban `report`**: table-driven processor tests (create happy path,
  NOT_FOUND rejection, truncation, transcript-failure tolerance, status
  transitions incl. invalid value) against sqlite-style test DB as in `ban`;
  consumer handler tests with mocked character/chat clients; REST handler tests
  for the PATCH validation matrix. `report/mock` kept in sync with the
  interface.
- **atlas-channel**: handler tests (decode→command emission via captured
  producer), status-consumer mapping tests (table in §4.5), writer body tests.
- **atlas-messages**: capture tests (command lines excluded, PET/PINK_TEXT
  excluded, caps enforced), query merge/sort tests; atlas-redis bounded-append
  unit tests in the lib.
- **atlas-ui**: follow the existing Bans feature's testing level.
- **Gates** (CLAUDE.md): `go test -race ./...`, `go vet ./...`,
  `go build ./...` per changed module; `docker buildx bake` for atlas-channel,
  atlas-ban, atlas-messages; `tools/redis-key-guard.sh` from repo root;
  `GOWORK=off` only inside guard scripts per the workspace-footgun rules.
- **Live acceptance** per PRD §10 against a v83 tenant, plus a v72 tenant (the
  lowest full-feature column, and the one whose opcodes share no values with
  v83) and a v61 tenant (sue-only: sue round-trips, claim UI stays disabled).

## 13. Explicit Deferrals (→ docs/TODO.md, no code TODOs)

- Quota / mesos-cost enforcement (sue code 2; claim modes 0x43/0x45/0x47/0x48 +
  real `remaining` counts) — plumbing already expressive.
- Accused-notification codes (sue 3 / claim mode 3).
- gms-12 / gms-92 enablement (blocked on registry files + IDBs) and jms claim
  support.
- gms-48 stays disabled permanently, not pending: neither request packet exists
  in that client (PRD §2.1/FR-6.5). Nothing to unblock.
- gms-61 claim support is likewise a permanent absence, not a deferral; v61
  ships sue only.
- Scheduled claim availability windows (writer already carries the hours).
