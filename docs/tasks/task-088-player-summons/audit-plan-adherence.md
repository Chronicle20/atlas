# Plan Audit — task-088-player-summons (Plan Adherence)

**Plan Path:** docs/tasks/task-088-player-summons/plan.md
**Audit Date:** 2026-06-12
**Branch:** task-088-player-summons
**Base Branch:** main
**Commit range:** 1b579d365..HEAD (42 feature commits)

## Executive Summary

Every plan task is faithfully implemented. The new `atlas-summons` service is registered, builds, and ships with real (non-stub) implementations of all six processor methods. The interim "conservative" damage ceiling was genuinely replaced by the faithful Cosmic-parity weapon-type port (`ConservativeMaxPerHit` is gone; `FaithfulMaxPerHit` is the only shipping ceiling). No `// TODO`, no `panic(`, no leftover Phase-0 `return nil` stubs remain in the new code. Cross-service Kafka envelopes (monster ADD_PUPPET/REMOVE_PUPPET/DAMAGE/APPLY_STATUS, SUMMON_STATUS events, buff APPLY) have JSON tags matching their consumers field-for-field. Opcode honesty is intact: IDA-confirmed values are used where IDBs exist, and v12/v84/v92 derived values are explicitly flagged `derived, unverified` in `summon-packet-delta.md`. All builds, vets, and tests pass.

**Verdict: every task faithfully implemented. READY_TO_MERGE (pending the parallel guideline audit).**

## Build & Test Results (actual output)

| Target | Build | Vet | Tests | Notes |
|---|---|---|---|---|
| services/atlas-summons/atlas.com/summons | PASS | PASS | PASS | `summon` pkg tests ok; all others no-test |
| services/atlas-monsters/atlas.com/monsters | n/a | n/a | PASS | regression clean (monster, information, consumers all ok) |
| libs/atlas-packet/summon (clientbound + serverbound) | n/a | n/a | PASS | round-trip tests over version variants |
| libs/atlas-constants/summon | n/a | n/a | PASS | roster (21 entries) |

## Task Completion

### Phase 0 — Scaffold
| Task | Status | Evidence |
|---|---|---|
| 0.1 Register service | IMPLEMENTED | services.json:432-436; go.work:76; kustomization.yaml:59; docker-bake.hcl:88; deploy/k8s/base/atlas-summons.yaml; env-configmap.yaml:70,141 |
| 0.2 Module/logger/leader cfg | IMPLEMENTED | summons/go.mod, logger/init.go, leaderconfig.go |
| 0.3 Model + builder | IMPLEMENTED | summon/model.go, builder.go, model_test.go (round-trip + AddHP clamp) |
| 0.4 Redis registry | IMPLEMENTED | summon/registry.go (store/field/owner idx + GetAll), registry_test.go |
| 0.5 Object-id allocator | IMPLEMENTED | summon/id_allocator.go |
| 0.6 Kafka event envelope + providers | IMPLEMENTED | summon/kafka.go, producer.go |
| 0.7 Processor interface + impl | IMPLEMENTED | summon/processor.go — all bodies real (see below) |
| 0.8 JSON:API resource + REST | IMPLEMENTED | summon/resource.go, rest.go; world/resource.go; rest/handler.go |
| 0.9 main.go boot + gate | IMPLEMENTED | main.go; bake target present; service builds |

### Phase 1 — Roster + spawn/despawn + v83 + channel
| Task | Status | Evidence |
|---|---|---|
| 1.1 Roster (21) | IMPLEMENTED | libs/atlas-constants/summon/roster.go; Lookup:57, IsSummonSkill:62; TestRosterHas21Entries asserts len==21 (passes) |
| 1.2 Skill-effect data client | IMPLEMENTED | data/skill/{model,processor,requests,rest}.go + effect/{model,rest}.go |
| 1.3 Spawn/Despawn/DespawnAllForOwner | IMPLEMENTED | processor.go:111-200 (Spawn), 375-395 (Despawn, oid release), 399-408 (cascade); conflictsMobility:413-417; re-cast loop:119-124; processor_spawn_test.go |
| 1.4 COMMAND_TOPIC_SUMMON consumer (SPAWN) | IMPLEMENTED | kafka/consumer/summon/{consumer,kafka}.go; wired in main.go |
| 1.5 Char-lifecycle despawn cascade | IMPLEMENTED | kafka/consumer/character/{consumer,kafka}.go → DespawnAllForOwner |
| 1.6 Expiry sweep | IMPLEMENTED | summon/expiry_task.go (leader-elected; GetAll grouped by tenant w/ tenant-scoped ctx:34-41); expiry_task_test.go |
| 1.7 v83 SummonSpawn/SummonRemove | IMPLEMENTED | atlas-packet/summon/clientbound/spawn.go, remove.go + _test.go |
| 1.8 channel skill-cast → SPAWN | IMPLEMENTED | atlas-channel character_skill_use.go:86-98 |
| 1.9 channel SUMMON_STATUS consumer broadcast | IMPLEMENTED | atlas-channel kafka/consumer/summon/consumer.go (6 events); writers+handlers registered in main.go |
| 1.10 v83 opcodes | IMPLEMENTED | template_gms_83_1.json (SummonSpawn writer + handler entries) |
| 1.11 Phase 1 gate | IMPLEMENTED | builds/tests green |

