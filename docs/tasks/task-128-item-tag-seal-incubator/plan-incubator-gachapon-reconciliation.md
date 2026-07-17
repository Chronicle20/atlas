# Incubator → gachapons reconciliation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Source the incubator's reward pool and roll from the existing `atlas-gachapons` DDD service (each Pigmy Egg = an `incubator`-kind gachapon) instead of the flat `incubator-rewards` tenant-config, and retire that config + its UI — without changing classic-gachapon behavior.

**Architecture:** Add a `kind` discriminator to the gachapon machine and an optional per-item `weight` to its rewards; the roll honors `weight` (single-stage weighted = incubator) when set and falls back to the existing tier roll (gachapon) when unset. The channel's inline incubator roll is replaced by a `POST /gachapons/rewards/select` call keyed by egg id; the `IncubatorUse` saga and version-gated `INCUBATOR_RESULT` packet are unchanged. The flat `incubator-rewards` tenant-config resource and its atlas-ui page are removed, folded into the existing gachapon admin.

**Tech Stack:** Go (atlas-gachapons, atlas-channel, atlas-tenants), GORM, JSON:API (api2go), Kafka sagas; TypeScript/React (atlas-ui, Vitest).

**Spec:** `design-incubator-gachapon-reconciliation.md` (decisions §6). PRD addendum: `prd.md` §11.

## Global Constraints

- **Behavior-preserving for classic gachapon.** Existing gachapons have no per-item `weight` (defaults to 0) → the tier roll (`selectTier` → uniform-within-tier + global pool) must be byte-for-byte the current path. `atlas-gachapons` + `atlas-consumables` suites stay green.
- **No service rename this task.** `atlas-gachapons` / module `gachapons` / topic `gachapon_reward_won` / DB name all unchanged (rename is a separate follow-up PR — spec §4).
- **Incubator kind is tier-agnostic and machine-local.** An `incubator`-kind roll draws only that machine's items (all tiers), weighted by `weight`; it does NOT merge the shared `global` pool.
- **Version-gated `INCUBATOR_RESULT` is untouched.** `libs/atlas-packet/incubator/clientbound/result.go` and the saga `IncubatorResult` step already carry `gachaponItemID = eggId`; do not modify them.
- **Service URL resolution:** the channel resolves the gachapons service via `requests.RootUrl("GACHAPONS")` relying on the `BASE_SERVICE_URL` fallback — do NOT add a hard-coded `GACHAPONS_SERVICE_URL` override (memory `bug_service_url_hardcoded_base_namespace`).
- **DDD patterns:** immutable model + Builder; Interface+Impl processor `NewProcessor(l, ctx[, db])`; JSON:API resource via api2go; no `*_testhelpers.go` (use Builders in tests).
- **Verification:** `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake` for `atlas-gachapons`, `atlas-channel`, `atlas-tenants` (their `go.mod`s are touched); atlas-ui `npm run build` + `npm run test`; `tools/redis-key-guard.sh` + `tools/goroutine-guard.sh` clean.

---

## Task 1: `kind` discriminator on the gachapon machine

**Files:**
- Modify: `services/atlas-gachapons/atlas.com/gachapons/gachapon/entity.go` (add column)
- Modify: `services/atlas-gachapons/atlas.com/gachapons/gachapon/model.go` (field + getter)
- Modify: `services/atlas-gachapons/atlas.com/gachapons/gachapon/builder.go` (SetKind)
- Modify: `services/atlas-gachapons/atlas.com/gachapons/gachapon/provider.go` (entity→model map)
- Modify: `services/atlas-gachapons/atlas.com/gachapons/gachapon/resource.go` (JSON:API `kind` attribute, in + out)
- Test: `services/atlas-gachapons/atlas.com/gachapons/gachapon/*_test.go`

**Interfaces:**
- Produces: `Model.Kind() string` (`"gachapon"` | `"incubator"`); `Builder.SetKind(string)`; entity column `Kind string gorm:"not null;default:gachapon"`.

- [ ] **Step 1: Failing test** — a `gachapon.Model` built without `SetKind` reports `Kind() == "gachapon"` (default); with `SetKind("incubator")` reports `"incubator"`. Add to the model/builder test.
- [ ] **Step 2: Run it, confirm FAIL** (`Kind` undefined).
- [ ] **Step 3: Implement** — add `kind string` to `Model` + `Kind()` getter; `Builder.kind` defaulting to `"gachapon"` in the builder constructor + `SetKind`; `entity.Kind` column with `default:gachapon`; map it in the provider both directions and in `resource.go` attributes. AutoMigrate adds the column; the `default:gachapon` backfills existing rows.
- [ ] **Step 4: Run tests, confirm PASS.**
- [ ] **Step 5: Commit** `feat(gachapons): add kind discriminator to gachapon machine`.

