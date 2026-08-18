# Backend Services Audit — Service-Wiring Recipe (task-232)

- **Scope:** All changed Go files under `services/` EXCEPT `atlas-channel`,
  `atlas-login`, `atlas-configurations`, `atlas-tenants`,
  `atlas-saga-orchestrator`, `atlas-query-aggregator` (owned by a parallel
  shard), and `libs/`/`tools/` (owned by a parallel shard).
- **Diff range:** `c8d44127cbb9eb2016c621463f86614b81c618e7` ..
  `418b2caf97da2f1c326cafaadca9218456d63daf`
- **Guideline source:** `docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md`
  (the canonical recipe, established on `atlas-monsters`, applied mechanically
  in Task 33–40 batches to the rest of the fleet).
- **Method:** 58 services in this shard were fanned out across 8 sonnet
  sub-agents (one per Task-33..40 batch, minus the services owned by the other
  shard), each running the recipe's mechanical checks — `service.Bootstrap`
  wiring, `SetHeaderParsers` ordering, `requests.RootUrl(` → `RootUrlFor`
  conversion (Patterns A/B/C/D/E), `wiring_test.go` presence, ctx-threading at
  every converted call site, and the "no domain package touched" rule (§4f) —
  with file:line citations. `atlas-monsters` (the reference implementation)
  was re-verified as part of batch 1.

## Overall verdict: NEEDS-WORK

Every one of the 58 services' **mechanical wiring** (the three canonical
points: `Bootstrap`, `SetHeaderParsers`, `RootUrl→RootUrlFor`) is correct —
zero misses, zero ordering violations, zero leftover `requests.RootUrl(`
sites, zero missing `wiring_test.go`, zero dropped `ctx` arguments. That part
of the fleet-wide rollout is clean.

Two services went beyond the recipe's three wiring points into real,
untested-by-the-mechanical-checks domain rework that the recipe's own §4f
rule says should have stopped for a decision instead of proceeding inline.
These are the audit's substantive findings.

---

## Blocking finding 1 — atlas-data: recipe §4f violation (domain rework, not wiring)

**Files:** `services/atlas-data/atlas.com/data/ingestrun/ingestrun.go`,
`runtime/ingest/heartbeat.go`, `runtime/ingest/progress.go`,
`runtime/rest/jobs.go`, `runtime/rest/resource.go`, `runtime/rest/watchdog.go`,
plus a new `atlas-env` direct dependency added to `go.mod`.

These files were rewritten to swap `redis.Registry[K,V]` for
`redis.EnvironmentRegistry[K,V]`, threading `env.Self()` through every
`Get`/`Put`/`Remove`/`UpdateWithTTL` call in the ingest job/run-tracking
control plane. This is a real type and behavior change to the control-plane
registry layer, not "Bootstrap / SetHeaderParsers / RootUrl→RootUrlFor plus
trivial ctx threading of an existing call." `runtime/rest/resource.go` in
particular is a handler file — the recipe's §4f list names
`handler/resource.go` explicitly as out of scope for a wiring batch.

None of the mechanical wiring-recipe greps catch this: `service.Bootstrap`,
`SetHeaderParsers`, and `requests.RootUrl(` are all clean in atlas-data (it
has zero outbound REST clients, so the third check is vacuously satisfied).
The Step-5 "re-run the counted-site greps" verification in the recipe
therefore reports this batch as complete while a real, unreviewed
registry-semantics change rode along in the same commit.

**Recommendation:** this rework may well be correct and even necessary for
the environment-scoped Redis migration tracked elsewhere in this task's plan
(Task 8/9), but it does not belong in a "service-wiring recipe" batch commit
per the recipe's own stated scope. Split it into its own reviewed change, or
explicitly document in the recipe/plan that atlas-data's batch is an
intentional exception to §4f with a citation to the task that authorizes it.

## Blocking finding 2 — atlas-doors: recipe §4f violation (domain rework, not wiring)

**File:** `services/atlas-doors/atlas.com/doors/door/registry.go`

