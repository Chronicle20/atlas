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
| atlas-ban | bans (`ban.Entity`) | Data | UNSCOPED | `services/atlas-ban/atlas.com/ban/ban/entity.go:15` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-ban/atlas.com/ban/ban/provider.go:16,32,40,52` | **Regraded UNSCOPED by §3.** Request-path reads above are still SCOPED via the automatic callback (original evidence stands for that path), but `ExpiredBanCleanup.Run` (`ban/task.go:28-36`) explicitly calls `database.WithoutTenantFilter(t.ctx)` then `t.db.WithContext(noTenantCtx).Where("permanent = ? AND expires_at <= ?", false, now).Delete(&Entity{})` — a bulk delete filtered only by a non-tenant predicate, no per-row tenant re-derivation. Wired live at boot (`main.go:94`, `rt.Context()`, 5-minute interval). See §3. |
| atlas-ban | reports (`report.Entity`) | Data | SCOPED | `services/atlas-ban/atlas.com/ban/report/entity.go:22` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-ban/atlas.com/ban/report/provider.go:34,45,56` | No raw SQL. |
| atlas-ban | login_history (`history.Entity`) | Data | UNSCOPED | `services/atlas-ban/atlas.com/ban/history/entity.go:15` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-ban/atlas.com/ban/history/provider.go:13,19,25` | **Regraded UNSCOPED by §3.** Request-path reads above are still SCOPED via the automatic callback (original evidence stands for that path), but `HistoryPurge.Run` (`ban/history/task.go:28-36`) explicitly calls `database.WithoutTenantFilter(t.ctx)` then `t.db.WithContext(noTenantCtx).Where("created_at < ?", cutoff).Delete(&Entity{})` — same bulk-delete-by-non-tenant-predicate shape. Wired live at boot (`main.go:97`, `rt.Context()`, 24-hour interval). See §3. |
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
| atlas-families | family_members (`family.Entity`) | Data | SCOPED | `services/atlas-families/atlas.com/family/family/entity.go:16` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-families/atlas.com/family/family/provider.go:14,28,42,53` | No raw SQL in request path (the `db.Exec` at `entity.go:43-107` is one-time `Migration` DDL: index/constraint creation, not a live query). No `WithoutTenantFilter`. |
| atlas-guilds | guilds (`guild.Entity`) | Data | SCOPED | `services/atlas-guilds/atlas.com/guilds/guild/entity.go:18` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-guilds/atlas.com/guilds/guild/provider.go:13,30,38,49`; writes at `services/atlas-guilds/atlas.com/guilds/guild/administrator.go:10,25,43,56,69` | No raw SQL; no `WithoutTenantFilter`. Automatic callback only. |
| atlas-guilds | threads (`thread.Entity`) | Data | SCOPED | `services/atlas-guilds/atlas.com/guilds/thread/entity.go:18` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-guilds/atlas.com/guilds/thread/provider.go:16,22`; writes at `services/atlas-guilds/atlas.com/guilds/thread/administrator.go:10,33,50` | No raw SQL; automatic callback only. |
| atlas-guilds | titles (`title.Entity`) | Data | SCOPED | `services/atlas-guilds/atlas.com/guilds/guild/title/entity.go:13` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; read at `services/atlas-guilds/atlas.com/guilds/guild/title/provider.go:10`; writes at `services/atlas-guilds/atlas.com/guilds/guild/title/administrator.go:10,14` | No raw SQL; automatic callback only. |
| atlas-guilds | members (`member.Entity`) | Data | SCOPED | `services/atlas-guilds/atlas.com/guilds/guild/member/entity.go:13` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-guilds/atlas.com/guilds/guild/member/provider.go:10,21`; writes at `services/atlas-guilds/atlas.com/guilds/guild/member/administrator.go:8,26,32` | No raw SQL; automatic callback only. |
| atlas-guilds | characters (`character.Entity`) | Data | SCOPED | `services/atlas-guilds/atlas.com/guilds/guild/character/entity.go:13` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; read at `services/atlas-guilds/atlas.com/guilds/guild/character/provider.go:10` | No raw SQL; automatic callback only. |
| atlas-guilds | replies (`reply.Entity`) | Data | SCOPED | `services/atlas-guilds/atlas.com/guilds/thread/reply/entity.go:15` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; read at `services/atlas-guilds/atlas.com/guilds/thread/reply/provider.go:10`; writes at `services/atlas-guilds/atlas.com/guilds/thread/reply/administrator.go:10,25` | No raw SQL; automatic callback only. |
| atlas-inventory | assets (`asset.Entity`) | Data | SCOPED | `services/atlas-inventory/atlas.com/inventory/asset/entity.go:22` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-inventory/atlas.com/inventory/asset/provider.go:12,18,24,30`; writes at `services/atlas-inventory/atlas.com/inventory/asset/administrator.go:10,57-116` | The `db.Exec` at `entity.go:12` is one-time `Migration` DDL (flag-bitmask backfill), not a live query. No `WithoutTenantFilter`. |
| atlas-inventory | compartments (`compartment.Entity`) | Data | SCOPED | `services/atlas-inventory/atlas.com/inventory/compartment/entity.go:15` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-inventory/atlas.com/inventory/compartment/provider.go:14,20,26`; writes at `services/atlas-inventory/atlas.com/inventory/compartment/administrator.go:10,25,45` | No raw SQL; automatic callback only. |
| atlas-keys | keys (`key.entity`) | Data | SCOPED | `services/atlas-keys/atlas.com/keys/key/entity.go:13` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-keys/atlas.com/keys/key/provider.go:11,17`; writes at `services/atlas-keys/atlas.com/keys/key/administrator.go:8,24,28` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-map-actions | map_scripts (`script.Entity`) | Data | SCOPED | `services/atlas-map-actions/atlas.com/map-actions/script/entity.go:17` (`TenantID`, column `tenant_id`); `libs/atlas-database/tenant_scope.go:31-37,75-79` (callback matches on DB column name, not Go field name); reads at `services/atlas-map-actions/atlas.com/map-actions/script/provider.go:12,23,37,45`; writes at `services/atlas-map-actions/atlas.com/map-actions/script/administrator.go:11,32,69,77,86,92` | No raw SQL; no `WithoutTenantFilter`. `DeleteAllMapScripts`/`DeleteAllByType` (`administrator.go:77,86`) still route through the automatic per-tenant WHERE — they are bulk-by-predicate, not `WithoutTenantFilter`. |
| atlas-maps | character_map_visits (`visit.Entity`) | Data | SCOPED | `services/atlas-maps/atlas.com/maps/visit/entity.go:14` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; read at `services/atlas-maps/atlas.com/maps/visit/provider.go:10`; writes at `services/atlas-maps/atlas.com/maps/visit/administrator.go:10,27` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-maps | character_locations (`location.entity`) | Data | SCOPED | `services/atlas-maps/atlas.com/maps/character/location/entity.go:19` (TenantId, part of PK); `libs/atlas-database/tenant_scope.go:75-79`; read at `services/atlas-maps/atlas.com/maps/character/location/provider.go:13`; writes at `services/atlas-maps/atlas.com/maps/character/location/administrator.go:14,31` | No raw SQL. Read/write helpers additionally take `tenantId` as an explicit parameter (defense-in-depth on top of the automatic callback). |
| atlas-marriages | marriages (`marriage.Entity`) | Data | SCOPED | `services/atlas-marriages/atlas.com/marriages/marriage/entity.go:21` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-marriages/atlas.com/marriages/marriage/provider.go:180,207,234`; writes at `services/atlas-marriages/atlas.com/marriages/marriage/administrator.go:63,94` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-marriages | proposals (`ProposalEntity`) | Data | UNSCOPED | `services/atlas-marriages/atlas.com/marriages/marriage/entity.go:84` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-marriages/atlas.com/marriages/marriage/provider.go:15,35,79,91,123,150,439`; writes at `services/atlas-marriages/atlas.com/marriages/marriage/administrator.go:14,47` | **Regraded UNSCOPED by §3.** Request-path reads/writes above are still SCOPED via the automatic callback (original evidence stands for that path), but `ProposalExpiryScheduler.getTenantsWithProposals` (`scheduler/proposal_expiry.go:112-119`) runs `s.db.Model(&marriage.ProposalEntity{}).Where("status = ?", ...).Distinct("tenant_id").Pluck(...)` directly on the struct's `db` field with no `.WithContext` anywhere on the chain — cross-tenant discovery read, per-row tenant re-derivation before the compensating write. See §3. |
| atlas-marriages | ceremonies (`CeremonyEntity`) | Data | UNSCOPED | `services/atlas-marriages/atlas.com/marriages/marriage/entity.go:145` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-marriages/atlas.com/marriages/marriage/provider.go:244,269,357,383,409`; writes at `services/atlas-marriages/atlas.com/marriages/marriage/administrator.go:110,151` | **Regraded UNSCOPED by §3.** Request-path reads/writes above are still SCOPED via the automatic callback (original evidence stands for that path), but `CeremonyTimeoutScheduler.getTenantsWithActiveCeremonies` (`scheduler/ceremony_timeout.go:108-115`) runs `s.db.Model(&marriage.CeremonyEntity{}).Where("status = ?", ...).Distinct("tenant_id").Pluck(...)` directly on the struct's `db` field with no `.WithContext` anywhere on the chain — same shape as `proposals`. See §3. |
| atlas-merchant | frederick_items, frederick_mesos (`ItemEntity`, `MesoEntity`) | Data | UNSCOPED | `services/atlas-merchant/atlas.com/merchant/frederick/entity.go:14,31` (TenantId); request-path reads/writes are `SCOPED` via `libs/atlas-database/tenant_scope.go:75-79` (e.g. `services/atlas-merchant/atlas.com/merchant/frederick/provider.go`); but `CleanupTask.Run` (`services/atlas-merchant/atlas.com/merchant/frederick/task.go:31`) runs `database.WithoutTenantFilter` and `cleanupExpiredItems`/`cleanupExpiredMesos` (`administrator.go:112-120,142-150`) `db.Where("stored_at < ?", cutoff).Delete(...)` with **no tenant predicate at all** | Blocking: a single tick of this periodic cleanup deletes expired-item/meso rows across every tenant in one statement — the delete predicate is purely time-based, with no per-row tenant re-derivation (unlike the notification task below). After per-PR DB isolation is removed, one PR environment's cleanup tick deletes another environment's data. |
| atlas-merchant | frederick_notifications (`NotificationEntity`) | Data | UNSCOPED | `services/atlas-merchant/atlas.com/merchant/frederick/notification_entity.go:13` (TenantId); `NotificationTask.Run` (`services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go:36`) runs `database.WithoutTenantFilter` then `Find(&notifications)` (`notification_task.go:39-41`) with **no tenant predicate**, returning due notifications for every tenant into one process | Blocking under the letter of this audit's verdict (the `SELECT` itself does not filter), though the design is more defensible than the cleanup task above: each row's subsequent write (`advanceNotification`/`deleteNotification`, `notification_task.go:72,76`) is addressed by the row's own unique `Id`, and the Kafka emit re-derives a `tenant.Model` per row (`notification_task.go:60-68`) before publishing — so the mutation itself cannot cross tenants, but the read does. |
| atlas-merchant | listings (`listing.Entity`) | Data | SCOPED | `services/atlas-merchant/atlas.com/merchant/listing/entity.go:14` (TenantId); `libs/atlas-database/tenant_scope.go:75-79` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-merchant | merchant_visits (`visit.Entity`) | Data | SCOPED | `services/atlas-merchant/atlas.com/merchant/visit/entity.go:15` (TenantId); `libs/atlas-database/tenant_scope.go:75-79` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-merchant | merchant_blacklists (`blacklist.Entity`) | Data | SCOPED | `services/atlas-merchant/atlas.com/merchant/blacklist/entity.go:15` (TenantId); `libs/atlas-database/tenant_scope.go:75-79` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-merchant | listing_search_counts (`searchcount.Entity`) | Data | SCOPED | `services/atlas-merchant/atlas.com/merchant/searchcount/entity.go:16` (TenantId); `libs/atlas-database/tenant_scope.go:75-79` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-merchant | messages (`message.Entity`) | Data | SCOPED | `services/atlas-merchant/atlas.com/merchant/message/entity.go:13` (TenantId); `libs/atlas-database/tenant_scope.go:75-79` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-merchant | shops (`shop.Entity`) | Data | UNSCOPED | `services/atlas-merchant/atlas.com/merchant/shop/entity.go:16` (TenantId); request-path reads/writes are `SCOPED` via `libs/atlas-database/tenant_scope.go:75-79`; but `ExpirationTask.Run` (`services/atlas-merchant/atlas.com/merchant/shop/task.go:29`) runs `database.WithoutTenantFilter` then `getExpired()` (`shop/provider.go:135-144`), whose `db.Where("expires_at IS NOT NULL AND expires_at < ? AND state IN (?, ?, ?)", ...)` (`provider.go:138`) carries **no tenant predicate** | Same shape as the frederick notification task: the cross-tenant `SELECT` (comment at `task.go:31-33` states the cross-tenant sweep is intentional) is followed by a per-row `tenant.Create`/`tenant.WithContext` reconstruction (`task.go:47-52`) before the compensating `CloseShopAndEmit` write, so the mutation is scoped by row identity even though the read is not. Still `UNSCOPED` per this audit's verdict (a query path exists with no filter). |
| atlas-mini-games | game_records (`record.Entity`) | Data | SCOPED | `services/atlas-mini-games/atlas.com/mini-games/record/entity.go:20` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-mini-games/atlas.com/mini-games/record/provider.go:21,37`; writes at `services/atlas-mini-games/atlas.com/mini-games/record/administrator.go:19,54` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-monster-book | monster_book_collections (`collection.entity`) | Data | SCOPED | `services/atlas-monster-book/atlas.com/monster-book/collection/entity.go:15` (TenantId, part of PK); reads/writes take `tenantId` explicitly at `services/atlas-monster-book/atlas.com/monster-book/collection/provider.go:12` and `administrator.go:29,56,85,91` | Explicit `tenantId` parameter threaded through every query builder (defense-in-depth on top of the automatic callback). No raw SQL. |
| atlas-monster-book | monster_book_cards (`card.entity`) | Data | SCOPED | `services/atlas-monster-book/atlas.com/monster-book/card/entity.go:15` (TenantId, part of PK); reads/writes take `tenantId` explicitly at `services/atlas-monster-book/atlas.com/monster-book/card/provider.go:13,19,25` and `administrator.go:22,78` | Explicit `tenantId` parameter throughout. No raw SQL. |
| atlas-mounts | character_mounts (`mount.Entity`) | Data | SCOPED | `services/atlas-mounts/atlas.com/mounts/mount/entity.go:15` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads/writes at `services/atlas-mounts/atlas.com/mounts/mount/administrator.go:12,34,45` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-mts | listings (`listing.entity`) | Data | UNSCOPED | `services/atlas-mts/atlas.com/mts/listing/entity.go:42` (TenantId); request-path reads/writes are `SCOPED` via `libs/atlas-database/tenant_scope.go:75-79`; but the periodic sweep (`services/atlas-mts/atlas.com/mts/task/periodic.go:106`) runs `database.WithoutTenantFilter` then calls `CountExpiredActive`/`GetExpiredActive` (`listing/administrator.go:54-58,63-65`), which carry no tenant predicate | Extensively documented as deliberate (`periodic.go:96-107`): the sweep discovers expired auctions across every tenant, then addresses each by its unique surrogate `uuid` and derives the target tenant from the row itself for the compensating write. Still `UNSCOPED` per this audit's verdict — the discovery query itself reads across every tenant. |
| atlas-mts | wish_entries (`wish.entity`) | Data | UNSCOPED | `services/atlas-mts/atlas.com/mts/wish/entity.go:50` (TenantId); request-path reads/writes are `SCOPED` via `libs/atlas-database/tenant_scope.go:75-79`; but `DeleteExpiredWanted` (`wish/administrator.go:139-141`) is called from the same `WithoutTenantFilter` sweep context (`task/periodic.go:106,186`) and its `db.Where("type = ? AND expires_at IS NOT NULL AND expires_at < ?", ...)` (`administrator.go:141`) carries **no tenant predicate and no per-row tenant re-derivation** — it is a bulk hard-delete | Blocking, same shape as the frederick cleanup task: unlike the listing sweep above, this delete has no per-row compensating tenant binding at all — one tick removes expired want-ads for every tenant in a single statement. |
| atlas-mts | mts_transactions (`transaction.entity`) | Data | SCOPED | `services/atlas-mts/atlas.com/mts/transaction/entity.go:25` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-mts/atlas.com/mts/transaction/provider.go:18,37`; write at `services/atlas-mts/atlas.com/mts/transaction/administrator.go:15` | No `WithoutTenantFilter` on this entity's own paths (the sweep above touches `listing`/`wish`, not `transaction`). No raw SQL. |
| atlas-mts | mts_serials (`serial.entity`) | Data | SCOPED | `services/atlas-mts/atlas.com/mts/serial/entity.go:19` (TenantId, part of PK); write at `services/atlas-mts/atlas.com/mts/serial/administrator.go:47` (`Next`, takes `tenantId` explicitly) | Explicit `tenantId` parameter. No raw SQL, no `WithoutTenantFilter`. |
| atlas-mts | holdings (`holding.entity`) | Data | SCOPED | `services/atlas-mts/atlas.com/mts/holding/entity.go:42` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-mts/atlas.com/mts/holding/provider.go:16,22,35` | No `WithoutTenantFilter` on this entity's own paths. No raw SQL. |
| atlas-mts | bids (`bid.entity`) | Data | SCOPED | `services/atlas-mts/atlas.com/mts/bid/entity.go:28` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-mts/atlas.com/mts/bid/provider.go:11,17,29` | No `WithoutTenantFilter` on this entity's own paths. No raw SQL. |
| atlas-notes | notes (`note.Entity`) | Data | SCOPED | `services/atlas-notes/atlas.com/notes/note/entity.go:13` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-notes/atlas.com/notes/note/provider.go:10,21,27`; writes at `services/atlas-notes/atlas.com/notes/note/administrator.go:10,24,41,47` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-npc-conversations | quest_conversations (`quest.Entity`) | Data | SCOPED | `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/entity.go:14` (`TenantID`, column `tenant_id`); `libs/atlas-database/tenant_scope.go:31-37,75-79`; reads at `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/provider.go:12,23,35`; writes at `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/administrator.go:11,32,74,82` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-npc-conversations | recipes (`recipe.Entity`) | Data | SCOPED | `services/atlas-npc-conversations/atlas.com/npc/conversation/recipe/entity.go:14` (`TenantID`, column `tenant_id`); `libs/atlas-database/tenant_scope.go:31-37,75-79`; reads at `services/atlas-npc-conversations/atlas.com/npc/conversation/recipe/provider.go:14,23,31`; writes at `services/atlas-npc-conversations/atlas.com/npc/conversation/recipe/administrator.go:12,32,41` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-npc-conversations | conversations (`npc.Entity`) | Data | SCOPED | `services/atlas-npc-conversations/atlas.com/npc/conversation/npc/entity.go:14` (`TenantID`, column `tenant_id`); `libs/atlas-database/tenant_scope.go:31-37,75-79`; reads at `services/atlas-npc-conversations/atlas.com/npc/conversation/npc/provider.go:12,23,35,43`; writes at `services/atlas-npc-conversations/atlas.com/npc/conversation/npc/administrator.go:11,32,73,81` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-npc-shops | shops (`shops.Entity`) | Data | SCOPED | `services/atlas-npc-shops/atlas.com/npc/shops/entity.go:12` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-npc-shops/atlas.com/npc/shops/provider.go:14,32,39`; writes at `services/atlas-npc-shops/atlas.com/npc/shops/administrator.go:15,32,53,64,70` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-npc-shops | commodities (`commodities.Entity`) | Data | SCOPED | `services/atlas-npc-shops/atlas.com/npc/commodities/entity.go:12` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-npc-shops/atlas.com/npc/commodities/provider.go:12,25,37,62,76`; writes at `services/atlas-npc-shops/atlas.com/npc/commodities/administrator.go:14,39,62,68,74,81,87` | The `db.Exec` at `entity.go:46` is one-time `Migration` DDL (index creation), not a live query. No `WithoutTenantFilter`. |
| atlas-party-quests | definitions (`definition.Entity`) | Data | SCOPED | `services/atlas-party-quests/atlas.com/party-quests/definition/entity.go:13` (`TenantID`, column `tenant_id`); `libs/atlas-database/tenant_scope.go:31-37,75-79`; reads at `services/atlas-party-quests/atlas.com/party-quests/definition/provider.go:11,21,31`; writes at `services/atlas-party-quests/atlas.com/party-quests/definition/administrator.go:10,30,65,72` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-pets | pets (`pet.Entity`) | Data | SCOPED | `services/atlas-pets/atlas.com/pets/pet/entity.go:18` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-pets/atlas.com/pets/pet/provider.go:11,22,37`; writes at `services/atlas-pets/atlas.com/pets/pet/administrator.go:13,39,57,75,93,111,129,147` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-pets | excludes (`exclude.Entity`) | Data | SCOPED | `services/atlas-pets/atlas.com/pets/pet/exclude/entity.go:26` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; write at `services/atlas-pets/atlas.com/pets/pet/administrator.go:153-172` (`setExcludes`) | Package has no `provider.go`/`administrator.go` of its own — its only query builder is `pet.setExcludes`, cited above (ambiguity rule). The `db.Exec` at `exclude/entity.go:15` is one-time `Migration` DDL (tenant_id backfill from the parent `pets` row), not a live query. `TenantId` is left zero in the `Create` struct literal (`administrator.go:161-166`) and injected by the automatic create callback (`tenant_scope.go:83-133`). |
| atlas-portal-actions | portal_scripts (`script.Entity`) | Data | SCOPED | `services/atlas-portal-actions/atlas.com/portal/script/entity.go:17` (`TenantID`, column `tenant_id`); `libs/atlas-database/tenant_scope.go:31-37,75-79`; reads at `services/atlas-portal-actions/atlas.com/portal/script/provider.go:12,23,35`; writes at `services/atlas-portal-actions/atlas.com/portal/script/administrator.go:11,32,74,82` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-quest | quest_statuses (`quest.Entity`) | Data | SCOPED | `services/atlas-quest/atlas.com/quest/quest/entity.go:16` (TenantId, indexed); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-quest/atlas.com/quest/quest/provider.go:11,27,33,44,61`; writes at `services/atlas-quest/atlas.com/quest/quest/administrator.go:13,30,55,74,96,134,156,178` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-quest | quest_progress (`progress.Entity`) | Data | SCOPED | `services/atlas-quest/atlas.com/quest/quest/progress/entity.go:13` (TenantId, indexed); `libs/atlas-database/tenant_scope.go:75-79` | No provider/administrator of its own — read via `quest.Entity.Progress` foreignKey preload (`quest/entity.go:26`) and written through `quest/administrator.go:96` (`setProgress`, takes `tenantId` explicitly). Own `TenantId` column, independently callback-scoped. |
| atlas-quest | quest_medal_maps (`medal.Entity`) | Data | N/A — orphaned | `services/atlas-quest/atlas.com/quest/quest/medal/entity.go:14-18` (no TenantId column, no FK annotation) | Brief pre-identified this as expected `TRANSITIVE` through the tenant-scoped quest-status parent; **not confirmed at source**. `medal.Migration` is never registered — `services/atlas-quest/atlas.com/quest/main.go:52` calls `database.SetMigrations(quest.Migration, progress.Migration, outboxlib.Migration)` only, so the `quest_medal_maps` table is never created. `grep -rn "medal\." services/atlas-quest --include=*.go` (excluding the entity/model files themselves) finds no provider, administrator, or caller anywhere in the service — the package is dead code with no live query path at all, not a live TRANSITIVE access reachable through a join. No verdict from the defined taxonomy fits an access path that does not exist; flagged for the controller rather than forced into TRANSITIVE. |
| atlas-rankings | character_rankings (`ranking.Entity`) | Data | SCOPED | `services/atlas-rankings/atlas.com/rankings/ranking/entity.go:20` (TenantId, uniqueIndex w/ character); explicit `tenant_id`/context-derived filters throughout `services/atlas-rankings/atlas.com/rankings/ranking/administrator.go:23-42,60-65` (`upsertBatch`, `pruneBefore`) | `pruneBefore` (`administrator.go:60-65`) is the one bulk-delete-by-predicate (`computed_at < ?`) in this third; its own comment documents it as "the highest-risk operation in this package" and it fails closed by calling `tenant.FromContext(db.Statement.Context)` itself before issuing the DELETE, rather than trusting the automatic callback alone (`administrator.go:44-59`). No raw SQL, no `WithoutTenantFilter`. |
| atlas-rankings | ranking_cycles (`ranking.CycleEntity`) | Data | SCOPED | `services/atlas-rankings/atlas.com/rankings/ranking/entity.go:50` (TenantId, uniqueIndex); explicit `tenantId` param at `services/atlas-rankings/atlas.com/rankings/ranking/administrator.go:72,87` (`startCycle`/`completeCycle`) | No raw SQL, no `WithoutTenantFilter`. |
| atlas-reactor-actions | reactor_scripts (`script.Entity`) | Data | SCOPED | `services/atlas-reactor-actions/atlas.com/reactor/script/entity.go:17` (`TenantID`, column `tenant_id`, composite index w/ reactor_id); `libs/atlas-database/tenant_scope.go:31-37,75-79` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-reward-pools | gachapons (`gachapon.entity`) | Data | SCOPED | `services/atlas-reward-pools/atlas.com/reward-pools/gachapon/entity.go:131` (TenantId, uniqueIndex w/ slug id); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-reward-pools/atlas.com/reward-pools/gachapon/provider.go:11,17` | The `tx.Exec` calls at `entity.go:72` are inside `migrateToSurrogatePK` — one-time structural DDL (PK migration to a tenant-scoped surrogate key, documented in the function's own comment), not a live query. No `WithoutTenantFilter`. |
| atlas-reward-pools | global_gachapon_items (`global.entity`) | Data | SCOPED | `services/atlas-reward-pools/atlas.com/reward-pools/global/entity.go:13` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-reward-pools/atlas.com/reward-pools/global/provider.go:11,17,23` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-reward-pools | gachapon_items (`item.entity`) | Data | SCOPED | `services/atlas-reward-pools/atlas.com/reward-pools/item/entity.go:13` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-reward-pools/atlas.com/reward-pools/item/provider.go:11,19,25,31` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-saga-orchestrator | sagas (`saga.Entity`) | Data | UNSCOPED | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/entity.go:16` (TenantId, indexed); request-path reads/writes are `SCOPED` via `libs/atlas-database/tenant_scope.go:75-79` (`store.go:53-97,101-225,253-265,303-330,334-350`); but `GetAllActive` (`store.go:228-236`) and `GetTimedOut` (`store.go:239-250`) both run `database.WithoutTenantFilter(ctx)` with no tenant predicate | Blocking per this audit's verdict, but same shape as the "cross-tenant discovery `SELECT` followed by per-row write addressed by the row's own id/derived tenant" group from part 2: both are startup/recovery reads (`main.go:215` `recoverSagas`, `main.go:268` `reapTimedOutSagas`) that reconstruct `tenant.Model` per row (`main.go:223-224,275-276`) before any subsequent `processor.Step`/`MarkEarliestPendingStep` write, which is addressed by `transactionId` — the mutation cannot cross tenants, but the discovery read does. |
| atlas-skills | macros (`macro.Entity`) | Data | SCOPED | `services/atlas-skills/atlas.com/skills/macro/entity.go:15` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-skills/atlas.com/skills/macro/provider.go:11,22`; writes at `services/atlas-skills/atlas.com/skills/macro/administrator.go:10,14` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-skills | skills (`skill.Entity`) | Data | SCOPED | `services/atlas-skills/atlas.com/skills/skill/entity.go:23` (TenantId, part of composite PK); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-skills/atlas.com/skills/skill/provider.go:11,17`; writes at `services/atlas-skills/atlas.com/skills/skill/administrator.go:14,33,49,85,92` | No raw SQL; no `WithoutTenantFilter`. |
| atlas-storage | storage_assets (`asset.Entity`) | Data | SCOPED | `services/atlas-storage/atlas.com/storage/asset/entity.go:14` (TenantId, indexed); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-storage/atlas.com/storage/asset/provider.go:8,32,43,59`; writes at `services/atlas-storage/atlas.com/storage/asset/administrator.go:9,56,62,68,76` | The `db.Exec` at `entity.go:63` (`Migration`) is one-time flag-bitmask backfill DDL, not a live query. No `WithoutTenantFilter`. |
| atlas-storage | storages (`storage.Entity`) | Data | SCOPED | `services/atlas-storage/atlas.com/storage/storage/entity.go:11` (TenantId, uniqueIndex w/ world+account); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-storage/atlas.com/storage/storage/provider.go:12,37`; writes at `services/atlas-storage/atlas.com/storage/storage/administrator.go:12,30,39,48` | No raw SQL; no `WithoutTenantFilter`. `storage/projection/provider.go` composes `asset.Model`/`storage.Model` in memory (`BuildProjection`) — no `entity.go` of its own, not a separate query path. |
| atlas-tenants | tenants (`tenant.Entity`) | Control | CONTROL | `services/atlas-tenants/atlas.com/tenants/tenant/entity.go:9-16` (no TenantId column — this table IS the tenant registry) | Confirmed at source per brief: no tenant-scoping column exists or could exist (the row *is* the tenant). Same disposition as `atlas-configurations`' `tenants` table — scoped by `environment` under Task 10/11, not by `tenant_id`. |
| atlas-tenants | configurations (`configuration.Entity`) | Data | SCOPED | `services/atlas-tenants/atlas.com/tenants/configuration/entity.go:14` (TenantId); explicit `"tenant_id": tenantID` filter at `services/atlas-tenants/atlas.com/tenants/configuration/provider.go:17-20`, threaded through every `GetBy*`/`GetAll*Provider` in the file (lines 15-448) | Explicit `tenant_id = ?` in every read (defense-in-depth on top of the automatic callback). No raw SQL, no `WithoutTenantFilter`. |
| atlas-trades | trade_escrow_items, trade_escrow_mesos, trade_escrow_meso_stakes, trade_escrow_meso_refunds (`ItemEntity`/`MesoEntity`/`MesoStakeEntity`/`MesoRefundEntity`) | Data | UNSCOPED | `services/atlas-trades/atlas.com/trades/escrow/entity.go:61,201` (+ MesoStakeEntity/MesoRefundEntity, all own TenantId); request-path reads/writes are `SCOPED`, explicit `tenantId` param throughout `provider.go:15-145` and `administrator.go`; but `AllItems`/`AllMesos` (`provider.go:175-190`) call `db.Find(...)` with no `WithContext` and no tenant filter | Blocking per this verdict, same "cross-tenant discovery read, per-row tenant re-derivation" shape as the group below: both are called only from `trade/settlement.go:1311-1316` (`ReconcileEscrow`), whose own comment states "It runs with NO tenant in context and restores each row's own tenant" (`settlement.go:1309-1310`) — reachable only from the boot path and retry ticker per `provider.go:173-174`. `AllMesoStakes` (`provider.go:158-164`) has the identical shape but **no production caller** — `grep -rn "AllMesoStakes"` finds only `escrow/migration_test.go:131,146`; it is exported, unscoped, and currently dead code, not a live gap. |
| atlas-trades | trade_ledger_entries, trade_ledger_sides, trade_ledger_items (`ledger.Entry`/`Side`/`ItemRow`) | Data | SCOPED | `services/atlas-trades/atlas.com/trades/ledger/entity.go:42,64` (TenantId); explicit `tenantId` param threaded through every function in `services/atlas-trades/atlas.com/trades/ledger/provider.go:22-144` and `administrator.go:52,107` | No cross-tenant `AllX`-style function exists in this package (unlike `escrow` and `settlement`). No raw SQL, no `WithoutTenantFilter`. |
| atlas-trades | trade_settlements, trade_settlement_sides, trade_settlement_items (`settlement.Entry`/`Side`/`ItemRow`) | Data | UNSCOPED | `services/atlas-trades/atlas.com/trades/settlement/entity.go:61,87` (TenantId); request-path reads/writes are `SCOPED`, explicit `tenantId` param throughout `provider.go:17-49` and `administrator.go`; but `allUnresolved` (`provider.go:72-80`) runs unfiltered and unscoped | Blocking per this verdict, same shape: called only from `settlement/processor.go:79-81` (`Unresolved`, package function "because there is no tenant to construct a Processor with at boot"), whose caller restores each row's tenant via `Model.Tenant()` per the doc comment at `processor.go:76-78`. Boot-path reconciliation only. |

