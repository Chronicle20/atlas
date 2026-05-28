# Backend Audit — task-028-character-domain-audit

- **Branch:** `task-028-character-domain-audit`
- **BASE → HEAD:** `c51166f6e` → `5f3e24afe` (44 commits)
- **Scope:** `libs/atlas-packet/character/**`, `libs/atlas-packet/model/character_list_entry.go`, `tools/packet-audit/**`, `services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go`
- **Date:** 2026-05-14
- **Build:** PASS
- **Tests:** PASS (atlas-packet, packet-audit, atlas-channel — all green)
- **Overall:** NEEDS_FIXES

## Phase 1 — Build & Test (Objective Gate)

| Module | Build | Test |
|---|---|---|
| `libs/atlas-packet` | PASS | PASS (all sub-packages green; `character/clientbound`, `character/serverbound`, `model` exercise round-trips across GMS v28/v83/v87/v95 + JMS v185 — `character/clientbound/expression_test.go:9-48`, `character/clientbound/item_upgrade_test.go:9-93`, `character/serverbound/expression_test.go:9-41`, `character/serverbound/move_test.go:9-65`, `character/clientbound/view_all_test.go:11-87`, `character/clientbound/add_entry_test.go:10-49`) |
| `tools/packet-audit` | PASS | PASS (`internal/atlaspacket/analyzer_test.go:5-69` covers early-return suffix-taint; `internal/atlaspacket/registry_test.go:9-107` covers `EncodeForeign` alt-key + Movement/Element registration) |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS (`kafka/consumer/expression/` has no test file, but the constructor signature change compiles cleanly across the rest of the service tree) |

## Phase 2 — Domain Discovery & Applicability

The audited scope contains **no domain packages** in the `services/<svc>/atlas.com/<svc>/internal/<domain>` sense (no `model.go` + `processor.go` + `administrator.go` + `resource.go` quartet). Therefore the standard DOM-* / SUB-* checklist mostly does **not** apply mechanically.

| Path | Classification |
|---|---|
| `libs/atlas-packet/character/clientbound/*.go` | Wire encoder library — no DDD model |
| `libs/atlas-packet/character/serverbound/*.go` | Wire decoder library — no DDD model |
| `libs/atlas-packet/model/*.go` | Wire-level helper types (`CharacterListEntry`, `Avatar`, `Movement`) — no DDD model |
| `tools/packet-audit/**` | Standalone CLI tool (Go AST analyzer + diff engine) |
| `services/atlas-channel/.../kafka/consumer/expression/consumer.go` | Service consumer file (only `InitConsumers` + `InitHandlers` + `handleEvent` — no domain model) |

I therefore audit against (a) build/test correctness, (b) the issues called out explicitly in the prompt, (c) gofmt / `go vet` hygiene, and (d) wire-equivalence of the structural changes (CharacterListEntry hoist, ItemUpgrade fields, CharacterExpression fields, Move/Expression JMS gate removals).

## Phase 3 — Targeted Findings

### F-01 — gofmt non-compliance across multiple touched files (Important)

Running `gofmt -l` on the files modified in this branch reports the following are not gofmt-canonical:

- `libs/atlas-packet/character/clientbound/item_upgrade.go` — newly broken by this branch (struct field columns `enchantResultFlag byte` not aligned with siblings: `libs/atlas-packet/character/clientbound/item_upgrade.go:33-41`; one-line accessors mis-aligned: `libs/atlas-packet/character/clientbound/item_upgrade.go:68-75`)
- `libs/atlas-packet/character/serverbound/expression.go` — newly broken by this branch (the new `Emote()` / `Duration()` / `ByItemOption()` accessors at `libs/atlas-packet/character/serverbound/expression.go:38-40` have over-padded right margin)
- `libs/atlas-packet/character/serverbound/move.go` — pre-existing at base (over-padded accessors at lines 28-36)
- `libs/atlas-packet/model/character_list_entry.go` — newly aggravated: the new `Avatar()` accessor was inserted at line 37 with correct padding, but adjacent `Gm()` line 39 has one fewer space than its neighbours and breaks gofmt
- `libs/atlas-packet/character/clientbound/info.go` — pre-existing at base (struct + accessor padding), not touched in any way that would have re-fmt'd it
- `libs/atlas-packet/character/clientbound/view_all.go` — pre-existing at base
- `services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go` — newly broken: the added `charpkt` import at `services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go:20` is appended after the third-party imports instead of being merged into the alphabetical group with the other `Chronicle20/atlas/libs/...` imports
- `tools/packet-audit/internal/atlaspacket/analyzer.go` — pre-existing at base
- `tools/packet-audit/cmd/run.go` — newly broken: tab/space alignment in the `EffectSimple/EffectQuest/EffectSkillUse` candidate slice at `tools/packet-audit/cmd/run.go:165-167`

