# Review: Task 13 — runbook

Commit: `e32c9ef15` (docs-only, `docs/runbooks/sparse-environments.md`, 155
insertions, 0 deletions).

Brief: `.superpowers/sdd/plan/task-13-brief.md`
Report: `.superpowers/sdd/plan/task-13-report.md`

Review bar per the dispatch: factual accuracy, not code style. Every command,
flag, path, and behaviour claim was checked against the source it names, not
against `design.md`'s intent, except where the brief explicitly redirected
verification to `design.md`/`plan.md` (Tasks 9/11/12's in-flight files).

## Scope

Reviewed the full diff (all 155 added lines, 7 new headings). Cross-checked
every factual claim against:

- `services/atlas-configurations/atlas.com/configurations/servicesuniq/migration.go`
- `services/atlas-configurations/atlas.com/configurations/services/{entity,processor}.go`
- `services/atlas-configurations/atlas.com/configurations/services/service/rest.go`
- `services/atlas-configurations/atlas.com/configurations/services/resource.go`
- `services/atlas-configurations/atlas.com/configurations/main.go`
- `libs/atlas-env/env.go`, `libs/atlas-env/registry.go`
- `libs/atlas-service/envregistry.go`
- `tools/derive-service-id.sh`
- `deploy/k8s/base/atlas-login.yaml`, `deploy/k8s/base/atlas-channel.yaml`
- `docs/tasks/task-243-sparse-env-scoping-activation/prd.md`,
  `design.md` (§4.3, §5.3, §5.4, decisions table, requirements list) — used
  as the source of truth for the activation/binding behaviour still owned by
  Tasks 9/11/12's in-flight files, per the controller's redirect.

Per the controller's instruction, I did not read Tasks 9/11/12's in-flight
`bootstrap.sh` / pr-sparse overlay / `pr-validation.yml` — matching the
implementer's own stated scope discipline.

## Findings

### Blocking

1. **`docs/runbooks/sparse-environments.md:355`** — "`services.Migration`
   builds a unique index on `services (type, environment)` at
   `atlas-configurations` startup" is factually wrong. `services.Migration`
   (`services/atlas-configurations/atlas.com/configurations/services/entity.go:11`)
   is `db.AutoMigrate(&Entity{}, &HistoryEntity{})` — it does not create any
   index. The unique index (`CREATE UNIQUE INDEX IF NOT EXISTS
   idx_services_type_env ON services (type, environment)`) is built by
   `servicesuniq.Migration`
   (`services/atlas-configurations/atlas.com/configurations/servicesuniq/migration.go:115`),
   run after `services.Migration` and `environmentcol.Migration` per
   `main.go:50-53`. The runbook's own §4.3-Layer-3 pre-flight section two
   paragraphs later correctly calls out `servicesuniq.Preflight` by its real
   package — so the mistake is isolated to the opening sentence of the D9
   section, not systemic, but it is exactly the sentence an operator would
   use to go find the migration and confirm the crash-loop mechanism before
   a production rollout. It misattributes the failure to a function that
   provably cannot produce it. `design.md`'s own §4.3 opening paragraph is
   loosely worded in the same direction (it introduces `services.Migration`
   and the crash-loop risk in the same breath before naming Layer 2 as the
   actual index-builder), so this looks like the implementer inherited the
   ambiguity from the design doc rather than inventing it fresh — but the
   review bar here is the committed source, and the committed source
   attributes the index unambiguously to `servicesuniq.Migration`.

### Non-blocking

None beyond the one blocking item — the rest of the new prose checked out
line for line against the source it names (see verification detail below).

## Verification detail (PASS unless noted above)

- **D9 teardown claim** (`cleanup.sh`'s environment-scoped reclaim goes
  through `ProcessorImpl.DeleteById`, which enqueues a tombstone): confirmed
  against `services/processor.go:222-229` —
  `DeleteById` runs `delete` and `enqueueServiceStatus(db, serviceId, nil)`
  inside one transaction, matching "nil-value tombstone." Matches
  `design.md:253-255` (§4.3 Layer 1) exactly, which is the correct
  verification source for `cleanup.sh` per the controller's redirect (Tasks
  9-12 own that file in-flight).
- **"Five `login-service` rows and five `channel-service` rows"**: matches
  `prd.md:123` verbatim — not invented.
- **Pre-flight SQL** (`SELECT type, environment, COUNT(*) ... HAVING
  COUNT(*) > 1`): matches `servicesuniq/migration.go:50-52`
  (`Preflight`) exactly, including column names.
- **Dedupe tiebreak prose** (derived id, else newest
  `service_history.created_at`, else lowest id): matches
  `resolveGroup` in `servicesuniq/migration.go:187-249` exactly, in order.
- **`login-service` co-resident-row example**: matches
  `prd.md:215` and `design.md:290` — not invented, correctly generalized as
  "the PRD records as an accepted multi-tenant case."
