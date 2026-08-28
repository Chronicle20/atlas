# JMS 185 CharacterData Divergences — Master Level & Extended SP — Design

Version: v1
Status: Draft
Created: 2026-08-28
Phase: 2 (design)
Input: `docs/tasks/task-275-jms185-characterdata-divergences/prd.md` (approved)

---

## 1. What this design decides

The PRD establishes *what* is wrong: two `GW_CharacterData` predicates
(`is_skill_need_master_level` and the extended-SP job test) are modelled from
GMS ≤ v87 clients and are wrong for GMS v92, GMS v95 and JMS 185. This document
decides *where the corrected predicates live*, *what shape their version
parameter takes*, and *how the fix is proven without moving a verified byte*.

It also closes all four of the PRD's §9 open questions with IDA evidence read
during this phase. Two of them changed the scope of the task; see §3.

---

## 2. Client truth re-confirmed in this phase

Every decompile below was read fresh from the checked-in IDBs on 2026-08-28.
Where it differs from or extends the PRD table, that is called out.

### 2.1 `is_skill_need_master_level`

| Version | Address | 430-434 arm |
|---|---|---|
| GMS v83 | `is_skill_need_master_level` @0x4e8f04 | **no arm** — falls through to `job%100 != 0 && job%10 == 2` |
| GMS v84 | `sub_4F0AD2` @0x4f0ad2 | **no arm** — same fallthrough |
| GMS v87 | `is_skill_need_master_level` @0x508f33 | arm present, `v1/10 == 43 \|\| !(v1%100)` → **return 0** |
| GMS v92 | `sub_4792F0` @0x4792f0 | `get_job_level(job) == 4 \|\| skill ∈ {4311003, 4321000, 4331002, 4331005}` |
| GMS v95 | `is_skill_need_master_level` @0x47ccb0 | same as v92, behind an `is_ignore_master_level_for_common` early-out |
| JMS 185 | `is_skill_need_master_level` @0x47d2a8 | same as v92; **no** Evan exception list |

**Correction to the PRD's reading of the "no arm" versions.** The PRD's table
records v83/v84 as "no arm — falls through". That is right, but the PRD's
Acceptance Criteria then says the test should assert "no arm on v83/v84, false
on v87". Those are not the same answer. On v83/v84 the fallthrough is
`job%100 != 0 && job%10 == 2`, which is **true for job 432** (`432%10 == 2`) and
false for 430/431/433/434. On v87 the arm returns false for all five. The table
test must assert the *observable* result per version, not the shape of the
client's control flow:

| job | v83 | v84 | v87 | v92 | v95 | jms185 |
|---|---|---|---|---|---|---|
| 430 | false | false | false | false | false | false |
| 431 | false | false | false | false¹ | false¹ | false¹ |
| 432 | **true** | **true** | false | false¹ | false¹ | false¹ |
| 433 | false | false | false | false¹ | false¹ | false¹ |
| 434 | false | false | false | **true** | **true** | **true** |

¹ except the four named skill ids, which are true regardless of job level.

### 2.2 `get_job_level` (OQ3 — resolved)

```
GMS v95 @0x47cb90        JMS 185 @0x47d347          GMS v92 sub_479260 @0x479260
if job%100==0 || job==2001: return 1
if job/10 == 43:  v = (job-430)/2   |  v = (job%10)/2   |  v = (job-430)/2
else:             v = job%10
lvl = v + 2
return lvl if lvl>=2 && (lvl<=4 || (lvl<=10 && is_evan_job(job))) else 0
```

`is_evan_job` (GMS v95 @0x47cad0) is `job/100 == 22 || job == 2001`; JMS inlines
`job/100 == 22` (equivalent here, because `job == 2001` already returned 1).
GMS v92's `sub_4791C0` occupies the same slot as v95's `is_evan_job`.

Two consequences:

- **`get_job_level` is pure integer arithmetic.** It needs no ported table and
  no WZ data. FR-3's "must come from the existing job model, not a
  re-derivation" is satisfied by porting this function verbatim into
  `atlas-constants/job`, *not* by reusing `job.Advancement`
  (`libs/atlas-constants/job/advancement.go`), which is a different function:
  `Advancement` returns -1 for the whole Evan stage line and has no 43x branch.
  Reusing `Advancement` here would be the same class of mistake as task-218's
  `GetSkillBook` off-by-one, which the existing comment at
  `libs/atlas-constants/skill/model.go:98-102` already warns about.
