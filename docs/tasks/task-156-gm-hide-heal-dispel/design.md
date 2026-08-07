# SuperGM Skills: Hide + Heal & Dispel — Design

Task: task-156-gm-hide-heal-dispel
Status: Draft for review
Created: 2026-07-10
Depends on PRD: `docs/tasks/task-156-gm-hide-heal-dispel/prd.md`

---

## 1. Summary

Implement two SuperGM active skills end-to-end in `atlas-channel`:

- **Heal + Dispel** (`SuperGmHealDispelId` = `9101000`) — a registered per-skill
  handler that restores HP/MP and cancels disease debuffs for **every** player in
  the caster's map.
- **Hide** (`SuperGmHideId` = `9101004`) — a registered per-skill handler that
  **toggles** GM invisibility using a persistent buff and a server-side spawn
  suppression gate.

Both follow the established `channelhandler.Register(skillId, Apply)` pattern
(reference: Cleric Heal `skill/handler/heal`, Mystic Door `skill/handler/mysticdoor`,
and the mount toggle `skill/handler/mount.go`). Both are dispatched from the generic
`UseSkill` orchestrator after it has validated the cast and consumed MP/cooldown.

### Grounding that reshapes the PRD

Investigation of the code changed three assumptions the PRD carried. These are the
load-bearing findings behind the design:

1. **`atlas-data` requires zero changes.** The PRD's Service Impact anticipated two
   `atlas-data` edits (a `SuperGmHideId` statup branch and surfacing HP/MP recovery).
   Neither is needed:
   - The recovery fields (`hp`, `mp`, `hpR`, `mpR`) are **already** parsed by
     `getEffect` (atlas-data `skill/reader.go:140-158`), serialized in the REST DTO,
     and deserialized into the channel-side `effect.Model` fields
     (`data/skill/effect/rest.go:92-95`). Only the channel-side **accessor methods**
     `MP()`, `HpR()`, `MpR()` are missing (`data/skill/effect/model.go`); `HP()` and
     `StatUps()` already exist. This is a pure channel-side projection gap.
   - `SuperGmHideId` must **stay** with no statup mapping in the reader. Giving it a
     statup would make the generic buff-apply path in `common.go:107-111` (guarded on
     `len(e.StatUps()) > 0`) fan the hide stat out to **party members in the map** —
     exactly wrong. The hide handler applies its own buff to the caster only, so the
     reader stays untouched (same reason the mount reader-injection is intercepted
     before the generic path).

2. **FR-16 (isCategory1) needs no action.** `SuperGmHealDispelId` is correctly in
   `isCategory1` (`reader.go:496`), which forces the *Skill's* `buff=false`. That flag
   only governs the buff boolean; the effect's `hp/mp/hpR/mpR` fields are populated
   **unconditionally** by `getEffect` regardless of it. The handler reads recovery via
   the new accessors; `SuperGmHealDispelId` stays in `isCategory1`.

3. **`ChangeMP` already exists (OQ-5 resolved).** A full `ChangeHP`/`ChangeMP`
   command/producer/processor triple exists channel-side (`character/producer.go:57-83`,
   `character/processor.go:271-277`, `kafka/message/character/kafka.go:16-17`). No new
   HP/MP command work is required — the heal component reuses both.

Net effect: the only services touched are **`atlas-channel`** (all new code) and
**`libs/atlas-constants`** (no change — all ids/types already exist). `atlas-data` and
`atlas-buffs` are **not modified**.

---

## 2. Open questions — resolved

