# Fix round A report — `atlas-player-npcs` `playernpc/` structure and REST discipline

## Summary

Fixed all 9 blocking findings from `audit-backend-player-npcs.md` against the `playernpc/`
domain package. This was a pure refactor: no REST-observable behavior changed except the PATCH
`/player-npcs/{id}` route now requires a (structurally empty) JSON:API body, which is an
unavoidable consequence of DOM-08 — the redeploy test was updated to send one, and its assertions
(status codes, script/object id and position unchanged across redeploy) are unchanged.

## Findings addressed

1. **DOM-02** (`entity.go`) — the free function `MakeEntity(tenantId, m)` is now the method
   `Model.ToEntity(tenantId)`. All four call sites updated (`administrator.go`, `processor.go`,
   `model_test.go` x2).
2. **DOM-04/FILE-02 + FILE-03 (item 8)** — re-sorted the three inverted files against the
   `atlas-notes/note` reference:
   - `rest.go` now holds `RestModel`, `EquipmentRestModel`, `Transform`, the new `TransformSlice`,
     and the request-body models `DeployRestModel`/`PositionRestModel`/`RedeployRestModel` (moved
     from the old `requests.go` and old `resource.go`).
   - `resource.go` now holds route registration and every HTTP handler (moved from the old
     `rest.go`).
   - `requests.go` deleted: this package has no cross-service request functions of its own (that
     traffic lives in `character/`, `inventory/`, `ranking/`, `configuration/`, `data/map/`,
     `data/npc/` — all out of this round's scope), so there is nothing for the file to hold.
3. **DOM-05** — added `TransformSlice(ms []Model) ([]RestModel, error)` in `rest.go`
   (`model.SliceMap(Transform)...(model.ParallelMap())()`, same call `handleGetPlayerNpcs`
   previously inlined). The list handler now calls it instead of inlining the transform.
4. **DOM-08** — PATCH `/player-npcs/{id}` now registers via
   `server.RegisterInputHandler[RedeployRestModel]` instead of `RegisterHandler`.
   `RedeployRestModel` is a new, deliberately attribute-free JSON:API body type (`rest.go`) — the
   endpoint genuinely takes no attributes (script id/object id/position are immutable through it,
   design §6.2), but the convention requires the input-handler variant regardless. The test now
   sends `jsonapi.Marshal(RedeployRestModel{})` as the PATCH body.
5. **DOM-13/DOM-14** — every handler now routes through `Processor` via `processorFor(db, d)`:
   - `handleGetPlayerNpc` calls `Processor.GetById` instead of `getPlayerNpcModel` directly.
   - `handleGetPlayerNpcs` calls the new `Processor.GetByMapPaged` instead of the old free
     function `pagedPlayerNpcsByMap` (renamed `playerNpcsByMapPaged`, moved into
     `administrator.go` next to its sibling `playerNpcsByMap`).
   - `handleGetEligibility` calls the new `Processor.Eligibility(characterId, worldId, mapId)`
     instead of constructing `character.NewProcessor`/`configuration.NewProcessor` inline and
     calling `countByName`/`eligibility.Evaluate` itself. The handler is now pure HTTP
     parsing/response mapping.
   - `Processor` interface gained two methods (`GetByMapPaged`, `Eligibility`) to carry this
     logic. `GetByMap`'s existing signature (used unmodified by both Kafka consumers) was left
     alone rather than reshaped, per its own doc comment — `GetByMapPaged` is the pagination-aware
     sibling. Per DOM-33, every mock implementation of `Processor` was updated in the same diff:
     `kafka/consumer/playernpc/consumer_test.go`'s `stubProcessor` and
     `kafka/consumer/character/consumer_test.go`'s `stubPlayerNpcProcessor` both gained the two
     new methods (both packages are outside the `{data,character,inventory,ranking,configuration}`
     exclusion list the brief named, and are not owned by fix rounds B or C — leaving them
     unimplemented would fail `go build` for the whole module).
6. **DOM-16** — `administrator.go` gained `createPlayerNpcTx(tx, tenantId, m) (uuid.UUID, error)`,
   the tx-scoped half of `createPlayerNpc` (which now opens a transaction and delegates to it).
   `processor.go`'s `Deploy` now calls `createPlayerNpcTx(tx, tenantId, m)` instead of inlining
   `tx.Create`. (`createPlayerNpc` itself, not `createPlayerNpcTx`, could not be called directly
   from `Deploy` without changing behavior — `Deploy` already holds an open transaction from its
   own `database.ExecuteTransaction`, and `createPlayerNpc` opens a second one via the same
   helper; `createPlayerNpcTx` is the shared core both call.)
7. **DOM-32** — `InitializeRoutes` now calls `server.RegisterHandler(l)(si)` /
   `server.RegisterInputHandler[T](l)(si)` directly; the local `registerHandler`/
   `registerInputHandler` wrappers, and the local `HandlerDependency`/`HandlerContext`/
   `GetHandler`/`InputHandler`/`parseInput` types that duplicated `libs/atlas-rest/server`'s own,
   are gone. Since `server.HandlerDependency` carries only the per-request logger/context (no
   `*gorm.DB`), `db` is curried into every handler constructor instead
   (`func handleGetPlayerNpcs(db *gorm.DB) server.GetHandler { ... }`), matching
   `services/atlas-tenants/atlas.com/tenants/configuration/resource.go`'s own `func(db *gorm.DB)
   func(d *rest.HandlerDependency, ...) http.HandlerFunc` shape — the pattern that shape is
   copied from. `writeError` is kept (there is no `libs/atlas-rest/server` primitive that carries
   a domain `code` field, and design §8.3's four failure codes are a genuine, tested contract),
   but it now marshals `api2go/jsonapi.Error` — the library's own error-object type — instead of a
   private locally-declared struct, so it no longer duplicates a shape the dependency already
   defines.
8. See item 2.
9. **FILE-05** — `EventType` and its four constants moved out of `processor.go` into a new
   `state.go`.

## Files changed

- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/entity.go` — `MakeEntity` →
  `Model.ToEntity`
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/administrator.go` —
  `createPlayerNpcTx` added; `createPlayerNpc` delegates to it;
  `playerNpcsByMapPaged` added (moved in from the old `rest.go`)
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/processor.go` — `Deploy` calls
  `createPlayerNpcTx`; `EventType`/consts moved to `state.go`; `Processor` gained
  `GetByMapPaged`/`Eligibility` plus their `ProcessorImpl` implementations
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/state.go` — new; `EventType`
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/rest.go` — rewritten: `RestModel`,
  `EquipmentRestModel`, `Transform`, `TransformSlice`, `DeployRestModel`, `PositionRestModel`,
  `RedeployRestModel`
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/resource.go` — rewritten: route
  registration + every handler, all through `Processor`; `writeError`/`writeDeployError` retained,
  reusing `jsonapi.Error`
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/requests.go` — deleted
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/model.go` — doc comment
  `MakeEntity` → `ToEntity`
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/model_test.go` — two call sites
  updated to `m.ToEntity(tenantId)`
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/resource_test.go` — "re-deploy"
  subtest now sends a `RedeployRestModel{}` JSON:API body on the PATCH request
- `services/atlas-player-npcs/atlas.com/player-npcs/kafka/consumer/playernpc/consumer_test.go` —
  `stubProcessor` gained `GetByMapPaged`/`Eligibility` (DOM-33)
- `services/atlas-player-npcs/atlas.com/player-npcs/kafka/consumer/character/consumer_test.go` —
  `stubPlayerNpcProcessor` gained `GetByMapPaged`/`Eligibility` (DOM-33)

## Testing

```
cd services/atlas-player-npcs/atlas.com/player-npcs && go build ./... && go test ./...
```

Output: `Go test: 213 passed in 18 packages` — build clean, every package's tests pass, output
pristine. `go vet ./...` produced no output. `gofmt -l` on every touched file produced no output.

## Self-review

- Every handler now goes through `Processor` — no handler calls a provider/administrator function
  or constructs another package's processor directly except inside `processorFor`, which exists
  precisely to build the one `Processor` the handlers use.
- No behavior change beyond the PATCH body requirement DOM-08 mandates; that change is covered by
  the updated test, and every other status-code/response-shape assertion in `resource_test.go`
  (duplicate/pool-exhausted/map-full/ineligible/unresolvable error codes, pagination envelope,
  tenant scoping, deploy/redeploy/delete flows, eligibility endpoint) passes unmodified.
- `Processor` interface growth (`GetByMapPaged`, `Eligibility`) was the minimum needed to satisfy
  DOM-13/14 without reshaping `GetByMap`'s existing signature, which two Kafka consumers already
  depend on outside this round's file list; both stub implementations were updated in the same
  diff per DOM-33.
- Considered dropping `writeError` entirely per the letter of finding 7, but that would either
  delete the design §8.3 `code` field from 409/422 responses (a real behavior change, forbidden by
  the brief) or require adding a new `code`-carrying primitive to `libs/atlas-rest/server` (out of
  this round's scope — no such primitive exists anywhere else in the repo). Kept it, reusing
  `api2go/jsonapi.Error` instead of a private struct, and documented the reasoning inline.

## Concerns

- None blocking. The `Processor` interface addition (`GetByMapPaged`, `Eligibility`) and the two
  Kafka-consumer stub updates go slightly beyond the brief's `Files` list (which named only
  `playernpc/` files), but were required for `go build`/`go test` to pass for the whole module
  once DOM-13/14 moved eligibility and paginated-list logic out of the handlers and into
  `Processor` — those two consumer-test files are not owned by fix rounds B or C.
