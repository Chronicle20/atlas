# Context — Incubator → gachapons reconciliation

Companion to `plan-incubator-gachapon-reconciliation.md` (spec:
`design-incubator-gachapon-reconciliation.md`, PRD addendum: `prd.md` §11). Extends
task-128; item-tag and sealing-lock are untouched. Rename to `atlas-reward-pools` is a
**separate follow-up PR**, not this work.

## Key files (current state)

| File | Role | Change |
|---|---|---|
| `services/atlas-gachapons/.../gachapon/{entity,model,builder,provider,resource}.go` | machine: `{id,name,npcIds,common/uncommon/rareWeight}`, surrogate-uuid PK | Task 1: add `kind` (`gachapon`\|`incubator`, default `gachapon`) |
| `services/atlas-gachapons/.../item/{entity,model,builder,provider,resource}.go` | per-machine reward `{gachaponId,itemId,quantity,tier}` | Task 2: add optional `weight`; add `GetByGachaponId` (all tiers) |
| `services/atlas-gachapons/.../reward/processor.go` | roll: `SelectReward`→`selectTier`→`getMergedPool`(machine+global)→`selectItem` (uniform) | Task 3: `poolItem.Weight`; weighted `selectItem`; `incubator`-kind branch (machine-only, tier-agnostic, no global) |
| `services/atlas-gachapons/.../deploy/seed/` | filesystem catalog (group `gachapons`, URLPrefix `/gachapons`) | Task 4: seed 9 Pigmy-Egg `incubator` machines from the task-128 incubator-rewards rows |
| `services/atlas-channel/.../incubator/{requests,processor,roll,rest}.go` | reads `tenants/.../configurations/incubator-rewards`, `PickWeighted` inline | Task 5: `requests`→gachapons `POST /gachapons/rewards/select?gachaponId=`; `processor.SelectReward(eggId)`; delete `roll.go`/`rest.go`/`PickWeighted`/`FilterByEgg` |
| `services/atlas-channel/.../socket/handler/character_cash_item_use.go` (~252–363) | incubator arm: inline roll → 4-step `IncubatorUse` saga | Task 6: swap 272–285 for `SelectReward`; saga + `INCUBATOR_RESULT` unchanged |
| `services/atlas-tenants/.../configuration/rest.go` + `services/atlas-tenants/configurations/incubator-rewards/*.json` | `IncubatorRewardRestModel` + resource + seed | Task 7: remove the resource entirely |
| `services/atlas-ui/src/{pages,services/api,lib/hooks/api,lib/schemas}/…incubator-rewards…` + `GachaponsPage`/`gachapons.service`/`useGachapons` | flat incubator-rewards admin + existing gachapon admin | Task 8: surface `kind`/`weight` in gachapon admin; delete incubator-rewards files + route/links |

## Decisions (spec §6, resolved 2026-07-16)

1. Rename → `atlas-reward-pools` — **deferred to follow-up PR** (not this task).
2. Weighting → **Option A**: optional per-item `weight`; set → weighted, unset → tier roll.
3. Roll site → **channel calls gachapons `rewards/select`**; keep the `IncubatorUse` saga. Consumables-unification deferred to the rename PR.
4. Carving → reconcile now on task-128 / PR #909; topic `gachapon_reward_won` + DB unchanged.

## Key facts (verified)

- `atlas-gachapons` is REST-only: `POST /gachapons/rewards/select?gachaponId=` (`reward/resource.go:23`) rolls and returns `{itemId,quantity,tier}`; it does NOT grant. `gachapon_reward_won` is emitted by `atlas-consumables` (`consumable/processor.go:1134`), consumed by the channel for broadcast — the gachapons service never emits it.
- The channel incubator arm already grants via the saga `award_reward` (`AwardAsset`) step and announces via `IncubatorResult` (`gachaponItemID = eggId`). Only the *reward source* changes.
- gachapon PK is a surrogate uuid with `(tenant_id, id)` unique (`gachapon/entity.go` `migrateToSurrogatePK`) — machine `id` is a slug/string, so `id = "<eggId>"` is valid.
- atlas-ui already has a gachapon admin (`GachaponsPage`, `GachaponDetailPage`, `gachapons.service.ts`, `useGachapons.ts`) — the fold-in target exists.

## Dependencies / sequencing

Task 1,2 (gachapons fields) → Task 3 (roll) → Task 4 (seed). Task 5 (channel processor) → Task 6 (handler). Task 7 (tenants) + Task 8 (ui) independent of 1–6 but land after so nothing references the removed config mid-stream. Task 9 = whole-suite verification. `go.mod` touched in `atlas-gachapons`, `atlas-channel`, `atlas-tenants` → `docker buildx bake` those three. Lands on `task-128-item-tag-seal-incubator` / PR #909.
