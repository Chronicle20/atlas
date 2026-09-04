# Task 27 review

Range: `3d4391dad..79f6bd566` (commit `79f6bd566`)
Brief: `.superpowers/sdd/plan/task-27-brief-impl.md`
Report: `.superpowers/sdd/plan/task-27-report.md`

## Scope

`git diff --stat 3d4391dad..79f6bd566` matches the four files the brief and the
task description named:

- `services/atlas-channel/atlas.com/channel/socket/handler/maker_skill.go` (+9/-4)
- `services/atlas-maker/atlas.com/maker/craft/resource_test.go` (+126/-4, net +130/-... see below)
- `services/atlas-maker/atlas.com/maker/craft/processor_test.go` (+39/-0)
- `docs/tasks/task-285-maker-skill-crafting/coverage-manifest.yaml` (+8/-2)

No out-of-scope files touched. Scope confirmed.

## 1. `makerResultFailedValue` comment grounding — FAIL (blocking)

New comment, `services/atlas-channel/atlas.com/channel/socket/handler/maker_skill.go:17-26`:

```
// makerResultFailedValue is the MAKER_RESULT nResult sent for every rejection
// and every atlas-maker-unreachable outcome. The value 2 is the wire-verified
// FAILED sentinel: libs/atlas-packet/character/clientbound/maker_result_test.go
// (NewMakerResultFailed(2)) asserts the encoded byte fixture `02 00 00 00`
// with length exactly 4 (nResult only, no mode) against IDA evidence that the
// client's nResult guard treats any value outside {0, 1} as the bodyless
// FAILED arm and reads nothing further (docs/tasks/task-285-maker-skill-crafting/plan.md:1455-1458).
```

The parenthetical `plan.md:1455-1458` is attached as the citation for the
claim "the client's nResult guard treats any value outside {0, 1} as the
bodyless FAILED arm and reads nothing further." I read that exact range:

```
$ nl -ba docs/tasks/task-285-maker-skill-crafting/plan.md | sed -n '1455,1458p'
  1455	`MakerResultFailed` — `nResult = 2`, nothing else:
  1456
  1457	```
  1458	02 00 00 00        nResult = 2
```

Lines 1455-1458 are only the byte fixture (`nResult = 2, nothing else` / the
hex dump). They say nothing about the guard's `{0, 1}` comparison, and
nothing about "reads nothing further." The nearest supporting sentence in
plan.md is three lines further down, **outside** the cited range:

```
1461: Assert the encoded length is exactly 4 bytes. The client stops reading
      at `nResult > 1`, so writing a mode here would desynchronise it.
```

— and even that line says `> 1`, not "outside {0, 1}" (the `{0, 1}` framing,
and the actual IDA addresses/`cmp` instructions, only exist in
`libs/atlas-packet/character/clientbound/maker_result_test.go`'s own
IDA-evidence comment above `TestMakerResultFailedByteOutput`, which the
sentence *does* cite correctly via the first, unnumbered reference).

The brief was explicit about this exact risk: "Read both cited locations
first and quote them accurately; if either does not say what this brief
claims, say so in the report rather than inventing a citation." The
implementer's report (`## 1`) claims plan.md:1455-1458 "is the fixture...
with the 'assert the encoded length is exactly 4 bytes' note" — but that
note is not inside 1455-1458 either; it's line 1461. The report's own
paraphrase silently widens the cited range to cover text that isn't in it,
and the shipped code comment inherits the same inaccurate line-anchored
citation.

This is not a request to re-litigate the value `2` (settled, unchanged) or
the overall claim (plan.md does support "stops reading at nResult > 1"
nearby) — it is that the comment cites a specific line range as evidence for
a sentence that range does not contain. A future reader who opens
`plan.md:1455-1458` on the strength of this comment will not find what the
comment says they will.

**Fix**: either move the citation to `plan.com:1461` (or `1455-1461`) so the
range actually contains the "stops reading" sentence, or drop the
line-anchor and cite the section more loosely.

## 2. `TestFailureLeavesStateUnchanged` — PASS

`services/atlas-maker/atlas.com/maker/craft/resource_test.go:581-718`. Before:
4 of 11 rejection rows had field-for-field assertions (per brief). Now: all
11 rows do.

Confirmed row-for-row parity against `TestEveryErrorCodeIsReturnedByItsOwnCondition`
(`resource_test.go:398-...`) by diffing the `name:`/`t.Run(...)` labels — both
tables carry exactly: `recipe_not_found`, `level_too_low`,
`skill_level_too_low`, `insufficient_materials`, `missing_prerequisite_item`,
`missing_prerequisite_quest`, `insufficient_mesos`, `inventory_full`,
`equip_not_found`, `no_crystal_mapping`, `craft_in_progress` — 11/11.

