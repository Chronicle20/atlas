# Review: Task 18 — atlas-npc-conversations REST handler scaffold alias

Range reviewed: `9f88119f8..cc0c55dcd` (3 commits, reviewed as a whole):

- `ee9e712c2` refactor(atlas-npc-conversations): alias rest scaffolding, close db over handler constructors
- `b73dc8b50` docs(atlas-npc-shops): correct stale rest handler package comment after scaffold aliasing
- `cc0c55dcd` refactor(atlas-npc-conversations): delegate id parsers to shared server parsers (fix round)

Brief: `.superpowers/sdd/plan/task-18-brief.md`
Report: `.superpowers/sdd/plan/task-18-report.md` (incl. Fix round section)

## Verdict

APPROVED — no blocking or non-blocking findings.

## Checklist and evidence

### 1. Zero `d.DB()` remaining, all 20 sites converted, none double-converted

`grep -rn 'd\.DB()' services/atlas-npc-conversations/atlas.com/npc` → zero hits.
`grep -n '^func.*Handler(db \*gorm.DB) rest\.' conversation/*/resource.go` → exactly
20 matches (7 npc, 6 quest, 5 item, 2 recipe), matching the brief's count exactly.
PASS.

### 2. Shape A applied consistently

All 20 converted handlers follow `func XxxHandler(db *gorm.DB) rest.GetHandler` /
`rest.InputHandler[RestModel]`, closing over `db`. Each `InitResource` drops
`(db)` from `rest.RegisterHandler(l)(si)` / `rest.RegisterInputHandler[RestModel](l)(si)`
(verified in all four `resource.go` files, e.g.
`conversation/npc/resource.go:26-27`) and passes `(db)` at each call site
(e.g. `conversation/npc/resource.go:30-37`, `conversation/item/resource.go:34-38`,
`conversation/quest/resource.go:26-31`, `conversation/recipe/resource.go:21-22`).
`ValidateConversationHandler` (`conversation/npc/resource.go:247`) correctly left
untouched — it never used `d.DB()`, and its call site at
`conversation/npc/resource.go:36` still passes the bare function, not `(db)`.
Confirmed `server.HandlerDependency` (`libs/atlas-rest/server/context.go:14-17`)
has no `DB()` method, so Shape A is the only correct shape. PASS.

### 3. Commit-per-service integrity

- `ee9e712c2` touches only 5 files under `services/atlas-npc-conversations/`.
- `b73dc8b50` touches exactly one file:
  `services/atlas-npc-shops/atlas.com/npc/rest/handler.go`, 6 insertions / 10
  deletions, and the diff (`git show b73dc8b50`) is comment-only — the `package rest`
  line and all code below are unchanged.
- `cc0c55dcd` touches exactly one file:
  `services/atlas-npc-conversations/atlas.com/npc/rest/handler.go`.

PASS.

### 4. Fix-round split

`git diff --stat ee9e712c2..cc0c55dcd -- conversation/npc/resource.go
conversation/quest/resource.go conversation/item/resource.go
conversation/recipe/resource.go` → empty output. The parser commit touched no
resource file. PASS.

### 5. Parser classification (controller ruling implemented correctly)

Pre-image (`git show 9f88119f8:.../rest/handler.go`) confirms:
- `ParseConversationId` was bare `mux.Vars` + `uuid.Parse`, `ParseNpcId`/`ParseQuestId`
  were bare `fmt.Sscanf("%d", ...)` with no extra checks, `ParseItemId` had the
  extra `|| itemId == 0` guard.

Final state (`services/atlas-npc-conversations/atlas.com/npc/rest/handler.go`):
- `ParseConversationId` (line 30-32) now delegates to
  `server.ParseUUIDId(l, "conversationId", next)`; `ConversationIdHandler` type
  deleted. Matches ruling.
- `ParseNpcId` (line 34-36) delegates to `server.ParseIntId[uint32](l, "npcId", next)`;
  `NpcIdHandler` deleted. Matches ruling.
- `ParseQuestId` (line 38-40) delegates to `server.ParseIntId[uint32](l, "questId", next)`;
  `QuestIdHandler` deleted. Matches ruling.
- `ParseItemId` (line 42-53) is byte-identical to the pre-image body (`fmt.Sscanf`
  + `err != nil || itemId == 0` guard); `ItemIdHandler` type survives at line 41.
  Correctly kept as a genuine keeper — the zero-rejection has no shared-parser
  equivalent.

The ruling itself is factually correct: `server.ParseIntId`
(`libs/atlas-rest/server/id_parser.go:16-26`) uses `strconv.Atoi`, a strictly
stricter numeric parse than `fmt.Sscanf("%d", ...)`'s prefix-accepting behavior —
consistent with the settled task-14 precedent, not the inverse the first
implementer argued. `server.ParseUUIDId`
(`libs/atlas-rest/server/id_parser.go:28-38`) is behaviorally identical to the
pre-image body. PASS.

### 6. Deleted `*Handler` type names — zero remaining references

`grep -rn 'ConversationIdHandler\b\|NpcIdHandler\b\|QuestIdHandler\b'
services/atlas-npc-conversations/` (code and comments, whole module) → zero
hits. PASS.

### 7. Import block fully live

Final `rest/handler.go` imports: `fmt`, `net/http`, `github.com/google/uuid`,
`github.com/gorilla/mux`, `jsonapi`, `logrus`, `server`. `fmt` and `mux` are
used by `ParseItemId`'s body (lines 46-48); `uuid` is used in
`ParseConversationId`'s `next func(conversationId uuid.UUID) http.HandlerFunc`
parameter type (line 30). `net/http`, `jsonapi`, `logrus`, `server` all used
elsewhere in the file. `go build ./...` succeeds, which would fail on an
unused import. PASS.

### 8. `b73dc8b50` docs fix correctness

Old comment described a "DB-parameterized variant of RegisterHandler … using
curried function composition to inject the database connection," which Task 15
already removed. New comment (`services/atlas-npc-shops/atlas.com/npc/rest/handler.go:1-8`)
states the aliases are thin wrappers over `libs/atlas-rest/server`, and that
"Handlers that need database access take a *gorm.DB parameter and close over
it; resource registrars pass db per call site rather than currying it through
the registration functions." This matches the actual code in the same file
(`var RegisterHandler = server.RegisterHandler` at line ~29, no db param;
`ParseNpcId`/`ParseCommodityId` at lines 34-39 delegate to shared parsers). No
contradiction between comment and code. PASS.

### 9. Untouched files

`git diff --name-only 9f88119f8..cc0c55dcd` lists exactly 6 files: the 4
resource.go files, `atlas-npc-conversations/.../rest/handler.go`, and
`atlas-npc-shops/.../rest/handler.go`. No `_test.go`, no `main.go`, nothing
under `libs/atlas-rest/` appears in the diff. The three `resource_test.go`
files (`conversation/npc/`, `conversation/quest/`, `conversation/recipe/`)
are confirmed present and absent from the diff (empty-diff, as required). PASS.

### 10. Build / test / gofmt

From `services/atlas-npc-conversations/atlas.com/npc`:
`go build ./...` → exit 0. `go test ./...` → all packages `ok` or `[no test
files]`, no failures. `gofmt -l .` → no output.

From `services/atlas-npc-shops/atlas.com/npc`:
`go build ./...` → exit 0. `go test ./...` → all packages `ok` or `[no test
files]`, no failures. `gofmt -l .` → no output.

PASS.

## Not evaluable

None — full scope was directly verifiable within the diff and its immediate
dependency (`libs/atlas-rest/server`).

## Notes (non-blocking)

None.
