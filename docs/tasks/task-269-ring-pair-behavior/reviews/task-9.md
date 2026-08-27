# Review: Task 9 — atlas-channel/ring REST consumer

Range: `411f368a8..41cb9b35c` (one commit, `41cb9b35c`)
Files: `services/atlas-channel/atlas.com/channel/ring/{model.go,rest.go,requests.go,rest_test.go}` (all new, +326)

## Verdict

APPROVED

## Scope

Matches the brief exactly: four new files in `atlas-channel/ring`, no caller
added (correctly deferred to Task 10). Reviewed the diff plus the producing
side (`services/atlas-cashshop/atlas.com/cashshop/ring/{rest.go,model.go,resource.go}`),
the `door` package's paginated-URL idiom it mirrors
(`door/rest.go`, `door/requests.go`), the JSON:API library actually used at
runtime (`libs/atlas-rest/requests/{paged.go,response.go}` →
`github.com/jtumidanski/api2go/jsonapi`), and `libs/atlas-rest/CLAUDE.md`'s
relationship-stub gotcha.

## Field-by-field: producing `RestModel` vs. consuming `RestModel`

| cashshop field (json tag) | channel field | verdict |
|---|---|---|
| `Id` (`json:"-"`, uuid.UUID) | `Id` (`json:"-"`, string) | matches — set via `SetID`/JSON:API `id`, parsed in `Extract` |
| `pairId` | `pairId` (string) | matches, parsed in `Extract` |
| `characterId` | `characterId` | matches |
| `partnerCharacterId` | `partnerCharacterId` | matches |
| `assetId` | `assetId` | decoded but not carried into `Model` — correct: brief's getter list does not name it |
| `itemTemplateId` | `itemTemplateId` | matches |
| `ringType` | `ringType` | matches, validated against `TypeCouple`/`TypeFriendship` |
| `state` | `state` | matches, validated against `StateActive`/`StateBroken`/`StateExpired` |
| `createdAt` | (absent) | correctly dropped — brief's getter list has no `CreatedAt()` |
| `cashId` (int64) | `cashId` (int64) | matches |
| `partnerCashId` (int64) | `partnerCashId` (int64) | matches |
| `partnerName` | `partnerName` | matches |

No silent field drop or mismap. `AssetId`/`CreatedAt` omissions are
deliberate and match the brief's stated interface, not oversights.

## Int64 `cashId` path (fault-injected)

Concern: does the real decode path (`jsonapi.Unmarshal` from
`github.com/jtumidanski/api2go/jsonapi`) preserve `9007199254740993` past a
`float64` intermediate, or does the test merely echo a Go struct through
itself?

I wrote a throwaway test (`ring/zzz_probe_test.go`, removed after running —
`git status --porcelain ring/` is clean) that fed a real JSON:API document
through `jsonapi.Unmarshal` into `[]RestModel`:

```
"attributes":{...,"cashId":9007199254740993,...}
```

Result: `result[0].CashId == 9007199254740993`, test passed. This confirms
`api2go/jsonapi.Unmarshal` decodes JSON-number attributes directly into the
struct's typed `int64` field (no `map[string]interface{}`/`float64`
intermediate for this path), so the committed `TestExtractLargeCashIdSurvives`
test, while it only exercises `Extract` on a hand-built `RestModel` and not
the wire decode, is not hiding a real precision-loss bug — the wire decode
was independently verified to be safe.

Non-blocking note: the committed test suite has no test that exercises
`jsonapi.Unmarshal` itself (only `Extract` on a hand-constructed struct). That
is reasonable for this task (no caller wires the HTTP round-trip yet) but
Task 10, which does add the `DrainProvider` call site, should carry an
httptest-backed integration test per the `libs/atlas-rest/CLAUDE.md`
guidance, since that is where a relationship-stub or decode regression would
actually be caught.

## Error paths (fault-injected)

Mutated `ring/rest.go` in place, ran the affected test, restored the file
(`cp` backup + verified `git status --porcelain ring/` empty afterward):

