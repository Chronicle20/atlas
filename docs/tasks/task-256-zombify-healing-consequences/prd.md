# Zombify Healing Consequences — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

`UNDEAD` (the ZOMBIFY disease) is already applied to players correctly. Monster skill
type 133 maps to the `"UNDEAD"` name (`libs/atlas-constants/monster/skill.go:44,132-133`),
`atlas-buffs` accepts it as a disease (`services/atlas-buffs/atlas.com/buffs/character/immunity.go:10`),
`libs/atlas-constants/character/temporary_stat.go:122` defines the stat type, and the
packet layer encodes it as a two-state diseased stat
(`libs/atlas-packet/model/character_temporary_stat.go:259,1136,1213`). Cure semantics
are also already correct: Priest Dispel deliberately excludes `UNDEAD`
(`services/atlas-channel/atlas.com/channel/skill/handler/dispel/dispel.go:31-42`),
SuperGM Heal+Dispel purges it
(`services/atlas-channel/atlas.com/channel/skill/handler/healdispel/healdispel.go:44`),
and no consumable cure spec maps to it (`collectCureTypes`,
`services/atlas-consumables/atlas.com/consumables/consumable/processor.go:129-136`).

What is missing is the *consequence*. In the reference implementation, being zombified
is what makes the disease matter: recovery potions restore half as much, and a Cleric
Heal cast by a zombified caster deals damage instead of healing. Atlas has zero reads of
zombify state on any healing path. `computeEffectPlan`
(`services/atlas-consumables/atlas.com/consumables/consumable/processor.go:163-230`)
computes flat `hp` and `hpR` at full value with no knowledge of the character's buffs,
and the Cleric Heal handler
(`services/atlas-channel/atlas.com/channel/skill/handler/heal/heal.go:126-143`) heals at
full value. Today ZOMBIFY is a cosmetic icon: a player can drink through it with no
penalty at all.

This task ports the healing consequence. The reference behavior is
`server/StatEffect.java:1402-1436` (`calcHPChange`) in the Cosmic source tree, read
directly rather than from memory:

```java
if (hp != 0) {
    if (!skill) {
        ...
        if (applyfrom.hasDisease(Disease.ZOMBIFY)) {
            hpchange /= 2;
        }
    } else { // assumption: this is heal
        float hpHeal = (applyfrom.getCurrentMaxHp() * (float) hp / (100.0f * affectedPlayers));
        hpchange += hpHeal;
        if (applyfrom.hasDisease(Disease.ZOMBIFY)) {
            hpchange = -hpchange;
            hpCon = 0;
        }
    }
}
if (hpR != 0) {
    hpchange += (int) (applyfrom.getCurrentMaxHp() * hpR) / (applyfrom.hasDisease(Disease.ZOMBIFY) ? 2 : 1);
}
```

Note that the line originally cited for this work — `client/Character.java:8963-8970`
(`applyHpMpChange`) — is only the *death-permission* half of the reference behavior:

```java
boolean zombify = hasDisease(Disease.ZOMBIFY);
...
boolean cannotApplyHp = hpchange != 0 && nextHp <= 0 && (!zombify || hpCon > 0);
```

That clause exists because the reference `applyHpMpChange` normally *vetoes* an HP change
that would drop the player to 0, and the zombify case has to opt back out of that veto so
a negated heal can kill. Atlas has no such veto: `ChangeHP`
(`services/atlas-character/atlas.com/character/character/processor.go:1373-1409`) clamps
via `enforceBounds`, writes the result, and emits `DIED` whenever the adjusted value is 0.
A negative delta already kills. **No change to `atlas-character` is required by this task**
— the death-permission half is already Atlas's default behavior. All the work lives in the
two effect-computation sites.

## 2. Goals

Primary goals:

- A zombified character who drinks an HP-restoring consumable recovers half the HP a
  non-zombified character would, for both the flat `hp` spec and the `hpR` percentage spec.
- A zombified Cleric casting Heal deals the computed heal amount as damage to every
  recipient of that cast instead of healing them, and that damage can kill.
- The zombify determination is read from live buff state at effect time, not cached or
  inferred.
