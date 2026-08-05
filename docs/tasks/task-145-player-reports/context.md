# Task 145 — Player Reports: Implementation Context

Companion to `plan.md`. Summarizes key files, decisions, and dependencies so an
executor (or reviewer) can orient without re-reading the whole design.

## Inputs

- `prd.md` — requirements (v1 scope; quotas/mesos/accused-notification/jms deferred)
- `design.md` — architecture + all design-phase decisions (authoritative)
- `packet-findings.md` — IDA-verified packet semantics (v83 @ port 13342, v95 @ 13341)
- `plan.md` — 25 bite-sized TDD tasks

## Architecture in one paragraph

atlas-channel decodes `SUE_CHARACTER` (existing codec, stub handler) and the new
`CLAIM_REQUEST`, emits a `CREATE` command on `COMMAND_TOPIC_REPORT`; atlas-ban's
new `report` domain consumes it, resolves the accused via atlas-character REST,
snapshots a chat transcript from atlas-messages REST (best-effort), persists the
`reports` row, and emits `EVENT_TOPIC_REPORT_STATUS`; atlas-channel's status
consumer maps `(kind, status, errorCode)` → result packet and delivers it via
`IfPresentByCharacterId` (buddy-consumer pattern, no correlation token). Claim UI
is enabled per-session at bootstrap by `ClaimSvrStatusChanged(1)` +
`ClaimAvailableTime(0,0)`; tenants without those writers in config (jms, gms-92)
skip the sends — config presence IS the feature gate. Chat capture rides the
existing atlas-messages chat path into a bounded per-character Redis sorted set
(new `AddBounded` on `libs/atlas-redis` `TenantKeyedSortedSet`).

## Key decisions (already made — do not relitigate)

| Decision | Choice | Why |
|---|---|---|
| Report store owner | atlas-ban | owns ban/history; complete domain template to mirror |
| Chat buffer | Redis via `libs/atlas-redis` (extend lib with `AddBounded`) | atlas-messages runs 2 replicas; in-memory fails multi-replica; guard bans raw go-redis |
| Transcript attach point | atlas-ban pulls from atlas-messages REST at create | snapshot-once; outage degrades to null transcript, never fails the report |
| Result delivery | async status event → channel consumer | matches buddy/whisper/note round trips |
| ClaimResult shape | discrete structs (Success/Notice), NOT a dispatcher family | only mode 2 carries payload (IDA v83+v95); no families.yaml/dispatcher-lint |
| Result bytes | config-resolved via `WithResolvedCode("operations", key, …)` | DOM-25 / config-drive-all-modes (task-103 precedent), even though version-stable |
| Sue identity forms | legacy = accusedId; v95 = subCommand treated as target name | resolution + rejection happens in ban; channel just forwards both fields |
| Caps | description 2000 chars, chat log 16384 bytes; truncate+log | user-generated content; truncated report beats vanished report |
| Capture scope | GENERAL/BUDDY/PARTY/GUILD/ALLIANCE/WHISPER/MESSENGER; not PET/PINK_TEXT/commands | one code path; exceeds PRD floor at ~zero cost |
| Capture bounds | 900s window / 200 lines per character (env-tunable), key TTL = window | covers client's own claim-log ring; idle buffers evaporate |
| remaining=100 | named constant, display-only | quotas untracked in v1 |
| REST | `/api/reports` flat, `?status=` plain query param | matches `bans?type=`; supersedes PRD's `/api/rpts` + `filter[status]` sketch |
| PATCH semantics | id-mismatch 400, invalid status 400 (ErrInvalidStatus), unknown 404 (gorm.ErrRecordNotFound) | atlas-notes UpdateNoteHandler pattern |
| DI for processor tests | `NewProcessorWithClients(l, ctx, db, charP, chatP)` in processor.go | no test-only constructors / testhelpers files allowed |

## Wire facts (IDA-verified — do not re-derive)

Opcode columns below are `v61/v72/v79 | v83/v84/v87/v95` (PRD §2.1). Bodies and
mode values are version-stable across each packet's span; only opcodes move.