Severity Important rather than Critical because none of this changes wire behaviour, build succeeds, and tests pass. But each of these files was actively edited on this branch; the branch should leave them at least no-worse than baseline, and ideally clean up newly-broken files. Fix command: `gofmt -w` over each listed file.

Evidence: `gofmt -l <file>` returns the file name; `gofmt -d <file>` shows the diffs cited above.

### F-02 — `consumer.go` hardcodes `duration=0`, drops `byItemOption` entirely (Important)

`services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go:52` constructs `charpkt.NewCharacterExpression(e.CharacterId, e.Expression, 0)`. The new constructor at `libs/atlas-packet/character/clientbound/expression.go:40` takes `duration uint32`; `byItemOption` has no constructor parameter at all and defaults to `false`. The Kafka event type `expression.Event` (`services/atlas-channel/atlas.com/channel/kafka/message/expression/kafka.go:27-34`) does not carry `Duration` or `ByItemOption` fields, so atlas-channel will always announce remote expressions to v95+ / JMS v185 clients with `duration=0` and `byItemOption=false`.

For GMS ≤ v87 this is invisible (duration is gated out — `libs/atlas-packet/character/clientbound/expression.go:62-67`). For v95+ and JMS v185 the wire shape is correct (server emits a 0-duration emote) but the data is stripped at the producer boundary. Since duration is part of the IDA-documented payload and is now properly gated through the encoder, the `expression.Event` message and the producer side should be extended to carry `Duration` and `ByItemOption` so the encoder can announce a non-zero duration. The current branch leaves the producer-side gap (and there is no `MessageBuilder.SetDuration()` etc. on the command/event in `kafka/message/expression/kafka.go`).

This is Important rather than Critical because: (a) v95+ clients accept 0-duration emotes (default-display); (b) the build compiles cleanly with the constructor change; (c) the prompt scope only asked for "constructor signature change ripple" verification — the ripple is correct and the call site compiles.

### F-03 — `MovementData()` is the new accessor name; in-tree callers updated, but `MoveData()` / `Movement()` would be more idiomatic (Minor)

`libs/atlas-packet/character/serverbound/move.go:36` exposes `func (m Move) MovementData() model.Movement`. Callers in `services/atlas-channel/atlas.com/channel/socket/handler/character_move.go:20`, `pet_movement.go:20`/`24`, `monster_movement.go:20`, `npc_action.go:23` use `.MovementData()`. Naming is fine; no action required. Flagged only because the prompt asked about ripple — confirmed no missed call sites.

### F-04 — `byItemOption` field name preserves IDA Hungarian prefix (Minor / accepted convention)

`libs/atlas-packet/character/clientbound/expression.go:37`, `libs/atlas-packet/character/serverbound/expression.go:35`. Hungarian `b`-prefix is non-idiomatic Go but is an accepted local convention in this library (also seen at `libs/atlas-packet/pet/serverbound/command.go:16` `byName bool`). Not blocking. The accessors expose `ByItemOption()` which capitalises correctly, so external API is fine.

### F-05 — `CharacterListEntry` `WriteBool(!m.gm)` hoist is wire-equivalent (PASS)

`libs/atlas-packet/model/character_list_entry.go:54` writes `WriteBool(!m.gm)`, then early-returns when `m.gm` is true at lines 55-57. The encoder still emits exactly one `rankEnabled` byte (`0x00` for GM, `0x01` for ranked) before the `m.gm` early return. For non-GM characters the byte is `0x01` and the rank fields follow. This matches the pre-hoist behaviour (old code wrote `WriteByte(0)` before returning when GM, and `WriteByte(1)` before the rank fields when non-GM). Wire shape unchanged. PASS.

