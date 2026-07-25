# Morph Potion Routing — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-09
---

## 1. Overview

Use-tab morph potions (item classification 221, "transformation" consumables — 30 items in v83 WZ `Item.wz/Consume/0221.img.xml`) are currently consumed with no effect. The morph applier already exists: `ApplyItemEffects` in `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` maps the `morph` spec to `TemporaryStatTypeMorph` with a `time`-derived duration. It is dead code for these items because `usesStandardConsumer` only routes classifications {200, 201, 202, 205} through `ConsumeStandard`; classification 221 falls through to `ConsumeBare`, which commits the reservation (item decrements) and applies nothing.

This task routes classification 221 through the standard consumer and adds support for the `morphRandom` spec (a weighted random morph table), which atlas-data already parses and serves but no service applies. Of the 30 items, 28 carry a fixed `morph` spec and 2 carry `morphRandom` (verified by per-item scan of the v83 WZ): 2211000 "Cliff's Special Potion" (random self-morph) and 2212000 "Maplemas Party Potion" (random morph applied to *another* character).

IDA verification of the v83 client (`MapleStory_dump.exe`, v83_Me IDB) established two scope-critical facts. First, 2212000 never reaches the normal use-item flow: `CDraggableItem::OnDoubleClicked` gates item ids matching `id/10000 == 221 && (id%10000)/1000 == 2` (i.e. 2212xxx) into a dedicated target-picker dialog (`CUIRandomMorphDlg` via `CWvsContext::SendRandomMorphOtherRequest`) with its own serverbound request and clientbound response (`CWvsContext::OnRandomMorphRes`, with "failed to find user" and "only in town" failure arms). That is a separate, packet-sized feature and is out of scope here. Second, there is no morph-cancel-on-hit mechanic to implement: the client only sets morph state from server temporary-stat packets (`CUser::SetMorphed` is called solely from `OnTemporaryStatChanged` and init paths), and the `superman` property in `Morph.wz` (present only on skill-transform morphs 1000/1001/1003 and gender variants 1100/1101/1103) gates *attacking while morphed*, not cancellation. Cosmic cancels morph only on death — and Atlas already cancels all buffs on death via the respawn saga (`services/atlas-channel/atlas.com/channel/respawn/processor.go:242`, `CancelAllBuffs`) — so death handling comes for free once morph rides the buff pipeline.

## 2. Goals

Primary goals:
- Classification-221 items apply their morph effect (and all other specs they carry, e.g. `hp`) when consumed.
- `morphRandom` items apply one morph selected by weighted random from the item's morph table (self-use path, i.e. 2211000).
- Morph buffs expire after the item's `time` spec and are cancelled on death via the existing respawn `CancelAllBuffs` saga step (no new work; verified by test or existing coverage).

