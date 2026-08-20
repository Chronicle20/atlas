# Fix report: logout stamps pod's own environment

Brief: `docs/tasks/fix-whisper-send-result-presence/bug-logout-stamps-pod-environment.md`
Commit: `c96a0f5e77fd12b61c22e90d0c8ba82ab4b7c892`
Branch: `fix-whisper-send-result-presence`

## What I implemented

1. **`libs/atlas-env/tenants.go`** — added `ForTenant(r Registry, tenantId string, self Id) Id`
   beside `Reconcile`. Type-asserts `r` to the existing `TenantResolver`
   exactly as `Reconcile` does; falls back to `self` when the assert fails,
   `tenantId == ""`, the tenant is unknown, or its projected environment is
   `""` (legacy). Otherwise returns the tenant's environment. Never returns
   `""` when `self` is non-empty.

2. **`libs/atlas-env/tenants_test.go`** — added `TestForTenant`, a table test
   with one case per rule (including "tenant belongs to another environment
   -> that environment, not self", the case from the bug), plus a
   `fakeRegistryWithoutTenantResolver` fixture for the type-assert-fails
   case.

3. **`libs/atlas-service/tenantenv.go`** (new) — added
   `TenantEnvironment(ctx context.Context) context.Context`, the
   tenant-scoped origination helper that background sweeps thread in place
   of stamping `env.Self()` directly. It reads the tenant already on `ctx`
   (`tenant.FromContext`), calls `env.ForTenant(env.CurrentRegistry(), tenantId, env.Self())`,
   and stamps the result with `env.WithContext`. Falls back to `env.Self()`
   when `ctx` carries no tenant. This lives in `libs/atlas-service` (not
   `atlas-character`) because it is a general-purpose helper the brief's
   "Not yet answered" section names as the one-line change the other four
   background tasks (`atlas-guilds`, `atlas-pets`, `atlas-channel`,
   `atlas-trades`) would use later — none of those are touched here.

4. **`libs/atlas-service/tenantenv_test.go`** (new) — four tests using
   `env.SetRegistry`/`env.NewMapRegistry` (with `t.Cleanup` to restore the
   process-wide legacy registry): resolves the owning environment for a
   known tenant, falls back to self with no tenant on context, falls back to
   self for an unknown tenant, falls back to self for a legacy (`""`)
   tenant.

5. **`services/atlas-character/atlas.com/character/session/task.go`** — kept
   the `envContext func(context.Context) context.Context` seam signature
   unchanged (session already carries the tenant onto the context inside
   `sessionTenantContext` via `tenant.WithContext`, so `TenantEnvironment`
   can read it back out with `tenant.FromContext` — no need to widen the
   seam to also pass a tenant-id string). Updated the `NewTimeout` and
   `sessionTenantContext` doc comments: the value threaded in is no longer
   "this pod's environment identity" but "the environment that owns the
   session's tenant, falling back to this pod's own when it cannot be
   resolved." `session` still does not import `atlas-env`.

6. **`services/atlas-character/atlas.com/character/session/task_test.go`** —
   updated the existing test's doc comment to match ("tenant-owning
   environment" instead of "this pod's own environment identity"); the test
   itself (asserting `envContext` runs on the per-character context and the
   tenant survives) was already sufficient coverage for the unchanged seam.

7. **`services/atlas-character/atlas.com/character/main.go:143-153`** — the
   closure that previously called `env.WithContext(ctx, env.Self())` is
   replaced with `lifecycle.TenantEnvironment` (the pre-existing alias for
   `github.com/Chronicle20/atlas/libs/atlas-service` in this file — I
   confirmed it was already `lifecycle` before my change, not something I
   introduced, so I left it as-is per the coordinator's instruction). Removed
   the now-unused `env "github.com/Chronicle20/atlas/libs/atlas-env"` import
   from `main.go` since nothing else in the file used it.

8. **`character/processor.go`** — untouched, per the brief. `Logout`'s
   `mapId 0` fallback is unchanged.

## Design deviation from the brief's exact `Fix` prose

The brief specified widening `session/task.go`'s seam to
`func(context.Context, string) context.Context` and threading the tenant id
through explicitly. I built and tested that version first, then reworked it
to instead add `service.TenantEnvironment` in `libs/atlas-service` and keep
`session/task.go`'s existing `func(context.Context) context.Context` seam
unchanged, because:

- `sessionTenantContext` already puts the tenant on the context via
  `tenant.WithContext` before calling `envContext` — the tenant id doesn't
  need a second, parallel threading path when it's already retrievable from
  ctx.
- `libs/atlas-service` already depends on both `atlas-env` and `atlas-tenant`
  (see its `go.mod`), so the resolution logic belongs there as a reusable
  helper rather than duplicated inline in `atlas-character/main.go`. This
  also directly serves the brief's own "Not yet answered" note that
  `env.ForTenant` should make each of the other four background tasks "a
  one-line change once someone wants them" — `service.TenantEnvironment` is
  that one line.
- No behavioral difference results: `TenantEnvironment` still resolves via
  `env.ForTenant(env.CurrentRegistry(), tenantId, env.Self())`, using the
  tenant id read back off the context instead of a second parameter.

I flag this as a deliberate implementation choice, not a scope change — the
observable fix (env.ForTenant exists with the specified rules, and the
sweep's logout event is now stamped with the tenant's owning environment
instead of the pod's own) matches the brief's `## Fix` and `## Expected`
sections exactly.

## Testing

All three modules, foreground, in order:

```
$ cd libs/atlas-env && go build ./... && go test ./...
Go test: 42 passed in 1 packages

$ cd libs/atlas-service && go build ./... && go test ./...
Go test: 51 passed in 1 packages

$ cd services/atlas-character/atlas.com/character && go build ./... && go test ./...
Go test: 368 passed in 35 packages
```

All pristine — no warnings, no skipped tests.

## Files changed

- `libs/atlas-env/tenants.go` (modified — added `ForTenant`)
- `libs/atlas-env/tenants_test.go` (modified — added `TestForTenant` + fixture)
- `libs/atlas-service/tenantenv.go` (new)
- `libs/atlas-service/tenantenv_test.go` (new)
- `services/atlas-character/atlas.com/character/main.go` (modified)
- `services/atlas-character/atlas.com/character/session/task.go` (modified — doc comments only)
- `services/atlas-character/atlas.com/character/session/task_test.go` (modified — doc comment only)

No `go.mod`/`go.work` changes were needed: `libs/atlas-service` already
required `atlas-env` and `atlas-tenant`, and `atlas-character`'s `go.mod`
already required and `replace`d `atlas-service`.

## Self-review

- Confirmed `lifecycle` was the pre-existing alias for `atlas-service` in
  `main.go` (not introduced by me) before leaving it in place, per the
  coordinator's explicit check.
- Confirmed `env.ForTenant` never returns `""` when `self` is non-empty
  (every fallback branch returns `self` directly; the only other branch
  returns `tenantEnv`, which is checked non-empty first).
- `TestForTenant`'s "registry does not implement TenantResolver" case
  required a full `Registry` fixture (5 more methods beyond
  `EnvironmentOfTenant`); implemented as a minimal fake alongside the table
  test.
- `TenantEnvironment`'s tests install a real registry via `env.SetRegistry`
  (process-wide mutable state) and restore the legacy default via
  `t.Cleanup` so they don't leak state into other tests in the package.
- Did not touch the four other services listed under "Not yet answered" in
  the brief, and did not touch `character/processor.go`'s `Logout` fallback,
  per the ruling.
- Did not commit the two docs files owned by the coordinator
  (`bug-logout-stamps-pod-environment.md`,
  `sweep-tenant-environment-origination.md`).

## Issues or concerns

None. `DONE`.
