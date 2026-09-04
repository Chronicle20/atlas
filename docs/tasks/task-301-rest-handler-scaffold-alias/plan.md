# REST Handler Scaffold Consolidation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce the 21 hand-rolled `services/*/rest/handler.go` files to type aliases over `libs/atlas-rest/server`, moving `*gorm.DB` off `HandlerDependency` and into the handler-constructor closure.

**Architecture:** `type HandlerDependency = server.HandlerDependency` is a *type alias*, so every existing `*rest.HandlerDependency` reference keeps compiling unchanged. Exactly two things break, and both break at compile time: `d.DB()` (the method does not exist on the shared type) and the `(db)` curry level in `rest.RegisterHandler(l)(db)(si)`. The Go compiler is therefore the exhaustive checker for this sweep — `go build ./...` in each service module either finds every site or the conversion is complete. Each service is its own Go module, so each conversion is an independently building, independently testable commit.

**Tech Stack:** Go 1.27, `github.com/Chronicle20/atlas/libs/atlas-rest/server`, `github.com/jtumidanski/api2go/jsonapi`, `gorm.io/gorm`, `github.com/gorilla/mux`, `github.com/sirupsen/logrus`.

**Spec:** `docs/tasks/task-301-rest-handler-scaffold-alias/design.md` (PRD: `docs/tasks/task-301-rest-handler-scaffold-alias/prd.md`)

## Global Constraints

Every task's requirements implicitly include this section.

- **No file under `libs/atlas-rest/` may change.** The library is complete; this task consumes it.
- **No `main.go` may change.** `InitResource` keeps its existing `func(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer` signature; the `db` it already receives is what gets passed to handler constructors.
- **No processor, provider, entity, or Kafka change.** Handler signatures change; processor signatures do not.
- **Omit the `ParseInput` wrapper from every converted file.** Zero services call `rest.ParseInput` (`grep -rn 'rest\.ParseInput' --include='*.go' services` → 0 matches). Once `RegisterInputHandler` delegates to `server.RegisterInputHandler`, the local wrapper is dead in all 21. This deliberately deviates from PRD FR-1.1's literal template; design §9.1 records it.
- **`server.RegisterHandler` / `server.RegisterInputHandler` are the targets — never the `Simple*` variants.** The `Simple*` variants omit `ParseTenant`, and all 21 parse the tenant.
- **Do not introduce `RegisterOptionalInputHandler` or `ParseOptionalInput`.** Zero references in the 21.
- **Do not rename or unexport handler functions.** Six services expose handlers as exported identifiers (`GetAllScriptsHandler`, `CreateNoteHandler`, …). Change their signature in place; renaming is a separate cleanup.
- **Never edit an existing test to make it pass.** The resource tests are the regression net. None of them sets an `ENVIRONMENT` header, so per design §6.1 row 1 they exercise the unchanged legacy path. **If a resource test fails, that is a finding to report, not a test to edit** — a failure means the conversion changed the header-absent path, which it must not.
- **No new tests are required.** `libs/atlas-rest/server/handler_test.go` and `context_test.go` already own the FR-4 behavior (`TestParseEnvironmentPutsTheHeaderOnTheContext`, `TestParseEnvironmentWithNoHeaderIsTheLegacyPath`, `TestParseEnvironmentRejectsAnUnknownEnvironment`, `TestParseTenantRejectsAMismatchedEnvironment`, `TestParseInput_MalformedJSONIs400JSONAPIShape`), and that library is not touched.
- **Two intended behavior changes**, which must appear in the PR description (design §6):
  - **FR-4.1** — `server.RegisterHandler` chains `RetrieveSpan → ParseEnvironment → ParseTenant`; the hand-rolled chain omitted `ParseEnvironment`. A request with **no** `ENVIRONMENT` header is byte-identical (`ParseEnvironment` short-circuits on `id == ""`). A request whose header names an unknown / DEACTIVATING / DELETED environment now 400s instead of being silently served from baseline data; a header disagreeing with the tenant's projected environment now 400s at `ParseTenant`. Note: `env.Reconcile` already runs in all 21 today — they already use `server.ParseTenant`. What changes is that a *non-empty* header env can now reach it.
  - **FR-4.2** — Malformed-body 400s gain a JSON:API errors document. The hand-rolled `ParseInput` wrote a bare `w.WriteHeader(http.StatusBadRequest)`; `server.ParseInput` calls `writeBadRequestJSONAPI`. Status is unchanged at 400.
- **The tracing span name and the `logrus.Fields{"originator": handlerName, "type": "rest_handler"}` log fields must stay identical.** Verified line-by-line identical between the two chains (design §6.3); no action needed, but do not perturb them.
- **Commit per service.** Each service is its own Go module. Where a task also does FR-1.3 parser delegation, that is a **second, separate commit** so it can be dropped by `git revert` without touching the conversion.
- **Task order matters only at the ends:** Tasks 1–2 are pilots that validate the recipe; Task 21 (docs) must be last so its "RESOLVED" claim is true when written.

---

## The three resource-file transform shapes

Every conversion task applies these. They are restated in each task because a task brief is read in isolation.

**Shape A — bare handler that uses `d.DB()`.** The common case. Wrap the constructor:

```go
// before
func handleGetMountForCharacter(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	// ... NewProcessor(d.Logger(), d.Context(), d.DB()) ...
}

// after
func handleGetMountForCharacter(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		// ... NewProcessor(d.Logger(), d.Context(), db) ...
	}
}
```

Reference implementation: `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`.

**Shape B — constructor already parameterized (already returns `rest.GetHandler`).** Add `db` as a *first parameter*; do **not** add a second wrapper layer:

```go
// before: func handleGetNameValidity(nameReservedOf NameReservedFunc) rest.GetHandler
// after:  func handleGetNameValidity(db *gorm.DB, nameReservedOf NameReservedFunc) rest.GetHandler
```

**Shape C — handler never touches the DB.** Leave the function completely alone. Only the registration line loses `(db)`.

**Selection rule:** add the `db` parameter if and only if `d.DB()` appears in that function's body or in a closure it owns. Do not curry a handler that does not need it.

**Registration lines** in `InitResource` drop exactly one call level:

```go
registerGet := rest.RegisterHandler(l)(db)(si)          →  rest.RegisterHandler(l)(si)
rest.RegisterInputHandler[M](l)(db)(si)(name, handler)  →  rest.RegisterInputHandler[M](l)(si)(name, handler)
```

…and the handler argument gains its `db`: `register("get_mount", handleGetMountForCharacter)` → `register("get_mount", handleGetMountForCharacter(db))` for Shape A/B handlers.

**Local-variable collision:** if a converted handler body already declares a local `db` (e.g. `db := d.DB().WithContext(d.Context())`), rename the local to `tx` rather than shadow the closed-over `db`. One known site — `services/atlas-mts/atlas.com/mts/testsupport/resource.go:274` — but check per file.

---

## Task 1: atlas-messages (pilot — pure alias swap)

Smallest possible case: no `*gorm.DB` anywhere in the service, one route, no `InputHandler`, and its `RegisterHandler` is already `(l)(si)` with no `db` curry. This proves the alias mechanics with zero transform risk.

### Files

- `services/atlas-messages/atlas.com/messages/rest/handler.go` — replace all 47 lines with the alias form
- `services/atlas-messages/atlas.com/messages/chat/resource.go` — read-only; verify it still compiles unchanged (it uses `rest.RegisterHandler(l)(si)` at line 76 and `*rest.HandlerDependency` at line 82, both alias-compatible)

Module root (`go build`/`go test` cwd): `services/atlas-messages/atlas.com/messages`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28` (the alias block)

- [ ] **Step 1: Replace `rest/handler.go` in full**

The file has no path-parameter helpers, so it becomes exactly this:

```go
package rest

