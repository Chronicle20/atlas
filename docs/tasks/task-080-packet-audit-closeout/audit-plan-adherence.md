# Plan Audit (Adherence) — task-080-packet-audit-closeout

**Plan Path:** docs/tasks/task-080-packet-audit-closeout/plan.md
**PRD Path:** docs/tasks/task-080-packet-audit-closeout/prd.md
**Audit Date:** 2026-06-04
**Branch:** task-080-packet-audit-closeout
**Base Branch:** main
**Reviewer file:** this file (`audit-plan-adherence.md`); the backend reviewer uses `audit.md`.

## Executive Summary

The plan was faithfully and completely implemented. All 47 commits between `main` and HEAD
map to plan tasks, and every task is supported by either landed code with byte-level tests,
or an IDA-grounded verdict/no-op spike note. The four affected Go modules
(`libs/atlas-packet`, `tools/packet-audit`, `services/atlas-maps/.../maps`,
`services/atlas-channel/.../channel`) plus `services/atlas-cashshop/.../cashshop` all build
and test green; `go vet` is clean on the two primary modules; no over-nested guards exist in
the region-dispatched bodies; and the closed-item regression guard (storage Show,
MonsterControl, SETFIELD/WarpToMap) holds — those files are untouched.

The several "no-op / verdict" outcomes flagged in the task brief are all CORRECT outcomes
backed by evidence: B1.3 (quest `nItemPos`) premise disproven, B1.2 gate corrected from the
plan's wrong `>83` to the IDA-correct `GMS && >=95`, B3.1–B3.6 verified-no-fix verdicts, the
emergent A5 region-dispatch analyzer descent, the JMS `isPoints→currency` follow-up fix, and
the `BuddyInvite` real-bug surfaced as an explicit follow-up (not buried).

**One presentational gap (non-blocking):** the four regenerated `SUMMARY.md` files still
display ~80 `❌`/`🔍` markers each with no in-file legend pointing to the accepted-exclusion
registry. Every such marker IS dispositioned in `docs/packets/ida-exports/_pending.md`
(OPAQUE / REPRESENTATION classes with IDA evidence), so PRD §4.8 ("residue in the registry")
is satisfied — but a naive reader of a SUMMARY cannot tell a blessed exclusion from an open
finding. See "Gaps" below.

## Task Completion

