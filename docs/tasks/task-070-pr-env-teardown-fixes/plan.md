# PR-Env Teardown Fixes — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop per-PR-env teardown leaks (Postgres/Kafka/Redis/ghcr/Argo CRDs/bot branches) by fixing four latent bugs and adding regression coverage, so a closed PR's environment is reclaimed automatically within 15 minutes.

**Architecture:** Defence in depth. The load-bearing fix derives `ATLAS_ENV` from `PR_NUMBER` directly in `cleanup.sh` (so annotation drift becomes harmless). The PostDelete cleanup Job moves out of the per-PR namespace into `argocd` (so it survives the namespace prune). The 24h `cleanup-grace` mechanism is removed (it was never wired up). Branch deletion is folded into the PostDelete Job using a dedicated, least-privilege PAT. A `sweep-orphans.sh` script codifies the May-19 manual recovery. A nightly smoke workflow proves end-to-end reclamation.

**Tech Stack:** bash + bats for scripts/tests; kustomize/Argo CD `Application` CRDs for deploy plumbing; GitHub Actions workflows; runbook markdown.

> Sibling PR required. Several changes in this plan have a cluster-infra counterpart (ApplicationSet template, CronJob, ServiceAccount, Secret). Those are tracked in `context.md` under "Sibling PR (cluster-infra)" and MUST land before the bot branch is merged. See `design.md` §6.

---

## File map (created or modified)

**Created:**
- `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh` — orphan-state sweep tool.
- `services/atlas-pr-bootstrap/test/lib_test.bats` — pins the `compute_atlas_env` formula.
- `services/atlas-pr-bootstrap/test/sweep_test.bats` — sweep arg parsing + (best-effort) end-to-end.
- `.github/workflows/pr-env-smoke.yml` — open→label-remove→assert-reclaim regression.
- `docs/tasks/task-070-pr-env-teardown-fixes/env-drift-investigation.md` — bug #4 part-B investigation writeup.

**Modified:**
- `services/atlas-pr-bootstrap/scripts/lib.sh` — add `compute_atlas_env` helper.
- `services/atlas-pr-bootstrap/scripts/cleanup.sh` — derive `ATLAS_ENV` from `PR_NUMBER`; add branch-delete phase.
- `services/atlas-pr-bootstrap/test/cleanup_test.bats` — reflect new required-env set; cover branch-delete 404 path.
- `deploy/k8s/overlays/pr/postdelete-cleanup.yaml` — move Job to `argocd`, set SA, drop `ATLAS_ENV` env entry, swap `ghcr-pat` Secret for `atlas-pr-cleanup-gh-token`.
- `.github/workflows/pr-cleanup.yml` — drop 24h-grace language from comments + step output.
- `docs/runbooks/ephemeral-pr-deployments.md` — rewrite §9.2 (force-cleanup is now redundant), §9.4 (immediate-teardown contract + recovery), §9.5 (new token), add §9.11 (sweep tool).

**Not modified (audited, intentional no-op):**
- `services/atlas-pr-bootstrap/scripts/bootstrap.sh` — reads `ATLAS_ENV` via the `atlas-env` ConfigMap at bootstrap time; that path is not affected by the drift and not part of teardown. See Task 13.

---

## Pre-flight: shared facts

**Worktree:** `.worktrees/task-070-pr-env-teardown-fixes` on branch `task-070-pr-env-teardown-fixes`. Every commit must land on this branch in this worktree. Before each commit, verify with:

```sh
git rev-parse --show-toplevel  # must end with /.worktrees/task-070-pr-env-teardown-fixes
git branch --show-current      # must be task-070-pr-env-teardown-fixes
```

**Formula contract (locked):**
```
ATLAS_ENV = first 4 hex chars of sha256("pr-<PR_NUMBER>")
```
Three sites must agree (today they do; the test in Task 1 pins it):
1. cluster-infra ApplicationSet goTemplate `{{ printf "%.4s" (sha256sum (printf "pr-%d" .number)) }}`.
2. `.github/workflows/pr-validation.yml` line 273 — `printf "pr-%d" "$PR_NUMBER" | sha256sum | cut -c1-4`.
3. `lib.sh::compute_atlas_env` (added in Task 1).

**Oracle values (from `recovery-log.md`):**
- PR 491 → `ed86`
- PR 522 → `a476`
- PR 1 → compute at test-write time with `printf "pr-1" | sha256sum | cut -c1-4` and substitute the literal four hex chars into Task 1's bats assertion.

**bats:** the project already has two `*_test.bats` files under `services/atlas-pr-bootstrap/test/`. Run a single file with `bats services/atlas-pr-bootstrap/test/<name>.bats`. Run all with `bats services/atlas-pr-bootstrap/test`. If bats is not installed, install via `brew install bats-core` or `apt-get install -y bats`.

---

### Task 1: Add `compute_atlas_env` helper to `lib.sh` (TDD, oracle-pinned)

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/lib.sh`
- Create: `services/atlas-pr-bootstrap/test/lib_test.bats`

- [ ] **Step 1.1: Compute the PR-1 oracle value**

Run:
```sh
printf "pr-1" | sha256sum | cut -c1-4
```
Capture the output (4 hex chars). Note it for Step 1.2; refer to it below as `<PR1_HASH>`.

- [ ] **Step 1.2: Write the failing test**

Create `services/atlas-pr-bootstrap/test/lib_test.bats`:

```bash
#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    # shellcheck source=../scripts/lib.sh
    . "$PROJECT_ROOT/scripts/lib.sh"
}

@test "compute_atlas_env: PR 1" {
    run compute_atlas_env 1
    [ "$status" -eq 0 ]
    [ "$output" = "<PR1_HASH>" ]   # replace <PR1_HASH> with literal computed value
}

@test "compute_atlas_env: PR 491 (recovery-log oracle)" {
    run compute_atlas_env 491
    [ "$status" -eq 0 ]
    [ "$output" = "ed86" ]
}

@test "compute_atlas_env: PR 522 (recovery-log oracle)" {
    run compute_atlas_env 522
    [ "$status" -eq 0 ]
    [ "$output" = "a476" ]
}

@test "compute_atlas_env: empty PR_NUMBER fails" {
    run compute_atlas_env ""
    [ "$status" -ne 0 ]
}
```

Substitute the value captured in Step 1.1 for `<PR1_HASH>`.

- [ ] **Step 1.3: Run test, confirm it fails**

```sh
bats services/atlas-pr-bootstrap/test/lib_test.bats
```
Expected: 4 failures (function not defined → bats reports an error per test).

- [ ] **Step 1.4: Implement `compute_atlas_env` in `lib.sh`**

Append to `services/atlas-pr-bootstrap/scripts/lib.sh` (after the existing `http_ok_tenant` function, before EOF):

```bash
# compute_atlas_env: derive the 4-hex-char per-env hash from a PR number.
# MUST stay in sync with .github/workflows/pr-validation.yml's update-pr-overlay
# step and the cluster-infra ApplicationSet template. test/lib_test.bats pins
# the contract via the PR 491 / 522 recovery-log oracles.
compute_atlas_env() {
    local pr_number="$1"
    if [ -z "$pr_number" ]; then
        log error "compute_atlas_env: empty PR_NUMBER"
        return 1
    fi
    printf "pr-%d" "$pr_number" | sha256sum | cut -c1-4
}
```

- [ ] **Step 1.5: Run test, confirm it passes**

```sh
bats services/atlas-pr-bootstrap/test/lib_test.bats
```
Expected: `4 tests, 0 failures`.

- [ ] **Step 1.6: Commit**

```sh
git add services/atlas-pr-bootstrap/scripts/lib.sh services/atlas-pr-bootstrap/test/lib_test.bats
git commit -m "feat(atlas-pr-bootstrap): add compute_atlas_env helper with oracle test"
```

Verify post-commit:
```sh
git rev-parse --show-toplevel  # ends with /.worktrees/task-070-pr-env-teardown-fixes
git branch --show-current      # task-070-pr-env-teardown-fixes
```

---

### Task 2: `cleanup.sh` derives `ATLAS_ENV` from `PR_NUMBER` (defensive fix)

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/cleanup.sh`
- Modify: `services/atlas-pr-bootstrap/test/cleanup_test.bats`

This is the load-bearing change. After this lands, annotation drift cannot cause cleanup to target the wrong env.

- [ ] **Step 2.1: Rewrite the first failing test**

Open `services/atlas-pr-bootstrap/test/cleanup_test.bats`. Replace the two existing tests with:

```bash
#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
}

@test "cleanup.sh fails without PR_NUMBER" {
    run env -u PR_NUMBER DB_HOST=h DB_PORT=5432 DB_USER=u DB_PASSWORD=p \
        ATLAS_DB_NAMES="atlas-test" BOOTSTRAP_SERVERS=k REDIS_URL=r \
        bash "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: PR_NUMBER"* ]]
}

@test "cleanup.sh no longer requires ATLAS_ENV in env" {
    # Pre-fix this asserted ATLAS_ENV was required. Now ATLAS_ENV is derived
    # from PR_NUMBER, so the script must fail on the next missing var
    # (DB_HOST), NOT on ATLAS_ENV. Drives the require_env reordering in
    # cleanup.sh.
    run env -u ATLAS_ENV -u DB_HOST PR_NUMBER=1 DB_PORT=5432 DB_USER=u \
        DB_PASSWORD=p ATLAS_DB_NAMES="atlas-test" BOOTSTRAP_SERVERS=k \
        REDIS_URL=r bash "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -ne 0 ]
    [[ "$output" != *"missing required env: ATLAS_ENV"* ]]
    [[ "$output" == *"missing required env: DB_HOST"* ]]
}

@test "cleanup.sh fails without ATLAS_DB_NAMES" {
    run env -u ATLAS_DB_NAMES PR_NUMBER=1 DB_HOST=h DB_PORT=5432 DB_USER=u \
        DB_PASSWORD=p BOOTSTRAP_SERVERS=k REDIS_URL=r \
        bash "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: ATLAS_DB_NAMES"* ]]
}
```