### Services in this third with no Postgres persistence (no rows)

Confirmed via `grep -rl "gorm.io/gorm\|\*gorm.DB"` returning empty across the
whole module for each: **atlas-asset-expiration, atlas-buffs, atlas-chairs,
atlas-chalkboards, atlas-channel, atlas-character-factory,
atlas-consumables, atlas-doors, atlas-drops, atlas-effective-stats,
atlas-expressions**. Nothing to classify for FR-8.1 — these services carry no
Postgres tables at all (state lives elsewhere, e.g. Redis, or is transient).

### Services in this second third with no Postgres persistence (no rows)

Confirmed via `grep -rl "gorm.io/gorm\|\*gorm.DB"` returning empty across the
whole module for each, and no `entity.go` found by the Step 1 enumeration:
**atlas-invites, atlas-kites, atlas-login, atlas-messages, atlas-messengers,
atlas-monster-death, atlas-monsters, atlas-parties, atlas-portals**. Nothing
to classify for FR-8.1 — these services carry no Postgres tables at all.

### Services in this third with no Postgres persistence (no rows)

Confirmed via `grep -rl "gorm.io/gorm"` returning empty across the whole
module for each, and no `entity.go` found by the Step 1 enumeration:
**atlas-query-aggregator, atlas-rates, atlas-reactors, atlas-renders,
atlas-rps, atlas-summons, atlas-transports, atlas-world**. Nothing to
classify for FR-8.1 — these services carry no Postgres tables at all.
`atlas-saga-orchestrator` was checked against this same grep and does NOT
belong on this list — `saga/entity.go` and `saga/store.go` both import
`gorm.io/gorm`, so it gets a row above instead.

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

