# Review: Task 14 — atlas-party-quests REST handler scaffold alias + id-parser delegation

Commit range: `06ae056e4..c63ded980` (4 commits)

- `6c1013780` refactor(atlas-party-quests): alias rest scaffolding, close db over handler constructors
- `6e9b4a6a6` refactor(atlas-party-quests): delegate bare-lookup id parsers to shared server parsers
- `33c227398` fix(atlas-account): correct stale doc comment referencing deleted AccountIdHandler type
- `c63ded980` refactor(atlas-party-quests): delegate ParseFieldInstance to server.ParseUUIDId (fix round)

## Verdict

APPROVED

## Checklist verification

### 1. `d.DB()` sites converted
`grep -rn 'd\.DB()' services/atlas-party-quests/atlas.com/party-quests` returns zero hits. All 12 sites (7 in `definition/resource.go`, 5 in `instance/resource.go`) converted, none double-converted. Confirmed by direct read of both files post-conversion.

### 2. Shape A applied consistently
Verified by reading both files in full:
- `definition/resource.go`: 7 handlers (`GetAllDefinitionsHandler`, `GetDefinitionHandler`, `GetDefinitionByQuestIdHandler`, `CreateDefinitionHandler`, `UpdateDefinitionHandler`, `DeleteDefinitionHandler`, `ValidateDefinitionsHandler`) are all `func Xxx(db *gorm.DB) rest.GetHandler` / `rest.InputHandler[RestModel]` closing over `db`.
- `instance/resource.go`: 5 handlers (`GetAllInstancesHandler`, `GetInstanceHandler`, `GetInstanceByCharacterHandler`, `GetInstanceByFieldHandler`, `GetTimerByCharacterHandler`) same shape.
- Both `InitResource` functions drop `(db)` from `rest.RegisterHandler(l)(si)` / `rest.RegisterInputHandler[RestModel](l)(si)` and pass `(db)` at each `router.HandleFunc(...)` call site (`definition/resource.go:23-31`, `instance/resource.go:21-29`).

### 3. `rest/handler.go` alias form
Matches the `2d6ba1ce6` (atlas-account) precedent exactly: `HandlerDependency`/`HandlerContext`/`GetHandler`/`InputHandler[M]` type aliases to `server.*`, `var RegisterHandler = server.RegisterHandler`, thin `RegisterInputHandler[M]` wrapper. `context`, `io`, `gorm.io/gorm` imports pruned in commit 1. After the fix round, `gorilla/mux` is also correctly dropped (no remaining `mux.Vars` call in the file — all six `Parse*` helpers now delegate). Final import block: `net/http`, `github.com/google/uuid`, `github.com/jtumidanski/api2go/jsonapi`, `github.com/sirupsen/logrus`, `github.com/Chronicle20/atlas/libs/atlas-rest/server` — all genuinely used.

### 4. Commit-split integrity (scaffold vs parser commit)
`git diff --stat 6c1013780..6e9b4a6a6 -- definition/resource.go instance/resource.go` is empty. Confirmed independently — the resource-file conversion commit and the parser-delegation commit do not overlap.

### 5. Commit-per-service integrity
- `6c1013780`: touches only `services/atlas-party-quests/...` (`definition/resource.go`, `instance/resource.go`, `rest/handler.go`).
- `6e9b4a6a6`: touches only `services/atlas-party-quests/atlas.com/party-quests/rest/handler.go`.
- `33c227398`: touches only `services/atlas-account/atlas.com/account/rest/handler.go`, 1 line, comment-only (`-// AccountIdHandler, for the character-slots sub-resource` → `+// ParseAccountId, for the character-slots sub-resource`).
- `c63ded980`: touches only `services/atlas-party-quests/atlas.com/party-quests/rest/handler.go`.

No cross-service or cross-commit bleed.

### 6. `Parse*` helper final states
Diffed pre-image (`06ae056e4:.../rest/handler.go`) against final state. All six delegations are behaviorally faithful:

