# Review: Task 1 — atlas-messages REST handler scaffold alias (pilot)

Range reviewed: `e4f016344..9d5b7b8a0` (single commit `9d5b7b8a0`).

## Scope

`git diff --stat e4f016344..9d5b7b8a0` shows exactly one file changed:
`services/atlas-messages/atlas.com/messages/rest/handler.go` (4 insertions, 38
deletions). No other file in the range is touched. This matches the brief's
single-file scope exactly.

## Findings

### PASS — file content matches brief verbatim
`cat services/atlas-messages/atlas.com/messages/rest/handler.go` after checkout
is byte-for-byte the block specified in `task-1-brief.md` Step 1:
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
No `InputHandler`/`RegisterInputHandler` block, no local struct, no gorm
import — matches the pattern at
`services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-16` for the aliased
portion (guilds additionally has `InputHandler`/`ParseGuildId`, which
atlas-messages correctly omits since it has none to preserve).

### PASS — global constraints
- `libs/atlas-rest/` unchanged: `git log -p e4f016344..9d5b7b8a0 -- 'libs/atlas-rest/*'` — empty.
- No `main.go` changed: `git log -p e4f016344..9d5b7b8a0 -- '**/main.go'` — empty.
- No processor/provider/entity/Kafka file changed — diff-stat confirms only `rest/handler.go`.
- No `ParseInput` wrapper introduced — absent from the file.
- `server.RegisterHandler` (not `Simple*`) is aliased — `var RegisterHandler = server.RegisterHandler` at `rest/handler.go:13`.
- No `RegisterOptionalInputHandler`/`ParseOptionalInput` introduced.
- No handler renamed/unexported — `chat.handleGetChatHistory` in
  `chat/resource.go:82` is untouched (confirmed via
  `git diff e4f016344..9d5b7b8a0 --stat -- .../chat/resource.go` → empty).
- No test file edited (diff-stat confirms).
- Commit is scoped to atlas-messages only (single commit in range, single service).
- Tracing span name and log fields preserved byte-for-byte: `libs/atlas-rest/server/register.go`'s
  `RegisterHandler` calls `RetrieveSpan(l, handlerName, context.Background(), ...)` and
  builds `logrus.Fields{"originator": handlerName, "type": "rest_handler"}` — identical
  to the deleted local implementation shown in the diff hunk (`rest/handler.go` old
  lines 20–29).

### PASS — acceptance greps (re-run directly, not trusted from report)
```
grep -q 'type HandlerDependency struct' rest/handler.go   -> not found (PASS)
grep -q 'gorm.io/gorm' rest/handler.go                    -> not found (PASS)
grep -rq 'd\.DB()' .                                      -> not found (PASS)
grep -c 'rest\.RegisterInputHandler' -r .                 -> 0 in every file (PASS)
```

### PASS — build and full test suite (module-local, re-run directly)
```
cd services/atlas-messages/atlas.com/messages && go build ./... && go vet ./... && go test ./...
```
All packages `ok` (or `[no test files]`), 0 failures, `go vet` clean.

### PASS — `chat/resource.go` compiles unchanged against the alias
`chat/resource.go:76` (`rest.RegisterHandler(l)(si)`) and `:82`
(`func handleGetChatHistory(d *rest.HandlerDependency, c *rest.HandlerContext) ...`)
are untouched and compile/test successfully against the new alias, confirming
arity/type compatibility as claimed.

## Note (non-blocking): behavior surface not exercised by this pilot

`server.RegisterHandler` now runs `ParseEnvironment` between `RetrieveSpan` and
`ParseTenant` (see `libs/atlas-rest/server/register.go:16`), which the deleted
local wrapper never did. This is the FR-4 environment-header behavior change
the task brief explicitly anticipates ("None of them sets an `ENVIRONMENT`
header, so they exercise the unchanged legacy path"). No atlas-messages
resource test sets that header, so this pilot does not exercise the new path
end-to-end — that is expected and stated as in-scope-elsewhere (owned by
`libs/atlas-rest/server/handler_test.go`/`context_test.go` per Global
Constraints), not a defect in this unit.

## Verdict

Spec compliance: full — every brief step and every applicable global
constraint is satisfied, verified independently (not merely trusted from the
implementer report).

Task quality: high — minimal, surgical diff; no scope creep; commit message
follows convention; build/tests pass; recipe is sound as the template for the
remaining 20 services.

No blocking or non-blocking defects found in the diff itself.
