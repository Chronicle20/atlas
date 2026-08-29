# Review: Task 5 — A governed BuildKit builder (Layer 4)

Commit under review: `19187a6b7` (range `99d015b55..19187a6b7`)
Brief: `.superpowers/sdd/plan/task-5-brief.md`
Report: `.superpowers/sdd/plan/task-5-report.md`

## Scope

`git diff --stat 99d015b55..19187a6b7` shows exactly the six files the brief
names:

```
deploy/buildkit/buildkitd.toml | 12 ++++++
docs/verification.md           | 30 ++++++++++++++
tools/build-services.sh        | 11 ++++-
tools/buildx-bootstrap.sh      | 82 +++++++++++++++++++++++++++++++++++++
tools/buildx-bootstrap_test.sh | 91 ++++++++++++++++++++++++++++++++++++++++
tools/verify.sh                |  3 ++-
6 files changed, 227 insertions(+), 2 deletions(-)
```

No 7th file (`measurements.md` untouched, confirmed by `git diff --numstat`
returning 6 lines). Reviewed the full diff hunks for all six files, plus
read-only `docker buildx ls` / `tools/buildx-bootstrap.sh --check` as
permitted by the dispatch.

## Findings

### 1. Config fidelity — PASS

`deploy/buildkit/buildkitd.toml` byte-compares identical to the brief's
verbatim TOML block (diff against a heredoc of the brief text exits 0).
`max-parallelism = 8` present at `deploy/buildkit/buildkitd.toml:2`; first
`gcpolicy` has `keepBytes = 40000000000`, `keepDuration = 604800`, and the
three filters in the brief's exact order (`:5-8`); second `gcpolicy` has
`all = true` and `keepBytes = 60000000000` (`:11-12`). Precondition for
Task 7's `max-parallelism` budget arithmetic is satisfied.

### 2. Script contract — PASS

`tools/buildx-bootstrap.sh` matches the brief's contract line for line:
- `--check`: makes no changes, exits 1 with
  `buildx-bootstrap: builder '<name>' does not exist — run
  tools/buildx-bootstrap.sh` to stderr (`tools/buildx-bootstrap.sh:57-61`).
- `--force`: `docker buildx rm "$name" || true` then falls through to
  create (`:63-65`, `:80`) — traced the control flow: after `rm`, `exists`
  is false, both `if` blocks are skipped, and execution reaches the
  unconditional `create` at the bottom. Correct.
- Already-exists, no `--force`: `docker buildx use "$name"` plus the
  exact `... (use --force to recreate from <config>)` hint (`:67-71`).
- Builder name `${ATLAS_BUILDER:-atlas}` (`:53`), config
  `deploy/buildkit/buildkitd.toml` (`:54`).
- `set -euo pipefail`; `ROOT="$(cd "$(dirname "$0")/.." && pwd)"; cd
  "$ROOT"` (`:21-23`) — matches `tools/verify.sh:20-23` exactly.
- Unknown option: `unknown option: $1` to stderr, exit 2 (`:47`).

Confirmed live: `tools/buildx-bootstrap.sh --check` (read-only, permitted)
returned rc 0 on this host, consistent with an `atlas` builder already
existing.

### 3. Header comment — PASS

`tools/buildx-bootstrap.sh:5-14` carries both required consequences: (1)
`docker-container` does not write the local image store by default without
`--load`/`--push`, and (2) switching drivers means a cold cache including
the `/go/pkg/mod` and `/root/.cache/go-build` mounts. Both are accurate and
match the brief's wording.

### 4. `--load` in build-services.sh — PASS

`tools/build-services.sh:19` passes `--builder "${ATLAS_BUILDER:-atlas}"`
and `--load` on the `exec docker buildx bake` line, after a
`./tools/buildx-bootstrap.sh --check` gate (`:18`). Header comment
(`:10-13`) states `--load` is mandatory under the container driver and
explains why (no local-image-store write, "succeed" while producing no
image). Matches the brief's literal snippet exactly.

### 5. verify.sh wiring — PASS

`tools/verify.sh:391-393`:
```sh
step "buildx builder" ./tools/buildx-bootstrap.sh --check
step "docker buildx bake (${#TARGETS[@]} target(s))" \
    docker buildx bake --builder "${ATLAS_BUILDER:-atlas}" --set "$BAKE_OUTPUT" "${TARGETS[@]}"
```
sits after `TARGETS` is fully constructed (lines 369-386, target-set logic
and fail-closed resolution untouched) and before the single bake `step` —
confirmed by reading the surrounding block. Still one `docker buildx bake`
invocation over the whole `TARGETS` array; Task 3's target-set construction
is untouched by this diff.

