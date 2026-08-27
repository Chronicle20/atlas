# Review — Task 8: `atlas-cashshop` ring read model (`cashId`, `partnerCashId`, `partnerName`)

Range reviewed: `60f87d269..02b6eeb81` (8 files, 385 insertions / 11 deletions).
Brief: `.superpowers/sdd/plan/task-8-brief.md`. Report:
`.superpowers/sdd/plan/task-8-report.md`.

## Verdict

CHANGES_REQUIRED — see blocking finding below. Everything the implementer
touched inside `ring.ProcessorImpl.GetByCharacterId` is correct and
well-tested, but the actual HTTP contract Task 9 depends on
(`GET /rings?filter[characterId]=`) does not carry the new fields, because
`ring/resource.go`'s `handleGetRings`/`handleGetRing` never call the enriched
path added by this diff.

## Scope confirmed

Diff matches the brief's file inventory plus two files the brief flagged as
conditional/necessary consequences:

- `ring/provider.go` — brief allowed a `byPairId` provider "if needed"; it was
  needed (`GetByPairId`/`byPairIdProvider` added, `ring/processor.go:70-84`
  in the new code — confirmed via diff hunk). Necessary, not scope creep.
- `cashshop/processor_ring.go:191` — the one existing call site of
  `ring.NewProcessor`, updated to pass `p.chaP` for the new fourth
  parameter. `cashshop.ProcessorImpl` already holds `chaP character.Processor`
  (`cashshop/processor.go:128`) — this is a mechanical consequence of the
  `NewProcessor` signature change, not unrequested scope.

`ring/entity.go` and `cashshop/inventory/asset/model.go` — confirmed
byte-for-byte unmodified (`git diff 60f87d269..02b6eeb81 -- ring/entity.go
cashshop/inventory/asset/model.go` is empty). No migration was added. This
satisfies design.md §5's rejection of a stored column ("Add cashId to
cash_rings as a stored column. Rejected... The join is cheap and
atlas-cashshop owns both sides.").

## Blocking

- `services/atlas-cashshop/atlas.com/cashshop/ring/resource.go:60-67`
  (`handleGetRings`) and `ring/resource.go:86` (`handleGetRing`) — the REST
  handlers behind `GET /rings?filter[characterId]=` and `GET /rings/{ringId}`
  build their response via `byCharacterIdPagedProvider(...)` → `Make` (paged
  handler) and the free function `GetById` (single handler), **not** via
  `ring.ProcessorImpl.GetByCharacterId`/`GetById`. Neither of those code paths
  runs the enrichment this diff added (`ring/processor.go`'s
  `GetByCharacterId`, which sets `CashId`/`PartnerCashId`/`PartnerName` via
  `Model.Builder()`). Every ring returned by the actual HTTP endpoint will
  therefore have `cashId: 0, partnerCashId: 0, partnerName: ""` regardless of
  the real data.

  This is not a hypothetical: design.md §4 diagrams the contract explicitly —
  `atlas-cashshop  GET /rings?filter[characterId]=N   (+cashId,
  +partnerCashId, +partnerName)` (design.md:291) — and Task 9's own brief
  states "Route. `GET /rings?filter[characterId]=<id>`, registered at
  `.../ring/resource.go:29`... Consumes from Task 8: the `cashId`,
  `partnerCashId`, `partnerName` JSON fields" (`task-9-brief.md:17-24`).
  `atlas-channel` is a separate service; it can only reach this data over
  that HTTP route, not by calling `ring.ProcessorImpl` directly. As shipped,
  Task 9's channel-side cache will populate every ring with zeroed
  `cashId`/`partnerCashId` and an empty `partnerName`, defeating the
  feature's entire purpose (couple/friendship ring pairing and the partner
  name shown on spawn/appearance/character-info packets).

  No test catches this: `ring/resource_test.go` has zero references to
  `CashId`/`PartnerCashId`/`PartnerName` (`grep -c` = 0), and the new
  `processor_test.go` only exercises `ProcessorImpl.GetByCharacterId`
  directly (in-package Go call), never through `resource.go`'s HTTP handler.
  A test that actually hit `handleGetRings` and asserted the enriched fields
  in the JSON:API response would have failed and caught this.

  Fix direction (not prescribing implementation): `handleGetRings` and
  `handleGetRing` need to route through `ProcessorImpl.GetByCharacterId` /
  `ProcessorImpl.GetById` (or equivalent enrichment) instead of the raw
  provider/`Make`/free-function paths, and a resource-level test needs to
  assert the enriched fields survive into the marshalled response.

## Verified correct (non-blocking)