- **GMS and JMS disagree on the 43x expression** — `(job-430)/2` vs `(job%10)/2`
  — but agree on every value in 430-434 (0,0,1,1,2). They diverge only for
  job ≥ 435, which no client has. Model the GMS form and pin the equality in a
  test comment; do not branch on it.

For 430-434 the arm therefore collapses to `job == 434 || skill ∈ {4311003,
4321000, 4331002, 4331005}`. The port keeps the `get_job_level(job) == 4` form
anyway, because that is what the client computes and a future 43x id would
follow it.

### 2.3 `is_ignore_master_level_for_common` (OQ2 — resolved, scope grew)

GMS v95 @0x47cc20 is a flat 16-id membership test. Its callers see a **false**
`is_skill_need_master_level` for every member. Decompiled id set (IDA rendered
four of these as `&loc_*` addresses; the immediates are 4220009 = 0x406469,
5120011 = 0x4E200B, 5220012 = 0x4FA6AC, 4120010):

```
1120012 1220013 1320011
2120009 2220009 2320010
3120010 3120011 3220009 3220010
4120010 4220009
5120011 5220012
32120009 33120010
```

**These are reachable today.** Fourteen of the sixteen belong to jobs 112, 122,
132, 212, 222, 232, 312, 322, 412, 422, 512, 522 — ordinary 4th-job identities
an Atlas tenant can create. Every one of them satisfies the common fallthrough
(`job%10 == 2`), so Atlas currently writes a master-level `Decode4` for them on
GMS v95 where the client reads none. That is a live 4-byte-per-skill shift in
`CharacterData` for any GMS v95 4th-job character, of exactly the same class as
the bug this task exists to fix. The remaining two ids (jobs 3212, 3312) match no
Atlas job identity; they are modelled anyway because membership is a flat id
list and omitting entries would make the list a partial port.

The PRD's §2 non-goal — "Modelling `is_ignore_master_level_for_common` beyond
what is needed to keep v95 correct for jobs Atlas can create" — is therefore
satisfied only by modelling the whole list. **This design treats it as in
scope**, as a v95-only arm.

### 2.4 Extended-SP job test

| Version | Address | Predicate |
|---|---|---|
| GMS ≤ 83 | — | no extended-SP path (Evan launched at v84) |
| GMS v84 | inline in `GW_CharacterStat::Decode` @0x4e9da4 | `job/100 == 22 \|\| job == 2001` |
| GMS v87 | inline @0x501e9c | `job/100 == 22 \|\| job == 2001` |
| GMS v92 | inline @0x4f50f4 / @0x4f5100 / @0x4f510f | `job/1000 == 3 \|\| job/100 == 22 \|\| job == 2001` |
| GMS v95 | `is_extendsp_job` @0x4f1e30 | `job/1000 == 3 \|\| job/100 == 22 \|\| job == 2001` |
| JMS 185 | `sub_5163A2` @0x5163a2, called @0x50eda2 | `job/1000 == 3 \|\| job/100 == 22 \|\| job == 2001` |

v95 and JMS confirmed by direct decompile this phase (both are one-line
functions and read identically). v84/v87/v92 are the PRD's inline readings,
unchanged.

Body, all versions: `Decode1` count, then count × (`Decode1` master-level index,
`Decode1` sp). JMS `sub_50E8B0` @0x50e8b0. Atlas already writes this shape.

---

## 3. Open questions closed

**OQ1 — does `DecodeChangeStat` share the helper? No, and it cannot.**
`libs/atlas-packet/stat/clientbound/changed.go` encodes a stat-type-keyed switch
(`changed.go:84`, `:135`) with **no job id in scope at all** — it holds only
`(statType, value)` pairs. It writes `TypeAvailableSP` as an unconditional
`WriteShort`. It is a genuine, pre-existing divergence of the same class (an
Evan on GMS ≥ 84 receiving `OnStatChanged` with an SP update gets a 2-byte
short where the client reads a 1-byte-counted array), but fixing it requires
threading the character's job into the stat-change model — a data-model change
the PRD explicitly excludes (§2, §6). **Out of scope; a follow-up task.** The
fix in this task cannot reach it implicitly, so no fixture coverage is owed
here. Recorded in §9.

**OQ2 — `is_ignore_master_level_for_common`: in scope.** See §2.3.

**OQ3 — `get_job_level`: portable verbatim, no table.** See §2.2. It does *not*
reuse `job.Advancement`.

