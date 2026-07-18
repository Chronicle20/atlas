# Reconciling the incubator onto `atlas-gachapons` (+ a service rename)

**Status:** design / proposal. Planning only — no implementation until approved.
**Origin:** follow-up to `design-incubator-pigmy.md`. task-128 shipped the incubator
Pigmy-Egg correction as a *flat tenant-config* reward pool rolled *inline in the
channel*. This document specifies folding that mechanic onto the existing
`atlas-gachapons` DDD service, and proposes renaming the service to reflect that it
then serves more than "gachapon."

**Recommended home:** a **new task (task-172)**, not more commits on PR #909. The
reconciliation supersedes task-128's incubator sub-feature and — with the rename —
touches ~14 hand-maintained registration files across two other services. Bundling
that onto the already-large item-tag/seal/incubator PR would make it un-reviewable.
This doc is the design input for that task.

---

## 1. The discrepancy (why this exists)

The incubator ("Pigmy Egg") and the classic gachapon are the **same mechanic** — a
player consumes a trigger item and is awarded one weighted-random item from a
pool keyed by that trigger. MapleStory literally treats the incubator as the
*precursor to gachapon* (the client stores both in `m_mGachaponItemInfo`,
`CItemInfo::RegisterGachaponItemInfo` v95 `@0x5bf040`). Yet Atlas implements them
with two different architectures — a discrepancy in **both** the data model **and**
the execution location:

| dimension | classic gachapon | incubator (task-128 as shipped) |
|---|---|---|
| reward-pool storage | `atlas-gachapons` DDD service (GORM: `gachapon`/`item`/`global`) | flat `incubator-rewards` **tenant-config** (`atlas-tenants`) |
| pool key | gachapon id (machine) | egg id (`4170000–4170009`) |
| **who rolls** | the **gachapons service**, server-side (`reward/processor.go:38` `SelectReward`) | **the channel edge**, inline (`character_cash_item_use.go` → `incubator.PickWeighted`) |
| roll model | 2-stage: `selectTier(common/uncommon/rare)` → uniform pick within tier (+ `global` pool) | 1-stage: per-item `weight`, `PickWeighted` |
| trigger driver | `atlas-consumables` (coupon consume) → gachapons REST | channel handler, in-process |
| award + broadcast | consumables emits `gachapon_reward_won` (`processor.go:1134`); channel consumer broadcasts a chat line | saga `IncubatorUse`: `DestroyAsset` + fire-and-forget `IncubatorResult` → `INCUBATOR_RESULT` packet |
| admin UI | (gachapon admin) | `incubator-rewards` tenant-config form |

**Root cause:** the incubator was built (task-128) before its gachapon nature was
recognized, and modeled on the `MtsConfigRestModel` flat-config pattern that was
nearest to hand. This session's per-egg correction (adding `eggId`, `FilterByEgg`,
version-gated `INCUBATOR_RESULT`) made the mechanic *correct* but **entrenched the
flat-config shape** rather than converging it onto gachapon. Rolling loot on the
client-facing channel edge — rather than in an authoritative domain service — is
itself the weaker of the two designs; gachapon does it right.

**Genuinely incubator-specific (must be preserved, not merged away):** the
**clientbound result**. Gachapon awards broadcast a chat line; the incubator renders
a modal Pigmy & Etran NPC dialog via the **version-gated** `INCUBATOR_RESULT`
(flat `itemId+count` on v83/84/87/jms; extended tail carrying `gachaponItemID = eggId`
on v95 — see `design-incubator-pigmy.md` §3). That divergence is real client
behavior and stays. The reconciliation targets the **reward-pool data + the roll**,
not the result packet.

---

## 2. What `atlas-gachapons` already provides (grounded)

`services/atlas-gachapons/atlas.com/gachapons/`:

- **`gachapon`** — `Model{tenantId, id string, name, npcIds []uint32, commonWeight, uncommonWeight, rareWeight}`. A machine: a name, the NPC(s) it's offered at, and tier weights.
- **`item`** — `Model{tenantId, id uint32, gachaponId string, itemId, quantity, tier string}`. A per-machine reward, bucketed into a tier.
- **`global`** — `Model{tenantId, id, itemId, quantity, tier}`. A shared pool merged into every machine's roll for the selected tier.
- **`reward`** — the roll. `SelectReward(gachaponId)` (`processor.go:38`): `selectTier(common,uncommon,rare)` (crypto/rand weighted) → `getMergedPool(gachaponId, tier)` = machine items + global items for that tier → `selectItem(pool)`. Exposed as **`POST /gachapons/rewards/select?gachaponId=…`** and `GET /gachapons/prize-pool` (`reward/resource.go:23-24`).
- **Seeding** — `seed/groups.go`: filesystem catalog (`NewFilesystemCatalogSource("SEED_CATALOG_ROOT","./deploy/seed")`), group `"gachapons"`, URLPrefix `/gachapons`.
- **Event** — `gachapon_reward_won` is emitted by **`atlas-consumables`**, not by the gachapons service; the gachapons service is a pure reward-pool + roll authority (REST). The channel only consumes the event to broadcast.

