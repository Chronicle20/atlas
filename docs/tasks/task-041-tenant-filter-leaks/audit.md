# Tenant Filter Leak Audit — task-041

Version: v1
Created: 2026-05-17

## Methodology

Pipeline run from worktree root per plan §Task 3:

- `rg` over every `db.<verb>(`, `tx.<verb>(`, `database.Query(`, `database.SliceQuery(`, and `WithoutTenantFilter` site under `services/`, minus `_test.go` (raw output: 351 line-level hits, condensed to one row per logical call-site/function below).
- `rg` for `\bTenantId\b\s+uuid\.UUID` to inventory tenant-scoped entities.
- Per-file scan for `TenantId` / `TenantID` (case-insensitive) to detect tenant-less entity files under in-scope services.
- `rg` for `tenant.WithContext` and `context.Background()` / `context.TODO()` to assess context discipline.
- `rg` for `\.Raw(`, `\.Exec(`, `Preload("…")`, and `WithoutTenantFilter` to surface F4 / F8 / F10 candidates.

Out of scope per plan §7 / design §7: `atlas-ui`, `atlas-assets`, `atlas-data` (read-only WZ), `atlas-wz-extractor`, `atlas-pr-bootstrap`, `atlas-runtime-orchestrator`, `atlas-tenants`. Their call sites are excluded from the table below. `atlas-saga-orchestrator/saga/store.go` and `atlas-data/searchindex/searchindex.go` `WithoutTenantFilter` paths are referenced as PASS-CROSS-TENANT for context only.

Classification semantics (from plan §Task 3, design §4):

- **PASS-CB** — entity has `tenant_id` column; call site threads `WithContext(ctx)` with a tenant-carrying context; callback rewrites the SQL. No fix.
- **PASS-EXPLICIT** — call site adds its own `WHERE tenant_id = ?` predicate. Harmless duplicate filter when callback also fires (`TestDoubleWhereIsHarmless`). No fix.
- **PASS-CROSS-TENANT** — intentionally bypasses tenant scoping via `database.WithoutTenantFilter(ctx)`; cross-tenant by design with justification comment + scope boundary. No fix.
- **LEAK-F<n>** — fails the F<n> check from context.md §threat-model. Listed in §Leaks for Task 6.
- **LEAK-F6 (resolved)** — historic; Task 2 already hardened `tenantCreateCallback` to inject `tenant_id`. Listed for record only, no further fix.
- **UNCLEAR** — needs reviewer judgment, with `resolve:` question in Fix cell.

## Summary

| Class | Count | Notes |
|---|---:|---|
| PASS-CB | 278 | Default state. Provider/administrator funcs invoked from tenant-aware processors via `p.db.WithContext(p.ctx)` or `ExecuteTransaction(p.db.WithContext(p.ctx), …)`. |
| PASS-EXPLICIT | 11 | Sites with hand-written `WHERE tenant_id = ?` (atlas-monster-book card/collection, atlas-maps/character/location). |
| PASS-CROSS-TENANT | 7 | atlas-merchant tasks + helpers (5), atlas-saga-orchestrator recovery (2). atlas-data search-index sites are listed in §PASS-CROSS-TENANT but out of scope (not in call-site table). |
| PASS-MIGRATION | 4 | `Exec(CREATE INDEX …)` / `Exec(UPDATE …)` invoked only from `Migration(db *gorm.DB)` at startup — schema/backfill DDL, no per-request data exposure. atlas-families holds 5 `Exec` lines collapsed into 1 row. |
| PASS-GLOBAL | 13 | atlas-configurations entities are intentionally tenant-less because the service is global. atlas-quest/quest/medal is dormant (not enumerated as a call site). |
| LEAK-F2 | 2 | atlas-ban background cleanup tasks rely on F2 (missing tenant in context) instead of explicit `WithoutTenantFilter`. |
| LEAK-F8 | 5 | 1 row in atlas-buddies (`Preload("Buddies")`) + 4 rows in atlas-pets (`Preload("Excludes")` and the two `excludes` write paths). The 4 atlas-pets rows collapse to one migration fix per design §5; counted as 5 rows / 2 fixes. |
| Unclear | 0 | — |
| LEAK-F6 (resolved) | — | Hardened by Task 2 (`tenantCreateCallback` injection). No outstanding sites. |
| **Call-site rows (table)** | **320** | Matches `awk '/^\| atlas-/ {n++} END {print n}'` over the Call-sites section. |

Counts are logical call sites (one per provider/administrator function or per task method), not raw `rg` line hits. The raw enumeration produced 351 line hits; chained calls (`Where(...).Find(...)`, `Where(...).Delete(...)`) collapse into one row, and the families `Exec` migration block collapses 5 line hits into 1 row.

## Tenant-scoped services in scope (29)

| Service | Tenant-scoped entities | Notes |
|---|---|---|
| atlas-account | account | |
| atlas-ban | ban, history | Background cleanup tasks → LEAK-F2 candidates. |
| atlas-buddies | list | `buddy` child is tenant-less → LEAK-F8. |
| atlas-cashshop | wallet, wishlist, inventory/compartment, inventory/asset | |
| atlas-character | character, saved_location, session/history | |
| atlas-drop-information | continent/drop, monster/drop, reactor/drop | |
| atlas-fame | fame | |
| atlas-families | family | |
| atlas-gachapons | gachapon, global, item | |
| atlas-guilds | guild, member, title, character, thread, reply | |
| atlas-inventory | compartment, asset | |
| atlas-keys | key | |
| atlas-map-actions | script | Script field name is `TenantID`. |
| atlas-maps | visit, character/location | |
| atlas-marriages | marriage, proposal, ceremony | |
| atlas-merchant | shop, listing, message, frederick item/meso/notification | Background tasks → PASS-CROSS-TENANT. |
| atlas-monster-book | card, collection | PASS-EXPLICIT (hand-written `tenant_id = ?`). |
| atlas-notes | note | |
| atlas-npc-conversations | npc, quest, recipe | All `TenantID` columns. |
| atlas-npc-shops | shops, commodities | |
| atlas-party-quests | definition | |
| atlas-pets | pet | `exclude` child tenant-less → LEAK-F8. |
| atlas-portal-actions | script | |
| atlas-quest | quest, progress | `medal` child entity is dormant. |
| atlas-reactor-actions | script | |
| atlas-saga-orchestrator | saga | Recovery paths PASS-CROSS-TENANT. |
| atlas-skills | skill, macro | |
| atlas-storage | storage, asset | |
| atlas-configurations | (none — global service) | All entities tenant-less by design. |