**OQ4 — where do the predicates live: `libs/atlas-constants/job`.** See D2.

---

## 4. Architecture decisions

### D1 — The version parameter is `(region string, major uint16)`, not `tenant.Model`

The PRD's FR-1 records "the confirmed decision is that it takes the tenant",
while §5 leaves the exact shape to this phase. **This design does not pass
`tenant.Model`,** for a reason that is a hard constraint rather than a
preference:

`libs/atlas-constants/go.mod` requires exactly one non-indirect dependency,
`github.com/google/uuid`. It does not depend on `libs/atlas-tenant`. That is
deliberate and load-bearing: the package's own tenant-keyed selector,
`constants.For(region string, major, minor uint16)`
(`libs/atlas-constants/constants/for.go:39`), takes decomposed scalars for
precisely this reason, and all ~20 call sites across `atlas-channel` and
`atlas-packet` already spell it `constants.For(t.Region(), t.MajorVersion(),
t.MinorVersion())`. Passing `tenant.Model` into `atlas-constants` would add a
new module edge to the repo's most-depended-upon library to save two argument
tokens at two call sites.

Alternatives weighed:

- **(A) `(region string, major uint16)` — chosen.** Matches `constants.For`
  exactly. Zero new module edges. Call sites read
  `job.NeedsMasterLevel(id, t.Region(), t.MajorVersion())`, which is the
  idiom already in the tree. Satisfies the PRD's actual requirement — "no
  caller passes a hand-derived `region == "GMS"` boolean" — because the caller
  now passes raw tenant facts and the *callee* owns every arm.
- **(B) `tenant.Model` parameter.** Rejected: new `atlas-constants` →
  `atlas-tenant` module dependency, contradicting the established
  decomposition boundary. Also makes the predicate untestable without
  constructing a tenant in a package that has never needed one.
- **(C) A resolved `MasterLevelRules` value struct returned once per encode,
  with a `.Needs(skillId)` method.** Rejected as YAGNI. The NFR is "must not
  allocate per skill"; option A allocates nothing — it is integer arithmetic
  and a `switch` on two scalars. The struct buys nothing and adds a lifetime
  to reason about.

**This is the one place this design departs from a PRD statement.** If the
"takes the tenant" decision was load-bearing for a reason not captured in the
PRD, this is the section to override before planning.

### D2 — Both predicates move to `libs/atlas-constants/job`

`NeedsMasterLevel` currently lives in `libs/atlas-constants/skill/model.go:115`.
It cannot stay there once it needs `get_job_level`: `job` imports `skill`
(`job/model.go:6`), so `skill` importing `job` is an import cycle, and the
`skill` package would have to re-implement job-level semantics that belong to
`job`.

The `job` package is the correct home for all three functions:

| Function | Home | Signature |
|---|---|---|
| `ClientJobLevel` | `job` | `func ClientJobLevel(jobId Id) int` |
| `NeedsMasterLevel` | `job` | `func NeedsMasterLevel(skillId skill.Id, region string, major uint16) bool` |
| `UsesExtendedSP` | `job` | `func UsesExtendedSP(jobId Id, region string, major uint16) bool` |

This also answers OQ4 affirmatively for the extended-SP predicate: it moves out
of `libs/atlas-packet/character` (`isEvanJob`, `data.go:271`) into
`atlas-constants/job`, next to the job identities and next to the master-level
rule, so both `GW_CharacterData` predicates have one home.

Per CLAUDE.md ("Prefer straightforward moves over re-exported type aliases"),
`skill.NeedsMasterLevel` is **deleted**, not forwarded. Its tests
(`skill/model_test.go:TestNeedsMasterLevelMatchesClientRule`) move with it to
`job/master_level_test.go`. `libs/atlas-packet/character/data.go` gains a `job`
import; the two call sites (`data.go:670`, `data.go:696`) change to
`job.NeedsMasterLevel(...)`.

`ClientJobLevel` is exported rather than unexported because it is a named client
function with its own address per version and its own test obligations, and
because `job.Advancement` sitting beside it makes the distinction between the two
worth stating in the type system rather than in a comment.

### D3 — Version arms are a `switch`, not a data table

