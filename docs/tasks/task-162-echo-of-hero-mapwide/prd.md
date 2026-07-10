# Echo of Hero — Map-Wide Buff Application — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

Echo of Hero (the level-200 beginner-line skill, ids `1005` Beginner, `10001005`
Noblesse, `20001005` Legend, `20011005` Evan — collectively "X005") is meant to buff
**every character in the caster's map**. In Atlas today it buffs only the caster.

The statup mapping is already correct: `services/atlas-data/atlas.com/data/skill/reader.go:224-225`
maps all four X005 ids to `TemporaryStatTypeEchoOfHero` with amount `x`. The gap is in
the application path: `UseSkill` in
`services/atlas-channel/atlas.com/channel/skill/handler/common.go:107-111` applies the
buff to the caster and then to party members selected by the client-sent
affected-member bitmap (`applyToParty`). Echo of Hero is not in the packet decoder's
`isPartyBuff` list (`libs/atlas-packet/model/skill_usage_info.go:192`), so no bitmap is
ever decoded for it (bitmap stays `0`, and `selectPartyMembers` returns nothing for a
zero bitmap). Beginners are usually partyless anyway. Net effect: self-only.

Reference behavior (verified in Cosmic source): `StatEffect.applyEchoOfHero`
(`src/main/java/server/StatEffect.java:906-916`) applies the effect to the caster and
then to **every other character** returned by `MapleMap.getMapPlayers()`
(`MapleMap.java:3083`), which returns all characters in the map with **no filtering** —
Cosmic buffs dead characters and hidden GMs too. This task deliberately deviates from
Cosmic on that point (see FR-2): dead characters and hidden GMs are excluded.

## 2. Goals

Primary goals:
- Casting any X005 skill applies the Echo of Hero buff to every eligible character in
  the caster's map (field-scoped: same world/channel/map/instance), not just the caster.
- Eligibility excludes dead characters (HP 0) and hidden GMs.
- Recipients and bystanders see the same foreign buff behavior existing party buffs
  produce today — no new packet work.

Non-goals:
- No changes to the wire decode (`libs/atlas-packet`). Echo of Hero stays out of
  `isPartyBuff`, `isMobAffectingBuff`, and `isAntiRepeatBuffSkill`.
- No changes to atlas-data (the statup mapping already exists).
- Not addressing the "not all inclusive" TODOs in `skill_usage_info.go` (32111004 etc.).
- No other map-wide skills; this covers the four X005 ids only.
- No changes to the Echo of Hero buff's stat semantics, duration, or amount — the
  effect's `Duration()` / `StatUps()` flow through unchanged.

## 3. User Stories

- As a **level-200 beginner** (or Noblesse/Legend/Evan equivalent), I want casting Echo
  of Hero to buff everyone in my map so that the skill matches its intended purpose.
- As a **character standing in the map**, I want to receive the Echo of Hero buff (and
  see it in my buff bar) when someone casts it, even though I'm not in the caster's
  party.
- As a **bystander**, I want to see the buff-received effect on other buffed characters
  the same way I do for party buffs today.
- As a **hidden GM**, I do not want to receive buffs from player casts while hidden.
- As a **dead character**, I should not receive the buff.

## 4. Functional Requirements

### 4.1 Cast routing

- **FR-1.** When `UseSkill` processes a cast whose skill id is one of
  `BeginnerEchoOfHeroId`, `NoblesseEchoOfHeroId`, `LegendEchoOfHeroId`,
  `EvanEchoOfHeroId` and the effect has `Duration() > 0` and non-empty `StatUps()`,
  the buff MUST be applied to the map-wide recipient set (FR-2) **instead of** the
  self + party-bitmap path at `common.go:107-111`. A single map-wide loop covers the
  caster (the caster is a session in the map); the caster MUST NOT receive the buff
  twice.
- **FR-1.1.** All other skills MUST be unaffected — the existing self + `applyToParty`
  path remains their behavior.
- **FR-1.2.** Consumption side effects already handled earlier in `UseSkill`
  (HP/MP consume, item consume, cooldown) apply to the caster only, unchanged.

### 4.2 Recipient selection

- **FR-2.** The recipient set is every character with a live session in the caster's
  field (same world, channel, map, and instance — use the existing field-scoped
  session enumeration, e.g. `ForSessionsInMap`), EXCLUDING:
  - **FR-2.1.** characters whose HP is 0 (dead), and
  - **FR-2.2.** hidden GMs — characters with an active buff sourced from
    `SuperGmHideId` (9101004), the hide-state representation established by task-156
    (`.worktrees/task-156-gm-hide-heal-dispel/docs/tasks/.../design.md` OQ-2: hide is a
    `DARK_SIGHT` buff with `SourceId == 9101004`). If task-156 has not landed, this
    check is vacuous (no such buff can exist) but MUST still be implemented so the
    exclusion becomes effective when task-156 lands. Deviation from Cosmic is
    intentional (Cosmic buffs everyone unfiltered).
- **FR-2.3.** Party membership, the client-sent bitmap, and the effect's LT/RB
  rectangle MUST all be ignored for recipient selection. (Cosmic's
  `applyEchoOfHero` likewise bypasses rectangle logic entirely.)
