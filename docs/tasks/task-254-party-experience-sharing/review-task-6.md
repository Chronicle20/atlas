# Review: Task 6 — per-character hint throttle (D10)

Commit range reviewed: `7e2278f81..72b4f7332` (single commit
`72b4f7332 feat(atlas-monster-death): throttle level-gate hints per tenant and character (D10)`).

Brief: `.superpowers/sdd/plan/task-6-brief.md` (Task 6 section).
Report: `.superpowers/sdd/plan/task-6-report.md`.

## Scope

`git diff --stat 7e2278f81..72b4f7332` shows exactly two new files, both under
`services/atlas-monster-death/atlas.com/monster/system_message/`:

- `throttle.go` (+78)
- `throttle_test.go` (+157)

No other files touched. Matches the brief's file list exactly. No wiring into
`Processor`/`kafka.go`/`producer.go` — correctly deferred, per both the brief
("Consumes: nothing") and the report, to Task 11.

## Requirement-by-requirement

1. **`throttleKey{tenantId uuid.UUID, characterId uint32}`, `Throttle{mu, window,
   capacity, now, last}`** — `throttle.go:11-14, 25-31` matches the brief's
   struct shape verbatim (unexported fields only, no exported struct fields).

2. **`NewThrottle(window, capacity, now) *Throttle`** — `throttle.go:34-41`,
   constructor-only, initializes `last` as an empty map. Matches.

3. **`Allow` semantics** — `throttle.go:45-64`:
   - Denies when `prior` exists and `n.Sub(prior) < t.window` (`throttle.go:53`)
     — this is the `elapsed < window` check the brief specifies, so a call at
     exactly the window boundary is **allowed**, matching
     `TestThrottle_Allow/boundary:_exactly_the_window_is_allowed`.
   - Sweep on capacity: `if len(t.last) >= t.capacity { ... delete stale ...
     }` (`throttle.go:56-62`), cutoff `n.Add(-t.window)`, deletes entries
     `ts.Before(cutoff)`. Matches the brief's sweep description exactly.
   - Records `t.last[k] = n` and returns `true` only after the sweep check,
     so a fresh key is always admitted regardless of capacity pressure (map
     can transiently exceed `capacity` if all entries are still fresh — this
     is the brief's own documented tradeoff, not a bug; brief only asserts
     `len(th.last) <= 4` after the entries in the test *are* stale).
   - Whole body under `t.mu.Lock()/Unlock()` (`throttle.go:46-47`) — single
     critical section, no lock released between read-check-write, so no
     TOCTOU race between the window check, the sweep, and the record.

4. **`GetHintThrottle()`** — `throttle.go:66-77`, package-level `sync.Once` +
   package-level `*Throttle`, lazily built with `NewThrottle(time.Minute,
   4096, time.Now)`. Matches the brief's window/capacity/clock values and
   mirrors the `sync.Once` pattern used by `GetMonsterRegistry()` in
   `atlas-monsters` (cited in both brief and report; not independently
   re-verified here since it is outside this unit's diff, but the shape
   `sync.Once.Do` + package var is a standard idiom and matches what the
   brief asked for).

## Test verification

All six `TestThrottle_Allow` subtests, `TestThrottle_SweepsWhenOverCapacity`,
and `TestThrottle_ConcurrentAllowIsRaceFree` are present in
`throttle_test.go` and match the brief's table/case descriptions exactly
(fake-clock via captured `time.Time` + closure, no `time.Sleep`; concurrency
test uses the real clock with 50 goroutines and a `sync.WaitGroup`, asserting
exactly 50 distinct characters admitted).

Ran directly in the worktree:

```
cd services/atlas-monster-death/atlas.com/monster && go build ./... && go test ./system_message/... -race -v
```

Result: all tests PASS, including under `-race` (no data race reported on
the shared `sync.Mutex`-protected map). Also ran `go vet ./system_message/...`
(clean) and `gofmt -l system_message/` (no output — correctly formatted).

The report's captured RED (`Throttle`/`NewThrottle` undefined, compile
failure) → GREEN (all tests pass) evidence is consistent with what a fresh
run reproduces; the RED step itself (moving the implementation aside) is not
independently re-run here since the implementation file exists in the commit
as expected, but the GREEN evidence is directly reproduced above.

## Conventions

- No exported struct fields, construction via `NewThrottle` only — matches
  the project's constructor-not-struct-literal convention (the "immutable
  models" rule technically targets domain models; this is a utility type,
  but the implementer's self-review correctly notes the same shape is
  preserved anyway).
- No `*_testhelpers.go` file created; test setup is inline via a struct
  literal + closure, appropriate for a non-domain utility with no existing
  Builder.
- No `// TODO`, no stub, no placeholder — confirmed by reading the full
  78-line implementation.
- Comment block on `Throttle` (`throttle.go:16-23`) documents the D10
  per-replica rationale, matching the brief's own text.

## Not evaluable

- The claim that `GetHintThrottle`'s `sync.Once` pattern "mirrors"
  `GetMonsterRegistry()` in `atlas-monsters/monster/registry.go` was not
  independently verified — that file is in a different service and outside
  this unit's diff, and correctness of Task 6 does not depend on that file's
  actual contents (the pattern itself is a standard, self-contained Go
  idiom). Not a blocking concern; noted only for completeness. Counted as
  not evaluable since it was never checked.

## Findings

None blocking. No non-blocking findings beyond the informational item above.

## Verdict

APPROVED.
