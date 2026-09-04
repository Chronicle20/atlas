# Review: Task 13 — atlas-account REST handler scaffold alias + id-parser delegation

Commit range: `d1b421ba3..06ae056e4` (2d6ba1ce6 conversion, 06ae056e4 id-parser delegation)
Brief: `.superpowers/sdd/plan/task-13-brief.md`

## Scope

Reviewed the two-commit diff touching:
- `services/atlas-account/atlas.com/account/rest/handler.go`
- `services/atlas-account/atlas.com/account/account/resource.go`

Cross-checked against `libs/atlas-rest/server/id_parser.go` (contract `ParseIntId` delegates to) and precedent commit `cc25f4579` (atlas-reactor-actions, Task 12).

## Findings

### 1. `d.DB()` conversion — PASS

`grep -rn 'd\.DB()' services/atlas-account/atlas.com/account` returns zero matches. All 9 original sites converted; no double-conversion found (build/tests confirm no leftover `.DB()` misuse).

### 2. Shape A applied consistently — PASS

`account/resource.go` — every exported/local handler constructor that needs `db` now takes `db *gorm.DB` and returns `rest.GetHandler` or `rest.InputHandler[M]`, closing over `db` (`resource.go:47,99,118,148,167,191,216,250,276`). `handleCreateAccount` (`resource.go:79`) and `handleDeleteAccountSession` (`resource.go:241`) correctly remain Shape C (no `db` dependency — they only touch a Kafka producer), matching the brief's "Shape A on each enclosing handler … Shape C elsewhere."

`InitResource` (`resource.go:23-45`) drops the `(db)` curry from both `rest.RegisterHandler(l)(si)` and the four `rest.RegisterInputHandler[...](l)(si)` calls, and passes `(db)` at each of the 9 call sites that need it (`resource.go:33-44`).

Confirmed exactly 4 `rest.RegisterInputHandler[...]` call sites: `RestModel`, `CreateRestModel`, `PinAttemptInputRestModel`, `PicAttemptInputRestModel` (`resource.go:26-29`).

### 3. `rest/handler.go` alias form — PASS

Matches the landed precedent (`cc25f4579`, `services/atlas-reactor-actions/atlas.com/reactor/rest/handler.go`) structurally: `HandlerDependency`, `HandlerContext`, `GetHandler`, `InputHandler[M]` type aliases, `RegisterHandler` var alias, `RegisterInputHandler[M]` thin wrapper. `ParseInput` correctly omitted (local dead code, now handled by `server.RegisterInputHandler` internally). `context`, `io`, `gorm.io/gorm` imports correctly pruned from `rest/handler.go`; `strconv` and `mux` retained because `ParseAccountIdAndWorldId` still needs them.

Cross-check: every alias the module imports is genuinely used in `account/resource.go` (`grep` for `rest.HandlerContext|HandlerDependency|GetHandler|InputHandler|RegisterHandler|RegisterInputHandler` outside `rest/handler.go` returns 20 hits, one for each alias). No dead aliases.

### 4. Commit-split integrity — PASS

`git diff --stat 2d6ba1ce6..06ae056e4 -- services/atlas-account/atlas.com/account/account/resource.go` is empty — the second commit touches only `rest/handler.go`, exactly as the brief specifies (Step 5 is a "separate commit").

### 5. `ParseAccountId` delegation and `ParseAccountIdAndWorldId` preservation — PASS

`ParseAccountId` now delegates to `server.ParseIntId[uint32](l, "accountId", next)` (`rest/handler.go:29-31`). Compared against `libs/atlas-rest/server/id_parser.go:17-27`:

```go
func ParseIntId[T IntegerId](l logrus.FieldLogger, varName string, next func(T) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value, err := strconv.Atoi(mux.Vars(r)[varName])
		if err != nil {
			l.WithError(err).Errorf("Error parsing %s as integer", varName)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(T(value))(w, r)
	}
}
```

Behavior is identical to the deleted original (`mux.Vars` lookup → `strconv.Atoi` → 400 on error → `next(uint32(value))(w, r)` on success). Only the log message text differs ("Error parsing accountId as integer" vs. the deleted original's "Error parsing id as uint32") — cosmetic only, not a behavioral divergence, and consistent with the settled precedent for this family of delegations.

`ParseAccountIdAndWorldId` in the post-refactor `rest/handler.go` (lines 38-56) is byte-identical to the pre-refactor version at `d1b421ba3:services/atlas-account/atlas.com/account/rest/handler.go` (lines 127-145, `AccountIdAndWorldIdHandler` type + function body) — diffed by hand, no changes. Correctly left alone per the brief (parses two vars, no shared equivalent).

`AccountIdHandler` type (the old `func(id uint32) http.HandlerFunc` alias used only by `ParseAccountId`'s deleted signature) is confirmed removed and unreferenced as a type: `grep -rn 'AccountIdHandler' services/atlas-account/` returns only one hit, a doc comment on `AccountIdAndWorldIdHandler` (`rest/handler.go:33`) describing it as "the world-scoped counterpart of AccountIdHandler" — non-blocking, see below.

### 6. Untouched files — PASS

- `git diff d1b421ba3..06ae056e4 -- .../account/resource_test.go .../main.go` — empty.
- `git diff d1b421ba3..06ae056e4 -- libs/atlas-rest/` — empty.

### 7. Build / test / gofmt — PASS

```
cd services/atlas-account/atlas.com/account && go build ./...   → exit 0, no output
cd services/atlas-account/atlas.com/account && go test ./...    → ok (account, account/account), no test files elsewhere
gofmt -l services/atlas-account/atlas.com/account                → no output (clean)
```

`account/resource_test.go` passes unedited, as required.

## Non-blocking notes

- `rest/handler.go:33` — the doc comment on `AccountIdAndWorldIdHandler` still says "AccountIdAndWorldIdHandler is the world-scoped counterpart of AccountIdHandler," but the `AccountIdHandler` type was deleted in this same change (Step 5). The comment is now a dangling reference to a removed identifier. Purely cosmetic (doesn't affect compilation or behavior); worth a one-line fix in a follow-up but not blocking this task.

## Not evaluable

None — full review surface (both commits, the one file pair touched, and the `server.ParseIntId` contract it now depends on) was directly inspected and independently verified against build/test/gofmt.

## Verdict

APPROVED_WITH_FINDINGS (one non-blocking cosmetic note; no blocking defects found).
