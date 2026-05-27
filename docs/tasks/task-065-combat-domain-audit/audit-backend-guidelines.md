# Backend Audit — task-065 (combat domain, final post-PR sweep)

- **Worktree:** `.worktrees/task-065-combat-domain-audit`
- **Branch:** `task-065-combat-domain-audit`
- **Base:** `main`
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-05-27
- **Build:** PASS (`libs/atlas-packet`, `services/atlas-channel`, `tools/packet-audit`)
- **Tests:** PASS — `go test ./... -count=1` and `go test -race` clean in all three changed modules
- **Vet:** PASS — `go vet ./...` clean in all three modules
- **Overall:** PASS

## Scope

This audit covers the Go-only deltas on `task-065-combat-domain-audit` vs `main`. The branch is dominated by docs / audit-report fixtures; Go changes split into:

1. **`libs/atlas-packet`** (shared library — LIB-* / immutability discipline)
   - `monster/clientbound/control.go` (+ `control_test.go`) — **wire-change item 3**
   - `monster/clientbound/destroy.go` (+ `destroy_test.go`)
   - `drop/clientbound/destroy.go` (+ `destroy_test.go`)
   - `monster/clientbound/movement_test.go` (new test file only)
   - `pet/clientbound/movement_test.go` (new test file only)

2. **`services/atlas-channel`** (service code — DOM-* / SUB-*)
   - `socket/writer/monster_control.go` — constructor signature widened to thread `aggro` through

3. **`tools/packet-audit`** (dev tooling — pragmatic discipline)
   - `cmd/run.go`, `cmd/disambiguation_test.go`
   - `internal/atlaspacket/{registry.go,analyzer.go,analyzer_test.go,registry_test.go}` + testdata
   - `internal/diff/combat_flatten_test.go`
   - `internal/idasrc/{export.go,export_test.go}` + testdata

There are no new domain packages, no new services, no new REST handlers, no new Kafka topics, no DB-touching code, and no goroutines/channels added on this branch. The full DOM-* checklist therefore degenerates to LIB-* (immutability of shared-library models), SUB-* / call-site discipline for the single atlas-channel writer file, and a pragmatic-discipline pass over the tooling.

## Build & Test Results

```
$ cd libs/atlas-packet && go build ./... && go test ./... -count=1
ok ... (all packages green; full pass)
$ go test -race ./monster/clientbound/... ./drop/clientbound/... ./pet/clientbound/... -count=1
ok  monster/clientbound  1.024s
ok  drop/clientbound     1.016s
ok  pet/clientbound      1.019s
$ go vet ./...   # clean

$ cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... -count=1
ok ... (all packages green)
$ go test -race ./socket/... ./kafka/consumer/monster/... -count=1
ok  socket/handler             1.027s
ok  socket/model               1.024s
ok  kafka/consumer/monster     1.027s
$ go vet ./...   # clean

$ cd tools/packet-audit && go build ./... && go test ./... -count=1
ok  cmd                                 0.225s
ok  internal/atlaspacket                0.425s
ok  internal/diff                       0.108s
ok  internal/idasrc                     0.004s
ok  internal/csv,internal/report,template
$ go vet ./...   # clean
```

## LIB-* Checklist Results — `libs/atlas-packet/monster/clientbound/control.go` (item 3, wire change)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| LIB-01 | Struct fields all private (immutability) | PASS | `control.go:28-34` — `controlType`, `uniqueId`, `aggro`, `monsterId`, `monster` are all lowercase. |
| LIB-02 | No setter methods | PASS | No `func (m *Control) Set...` in `control.go`; only constructor + value-receiver getters. Grep `func.*Set[A-Z]` in the diff returns zero hits. |
| LIB-03 | Value-receiver getters | PASS | `control.go:53-58` — every accessor is `func (m Control)`, no pointer receivers. |
| LIB-04 | Constructor returns value (not pointer) | PASS | `control.go:43` — `func NewMonsterControl(...) Control` (value, not `*Control`). |
| LIB-05 | New field (`aggro bool`) threaded through both Encode and Decode for round-trip correctness | PASS | Encode `control.go:68-73` writes `byte(1)`/`byte(0)` guarded by `controlType > Reset`; Decode `control.go:86` reads `r.ReadByte() != 0` under the same guard. Round-trip pinned by `TestMonsterControlActiveInit`, `TestMonsterControlReset`, `TestMonsterControlActiveRequest` (`control_test.go:10-45`). |
| LIB-06 | Wire-byte semantic anchored by an explicit-byte test (not just round-trip) | PASS | `control_test.go:51-76` — `TestMonsterControlAggroByteReflectsState` asserts `out[5] == 0x01` for aggro=true and `0x00` for aggro=false. |
| LIB-07 | 5-variant round-trip baseline | PASS | All three round-trip tests iterate `test.Variants`. Pet/monster movement test files (`pet/clientbound/movement_test.go:11-18`, `monster/clientbound/movement_test.go:10-28`) also added the 5-variant baseline. |
| LIB-08 | No package-level mutable state introduced | PASS | `control.go` adds no `var` outside the existing `ControlType` const block (unchanged). |
| LIB-09 | Reset path does not emit aggro byte | PASS | `control.go:68` guards `if m.controlType > ControlTypeReset`; `StopControlMonsterBody` passes `false` purely for clarity (`monster_control.go:35`). |
| LIB-10 | Constructor signature back-compat verified at every call site | PASS | Grep `NewMonsterControl` returns 5 hits — 1 production caller (`services/atlas-channel/atlas.com/channel/socket/writer/monster_control.go:52`), 4 test callers (all in `control_test.go`). The single production caller (`socket/writer/monster_control.go`) was updated in the same commit. |

