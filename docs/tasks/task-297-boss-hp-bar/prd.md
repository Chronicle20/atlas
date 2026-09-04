# Boss HP Bar (FIELD_EFFECT / BOSS_HP) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-09-04

---

## 1. Overview

When a player fights a major boss — Zakum, Horntail, Papulatus, Pianus — the MapleStory client
renders a wide HP gauge across the top of the screen, colored by the mob's WZ `hpTagColor` /
`hpTagBgcolor` attributes. The gauge is entirely server-driven: the client only draws what the
server sends in `CField::OnFieldEffect` mode `BOSS_HP`. Atlas never sends it, so every boss in
the game today is fought with no top-of-screen gauge at all — the only HP feedback is the small
over-head percentage bar (`SHOW_MONSTER_HP` / `MonsterHealth`), which is a different, much less
legible affordance.

Everything below the emit site already exists. The packet struct is written and byte-verified on
five client versions (`libs/atlas-packet/field/clientbound/effect.go:125` `EffectBossHp`, with
`packet-audit:verify` markers for gms_v79/v83/v87/v95 and jms_v185 in
`libs/atlas-packet/field/clientbound/effect_test.go:10-18`). The body helper is wired
(`libs/atlas-packet/field/field_effect_body.go:57` `FieldEffectBossHpBody`). The `BOSS_HP` mode
byte is present in the `FieldEffect` writer's `operations` table in nine of eleven seed templates,
and in the dispatcher spec at `docs/packets/dispatchers/field_effect.yaml:14`. atlas-data already
parses both tag colors out of `Mob.wz` (`services/atlas-data/atlas.com/data/monster/reader.go:85-89`)
and serves them (`services/atlas-data/atlas.com/data/monster/rest.go:37-38`).

The gap is exactly two things: (a) atlas-channel has zero call sites for `FieldEffectBossHpBody`,
and (b) the channel's deliberately narrow monster projection
(`services/atlas-channel/atlas.com/channel/data/monster/rest.go:8-12`) declares only `Boss` and
`FixedDamage`, so the tag colors never reach the channel. This task closes both.

## 2. Goals

Primary goals:

- Broadcast the boss HP gauge to every character in the field whenever a boss with a non-zero WZ
  HP tag color takes damage.
- Clear the gauge when the boss dies.
- Show the current gauge to a character who enters the field while the boss fight is already in
  progress.
- Plumb `tag_color` / `tag_background_color` from atlas-data into atlas-channel's monster
  projection without changing any Kafka event contract.

Non-goals:

- No new packet codec, struct change, or version-gate work in `libs/atlas-packet` — `EffectBossHp`
  is already implemented and verified.
- No aggregation of multi-part bosses (Zakum arms, Horntail heads) into a single logical bar.
  Each monster with a tag color drives the gauge independently, exactly as Cosmic does.
- No changes to the existing `MonsterHealth` (`SHOW_MONSTER_HP`) behavior for bosses or for the
  party HP indicator. Both bars coexist.
- No `atlas-ui` changes.
- No general bring-up of the `FieldEffect` writer on the `gms_12` and `gms_92` templates (see
  §9 Open Questions — those templates lack the writer entirely, which is a pre-existing family-wide
  gap, not a `BOSS_HP` gap).

## 3. User Stories

- As a player fighting Zakum, I want a large HP gauge at the top of my screen so I can read the
  boss's remaining health at a glance during a chaotic fight.
- As a player, I want the gauge tinted with the boss's own colors so different bosses are visually
  distinguishable.
- As a player who enters the boss map after the fight has started, I want to see the boss's
  current HP immediately rather than waiting for the next hit to land.
- As a player, I want the gauge to reach zero and stop when the boss dies, rather than lingering
  at a stale value.
- As a player fighting an ordinary (non-tagged) monster, I want no top-of-screen gauge — the
  existing over-head bar is the correct affordance there.

## 4. Functional Requirements

### 4.1 Gating

- FR-1: A monster qualifies for the boss HP gauge if and only if **both** are true:
  - the monster's atlas-data `boss` attribute is `true`, and
  - the monster's atlas-data `tag_color` is non-zero.
- FR-2: A monster that fails either half of FR-1 MUST NOT produce any `BOSS_HP` field effect. In
  particular, a boss-flagged monster with `tag_color == 0` (e.g. a mob flagged boss for drop or
  status-immunity purposes but with no WZ HP tag) produces no gauge.
