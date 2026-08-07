# Echo of Hero — Map-Wide Buff Application — Product Requirements Document

Version: v2
Status: Draft
Created: 2026-07-10
Updated: 2026-08-07 (rebased onto main; version scope widened to all 11 provisioned versions)
---

## 1. Overview

Echo of Hero (the level-200 beginner-line skill, ids `1005` Beginner, `10001005`
Noblesse, `20001005` Legend, `20011005` Evan — collectively "X005") is meant to buff
**every character in the caster's map**. In Atlas today it buffs only the caster.

The statup mapping is already correct: `services/atlas-data/atlas.com/data/skill/reader.go:253-254`
maps all four X005 ids to `TemporaryStatTypeEchoOfHero` with amount `x`. The gap is in
the application path: `UseSkill` in
`services/atlas-channel/atlas.com/channel/skill/handler/common.go:175-179` applies the
buff to the caster and then to party members selected by the client-sent
affected-member bitmap (`applyToParty`). Echo of Hero is not in the packet decoder's
`isPartyBuff` list (`libs/atlas-packet/model/skill_usage_info.go:196`), so no bitmap is
ever decoded for it (bitmap stays `0`, and `selectPartyMembers` returns nothing for a
zero bitmap — `recipients.go:236-238`). Beginners are usually partyless anyway. Net
effect: self-only.

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
- The behavior is correct on **every provisioned client version** (§4.5), without
  per-version branching in the handler.

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
- As an **operator running a legacy-version tenant** (v0.61–v0.79), I want the skill to
  behave correctly for whichever X005 variants that version actually ships, and to be
  inert for versions that ship none.

## 4. Functional Requirements

### 4.1 Cast routing

- **FR-1.** Echo of Hero MUST be routed through the per-skill handler registry
  (`skill/handler/registry.go`), registered against the four **skill identities**
  `skill.BeginnerEchoOfHero`, `skill.NoblesseEchoOfHero`, `skill.LegendEchoOfHero`,
  `skill.EvanEchoOfHero` — not against raw wire ids. `UseSkill` resolves the incoming
  wire id to its `Identity` through the tenant's version set before calling `Lookup`
  (`common.go:194-201`), so a single registration set covers every version.
- **FR-1.1.** The handler applies the buff to the map-wide recipient set (FR-2)
  **excluding the caster**. The caster's own copy is already applied by the generic
  buff step that runs earlier in `UseSkill` (`common.go:175-179`); because X005 is not
  an `isPartyBuff`, its bitmap is always `0` and the accompanying `applyToParty` call
  is a no-op. Net effect: the caster is buffed exactly once and `common.go` needs
  **no modification**.
- **FR-1.2.** The handler MUST apply the same gate as the generic buff step —
  `e.Duration() > 0 && len(e.StatUps()) > 0` — so a zero-duration or statup-less
  effect fans out to nobody.
- **FR-1.3.** All other skills MUST be unaffected — the existing self + `applyToParty`
  path remains their behavior.
- **FR-1.4.** Consumption side effects already handled earlier in `UseSkill`
  (HP/MP consume, item consume, cooldown) apply to the caster only, unchanged.

### 4.2 Recipient selection

- **FR-2.** The recipient set is every character with a live session in the caster's
  field (same world, channel, map, and instance) via the existing
  `SelectAllCharactersInMap` (`recipients.go:204`), EXCLUDING:
  - **FR-2.1.** the caster (already buffed — FR-1.1),
  - **FR-2.2.** characters whose HP is 0 (dead), and
  - **FR-2.3.** hidden GMs, detected with the canonical
    `character/buff.IsGmHidden(ctx, bs)` helper (`character/buff/hidden.go:21`).
    This helper resolves the buff's `SourceId` **through the tenant's version set**
    before comparing to `skill.SuperGmHide`, which is required for correctness: the
    SuperGM Hide wire id is `5101004` at v0.48 and `9101004` at v0.62+. A raw compare
    against either literal is version-incorrect and is banned by
    `tools/skill-job-id-guard.sh`. Deviation from Cosmic is intentional (Cosmic buffs
    everyone unfiltered).