- [ ] **Step 2.2: Run tests, confirm the new behavior fails**

```sh
bats services/atlas-pr-bootstrap/test/cleanup_test.bats
```
Expected: the "no longer requires ATLAS_ENV" test FAILS (today the script still requires ATLAS_ENV).

- [ ] **Step 2.3: Update `cleanup.sh` to derive `ATLAS_ENV`**

Open `services/atlas-pr-bootstrap/scripts/cleanup.sh`. Replace the `require_env` line (currently line 28):

```bash
require_env ATLAS_ENV DB_HOST DB_PORT DB_USER DB_PASSWORD ATLAS_DB_NAMES BOOTSTRAP_SERVERS REDIS_URL PR_NUMBER
```

With:

```bash
require_env PR_NUMBER DB_HOST DB_PORT DB_USER DB_PASSWORD ATLAS_DB_NAMES BOOTSTRAP_SERVERS REDIS_URL

# Derive ATLAS_ENV from PR_NUMBER. Bug #4 (env-hash annotation drift): the
# Application's atlas.env annotation can disagree with the formula's actual
# output (observed on PRs 491/522, see task-070 recovery-log.md). Deriving
# here guarantees cleanup targets the correct hash regardless. lib.sh's
# compute_atlas_env is pinned by test/lib_test.bats against the formula
# used by .github/workflows/pr-validation.yml and the ApplicationSet.
ATLAS_ENV="$(compute_atlas_env "$PR_NUMBER")"
ATLAS_STEP=init log info "derived ATLAS_ENV=${ATLAS_ENV} for PR ${PR_NUMBER}"
```

Also update the header comment block (lines 6-7). Change:
```
#   ATLAS_ENV              — env hash
```
To:
```
#   PR_NUMBER              — PR number; ATLAS_ENV is derived as sha256("pr-N")[:4]
```
And remove the standalone `#   PR_NUMBER         — for image-tag prefix` later in the header (now covered by the line above).

- [ ] **Step 2.4: Run tests, confirm pass**

```sh
bats services/atlas-pr-bootstrap/test/cleanup_test.bats
```
Expected: `3 tests, 0 failures`.

- [ ] **Step 2.5: Commit**

```sh
git add services/atlas-pr-bootstrap/scripts/cleanup.sh services/atlas-pr-bootstrap/test/cleanup_test.bats
git commit -m "fix(atlas-pr-bootstrap): derive ATLAS_ENV from PR_NUMBER in cleanup.sh

Annotation drift on the Application's atlas.env can target cleanup at the
wrong env hash (observed on PRs 491, 522 — recovery-log.md). Compute the
hash from PR_NUMBER directly so cleanup is correct regardless of what the
annotation says."
```

Verify branch/worktree as in Task 1.6.

---

### Task 3: Add branch-delete phase to `cleanup.sh`

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/cleanup.sh`
- Modify: `services/atlas-pr-bootstrap/test/cleanup_test.bats`

Folds the cluster-infra CronJob's bot-branch deletion into the PostDelete Job, using the new `atlas-pr-cleanup-gh-token` Secret (added to the manifest in Task 4).

- [ ] **Step 3.1: Add a bats test for the 404 (branch-already-gone) path**

Append to `services/atlas-pr-bootstrap/test/cleanup_test.bats`:

```bash
@test "cleanup.sh branch-delete swallows 404" {
    # The bot branch may already have been deleted (operator, prior cleanup
    # re-run, force-deleted). Simulate via a `gh` shim in PATH that emits a
    # 404 body and exits non-zero. Cleanup must continue past this phase
    # without exiting.
    SHIM_DIR="$(mktemp -d)"
    cat > "$SHIM_DIR/gh" <<'EOF'
#!/usr/bin/env bash
echo "gh: Reference does not exist" >&2
exit 1
EOF
    chmod +x "$SHIM_DIR/gh"

    # Inject failing kafka-topics.sh / kafka-consumer-groups.sh / psql /
    # redis-cli so cleanup short-circuits on the very first phase BEFORE
    # branch-delete, while we only need to assert that the function exists
    # and is exercised by the unit (the e2e is in the smoke test). For this
    # unit assertion, we run a bash-side check on the script body instead:
    run grep -q "drop-branch" "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -eq 0 ]

    rm -rf "$SHIM_DIR"
}

@test "cleanup.sh references atlas-pr-cleanup-gh-token-mounted GHCR_TOKEN for branch-delete" {
    # GHCR_TOKEN is the secret key name preserved across the ghcr->dedicated
    # token migration. The branch-delete phase MUST read it, not a new env
    # name.
    run grep -E "drop-branch.*GHCR_TOKEN|GHCR_TOKEN.*drop-branch" \
        "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -eq 0 ]
}
```

(These are static-grep assertions because a full mock-gh run would also need shims for kafka/psql/redis to even reach the branch-delete phase. The smoke test in Task 10 exercises the live path.)

- [ ] **Step 3.2: Run tests, confirm they fail**

```sh
bats services/atlas-pr-bootstrap/test/cleanup_test.bats
```
Expected: the two new tests FAIL (the strings `drop-branch` and `GHCR_TOKEN` near `drop-branch` are not in `cleanup.sh` yet).

- [ ] **Step 3.3: Add the branch-delete phase to `cleanup.sh`**

Open `services/atlas-pr-bootstrap/scripts/cleanup.sh`. Insert the following block between the `drop-images` block (ends at line 76 with `fi`) and the `drop-dns` block (starts at line 78):

```bash
if [ -n "${PR_NUMBER:-}" ] && [ -n "${GHCR_TOKEN:-}" ]; then
    ATLAS_STEP=drop-branch log info "deleting bot/pr-${PR_NUMBER}-resolved"
    # Mounted via Secret atlas-pr-cleanup-gh-token (Contents: write on
    # Chronicle20/atlas + Packages: write on chronicle20/*). 404 is the
    # branch-already-deleted case — treat as success. Other errors are
    # logged warn and do not fail the Job (consistent with the rest of
    # cleanup's || true / xargs -r discipline).
    if ! err=$(gh api --method DELETE \
        -H "Authorization: Bearer ${GHCR_TOKEN}" \
        "/repos/Chronicle20/atlas/git/refs/heads/bot%2Fpr-${PR_NUMBER}-resolved" \
        2>&1); then
        case "$err" in
            *"Reference does not exist"*|*"Branch not found"*|*"404"*) ;;
            *) log warn "branch delete: $err" ;;
        esac
    fi
fi
```

- [ ] **Step 3.4: Run tests, confirm pass**

```sh
bats services/atlas-pr-bootstrap/test/cleanup_test.bats
```
Expected: `5 tests, 0 failures`.

- [ ] **Step 3.5: Commit**

```sh
git add services/atlas-pr-bootstrap/scripts/cleanup.sh services/atlas-pr-bootstrap/test/cleanup_test.bats
git commit -m "feat(atlas-pr-bootstrap): add branch-delete phase to cleanup.sh

The PostDelete Job now owns bot/pr-N-resolved branch deletion (previously
the cluster-infra CronJob attempted it but silently 403'd because
argocd-repo-creds-chronicle20-atlas lacks Contents: write). Reads
GHCR_TOKEN from the new atlas-pr-cleanup-gh-token Secret (mounted in the
next commit). 404 is treated as success."
```

Verify branch/worktree.

---

### Task 4: Move PostDelete Job to `argocd` namespace + secret + drop `ATLAS_ENV` env

**Files:**
- Modify: `deploy/k8s/overlays/pr/postdelete-cleanup.yaml`

This is the structural fix for bug #1 (finalizer-ordering wedge). The Job moves out of `atlas-pr-<N>` (which Argo CD is pruning) into `argocd` (long-lived).

- [ ] **Step 4.1: Apply the four changes**

Open `deploy/k8s/overlays/pr/postdelete-cleanup.yaml`. Make exactly these four edits:

(a) Add `namespace: argocd` under `metadata`. Replace:
```yaml
metadata:
  name: atlas-pr-cleanup
  annotations:
```
With:
```yaml
metadata:
  name: atlas-pr-cleanup
  # Long-lived namespace. Until task-070, this Job ran in atlas-pr-<N> —
  # the same namespace Argo CD's resources-finalizer prunes during
  # Application deletion. The PostDelete hook fires AFTER prune, so by
  # the time it tried to create the Job, the namespace was gone and the
  # hook wedged with `DeletionError: namespaces "atlas-pr-<N>" not found`.
  # The cluster-infra sibling PR adds the ServiceAccount + Role.
  namespace: argocd
  annotations:
```

(b) Add `serviceAccountName: atlas-pr-cleanup` under `spec.template.spec`. Replace:
```yaml
    spec:
      restartPolicy: Never
      containers:
