# Review: Task 8 — the pending-dialog store (Maple Life)

Commit range: `5f427eb3d..dac10b874` (single commit `dac10b874`,
"feat(atlas-channel): add the Maple Life pending-dialog registry").

Brief: `.superpowers/sdd/plan/task-8-brief.md` (Controller addendum).
Report: `.superpowers/sdd/plan/task-8-report.md`.

## Scope

```
git diff --stat 5f427eb3d..dac10b874
 .../atlas.com/channel/maplelife/registry.go        | 207 ++++++++++++++
 .../atlas.com/channel/maplelife/registry_test.go   | 285 ++++++++++++++++++++
 2 files changed, 492 insertions(+)
```

Two new files only, exactly as scoped. No packet codec, template, registry
file, or `tools/packet-audit` artifact touched — matches the controller
addendum's stated scope. Reference file read for comparison:
`services/atlas-channel/atlas.com/channel/remotemerchant/registry.go`.

## Checklist

### 1. Tenant scoping is complete and non-vacuous — PASS

- `Get`, `Take`, `Put`, `Submit`, `ClearAccount` key by
  `Key{Tenant: t, AccountId: accountId}` (`registry.go:96,103,112,151,169`),
  so tenant is baked into the map key for every lookup/mutation.
- `TakeByTransactionId` and `Sweep` iterate the whole map and gate with
  `k.Tenant.Is(t)` (`registry.go:135`, `registry.go:194`), matching the
  reference file's scoping pattern and its documented rationale
  (`registry.go:180-188`, mirroring `remotemerchant/registry.go:106-114`).
- `tenant.Model` (`libs/atlas-tenant/tenant.go:10-15`) is a plain comparable
  struct (`uuid.UUID`, `string`, two `uint16`), so both map-key equality and
  `.Is()` compare on value, not identity.
- Test fixtures are genuinely distinct: `mustTenant` (`registry_test.go:15-22`)
  calls `tenant.Create(uuid.New(), ...)` per invocation, so `a, b :=
  mustTenant(t), mustTenant(t)` in `TestPutThenGet`, `TestTakeByTransactionIdIsTenantScoped`,
  and `TestSweepIsTenantScoped` produce two tenants with different UUIDs —
  not a fixture that compares equal to itself. Confirmed by reading
  `tenant.Create`/`Model.Is` (`libs/atlas-tenant/tenant.go:66-75`,
  `processor.go:31`).
- `TestSweepIsTenantScoped` (`registry_test.go:258-279`) preloads expired
  entries under both tenants at the same instant and asserts each tenant's
  `Sweep` call returns and removes only its own account — a real proof, not
  a same-tenant round trip.
- `TestTakeByTransactionIdIsTenantScoped` (`registry_test.go:170-194`) uses
  the *same* transaction id `"tx-1"` under both tenants and asserts taking
  under A leaves B's entry (with B's own `CandidateName`) intact — this is
  the strongest possible proof since it also rules out an unscoped linear
  scan matching the first hit regardless of tenant.

### 2. Keyed by `AccountId`, not `CharacterId` — PASS

`Key` is `{Tenant tenant.Model; AccountId uint32}` (`registry.go:51-54`);
every public method takes `accountId uint32`, not `characterId`. `Entry`
retains a `CharacterId uint32` field per the brief's block (populated by a
later task once the character is created), but it plays no role in keying,
lookup, or scoping. The package doc comment (`registry.go:1-4`) states the
rationale explicitly, distinct from `remotemerchant`'s character keying.

### 3. Interface matches the brief's block verbatim — PASS

Compared field-by-field against `task-8-brief.md:48-76`:

- `Phase string`, `PhaseOpen = "OPEN"`, `PhaseSubmitted = "SUBMITTED"` —
  match (`registry.go:26-35`).
- `Key{Tenant tenant.Model; AccountId uint32}` — match (`registry.go:51-54`).
- `Entry` fields and order — `CharacterId, WorldId, ItemId, Slot,
  UpdateTime, Phase, TransactionId, CandidateName, At` — match exactly
  (`registry.go:57-67`).