- **FR-2.4.** Party membership, the client-sent bitmap, and the effect's LT/RB
  rectangle MUST all be ignored for recipient selection. (Cosmic's
  `applyEchoOfHero` likewise bypasses rectangle logic entirely.)
- **FR-2.5.** A per-recipient failure (e.g. buff fetch error for the hidden check)
  MUST skip that recipient with a log line and continue; it MUST NOT abort the cast
  or the remaining recipients.

### 4.3 Buff application

- **FR-3.** Each recipient receives the buff via the existing
  `buff.NewProcessor(l, ctx).Apply(f, casterId, sourceId, skillLevel, duration,
  statups)` operator — identical parameters to the caster's own apply in the generic
  step (source id is the raw cast wire id `info.SkillId()`; duration and statups come
  from the effect model). No new buff semantics.
- **FR-3.1.** Foreign visibility (buff bar on the recipient, buff-received effect seen
  by bystanders) MUST be whatever the existing per-recipient `Apply` emission already
  produces for party buffs — explicitly no new writers, packets, or opcodes.

### 4.4 Wire decode

- **FR-4.** `libs/atlas-packet/model/skill_usage_info.go` MUST NOT change. The X005
  cast packet decodes exactly as today (updateTime, skillId, skillLevel only).
  Verified in design §1 against the v83 IDB; see §4.5 for the cross-version argument.

### 4.5 Version scope

- **FR-5.** The behavior applies to **all 11 provisioned client versions** — every
  version with a generated identity set under `libs/atlas-constants/skill/` and a seed
  template under `services/atlas-configurations/seed-data/templates/`:

  | Version key | Beginner `1005` | Noblesse `10001005` | Legend `20001005` | Evan `20011005` | Status |
  |---|:---:|:---:|:---:|:---:|---|
  | `gms_v12`  | — | — | — | — | **n/a** — ships no X005 |
  | `gms_v48`  | — | — | — | — | **n/a** — ships no X005 |
  | `gms_v61`  | ✓ | — | — | — | in scope |
  | `gms_v72`  | ✓ | ✓ | — | — | in scope |
  | `gms_v79`  | ✓ | ✓ | ✓ | — | in scope |
  | `gms_v83`  | ✓ | ✓ | ✓ | — | in scope |
  | `gms_v84`  | ✓ | ✓ | ✓ | ✓ | in scope |
  | `gms_v87`  | ✓ | ✓ | ✓ | ✓ | in scope |
  | `gms_v92`  | ✓ | ✓ | ✓ | ✓ | in scope |
  | `gms_v95`  | ✓ | ✓ | ✓ | ✓ | in scope |
  | `jms_v185` | ✓ | ✓ | ✓ | ✓ | in scope |

  Source of truth: the `Id → Identity` maps in
  `libs/atlas-constants/skill/version_<key>_gen.go`. (Note this is the **tenant
  version set (11)**, not the packet coverage matrix's 9 columns — the matrix omits
  `gms_v12` and `gms_v92`. The matrix axis is irrelevant here because FR-4 makes this
  a zero-packet change.)

- **FR-5.1.** No per-version gating, `MajorVersion()` comparison, or version table
  may appear in the handler. Version correctness is a *structural* property of two
  existing mechanisms: (a) registry dispatch is keyed on `Identity` after a
  version-set resolve, so on `gms_v61` only wire `1005` reaches the handler and on
  `gms_v48` nothing does; (b) `buff.IsGmHidden` resolves the hide wire id through the
  same version set. Registering all four identities is therefore automatically
  correct — and inert — on every version above.
- **FR-5.2.** No tenant-config, opcode-table, or seed-template changes. The X005 cast
  arrives on the already-routed generic skill-use handler on every version.
- **FR-5.3.** The design-phase IDA verification (design §1) was performed on the v83
  client. It is not re-run per version; the cross-version argument is that FR-4 asserts
  a *negative* (X005 is absent from `isPartyBuff` / `isAntiRepeatBuffSkill` and the
  cast carries no bitmap), and any version where that were false would surface as a
  decode desync on the existing generic skill-use path — a loud failure, not a silent
  one. Recorded as a residual risk, not a verified claim (OQ-1).

