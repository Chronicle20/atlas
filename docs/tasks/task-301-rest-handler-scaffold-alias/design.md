# REST Handler Scaffold Consolidation — Design

Version: v1
Status: Draft
Created: 2026-09-04
PRD: [prd.md](prd.md)
Branch point: `f1e4ac405` (worktree `task-301-rest-handler-scaffold-alias`)

---

## 1. Scope of this document

The PRD settles *what* changes. This document settles *how*, and records four
places where fresh evidence from the tree contradicts or refines the PRD.
Those are collected in §9 so the planner does not have to reconcile them by
reading both documents.

Everything below was re-derived from the worktree at `f1e4ac405`, not carried
over from the PRD's tables.

---

## 2. Evidence baseline

Re-derived counts (the PRD asks for this in §7):

| Fact | Command | Result |
|---|---|---|
| `rest/handler.go` files | `find services -name handler.go -path '*/rest/*'` | 61 |
| Hand-rolled | `grep -rl "type HandlerDependency struct" --include=handler.go services` | 21 |
| Alias form | `grep -rl "= server.HandlerDependency" --include=handler.go services` | 40 |
| `d.DB()` call sites | `grep -rn "d\.DB()" --include='*.go' services` | 174 |
| Files containing `d.DB()` | same, `-l` | 40 |
| Files declaring `InitResource` in the 21 | `grep -rl 'func InitResource'` | 45 |
| Resource-level test files in the 21 | `grep -rln 'InitResource\|initResource' --include='*_test.go'` | 42 |

All 21 match the PRD's service list exactly. The per-service `rest.Register*`
counts in the PRD §7 table reproduce (e.g. atlas-character 7 `RegisterHandler`
+ 9 `RegisterInputHandler` = 16; atlas-mts 6 + 6 = 12; atlas-account 1 + 4 = 5).

**Every one of the 174 `d.DB()` call sites lives in a file named `resource.go`,
with exactly one exception:**
`services/atlas-character/atlas.com/character/character/name_validity_resource.go`.
The blast radius of FR-2 is therefore 40 files, all of them REST resource
files. Nothing in a processor, provider, entity, or Kafka package is touched.

Each service is its own Go module (69 `go.mod` files under `services/`), so a
per-service commit builds and tests independently.

---

## 3. The load-bearing architectural decision: alias, not move

`type HandlerDependency = server.HandlerDependency` is a **type alias**, not a
type definition. This is what makes a 40-file, 174-call-site sweep tractable,
and it deserves to be stated as the design's central choice rather than
treated as cosmetic:

- Every existing `*rest.HandlerDependency` / `*rest.HandlerContext` /
  `rest.GetHandler` / `rest.InputHandler[M]` reference in the 45 resource
  files, the 42 resource tests, and helper signatures like
  `func writeModel(d *rest.HandlerDependency, ...)` and
  `type ReaderFactory func(d *rest.HandlerDependency) BalanceReader`
  (`services/atlas-mts/atlas.com/mts/wallet/resource.go:27`) keeps compiling
  **unchanged**. The alias is the same type.
- Exactly two things break, and both break at compile time:
  1. `d.DB()` — the method does not exist on `server.HandlerDependency`
     (`libs/atlas-rest/server/context.go:13-28` declares only `Logger()` and
     `Context()`).
  2. The `(db)` curry level — `server.RegisterHandler` is
     `func(l) func(si) func(name, handler)`, one level shallower than the
     hand-rolled `func(l) func(db) func(si) func(name, handler)`.

**The compiler is the exhaustive checker for this sweep.** There is no silent
failure mode where a call site is missed: `go build ./...` in each service
module either finds every `d.DB()` and every stale curry or the conversion is
complete. This is the concrete reason the PRD can decline a lint rule or
`verify.sh` guard (PRD §2 non-goals) without leaving a hole — the guard would
only protect *future* services, which is exactly what FR-5 addresses instead.

The alternative — defining `type HandlerDependency server.HandlerDependency`
or moving the type into each service — was not considered seriously: either
would break all ~200 unqualified references and turn a bounded transform into
a rewrite.

---

## 4. Target shape of a converted `rest/handler.go`

### 4.1 The declaration set is minimal, not templated