Every capability the incubator needs — per-key weighted pools, a shared/global pool,
a server-side roll endpoint, a seeded catalog, an admin surface — already exists here.

---

## 3. Reconciliation design

### 3.1 Model each Pigmy Egg as a gachapon

One gachapon row per egg, `id = the egg item id as string` (e.g. `"4170005"`).
`npcIds` = that region's Pigmy & Etran success NPC (authoritatively from
`incubatorInfo.img/{eggId}/su`; interim from the region map in
`design-incubator-pigmy.md` §6). `name` = the region label. The egg's reward pool
becomes `item` rows keyed by that gachapon id.

**Add a `kind` discriminator** to the `gachapon` model: `"gachapon" | "incubator"`.
Two mechanics now share the roll engine but differ in trigger and result packet;
`kind` lets the admin UI filter, lets seeding partition, and lets eligibility be
scoped without an id-range heuristic. (Range `4170000–4170009` is a *client* gate,
`CUIIncubator::PutItem`; the server should key off "is there an `incubator`-kind
gachapon with this id," per `design-incubator-pigmy.md` §7's "key off the set that
exists, not a hard range" rule.)

### 3.2 Reconcile the weighting model — the one real modeling decision

Gachapon rolls **2-stage** (tier, then uniform within tier); the incubator config
rolls **1-stage per-item weight**. These must converge. Options:

- **(A) Add an optional per-item `weight` to the `item` model.** When set, the roll
  is single-stage weighted over the pool (incubator semantics); when unset/zero,
  the existing tier roll applies (gachapon semantics unchanged). One `reward`
  processor serves both. **Recommended** — additive, backward-compatible, no data
  migration for existing gachapons, and it makes the service genuinely general.
- **(B) Bucket incubator rewards into the three tiers.** No schema change, but it
  *loses* the incubator's fine-grained per-item weights (task-128 seeded explicit
  weights) — a behavior regression. Rejected unless the weights are known to be
  tier-collapsible.

Option A is the crux the plan phase must lock. It generalizes the roll from
"tiered gacha" to "weighted-pool draw," which is exactly what justifies the rename
(§4).

### 3.3 Rewire the roll: channel-inline → gachapons REST

Replace the channel's inline `incubator.{rest,roll,requests}.go` +
`GetRewardsForEgg`/`PickWeighted` with a call to the gachapons roll endpoint keyed
by egg id: **`POST /gachapons/rewards/select?gachaponId=<eggId>`** → `{itemId,
quantity}`. Eligibility = the target is an `incubator`-kind gachapon that exists.
This moves the roll off the client-facing edge into the authoritative service —
matching how gachapon already works, and closing the DOM concern of rolling loot in
the channel.

The `IncubatorUse` saga is **kept**: `DestroyAsset` (consume egg + incubator) and the
fire-and-forget `IncubatorResult` step still emit the version-gated `INCUBATOR_RESULT`
with `gachaponItemID = eggId`. Only the *source of the awarded item* changes (service
roll instead of inline roll). Whether the roll is invoked by the channel before saga
creation, or promoted into a saga step / into `atlas-consumables` alongside the
gachapon-coupon path, is a plan-phase decision — see §6 open question.

### 3.4 Retire the flat `incubator-rewards` config

Once pools live in gachapons, remove:
- `atlas-tenants` `IncubatorRewardRestModel` + `ExtractIncubatorReward` + the
  `incubator-rewards` resource registration (`configuration/rest.go`) and its seed JSON.
- channel `incubator/{rest,roll,requests}.go` (the config reader + `PickWeighted`).
- atlas-ui `incubator-rewards.service.ts`, its schema, hooks
  (`useIncubatorRewards`), tests, and the `tenants-incubator-rewards-form` page —
  **folded into the gachapon admin** filtered to `kind=incubator` (region label +
  success NPC per pool, as `design-incubator-pigmy.md` §5 intended).

Migration: seed the per-egg pools task-128 created as `incubator`-kind gachapons in
the gachapons seed catalog (`deploy/seed`), so existing tenants converge on
service-provisioning. No live tenant currently depends on `incubator-rewards` beyond
task-128's own seed, so this is a seed-catalog move, not a data backfill.

---

## 4. Service rename proposal

With §3.2 done, the service is no longer gachapon-specific: it is a **weighted
reward-pool authority** keyed by an arbitrary trigger item, serving gachapon *and*
incubator today and any future draw/loot-pool mechanic (event boxes, mystery bags,
etc.). The name should say that.

### Candidates

| name | read | note |
|---|---|---|
| **`atlas-reward-pools`** (recommended) | "the service that owns weighted reward pools + the draw" | most literal to what it does; `pool`/`draw` already the domain vocabulary (`getMergedPool`, `prize-pool`) |
| `atlas-gacha` | concise; "gacha" = the draw-family (gachapon + incubator are both gacha) | shorter, but reads as jargon and is close enough to "gachapon" to blur the point of renaming |
| `atlas-prizes` / `atlas-loot` | generic reward/loot service | fine, but less precise than "reward-pools"; `loot` collides conceptually with monster drops (a different subsystem) |

