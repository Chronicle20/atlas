# Review: Task 24 — the channel's `CreateMapleLife` factory client

Commit under review: `6880ad6c2` (`646946970..6880ad6c2`)
Brief: `.superpowers/sdd/plan/task-24-brief.md`
Report: `.superpowers/sdd/plan/task-24-report.md`

## Scope

`git diff --stat 6880ad6c2`:

```
character/factory/processor.go      | 17 +++
character/factory/processor_test.go | 140 +++++++++++++++
character/factory/requests.go       | 25 ++++
character/factory/rest.go           | 27 ++++
4 files changed, 209 insertions(+)
```

Matches the brief's four target files exactly. No unrelated files touched.
`scope_confirmed`: reviewed the diff of `6880ad6c2` plus the factory-side
files it must match (`services/atlas-character-factory/.../factory/maple_life.go`,
`maple_life_rest.go`, `resource.go`) and `libs/atlas-rest/requests/post.go`.

## 1. Does it do what the brief said

- `MapleLifeResource = "factory/characters/maple-life"` — `requests.go:14`
  — matches the brief's constant verbatim.
- `CreateMapleLife` signature on `Processor` — `processor.go:23` — matches
  the brief's exact signature and doc comment.
- `requestCreateMapleLife` reuses `getBaseRequest` / `RootUrlFor(ctx,
  "CHARACTER_FACTORY")` and `requests.PostRequest[CreateCharacterResponse]`,
  same shape as `requestCreate` — `requests.go:56-74`. PASS.
- `CreateCharacterResponse` reused unchanged (`rest.go:82-96` untouched by
  the diff) — PASS, per brief instruction.
- `SeedCharacter` left in place, untouched — `processor.go` diff only adds,
  never removes lines from the existing function. PASS.
- Test file adds exactly the two tests the brief specifies
  (`TestCreateMapleLifePostsTheChosenValues`,
  `TestCreateMapleLifeSurfacesStatuses`), with the field/status table the
  brief lays out. PASS.

## 2. Cross-service seam — channel vs. factory

Read both sides directly (not inferred):

**JSON:API type string**
- Channel: `rest.go` — `func (r MapleLifeCreateRestModel) GetName() string { return "maple-life-create" }`
- Factory: `maple_life_rest.go:8` — same, `"maple-life-create"`.
- MATCH.

**Struct fields / json tags**, channel (`rest.go`) vs factory (`maple_life.go:26-37`):

| field | channel type/tag | factory type/tag | match |
|---|---|---|---|
| AccountId | `uint32 json:"accountId"` | `uint32 json:"accountId"` | yes |
| WorldId | `byte json:"worldId"` | `byte json:"worldId"` | yes |
| Name | `string json:"name"` | `string json:"name"` | yes |
| ClassOrdinal | `uint32 json:"classOrdinal"` | `uint32 json:"classOrdinal"` | yes |
| Gender | `byte json:"gender"` | `byte json:"gender"` | yes |
| Face | `uint32 json:"face"` | `uint32 json:"face"` | yes |
| Hair | `uint32 json:"hair"` | `uint32 json:"hair"` | yes |
| HairColor | `uint32 json:"hairColor"` | `uint32 json:"hairColor"` | yes |
| SkinColor | `byte json:"skinColor"` | `byte json:"skinColor"` | yes |
| SP | `byte json:"sp"` | `byte json:"sp"` | yes |

All ten fields match field-for-field and tag-for-tag. PASS.

**Route path.** Factory registers (`factory/resource.go:26-30`):

```go
fr := router.PathPrefix("/factory/characters").Subrouter()
fr.HandleFunc("/maple-life", ...).Methods(http.MethodPost)
```

→ full path `/factory/characters/maple-life`. Client posts to
`root + "factory/characters/maple-life"` (`requests.go:74`), same
concatenation pattern `requestCreate` already uses for
`root + "characters/seed"`. The httptest run in
`TestCreateMapleLifePostsTheChosenValues` proves this resolves to a path
ending in `factory/characters/maple-life` against a real server (verified
independently below, not just by reading the test). PASS.

There is no `from-preset` client yet in `atlas-channel` (grepped, no hits) —
the brief's "sibling `from-preset` client" reference does not exist in this
repo as a pattern to diff against; not a defect, just noting the brief's
premise for that particular check was unfounded in this codebase.

