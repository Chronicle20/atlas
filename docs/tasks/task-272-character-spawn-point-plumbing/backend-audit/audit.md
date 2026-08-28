# Backend Audit — task-272-character-spawn-point-plumbing

- **Scope:** Changed Go packages, commit range `b284bcebf..61e5e4b94`
- **Services:** atlas-cashshop, atlas-channel, atlas-consumables, atlas-login, atlas-messages, atlas-npc-shops, atlas-pets, atlas-query-aggregator
- **Guidelines Source:** backend-dev-guidelines skill (`.claude/skills/backend-dev-guidelines/resources/audit-checklist.md`)
- **Date:** 2026-08-27
- **Build:** PASS (all 8 modules)
- **Tests:** PASS (all 8 modules, 0 failures)
- **Overall:** NEEDS-WORK

## Build & Test Results

`go build ./...` and `go test ./... -count=1` were run independently for each of the
8 affected modules from this worktree. All exited clean, no `FAIL` lines in any
module's test output.

| Module | Build | Tests |
|---|---|---|
| atlas-cashshop | PASS | PASS |
| atlas-channel | PASS | PASS |
| atlas-consumables | PASS | PASS |
| atlas-login | PASS | PASS |
| atlas-messages | PASS | PASS |
| atlas-npc-shops | PASS | PASS |
| atlas-pets | PASS | PASS |
| atlas-query-aggregator | PASS | PASS |

`tools/goroutine-guard.sh` run against every changed non-test `.go` file: exit 0
(no bare `go` statements introduced).

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE-01..06 | Yes | Every changed package has a `model.go`/`rest.go`/`builder.go` |
| DOM structure (DOM-01..05,11,16) | Yes | All 8 `character` packages have `model.go`; 6 of 8 have `rest.go` in the diff |
| REST (DOM-06..09,12..15,17..19,32) | No new instances | No `resource.go` or `processor.go` in any changed package; family document opened for DOM-04/18/19 (rest.go-only triggers) |
| SUB-01..04 | N/A | No changed package has `resource.go` without `model.go` |
| Constants reuse (DOM-21) | No | Diff declares no new type/const block/numeric-literal classification — see Constants section below |
| Testing (DOM-10,20,24,33) | Yes | Diff touches `_test.go` in every changed package |
| Cache (DOM-29) | No | No `cache.go`, no cached struct state in changed code |
| Messaging (DOM-30) | No | No `producer.go` touched, no `AndEmit`/`message.Emit` call added |
| Multi-tenancy (DOM-31) | Yes | `rest.go` changed in 6/8 packages |
| Migration hygiene (DOM-34,35) | No | No symbol moved between service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | No | No `libs/atlas-*` module added, no topic env var touched |
| Runtime safety (DOM-26) | Yes | Non-test `.go` files changed — `tools/goroutine-guard.sh` exit 0 |
| Channel wire values (DOM-25) | Yes (document opened) | `services/atlas-channel` and `services/atlas-login` socket/writer files touched |
| Resilience (DOM-27,28) | No | No DB-backed handler error branch changed, no `model.Decorator` touched |
| External clients (EXT-01..04) | Yes | Every changed `character` package's (untouched) `requests.go` calls `requests.RootUrlFor(ctx, "CHARACTERS")` / `requests.GetRequest[T]` |
| Scaffolding (SCAFFOLD-01..09) | No | No new `services/atlas-<svc>/` directory, no new channel Writer/Handler registered, `routes.conf` untouched |
| Security (SEC-01..04) | No | None of the 8 services in scope handle auth/tokens/redirects/secrets in the changed files |
| patterns-provider.md (foundational) | No | No `provider.go` in any changed package |
| patterns-functional.md (foundational) | No | No curried constructor/decorator/combinator added |

## Checklist Results

### atlas-cashshop / character (domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go validating Build() | N/A | `builder.go` not touched by this diff; not evaluated (out of scope) |
| DOM-02/03 | ToEntity/Make in entity.go | N/A | Package has no `entity.go` (read-only cross-service mirror model) |
| DOM-04 | `Transform(Model)(RestModel,error)` in rest.go | PASS | `services/atlas-cashshop/atlas.com/cashshop/character/rest.go:93` |
| DOM-05 | TransformSlice used by list handlers | N/A | Package has no `resource.go`/list handler; `TransformSlice` absent, no inline-loop violation possible |
| DOM-09/12-15/17/32 | resource.go / processor.go handler rules | N/A | Package has no `resource.go`, no `processor.go` change in diff |
| DOM-18 | RestModel implements GetName/GetID/SetID | PASS | `rest.go:47,51,55` (unchanged by diff, confirmed present) |
| DOM-19 | Flat request models | N/A | No new request model added |
| DOM-21 | No redeclared atlas-constants type | N/A | No new type/const declared; `libs/atlas-constants/` has no SpawnPoint/PortalId domain type to redeclare against (confirmed by grep) |
| DOM-31 | No tenant/trace in RestModel | PASS | `rest.go:40` `SpawnPoint uint32` is character position data, not tenant/trace state |
| FILE-01..06 | File placement | PASS | Accessor in `model.go:211`; `Extract` mapping in `rest.go:155` (`spawnPoint: m.SpawnPoint,`); no catch-all file |
| DOM-20 | Table-driven tests | PASS (with note) | New assertion in `rest_test.go:55-57` is a single-value direct-literal check appended to an existing test, not a new multi-case test function; nothing to tabulate |
| EXT-01 | Target RestModel has SetToOneReferenceID/SetToManyReferenceIDs | PASS | `rest.go:78,82` (pre-existing, unaffected) |
| EXT-04 | `requests.RootUrlFor`, not hardcoded DNS | PASS | `requests.go:24` `requests.RootUrlFor(ctx, "CHARACTERS")` |
| EXT-02/03 | Integration test + 404-vs-5xx mapping | Not evaluable from diff | `requests.go` untouched, no new external call added by this diff |

