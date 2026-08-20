# Review — Task 3: Record the resolved tenant on the environment record

Range reviewed: `82a34ffe9..0c0b1cef8` (single commit `0c0b1cef8`).

Files changed: `services/atlas-pr-bootstrap/scripts/bootstrap.sh`,
`services/atlas-pr-bootstrap/test/env_record_test.bats`. Matches the brief's
file list exactly.

## Focused check: renamed pre-existing tests in `env_record_test.bats`

Compared the post-change file against the pre-change file
(`git show 82a34ffe9:services/atlas-pr-bootstrap/test/env_record_test.bats`)
line by line for every pre-existing (Task 1) test.

The curl shim changed from a single-call shim (`CURL_BODY`/`CURL_RC`, `>`
truncating write of argv) to a two-call shim (`GET_BODY`/`GET_RC` for the
non-PATCH branch, `PATCH_RC` for the branch whose argv contains a literal
`PATCH`, `>>` appending write of argv). Five of the eight pre-existing tests
touch stub state; all five were checked:

| Test | Pre | Post | Verdict |
|---|---|---|---|
| `env_record_get GETs this environment's record...` | `CURL_BODY='{"data":{"id":"pr-1411"}}'` | `GET_BODY=` same value | Equivalent. `env_record_get` issues one GET, no `PATCH` argv token, so the new shim takes the `GET_BODY` branch — same body echoed, same assertions (`status`, `$output`, both `grep -qF` header/URL checks) unchanged. |
| `env_record_get mirrors curl's exit status on a 404` | `CURL_RC=22` | `GET_RC=22` | Equivalent. Same single-GET call path; `status -eq 22` assertion unchanged. |
| `env_record_patch sends all five attributes plus the record id` | `CURL_BODY=""` (dead — PATCH calls never read `CURL_BODY`, no assertion depended on it) | line removed entirely | No semantic change. All seven `jq` assertions against `patch_payload` are byte-identical to the pre-change file. `env_record_patch`'s curl call includes `-X PATCH`, so the shim takes the PATCH branch and returns 0 with no body regardless — matches the old shim's `CURL_BODY=""` (empty echo) exactly. |
| `env_record_patch targets the environments PATCH route with the ENVIRONMENT header` | `CURL_BODY=""` | removed | Same as above — no assertion depended on the value; `grep -qF` checks against `$CURL_ARGS` unchanged. |
| `env_record_patch accepts an empty overrides object` | `CURL_BODY=""` | removed | Same — the two `jq` assertions against `patch_payload` are unchanged. |
| `env_record_patch propagates a failing PATCH` | `CURL_RC=22` | `PATCH_RC=22` | Equivalent. `env_record_patch`'s curl call carries `-X PATCH`, so the new shim's PATCH branch checks `PATCH_RC` — same effective failure injection, same `status -eq 22` assertion. |

`env_record_get fails when ATLAS_UI_BASE is unset` and `env_record_get
fails when ATLAS_ENVIRONMENT is unset` touch no stub state and are
byte-identical pre/post.

**Conclusion: no assertion was weakened, dropped, or changed in strength.**
Every renamed/removed line is either a straight rename that preserves the
exact same branch of the new shim being exercised, or removal of a dead
assignment the new shim never reads (and never affected the assertions in
the first place, since PATCH calls in the old shim also ignored
`CURL_BODY`'s content for anything but the return-value echo, which is
empty either way). The `patch_payload` / `grep -qF` / `[ "$status" -eq N ]`
assertions themselves are untouched across all eight pre-existing tests.

## Global constraints

1. **PATCH carries all five attributes and a non-empty phase, read from the
   current record.** `record_environment_tenant` (`bootstrap.sh:114-126`)
   GETs the record, reads `phase` first and fails closed
   (`log error ... "no control-plane environment record..."`, `return 1`)
   if empty, then reads `baseline`/`namespace`/`overrides` from the same
   body and calls `env_record_patch "$phase" "$baseline" "$namespace"
   "$tenant" "$overrides"` — all five params populated, phase always
   non-empty on the success path. Confirmed against
   `services/atlas-configurations/atlas.com/configurations/environments/processor.go:224-226`
   reasoning cited in the code comment (not independently re-verified since
   that file is untouched and outside this task's file list, but the
   comment's claim matches env-record.sh's own doc comment from Task 1,
   which is read-only and already carries the same claim).

2. **PATCH runs before the tenant-config clone.** Call site at
   `bootstrap.sh:361-364` sits directly after `log info "using
   TENANT_ID=$TENANT_ID..."` and before `ATLAS_STEP=tenant-config` (line
   378) — confirmed by line-range grep, not just presence of the function.