### Phase 2 — Movement
| Task | Status | Evidence |
|---|---|---|
| 2.1 SummonMove packets | IMPLEMENTED | clientbound/move.go, serverbound/move.go + _test.go |
| 2.2 Move processor + ownership + MOVED | IMPLEMENTED | processor.go:206-222 (ownership check:211; raw movement relay); processor_move_test.go |
| 2.3 MOVE command + channel handler/broadcast + opcodes | IMPLEMENTED | summon_move.go:25; SummonMove writer; template opcodes |
| 2.4 Phase 2 gate | IMPLEMENTED | green |

### Phase 3 — Attacker + ceiling
| Task | Status | Evidence |
|---|---|---|
| 3.1 SummonAttack packets | IMPLEMENTED | clientbound/attack.go, serverbound/attack.go + _test.go |
| 3.2 Effective-stats client | IMPLEMENTED | effectivestats/{model,processor,requests,rest}.go |
| 3.3 Conservative ceiling (interim) | IMPLEMENTED then REPLACED | shipped at commit 1b0d21b2c; superseded by 3.6 in same phase (see below) |
| 3.4 Attack processor | IMPLEMENTED | processor.go:231-306 — owner credit via monster DAMAGE (285), stun/freeze APPLY_STATUS (291), Gaviota self-cancel (302-304), clamp + warn-only alert (276-281, no BAN emit); processor_attack_test.go |
| 3.5 ATTACK command + channel + opcodes | IMPLEMENTED | summon_attack.go:30; SummonAttack writer; template opcodes |
| 3.6 Faithful weapon-type ceiling | IMPLEMENTED (replaces 3.3) | ceiling.go:26 FaithfulMaxPerHit; called at processor.go:267; ConservativeMaxPerHit absent from tree; ceiling_test.go has Cosmic-parity expected values (sword/magic/bow) |
| 3.7 Phase 3 gate | IMPLEMENTED | green |

### Phase 4 — Puppet
| Task | Status | Evidence |
|---|---|---|
| 4.1 SummonDamage packets | IMPLEMENTED | clientbound/damage.go, serverbound/damage.go + _test.go |
| 4.2 monsters ADD_PUPPET/REMOVE_PUPPET + set + bias | IMPLEMENTED | monsters kafka.go:24-25,107-127; consumer.go:173-193; puppet_registry.go; controller bias processor.go:255-268 (additive early-return, least-loaded path untouched); puppet_test.go. No regression — 13 pre-existing handlers intact. |
| 4.3 summons emits puppet signals | IMPLEMENTED | processor.go:193-197 (ADD_PUPPET on spawn), 388-392 (REMOVE_PUPPET on despawn); processor_puppet_test.go |
| 4.4 Damage processor (HP dec, destroy@0) | IMPLEMENTED | processor.go:349-371 (ownership:354, AddHP:359, Despawn at <=0:367-369); processor_damage_test.go |
| 4.5 DAMAGE command + channel + opcodes | IMPLEMENTED | summon_damage.go:25; SummonDamage writer; template opcodes |
| 4.6 Phase 4 gate | IMPLEMENTED | green |

