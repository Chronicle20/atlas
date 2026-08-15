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
**atlas-invites, atlas-login, atlas-messages, atlas-messengers,
atlas-monster-death, atlas-monsters, atlas-parties, atlas-portals**. Nothing
to classify for FR-8.1 — these services carry no Postgres tables at all.

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
`CONTROL` rows either — every service in this third is data-plane. Eight
services in this third (`atlas-invites, atlas-login, atlas-messages,
atlas-messengers, atlas-monster-death, atlas-monsters, atlas-parties,
atlas-portals`) have no Postgres persistence at all — see the "no rows"
section above.
