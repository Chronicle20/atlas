# Review: Task 5 — atlas-maps back-effect registry and processor

Range: `dffb2a82c..827d1592b` (single commit `827d1592b`, `feat(atlas-maps): add per-field back-effect registry and processor`)
Brief: `.superpowers/sdd/plan/task-5-brief.md`
Implementer report: `.superpowers/sdd/plan/task-5-report.md`

## Scope

`git diff --stat dffb2a82c..827d1592b`:

```
.../atlas.com/maps/map/backeffect/processor.go     | 48 ++++++++++++
.../atlas.com/maps/map/backeffect/registry.go      | 76 +++++++++++++++++++
.../atlas.com/maps/map/backeffect/registry_test.go | 86 ++++++++++++++++++++++
3 files changed, 210 insertions(+)
```

Exactly the three files listed in the brief's "Files" section. No scope drift.

## Findings

### 1. Interface contract — PASS

Committed code vs. brief's "Interfaces produced" block, verbatim comparison:

- `FieldKey{Tenant tenant.Model; Field field.Model}` — `registry.go:10-13`, matches.
- `BackEffectEntry{Effect byte; FieldId uint32; PageId byte; Duration uint32}` — `registry.go:15-24`, field names, order and types match exactly (the `Duration` field carries an added doc comment, no signature change).
- `Processor` interface: `Set(f field.Model, entry BackEffectEntry)`, `Clear(f field.Model) bool`, `GetActive(f field.Model) []BackEffectEntry` — `processor.go:12-16`, matches.
- `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` — `processor.go:23-25`, matches.

No silent rename. Tasks 6/7 can consume this as specified.

### 2. Concurrency correctness — PASS

- `registry.go:44-56` `Set` takes the exclusive lock (`r.mutex.Lock()`) for the full read-scan-and-write; `registry.go:58-65` `Get` takes `RLock()`; `registry.go:67-76` `Clear` takes `Lock()`. Lock scope is correct — no unlocked window between check and mutation.
- `Get` returns a defensive copy (`make` + `copy`, `registry.go:61-64`), not a slice aliasing the registry's backing array — confirmed by reading the code, and exercised indirectly by `TestSetReplacesSamePageInPlace`/`TestClearRemovesEveryPage` (though no test explicitly mutates the returned slice and re-reads to catch aliasing; this is a code-reading confirmation, not a test-enforced one).
- `Clear`'s "returns whether anything was removed" is computed from the map's `ok` result under the same exclusive lock that performs the `delete`, so there's no TOCTOU race between two concurrent `Clear` calls — the second racer sees `ok == false` because the map entry is already gone by the time it acquires the lock.
- `go test -race ./map/...` passed clean (includes `map/backeffect`), see verification below.

### 3. Tenant isolation — PASS

`FieldKey` embeds `tenant.Model` (comparable struct, confirmed via `libs/atlas-tenant/tenant.go`) alongside `field.Model` (comparable struct — `worldId`, `channelId`, `mapId` scalar ids plus a `uuid.UUID` array instance field, `libs/atlas-constants/field/model.go:14-19`). Two tenants with distinct UUIDs produce distinct map keys, making cross-tenant collision structurally impossible (not just untested-for). `TestBackEffectIsTenantIsolated` (`registry_test.go:68-86`) exercises this directly with two `tenant.Create` calls with distinct `uuid.New()` values and asserts zero-length `GetActive` on the non-writing tenant.

### 4. Pattern conformance — PASS

Compared against `map/jukebox/registry.go` and `map/jukebox/processor.go` (the named references):

- Same mutex + `sync.Once` package-level singleton shape (`getRegistry()`), same `FieldKey` struct.
- Correctly drops `ExpiresAt`, `ExpiredEntry`, `GetExpired`, `DeleteEntry` per the brief's explicit instruction — there is no reaper for back effects (design §3.1/§3.4: cleared only by explicit `CLEAR_BACK_EFFECT`).
- `Duration` carries the required doc comment (`registry.go:19-23`) explicitly stating it is not an expiry, referencing the jukebox `ExpiresAt` distinction as instructed.
- `ProcessorImpl{l, ctx}`, `NewProcessor`, `var _ Processor = (*ProcessorImpl)(nil)` assertion, and `tenant.MustFromContext(p.ctx)` to build `FieldKey` in every method — all present and matching the jukebox processor's shape.
- No `*_testhelpers.go` file; test setup lives inline in `registry_test.go`, matching the jukebox package's pattern (no test-only constructors).

### 5. Test honesty — PASS

- `TestSetThenGetActive`, `TestSetReplacesSamePageInPlace`, `TestClearRemovesEveryPage`, `TestBackEffectIsTenantIsolated` — all four tests from the brief's table are present and assert the specific values/lengths/positions the brief specifies (e.g. replaced page keeps position 0, `Clear` returns `false` on the second call).
- Implementer report documents a RED run (`undefined: NewProcessor` / `undefined: BackEffectEntry`, build failure before the package existed) followed by a GREEN run — genuine TDD evidence, not asserted after the fact.

### Non-blocking note

`Clear`'s debug log (`processor.go:40`, `"Cleared back effects in map [%d] instance [%s]."`) logs only map id/instance, while the brief's step 4 says to log "map id, instance, page id and effect" for both `Set` and `Clear`. The implementer's self-review discloses this explicitly: `Clear` removes every page for the field at once, so there is no single page/effect to attach to the log line, and inventing a synthetic value would be worse than omitting it. This is a reasonable, disclosed deviation from slightly underspecified brief wording, not a defect — no action required.

## Verification run (module-local, not tools/verify.sh)

```
cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./...
```
All packages `ok`, including `atlas-maps/map/backeffect`.

```
cd services/atlas-maps/atlas.com/maps && go test -race ./map/...
```
All packages `ok` (`map`, `map/backeffect`, `map/character`, `map/jukebox`, `map/monster`, `map/timer`), no race detected.

## Verdict

APPROVED
