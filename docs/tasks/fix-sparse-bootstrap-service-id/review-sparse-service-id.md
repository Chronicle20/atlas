# Review: fix-sparse-bootstrap-service-id (77d4b2ed1..dac6436b9)

Reviewed against `docs/tasks/fix-sparse-bootstrap-service-id/bug-sparse-service-id.md`.

## Scope

`git diff --stat 77d4b2ed1..dac6436b9`:

```
services/atlas-pr-bootstrap/Dockerfile             |   3 +-
services/atlas-pr-bootstrap/scripts/bootstrap.sh   |  37 ++++++++++++++++++---
services/atlas-pr-bootstrap/scripts/service-config.sh | 43 +++++++++++++++++++++-
services/atlas-pr-bootstrap/test/bootstrap_test.bats    |  29 ++++
services/atlas-pr-bootstrap/test/dockerfile_test.bats   |   9 ++
services/atlas-pr-bootstrap/test/service_config_test.bats | 81 ++++++++++++
tools/verify.sh                                         |  22 ++++
```

Matches the bug file's stated fix (Dockerfile package, `new_uuid()`, guard +
header + `|| exit 1` on the three sparse callers, bats coverage, verify.sh
gate). No scope mismatch.

## Findings

### 1. `new_uuid` correctness — PASS

`services/atlas-pr-bootstrap/scripts/service-config.sh:29-45`. Verified by
running the full bats suite (`bats services/atlas-pr-bootstrap/test`, 96/96
ok) and by isolated experiment:

- `${_SC_UUID_PROC-/proc/sys/kernel/random/uuid}` (single dash, not `:-`) is
  correct for the test seam: a test that sets `_SC_UUID_PROC=""` to force
  "no source" must not fall through to the real kernel path the way `:-`
  would (empty counts as unset under `:-`). `-` treats "set to empty" as set,
  so `[ -r "" ]` correctly fails. Confirmed the actual bats tests
  (`service_config_test.bats:143-183`) instead point it at a nonexistent
  path rather than empty string, which works under either form — but `-` is
  still the more defensible choice for the documented contract.
- Safe under `set -u`: both `-` and `:-` forms are immune to `set -u`
  regardless of branch taken; `local id=""` is initialized before any use.
- Safe under `pipefail`: `read -r id < "$proc"` is not a pipeline; no risk.
- Regex `^[0-9a-fA-F]{8}-...{12}$` is a shape check only (no version/variant
  nibble check), matches its documented intent.

### 2. `|| exit 1` on the three sparse callers — comment/rationale is wrong, but behavior is not harmful (non-blocking, but flagged as requested)

**The stated mechanism is incorrect.** `bootstrap.sh:14-21` shows:

```
set -euo pipefail
. "$(dirname "$0")/lib.sh"
# lib.sh resets options to `set -uo pipefail` ...
set -e
```

`bootstrap.sh` restores `set -e` two lines after sourcing `lib.sh`, and this
restoration is **not part of this diff** (`git diff` confirms `set -e` at
line 21 is unchanged). `-e` is active for the entire rest of the script,
including the sparse block at `bootstrap.sh:429-441`. The new comment there
(`bootstrap.sh:439-440`) and `bug-sparse-service-id.md:114-118` ("Fix" item
4) both assert:

> lib.sh turns `set -e` back off, so without this a failed creation was
> only a log line and bootstrap marched on

I verified this empirically (three separate repros, including one that
sources the actual `lib.sh` + a faithful `create_service_config` copy with
`_SC_UUID_PROC` forced to fail): a **bare, untested call** to
`create_service_config` that returns non-zero already aborts the script
under the live `-e`, with or without `|| exit 1` appended — in every trial,
`exit=1` and the "reached after bare call" marker never printed. `-e` does
not need the `||` to fire here; it already would have, because
`create_service_config ...;` alone is a simple untested statement, not
inside an `if`/`&&`/`||` construct.

