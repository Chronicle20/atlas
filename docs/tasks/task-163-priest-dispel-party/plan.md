# Priest Dispel — Party Debuff Cure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

Revised 2026-08-07 — rebased onto `main` @ `e0f5bd01d`. The v1 plan's Task 1
(build the `CANCEL_BY_TYPES` producer) is **deleted**: it is already on `main`.
Task numbering restarts to reflect the real remaining work, and Tasks 3–4 are
new (per-version verification).

**Goal:** Casting Priest Dispel (identity `skill2.PriestDispel`, wire `2311001`
on all supported versions) cures the six dispellable debuffs (CURSE, DARKNESS,
POISON, SEAL, WEAKEN, SLOW) on the caster and bitmap-selected in-map party
members, honoring the skill's per-recipient prop roll — **and that behavior is
verified per supported version, not assumed from v83.**

**Architecture:** One new package, `atlas-channel/skill/handler/dispel`,
registered on the version-blind `skill2.Identity` and calling the **existing**
`buff.Processor.CancelByTypes` (built by task-156). Plus a read-only per-version
audit of the serverbound decode that supplies the party bitmap, with a narrowly
scoped decoder fix only if a divergence is found. Zero changes to `atlas-buffs`,
`atlas-monsters`, `common.go`, or the buff producer.

**Tech Stack:** Go, Kafka (segmentio/kafka-go via atlas-kafka producer), logrus,
atlas-constants, ida-pro-mcp (audit only).

## Global Constraints

- Working directory for Go commands: `services/atlas-channel/atlas.com/channel`
  (module `atlas-channel`) inside the task worktree
  `.worktrees/task-163-priest-dispel-party/`. Never edit the main checkout.
- The cure set is EXACTLY `CURSE, DARKNESS, POISON, SEAL, WEAKEN, SLOW`, held as
  `[]string` built from `charconst.TemporaryStatType` constants.
  STUN / SEDUCE / CONFUSE / UNDEAD / STOP_PORTION / STOP_MOTION / FEAR are
  intentionally excluded.
- **Do not create** any `CANCEL_BY_TYPES` constant, body type, producer, or
  processor method. All exist on `main`. `kafka/message/buff/kafka.go`,
  `character/buff/producer.go`, and `character/buff/processor.go` must end the
  task with zero diffs.
- **Register on the Identity** (`skill2.PriestDispel`), never on
  `skill2.PriestDispelId`. `skill/handler/registry.go` accepts only
  `skill2.Identity`.
- No raw wire-id comparison in the new package. No `constants.For`, no
  `tenant.MustFromContext` inside the handler.
- `services/atlas-buffs/` and `services/atlas-monsters/` must have ZERO diffs.
- `skill/handler/common.go` must be untouched.
- The `registrations.go` edit is **additive** — all seven existing blank imports
  stay.
- No `*_testhelpers.go` files; use the project's Builder pattern for test setup.
- Seams are package-level func vars overridden in tests with `t.Cleanup`
  restore.
- The handler never returns a non-nil error.
- Commit messages: `<type>(task-163): <summary>` on branch
  `task-163-priest-dispel-party`.

---

### Task 0: Confirm the `main` baseline (no code changes)

Guards against re-implementing landed work. If any assertion below fails, STOP
and re-read `main` before proceeding — the plan's premises have moved again.

- [x] **Step 1: Assert the producer surface exists**

From the worktree root:
```bash
grep -n "CommandTypeCancelByTypes\|CancelByTypesCommandBody" services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go
grep -n "func CancelByTypesCommandProvider" services/atlas-channel/atlas.com/channel/character/buff/producer.go
grep -n "CancelByTypes(f field.Model, characterId uint32, types \[\]string) error" services/atlas-channel/atlas.com/channel/character/buff/processor.go
```
Expected: all three hit. The processor signature is **uncurried, `[]string`,
returns `error`** — the handler adapts to it.

- [x] **Step 2: Assert the registry is Identity-keyed**

```bash
grep -n "map\[skill2.Identity\]Handler\|func Register\|func Lookup" services/atlas-channel/atlas.com/channel/skill/handler/registry.go
```
Expected: `registry map[skill2.Identity]Handler`, `Register(id skill2.Identity, h Handler)`.

- [x] **Step 3: Record the current registrations list**

```bash
cat services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go
```
Expected: seven blank imports (heal, healdispel, hide, mprecovery, mysticdoor,
resurrection, timeleap). Note them — Task 2 adds an eighth and removes none.

- [x] **Step 4: Confirm the identity↔wire binding on every version**

