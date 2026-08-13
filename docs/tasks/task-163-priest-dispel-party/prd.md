# Priest Dispel — Party Debuff Cure — Product Requirements Document

Version: v2
Status: Draft
Created: 2026-07-10
Revised: 2026-08-07 — brought current with `main` @ `e0f5bd01d`; scope widened to
all supported client versions.
---

## 0. What Changed in v2

The v1 PRD was written on 2026-07-10 against a `main` that has since advanced by
254 commits. Four of its premises are no longer true, and the version scope was
implicit. Every change below is cited to current `main`.

| v1 premise | Current `main` | Impact |
|---|---|---|
| "The only missing pieces are a channel-side `CANCEL_BY_TYPES` producer and a Dispel handler" | The producer **already exists** — task-156 landed it: `CancelByTypesCommandProvider` (`character/buff/producer.go:89`), `Processor.CancelByTypes(f, characterId, types []string) error` (`character/buff/processor.go:25,82`), `CommandTypeCancelByTypes` + `CancelByTypesCommandBody` (`kafka/message/buff/kafka.go:17,64`), and a wire-contract test (`character/buff/producer_test.go:19`) | FR-1/FR-2 become **consume, do not build**. Signature differs from v1's design: uncurried, `[]string`, returns `error`. |
| "register via `channelhandler.Register(skill2.PriestDispelId, Apply)`" | The registry is keyed on **`skill2.Identity`**, not a wire id (`skill/handler/registry.go`, task-187). `UseSkill` resolves wire→identity via `constants.For(region,major,minor).Skill.Resolve` before `Lookup` (`skill/handler/common.go`) | FR-3 must register `skill2.PriestDispel` (the Identity), never `PriestDispelId`. This is what makes the handler version-independent. |
| "No mob→character disease infliction path exists yet… live-play Dispel will typically find nothing to cancel" | **It exists.** `atlas-monsters` emits `APPLY` disease commands: `applyDiseaseCommandProvider` (`monster/disease.go`), driven by `SkillTypeToDiseaseName` (`libs/atlas-constants/monster/skill.go:112`) at `monster/processor.go:1239-1252` | Dispel is now meaningful in live play. Acceptance uses a real mob debuff, not only a seeded `APPLY`. |
| "task-156 (in design) plans the same producer; whichever lands second rebases" | task-156 **landed**: `skill/handler/healdispel/healdispel.go` purges 11 disease types via the shared `CancelByTypes` | task-163 is the second lander. It consumes; there is nothing left to coordinate. |

Newly in scope: **all 11 supported client versions** (§3), and the per-version
correctness of the serverbound decode that feeds this handler its party bitmap
(§4.4) — the one place where "works on v83" does not imply "works everywhere."

## 1. Overview

Priest Dispel (identity `skill2.PriestDispel`, wire `2311001` on every supported
version — see §3) is a dual-effect skill: it "cancels out all spell effects of
the enemies within a certain area, along with curing everyone in the party of
all sicknesses" (in-game description).

Atlas implements only the mob half. `applyToMobs`
(`services/atlas-channel/atlas.com/channel/skill/handler/common.go`) classifies
Dispel as a magic-class cancel (`isCrashOrDispel` / `dispelSkillClass`, both
Identity-keyed since task-187) and cancels mob buffs with magic-reflect
awareness. The party half is missing entirely: casting Dispel does nothing for
debuffed party members.

Everything downstream already exists:

- The client's affected-party-member bitmap is decoded and exposed as
  `info.AffectedPartyMemberBitmap()` (`libs/atlas-packet/model/skill_usage_info.go`),
  consumed today by the generic party-buff apply in `UseSkill`.
- Map-wide party member selection exists as `SelectPartyMembersInMap`
  (`skill/handler/recipients.go:188`), already filtering offline / other-map /
  no-session / dead members and decoding the MSB-first bitmap.
- The channel-side `CANCEL_BY_TYPES` producer exists (`character/buff`), built
  by task-156.
- `atlas-buffs` consumes `CANCEL_BY_TYPES` (`kafka/consumer/character/consumer.go:93`
  → `CancelByStatTypes`), cancels matching buffs, and emits the `EXPIRED` status
  events `atlas-channel` already turns into client buff-cancel packets.

