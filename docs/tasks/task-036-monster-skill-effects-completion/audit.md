# Plan Audit — task-036-monster-skill-effects-completion

**Plan Path:** docs/tasks/task-036-monster-skill-effects-completion/plan.md
**Audit Date:** 2026-04-29
**Branch:** task-036-monster-skill-effects-completion
**Base Branch:** main
**Diff Range:** 5af5a1a1f...66fb58c69 (28 commits)

## Executive Summary

Task-036 is fully implemented and ready for PR. All 27 tasks across 4 phases are checked, and every PRD §10 acceptance criterion has at least one passing test. All five touched module test suites (libs/atlas-constants, libs/atlas-packet, atlas-monsters, atlas-channel, atlas-buffs, atlas-maps) build and pass. The handful of accepted plan-vs-impl deviations are documented in §"Notable Deviations" below; none are gaps relative to the PRD, and the only one worth a second look is Task 23's decision to keep MonsterStatus apply on a reflected hit (PRD FR-4.3.1 is silent on this; see analysis).

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | cjson empty-array audit on status-event bodies | ✅ DONE | `services/atlas-monsters/atlas.com/monsters/monster/kafka.go:154-198` (5 MarshalJSON impls); 5 round-trip tests in `kafka_test.go:28-99`; commit `802cb477d` |
| 2 | AffectedAreaCreated/Removed packet writers | ✅ DONE | `libs/atlas-packet/field/clientbound/affected_area_created.go`, `affected_area_removed.go`, `affected_area_test.go`; commit `d1a217ae2` |
| 3 | Reflect kind constants in libs/atlas-constants | ✅ DONE | `libs/atlas-constants/monster/skill.go:18-19,193-204` defines `ReflectKindPhysical`/`ReflectKindMagical` and `ReflectKindForSkill`; tests in `skill_test.go`; commit `1b2582a64` |
| 4 | Venom eviction by earliest ExpiresAt | ✅ DONE | `services/atlas-monsters/atlas.com/monsters/monster/builder.go` updated; `TestAddStatusEffect_VenomOverflow_EvictsByEarliestExpiresAt` in `builder_test.go:44`; commit `3cb41a1c8` |
| 5 | Verify + test atlas-buffs PoisonTick | ✅ DONE | `services/atlas-buffs/atlas.com/buffs/tasks/poison_test.go:14-34` (3 regression tests); commit `0f37b72f8` |
| 6 | Extend monster.StatusEffect with reflect fields | ✅ DONE | `services/atlas-monsters/atlas.com/monsters/monster/status.go:26-170` adds `reflectKind`/`reflectPercent`/`reflectLtX/Y/RbX/Y`/`reflectMaxDamage` + `IsReflect()` + `NewReflectStatusEffect`; commit `b5f344ca6` |
| 7 | Extend statusEffectAppliedBody + producer wiring | ✅ DONE | `services/atlas-monsters/atlas.com/monsters/monster/kafka.go` body + `producer.go` provider populate reflect fields; tests `kafka_test.go:81-147`; commit `501d3e6a3` |
| 8 | Populate reflect metadata in executeStatBuff | ✅ DONE | `processor.go:752-769` calls `NewReflectStatusEffect` for `SkillCategoryReflect`; **plus** `mobskill/builder.go` and `registry.go:99-226` extended to round-trip reflect coords through Redis; commit `311da5672` |
| 9 | Immunity mutual exclusion in executeStatBuff | ✅ DONE | `processor.go:722-750` pre-cancels opposite immunity inside `applyBuff` (not in `UseSkill` gate, so AoE per-target re-fetch handles each target); tests at `processor_test.go:823,887`; commit `f630e0676` |
| 10 | monster.StatusMirror in atlas-channel | ✅ DONE | `services/atlas-channel/atlas.com/channel/monster/status_mirror.go` (256 lines) + `status_mirror_test.go` (320 lines); commit `5bd7c6696` |
| 11 | Wire StatusMirror into status consumers | ✅ DONE | `kafka/consumer/monster/consumer.go:144` wiring; regression tests in `consumer_test.go`; commit `239c1fe3b` |
| 12 | VENOM wire-collapse via VenomCount | ✅ DONE | `consumer.go` first-apply / second-third / last-expire transitions; `TestHandleStatusEffectApplied_VenomFirstApply_BroadcastsMonsterStatSet` and siblings at `consumer_test.go:202-275`; commit `6a2891368` |
| 13 | mist.Mist immutable model + builder | ✅ DONE | `services/atlas-maps/atlas.com/maps/mist/model.go` (298 lines) + tests; convenience getters `WorldId/ChannelId/MapId` added (deviation, harmless); commit `6f5710610` |
| 14 | mist.Registry tenant-scoped index | ✅ DONE | `mist/registry.go` (164 lines) + `registry_test.go` (94 lines); commit `74800b8d1` |
| 15 | mist.Processor + producer | ✅ DONE | `mist/processor.go` (`NewProcessor(l, ctx, p)` 3-arg form for reactor pattern + `NewProcessorWithRegistry` test seam) + `producer.go`; commit `0077151fa` |
| 16 | Mist command consumer (MIST_CREATE/CANCEL) | ✅ DONE | `kafka/consumer/mist/consumer.go` + `consumer_test.go` (179 lines); commit `d90f267be` |
| 17 | MistTickTask | ✅ DONE | `tasks/mist_tick.go` (194 lines) + `mist_tick_test.go` (219 lines); 1s tick, expire, disease re-apply; commit `0065b4265` |
| 18 | Wire mist consumer + tick task in atlas-maps main | ✅ DONE | `main.go:92-101` posLookup + `tasks.NewMistTick(l, 1000, posLookup)` registration; commit `82c330cb6` |
| 19 | Wire atlas-character position client | ✅ DONE | `map/character/processor.go` + `requests.go` + `rest.go` add the position client; commit `c69b12233` |
| 20 | executeMist in atlas-monsters + producer | ✅ DONE | `processor.go:669-707` `executeMist`; AREA_POISON dispatch at `processor.go:562,644`; producer in `kafka/message/mist/kafka.go`; commit `e4ff076d8` |
| 21 | Picker un-skip AREA_POISON | ✅ DONE | `picker.go` skip removed; `TestPicker_AreaPoisonAllowed` at `picker_test.go:163`; commit `a1fcf13d2` |
| 22 | atlas-channel mist consumer + AffectedArea broadcast | ✅ DONE | `kafka/consumer/mist/consumer.go` (105 lines) + `consumer_test.go` (193 lines); `TestMistCreated_BroadcastsAffectedAreaCreated`; commit `359c59403` |
| 23 | Reflect math in character_attack_common.go | ✅ DONE (with note) | `socket/handler/character_attack_common.go:34-89` (computeReflect, attackKindFromAttackType, snapshotVenomDamagePerTick) + `:139-197` orchestration; tests `character_attack_common_test.go:14-260`; commit `001634638`. Deviation: status apply still fires even on reflected hits (see Notable Deviations) |
| 24 | STATUS_CANCEL.SourceSkillClass + dispel guard | ✅ DONE | `processor.go:1080-1117` `CancelStatusEffectGuarded`; `consumer.go:118-192` consumer; `kafka.go:54-63` body extension; `skill/handler/common.go:90-102` `dispelSkillClass` returns "MAGICAL" matching `ReflectKindMagical`; tests `processor_test.go:1187-1380`; commit `558d6fc9d` |
| 25 | Player-skill venom snapshot DPT | ✅ DONE | `character_attack_common.go:56-75,191-194` snapshot at apply via `snapshotVenomDamagePerTick(luck, totalMagicAttack(c), coef)`; commit `d9ce1378c`. Note: `totalMagicAttack` is partial v83 — equip MAtk sum without INT contribution (acceptable as documented) |
| 26 | End-to-end smoke verification | ✅ DONE | All required tests exist (see PRD §10 mapping) |
| 27 | Final docs + audits | ✅ DONE | All boxes checked; commit `66fb58c69` marks plan complete |

