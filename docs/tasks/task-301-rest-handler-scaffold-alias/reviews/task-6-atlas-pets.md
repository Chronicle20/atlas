# Review: Task 6 — atlas-pets REST handler scaffold aliasing

**Commit range:** f3a14e69c..3ae29d519
**Commits:** 3a88c569f (scaffold conversion), 3ae29d519 (id-parser delegation)
**Verdict: APPROVED**

## Scope

`git diff --stat f3a14e69c..3ae29d519`:
- `services/atlas-pets/atlas.com/pets/pet/resource.go` | 251 changed
- `services/atlas-pets/atlas.com/pets/rest/handler.go` | 115 changed

Matches the brief exactly (module `services/atlas-pets/atlas.com/pets`, 2 files). No other files touched; `main.go` untouched (`git diff f3a14e69c..3ae29d519 -- services/atlas-pets/atlas.com/pets/main.go` empty); `pet/resource_paginate_test.go` diff is empty (untouched, stays green).

## Checks

1. **`rest/handler.go` scaffold alias (commit 1)** — PASS.
   `HandlerDependency`, `HandlerContext`, `GetHandler`, `InputHandler[M]` are now type aliases to `server.*`; `RegisterHandler` is a `var` alias; `RegisterInputHandler` remains a function wrapper (generic, can't be a var) delegating to `server.RegisterInputHandler[M](l)`. Matches the prescribed pattern (`atlas-guilds/rest/handler.go:1-28`). Old `HandlerDependency` struct, `HandlerContext` struct, `ParseInput[M]`, and the four-level-curried `RegisterHandler`/`RegisterInputHandler` implementations are deleted. `context`, `io`, `gorm.io/gorm` imports pruned as instructed.

2. **`pet/resource.go` Shape A conversion** — PASS.
   `grep -rn 'd\.DB()'` across the module (`services/atlas-pets/atlas.com/pets`) returns zero matches — all 4 original `d.DB()` sites converted.
   - `handleGetPet(db *gorm.DB) rest.GetHandler` (resource.go:39)
   - `handleGetPetsForCharacter(db *gorm.DB) rest.GetHandler` (resource.go:59)
   - `handleCreate(db *gorm.DB) rest.InputHandler[RestModel]` (resource.go:153)
   - `handleUpdate(db *gorm.DB) rest.InputHandler[RestModel]` (resource.go:193)
   Each takes `db` as a constructor parameter and closes over it; `InitResource` (resource.go:23-37) drops `(db)` from `rest.RegisterHandler(l)(si)` / `rest.RegisterInputHandler[RestModel](l)(si)` and instead passes `(db)` at each handler's registration call site (`handleGetPetsForCharacter(db)`, `handleCreate(db)` ×2, `handleGetPet(db)`, `handleUpdate(db)`). Exactly Shape A as specified.
   - No Shape C candidates existed in this file (all four exported handlers touch the DB via `NewProcessor(...db)`), consistent with "4 `d.DB()` sites, one resource file" in the brief.

3. **Id-parser delegation (commit 2, separate as required by global constraints)** — PASS.
   `ParseCharacterId` / `ParsePetId` now delegate to `server.ParseIntId[uint32](l, "characterId"/"petId", next)` (handler.go:26-32). Verified `server.ParseIntId[T IntegerId](l, varName, next)` exists at `libs/atlas-rest/server/id_parser.go:16`, matching the call signature. Dead `CharacterIdHandler` / `PetIdHandler` named types and their bespoke `mux.Vars`+`strconv.Atoi` bodies are deleted; `strconv` and `mux` imports pruned. Commit contains only this file, satisfying the "separate commit" constraint.

4. **Operator-surface handlers unaffected** — the local `operatorErrorObject`/`writeNotFound`/`writeForbidden` helpers (resource.go:252-296), added by an earlier task (task-224), are untouched by this diff — correctly out of scope for a scaffold-aliasing task.

5. **Build / test** — ran locally in the worktree:
   - `go build ./...` — exit 0.
   - `go test ./...` — all packages pass, including `atlas-pets/pet` (3.278s, contains `resource_paginate_test.go`) and `atlas-pets` root.

6. **libs/atlas-rest immutability, no new/edited tests** — confirmed: diff touches only the two files listed above; `libs/` untouched; no `_test.go` files in the diff.

## Not evaluable

None — the full review surface (2 changed files, their build, and their tests) was directly inspectable and exercised.

## Findings

None. No blocking or non-blocking findings.
