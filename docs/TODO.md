# Atlas Project TODO

This document tracks planned features and improvements for the Atlas MapleStory server project.

---

## Priority Summary

### Critical (Core Gameplay)
- [ ] **Character Attack Effects** - 26 unimplemented combat mechanics in `character_attack_common.go` (projectile consumption shipped in task-007)
- [ ] **Character Damage Effects** - 10 defensive abilities not processed
- [ ] **atlas-object-id silent-collision fallback** - On Redis allocator failure, monsters/reactors/drops fall back to returning `objectid.MinId` instead of failing the spawn, so every entity spawned during a Redis outage gets ID 1,000,000 and they all collide in storage

### High Priority (Feature Incomplete)
- [ ] **Reactor Actions** - Boss weakening, environment manipulation, mass kill sagas
- [ ] **Lint burn-down (task-171 follow-up)** - The Go linter layer of
  `tools/lint.sh` is rev-gated (`--new-from-rev` merge-base) so only new code
  fails CI. Burn down: fix pre-existing `standard`-group findings per module
  (run `tools/lint.sh --check --go --base <ancient-rev>` to enumerate), remove
  any escape-hatch exclusions in `.golangci.yml` marked "task-171 burn-down",
  then delete the `--new-from-rev` gating from `tools/lint.sh` so the linter
  layer enforces whole-tree like the formatters already do.
  - UI eslint suppressions: task-171 Task 3 (commit 947c45f71) landed 9 inline
    `eslint-disable` suppressions in atlas-ui (6 `react-hooks/set-state-in-effect`
    for genuine async-fetch/timer effects in
    `services/atlas-ui/src/components/item-name-cell.tsx`,
    `services/atlas-ui/src/components/map-cell.tsx`,
    `services/atlas-ui/src/lib/hooks/api/useAccountByName.ts`,
    `services/atlas-ui/src/pages/AccountsPage.tsx`,
    `services/atlas-ui/src/components/features/npc/conversation/ConversationCanvas.tsx`,
    and `services/atlas-ui/src/lib/hooks/useBreadcrumbs.ts`; 3
    `react-hooks/use-memo` on variadic-dependency hooks in
    `services/atlas-ui/src/lib/utils/debounce.ts`). Burn down by migrating the
    ad-hoc fetch/loading-state hooks to React Query (which removes the manual
    loading state), then removing the suppressions.
  - atlas-tenant aliasing convention: `libs/atlas-tenant` declares `package
    tenant` but its directory path ends in `atlas-tenant`, so every import
    MUST be aliased as `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`
    — an unaliased import makes goimports inject a duplicate and break the
    build (CI `lint-go --check` catches it, but confusingly).
    `.golangci.yml` carries an explanatory NOTE at the goimports settings. A
    durable fix (rename the package/dir to match, or switch the formatter to
    gci) can retire this convention.
  - task-261 staticcheck QF1011 exclusion:
    `services/atlas-channel/atlas.com/channel/character/processor_test.go`
    carries `var _ func(character.Model) character.Model =
    mock.NewMockProcessor().PartyDecorator`, a compile-time assertion that
    `PartyDecorator` has exactly that signature. The go 1.27 / golangci-lint
    v2.13.1 migration's gofumpt sweep reformatted that line (dropped a
    redundant paren pair), which put it under `--new-from-rev` and surfaced
    staticcheck's QF1011 "could omit type" suggestion. Applying QF1011 would
    replace the explicit type with `var _ = ...`, which still compiles for
    any signature `PartyDecorator` happens to have and so silently drops the
    assertion — a behavioral change, not a mechanical one, so `.golangci.yml`
    carries a path-scoped exclusion instead. Burn down by rewriting the
    assertion in a form staticcheck accepts without weakening it (e.g. a
    named `var partyDecorator func(character.Model) character.Model =
    mock.NewMockProcessor().PartyDecorator` binding used elsewhere in the
    test, or an explicit interface-conformance check), then remove the
    exclusion.

## MTS backend-audit follow-up — pre-existing debt (from task-102)

The task-102 full-service backend audits surfaced pre-existing DOM findings in
code task-102 did NOT create (its own additions audited clean). Deferred to a
dedicated follow-up rather than expanding task-102 into unrelated domains. See
`docs/tasks/task-102-mts-marketplace/audit-atlas-cashshop.md` and
`audit-atlas-tenants.md` for file:line detail.

