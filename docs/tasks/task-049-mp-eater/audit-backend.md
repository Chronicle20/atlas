# Backend Audit — task-049-mp-eater

- **Scope:** Diff `898e60bc6..b0e999647` (16 commits) on branch `task-049-mp-eater`.
- **Services touched:** `atlas-channel`, `atlas-monsters` (Go only; no frontend).
- **Guidelines source:** `.claude/skills/backend-dev-guidelines/`
- **Date:** 2026-05-03
- **Build:** PASS (both services)
- **Tests:** PASS (atlas-channel ok in 0.010s for `socket/handler`; atlas-monsters ok in 161.032s for `monster`; `monster/builder_test.go` ok in 0.025s; `kafka/consumer/monster` ok in 0.007s)
- **Overall:** PASS (with two NON-BLOCKING observations)

## Build & Test Results

```
cd services/atlas-channel/atlas.com/channel  && go build ./... && go test ./... -count=1   → PASS
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./... -count=1   → PASS
```

New tests (all passing):
- atlas-channel `socket/handler/character_attack_mp_eater_test.go` — 23 sub-tests (`TestMpEaterShouldProc`: 7, `TestResolveMpEaterSkillId`: 11, `TestMpEaterAbsorbAmount`: 5).
- atlas-channel `monster/builder_test.go` — `TestModelBuilder_SetMaxMp`, `TestCloneModel_PreservesMaxMp`.
- atlas-monsters `monster/drain_mp_test.go` — 7 cases (HappyPath, ClampsAtZero, SkipsZeroMaxMp, SkipsZeroCurrentMp, SkipsZeroRequest, MissingMonster, SkipsBoss).

## Scope classification

The changed packages are **wire/processor mutations on existing domains**, not new domain creation. Neither the channel-side `monster/` package nor the channel-side `socket/handler/` package owns a REST resource, GORM entity, or `model.go`-based domain in the DOM-* sense. Likewise, `atlas-monsters/monster/` is a long-standing domain whose `model.go`, `builder.go`, `processor.go`, etc. predate this branch and were not introduced here. The DOM-01 through DOM-19 row-by-row checklist therefore does not apply turn-by-turn — the relevant gates are: shared-libs reuse (DOM-21), immutable-model + builder discipline, functional composition for kafka, multi-tenancy gating, and TDD coverage. Each is checked below.

## Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | Shared-libs reuse — no redeclaration of types/constants from `libs/atlas-constants/` | PASS | `character_attack_common.go:19-22, 89-99` uses `field`, `job`, `monster2`, `skill3` from `libs/atlas-constants/...`. MP Eater skill ids (`skill3.FirePoisionWizardMpEaterId`, `skill3.IceLightningWizardMpEaterId`, `skill3.ClericMpEaterId`) come from `libs/atlas-constants/skill/constants.go:3012,3034,3056`. Job ids (`job.FirePoisonWizardId`, etc.) come from `libs/atlas-constants/job/constants.go:1158-1166`. The `mpEaterSkillIds` map at `character_attack_common.go:89-99` is **not** a duplicate of an existing shared symbol — see Non-Blocking note below. |
| DOM-MUT-01 | Multi-tenancy gating in MP_CHANGED consumer | PASS | `kafka/consumer/monster/consumer.go:535` `sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId)` precedes any side effect; same pattern as siblings (e.g., `:96, :144, :174, :230, :258, :284, :304, :371, :434, :468, :502, :516`). |
| DOM-MUT-02 | Channel-side `monster.Model` immutability + builder discipline | PASS | `monster/model.go:25-41` private fields only; `monster/model.go:43-129` accessor-only methods; `monster/builder.go:13-28` mirror builder; `monster/builder.go:74-77` adds `SetMaxMp`; `monster/builder.go:40-57` `CloneModel` preserves `maxMp`/`mp`; `monster/builder.go:110-130` `Build()` validates `uniqueId`. |
| DOM-MUT-03 | `effect.Model` exposure of new attributes is read-only | PASS | `data/skill/effect/model.go:9-64` keeps fields private; `:127-135` `Prop()` and `X()` are pure accessors. No setters introduced. |
| DOM-MUT-04 | Functional composition for kafka producers | PASS | `monster/producer.go:151-167` `DrainMpCommandProvider` returns `model.Provider[[]kafka.Message]` via `producer.SingleMessageProvider(key, value)`. `kafka/message/monster/kafka.go:79-83` `DrainMpCommandBody` exported. Channel emit at `monster/processor.go:90-93` uses curried `producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(...)`. atlas-monsters: `monster/producer.go:130-138` `mpChangedStatusEventProvider` uses `statusEventProvider` → `producer.SingleMessageProvider`. |
| DOM-MUT-05 | Curried consumer registration | PASS | atlas-channel `kafka/consumer/monster/consumer.go:31-37` registers via `rf(consumer2.NewConfig(l)(...)...)`. atlas-monsters `kafka/consumer/monster/consumer.go:18-25,49-51` registers `handleDrainMpCommand` through the same curried `rf` chain. |
| DOM-MUT-06 | Channel-side MP_CHANGED handler refunds caster MP and broadcasts visual | PASS | `kafka/consumer/monster/consumer.go:530-559` switches on `e.Body.Reason`; `MpChangeReasonMpEater` calls `character.NewProcessor(l,ctx).ChangeMP(f, ..., int16(e.Body.Amount))` then `socketHandler.AnnounceSkillSpecial` + `AnnounceForeignSkillSpecial`; default case logs and ignores so unknown reasons don't fan out. |
| DOM-MUT-07 | atlas-monsters `DrainMp` re-screens guards authoritatively | PASS | `monster/processor.go:1345-1383` rechecks `Alive()`, `MaxMp == 0`, `Mp == 0`, `requestedAmount == 0`, then Boss via information lookup (with `testInformationLookup` test seam, mirroring `UseBasicAttack` at `:715-720`). Clamp is via `DeductMp` which returns post-clamp state; `actual := preMp - post.Mp()` produces the post-clamp delta and short-circuits on `actual == 0`. |
| DOM-MUT-08 | Side-effect emission only via processor (`emit` indirection) | PASS | `monster/processor.go:60-90` emitter abstraction unchanged; `DrainMp` at `:1382` emits via `p.emit(EnvEventTopicMonsterStatus, mpChangedStatusEventProvider(...))`. No bypass to producer.ProviderImpl. |
| DOM-MUT-09 | Reason discriminator on `MP_CHANGED` is namespaced and gated | PASS | Constants centralised: `kafka/message/monster/kafka.go:107` (channel side) and `monster/kafka.go:36` (monsters side) both define `MpChangeReasonMpEater = "MP_EATER"`. Channel handler default case in `consumer.go:555-557` logs unknown reasons rather than crashing. |
| DOM-MUT-10 | `mpEaterTryProc` swallows errors at the call boundary | PASS | `socket/handler/character_attack_common.go:130-188` returns nothing; uses `Errorf`/`Debugf` and never returns or `panic`s — matches the documented contract at `:127-129`. Consistent with siblings: VENOM apply (`:280, :328`), `mp.Damage` (`:313`), `mp.ApplyStatus` (`:328`) all use `_ = ...` or `WithError(err).Errorf(...)`. |
| DOM-MUT-11 | Pure helpers + table-driven tests | PASS | `mpEaterShouldProc` at `:109-114` is pure; `resolveMpEaterSkillId` at `:101-104` pure; `mpEaterAbsorbAmount` at `:119-124` pure. Tests: `character_attack_mp_eater_test.go:11-87` follow `cases := []struct{...}` + `t.Run` pattern. |
| DOM-MUT-12 | Per-monster proc gated to magic skill attacks only | PASS | `socket/handler/character_attack_common.go:334-336` requires `ai.AttackType() == packetmodel.AttackTypeMagic && ai.SkillId() > 0`. |
| DOM-MUT-13 | Cheap pre-screen on channel before crossing the wire | PASS | `mpEaterTryProc` at `:139-180` exits early on no-eater job, level 0, prop ≤ 0, monster fetch failure, MaxMp/Mp = 0, failed roll, amount = 0 — all before `mp.DrainMp`. |
| DOM-MUT-14 | No `os.Getenv()` in handlers | PASS | grep returned 0 matches in the changed handler files. |
| DOM-MUT-15 | Logger plumbing (`FieldLogger`, not `*Logger`) | PASS | `monster/processor.go:14-25` `Processor.l logrus.FieldLogger`; `mpEaterTryProc(l logrus.FieldLogger, ...)` at `character_attack_common.go:131`; consumer handlers receive `l logrus.FieldLogger` from atlas-kafka adapter. |
| DOM-MUT-16 | DRAIN_MP test seam (information lookup) follows existing pattern | PASS | `monster/processor.go:1361-1366` mirrors `UseBasicAttack` at `:715-720`; both use the same `testInformationLookup` package var, isolating tests from atlas-data HTTP. |

