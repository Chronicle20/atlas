# Backend Audit — task-154 Dark Knight Berserk

- **Scope:** `services/atlas-buffs/atlas.com/buffs` (new `berserk/`, `external/{character,skills,effectivestats,dataskill}`, `kafka/message/{characterstatus,skillstatus}`, `kafka/consumer/{characterstatus,skillstatus}`, `tasks/berserk.go`, `character/maxhp.go`, mods to `character/processor.go`, `kafka/message/character/kafka.go`, `main.go`) and `services/atlas-channel/atlas.com/channel` (`kafka/message/buff/kafka.go`, `socket/handler/effects.go`, `kafka/consumer/buff/consumer.go`).
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-17
- **Build:** PASS (verified clean by controller prior to this audit; not re-run)
- **Tests:** PASS (`go test -race`/`vet` clean both modules per controller; `docker buildx bake atlas-buffs atlas-channel` built; `redis-key-guard.sh`/`goroutine-guard.sh` exit 0 — not re-run per instructions)
- **Overall:** NEEDS-WORK

## Structural note

`atlas-buffs` has no GORM/database dependency at all (`grep -n "gorm" services/atlas-buffs/go.mod` → no matches; `services/atlas-buffs/atlas.com/buffs/main.go` never calls `database.Connect`). It is Redis (`libs/atlas-redis`) + Kafka only, service-wide, not a task-154 invention. DOM checks that assume a GORM domain (DOM-02/03/11/16 — `ToEntity`/`Make`/provider/administrator) are **N/A** for `berserk` and are graded N/A below rather than FAIL, mirroring the pre-existing `character` package's Redis-registry shape (`character/registry.go`).

## Domain Checklist Results