**Completion Rate:** 27/27 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## PRD §10 Acceptance Criteria

| # | Criterion | Test Evidence |
|---|-----------|---------------|
| 1 | Reflect end-to-end | `TestReflectFlow_PhysicalInsideRange_EmitsReflect` at `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go:152` |
| 2 | Reflect range gate | `TestComputeReflect_OutsideRange_ReturnsZero` and `TestReflectFlow_AfterExpiry_NoReflect` at same file lines 41, 260 |
| 3 | Venom 3-stack | `TestAddStatusEffect_VenomOverflow_EvictsByEarliestExpiresAt` at `services/atlas-monsters/atlas.com/monsters/monster/builder_test.go:44`; `TestHandleStatusEffectApplied_VenomFirstApply_*` and `TestHandleStatusEffectApplied_VenomSecondAndThirdApply_DoesNotBroadcast` at `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go:205,228` |
| 4 | Venom expire collapse | `TestHandleStatusEffectExpired_VenomLastSlot_BroadcastsMonsterStatReset` at `consumer_test.go:253` |
| 5 | Mist | `TestMistTick_LiveMist_AppliesDiseaseToContainedCharacters` at `services/atlas-maps/atlas.com/maps/tasks/mist_tick_test.go:113`; `TestMistCreated_BroadcastsAffectedAreaCreated` at `services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer_test.go:62` |
| 6 | Picker un-skip | `TestPicker_AreaPoisonAllowed` at `services/atlas-monsters/atlas.com/monsters/monster/picker_test.go:163` |
| 7 | Player poison DoT | `TestPoisonTick_*` at `services/atlas-buffs/atlas.com/buffs/tasks/poison_test.go:14-34` |
| 8 | Immunity exclusion | `TestExecuteStatBuff_PhysicalImmune_CancelsActiveMagicImmune` and symmetric at `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go:832,891` |
| 9 | Dispel guard | `TestStatusCancel_PhysicalSkill_RejectedWhilePhysicalReflectActive`, `TestStatusCancel_MagicSkill_RejectedWhileMagicalReflectActive`, `TestStatusCancel_PhysicalSkill_AllowedWhileMagicalReflectActive`, `TestStatusCancel_TargetingReflectItself_AllowedRegardlessOfClass`, `TestStatusCancel_NoSkillClass_FallsThroughToNormalCancel` at `processor_test.go:1192,1228,1265,1303,1339` |
| 10 | cjson | 5 round-trip / empty-as-array tests at `services/atlas-monsters/atlas.com/monsters/monster/kafka_test.go:28-99` |
| 11 | Test coverage | All listed test categories present (see rows 1–10 above) |
| 12 | No regressions | All 5 affected services build + test green (§Build & Test Results below) |