- [ ] **atlas-cashshop `wallet` domain (Important ×4):** no `builder.go` (Model
  from raw struct literals, no validated `Build()`); no `Model.ToEntity()`; eager
  provider (`db.First` wrapped in `FixedProvider` instead of lazy `database.Query`);
  POST/PATCH handlers collapse every error to 500 (should be 400/404/409).
- [ ] **atlas-tenants configuration resource (Important ×2):** no `TransformSlice`
  (list handlers use inline loops); no `Model.ToEntity()`. Inherited by copy-paste
  from the Route/Vessel resources (the `mts-configs` additions themselves are clean).

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
- [x] Combo Drain
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
- [x] Write doors for party — implemented in task-093: `libs/atlas-packet/party/clientbound/created.go` (WithDoor/PartyCreatedBodyWithDoor) + `services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go` (handleCreated fetches leader door via GetByOwner, wires town/target/x/y into party-created packet, FR-3.3)
- [ ] **gms_v92 Mystic Door opcodes PARKED** — no v92 IDB available to IDA-verify SpawnDoor/RemoveDoor/SpawnPortal/RemoveTownDoor/EnterDoorHandle opcodes; `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` intentionally left without door rows (task-093). Unblocks when a v92 IDB exists (same situation as v92 MountFoodHandle, task-086).
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

### atlas-packet
- [ ] **Foreign-CTS shapes that still disagree with the client (non-disease).** Found while sweeping `SecondaryStat::DecodeForRemote` across all ten clients for task-195 / #1196; left alone there because none is a disease and none is reachable today. Evidence in `docs/tasks/task-195-foreign-disease-mobskill/investigation.md` §6.
  - `ShadowPartner` (v87+) foreign-writes `Short(level) + Short(sourceId)` via `LevelSourceForeignValueWriter`, truncating a 7-digit player skill id into 16 bits. The client reads one `Decode4` reason (`rShadowPartner` in the v95 PDB).
  - `BanMap` foreign-writes 4 bytes unconditionally, but gms_v48's remote decoder (`sub_5CBA1F`) reads none for that bit.
  - v95's remote decoder reads a `Decode4` reason for `Mechanic`, `DarkAura`, `BlueAura`, `YellowAura`; the registry has them `NoOp`. Latent only — atlas never originates these stats, so the bits are never set.
  - gms_v61's remote decoder has no `ReverseInput` branch at all, so a v61 tenant setting `Confuse` would desync. Atlas has no v61 Confuse source today.

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

The mechanical migration landed: `jest.*` → `vi.*`, `next/navigation` + `next/link` mocks swapped for `react-router-dom` equivalents. `grep -rlE 'jest\.(fn|mock|spyOn)' src` now returns zero. The suite stands at **1890 passed / 0 skipped / 0 failed** across 234 test files (Vitest).

Tests are **no longer excluded from `tsc -b`** — `tsconfig.app.json` includes all of `src` and excludes only `src/lib/api/examples/**`, so `npm run build` type-checks test files under the same strict flags as production code. (Corrected 2026-08-07: this paragraph previously claimed the opposite, which misled task-199 into shipping four commits that passed `npm run test` while failing `tsc -b`.)

All previously-skipped tests have been resolved:

- [x] ~~`src/components/features/tenants/__tests__/CreateTenantDialog.test.tsx`~~ — fixed region selector to tolerate multiple matches.
- [x] ~~`src/lib/utils/__tests__/toast.test.ts`~~ — swapped `jest.fn` → `vi.fn`.
- [x] ~~`src/lib/api/__tests__/errors.test.ts`~~ — production-mode cases now use `vi.stubEnv('DEV', false)`.
- [x] ~~`src/lib/breadcrumbs/__tests__/resolvers.test.ts`~~ — batch-resolution tests un-skipped; the helpers already resolve correctly under Vitest.
- [x] ~~`src/components/features/characters/__tests__/CharacterRenderer.test.tsx`~~ — reintroduced `data-testid="character-image"` on the migrated `<img>` markup.
- Deleted obsolete `accounts.service.test.ts`, `templates.service.test.ts`, `useTemplates.test.tsx`, and `conversations.service.test.ts` — they targeted class-based `BaseService` methods (`validate`, `transformResponse`, etc.) removed in the plain-object rewrite. Current surfaces are covered by the hook tests under `lib/hooks/api/__tests__/`.