- FR-3: `tag_background_color` is passed through verbatim and does not participate in gating; a
  qualifying monster with `tag_background_color == 0` still gets a gauge.

### 4.2 Damage trigger

- FR-4: On a monster `DAMAGED` status event
  (`services/atlas-channel/.../kafka/consumer/monster/consumer.go` `handleStatusEventDamaged`),
  for a monster qualifying under FR-1, the channel MUST announce
  `FieldEffectBossHpBody(monsterId, currentHp, maxHp, tagColor, tagBackgroundColor)` on the
  `FieldEffectWriter` to **every session in the field**, where:
  - `monsterId` is the monster's **template** id (`monster.Model.MonsterId()`), not its unique id;
  - `currentHp` is the live monster's `Hp()` after the damage was applied;
  - `maxHp` is the live monster's `MaxHp()`.
- FR-5: The gauge broadcast is in addition to, and does not replace, the existing map-wide
  `MonsterHealth` broadcast on the `e.Body.Boss` branch. Both packets are sent for a qualifying
  monster.
- FR-6: The gauge broadcast MUST fire for every `DamageSource`, matching the existing HP-bar
  policy documented at `shouldEchoDamagePacket` ("The HP-bar packet, by contrast, is server-driven
  for every source"). It is NOT gated on `shouldEchoDamagePacket`.
- FR-7: The gauge broadcast MUST NOT block or delay the damage-number or `MonsterHealth` path.
  It runs on the same asynchronous dispatch discipline the surrounding handler already uses
  (`routine.Go`).

### 4.3 Death trigger

- FR-8: On a monster `DESTROYED` status event
  (`handleStatusEventDestroyed`), for a monster qualifying under FR-1, the channel MUST announce
  one final `FieldEffectBossHpBody(monsterId, 0, maxHp, tagColor, tagBackgroundColor)` to every
  session in the field, so the gauge visibly empties.
- FR-9: The final gauge broadcast MUST be resolved from monster state captured **before** the
  handler evicts the monster from the live mirror
  (`monster.GetLiveMirror().Remove(t, e.UniqueId)`). `StatusEventDestroyedBody` carries no
  `monsterId`, so the template id and max HP must come from the live monster looked up prior to
  eviction.
- FR-10: If the monster cannot be resolved at destroy time (already evicted, mirror miss), the
  handler MUST log at error level and continue — the destroy path's existing behavior is
  unaffected.

### 4.4 Field-entry trigger

- FR-11: When a character enters a field
  (`services/atlas-channel/.../kafka/consumer/map/consumer.go`, the existing per-session monster
  enumeration around `spawnMonsterForSession`), for each monster already in the field that
  qualifies under FR-1, the channel MUST send that character — and only that character — a
  `FieldEffectBossHpBody` carrying the monster's current HP.
- FR-12: The field-entry gauge MUST be sent **after** the corresponding `MonsterSpawn` packet for
  that monster, preserving the existing Spawn-before-anything-else ordering invariant documented
  at `kafka/consumer/map/consumer.go:755-772`.
- FR-13: If the field contains several qualifying monsters, one gauge is sent per qualifying
  monster, in the same order the monsters are enumerated. The client renders the last one; this
  is accepted behavior and matches the damage path, where the most recently damaged boss wins.
- FR-14: Field entry MUST NOT emit a gauge for a field with no qualifying monster, and MUST NOT
  perform a monster-data lookup for a field with no monsters.

### 4.5 Data resolution

- FR-15: `services/atlas-channel/atlas.com/channel/data/monster/rest.go` `RestModel` MUST gain
  `TagColor byte \`json:"tag_color"\`` and `TagBackgroundColor byte \`json:"tag_background_color"\``,
  with matching immutable accessors on `data/monster.Model` and both `Extract` and `Transform`
  updated symmetrically.
- FR-16: The channel MUST resolve tag colors by monster **template** id through the existing
  `data/monster` processor. It MUST NOT reach into another service's internals or add the fields
  to a Kafka event body.
- FR-17: A failed monster-data lookup MUST be treated as "does not qualify" — log and skip the
  gauge. It MUST NOT abort the damage, destroy, or field-entry handler.

## 5. API Surface

No new or modified HTTP endpoints.

Consumed, unchanged: `GET /monsters/{monsterId}` on atlas-data, whose `attributes` already include
`tag_color` and `tag_background_color` (`services/atlas-data/atlas.com/data/monster/rest.go:37-38`).
This task only widens atlas-channel's client-side projection of that existing response.

No Kafka topic, event type, or event body changes. `StatusEventDamagedBody` and
`StatusEventDestroyedBody` (`services/atlas-channel/.../kafka/message/monster/kafka.go:223-259`)
are untouched.

## 6. Data Model

No database entities, migrations, or persisted state.

Two fields added to an in-memory REST projection:

| Type | Field | Go type | Source |
|---|---|---|---|
| `channel/data/monster.RestModel` | `TagColor` | `byte` | atlas-data `tag_color` |
| `channel/data/monster.RestModel` | `TagBackgroundColor` | `byte` | atlas-data `tag_background_color` |
| `channel/data/monster.Model` | `tagColor` | `byte` | `Extract` |
| `channel/data/monster.Model` | `tagBackgroundColor` | `byte` | `Extract` |

Tenancy: all lookups flow through the existing `requests.Provider` path, which already carries
tenant via `context`. No new tenant scoping is introduced.

Packet payload (already implemented, `EffectBossHp`): `mode:byte`, `monsterId:int`,
`currentHp:int`, `maxHp:int`, `tagColor:byte`, `tagBackgroundColor:byte`.

## 7. Service Impact

**atlas-channel** (all changes)

- `data/monster/rest.go`, `data/monster/model.go` — add the two tag-color fields and accessors.
- `kafka/consumer/monster/consumer.go` — `handleStatusEventDamaged`: add the qualifying check and
  the map-wide gauge broadcast alongside the existing `MonsterHealth` announcement.
- `kafka/consumer/monster/consumer.go` — `handleStatusEventDestroyed`: capture monster state before
  eviction and send the zero-HP gauge.
- `kafka/consumer/map/consumer.go` — the character-enters-field monster enumeration: send the
  current gauge for qualifying monsters, after their spawn packet.
- A shared helper resolving "does this monster qualify, and with what colors" so the three call
  sites do not each re-derive the rule.

**atlas-data** — no change. Already parses and serves the tag colors.

**libs/atlas-packet** — no change. `EffectBossHp`, `FieldEffectBossHpBody`, and the `BOSS_HP`
dispatcher entry all exist and are verified.

**atlas-configurations** — no change in scope. See §9.

## 8. Non-Functional Requirements

- **NFR-1 (performance).** The damage path is the hottest path in the channel. Resolving tag colors
  MUST NOT add an unconditional per-damage-event HTTP round trip to atlas-data for ordinary mobs.
  The `e.Body.Boss` flag already on the event is a cheap pre-filter: only boss-flagged monsters
  should trigger a data lookup at all. Whether the lookup result additionally needs caching is a
  design decision (§9) — `data/monster.ProcessorImpl.GetById` is currently an uncached REST call.
- **NFR-2 (correctness under version drift).** The mode byte MUST be resolved from the tenant
  template's `operations` table via `WithResolvedCode`, never hard-coded.
  `FieldEffectBossHpBody` already does this.
- **NFR-3 (graceful degradation).** On a tenant whose template has no `FieldEffect` writer
  (`gms_12`, `gms_92` today), the announcement MUST fail soft — logged, no panic, no effect on the
  rest of the damage/destroy/entry handlers.
- **NFR-4 (multi-tenancy).** Every broadcast is scoped by the existing
  `sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId)` guard and the field-scoped session
  enumeration. No cross-tenant or cross-channel leakage.
- **NFR-5 (observability).** Each failure mode — data lookup failure, unresolvable monster at
  destroy time, announce failure — logs at error level with monster unique id and field.
- **NFR-6 (no regression).** The wire output for non-qualifying monsters MUST be byte-identical to
  today's, including the existing `MonsterHealth` and `MonsterDamage` behavior.

## 9. Open Questions

1. **`gms_12` and `gms_92` templates have no `FieldEffect` writer at all.** Verified by inspection
   of `services/atlas-configurations/seed-data/templates/`: nine of eleven templates carry
   `{"writer": "FieldEffect", "fname": "CField::OnFieldEffect", "options": {"operations": {... "BOSS_HP": 5 ...}}}`;
   `template_gms_12_1.json` and `template_gms_92_1.json` carry only `FieldEffectWeather`.
   `docs/packets/dispatchers/field_effect.yaml` likewise has no `gms_v92` mode column. Adding the
   whole `CField::OnFieldEffect` writer to those two versions is a version bring-up task, not a
   boss-HP task — but it means the feature is silently absent on those tenants. Decide in design:
   flag-and-defer (recommended), or widen scope to add the writer + `BOSS_HP` mode for `gms_v92`.
   Note `gms_v48` also lacks `REWARD_RULLET` but does have `BOSS_HP`, so it is fine.
2. **Caching the monster-data lookup.** `data/monster.ProcessorImpl.GetById` is an uncached REST
   call. Boss fights produce a high damage-event rate. Design must decide between (a) relying on
   the `e.Body.Boss` pre-filter alone, (b) a small per-tenant template-id cache in the channel, or
   (c) reading tag colors off the live monster mirror if it can be made to carry them.
3. **Destroy-time monster resolution.** `StatusEventDestroyedBody` has no `monsterId`. Design must
   pick the source for template id and max HP before eviction — the live mirror
   (`monster.GetLiveMirror()`), or `monster.NewProcessor(...).GetById(e.UniqueId)` called before
   `Remove`. Ordering against the existing eviction calls matters.
4. **Field-entry call site.** The exact hook — inside `spawnMonsterForSession` versus a separate
   pass after the monster enumeration completes — affects FR-12's ordering guarantee and how many
   times per entry the data lookup runs.
5. **Live smoke coverage.** Which boss and which client version to smoke-test against. Zakum
   (`8800002`) and Horntail (`8810018`) are the reference mobs named in the WZ data.

## 10. Acceptance Criteria

- [ ] `channel/data/monster.RestModel` and `.Model` carry `TagColor` / `TagBackgroundColor`, with
      `Extract` and `Transform` symmetric, covered by a unit test that unmarshals an atlas-data
      monster payload containing both fields.
- [ ] A single shared qualification helper implements FR-1 (boss flag AND `tagColor != 0`) and is
      used by all three call sites.
- [ ] Unit test: a boss-flagged monster with `tagColor == 0` produces no `BOSS_HP` announcement.
- [ ] Unit test: a non-boss monster with a non-zero `tagColor` produces no `BOSS_HP` announcement.
- [ ] Unit test: a qualifying monster on `DAMAGED` produces a `BOSS_HP` announcement to every
      session in the field, carrying the monster **template** id, post-damage `Hp()`, `MaxHp()`,
      and both tag colors.
- [ ] Unit test: a qualifying monster on `DAMAGED` still produces the existing map-wide
      `MonsterHealth` announcement (FR-5 non-regression).
- [ ] Unit test: `DAMAGED` from a `DamageSource` for which `shouldEchoDamagePacket` is false still
      produces the gauge (FR-6).
- [ ] Unit test: a qualifying monster on `DESTROYED` produces exactly one `BOSS_HP` announcement
      with `currentHp == 0`, resolved before live-mirror eviction.
- [ ] Unit test: a character entering a field containing a qualifying monster receives a `BOSS_HP`
      gauge for it, ordered after that monster's `MonsterSpawn`, and addressed only to that
      character.
- [ ] Unit test: a character entering a field with no qualifying monster receives no `BOSS_HP`
      packet and triggers no monster-data lookup.
- [ ] Unit test: a monster-data lookup failure on any of the three paths logs and skips the gauge
      without aborting the handler.
- [ ] Byte-level assertion that the encoded `BOSS_HP` payload matches `EffectBossHp`'s verified
      layout with the mode byte resolved from the tenant `operations` table, not hard-coded.
- [ ] Live smoke on a boss with a non-zero `hpTagColor`: the top-of-screen gauge appears on first
      hit, tracks damage, empties on kill, and is visible to a player who enters mid-fight.
- [ ] Live smoke: an ordinary mob produces no top-of-screen gauge and its over-head bar is
      unchanged.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] The `gms_12` / `gms_92` missing-`FieldEffect`-writer finding is recorded (either resolved in
      scope or written up as a follow-up with the evidence from §9.1).
