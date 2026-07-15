# MTS Marketplace — Implementation Context

Companion to `plan.md`. Read this first; it captures the key files, decisions, and
dependencies an implementer needs before touching code. Every path is repo-relative
to the worktree root `.worktrees/task-102-mts-marketplace/`.

## Inputs

- `prd.md` (approved) — product requirements.
- `design.md` (approved) — architecture, alternatives, sequencing.
- `research-scaffold.md` — IDA-verified protocol facts (opcodes, mode tables,
  custody model, economic knobs).

## Scope reality

This is a large, multi-subsystem feature: a new Go service (`atlas-mts`), new shared
saga actions, saga-orchestrator wiring, atlas-channel socket wiring, an atlas-tenants
config resource, and an atlas-ui surface. The plan is sequenced into 10 phases that
mirror `design.md §15`. **Each phase is independently buildable**; do not start a
phase until the prior one's gates are green. The phases are large enough that they
should each be executed as their own subagent-driven run with a build/test gate at
the boundary.

## The hard gate — Phase 0 (packet verification)

**No in-game MTS flow code (channel handlers, saga-triggering, settlement) is written
until the serverbound packet cells promote in `docs/packets/audits/STATUS.md`.**
Phase 0 is executed with the `packet-verifier` agent / `/verify-packet` playbook
(`docs/packets/audits/VERIFYING_A_PACKET.md`), one cell per packet × version, batched
per IDB. Per the dispatcher-family rule
([[feedback_dispatcher_mode_byte_is_false_pass]]), **every `ITC_OPERATION` mode arm
gets its own byte fixture** — enumerating mode bytes is a false pass.

CRITICAL grounding rule (CLAUDE.md "No Inventing"): the per-mode packet **decode read
order** must come from the promoted Phase-0 fixtures / IDA, never from memory or
guesswork. Plan tasks that decode a packet body reference "the Phase-0-verified read
order for mode X" rather than inventing bytes. If a read order is not yet verified,
that handler arm is blocked — stop and escalate, do not fabricate.

Phase 0 also resolves two design questions:
- **§9.1 real-time bidding** — is there a server-*pushed* auction-state/outbid packet?
  If yes → live outbid path; if no → escrow-highest-bid-wins-at-expiry (the default
  baseline). The custody/settlement core is identical either way.
- **§9.4 jms scope** — record the supported jms surface; omit clientbound-absent flows.

task-096 already byte-verified the **clientbound** result writers
(`MTS_OPERATION`/`MTS_OPERATION2`); those are reused, not re-verified.

## Key existing patterns to mirror (with file:line anchors)

### Service skeleton — mirror `atlas-gachapons` (REST) + `atlas-cashshop` (Kafka)
- Module layout: `services/atlas-mts/atlas.com/mts/`, `go.mod` module name `atlas-mts`.
- Immutable model: `services/atlas-gachapons/atlas.com/gachapons/gachapon/model.go`
  (private fields + getters).
- Builder: `.../gachapon/builder.go` (`NewBuilder(...)` → `.Set*()` → `.Build()` with
  nil/empty validation).
- Entity + surrogate-UUID PK + `(tenant_id, id)` unique index + explicit `Migration`:
  `.../gachapon/entity.go` (note the `migrateToSurrogatePK` precedent for the
  slug-collision and column-order bug families —
  [[bug_tenant_table_slug_only_pk_collides]],
  [[bug_baseline_restore_column_order_drift]]).
- Provider (query): `.../gachapon/provider.go` (`database.Query`/`SliceQuery`,
  `modelFromEntity`).
- Administrator (mutation): `.../gachapon/administrator.go` (explicit column maps;
  `database.ExecuteTransaction`).
- Processor `Interface`+`Impl`, `NewProcessor(l, ctx, db)`,
  `db.WithContext(p.ctx)`: `.../gachapon/processor.go`.
- REST (JSON:API): `.../gachapon/rest.go` + `.../gachapon/resource.go`
  (`InitResource(si)(db)`, `rest.RegisterHandler`/`RegisterInputHandler[M]`,
  `tenant.MustFromContext`).
- Test pattern (Builder-based, sqlite in-memory, **no `*_testhelpers.go`**):
  `.../gachapon/processor_test.go` and `.../test/processor.go`.
- REST-only `main.go`: `services/atlas-gachapons/atlas.com/gachapons/main.go`.
- Kafka + consumer/producer `main.go`:
  `services/atlas-cashshop/atlas.com/cashshop/main.go` (consumer manager
  `consumer.GetManager().AddConsumer`, curried `InitConsumers(l)(cmf)(groupId)` +
  `InitHandlers(l)(db)(register)`, producer teardown).
