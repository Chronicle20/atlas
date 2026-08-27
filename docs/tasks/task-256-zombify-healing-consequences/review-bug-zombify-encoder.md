# review: bug-zombify-no-visible-effect — encoder fix (`2e91e814d`)

Reviewer: task-reviewer (sonnet)
Range: `b9904f7f6..2e91e814d` (single commit `2e91e814d`)
Requirement: `docs/tasks/task-256-zombify-healing-consequences/bug-zombify-no-visible-effect.md` (`## Root cause`, `## Fix`, `### Settled`)
Implementer report: `docs/tasks/task-256-zombify-healing-consequences/report-bug-zombify-encoder.md` (treated as unverified claims, checked below)

## Scope

`git diff --stat b9904f7f6..2e91e814d`:

```
libs/atlas-packet/model/character_temporary_stat.go          | 28 +++++++++---
libs/atlas-packet/model/character_temporary_stat_test.go     | 51 ++++++++++++++++++++--
services/atlas-messages/.../disease/commands.go               | 20 ++++++++-
services/atlas-messages/.../disease/commands_test.go          | 44 +++++++++++++++++++
4 files changed, 133 insertions(+), 10 deletions(-)
```

Matches the report's file list exactly. Reviewed: the full diff of all four
files, plus the contracts the diff's correctness depends on — signatures of
`NewCharacterTemporaryStatBaseWithOptions`, `AddStat`, `ApplyCommandProvider`
(both atlas-messages and atlas-channel copies), `buff.NewBuff`/`Model`, and the
consumer call site `character_buff_give.go` that wires atlas-buffs output into
the CTS the encoder reads. All pre-existing and unmodified by this commit —
read only to verify the seam, not surveyed further.

## Findings

### 1. Requirement fidelity — PASS

The `## Fix` table's first row is implemented essentially verbatim:
`libs/atlas-packet/model/character_temporary_stat.go:1424-1428` adds an
`if bs.name == character.TemporaryStatTypeUndead` branch in the
`twoStateDynamic` default case, using the same
`NewCharacterTemporaryStatBaseWithOptions(true, s.Value(), composite, narrow)`
shape as the existing `EnergyCharge` case, with
`composite := s.SourceId() | (int32(s.Level()) << 16)`.

- `DashSpeed`/`DashJump` are left falling through to the zeroed
  `NewCharacterTemporaryStatBase(true, narrow)` default (`:1429`) — per the
  brief's explicit instruction to leave them alone. Confirmed at
  `character_temporary_stat.go:1420-1422` (comment) and `:1429` (code).
- `UNDEAD` stays in `baseStatNames` (`:1132-1136`), unmoved — confirmed by
  `git diff`, that map is untouched by this commit.
- The GM command row (`services/atlas-messages/.../disease/commands.go`) is
  implemented: new `diseaseMobSkill` helper supplies
  `sourceId = monsterconstant.SkillTypeUndead` (133) for UNDEAD only, `level = 1`
  unconditionally (unchanged from the pre-existing literal — see finding 5).
- Fixture re-pinning (`## Fix` row 4) is **not done** in this commit — see
  finding 6 (non-blocking, disclosed).

### 2. Field mapping — PASS, matches `### Settled` exactly

`NewCharacterTemporaryStatBaseWithOptions(bDynamicTermSet bool, nOption int32, rOption int32, narrowTimeField bool)`
(`character_temporary_stat.go:432`) assigns positionally: `s.Value()` → `nOption`,
`composite` → `rOption`. The struct's `Encode` writes
`w.WriteInt32(m.nOption)` then `w.WriteInt32(m.rOption)` (`:471-472`) — nOption
first, rOption second, matching the brief's `### Settled` IDA citation of
`TemporaryStatBase<long>::DecodeForClient` (`nOption` at `this+3`, `rOption` at
`this+4`, i.e. second int32). The composite formula
`sourceId | (level << 16)` matches `MobSkillReasonForeignValueWriter`'s
documented convention cited in the brief. No off-by-one; nOption correctly
carries the disease amount (`s.Value()`), rOption correctly carries the mob-skill
composite, not swapped.