import (
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

var RegisterHandler = server.RegisterHandler
```

No `InputHandler` / `RegisterInputHandler` block: `grep -c 'rest\.RegisterInputHandler' -r services/atlas-messages` is 0 and must stay 0 (FR-1.2).

- [ ] **Step 2: Build and test**

```bash
cd services/atlas-messages/atlas.com/messages && go build ./... && go test ./...
```

Expected: both exit 0. `chat/resource.go` needs no edit — its `rest.RegisterHandler(l)(si)` already matches the shared arity.

- [ ] **Step 3: Confirm the acceptance greps for this service**

```bash
grep -q 'type HandlerDependency struct' services/atlas-messages/atlas.com/messages/rest/handler.go || echo "no local struct — correct"
grep -q 'gorm.io/gorm' services/atlas-messages/atlas.com/messages/rest/handler.go || echo "no gorm import — correct"
grep -rq 'd\.DB()' services/atlas-messages/atlas.com/messages/ || echo "no d.DB() — correct"
```

- [ ] **Step 4: Commit**

```bash
git add services/atlas-messages/atlas.com/messages/rest/handler.go
git commit -m "refactor(atlas-messages): alias rest scaffolding to atlas-rest/server"
```

---

## Task 2: atlas-mounts (pilot — Shape A end to end)

One `d.DB()`, one route, no `InputHandler`. Proves the Shape-A transform and the curry drop.

### Files

- `services/atlas-mounts/atlas.com/mounts/rest/handler.go` — replace lines 1–58 (the scaffolding) with the alias form; keep `ParseCharacterId` (lines 59–71) unchanged in this commit
- `services/atlas-mounts/atlas.com/mounts/mount/resource.go` — Shape A on `handleGetMountForCharacter` (line 26, `d.DB()` at line 29); curry drop at line 19

Module root: `services/atlas-mounts/atlas.com/mounts`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28` (alias block), `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45` (Shape A)

- [ ] **Step 1: Rewrite `rest/handler.go`**

Replace everything above the `CharacterIdHandler` type declaration with the alias block, keeping the existing `ParseCharacterId` body verbatim:

```go
package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

var RegisterHandler = server.RegisterHandler

type CharacterIdHandler func(characterId uint32) http.HandlerFunc

// ... existing ParseCharacterId body, unchanged ...
```

Prune imports to exactly what remains referenced. `context`, `jsonapi`, and `gorm.io/gorm` all become unused here.

- [ ] **Step 2: Convert `mount/resource.go`**

Shape A on `handleGetMountForCharacter`, and drop the `(db)` curry in `InitResource`:

```go
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/characters/{characterId}/mount").Subrouter()
			r.HandleFunc("", registerGet("get_mount_for_character", handleGetMountForCharacter(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetMountForCharacter(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := NewProcessor(d.Logger(), d.Context(), db)
				// ... rest of the existing body unchanged ...
			}
		})
	}
}
```

`InitResource` keeps its signature. `gorm.io/gorm` stays imported in `resource.go` (it is now used by the handler constructor too).

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-mounts/atlas.com/mounts && go build ./... && go test ./...
```

Expected: both exit 0. The service has no resource test; `go build` is the gate.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-mounts/atlas.com/mounts/rest/handler.go services/atlas-mounts/atlas.com/mounts/mount/resource.go
git commit -m "refactor(atlas-mounts): alias rest scaffolding, close db over handler constructor"
```

- [ ] **Step 5: Delegate `ParseCharacterId` (FR-1.3) — separate commit**

The body is a bare `mux.Vars` + `strconv.Atoi` on `characterId`, and `rest.CharacterIdHandler` has zero references outside this file (`grep -rn 'rest\.CharacterIdHandler' services/atlas-mounts` → 0). Delete the named type and delegate:

```go
func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}
```

Call sites pass func literals, which satisfy the unnamed parameter type unchanged. Prune `strconv` and `github.com/gorilla/mux` from the imports.

- [ ] **Step 6: Build, test, commit the delegation**

```bash
cd services/atlas-mounts/atlas.com/mounts && go build ./... && go test ./...
git add services/atlas-mounts/atlas.com/mounts/rest/handler.go
git commit -m "refactor(atlas-mounts): delegate ParseCharacterId to server.ParseIntId"
```

---

## Task 3: Delete the two dead handler packages (FR-3)

`atlas-fame` and `atlas-events` each carry a `rest/handler.go` with zero consumers. `atlas-events` already registers via `server.RegisterHandler(l)(si)` directly. Deleting is correct; converting them would be landing dead code.

### Files

- `services/atlas-fame/atlas.com/fame/rest/handler.go` — delete (the `rest/` directory contains only this file, so the directory goes with it)
- `services/atlas-events/atlas.com/events/rest/handler.go` — delete (same)
- `services/atlas-events/atlas.com/events/event/definition/resource.go` — read-only; confirms `atlas-events` already uses `server.RegisterHandler(l)(si)` at line 32
- `services/atlas-mts/atlas.com/mts/testsupport/resource.go` — read-only; unrelated, listed only because it is the other `InitResource` shape

Module roots: `services/atlas-fame/atlas.com/fame`, `services/atlas-events/atlas.com/events`

- [ ] **Step 1: Re-verify zero consumers (FR-3.3 requires a fresh grep, not the design's word)**

```bash
grep -rn '"atlas-fame/rest"' --include='*.go' services/atlas-fame || echo "atlas-fame: no consumers — correct"
grep -rn '"atlas-events/rest"' --include='*.go' services/atlas-events || echo "atlas-events: no consumers — correct"
ls services/atlas-fame/atlas.com/fame/rest services/atlas-events/atlas.com/events/rest
```

Expected: both greps print the "no consumers" line, and each `rest/` directory lists exactly `handler.go`. **If either grep returns a hit, STOP and report** — that service must be converted per the Task-1 recipe instead of deleted, and the plan needs amending.

- [ ] **Step 2: Delete both packages**

```bash
git rm -r services/atlas-fame/atlas.com/fame/rest services/atlas-events/atlas.com/events/rest
```

- [ ] **Step 3: Build and test both modules**

```bash
cd services/atlas-fame/atlas.com/fame && go build ./... && go test ./...
cd services/atlas-events/atlas.com/events && go build ./... && go test ./...
```

Expected: both exit 0. `atlas-events` has two resource tests (`event/definition/resource_test.go`, `event/occurrence/resource_test.go`) that must stay green.

- [ ] **Step 4: Commit**

```bash
git commit -m "refactor(atlas-fame,atlas-events): delete unreferenced rest handler packages"
```

---

## Task 4: atlas-rankings

3 `d.DB()` sites, one resource file, no `InputHandler`.

### Files

- `services/atlas-rankings/atlas.com/rankings/rest/handler.go` — replace lines 1–97 with the alias form; keep `ParseCharacterId` (lines 98–110)
- `services/atlas-rankings/atlas.com/rankings/ranking/resource.go` — Shape A / C per handler; curry drop in `InitResource`
- `services/atlas-rankings/atlas.com/rankings/ranking/resource_test.go` — read-only; must stay green unedited

Module root: `services/atlas-rankings/atlas.com/rankings`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`, `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

var RegisterHandler = server.RegisterHandler
```

…followed by the existing `CharacterIdHandler` type and `ParseCharacterId` body verbatim. **Omit `InputHandler` and `RegisterInputHandler`** — `grep -c 'rest\.RegisterInputHandler' -r services/atlas-rankings` is 0, and the file's current `InputHandler`/`ParseInput` declarations are already-dead scaffolding this task removes (FR-1.2). Prune `context`, `io`, `jsonapi`, and `gorm.io/gorm` from the imports; keep whatever `ParseCharacterId` still needs (`net/http`, `strconv`, `mux`, `logrus`).

- [ ] **Step 2: Convert `ranking/resource.go`**

For each handler in the file: `grep -n 'd\.DB()' services/atlas-rankings/atlas.com/rankings/ranking/resource.go` locates the 3 sites. Apply Shape A to each enclosing handler (wrap in `func handleX(db *gorm.DB) rest.GetHandler { return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc { … } }`, replacing `d.DB()` with `db`); leave DB-free handlers alone (Shape C). In `InitResource`, drop the `(db)` level from every `rest.RegisterHandler(l)(db)(si)` and pass `(db)` to each Shape-A handler at its registration site.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-rankings/atlas.com/rankings && go build ./... && go test ./...
```

Expected: both exit 0, `ranking/resource_test.go` passing unedited.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-rankings/atlas.com/rankings
git commit -m "refactor(atlas-rankings): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 5: Delegate the id parser (FR-1.3) — separate commit**

`ParseCharacterId` is a bare `mux.Vars` + `strconv.Atoi` on `characterId`; `rest.CharacterIdHandler` has zero external references. Replace with:

```go
func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}
```

Delete the `CharacterIdHandler` type; prune `strconv` and `mux`.

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-rankings/atlas.com/rankings && go build ./... && go test ./...
git add services/atlas-rankings/atlas.com/rankings/rest/handler.go
git commit -m "refactor(atlas-rankings): delegate ParseCharacterId to server.ParseIntId"
```

---

## Task 5: atlas-drop-information

4 `d.DB()` sites across three resource files, no `InputHandler`.

### Files

- `services/atlas-drop-information/atlas.com/dis/rest/handler.go` — replace lines 1–97 with the alias form; keep `ParseMonsterId` and `ParseItemId` (lines 98–124)
- `services/atlas-drop-information/atlas.com/dis/reactor/resource.go` — Shape A / C; curry drop
- `services/atlas-drop-information/atlas.com/dis/monster/drop/resource.go` — Shape A / C; curry drop
- `services/atlas-drop-information/atlas.com/dis/continent/resource.go` — Shape A / C; curry drop
- `services/atlas-drop-information/atlas.com/dis/continent/resource_test.go` — read-only; must stay green
- `services/atlas-drop-information/atlas.com/dis/monster/drop/resource_test.go` — read-only; must stay green

Module root: `services/atlas-drop-information/atlas.com/dis`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`, `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

var RegisterHandler = server.RegisterHandler
```

…then the existing `MonsterIdHandler` / `ParseMonsterId` / `ItemIdHandler` / `ParseItemId` declarations verbatim. **Omit `InputHandler`, `ParseInput`, and `RegisterInputHandler`** — `rest.RegisterInputHandler` has 0 call sites in this service; the current declarations are dead scaffolding (FR-1.2, design §4.1).

- [ ] **Step 2: Convert the three resource files**

In each: `grep -n 'd\.DB()' <file>` to locate the sites, apply Shape A to each enclosing handler, Shape C to the rest, and drop the `(db)` curry level in each `InitResource`, passing `(db)` to Shape-A handlers at their registration sites.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-drop-information/atlas.com/dis && go build ./... && go test ./...
```

Expected: both exit 0, both resource tests passing unedited.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-drop-information/atlas.com/dis
git commit -m "refactor(atlas-drop-information): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 5: Delegate the id parsers (FR-1.3) — separate commit**

Both bodies are bare `mux.Vars` + `strconv.Atoi`; neither named type has an external reference. Replace both and delete `MonsterIdHandler` / `ItemIdHandler`:

```go
func ParseMonsterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "monsterId", next)
}

