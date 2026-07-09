# Player Reports (Sue / Claim System) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-09
---

## 1. Overview

MapleStory clients of the Atlas-supported era ship two player-reporting mechanisms, and Atlas currently implements neither end-to-end:

1. **Sue** — the lightweight `/`-command report (`CField::SendChatMsgSlash#SueCharacter`). The serverbound packet is already decoded and T1-verified for all four GMS versions, but the atlas-channel handler is a decode-and-log stub: no persistence, no processor call, and the `SUE_CHARACTER_RESULT` clientbound packet (the reporter's feedback line) is unimplemented (STATUS.md ❌ for all versions).
2. **Claim** — the full harassment-report window (`CUIClaim`), which is what Cosmic's `ReportHandler` actually corresponds to. It submits a target name, a reason type, a free-text description, and — for chat-based claims — a **client-supplied chat log**. None of its four packets (`CLAIM_REQUEST` serverbound; `CLAIM_RESULT`, `CLAIM_AVAILABLE_TIME`, `CLAIM_STATUS_CHANGED` clientbound) are implemented. Critically, the client keeps the claim UI disabled unless the server has sent claim-server status/availability packets, so the feature is invisible to players today.

This task wires both mechanisms into a persistent report store owned by **atlas-ban** (which already owns the moral neighborhood: `ban` and `history` domains), gives GMs an atlas-ui admin surface to list and resolve reports, and adds server-side chat capture so a report's client-supplied chat log can be corroborated against a server-recorded transcript (client logs are trivially forgeable).

All packet semantics below were IDA-verified against the v83 and v95 clients on 2026-07-09; see `packet-findings.md` in this folder for decompile evidence (function names, addresses, instances).

## 2. Goals

Primary goals:

- Persist every player report (sue and claim) in atlas-ban, tenant-scoped, with a reviewable lifecycle (`open` → `reviewed` → `actioned`).
- Send the reporter the correct result packet with IDA-verified result codes (success, unable-to-locate, failure), for both mechanisms.
- Enable the claim UI in the client by emitting `CLAIM_STATUS_CHANGED` / `CLAIM_AVAILABLE_TIME`.
- Store the client-supplied chat log verbatim on chat claims, and attach a server-side transcript snapshot captured from the existing chat event stream for corroboration.
- Give GMs an atlas-ui admin page to list, inspect, and resolve reports.

Non-goals:

- **Abuse controls / quotas** — the clients support per-day (sue code 2) and per-week (claim modes 0x45/0x47/0x48) quota messaging, and a mesos cost (claim mode 0x43). v1 enforces none of these. They are deliberately deferred; the result-code plumbing must be able to express them later (the writer accepts the code as a parameter rather than hard-coding success).
- **Automated punishment** — no report-driven ban creation. GMs act manually via existing tooling.
- **In-game GM notification** — stored-for-review only; no whisper/notice push to online GMs.
- **Notifying the accused** — both protocols include a "you have been reported" code (sue 3, claim mode 3) targeted at the *accused*; v1 does not send it.
- **jms** — `SUE_CHARACTER` is version-absent in jms (no send-site). Claim opcodes exist in the jms matrix but jms is out of scope for v1; the claim UI simply stays disabled there (no status packet sent).

## 3. User Stories

- As a player, I want to report another player via the report window (with reason, description, and — for harassment — my chat log) so that a GM can review the incident.
- As a player, I want to sue a player via the `/`-command so that quick reports are possible without the full window.
- As a player, I want immediate feedback (success / "unable to locate the user") so I know my report registered.
- As a GM, I want to list open reports for my tenant, read the description, the client-supplied chat log, and the server-side transcript, so that I can judge the claim's credibility.
- As a GM, I want to mark a report `reviewed` or `actioned` so the queue reflects real state.

## 4. Functional Requirements

### 4.1 Sue flow (atlas-channel)

- FR-1.1: `SueCharacterHandleFunc` MUST forward decoded sue reports to a processor instead of only logging. Legacy versions (v83/v84/v87) supply the accused **character id**; v95 supplies a **sub-command string** — both forms must be persisted with whatever identity field the wire provided, plus the resolved counterpart (see FR-3.4).
- FR-1.2: The channel MUST reply with `SUE_CHARACTER_RESULT` (new writer, opcode 0x037 all four versions) carrying a single result byte with the verified semantics: `0` success, `1` unable to locate the user, `2` daily-limit (reserved, unused in v1), `3` reported-notice to accused (reserved, unused in v1), any other value = generic failure.
- FR-1.3: v1 result policy: `0` after successful persistence; `1` if the accused cannot be resolved to a character in the tenant; generic failure code on internal error.

### 4.2 Claim flow (atlas-channel)

- FR-2.1: New serverbound codec + handler for `CLAIM_REQUEST` (v83 0x06A, v84 0x06A, v87 0x06D, v95 0x076). Body (v95-verified; v83 send-site must be byte-verified during implementation — see FR-6): `bChatClaim (byte)`, `targetName (string)`, `reasonType (byte)`, `description (string)`, and — only when `bChatClaim == 1` — `chatLog (string)`.
- FR-2.2: New clientbound writer `CLAIM_RESULT` (v83/v84/v87 0x02D, v95 0x02C) supporting at minimum the verified modes: `2` success (byte `hasRemaining` + int32 `remainingCount` follow), `0x41` try-again-later (generic failure), `0x42` re-check character name (target not found). Remaining verified modes (`3`, `0x43`, `0x44`, `0x45`, `0x47`, `0x48`) MUST be expressible by the writer (mode is a parameter) but are not emitted by v1 logic.
- FR-2.3: New clientbound writers `CLAIM_AVAILABLE_TIME` (2 bytes: open hour, close hour) and `CLAIM_STATUS_CHANGED` (1 byte: connected flag). The channel MUST send `CLAIM_STATUS_CHANGED(1)` and an availability window to each session so the client enables the claim UI. v1 sends an always-open window (`0, 0` — verified: the client treats `open == close == 0` as always-available). The exact emission point (post-login/field-enter sequence) is a design-phase decision.
- FR-2.4: v1 result policy: mode `2` with `hasRemaining=1, remainingCount=<large constant>` on success (quotas are not tracked); mode `0x42` when the target name does not resolve in the tenant; mode `0x41` on internal error.
- FR-2.5: Mode/result bytes MUST be config-resolved per DOM-25 where version-variant, never hard-coded in domain logic.

### 4.3 Report persistence (atlas-ban)

- FR-3.1: New `report` domain in atlas-ban following the existing entity/model/builder/processor/provider/rest/resource pattern (mirror the `ban` package).
- FR-3.2: Reports are created via a Kafka command emitted by atlas-channel and consumed by atlas-ban (standard command topic + curried consumer registration). Persistence success/failure flows back to the channel as a status event so the channel can emit the correct result packet.
- FR-3.3: A report record captures: report kind (`sue` | `claim`), reporter character id, accused identity (see FR-3.4), reason/flag byte as sent on the wire, description text (sue `reason` / claim `description`), client chat log (nullable, claim-only), server transcript snapshot (nullable), status, timestamps.
- FR-3.4: Accused identity resolution: claim supplies a name, legacy sue supplies a character id, v95 sue supplies a sub-command string. The processor MUST resolve name↔id via atlas-character lookup where possible and store both; when resolution fails the report is not persisted and the not-found result code is returned (sue `1` / claim `0x42`).
- FR-3.5: Report status lifecycle: `open` (initial) → `reviewed` → `actioned`. Status changes via REST PATCH only (no Kafka path in v1). Transitions are unrestricted among the three states in v1.

### 4.4 Server-side chat capture (atlas-messages)

- FR-4.1: atlas-messages (or a consumer of its existing `EnvEventTopicChat` event stream — final home is a design-phase decision, defaulting to atlas-messages per owner preference) MUST retain a bounded, tenant-scoped buffer of recent chat lines per character (content, sender, timestamp, map/field context).
- FR-4.2: The buffer MUST be queryable by the report pipeline: "recent lines involving characters A and B" at report time. The snapshot is attached to the report record at creation; the buffer itself is short-retention working state, not an archive.
- FR-4.3: Retention bound (line count and/or age) is configurable; default sized to comfortably cover the client's own claim-log window. Exact figures are a design-phase decision.
- FR-4.4: Only chat visible in the reporting context needs capture in v1 (general/field chat at minimum). Whisper/party/guild coverage is a design-phase scoping decision — the PRD requires general chat as the floor.

### 4.5 GM review surface (atlas-ui)

- FR-5.1: New admin page listing reports for the active tenant: kind, reporter, accused, reason, status, created-at; filterable by status.
- FR-5.2: Detail view shows description, client-supplied chat log, and server transcript snapshot side by side.
- FR-5.3: GM can PATCH status (`reviewed`, `actioned`) from the detail view. Requests use the JSON:API envelope (known gotcha for input-handler endpoints).

### 4.6 Packet verification & registry hygiene

- FR-6.1: Every new packet (1 sue clientbound + 4 claim) MUST go through the packet-verifier flow per version: decompiled read/write order, byte-fixture test with `packet-audit:verify` marker, evidence record, matrix regeneration. Static claims are not verification.
- FR-6.2: The v83 `CLAIM_REQUEST` send-site is currently **unnamed** in the v83 IDB (`csv-import` provenance); naming it and confirming the v83 body against the v95 shape is part of this task, not a deferrable gap.
- FR-6.3: New handler/writer opcodes MUST be added to the seed templates for all supported versions AND documented as a live-tenant config patch (seed templates only apply at tenant creation — known gotcha). Every new `socket.handlers` entry MUST carry a validator (`LoggedInValidator`), or it is silently dropped.

## 5. API Surface

New JSON:API resources in atlas-ban (final routes at design time; shapes below are the contract):

- `GET /api/rpts/reports?filter[status]=open` — list reports for the tenant (tenant from header context, as elsewhere). Resource type `reports`.
- `GET /api/rpts/reports/{reportId}` — single report with full detail (chat log + transcript included as attributes).
- `PATCH /api/rpts/reports/{reportId}` — update `status`. Body is a JSON:API envelope; invalid transitions/values → 400; unknown id → 404.

Attributes (response): `kind`, `reporterId`, `reporterName`, `accusedId`, `accusedName`, `reasonType`, `description`, `chatLog` (nullable), `serverTranscript` (nullable, array of `{timestamp, senderId, senderName, text}`), `status`, `createdAt`, `updatedAt`.

Kafka (names finalized at design time, following existing command/event conventions):

- Command `REPORT` (channel → ban): create report (kind, reporter, accused identity as sent, reason, description, chat log).
- Status event (ban → channel): created / error, correlated so the channel session can emit the result packet.

atlas-messages query surface (internal): provider for "recent chat lines involving characters A and B" — REST or direct-consumer design decided in the design phase.

## 6. Data Model

New table in atlas-ban (GORM AutoMigrate, following `ban`/`history` conventions):

`reports`
- `id` uuid PK (surrogate — never a business-value PK; known multi-tenant collision gotcha)
- `tenant_id` uuid, indexed; unique constraints tenant-scoped
- `kind` string enum: `sue` | `claim`
- `reporter_id` uint32 (character id), `reporter_name` string
- `accused_id` uint32 (nullable when unresolved at write time — should not occur in v1 since unresolved → rejected), `accused_name` string
- `reason_type` smallint (wire byte: sue `flag` / claim `nType`)
- `description` text
- `chat_log` text nullable (client-supplied, claim chat-claims only, stored verbatim)
- `server_transcript` jsonb nullable (snapshot at creation)
- `status` string enum: `open` | `reviewed` | `actioned`, default `open`, indexed with `tenant_id`
- `created_at`, `updated_at`

Chat-capture storage (atlas-messages side) is working state, not part of the durable model; its shape (in-memory registry vs Redis via `libs/atlas-redis` vs table) is a design decision. If Redis is used, it MUST go through `libs/atlas-redis` (redis-key-guard invariant).

## 7. Service Impact

- **atlas-channel** — wire `SueCharacterHandleFunc` to a processor + Kafka command; new `ClaimRequest` handler; new writers (`SueCharacterResult`, `ClaimResult`, `ClaimAvailableTime`, `ClaimSvrStatusChanged`); emit claim-enable packets in the session setup sequence; consume ban's status event to deliver results.
- **libs/atlas-packet** — new clientbound codecs for the 5 packets; serverbound `ClaimRequest` codec; byte-fixture tests per version.
- **atlas-ban** — new `report` domain (entity/model/processor/provider/rest/resource/producer + consumer registration + mock); AutoMigrate addition.
- **atlas-messages** — chat capture buffer + query provider (or a documented alternative consumer service if design finds a better home).
- **atlas-ui** — reports admin page (list + detail + status PATCH).
- **tenant seed templates + live config** — handler/writer opcode entries for all supported versions; live-tenant patch documented in the task; validators on every handler entry.
- **Dockerfile / go.work** — no new shared libs anticipated; if one appears, follow the two-COPY-lines + go.work rule.

## 8. Non-Functional Requirements

- **Multi-tenancy**: every read/write tenant-scoped via `tenant.MustFromContext`; Kafka messages carry tenant headers per existing convention.
- **Size limits**: client-supplied strings (description, chat log) MUST be length-capped server-side before persistence (caps chosen at design time) — the wire allows arbitrary strings and this is user-generated content.
- **Observability**: report creation, rejection (unresolved target), and status changes logged at info with tenant/reporter/accused fields.
- **Performance**: chat capture must not add per-message synchronous I/O in the hot chat path beyond the existing Kafka emit; the buffer update rides the existing event stream.
- **Security**: report REST endpoints are admin-surface only (same posture as existing atlas-ban resources); chat log content is rendered as text in the UI (no HTML injection).

## 9. Open Questions

- Where exactly in the login/field-enter sequence the claim-enable packets are sent (design phase; must precede the player opening the claim UI).
- Final home and mechanism for chat capture (atlas-messages internal store vs dedicated consumer; in-memory vs Redis vs table) and concrete retention bounds.
- Whether whisper/party/guild chat is captured in v1 or general/field chat only.
- Claim availability window: confirmed always-open for v1; revisit if a tenant ever wants scheduled windows (`CLAIM_AVAILABLE_TIME` already carries the hours).

## 10. Acceptance Criteria

- [ ] Sue report from a live client persists a `sue` report in atlas-ban and the reporter sees the success chat-log line (result 0); suing a nonexistent name/id yields result 1 and persists nothing.
- [ ] Claim UI opens in the client (status/availability packets sent); submitting a chat claim persists a `claim` report with the client chat log stored verbatim and a server transcript snapshot attached; reporter sees the success notice with remaining-count text.
- [ ] Claim against an unresolvable name yields mode 0x42 and persists nothing; internal errors yield the generic failure code for each mechanism.
- [ ] All 5 new packets byte-fixtured and matrix-promoted per version via the packet-verifier flow, including the named-and-verified v83 `CLAIM_REQUEST` send-site.
- [ ] `GET`/`PATCH` report endpoints behave per §5 including error cases; atlas-ui page lists, filters by status, shows detail, and updates status.
- [ ] Seed templates updated for all supported versions with validators on every handler entry; live-tenant config patch steps documented in the task folder.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake` clean for every touched service; `tools/redis-key-guard.sh` clean.
- [ ] Deferred quota/mesos enforcement recorded as an explicit follow-up in docs/TODO.md (not code TODOs).