### berserk (Redis-backed domain: `model.go` + `builder.go`, no `entity.go` — service-wide, N/A per above)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | PASS | `berserk/builder.go:20` `NewBuilder(...)`, fluent `SetChannel`/`SetCharacterLevel`/`SetDirtyAt`, `Build()` at `berserk/builder.go:44` |
| DOM-02 | ToEntity() | N/A | No GORM entity in this service (see Structural note) |
| DOM-03 | Make(Entity) | N/A | Same |
| DOM-04 | Transform function | N/A | `berserk` exposes no REST surface (no `rest.go`) |
| DOM-05 | TransformSlice | N/A | Same |
| DOM-06 | Processor accepts FieldLogger | PASS | `berserk/processor.go:46` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-07 | Handlers pass d.Logger() | N/A | No REST handlers in this package |
| DOM-08 | RegisterInputHandler for POST/PATCH | N/A | No REST routes |
| DOM-09 | Transform errors handled | N/A | No Transform calls |
| DOM-10 | Test DB tenant callbacks | N/A | No SQL DB; Redis tests use `miniredis.RunT` + `InitRegistry` (`berserk/registry_test.go:18-23`) |
| DOM-11 | Providers lazy-evaluated | N/A | No `provider.go`; Redis registry reads are direct (`registry.go:59-75`) |
| DOM-12 | No os.Getenv in handlers | PASS | `grep os.Getenv` in `berserk/` and `tasks/berserk.go` → no matches |
| DOM-13 | No cross-domain logic in handlers | N/A | No handlers |
| DOM-14 | Handlers don't call providers directly | N/A | No handlers |
| DOM-15 | No direct entity creation in handlers | N/A | No handlers, no entities |
| DOM-16 | administrator.go for writes | N/A | Redis writes go through `registry.go` (service-wide Redis-registry convention, not GORM) |
| DOM-17 | Domain error → HTTP status | N/A | No REST surface |
| DOM-18 | JSON:API interface on REST models | N/A | No `rest.go` in `berserk` itself |
| DOM-19 | Flat request models | N/A | No request models |
| DOM-20 | Table-driven tests | PASS | `berserk/evaluate_test.go:15-24` (`tests := []struct{...}` + loop); `character/maxhp_test.go:28-38` (`cases := []struct{...}` + `t.Run`) |
| DOM-21 | No duplication of atlas-constants types | PASS | `skill.DarkKnightBerserkId` used from `libs/atlas-constants/skill` (`berserk/processor.go:56`, confirmed defined at `libs/atlas-constants/skill/constants.go:3011`); `stat.TypeHp`/`stat.TypeMaxHp` from `libs/atlas-constants/stat` (`berserk/processor.go:114,118`); `constants.TemporaryStatTypeHyperBodyHP` from `libs/atlas-constants/character` (`character/maxhp.go:8,17`); `world.Id`/`channel.Id` from `libs/atlas-constants` throughout (`berserk/model.go:7-8`) |
| DOM-22 | Dockerfile 4-mentions per direct lib require | N/A | No go.mod/go.sum changes in this branch (`git diff` on both files is empty) — no new direct lib requires |
| DOM-23 | Kafka topic naming convention | PASS | `EVENT_TOPIC_CHARACTER_STATUS`, `EVENT_TOPIC_SKILL_STATUS`, `EVENT_TOPIC_CHARACTER_BUFF_STATUS` all present as `KEY: "KEY"` in `deploy/k8s/base/env-configmap.yaml:92,96,146` (COMMAND_TOPIC_CHARACTER/COMMAND_TOPIC_CHARACTER_BUFF at lines 19-20); `deploy/k8s/base/atlas-buffs.yaml` has no literal topic-value override |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS | `berserk/testmain_test.go:11`, `kafka/consumer/characterstatus/testmain_test.go:11`, `kafka/consumer/skillstatus/testmain_test.go:11` all call `producertest.InstallNoop()` in `TestMain`; `grep -rn "ResetInstance\|t.Cleanup"` across all three packages → no matches (no premature un-stub) |
| DOM-25 | Client wire values config-resolved | PASS (no new packet work) | `libs/atlas-packet/character/effect_body.go` is untouched by this branch (`git diff --stat -- libs/atlas-packet` empty). The mode byte is already resolved via `atlas_packet.ResolveCode(l, options, "operations", ...)` (`effect_body.go:65,77`, pre-existing). The `active`/`darkForceEffect` bool threaded through `CharacterSkillUseEffectBody` (`socket/handler/effects.go:50`) is a semantic derived flag (compared against `skill.DarkKnightBerserkId`, an atlas-constants Id), not a client lookup-table byte — matches the pre-existing `isDragonFury`/`isMonsterMagnet` pattern. Domain service (atlas-buffs) emits a semantic `Active bool` + `SkillId`, not a raw wire code (`kafka/message/character/kafka.go:102-109`) |
| DOM-26 | Goroutines via routine.Go | PASS | `berserk/processor.go:239` `routine.Go(l, ctx, func(_ context.Context) {...})`; `tasks/task.go:19` same; `main.go:70,73,76` same. `grep -rnE '^\s*go (func|[A-Za-z_])'` on non-test files in scope → no matches (bare `go` only appears in `berserk/registry_test.go` test helper, which is excluded) |
| DOM-27 | Transient DB errors → 503 | N/A | No DB in this service |
| DOM-28 | No silent degradation in decorators | N/A | No `model.Decorator` implementations added by this feature |

### File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | PASS | `berserk/processor.go` holds `type Processor interface` (line 25) + `NewProcessor` (line 46) + all `ProcessorImpl` methods |
| FILE-02 | RestModel/Transform in rest.go | PASS | `external/{character,skills,effectivestats,dataskill}/rest.go` each hold their `RestModel` + JSON:API methods; `berserk` has no rest.go (no REST surface, N/A) |
| FILE-03 | Cross-service request funcs in requests.go | PASS | `external/{character,skills,effectivestats,dataskill}/requests.go` each hold `getBaseRequest()` + `requests.GetRequest[RestModel]` calls |
| FILE-04 | Entity+Migration+TableName in entity.go | N/A | No GORM entities in this service |
| FILE-05 | Builder/Model/administrator/provider/state.go placement | PASS | `berserk/builder.go` (Builder), `berserk/model.go` (Model) — placed correctly; no administrator.go/provider.go/state.go needed (Redis-only domain) |
| FILE-06 | No package-named catch-all file | PASS | No `berserk.go`/`character.go` file bundling ≥2 responsibilities; `tasks/berserk.go` is a single-purpose `Task` implementation matching sibling `tasks/poison.go`/`tasks/expiration.go` |

