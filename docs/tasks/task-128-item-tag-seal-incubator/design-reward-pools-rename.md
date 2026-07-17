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
| Kafka topic | `gachapon_reward_won` | `pool_reward_won` |
| topic env | `EVENT_TOPIC_GACHAPON_REWARD_WON` | `EVENT_TOPIC_POOL_REWARD_WON` |
| REST prefix / roll endpoint (KEPT) | `/gachapons`, `POST /gachapons/{id}/rewards/select` | unchanged |
| seed group / catalog dir (KEPT) | `gachapons`, `deploy/seed/*/gachapons/` | unchanged |
| REST domain env (callers, KEPT) | `GACHAPONS` / `GACHAPONS_URL` | unchanged |
| domain type / tables / JSON:API resource types (KEPT) | `gachapon`, `gachapons`, `gachapon_items`, `gachapon`/`gachapon-rewards` | unchanged |

**Coherence note (revised 2026-07-17):** the REST URL prefix, seed catalog dir/group,
REST domain env, and JSON:API resource types are KEPT as `gachapon(s)` — they name the
**resource** (`gachapon`, one kind of pool), which the boundary keeps, not the **service**
(`atlas-reward-pools`). Renaming the URL to `/reward-pools` while the resource type stays
`gachapon` would be an incoherent half-measure; renaming the resource type too is the
deferred deep-domain rename. nginx already routes `/api/gachapons` → the `atlas-reward-pools`
host (Task 2), so no caller/seed-dir churn is needed. Only the **service identity** (module,
dir, DB, deploy, host) + **topic** + **unification** change.

## Piece 3 — unification (CORRECTED 2026-07-17: saga-orchestrator, NOT consumables)

**Correction:** an earlier version of this doc said the gachapon roll lives in
`atlas-consumables`. That was wrong. The classic gachapon roll+emit is owned by the
**`atlas-saga-orchestrator`**: `handleSelectGachaponReward` (`saga/handler.go:2635`) calls
`h.gachaponP.SelectReward(gachaponId)` (the reward-pools REST client), dynamically appends an
`AwardAsset` step for the won item, then `handleEmitGachaponWin` (`:2732`) emits
`gachapon_reward_won` (reading the assetId from the completed AwardAsset step). Consumables'
topic is `EVENT_TOPIC_CONSUMABLE_STATUS`, unrelated. So "unify through consumables" is a
mischaracterization — the correct unification is through the **saga-orchestrator**.

**Today's incubator (post-reconciliation):** the **channel** rolls inline
(`incubator.SelectReward(eggId)`), then builds an `IncubatorUse` saga with a *pre-rolled*
reward: `DestroyAsset`(egg) + `DestroyAsset`(incubator) + `AwardAsset`(reward) +
`IncubatorResult`(emit `INCUBATOR_RESULT`). The roll is channel-side; the gachapon roll is
saga-side. THAT asymmetry is what "unify" should remove.

**Accurate unification:** generalize the saga's `SelectGachaponReward` step into a pool-generic
`SelectPoolReward` (keyed by poolId = gachaponId or eggId, via the renamed reward-pools client),
and route the incubator through it: the channel stops rolling inline and instead creates a saga
with `DestroyAsset`(egg) + `DestroyAsset`(incubator) + `SelectPoolReward`(eggId) [saga rolls +
dynamic AwardAsset] + a result-emit step branched by pool `kind` — `gachapon` → the win event
(renamed `pool_reward_won`), `incubator` → the version-gated `INCUBATOR_RESULT`
(`gachaponItemID = eggId`, UNCHANGED wire output). The channel's inline `incubator.SelectReward`
REST client (built in the reconciliation) is removed; the `INCUBATOR_RESULT` codec is untouched.

**Risk (elevated vs the earlier framing):** this refactors the WORKING classic-gachapon saga
path (`handleSelectGachaponReward`/`handleEmitGachaponWin` + their compensators + dynamic
AwardAsset step) to be pool-generic — so it can regress classic gachapon, not just incubator.
The classic gachapon flow must stay behaviorally identical; the incubator `INCUBATOR_RESULT`
bytes must stay identical before/after.

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
