# Atlas Project TODO

This document tracks planned features and improvements for the Atlas MapleStory server project.

---

## Priority Summary

### Critical (Core Gameplay)
- [ ] **Character Attack Effects** - 26 unimplemented combat mechanics in `character_attack_common.go` (projectile consumption shipped in task-007)
- [ ] **Character Damage Effects** - 10 defensive abilities not processed
- [ ] **atlas-object-id silent-collision fallback** - On Redis allocator failure, monsters/reactors/drops fall back to returning `objectid.MinId` instead of failing the spawn, so every entity spawned during a Redis outage gets ID 1,000,000 and they all collide in storage

### High Priority (Feature Incomplete)
- [ ] **TokenItem Purchasing** - Returns "not implemented" error in NPC shops
- [ ] **Reactor Actions** - Boss weakening, environment manipulation, mass kill sagas

## Leader-election adoption (depends on task-064)

Each entry below is a per-service follow-up task — adopt `libs/atlas-lock`
for that service's sweep tickers so the Deployment can scale beyond one
replica without duplicating Kafka emission. See PRD §7.3 of
`docs/tasks/task-064-redis-leader-election/prd.md` for the catalogue.

- [ ] atlas-buffs — gate `NewExpiration`, `NewPoisonTick` (`services/atlas-buffs/atlas.com/buffs/main.go:63-64`)
- [ ] atlas-ban — gate `NewExpiredBanCleanup`, `NewHistoryPurge` (`services/atlas-ban/atlas.com/ban/main.go:79-80`)
- [ ] atlas-drops — gate `NewExpirationTask` (`services/atlas-drops/atlas.com/drops/main.go:92`)
- [ ] atlas-pets — gate `NewHungerTask` (`services/atlas-pets/atlas.com/pets/main.go:89`)
- [ ] atlas-skills — gate `NewExpirationTask` (`services/atlas-skills/atlas.com/skills/main.go:77`)
- [ ] atlas-reactors — gate `NewCooldownCleanup` (`services/atlas-reactors/atlas.com/reactors/main.go:68`)
- [ ] atlas-maps — gate `NewRespawn`, `NewWeather`, `NewMistTick` (`services/atlas-maps/atlas.com/maps/main.go:105-107`)
- [ ] atlas-merchant — gate `NewExpirationTask`, `NewCleanupTask`, `NewNotificationTask` (`services/atlas-merchant/atlas.com/merchant/main.go:79-81`)
- [ ] atlas-guilds — gate `NewTransitionTimeout` (`services/atlas-guilds/atlas.com/guilds/main.go:99`)
- [ ] atlas-account — gate `NewTransitionTimeout` (`services/atlas-account/atlas.com/account/main.go:76`)
- [ ] atlas-world — gate `NewExpiration` (`services/atlas-world/atlas.com/world/main.go:90`)
- [ ] atlas-invites — gate `NewInviteTimeout` (`services/atlas-invites/atlas.com/invites/main.go:80`)
- [ ] atlas-expressions — gate `NewRevertTask` (`services/atlas-expressions/atlas.com/expressions/main.go:49`)
- [ ] atlas-character — review `NewTimeout` (`services/atlas-character/atlas.com/character/main.go:102`); gate iff the work is global, not per-pod-session

The following two services are **review-and-decline** — listed for completeness, not for adoption:
- atlas-login — `NewTimeout` is per-pod session timeout, do NOT gate
- atlas-channel — `NewHeartbeat` is per-pod state by design, do NOT gate

---

## Services

### Buddies Service
- [ ] Trigger channel request for target when adding buddy (`list/processor.go:219`)
- [ ] Trigger channel request for target when accepting buddy (`list/processor.go:389`)

### Chalkboards Service
- [ ] Ensure character is in a valid location for chalkboard (`chalkboard/processor.go:53`)
- [ ] Ensure character is alive before setting chalkboard (`chalkboard/processor.go:54`)