Strict `tsconfig.app.json` status — all 7 home-hub strict flags are now on, for test files as well as production code (see the last item):

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
- [x] ~~Manual smoke test: tenant switching invalidates the React Query cache (new invariant from Phase 2, see `docs/tasks/task-004-atlas-ui-vite-migration/risks.md` R6).~~ Done (task-091) — the synchronous `applyTenant` set-before-fetch ordering is now covered by an automated unit test in `services/atlas-ui/src/context/__tests__/tenant-context.test.tsx` (assertions run inside the `act` callback, before the re-render). The header-passthrough smoke test below remains.
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
- [ ] **Non-explorer 4th-job presets** — extend `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` with ~~Cygnus /~~ Aran / Resistance / Legend 4th-job presets. (Cygnus 4th job is struck: verified in task-202 that no Cygnus 4th-job skills exist at any supported version — the WZ `skill` node is present but empty at 1112/1212/1312/1412/1512. See `docs/tasks/task-202-version-correct-job-hierarchy/availability-audit.md`.)

## task-145-player-reports follow-ups

Deferred from task-145 (player sue/claim reports). Design/plan/findings live under
`docs/tasks/task-145-player-reports/`.

### Feature scope deferred (result-code plumbing already expressive; wiring deferred)

- [x] **Claim quota / mesos-cost enforcement.** Done. A claim now costs
  `report.ClaimCostMesos` (300), charged only after atlas-ban confirms creation, with an
  affordability pre-check that emits claim mode `0x43` (`NOT_ENOUGH_MESOS`). atlas-ban
  counts the reporter's claims in a rolling `ClaimQuotaWindow` (7 days), rejects at
  `MaxClaimsPerWindow` (100) with mode `0x45` (`EXCEEDED`), and reports the true remaining
  count in the success payload instead of the former hard-coded 100.