`atlas-data` and `atlas-tenants` are out of scope; not enumerated here.

## Call-sites

One row per logical call site/function. File:line points to the representative line for the function.

### atlas-account

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-account | services/atlas-account/atlas.com/account/account/administrator.go:18 | create | W | PASS-CB | — |
| atlas-account | services/atlas-account/atlas.com/account/account/administrator.go:43 | deleteById | W | PASS-CB | — |
| atlas-account | services/atlas-account/atlas.com/account/account/provider.go:13 | byIdEntityProvider | R | PASS-CB | — |
| atlas-account | services/atlas-account/atlas.com/account/account/provider.go:24 | byNameEntityProvider | R | PASS-CB | — |
| atlas-account | services/atlas-account/atlas.com/account/account/provider.go:35 | allProvider | R | PASS-CB | — |

### atlas-ban

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-ban | services/atlas-ban/atlas.com/ban/ban/administrator.go:26 | create | W | PASS-CB | — |
| atlas-ban | services/atlas-ban/atlas.com/ban/ban/administrator.go:37 | deleteBan | W | PASS-CB | — |
| atlas-ban | services/atlas-ban/atlas.com/ban/ban/provider.go:14 | byIdProvider | R | PASS-CB | — |
| atlas-ban | services/atlas-ban/atlas.com/ban/ban/provider.go:25 | allProvider | R | PASS-CB | — |
| atlas-ban | services/atlas-ban/atlas.com/ban/ban/provider.go:36 | byTypeProvider | R | PASS-CB | — |
| atlas-ban | services/atlas-ban/atlas.com/ban/ban/provider.go:48 | activeIpBansProvider | R | PASS-CB | — |
| atlas-ban | services/atlas-ban/atlas.com/ban/ban/provider.go:60 | activeByValueProvider | R | PASS-CB | — |
| atlas-ban | services/atlas-ban/atlas.com/ban/ban/task.go:27 | ExpiredBanCleanup.Run | W | LEAK-F2 | done: bd7f6832a — added `ctx` field to `ExpiredBanCleanup`, wrapped delete with `t.db.WithContext(database.WithoutTenantFilter(t.ctx))`, constructor + main.go updated. |
| atlas-ban | services/atlas-ban/atlas.com/ban/history/administrator.go:22 | create | W | PASS-CB | — |
| atlas-ban | services/atlas-ban/atlas.com/ban/history/administrator.go:33 | purgeOlderThan | W | PASS-CB | — |
| atlas-ban | services/atlas-ban/atlas.com/ban/history/provider.go:13 | byAccountIdProvider | R | PASS-CB | — |
| atlas-ban | services/atlas-ban/atlas.com/ban/history/provider.go:24 | byIpProvider | R | PASS-CB | — |
| atlas-ban | services/atlas-ban/atlas.com/ban/history/provider.go:35 | byHwidProvider | R | PASS-CB | — |
| atlas-ban | services/atlas-ban/atlas.com/ban/history/task.go:29 | HistoryPurge.Run | W | LEAK-F2 | done: 4993da0a3 — added `ctx` field to `HistoryPurge`, wrapped delete with `t.db.WithContext(database.WithoutTenantFilter(t.ctx))`, constructor + main.go updated. |

### atlas-buddies

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-buddies | services/atlas-buddies/atlas.com/buddies/list/administrator.go:19 | create | W | PASS-CB | — |
| atlas-buddies | services/atlas-buddies/atlas.com/buddies/list/administrator.go:44 | addBuddy | W | PASS-CB | — |
| atlas-buddies | services/atlas-buddies/atlas.com/buddies/list/administrator.go:65 | deleteBuddy | W | PASS-CB | — |
| atlas-buddies | services/atlas-buddies/atlas.com/buddies/list/administrator.go:85 | updateBuddy (channel) | W | PASS-CB | — |
| atlas-buddies | services/atlas-buddies/atlas.com/buddies/list/administrator.go:109 | updateBuddy (shop flag) | W | PASS-CB | — |
| atlas-buddies | services/atlas-buddies/atlas.com/buddies/list/administrator.go:137 | deleteList | W | PASS-CB | — |
| atlas-buddies | services/atlas-buddies/atlas.com/buddies/list/administrator.go:177 | saveList | W | PASS-CB | — |
| atlas-buddies | services/atlas-buddies/atlas.com/buddies/list/provider.go:13 | byCharacterIdEntityProvider | R | LEAK-F8 | done: e775ff0be — added `TenantId uuid.UUID` (indexed) to `buddy.Entity` + idempotent backfill in `Migration` from `lists.tenant_id` via `list_id` FK. Preload now covered by callback. |

### atlas-cashshop

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/wallet/administrator.go:17 | create | W | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/wallet/administrator.go:38 | update | W | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/wallet/administrator.go:47 | deleteByAccountId | W | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/wallet/provider.go:13 | byAccountIdEntityProvider | R | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/wishlist/administrator.go:16 | create | W | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/wishlist/administrator.go:24 | deleteByItemId | W | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/wishlist/administrator.go:28 | clearByCharacterId | W | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/wishlist/provider.go:13 | byCharacterIdEntityProvider | R | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/compartment/administrator.go:19 | create | W | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/compartment/administrator.go:29 | deleteById | W | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/compartment/administrator.go:37 | save (lookup) | W | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/compartment/administrator.go:45 | save (write) | W | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/compartment/provider.go:16 | byIdEntityProvider | R | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/compartment/provider.go:28 | byAccountAndTypeEntityProvider | R | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/compartment/provider.go:40 | byAccountEntityProvider | R | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/administrator.go:45 | createAsset | W | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/administrator.go:76 | createAssetWithExpiration | W | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/administrator.go:84 | deleteById | W | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/provider.go:15 | byIdEntityProvider | R | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/provider.go:25 | byCompartmentIdEntityProvider | R | PASS-CB | — |
| atlas-cashshop | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/provider.go:35 | byCashIdEntityProvider | R | PASS-CB | — |