- **`gofmt -l` clean.** `gofmt -l services/atlas-cashshop/atlas.com/cashshop`
  produced no output; import blocks in every changed file (`processor.go`,
  `provider.go`, `builder.go`, `model.go`, `rest.go`, `processor_test.go`,
  `cashshop/processor_ring.go`) are alphabetically sorted within their
  groups and match the existing repo convention of merging local
  (`atlas-cashshop/...`) packages with stdlib in the first block (see
  pre-existing `cashshop/processor_ring.go` import block, which already did
  this before this diff).
- **Build and tests green.** `go build ./...` clean;
  `go test ./...` — every package with tests reports `ok`, including
  `atlas-cashshop/ring`.
- **`Model.Builder()` pattern.** Matches an existing repo convention
  (confirmed: `services/atlas-guilds/atlas.com/guilds/guild/member/builder.go:123`
  has the identical `func (m Model) Builder() *Builder` shape), not an
  invented pattern.
- **Enrichment fail-soft behavior is correctly implemented and tested.**
  `ring/processor.go:63-104` (`GetByCharacterId`): own-asset lookup, sibling
  lookup via `GetByPairId`, and `chaP.GetById()` are each wrapped in
  `if ..., err := ...; err == nil { ... }`, so a failure in any one leaves
  the corresponding field at its zero value and never turns into a returned
  error. All three brief-specified cases are covered by
  `processor_test.go`'s three subtests (both halves present / sibling row
  missing / character service unavailable), and the "character service
  unavailable" subtest genuinely exercises a live connection-refused
  failure (`httptest.NewServer` started then `.Close()`d) rather than an
  unset env var — a real RED/GREEN-honest test, not one that would pass
  either way.
- **No N+1 / unbounded fan-out on `GetByCharacterId`.** Per returned half:
  one asset lookup for itself, one `GetByPairId` query (bounded to the
  handful of rows sharing a `pairId`, normally ≤2), one asset lookup for
  the sibling, and one character lookup — all bounded per-half, not nested
  over an unbounded collection. `GetByPairId` correctly excludes the row
  matching `half.Id()` before taking the first remaining row
  (`processor.go:79-88`), so it is also correct for the self-referential
  edge case (a row cannot match itself, since ids are distinct UUIDs).
- **`character.Processor.GetById()(id)` call shape.** Matches the existing
  decorator-curried signature (`character/processor.go:41`:
  `func (p *ProcessorImpl) GetById(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error)`).
- **`ring.NewProcessor` signature change has exactly one call site.**
  Confirmed via repo-wide grep — only `cashshop/processor_ring.go:191`
  calls `ring.NewProcessor`; no other caller was missed.
- **No new domain type or constant needed checking against
  `libs/atlas-constants/`** — the three new fields are service-local
  computed values (an `int64` id and a `string` name), not domain
  enums/constants.
- **`TestTransform` extension matches the brief's table exactly**
  (`rest_test.go:670-702`): `cashId: 9007199254740993` (> 2^53, proving
  no float-narrowing), `partnerCashId: -1`, `partnerName: "PartnerChar"`,
  each asserted against the corresponding `RestModel` field.

## Not evaluable

- Whether `atlas-channel`'s Task 9 implementation itself would have caught
  the resource.go gap independently — out of this unit's scope; Task 9 has
  not landed yet in this diff range.

## Not evaluable / out of scope for this diff

None beyond the item above — the review surface (the diff plus
`ring/resource.go`, which the diff's `GetByCharacterId` change should have
reached but did not) was fully inspectable.

---

## Fix round 1 re-review (Ruling 22 finding)

Scope: `.superpowers/sdd/plan/review-00dc42a52..f8c5f9d42.diff` only —
`ring/processor.go`, `ring/resource.go`, `ring/resource_test.go`. Not a
re-review of Task 8 as a whole.

### Finding disposition: ADDRESSED

`handleGetRings` (`ring/resource.go:56-58`) now constructs
`chaP := character.NewProcessor(d.Logger(), d.Context())` and
`p := NewProcessor(d.Logger(), d.Context(), db, chaP)`, then calls
`p.GetByCharacterIdPaged(uint32(parsed), page)` — replacing the old
`byCharacterIdPagedProvider`/`model.MapPaged(Make)` pair that never touched
enrichment. `handleGetRing` (`ring/resource.go:82-84`) does the same
construction and calls `p.GetById(ringId)` in place of the free `GetById`
call. Both routes now reach `(*ProcessorImpl).enrich`.

