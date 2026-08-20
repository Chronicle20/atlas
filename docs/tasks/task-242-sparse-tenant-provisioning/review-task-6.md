# Review: Task 6 — Pin the rendered manifests with a guard script

**Commit range:** `9333f9cea..3ac3d2b46` (single commit `3ac3d2b46`,
`test(deploy): pin the per-overlay environment id and ingress default`)

**Brief:** `.superpowers/sdd/plan/task-6-brief.md`
**Report:** `.superpowers/sdd/plan/task-6-report.md`

## Scope

`git diff --stat 9333f9cea..3ac3d2b46`:

```
tools/overlay-env-guard.sh | 262 +++++++++++++++++++++++++++++++++++++++++++++
tools/verify.sh            |   5 +-
2 files changed, 265 insertions(+), 2 deletions(-)
```

Matches the brief exactly: one new script, one small `verify.sh` wiring
diff. No other files in the range. `deploy/` is untouched by the commit
(confirmed by `git diff 9333f9cea..3ac3d2b46 -- deploy/` returning empty) —
the Step 2 drift test was fully reverted before commit, as required.

## Method

Read the full script (`tools/overlay-env-guard.sh`, 262 lines) and the
`verify.sh` diff. Then, for every one of the 12 assertions in the brief's
table, deliberately corrupted the underlying source file the assertion
reads (`deploy/k8s/overlays/{main,pr,pr-sparse}/kustomization.yaml`,
`deploy/k8s/overlays/main/environment-record.yaml`,
`deploy/k8s/base/atlas-ingress.yaml`, `deploy/k8s/base/kustomization.yaml`),
re-ran `./tools/overlay-env-guard.sh`, confirmed the specific FAIL line and
non-zero exit, then restored the file and confirmed `git status --porcelain
deploy/` was clean before moving to the next assertion. Also ran
`shellcheck`, checked the `kustomize`-absent error path by masking `PATH`,
and confirmed the `verify.sh` wiring and `--facts` selection.

## Findings — all 12 assertions confirmed RED-capable

| # | Overlay | Break applied | Result |
|---|---|---|---|
| 1 | main | `ATLAS_ENVIRONMENT=main` → `=mian` | FAIL: `overlays/main atlas-env ConfigMap does not have ATLAS_ENVIRONMENT: main` |
| 2 | main | `namespace: atlas-main` → `atlas-mainx` (literal 1 left untouched) | FAIL only on the derived-match assertion (`expected mainx`); assertion 1 stayed green — proves assertion 2 genuinely reads `namespace:` at runtime, not a hard-coded `main` on both sides |
| 3 | main | Job `sync-wave: "11"` → `"12"` | FAIL: missing sync-wave "11" |
| 4a | main | Job script host `atlas-configurations...` → `atlas-ingress...` | FAIL: "does not target atlas-configurations..." |
| 4b | main | left `atlas-configurations...` intact, added a second, unrelated `atlas-ingress` string elsewhere in the same doc | FAIL: "routes through atlas-ingress instead of bypassing it" — proves both branches of the assertion are reachable, not just the "missing" one |
| 5 | main | `"phase": "ACTIVE"` → `"PENDING"` | FAIL: missing phase/name pair |
| 6 | pr | `ATLAS_ENVIRONMENT=` → `=pr-1` | FAIL: pr ConfigMap does not have `ATLAS_ENVIRONMENT: ""` |
| 7 | pr | injected a fake `atlas-environment-record` Job resource into the pr overlay | FAIL: "unexpectedly renders an atlas-environment-record Job" |
| 8 | pr-sparse | `ATLAS_ENVIRONMENT=pr-PLACEHOLDER_PR_NUMBER` → `=pr-99` | FAIL: pr-sparse ConfigMap literal mismatch |
| 9 | main+pr | renamed `ATLAS_ENVIRONMENT_DEFAULT` env var name in base | FAIL on both main and pr; pr-sparse still SKIP (unaffected, as ruled) |
| 10 | main+pr | stripped `ATLAS_ENVIRONMENT_DEFAULT` out of the `NGINX_ENVSUBST_FILTER` value in base | FAIL on both main and pr |
| 11 | all 3 | renamed `env-default.conf.template` in base's ConfigMap generator + moved the file | FAIL on all three overlays |
| 12a | all 3 | `$atlas_environment` → `$http_environment` (single line) | FAIL: "missing proxy_set_header ENVIRONMENT $atlas_environment;" on all three |
| 12b | all 3 | kept the new line, re-added the old `$http_environment` line alongside it | FAIL: "still has the old proxy_set_header ENVIRONMENT $http_environment; line" on all three — proves the negative half of assertion 12 is reachable too |

No vacuous assertion found. Every `grep -F`/`contains` call is anchored to
a document scoped by `get_doc()` (kind + 2-space-indented top-level
`metadata.name:`), which correctly distinguishes a Job's own metadata name
from a nested container name at deeper indent (verified against the real
render: `atlas-environment-record` at column 2 vs. `environment-record`
container name at column 8).

Baseline run (`./tools/overlay-env-guard.sh`) is green, exit 0, with 20
PASS lines (assertions 1-8, 11-12 x3, 9-10 x2) plus one explicit SKIP line
for pr-sparse's 9/10 pair.

## Controller ruling on assertions 9/10 (pr-sparse)

Verified `tools/overlay-env-guard.sh:161-227`. Assertions 9 and 10 are
scoped to `main` and `pr` only via `check_ingress_default_env main "$MAIN"`
and `check_ingress_default_env pr "$PR"` — no call for `pr-sparse`.
Instead, line 227 prints an explicit `SKIP` line naming both required
citation points verbatim:

