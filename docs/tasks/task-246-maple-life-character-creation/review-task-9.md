# Review: Task 9 — atlas-character-factory REST client in atlas-channel

**Commit range:** `dac10b874..ad1557fb9` (single commit `ad1557fb9`)
**Brief:** `.superpowers/sdd/plan/task-9-brief.md` (Controller addendum)
**Report:** `.superpowers/sdd/plan/task-9-report.md`

## Scope

`git diff --stat dac10b874..ad1557fb9`:

```
services/atlas-channel/atlas.com/channel/character/factory/processor.go       |  50 ++
services/atlas-channel/atlas.com/channel/character/factory/processor_test.go  | 200 ++++
services/atlas-channel/atlas.com/channel/character/factory/requests.go        |  50 ++
services/atlas-channel/atlas.com/channel/character/factory/rest.go            |  69 ++
4 files changed, 369 insertions(+)
```

Matches the brief's file list exactly. `services/atlas-login/` has zero diff in this range (`git diff --stat dac10b874..ad1557fb9 -- services/atlas-login/` returns empty), and `deploy/` has zero diff in this range. Scope confirmed — matches the brief's task boundary; no in-flight Task-10 spillover found in this range.

## Checks

### 1. `RestModel` field set / JSON tags match atlas-login's `rest.go` exactly

`diff services/atlas-login/atlas.com/login/character/factory/rest.go services/atlas-channel/atlas.com/channel/character/factory/rest.go` → **files are byte-identical** (diff exit 0). Confirmed field-by-field via direct read of both files: `Id json:"-"`, `AccountId json:"accountId"`, `WorldId json:"worldId"`, `Name`, `Gender`, `JobIndex`, `SubJobIndex`, `Face`, `Hair`, `HairColor`, `SkinColor`, `Top`, `Bottom`, `Shoes`, `Weapon`, `Level`, `Strength`, `Dexterity`, `Intelligence`, `Luck`, `Hp`, `Mp`, `MapId` — same types, same tags, same order. `CreateCharacterResponse{TransactionId string `json:"transactionId"`}` and its `GetName`/`GetID`/`SetID` are also identical. **PASS.**

### 2. `transactionId` divergence present and correct

`services/atlas-channel/.../processor.go:38-46`:
```go
resp, err := requestCreate(...)(p.l, p.ctx)
if err != nil {
    return "", err
}
return resp.TransactionId, nil
```
`resp` is `CreateCharacterResponse`, and `resp.TransactionId` is populated directly from the JSON:API `attributes.transactionId` field via `json:"transactionId"` — not derived from `GetID()`/`SetID()` (which round-trip through `TransactionId` too, but the code reads the struct field directly, avoiding any risk of the two diverging). This matches the response shape the test fixtures use: `{"data":{"type":"characters","id":"tx-42","attributes":{"transactionId":"tx-42"}}}`. Verified test `TestSeedCharacterReturnsTransactionId/accepted` asserts `txId == "tx-42"` and it passes (`go test` output below). atlas-login's `processor.go` at the same call site discards the value (`_, err := requestCreate(...)`) — the divergence is present, minimal, and correctly commented in the new `Processor` interface doc comment. **PASS.**

### 3. Defaults match atlas-login's; session-supplied ids pass through unaltered

`requests.go:33-43` (channel) is byte-identical to `requests.go` (login): `Level: 1`, `Hp: 50`, `Mp: 5`, `MapId: 0` hard-coded; `AccountId: accountId` and `WorldId: worldId` pass the caller's arguments straight through, unaltered and undefaulted. Test `TestSeedCharacterSendsSessionSuppliedIds` asserts the captured request body contains `"accountId":1001` (the test's `seedAccountId`), `"worldId":0`, `"level":1`, `"hp":50`, `"mp":5`, `"mapId":0` — all pass. **PASS.**

### 4. Outbound call is tenant-aware

`processor.go` calls `requestCreate(...)(p.l, p.ctx)` where `p.ctx` is the context passed into `NewProcessor`, threading the tenant through to `requests.PostRequest`, identical to the atlas-login pattern (`requests.RootUrlFor(ctx, ...)` and the same `(l, ctx)` invocation shape). Test `TestSeedCharacterCarriesTenantHeader` builds a real tenant via `tenant.Create(...)`, puts it in the context via `tenant.WithContext`, and asserts the captured `r.Header.Get(tenant.ID)` equals the tenant's id — this test actually fires (captured header assigned inside the httptest handler, checked after the call) and passes. **PASS.**

### 5. `services/atlas-login/` unmodified

`git diff --stat dac10b874..ad1557fb9 -- services/atlas-login/` → empty. **PASS.**

### 6. No service-config seed row / no ingress entry added

`git diff dac10b874..ad1557fb9 -- deploy/` → empty. Only the four `factory/` files under `services/atlas-channel` changed. **PASS** — matches the brief's pre-verified finding that `CHARACTER_FACTORY_SERVICE_URL`/`BASE_SERVICE_URL` resolution and the existing `^/api/characters/seed(/.*)?$` route already cover this.

### 7. Standard checks — no stubs, no `*_testhelpers.go`, tests non-vacuous

- No placeholder/stub bodies; `SeedCharacter` does real work (builds the request, posts, returns the real error/value).
- No `*_testhelpers.go` file added; test setup uses `httptest.NewServer` and env-var injection (`t.Setenv`), consistent with the repo convention of not adding test-only constructor files.
- Ran `go test ./character/factory/... -v` from `services/atlas-channel/atlas.com/channel`:
  ```
  --- PASS: TestSeedCharacterSendsSessionSuppliedIds (0.03s)
  --- PASS: TestSeedCharacterCarriesTenantHeader (0.00s)
  --- PASS: TestSeedCharacterPostsToCharactersSeed (0.00s)
  --- PASS: TestSeedCharacterReturnsTransactionId (0.00s)
      --- PASS: TestSeedCharacterReturnsTransactionId/accepted (0.00s)
      --- PASS: TestSeedCharacterReturnsTransactionId/rejected (0.00s)
      --- PASS: TestSeedCharacterReturnsTransactionId/unreachable (0.00s)
  PASS
  ```
- Each `httptest.Server` handler in the four table cases assigns a `captured*` variable that is read and asserted after `callSeedCharacter` returns — none of the captures are dead. The "rejected" case (`processor_test.go`, `TestSeedCharacterReturnsTransactionId/rejected`) asserts `errors.Is(err, requests.ErrBadRequest)` and `txId == ""`, both of which are meaningful (error propagated, not swallowed; empty id on failure, not the caller's tx-42 fixture from a different subtest leaking through). The already-resolved item from the task prompt (brief's "contains `400`" text vs. the library's fixed `ErrBadRequest` sentinel) is confirmed handled correctly here — the implementer used the stronger `errors.Is` check instead of a substring match, and I independently confirm this against `libs/atlas-rest/requests/post.go`/`get.go` as described in the prompt. **PASS.**

## Findings

None. No blocking or non-blocking findings identified in this unit.

## Not evaluable

- Task 13 (the consumer of this `Processor.SeedCharacter`) and Task 14 (the async seed-status correlation on `transactionId`) are out of scope for this unit and were not reviewed here — only the shape of the returned value that Task 14 will consume was checked.

## Verdict

APPROVED.
