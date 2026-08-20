# review: tenant-derived environment origination (commits c96a0f5e7, 9f67a6332)

Range reviewed: `89aea18c9..HEAD` (two commits: `c96a0f5e7` "fix(atlas-character):
resolve tenant-owning environment for the session-timeout logout sweep",
`9f67a6332` "fix(services): derive background sweep environment from tenant,
not pod, in four services").

Requirement documents: `bug-logout-stamps-pod-environment.md` (root cause +
Fix inventory) and `sweep-tenant-environment-origination.md` (the four
sibling sites).

`git diff --stat 89aea18c9..HEAD`:

```
libs/atlas-env/tenants.go                                       | 21 +++++
libs/atlas-env/tenants_test.go                                  | 77 +++++++++
libs/atlas-service/tenantenv.go                                 | 28 +++++
libs/atlas-service/tenantenv_test.go                             | 89 ++++++++++
services/atlas-channel/atlas.com/channel/character/combo/task.go| 28 +++---
services/atlas-channel/atlas.com/channel/main.go                | 17 +--
services/atlas-character/atlas.com/character/main.go             | 18 +--
services/atlas-character/atlas.com/character/session/task.go     | 22 +--
services/atlas-character/atlas.com/character/session/task_test.go|  8 +-
services/atlas-guilds/atlas.com/guilds/guild/task.go              | 28 +--
services/atlas-guilds/atlas.com/guilds/main.go                    |  5 +-
services/atlas-pets/atlas.com/pets/main.go                        | 18 +--
services/atlas-pets/atlas.com/pets/pet/task.go                    | 23 +--
services/atlas-trades/atlas.com/trades/main.go                    | 17 +--
services/atlas-trades/atlas.com/trades/trade/settlement.go        | 19 +--
15 files changed, 317 insertions(+), 101 deletions(-)
```

`go build ./... && go test ./...` ran clean (module-local) in all seven
touched modules: `libs/atlas-env`, `libs/atlas-service`,
`services/atlas-character/.../character`, `services/atlas-guilds/.../guilds`,
`services/atlas-pets/.../pets`, `services/atlas-channel/.../channel`,
`services/atlas-trades/.../trades`.

## 1. `env.ForTenant` (libs/atlas-env/tenants.go)

```go
func ForTenant(r Registry, tenantId string, self Id) Id {
	tr, ok := r.(TenantResolver)
	if !ok || tenantId == "" {
		return self
	}
	tenantEnv, known := tr.EnvironmentOfTenant(tenantId)
	if !known || tenantEnv == "" {
		return self
	}
	return tenantEnv
}
```

PASS on logic: never returns `""` when `self` is non-empty (every fallback
path returns `self` verbatim; the only path that returns something else
returns a non-empty `tenantEnv`). PASS on the legacy asymmetry: an empty
projected environment (`tenantEnv == ""`) falls back to `self`, matching
`Reconcile`'s documented legacy semantics one function up
(`libs/atlas-env/tenants.go:88-94`), not "belongs to nothing." Verified by
`TestForTenant`'s five table cases (`libs/atlas-env/tenants_test.go:142-202`),
including the exact case the bug describes ("tenant belongs to another
environment -> that environment, not self").

**Finding (blocking): `Reconcile`'s doc comment was orphaned onto `ForTenant`,
and `Reconcile` now has no doc comment at all.**

`ForTenant`'s own doc comment (`tenants.go:58-66`) was inserted directly
after `Reconcile`'s pre-existing doc comment (`tenants.go:43-57`, including
the "CAUTION" paragraph) with no blank line between them, and directly
before `ForTenant`'s `func` line — but `Reconcile`'s `func` declaration is
now 21 lines further down with nothing above it. Go's doc-comment adjacency
rule (a comment block is a declaration's doc comment only when it
immediately precedes that declaration, no blank line) attaches the whole
merged block to `ForTenant` and leaves `Reconcile` undocumented. Confirmed
with `go doc`:

```
$ go doc -all . | sed -n '/func ForTenant/,/^$/p'
func ForTenant(r Registry, tenantId string, self Id) Id
    Reconcile resolves the operation's environment from the header and the
    tenant, and returns ErrEnvironmentMismatch when they disagree (FR-7.7).
    ...

$ go doc -all . | sed -n '/func Reconcile/p'
func Reconcile(r Registry, headerEnv Id, tenantId string) (Id, error)
```

`go doc Reconcile` now prints nothing — the CAUTION note (a genuinely
load-bearing warning about the `tenantId == ""` short-circuit being
indistinguishable from an ordering bug) is no longer attached to the
function it warns about. `ForTenant`'s doc now confusingly opens by
describing `Reconcile`. This is a real defect in exactly the pairing the
task brief asked to be verified ("compare the two functions directly") —
`libs/atlas-env/tenants.go:43-67`.

## 2. `service.TenantEnvironment` (libs/atlas-service/tenantenv.go)

```go
func TenantEnvironment(ctx context.Context) context.Context {
	self := env.Self()
	t, err := tenant.FromContext(ctx)()
	if err != nil {
		return env.WithContext(ctx, self)
	}
	return env.WithContext(ctx, env.ForTenant(env.CurrentRegistry(), t.Id().String(), self))
}
```

PASS: falls back to `env.Self()` when `ctx` carries no tenant (`err != nil`
branch), and otherwise resolves via `env.ForTenant`. It never introduces the
disagreement the bug describes: it never touches or requires an
`ENVIRONMENT` header on `ctx` (there is none — these are background sweeps),
it leaves the tenant already on `ctx` untouched, and it stamps either (a)
the tenant's own registered environment (agrees by construction) or (b)
`self` for an unknown/legacy tenant, both of which `Reconcile` on the
consumer side treats as non-mismatching (unknown tenant trusts the header;
legacy/empty-env tenant trusts the header) — so a background emit that goes
through `TenantEnvironment` can never produce the header/tenant disagreement
that `libs/atlas-rest/server/handler.go:151` / `libs/atlas-kafka/consumer/gate.go:68`
hard-reject. Verified by the four cases in
`libs/atlas-service/tenantenv_test.go` (owning environment, no tenant on
ctx, unknown tenant, legacy tenant) — each asserts the actual stamped
environment id via `env.MustFromContext`, not merely "some value was set."

## 3. The five wiring sites — is the tenant on ctx when the closure runs?

Traced each by hand at the point `envContext`/`TenantEnvironment` is
invoked:

1. **atlas-character** `session/task.go:89`:
   `t.envContext(tenant.WithContext(sctx, m.Tenant()))` — tenant applied
   first. PASS.
2. **atlas-guilds** `guild/task.go:77`:
   `envContext(tenant.WithContext(ctx, g.Tenant()))` inside
   `processExpiredCoordinations`. PASS.
3. **atlas-pets** `pet/task.go:63`:
   `t.envContext(tenant.WithContext(sctx, tn))` inside
   `ownerTenantContext`. PASS.
4. **atlas-channel** `character/combo/task.go:83`:
   `envContext(tenant.WithContext(ctx, e.Tenant()))` inside
   `processExpiries`. PASS.
5. **atlas-trades** `trade/settlement.go`, all three `applyEnvContext` call
   sites:
   - `settlement.go:203` — `applyEnvContext(tenant.WithContext(context.Background(), p.t))`
   - `settlement.go:1301` — `applyEnvContext(tenant.WithContext(ctx, t))`
   - `settlement.go:1414` — `applyEnvContext(tenant.WithContext(ctx, r.tenant))`
   All three wrap `tenant.WithContext` first. PASS.

No site falls back to `env.Self()` due to a missing tenant on ctx at
invocation time; all five are genuine one-line closure swaps as the briefs
promised.

## 4. Boundary: `socket.WithSelfEnvironment` and socket/listener paths

PASS: `git diff --stat` shows no changes under
`services/atlas-channel/.../socket/` or any `atlas-login` file.
`socket.WithSelfEnvironment` (`socket/init.go:52`) is unchanged and its only
other callers (`socket.NewListenerContext` at
`services/atlas-channel/atlas.com/channel/main.go:420`,
`socket/handler/handle.go`) were not touched — confirmed by grepping for
`WithSelfEnvironment` post-diff: the only remaining call site outside its
own package/tests is the socket-listener path, and the combo decay tick's
`main.go:349` line now reads `combo.NewDecayTick(l, rt.Context(), time.Second,
service.TenantEnvironment)`. The sweep redirected exactly the one call site
named in the brief.

## 5. Test honesty — new contract vs. "some environment was stamped"

- `libs/atlas-env/tenants_test.go` `TestForTenant` — PASS, asserts the
  resolved `Id` per rule, including the exact bug scenario.
- `libs/atlas-service/tenantenv_test.go` — PASS, asserts the resolved `Id`
  via `env.MustFromContext`, distinguishing owning/unknown/legacy/no-tenant
  cases.
- `services/atlas-character/atlas.com/character/session/task_test.go` — the
  diff here is doc-comment prose only (`git diff --stat`: 8 lines, all
  comment text); the test body is byte-identical to before this branch. It
  asserts `tctx.Value(envMarkerKey) == "stamped"` and
  `tenant.MustFromContext(tctx) == tn` against a spy `envContext` that is a
  pure passthrough (`context.WithValue`). Because the spy never strips
  values, checking the tenant on the *output* is equivalent evidence that
  the tenant was present on the *input* — this is legitimate coverage of
  "tenant already on ctx when envContext runs," even though it predates this
  branch and wasn't extended.
- `services/atlas-trades/atlas.com/trades/trade/settlement_env_test.go` —
  PASS, pre-existing and genuinely strong: the spy `envContext` closures in
  `TestDetachedAppliesEnvContext`, `TestReconcileAppliesEnvContext`,
  `TestReconcileEscrowAppliesEnvContext` call
  `tenant.MustFromContext(c)` *inside* the closure and assert the id
  matches the expected tenant — this directly proves the seam receives the
  tenant, for all three `applyEnvContext` sites. No extension needed; the
  implementer's report is accurate here.
- `services/atlas-pets/atlas.com/pets/pet/task_test.go`
  `TestTimeoutOwnerTenantContextAppliesEnvContext` — same passthrough-spy
  reasoning as the character test: asserts `tenant.MustFromContext(tctx) ==
  tn` on the *output* of a value-preserving spy, which is equivalent
  evidence for the input. Adequate, unchanged by this diff (pre-existing).

**Finding (blocking): the guild and channel-combo origination tests do not
assert tenant receipt at all — they were not extended as the sweep brief
required, and they pass identically before and after this fix.**

- `services/atlas-guilds/atlas.com/guilds/guild/task_test.go`
  `TestProcessExpiredCoordinationsAppliesEnvContextToAct` (unchanged by this
  diff — not in the file list) only checks
  `gotMarker := ctx.Value(envMarkerKey("marker"))` on the context `act`
  receives. It never calls `tenant.MustFromContext` anywhere, on input or
  output. It cannot distinguish "envContext ran on a context that already
  had the tenant" from "envContext ran on a bare context and the tenant was
  never there" — the assertion is purely about the spy's own marker
  round-tripping through `processExpiredCoordinations`'s plumbing, which is
  identical regardless of what `envContext` actually does with tenants. This
  test passed before `c96a0f5e7`/`9f67a6332` and passes after; it is not
  coverage of the fix.
- `services/atlas-channel/atlas.com/channel/character/combo/task_test.go`
  `TestProcessExpiriesAppliesEnvContextToCancel` (unchanged by this diff)
  has the identical shape and the identical gap: only checks `gotMarker` on
  the `cancel` context, never asserts a tenant anywhere.
- The implementer's own report
  (`docs/tasks/fix-whisper-send-result-presence/sweep-tenant-environment-report.md`,
  "Tests" section) claims all four existing tests "already assert... the
  seam receives a context carrying the expected tenant," citing the guild
  and combo tests as proof "the marker lands on the context act/cancel
  receives." That is a different, weaker claim than what actually needs
  proving (that the tenant was present on the context handed *to*
  `envContext`), and the report conflates the two. The sweep brief is
  explicit: "Assert that the seam receives a context carrying the expected
  tenant" (`sweep-tenant-environment-origination.md:89-90`) — two of the
  four sites do not do this.

This does not point at a functional bug in the shipped code (section 3
above traced all five sites by hand and confirmed the tenant genuinely is on
ctx at each call site), but it is a real gap against an explicit,
enumerated requirement, and a real gap in regression protection: if a future
edit reordered `tenant.WithContext(...)`/`envContext(...)` in
`guild/task.go` or `character/combo/task.go`, neither test would catch it.

## Not evaluable

- Whether `atlas-maps`/`atlas-channel`'s LOGOUT consumer path now actually
  observes the fixed behavior live (the bug file's reproduction steps) is
  out of this diff's surface — no consumer-side code changed, and the bug
  file explicitly scopes the live re-test to a separate deployment step, not
  this commit range.
- `libs/atlas-env/registry.go`'s `MapRegistry.EnvironmentOfTenant` /
  `ApplyTenant` implementation (the concrete `TenantResolver`) was not
  touched by this diff and was not re-audited here; `ForTenant` only
  consumes its documented interface.

## Summary

Runtime logic is correct at every checked seam: `ForTenant`'s rules match
the bug file exactly, `TenantEnvironment` cannot produce a
tenant/environment disagreement, and all five wiring sites genuinely have
the tenant on `ctx` before the closure runs, with `socket.WithSelfEnvironment`
and the socket/listener paths left untouched. Two defects keep this out of
a clean approval: a doc-comment placement bug that silently strips
`Reconcile`'s documentation (including a load-bearing CAUTION note) and
misattaches it to `ForTenant`, and two of the four sibling sweep tests
(guild, channel-combo) that were not extended per the sweep brief's explicit
instruction and do not actually prove the fix — they pass identically
before and after.