### atlas-character

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-character | services/atlas-character/atlas.com/character/character/administrator.go:44 | create | W | PASS-CB | — (TenantId is set explicitly on entity — defense-in-depth; harmless once F6 callback injection lands per Task 2). |
| atlas-character | services/atlas-character/atlas.com/character/character/administrator.go:52 | deleteById | W | PASS-CB | — |
| atlas-character | services/atlas-character/atlas.com/character/character/provider.go:13 | byIdEntityProvider | R | PASS-CB | — |
| atlas-character | services/atlas-character/atlas.com/character/character/provider.go:19 | forAccountInWorld | R | PASS-CB | — |
| atlas-character | services/atlas-character/atlas.com/character/character/provider.go:25 | forAccount | R | PASS-CB | — |
| atlas-character | services/atlas-character/atlas.com/character/character/provider.go:32 | byName | R | PASS-CB | — |
| atlas-character | services/atlas-character/atlas.com/character/character/provider.go:43 | all | R | PASS-CB | — |
| atlas-character | services/atlas-character/atlas.com/character/saved_location/administrator.go:31 | upsert (lookup) | W | PASS-EXPLICIT | — Caller's curried function takes `tenantId` and the query uses `character_id = ? AND location_type = ?` without `tenant_id`; the saved_location `Entity` has `TenantId` in a unique composite index (`idx_saved_location_lookup`). Callback rewrites to add `tenant_id = ?`. PASS-CB. |
| atlas-character | services/atlas-character/atlas.com/character/saved_location/administrator.go:39 | delete | W | PASS-CB | — |
| atlas-character | services/atlas-character/atlas.com/character/session/history/administrator.go:22 | create | W | PASS-CB | — |
| atlas-character | services/atlas-character/atlas.com/character/session/history/administrator.go:41 | openOpenSession | W | PASS-CB | — |
| atlas-character | services/atlas-character/atlas.com/character/session/history/administrator.go:56 | overlapping | R | PASS-CB | — |
| atlas-character | services/atlas-character/atlas.com/character/session/history/administrator.go:76 | overlapping2 | R | PASS-CB | — |

### atlas-configurations (global service — out of tenant scope)

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-configurations | services/atlas-configurations/atlas.com/configurations/services/administrator.go:16 | create | W | PASS-GLOBAL | Entity is global (no `tenant_id` column); service stores cross-tenant service definitions. F4 (raw `Exec`) is benign because the table has no `tenant_id` — callback would skip regardless. |
| atlas-configurations | services/atlas-configurations/atlas.com/configurations/services/administrator.go:36 | update.history | W | PASS-GLOBAL | — |
| atlas-configurations | services/atlas-configurations/atlas.com/configurations/services/administrator.go:43 | update.save | W | PASS-GLOBAL | — |
| atlas-configurations | services/atlas-configurations/atlas.com/configurations/services/administrator.go:60 | delete.history | W | PASS-GLOBAL | — |
| atlas-configurations | services/atlas-configurations/atlas.com/configurations/services/administrator.go:65 | delete.delete | W | PASS-GLOBAL | — |
| atlas-configurations | services/atlas-configurations/atlas.com/configurations/templates/administrator.go:23 | save | W | PASS-GLOBAL | — |
| atlas-configurations | services/atlas-configurations/atlas.com/configurations/templates/administrator.go:38 | delete | W | PASS-GLOBAL | — |
| atlas-configurations | services/atlas-configurations/atlas.com/configurations/templates/processor.go:93 | createTemplate | W | PASS-GLOBAL | — |
| atlas-configurations | services/atlas-configurations/atlas.com/configurations/tenants/administrator.go:25 | delete.history | W | PASS-GLOBAL | tenants/Entity has nullable `tenant_id uuid` but no `not null`; entity stores tenant-config mappings owned globally by the configurations service. Callback would inject only if column is detected as "tenant_id" + ctx has a tenant; this service runs without tenant context. |
| atlas-configurations | services/atlas-configurations/atlas.com/configurations/tenants/administrator.go:30 | delete.row | W | PASS-GLOBAL | — |
| atlas-configurations | services/atlas-configurations/atlas.com/configurations/tenants/administrator.go:50 | update.history | W | PASS-GLOBAL | — |
| atlas-configurations | services/atlas-configurations/atlas.com/configurations/tenants/administrator.go:59 | update.save | W | PASS-GLOBAL | — |
| atlas-configurations | services/atlas-configurations/atlas.com/configurations/tenants/processor.go:128 | createTenant | W | PASS-GLOBAL | — |

### atlas-drop-information

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-drop-information | services/atlas-drop-information/atlas.com/dis/continent/drop/administrator.go:22 | bulkCreate | W | PASS-CB | — |
| atlas-drop-information | services/atlas-drop-information/atlas.com/dis/continent/drop/provider.go:13 | all | R | PASS-CB | — |
| atlas-drop-information | services/atlas-drop-information/atlas.com/dis/monster/drop/administrator.go:22 | bulkCreate | W | PASS-CB | — |
| atlas-drop-information | services/atlas-drop-information/atlas.com/dis/monster/drop/provider.go:13 | all | R | PASS-CB | — |
| atlas-drop-information | services/atlas-drop-information/atlas.com/dis/reactor/drop/administrator.go:20 | bulkCreate | W | PASS-CB | — |
| atlas-drop-information | services/atlas-drop-information/atlas.com/dis/reactor/drop/provider.go:13 | all | R | PASS-CB | — |

### atlas-fame

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-fame | services/atlas-fame/atlas.com/fame/fame/administrator.go:24 | create | W | PASS-CB | — |
| atlas-fame | services/atlas-fame/atlas.com/fame/fame/administrator.go:32 | deleteByCharacterId | W | PASS-CB | — |
| atlas-fame | services/atlas-fame/atlas.com/fame/fame/provider.go:15 | byCharacterIdLastMonthEntityProvider | R | PASS-CB | — |

### atlas-families

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-families | services/atlas-families/atlas.com/family/family/administrator.go:98 | save | W | PASS-CB | — |
| atlas-families | services/atlas-families/atlas.com/family/family/administrator.go:113 | deleteByCharacterId | W | PASS-CB | — |
| atlas-families | services/atlas-families/atlas.com/family/family/entity.go:42–88 | Migration (`Exec(CREATE INDEX ...)` ×5) | MIG | PASS-MIGRATION | Schema DDL invoked at boot; not a request-time data path. |
| atlas-families | services/atlas-families/atlas.com/family/family/provider.go:15 | byCharacterIdEntityProvider | R | PASS-CB | — |
| atlas-families | services/atlas-families/atlas.com/family/family/provider.go:29 | byIdEntityProvider | R | PASS-CB | — |
| atlas-families | services/atlas-families/atlas.com/family/family/provider.go:43 | bySeniorIdEntityProvider | R | PASS-CB | — |

