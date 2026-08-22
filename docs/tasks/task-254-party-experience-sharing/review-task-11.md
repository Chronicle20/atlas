# Review: Task 11 — inject collaborators into `monster.ProcessorImpl`

**Range reviewed:** `c386b7a06..b99352738` (single commit `b99352738`)
**Brief:** `.superpowers/sdd/plan/task-11-brief.md`
**Report:** `.superpowers/sdd/plan/task-11-report.md`

## Scope confirmed

The commit touches exactly the two files the brief's Files section names:

- `services/atlas-monster-death/atlas.com/monster/monster/processor.go` (+87/-11)
- `services/atlas-monster-death/atlas.com/monster/monster/processor_di_test.go` (new, 126 lines)

No other file is touched. Matches the brief's stated scope exactly.

## Findings

### 1. `NewProcessor` production defaults — PASS

`processor.go` binds all eight fields (`cp`, `pp`, `rp`, `ip`, `fp`, `smp`,
`ht`, `cfg`) in `NewProcessor`, matching the brief's snippet verbatim
(field order, names, constructor calls). Confirmed via `git show
b99352738 -- '*/monster/processor.go'`.

### 2. `ProcessorOption` / `With*` surface — PASS, matches brief's Interfaces section

All eight requested options are present and each mutates exactly one field:
`WithCharacterProcessor`, `WithPartyProcessor`, `WithRatesProcessor`,
`WithInformationProcessor`, `WithFieldProcessor`, `WithSystemMessageProcessor`,
`WithHintThrottle`, `WithExperienceConfig`. `(*ProcessorImpl).With` is a
shallow-copy clone (`clone := *p; cp := &clone`) that applies options to the
clone and returns it — verbatim from the brief, and correctly avoids binding
any method value (all fields are interfaces or a `*Throttle` pointer), per
the `atlas-pets` hazard note. No drift from the brief's Interfaces section —
this is the surface Task 12 will consume.

Test `TestWith_ReturnsCloneAndDoesNotMutateOriginal`
(`processor_di_test.go:26-42`) proves the clone is a distinct pointer and the
original is untouched. `TestWith_AppliesEveryOption`
(`processor_di_test.go:44-88`) exercises all eight options together.
`TestNewProcessor_BindsProductionDefaults` (`processor_di_test.go:90-126`)
asserts every collaborator field is non-nil and `cfg == LoadExperienceConfig()`
on a bare `NewProcessor`. All three ran and passed:

```
$ go test ./monster/ -run 'TestWith_|TestNewProcessor_' -v
--- PASS: TestWith_ReturnsCloneAndDoesNotMutateOriginal
--- PASS: TestWith_AppliesEveryOption
--- PASS: TestNewProcessor_BindsProductionDefaults
PASS
```

### 3. `CollectToMap` → plain loop substitution — PASS, genuinely equivalent

Verified the claim in the implementer's report and self-review against the
actual `_map` package:

- `services/atlas-monster-death/atlas.com/monster/map/processor.go:12` —
  interface method `CharacterIdsInField(f field.Model) ([]uint32, error)`.
- `services/atlas-monster-death/atlas.com/monster/map/processor.go:26-28` —
  implementation is `return CharacterIdsInFieldModelProvider(p.l)(p.ctx)(f)()`,
  i.e. it already drains the old provider and returns the slice/error pair
  directly. The claim that this method returns `([]uint32, error)` rather than
  a `model.Provider` is correct.

Compared the old and new logic side by side:

- Old: `model.CollectToMap[uint32, uint32, bool](_map.CharacterIdsInFieldModelProvider(...), identity, constTrue)()`.
  `CollectToMap` (`libs/atlas-model/model/processor.go:590-601`) calls the
  provider, returns the error immediately on failure (map is `nil` in that
  case), otherwise builds `result := make(map[K]V)` and populates it from the
  slice.
- New (`processor.go`, `produceDistribution`): `fieldCharacterIds, err :=
  p.fp.CharacterIdsInField(f)` with an immediate `if err != nil { return
  model.ErrorProvider[...](err) }`, then `cim := make(map[uint32]bool)` and a
  `for _, m := range fieldCharacterIds { cim[m] = true }` loop.

This is a faithful transliteration: same error-short-circuit-before-map-build
ordering, same `map[uint32]bool` shape and `true` value for every ID, same
empty/nil-slice behavior (`range` over a nil or empty slice is a no-op,
yielding an empty map either way, identical to `CollectToMap`'s behavior on an
empty `ms`). No behavioral drift.

### 4. Bodies routed through injected fields — PASS

`CreateDrops`: `rates.GetForCharacter(...)` → `p.rp.GetForCharacter(...)`;
`party.NewProcessor(p.l, p.ctx).GetByMemberId(...)` → `p.pp.GetByMemberId(...)`.
`DistributeExperience`/`produceDistribution`: `character.NewProcessor(...).GetById`
→ `p.cp.GetById`, `rates.GetForCharacter` → `p.rp.GetForCharacter`,
`information.GetById` → `p.ip.GetById`, field-characters lookup routed as
above. `distributeCharacterExperience`: `character.NewProcessor(...).AwardExperience`
→ `p.cp.AwardExperience`. All match the brief's Step 3 instructions
one-for-one. `drop.NewProcessor` and `quest.GetStartedQuestIds` are correctly
left untouched — out of scope per the brief.

The two pre-existing `// TODO parties` / `// TODO account for healing`
comments in `produceDistribution` are untouched, as expected (Task 12 removes
them, not this task).

### 5. Behaviour preservation — PASS

`go build ./... && go test ./monster/...` from the module root
(`services/atlas-monster-death/atlas.com/monster`) passes clean:

```
ok  	atlas-monster-death/monster	0.079s
ok  	atlas-monster-death/monster/drop	(cached)
ok  	atlas-monster-death/monster/information	(cached)
```

`gofmt -l` and `go vet ./monster/...` report nothing. No pre-existing test in
the `monster` package changed behavior or assertions — confirms the "pure
plumbing, no behaviour change" framing of the task.

### 6. Config-loaded-inside-`NewProcessor` — ruled, not a finding

Per the task instructions, `LoadExperienceConfig()` is called inside
`NewProcessor` rather than threaded from `main.go`; `main.go` is untouched.
This matches context.md §3.1 and is explicitly ruled out of scope for this
review.

## Not evaluable

None. The full diff surface (two files, both new/modified by this commit)
was read and verified against the brief, and the module builds and tests
cleanly.

## Verdict

APPROVED. No blocking or non-blocking findings.
