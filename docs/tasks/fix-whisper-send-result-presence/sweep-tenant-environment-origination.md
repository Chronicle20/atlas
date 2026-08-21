# sweep: tenant-derived environment origination in the four sibling background tasks

Companion to `bug-logout-stamps-pod-environment.md`. Read that file first — it
carries the live evidence and the root cause. This file is only the inventory
for the four remaining sites, which the user ruled in scope on 2026-08-20.

## The defect, restated

Each site below runs a periodic, tenant-scoped sweep with no request or message
to inherit `ENVIRONMENT` from, so `main.go` threads in an origination closure.
Every one of those closures stamps **this pod's own** environment
(`env.Self()`). For a baseline pod doing work on behalf of a *sparse*
environment's tenant that is the wrong environment, and FR-7.7 rejects the
disagreement outright: the REST call 400s
(`libs/atlas-rest/server/handler.go:151`) and the Kafka emit is dropped at the
ownership gate with `reason="mismatched"`
(`libs/atlas-kafka/consumer/gate.go:68`). The work is silently discarded by
every deployment.

Reproduced for `atlas-character`'s logout only. These four are the same code
shape, not separately reproduced — say so in the commit message.

## Prerequisite

This sweep depends on the helper landed by the `atlas-character` fix:

```go
// libs/atlas-service
func TenantEnvironment(ctx context.Context) context.Context
```

It reads the tenant already on `ctx`, resolves that tenant's environment via
`env.ForTenant(env.CurrentRegistry(), …)`, and falls back to `env.Self()` when
`ctx` carries no tenant or the registry does not know it. Do not reimplement
it, and do not change its contract — `atlas-character` depends on it too.

**Every call site below already applies its closure to a context that ALREADY
carries the tenant.** That is what makes each of these a one-line wiring swap:
no seam signature changes, no domain-package edits beyond doc comments.

## The four sites

1. `services/atlas-guilds/atlas.com/guilds/main.go:122-124` — the inline
   closure passed to `guild.NewTransitionTimeout` becomes
   `service.TenantEnvironment`.
   Call site is `guild/task.go` `processExpiredCoordinations`:
   `envContext(tenant.WithContext(ctx, g.Tenant()))`.
   Update the `NewTransitionTimeout` and `processExpiredCoordinations` doc
   comments — both currently say "this pod's own environment identity
   (env.Self())".

2. `services/atlas-pets/atlas.com/pets/main.go:115-117` — the inline closure
   passed to `pet.NewHungerTask` becomes `service.TenantEnvironment`.
   Call site is `pet/task.go` `ownerTenantContext`:
   `t.envContext(tenant.WithContext(sctx, tn))`.
   Update the `NewHungerTask` and `ownerTenantContext` doc comments.

3. `services/atlas-channel/atlas.com/channel/main.go:349` —
   `combo.NewDecayTick(l, rt.Context(), time.Second, socket.WithSelfEnvironment)`
   becomes `…, service.TenantEnvironment)`.
   Call site is `character/combo/task.go` `processExpiries`:
   `envContext(tenant.WithContext(ctx, e.Tenant()))`.
   Update the `NewDecayTick` and `processExpiries` doc comments, both of which
   name `env.Self()` explicitly.
   **Do NOT change `socket.WithSelfEnvironment` itself, nor any other caller of
   it.** The socket paths (`socket/init.go`, `socket/handler/handle.go`,
   `socket.NewListenerContext`) are correct as they stand: a game client's
   connection genuinely belongs to the pod that terminates it. Only the combo
   decay tick's use of that helper is wrong.

4. `services/atlas-trades/atlas.com/trades/main.go:79` —
   `trade.SetEnvContext(withSelfEnvironment)` becomes
   `trade.SetEnvContext(service.TenantEnvironment)`.
   All three `applyEnvContext` call sites in `trade/settlement.go` (lines 198,
   1296, 1409) already wrap `tenant.WithContext(…)`, so the tenant is present
   in every one.
   Delete the now-unused `withSelfEnvironment` function and its doc comment
   (main.go:30-42) rather than leaving it dead, and drop the `atlas-env` import
   if nothing else in that file uses it. Update the
   `// --- environment origination ---` block comment in `settlement.go`
   (lines 71-89), which describes the old semantics.

## Tests

For each service, extend the existing test that already exercises the
origination seam — every one of these packages has one, because a nil
`envContext` is documented as a caller bug and the pure sweep functions
(`processExpiredCoordinations`, `ownerTenantContext`, `processExpiries`,
`applyEnvContext`) were extracted specifically to be testable. Assert that the
seam receives a context carrying the expected tenant. Do not add a new
`*_testhelpers.go`; use the project's Builder pattern.

`service.TenantEnvironment`'s own resolution rules are unit-tested in
`libs/atlas-service` by the `atlas-character` fix — do not duplicate that
coverage per service.

## Verification

Module-local only, one per touched service, plus the shared lib:

```
libs/atlas-service
services/atlas-guilds/atlas.com/guilds
services/atlas-pets/atlas.com/pets
services/atlas-channel/atlas.com/channel
services/atlas-trades/atlas.com/trades
```

`go build ./... && go test ./...` in each. Do not run `tools/verify.sh` — the
repo-wide gate runs separately, in its own context.

## Out of scope

- `env.Self()` in socket/listener paths (`atlas-channel`, `atlas-login`). See
  site 3.
- Making `atlas-maps` idempotent against a missed LOGOUT. Noted in the bug
  file; do not build it.
