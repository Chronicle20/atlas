# Review: Task 7 — atlas-quest REST handler scaffold alias

Commit range: `3ae29d519..b96fa96b6`
Commits reviewed:
- `08363da47` — refactor(atlas-quest): alias rest scaffolding, close db over the three input handlers
- `b96fa96b6` — refactor(atlas-quest): delegate id parsers to server.ParseIntId

Brief: `.superpowers/sdd/plan/task-7-brief.md` (plan.md lines 563-664)
Report: `.superpowers/sdd/plan/task-7-report.md`

## Scope

Files touched:
- `services/atlas-quest/atlas.com/quest/rest/handler.go`
- `services/atlas-quest/atlas.com/quest/quest/resource.go`

Matches the brief's declared file list exactly. `quest/resource_paginate_test.go` was
confirmed untouched (`git diff 3ae29d519..b96fa96b6 -- .../resource_paginate_test.go`
produced no output).

## Findings

### 1. `rest/handler.go` scaffolding alias — PASS

Commit `08363da47` rewrites lines 1-97 to the prescribed alias form exactly:
`HandlerDependency`, `HandlerContext`, `GetHandler`, `InputHandler[M]` type-aliased to
`server.*`; `RegisterHandler` bound as `var RegisterHandler = server.RegisterHandler`;
`RegisterInputHandler[M]` delegates to `server.RegisterInputHandler[M](l)`. Local
`ParseInput` correctly omitted per the brief's explicit instruction (verified: no
`ParseInput` reference remains anywhere in the module —
`grep -n 'ParseInput\b' -r services/atlas-quest/atlas.com/quest` returns nothing).
`context`, `io`, `gorm.io/gorm` imports pruned from `rest/handler.go` as required.

Verified `libs/atlas-rest/server` exports match the aliased signatures exactly
(`server/context.go:14,31,43,45`, `server/register.go:11,26`).

### 2. Shape A conversion of the three input handlers — PASS

`git diff -U0 3ae29d519..08363da47 -- .../quest/resource.go` shows exactly 12 hunks, all
confined to: the `InitResource` registration block (3 lines), and the bodies of
`handleStartQuest`, `handleCompleteQuest`, `handleUpdateQuestProgress`. Each was
correctly re-wrapped `func handleX(db *gorm.DB) rest.InputHandler[M] { return func(d, c,
i M) http.HandlerFunc { ... d.DB() replaced by db ... } }`, matching the brief's example
verbatim (`resource.go:153,198,327` pre-conversion → confirmed post-conversion via diff).

`grep -rn 'd\.DB()' services/atlas-quest/atlas.com/quest` returns zero matches — no
`d.DB()` site was missed or double-converted.

### 3. Six already-migrated GET handlers left structurally alone — PASS

Confirmed via the same `-U0` hunk list: no hunk touches `handleGetQuestsByCharacter`,
`handleGetQuestsByCharacterAndState`, `handleGetQuestByCharacterAndId`,
`handleForfeitQuest`, `handleGetQuestProgress`, or `handleDeleteQuestsByCharacter`
bodies. Only their registration call sites in `InitResource` changed, and only by
losing the standalone `(db)` curry now baked into `registerGet := rest.RegisterHandler(l)(si)`
(was `rest.RegisterHandler(l)(db)(si)`). No handler was converted twice, and no
correctly-aliased type was reverted to a local definition — pre-image confirms these
six were already `func handleX(db *gorm.DB) rest.GetHandler` before this task, matching
the brief's claim, and their signatures are byte-identical post-conversion.

### 4. `InitResource` registration lines — PASS

All five call sites updated consistently:
- `registerGet := rest.RegisterHandler(l)(db)(si)` → `rest.RegisterHandler(l)(si)` (db curry moved to per-handler closures)
- `rest.RegisterInputHandler[StartQuestInputRestModel](l)(db)(si)(StartQuest, handleStartQuest)` → `...RegisterInputHandler[...](l)(si)(StartQuest, handleStartQuest(db))`
- same pattern for `CompleteQuest` and `UpdateQuestProgress`.

No registration line was missed; `go build ./...` and `go test ./...` both succeed
(confirmed independently below), which would fail on a curry-arity mismatch.

### 5. `InputHandler`/`ParseInput`/`RegisterInputHandler` scaffolding preserved (not dropped) — PASS

Per the review brief's explicit warning: this service genuinely uses the input-handler
scaffolding (three `POST`/`PATCH` handlers with jsonapi bodies). The implementer kept
`InputHandler[M]` and `RegisterInputHandler[M]` aliased to the shared `server` package
rather than dropping them as dead code — correct, since dropping them here would have
broken `StartQuest`, `CompleteQuest`, and `UpdateQuestProgress`.

### 6. Commit split (Global Constraint: conversion / id-parser-delegation in separate commits) — PASS

`08363da47` contains only the scaffolding alias + Shape A conversion
(`resource.go`, `rest/handler.go`, 128 insertions / 224 deletions across those two files
combined per the full range diffstat). `b96fa96b6` touches only `rest/handler.go` (44
lines, 6 insertions / 38 deletions) and contains only the `CharacterIdHandler` /
`QuestStatusIdHandler` / `QuestIdHandler` → `server.ParseIntId[uint32]` delegation, plus
pruning `strconv` and `github.com/gorilla/mux` imports from `rest/handler.go`. Confirmed
`resource.go` has zero diff between `08363da47` and `b96fa96b6`.

`grep -n 'CharacterIdHandler\|QuestStatusIdHandler\|QuestIdHandler'` across the module
returns nothing — the three named handler types were fully deleted, not left as dead
code.

### 7. `libs/atlas-rest/` untouched, no `main.go` changes, no test edits — PASS

`git diff 3ae29d519..b96fa96b6 --name-only` returns only the two `atlas-quest` files
listed above. No `main.go`, no `libs/`, no `_test.go` file in the diff.

### 8. Build / test / format — PASS

```
cd services/atlas-quest/atlas.com/quest && gofmt -l . && go build ./... && go test ./...
```
`gofmt -l .` — no output (clean). `go build ./...` — exit 0. `go test ./...` — all
packages `ok` (quest, quest/progress, kafka/consumer/*, root), no failures, no skips
beyond packages with no test files. `resource_paginate_test.go` runs and passes
unedited, as part of the `atlas-quest/quest` package's green result.

## Not evaluable

None. The full diff surface (two files, two commits) was read and traced against the
brief; the shared `libs/atlas-rest/server` contract referenced by the aliases was
confirmed by reading its exported signatures directly.

## Verdict rationale

No requirement from the brief was dropped. No handler was converted twice or
regressed to a local definition. The six pre-migrated GET handlers were correctly left
structurally alone with only their registration curry updated. The input-handler
scaffolding — the one service-specific risk flagged for this task — was correctly
preserved via alias rather than dropped. The commit split respects the global
constraint. Build, tests, and formatting are all clean, and the existing test file was
not touched.
