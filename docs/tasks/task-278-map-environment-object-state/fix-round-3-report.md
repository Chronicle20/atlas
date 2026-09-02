# Fix round 3 report — lint gate failure

## What I did

Fixed the single staticcheck QF1012 finding at
`services/atlas-maps/atlas.com/maps/data/map/object/processor_test.go:29`.
Replaced:

```go
b.WriteString(fmt.Sprintf(
    `{"id":"object-%d","type":"objects","attributes":{"name":"object-%d","state":%d}}`,
    id, id, id,
))
```

with:

```go
fmt.Fprintf(&b,
    `{"id":"object-%d","type":"objects","attributes":{"name":"object-%d","state":%d}}`,
    id, id, id,
)
```

Format string and arguments are unchanged; only the call shape changed. Did
not touch `services/atlas-maps/atlas.com/maps/data/map/monster/processor_drain_test.go`,
which uses the identical idiom but is outside this diff (per brief
constraint).

Both `fmt` and `strings` imports remain used elsewhere in the file (`fmt.Sprintf`
on line 34/35, `strings.Builder` declaration on line 24), so no import changes
were needed.

## Verification

```sh
cd services/atlas-maps/atlas.com/maps && gofmt -l . && go build ./... && go test ./data/map/object/... -count=1 -v
```

Output:

```
=== RUN   TestGetDefaultStateReturnsDeclaredState
--- PASS: TestGetDefaultStateReturnsDeclaredState (0.01s)
=== RUN   TestGetDefaultStateReturnsErrUnknownObjectForUndeclaredName
--- PASS: TestGetDefaultStateReturnsErrUnknownObjectForUndeclaredName (0.00s)
=== RUN   TestGetDefaultStateDrainsBeyondOnePage
--- PASS: TestGetDefaultStateDrainsBeyondOnePage (0.00s)
PASS
ok  	atlas-maps/data/map/object	0.023s
```

`gofmt -l .` produced no output (clean). All three tests in `data/map/object`
pass, including `TestGetDefaultStateDrainsBeyondOnePage` (the page-2 drain
assertion).

## Files changed

- `services/atlas-maps/atlas.com/maps/data/map/object/processor_test.go` —
  `WriteString(fmt.Sprintf(...))` → `fmt.Fprintf(&b, ...)`, no behavior change.

## Self-review

- Diff is exactly the two-line shape change the brief specified; no
  assertions, fixtures, or test names touched.
- `git diff --stat` for the commit shows only this one file, 2 insertions/2
  deletions.
- Did not run `tools/verify.sh`, `tools/lint.sh`, `-race`, or docker per
  brief instruction — module-local build/test/gofmt only.

## Issues or concerns

None. The fix is mechanical and verified locally; the repo-wide gate re-run
is left to the controller as instructed.

## Commit

`0054591a5` — fix(atlas-maps): use fmt.Fprintf instead of WriteString(fmt.Sprintf) in object test

Branch confirmed: `task-278-map-environment-object-state`, worktree root
confirmed as `<repo-root>/.worktrees/task-278-map-environment-object-state`.