```bash
grep -rn "PriestDispel: *2311001" libs/atlas-constants/skill/ | wc -l
```
Expected: `11`. If it is not 11, §3.1 of the PRD is stale and the version table
must be regenerated before continuing.

---

### Task 1: Dispel handler subpackage with tests

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/dispel/dispel.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/dispel/dispel_test.go`
  (new, internal `package dispel` — it overrides unexported seams)

**Interfaces:**
- Consumes:
  - `channelhandler.Register(id skill2.Identity, h channelhandler.Handler)` /
    `channelhandler.Lookup(id skill2.Identity) (Handler, bool)`
    (`skill/handler/registry.go`).
  - `channelhandler.SelectPartyMembersInMap(l, ctx, f, casterId, memberBitmap byte) []channelhandler.PartyRecipient`
    (`skill/handler/recipients.go:188`) — filters offline / other-map /
    no-session / dead, decodes the MSB-first bitmap, returns nil for
    `bitmap == 0 || bitmap >= 128`.
  - `channelhandler.PartyRecipient.Id()` and `channelhandler.NewPartyRecipientBuilder()` (tests).
  - `buff.NewProcessor(l, ctx).CancelByTypes(f field.Model, characterId uint32, types []string) error`
    (`character/buff/processor.go:82`) — existing, unchanged.
  - `e.Prop() float64` — already normalized 0.0–1.0; no `/100`.
  - `skill2.PriestDispel` (`libs/atlas-constants/skill/identities_gen.go:177`) — the **Identity**.
  - `charconst.TemporaryStatTypeCurse/Darkness/Poison/Seal/Weaken/Slow`.
- Produces: `dispel.Apply` matching `channelhandler.Handler`, registered for
  `skill2.PriestDispel` in `init()`.

- [x] **Step 1: Write the failing tests**

Create `dispel_test.go` as internal `package dispel`. Cover, at minimum:

| Test | Asserts |
|---|---|
| `TestDispelRegisteredOnIdentity` | `channelhandler.Lookup(skill2.PriestDispel)` returns `ok=true`, non-nil handler. Uses the **Identity** — a wire-keyed registration would not compile, so this is the version-independence test. |
| `TestDispelCuresCasterAndMembersInOrder` | all prop-pass → one emit per recipient, caster first then selector order; the types slice is exactly the six strings in order |
| `TestDispelSelectorReceivesCastArgs` | selector sees the exact `(f, casterId, bitmap)` from `info` |
| `TestDispelEmptySelectorCastsCasterOnly` | nil selector result → caster-only emit |
| `TestDispelPropRollPerRecipient` | alternating pass/fail → only passing recipients emitted; roll called once per recipient; cast returns nil |
| `TestDispelEmitErrorContinues` | emit error on recipient k → recipients after k still emitted; cast returns nil |
| `TestDispelZeroPropCuresNobody` | real `propRollFunc` with `effect.Model{}` (prop 0) → zero emits |
| `TestPropRollBoundaries` | `propRollFunc`: 0 and −0.5 → false; 1.0 and 1.5 → true |
| `TestDispelSummaryLogFields` | a `dispel_party_cure_summary` Debug entry carries `caster`, `bitmap`, `recipients_selected`, `cures_emitted`, `prop_skipped` with correct values (this is the FR-14 diagnostic) |

Harness notes:
- Install the three seams (`selectPartyMembersFunc`, `propRollFunc`,
  `cancelByTypesFunc`) with `t.Cleanup` restore.
- `cancelByTypesFunc` now has signature
  `func(l, ctx, f field.Model, characterId uint32, types []string) error` —
  the stub records `(characterId, types)` per call, not a bound operator.
- Build `info` with `packetmodel.NewSkillUsageInfoBuilder().SetSkillId(2311001).SetSkillLevel(1).SetAffectedPartyMemberBitmap(0x30).Build()`.
- `effect.Model{}` zero value has Prop 0 → the real roll always fails; tests
  needing passes must build via `effect.Extract(effect.RestModel{Prop: 1.0})`.
- Use `logrus/hooks/test` (`logtest.NewNullLogger()`) for the summary assertion.

- [x] **Step 2: Run tests to verify they fail**

```bash
go test ./skill/handler/dispel/ -v
```
Expected: compile failure — package does not exist.

- [x] **Step 3: Implement the handler**

Create `dispel.go`. Shape (design §3.2):

```go
func init() {
    // Identity, not wire id: registry.go is keyed on skill2.Identity and
    // UseSkill resolves the incoming wire id through
    // constants.For(...).Skill.Resolve before Lookup (task-187). One
    // registration covers all provisioned versions.
    channelhandler.Register(skill2.PriestDispel, Apply)
}