### Channel Service
- [ ] Handle v83 trailing updateTime for cash item use (`character_cash_item_use.go:59`)
- [ ] Timing issue with loading pre-existing chalkboards
- [ ] Timing issue with loading pre-existing chairs
- [ ] Parties: Party Portals missing. Party member map, level, job, and name changes need to be considered
- [ ] Identify correct compartment type based on character job for cash shop (`cashshop/processor.go:105,150`)
- [ ] Select correct compartment in cash shop entry (`cash_shop_entry.go:59`)
- [ ] Block cash shop entry during: Vega scrolling, events, mini dungeons, already in shop (`cash_shop_entry.go:29-32`)
- [ ] Restrict skill targets to those in range based on bitmap (`skill/handler/common.go:48`)
- [ ] Pet lookup for movement processing (`movement/processor.go:80`)
- [ ] Optimize extra queries in pet consumer (`kafka/consumer/pet/consumer.go:238,276`)
- [ ] Pet skill and item writing (`socket/writer/character_info.go:33`)
- [ ] Query cash shop for whisper targets (`character_chat_whisper.go:73`)
- [ ] Remote channel lookup for whispers (`character_chat_whisper.go:84`)
- [ ] Send rejection to requester for declined invites (`kafka/consumer/invite/consumer.go:138`)
- [ ] Medal name retrieval (`kafka/consumer/message/consumer.go:211`)
- [ ] Server notice on map change failure (`socket/handler/map_change.go:42`)
- [ ] Verify not in mini dungeon for channel change (`channel_change.go:35`)
- [ ] Send server notice on channel change failure (`channel_change.go:40`)
- [ ] Validate NPC has ability to move (`npc_action.go:25`)
- [ ] Handle quest-in-progress states in NPC conversations (`npc_continue_conversation.go:25,27,31,40`)
- [ ] Announce guild operation errors (`guild_operation.go:138`)
- [ ] Send buddy operation errors to requester (`buddy_operation.go:48`)
- [ ] NPC producer NpcId population (`npc/producer.go:32,47`)
- [ ] NPC shop commodities model incomplete (`npc/shops/commodities/model.go:69`)
- [ ] Cash shop inventory item padded string and unknown fields (`socket/writer/cash_shop_operation.go:117,119,120`)
- [ ] Guild operation byte value (`socket/writer/guild_operation.go:94`)
- [ ] Buddy operation shop flag (`socket/writer/buddy_operation.go:118`)
- [ ] Multiple services have different cash shop message implementations (`kafka/message/cashshop/kafka.go:72`)
- [ ] Field migration bug not using instance (`kafka/consumer/character/consumer.go:79`)

#### Character Attack System (26 unimplemented effects)
Location: `socket/handler/character_attack_common.go`
- [x] ~~Projectile consumption on ranged attacks~~ — shipped in task-007 (bow/crossbow/claw/gun; Shadow Partner doubling; Soul Arrow skip; rechargeable qty=0 preservation in atlas-inventory)
- [ ] Apply cooldown
- [ ] Cancel dark sight / wind walk
- [ ] Apply combo orbs (add or consume)
- [ ] Decrease HP from DragonKnight Sacrifice
- [ ] Apply attack effects (heal, MP consumption, dispel, cure all, combo reset)
- [ ] Destroy Chief Bandit exploded mesos
- [ ] Apply Pick Pocket
- [ ] Increase HP from Energy Drain, Vampire, or Drain
- [ ] Apply Bandit Steal
- [ ] Fire Demon ice weaken
- [ ] Ice Demon fire weaken
- [ ] Homing Beacon / Bullseye
- [ ] Flame Thrower
- [ ] Snow Charge
- [ ] Hamstring
- [ ] Slow
- [ ] Blind
- [ ] Paladin / White Knight charges
- [ ] Combo Drain
- [ ] Mortal Blow
- [ ] Three Snails consumption
- [ ] Heavens Hammer
- [ ] ComboTempest
- [ ] BodyPressure
- [ ] Monster Weapon Atk Reflect
- [ ] Monster Magic Atk Reflect
- [x] Apply MPEater
- [ ] Passive no-consume for projectiles: Expert Marksmanship, Claw Mastery roll-to-preserve (planner stub in `socket/handler/character_attack_projectile.go`; Mortal Blow already listed above covers its passive-skip too)
- [ ] Characterize `AttackInfo.javlin` flag semantics and revisit projectile-consumption bailout (TODO cross-refs at `libs/atlas-packet/model/attack_info.go:153` ↔ `socket/handler/character_attack_projectile.go` planner javlin gate)