## Sub-Domain Checks (SUB-*)

Not applicable — no new sub-domains were introduced. The change adds new commands/events to existing wire packages and new methods/handlers to existing processors/consumers.

## External HTTP Client (EXT-*)

Not applicable — no new outbound atlas-service client was introduced. The atlas-channel `monster/processor.go:GetById` REST call (used by `mpEaterTryProc` to read `MaxMp`/`Mp`) reuses the pre-existing `requestById` plumbing (`processor.go:27-29`) without modification.

## Security (SEC-*)

Not applicable — no auth, token, JWT, redirect, or secret-handling code was changed. The only credential-adjacent surface is the tenant-scoped Kafka envelope, which is gated as documented in DOM-MUT-01.

## Summary

### Blocking
None.

### Non-Blocking observations

1. **`mpEaterSkillIds` allow-list lives in the service rather than `libs/atlas-constants/job`** (`socket/handler/character_attack_common.go:89-99`). The map encodes a job-line inheritance fact (Cleric line → ClericMpEater for all of {Cleric, Priest, Bishop}; same for the FP and IL lines). `libs/atlas-constants/job/constants.go:194-301` already lists each second-job's skill set including the `MpEater` entry, but it does not expose a "given any job in this line, return its MP Eater skill" lookup, and Priest/Bishop/etc. don't carry the MpEater entry on their own job rows (the canonical pattern is that subsequent advancements inherit). Because the shared lib has no equivalent helper today, redeclaring it here is **not** a DOM-21 violation under the rule as written. **However**, this is exactly the kind of cross-service concept the project's CLAUDE.md guidance asks to push into shared libs over time. Suggest filing a follow-up to add a `job.MpEaterSkillFor(jobId) (skill.Id, bool)` helper in `libs/atlas-constants` so a future reviewer doesn't have to re-derive line membership in another service. Not blocking for this task.

2. **Skill-id constant typo in shared lib leaks into the service.** The shared lib spells `FirePoisionWizardMpEaterId` / `FirePoisionWizardMpEater` (`Poision` instead of `Poison`), which the service is forced to mirror at `character_attack_common.go:90, 91, 92`. Test file likewise: `character_attack_mp_eater_test.go:43-45`. Not introduced by this task — the misspelling pre-exists in `libs/atlas-constants/skill/constants.go:485, 3012`. Worth a separate cleanup PR (rename + sweep callers); flagged here as an observation, not a defect of task-049.

3. **Monsters-side MP_CHANGED struct is unexported.** `atlas-monsters/monster/kafka.go:39-48,157-163` use lowercase `statusEvent[E]` and `statusEventMpChangedBody`. atlas-channel mirrors with exported `kafka/message/monster/kafka.go:110-119,215-221`. The convention is intentional ("wire types exported on the channel side, unexported on the monsters consumer side per existing convention," per the audit prompt) and matches every other event body in both files (e.g., `statusEventDamagedBody` vs `StatusEventDamagedBody`). Not a finding — recorded for completeness so the next reviewer doesn't flag it.

## Conclusion

The MP Eater wiring follows existing Atlas conventions across both services: immutable models with builder pattern, curried functional kafka producers/consumers, processor-only side effects, multi-tenant gating on every consumer entry, channel pre-screen + monster authoritative re-check, and TDD coverage on every pure helper plus the new processor method. No DOM-* violations were found. Non-blocking observations are about library hygiene (shared-lib promotion candidates and a pre-existing misspelling), not this task's correctness.
