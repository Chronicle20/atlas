# Review — Task R6 (commit 6c6e81622)

## Scope

Single commit `6c6e81622` — "fix(atlas-wz): check errcheck-flagged Close/Fprint
returns in wzdiff". Brief: `.superpowers/sdd/plan/task-R6-brief.md`.
Implementer report: `.superpowers/sdd/plan/task-R6-report.md`.

Files touched (confirmed via `git diff --stat 6c6e81622~1 6c6e81622`):

```
libs/atlas-wz/wzdiff/allowlist.go |  2 +-
libs/atlas-wz/wzdiff/run.go       | 22 +++++++++++-----------
libs/atlas-wz/wzdiff/selfcheck.go | 14 +++++++-------
libs/atlas-wz/wzdiff/xmlload.go   |  2 +-
4 files changed, 20 insertions(+), 20 deletions(-)
```

No other file changed. `go.work.sum` untouched.

## Requirement 1 — fix the 8 listed findings, idiom-matched, no `//nolint`

PASS.

- `libs/atlas-wz/wzdiff/allowlist.go:46` — `defer f.Close()` →
  `defer func() { _ = f.Close() }()`. The handle is `os.Open(path)` at
  `allowlist.go:41` — a read-only open. Discarding the deferred Close error
  is the correct, low-risk idiom here (brief's own carve-out).
- `libs/atlas-wz/wzdiff/xmlload.go:23` — same fix. Handle is `os.Open(path)`
  at `xmlload.go:18`, also read-only. Same reasoning applies.
- `libs/atlas-wz/wzdiff/run.go:184,200,202,219,220,231` — all six
  `fmt.Fprintf`/`fmt.Fprintln` calls prefixed with `_, _ =`. Diff confirms
  the call sites and argument lists are byte-identical apart from the
  prefix — no text, format string, or argument changed. `w` is an
  `io.Writer` function parameter (`run.go:169`, `run.go:199`), a CLI output
  sink, not a file the process itself wrote and needs to fsync/verify — the
  "unchecked write is fine" case, not the "written file whose Close error
  matters" case (there is no file write in this commit at all; only
  `os.Open`/read paths get a deferred Close).

`grep -n nolint` on the full commit diff: no matches. No `//nolint`
suppression introduced.

## Requirement 2 — no other file changes, go.work.sum untouched

PASS. Confirmed via `git diff --stat 6c6e81622~1 6c6e81622` (4 files, all
listed above) and `git show 6c6e81622 --stat -- go.work.sum` (empty diff for
that path).

## Requirement 3 — verification commands, run for real

All three commands specified in the brief were re-run independently in this
review (not just trusted from the report):

```
$ cd libs/atlas-wz && GOTOOLCHAIN=go1.26.0 go build ./...
BUILD_OK
$ GOTOOLCHAIN=go1.26.0 go test ./...
ok for all packages with tests (wzdiff: ok, 0.009s); no failures
$ GOTOOLCHAIN=go1.26.0 ../../.cache/tools/bin/golangci-lint-v2.12.2 run \
    --allow-parallel-runners -c ../../.golangci.yml \
    --new-from-rev a6820d1c4 ./...
0 issues.
EXIT:0
$ GOTOOLCHAIN=go1.26.0 ../../.cache/tools/bin/golangci-lint-v2.12.2 fmt \
    --diff -c ../../.golangci.yml ./...
(no output)
EXIT:0
```

All PASS, matching the report's claims.

## Scope note — `selfcheck.go` (6 additional findings)

Claim to verify: `selfcheck.go` is new on this branch (added in `8c86277c0`,
after gate base `a6820d1c4`) and is therefore legitimately caught by the
rev-gated linter even though the brief's 8-item list (taken from an earlier
gate log) didn't include it.

Confirmed from git directly:

```
$ git log --oneline -- libs/atlas-wz/wzdiff/selfcheck.go
6c6e81622 fix(atlas-wz): check errcheck-flagged Close/Fprint returns in wzdiff
8c86277c0 feat(atlas-wz): whole-archive size-accounting self-check
$ git merge-base --is-ancestor a6820d1c4 8c86277c0 && echo yes
yes
$ git show a6820d1c4:libs/atlas-wz/wzdiff/selfcheck.go
fatal: path 'libs/atlas-wz/wzdiff/selfcheck.go' does not exist in 'a6820d1c4'
```

`selfcheck.go` did not exist at the gate base and was introduced by this
branch after that point. The rev-gated `--new-from-rev a6820d1c4` command
would flag it, and the earlier gate log the brief quoted from evidently
predates `8c86277c0`. The implementer's claim is correct, not a
rationalization.

The 6 additional edits in `selfcheck.go` (lines 104, 108, 109, 111, 117, 120
per the report) are the identical mechanical class: `fmt.Fprintf`/
`fmt.Fprintln` returns on an `io.Writer` parameter (`WriteSelfCheckReport(w
io.Writer, ...)`, `selfcheck.go:103`), same `_, _ =` idiom, same "no file
write in this function" shape as `run.go`. Diff inspection
(`git show 6c6e81622 -- libs/atlas-wz/wzdiff/selfcheck.go`) shows only the
prefix added at each call site — no format string, argument, or control-flow
change. Not scope creep with behavior in it; it is the same fix applied to
a sibling file that the brief's own selection criterion ("code this branch
introduced," "the rev-gated linter flags it too") already covers. The
commit message discloses this deviation explicitly and accurately.

This does technically violate the brief's literal "No other file may
change" (Requirement 2), since `selfcheck.go` wasn't one of the three named
files. I'm treating this as a disclosed, justified, in-class deviation
rather than a blocking defect: the alternative (leaving `selfcheck.go`'s
findings unfixed) would leave Requirement 3's mandated lint-gate-green
condition unmet, which the brief itself prioritizes. Flagging as
non-blocking rather than blocking.

## Behavior-change check (the reviewer's specific charge)

- No altered output text: confirmed by diff — every `fmt.Fprint*` call site
  is byte-identical except for the `_, _ =` prefix, across `run.go` and
  `selfcheck.go`.
- No swallowed error that previously propagated: `fmt.Fprint*` return
  values were never checked or propagated before this change either (they
  were simply unassigned expression statements) — the errcheck-flagged
  return value was always dropped, at compile time as well as at runtime,
  before this commit. This commit only makes the drop explicit; it does not
  newly discard an error that was previously returned/logged/propagated.
- No changed exit code: no `os.Exit`, no error return, no control flow
  touched by any hunk in this commit — confirmed by reading the full diff
  above, which only changes the LHS of each Fprint*/Close call.
- No `//nolint` suppression: confirmed absent (grep above).
- Close-on-write-vs-Close-on-read distinction: both `Close` sites
  (`allowlist.go:46`, `xmlload.go:23`) are on handles from `os.Open` (read),
  not `os.Create`/`os.OpenFile` with a write flag. Neither site is a
  written file whose Close error (e.g. a deferred flush failure) would need
  to propagate. Correct case classification by the implementer.

## Not evaluable

None. The commit's full diff, both touched call-site contexts, and all
verification commands specified in the brief were inspected or re-run
directly within this review.

## Verdict

APPROVED_WITH_FINDINGS — one non-blocking note (the `selfcheck.go`
deviation from the literal "no other file may change" instruction, disclosed
and justified, same mechanical class, does not change behavior).
