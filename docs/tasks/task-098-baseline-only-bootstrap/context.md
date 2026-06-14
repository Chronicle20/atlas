# Context — task-098 Baseline-Only Ephemeral Bootstrap

Companion to `plan.md`. Orientation for an engineer with zero prior context on this codebase.

## What this task is

The per-PR ephemeral environment bootstrap (`atlas-pr-bootstrap`) currently has three data-provisioning modes (`auto|baseline|full`). The `full` mode re-ingests a ~1 GB WZ tree into shared MinIO (per-tenant bloat), and the `auto` mode silently falls back to `full` when no baseline exists. A separate init container (`fetch-wz-canonical`) unconditionally downloads `atlas.zip` and 404s when it's absent, wedging every new env (this happened in prod on 2026-06-14).

This task makes the bootstrap **baseline-only**: it restores a published canonical baseline and nothing else, and a new read-only **preflight** hard-fails *before any data-affecting work* when no baseline exists for the env's version. It's a deletion-plus-guard change — no Go, no API, no schema.

## Key files (all under the worktree `.worktrees/task-098-baseline-only-bootstrap/`)

| File | Why it matters |
|---|---|
| `services/atlas-pr-bootstrap/scripts/bootstrap.sh` | The bootstrap script. The whole behavioral change lives here. `set -euo pipefail`; sources `lib.sh`. |
| `services/atlas-pr-bootstrap/scripts/lib.sh` | Shared helpers: `log` (→ stderr), `require_env`, `retry max sleep cmd…`, `http_ok`, `http_ok_tenant`. The plan reuses `retry`. |
| `services/atlas-pr-bootstrap/test/bootstrap_test.bats` | bats tests. Existing pattern: run the real script with a doctored env / `PATH` and assert on `$status`/`$output`. |
| `deploy/k8s/overlays/pr/sync-bootstrap.yaml` | The bootstrap Job manifest (Argo `Sync` hook). Holds the init container + `/opt/wz` `emptyDir` volume to delete. |
| `docs/runbooks/ephemeral-pr-deployments.md` | Operator runbook; §9.1 documents the old `atlas.zip`/`BOOTSTRAP_MODE` flow. |
| `docs/runbooks/canonical-version-migration.md` | task-095 runbook; its step 4 publishes per-version baselines. The bootstrap now *depends* on it for cold-start. |

## Important codebase facts (verified, don't re-derive)

- **`atlas-pr-bootstrap` has no `go.mod`.** It's a shell container. The CLAUDE.md `go test`/`go vet`/`docker buildx bake` rules do **not** apply. Verification = bats + shellcheck + `kustomize build` + (optional) `docker build`.
- **`log()` writes to stderr only** (lib.sh). Functions that need to return a value `echo` it to stdout; a stray `log` inside a `$()`-captured function corrupts the captured value (PR-544 bug). That's why the new `preflight_baseline` returns its HTTP codes through a **global** (`BASELINE_PROBE_CODE`) rather than via command substitution.
- **`retry max sleep cmd args…`** runs `cmd` in the *current* shell (no subshell), so a predicate it calls can set globals that survive — same pattern the existing `data_processing_done` stability check uses.
- **`exit` inside `$(...)` only kills the subshell**, not the script. So `probe_baseline_object` (which can `exit 1`) is called directly, never inside command substitution. This is the single most important correctness constraint in the new code.
- **The script overwrites env-injected `REGION`/`MAJOR_VERSION`/`MINOR_VERSION`** with the canonical values from `/atlas/canonical/tenant.json` partway through (current lines ~211-213). The preflight therefore reads the *canonical* values up front (via `CANONICAL_TENANT_JSON`) so it probes exactly the version the later restore will request — not the initial configmap values.
- **The `/opt/wz` volume is an `emptyDir`, not a PVC** (design OQ-1, verified: `grep -rn atlas-wz-canonical deploy` is empty). Removal is three manifest blocks only; there is no PVC/PV to delete.
- **`canonical_baseline_exists()`** (current lines ~127-132) is the existing binary HEAD probe. The new preflight supersedes it; it gets deleted in Task 3.

## Design decisions already locked (from design.md)

- **Approach B** (PRD-specified): remove `full` entirely; baseline-only with a fail-fast preflight. Alternatives A (keep `full` as opt-in) and C (move enforcement into atlas-data) were rejected — A leaves the bloat one env-var away; C is out of scope (no atlas-data changes allowed).
- **Preflight placement:** first action after the TENANT_ID UUID-shape check, before `wait-ready`. It's an anonymous MinIO HEAD with no dependency on atlas-data being up — failing here means the Job dies before touching tenants/configs/deployments.
- **Three-way probe semantics (design §3.3):** both objects 200 → proceed; either 404 → fail fast ("publish a baseline" + runbook link, names the version); connection failure `000` → bounded retry, then a *distinct* "MinIO unreachable" error (never misreport a transient blip as "missing").
- **OQ-3:** probe **both** `documents.dump` and `documents.dump.sha256` so a half-published baseline fails the preflight, not the later restore.
- **`CANONICAL_TENANT_JSON`** is the single source of truth for the canonical-tenant path, used by both the preflight and the tenant-create step; overridable so bats can point at a fixture without a cluster.

## Acceptance grep gate (the authoritative "done" check)

These tokens must return **no hits** in `services/atlas-pr-bootstrap/` and `deploy/k8s/overlays/pr/`:

```
BOOTSTRAP_MODE  WZ_CANONICAL  fetch-wz-canonical  /opt/wz  resolve_mode  atlas-canonical/atlas.zip
```

(Historical references under `docs/tasks/task-063|071/` are pre-existing and out of scope — they're not in the scanned paths.)

## Verification commands (no cluster needed for 1-4)

1. `cd services/atlas-pr-bootstrap && bats test/`
2. `shellcheck services/atlas-pr-bootstrap/scripts/*.sh`
3. `kustomize build deploy/k8s/overlays/pr >/dev/null` (or `kubectl kustomize`)
4. the acceptance grep above
5. `docker build -t atlas-pr-bootstrap:task098 services/atlas-pr-bootstrap` (if Docker available; not a `bake` target)

## Test-host prerequisites

The new bats tests need `jq` and `timeout` on the host; both have `skip` guards so the suite still runs where they're absent. The `curl`/`kubectl` PATH shims and a fixture `tenant.json` let the real script run preflight without a cluster — the `kubectl` shim doubles as a sentinel proving the preflight exits before any cluster mutation.

## Out of scope (do not touch)

- `atlas-data` code / `/api/data/wz` / `/api/data/process` endpoints (operators still use them).
- `baseline/publish` / `baseline/restore` (task-095 owns them).
- `predelete-purge` and the external PostDelete DB-drop orphaned-DB leak (cluster-infra repo; see `bug_ephemeral_db_teardown_leak_superuser`).
- Deleting the now-unreferenced `atlas-canonical/atlas.zip` object (operator's call; not this task).
- Which version(s) an env bootstraps; multi-version bootstrap.
