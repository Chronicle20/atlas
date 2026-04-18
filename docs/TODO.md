# Atlas Project TODO

This document tracks planned features and improvements for the Atlas MapleStory server project.

---

## Priority Summary

### Critical (Core Gameplay)
- [ ] **Character Attack Effects** - 27 unimplemented combat mechanics in `character_attack_common.go`
- [ ] **Character Damage Effects** - 10 defensive abilities not processed

### High Priority (Feature Incomplete)
- [ ] **TokenItem Purchasing** - Returns "not implemented" error in NPC shops
- [ ] **Reactor Actions** - Boss weakening, environment manipulation, mass kill sagas

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

#### Character Attack System (27 unimplemented effects)
Location: `socket/handler/character_attack_common.go:94-120`
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
- [ ] Apply MPEater

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

---

## Libraries

### atlas-constants
- [ ] BladeRecruit job ID handling (`job/model.go:92`)
- [ ] Translated name for FairytaleLandBeanstalkClimb2 (`map/constants.go:1641`)
- [ ] Define HiddenStreet Nett's Pyramid battle room maps (926010100-926023500) (`map/model.go:434`)

---

## Architectural

### Cross-Topic Kafka Atomicity
- [ ] Operations that produce to multiple Kafka topics (e.g., meso change + item create) are not atomic — if the first topic produce succeeds but the second fails, state becomes inconsistent. Consider Kafka transactional producers, an outbox pattern, or consolidating related commands onto a single topic.

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
- [ ] Shrink `services/atlas-ui/src/lib/api/client.ts` to the < 700 LOC soft target (currently ~1800). Pragmatic drops: unused cache primitives, progress-tracker bytes-per-second math, revalidation/dedupe layers that the new React Query-owned cache already provides.
- [ ] Delete `services/atlas-ui/src/services/api/base.service.ts` (499 LOC) and inline `ServiceOptions`/`QueryOptions`/`BatchOptions`/`BatchResult`/`ValidationError` into `src/lib/api/query-params.ts` + `src/lib/api/json-api.ts`. Blocked on: every service module extends `BaseService` and ~15 hooks import the type exports. Restructure in a follow-up PR.
- [ ] Remove the per-call `api.setTenant(tenant)` invocations across ~20 service modules. The `TenantProvider` effect now wires this centrally — the per-call invocations are redundant duplicates. Also drop the `tenant` parameter from service method signatures once the duplicates are gone.

### Phase 3 deferrals (page port)
- [ ] Audit `useSearchParams` semantics on filter-heavy pages (`ItemsPage`, `MapsPage`, `MerchantsPage`, `MonstersPage`, `NpcsPage`, `ReactorsPage`). The Phase 3 mechanical rewrite destructured the RR v7 tuple (`const [searchParams] = useSearchParams()`) so call sites compile, but the exact push/replace flow on filter changes should be spot-checked against Next.js behaviour (R1 in risks.md).
- [ ] Route-level `React.lazy` splitting for the 46 pages. The current bundle is a single ~1.1 MB chunk (gzip ~300 KB). Lazy-load detail pages and rarely-visited routes to shrink the initial payload.
- [ ] Revisit the one `INEFFECTIVE_DYNAMIC_IMPORT` warning from `vite build` (`src/lib/breadcrumbs/resolvers.ts` dynamically imports service modules that are also statically imported by hooks).

### Phase 4 deferral (data fetching consolidation — whole phase)
- [ ] Replace every remaining `useState` + `useEffect` + service-call pattern with a React Query hook. Create missing hooks under `src/lib/hooks/api/use-<resource>.ts`. Centralise query keys in `src/lib/hooks/api/query-keys.ts`. Wire `useMutation.onSuccess` → `queryClient.invalidateQueries` for every write hook. Delete the legacy `lib/hooks/` wrappers (`useNpcData`, `useItemData`, `useMobData`, `useSkillData`). Completion bar: `grep -rn "useEffect.*fetch\|useEffect.*\.service" services/atlas-ui/src/pages/` returns 0.

### Phase 5 (Jest → Vitest — mechanical migration shipped; follow-ups below)