### Wire-semantic sanity check for item 3 (the byte(5) -> byte(1)/byte(0) decision)

The commit message argues v95's `CMobPool::OnMobControl` treats the post-mobId byte as a non-zero flag (any non-zero ≡ "controller has aggro responsibility"). I verified the in-tree comment chain documents this reasoning:

- `libs/atlas-packet/monster/clientbound/control.go:36-42` — the constructor comment cites task-065 post-phase-b.md item 3 and states v95 reads the byte as a non-zero flag.
- `services/atlas-channel/atlas.com/channel/monster/model.go:57-64` — `ControllerHasAggro()` documents the v83 protocol-compat semantic ("the field is wire-named `useSkills`").
- The Decode path symmetrically reads `r.ReadByte() != 0` — so the round-trip never collapses, but the explicit wire test at `control_test.go:51-76` is what fixes the byte value rather than relying on round-trip alone.

I do NOT independently verify the IDA disassembly here; that's covered by the `docs/packets/audits/gms_v95/MonsterControl.{md,json}` artifacts on this branch (out-of-scope for the backend-dev pass). The in-code reasoning is internally consistent, the round-trip is symmetric, and a regression to byte(5) would now fail `TestMonsterControlAggroByteReflectsState`.

## LIB-* Checklist Results — other touched lib files

| File | Status | Notes |
|------|--------|-------|
| `libs/atlas-packet/monster/clientbound/destroy.go` | PASS | Private fields (`destroy.go:26-30`), value-receiver getters (`:47-53`), constructor returns value (`:32,:39`), guarded swallow field (`:60-62` encode / `:71-73` decode). `TestMonsterDestroyBySwallow` (`destroy_test.go:20-42`) covers 5 variants + explicit 9-byte wire length + 5-byte regression for non-swallow. |
| `libs/atlas-packet/drop/clientbound/destroy.go` | PASS | Private fields with comment-anchored wire shape (`destroy.go:32-43`), conditional wire emission by destroyType (`:85-93`), legacy constructor preserved for back-compat (`:51-57`), new `NewDropDestroyExplode` correctly takes `int16` (matching v95 IDA cited in the comment `:59-62`). Tests pin wire lengths 7/13 for the explode/pet-pickup paths (`destroy_test.go:34-68`). |
| `libs/atlas-packet/monster/clientbound/movement_test.go` (new) | PASS | 5-variant round-trip baseline only; no production-code changes. |
| `libs/atlas-packet/pet/clientbound/movement_test.go` (new) | PASS | 5-variant round-trip baseline only; no production-code changes. |

## DOM-* Checklist Results — `services/atlas-channel/atlas.com/channel/socket/writer/monster_control.go`