### External HTTP Client Checklist (`external/character`, `external/skills`, `external/effectivestats`, `external/dataskill`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | JSON:API target implements relationship interfaces | **FAIL** (×4) | None of the four new `RestModel`s implement `SetToOneReferenceID`/`SetToManyReferenceIDs`: `external/character/rest.go:1-28`, `external/skills/rest.go:1-26`, `external/effectivestats/rest.go:1-21`, `external/dataskill/rest.go:1-32`. `libs/atlas-rest/CLAUDE.md:23-25` documents this as mandatory boilerplate "even when the caller doesn't care about the relationship payload," citing task-037 biting this exact gap twice. Currently latent (none of the four upstream REST models — `atlas-character/character/rest.go`, `atlas-skills/skill/rest.go`, `atlas-effective-stats` stat rest, `atlas-data` skill rest — implement `GetReferences` today, so no `relationships` block is emitted yet), but the guideline requires the stub defensively regardless of current upstream shape. |
| EXT-02 | httptest-backed integration test | **FAIL** (×4) | `find services/atlas-buffs/atlas.com/buffs/external -name "*_test.go"` → zero files. No `httptest.NewServer` anywhere in `external/`. Per `libs/atlas-rest/CLAUDE.md:32`, "The `FakeClient` mocks under `mock/` packages bypass the unmarshal path and won't catch this" — and there isn't even a FakeClient mock here; the unmarshal path for all four clients is completely untested. |
| EXT-03 | 404 vs other failures distinguished | PASS | `berserk/processor.go:56-65` distinguishes `requests.ErrNotFound` (skill never learned → level 0, not an error) from other errors (propagated). No blanket "treat every error as not-found" pattern found in `reevaluate` (`processor.go:193-228`, which propagates raw errors from `getCharacter`/`getMaxHp`/`getEffectX` without reclassifying them as not-found). |
| EXT-04 | Service URL via RootUrl, not hardcoded | PASS | `external/character/requests.go:15` `requests.RootUrl("CHARACTERS")`; `external/skills/requests.go:15` `requests.RootUrl("SKILLS")`; `external/effectivestats/requests.go:16` `requests.RootUrl("EFFECTIVE_STATS")`; `external/dataskill/requests.go:15` `requests.RootUrl("DATA")` |

### atlas-channel changes

| Area | Status | Evidence |
|------|--------|----------|
| Kafka message mirror (`kafka/message/buff/kafka.go`) | PASS | `BerserkStatusEventBody` added at lines 76-91 mirroring atlas-buffs' producer shape; golden decode test `kafka_test.go:20-40` cross-checks against the emit-side fixture (per its own comment, the emit-side twin lives in `atlas-buffs berserk/producer_test.go`) |
| Consumer wiring (`kafka/consumer/buff/consumer.go`) | PASS | `handleStatusEventBerserk` registered via curried `InitHandlers` (lines 52-56), follows sibling `handleStatusEventApplied`/`handleStatusEventExpired` pattern; tenant/world/channel guard via `sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.Body.ChannelId)` (line 147) |
| Packet translation (`socket/handler/effects.go`) | PASS | `AnnounceBerserkEffect`/`AnnounceForeignBerserkEffect` (lines 46-64) reuse the existing config-resolved `CharacterSkillUseEffectBody`/`ForeignBody` encoders; no new wire-byte literals introduced |
| No unit test for `handleStatusEventBerserk` | Accepted (per task brief) | No session-harness exists in this service for consumer-level testing; cross-service JSON contract is covered by the golden decode test instead. Not re-derived as a finding per instructions. |

## Anti-Pattern Findings

### 1. Cosmic source citations in code comments (CLAUDE.md: "No Cosmic citations in code comments")

**FAIL** — 7 occurrences, 2 with explicit `Character.java:NNNN` line citations:

- `berserk/model.go:11` — `// Broadcast cadence (Cosmic parity: Character.java:1867 — 5000ms delay, 3000ms`
- `berserk/evaluate.go:7` — `// Strict less-than is Cosmic parity (Character.java:1852): at exactly x% the`
- `berserk/evaluate_test.go:18` — test name string contains `Character.java:1852`
- `berserk/registry.go:209` — `// (Cosmic parity: every re-evaluation replaces the schedule, design D2).`
- `berserk/processor.go:128` — `// HandleTransfer covers MAP_CHANGED and CHANNEL_CHANGED (Cosmic re-checks on`
- `berserk/processor_test.go:162` — `// the schedule (Cosmic cancel-and-replace semantics); the broadcast claim`
- `kafka/consumer/characterstatus/consumer_test.go:106` — assertion message string `"Cosmic re-checks on transfer"`