The `Decode` side at `libs/atlas-packet/model/character_list_entry.go:71-95` correctly reads the rankEnabled byte first (line 80), sets `m.gm = true` and returns when zero (lines 81-84), and reads rank fields when non-zero. Round-trip test `add_entry_test.go:10-49` and `view_all_test.go:28-59` exercise this across all five `pt.Variants` — green.

### F-06 — `view_all.go` pre-init `viewAll=true` on Decode (PASS)

`libs/atlas-packet/character/clientbound/view_all.go:97-101` pre-initializes each `model.CharacterListEntry` with `viewAll=true` so its `Decode` method (`libs/atlas-packet/model/character_list_entry.go:76-78`) skips the family/viewAll placeholder byte that VIEW_ALL_CHAR packets do not carry. Comment is accurate. Round-trip green at `view_all_test.go:28-59`.

### F-07 — `ItemUpgrade` version-gate consistency between encoder and decoder (PASS)

`libs/atlas-packet/character/clientbound/item_upgrade.go:91` (encoder) and `libs/atlas-packet/character/clientbound/item_upgrade.go:114` (decoder) gate `enchantCategory` identically: `t.Region() == "GMS" && t.MajorVersion() > 87`.

`libs/atlas-packet/character/clientbound/item_upgrade.go:98` (encoder) and `libs/atlas-packet/character/clientbound/item_upgrade.go:119` (decoder) gate `enchantResultFlag` identically: `(t.Region() == "GMS" && t.MajorVersion() > 87) || t.Region() == "JMS"`.

Comment block at lines 16-32 accurately documents the IDA cross-version reality (v83/v87 = 4 × Decode1; v95 adds Decode4 + 2 × Decode1; JMS v185 = 5 × Decode1, no Decode4). Round-trip tests at `item_upgrade_test.go:9-93` exercise both `NewItemUpgrade` and `NewItemUpgradeEnchant` across all five variants and explicitly assert `enchantCategory==0` for non-GMS-v95+ and `enchantResultFlag==0` for v83/v87. PASS.

### F-08 — `Move` JMS gate removals across multiple sequential gates (PASS)

`libs/atlas-packet/character/serverbound/move.go:56-71` (encoder) and `libs/atlas-packet/character/serverbound/move.go:82-97` (decoder) use four separate `if t.Region() == "GMS" && t.MajorVersion() > 83` blocks for `dr0/dr1`, `dr2/dr3`, `dwKey/crc32`, plus one `if t.Region() == "GMS" && t.MajorVersion() > 28` block for `crc`. Gates are flat (not nested), encoder and decoder use identical conditions, and the IDA reference at lines 49-51 (`CVecCtrlUser::EndUpdateActive@0xaaa076`) documents that JMS v185 does not carry these fields. The serial-gate style is repetitive but correct; collapsing into a single guard variable would be a stylistic refactor, not a correctness issue. PASS.

Round-trip test `move_test.go:30-49` asserts `Dr0()==100 ... Crc32()==700` only for `GMS && MajorVersion>83` and `Crc()==500` only for `GMS && MajorVersion>28`. Green.

### F-09 — `CharacterInfo` monster-book gate widened to include v87 (PASS)

`libs/atlas-packet/character/clientbound/info.go:93` (encoder) and `libs/atlas-packet/character/clientbound/info.go:158` (decoder) gate the 5-Decode4 monster book block on `(t.Region() == "GMS" && t.MajorVersion() <= 87) || t.Region() == "JMS"`. Comment at line 91-92 cites `IDA v87 CWvsContext::OnCharacterInfo@0xabb181` for inclusion. Encoder/decoder match. PASS.

### F-10 — `CharacterExpression` clientbound version gates correctness (PASS, with one doc-vs-code subtlety)

`libs/atlas-packet/character/clientbound/expression.go:62-67` (encoder) and `libs/atlas-packet/character/clientbound/expression.go:80-85` (decoder) both apply:
- GMS && MajorVersion>87: writes/reads `Decode4(duration) + Decode1(byItemOption)`
- JMS: writes/reads `Decode4(duration)` only

The struct doc-comment at line 23 says `Decode4  duration      — display duration in ms [GMS>87 or JMS]` — accurate. Line 24 says `Decode1  byItemOption  — item-option emotion flag [GMS>87 only]` — accurate.