The actual, correct reason the guard changes anything is different from
what's stated: **pre-fix, nothing about an empty `svc_id` ever returned
non-zero at all** — `curl` succeeded (the server minted its own id),
`kubectl set env SERVICE_ID=` also "succeeded" (writes an empty value
without failing). So `-e` had nothing to catch; the defect was the absence
of a failure signal, not `-e` being off. The new guard clauses
(`bootstrap.sh:395-399`, POST failure at `:420`) are what create the
failure signal; `|| exit 1` on the callers is then redundant given the
already-active `-e`, though harmless (defensive, and matches
`require_env`'s explicit-`exit`-over-implicit-`-e` style).

This is a documentation-accuracy defect in landed code comments and the bug
file, not a functional one — the script's abort behavior on a real failure
is correct either way. Given CLAUDE.md's evidence-grounding rule and that
this was called out as the highest-risk item to verify adversarially, I'm
flagging it as a real finding rather than silently accepting "looks
plausible." Recommend fixing the comment (and the bug file's Fix §4) to say
the guard's job is to *produce* a failure signal that didn't exist before,
not to work around a disabled `-e` — but this does not block merge.

**atlas-drops (not deployed in sparse mode) cannot newly fail the whole
bootstrap.** `bootstrap.sh:428`: `kubectl set env deployment/"$deployment"
SERVICE_ID="$svc_id" 2>/dev/null || log warn "..."` is the last statement in
`create_service_config`; the function's return value is therefore the exit
status of the `||`'s right-hand branch (`log warn`, which always succeeds).
A missing `atlas-drops` Deployment still only warns — confirmed by reading
the code, matching the bug file's "Deliberately not changed" section. Not a
regression.

### 3. ENVIRONMENT header — PASS

`bootstrap.sh:404-419` sends `-H "ENVIRONMENT: $ATLAS_ENVIRONMENT"` on the
POST. Confirmed against source:

- `libs/atlas-env/env.go:26`: `const Key = "ENVIRONMENT"` — the header name
  matches exactly.
- `services/atlas-configurations/atlas.com/configurations/services/administrator.go:16-23`
  (`create`): inserts `string(env.MustFromContext(ctx))` directly as the
  environment column. **`create` never calls `scope.AuthorizeWrite`** —
  that check only exists in `update`/`delete` (lines ~36-79), which need it
  to compare the caller against an *existing* row's owning environment. A
  `create` has no existing row to compare against, so there is no path by
  which adding this header to a POST can produce a 403. Confirmed by
  reading the full file — `AuthorizeWrite` is not referenced in `create`.

### 4. Test honesty — mostly PASS, one vacuous (non-diagnostic) test noted

Ran the new/changed bats files against the pre-fix scripts
(`git show 77d4b2ed1:...`) to confirm each new assertion actually fails
without the change:

- `service_config_test.bats` (6 new tests): 5 of 6 fail correctly against
  pre-fix `service-config.sh` (`new_uuid` undefined → 127, or the
  malformed-UUID assertion fails). **One passes vacuously**: "build_service_config: sparse id is a well-formed UUID, not empty" (line
  ~186) passes against the *old* code too, because the review/dev host has
  real `uuidgen` on `PATH` — it doesn't reproduce the "binary absent"
  condition the bug depends on. That's fine as a happy-path assertion, but
  it is not diagnostic of this bug on its own; the adjacent test
  ("...sparse fails when no UUID source is available", which strips PATH
  and points `_SC_UUID_PROC` at a nonexistent file) is the one that
  actually exercises the regression, and it does correctly fail pre-fix.
- `bootstrap_test.bats` (2 new grep/sed-based static assertions): both
  correctly fail against pre-fix `bootstrap.sh` (verified directly — no
  header string, no guard line before `kubectl set env`).
- `dockerfile_test.bats` (1 new test): correctly fails against pre-fix
  `Dockerfile` (no `util-linux` line).
- `PATH="$bindir"; run ...; PATH="$oldpath"` ordering: in all three affected
  tests, `PATH` is restored **immediately after the `run` line**, before any
  assertion (`[ "$status" -eq 0 ]` etc.). Since `run` always itself returns
  0 (it captures the command's status into `$status` without propagating
  failure), and bats test bodies do not have `set -e` active at the
  point where a plain `[ ]` assertion fails (bats' own trap mechanism
  handles that), a failing assertion does not skip the `PATH="$oldpath"`
  line — it already ran two lines earlier. No leakage risk found.

### 5. `setup()` sourcing `lib.sh` in `service_config_test.bats` — PASS

`lib.sh` runs `set -uo pipefail` at source time (no `-e`). Ran the full
10 pre-existing tests plus the 6 new ones in the same file
post-change: all 16 pass (`bats test/service_config_test.bats` → `1..16`,
all ok). No regression to the pre-existing tests from the added `-uo
pipefail`.

### 6. `verify.sh` gate — PASS, with one documentation gap noted

- `touched '^services/atlas-pr-bootstrap/'` (`tools/verify.sh:519`) is
  anchored the same way every other guard in the file anchors against
  `$CHANGED` (repo-root-relative, no leading `./`, populated by `git diff
  --name-only`). Confirmed this diff's own file list
  (`git diff --name-only 77d4b2ed1..dac6436b9 --
  services/atlas-pr-bootstrap`) matches the pattern.
- Ran `bats services/atlas-pr-bootstrap/test` directly (the same invocation
  `verify.sh` makes): 96/96 pass.
- **CI does not run `tools/verify.sh` at all** — confirmed by grepping
  `.github/workflows/*.yml` for `verify.sh` (only appears in comments) and
  for `bats` (does not appear anywhere in workflows). `tools/verify.sh` is
  exclusively the local/agent pre-PR gate (per CLAUDE.md, "Done means
  verified"); CI implements its own parallel job-per-guard equivalents.
  The hard-failure-on-missing-`bats` behavior therefore cannot break CI —
  it can only fail a developer's local flagless run, which is the intended
  effect (forcing an install rather than a silent skip, matching the
  precedent of `--require-shellcheck` for the shell-tooling guard).
- **Gap**: this new guard is not documented in `docs/verification.md`,
  which is the repo's stated home for "per-guard invariants, escape
  hatches, known CI drift" (including the exact "CI doesn't run this"
  fact established above). Every other verify.sh guard I'd expect to have
  an entry there; this one doesn't get one. Non-blocking, but worth adding
  before/at PR time so the "known CI drift" is written down where the next
  person debugging a verify.sh disagreement will look.

## Not evaluable

- The precise mechanism by which the `ENVIRONMENT` header is extracted from
  the HTTP request into `context.Context` (the middleware wiring, as
  opposed to the `env.Key` constant and `MustFromContext` consumer) was not
  traced — it lives in a shared REST framework outside this diff's touched
  files and outside `services/atlas-configurations`'s services package. I
  verified the header name (`env.Key = "ENVIRONMENT"`) matches what
  bootstrap.sh sends and that this is an established task-232 convention
  (design §8.1) this diff conforms to, not one it introduces — sufficient
  for this review's purpose, but the full request-to-context path was not
  read end-to-end.
- Live Argo/Kubernetes behavior (e.g., an actual retried Job on a real
  failure) was not exercised against a live cluster; the `backoffLimit: 3`
  / `hook-delete-policy` reasoning is based on reading
  `deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml` only.

## Verdict

No blocking defects found. One real, verified-false claim in landed
documentation (both `bootstrap.sh`'s new comment and the bug file's Fix §4)
about *why* `|| exit 1` is needed — the stated "`-e` is off" premise is
factually wrong (`-e` is restored two lines after sourcing `lib.sh` and
never toggled off again), though the change itself is harmless and the
guard clauses it wraps are the actual, correct fix for the real defect. One
non-diagnostic (vacuously-passing) new test that doesn't invalidate the
suite since its neighbor test does exercise the regression. One
documentation gap (new verify.sh guard undocumented in
docs/verification.md).
