# Query-path scope audit (FR-8.1)

Every Postgres access path in the fleet, classified. A `tenant_id` column is
not evidence; the evidence is the `WHERE` clause of every read and the
assignment of every write.

Verdicts:
- `SCOPED` — every query path filters on tenant (data plane) or environment
  (control plane). Cite the file:line of the narrowest query builder.
- `TRANSITIVE` — no direct scoping column; every access goes through a
  `SCOPED` parent. Cite the parent and the join.
- `UNSCOPED` — at least one query path does not filter. **Blocking.** Cite it.
- `CONTROL` — control plane; scoped by environment after Task 10/11.
- `FORCES-ISOLATED` — cannot be scoped; the service must escalate to isolated
  mode (FR-9.3). Requires a written reason.

## Fleet-wide mechanism (read this before the table)

`libs/atlas-database/connection.go:139` — `Connect()` calls
`registerTenantCallbacks(l, db)` unconditionally for every service that
opens its DB through the shared connector. `libs/atlas-database/tenant_scope.go`
registers GORM callbacks on Query/Row/Create/Update/Delete
(`tenant_scope.go:47-52`) that, for any entity whose GORM schema has a
`tenant_id` column (`hasTenantColumn`, `tenant_scope.go:31-37`):

- inject `WHERE tenant_id = <ctx tenant>` into every read/update/delete
  (`tenantQueryCallback`, `tenant_scope.go:54-81`), and
- inject `tenant_id = <ctx tenant>` into every create whose struct literal
  left the field zero (`tenantCreateCallback` /
  `injectTenantIdIfZero`, `tenant_scope.go:83-133`),

unless the call's context was wrapped in `database.WithoutTenantFilter`
(`tenant_scope.go:22-24`), or the request context carries no tenant at all
(callback logs at Debug and silently no-ops, `tenant_scope.go:69-73`).

Consequence for this audit: an entity struct with a `TenantId` field, read
and written through ordinary `*gorm.DB` calls (no raw `Exec`/`Raw` bypassing
the ORM, no `WithoutTenantFilter` without a hand-verified replacement filter),
is `SCOPED` by this global mechanism even where the local `provider.go`/
`administrator.go` carries no explicit `tenant_id` WHERE of its own — the
explicit filters present in some packages are defense-in-depth, not the sole
scoping mechanism. Each `SCOPED` row below cites the callback file:line
alongside the entity's own `TenantId` field and, where present, its explicit
filter. Every row was checked for raw-SQL bypass (`grep -n ".Exec(\|.Raw("`
over the package, excluding `_test.go`) and for `WithoutTenantFilter` usage;
both are called out explicitly where found.

Services confirmed to have **no Postgres persistence at all** (no
`gorm.io/gorm` import anywhere in the module) are listed at the end of this
task's section rather than given rows — there is nothing to classify.

