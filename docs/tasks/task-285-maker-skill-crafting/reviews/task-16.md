# Task 16 Review — `atlas-maker` build, deploy, and ingress registration

Commit range: `c7fa78fa0..b635cf961` (single commit `b635cf961`)
Brief: `.superpowers/sdd/plan/task-16-brief.md`
Report: `.superpowers/sdd/plan/task-16-report.md`

## Scope

Reviewed the full diff (17 files, +128/-3) plus the files it required to
verify against: `docs/adding-a-new-service.md`, `libs/atlas-kafka` topic
fallback comment (referenced, not touched), `deploy/k8s/overlays/pr/scripts/
gen-db-name-suffix.sh`, and `deploy/k8s/overlays/pr-sparse/` (not touched by
the diff, but load-bearing — see Finding 1). Also read Task 15's
`services/atlas-maker/atlas.com/maker/main.go` to independently confirm the
two overridden-instruction claims.

## Findings

### Blocking

**1. `deploy/k8s/overlays/pr-sparse/patches/db-name-suffix.yaml` has no
`atlas-maker` entry — the branch's own `verify.sh` already caught this and
the ledger already recorded the failure.**

- `docs/tasks/task-285-maker-skill-crafting/agent-ledger.tsv:52` records, at
  this exact commit: `2026-08-29T12:24:14Z agent Task 16 gate ... FAIL
  b635cf961`.
- The saved gate log `/tmp/t285-gate-s6b.log:78-82,116,118` shows the actual
  failure: `sparse-baseline-scoping-guard: FAIL - 1 of 38 DB_NAME values are
  not baseline-suffixed, e.g. atlas-maker=atlas-maker` → `✗ sparse baseline
  scoping FAILED` → `1 check(s) FAILED — the branch is not ready.`
- Root cause, confirmed directly: `deploy/k8s/overlays/pr-sparse/patches/
  db-name-suffix.yaml` is itself generated output (`deploy/k8s/overlays/pr/
  scripts/gen-db-name-suffix.sh:1-21` documents `Usage: gen-db-name-suffix.sh
  [OVERLAY] [SUFFIX_TOKEN]` with a second mode `pr-sparse /
  PLACEHOLDER_BASELINE_ENVIRONMENT — sparse mode: this env shares the
  BASELINE's databases … Leaving base's unsuffixed DB_NAME to stand does NOT
  do that`). Grepping the committed file for `name: atlas-maker` returns
  nothing (confirmed with `grep -n -B2 -A12 "name: atlas-maker"
  deploy/k8s/overlays/pr-sparse/patches/db-name-suffix.yaml`, exit 1).
  Re-running `gen-db-name-suffix.sh pr-sparse PLACEHOLDER_BASELINE_ENVIRONMENT`
  in a scratch check (reverted, tree left clean) adds exactly the missing
  13-line `atlas-maker` document with `DB_NAME:
  "atlas-maker-PLACEHOLDER_BASELINE_ENVIRONMENT"` — proving this is the fix
  and that nothing else needs to change.
- Effect: any `pr-sparse` (shared-baseline) ephemeral environment for
  `atlas-maker` renders `DB_NAME=atlas-maker` unsuffixed, which is Trap 4
  from the brief (`SQLSTATE 3D000` crash-loop) reproduced in a third overlay
  the brief's Step 4 never named and `docs/adding-a-new-service.md` never
  mentions (`grep -n "pr-sparse" docs/adding-a-new-service.md` returns
  nothing — the authority doc itself has a gap here, which is exactly the
  class of blind spot this task exists to close).
- This is not a hypothetical: it is a currently-failing gate on this exact
  commit, already on record in the task's own ledger. The unit is not done.

### Non-blocking