// dispellableStatTypes is the exact Dispel cure set (PRD FR-5, Cosmic
// Character.dispelDebuffs parity). []string to match the shared
// buff.Processor.CancelByTypes signature (healdispel.diseaseTypes precedent).
// STUN / SEDUCE / CONFUSE / UNDEAD / STOP_PORTION / STOP_MOTION / FEAR are
// intentionally excluded — atlas-monsters can inflict them, but they are
// cure-all (purgeDebuffs) semantics owned by SuperGM Heal+Dispel.
var dispellableStatTypes = []string{
    string(charconst.TemporaryStatTypeCurse),
    string(charconst.TemporaryStatTypeDarkness),
    string(charconst.TemporaryStatTypePoison),
    string(charconst.TemporaryStatTypeSeal),
    string(charconst.TemporaryStatTypeWeaken),
    string(charconst.TemporaryStatTypeSlow),
}

var selectPartyMembersFunc = channelhandler.SelectPartyMembersInMap

// Mirrors the unexported propRollFunc in skill/handler/common.go exactly.
// e.Prop() is pre-normalized 0.0–1.0 — no /100.
var propRollFunc = func(prop float64) bool {
    if prop <= 0 { return false }
    if prop >= 1 { return true }
    return rand.Float64() <= prop
}

var cancelByTypesFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, types []string) error {
    return buff.NewProcessor(l, ctx).CancelByTypes(f, characterId, types)
}
```

`Apply(l)(ctx)(wp, f, characterId, info, e) error`:
1. `bitmap := info.AffectedPartyMemberBitmap()`;
   `members := selectPartyMembersFunc(l, ctx, f, characterId, bitmap)`.
2. `recipients := [characterId] + members[i].Id()`.
3. Per recipient: `if !propRollFunc(e.Prop()) { propSkipped++; continue }`;
   `if err := cancelByTypesFunc(l, ctx, f, recipientId, dispellableStatTypes); err != nil { log error; continue }`;
   `curesEmitted++`.
4. One `Debug("dispel_party_cure_summary")` with
   `caster, skill_id, skill_level, bitmap, recipients_selected, cures_emitted, prop_skipped`.
5. Return `nil` always.

Do **not** import `constants` or `tenant`; do **not** call `skill2.Is` on
`info.SkillId()`.

- [x] **Step 4: Run tests to verify they pass**

```bash
go test -race ./skill/handler/dispel/ -v && go vet ./skill/handler/dispel/
```
Expected: all tests PASS, vet clean.

- [x] **Step 5: Verify nothing shared moved**

From the worktree root:
```bash
git status --porcelain -- \
  services/atlas-channel/atlas.com/channel/skill/handler/common.go \
  services/atlas-channel/atlas.com/channel/character/buff/ \
  services/atlas-channel/atlas.com/channel/kafka/message/buff/ \
  services/atlas-buffs/ services/atlas-monsters/
```
Expected: EMPTY.

- [x] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/dispel/
git commit -m "feat(task-163): Priest Dispel party debuff cure handler"
```

---

### Task 2: Registration, docs, and module-wide verification

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`
- Modify: `services/atlas-channel/docs/domain.md`

- [x] **Step 1: Add the blank import (additive)**

Insert one line into the existing import block, keeping all seven current
imports:

```go
_ "atlas-channel/skill/handler/dispel"       // Priest Dispel party cure — task-163
```

Verify nothing was dropped:
```bash
grep -c "atlas-channel/skill/handler/" services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go
```
Expected: `8`.

- [x] **Step 2: Document the handler**

Add a paragraph to `services/atlas-channel/docs/domain.md` next to the existing
`skill/handler/healdispel` entry, matching its shape: registered identity,
recipient selection (caster + bitmap-selected in-map party members, map-wide),
the six-type cure set and why STUN/SEDUCE/CONFUSE are excluded, per-recipient
prop roll, and the log-and-continue failure policy.

- [x] **Step 3: Run the full module verification**

From `services/atlas-channel/atlas.com/channel`:
```bash
go build ./... && go vet ./... && go test -race ./...
```
Expected: all clean / all tests PASS — including the untouched `healdispel`,
`heal`, `resurrection`, and `common_apply_to_mobs_test.go` suites.

- [x] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go services/atlas-channel/docs/domain.md
git commit -m "feat(task-163): register dispel handler and document it"
```

---

### Task 3: Per-version audit of the Dispel serverbound decode (read-only)

**Files:**
- Create: `docs/tasks/task-163-priest-dispel-party/version-findings.md`