- `Expired{Tenant, AccountId, Entry}` — match (`registry.go:70-74`).
- `GetRegistry() *Registry`, `Put`, `Get`, `Take`, `TakeByTransactionId`,
  `Submit`, `ClearAccount`, `Sweep` — all present with the brief's exact
  signatures and return shapes (`registry.go:86,96,103,112,128,151,169,189`).
- `OpenTTL = 5 * time.Minute`, `SubmittedTTL = 30 * time.Second` — match
  (`registry.go:41,49`).

No extra exported surface beyond what the brief lists; `Get` cited in the
report as "added" is already in the brief's block (`task-8-brief.md:68`), so
that is not a deviation, just the report's phrasing.

### 4. `SubmittedTTL > 10*time.Second` is asserted directly — PASS

`TestSubmittedTTLOutlivesSagaTimeout` (`registry_test.go:281-285`) asserts
`SubmittedTTL <= 10*time.Second` fails the test, comparing the real constant
against a literal `10*time.Second`, not an alias of itself. Non-tautological.

### 5. Concurrency correctness — PASS

- `sync.RWMutex` used correctly: `Get` takes `RLock`/`RUnlock`
  (`registry.go:104-105`); every mutator (`Put`, `Take`,
  `TakeByTransactionId`, `Submit`, `ClearAccount`, `Sweep`) takes the full
  `Lock`/`Unlock`. No read lock held across a write.
- `Registry` is only ever obtained via `GetRegistry()` returning `*Registry`
  (`registry.go:86-91`); the mutex is never copied by value — no method
  receiver is `Registry` (value), all are `*Registry`.
- `Get`, `Take`, `Submit`, `TakeByTransactionId` all return `Entry` by
  value (a plain struct of scalars/`time.Time`, no embedded map/slice/pointer
  aliasing the registry's internal map), so no caller can mutate registry
  state through a returned value. `Sweep` returns `[]Expired`, a fresh slice
  built inside the lock (`registry.go:192,202`), not a view into `pending`.

### 6. `Take` / `TakeByTransactionId` are genuinely once-only — PASS

- `Take` (`registry.go:112-121`): read-check-delete happens under one
  `Lock`/`Unlock`, no window for a second caller to observe the entry after
  the first has begun removing it. `TestTakeRemoves`
  (`registry_test.go:78-95`) proves the second call misses.
- `TakeByTransactionId` (`registry.go:128-144`): same shape — the scan,
  match, and `delete` all happen inside the single `Lock`. `TestTakeByTransactionId`
  (`registry_test.go:140-168`) proves a second call on the same id misses,
  and that empty transaction ids never match (guards the `PhaseOpen`-with-
  empty-`TransactionId` case explicitly, `registry.go:131-133`).

### 7. Test builders / no placeholders — PASS

- No `*_testhelpers.go` file created; `mustTenant` and `openEntry` are plain
  helper functions living inside `registry_test.go` itself
  (`registry_test.go:15-32`), matching the `remotemerchant` pattern and
  CLAUDE.md's builder guidance.
- No stubbed method bodies or placeholder comments in `registry.go`; every
  method is a complete, real implementation.

## Test execution

Ran the module-local suite (no repo-wide `verify.sh`, per the controller
addendum):

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./maplelife/... -v
```

All 11 tests pass (`TestPutThenGet`, `TestPutIsIdempotentPerAccount`,
`TestTakeRemoves`, `TestSubmitTransitionsPhase`, `TestSubmitWithoutOpenFails`,
`TestTakeByTransactionId`, `TestTakeByTransactionIdIsTenantScoped`,
`TestClearAccount`, `TestSweepUsesPhaseSpecificTTL`, `TestSweepIsTenantScoped`,
`TestSubmittedTTLOutlivesSagaTimeout`). The report's quoted RED-phase build
failure (`undefined: Entry`, `undefined: GetRegistry`, ...) is consistent
with what step 2 of the brief expects to observe before `registry.go` exists.

## Not evaluable

None. The unit is pure in-memory state with no I/O and no cross-service
seam; the full surface (registry + test) was read and exercised.

## Verdict

APPROVED. No blocking or non-blocking findings.