Round-trip test `expression_test.go:9-48` asserts `output.Duration()==input.Duration()` when `(GMS && Major>87) || JMS` and `output.ByItemOption()==input.ByItemOption()` when `GMS && Major>87` only. PASS.

Subtlety: the inline `// duration and byItemOption added after GMS v87 (first seen in v95).` comment at line 59 is partially misleading because it is followed by a JMS branch that also writes duration. The follow-up doc-fix commit `eaee95f` patches the doc-comment correctly and logs the JMS semantic mismatch in `_pending.md`. Code is correct; comment is acceptable. PASS.

### F-11 — `ExpressionRequest` (serverbound) JMS narrowing (PASS, with documented semantic mismatch deferred)

`libs/atlas-packet/character/serverbound/expression.go:58-61` and lines 73-76 gate both `duration` and `byItemOption` on `t.Region() == "GMS" && t.MajorVersion() > 87`. JMS does **not** read these fields — comment at lines 27-31 explains the IDA JMS v185 semantic mismatch (`CWvsContext::SendEmotionChange@0xb0b8be` encodes only `Encode4(charId)`, fundamentally different from GMS). The branch correctly narrows the gate. The remaining JMS semantic-mismatch (charId reinterpreted as emote) is documented as deferred in `docs/packets/ida-exports/_pending.md` per commit `eaee95f`. PASS for this branch's scope.

### F-12 — `analyzer.go` `(*callCtx).conjoin()` shadowing the package-level `conjoin` (PASS)

`tools/packet-audit/internal/atlaspacket/analyzer.go:202` defines `func (cc *callCtx) conjoin() *GuardExpr` which delegates to the package-level `conjoin([]*GuardExpr)` at line 691. Method-resolution is unambiguous: callers using `cc.conjoin()` always invoke the method; the inner `return conjoin(*cc.stack)` and `return conjoin(combined)` both invoke the package-level function via name resolution (no `cc.` prefix). No shadowing bug. The method correctly merges `cc.suffixGuards` into the AND-stack before delegating. PASS.

### F-13 — `analyzer.go` suffix-taint walker on early-return (PASS)

`tools/packet-audit/internal/atlaspacket/analyzer.go:215-240` (`blockTerminatesWithReturn`) descends terminating IfStmts via the `else` branch only and explicitly does not descend loops (per design §3.3). Lines 244-274 (the `*ast.IfStmt` arm of `walk`) compute `thenReturns` / `elseReturns` and:
- both → `cc.unreachableSuffix = true` (line 269)
- thenReturns only → `cc.pushSuffixGuard(negate(g))` (line 271) so the surviving branch's negated guard tints siblings
- elseReturns only → `cc.pushSuffixGuard(g)` (line 273)

Lines 275-289 (the `*ast.BlockStmt` arm) save and restore both `cc.suffixGuards` and `cc.unreachableSuffix` per block scope (lines 277-280, 288-289), and break out of the statement loop when `unreachableSuffix` is true (line 282). This correctly scopes suffix-taint to a single block and prevents leaking across nested scopes.

Tests at `internal/atlaspacket/analyzer_test.go:18-69` assert the three semantic cases (then-returns, else-returns, else-not-present-no-taint). Green. PASS.

### F-14 — `registry.go` alt-key collision risk for `EncodeForeign` (PASS)

`tools/packet-audit/internal/atlaspacket/registry.go:116` synthesises the alt-key as `recvType + "::EncodeForeign"`. Since `::` is not valid in a Go identifier, no real type can collide. The pre-existing `entry.Calls != nil && fd.Name.Name != "Encode"` skip clause is widened at line 99 to also pass through `"EncodeForeign"` so both `Encode` and `EncodeForeign` register on the same struct. Test at `internal/atlaspacket/registry_test.go:40-56` asserts `CharacterTemporaryStat::EncodeForeign` and `CharacterTemporaryStat` both resolve. PASS.

### F-15 — `diff.go` `flattenWithRegistryGuarded` cycle guard correctness (PASS)

`tools/packet-audit/internal/diff/diff.go:137-164`. Visited-set DFS:
- Entry into a recurse that's already in `visited` → emit the call unchanged (lines 148-152)
- Otherwise mark `visited[c.RecurseType] = true`, recurse into the sub-call list, then `delete(visited, c.RecurseType)` on exit (lines 154-158)

