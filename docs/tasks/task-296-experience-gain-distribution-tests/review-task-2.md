# Review: Task 2 — Extract `buildIncreaseExperienceConfig`

- Commit range: `dc9c7707e..fffe4fdde`
- Commit under review: `fffe4fdde` — refactor(atlas-channel): extract buildIncreaseExperienceConfig from announceExperienceGain
- Brief: `.superpowers/sdd/plan/task-2-brief.md`
- Report: `.superpowers/sdd/plan/task-2-report.md`

## Scope

`git log --oneline dc9c7707e..fffe4fdde` shows a single commit. `git diff --stat`
shows exactly one file touched:

```
.../channel/kafka/consumer/character/consumer.go | 85 ++++++++++++----------
1 file changed, 47 insertions(+), 38 deletions(-)
```

Matches the brief's constraint that only `consumer.go` may be touched. Confirmed
`kafka/message/character/kafka.go` (Task 1's file) and all test files show no
diff in the range (`git diff --stat` / `--name-only ... | grep test` both
empty).

## Findings

### PASS — Pure mechanical extraction, chain byte-identical, order preserved

Compared the parent commit's inline chain (`dc9c7707e:.../consumer.go`,
originally at lines 369-404 inside `announceExperienceGain`'s innermost
closure) against the new `buildIncreaseExperienceConfig` function body
(`fffe4fdde:.../consumer.go:367-406`) via direct diff of the extracted
blocks. The two are line-for-line identical:

- Same 14-arm `else if` chain, same order (White, Yellow, Chat, MonsterBook,
  MonsterEvent, PlayTime, Wedding, SpiritWeek, Party, Item, InternetCafe,
  RainbowWeek, PartyRing, CakePie).
- Same `character2.ExperienceDistributionType*` operands, same field
  assignments, same `int32(...)`/`byte(...)` conversions.
- WHITE/YELLOW last-wins overwrite behavior unchanged: both arms
  unconditionally set `c.White` and `c.Amount` on every match, with no
  guard against re-overwriting on a later matching element — identical to
  the pre-extraction code. No conversion to `switch`, no dedup, no
  early-return.
- Only textual difference is the loop variable, renamed from
  `distributions` to `ds` to match the new function's parameter name — an
  expected, brief-sanctioned rename (the brief specifies the signature
  `func buildIncreaseExperienceConfig(ds []character2.ExperienceDistributions) ...`).

Evidence: `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:367-406`
(new function) vs. the removed block shown in `git diff dc9c7707e..fffe4fdde`
covering the old `@@ -359,49 +359,58 @@` hunk — the deleted 14-arm chain and
added 14-arm chain match one-for-one when diffed directly (verified with a
targeted `diff` of the two chain bodies, zero delta beyond the loop variable
rename and indentation from de-nesting out of the four-level closure).

### PASS — Function is pure

`buildIncreaseExperienceConfig` (consumer.go:367-406) takes only
`ds []character2.ExperienceDistributions`, returns
`model2.IncreaseExperienceConfig`, and references no logger, no `context`,
no `session`, and no package-level state. Confirmed by direct read of the
function body.

### PASS — Call site reduction matches the brief; packet splat and error handling unchanged

`announceExperienceGain`'s innermost closure (consumer.go, starting at the
line after `buildIncreaseExperienceConfig`) now reads
`c := buildIncreaseExperienceConfig(distributions)` and is otherwise
byte-identical to the parent commit's closure body: the four-level currying,
`session.Announce(l)(ctx)(wp)(charcb.CharacterStatusMessageWriter)(...)`
call, the 17-argument positional splat, the error log string
(`"Unable to announce experience gain to character [%d]."`), and both
returns are unchanged. Verified with a direct diff of the full
`announceExperienceGain` function body between the parent and head commits —
the only delta is the single collapsed line.

### PASS — No import changes; imports otherwise untouched

`diff` of the first 25 lines (import block) between parent and head
`consumer.go` is empty. `character2` and `model2` aliases were already
present and are the only ones the new function uses.

### PASS — Build, tests, gofmt

Ran from `services/atlas-channel/atlas.com/channel`:

```
$ go build ./...
(no output — success)
$ go test ./kafka/consumer/character/...
ok  	atlas-channel/kafka/consumer/character	0.014s
$ gofmt -l kafka/consumer/character/consumer.go
(no output — clean)
```

### PASS — Task 1 file and test files untouched

`git diff --stat dc9c7707e..fffe4fdde -- .../kafka/message/character/kafka.go`
is empty. `git diff --name-only dc9c7707e..fffe4fdde | grep -i test` is empty.

## Not evaluable

None. The full review surface (the single touched file, its diff against the
parent commit, and its imports/call site) was directly inspected and
verified against the pre-extraction source.

## Verdict rationale

This is exactly the pure mechanical extraction the brief specified: the
14-arm chain moved verbatim (no reordering, no switch conversion, no
overwrite-semantics change), the new function is pure, the call site and
packet splat are unchanged, no unrelated files were touched, and build/test/
gofmt all pass.