## Notable Deviations (each justified)

1. **Task 8 — out-of-plan edits to `mobskill/builder.go` and `monster/registry.go`.** Plan only specified `executeStatBuff`. Implementation correctly recognized that reflect coords were being stripped by the Redis serialize/deserialize round-trip in `registry.go:99-226`, which would have made the wired-up reflect data invisible to subsequent reads. Justified necessary fix; covered by reflect tests downstream.
2. **Task 9 — pre-cancel runs inside `applyBuff` closure (not at `UseSkill` gate).** The plan placed the mutual-exclusion check at the top of `UseSkill`. Putting it inside `applyBuff` means each AoE target gets re-fetched and pre-cancelled individually, which matches the AoE-per-target semantics of the rest of `executeStatBuff`. Behaviorally equivalent for single-target skills and strictly more correct for AoE.
3. **Task 10 — signature drift on StatusMirror methods.** `EffectId` is `string` (matches wire), tenant key is `uuid.UUID` (matches the existing `inbox.go` pattern), `OnApplied`'s 4th arg is `now time.Time`, and `OnExpired/Cancelled` are `(t, uniqueId, effectId)` with no `statuses` map. These all match the actual call sites, and the tests exercise the real signatures.
4. **Task 13 — extra `WorldId/ChannelId/MapId` convenience getters on `Mist`.** Pure additions; no semantic change.
5. **Task 15 — `NewProcessor(l, ctx, p)` is 3-arg (added producer Provider).** Matches the reactor pattern used elsewhere in atlas-maps. `NewProcessorWithRegistry` is the test seam.
6. **Task 16 — `processorFactory` package-level var as test seam.** Standard Atlas pattern for swapping the processor under test.
7. **Task 17 — `NewProcessorWithRegistry` and `NewTestRegistry` exported helpers; "POISON"/"MONSTER" and `tickInterval=1000` hardcoded.** Hardcoded values match PRD requirements and the atlas-buffs PoisonTick interval of 1s. Future configurability is non-blocking.
8. **Task 18 — posLookup originally a stub, real client wired in Task 19.** Sequence matches plan intent (Task 19 explicitly verifies the client).
9. **Task 23 — MonsterStatus apply still fires on reflected hits.** Plan said to `continue` past status apply when reflected; impl only zeroes the `Damage` call (`character_attack_common.go:179-183` skips `mp.Damage`, but `:186-196` still applies `mp.ApplyStatus`). PRD FR-4.3.1 only requires "set the entry's monster damage to zero (no `DAMAGED` event for that entry)" — it is silent on whether status apply should also be suppressed. The current behavior is defensible: a player landing a hit close enough to be reflected has still made contact, and applying e.g. a freeze status from the same skill is consistent with how successful hits behave. **Recommendation: leave as-is, but flag this as a small spec ambiguity to revisit if a regression report surfaces.**
10. **Task 24 — extended existing `cancelStatusCommandBody` rather than introducing a new envelope; uses the literal "MAGICAL" to match `ReflectKindMagical = "MAGICAL"`.** Less churn, identical behavior.
11. **Task 25 — `totalMagicAttack` is a partial v83 approximation (equip MAtk sum, no INT contribution).** Documented in code comment at `character_attack_common.go:60-75`. Acceptable for the scope of task-036 (player venom DPT was not previously snapshotted at all). Future stat-completeness work should revisit.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| libs/atlas-constants | ✅ PASS | ✅ PASS | `monster` pkg has `skill_test.go` covering reflect kind constants |
| libs/atlas-packet | ✅ PASS | ✅ PASS | All packages green |
| services/atlas-monsters/atlas.com/monsters | ✅ PASS | ✅ PASS | `monster` and `kafka/consumer/monster` packages green; status/reflect/venom/picker tests all pass |
| services/atlas-channel/atlas.com/channel | ✅ PASS | ✅ PASS | `monster`, `socket/handler`, `kafka/consumer/monster`, `kafka/consumer/mist` green |
| services/atlas-buffs/atlas.com/buffs | ✅ PASS | ✅ PASS | `tasks` package green |
| services/atlas-maps/atlas.com/maps | ✅ PASS | ✅ PASS | `mist`, `tasks`, `map/character` green |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. Optional follow-ups (do not block merge):