CLAUDE.md is explicit: "No Cosmic citations in code comments — reference, not source of truth; cite IDA/WZ instead; scrub files you touch." These comments cite Cosmic (a decompiled/leaked GMS-derivative source, not IDA/WZ) as the behavioral authority, including two direct `Character.java:NNNN` line references. Should be rewritten to cite IDA verification or WZ data, or dropped to a design-doc reference only.

## Known Minor Findings — Triage

| # | Finding | Verdict | Reasoning |
|---|---------|---------|-----------|
| 1 | `Track` swallows tenant set-add error (`_ = r.tenants.Add(...)`, `berserk/registry.go:49`) | **Acceptable for merge** | Mirrors existing convention at `character/registry.go:108`. SAdd is idempotent — a missed add self-heals on the next `Track` call for that tenant (e.g. next login). Not a task-154-introduced regression; matches service-wide convention. |
| 2 | `StoreEvaluation` failure leaves `dirtyAt` cleared with nothing re-arming it (`berserk/processor.go:225-227`) | **Should-fix, non-blocking** | `ClaimReeval` clears `dirtyAt` as part of the atomic claim (`registry.go:170-185`) *before* `reevaluate()` runs; if the terminal `StoreEvaluation` write itself fails (e.g. transient Redis error) after all REST lookups already succeeded, the entry is left permanently non-dirty with no retry scheduled — unlike the `rearm(...)` closure used for the three lookup-failure paths (`processor.go:194-199`). In practice this self-heals via any external trigger (HP change, transfer, login, skill update all call `MarkDirty`/`Track`), so a Dark Knight actively playing recovers within seconds; an idle Dark Knight standing still with no stat changes could stay stuck on stale `Active` state indefinitely. Recommend wrapping the `StoreEvaluation` error path with the same `rearm` pattern in a follow-up. |
| 3 | `HandleStatChanged` issues 2 Redis WATCH txns per HP change for every character, including non-Dark-Knights (`berserk/processor.go:108-126`) | **Acceptable for merge** | `UpdateChannel`/`MarkDirty` both no-op on `ErrNotFound` (untracked characters), so this is a performance/Redis-load concern, not a correctness bug. Design-documented requirement (D8: every channel-bearing event refreshes routing). Given STAT_CHANGED is a normal-frequency event (not per-tick), this is unlikely to be a bottleneck at current scale; flag as a follow-up optimization target (e.g. gate on a fast tracked-set membership check) rather than a merge blocker. |
| 4 | `ExpireBuffs` hooks the berserk mark inside the `Emit` closure, unlike the other 4 call sites (`character/processor.go:180` vs. lines 71, 101, 125, 161) | **Should-fix, non-blocking** | Confirmed: `Apply`/`Cancel`/`CancelAll`/`CancelByStatTypes` all call `markBerserkDirtyOnMaxHpChange` *after* `message.Emit` returns successfully (`if err != nil { return err }` gate before the call). `ExpireBuffs` calls it *inside* the closure, before the buffer's Kafka flush and without an error gate — if a *later* character's `buf.Put` in the same batch fails and aborts the whole `Emit`, an *earlier* character's berserk dirty-mark has already been committed to Redis even though its corresponding EXPIRED event was never actually flushed to Kafka. The re-evaluation 2s later then reads stale effective-stats data (atlas-effective-stats never got the event to recompute max HP). Low likelihood (requires a `buf.Put` failure mid-batch) but a real correctness gap relative to the pattern the other 4 sites establish for exactly this reason. Recommend moving the call outside the closure to match. |
| 5 | `GetAll` swallows the Redis error and returns nil; `Registry` has no logger (`berserk/registry.go:68-75`, struct at `registry.go:23-26`) | **Should-fix, non-blocking** | Confirmed: `Registry` has no `l logrus.FieldLogger` field anywhere, and `GetAll`'s signature (`[]Model`, no error) makes a Redis outage during a scan tick indistinguishable from "no Dark Knights currently tracked" — with zero log output, since there's no logger to log to. This mirrors the pre-existing `character/registry.go:140` `GetCharacters` shape (same no-logger, same swallow), so it's a service-wide convention rather than a task-154-introduced regression. Still a genuine observability gap — a sustained Redis outage would silently disable the entire berserk feature (and by the same shape, the buff-expiration feature) with no operator-visible signal beyond "no ticks happening." Worth a follow-up ticket to thread a logger through `Registry` and log at Warn on swallowed errors in both `berserk` and `character` registries, but not blocking for this PR since it doesn't regress existing behavior. |