### Phase 5 — Beholder aura
| Task | Status | Evidence |
|---|---|---|
| 5.1 Snapshot aura/hex at spawn | IMPLEMENTED | processor.go:154-180 — reads real WZ statups via GetEffect for AURA_OF_THE_BEHOLDER + HEX_OF_THE_BEHOLDER; buffChanges built from hex.Statups() (not invented); processor_beholder_test.go |
| 5.2 Leader-elected heal/buff sweep | IMPLEMENTED | summon/beholder_task.go (leader-only) — sweepHeal emits CHANGE_HP (charmsg.ChangeHPProvider:77), sweepBuff emits buff APPLY with SourceId=-1320009 (set at processor.go:176, emitted beholder_task.go:101); beholder_task_test.go asserts -1320009 |
| 5.3 SummonSkill packet + broadcast + opcode | IMPLEMENTED | clientbound/skill.go + _test.go; SummonSkill writer; v83 opcode seeded |
| 5.4 Phase 5 gate | IMPLEMENTED | green |

### Phase 6 — Multi-version
| Task | Status | Evidence |
|---|---|---|
| 6.1 Delta doc + opcode harvest | IMPLEMENTED | summon-packet-delta.md (per-packet per-version tables, confirmation column) |
| 6.2 Version-conditional encode/decode | IMPLEMENTED | spawn.go:84,109 GMS-only branch gated `IsRegion("GMS") && MajorAtLeast(95)`; other 5 packets version-stable (documented); spawn_test.go has V83 + V95 + JMS185-negative byte tests |
| 6.3 Opcodes in 6 remaining templates | IMPLEMENTED | all 7 templates (gms 12/83/84/87/92/95 + jms 185) carry summon writer/handler entries |
| 6.4 Later-version no-op test | IMPLEMENTED | commit 3e42adc0f — Spawn with a non-roster (later-version) skill is a graceful no-op |
| 6.5 Final gate + code review | IMPLEMENTED | this audit + parallel guideline audit |

## Critical Checks

- **No silent stubs / TODOs:** PASS. Grep across the new summons tree found zero `// TODO`/`FIXME`/`XXX`/`HACK`, zero `panic(`, and no leftover Phase-0 placeholder bodies. The one "later phases" string (processor.go:44) is a legitimate comment describing the emit indirection, not a stub. All six processor methods (Spawn/Move/Attack/Damage/Despawn/DespawnAllForOwner) carry real, tested implementations.

- **Conservative → faithful ceiling:** PASS. `ConservativeMaxPerHit` does not exist anywhere in the tree. `FaithfulMaxPerHit` (ceiling.go:26) is the only ceiling and is the live call site (processor.go:267). The interim clamp landed at commit 1b0d21b2c and was replaced by the faithful port at a7a287ddd, both before the Phase 3 gate — no released state ships the approximation.

- **Cross-service JSON tags:** PASS. Independent envelope-by-envelope comparison found no mismatches:
  - COMMAND_TOPIC_MONSTER: summons `monster/producer.go` add/remove-puppet + damage + apply-status bodies are byte-identical to monsters' `kafka.go` decoders.
  - EVENT_TOPIC_SUMMON_STATUS: summons `summon/kafka.go` `StatusEvent[E]` + all 6 body types identical to channel's consumer envelope.
  - COMMAND_TOPIC_SUMMON: channel producer vs summons consumer identical.
  - Buff APPLY: summons `buff/producer.go` identical to atlas-buffs consumer; SourceId = -1320009 confirmed.

- **Opcode honesty:** PASS. `summon-packet-delta.md` cites IDA-confirmed values where IDBs exist (v83/v87/v95/jms185) and explicitly marks v12, v84, and v92 handler/writer opcodes `derived, unverified — confirm against capture`. v84's collision risk against the pet band and v92's possible v95-restructured-band ambiguity are both called out. Nothing is silently guessed.

## Skipped / Deferred Tasks

None. Every plan task has corresponding implemented + tested code on the branch.

## Deviations (all faithful, with rationale)

1. **Processor interface signature evolved** vs the Phase-0 skeleton in plan.md:
   - `Spawn(...)` gained `auraLevel byte, hexLevel byte` to thread the caster's trained Beholder aura/hex skill levels (needed by Phase 5 snapshot). This is anticipated by the plan's Phase 5 design.
   - `Move(...)` gained `rawMovement []byte` so MOVED carries the raw movement blob for byte-faithful rebroadcast (Task 2.3 explicitly references `RawMovement`).
   These are additive and consistent with later-phase requirements; not gaps.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (subject to the parallel backend-guidelines audit at audit.md)

## Action Items

None required for plan adherence. Operational follow-up (already noted in the plan, not a code task): live tenants need a config patch + channel restart to pick up the new opcodes (seed templates only apply at tenant creation). The v12/v84/v92 derived-unverified opcodes must be capture-confirmed before those versions go live — already documented in summon-packet-delta.md.