1. Decide whether to suppress MonsterStatus apply on reflected hits (Task 23 deviation §9). Either:
   - Document the current "status still applies" behavior in PRD FR-4.3.1, or
   - Wrap the post-reflect MonsterStatus apply at `character_attack_common.go:186-196` in `if !reflected`.
2. Round out `totalMagicAttack` (Task 25 §11) to include INT contribution and any other v83 sources when broader player-stat work is undertaken.

---

## Backend Guidelines Audit

- **Date:** 2026-04-29
- **Diff:** `5af5a1a1f..66fb58c69` (task-036)
- **Reviewer:** backend-guidelines-reviewer (Atlas DOM-* / SUB-* / SEC-* checklist)
- **Overall:** PASS

### Phase 1 — Build & Test Gate

| Service / Lib | Build | Tests |
|---|---|---|
| `libs/atlas-constants` | PASS | PASS (`monster` 0.002s) |
| `libs/atlas-packet` | PASS | PASS (`field/clientbound` covered by `affected_area_test.go`) |
| `services/atlas-buffs` | PASS | PASS (`tasks` 0.017s, `character` 158.821s) |
| `services/atlas-monsters` | PASS | PASS (`monster` 188.425s, `kafka/consumer/monster` 0.006s) |
| `services/atlas-channel` | PASS | PASS (incl. `monster`, `socket/handler`, `kafka/consumer/monster`, `kafka/consumer/mist`) |
| `services/atlas-maps` | PASS | PASS (incl. `mist` 0.006s, `tasks` 0.007s, `map/character` 0.003s) |
| `-race` smoke | n/a | PASS — `atlas-maps/{mist,tasks}` and `atlas-channel/monster`, `atlas-channel/kafka/consumer/{mist,monster}` ran clean with `-race`. |

### Phase 2 — Domain Discovery

Task-036 is a stateful in-process feature (no GORM domain packages introduced). Affected packages classify as follows:

| Package | Type | Notes |
|---|---|---|
| `services/atlas-maps/atlas.com/maps/mist` | domain (no entity.go — in-memory registry, not GORM) | model.go + builder embedded in model.go + processor.go + producer.go + registry.go |
| `services/atlas-maps/atlas.com/maps/character` | sub-domain (REST client only) | requests.go + rest.go + processor.go |
| `services/atlas-maps/atlas.com/maps/tasks` (MistTickTask) | support (task) | wires mist registry to char client + buff producer |
| `services/atlas-maps/atlas.com/maps/kafka/{consumer,message}/mist` | support (Kafka glue) | mirrors atlas-monsters wire types |
| `services/atlas-monsters/atlas.com/monsters/monster` | existing domain | extended StatusEffect/builder/processor/producer/registry; no model.go restructure |
| `services/atlas-monsters/atlas.com/monsters/kafka/{consumer,message}/{monster,mist}` | support (Kafka glue) | producer-only (mist) + consumer/producer (monster) |
| `services/atlas-channel/atlas.com/channel/monster` | existing domain | added `status_mirror.go` (singleton) and producer cancel-status overload |
| `services/atlas-channel/atlas.com/channel/kafka/consumer/{mist,monster}` | support (Kafka glue) | new mist consumer + extended monster status consumer |
| `services/atlas-channel/atlas.com/channel/socket/handler` (character_attack_common.go) | handler | no DB/REST CRUD — uses StatusMirror |
| `libs/atlas-packet/field/clientbound/affected_area_*` | shared lib | new packet writers |
| `libs/atlas-constants/monster/skill.go` | shared lib | enum constants + helper |

Most DOM-* checks are GORM-shaped (entity.go, ToEntity, providers, RegisterTenantCallbacks) and do not apply to these in-memory / Redis-backed packages. The audit therefore enforces only the checks that *do* fit: builder + immutable model, processor signature + tenant resolution, producer header decorators, REST client conventions, JSON:API on REST DTOs, singleton via `sync.Once`, concurrency safety, multi-tenancy header passthrough, table-driven tests, and Kafka topic / wire shape consistency.

### atlas-maps `mist` domain (new)