### atlas-gachapons

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-gachapons | services/atlas-gachapons/atlas.com/gachapons/gachapon/administrator.go:24 | create | W | PASS-CB | — |
| atlas-gachapons | services/atlas-gachapons/atlas.com/gachapons/gachapon/administrator.go:50 | deleteById | W | PASS-CB | — Struct-where `db.Where(&entity{ID: id})` is a struct query with zero `TenantId`; GORM skips zero fields. Callback adds `tenant_id = ?` from ctx. PASS-CB. |
| atlas-gachapons | services/atlas-gachapons/atlas.com/gachapons/global/administrator.go:16 | create | W | PASS-CB | — |
| atlas-gachapons | services/atlas-gachapons/atlas.com/gachapons/global/administrator.go:31 | deleteById | W | PASS-CB | — |
| atlas-gachapons | services/atlas-gachapons/atlas.com/gachapons/item/administrator.go:17 | create | W | PASS-CB | — |
| atlas-gachapons | services/atlas-gachapons/atlas.com/gachapons/item/administrator.go:32 | deleteById | W | PASS-CB | — |

### atlas-guilds

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/administrator.go:17 | create | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/administrator.go:35 | updateName | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/administrator.go:48 | updateNotice | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/administrator.go:61 | updateEmblem | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/administrator.go:69 | deleteById | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/provider.go:13 | allProvider | R | PASS-CB | — `Preload("Members")` / `Preload("Titles")` target tables that DO have `tenant_id` (`guild_members`, `guild_titles`). Preload sub-query passes through callback. |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/provider.go:24 | byIdEntityProvider | R | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/provider.go:35 | byWorldAndName | R | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/member/administrator.go:19 | create | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/member/processor.go:58 | deleteByGuildAndCharacter | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/member/provider.go:12 | byGuildIdProvider | R | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/member/provider.go:23 | byGuildAndCharacter | R | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/character/processor.go:48 | upsert (insert path) | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/character/processor.go:55 | upsert (update path) | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/character/provider.go:12 | byCharacterIdEntityProvider | R | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/title/administrator.go:23 | create | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/title/processor.go:50 | deleteByGuildId | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/guild/title/provider.go:12 | byGuildIdProvider | R | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/thread/administrator.go:26 | create | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/thread/administrator.go:43 | update | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/thread/administrator.go:51 | deleteByGuildAndId | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/thread/provider.go:12 | byGuildIdProvider | R | PASS-CB | — `Preload("Replies")` targets `thread_replies` which has `tenant_id`. |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/thread/provider.go:23 | byGuildAndId | R | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/thread/reply/administrator.go:18 | create | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/thread/reply/administrator.go:26 | deleteByThreadAndId | W | PASS-CB | — |
| atlas-guilds | services/atlas-guilds/atlas.com/guilds/thread/reply/provider.go:12 | byThreadIdProvider | R | PASS-CB | — |

### atlas-inventory

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-inventory | services/atlas-inventory/atlas.com/inventory/asset/administrator.go:47 | create | W | PASS-CB | — |
| atlas-inventory | services/atlas-inventory/atlas.com/inventory/asset/administrator.go:95 | deleteById | W | PASS-CB | — |
| atlas-inventory | services/atlas-inventory/atlas.com/inventory/asset/entity.go:12 | Migration `Exec(UPDATE assets ...)` | MIG | PASS-MIGRATION | Boolean-to-bitmask flag backfill at boot. |
| atlas-inventory | services/atlas-inventory/atlas.com/inventory/compartment/administrator.go:17 | create | W | PASS-CB | — |
| atlas-inventory | services/atlas-inventory/atlas.com/inventory/compartment/administrator.go:36 | update | W | PASS-CB | — |
| atlas-inventory | services/atlas-inventory/atlas.com/inventory/compartment/administrator.go:45 | deleteById | W | PASS-CB | — |
| atlas-inventory | services/atlas-inventory/atlas.com/inventory/compartment/provider.go:21 | byCharacterIdEntityProvider | R | PASS-CB | — |
| atlas-inventory | services/atlas-inventory/atlas.com/inventory/compartment/provider.go:27 | byCharacterAndTypeEntityProvider | R | PASS-CB | — |

### atlas-keys

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-keys | services/atlas-keys/atlas.com/keys/key/administrator.go:17 | create | W | PASS-CB | — |
| atlas-keys | services/atlas-keys/atlas.com/keys/key/administrator.go:29 | deleteByCharacterId | W | PASS-CB | — |
| atlas-keys | services/atlas-keys/atlas.com/keys/key/provider.go:12 | byCharacterIdAndKey | R | PASS-CB | — |
| atlas-keys | services/atlas-keys/atlas.com/keys/key/provider.go:18 | byCharacterId | R | PASS-CB | — |

### atlas-map-actions

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-map-actions | services/atlas-map-actions/atlas.com/map-actions/script/administrator.go:21 | upsert (insert) | W | PASS-CB | — script Entity field is `TenantID` (capital D); GORM normalizes to `tenant_id` column → callback handles. |
| atlas-map-actions | services/atlas-map-actions/atlas.com/map-actions/script/administrator.go:36 | upsert (lookup existing) | R | PASS-CB | — |
| atlas-map-actions | services/atlas-map-actions/atlas.com/map-actions/script/administrator.go:58 | upsert (update path lookup) | R | PASS-CB | — |
| atlas-map-actions | services/atlas-map-actions/atlas.com/map-actions/script/administrator.go:71 | deleteById | W | PASS-CB | — |
| atlas-map-actions | services/atlas-map-actions/atlas.com/map-actions/script/provider.go:14 | byIdProvider | R | PASS-CB | — |
| atlas-map-actions | services/atlas-map-actions/atlas.com/map-actions/script/provider.go:26 | byNameAndType | R | PASS-CB | — |
| atlas-map-actions | services/atlas-map-actions/atlas.com/map-actions/script/provider.go:38 | byName | R | PASS-CB | — |
| atlas-map-actions | services/atlas-map-actions/atlas.com/map-actions/script/provider.go:48 | all | R | PASS-CB | — |