func ParseItemId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "itemId", next)
}
```

Prune `strconv` and `mux`.

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-drop-information/atlas.com/dis && go build ./... && go test ./...
git add services/atlas-drop-information/atlas.com/dis/rest/handler.go
git commit -m "refactor(atlas-drop-information): delegate id parsers to server.ParseIntId"
```

---

## Task 6: atlas-pets

4 `d.DB()` sites, one resource file, 3 `rest.RegisterInputHandler` call sites.

### Files

- `services/atlas-pets/atlas.com/pets/rest/handler.go` — replace lines 1–97 with the alias form; keep `ParseCharacterId` / `ParsePetId` (lines 98–125)
- `services/atlas-pets/atlas.com/pets/pet/resource.go` — Shape A / C; curry drop on both `RegisterHandler` and `RegisterInputHandler`
- `services/atlas-pets/atlas.com/pets/pet/resource_paginate_test.go` — read-only; must stay green

Module root: `services/atlas-pets/atlas.com/pets`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`, `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then the existing `CharacterIdHandler` / `ParseCharacterId` / `PetIdHandler` / `ParsePetId` declarations verbatim. `RegisterInputHandler` must stay a function, not a `var` — Go has no generic variables. Omit the local `ParseInput` wrapper. Prune `context`, `io`, and `gorm.io/gorm`.

- [ ] **Step 2: Convert `pet/resource.go`**

`grep -n 'd\.DB()' services/atlas-pets/atlas.com/pets/pet/resource.go` locates the 4 sites. Shape A on each enclosing handler; Shape C elsewhere. Input handlers use the same shape but return `rest.InputHandler[M]`:

```go
func handleCreatePet(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		// ... d.DB() → db ...
	}
}
```

In `InitResource`, drop `(db)` from both `rest.RegisterHandler(l)(db)(si)` and `rest.RegisterInputHandler[M](l)(db)(si)`, and pass `(db)` to each converted handler at its registration site.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-pets/atlas.com/pets && go build ./... && go test ./...
```

Expected: both exit 0, `pet/resource_paginate_test.go` passing unedited.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-pets/atlas.com/pets
git commit -m "refactor(atlas-pets): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 5: Delegate the id parsers (FR-1.3) — separate commit**

Both bodies are bare `mux.Vars` + `strconv.Atoi`; neither named type has an external reference. Replace and delete `CharacterIdHandler` / `PetIdHandler`:

```go
func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}

func ParsePetId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "petId", next)
}
```

Prune `strconv` and `mux`.

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-pets/atlas.com/pets && go build ./... && go test ./...
git add services/atlas-pets/atlas.com/pets/rest/handler.go
git commit -m "refactor(atlas-pets): delegate id parsers to server.ParseIntId"
```

---

## Task 7: atlas-quest (partially migrated already)

Only 3 `d.DB()` sites, all in *input* handlers. Six of its GET handlers are **already** in the target `func handleX(db *gorm.DB) rest.GetHandler` shape (`quest/resource.go:57,87,117,234,258,344`) — do not re-wrap them; they only need their registration line's `(db)` curry dropped. `handleGetQuestsByCharacterAndState` at line 87 is Shape B and already db-parameterized.

### Files

- `services/atlas-quest/atlas.com/quest/rest/handler.go` — replace lines 1–97 with the alias form; keep `ParseCharacterId` / `ParseQuestStatusId` / `ParseQuestId` (lines 98–138)
- `services/atlas-quest/atlas.com/quest/quest/resource.go` — Shape A on the three input handlers (`handleStartQuest:153`, `handleCompleteQuest:198`, `handleUpdateQuestProgress:327`); curry drop everywhere; leave the six already-migrated GET handlers structurally alone
- `services/atlas-quest/atlas.com/quest/quest/resource_paginate_test.go` — read-only; must stay green

Module root: `services/atlas-quest/atlas.com/quest`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`; the in-file already-correct shape at `services/atlas-quest/atlas.com/quest/quest/resource.go:57`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then the three existing helper type + `Parse*` declarations verbatim. Omit the local `ParseInput`. Prune `context`, `io`, `gorm.io/gorm`.

- [ ] **Step 2: Convert `quest/resource.go`**

Apply Shape A to `handleStartQuest`, `handleCompleteQuest`, and `handleUpdateQuestProgress` only — they are the only three with `d.DB()`. Their input-handler form:

```go
func handleStartQuest(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		// ... d.DB() → db ...
	}
}
```

Then drop the `(db)` curry from every registration line in `InitResource`. The six already-migrated GET handlers keep their bodies; only their registration expression changes from `rest.RegisterHandler(l)(db)(si)("name", handleX(db))` to `rest.RegisterHandler(l)(si)("name", handleX(db))`.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-quest/atlas.com/quest && go build ./... && go test ./...
```

Expected: both exit 0, `quest/resource_paginate_test.go` passing unedited.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-quest/atlas.com/quest
git commit -m "refactor(atlas-quest): alias rest scaffolding, close db over the three input handlers"
```

- [ ] **Step 5: Delegate the id parsers (FR-1.3) — separate commit**

All three bodies are bare `mux.Vars` + `strconv.Atoi`; none of the named types has an external reference. Replace and delete `CharacterIdHandler` / `QuestStatusIdHandler` / `QuestIdHandler`:

```go
func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}

func ParseQuestStatusId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "questStatusId", next)
}

func ParseQuestId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "questId", next)
}
```

Prune `strconv` and `mux`.

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-quest/atlas.com/quest && go build ./... && go test ./...
git add services/atlas-quest/atlas.com/quest/rest/handler.go
git commit -m "refactor(atlas-quest): delegate id parsers to server.ParseIntId"
```

---

## Task 8: atlas-parcel

6 `d.DB()` sites, one resource file, 2 `rest.RegisterInputHandler` call sites. Its two `Parse*` helpers already use unnamed `next func(...)` parameter types, so the FR-1.3 step only replaces bodies.

### Files

- `services/atlas-parcel/atlas.com/parcel/rest/handler.go` — replace lines 1–98 with the alias form; keep `ParseCharacterId` / `ParseParcelId` (lines 99–128)
- `services/atlas-parcel/atlas.com/parcel/parcel/resource.go` — Shape A / C; curry drop
- `services/atlas-parcel/atlas.com/parcel/parcel/resource_test.go` — read-only; must stay green

Module root: `services/atlas-parcel/atlas.com/parcel`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`, `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then `ParseCharacterId` and `ParseParcelId` verbatim. Omit the local `ParseInput`. Prune `context`, `io`, `gorm.io/gorm`.

- [ ] **Step 2: Convert `parcel/resource.go`**

