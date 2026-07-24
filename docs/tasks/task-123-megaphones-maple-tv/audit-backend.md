# Backend Audit — task-123-megaphones-maple-tv

- **Scope:** Go changes on branch `task-123-megaphones-maple-tv` (megaphones & Maple TV) across `libs/atlas-packet`, `libs/atlas-saga`, `libs/atlas-redis`, `services/atlas-world/.../broadcast`, `services/atlas-saga-orchestrator/.../saga`, `services/atlas-channel/.../{socket/handler,worldbroadcast,kafka}`.
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/EXT-* checklists)
- **Date:** 2026-07-17
- **Build:** PASS (`go build ./...` clean in all 6 changed modules; `docker buildx bake atlas-world` succeeds — new direct deps `atlas-lock`/`atlas-saga` already covered by the shared root Dockerfile's hardcoded lib list)
- **Vet:** PASS (`go vet ./...` clean in all 6 changed modules)
- **Tests:** PASS (`go test ./... -count=1` green in `libs/atlas-packet`, `libs/atlas-saga`, `libs/atlas-redis`, `atlas-world`, `atlas-saga-orchestrator`, `atlas-channel`; no FAIL/panic lines; no suspiciously-slow packages indicating an unstubbed Kafka producer)
- **Guards:** `tools/goroutine-guard.sh` exit 0; `tools/redis-key-guard.sh` exit 0
- **Overall:** NEEDS-WORK (1 Critical, 2 Important, 2 Minor)

## Findings

### CRITICAL — DOM-25: gms_v92 has zero megaphone/TV wire tables; no code-level version gate protects it

`services/atlas-configurations/seed-data/templates/template_gms_92_1.json` has **no** `"writer": "WorldMessage"` entry and **zero** `AvatarMegaphoneResult`/`TvSetMessage`/`TvSendMessageResult`/`TvClearMessage` writer entries (verified via `grep -c` across all 11 templates). `gms_v87` and `gms_v95` — the versions immediately bracketing v92 in `deploy/k8s/base/versions.json` — both have full coverage. `git log c9490b724..HEAD -- services/atlas-configurations/seed-data/templates/` shows this branch's 5 template-touching commits (`f25492f1c`, `b413d8126`, `27d052335`, `fbb66ed75`, `5366cf13e`) only modified `template_gms_{83,84,87,95}_1.json` and `template_jms_185_1.json` — v92 was never seeded.

`deploy/k8s/base/versions.json:13` lists `gms/92/1` as a **live-served** version. `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_megaphone.go` has exactly one version gate — `t.Region() == "GMS" && t.MajorVersion() >= 95` at line 109, for the Skull-Megaphone→TV branch only. Every other megaphone tier (basic/super/item/triple at `handleMegaphoneUse` case 1/2/6/7) and avatar megaphone (`handleAvatarMegaphoneUse`) run unconditionally for any tenant, including v92.

Traced the actual failure mode: `writer.Producer` (`libs/atlas-socket/writer/writer.go:25-27`) is a `map[string]BodyFunc` lookup by writer name, built from the tenant's seed-template `writers` list at startup. Since v92's template has no `WorldMessage`/`AvatarMegaphoneResult`/`TvSetMessage`/`TvSendMessageResult`/`TvClearMessage` entries, `writerProducer(writerName)` returns an error *before* `ResolveCode` ever runs (`session/processor.go:248-253`), which the megaphone/worldbroadcast consumers catch and merely log at Warn (`kafka/consumer/megaphone/consumer.go:119-121`, `kafka/consumer/worldbroadcast/consumer.go:180-182`). Net effect on a v92 tenant: a player uses a megaphone or Maple TV cash item, the saga's `consume_*_item` step still destroys the item (saga steps are unconditional — `character_cash_item_use_megaphone.go:160-189`, `:243-249`), and **no packet is ever sent to any client** — a silent full-feature failure with real item loss, on a version this task's own `versions.json` declares supported. This is exactly the class of defect DOM-25 exists to catch ("Confirm the tables exist in EVERY supported version's seed template") and is a code defect, not a verification-coverage note (that exemption in the task brief covers unverified *serverbound* cells, not entirely absent *clientbound* writer registrations for a live version).

**Fix required:** either (a) seed `template_gms_92_1.json` with IDA-verified `WorldMessage`/`AvatarMegaphoneResult`/`TvSetMessage`/`TvSendMessageResult`/`TvClearMessage` tables (note: the per-version `errorCodes` values are NOT stable across versions — v83/84/87/95 show `WAITING_LINE: 83/86/88/96` respectively, so v92 needs its own IDA derivation, not a copy), or (b) add an explicit version gate excluding v92 from these code paths with a design-doc citation, mirroring the existing `MajorVersion() >= 95` pattern.

### IMPORTANT — DOM-21: raw `byte` used for WorldId/ChannelId instead of the shared `world.Id`/`channel.Id` types

Every other Kafka message package in this branch's diffed services uses `world.Id`/`channel.Id` for these fields (verified: `kafka/message/{cashshop,compartment,food,macro,note,reactor,storage,door,drop,party,system_message}/kafka.go` in atlas-channel all declare `WorldId world.Id` / `ChannelId channel.Id`). The new megaphone/broadcast message DTOs introduced by this task instead use bare `byte`, losing the type safety and self-documentation the shared library provides, and creating an inconsistency with the sibling `atlas-saga` payload types (`EmitMegaphonePayload`/`EnqueueWorldBroadcastPayload` in `libs/atlas-saga/payloads.go:979-980,993-994` correctly use `world.Id`/`channel.Id`) — the downgrade to `byte` happens specifically at the Kafka wire-envelope layer:

- `services/atlas-channel/atlas.com/channel/kafka/message/megaphone/kafka.go:23-24` — `WorldId byte`, `ChannelId byte`
- `services/atlas-channel/atlas.com/channel/kafka/message/worldbroadcast/kafka.go:30,34` — `WorldId byte`, `ChannelId byte`
- `services/atlas-world/atlas.com/world/kafka/message/broadcast/kafka.go:23-24,49,67,71` — `EnqueueCommand.WorldId/ChannelId`, `StartedPayload.ChannelId`, `StatusEvent.WorldId/ChannelId`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/broadcast/kafka.go:26-27`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/megaphone/kafka.go:19-20`

### IMPORTANT — EXT-01: worldbroadcast REST client model missing relationship-interface stubs

`services/atlas-channel/atlas.com/channel/worldbroadcast/rest.go:15-34` defines `RestModel` with `GetName()`/`GetID()`/`SetID()` but no `SetToOneReferenceID`/`SetToManyReferenceIDs`, even as no-ops. Sibling REST-client packages in the same service implement both defensively: `services/atlas-channel/atlas.com/channel/door/rest.go:45-49`, `services/atlas-channel/atlas.com/channel/storage/rest.go:57-61`, `services/atlas-channel/atlas.com/channel/cashshop/inventory/asset/rest.go:63-68`. Per `libs/atlas-rest/CLAUDE.md` (cited in the guideline as EXT-01's rationale), a future upstream response carrying a `relationships` block will error api2go's unmarshal without these — the same class of bug that surfaced twice in task-037 as misleading "not found" errors.

### MINOR — DOM-25 follow-up: legacy versions (v12/48/61/72/79) also lack the new writer tables, unverified as intentional

`template_gms_12_1.json` has zero `WorldMessage` op entries at all (pre-existing, not touched by this branch); `template_gms_{48,61,72,79}_1.json` have the `MEGAPHONE`/`SUPER_MEGAPHONE`/(61-79 only)`ITEM_MEGAPHONE`/`MULTI_MEGAPHONE` operations tables from a prior task, but none of them have `AvatarMegaphoneResult`/`TvSetMessage`/`TvSendMessageResult`/`TvClearMessage`. This is plausibly genuine feature-absence on pre-Big-Bang clients (unlike v92, these are far older and the task's own code shows awareness of WZ-based item-absence, e.g. `character_cash_item_use_megaphone.go:151` "type-8 has no item in v83 WZ (design D11)"), but unlike the v95 Skull-Megaphone gate, there is no code comment or design-doc citation establishing that avatar-megaphone/Maple-TV items are unobtainable on v12-79. Flag for explicit verification rather than silent reliance on WZ item absence.

### MINOR — no dedicated test file for `character_cash_item_use_megaphone.go`

332 lines of branching business logic (megaphone-tier classification switch, TV wait-cap rejection with config-resolved reject reasons, avatar-megaphone reject path) with no `character_cash_item_use_megaphone_test.go`. Not a guideline violation per se — only 14 of 130 files in `services/atlas-channel/atlas.com/channel/socket/handler/` have dedicated tests, so this matches the package's existing (sparse) testing convention rather than deviating from it — but it is the largest untested new file in the diff and covers exactly the config-resolution reject paths DOM-25 cares about.

## What passed (evidence)

- **DOM-25 core implementation (v83/84/87/95/jms):** `libs/atlas-packet/chat/world_message_body.go`, `avatar_megaphone_body.go`, `tv/tv_body.go` all resolve wire codes via `atlas_packet.WithResolvedCode`/`ResolveCode` — zero client-wire literals found outside codec internals. `AvatarMegaphoneResultReason`/`TvMessageType`/`TvResultReason` are semantic string enums; `WAITING_LINE`/`LEVEL_GATE` codes are IDA-verified per version and differ per version (83→83/84, 84→86/87, 87→88/89, 95→96/97 — confirmed non-stable, ruling out a lazy copy-paste).
- **DOM-25(c) domain-service semantic keys:** `TvMessageType string` rides through `saga/payloads.go:1001`, `kafka/message/broadcast/kafka.go:77`, `broadcast/model.go:26` as a semantic key (`NORMAL|STAR|HEART`), never a byte — matches the controller's adjudication.
- **DOM-26 (routine.Go):** `services/atlas-world/atlas.com/world/main.go:108,123,146` use `routine.Go`; no bare `go` statements in non-test code. `tools/goroutine-guard.sh` exit 0.
- **Redis key guard:** `tools/redis-key-guard.sh` exit 0.
- **CAS-retry purity (highest-risk area):** `services/atlas-world/atlas.com/world/broadcast/processor.go:60-107,167-221` — both `Enqueue`'s and `sweepQueue`'s `fn` closures passed to `Registry.Upsert` only compute a new `QueueModel` and stash observations into local vars for the caller to read after `Upsert` returns; all Kafka emission happens strictly after `Upsert` returns, using the winning attempt's stashed values. Extensively self-documented (`processor.go:62-85`) explaining why append+activate must be one CAS transaction, not two. `libs/atlas-redis/tenant_registry.go:130-176` `Update`'s retry loop matches this contract exactly.
- **DOM-23 (topic naming):** `EVENT_TOPIC_MEGAPHONE`, `EVENT_TOPIC_WORLD_BROADCAST_STATUS`, `COMMAND_TOPIC_WORLD_BROADCAST` all present in `deploy/k8s/base/env-configmap.yaml` (lines 123, 156, 78) with `KEY: "KEY"` shape; zero literal `env:`/`value:` duplication found in `atlas-world.yaml`/`atlas-channel.yaml`/`atlas-saga-orchestrator.yaml`.
- **DOM-22 (Dockerfile/go.mod):** atlas-world's go.mod gained direct requires on `atlas-lock` and `atlas-saga`; `docker buildx bake atlas-world` succeeds from the worktree root (the shared root Dockerfile's hardcoded lib enumeration already covered both, per this repo's actual convention — no per-service Dockerfile exists here, superseding the generic 4-mentions-per-service-Dockerfile check text). No go.mod changes in atlas-channel, atlas-saga-orchestrator, or the touched libs.
- **Kafka consumer offset discipline:** `services/atlas-channel/atlas.com/channel/kafka/consumer/{megaphone,worldbroadcast}/consumer.go` both use `consumer.SetStartOffset(kafka.LastOffset)` (fire-and-forget render fan-out); `services/atlas-world/atlas.com/world/kafka/consumer/broadcast/consumer.go` deliberately omits it with a documented rationale (one-shot command, self-healing loss unacceptable).
- **File-responsibilities split:** `services/atlas-channel/atlas.com/channel/worldbroadcast/{processor.go,requests.go,rest.go}` and `services/atlas-world/atlas.com/world/broadcast/{model.go,processor.go,registry.go,resource.go,rest.go,task.go}` are correctly split per file responsibility — no `<pkg>.go` catch-all collapsing Processor+RestModel+requests.
- **EXT-02/EXT-03 (worldbroadcast client):** `worldbroadcast/rest_test.go:96-154` has an `httptest.NewServer`-backed round-trip test with a representative JSON:API fixture, and a dedicated 404 test asserting `errors.Is(err, requests.ErrNotFound)` (not swallowed to a generic "not found").
- **DOM-24 (Kafka producer stubbing):** `services/atlas-world/atlas.com/world/broadcast/testmain_test.go` installs `producertest.InstallNoop()` (Pattern A); full `atlas-channel`/`atlas-world`/`atlas-saga-orchestrator` test suites complete in well under a second per package with no signs of the ~42s unstubbed-producer penalty.

## Summary

### Blocking (must fix)
- DOM-25: gms_v92 (live-served per versions.json) has no `WorldMessage`/`AvatarMegaphoneResult`/`TvSetMessage`/`TvSendMessageResult`/`TvClearMessage` writer registration and no code-level version gate — megaphone/TV cash items silently consume with zero client effect on that version.

### Non-Blocking (should fix)
- DOM-21: `byte` instead of `world.Id`/`channel.Id` in 5 new Kafka message DTOs (channel megaphone/worldbroadcast, world broadcast, saga-orchestrator broadcast/megaphone).
- EXT-01: `worldbroadcast/rest.go` RestModel missing `SetToOneReferenceID`/`SetToManyReferenceIDs` no-op stubs.
- DOM-25 (minor): v12/48/61/72/79 writer-table absence for avatar-megaphone/TV is unverified as intentional (no design citation, unlike the v95 gate).
- Testing (minor): no dedicated test for `character_cash_item_use_megaphone.go` (matches existing package convention, but is the largest untested file in the diff).

---

## Controller resolution (2026-07-17)

Adjudicated each finding against source; fixes landed in commit `eb309133e`.

- **Critical (DOM-25, "gms_92 item loss") — REFRAMED + FIXED.** The finding named gms_92, but `template_gms_92_1.json` (and gms_12) have **zero** `CharacterCashItemUseHandle` handlers (`grep -c` = 0) — the megaphone code is only reachable via that handler, so v92/v12 can never initiate a megaphone and there is no item loss there (design D9's exclusion is sound). The **real** exposure is the legacy versions gms_48/61/72/79, which **do** carry the USE_CASH_ITEM handler (`grep -c` = 1 each) but have no megaphone writer tables. This branch's new classification dispatch would fire the consume saga on those versions and destroy the cash item with no broadcast rendered. FIXED: the megaphone/avatar dispatch is now gated to `MajorVersion() >= 83`, so legacy versions ignore the use without consuming (restoring the pre-feature no-op fall-through).
- **Important (DOM-21, byte vs world.Id) — NOT A DEFECT.** The sibling `kafka/message/gachapon/kafka.go` (the precedent this plan mirrors byte-for-byte for cross-service JSON parity) also uses `byte` for WorldId/ChannelId. `world.Id`/`channel.Id` are `byte` aliases, so the JSON wire is identical; matching the mandated precedent is correct. No change.
- **Important (EXT-01, missing relationship stubs) — FIXED.** Added no-op `SetToOneReferenceID`/`SetToManyReferenceIDs` to `worldbroadcast/rest.go` for interface-consistency with sibling channel REST-client models. The resource carries no relationships (Task 11's review traced api2go and confirmed the omission was functionally safe); the stubs are defensive consistency.
- **Minor (legacy versions lack writer tables) — SUBSUMED** by the v83+ gate above (they now never reach the consume path).
- **Minor (no dedicated test for character_cash_item_use_megaphone.go) — ACKNOWLEDGED.** Matches the repo convention for socket handlers (no unit harness; exercised via build + live acceptance); the snapshot converters it depends on ARE unit-tested (`socket/model/snapshot_test.go`).
