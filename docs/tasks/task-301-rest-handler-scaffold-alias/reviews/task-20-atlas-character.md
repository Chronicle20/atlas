# Review — Task 20 (atlas-character), task-301

Commit range: `0fd55ce60..06bb4733f` (5ce014ed4 conversion, 06bb4733f id-parser delegation)
Brief: `.superpowers/sdd/plan/task-20-brief.md` (+ `task-20-brief-cont.md`)
Reviewer: task-reviewer (independent verification; implementer report treated as a
claim to check, not evidence, per the provenance note — the conversion commit
5ce014ed4 was authored by an agent that never itself ran final verification).

## Scope confirmed

Reviewed the full diff `0fd55ce60..06bb4733f`: eight files in
`services/atlas-character/atlas.com/character` (`rest/handler.go`,
`character/resource.go`, `character/name_validity_resource.go`,
`equipslot/resource.go`, `pending_change/resource.go`,
`saved_location/resource.go`, `session/history/resource.go`,
`teleport_rock/resource.go`). No files outside this path changed
(`git diff --name-only 0fd55ce60..06bb4733f` — all paths under the module root).
This matches the brief's declared scope exactly. No scope mismatch.

## Findings

### 1. Zero `d.DB()` remaining, no double-conversion — PASS

`grep -rn 'd\.DB()' --include='*.go' services/atlas-character/atlas.com/character`
returns 0 hits. Per-file handler-count spot check confirms all 26 pre-image sites
converted with no duplication:

- `character/resource.go`: 8 handlers now take `db *gorm.DB` as sole/first param
  (`handleGetCharacters`, `handleGetCharactersForAccountInWorld`,
  `handleGetCharactersByName`, `handleGetCharacter`, `handleCreateCharacter`,
  `handleDeleteCharacter`, `handleChangeCharacterWorld`, `handleUpdateCharacter`) —
  matches brief's count of 8.
- `pending_change/resource.go`: 7 handlers converted — matches brief's count of 7.
- `saved_location/resource.go`: 3 handlers converted — matches brief's count of 3.
- `teleport_rock/resource.go`: 3 sites (Shape B, see below) — matches.
- `equipslot/resource.go`: 2 handlers converted — matches.
- `session/history/resource.go`: 2 handlers converted — matches.
- `character/name_validity_resource.go`: 1 site (Shape B) — matches.

All `NewProcessor(...)` call sites inside these handlers reference the closed-over
`db` variable, not a stray `d.DB()` (verified by grep across all six Shape-A/C
files at `pending_change/resource.go:53,90,121,146,193,247,299`,
`saved_location/resource.go:46,84,111`, `session/history/resource.go:59,101`,
`equipslot/resource.go:35,65`).

### 2. The three shapes — PASS

- **Shape A/C curry drop** (`character/resource.go`, `equipslot`, `pending_change`,
  `saved_location`, `session/history`): confirmed at `character/resource.go:50`
  (`func handleGetCharacters(db *gorm.DB) rest.GetHandler`) and equivalently across
  all other handlers in these five files. Register expressions in each
  `InitResource` now pass `(db)` directly, e.g.
  `character/resource.go:37` `registerGet("get_characters", handleGetCharacters(db))`.

- **Shape B — `name_validity_resource.go:28`**:
  `func handleGetNameValidity(db *gorm.DB, nameReservedOf NameReservedFunc) rest.GetHandler`
  — `db` added as a genuine first parameter to the existing constructor, body's
  single `d.DB()` replaced with `db` at the `NewProcessor(...)` call. No second
  wrapper/closure layer was introduced (function still returns
  `rest.GetHandler` directly, one nesting level, same as pre-image). PASS.

- **Shape B — `teleport_rock/resource.go:67,97`**:
  `handleAddTeleportRockMap(db *gorm.DB, worldIdOf WorldIdOf)` and
  `handleRemoveTeleportRockMap(db *gorm.DB, worldIdOf WorldIdOf)` — `db` correctly
  added as first parameter alongside the existing `worldIdOf` parameter, no extra
  closure layer. PASS. (`handleGetTeleportRockMaps(db)` at line 44 is the plain
  Shape-A case in the same file — also correct.)

- **Third curry level preserved (`character/resource.go` `InitResource`)** — PASS.
  Pre-image (`0fd55ce60`): `func InitResource(si ...) func(db *gorm.DB) func(nameReservedOf NameReservedFunc) server.RouteInitializer`.
  Post-image: identical signature, byte-for-byte
  (`character/resource.go:27-28`). Only the *inner* register expressions changed
  from `rest.RegisterHandler(l)(db)(si)(...)` to `rest.RegisterHandler(l)(si)(...)`
  (the `HandlerDependency` no longer needs `db` curried through
  `RegisterHandler`/`RegisterInputHandler` — that's the whole point of the alias
  scaffold), and `handleGetNameValidity(nameReservedOf)` became
  `handleGetNameValidity(db, nameReservedOf)` at line 41 — correctly passing `db`
  first per the Shape-B contract while retaining `nameReservedOf` as the second
  argument. The `nameReservedOf` curry level itself was never touched.

### 3. `rest/handler.go` matches Task 19 alias precedent — PASS

Final `rest/handler.go` (28 lines) declares the five aliases
(`HandlerDependency`, `HandlerContext`, `GetHandler`, `InputHandler[M]`,
`RegisterHandler`) plus a generic `RegisterInputHandler[M]` wrapper, structurally
identical in shape to the Task 19 (atlas-mts) precedent. Import block is fully
live: `net/http`, `jsonapi`, `logrus`, `server` — no unused `mux`/`strconv` (those
were pruned in commit 2 along with the id-parser bodies that used them). No
leftover local `ParseInput` — confirmed absent from both the diff and a
whole-module grep (`grep -rn ParseInput` across the module returns 0 hits per the
implementer's report and independently re-confirmed no local declaration exists
in the final file).

