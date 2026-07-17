# atlas-gachapons → atlas-reward-pools: rename + topic + consumables-unification

**Status:** design. Ships on task-128 / PR #909 (user directed all three deferred items be done in this task). Follows the reconciliation (`design-incubator-gachapon-reconciliation.md`).

**Three pieces (user-confirmed 2026-07-17):**
1. **Service rename** `atlas-gachapons` → `atlas-reward-pools` (module, dir, deploy, DB name, REST prefix, seed catalog, ~14 registration files).
2. **Topic rename** `gachapon_reward_won` → `pool_reward_won`.
3. **Consumables-unification** — route the incubator trigger through `atlas-consumables` like the gachapon coupon, so both draws share one trigger→roll→award→event path.

## Scope boundary (deliberate)

**Renamed:** the deployable/service identity — module `atlas-gachapons`→`atlas-reward-pools`, dir `services/atlas-gachapons`→`services/atlas-reward-pools`, DB name, k8s/deploy/ingress/compose, REST URL prefix `/gachapons`→`/reward-pools`, seed group + catalog dir `deploy/seed/*/gachapons/`→`.../reward-pools/`, Kafka topic (piece 2).

**KEPT (flagged):** the internal `gachapon` domain type, the `gachapons`/`gachapon_items` tables, and the JSON:API resource types (`gachapon`, `gachapon-rewards`). Rationale: a gachapon is one legitimate `kind` of reward pool (`kind ∈ {gachapon, incubator}`); renaming the tables means schema migration on top of the entity's custom `migrateToSurrogatePK` + baseline/restore column-order risk (`bug_baseline_restore_column_order_drift`) for zero functional gain. Extendable later if desired.

## Naming scheme

| thing | old | new |
|---|---|---|
| service / module / dir | `atlas-gachapons` | `atlas-reward-pools` |
| DB name | `atlas-gachapons` | `atlas-reward-pools` |
| REST prefix | `/gachapons` | `/reward-pools` |
| roll endpoint | `POST /gachapons/{id}/rewards/select` | `POST /reward-pools/{id}/rewards/select` |
| seed group / catalog dir | `gachapons` / `deploy/seed/*/gachapons/` | `reward-pools` / `deploy/seed/*/reward-pools/` |
| REST domain env (callers) | `GACHAPONS` / `GACHAPONS_URL` | `REWARD_POOLS` / `REWARD_POOLS_URL` (→ BASE_SERVICE_URL fallback) |
| Kafka topic | `gachapon_reward_won` | `pool_reward_won` |
| topic env | `EVENT_TOPIC_GACHAPON_REWARD_WON` | `EVENT_TOPIC_POOL_REWARD_WON` |
| domain type / tables (KEPT) | `gachapon`, `gachapons`, `gachapon_items` | unchanged |

## Piece 3 — consumables-unification (the real design work)

**Today:** the classic gachapon coupon flows client→channel→(cash dialog)→**atlas-consumables** which rolls (calls reward-pools `rewards/select`), awards, and emits `gachapon_reward_won`; the channel consumer broadcasts a chat line. The incubator, by contrast, is handled entirely in the **channel** (`character_cash_item_use.go` → `IncubatorUse` saga: consume egg + incubator, award, emit `INCUBATOR_RESULT`).

**Unified:** move the incubator trigger into `atlas-consumables` alongside the gachapon-coupon path. Consumables consumes the trigger (coupon, or incubator+egg), calls the pool roll keyed by pool id (gachapon id / egg id), awards, and emits **one** `pool_reward_won` event carrying a **presentation discriminator** (`kind`: `gachapon` | `incubator`, + `poolId`/`eggId`). The **channel consumer branches on `kind`**: `gachapon` → chat broadcast (as today); `incubator` → the version-gated `INCUBATOR_RESULT` (with `gachaponItemID = eggId`, as today). The `INCUBATOR_RESULT` codec and its version gating are UNCHANGED — only *where the event originates* moves (channel-inline saga → consumables producer + channel consumer branch).

**Consequence:** the channel's inline `IncubatorUse` saga + the `incubator` REST client (built in the reconciliation) are removed; the incubator eligibility gate (`isPigmyEgg`) + egg-target resolution move into consumables. `atlas-saga-orchestrator`'s `IncubatorResult` step and the `EVENT_TOPIC_INCUBATOR_RESULT` path are re-evaluated (the incubator result now rides `pool_reward_won` + the channel branch, so the separate incubator saga type may be retired — confirm during implementation, do not assume).

## Execution order (each an SDD task, verified + reviewed)

1. **Service/module rename** — `git mv` the dir, `module atlas-reward-pools`, rewrite the 18 internal import paths, go.work, docker-bake.hcl, services.json. Build the renamed module.
2. **Deploy/registration rename** — k8s base (`atlas-gachapons.yaml`→`atlas-reward-pools.yaml`) + both overlays, `db-bootstrap.sh` (DB name; and the ephemeral DROP list — `bug_ephemeral_db_teardown_leak_superuser`), `routes.conf`(+`.generated`), ingress, compose. Run `service-registration-guard.sh`.
3. **REST prefix + callers** — service resource registrations `/gachapons`→`/reward-pools`; callers: channel (`RootUrl` domain), saga-orchestrator gachapon client, atlas-ui `/api/gachapons`→`/api/reward-pools`; seed group + catalog dir rename (`git mv` the 7 version dirs' `gachapons/`→`reward-pools/`). Route regex in `routes.conf`.
4. **Topic rename** — `gachapon_reward_won`→`pool_reward_won` across channel consumer + saga producer + consumables producer + the config templates (`EVENT_TOPIC_GACHAPON_REWARD_WON`→`EVENT_TOPIC_POOL_REWARD_WON` in every `template_*.json`), + live-config runbook note.
5. **Consumables-unification** — teach consumables the incubator trigger (item `5060002` + egg target), emit unified `pool_reward_won{kind,poolId/eggId,...}`; channel consumer branches kind→chat vs `INCUBATOR_RESULT`; remove the channel-inline incubator saga + REST client; re-evaluate the saga-orchestrator `IncubatorResult` step.
6. **Verify** — `go build/vet/test-race` all changed modules; **`docker buildx bake` is now MANDATORY** (go.mod changed: renamed module + consumables + channel + saga); `service-registration-guard.sh`; redis/goroutine guards; atlas-ui build+test; matrix `--check` (INCUBATOR_RESULT codec untouched → should stay green).

## Risks

- **Topic cutover** (piece 2/4): producers (consumables/saga) + consumer (channel) + live tenant config must switch atomically. Old `gachapon_reward_won` events in flight during a rolling deploy are dropped — acceptable for dev/PR; note in the runbook.
- **DB rename** (piece 2): a fresh `atlas-reward-pools` DB is created on deploy; the old `atlas-gachapons` data is abandoned → re-seed the catalog. Fine for dev/PR (no prod data).
- **Unification** (piece 5) reworks the incubator flow just built in the reconciliation — the `INCUBATOR_RESULT` wire output must be byte-identical before/after (channel branch produces the same packet the saga did).