**2. GHCR tag pin (`main-b284bce`) does not exist for `atlas-maker` yet and
cannot until the branch merges and CI runs its first build.** The report
flags this itself (task-16-report.md:75-90) as a self-healing, not a manual,
step — the reasoning is sound (`main-publish.yml`'s bump job only rewrites
entries for services rebuilt in that CI run, and merging this branch
triggers exactly that rebuild). Not a defect in the unit, but worth the
controller's explicit sign-off since it's a deviation from the brief's Step
3 instruction to confirm the tag exists via `docker manifest inspect`
pre-commit.

## The four silent-failure traps — verified directly

1. **`images:` entry in both overlays.** Confirmed present with correct
   `name:`/`newTag:` shape in both:
   `deploy/k8s/overlays/main/kustomization.yaml` (`newTag: main-b284bce`) and
   `deploy/k8s/overlays/pr/kustomization.yaml` (`newTag: latest`) — verified
   with `yq '.images[] | select(.name == "ghcr.io/chronicle20/atlas-maker/
   atlas-maker")'` against both files, both returned a match. PASS.
2. **`configMapGenerator: replace` env-key drop.** Not applicable — Task 15's
   `main.go` (read in full) has zero Kafka topic references
   (`grep -ni "kafka\|consumerGroup\|topic"` → no match), so there are no
   topic keys that could be dropped between base and either overlay's
   `atlas-env` generator. Confirmed by rendering `deploy/k8s/base`,
   `deploy/k8s/overlays/main`, and `deploy/k8s/overlays/pr` and diffing the
   `atlas-maker` Deployment's env block by hand: the two DB_NAME/ATLAS_ENV
   overrides are the only differences from base, both intentional. PASS
   (N/A, correctly recorded as N/A rather than silently skipped).
3. **Missing topic env var silent fallback.** N/A for the same reason as #2
   — no topics exist for this service yet. PASS (N/A).
4. **Unsuffixed `DB_NAME` crash-loop.** PASS for `main` and `pr`:
   `kubectl kustomize deploy/k8s/overlays/main` renders
   `DB_NAME: atlas-maker-main`; `kubectl kustomize deploy/k8s/overlays/pr`
   renders `DB_NAME: atlas-maker-PLACEHOLDER_ATLAS_ENV` (correctly
   templated for per-PR substitution). **FAIL for `pr-sparse`** — see
   Finding 1. This is the same trap the brief names, reproduced in the one
   overlay nobody enumerated.

## Two overridden-instruction claims — independently confirmed

**(a) Main-overlay patches are hand-maintained, not generator-owned.**
Confirmed. `docs/adding-a-new-service.md:63-64`: "Unlike the **main** overlay
(§3), whose patches are all hand-maintained, four PR-overlay pieces are
script-generated." No generator script exists under
`deploy/k8s/overlays/main/` (`ls deploy/k8s/overlays/main/scripts` → no such
directory) and neither `patches/db-name-suffix.yaml` nor
`patches/atlas-env-env.yaml` carries a "generated" header, unlike their
`pr`/`pr-sparse` counterparts which explicitly say `# Generated by
deploy/k8s/overlays/pr/scripts/gen-db-name-suffix.sh …`. The brief's
addendum list mislabeled these two files; the implementer correctly followed
the doc over the brief per the standing "the doc is the authority" ruling.
Confirmed correct.

**(b) Task 15's `main.go` has no `consumerGroupId` — no Kafka consumers
wired yet.** Confirmed by reading `services/atlas-maker/atlas.com/maker/
main.go` in full (65 lines): it bootstraps a DB connection (unused, commented
as reserved for Tasks 17-18) and a REST server with only a readiness route;
no `AddRouteInitializer` wires a Kafka consumer, and no `consumerGroupId`
string appears anywhere. `git diff` confirms
`deploy/k8s/overlays/pr/patches/consumer-group-env.yaml` has zero changes in
this commit — consistent with the generator's own detection regex correctly
finding nothing to add. Confirmed correct; the absent consumer-group patch
entry is not a skipped step.

## README move (controller addendum Step 0)

