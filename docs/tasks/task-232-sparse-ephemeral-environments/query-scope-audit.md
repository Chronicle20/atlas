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
| atlas-marriages | proposals (`ProposalEntity`) | Data | SCOPED | `services/atlas-marriages/atlas.com/marriages/marriage/entity.go:84` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-marriages/atlas.com/marriages/marriage/provider.go:15,35,79,91,123,150,439`; writes at `services/atlas-marriages/atlas.com/marriages/marriage/administrator.go:14,47` | No raw SQL. |
| atlas-marriages | ceremonies (`CeremonyEntity`) | Data | SCOPED | `services/atlas-marriages/atlas.com/marriages/marriage/entity.go:145` (TenantId); `libs/atlas-database/tenant_scope.go:75-79`; reads at `services/atlas-marriages/atlas.com/marriages/marriage/provider.go:244,269,357,383,409`; writes at `services/atlas-marriages/atlas.com/marriages/marriage/administrator.go:110,151` | No raw SQL. |
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
Postgres-having services from thirds 1–2, plus a shape-2 spot-check of third
3's 9, for both mechanisms.

### Step 1: Confirmed at source

`libs/atlas-database/tenant_scope.go` (`tenantQueryCallback`, lines 54-81;
`tenantCreateCallback`, lines 83-118):

- **No tenant in context:** `tenant.FromContext(ctx)()` (line 69) returns an
  error when the context carries no tenant. The callback's response to that
  error is `l.Debugf("No tenant in context for query on %s, skipping tenant
  filter.", ...)` followed by a bare `return` (lines 70-73) — **no WHERE
  clause is added, no error is raised, no `db.Error` is set.** The query
  proceeds exactly as built, at whatever scope its own explicit filters (if
  any) provide. Same shape in `tenantCreateCallback` (lines 98-101): on
  `tenant.FromContext` error it just `return`s, so `TenantId` is left at
  whatever the struct literal set (typically zero) rather than being injected.
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

Premise confirmed as stated in the brief. Proceeding with the sweep.

### Step 2–3: Sweep and trace

Method: for each of the 27 services in thirds 1–2 (the 19 named across
§1's rows for thirds 1–2, plus the 8 services already confirmed to have no
Postgres persistence were excluded — persistence-bearing services swept:
`atlas-account, atlas-ban, atlas-buddies, atlas-cashshop, atlas-character,
atlas-configurations, atlas-data, atlas-drop-information` (third 1) and
`atlas-fame, atlas-families, atlas-guilds, atlas-inventory, atlas-keys,
atlas-map-actions, atlas-maps, atlas-marriages, atlas-merchant,
atlas-mini-games, atlas-monster-book, atlas-mounts, atlas-mts, atlas-notes,
atlas-npc-conversations, atlas-npc-shops, atlas-party-quests, atlas-pets,
atlas-portal-actions` (third 2, 19 services) — 27 total):

```
$ xargs -a <(for s in atlas-account atlas-ban atlas-buddies atlas-cashshop atlas-character \
  atlas-configurations atlas-data atlas-drop-information atlas-fame atlas-families atlas-guilds \
  atlas-inventory atlas-keys atlas-map-actions atlas-maps atlas-marriages atlas-merchant \
  atlas-mini-games atlas-monster-book atlas-mounts atlas-mts atlas-notes atlas-npc-conversations \
  atlas-npc-shops atlas-party-quests atlas-pets atlas-portal-actions; do echo services/$s; done) \
  grep -rln --include='*.go' -e "context.Background()" -e "context.TODO()" | grep -v _test.go
```
returned 33 non-test files. Every hit falls into one of four buckets, each
traced to its query (or confirmed not to reach one):

1. **REST handler boilerplate** (`*/rest/handler.go`, present in every one of
   the 27 services) — `server.RetrieveSpan(l, handlerName, context.Background(),
   func(sl, sctx) { return server.ParseTenant(fl, sctx, func(tl, tctx) {
   handler(&HandlerDependency{..., ctx: tctx}) }) })`. Traced
   `server.ParseTenant` to source (`libs/atlas-rest/server/handler.go:34-59`):
   it reads the tenant ID/region/version from HTTP headers and calls `next`
   with a tenant-bearing `tctx`, which is what every handler's `db.WithContext`
   actually uses. `context.Background()` here is only the seed the tracer/
   header-parsing wrapper starts from — it never reaches a query un-augmented.
   Not a finding, for all 27 services.
2. **Periodic ticker tasks that reconstruct tenant per-entity before any DB
   call** — `atlas-account` `account.Timeout.Run` (`task.go:29`),
   `atlas-character` `session.Timeout.Run` (`session/task.go:33`),
   `atlas-guilds` `guild.Timeout.Run` (`task.go:31`), `atlas-mounts`
   `TirednessTask.Run` (`mount/task.go:58`), `atlas-pets` `Timeout.Run`
   (`pet/task.go:29`). All five follow the identical shape: `sctx :=
   ...Start(context.Background(), ...)` is used only to enumerate an
   in-memory/Redis registry (`GetRegistry().GetAll/GetActive/GetExpired`,
   never `*gorm.DB`-backed), then for every entry `tctx :=
   tenant.WithContext(sctx, entry.Tenant())` is built **before** constructing
   any processor that touches `t.db`. The DB-touching processor
   (`NewProcessor(t.l, tctx, t.db)` / `applyTick(..., tctx, ...)`) always
   receives the per-row `tctx`, never the bare `sctx`. Not a finding for any
   of the five — same defended pattern noted in project memory for buff/skill
   tickers.
3. **`atlas-maps`' `respawn.go`/`weather.go`/`mist_tick.go`/
   `map/timer/processor.go`** — traced each `context.Background()` site; none
   of the four files contains a `gorm`/`*gorm.DB`/`db.` call at all (`grep -n
   "gorm\|db\.\|WithContext"` returns no query-builder hits). These tasks are
   entirely Redis-registry-driven; `atlas-maps`' two Postgres tables
   (`character_map_visits`, `character_locations`) are not reachable from
   them. Not a finding.
4. **`atlas-data`** — `baseline/restore.go:108-112`
   (`cleanupAfterFailure`) builds `ctx, cancel :=
   context.WithTimeout(context.Background(), cleanupTimeout)` then runs
   `db.WithContext(ctx).Exec("DELETE FROM "+t+" WHERE tenant_id = ?",
   target.String())` — the context carries no tenant, but the raw SQL's own
   `WHERE tenant_id = ?` is bound explicitly to the `target` parameter, so the
   delete is scoped at the SQL text itself, independent of the callback. Not
   a finding (same explicit-bind pattern as the search-index partition
   reads in §1). `runtime/rest/jobs.go:136,140` (`loadTemplateFromConfigMap`,
   `discoverControllerImage`) use `context.Background()` for Kubernetes API
   calls (`ConfigMap`/image discovery), never a Postgres query. Not a finding.

**Shape-2 check** (query builders that never receive `.WithContext`,
performed across all 27 third-1/2 services plus all 9 third-3 services with
Postgres persistence):

```
$ for d in services/atlas-<svc>...; do
    find "$d" -name '*.go' ! -name '*_test.go' | xargs grep -l "gorm.io/gorm" | wc -l   # gorm_files
    find "$d" -name '*.go' ! -name '*_test.go' | xargs grep -l '\.WithContext(' | wc -l # withcontext_files
  done
```
Every one of the 27 + 9 = 36 services with `gorm_files > 0` also has
`withcontext_files > 0` — no service's persistence layer is entirely
detached from context threading (full per-service counts are reproducible
with the command above; example: `atlas-mts gorm_files=38
withcontext_files=12`, `atlas-saga-orchestrator gorm_files=2
withcontext_files=3`). One **dead-code** shape-2 instance was found at the
function level: `atlas-npc-shops` `GetDistinctTenants(db *gorm.DB)`
(`services/atlas-npc-shops/atlas.com/npc/shops/cache.go:75-79`) —
`db.Model(&Entity{}).Distinct("tenant_id").Pluck("tenant_id", &tenantIds)`,
which never calls `.WithContext` and takes a bare `db` param, is deliberately
cross-tenant by intent (it exists to discover every tenant's distinct ids).
`grep -rln "GetDistinctTenants" services/ libs/` finds only its own
definition — **no production or test caller exists anywhere in the repo.**
Not a live finding (no reachable path), same disposition as `AllMesoStakes`
in §1's `atlas-trades` row — flagged here so a future caller does not wire it
up assuming it is already tenant-safe; if it is ever wired up it needs an
explicit `WithoutTenantFilter` + a documented reason, not ambient reliance on
whatever context happens to reach it.

**Third-3 re-check.** `atlas-trades`: `grep -rn "context.Background()\|
context.TODO()"` over non-test files finds exactly one production hit outside
the two already-documented rows —
`services/atlas-trades/atlas.com/trades/trade/settlement.go:157`
(`c.ctx = tenant.WithContext(context.Background(), p.t)`), which wraps a
tenant before use (not a finding, same bucket-2 shape). The two documented
`UNSCOPED` rows were re-traced end to end: `main.go:139-142` starts
`ReconcileAtBoot` in a `routine.Go(l, rt.Context(), ...)` goroutine off a
runtime-root context (no tenant), which calls `ReconcileEscrow` (confirmed at
`settlement.go:1280-1300` calling into `settlement.go:1311`) and, via
`ReconcileAtBoot` → `settlement.Unresolved` (`processor.go:73-81`), the
`allUnresolved` provider (`provider.go:61-80`) — both confirmed exactly as
recorded in §1. No new `atlas-trades` finding. `atlas-saga-orchestrator`
(`gorm_files=2, withcontext_files=3`) and the remaining 7 third-3 services
were shape-2 spot-checked above with no additional finding.

### Step 4: §1 regrades

No `SCOPED` row in §1 is contradicted by this sweep. Every tenant-less
`context.Background()`/`context.TODO()` call site that reaches a Postgres
query either resolves to a tenant before the query runs (bucket 2/4 above,
production pattern) or is the standard REST-boilerplate wrapper that never
reaches a query un-augmented (bucket 1). No table currently marked `SCOPED`
has an unguarded tenant-less path. **No rows regraded.**

### Step 6: Counts (mechanically derived)

```
$ grep -c '^| atlas-' docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md
95
```
(unchanged from §1 — this sweep added zero rows and regraded zero rows, per
Step 4 above).

Services swept for both mechanisms in this section: 27 (third-1 8 +
third-2 19, all with `### Files`-listed Postgres persistence) plus 9 from
third 3 re-checked for shape 2 and the two known findings = 36 services
total, all of the fleet's Postgres-having services covered by §1.

New `UNSCOPED` findings from this sweep: **0**. New dead-code shape-2
candidates flagged (non-live, not counted as findings): **1**
(`atlas-npc-shops` `GetDistinctTenants`). §1 rows regraded: **0**.

### Findings

This sweep's premise (the fleet-wide callback fails open against a
tenant-less context and against a `*gorm.DB` that never receives
`.WithContext`) is confirmed at source (Step 1). Applying both tests across
every Postgres-having service in thirds 1–2 plus a shape-2 check of third 3
found **no additional `UNSCOPED` rows beyond the two `atlas-trades` instances
already recorded in §1** (`escrow.AllItems`/`AllMesos` and
`settlement.allUnresolved`), which this sweep independently re-traced and
confirms exactly as documented. The two failure shapes identified in the
brief — bulk write with no per-row tenant discrimination, versus cross-tenant
discovery read with per-row tenant re-derivation and PK-addressed writes —
were both looked for; no new instance of either shape was found. The only
new artifact is a dead, uncalled shape-2 function
(`atlas-npc-shops.GetDistinctTenants`) that is unscoped by construction but
unreachable, flagged for the record rather than counted as a finding.

**Input for Task 15's `tenant-scope-guard`:** a mechanical check that would
have caught the two confirmed `atlas-trades` instances (and would catch a
regression of this shape) is a lint over call sites that pass
`database.WithoutTenantFilter(ctx)` (or a context built from
`context.Background()`/`context.TODO()`/a runtime-root context) into a query
builder for a `tenant_id`-bearing table, flagging any such call site whose
*only* per-row tenant restoration happens after the read (i.e., the read
itself is unscoped) unless the call site is on an explicit allowlist with a
recorded reason — mirroring how `atlas-rankings`' `pruneBefore` opts in to
failing closed by calling `tenant.FromContext` itself before the delete. A
narrower, cheaper version of the same check: flag any `*gorm.DB` query
builder function that is exported but has zero non-test callers in the
module (would have caught both `AllMesoStakes` and `GetDistinctTenants`
without needing call-graph tracing into `WithoutTenantFilter` usage at all).