### atlas-channel / character (domain) + socket/writer (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | Transform in rest.go | PASS | `services/atlas-channel/atlas.com/channel/character/rest.go:89` |
| FILE-01..06 | File placement | PASS | Accessor `model.go:240`; `Extract` mapping `rest.go:152` |
| DOM-25 | Channel wire values resolved from tenant table, not Go literals | N/A | `byte(c.SpawnPoint())` at `socket/writer/character_data.go:47` narrows a raw positional data value (spawn-point index), not a dispatcher mode / sub-op / message / fail-reason code the client looks up in a table — outside DOM-25's enumerated scope (`anti-patterns.md` "dispatcher modes, sub-operation codes, message types, notice/fail reason codes") |
| DOM-20 | Table-driven tests | PASS | `socket/writer/character_data_test.go` new `TestBuildCharacterData_SpawnPoint` uses `tests := []struct{...}{ ... }` + `t.Run` (in-range / truncation cases) |
| DOM-31 | No tenant/trace in RestModel | PASS | `rest.go:38` `SpawnPoint uint32` is character data |
| EXT-01/04 | RestModel reference setters / RootUrlFor | PASS | `rest.go:77,81`; `requests.go:20` `requests.RootUrlFor(ctx, "CHARACTERS")` |

### atlas-consumables / character (domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | Transform in rest.go | PASS | `rest.go:132` (pre-existing, untouched by this diff) |
| FILE-01..06 | File placement | PASS | Accessor only touched: `model.go:213`. `rest.go` already had `spawnPoint: rm.SpawnPoint,` (`rest.go:124`) and `SpawnPoint: m.spawnPoint,` (`rest.go:158`) prior to this diff — confirms the bug was purely the stubbed accessor, not a missing plumbing seam |
| DOM-20 | Table-driven test | PASS (with note) | New assertion in `rest_test.go:48-51` is a single direct-literal check (275, above byte ceiling) appended to the existing round-trip test |
| DOM-31 | No tenant/trace in RestModel | PASS | `rest.go:38` |

### atlas-login / character (domain) + socket/writer (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go validating Build() | Not evaluable from diff | `builder.go` not touched; `SetSpawnPoint` confirmed pre-existing via targeted lookup (`builder.go`, called from `rest.go:156`) but the file's general Build()-validation compliance was not surveyed — out of scope |
| DOM-04 | Transform in rest.go | PASS | `rest.go:97` |
| FILE-01..06 | File placement | PASS | Accessor `model.go:222`; `Extract` builder chain adds `SetSpawnPoint(m.SpawnPoint).` at `rest.go:156` |
| DOM-25 | Channel wire values | N/A | `byte(c.SpawnPoint())` at `socket/writer/character_list.go:56` is the same raw-data narrowing case as atlas-channel above — not a client lookup-table code |
| DOM-20 | Table-driven tests | PASS | New `socket/writer/character_list_test.go` `TestToCharacterListEntry_SpawnPoint` uses `tests := []struct{...}` + `t.Run` |
| DOM-31 | No tenant/trace in RestModel | PASS | `rest.go:38` |

### atlas-messages / character (domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | Transform in rest.go | PASS | `rest.go:102` (pre-existing, untouched) |
| FILE-01..06 | File placement | PASS | Accessor only: `model.go:205`. `rest.go` already had `SpawnPoint: m.spawnPoint,` (`rest.go:129`) and `spawnPoint: rm.SpawnPoint,` (`rest.go:163`) prior to this diff |
| DOM-20 | Table-driven test | PASS (with note) | Single direct-literal assertion appended to existing `TestTransformRoundTrip`, `rest_test.go:413-417` |
| DOM-31 | No tenant/trace in RestModel | PASS | `rest.go:38` |