New test `TestCTSUndeadPopulatedBlock` (`character_temporary_stat_test.go:1043-1055`)
pins `AddStat(nil)(tn)("UNDEAD", 133, 1, 5, ...)` → head bytes
`01 00 00 00 85 00 05 00` = `nOption=1` (LE) then `rOption=327813=0x00050085`
(LE) = `133 | 5<<16`. Arithmetic checked: `5*65536+133 = 327813`. Correct, and
this assertion would fail against the pre-fix code (which falls to the zeroed
default for UNDEAD) — a genuine regression pin, not a test that passes either
way.

### 3. Load-bearing claim — "no pre-existing fixture needed re-pinning" — PASS, verified by enumeration

Enumerated every UNDEAD-touching test in the repo before this commit
(`grep -rn Undead libs/atlas-packet`):

- `character_temporary_stat_test.go:80` — comment only, part of an unrelated
  mask test (`SLOW` only, group referenced generically).
- `character_temporary_stat_test.go:360` — comment only (v61 6-member group,
  no Undead).
- `buff_give_test.go` — `TestBuffGiveEmptyRoundTrip`, `TestBuffGiveJMSMask`,
  `TestBuffGiveForeignJMSMask`, `TestBuffGiveV79Mask`,
  `TestBuffGiveForeignV79Mask`, `TestBuffGiveV72Mask` and siblings all encode
  `model.NewCharacterTemporaryStat()` — an **empty** CTS with no stats added at
  all, let alone UNDEAD with a nonzero composite. `TestBuffGiveDiseaseTrailer`
  and `TestBuffGiveBuffTrailer` set `Slow`/`Invincible`, never `Undead`.

No pre-existing test in the repo sets UNDEAD with a nonzero `sourceId`/`level`.
The claim holds: every UNDEAD-touching fixture pre-commit either doesn't touch
UNDEAD's byte content at all (empty CTS) or is a comment. `go test ./...` in
`libs/atlas-packet` is green post-commit (verified directly, not trusted from
the report).

### 4. Cross-service seam — PASS, traced by hand

atlas-messages → atlas-buffs → atlas-channel path, traced through pre-existing
(unmodified) code:

- `commands.go:134`: `sourceId, level := diseaseMobSkill(statName)`, then
  `buff.ApplyCommandProvider(f, id, 0, sourceId, level, duration, changes)`.
  Signature `(field, characterId, fromId, sourceId, level, duration, changes)`
  confirmed at `services/atlas-messages/.../kafka/message/buff/kafka.go:44` —
  positional mapping correct (`fromId=0`, `sourceId`/`level` in their own slots,
  not swapped).
- atlas-channel's buff consumer (`kafka/consumer/buff/consumer.go`) builds
  `buff.NewBuff(e.Body.SourceId, e.Body.Level, ...)` (e.g. `:172`) — carries the
  APPLY message's SourceId/Level through unchanged. This plumbing predates this
  commit (per the bug doc, fixed in `49465ebac`) and is not touched by the diff.
- `character_buff_give.go:22,37`: `cts.AddStat(l)(t)(c.Type(), b.SourceId(), c.Amount(), b.Level(), b.ExpiresAt())`.
  `AddStat` signature `(n, sourceId, amount, level, expiresAt)`
  (`character_temporary_stat.go:638`) — positional mapping correct.
- Both `EncodeForeign` (`:1121`) and the self encode (`:939`) call the same
  `getBaseTemporaryStats`, so the fix covers `CharacterBuffGive` and
  `CharacterBuffGiveForeign` from a single code path — matches the brief's
  claim that `CUser` (base of `CUserLocal`/`CUserRemote`) covers both the
  diseased character and observers.

`sourceId`/`level` survive the whole path as distinct int32/byte fields, not
merged or dropped, and reach the encoder as the values the commit sets.
`TestDiseaseMobSkillSuppliesUndeadSourceId` and
`TestDiseaseMobSkillLeavesOtherDiseasesUnestablished`
(`commands_test.go:71-113`) assert the NEW contract (UNDEAD gets
`SkillTypeUndead`=133, others stay 0), not just the old shape.

### 5. `level=1` for all diseases — PASS, genuinely unchanged for non-UNDEAD

Pre-commit: `buff.ApplyCommandProvider(f, id, 0, 0, 1, duration, changes)` —
literal `sourceId=0, level=1`. Post-commit: `diseaseMobSkill` sets
`level = 1` unconditionally for every disease name, and only sets `sourceId`
nonzero when `statName == TemporaryStatTypeUndead` (Go zero-value `0` for every
other case). `TestDiseaseMobSkillLeavesOtherDiseasesUnestablished` iterates
10 other disease types and asserts `sourceId == 0` for each. `level` is not
separately asserted per-disease in the new tests, but the function body makes
it structurally impossible for `level` to vary by disease (single
unconditional assignment before the branch) — non-UNDEAD behaviour is
unchanged both empirically (test) and by code inspection.