| OQ | Decision | Grounding |
|----|----------|-----------|
| **OQ-1** hide stat: `DARK_SIGHT` vs `SNEAK` | **`DARK_SIGHT`** | The v83 client (`MapleStory_dump.exe`) has distinct `CUser::IsDarkSight` (stat @+468, `0x4f0d45`) and `CUser::IsSneak` (stat @+2644, `0x4f0d64`) predicates. Both are read by the local render path (`CUserLocal::Update`) and by `CMob::IsTargetInAttackRange` (`0x66a517`) — so both render local translucency and make the caster untargetable by mobs (supports FR-14.1). `DARK_SIGHT` is the stat the atlas WZ reader **already** produces (Rogue Dark Sight, `reader.go:287`), so it has a proven server→client apply/cancel pipeline. `SNEAK` (`TemporaryStatTypeSneak`) is slot-reserved in the packet ordering table (`character_temporary_stat.go:214`, no-op foreign encode) but is **never originated** by any effect or producer — choosing it means building a stat pipeline from scratch with no precedent. Both foreign-encode as no-ops, so foreign visibility is identical; local render and existing infra decide it. |
| **OQ-2** hide state storage: buff (A) vs flag (B) | **Option A — a `DARK_SIGHT` buff sourced from `SuperGmHideId`, `duration = math.MaxInt32`** | The PRD prefers A "unless the buff's foreign-visibility semantics conflict." They do not: `DARK_SIGHT` foreign-encodes as a no-op, and the caster is despawned regardless, so the buff's foreign broadcast is benign. Mounts are an exact precedent (`mount.go:18-22`): a toggle skill using an effectively-permanent buff because atlas-buffs **rejects `duration <= 0`** (`buff/model.go:97`, `ErrInvalidDuration`) — there is no "never expires" path, so `math.MaxInt32` (~24.8 days) is the convention. Buffs persist across map changes (per-character, not per-map), and `spawnCharacterForSession` already fetches the buff list, so the suppression gate reads state it already has. |
| **OQ-3** hidden-cast broadcast | **Suppress the foreign skill-use animation when the caster is hidden** (FR-17 default) | A foreign animation for an invisible caster leaks position. Self animation always fires; foreign fires only when the caster is visible. |
| **OQ-4** self-Heal while hidden | **Benefits apply without revealing the caster** | Heal/dispel are delivered via `ChangeHP`/`ChangeMP` and `CancelByTypes` commands keyed by recipient character id — none of these spawn or reference the caster's avatar to recipients. A hidden caster stays hidden. |
| **OQ-5** `ChangeMP` exists? | **Yes — reuse it** | See §1.3. |

---

## 3. Architecture

### 3.1 Dispatch & registration (FR-1, FR-2, FR-3)

Two new subpackages, each registering in `init()` and blank-imported from
`skill/handler/registrations/registrations.go`:

```
skill/handler/healdispel/   -> channelhandler.Register(skill2.SuperGmHealDispelId, Apply)
skill/handler/hide/         -> channelhandler.Register(skill2.SuperGmHideId, Apply)
```

Each `Apply` matches the `channelhandler.Handler` curried signature exactly
(`skill/handler/registry.go:18-24`):

```go
func(l logrus.FieldLogger) func(ctx context.Context) func(
    wp writer.Producer, f field.Model, characterId uint32,
    info packetmodel.SkillUsageInfo, e effect.Model,
) error
```

Both are dispatched by `UseSkill` (`common.go:117-121`) after MP/cooldown handling.
Neither skill carries statups in the WZ reader, so the generic buff-apply path
(`common.go:107-111`) is skipped for them and no `common.go` short-circuit (as mounts
need) is required — a plain registered handler suffices.

### 3.2 SuperGM gate (FR-4, FR-4.1)

Both handlers, before any effect, load the caster and reject non-SuperGM casters:

```go
if !job.IsA(job.Id(c.JobId()), job.SuperGmId) {
    l.Warnf("Character [%d] cast SuperGM skill [%d] without SuperGM job; rejecting.", characterId, info.SkillId())
    return nil   // no effect, no error surfaced to other players
}
```

`job.SuperGmId = Id(910)` and `job.IsA` (`job/model.go:31`) resolve so that only job
910 matches — plain GM (`GmId = 900`) does **not** (`Is(900, 910)` is false). Return
`nil` (not an error) so the cast is a silent no-op for a non-SuperGM; the rejection is
warn-logged.

### 3.3 Heal + Dispel handler (`skill/handler/healdispel`)

Flow for `Apply`:

1. **Gate** SuperGM (§3.2). Load caster via `character.NewProcessor(...).GetById`.
2. **Compute hidden state** for the broadcast decision: `hidden := isHidden(caster)`
   (the §3.4 predicate). Used only for FR-17.
3. **Select recipients** = every player in the caster's map, via a new map-wide
   selector (§3.6). Includes the caster.
4. **Per recipient**, restore HP and MP and dispel diseases (FR-6, FR-6.1, FR-7):
   - Fetch effective stats (`effective_stats.NewProcessor(...).GetByCharacterId(world, channel, recipientId)`)
     for effective `MaxHp`/`MaxMp`, mirroring how Cleric Heal clamps (`heal.go:87-122`).
   - Restore amount mirrors the established consumables recovery formula
     (`consumables/consumable/processor.go:126-141`): flat + ratio.
     - `restoreHp = int(e.HP()) + floor(effMaxHp * e.HpR())`
     - `restoreMp = int(e.MP()) + floor(effMaxMp * e.MpR())`
     - Clamp each delta to `[0, effMax - current]` (FR-6.1), then
       `cp.ChangeHP(f, recipientId, delta)` / `cp.ChangeMP(f, recipientId, delta)`.
       (`ChangeHP`/`ChangeMP` emit Kafka deltas; the character service also clamps, but
       clamping here matches the Heal handler and avoids over-emitting.)
   - Dispel: `dispelProc.CancelByTypes(f, recipientId, diseaseTypes)` (§3.5), where
     `diseaseTypes` is the 11-string disease set. The resulting `EXPIRED` events are
     broadcast to self + foreign by the existing buff-status consumer
     (`kafka/consumer/buff/consumer.go:93-127`) — the handler does **not** hand-roll the
     cancel broadcast (FR-8).
5. **No experience** (FR-9): never call `AwardExperience`. (This is the key divergence
   from Cleric Heal, which has an undead/party XP path.)
6. **Per-recipient failure isolation** (FR-10): each recipient's HP/MP/dispel runs in a
   `for` body that logs and `continue`s on error; one failure never aborts the others.
7. **Skill-use broadcast** (FR-17): `AnnounceSkillUse` to the caster's session always;
   `AnnounceForeignSkillUse` to other sessions **only if `!hidden`**.

**Recovery-value verification (execute-time gate).** WZ data is not in the repo
(atlas-data mounts it at runtime), so the exact `9101000` recovery fields (`hp`/`hpR`
flat-vs-ratio split) are **unverified here**. The flat+ratio formula is robust to
either shape (a zero field contributes nothing), but the plan/execute phase MUST
confirm the actual WZ values against live WZ data per CLAUDE.md before claiming the
heal magnitude correct. This is a required verification step, not a guess to be shipped.

### 3.4 Hide handler (`skill/handler/hide`)

Structurally modeled on `mount.go` (toggle via a persistent source-keyed buff), plus
explicit spawn/despawn broadcasts that mounts don't need.

**State model.** "Hidden" ⇔ the caster has an active buff whose `SourceId() ==
int32(SuperGmHideId)` and `!Expired()` — the exact `isMounted` shape
(`mount.go:171-182`):

```go
func isHidden(bp buff.Processor, characterId uint32) (bool, error) {
    bs, err := bp.GetByCharacterId(characterId)
    if err != nil { return false, err }
    for _, b := range bs {
        if b.SourceId() == int32(skill2.SuperGmHideId) && !b.Expired() {
            return true, nil
        }
    }
    return false, nil
}
```

Keying on **`SourceId`, not the stat type**, is essential: Rogue Dark Sight also
produces a `DARK_SIGHT` stat but must remain visible to players. Only a
`SuperGmHideId`-sourced buff means "GM-hidden."

**Toggle flow for `Apply`:**