### Findings (part 2 of 3)

Five `UNSCOPED` rows in this third (covering six tables — one row groups
`frederick_items`/`frederick_mesos`, which share a cleanup task), all from the
same architectural pattern —
a periodic background task that runs cross-tenant `WithoutTenantFilter`
discovery/cleanup queries with no `WHERE tenant_id` of their own, motivated by
the fact that these tickers run once per service instance with no per-tenant
event context to scope by (the same constraint noted for buff/skill tickers in
project memory):

- **Bulk delete filtered only by a non-tenant predicate, no per-row tenant
  re-derivation** — genuinely unscoped, blocking: `atlas-merchant`
  `frederick_items`/`frederick_mesos` (`frederick/task.go:31`,
  `administrator.go:112-120,142-150`) and `atlas-mts` `wish_entries`
  (`wish/administrator.go:139-141`, invoked from `task/periodic.go:186` under
  the same unscoped context as the listing sweep).
- **Cross-tenant discovery `SELECT` followed by a per-row write addressed by
  the row's own unique id/derived tenant** — the mutation itself cannot land
  on the wrong tenant's row, but the discovery read crosses tenant boundaries
  within one process: `atlas-merchant` `frederick_notifications`
  (`notification_task.go:36-41,60-68`), `atlas-merchant` `shops`
  (`shop/task.go:29-52`), `atlas-mts` `listings` (`task/periodic.go:106`,
  `listing/administrator.go:54-65`).

