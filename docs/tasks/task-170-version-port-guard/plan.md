# Version-Port Coverage Guard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire LB ports for gms v48/61/72/79 and add a CI guard that fails when the socket-config-template version set diverges from `deploy/k8s/base/versions.json`.

**Architecture:** Two pure shell+jq deliverables in `tools/`. Part B (Task 1) adds four rows to `versions.json` and regenerates the LB yaml via the existing `gen-lb-ports.sh`, making the tree internally consistent. Part A (Tasks 2–3) adds `check-version-coverage.sh` — a strict, bidirectional `(region, major)` set-equality check between the template filenames and `versions.json` — with a hermetic test, then wires it into the existing `gen-lb-ports` CI job.

**Tech Stack:** Bash, jq, GitHub Actions. No Go changes.

## Global Constraints

- Comparison key is `(region, majorVersion)` — never the full triple. Two minors of one major must collapse to one entry.
- Strict bidirectional equality: fail on template-without-version AND version-without-template. No allowlist/`deferred` escape hatch.
- Both new scripts: `#!/usr/bin/env bash`, `set -euo pipefail`, resolve repo root via `git rev-parse --show-toplevel`, `chmod +x`, shellcheck-clean.
- Port formula (from `services/atlas-pr-bootstrap/scripts/version-ports.sh`): login `= major*100`, channel `= login+1`. Do not hardcode ports anywhere except test fixtures.
- Work in the worktree `.worktrees/task-170-version-port-guard` (branch `task-170-version-port-guard`). Verify branch after each commit.
- **Task order matters:** Task 1 (wiring) precedes Task 2 (guard) so the guard's real-tree verification passes; the guard on an unwired tree would (correctly) fail.

---

### Task 1: Wire gms v48/61/72/79 LB ports

**Files:**
- Modify: `deploy/k8s/base/versions.json` (append 4 entries to `.versions[]`)
- Modify (regenerated): `deploy/k8s/base/atlas-login.yaml`, `deploy/k8s/base/atlas-channel.yaml`

**Interfaces:**
- Consumes: `tools/gen-lb-ports.sh` (existing), `version-ports.sh` formula.
- Produces: a `versions.json` whose `(region, major)` set equals the template set — Task 2's guard relies on this being true on the real tree.

- [ ] **Step 1: Add the four rows to `versions.json`**

Insert after the `gms 12` line (order is cosmetic — `gen-lb-ports.sh` sorts by `majorVersion`), so `.versions[]` reads:

```json
{
  "$schema": "./versions.schema.json",
  "description": "Game versions this environment exposes on the login/channel LoadBalancers. Edit this list + run tools/gen-lb-ports.sh to (re)generate the port blocks in atlas-login.yaml/atlas-channel.yaml.",
  "versions": [
    { "region": "gms", "majorVersion": 12,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 48,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 61,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 72,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 79,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 83,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 84,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 87,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 92,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 95,  "minorVersion": 1 },
    { "region": "jms", "majorVersion": 185, "minorVersion": 1 }
  ]
}
```

- [ ] **Step 2: Regenerate the LB yaml**

Run: `tools/gen-lb-ports.sh`
Expected output: `gen-lb-ports: wrote .../atlas-login.yaml` and `... atlas-channel.yaml`.

- [ ] **Step 3: Verify the new ports landed and nothing drifted**

Run:
```bash
tools/gen-lb-ports.sh --check
grep -E 'containerPort: (4800|6100|7200|7900)' deploy/k8s/base/atlas-login.yaml
grep -E 'containerPort: (4801|6101|7201|7901)' deploy/k8s/base/atlas-channel.yaml
```
Expected: `--check` exits 0 (no output / no diff); each `grep` prints its four lines.

- [ ] **Step 4: Verify existing gen-lb-ports tests still pass**

Run: `tools/gen-lb-ports_test.sh`
Expected: ends with `ALL PASS`.

- [ ] **Step 5: Commit**

```bash
git add deploy/k8s/base/versions.json deploy/k8s/base/atlas-login.yaml deploy/k8s/base/atlas-channel.yaml
git commit -m "feat(deploy): wire LB ports for gms v48/61/72/79 (task-170)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: `check-version-coverage.sh` guard + hermetic test

**Files:**
- Create: `tools/check-version-coverage.sh`
- Create (test): `tools/check-version-coverage_test.sh`

**Interfaces:**
- Consumes: `deploy/k8s/base/versions.json`, `services/atlas-configurations/seed-data/templates/template_*.json`.
- Produces: `tools/check-version-coverage.sh` — exit 0 when the `(region, major)` template set equals the `versions.json` set, exit 1 otherwise with a stderr line naming each offending `<region> <major>`. Task 3 invokes it by path in CI.

- [ ] **Step 1: Write the hermetic test**

Create `tools/check-version-coverage_test.sh`:

```bash
#!/usr/bin/env bash
# check-version-coverage_test.sh — hermetic regression tests for
# tools/check-version-coverage.sh. Builds a throwaway git repo with fixture
# templates + versions.json (the script resolves paths via git rev-parse).
#     tools/check-version-coverage_test.sh
set -euo pipefail

SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/check-version-coverage.sh"
[ -x "$SCRIPT" ] || { echo "FATAL: $SCRIPT not executable" >&2; exit 2; }

fails=0
assert_eq() { if [ "$2" = "$3" ]; then echo "ok   - $1"; else echo "FAIL - $1 (want '$2', got '$3')" >&2; fails=$((fails+1)); fi; }
assert_contains() { if printf '%s\n' "$3" | grep -qF -- "$2"; then echo "ok   - $1"; else echo "FAIL - $1 (missing '$2')" >&2; fails=$((fails+1)); fi; }

tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" config user.email t@t.t; git -C "$tmp" config user.name t
TPL="$tmp/services/atlas-configurations/seed-data/templates"
mkdir -p "$tmp/deploy/k8s/base" "$TPL" "$tmp/tools"
cp "$SCRIPT" "$tmp/tools/check-version-coverage.sh"

set_versions()    { printf '{ "versions": [ %s ] }\n' "$1" > "$tmp/deploy/k8s/base/versions.json"; }
reset_templates() { rm -f "$TPL"/template_*.json; }
add_template()    { : > "$TPL/template_$1.json"; }   # $1 = gms_83_1
run() { set +e; out="$( cd "$tmp" && ./tools/check-version-coverage.sh 2>&1 )"; rc=$?; set -e; }

# --- Test 1: in-sync sets pass ---
reset_templates; add_template gms_12_1; add_template gms_83_1
set_versions '{ "region": "gms", "majorVersion": 12, "minorVersion": 1 }, { "region": "gms", "majorVersion": 83, "minorVersion": 1 }'
run
assert_eq "in-sync exit 0" "0" "$rc"

# --- Test 2: template without a versions.json entry fails + names it ---
reset_templates; add_template gms_48_1; add_template gms_83_1
set_versions '{ "region": "gms", "majorVersion": 83, "minorVersion": 1 }'
run
assert_eq "template-without-version exit 1" "1" "$rc"
assert_contains "names gms 48" "gms 48" "$out"

# --- Test 3: versions.json entry without a template fails + names it ---
reset_templates; add_template gms_83_1
set_versions '{ "region": "gms", "majorVersion": 83, "minorVersion": 1 }, { "region": "gms", "majorVersion": 92, "minorVersion": 1 }'
run
assert_eq "version-without-template exit 1" "1" "$rc"
assert_contains "names gms 92" "gms 92" "$out"

# --- Test 4: two minors of one major, single version entry → no false mismatch ---
reset_templates; add_template gms_83_1; add_template gms_83_2
set_versions '{ "region": "gms", "majorVersion": 83, "minorVersion": 1 }'
run
assert_eq "two-minors-one-major exit 0" "0" "$rc"

echo; [ "$fails" -eq 0 ] && echo "ALL PASS" || { echo "$fails FAILED" >&2; exit 1; }
```

Then: `chmod +x tools/check-version-coverage_test.sh`

- [ ] **Step 2: Create an always-pass stub so the harness runs (produces a real red)**

Create `tools/check-version-coverage.sh` as a stub that always exits 0, and `chmod +x tools/check-version-coverage.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
exit 0
```

- [ ] **Step 3: Run the test to verify it fails (red)**

Run: `tools/check-version-coverage_test.sh`
Expected: FAIL — Test 1 & 4 pass but `template-without-version exit 1` and `version-without-template exit 1` FAIL (stub always exits 0), ending with `2 FAILED`.

- [ ] **Step 4: Implement the real guard**

Replace `tools/check-version-coverage.sh` with:

```bash
#!/usr/bin/env bash
# check-version-coverage.sh — CI guard: the set of client versions that have a
# socket config template MUST equal the set declared in versions.json, keyed on
# (region, majorVersion). Fails (exit 1) in either direction:
#   - a template with no versions.json entry (the version would ship with no LB
#     ports — the blind spot that let PR #971 land v48/61/72/79 unported), or
#   - a versions.json entry with no template (LB ports for a phantom version).
# Pure shell + jq; modifies nothing. See docs/tasks/task-170-version-port-guard.
set -euo pipefail
export LC_ALL=C   # deterministic sort/comm collation

REPO_ROOT="$(git rev-parse --show-toplevel)"
VERSIONS="$REPO_ROOT/deploy/k8s/base/versions.json"
TEMPLATE_DIR="$REPO_ROOT/services/atlas-configurations/seed-data/templates"

[ -f "$VERSIONS" ]     || { echo "check-version-coverage: missing $VERSIONS" >&2; exit 1; }
[ -d "$TEMPLATE_DIR" ] || { echo "check-version-coverage: missing $TEMPLATE_DIR" >&2; exit 1; }

# (region major) pairs declared in versions.json, sorted & unique.
versions_set() {
    jq -r '.versions[] | "\(.region) \(.majorVersion)"' "$VERSIONS" | sort -u
}