| Service | Table / entity | Plane | Verdict | Evidence (file:line) | Notes |
|---|---|---|---|---|---|
| atlas-account | accounts (`account.Entity`) | Data | SCOPED | `services/atlas-account/atlas.com/account/account/entity.go:14` (TenantId field); `libs/atlas-database/tenant_scope.go:75-79` (automatic WHERE injection); reads at `services/atlas-account/atlas.com/account/account/provider.go:14,25`; writes at `services/atlas-account/atlas.com/account/account/administrator.go:12-23,36,43` | No raw SQL; no `WithoutTenantFilter`. Explicit WHERE in provider.go is by `id`/`name` only — tenant scoping is entirely the automatic callback. |
| atlas-ban | bans (`ban.Entity`) | Data | SCOPED | `services/atlas-ban/atlas.com/ban/ban/entity.go:15` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-ban/atlas.com/ban/ban/provider.go:16,32,40,52` | No raw SQL. Reads filter by id/type/value only; tenant scoping is the automatic callback. |
| atlas-ban | reports (`report.Entity`) | Data | SCOPED | `services/atlas-ban/atlas.com/ban/report/entity.go:22` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-ban/atlas.com/ban/report/provider.go:34,45,56` | No raw SQL. |
| atlas-ban | login_history (`history.Entity`) | Data | SCOPED | `services/atlas-ban/atlas.com/ban/history/entity.go:15` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-ban/atlas.com/ban/history/provider.go:13,19,25` | No raw SQL. |
| atlas-buddies | lists (`list.Entity`) | Data | SCOPED | `services/atlas-buddies/atlas.com/buddies/list/entity.go:14` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; read at `services/atlas-buddies/atlas.com/buddies/list/provider.go:14`; writes at `services/atlas-buddies/atlas.com/buddies/list/administrator.go:12-24,65,137` | No raw SQL, no `WithoutTenantFilter`. |
| atlas-buddies | buddies (`buddy.Entity`) | Data | SCOPED | `services/atlas-buddies/atlas.com/buddies/buddy/entity.go:16` (TenantId, indexed); `libs/atlas-database/tenant_scope.go:75-79,83-118`; writes (create/delete) at `services/atlas-buddies/atlas.com/buddies/list/administrator.go:36-44,132` | Has its own `TenantId` column (independently callback-scoped on every op) in addition to being reachable transitively via `list.Id`→`ListId` join/Preload (`list/provider.go:14`). Migration backfill at `buddy/entity.go:14-20` runs a one-time raw `db.Exec` UPDATE keyed by joining to `lists.tenant_id`, not a request-time read path — not a live query-scope gap. |
| atlas-cashshop | accounts / cash wallets (`wallet.Entity`) | Data | SCOPED | `services/atlas-cashshop/atlas.com/cashshop/wallet/entity.go:15` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; read at `services/atlas-cashshop/atlas.com/cashshop/wallet/provider.go:14` | No raw SQL. |
| atlas-cashshop | coupons (`coupon.Entity`) | Data | SCOPED | `services/atlas-cashshop/atlas.com/cashshop/coupon/entity.go:22` (TenantId, uniqueIndex w/ tenant+code); explicit reads at `services/atlas-cashshop/atlas.com/cashshop/coupon/provider.go:20,31,62` | Explicit `tenant_id = ?` in every read (defense-in-depth on top of the automatic callback). |
| atlas-cashshop | wishlist_items (`wishlist.Entity`) | Data | SCOPED | `services/atlas-cashshop/atlas.com/cashshop/wishlist/entity.go:14` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; read at `services/atlas-cashshop/atlas.com/cashshop/wishlist/provider.go:15` | No raw SQL; automatic callback only. |
| atlas-cashshop | cash_surprise_openings (`opening.entity`) | Data | SCOPED | `services/atlas-cashshop/atlas.com/cashshop/surprise/opening/entity.go:24` (TenantId, part of PK); write at `services/atlas-cashshop/atlas.com/cashshop/surprise/opening/administrator.go:27-33` | Insert-only ledger; TenantId set explicitly in struct literal (`administrator.go:28`), also part of primary key. |
| atlas-cashshop | coupon_redemptions (`redemption.Entity`) | Data | SCOPED | `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/entity.go:21` (TenantId, uniqueIndex); explicit reads at `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/provider.go:21,30` | Explicit `tenant_id = ?` in every read. |
| atlas-cashshop | coupon_batches (`batch.Entity`) | Data | SCOPED | `services/atlas-cashshop/atlas.com/cashshop/coupon/batch/entity.go:16` (TenantId); explicit reads at `services/atlas-cashshop/atlas.com/cashshop/coupon/batch/provider.go:15,31` | Explicit `tenant_id = ?` in every read. |
| atlas-cashshop | cash_assets (`asset.Entity`) | Data | SCOPED | `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/entity.go:15` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/provider.go:16,26,36` | No raw SQL; automatic callback only (reads filter by id/compartment/cash id). |
| atlas-cashshop | cash_compartments (`compartment.Entity`) | Data | SCOPED | `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/compartment/entity.go:14` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/compartment/provider.go:17,29,41` | No raw SQL; automatic callback only. |
| atlas-character | saved_locations (`saved_location.entity`) | Data | SCOPED | `services/atlas-character/atlas.com/character/saved_location/entity.go:16` (TenantId, part of unique index); writes/reads at `services/atlas-character/atlas.com/character/saved_location/administrator.go:9-28,29-36,38-40` | Query builders live in `administrator.go`, not `provider.go` (package deviates from convention — `provider.go` holds only the entity→Model mapper). Upsert sets TenantId in struct literal (`administrator.go:12`); reads/deletes rely on the automatic callback (no explicit tenant_id in their WHERE, `administrator.go:31,39`). |
| atlas-character | teleport_rock_maps (`teleport_rock.entity`) | Data | SCOPED | `services/atlas-character/atlas.com/character/teleport_rock/entity.go:16` (TenantId, part of unique index); explicit reads/writes at `services/atlas-character/atlas.com/character/teleport_rock/administrator.go:10-12,22-23,31,44-45,51-53` | Query builders live in `administrator.go`. Every path passes `tenantId` explicitly (defense-in-depth on top of the automatic callback). |
| atlas-character | characters (`character.entity`) | Data | SCOPED | `services/atlas-character/atlas.com/character/character/entity.go:29` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-character/atlas.com/character/character/provider.go:14,20,26` | No raw SQL; automatic callback only. |
| atlas-character | session_history (`history.entity`) | Data | SCOPED | `services/atlas-character/atlas.com/character/session/history/entity.go:16` (TenantId, indexed); reads/writes at `services/atlas-character/atlas.com/character/session/history/administrator.go:15-17,34,44,59,79,90` | Query builders live in `administrator.go`, not `provider.go` (package has no `provider.go`). No explicit `tenant_id` in any WHERE — scoping is entirely the automatic callback. |
| atlas-configurations | templates (`templates.Entity`) | Control | CONTROL | `services/atlas-configurations/atlas.com/configurations/templates/entity.go:15-20` (no TenantId column — global socket template registry); reads at `services/atlas-configurations/atlas.com/configurations/templates/provider.go:24,37` | No tenant scoping column by design (control-plane, shared by every tenant on a version per `isolation-audit.md` §1). To be scoped by `environment` under Task 10/11; currently unscoped by any column. |
| atlas-configurations | services / service_history (`services.Entity`, `services.HistoryEntity`) | Control | CONTROL | `services/atlas-configurations/atlas.com/configurations/services/entity.go:15-30` (no TenantId column); reads at `services/atlas-configurations/atlas.com/configurations/services/provider.go:24` | Control-plane service registry, shared. Same disposition as templates. |
| atlas-configurations | tenants / tenant_history (`tenants.Entity`, `tenants.HistoryEntity`) | Control | CONTROL | `services/atlas-configurations/atlas.com/configurations/tenants/entity.go:15-30` (no TenantId column — this table IS the tenant registry); reads at `services/atlas-configurations/atlas.com/configurations/tenants/provider.go:24,37` | The tenant registry itself cannot be scoped to a tenant (category error per `isolation-audit.md` §1). To be scoped by `environment` under Task 10/11. |
| atlas-data | reactor_search_index (`reactor.SearchIndexEntity`) | Data | SCOPED | `services/atlas-data/atlas.com/data/reactor/entity.go:12` (TenantId, part of PK); reads via `services/atlas-data/atlas.com/data/searchindex/searchindex.go:181,207,237,283,306` (`Search`/`SearchWithFilter`/`Count`/`CountWithFilter`, all filter `"tenant_id = ?"`) | Reads bypass the automatic callback (`database.WithoutTenantFilter`, `searchindex.go:138,235,264,304`) but explicitly filter on a resolved partition tenant id (`searchindex.go:97-116`, `ResolvePartitionTenantId`) — either the request tenant or a version-scoped "canonical" shared-content tenant when the request tenant has zero rows of its own. Every generated query still carries `tenant_id = ?` bound to that resolved id; this is a deliberate shared-content partitioning design (issue #1213), not an unscoped gap. |
| atlas-data | npc_search_index (`npc.SearchIndexEntity`) | Data | SCOPED | `services/atlas-data/atlas.com/data/npc/entity.go:12` (TenantId, part of PK); same `searchindex.go` read paths as reactor_search_index above | Same partition-resolution pattern as reactor_search_index. |
| atlas-data | npc_spawn_index (`npc.SpawnIndexEntity`) | Data | SCOPED | `services/atlas-data/atlas.com/data/npc/spawn_index.go:15` (TenantId, part of PK); read at `services/atlas-data/atlas.com/data/npc/spawn_index.go:33-41` (`SpawnMapsFor`) | Bypasses automatic callback (`database.WithoutTenantFilter`, `spawn_index.go:40`) but explicitly filters `"tenant_id = ? AND npc_id = ?"` (`spawn_index.go:41`) on a resolved partition id (`searchindex.ResolvePartitionTenantId`, `spawn_index.go:34`). Same shared-content pattern as the search-index tables; this is the table flagged in project memory as absent from `baseline.DumpTables` (an ingest/restore-completeness issue, not a query-scope gap). |
| atlas-data | documents (`document.Entity`) | Data | SCOPED | `services/atlas-data/atlas.com/data/document/entity.go:17` (TenantId, uniqueIndex w/ type+docid); `libs/atlas-database/tenant_scope.go:75-79` | No raw SQL; automatic callback (no explicit tenant filter found in this package's read paths beyond the callback). |
| atlas-data | monster_search_index (`monster.SearchIndexEntity`) | Data | SCOPED | `services/atlas-data/atlas.com/data/monster/entity.go:12` (TenantId, part of PK); same `searchindex.go` read paths as reactor_search_index above | Same partition-resolution pattern. |
| atlas-data | monster_spawn_index (`monster.SpawnIndexEntity`) | Data | SCOPED | `services/atlas-data/atlas.com/data/monster/spawn_index.go:15` (TenantId, part of PK); read at `services/atlas-data/atlas.com/data/monster/spawn_index.go:39-47` (`SpawnMapsFor`) | Same partition-resolution pattern as npc_spawn_index. |
| atlas-data | map_search_index (`_map.SearchIndexEntity`) | Data | SCOPED | `services/atlas-data/atlas.com/data/map/entity.go:12` (TenantId, part of PK); same `searchindex.go` read paths as reactor_search_index above | Same partition-resolution pattern; no separate `map_spawn_index` table exists (`find . -name spawn_index.go` returns only `npc/spawn_index.go` and `monster/spawn_index.go`). |
| atlas-drop-information | reactor_drops (`drop.entity`, reactor pkg) | Data | SCOPED | `services/atlas-drop-information/atlas.com/dis/reactor/drop/entity.go:13` (TenantId); `libs/atlas-database/tenant_scope.go:75-79` | No non-test raw SQL; automatic callback only. |
| atlas-drop-information | monster_drops (`drop.entity`, monster pkg) | Data | SCOPED | `services/atlas-drop-information/atlas.com/dis/monster/drop/entity.go:13` (TenantId); `libs/atlas-database/tenant_scope.go:75-79` | No non-test raw SQL; automatic callback only. |
| atlas-drop-information | continent_drops (`drop.entity`, continent pkg) | Data | SCOPED | `services/atlas-drop-information/atlas.com/dis/continent/drop/entity.go:13` (TenantId); `libs/atlas-database/tenant_scope.go:75-79` | No non-test raw SQL; automatic callback only. |
| atlas-fame | logs (`fame.Entity`) | Data | SCOPED | `services/atlas-fame/atlas.com/fame/fame/entity.go:15` (TenantId); `libs/atlas-database/tenant_scope.go:75-79` | No raw SQL; automatic callback only. |

### Services in this third with no Postgres persistence (no rows)

Confirmed via `grep -rl "gorm.io/gorm\|\*gorm.DB"` returning empty across the
whole module for each: **atlas-asset-expiration, atlas-buffs, atlas-chairs,
atlas-chalkboards, atlas-channel, atlas-character-factory,
atlas-consumables, atlas-doors, atlas-drops, atlas-effective-stats,
atlas-expressions**. Nothing to classify for FR-8.1 — these services carry no
Postgres tables at all (state lives elsewhere, e.g. Redis, or is transient).

### Findings

No `UNSCOPED` rows in this third. Every entity with a `TenantId` column relies
on (at minimum) the fleet-wide automatic GORM tenant callback
(`libs/atlas-database/tenant_scope.go`); several packages additionally carry
explicit `tenant_id` filters as defense-in-depth. The `atlas-data`
search-index and spawn-index tables use a deliberate, explicitly-filtered
partition-resolution pattern (`searchindex.ResolvePartitionTenantId`) that
bypasses the automatic callback by design, for shared-content fallback — every
generated query still carries an explicit `tenant_id = ?` bound to the
resolved partition, so this is `SCOPED`, not `UNSCOPED`. No follow-up task
added to the plan for this third.

The `atlas-configurations` control-plane tables (`templates`, `services`,
`tenants` + their history tables) carry no tenant scoping column by design —
correctly `CONTROL`, to be scoped by `environment` once Task 10/11 lands.