**Why this task exists:** `libs/atlas-packet/model/skill_usage_info.go` gates
three optional field groups on hard-coded, **version-blind** raw wire-id lists
(`isAntiRepeatBuffSkill`, `isPartyBuff`, `isMobAffectingBuff`), each carrying a
`// TODO this is not all inclusive` comment. `2311001` is in all three, so the
assumed layout is
`updateTime(4) skillId(4) slv(1) castX(2) castY(2) bitmap(1) delay(2) mobCount(1) mobIds(4×N) delay(2)`
— with `delay` read twice. Those lists were derived from **v83 only**. If the
membership disagrees with a client, every field after the mismatch reads at the
wrong offset, the bitmap decodes to 0, `SelectPartyMembersInMap` returns nil,
and Dispel silently becomes caster-only with no error. That exact bug shipped
twice already (task-111 Bishop Resurrection 2321006; task-155/PR#1136 Buccaneer
Time Leap 5121010).

**This task changes no code.** Its output is evidence.

- [x] **Step 1: Record the current decoder's assumed layout**

Read `libs/atlas-packet/model/skill_usage_info.go` and write the exact assumed
field order for `2311001` into `version-findings.md`, noting the double `delay`
read. This is the hypothesis every version is tested against.

- [x] **Step 2: Audit each reachable version**

For each of gms_48, 61, 72, 79, 83, 84, 87, 92, 95, jms_185 (gms_12 is excluded
— Step 3), determine from that version's client whether a `2311001` cast writes
castX/castY, the affected-member bitmap, and the mob list:

- Prefer the checked-in export under `docs/packets/audits/<version>/` where one
  exists (all but gms_92); cite the export hash from
  `docs/packets/audits/STATUS.md`.
- Otherwise read the IDB via ida-pro-mcp — confirm the instance matches the
  version before reading, and use `func_query` with `name_regex` (the documented
  method). Target functions, by v83 name: `CUserLocal::SendSkillUseRequest`
  (@0x96d399 on v83), `is_antirepeat_buff_skill` (@0x96d6ca on v83),
  `CUserLocal::FindParty` (@0x96db3f on v83). Name any unnamed equivalents while
  you are in the IDB.
- gms_92 has **no matrix column and no export** — a named v92 IDB exists, so
  derive from IDA and record the finding here (there is no evidence file to
  pin).

Record one row per version:

| Version | castX/castY written? | bitmap written? | mob list written? | Matches current decoder? | Evidence |

- [x] **Step 3: Record gms_12 as unreachable**

```bash
grep -c CharacterUseSkillHandle services/atlas-configurations/seed-data/templates/template_gms_12_1.json
grep -c '"handler"' services/atlas-configurations/seed-data/templates/template_gms_12_1.json
```
Expected: `0` and `24`. Record: no skill-use packet is routed on gms_12, so no
per-skill handler fires; Dispel is unreachable there and out of scope.

- [x] **Step 4: Confirm per-version effect data**

