# Merging main (task-232) into task-231

## What was found

`origin/main` at `b4cf63cfa` carries task-232 (sparse ephemeral environments,
`a3586c14f`, 1346 files). atlas-events was written before those invariants
existed, so the merge left the branch buildable but non-conformant.

The merge itself had six conflicts, all mechanical: four additive list/manifest
conflicts (main-overlay image tags, `db-name-suffix.yaml`, the `ATLAS_DB_NAMES`
/ `ATLAS_SERVICES` literals in the pr overlay and the cleanup ConfigMap) resolved
as unions, plus two end-of-file test/export additions in
`services/atlas-ui/src/services/api/index.ts` and
`services/atlas-monsters/.../monster/registry_test.go`. Only the last had real
content: main's two new tenant-scoping tests call `CreateMonster` with the
pre-branch arity, since this branch added the two spawn-source arguments.
Fixed by passing `"", ""`, the convention already used at all 20 other call
sites in that file.

## What it means

Four task-232 guards failed on atlas-events (`env-bootstrap-guard`,
`service-name-guard`, `scopeguard` entity + call-site rules) and one gap no
guard catches. All are fixed in `4a1048d9f`; see that commit message for the
per-gap rationale. The two that were genuine correctness bugs rather than
conformance items:

1. **The scheduling poller had no environment dimension.** It is deliberately
   cross-tenant by design (§4.2). Once a PR environment shares the database, a
   baseline pod would claim a PR row into PROCESSING under its own instanceId
   and either execute it against the wrong deployment's handlers or strand it
   for a full lease. Fixed with an injected `TenantOwnership` predicate applied
   *before* the claiming UPDATE and inside Reclaim. Injected from `main.go`
   because `libs/atlas-env` may not be imported from a domain package
   (NG5/FR-4.5) — the same reason atlas-saga-orchestrator resolves ownership in
   its own `main.go`.

2. **Outbound REST used `requests.RootUrl`**, which returns `BASE_SERVICE_URL`
   unconditionally. From an ephemeral-environment pod every MAPS / TRANSPORTS /
   TENANTS call went to main's ingress — precisely the leak FR-3.5/G4 exists to
   close. Converted to `requests.RootUrlFor(ctx, ...)`.

## Known remaining gap (pre-existing on main, NOT introduced here)

Five other services still call the context-free `requests.RootUrl` and have the
same outbound-routing leak: atlas-channel (`data/commodity`, `events`,
`pendingchange`, `dragon`), atlas-character (`configuration`, `pending_change` —
12 call sites), atlas-dragons (`character`), atlas-pets (`data/cash`), and
atlas-saga-orchestrator (`pending_change`). Most arrived in task-225/task-227,
after task-232's sweep. This is main's debt, deliberately left untouched by this
branch. Worth its own task — and worth a guard, since nothing currently prevents
a new `RootUrl` call site from being added.

## Next step

Run the flagless `tools/verify.sh` (the merge changed `go.work`, so it fans out
to all 89 modules and bakes 66 images — budget accordingly), then the pre-PR
code review. No further code work is known to be outstanding.