```
overlay-env-guard: SKIP - overlays/pr-sparse atlas-ingress ATLAS_ENVIRONMENT_DEFAULT/NGINX_ENVSUBST_FILTER (ns-overrides.yaml:38-46's env: #PLACEHOLDER_NS_OVERRIDES YAML-parses to env: null, wiping the container's env list until .github/workflows/pr-validation.yml:1027 substitutes it at PR-apply time; by design, not this guard's concern)
```

This matches the ruling exactly: no silent omission, both file:line
citations present. Assertions 8, 11, 12 still cover pr-sparse (confirmed
above — the FAIL lines for 11 and 12b explicitly include
`overlays/pr-sparse`). `ns-overrides.yaml` and
`.github/workflows/pr-validation.yml` were not touched by this commit
(confirmed: not in `git diff --stat`).

## Other checks

- **No new toolchain dependency.** The script uses only `kustomize`,
  `awk`, `grep -F`, `sed`, `mktemp`, `git` — no `yq` call anywhere in the
  file (`grep -n yq tools/overlay-env-guard.sh` → no matches).
- **`kustomize`-absent path.** `PATH=/usr/bin:/bin
  ./tools/overlay-env-guard.sh` → stderr `overlay-env-guard: kustomize not
  found on PATH`, exit 1. Exact string match to the brief's requirement.
- **Accumulation, not fail-fast.** `set -euo pipefail` is set, but every
  assertion block is written as an `if`/`elif`/`else` that calls `fail()`
  (which only sets `status=1` and returns) rather than `exit`d directly —
  confirmed empirically: every corruption test above produced exactly one
  (or two, for #1/#2) FAIL line(s) while every other assertion still ran
  and printed its own PASS/SKIP in the same invocation.
- **`verify.sh` wiring** (`tools/verify.sh:554-561` in the diff):
  `touched` pattern extended to
  `'^(deploy/|tools/gen-lb-ports\.sh|tools/overlay-env-guard\.sh|.*versions\.json)'`,
  a fourth `step "overlay env drift" ./tools/overlay-env-guard.sh` added,
  and the `skip` message updated to `"LB port / version coverage / overlay
  env (no deploy or versions.json change)"`. Matches the brief's Step 3
  block verbatim. `./tools/verify.sh --facts` output includes `gate=overlay
  env drift` among `gates_selected=12`.
- **Executable bit.** `tools/overlay-env-guard.sh` is `-rwxrwxr-x` in the
  working tree.
- **shellcheck.** `shellcheck tools/overlay-env-guard.sh` → clean, exit 0.
  The two `SC2016` disables (for the literal nginx `$atlas_environment` /
  `$http_environment` strings compared via `grep -F`) are legitimate —
  those are not shell expansions.
- **`--help`.** Prints `usage: overlay-env-guard.sh [--help]`, exit 0. No
  other flags, matching the brief.
- **Step 2 revert.** `git diff 9333f9cea..3ac3d2b46 -- deploy/` is empty;
  `deploy/k8s/overlays/main/kustomization.yaml` is unchanged by this
  commit. `git status --porcelain deploy/` is clean after all of this
  review's own drift tests.

## Minor, non-blocking observations

- `tools/overlay-env-guard.sh:184` — the assertion-9 substring check is
  `contains "$doc" "- name: ATLAS_ENVIRONMENT_DEFAULT"`. Because
  `contains()` uses `grep -qF` (substring, not exact-line) match, a
  hypothetical rename to a value that has `ATLAS_ENVIRONMENT_DEFAULT` as a
  *prefix* of a longer name (e.g. `ATLAS_ENVIRONMENT_DEFAULT_X`) would
  still satisfy this check, since the original string is a substring of
  the new one. Confirmed this by accident during testing (first drift
  attempt appended `_X` to the var name and produced a false PASS; a
  clean rename to `XATLAS_ENVIRONMENT_DEFAULT` correctly failed). This is
  a real, narrow gap in assertion 9's `contains` check, but it requires an
  unlikely renaming pattern (suffix-preserving) to slip through, and the
  same class of imprecision exists in the brief's own suggested `grep -F`
  approach for "literal-string facts." Not blocking — flagging for
  awareness, not a required fix.
- Assertion 10's check (`value: POD_NAMESPACE|NS_|ATLAS_ENVIRONMENT_DEFAULT`)
  is a full literal match of the current filter value rather than "value
  contains ATLAS_ENVIRONMENT_DEFAULT" as the brief's assertion text reads
  literally — it would false-FAIL if the filter's variable ordering
  changed (e.g. `ATLAS_ENVIRONMENT_DEFAULT|POD_NAMESPACE|NS_`) even though
  the fact ("value contains ATLAS_ENVIRONMENT_DEFAULT") would still hold.
  This makes the guard slightly more brittle than the brief's letter
  requires, but strictly in the safe direction (over-strict, not
  under-strict) and does not create a vacuous-pass risk. Not blocking.

## Verdict

All 12 assertions are wired to genuinely fail on the fact they claim to
check (verified by breaking each one, including both branches of the two
assertions with negative/positive sub-checks). The pr-sparse SKIP for
assertions 9/10 follows the controller's ruling exactly, citing both
required file:line locations, and does not silently narrow coverage — 8,
11, 12 remain checked against pr-sparse. Assertion 2 is genuinely derived
from the overlay's `namespace:` at runtime, not hard-coded on both sides.
The script accumulates rather than exits on first failure. No new
toolchain dependency; the `kustomize`-absent message matches exactly.
`verify.sh` wiring matches the brief's Step 3 verbatim, and `--facts`
confirms selection. Step 2's temporary drift test was fully reverted.
`shellcheck` is clean.

No blocking findings.