### atlas-maps

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-maps | services/atlas-maps/atlas.com/maps/visit/administrator.go:19 | recordVisit (FirstOrCreate) | W | PASS-CB | — Entity has `tenant_id` in composite unique index. |
| atlas-maps | services/atlas-maps/atlas.com/maps/visit/administrator.go:28 | deleteByCharacterId | W | PASS-CB | — |
| atlas-maps | services/atlas-maps/atlas.com/maps/visit/provider.go:13 | byCharacterIdProvider | R | PASS-CB | — |
| atlas-maps | services/atlas-maps/atlas.com/maps/visit/provider.go:24 | byCharacterAndMapProvider | R | PASS-CB | — |
| atlas-maps | services/atlas-maps/atlas.com/maps/character/location/administrator.go:19 | upsertLocation | W | PASS-CB | — `db.Save(&e)` uses primary-key match; entity has `TenantId` as composite primary key. Callback layered on top. |
| atlas-maps | services/atlas-maps/atlas.com/maps/character/location/administrator.go:33 | deleteLocation | W | PASS-EXPLICIT | Hand-written `tenant_id = ? AND character_id = ?`. Defense-in-depth duplicate filter with callback. |
| atlas-maps | services/atlas-maps/atlas.com/maps/character/location/provider.go:17 | byCharacterIdProvider | R | PASS-EXPLICIT | Same shape. |

### atlas-marriages

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/administrator.go:36 | createProposal | W | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/administrator.go:52 | updateProposal | W | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/administrator.go:83 | createMarriage | W | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/administrator.go:99 | updateMarriage | W | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/administrator.go:140 | createCeremony | W | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/administrator.go:156 | updateCeremony | W | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:19 | byProposalId | R | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:42 | proposalByActors | R | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:75 | pendingProposals | R | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:112 | proposalByPair | R | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:142 | proposalsByProposer | R | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:173 | proposalByPair2 | R | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:201 | marriageByCharacters | R | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:229 | byMarriageId | R | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:254 | byEitherCharacter | R | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:284 | byCeremonyId | R | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:309 | byMarriageIdCeremony | R | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:396 | scheduledCeremonies | R | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:423 | activeCeremonies | R | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:451 | abandonedCeremonies | R | PASS-CB | — |
| atlas-marriages | services/atlas-marriages/atlas.com/marriages/marriage/provider.go:482 | expiredProposals | R | PASS-CB | — |

### atlas-merchant

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/shop/administrator.go:11 | create | W | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/shop/administrator.go:21 | save | W | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/shop/provider.go:18 | byIdProvider | R | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/shop/provider.go:32 | byCharacterId | R | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/shop/provider.go:43 | byCharacterAndType | R | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/shop/provider.go:57 | byLocation | R | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/shop/provider.go:68 | openOrMaintenance | R | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/shop/provider.go:79 | expiredOpenOrMaintenance | R | PASS-CB | — Called from `shop/task.go:28` under `WithoutTenantFilter`; per-row tenant context is reconstructed before each `CloseShopAndEmit` (line 51). |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/shop/task.go:28 | ExpirationTask.Run | RW | PASS-CROSS-TENANT | Explicit `database.WithoutTenantFilter(t.ctx)`; bypass scope ends at line 52 where `tenant.WithContext(t.ctx, ten)` rebuilds per-tenant context before processor calls. |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/listing/administrator.go:12 | create | W | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/listing/administrator.go:22 | deleteById | W | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/listing/administrator.go:48 | deleteByShopId | W | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/listing/provider.go:17 | byShopIdProvider | R | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/listing/provider.go:28 | byShopAndOrder | R | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/message/administrator.go:22 | create | W | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/message/provider.go:13 | byShopIdProvider | R | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/administrator.go:28 | bulkStore (tx.Create item) | W | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/administrator.go:53 | storeMeso | W | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/administrator.go:63 | clearItems | W | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/administrator.go:73 | clearMesos | W | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/administrator.go:93 | createNotification | W | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/administrator.go:103 | clearNotifications | W | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/administrator.go:113 | cleanupExpiredItems | W | PASS-CROSS-TENANT | Called from `task.go:31` under `WithoutTenantFilter`. Tenant-less by intent (global expiry sweep). |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/administrator.go:133 | deleteNotification | W | PASS-CB | — Called from `notification_task.go` per-row after deriving per-shop tenant; bypass ends at `tenant.WithContext` call there. |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/administrator.go:143 | cleanupExpiredMesos | W | PASS-CROSS-TENANT | Same as cleanupExpiredItems. |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/provider.go:12 | itemsByCharacter | R | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/provider.go:23 | mesosByCharacter | R | PASS-CB | — |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/task.go:28 | CleanupTask.Run | W | PASS-CROSS-TENANT | Explicit `WithoutTenantFilter`. Bypass scope ends within Run. |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go:36 | NotificationTask.Run | RW | PASS-CROSS-TENANT | Explicit `WithoutTenantFilter`; per-notification tenant rebuilt before downstream calls. |

### atlas-monster-book

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-monster-book | services/atlas-monster-book/atlas.com/monster-book/card/administrator.go:28 | upsertCard (lookup+upsert) | RW | PASS-EXPLICIT | Hand-written `tenant_id = ? AND character_id = ? AND card_id = ?`. Duplicate with callback (harmless). |
| atlas-monster-book | services/atlas-monster-book/atlas.com/monster-book/card/administrator.go:43 | upsertCard (create branch) | W | PASS-CB | — |
| atlas-monster-book | services/atlas-monster-book/atlas.com/monster-book/card/administrator.go:78 | deleteAllForCharacter | W | PASS-EXPLICIT | Hand-written `tenant_id = ? AND character_id = ?`. |
| atlas-monster-book | services/atlas-monster-book/atlas.com/monster-book/card/provider.go:14 | allByCharacter | R | PASS-EXPLICIT | Hand-written `tenant_id = ?`. |
| atlas-monster-book | services/atlas-monster-book/atlas.com/monster-book/card/provider.go:20 | byCharacterAndCard | R | PASS-EXPLICIT | Same. |
| atlas-monster-book | services/atlas-monster-book/atlas.com/monster-book/card/provider.go:26 | byCharacterAndSpecial | R | PASS-EXPLICIT | Same. |
| atlas-monster-book | services/atlas-monster-book/atlas.com/monster-book/collection/administrator.go:84 | byCharacterLookup | R | PASS-EXPLICIT | Same. |
| atlas-monster-book | services/atlas-monster-book/atlas.com/monster-book/collection/administrator.go:89 | deleteByCharacter | W | PASS-EXPLICIT | Same. |
| atlas-monster-book | services/atlas-monster-book/atlas.com/monster-book/collection/provider.go:13 | byCharacterProvider | R | PASS-EXPLICIT | Same. |

