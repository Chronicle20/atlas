# Backend Audit — task-128 (item-tag / seal / incubator)

- **Scope:** whole-branch Go review, range `38d4d0ba2..0f555b16a8`
- **Modules:** atlas-inventory, atlas-saga-orchestrator, atlas-tenants, atlas-channel, atlas-storage, atlas-merchant, atlas-data; libs/atlas-constants, libs/atlas-packet, libs/atlas-saga
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/EXT-*/SEC-*)
- **Date:** 2026-07-03
- **Build:** PASS (all 7 service modules `go build ./...` exit 0)
- **Tests:** PASS (all changed modules `go test ./... -count=1`, incl. atlas-channel)
- **go vet:** PASS (inventory, saga-orchestrator, tenants sampled clean)
- **Overall:** NEEDS-WORK (one Important deploy-config gap; everything else PASS)

## Objective Gate (Phase 1)

- `go build ./...`: PASS for all 7 modules.
- `go test ./... -count=1`: PASS for inventory, saga-orchestrator, tenants, storage, merchant, data, channel. No FAIL lines. inventory/compartment 0.466s, inventory/asset 0.012s, saga 0.462s.
- `go vet`: clean on the changed packages.

## Findings

### IMPORTANT — DOM-23: new Kafka topic `EVENT_TOPIC_INCUBATOR_RESULT` not wired into deploy config

