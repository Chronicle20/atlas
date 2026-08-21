# Review: Task 6 — extract kill epilogue, carry `deathType` on the wire

Commit range: `4c4a4c3f4..7526324eb` (single commit `7526324eb`)
Brief: `.superpowers/sdd/plan/task-6-brief.md`
Report: `.superpowers/sdd/plan/task-6-report.md`

## Scope

`git diff --stat` — 5 files, 125 insertions / 28 deletions:

- `monster/kafka.go` (+12/-1)
- `monster/processor.go` (+31/-23)
- `monster/processor_catch.go` (+1/-1)
- `monster/producer.go` (+4/-3)
- `monster/producer_test.go` (+77 new file additions)

Matches the brief's Files list plus one undeclared file, `processor_catch.go`,
which the implementer disclosed and justified (required to compile after
`destroyedStatusEventProvider`'s signature changed). Scope confirmed —
no drive-by changes outside what the signature change forced.

## Findings

### 1. Constants and body fields (`kafka.go:57-65, 99-106, 142-149`) — PASS

`DeathTypeUnset byte = 0` / `DeathTypeFadeOut byte = 1` added verbatim per the
brief, with the design-D9 comment. `DeathType byte `json:"deathType"`` is on
`statusEventDestroyedBody` (kafka.go:105) and as the **last field** of
`statusEventKilledBody` (kafka.go:148) — matches brief's stated field order.
Note: the diff hunk context line for this change (`@@ ... type
statusEventCreatedBody struct {`) is misleading on a naive read — the actual
added lines land inside `statusEventDestroyedBody`, confirmed by direct read
of `kafka.go:99-106`. `statusEventCreatedBody` itself is untouched.

Zero-value semantics: `byte` zero-values to `0` = `DeathTypeUnset`, and D9's
consumer contract (`atlas-channel`, task 11) maps `0` → fade-out. An omitting
producer (pre-task-253, or this file's own `Catch`/`Destroy` call sites)
therefore decodes to exactly the pre-task-253 wire behavior. Field name
`deathType`, type `byte` — matches the interface contract task 11/12 will
consume against.

### 2. Provider signatures (`producer.go:31,149`) — PASS

Both `destroyedStatusEventProvider` and `killedStatusEventProvider` take a
trailing `deathType byte` and set it in the body literal. Matches brief
exactly.

### 3. `finalizeKill` extraction (`processor.go:553-580`) — PASS, verified as a pure move

Diffed the extracted body against the pre-extraction `damageCore` block
(`git diff` hunk) line by line: cooldown/attack-cooldown clear, drop-timer
unregister, status-effect-cancelled emit loop, KILLED emit (now carrying
`deathType`), registry removal, boss-revive spawn — identical sequence, no
reordering, no dropped step. `damageCore`'s kill branch now calls
`p.finalizeKill(last.Monster, last.CharacterId, isBoss, revives,
DeathTypeFadeOut)` (processor.go:636) — an ordinary kill, `DeathTypeFadeOut`
is correct.

### 4. `Destroy` call site (`processor.go:1356`) — PASS

`destroyedStatusEventProvider(m, DeathTypeUnset)`. Byte-identical to
pre-change wire output (old code passed no field, defaulting to the same `0`).

### 5. Two undeclared-but-forced call sites — evaluated for semantic correctness, not just compilation

- **`DamageFriendly`'s inline kill (`processor.go:740`)** — passes
  `DeathTypeFadeOut`. This path is an ordinary friendly-monster kill (mob-vs-mob
  damage resolving to death), not a self-destruct or catch/despawn. It runs its
  own near-duplicate of the epilogue (cooldown clear, status-effect cancel,
  KILLED emit, registry removal — processor.go:728-745) rather than calling
  `finalizeKill` (no revives applies to friendly monsters, and this path wasn't
  named in the brief's extraction scope). `DeathTypeFadeOut` is the
  semantically correct value for an ordinary kill and preserves the pre-change
  wire output. Not blocking, but worth flagging as a candidate for a future
  cleanup: this is now the *only* remaining kill-epilogue duplicate outside
  `finalizeKill`, and design D5's "every death runs exactly this sequence"
  claim is technically violated by this one path (it always was, pre-task-253,
  and this task's brief didn't ask to fix it) — noted as non-blocking since
  fixing it is out of this task's declared scope.
- **`Catch`'s success path (`processor_catch.go:170`)** — passes
  `DeathTypeUnset`. A catch removes the monster without an on-field kill
  animation; this is categorically closer to `Destroy` (silent removal) than to
  an animated `finalizeKill` death, and `DeathTypeUnset` preserves the exact
  pre-change wire output (the field didn't exist before, defaulting to `0`).
  Correct and conservative choice for an unspecified path.

Both values are deliberate, not accidental zeros, and both preserve prior wire
behavior for paths the brief was silent on — the correct default when a task
forces a compile-fix in code outside its stated scope.

### 6. `producer_test.go` — coherent, complete, not corrupted by the reported tooling issue — PASS

Read the full 151-line file. Structure: 2 pre-existing tests
(`TestStartControlBodyEncodesControllerHasAggro`,
`TestAggroChangedBodyEncoding`, lines 15-73, unchanged) followed by exactly the
3 new tests the brief specified:

- `TestKilledBodyCarriesDeathType` (lines 75-115) — table-driven, 3 cases
  (`ordinary kill` / `self-destruct action 3` / `no killer`), matches the
  brief's table verbatim including the `killerId=0` → `ActorId=0` case.
- `TestDestroyedBodyCarriesDeathType` (lines 117-139).
- `TestKilledBodyDeathTypeIsOmittedShapeCompatible` (lines 141-150) — raw JSON
  omitting `deathType`, decodes to `0`. Pins the D9 rolling-deploy claim.

`grep -c "^func Test"` → 5 (2 old + 3 new), no duplicates, no truncated
function bodies, no dangling braces. Ran directly:

```
$ go test ./monster/ -run 'DeathType|StartControl|AggroChanged' -v
--- PASS: TestStartControlBodyEncodesControllerHasAggro
--- PASS: TestAggroChangedBodyEncoding
--- PASS: TestKilledBodyCarriesDeathType (3 subtests, all PASS)
--- PASS: TestDestroyedBodyCarriesDeathType
--- PASS: TestKilledBodyDeathTypeIsOmittedShapeCompatible
PASS
ok  	atlas-monsters/monster	1.581s
```

`go build ./...` and `go vet ./...` clean from the module root. The
implementer's reported Edit-tool data loss (worked around via a Bash heredoc
append) did not leave any artifact in the committed file — no half-applied
test bodies, no duplicated `TestKilledBodyCarriesDeathType` definitions.

### 7. Every producer call site passes a deliberate value — swept

```
$ grep -n "killedStatusEventProvider\|destroyedStatusEventProvider" -r monster/*.go
monster/processor_catch.go:170   destroyedStatusEventProvider(claimed, DeathTypeUnset)
monster/processor.go:568         killedStatusEventProvider(m, killerId, isBoss, m.DamageSummary(), deathType)   [finalizeKill, parameterized]
monster/processor.go:740         killedStatusEventProvider(s.Monster, 0, false, s.Monster.DamageSummary(), DeathTypeFadeOut)
monster/processor.go:1356        destroyedStatusEventProvider(m, DeathTypeUnset)
monster/producer.go:31,149       (provider definitions)
monster/producer_test.go:90,119  (test call sites)
```

4 real call sites, all pass an explicit named constant or a parameter that
flows from one (`finalizeKill`'s `deathType`, itself called only with
`DeathTypeFadeOut` at the one current call site, `processor.go:636`). No site
passes a bare `0` literal or leaves the argument to compiler default.

### Repo conventions

`Hp > -1 || RemoveAfter > -1` presence predicate not touched by this unit —
consistent with the standing ruling that this task doesn't touch presence
logic. No raw-comparison packet-codec sites introduced (this unit is
Go-domain-only, no packet codec file touched, consistent with `libs/atlas-packet`
being untouched here — that's task 5/7's surface).

## Not evaluable

None. Full diff surface (5 files, all touched lines) was read; no file over
the slice-first threshold.

## Verdict rationale

Every brief interface item is present with the exact name/type/field-order
specified. `finalizeKill` is a verified pure move. The two undeclared call
sites are judged individually and both pass deliberate, semantically-sound
values that preserve pre-change wire behavior — exactly the right default for
paths the brief didn't specify. The reported test-file tooling issue left no
trace in the committed content. One non-blocking observation (`DamageFriendly`
still duplicates the epilogue instead of calling `finalizeKill`) is noted for
awareness but is pre-existing behavior outside this task's declared scope.

```text
verdict: APPROVED
artifact: docs/tasks/task-253-self-destructing-mobs/review-task-6.md
scope_confirmed: monster/kafka.go, monster/producer.go, monster/processor.go, monster/processor_catch.go, monster/producer_test.go — matches brief's Files list plus one disclosed, justified compile-forced file
blocking: 0
non_blocking: 1
  - services/atlas-monsters/atlas.com/monsters/monster/processor.go:728-745 — DamageFriendly's inline kill still duplicates the finalizeKill epilogue rather than calling it; pre-existing, out of this task's declared scope, flagged for a future cleanup only
not_evaluable: 0
```