Fault-injected to confirm the new test is not tautological: temporarily
replaced `enrich`'s body with `return half` (a no-op) in
`ring/processor.go`, removed the now-unused `asset` import to keep it
compiling, and ran
`go test ./ring/... -run TestGetRingsAndGetRingCarryEnrichedFields -v`.
Both subtests failed as expected —
`expected: int(9001) / actual: float64(0)` for `cashId`,
`expected: int(9002) / actual: float64(0)` for `partnerCashId`, and
`expected: "Partner" / actual: ""` for `partnerName` — on both the list
and single-resource routes. Restored the file from the pre-fault copy and
confirmed `git status --porcelain` shows no diff of my own (only the
pre-existing `go.work.sum` modification and untracked task-folder files,
neither touched here).

### Paging semantics

`GetByCharacterIdPaged` (`ring/processor.go`) still runs
`byCharacterIdPagedProvider(p.t, characterId, page)(db)()` — the same
`LIMIT`/`OFFSET` DB-level query the route used before — and only enriches
the page's own rows, not the character's full holding. It returns the same
`model.Paged[Model]{Items, Total, Page}` envelope.
`TestAnotherTenantCannotReadTheseRings`'s `doc.Meta["total"]` assertions
(unmodified) still pass per the report's `go test` output, confirming no
regression to the paging envelope.

### Fail-soft behavior

Unchanged and still tested. `enrich` (`ring/processor.go`) still swallows
each of the three lookup failures (own asset, sibling asset, character
lookup) to the zero value via `if ..., err := ...; err == nil { b.Set... }`
— no `return nil, err` path added for any of the three. The pre-existing
`processor_test.go`'s `TestGetByCharacterIdEnrichesCashIdAndPartnerName`
("sibling row missing" / "character service unavailable" subtests) is
unmodified by this fix diff and its assertions target `enrich`'s logic,
which was only *relocated* (extracted into a shared method used by all
three of `GetByCharacterId`, `GetByCharacterIdPaged`, and `GetById`), not
altered.

The 404-for-unknown/cross-tenant-ring behavior is also preserved:
`handleGetRing` still checks `errors.Is(err, gorm.ErrRecordNotFound)`
before enrichment runs (`ring/resource.go`), and `(*ProcessorImpl).GetById`
returns the raw lookup error before calling `enrich` (`ring/processor.go`:
`half, err := GetById(...); if err != nil { return Model{}, err }`), so a
cross-tenant/unknown id still surfaces `gorm.ErrRecordNotFound` unwrapped.

### `ring/entity.go` and migrations

`git diff 00dc42a52..f8c5f9d42 -- ring/entity.go` is empty — untouched.
No new migration file in the diff (`git log 00dc42a52..f8c5f9d42 --stat`
shows only `processor.go`, `resource.go`, `resource_test.go` changed).
Consistent with design.md §5's rejection of a stored column.

### Formatting

`gofmt -l ring/` and `gofmt -l .` (module root) both produce no output —
clean, including `resource.go`'s changed import block (`character` added
to the local group, `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`
removed from the atlas-libs group since `handleGetRings`/`handleGetRing`
no longer call `tenant.MustFromContext` directly — that now happens inside
`NewProcessor`).

### Per-request processor construction

Matches the repo's existing pattern for this service, not a new cost the
surrounding code avoids. Grepped sibling resource files in the same
module: `wallet/resource.go` (`NewProcessor(d.Logger(), d.Context(), db).GetByAccountId(...)`
etc.) and `coupon/resource.go` (`NewProcessor(d.Logger(), d.Context(), db)`
at multiple handlers) both construct a fresh processor per request inside
the handler closure. The ring fix's
`chaP := character.NewProcessor(...); p := NewProcessor(..., chaP)` is the
same shape, with one extra dependency threaded through because `enrich`
needs a `character.Processor`. Not a deviation and not a meaningfully new
per-request cost (both `character.NewProcessor` and `ring.NewProcessor`
are cheap struct constructions with no I/O at construction time — I/O only
happens when `enrich`'s three lookups run, which the fix's own report
notes is bounded per-half).

### New breakage check

None found. `go build ./... && go test ./...` (per the report, reproduced
here for `./ring/...`) is green; `TestAnotherTenantCannotReadTheseRings`
still passes with only benign extra gorm "record not found" trace log
noise (that test's seeded rings have no matching asset rows, which is
expected fail-soft output, not an assertion failure).

### Not evaluable

- Task 9 (`atlas-channel`) consumption of the now-enriched fields — out of
  this diff's range, not re-verified here.

### Verdict

APPROVED — the open blocking finding from the original Task 8 review is
ADDRESSED, with fault-injection confirming the closing test is genuine
(not tautological), and no new breakage introduced by the fix diff.