`git mv services/atlas-maker/atlas.com/maker/README.md
services/atlas-maker/README.md` — confirmed via `git diff --stat` (pure
rename, 0 insertions/deletions). The moved README (read in full) contains no
relative markdown links at all, so nothing could break from the move.
`.github/config/services.json`'s `module_path` still correctly points at
`services/atlas-maker/atlas.com/maker` (the actual Go module root, unaffected
by the README move). The only remaining references to the old README path
are in `docs/tasks/task-285-maker-skill-crafting/plan.md` — a historical
planning artifact, not a live link, out of this unit's scope to fix.

## Mechanical checks — re-run directly, not trusted from the report

```
$ tools/service-registration-guard.sh
service-registration-guard: clean          (exit 0)

$ tools/gen-routes.sh --check
gen-routes: up to date                     (exit 0)

$ kubectl kustomize deploy/k8s/overlays/main   → exit 0, only pre-existing
  commonLabels-deprecated warning
$ kubectl kustomize deploy/k8s/overlays/pr     → exit 0, same pre-existing warning
```

Both match the report's claims. **Not independently re-run**: `docker buildx
bake atlas-maker` (long-running, not required to confirm the registration
defect already proven via the saved gate log) and `go build/test` (no Go
files changed in this diff; Task 15 already covers that surface).

## Other rows of `docs/adding-a-new-service.md` — spot-verified

- `.github/config/services.json` entry shape matches the doc's required
  keys (`name`, `type: go-service`, `path`, `module_path`, `docker_image`,
  `docker_context: "."`) — confirmed via diff.
- `docker-bake.hcl`'s `go_services` list — entry present, alphabetically
  placed.
- Base manifest: no `namespace:`, `DB_NAME: "atlas-maker"` unsuffixed,
  container name `maker`, `atlas.seed-catalog: "true"` label present and
  the seed-catalog sidecar's JSON patch was proven to actually apply (the
  report's self-caught `volumes: []`/`volumeMounts: []` fix, confirmed
  present in the committed file and confirmed necessary by the successful
  `kubectl kustomize` render above).
- `deploy/shared/routes.conf` + `tools/gen-routes.sh` regeneration of
  `deploy/k8s/base/routes.conf.template.generated` and (correctly, per the
  implementer's own investigation) `deploy/k8s/base/atlas-ingress.yaml`'s
  `NS_ATLAS_MAKER` env var and `deploy/k8s/base/ns-vars.generated.yaml` —
  all three are consistent and `--check` passes.
- `tools/db-bootstrap.sh` — unsuffixed `atlas-maker` added to `DBS`.
- `ATLAS_DB_NAMES` in `deploy/k8s/overlays/pr/kustomization.yaml` and the
  regenerated `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`
  — both updated, both include `atlas-maker`.

All rows above pass. The one row that fails is `pr-sparse`'s DB-name-suffix
patch (Finding 1), which sits outside every list the brief or
`docs/adding-a-new-service.md` enumerates — which is itself evidence the
authority doc needs a `pr-sparse` row added, separately from fixing this
unit.

## Not evaluable

- `docker buildx bake atlas-maker` — not re-run (long build, no reason to
  doubt the report's claim given the Dockerfile/module surface is untouched
  by this diff and was already proven to build in Task 15).
- Whether `main-b284bce` remains the fleet's dominant tag at merge time —
  time-sensitive external fact, not verifiable from the repo tree.

## Verdict rationale

The unit does almost everything the brief and the authority doc require, and
the two places where the implementer diverged from explicit brief/ruling
text were verified correct against the doc. But the branch's own `verify.sh`
gate already failed on this exact commit for a real, silently-crash-looping
gap (`pr-sparse` DB_NAME unsuffixed) that this task exists specifically to
prevent from reaching main. This is not a matter of interpretation — it is a
recorded FAIL in the task's own ledger, reproduced and root-caused directly
against the tree. The task is not done.