#### Character Damage System (10 unimplemented effects)
Location: `socket/handler/character_damage.go:24-33`
- [ ] Process Mana Reflection
- [ ] Process Achilles
- [ ] Process Combo Barrier
- [ ] Process Body Pressure
- [ ] Process PowerGuard
- [ ] Process Paladin Divine Shield
- [ ] Process Aran High Defense
- [ ] Process MagicGuard
- [ ] Process MesoGuard
- [ ] Decrease battleship HP

#### Protocol/Version Compatibility
- [ ] Test buddy model with JMS before moving to library (`socket/model/buddy.go:28`)
- [ ] Proper temp stat encoding for GMS v12 (`socket/model/monster.go:206`)
- [ ] Complete skill ID list for skill_usage_info (`socket/model/skill_usage_info.go:65,123,166`)
- [ ] Battle Mage attack info handling (`socket/model/attack_info.go:96,139`)
- [ ] Look up actual buff values if riding mount (`socket/model/character.go:482`)
- [ ] Document GMS v83/v95 constants (`socket/writer/character_attack_common.go:42,51,59`)
- [ ] Wild Hunter swallow (`socket/writer/character_attack_common.go:118`)
- [ ] BlazeWizardSpellMastery handling (`socket/writer/character_attack_common.go:158,171`)
- [ ] Clean up character spawn code (`socket/writer/character_spawn.go:76`)
- [ ] Handle GMS-JMS ring encoding differences (`socket/writer/character_spawn.go:101`)
- [ ] Fix crash issues in character effects (`socket/writer/character_effect.go:265,276`)
- [ ] Quest complete communication (`socket/writer/character_effect.go:119`)
- [ ] Write doors for party (`socket/writer/party_operation.go:32,191`)
- [ ] Party operation auto-reject flag (`socket/writer/party_operation.go:131`)
- [ ] Test party operations with JMS (`socket/writer/party_operation.go:200`)
- [ ] JMS map codes for cash shop (`socket/writer/cash_shop_operation.go:128`)
- [ ] Load gifts in cash shop (`socket/writer/cash_shop_operation.go:131`)

#### Remaining No-op Decode Packets (Category 2)
These packets have empty Decode implementations by design — they require runtime context
that is not available on the wire:
- [ ] `AttackWriter` (`character/attack_writer.go`) — variable damage counts, skill-dependent fields
- [ ] `EffectSkillUse` (`character/effect_skill_use.go`) — conditional bools not self-describing on wire
- [ ] `EffectSkillUseForeign` (`character/effect_skill_use.go`) — conditional bools not self-describing on wire

### Character Service
- [ ] Blocked name checking disabled (`processor.go:206`)
- [ ] Determine appropriate drop type and mod (`processor.go:741`)
- [ ] Define AP auto-assign range for Beginner/Noblesse/Legend (`processor.go:1252`)
- [ ] Award job change AP (Cygnus only?) (`processor.go:1477`)

### Character Factory Service
- [ ] BladeRecruit job ID handling (`job/model.go:13`)

### Consumables Service
- [ ] Consume Vega scroll (`consumable/processor.go:523`)
- [ ] Handle spikes/cursed property (`consumable/processor.go:526`)
- [ ] Field migration for monster requests (`monster/requests.go:28`)