- [ ] **Sue daily limit.** `sue` result code 2 (`DAILY_LIMIT`, "you may only report users 10
  times a day") is still never emitted — the claim quota above deliberately excludes sue,
  and nothing counts sue volume. Claim modes `0x47` (`TIME_WINDOW`) and `0x48`
  (`FALSE_REPORT_CITED`) likewise remain expressible but unused: Atlas advertises an
  always-open claim window (`open=0, close=0`) and tracks no false-report citations.
- [ ] **Accused-notification codes.** `sue` result code 3 and claim mode `0x03`
  (`REPORTED_NOTICE`) are accepted by the writers' operations tables, but nothing sends a
  notice to the *accused* character today — only the reporter's own result/claim packet is
  ever announced (`services/atlas-channel/atlas.com/channel/kafka/consumer/report/consumer.go`
  `handleStatusEvent`/`reportAnnouncer` target only `ReporterId`). Wiring this needs a second
  announce target (the accused's session) keyed off `AccusedId`, when prioritized.

### Blocked (need re-verification once unblocked, not scope decisions)

- [ ] **gms-12 report enablement.** Blocked on registry files + IDA export for gms_v12 —
  there is no `docs/packets/registry/gms_v12.yaml` and no gms_v12 column in the coverage
  matrix at all, so sue/claim opcodes for gms-12 are genuinely unverifiable today (unlike
  gms_v92, which this branch fully brought up — registry `docs/packets/registry/gms_v92.yaml`,
  IDA export, matrix column, all six sue/claim ops wired into
  `template_gms_92_1.json`, all 5 gms_92 report cells verified ✅). Config-entry work only
  when a gms_v12 IDB/registry becomes available — no code changes anticipated.
- [ ] **3 jms_185 clientbound claim cells blocked on a wedged IDA session.**
  `CLAIM_RESULT`/`CLAIM_AVAILABLE_TIME`/`CLAIM_STATUS_CHANGED` are live-routed on jms
  (`CWvsContext::OnPacket` @ `0xaebfe7`, cases `0x2A`/`0x2B`/`0x2C`, handlers named at
  `0xb0e9c3`/`0xb0ec69`/`0xb0ec92` — see `packet-findings.md` §7.3) and are ready to verify,
  but the jms IDA session (`b6864e54`) was wedged for this entire campaign: `idb_list`
  reported `is_active:true` with a recent `last_accessed`, but a direct `lookup_funcs` call
  against it timed out while sibling sessions responded normally in the same window. This is
  an infrastructure outage, not unscoped work — retry the verification pass
  (`/verify-packet` × those 3 cells) once the instance is healthy. Note: jms has **no**
  `CLAIM_REQUEST` send-site at all (5 independent exhaustive searches, §7.3) and **no** `sue`
  at all (§7.4) — those are genuine, already-recorded absences, not blocked work; do not
  re-open them.

### Test-coverage gaps (design/behavior judged correct; coverage deferred during review)

- [ ] `[]TranscriptLine` nil-vs-empty-slice asymmetry is untested
  (`services/atlas-ban/atlas.com/ban/report/model.go`). The frontend already handles both
  cases explicitly (`ReportDetailPage.tsx`'s `serverTranscript && serverTranscript.length > 0`
  guard treats a missing and an empty transcript identically), but the Go side has no test
  pinning that a zero-length captured transcript round-trips as `[]TranscriptLine{}` rather
  than `nil` (or vice versa) through the `jsonb` column.
- [ ] `chat/processor.go`'s `RecentInvolving` doc comment
  (`services/atlas-messages/atlas.com/messages/chat/processor.go:19-20`) asserts a
  "merged and sorted ascending by timestamp" contract with no `httptest` coverage of the
  404/empty/refused response paths for `/api/chat/history`.
- [ ] Report consumer wiring (topic resolution, header-parser registration, tenant
  round-trip on the happy path) in
  `services/atlas-ban/atlas.com/ban/kafka/consumer/report/` is verified only by
  code-comparison against the contract file, not by an in-package test. Live acceptance
  (task-145 plan Step 6, human-executed against real tenants) is the actual gate for this
  path; add an in-package consumer test if that live pass is not run promptly.
- [ ] `libs/atlas-redis/keyed_sorted_set.go`'s `AddBounded` (line 60) tests omit the
  exact-score-boundary case and the `maxCount<=0`/`ttl<=0` no-op branches.
- [ ] `services/atlas-messages/atlas.com/messages/chat/resource.go`'s
  `handleGetChatHistory` (line 82) has no end-to-end `httptest` coverage of the
  400/500/200 response shapes.
- [ ] Handler→processor argument-order wiring for the sue/claim handlers
  (`services/atlas-channel/atlas.com/channel/socket/handler/`) is verified only by code
  inspection; the handler tests re-pin codec decode (matching this codebase's pre-existing
  convention for other handlers) rather than asserting the processor call shape directly.

### Tooling defects found in `tools/packet-audit` (all confirmed live during this branch)

- [ ] **Direction inference produces silently-confident false-empty records.** A brand-new
  serverbound FName absent from both the prior-export roster and `candidatesFromFName` falls
  back to `DirClientbound` (`export.go`'s `directionFor`/`dirOf`); `ParseDecompile`
  (`parse.go`) then searches only for `CInPacket::Decode*`, finds none in a pure
  `COutPacket` send-site, and returns **zero calls without erroring** — counted as
  "1 resolved, 0 unresolved" rather than flagged as a gap. Reproduced live twice on this
  branch (gms_v92's `SendClaimRequest`, and again against the correct IDA endpoint during
  Task 23). Narrow mitigation already identified: treat a zero-call parse for a
  newly-introduced FName with no direction source as `unresolved` rather than confidently
  empty — does not touch the well-exercised inference paths for known FNames.
- [ ] **The CLI's default `--ida-url` points at a stale server.** Hardcoded default is
  `http://192.168.20.3:13337/mcp`; the working endpoint in this environment is
  `http://192.168.20.3:8745/mcp`. Port 13337 runs an older MCP schema whose `survey_binary`
  rejects the `database` parameter, so any export run against the CLI's own default dies
  with `Invalid params: unexpected parameters: ['database']`. This is a wrong default value
  in the tool, not an environment quirk — fix the default flag.
- [ ] **`md5: "unavailable"` is written as a value, silently.** A full (non-`--splice`)
  `export --version gms_v83` against the correct IDA endpoint writes `"md5": "unavailable"`
  with exit 0, because `survey_binary` reports no hash for that IDB — this value would then
  serve as a freshness anchor that can never match a real hash. Currently latent only because
  every affected export on this branch used `--splice`. Fix: treat `"unavailable"` (and any
  other non-hex string) as an error, not a value, before any full re-export.
- [ ] **`guardFromIf` can't resolve a bare call to a package-level bool helper as a version
  guard.** `internal/atlaspacket/analyzer.go`'s `guardFromIf` re-prints an `if` condition's AST
  to text and reparses it via `ParseGuard`, which only compiles `t.Region()`/`t.MajorVersion()`/
  `t.MinorVersion()` comparisons combined with `&&`/`||`/`!`; a bare call like
  `if UpdateTimeFirst(t) { ... }` (`cash/serverbound/item_use.go`) fails to parse and falls back
  to an always-true guard, so `FlattenWithRegistry` keeps every branch under that guard
  unconditionally regardless of version. Confirmed live on task-246: this stranded
  `CashItemUseMapleLife`'s `prefixName: "ItemUse"` header composition (the composed prefix
  needs `UpdateTimeFirst`'s version predicate to align the leading `update_time` write against
  each version's real header shape), so the candidate was left unlinked
  (`tools/packet-audit/cmd/run.go`, `CUICharacterSaleDlg::SendCreateNewCharacter`) rather than
  landing on a guard the resolver can't model faithfully. A prototype fix (inline a
  package-level, single-parameter, single-statement `return <expr>` bool helper's body into the
  guard before parsing) closed this cell cleanly but changed report content for other,
  already-audited packets sharing the same shape — `chat/serverbound/whisper.go`'s
  `whisperHasUpdateTime(t)` and `cash/serverbound/shop_operation_buy*.go`'s
  `legacyGMS(t)`/`buyOmitsCurrency(t)` at minimum (observed via full-directory regen diff
  against the previously-committed `docs/packets/audits/gms_v83/`) — too broad a blast radius
  to land without re-verifying every affected sibling cell by hand. A scoped fix (e.g. resolving
  only when the guard sits directly under the `prefixName`-composed method, or requiring an
  explicit per-candidate opt-in) is likely lower-risk than the general resolver change.

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
  the `infer`/`validate` + `--ida-database` multi-IDB workflow and the dispatch-selector schema;
  re-curate the `_pending.md` registries to mark `#`-mode entries as live-verified.
- [ ] Optional: a `validate` mode that also handles if/else-chain dispatch handlers
  (e.g. `CLogin::OnCheckPasswordResult`) — currently honest `unverifiable` (a genuine
  static-extraction wall; may not be worth the complexity).

## task-190 follow-up: USER_CALC_DAMAGE_STAT_SET_REQUEST handler (reserved as task-192)

Deferred from task-190 (disease-duration + CANCEL_DEBUFF). `USER_CALC_DAMAGE_STAT_SET_REQUEST`
is the tail of the same client handshake task-190 implements handlers for:
`CWvsContext::OnTemporaryStatReset` ends with `if (IsCalcDamageStat(mask)) { COutPacket(0x6C);
SendPacket(...); }` — so it fires more often once task-190's FR-2 (temporary-stat reset routing)
ships. It was kept out of scope on evidence, not overlooked: unlike `CANCEL_DEBUFF`, this send is
one-shot per stat reset, not a per-frame loop, so it cannot wedge a client the way an unhandled
`CANCEL_DEBUFF` did — the cost of leaving it unhandled is a possibly-stale client-side
damage-range display, not a hang. See `docs/tasks/task-190-disease-duration-cancel-debuff/investigation.md`
§8.3 for the IDA evidence.

- [ ] **Implement `USER_CALC_DAMAGE_STAT_SET_REQUEST` as task-192** — use 192 for this
  follow-up rather than calling `tools/task-numbers.sh next` fresh (it will now return a
  higher number, since 192 is already reserved).
  **Do not use task-184.** 184 was previously assigned to a gms_61 opcode-corruption incident
  (7 template edits wrong, caught by `matrix --check`) whose branch and PR were reverted and
  deleted. The deletion is what let `tools/task-numbers.sh next` report 184 as free when this
  entry was first written — its folder/branch/commit-subject scan is blind to anything whose
  artifacts were deleted after a revert. The commit that recorded this very note has since
  registered 184 in the tool's git-history scan, so `next` no longer offers it — but that
  registration is incidental, not the reason to avoid it: a number can look free to
  `tools/task-numbers.sh` and still be historically used, so treat the tool's output as a
  hint, not proof, and skip 184 on the historical grounds above regardless of what `next`
  reports at any given moment.
  Opcode is IDA-confirmed for only three of the ten live-tenant versions so far:
  GMS v48 `0x56` (86), GMS v61 `0x63` (99), GMS v83 `0x6C` (108)
  (`investigation.md:214`). The remaining seven (v72, v79, v84, v87, v92, v95, JMS v185) need
  the same per-version IDB pass task-190 ran for `CANCEL_DEBUFF` before a handler can be wired
  and routed in the seed templates.
  **Opcode collision to respect:** at GMS v61 the byte `0x63` is this packet
  (`USER_CALC_DAMAGE_STAT_SET_REQUEST`), while at GMS v83/v84 the same byte `0x63` is
  `CANCEL_DEBUFF` (`investigation.md:172-184`) — the two must never be routed by a hard-coded
  `0x63` constant; always resolve per-tenant from the version-specific template/registry
  entry, the same way task-190's `CancelDebuffHandle` routing does.

## task-217 Aran combo counter — landed

Shipped: an Aran/Legend holding a polearm and owning Combo Ability builds a
server-authoritative combo count. The client sends a body-less
`ARAN_COMBO_COUNTER` from its melee-hit path; atlas-channel re-derives every
gate (job, weapon, skill), advances a process-local tenant-keyed
`ComboMirror`, applies the Combo Ability buff once per combo chain, and
echoes the count back with `SHOW_COMBO` (one 4-byte little-endian value). A
1 Hz decay tick expires idle combos and cancels the buff; it deliberately
sends no packet — `DrawCombo` early-returns on a non-positive count without
releasing its digit layers, and the client clears its own HUD on the same
idle window (design.md §5.3). Six versions in scope — gms v83/v84/v87/v92/v95
and jms v185 — all twelve matrix cells (`ARAN_COMBO_COUNTER` serverbound +
`SHOW_COMBO` clientbound, per version) promoted to `✅`. See
`docs/tasks/task-217-aran-combo-counter/{prd,design,plan}.md`.

- [x] **Combo count build-up, decay, and `SHOW_COMBO` echo.** Done.
- [ ] **Combo *consumption* remains out of scope.** Combo Smash (`21100004`),
  Combo Fenrir (`21110004`), and Combo Tempest (`21120006`) still don't spend
  the count — that overlaps task-166's attack-pipeline surface (there is
  already a `// TODO ComboTempest` at
  `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:1090`)
  and would duplicate work in the same functions. Left un-observable as drift
  because the client's own `ClearCombo`/`DoActiveSkill` path clears its combo
  and sends the cancel on skill use, which resets the server mirror to match
  (design.md §5.5).
- [x] **Version-anchor correction.** The missing-features research corpus
  (`docs/research/missing-features/skills-and-buffs.md` §7 and
  `new-jobs-and-version-delta.md` §5) titles this entry "Aran combo counter
  (v84+)" — that anchor is wrong. `ARAN_COMBO_COUNTER` is present from **v83**
  onward (`docs/packets/audits/STATUS.md:726`; prd.md §1.2); v84 is the
  version that adds Evan, not Aran. That corpus is untracked in this worktree
  (confirmed absent under `docs/research/` here), so the correction could not
  be applied to the corpus files directly and is recorded here instead — apply
  it to `skills-and-buffs.md` §7 and `new-jobs-and-version-delta.md` §5 the
  next time that corpus is checked into a worktree that has it.

## task-219 follow-up: cash WZ re-ingest for morph-coupon `spec/morph`/`spec/hp`

Deferred from task-219 (transformation/morph coupons). `atlas-data`'s `cash/reader.go` now
materialises `spec/morph`/`spec/hp` for `Cash/0530.img.xml` items, and `atlas-consumables`'
`ConsumeMorphCoupon` applies them, but the reader change only takes effect for **newly
ingested** WZ data. Every tenant whose cash WZ was ingested before this change still serves
`Cash/0530` items with those spec fields absent from the stored data. Until the re-ingest
below runs for a given tenant, using a morph coupon there consumes the item and applies
nothing — the "both absent" row of the design's error table — so this note is meant to make
that first bug report self-answering rather than a mystery.

- [ ] **Re-ingest cash WZ for every provisioned tenant** so `spec/morph`/`spec/hp` populate
  from the existing `Cash/0530.img.xml` source data (no WZ content change needed, only a
  re-parse via the existing ingest path).
- [ ] **Verify per tenant** with a live `GET /data/{tenantId}/cash-items/5300000` and confirm
  the response's `spec` contains `morph: 1`, `hp: 50`, `time: 600000` (the PRD's worked
  example item). Repeat across provisioned tenants — this is the PRD §10 acceptance criterion
  this follow-up exists to close out operationally.

