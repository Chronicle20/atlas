# Review — fix rounds A/B/C (`9c35b3870..15f6cb629`)

Reviewed commits: `15f6cb629` (round A), `8a74e2f53` (round B), `b9ebb8883` (round C).
Ignored `e00a7abc2` (docs-only) per instructions.

Scope: the diff of these three commits against the two audits' 15 blocking findings, plus the four
specific questions raised. Read the full diffs for every touched file (`git show <sha> -- <path>`),
cross-checked handler bodies against pre-fix `rest.go` at `9c35b3870` where behavior equivalence
mattered, and ran `go build ./... && go test ./... && go vet ./...` in all three touched modules
(`atlas-player-npcs`, `atlas-channel`, `atlas-query-aggregator`).

## Behavior-preservation check (the primary question)

**No behavioral regression found.**

- `playernpc.Deploy` (processor.go): the inline `tx.Create(&entity)` / equipment-insert loop was
  extracted verbatim into `administrator.go`'s `createPlayerNpcTx(tx, tenantId, m)`. Byte-for-byte
  same statements, same `equipment[i].Id = uuid.Nil` reset, same error propagation. Confirmed via
  `git show 15f6cb629 -- .../processor.go` and `.../administrator.go`.
- `handleGetEligibility` (resource.go:344-397): the character lookup, `ErrNotFound` → 422
  `unresolvable` mapping, `countByName`, and `eligibility.Evaluate(cfg, c, existingCount, true)`
  call sequence is preserved exactly, now split across `Processor.Eligibility` (business logic) and
  the handler (HTTP parsing/response). One minor semantic note, non-blocking: pre-fix, only the
  `character.GetById` error was checked against `requests.ErrNotFound`; `countByName`'s error went
  straight to `WriteErrorResponse`. Post-fix, both errors returned by `Processor.Eligibility` pass
  through the same `errors.Is(err, requests.ErrNotFound)` check in the handler. `countByName` is a
  DB query and cannot itself produce `requests.ErrNotFound` (an HTTP-client sentinel), so this is
  inert today, but it is a latent behavioral coupling a future error path could trip on.
- `handleGetPlayerNpcs`: `TransformSlice` (rest.go:112-114) is `model.SliceMap(Transform)(...)`,
  the exact call the old handler inlined. Pagination envelope construction
  (`paginate.EnvelopeFor(paged)`) unchanged.
- PATCH `/player-npcs/{id}` (DOM-08): now requires a JSON:API body via
  `RegisterInputHandler[RedeployRestModel]`. `RedeployRestModel` carries no attributes and
  `handleRedeployPlayerNpc` still resolves the id from the URL path (`ParseUUIDId`), not the body —
  confirmed at resource.go:251-269. The only observable change is the new body requirement, which
  the brief explicitly authorized as the unavoidable DOM-08 consequence; `resource_test.go`'s only
  edit is additive (marshals and sends `RedeployRestModel{}`, no assertion touched). Swept for
  in-repo callers of this endpoint (`grep` across `atlas-channel`/`atlas-query-aggregator` for
  redeploy/PATCH player-npc callers) — none exist yet, so no cross-service caller is broken by the
  new body requirement.
- Kafka consumer paths (`kafka/consumer/playernpc/consumer.go:123`,
  `kafka/consumer/character/consumer.go:134`) still call `Processor.GetByMap` with its original
  signature — untouched, per the brief's own note that `GetByMapPaged` is a new sibling, not a
  reshape.
- `Model.ToEntity(tenantId)` (entity.go) is `MakeEntity(tenantId, m)` with the receiver moved;
  method body is untouched (diff shows only the signature line changed).
- `go build ./... && go test ./... && go vet ./...` all green in `atlas-player-npcs`
  (213 tests passed across 18 packages), and `go build ./... && go test ./...` green in
  `atlas-channel` and `atlas-query-aggregator`.

## Ruling on the four specific questions

### 1. `writeError` kept instead of deleted (DOM-32)

**Ruling: justified, not unfinished.** Read `libs/atlas-rest/server/error.go` in full.
`WriteErrorResponse` emits `{status, title}` only; `WriteBadRequest` emits
`{status, title, detail}`. Neither type (`errorObject`/`badRequestError`) carries a `code` field.
Swept the rest of `libs/atlas-rest/server/*.go` for any other `code`-carrying type — none exists.
Design §8.3 (`docs/tasks/task-251-player-npcs/design.md:637-642`) requires "a distinguishing `code`
on REST" for the four failure codes (`pool_exhausted`, `map_full`, `duplicate`, `ineligible`) plus
422 for unresolvable character/map — a genuine, tested contract (`resource_test.go`'s
duplicate/pool-exhausted/map-full/ineligible/unresolvable assertions). No `libs/atlas-rest/server`
primitive can carry this today, so deleting `writeError` per DOM-32's literal wording would either
drop the `code` field (a real behavior/contract change, forbidden by the "no behavior change"
constraint) or require adding a new primitive to `libs/atlas-rest/server` — out of this round's
scope. The fix reduces duplication where it safely can: `writeError` now marshals
`api2go/jsonapi.Error` (a real, already-imported library type that does carry `Code`) instead of a
hand-rolled private struct, closing the "hand-rolled duplicate shape" half of the finding while
keeping the field the design requires. This is a correct, well-reasoned partial resolution, not an
overlooked cleanup.

