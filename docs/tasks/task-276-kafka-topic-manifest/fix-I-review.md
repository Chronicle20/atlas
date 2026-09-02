# Review: Fix I — gitignore the atlas-kafka gen build artifact

Commit under review: `4d44ff6e1` (parent `f46773f31`)
Brief: `.superpowers/sdd/plan/fix-I-brief.md`
Diagnosis: `docs/tasks/task-276-kafka-topic-manifest/fix-I-diagnosis.md`
Implementer report: `.superpowers/sdd/plan/fix-I-report.md`

## Restriction honored

Per instructions, `tools/verify.sh`, `tools/verify_test.sh`, and `tools/lint.sh`
were NOT executed in this review — the flagless gate is running concurrently
in this worktree and a second run would race it. Findings below are based on
the diff and on cheap, read-only `git` commands.

## Scope

The unit is a single-file `.gitignore` diff. Reviewed the diff itself plus the
mechanical consequences requested in the review brief (tracked-file collision
check, `check-ignore` resolution, binary-not-committed check). No other files
in the commit, so no cross-file/service-boundary surface to trace.

## Mechanical verification (quoted)

**1. Exactly one file changed, and it is `.gitignore`:**

```
$ git diff-tree --no-commit-id --name-only -r 4d44ff6e1
.gitignore
```

PASS — the commit touches nothing else.

**2. Diff content:**

```
$ git show 4d44ff6e1
.gitignore | 5 +++--
 1 file changed, 3 insertions(+), 2 deletions(-)

  @@ -70,6 +70,7 @@ deploy/k8s/secrets.yaml
  -# `go build ./...` in libs/atlas-constants/gen names its binary after the
  -# directory (gen) — trivially recreated, never a deliverable.
  +# `go build ./...` in libs/atlas-constants/gen and libs/atlas-kafka/gen names
  +# their binary after the directory (gen) — trivially recreated, never a deliverable.
   /libs/atlas-constants/gen/gen
  +/libs/atlas-kafka/gen/gen
```

The added pattern is `/libs/atlas-kafka/gen/gen` — anchored at repo root
(leading `/`), full path to the single artifact, not a bare `gen` or a
directory-wide `gen/` pattern. It mirrors the existing
`/libs/atlas-constants/gen/gen` precedent exactly, and the comment above it
was extended (not duplicated) to describe both entries as a pair. This matches
the brief's instruction verbatim ("reuse its comment style so the two read as
a pair, or extend the existing comment to cover both").

**3. No currently-tracked file becomes newly ignored:**

```
$ git ls-files --ignored --exclude-standard -c
(no output)
```

PASS — empty, per the brief's own acceptance criterion. The new pattern does
not shadow any file already under version control.

**4. The new line resolves for the actual artifact path:**

```
$ git check-ignore -v libs/atlas-kafka/gen/gen
.gitignore:76:/libs/atlas-kafka/gen/gen	libs/atlas-kafka/gen/gen
```

PASS — resolves to the newly added line 76, not to some pre-existing broader
rule.

**5. The binary itself is not committed as content, and is now correctly ignored:**

```
$ git status --porcelain=v1 --ignored | grep -i "libs/atlas-kafka/gen"
!! libs/atlas-kafka/gen/gen

$ ls -la libs/atlas-kafka/gen/gen
755  gen  8.4M
```

The 8.4 MB binary is present in the working tree (left over from the
implementer's own build-to-verify step, per their report) but shows as `!!`
(ignored), not `??` (untracked) or staged/tracked. `git diff-tree` above
already confirmed it is not part of the commit's tree. PASS on both counts:
not committed, and now suppressed by the new pattern rather than surfacing as
an untracked file that would trigger the `libs/` fan-out.

## Blocking-condition checklist (per review brief)

- Modification under `tools/` (`verify.sh`, `verify_test.sh`,
  `lib/go-work.sh`): NONE — `git diff-tree` confirms the commit touches only
  `.gitignore`. PASS.
- `libs/atlas-kafka/gen/gen` committed as content: NO — confirmed absent from
  the commit's file list, and currently sits ignored/untracked in the working
  tree, not staged. PASS.
- Ignore pattern broader than needed: NO — `/libs/atlas-constants/gen/gen`-style
  anchored full path, not `gen/` or bare `gen`. `git ls-files --ignored
  --exclude-standard -c` returning empty is the direct proof no tracked path
  anywhere in the repo (including any unrelated `gen` source file/dir) is
  caught by it. PASS.

## Brief compliance

- Single entry added, adjacent to the `libs/atlas-constants` precedent, same
  comment style extended to cover both — matches the brief's "Files" and
  "Patterns to copy" sections exactly.
- Hard constraints (no `tools/` edits, no binary commit, no gate re-run, no
  `git add -A`) all honored per the diff and the implementer's own report,
  cross-checked mechanically above rather than taken on trust.
- Root cause in the diagnosis (untracked generator binary inside a `libs/`
  module causing `verify.sh` to see a `libs/` change and fan out to 2
  consumers) is addressed at its source: the artifact no longer surfaces as an
  untracked file.

## Not evaluable

- Whether the fix actually makes the flagless `tools/verify.sh` gate pass
  end-to-end (the original 5 failing assertions) is NOT evaluated here per
  explicit instruction not to run `verify.sh`/`verify_test.sh`/`lint.sh` while
  the flagless gate runs concurrently. The implementer's report claims a prior
  `--facts --quick` run (before this fix, after manually removing the stray
  binary) returned `changed_libs=none`, `fanout_reason=none`,
  `modules_selected=0` — consistent with the diagnosis, but that run predates
  this commit and was not independently reproduced by this review.

## Verdict

APPROVED. The change is exactly the one-line, precedent-matching fix the
brief specified, does not touch any gate logic, does not commit the binary,
and the ignore pattern is correctly anchored with no tracked-file collision.
