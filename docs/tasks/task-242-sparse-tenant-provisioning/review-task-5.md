# Review: Task 5 — Default the ingress ENVIRONMENT header per overlay

**Commit range:** `3b4ae424c..9333f9cea` (single commit `9333f9cea`)
**Brief:** `.superpowers/sdd/plan/task-5-brief.md`
**Implementer's report:** `.superpowers/sdd/plan/task-5-report.md`

## Scope

`git diff --stat 3b4ae424c..9333f9cea`:

```
deploy/k8s/base/atlas-ingress.yaml        | 31 +++++++++++++++++++++++++++----
deploy/k8s/base/env-default.conf.template | 23 +++++++++++++++++++++++
deploy/k8s/base/kustomization.yaml        |  1 +
3 files changed, 51 insertions(+), 4 deletions(-)
```

Matches the brief's Files list exactly (new template, kustomization
generator entry, four edits in `atlas-ingress.yaml`). No drift outside the
named files. `routes.conf.template.generated` untouched (confirmed no diff
in the range).

Additionally traced the interaction between this change and
`deploy/k8s/overlays/pr-sparse/ns-overrides.yaml` (not touched by this
commit, but strategic-merge-patches the exact Deployment/container this
commit edits), since the brief explicitly asked for a live
`overlays/pr-sparse` kustomize render.

## Findings

### PASS — `env-default.conf.template` matches the brief's Step 1 verbatim

`deploy/k8s/base/env-default.conf.template:1-23` is byte-for-byte the
content specified in the brief (comment block plus the `map $http_environment
$atlas_environment { "" "${ATLAS_ENVIRONMENT_DEFAULT}"; default
$http_environment; }` block).

### PASS — `kustomization.yaml` generator entry (Step 2)

`deploy/k8s/base/kustomization.yaml:83-86` (diff) adds
`env-default.conf.template` as a second file in the existing
`atlas-ingress-routes` `configMapGenerator`, exactly as specified.

### PASS — `atlas-ingress.yaml` wiring (Step 3a–3e)

- `deploy/k8s/base/atlas-ingress.yaml:29-32` — `include
  /etc/nginx/conf.d/env-default.conf;` added in `http{}`, before
  `log_format` and before the `server{}` block that uses
  `$atlas_environment` — correct `map` placement/ordering.
- `deploy/k8s/base/atlas-ingress.yaml:34-41` — `log_format main_env`'s last
  field changed to `'env=$atlas_environment'` and the preceding comment
  updated to describe the resolved environment.
- `deploy/k8s/base/atlas-ingress.yaml:58` — `proxy_set_header ENVIRONMENT
  $atlas_environment;` (was `$http_environment`). Confirmed by grep: zero
  remaining `$http_environment` references anywhere in
  `atlas-ingress.yaml`; all four `$atlas_environment` references present.
- `deploy/k8s/base/atlas-ingress.yaml:111-134` (diff) — `NGINX_ENVSUBST_FILTER`
  extended to `"POD_NAMESPACE|NS_|ATLAS_ENVIRONMENT_DEFAULT"`, and the
  `ATLAS_ENVIRONMENT_DEFAULT` env var added via `configMapKeyRef` on
  `atlas-env`/`ATLAS_ENVIRONMENT`, with **no** `optional: true` — confirmed
  present on the rendered Deployment for base, overlays/main, overlays/pr
  (see verification below).
- `deploy/k8s/base/atlas-ingress.yaml:275-278` (diff) — `nginx-routes-volume`
  mount for `env-default.conf.template` at
  `/etc/nginx/templates/env-default.conf.template`, matching the existing
  `routes.conf.template` mount's pattern (`/etc/nginx/templates/*.template`
  → rendered by the nginx image's envsubst entrypoint to
  `/etc/nginx/conf.d/*`, matching the `include` path added in 3a).

### PASS — `routes.conf.template.generated` untouched, no `proxy_set_header ENVIRONMENT` inside it

`git diff 3b4ae424c..9333f9cea -- deploy/k8s/base/routes.conf.template.generated`
is empty. `grep -n "proxy_set_header ENVIRONMENT"
deploy/k8s/base/routes.conf.template.generated` returns no matches.

### PASS — live renders: base / overlays/main / overlays/pr match expected values

Ran `kubectl kustomize` (v1.32.3 client / kustomize v5.5.0) against each
target and read the `atlas-env` ConfigMap's `ATLAS_ENVIRONMENT` key
(the value `configMapKeyRef` resolves to at pod start) plus the ingress
Deployment's env list:

| Overlay | `ATLAS_ENVIRONMENT` (atlas-env CM) | Expected | Match |
|---|---|---|---|
| base | `` (empty) | — | — |
| overlays/main | `main` | `main` | yes |
| overlays/pr | `` (empty) | empty | yes |
| overlays/pr-sparse | `pr-PLACEHOLDER_PR_NUMBER` | `pr-PLACEHOLDER_PR_NUMBER` | yes (CM value correct; see below for whether it reaches the pod) |

`ATLAS_ENVIRONMENT_DEFAULT` env var (`configMapKeyRef`, no `optional:
true`) present on the rendered `atlas-ingress` Deployment's `nginx`
container for base, overlays/main, overlays/pr — confirmed by parsing the
rendered YAML with `yaml.safe_load_all`, not by grep/eyeball.

`gen-routes.sh --check` → `gen-routes: up to date` (exit 0).
`pr-sparse-mirror-guard.sh` → `pr-sparse-mirror-guard: up to date` (exit 0).
`git status --porcelain deploy/k8s/overlays/pr-sparse` → empty (overlay
genuinely untouched).