### 2. `Processor` interface widened with `GetByMapPaged`/`Eligibility` — every implementation updated?

**Ruling: complete, no stale mock.** `grep -rn "playernpc.Processor\b"` across the module finds
exactly four consumers: `main.go` (wiring only, no mock), `kafka/consumer/playernpc/consumer.go`
and `kafka/consumer/character/consumer.go` (production code, both use the interface, not a
concrete struct — nothing to update there), and the two `_test.go` files whose stubs implement the
full interface. Both `stubProcessor` (`kafka/consumer/playernpc/consumer_test.go:76-83`) and
`stubPlayerNpcProcessor` (`kafka/consumer/character/consumer_test.go:94-99`) gained both new
methods in the same commit, and each file still asserts `var _ playernpc.Processor = (*stub...)(nil)`,
which would fail to compile if either mock were incomplete. `go build ./...` and `go vet ./...`
confirm this — both are clean. `playernpc.ProcessorImpl` itself implements both new methods
(processor.go:150-172). No other package in the repo declares a type asserting conformance to
`playernpc.Processor` (checked `atlas-channel` and `atlas-query-aggregator` — they only consume
their own local `playernpc` packages, unrelated types).

### 3. EXT-01 stubs applied to `GroundRequestRestModel`, a POST body

**Ruling: over-application; harmless but not required, and the brief itself already flags this as
a judgment call rather than a needed fix.** `SetToOneReferenceID`/`SetToManyReferenceIDs` exist
because `api2go/jsonapi.Unmarshal` walks a `relationships` block on *decode*
(`libs/atlas-rest/CLAUDE.md`'s own framing: "JSON:API target structs MUST implement the
relationship interfaces if the upstream response has any relationships block" — decoding an
upstream *response*). `GroundRequestRestModel` is never an `Unmarshal` target anywhere in this
service: `grep -rn "GroundRequestRestModel"` shows exactly one construction site
(`data/map/requests.go:34`, `GroundRequestRestModel{Points: points}`, marshaled outbound as a POST
body) and one test reference building the struct directly — no `jsonapi.Unmarshal(&GroundRequestRestModel{})`
call exists. `jsonapi.Marshal` does not require these methods (it uses a different interface pair
for relationship *serialization*, not `Set*`). So the stubs on `GroundRequestRestModel` are dead
code from EXT-01's actual purpose, not a functional risk, but also not something the finding
required — the audit's finding text names `GroundRequestRestModel` alongside the other two structs,
which is what the implementer followed literally, but the underlying rule (decode-target
correctness) does not reach a POST-only body. Non-blocking: this is inert boilerplate, safe to
leave, but a future cleanup could drop it without losing coverage.

### 4. `playernpc_equipment_decode_test.go` at module root, `package main`

**Ruling: acceptable as-is; a move is optional polish, not a defect.** Repo-wide precedent check
(`find services -maxdepth 3 -name "*_test.go"` filtered to module roots) shows every one of the
~50 services in this monorepo has at least one root-level `package main` test file
(`wiring_test.go`, and in several services `main_test.go`/`tick_test.go`/`bootstrap_test.go`
alongside it) — this is an established, load-bearing repo convention, not an anomaly this file
introduced. `go build ./...`/`go test ./...` confirm the file compiles and runs cleanly as part of
the module. The test itself (`playernpc_equipment_decode_test.go:20-74`) imports the real, exported
`playernpc.RestModel`/`playernpc.EquipmentRestModel` types and round-trips them through real
`jsonapi.Marshal`/`jsonapi.Unmarshal` — it does not need unexported access, so it could equally be
`package main_test` (marginally more idiomatic, since it only consumes exported surface) or, now
that round A has landed and the parallelism constraint that forced it out of `playernpc/` no longer
applies, moved into `playernpc/` itself as `playernpc_test.go` alongside `resource_test.go`. Neither
move changes the test's substance or its "not broken" finding. Recommend (non-blocking): relocate
into `playernpc/` in a follow-up so the equipment-decode coverage lives next to the type it tests,
consistent with `resource_test.go`/`model_test.go` for the rest of the package's test suite — but
this is tidiness, not a correctness or convention violation, and does not block this review.

## EXT-02 real-decode-path check

Confirmed for every new EXT-02 test across both rounds that it exercises a genuine decode, not a
hand-built model:

- `atlas-player-npcs/data/map/rest_test.go`: `TestGetById_HttpDecode`/`TestGround_HttpDecode` —
  real `httptest.NewServer`, real `Processor.GetById`/`Processor.Ground` calls reading
  `DATA_SERVICE_URL` via `t.Setenv`, decoding a JSON:API fixture through `requests.GetRequest`.