| Helper | Original body | Final delegation | Path var match |
|---|---|---|---|
| `ParseDefinitionId` | bare `uuid.Parse(mux.Vars(r)["definitionId"])` | `server.ParseUUIDId(l, "definitionId", next)` | yes |
| `ParseInstanceId` | bare `uuid.Parse(mux.Vars(r)["instanceId"])` | `server.ParseUUIDId(l, "instanceId", next)` | yes |
| `ParseQuestId` | `mux.Vars` + `== ""` check | `server.ParseStringId(l, "questId", next)` | yes (settled precedent: emptiness→presence narrowing) |
| `ParseCharacterId` | bare `strconv.Atoi(mux.Vars(r)["characterId"])` | `server.ParseIntId[uint32](l, "characterId", next)` | yes |
| `ParseFieldInstance` | bare `uuid.Parse(mux.Vars(r)["fieldInstance"])` | `server.ParseUUIDId(l, "fieldInstance", next)` (delegated in fix-round commit `c63ded980`) | yes |
| `ParseMapId` | `mux.Vars` + `fmt.Sscanf(mapIdStr, "%d", &mapId)` | `server.ParseIntId[uint32](l, "mapId", next)` | yes (settled precedent: Sscanf→Atoi narrowing) |

All six named `*Handler` types (`DefinitionIdHandler`, `InstanceIdHandler`, `QuestIdHandler`, `CharacterIdHandler`, `FieldInstanceHandler`, `MapIdHandler`) were deleted along with their corresponding delegated helper. `grep -rn` for each name across `services/atlas-party-quests` and `services/atlas-account` returns zero hits (the one `QuestIdHandler` substring match is the unrelated, still-live `GetDefinitionByQuestIdHandler`) — no dangling reference in code or comments. This is exactly the Task-13-class defect the checklist calls out, and it is clean here.

**Ruling 1 verification (`ParseFieldInstance`)**: confirmed behaviorally faithful — `server.ParseUUIDId(l, "fieldInstance", next)` reproduces the original bare `uuid.Parse(mux.Vars(r)["fieldInstance"])` exactly, same error path (400 + log on parse failure). `instance/resource.go`'s `GetInstanceByFieldHandler` still calls `rest.ParseFieldInstance(d.Logger(), func(fieldInstance uuid.UUID) http.HandlerFunc {...})` unchanged — the call site's `next` signature (`func(uuid.UUID) http.HandlerFunc`) is unaffected by the type-alias-to-bare-func-literal change, confirmed by `go build ./...` succeeding. `type FieldInstanceHandler` deletion left nothing dangling.

**Ruling 2 verification (`ParseMapId` "unreferenced" claim)**: `grep -rn "ParseMapId\|MapIdHandler" services/atlas-party-quests` returns exactly one hit — the `func ParseMapId` declaration itself in `rest/handler.go:47`. No caller anywhere in the module. Confirmed independently: the report's "dead code, currently unreferenced" claim is accurate.

### 7. Untouched files
`git diff --stat 06ae056e4..c63ded980` for `definition/resource_test.go`, `instance/resource_paginate_test.go`, `main.go`, `libs/atlas-rest/` is empty across the whole range. Confirmed.

### 8. Build / test / gofmt
Ran independently, not trusted from the report:
```
cd services/atlas-party-quests/atlas.com/party-quests && go build ./... && gofmt -l . && go test ./...
```
→ build OK, `gofmt -l .` produced no output, all packages `ok` (definition, instance, condition, guild, party, reward, tenant all pass; no-test-file packages listed `?` as expected).
```
cd services/atlas-account/atlas.com/account && go build ./... && go test ./...
```
→ build OK, all packages `ok`. atlas-account still builds and passes after the doc-comment fix in `33c227398`.

## Controller rulings — verified, not re-opened

- `ParseFieldInstance` delegation (ruling 1): code matches the ruling, behaviorally faithful, type cleanly deleted. See §6 above.
- `ParseMapId` narrowing + "unreferenced" claim (ruling 2): claim independently confirmed via grep. See §6 above.
- `ParseStringId` presence-vs-emptiness narrowing (settled precedent): applies identically to `ParseQuestId` here; not re-litigated.

## Findings

None blocking. None non-blocking.

## Not evaluable

None — the full review surface (four commits, both resource files, the shared `rest/handler.go`, and the atlas-account cross-service comment fix) was read in full and independently verified by build/test/grep, not merely trusted from the implementer report.

---

```text
verdict: APPROVED
artifact: docs/tasks/task-301-rest-handler-scaffold-alias/reviews/task-14-atlas-party-quests.md
scope_confirmed: reviewed all 4 commits in 06ae056e4..c63ded980 (atlas-party-quests scaffold alias, id-parser delegation incl. fix-round ParseFieldInstance, and the atlas-account cross-service doc-comment fix) — matches the stated task scope exactly, no drift
blocking: 0
non_blocking: 0
not_evaluable: 0
```