1. **Gate** SuperGM (§3.2). Load caster.
2. `hidden, _ := isHidden(bp, characterId)`.
3. **If not hidden → hide ON:**
   - `bp.Apply(f, characterId, int32(SuperGmHideId), info.SkillLevel(), math.MaxInt32,
     []statup.Model{ statup.NewModel(string(charconst.TemporaryStatTypeDarkSight), 1) })(characterId)`.
     The `DARK_SIGHT` amount must be **non-zero** — `CUser::IsDarkSight` tests the stat
     `!= 0`. The self buff-give (from the `APPLIED` status event) renders the caster's
     local hidden translucency; the foreign buff-give is a no-op encode.
   - **Despawn the caster from every other session in the map** (FR-12): broadcast
     `CharacterDespawn(characterId)` via `_map...ForOtherSessionsInMap`. See §3.7 for
     the exported broadcast helper.
4. **If hidden → hide OFF:**
   - `bp.Cancel(f, characterId, int32(SuperGmHideId))` — the `EXPIRED` self buff-cancel
     removes the local `DARK_SIGHT` stat.
   - **Spawn the caster into every other session in the map** (FR-13): broadcast a
     `CharacterSpawn` for the caster via `_map...ForOtherSessionsInMap`. See §3.7.
5. **Skill-use broadcast** (FR-17, using the **pre-toggle** `hidden` value): self
   `AnnounceSkillUse` always; foreign `AnnounceForeignSkillUse` only if `!hidden`
   (i.e. suppressed both when hiding — about to vanish — and when revealing from a
   hidden state, where the spawn itself restores visibility).

**Ordering note.** On hide-ON the async `APPLIED` foreign buff-give (Kafka round-trip)
arrives after the synchronous despawn and references an already-despawned character →
the client ignores it (and it is a no-op encode regardless). On hide-OFF the async
`EXPIRED` foreign buff-cancel is likewise benign. The authoritative visibility change is
the handler's synchronous despawn/spawn plus the §3.8 suppression gate — never the buff
broadcast.

**Persistence across maps (FR-14).** The `math.MaxInt32` buff persists per-character
across map changes. When the hidden GM warps, the §3.8 suppression gate (which reads the
live buff) keeps them hidden in the new map. Natural expiry is a non-issue at
`math.MaxInt32` (~24.8 days); toggle-off is the canonical reveal, exactly as mounts.

### 3.5 New channel-side `CancelByTypes` buff producer (dispel, FR-8)

There is currently **no** `CancelByTypes` producer/processor on the channel side — only
`APPLY` and `CANCEL` (`kafka/message/buff/kafka.go:12-16`, `character/buff/producer.go`,
`character/buff/processor.go`). `atlas-buffs` already consumes `CANCEL_BY_TYPES`
(`kafka/consumer/character/consumer.go:81-89` → `CancelByStatTypes`) and emits `EXPIRED`
status events, and `atlas-consumables` has a working producer that is the drop-in
template (`consumables/character/buff/producer.go:57-72`). Add, mirroring it:

- `kafka/message/buff/kafka.go`: `CommandTypeCancelByTypes = "CANCEL_BY_TYPES"` and
  `CancelByTypesCommandBody{ Types []string }`.
- `character/buff/producer.go`: `CancelByTypesCommandProvider(f, characterId, types)`.
- `character/buff/processor.go`: `CancelByTypes(f, characterId, types) error` on the
  interface + impl, publishing to `COMMAND_TOPIC_CHARACTER_BUFF`.

The disease set is listed explicitly channel-side (the atlas-buffs `diseaseStatTypes`
map at `buffs/character/immunity.go:7-20` is unexported):

```go
var diseaseStatTypes = []string{
    "STUN", "POISON", "SEAL", "DARKNESS", "WEAKEN", "CURSE",
    "SEDUCE", "CONFUSE", "UNDEAD", "SLOW", "STOP_PORTION",
}
```

These are `character.TemporaryStatType` string values; the plan should reference the
`libs/atlas-constants/character` constants where they exist rather than bare literals
(DOM-21). Broadcast of the resulting cancels is entirely handled by the existing
buff-status consumer — no new broadcast code.

### 3.6 Map-wide recipient selector (FR-5)

Existing selectors in `skill/handler/recipients.go`
(`SelectInRangePartyMembers`, `SelectPartyMembersInMap`) are **party-bitmap scoped**
(they reject `memberBitmap == 0` and filter by party slot). Heal + Dispel needs **all**
players in the map irrespective of party. Add a new selector:

```go
func SelectAllCharactersInMap(l, ctx, f field.Model) []PartyRecipient
```

- Source ids from `inMapCharacterIdsFunc` (the live-session set already used by
  `recipients.go`, backed by `_map...ForSessionsInMap`) — this is the same "who is in
  the map" source the spawn paths use (`GetCharacterIdsInMap`).
- For each id, load the character (`loadPartyMemberFunc` / `character.GetById`) to
  populate `Hp`/`MaxHp`; the handler augments `MaxHp`/`MaxMp` from effective stats
  per §3.3.
- No bitmap filter, no rectangle. Returns the caster too (the handler does not skip it).

Reuse the existing `PartyRecipient` value type and its function seams so the selector is
unit-testable offline like the current ones.

### 3.7 Exported hide/reveal broadcast helpers

The spawn-packet construction lives in the map consumer's **unexported**
`spawnCharacterForSession` (`kafka/consumer/map/consumer.go:427`) and `despawnForSession`
(`:464`). The hide handler is in a different package and must not reach into consumer
internals (CLAUDE.md service-boundary rule). Resolution: expose a thin, testable broadcast
API the handler calls through a function seam (mount-style `deps`):

- Add exported `DespawnCharacterInMap(l, ctx, wp)(f, characterId) error` and
  `SpawnCharacterInMap(l, ctx, wp)(f, characterId) error` in the map consumer package
  (or a small `map` broadcast helper), each wrapping `ForOtherSessionsInMap` over the
  existing (now-reused) `despawnForSession` / `spawnCharacterForSession` operators. This
  keeps spawn-body construction (buffs + guild + `enteringField=false`) in one place so
  the hide reveal produces a **byte-identical** spawn packet to normal map entry
  (important for FR-8/client stability), rather than the handler re-deriving it.
- The hide handler depends on these via injected seams (`despawnFromOthers`,
  `spawnToOthers`) exactly as `mount.go` injects `applyBuff`/`cancelBuff`, so the toggle
  logic is unit-testable without Kafka/REST/session.

### 3.8 Spawn suppression gate (FR-12, FR-14, §8 race-safety)

`spawnCharacterForSession` (`consumer.go:427-442`) is the **single choke point** through
which every character-spawn passes (both `enterMap` → others, and `SpawnForSelf` →
entering viewer), and it **already fetches the spawned character's buff list**
(`buff.NewProcessor(...).GetByCharacterId(c.Id())` at `:432`). Insert the gate there:

```go
bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(c.Id())
...
if isHiddenFromBuffs(bs) {   // any active buff with SourceId == SuperGmHideId
    return nil               // suppress: do not spawn this character to viewer s
}
return session.Announce(...)(charpkt.CharacterSpawnWriter)(writer.CharacterSpawnBody(c, bs, g, enteringField))(s)
```

Because the gate lives in the same function that emits the spawn — and reads the buff
list already loaded there — a player entering the map while a GM is hidden is filtered
**before** the spawn is ever written (satisfies §8: "the suppression check MUST live in
the same broadcast path that emits the spawn," no momentary visibility, no best-effort
follow-up despawn). `c` is never the viewer themselves in either loop
(`k != s.CharacterId()`), so self-view is never suppressed.

**Scope decision:** v1 hides the GM from **all** other viewers (no "GMs see hidden GMs"
exception), matching "invisible to other players." That avoids a per-spawn viewer-job
lookup. Extension point noted: gate on the viewer's job to let privileged viewers see
hidden GMs later. Plain `GmHideId` (`9001004`) is out of scope but would "fall out for
free" by adding it to the source-id set.

---

## 4. Data flow

**Heal + Dispel cast:**