## Summary

### Blocking (must fix before merge)

- **EXT-01** (×4): `external/character/rest.go`, `external/skills/rest.go`, `external/effectivestats/rest.go`, `external/dataskill/rest.go` — none implement `SetToOneReferenceID`/`SetToManyReferenceIDs`. Documented mandatory boilerplate (`libs/atlas-rest/CLAUDE.md`), with a two-time historical precedent (task-037) for exactly this class of client shipping without it.
- **EXT-02** (×4): same four packages — zero `httptest`-backed integration tests for any new external client. The unmarshal path (where EXT-01's gap would actually surface as a runtime failure) is completely untested.
- **Cosmic citations**: 7 occurrences across `berserk/model.go:11`, `berserk/evaluate.go:7`, `berserk/evaluate_test.go:18`, `berserk/registry.go:209`, `berserk/processor.go:128`, `berserk/processor_test.go:162`, `kafka/consumer/characterstatus/consumer_test.go:106` — violates the explicit CLAUDE.md project rule; two are direct `Character.java:NNNN` line citations that should cite IDA/WZ verification instead.

### Non-Blocking (should fix)

- `berserk/processor.go:225-227` — `StoreEvaluation` failure doesn't re-arm `dirtyAt` (Known Minor #2).
- `character/processor.go:180` — `ExpireBuffs` marks berserk-dirty inside the `Emit` closure without an error gate, unlike the other 4 call sites (Known Minor #4).
- `berserk/registry.go:68-75` / `registry.go:23-26` — `GetAll` swallows Redis errors silently; no logger on `Registry` (Known Minor #5, mirrors pre-existing `character/registry.go` shape).
- `berserk/cache.go:19-42` — `EffectXCache` has no TTL/expiration (patterns-cache.md's Key Takeaways table lists TTL as a standard requirement); justified in-comment as "immutable for tenant lifetime" but would silently serve stale effect thresholds if atlas-data's skill balance is ever hot-patched without an atlas-buffs restart.
- `berserk/processor.go:108-126` — `HandleStatChanged` issues 2 Redis WATCH txns per HP change for every character, tracked or not (Known Minor #3, performance follow-up).
- `berserk/registry.go:49` — `Track` swallows the tenant set-add error (Known Minor #1, self-healing, matches existing convention).

### Approved deviations (not findings, per task brief)

- `routine.Go` usage in `berserk/processor.go:239`, `main.go:70,73,76`, `tasks/task.go:19` — correct per CLAUDE.md mandate.
- `updateWithRetry` bounded-retry vs. single-attempt `ClaimReeval`/`ClaimBroadcast` — intended single-winner semantic (design D2), user-approved.
- Missing `continue` after re-eval branch in `ProcessTicks` (`berserk/processor.go:167-188`) — intentional broadcast-starvation fix per FR-5, user-approved.
- No unit test for `handleStatusEventBerserk` — no session harness exists in atlas-channel; covered by golden decode test instead.
- GM-hide on foreign broadcast absent — explicit PRD §9.1 follow-up, deliberate.

## Final Verdict

**NEEDS-WORK.** Build/tests/guards are clean (per controller). The four new external REST clients (`character`, `skills`, `effectivestats`, `dataskill`) ship without the mandatory JSON:API relationship-interface stubs (EXT-01) and without any httptest-backed integration coverage (EXT-02) — a documented, twice-bitten class of bug in this codebase (task-037). The Cosmic-citation comments violate an explicit CLAUDE.md project rule and should be scrubbed. Neither blocker is large: EXT-01 is ~2 lines × 4 files, EXT-02 is one httptest-based test per client package, and the Cosmic citations are a find-and-reword pass. The two Should-fix items (`ExpireBuffs` ordering, `StoreEvaluation` non-rearm) are real but low-likelihood correctness gaps worth a fast follow-up rather than a hard block.