1. Removed the `if err != nil { return Model{}, err }` guard after
   `uuid.Parse(rm.Id)` → `TestExtractInvalidId` failed
   (`expected error for invalid id, got nil`). Confirms the test pins the
   error return, not a "looks right" tautology.
2. Reduced `parseType` to `return Type(s), nil` unconditionally (dropping the
   `switch`/`default` guard) → `TestExtractUnknownRingType` failed
   (`expected error for unknown ring type, got nil`). Confirms the same for
   the ring-type guard.

Both `rest.go:52-55` (id) and `rest.go:82-89` (`parseType`) are load-bearing;
the tests would catch a regression in either.

## Pre-ruled items — verified, not re-litigated

- `requestByCharacterId` (`ring/requests.go:24-30`) returns `(string, error)`,
  not `requests.Request[[]RestModel]`. `door/requests.go:20-32`'s
  `inFieldUrl`/`byOwnerUrl` use the identical bare-URL-for-`DrainProvider`
  idiom and the doc comment there gives the same reason (pagination requires
  per-page `page[number]`/`page[size]` params `DrainProvider` appends
  itself). `ring/requests.go`'s own comment (lines 18-23) states this
  rationale and cites the door precedent. Task 10 can consume this exactly
  as `door`'s processor consumes its own URL builders — confirmed by reading
  `DrainProvider`'s signature in `libs/atlas-rest/requests/paged.go:116`,
  which takes a bare `url string`, not a `requests.Request[T]`.
- `Type`/`State` constants re-declared channel-side
  (`ring/model.go:10-25`) rather than promoted to `libs/atlas-constants` —
  doc comment correctly distinguishes this from
  `ClassificationRing` (`libs/atlas-constants/item/constants.go:24`), mirrors
  cashshop's own rationale comment in `cashshop/ring/model.go`.

## JSON:API relationship-stub gotcha

`libs/atlas-rest/CLAUDE.md` flags that `api2go/jsonapi.Unmarshal` errors out
on any target struct lacking `UnmarshalToOneRelations`/
`UnmarshalToManyRelations` when the response has a `relationships` block,
even if the caller doesn't need it — and this has bitten two prior tasks.
`ring/rest.go:41-45` implements both `SetToOneReferenceID` and
`SetToManyReferenceIDs` as no-ops, matching the `door` pattern. Correct
precaution taken proactively.

## Test honesty

All four tests in `rest_test.go` were fault-injection-verified above (the
two error-path tests) or independently reasoned about (the happy-path and
large-cashId tests, which exercise real UUID parsing and enum validation
inside `Extract`, not just field echo — `Extract` does non-trivial work:
`uuid.Parse` twice and two enum switches). None are tautological in the
sense flagged as a pattern on this branch.

## Build/test status

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./ring/... && go test ./ring/... -v
```
All pass (4/4 tests), confirmed directly (not from the implementer's report).

## Non-blocking findings

1. No test exercises the actual `jsonapi.Unmarshal` wire-decode path for this
   package (only `Extract` on a hand-built struct) — reasonable for this
   task since there is no caller yet, but flag for Task 10 to add an
   httptest-backed integration test per `libs/atlas-rest/CLAUDE.md`'s
   "How to be sure you got it right" guidance, given this package sits
   exactly in the failure mode that guidance describes.
2. Doc comments in `ring/model.go:73,80,88` cite specific line ranges in
   `atlas-cashshop/ring/model.go` (`81-88`, `90-95`, `97-101`) for
   `CashId`/`PartnerCashId`/`PartnerName`. I did not re-verify these exact
   line numbers against the cashshop file's current state (cosmetic; the
   semantic content of each comment is correct against the file as read).

## Not evaluable

None — the full surface (four new files) was read and exercised.

## Report honesty check

The implementer's report (`task-9-report.md`) accurately describes the
`requestByCharacterId` signature deviation, flags it explicitly as a
resolved ambiguity rather than hiding it, and its "Concerns" section matches
what I independently found. No discrepancy between the report and the diff.