### Data Service
- [ ] Player NPCs and CPQ support (`map/reader.go:116`)
- [ ] Validate skill reader logic (`skill/reader.go:174`)
- [ ] Handle map chairs (`skill/reader.go:178`)
- [ ] Handle LT in skills (`skill/reader.go:189`)
- [ ] Support mount types: SpaceShip, YetiMount1/2, Broomstick, BalrogMount (`skill/reader.go:210`)
- [ ] WindBreakerFinal statup validation (`skill/reader.go:231`)
- [ ] Weird logic check (`skill/reader.go:251`)
- [ ] Space dash handling (`skill/reader.go:280`)
- [ ] Power explosion handling (`skill/reader.go:293`)
- [ ] Better naming for skill properties (`skill/reader.go:425`)
- [ ] SnowCharge passes Duration as the WhiteKnightCharge stat amount; after task-054 this is 1000x larger (now ms, was raw seconds). Right fix: pass a charge-amount field (likely `e.X()`), not Duration (`skill/reader.go:373`)
- [ ] Skill effect cooldown unit normalization (post task-054): the `cooltime` XML attribute is read directly into `Cooldown uint32` with no conversion. Cooldown flows through atlas-character via the skill subsystem; unit semantics need a separate audit + fix. Companion follow-up to task-054 which only normalized Duration (`skill/reader.go:154`)

### Guilds Service
- [ ] Improve guild creation logic (`guild/processor.go:197`)
- [ ] Validate guild name (`guild/processor.go:237`)
- [ ] Respond with failure on guild errors (`guild/processor.go:320`)
- [ ] Proper error handling (`guild/processor.go:483,487`)
- [ ] Second query for party information (`party/rest.go:92`)

### Inventory Service
- [ ] Migrate TransactionId usage (5 locations in `kafka/consumer/compartment/consumer.go:118,133,148,214,266`)
- [ ] TransactionId removal from producers (`compartment/producer.go:63,124,138,153`)

### Invite Service
- [ ] Invites should be able to be queued

### Login Service

#### Error Response Handling
- [ ] Character view all selected PIC errors (`character_view_all_selected_pic.go:35,73,79`)
- [ ] Register PIC errors (`register_pic.go:37,42`)
- [ ] Accept TOS error (`accept_tos.go:31`)
- [ ] Character view all selected PIC register errors (`character_view_all_selected_pic_register.go:35,54,61,67`)
- [ ] Character view all selected errors (`character_view_all_selected.go:33,52,58`)

#### Other Login TODOs
- [ ] Blocked name checking disabled (`character/processor.go:56`)
- [ ] Clarify gender defaulting logic (`create_character.go:56`)
- [ ] Verify character is not engaged before deletion (`delete_character.go:95`)
- [ ] Verify character is not part of a family before deletion (`delete_character.go:96`)

### Monster Death Service
- [ ] Determine drop type (`monster/processor.go:22`)
- [ ] Party drop distribution (`monster/processor.go:149`)
- [ ] Account for healing (`monster/processor.go:160`)

### NPC Conversations Service
- [ ] Stale TODO comment in condition evaluator (`conversation/processor.go:590`)

### NPC Shops Service
- [ ] **Implement TokenItem purchasing** (`shops/processor.go:430`)

### Pets Service
- [ ] Generate cashId if cashId == 0 (`pet/processor.go:199`)

### Portals Service
- [ ] Transmit stats in portal transitions (`character/kafka.go:26`)

### Reactor Actions Service
- [ ] Create saga action for boss weakening (`script/executor.go:229,243`)
- [ ] Create saga action for environment object manipulation (`script/executor.go:250,260`)
- [ ] Create saga action for mass monster killing (`script/executor.go:267,272`)

### Reactors Service
- [ ] Implement `activateByTouch` reactor activation. 9 GPQ reactors (`6109013`, `6109014`, `6109021`–`6109027`) set `activateByTouch=1` in their .wz data (exposed via atlas-data `reader.go:79`) but atlas-reactors has no code path to trigger them from a character walking into their bounding area. All 9 are also skill-gated (types 5/6/7), so they remain activatable via hit for now. Full implementation needs a character-position signal from atlas-channel to atlas-reactors and a bounds check against every `activateByTouch` reactor on the map. Deferred from the reactor-persist/timer fix.

