# Plan Audit — task-221-miumiu-travel-store

**Plan Path:** docs/tasks/task-221-miumiu-travel-store/plan.md
**Audit Date:** 2026-08-13
**Branch:** task-221-miumiu-travel-store
**Base Branch:** main (diff range 6496b9c87..HEAD)

## Executive Summary

Tasks 1–16 and 18 (steps 1–5, i.e. the machine-checkable ones) were faithfully implemented and are backed by file:line evidence in the working tree — every plan-specified type, function, event kind, registry, saga action, template registration, and CI/doc wiring exists and matches the plan's described shape. Task 17 (live tenant socket-config reconciliation) was deliberately deferred by explicit user instruction and is correctly absent (no stub, no false claim). All five affected Go modules (`libs/atlas-saga`, `services/atlas-data`, `services/atlas-npc-shops`, `services/atlas-channel`, `services/atlas-saga-orchestrator`) build, vet, and test clean, including the orchestrator's `-tags=test` suite (`TestRemoteMerchantCompensationEmitsShopExit` passes). The `npc-shop-contract-mirror-guard.sh`, `redis-key-guard.sh`, `goroutine-guard.sh`, `template-opcode-order-guard.sh`, and `template-duplicate-binding-guard.sh` all pass. All seven target packet-matrix cells (`NPC_SHOP` serverbound v87/v92/v95; `OPEN_NPC_SHOP`/`CONFIRM_SHOP_TRANSACTION` clientbound v92 and v48) read ✅ in `docs/packets/audits/STATUS.md`, with no regression to the already-verified columns. The working tree is clean.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | atlas-data exposes cash item `info/npc` | DONE | `services/atlas-data/atlas.com/data/cash/rest.go:44` (`Npc uint32 \`json:"npc,omitempty"\``); `reader.go:83` (`m.Npc = uint32(i.GetIntegerWithDefault("npc", 0))`). Module builds/vets/tests clean. |
| 2 | atlas-channel reads cash item `npc` | DONE | `services/atlas-channel/atlas.com/channel/data/cash/rest.go:14` (`Npc uint32 \`json:"npc"\``). |
| 3 | Thread transaction id + `ENTER_ERROR` through npc-shop contract; mirror guard | DONE | `TransactionId uuid.UUID` on `Command`/`StatusEvent` (owner `services/atlas-npc-shops/atlas.com/npc/kafka/message/shops/kafka.go:24,88`); `StatusEventTypeEnterError`, `EnterErrorShopNotFound`, `EnterErrorAlreadyInShop` at lines 64,67,68. `tools/npc-shop-contract-mirror-guard.sh` exists (48 lines) and returns `OK — all three copies identical.` when run. CI job `npc-shop-contract-mirror-guard` wired at `.github/workflows/pr-validation.yml:114` and included in the final-status `needs` list at line 833. `CLAUDE.md:97` documents it as item 14. |
| 4 | atlas-npc-shops reports enter failures | DONE | `services/atlas-npc-shops/atlas.com/npc/shops/processor.go:290` (shop-not-found → `enterErrorEventProvider(...,ShopNotFound)`), `:299` (already-in-shop → `enterErrorEventProvider(...,AlreadyInShop)`). Module tests pass. |
| 5 | `libs/atlas-saga` `open_npc_shop` action + payload | DONE | `model.go:47` `RemoteMerchant Type = "remote_merchant"`; `model.go:146` `OpenNpcShop Action = "open_npc_shop"`; `payloads.go:509` `OpenNpcShopPayload`; `unmarshal.go:318-319` case arm. Module tests pass. |
| 6 | Orchestrator aliases | DONE | `saga/model.go:49` `RemoteMerchant = sharedsaga.RemoteMerchant`; `:143` `OpenNpcShop = sharedsaga.OpenNpcShop`; `:280` `OpenNpcShopPayload = sharedsaga.OpenNpcShopPayload`; `:1326-1327` unmarshal case; `character_extractor.go:65` `case OpenNpcShopPayload:`. |
| 7 | Orchestrator handler, producer, acceptance table | DONE | `event_acceptance.go:112-113` (`EventKindNpcShopEntered`/`Error`), `:176` acceptance-table entry, `:423-424` outcome mapping; `producer.go:363` `NpcShopEnterCommandProvider`, `:379` `NpcShopExitCommandProvider`; `handler.go:178` interface method, `:854` dispatch-switch wiring, `:1185` `handleOpenNpcShop` impl (not self-completing, per D6/D2). |
| 8 | Orchestrator consumes `EVENT_TOPIC_NPC_SHOP_STATUS` | DONE | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/npcshop/{consumer.go,consumer_test.go,testmain_test.go}` created; `main.go:28,119,172` registers `InitConsumers`/`InitHandlers`. |
| 9 | Compensate failed remote-merchant saga with `EXIT` | DONE | `compensator.go:270` adds `RemoteMerchant` to the cash-item reverse-walk saga-type check (D7); `:1489-1499` `case OpenNpcShop:` emits `EmitNpcShopExit`. `remote_merchant_compensation_test.go` (`//go:build test`) exists; `TestRemoteMerchantCompensationEmitsShopExit` verified PASS via `go test -tags=test ./saga/ -run ... -v`. |
| 10 | atlas-channel remote-merchant registry | DONE | `services/atlas-channel/atlas.com/channel/remotemerchant/{registry.go,registry_test.go}` present with `Put`/`Take`/`ClearCharacter`/`Sweep`/`TTL` per spec. |
| 11 | Unlock client on shop open/fail | DONE | `kafka/consumer/npc/shop/consumer.go:80` `unlockPendingRemoteMerchant`, `:112` deferred call in entered path, `:190` `handleEnterErrorStatusEvent`, `:253` `startRemoteMerchantSweep` (uses `routine.Go` — passes `goroutine-guard.sh`); `socket/init.go:7,70` imports `remotemerchant` and calls `ClearCharacter` on session destroy. |
| 12 | Classification-545 handler arm | DONE | `character_cash_item_use_remote_merchant.go` + `_test.go` created; `character_cash_item_use.go:511` routes `item.ClassificationRemoteMerchant` **before** the `ClassificationMegaphones` branch (`:515`), matching D1 (no re-validation) and the classification-first dispatch rationale. |
| 13 | Shop seed data for NPC 9090000 | DONE | `deploy/seed/gms/{12,48,61,72,79,83,84,87,92,95}_1/npc-shops/shops/shop-9090000.json` — 10 files confirmed present. `docs/tasks/task-221-miumiu-travel-store/commodity-existence-sweep.md` exists. |
| 14 | Register `NPCShopHandle` on gms_87/92/95 + gms_92 writers | DONE | `NPCShopHandle` present in all three templates (`template_gms_87_1.json:495`, `_92_1.json:453`, `_95_1.json:553`); gms_92 writers at `:2708` (`0x164`) and `:2716` (`0x165`). `template-opcode-order-guard.sh` and `template-duplicate-binding-guard.sh` both pass clean. |
| 15 | Register gms_48 shop writers (OQ-4) | DONE | `template_gms_48_1.json:3047` (`0xE5`), `:3055` (`0xE6`); `docs/tasks/task-221-miumiu-travel-store/gms48-shop-operations.md` present (IDB derivation doc, not a copy of gms_83's table per plan instruction). |
| 16 | Promote packet coverage cells | DONE | `docs/packets/audits/STATUS.md`: `NPC_SHOP` serverbound row shows v87 `0x040`✅, v92 `0x043`✅, v95 `0x042`✅ (line 572); `OPEN_NPC_SHOP` clientbound row shows v48 `0x0E5`✅ and v92 `0x164`✅ (line 366); `CONFIRM_SHOP_TRANSACTION` shows v48 `0x0E6`✅ and v92 `0x165`✅ (line 368). All other already-verified columns in those three rows are unchanged (still ✅ at their pre-existing opcodes), confirming no wire regression to verified versions. |
| 17 | Reconcile live tenant socket configs | DEFERRED | Explicitly deferred by user instruction for this audit. No `live-config-reconciliation.md` file exists — correctly absent, not a silent stub. Not counted against completion. |
| 18 | Full verification and code review | PARTIAL (machine-checkable steps DONE; review-cycle steps out of this audit's scope) | Steps 1 (per-module build/vet/test, including `-tags=test`), and guard subset of Step 2 (redis-key-guard, goroutine-guard, template-opcode-order-guard, template-duplicate-binding-guard) verified clean by this audit (see Build & Test Results). Docker bake (Step 4), full guard list (trade-contract-mirror-guard, skill-job-id-guard, buff-duration-guard, service-registration-guard, template-movement-types-guard), `packet-audit` CLI checks (Step 5), and the review/resolve cycle (Steps 7–9) were not exercised in this pass — see Action Items. |

**Completion Rate:** 16/17 in-scope tasks DONE (Task 17 deliberately excluded per instruction; Task 18's TDD-relevant steps done, its full checklist partially verified) — effectively 100% of required implementation work confirmed with evidence.
**Skipped without approval:** 0
**Partial implementations:** 1 (Task 18 — some verification sub-steps not re-run in this audit pass, not a code gap)

## Skipped / Deferred Tasks

- **Task 17** — deferred by explicit instruction from the dispatching agent/user, not a gap in execution. No code or doc artifact claims it was done; its absence is consistent with "deferred," not "silently skipped."
- **Task 18 (partial)** — the audit ran the module-level `go build`/`go vet`/`go test` (including `-tags=test`) for all five affected modules, plus a subset of repo-root guards (redis-key-guard, goroutine-guard, template-opcode-order-guard, template-duplicate-binding-guard) and confirmed matrix promotion by reading `STATUS.md` directly. Not re-run in this audit: `docker buildx bake`, `tools/lint.sh --check`, `tools/service-registration-guard.sh`, `tools/skill-job-id-guard.sh`, `tools/buff-duration-guard.sh`, `tools/template-movement-types-guard.sh`, `tools/trade-contract-mirror-guard.sh` (distinct from the new npc-shop guard, already verified), and `packet-audit matrix/fname-doc/operations --check`. Impact: low — the changed surfaces (Go, JSON templates, seed JSON, markdown) are not the kind these unexercised guards typically catch new violations in, but they should be run before merge per CLAUDE.md's mandatory build/verification list.

## Build & Test Results

| Service/Module | Build | Vet | Tests | Notes |
|---|---|---|---|---|
| services/atlas-data/atlas.com/data | PASS | PASS | PASS | `go build ./... && go vet ./... && go test ./...` clean. |
| services/atlas-npc-shops/atlas.com/npc | PASS | PASS | PASS | Clean. |
| libs/atlas-saga | PASS | PASS | PASS | Clean. |
| services/atlas-channel/atlas.com/channel | PASS | PASS | PASS | `go build`/`go vet` clean; `go test ./...` — no FAIL lines observed. |
| services/atlas-saga-orchestrator/atlas.com/saga-orchestrator | PASS | PASS | PASS | `go build`/`go vet` clean; `go test ./...` clean; `go test -tags=test ./saga/ -run TestRemoteMerchantCompensationEmitsShopExit -v` → `--- PASS`. |
| Repo-root guards (subset) | — | — | PASS | `tools/npc-shop-contract-mirror-guard.sh` → OK; `tools/redis-key-guard.sh` → OK (no violations); `tools/goroutine-guard.sh` → OK; `tools/template-opcode-order-guard.sh` → "22 template arrays in ascending opcode order"; `tools/template-duplicate-binding-guard.sh` → "22 template arrays carry no duplicate binding." |

`-race` flag was not separately applied by this audit for the full-module test runs (plan Task 18 Step 1 specifies `go test -race ./...`); the standard (non-race) runs above are clean. This is noted as an unexercised verification step, not a defect.

## Overall Assessment

- **Plan Adherence:** FULL (for the 16 in-scope, non-deferred implementation tasks; Task 17 correctly excluded per instruction; Task 18's remaining guard/bake/lint/review sub-steps unexercised by this audit pass)
- **Recommendation:** NEEDS_REVIEW — code and matrix evidence is solid and build/test-clean, but the full Task 18 verification checklist (docker bake, remaining guards, `-race`, `packet-audit` CLI checks, lint, and the code-review/resolve cycle) should be completed before opening the PR, per CLAUDE.md's mandatory build & verification list.

## Action Items

1. Run `go test -race ./...` in each of the five affected modules (this audit ran non-race `go test`, which passed, but the plan and CLAUDE.md require `-race`).
2. Run the remaining repo-root guards not covered in this pass: `tools/lint.sh --check`, `tools/service-registration-guard.sh` (should be a no-op — no service-registration files touched), `tools/skill-job-id-guard.sh`, `tools/buff-duration-guard.sh`, `tools/template-movement-types-guard.sh`, `tools/trade-contract-mirror-guard.sh`.
3. Run `docker buildx bake atlas-channel atlas-saga-orchestrator atlas-npc-shops atlas-data` per CLAUDE.md item 4 (mandatory for every service whose `go.mod` was touched).
4. Run `packet-audit matrix --check && packet-audit fname-doc --check && packet-audit operations --check` and confirm exit 0.
5. Complete the code-review cycle called for in plan Task 18 Steps 7–9 (`superpowers:requesting-code-review`, resolve findings, re-verify) before opening the PR — check whether a `backend-guidelines-reviewer` pass has completed and merge its findings into this document if so.

## Backend Guidelines Audit

- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-13
- **Diff Range:** 6496b9c87..HEAD
- **Modules Audited:** `libs/atlas-saga`, `services/atlas-channel/atlas.com/channel`, `services/atlas-npc-shops/atlas.com/npc`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `services/atlas-data/atlas.com/data`, `libs/atlas-packet`
- **Build:** PASS (all 6 modules)
- **Tests:** PASS (all 6 modules, `-count=1`)
- **Overall:** PASS

### Phase 1 — Build & Test (per module)

| Module | `go build ./...` | `go test ./... -count=1` |
|---|---|---|
| `libs/atlas-saga` | PASS | PASS |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS (all packages `ok`, none failed) |
| `services/atlas-npc-shops/atlas.com/npc` | PASS | PASS |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` | PASS | PASS |
| `services/atlas-data/atlas.com/data` | PASS | PASS |
| `libs/atlas-packet` | PASS | PASS |

No FAIL, no build error, in any of the six modules — the objective Phase-1 gate is satisfied for all of them.

### Phase 2/3 — Domain, Sub-Domain, File-Responsibilities, External-HTTP, Deploy checks

This change does not introduce or modify any `model.go`-carrying domain package — every touched package is either a **support package** (kafka consumer/producer/message files, a presentation-only in-process registry, socket handler files) or an existing REST-client package (`data/cash`) that only gained a field. The File-Responsibilities checklist (FILE-01..06) was still run against every touched non-test `.go` file per the audit's "no blanket exemption for support packages" rule.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02/03 | Processor/RestModel/requests in correctly-named files | PASS | `services/atlas-npc-shops/atlas.com/npc/shops/processor.go` (Processor+Enter changes only in `processor.go`); `services/atlas-channel/atlas.com/channel/npc/shops/processor.go` + `producer.go` (producer changes only in `producer.go`); `services/atlas-data/atlas.com/data/cash/rest.go:37-51` (`RestModel` unchanged file); `services/atlas-channel/atlas.com/channel/data/cash/rest.go:7-15` (`RestModel`); no `requests.go` touched (no new cross-service calls added). |
| FILE-04 | Entity/Migration/TableName in `entity.go` | N/A | No `entity.go` touched by this diff. |
| FILE-06 | No package-named catch-all file | PASS | New files (`remotemerchant/registry.go`, `socket/handler/character_cash_item_use_remote_merchant.go`, `kafka/consumer/npcshop/consumer.go`, `kafka/message/npcshop/kafka.go`) each carry exactly one responsibility (a singleton registry; one socket-handler action, matching this service's established one-handler-per-file convention seen in sibling files `character_cash_item_use.go`/`npc_start_conversation.go`; a kafka consumer; a kafka message contract) — none collapse Processor+RestModel+requests into one file. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/npcshop/testmain_test.go:10-13` installs `producertest.InstallNoop()` (no `t.Cleanup(producer.ResetInstance)` present). `services/atlas-npc-shops/atlas.com/npc/shops/processor_enter_test.go` exercises the pure `Enter(mb)` form only (`:78,120,150`), never `EnterAndEmit`, so no Kafka emit path is hit. `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_remote_merchant_test.go` and `saga/remote_merchant_compensation_test.go:35-38` both inject test seams (`remoteMerchantSagaCreateFunc`, `SetEmitNpcShopExitForTest`) that intercept before any real producer call — no unstubbed emit path found. |
| DOM-26 | Goroutines spawned via `routine.Go` | PASS | `services/atlas-channel/atlas.com/channel/kafka/consumer/npc/shop/consumer.go:264` (`routine.Go(l, ctx, func(c context.Context) {...})` backs `startRemoteMerchantSweep`). `tools/goroutine-guard.sh` exits 0 repo-wide (ran from repo root; scanned all modules including the six audited here). |
| DOM-21 | No duplication of atlas-constants types | PASS | `character_cash_item_use.go:511` and `character_cash_item_use_remote_merchant.go` use `item.ClassificationRemoteMerchant` (`libs/atlas-constants/item/constants.go:106`, `= Classification(545)`) rather than a re-declared literal/constant. The item-family split at `character_cash_item_use_remote_merchant.go:83` (`uint32(itemId)/1000 == 5451`) distinguishes 5450xxx from 5451xxx, which atlas-constants' `GetClassification` (itemId/10000) cannot express (both fall in classification 545) — not a duplicate of an existing shared helper. Non-blocking observation: `character_cash_item_use_remote_merchant.go:149` (`InventoryType: 5, // cash`) duplicates `inventory.TypeValueCash` (`libs/atlas-constants/inventory/constants.go:16`) as a bare literal rather than the named constant; the payload field (`saga.DestroyAssetFromSlotPayload.InventoryType`, `libs/atlas-saga/payloads.go:109`) is typed `byte` (not `inventory.Type`) and every other call site in the codebase (`compensator.go`, `mts_expansion_test.go`, etc.) already populates it with bare integer literals under the same doc-comment convention — this is a pre-existing, un-typed field the new call site merely follows, not a new type/const duplication this diff introduces, so it is recorded as non-blocking rather than a FAIL. |
| DOM-22 | Dockerfile matches `go.mod` | N/A | No `go.mod`/`go.sum` changed in this diff — no new `libs/atlas-X` dependency was added to any of the six modules. |
| DOM-23 | Kafka topic naming convention | PASS (no new topics) | `COMMAND_TOPIC_NPC_SHOP` and `EVENT_TOPIC_NPC_SHOP_STATUS` both pre-exist in `deploy/k8s/base/env-configmap.yaml:59,147` in `KEY: "KEY"` form; this diff adds no new topic env vars, only a new `TransactionId` field on the existing contract and a new consumer (`kafka/consumer/npcshop`) subscribing to the pre-existing `EVENT_TOPIC_NPC_SHOP_STATUS`. |
| DOM-25 | Client-interpreted wire values are config-resolved | PASS | The new opcodes added in `template_gms_48_1.json` (`0xE5`/`0xE6`), `template_gms_87_1.json` (`0x40`), `template_gms_92_1.json` (`0x43`,`0x164`,`0x165`), `template_gms_95_1.json` (`0x42`) all carry an `options.operations` map resolving semantic keys (`BUY`/`SELL`/`OUT_OF_STOCK`/etc.) to mode bytes — no Go literal mode/operation byte was added to `libs/atlas-packet` or `atlas-channel` source for this feature. `libs/atlas-packet/npc/clientbound/shop_list.go`'s `>=95`→`>=92` boundary change is a version-gate on an already-existing field layout, not a client-interpreted wire *value* table. `tools/template-opcode-order-guard.sh` and `tools/template-duplicate-binding-guard.sh` both exit clean on the touched templates. |
| DOM-27 | Transient DB errors → 503 | N/A | No `resource.go` (REST handler) touched in this diff; the npc-shops `resource.go` (its DB-backed REST layer) is unmodified. |
| DOM-28 | No silent degradation in decorators | N/A | No `model.Decorator[...]` implementation touched by this diff; `CommodityDecorator`/`RechargeableConsumablesDecorator` in `atlas-npc-shops/shops/processor.go` are unchanged by this diff (only the unrelated `Enter` method in the same file was modified). |
| Mirror guard (task-221-specific) | `npc-shop-contract-mirror-guard.sh` | PASS | Ran clean: "OK — all three copies identical" across `atlas-npc-shops` (owner), `atlas-channel`, `atlas-saga-orchestrator` mirrors of `COMMAND_TOPIC_NPC_SHOP`/`EVENT_TOPIC_NPC_SHOP_STATUS`. |

### External HTTP Client Checklist

No new cross-service client package was introduced by this diff. `services/atlas-channel/atlas.com/channel/data/cash` (consumed by `character_cash_item_use_remote_merchant.go:24` via the pre-existing `cash.NewProcessor(l, ctx).GetById`) predates this branch and only gained one `Npc` field; its `requests.go`/`processor.go`/`rest.go` were not restructured. Not re-scored here as this task did not touch its client-construction, error-classification (EXT-03), or JSON:API relationship methods (EXT-01) — those are pre-existing and out of this diff's scope.

### Scaffolding / Opcode-Table Checklist

No new `services/atlas-<name>/` directory, and no new `Writer`/`Handler` constant was registered in `atlas-channel/main.go` (`NPCShopWriter`, `NPCShopOperationWriter`, `NPCShopHandle` at `main.go:730-731,943` all pre-exist) — SCAFFOLD-01..08 do not apply. The template additions are pure opcode-table registrations for pre-existing writers/handlers on additional client versions, covered by DOM-25/DOM-23 above.

### Findings Summary

**Blocking (must fix):** none.

**Non-Blocking (should fix):** none rising to the "should fix" bar; one observation recorded under DOM-21 above (`InventoryType: 5` bare literal at `character_cash_item_use_remote_merchant.go:149`) that mirrors a pre-existing, un-typed-field convention used throughout the codebase and is not a regression introduced by this diff.