This file was converted from a manually tenant-prefixed
`atlasredis.Registry`/`KeyedSet` to `atlasredis.TenantRegistry`/
`TenantKeyedSet`, with key-suffix helper functions losing their tenant-id
parameter (`storeSuffix(t, id)` → `storeSuffix(id)`), and `GetAll`/`Clear`
replaced by `GetAllAcrossTenants`/`ClearAllAcrossTenants`. This is a real
domain-registry logic and API-shape change (a "D7" cross-tenant discipline
refactor), not a wiring-recipe edit, and again invisible to the recipe's
own Step-5 grep-based verification since it touches none of `Bootstrap`,
`SetHeaderParsers`, or `RootUrl(`.

**Recommendation:** same as atlas-data — separate this from the wiring-recipe
commit, or record an explicit, plan-cited exception.

Note: `services/atlas-doors/atlas.com/doors/door/expiry_task.go`'s
`envContext` parameter addition in the same diff is **not** part of this
finding — that is the legitimate Step-1 `tenant.WithContext` non-test
origination-site handling described below, confined to a signature/DI change
with no registry-semantics rework.

---

## Non-blocking observation — widespread `tenant.WithContext` origination-site DI, correctly scoped

A large number of services in this shard carry a small, consistent extra
pattern beyond the recipe's literal three wiring points: a background
sweep/ticker function (which builds `tenant.WithContext(ctx, t)` with no
inbound request to carry an `ENVIRONMENT` header) gains an
`envContext func(context.Context) context.Context` parameter, injected from
`main.go` via `env.WithContext(ctx, env.Self())`, applied immediately before
the existing `tenant.WithContext` call. This is the recipe Step 1's mandated
`tenant.WithContext` non-test audit/classification obligation, done
correctly and mechanically everywhere it was found — confined to
constructor/method signatures and the origination call site, with no
business-logic (`Hit`/`Trigger`/`Create`/`Destroy`/domain-rule) changes, and
each pinned by a new test. This is **not** a deviation, but is called out
because the recipe document as currently written references a "Step 3b"
classification step in its Step-1 prose that has no corresponding numbered
section in `service-wiring-recipe.md` itself — a documentation gap, not an
implementation defect. Recommend adding the missing Step 3b section so
future batches (and this audit) have a citable rule rather than inferring
the correct scope from precedent.

Services carrying this pattern, all judged correctly scoped:
`atlas-character` (`session/task.go`), `atlas-drops` (`drop/task.go`),
`atlas-reactors` (`reactor/processor.go`, `reactor/mock/processor.go`),
`atlas-rps` (`game/task.go`), `atlas-skills` (`skill/processor.go`,
`tasks/expiration.go`), `atlas-summons` (`summon/beholder_task.go`,
`summon/expiry_task.go`), `atlas-trades` (`trade/settlement.go`),
`atlas-world` (`channel/processor.go`, `broadcast/task.go`), `atlas-doors`
(`door/expiry_task.go`).

Two comment-only, no-op touches were also observed and confirmed harmless:
`atlas-kites/kite/registry.go` (comment + gofmt) and
`atlas-messengers/messenger/processor.go` (comment only).

`atlas-maps` and `atlas-merchant` additionally carry per-tenant ticker
`envContext` wiring in `tasks/*.go` files plus (atlas-maps only) a
`KeyedHash`→`TenantKeyedHash` Redis-key conversion in
`map/monster/registry.go` — same class of work as the "widespread" list
above, co-mingled with an otherwise-clean wiring commit. Not re-flagged as a
third §4f violation because, unlike atlas-data/atlas-doors, no sub-agent
identified a matching test/behavior gap in these — but worth noting the
scope crept in the same direction and a stricter read of §4f could class
`atlas-maps`' registry conversion the same as atlas-doors'. Recorded here for
the record; not double-counted as a blocking finding.

**atlas-transports** deliberately did **not** get this treatment at its three
`tenant.WithContext` sites in `main.go` (lines ~104, 120, 151) — confirmed by
the commit message for `418b2caf9`, which states the Bootstrap wiring was
added but `env.WithContext` was deliberately left out at those sites, because
that conversion is Task 41/42 (`ForEachOwnedEnvironment` ticker rework),
explicitly out of scope for the wiring-recipe batches. `grep -rn
"ForEachOwnedEnvironment"` returns nothing anywhere in this shard's
services, confirming Task 41/42 has not landed in this diff range — this is
expected and not evaluated further per the audit's scope instructions.