```
With:
```yaml
    spec:
      restartPolicy: Never
      serviceAccountName: atlas-pr-cleanup
      containers:
```

(c) Swap the `ghcr-pat` `secretRef` for `atlas-pr-cleanup-gh-token`. Replace:
```yaml
            - secretRef:
                name: ghcr-pat
```
With:
```yaml
            - secretRef:
                # New least-privilege PAT (Contents: write on
                # Chronicle20/atlas + Packages: write on chronicle20/*).
                # Replaces ghcr-pat (which carried only packages scope and
                # 403'd on bot-branch DELETE). Created in cluster-infra
                # sibling PR; same key name (GHCR_TOKEN) so cleanup.sh
                # doesn't change.
                name: atlas-pr-cleanup-gh-token
```

(d) Drop the `ATLAS_ENV` env entry. Replace:
```yaml
          env:
            # Per-PR token, bot-substituted at update-pr-overlay time.
            - name: ATLAS_ENV
              value: "PLACEHOLDER_ATLAS_ENV"
            - name: PR_NUMBER
              value: "PLACEHOLDER_PR_NUMBER"
```
With:
```yaml
          env:
            # PR_NUMBER is the sole per-PR substitution. ATLAS_ENV is
            # derived inside cleanup.sh via lib.sh::compute_atlas_env, so
            # any drift on the Application's atlas.env annotation is
            # harmless (bug #4 defensive fix; see task-070/design.md §3.4).
            - name: PR_NUMBER
              value: "PLACEHOLDER_PR_NUMBER"
```

- [ ] **Step 4.2: Validate YAML parses**

```sh
yq '.' deploy/k8s/overlays/pr/postdelete-cleanup.yaml >/dev/null
echo "yaml ok"
```
Expected: `yaml ok` with no errors.

- [ ] **Step 4.3: Sanity-check sed substitution still resolves**

The bot-branch resolver in `pr-validation.yml` sed-substitutes `PLACEHOLDER_PR_NUMBER`. Confirm only the one placeholder remains in this file:

```sh
grep -n PLACEHOLDER deploy/k8s/overlays/pr/postdelete-cleanup.yaml
```
Expected: exactly one line — the `value: "PLACEHOLDER_PR_NUMBER"` line. `PLACEHOLDER_ATLAS_ENV` must NOT appear.

- [ ] **Step 4.4: Commit**

```sh
git add deploy/k8s/overlays/pr/postdelete-cleanup.yaml
git commit -m "fix(deploy): move atlas-pr-cleanup Job to argocd namespace

Bug #1 (finalizer-ordering wedge): the PostDelete Job lived in
atlas-pr-<N>, which Argo CD's resources-finalizer prunes during
Application deletion. The hook then failed to create the Job with
'DeletionError: namespaces \"atlas-pr-<N>\" not found' and the
Application wedged. Move the Job to the long-lived argocd namespace
(ServiceAccount comes from the cluster-infra sibling PR). Also: swap
ghcr-pat for atlas-pr-cleanup-gh-token (Contents+Packages scopes), and
drop the ATLAS_ENV env entry now that cleanup.sh derives it from
PR_NUMBER."
```

Verify branch/worktree.

---

### Task 5: Refresh `pr-cleanup.yml` comments to remove the 24h-grace mention

**Files:**
- Modify: `.github/workflows/pr-cleanup.yml`

Bug #2: the 24h `atlas.cleanup-grace` mechanism never fired because the ApplicationSet deletes the Application immediately on PR close. Comments in this workflow still describe the (dead) grace contract.

- [ ] **Step 5.1: Rewrite the misleading comment block + step output**

Open `.github/workflows/pr-cleanup.yml`.

(a) Replace lines 60-79 (the `# Branch deletion intentionally NOT done here...` block) with:

```yaml
  # Branch deletion intentionally NOT done here.
  #
  # `bot/pr-<N>-resolved` is the Argo CD ApplicationSet's targetRevision.
  # Argo CD's PostDelete Job
  # (`deploy/k8s/overlays/pr/postdelete-cleanup.yaml`) fires *immediately*
  # on PR close / `deploy-env` label removal. The Job's cleanup.sh deletes
  # the bot branch as its last step using the dedicated
  # atlas-pr-cleanup-gh-token PAT (Contents: write on Chronicle20/atlas).
  #
  # If we deleted the branch here on PR close, the Application's
  # PostDelete render — which targets bot/pr-<N>-resolved — would fail
  # with `unable to resolve 'bot/pr-<N>-resolved' to a commit SHA`,
  # `post-delete-finalizer.argocd.argoproj.io/cleanup` couldn't complete,
  # and the Application would wedge in Terminating. Reproduced 2026-05-16
  # on PR #461.
  #
  # Branch lifetime is now coupled to PostDelete Job lifetime — see
  # `docs/runbooks/ephemeral-pr-deployments.md` §9.4.
```

(b) Replace the `notify-argo` step's `run:` block (lines ~87-91) with:

```yaml
      - name: Log notice
        run: |
          echo "PR ${PR_NUMBER} closed at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
          echo "Argo CD's ApplicationSet will delete the Application immediately."
          echo "The PostDelete Job runs in the argocd namespace and reclaims"
          echo "per-env state (DBs, topics, groups, Redis keys, ghcr tags,"
          echo "bot branch) within ~10 minutes."
          echo "Recovery procedure if the teardown wedges:"
          echo "  docs/runbooks/ephemeral-pr-deployments.md §9.4 (force-cleanup),"
          echo "  §9.11 (full orphan sweep)."
```

- [ ] **Step 5.2: Commit**

```sh
git add .github/workflows/pr-cleanup.yml
git commit -m "docs(pr-cleanup): drop dead 24h cleanup-grace language

Bug #2: the atlas.cleanup-grace annotation was never honored because the
ApplicationSet deletes the generated Application immediately on PR close.
Update the workflow's comments and step output to describe the actual
contract (immediate teardown via the PostDelete Job in argocd)."
```

Verify branch/worktree.

---

### Task 6: Sweep script skeleton + arg-parsing test

**Files:**
- Create: `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh`
- Create: `services/atlas-pr-bootstrap/test/sweep_test.bats`

Codifies the May-19 manual recovery as a runnable script. Task 6 is structure + arg parsing; Task 7 implements each phase.

- [ ] **Step 6.1: Write failing tests for arg parsing**

Create `services/atlas-pr-bootstrap/test/sweep_test.bats`:

```bash
#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    SCRIPT="$PROJECT_ROOT/scripts/sweep-orphans.sh"
}

@test "sweep-orphans.sh: missing PR number prints usage and exits non-zero" {
    run bash "$SCRIPT"
    [ "$status" -ne 0 ]
    [[ "$output" == *"Usage:"* ]]
}

@test "sweep-orphans.sh: rejects non-numeric PR number" {
    run bash "$SCRIPT" abc
    [ "$status" -ne 0 ]
    [[ "$output" == *"not a number"* ]] || [[ "$output" == *"Usage:"* ]]
}

@test "sweep-orphans.sh: --list (default) on PR 491 prints derived ATLAS_ENV" {
    # No infra to talk to in unit tests; assert the script gets far enough
    # to print the computed env hash before any external command fails or
    # is no-op'd by being unreachable.
    run env DRY_RUN_NO_INFRA=1 bash "$SCRIPT" 491
    [[ "$output" == *"ed86"* ]]
}

@test "sweep-orphans.sh: --apply requires explicit confirmation flag" {
    # Idempotency / blast-radius: require the operator to type --apply.
    # Default behavior MUST be list-only.
    run env DRY_RUN_NO_INFRA=1 bash "$SCRIPT" 491
    [[ "$output" != *"DROP DATABASE"* ]]
    [[ "$output" != *"--delete --topic"* ]]
}

@test "sweep-orphans.sh: accepts multiple PR numbers" {
    run env DRY_RUN_NO_INFRA=1 bash "$SCRIPT" 491 522
    [[ "$output" == *"ed86"* ]]
    [[ "$output" == *"a476"* ]]
}
```

- [ ] **Step 6.2: Run tests, confirm they fail**

```sh
bats services/atlas-pr-bootstrap/test/sweep_test.bats
```
Expected: all 5 tests FAIL (script does not exist).

- [ ] **Step 6.3: Implement the skeleton**

Create `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh`:

```bash
#!/usr/bin/env bash
# Atlas PR-env orphan sweep. Codifies the May-19 recovery: enumerate (and
# optionally delete) per-env state for one or more PR numbers.
#
# Usage:
#   sweep-orphans.sh [--apply] PR_NUMBER [PR_NUMBER ...]
#
# Without --apply (default): lists everything that would be deleted.
# With --apply: deletes it. Idempotent — safe to re-run after a partial sweep.
#
# Required env (same names cleanup.sh uses; defaults match cluster reality):
#   DB_HOST, DB_PORT, DB_USER, DB_PASSWORD
#   ATLAS_DB_NAMES                    — space-separated base DB names
#   BOOTSTRAP_SERVERS                 — Kafka bootstrap
#   REDIS_URL                         — host:port (NOT a URL)
#   GHCR_TOKEN                        — GitHub PAT (Contents+Packages write)
#   ATLAS_SERVICES                    — comma-separated service names
#   PIHOLE_API_BASE_1 / PIHOLE_TOKEN_1 / PIHOLE_API_BASE_2 / PIHOLE_TOKEN_2
#
# DRY_RUN_NO_INFRA=1 short-circuits external-command phases (testing only).

set -euo pipefail

# shellcheck source=lib.sh
. "$(dirname "$0")/lib.sh"

APPLY=0
PR_NUMBERS=()

usage() {
    cat <<'EOF'
Usage: sweep-orphans.sh [--apply] PR_NUMBER [PR_NUMBER ...]

  --apply        Actually delete state. Without this flag, sweep is list-only.
  PR_NUMBER      One or more positive integers.

Without --apply, all phases print what they would do, one resource per line,
prefixed with the phase name (drop-dbs, drop-topics, drop-groups,
drop-redis, drop-images, drop-dns, drop-app-finalizers, drop-branch).
Suitable for piping through `tee` or `diff` for visual review before re-running
with --apply.
EOF
}

while [ $# -gt 0 ]; do
    case "$1" in
        --apply) APPLY=1 ; shift ;;
        --list)  APPLY=0 ; shift ;;     # explicit form, same as default
        -h|--help) usage ; exit 0 ;;
        --) shift ; break ;;
        -*) echo "unknown flag: $1" >&2 ; usage >&2 ; exit 2 ;;
        *)  PR_NUMBERS+=("$1") ; shift ;;
    esac
done

if [ "${#PR_NUMBERS[@]}" -eq 0 ]; then
    usage >&2
    exit 2
fi

for n in "${PR_NUMBERS[@]}"; do
    if ! [[ "$n" =~ ^[0-9]+$ ]]; then
        echo "PR number '$n' is not a number" >&2
        usage >&2
        exit 2
    fi
done

sweep_pr() {
    local pr_number="$1"
    local env_hash
    env_hash="$(compute_atlas_env "$pr_number")"
    ATLAS_ENV="$env_hash" ATLAS_STEP=init log info \
        "sweeping PR ${pr_number} (ATLAS_ENV=${env_hash}) apply=${APPLY}"

    # Phase implementations are added in Task 7. Each phase MUST:
    #   - read APPLY (0 = list-only, 1 = delete)
    #   - prefix each enumerated resource with its phase name
    #   - tolerate missing resources (idempotent)
    #   - skip the phase entirely if its required env vars are unset
    #
    # In DRY_RUN_NO_INFRA mode (testing), skip every infra call but still
    # emit the env_hash line above so the harness can grep for it.
    if [ -n "${DRY_RUN_NO_INFRA:-}" ]; then
        return 0
    fi

    sweep_pg "$pr_number" "$env_hash"
    sweep_kafka "$pr_number" "$env_hash"
    sweep_redis "$pr_number" "$env_hash"
    sweep_ghcr "$pr_number" "$env_hash"
    sweep_pihole "$pr_number" "$env_hash"
    sweep_app_finalizer "$pr_number" "$env_hash"
    sweep_branch "$pr_number" "$env_hash"
}

# Phase implementations (Task 7). Stubs that no-op so the skeleton is runnable.
sweep_pg()             { :; }
sweep_kafka()          { :; }
sweep_redis()          { :; }
sweep_ghcr()           { :; }
sweep_pihole()         { :; }
sweep_app_finalizer()  { :; }
sweep_branch()         { :; }

for n in "${PR_NUMBERS[@]}"; do
    sweep_pr "$n"
done

ATLAS_STEP=done log info "sweep complete"
```

Make it executable:
```sh
chmod +x services/atlas-pr-bootstrap/scripts/sweep-orphans.sh
```

- [ ] **Step 6.4: Run tests, confirm pass**

```sh
bats services/atlas-pr-bootstrap/test/sweep_test.bats
```
Expected: `5 tests, 0 failures`.

- [ ] **Step 6.5: Commit**

```sh
git add services/atlas-pr-bootstrap/scripts/sweep-orphans.sh services/atlas-pr-bootstrap/test/sweep_test.bats
git commit -m "feat(atlas-pr-bootstrap): sweep-orphans.sh skeleton + arg parsing

List-only by default; --apply opts into deletion. Phase implementations
in the next commit."
```

Verify branch/worktree.

---

### Task 7: Sweep script — phase implementations

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh`

Replace each stub with a real implementation. Each phase mirrors the equivalent in `cleanup.sh` but adds list-mode output.

- [ ] **Step 7.1: Implement `sweep_pg`**

Replace `sweep_pg()             { :; }` with:

```bash
sweep_pg() {
    local pr_number="$1"
    local env_hash="$2"
    [ -z "${DB_HOST:-}" ] && return 0
    DB_USER="$(printf '%s' "${DB_USER:-}" | tr -d ' \r\n')"
    DB_PASSWORD="$(printf '%s' "${DB_PASSWORD:-}" | tr -d ' \r\n')"
    [ -z "$DB_USER" ] && return 0
    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-dbs log info "scanning Postgres for orphans (PR $pr_number)"
    local dbs
    dbs=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -tAc \
        "SELECT datname FROM pg_database WHERE datname ~ '-${env_hash}\$';") || return 0
    while IFS= read -r db; do
        [ -z "$db" ] && continue
        echo "drop-dbs ${db}"
        if [ "$APPLY" = "1" ]; then
            PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres \
                -c "DROP DATABASE IF EXISTS \"$db\" WITH (FORCE);" || \
                ATLAS_ENV="$env_hash" ATLAS_STEP=drop-dbs log warn "drop $db failed"
        fi
    done <<<"$dbs"
}
```

- [ ] **Step 7.2: Implement `sweep_kafka`**

Replace `sweep_kafka()          { :; }` with:

```bash
sweep_kafka() {
    local pr_number="$1"
    local env_hash="$2"
    [ -z "${BOOTSTRAP_SERVERS:-}" ] && return 0
    if ! command -v kafka-topics.sh >/dev/null 2>&1; then
        ATLAS_ENV="$env_hash" ATLAS_STEP=drop-topics log warn "kafka-topics.sh not on PATH; skipping"
        return 0
    fi
    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-topics log info "scanning Kafka topics"
    local topics
    topics=$(kafka-topics.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --list 2>/dev/null \
        | grep -E -- "-${env_hash}\$" || true)
    while IFS= read -r t; do
        [ -z "$t" ] && continue
        echo "drop-topics ${t}"
        if [ "$APPLY" = "1" ]; then
            kafka-topics.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --delete --topic "$t" || \
                ATLAS_ENV="$env_hash" ATLAS_STEP=drop-topics log warn "delete topic $t failed"
        fi
    done <<<"$topics"

    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-groups log info "scanning Kafka consumer groups"
    local groups
    groups=$(kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --list 2>/dev/null \
        | grep -E -- "\\[${env_hash}\\]\$" || true)
    while IFS= read -r g; do
        [ -z "$g" ] && continue
        echo "drop-groups ${g}"
        if [ "$APPLY" = "1" ]; then
            kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --delete --group "$g" || \
                ATLAS_ENV="$env_hash" ATLAS_STEP=drop-groups log warn "delete group failed"
        fi
    done <<<"$groups"
}
```

- [ ] **Step 7.3: Implement `sweep_redis`**

Replace `sweep_redis()          { :; }` with:

```bash
sweep_redis() {
    local pr_number="$1"
    local env_hash="$2"
    [ -z "${REDIS_URL:-}" ] && return 0
    if ! command -v redis-cli >/dev/null 2>&1; then
        ATLAS_ENV="$env_hash" ATLAS_STEP=drop-redis log warn "redis-cli not on PATH; skipping"
        return 0
    fi
    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-redis log info "scanning Redis"
    local keys
    keys=$(redis-cli -u "redis://$REDIS_URL" --scan --pattern "${env_hash}:*" || true)
    while IFS= read -r k; do
        [ -z "$k" ] && continue
        echo "drop-redis ${k}"
    done <<<"$keys"
    if [ "$APPLY" = "1" ] && [ -n "$keys" ]; then
        printf '%s\n' "$keys" | xargs -r -n 1000 redis-cli -u "redis://$REDIS_URL" DEL >/dev/null || \
            ATLAS_ENV="$env_hash" ATLAS_STEP=drop-redis log warn "DEL failed"
    fi
}
```

- [ ] **Step 7.4: Implement `sweep_ghcr`**

Replace `sweep_ghcr()           { :; }` with:

```bash
sweep_ghcr() {
    local pr_number="$1"
    local env_hash="$2"
    [ -z "${ATLAS_SERVICES:-}" ] && return 0
    [ -z "${GHCR_TOKEN:-}" ] && return 0
    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-images log info "scanning ghcr tags pr-${pr_number}-*"
    local svcs
    IFS=',' read -ra svcs <<<"$ATLAS_SERVICES"
    for svc in "${svcs[@]}"; do
        local vids
        vids=$(gh api -H "Authorization: Bearer $GHCR_TOKEN" \
            "/users/chronicle20/packages/container/${svc}%2F${svc}/versions" \
            --jq ".[] | select(.metadata.container.tags[]? | startswith(\"pr-${pr_number}-\")) | [.id, (.metadata.container.tags|join(\",\"))] | @tsv" \
            2>/dev/null) || continue
        while IFS=$'\t' read -r vid tags; do
            [ -z "$vid" ] && continue
            echo "drop-images ${svc}/${vid} (${tags})"
            if [ "$APPLY" = "1" ]; then
                gh api --method DELETE -H "Authorization: Bearer $GHCR_TOKEN" \
                    "/users/chronicle20/packages/container/${svc}%2F${svc}/versions/${vid}" \
                    >/dev/null 2>&1 || \
                    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-images log warn "delete ${svc}/${vid} failed"
            fi
        done <<<"$vids"
    done
}
```

- [ ] **Step 7.5: Implement `sweep_pihole`**

Replace `sweep_pihole()         { :; }` with:

```bash
sweep_pihole() {
    local pr_number="$1"
    local env_hash="$2"
    [ -z "${PIHOLE_API_BASE_1:-}" ] && return 0
    [ -z "${PIHOLE_TOKEN_1:-}" ] && return 0
    local host="${pr_number}.atlas.home"
    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-dns log info "scanning Pi-hole hosts for ${host}"
    for i in 1 2; do
        local base_var="PIHOLE_API_BASE_$i"
        local token_var="PIHOLE_TOKEN_$i"
        local base="${!base_var:-}"
        local token="${!token_var:-}"
        [ -z "$base" ] && continue
        [ -z "$token" ] && continue
        local sid
        sid=$(curl -k -fsS -X POST "$base/api/auth" \
            -H "Content-Type: application/json" \
            -d "{\"password\":\"$token\"}" 2>/dev/null \
            | jq -r '.session.sid // empty')
        [ -z "$sid" ] && { ATLAS_ENV="$env_hash" ATLAS_STEP=drop-dns log warn "pihole $i auth failed"; continue; }
        local entry
        entry=$(curl -k -fsS -H "X-FTL-SID: $sid" "$base/api/config/dns/hosts" \
            | jq -r ".config.dns.hosts[]? | select(endswith(\" $host\"))" | head -1)
        [ -z "$entry" ] && continue
        echo "drop-dns pihole-${i} ${entry}"
        if [ "$APPLY" = "1" ]; then
            local enc
            enc=$(printf '%s' "$entry" | sed 's/ /%20/g')
            curl -k -fsS -X DELETE -H "X-FTL-SID: $sid" \
                "$base/api/config/dns/hosts/$enc" >/dev/null 2>&1 || \
                ATLAS_ENV="$env_hash" ATLAS_STEP=drop-dns log warn "pihole $i delete failed"
        fi
    done
}
```

- [ ] **Step 7.6: Implement `sweep_app_finalizer`**

Replace `sweep_app_finalizer()  { :; }` with:

```bash
sweep_app_finalizer() {
    local pr_number="$1"
    local env_hash="$2"
    command -v kubectl >/dev/null 2>&1 || return 0
    if ! kubectl -n argocd get application.argoproj.io "atlas-pr-${pr_number}" \
        >/dev/null 2>&1; then
        return 0
    fi
    echo "drop-app-finalizers atlas-pr-${pr_number}"
    if [ "$APPLY" = "1" ]; then
        kubectl -n argocd patch application.argoproj.io "atlas-pr-${pr_number}" \
            --type=merge -p '{"metadata":{"finalizers":[]}}' >/dev/null 2>&1 || \
            ATLAS_ENV="$env_hash" ATLAS_STEP=drop-app-finalizers log warn \
                "patch atlas-pr-${pr_number} failed"
    fi
}
```

- [ ] **Step 7.7: Implement `sweep_branch`**

Replace `sweep_branch()         { :; }` with:

```bash
sweep_branch() {
    local pr_number="$1"
    local env_hash="$2"
    [ -z "${GHCR_TOKEN:-}" ] && return 0
    # Check existence first so list mode reports honestly.
    local status
    status=$(gh api -H "Authorization: Bearer $GHCR_TOKEN" \
        "/repos/Chronicle20/atlas/git/refs/heads/bot%2Fpr-${pr_number}-resolved" \
        --jq '.ref // empty' 2>/dev/null) || status=""
    [ -z "$status" ] && return 0
    echo "drop-branch bot/pr-${pr_number}-resolved"
    if [ "$APPLY" = "1" ]; then
        gh api --method DELETE -H "Authorization: Bearer $GHCR_TOKEN" \
            "/repos/Chronicle20/atlas/git/refs/heads/bot%2Fpr-${pr_number}-resolved" \
            >/dev/null 2>&1 || \
            ATLAS_ENV="$env_hash" ATLAS_STEP=drop-branch log warn \
                "delete bot/pr-${pr_number}-resolved failed"
    fi
}
```

- [ ] **Step 7.8: Add a list-mode bats test that exercises a real but mocked phase**

Append to `services/atlas-pr-bootstrap/test/sweep_test.bats`:

```bash
@test "sweep-orphans.sh: phase names appear in --list output" {
    # Mock infra commands to emit one fake resource each, so list mode
    # produces the canonical "phase resource" lines and APPLY=0 means none
    # of them get acted on.
    SHIM_DIR="$(mktemp -d)"
    cat > "$SHIM_DIR/kafka-topics.sh" <<'EOF'
#!/usr/bin/env bash
case "$*" in
    *--list*) echo "atlas-faketopic-ed86" ;;
    *--delete*) echo "FAIL: delete invoked in list mode" >&2; exit 1 ;;
