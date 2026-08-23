# Bug: a UI-only PR gets a sparse environment that does not deploy the changed atlas-ui

Found on PR #1458 (`task-258-refreshable-empty-states`, branch head `ef16cd9fb`,
base `1461bfc96`) — 37 changed files, all under `services/atlas-ui/`.

## Reproduced

Not a runtime reproduction: the defect is visible in the PR's own mode-report
comment plus the two scripts that produce it. Confirmed by reading the emitted
report on PR #1458 and by sweeping the override-set computation.

## Observed

`<!-- atlas:mode-report -->` on PR #1458:

```
### Ephemeral environment: **sparse**
| Mode | `sparse` |
| Reason | no service dependency-graph impact; only the mandatory atlas-login/atlas-channel floor deployed |
| Workloads deployed | 2 of 77 |
| Override set | `atlas-channel`, `atlas-login` |
| Everything else | served by `main` |
```

The only service the PR changes — `atlas-ui` — is not in the override set, so
`atlas-pr-1458` serves the **baseline** `main` build of the UI. The PR
environment cannot exercise the change it exists to validate.

The consequence is not merely "atlas-ui isn't added": the sparse overlay
actively routes away from it. In `.github/workflows/pr-validation.yml`
(`update-pr-overlay`):

- `PLACEHOLDER_DELETE_BLOCK` (~line 976) emits a `$patch: delete` for every
  base Deployment not in `KEEP=" $OVERRIDE_SERVICES atlas-ingress "` —
  `deploy/k8s/base/atlas-ui.yaml`'s Deployment is deleted from the namespace.
- `PLACEHOLDER_NS_OVERRIDES` (~line 1002) then sets `NS_ATLAS_UI` to the
  baseline namespace, so `atlas-ingress`'s `location /`
  (`deploy/k8s/base/routes.conf.template.generated:678` →
  `atlas-ui.${NS_ATLAS_UI}.svc.cluster.local:80`) proxies to `main`'s UI.

The image itself *was* built: the docker build matrix and the "Bump image tags
for built services" step both read `docker-services-matrix`, which does include
`atlas-ui`. Only the override set — hence the deployment — omits it.

## Expected

A PR whose only change is `services/atlas-ui/**` gets a sparse environment whose
override set includes `atlas-ui`, so the namespace runs the PR's own UI image
and `NS_ATLAS_UI` stays local.

## Root cause

`tools/mode-select.sh` builds the override set from the Go-only slice of the
`cideps` output:

```sh
SVC_NAMES=$(printf '%s' "$OUT" | jq -r '."go-services"[].name' 2>/dev/null || true)
OVERRIDES=$(printf '%s\natlas-login\natlas-channel\n' "$SVC_NAMES" | ... )
```

`atlas-ui` is `"type": "node-service"` in `.github/config/services.json:559` and
has no `go.mod` under `atlas.com/`, so `tools/cideps/graph.go:113` deliberately
skips it — it can never appear in `go-services`. It *does* appear in the
`docker-services` slice: `tools/cideps/main.go` merges names passed via
`--changed-services` back into `dockerSvcNames` precisely because non-Go
services would otherwise be dropped. `mode-select.sh` never reads that slice.

So for a UI-only PR: `CHANGED_SVCS=atlas-ui` → `go-services` is empty →
`SVC_NAMES` empty → override set collapses to the mandatory floor, and the
reason line reports "no service dependency-graph impact", which is true only of
the Go graph.

This was a known-but-unclosed hole, not an accident: the comment at
`.github/workflows/pr-validation.yml:981-983` already records that
`all-go-services` "excludes atlas-ui (which DOES have one [a base Deployment],
deliberately not part of the sparse override-set computation)", and
`deploy/k8s/overlays/pr-sparse/README.md:96-103` still describes the override
set as a fixed four-service list.

Nothing downstream blocks `atlas-ui` from being in the override set:

- `PLACEHOLDER_SERVICE_ID_BLOCK` / precreate loops (~line 1040) skip any
  override-set service whose base Deployment carries no `SERVICE_TYPE` env —
  `deploy/k8s/base/atlas-ui.yaml` carries none, so it is skipped cleanly.