```
client UseSkill(9101000)
  -> socket/handler/character_skill_use validates ownership/level/cost
  -> UseSkill (common.go): consume MP, apply cooldown; generic buff path skipped
       (no statups); dispatch -> healdispel.Apply
  -> gate SuperGM
  -> SelectAllCharactersInMap(f)                       [FR-5]
  -> per recipient:
       effective_stats.GetByCharacterId -> effMaxHp/effMaxMp
       ChangeHP(delta), ChangeMP(delta)  (clamped)     [FR-6/6.1]  -> COMMAND_TOPIC_CHARACTER
       buff.CancelByTypes(diseaseTypes)                [FR-7/8]    -> COMMAND_TOPIC_CHARACTER_BUFF / CANCEL_BY_TYPES
            -> atlas-buffs cancels -> EXPIRED events -> channel buff consumer broadcasts cancel (self+foreign)
  -> AnnounceSkillUse (self); AnnounceForeignSkillUse (only if !hidden)   [FR-17]
  (no AwardExperience)                                   [FR-9]
```

**Hide toggle cast:**

```
client UseSkill(9101004)
  -> UseSkill: generic buff path skipped (no statups); dispatch -> hide.Apply
  -> gate SuperGM; hidden := isHidden(caster)
  IF !hidden (hide ON):
      buff.Apply(DARK_SIGHT, source=9101004, dur=MaxInt32)  -> COMMAND_TOPIC_CHARACTER_BUFF / APPLY
           -> APPLIED event -> self buff-give (local translucency)
      DespawnCharacterInMap(f, caster)  -> CharacterDespawn to other sessions   [FR-12]
  IF hidden (hide OFF):
      buff.Cancel(source=9101004)                            -> EXPIRED event -> self buff-cancel (local reveal)
      SpawnCharacterInMap(f, caster)    -> CharacterSpawn to other sessions     [FR-13]
  -> AnnounceSkillUse (self); AnnounceForeignSkillUse (only if pre-toggle !hidden)  [FR-17]

Later map entry while hidden:
  enterMap / SpawnForSelf -> spawnCharacterForSession -> reads caster buffs
       -> SuperGmHide-sourced buff present -> suppress spawn to non-caster viewers   [FR-14/§8]
```

---

## 5. Alternatives considered

- **Hide state as a character/session flag (Option B)** instead of a buff. Rejected for
  v1: it duplicates persistence and local-render infrastructure the buff system already
  provides (a new registry, a hand-built self temporary-stat packet), for no correctness
  gain — Option A's foreign broadcasts are benign (no-op encode) and the `math.MaxInt32`
  duration removes the only edge (natural expiry). Revisit only if a future requirement
  (e.g. hide state surviving channel changes, which a channel-local flag would *lose* but
  atlas-buffs *keeps*) changes the calculus. Notably, the buff already wins on
  channel-change persistence.
- **Consumer-driven despawn/spawn** (extend the buff-status `APPLIED`/`EXPIRED` consumer
  to despawn/spawn on `SuperGmHide` source) instead of handler-driven. This would unify
  toggle-off and natural-expiry into one reveal path. Rejected: it adds async latency
  between cast and visibility change, couples the generic buff consumer to hide
  semantics, and the PRD explicitly specifies handler-driven despawn/spawn (FR-12/FR-13).
  With `math.MaxInt32` there is no natural expiry to unify against.
- **Extending the `heal` package** to share its `appliedPerRecipient`/`HealAmount`
  clamp helpers. Rejected: those helpers encode Cleric-specific magic-attack/INT and
  undead-XP formulas that a flat+ratio GM restore does not want. `healdispel` replicates
  only the small `effectiveMax…OrBase` clamp idiom, keeping the two skills decoupled.

---

## 6. Testing strategy

Following `superpowers:test-driven-development` and the project Builder pattern (no
`*_testhelpers.go`), all logic behind function seams so tests run offline (Kafka/REST/
session mocked), mirroring `mount.go`'s `deps` and `recipients.go`'s seams.

**Heal + Dispel (`healdispel`):**
- Non-SuperGM caster → no `ChangeHP`/`ChangeMP`/`CancelByTypes` emitted, warn logged (AC).
- Recipients = all map players incl. caster; a non-party player still receives HP/MP +
  dispel.