### atlas-notes

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-notes | services/atlas-notes/atlas.com/notes/note/administrator.go:14 | create | W | PASS-CB | — |
| atlas-notes | services/atlas-notes/atlas.com/notes/note/administrator.go:27 | update | W | PASS-CB | — |
| atlas-notes | services/atlas-notes/atlas.com/notes/note/administrator.go:42 | deleteById | W | PASS-CB | — |
| atlas-notes | services/atlas-notes/atlas.com/notes/note/administrator.go:48 | deleteByCharacterId | W | PASS-CB | — |
| atlas-notes | services/atlas-notes/atlas.com/notes/note/provider.go:12 | byIdProvider | R | PASS-CB | — |
| atlas-notes | services/atlas-notes/atlas.com/notes/note/provider.go:23 | byCharacterId | R | PASS-CB | — |
| atlas-notes | services/atlas-notes/atlas.com/notes/note/provider.go:34 | all | R | PASS-CB | — |

### atlas-npc-conversations

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/npc/administrator.go:21 | upsert (create) | W | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/npc/administrator.go:37 | upsert (lookup) | R | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/npc/administrator.go:62 | upsert (refetch) | R | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/npc/administrator.go:75 | deleteById | W | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/npc/provider.go:14 | byIdProvider | R | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/npc/provider.go:25 | byNpcIdProvider | R | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/npc/provider.go:35 | all | R | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/npc/provider.go:45 | byNpcAll | R | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/quest/administrator.go:21 | upsert (create) | W | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/quest/administrator.go:37 | upsert (lookup) | R | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/quest/administrator.go:63 | upsert (refetch) | R | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/quest/administrator.go:76 | deleteById | W | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/quest/provider.go:13 | byIdProvider | R | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/quest/provider.go:24 | byQuestIdProvider | R | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/quest/provider.go:34 | all | R | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/recipe/administrator.go:21 | bulkCreate (tx.Create) | W | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/recipe/administrator.go:33 | deleteByConversation | W | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/recipe/administrator.go:41 | deleteAll | W | PASS-CB | — `db.Where("1 = 1").Delete(&Entity{})` — `1=1` is a placeholder; callback still appends `tenant_id = ?` via `clause.Where{Eq}` (TestDoubleWhereIsHarmless covers the predicate combinator). |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/recipe/provider.go:14 | byItemId | R | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/recipe/provider.go:26 | byNpcId | R | PASS-CB | — |
| atlas-npc-conversations | services/atlas-npc-conversations/atlas.com/npc/conversation/recipe/provider.go:36 | all | R | PASS-CB | — |

### atlas-npc-shops

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/shops/administrator.go:21 | create | W | PASS-CB | — |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/shops/administrator.go:33 | byNpcLookup | R | PASS-CB | — |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/shops/administrator.go:42 | save | W | PASS-CB | — |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/shops/administrator.go:53 | deleteAll | W | PASS-CB | — `Where("1 = 1")` placeholder, callback adds tenant predicate. |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/shops/administrator.go:77 | bulkCreate (tx.Create) | W | PASS-CB | — |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/shops/provider.go:15 | byNpcIdProvider | R | PASS-CB | — |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/shops/provider.go:30 | all | R | PASS-CB | — |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/commodities/administrator.go:29 | create | W | PASS-CB | — |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/commodities/administrator.go:40 | byIdLookup | R | PASS-CB | — |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/commodities/administrator.go:52 | save | W | PASS-CB | — |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/commodities/administrator.go:62 | deleteById | W | PASS-CB | — |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/commodities/administrator.go:68 | deleteByNpcId | W | PASS-CB | — |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/commodities/administrator.go:74 | deleteAll | W | PASS-CB | — |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/commodities/administrator.go:100 | bulkCreate (tx.Create) | W | PASS-CB | — |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/commodities/entity.go:46 | Migration `Exec(CREATE INDEX)` | MIG | PASS-MIGRATION | DDL at boot. |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/commodities/provider.go:14 | byNpcIdProvider | R | PASS-CB | — |
| atlas-npc-shops | services/atlas-npc-shops/atlas.com/npc/commodities/provider.go:27 | all | R | PASS-CB | — |

### atlas-party-quests

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-party-quests | services/atlas-party-quests/atlas.com/party-quests/definition/administrator.go:20 | upsert (create) | W | PASS-CB | — Entity field `TenantID`. |
| atlas-party-quests | services/atlas-party-quests/atlas.com/party-quests/definition/administrator.go:34 | upsert (lookup) | R | PASS-CB | — |
| atlas-party-quests | services/atlas-party-quests/atlas.com/party-quests/definition/administrator.go:55 | upsert (refetch) | R | PASS-CB | — |
| atlas-party-quests | services/atlas-party-quests/atlas.com/party-quests/definition/administrator.go:67 | deleteById | W | PASS-CB | — |
| atlas-party-quests | services/atlas-party-quests/atlas.com/party-quests/definition/provider.go:13 | byIdProvider | R | PASS-CB | — |
| atlas-party-quests | services/atlas-party-quests/atlas.com/party-quests/definition/provider.go:23 | byQuestIdProvider | R | PASS-CB | — |
| atlas-party-quests | services/atlas-party-quests/atlas.com/party-quests/definition/provider.go:32 | all | R | PASS-CB | — |

### atlas-pets

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-pets | services/atlas-pets/atlas.com/pets/pet/administrator.go:29 | create | W | PASS-CB | — |
| atlas-pets | services/atlas-pets/atlas.com/pets/pet/administrator.go:111 | deleteById | W | PASS-CB | — |
| atlas-pets | services/atlas-pets/atlas.com/pets/pet/administrator.go:119 | replaceExcludes (delete) | W | LEAK-F8 | done: 9e51b335f — added `TenantId uuid.UUID` (indexed) to `exclude.Entity` + idempotent backfill in `Migration` from `pets.tenant_id` via `pet_id` FK. Delete now covered by callback. |
| atlas-pets | services/atlas-pets/atlas.com/pets/pet/administrator.go:133 | replaceExcludes (create) | W | LEAK-F8 | done: 9e51b335f — covered by the same `exclude.Entity` tenant_id column + backfill. |
| atlas-pets | services/atlas-pets/atlas.com/pets/pet/provider.go:13 | byIdEntityProvider | R | LEAK-F8 | done: 9e51b335f — preload now covered by callback once `excludes.tenant_id` exists. |
| atlas-pets | services/atlas-pets/atlas.com/pets/pet/provider.go:24 | byOwnerEntityProvider | R | LEAK-F8 | done: 9e51b335f — same as above. |

