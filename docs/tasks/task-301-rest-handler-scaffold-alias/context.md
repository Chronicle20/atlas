# task-301 — Planning Context

Companion to `plan.md`. Records the key files, the decisions the plan bakes in,
the places the plan corrects `design.md`, and the sizing calls a reviewer would
otherwise have to reconstruct.

## Key files

| File | Role |
|---|---|
| `libs/atlas-rest/server/context.go:14-28` | `server.HandlerDependency` — declares only `Logger()` and `Context()`. The absence of `DB()` is what makes `d.DB()` a compile error after the alias swap. **Not touched.** |
| `libs/atlas-rest/server/register.go:11-24` | `server.RegisterHandler` — `func(l) func(si) func(name, handler)`, one curry level shallower than the hand-rolled form. **Not touched.** |
| `libs/atlas-rest/server/handler.go:67-100` | `ParseEnvironment` / `ParseTenant` — the FR-4.1 delta. **Not touched.** |
| `libs/atlas-rest/server/id_parser.go` | `ParseIntId[T]` (constraint `~uint32 \| ~int32 \| ~int8 \| ~uint8 \| ~uint16`), `ParseUUIDId`, `ParseStringId` — the FR-1.3 delegation targets. |
| `services/atlas-guilds/atlas.com/guilds/rest/handler.go:1-36` | The canonical alias-form `handler.go`. Every conversion copies its shape. |
| `services/atlas-configurations/atlas.com/configurations/environments/resource.go:45` | The canonical Shape-A handler constructor: `func handleX(db *gorm.DB) rest.GetHandler`. |
| `.claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md:150` | `## Audit verification — SCAFFOLD-01..09`; Task 21 appends SCAFFOLD-10 here. |
| `docs/architectural-improvements.md` ~line 355 | `## Low: Duplicated Database/REST Boilerplate`; Task 21 amends it to RESOLVED in the `## Low: Kafka Retry Logic` house style. |

## Decisions baked into the plan

1. **Alias, not move.** `type HandlerDependency = server.HandlerDependency` is a
   type alias, so all ~200 existing `*rest.HandlerDependency` references — plus
   helper signatures like `type ReaderFactory func(d *rest.HandlerDependency) BalanceReader`
   (`services/atlas-mts/atlas.com/mts/wallet/resource.go:28`) and all 42 resource
   tests — keep compiling untouched. Exactly two things break, both at compile
   time: `d.DB()` and the `(db)` curry level. **The compiler is the exhaustive
   checker**; there is no silent-miss failure mode, which is why the PRD can
   decline a lint guard without leaving a hole.

2. **No local `ParseInput` wrapper anywhere.** `grep -rn 'rest\.ParseInput'` → 0
   fleet-wide. Once `RegisterInputHandler` delegates, the wrapper is dead in all
   21. Keeping it would land 21 new pieces of dead code in a task whose purpose
   is deleting dead scaffolding. Deviates from PRD FR-1.1's literal template
   (design §9.1).

3. **`RegisterHandler` is a `var`, `RegisterInputHandler` is a function.** Go has
   no generic variables, so the input variant needs the wrapper function.

4. **Four services omit the input block entirely** — verified
   `rest.RegisterInputHandler` call-site count is 0 for `atlas-messages`,
   `atlas-mounts`, `atlas-drop-information`, `atlas-rankings`. The last two
   *declare* `InputHandler`/`ParseInput` today with no caller; that is
   pre-existing dead scaffolding this task removes (FR-1.2).

5. **FR-1.3 parser delegation is a second commit per service**, always. It is the
   droppable part of the change (PRD open question 2). Isolating it means review
   can `git revert` it without re-editing the conversion.

6. **Named `*IdHandler` types are deleted, not preserved**, wherever their helper
   delegates — verified zero references outside their own `handler.go`. Call
   sites pass func literals, which satisfy the unnamed parameter type unchanged.

7. **No new tests.** `libs/atlas-rest/server/{handler,context}_test.go` already
   own the FR-4 behavior and that library is untouched. The 42 resource tests are
   the regression net; none sets an `ENVIRONMENT` header, so they exercise the
   unchanged legacy path. A resource-test failure is a **finding to report**, not
   a test to edit.

## Where the plan corrects `design.md`

