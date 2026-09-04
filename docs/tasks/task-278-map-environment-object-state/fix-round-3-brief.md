# Fix round 3 — lint gate failure

## Facts

```
task=task-278-map-environment-object-state
worktree=<repo-root>/.worktrees/task-278-map-environment-object-state
branch=task-278-map-environment-object-state
head=6c9acd433
toolchain=go1.27.0
```

## Origin

The flagless `tools/verify.sh` run at `6c9acd433` returned exit 1. Exactly one
check failed — everything else (69 modules built/vetted/tested with `-race`,
plus every guard) passed. The failing block, verbatim:

```
services/atlas-maps/atlas.com/maps/data/map/object/processor_test.go:29:3: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
		b.WriteString(fmt.Sprintf(
		^
1 issues:
* staticcheck: 1
lint.sh: LINT FAIL — services/atlas-maps/atlas.com/maps
```

## The fix

In `services/atlas-maps/atlas.com/maps/data/map/object/processor_test.go`
around line 29, replace the `b.WriteString(fmt.Sprintf(...))` call on the
`strings.Builder` with `fmt.Fprintf(&b, ...)`, keeping the format string and
arguments identical.

Drop the `fmt` or `strings` import only if it genuinely becomes unused — check,
do not assume. `gofmt -l` must come back clean.

## Why the pattern was there

This test was written by copying
`services/atlas-maps/atlas.com/maps/data/map/monster/processor_drain_test.go`,
which uses the same `WriteString(fmt.Sprintf(...))` idiom. **Do not "fix" the
monster file** — it is outside this change's diff, the lint gate only grades
changed code, and touching it widens the diff for no gate benefit.

## Constraint

This is a lint-shape change only. The test's behaviour must be byte-identical:
the JSON:API fixture it renders, the two-page drain it sets up, and every
assertion stay exactly as they are. If the test output changes at all, you have
done too much.

## Verification

```sh
cd services/atlas-maps/atlas.com/maps && gofmt -l . && go build ./... && go test ./data/map/object/... -count=1 -v
```

The three tests in `data/map/object` must still pass, including the page-2
drain assertion.

Do NOT run `tools/verify.sh`, `tools/lint.sh`, `-race`, or docker — the
repo-wide gate is re-run separately after you commit.

## Files

- `services/atlas-maps/atlas.com/maps/data/map/object/processor_test.go` — the only file to change; line ~29, `WriteString(fmt.Sprintf(` → `fmt.Fprintf(&b,`

Patterns to copy: none needed — this is a one-call rewrite.

## Commit

One commit on `task-278-map-environment-object-state`. Add the named path only;
no `git add -A` / `git add .`. No destructive git ops. Verify the branch after
committing.
