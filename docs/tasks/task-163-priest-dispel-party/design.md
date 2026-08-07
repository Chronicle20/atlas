# Priest Dispel — Party Debuff Cure — Design

Task: task-163-priest-dispel-party
PRD: docs/tasks/task-163-priest-dispel-party/prd.md (v2, approved)
Status: Proposed
Created: 2026-07-10
Revised: 2026-08-07 — rebased onto `main` @ `e0f5bd01d`; §3.1–3.3 deleted
(already built), §3.4 re-keyed to Identity, new §4 (version matrix) added.

---

## 1. Problem Recap

Priest Dispel (`skill2.PriestDispel`, wire `2311001` on all 11 supported
versions) is dual-effect. The mob half (cancel mob buffs, magic-reflect-aware)
is already implemented in `applyToMobs`
(`services/atlas-channel/atlas.com/channel/skill/handler/common.go`,
Identity-keyed via `isCrashOrDispel` / `dispelSkillClass`). The party half —
cure CURSE, DARKNESS, POISON, SEAL, WEAKEN, SLOW on the caster and
bitmap-selected in-map party members — is missing.

Since the v1 design, `main` has closed three of the four gaps it named:

- The channel-side `CANCEL_BY_TYPES` producer, message type, and processor
  method all exist (built by task-156, `character/buff` + `kafka/message/buff`).
- `atlas-buffs` consumes `CANCEL_BY_TYPES` and emits `EXPIRED` status events
  that `atlas-channel` converts to client buff-cancel packets — unchanged.
- `atlas-monsters` now inflicts the diseases (`applyDiseaseCommandProvider`,
  `monster/disease.go`), so the cure has real work to do in live play.

What remains is **one new package** — the per-skill Dispel handler — plus a
per-version verification pass on the serverbound decode that feeds it the party
bitmap (§4).

## 2. Approaches Considered

### Approach A — per-skill handler subpackage over the existing shared producer (RECOMMENDED)

New `skill/handler/dispel` subpackage registering via the per-skill registry
(`channelhandler.Register(skill2.PriestDispel, Apply)` in `init()`,
blank-imported from `registrations/registrations.go`), calling the **existing**
`buff.Processor.CancelByTypes`.

- Matches the established precedent exactly (`heal`, `healdispel`, `hide`,
  `mprecovery`, `mysticdoor`, `resurrection`, `timeleap`): the per-skill
  dispatcher in `UseSkill` already runs after `applyToMobs`, so the two Dispel
  halves stay independent with zero orchestrator changes.
- Reuses task-156's producer verbatim — no shared-surface churn, no risk to the
  landed SuperGM path.
- Registration is on the version-blind `skill2.Identity`, so the handler is
  correct on every version without a version branch (PRD FR-9/FR-10).
- Testable with the local-seam idiom the sibling handlers use (package-level
  func vars, overridden and `t.Cleanup`-restored in tests).

### Approach B — inline the party cure in `common.go` next to `applyToMobs`

Add an `if skill2.IsIdentity(castId, skill2.PriestDispel) { curePartyDebuffs(...) }`
branch in `UseSkill`.