---

## Per-service mechanical checklist results (all 58 services + atlas-monsters reference)

Legend: B = `service.Bootstrap` wiring, H = `SetHeaderParsers` ordering,
R = `requests.RootUrl(` zero-hit / conversion, W = `wiring_test.go`,
S = domain-package scope (§4f), C = ctx threading at converted call sites.

| Service | B | H | R | W | S | C | Notes |
|---|---|---|---|---|---|---|---|
| atlas-monsters (reference) | PASS `main.go:57` | PASS (5 sites) | PASS | PASS | PASS | PASS | Re-verified clean |
| atlas-account | PASS `main.go:49` | PASS | PASS | PASS | PASS | PASS | |
| atlas-asset-expiration | PASS `main.go:23` | PASS | PASS (Pattern C) | PASS | PASS | PASS | |
| atlas-ban | PASS `main.go:52` | PASS | PASS | PASS | PASS | PASS | |
| atlas-buddies | PASS `main.go:49` | PASS | PASS | PASS | PASS | PASS | |
| atlas-buffs | PASS `main.go:48` | PASS | PASS | PASS | PASS | PASS | |
| atlas-cashshop | PASS `main.go:59` | PASS | PASS | PASS | PASS | PASS | |
| atlas-chairs | PASS `main.go:44` | PASS | PASS | PASS | PASS | PASS | |
| atlas-chalkboards | PASS `main.go:44` | PASS | PASS (no outbound client) | PASS | PASS | N/A | |
| atlas-character | PASS `main.go:66` (`lifecycle` alias) | PASS (7 sites) | PASS (incl. Pattern D `location/requests.go`) | PASS (adapted for alias) | PASS + Step-1 origination in `session/task.go` | PASS | Alias documented in wiring_test.go comment |
| atlas-character-factory | PASS `main.go:49-50` | PASS | PASS incl. Pattern E `name_validity_requests.go` (both `RootUrlFor` conversion AND `EnvHeaderDecorator` addition present, §4e) | PASS | PASS | PASS | |
| atlas-consumables | PASS `main.go:30` | PASS (7 sites) | PASS (13 files) | PASS | PASS | PASS | |
| atlas-data | PASS `main.go:76` | PASS | PASS (no outbound client) | PASS | **FAIL §4f** — see Blocking finding 1 | N/A | |
| atlas-doors | PASS `main.go:55` | PASS (3 sites) | PASS incl. 2x Pattern D | PASS | **FAIL §4f** — see Blocking finding 2 | PASS | |
| atlas-drop-information | PASS `main.go:43` | N/A (no consumers) | PASS | PASS | PASS | N/A | Smallest possible diff |
| atlas-drops | PASS `main.go:56` | PASS | PASS | PASS | PASS + Step-1 origination in `drop/task.go` | PASS | |
| atlas-effective-stats | PASS `main.go:45` | PASS (4 sites) | PASS (6 files) | PASS | PASS | PASS | |
| atlas-expressions | PASS `main.go:30` | PASS (2 sites) | PASS (no outbound client) | PASS | PASS + Step-1 origination `expression/task.go` | N/A | |
| atlas-fame | PASS `main.go:26` | PASS (2 sites) | PASS | PASS | PASS | PASS | |
| atlas-families | PASS `main.go:44` | PASS (2 sites) | PASS (no outbound client) | PASS | PASS | N/A | |
| atlas-guilds | PASS `main.go:61` | PASS (4 sites) | PASS | PASS | PASS + Step-1 origination `guild/task.go`, `coordinator/registry.go` | PASS | |
| atlas-inventory | PASS `main.go:50` | PASS (3 sites) | PASS (8 files) | PASS | PASS + tenant-scoped lock keying (`compartment/lock_registry.go`) — out of scope, not re-evaluated | PASS | |
| atlas-invites | PASS `main.go:51` | PASS (2 sites) | PASS (no outbound client) | PASS | PASS + Step-1 origination `invite/task.go` | N/A | |
| atlas-keys | PASS `main.go:43` | PASS (1 site) | PASS (no outbound client) | PASS | PASS | N/A | |
| atlas-kites | PASS `main.go:44` | PASS (2 sites) | PASS | PASS | PASS (comment-only extra touch) | PASS | |
| atlas-map-actions | PASS `main.go:46` | PASS (2 sites) | PASS | PASS | PASS | PASS | |
| atlas-maps | PASS `main.go:66` | PASS (7 sites) | PASS incl. Pattern D `character/requests.go` | PASS | PASS on wiring; co-mingled `tasks/*.go` envContext + `TenantKeyedHash` registry conversion — see observation above | PASS | |
| atlas-marriages | PASS `main.go:44` | PASS | PASS | PASS | PASS | PASS | |
| atlas-merchant | PASS `main.go:57` | PASS (3 sites) | PASS | PASS | PASS on wiring; co-mingled `frederick/notification_task.go`, `shop/task.go` envContext — see observation above | PASS | |
| atlas-messages | PASS `main.go:63` | PASS (1 site) | PASS (11 files) | PASS | PASS | PASS | |
| atlas-messengers | PASS `main.go:45` | PASS (3 sites) | PASS | PASS | PASS (comment-only extra touch) | PASS | |
| atlas-mini-games | PASS `main.go:47` | PASS (3 sites) | PASS incl. 4x Pattern D | PASS | PASS | PASS | |
| atlas-monster-book | PASS `main.go:49` | PASS (2 sites) | PASS incl. Pattern D `data/consumable/requests.go` | PASS | PASS | PASS | |
| atlas-monster-death | PASS `main.go:20` | PASS (1 site) | PASS (9 files) | PASS | PASS | PASS | |
| atlas-mounts | PASS `main.go:51` | PASS (3 sites) | PASS (no outbound client) | PASS | PASS + Step-1 origination `mount/task.go` | N/A | |
| atlas-mts | PASS `main.go:54` | PASS (2 sites) | PASS (2 files) | PASS | PASS | PASS | |
| atlas-notes | PASS `main.go:46` | PASS (2 sites) | PASS (no outbound client) | PASS | PASS | N/A | |
| atlas-npc-conversations | PASS `main.go:52` | PASS (4 sites) | PASS (8 files) | PASS | PASS | PASS | |
| atlas-npc-shops | PASS `main.go:52` | PASS (2 sites) | PASS (7 files) | PASS | PASS (incl. legitimate dead-wrapper deletion) | PASS | |
| atlas-parties | PASS `main.go:45` | PASS (3 sites) | PASS incl. Pattern D `location/requests.go` | PASS | PASS | PASS | |
| atlas-party-quests | PASS `main.go:55` | PASS (3 sites) | PASS (4 files) | PASS | PASS | PASS | |
| atlas-pets | PASS `main.go:53` | PASS (3 sites) | PASS incl. Pattern D `location/requests.go` | PASS | PASS + Step-1 origination `pet/task.go` | PASS | |
| atlas-portal-actions | PASS `main.go:48` | PASS (2 sites) | PASS | PASS | PASS | PASS | |
| atlas-portals | PASS `main.go:43` | PASS (2 sites) | PASS | PASS | PASS | PASS | |
| atlas-quest | PASS `main.go:49` | PASS (4 sites) | PASS (2 files) | PASS | PASS | PASS | |
| atlas-rankings | PASS `main.go:43` | N/A (no consumers) | PASS (3 files) | PASS | PASS + Step-1 origination `tasks/recompute.go` | PASS | |
| atlas-rates | PASS `main.go:44` | PASS (4 sites) | PASS (5 files) | PASS | PASS | PASS | |
| atlas-reactor-actions | PASS `main.go:44` | PASS (1 site) | PASS (inline in `script/executor.go`, `script/evaluator.go`) | PASS | PASS | PASS | |
| atlas-reactors | PASS `main.go:49` | PASS (2 sites) | PASS | PASS | PASS + Step-1 origination `reactor/processor.go` | PASS | |
| atlas-renders | PASS `main.go:27` | N/A (no consumers) | N/A (no outbound client) | PASS | PASS | N/A | |
| atlas-reward-pools | PASS `main.go:42` | N/A (no consumers) | N/A (no outbound client) | PASS | PASS | N/A | |
| atlas-rps | PASS `main.go:43` | PASS (1 site) | PASS | PASS | PASS + Step-1 origination `game/task.go` | PASS | |
| atlas-skills | PASS `main.go:53` | PASS (3 sites) | N/A (no outbound client) | PASS | PASS + Step-1 origination `skill/processor.go`, `tasks/expiration.go` | N/A | |
| atlas-storage | PASS `main.go:61` (`lifecycle` alias) | PASS (4 sites) | PASS (3 files) | PASS (adapted for alias) | PASS (Task 4 npc-context fix confirmed separate commit) | PASS | |
| atlas-summons | PASS `main.go:69` | PASS (2 sites) | PASS (3 files) | PASS | PASS + Step-1 origination `summon/beholder_task.go`, `summon/expiry_task.go` | PASS | |
| atlas-trades | PASS `main.go:73` | PASS (6 sites) | PASS (7 files) | PASS | PASS + Step-1 origination `trade/settlement.go` | PASS | |
| atlas-transports | PASS `main.go:59` | PASS (5 sites) | PASS (5 files, Pattern C) | PASS | PASS — Task 41/42 deliberately NOT applied at 3 `tenant.WithContext` sites in main.go (confirmed via commit message + absent `ForEachOwnedEnvironment`) | PASS | Out-of-scope note, not a FAIL |
| atlas-world | PASS `main.go:76-83` | PASS (3 sites) | PASS (no outbound client) | PASS | PASS + Step-1 origination `channel/processor.go`, `broadcast/task.go` | N/A | |

