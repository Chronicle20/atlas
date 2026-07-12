# Monster Movement Local State (PS-3) — Context

Companion to `plan.md`. Summarizes key files, verified facts, decisions, and dependencies an implementer (or reviewer) needs without re-deriving them from the codebase.

## Task documents

- PRD: `docs/tasks/task-120-monster-move-local-state/prd.md`
- Design (authoritative spec): `docs/tasks/task-120-monster-move-local-state/design.md`
- Plan: `docs/tasks/task-120-monster-move-local-state/plan.md` (7 tasks: 4 atlas-channel, 2 atlas-monsters, 1 verification)

## Key files

### atlas-channel (`services/atlas-channel/atlas.com/channel/`)

| File | Role |
|---|---|
| `movement/processor.go:111-211` | `ForMonster` — the hot path being de-RESTed. Reads exactly: field identity (line 118), `Mp()` (125), `MonsterId()` (128/130), `ControllerHasAggro()` (144). |
| `movement/processor_test.go` | Existing `TestComputeAckMp_*`/`TestNarrowSkill_*` pin the ack math; new `TestResolveLiveMonster_*` land here. |
| `monster/status_mirror.go` | The `StatusMirror` precedent the new `LiveMirror` copies (singleton via `sync.Once`, `sync.RWMutex`, tenant→uniqueId nesting, `EvictTenant`). |
| `monster/inbox.go` | `NextSkillInbox` — second precedent; note it uses explicit `InitNextSkillInbox()`, while `StatusMirror`/`LiveMirror` use lazy `Get*()`. |
| `monster/builder.go` | Model builder — gains `SetControllerHasAggro` (the Model field exists at `model.go:34` but the builder can't set it today; `CloneModel` also silently drops it; no non-test `CloneModel` callers in this package). |
| `monster/model.go` | Getters used by `LiveEntryFromModel`: `Field()`, `MonsterId()`, `Mp()`, `MaxMp()`, `ControllerHasAggro()`. |
| `kafka/consumer/monster/consumer.go` | All `monster_status_event` handlers; gains mirror write paths. Existing seam precedent: `monsterStatSetBroadcaster` spy vars (line ~362). Consumer uses `kafka.LastOffset` (line 37) — restarted pods start with an empty mirror. |
| `kafka/consumer/monster/consumer_test.go` | Has `newTestTenant`/`newTestServer` helpers and direct-handler-invocation test pattern. |
| `kafka/message/monster/kafka.go` | Event bodies. `StatusEventMpChangedBody.MonsterMpAfter` already exists (line ~215). Gains the 3 new Reason constants (consumer-side docs only). |
| `monster/information/{processor,rest,requests}.go` | Uncached REST fetch to atlas-data being fronted by the TTL cache. Only non-test call site: `movement/processor.go:128`. |
| `main.go:287` | `listener.RegisterEvictor` block — add `GetLiveMirror().EvictTenant(tid)` + `monsterinfo.EvictTenant(tid)`. |
| `main.go:330-338` | REST server chain — add `MountHandler("/metrics", promhttp.Handler())`. Mounts under `SetBasePath("/api/")` ⇒ endpoint is **`/api/metrics`** (same gotcha family as the `/api/readyz` probe-path bug). |

### atlas-monsters (`services/atlas-monsters/atlas.com/monsters/`)

| File | Role |
|---|---|
| `monster/processor.go` | `UseSkill` (585, deduct at 626-633), `UseBasicAttack` (771, deduct at 822-828), `DrainMp` (~1439-1490, already emits MP_CHANGED). `emit` is an injectable struct field (`processor.go:63,85`); `testInformationLookup` seam at line 68; a parallel `testMobSkillLookup` seam is added for UseSkill tests. |
| `monster/producer.go:124-139` | `mpChangedStatusEventProvider(m, characterId, skillId, reason, amount)` — reused verbatim for all three new emissions; `MonsterMpAfter` comes from `m.Mp()` of the post-mutation model. |
| `monster/kafka.go:36` | `MpChangeReasonMpEater` const block — gains `SKILL_CAST`/`BASIC_ATTACK`/`RECOVERY`. |
| `monster/recovery_task.go` | 10s tick; `ApplyRecovery` returns `(Model, hpApplied, mpApplied, error)` (`registry.go:497`) and `Run()` currently discards `mpApplied` (line ~113). Seam family: `infoFn`/`applyFn`/`emitFn`; gains `mpEmitFn`. |
| `monster/recovery_task_test.go` | Struct-literal test construction. **The first test (~line 33) uses the real `applyFn: r.ApplyRecovery` and WILL hit the new `mpEmitFn` — it must be updated or it nil-panics.** |
| `monster/registry.go:604` | `DeductMp(t, uniqueId, amount) (Model, error)` — returns post-deduct model (currently discarded at both new emission sites). |
| `monster/information/{cache,processor,metrics}.go` | task-060 Redis-backed cache — the semantic template (env parsing, `requests.ErrNotFound` classification, negative TTL, metrics naming) for the channel's memory-backed cache. |

## Decisions (resolved in design; do not relitigate)

1. **Initial MP at CREATED (OQ1):** seed the mirror from the REST fetch `handleStatusEventCreated` already performs (`consumer.go:130`). No CREATED-body enrichment, no first-move penalty.
2. **MP_CHANGED coverage (OQ2):** three verified gaps (skill-cast deduct, basic-attack deduct, recovery regen) are closed additively in atlas-monsters. Without this, mirror MP decays monotonically and the client mob brain stops proposing conMP attacks — that would be behavior drift.
3. **Staleness sweep (OQ3):** ticker every 5m, evict entries with `LastWrite` older than 30m. Constants, not env. Touch-on-read rejected (write on the hot read path).
4. **Events never create mirror entries** — `UpdateMp`/`UpdateAggro` are update-only. A partial entry with defaulted-false aggro would render the mob idle on the wire (`useSkills=false`). The movement fallback is the only entry creator besides the CREATED seed.
5. **In-process only** — no Redis for either cache (explicit PRD user decision); no shared TTL-cache lib (YAGNI until a third consumer).
6. **HP is not mirrored** — DAMAGED events carry deltas, not absolutes; PS-1 can extend the event surface later. `MaxMp` IS mirrored (free at seed, needed by PS-1 `DrainMp` pre-screen).
7. **Field-consistency rejection returns nil** — today's `return err` at `processor.go:120` is always nil there (err was from a successful GetById). Preserving that is deliberate; "fixing" it to non-nil is behavior drift.
8. **Amount semantics on new MP_CHANGED events:** deduct sites emit the requested cost (exact — the MP-sufficiency gate guarantees no clamp); recovery emits the pre/post delta best-effort. Consumers only read `MonsterMpAfter`.

## Dependencies & ordering

- Task 1 (LiveMirror) must land before Tasks 2 and 3 (both consume it). Task 4 (info cache) is independent of 2/3 but shares the main.go evictor block with Task 1. Task 5 defines `MpChangeReasonRecovery` used by Task 6. Task 7 gates completion.
- New Go dependency for atlas-channel: `github.com/prometheus/client_golang v1.23.2` (version matched to atlas-monsters). No new shared lib ⇒ no Dockerfile/go.work edits expected.
- Deploy-order free (mixed-version safe): old channel + new monsters ⇒ unknown Reasons hit the MP_CHANGED `default:` debug branch; new channel + old monsters ⇒ mirror MP lags no worse than today's post-command REST read and self-corrects via fallback/sweep.

## Test-environment facts (verified, will bite if ignored)

- `newTestTenant` already exists in the channel `monster` package (`inbox_test.go:11`) — redefining it in `live_mirror_test.go` is a compile error.
- The channel consumer handlers make REST calls to atlas-maps that fail fast and only log in tests; every mirror mutation is placed before or independent of those calls, so handler tests assert mirror state directly.
- atlas-monsters `ProcessorImpl` test literals often omit `emit` — any code path that newly dereferences `p.emit` (both deduct emissions) requires updating the literals that reach it (`TestUseBasicAttack_HappyPath_DeductsMpAndRegistersCooldown` at `processor_test.go:1481`).
- UseSkill's animation-delay lookup (`information.GetById`, not behind `testInformationLookup`) does a live HTTP attempt in tests; it fails fast, `animDelay=0`, execution is synchronous. Existing tests already tolerate this.
- Skill id 126 (Slow) maps to `SkillCategoryDebuff` (`libs/atlas-constants/monster/skill.go`); with `inFieldFn` injected to return no targets, `UseSkill`'s executor is a no-op — isolates deduct+emit.
- The singleton `LiveMirror`/info-cache are shared across tests in a package run; per-test isolation comes from fresh tenants (`tenant.Create(uuid.New(), ...)`), plus `resetInfoCache()` for env-sensitive cache tests.
- Go workspace: run `go test/vet/build` from the module dirs; `go mod tidy` may need `GOWORK=off`.

## Verification gate (CLAUDE.md — mandatory before "done")

`go test -race ./...`, `go vet ./...`, `go build ./...` in BOTH modules; `docker buildx bake atlas-channel atlas-monsters` from the worktree root; `tools/redis-key-guard.sh` from the worktree root. Then `superpowers:requesting-code-review` BEFORE any PR.