For each reachable version, confirm `atlas-data` resolves an effect for
`2311001` and record the prop curve (`GET /api/data/skills/2311001` with that
tenant's region/major/minor headers). A version with no effect aborts the cast
in `CharacterUseSkillHandleFunc` before `UseSkill` — a silent, version-specific
no-op the handler cannot compensate for. Do **not** cite prop values from
memory; the v83 figures (34 at L1 → 100 at L20) are a reference to re-verify,
not a source.

- [x] **Step 5: Mark the unreadable honestly**

Any version whose client could not be read is recorded as **unverified, with the
reason** (no IDB / instance unavailable / function not located). It is never
folded into the v83 assumption and never reported as passing. Same for any
version whose `atlas-data` lookup could not be run.

- [x] **Step 6: Commit**

```bash
git add docs/tasks/task-163-priest-dispel-party/version-findings.md
git commit -m "docs(task-163): per-version audit of the Dispel skill-use decode"
```

---

### Task 4: Decoder fix for divergent versions (CONDITIONAL — skip if Task 3 found none)

**Files (only if triggered):**
- Modify: `libs/atlas-packet/model/skill_usage_info.go`
- Test: `libs/atlas-packet/model/skill_usage_info_test.go`

Run this task **only** for versions Task 3 recorded as diverging from the
current decoder. If every reachable version confirmed, skip to Task 5 and note
in `version-findings.md` that no code change was required.

- [x] **Step 1: Write the failing byte-layout test**

For each divergent version, add a test that feeds the true byte sequence for a
`2311001` cast (as derived in Task 3) and asserts
`AffectedPartyMemberBitmap()` decodes to the expected non-zero value. The
task-111 precedent: the wire-layout regression test lands with the fix.

- [x] **Step 2: Correct the decode for those versions only**

Scope discipline — the fix makes **Dispel's** bitmap decode correctly on every
reachable version. It does **not** become a general version-aware rewrite of the
three membership lists for all ~50 skills; that is a separate task with its own
evidence budget. Leave the `// TODO this is not all inclusive` comments in place
and add a comment citing this task's finding and its evidence.

- [x] **Step 3: Verify**

From `libs/atlas-packet`:
```bash
go test -race ./... && go vet ./... && go build ./...
```
Then from `services/atlas-channel/atlas.com/channel`:
```bash
go test -race ./... && go build ./...
```
Expected: clean in both — a decoder change ripples into every skill handler.

- [x] **Step 4: Commit**

```bash
git add libs/atlas-packet/model/
git commit -m "fix(task-163): correct Dispel skill-use decode on <versions>"
```

---

### Task 5: Cross-module verification gates

**Files:** none created or modified — verification only (CLAUDE.md Build &
Verification + design §8). A failure here reopens the offending task; do not
claim done until every gate passes.

- [x] **Step 1: Docker bake the changed service**

From the worktree root:
```bash
docker buildx bake atlas-channel
```
Expected: image builds. (`go build` will NOT catch a missing `COPY libs/...` in
the shared Dockerfile — only bake will. No new lib was added, so no Dockerfile
edits are expected.)

- [x] **Step 2: Repo guards**

From the worktree root:
```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/skill-job-id-guard.sh
tools/lint.sh --check
```
Expected: all exit 0. `skill-job-id-guard.sh` is the one that would catch a raw
wire-id comparison slipping into the new package. `lint.sh --check` needs nvm22
on PATH for its frontend leg — run `tools/lint.sh` (fix mode) before committing.

- [x] **Step 3: Confirm the negative-space constraints**

From the worktree root:
```bash
git diff --stat main...HEAD -- \
  services/atlas-buffs/ services/atlas-monsters/ \
  services/atlas-channel/atlas.com/channel/skill/handler/common.go \
  services/atlas-channel/atlas.com/channel/character/buff/ \
  services/atlas-channel/atlas.com/channel/kafka/message/buff/
grep -c "atlas-channel/skill/handler/" services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go
git log --oneline main..HEAD
```
Expected: the `diff --stat` output is EMPTY; the grep prints `8`; the log shows
the task's doc + code commits.

- [x] **Step 4: Re-run the changed-module gates**

From `services/atlas-channel/atlas.com/channel`:
```bash
go test -race ./... && go vet ./... && go build ./...
```
And, if Task 4 ran, from `libs/atlas-packet`:
```bash
go test -race ./... && go vet ./... && go build ./...
```

- [x] **Step 5: Confirm the per-version claim is backed**

Open `docs/tasks/task-163-priest-dispel-party/version-findings.md` and confirm
every one of the 11 versions has a row with either evidence or an explicit
"unverified — reason". A blank row, or a version silently absent, means the
"all supported versions" claim is not yet earned — do not report it as met.

- [x] **Step 6: Code review**

Run `superpowers:requesting-code-review` (mandatory before any PR). It should
dispatch `plan-adherence-reviewer` and `backend-guidelines-reviewer`; findings
land in `docs/tasks/task-163-priest-dispel-party/audit.md`.

---

## Out of Scope (do not implement)

- Any `CANCEL_BY_TYPES` producer / message type / processor method — all exist
  on `main` (task-156). Reshaping the existing `CancelByTypes` signature is also
  out of scope: `healdispel` and the Homing Beacon path depend on it.
- SuperGM Heal+Dispel (`9101000`) — landed as task-156.
- Mob-side Dispel changes (`applyToMobs`, reflect handling) — already
  implemented.
- Curing STUN / SEDUCE / CONFUSE / UNDEAD / STOP_PORTION / STOP_MOTION / FEAR.
- Mob→character disease infliction — already exists in `atlas-monsters`
  (`applyDiseaseCommandProvider`). It is the debuff *source* used in acceptance,
  not a target of change.
- A general version-aware rewrite of `SkillUsageInfo.Decode`'s three membership
  lists for all skills. Task 4 fixes Dispel's read path only.
- Making Priest Dispel reachable on `gms_12_1` — that template routes no
  `CharacterUseSkillHandle` at all; wiring the skill-use path into a login-only
  template is a separate task.
- Skill-use announce packets — `CharacterUseSkillHandleFunc` already emits self
  and foreign announces for every cast, Dispel included.
- Seed-template changes — no new opcode, handler, or writer is introduced.
- `atlas-buffs` or `atlas-monsters` changes of any kind.