All five are marked `UNSCOPED` per this audit's verdict definition (`at least
one query path does not filter`), not softened. Whether the second group's
per-row compensation is sufficient disposition for FR-9.x is a policy call for
Task 15's `tenant-scope-guard` allowlist, not this audit's to make — the
evidence and the distinction between the two failure shapes is recorded in
each affected row's Notes above.

No `FORCES-ISOLATED` or `TRANSITIVE` rows in this third — none of the 27
services' entities were found to require isolation escalation or route
exclusively through a `SCOPED` parent with no `TenantId` of their own; every
entity in this third carries its own `TenantId`/`TenantID` column. No
`CONTROL` rows either — every service in this third is data-plane. Nine
services in this third (`atlas-invites, atlas-kites, atlas-login,
atlas-messages, atlas-messengers, atlas-monster-death, atlas-monsters,
atlas-parties, atlas-portals`) have no Postgres persistence at all — see the
"no rows" section above.

### Findings (part 3 of 3)

Three `UNSCOPED` rows in this third, all the same "cross-tenant discovery
`SELECT` at boot/on a ticker, followed by a per-row write addressed by the
row's own unique id with the tenant reconstructed per row" shape as the
second group in part 2 — none is a bulk delete filtered only by a
non-tenant predicate:

- `atlas-saga-orchestrator` `sagas` — `GetAllActive`/`GetTimedOut`
  (`saga/store.go:228-236,239-250`), both `database.WithoutTenantFilter`,
  feeding `recoverSagas`/`reapTimedOutSagas` (`main.go:202-236,267-294`),
  which reconstruct `tenant.Model` per row before any write.
- `atlas-trades` `trade_escrow_items`/`trade_escrow_mesos` — `AllItems`/
  `AllMesos` (`escrow/provider.go:166-190`), unfiltered `db.Find`, called only
  from `ReconcileEscrow` (`trade/settlement.go:1300-1319`), whose own comment
  states it "runs with NO tenant in context and restores each row's own
  tenant."
- `atlas-trades` `trade_settlements`/`trade_settlement_sides`/
  `trade_settlement_items` — `allUnresolved` (`settlement/provider.go:61-80`),
  unfiltered `db.Find`, called only from `settlement.Unresolved`
  (`settlement/processor.go:73-81`), whose caller restores each row's tenant
  via `Model.Tenant()`.

A fourth candidate, `atlas-trades` `AllMesoStakes` (`escrow/provider.go:156-164`),
has the identical unfiltered shape but is called only from
`escrow/migration_test.go` — no production caller exists. It is exported,
dead, and unscoped; not counted as a live finding, but flagged here so a
future caller does not wire it up assuming it is already tenant-safe.

One entity, `atlas-quest` `quest_medal_maps` (`medal.Entity`,
`quest/medal/entity.go`), does not fit the five-verdict taxonomy at all: the
brief expected it to confirm as `TRANSITIVE` through the tenant-scoped
`quest_statuses` parent, but at source the table is never migrated
(`medal.Migration` is not in `main.go:52`'s `SetMigrations` call) and has no
provider/administrator anywhere in the service — there is no live query path
to classify, TRANSITIVE or otherwise. See that row's Notes; this needs the
controller's attention rather than a forced verdict.

Two entities were confirmed exactly as the brief pre-identified:
`atlas-tenants` `tenant.Entity` is `CONTROL` (no tenant-scoping column
exists or could exist — the row *is* the tenant registry, same as
`atlas-configurations`' `tenants` table); no medal-shaped TRANSITIVE row
exists in this third given the finding above.

No `FORCES-ISOLATED` rows in this third. Eight services in this third
(`atlas-query-aggregator, atlas-rates, atlas-reactors, atlas-renders,
atlas-rps, atlas-summons, atlas-transports, atlas-world`) have no Postgres
persistence at all — see the "no rows" section above.

### Counts (part 3 of 3, mechanically derived)

Total row count in the document (cumulative across Tasks 1–3, one running
table):

```
$ grep -c '^| atlas-' docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md
95
```

Per-service row count for the 9 services in this third that carry rows:

```
$ grep '^| atlas-' docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md | awk -F'|' '{print $2}' | sed 's/^ *//;s/ *$//' | sort | uniq -c | grep -E '^\s*[0-9]+ atlas-(query-aggregator|quest|rankings|rates|reactor-actions|reactors|renders|reward-pools|rps|saga-orchestrator|skills|storage|summons|tenants|trades|transports|world)$'
      3 atlas-quest
      2 atlas-rankings
      1 atlas-reactor-actions
      3 atlas-reward-pools
      1 atlas-saga-orchestrator
      2 atlas-skills
      2 atlas-storage
      2 atlas-tenants
      3 atlas-trades
```

That is 19 rows (3+2+1+3+1+2+2+2+3) across 9 services with persistence. The
`atlas-trades` count of 3 rows covers 6 tables total (the first row groups
4 escrow tables — `trade_escrow_items`/`mesos`/`meso_stakes`/`meso_refunds` —
matching part 2's `frederick_items`/`frederick_mesos` grouping convention;
the other two rows are ledger and settlement, 3 tables each collapsed to one
row per package as done throughout this document).

8 services in this third carry no Postgres persistence — explicit count:

```
$ printf '%s\n' atlas-query-aggregator atlas-rates atlas-reactors atlas-renders atlas-rps atlas-summons atlas-transports atlas-world | wc -l
8
```

9 (with rows) + 8 (no persistence) = 17, matching the count of services this
task's `### Files` section lists:

```
$ diff <(printf '%s\n' query-aggregator quest rankings rates reactor-actions reactors renders reward-pools rps saga-orchestrator skills storage summons tenants trades transports world | sort) \
       <(printf '%s\n' quest rankings reactor-actions reward-pools saga-orchestrator skills storage tenants trades query-aggregator rates reactors renders rps summons transports world | sort)
(no output — identical sets; every service named in the brief's ### Files got either a row group or a no-persistence line, none silently omitted)
```

## 2. Non-Postgres deployment-scoped resources (FR-8.6)

Every resource whose isolation currently depends on deployment identity, with
a disposition: **scope it**, or **forces isolated mode**.

| Resource | Where | Current isolation | Disposition |
|---|---|---|---|
| Redis key prefix | `libs/atlas-redis/keys.go:15` | `ATLAS_ENV` package-level var | Scoped: data plane → tenant-scoped API (Tasks 4–8); prefix stays load-bearing for isolated mode only |
| Kafka topic names | `overlays/pr/scripts/gen-topic-config.sh` | `-<ATLAS_ENV>` suffix | Scoped: sparse consumes unsuffixed topics + ownership gate (Task 25) |
| Postgres DB names | `overlays/pr/scripts/gen-db-name-suffix.sh` | `<db>-<ATLAS_ENV>` | Scoped: sparse shares `main`'s databases (D1) |
| Consumer group ids | `libs/atlas-kafka/consumergroup/resolver.go` | runtime-resolved | Already correct; no change |
| Object id allocation | `libs/atlas-object-id` | Redis keys `<prefix>:oid:<tenantId>:next` / `:free` (`allocator.go:104-110`) | Scoped: the key already incorporates `tenant.Model.Id()` directly (`counterKey`/`freeKey`, `allocator.go:104-110`) — allocation is tenant-scoped independent of environment. The `<prefix>` component is the same `atlasredis.KeyPrefix()` (`ATLAS_ENV`-derived) as the Redis key prefix row above and follows that row's disposition, not a separate one. |
| Outbox advisory lock | `libs/atlas-outbox/lock.go` | single constant key per DB | Deliberately global; now serialises drainers across environments — throughput coupling, not correctness (design §8.4) |
| MinIO canonical objects | `services/atlas-pr-bootstrap/scripts/reconcile-minio.sh` | Objects keyed `tenants/<tenantId>/...` (or `shared/`, operator-gated) — already tenant-scoped, not environment-scoped (`services/atlas-data/atlas.com/data/wzinput/scope.go:21-33`, `.../tenantpurge/purge.go:46`, `.../minioreconcile/reconcile.go:86`) | Scoped: the object *keys* need no change — they were never environment-scoped. What is environment-shaped today is the reconcile **script's discovery mechanism**: `reconcile-minio.sh:24-46` builds its keep-list by enumerating tenants per-namespace across every `atlas-pr-*` + `atlas-main` k8s namespace. Under sparse, most per-PR namespaces disappear, so this discovery step should instead query the `atlas-tenants` `tenant.Entity` registry (CONTROL, environment-scoped per Task 10/11) for the live-tenant set of a given environment, rather than enumerating namespaces. Flagged here so the Task 50 implementer does not have to re-derive this — the object model is already correct, only the script's tenant-discovery source needs to change. |
| Login/channel ports + advertised IP | `tools/gen-lb-ports.sh`, `services/atlas-pr-bootstrap/scripts/version-ports.sh` | per-namespace LoadBalancer | Scoped in Task 46 |

## 3. Second-mechanism sweep: tenant-less call contexts

§1 classified every Postgres access path by one test: does it call
`database.WithoutTenantFilter`? Task 3's review found that test incomplete —
the fleet-wide GORM callback also fails open when a query is reached through a
context that carries **no tenant at all** (no `WithoutTenantFilter` needed;
the callback just silently no-ops), and separately when a query builder never
receives `.WithContext(ctx)` at all. This section re-sweeps §1's 27
Postgres-having services from thirds 1–2, plus a shape check of third 3's 9,
for both mechanisms.

**Fix-round-1 correction.** The first pass of this section reported zero new
findings. That was wrong: it missed two live `atlas-ban` bulk deletes that
explicitly call `database.WithoutTenantFilter` with no tenant predicate
(should have been caught even by Tasks 1–3's original test — task.go files
were simply never in the files Task 1 read for that service), and two live
`atlas-marriages` cross-tenant discovery reads that never call `.WithContext`
at all. Both classes are recorded below, along with the corrected method that
catches the `atlas-marriages` shape and the completeness table that should
have shipped in the first pass.

### Step 1: Confirmed at source

`libs/atlas-database/tenant_scope.go` (`tenantQueryCallback`, lines 54-81;
`tenantCreateCallback`, lines 83-118):

- **No tenant in context:** `tenant.FromContext(ctx)()` (line 69) returns an
  error when the context carries no tenant. The callback's response to that
  error is `l.Debugf("No tenant in context for query on %s, skipping tenant
  filter.", ...)` followed by a bare `return` (lines 70-73) — **no WHERE
  clause is added, no error is raised, no `db.Error` is set.** The query
  proceeds exactly as built. Same shape in `tenantCreateCallback` (lines
  98-101): on `tenant.FromContext` error it just `return`s, so `TenantId` is
  left at whatever the struct literal set (typically zero) rather than being
  injected.
- **Query built on a `*gorm.DB` never given `.WithContext`:** the callback
  reads `db.Statement.Context` (line 60), which GORM defaults to
  `context.Background()` when `.WithContext` was never called on that
  statement chain. That context carries no tenant, so this collapses into the
  same "no tenant in context" branch above — same silent no-op, same
  no-error.
- **Logging:** only a `Debugf` in the query/update/delete path (line 71); the
  create path (`tenantCreateCallback`) does not even log on this branch. No
  `Warn`/`Error`, no metric, no propagated `db.Error`. A tenant-less query
  against a `tenant_id`-bearing table is invisible at runtime unless someone
  is tailing debug logs.

Premise confirmed as stated in the brief.

### Step 2–3: Sweep and trace — corrected method and results

**Method used (per call site, not per service).** The fix-round review's root
cause is exactly right: an aggregate "does this service have `WithContext`
anywhere" test cannot see one unwrapped call site coexisting with correctly
wrapped ones in the same service. The corrected method operates at the call
site:

1. **Struct-held `db`+`ctx` field scan.** Long-lived structs (schedulers,
   tickers, tasks, registries) that hold both a `ctx context.Context` field
   and a `db *gorm.DB` field are exactly the shape where one method can use
   `ctx` and forget to combine it with `db` while a sibling method does it
   correctly. Enumerated every struct in scope with both field types
   (`grep -E '^\s*ctx\s+context\.Context\s*$'` and
   `^\s*db\s+\*gorm\.DB\s*$'` co-occurring in one file), narrowed to the
   boot/ticker-shaped ones (`*task.go`, `*_task.go`,
   `scheduler/*.go`) since request-scoped `processor.go` structs are already
   covered by the REST-boilerplate trace in bucket 1 below. For every such
   struct, grepped every call site of a GORM verb reachable from its `db`
   field — `grep -nE '\.db\.(Model|Where|Find|First|Create|Save|Delete|
   Updates|Update|Pluck|Count|Exec|Raw)\('` — and checked, at that literal
   call site, whether `.WithContext(` appears on the same chain. Also ran the
   same regex against `.DB.` (capitalized field name) and confirmed `tx`
   (the other field name used for `*gorm.DB` in scope) is only ever a
   short-lived transaction closure parameter, never a long-lived struct
   field paired with a separately-held `ctx`, in every case but one
   (`atlas-mts/saga/outbox_emitter.go`), which is constructed fresh inside an
   already-scoped transaction and is not boot/ticker-reachable.