- Rejected: grows the orchestrator that the per-skill registry exists to shrink;
  every future cure-type skill (Hero's Will, cure items) would pile more special
  cases into `UseSkill`; testing requires driving the whole `UseSkill` pipeline
  instead of one handler.

### Approach C — push skill semantics into atlas-buffs (a `DISPEL` command)

Emit a skill-level command (`DISPEL`, carrying skillId/level) and let atlas-buffs
decide the stat set and prop roll.

- Rejected: atlas-buffs is deliberately a dumb buff registry with a generic
  `CANCEL_BY_TYPES` already in place and already consumed by `healdispel`;
  skill semantics (which stats, per-recipient prop) are channel-side knowledge
  (WZ effect data lives there). Would also duplicate the prop-roll seam
  server-side without access to `effect.Model`, and would make atlas-buffs
  version-aware for no reason.

### Approach D — reshape `Processor.CancelByTypes` to the v1 curried signature

Change `CancelByTypes(f, characterId, types []string) error` into
`CancelByTypes(f, types) model.Operator[uint32]`.

- Rejected: the current signature is on `main`, exercised by `healdispel` and by
  the Homing Beacon path in `socket/handler/character_attack_common.go`, and
  pinned by `character/buff/producer_test.go`. Reshaping it for one new caller
  buys nothing (the handler's loop is three lines either way) and puts two
  landed features at risk. The handler adapts to the shared surface, not the
  reverse.

**Decision: Approach A**, consuming the existing uncurried `[]string` signature.

## 3. Component Design

All production changes in
`services/atlas-channel/atlas.com/channel/skill/handler/dispel/` plus one blank
import. No `atlas-buffs`, no `atlas-monsters`, no `common.go`, no producer
changes.

### 3.1 What is NOT built (deleted from the v1 design)

The v1 design's §3.1 (message constant + body type), §3.2 (producer provider)
and §3.3 (processor method) are **already on `main`** and must not be
reimplemented:

| v1 design section | Current `main` |
|---|---|
| §3.1 `CommandTypeCancelByTypes` + `CancelByTypesCommandBody` | `atlas-channel/kafka/message/buff/kafka.go:17,64` |
| §3.2 `CancelByTypesCommandProvider(f, characterId, types []string)` | `atlas-channel/character/buff/producer.go:89` |
| §3.3 `Processor.CancelByTypes(...)` | `atlas-channel/character/buff/processor.go:25,82` — signature `(f field.Model, characterId uint32, types []string) error`, **not** the curried `model.Operator[uint32]` v1 proposed |
| §3.3's "producer test" | `atlas-channel/character/buff/producer_test.go:19` `TestCancelByTypesCommandProvider` |

Consequence for the handler: it holds its cure set as a package-level
`[]string` and calls the emitter once per recipient, rather than binding an
operator once per cast. Functionally identical; one fewer indirection.

### 3.2 Dispel handler — `skill/handler/dispel/` (the only new package)

`dispel.go`:

```go
func init() {
    // Identity, not wire id: skill/handler/registry.go is keyed on
    // skill2.Identity and UseSkill resolves the incoming wire id through
    // constants.For(...).Skill.Resolve before Lookup (task-187). This single
    // registration covers all 11 provisioned versions.
    channelhandler.Register(skill2.PriestDispel, Apply)
}

// dispellableStatTypes is the exact Dispel cure set (PRD FR-5, Cosmic
// Character.dispelDebuffs parity). Held as []string to match the shared
// buff.Processor.CancelByTypes signature — the healdispel.diseaseTypes
// precedent. STUN / SEDUCE / CONFUSE / UNDEAD / STOP_PORTION / STOP_MOTION /
// FEAR are intentionally excluded: atlas-monsters can inflict them, but they
// are purgeDebuffs (cure-all) semantics owned by SuperGM Heal+Dispel.
var dispellableStatTypes = []string{
    string(charconst.TemporaryStatTypeCurse),
    string(charconst.TemporaryStatTypeDarkness),
    string(charconst.TemporaryStatTypePoison),
    string(charconst.TemporaryStatTypeSeal),
    string(charconst.TemporaryStatTypeWeaken),
    string(charconst.TemporaryStatTypeSlow),
}
```

`Apply` implements `channelhandler.Handler`. Flow:

1. **Recipients** (FR-4): the caster id, plus
   `SelectPartyMembersInMap(l, ctx, f, characterId, info.AffectedPartyMemberBitmap())`
   (`recipients.go:188`) — map-wide, no rectangle. Only ids are needed; unlike
   `heal`, no caster load, no position, no effective-stats fetch. The selector
   already filters offline / other-map / no-session / dead members and applies
   the MSB-first bitmap decode, and returns nil for `bitmap == 0 || bitmap >= 128`.
2. **Per recipient** (FR-6, FR-7): roll `propRollFunc(e.Prop())`; on fail,
   increment `propSkipped` and continue (never fails the cast). On pass, call
   `cancelByTypesFunc(l, ctx, f, recipientId, dispellableStatTypes)`; on emit
   error, log at error level with the recipient id and continue (the
   `healdispel` per-recipient error pattern). Count `curesEmitted`.
3. **Summary** (FR-8, FR-14): one `Debug` line with structured fields
   `caster, skill_id, skill_level, bitmap, recipients_selected, cures_emitted,
   prop_skipped` — the `buildSummaryFields` precedent, local to the package.
   `bitmap` and `recipients_selected` together are the field signature for the
   §4 decode failure mode.
4. Return `nil` always — all failures are logged, none abort.

**Seams** (sibling-handler style package-level func vars, test-overridden with
`t.Cleanup` restore):

```go
var selectPartyMembersFunc = channelhandler.SelectPartyMembersInMap

// Mirrors skill/handler/common.go's unexported propRollFunc exactly: the
// parent seam is unexported and cannot be shared across the package boundary,
// and exporting a mutable seam from `handler` just for this would leak test
// plumbing. prop <= 0 -> false; prop >= 1 -> true; else rand.Float64() <= prop.
var propRollFunc = func(prop float64) bool { ... }

var cancelByTypesFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, types []string) error {
    return buff.NewProcessor(l, ctx).CancelByTypes(f, characterId, types)
}
```

**Prop semantics note**: `effect.Model.Prop()` is already normalized to 0.0–1.0
(`data/skill/effect/model.go`), so no `/100` scaling in the handler. The
`prop <= 0 → false` defensive arm is kept identical to the mob path (a
zero-prop effect cures nobody, consistent with `applyToMobs`) — and it is *not*
unreachable across versions, because per-version WZ data is not assumed (PRD
FR-15/FR-16).

**Version-independence** (PRD FR-9/FR-10): the package imports no `constants`,
no `tenant`, and performs no `skill2.Is` on `info.SkillId()`. The only skill
reference is the `skill2.PriestDispel` Identity in `init()` and
`skill2.Id(info.SkillId())` as an opaque value in the summary log field.

### 3.3 Registration — `skill/handler/registrations/registrations.go`

**Additive** edit only. The file currently holds seven blank imports (heal,
healdispel, hide, mprecovery, mysticdoor, resurrection, timeleap). Insert one
in alphabetical position:

```go
_ "atlas-channel/skill/handler/dispel"       // Priest Dispel party cure — task-163
```

The v1 design's replacement snippet listed only three imports and would delete
four landed handlers — do not paste it.

### 3.4 Documentation — `services/atlas-channel/docs/domain.md`

Add a paragraph for the new handler next to the existing
`skill/handler/healdispel` entry, following its shape: registered identity,
recipient selection, cure set, prop behavior, failure policy.

### 3.5 Explicit non-changes

- `common.go` untouched: `applyToMobs` already handles the mob half (including
  the magic-class reflect skip); the per-skill dispatcher already runs after it.
  Dispel's WZ effect has no statups/duration, so the generic party-buff apply in
  `UseSkill` never fires for it.
- `character/buff/*` and `kafka/message/buff/*` untouched (§3.1).
- No skill-use announce packets added: `CharacterUseSkillHandleFunc` already
  emits the self and foreign `AnnounceSkillUse` broadcasts for every cast,
  including Dispel. Adding more would double-announce.
- `services/atlas-buffs/` and `services/atlas-monsters/` — zero diffs
  (acceptance-gated).

## 4. Version Design

The handler itself is version-blind by construction (§3.2). The version work is
entirely about proving the *inputs* it receives are correct on each version.

### 4.1 Supported version set

11 tenant versions (`deploy/k8s/base/versions.json`,
`libs/atlas-constants/constants/registry_gen.go`, 11 seed templates):
gms_12_1, 48_1, 61_1, 72_1, 79_1, 83_1, 84_1, 87_1, 92_1, 95_1, jms_185_1.
The packet coverage matrix has only 9 columns — gms_12 and gms_92 have no audit
dir and no export, so findings for those two live in the task folder only.

### 4.2 Three version-sensitive inputs, and where each is resolved

| Input | Resolution point | Version risk |
|---|---|---|
| Skill identity | `constants.For(region,major,minor).Skill.Resolve` in `UseSkill`, then `Lookup(identity)` | **None.** `PriestDispel → 2311001` on all 11 generated version sets; not in task-187's `divergences.csv`. Identity keying makes it robust anyway. |
| Effect (`prop`) | `atlas-data` `GetEffect(skillId, level)` per tenant version, in `CharacterUseSkillHandleFunc` before `UseSkill` | **Low but real.** A version whose data lacks 2311001 aborts the cast upstream of the handler — a silent per-version no-op. Verify presence + record the prop curve per version (PRD FR-15). |
| Party bitmap | `SkillUsageInfo.Decode` (`libs/atlas-packet/model/skill_usage_info.go`) | **High.** §4.3. |

### 4.3 The decode risk, and the design response

`Decode` gates three optional field groups on **hard-coded, version-blind raw
wire-id lists** (`isAntiRepeatBuffSkill`, `isPartyBuff`, `isMobAffectingBuff`),
each carrying a `// TODO this is not all inclusive` comment. `2311001` is in all
three, giving an assumed layout of:

```
updateTime(4) skillId(4) slv(1) castX(2) castY(2) bitmap(1) delay(2) mobCount(1) mobIds(4×N) delay(2)
```

with `delay` read twice. Those lists — and the MSB-first slot decode at
`recipients.go:257-259` — were derived from **v83 only**
(`CUserLocal::SendSkillUseRequest` @0x96d399, `is_antirepeat_buff_skill`
@0x96d6ca, `CUserLocal::FindParty` @0x96db3f). The serverbound `SPECIAL_MOVE`
row is ❌ on all nine matrix columns, so no version has pinned evidence.

If the membership disagrees with a given client, every field after the mismatch
is read at the wrong offset; the bitmap decodes to 0; the selector's
`bitmap == 0` gate returns nil; and Dispel silently becomes caster-only with no
error. Precedent: task-111 (Bishop Resurrection 2321006) and task-155/PR#1136
(Buccaneer Time Leap 5121010) were both exactly this bug.

**Design response — three parts, in this order:**

1. **Instrument first.** The FR-8 summary line carries `bitmap` and
   `recipients_selected`. This is deliberately shipped *before* the per-version
   audit, so the failure mode is diagnosable from a log line on any version
   rather than requiring a repro.
2. **Audit per version (read-only).** For each of the 10 reachable versions,
   read that version's client and record whether a `2311001` cast writes
   castX/castY, the bitmap, and the mob list. Evidence per version: IDB name +
   function address, or the checked-in export hash from
   `docs/packets/audits/<version>/`. Versions that cannot be read are recorded
   as unverified with the reason (PRD FR-13) — never assumed to match v83.
3. **Fix narrowly if a divergence is found.** The correction lives in
   `skill_usage_info.go`, gated per version, with a byte-layout regression test
   per affected version. It does **not** become a general rewrite of the three
   lists for all ~50 skills — that is a separate task with its own evidence
   budget. If every reachable version confirms, `libs/atlas-packet` is untouched
   and the audit output is documentation only.

### 4.4 gms_12: unreachable, by evidence

`template_gms_12_1.json` routes 24 handlers and `CharacterUseSkillHandle` is not
among them (v83 routes 134, including it). No skill-use packet is decoded on
gms_12, so no per-skill handler of any kind fires there. Dispel is recorded as
unreachable on gms_12 with that evidence, and no code or template change is made
for it — wiring the whole skill-use path into a login-only template is a
different task.

## 5. Data Flow

```
client USE_SKILL packet (2311001)
  └─ CharacterUseSkillHandleFunc
       ├─ SkillUsageInfo.Decode  ← the per-version risk (§4.3)
       ├─ skill-book / HP / cooldown gates
       ├─ atlas-data GetEffect(2311001, lvl)  ← per-version prop (§4.2)
       └─ UseSkill (common.go)
            ├─ castId = constants.For(t).Skill.Resolve(2311001) → PriestDispel
            ├─ applyToMobs — mob-buff cancel, reflect-aware   [existing, unchanged]
            └─ Lookup(PriestDispel) → dispel.Apply            [new]
                 ├─ SelectPartyMembersInMap(bitmap)  → member ids
                 ├─ per recipient (caster + members):
                 │    propRoll(e.Prop()) pass? → CancelByTypes → COMMAND_TOPIC_CHARACTER_BUFF
                 └─ summary debug log (bitmap, recipients, cures, prop_skipped)
atlas-buffs consumer (existing)
  └─ CancelByStatTypes → cancels intersecting buffs → EXPIRED status events
atlas-channel buff status consumer (existing)
  └─ EXPIRED → client buff-cancel packet (recipient + foreign)
```

Debuff source (existing, no change): `atlas-monsters` mob skill →
`SkillTypeToDiseaseName` → `applyDiseaseCommandProvider` → `APPLY` on
`COMMAND_TOPIC_CHARACTER_BUFF` → atlas-buffs.

## 6. Error Handling

| Failure | Behavior |
|---|---|
| Party load fails / caster partyless | Selector returns nil → caster-only cure (existing selector contract) |
| Bitmap 0 or ≥128 (incl. a §4.3 decode mismatch) | Selector returns nil → caster-only cure; the summary line records `bitmap` + `recipients_selected=1`, which is the diagnostic signature |
| Prop roll fails for a recipient | That recipient skipped; counted in `prop_skipped`; cast continues |
| Kafka emit fails for a recipient | Error log with recipient id; remaining recipients still processed (FR-7) |
| Recipient has no matching debuffs | atlas-buffs `CancelByStatTypes` no-ops for them (existing behavior); harmless |
| atlas-data has no effect for 2311001 on this version | Cast aborts in `CharacterUseSkillHandleFunc` before `UseSkill`; the handler never runs (upstream behavior, unchanged — surfaced by PRD FR-15) |

The handler never returns a non-nil error; the dispatcher in `UseSkill` logs
handler errors anyway, but Dispel logs its own and returns nil.

## 7. Testing Strategy

Unit (in `atlas-channel`, Builder-pattern setup, no `*_testhelpers.go`):

- **`dispel` handler tests** (internal `package dispel` — they override
  unexported seams; table-driven):
  - caster + N bitmap-selected members, all prop-pass → N+1 emit calls with
    exactly the six type strings, in caster-first order.
  - deterministic prop pattern (pass/fail alternating) → only passing recipients
    emitted; `prop_skipped` count correct; cast never errors.
  - emit error on recipient k → recipients k+1.. still emitted (FR-7).
  - selector receives the exact `(f, casterId, bitmap)` from the packet info;
    empty selector result → caster-only.
  - prop boundaries: 0 → no cures; 1.0 → all cures (no RNG).
  - summary log fields present with correct counts (FR-8/FR-14).
- **Registration test**: `channelhandler.Lookup(skill2.PriestDispel)` returns a
  non-nil handler after the package's `init()` (the `registry_test.go`
  precedent). Asserting on the **Identity** is itself the version-independence
  test — a wire-keyed registration would not compile.
- **No new producer test.** `character/buff/producer_test.go` already pins the
  `CANCEL_BY_TYPES` wire contract; duplicating it in this task would be dead
  weight.

Per-version (§4):

- **Decode audit** (read-only) → a findings table in the task folder, one row
  per version, each with evidence or an explicit "unverified — reason".
- **Byte-layout regression test** in `libs/atlas-packet` — **only** for a
  version whose audit found a divergence, asserting the corrected field order
  round-trips the bitmap (task-111 precedent).
- **Effect-resolution check** per reachable version via `atlas-data`
  (`GET /api/data/skills/2311001` with the tenant's headers), recording the prop
  curve. Documentation, not a test.

Regression:

- Existing `common_apply_to_mobs_test.go` untouched and green (mob half
  byte-for-byte unchanged).
- Existing `healdispel` tests untouched and green (shared producer unchanged).

Acceptance (live): inflict a debuff on a party member from a real mob skill
(e.g. `SkillTypeSeal` → `APPLY {type:"SEAL"}`), cast Dispel, observe
`CANCEL_BY_TYPES` → `EXPIRED` → client buff-cancel packet. Then confirm a
non-dispellable disease (`STUN` or `SEDUCE`) survives the same cast.

## 8. Verification Gates

- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
  `services/atlas-channel/atlas.com/channel` — and in `libs/atlas-packet` if
  §4.3 step 3 touched it.
- `docker buildx bake atlas-channel` from the worktree root.
- `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
  `tools/skill-job-id-guard.sh`, `tools/lint.sh --check` clean from the repo
  root.
- `git diff --stat main...HEAD` shows zero changes under `services/atlas-buffs/`,
  `services/atlas-monsters/`, `skill/handler/common.go`, `character/buff/`, and
  `kafka/message/buff/`.
- `registrations.go` retains all seven pre-existing blank imports.
- Code review (`superpowers:requesting-code-review`) before any PR.

## 9. Coordination

None outstanding. task-156 (`gm-hide-heal-dispel`) has **landed**; it owns the
shared `CancelByTypes` producer and the 11-type `healdispel` purge set.
task-163 consumes that producer unchanged and keeps its own six-type set
handler-local, so the two cure lists stay independent by design.