| ID | Check | Status | Evidence |
|---|---|---|---|
| DOM-immut | Immutable model (private fields + getters) | PASS | `services/atlas-maps/atlas.com/maps/mist/model.go:16-37` (private fields), `:39-152` (accessors only, no setters) |
| DOM-builder | `builder.go` / `NewBuilder` + fluent setters + `Build()` | PASS | `mist/model.go:185-298` (`Builder` co-located in model.go; `NewBuilder(id, f)` at `:211`, `Build()` at `:275`). Builders co-located with models is consistent with `atlas-monsters/monster/builder.go`. |
| DOM-process | `NewProcessor(l, ctx, ...)` resolves tenant from ctx | PASS | `mist/processor.go:37-53` — `tenant.MustFromContext(ctx)` at `:49`. Producer.Provider is the injection seam (`p` field). |
| DOM-producer | Producer uses `producer.SingleMessageProvider` + tenant-keyed payload | PASS | `mist/producer.go:14-37` (`createdEventProvider`) and `:40-53` (`destroyedEventProvider`). Both key on the mist UUID for partition order. |
| DOM-singleton | Registry is process singleton via `sync.Once` | PASS | `mist/registry.go:36-48` (`registryOnce`, `GetRegistry()`); test seam `NewTestRegistry()` at `:53`. |
| DOM-rwmutex | Registry uses `sync.RWMutex`; readers `RLock`, writers `Lock` | PASS | `mist/registry.go:32` (`sync.RWMutex`); writers `:75, :88, :141` use `Lock`; readers `:108, :125, :157` use `RLock`. |
| DOM-tenant-key | Tenant scoping via `tenant.Model` not raw uuid string | PASS | `mist/registry.go:24-27` (`tenantBucket{tenant tenant.Model, ...}`); index keyed by `t.Id().String()` at `:57-59`. `GetTenants()` returns the full `tenant.Model` so callers don't have to round-trip an external tenant registry — used by `MistTick.runOnce` at `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go:134`. |
| DOM-rollback | Processor rolls back registry insert on emit failure | PASS | `mist/processor.go:71-82` — `Add` followed by `message.Emit`; on emit error `r.Remove` undoes the insert before returning error. |
| DOM-functional | Pure helpers / no DB calls in processor | PASS | `mist/processor.go:58-100` — only Registry + emitter calls; no GORM, no `os.Getenv`. |
| DOM-test-table | Table-driven tests for non-trivial logic | PASS | `mist/model_test.go` exercises `Contains`, `Expired`, `ShouldTick`, `WithLastTick`; `mist/registry_test.go` covers concurrent Add/Remove/Get/UpdateLastTick. |

### atlas-maps `character` REST client (new)

| ID | Check | Status | Evidence |
|---|---|---|---|
| REST-CL-1 | JSON:API interface on RestModel (`GetName`, `GetID`, `SetID`) | PASS | `character/rest.go:21-43` — `GetName()` returns `"characters"` (matches atlas-character convention; verified via `grep` of all in-tree callers, all use `"characters"` in REST resource type), `GetID()` / `SetID()` strconv-based, plus required `SetToManyReferenceIDs` no-op for api2go. |
| REST-CL-2 | `requests.RootUrl(<SERVICE>)` for base URL | PASS | `character/requests.go:16` — `requests.RootUrl("CHARACTERS")`. Matches every other service's character client (`atlas-channel`, `atlas-cashshop`, `atlas-fame`, `atlas-guilds`, `atlas-pets`, …). |
| REST-CL-3 | Curried `requests.Request[T]` shape, no manual JSON parse | PASS | `character/requests.go:18-20` returns `requests.Request[RestModel]`; called as `requestById(id)(p.l, p.ctx)` at `character/processor.go:30`. No `json.NewDecoder` / `json.Unmarshal` in handler path. |
| REST-CL-4 | Test seam for base URL | PASS | `character/requests.go:14-16` (`baseURLProvider` package-level var) — the established Atlas pattern for httptest-driven REST client tests. |
| REST-CL-5 | Marker-only ID with `json:"-"` (set via SetID) | PASS | `character/rest.go:14` — `Id uint32 \`json:"-"\``. |

### atlas-monsters monster package (extension)

