# Fix report: two blocking review findings (tenant-environment origination)

Source: `docs/tasks/fix-whisper-send-result-presence/review-tenant-environment.md`

## Finding 1 — libs/atlas-env/tenants.go doc-comment regression

`ForTenant`'s doc comment and body had been inserted between `Reconcile`'s
doc comment and `func Reconcile`, so Go's adjacency rule merged the whole
`Reconcile` doc block (including the load-bearing CAUTION paragraph) onto
`ForTenant`, and left `Reconcile` undocumented.

Fix: relocated `ForTenant` (doc comment + body) to after `Reconcile`'s
closing brace. No wording was changed in either doc comment — pure
relocation.

Verification with `go doc -all .` in `libs/atlas-env`:

```
$ go doc -all . | sed -n '/func ForTenant/,/^$/p'
func ForTenant(r Registry, tenantId string, self Id) Id
    ForTenant resolves the environment for tenant-scoped work that arrived
    with no environment of its own (a background sweep, not a request).
    It type-asserts r to TenantResolver exactly as Reconcile does; if the
    assert fails, tenantId is empty, the tenant is unknown, or its projected
    environment is "" (legacy -- see the comment on Reconcile explaining why
    empty means "don't filter", not "belongs to nothing"), it falls back to
    self. Otherwise it returns the tenant's environment. ForTenant never returns
    "" when self is non-empty -- the FR-1.8 guarantee NewTimeout's doc comment
    depends on must survive.

---
$ go doc -all . | sed -n '/func Reconcile/,/^$/p'
func Reconcile(r Registry, headerEnv Id, tenantId string) (Id, error)
    Reconcile resolves the operation's environment from the header and the
    tenant, and returns ErrEnvironmentMismatch when they disagree (FR-7.7).
    An unknown tenant trusts the header: during activation the tenant and
    environment records arrive on different topics and therefore different
    partitions, so a tenant may be visible before or after its environment
    (design §7.3). This does not weaken D4 — an unknown ENVIRONMENT is still
    rejected by the ownership gate.
```

Both functions now print their own text. The CAUTION paragraph (about the
`tenantId == ""` short-circuit resembling a parser-ordering bug) is part of
`Reconcile`'s doc comment body and correctly attached to `Reconcile` after
the relocation — confirmed by reading the file directly after the edit.

## Finding 2 — sibling seam tests did not assert the new contract

### guild/task_test.go — TestProcessExpiredCoordinationsAppliesEnvContextToAct

The `envContext` spy now captures the context it receives via
`tenant.MustFromContext(ctx)` before stamping its marker, and the test
asserts `gotInputTenant == ten` (the same tenant fixture used to build the
expired coordination via `buildExpiredCoordination`). The existing
downstream-marker and leaderId assertions are unchanged, only extended.
Doc comment updated to describe the tenant-derived-environment property
rather than "this pod's own environment identity."

### character/combo/task_test.go — TestProcessExpiriesAppliesEnvContextToCancel

Same treatment: the `envContext` spy captures `tenant.MustFromContext(ctx)`
into `gotInputTenant` before stamping the marker, and the test asserts it
equals `tn` (the tenant on the `Expired` fixture). Added the
`atlas-tenant` import (was previously not imported by this test file).
Existing marker/`n` assertions kept, doc comment updated similarly.

### Sibling check — atlas-pets and atlas-trades

- `services/atlas-pets/atlas.com/pets/pet/task_test.go`
  `TestTimeoutOwnerTenantContextAppliesEnvContext`: the spy's
  `envContext` closure is `context.WithValue(ctx, envMarkerKey("marker"),
  "stamped")` — a genuine passthrough that never strips or replaces any
  existing context value. The test asserts `tenant.MustFromContext(tctx) ==
  tn` on the *output*. Because the spy is value-preserving, this is
  equivalent evidence that the tenant was present on the *input* handed to
  envContext. Verified this reasoning by re-reading the closure body — it
  is exactly `context.WithValue` on the incoming `ctx`, no rebuild. No
  change needed; this test already proves the property.
- `services/atlas-trades/atlas.com/trades/trade/settlement_env_test.go`
  (`TestDetachedAppliesEnvContext`, `TestReconcileAppliesEnvContext`,
  `TestReconcileEscrowAppliesEnvContext`): each spy closure calls
  `tenant.MustFromContext(c)` *inside* the closure, on the input `c`, and
  asserts `seenTenantId == tm.Id().String()` — this directly proves the
  seam receives the tenant, for all three `applyEnvContext` call sites. No
  change needed.

Both sibling files were already adequate, matching the reviewer's original
assessment; no extension made to them.

## Verification (foreground, non-backgrounded)

```
$ cd libs/atlas-env && go build ./... && go test ./...
ok  	github.com/Chronicle20/atlas/libs/atlas-env	0.003s
```

```
$ cd services/atlas-guilds/atlas.com/guilds && go build ./... && go test ./...
ok  	atlas-guilds	(cached)
ok  	atlas-guilds/guild	0.065s
ok  	atlas-guilds/guild/member	(cached)
ok  	atlas-guilds/guild/title	(cached)
... (all other packages ok / no test files)
```

```
$ cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
ok  	atlas-channel/character/combo	(covered in full run, all ok)
... (all other packages ok / no test files)
```

Targeted re-run of the changed combo tests, verbose:

```
$ go test ./character/combo/... -v -run TestProcessExpiries
=== RUN   TestProcessExpiriesCancelsOncePerEntry
--- PASS: TestProcessExpiriesCancelsOncePerEntry (0.00s)
=== RUN   TestProcessExpiriesSwallowsCancelFailure
--- PASS: TestProcessExpiriesSwallowsCancelFailure (0.00s)
=== RUN   TestProcessExpiriesEmptySweepDoesNothing
--- PASS: TestProcessExpiriesEmptySweepDoesNothing (0.00s)
=== RUN   TestProcessExpiriesAppliesEnvContextToCancel
--- PASS: TestProcessExpiriesAppliesEnvContextToCancel (0.00s)
PASS
ok  	atlas-channel/character/combo	0.007s
```

## Files changed

- `libs/atlas-env/tenants.go` — relocated `ForTenant` below `Reconcile`.
- `services/atlas-guilds/atlas.com/guilds/guild/task_test.go` — spy now
  asserts tenant on input; doc comment updated.
- `services/atlas-channel/atlas.com/channel/character/combo/task_test.go`
  — spy now asserts tenant on input; `atlas-tenant` import added; doc
  comment updated.

## Self-review

- Doc-comment relocation is a pure move; no wording changed, confirmed via
  `go doc -all .` output above showing both functions' original text
  intact and correctly attributed.
- Both extended tests keep their pre-existing assertions and only add the
  new input-tenant assertion, per the brief's "add to them, do not
  replace them."
- Checked the two sibling files (pets, trades) that the reviewer already
  marked adequate; confirmed independently by reading their spy closures —
  both genuinely prove the property (pets via a value-preserving
  passthrough + output check, trades via a direct input check inside the
  closure). No changes were needed there.
- `docs/tasks/fix-whisper-send-result-presence/agent-ledger.tsv` was
  already modified in the working tree before this session started (not
  by this session's edits) and was deliberately left out of this commit
  per the brief's "do not commit anything under docs/."

## Commit

`98ef8b11a` — `fix(atlas-env): restore Reconcile doc comment, assert tenant on envContext input`

Branch: `fix-whisper-send-result-presence`
Worktree: `<repo-root>/.worktrees/fix-whisper-send-result-presence`