`NeedsMasterLevel` resolves three independent behaviours from `(region, major)`:
the Evan exception list, the Dual Blade arm, and the v95 ignore list. A
generated per-version table (the `version_<r>_<maj>_<min>_gen.go` mechanism the
package uses for identity Sets) is the wrong tool: those tables are generated
from WZ manifests, whereas these arms are hand-read from decompiles and each
carries an IDA address that must appear in a comment next to the code it
justifies. Three small helpers with explicit `MajorAtLeast`-style comparisons
keep the address citation adjacent to the arm:

```go
// hasEvanExceptions reports whether this client's Evan arm carries the
// three-skill exception list. Present GMS v84 (@0x4f0ad2) .. v95 (@0x47ccb0);
// absent GMS v83 (@0x4e8f04) and JMS 185 (@0x47d2a8).
func hasEvanExceptions(region string, major uint16) bool

// dualBladeArm reports which of the three shapes this client's 430-434 arm
// takes: none (GMS <= 84), always-false (GMS 87 @0x508fa4), or job-level
// (GMS >= 92 @0x479371, @0x47ccb0; JMS 185 @0x47d2f9).
func dualBladeArm(region string, major uint16) dualBladeShape

// ignoresCommonMasterLevel reports whether this client early-outs on
// is_ignore_master_level_for_common. GMS v95 only (@0x47cc20).
func ignoresCommonMasterLevel(region string, major uint16) bool
```

`region` is compared case-insensitively via the same `strings.ToUpper`
normalisation `constants.For` applies (`for.go:40`), so a caller passing a
lower-case region cannot silently take the GMS-83 baseline arm.

**Unprovisioned-version fallback.** Unlike `constants.For`, these predicates do
not fall back to a baseline Set — they fall through their `switch` to the arm
the nearest lower provisioned version uses, because the arms are monotone in
`major` within a region. A JMS tenant at any major takes the JMS arms; a GMS
tenant at major 90 takes the ≥ 87 arms. This is stated in a comment; there is
no logging, because the predicate runs per skill inside an encode loop.

### D4 — Encode and decode read one exported helper (FR-8)

`encodeStats` (`data.go:316`) and `decodeStats` (`data.go:397`) both become:

```go
if job.UsesExtendedSP(job.Id(m.Stats.JobId), t.Region(), t.MajorVersion()) {
```

The mirror-image property is enforced structurally (one call, two sites) and
asserted by the round-trip tests in §6. `isEvanJob` and its test
(`data_evan_test.go:13`) are deleted; the test's cases move into
`job/extended_sp_test.go` and gain per-version columns.

`m.Stats.JobId` is `uint16` and `job.Id` is `uint16`
(`job/constants.go:3`), so the conversion is free.

### D5 — The extended-SP *body* is unchanged; only the gate moves

`encodeStats` writes a literal `w.WriteByte(0)` count and `decodeStats` reads
the count and discards the pairs. Atlas has no per-master-level SP model, so
there is nothing to write; PRD §6 forbids a data-model change. This design keeps
that shape, which means:

- Encode/decode stay lossless round-trip **at count 0**, which is the only value
  Atlas produces. The round-trip test asserts that explicitly rather than
  implying generality.
- The decode side's discard loop is retained (it must consume a nonzero count
  from a client-authored fixture correctly), and a decode-only test feeds a
  count-2 fixture to prove the reader advances the right number of bytes.

Modelling real extended-SP contents is recorded as a follow-up in §9.

### D6 — No behaviour change for `atlas-channel`

`services/atlas-channel/atlas.com/channel/socket/writer/character_data.go:76-80`
holds only comment references to the old `skill.NeedsMasterLevel` prose. Those
comments are updated to name `job.NeedsMasterLevel` and to state that the
predicate is now version-resolved. No code there changes; the tenant is already
in scope at the writer.

### D7 — The stale comment block is rewritten, not patched

`skill/model.go:80-114` is a 35-line comment whose Dual Blade paragraph is now
false ("atlas-constants defines no 430-434 job" — `BladeRecruit`…`BladeMaster`
exist at `job/identities_gen.go:41-45` and are bound and available in
`job/version_jms_185_1_gen.go:41-45,219-223` since task-204). Since the function
moves (D2), the comment moves with it and is rewritten around the per-version
table in §2.1 and §2.4, one IDA address per arm, satisfying FR-5 and the
NFR on citations.

---

## 5. Resulting behaviour, per version

`NeedsMasterLevel(skillId, region, major)`:

```
if ignoresCommonMasterLevel(region, major) && isIgnoredCommonSkill(skillId):
    return false                                  # GMS v95 only, @0x47cc20
job := skillId / 10000
if job/100 == 22 || job == 2001:                  # Evan
    lvl := ClientJobLevel(job)
    if lvl == 9 || lvl == 10: return true
    return hasEvanExceptions(region, major) &&
           skillId in {22111001, 22141002, 22140000}
if job/10 == 43:                                  # Dual Blade
    switch dualBladeArm(region, major):
      case none:        break                     # fall through to common
      case alwaysFalse: return false              # GMS v87
      case jobLevel:    return ClientJobLevel(job) == 4 ||
                               skillId in {4311003, 4321000, 4331002, 4331005}
if job%100 == 0: return false
return job%10 == 2
```

`UsesExtendedSP(jobId, region, major)`:

```
if region == "GMS" && major < 84: return false    # no extended-SP path
if region == "GMS" && major < 92:                 # v84 @0x4e9da4, v87 @0x501e9c
    return jobId/100 == 22 || jobId == 2001
return jobId/1000 == 3 || jobId/100 == 22 || jobId == 2001   # v92, v95, JMS
```

Byte-movement audit against §2's tables, for jobs an Atlas tenant can create
today:

| Version | Master level | Extended SP |
|---|---|---|
| GMS ≤ 83 | unchanged (no 43x job creatable; Evan arm identical) | unchanged (gate stays false) |
| GMS v84, v87 | unchanged | unchanged (narrow predicate preserved) |
| GMS v92 | unchanged (`/1000 == 3` arm unreachable — the only 3xx identities are Bowman 300 … Marksman 322, all < 1000 after `/1000`) | unchanged, same reason |
| **GMS v95** | **CHANGES** — 14 reachable 4th-job skills lose their master-level int (§2.3). This is a bug fix, and the v95 `SET_FIELD` evidence must be re-pinned if its fixture holds any of the 16 ids. | unchanged |
| JMS 185 | changes only for 43x jobs (previously fell through to `job%10 == 2`, so 432 was wrong) | **CHANGES** — Evan jobs now take the array instead of the short |

