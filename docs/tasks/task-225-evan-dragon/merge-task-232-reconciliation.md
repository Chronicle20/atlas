# Merging main (task-232) into task-225 — what it required

`origin/main` was merged into `task-225-evan-dragon` at `c3c2deb54`, bringing in
task-232 (sparse ephemeral environments, `a3586c14f`). task-225 predates that
task's guards and generators, so the merge left the branch red in several ways.
All of it is fixed in `72d114bac`.

## Merge conflicts (5, resolved in c3c2deb54)

- `deploy/k8s/overlays/main/kustomization.yaml` — took main's `main-a3586c1`
  tag bump for shared services, kept the branch's new `atlas-dragons` entry.
- `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml` — took
  main's `ATLAS_DB_NAMES`; took main's `ATLAS_SERVICES` and re-inserted
  `atlas-dragons`.
- `docs/packets/audits/{STATUS.md,status.json}` — generated; regenerated.
- `services/atlas-configurations/.../socket/corpus_test.go` — union of both
  binding sets: 3281 (main) + 24 (task-225 dragons) = **3305**. Confirmed by
  running the test, not by arithmetic alone.

## task-232 reconciliation (72d114bac)

Branch-side, caused by task-225 being written before task-232:

| Guard | Finding |
|---|---|
| `env-bootstrap-guard` | `atlas-dragons` called `service.Bootstrap` without `WithEnvironmentRegistry` |
| `rediskeyguard` (D7) | `dragon/registry.go` used the withdrawn bare `NewRegistry`/`NewKeyedSet` |
| `service-name-guard` | `atlas-dragons` Deployment had no `SERVICE_NAME` |
| `pr-sparse-mirror-guard` | task-225's dragons block desynced `pr-sparse/patches/consumer-group-env.yaml` |

The Redis migration is the substantive one: `Registry`→`TenantRegistry`,
`KeyedSet`→`TenantKeyedSet`, and the hand-rolled tenant segment removed from
`storeSuffix`/`fieldSuffix` because the tenant-scoped types prefix keys
themselves. Nine call sites, all inside `registry.go`; every method already
took `t tenant.Model`. Safe to change the key layout — the service is unreleased.

Inherited from main, but blocking this branch's gates so fixed here:

- `deploy/k8s/base/atlas-families.yaml` had no `SERVICE_NAME`. task-227
  (#1370) added the service after task-232 landed. **`main` is red on
  `service-name-guard` today, independent of this branch.**
- `routes.conf.template.generated` / `ns-vars.generated.yaml` were stale.
  task-232 reworked `gen-routes.sh` to resolve each service via a per-service
  `NS_ATLAS_*` variable instead of a hardcoded `${POD_NAMESPACE}`; the dragons,
  families and npc-conversations routes still had the old form, which routes a
  sparse ephemeral environment's traffic into the baseline namespace.

## packet-matrix CI failure

`toolTreeSHA()` (`tools/packet-audit/cmd/matrix.go:491`) hashes
`git ls-tree -r HEAD tools/packet-audit` — **HEAD, not the working tree**. The
regeneration during the merge ran before the merge was committed, so it recorded
the pre-merge tool tree and CI flagged both files stale. Regenerating after the
merge commit fixes it. Anyone regenerating the matrix mid-merge will hit this.

## State / next step

Green: `env-bootstrap-guard`, `env-domain-guard`, `scope-guard`,
`producer-seam-guard`, `pr-sparse-mirror-guard`, `service-name-guard`,
`gen-routes.sh --check`, `gen-tenant-tables.sh --check`,
`go-analyzer-guards.sh` (88 modules), `packet-audit matrix --check`,
`atlas-dragons` build + tests, `atlas-configurations` socket tests.

**Not yet run: the flagless `tools/verify.sh`.** It selects 88 modules and 65
docker bakes here (shared-lib fan-out via `go.work`), so it is a long run and
has to pass before this branch is PR-ready.