`atlas-channel` is the only Go service touched on this branch. The single touched file is a packet-encoder helper, not a domain package — it has no `model.go`/`processor.go`/`administrator.go`/`provider.go`. The full DOM checklist is therefore not applicable; the relevant subset is:

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Logger param is `logrus.FieldLogger` | PASS | `monster_control.go:39` — `func(l logrus.FieldLogger, ctx context.Context) ...`. No `*logrus.Logger`. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | Grep on the diff yields zero matches. |
| DOM-13 | Writer does not call other domains directly | PASS | `monster_control.go` imports `atlas-channel/data/map` (snap helper) and `atlas-channel/monster` (model) only; no cross-domain processor calls. |
| DOM-15 | No direct DB writes from this layer | PASS | No `db.Create/Save/Delete` in the file. |
| DOM-21 | No duplication of `libs/atlas-constants` types | PASS | The file declares `ControlMonsterType` (int8) and its 8 enum values (`monster_control.go:15-24`) — but these mirror the protocol constants on the lib side (`libs/atlas-packet/monster/clientbound/control.go:17-26`) and are pre-existing (not added on this branch). `atlas-constants` does not define a `MonsterControlType`; this is a packet-protocol concern, not a shared domain id. |
| Currying preserved | Curried `Announce` pattern intact | PASS | All 5 call sites in `kafka/consumer/{monster,map}/consumer.go` still chain `session.Announce(l)(ctx)(wp)(monsterpkt.MonsterControlWriter)(writer.Start...)` — the new `aggro` parameter widens the inner writer call, not the curry chain. |
| Aggro source is real | `aggro` threaded from event/model | PASS | `consumer.go:150,300,321,343` and `kafka/consumer/map/consumer.go:467` pass either `m.ControllerHasAggro()` (the model getter at `monster/model.go:63`) or `e.Body.ControllerHasAggro` (event payload). No callers pass a hardcoded literal `true`/`false` except `StopControlMonsterBody` (Reset never emits the byte — see LIB-09). |
| No new setter on monster model | PASS | The branch adds zero changes to `services/atlas-channel/atlas.com/channel/monster/model.go` (verified via `git diff --stat main..HEAD -- services/atlas-channel/atlas.com/channel/monster/`). `ControllerHasAggro()` is a value-receiver getter; `controllerHasAggro` already existed in the model. |
| No new aggregate / processor / administrator surface | N/A | No domain-orchestration code introduced. |

## Tooling (`tools/packet-audit`) — pragmatic-discipline pass

Tooling lives outside the service plane, so DOM-* / SUB-* do not strictly apply. I scanned for cross-cutting smells: package-level mutable state, panics, TODO/FIXME, swallowed errors, surprise globals.

| Concern | Status | Evidence |
|---------|--------|----------|
| No new `TODO` / `FIXME` markers | PASS | `git diff main..HEAD` over the tooling dirs returns zero added TODO/FIXME lines. |
| No `panic(` / `os.Exit(` introduced | PASS | No added `panic(` lines in the diff. |
| Errors returned not swallowed | PASS | `registry.go:60-103` ignores parse errors during AST walks (`return nil // ignore broken files`) — documented intent. `export.go:115` / `:132` return wrapped errors with call-position context. `Resolve` (`:83`) and `resolveWithVisited` (`:94`) propagate errors up. |
| Cycle detection on Delegate | PASS | `export.go:94-103` maintains a `visited` set keyed on fname, returns `Delegate cycle through ...` if the same fname appears twice on the active descent path. `defer delete(visited, fname)` allows diamond patterns (correct — comment at `:91-93` explains why). |
| Guard scope on Delegate inlining | PASS | `export.go:122-127` AND-s the outer Delegate's guard onto each spliced call's existing guard via `combineGuards` (`:141-152`), which omits empty parts to avoid `() && (x)` noise. |
| Item 4 — qualified registry keys preserve back-compat | PASS | `registry.go:169-178` keeps the short-name lookup path for callers that don't know `pkgPath`; ambiguous short names return `(nil, false)` (correct — caller must disambiguate). `Qualify(hint, contextPkg)` prefers same-pkg matches (`:240-247`) and preserves the `::EncodeForeign` suffix across qualification (`:226-230`). |
| Item 5 — wire-mutex collapse safety | PASS | `analyzer.go:281-330` peeks both branches via a `scratchWalk` (fresh stack/out, shared read-only registry & range/field maps) and only collapses when shapes match Kind+Op+RecurseType across all positions. Divergent shapes fall through to the standard per-branch walk with guards intact — comments at `:264-280` document the mutex / non-mutex cases. |
| Item 6 — dispatcher annotation list is narrow + tested | PASS | `export.go:172-191` enumerates exactly 3 dispatchers (`per-mob`, `per-pet`, `per-pet-remote`) with a comment requiring a matching test for any addition (`:170-171`). `export_test.go` covers each case. Unknown kinds return nil — forward-compat. |
| Item 7 — `AssignStmt` walking does not break Encode discipline | PASS | `analyzer.go:423-432` walks RHS of an assignment so Decode methods (`m.field = r.ReadByte()`) are analyzable; encoder bodies don't use this pattern, so there's no risk of double-counting. The case `*ast.AssignStmt` is correctly placed alongside `*ast.ExprStmt` rather than inside the default `ast.Inspect` fallback (which would have re-walked nested calls). |
| Item 8 — Delegate splice does not flatten dispatcher prefixes | PASS | Each delegate target re-enters `resolveWithVisited` (`export.go:117`), which prepends its OWN `dispatcherPrefix(...)` to its sub.Calls before returning — so prefix bytes ride along with the spliced sub. Correct semantically: when A delegates to B and B is `per-mob`, B's mobId Decode4 ends up in A's resolved Calls. |
| Concurrency safety of tooling | N/A | The tooling is single-threaded CLI; no new goroutines, channels, or shared mutable state. |
| Determinism of TypeRegistry walk | NOTE (not a fail) | `filepath.WalkDir` is lexicographic per directory entry in Go ≥1.16, so the registry walk is already deterministic. The new `byShort` index appends multiple qualified keys per short name (`registry.go:98`) — when ambiguous, `Calls(typeName)` returns `(nil, false)` and the analyzer falls back to the legacy variable-name behavior. This is correct and documented at `registry.go:166-178`. |