- HP/MP clamp: delta never exceeds `effMax - current`; flat-only, ratio-only, and mixed
  recovery shapes each produce the right delta.
- Dispel emits `CANCEL_BY_TYPES` with exactly the 11 disease strings.
- No `AwardExperience` call on any path (AC).
- One recipient's `ChangeHP` error does not abort the others (FR-10).
- Skill-use: foreign announce suppressed when caster hidden, sent when visible.

**Hide (`hide`):**
- Non-SuperGM caster → no buff apply/cancel, no despawn/spawn, warn logged (AC).
- Visible caster casts → `Apply(DARK_SIGHT, source=9101004, MaxInt32)` + despawn-to-others
  (AC).
- Hidden caster casts → `Cancel(source=9101004)` + spawn-to-others (AC).
- `isHidden` keys on `SourceId==9101004 && !Expired()`; a `DARK_SIGHT` buff from
  `RogueDarkSightId` does **not** read as hidden (collision guard).
- Foreign skill-use suppressed for a hidden caster.

**Spawn suppression:**
- `spawnCharacterForSession` for a character with a `SuperGmHide`-sourced buff returns
  without announcing (no spawn packet); a normal character and a Rogue-Dark-Sight
  character both still spawn.
- Regression: existing Cleric Heal, Mystic Door, mount, and normal spawn/despawn tests
  stay green.

**Byte-level (execute-time, per acceptance criteria):** any `CharacterSpawn`/
`CharacterDespawn`/buff give/cancel packet exercised on the hide/reveal path is
byte-verified against source; the `9101000` WZ recovery values are confirmed against live
WZ data (§3.3).

---

## 7. Service impact (corrected)

**`atlas-channel`** — all new code:
- `skill/handler/healdispel/` (+ registration) and `skill/handler/hide/` (+ registration);
  both blank-imported in `registrations/registrations.go`.
- `skill/handler/recipients.go`: new `SelectAllCharactersInMap`.
- `character/buff/`: new `CancelByTypes` producer + processor method; new
  `CommandTypeCancelByTypes` + `CancelByTypesCommandBody` in `kafka/message/buff/kafka.go`.
- `data/skill/effect/model.go`: new `MP()`, `HpR()`, `MpR()` accessors.
- `kafka/consumer/map/consumer.go`: suppression gate in `spawnCharacterForSession`; new
  exported `DespawnCharacterInMap` / `SpawnCharacterInMap` broadcast helpers.

**`atlas-data`** — **no change** (§1.1).
**`atlas-buffs`** — **no change** (already consumes `CANCEL_BY_TYPES`, emits `EXPIRED`).
**`libs/atlas-constants`** — **no change** (`SuperGmHealDispelId`, `SuperGmHideId`,
`SuperGmId`, `TemporaryStatTypeDarkSight` all exist).

Because only `atlas-channel`'s `go.mod` is touched, the mandatory
`docker buildx bake atlas-channel` (CLAUDE.md build step 4) covers the container check;
`go test -race`, `go vet`, `go build`, and `tools/redis-key-guard.sh` run across the
changed module.

---

## 8. Risks & mitigations

- **`9101000` recovery magnitude unverified** (WZ not in repo). Mitigation: flat+ratio
  formula tolerant to either shape; execute-time WZ verification is a hard gate (§3.3).
- **`DARK_SIGHT` local-render byte encoding** for the self buff-give must be confirmed
  against the v83 client (the stat must serialize such that `IsDarkSight` reads non-zero).
  Mitigation: byte-verify the self give packet at execute; amount set non-zero.
- **Stray async foreign buff give/cancel** on hide toggle. Mitigation: no-op foreign
  encode + explicit despawn/spawn + suppression gate make them inert (§3.4 ordering note).
- **Per-spawn buff fetch cost** in the suppression gate. No new cost: `spawnCharacterForSession`
  already fetches the buff list for every spawn; the gate reuses it.
- **Multi-tenancy**: all processors resolve tenant from context and commands carry tenant
  headers via the existing producer pattern; hide state (a per-character buff) and heal
  recipients (map-scoped) are inherently tenant-isolated.