- Generic Kafka envelope (`Command`, `StatusEvent[E]`, topic env constants):
  `services/atlas-cashshop/atlas.com/cashshop/kafka/message/wallet/kafka.go`.
- Lazy per-tenant config cache: `atlas-cashshop` `configuration/registry.go`
  (`GetTenantConfig`, `sync.RWMutex` double-check, default on miss).

### Expiration ticker — mirror `atlas-asset-expiration`
- `services/atlas-asset-expiration/atlas.com/asset-expiration/task/periodic.go`
  (`PeriodicTask`, `time.Ticker` + `stopCh` + `sync.WaitGroup`, env interval,
  `tdm.TeardownFunc(task.Stop)`). The MTS ticker is **DB-driven** (sweeps offline
  sellers' listings), unlike the expiration service's in-memory session iteration —
  it enumerates active tenants and queries each tenant's `ends_at < now` listings.
  Reconstruct tenant context per iteration via `tenant.Create(...)` +
  `tenant.WithContext(...)`.

### Saga actions — mirror the cash-shop custody family
- Shared lib: `libs/atlas-saga/model.go` (Action constants — see the cash-shop block
  lines 123–126: `TransferToCashShop`/`WithdrawFromCashShop`/`AcceptToCashShop`/
  `ReleaseFromCashShop`; and `SagaType`/`Type` enum line ~19) and
  `libs/atlas-saga/payloads.go` (`TransferToCashShopPayload` line 518,
  `WithdrawFromCashShopPayload` line 529, `ReleaseFromCharacterPayload` line 540,
  `AwardMesosPayload`, `AwardCurrencyPayload`).
- Orchestrator (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/`):
  - `saga/handler.go` — action handler switch (`GetHandler`, cash-shop cases).
  - `saga/event_acceptance.go` — `acceptanceTable` (which event kind completes/fails
    each action; composite actions map to `{}`).
  - `saga/compensator.go` — compensator inverses + reverse-walk dispatch.
  - `saga/processor.go` — `expandTransferToCashShop`/`expandWithdrawFromCashShop`
    composite expansion; the duplicate-step-completion idempotency guard
    (`stepCompletedWithResultOnce`).
  - `cashshop/processor.go` — `AcceptAndEmit`/`ReleaseAndEmit`/`AwardCurrencyAndEmit`
    dispatch to `COMMAND_TOPIC_CASH_COMPARTMENT`.
  - Saga timeout: `saga/builder.go` `SetTimeout(...)`, default 5m
    (`saga/store.go`). MTS sagas are short/fixed-length but **must set timeout
    explicitly** via `base + perStep*N` and record `N`
    ([[bug_preset_creation_saga_flat_timeout]]).
  - Reverse-walk integration test shape: `saga/preset_integration_test.go`.

### Channel wiring — mirror cash-shop / messenger / storage
- Migration entry handler: `services/atlas-channel/.../socket/handler/cash_shop_entry.go`
  (`CashShopEntryHandleFunc`: save character, leave channel/map, multi-stage announce).
- Mode dispatcher: `.../socket/handler/messenger_operation.go`
  (`MessengerOperationHandleFunc` + `isMessengerShopOperation`, which resolves the
  sub-op from `options["operations"][KEY]`).
- Meso fee via saga: `.../socket/handler/storage_operation.go` `handleRetrieveAsset`
  (builds `saga.Saga{Steps:[]saga.Step{...AwardMesos, Amount:-fee...}}`,
  `saga.NewProcessor(l,ctx).Create(sagaTx)`).
- Handler/validator registration: `services/atlas-channel/.../channel/main.go`
  (`handlerMap[...]=...Func`, `produceValidators()`), and
  `libs/atlas-opcodes/producer.go` `BuildHandlerMap` (**missing validator → silently
  skipped** — every handler needs a validator,
  [[bug_socket_handler_missing_validator_silently_dropped]]).
- Existing task-096 clientbound MTS writers:
  `libs/atlas-packet/field/clientbound/mts_operation.go` (mode dispatcher, 35 mode
  structs), `.../mts_operation2.go`, `libs/atlas-packet/field/mts_operation_body.go`
  (`WithResolvedCode("operations", KEY, ...)`).
- Channel saga producer: `services/atlas-channel/.../channel/saga/processor.go`
  (`Create(s) → COMMAND_TOPIC_SAGA`).

### atlas-tenants config resource — mirror "routes"/"vessels"
- `services/atlas-tenants/atlas.com/tenants/configuration/`: `rest.go` (RestModel +
  Transform/Extract + JsonData), `processor.go` (CRUD + provider + Seed methods),
  `resource.go` (handlers + `RegisterRoutes`), `kafka.go` (event constants + provider),
  `provider.go`, `seed.go`, `administrator.go` (generic — no change),
  `mock/processor.go` (must add new methods).
- Path param parser: `services/atlas-tenants/atlas.com/tenants/rest/handler.go`
  (`ParseRouteId`/`ParseVesselId` → add `ParseMtsConfigId`).
- Generic `Entity`/`Model` (`configurations` table: tenant_id, resource_name,
  resource_data JSONB) needs **no change**.

### Socket opcode + `operations` mode-table seeds (all five versions)
- `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`,
  `_gms_84_1.json`, `_gms_87_1.json`, `_gms_95_1.json`, `_jms_185_1.json`
  (`socket.handlers[]` = {opCode, validator, handler, options},
  `socket.writers[]` = {opCode, writer, options}).
- Per-version `operations` mode tables are **version-dependent** (non-uniform shift);
  a missing per-version table makes `ResolveCode` return 99 and crashes the client
  ([[bug_operations_mode_tables_missing_v87_v95_jms]]). Populate each from that
  version's dispatcher switch, IDA-verified in Phase 0 — **not** copied from v83.
- Per-version opcodes (research-scaffold §4): ENTER_MTS SB v83 0x9C / v87 0xA4 /
  v95 0xB4; ITC_STATUS_CHARGE 0xFB/0x109/0x132; ITC_QUERY_CASH_REQUEST
  0xFC/0x10A/0x133; ITC_OPERATION 0xFD/0x10B/0x134; MTS_OPERATION2 CB
  0x15B/0x170/0x19B; MTS_OPERATION CB 0x15C/0x171/0x19C. v84 and jms opcodes are
  Phase-0 deliverables.

### Service registration (3 hand-synced places + k8s)
- `.github/config/services.json` — add `atlas-mts`.
- `docker-bake.hcl` `go_services` — add `"atlas-mts"` (HCL can't read JSON;
  hand-synced — [[reference_docker_bake_hand_synced]]).
- `go.work` — add `./services/atlas-mts/atlas.com/mts`.
- `deploy/k8s/base/atlas-mts.yaml` — mirror `atlas-gachapons.yaml`
  (`DB_NAME=atlas-mts`, `atlas-env`). **No new socket ports** (channel-migrated
  stage). If a readiness mount is added, probe path is `/api/readyz`
  ([[bug_readiness_probe_path_under_api_basepath]]).
- **No new shared lib anticipated** → no root-`Dockerfile` COPY edits. Saga
  actions/payloads go in the existing `libs/atlas-saga`. If that changes, add the two
  COPY lines + `go.work` line per CLAUDE.md.

## Economic knobs (tenant config defaults — research-scaffold §2)

`listingFee` 5000 meso, `commissionRate` 0.10 (buyer-markup), `maxActiveListings` 10,
`minLevel` 10, `auctionMinHours` 24, `auctionMaxHours` 168 (1h step), `priceFloor`
110 NX (IDA-verified), `pageSize` 16, `minBidIncrement` (config).

## Currency / wallet facts

Two-bucket wallet (NX Prepaid + Maple Points; IDA-verified). `ADJUST_CURRENCY` via
`COMMAND_TOPIC_WALLET` / `EVENT_TOPIC_WALLET_STATUS`, `currencyType` 2=points /
3=prepaid, `amount int32` signed. Buyer debited marked-up price in Prepaid; seller
credited list value in Points; commission (markup − list value) is the sink (never
credited).

## Verification gates (CLAUDE.md — run from worktree root before PR)

1. `go test -race ./...` clean in every changed module.
2. `go vet ./...` clean in every changed module.
3. `go build ./...` clean in every changed service.
4. `docker buildx bake atlas-mts` (+ every service whose `go.mod` was touched:
   atlas-saga-orchestrator, atlas-channel, atlas-tenants).
5. `tools/redis-key-guard.sh` clean from the repo root.
6. atlas-ui: `npm run build` (type-checks tests too —
   [[reference_atlas_ui_build_typechecks_tests]]) + `npm test`; gate on
   build+test + no-new-lint-errors (lint baseline is pre-broken —
   [[reference_atlas_ui_npm_nvm_and_lint_baseline]]; source nvm 22 first).
7. Code review (`superpowers:requesting-code-review`) **before** opening the PR.

## Operational rollout note (not code)

Existing tenants do not retroactively receive new handler/writer opcodes or
`operations` tables from a seed template; the live tenant config must be patched and
the channel restarted ([[bug_new_opcodes_not_in_live_tenant_config]]). Capture this
as a rollout-checklist step.
