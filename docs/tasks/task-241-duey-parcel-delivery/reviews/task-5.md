# Review: Task 5 — Register atlas-parcel across build, CI, k8s and databases

Commit range: `6879aa7e0..37a0fb601` (single commit `37a0fb601`, 17 files
changed, 162 insertions / 3 deletions, no Go source).

Brief: `.superpowers/sdd/plan/task-5-brief.md`
Implementer report: `.superpowers/sdd/plan/task-5-report.md`

## Scope confirmed

Diff matches the brief's file list plus one legitimately-discovered addition
(`deploy/k8s/base/atlas-ingress.yaml` — the `NS_ATLAS_PARCEL` env var, flagged
by `tools/gen-routes.sh` as required and documented in the report as an
addition beyond the brief's named files). No Go files touched, consistent
with the concurrent lint-fix round being out of scope here. Scope matches the
work found.

## Guard clears

```
$ tools/service-registration-guard.sh
service-registration-guard: clean
```

`tools/service-registration-guard.sh` itself is untouched by this commit
(`git diff 6879aa7e0..37a0fb601 --stat -- tools/service-registration-guard.sh`
is empty), so this is the same guard that failed with `FAIL atlas-parcel: no
base manifest deploy/k8s/base/atlas-parcel.yaml` at the base commit. PASS.

## Ruling 1 — topic scope

`deploy/k8s/base/env-configmap.yaml` gains exactly three identity entries:
`COMMAND_TOPIC_PARCEL_CUSTODY`, `EVENT_TOPIC_PARCEL_CUSTODY_STATUS`,
`EVENT_TOPIC_PARCEL_STATUS`. Same three (with `-main` / `-PLACEHOLDER_ATLAS_ENV`
suffixes) appear in `deploy/k8s/overlays/main/kustomization.yaml` and
`deploy/k8s/overlays/pr/kustomization.yaml`'s `configMapGenerator` literals.
`COMMAND_TOPIC_PARCEL` does not appear anywhere in the diff
(`git diff 6879aa7e0..37a0fb601 | grep -i world` and a direct grep for
`COMMAND_TOPIC_PARCEL"` confirm only the `_CUSTODY` variant is present).
Neither over- nor under-registered. PASS.

## Generated vs hand-edited files

`ns-vars.generated.yaml` and `routes.conf.template.generated` — ran
`tools/gen-routes.sh` after the review's own edits were already in the
worktree; output: `gen-routes: up to date`, i.e. the committed generated
files are byte-identical to what the generator produces from the current
`deploy/shared/routes.conf` + `atlas-ingress.yaml` NS_* set. Not hand-drifted.
PASS.

The PR-overlay generator-owned files (`db-name-suffix.yaml`,
`consumer-group-env.yaml`, the PR topic literals in `kustomization.yaml`, and
`atlas-pr-cleanup-env.example.yaml`) all show the expected mechanical
`PLACEHOLDER_ATLAS_ENV` shape matching every neighboring entry — consistent
with generator output, no hand-authored deviation visible in the diff. PASS.

## Nginx route path vs Task 4's resource.go

`services/atlas-parcel/atlas.com/parcel/parcel/resource.go:39-44` registers:
- `PathPrefix("/parcels")` → `GET /parcels`, `GET /parcels/{parcelId}`
- `PathPrefix("/characters/{characterId}")` → `GET /characters/{characterId}/parcel-status`

`main.go:42,72` sets `prefix: "/api/"` via `SetBasePath(GetServer().GetPrefix())`
— same pattern as `atlas-mts/main.go:50,112` — so the served paths are
`/api/parcels`, `/api/parcels/{parcelId}`, and
`/api/characters/{characterId}/parcel-status`.

`deploy/shared/routes.conf` (and its generated
`deploy/k8s/base/routes.conf.template.generated`) add:
- `^/api/parcels(/.*)?$` → `atlas-parcel:8080`
- `^/api/characters/[^/]+/parcel-status(/.*)?$` → `atlas-parcel:8080`

Both regexes cover the actual routes Task 4 shipped, and `proxy_pass
http://$u$request_uri` forwards the full `/api/...` path, matching the
service's `/api/` base path. PASS — no latent 404 for Task 26's consumers.

## `main-b4cf63c` precedent

Report claims precedent from atlas-events' registration commit `545278051`.
Verified directly: `git show 545278051 -- deploy/k8s/overlays/main/kustomization.yaml`
shows `- name: ghcr.io/chronicle20/atlas-events/atlas-events` /
`newTag: main-d23dffc` added for a then brand-new image, using the fleet's
then-current tag exactly as this commit does with `main-b4cf63c`. Precedent
is real, not fabricated. Accepted as an expected first-registration
condition, not a blocking deploy defect — matches the pattern this repo has
already shipped once.

## Pattern check — silent world-0 defaulting

No occurrence. This commit is pure list/manifest registration (services.json,
docker-bake.hcl, k8s manifests/overlays, routes.conf, db-bootstrap.sh) with
no code path that resolves or defaults a world id. `grep -i world` over the
full diff only turns up the pre-existing `EVENT_TOPIC_WORLD_RATE` literal
(unrelated, untouched) and the `atlas-world` service name inside the
untouched-except-for-insertion `ATLAS_SERVICES` string. N/A — no finding.

## Other checks

- `deploy/k8s/base/atlas-parcel.yaml` is a faithful mirror of
  `atlas-mts.yaml` (container `parcel`, `DB_NAME: "atlas-parcel"` unsuffixed,
  no `namespace:` field, `envFrom: configMapRef: atlas-env` — matches every
  other base manifest's mechanism for picking up topic vars, verified no
  base manifest lists per-service `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` `env:`
  entries individually). PASS.
- `docker buildx bake atlas-parcel` reported success in the implementer's
  report (not independently re-run in this review — build-tool output,
  low-risk given `docker-bake.hcl`'s one-line addition mirrors 60+ existing
  entries).
- Alphabetical placement spot-checked in `services.json`, `docker-bake.hcl`,
  `env-configmap.yaml`, `kustomization.yaml` (base), `db-bootstrap.sh`,
  `routes.conf` — all correctly ordered relative to neighbors.
- `KAFKA_CONSUMER_GROUP` correctly omitted from the main overlay (per brief)
  and correctly present (generator-produced) in the PR overlay's
  `consumer-group-env.yaml`, sourced from Task 1's
  `consumergroup.Resolve("Parcel Service")` — value
  `"Parcel Service [PLACEHOLDER_ATLAS_ENV]"` matches the generator's naming
  convention seen on neighboring entries.

## Not evaluable

- `docker buildx bake atlas-parcel` was not re-run independently in this
  review (relies on the implementer's reported exit 0); the change to
  `docker-bake.hcl` is a single-line, low-risk addition to an established
  list so this is a low-confidence gap, not a suppressed concern.
- The two named operator follow-ups (creating `atlas-parcel-main` on
  postgres.home, flipping the GHCR package public) are correctly out of
  implementer scope per the brief and not evaluable from this worktree.

## Verdict

All in-scope requirements are met: guard clears, topic registration is
exactly the specified three (no `COMMAND_TOPIC_PARCEL` leakage), generated
files are generator-produced and consistent with `routes.conf`, the nginx
route regexes correctly cover both route groups Task 4 shipped, the
`main-b4cf63c` pinned-tag concern has real precedent, and there is no
world-id defaulting. No blocking findings.