**Envelope / response.** Both sides use the same, unchanged
`CreateCharacterResponse{TransactionId string \`json:"transactionId"\`}`
with JSON:API type `"characters"`. Factory writes `202 Accepted` +
`server.MarshalResponse[CreateCharacterResponse]`; the channel-side
`PostRequest` treats `202` as success (`post.go`: only rejects when status
is none of `200/201/202/204`). PASS.

**Method.** Factory registers `.Methods(http.MethodPost)`; client uses
`requests.PostRequest`. PASS.

## 3. Pattern conformance with the other factory client in-package

`requestCreateMapleLife` mirrors `requestCreate`: same `getBaseRequest`
call, same `requests.ErrorRequest[CreateCharacterResponse](err)` early
return, same `requests.PostRequest[CreateCharacterResponse](root+Resource,
i)` call shape. `CreateMapleLife` on `ProcessorImpl` mirrors
`SeedCharacter`: debug-logs inputs, calls the request function, returns
`resp.TransactionId, nil` or `"", err`. No bespoke shape. PASS.

## 4. The 502/500 judgement call

Read `libs/atlas-rest/requests/post.go` (`createOrUpdate`) directly:

```go
if statusCode == http.StatusBadRequest { return result, ErrBadRequest }
if statusCode == http.StatusNotFound   { return result, ErrNotFound }
if statusCode == http.StatusConflict   { return result, ErrConflict }
if statusCode != http.StatusOK && statusCode != http.StatusCreated &&
   statusCode != http.StatusAccepted && statusCode != http.StatusNoContent {
    return result, errors.New("unknown error")
}
```

No case for 502 or 500; both fall through to the generic `errors.New("unknown
error")`. `ErrServiceUnavailable` does not appear anywhere in `post.go` —
grepped the whole `libs/atlas-rest/requests` package, the only
`ErrServiceUnavailable` producer is in `get.go`'s 503-retry path. The
implementer's claim is **confirmed true**. The weaker "non-nil error only"
assertion for 502/500 in `TestCreateMapleLifeSurfacesStatuses` is therefore
the *correct* assertion, not a gap — asserting
`errors.Is(err, requests.ErrServiceUnavailable)` here would be a false
assertion against actual `post.go` behavior. Not a finding.

## 5. Test honesty — mutation-tested

Ran two independent mutations against the committed code, confirmed a real
failure, then reverted (`git status --porcelain` clean afterward):

1. Path mutation: `MapleLifeResource` changed to
   `"factory/characters/wrong-path"` →
   `TestCreateMapleLifePostsTheChosenValues` failed with
   `expected request path to end with [factory/characters/maple-life], got
   [/factory/characters/wrong-path]`.
2. Positional field-swap mutation: `Face: face,` changed to `Face: hair,` in
   `requestCreateMapleLife` → same test failed with
   `expected request body to contain "\"face\":21000", got
   ...\"face\":31000,\"hair\":31000...` — proving the "each value distinct
   and non-zero" design in the brief actually catches a positional mix-up.

Both mutations reverted; `git status --porcelain
services/atlas-channel/atlas.com/channel/character/factory/` is clean.
PASS — tests are not hollow.

## 6. Build / test verification

- `go vet ./character/factory/...` — clean.
- `go build ./character/factory/...` — `BUILD_OK`.
- `go test ./character/factory/...` — `ok atlas-channel/character/factory`.
- Full-module `go build ./...` for `atlas-channel` currently fails at
  `socket/handler/maple_life_create.go:193: undefined: logCreateFailure`.
  This is **not** caused by commit `6880ad6c2`: `git status` shows
  `socket/handler/maple_life_create.go` as a pre-existing uncommitted
  modification already in this worktree (Task 25's in-progress submit-handler
  rewire, which is out of this review's scope — it calls `CreateMapleLife`
  but was not part of the commit under review and was not touched by it).
  Recorded under Not evaluable / non-blocking for this unit; Task 25's own
  review must gate on it.

## Not evaluable

- Full-module build of `atlas-channel` (blocked by unrelated in-progress
  Task 25 changes already in the worktree, not part of this commit).
- Task 25's actual call site is out of scope for this review (per the
  brief, Task 25 is a separate unit); only the exported signature and error
  contract this client presents to that caller were checked.

## Verdict rationale

No requirement drift, no seam mismatch (type string, all ten fields/tags,
route path, method, envelope all confirmed against the factory source
directly), pattern matches the sibling client, the one flagged judgement
call is verified correct against `post.go`, and the tests are proven
non-hollow by two independent mutations. No blocking findings.
