# Review — Task 4 (sparse-tenant-provisioning): baseline `ATLAS_ENVIRONMENT` + `main` environment record

Range: `0c0b1cef8..e63598524`. Two commits:

- `e0722217b` — Task 4 proper. Reviewed in full.
- `e63598524` — controller's mechanical mirror repair of
  `deploy/k8s/overlays/pr-sparse/patches/consumer-group-env.yaml`. Confirmed
  byte-identical to `overlays/pr`'s copy and free of any literal `main`; not
  reviewed line by line per instructions.

## Scope confirmed

Diff touches exactly the four files the brief named:
`deploy/k8s/base/env-configmap.yaml`,
`deploy/k8s/overlays/main/kustomization.yaml`,
`deploy/k8s/overlays/main/environment-record.yaml` (new),
`deploy/k8s/overlays/pr/kustomization.yaml`, plus the separately-scoped
mirror repair to `pr-sparse/patches/consumer-group-env.yaml`. No other files
touched. `git diff --stat 0c0b1cef8..e63598524` matches this exactly (5
files, 130 insertions, 0 deletions). No `pr-sparse` edits from the task-4
commit itself (`git show e0722217b --stat | grep -c pr-sparse` → 0).

## Checks performed and evidence

### 1. `ATLAS_ENV` vs `ATLAS_ENVIRONMENT` — not conflated

`git show e0722217b | grep -n 'ATLAS_ENV='` → no match (exit 1, none of the
four hash-suffix `ATLAS_ENV=` values were touched). Every added line is
`ATLAS_ENVIRONMENT=...` or `ATLAS_ENVIRONMENT: ""`. PASS.

### 2. `ATLAS_ENVIRONMENT` is DEFINED everywhere, never absent

- Base configmap (`deploy/k8s/base/env-configmap.yaml:18`):
  `ATLAS_ENVIRONMENT: ""` — present, empty, first entry under `data:`, with
  the verbatim rationale comment.
- `overlays/pr/kustomization.yaml` (`behavior: replace` generator) adds its
  own `- ATLAS_ENVIRONMENT=` (line 218 of the diff, confirmed present in
  file), independently of the base key — correct, since `replace` would
  otherwise drop the base's key entirely. Rendered:
  `kubectl kustomize deploy/k8s/overlays/pr | grep ATLAS_ENVIRONMENT` →
  `288:  ATLAS_ENVIRONMENT: ""`. PASS.
- `overlays/main/kustomization.yaml` (`behavior: replace`) adds
  `- ATLAS_ENVIRONMENT=main`. Rendered:
  `kubectl kustomize deploy/k8s/overlays/main | grep ATLAS_ENVIRONMENT` →
  `236:  ATLAS_ENVIRONMENT: main`. PASS.

### 3. Baseline id is exactly `main`