This is the textbook DFS cycle-detection (mark on entry, unmark on exit) that allows DAG re-visits across siblings. `KindRepeat` recursive expansion at line 144 reuses the same `visited` map so a loop body containing a recurse is also cycle-guarded.

Caveats (not blocking):
- `visited` is a `map[string]bool` shared across the whole flatten call. When recursion encounters a guard whose `Eval(ctx)==false` (line 140) the recurse is skipped before `visited` is marked, which is correct. When the recurse target is unknown (`reg.Calls(c.RecurseType)` returns false at line 154) the call falls through to the unconditional `out = append(out, c)` at line 161 — also correct.
- The entry helper `FlattenWithRegistry` at line 128 always allocates a fresh `visited` map, so callers don't share state across invocations. PASS.

### F-16 — `guard.go` `String()` nil-safe (PASS)

`tools/packet-audit/internal/atlaspacket/guard.go:29-34` checks `if g == nil { return "" }`. This protects the test helper at `internal/atlaspacket/analyzer_test.go:74` (`return g.String()`) from panicking when a guard is nil. PASS.

### F-17 — `cmd/run.go` `candidatesFromFName` additions (PASS)

`tools/packet-audit/cmd/run.go:131-425`. Each new character-domain case maps an IDA `FName` to one or more `(name, dir)` candidates. Coverage spans the buckets called out in the prompt (T7-T17): clientbound hot path (Spawn / Attack / Damage / BuffGive / BuffGiveForeign / Movement / SkillChange / BuffCancel{,Foreign} / Effect / SkillCooldown / AppearanceUpdate), misc-state (ChairShow / ChalkboardUse / Expression / Hint / Info / SitResult), tail (Delete / StatusMessage / ItemUpgrade / KeyMap / KeyMapAutoHp / KeyMapAutoMp), spawn/list (CharacterViewAll{Count,Characters,SearchFailed,Error} / AddCharacterEntry / AddCharacterError / Despawn / NameResponse), serverbound hot (Move / HealOverTime / InfoRequest / BuffCancelRequest / ItemCancel), serverbound chairs/expression (ChairFixed / ChairPortable / ChalkboardClose / ExpressionRequest / DropMeso / KeyMapChange + the two duplicate-suppressing returns of `nil` for `ChangePetConsumeItemID` / `ChangePetConsumeMPItemID` at lines 386, 390), serverbound lifecycle (Distribute{Ap,Sp} / AutoDistributeAp / CheckName / CreateCharacter / DeleteCharacter).

Two FNames intentionally return `nil` to skip duplicates of the parent KeyMap dispatcher; comments explain. PASS.

## DOM-* Checklist Mapping (truncated — most are N/A)

The scope contains no domain packages. For completeness:

| ID | Status | Note |
|---|---|---|
| DOM-01..05 (builder, ToEntity, Make, Transform, TransformSlice) | N/A | No DDD domain in scope. Library types use simple constructor functions (`NewItemUpgrade`, `NewCharacterExpression`, `NewCharacterListEntry` etc.) which is appropriate for wire-encoder structs. |
| DOM-06..09 (processor logger / handler patterns) | N/A | No processor / resource handlers in scope. The single consumer file (`expression/consumer.go`) is a Kafka consumer, not a REST resource. |
| DOM-10..11 (test DB tenant callbacks, lazy providers) | N/A | No DB or providers in scope. |
| DOM-12 (no `os.Getenv` in handlers) | PASS | Grep `os.Getenv` against `services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go` → 0 matches. |
| DOM-13..17 (cross-domain logic / direct entity creation / error mapping) | N/A | No HTTP handlers in scope. Consumer file at `services/atlas-channel/.../expression/consumer.go:46-57` calls into `_map.NewProcessor(...).ForOtherSessionsInMap(...)` and `session.Announce(...)` only — no cross-domain logic, no DB writes. |
| DOM-18..19 (JSON:API interfaces, flat REST models) | N/A | No REST in scope. |
| DOM-20 (table-driven tests) | PASS | All round-trip tests use `for _, v := range pt.Variants { t.Run(v.Name, func(t *testing.T) {...}) }` — `expression_test.go:10-11`, `item_upgrade_test.go:10-11`, `move_test.go:20-21`, etc. |
| DOM-21 (no atlas-constants type duplication) | PASS | No new types declared that overlap atlas-constants. `view_all.go:57` and `view_all.go:67` use `world.Id` from `libs/atlas-constants/world` correctly. The expression Kafka event at `kafka/message/expression/kafka.go` uses `world.Id`, `channel.Id`, `_map.Id` correctly. |
| DOM-22 (Dockerfile lib mentions) | N/A | No new lib direct-require in `services/atlas-channel/atlas.com/channel/go.mod` introduced by this branch. The `charpkt` import already existed transitively. |
| DOM-23 (Kafka topic naming) | N/A | No new topic constants introduced. `EnvExpressionEvent = "EVENT_TOPIC_EXPRESSION"` was pre-existing. |