## SEC-* Checklist

Not applicable — no auth/token/redirect code on this branch.

## Other discipline checks

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Hardcoded secrets | PASS | No added literals matching key/password/secret patterns in the Go diff. |
| `*logrus.Logger` in any signature | PASS | Grep on the diff returns zero hits — all logger params are `logrus.FieldLogger`. |
| `logrus.StandardLogger()` usage | PASS | Zero matches in the diff. |
| Goroutine + tenant-context discipline | N/A | No new goroutines on this branch. |
| Kafka producer stubbing (DOM-24) | N/A | The new tests do not emit Kafka messages — they round-trip / wire-length test only. No `AndEmit` / `producer.Produce` / `message.Emit` calls added in any `_test.go` on this branch. |
| Dockerfile mentions (DOM-22) | N/A | No new `libs/atlas-*` direct require on any service's `go.mod` (`git diff main..HEAD -- services/*/atlas.com/*/go.mod` yields no result). The shared lib changes are to existing `libs/atlas-packet`, which is already wired everywhere. |
| Kafka topic configmap (DOM-23) | N/A | No new `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` constants added on this branch. |
| Scaffolding checklist (SCAFFOLD-*) | N/A | No new service added; no new atlas-channel `Writer`/`Handler` constants added; no new `libs/atlas-packet/character/{clientbound,serverbound}/*` packages. The `MonsterControlWriter` constant `libs/atlas-packet/monster/clientbound/control.go:13` is unchanged. |

## Notes / Observations (non-blocking)

1. **`NewDropDestroyExplode` not yet called from production.** `libs/atlas-packet/drop/clientbound/destroy.go:62` is new but currently only exercised by tests. `services/atlas-channel/atlas.com/channel/kafka/consumer/drop/consumer.go:132` still calls the legacy `NewDropDestroy(e.DropId, DropDestroyTypeExplode, 0, -1)` path, which the new constructor explicitly documents as emitting `int16(0)` for the explode delay. This is consistent with the comment at `destroy.go:50-57`. If a future caller needs a non-zero delay, the new constructor is ready. Not a guideline violation.

2. **`NewMonsterDestroyBySwallow` not yet called from production.** Same situation — the helper exists for the upcoming character-eater wiring, but no current `kafka/consumer/monster/consumer.go` path emits `DestroyTypeSwallow`. Documented in the comment at `libs/atlas-packet/monster/clientbound/destroy.go:36-39`.

3. **`ControlMonsterType` lives in both `libs/atlas-packet/monster/clientbound/control.go:15-26` and `services/atlas-channel/atlas.com/channel/socket/writer/monster_control.go:15-24`.** Both are pre-existing (not added on this branch); the service-side copy converts via `monsterpkt.ControlType(controlType)` at `monster_control.go:52`. This is the established atlas pattern (service mirrors the protocol enum to avoid an import cycle / leak of `atlas-packet` into the writer's public type surface). Not a regression.

4. **Dev-tooling test files use `runtime.Caller(0)`-based path resolution** (e.g. `combat_flatten_test.go:23-28`, `disambiguation_test.go:34-36`). This is the established pattern in `tools/packet-audit` and is appropriate for read-only AST-walking tests that need the live `libs/atlas-packet` tree on disk; no goroutines or t-globals involved.

## Summary

### Blocking (must fix)
None.

### Non-Blocking (should fix)
None.

### Overall verdict
PASS. The branch's only wire-shape change (item 3, `MonsterControl` aggro byte) is correctly anchored by an explicit-byte test, the constructor was widened safely (single production caller, immutability preserved, no setter introduced), and all other deltas are docs / dev-tooling / test scaffolding that meet the project's pragmatic-discipline bar. Build and test are clean, including `-race` on the touched packages.
