# Player Reports (Sue / Claim System) — Product Requirements Document

Version: v2
Status: Draft
Created: 2026-07-09
Updated: 2026-08-04 — extended to the four client versions added to the
coverage matrix since v1 (gms_v48, gms_v61, gms_v72, gms_v79); see §2.1.
---

## 1. Overview

MapleStory clients of the Atlas-supported era ship two player-reporting mechanisms, and Atlas currently implements neither end-to-end:

1. **Sue** — the lightweight `/`-command report (`CField::SendChatMsgSlash#SueCharacter`). The serverbound packet is already decoded and T1-verified for seven GMS versions (v61 through v95 — it is genuinely version-absent on v48, see §2.1), but the atlas-channel handler is a decode-and-log stub: no persistence, no processor call, and the `SUE_CHARACTER_RESULT` clientbound packet (the reporter's feedback line) is unimplemented (STATUS.md ❌ for all versions).
2. **Claim** — the full harassment-report window (`CUIClaim`), which is what Cosmic's `ReportHandler` actually corresponds to. It submits a target name, a reason type, a free-text description, and — for chat-based claims — a **client-supplied chat log**. None of its four packets (`CLAIM_REQUEST` serverbound; `CLAIM_RESULT`, `CLAIM_AVAILABLE_TIME`, `CLAIM_STATUS_CHANGED` clientbound) are implemented. Critically, the client keeps the claim UI disabled unless the server has sent claim-server status/availability packets, so the feature is invisible to players today.

This task wires both mechanisms into a persistent report store owned by **atlas-ban** (which already owns the moral neighborhood: `ban` and `history` domains), gives GMs an atlas-ui admin surface to list and resolve reports, and adds server-side chat capture so a report's client-supplied chat log can be corroborated against a server-recorded transcript (client logs are trivially forgeable).

All packet semantics below were IDA-verified against the v83 and v95 clients on 2026-07-09, and re-swept against the v48/v61/v72/v79 IDBs on 2026-08-04 when those columns entered the coverage matrix; see `packet-findings.md` in this folder for decompile evidence (function names, addresses, instances).

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

## 2.1 Version Support

When this PRD was first written the coverage matrix tracked five columns and
this task targeted four (v83/v84/v87/v95). The matrix now tracks **nine**
(`docs/packets/PROCESS.md` §Version set), adding **gms_v48, gms_v61, gms_v72,
gms_v79** — all four of which now also ship tenant seed templates. This task
covers every version whose client can actually *submit* a report. Support is
per-mechanism, because the two mechanisms did not arrive in the client at the
same version:

| version | sue submit | sue result | claim submit | claim clientbound trio | scope |
|---|---|---|---|---|---|
| gms_v48 | **absent** | 0x2C | **absent** | 0x25 / 0x26 / 0x27 | **none** |
| gms_v61 | 0x68 | 0x34 | **absent** | 0x2A / 0x2B / 0x2C | sue only |
| gms_v72 | 0x71 | 0x34 | **0x69** | 0x2A / 0x2B / 0x2C | full |
| gms_v79 | 0x70 | 0x34 | **0x68** | 0x2A / 0x2B / 0x2C | full |
| gms_v83 | 0x72 | 0x37 | 0x6A | 0x2D / 0x2E / 0x2F | full |
| gms_v84 | 0x72 | 0x37 | 0x6A | 0x2D / 0x2E / 0x2F | full |
| gms_v87 | 0x75 | 0x37 | 0x6D | 0x2D / 0x2E / 0x2F | full |
| gms_v95 | 0x7E | 0x37 | 0x76 | 0x2D / 0x2E / 0x2F | full |
| jms_v185 | n-a | n-a | 0x65 | 0x2A / 0x2B / 0x2C | out of scope (§2) |

Sue-submit and clientbound opcodes are as generated in
`docs/packets/audits/STATUS.md`. The two **bold new claim-submit opcodes are a
registry correction produced by this task** — see §4.6 FR-6.4.

Three consequences drive the scoping above:

- **v48 gets no template entries at all.** Its client can receive
  `SUE_CHARACTER_RESULT` and the claim trio, but has no way to send either
  request, so those writers would answer packets that never arrive. Verified,
  not assumed: the full decompile of v48 `CField::SendChatMsgSlash` @ `0x4c3e96`
  contains exactly two `COutPacket` sites — opcode 152 (`Encode1`) and opcode 40
  (`EncodeStr`) — and neither is the `Encode4/Encode1/EncodeStr` sue shape.
  v48 therefore joins jms and gms_92 as a config-disabled version.
- **v61 gets sue entries only.** Sue submit is T1-verified there, but no claim
  send-site exists, so sending `CLAIM_STATUS_CHANGED` would enable a UI that
  cannot submit. Claim stays disabled by omitting its template entries.
- **v72 and v79 get the full feature**, including claim — see FR-6.4 for why
  this contradicts the matrix as currently generated.

gms_12 and gms_92 ship seed templates but have no registry file and no IDB, so
their opcodes cannot be verified; they stay disabled exactly as gms_92 already
was (unchanged from v1, recorded as a deferral).

## 3. User Stories

- As a player, I want to report another player via the report window (with reason, description, and — for harassment — my chat log) so that a GM can review the incident.
- As a player, I want to sue a player via the `/`-command so that quick reports are possible without the full window.
- As a player, I want immediate feedback (success / "unable to locate the user") so I know my report registered.
- As a GM, I want to list open reports for my tenant, read the description, the client-supplied chat log, and the server-side transcript, so that I can judge the claim's credibility.
- As a GM, I want to mark a report `reviewed` or `actioned` so the queue reflects real state.

## 4. Functional Requirements

### 4.1 Sue flow (atlas-channel)

- FR-1.1: `SueCharacterHandleFunc` MUST forward decoded sue reports to a processor instead of only logging. Legacy versions (v61/v72/v79/v83/v84/v87) supply the accused **character id**; v95 supplies a **sub-command string** — both forms must be persisted with whatever identity field the wire provided, plus the resolved counterpart (see FR-3.4). The legacy `Encode4(characterId) / Encode1(flag) / EncodeStr(reason)` body is unchanged across v61–v87 (v79 send-site @ `0x51825e` re-confirmed 2026-08-04), so the existing serverbound codec covers the new columns without a version branch.
- FR-1.2: The channel MUST reply with `SUE_CHARACTER_RESULT` (new writer; opcode 0x34 on v61/v72/v79 and 0x37 on v83/v84/v87/v95 — config-resolved per §2.1, never hard-coded) carrying a single result byte with the verified semantics: `0` success, `1` unable to locate the user, `2` daily-limit (reserved, unused in v1), `3` reported-notice to accused (reserved, unused in v1), any other value = generic failure.
- FR-1.3: v1 result policy: `0` after successful persistence; `1` if the accused cannot be resolved to a character in the tenant; generic failure code on internal error.

### 4.2 Claim flow (atlas-channel)

- FR-2.1: New serverbound codec + handler for `CLAIM_REQUEST` (v72 0x069, v79 0x068, v83 0x06A, v84 0x06A, v87 0x06D, v95 0x076). Body (v95-verified; v72 and v79 verified 2026-08-04 — see FR-6.4; v83 send-site must be byte-verified during implementation — see FR-6.2): `bChatClaim (byte)`, `targetName (string)`, `reasonType (byte)`, `description (string)`, and — only when `bChatClaim == 1` — `chatLog (string)`. The body is identical across v72–v95, so one codec with no version branch covers all six columns.
- FR-2.2: New clientbound writer `CLAIM_RESULT` (v72/v79 0x02A, v83/v84/v87 0x02D, v95 0x02C) supporting at minimum the verified modes: `2` success (byte `hasRemaining` + int32 `remainingCount` follow), `0x41` try-again-later (generic failure), `0x42` re-check character name (target not found). Remaining verified modes (`3`, `0x43`, `0x44`, `0x45`, `0x47`, `0x48`) MUST be expressible by the writer (mode is a parameter) but are not emitted by v1 logic.
- FR-2.3: New clientbound writers `CLAIM_AVAILABLE_TIME` (2 bytes: open hour, close hour; v72/v79 0x02B, v83/v84/v87 0x02E, v95 0x02D) and `CLAIM_STATUS_CHANGED` (1 byte: connected flag; v72/v79 0x02C, v83/v84/v87 0x02F, v95 0x02E). The channel MUST send `CLAIM_STATUS_CHANGED(1)` and an availability window to each session so the client enables the claim UI. v1 sends an always-open window (`0, 0` — verified: the client treats `open == close == 0` as always-available). The exact emission point (post-login/field-enter sequence) is a design-phase decision.
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

- FR-6.1: Every new packet (1 sue clientbound + 4 claim) MUST go through the packet-verifier flow per version: decompiled read/write order, byte-fixture test with `packet-audit:verify` marker, evidence record, matrix regeneration. Static claims are not verification. Per §2.1 the target is **31 cells**, not 20: `SueCharacterResult` × 7 (v61–v95) plus each of the four claim packets × 6 (v72–v95).
- FR-6.2: The v83 `CLAIM_REQUEST` send-site is currently **unnamed** in the v83 IDB (`csv-import` provenance); naming it and confirming the v83 body against the v95 shape is part of this task, not a deferrable gap.
- FR-6.3: New handler/writer opcodes MUST be added to the seed templates for every version in scope per §2.1 AND documented as a live-tenant config patch (seed templates only apply at tenant creation — known gotcha). Every new `socket.handlers` entry MUST carry a validator (`LoggedInValidator`), or it is silently dropped. Entries go at their sorted `opCode` position — `tools/template-opcode-order-guard.sh` enforces strictly ascending order in both arrays.
- FR-6.4: **Registry correction — `CLAIM_REQUEST` on v72 and v79.** `docs/packets/audits/STATUS.md` currently renders both cells `⬜` (n-a), which is a false negative: the send-site exists and is already named in both IDBs — `CWvsContext::SendClaimRequest` @ `0x91f2b4` (v72) and @ `0x9711ff` (v79). Both build the v95 body exactly: `COutPacket(105)` / `COutPacket(104)` respectively, then `Encode1(bChatClaim)`, `EncodeStr(targetName)`, `Encode1(reasonType)`, `EncodeStr(description)`, and `EncodeStr(chatLog)` only when `bChatClaim` is set. This task MUST add the two registry rows and splice both IDA exports so the matrix stops reporting the feature as absent. Because the functions are already named, no IDB naming work is required here (unlike FR-6.2's v83 case).
- FR-6.5: **Verified absences.** The `⬜` cells this task relies on for scoping MUST be recorded in `packet-findings.md` with their evidence rather than inherited as assumptions: v48 `SUE_CHARACTER` (full decompile of `CField::SendChatMsgSlash` @ `0x4c3e96` — no sue send-site), and `CLAIM_REQUEST` on v48/v61 (no `CUIClaim` class, no named send-site, no `COutPacket(96)` in the CWvsContext region, and the layout slot immediately preceding `OnClaimResult` — which holds `SendClaimRequest` on v72/v79 — contains only a 0x2d-byte function). These are discovery-pass negatives; a nonexistent packet cannot carry a byte-fixture.

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
- [ ] All 5 new packets byte-fixtured and matrix-promoted per version via the packet-verifier flow — 31 cells per FR-6.1 — including the named-and-verified v83 `CLAIM_REQUEST` send-site.
- [ ] v72 and v79 `CLAIM_REQUEST` registry rows added and exports spliced (FR-6.4); `packet-audit matrix --check` exits 0 and those two cells no longer render `⬜`.
- [ ] The v48 sue and v48/v61 claim absences are recorded in `packet-findings.md` with decompile evidence (FR-6.5).
- [ ] `GET`/`PATCH` report endpoints behave per §5 including error cases; atlas-ui page lists, filters by status, shows detail, and updates status.
- [ ] Seed templates updated for every version in scope per §2.1 (sue-only for gms_61; full for gms_72/79/83/84/87/95; none for gms_48/92/12 and jms) with validators on every handler entry; `tools/template-opcode-order-guard.sh` clean; live-tenant config patch steps documented in the task folder.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake` clean for every touched service; `tools/redis-key-guard.sh` clean.
- [ ] Deferred quota/mesos enforcement recorded as an explicit follow-up in docs/TODO.md (not code TODOs).