Every row (`resource_test.go:604-607`, `:712-715`) asserts
`h.character`/`h.etc`/`h.equip` equal before/after and the emitter call
count (`d.em.calls`) is exactly the expected count (0, except
`craft_in_progress`'s pre-seeded legitimate first call at `:696-702`, whose
snapshot is correctly taken *after* that first call — `:704-706` — so only
the second, rejected call is under test). This is a real strengthening, not
padding: each subtest independently drives its own harness/deps and its own
mutate/body pair, so a regression in any one rejection path's state-mutation
guard would fail its own row.

## 3. Non-zero worldId/channelId → saga payload — PASS

`services/atlas-maker/atlas.com/maker/craft/processor_test.go:763-799`,
`TestNonZeroWorldAndChannelReachEmittedSaga`: drives `craft.Request{WorldId:
world.Id(3), ChannelId: channel.Id(7), ...}` through `p.Create`, locates the
`AwardMesos` step in the emitted saga, and asserts
`mesos.WorldId == world.Id(3)` and `mesos.ChannelId == channel.Id(7)`
(`:797-799`) — both fields checked, both non-zero.

Traced the wiring in `processor.go` independently:
`grep -n "WorldId\|ChannelId" processor.go` shows `req.WorldId`/`req.ChannelId`
assigned straight into `saga.AwardMesosPayload{...}` at three call sites
(`:171-174` createOrUpgrade, `:262-265` crystal, `:373-376` disassemble). The
new test only exercises the `createOrUpgrade` (mode 1) path via
`eligibleRecipeFixture`, but that is sufficient to prove the field survives
end-to-end for at least one arm; if `WorldId`/`ChannelId` were dropped or
hard-coded to zero on that path the assertions at `:798-799` would fail
(`world.Id(3) != world.Id(0)`). Test is honest, not a tautology.

Minor non-blocking observation: only the `createOrUpgrade` (mode 1) arm is
covered by this new test; `crystal` (mode 3) and `disassemble` (mode 4) reuse
the same `req.WorldId`/`req.ChannelId` assignment pattern but are not
independently exercised for non-zero values. Not required by the brief
(which asked for "a test that drives a craft with non-zero world/channel"),
and the three call sites are structurally identical, so this is a coverage
note, not a defect.

## 4. Coverage-manifest `out_of_scope: model/asset` removal — VERIFIED, correct

Independently re-ran the two checks the brief specified:

```
$ git diff 9cd1ec5af..HEAD --stat -- libs/atlas-packet
 .../character/clientbound/maker_result.go          | 395 ++
 .../character/clientbound/maker_result_test.go     | 463 ++
 libs/atlas-packet/character/maker_result_body.go   |  73 ++
 .../character/maker_result_body_test.go            |  98 ++
 .../character/serverbound/maker_skill.go           | 173 ++
 .../character/serverbound/maker_skill_test.go      | 321 ++
```

No `model/asset` file appears anywhere in that delta — the claim that
`out_of_scope: model/asset` did not correspond to any branch touch is
correct.

```
$ grep -n "out_of_scope" -A3 -B3 docs/packets/PROCESS.md
153:out_of_scope:        # packets the diff may touch that are DELIBERATELY not this
154-  - model/asset      # task's coverage (incidental edit, shared-struct churn).
```

Confirms `model/asset` is the literal placeholder value in PROCESS.md's
schema-example block, exactly as the implementer's report and the manifest's
new comment claim. The removal is well-grounded, not a "silence the critic"
move — the replacement comment (`coverage-manifest.yaml:15-22`) explains the
reasoning and cites the same evidence, so a future reviewer can re-verify it
without re-deriving it. The remaining manifest content (`ops: MAKER_SKILL,
MAKER_RESULT`; 8 `versions`) matches the 6 touched files' two packet families
one-for-one.

## Build/lint spot-check (not a substitute for `tools/verify.sh`)

Per the brief's note not to run `tools/verify.sh` (already running elsewhere
in this session), did a narrow local build only on the touched packages:

```
$ cd services/atlas-maker/atlas.com/maker && go build ./craft/...
(clean, no output)
$ cd services/atlas-channel/atlas.com/channel && go build ./socket/handler/...
(clean, no output)
```

Both compile. Did not run `go test`/lint independently — the implementer's
report already quotes those outputs (`go build`/`go test` clean, `lint.sh`
`0 issues.` for both modules); no reason to distrust the quoted output, and
re-running the full test suites was not necessary to evaluate the four
specific points under review.

## Not evaluable

- Whether `lint.sh --go` genuinely reports 0 issues was not independently
  re-run (relied on the implementer's quoted output per the note above).
  Low risk: the two touched test files and the one handler file are small
  and the diff is additive/table-driven, matching existing lint-clean
  patterns in the same files.

## Summary

Three of the four points are correctly closed with real, falsifiable
evidence. Point 1 (comment grounding) ships a citation that, read literally,
points to a line range that does not contain the sentence it is attached to
— exactly the failure mode the brief warned the implementer to check for
and self-report rather than paper over. The report's own paraphrase of the
citation independently mis-describes what's on those lines, which suggests
the citation was written from memory/proximity rather than from a careful
re-read of the exact 1455-1458 slice.
