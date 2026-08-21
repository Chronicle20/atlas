# Review: Task 9 — `atlas-monster-death` plan value types, `computeAward`, hint text

Commit range: `e774a09bb..ccf674ea1` (single commit `ccf674ea1`)
Brief: `.superpowers/sdd/plan/task-9-brief.md`
Report: `.superpowers/sdd/plan/task-9-report.md`

## Scope

`git show --stat ccf674ea1`:

```
services/atlas-monster-death/atlas.com/monster/monster/experience.go      | 120 +++++++++
services/atlas-monster-death/atlas.com/monster/monster/experience_test.go | 273 +++++++++++++++++++++
2 files changed, 393 insertions(+)
```

Two new files, both fully reviewed (read whole — under 20 KB each and both
new, so no slicing needed). No other files touched by this commit.

## Findings

### 1. Exported surface matches the brief's Interfaces section — PASS

`experience.go:8-196` (types) and `:206` (`computeAward` signature) declare
exactly: `DamageInput`, `SoloInput`, `MemberInput`, `PartyInput`,
`ExperienceInput`, `Recipient`, `Exclusion`, `ExperiencePlan`, and
`func computeAward(r Recipient, expRate float64, cfg ExperienceConfig) (uint32, uint32, bool)`,
`func levelGateHintText(name string, level uint32) string` — a byte-for-byte
match against brief lines 60 and 121-241 (the implementation block is
identical to the brief's Step 3 code, field for field, comment for comment).
Since Tasks 10-13 build directly on this exported surface, this is the
highest-priority check for this task, and it passes cleanly: no renamed
field, no reordered parameter, no changed return arity.

### 2. `computeAward` arithmetic and guard/clamp behavior — PASS

`experience.go:206-221`, `:225-236`. Verified against the brief's derivation
table by hand for the two non-trivial rows:

- `unequal levels, MVP`: `share = 0.8·100/150 + 0.2 = 0.733333…`; `personal =
  733.33… → 733`; `bonus = 0.10 · 733.33… = 73.33… → 73`. Matches
  `wantPersonal: 733, wantBonus: 73` (`experience_test.go:89-99`).
- `four-member party bonus`: `share = 0.8·50/200 = 0.2`; `personal = 200`;
  `bonus = 0.20·200 = 40`. Matches `experience_test.go:122-132`.

Guard ordering is correct: `TotalPartyLevel == 0` short-circuits before any
division (`experience.go:207-209`), avoiding a `0/0` NaN entirely rather than
relying on `toAwardAmount` to catch it downstream. `toAwardAmount`
(`:225-236`) independently guards NaN/Inf, negative, and `>= MaxUint32` for
both `personal` and `bonus`, and no path casts a `float64` to `uint32`
without going through it first — confirmed by inspection, `uint32(v)` appears
exactly once at `:235`, gated by the three prior checks.

Per the task instructions, the beyond-FR-8.6 clamp (line `232-234`) is a
ruled decision (context.md §3.2), not a finding.

### 3. Rate-before-split ordering (FR-8.2) — PASS

`experience.go:210` computes `exp := r.PooledExp * expRate` before the split
at `:211-214`, and `bonusF := r.PartyBonusMod * personalF` (`:216`) derives
the bonus from the already-rate-multiplied `personalF`, so the bonus is
rate-multiplied for free as the brief states (task-9-brief.md:55). The `bonus
is rate-multiplied (FR-8.2)` test row (`expRate: 3.0` → `1800`/`180`) passed
in the live run below.

### 4. `levelGateHintText` — PASS

`experience.go:239-241` renders the exact string demanded by
task-9-brief.md:104, and `experience_test.go:266-271` asserts with `!=` on
the whole string (not `strings.Contains`), per the brief's explicit
instruction at line 107.

### 5. Test file fidelity — PASS

`experience_test.go` contains all 14 `TestComputeAward` table rows with
values matching the brief's table exactly (`:9-197`), the 3-case
`TestComputeAward_NonFiniteIsGuarded` using `math.NaN()`/`math.Inf(1)`
exactly as specified rather than a literal that can't express non-finite
values (`:199-254`), and `TestLevelGateHintText` (`:256-262`). Every row
constructs a `Recipient` literal directly — no test-only constructor, no
`*_testhelpers.go` file, consistent with the repo's Builder-pattern
convention for cases where it doesn't apply.

### 6. Live verification — PASS

Ran from the module root (`services/atlas-monster-death/atlas.com/monster`):

```
go build ./...          # clean, no output
go vet ./monster/...     # clean, no output
gofmt -l monster/experience.go monster/experience_test.go   # no output (already gofmt'd)
go test ./monster/ -run 'TestComputeAward|TestLevelGateHintText' -v
```

All 14 `TestComputeAward` subtests PASS, all 3
`TestComputeAward_NonFiniteIsGuarded` subtests PASS, `TestLevelGateHintText`
PASS. (One earlier `go build` attempt in this same session transiently
failed with `"sort" imported and not used` in `interval.go`, a file this
commit does not touch and does not modify — a second immediate rebuild was
clean, and `git status`/`git diff ccf674ea1` show the working tree exactly
matches HEAD with zero uncommitted changes. This reads as a transient/other-
agent artifact in the shared worktree at the moment of that one run, not a
defect in this commit; `interval.go` at HEAD does use `sort.Slice` at
`interval.go:37`.)

### 7. Determinism (FR-9.1) — not applicable

This unit contains no iteration over a map or slice whose order could reach
an observable output (`computeAward` is a pure per-`Recipient` function;
`levelGateHintText` is a pure string format). No finding.

### 8. Exported fields on `Recipient`/`ExperiencePlan`/etc. — not a finding

These are plan/DTO-shaped value types the brief specifies with exported
fields directly in its Step-3 code block (task-9-brief.md:121-241), not
domain aggregates that should route through a `Builder`. The implementer's
self-review at task-9-report.md:46 correctly identifies this distinction.
Since the brief itself is the source of the exported-field shape and the
implementation matches it verbatim, this is not a deviation to flag.

## Not evaluable

None — the full unit (two new files, ~120 + 273 lines) is self-contained,
depends only on `ExperienceConfig`/`DefaultExperienceConfig()` from Task 8
(already landed and confirmed present at `monster/config.go:19,28`), and was
read and verified in full.

## Verdict rationale

Every requirement in the brief's Interfaces section, Step 1 test table, and
Step 3 implementation block is met exactly, with no drift in the exported
surface that Tasks 10-13 depend on. Live build/vet/gofmt/test all clean.

---

verdict: APPROVED
artifact: docs/tasks/task-254-party-experience-sharing/review-task-9.md
scope_confirmed: commit ccf674ea1 only — services/atlas-monster-death/atlas.com/monster/monster/experience.go and experience_test.go (new files); no other files in range e774a09bb..ccf674ea1
blocking: 0
non_blocking: 0
not_evaluable: 0