# (region major) pairs parsed from template_<region>_<major>_<minor>.json names.
templates_set() {
    local f base region rest major
    for f in "$TEMPLATE_DIR"/template_*.json; do
        [ -e "$f" ] || continue          # nullglob-safe
        base="$(basename "$f" .json)"    # template_gms_83_1
        base="${base#template_}"         # gms_83_1
        region="${base%%_*}"             # gms
        rest="${base#*_}"                # 83_1
        major="${rest%%_*}"              # 83
        printf '%s %s\n' "$region" "$major"
    done | sort -u
}

vset="$(versions_set)"
tset="$(templates_set)"

missing_versions="$(comm -23 <(printf '%s\n' "$tset") <(printf '%s\n' "$vset"))"
missing_templates="$(comm -13 <(printf '%s\n' "$tset") <(printf '%s\n' "$vset"))"

rc=0
if [ -n "$missing_versions" ]; then
    rc=1
    while read -r region major; do
        [ -z "$region" ] && continue
        echo "check-version-coverage: $region $major has a socket config template but no deploy/k8s/base/versions.json entry — add it and run tools/gen-lb-ports.sh, or remove the template." >&2
    done <<< "$missing_versions"
fi
if [ -n "$missing_templates" ]; then
    rc=1
    while read -r region major; do
        [ -z "$region" ] && continue
        echo "check-version-coverage: $region $major has a deploy/k8s/base/versions.json entry (LB ports) but no socket config template in services/atlas-configurations/seed-data/templates." >&2
    done <<< "$missing_templates"
fi

[ "$rc" -eq 0 ] && echo "check-version-coverage: OK (template set == versions.json set)"
exit "$rc"
```

- [ ] **Step 5: Run the test to verify it passes (green)**

Run: `tools/check-version-coverage_test.sh`
Expected: ends with `ALL PASS`.

- [ ] **Step 6: Verify the guard is clean on the real tree**

Run: `tools/check-version-coverage.sh`
Expected: `check-version-coverage: OK (template set == versions.json set)`, exit 0. (Depends on Task 1 having wired v48/61/72/79.)

- [ ] **Step 7: shellcheck both new files**

Run: `shellcheck tools/check-version-coverage.sh tools/check-version-coverage_test.sh`
Expected: no output, exit 0.

- [ ] **Step 8: Commit**

```bash
git add tools/check-version-coverage.sh tools/check-version-coverage_test.sh
git commit -m "feat(tools): add version-port coverage guard (task-170)

Fails when the socket-config-template version set diverges from
deploy/k8s/base/versions.json — the blind spot that let PR #971 ship
gms v48/61/72/79 with no LB ports.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Wire the guard into CI

**Files:**
- Modify: `.github/workflows/pr-validation.yml` (add a step to the existing `gen-lb-ports` job)

**Interfaces:**
- Consumes: `tools/check-version-coverage.sh` from Task 2.
- Produces: no new job/summary wiring — the existing `LBPORTS_RESULT` summary row already reports this job's pass/fail.

- [ ] **Step 1: Add the guard step to the `gen-lb-ports` job**

In `.github/workflows/pr-validation.yml`, the `gen-lb-ports` job currently ends with:

```yaml
      - name: LB port manifests match versions.json
        run: ./tools/gen-lb-ports.sh --check
```

Append a second step immediately after it (same indentation, same `steps:` list):

```yaml
      - name: Version coverage matches templates
        run: ./tools/check-version-coverage.sh
```

- [ ] **Step 2: Verify YAML is well-formed and the step is in the right job**

Run:
```bash
python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/pr-validation.yml'))" && echo "YAML OK"
grep -n -A1 'Version coverage matches templates' .github/workflows/pr-validation.yml
```
Expected: `YAML OK`; the grep shows the new step name followed by its `run:` line. Confirm by eye that it sits inside the `gen-lb-ports:` job's `steps:` (after the existing `--check` step), not in another job.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/pr-validation.yml
git commit -m "ci: run version-port coverage guard in the gen-lb-ports job (task-170)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Final verification (after all tasks)

Run from the worktree root:

```bash
tools/gen-lb-ports.sh --check          # exit 0
tools/check-version-coverage.sh        # exit 0, "OK"
tools/gen-lb-ports_test.sh             # ALL PASS
tools/check-version-coverage_test.sh   # ALL PASS
shellcheck tools/check-version-coverage.sh tools/check-version-coverage_test.sh   # clean
git log --oneline -4                   # design + 3 task commits, branch task-170-version-port-guard
```

Then run code review (`superpowers:requesting-code-review`) before opening the PR.

## Notes

- No Go modules touched → the CLAUDE.md `docker buildx bake` / `go test` / redis- and goroutine-guard gates do not apply. Verification here is the shell test suites + shellcheck + the two guards running clean.
- The four legacy versions remain non-playable (WZ ingest, tenant provisioning, playthrough are the deferred Stage G/H/I follow-up). This task only exposes the ports and makes the next silent deferral impossible.
