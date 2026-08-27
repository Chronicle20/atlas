# Review — Task 16 continuation (resolve the user-ruled Maple Life content values)

**Unit under review:** commit `2ba40b634` — "docs(task-246): resolve the
user-ruled Maple Life content values". Docs-only, one file:
`docs/tasks/task-246-maple-life-character-creation/maple-life-content.md`
(254 insertions, 53 deletions).

**Brief:** `.superpowers/sdd/plan/task-16-brief-cont.md`
**Report:** `.superpowers/sdd/plan/task-16-report-cont.md`

## Scope note (non-blocking, but worth recording)

The commit range given in the task, `677b62c51..2ba40b634`, actually contains
**two** commits, not one:

```
git log --oneline 677b62c51..2ba40b634
2ba40b634 docs(task-246): resolve the user-ruled Maple Life content values
2f7faecf7 feat(saga): carry unspent ap and sp on character creation
```

`2f7faecf7` is an unrelated Go feature commit (saga-orchestrator/atlas-saga,
349 lines across 11 files) that landed on this branch between the base and
the docs commit; it is not part of this unit and was not reviewed. Isolating
the actual unit (`git diff --stat 2f7faecf7..2ba40b634`) confirms it matches
the report exactly: one file, 254/53. The docs commit itself is exactly what
the brief and report describe — the "one commit, docs-only" framing just
undercounts the range by one unrelated commit. Not a defect in the reviewed
work; flagged so the discrepancy isn't silently absorbed.

## 1. §5.3(b) combining rule — the highest-stakes claim

Read `ProcessLevelChange` and `resolveHPMPGainParams` directly
(`services/atlas-character/atlas.com/character/character/processor.go`).

- `hpMPParams := p.resolveHPMPGainParams(c)` is called **once**, at
  `processor.go:1600`, before the level-up loop begins. Confirmed — not
  recomputed inside `for i := range amount` (`processor.go:1602-1624`).
- `rollHPMPGain(hpMPParams)` is called once per iteration inside that same
  loop (`processor.go:1621`), reusing the identical, already-resolved
  `hpMPParams` for every level-up in the batch. Confirmed.
- `resolveHPMPGainParams` sets `params.hpBonus = se.X()` at
  `processor.go:1810` (Warrior) and `params.mpBonus = se.X()` at
  `processor.go:1817` (Magician) — both cited lines match exactly.
- Skill.wz cross-check: `Skill.wz/100.img.xml` node `1000001`
  (`level/1..10`) has `x = 4,8,12,...,40` / `y = 3,6,9,...,30` — exact match
  to the document's table and its `x = 4L, y = 3L` formula.
  `Skill.wz/200.img.xml` node `2000001` has `x = 2,4,...,20` / `y =
  1,2,...,10` — exact match to `x = 2L, y = L`.
- Arithmetic: Warrior HP midpoint `50 + 29×26 = 804` (interval `24..28`,
  avg 26) — correct. Magician MP midpoint `5 + 29×23 = 672` (interval
  `22..24`, avg 23) — correct. `29×4 = 116`, `29×2 = 58` — the two closed
  forms `804 + 116×nSP` and `672 + 58×nSP` are arithmetically sound given the
  stated combining rule.