### 6. Test suite quality — PASS, reconciled

Brief's 8-row table maps to exactly 15 assertions with no padding:

| brief row | assertions | count |
|---|---|---|
| `-h` usage | exit 0 + stdout contains `usage:` | 2 |
| unknown option | exit 2 + stderr mentions it | 2 |
| `--check` fails, absent builder | non-zero + remedy names script | 2 |
| config path resolves | script names path + file exists | 2 |
| parallelism declared | contains `max-parallelism = 8` | 1 |
| hard ceiling declared | `all = true` + `keepBytes = 60000000000` | 2 |
| `--check` succeeds after bootstrap (docker) | bootstrap exit 0 + `--check` exit 0 | 2 |
| idempotent bootstrap (docker) | second bootstrap exit 0 + `buildx ls` count == 1 | 2 |

2+2+2+2+1+2+2+2 = 15. Every one of the 15 traces to a specific brief row;
none are extraneous or duplicate assertions on the same fact — reconciles
the implementer's "15/15" against the 8 table rows cleanly.

Docker-touching cases are gated on `command -v docker >/dev/null 2>&1`
(`tools/buildx-bootstrap_test.sh:60`); on a docker-present host (this one)
all 15 assertions run — the gate does not no-op the suite when docker is
present, it only widens coverage. The two docker-only cases operate on a
disposable `ATLAS_BUILDER=zz-atlas-test-$$` name, never the real `atlas`
builder, with `trap cleanup EXIT` removing it.

**Non-blocking observation**: removing the disposable test builder while it
is the currently-selected (`--use`d) builder causes buildx to fall back to
`default` as the ambient active builder — verified live: a `docker buildx
ls` taken during this review showed `default*` selected rather than
`atlas*`, contradicting the controller-confirmed evidence snapshot taken
earlier. This is a real, reproducible side effect of running
`buildx-bootstrap_test.sh`'s docker-gated cases (also self-reported in the
implementer's report, task-5-report.md:140-148, as a known and since-fixed
transient). It does not affect correctness of the code under review: every
caller in this diff (`build-services.sh`, `verify.sh`) passes
`--builder "${ATLAS_BUILDER:-atlas}"` explicitly on every invocation, so
ambient buildx-context selection is never relied upon. Worth a one-line
note in the test's header comment for the next person confused by a
"randomly reset" default builder on a shared host, but not a contract
violation of anything the brief specifies.

### 7. File scope — PASS

`git diff --numstat 99d015b55..19187a6b7` returns exactly 6 lines; the
commit's own `--stat` matches. No 7th file, no `measurements.md` edit.

## Known deferral — acceptance criterion 4

The implementer did not run the full `all-go-services` bake and did not
touch `measurements.md`, deferring the `docker system df` before/after
cache-ceiling measurement to the controller's branch-end bake, per explicit
dispatch instruction not to run a ~30-minute bake in this session.

Assessment: this leaves no CODE defect unverified. The only artifact that
encodes the 60 GB ceiling is `deploy/buildkit/buildkitd.toml`, which
byte-compares identical to the brief's verbatim config (see Finding 1) —
that is the full extent of what static review can confirm. Whether BuildKit
actually honors the two-tier GC policy under real cache pressure across a
full bake is an operational fact about the `docker-container`/buildkitd
runtime, not something inspectable from the diff; it requires the cache to
actually accumulate past the reclaimable threshold, which a single-service
build (`atlas-ban`, exercised and passing per the report) cannot exercise.
Deferring that measurement to the branch-end bake is a missing
*measurement*, not an unverified code path.

## Not evaluable

- Whether the two-tier GC policy actually caps the cache at 60 GB under
  buildkitd's real GC implementation — requires the deferred full bake
  (see above), out of this review's read-only/no-bake constraints.
- `tools/buildx-bootstrap_test.sh` docker-only cases and `tools/verify_test.sh`'s
  42/42 result were controller-confirmed and not independently re-run here
  per the "do not run verify/verify_test/lint/bake" constraint.

## Verdict

APPROVED_WITH_FINDINGS — one non-blocking observation (test's transient
ambient-builder side effect), no blocking defects found. Config, script
contract, header comment, both caller wirings, and the test suite all match
the brief precisely; the deferred acceptance-criterion-4 measurement is a
missing measurement, not an unverified code defect.