`grep -n 'd\.DB()' services/atlas-parcel/atlas.com/parcel/parcel/resource.go` locates the 6 sites. Shape A on each enclosing handler (GET handlers return `rest.GetHandler`, input handlers return `rest.InputHandler[M]`); Shape C elsewhere. Drop the `(db)` curry from both register expressions in `InitResource` and pass `(db)` to each converted handler.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-parcel/atlas.com/parcel && go build ./... && go test ./...
```

Expected: both exit 0, `parcel/resource_test.go` passing unedited.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-parcel/atlas.com/parcel
git commit -m "refactor(atlas-parcel): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 5: Delegate the id parsers (FR-1.3) — separate commit**

`ParseCharacterId` reads `mux.Vars(r)["characterId"]` then `strconv.ParseUint(...,10,32)`; `ParseParcelId` is a bare `mux.Vars(r)["parcelId"]` string lookup with an `ok`/empty check. Both are bare lookups and both already take unnamed func parameter types, so only the bodies change:

```go
func ParseCharacterId(l logrus.FieldLogger, next func(characterId uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}

func ParseParcelId(l logrus.FieldLogger, next func(parcelId string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "parcelId", next)
}
```

Note the one behavior narrowing: `server.ParseStringId` 400s only when the var is absent, not when it is present-but-empty. Gorilla mux does not match an empty path segment for `{parcelId}`, so the empty case is unreachable through the router. Prune `strconv` and `mux`.

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-parcel/atlas.com/parcel && go build ./... && go test ./...
git add services/atlas-parcel/atlas.com/parcel/rest/handler.go
git commit -m "refactor(atlas-parcel): delegate id parsers to shared server parsers"
```

---

## Task 9: atlas-notes

7 `d.DB()` sites, one resource file, 1 `rest.RegisterInputHandler` call site. This service exports its handlers (`CreateNoteHandler`, …) — change signatures in place, do not rename.

### Files

- `services/atlas-notes/atlas.com/notes/rest/handler.go` — replace lines 1–97 with the alias form; keep `ParseCharacterId` / `ParseNoteId` (lines 98–124)
- `services/atlas-notes/atlas.com/notes/note/resource.go` — Shape A / C; curry drop

Module root: `services/atlas-notes/atlas.com/notes`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`, `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then `CharacterIdHandler` / `ParseCharacterId` / `NoteIdHandler` / `ParseNoteId` verbatim. Omit the local `ParseInput`. Prune `context`, `io`, `gorm.io/gorm`.

- [ ] **Step 2: Convert `note/resource.go`**

`grep -n 'd\.DB()' services/atlas-notes/atlas.com/notes/note/resource.go` locates the 7 sites. Shape A on each enclosing handler — including the exported ones, whose signature becomes e.g. `func CreateNoteHandler(db *gorm.DB) rest.InputHandler[RestModel]`. Verified: no test and no other package references any of the exported handler identifiers, so changing their signature in place is safe. Drop the `(db)` curry in `InitResource` and pass `(db)` at each registration site.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-notes/atlas.com/notes && go build ./... && go test ./...
```

Expected: both exit 0.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-notes/atlas.com/notes
git commit -m "refactor(atlas-notes): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 5: Delegate the id parsers (FR-1.3) — separate commit**

Both bodies are bare `mux.Vars` + `strconv.Atoi`; neither named type has an external reference. Replace and delete `CharacterIdHandler` / `NoteIdHandler`:

```go
func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}

func ParseNoteId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "noteId", next)
}
```

Prune `strconv` and `mux`.

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-notes/atlas.com/notes && go build ./... && go test ./...
git add services/atlas-notes/atlas.com/notes/rest/handler.go
git commit -m "refactor(atlas-notes): delegate id parsers to server.ParseIntId"
```

---

## Task 10: atlas-map-actions

6 `d.DB()` sites in `script/resource.go`, 1 `rest.RegisterInputHandler` call site. This service exports its handlers (`GetAllScriptsHandler`, …). Its `ParseScriptName` is **not** a bare int/UUID lookup and keeps its body.

`atlas-map-actions`, `atlas-portal-actions`, and `atlas-reactor-actions` are near-identical copies of each other (same six exported `*ScriptHandler` functions at the same line numbers). Convert this one carefully; Tasks 11 and 12 apply the same edit rather than re-deriving it.

### Files

- `services/atlas-map-actions/atlas.com/map-actions/rest/handler.go` — replace lines 1–97 with the alias form; keep `ParseScriptId` / `ParseScriptName` (lines 98–125)
- `services/atlas-map-actions/atlas.com/map-actions/script/resource.go` — Shape A / C; curry drop
- `services/atlas-map-actions/atlas.com/map-actions/script/resource_test.go` — read-only; must stay green

Module root: `services/atlas-map-actions/atlas.com/map-actions`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`, `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then `ScriptIdHandler` / `ParseScriptId` / `ScriptNameHandler` / `ParseScriptName` verbatim. Omit the local `ParseInput`. Prune `context`, `io`, `gorm.io/gorm`; keep whatever the two helpers still reference.

- [ ] **Step 2: Convert `script/resource.go`**

`grep -n 'd\.DB()' services/atlas-map-actions/atlas.com/map-actions/script/resource.go` locates the 6 sites. Shape A on each enclosing handler, including the exported `*Handler` identifiers (signature changes in place; no rename). Drop the `(db)` curry in `InitResource`; pass `(db)` at each registration site.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./...
```

Expected: both exit 0, `script/resource_test.go` passing unedited.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-map-actions/atlas.com/map-actions
git commit -m "refactor(atlas-map-actions): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 5: Delegate `ParseScriptId` only (FR-1.3) — separate commit**

`ParseScriptId` is a bare `mux.Vars` + `uuid.Parse` on `scriptId`; `rest.ScriptIdHandler` has no external reference. Delegate it and delete the type:

```go
func ParseScriptId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "scriptId", next)
}
```

**Leave `ParseScriptName` exactly as it is** — design §4.3 identifies it as a non-bare-lookup helper. Prune imports only if `mux`/`uuid` become unreferenced (they will not: `ParseScriptName` still uses `mux`).

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./...
git add services/atlas-map-actions/atlas.com/map-actions/rest/handler.go
git commit -m "refactor(atlas-map-actions): delegate ParseScriptId to server.ParseUUIDId"
```

---

## Task 11: atlas-portal-actions

6 `d.DB()` sites in `script/resource.go`, 1 `rest.RegisterInputHandler` call site. Structurally a copy of `atlas-map-actions` (Task 10) — apply the same edit; the only differences are the second helper (`ParsePortalId`, a string) and the module path.

### Files

- `services/atlas-portal-actions/atlas.com/portal/rest/handler.go` — replace lines 1–97 with the alias form; keep `ParseScriptId` / `ParsePortalId` (lines 98–125)
- `services/atlas-portal-actions/atlas.com/portal/script/resource.go` — Shape A / C; curry drop
- `services/atlas-portal-actions/atlas.com/portal/script/resource_pagination_test.go` — read-only; must stay green

Module root: `services/atlas-portal-actions/atlas.com/portal`

Patterns to copy: `services/atlas-map-actions/atlas.com/map-actions/script/resource.go` **after Task 10 lands** (same six handlers, same line numbers); `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then `ScriptIdHandler` / `ParseScriptId` / `PortalIdHandler` / `ParsePortalId` verbatim. Omit the local `ParseInput`. Prune `context`, `io`, `gorm.io/gorm`.

- [ ] **Step 2: Convert `script/resource.go`**

`grep -n 'd\.DB()' services/atlas-portal-actions/atlas.com/portal/script/resource.go` locates the 6 sites. Shape A on each enclosing handler, including the exported `*Handler` identifiers. Drop the `(db)` curry in `InitResource`; pass `(db)` at each registration site.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-portal-actions/atlas.com/portal && go build ./... && go test ./...
```

Expected: both exit 0, `script/resource_pagination_test.go` passing unedited.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-portal-actions/atlas.com/portal
git commit -m "refactor(atlas-portal-actions): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 5: Delegate both id parsers (FR-1.3) — separate commit**

`ParseScriptId` is a bare `mux.Vars` + `uuid.Parse`; `ParsePortalId` is a bare `mux.Vars` string lookup. Neither named type has an external reference. Replace and delete `ScriptIdHandler` / `PortalIdHandler`:

```go
func ParseScriptId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "scriptId", next)
}

func ParsePortalId(l logrus.FieldLogger, next func(string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "portalId", next)
}
```

**Before replacing `ParsePortalId`, read its body and confirm it is a bare lookup** with no extra validation; if it does anything more (a numeric coercion, an emptiness rule that the router can actually reach), leave it alone and note that in the task report. Prune `mux` if it becomes unreferenced.

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-portal-actions/atlas.com/portal && go build ./... && go test ./...
git add services/atlas-portal-actions/atlas.com/portal/rest/handler.go
git commit -m "refactor(atlas-portal-actions): delegate id parsers to shared server parsers"
```

---

## Task 12: atlas-reactor-actions

6 `d.DB()` sites in `script/resource.go`, 1 `rest.RegisterInputHandler` call site. Structurally a copy of `atlas-map-actions` (Task 10) and `atlas-portal-actions` (Task 11).

### Files

- `services/atlas-reactor-actions/atlas.com/reactor/rest/handler.go` — replace lines 1–97 with the alias form; keep `ParseScriptId` / `ParseReactorId` (lines 98–125)
- `services/atlas-reactor-actions/atlas.com/reactor/script/resource.go` — Shape A / C; curry drop
- `services/atlas-reactor-actions/atlas.com/reactor/script/resource_pagination_test.go` — read-only; must stay green

Module root: `services/atlas-reactor-actions/atlas.com/reactor`

Patterns to copy: `services/atlas-portal-actions/atlas.com/portal/script/resource.go` **after Task 11 lands**; `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then `ScriptIdHandler` / `ParseScriptId` / `ReactorIdHandler` / `ParseReactorId` verbatim. Omit the local `ParseInput`. Prune `context`, `io`, `gorm.io/gorm`.