## task-219 follow-up: `ConsumeCashPetFood` compartment-type inconsistency (investigation, not a confirmed bug)

Side-observation from task-219's final whole-branch code review, unrelated to the morph-coupon
(0530) feature itself. `services/atlas-consumables/atlas.com/consumables/consumable/processor.go`,
in `ConsumeCashPetFood` (item family 0524 pet food — unambiguously a Cash-compartment item): the
first `ConsumeError` call correctly passes `inventory2.TypeValueCash`, but the `AwardFullness`
error path and the final `ConsumeItem`/`ConsumeError` pair (lines 601, 604, 606 as of this branch)
pass `inventory2.TypeValueUse` instead. Whether this causes actual item loss/duplication at
runtime, or `ConsumeItem`'s type parameter is unused on that code path, is **unverified** —
investigate before proposing a fix. See design §7.3 and the plan's Self-Review "Known limitation
carried forward, not fixed" note in `docs/tasks/task-219-cash-morph-coupons/design.md` and
`docs/tasks/task-219-cash-morph-coupons/plan.md` for the fuller reasoning.

- [ ] **Determine runtime impact** — trace whether `compartment.Processor.ConsumeItem`'s
  compartment-type argument is actually load-bearing (does passing `TypeValueUse` for a Cash
  item risk consuming from/erroring against the wrong compartment?) before deciding this needs
  a fix.