## Not evaluable from the diff

- Whether the `TenantKeyedHash`/`envContext` co-mingled work in atlas-maps and
  atlas-merchant (see observation above) is itself correct — sub-agents
  confirmed it exists and is confined to signature/DI changes plus one
  registry-key-type conversion in atlas-maps, but a full correctness review of
  that non-wiring-recipe work (Redis key migration semantics, concurrency
  shape of the ticker loops) was out of this audit's scope per the task
  instructions (Task 41/42 territory) and was not performed.
- Full correctness of the `atlas-inventory` `compartment/lock_registry.go`
  tenant-scoped lock keying change — confirmed to exist, not evaluated
  (same reason).
- Whether atlas-data's and atlas-doors' registry rework (Blocking findings 1
  and 2) is functionally correct — only that it exists and falls outside the
  wiring recipe's declared scope. A correctness review of those specific
  redis-registry/tenant-registry conversions would require reading
  `libs/atlas-redis` (owned by the other shard) and is not performed here.
- Task 41/42 (`ForEachOwnedEnvironment` ticker conversions) fleet-wide status
  — explicitly out of scope per the audit brief; confirmed absent from this
  diff range via `grep -rn "ForEachOwnedEnvironment"` returning zero hits
  across the entire `services/` tree.

## Summary

### Blocking (must fix or explicitly except)
- atlas-data: `ingestrun.go`, `runtime/ingest/{heartbeat,progress}.go`,
  `runtime/rest/{jobs,resource,watchdog}.go` — redis registry rework beyond
  recipe §4f scope; not caught by the recipe's own verification greps.
- atlas-doors: `door/registry.go` — `TenantRegistry`/`TenantKeyedSet`
  conversion beyond recipe §4f scope; not caught by the recipe's own
  verification greps.

### Non-Blocking (should fix)
- `service-wiring-recipe.md` references a "Step 3b" `tenant.WithContext`
  classification step in its Step-1 prose with no corresponding numbered
  section in the document — add it so the widespread, correctly-implemented
  `envContext` origination pattern (11+ services) has a citable rule instead
  of being inferred from precedent each time.
- Consider re-classifying atlas-maps' `map/monster/registry.go`
  `KeyedHash`→`TenantKeyedHash` conversion the same way as atlas-doors'
  (Blocking finding 2) for consistency, since both are Redis-key-scheme
  changes riding along in a wiring-recipe commit.