### 4. Commit-split integrity — PASS

`git diff --stat 5ce014ed4..06bb4733f -- <all seven resource files>` is empty
(independently re-run, confirmed empty). `git diff --stat 5ce014ed4..06bb4733f`
overall touches only `rest/handler.go` (4 insertions, 26 deletions). Commit 1
(`5ce014ed4`) is exactly the scaffold conversion; commit 2 (`06bb4733f`) is
exactly the id-parser delegation. No leakage either direction.

### 5. The two id-parser helpers — PASS, delegation correct

Pre-image (`git show 0fd55ce60:.../rest/handler.go`) bodies for both parsers are
bare `mux.Vars(r)[varName]` → `strconv.Atoi` → cast → `w.WriteHeader(400)` on
error, with **no** additional semantic check (no zero-check, no enum/allowed-set
validation) in either:

```go
func ParseCharacterId(l logrus.FieldLogger, next CharacterIdHandler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        characterId, err := strconv.Atoi(mux.Vars(r)["characterId"])
        if err != nil { ...; w.WriteHeader(http.StatusBadRequest); return }
        next(uint32(characterId))(w, r)
    }
}
func ParseInventoryType(l logrus.FieldLogger, next InventoryTypeHandler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        inventoryType, err := strconv.Atoi(mux.Vars(r)["inventoryType"])
        if err != nil { ...; w.WriteHeader(http.StatusBadRequest); return }
        next(int8(inventoryType))(w, r)
    }
}
```

`ParseInventoryType` (the one flagged for scrutiny) validates nothing beyond
"parses as an integer" — no range check against a known set of inventory-type
enum values. Both are genuine bare lookups; the delegation to
`server.ParseIntId[uint32]("characterId", ...)` /
`server.ParseIntId[int8]("inventoryType", ...)` is behavior-preserving modulo the
pre-approved `strconv.Atoi`-via-`int` → direct-`T`-parse narrowing established
across Tasks 11-19. Generic type argument matches the original target type in
both cases (`uint32` for characterId, `int8` — not `uint32` — for inventoryType,
correctly noted). `libs/atlas-rest/server/id_parser.go:12-14`'s `IntegerId`
constraint (`~uint32 | ~int32 | ~int8 | ~uint8 | ~uint16`) includes `~int8`, so
the delegation compiles and type-checks.

### 6. `CharacterIdHandler`/`InventoryTypeHandler` deletion — PASS

`grep -rn 'CharacterIdHandler\|InventoryTypeHandler' --include='*.go' services/atlas-character/atlas.com/character`
returns 0 hits — independently re-run, zero references in code or comments
anywhere in the module.

### 7. Untouched files — PASS

- `main.go`: `git diff --stat 0fd55ce60..06bb4733f -- .../main.go` — empty.
  `InitResource` call sites at `main.go:128-139` still curry `(GetServer())(db)(...)`
  at the outer level, consistent with the preserved outer curry shape on every
  `InitResource` function (only the *inner* register-expression curry was
  dropped, never the `InitResource` signature itself).
- `libs/atlas-rest/`: empty diff over the whole range.
- All `*_test.go`: empty diff over the whole range (`git diff --stat 0fd55ce60..06bb4733f -- '**/*_test.go'`).
  The six existing resource tests (`character/resource_test.go`,
  `character/name_validity_resource_test.go`, `equipslot/resource_test.go`,
  `pending_change/resource_test.go`, `teleport_rock/resource_test.go`,
  `session/history/resource_test.go`) match the brief's list exactly.
  `saved_location` correctly has no `resource_test.go` (confirmed via `find`) — it
  was never in the brief's test list, and `go test` reports
  `atlas-character/saved_location [no test files]`, which is expected, not a gap.

### 8. Build / test / gofmt — PASS (independently run, not taken from the report)

- `go build ./...` from `services/atlas-character/atlas.com/character` — exit 0,
  no output.
- `gofmt -l .` — exit 0, no files listed.
- `go test ./...` (foreground, backgrounded automatically past the 120s
  no-output threshold, awaited to completion) — all packages `ok`, including
  `atlas-character/character` (10.378s), `atlas-character/pending_change`
  (226.433s), `atlas-character/equipslot`, `atlas-character/session/history`,
  `atlas-character/teleport_rock`, `atlas-character/kafka/consumer/character`
  (22.044s). No failures, no skips beyond `[no test files]` packages that never
  had tests.

## Not evaluable

None. All checklist items in the review brief were directly verifiable from the
diff plus one build/test/gofmt run against the module, within the stated scope.

## Verdict rationale

Independent re-verification (not leaning on the implementer's report, per the
provenance note) confirms: all 26 `d.DB()` sites converted with no double
conversion, all three shapes (A/C curry-drop, B first-param, and the
`nameReservedOf` third-curry-level preservation) applied exactly as specified,
the commit split is clean, both id-parser helpers were correctly classified as
bare lookups and delegated, the deleted named types have zero remaining
references, and the module builds, gofmts clean, and passes its full test suite
including the four-minute `pending_change` package. No blocking or non-blocking
findings.

---

verdict: APPROVED
artifact: docs/tasks/task-301-rest-handler-scaffold-alias/reviews/task-20-atlas-character.md
scope_confirmed: full diff 0fd55ce60..06bb4733f (5ce014ed4 conversion + 06bb4733f id-parser delegation), eight files under services/atlas-character/atlas.com/character; independently verified against pre-images at 0fd55ce60, not the implementer report
blocking: 0
non_blocking: 0
not_evaluable: 0