- **FR-2.4.** A per-recipient failure (e.g. character fetch error for the HP check)
  MUST skip that recipient with a log line and continue; it MUST NOT abort the cast
  or the remaining recipients.

### 4.3 Buff application

- **FR-3.** Each recipient receives the buff via the existing
  `buff.NewProcessor(l, ctx).Apply(f, casterId, sourceId, skillLevel, duration,
  statups)` operator — identical parameters to today's self-apply (source id is the
  cast skill id; duration and statups come from the effect model). No new buff
  semantics.
- **FR-3.1.** Foreign visibility (buff bar on the recipient, buff-received effect seen
  by bystanders) MUST be whatever the existing per-recipient `Apply` emission already
  produces for party buffs — explicitly no new writers, packets, or opcodes.

### 4.4 Wire decode

- **FR-4.** `libs/atlas-packet/model/skill_usage_info.go` MUST NOT change. The X005
  cast packet decodes exactly as today (updateTime, skillId, skillLevel only).
  *Design-phase verification:* confirm via IDA (v83 client, `CUserLocal` skill-use
  send path) that the client sends no affected-member bitmap or extra payload for
  X005. If verification shows extra bytes, STOP and revisit this PRD — do not
  improvise a decode change.

### 4.5 Version scope

- **FR-5.** The behavior applies to all supported tenant versions (v83, v84, v87,
  v92, v95, jms). The four skill ids are version-stable constants; the change is
  server-side routing only, with no per-version gating, no new packets, and no
  tenant-config/opcode-table changes.

## 5. API Surface

None. No REST endpoints, no new Kafka topics, no packet/opcode changes. The existing
buff-application command flow (atlas-channel → atlas-buffs) is reused per recipient.

## 6. Data Model

None. No entities, migrations, or WZ/data changes. (`reader.go` statup mapping for
X005 already exists and is unchanged.)

## 7. Service Impact

- **services/atlas-channel** — only code change. `skill/handler` gains an Echo of
  Hero routing branch in `UseSkill` (or a per-skill mechanism consistent with the
  existing handler registry — design decision) plus a map-wide recipient selector
  alongside the existing party selectors in `skill/handler/recipients.go`.
- **libs/atlas-packet** — explicitly unchanged (FR-4).
- **services/atlas-data** — unchanged.
- **services/atlas-buffs** — unchanged (existing Apply flow handles per-character
  application and emission).

## 8. Non-Functional Requirements

- **Multi-tenancy:** all lookups and emissions flow through the existing
  tenant-scoped context (`tenant.MustFromContext`); the recipient enumeration is
  field-scoped and therefore tenant-scoped already.
- **Performance:** recipient count is bounded by map population. The per-recipient
  HP check requires a character fetch (as `selectPartyMembers` does today, but
  map-wide rather than ≤6 members); design should note the cost and prefer batch or
  already-cached data where the existing helpers provide it. No new hot loops.
- **Concurrency:** `ForSessionsInMap` runs callbacks concurrently — any shared
  accumulator must be synchronized (see the `inMapCharacterIdsFunc` mutex precedent
  at `recipients.go:57-70`).
- **Observability:** log a per-cast summary (caster, skill id, recipients applied,
  skipped-dead, skipped-hidden, fetch-failures) at debug level, following the
  existing `mob_buff_apply_summary` style.
- **Testing:** unit tests use the existing seam-variable + Builder patterns
  (`SkillUsageInfoBuilder`, `PartyRecipientBuilder` precedents); no
  `*_testhelpers.go` files.

## 9. Open Questions

- **OQ-1.** Client wire format for X005 casts (FR-4) — expected "no extra bytes";
  verify in IDA during design. If false, the decode question reopens.
- **OQ-2.** Exclusion phrasing: the user directive was interpreted as "exclude
  **hidden** GMs and dead characters." Visible GMs are treated as normal recipients.
  Confirm during design review if that interpretation is wrong.
- **OQ-3.** Whether the buff-received visual on non-party recipients renders
  correctly client-side across all versions (expected yes — same packets as party
  buffs); verify during implementation testing.

## 10. Acceptance Criteria

- [ ] Casting any of the four X005 skills applies the Echo of Hero buff to the caster
      and every other live-session character in the caster's field.
- [ ] Dead characters (HP 0) in the map do not receive the buff.
- [ ] Characters with an active `SourceId == 9101004` buff do not receive the buff
      (test via stubbed buff state; effective in production once task-156 lands).
- [ ] Characters in other maps/channels/instances do not receive the buff.
- [ ] The caster receives the buff exactly once.
- [ ] Non-X005 skills' behavior is unchanged (existing party-buff tests still pass).
- [ ] `libs/atlas-packet` diff is empty.
- [ ] Unit tests cover: routing (X005 → map-wide, non-X005 → legacy path), recipient
      exclusions (dead, hidden, other-map), per-recipient fetch-failure skip, and
      caster-included-once.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-channel;
      `tools/redis-key-guard.sh` clean; `docker buildx bake atlas-channel` clean if
      `go.mod` is touched.
