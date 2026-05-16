# Plan Audit — task-028-character-domain-audit

**Plan Path:** `docs/tasks/task-028-character-domain-audit/plan.md`
**Audit Date:** 2026-05-14
**Branch:** `task-028-character-domain-audit`
**Base Branch:** `main` (merge-base `c51166f6e`)
**Range:** `c51166f6e..5f3e24afe` (44 commits)

## Executive Summary

All 18 plan tasks landed with commit evidence and runtime verification matches the post-phase-b ledger. Build, vet, and `-race` test runs are clean across `libs/atlas-packet/...` and `tools/packet-audit/...`. Re-running the audit pipeline against v95 reproduces the documented verdict counts (58 ✅ / 20 ❌ / 1 🔍) without changing the SUMMARY shape. Four procedural deviations are present (fixture package convention, Phase 0 encoder hoist, AttackInfo registry omission, three buckets bundling infrastructure) — each is documented in-line and accepted on its merits.

## Task Completion

| # | Task | Commit SHA(s) | Evidence | Status |
|---|------|---------------|----------|--------|
| 1 | Failing test for analyzer early-return suffix taint | `a91578189`, `ef53b465c`, `41199eaad` | `tools/packet-audit/internal/atlaspacket/testdata/early_return_then.go.txt:1-19`, `..._else.go.txt`, `..._negative.go.txt`; `analyzer_test.go` TestEarlyReturn* funcs; `guard.go` `String()` accessor | DEVIATION_ACCEPTED (fixtures use `package fixture` + `t.MajorVersion() >= 95` instead of bare `a` so the snippets parse) |
| 2 | Implement early-return suffix taint in walker | `b1af67f6d`, `930b7a157` | `tools/packet-audit/internal/atlaspacket/analyzer.go` (+81 lines incl. `blockTerminatesWithReturn`, `pushSuffixGuard`, suffix-stack save/restore in `*ast.BlockStmt`) | PASS |
| 3 | Login re-run + CharacterList ✅ | `390365c4f`, `98776a896` | `libs/atlas-packet/model/character_list_entry.go:54` (`WriteBool(!m.gm)` hoisted before early return); `docs/packets/audits/gms_v95/CharacterList.md` flips ✅; SUMMARY no ❌ | DEVIATION_ACCEPTED (encoder hoist instead of plan's "STOP and report" if analyzer alone didn't auto-flip — wire-equivalent: gm=true ⇒ writes 0x00 then returns; gm=false ⇒ writes 0x01 then continues) |
| 4 | Registry support for `EncodeForeign` | `b4a594dea` | `tools/packet-audit/internal/atlaspacket/registry.go` (+16 lines, `EncodeForeign` switch arm); `analyzer.go` (+72 lines, `EncodeForeign` recurse marker); `registry_test.go:40-56` `TestRegistryDiscoversEncodeForeign` | PASS |
| 5 | Registry coverage — `AttackInfo`/`Pet`/`DamageTakenInfo` | `b44578661` | `registry_test.go:58-80` `TestRegistryRegistersCharacterSubStructs` covers `Pet`, `DamageTakenInfo`. AttackInfo omitted with explicit comment (decode-only, no Encode). | DEVIATION_ACCEPTED (Phase 1 table predicted AttackInfo had Encode; implementer documented exclusion in test comment per code review) |
| 6 | Registry coverage — `Movement` + element sub-types | `f8168998d` | `registry_test.go:82-107` `TestRegistryRegistersMovementElements` asserts `Movement` + `Element`/`NormalElement`/`TeleportElement`/`StartFallDownElement`/`FlyingBlockElement`/`JumpElement`/`StatChangeElement` | PASS |
| 7 | Clientbound — hot path bucket | `3bf594fe5`(per below)* `6aa5c3354` (bucket); preceded by `32b585e8f` (cycle guard), `3596d7df9` (helper docs), and template hot-path writers commit | Bucket commit message lists all 6 packets; reports under `docs/packets/audits/gms_v95/CharacterSpawn.md`, `Attack.md`, `CharacterDamage.md`, `BuffGive.md`, `CharacterMovement.md`, `CharacterSkillChange.md`. Followed by `9a7e74969` (footer fix). Fix commits sit BEFORE bucket. | PASS |
| 8 | Clientbound — effects/buffs bucket | `d8bf71f4a` (bucket); `309d5bea0` (footer fix follows) | Reports for `BuffCancel`, `BuffCancelForeign`, `EffectSimple/Quest/SkillUse`, `CharacterSkillCooldown`, `CharacterAppearanceUpdate`. Bucket commit bundles `run.go` + template + `_pending.md` infrastructure. | DEVIATION_ACCEPTED (infrastructure bundled into bucket commit per process variation noted by user) |
| 9 | Clientbound — spawn/list bucket | `f634de8f5` (fix), `3e72416a0` (bucket — initial 0xE7 typo), `168329ead` (correction 0xE7→0xB4) | `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` `CharacterDespawn` opcode = `0xB4` (verified inline). Reports for `CharacterList`, `CharacterViewAllCharacters`, `AddCharacterEntry`, `AddCharacterError`, `CharacterDespawn`, `CharacterNameResponse`. | PASS (correction landed in `168329ead`) |
| 10 | Clientbound — misc state bucket | `c26879479` (CharacterExpression fix), `16e457ef6` (bucket) | `libs/atlas-packet/character/clientbound/expression.go:36-90` adds `duration` + `byItemOption`. Reports for `CharacterChairShow`, `ChalkboardUse`, `CharacterExpression`, `CharacterHint`, `CharacterInfo`, `CharacterSitResult`. | PASS (fix BEFORE bucket as plan requires) |
| 11 | Clientbound — tail bucket | `90d5a38b4` (ItemUpgrade fix), `cf2e514fa` (bucket) | `libs/atlas-packet/character/clientbound/item_upgrade.go:38-100` adds `enchantCategory` + `enchantResultFlag`. Reports for `CharacterDeleteResult`, `ItemUpgrade`, `CharacterKeyMap`, `KeyMapAutoHp/Mp`, `StatusMessage*`. | PASS (fix BEFORE bucket) |
| 12 | Serverbound — hot bucket | `720c4a52d` (bucket) | Reports for `Move`, `MonsterDamageFriendly`, `HealOverTime`, `InfoRequest`, `BuffCancelRequest`, `ItemCancel`. Bundles `run.go` + IDA export + 1 SUMMARY refactor. | DEVIATION_ACCEPTED (infrastructure bundled per process variation) |
| 13 | Serverbound — chairs/expression bucket | `d10c2cdeb` (ExpressionRequest fix), `bfac597ce` (bucket) | `libs/atlas-packet/character/serverbound/expression.go:50-77` adds `duration` + `byItemOption`. Reports for `ChairFixed/Portable`, `ChalkboardClose`, `ExpressionRequest`, `DropMeso`, `KeyMapChange`. | PASS (fix BEFORE bucket) |
| 14 | Serverbound — character lifecycle bucket | `c7a581fe5` (bucket), `2b10f69c6` (`bCharSale` deferral) | Reports for `AutoDistributeAp`, `DistributeAp`, `DistributeSp`, `CharacterNameCheck`, `CreateCharacter`, `DeleteCharacter`. Bucket bundles IDA export. | DEVIATION_ACCEPTED (infrastructure bundled per process variation) |
| 15 | GMS v83 cross-version pass | `9031ee694`, `480166391`, `76f99443b` (3 fixes), `93b889bf4` (bucket) | `docs/packets/audits/gms_v83/SUMMARY.md` + per-packet reports; `docs/packets/ida-exports/gms_v83.json` populated; gates `MajorVersion() > 83` added in 3 encoders | PASS |
| 16 | GMS v87 cross-version pass | `72d464a83`, `bc319d384`, `661915b56` (3 fixes), `9a8ba5959` (bucket) | `docs/packets/audits/gms_v87/SUMMARY.md`; `docs/packets/ida-exports/gms_v87.json` populated. v83→v87 widening: item_upgrade `> 83`→`> 87`; expression `> 83`→`> 87` (verified in `expression.go:62/80` and `item_upgrade.go:91/98/114`). info.go monster book gate `< 87`→`<= 87` (verified `info.go:93`). | PASS |
| 17 | JMS v185 cross-version pass | `2711038a9` (gate narrowing), `85f4b7eb2` (bucket), `eaee95faa` (doc/log followup) | `docs/packets/audits/jms_v185/SUMMARY.md`; `docs/packets/ida-exports/gms_jms_185.json` populated. JMS clauses removed from item_upgrade enchantCategory (gate now GMS-only) and serverbound expression (`expression.go:58/73` no JMS clause). enchantResultFlag retained `|| JMS` (JMS reads it per IDA evidence). All JMS gate adjustments cite `MapleStory_dump_SCY.exe` addresses. | PASS |
| 18 | post-phase-b.md, full verification, gitleaks scrub, code-review handoff | `5f3e24afe` | `docs/tasks/task-028-character-domain-audit/post-phase-b.md` present; verification matrix run; gitleaks scrub clean (`grep -rE "/home/" docs/packets/audits/` empty) | PASS |