- The "no SERVICE_ID-carrying override-set Deployment" guard (~line 1071) is
  still satisfied by the `atlas-login`/`atlas-channel` floor.
- The render-verification loop (~line 1146) applies the same `SERVICE_TYPE`
  filter and therefore skips `atlas-ui` too.
- `environment-record.yaml`'s `overrides` map just gains an `atlas-ui` entry
  pointing at `atlas-pr-<N>` — the ownership gate reads it by name.

## Fix

Make the override set cover non-Go services that have their own base
Deployment, instead of only Go services.

- `tools/mode-select.sh` — replace the `."go-services"[].name` extraction with
  the union of `."go-services"[].name` and `."docker-services"[].name`, then
  filter that union to names for which `deploy/k8s/base/<svc>.yaml` exists and
  declares a `Deployment` named `<svc>` (a service with no base Deployment must
  not enter the override set: it would land in `overrides` in the environment
  record and in `KEEP` while nothing deploys it). Keep the mandatory
  `atlas-login`/`atlas-channel` floor and the existing sort/dedupe. Update the
  `REASON` line so a non-Go-only change no longer reports "no service
  dependency-graph impact" when it does have deployable impact. Preserve the
  forced-sparse and cideps-failure paths unchanged.
- `tools/mode-select_test.sh` — add cases: (a) `services/atlas-ui/src/...`
  changed → mode `sparse`, override set contains `atlas-ui` plus the floor;
  (b) the existing Go-service case still yields exactly its prior set (no
  spurious additions from `docker-services`); (c) a changed service with no
  base Deployment is excluded from the override set.
- `.github/workflows/pr-validation.yml` — the `PLACEHOLDER_DELETE_BLOCK`
  comment block (~lines 977-985) states as fact that `atlas-ui` is
  "deliberately not part of the sparse override-set computation". Correct it;
  the `all-go-services`-vs-base-Deployment discrepancy it documents is still
  real and must stay.
- `deploy/k8s/overlays/pr-sparse/README.md` — "The override set" (lines 96-103)
  describes a fixed four-service list. Restate it as computed by
  `tools/mode-select.sh` = affected Go services ∪ changed non-Go services with a
  base Deployment ∪ the mandatory floor, plus always-local `atlas-ingress`.

No manifest or overlay change is required; the fix is entirely in the
override-set computation and its documentation.

## Not yet answered

- Whether any other non-Go service with a base Deployment (e.g. `atlas-assets`,
  if it has one) should be treated identically — the rule above admits it
  automatically, which is believed correct but has not been exercised by a PR.
- Whether a UI-only sparse environment additionally needs the baseline's
  `atlas-ui` traffic split considered (it does not today: `NS_ATLAS_UI` is a
  single-target nginx variable), so this is expected to be a no-op concern.
- Not re-tested live: PR #1458 must be re-run (push to the branch) and its
  mode-report re-read to confirm `atlas-ui` appears in the override set and the
  namespace runs the PR image.

## Resolution

Fixed in `cc1c27899` — "fix(ci): include non-Go services with a base Deployment
in the sparse override set". The override set is now the union of `go-services`
and `docker-services`, filtered to names with their own
`deploy/k8s/base/<svc>.yaml` Deployment, plus the mandatory floor. Four files:
`tools/mode-select.sh`, `tools/mode-select_test.sh` (3 new cases),
`.github/workflows/pr-validation.yml` (stale comment), and
`deploy/k8s/overlays/pr-sparse/README.md` (stale override-set description).

Verified:

- `./tools/mode-select_test.sh` → PASS; `shellcheck --severity=error` clean.
- `printf 'services/atlas-ui/src/app/page.tsx\n' | ./tools/mode-select.sh` →
  `sparse` / `atlas-channel atlas-login atlas-ui` / `affected services: atlas-ui`.
- `tools/verify.sh --quick --base 1461bfc96` → exit 0 (includes the
  pr-sparse mirror drift, sparse baseline scoping, and mode-select decision
  table guards).

Live re-test still pending: PR #1458 must re-run so its `atlas:mode-report`
comment can be re-read for `atlas-ui` in the override set, and the
`atlas-pr-1458` namespace checked for a local `atlas-ui` Deployment on the PR
image.