FR-1.1 shows a seven-declaration template; FR-1.2 says to omit unused
declarations. Fresh evidence makes the second rule bite harder than the PRD
anticipated:

```
grep -rn 'rest\.ParseInput' --include='*.go' services   →  0 matches
```

**No service in the fleet calls `rest.ParseInput` directly.** In the
hand-rolled form the local `ParseInput` is reached only from the local
`RegisterInputHandler`. Once `RegisterInputHandler` delegates to
`server.RegisterInputHandler` — which calls `server.ParseInput` internally
(`libs/atlas-rest/server/register.go:30-42`) — the local `ParseInput` wrapper
is dead in **all 21** converted files.

Decision: **omit the `ParseInput` wrapper from every converted file.** Go does
not error on an unused package-level function, so keeping it would compile;
it would also be 21 new pieces of dead code landed by a task whose entire
purpose is deleting dead scaffolding. This deviates from the literal FR-1.1
template and is intentional; §9.1 records it.

The resulting declaration set, per service:

**Always** (all 19 converted services):

```go
type HandlerDependency = server.HandlerDependency
type HandlerContext = server.HandlerContext
type GetHandler = server.GetHandler

var RegisterHandler = server.RegisterHandler
```

`GetHandler` is not optional even where it is unreferenced today: after the
FR-2.1 transform every DB-backed handler constructor returns `rest.GetHandler`.

**Additionally, where `rest.RegisterInputHandler` has at least one call site:**

```go
type InputHandler[M any] = server.InputHandler[M]

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}
```

`RegisterInputHandler` cannot be a `var` the way `RegisterHandler` can: Go has
no generic variables, so the wrapper function is required. `RegisterHandler`
is non-generic and takes the `var` form, matching
`services/atlas-guilds/atlas.com/guilds/rest/handler.go:26`.

**Services that omit the input block** (verified `rest.RegisterInputHandler`
call-site count = 0): `atlas-messages`, `atlas-mounts`,
`atlas-drop-information`, `atlas-rankings`.

Note that `atlas-drop-information` and `atlas-rankings` *declare*
`InputHandler` and `ParseInput` today with no caller — they are already
carrying dead scaffolding, and this is the task that removes it.

### 4.2 No `RegisterSimpleHandler`

FR-4.4 confirmed against the source: every hand-rolled `RegisterHandler`
chains `server.ParseTenant` (e.g.
`services/atlas-character/atlas.com/character/rest/handler.go:71-80`), and
`server.RegisterSimpleHandler` (`register.go:80-92`) omits it. All 21 target
`server.RegisterHandler` / `server.RegisterInputHandler`.

`RegisterOptionalInputHandler` / `ParseOptionalInput` also exist in the
library. Verified: zero references anywhere in the 21. They are not part of
this task.

### 4.3 Path-parameter helpers (FR-1.3)

Each of the 21 declares 0–7 `Parse*` helpers plus a named `next` type:

```go
type CharacterIdHandler func(characterId uint32) http.HandlerFunc

func ParseCharacterId(l logrus.FieldLogger, next CharacterIdHandler) http.HandlerFunc {
	// strconv.Atoi(mux.Vars(r)["characterId"]) ...
}
```

Two facts settle how to handle these:

- `server.ParseIntId[T]`'s constraint is
  `~uint32 | ~int32 | ~int8 | ~uint8 | ~uint16` (`id_parser.go:12-14`), which
  covers every integer helper found, including `atlas-character`'s
  `int8` `ParseInventoryType`. `server.ParseUUIDId` and `server.ParseStringId`
  cover the rest.
- **The named `*IdHandler` types have zero references outside their own
  `handler.go`** (`grep -rn 'rest\.CharacterIdHandler\|rest\.NpcIdHandler\|…'`
  → 0). They can be deleted rather than preserved.

Decision: for helpers whose body is a bare `strconv.Atoi` / `uuid.Parse` /
`mux.Vars` lookup, replace body **and** signature with the `atlas-guilds`
form, deleting the named type:

```go
func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}
```

