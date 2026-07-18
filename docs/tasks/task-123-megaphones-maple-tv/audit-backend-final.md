# Backend Audit (final, full-branch) — task-123-megaphones-maple-tv

- **Scope:** Go changes on branch `task-123-megaphones-maple-tv` at HEAD `4485129a8` (50 commits, diff base `c9490b724`) — megaphones + Maple TV extended to all 9 client versions. `libs/atlas-packet`, `libs/atlas-saga`, `libs/atlas-redis`, `services/atlas-world/.../broadcast`, `services/atlas-saga-orchestrator/.../saga`, `services/atlas-channel/.../{socket/handler,worldbroadcast,kafka}`.
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/EXT-* checklists)
- **Date:** 2026-07-18
- **Build:** PASS — `go build ./...` clean in all 6 changed modules (libs/atlas-packet, libs/atlas-saga, libs/atlas-redis, atlas-world, atlas-saga-orchestrator, atlas-channel)
- **Vet:** PASS — `go vet ./...` clean in all 6 modules
- **Tests:** PASS — `go test ./... -count=1` green in all 6 modules; zero `FAIL`/`panic` lines; no anomalously slow package (no unstubbed-Kafka-producer symptom)
- **Guards:** `tools/redis-key-guard.sh` exit 0; `tools/goroutine-guard.sh` exit 0
- **Overall:** NEEDS-WORK (1 Important, 2 Minor)

This supersedes `docs/tasks/task-123-megaphones-maple-tv/audit-backend.md` (which reviewed the branch as of commit `eb309133e`, before the "full prod support for all 9 versions" legacy expansion — commits `362e165c`..`4485129a8`). All findings and PASS evidence below were independently re-verified against current HEAD, not carried over from that report.

## Findings

### IMPORTANT — DOM-01: `atlas-world/broadcast` domain model has no `builder.go`; core types are fully public and constructed by raw literal, not validated

`services/atlas-world/atlas.com/world/broadcast/` has a `model.go` (`QueueModel`, `Entry`, `Payload` at lines 19-42), which per the Phase-2 classification rule ("has `model.go` → full DOM checklist applies") makes this a domain package. `file-responsibilities.md`'s own `model.go` entry requires "immutable domain objects with private fields and accessor methods," and `builder.go` requires a "Fluent API for constructing validated domain models. `Build()` enforces invariants." Neither holds here:

- `broadcast/model.go:33-42` (`Entry`) and `:44-47` (`QueueModel`) declare every field exported (`Id`, `CharacterId`, `Payload`, `DurationSeconds`, `ActivatedAt`, `ExpiresAt`, `Active`, `Pending`) — no private-field+getter pattern.
- No `builder.go` exists in the package (`ls services/atlas-world/atlas.com/world/broadcast/` → `model.go model_test.go processor.go processor_test.go registry.go resource.go rest.go rest_test.go task.go testmain_test.go`).
- `services/atlas-world/atlas.com/world/kafka/consumer/broadcast/consumer.go:49-65` constructs `broadcast.Entry{...}` via a raw struct literal directly from Kafka-message fields — no validation function is ever invoked on construction, matching the anti-pattern table's "Mutable public fields | Violates immutability."