- [ ] **Step 2: Convert `script/resource.go`**

`grep -n 'd\.DB()' services/atlas-reactor-actions/atlas.com/reactor/script/resource.go` locates the 6 sites. Shape A on each enclosing handler, including the exported `*Handler` identifiers. Drop the `(db)` curry in `InitResource`; pass `(db)` at each registration site.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-reactor-actions/atlas.com/reactor && go build ./... && go test ./...
```

Expected: both exit 0, `script/resource_pagination_test.go` passing unedited.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-reactor-actions/atlas.com/reactor
git commit -m "refactor(atlas-reactor-actions): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 5: Delegate both id parsers (FR-1.3) — separate commit**

`ParseScriptId` is a bare `mux.Vars` + `uuid.Parse`; `ParseReactorId` is a bare `mux.Vars` string lookup. Neither named type has an external reference. Replace and delete `ScriptIdHandler` / `ReactorIdHandler`:

```go
func ParseScriptId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "scriptId", next)
}

func ParseReactorId(l logrus.FieldLogger, next func(string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "reactorId", next)
}
```

**Read `ParseReactorId`'s body first and confirm it is a bare lookup**; if it does anything more, leave it alone and say so in the task report. Prune `mux` if it becomes unreferenced.

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-reactor-actions/atlas.com/reactor && go build ./... && go test ./...
git add services/atlas-reactor-actions/atlas.com/reactor/rest/handler.go
git commit -m "refactor(atlas-reactor-actions): delegate id parsers to shared server parsers"
```

---

## Task 13: atlas-account

9 `d.DB()` sites, one resource file, 4 `rest.RegisterInputHandler` call sites. `ParseAccountIdAndWorldId` parses **two** path vars and has no shared equivalent — it keeps its body.

### Files

- `services/atlas-account/atlas.com/account/rest/handler.go` — replace lines 1–97 with the alias form; keep `ParseAccountId` / `ParseAccountIdAndWorldId` (lines 98–136)
- `services/atlas-account/atlas.com/account/account/resource.go` — Shape A / C; curry drop
- `services/atlas-account/atlas.com/account/account/resource_test.go` — read-only; must stay green

Module root: `services/atlas-account/atlas.com/account`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`, `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then `AccountIdHandler` / `ParseAccountId` / `AccountIdAndWorldIdHandler` / `ParseAccountIdAndWorldId` verbatim. Omit the local `ParseInput`. Prune `context`, `io`, `gorm.io/gorm`.

- [ ] **Step 2: Convert `account/resource.go`**

`grep -n 'd\.DB()' services/atlas-account/atlas.com/account/account/resource.go` locates the 9 sites. Shape A on each enclosing handler (GET → `rest.GetHandler`, input → `rest.InputHandler[M]`); Shape C elsewhere. Drop the `(db)` curry from both register expressions in `InitResource` and pass `(db)` to each converted handler.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-account/atlas.com/account && go build ./... && go test ./...
```

Expected: both exit 0, `account/resource_test.go` passing unedited.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-account/atlas.com/account
git commit -m "refactor(atlas-account): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 5: Delegate `ParseAccountId` only (FR-1.3) — separate commit**

`ParseAccountId` is a bare `mux.Vars` + `strconv.Atoi` on `accountId`; `rest.AccountIdHandler` has no external reference. Delegate and delete the type:

```go
func ParseAccountId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "accountId", next)
}
```

**Leave `ParseAccountIdAndWorldId` and its `AccountIdAndWorldIdHandler` type exactly as they are** — it parses two vars (`accountId`, `worldId`) into `func(uint32, byte)`, and `server.ParseIntId` handles one var into one value. Design §4.3 names it as a keeper. `strconv` and `mux` stay imported for it.

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-account/atlas.com/account && go build ./... && go test ./...
git add services/atlas-account/atlas.com/account/rest/handler.go
git commit -m "refactor(atlas-account): delegate ParseAccountId to server.ParseIntId"
```

---

## Task 14: atlas-party-quests

12 `d.DB()` sites across two resource files, 1 `rest.RegisterInputHandler` call site, 6 `Parse*` helpers. `ParseFieldInstance` is named in design §4.3 as a non-bare-lookup keeper — verify its body before touching it.

### Files

- `services/atlas-party-quests/atlas.com/party-quests/rest/handler.go` — replace lines 1–99 with the alias form; keep the six `Parse*` helpers (lines 100–185)
- `services/atlas-party-quests/atlas.com/party-quests/definition/resource.go` — Shape A / C; curry drop
- `services/atlas-party-quests/atlas.com/party-quests/instance/resource.go` — Shape A / C; curry drop
- `services/atlas-party-quests/atlas.com/party-quests/definition/resource_test.go` — read-only; must stay green
- `services/atlas-party-quests/atlas.com/party-quests/instance/resource_paginate_test.go` — read-only; must stay green

Module root: `services/atlas-party-quests/atlas.com/party-quests`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`, `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then the six existing helper type + `Parse*` declaration pairs verbatim (`DefinitionIdHandler`/`ParseDefinitionId`, `InstanceIdHandler`/`ParseInstanceId`, `QuestIdHandler`/`ParseQuestId`, `CharacterIdHandler`/`ParseCharacterId`, `FieldInstanceHandler`/`ParseFieldInstance`, `MapIdHandler`/`ParseMapId`). Omit the local `ParseInput`. Prune `context`, `io`, `gorm.io/gorm`.

- [ ] **Step 2: Convert both resource files**

In each: `grep -n 'd\.DB()' <file>` locates the sites (12 across the two). Shape A on each enclosing handler; Shape C elsewhere. Drop the `(db)` curry in each `InitResource` and pass `(db)` at each registration site. **Watch for a local `db :=` inside a converted body** — rename it to `tx` rather than shadow the closed-over `db`.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-party-quests/atlas.com/party-quests && go build ./... && go test ./...
```

Expected: both exit 0, both resource tests passing unedited.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-party-quests/atlas.com/party-quests
git commit -m "refactor(atlas-party-quests): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 5: Delegate the bare-lookup id parsers (FR-1.3) — separate commit**

Read each helper body first; delegate only the ones that are a bare `mux.Vars` lookup plus one conversion, and delete that helper's named type:

```go
func ParseDefinitionId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "definitionId", next)
}

func ParseInstanceId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "instanceId", next)
}

func ParseQuestId(l logrus.FieldLogger, next func(string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "questId", next)
}

func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}

func ParseMapId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "mapId", next)
}
```

**Leave `ParseFieldInstance` and its `FieldInstanceHandler` type alone** — design §4.3 identifies it as a non-bare-lookup keeper. If reading a body shows extra validation beyond the bare lookup (an emptiness rule the router can reach, a zero-value rejection, a `fmt.Sscanf`), skip that helper too and record which ones you skipped and why in the task report — a silent behavior narrowing violates FR-4.3.

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-party-quests/atlas.com/party-quests && go build ./... && go test ./...
git add services/atlas-party-quests/atlas.com/party-quests/rest/handler.go
git commit -m "refactor(atlas-party-quests): delegate bare-lookup id parsers to shared server parsers"
```

---

## Task 15: atlas-npc-shops

12 `d.DB()` sites across two resource files, 4 `rest.RegisterInputHandler` call sites.

### Files

- `services/atlas-npc-shops/atlas.com/npc/rest/handler.go` — replace lines 1–110 with the alias form; keep `ParseNpcId` / `ParseCommodityId` (lines 111–139)
- `services/atlas-npc-shops/atlas.com/npc/commodities/resource.go` — Shape A / C; curry drop
- `services/atlas-npc-shops/atlas.com/npc/shops/resource.go` — Shape A / C; curry drop
- `services/atlas-npc-shops/atlas.com/npc/commodities/resource_test.go` — read-only; must stay green
- `services/atlas-npc-shops/atlas.com/npc/shops/resource_paginate_test.go` — read-only; must stay green

Module root: `services/atlas-npc-shops/atlas.com/npc`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`, `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then `NpcIdHandler` / `ParseNpcId` / `CommodityIdHandler` / `ParseCommodityId` verbatim. Omit the local `ParseInput`. Prune `context`, `io`, `gorm.io/gorm`. Note this file's scaffolding block runs longer than its peers (`ParseInput` starts at line 60, the first path helper at line 111) — check what else lives above line 111 before deleting, and preserve anything that is not scaffolding.

- [ ] **Step 2: Convert both resource files**