3. **Isolated mode is a clean no-op.** Call site is gated by `[
   "${ATLAS_MODE:-isolated}" = "sparse" ]` (`bootstrap.sh:361`) — the exact
   same gate expression `env_header_init` uses (`bootstrap.sh:57`) for the
   same fact. In isolated mode (default) the branch body never executes:
   no `env_record_get`/`env_record_patch` call, no log line, no error.

4. **A non-zero `env_record_get` cannot abort bootstrap under `set -e`.**
   `body=$(env_record_get) || body=""` (`bootstrap.sh:116`) is a compound
   command; the `||` fallback runs and prevents `set -e` from tripping on
   a nonzero GET (missing record / unset `ATLAS_UI_BASE`/`ATLAS_ENVIRONMENT`).
   The subsequent `phase` extraction then legitimately fails closed via the
   explicit `if [ -z "$phase" ]; then ... return 1; fi` — an intentional,
   caught failure, not an uncaught `set -e` abort.

5. **Function opens/closes at column 0; `declare -F` guard present.**
   Verified directly: `sed -n '114p;126p' bootstrap.sh` shows
   `record_environment_tenant() {` and `}` both at column 0, no leading
   whitespace. `env_record_test.bats` extracts it with
   `sed -n '/^record_environment_tenant()/,/^}/p'` and guards the
   extraction with `declare -F record_environment_tenant >/dev/null || {
   echo "record_environment_tenant not extracted..." >&2; return 1; }`
   (added to `setup()`), matching Task 1's guard shape for the other two
   helpers. No stray column-0 `}` inside the function body that would
   truncate the `sed` extraction early — checked the full function text.

6. **`env-record.sh` and `cleanup.sh` are read-only in range.**
   `git diff 82a34ffe9..0c0b1cef8 -- services/atlas-pr-bootstrap/scripts/env-record.sh`
   and the same for `cleanup.sh` both produce empty output — confirmed.

7. **No MIRRORS-array file touched.** `git diff --stat` for the range
   shows exactly two files changed (`bootstrap.sh`,
   `test/env_record_test.bats`); neither is in
   `tools/pr-sparse-mirror-guard.sh:31-41`'s `MIRRORS` array (which lists
   `deploy/k8s/overlays/{pr,pr-sparse}` files only).

## Other observations (non-blocking)

- `record_environment_tenant`'s `env_record_patch` return code is
  propagated as-is (e.g. curl's `22` on a 404-during-PATCH), and the call
  site additionally forces the script to `exit 1` via `|| exit 1` rather
  than preserving the original code. This satisfies FR-3.5's "fail loudly"
  requirement; it does not preserve the specific curl exit status, which
  the brief and tests do not require (the new test only asserts `status
  -ne 0`, not a specific value).
- `patch_payload` compares fine against the two-call shim because
  `CURL_ARGS` is now appended (`>>`) rather than truncated (`>`), so both
  the GET argv and PATCH argv are visible for tests that need to inspect
  the PATCH body specifically — the helper still finds the one line that
  parses as JSON.

## Verification performed

- Read the full diff via the review package and cross-checked against the
  live worktree files (`bootstrap.sh`, `env_record_test.bats`,
  `env-record.sh`, `cleanup.sh`).
- Ran `git show 82a34ffe9:...env_record_test.bats` and diffed the eight
  pre-existing tests by hand against the current file.
- Ran `git diff 82a34ffe9..0c0b1cef8` scoped to `env-record.sh`,
  `cleanup.sh`, and `git diff --stat` for the full range.
- Did not re-run the bats suite (per instructions); relied on the
  implementer report's 16/16 and 137/137 results.

## Not evaluable

- FR-3.2 ("The PATCH MUST target the baseline's `atlas-ingress`") is a
  Task 1 concern implemented in the read-only `env-record.sh`
  (`ATLAS_UI_BASE` resolution) and the deploy overlay wiring — outside this
  task's file list and not re-verified here beyond confirming the file is
  unchanged.
- The `processor.go:224-226` validate-before-backfill claim cited in the
  code comment was not independently re-read from
  `services/atlas-configurations` source; it is an unchanged claim
  carried over from Task 1's `env-record.sh` doc comment (read-only in this
  task) and is outside this unit's touched-file surface.

## Disposition

No blocking findings. The renamed-test focus check found no weakened,
neutered, or vacuous assertions — every pre-existing test still asserts
what it asserted before, exercised through the equivalent branch of the new
two-call shim. All global constraints (phase backfill, ordering, isolated
no-op, `set -e` safety, column-0/guard shape, read-only files, MIRRORS
untouched) are satisfied and verified at specific line numbers.