This is not a data-corruption risk — the CAS layer (`registry.go`'s `Upsert`, `TenantRegistry.Update`) that guards concurrent mutation is independently correct (see PASS evidence below) — but the domain-model construction pattern itself deviates from the guideline. Note: `DOM-02`/`DOM-03`/`DOM-16` (`ToEntity`/`Make`/`administrator.go`) are **not** cited here — this package is deliberately Redis-CAS-backed with no GORM entity anywhere in its call graph (`registry.go` wraps `atlas-redis.TenantRegistry` directly), and those three checks are defined in terms of GORM entities and DB mutation that simply do not exist in this substrate; citing their absence would not be enforcing a real gap.

### MINOR — DOM-18: `atlas-world/broadcast/rest.go` `RestModel` missing `SetID`

`services/atlas-world/atlas.com/world/broadcast/rest.go:19-25` implements `GetName()`/`GetID()` but not `SetID()`. Two of three sibling GET-oriented `RestModel`s in the same service module implement it (`services/atlas-world/atlas.com/world/channel/rest.go:34`, `services/atlas-world/atlas.com/world/world/rest.go:36`); a third (`configuration/rest.go`) does not. Functionally harmless today (the resource is GET-only, `SetID` is never invoked on the marshal path), but incomplete against the documented JSON:API interface (`GetName()`, `GetID()`, `SetID()`).

### MINOR — DOM-25 follow-up: legacy `AvatarMegaphoneResult` writer tables seeded but unreachable

`template_gms_48_1.json`/`template_gms_61_1.json`/`template_gms_72_1.json`/`template_gms_79_1.json` all carry an `AvatarMegaphoneResult` writer with per-version `errorCodes` (`WAITING_LINE`/`LEVEL_GATE` = 48/49, 55/56, 63/64, 75/76 respectively — verified via direct JSON parse of all four templates). However `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:261-265` unconditionally returns (without consuming, without dispatching) for `category == item.ClassificationAvatarMegaphone && t.MajorVersion() < 83`, **before** `handleAvatarMegaphoneUse` (whose `reject()` at `character_cash_item_use_megaphone.go:366-372` is the only call site of the `AvatarMegaphoneResult` writer) is ever reached. The four tables are therefore currently dead config on all four legacy templates — not a DOM-25 violation (extra tables cannot cause item loss, the direction the checklist cares about) but worth flagging: either avatar-megaphone should be opened on legacy once a serverbound send case is located, or the tables should carry a comment marking them as forward-looking / to be removed if the gate is never opened.

## What passed (evidence, independently re-verified at HEAD `4485129a8`)

- **DOM-25 core (config-resolved client bytes), all in-scope + legacy versions:**
  - `libs/atlas-packet/chat/world_message_body.go:24-45` — all four `WorldMessage*Body` functions route through `atlas_packet.WithResolvedCode("operations", ...)`; zero literal mode bytes.
  - `libs/atlas-packet/chat/avatar_megaphone_body.go:29-36` and `libs/atlas-packet/tv/tv_body.go` (`TvSetMessageBody`, `TvSendMessageResultErrorBody`) resolve via `atlas_packet.ResolveCode`/`WithResolvedCode` against `errorCodes`/`messageTypes` tables; `AvatarMegaphoneResultReason`/`TvResultReason`/`TvMessageType` are semantic string enums, never bytes, consumed at `character_cash_item_use_megaphone.go:277,371` and `:306`.
  - Verified by direct JSON parse of all 9 templates that the tables exist everywhere the wire is exercised: `template_gms_{83,84,87,95}_1.json` and `template_jms_185_1.json` all carry `AvatarMegaphoneResult.errorCodes` (values non-stable per version: 83/84 → 86/87 → 88/89 → 96/97, confirming genuine per-version IDA derivation, not a copy-paste) and `TvSetMessage.messageTypes` / `TvSendMessageResult.errorCodes` (stable 0/1/2 and 1/2/3 across all five — plausible given the ledger's IDA citations). `template_gms_{48,61,72,79}_1.json` correctly have **zero** `Tv*` writer entries (`grep -c '"writer": "Tv'` = 0 on all four) matching the code-level TV block on legacy (see below).
  - `WorldMessage.operations` table carries `ITEM_MEGAPHONE`/`MULTI_MEGAPHONE` on every version where the handler can reach tier 6/7: `template_gms_61_1.json` (tier 6 only, matches the code's `case 61: allowed = allowed || tier == 6`), `template_gms_72_1.json`/`template_gms_79_1.json` (tier 6+7), `template_gms_83/84/87/95/jms185` (unconditional ≥83 dispatch). `template_gms_48_1.json` correctly has **no** `ITEM_MEGAPHONE`/`MULTI_MEGAPHONE` entries, matching the code's default `tier <= 4` (v48 has no switch case, so tiers 6/7 stay blocked).

- **Legacy version gate (`character_cash_item_use.go:261-294`), verified for correctness, not just presence:**
  - Boundary is `t.MajorVersion() < 83` (not the documented `>83`-off-by-one trap) — avatar megaphone is blocked entirely below 83 (`:262-265`, unconditional); megaphone tiers use an allow-list (`tier <= 4` baseline, `+6` for v61, `+6,7` for v72/79 — `:274-283`) that was cross-checked against each template's actual `WorldMessage.operations`/`Tv*` writer presence above and matches exactly (no tier is allowed in code without its writer table existing, and no tier's writer table exists without the code allowing it — excepting the dead `AvatarMegaphoneResult` tables noted above, which are extra, not missing).
  - Tier 5 (Maple TV) is never added by any switch arm — confirmed blocked on all four legacy versions regardless of serverbound-wire verification status, per the extensive in-code citation (`:226-241`) and the independently-confirmed zero `Tv*` writer count.
  - `gms_92`/`gms_12` (both live-served per `deploy/k8s/base/versions.json:9,5`) re-confirmed to have **zero** `CharacterCashItemUseHandle` handler entries in their templates (`grep -c` = 0 on both `template_gms_92_1.json` and `template_gms_12_1.json`), so `CharacterCashItemUseHandleFunc` (registered at `main.go:843`) is unreachable for those tenants — the megaphone/TV code paths cannot fire, no item-loss exposure. This is the same conclusion the prior `eb309133e` fix reached; re-verified independently rather than trusted.
  - `updateTimeFirst := t.MajorVersion() >= 87` (`character_cash_item_use.go:42`) — the one other `MajorVersion()` boundary touched by this branch — uses `>=`, avoiding the known `>83`-is-off-by-one-for-v84 trap (memory: v84 is byte-identical to v83).

- **Concurrency (CAS retry purity), re-verified with no code changes since the prior fix:**
  - `git log eb309133e..4485129a8 -- services/atlas-world/atlas.com/world/broadcast/ libs/atlas-redis/tenant_registry.go` is empty — neither file changed after the prior audit's fix landed.
  - `libs/atlas-redis/tenant_registry.go:130-176` `Update`'s `fn` is invoked inside `txFn` (the `Watch` callback) and only computes/stashes `result`; all Kafka/side-effects happen in callers, strictly after `Update` returns. Retry loop (`:165-175`) re-runs `txFn` (thus `fn`) on `goredis.TxFailedErr`, up to `updateMaxRetries`.
  - `services/atlas-world/atlas.com/world/broadcast/processor.go:90-107` (`Enqueue`) and `:177-189` (`sweepQueue`) — both `fn` closures passed to `Upsert` only mutate local `QueueModel` and stash observations (`preAppendWaitSeconds`, `activated`, `expired`) into outer-scope vars; every `kmessage.Emit`/`mb.Put` call happens after `Upsert` returns (`:119-140`, `:191-220`), reading only the winning attempt's stashed values. Explicitly documented at `:60-85` why append+activate must be one CAS transaction.
  - Genuine contention test: `libs/atlas-redis/tenant_registry_test.go:291-334` (`TestTenantRegistry_Update_RetriesOnContention`) — an interloper client writes mid-transaction via a real miniredis instance; asserts the retried `fn` observed the interloper's write (`"interloper-updated"`), not a faked/simulated retry count.

- **Kafka discipline:**
  - Fan-out consumers use `kafka.LastOffset`: `services/atlas-channel/atlas.com/channel/kafka/consumer/megaphone/consumer.go:45`, `.../worldbroadcast/consumer.go:38`. The one-shot command consumer deliberately omits it: `services/atlas-world/atlas.com/world/kafka/consumer/broadcast/consumer.go:20-24` (documented rationale — a missed enqueue command is not self-healing).
  - Both cross-service broadcast producers key by `worldId`: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/producer.go:324` (`MegaphoneBroadcastEventProvider`) and `:345` (`WorldBroadcastEnqueueCommandProvider`), both via `producer.CreateKey(int(payload.WorldId))`.

- **DOM-24 (producer stubbing):** `services/atlas-world/atlas.com/world/broadcast/testmain_test.go:10-13` and `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/testmain_test.go:10-13` both call `producertest.InstallNoop()` in `TestMain` with no `t.Cleanup(producer.ResetInstance)` reverting it mid-package.

- **EXT checklist (`worldbroadcast` REST client, `services/atlas-channel/atlas.com/channel/worldbroadcast/`):**
  - EXT-01: `rest.go:39-46` — `SetToOneReferenceID`/`SetToManyReferenceIDs` no-op stubs present (fixed from the prior audit round, re-verified still present at HEAD).
  - EXT-02: `rest_test.go` has an `httptest.NewServer`-backed round trip with a representative JSON:API fixture.
  - EXT-03: `rest_test.go:151-152` asserts `errors.Is(err, requests.ErrNotFound)` on a genuine 404, not swallowed to a generic error.
  - EXT-04: `requests.go:18-20` — `requests.RootUrl("WORLDS")`, no hardcoded host.

- **File Responsibilities:** `services/atlas-channel/atlas.com/channel/worldbroadcast/{processor.go,requests.go,rest.go}`, `services/atlas-world/atlas.com/world/broadcast/{model.go,processor.go,registry.go,resource.go,rest.go,task.go}`, and the Kafka message packages (`kafka/message/megaphone/kafka.go` 31 lines pure DTO, `kafka/message/worldbroadcast/kafka.go` 44 lines pure DTO) all correctly split by responsibility — no `<pkg>.go` catch-all collapsing Processor+RestModel+requests found anywhere in the diffed packages.

- **DOM-26 / DOM-21 / multi-tenancy:** `tools/goroutine-guard.sh` exit 0 (no bare `go` in non-test code across the whole repo, including this branch's new files); `tenant.MustFromContext(ctx)` used consistently (`character_cash_item_use.go:30`, `broadcast/processor.go:50`); `byte` for `WorldId`/`ChannelId` in the new Kafka DTOs matches the mandated `gachapon` sibling precedent (prior audit's adjudication, unchanged code, not re-litigated here).

- **DOM-22 (Dockerfile/go.mod):** No `go.mod` changes in any of the three services since the prior audit's verified `docker buildx bake atlas-world` pass (`git diff eb309133e..4485129a8 --stat -- '**/go.mod'` empty for `atlas-world`/`atlas-channel`/`atlas-saga-orchestrator`); the repo's actual root `Dockerfile` is a single shared, parameterized file (not per-service) whose `go.work` generation loop (`Dockerfile:93-96`) unconditionally includes `atlas-lock`/`atlas-saga` — the generic 4-mentions-per-service-Dockerfile text does not apply to this repo's actual build structure, as previously adjudicated.

## Summary

### Blocking (must fix)
- None (0 Critical).

### Non-Blocking (should fix)
- **[Important] DOM-01** — `services/atlas-world/atlas.com/world/broadcast/model.go`: no `builder.go`; `QueueModel`/`Entry`/`Payload` have fully public fields, constructed by raw struct literal at `kafka/consumer/broadcast/consumer.go:49` with no validation.
- **[Minor] DOM-18** — `services/atlas-world/atlas.com/world/broadcast/rest.go:11-25`: `RestModel` missing `SetID()`, present on 2 of 3 sibling GET-resource models in the same service.
- **[Minor] DOM-25 follow-up** — `template_gms_{48,61,72,79}_1.json` carry `AvatarMegaphoneResult.errorCodes` tables that are currently unreachable (the code gate blocks avatar megaphone entirely below MajorVersion 83); harmless but dead config, flag for a decision (open the gate later or remove/annotate the tables).

---

## Resolution (controller, commit c7423f5d4)
All three findings fixed:
- **DOM-01 (Important):** added `broadcast.NewEntry(family, id, characterId, payload, durationSeconds) (Entry, error)` validating `family ∈ {TV,AVATAR}`; consumer now builds via the constructor and handles its error (duplicate pre-check removed). Fields stay public (Redis JSON DTO).
- **DOM-18 (Minor):** added `SetID` to the world-side `broadcast/rest.go` RestModel, matching sibling signature.
- **DOM-25 follow-up (Minor):** removed the dead `AvatarMegaphoneResult` writer + `errorCodes` from `template_gms_{48,61,72,79}_1.json` (avatar send is gated off <v83); `SetAvatarMegaphone`/`ClearAvatarMegaphone` render writers kept.
Verified: atlas-world + atlas-configurations build/vet/test clean; all 4 packet-audit gates + redis/goroutine guards exit 0. The pre-existing registry-fname matrix under-report (not task-123) remains a separate follow-up.