## 5. API Surface

None. No REST endpoints, no new Kafka topics, no packet/opcode changes. The existing
buff-application command flow (atlas-channel → atlas-buffs) is reused per recipient.

## 6. Data Model

None. No entities, migrations, or WZ/data changes. (`reader.go` statup mapping for
X005 already exists and is unchanged.)

## 7. Service Impact

- **services/atlas-channel** — only code change. A new
  `skill/handler/echoofhero/` package registering the four identities, plus one blank
  import line in `skill/handler/registrations/registrations.go`. `common.go` is
  **unchanged** (FR-1.1).
- **libs/atlas-packet** — explicitly unchanged (FR-4).
- **libs/atlas-constants** — unchanged; all four identities and their per-version
  bindings already exist (DOM-21: define nothing new).
- **services/atlas-data** — unchanged.
- **services/atlas-buffs** — unchanged (existing Apply flow handles per-character
  application and emission).

## 8. Non-Functional Requirements

- **Multi-tenancy:** all lookups and emissions flow through the existing
  tenant-scoped context (`tenant.MustFromContext`); the recipient enumeration is
  field-scoped and therefore tenant-scoped already.
- **Performance:** recipient count is bounded by map population. `SelectAllCharactersInMap`
  already costs one character fetch per head; the hidden-GM check adds one buff fetch
  per head. This matches what `healdispel` (SuperGM Heal + Dispel) already does
  map-wide today, so the cost profile is established, not new.
- **Concurrency:** `ForSessionsInMap` runs callbacks concurrently — the existing
  mutex-guarded `inMapCharacterIdsFunc` (`recipients.go:64-78`) is reused verbatim;
  the fetch/filter loop is sequential, so no new shared-state hazards.
- **Observability:** log a per-cast summary (caster, skill id, recipients applied,
  skipped-dead, skipped-hidden, fetch-failures) at debug level, following the
  existing `mob_buff_apply_summary` style.
- **Testing:** unit tests use the existing seam-variable + Builder patterns
  (`SkillUsageInfoBuilder`, `PartyRecipientBuilder`, and the `healDispelDeps` seam
  struct precedent); no `*_testhelpers.go` files.

## 9. Open Questions

- **OQ-1.** Cross-version wire format (FR-5.3) — verified on v83 only; the other ten
  versions rest on the negative-assertion argument above. **Unverified** beyond that;
  re-open if a decode desync is ever observed on an X005 cast.
- **OQ-2.** *(Closed)* Exclusion phrasing: exclude **hidden** GMs and dead characters;
  visible GMs are normal recipients. Confirmed at design time.
- **OQ-3.** Whether the buff-received visual on non-party recipients renders
  correctly client-side across all versions (expected yes — same packets as party
  buffs, and `EchoOfHero` is already registered with `NoOpForeignValueWriter` at
  `libs/atlas-packet/model/character_temporary_stat.go:133`); verify during
  implementation testing.

## 10. Acceptance Criteria

- [ ] Casting any of the four X005 skills applies the Echo of Hero buff to the caster
      and every other live-session character in the caster's field.
- [ ] Dead characters (HP 0) in the map do not receive the buff.
- [ ] Hidden GMs do not receive the buff, detected via `buff.IsGmHidden` (version-aware).
- [ ] Characters in other maps/channels/instances do not receive the buff.
- [ ] The caster receives the buff exactly once.
- [ ] Non-X005 skills' behavior is unchanged (existing party-buff tests still pass).
- [ ] `libs/atlas-packet` diff is empty; `common.go` diff is empty.
- [ ] All four identities resolve to a registered handler via `Lookup` (registration test).
- [ ] Unit tests cover: recipient exclusions (caster, dead, hidden, other-map),
      per-recipient fetch-failure skip, and the zero-duration/no-statup no-op gate.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-channel;
      `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/skill-job-id-guard.sh`,
      and `tools/lint.sh --check` clean; `docker buildx bake atlas-channel` clean if
      `go.mod` is touched.