Call sites pass func literals, which satisfy the unnamed parameter type
unchanged. Helpers whose parsing is *not* a bare lookup keep their bodies:
identified candidates are `atlas-account`'s `AccountIdAndWorldIdHandler`,
`atlas-party-quests`/`atlas-events`' `FieldInstanceHandler`, and
`atlas-map-actions`' `ScriptNameHandler`. `atlas-mts` (6 helpers, zero
`strconv.Atoi`/`uuid.Parse`) and `atlas-parcel` already delegate.

Per PRD open question 2: this is the one part of the change that is
droppable. If review finds it noisy, dropping it costs nothing — FR-1.4 and
FR-1.5 are unaffected. It should be a *separate commit per service* so it can
be dropped by `git revert` rather than by re-editing.

---

## 5. Moving `*gorm.DB` off the dependency (FR-2)

### 5.1 Three call-site shapes, three transforms

The 40 affected files are not uniform. Applying one transform blindly would
produce needless double-currying.

**Shape A — bare handler, uses `d.DB()`.** The common case.

```go
// before
func handleGetFame(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc { … d.DB() … }
// after
func handleGetFame(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc { … db … }
}
```

Reference implementation:
`services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`.

**Shape B — constructor already parameterized.** Add `db` as a parameter; do
**not** add a second wrapper layer.

```go
// before: func handleGetNameValidity(nameReservedOf NameReservedFunc) rest.GetHandler
// after:  func handleGetNameValidity(db *gorm.DB, nameReservedOf NameReservedFunc) rest.GetHandler
```

Known Shape-B sites: `atlas-character` `teleport_rock/resource.go:67,97` and
`character/name_validity_resource.go:24`; `atlas-mts`
`wallet/resource.go:76`; `atlas-quest` `quest/resource.go:87`
(`handleGetQuestsByCharacterAndState(db, state)` — already Shape B *and*
already db-parameterized).

**Shape C — handler never touches the DB.** Leave the function completely
alone; only the registration line loses `(db)`. Applies to all of
`atlas-messages` (zero `d.DB()` in the service) and to most of
`atlas-mts/wallet`.

**Selection rule:** add the `db` parameter if and only if `d.DB()` appears in
that function's body or in a closure it owns. Do not curry a handler that does
not need it.

### 5.2 `atlas-quest` is partially migrated already (PRD open question 1)

The PRD flags atlas-quest's 15 `rest.Parse*` against 3 `d.DB()` as anomalous.
Cause found: six of its handlers are **already** in the target shape —
`handleGetQuestsByCharacter(db *gorm.DB) rest.GetHandler` and five siblings
(`quest/resource.go:57,87,117,234,258,344`). The three remaining `d.DB()` are
in its three *input* handlers (`handleStartQuest:153`,
`handleCompleteQuest:198`, `handleUpdateQuestProgress:327`), which were never
curried. atlas-quest therefore needs Shape A applied to three functions and
the curry drop everywhere; the FR-2.1 transform applies unchanged. Open
question 1 is resolved, no special handling needed.

### 5.3 Local-variable collision

`services/atlas-mts/atlas.com/mts/testsupport/resource.go:274` declares
`db := d.DB().WithContext(d.Context())` inside a handler that, post-transform,
closes over an outer `db`. Shadowing compiles but reads as a bug. Rename the
local:

```go
tx := db.WithContext(d.Context())
```

The same file's line 397 (`listing.BackdateEndsAt(d.DB().WithContext(…), …)`)
and line 420 (`task.Sweep(d.Logger(), d.Context(), d.DB())`) are plain
substitutions. This is the only collision found; implementers must still
check for it per file rather than assume.

### 5.4 Exported handler functions

Six services expose their handlers as exported identifiers rather than
`handleX` — `GetAllScriptsHandler`, `CreateNoteHandler`,
`ValidateConversationHandler`, `GetTimerByCharacterHandler`, and ~35 siblings
across `atlas-map-actions`, `atlas-notes`, `atlas-npc-conversations`,
`atlas-portal-actions`, `atlas-reactor-actions`, `atlas-party-quests`.

Verified: **no test and no other package references any of them** — they are
exported by habit, not by contract. Change their signature in place per §5.1.
Do **not** rename or unexport them; that is a separate cleanup and would
inflate the diff for no acceptance-criteria gain.

### 5.5 `InitResource` is untouched (FR-2.2)