- `GetSkillLevel` (`services/atlas-character/atlas.com/character/character/model.go:202-209`)
  returns `0` for an unlearned skill — confirmed, so the document's disclosed
  inference ("unlearned skill sits at level 0 with no data node, contributes
  0 — convention, not a read WZ node") is consistent with both the WZ data
  (no `level/0` node in either skill) and the live-code fallback path
  (`GetEffect` errors on a nonexistent level, leaving `hpBonus`/`mpBonus` at
  their zero value).

**Verdict: PASS.** The control-flow claim, the field choice, the WZ table,
and the arithmetic are all independently verified accurate. This is the
single highest-stakes claim in the document and it holds up under direct
inspection of the cited code and data, not just the document's prose.

One citation-precision nit: §5.3(b) cites `resolveHPMPGainParams` as
`processor.go:1721-1822`; the function actually spans `1721-1821` (off by
one line — the closing brace is at 1821, not 1822). Non-blocking.

## 2. The `x` vs `y` asymmetry claim

Read `getMaxHpGrowth` and `getMaxMpGrowth` directly.

- `getMaxHpGrowth`: `resMax = uint16(int16(resMax) + se.Y())` at
  `processor.go:1243` — confirmed, `se.Y()`.
- `getMaxMpGrowth`: `resMax = uint16(int16(resMax) + se.X())` at
  `processor.go:1319` — confirmed, `se.X()`.

The asymmetry is real and both cited line numbers are exact. The document
correctly frames this as quoted-as-found, not fixed, and correctly notes it
does not affect the level-up-path conclusion (both level-up branches read
`se.X()`, confirmed above at `:1810`/`:1817`). Per the task's explicit
out-of-scope list, the asymmetry-as-a-bug is not raised here.

**Verdict: PASS.**

## 3. The Skill.wz table

Both skill nodes read directly from `<local-wz-root>/Skill.wz/100.img.xml`
(node `1000001`) and `<local-wz-root>/Skill.wz/200.img.xml` (node
`2000001`); all 20 `x`/`y` values across both tables match the document's
transcription exactly (verified level-by-level, not spot-checked). Skill ids
`1000001`/`2000001` also confirmed to be
`WarriorImprovedMaxHpIncreaseId`/`MagicianImprovedMaxMpIncreaseId` in
`libs/atlas-constants/skill/constants.go:2933`/`:3023`, matching what §3
already cites for `spSkillId`.

The "unlearned skill sits at level 0, no WZ node" claim is disclosed
explicitly as convention/inference in the document's own prose
("this is not read from an explicit WZ node, it is the standard convention
every other skill in this tree also follows") rather than presented as a
directly-read fact — this is the correct disclosure discipline. It is not
quietly relied on elsewhere: the only place it matters (the `nSP=0` reduction
in the combining-rule formula) is stated explicitly.

**Verdict: PASS.**

## 4. The five town-map ids (§5.1)

All five `libs/atlas-constants/map/constants.go` line citations verified
exact by direct line lookup: `VictoriaRoadHenesysId:42`,
`VictoriaRoadElliniaId:99`, `VictoriaRoadPerionId:158`,
`VictoriaRoadKerningCityId:183`, `VictoriaRoadNautilusHarborId:521` — all
match.

All five `mapDesc` quotations cross-checked against
`<local-wz-root>/String.wz/Map.img.xml` verbatim (modulo XML entity decoding
of `&apos;`) — exact matches for Henesys, Ellinia, Perion, Kerning City, and
Nautilus Harbor.

Nautilus disambiguation: the document records the candidate interior rooms
it rejected and the specific reasoning for the harbor node
(`120000000`) over the Navigation Room (`120000101`, the NPC's own room and
one of the two candidates the original job-advancement-NPC sweep found).
Checked `120000101`'s `mapDesc` directly — "A Navigation Room where the
captain of The Nautilus, Kyrin, resides." — genuinely unrelated flavor text
with no creation-town phrasing, exactly as the document describes. This is a
real citation with real disambiguation reasoning, not the user's ruling
standing in for one.

**Verdict: PASS.**

## 5. §4 shrinking, both directions

Read the current §4 in full. It now lists exactly three items — class
ordinal order (2/3/4), top/bottom/shoes synthesis, gender — plus a "closed
this pass" pointer to §5 for the three ruled items. No stale UNCONFIRMED
survives for `hp`/`mp`, `mapId`, or Pirate's weapon; §3's own subsections for
those three were edited in place to point at §5 rather than carry the old
UNCONFIRMED text (verified in the diff).

One residual gap, non-blocking: §5.3(a) explicitly defers a real decision —
Pirate's `mp` midpoint is `599.5` (interval `18..23` has an odd span), and
the document states in prose that "Task 20 must round `599.5` one direction"
and that this is "a policy choice... not a fact this pass can resolve." That
is precisely the shape of thing the brief says should appear in §4 ("plus
anything Step 1 or Step 3 leaves UNCONFIRMED"), yet it isn't listed there —
it exists only as inline prose inside §5.3(a). It is not silently dropped
(the document says outright that the choice is unresolved and names the
consumer), so this is not the dangerous failure mode the brief warns about;
it's a cross-referencing gap, not a content gap. Given the document itself
calls the magnitude "not large enough (±0.5 MP) to warrant blocking on it,"
this is non-blocking, but a future pass should either add it as a fourth §4
item or fold the rounding call into Task 20's brief directly so it isn't
lost between two sections.

No item was found closed-but-still-marked-open, and no item was found
silently dropped whose underlying uncertainty was not actually resolved by
one of the three rulings.

**Verdict: PASS, with the one non-blocking cross-reference gap above.**

## 6. The citation fix

`libs/atlas-constants/job/constants.go` read directly, line-counted from
`const (` at line 94:

- `WarriorId:96`, `MagicianId:106`, `BowmanId:116`, `RogueId:123`,
  `PirateId:130` — all five confirmed exact by direct grep/line-count.

`RogueId` is now correctly cited at `:123` (was `:127`), matching the
review's Finding 2, and the four neighbouring job-id citations were verified
independently rather than trusted from the report's prose — all accurate.

**Verdict: PASS.**

## Other checks

- §5.2 (Pirate equipment): `1482000`/`1492000` both confirmed present as
  `award_item` operations in
  `deploy/seed/gms/95_1/npc-conversations/npc/npc-1090000.json` (lines
  688/694/745/752). The `mapleLife` schema's `Equipment []EquipmentEntry`
  list field confirmed to exist verbatim in
  `services/atlas-configurations/atlas.com/configurations/tenants/characters/preset/rest.go:43`.
- No Go file changed by the reviewed commit; `tools/verify.sh` correctly not
  run, consistent with the brief's instruction.

## Summary

Every claim checked against its cited primary source — repo constants, WZ
XML nodes, and live `processor.go` control flow — held up exactly as
described, including the highest-stakes claim (the once-resolved,
loop-invariant `hpMPParams` in `ProcessLevelChange`) and the asymmetry claim
this pass explicitly declined to "fix." Citations are precise to the line in
all cases checked except one, off by a single line. The one process note
(§4/§5.3a cross-reference gap for Pirate's half-integer MP midpoint) does not
rise to a blocking defect: it is disclosed, scoped, and small in magnitude.

No invented value was found anywhere in the reviewed material.