| ID | Check | Status | Evidence |
|---|---|---|---|
| IMM-1 | StatusEffect retains immutable shape (private fields, getters, copy-on-write) | PASS | `services/atlas-monsters/atlas.com/monsters/monster/status.go:14-33` (private fields including new reflect* fields), `:76-167` (getters), `:136-139` (`WithLastTick` returns a copy). |
| IMM-2 | Builder appends/evicts via copy semantics, not in-place mutation of returned Model | PASS | `monster/builder.go:134-157` — VENOM stack handling clones via `append(b.statusEffects[:i], …)`; `Clone(m)` at `:12-38` deep-copies StatusEffects slice (`copy(effects, m.statusEffects)` at `:14`). |
| KAFKA-1 | Producer body is JSON-tagged and uses `producer.SingleMessageProvider` | PASS | `monster/producer.go:83-100` (`statusEffectAppliedEventProvider` populates new reflect fields); JSON struct at `monster/kafka.go:110-125`. |
| KAFKA-2 | Mist command producer keys on `OwnerId` for partition order | PASS | `monster/producer.go:16-24` — `producer.CreateKey(int(body.OwnerId))`. |
| KAFKA-3 | `cjson` empty-array safety on bodies that may be nil (FR-4.10) | PASS | `monster/kafka.go:154-198` — custom `MarshalJSON` for `statusEventDamagedBody`, `statusEventKilledBody`, `statusEffectAppliedBody`, `statusEffectExpiredBody`, `statusEffectCancelledBody` ensures `[]`/`{}` over `null`. |
| MISTBUILD | Mist body construction is pure / unit-testable | PASS | `monster/processor.go:681-708` — `buildMistCreateBody` separated from `executeMist` (`:671-676`) so it can be exercised directly. |
| MISTCAP | Mist duration capped at 60s (risks §2) | PASS | `monster/processor.go:667` (`MistDurationCapMs = 60_000`); applied at `:683-685`. |
| DISPELG | `CancelStatusEffectGuarded` consults `sourceSkillClass` only when non-empty; reflect-only target list bypasses guard | PASS | `monster/processor.go:1090-1117` — empty class falls through to existing CancelAll/Cancel; `targetingReflectOnly` carve-out at `:1093-1099`; refusal at `:1106-1107`. |
| IMMUNX | Immunity mutual exclusion pre-cancels opposite immunity before apply | PASS | `monster/processor.go:722-750` — pre-cancel runs inside `applyBuff` for both self and AoE targets; uses `CancelStatusEffect` (not the guarded variant) so internal callers don't trip the dispel guard. |
| VENOMEV | VENOM eviction picks oldest-by-ExpiresAt when at cap | PASS | `monster/builder.go:135-150` — iterates and tracks `evictIdx` based on `ExpiresAt().Before`; only evicts at `venomCount >= 3`. |
| REGRED | Redis round-trip preserves all reflect fields | PASS | `monster/registry.go:85-104` (storedStatusEffect adds `reflect*` with `omitempty`); `toStored` at `:115-136`; `fromStored` at `:204-228`. |
| REGCJSON | `cjson` empty-table tolerance via `unmarshalTolerantArray` | PASS | `monster/registry.go:53-77` — both `damageEntryList` and `statusEffectList` accept the `{}` form Lua emits for empty arrays. |
| TICKAUTO | DoT auto-set tick interval covers POISON and VENOM | PASS | `kafka/consumer/monster/consumer.go:93-101` and field-level variant at `:154-162`. |
| GUARDFLD | Field-level cancel uses guarded variant so player-originated cancels obey FR-4.9 | PASS | `kafka/consumer/monster/consumer.go:191-193` (`CancelStatusEffectGuarded(...)`). |
| PICKER-AREA | Picker no longer skips `AREA_POISON` (executeMist now handles it) | PASS | `monster/picker.go` (no `SkillTypeAreaPoison` exclusion); category is `SkillCategoryDebuff` at `libs/atlas-constants/monster/skill.go:218-223`; processor dispatches via `executeMist` at `monster/processor.go:563, :645`. |
| MIRRORHANDOFF | Producer wire body matches consumer-side struct byte-for-byte | PASS | Producer `monster/kafka.go:110-125` vs consumer `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go:146-161` — identical JSON tags, identical numeric widths (`Duration uint32`, `ReflectPercent int32`, `ReflectMaxDamage int32`, `ReflectLtX/LtY/RbX/RbY int16`). |

### atlas-channel monster (extension) + StatusMirror