2. **`WithoutTenantFilter` fleet grep**, since it is a stronger, more direct
   signal than `context.Background()` when present — a call site that
   explicitly opts out of the callback and then applies no filter of its own
   is unscoped by construction, regardless of how the tenant-less context
   arrived (literal `context.Background()`, `rt.Context()`, or any other
   runtime-root context).
3. **`context.Background()`/`context.TODO()` sweep**, tracing every hit to
   its query (or confirming it doesn't reach one) — this pass is exhaustive
   this time; see the completeness table below.
4. **Full `NewTicker`/`time.Tick`/`cron` sweep** over thirds 1–2, tracing
   every hit found — this catches boot-time schedulers built on `rt.Context()`
   that the literal `context.Background()` grep cannot see (this is exactly
   how the `atlas-marriages` schedulers evaded the first pass: both are
   constructed with `rt.Context()`, never the literal string
   `context.Background()`).

**Finding 1 — `atlas-ban`, `bans` and `login_history` (bulk delete, no
per-row tenant discrimination — blocking).** The `WithoutTenantFilter` grep
(method 2) found two hits Task 1 never traced:

- `services/atlas-ban/atlas.com/ban/ban/task.go:28-36`
  (`ExpiredBanCleanup.Run`) — `noTenantCtx :=
  database.WithoutTenantFilter(t.ctx)` then `t.db.WithContext(noTenantCtx).
  Where("permanent = ? AND expires_at <= ?", false, now).Delete(&Entity{})`.
  No tenant predicate anywhere in the WHERE. Wired live at boot:
  `main.go:94` — `tasks.Register(l, rt.Context())(ban.NewExpiredBanCleanup(l,
  rt.Context(), db, 5*time.Minute))`. Table: `bans` (§1 row now `UNSCOPED`).
- `services/atlas-ban/atlas.com/ban/history/task.go:28-36`
  (`HistoryPurge.Run`) — identical shape: `noTenantCtx :=
  database.WithoutTenantFilter(t.ctx)` then `t.db.WithContext(noTenantCtx).
  Where("created_at < ?", cutoff).Delete(&Entity{})`. Wired live at boot:
  `main.go:97` — `tasks.Register(l, rt.Context())(history.NewHistoryPurge(l,
  rt.Context(), db, 24*time.Hour))`. Table: `login_history` (§1 row now
  `UNSCOPED`).

Both explicitly opt out of the automatic callback and apply no compensating
filter — a single tick deletes expired bans/purges login history across
every tenant in one statement. Same blocking shape as `atlas-merchant`
`frederick_items`/`frederick_mesos` and `atlas-mts` `wish_entries` in §1's
part-2 findings.

**Finding 2 — `atlas-marriages`, `proposals` and `ceremonies` (cross-tenant
discovery read, per-row tenant re-derivation, PK-addressed write).** The
struct-field scan (method 1) found two hits the first pass's per-service
aggregate check could not see, because `atlas-marriages`' `provider.go` uses
`WithContext` throughout and only these two scheduler reads don't:

- `services/atlas-marriages/atlas.com/marriages/scheduler/
  proposal_expiry.go:112-119` (`getTenantsWithProposals`) —
  `s.db.Model(&marriage.ProposalEntity{}).Where("status = ?",
  marriage.ProposalStatusPending).Distinct("tenant_id").Pluck("tenant_id",
  &tenantIds)`, called directly on the scheduler's `db` field with no
  `.WithContext` anywhere on the chain. Full path: `main.go:58` constructs
  `NewProposalExpiryScheduler(l, rt.Context(), db)` (tenant-less
  runtime-root context) → `main.go:59` `.Start()` → `routine.Go(s.log,
  s.ctx, s.run)` → `run()`'s `time.NewTicker(5*time.Minute)` loop →
  `processExpiredProposals()` → `getTenantsWithProposals()` (the unscoped
  read, table `proposals`) → for each returned tenant id,
  `processExpiredProposalsForTenant` reconstructs `tenantCtx :=
  tenant.WithContext(s.ctx, tenantModel)` and builds a tenant-scoped
  `marriage.NewProcessor` before the compensating write
  (`proposal_expiry.go:132-136`). The read crosses every tenant; the write
  does not.
- `services/atlas-marriages/atlas.com/marriages/scheduler/
  ceremony_timeout.go:108-115` (`getTenantsWithActiveCeremonies`) — identical
  shape: `s.db.Model(&marriage.CeremonyEntity{}).Where("status = ?",
  marriage.CeremonyStatusActive).Distinct("tenant_id").Pluck("tenant_id",
  &tenantIds)`, no `.WithContext`. Full path: `main.go:62` constructs
  `NewCeremonyTimeoutScheduler(l, rt.Context(), db)` → `main.go:63`
  `.Start()` → 1-minute ticker → `processActiveCeremonies()` →
  `getTenantsWithActiveCeremonies()` (the unscoped read, table `ceremonies`)
  → `processActiveCeremoniesForTenant` reconstructs `tenantCtx` before the
  compensating write (`ceremony_timeout.go:128-132`). Same shape as
  `proposals`.

Both schedulers correctly use `s.ctx` for `retry.Try` and correctly
reconstruct `tenantCtx` before every write — the gap is narrowly the two
`getTenantsWith*` discovery reads. This is the identical "cross-tenant
discovery `SELECT`, per-row tenant reconstruction, PK-addressed write" shape
already recorded for `atlas-mts` `listings`, `atlas-merchant`
`frederick_notifications`/`shops`, and `atlas-saga-orchestrator` `sagas`.

**Full `NewTicker`/`time.Tick`/`cron` sweep (method 4), thirds 1–2:**

```
$ grep -lnE 'time\.NewTicker|time\.Tick\(|"github.com/robfig/cron|cron\.New' <27-service file list>
services/atlas-cashshop/.../reservation/cache.go
services/atlas-data/.../runtime/rest/watchdog.go
services/atlas-data/.../runtime/ingest/heartbeat.go
services/atlas-marriages/.../scheduler/proposal_expiry.go   (Finding 2)
services/atlas-marriages/.../scheduler/ceremony_timeout.go  (Finding 2)
services/atlas-mts/.../task/periodic.go                     (already §1, part 2)
services/atlas-party-quests/.../main.go
```
Traced each non-already-documented hit:
- `atlas-cashshop` `reservation/cache.go:107` — no `gorm`/`db.`/`WithContext`
  in the file; Redis-only reservation-TTL eviction. Not a finding.
- `atlas-data` `runtime/rest/watchdog.go:38` and `runtime/ingest/
  heartbeat.go:48` — neither file contains a `gorm`/`db.` hit; Kubernetes
  liveness/ingest-heartbeat pings, no Postgres reachable. Not a finding.
- `atlas-party-quests` `main.go:93-118` — `AutoMigrate` only (schema
  migration, not a live query); the 1-second ticker at line 93 iterates
  `tenants` (fetched once at boot from the tenant registry) and builds
  `ctx := tenant.WithContext(rt.Context(), t)` (line 102) **before**
  constructing `instance.NewProcessor(l, ctx, db)` — tenant reconstructed
  per tenant before every DB-touching call. Same defended pattern as the
  `account`/`character`/`guilds`/`mounts`/`pets` tickers already traced
  below. Not a finding.

The four already-traced `task.go` tickers from the first pass (`atlas-account`,
`atlas-character` session, `atlas-guilds`, `atlas-mounts`, `atlas-pets`) were
re-confirmed unaffected by this method — none holds `db` and `ctx` as
separate fields *and* calls a GORM verb directly on the `db` field; all
reconstruct `tctx := tenant.WithContext(sctx, entry.Tenant())` before ever
touching `t.db`.

**`context.Background()`/`context.TODO()` completeness table (method 3, all
33 non-test hits, corrected from the first pass's overclaim of exhaustive
tracing).** Format: file — disposition.

| File | Disposition |
|---|---|
| `atlas-account/rest/handler.go` (×2) | REST boilerplate (bucket 1) |
| `atlas-account/account/processor.go:153,165,173,359` incl. `Teardown` | In-memory `Registry`, no gorm; `context.Background()` only seeds a span or wraps a per-tenant `tenant.WithContext` call. Not a finding. |
| `atlas-account/account/task.go:29` | Ticker (bucket 2) — traced, not a finding |
| `atlas-ban/rest/handler.go` | REST boilerplate (bucket 1) |
| `atlas-cashshop/.../reservation/cache.go:29` | Redis-only cache, no gorm. Not a finding |
| `atlas-character/session/task.go:33` | Ticker (bucket 2) — traced, not a finding |
| `atlas-character/rest/handler.go` | REST boilerplate (bucket 1) |
| `atlas-data/runtime/rest/jobs.go:136,140` | Kubernetes ConfigMap/image-discovery API calls, never Postgres. Not a finding |
| `atlas-data/baseline/restore.go:108-112` | `cleanupAfterFailure`, raw `Exec` with explicit `WHERE tenant_id = ?` bound to `target` param — scoped at SQL text. Not a finding |
| `atlas-drop-information/rest/handler.go` | REST boilerplate (bucket 1) |
| `atlas-fame/rest/handler.go` | REST boilerplate (bucket 1) |
| `atlas-guilds/guild/task.go:31` | Ticker (bucket 2) — traced, not a finding |
| `atlas-guilds/coordinator/registry.go:105` | In-memory registry, no gorm. Not a finding |
| `atlas-inventory/compartment/lock_registry.go:33,40,44` | Redis distributed lock, no gorm. Not a finding |
| `atlas-inventory/compartment/reservation_registry.go:74,95,107` | Redis reservation registry, no gorm. Not a finding |
| `atlas-map-actions/rest/handler.go` | REST boilerplate (bucket 1) |
| `atlas-maps/tasks/respawn.go:34` | No gorm/db in file. Not a finding |
| `atlas-maps/tasks/weather.go:30` | No gorm/db in file. Not a finding |
| `atlas-maps/tasks/mist_tick.go:320,343` | No gorm/db in file. Not a finding |
| `atlas-maps/map/timer/processor.go:120` | No gorm/db in file. Not a finding |
| `atlas-mounts/rest/handler.go` | REST boilerplate (bucket 1) |
| `atlas-mounts/mount/task.go:58` | Ticker (bucket 2) — traced, not a finding |
| `atlas-mts/rest/handler.go` | REST boilerplate (bucket 1) |
| `atlas-mts/test/database.go` | Test-only helper (`func(t *testing.T, ...)`), no production caller. Not a finding |
| `atlas-notes/rest/handler.go` | REST boilerplate (bucket 1) |
| `atlas-npc-conversations/rest/handler.go` | REST boilerplate (bucket 1) |
| `atlas-npc-shops/shops/cache.go:71` | Redis consumable cache; `GetDistinctTenants(db *gorm.DB)` in the same file is a **separate** dead-code shape-2 candidate, not this line — see below |
| `atlas-npc-shops/rest/handler.go` | REST boilerplate (bucket 1) |
| `atlas-npc-shops/test/database.go` | Test-only helper. Not a finding |
| `atlas-party-quests/rest/handler.go` | REST boilerplate (bucket 1) |
| `atlas-pets/rest/handler.go` | REST boilerplate (bucket 1) |
| `atlas-pets/pet/task.go:29` | Ticker (bucket 2) — traced, not a finding |
| `atlas-portal-actions/rest/handler.go` | REST boilerplate (bucket 1) |

**Bucket 1 (REST handler boilerplate)** — `server.RetrieveSpan(l, handlerName,
context.Background(), func(sctx) { server.ParseTenant(fl, sctx, func(tctx) {
handler(...ctx: tctx) }) })`, present in every one of the 27 services. Traced
`server.ParseTenant` to source (`libs/atlas-rest/server/handler.go:34-59`):
it parses tenant id/region/version from HTTP headers and calls `next` with a
tenant-bearing `tctx` — `context.Background()` here never reaches a query
un-augmented.

**Bucket 2 (periodic tickers that reconstruct tenant per-entity before any DB
call)** — `atlas-account` `Timeout.Run` (`task.go:29`), `atlas-character`
`session.Timeout.Run` (`session/task.go:33`), `atlas-guilds` `Timeout.Run`
(`task.go:31`), `atlas-mounts` `TirednessTask.Run` (`mount/task.go:58`),
`atlas-pets` `Timeout.Run` (`pet/task.go:29`). All five enumerate an
in-memory/Redis registry off the bare context, then build `tctx :=
tenant.WithContext(sctx, entry.Tenant())` before constructing any
DB-touching processor.

**Shape-2 check (query builders that never receive `.WithContext`, per call
site, corrected method).** The struct-field-scan method above is the
corrected shape-2 test; it found the two `atlas-marriages` hits and nothing
else in scope. One **dead-code** shape-2 instance (different shape — an
exported function with a bare `db` parameter and zero callers, not a
struct-held field) remains as recorded in the first pass: `atlas-npc-shops`
`GetDistinctTenants(db *gorm.DB)`
(`services/atlas-npc-shops/atlas.com/npc/shops/cache.go:75-79`) —
`db.Model(&Entity{}).Distinct("tenant_id").Pluck(...)`, never calls
`.WithContext`, and `grep -rln "GetDistinctTenants" services/ libs/` finds
only its own definition — no caller anywhere in the repo, production or
test. Not a live finding; flagged so a future caller doesn't wire it up
assuming it's already tenant-safe.

**Third-3 re-check.** Re-ran the `WithoutTenantFilter` fleet grep and the
struct-field scan over the 9 third-3 services; no new hits beyond the two
already-documented `atlas-trades` rows (`escrow.AllItems`/`AllMesos`,
`settlement.allUnresolved`) and the `atlas-saga-orchestrator` `sagas` row,
all re-traced end to end and confirmed exactly as recorded in §1. One
additional production `context.Background()` hit,
`atlas-trades/trade/settlement.go:157` (`c.ctx =
tenant.WithContext(context.Background(), p.t)`), wraps a tenant before use —
bucket-2 shape, not a finding.

### Step 4: §1 regrades

Four `SCOPED` rows are regraded to `UNSCOPED`, each with the original
evidence cell preserved and a Notes pointer added:

- `atlas-ban` `bans` (row: `bans (ban.Entity)`) — Finding 1.
- `atlas-ban` `login_history` (row: `login_history (history.Entity)`) —
  Finding 1.
- `atlas-marriages` `proposals` (row: `proposals (ProposalEntity)`) —
  Finding 2.
- `atlas-marriages` `ceremonies` (row: `ceremonies (CeremonyEntity)`) —
  Finding 2.

No other `SCOPED` row in §1 is contradicted by this sweep.

### Step 6: Counts (mechanically derived)

```
$ grep -c '^| atlas-' docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md
95

$ grep '^| atlas-' docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md | awk -F'|' '{print $5}' | sed 's/^ *//;s/ *$//' | sort | uniq -c
      4 CONTROL
      1 N/A — orphaned
     78 SCOPED
     12 UNSCOPED
```
95 total, unchanged (§3 added zero new rows, only regraded four existing
ones: 82→78 SCOPED, 8→12 UNSCOPED).

New `UNSCOPED` findings from this fix round: **4** (2 `atlas-ban`, 2
`atlas-marriages`). Fleet-wide `UNSCOPED` total after this sweep: **12** (8
from §1 parts 1–3 + 4 from this section). §1 rows regraded: **4**.

### Findings

This sweep's premise (the fleet-wide callback fails open against a
tenant-less context and against a `*gorm.DB` that never receives
`.WithContext`) is confirmed at source (Step 1). The corrected per-call-site
method — a struct-field scan for long-lived `ctx`+`db` pairs, a fleet-wide
`WithoutTenantFilter` grep, an exhaustive `context.Background()`/`TODO()`
trace, and a full `NewTicker`/`cron` sweep — found four additional live
`UNSCOPED` rows beyond the two `atlas-trades` instances already recorded in
§1: `atlas-ban` `bans`/`login_history` (bulk delete, no per-row tenant
discrimination — blocking, same shape as `atlas-merchant`/`atlas-mts`) and
`atlas-marriages` `proposals`/`ceremonies` (cross-tenant discovery read,
per-row tenant re-derivation, PK-addressed write — same shape as
`atlas-mts`/`atlas-merchant`/`atlas-saga-orchestrator`). The `atlas-ban`
pair is notable because it explicitly calls `database.WithoutTenantFilter`
— it should have been catchable by Tasks 1–3's original test, and was missed
because `task.go` was never in the file set Task 1 read for that service,
not because of the second-mechanism gap this task exists to close. The
`atlas-marriages` pair is the mechanism this task was created for: a
struct-held `db` field queried directly with no `.WithContext`, invisible to
a per-service aggregate check because the same file's other query paths use
`WithContext` correctly.