### atlas-npc-shops / character (domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go validating Build() | **FAIL (Important)** | `builder.go` was directly modified by this diff (3 new setters, `builder.go:116-118`), bringing the whole file into scope. `Build()` at `builder.go:123-158` performs zero invariant validation — it is a bare struct literal copy from `b.*` fields, matching every other field in the builder (`SetGm`, `SetMeso`, etc. are equally unvalidated). No documented exception in `file-responsibilities.md` or `anti-patterns.md` carves out read-only cross-service mirror models from DOM-01's "validating `Build()`" requirement. |
| DOM-04 | Transform in rest.go | PASS | `rest.go:99` |
| FILE-01..06 | File placement | PASS | Accessor `model.go:208`; `Extract` additions (`x: rm.X`, `y: rm.Y`, `stance: rm.Stance`) at `rest.go:163-165`; new builder setters at `builder.go:116-118`; no catch-all file |
| DOM-20 | Table-driven tests | WARN | New `TestTransform_PositionalFieldsFromBuilder` (`rest_test.go:87-119`) is single-case, not `tests := []struct{...}` + `t.Run`; the `TestTransformRoundTrip` additions (`rest_test.go:52-65`) are single direct-literal assertions on an existing test — neither has multiple scenarios to tabulate, so this is non-blocking |
| DOM-31 | No tenant/trace in RestModel | PASS | `rest.go:38` |
| EXT-01/04 | RestModel reference setters / RootUrlFor | PASS | `rest.go:86,90`; `requests.go:17` |

### atlas-pets / character (domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | Transform in rest.go | PASS | `rest.go:107` |
| FILE-01..06 | File placement | PASS | Accessor `model.go:207`; `Extract` adds `spawnPoint: m.SpawnPoint` (`rest.go:103`); `Transform` adds `SpawnPoint: m.spawnPoint` (`rest.go:113`) — this service needed both legs, unlike the others where one leg pre-existed |
| DOM-20 | Table-driven test | PASS (with note) | New `TestExtract_SpawnPoint` (`rest_test.go:35-46`) is a single-case direct test, not tabulated — nothing to tabulate for one input |
| DOM-31 | No tenant/trace in RestModel | PASS | `rest.go:38` |

### atlas-query-aggregator / character (domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | Transform in rest.go | PASS | `rest.go:92` |
| FILE-01..06 | File placement | PASS | Accessor `model.go:224`; `Transform` drops the redundant `uint32(...)` cast at `rest.go:128` (`SpawnPoint: m.SpawnPoint(),` — was `uint32(m.SpawnPoint())` when the accessor was `byte`) |
| DOM-20 | Table-driven tests | PASS (with note) | New `TestExtract_SpawnPoint` and `TestTransform_SpawnPointPreservesUint32` (`rest_test.go`, new file) are each single-case; correctly assert full uint32 fidelity (300, above byte ceiling) is preserved on the JSON:API re-serve path — this service does not narrow to a wire byte, unlike atlas-channel/atlas-login |
| DOM-31 | No tenant/trace in RestModel | PASS | `rest.go:39` |

## Constants (DOM-21)

Checked `libs/atlas-constants/` for an existing `SpawnPoint`/portal-id domain
type before accepting the bare `uint32` typing used throughout this diff:

```
grep -rniE "spawnpoint|portal" libs/atlas-constants/
```

returned only unrelated hits (`FieldLimitNoPortalScroll`, map-name comments
mentioning "has portals" / "no portals") — no `SpawnPoint` or `PortalId` type
exists in `libs/atlas-constants`. DOM-21's trigger ("diff declares a new type,
named const block, or numeric-literal classification") did not fire in the
first place: the diff only widens an existing private `uint32` field's
exposed accessor type and adds JSON:API field mappings, it declares no new
type. **N/A**, both on the negative trigger and on the absence of a
pre-existing shared type to redeclare against.

`RestModel.SpawnPoint` is `uint32` consistently across all 8 services
(`rest.go:38` or `:39/:40` in each package, confirmed by direct grep across
all 8 files) — no cross-service typing inconsistency was introduced.

## Not evaluable from the diff

- EXT-02 (httptest-backed integration test for the cross-service GET call): `requests.go` is untouched by this diff in all 8 packages and no new external call was added; would need to read each package's existing requests-test coverage, which is outside this diff's surface.
- EXT-03 (404-vs-5xx error mapping on the cross-service client): same — `requests.go` untouched, unrelated to the SpawnPoint change.
- DOM-01 (validating `Build()`) for atlas-cashshop, atlas-channel, atlas-consumables, atlas-login, atlas-messages, atlas-pets, atlas-query-aggregator: `builder.go` was not touched by this diff in these 7 packages (only npc-shops's was), so a full compliance sweep of each builder's `Build()` was not performed — flagging it for npc-shops (the one touched builder.go) only. Whether the other 7 share the same unvalidated-`Build()` shape was not surveyed, and is out of this diff's scope.

## Summary

### Blocking (must fix)

- DOM-01: `services/atlas-npc-shops/atlas.com/npc/character/builder.go:123-158` — `Build()` performs no invariant validation despite this diff directly modifying the file (adding `SetX`/`SetY`/`SetStance`). No documented exception exists for this package shape.

### Non-Blocking (should fix)

- DOM-20: Several new/appended tests across atlas-cashshop, atlas-consumables, atlas-messages, atlas-npc-shops (`TestTransform_PositionalFieldsFromBuilder`), and atlas-pets are single-case assertions rather than the `tests := []struct{...}` + `t.Run` table-driven shape. Each case has nothing to tabulate (single scenario), so this is a style preference note, not a hard violation.