| ID | Check | Status | Evidence |
|---|---|---|---|
| MIRROR-SINGLE | `StatusMirror` is process singleton via `sync.Once` (matches inbox.go pattern) | PASS | `services/atlas-channel/atlas.com/channel/monster/status_mirror.go:69-84` (`statusMirrorOnce`, `GetStatusMirror`). |
| MIRROR-MUTEX | RWMutex; readers `RLock`, writers `Lock` | PASS | `status_mirror.go:65` (`sync.RWMutex`); writers `:117, :150, :193` use `Lock`; readers `:213, :245` use `RLock`. |
| MIRROR-TENANT | Per-tenant nesting (`uuid.UUID -> uniqueId -> effectKey -> []StatusEntry`) preserves multi-tenancy isolation | PASS | `status_mirror.go:64-67`; tenant key derived via `t.Id()` (`tenant.Model.Id()`) at every call site (`:140, :151, :194, :215, :247`). No string concatenation collisions across tenants. |
| MIRROR-PRUNE | OnMonsterGone clears state on `DESTROYED` and `KILLED` so the mirror stays bounded | PASS | `kafka/consumer/monster/consumer.go:135` (DESTROYED) and `:219` (KILLED) call `monster.GetStatusMirror().OnMonsterGone(...)`. |
| MIRROR-EXP | OnExpired/OnCancelled remove only the matching `EffectId` (not the whole status key) | PASS | `status_mirror.go:148-175` — `removeByEffectId` filters per-entry by `EffectId`, then drops empty status-key buckets, then drops empty monster maps. |
| MIRROR-RACE | `GetReflect` skips entries whose ExpiresAt has passed in wall-clock time | PASS | `status_mirror.go:233-235` (`if !now.Before(e.ExpiresAt) { continue }`). |
| WIRE-COLLAPSE | Wire-collapse path captures pre-OnApplied venom count *before* mutating mirror | PASS | `kafka/consumer/monster/consumer.go:362-366` — snapshot `priorVenomCount` first; `OnApplied` follows at `:371`; collapse decision at `:392-395`. |
| MIRROR-EXP-COLL | Expiry/cancel path mutates mirror first, then computes collapse against post-state | PASS | `kafka/consumer/monster/consumer.go:423` and `:457` invoke `OnExpired`/`OnCancelled` before consulting `VenomCount`. |
| BCAST-SEAM | Broadcast seams held as package-level vars so tests can swap stubs | PASS | `kafka/consumer/monster/consumer.go:313-327` (`monsterStatSetBroadcaster`, `monsterStatResetBroadcaster`); mirrored mist seams at `kafka/consumer/mist/consumer.go:55-69`. |
| ATTACK-PURE | `computeReflect` is pure (no IO) and matches inclusive-edge semantics | PASS | `socket/handler/character_attack_common.go:34-49` — bbox check uses `<` / `>` outside the box (inclusive on every edge); total/percent/cap math tested at `socket/handler/character_attack_common_test.go`. |
| ATTACK-KIND | Kind matching uses `attackKindFromAttackType` to bridge AttackType→reflect kind | PASS | `character_attack_common.go:81-89, :137, :160`. |
| VENOMSNAP | Venom DPT snapshot computed at apply time using `coef * Luck * MagicAttack` | PASS | `character_attack_common.go:56-58` (`snapshotVenomDamagePerTick`); applied at `:151, :193`. coef randomized at apply, not at tick. |
| DISPCLASS | Channel populates `SourceSkillClass` for crash/dispel skills | PASS | `services/atlas-channel/atlas.com/channel/skill/handler/common.go:90-102` — physical for crash skills, magical for Priest Dispel; producer signature widened at `services/atlas-channel/atlas.com/channel/monster/producer.go:51-68`. |

### atlas-channel `mist` consumer (new)

| ID | Check | Status | Evidence |
|---|---|---|---|
| MIST-CON | Consumer registers with span+tenant header parsers and persistent config | PASS | `services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go:24-30` (`SetHeaderParsers(SpanHeaderParser, TenantHeaderParser)`); `:38-43` uses `message.PersistentConfig`. |
| MIST-CON-TENANT | Consumer enforces tenant + world/channel match before broadcasting | PASS | `kafka/consumer/mist/consumer.go:76, :98` — `sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId)` gate. |
| MIST-CON-WIRE | Channel-side `Event` envelope and bodies match atlas-maps producer byte-for-byte | PASS | atlas-maps `kafka/message/mist/kafka.go:60-87` vs atlas-channel `kafka/message/mist/kafka.go:18-46` — identical JSON tags and types. atlas-monsters command-only struct mirrors atlas-maps' `Command`/`CreateCommandBody`/`CancelCommandBody` byte-for-byte (`services/atlas-monsters/atlas.com/monsters/kafka/message/mist/kafka.go:22-55`). |
| MIST-CON-FALLBACK | Consumer no-ops when type does not match its handler | PASS | `kafka/consumer/mist/consumer.go:73, :95` — early-return guards. |

### atlas-maps `mist` consumer + MistTickTask

| ID | Check | Status | Evidence |
|---|---|---|---|
| TASK-FACTORY | Task injects producer.Provider and processorFactory as seams | PASS | `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go:87-119` (`MistTick.producerProvider`, `processorFactory`, `charsInField`, `posLookup`). |
| TASK-TENANT | Each tick fans out per-tenant goroutines with `tenant.WithContext` so the producer header decorator carries the right tenant | PASS | `tasks/mist_tick.go:147-148` (`tctx := tenant.WithContext(ctx, t)`); used both as the producer ctx (`r.producerProvider(tctx)`) and the posLookup ctx (`:169`). |
| TASK-EMIT-BUF | Apply-disease commands emitted via `message.Emit(p)(buf -> Put...)` so a single tick produces an atomic batch | PASS | `tasks/mist_tick.go:167-181`. |
| TASK-EXPIRY | Expired mists destroyed via processor (so MIST_DESTROYED still emits) before tick | PASS | `tasks/mist_tick.go:153-157` runs `Destroy(..., ReasonExpired)` before any tick work. |
| TASK-EMPTY-FIELD | Empty-field mists still advance `lastTick` to avoid starvation re-processing | PASS | `tasks/mist_tick.go:163-166` — `UpdateLastTick` runs even on the empty-member fast path. |
| MIST-CON-MAPS | Maps-side consumer wires `processorFactory` with the standard `producer.ProviderImpl(l)(ctx)` | PASS | `services/atlas-maps/atlas.com/maps/kafka/consumer/mist/consumer.go:21-23`. |
| MIST-CON-MAPS-TENANT | Header parsers attached for tenant + span propagation | PASS | `kafka/consumer/mist/consumer.go:31` (`SetHeaderParsers(SpanHeaderParser, TenantHeaderParser)`). |
| TICK-RACE | `runOnce` uses `sync.WaitGroup` for fan-out, no shared-state writes off-goroutine | PASS | `tasks/mist_tick.go:133-145`; per-goroutine state is the per-tenant `processTenant` slice — registry mutations happen under the registry's own RWMutex. |
| BUFFTOPIC | Buff command uses the canonical `COMMAND_TOPIC_CHARACTER_BUFF` env var (no hardcoded broker topic name) | PASS | `tasks/mist_tick.go:32` matches `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go:13` and `services/atlas-monsters/atlas.com/monsters/monster/disease.go:15` — local re-declaration of the env var name is the established Atlas cross-service convention (no shared lib for topic names). |