The only remaining non-live artifact is the dead, uncalled shape-2 function
`atlas-npc-shops.GetDistinctTenants`, flagged for the record rather than
counted as a finding.

**Input for Task 15's `tenant-scope-guard`:** two mechanical checks would
have caught all four of this round's findings:

1. A lint over call sites that pass `database.WithoutTenantFilter(ctx)` (or a
   context built from `context.Background()`/`context.TODO()`/a runtime-root
   context) into a query builder for a `tenant_id`-bearing table, flagging
   any such site whose *only* tenant restoration happens after the read
   unless allowlisted with a recorded reason (mirroring `atlas-rankings`'
   `pruneBefore` fail-closed pattern) — catches the `atlas-marriages` shape
   and would also have caught `atlas-ban` since it uses the same
   `WithoutTenantFilter` call the lint keys on.
2. A struct-shape lint: any struct with both a `ctx context.Context` field
   and a `db`/`DB` `*gorm.DB` field, where a GORM verb is called directly on
   the `db` field without `.WithContext` on the same chain — this is the
   exact shape that evaded the first pass's per-service aggregate check and
   would generalize to catch a third instance of the same bug class before
   it reaches production.
3. (Carried from the first pass, cheaper but narrower) flag any exported
   `*gorm.DB` query-builder function with zero non-test callers in its
   module — would have caught `AllMesoStakes` and `GetDistinctTenants`.

### Fix rounds 2–3: `status.go:148` disposition and fleet-wide reconciliation