- **§4.3 is wrong about `atlas-mts` and `atlas-parcel` "already delegating."**
  They do not. Their helpers hand-roll `mux.Vars` + `strconv.ParseUint` — the
  design's grep looked for `strconv.Atoi` and missed them. Both are bare lookups
  and both are delegated in Tasks 8 and 19, with the two narrowings
  (`ParseUint(...,10,8)` overflow rejection; present-but-empty string checks)
  named inline as unreachable through gorilla mux.

- **`atlas-npc-conversations` gets no delegation commit at all** — a case the
  design does not enumerate. Three of its four helpers parse with
  `fmt.Sscanf(s, "%d", &v)`, which accepts a numeric prefix (`"12abc"` succeeds)
  where `strconv.Atoi` rejects it, and `ParseItemId` additionally rejects
  `itemId == 0`. Delegating would narrow accepted input, which FR-4.3 forbids.
  Task 18 Step 5 says this explicitly so a reviewer does not read the omission as
  an oversight.

- **`atlas-npc-shops`'s scaffolding block is longer than its peers** —
  `ParseInput` starts at line 60 and the first path helper at line 111, versus
  ~47/~98 elsewhere. Task 15 tells the implementer to read what lives above line
  111 before deleting.

## Deliberately oversized tasks

`plan-lint.sh` F4 warns on tasks touching more than ~6 files. Two are over, and
neither can be split:

- **Task 19 (`atlas-mts`, 7 files)** and **Task 20 (`atlas-character`, 8 files)**.
  Each service is a single Go module, and `rest/handler.go` must change together
  with every file that consumes it — the alias swap removes `DB()` and one curry
  level, so a half-converted module does not compile. Splitting either across two
  implementer contexts would hand the first implementer a module that cannot
  build or test, defeating the per-task verification gate. They are the two
  largest services (26 and 18 `d.DB()` sites) and go last in the ordering so the
  recipe is well-exercised before it meets them.

Task 3 touches two services (`atlas-fame`, `atlas-events`) but is a pure deletion
of two files with zero consumers, so the same-mechanical-change exception
applies.

`plan-lint.sh` also warns "Task 21 spans 2 services." That is a false positive
from the linter's path extractor: Task 21 edits only
`.claude/skills/.../scaffolding-checklist.md` and
`docs/architectural-improvements.md`. The two `services/` paths it sees are the
reference implementations the new checklist item points readers at
(`services/atlas-guilds/.../rest/handler.go` and
`services/atlas-configurations/.../environments/resource.go:45`) — cited, not
edited. The plan currently lints at **0 errors, 1 advisory warning**.

## Ordering dependencies

Only the ends are constrained:

- **Tasks 1–2 first** (`atlas-messages`, `atlas-mounts`) — pilots. Task 1 is a
  pure alias swap with no registration change at all (its `RegisterHandler` is
  already `(l)(si)`); Task 2 is the smallest end-to-end Shape-A case. If the
  recipe is wrong, it is cheapest to discover here.
- **Task 21 (docs) after every conversion** — its `RESOLVED` claim and its
  `grep -rl 'type HandlerDependency struct'` verification are only true once the
  sweep is complete.
- **Task 22 last** — the fleet-wide acceptance greps and the flagless
  `tools/verify.sh`.
- Tasks 4–20 are mutually independent (no shared code between services) and
  parallelize across fresh-context agents without conflict. Tasks 10 → 11 → 12
  (`map-actions`, `portal-actions`, `reactor-actions`) are near-identical copies;
  running them in order lets 11 and 12 copy 10's diff rather than re-derive it.

## What must reach the PR description

The two intended behavior deltas, stated as deltas and not as no-ops:

1. **`ParseEnvironment` joins the chain (FR-4.1).** These 21 are currently the
   only REST services in the fleet that ignore the `ENVIRONMENT` header. Note the
   PRD's own wording overstates this: `env.Reconcile` already runs in all 21
   today, because they already use `server.ParseTenant`. What changes is that a
   *non-empty* header env can now reach it. Header-absent requests are
   byte-identical. A header naming an unknown/DEACTIVATING/DELETED environment,
   or one disagreeing with the tenant's projection, now 400s instead of being
   silently served from baseline data. All 21 install a real `env.MapRegistry`
   via `service.WithEnvironmentRegistry`, so this is live, not theoretical.
2. **Malformed-body 400s gain a JSON:API errors document (FR-4.2).** Status
   unchanged at 400; body goes from empty to populated.