One open question, not a defect in this commit: whether `level=1` is a value
the client's MobSkill table actually resolves for skill id 133 (i.e. whether
the GM-command path's *animation* renders, versus just carrying a nonzero
composite). The brief's own `## Not yet answered` section already defers this
to a live re-test rather than more static analysis, and the implementer's
comment (`commands.go:47-51`) discloses this is unestablished. Not a
regression — `level=1` is the pre-existing literal, not an invented value.

### 6. Deferred packet-audit re-pinning — non-blocking, but a real gap

The brief's `## Fix` table explicitly says: *"This change crosses a wire
contract with verified coverage cells, so it is a `packet-verifier` fan-out,
not a plain implementer edit,"* and its row 4 calls for re-pinning
`docs/packets/audits/` evidence records. This commit does not touch
`docs/packets/audits/` at all (confirmed: `git diff --stat` shows only the 4
files above), and the new UNDEAD-populated-block test
(`TestCTSUndeadPopulatedBlock`) carries no `packet-audit:verify` annotation —
it lives outside the audited coverage matrix that `buff_give_test.go` belongs
to. `grep` across `docs/packets/audits/` for "undead"/"zombify"/"zeroed" found
no stale text, so no existing audit record is now factually wrong — but the
matrix has a **coverage gap**: `BuffGive`/`BuffGiveForeign` for the versions
where UNDEAD is a two-state member (gms_v72/79/83/84/87/92/95, jms_v185) have
no audited fixture exercising a populated UNDEAD block, and `gms_v48`'s
"legacy path never reaches this branch" claim is asserted by inference, not
verified. The implementer's report discloses this explicitly and recommends a
follow-up `packet-verifier` pass — so this is a disclosed, tracked gap rather
than a silent one, but it means the brief's own dispatch-shape instruction
("packet-verifier fan-out, not a plain implementer edit") was not followed for
this commit, and the audit matrix does not yet reflect the new wire contract.

## Not evaluable

- **The icon half of the fix (client-visible outcome).** The brief's own
  `## Not yet answered` section defers this to a live re-test; nothing in this
  diff or its test surface can confirm or refute it from static review. Out of
  this review's reach by the brief's own admission.
- **Whether `level=1` actually resolves to a valid MobSkill(133) table entry on
  the v83 client** — requires client-side data/behavior not available to this
  review.

## Verdict rationale

All four priority checks in the dispatch brief (load-bearing fixture claim,
field-mapping correctness, DashSpeed/DashJump + baseStatNames unchanged,
cross-service seam with a test asserting the new contract) pass with cited
evidence. The one real gap — the deferred `packet-verifier` fan-out over
`docs/packets/audits/` — is disclosed by the implementer's own report, does not
corrupt any existing verified fixture (confirmed by enumeration), and is
tracked as follow-up work rather than silently dropped. It is a genuine
finding worth surfacing but does not block this commit's correctness.

---

verdict: APPROVED_WITH_FINDINGS
artifact: docs/tasks/task-256-zombify-healing-consequences/review-bug-zombify-encoder.md
scope_confirmed: reviewed the full diff of b9904f7f6..2e91e814d (4 files: libs/atlas-packet/model/character_temporary_stat.go + _test.go, services/atlas-messages disease/commands.go + commands_test.go), plus signatures/call sites of NewCharacterTemporaryStatBaseWithOptions, AddStat, ApplyCommandProvider (atlas-messages + atlas-channel), buff.NewBuff/Model, and character_buff_give.go — the seam this diff's correctness depends on. Matches the range given; no scope mismatch.
blocking: 0
non_blocking: 1
  - docs/packets/audits/ (not touched by this commit) — brief's own Fix table calls this a "packet-verifier fan-out, not a plain implementer edit"; BuffGive/BuffGiveForeign coverage cells for gms_v72/79/83/84/87/92/95 and jms_v185 have no audited fixture exercising a populated UNDEAD block, and the gms_v48 "legacy path unreached" claim is asserted by inference only. Disclosed in report-bug-zombify-encoder.md but not actioned in this commit.
not_evaluable: 2