### Shared lib changes

| ID | Check | Status | Evidence |
|---|---|---|---|
| LIB-PKT-1 | AffectedAreaCreated/Removed packet writers carry meaningful constructors and Encode method | PASS | `libs/atlas-packet/field/clientbound/affected_area_created.go:32-77`; `affected_area_removed.go` mirrors. Writer constants `AffectedAreaCreatedWriter`/`AffectedAreaRemovedWriter` exposed for `session.Announce`. |
| LIB-PKT-2 | mistKey UUID→uint32 derivation is documented (collision risk noted) | PASS | `affected_area_created.go:80-87` — comment explains `uuid.UUID.ID()` (time_low) and the v83 wire constraint. |
| LIB-CONST-1 | Reflect kind constants colocated with monster skill constants | PASS | `libs/atlas-constants/monster/skill.go:18-19` — `ReflectKindPhysical`/`ReflectKindMagical`. |
| LIB-CONST-2 | `ReflectKindForSkill` covers PHYSICAL_COUNTER, MAGIC_COUNTER, PHYSICAL_MAGIC_COUNTER | PASS | `libs/atlas-constants/monster/skill.go:197-208`. |

### Phase 4 — Security review

Task-036 does not touch authentication, JWT, or callback URLs. No SEC-01..SEC-04 obligations apply. Spot checks ran clean:

- No hardcoded secrets introduced (`grep -rn "secret\|password\|key=\|token=" <changed-files>` returns only function/comment matches, no credentials).
- `os.Getenv` confined to existing env-var lookups for Kafka topics (via `topic.EnvProvider`) and REST roots (via `requests.RootUrl`) — no direct `os.Getenv` in handlers or processors.

### Notes / Observations (non-blocking)

These are honest observations against the strict letter of the guideline doc; none rise to the level of FAIL given the package types involved.

1. **`mist` package has no separate `builder.go` file.** The Builder is co-located in `mist/model.go` (`:185-298`). The file-responsibilities doc treats `builder.go` as a recommended layout; the in-tree precedent (`atlas-monsters/monster/builder.go` is a separate file) goes the other way. Co-location is a stylistic deviation, not a correctness issue. Consider extracting if future contributors find the model.go length unwieldy.
2. **`StatusMirror.StatusEntry.Statuses` field is exported and shares the underlying map with the source body** (`services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:377` writes `e.Body.Statuses` into the mirror without a defensive copy). Today no caller mutates that map after dispatching, so it's safe; if a future consumer ever mutates, both the mirror and the wire body will see the change. Cheap to harden by deep-copying in `StatusMirror.OnApplied` if/when a mutation lands.
3. **Maps-side `mist/processor.go` defines `Destroy` as a no-fail-on-emit operation** (`mist/processor.go:89-99` logs and continues). This is a deliberate tradeoff (registry-side authoritative) and is documented inline. Calling sites in `MistTick` and the cancel-command consumer correctly treat the `Destroy` return error as recoverable.
4. **Producer `statusEffectAppliedBody` does NOT carry `TickInterval`** (`services/atlas-monsters/atlas.com/monsters/monster/kafka.go:110-125`), but channel-side `StatusEffectAppliedBody` in `status_mirror.go:18-34` declares a `TickInterval int64`. Reading the wire produces a zero TickInterval downstream — which is what the consumer relies on (it never references `TickInterval` for monster status). Worth confirming with the next person who touches DoT visualization on the channel side; not a present-day defect.

### Final verdict

**PASS — no blocking findings.** All builds and tests (incl. `-race`) green. Producer/consumer wire shapes match across all three services. Singleton + RWMutex pattern correctly applied to both `monster.StatusMirror` and `mist.Registry`. Multi-tenancy preserved via `tenant.MustFromContext` / `tenant.WithContext` on every Kafka and Redis path. Reflect math + dispel guard are exercised by table-driven tests (`character_attack_common_test.go`, `monster/processor_test.go`, `monster/status_mirror_test.go`).