Note: the four LEAK-F8 pet rows collapse to one fix — add tenant_id to `exclude.Entity`. Counted as 1 LEAK-F8 in the Summary.

### atlas-portal-actions

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-portal-actions | services/atlas-portal-actions/atlas.com/portal/script/administrator.go:21 | upsert (create) | W | PASS-CB | — |
| atlas-portal-actions | services/atlas-portal-actions/atlas.com/portal/script/administrator.go:37 | upsert (lookup) | R | PASS-CB | — |
| atlas-portal-actions | services/atlas-portal-actions/atlas.com/portal/script/administrator.go:63 | upsert (refetch) | R | PASS-CB | — |
| atlas-portal-actions | services/atlas-portal-actions/atlas.com/portal/script/administrator.go:76 | deleteById | W | PASS-CB | — |
| atlas-portal-actions | services/atlas-portal-actions/atlas.com/portal/script/provider.go:14 | byIdProvider | R | PASS-CB | — |
| atlas-portal-actions | services/atlas-portal-actions/atlas.com/portal/script/provider.go:25 | byPortalIdProvider | R | PASS-CB | — |
| atlas-portal-actions | services/atlas-portal-actions/atlas.com/portal/script/provider.go:35 | all | R | PASS-CB | — |

### atlas-quest

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:23 | create | W | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:37 | clearProgress | W | PASS-CB | — `progress.Entity` has `tenant_id` index; `quest_status_id` is `uint32` but callback adds tenant filter. |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:49 | saveQuest | W | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:68 | saveQuest (alt) | W | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:81 | reset | W | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:93 | saveProgress | W | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:110 | bulkSave | W | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:125 | createProgress | W | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:144 | deleteByQuestStatusId | W | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:149 | deleteEntity | W | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:158 | bulkDelete (lookup) | R | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:163 | bulkDelete (progress) | W | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:166 | bulkDelete (entity) | W | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:187 | bulkCreateProgress | W | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/administrator.go:200 | bulkCreateProgress2 | W | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/provider.go:13 | byIdProvider | R | PASS-CB | — `Preload("Progress")` targets `progress` table which has `tenant_id`. |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/provider.go:24 | byCharacterIdProvider | R | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/provider.go:35 | byCharacterAndQuestId | R | PASS-CB | — |
| atlas-quest | services/atlas-quest/atlas.com/quest/quest/provider.go:46 | byCharacterAndStateProvider | R | PASS-CB | — |

Note: `quest_medal_maps` (`services/atlas-quest/atlas.com/quest/quest/medal/entity.go`) is declared but no call sites reference it — dormant entity (PASS-GLOBAL/dormant).

### atlas-reactor-actions

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-reactor-actions | services/atlas-reactor-actions/atlas.com/reactor/script/administrator.go:21 | upsert (create) | W | PASS-CB | — |
| atlas-reactor-actions | services/atlas-reactor-actions/atlas.com/reactor/script/administrator.go:37 | upsert (lookup) | R | PASS-CB | — |
| atlas-reactor-actions | services/atlas-reactor-actions/atlas.com/reactor/script/administrator.go:62 | upsert (refetch) | R | PASS-CB | — |
| atlas-reactor-actions | services/atlas-reactor-actions/atlas.com/reactor/script/administrator.go:75 | deleteById | W | PASS-CB | — |
| atlas-reactor-actions | services/atlas-reactor-actions/atlas.com/reactor/script/provider.go:14 | byIdProvider | R | PASS-CB | — |
| atlas-reactor-actions | services/atlas-reactor-actions/atlas.com/reactor/script/provider.go:25 | byReactorIdProvider | R | PASS-CB | — |
| atlas-reactor-actions | services/atlas-reactor-actions/atlas.com/reactor/script/provider.go:35 | all | R | PASS-CB | — |

### atlas-saga-orchestrator

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-saga-orchestrator | services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/store.go:217 | GetAllActive | R | PASS-CROSS-TENANT | Recovery on startup; `WithoutTenantFilter` explicit. Per-saga tenant rebuilt downstream. |
| atlas-saga-orchestrator | services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/store.go:228 | GetTimedOut | R | PASS-CROSS-TENANT | Same; uses `SKIP LOCKED` for safe global scan. |

### atlas-skills

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-skills | services/atlas-skills/atlas.com/skills/skill/administrator.go:23 | create | W | PASS-CB | — |
| atlas-skills | services/atlas-skills/atlas.com/skills/skill/administrator.go:85 | deleteByCharacterId | W | PASS-CB | — |
| atlas-skills | services/atlas-skills/atlas.com/skills/skill/administrator.go:92 | deleteByCharacterAndId | W | PASS-CB | — |
| atlas-skills | services/atlas-skills/atlas.com/skills/skill/provider.go:13 | byCharacterIdProvider | R | PASS-CB | — |
| atlas-skills | services/atlas-skills/atlas.com/skills/skill/provider.go:24 | byCharacterAndIdProvider | R | PASS-CB | — |
| atlas-skills | services/atlas-skills/atlas.com/skills/macro/administrator.go:10 | deleteByCharacterId | W | PASS-CB | — |
| atlas-skills | services/atlas-skills/atlas.com/skills/macro/administrator.go:25 | create | W | PASS-CB | — |
| atlas-skills | services/atlas-skills/atlas.com/skills/macro/provider.go:13 | byCharacterIdProvider | R | PASS-CB | — |

### atlas-storage

| Service | file:line | function | op | class | fix |
|---|---|---|---|---|---|
| atlas-storage | services/atlas-storage/atlas.com/storage/storage/administrator.go:20 | create | W | PASS-CB | — |
| atlas-storage | services/atlas-storage/atlas.com/storage/storage/administrator.go:49 | deleteById | W | PASS-CB | — |
| atlas-storage | services/atlas-storage/atlas.com/storage/storage/provider.go:14 | byWorldAndAccount | R | PASS-CB | — |
| atlas-storage | services/atlas-storage/atlas.com/storage/storage/provider.go:39 | byAccount | R | PASS-CB | — |
| atlas-storage | services/atlas-storage/atlas.com/storage/asset/administrator.go:47 | create | W | PASS-CB | — |
| atlas-storage | services/atlas-storage/atlas.com/storage/asset/administrator.go:57 | deleteById | W | PASS-CB | — |
| atlas-storage | services/atlas-storage/atlas.com/storage/asset/administrator.go:63 | deleteByStorageId | W | PASS-CB | — |
| atlas-storage | services/atlas-storage/atlas.com/storage/asset/entity.go:61 | Migration `Exec(UPDATE storage_assets ...)` | MIG | PASS-MIGRATION | Flag bitmask backfill at boot. |
| atlas-storage | services/atlas-storage/atlas.com/storage/asset/provider.go:11 | byStorageId | R | PASS-CB | — |
| atlas-storage | services/atlas-storage/atlas.com/storage/asset/provider.go:29 | byId | R | PASS-CB | — |
| atlas-storage | services/atlas-storage/atlas.com/storage/asset/provider.go:40 | byStorageAndTemplate | R | PASS-CB | — |
| atlas-storage | services/atlas-storage/atlas.com/storage/asset/provider.go:56 | byStorageAndInventoryType | R | PASS-CB | — |