## Task 2: optional `weight` on the gachapon item

**Files:**
- Modify: `services/atlas-gachapons/atlas.com/gachapons/item/entity.go` (`Weight uint32 gorm:"not null;default:0"`)
- Modify: `item/model.go` (`weight` field + `Weight()` getter), `item/builder.go` (`SetWeight`), `item/provider.go` (map), `item/resource.go` (JSON:API `weight` attr)
- Add: `item/provider.go` — `GetByGachaponId(gachaponId string) model.Provider[[]Model]` (all tiers, no tier filter)
- Test: `item/*_test.go`

**Interfaces:**
- Produces: `Model.Weight() uint32`; `Builder.SetWeight(uint32)`; `item.Processor.GetByGachaponId(gachaponId)` returning all items for the machine regardless of tier.

- [ ] **Step 1: Failing test** — item built without `SetWeight` → `Weight() == 0`; with `SetWeight(50)` → `50`. And `GetByGachaponId` returns items across multiple tiers for one gachaponId (seed two items with different tiers, assert both returned).
- [ ] **Step 2: Run, confirm FAIL.**
- [ ] **Step 3: Implement** the field/getter/builder/provider/resource mapping and the `GetByGachaponId` query (`WHERE tenant_id = ? AND gachapon_id = ?`).
- [ ] **Step 4: Run, confirm PASS.**
- [ ] **Step 5: Commit** `feat(gachapons): add optional per-item weight + GetByGachaponId`.

## Task 3: weight-aware roll + incubator-kind branch

**Files:**
- Modify: `services/atlas-gachapons/atlas.com/gachapons/reward/processor.go`
- Test: `reward/processor_test.go`

**Interfaces:**
- Consumes: `gachapon.Model.Kind()` (Task 1), `item.Processor.GetByGachaponId` + `item.Model.Weight()` (Task 2).
- The public `SelectReward(gachaponId string) (Model, error)` signature is unchanged.

- [ ] **Step 1: Failing tests** —
  - `selectItem` becomes weight-aware: with a pool where item A weight 1, item B weight 0-treated-as… (define: if ANY item has weight>0, weight 0 items are excluded/uniform? Decision: when total weight > 0, selection is weighted by `weight` and zero-weight items are unreachable; when total weight == 0, fall back to current uniform pick). Test both: a pool with weights {A:1,B:3} never returns anything outside {A,B} and honors the distribution deterministically by stubbing the RNG boundary; a pool with all weights 0 uses uniform (existing behavior preserved).
  - `SelectReward` on an `incubator`-kind gachapon draws only that machine's items (all tiers, no `global` merge) weighted by `weight`, and never calls `selectTier`. Seed an incubator gachapon with two weighted items + a `global` row that must NOT appear; assert only machine items are reachable.
  - `SelectReward` on a `gachapon`-kind machine is unchanged (tier path + global merge).
- [ ] **Step 2: Run, confirm FAIL.**
- [ ] **Step 3: Implement** —
  - Add `Weight uint32` to `poolItem`; thread `mi.Weight()` through `getMergedPool`.
  - Rewrite `selectItem` to weighted selection over `pool[i].Weight` when `sum(weights) > 0`, else the current uniform `rand.Int(len(pool))`.
  - In `SelectReward`, after loading `g`: `if g.Kind() == "incubator"` → pool = `item.GetByGachaponId(gachaponId)` mapped to `poolItem{ItemId, Quantity, Weight}` (no global, no tier), `tier := ""`; `selectItem(pool)`. Else the existing `selectTier` → `getMergedPool` path.
  - Keep using `crypto/rand` (not `math/rand`) to match the existing roll.
- [ ] **Step 4: Run, confirm PASS.**
- [ ] **Step 5: Commit** `feat(gachapons): weight-aware roll + incubator-kind branch`.

## Task 4: seed the Pigmy Egg incubator machines

**Files:**
- Modify: gachapons seed catalog schema + rows under `services/atlas-gachapons/atlas.com/gachapons/deploy/seed/` (extend the gachapon record with `kind`, and item records with `weight`)
- Source data: the per-egg pools + `eggId` currently in `services/atlas-tenants/configurations/incubator-rewards/*.json` (task-128 seed) and the region→success-NPC map in `design-incubator-pigmy.md` §6.
- Test: seeder catalog parse test if one exists for gachapons; otherwise a round-trip test that the seeded incubator machine is retrievable and rolls.

**Interfaces:**
- Produces: one `incubator`-kind gachapon per eligible egg (`4170000–4170009`, minus `4170008` if still unresolved — see `design-incubator-pigmy.md` §7), `id = "<eggId>"`, `npcIds = [successNpc]`, its reward rows carrying `weight` copied from the incubator-rewards seed.