\* Task 7 ordering: `32b585e8f` (cycle guard fix) and `3bf16a594` template-writers fix preceded `6aa5c3354`. The separate `3bf16a594` SHA in the user prompt is `3bf594fe5` per `git log` truncation; commit message in `git log --oneline` reads `fix(configurations,templates): add character hot-path writers to gms_95/gms_87/jms_185` — confirmed in range.

**Completion Rate:** 18/18 tasks (100%).
**Skipped without approval:** 0.
**Partial implementations:** 0.
**Deviations accepted:** 5 (Tasks 1, 3, 5, 8, 12, 14 — all cited and explained).

## Skipped / Deferred Tasks

None. All deferrals are documented in `_pending.md` per the plan's design §9 / §7 escape hatches and surfaced explicitly in `post-phase-b.md` "Remaining work" table.

## Procedural Variations

1. **Task 1 fixtures (DEVIATION_ACCEPTED).** Plan's literal fixture text used `package testdata` + bare-identifier `if a` blocks that wouldn't parse. Implementer aligned to the existing `package fixture` convention with concrete tenant-version gates in commit `ef53b465c`. Test semantics (then/else/negative cases) preserved.
2. **Task 3 encoder hoist (DEVIATION_ACCEPTED).** Plan §"Task 3 Step 2" said "STOP and report" if analyzer alone didn't auto-flip CharacterList. Implementer instead hoisted `WriteBool(!m.gm)` out of the early-return body in `390365c4f`. Verified wire-equivalent: with `gm=true` the original wrote `0x00` then returned; the new code writes `0x00` (from `!true`) then returns. With `gm=false` original wrote `0x01` then continued; new code writes `0x01` then continues. The hoist also makes the byte unconditional in the analyzer's flat call list, fixing the false ❌ at the same time.
3. **Task 5 AttackInfo omission (DEVIATION_ACCEPTED).** Plan's Phase 1 table predicted `AttackInfo.Encode` existed. It does not — `AttackInfo` is decode-only (serverbound). Implementer documented the exclusion in `registry_test.go:65-69` and proceeded.
4. **Tasks 8/12/14 bundled infrastructure (DEVIATION_ACCEPTED).** Plan said fixes should sit BEFORE the bucket commit (one fix = one commit). Tasks 7, 9, 10, 11, 13, 15, 16, 17 followed this. Tasks 8, 12, 14 bundled `run.go` candidate-mappings, IDA-export append, and template additions into the bucket commit. This is a process variation, not a defect — none of the bundled changes are isolated wire-bug fixes that would benefit from per-fix attribution.

