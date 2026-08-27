# Task 16 batch `channel-a` review

Commit reviewed: `aea948a4a` ("feat(atlas-channel): add Transform and round-trip tests")
Brief: `.superpowers/sdd/plan/task-16-brief-channel-a.md`
Report: `.superpowers/sdd/plan/task-16-channel-a-report.md`

## Scope

`git show --stat aea948a4a` confirms exactly 10 files touched, all within
`services/atlas-channel/atlas.com/channel/{account,buddylist,character/buff,character/teleportrock,data/item}`:

```
 account/rest.go               | 19 +++++++
 account/rest_test.go          | 37 +++++++++++++
 buddylist/rest.go             | 22 ++++++++
 buddylist/rest_test.go        | 60 ++++++++++++++++++++++
 character/buff/rest.go        | 17 ++++++
 character/buff/rest_test.go   | 36 +++++++++++++
 character/teleportrock/rest.go        |  7 +++
 character/teleportrock/rest_test.go   | 33 ++++++++++++
 data/item/rest.go             |  9 ++++
 data/item/rest_test.go        | 28 +++++++++-
 10 files changed, 267 insertions(+), 1 deletion(-)
```

Matches the brief's five-package `channel-a` batch. No file outside these five packages was touched.

## 1. Field coverage — derived independently per package

### account
`account/model.go` `Model` struct fields (lines 4-18): `id, name, password, pin, pic, birthDate, loggedIn, lastLogin, gender, banned, tos, language, country, characterSlots` — 14 fields.

`Transform` (`account/rest.go:80-97`) sets `Id, Name, Password, Pin, Pic, BirthDate, LoggedIn, LastLogin, Gender, Banned, TOS, Language, Country, CharacterSlots` — all 14 fields present, `Id` re-encoded via `strconv.FormatUint`, `LoggedIn` cast to `byte`. Matches `Extract` (`account/rest.go:99-121`) field-for-field, which reads every one of the same `RestModel` fields. PASS.

### buddylist
`buddylist/model.go` `Model` struct fields (lines 9-15): `tenantId, id, characterId, capacity, buddies` — 5 fields.

`Transform` (`buddylist/rest.go`) sets `Id, TenantId, CharacterId, Capacity, Buddies` — all 5 covered. Nested `buddy.Model` -> `buddy.RestModel` conversion inside `Transform` uses only `buddy.Model`'s public accessors (`CharacterId()`, `Group()`, `Name()`, `ChannelId()`, `InShop()`, `Pending()`), correctly avoiding cross-package unexported-field access. `buddy.Model.listId` has no getter and no `RestModel.ListId` counterpart; `buddy.Extract` (verified) never sets it, so it round-trips as the zero value regardless of `Transform` — pre-existing gap, not introduced by this task. PASS.

### character/buff
`character/buff/model.go` `Model` struct fields (lines 22-30): `sourceId, level, duration, changes, createdAt, expiresAt, noExpiry` — 7 fields.

`Transform` (`character/buff/rest.go`) sets `SourceId, Level, Duration, Changes, CreatedAt, ExpiresAt, NoExpiry` — all 7 covered, including the nested `stat.Model` slice converted via the pre-existing `stat.Transform` (unmodified by this commit). PASS.

### character/teleportrock
`character/teleportrock/model.go` `Model` struct fields (lines 8-11): `regular, vip` — 2 fields.

`Transform` (`character/teleportrock/rest.go:34-39`) sets `Regular, Vip` — both covered. PASS.

### data/item
`data/item/model.go` `Model` struct fields (lines 4-7): `itemId, name` — 2 fields.

`Transform` (`data/item/rest.go`) sets `Id` (re-encoded via `strconv.FormatUint`) and `Name` — both covered. PASS.

No sibling-package analogy was used; each package's own `model.go` was read directly.

## 2. Faithfulness to `Extract`, not the JSON surface

Verified against the actual `Extract` bodies:

- `character/buff/rest.go:51-65` — `Extract` builds `Model{sourceId: rm.SourceId, level: rm.Level, duration: rm.Duration, changes: cs, createdAt: rm.CreatedAt, expiresAt: rm.ExpiresAt, noExpiry: rm.NoExpiry}`. `rm.Id` is never referenced. `RestModel.Id` is tagged `json:"-"` (`character/buff/rest.go:11`) and `Model` carries no id field. `Transform` correctly does not populate `RestModel.Id`. Claim CONFIRMED.
- `character/teleportrock/rest.go:41-43` — `Extract` is `return NewModel(rm.Regular, rm.Vip), nil`; `rm.Id` is never referenced. `RestModel.Id` is `json:"-"` (line 8) and `Model` carries no id field. `Transform` correctly does not populate it. Claim CONFIRMED.

Both implementer claims verified against the actual `Extract` source, not accepted on assertion.

## 3. No behavior change outside the addition

`git show aea948a4a` diff for every `rest.go` file is a pure addition (a new `Transform` function inserted before the existing `Extract`); no `Extract` body line, no `Build()` line, and no `RestModel` struct field was touched in any of the five packages. Confirmed by reading each hunk directly (no `-` lines inside any `rest.go` diff, only `+` lines and unchanged context). FR-17 and PRD §5 hold.

## 4. No file overwritten

`git show --stat aea948a4a` totals `267 insertions(+), 1 deletion(-)`. The single deletion is in `data/item/rest_test.go`:
```
-import "testing"
```
replaced by a multi-line `import (...)` block adding `"reflect"` — a mechanical import-statement expansion, not a content loss. `buddylist/rest_test.go`, `character/buff/rest_test.go`, and `character/teleportrock/rest_test.go` are new files (`+N -0`). `account/rest_test.go` is pure append (`+37 -0`, existing `TestExtractBirthDate` etc. untouched). No pre-existing test was destroyed.

## 5. Tests actually constrain the code (mutation check)

Performed on `account/rest.go`: changed `Gender: m.gender,` to `Gender: 0,` inside `Transform`, then ran:
```
go test ./account/... -run TestTransformRoundTrip -v
```
Result: `FAIL — round trip mismatch. Expected {... gender:2 ...}, got {... gender:0 ...}`. The test detects a single dropped/altered field. Reverted the edit; confirmed via `git diff --stat account/ buddylist/ character/buff/ character/teleportrock/ data/item/` (empty output) that the tree is clean.

## Module test run (scoped)

```
cd services/atlas-channel/atlas.com/channel
go test ./account/... ./buddylist/... ./character/buff/... ./character/teleportrock/... ./data/item/... -run TestTransformRoundTrip -v
```
All five `TestTransformRoundTrip` cases PASS (plus unrelated pre-existing tests in the same packages, unaffected).

## Not evaluable

None. All five packages' `Model`, `RestModel`, `Extract`, and `Transform` were read directly from source; no part of the review surface required inference from an unread file.

## Verdict rationale

All five `Transform` implementations are correct, complete inverses of their respective `Extract` functions, derived independently from each package's own `model.go`. The two `RestModel.Id`-not-populated claims were verified against the actual `Extract` source rather than accepted on assertion, and both hold. The diff is strictly additive (one mechanical import-line replacement aside). No pre-existing test was destroyed. An independent mutation test confirms the round-trip tests genuinely constrain field-level correctness. No findings.