In each: `grep -n 'd\.DB()' <file>` locates the sites (12 across the two). Shape A on each enclosing handler; Shape C elsewhere. Drop the `(db)` curry from both register expressions in each `InitResource` and pass `(db)` at each registration site.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-npc-shops/atlas.com/npc && go build ./... && go test ./...
```

Expected: both exit 0, both resource tests passing unedited.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-npc-shops/atlas.com/npc
git commit -m "refactor(atlas-npc-shops): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 5: Delegate both id parsers (FR-1.3) — separate commit**

`ParseNpcId` is a bare `mux.Vars` + `strconv.Atoi`; `ParseCommodityId` is a bare `mux.Vars` + `uuid.Parse`. Neither named type has an external reference. Replace and delete `NpcIdHandler` / `CommodityIdHandler`:

```go
func ParseNpcId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "npcId", next)
}

func ParseCommodityId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "commodityId", next)
}
```

Prune `strconv`, `mux`, and `uuid` if they become unreferenced.

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-npc-shops/atlas.com/npc && go build ./... && go test ./...
git add services/atlas-npc-shops/atlas.com/npc/rest/handler.go
git commit -m "refactor(atlas-npc-shops): delegate id parsers to shared server parsers"
```

---

## Task 16: atlas-ban

15 `d.DB()` sites across three resource files, 2 `rest.RegisterInputHandler` call sites.

### Files

- `services/atlas-ban/atlas.com/ban/rest/handler.go` — replace lines 1–98 with the alias form; keep `ParseBanId` / `ParseAccountId` / `ParseReportId` (lines 99–142)
- `services/atlas-ban/atlas.com/ban/ban/resource.go` — Shape A / C; curry drop
- `services/atlas-ban/atlas.com/ban/report/resource.go` — Shape A / C; curry drop
- `services/atlas-ban/atlas.com/ban/history/resource.go` — Shape A / C; curry drop
- `services/atlas-ban/atlas.com/ban/ban/resource_test.go` — read-only; must stay green
- `services/atlas-ban/atlas.com/ban/report/resource_test.go` — read-only; must stay green
- `services/atlas-ban/atlas.com/ban/history/resource_test.go` — read-only; must stay green

Module root: `services/atlas-ban/atlas.com/ban`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`, `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then `BanIdHandler` / `ParseBanId` / `AccountIdHandler` / `ParseAccountId` / `ReportIdHandler` / `ParseReportId` verbatim. Omit the local `ParseInput`. Prune `context`, `io`, `gorm.io/gorm`.

- [ ] **Step 2: Convert the three resource files**

In each: `grep -n 'd\.DB()' <file>` locates the sites (15 across the three). Shape A on each enclosing handler; Shape C elsewhere. Drop the `(db)` curry in each `InitResource` and pass `(db)` at each registration site.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-ban/atlas.com/ban && go build ./... && go test ./...
```

Expected: both exit 0, all three resource tests passing unedited.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-ban/atlas.com/ban
git commit -m "refactor(atlas-ban): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 5: Delegate the id parsers (FR-1.3) — separate commit**

`ParseBanId` and `ParseAccountId` are bare `mux.Vars` + `strconv.Atoi`; `ParseReportId` is a bare `mux.Vars` + `uuid.Parse`. None of the named types has an external reference. Replace all three and delete `BanIdHandler` / `AccountIdHandler` / `ReportIdHandler`:

```go
func ParseBanId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "banId", next)
}

func ParseAccountId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "accountId", next)
}

func ParseReportId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "reportId", next)
}
```

Prune `strconv`, `mux`, and `uuid`.

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-ban/atlas.com/ban && go build ./... && go test ./...
git add services/atlas-ban/atlas.com/ban/rest/handler.go
git commit -m "refactor(atlas-ban): delegate id parsers to shared server parsers"
```

---

## Task 17: atlas-reward-pools

16 `d.DB()` sites across four resource files, 3 `rest.RegisterInputHandler` call sites. Its two `Parse*` helpers already take unnamed `next func(...)` parameter types.

### Files

- `services/atlas-reward-pools/atlas.com/reward-pools/rest/handler.go` — replace lines 1–97 with the alias form; keep `ParseGachaponId` / `ParseItemId` (lines 98–126)
- `services/atlas-reward-pools/atlas.com/reward-pools/gachapon/resource.go` — Shape A / C; curry drop
- `services/atlas-reward-pools/atlas.com/reward-pools/global/resource.go` — Shape A / C; curry drop
- `services/atlas-reward-pools/atlas.com/reward-pools/reward/resource.go` — Shape A / C; curry drop
- `services/atlas-reward-pools/atlas.com/reward-pools/item/resource.go` — Shape A / C; curry drop
- Read-only, must stay green unedited: `resource_test.go` in each of `gachapon/`, `global/`, `reward/`, `item/` under `services/atlas-reward-pools/atlas.com/reward-pools/`

Module root: `services/atlas-reward-pools/atlas.com/reward-pools`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`, `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then `ParseGachaponId` and `ParseItemId` verbatim (this file declares no named `*Handler` helper types). Omit the local `ParseInput`. Prune `context`, `io`, `gorm.io/gorm`.

- [ ] **Step 2: Convert the four resource files**

In each: `grep -n 'd\.DB()' <file>` locates the sites (16 across the four). Shape A on each enclosing handler; Shape C elsewhere. Drop the `(db)` curry from both register expressions in each `InitResource` and pass `(db)` at each registration site.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-reward-pools/atlas.com/reward-pools && go build ./... && go test ./...
```

Expected: both exit 0, all four resource tests passing unedited.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-reward-pools/atlas.com/reward-pools
git commit -m "refactor(atlas-reward-pools): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 5: Delegate both id parsers (FR-1.3) — separate commit**

`ParseGachaponId` is a bare `mux.Vars(r)["gachaponId"]` string lookup with an `ok`/empty check; `ParseItemId` is `mux.Vars(r)["itemId"]` + `strconv.Atoi`. Both already take unnamed func parameter types, so only the bodies change:

```go
func ParseGachaponId(l logrus.FieldLogger, next func(gachaponId string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "gachaponId", next)
}

func ParseItemId(l logrus.FieldLogger, next func(itemId uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "itemId", next)
}
```

The `gachaponId == ""` case is unreachable through the router (gorilla mux does not match an empty path segment for `{gachaponId}`), so `server.ParseStringId`'s absence-only check is equivalent in practice. Prune `strconv` and `mux`.

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-reward-pools/atlas.com/reward-pools && go build ./... && go test ./...
git add services/atlas-reward-pools/atlas.com/reward-pools/rest/handler.go
git commit -m "refactor(atlas-reward-pools): delegate id parsers to shared server parsers"
```

---

## Task 18: atlas-npc-conversations

20 `d.DB()` sites across four resource files, 3 `rest.RegisterInputHandler` call sites. **Its four `Parse*` helpers are all non-delegatable** — see Step 5.

### Files

- `services/atlas-npc-conversations/atlas.com/npc/rest/handler.go` — replace lines 1–98 with the alias form; keep all four `Parse*` helpers (lines 99–160) unchanged
- `services/atlas-npc-conversations/atlas.com/npc/conversation/npc/resource.go` — Shape A / C; curry drop
- `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/resource.go` — Shape A / C; curry drop
- `services/atlas-npc-conversations/atlas.com/npc/conversation/item/resource.go` — Shape A / C; curry drop
- `services/atlas-npc-conversations/atlas.com/npc/conversation/recipe/resource.go` — Shape A / C; curry drop
- Read-only, must stay green unedited: `resource_test.go` in each of `npc/`, `quest/`, `recipe/` under `services/atlas-npc-conversations/atlas.com/npc/conversation/`

Module root: `services/atlas-npc-conversations/atlas.com/npc`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`, `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then `ConversationIdHandler` / `ParseConversationId` / `NpcIdHandler` / `ParseNpcId` / `QuestIdHandler` / `ParseQuestId` / `ItemIdHandler` / `ParseItemId` verbatim. Omit the local `ParseInput`. Prune `context`, `io`, `gorm.io/gorm`; `fmt` stays (`ParseNpcId`/`ParseQuestId`/`ParseItemId` use `fmt.Sscanf`).

- [ ] **Step 2: Convert the four resource files**

In each: `grep -n 'd\.DB()' <file>` locates the sites (20 across the four). Shape A on each enclosing handler; Shape C elsewhere. Drop the `(db)` curry from both register expressions in each `InitResource` and pass `(db)` at each registration site. **Watch for a local `db :=` inside a converted body** — rename it to `tx`.

- [ ] **Step 3: Build and test**

```bash
cd services/atlas-npc-conversations/atlas.com/npc && go build ./... && go test ./...
```

Expected: both exit 0, all three resource tests passing unedited.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc
git commit -m "refactor(atlas-npc-conversations): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 5: No FR-1.3 delegation for this service — record why**

Do not add a delegation commit here. All four helpers fail the "bare lookup" test:

- `ParseNpcId`, `ParseQuestId`, `ParseItemId` parse with `fmt.Sscanf(s, "%d", &v)`, which accepts a numeric *prefix* (`"12abc"` succeeds). `server.ParseIntId` uses `strconv.Atoi`, which rejects it. Delegating would narrow accepted input — a behavior change FR-4.3 forbids.
- `ParseItemId` additionally rejects `itemId == 0`, which no shared parser does.
- `ParseConversationId` is a bare `mux.Vars` + `uuid.Parse` and *would* be delegatable in isolation, but it is left with its siblings so this service's helper block stays internally consistent; converting one of four buys nothing.

State this in the task report so the reviewer does not read the omission as an oversight.

---

## Task 19: atlas-mts

18 `d.DB()` sites across five resource files, 6 `rest.RegisterInputHandler` call sites, plus the one known local-variable collision in the fleet. **Deliberately left large** (7 files, one module): `rest/handler.go` and its consumers must change together or the module does not build, so this cannot be split across two implementer contexts. See `context.md`.

### Files

- `services/atlas-mts/atlas.com/mts/rest/handler.go` — replace lines 1–99 with the alias form; keep the five `Parse*` helpers (lines 100–180)
- `services/atlas-mts/atlas.com/mts/listing/resource.go` — Shape A / C; curry drop
- `services/atlas-mts/atlas.com/mts/transaction/resource.go` — Shape A / C; curry drop
- `services/atlas-mts/atlas.com/mts/wish/resource.go` — Shape A / C; curry drop
- `services/atlas-mts/atlas.com/mts/holding/resource.go` — Shape A / C; curry drop
- `services/atlas-mts/atlas.com/mts/testsupport/resource.go` — Shape A / C; curry drop; **`db` collision at line 274**
- `services/atlas-mts/atlas.com/mts/wallet/resource.go` — mostly Shape C (`InitResource` at line 62 registers, but the file has no `d.DB()`); curry drop only. Its `type ReaderFactory func(d *rest.HandlerDependency) BalanceReader` at line 28 keeps compiling unchanged — that is the alias doing its job
- `services/atlas-mts/atlas.com/mts/listing/resource_test.go`, `listing/cancel_resource_test.go`, `holding/resource_test.go`, `wish/resource_test.go`, `testsupport/resource_test.go` — read-only; must stay green

Module root: `services/atlas-mts/atlas.com/mts`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28`, `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then the five existing `Parse*` helpers (`ParseWorldId`, `ParseCharacterId`, `ParseAccountId`, `ParseListingId`, `ParseHoldingId`) verbatim, with their doc comments. This file declares no named `*Handler` helper types — the helpers already take unnamed `next func(...)`. Omit the local `ParseInput`. Prune `context`, `io`, `gorm.io/gorm`. Confirm the exact import path for `world` from the current file before writing it.

- [ ] **Step 2: Convert the six resource files**

In each: `grep -n 'd\.DB()' <file>` locates the sites (18 across the five that have any). Shape A on each enclosing handler; Shape C elsewhere. `wallet/resource.go` has no `d.DB()` — its `InitResource` only loses the `(db)` curry level. Drop the `(db)` curry from both register expressions in each `InitResource` and pass `(db)` at each registration site.

- [ ] **Step 3: Fix the known `db` collision in `testsupport/resource.go`**

Line 274 declares `db := d.DB().WithContext(d.Context())` inside a handler that, post-transform, closes over an outer `db`. Shadowing compiles but reads as a bug. Rename the local:

```go
tx := db.WithContext(d.Context())
```

…and update its uses within that function. The same file's line 397 (`listing.BackdateEndsAt(d.DB().WithContext(…), …)`) and line 420 (`task.Sweep(d.Logger(), d.Context(), d.DB())`) are plain `d.DB()` → `db` substitutions with no collision.

- [ ] **Step 4: Build and test**

```bash
cd services/atlas-mts/atlas.com/mts && go build ./... && go test ./...
```

Expected: both exit 0, all five resource tests passing unedited.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-mts/atlas.com/mts
git commit -m "refactor(atlas-mts): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 6: Delegate the bare-lookup id parsers (FR-1.3) — separate commit**

Correction to design §4.3, which claims these already delegate: they do not. Each hand-rolls `mux.Vars` plus `strconv.ParseUint` (the design's grep looked for `strconv.Atoi` and missed them). All five are bare lookups, and all already take unnamed func parameter types, so only the bodies change. `world.Id` is `type Id byte` (`libs/atlas-constants/world/constants.go:3`), which satisfies `server.IntegerId`'s `~uint8` term:

```go
// ParseWorldId parses the {worldId} path var into a world.Id (a byte).
func ParseWorldId(l logrus.FieldLogger, next func(worldId world.Id) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[world.Id](l, "worldId", next)
}

// ParseCharacterId parses the {characterId} path var into a uint32.
func ParseCharacterId(l logrus.FieldLogger, next func(characterId uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}

// ParseAccountId parses the {accountId} path var into a uint32.
func ParseAccountId(l logrus.FieldLogger, next func(accountId uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "accountId", next)
}

// ParseListingId parses the {listingId} path var (a UUID string).
func ParseListingId(l logrus.FieldLogger, next func(listingId string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "listingId", next)
}

// ParseHoldingId parses the {holdingId} path var (a UUID string).
func ParseHoldingId(l logrus.FieldLogger, next func(holdingId string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "holdingId", next)
}
```

Two narrowings worth naming, neither reachable through the router: `strconv.ParseUint(s,10,8)` rejects `"300"` for `worldId` where `strconv.Atoi` + `world.Id(...)` truncates it to `44`; and the `listingId == ""` / `holdingId == ""` checks become absence-only. If either matters to a caller, keep that helper's body and say so in the task report. Prune `strconv` and `mux`.

- [ ] **Step 7: Build, test, commit**

```bash
cd services/atlas-mts/atlas.com/mts && go build ./... && go test ./...
git add services/atlas-mts/atlas.com/mts/rest/handler.go
git commit -m "refactor(atlas-mts): delegate id parsers to shared server parsers"
```

---

## Task 20: atlas-character

26 `d.DB()` sites across seven files — the largest conversion, and the only one with a `d.DB()` site outside a file named `resource.go`. **Deliberately left large** (8 files, one module): `rest/handler.go` and its consumers must change together or the module does not build. See `context.md`.

### Files

- `services/atlas-character/atlas.com/character/rest/handler.go` — replace lines 1–97 with the alias form; keep `ParseCharacterId` / `ParseInventoryType` (lines 98–124)
- `services/atlas-character/atlas.com/character/character/resource.go` — Shape A / C; curry drop. Its `InitResource` at line 27 has a **third curry level** for `nameReservedOf` — preserve that level; only the `(db)` level of the *register* expressions goes
- `services/atlas-character/atlas.com/character/character/name_validity_resource.go` — **Shape B** at line 24: `handleGetNameValidity(nameReservedOf NameReservedFunc) rest.GetHandler` becomes `handleGetNameValidity(db *gorm.DB, nameReservedOf NameReservedFunc) rest.GetHandler`. Do not add a second wrapper layer
- `services/atlas-character/atlas.com/character/teleport_rock/resource.go` — **Shape B** at lines 67 and 97; add `db` as a first parameter
- `services/atlas-character/atlas.com/character/equipslot/resource.go` — Shape A / C; curry drop
- `services/atlas-character/atlas.com/character/pending_change/resource.go` — Shape A / C; curry drop
- `services/atlas-character/atlas.com/character/saved_location/resource.go` — Shape A / C; curry drop
- `services/atlas-character/atlas.com/character/session/history/resource.go` — Shape A / C; curry drop
- `services/atlas-character/atlas.com/character/character/resource_test.go`, `character/name_validity_resource_test.go`, `equipslot/resource_test.go`, `pending_change/resource_test.go`, `teleport_rock/resource_test.go`, `session/history/resource_test.go` — read-only; must stay green

Module root: `services/atlas-character/atlas.com/character`

Patterns to copy: `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-28` (alias block), `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45` (Shape A)

- [ ] **Step 1: Rewrite the scaffolding half of `rest/handler.go`**

```go
package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

…then `CharacterIdHandler` / `ParseCharacterId` / `InventoryTypeHandler` / `ParseInventoryType` verbatim. Omit the local `ParseInput`. Prune `context`, `io`, `gorm.io/gorm`.

- [ ] **Step 2: Convert the two Shape-B files first**

`name_validity_resource.go:24` and `teleport_rock/resource.go:67,97` already return `rest.GetHandler`. Add `db *gorm.DB` as the **first** parameter and replace `d.DB()` with `db` in the body:

```go
func handleGetNameValidity(db *gorm.DB, nameReservedOf NameReservedFunc) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		// ... d.DB() → db ...
	}
}
```

Update their call sites in `character/resource.go`'s `InitResource` to pass `db` first.

- [ ] **Step 3: Convert the five Shape-A/C files**

`character/resource.go`, `equipslot/resource.go`, `pending_change/resource.go`, `saved_location/resource.go`, `session/history/resource.go`. In each: `grep -n 'd\.DB()' <file>` locates the sites. Shape A on each enclosing handler; Shape C elsewhere. Drop the `(db)` curry from every register expression and pass `(db)` at each registration site. In `character/resource.go`, preserve `InitResource`'s extra `nameReservedOf` curry level — only the register expressions change.

- [ ] **Step 4: Build and test**

```bash
cd services/atlas-character/atlas.com/character && go build ./... && go test ./...
```

Expected: both exit 0, all six resource tests passing unedited.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-character/atlas.com/character
git commit -m "refactor(atlas-character): alias rest scaffolding, close db over handler constructors"
```

