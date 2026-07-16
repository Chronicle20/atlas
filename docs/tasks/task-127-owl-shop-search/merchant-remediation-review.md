# Backend Guidelines Review — Merchant-Lifecycle Remediation Commits

- **Scope:** commits `08e01ec74..HEAD` only (6 commits: 08e01ec74, 0a1c5cbb2, 4c14d19a7, 6432c9203, c4aca9b50, 9a5abb75f)
- **Guidelines Source:** backend-dev-guidelines skill (DOM-* / FILE-* / EXT-* checklists)
- **Date:** 2026-07-14
- **Build:** PASS (`go build ./...` clean in libs/atlas-packet, atlas-channel, atlas-merchant)
- **Tests:** PASS (`go test ./... -count=1` — all three modules, zero failures; atlas-merchant/shop 10.7s is miniredis+sqlite volume, not producer hangs — see DOM-24)
- **Overall:** NEEDS-WORK — 5 diff-introduced FAILs, 2 pre-existing FAILs in touched packages, 4 warnings

## Packages in scope

| Package | Classification |
|---|---|
| `services/atlas-merchant/.../shop` | Domain (model.go) |
| `services/atlas-merchant/.../frederick` | Domain (model.go; only rest.go added by this diff) |
| `services/atlas-merchant/.../kafka/consumer/character` | Support |
| `services/atlas-channel/.../merchant` (+ mock) | Support / REST client |
| `services/atlas-channel/.../kafka/consumer/merchant` | Support |
| `services/atlas-channel/.../socket/handler`, `socket/model` | Support |
| `libs/atlas-packet/interaction`, `libs/atlas-packet/merchant` | Packet codec lib |

---

## Blocking (diff-introduced FAILs)

### F1 — DOM-14 / DOM-13: handler calls a cross-domain provider directly (Important)
`services/atlas-merchant/atlas.com/merchant/shop/resource.go:275` — `handleGetCharacterFrederick` calls
`frederick.HasItemsOrMesos(characterId)(db.WithContext(d.Context()))()`, a **provider** function
(`frederick/provider.go:32`), directly from a REST handler. anti-patterns.md: `resource.go → provider.go`
is WRONG — handlers must call processors. The documented circular-dependency exception does not apply:
there is no circularity (shop already imports frederick) and frederick has a Processor
(`frederick/processor.go:31`). Compounding it, the route `/characters/{id}/frederick` and its handler live
in the **shop** package while serving a **frederick** resource (DOM-13: cross-domain logic in a handler).
Fix: give frederick its own `resource.go` (or at minimum route through a frederick processor method such
as `HasPending(characterId)`); shop's processor already shows the correct call shape
(`shop/processor.go:222` goes through the processor layer).

### F2 — DOM-04: no `Transform` for the new frederick REST model (Important)
`services/atlas-merchant/atlas.com/merchant/frederick/rest.go` (new file) defines `StatusRestModel` with no
`Transform` function; the DTO is assembled inline inside the shop handler
(`shop/resource.go:280-283`). file-responsibilities.md places model→REST transformation in the owning
package's `rest.go`. Same cluster as F1 — fixing F1 properly resolves this.

### F3 — EXT-01: client target struct missing relationship interface methods (Important)
`services/atlas-channel/atlas.com/channel/merchant/rest.go:143-160` — the new `FrederickStatusRestModel`
implements only `GetName/GetID/SetID`; it lacks `SetToOneReferenceID` and `SetToManyReferenceIDs`.
The sibling `RestModel` in the **same file** implements both (rest.go:75, rest.go:79). Per EXT-01
(libs/atlas-rest guidance, task-037 precedent), both must be present even as no-ops — without them api2go
fails with a misleading "not found" the moment the upstream response carries a `relationships` block.

### F4 — EXT-02: no httptest-backed integration test for the new cross-service call (Important)
The new client path `requestFrederickStatus` (`channel/merchant/requests.go:52-54`) →
`HasFrederickPending` (`channel/merchant/processor.go:72-78`) has no `httptest.NewServer`-backed test
anywhere in the package (grep: zero `httptest` hits under `channel/merchant/`). EXT-02 requires a fixture
test that exercises real JSON:API unmarshal of the upstream shape; the mock
(`mock/processor.go:65-70`) bypasses unmarshal and does not satisfy this.

### F5 — FILE-05: expiration task re-implements the provider query inline (Important)
`services/atlas-merchant/atlas.com/merchant/shop/task.go:33-35` duplicates the exact WHERE clause of
`getExpired` (`shop/provider.go:82-88`). This remediation had to edit **both copies in lockstep**
(same `state IN (Draft, Open, Maintenance)` + Go-side cutoff change in each) — precisely the drift hazard
the provider layer exists to prevent. The provider is db-parameterized; the task can call
`getExpired()(t.db.WithContext(noTenantCtx))()` and keep the cross-tenant behavior. Pre-existing shape,
but the diff modified the duplicated query rather than consolidating it, so it is graded in-scope.

## Pre-existing FAILs in touched packages (not introduced by these commits)