- [ ] **Step 1: Failing test** — after seeding, `gachapon.GetById("4170005")` returns an `incubator`-kind machine and `reward.SelectReward("4170005")` returns an item from that egg's pool.
- [ ] **Step 2: Run, confirm FAIL.**
- [ ] **Step 3: Implement** — extend the seed catalog format to carry `kind` (gachapon default) + item `weight`; author the incubator machines from the task-128 incubator-rewards rows (preserve item ids, quantities, weights, eggId→machine id). Use the interim region NPCs from `design-incubator-pigmy.md` §6 for `npcIds` (Ludibrium `4170005`, Nautilus `4170009` confirmed; others interim).
- [ ] **Step 4: Run, confirm PASS.**
- [ ] **Step 5: Commit** `feat(gachapons): seed Pigmy Egg incubator machines`.

## Task 5: channel — gachapons-backed incubator processor

**Files:**
- Rewrite: `services/atlas-channel/atlas.com/channel/incubator/requests.go` (GET config → `POST /gachapons/rewards/select?gachaponId=<eggId>`)
- Rewrite: `services/atlas-channel/atlas.com/channel/incubator/processor.go` (`GetRewardsForEgg` → `SelectReward(eggId) (Reward, error)`)
- Delete: `incubator/roll.go` + `roll_test.go` (`PickWeighted`), the config `RewardRestModel`/`FilterByEgg`/`rest.go` reader
- Modify/replace: `incubator/requests_test.go` to cover the new select request
- Test: `incubator/processor_test.go` for `SelectReward` (mock the REST response)

**Interfaces:**
- Produces: `incubator.Processor.SelectReward(eggId uint32) (Reward, error)` where `Reward` exposes `ItemId() uint32` and `Quantity() uint32` (keep the existing `Reward` accessor shape so the handler arm needs no field renames).
- Consumes: gachapons `POST /gachapons/rewards/select?gachaponId=<id>` returning the reward JSON:API resource (`itemId`, `quantity`).

- [ ] **Step 1: Failing test** — `SelectReward(4170005)` issues a POST to the gachapons `rewards/select` URL with `gachaponId=4170005` and maps the response to `Reward{ItemId, Quantity}`. Mock via the rest requests test harness used elsewhere in the channel.
- [ ] **Step 2: Run, confirm FAIL.**
- [ ] **Step 3: Implement** — `requests.go` builds `requests.RootUrl("GACHAPONS") + "gachapons/rewards/select?gachaponId=" + id` as a `PostRequest` (match the gachapons `handleSelectReward` contract); `processor.SelectReward` calls it and returns the mapped `Reward`. Remove the tenants-config path, `PickWeighted`, `FilterByEgg`, `RewardRestModel`.
- [ ] **Step 4: Run, confirm PASS** (`go test ./incubator/...`).
- [ ] **Step 5: Commit** `refactor(channel): source incubator reward from gachapons service`.

## Task 6: channel — rewire the incubator handler arm

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` lines ~272–297 (the roll block only)
- Test: exercise via the existing handler test surface if present; otherwise assert `SelectReward` is the roll source (the processor test in Task 5 covers the roll; this task is the wiring).

**Interfaces:**
- Consumes: `incubator.Processor.SelectReward(eggId)` (Task 5).

- [ ] **Step 1:** Replace the `GetRewardsForEgg` + `PickWeighted` block (272–285) with:
  ```go
  reward, err := incubator.NewProcessor(l, ctx).SelectReward(eggId)
  if err != nil {
      l.WithError(err).Warnf("Character [%d] used incubator on egg [%d]; no reward selected.", s.CharacterId(), eggId)
      announceFailure(eggId)
      return
  }
  ```
  Keep the downstream `inventory.TypeFromItemId(reward.ItemId())` capacity check and the 4-step saga (`consume_sacrifice`/`consume_incubator`/`award_reward`/`announce_result`) exactly as-is — `reward.ItemId()` / `reward.Quantity()` accessors are unchanged.
- [ ] **Step 2: Build** `go build ./...` in atlas-channel — confirm the `rand` import (used only by the deleted `PickWeighted` closure) is removed if now unused, and `isPigmyEgg` gate stays.
- [ ] **Step 3: Run** `go test -race ./...` in atlas-channel — confirm green.
- [ ] **Step 4: Commit** `refactor(channel): incubator handler rolls via gachapons SelectReward`.

## Task 7: tenants — retire the `incubator-rewards` resource

**Files:**
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/rest.go` (remove `IncubatorRewardRestModel`, `ExtractIncubatorReward`, the `incubator-rewards` registration)
- Delete: `services/atlas-tenants/configurations/incubator-rewards/*.json` seed
- Grep-remove any remaining `incubator-rewards` references in atlas-tenants
- Test: adjust `configuration` tests that referenced the removed model