Non-goals:
- The 2212000 "morph another player" flow (`SendRandomMorphOtherRequest` / `OnRandomMorphRes` packets, target selection UI, town-only validation). The client intercepts 2212xxx double-clicks before any use-item packet is sent, so the standard consume path is unreachable for it. This needs its own packet-audit + implementation task; it should be filed as a follow-up backlog item when this task closes.
- Cash morph coupons (`Cash/0530.img.xml`, classification 530) — tracked separately in the item-effects backlog (Wholly-missing #16).
- Server-side rejection/anti-cheat for attack packets arriving while item-morphed (the client blocks attacking for non-`superman` morphs; Cosmic treats such packets as a hacked client and disconnects). Atlas has no equivalent enforcement pattern today and adding one is an anti-cheat concern, not an item-effects concern.
- Any client packet/writer changes. Morph is an existing temporary stat (`TemporaryStatTypeMorph`) that rides the current buff pipeline.

## 3. User Stories

- As a player, I want drinking a morph potion (e.g. Sky-Blue Wing Potion) to transform my character for the item's duration so that the item does what its description says instead of silently vanishing.
- As a player, I want Cliff's Special Potion to transform me into a random creature per its weighted table so that the "risk" flavor of the item works.
- As a player, I want the morph to disappear when I die, and when its duration ends, so that transformation behaves like the original game.

## 4. Functional Requirements

### 4.1 Routing (classification 221)

- FR-1: `usesStandardConsumer` MUST return true for classification 221, using the existing named constant `item.ClassificationConsumableTransformation` (`libs/atlas-constants/item/constants.go:39`) — not a raw numeric literal. Consuming any 221 item then flows through `ConsumeStandard` → `ApplyItemEffects`.
- FR-2: All other specs on 221 items MUST continue to apply via the existing generic spec handling (e.g. `hp: 50` on 2211000/2212000 heals; `time` drives buff duration). No special-casing beyond routing.

### 4.2 Fixed morph (28 items)

- FR-3: An item whose `morph` spec is > 0 applies a buff with stat `TemporaryStatTypeMorph`, amount = the morph id, duration = `time` spec / 1000 seconds, source = `-itemId` (already implemented at `processor.go:172-174`; covered by routing). A unit test MUST pin this end-to-end for a classification-221 item.

### 4.3 Random morph (`morphRandom`)

- FR-4: The consumables-side data model (`services/atlas-consumables/atlas.com/consumables/data/consumable/model.go`) MUST expose the already-populated `morphs map[uint32]uint32` field via a `Morphs()` getter. (The REST layer already deserializes it: `rest.go:70`.)
- FR-5: When an item has no positive `morph` spec but a non-empty `Morphs()` table, `ApplyItemEffects` MUST select exactly one morph id by weighted random — probability of morph `m` = `weight(m) / sum(all weights)` — and apply it as the `TemporaryStatTypeMorph` amount, same duration/source rules as FR-3. Do not assume weights sum to 100 (the two known items' weights do sum to 100, but the sum is data).
- FR-6: Selection MUST be deterministic under test (seedable/injectable randomness seam consistent with existing service patterns — the plan phase should reuse whatever seam task-131 random-reward items established, if any, rather than inventing a new one).
- FR-7: If both `morph > 0` and a non-empty `Morphs()` table are present (no such item exists in v83 data), the fixed `morph` spec wins. No error, no double-apply.

### 4.4 Explicitly not required

- FR-8 (verified non-requirement): No morph cancellation on taking damage. Verified against the v83 client (no client-side clear; morph state changes only via server temp-stat updates) and Cosmic (cancels only in `playerDead()`). Death cancellation already exists in Atlas via the respawn saga's `CancelAllBuffs`.

## 5. API Surface

No REST API changes. No new Kafka topics or message types. The change is internal routing + effect application inside atlas-consumables; buffs are applied through the existing atlas-buffs `Apply` command path (`buff.NewProcessor(...).Apply`), identical to the other stat-up specs.

## 6. Data Model

No database or schema changes. No atlas-data changes: `reader.go:142` (fixed `morph`) and `reader.go:153-160` (`morphRandom` → `Morphs` map) already parse and serve both specs, and the consumables REST model already carries them. The only model change is the additive `Morphs()` getter (FR-4).

## 7. Service Impact

- `atlas-consumables` — the only changed service:
  - `consumable/processor.go`: add `ClassificationConsumableTransformation` to `usesStandardConsumer`; add the weighted-random morph branch to `ApplyItemEffects`.
  - `data/consumable/model.go`: add `Morphs()` getter.
  - Tests for routing, fixed morph, weighted random selection (distribution + determinism), and the FR-7 precedence rule.
- `atlas-data`, `atlas-buffs`, `atlas-channel` — no changes (verified: parsing, buff application, and death cancellation already exist).

## 8. Non-Functional Requirements

- Multi-tenancy: no new tenant configuration. Behavior is driven entirely by per-tenant WZ-derived item data from atlas-data; versions whose 221 items differ get their own data automatically.
- Version coverage (all live client versions on main): The atlas-consumables change is **version-neutral by construction** — routing (`usesStandardConsumer`) keys off item *classification*, and `computeEffectPlan`/`morphRandom` carry no version literals, so the feature fires for every tenant version whose atlas-data serves classification-221 items. When this PRD was first written (2026-07-09) the live set was gms_83/84/87/92/95/12 + jms_185; the four legacy templates **gms_48, gms_61, gms_72, gms_79 landed on main 2026-07-13** and are in scope for this task. Three facts (verified against main) make legacy support require no code delta beyond this task's atlas-consumables change:
  1. `CharacterItemUseHandle` (the use-tab serverbound consume handler) is wired in **every** seed template, including gms_48 — so the consume request reaches atlas-consumables on all versions.
  2. `TemporaryStatTypeMorph` is registered **unconditionally** at a version-stable bit in `libs/atlas-packet/model/character_temporary_stat.go` (ahead of the v87/v95 gates), and `legacyGmsMask` handles the pre-v61 8-byte SecondaryStat mask width — so the MORPH temporary-stat encodes on legacy clients too.
  3. Death cancellation (`CancelAllBuffs` via the respawn saga) is likewise version-independent.
  Two items remain **unverified** and are validated in Task 6 (live atlas-data REST, read-only — not a code change): (a) whether the legacy WZ (v48/61/72/79) actually contains classification-221 items with `morph`/`morphRandom` specs (if absent, the feature is a harmless no-op on that version, not a regression); (b) that a MORPH temporary stat round-trips on at least one legacy client, including the pre-v61 mask path — the bit position is currently asserted only by a code comment, with no legacy fixture pinning it.
- Version stability: `TemporaryStatTypeMorph` already exists in the temporary-stat pipeline; no client-interpreted wire values are introduced by this task (DOM-25 not implicated — the morph id amount is game data from WZ, not a version-dependent wire byte).
- Observability: apply-path failures follow the existing `ApplyItemEffects` logging conventions (errors logged, consumption not rolled back — same semantics as existing stat-up specs).
- Testing: standard project verification — `go test -race`, `go vet`, `go build` in atlas-consumables; `docker buildx bake atlas-consumables` if `go.mod` is touched (not expected).

## 9. Open Questions

None blocking. One decision deferred to design: where the weighted-random selection seam lives (inline in `ApplyItemEffects` vs. a small pure helper on the consumable model) — pure-helper is recommended for testability, subject to the task-131 precedent check in FR-6.

## 10. Acceptance Criteria

- [ ] Consuming a classification-221 item with a fixed `morph` spec applies a `MORPH` temporary stat with the item's morph id and `time/1000` duration, and decrements the item — pinned by unit test.
- [ ] Consuming an item with `morphRandom` (2211000-shaped fixture) applies exactly one morph from the table; a seeded test pins selection, and a distribution-style test (or exhaustive seam test) verifies weighting by `prop/sum(props)`.
- [ ] `hp` and other coexisting specs on 221 items still apply (single test asserting HP recovery alongside morph).
- [ ] Fixed `morph` takes precedence over `morphRandom` when both are present (fixture-only case).
- [ ] Classification-221 items no longer route to `ConsumeBare` (routing unit test on `usesStandardConsumer`).
- [ ] No changes required in atlas-data / atlas-buffs / atlas-channel; verified by the diff touching only atlas-consumables.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-consumables; redis-key-guard clean.
- [ ] Legacy-version support confirmed (all live versions on main, added 2026-07-13: gms_48/61/72/79): via live atlas-data REST, at least one legacy tenant's classification-221 item data is inspected — if it carries morph specs, an end-to-end consume is confirmed to emit a MORPH temporary stat the legacy client reads (including the pre-v61 mask path for v48/61); if legacy WZ carries no 221 items, that is recorded as a documented no-op (feature intact, nothing to route). Read-only: no code, template, or WZ change.
- [ ] Follow-up backlog item filed for the 2212000 morph-other packet flow (serverbound request + clientbound `OnRandomMorphRes`, town-only rule, target resolution) with the IDA evidence from this PRD referenced.