Confirmed across the 21: the signature is
`func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer`,
with one variant — `atlas-character`'s `character/resource.go:27` has a third
curry level for `nameReservedOf`. In every case `db` is already in scope at
the point where handler constructors are called, so no `main.go` changes and
no `InitResource` signature changes. The 42 resource tests, which call
`InitResource(si)(db)(router, l)`, keep compiling.

Registration lines change only by dropping one call:

```go
registerGet := rest.RegisterHandler(l)(db)(si)   →  rest.RegisterHandler(l)(si)
rest.RegisterInputHandler[M](l)(db)(si)(name, h) →  rest.RegisterInputHandler[M](l)(si)(name, h)
```

---

## 6. Behavior deltas — corrected (FR-4)

### 6.1 FR-4.1 as written overstates the change

The PRD says conversion will "run `env.Reconcile` inside `ParseTenant`" for
the first time. That is not accurate: the hand-rolled chains already call
**`server.ParseTenant`** — the shared one — so `env.Reconcile` already runs in
all 21 today (`libs/atlas-rest/server/handler.go:151`).

What actually happens today:

- `ParseEnvironment` never runs, so nothing calls `env.WithContext` before
  `ParseTenant`.
- `env.MustFromContext(tctx)` returns `""`. It does not panic —
  `libs/atlas-env/env.go:71-74` is `id, _ := FromContext(ctx)(); return id`.
- `Reconcile(registry, "", tenantId)` (`libs/atlas-env/tenants.go:58-85`)
  with an empty header env returns the tenant's projected environment and
  **cannot** return an error: the `headerEnv != tenantEnv` branch at line 80 is
  unreachable when `headerEnv == ""` (line 77 returns first).

So the precise delta is:

| Request | Today (21 services) | After conversion |
|---|---|---|
| No `ENVIRONMENT` header | env resolved from tenant projection; never 400s at this gate | **Byte-identical.** `ParseEnvironment` short-circuits on `id == ""` (`handler.go:71`) and `IsProvisionable("")` returns `true` (`registry.go:192-194`). |
| `ENVIRONMENT` header naming a known PROVISIONING/ACTIVE env matching the tenant | header ignored; env taken from projection | Admitted; `environment` log field added; same resolved env. |
| `ENVIRONMENT` header naming an unknown / DEACTIVATING / DELETED env | **silently served from baseline data** | 400 at `ParseEnvironment`. |
| `ENVIRONMENT` header disagreeing with the tenant's projected env | **silently served** — `Reconcile`'s mismatch branch is unreachable | 400 at `ParseTenant`. |

This is a live delta, not a theoretical one: **all 21 install a real
`env.MapRegistry`** via `service.WithEnvironmentRegistry`
(`libs/atlas-service/envregistry.go:33,52-54`) — verified per service. The
permissive `legacyRegistry` default (`registry.go:261-273`) does not apply to
them.

Framed correctly, these 21 are currently the *only* REST services in the fleet
that ignore the `ENVIRONMENT` header. FR-4.1 is closing that gap, not
inventing a new gate. The PR description must say this in these terms.

### 6.2 FR-4.2 stands as written

Hand-rolled `ParseInput` writes a bare `w.WriteHeader(http.StatusBadRequest)`
(e.g. `atlas-character/rest/handler.go:53,60`); `server.ParseInput`
(`context.go:47-67`) calls `writeBadRequestJSONAPI`. Status unchanged at 400,
body goes from empty to a JSON:API errors document.

### 6.3 FR-4.3 verified

Both chains emit `logrus.Fields{"originator": handlerName, "type":
"rest_handler"}` and both name the span `handlerName` via
`server.RetrieveSpan`. Compared line by line
(`register.go:11-24` vs `atlas-character/rest/handler.go:69-83`) — identical.

---

## 7. Testing strategy

**No new tests are required, and here is the evidence for that claim rather
than an assumption.**

`libs/atlas-rest/server` already owns the behavior FR-4 introduces:

- `handler_test.go`: `TestParseEnvironmentPutsTheHeaderOnTheContext`,
  `…WithNoHeaderIsTheLegacyPath`, `…RejectsAnUnknownEnvironment`,
  `…AdmitsAProvisioningEnvironment`, `…RejectsADeactivatingEnvironment`,
  `…RejectsADeletedEnvironment`, `TestParseTenantRejectsAMismatchedEnvironment`.