## SUB-* Checklist
N/A — no sub-domain packages in scope.

## SEC-* Checklist
N/A — no auth/token surface modified.

## Summary

### Blocking (Critical) — none

### Important (should fix before merge)
- **F-01** — gofmt non-compliance on touched files: `libs/atlas-packet/character/clientbound/item_upgrade.go`, `libs/atlas-packet/character/serverbound/expression.go`, `libs/atlas-packet/model/character_list_entry.go`, `services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go`, `tools/packet-audit/cmd/run.go` were each gofmt-clean before this branch and are now broken. Fix with `gofmt -w` on each.
- **F-02** — `services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go:52` hardcodes `duration=0` and never threads `byItemOption`. The `expression.Event` Kafka message at `kafka/message/expression/kafka.go:27-34` should be extended with `Duration uint32` and `ByItemOption bool` fields, otherwise v95+/JMS clients will always observe duration=0 emotes regardless of what the originating server emitted. Producer side change is out of this branch's stated scope, but the data loss should be tracked.

### Minor (nice to have, not required)
- **F-03** — `MovementData()` accessor name; all in-tree callers updated, no action required.
- **F-04** — `byItemOption` Hungarian prefix; matches local convention (`pet/serverbound/command.go:16` `byName`).

### PASS verdicts (file:line evidence above)
- F-05 CharacterListEntry hoist wire-equivalence (`libs/atlas-packet/model/character_list_entry.go:54-67`)
- F-06 view_all viewAll pre-init (`libs/atlas-packet/character/clientbound/view_all.go:97-101`)
- F-07 ItemUpgrade encoder/decoder gate symmetry (`libs/atlas-packet/character/clientbound/item_upgrade.go:91/98/114/119`)
- F-08 Move JMS gate removals (`libs/atlas-packet/character/serverbound/move.go:56-71/82-97`)
- F-09 CharacterInfo monster-book v87 inclusion (`libs/atlas-packet/character/clientbound/info.go:93/158`)
- F-10 CharacterExpression clientbound gates (`libs/atlas-packet/character/clientbound/expression.go:62-67/80-85`)
- F-11 ExpressionRequest serverbound narrowing (`libs/atlas-packet/character/serverbound/expression.go:58-61/73-76`)
- F-12 `(*callCtx).conjoin()` no-shadowing (`tools/packet-audit/internal/atlaspacket/analyzer.go:202-210`)
- F-13 suffix-taint walker correctness (`tools/packet-audit/internal/atlaspacket/analyzer.go:215-289` + tests `analyzer_test.go:18-69`)
- F-14 EncodeForeign alt-key (`tools/packet-audit/internal/atlaspacket/registry.go:99/106-121` + tests `registry_test.go:40-56`)
- F-15 cycle-guarded flatten (`tools/packet-audit/internal/diff/diff.go:137-164`)
- F-16 nil-safe guard String (`tools/packet-audit/internal/atlaspacket/guard.go:29-34`)
- F-17 candidatesFromFName coverage (`tools/packet-audit/cmd/run.go:131-425`)

## Verdict

**NEEDS_FIXES** — gating on F-01 (gofmt) and F-02 (consumer drops `Duration`/`ByItemOption`).

Build is green, tests are green, all wire-shape changes are correct and round-trip-tested across the five `pt.Variants`. The two outstanding items are: gofmt hygiene on touched files (one command to fix), and a producer-side data-loss gap where the new `CharacterExpression.duration` / `ByItemOption` fields are only ever populated as zero/false by the live event path. Once those are addressed (or F-02 is explicitly deferred to a follow-up task with a TODO citation in `consumer.go:52`), this branch is SHIP.