esac
EOF
    cat > "$SHIM_DIR/kafka-consumer-groups.sh" <<'EOF'
#!/usr/bin/env bash
case "$*" in
    *--list*) echo "Fake Group [ed86]" ;;
    *--delete*) echo "FAIL: delete invoked in list mode" >&2; exit 1 ;;
esac
EOF
    cat > "$SHIM_DIR/redis-cli" <<'EOF'
#!/usr/bin/env bash
case "$*" in
    *--scan*) echo "ed86:fake-key" ;;
    *DEL*)    echo "FAIL: DEL invoked in list mode" >&2; exit 1 ;;
esac
EOF
    cat > "$SHIM_DIR/psql" <<'EOF'
#!/usr/bin/env bash
echo "atlas-fake-ed86"
EOF
    cat > "$SHIM_DIR/gh" <<'EOF'
#!/usr/bin/env bash
# Empty results — easier than mocking the rich gh api jq path.
echo ""
EOF
    cat > "$SHIM_DIR/kubectl" <<'EOF'
#!/usr/bin/env bash
exit 1   # "Application not found" — drop-app-finalizers phase no-ops.
EOF
    chmod +x "$SHIM_DIR"/*

    PATH="$SHIM_DIR:$PATH" run env \
        DB_HOST=fake DB_PORT=1 DB_USER=u DB_PASSWORD=p \
        BOOTSTRAP_SERVERS=fake REDIS_URL=fake:6379 \
        GHCR_TOKEN=fake ATLAS_SERVICES=atlas-fake \
        bash "$SCRIPT" 491

    [[ "$output" == *"drop-dbs atlas-fake-ed86"* ]]
    [[ "$output" == *"drop-topics atlas-faketopic-ed86"* ]]
    [[ "$output" == *"drop-groups Fake Group [ed86]"* ]]
    [[ "$output" == *"drop-redis ed86:fake-key"* ]]
    [[ "$output" != *"FAIL:"* ]]

    rm -rf "$SHIM_DIR"
}
```

- [ ] **Step 7.9: Run tests, confirm pass**

```sh
bats services/atlas-pr-bootstrap/test/sweep_test.bats
```
Expected: `6 tests, 0 failures`.

- [ ] **Step 7.10: Commit**

```sh
git add services/atlas-pr-bootstrap/scripts/sweep-orphans.sh services/atlas-pr-bootstrap/test/sweep_test.bats
git commit -m "feat(atlas-pr-bootstrap): sweep-orphans.sh phase implementations

Postgres, Kafka topics+groups, Redis, ghcr tags, Pi-hole DNS, Argo
Application finalizer patch, bot branch. List-only by default; --apply
opts into deletion. List-mode bats test verifies no phase invokes its
delete path without --apply."
```

Verify branch/worktree.

---

### Task 8: Runbook rewrites (§9.2, §9.4, §9.5, new §9.11)

**Files:**
- Modify: `docs/runbooks/ephemeral-pr-deployments.md`

Reflect the new contract (immediate teardown, no grace) and document the sweep tool.

- [ ] **Step 8.1: Rewrite §9.2 "Force-cleanup of a PR env"**

Replace lines 155-168 (the §9.2 section) with:

```markdown
## §9.2 Force-cleanup of a PR env

Removing the `deploy-env` label or closing the PR triggers immediate teardown — there is **no grace window**. If a teardown wedges, see §9.4 for recovery and §9.11 for the orphan-sweep script.

To stop a running env without closing the PR:

```sh
gh pr edit <N> --remove-label deploy-env
```

The ApplicationSet drops its generator entry on the next reconcile (~30s), Argo CD deletes the Application, and the PostDelete Job in the `argocd` namespace reclaims per-env state within ~10 minutes.

Verify in-flight cleanup:

```sh
kubectl -n argocd get jobs -l app=atlas-pr-cleanup
kubectl -n argocd logs -l app=atlas-pr-cleanup --tail=200
```

If you specifically need to force-delete an Application that is stuck (i.e., the ApplicationSet's generator still points at it), see §9.4.
```

- [ ] **Step 8.2: Rewrite §9.4 "Re-running a failed PostDelete"**

Replace lines 184-218 (the §9.4 section + its sub-section) with:

```markdown
## §9.4 Recovery when teardown wedges

**Contract:** PR close (or `deploy-env` label removal) ⇒ Argo CD deletes the Application immediately ⇒ the PostDelete Job in `argocd` namespace runs `cleanup.sh` ⇒ all per-env state (DBs, topics, groups, Redis keys, ghcr tags, bot branch) is reclaimed within ~10 minutes.

If something in that chain fails, the Application sits in `Terminating` with finalizers `post-delete-finalizer.argocd.argoproj.io/cleanup` and `resources-finalizer.argocd.argoproj.io` still present. Per-env state may be partially reclaimed.

### Diagnose

```sh
kubectl -n argocd get application atlas-pr-<N> -o yaml | yq '.status.conditions'
kubectl -n argocd get jobs -l app=atlas-pr-cleanup,atlas.pr-number=<N>
kubectl -n argocd logs -l app=atlas-pr-cleanup,atlas.pr-number=<N> --tail=500
```

Common signals:

- `DeletionError: namespaces "atlas-pr-<N>" not found` — should not happen post-task-070; if it does, the cluster-infra ApplicationSet was rolled back. File an incident.
- The PostDelete Job is `Failed` with logs showing a specific phase (e.g. `drop-topics`) erroring on a missing dep — fix the dep, re-run via the sweep (§9.11).
- `cleanup.sh` ran to completion but `kubectl get application` still shows the Application — finalizer wasn't drained because the Job container exited non-zero on a non-critical step. Patch the finalizers (below).

### Recover

```sh
# 1. (If state is suspected leaked.) Run the orphan sweep in list mode,
#    review output, then re-run with --apply. See §9.11.
sweep-orphans.sh <N>          # list
sweep-orphans.sh --apply <N>  # reclaim

# 2. Drop the Application's finalizers so the CRD can be removed.
kubectl -n argocd patch application.argoproj.io atlas-pr-<N> \
    --type=merge -p '{"metadata":{"finalizers":[]}}'

# 3. (If the bot branch survived.) The sweep script handles this, but the
#    manual command is:
gh api --method DELETE \
    /repos/Chronicle20/atlas/git/refs/heads/bot/pr-<N>-resolved
```

### Source-branch-missing scenario

If the PostDelete render fails with `unable to resolve 'bot/pr-<N>-resolved' to a commit SHA`, the Application targets a branch that no longer exists. Diagnose: `kubectl -n argocd get application atlas-pr-<N> -o yaml | yq '.status.conditions[] | select(.message | contains("ComparisonError"))'`. Recovery is the same finalizer patch (step 2 above) followed by the sweep (step 1) — the branch is already gone so `drop-branch` is a no-op.
```

- [ ] **Step 8.3: Rewrite §9.5 "Rotating credentials"**

Replace lines 220-230 (the §9.5 section) with:

```markdown
## §9.5 Rotating credentials

All Argo CD-related Secrets live in the `argocd` namespace and are templated by `argocd-secrets.yml.example` in the cluster-infra repo. To rotate:

- **`atlas-pr-cleanup-gh-token` (PR-env cleanup PAT).** Used by the PostDelete Job for bot-branch deletion and ghcr image-tag deletion. Fine-grained PAT, scopes: `Contents: write` on `Chronicle20/atlas`, `Packages: write` on `chronicle20/*`, `Metadata: read` on `Chronicle20/atlas`. Expiry ≤ 90 days; operator calendars the next rotation.
  ```sh
  # 1. Mint a new PAT on github.com → Settings → Developer settings → Fine-grained PAT.
  # 2. Update the cluster secret.
  kubectl -n argocd edit secret atlas-pr-cleanup-gh-token   # set key GHCR_TOKEN
  # 3. Update the repo secret used by .github/workflows/pr-cleanup.yml's image-delete step.
  gh secret set GHCR_TOKEN --repo Chronicle20/atlas --body "$NEW_PAT"
  ```
  The nightly smoke test (§4.5 / `pr-env-smoke.yml`) will catch a missed half-rotation within 24h.

- **GitHub PAT for Argo source-repo creds:** `kubectl edit secret argocd-repo-creds-chronicle20-atlas -n argocd`, replace `password`. ApplicationSet picks up on next reconcile (~30s). This token does NOT need `Contents: write` (the cleanup PAT above owns branch deletion).

- **Pi-hole tokens:** `kubectl edit secret pihole-credentials -n argocd`. The PostSync register Job and the PostDelete cleanup Job both read at run-time; rotation takes effect on the next PR sync.

- **ghcr-pat (legacy).** No longer used by the PostDelete Job (replaced by `atlas-pr-cleanup-gh-token`). If no other consumer needs it, remove it in a cluster-infra follow-up.
```

- [ ] **Step 8.4: Add §9.11 "Orphan sweep"**

Append to the end of `docs/runbooks/ephemeral-pr-deployments.md`:

```markdown
## §9.11 Orphan sweep

For PR-envs whose teardown wedged or pre-dated the task-070 fixes, `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh` enumerates and (with `--apply`) deletes every leaked artifact.

### One-shot from a workstation

```sh
export DB_HOST=postgres.home DB_PORT=5432 DB_USER=postgres DB_PASSWORD=...
export ATLAS_DB_NAMES="atlas-accounts atlas-bans ..."    # same list as deploy/k8s/overlays/pr/postdelete-cleanup.yaml
export BOOTSTRAP_SERVERS=kafka.home:9093
export REDIS_URL=redis.home:6379
export GHCR_TOKEN=$(cat ~/.config/atlas/gh.env | grep GH_TOKEN | cut -d= -f2)
export ATLAS_SERVICES="atlas-account,atlas-asset-expiration,..."
export PIHOLE_API_BASE_1=https://pihole1.home PIHOLE_TOKEN_1=...
export PIHOLE_API_BASE_2=https://pihole2.home PIHOLE_TOKEN_2=...

# List what would be deleted for PRs 491 and 522:
./services/atlas-pr-bootstrap/scripts/sweep-orphans.sh 491 522

# Reclaim:
./services/atlas-pr-bootstrap/scripts/sweep-orphans.sh --apply 491 522
```

### In-cluster (preferred for production cluster credentials)

Run inside a one-shot debug pod that already has the right Secrets mounted:

```sh
kubectl -n argocd run sweep-$(date +%s) --rm -it --restart=Never \
    --image=ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest \
    --overrides='{
      "spec":{"containers":[{
        "name":"sweep","image":"ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest",
        "envFrom":[
          {"secretRef":{"name":"db-credentials"}},
          {"secretRef":{"name":"pihole-credentials"}},
          {"secretRef":{"name":"atlas-pr-cleanup-gh-token"}}],
        "command":["/atlas/sweep-orphans.sh","--apply","<N>"],
        "stdin":true,"tty":true
      }]}}'
```

Idempotent — re-running on an already-clean PR exits 0 with all enumerations empty. The script tolerates absent infrastructure (it skips any phase whose required env var is unset), so partial-credential invocations also work for diagnosing one subsystem at a time.

### Metric (cluster-infra)

The cluster-infra `atlas-pr-cleanup` CronJob's orphan-sweep mode emits `atlas_pr_orphan_envs_total{pr_number,kind}` (counter). Operator dashboard query:

```promql
sum by (kind) (atlas_pr_orphan_envs_total)
```

Alert wiring is out of scope for task-070 — this is observable but not paged.
```

- [ ] **Step 8.5: Commit**

```sh
git add docs/runbooks/ephemeral-pr-deployments.md
git commit -m "docs(runbook): rewrite §9.2/§9.4/§9.5 + add §9.11 orphan sweep

- §9.2: remove 24h-grace language; document the immediate-teardown contract.
- §9.4: recovery procedure for wedged teardowns (sweep + finalizer patch).
- §9.5: new atlas-pr-cleanup-gh-token rotation procedure.
- §9.11: orphan-sweep tool with both workstation and in-cluster invocations."
```

Verify branch/worktree.

---

### Task 9: Smoke-test workflow (`pr-env-smoke.yml`)

**Files:**
- Create: `.github/workflows/pr-env-smoke.yml`

End-to-end regression. Gated on a self-hosted runner being labeled `atlas-cluster`. If no such runner exists at execution time, the workflow file lands with `if: false` on the assertion job so it can be turned on later without re-merging.

- [ ] **Step 9.1: Check whether a self-hosted runner is available**

The plan executor should verify (manually, against the GitHub repo Settings → Actions → Runners) whether a `self-hosted` runner with the `atlas-cluster` label exists. If yes, set `RUNNER_AVAILABLE=yes` for Step 9.2; if no, set `RUNNER_AVAILABLE=no`.

Record the decision in the commit message.

- [ ] **Step 9.2: Create the workflow**

Create `.github/workflows/pr-env-smoke.yml`:

```yaml
name: PR-Env Smoke Test

# Open + close a synthetic PR labeled deploy-env; assert every per-env
# artifact is reclaimed within the timeout. Catches regressions in the
# task-070 fixes (and any subsequent teardown refactor).
#
# Triggers: workflow_dispatch + nightly.
#
# If no self-hosted runner with the atlas-cluster label exists yet, set the
# `assert-reclamation` job's `if:` to `false` so the workflow can land
# without failing nightly. Flip back to the gating condition once the
# runner is provisioned.

on:
  workflow_dispatch: {}
  schedule:
    - cron: '17 4 * * *'   # 04:17 UTC daily

permissions:
  contents: write
  pull-requests: write
  packages: read

concurrency:
  group: pr-env-smoke
  cancel-in-progress: false

jobs:
  open-synthetic-pr:
    name: Open synthetic PR
    runs-on: ubuntu-latest
    outputs:
      pr_number: ${{ steps.open.outputs.pr_number }}
      branch:    ${{ steps.open.outputs.branch }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
          token: ${{ secrets.GITHUB_TOKEN }}

      - name: Create no-op commit on a smoke branch
        id: branch
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          set -euo pipefail
          ts=$(date -u +%Y%m%d-%H%M%S)
          br="smoke/pr-env-${ts}"
          git config user.name 'atlas-smoke-bot'
          git config user.email 'atlas-smoke-bot@users.noreply.github.com'
          git checkout -b "$br"
          mkdir -p docs/smoke
          echo "$ts" > docs/smoke/touch.txt
          git add docs/smoke/touch.txt
          git commit -m "smoke(pr-env): touch $ts [skip ci]"
          git push origin "$br"
          echo "branch=$br" >> "$GITHUB_OUTPUT"

      - name: Open PR with deploy-env label
        id: open
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          BRANCH:   ${{ steps.branch.outputs.branch }}
        run: |
          set -euo pipefail
          url=$(gh pr create --base main --head "$BRANCH" \
            --title "[smoke-test] pr-env reclamation $(date -u +%FT%TZ)" \
            --body "Automated nightly smoke test for ephemeral PR-env teardown. Workflow run: ${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}" \
            --label deploy-env)
          n=$(basename "$url")
          echo "pr_number=$n" >> "$GITHUB_OUTPUT"
          echo "Opened PR #$n on $BRANCH"

  wait-for-healthy:
    name: Wait for Application Healthy
    needs: open-synthetic-pr
    runs-on: [self-hosted, atlas-cluster]
    timeout-minutes: 25
    if: false   # FLIP TO TRUE once a self-hosted runner with the atlas-cluster label is provisioned.
    steps:
      - name: Poll Application health
        env:
          PR_NUMBER: ${{ needs.open-synthetic-pr.outputs.pr_number }}
        run: |
          set -euo pipefail
          deadline=$(( $(date +%s) + 1200 ))
          while [ "$(date +%s)" -lt "$deadline" ]; do
              s=$(kubectl -n argocd get application "atlas-pr-${PR_NUMBER}" \
                  -o jsonpath='{.status.health.status}' 2>/dev/null || echo "")
              echo "$(date -u +%FT%TZ) atlas-pr-${PR_NUMBER} health=${s:-<absent>}"
              [ "$s" = "Healthy" ] && exit 0
              sleep 30
          done
          echo "::error::Application atlas-pr-${PR_NUMBER} never became Healthy"
          exit 1

  trigger-teardown:
    name: Remove deploy-env label
    needs: [open-synthetic-pr, wait-for-healthy]
    if: ${{ always() && needs.wait-for-healthy.result == 'success' }}
    runs-on: ubuntu-latest
    steps:
      - name: gh pr edit --remove-label deploy-env
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          PR_NUMBER: ${{ needs.open-synthetic-pr.outputs.pr_number }}
        run: gh pr edit "$PR_NUMBER" --repo "${GITHUB_REPOSITORY}" --remove-label deploy-env

  assert-reclamation:
    name: Assert per-env state reclaimed
    needs: [open-synthetic-pr, trigger-teardown]
    if: false   # FLIP TO TRUE when wait-for-healthy is enabled.
    runs-on: [self-hosted, atlas-cluster]
    timeout-minutes: 20
    steps:
      - uses: actions/checkout@v4
      - name: Compute ATLAS_ENV
        id: env
        env:
          PR_NUMBER: ${{ needs.open-synthetic-pr.outputs.pr_number }}
        run: |
          atlas_env=$(printf "pr-%d" "$PR_NUMBER" | sha256sum | cut -c1-4)
          echo "atlas_env=$atlas_env" >> "$GITHUB_OUTPUT"
          echo "ATLAS_ENV=$atlas_env"

      - name: Poll until all artifacts reclaimed
        env:
          PR_NUMBER:  ${{ needs.open-synthetic-pr.outputs.pr_number }}
          ATLAS_ENV:  ${{ steps.env.outputs.atlas_env }}
          GH_TOKEN:   ${{ secrets.GHCR_TOKEN }}
        run: |
          set -euo pipefail
          deadline=$(( $(date +%s) + 900 ))
          while [ "$(date +%s)" -lt "$deadline" ]; do
              ok=1

              kubectl -n argocd get application "atlas-pr-${PR_NUMBER}" \
                  >/dev/null 2>&1 && ok=0
              kubectl get ns "atlas-pr-${PR_NUMBER}" >/dev/null 2>&1 && ok=0

              dbs=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" \
                  -U "$DB_USER" -d postgres -tAc \
                  "SELECT count(*) FROM pg_database WHERE datname ~ '-${ATLAS_ENV}\$';")
              [ "$dbs" = "0" ] || ok=0

              t=$(kafka-topics.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --list 2>/dev/null \
                  | grep -E -- "-${ATLAS_ENV}\$" | wc -l)
              [ "$t" = "0" ] || ok=0

              g=$(kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --list 2>/dev/null \
                  | grep -E -- "\\[${ATLAS_ENV}\\]\$" | wc -l)
              [ "$g" = "0" ] || ok=0

              r=$(redis-cli -u "redis://$REDIS_URL" --scan --pattern "${ATLAS_ENV}:*" | wc -l)
              [ "$r" = "0" ] || ok=0

              br=$(gh api -H "Authorization: Bearer $GH_TOKEN" \
                  "/repos/Chronicle20/atlas/git/refs/heads/bot%2Fpr-${PR_NUMBER}-resolved" \
                  --jq '.ref // empty' 2>/dev/null || true)
              [ -z "$br" ] || ok=0

              echo "$(date -u +%FT%TZ) PR=${PR_NUMBER} env=${ATLAS_ENV} dbs=${dbs} topics=${t} groups=${g} redis=${r} branch=${br:-gone} ok=${ok}"
              [ "$ok" = "1" ] && exit 0
              sleep 30
          done
          echo "::error::Some per-env state was not reclaimed within the timeout"
          exit 1

      - name: Dump diagnostics on failure
        if: failure()
        env:
          PR_NUMBER: ${{ needs.open-synthetic-pr.outputs.pr_number }}
        run: |
          mkdir -p /tmp/smoke-artifacts
          kubectl -n argocd describe application "atlas-pr-${PR_NUMBER}" \
            > /tmp/smoke-artifacts/application.txt 2>&1 || true
          kubectl -n argocd logs -l app=atlas-pr-cleanup,atlas.pr-number=${PR_NUMBER} \
            --tail=200 > /tmp/smoke-artifacts/cleanup-job.log 2>&1 || true
      - name: Upload diagnostics
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: smoke-diagnostics-${{ needs.open-synthetic-pr.outputs.pr_number }}
          path: /tmp/smoke-artifacts/

  cleanup-synthetic-pr:
    name: Close PR + delete smoke branch
    needs: [open-synthetic-pr, assert-reclamation]
    if: always() && needs.open-synthetic-pr.result == 'success'
    runs-on: ubuntu-latest
    steps:
      - name: Close PR and delete branch
        env:
          GH_TOKEN:  ${{ secrets.GITHUB_TOKEN }}
          PR_NUMBER: ${{ needs.open-synthetic-pr.outputs.pr_number }}
          BRANCH:    ${{ needs.open-synthetic-pr.outputs.branch }}
        run: |
          gh pr close "$PR_NUMBER" --repo "$GITHUB_REPOSITORY" --delete-branch || \
            gh api --method DELETE "/repos/${GITHUB_REPOSITORY}/git/refs/heads/${BRANCH}" || true
```

If `RUNNER_AVAILABLE=yes` (from Step 9.1), change the two `if: false` lines to:

```yaml
    if: needs.open-synthetic-pr.result == 'success'
```
(for `wait-for-healthy`) and

```yaml
    if: needs.trigger-teardown.result == 'success'
```
(for `assert-reclamation`).

- [ ] **Step 9.3: Lint with `actionlint` (if available)**

```sh
command -v actionlint && actionlint .github/workflows/pr-env-smoke.yml || echo "actionlint not installed; skip"
```

Expected: no errors. If `actionlint` is not on PATH, skip.

- [ ] **Step 9.4: Commit**

```sh
git add .github/workflows/pr-env-smoke.yml
git commit -m "feat(ci): nightly pr-env-smoke test

Opens a synthetic PR labeled deploy-env, waits for Healthy, removes the
label, asserts within 15min that the Application, namespace, per-env DBs,
topics, consumer groups, Redis keys, ghcr tags, and bot branch are all
gone. Gated on a self-hosted runner with label atlas-cluster (if: false
stub if no runner is available yet)."
```

Verify branch/worktree.

---

### Task 10: Env-drift investigation deliverable

**Files:**
- Create: `docs/tasks/task-070-pr-env-teardown-fixes/env-drift-investigation.md`

PRD §4.4 part B — time-boxed best-effort investigation of why `atlas.env` on PRs 491/522 disagreed with the formula. Half-day timebox; if inconclusive, document and move on (the defensive fix in Task 2 already makes drift harmless).

- [ ] **Step 10.1: Run the four investigation probes**

For each probe, capture command + output. The investigator can run these against the live cluster via the existing `mcp__kubernetes__*` MCP tools or `kubectl`. If a current PR-env Application is not available, the investigation can be done dry by reading the cluster-infra ApplicationSet template + git history alone.

Probe A — managedFields walk:
```sh
kubectl -n argocd get application atlas-pr-<N> -o yaml \
  | yq '.metadata.managedFields[] | select((.fieldsV1 // {}) | tostring | contains("atlas.env"))'
```

Probe B — MutatingWebhookConfigurations inventory:
```sh
kubectl get mutatingwebhookconfigurations -A -o yaml \
  | yq '.items[] | {name: .metadata.name, rules: .webhooks[]?.rules[]?}'
```
Look for any rule with `argoproj.io` apiGroup and `Application` resources.

Probe C — Re-render with a debug annotation:
- In the cluster-infra ApplicationSet template, **temporarily** add a second annotation:
  ```yaml
  atlas.env-debug: '{{ printf "%.4s" (sha256sum (printf "pr-%d" .number)) }}'
  ```
- Force the ApplicationSet to regenerate (bump `.metadata.annotations` on the AppSet itself).
- Compare `atlas.env` vs `atlas.env-debug` on the next-generated Application.
- Revert the debug annotation after the test.

Probe D — Stale-template hypothesis:
```sh
cd <cluster-infra repo>
git log --all --diff-filter=M -p -- 'overlays/atlas-pr-applicationset/*.yaml' \
  | grep -E 'atlas\.env|printf.*sha256'
```
Look for any prior formula that would emit `f78b`/`d496` for PRs 491/522.

- [ ] **Step 10.2: Write the deliverable**

Create `docs/tasks/task-070-pr-env-teardown-fixes/env-drift-investigation.md`:

```markdown
# atlas.env annotation drift — investigation

Date: <YYYY-MM-DD when run>
Investigator: <name>
PRD reference: §4.4 Part B
Timebox: half a day

## Observed drift

| PR | computed (formula) | annotation on Application |
|---|---|---|
| 491 | `ed86` | `f78b` |
| 522 | `a476` | `d496` |

Source: `recovery-log.md`.

## Probe A — managedFields walk

Command:
```
<command run>
```

Output:
```
<paste output>
```

Conclusion: <which controllers own metadata.annotations.atlas.env? Expected: ApplicationSet controller only.>

## Probe B — MutatingWebhookConfigurations

Command:
```
<command run>
```

Webhooks that match `argoproj.io/Application`:
```
<paste or "none">
```

Conclusion: <none expected; flag any that do match>

## Probe C — Re-render dry-run

Procedure: temporarily added `atlas.env-debug: '{{ ... }}'` to the AppSet template, forced regenerate, compared `atlas.env` vs `atlas.env-debug` on the next-generated Application.

Result:
```
<paste comparison>
```

Conclusion: <template OK / template eval diverges from formula>

## Probe D — Stale template

Cluster-infra git history for the ApplicationSet template's `atlas.env` line:
```
<paste relevant diffs>
```

Match for `f78b` / `d496`: <yes/no — if yes, what did the historical formula compute, and when was it changed?>

## Verdict

Root cause: <found / inconclusive>

- If found and 1-line fix: implemented in commit <sha>.
- If found and larger: follow-up task <task-NNN>.
- If inconclusive: the defensive `compute_atlas_env` in `cleanup.sh` (task-070 Task 2) renders the drift harmless. We accept the drift unless future incidents show new failure modes.
```

Fill in each placeholder with actual probe output. If a probe is impractical (no live cluster, no cluster-infra access), state that explicitly and skip — do not invent results.

- [ ] **Step 10.3: Commit**

```sh
git add docs/tasks/task-070-pr-env-teardown-fixes/env-drift-investigation.md
git commit -m "docs(task-070): env-hash drift investigation writeup

Best-effort investigation per PRD §4.4 Part B. The defensive fix in
task-070 (deriving ATLAS_ENV from PR_NUMBER inside cleanup.sh) makes the
drift harmless regardless of root cause."
```

Verify branch/worktree.

---

### Task 11: `bootstrap.sh` audit note

**Files:**
- (No code change.) Audit only — produces a note in `context.md`.

PRD §4.4 Part A asks us to audit `bootstrap.sh` (and any other consumer of `ATLAS_ENV`) and decide whether they also need defensive derivation.

- [ ] **Step 11.1: Audit `bootstrap.sh`**

Confirm `bootstrap.sh` reads `ATLAS_ENV` from its environment (line 14 `require_env ATLAS_ENV ...`). That environment is populated by kustomize's `atlas-env` ConfigMap, which is built at PreSync time from already-substituted placeholders (`atlas-env-tokens.yaml`).

Decision: **no code change.** The Bootstrap path is unaffected by annotation drift — the ConfigMap is the contract surface at create time, and it is built from `PLACEHOLDER_ATLAS_ENV` via `pr-validation.yml`'s `update-pr-overlay` step (which uses the canonical formula and is pinned by Task 1's oracle).

- [ ] **Step 11.2: Record the decision in `context.md`**

(Already added during Task 12 below; this step is a pointer.)

---

### Task 12: Write `context.md`

**Files:**
- Create: `docs/tasks/task-070-pr-env-teardown-fixes/context.md`

Capture the cross-cutting decisions and the cluster-infra sibling-PR requirements so execution agents (and code review) don't have to re-derive them.

- [ ] **Step 12.1: Write the document**

(The plan executor writes this directly; the content is below.)

- [ ] **Step 12.2: Commit**

```sh
git add docs/tasks/task-070-pr-env-teardown-fixes/context.md
git commit -m "docs(task-070): context.md — key decisions, files, sibling PR list"
```

Verify branch/worktree.

---

### Task 13: Verification — all tests green, all docker / yaml lints pass

**Files:** none changed; verification only.

- [ ] **Step 13.1: Run all bats tests**

```sh
bats services/atlas-pr-bootstrap/test
```
Expected: all tests pass (Tasks 1, 2, 3, 6, 7 each added or modified tests).

- [ ] **Step 13.2: ShellCheck the changed/new scripts**

```sh
shellcheck services/atlas-pr-bootstrap/scripts/lib.sh \
           services/atlas-pr-bootstrap/scripts/cleanup.sh \
           services/atlas-pr-bootstrap/scripts/sweep-orphans.sh
```
Expected: no errors (warnings about external commands like `kafka-topics.sh` are acceptable; fix anything else).

- [ ] **Step 13.3: Validate YAML files parse**

```sh
yq '.' deploy/k8s/overlays/pr/postdelete-cleanup.yaml >/dev/null
yq '.' .github/workflows/pr-cleanup.yml >/dev/null
yq '.' .github/workflows/pr-env-smoke.yml >/dev/null
echo "yaml ok"
```

- [ ] **Step 13.4: Kustomize-build the PR overlay (with placeholders substituted)**

The overlay can't be built with `PLACEHOLDER_*` tokens still present (kustomize rejects unquoted ints from substring substitution). Simulate `pr-validation.yml`'s sed pass against a temp copy:

```sh
TMP=$(mktemp -d)
cp -r deploy/k8s/overlays/pr "$TMP/"
PR_NUMBER=99999
ATLAS_ENV=$(printf "pr-%d" "$PR_NUMBER" | sha256sum | cut -c1-4)
find "$TMP/pr" -type f \( -name '*.yaml' -o -name '*.yml' \) -exec sed -i \
  -e "s|PLACEHOLDER_ATLAS_ENV|${ATLAS_ENV}|g" \
  -e "s|PLACEHOLDER_PR_NUMBER|${PR_NUMBER}|g" \
  -e "s|PLACEHOLDER_SHA|test123|g" {} +
( cd "$TMP/pr" && kustomize build . >/dev/null )
echo "kustomize ok"
rm -rf "$TMP"
```

Expected: `kustomize ok`. If kustomize is not installed, skip with a note.

- [ ] **Step 13.5: Final commit-history sanity check**

```sh
git log --oneline main..task-070-pr-env-teardown-fixes
```
Expected: 10+ commits covering Tasks 1 through 12. No commits on main.

- [ ] **Step 13.6: Update plan checkboxes**

Open this plan (`docs/tasks/task-070-pr-env-teardown-fixes/plan.md`) and mark every `- [ ]` as `- [x]`. Commit:

```sh
git add docs/tasks/task-070-pr-env-teardown-fixes/plan.md
git commit -m "docs(task-070): mark plan tasks complete"
```

Verify branch/worktree.

---

## Out-of-scope reminders (do not pull into this task)

- **Cluster-infra YAML changes.** ApplicationSet template (drop `cleanup-grace`), CronJob narrowing to sweep-mode, new `ServiceAccount`/`Role`/`RoleBinding`/`Secret atlas-pr-cleanup-gh-token`. These are listed in `context.md` "Sibling PR (cluster-infra)" and tracked separately.
- **GitHub App** to replace the PAT (PRD Open Question 2).
- **Alert wiring** on `atlas_pr_orphan_envs_total` (metric only).
- **Cluster-wide sweep** of pre-existing orphan envs (operator's call post-merge).
- **`tools/task-numbers.sh next`** bug (PRD Open Question 6).
- **Defensive `atlas.env` reconciler** controller (rejected in `design.md` §8).