- Existing non-zombified behavior is byte-for-byte unchanged, including the existing
  per-recipient max-HP clamp on Heal and the existing HP/MP ordering in `ApplyItemEffects`.

Non-goals:

- Changing how `UNDEAD` is applied, expired, or immunity-gated. `atlas-buffs` disease
  handling is correct and untouched.
- Changing the Priest Dispel or SuperGM Heal+Dispel cure sets. Both already match the
  reference: Dispel excludes `UNDEAD`, Heal+Dispel purges it.
- MP consequences. The reference gates zombify on HP only; `mp` and `mpR` are unaffected.
- Monster-side undead (`Monster.Undead()` in `atlas-monsters` / `atlas-data`) — an
  unrelated concept (Heal damages undead *mobs*), not player zombify.
- Chakra, Recovery Aura, MP Recovery, and SuperGM Heal+Dispel. See §4.4 for why each is
  out of scope and what would have to be decided before including it.
- An `Alchemist` recovery multiplier. Atlas has no alchemist modifier on any recovery path
  today (grep: `HermitAlchemist` appears only as a skill-id constant), so the reference's
  `alchemistModifyVal` ordering is not yet reachable.

## 3. User Stories

- As a player fighting a zombify-inflicting monster, I want my potions to visibly
  under-heal so that the debuff is a real threat I have to play around rather than an
  icon I can ignore.
- As a player, I want a clear failure mode — half-heal, not a silent no-op — so I can tell
  the difference between "zombified" and "the potion did nothing".
- As a Cleric, I want my Heal to turn against my party while I am zombified so that
  curing or waiting out the debuff before casting is a real decision.
- As a party member, I want a zombified Cleric's Heal to be able to kill me so the
  mechanic carries the same stakes it does in the reference server.
- As a service developer, I want the zombify check to live in one named predicate per
  service so a future healing path can adopt it without re-deriving the rule.

## 4. Functional Requirements

### 4.1 Zombify predicate

- **FR-1.** A character is *zombified* when their live buff list from `atlas-buffs`
  contains at least one non-expired buff carrying a stat change whose type equals
  `character.TemporaryStatTypeUndead` (`"UNDEAD"`).
- **FR-2.** The predicate is evaluated per effect application, from a fresh read. No
  caching, no memoization across casts.
- **FR-3.** If the buff read fails (transport error, non-2xx, timeout), the character is
  treated as **not zombified**. Log at `Warn` with the character id and the error. Rationale:
  failing open preserves today's behavior on an infrastructure fault, so an
  `atlas-buffs` outage degrades to "the debuff has no effect" rather than "every Cleric
  Heal in the world becomes party-wide damage".

### 4.2 Consumables

- **FR-4.** `computeEffectPlan` takes the caller's zombified state as an input and halves
  every HP entry it produces when that state is true.
- **FR-5.** The flat `SpecTypeHP` value is halved: `int16(val) / 2` using Go integer
  division (truncation toward zero), matching the reference's `hpchange /= 2` on an `int`.
- **FR-6.** The `SpecTypeHPRecovery` value is halved on the same rule. The existing
  computation is `int16(math.Floor(float64(c.MaxHp()) * pct))`; the halving applies to the
  resulting integer, so the result is `floor(maxHp * pct) / 2` under integer division.
  The reference computes `(int)(maxHp * hpR) / (zombify ? 2 : 1)` — cast to int first,
  then integer-divide — which is the same order of operations.
- **FR-7.** A halved value that truncates to 0 produces **no** `ChangeHP` call. An HP entry
  of 0 is dropped from `plan.hpChanges` rather than dispatched as a zero-delta command.
  (Today a spec value of 0 is already filtered by the `val > 0` guard; this keeps that
  invariant after halving.)
- **FR-8.** `plan.mpChanges`, `plan.cureTypes`, `plan.statups`, and `plan.duration` are
  unaffected by zombify.
- **FR-9.** Ordering in `ApplyItemEffects` is unchanged: cure first, then HP/MP, then
  status-up buffs. In particular the zombify read must not be interleaved between the
  cure dispatch and the HP dispatch — the task-051 D3 ordering rationale documented at
  `processor.go:241-246` still holds.