- SUE_CHARACTER_RESULT: opcode 0x34 (v61/v72/v79) / 0x37 (v83/v84/v87/v95); 1 byte; 0 success, 1 not-found, 2 daily-limit, 3 accused-notice, other = generic failure; chat-log line, not modal. Present-but-unreachable on v48 (0x2C — no sue send-site there); n-a on jms.
- CLAIM_RESULT: 0x2A (v72/v79) / 0x2D/0x2D/0x2D/0x2C (v83/v84/v87/v95); mode byte; ONLY mode 2 has payload (`byte hasRemaining, int32 remaining`); notices: 3, 0x41-0x45, 0x47, 0x48; modal; unknown modes silently ignored.
- CLAIM_AVAILABLE_TIME: 0x2B (v72/v79) / 0x2E/0x2E/0x2E/0x2D; `byte open, byte close`; 0/0 = always available (explicit client branch).
- CLAIM_STATUS_CHANGED: 0x2C (v72/v79) / 0x2F/0x2F/0x2F/0x2E; `byte connected`; required for the claim UI to open at all.
- CLAIM_REQUEST (serverbound): 0x69 (v72) / 0x68 (v79) / 0x6A/0x6A/0x6D/0x76; `byte bChatClaim, string target, byte nType, string description, [string chatLog if bChatClaim==1]`; v95-verified, and v72 (`0x91f2b4`) / v79 (`0x9711ff`) verified 2026-08-04. **v83 send-site is unnamed in the v83_Me IDB — naming + byte-verifying it is Task 23, in scope. v72/v79 are already named but have no registry row — adding it is Task 23b, in scope.**
- **Version-absent, verified (Task 23c):** sue on v48; claim on v48 and v61. The claim mechanism enters the GMS client between v61 and v72.
- Client pre-gates claims: needs `m_bClaimSvrConnected`, an open window, and (chat claims) the target present in the local chat log.

## Pattern files to mirror (read before implementing)

| New code | Model |
|---|---|
| packet codecs | `libs/atlas-packet/field/serverbound/sue_character.go` (+`_test.go`), `libs/atlas-packet/note/clientbound/operation.go` |
| code resolution | `libs/atlas-packet/resolve.go` (`WithResolvedCode`, factory returns `packet.Encoder`) |
| ban report domain | `services/atlas-ban/atlas.com/ban/ban/*` (entity/model/builder/provider/administrator/processor/producer/resource/rest) |
| ban consumer | `services/atlas-ban/atlas.com/ban/kafka/consumer/ban/consumer.go` |
| ban REST helpers | `services/atlas-ban/atlas.com/ban/rest/handler.go` (RegisterHandler/RegisterInputHandler/ParseBanId) |
| PATCH handler | atlas-notes `note/resource.go` `UpdateNoteHandler` (id-match 400 / 404 / echo resource) |
| mock | `services/atlas-notes/atlas.com/notes/note/mock/processor.go` (XxxFunc fields) |
| channel domain emit | `services/atlas-channel/atlas.com/channel/buddylist/{processor,producer}.go` |
| channel status consumer | `services/atlas-channel/atlas.com/channel/kafka/consumer/buddylist/consumer.go` (IsWorld guard → IfPresentByCharacterId → Announce) |
| session bootstrap sends | `CH/kafka/consumer/session/consumer.go:155-338` `processStateReturn` goroutine blocks |
| operations-table writer | `CH/socket/writer/world_message.go` (ResolveCode "operations") |
| redis registry | `services/atlas-merchant/atlas.com/merchant/visitor/registry.go` (InitRegistry/GetRegistry) |
| redis sorted set + tests | `libs/atlas-redis/keyed_sorted_set.go` + `_test.go` (miniredis, keyPrefix reset) |
| messages capture point | `services/atlas-messages/atlas.com/messages/message/processor.go` (after command-registry check) |
| ban-side REST clients | `services/atlas-messages/atlas.com/messages/character/{requests,processor}.go` (RootUrl, SliceProvider+First) |
| UI feature | Bans: `BansPage.tsx`, `bans-columns.tsx`, `BanDetailPage.tsx`, `bans.service.ts`, `useBans.ts`, `types/models/ban.ts`, `components/features/bans/*` |