The mechanical migration landed: `jest.*` → `vi.*`, `next/navigation` + `next/link` mocks swapped for `react-router-dom` equivalents, and `npm run test` now reports **543 passed / 34 skipped / 0 failed** across 32 test files (Vitest). Tests are excluded from `tsc -b` because ~100 test files carry pre-existing semantic type errors that are orthogonal to the migration.

Skipped tests to un-skip (search for `it.skip(` or `describe.skip(` next to the migration comment):

- [ ] `src/components/features/tenants/__tests__/CreateTenantDialog.test.tsx` — `renders dialog with form fields when open`: stale DOM selectors match multiple elements after shadcn's dialog rev; fix the queries.
- [ ] `src/lib/utils/__tests__/toast.test.ts` — `should retry on failure and eventually succeed`: timer/retry loop semantics differ under Vitest fake timers; switch to `vi.useFakeTimers()` + `await vi.advanceTimersByTimeAsync()` pattern.
- [ ] `src/lib/api/__tests__/cache.test.ts` — six cache tests fail with "Body is unusable: Body has already been read" under undici's native `fetch`. The client's cache layer re-reads the Response body; rework to clone or consume via `.json()` once. Expected to go away when Phase 2's client shrink deletes the cache primitive entirely.
- [ ] `src/lib/api/__tests__/errors.test.ts` — two production-mode tests toggle `process.env.NODE_ENV = 'production'` to exercise `import.meta.env.PROD` branches. Vite resolves `import.meta.env` at build time, so the runtime flip is a no-op. Fix: refactor `errors.ts` to read from an injectable predicate, or drop the branch and use the dev-tool runtime guard.
- [ ] `src/lib/breadcrumbs/__tests__/resolvers.test.ts` — two batch-resolution tests hit 5s timeout; the `Promise.allSettled` mock pipeline never resolves under Vitest. Needs a rewrite using `vi.waitUntil` or explicit `await` points.
- [ ] `src/services/api/__tests__/conversations.service.test.ts` — two graph-traversal tests compare against a stale fixture; regenerate the fixture from the current `ConversationsService.validateStateConsistency` output.
- [ ] `src/components/features/characters/__tests__/CharacterRenderer.test.tsx` — the entire file is `describe.skip`'d. Every assertion looks for `data-testid="character-image"`, which the migrated `<img>` markup no longer emits. Either re-add the `data-testid` on the component or rewrite the selectors.

Plus: re-enable the strict tsconfig flags (`verbatimModuleSyntax`, `noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`, `noUnusedLocals`, `noUnusedParameters`, `erasableSyntaxOnly`, `noUncheckedSideEffectImports`) and drop the `src/**/*.test.ts(x)` + `src/**/__tests__/**` excludes from `tsconfig.app.json`. Expect ~100 errors mostly in test files and `base.service.ts` generics — most will disappear when Phase 2's client shrink + base.service.ts deletion lands.

### Phase 7 deferrals (docs)
- [ ] Rewrite `services/atlas-ui/docs/service-layer.md` and `services/atlas-ui/docs/error-handling.md`. Both still reference `NEXT_PUBLIC_API_URL`, `next/image`, and the App Router. Not blocking migration — they live under `services/atlas-ui/docs/` as historical architecture notes. Keep in sync or delete.
- [ ] Verify no remaining `next-themes` wrapper edge cases (system preference, theme flicker on initial SSR-ish load). The simplified `ThemeProvider` drops the "system" option in favour of explicit light/dark — revisit if users miss it.

### Tenant-switch invariant (correctness)
- [ ] Manual smoke test: tenant switching invalidates the React Query cache (new invariant from Phase 2, see `docs/tasks/task-004-atlas-ui-vite-migration/risks.md` R6). The Vitest covers the effect firing; a real-tenant E2E check is still needed.
- [ ] Manual smoke test: all four tenant headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`, SCREAMING_SNAKE_CASE) reach Go services unchanged — verify in devtools or server logs.

### Playwright (not in task-004 scope)
- [ ] No existing e2e suite. A smoke-test Playwright project covering the 46 routes + tenant switch would catch regressions that feature-parity refactors are prone to.
