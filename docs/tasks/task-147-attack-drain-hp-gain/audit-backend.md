# Backend Audit — task-147-attack-drain-hp-gain (atlas-channel)

- **Service Path:** `services/atlas-channel/atlas.com/channel`
- **Scope:** `socket/handler/character_attack_common.go`, `socket/handler/character_attack_drain_test.go` (new) — diff `01a0a3bb0..59f20e2df`
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*, SUB-*, FILE-*, SEC-*)
- **Date:** 2026-07-25
- **Build:** PASS
- **Vet:** PASS
- **Tests:** PASS (22/22 new/touched subtests pass; full module `go test -race -count=1 ./...` clean)
- **Guards:** `tools/redis-key-guard.sh` PASS (exit 0), `tools/goroutine-guard.sh` PASS (exit 0)
- **Overall:** PASS

## Build & Test Results

```
$ cd services/atlas-channel/atlas.com/channel && go build ./...
(clean, no output)

$ go vet ./...
(clean, no output)

$ go test -race -count=1 ./...
(no FAIL lines; all packages "ok" or "[no test files]")

$ go test -race -count=1 -run 'TestIsDrainSkill|TestDrainHealAmount|TestOnDamageApplied|TestDrainTryHeal' -v ./socket/handler/...
--- PASS: TestIsDrainSkill (7 subtests)
--- PASS: TestDrainHealAmount (14 subtests)
--- PASS: TestOnDamageApplied_ReceivesSummedDamageTotal
--- PASS: TestOnDamageApplied_NotCalledForZeroDamageEntry
--- PASS: TestOnDamageApplied_NotCalledForReflectedEntry
--- PASS: TestDrainTryHeal_EmitsCappedHeal
--- PASS: TestDrainTryHeal_MonsterFetchError_SkipsHeal
--- PASS: TestDrainTryHeal_ZeroEffectiveStats_SkipsHeal
--- PASS: TestDrainTryHeal_EmitErrorSwallowed
--- PASS: TestDrainTryHeal_PerMonsterCaps
ok  atlas-channel/socket/handler  1.060s

$ tools/redis-key-guard.sh   → exit 0, no findings under services/atlas-channel
$ tools/goroutine-guard.sh   → exit 0, no findings under services/atlas-channel
```

## Package Classification

`services/atlas-channel/atlas.com/channel/socket/handler/` contains none of `model.go`, `resource.go`, `processor.go`, `entity.go`, `rest.go`, `administrator.go`, `builder.go`, `provider.go` (verified via directory listing). It is the socket packet-dispatch layer, not a DDD domain/sub-domain package or a REST resource package — `file-responsibilities.md`'s `resource.go`→`processor.go`→`provider.go` JSON:API layering targets domain packages, which this is not. The DOM-*/SUB-*/FILE-* checklists therefore do not literally apply to this diff; the change adds pure functions (`isDrainSkill`, `drainHealAmount`) and one orchestrator (`drainTryHeal`) inside the existing handler-support file, plus widens an existing hook signature. No new Processor, RestModel, entity, or administrator type is introduced. Checked mechanically below against the checks that do transfer (DOM-21 constant reuse, error-swallowing/no-cross-domain-write discipline, integer safety, Kafka emission idiom, test quality) since those are language- and idiom-level, not file-placement-level.

## Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No duplication of atlas-constants skill ids | PASS | `character_attack_common.go:83-92` `isDrainSkill` switches on `skill3.AssassinDrainId`, `skill3.MarauderEnergyDrainId`, `skill3.ThunderBreakerStage3EnergyDrainId`, `skill3.NightWalkerStage2VampireId` — all defined in `libs/atlas-constants/skill/constants.go:3150,3205,3289,3367`. No numeric skill-id literal appears anywhere in the diff's production code (`git diff` confirms only the four named constants and `skill3.Id(ai.SkillId())` at line 448). |
| — | Error handling never propagates/aborts | PASS | `drainTryHeal` (`character_attack_common.go:323-353`) has a `void` signature — it cannot return an error to its caller. Monster-fetch failure is logged via `l.WithError(err).Debugf(...)` and the function returns early (`336-339`); `ChangeHP` emit failure is logged via `l.WithError(err).Errorf(...)` and swallowed (`350-352`), never surfaced. It is invoked from the `onDamageApplied` closure (`448-450`), itself `func(monsterId uint32, totalDamage uint32)` — no return path exists back into `processAttack`, so damage application, the attack broadcast (`462-487`), and projectile consumption (`492-496`) all proceed unaffected regardless of drain outcome. |
| — | Integer safety (uint32/int16 cap math, no overflow) | PASS | `drainHealAmount` (`character_attack_common.go:235-250`) performs all multiplication/comparison in `uint64` (`239-245`) before the single `int16` clamp at `246-248`; max possible product `uint64(4294967295) * uint64(32767)` (~1.4e14) is far under the `uint64` ceiling, so no intermediate overflow. The damage-total accumulator in `processDamageInfoEntry` (`197-205`) sums in `uint64` and explicitly saturates to `math.MaxUint32` (`202-204`) before narrowing to `uint32`, preventing wraparound on pathological per-line damage arrays. Test `character_attack_drain_test.go:75` (`4_000_000_000` damage, `x=100`, max `uint32` caps) asserts the clamp lands exactly at `32767`. |
| — | Kafka `ChangeHP` follows established command idiom | PASS | Production wiring at `character_attack_common.go:449` passes `cp.ChangeHP` (the same `character.Processor.ChangeHP` already used for HP-cost debit at line 394) as `drainTryHeal`'s `changeHP` collaborator. `character/processor.go:276-278` implements it as `producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(ChangeHPCommandProvider(f, characterId, amount))` — identical idiom to sibling `SetHP`/`ChangeMP`/`RequestDropMeso` (`processor.go:280-286`). No new emission path was invented. |
| — | Lazy loader rename consistency + at-most-once fetch | PASS | `git diff` over the branch shows `loadVenomStats`→`loadEffectiveStats` renamed at every site with no straggler: the `damageInfoEntryDeps` field (`105-106`), both VENOM branches in `processDamageInfoEntry` (`143`, `190`), the outer closure definition + guard flag (`417-429`), the `deps` struct literal (`437`), and the `drainTryHeal` call site (`449`), which passes the *same* `loadEffectiveStats` closure — not a second instance. `grep -rn loadVenomStats` over the diff returns zero hits. The `effectiveStatsLoaded` boolean (`416`, checked `418`) guarantees at most one `effective_stats.NewProcessor(...).GetByCharacterId(...)` call per `processAttack` invocation, shared between VENOM DPT and every drain heal in that attack. |
| — | No `MajorVersion` gate on drain (documented, correctly absent) | PASS | `isDrainSkill` (`83-92`) contains no version comparison. Consistent with the documented rationale: `processAttack` destroys the session at `376-379` if the caster does not own `ai.SkillId()`, making a version-absent drain skill structurally unreachable — a version gate here would be redundant/wrong, and none was added. |
| — | No attack-type gate on drain (documented, correctly absent) | PASS | `onDamageApplied` (`444-451`) gates MP Eater on `ai.AttackType() == packetmodel.AttackTypeMagic` (`445`) but gates drain only on `ai.SkillId() > 0 && isDrainSkill(...)` (`448`) — no `AttackType` comparison, matching the requirement that the four drain skills span melee/ranged/energy. |
| — | Test quality — assert real behavior, not vacuous pass | PASS | 22 subtests across 6 functions in `character_attack_drain_test.go`. Cap math is table-driven with independently-verifiable arithmetic (e.g. `333*16/100=53` at line 60, `2001/2=1000` floor at line 66, "tighter of two caps wins" at line 67). Error/skip paths assert the mock *was* invoked before asserting no downstream effect — e.g. `TestDrainTryHeal_EmitErrorSwallowed` (`276-296`) asserts `changeHPCalls == 1` (proving the error branch executed, not skipped) rather than only checking absence of panic. `TestOnDamageApplied_NotCalledForReflectedEntry` (`160-198`) constructs a real `monster.ReflectInfo`/`monster.Model` via the project's Builder pattern (`monster.NewModelBuilder`), not a hand-rolled struct literal, consistent with the Test Helper Pattern rule in `CLAUDE.md`. |

## Non-Blocking Observations

- `character_attack_drain_test.go:222,246,265,290,317-318` pass raw skill-id literals (`4101005`, `14101006`) as the `skillId` argument to `drainTryHeal` instead of `skill3.AssassinDrainId` / `skill3.NightWalkerStage2VampireId`. This does **not** trigger DOM-21 — the parameter is consumed only as an opaque logging value inside `drainTryHeal` (`character_attack_common.go:347-348`), never classified or switched on — but using the named constants would be more resilient to any future skill-id renumbering and is a one-line readability improvement.
- Pre-existing (not introduced by this branch, noted for context only): `processAttack`'s inline `_ = cp.ChangeHP(...)` HP-cost debit at `character_attack_common.go:394` and `_ = cp.ChangeMP(...)` at `397` discard the error entirely (no log), whereas the new `drainTryHeal` path logs on failure. Not a regression — the new code is strictly more careful than the code it sits beside — but flagged since the branch did not bring the neighboring lines up to the same standard. Out of scope per the audit instructions (pre-existing condition, not introduced by this branch).

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- Consider replacing raw skill-id literals in `character_attack_drain_test.go` (lines 222, 246, 265, 290, 317-318) with the `skill3.*Id` constants used elsewhere in the same file, for consistency and resilience to renumbering.