## Registration points (exact locations)

- `BAN/main.go:57` migrations; `:60-67` consumer blocks; `:76-77` route initializers.
- `CH/main.go:174` area `InitConsumers` block; `:416-437` `register(x.InitHandlers(fl)(sc)(wp)(rh))` block; `produceWriters()` at `:608-793`; `produceHandlers()` at `:795-902` (`SueCharacterHandle` at `:836`).
- `MSG/main.go:77-83` REST server block (needs new Server info struct + first domain route); `:41` area for `redis.Connect` + `chat.InitRegistry`.
- UI: `App.tsx:79-80` (bans routes), `app-sidebar.tsx:64-68` (Security group), `lib/breadcrumbs/routes.ts` `ROUTE_CONFIGS`.

## Config / deploy surface

- Seed templates: `services/atlas-configurations/seed-data/templates/template_gms_{83,84,87,95}_1.json` — handlers entry shape `{"opCode","validator","handler"}`, writers `{"opCode","writer","options":{"operations":{...}}}` (hex-string values, see FameResponse). **gms_92 + jms_185: no entries.**
- Operations guard: `docs/packets/dispatchers/*.yaml` + `go run ./tools/packet-audit operations --check` (templates dir + dispatchers dir defaults are correct from repo root).
- Topics: `deploy/k8s/base/env-configmap.yaml` is source of truth; regenerate overlays with `deploy/k8s/overlays/pr/scripts/gen-topic-config.sh`.
- Ingress: `deploy/shared/routes.conf` + `deploy/compose/routes.conf` (bans block at shared:312), then `deploy/scripts/sync-k8s-ingress-routes.sh`. The messages chat-history endpoint gets NO ingress entry (server-to-server only).
- `REDIS_URL` already in the shared `atlas-env` configmap (env-configmap.yaml:151) — atlas-messages needs no manifest change for it.
- Live tenants need the config PATCH + channel restart (`live-config-patch.md`, Task 18) — seed templates only apply at tenant creation.

## Gotchas that WILL bite if forgotten

- Tenant scoping is automatic via `database.RegisterTenantCallbacks` + `db.WithContext` — don't add `tenant_id` WHERE clauses, but DO register callbacks in test DB setup.
- `sqlite` tests can't use Postgres `gen_random_uuid()` defaults — report ids are `uuid.New()` in Go (administrator.create).
- Validator-less `socket.handlers` entries are silently dropped (`BuildHandlerMap` continues).
- New opcodes are invisible to live tenants until config PATCH + channel restart.
- `requests.ErrNotFound` (HTTP 404) vs `model.ErrNoResultFound` (empty by-name list) both map to NOT_FOUND; transport errors map to INTERNAL (never misreport a working name as missing).
- `message.Emit` skips emission when the callback errors — business rejections must `buf.Put` the ERROR event and return nil.
- `go mod tidy` only AFTER imports exist; never `go work sync`; guard scripts from repo root without global GOWORK=off.
- IDA export splices are surgical; unresolvable fnames are stop-and-ask.
- v84 clientbound opcode table is shifted vs v83 above ~0x3D — verify v84 cells from the v84 IDB.
- Chat log / transcript render as text nodes in the UI (`whitespace-pre-wrap`), never `dangerouslySetInnerHTML`.

## Dependency order

```
T1-3 (packet lib) ──┬─→ T13-17 (channel) ──→ T18-19 (config/ingress)
T4-9 (ban)  ────────┤                         T24 (verify, needs T23)
T10-12 (redis/msgs) ┘   T20-22 (UI, needs T9 contract)
T23 (IDA naming) ──→ T24 ──→ T25 (gates + acceptance)
```

T4-9, T10-12, and T20-22 are mutually independent workstreams.