Recommendation: **`atlas-reward-pools`**. Naming is the user's call — this doc
proposes; the plan phase commits one name.

### Blast radius (this is why it's a separate task)

A service rename is the "adding-a-new-service" checklist in reverse — every
hand-maintained registration list, verified present today by grep:

- `.github/config/services.json` (single source of truth)
- `docker-bake.hcl` (`go_services`, hand-synced — memory `reference_docker_bake_hand_synced`)
- `go.work`
- `tools/db-bootstrap.sh` (DB name; and the ephemeral-env DROP list — memory `bug_ephemeral_db_teardown_leak_superuser`)
- `deploy/k8s/base/atlas-gachapons.yaml` + `base/kustomization.yaml` + `base/routes.conf.template.generated`
- `deploy/k8s/overlays/{main,pr}/kustomization.yaml` + their `db-name-suffix.yaml` patches + `overlays/main/patches/atlas-env-env.yaml`
- `deploy/shared/routes.conf`
- `deploy/compose/docker-compose.core.yml`
- the Go module path (`services/atlas-gachapons/atlas.com/gachapons` → new) and every import of it
- REST resource URLPrefix (`/gachapons`) + seed group name (`"gachapons"`) — and the ingress route that fronts it
- **Kafka topic + DB**: `gachapon_reward_won` and the DB name. These carry the *classic* mechanic's identity. **Recommendation: do NOT rename the topic or DB in the same stroke** — renaming a live Kafka topic is a data-plane migration (consumers in atlas-channel, producers in atlas-consumables/saga) with real risk (memory: new/renamed topics silently drop if not in live tenant config). Rename the *service and module*; leave `gachapon_reward_won` as a stable event name (it describes the classic sub-mechanic) and the DB name as-is, or migrate them in a separate follow-up with a dual-read window.

`tools/service-registration-guard.sh` machine-checks most of these lists and is a CI
gate — run it (and the doc's Verification section) as the rename's acceptance test.

---

## 5. Non-goals

- **Not** merging the `INCUBATOR_RESULT` packet into the gachapon broadcast — the
  version-gated NPC-dialog result is real, distinct client behavior (§1).
- **Not** renaming the `gachapon_reward_won` Kafka topic or the DB in the same task
  (§4 blast-radius note) — service/module rename only; topic/DB migration is a
  separately-sequenced follow-up if desired.
- **Not** ingesting `incubatorInfo.img` — orthogonal (that's `design-incubator-pigmy.md`
  Phase 7, for authoritative region labels + eligible-egg set + `4170008`). The
  reconciliation works with tenant-seeded pools regardless.

---

## 6. Resolved decisions (2026-07-16)

1. **Rename target** — **`atlas-reward-pools`**. (Deferred to a follow-up PR, per #4.)
2. **Weighting model** — **Option A**: add an optional per-item `weight` to the
   `item` model. Set → single-stage weighted (incubator); unset/zero → existing tier
   roll (gachapon). Additive, no migration for existing gachapons. §3.2.
3. **Roll invocation site** — **keep the channel as the roll caller**. It already owns
   the `IncubatorUse` saga, so the change is a single `POST /gachapons/rewards/select`
   call plus deletion of the inline roll — behavior-preserving and contained.
   Promoting the incubator into `atlas-consumables` (unifying both draw paths) is
   deferred to the rename follow-up, where unifying the paths is the point. §3.3.
4. **Task carving** — **reconcile now on task-128** (this branch / PR #909), as an
   addendum to the task's purpose: data + roll reconciliation (§3); no rename;
   `gachapon_reward_won` topic and DB unchanged. The **service/module rename to
   `atlas-reward-pools`** (§4) — plus the optional consumables-unification (#3) — is a
   **separate follow-up PR** so the ~14-file registration churn stays out of the
   behavior change.

---

## 7. Scope on task-128 (this PR) vs the rename follow-up

**On task-128 now (behavior-preserving reconciliation):**
- `atlas-gachapons`: add `kind` to `gachapon`, optional `weight` to `item`; the roll
  honors `weight` when present (§3.1, §3.2).
- Seed the per-egg pools as `incubator`-kind gachapons (§3.4 migration).
- `atlas-channel`: replace the inline incubator roll with a `POST /gachapons/rewards/select`
  call by egg id; keep the `IncubatorUse` saga + version-gated `INCUBATOR_RESULT` (§3.3).
- Retire the flat `incubator-rewards` tenant-config resource + fold its UI into the
  gachapon admin filtered to `kind=incubator` (§3.4).

**Deferred to the rename follow-up PR:**
- Rename the service/module `atlas-gachapons` → `atlas-reward-pools` across the ~14
  registration files (§4 blast radius).
- Optionally unify the incubator trigger into `atlas-consumables` (§3.3 alternative).
- Optionally migrate the `gachapon_reward_won` topic / DB names (§5 non-goal today).

The PRD addendum (`prd.md` §11) records this scope addition to task-128.