- [ ] **If confirmed as a defect, fix the three `TypeValueUse` call sites in `ConsumeCashPetFood`
  to `TypeValueCash`**, matching the function's own first `ConsumeError` call and the pattern
  `ConsumeMorphCoupon` (task-219) follows.

## task-277 follow-up: tenant Item.wz re-ingest for Writ of Solomon `spec/exp`

Deferred from task-277 (stored gachapon EXP items). Commit `d321b2e92` added
`atlas-data`'s parse of the consumable `info/maxLevel` field and `spec/exp` spec, and
`atlas-consumables`' `consumeSolomon` reads both to bank a Writ of Solomon's EXP, but the
reader change only takes effect for **newly ingested** WZ data. `document.DbStorage.Add`
persists the parsed `consumable.RestModel` as a JSON blob at ingest time and
`Storage.ByIdProvider` serves that blob verbatim thereafter; ingestion never re-runs at
request time or pod start. Every tenant whose `Item.wz` was ingested before this change
still serves `Consume/2370000` (and every other consumable) with `spec/exp` absent and
`info/maxLevel: 0`. Until the re-ingest below runs for a given tenant, using a Writ of
Solomon there is rejected via `ErrSolomonNoExperience` — the item is preserved, never
destroyed — so this note is meant to make that first bug report self-answering rather than
a mystery.

- [ ] **Re-ingest Item.wz `Consume` for every provisioned tenant, plus the canonical
  `GMS/83/1` dataset**, so `spec/exp` and `info/maxLevel` populate from the existing
  `Item.wz/Consume/0237.img.xml` source data (no WZ content change needed, only a re-parse
  via the existing `PATCH /data/wz` + `POST /data/process` ingest path).
- [ ] **Verify per tenant** with a live `GET /api/data/consumables/2370000` and confirm the
  response's `spec` contains `exp: 100000` and `maxLevel: 50` (the WZ source values at
  `Item.wz/Consume/0237.img.xml`, node `02370000`). Repeat across provisioned tenants — this
  is the PRD's own acceptance item this follow-up exists to close out operationally.