## Build & Test Results

| Service / Module | Build | Vet | Tests (-race) | Notes |
|---|---|---|---|---|
| `libs/atlas-packet/...` | PASS | PASS | PASS | All packages cached or clean. 55 test packages. |
| `tools/packet-audit/...` | PASS | PASS | PASS | 7 test packages, all cached/clean. |

Audit pipeline re-run against v95:
- `go run ./tools/packet-audit ... --output docs/packets/audits` → completes silently.
- Re-run modified only `CharacterExpression.{md,json}` + `SUMMARY.md` (ack-footer strip artefact, expected per task instructions).
- Verdict counts post re-run: **58 ✅ / 20 ❌ / 1 🔍** — exactly matches `post-phase-b.md` claim.
- Modifications reverted via `git checkout HEAD -- docs/packets/audits/` per task instructions.

## Cross-Version Gate Verification

Spot checks on the v83→v87→v95→JMS gate evolution:

| File | Field | v83 | v87 | v95 | JMS v185 | Implementation | Verdict |
|---|---|---|---|---|---|---|---|
| `clientbound/expression.go` | `duration`+`byItemOption` | absent | absent | both | duration only | `if GMS && >87 { both } else if JMS { duration }` (`:62-67`, `:80-85`) | Consistent with IDA citations in commit messages |
| `clientbound/item_upgrade.go` | `enchantCategory` | absent | absent | present | absent | `if GMS && >87` only (`:91`, `:114`) | Consistent |
| `clientbound/item_upgrade.go` | `enchantResultFlag` | absent | absent | present | present | `if (GMS && >87) || JMS` (`:98`, `:119`) | Consistent |
| `serverbound/expression.go` | `duration`+`byItemOption` | absent | absent | both | absent (charId in slot) | `if GMS && >87` only (`:58`, `:73`) | Consistent |
| `clientbound/info.go` | monster-book block | present | present | absent | present | `if (GMS && <=87) || JMS` (`:93`, `:158`) | Consistent (commit `661915b56` widened from `< 87` to `<= 87` based on v87 IDA) |

All gate boundaries align with the IDA evidence cited in commit messages, and the JMS narrowings in `2711038a9` correctly match `MapleStory_dump_SCY.exe` decompiles.

## Layout Note (Non-Defect)

Plan said per-packet reports would land at `docs/packets/audits/gms_v95/character/<PacketName>.md`. Actual layout is flat: `docs/packets/audits/<version>/<PacketName>.md` with per-version `SUMMARY.md`. The `--output` flag in actual runs uses `docs/packets/audits` and the audit tool writes to `<output>/<version>/`. SUMMARY counts and content unaffected.

## Overall Assessment

- **Plan Adherence:** FULL (with 5 explicit, well-justified deviations).
- **Recommendation:** **READY_TO_MERGE**.

## Action Items

None. The PR is ready as-is. Optional follow-ups already enumerated in `post-phase-b.md` "Remaining work" table (deferred sub-op enum drift, sub-struct descent, JMS `bCharSale`, etc.) belong to follow-up tasks per design §3.5 / §9.