### FINDING (non-blocking, pre-existing, out of this diff's root cause) — `ATLAS_ENVIRONMENT_DEFAULT` does not currently reach the `overlays/pr-sparse` ingress Deployment; no env var does

Rendering `kubectl kustomize deploy/k8s/overlays/pr-sparse` today and
inspecting the `atlas-ingress` Deployment's `nginx` container shows an
**empty env list** — not just missing `ATLAS_ENVIRONMENT_DEFAULT`, but
missing `POD_NAMESPACE`, every `NS_*` variable, and every other env var
base defines on that container.

Root cause: `deploy/k8s/overlays/pr-sparse/ns-overrides.yaml` is a
strategic-merge patch on `apps/v1 Deployment atlas-ingress`'s `nginx`
container that is checked in as:

```yaml
containers:
  - name: nginx
    env:
      #PLACEHOLDER_NS_OVERRIDES
```

Parsed, `env:` here is `None` (the only line under it is a full-line
comment), not an empty list. `python3 -c "yaml.safe_load(...)"` on this
file confirms `env == None`. A strategic-merge patch that sets a
list-typed field to `null` deletes that field from the merged object
entirely — it does not "layer zero entries" as the file's own comment
claims. This is empirically reproducible: `kubectl kustomize
deploy/k8s/overlays/pr-sparse` in the checked-out repo yields `env: []` on
that container, wiping every base-defined env var, not merging by the
`name` patch-merge-key as a genuinely empty *list* (`env: []`) would.

This is confirmed **pre-existing**: checking out the pre-Task-5 commit
`3b4ae424c` and rendering the same overlay produces the identical empty
env list — `NS_ATLAS_*` vars were already being wiped before this task's
commit. This diff did not cause the defect; it inherited it. The
implementer's report tested `kubectl kustomize deploy/k8s/overlays/pr-sparse
> /tmp/prsparse.yaml # exit 0` and `git status --porcelain
deploy/k8s/overlays/pr-sparse` (empty) but never grepped/parsed that
render for `ATLAS_ENVIRONMENT_DEFAULT` the way it did for `overlays/pr` —
so the report's "all expected outcomes... were met" does not actually
cover the pr-sparse case the reviewing brief asked to verify.

Per the file's own comments, `#PLACEHOLDER_NS_OVERRIDES` is meant to be
filled by CI ("Task 50," not yet landed anywhere in this repo — `grep -rn
PLACEHOLDER_NS_OVERRIDES` outside this one file returns nothing) before a
real per-PR deploy applies the manifest. If that substitution genuinely
happens before every real apply, this defect may never manifest in
production, only in a local/CI-dry-run `kubectl kustomize` against the
checked-in placeholder state. But that substitution step does not exist
in this repo yet, so today, a literal `kubectl kustomize
deploy/k8s/overlays/pr-sparse | kubectl apply` would strip the ingress
Deployment's `nginx` container of `POD_NAMESPACE`/`NS_*` routing vars
entirely — a pre-existing, unrelated, and apparently more severe latent
break than the one this review was scoped to find.

Disposition: **not attributable to this commit** (identical behavior
proven at the pre-Task-5 base commit) and outside this diff's edited
files, so not blocking for Task 5. Flagged because the task-5 brief
explicitly asked for a live `overlays/pr-sparse` verification with a
named expected value, and that expectation is not met as of the current
repo state — the controller should not treat "pr-sparse renders and the
mirror guard is green" as proof the new default reaches pr-sparse's
ingress pod. Whoever owns "Task 50" (the CI NS_* substitution step) should
be made aware `ns-overrides.yaml`'s checked-in placeholder does not behave
as its own comment claims.

### PASS — empty-default no-op behavior

`env-default.conf.template`'s comment claims nginx omits a
`proxy_set_header` whose value resolves to the empty string rather than
sending an empty header. This matches documented nginx behavior (`ngx_http_proxy_module`:
setting a proxied header to an empty string suppresses it). Combined with
`overlays/pr`'s `ATLAS_ENVIRONMENT` being empty (confirmed in the render
table above) and `overlays/pr` not being subject to the `ns-overrides.yaml`
patch (that file lives only in `overlays/pr-sparse`), untagged requests on
`overlays/pr` do stay untagged as intended.

## Not evaluable

- Whether "Task 50" (CI substitution of `#PLACEHOLDER_NS_OVERRIDES`) exists
  or is planned elsewhere in the task-242 plan was not checked beyond a
  repo-wide grep for the placeholder string; if it exists as a script not
  yet wired into this repo (e.g. lives in a CI pipeline definition outside
  the repo), that is outside this review's surface.
- Runtime behavior of the nginx `map`/`include`/`envsubst` chain was
  verified by reading nginx documentation and the config's static
  structure, not by actually running the nginx container image; no live
  pod was started.

## Verdict rationale

All five brief steps are implemented exactly as specified, mechanically
verified against live `kubectl kustomize` renders for base/main/pr (not
taken at the implementer's word), plus the two named guard scripts both
exit 0. The one substantive finding — pr-sparse's ingress Deployment
losing its entire env list, including the new `ATLAS_ENVIRONMENT_DEFAULT`
— traces to a file this commit did not touch and is reproducible
identically before this commit, so it is not this unit's defect to fix,
but it does mean the brief's own acceptance check ("pr-PLACEHOLDER_PR_NUMBER
for pr-sparse") is not actually satisfied end-to-end today, and the
implementer's report did not surface that gap.