The single missing piece is a per-skill Dispel handler that selects recipients,
rolls the skill's prop per recipient, and calls the existing producer.

## 2. Goals

Primary goals:

- Casting Dispel cures the six dispellable debuff stat types on the caster and
  on party members in the same map selected by the client's affected-member
  bitmap.
- The skill's WZ success chance (`prop`) is honored per recipient, matching
  Cosmic's per-`applyTo` `makeChanceResult()` roll.
- **The behavior is correct on every supported version**, and where it is not
  reachable on a version, that is stated with evidence rather than assumed
  (§3, §4.4).

Non-goals:

- SuperGM Heal+Dispel (`9101000` / `skill2.SuperGmHealDispel`) — landed as
  task-156 (`skill/handler/healdispel`). Its 11-type purge set is deliberately
  wider than Dispel's six.
- Any change to the existing mob-side Dispel path (`applyToMobs`, reflect
  handling, rect verification) — already implemented and out of scope.
- Curing STUN / SEDUCE / CONFUSE / UNDEAD / STOP_PORTION / STOP_MOTION / FEAR.
  `atlas-monsters` can inflict all of these
  (`libs/atlas-constants/monster/skill.go:112-142`), but Dispel cures only the
  six below; the rest are cure-all (`purgeDebuffs`) semantics owned by
  Heal+Dispel and cure items.
- Building any `CANCEL_BY_TYPES` producer, message type, or processor method —
  all present on `main`.
- `atlas-buffs` changes.
- Making Priest Dispel reachable on `gms_12_1` — that template routes no
  `CharacterUseSkillHandle` at all (§3), so it is a whole-skill-system gap, not
  a Dispel gap.
- Rewriting `SkillUsageInfo.Decode`'s membership lists into version-aware
  tables. §4.4 requires *verifying* the current version-blind lists against each
  client and *recording* the result; a divergence found becomes a scoped fix for
  Dispel's own read path, not a general refactor.

## 3. Supported Versions

Two different "supported version" lists exist in this repo and they do not
match. The authoritative one for this task is the tenant version set.

- **Tenant version set — 11 versions.** `deploy/k8s/base/versions.json`,
  mirrored by `libs/atlas-constants/constants/registry_gen.go` and by 11 seed
  templates under `services/atlas-configurations/seed-data/templates/`:
  `gms_12_1, gms_48_1, gms_61_1, gms_72_1, gms_79_1, gms_83_1, gms_84_1,
  gms_87_1, gms_92_1, gms_95_1, jms_185_1`.
- **Packet coverage matrix — 9 columns.** `docs/packets/audits/STATUS.md` omits
  `gms_12` and `gms_92` (no audit dir, no export). Those two versions can still
  be reasoned about from a named IDB, but they have no evidence record to pin.

### 3.1 Per-version status