- **`ATLAS_ENVIRONMENT` precondition** (`IsOwner` reads `r.self`, `r.self`
  does not update on a ConfigMap-only change because it consumes
  `envFrom`): `libs/atlas-env/env.go:80` (`Self()` reads `os.Getenv(SelfVar)`
  live) plus `libs/atlas-service/envregistry.go:53`
  (`env.NewMapRegistry(env.Self(), time.Now)`, called once at process
  construction) together confirm `r.self` is fixed at pod start, so a
  ConfigMap update alone does not change the running value until the pod
  restarts — exactly the runbook's claim. `libs/atlas-env/registry.go:205-218`
  confirms `IsOwner` compares against `r.self`. Matches `design.md:366-375`
  (§5.4) closely, including the "PRD non-goal / separate defect" framing.
- **`kubectl ... jsonpath` check for `ATLAS_ENVIRONMENT`**: syntactically
  valid jsonpath for the described purpose; no repo command is asserted
  wrong here (it's an ad hoc kubectl invocation, not a repo script).
- **`tools/derive-service-id.sh <service-type> <environment>` usage**:
  confirmed the script exists at that path and takes exactly those two
  positional args (`tools/derive-service-id.sh:1-20`). The implementer's
  report explicitly notes it used the real committed path rather than
  `design.md`'s sketched location — verified true; the script is at
  `tools/derive-service-id.sh`, not `deploy/k8s/overlays/pr/scripts/`.
- **`GET /api/configurations/services/{id}` + `attributes.environment`**:
  route confirmed at
  `services/atlas-configurations/atlas.com/configurations/services/resource.go:22`
  (`/configurations/services/{serviceId}`, GET); JSON attribute name
  confirmed as `environment` via `json:"environment"` tags in
  `services/service/rest.go:20,66,94`, exercised by
  `TestGenericRestModel_JSONMarshal_EnvironmentAttribute` et al.
- **Consumer-group verification cross-reference**: the "Verifying a consumer
  group is seeded (FR-4.9, FR-5.3)" section it points to exists at
  `docs/runbooks/sparse-environments.md:125` (pre-existing, untouched by this
  diff) — the cross-reference target is real.
- **FR-4.4 citation for a `-` offset column**: `prd.md:167` names FR-4.4 as
  exactly the replay/loss failure mode this describes ("Activating without
  FR-4.3 MUST NOT be shipped... replaying the whole retention window or
  losing everything produced before first poll") — correctly cited, not the
  adjacent FR-4.9/FR-5.3 the pre-existing section header uses.
- **Readiness probe values**: `initialDelaySeconds: 10, periodSeconds: 10,
  failureThreshold: 30` confirmed verbatim in both
  `deploy/k8s/base/atlas-login.yaml:43-45` and
  `deploy/k8s/base/atlas-channel.yaml:43-45` (Task 7's committed change).
  "5-minute catch-up budget" is `10 + 10*30 = 310s ≈ 5.17 min`, a reasonable
  rounding, not a fabricated number.
- **No live measurement claim**: the report's flagged judgement call is
  handled correctly. The new "Readiness probe timing" section states "No
  live measurement has been recorded here yet" and gives instructions for
  recording one later, rather than inventing a number. This is exactly the
  posture the review dispatch asked me to confirm, and it holds — no number
  is asserted as observed anywhere in the new prose.
- **D8 activation-window content**: matches `design.md:350-364` (§5.3)
  point for point — never processed twice, dropped-or-baseline-processed,
  empty in practice because ingress does not route during `PROVISIONING`.
- **D8/D9 decision-table labels**: `design.md:82-83` confirms D8 = the
  activation window, D9 = teardown-and-recreate, matching the runbook's
  section headers.
- **FR-1.1–FR-1.4 range**: matches `prd.md:101-111`, the binding
  requirements the "Verifying a service-config binding" section title
  claims to cover.
- **Heading structure**: new `## ` / `### ` headings are consistent in level
  and style with the pre-existing document (`grep -n "^## \|^### "` shows a
  uniform pattern before and after the diff boundary at line 349).
- **CLAUDE.md conventions**: `grep -n "/home/\|/Users/"` — no match, no
  literal home/absolute path in the new prose. `CR count: 0` on the whole
  file — LF preserved throughout, no CRLF introduced. Diff is pure addition
  (155 insertions, 0 deletions) — no existing line was touched, so no
  pre-existing line ending could have been altered.

## Not evaluable

None. The full diff surface was checked against real source; nothing in
scope required a live rollout to confirm except the readiness-timing number,
which the implementer correctly declined to fabricate (see above) rather
than leaving it silently unverifiable.

## scope_confirmed

Reviewed the full 155-line diff of `docs/runbooks/sparse-environments.md`
(commit `e32c9ef15`) against the source files and repo tooling it describes,
and against `design.md`/`prd.md` for the activation/binding behaviour still
owned by Tasks 9/11/12's in-flight working tree, per the controller's
redirect. No scope mismatch — the commit is exactly the one file the brief
named, docs-only, no code touched.