The GMS v95 row is the one place FR-9 ("no already-verified GMS cell may change
a byte") is knowingly violated, because the currently-verified bytes are wrong.
Handling: check the v95 `field.clientbound.FieldSetField` evidence fixture for
any of the sixteen ids; if present, re-derive and re-pin with the corrected read
order and record the reason in the evidence note. If absent, the fixture is
byte-identical and only the codec changes.

---

## 6. Testing strategy

Three layers, each proving something the others cannot.

**Layer 1 — predicate table tests (`libs/atlas-constants/job`).** Pure, fast,
and the place the per-version matrix from §2.1 is pinned cell by cell:

- `TestClientJobLevel` — beginner/root → 1; 2001 → 1; 430-434 → 2,2,3,3,4;
  Evan 2210-2218 → 2…10; non-Evan tier > 4 → 0. Comment records the GMS/JMS
  `(job-430)/2` vs `(job%10)/2` equivalence over 430-434.
- `TestNeedsMasterLevel_DualBladePerVersion` — the full §2.1 grid, six version
  columns × jobs 430-434 × {a generic skill, each of the four named ids}.
  Explicitly asserts job 432 true on v83/v84 (the fallthrough) and false on v87.
- `TestNeedsMasterLevel_EvanArmUnchanged` — the migrated assertions from
  `skill/model_test.go`, extended with a JMS and a v83 column proving the
  exception list is absent there.
- `TestNeedsMasterLevel_IgnoreCommonV95Only` — all sixteen ids false on v95,
  and the twelve reachable ones true on v83/v84/v87/v92/JMS.
- `TestUsesExtendedSP` — the §2.4 grid: `job/1000 == 3` true on v92/v95/JMS and
  false on v84/v87; 22xx and 2001 true on v84+; everything false on GMS ≤ 83.
  Includes a pinned case for the 3xxx arm even though no Atlas identity reaches
  it (FR-7).

**Layer 2 — byte-exact `CharacterData` fixtures
(`libs/atlas-packet/character`).** Built with the existing
`pt.CreateContext(region, major, minor)` helper and the `data_evan_test.go`
`mk(...)` pattern:

- `TestCharacterDataJMSDualBlade` — JMS 185, job 434, skills = {one 4th-job id
  that needs master level, one that does not}, asserting the exact byte length
  delta and the plain-SP short present (FR-10: a Dual Blade is *not* an
  extended-SP job, `430/1000 == 0`).
- `TestCharacterDataJMSEvanExtendedSP` — JMS 185, job 2218, asserting the
  1-byte count replaces the 2-byte short at the SP offset.
- `TestCharacterDataV95IgnoreCommon` — GMS v95, job 112 holding 1120012,
  asserting no trailing master-level int.
- `TestCharacterDataNoByteMovement_v84` / `_v95` — a creatable non-Evan,
  non-43x character encoded before and after, byte-identical golden. These are
  the FR-9 regression guards and must be written against the **current** encoder
  output captured before the change.

**Layer 3 — round-trip.** `Encode → Decode → Encode` for each new shape,
proving D4's mirror property, plus a decode-only fixture with a count-2 extended
SP block proving the reader consumes 1 + 2×2 bytes (D5).

**Gates.** `packet-audit matrix --check`, `fname-doc --check`, and
`operations --check` exit 0; flagless `tools/verify.sh` exits 0.

---

## 7. Files touched

| Path | Change |
|---|---|
| `libs/atlas-constants/job/master_level.go` | new — `ClientJobLevel`, `NeedsMasterLevel`, the three version-arm helpers, the sixteen-id ignore list |
| `libs/atlas-constants/job/master_level_test.go` | new — Layer 1 master-level tests, absorbing `skill/model_test.go`'s migrated cases |
| `libs/atlas-constants/job/extended_sp.go` | new — `UsesExtendedSP` |
| `libs/atlas-constants/job/extended_sp_test.go` | new — Layer 1 extended-SP tests, absorbing `data_evan_test.go:TestIsEvanJob`'s cases |
| `libs/atlas-constants/skill/model.go` | delete `NeedsMasterLevel` and its comment block (D2, D7) |
| `libs/atlas-constants/skill/model_test.go` | delete `TestNeedsMasterLevelMatchesClientRule` (moved) |
| `libs/atlas-packet/character/data.go` | delete `isEvanJob`; gate at `:316`/`:397` calls `job.UsesExtendedSP`; `:670`/`:696` call `job.NeedsMasterLevel`; add `job` import |
| `libs/atlas-packet/character/data_evan_test.go` | delete `TestIsEvanJob`; keep and extend the encode fixtures |
| `libs/atlas-packet/character/data_master_level_test.go` | new — Layer 2 + 3 fixtures |
| `services/atlas-channel/.../socket/writer/character_data.go` | comment text only (D6) |
| `docs/packets/evidence/gms_v95/field.clientbound.FieldSetField.yaml` | re-pin **only if** its fixture holds one of the sixteen ids (§5) |

No Kafka topic, REST route, migration, or UI change.

---

## 8. Risks

- **The v95 scope growth (§2.3) is the main risk.** It converts a
  "JMS-only, no GMS movement" task into one that intentionally changes verified
  GMS v95 bytes. The mitigation is that the change is provably in the client's
  direction and is fixture-pinned in both directions, but it means the v95
  evidence re-pin is a required step, not a contingency.
- **`packet-audit` may flag the v95 `SET_FIELD` cell as degraded** after the
  codec change even if the fixture bytes are unaffected, because the documented
  read order changes. Budget a re-pin.
- **The D1 departure from the PRD's "takes the tenant" line.** If overridden,
  the change is mechanical (swap two scalars for one struct) but adds a module
  edge; it should be decided before planning, not during.
- **Deleting `skill.NeedsMasterLevel` is a cross-package break.** The only
  callers are the two in `data.go` plus tests, confirmed by a repo-wide grep,
  so the blast radius is contained — but the plan should re-run that grep rather
  than trusting this line.

---

## 9. Out of scope / follow-ups

1. **`stat/clientbound/changed.go` extended-SP divergence (OQ1).** Real,
   pre-existing, and unreachable from this change. Needs the character's job
   threaded into the stat-change model. Separate task.
2. **Modelling extended-SP contents (D5).** Atlas writes a count of 0 because it
   has no per-master-level SP allocation model. When SP allocation lands, the
   encoder and the `CharacterStats` model both need the array.
3. **Jobs 3212 / 3312 in the v95 ignore list.** Modelled for fidelity; no Atlas
   identity binds them. If a future version bring-up introduces them, the
   membership already covers them.
4. **Dual Blade gameplay** — `job.FromIndex` still carries a
   `// jobId = job.BladeRecruit TODO` at `job/model.go:113`. Creating a Dual
   Blade is a separate task; this one only makes the wire correct if one exists.