- `context_test.go`: `TestParseInput_MalformedJSONIs400JSONAPIShape`,
  `TestParseInput_EmptyBodyIsStill400JSONAPIShape`.

Since `libs/atlas-rest/` is explicitly not touched (PRD acceptance criteria),
that coverage is already green and stays green. The converted services are
consumers of it.

What must actually be run and reported:

1. Per service, in its own module: `go build ./...` and `go test ./...`.
2. The 42 resource-test files are the regression net for the conversion. None
   of them sets an `ENVIRONMENT` header (they exercise the legacy path), so
   per §6.1 row 1 they should pass unchanged. **If any resource test fails,
   that is a finding to report, not a test to edit** — a failure means the
   conversion changed the header-absent path, which it must not.
3. Flagless `tools/verify.sh` exits 0 before the branch is called done.

---

## 8. Decomposition and commit strategy (PRD open question 3)

Per-service commits. Each service is its own Go module, so each commit builds
and tests in isolation and a bisect lands on one service — which is what PRD
§8 "Reviewability" asks for.

**Stage 0 — pilots (validate the recipe before fanning out).**

1. `atlas-messages` — 47-line handler.go, no `*gorm.DB` at all, one route.
   Pure alias swap + curry drop. Proves the alias mechanics.
2. `atlas-mounts` — one `d.DB()`, one route, no `InputHandler`. Proves the
   Shape-A transform end to end.

**Stage 1 — deletions (FR-3).** One commit removing both dead packages.
Freshly re-verified at design time: `grep -rn '"atlas-fame/rest"'` → 0 and
`grep -rn '"atlas-events/rest"'` → 0; `atlas-events` already registers via
`server.RegisterHandler(l)(si)` at `event/definition/resource.go:32` and
`event/occurrence/resource.go:30`. Both `rest/` directories contain only
`handler.go`, so the directory goes with the file. FR-3.3 still requires this
grep be re-run at implementation time — the design-time result does not
discharge it.

**Stage 2 — the remaining 17 conversions**, ordered small→large so the recipe
is well-exercised before it meets `atlas-character`:

`atlas-rankings` (3), `atlas-drop-information` (4), `atlas-pets` (4),
`atlas-quest` (3, partially migrated), `atlas-map-actions` (6),
`atlas-parcel` (6), `atlas-portal-actions` (6), `atlas-reactor-actions` (6),
`atlas-notes` (7), `atlas-account` (9), `atlas-party-quests` (12),
`atlas-npc-shops` (12), `atlas-ban` (15), `atlas-reward-pools` (16),
`atlas-mts` (18), `atlas-npc-conversations` (20), `atlas-character` (26).
(Number is that service's `d.DB()` count.)

The four `script/resource.go` services — `atlas-map-actions`,
`atlas-portal-actions`, `atlas-reactor-actions` — are near-identical copies of
each other (same six exported `*ScriptHandler` functions at the same line
numbers). Convert one, then apply the same edit to the others; do not
re-derive.

**Stage 3 — FR-1.3 parser delegation**, one commit per service, separate from
that service's Stage-2 commit so it is independently revertible (§4.3).

**Stage 4 — docs (FR-5).** One commit.

Independence: the 19 services share no code, so Stage 2 parallelizes across
fresh-context agents without conflict. Stage 4 touches two shared files and
must be serialized after Stage 2/3 so its claims are true when written.

---

## 9. Where this design corrects the PRD

### 9.1 `ParseInput` wrapper is omitted everywhere (refines FR-1.1/FR-1.2)

FR-1.1's template includes `func ParseInput[M any](…)`. Zero services call
`rest.ParseInput`, so the wrapper is dead in all 21 once
`RegisterInputHandler` delegates. Omitting it is the FR-1.2 rule applied
consistently. FR-1.4 and FR-1.5 acceptance are unaffected.

### 9.2 FR-4.1's third bullet is wrong (§6.1)

`env.Reconcile` already runs in these 21 today, because they already use
`server.ParseTenant`. What changes is that a *non-empty* header env can now
reach it. The corrected table in §6.1 is what belongs in the PR description.

### 9.3 `docs/architectural-improvements.md` line numbers are stale (FR-5.4)

FR-5.4 cites "lines 87–93" and a task-208 `**Done so far:**` entry at line 53.
Neither exists in the file at `f1e4ac405` (406 lines total; `grep -n 'Done so
far'` → no match). The actual entry is:

> `## Low: Duplicated Database/REST Boilerplate` — line ~355, with the
> `### Problem` text at line 361 and `### Recommendation` following.

The house style for a completed entry in this file is
`### Status: RESOLVED` followed by an `**Implemented:**` bullet block — see
`## Low: Kafka Retry Logic` immediately below it.

Further: that entry names three duplicated files — `database/connection.go`,
`rest/handler.go`, and `rest/request.go`. Verified:

```
find services -path '*/rest/request.go'          →  0
find services -path '*/database/connection.go'   →  0
```

Both are already gone fleet-wide. `rest/handler.go` is the last of the three,
so this task makes the entry **fully** RESOLVED rather than partially — write
it that way, in the Kafka-Retry style, and keep the `### Original Problem`
text below it as that entry does.

### 9.4 PRD §7's "19 converted" is right, but two of the 19 are near-no-ops

`atlas-messages` has no `*gorm.DB` anywhere and `atlas-mounts` has one
`d.DB()`. Both are pilots (§8 Stage 0), not representative work items; the
plan should not size them like their peers.

---

## 10. Scaffolding documentation (FR-5)

`.claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md`
has no `rest/handler.go` item — confirmed; `SCAFFOLD-04` is
`deploy/shared/routes.conf`. The audit table runs SCAFFOLD-01..09, so the new
item is **SCAFFOLD-10**, added in the existing
`| ID | How to verify | Pass criteria |` form.

Proposed row (the planner should treat the wording as a starting point, the
commands as fixed):

| ID | How to verify | Pass criteria |
|----|---------------|---------------|
| SCAFFOLD-10 | `grep -c '= server.HandlerDependency' services/atlas-<service>/atlas.com/<svc>/rest/handler.go` and `grep -c 'type HandlerDependency struct' …` and `grep -c 'gorm.io/gorm' …` | First is ≥1; second and third are 0. `rest/handler.go` aliases the shared scaffolding (`libs/atlas-rest/server`) and declares no scaffolding of its own. A DB-backed service closes its `*gorm.DB` over the handler constructor — `func handleX(db *gorm.DB) rest.GetHandler` — and never puts it on `HandlerDependency`; `grep -c 'd\.DB()' -r services/atlas-<service>` must be 0. Skip for Kafka-only services (no `rest/` package). |

The prose checklist above the audit table also needs a short section (it
currently jumps from Bruno to ingress with no code-level handler guidance);
place it as a new numbered step and update the `## Conditional Steps` list to
note it is REST-only.

FR-5.3: `docs/adding-a-new-service.md:18` — verify the pointer to this
checklist still resolves; no structural change.

---

## 11. Risks

| Risk | Assessment |
|---|---|
| A missed `d.DB()` or stale curry ships silently | Not possible — both are compile errors (§3). |
| FR-4.1 breaks a live caller sending a bad `ENVIRONMENT` header | Real, and intended. Such a caller is today being served baseline data for an environment it named — the silent-wrong-answer case FR-4.1 exists to end. Must be in the PR description. |
| A resource test starts failing | Would mean the header-absent path changed, which §6.1 says it must not. Treat as a finding, not a test to fix (§7). |
| FR-1.3 inflates review surface | Mitigated by isolating it to its own per-service commits (§8 Stage 3), so it is revertible without touching the conversion. |
| Docs claims outrun the code | Stage 4 is serialized last (§8) so FR-5.4's "RESOLVED" is true when written. |
| `atlas-mts/testsupport` `db` shadowing reads as a bug | Rename the local to `tx` (§5.3). One known site; check per file. |

---

## 12. Out of scope (restated for the planner)

Unchanged from PRD §2, with one addition: `RegisterOptionalInputHandler` /
`ParseOptionalInput` exist in the library and have zero references in the 21
(§4.2). Do not introduce them.

No file under `libs/atlas-rest/` changes. No `main.go` changes. No
`InitResource` signature changes. No processor, provider, entity, Kafka, or
`atlas-ui` changes.