- **FR-10.** The zombify read happens once per `ApplyItemEffects` invocation, before the
  plan is computed, and its result is threaded into `computeEffectPlan` as a plain
  boolean. `computeEffectPlan` stays pure and directly unit-testable, as its doc comment
  requires.

### 4.3 Cleric Heal

- **FR-11.** The Heal handler evaluates the zombify predicate for the **caster**
  (`characterId`), not for each recipient. This matches the reference, which reads
  `applyfrom.hasDisease(Disease.ZOMBIFY)` — `applyfrom` is the casting character. A
  zombified Cleric therefore damages the entire party, including non-zombified members;
  a non-zombified Cleric heals a zombified party member at full value.
- **FR-12.** When the caster is zombified, the per-target amount is negated:
  `perTarget` becomes `-perTarget`. The magnitude is the existing `HealAmount(...)`
  result, computed identically (same `e.HP()`, magic attack, INT, recipient count, and
  variance roll). Zombify changes only the sign.
- **FR-13.** The existing `appliedPerRecipient(perTarget, r)` clamp raises a
  correctness hazard and must be handled explicitly. It clamps a positive heal to the
  recipient's remaining headroom (`MaxHp - Hp`). Applied to a negative amount it would
  either return 0 or clamp wrongly. For a negated cast the per-recipient delta must
  instead be clamped downward so it never removes more than the recipient's current HP:
  the delta passed to `ChangeHP` is `-min(magnitude, r.Hp)`, expressed as an `int16` and
  saturated at `math.MinInt16`. A recipient already at 0 HP is skipped.
- **FR-14.** A negated Heal that brings a recipient to exactly 0 HP kills them.
  `atlas-character`'s `ChangeHP` already emits `DIED` at `adjusted == 0`
  (`processor.go:1396-1405`); no new death path is introduced. This is the intended
  behavior, matching the reference's `!zombify || hpCon > 0` opt-out.
- **FR-15.** Heal XP must not be awarded for a negated cast. `HealXp(perTarget, ...)` is
  derived from the applied amount; feeding it a negative `perTarget` would produce a
  negative or nonsensical award. When the caster is zombified, the XP block at
  `heal.go:145-156` is skipped entirely.
- **FR-16.** The skill-use broadcast is unchanged. Both `AnnounceSkillUse` and
  `AnnounceForeignSkillUse` still fire on a negated cast — the client plays the Heal
  animation either way; only the HP deltas differ.
- **FR-17.** The reference also forces `hpCon = 0` on a negated heal. In Atlas, cast-time
  HP cost is applied generically before handler dispatch
  (`services/atlas-channel/atlas.com/channel/skill/handler/common.go:137-138`,
  gated on `e.HPConsume() > 0`). Cleric Heal is expected to have `HPConsume == 0` in WZ
  data, which would make this clause a no-op. **The design phase must confirm the actual
  WZ `hpCon` for skill Heal across provisioned versions before deciding whether any
  suppression code is warranted.** If it is confirmed 0 on every version, record that
  finding and implement nothing; if it is non-zero on any version, the suppression must be
  implemented at the `common.go` site under a zombify-and-heal condition. Do not implement
  speculative suppression.

### 4.4 Explicitly deferred healing paths

Each of these restores HP and was considered. None is in scope for this task; each is
listed with the specific reason so a follow-up does not have to re-derive it.

- **Chakra** (`skill/handler/chakra`). The reference does **not** gate Chakra's
  `makeHealHP` contribution on zombify — it is added after the zombify branches in
  `calcHPChange`. Including it would be a deliberate divergence, not a port.
- **Recovery Aura** (`skill/handler/recoveryaura`). Restores over time from an aura
  rather than a single cast; the reference's per-cast `calcHPChange` model does not map
  onto it cleanly, and whether the aura owner's or the recipient's zombify state applies
  is an open design question.
- **MP Recovery** (`skill/handler/mprecovery`). MP only; the reference gates zombify on
  HP only.
- **SuperGM Heal+Dispel** (`skill/handler/healdispel`). It purges `UNDEAD` as part of the
  same cast (`healdispel.go:44`), so the ordering — dispel-then-heal versus
  heal-then-dispel — determines whether zombify could ever apply. It is a GM utility with
  intentional full-restore semantics; leaving it unconditional is the safe default.

