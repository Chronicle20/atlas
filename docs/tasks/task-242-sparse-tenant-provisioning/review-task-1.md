# Review: Task 1 — Extract environment-record helpers into env-record.sh

**Range:** `8dfb4f99a..1147693f0`
**Scope:** `services/atlas-pr-bootstrap/scripts/env-record.sh` (new),
`services/atlas-pr-bootstrap/scripts/cleanup.sh`,
`services/atlas-pr-bootstrap/Dockerfile`,
`services/atlas-pr-bootstrap/test/dockerfile_test.bats`,
`services/atlas-pr-bootstrap/test/env_record_test.bats` (new).
Docs artifacts (`context.md`/`design.md`/`plan.md`/`prd.md`) in the range are
prior-phase commits, not part of this unit — not reviewed here.

## Findings

### 1. Verbatim body extraction — PASS

Diffed the pre-change `_dcp_env_get` (originally `cleanup.sh:91-97`) and
`_dcp_patch_phase` (originally `cleanup.sh:150-166`) against the new
`env_record_get`/`env_record_patch` in
`services/atlas-pr-bootstrap/scripts/env-record.sh:14-21` and `:35-51`.
Character-for-character identical: same `curl -fsS` flags, same header set
(`Accept: application/vnd.api+json`, `ENVIRONMENT: $ATLAS_ENVIRONMENT`,
`Content-Type: application/vnd.api+json`), same `jq -nc` payload template,
same `-X PATCH`, same URL construction, same `2>/dev/null` / `>/dev/null`
redirection. No re-derivation.

### 2. cleanup.sh delegation shape — PASS

`services/atlas-pr-bootstrap/scripts/cleanup.sh:96-98` and `:154-156`:
both functions keep their original names (`_dcp_env_get`,
`_dcp_patch_phase`) and their full original doc comments, each with exactly
one added line ("Body now lives in env-record.sh ..."). Bodies are one-line
delegations (`env_record_get` / `env_record_patch "$@"`). All four callers
(`_dcp_env_phase`, `do_deactivate`, `do_drop_control_plane`,
`do_sweep_tenant`) are untouched and still call through the same names —
confirmed no other cleanup.sh hunks besides the source line and these two
function bodies.

### 3. cleanup_test.bats unmodified — PASS

`git diff 8dfb4f99a..1147693f0 -- services/atlas-pr-bootstrap/test/cleanup_test.bats`
returns empty. Zero edits, as required — this is the refactor's acceptance
test and it is untouched.

### 4. env-record.sh sourced-only contract — PASS

- `stat -c '%A' services/atlas-pr-bootstrap/scripts/env-record.sh` →
  `-rw-rw-r--`, not executable.
- No `set` statement anywhere in the file (grep for `^set\b|set -` returns
  nothing) — option state stays owned by `lib.sh`.
- `Dockerfile:69`'s `RUN chmod +x ...` line lists
  `bootstrap.sh cleanup.sh sweep-orphans.sh reclaim-main-bare-keys.sh
  predelete-purge.sh reconcile-minio.sh` — `env-record.sh` is correctly
  absent.
- `Dockerfile:60` adds `COPY scripts/env-record.sh /atlas/env-record.sh`
  in the right place (below `service-config.sh`, above `bootstrap.sh`).
- `test/dockerfile_test.bats:34` adds
  `[ "$base" = "env-record.sh" ] && continue` beside the existing
  `lib.sh`/`version-ports.sh`/`service-config.sh` exclusions.

### 5. Function boundary shape (sed extraction contract) — PASS

Both `env_record_get()` and `env_record_patch()` in `env-record.sh` open
with `name() {` at column 0 and close with `}` at column 0 (verified with
`cat -A`, no leading whitespace, no trailing content on the brace line).
Satisfies the `sed -n '/^name()/,/^}/p'` extraction contract used elsewhere
in this test suite family — although this particular new test sources the
file directly rather than sed-extracting it (see #6), so this property is
belt-and-suspenders here, not load-bearing for this suite specifically; it
is, however, load-bearing for any *other* bats file that later chooses to
sed-extract these two functions instead of sourcing the whole file.

### 6. env_record_test.bats — PASS

`services/atlas-pr-bootstrap/test/env_record_test.bats`:
- Sources `lib.sh` then `env-record.sh` (setup, matching
  `service_config_test.bats:3-17`'s pattern — correct choice here since
  `env-record.sh`, unlike `bootstrap.sh`/`cleanup.sh`, is genuinely
  sourceable with no top-level execution).
- Carries the `declare -F` extraction guard for **both** `env_record_get`
  and `env_record_patch` (lines 17-24), matching
  `data_ingest_test.bats:25-34`'s pattern — a missing definition fails
  `setup` explicitly instead of turning every "must fail" assertion green
  via a 127 exit from `run`.
- curl shim (lines 30-34) records argv to `$CURL_ARGS`, echoes
  `$CURL_BODY`, honors `$CURL_RC` — matches `data_ingest_test.bats:41-48`.
- All 8 bats test names match the brief verbatim; all assertions
  (GET header/URL, unset-var short-circuit before curl runs, exit-code
  mirroring on failure, PATCH's 5-attribute + id payload, empty-overrides
  case, PATCH failure propagation) match the brief's table.
- Fixed env (`ATLAS_UI_BASE=http://ui`, `ATLAS_ENVIRONMENT=pr-1411`) set in
  `setup`, matches brief.

### 7. PATCH body five-attribute + non-empty-phase contract — PASS

`env_record_patch`'s `jq -nc` template
(`env-record.sh:44-50`) always sets `baseline`, `namespace`, `tenant`,
`overrides`, and `phase` from its five positional args — none are
conditionally omitted, so every call sends all five attributes. `phase`
is `$1`, a required positional argument with no default; the function does
not itself validate non-emptiness (matching the pre-extraction original
exactly — the validation lives server-side, per the doc comment above the
function), so a phase-less PATCH would still be a caller bug, not something
this extraction introduced or fixed. This is a verbatim, behavior-preserving
move — consistent with the task's explicit "do not re-derive" instruction.

### 8. MIRRORS array untouched — PASS

None of the 9 files listed in `tools/pr-sparse-mirror-guard.sh:31-41`
appear in the diff's changed-file list.

### 9. Scope — PASS

Diff touches exactly the 5 files named in the brief:
`scripts/env-record.sh` (new), `scripts/cleanup.sh`, `Dockerfile`,
`test/dockerfile_test.bats`, `test/env_record_test.bats`. `bootstrap.sh` is
untouched (correctly deferred to a later task in this plan — task 1 is
scoped only to extracting the helper and wiring cleanup.sh to it).

## Not evaluable

- Did not re-run `bats services/atlas-pr-bootstrap/test` — per instructions,
  relying on the implementer's report (114/114 passing, `cleanup_test.bats`'s
  26 cases included, 0 `not ok` lines).
- shellcheck output not independently re-run; implementer's report claims
  0 findings on `env-record.sh` and pre-existing SC1091/SC2317 noise on
  `cleanup.sh` unrelated to this change — plausible given the diff shape
  (source-line addition + body deletion only) but not independently
  verified.

## Verdict

No defects found. Verbatim-move constraint honored exactly, sourced/
non-executable contract honored exactly, `cleanup_test.bats` genuinely
untouched, new test suite follows the required patterns and guard.