## Leaks (Task 6 fix inventory)

| Class | Service | Site | Fix sub-task |
|---|---|---|---|
| LEAK-F2 | atlas-ban | services/atlas-ban/atlas.com/ban/ban/task.go:24-31 (`ExpiredBanCleanup.Run`) | Add `ctx context.Context` to struct; build via `database.WithoutTenantFilter(t.ctx)` and `t.db.WithContext(...)` rather than relying on missing-tenant-in-context (F2). Mirror pattern in `services/atlas-merchant/atlas.com/merchant/shop/task.go`. Threading of ctx into the constructor and main wiring required. |
| LEAK-F2 | atlas-ban | services/atlas-ban/atlas.com/ban/history/task.go:26-33 (`HistoryPurge.Run`) | Same fix shape as `ExpiredBanCleanup`. |
| LEAK-F8 | atlas-buddies | `services/atlas-buddies/atlas.com/buddies/buddy/entity.go` (table `buddies`) consumed by `services/atlas-buddies/atlas.com/buddies/list/provider.go:13` via `Preload("Buddies")`. | Add `TenantId uuid.UUID` (column `tenant_id`) + `gorm:"not null"` + idempotent `Migration` that backfills from `lists.tenant_id` via the existing FK (`list_id` ↔ `lists.id`). Once column exists, callback covers the preload sub-query. Update `Make` / `ToEntity` if a builder change is needed; verify `lists.Entity.Buddies` foreign key resolves the preload correctly. |
| LEAK-F8 | atlas-pets | `services/atlas-pets/atlas.com/pets/pet/exclude/entity.go` (table `excludes`) consumed by `pet/administrator.go:119`, `pet/administrator.go:133`, `pet/provider.go:13`, `pet/provider.go:24`. | Add `TenantId uuid.UUID` + `gorm:"not null"` + idempotent backfill `Migration` (join via `pet_id` to `pets.tenant_id`). Update `Make` builder. Once column exists the four LEAK-F8 rows above collapse to PASS-CB. Pet id collision risk is highest of the F8 sites (auto-increment uint32). |

LEAK-F6 (resolved): Task 2 hardened `tenantCreateCallback` to inject `tenant_id` on Create when missing. No outstanding leak sites — providers that still assign `TenantId` on the entity struct (e.g., atlas-character, atlas-fame, atlas-marriages) are defense-in-depth duplicates and harmless; removal is the explicit out-of-scope follow-up per design §5.

No outstanding F1, F3 (in the original sense — every tenant-scoped table the callback covers has a `tenant_id` column), F4 (every `Exec` is migration DDL or asset-flag bitmask backfill, all from a Migration function), F5 (no SQL `JOIN` clauses in the enumerated set — preloads handled under F8), F7 (no struct-where uses of tenant-scoped fields without explicit handling — `gachapons` struct-where omits TenantId, callback rescues), F9 (test setups register callbacks via `database.Connect` or `RegisterTenantCallbacks` — to be reverified by Task 8 smoke-test pass), or F10 (every existing `WithoutTenantFilter` site has a justification comment and a scope boundary verified above).

## PASS-CROSS-TENANT sites (full list)

Listed for Task 8 / future audit visibility. Every site is intentional cross-tenant; the bypass scope is verified to terminate before any tenant-aware downstream call.

| Service | Site | Justification |
|---|---|---|
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/shop/task.go:28 (`ExpirationTask.Run`) | Startup/periodic global sweep of expired shops; per-shop tenant context rebuilt at line 51 (`tenant.WithContext(t.ctx, ten)`) before each `CloseShopAndEmit`. |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/task.go:28 (`CleanupTask.Run`) | Global meso/item-cleanup sweep; bypass scope terminates inside `Run`. |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go:36 (`NotificationTask.Run`) | Global notification scan; per-notification tenant rebuilt before downstream effects. |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/administrator.go:111-118 (`cleanupExpiredItems`) | Called only from `CleanupTask.Run` under `WithoutTenantFilter`. No other callers verified by grep. |
| atlas-merchant | services/atlas-merchant/atlas.com/merchant/frederick/administrator.go:141-148 (`cleanupExpiredMesos`) | Same as above. |
| atlas-saga-orchestrator | services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/store.go:215-222 (`GetAllActive`) | Startup recovery; loads all in-flight sagas across tenants for processor revival. |
| atlas-saga-orchestrator | services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/store.go:226-236 (`GetTimedOut`) | Periodic timeout scan with `SKIP LOCKED`; per-saga tenant restored before compensation. |
| atlas-data | services/atlas-data/atlas.com/data/searchindex/searchindex.go:73-99 (`ResolveTenantId`), 109-205 (`Query`), 209-274 (`QueryFilter`), 239-275 (`Count`), 278-330 (`CountWithFilter`) | Global WZ search; tenant filter added as explicit `tenant_id = ?` in the `where` slice (`searchindex.go:213-214`, `:282-283`). Out of scope for this task; listed for completeness only. |

## Internal-consistency check

Run the three commands documented in plan §Task 3 Step 8 from the worktree root; all three must print `OK` (the awk line returns the call-site row count which should match the Summary total of 137).

(The plan's verifier commands are not embedded verbatim here because they themselves match this file's LEAK rows when grepped; run them from the plan as instructed.)

## Followup notes for Task 8 (per-service smoke tests)

Strict per-PRD §10 (resolved decision OQ-5): every service in §"Tenant-scoped services in scope" needs at least one read and one write provider test on the sqlite in-memory pattern (`libs/atlas-database/tenant_scope_test.go`). That is 28 services × 2 tests = 56 thin tests (atlas-configurations is intentionally global and exempt; LEAK-F8 fixes in atlas-pets and atlas-buddies each add a third test exercising the preload path post-migration).
