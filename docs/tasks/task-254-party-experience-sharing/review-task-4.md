# Review: Task 4 — `atlas-monster-death` `rates`/`map` processors, `rates` builder

Range reviewed: `8f24b5dc4..2f91e3cc0` (single commit `2f91e3cc0`)
Brief: `.superpowers/sdd/plan/task-4-brief.md` (Task 4 section)
Report: `.superpowers/sdd/plan/task-4-report.md`

## Scope

`git diff --stat` shows exactly 6 new files, 182 insertions, 0 deletions —
matches the brief's file list one-for-one:

- `rates/processor.go` (new)
- `rates/mock/processor.go` (new)
- `rates/builder.go` (new)
- `rates/builder_test.go` (new)
- `map/processor.go` (new)
- `map/mock/processor.go` (new)

No other files touched. `rates/provider.go` and `map/provider.go` are
untouched, as required (read-only per brief).

## Requirement-by-requirement

1. **`rates.Processor` / `rates.NewProcessor` / `GetForCharacter`** —
   `rates/processor.go:9-11,13-21,27-28` matches the brief's code block
   exactly: interface with `GetForCharacter(ch channel.Model, characterId
   uint32) Model`, `ProcessorImpl{l, ctx}`, `NewProcessor`, `var _ Processor
   = (*ProcessorImpl)(nil)`, delegating to the existing free function
   `GetForCharacter(p.l)(p.ctx)(ch, characterId)` — PASS.

2. **`rates.NewBuilder` / `Set*` / `Build()`** — `rates/builder.go:1-47`.
   Four unexported `float64` fields, `NewBuilder()` seeds all four to `1.0`
   (`builder.go:9-14`), four chained `Set*` methods returning `*Builder`
   (`builder.go:17-33`), `Build() Model` with no error (`builder.go:36-43`,
   matches brief's explicit "no error — nothing to validate" instruction) —
   PASS.

3. **`rates/mock` fallthrough to `rates.Default()`** —
   `rates/mock/processor.go:16-19`: nil-func fallthrough returns
   `rates.Default()`, not `rates.Model{}`. Confirmed `rates.Default()`
   (`rates/model.go:28-35`) sets all four rates to `1.0`, matching the
   brief's deliberate-deviation instruction (a zero `Model` would make
   `ExpRate() == 0`, silently awarding nothing) — PASS.

4. **`_map.Processor` / `_map.NewProcessor` / `CharacterIdsInField`** —
   `map/processor.go:1-28`. Package `_map` (matches `map/provider.go:1`),
   interface `CharacterIdsInField(f field.Model) ([]uint32, error)`,
   `ProcessorImpl{l, ctx}`, `NewProcessor`, `var _ Processor =
   (*ProcessorImpl)(nil)`, delegates to
   `CharacterIdsInFieldModelProvider(p.l)(p.ctx)(f)()` — matches
   `map/provider.go:19` signature exactly — PASS.

5. **`map/mock` fallthrough to `[]uint32{}, nil`** —
   `map/mock/processor.go:16-19` — PASS. This one correctly does NOT need
   the `Default()`-style deviation (empty slice + nil error is a legitimate,
   harmless zero-equivalent for "no characters in field", unlike a zero EXP
   rate).

6. **Builder test coverage (Step 1 of brief)** — `rates/builder_test.go`
   contains exactly the two tests specified:
   `TestNewBuilder_DefaultsToUnitRates` (asserts `1.0` on all four
   accessors, `builder_test.go:7-20`) and `TestBuilder_SetsEachRate`
   (asserts `2.5/3.0/4.0/5.0`, `builder_test.go:22-38`) — PASS.

## Repo conventions / global constraints

- **Immutable models** — `rates.Model` unchanged (already unexported fields,
  value receivers, pre-existing). `Builder` is a mutable helper type used
  only by tests, mirroring `monster/information/builder.go` precedent
  exactly (`information/builder.go:1-6` vs `rates/builder.go:1-8`) — no
  exported struct fields on the domain model itself. PASS.
- **Builders for test setup, no `*_testhelpers.go`** — `builder.go` is the
  builder; no test-only constructor files created. PASS.
- **Constants reuse** — `channel.Model` and `field.Model` imported from
  `github.com/Chronicle20/atlas/libs/atlas-constants/...`, not redeclared.
  PASS.
- **No stubs/TODOs** — `grep -n TODO` across all six new files: no matches.
  PASS.
- **No service-boundary reach-in** — both processors delegate to existing
  in-module free functions (`rates.GetForCharacter`,
  `_map.CharacterIdsInFieldModelProvider`); no cross-service imports
  introduced. PASS.
- **Determinism (FR-9.1)** — no maps, no iteration, over any collection in
  the new code; not applicable to this unit's logic. N/A, no violation.
- **Module-local verification only** — report ran
  `go build ./... && go test ./...` from
  `services/atlas-monster-death/atlas.com/monster`; confirmed independently
  below.

## Independent verification

```
$ cd services/atlas-monster-death/atlas.com/monster
$ go build ./...          # exit 0
$ go vet ./...            # exit 0, no output
$ go test ./rates/... ./map/... -v
--- PASS: TestNewBuilder_DefaultsToUnitRates (0.00s)
--- PASS: TestBuilder_SetsEachRate (0.00s)
ok  	atlas-monster-death/rates
--- PASS: TestCharacterIdsInFieldModelProviderDrainsBeyondOnePage (0.05s)
ok  	atlas-monster-death/map
$ go test ./...           # all ok, map/provider_drain_test.go stayed green
```

`map/provider_drain_test.go`'s `TestCharacterIdsInFieldModelProviderDrainsBeyondOnePage`
is untouched and still green — the brief's explicit non-regression check
holds.

## Test honesty

The builder tests are new-behavior tests (no prior `Builder` existed), so
"would it fail without the change" is trivially yes — `NewBuilder` is
undefined without `builder.go`. Report documents the RED step
(`undefined: NewBuilder`) consistent with this. No pre-existing test was
weakened or its assertions loosened to make this land.

## Not evaluable

None — this unit is small, self-contained, and fully covered by the diff
plus the two read-only files it delegates to (`rates/provider.go`,
`map/provider.go`), both of which were read to confirm signature
compatibility.

## Verdicts

- **Spec compliance**: full — every interface, file, and behavior specified
  in the Task 4 brief is present, matches signatures exactly, and the one
  deliberate deviation (`rates/mock` falling through to `Default()` instead
  of zero value) is implemented and correctly justified.
- **Task quality**: clean — mechanical, matches existing `party` processor
  precedent byte-for-byte in shape, builder matches `information.Builder`
  precedent with the one documented, brief-mandated difference (no error
  return). No TODOs, no stubs, build/vet/test all green.

No findings.
