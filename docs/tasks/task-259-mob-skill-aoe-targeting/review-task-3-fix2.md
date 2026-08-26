# Task 3 — Fix round 2 re-review (scoped)

Scope: commit `056e1515d3d9acc04bba04ce867dff497ad3dc8d` (range `f5fa66c..056e151`) only —
`services/atlas-monsters/atlas.com/monsters/monster/disease_targets.go` (+4/-2).

## The one open finding

> `monster/disease_targets.go:71:3: goroutineguard: bare go statement; use routine.Go from
> libs/atlas-routine (or add //goroutine-guard:allow <justification>)`

**Ruling required:** convert the fan-out to `routine.Go`, not a lint-suppress comment.

**Verdict: ADDRESSED.**

Evidence:
- `disease_targets.go:73-83` now calls `routine.Go(p.l, p.ctx, func(_ context.Context) { ... })`
  instead of a bare `go func(i int, id uint32) { ... }(i, id)`. No `//goroutine-guard:allow`
  comment was added anywhere in the diff.
- `routine.Go` signature confirmed at `libs/atlas-routine/routine.go:15`:
  `func Go(l logrus.FieldLogger, ctx context.Context, fn func(context.Context))`.
- `ProcessorImpl.l` is `logrus.FieldLogger` and `ProcessorImpl.ctx` is `context.Context`
  (`monster/processor.go:96-97`) — the call site's argument types match exactly.
- Ran the full-repo goroutine-guard sweep: `./tools/goroutine-guard.sh` → `goroutineguard: 89
  module(s), 8 parallel` and exit 0. The gate that originally flagged this line now passes, and
  no other bare `go` statement was newly introduced by this diff (the diff touches only this one
  file/hunk).

## Correctness properties — explicit check

1. **`wg.Wait()` cannot hang on a recovered panic.**
   `routine.Go`'s own recover sits in an outer `defer` around the call to `fn(ctx)`
   (`libs/atlas-routine/routine.go:17-23`). Inside `fn`, the body's own defers are
   `defer wg.Done()` (disease_targets.go:74, registered first) then
   `defer func() { <-sem }()` (disease_targets.go:76, registered second). Go defers run LIFO
   *within* a function, so on panic inside `fn`'s body: the sem-release defer runs, then
   `wg.Done()` runs, and only then does the panic propagate out of `fn(ctx)` to be caught by
   `routine.Go`'s outer recover. `wg.Done()` is guaranteed to fire before the panic reaches the
   recover. **PASS — no hang.**

2. **Loop-variable capture.** `for i, id := range ids` at disease_targets.go:71; module go.mod
   pins `go 1.25.5` (`services/atlas-monsters/atlas.com/monsters/go.mod:3`), which is well past
   the Go 1.22 per-iteration-loop-variable semantics change, so each iteration's `i`/`id` are
   already distinct bindings captured correctly by the closure with no explicit `(i, id)`
   parameter-passing needed. **PASS.**

3. **Index-based slot assembly intact.** `slots[i] = &positionedCharacter{...}` unchanged at
   disease_targets.go:82; the `out` re-assembly loop below is untouched. **PASS.**

4. **No widening on unresolvable position.** The `if err != nil { ...; return }` branch
   (disease_targets.go:78-81) is byte-for-byte unchanged from the pre-fix version — it still just
   returns from the goroutine, leaving `slots[i]` nil, which the assembly loop skips. **PASS.**

5. **No `errgroup` introduced, `x/sync` still indirect.**
   `git diff f5fa66c..056e151 -- go.mod go.sum` for this module is empty — no dependency file
   touched. `grep -n "x/sync" go.mod` → `golang.org/x/sync v0.22.0 // indirect` (unchanged).
   `grep -rn "errgroup" monster/` → no hits. `libs/atlas-routine` was already a resolvable
   dependency before this fix (present in `go.mod` with the local `replace` directive at
   `go.mod:107`, both pre-existing and unchanged by this diff) — confirmed by the empty
   `go.mod`/`go.sum` diff and by `go build ./...` succeeding with no errors. **PASS.**

6. **Concurrency bound unchanged.** `positionLookupConcurrency = 8` (disease_targets.go:53) is
   untouched by the diff; `sem := make(chan struct{}, positionLookupConcurrency)` still gates the
   fan-out. **PASS.**

## Independent test confirmation

Ran directly (not taken from the report):

```
$ cd services/atlas-monsters/atlas.com/monsters && go test ./monster/... -run TestGetDiseaseTargets -race -count=5
ok  	atlas-monsters/monster	1.300s
ok  	atlas-monsters/monster/consumable	1.053s [no tests to run]
...
```

`atlas-monsters/monster` package passed 5 repeated runs under `-race`, no data races, no
failures.

Also independently ran `go build ./...` for the module (clean, no output) and the repo-wide
`./tools/goroutine-guard.sh` (exit 0, `89 module(s)` scanned) as evidence for findings 5 and the
lint-gate verdict above.

## Not evaluable

None — every property in scope was directly checkable from the diff, the called library's source,
and a live test/build/lint run within this review's surface.

## Disposition

All correctness properties hold. The lint finding is genuinely fixed via `routine.Go`, not
suppressed. No new breakage introduced by this 4-insertion/2-deletion diff.