- **DOM-01 (frederick):** no `builder.go` — package predates the diff; only `rest.go` was added here.
- **DOM-02 (frederick):** no `ToEntity()` on `ItemModel`/`MesoModel` (`frederick/entity.go` has `Migration`
  only; `MakeItem`/`MakeMeso` live in `model.go`, satisfying DOM-03's spirit but not the `entity.go`
  placement). Recorded as debt; out of this remediation's blast radius.

## Non-blocking (warnings)

- **W1 — error-kind conflation in occupancy resolution.** `shop/processor.go:947` —
  `if shopId, err := vr.GetShopForCharacter(...); err == nil { return shopId, nil }` swallows **every**
  visitor-registry error (the registry propagates raw Redis errors, `visitor/registry.go:86-93`), treating
  a transient Redis failure the same as "not visiting" and falling through to owner-occupancy. Distinguish
  miss from failure (sentinel error) so a Redis blip cannot silently mis-route a visiting owner to their
  own shop. (DOM-28-adjacent; no data is dropped, so graded WARN.)
- **W2 — enum placement.** `shop/logout_policy.go` defines `LogoutOutcome` (enum) + `LogoutAction`
  (state-transition policy). file-responsibilities.md designates `state.go` for "domain-specific enums …
  state transition helpers", and `shop/state.go` already holds `ShopType`/`State`/`CloseReason`. Defensible
  as a single-purpose file under FILE-06, hence WARN not FAIL — but consolidation into `state.go` matches
  the table.
- **W3 — gms_48 template inconsistency surfaced by the new reject path.** `template_gms_48_1.json` routes
  `CharacterInteractionHandle` with a `CREATE` op but has **no** `CharacterInteraction` writer and no
  `enterError` table. On a v48 tenant the new `rejectCreate` reply is silently lost (verified: the announce
  fails at writer lookup — `libs/atlas-socket/writer/writer.go:27-34`, `session/processor.go:243-249` — so
  ResolveCode's 99 never reaches the wire). Pre-existing template state; worth reconciling (either route the
  writer + enterError for v48 or drop the CREATE handler op).
- **W4 — hand-mirrored state bytes.** `channel/merchant/model.go:53-58` adds `StateDraft=1`/`StateClosed=4`
  mirroring `atlas-merchant/shop/state.go` by hand (comment documents the mirror). No atlas-constants
  equivalent exists, so DOM-21 passes, but the mirror has no compile-time link — drift risk if merchant
  states ever change.

## Checklist results (evidence)

### DOM-25 — client wire values config-resolved: PASS
- New `HiredMerchantOperationErrorUnknownBody` resolves its mode via
  `atlas_packet.WithResolvedCode("operations", HiredMerchantOperationModeErrorUnknown, …)`
  (`libs/atlas-packet/merchant/operation_body.go:41-45`); semantic keys are strings
  (`operation_body.go:17-24`); the literals 7/8/9/11 appear only in comments and byte-fixture tests.
- Handler reply paths in `hired_merchant_operation.go` use only the config-resolved body funcs
  (OpenShop/ErrorUnknown/ErrorRetrieveFromFredrick/ErrorUnableToOpenTheStore).
- `rejectCreate` resolves via `ResolveCode(…, "enterError", UNABLE)`
  (`libs/atlas-packet/interaction/clientbound/interaction_body.go:73-80`).
- Template coverage verified per version: `operations` keys (OPEN_SHOP/ERROR_UNKNOWN/
  ERROR_RETRIEVE_FROM_FREDRICK/ERROR_UNABLE_TO_OPEN_THE_STORE) present in every template that routes the
  `HiredMerchantOperation` writer (gms_61/72/79/83/84/87/95, jms_185); `enterError` present in every
  template where the CREATE arm is both routed and writable (gms_61/72/83/84; v79 has the table, arm not
  routed; v87/95/jms_185 route neither — see W3 for the v48 anomaly). Rollout tooling for live tenants
  exists on the branch (`docs/tasks/task-127-owl-shop-search/deployment.md`,
  `patch-hired-merchant-writers.sh`).
- The room `position` byte (`libs/atlas-packet/interaction/room.go:163-166`) is recipient data (slot
  index), not a lookup-switch code; encoded inside codec internals with IDA citations.

### DOM-21 — atlas-constants reuse: PASS
`character_interaction.go:97-105` uses `item.GetClassification(item.Id(...))`,
`item.ClassificationStorePermit`, `item.ClassificationHiredMerchant` from
`libs/atlas-constants/item/constants.go:73,83,129` — no service-local `/10000` reimplementation.
`LogoutOutcome` has no shared equivalent (service policy enum).

### Interface/mock sync: PASS
`HasFrederickPending` added to `channel/merchant/processor.go:24` (interface), `processor.go:72-78`
(impl), and `mock/processor.go:18,65-70` (func field + nil-check method). All modules compile;
`go test ./merchant/...` passes.