- [ ] **Step 6: Delegate both id parsers (FR-1.3) — separate commit**

Both bodies are bare `mux.Vars` + `strconv.Atoi`; neither named type has an external reference. `server.IntegerId`'s constraint includes `~int8`, so `ParseInventoryType` delegates too. Replace and delete `CharacterIdHandler` / `InventoryTypeHandler`:

```go
func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}

func ParseInventoryType(l logrus.FieldLogger, next func(int8) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[int8](l, "inventoryType", next)
}
```

Prune `strconv` and `mux`.

- [ ] **Step 7: Build, test, commit**

```bash
cd services/atlas-character/atlas.com/character && go build ./... && go test ./...
git add services/atlas-character/atlas.com/character/rest/handler.go
git commit -m "refactor(atlas-character): delegate id parsers to server.ParseIntId"
```

---

## Task 21: Scaffolding and architecture documentation (FR-5)

Must run **after** every conversion so the "RESOLVED" claim is true when written.

### Files

- `.claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md` — add a prose step for `rest/handler.go`, add row SCAFFOLD-10 to the audit table (currently SCAFFOLD-01..09, header at line 150), and note the new step in `## Conditional Steps` (line 144)
- `docs/architectural-improvements.md` — amend the `## Low: Duplicated Database/REST Boilerplate` entry (heading at ~line 355) to RESOLVED
- `docs/adding-a-new-service.md` — read-only; verify the checklist pointer at lines 16–18 still resolves, then leave it alone (FR-5.3)

Patterns to copy: `docs/architectural-improvements.md`'s `## Low: Kafka Retry Logic` entry (immediately below the target entry) — `### Status: RESOLVED`, an `**Implemented:**` bullet block, a `**Files:**` block, then the preserved `### Original Problem`

- [ ] **Step 1: Verify the `adding-a-new-service.md` pointer resolves**

```bash
sed -n '14,20p' docs/adding-a-new-service.md
test -f .claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md && echo OK
```

Expected: the prose points at `.claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md` and the file exists. No edit; FR-5.3 asks only that the pointer be verified.

- [ ] **Step 2: Add the prose checklist step**

The checklist currently runs `## 3. Bruno Collection (REST services only)` → `## 4. Ingress Route (REST services only)` with no code-level handler guidance between them. Insert a new REST-only numbered step (renumbering the sections below it) covering:

- `rest/handler.go` declares only aliases over `libs/atlas-rest/server` — `HandlerDependency`, `HandlerContext`, `GetHandler`, and (only if the service has an input handler) `InputHandler[M]`; plus `var RegisterHandler = server.RegisterHandler` and, where needed, the generic `RegisterInputHandler` wrapper. Do not declare a local `ParseInput` wrapper — no service calls one.
- Copy from `services/atlas-guilds/atlas.com/guilds/rest/handler.go`, not from an older service.
- A DB-backed service closes its `*gorm.DB` over the handler constructor — `func handleX(db *gorm.DB) rest.GetHandler` — and never puts it on `HandlerDependency`. Reference: `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`.
- Service-specific path-parameter helpers stay in the service's `rest` package and delegate to `server.ParseIntId` / `server.ParseUUIDId` / `server.ParseStringId` when the path segment is a plain int, UUID, or string.

Then add the step to the `## Conditional Steps` list (line 144) marked REST-only.

- [ ] **Step 3: Add SCAFFOLD-10 to the audit table**

Update the section heading from `## Audit verification — SCAFFOLD-01..09` to `## Audit verification — SCAFFOLD-01..10` and append this row, matching the existing `| ID | How to verify | Pass criteria |` form:

| ID | How to verify | Pass criteria |
|----|---------------|---------------|
| SCAFFOLD-10 | `grep -c '= server.HandlerDependency' services/atlas-<service>/atlas.com/<svc>/rest/handler.go`, `grep -c 'type HandlerDependency struct' services/atlas-<service>/atlas.com/<svc>/rest/handler.go`, `grep -c 'gorm.io/gorm' services/atlas-<service>/atlas.com/<svc>/rest/handler.go`, and `grep -rc 'd\.DB()' services/atlas-<service>/` | First is ≥1; the other three are 0. `rest/handler.go` aliases the shared scaffolding (`libs/atlas-rest/server`) and declares no scaffolding of its own. A DB-backed service closes its `*gorm.DB` over the handler constructor — `func handleX(db *gorm.DB) rest.GetHandler` — and never puts it on `HandlerDependency`. Skip for Kafka-only services (no `rest/` package). |

- [ ] **Step 4: Amend `docs/architectural-improvements.md`**

Rewrite the `## Low: Duplicated Database/REST Boilerplate` entry in the house style used by `## Low: Kafka Retry Logic` directly below it: a `### Status: RESOLVED` heading, an `**Implemented:**` bullet block, and the existing `### Problem` / `### Recommendation` text preserved below as `### Original Problem` / `### Original Recommendation`. Do not delete the entry.

The entry names three duplicated files. Verify each before writing the claim:

```bash
find services -path '*/rest/request.go'
find services -path '*/database/connection.go'
grep -rl 'type HandlerDependency struct' --include=handler.go services
```

Expected: all three produce no output — `request.go` and `database/connection.go` were already gone fleet-wide, and `rest/handler.go` was the last of the three. That makes the entry **fully** resolved; write it that way. **If the third grep returns anything, a conversion task was missed — STOP and report rather than writing a false RESOLVED.**

The `**Implemented:**` block should record: 19 services converted to alias form over `libs/atlas-rest/server`; 2 dead handler packages (`atlas-fame`, `atlas-events`) deleted; `*gorm.DB` moved off `HandlerDependency` into the handler-constructor closure at 174 call sites; and the two intended behavior deltas (`ParseEnvironment` now in the chain; malformed-body 400s now carry a JSON:API errors document).

- [ ] **Step 5: Commit**

```bash
git add .claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md docs/architectural-improvements.md
git commit -m "docs(task-301): add SCAFFOLD-10 handler item, mark REST boilerplate duplication resolved"
```

---

## Task 22: Fleet-wide acceptance sweep and verification gate

The per-service tasks each proved their own module. This task proves the fleet-wide acceptance criteria and runs the repo gate.

### Files

- No file changes expected. If this task needs an edit, an earlier task was incomplete — fix it in that service's own commit rather than a catch-all one.

- [ ] **Step 1: Run every acceptance grep from PRD §10**

```bash
grep -rl "type HandlerDependency struct" --include=handler.go services || echo "OK: no hand-rolled struct remains"
grep -rl "= server.HandlerDependency" --include=handler.go services | wc -l   # expect: 59
grep -rn "d\.DB()" --include='*.go' services || echo "OK: no d.DB() remains"
grep -rln 'gorm.io/gorm' --include=handler.go services || echo "OK: no handler.go imports gorm"
test -e services/atlas-fame/atlas.com/fame/rest || echo "OK: atlas-fame/rest deleted"
test -e services/atlas-events/atlas.com/events/rest || echo "OK: atlas-events/rest deleted"
grep -rn 'RegisterSimpleHandler\|RegisterSimpleInputHandler\|RegisterOptionalInputHandler\|ParseOptionalInput' --include='*.go' services || echo "OK: no Simple/Optional variants introduced"
git diff --stat 31a791e3a -- 'services/*/main.go' 'libs/atlas-rest'          # expect: empty
```

Every line must produce the stated result. Record the actual output of each in the task report — a claim without the output is not evidence.

- [ ] **Step 2: Run the flagless verification gate**

```bash
tools/verify.sh
```

Expected: exit 0. `--quick` / `--no-docker` do not count for this gate — they skip the bake and `-race`.

- [ ] **Step 3: Confirm the tree is clean and on the task branch**

```bash
git status --porcelain          # expect: empty
git branch --show-current       # expect: task-301-rest-handler-scaffold-alias
git rev-parse --show-toplevel   # expect: path ending /.worktrees/task-301-rest-handler-scaffold-alias
```

- [ ] **Step 4: Report, do not commit**

This task produces no commit. Report the grep outputs and the `verify.sh` exit code. Code review (per `docs/review-protocol.md`) runs next, before any PR.