## 5. API Surface

No new HTTP endpoints, request shapes, or Kafka message types.

One new **client** of an existing endpoint: `atlas-consumables` gains a read path to
`atlas-buffs`.

- **Endpoint (existing, unchanged):** `GET {BUFFS}/characters/{characterId}/buffs`
  — declared at `services/atlas-buffs/atlas.com/buffs/character/resource.go:23`.
- **Pagination:** the endpoint is paginated server-side (task-117). The existing
  `atlas-channel` client consumes it via `requests.DrainProvider`, which appends its own
  `page[number]` / `page[size]` params; the URL is built bare, not as a
  `requests.Request`. See
  `services/atlas-channel/atlas.com/channel/character/buff/requests.go:11-28` for the
  precedent to mirror.
- **Root URL key:** `requests.RootUrlFor(ctx, "BUFFS")`. `atlas-consumables` must have the
  corresponding service-root configuration available in its deployment environment; the
  design phase confirms whether it is already present or must be added.
- **Error cases:** any non-2xx, transport error, or drain failure resolves to
  "not zombified" per FR-3.

`atlas-channel` needs no new client — `buff.Processor.GetByCharacterId` already exists at
`services/atlas-channel/atlas.com/channel/character/buff/processor.go:21,74-76`.

## 6. Data Model

No new entities, no schema changes, no migrations.

Read-only consumption of the existing buff model. In `atlas-channel` the shape is
`buff.Model` with `Changes() []stat.Model` and `Expired() bool`
(`services/atlas-channel/atlas.com/channel/character/buff/model.go:21-52`), where
`stat.Model` is `{Type character.TemporaryStatType; Amount int32}`. `atlas-consumables`
already has an identical `stat.Model`
(`services/atlas-consumables/atlas.com/consumables/character/buff/stat/model.go`) and
will need the matching read-side `Model` plus its REST model.