---

## Libraries

### atlas-constants
- [ ] BladeRecruit job ID handling (`job/model.go:92`)
- [ ] Translated name for FairytaleLandBeanstalkClimb2 (`map/constants.go:1641`)
- [ ] Define HiddenStreet Nett's Pyramid battle room maps (926010100-926023500) (`map/model.go:434`)

### atlas-object-id
- [ ] **Silent ID-collision on Redis failure.** `IdAllocator.Allocate` in each consumer (`services/atlas-monsters/atlas.com/monsters/monster/id_allocator.go:38-41`, and the inline equivalents in atlas-reactors and atlas-drops registries) swallows the error from `objectid.Allocator.Allocate` and returns `objectid.MinId` (1,000,000) as a fallback. Effect: during a Redis outage every monster, reactor, or drop spawned across the deployment is assigned the same id (1,000,000) and they collide in the per-tenant `<entity>:{tenantId}:{id}` storage key — only one entity survives in storage even though many were created. The v83 client also crashes on duplicate oids in the same field. Fix: propagate the allocation error all the way up to the spawn caller (Create/CreateAndEmit/etc.) and fail the spawn loudly. Discovered while documenting the shared allocator in task-019.

---

## Architectural

### Cross-Topic Kafka Atomicity
- [ ] Operations that produce to multiple Kafka topics (e.g., meso change + item create) are not atomic — if the first topic produce succeeds but the second fails, state becomes inconsistent. Consider Kafka transactional producers, an outbox pattern, or consolidating related commands onto a single topic.