`overlays/main` sets `namespace: atlas-main` (unchanged,
`deploy/k8s/overlays/main/kustomization.yaml:7`) and
`ATLAS_ENVIRONMENT=main` — `atlas-` stripped, matching the CI derivation
referenced in the comment (`.github/workflows/pr-validation.yml:921`). The
new Job's POST body also sets `"baseline": "main"`, `"namespace":
"atlas-main"`, `"name": "main"` — internally consistent with
`MapRegistry.IsOwner`'s `rec.Baseline == r.self` comparison. PASS.

### 4. Isolated mode (`overlays/pr`) unchanged behaviourally

Rendered `ATLAS_ENVIRONMENT: ""` (empty string, present) — confirmed above.
No other keys in the `pr` overlay's `atlas-env` generator were touched
(diff only inserts one new literal line above `BASE_SERVICE_URL`). PASS.

### 5. `pr-sparse` untouched beyond the mirror repair, and no literal `main`

- `git show e0722217b --stat | grep -c pr-sparse` → `0`: task-4's commit
  made zero edits under `overlays/pr-sparse/`.
- The separate mirror-repair commit only touches
  `patches/consumer-group-env.yaml`, one of the nine files in
  `tools/pr-sparse-mirror-guard.sh:31-41` MIRRORS array — confirmed byte-
  identical between `overlays/pr/patches/consumer-group-env.yaml` and
  `overlays/pr-sparse/patches/consumer-group-env.yaml` (`diff` exit 0), and
  `grep -n 'main' pr-sparse/patches/consumer-group-env.yaml` returns
  nothing.
- Rendered `pr-sparse`:
  `ATLAS_ENVIRONMENT: pr-PLACEHOLDER_PR_NUMBER` at line 271 (still
  template-placeholder, unchanged by this task). `grep -nw 'main'` on the
  full ~9000-line render surfaces only shell-script comments/function names
  (`main() { ... }`, NG6 offset-seeding comments) — none in control-plane
  wiring (`baseline`/`namespace` fields still read
  `PLACEHOLDER_BASELINE_ENVIRONMENT` / `PLACEHOLDER_BASELINE_NAMESPACE`).
  PASS.

### 6. `main`'s environment-record Job — target, idempotency, shape

- POSTs to `http://atlas-configurations.atlas-main.svc.cluster.local:8080/…`
  — the ClusterIP DNS name, not the ingress. Confirmed
  `atlas-configurations` container/service exposes port 8080
  (`deploy/k8s/base/atlas-configurations.yaml:20,45,47`). No dependency on
  `atlas-ingress` health or the (not-yet-landed) ingress default, so the
  Job cannot 400 itself. PASS.
- GET-then-POST idempotency: `curl -fsS "$base/$name"` short-circuits with
  `exit 0` before the POST — matches
  `deploy/k8s/overlays/pr-sparse/environment-record.yaml`'s pattern
  exactly. PASS.
- `sync-wave: "11"`, `Force=true,Replace=true`, `backoffLimit: 3`,
  `alpine:3.20` + `apk add --no-cache --quiet curl` — all match the
  pr-sparse Job's shape. `phase: "ACTIVE"` (vs pr-sparse's
  `"PROVISIONING"`) and `sync-wave: "11"` (vs pr-sparse's `"1"`) are
  deliberate, documented deviations appropriate to main always being active
  and needing to run after wave-10 Deployments are healthy
  (`deploy/k8s/base/kustomization.yaml:97-106`, referenced in the Job's own
  header comment). `tenant: ""` is explained and consistent with
  `do_sweep_tenant` being sparse-only. PASS.
- `overlays/main/kustomization.yaml` adds `- environment-record.yaml` to
  `resources`, immediately below `atlas-env-tokens.yaml` — confirmed
  present at kustomization.yaml:10. PASS.

### 7. Overlays render clean

`kubectl kustomize` on all three overlays (`main`, `pr`, `pr-sparse`) exits
0 with no errors — ran directly, not taken on the report's word.

### 8. Mirror guard

`./tools/pr-sparse-mirror-guard.sh` → `pr-sparse-mirror-guard: up to date`,
exit 0. The report's claim of a pre-existing failure is now moot: the
controller's separate mechanical commit (`e63598524`) fixed it, and the
guard is green at HEAD of this range. Confirms the mirror-repair commit did
what it claimed.

### 9. `overlay-env-guard.sh` reference

The `main` kustomization's new comment
(`deploy/k8s/overlays/main/kustomization.yaml:46`) references
`tools/overlay-env-guard.sh`, which does not exist yet in this range —
verified this is intentionally forward-referencing Task 6's deliverable
(`docs/tasks/task-242-sparse-tenant-provisioning/context.md:18,181`,
`plan.md:1058` describe it as a Task 6 new file). Not a defect in Task 4.

## Not evaluable

None. All checks in this task's scope were run against live tool output
(kustomize render, diff, grep, guard script), not taken on the
implementer's report.

## Verdict

All brief requirements are met exactly as specified, all four global
constraints hold under direct verification, and the mechanical mirror
repair is confirmed byte-identical with no literal baseline name leaked
into `pr-sparse`. No blocking or non-blocking findings.
