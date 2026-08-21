# Review: Task 5 — `position` last-known-position registry

Commit: `d5433181a` (range `1182985b8..d5433181a`)
Brief: `.superpowers/sdd/plan/task-5-brief.md`
Report: `.superpowers/sdd/plan/reports/task-5-report.md`

## Scope reviewed

- `services/atlas-channel/atlas.com/channel/position/registry.go` (new)
- `services/atlas-channel/atlas.com/channel/position/registry_test.go` (new)
- `services/atlas-channel/atlas.com/channel/session/processor.go` (edit, +15/-0)
- `services/atlas-channel/atlas.com/channel/session/position_hook_test.go` (new)
- Read-only reference: `character/chakra/registry.go`, `session/aran_combo_hook_test.go`

`git diff --stat 1182985b8..d5433181a` confirms exactly these four files, matching
the brief's file list. No scope drift.

`position.GetRegistry().Put` is called only from tests in this commit — per the
task dispatch note, the production writer is Task 6's `movement/processor.go`
work, correctly deferred and disclosed in the implementer report rather than
hidden. Not treated as a finding.

## Findings

### 1. Tenant scoping — PASS

`position/registry.go:19-22` defines `Key{Tenant tenant.Model, CharacterId uint32}`,
and every method (`Put` line 58-62, `Lookup` line 65-70, `Clear` line 73-77)
builds the map key from both fields. `tenant.Model` (`libs/atlas-tenant/tenant.go:10-15`)
is a plain comparable struct (`uuid.UUID`, `string`, two `uint16`s), so it is a
valid, value-equal map key — same shape as `character/chakra/registry.go:31-34`'s
`Key`, the pattern this task was told to copy. `TestRegistry_TenantIsolated`
(`position/registry_test.go:46-55`) pins a `Put` under tenant A and asserts a
`Lookup` under tenant B for the same character id returns `ok == false` — this
actually exercises the isolation, not just a same-tenant round trip. No
cross-tenant bleed.

### 2. Concurrency — PASS

`Registry.mutex` is a `sync.RWMutex`; `Put` and `Clear` take `Lock()`, `Lookup`
takes `RLock()` (`position/registry.go:58-77`), matching the chakra registry's
locking discipline exactly. `GetRegistry()` uses `sync.Once` for the singleton
(`registry.go:40-55`), also matching. Ran `go test -race ./position/... ./session/...`
from the module root — passes clean (no data races reported).

### 3. Destroy hook fires on every destroy path — PASS

`session/processor.go:474-478` adds `clearLastPositionOnDestroy`, called from
`Destroy` at line 422, immediately after the existing
`clearAranComboOnDestroy(p.ctx, s.CharacterId())` call at line 418 — same
placement, same `if characterId != 0` guard, same doc-comment framing the brief
specified. `Destroy` is the single funnel: every call site that tears down a
session (`socket/handler/*.go`, `kafka/consumer/session/consumer.go`,
`session/task.go`) routes through `ProcessorImpl.Destroy` /
`DestroyById`/`DestroyByIdWithSpan`, which all resolve to the same `Destroy`
method (`session/processor.go:406-433`) — there is exactly one `Destroy` body,
so there is no second destroy path that could skip the new clear call. This
mirrors the Aran combo counter, which relies on the identical funnel and is
accepted precedent in this codebase.

`session/position_hook_test.go` copies `aran_combo_hook_test.go`'s shape:
`TestClearLastPositionOnDestroy_NonZeroCharacter_ClearsState` seeds via `Put`,
asserts precondition via `Lookup` before calling the hook (fail-fast on bad
setup), calls `clearLastPositionOnDestroy`, then asserts `Lookup` now misses.
`TestClearLastPositionOnDestroy_ZeroCharacter_NoOp` seeds an unrelated
character (id 42), calls the hook with `characterId == 0`, and asserts
character 42's entry survives — this is a genuine no-op pin, not merely "call
succeeds."

### 4. Repo conventions — PASS

- Singleton + `GetRegistry()` pattern copied faithfully from `chakra/registry.go`.
- Package doc comment (`position/registry.go:1-9`) documents the import
  boundary rationale (`sync` + `libs/atlas-tenant` only) as load-bearing,
  per the brief's explicit instruction.
- `gofmt -l` on all four touched/added files: no output (already formatted).
- `go vet ./position/... ./session/...`: clean.
- No TTL/sweeper, with the bounded-by-connected-characters rationale
  documented on `Registry` (`registry.go:31-34`), matching the brief.

### 5. Test honesty — PASS

`TestRegistry_Clear` (`registry_test.go:72-81`) explicitly pins `Clear`
behaviour (`Put` then `Clear` then `Lookup` must miss) — not just `Put`/`Lookup`.
`TestClearLastPositionOnDestroy_NonZeroCharacter_ClearsState` in the session
package pins the same clearing behaviour through the destroy-hook seam, and its
sibling zero-character test pins the no-op/isolation edge. Manually confirmed
(by reading, not running with an intentionally-reverted patch) that
`TestRegistry_TenantIsolated` would fail without the `Tenant` field in `Key`
(it would collapse tenant A and B onto the same character-id key) — this is a
real assertion of the tenant-scoping requirement, not a tautology.

## Verification run

```
cd services/atlas-channel/atlas.com/channel
go build ./...                                    # clean
go test -race ./position/... ./session/...         # ok, ok, no races
gofmt -l position/registry.go position/registry_test.go \
        session/position_hook_test.go session/processor.go   # no output
go vet ./position/... ./session/...                 # clean
```

## Not evaluable

- Full `tools/verify.sh` gate was not run (out of scope for a per-unit
  reviewer per this task's instructions; module-local build/test/vet above
  substitutes).
- Task 6's writer (`movement/processor.go`'s `ForCharacter` calling
  `position.GetRegistry().Put`) is out of scope for this review by design —
  not evaluated here.

## Verdict

APPROVED. No blocking or non-blocking findings. Implementation matches the
brief exactly (types, method signatures, placement, doc-comment framing), the
registry is correctly tenant-scoped and concurrency-safe, the destroy hook
fires from the single `Destroy` funnel shared with the pinned Aran-combo
precedent, and the tests genuinely assert tenant isolation and clear
behaviour rather than merely covering the happy path.