### Character-creation saga races inventory-compartment creation
- [ ] **Race between `award_item_*` and atlas-inventory compartment creation.** The character-creation saga advances `create_character` → `award_item_0` (CREATE_ASSET) the moment `EventKindCharacterCreated` arrives (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go:161`). atlas-inventory independently consumes `CHARACTER_STATUS.CREATED` (`services/atlas-inventory/atlas.com/inventory/kafka/consumer/character/consumer.go:43-53`) and creates the 5 compartments serially in one tx. With cross-node Postgres latency, the CREATE_ASSET command lands at atlas-inventory before the Etc/Use/etc. compartment rows are committed; the lookup at `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:991` (`GetByCharacterAndType`) returns `record not found`, the saga step never gets a success event, the saga times out at 10s, and compensation deletes the character. Observed on 2026-05-15 with multi-namespace deployment (atlas-pr-461); the race always existed but pre-migration single-node Postgres was fast enough to commit compartments inside the ~67ms window between CHARACTER_CREATED and CREATE_ASSET. Fix candidates (option B from the triage session): add a new `AwaitInventoryCreated` saga action that the orchestrator waits on after `create_character` and before any `award_item_*`/`equip_*` step. Requires: new Action+payload in `libs/atlas-saga`; new `EventKindInventoryCreated`; acceptance-table entry; no-op handler in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:703` GetHandler so dispatch doesn't trip the unknown-action guard at `processor.go:947`; new consumer for `EVENT_TOPIC_INVENTORY_STATUS` in the orchestrator; `TransactionId` added to `services/atlas-inventory/atlas.com/inventory/kafka/message/inventory/kafka.go` `StatusEvent` struct (wire-format change); `CreatedEventStatusProvider` to embed transactionId; consumer to forward incoming `e.TransactionId` to `inventory.CreateAndEmit` instead of `uuid.New()`; `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go` builder to insert the await step. The pre-existing `AwaitCharacterCreated` constant at `libs/atlas-saga/model.go:135` is declared but unused — no working "passive wait" pattern in the orchestrator, so this is greenfield.

---

## Notes

### Summary Statistics
- **Total inline TODOs found**: ~170 across the codebase
- **Most concentrated areas**:
  - Channel Service: ~95 TODOs (socket handlers, writers, models)
  - Login Service: ~20 TODOs (error handling, character operations)
  - Data Service: ~10 TODOs (skill reader, map reader)
  - Inventory Service: ~9 TODOs (compartments, Kafka, TransactionId migration)
  - Character Service: ~4 TODOs (stat calculations, job changes)
  - Reactor Actions: 6 TODOs (saga actions for reactor operations)
  - Guilds: 6 TODOs (validation, error handling)

### Changes Since Last Review (2026-02-08)
- **Removed 7 stale references** that no longer exist in code:
  - `equipable/model.go:111` (inventory) - file doesn't exist
  - `asset/processor.go:309,386,392,431,437,595` (inventory) - TODOs removed
  - `kafka/consumer/drop/consumer.go:46,51` (inventory) - TODOs removed
  - `equipable/producer.go:36` (consumables) - TODO removed
  - `after_login.go:99` (login) - PIN termination implemented
  - Pre-compute HP/MP TODO (character) - removed from code
- **Updated line numbers** across inventory, login, character, and set_field writer

---

## atlas-ui Frontend

Deferred items from task-004 (Vite + React Router migration). The migration itself merged Phases 0, 1, 2, 3, 6, and 7; the items below were explicitly held back — in most cases because addressing them in the same PR would have multiplied the diff without changing feature parity, which was the migration's only correctness bar.

### Phase 2 deferrals (API client shrink)

- [x] ~~Shrink `services/atlas-ui/src/lib/api/client.ts` to the < 700 LOC soft target~~ — Done. Reduced from 1801 LOC → 333 LOC by deleting the cache layer, request deduplication, progress tracker, stream downloads, and retry state machine (React Query owns those responsibilities now).
- [x] ~~Remove the per-call `api.setTenant(tenant)` invocations across ~20 service modules.~~ Done — see the `refactor(atlas-ui): remove per-call api.setTenant duplicates` commit.
- [x] ~~Delete `services/atlas-ui/src/services/api/base.service.ts`~~ — Done. Every service rewritten as a plain object. Types extracted to `src/lib/api/query-params.ts` (145 LOC). Total API-layer LOC went from 2300 → 478 (79% reduction).
- [x] ~~Drop the `_tenant` parameter from service method signatures.~~ Done — 23 service files + ~60 caller sites updated; test assertions re-baselined.

### Phase 3 deferrals (page port)
- [ ] Audit `useSearchParams` semantics on filter-heavy pages (`ItemsPage`, `MapsPage`, `MerchantsPage`, `MonstersPage`, `NpcsPage`, `ReactorsPage`). The Phase 3 mechanical rewrite destructured the RR v7 tuple (`const [searchParams] = useSearchParams()`) so call sites compile, but the exact push/replace flow on filter changes should be spot-checked against Next.js behaviour (R1 in risks.md).
- [x] ~~Route-level `React.lazy` splitting for the 46 pages.~~ Done — main chunk is 256 KB (77 KB gzip); detail/rare pages lazy-load.
- [x] ~~Revisit the `INEFFECTIVE_DYNAMIC_IMPORT` warning from `vite build`.~~ No longer emitted by the current build.

### Phase 4 (data fetching consolidation — done)

- [x] ~~Convert every page that still carries a data-fetching `useEffect` to React Query.~~ Done across 27 pages in three passes. Completion bar `grep -rn "useEffect.*fetch\|useEffect.*\.service" services/atlas-ui/src/pages/` returns 0. Filter/search pages (ItemsPage, MapsPage, MerchantsPage, MonstersPage, NpcsPage, ReactorsPage) also dropped the `autoSearched` ref and let the URL's `?q=…` drive a single `useQuery`.
- Query keys stay colocated with each hook module (`xKeys = { all, lists, list, details, detail }`). Keeping them local is idiomatic React Query; factor into a shared `query-keys.ts` only if a cross-module invalidation layer becomes a real need.
- The `lib/hooks/useNpcData` / `useItemData` / `useMobData` / `useSkillData` hooks stay: they do non-trivial composition (service call + `getAssetIconUrl` derivation + batch/cache helpers) that the `lib/hooks/api/use<Resource>` hooks don't replicate. Delete them only if the composition moves into an equivalent `api/` hook.

### Phase 5 (Jest → Vitest — mechanical migration shipped; follow-ups below)

The mechanical migration landed: `jest.*` → `vi.*`, `next/navigation` + `next/link` mocks swapped for `react-router-dom` equivalents. Follow-up cleanup reports **471 passed / 0 skipped / 0 failed** across 26 test files (Vitest). Tests are excluded from `tsc -b` because test files carry pre-existing semantic type errors that are orthogonal to the migration.

All previously-skipped tests have been resolved:

- [x] ~~`src/components/features/tenants/__tests__/CreateTenantDialog.test.tsx`~~ — fixed region selector to tolerate multiple matches.
- [x] ~~`src/lib/utils/__tests__/toast.test.ts`~~ — swapped `jest.fn` → `vi.fn`.
- [x] ~~`src/lib/api/__tests__/errors.test.ts`~~ — production-mode cases now use `vi.stubEnv('DEV', false)`.
- [x] ~~`src/lib/breadcrumbs/__tests__/resolvers.test.ts`~~ — batch-resolution tests un-skipped; the helpers already resolve correctly under Vitest.
- [x] ~~`src/components/features/characters/__tests__/CharacterRenderer.test.tsx`~~ — reintroduced `data-testid="character-image"` on the migrated `<img>` markup.
- Deleted obsolete `accounts.service.test.ts`, `templates.service.test.ts`, `useTemplates.test.tsx`, and `conversations.service.test.ts` — they targeted class-based `BaseService` methods (`validate`, `transformResponse`, etc.) removed in the plain-object rewrite. Current surfaces are covered by the hook tests under `lib/hooks/api/__tests__/`.

Strict `tsconfig.app.json` status — all 7 home-hub strict flags are now on for production code:

- [x] ~~`noImplicitOverride`, `noUncheckedIndexedAccess`, `noUncheckedSideEffectImports`.~~ Done.
- [x] ~~`verbatimModuleSyntax`.~~ Done — ~30 call sites converted to `import { type X, Y }`.
- [x] ~~`erasableSyntaxOnly`.~~ Done — `BanType`, `BanReasonCode`, `WeaponType`, `CompartmentType`, `EntityType` converted to `as const` objects + companion types. `ResolverError`'s parameter-property constructor rewritten.
- [x] ~~`exactOptionalPropertyTypes`.~~ Done — no production hits needed fixing.
- [x] ~~`noUnusedLocals` + `noUnusedParameters`.~~ Done — ~80 hits fixed (unused React imports, unused destructures, `_tenant` prefix).
- [x] ~~Drop the `src/**/*.test.ts(x)` + `src/**/__tests__/**` excludes from `tsconfig.app.json`.~~ Done — 157 errors cleared across 12 test files: swapped `MockedFunction<typeof serviceObject>` → `Mocked<typeof serviceObject>` (or `vi.mocked(x)`) so the plain-object services typecheck; rebuilt `TenantBasic` mocks against the current `{ name, region, majorVersion, minorVersion }` schema; narrowed mock fixtures to satisfy `exactOptionalPropertyTypes`; swapped Jest-only `fail` for `expect.fail`; dropped stray unused imports. Test files now compile under the same strict flags as production code.

### Phase 7 deferrals (docs)
- [x] ~~Rewrite `services/atlas-ui/docs/service-layer.md` and `services/atlas-ui/docs/error-handling.md`.~~ Done — both now describe the Vite/RR/React Query stack. `CONTAINER_DEPLOYMENT.md` and the `BaseService` reference in `api-integration-patterns.md` also updated.
- [ ] Verify no remaining `next-themes` wrapper edge cases (system preference, theme flicker on initial SSR-ish load). The simplified `ThemeProvider` drops the "system" option in favour of explicit light/dark — revisit if users miss it.

### Tenant-switch invariant (correctness)
- [ ] Manual smoke test: tenant switching invalidates the React Query cache (new invariant from Phase 2, see `docs/tasks/task-004-atlas-ui-vite-migration/risks.md` R6). The Vitest covers the effect firing; a real-tenant E2E check is still needed.
- [ ] Manual smoke test: all four tenant headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`, SCREAMING_SNAKE_CASE) reach Go services unchanged — verify in devtools or server logs.

### Playwright (not in task-004 scope)
- [ ] No existing e2e suite. A smoke-test Playwright project covering the 46 routes + tenant switch would catch regressions that feature-parity refactors are prone to.

---

## task-037 character-presets follow-ups

Logged from `docs/tasks/task-037-character-presets/` design §7.

- [ ] **atlas-npc-shops deterministic stats migration** — set `UseAverageStats=true` in `services/atlas-npc-shops/atlas.com/npc/compartment/producer.go:13-19` so shop-bought equipment uses base stats verbatim.
- [ ] **atlas-character-factory player-creation deterministic stats** — set `UseAverageStats=true` for the four equip steps in `buildCharacterCreationSaga` (`services/atlas-character-factory/atlas.com/character-factory/factory/processor.go:138-211`).
- [ ] **AdminBootstrapWizard saga transactionId polling** — replace the "mutation resolved = success" assumption with per-row saga status polling (atlas-ui `AdminBootstrapWizard.tsx` step 4).
- [ ] **`<ItemPicker>` / `<SkillPicker>` components** — replace free-text uint32 inputs in `services/atlas-ui/src/pages/{templates,tenants}-character-presets-form.tsx` with searchable pickers backed by atlas-data.
- [ ] **Non-explorer 4th-job presets** — extend `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` with Cygnus / Aran / Resistance / Legend 4th-job presets.

## task-081 packet-audit validation follow-ups

Deferred from task-081 (validation-pivot). The exporter + validation toolchain + the
unattended four-version proof shipped; these refinements were registered rather than
left silent (see `docs/tasks/task-081-ida-export-reharvest/four-version-validation-results.md`).

- [ ] **Resolve demangled `Class::Method` helper names** in the MCP client — the new
  ida-pro-mcp `lookup_funcs` returns "Not found" for demangled names (only addr/`sub_XXXX`/
  mangled resolve), so named-helper descent yields `Unresolved` spans that lower match
  scores and suppress high-confidence annotations. Fix (mangle-and-retry or a `func_query`/
  `find_regex` name search) → higher recall → more confidently-validated `#`-mode shapes.
- [ ] **Triage the high-confidence divergences** surfaced by `validate` (≈6 across the four
  versions in the proof run) in IDA: real Atlas wire bug → fix `libs/atlas-packet/...` with a
  per-version byte test; hand-tracing error → correct the one baseline `calls` entry citing
  the IDA address. Re-validate to ✅.
- [ ] **Commit the bootstrapped `dispatch` selector annotations** (V5) into the four
  `docs/packets/ida-exports/*.json` baselines (additive only; `calls` unchanged) once a
  human-confirmation pass over the "ambiguous" picks lifts coverage — then re-validation is
  fully repeatable from committed inputs.
- [ ] **V7 ledger/guide**: update `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` with
  the `infer`/`validate` + `--ida-port` multi-IDB workflow and the dispatch-selector schema;
  re-curate the `_pending.md` registries to mark `#`-mode entries as live-verified.
- [ ] Optional: a `validate` mode that also handles if/else-chain dispatch handlers
  (e.g. `CLogin::OnCheckPasswordResult`) — currently honest `unverifiable` (a genuine
  static-extraction wall; may not be worth the complexity).