**Interfaces:** removes the `incubator-rewards` configuration resource entirely.

- [ ] **Step 1:** Grep `incubator-rewards` and `IncubatorReward` across `services/atlas-tenants`; enumerate every site.
- [ ] **Step 2:** Remove the model, extractor, registration, and seed JSON; delete/adjust the tests that covered them (a resource that no longer exists needs no test).
- [ ] **Step 3: Run** `go test -race ./...` + `go vet ./...` in atlas-tenants — green.
- [ ] **Step 4: Commit** `refactor(tenants): remove incubator-rewards config (moved to gachapons)`.

## Task 8: atlas-ui — fold incubator pools into the gachapon admin

**Files:**
- Modify: `services/atlas-ui/src/services/api/gachapons.service.ts`, `src/lib/hooks/api/useGachapons.ts`, `src/pages/GachaponsPage.tsx` / `gachapons-columns.tsx` / `GachaponDetailPage.tsx` — surface `kind` + item `weight`; allow creating/editing an `incubator`-kind machine (region label + success NPC + weighted items).
- Delete: `src/pages/TenantsIncubatorRewardsPage.tsx`, `src/pages/tenants-incubator-rewards-form.tsx` (+ its `__tests__`), `src/services/api/incubator-rewards.service.ts` (+ test), `src/lib/hooks/api/useIncubatorRewards.ts` (+ test), `src/lib/schemas/incubator-rewards.schema.ts` (+ test)
- Modify: `src/App.tsx` (drop the `/tenants/:id/incubator-rewards` route + lazy import), `src/components/app-sidebar.tsx` / `src/components/features/tenants/TenantDetailLayout.tsx` (drop the incubator-rewards link)
- Test: update `GachaponsPage`/service/hook tests for the `kind`/`weight` fields (follow the existing gachapons test idiom; `z.number()` not `z.coerce.number()` per the house form pattern).

**Interfaces:** the gachapon admin becomes the single surface for both kinds; `incubator-rewards` UART removed.

- [ ] **Step 1: Failing test** — gachapons service/hook round-trips `kind` and item `weight`; the create form accepts an `incubator`-kind machine. Extend the existing gachapons tests.
- [ ] **Step 2: Run** `npm run test` — confirm the new assertions FAIL.
- [ ] **Step 3: Implement** the field additions + form; delete the incubator-rewards files and their route/links.
- [ ] **Step 4: Run** `npm run build` (tsc -b + vite) and `npm run test` — both green, no new lint errors (`reference_atlas_ui_npm_nvm_and_lint_baseline`: source nvm 22 first).
- [ ] **Step 5: Commit** `refactor(ui): fold incubator rewards into gachapon admin`.

## Task 9: verification, docs, deploy note

**Files:**
- Modify: `services/atlas-channel/docs/kafka.md` / `services/atlas-gachapons` docs if the event/endpoint surface changed (it does not — `gachapon_reward_won` and `rewards/select` are unchanged; note the new incubator consumer of `rewards/select`).
- Modify: `docs/tasks/task-128-item-tag-seal-incubator/deploy-runbook.md` — the incubator now needs `incubator`-kind gachapons seeded in `atlas-gachapons` (not `incubator-rewards` tenant config); channel restart unchanged.
- No STATUS.md / packet-matrix change (the `INCUBATOR_RESULT` codec is untouched).

- [ ] **Step 1:** Full suite — `go test -race ./...`, `go vet ./...`, `go build ./...` in `atlas-gachapons`, `atlas-channel`, `atlas-tenants`; `docker buildx bake atlas-gachapons atlas-channel atlas-tenants` from the worktree root; atlas-ui `npm run build && npm run test`; `tools/redis-key-guard.sh` + `tools/goroutine-guard.sh`.
- [ ] **Step 2:** Update the deploy runbook + service docs.
- [ ] **Step 3: Commit** `docs(task-128): incubator reconciliation verification + runbook`.

---

## Self-review notes

- **Spec coverage:** Task 1–3 = §3.1/§3.2 (kind + weight + roll); Task 4 = §3.4 migration/seed; Task 5–6 = §3.3 (channel rewire, saga kept); Task 7–8 = §3.4 (retire config + UI); Task 9 = Global Constraints verification. Rename (§4) is explicitly out (follow-up).
- **Type consistency:** `Kind()`/`SetKind`/`Weight()`/`SetWeight`/`GetByGachaponId`/`SelectReward(eggId)`/`Reward.ItemId()/Quantity()` are used identically across tasks.
- **Behavior preservation:** classic gachapon path is guarded by "weight 0 → uniform, kind gachapon → tier+global" — Task 3 tests assert it explicitly.