| Version | `PriestDispel` wire id | `CharacterUseSkillHandle` routed | Dispel reachable | Notes |
|---|---|---|---|---|
| gms_12_1 | 2311001 | **No** | **No** | Login-only minimal template (24 handlers vs v83's 134). No skill-use packet is routed, so no skill handler of any kind fires. Out of scope (§2). |
| gms_48_1 | 2311001 | Yes | Yes | Matrix column; audit export present. |
| gms_61_1 | 2311001 | Yes | Yes | Matrix column; audit export present. |
| gms_72_1 | 2311001 | Yes | Yes | Matrix column; audit export present. |
| gms_79_1 | 2311001 | Yes | Yes | Matrix column; audit export present. |
| gms_83_1 | 2311001 | Yes | Yes | Baseline; the only version any current decode assumption was derived from. |
| gms_84_1 | 2311001 | Yes | Yes | Matrix column; audit export present. |
| gms_87_1 | 2311001 | Yes | Yes | Matrix column; audit export present. |
| gms_92_1 | 2311001 | Yes | Yes | **No matrix column / no audit export.** Named v92 IDB exists; derive from IDA and record in the task folder. |
| gms_95_1 | 2311001 | Yes | Yes | Matrix column; audit export present. |
| jms_185_1 | 2311001 | Yes | Yes | Matrix column; audit export present. |

Sources: wire ids from the generated identity↔wire maps
(`libs/atlas-constants/skill/version_gms_*_gen.go`, `version_jms_185_1_gen.go`
— `PriestDispel: 2311001` in all 11); handler routing from
`grep CharacterUseSkillHandle services/atlas-configurations/seed-data/templates/`
(10 of 11 hits; `template_gms_12_1.json` absent).

### 3.2 What is and is not version-divergent here

- **The wire id is NOT divergent.** `PriestDispel` binds to `2311001` on all 11
  version sets, and `PriestDispelId` does not appear in task-187's
  `divergences.csv`, so `tools/skill-job-id-guard.sh` does not fire on it.
- **Registration must still be Identity-keyed** — not because 2311001 moves,
  but because `skill/handler/registry.go` accepts only `skill2.Identity` and
  `UseSkill` looks up the *resolved* identity. A wire-keyed registration will
  not compile, and a wire-keyed comparison anywhere in the handler would be a
  latent version bug even where the id happens to agree today.
- **WZ effect data is per-version by construction.** `e.Prop()`,
  `e.LT()/RB()`, and `e.MobCount()` come from `atlas-data` for the requesting
  tenant's version. Nothing about the prop is hard-coded; the v1 PRD's "prop 34
  at level 1 rising to 100 at level 20" is a **v83 reference value only** and
  must not be asserted for other versions without a per-version read (§4.5).
- **The serverbound decode IS the version risk.** See §4.4.

## 4. Functional Requirements

### 4.1 Reuse of the existing `CANCEL_BY_TYPES` producer

- **FR-1.** The task MUST NOT add a `CANCEL_BY_TYPES` command constant, body
  type, producer provider, or processor method. All four exist on `main`:
  - `buff.CommandTypeCancelByTypes` / `buff.CancelByTypesCommandBody{Types []string}`
    (`atlas-channel/kafka/message/buff/kafka.go:17,64`)
  - `CancelByTypesCommandProvider(f field.Model, characterId uint32, types []string)`
    (`atlas-channel/character/buff/producer.go:89`)
  - `Processor.CancelByTypes(f field.Model, characterId uint32, types []string) error`
    (`atlas-channel/character/buff/processor.go:25,82`)
- **FR-2.** The Dispel handler MUST call `CancelByTypes` with its **own** six-type
  set. The existing signature is uncurried and takes `[]string`; the handler
  holds its set as a package-level `[]string` built from
  `charconst.TemporaryStatType` constants (the `healdispel.diseaseTypes`
  precedent). No signature change to the shared processor.

### 4.2 Dispel skill handler

- **FR-3.** A new `skill/handler/dispel` subpackage MUST register via
  `channelhandler.Register(skill2.PriestDispel, Apply)` in `init()` — the
  **Identity**, not `PriestDispelId` — and be blank-imported from
  `skill/handler/registrations/registrations.go`. The blank-import edit MUST be
  **additive**: `registrations.go` currently lists seven handlers (heal,
  healdispel, hide, mprecovery, mysticdoor, resurrection, timeleap) and none may
  be dropped. The handler runs from the per-skill dispatcher in `UseSkill`,
  after the mob-side `applyToMobs` call — the two halves stay independent.
- **FR-4.** Recipient selection MUST be the caster plus
  `SelectPartyMembersInMap(l, ctx, f, casterId, info.AffectedPartyMemberBitmap())`
  — map-wide, no rectangle limit. (The WZ lt/rb rect governs the *mob* half,
  which `applyToMobs` already enforces; the party cure is "everyone in the
  party", per the skill description and owner decision.)
- **FR-5.** The cure set MUST be exactly:
  `CURSE, DARKNESS, POISON, SEAL, WEAKEN, SLOW`
  (`charconst.TemporaryStatTypeCurse/Darkness/Poison/Seal/Weaken/Slow`,
  `libs/atlas-constants/character/temporary_stat.go`). Cosmic parity:
  `Character.dispelDebuffs()`. The set is a package-level constant slice, not
  built inline per cast. These six names match exactly six of the disease names
  `atlas-monsters` inflicts (`SkillTypeToDiseaseName`: SEAL, DARKNESS, WEAKEN,
  CURSE, POISON, SLOW), so every one of them is curable in live play.
- **FR-6.** The skill's success chance MUST be rolled once per recipient
  (caster included) using the effect's `prop`, mirroring the `propRollFunc`
  semantics in `common.go` (`prop <= 0` → false, `prop >= 1` → true, else
  `rand.Float64() <= prop`) behind a package-local seam so tests can inject
  deterministic rolls. `effect.Model.Prop()` is already normalized to 0.0–1.0 —
  no `/100` scaling. A failed roll skips that recipient's cure; it never fails
  the cast.
- **FR-7.** One `CANCEL_BY_TYPES` command MUST be emitted per recipient that
  passes the prop roll, carrying the six stat types. Per-recipient emit failures
  are logged and do not abort remaining recipients (pattern: `healdispel`'s
  per-recipient error handling).
- **FR-8.** The handler MUST emit a per-cast structured debug summary (caster,
  skill id, skill level, bitmap, recipients selected, cures emitted,
  prop-skipped count) following the `buildSummaryFields` precedent in
  `common.go`.

### 4.3 Version-independence of the handler

- **FR-9.** The handler MUST contain no raw wire-id comparison. Registration is
  by Identity (FR-3); any further skill test inside the handler uses
  `skill2.IsIdentity` against a resolved identity, never `skill2.Is` on
  `info.SkillId()`. Rationale: `tools/skill-job-id-guard.sh` and the task-187
  contract — a wire compare that is correct today silently breaks the first
  time a version rebinds the id.
- **FR-10.** The handler MUST NOT read tenant version, region, or major/minor
  directly. Everything version-sensitive reaches it already resolved: the
  identity via the registry, the effect via `atlas-data`, the bitmap via the
  decoder. If a per-version behavior difference is discovered under §4.4, it is
  fixed at the decode layer, not branched on inside the handler.

### 4.4 Per-version verification of the party-bitmap decode

This is the one requirement that "all supported versions" actually buys, and it
is a real risk, not a formality.

`libs/atlas-packet/model/skill_usage_info.go` `Decode` reads the serverbound
skill-use packet with **three hard-coded, version-blind raw-wire-id membership
lists**, each carrying a `// TODO this is not all inclusive` comment:
`isAntiRepeatBuffSkill` (reads `castX`/`castY`), `isPartyBuff` (reads the
affected-member bitmap byte), `isMobAffectingBuff` (reads a mob count + ids +
delay). Wire id `2311001` is in **all three**, so the decoder's assumed layout
for a Dispel cast is:

```
updateTime(4) skillId(4) slv(1) castX(2) castY(2) bitmap(1) delay(2) mobCount(1) mobIds(4×N) delay(2)
```

— note the delay field is read twice, once under the `isPartyBuff` arm's
`PriestDispelId` special case and once under the `isMobAffectingBuff` arm.

Why this matters more than it looks:

- The client writes each of these fields **conditionally**, by the skill's
  client-side category. If the server list membership disagrees with the client
  for a given version, every field after the mismatch is read at the wrong
  offset. The bitmap then decodes to garbage or 0, `SelectPartyMembersInMap`'s
  `memberBitmap == 0 || memberBitmap >= 128` gate (`recipients.go:236`) returns
  nil, and Dispel silently degrades to caster-only — **with no error and no log
  line**. This exact failure has already happened twice: Bishop Resurrection
  2321006 (task-111) and Buccaneer Time Leap 5121010 (task-155, PR#1136).
- The lists and the `recipients.go:236` slot-order comment are both derived from
  **v83 only** (`CUserLocal::SendSkillUseRequest` @0x96d399,
  `is_antirepeat_buff_skill` @0x96d6ca, `CUserLocal::FindParty` @0x96db3f), and
  applied version-blind to all 10 reachable versions.
- The serverbound `SPECIAL_MOVE` row (`CUserLocal::SendSkillUseRequest`) is
  **❌ on all nine matrix columns** with no codec listed — there is no pinned
  evidence for this packet's layout on **any** version, including v83.

Therefore:

- **FR-11.** For each of the 10 versions where Dispel is reachable (§3.1), the
  task MUST determine, from that version's client, whether a `2311001` cast
  writes `castX/castY`, whether it writes the affected-member bitmap, and
  whether it writes the mob list — i.e. the true field order — and record the
  finding per version in the task folder with its evidence (IDB, function
  address or export hash).
- **FR-12.** Where a version's true layout matches the current decoder, the
  finding is recorded as confirmed and nothing changes. Where it diverges, the
  decoder MUST be corrected such that the Dispel party bitmap decodes correctly
  on **every** reachable version, and the correction MUST be covered by a
  byte-layout regression test per affected version (the task-111 precedent:
  wire-layout test alongside the list fix).
- **FR-13.** Where a version's client cannot be read (no IDB, no export), the
  task MUST say so explicitly per version and MUST NOT record it as verified.
  An unread version is reported as unverified with the reason — it is never
  silently folded into the v83 assumption.
- **FR-14.** The handler MUST log the decoded bitmap and the resulting recipient
  count on every cast (FR-8 covers this). This is the field-diagnosable
  signature for the failure mode above: `bitmap=0, recipients_selected=1` on a
  version where the caster demonstrably had a party.

### 4.5 Per-version effect data

- **FR-15.** The task MUST confirm that `atlas-data` resolves a skill effect for
  `2311001` on each reachable version, and record each version's `prop` curve.
  A version whose `GetEffect` lookup fails aborts the entire cast in
  `CharacterUseSkillHandleFunc` before `UseSkill` runs, so the handler would
  never fire — a silent, version-specific no-op.
- **FR-16.** No prop value may be hard-coded anywhere in the implementation. The
  v83 reference figures (prop 34 at level 1 → 100 at level 20) are documentation
  only and MUST be re-read per version rather than assumed.

### 4.6 Downstream behavior (verification only — no changes)

- **FR-17.** No `atlas-buffs` change: `CancelByStatTypes` already cancels
  matching buffs and emits `EXPIRED` status events per cancelled buff, which
  `atlas-channel`'s buff status consumer already converts to client buff-cancel
  packets. The task MUST verify this end-to-end path in acceptance, not
  reimplement it.

## 5. API Surface

No REST changes. No **new** Kafka surface — the task is the first non-GM
consumer of an existing command:

- **Producer (existing, atlas-channel):** `COMMAND_TOPIC_CHARACTER_BUFF`, type
  `CANCEL_BY_TYPES`, body
  `{ "types": ["CURSE","DARKNESS","POISON","SEAL","WEAKEN","SLOW"] }`, envelope
  fields (worldId/channelId/mapId/instance/characterId) per the existing
  `Command[E]` in `kafka/message/buff/kafka.go`. Wire shape already pinned by
  `character/buff/producer_test.go`.
- **Consumer (existing, atlas-buffs):** already handles this type; no change.

## 6. Data Model

None. No new entities, no persistence. Buff state lives in atlas-buffs'
in-memory registry as today.

## 7. Service Impact

- **`atlas-channel`** (primary changed service):
  - `skill/handler/dispel/` — new handler subpackage + tests.
  - `skill/handler/registrations/registrations.go` — one added blank import
    (additive; the seven existing imports stay).
  - `docs/domain.md` — document the new handler alongside the existing
    `skill/handler/healdispel` paragraph.
- **`libs/atlas-packet`** (conditional, FR-12): `model/skill_usage_info.go` and
  its tests, **only if** §4.4 finds a version whose true layout diverges from the
  current lists. If every reachable version confirms, this module is untouched.
- **`atlas-buffs`** — no change (verified: command, processor, event emission
  all present).
- **`atlas-monsters`** — no change (it is the debuff *source* under test, not a
  target of change).
- **Seed templates** — no change. `CharacterUseSkillHandle` is already routed on
  all 10 reachable versions; the task adds no new opcode, handler, or writer.

## 8. Non-Functional Requirements

- **Multi-tenancy:** all emission via the existing tenant-aware producer chain
  (`tenant.MustFromContext(ctx)` implicit in `producer.ProviderImpl`); no
  tenant-specific behavior — stat-type strings are semantic keys, not client
  wire values (DOM-25 not implicated).
- **Version-awareness:** per FR-9/FR-10, all version sensitivity lives in the
  registry resolution and the decoder, never in the handler.
- **Performance:** one Kafka message per cured recipient per cast (≤6 party
  members); negligible.
- **Observability:** per-cast debug summary (FR-8, FR-14); per-recipient emit
  errors at error level.
- **Testing:** Builder-pattern test setup (project rule — no
  `*_testhelpers.go`); deterministic prop-roll injection via the seam;
  registry/registration test mirroring the `registry_test.go` precedent;
  per-version byte-layout regression tests for any FR-12 decoder fix.

## 9. Open Questions

1. **(FR-11, blocking the version claim — not the implementation.)** The true
   serverbound field order for a `2311001` cast is unverified on every version,
   including v83. The implementation can proceed against the current decoder;
   the task cannot claim "works on all supported versions" until §4.4 is
   discharged. If a divergence is found, the fix is scoped to Dispel's read path
   (FR-12), not a general rewrite of the three membership lists.
2. **(gms_92 evidence.)** gms_92 has no matrix column and no checked-in export,
   so an FR-11 finding for it can be recorded only in the task folder (a named
   v92 IDB exists). This is a documentation-location constraint, not a gap in
   the finding itself.
3. **(gms_12.)** Declared out of scope in §2 with evidence (no
   `CharacterUseSkillHandle` in `template_gms_12_1.json`). Wiring the whole
   skill-use path for a login-only template is a separate task.

## 10. Acceptance Criteria

Implementation:

- [ ] `skill/handler/dispel` registered for `skill2.PriestDispel` (the
      **Identity**); blank import added to `registrations/registrations.go`
      **without removing** any of the seven existing imports.
- [ ] Zero new `CANCEL_BY_TYPES` producer/message/processor code — the diff
      touches none of `kafka/message/buff/kafka.go`,
      `character/buff/producer.go`, `character/buff/processor.go`.
- [ ] Casting Dispel emits one `CANCEL_BY_TYPES` command per prop-passing
      recipient (caster + bitmap-selected in-map members) with exactly the six
      stat types `CURSE, DARKNESS, POISON, SEAL, WEAKEN, SLOW`.
- [ ] Prop roll is per-recipient and test-injectable; a failed roll skips only
      that recipient.
- [ ] No raw wire-id comparison anywhere in the new package (FR-9).
- [ ] Mob-side Dispel behavior is byte-for-byte unchanged (existing
      `common_apply_to_mobs_test.go` still passes, no diff in `common.go`).
- [ ] No changes under `services/atlas-buffs/` or `services/atlas-monsters/`.
- [ ] Per-cast summary log line present with bitmap, recipient, cure, and
      prop-skip counts.
- [ ] `atlas-channel/docs/domain.md` documents the handler.

Per-version (§3, §4.4, §4.5):

- [ ] A per-version findings table exists in the task folder covering all 11
      versions: for each, the `2311001` serverbound field order with its evidence
      (IDB + function address, or export hash), or an explicit "unverified —
      reason" (FR-13). gms_12 is recorded as unreachable with its evidence.
- [ ] Every reachable version is either confirmed to match the current decoder,
      or corrected under FR-12 with a byte-layout regression test.
- [ ] `atlas-data` effect resolution for `2311001` confirmed per reachable
      version, with each version's prop curve recorded (FR-15). No prop value
      hard-coded in code (FR-16).

End-to-end:

- [ ] A **mob-inflicted** debuff (e.g. a `SkillTypeSeal` mob skill →
      `APPLY {type: "SEAL"}` from `atlas-monsters`) on a party member is
      cancelled by a Dispel cast, and the member's client receives the
      buff-cancel packet via the existing `EXPIRED` event path.
- [ ] A directly seeded debuff (`APPLY` with a debuff stat type) is likewise
      cancelled — the deterministic path for CI/manual verification.
- [ ] A debuff **outside** the six (e.g. `STUN` or `SEDUCE`) survives a Dispel
      cast, confirming the set is not a cure-all.

Gates:

- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
      `atlas-channel` — and in `libs/atlas-packet` if FR-12 touched it.
- [ ] `docker buildx bake atlas-channel` succeeds from the worktree root.
- [ ] `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
      `tools/skill-job-id-guard.sh`, and `tools/lint.sh --check` clean from the
      repo root.
- [ ] Code review (`superpowers:requesting-code-review`) run before opening the
      PR.