- Producer: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/incubator/kafka.go:4`
- Consumer: `services/atlas-channel/atlas.com/channel/kafka/message/incubator/kafka.go:4`
- Missing from: `deploy/k8s/base/env-configmap.yaml` (141 topic entries, `KEY: "KEY"` shape; a comparable recent feature topic `EVENT_TOPIC_GACHAPON_REWARD_WON` is present at line 110). Also absent from `deploy/k8s/overlays/main/kustomization.yaml` and `.../overlays/pr/kustomization.yaml`. No `deploy/` file was changed on this branch.

Why it matters:
- `atlas-kafka-precreate` derives topics to create from the `COMMAND|EVENT_TOPIC_*` keys in the configmap (`deploy/k8s/base/atlas-kafka-precreate.yaml:42`). The incubator topic is never precreated.
- Every real topic is per-environment suffixed (e.g. `EVENT_TOPIC_ASSET_STATUS-main` in `overlays/main/kustomization.yaml:116`). Because `EVENT_TOPIC_INCUBATOR_RESULT` has no configmap entry, the env var is unset in both pods; `topic.EnvProvider` (`libs/atlas-kafka/topic/provider.go:16-19`) falls back to the bare token literal `"EVENT_TOPIC_INCUBATOR_RESULT"`.
- Consequence: producer and consumer both fall back to the same bare literal, so they still agree in a single environment (feature is not a hard outage), BUT the topic is un-suffixed and un-isolated across environments — a PR/ephemeral env sharing brokers publishes/consumes the same bare `EVENT_TOPIC_INCUBATOR_RESULT` as main, leaking incubator-result messages across environments. This is exactly the class of break the topic-config convention exists to prevent.
- The task's `deploy-runbook.md` covers per-tenant config PATCH, reward-pool seed, and channel restart, but does NOT mention adding the topic to `env-configmap.yaml`. Unacknowledged miss.

Fix: add `EVENT_TOPIC_INCUBATOR_RESULT: "EVENT_TOPIC_INCUBATOR_RESULT"` to `deploy/k8s/base/env-configmap.yaml` (and regenerate the overlay configMapGenerator topic lists via `gen-topic-config.sh`).

### MINOR — EXT-01 (latent): incubator client REST model omits relationship setters

- `services/atlas-channel/atlas.com/channel/incubator/rest.go:11-16` — `RewardRestModel` implements `GetName/GetID/SetID` but not `SetToOneReferenceID`/`SetToManyReferenceIDs`.
- Does NOT break today: the upstream `IncubatorRewardRestModel` (`services/atlas-tenants/.../configuration/rest.go:227`) has no `GetReferences`, so the JSON:API response carries no `relationships` block and api2go never invokes the setters.
- Deviates from sibling tenants-config client convention (`services/atlas-channel/atlas.com/channel/transport/route/rest.go:85` implements them). Latent risk if the tenants resource ever adds a relationship. Add no-op setters for defensiveness.

### MINOR — EXT-02: no httptest-backed unmarshal test for the incubator config client

- `services/atlas-channel/atlas.com/channel/incubator/` has `roll_test.go` (PickWeighted unit test) but no `httptest.NewServer` test exercising `requestRewards`/`GetRewards` decode. Consistent with the rest of the codebase (sibling config clients also omit these), so not blocking — noted for parity with EXT-02.

### MINOR — carried-forward items (verified)

- (a) Packet sub-body/writer `String()` omission (`ItemUseItemTag/Seal/Incubator`, `IncubatorResult`): CONFIRMED cosmetic only. The outer `cash/serverbound.ItemUse` (which is logged via `p.String()` in the handler) has `String()`; the sub-bodies are never logged that way, and `IncubatorResult` is clientbound (encoded, not logged). Non-issue.
- (b) gofmt misalignment across inventory/storage/channel asset files and `libs/atlas-constants/item/constants.go`: CONFIRMED pre-existing — every flagged file is already gofmt-dirty at base commit `38d4d0ba2`. Branch did not introduce it. `go vet`/build/test all clean. Recommend a separate repo-wide `gofmt -w`. Non-blocking.
- (c) tenants create→get-all test asserting only `itemId`: minor coverage gap, non-blocking.
- New `item.ItemTag/SealingLock/Incubator/SealingLock{7,30,90,365}Day` constants are unreferenced, but that matches the `constants.go` catalog convention (pre-existing named IDs like `MapleSyrupOneHundredPercent` are also unused). Non-issue.

## Checklist Results (representative)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM (immutable/Clone) | owner copied in Clone + Build | PASS | inventory asset/builder.go:19,168; storage asset/builder.go:18,189 |
| owner threading (entity/rest/kafka/AssetData) | no dropped copy | PASS | inventory administrator.go:20, entity.go, rest.go:73; storage model.go:180,238; saga-orch processor.go:1382,1503; merchant kafka/message/asset/kafka.go:15 |
| compartment mutators (SET_OWNER/APPLY_LOCK) | message.Emit+buffer, admin layer, tenant-safe update | PASS | inventory compartment/processor.go:981,1015; asset/processor.go:271,287,305; administrator.go:61,65 |
| lock-aware expiration | locked asset unlocks, not destroyed | PASS | inventory compartment/processor.go:942-945; asset/processor.go:305 |
| ApplyLock laundering guard | rejects non-lock expiration | PASS | inventory asset/processor.go:290-292; channel handler:184-188 |
| saga handler actions | SetAssetOwner/ApplyAssetLock/IncubatorResult | PASS | saga/handler.go:868-1090 |
| AcceptEvent gating (asset UPDATED) | SetAssetOwner/ApplyAssetLock accept asset.updated; IncubatorResult terminal | PASS | saga/event_acceptance.go:105-107; consumer/asset/consumer.go handleAssetUpdatedEvent |
| compensator reverse-walk | re-create consumed, destroy awarded, reverse order, errors don't abort chain | PASS | saga/compensator.go:1226-1281 |
| DOM-08 POST/PATCH input handler | RegisterInputHandler[IncubatorRewardRestModel] | PASS | tenants configuration/resource.go:853,875,876 |
| DOM-10 test tenant callbacks | RegisterTenantCallbacks | PASS | inventory compartment/processor_test.go:53; storage asset/provider_test.go:41 |
| DOM-18/19 JSON:API | GetName/GetID/SetID, flat request | PASS | incubator/rest.go; tenants configuration/rest.go:236-247 |
| DOM-21 constants reuse | item.Id / item.GetClassification / inventory.Type | PASS | constants.go uses Id(); handler uses item.GetClassification, inventory.TypeFromItemId |
| DOM-24 producer stub in emitting tests | tests exercise buffer `Method(mb)`, not `AndEmit` | PASS | compartment/processor_test.go uses message.NewBuffer, no unstubbed emit (0.466s) |
| Keyed Redis outside atlas-redis | none introduced | PASS | new logic is DB-backed; registries use atlas-redis lib |
| EXT-04 URL via RootUrl | requests.RootUrl("TENANTS") | PASS | incubator/requests.go:15 |
| SEC-* | n/a (no auth/token/redirect surface) | N/A | — |

## Design point verified (not a defect)

The ItemTag/Seal cash-item sagas end with `SetAssetOwner`/`ApplyAssetLock` as the terminal step. `DispatchCashItemUseRollbacks` defines no inverse for those two actions — correct, because a terminal step that fails never reached Completed (so needs no undo) and a terminal step that completes means the saga succeeded. The reverse-walk correctly covers the earlier `DestroyAsset`/`DestroyAssetFromSlot`/`AwardAsset` steps.

## Summary

### Blocking / Important
- DOM-23: `EVENT_TOPIC_INCUBATOR_RESULT` missing from `deploy/k8s/base/env-configmap.yaml` (and both overlays) → topic not precreated and not per-environment isolated.

### Non-blocking (Minor)
- EXT-01: incubator `RewardRestModel` missing no-op relationship setters (latent).
- EXT-02: no httptest unmarshal test for the incubator config client (consistent with codebase).
- Pre-existing gofmt misalignment (not branch-introduced); recommend repo-wide `gofmt -w`.
- tenants create→get-all test asserts only `itemId`.