Fix round 2 covered the `status.go:148` disposition and a
`WithoutTenantFilter` reconciliation rooted at `services/` only (a scoping
error in that round's brief, not its execution). Fix round 3 widens the
root to the whole repo (`services/ libs/`), which surfaces one additional
live finding — `libs/atlas-database/idempotency.go:143` — folded into the
same reconciliation table below rather than kept as a separate pass.

**`status.go:148` — `SCOPED`.**
`services/atlas-data/atlas.com/data/data/status.go:148` — `handleGetStatus`
builds `ctx := database.WithoutTenantFilter(d.Context())` and passes it to
`queryStatus`, which filters `documents` by `Where("tenant_id = ?",
tenantId)`. `tenantId` comes from `resolveStatusTenantId`: either the
request tenant's own id (default / `scope=tenant`) or the version-scoped
canonical shared-content id (`canonical.TenantId(region, major, minor)`,
`scope=shared`, gated behind an `X-Atlas-Operator` header check). The bypass
exists so an operator's `scope=shared` read is not AND-ed with their own
`tenant_id` and made to miss the canonical rows. Every generated query still
carries an explicit `tenant_id = ?` bound to one resolved id — same pattern
as §1 rows 75–81 (`reactor_search_index`, `npc_search_index`,
`npc_spawn_index`, `monster_search_index`, `monster_spawn_index`,
`documents`, `map_search_index`).

**`libs/atlas-database/idempotency.go:143` — `UNSCOPED` (bulk write, no
per-row tenant discrimination — blocking).** First-class row, not folded
into an existing one: this is a shared-library ticker
(`StartIdempotencySweeper`), not a service-owned one, wired live from three
services.

- Service: `libs/atlas-database (shared)` — consuming services:
  `atlas-cashshop` (`services/atlas-cashshop/atlas.com/cashshop/main.go:66`),
  `atlas-storage` (`services/atlas-storage/atlas.com/storage/main.go:72`),
  `atlas-inventory` (`services/atlas-inventory/atlas.com/inventory/main.go:61`)
  — all three call `database.StartIdempotencySweeper(l, rt.Context(), db,
  database.DefaultIdempotencyRetention, database.DefaultIdempotencySweep)`
  at boot, each passing its own runtime-root `rt.Context()`.
- Table: `idempotency_keys` (`IdempotencyEntity`,
  `libs/atlas-database/idempotency.go:31-40`, composite PK
  `(tenant_id, key)`).
- Read at source: `StartIdempotencySweeper`
  (`libs/atlas-database/idempotency.go:141-157`) starts a `routine.Go` loop
  holding a `time.NewTicker(interval)`; on every tick it calls
  `SweepIdempotency(sweepCtx, db, retention)` where `sweepCtx :=
  WithoutTenantFilter(rctx)` (line 143) is computed once, outside the loop,
  from the boot-time `rctx` — never refreshed per tenant. `SweepIdempotency`
  (lines 100-105) runs `db.WithContext(ctx).Where("created_at < ?",
  cutoff).Delete(&IdempotencyEntity{}).Error` — the only predicate is the
  retention cutoff; there is no `tenant_id` anywhere in the query, and no
  per-row tenant re-derivation before or after the delete (unlike the
  `atlas-merchant` shop/`atlas-mts` listings discovery-read shape, which
  reconstructs a `tenant.Model` per row before its compensating write). The
  code comment confirms this is deliberate:
  `// The sweep is cross-tenant, so it runs with tenant filtering disabled`
  (line 139).
- Shape: **bulk write, no per-row tenant discrimination — blocking**, same
  class as `atlas-ban` `bans`/`login_history` and `atlas-mts`
  `wish_entries`: one tick deletes every tenant's expired idempotency claim
  rows past retention in a single statement, across all three consuming
  services. After per-PR DB isolation is removed, one environment's hourly
  sweep (`DefaultIdempotencySweep = time.Hour`,
  `libs/atlas-database/idempotency.go:134`) deletes another environment's
  idempotency rows once they're older than the 7-day retention
  (`DefaultIdempotencyRetention`, line 133).

**Fleet-wide `WithoutTenantFilter` reconciliation, widened root.** The
prior fix round's grep was scoped to `services/` only — an error in that
round's brief, not in its execution. Re-run with the root widened to the
whole repo:

```
$ grep -rn "WithoutTenantFilter" --include="*.go" . | wc -l
42
```

38 under `services/` (unchanged from fix round 2) plus 4 under `libs/`:

```
$ grep -rn "WithoutTenantFilter" --include="*.go" libs/
libs/atlas-database/tenant_scope_test.go:106:	ctx := WithoutTenantFilter(tenantContext(tid1))
libs/atlas-database/idempotency.go:143:		sweepCtx := WithoutTenantFilter(rctx)
libs/atlas-database/tenant_scope.go:20:// WithoutTenantFilter returns a context that disables automatic tenant filtering.
libs/atlas-database/tenant_scope.go:22:func WithoutTenantFilter(ctx context.Context) context.Context {
```

Every one of the 42 hits, reconciled:

| `file:line` | Accounted for in | Verdict |
|---|---|---|
| `services/atlas-ban/atlas.com/ban/history/task.go:32` | §3 Finding 1 → §1 row 57 (`login_history`) | UNSCOPED |
| `services/atlas-ban/atlas.com/ban/ban/task.go:30` | §3 Finding 1 → §1 row 55 (`bans`) | UNSCOPED |
| `services/atlas-data/atlas.com/data/data/status.go:148` | This entry (new, above) | SCOPED |
| `services/atlas-data/atlas.com/data/reactor/search_test.go:48` | Test setup for §1 row 75 (`reactor_search_index`) | Test path |
| `services/atlas-data/atlas.com/data/searchindex/searchindex.go:95` | Comment (doc-comment on `ResolvePartitionTenantId`), not a call site | N/A — comment |
| `services/atlas-data/atlas.com/data/searchindex/searchindex.go:99` | `ResolvePartitionTenantId` — feeds §1 rows 75/76/79/81; explicit `Where("tenant_id = ?", t.Id())` bound to the request tenant before resolving the partition | SCOPED |
| `services/atlas-data/atlas.com/data/searchindex/searchindex.go:138` | §1 row 75 Notes (cited directly) — `Search` | SCOPED |
| `services/atlas-data/atlas.com/data/searchindex/searchindex.go:235` | §1 row 75 Notes (cited directly) — `SearchWithFilter` | SCOPED |
| `services/atlas-data/atlas.com/data/searchindex/searchindex.go:264` | §1 row 75 Notes (cited directly) — `Count` | SCOPED |
| `services/atlas-data/atlas.com/data/searchindex/searchindex.go:304` | §1 row 75 Notes (cited directly) — `CountWithFilter` | SCOPED |
| `services/atlas-data/atlas.com/data/searchindex/searchindex_test.go:61` | Test setup for §1 rows 75/76/79/81 | Test path |
| `services/atlas-data/atlas.com/data/searchindex/searchindex_test.go:324` | Test setup for §1 rows 75/76/79/81 | Test path |
| `services/atlas-data/atlas.com/data/npc/search_test.go:53` | Test setup for §1 row 76 (`npc_search_index`) | Test path |
| `services/atlas-data/atlas.com/data/npc/spawn_index.go:40` | §1 row 77 (`npc_spawn_index`) Notes (cited directly) — `SpawnMapsFor` | SCOPED |
| `services/atlas-data/atlas.com/data/monster/search_test.go:48` | Test setup for §1 row 79 (`monster_search_index`) | Test path |
| `services/atlas-data/atlas.com/data/monster/spawn_index.go:46` | §1 row 80 (`monster_spawn_index`) Notes (cited directly) — `SpawnMapsFor` | SCOPED |
| `services/atlas-data/atlas.com/data/map/search_test.go:27` | Test setup for §1 row 81 (`map_search_index`) | Test path |
| `services/atlas-data/atlas.com/data/item/string_search_test.go:92` | Test setup — `item` string search/registry has no §1 row of its own (indexes are Redis/in-memory per §1's item-search disposition); no Postgres `WithoutTenantFilter` production call site in this package | Test path |
| `services/atlas-data/atlas.com/data/item/string_registry_test.go:57` | Test setup, same package as above | Test path |
| `services/atlas-merchant/atlas.com/merchant/frederick/task.go:31` | §1 row 102 (`frederick_items`, `frederick_mesos`) | UNSCOPED |
| `services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go:36` | §1 row 103 (`frederick_notifications`) | UNSCOPED |
| `services/atlas-merchant/atlas.com/merchant/shop/task.go:29` | §1 row 109 (`shops`) | UNSCOPED |
| `services/atlas-mts/atlas.com/mts/listing/administrator.go:53` | Comment, not a call site — describes the sweep context consumed by §1 row 114 (`listings`) | N/A — comment |
| `services/atlas-mts/atlas.com/mts/listing/provider.go:207` | Comment, not a call site — same sweep, §1 row 114 | N/A — comment |
| `services/atlas-mts/atlas.com/mts/listing/processor.go:586` | Comment, not a call site — same sweep, §1 row 114 | N/A — comment |
| `services/atlas-mts/atlas.com/mts/wish/administrator.go:136` | Comment, not a call site — describes the sweep context consumed by §1 row 115 (`wish_entries`) | N/A — comment |
| `services/atlas-mts/atlas.com/mts/task/periodic.go:99` | Comment, not a call site — describes `sweepCtx` below | N/A — comment |
| `services/atlas-mts/atlas.com/mts/task/periodic.go:106` | §1 row 114 (`listings`) and row 115 (`wish_entries`) — `sweepCtx` feeds both `CountExpiredActive`/`GetExpiredActive` and `DeleteExpiredWanted` | UNSCOPED |
| `services/atlas-mts/atlas.com/mts/task/periodic.go:123` | Comment, not a call site | N/A — comment |
| `services/atlas-mts/atlas.com/mts/task/periodic.go:199` | Comment, not a call site | N/A — comment |
| `services/atlas-mts/atlas.com/mts/task/periodic_test.go:40` | Comment in test file | N/A — comment |
| `services/atlas-mts/atlas.com/mts/task/periodic_test.go:62` | Test setup for §1 rows 114/115 | Test path |
| `services/atlas-mts/atlas.com/mts/task/periodic_test.go:71` | Test setup for §1 rows 114/115 | Test path |
| `services/atlas-mts/atlas.com/mts/task/periodic_test.go:82` | Test setup for §1 rows 114/115 | Test path |
| `services/atlas-mts/atlas.com/mts/task/periodic_test.go:200` | Test setup for §1 rows 114/115 | Test path |
| `services/atlas-mts/atlas.com/mts/testsupport/resource.go:416` | Comment, not a call site — describes test-helper tenant scoping | N/A — comment |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/store.go:230` | §1 row 139 (`sagas`) — `GetAllActive` | UNSCOPED |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/store.go:241` | §1 row 139 (`sagas`) — `GetTimedOut` | UNSCOPED |
| `libs/atlas-database/tenant_scope_test.go:106` | Test setup for `libs/atlas-database`'s own callback tests | Test path |
| `libs/atlas-database/idempotency.go:143` | This entry (new, above) — `idempotency_keys`, `libs/atlas-database (shared)` | UNSCOPED |
| `libs/atlas-database/tenant_scope.go:20` | Comment (doc-comment on `WithoutTenantFilter`'s own definition), not a call site | N/A — comment |
| `libs/atlas-database/tenant_scope.go:22` | The definition of `WithoutTenantFilter` itself, not a call site | N/A — definition |

Result: every one of the 42 fleet-wide `WithoutTenantFilter` hits (38 under
`services/`, 4 under `libs/`) resolves to an already-recorded §1 row
(UNSCOPED or explicitly-filtered SCOPED), the new `status.go` SCOPED entry,
the new `idempotency_keys` UNSCOPED entry, a doc comment/definition site
that is not a call site, or a test-setup path. **One new UNSCOPED finding**
(`libs/atlas-database` `idempotency_keys`, consumed by `atlas-cashshop`,
`atlas-storage`, `atlas-inventory`). Fleet-wide `UNSCOPED` total after this
round: **13** (12 from fix round 1 + 1 from `idempotency_keys`).

**Other `libs/`-resident background loops reaching a query.** Per the
brief's routing question: swept `libs/` for every `time.NewTicker`/
`time.Tick`/cron construction site and checked each for Postgres
reachability —

```
$ grep -rlnE 'time\.NewTicker|time\.Tick\(|"github.com/robfig/cron|cron\.New' libs/ --include="*.go"
libs/atlas-redis/coalesced.go
libs/atlas-redis/tenant_coalesced.go
libs/atlas-outbox/drainer.go
libs/atlas-database/idempotency.go
libs/atlas-lock/leader.go
```

- `libs/atlas-redis/coalesced.go`, `libs/atlas-redis/tenant_coalesced.go`,
  `libs/atlas-lock/leader.go` — none imports `gorm.io/gorm` or references a
  `*gorm.DB`/`*sql.DB` (`grep -l "gorm\|\*sql\.DB"` over the three returns
  nothing). Redis/distributed-lock only; never reaches Postgres. Not a
  finding.
- `libs/atlas-outbox/drainer.go` — **is** a `libs/`-resident ticker reaching
  Postgres: `Run` (line 81, `time.NewTicker(d.cfg.pollInterval)`) and
  `runSweeper` (line 200, `time.NewTicker(d.cfg.sweeperInterval)`) both
  drive `d.db.WithContext(ctx)` queries against `outbox_entries`
  (`SweepOnce`, `libs/atlas-outbox/drainer.go:185-188`, `Where("sent_at IS
  NOT NULL AND sent_at < ?", cutoff).Delete(...)`, no tenant predicate).
  Read `libs/atlas-outbox/entity.go:9-21`: `Entity` (`outbox_entries`) has
  **no `TenantId` field at all** — `ID`, `Topic`, `MessageKey`,
  `MessageValue`, `Headers`, `EnqueuedAt`, `SentAt`, `Attempts`,
  `LastError`, nothing else. The fleet callback
  (`libs/atlas-database/tenant_scope.go`) keys off a `TenantId` field via
  reflection; a struct that never declares one is not a
  tenant-partitioned table in the first place, so this audit's method (does
  a query bypass or evade the automatic per-row tenant filter) does not
  apply to it — there is no tenant filter to evade. It is one relay queue
  shared by every tenant in a service's database by design (each row's
  tenant identity lives inside `MessageValue`'s serialized envelope, not as
  a queryable column), wired per-service (`outboxlib.NewDrainer(l, db,
  publisher, ...)`, e.g. `services/atlas-cashshop/atlas.com/cashshop/main.go:76`
  and 26 other services). **Not a tenant-scope finding under this audit's
  method** — but flagged here because it is still a `libs/`-resident ticker
  outside a `services/`-rooted ticker inventory, and its cross-*environment*
  exposure surface (if per-PR Postgres isolation is removed, whether
  `outbox_entries` rows from different environments can collide or be
  drained by the wrong environment's Kafka publisher) is a different axis
  than tenant scoping and was not evaluated here.

**Answer to the routing question:** one other `libs/`-resident background
loop of this shape exists — `libs/atlas-outbox/drainer.go` — but it does not
belong to the `idempotency_keys` finding class: it operates on a table with
no `tenant_id` column, so there is no tenant filter for it to omit. It is
still outside a `services/`-rooted ticker inventory and worth its own
routing decision on the cross-environment (not cross-tenant) axis. No third
`libs/`-resident ticker reaching Postgres was found; `atlas-redis` and
`atlas-lock`'s tickers never reach Postgres at all.

## 4. UNSCOPED dispositions

Task 4A. Derives its row inventory mechanically from §1 and §3:

```
$ grep -n "| UNSCOPED |" docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md
```

(§1 rows 55, 57, 100, 101, 102, 103, 109, 114, 115, 139, 146, 148 — twelve
rows — plus the §3 first-class finding `libs/atlas-database/idempotency.go:143`,
recorded at line 834 of the reconciliation table but never given a §1 row
because it is shared-library code with no single owning service. Thirteen
rows total.)

Every row was traced to its query builder and read for a stated intent
(source comment, `git log`/blame, or domain semantics) before disposition.
No code was changed anywhere in this task — every row below classified
`INTENDED-GLOBAL` on direct evidence; none reached `TENANT-DEFECT` or
`UNDECIDED`. Per Step 4 of the brief: **Step 4 (fix every `TENANT-DEFECT`
row) is skipped — there are none.** A task that correctly changes no code is
the outcome here, not a shortfall.

### 4.1 Row-by-row disposition

| # | Service | Table(s) | Entry point (file:line) | Shape | Disposition | Evidence |
|---|---|---|---|---|---|---|
| 1 | atlas-ban | `bans` | `ban/task.go:28` `ExpiredBanCleanup.Run`, wired `main.go:94` (5 min) | bulk-write | `INTENDED-GLOBAL` | Source comment, `ban/task.go:26-28`: *"Run deletes all expired temporary bans across all tenants. This intentionally bypasses the processor layer and operates without tenant context, performing a single global sweep rather than iterating per-tenant."* |
| 2 | atlas-ban | `login_history` | `ban/history/task.go:28` `HistoryPurge.Run`, wired `main.go:97` (24 h) | bulk-write | `INTENDED-GLOBAL` | Source comment, `ban/history/task.go:27-29`: *"Run deletes all login history records older than RetentionDays across all tenants. This intentionally bypasses the processor layer and operates without tenant context, performing a single global sweep rather than iterating per-tenant."* |
| 3 | atlas-marriages | `proposals` | `scheduler/proposal_expiry.go:112` `getTenantsWithProposals`, wired `main.go:58-59` (5 min ticker) | discovery-read | `INTENDED-GLOBAL` | Structural design, not incidental: `processExpiredProposals` (`proposal_expiry.go:~100-107`) explicitly enumerates `tenantIds` then loops `for _, tenantId := range tenantIds { s.processExpiredProposalsForTenant(tenantId) }` (comment `// Process each tenant`), and `processExpiredProposalsForTenant` reconstructs a fresh `tenant.Model`/`tenantCtx` per id (`proposal_expiry.go:126-129`) before touching `marriage.NewProcessor`. The discovery query exists specifically to drive that per-tenant loop — this is the documented pattern of a background scheduler that discovers-then-iterates every tenant in one deployment, not an accidentally-unfiltered lookup. |
| 4 | atlas-marriages | `ceremonies` | `scheduler/ceremony_timeout.go:108` `getTenantsWithActiveCeremonies`, wired `main.go:62-63` (1 min ticker) | discovery-read | `INTENDED-GLOBAL` | Identical structure to row 3: `processActiveCeremonies` enumerates `tenantIds` (comment `// Process each tenant`, `ceremony_timeout.go:~102`) then loops per tenant id, reconstructing `tenantCtx` in `processActiveCeremoniesForTenant` before any write. |
| 5 | atlas-merchant | `frederick_items`, `frederick_mesos` | `frederick/task.go:29` `CleanupTask.Run` (`cleanupExpiredItems`/`cleanupExpiredMesos`, `administrator.go:111,141`), wired via `NewCleanupTask` | bulk-write | `INTENDED-GLOBAL` | No direct source comment on `CleanupTask.Run` or `cleanupExpiredItems`/`cleanupExpiredMesos` themselves — checked `git log -p --follow -- frederick/administrator.go` and `frederick/task.go`; commit subjects are generic refactors ("Refactor atlas-merchant to align with backend developer guidelines", "Rename atlas-* modules...", lint pass) with no discussion of tenant scope. Disposition rests on domain semantics: this is a custody-expiry reaper (deletes Frederick-held items/mesos past a 100-day retention cutoff) — structurally identical (`WithoutTenantFilter` + `Where(cutoff).Delete`, no per-row re-derivation) to rows 1/2/9/13 above/below, which carry explicit "deliberate global sweep" comments for the same reaper shape in the same codebase. Recorded honestly: evidence here is domain-semantics + architectural-consistency, not a direct comment — weaker than rows 1/2/9/13 but not absent. |
| 6 | atlas-merchant | `frederick_notifications` | `frederick/notification_task.go:36` `NotificationTask.Run` (`Find`, `notification_task.go:39-41`), wired via `NewNotificationTask` | discovery-read | `INTENDED-GLOBAL` | No comment on the query itself, but the loop body (`notification_task.go:58-79`) reconstructs `ten, _ := tenant.Create(n.TenantId, n.TenantRegion, n.TenantMajor, n.TenantMinor)` and `tctx := tenant.WithContext(t.ctx, ten)` per row before the Kafka emit, and every subsequent write (`advanceNotification`/`deleteNotification`) is addressed by the row's own `Id` — the identical discover-then-restore-per-row shape as rows 3/4/7/8/10/11/12, all of which are `INTENDED-GLOBAL`. Domain semantics: a due-notification sweep across a single deployment's tenants is exactly the class of job this architecture assigns to a background ticker. |
| 7 | atlas-merchant | `shops` | `shop/task.go:29` `ExpirationTask.Run` (`getExpired`, `shop/provider.go:135-144`), wired via `NewExpirationTask` | discovery-read | `INTENDED-GLOBAL` | Source comment, `shop/task.go:30-32`: *"Single source of truth for the expiry predicate (incl. Draft — a hired merchant abandoned during setup must still be reaped at its 24h expiry); run cross-tenant so one task instance sweeps every tenant."* |
| 8 | atlas-mts | `listings` | `task/periodic.go:106` `Sweep` (`CountExpiredActive`/`GetExpiredActive`, `listing/administrator.go:54-58,63-65`), wired via the periodic sweep loop | discovery-read | `INTENDED-GLOBAL` | Source comment, `task/periodic.go:91-101` (`Sweep`'s own doc comment): *"it discovers active auction listings whose ends_at has passed (across ALL tenants)... Tenant context reconstruction (THE crux): the listings table stores only a tenant_id uuid — no region/version — so a full tenant.Model cannot be rebuilt... Instead the sweep runs cross-tenant... the expire transition takes the holding's tenant_id from the listing ROW itself... Each listing is addressed by its unique surrogate uuid."* |
| 9 | atlas-mts | `wish_entries` | `task/periodic.go:106` sweep context feeds `wish/administrator.go:139` `DeleteExpiredWanted` | bulk-write | `INTENDED-GLOBAL` | Source comment, `wish/administrator.go:132-136`: *"The periodic sweep calls this under a WithoutTenantFilter handle so it removes expired want-ads across every tenant."* |
| 10 | atlas-saga-orchestrator | `sagas` | `saga/store.go:228` `GetAllActive` (recovery) and `saga/store.go:239` `GetTimedOut` (reaper), wired `main.go:182` `recoverSagas` and `main.go:238-` `startReaper`/`reapTimedOutSagas` | discovery-read | `INTENDED-GLOBAL` | `GetAllActive` carries its own doc comment, `saga/store.go:227`: *"GetAllActive returns all active and compensating sagas across all tenants (for startup recovery)"*. `GetTimedOut` (`store.go:238`) has no separate comment but is the identical shape in the same store, called from `reapTimedOutSagas` (`main.go:267`), and both callers reconstruct `t, _ := tenant.Create(e.TenantId, ...)` / `ctx := tenant.WithContext(...)` per returned row (`main.go:222-223`, `main.go:~275`) before `processor.Step`. Boot-time recovery and a timeout reaper both plausibly need a whole-deployment view by domain semantics (there is no tenant to scope a startup scan to). |
| 11 | atlas-trades | `trade_escrow_items`, `trade_escrow_mesos` (+ `trade_escrow_meso_stakes`/`trade_escrow_meso_refunds` via the same entity family) | `escrow/provider.go:175` `AllItems`, `:190` `AllMesos`, called only from `trade/settlement.go:1311` `ReconcileEscrow` | discovery-read | `INTENDED-GLOBAL` | Source comment on `AllItems`, `escrow/provider.go:167-173`: *"Deliberately un-scoped: startup reconciliation runs before any request has supplied a tenant, and each row carries the tenant quad needed to restore one... This and AllMesos are the ONLY queries in the package that cross tenants, and both are reachable only from the boot path and the retry ticker."* Caller comment, `settlement.go:1300-1309` (`ReconcileEscrow`): *"It runs with NO tenant in context and restores each row's own tenant, the same shape as Reconcile. A failure for one tenant does not stop the others."* `AllMesoStakes` (`escrow/provider.go:156-158`) carries the identical comment and shape but **no production caller** (`grep -rn "AllMesoStakes"` finds only `escrow/migration_test.go`) — dead code, not a live gap; not a candidate for any fix regardless of disposition. |
| 12 | atlas-trades | `trade_settlements`, `trade_settlement_sides`, `trade_settlement_items` | `settlement/provider.go:72` `allUnresolved`, exposed via `settlement/processor.go:79` `Unresolved`, boot-path caller | discovery-read | `INTENDED-GLOBAL` | Source comment, `settlement/processor.go:73-77`: *"Unresolved returns every unfinished settlement across every tenant, oldest first, for startup reconciliation. It is a package function rather than a Processor method because there is no tenant to construct a Processor with at boot: each returned Model carries the tenant it belongs to, and the caller restores it per row via Model.Tenant()."* |
| 13 | `libs/atlas-database` (shared; consumed live by atlas-cashshop, atlas-storage, atlas-inventory) | `idempotency_keys` | `libs/atlas-database/idempotency.go:143` `sweepCtx := WithoutTenantFilter(rctx)` inside `StartIdempotencySweeper`, wired at each consumer's `main.go` boot (`atlas-cashshop/main.go:66`, `atlas-storage/main.go:72`, `atlas-inventory/main.go:61`) | bulk-write | `INTENDED-GLOBAL` | Source comment, `idempotency.go:139`: *"// The sweep is cross-tenant, so it runs with tenant filtering disabled"*. Shared-library code raises the bar for a fix (any change alters behavior for all three consumers at once per the brief), but the comment is unambiguous and the row does not reach that bar in the first place — no fix is being considered. |

### 4.2 Summary

- 13/13 rows: `INTENDED-GLOBAL`.
- 0/13 rows: `TENANT-DEFECT`. **Step 4 (fix every `TENANT-DEFECT` row) does
  not apply — no code was changed by this task.**
- 0/13 rows: `UNDECIDED`. Row 5 (`frederick_items`/`frederick_mesos`) has the
  weakest direct evidence of the thirteen — no source comment or informative
  commit message was found on the query builders themselves — but domain
  semantics (a custody-expiry reaper, structurally identical to three other
  rows that do carry explicit "deliberate global sweep" comments for the same
  shape) is evidence the brief names as legitimate, and no evidence points
  the other way. Recorded as `INTENDED-GLOBAL` with that caveat rather than
  escalated as `UNDECIDED`, since "genuinely ambiguous" does not describe a
  row where every available signal points one direction and none points the
  other.

The pattern across all thirteen: every row is a background/periodic loop
whose cross-tenant read exists to discover work items (or, for the four
bulk-write rows, to delete by a time predicate) once per deployment, not once
per tenant. Under today's per-environment-owns-its-database model this is
correct. It becomes wrong only once environments stop owning separate
databases (decision D1) — at that point a sparse PR environment's sweep
reaches `main`'s rows, which is an environment-isolation defect, not a
tenant-scope defect. That conversion is explicitly out of scope for this
task and belongs to Tasks 41–42.

### 4.3 Task 42 hand-off list

Task 42 builds `ticker-dispositions.md` from ticker files under `services/`
and converts each `INTENDED-GLOBAL` entry point to per-environment iteration.
The following entry points are the precise conversion targets, one per row
above (dead-code paths and comment-only citations excluded):

1. `services/atlas-ban/atlas.com/ban/ban/task.go:28` — `ExpiredBanCleanup.Run`
2. `services/atlas-ban/atlas.com/ban/history/task.go:28` — `HistoryPurge.Run`
3. `services/atlas-marriages/atlas.com/marriages/scheduler/proposal_expiry.go:112` — `getTenantsWithProposals` (and its caller `processExpiredProposals`)
4. `services/atlas-marriages/atlas.com/marriages/scheduler/ceremony_timeout.go:108` — `getTenantsWithActiveCeremonies` (and its caller `processActiveCeremonies`)
5. `services/atlas-merchant/atlas.com/merchant/frederick/task.go:29` — `CleanupTask.Run`
6. `services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go:36` — `NotificationTask.Run`
7. `services/atlas-merchant/atlas.com/merchant/shop/task.go:29` — `ExpirationTask.Run`
8. `services/atlas-mts/atlas.com/mts/task/periodic.go:106` — `Sweep` (feeds both `listings` and `wish_entries`)
9. `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go:182` (`recoverSagas`, via `store.go:228` `GetAllActive`) and `main.go:238-` (`startReaper`/`reapTimedOutSagas`, via `store.go:239` `GetTimedOut`)
10. `services/atlas-trades/atlas.com/trades/trade/settlement.go:1311` — `ReconcileEscrow` (via `escrow/provider.go:175,190` `AllItems`/`AllMesos`)
11. `services/atlas-trades/atlas.com/trades/settlement/processor.go:79` — `Unresolved` (via `settlement/provider.go:72` `allUnresolved`)

**Outside Task 42's `services/`-rooted ticker inventory — carry explicitly,
per the controller's brief context (verified facts, not re-derived here):**

12. `libs/atlas-database/idempotency.go:141` — `StartIdempotencySweeper`
    (feeds `idempotency_keys`, row 13 above). Shared-library ticker consumed
    live by `atlas-cashshop`, `atlas-storage`, `atlas-inventory` — a
    conversion here changes behavior for all three consumers in one change,
    unlike the eleven service-owned entry points above.
13. `libs/atlas-outbox/drainer.go:81` (`Run`) and `:200` (`runSweeper`),
    draining `outbox_entries` — **not a tenant-scope finding** (per §3: the
    `Entity` has no `TenantId` column at all, so there is no tenant filter to
    evade; the fleet callback never applies to it). It is nonetheless a
    `libs/`-resident ticker outside the `services/`-rooted inventory, and its
    cross-*environment* exposure (whether one environment's drainer can pick
    up another environment's `outbox_entries` rows once per-PR Postgres
    isolation is removed) is a live open question on a different axis than
    tenant scoping, unevaluated by this audit. Consumed by 27 services via
    `outboxlib.NewDrainer`.
