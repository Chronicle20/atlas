# Review: Task 18 — persist unspent `ap`/`sp` on character creation

Commit under review: `27d3b3d59305a661f49720542485c9f52f45d7b1`
Brief: `.superpowers/sdd/plan/task-18-brief.md`
Report: `.superpowers/sdd/plan/task-18-report.md`

## Scope

`git show --stat 27d3b3d`:

```
character/administrator.go                            | 14 ++++++++++++--
character/processor.go                                 |  2 +-
character/processor_test.go                             | 58 ++++++++++++
kafka/consumer/character/consumer.go                    |  2 +
kafka/consumer/character/consumer_test.go (new)          | 104 ++++++++++++++
kafka/message/character/kafka.go                         |  2 +
```

All six files are under `services/atlas-character/atlas.com/character/`.
`git show --stat` shows no file under `services/atlas-login/` and no
migration file. This matches the brief's stated scope exactly — one commit,
no schema work, additive plumbing.

## 1. Cross-service wire seam

Compared field-for-field:

- `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:80-81` — `AP uint16 \`json:"ap,omitempty"\`` / `SP string \`json:"sp,omitempty"\``
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/character/kafka.go:206-207` — `AP uint16 \`json:"ap,omitempty"\`` / `SP string \`json:"sp,omitempty"\``

Field names, Go types, and JSON tags are identical on both sides. **PASS.**

## 2. End-to-end persistence

Traced the full hop chain and compared each hop against the pre-existing
`Meso`/`Gm` threading, which follows the identical shape at every hop:

- `kafka/consumer/character/consumer.go:371-389` — builder chain gained
  `.SetAp(c.Body.AP).SetSp(c.Body.SP)` alongside the existing
  `.SetGm(c.Body.Gm).SetMeso(c.Body.Meso)`. Same pattern, same site.
- `character/processor.go:383` — `create(...)` call now passes
  `input.ap, input.sp` as two trailing arguments, appended after the
  pre-existing `input.gm, input.meso`.
- `character/administrator.go:17` (signature) and `:47-48` (entity
  assignment) — `create(...)` gained `ap uint16, sp string` params and sets
  `AP: ap` / `SP: sp` on the entity, next to the existing `GM: gm` / `Meso:
  meso`.
- `entity.go:57-58` — `AP uint16` / `SP string` columns already existed
  (read-only in this commit, confirmed via `git show --stat`: entity.go is
  not in the diff).

No hop drops the value; the shape exactly mirrors the `Meso`/`Gm` path already
in place. **PASS.**

## 3. SP-default whitespace fix

`administrator.go:40-43` (per the diff hunk) replaces the previous
hard-coded `SP: "0, 0, 0, 0, 0, 0, 0, 0, 0, 0"` (spaced) literal with a
guard: `if sp == "" { sp = "0,0,0,0,0,0,0,0,0,0" }` (bare comma), then
`SP: sp` on the entity — matching the separator `Model.SPs()`
(`model.go:124-135`) actually parses (`strings.Split(m.sp, ",")` +
`strconv.Atoi` per element, returning early on the first parse error).

Test `TestCreateCharacterDefaultsApAndSp`
(`character/processor_test.go`, new) creates a character through
`character.NewProcessor(...).Create(...)` with no `SetAp`/`SetSp` calls, then
asserts `len(c.SPs()) == 10` and every element is `0`. This exercises the
real creation path (`Create` → `create` → entity insert → re-read through
the `Model`), not a unit test of the literal string.

The implementer's report includes a RED/GREEN trace: with the fallback
literal reverted to the spaced form (signature otherwise unchanged), this
exact test failed with `SP should be 0,0,0,0,0,0,0,0,0,0, was 0, 0, 0, 0, 0,
0, 0, 0, 0, 0`; with the bare-comma fallback restored, it passed. I did not
re-run that revert myself (it would require checking out an intermediate
state), but the artifact evidence is concrete enough — a full go test error
string, not a hand-wave — and the passing test as committed genuinely
regresses the whitespace bug (confirmed by inspecting `SPs()`'s parse logic
directly, above). **PASS.**

## 4. Additive-only behaviour

- `git show --stat 27d3b3d` touches no file under `services/atlas-login/`.
- The one existing caller of `create(...)` (`character/processor.go`) is
  updated in lock-step with the signature change — `grep -rn "= create(\|:=
  create("` inside `character/` confirms `processor.go` is the only call
  site, so there is no orphaned pre-change caller.
- A caller that supplies no `ap`/`sp` (`SetAp`/`SetSp` never called on the
  builder) still gets `AP == 0`, `SP == "0,0,0,0,0,0,0,0,0,0"` — the same
  effective zero-AP behavior as before, and the *intended* SP string content
  (ten zeroes) rather than the previous malformed one. This is the bug fix,
  not a behavior change in effect (three-cornered case: today's callers
  already intend to represent "no unspent SP," they just got a broken
  encoding of that intent).

**PASS.**

## Residual claims (context only, not re-raised as findings per controller ruling)

Verified against source, not re-litigated as a gap:

- `create(...)` (`administrator.go:17-50`) and `SetSP(amount, bookId)`
  (`administrator.go:292-299`) are the only two writers of the `SP` entity
  column. Confirmed by reading `administrator.go` in full for `e.SP =` /
  `SP:` assignment sites — no others exist.
- `SetSP` (`administrator.go:292-299`) does
  `strings.Split(e.SP, ",")` → mutate one index → `strings.Join(sps, ",")`;
  it does not reformat the other nine elements of an already-malformed
  string.
- A row created by the pre-fix `create(...)` therefore still parses to a
  length-1 slice via `SPs()`/`SP(i)`, confirmed directly against
  `model.go:124-135`'s early-return-on-parse-error logic.
- `SPString()` (`model.go:181-183`) is a direct pass-through of the stored
  string — no normalization on read.

All four are accurate. No repair routine or migration was written, correctly
per the controller ruling that this decision is deferred to the user.

## Verification

```
cd services/atlas-character/atlas.com/character && go build ./... && go test ./...
```

Ran it myself: build succeeded, all packages with test files report `ok`
(including `atlas-character/character` and
`atlas-character/kafka/consumer/character`, the two packages with new
tests). Matches the report's claimed output.

## Findings

None. No blocking, no non-blocking, no not-evaluable items.

## Verdict

APPROVED
