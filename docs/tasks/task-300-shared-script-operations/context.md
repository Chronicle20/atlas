# task-300 — Implementation Context

Companion to `plan.md`. Everything here was verified against the tree at
`31a791e3a` + the branch's design commit `9c145fb04`, inside
`.worktrees/task-300-shared-script-operations`.

---

## Key files

### The new library

| Path | Role |
|---|---|
| `libs/atlas-script-core/ops/ops.go` | `Step`, `Target`+builder, `Resolver`, `DirectResolver`, `ParamError`, param/range helpers, the overridable `now` clock |
| `libs/atlas-script-core/ops/message.go` | `SendMessage` |
| `libs/atlas-script-core/ops/monster.go` | `SpawnMonster` |
| `libs/atlas-script-core/ops/environment.go` | `MoveEnvironment`, `ResetEnvironment` |
| `libs/atlas-script-core/ops/effect.go` | `ShowIntro`, `ShowHint`, `PlayPortalSound`, `ApplyConsumableEffect` |
| `libs/atlas-script-core/ops/skill.go` | `CreateSkill`, `UpdateSkill` |
| `libs/atlas-script-core/ops/quest.go` | `StartQuest`, `StageClearAttemptPq`, `QuestDefaults` |
| `libs/atlas-script-core/ops/movement.go` | `WarpToPortal`, `WarpToSavedLocation`, `SaveLocation`, `StartInstanceTransport` |

One file per family, each small enough to hold in context whole.

### The four executors

| Service | Executor | Dispatch | Shared ops |
|---|---|---|---|
| map-actions | `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` (323L) | `switch` at `:37-58` | 5 |
| reactor-actions | `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go` (566L) | `switch` at `:49-90` | 5 |
| portal-actions | `services/atlas-portal-actions/atlas.com/portal/script/executor.go` (691L) + `script/optable.go` (104L) | `opTable` map at `optable.go:37-77` | 10 |
| npc-conversations | `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go` (2,738L) | `switch` inside `createStepForOperation` `:1225-2677` | 14 |

npc has **no** `move_environment`/`reset_environment` case — that pair is map+reactor
only. 5 + 5 + 10 + 14 = 34 call sites collapsing onto 16 implementations.

### Module roots (`go build` / `go test` cwd)