Multi-tenancy is carried by `context.Context` on the outbound REST call, as with every
other cross-service read in these services. No tenant field is added to any struct.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-consumables` | `character/buff` gains `GetByCharacterId` plus the REST model, requests file, and mock entry (`character/buff/mock/processor.go` must stay in sync with the interface). `consumable/processor.go`: `computeEffectPlan` takes a `zombified bool`; `ApplyItemEffects` reads the predicate once and threads it in. |
| `atlas-channel` | `skill/handler/heal/heal.go`: read caster zombify state via the existing `buff.Processor.GetByCharacterId`, negate `perTarget`, apply the damage-side clamp (FR-13), skip the XP block (FR-15). Add the read as a replaceable function seam so the handler stays unit-testable offline, matching the `healDispelDeps` and `dispel.cancelByTypesFunc` precedents in sibling handlers. |
| `atlas-buffs` | None. Serves the existing endpoint unchanged. |
| `atlas-character` | None. `ChangeHP` already permits a negative delta to reach 0 and emit `DIED`. |
| `libs/atlas-constants` | None. `TemporaryStatTypeUndead` already exists. |
| `atlas-ui` | None. |

## 8. Non-Functional Requirements

- **Latency.** Each consumable use adds one synchronous REST round-trip to `atlas-buffs`,
  and each Cleric Heal cast adds one. Both are on paths that already perform multiple
  cross-service reads (Heal alone already issues one `atlas-effective-stats` call per
  recipient), so one additional single call per cast is proportionate. The read must not
  be issued per-recipient — one read per cast, for the caster only (FR-11).
- **Failure isolation.** Per FR-3 the failure mode is fail-open. A buff-service fault must
  never convert a Heal into party-wide damage, and must never block a potion from healing.
- **Observability.** Log at `Debug` when a zombify-modified effect is applied, including
  the character id and the pre/post amounts, so a support report of "my potion healed
  wrong" is diagnosable from logs alone. Log at `Warn` on a failed buff read (FR-3).
- **Multi-tenancy.** The buff read is tenant-scoped through `context.Context`, consistent
  with every other REST client in these services. No tenant-specific branching.
- **Determinism in tests.** `computeEffectPlan` must remain a pure function of
  `(character, consumable, zombified)` so the halving is pinnable by plain unit tests with
  no Kafka, REST, or clock. The Heal negation must be reachable through a function seam
  for the same reason.
- **Backward compatibility.** Every existing test in
  `services/atlas-consumables/.../consumable/processor_test.go` and
  `morph_coupon_test.go` that asserts `hpChanges` must continue to pass unchanged, because
  those cases are all non-zombified. If a signature change to `computeEffectPlan` requires
  touching them, the *expected values* must not move.

## 9. Open Questions

- **OQ-1 (must resolve in design).** What is Cleric Heal's WZ `hpCon` across all
  provisioned versions? FR-17 hinges on it. If it is 0 everywhere, no suppression code is
  written and the finding is recorded.
- **OQ-2 (must resolve in design).** Does `atlas-consumables`' deployment configuration
  already expose a `BUFFS` service root for `requests.RootUrlFor`? It currently only
  *produces* to the buff Kafka topic and has never read from the service.
- **OQ-3.** Should the negated-Heal death be attributed to the caster rather than
  `KillerTypeUnknown`? `ChangeHP` hard-codes `character2.KillerTypeUnknown` at
  `processor.go:1404`, so caster attribution would require a new `ChangeHP` variant. The
  proposed default is to accept `KillerTypeUnknown` and not widen the `atlas-character`
  surface.
- **OQ-4.** Should a zombified player see any client-side feedback distinguishing a
  half-heal from a full heal? The client renders the HP delta it is told about, so the
  reduced number is already visible; no additional packet is proposed.
- **OQ-5.** Does any other consumable classification reach HP restoration outside
  `ApplyItemEffects`? `usesStandardConsumer` (`processor.go:116-123`) admits
  classifications 200, 201, 202, 205, and transformation; anything else falls through to
  `ConsumeBare`. Design should confirm no HP-restoring item bypasses the plan.

## 10. Acceptance Criteria

Consumables:

- [ ] A zombified character drinking an item with `SpecTypeHP = 300` receives a single
      `ChangeHP` of `+150`; a non-zombified character receives `+300`.
- [ ] A zombified character with 1000 effective max HP drinking an item with
      `SpecTypeHPRecovery = 25` receives `+125`; non-zombified receives `+250`.
- [ ] An item carrying both `hp` and `hpR` produces two halved entries, in the existing
      order (flat first, then ratio).
- [ ] A halved value that truncates to 0 dispatches no `ChangeHP` call.
- [ ] `mp`, `mpR`, cure types, statups, and duration are identical zombified and not.
- [ ] `computeEffectPlan` remains pure and is unit-tested for both zombify states with no
      Kafka or REST.
- [ ] A failed `atlas-buffs` read logs at `Warn` and yields full-value healing.

Cleric Heal:

- [ ] A non-zombified caster's Heal produces identical `ChangeHP` deltas to today for
      every recipient, including the max-HP headroom clamp.
- [ ] A zombified caster's Heal produces a negative delta for every recipient, of the same
      magnitude the non-zombified cast would have produced before clamping.
- [ ] A recipient with less current HP than the damage magnitude receives exactly
      `-currentHp`, not the full magnitude.
- [ ] A recipient at 0 HP is skipped (no `ChangeHP` call).
- [ ] A recipient brought to 0 HP triggers the existing `DIED` emission from
      `atlas-character` — asserted at the `ChangeHP` call boundary, not by re-testing
      `atlas-character`.
- [ ] No experience is awarded on a negated cast.
- [ ] `AnnounceSkillUse` and `AnnounceForeignSkillUse` still fire on a negated cast.
- [ ] The zombify state read is for the caster only, and issued exactly once per cast
      regardless of recipient count — asserted by a call-counting seam.
- [ ] A zombified party member healed by a non-zombified caster receives a normal
      positive heal.

Repository gates:

- [ ] OQ-1 and OQ-2 are answered in `design.md` with cited evidence.
- [ ] `tools/verify.sh` (flagless) exits 0.
- [ ] Code review completed before the PR is opened.