- `atlas-player-npcs/inventory/rest_test.go`: `TestGetByCharacterId_HttpDecode` — full
  inventory→compartments→assets `included` chain served over httptest, decoded via
  `Processor.GetByCharacterId`.
- `atlas-player-npcs/character/rest_test.go`, `data/npc/rest_test.go`: same shape, verified present
  and using `httptest.NewServer` + real `Processor` calls.
- `atlas-player-npcs/playernpc_equipment_decode_test.go` and
  `atlas-channel/playernpc/rest_test.go`'s `TestRestModel_EquipmentRoundTrip`: both round-trip a
  non-empty `Equipment`/`EquipmentRestModel` slice through real `jsonapi.Marshal` →
  `jsonapi.Unmarshal` (not `httptest`, since the seam under test is struct-level JSON:API codec
  behavior, not an HTTP client) — the atlas-channel version additionally runs the result through
  `Extract` and asserts the domain model. Both report "decode is not broken," and both are provably
  real (they would fail if `Equipment`/`EquipmentRestModel` had a relationship-walk bug — verified
  by reading the actual marshal/unmarshal calls, not trusting the report's prose).
- `atlas-query-aggregator/playernpc/rest_test.go`: `TestGetEligibility_HappyPath` and
  `TestGetEligibility_InfrastructureError` — real `httptest.NewServer`, asserts the query string the
  real request builds and decodes a real (non-JSON:API) JSON body via the real `GetEligibility`
  call; the second test confirms a 5xx surfaces as a genuine error rather than a silently-eligible
  false positive, then separately exercises `NewUnavailableEligibility`'s fail-closed shape.

All of these are httptest-/jsonapi-backed against real production code paths, not fixtures asserted
against a hand-built struct. This closes the EXT-02 finding as intended.

## Other findings swept (no separate ruling requested, checked for completeness)

- DOM-01 (atlas-channel `Builder.Build()`): now `(Model, error)` with `ErrMissingId`/
  `ErrMissingCharacterId`/`ErrIncoherentPosition`, plus `MustBuild()`. Call-site sweep
  (`grep -rn "playernpc.NewBuilder\|\.Build()"` in `atlas-channel`) found zero external callers of
  this builder's `.Build()` — `rest.go`'s `Extract` builds `Model{...}` directly and no test used
  `.Build()` before this round — so no update was needed elsewhere. `go build ./...` for the module
  confirms this.
- FILE-05 (query-aggregator): `EligibilityModel`/`NewEligibilityModel`/`NewUnavailableEligibility`
  moved verbatim to `model.go`; `processor.go` reduced to `Processor`/`ProcessorImpl`/
  `GetEligibility`. `validation/context.go`'s `GetPlayerNpcEligibility` consumer still compiles and
  its tests still pass (`go test ./validation/...` green).
- DOM-02/04/05/13/14/16/32, FILE-02/03/05 (round A's remaining findings): each verified against its
  cited line in the pre-fix source and confirmed resolved — `rest.go` now holds `RestModel`/
  `Transform`/`TransformSlice`/request-body models, `resource.go` now holds every handler routed
  through `Processor`, `requests.go` deleted (package genuinely has no cross-service request
  functions of its own), `EventType` lives in `state.go`.
- SCAFFOLD-06: `atlas-player-npcs` entry added to `docker-compose.core.yml`, alphabetically placed,
  shape matches its `atlas-pets` neighbor; the two missing topic env vars are passed explicitly
  following the existing `atlas-storage`/`atlas-configurations` precedent for topics absent from
  `.env.example` (`grep` confirms `PLAYER_NPC` is indeed absent from `.env.example`).

## Findings

No blocking findings.

Non-blocking:
1. `playernpc/resource.go:381-389` — `Processor.Eligibility` conflates the character-lookup error
   and the `countByName` error under one return value, and the handler's `errors.Is(err,
   requests.ErrNotFound)` check now applies to both. Inert today (`countByName` cannot produce that
   sentinel), but worth a comment or a split return if `Eligibility`'s error surface ever grows.
2. `data/map/rest.go:102-108` — `SetToOneReferenceID`/`SetToManyReferenceIDs` stubs on
   `GroundRequestRestModel` are dead code (it is never an `Unmarshal` target); harmless, could be
   dropped in a later pass without losing EXT-01 coverage.
3. `playernpc_equipment_decode_test.go` — acceptable at the module root per repo-wide precedent
   (`wiring_test.go` and siblings exist in every service), but now that round A has landed and the
   `playernpc/`-exclusivity constraint that forced this location is gone, moving it into
   `playernpc/` as `playernpc_test.go` would be tidier and is a one-file follow-up, not required.

## Not evaluable

None — all four questions and the primary behavior-preservation question were answerable within
this diff's surface plus `libs/atlas-rest/server`, `libs/atlas-rest/CLAUDE.md`, and design.md §8.3,
which the diff's correctness genuinely depends on.