- `libs/atlas-script-core`
- `services/atlas-map-actions/atlas.com/map-actions`
- `services/atlas-reactor-actions/atlas.com/reactor`
- `services/atlas-portal-actions/atlas.com/portal`
- `services/atlas-npc-conversations/atlas.com/npc`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`

All six are already `use`d in `go.work` (`:20`, `:58`, `:71`, `:78`, `:84`, plus the
orchestrator's line).

---

## Decisions carried from design, and what they cost

### OQ-1 — `ops` is a sub-package of `atlas-script-core`, not a sibling module

Verified, not assumed:

- `libs/atlas-saga/go.mod` requires only `atlas-constants` and `google/uuid`. It does
  **not** require `atlas-script-core`, so the new edge creates no cycle.
- All four consuming services already require **both** libs with `replace` lines in
  place, so **no service `go.mod` changes at all**: map `go.mod:11-12,94,96`; reactor
  `go.mod:11-12,94,96`; portal `go.mod:12-13,101,105`; npc `go.mod:13-14,100,102`.
  Only `libs/atlas-script-core/go.mod` changes (Task 1 Step 1).

Cost accepted: a future consumer wanting only `condition`/`operation`/`context` also
pulls `atlas-saga` — one small module (`uuid` + `atlas-constants`).

### OQ-2 — `Step` is a new three-field type, not `saga.Step[any]`

`libs/atlas-saga` exposes step construction **only** through
`Builder.AddStep(id, status, action, payload)` (`builder.go:52`) — there is no
free-standing `Step[any]` constructor. Returning the library's own type would require
adding one to `atlas-saga`, which PRD §7 says to avoid. Confirmed by reading
`builder.go:19-67`.

`Step.AppendTo(b, id)` is what the three rule services use; npc uses the accessors
directly because its handler returns `(stepId, status, action, payload, error)`.

### OQ-3 — `SpawnMonster` instance rule

`Instance = Target.Field().Instance()` only when the effective map id equals
`Target.Field().MapId()`; otherwise `uuid.Nil`. Forced by
`deploy/seed/*/npc-conversations/npc/npc-1063017.json`, which spawns `9300346` into
`mapId: 910510202` and then warps the character there. Copying the current field's
instance unconditionally would address a field that does not exist.

### OQ-4 — the two-method `Resolver` is sufficient

`evaluateContextValueAsInt` (`operation_executor.go:179-208`) is exactly
`evaluateContextValue` followed by arithmetic-or-`Atoi`. Every shared npc case uses only
those two helpers. The one thing that is *not* value resolution — `start_quest`'s
fallback to `ctx.Context()["questId"]` and `ctx.NpcId()` — is a Redis read and stays in
the service, passed in as `ops.QuestDefaults` (Task 11).

### OQ-5 — `stage_clear_attempt` and `stage_clear_attempt_pq` are one action

`saga.StageClearAttemptPqPayload` (`payloads.go:1306-1309`) carries both `InstanceId`
(reactor) and `CharacterId` (npc); the orchestrator branches on which is set
(`handler.go:3717-3734`). FR-17 kept. Reactor's `getPqInstanceByCharacter` REST call
(`executor.go:559-566`) stays local and the resolved id is passed in.

---

## Two library-internal decisions the design left open

**`expiration` is read as a string, not through `Resolver.Int`.** The `"-1"` sentinel
must be compared before any numeric parse, and the value is 64-bit epoch milliseconds,
which does not fit the int-width range helpers. `decodeSkillParams` is therefore the one
place in `ops` that calls `strconv` on a param value, and Task 5 requires a comment
saying why. Everything else routes through the `Resolver` per FR-4.

**An overridable package clock (`var now = time.Now`).** `CreateSkill`/`UpdateSkill`
produce a `time.Time`, which is otherwise untestable field-by-field. Tests swap `now`
via a `t.Cleanup`-restoring helper. This is the smallest thing that makes the skill
payload assertions exact rather than tolerance-based.

**`WarpToPortal` leaves `Instance` as `uuid.Nil`.** `WarpToPortalPayload` has an
`Instance` field (`payloads.go:47`), but neither current caller populates it — portal's
`executeWarp` (`executor.go:163-178`) and npc's `warp_to_map`
(`operation_executor.go:1446-1453`) both omit it. Destination-field addressing is not
this task's problem; preserving `uuid.Nil` keeps §5.3's "portal unchanged" honest. Task 6
pins it with a test so a future change is deliberate.

---

## Behaviour changes, and the evidence they are safe

| Change | FR | Evidence |
|---|---|---|
| reactor `spawn_monster` bad `x`/`y`/`count` now errors instead of defaulting | FR-15 | design §7 sweep: all 110 seeded `monsterId`/`x`/`y` and 33 `count` values are plain integer literals |
| npc `warp_to_map` `mapId` becomes required | §5.3 | sweep: 3,377 `warp_to_map` + 594 `warp` ops, **all** carry `mapId` |
| npc `send_message` `messageType` becomes optional | FR-13 | relaxation; no seeded script omits it, so it is unused but harmless |
| npc `spawn_monster` `x`/`y` become optional | FR-15 | relaxation |
| map + npc `SpawnMonsterPayload.Instance` now populated | FR-16 | orchestrator uses it to select the field (`handler.go:2087-2124`); OQ-3 rule bounds it; Task 14 pins the seam |
| map `SpawnMonsterPayload.Team` now populated | FR-16 | orchestrator already reads `payload.Team` on the same call; map was sending the zero value |
| npc `create_skill`/`update_skill` honour `expiration` | §5.4 | additive; npc hard-coded now+365d at `:1561`,`:1605` |
| portal `level`/`masterLevel` widen from int8 to byte | §5.4 | additive; portal's `ParseInt(...,10,8)` at `:328` rejected 128-255 for a `byte` field |
| `drop_message` accepts `type` and `messageType` everywhere, with `"5"`/`"6"` mapping on both | FR-13 | sweep: no seeded script uses `type` or the numerics — reactor's mapping is currently dead in seed data, carried forward for out-of-repo tenant data |

The FR-20 sweep is **re-run** in Task 14 Step 5 and its output committed to
`sweep-result.md`, so the recorded result cannot go stale. A disagreement with design §7
is a stop-and-report, not a fix-forward.

---

## Task sizing notes

Fourteen tasks. Splits were drawn so a reviewer can reject one while approving its
neighbour, and so no implementer hits the 120-tool-call `PARTIAL` hand-back.

- **Tasks 1–6 (library) land before any service is touched.** The library is green on
  its own, so a failure in Tasks 7–11 is unambiguously a delegation bug.
- **Tasks 7–9 are independent of each other** once Task 6 lands. They may be dispatched
  in parallel if the controller wants — different modules, no shared files.
- **npc splits into Tasks 10 and 11** purely on size: 14 interleaved cases in a 2,738-line
  file is more edit surface than one implementer should carry. Task 10 lands the resolver
  adapter (which Task 11 consumes), so they are strictly ordered.

### Deliberately large: Task 12 (FR-11 shim removal)

**8 files, one service — above the ~6-file guideline.** Kept whole on purpose:

- The change is a mechanical import repoint. The deleted members are **type aliases**
  (`saga/model.go:10-84`, `:87-171`; `saga/builder.go:8,12`), so the identities are
  unchanged and the compiler catches every missed call site. There is no behaviour to
  assert and nothing a reviewer could accept in half.
- Splitting it would leave the module uncompilable between commits — you cannot delete
  the aliases in one commit and repoint the importers in another.
- The importer set is exhaustive and small: `conversation/operation_executor.go:9`,
  `conversation/processor.go:6`, and three test files
  (`operation_executor_test.go:5`, `operation_executor_petevolution_test.go:6`,
  `processor_rps_test.go:4`).

Every other task is at or under 6 files and within one service.

---

## Things that will bite

**`libs/atlas-script-core` has 8 `.go` files, not 2.** `plan-context.sh` reported the
directory as "2 files" (it counted `go.mod`/`go.sum`); the real contents are four
sub-packages: `condition/` (2), `context/` (2), `operation/` (2), `outcome/` (2). No
`ops` package exists yet.

**`DirectResolver.Int` is a superset of what map/reactor/portal do today.** Those
services used `strconv.ParseUint`/`ParseInt`/`Atoi` directly;
`context.EvaluateValueAsInt` (`arithmetic.go:91`) additionally accepts arithmetic
expressions like `"10 * 5"`. That is additive and intended. Negative literals still work:
`EvaluateArithmeticExpression` explicitly skips the subtraction branch when the string
has a leading `-` (`arithmetic.go:69`) and falls through to `strconv.Atoi`
(`arithmetic.go:85`). Task 1 pins this with a `"-1"` case.

**`kind` on `move_environment` is read raw, not through the resolver.** Both current
call sites pass `params["kind"]` straight to `field.ParseObjectKind`
(map `executor.go:271`, reactor `executor.go:302`). Keep it that way; a blank value
legitimately defaults to `ObjectKindEnvironment` (`constants.go:29-42`).

**Do not confuse `WarpToPortal` with `WarpToRandomPortal`.** `libs/atlas-saga` defines
both (`payloads.go:36-40` / `:42-51`; `model.go:117` / `:118`). npc's
`warp_to_random_portal` case (`operation_executor.go:1458+`) is **out of scope**.

**Portal's `optable.go` must not be edited.** `optable_test.go` is the FR-9 regression
detector and must pass unchanged. `init()` panics on an entry with `opClassUnset`
(`optable.go:96-100`), so a botched edit fails at process start, not at test time.

**Portal's executor tests need Redis.** `executor_test.go:26-36` stands up `miniredis`
and calls `action.InitRegistry` in `TestMain`, because `executeWarp` and
`executeWarpToSavedLocation` touch the package-global pending-action registry. npc's
tests do the same per-test via `miniredis.RunT` (`operation_executor_test.go:28-30`).

**No guard in `tools/` has a `_test.sh` sibling today.** Task 13's is the first.
`tools/verify.sh:848-866` already auto-discovers and runs `tools/*_test.sh` for changed
`tools/` scripts (`changed_tool_suites`), so no extra wiring is needed — but there is no
existing guard test to copy from. Use `tools/lib/analyzer-guard_test.sh` (80 lines) as
the harness style reference instead.

**`docs/TODO.md:292-294`'s line references are all stale.** `:292` points at
`script/executor.go:250,260` for "environment object manipulation", but
`move_environment`/`reset_environment` are implemented today at
`services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:284-350`. Task 13
re-points the two boss/monster entries at their real locations
(`executeWeakenAreaBoss` `:262`, `executeKillAllMonsters` `:354` — re-confirm after
Task 8 shifts line numbers) and retires the environment entry.

---

## Verification

- Per task: module-local `go build ./... && go test ./<pkg>/... && go vet ./<pkg>/...`
  from that task's module root. Repo-wide verification belongs to `task-verifier` in its
  own context, not to the implementer.
- Task 13 adds `./tools/script-ops-guard.sh` to the gate; it must exit 0 against the real
  tree once Tasks 7–11 land.
- Task 14 Step 6 runs the **flagless** `tools/verify.sh`. `--quick`/`--no-docker` skip
  the bake and `-race` and do not count as done.
- Code review is a separate gate from `verify.sh` and runs before the PR. The
  `SpawnMonster` change crosses a service boundary into `atlas-saga-orchestrator`, so
  Task 14's seam test is the "a test asserts the NEW contract" requirement, not optional
  polish.