### DOM-24 — producer stubbing: PASS
atlas-merchant emits go through the transactional outbox (`shop/processor.go:1127-1133`
`message.Emit(outbox.EmitProvider(...))`), not the raw Kafka producer, so test emits are DB writes;
new tests (`processor_test.go:743+`) call only pure `CreateShop`. `atlas-channel/merchant` tests run in
0.007s with no `ProviderImpl` call sites in tests.

### DOM-26 — goroutines: PASS. No bare `go` statements in the diff.

### DOM-27 — transient errors: PASS
New handler uses `server.WriteErrorResponse(d.Logger())(w)(err)` (`shop/resource.go:278`); classifier
registered at `atlas-merchant/main.go:83`.

### DOM-10 / tenant scoping: PASS
`RegisterTenantCallbacks` in both test setups (`shop/processor_test.go:36`,
`frederick/processor_test.go:28`). New queries: `hasItemsOrMesos` invoked with
`db.WithContext(...)` at both call sites (`shop/processor.go:222`, `shop/resource.go:275`);
`activeShops.Get(p.ctx, p.t, characterId)` is tenant-keyed (`shop/processor.go:952`); the expiration task
intentionally uses `database.WithoutTenantFilter` with per-row tenant reconstruction
(`shop/task.go:28,48-53`) — the documented cross-tenant pattern.

### Immutability / builder discipline: PASS
`Room.SetOwnerLedger` is a value-receiver copy-and-return (`libs/atlas-packet/interaction/room.go:130-136`);
channel `merchant.Model` keeps private fields + getters (`model.go:44`); `LogoutAction` is pure.

### DOM-20 — tests: PASS
`shop/logout_policy_test.go:19-37` is table-driven over all 8 (type × state) cells; the wire-shape change
is pinned by byte fixtures (`libs/atlas-packet/interaction/room_test.go`: owner/visitor round-trips,
`TestRoomPositionByteSemantics` asserting `b[2] == 0x00` owner / `0x03` visitor slot;
updated v48/61/72/79 fixtures). Note: the changed clientbound Room codec backs no tier-1 coverage-matrix
row (`status.json` tracks only InteractionChat/UpdateMerchant clientbound), so no pinned evidence cell was
invalidated.

### Other
- DOM-12: no `os.Getenv` in touched handlers. DOM-15/16: no direct writes in handlers; no new write ops.
- DOM-08/19: no new POST/PATCH endpoints (frederick route is GET via the shared `registerHandler`).
- DOM-22/23: no `go.mod`, Dockerfile, topic, or manifest changes in the range — N/A.
- SCAFFOLD-*: no new service, no new Writer/Handler constants registered in channel `main.go` — N/A.
- EXT-03: `HasFrederickPending` propagates the original error; the caller fails closed
  (`hired_merchant_operation.go` replies ERROR_UNABLE on query failure) — PASS.
- EXT-04: URL via `getBaseRequest()` = `requests.RootUrl` + `FrederickResource` — PASS.
- Dead-code cleanup: the removed `ToPacketRoom` trio + interface method left no orphans
  (`mini_room.go` `messages` field still used by `Enter` at lines 255-256).

## Summary

| | Count |
|---|---|
| Diff-introduced FAIL | 5 (F1 DOM-14/13, F2 DOM-04, F3 EXT-01, F4 EXT-02, F5 FILE-05) |
| Pre-existing FAIL (touched pkgs) | 2 (DOM-01, DOM-02 — frederick) |
| WARN | 4 (W1 registry error conflation, W2 enum placement, W3 gms_48 template, W4 mirrored state bytes) |
| Overall | **NEEDS-WORK** |

---

## Remediation applied (2026-07-14, same branch)

- **F1/F2 FIXED:** the Fredrick status endpoint moved into the frederick domain — `frederick/resource.go` (route + handler via `Processor.HasPending`), `frederick/rest.go` `TransformStatus`, registered in `main.go`; the shop package no longer touches frederick internals.
- **F3 FIXED:** `FrederickStatusRestModel` gained `SetToOneReferenceID`/`SetToManyReferenceIDs` stubs (channel `merchant/rest.go`).
- **F4 FIXED:** `TestHasFrederickPending` (channel `merchant/rest_test.go`) exercises URL construction + JSON:API unmarshal against an httptest server.
- **F5 FIXED:** `ExpirationTask.Run` now consumes the `getExpired()` provider (single source of truth for the expiry predicate, run with `WithoutTenantFilter`).
- **W1 FIXED:** `GetShopForCharacter` only falls through to owner occupancy on `atlasredis.ErrNotFound`; other registry errors propagate.
- **W2 FIXED:** `LogoutOutcome`/`LogoutAction` moved into `shop/state.go` (file deleted).
- **W3 ACCEPTED:** gms_48 has no shop feature (no CUIItemUpgrade-era dialogs); the CREATE reject failing at writer lookup on v48 is the correct outcome. No template change.
- **W4 ACCEPTED:** the channel-side state-byte mirror is inherent to the REST boundary (services share no domain module); values are wire-contract documented on both sides. Promoting shop state to atlas-constants is a candidate follow-up, not done here.
- Pre-existing FAILs in touched packages are out of scope for this remediation and remain recorded above.