| Task | Status | Evidence (file:line / commit) |
|---|---|---|
| A0 baseline | DONE | `01ce7685d`; `docs/tasks/.../analyzer-baseline.md` (66KB) |
| A1 width-label equivalence | DONE | `71ab9002f`,`f65602aea`; `tools/packet-audit/internal/diff/diff.go`, `diff_test.go` (+148 lines); targeted tests pass |
| A2 qualified struct-name collisions | DONE | `696ae8e0d`; `tools/packet-audit/cmd/run.go`, `cmd/disambiguation_test.go` (+42); tests pass |
| A3 sub-struct descent + Opaque flag | DONE | `47a829c39`,`65911bd69`; `internal/atlaspacket/registry.go` (+186), `testdata/substruct_no_encode.go.txt`; tests pass |
| A4 early-return modeling | DONE | `f353019c5`; `internal/atlaspacket/analyzer_test.go`; baseline post-Phase-A section appended |
| A5 region-dispatch analyzer descent (emergent) | DONE | `b3a01e423`; `internal/atlaspacket/analyzer.go` (+199), `testdata/region_dispatch.go.txt`. Correctly added because B5.1/B1.5 region-dispatch created an analyzer blind spot |
| B1.1a mist `CreatedBody` event contract | DONE | `a6abc4453`; `services/atlas-maps/.../kafka/message/mist/kafka.go`, `mist/model.go` (Type field+getter+setter), `mist/producer.go`, `producer_test.go` |
| B1.1b mist `nType` source spike | DONE (verdict) | `e24677ad8`; `spike-affectedarea.md`: Type=0 correct for skill/disease mist (atlas-maps never makes nType==3 item-area-buff); four-version read-order table with addresses |
| B1.1c `AffectedAreaCreated` RECT layout + tStart gate | DONE | `45711afce`; `affected_area_created.go:86-110` (abs RECT, `v95Plus := Region=="GMS" && MajorVersion>=95` gate at :91); `affected_area_test.go` (+95) byte-shape 39/43 |
| B1.1d channel consumer wiring | DONE | `fff7668ee`; `services/atlas-channel/.../kafka/consumer/mist/consumer.go`, `kafka/message/mist/kafka.go` (mirrors fields), `consumer_test.go` |
| B1.2 chat `Multi` updateTime gate | DONE (gate corrected) | `25faf971c`,`773680c12`; `chat/serverbound/multi.go:54,71` gate = `GMS && >=95` (plan's `>83` was WRONG; v95 carries it, v87/v83/JMS do not — IDA-confirmed); `multi_test.go` (+33) |
| B1.3 quest `nItemPos` | NO-OP (verdict, correct) | `55f534c1d`; `spike-quest-actions.md`: premise disproven across all 4 IDBs (no nItemPos in `CQuest::StartQuest` v83/v87/v95/JMS); autoStart gate confirmed not-inverted. Inserting it would corrupt every quest packet. No code change — correct |
| B1.4 quest `ActionRestoreLostItem` redesign | DONE | `54110c020`; `action_restore_lost_item.go:12-43` (count-prefixed `[]uint32`); handler `quest_action.go` iterates `sp.ItemIds()`; `action_restore_lost_item_test.go` |
| B1.5 EffectWeather JMS branch | DONE | `a0d62f5af`; `effect_weather.go:36-94` region-dispatched `encodeJMS`/`encodeGMS`/`decode*`; `effect_weather_test.go`; `spike-effectweather.md` |
| B2.1 NPC continue-conversation discriminator | DONE | `4fa1e3d52`; `npc_continue_conversation.go:14-79` (`bodyKindFor`: 3/14→text, 5/8/9→selection, 0/1/2/13→none; hardcoded `==2` removed); `npc_continue_conversation_test.go` (+26). (Pre-existing quest `// TODO` comments retained — not new stubs) |
| B2.2 hired-merchant serverbound decode + handler | DONE | `02998a29e`; `merchant/serverbound/operation.go` (full `Operation` struct+Decode, JMS 0x37, mode 0); `hired_merchant_operation.go` (TODO stub fully replaced with merchant-processor dispatch); `operation_test.go` |
| B2.3 merchant modes 1/8/11 disposition | DONE | `e01915c2f`,`c04dcde96`; `merchant/clientbound/operation.go:165+` `EntrustedShopUnknownChannel` mode-8 emitter, mode-11 constant; `operation_test.go`; `spike-merchant-mode1.md` (mode 1 client/KMS-only) |
| B5.1a JMS ShopOperationBuy | DONE | `24b7eac38`; `cash/serverbound/shop_operation_buy.go:40-98` region dispatch (`encodeJMS`/`encodeGMS`, GMS keeps ≤2 guards); `shop_operation_buy_test.go` (+62) |
| B5.1b gift | DONE | `32fb3b927`; `shop_operation_gift.go`, `_test.go` |
| B5.1c buy_couple | DONE | `adeb17828`; `shop_operation_buy_couple.go`, `_test.go` |
| B5.1d buy_friendship | DONE | `58b11f56d`; `shop_operation_buy_friendship.go`, `_test.go` |
| B5.1e rebate_locker_item | DONE | `4b9fe881b`; `shop_operation_rebate_locker_item.go`, `_test.go` |
| B5.1f JMS template op-byte map + interaction remaps | DONE | `2fae37786`,`38b31491b`; `template_jms_185_1.json:540-552` CashShopOperationHandle (BUY=3,GIFT=46/0x2E,REBATE=27/0x1B,BUY_COUPLE=30/0x1E,BUY_FRIENDSHIP=36/0x24); interaction MERCHANT_BUY=31/0x1F at :534 |
| B5.1g verify JMS routing into wallet | DONE | `a8ea28bfb` + follow-up `b70b07079`; `cashshop/processor_test.go`, `producer_test.go`; `resolvePurchaseCurrency` maps JMS isPoints→MaplePoints(2) |
| B3.1 messenger serverbound enum | DONE (verdict) | `f873b562c`; `spike-messenger.md`: VERIFIED-NO-FIX, enum {0,2,3,5,6} |
| B3.2 messenger declineMode | DONE (verdict) | `f873b562c`; `spike-messenger.md`: VERIFIED-NO-FIX |
| B3.3 npc shop clientbound modes | DONE (verdict) | `e2d347a1a`; `spike-npc-shop.md`: VERIFIED-NO-FIX |
| B3.4 npc shop serverbound op-bytes | DONE (verdict) | `e2d347a1a`; `spike-npc-shop.md`: VERIFIED-NO-FIX |
| B3.5 7 interaction sub-ops | DONE (verdict) | `eb257b18c`; `spike-interaction-subops.md`: per-sub-op verdicts |
| B3.6 social enum-drift four-version | DONE (verdict) | `e728fbf93`,`cf73e18a6`; `spike-social-enum-drift.md` (319 lines, four-version table); NOTE REFRESH=2 concern resolved no-bug |
| B4.1 v87 stat-Changed + ui-Lock gates | DONE | `2069586e6`; `stat/clientbound/changed_test.go` (+v87 assertions), `ui/clientbound/lock_test.go` (+v87 assertions); both gates confirmed correct against v87 IDB (v87 mirrors v83) |
| B6.1 login export + audit + verdicts | DONE | `600476c2a`,`6be00c221`,`739791be2`,`103a4c0fc`,`773680c12`; `login/serverbound/request.go` (GMS PartnerCode/unknown2 gate, Region=="GMS"), `request_test.go` (+79); `spike-login.md` (234 lines), `spike-login-harvest.md` (4-version harvest) |
| E1 regenerate four SUMMARY | DONE | `0e8a9f5c1`,`2f8fbf1a5`; all four `docs/packets/audits/*/SUMMARY.md` regenerated AFTER the A5 analyzer commit (0e8a9f5c1 > b3a01e423) |
| E2 curate `_pending` → registry | DONE | `9d3c99097`; `docs/packets/ida-exports/_pending.md` (accepted-exclusions registry + §9 follow-up), `docs/packets/audits/gms_v95/_pending.md` |
| E3 TOTAL + new-version guide | DONE | `d4af9b42f`; `gms_v95/TOTAL.md` ("BASELINE COMPLETE — zero open actionable deferrals"), `STARTING_A_NEW_VERSION_PASS.md` (13KB) |
| F1 verify gates | DONE (this audit re-ran) | builds/tests/vet green; nesting clean; closed items untouched; redis N/A. `docker buildx bake` not re-run by this audit (see Gaps) |
| F2 code review before PR | IN PROGRESS | this review + the parallel backend reviewer (`audit.md`) are the F2 step |

**Completion Rate:** 38/38 plan tasks accounted for (100%).
**Skipped without approval:** 0.
**Partial implementations:** 0 functional. (1 presentational gap — SUMMARY legend.)
**No-op / verdict outcomes (all correct, evidence-backed):** B1.3, B3.1–B3.6 (verdict spikes), plus the corrected B1.2 gate.

## Build & Test Results (re-run during this audit)

| Module | Build | Tests | Vet | Notes |
|---|---|---|---|---|
| libs/atlas-packet | PASS | PASS | PASS | full `go test ./...` green incl. all new byte-shape tests |
| tools/packet-audit | PASS | PASS | PASS | diff/cmd/atlaspacket targeted analyzer tests pass |
| services/atlas-maps/.../maps | PASS | PASS | (n/r) | mist producer_test green |
| services/atlas-channel/.../channel | PASS | PASS | (n/r) | handler/cashshop/mist-consumer tests green; no FAIL |
| services/atlas-cashshop/.../cashshop | PASS | PASS | (n/r) | wallet + cashshop tests green (JMS currency follow-up) |

- Nesting-cap guard: no encoder/decoder in the changed region-dispatched files
  (effect_weather, all 5 cash bodies, affected_area_created, chat multi) exceeds 2 nested guards.
- Closed-item regression guard: `git diff main...HEAD` shows storage Show, MonsterControl,
  SETFIELD/WarpToMap files untouched (empty diff).
- redis-key-guard: N/A — no redis files changed on the branch.

## PRD §10 Acceptance-Criteria Coverage

| Criterion | Status | Evidence |
|---|---|---|
| B1.1–B1.5 wire bugs + byte tests + gates; AffectedArea RECT/tStart; EffectWeather JMS | MET | B1.1a–d, B1.2, B1.4, B1.5 landed with tests; B1.3 correctly no-op |
| B2.1–B2.3 handler fixes; continue-conversation routing; hired-merchant; mode 8 (1/11 dispositioned) | MET | B2.1/B2.2/B2.3 landed with tests + spike |
| B3.1–B3.6 verification deferrals → verdicts; social four-version enum-drift | MET | 6 IDA-grounded verdict spikes; no real divergence found |
| B4.1 v87 stat-Changed + ui-Lock gates confirmed | MET | v87 byte assertions added; gates confirmed correct |
| B5.1 JMS cash bodies + template remaps + wallet routing; interaction remaps; no 3rd nested guard | MET | 5 JMS bodies + template + routing + currency follow-up; nesting clean |
| B6 login export + audit + verdicts; bare handlers; v87 quirks | MET | login harvest+verdict spikes; PartnerCode gate; backlog dispositioned |
| §4.7 analyzer enhanced; clean four-version re-run (no spurious from named classes) | PARTIAL | analyzer enhanced (A1–A5) + tests; BUT regenerated SUMMARYs still show the named-class markers (see Gap 1). Residue IS in the registry, so the de-noising intent is recorded, but the SUMMARY output itself is not legend-annotated |
| §4.8 four SUMMARYs + zeroed `_pending` (both) + baseline-complete TOTAL + guide | MET | all four SUMMARYs regenerated; both `_pending` curated to accepted-exclusions; TOTAL states baseline-complete; new-version guide added |
| Closed items untouched + green | MET | empty diff on guarded files |
| All build/verify gates pass (test-race, vet, build, bake, redis-guard) | MOSTLY MET | test/vet/build green; nesting clean; redis N/A. `docker buildx bake` per touched go.mod (atlas-maps, atlas-channel, atlas-cashshop, atlas-configurations) NOT executed during this audit — see Gap 2 |
| Code review run before PR | IN PROGRESS | this audit is part of F2 |

## Gaps / Findings (adversarial)

1. **SUMMARY residue is dispositioned but not legend-annotated (minor, presentational).**
   Each `docs/packets/audits/*/SUMMARY.md` still shows ~77–85 `❌`/`🔍` rows, several of which
   are exactly the PRD §4.7 named false-positive classes (`CharacterViewAllCharacters`,
   `StorageUpdateAssets`, `InventoryAdd`/`InventoryChangeBatch`, `MessengerAdd`,
   `GuildBBSThreadList`). The analyzer enhancements (A1–A5) did NOT flip these to ✅ in the
   regenerated output. They ARE accounted for as accepted permanent exclusions in
   `docs/packets/ida-exports/_pending.md` §3/§4/§5 (OPAQUE + REPRESENTATION, with IDA
   evidence), and `TOTAL.md` §5 makes the baseline-complete statement — so PRD §4.8 ("residue
   in the registry with justification") is satisfied. The literal PRD §4.7 wording ("a fresh
   run emits no spurious ❌/🔍") is not met at the SUMMARY layer, because the named classes are
   register-boundary OPAQUE types the analyzer intentionally does not chase (a documented Q4
   design decision), not bugs. **Impact: low.** **Recommendation:** add a one-line legend to
   each SUMMARY header pointing readers to `_pending.md` so a `❌`/`🔍` is not mistaken for an
   open action. Not a merge blocker.

2. **`docker buildx bake` not executed in this audit.** Plan F1 Step 5 mandates a bake for
   every service whose `go.mod` was touched (atlas-maps, atlas-channel, atlas-cashshop, and
   atlas-configurations if its go.mod changed). This read-only audit did not run docker.
   `go build`/`go test` are green, but per CLAUDE.md only `bake` catches a missing shared-lib
   `COPY` in the root Dockerfile. No new `libs/` were added (all changes are inside existing
   `libs/atlas-packet`), so the risk of a missing COPY line is effectively zero here — but the
   gate should still be run before merge if not already done in the execution session.

3. **No functional gaps found.** Every code task has landed code + a byte-level test; every
   verdict task has an IDA-grounded spike note with addresses; the two real bugs surfaced
   mid-task (JMS isPoints→currency, BuddyInvite extra fields) were NOT buried — one is fixed
   on-branch (`b70b07079`) and the other is explicitly registered as a follow-up in
   `_pending.md` §9 ("Surfaced as a follow-up task (NOT accepted here)").

## Overall Assessment

- **Plan Adherence:** FULL (functional). One minor presentational deviation in SUMMARY rendering.
- **Recommendation:** READY_TO_MERGE pending (a) the `docker buildx bake` gate from the
  execution session and (b) optionally the SUMMARY legend nicety. No code rework required.

## Action Items

1. (Optional, low) Add a legend line to the four `SUMMARY.md` headers cross-referencing
   `docs/packets/ida-exports/_pending.md` so residual `❌`/`🔍` rows read as blessed exclusions.
2. (Required if not already done in execution) Run `docker buildx bake atlas-maps`,
   `atlas-channel`, `atlas-cashshop` (+ `atlas-configurations` if its go.mod changed) from the
   worktree root before opening the PR (CLAUDE.md mandatory gate).
3. (None functional) No skipped or partial implementation tasks to remediate.
