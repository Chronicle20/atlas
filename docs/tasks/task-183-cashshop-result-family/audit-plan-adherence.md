# Plan Audit — task-183-cashshop-result-family

**Plan Path:** docs/tasks/task-183-cashshop-result-family/plan.md
**Audit Date:** 2026-07-27
**Branch:** task-183-cashshop-result-family
**Base Branch:** cdfb71aa317cbec966709a85f5811cb886a18bb5 (merge-base with main)

## Executive Summary

The plan's four waves (RE catalog, modern-5 codecs, modern-5 verification, legacy + close-out) were implemented in full and are well-grounded: 57 arms (9 pre-existing + 48 new — a documented recount of the plan's "56/47" prose miscount) are each a discrete struct with `run.go` `#`-entries, config-resolved mode/reason bytes, and `packet-audit:verify` markers/evidence for every applicable version. All 81 plan checkboxes are unchecked in `plan.md` (the executor never ticked them), but the git history (37 commits) and code state independently confirm the work was done — checkbox state is a documentation gap, not an implementation gap. Builds, tests, `go vet`, and every `packet-audit` gate (`dispatcher-lint`, `operations --check`, `gate-check --check`, `matrix --check`, `fname-doc --check`) pass clean, as do the repo-root guards (`tools/lint.sh --check` scoped to the two touched modules, `redis-key-guard.sh`, `goroutine-guard.sh`, `template-opcode-order-guard.sh`). The one genuine gap: Task 4.2 (code review before PR) ran and produced real findings — including a `packet-completeness-critic` finding that a **pre-existing, now-disproven** `packet-audit:verify` marker (`CashCashItemMovedToInventory`/`gms_v72`, from task-096/PR#971) should be fixed on this branch per the "no deferring producible work" rule — but the finding was left unaddressed and the three review artifacts (`audit-backend.md`, `audit-backend.json`, `completeness-critic.md`) are still uncommitted/untracked, so the tree is not clean as Task 4.2 Step 3 requires.

## Task Completion

Plan has 81 `- [ ]` checkboxes across Waves 0–4; **0 are checked**. Status below is evidence-based (git log/diff/code), not checkbox-based.

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 0.1 | Author coverage-manifest.yaml | DONE | `docs/tasks/task-183-cashshop-result-family/coverage-manifest.yaml` matches plan content verbatim (ops/versions/out_of_scope). Commit `55fee5d7a`. |
| 0.2 | arm-catalog.md skeleton + v95 reference | DONE | `arm-catalog.md` created, 68 markdown table rows (57 arm rows + notes), v95 column seeded from Appendix A. Commit `76fb4efd2`. |
| 0.3 | RE v95 dispatcher + all arms, IDB naming | DONE | Read-order rows for all 57 arms present with decompile addresses (e.g. `arm-catalog.md:20` `BUY_FAILED … reason:Decode1@0x496a24`). Commits `a0c10da4b`, `201fe6aa1`, `00bc869a5`, `859ab8735`, `864cc99d5`. IDB `idb_save` steps not independently verifiable from this worktree (no IDA access in audit context) — code/catalog evidence is internally consistent and decompile-cited throughout. |
| 0.4 | RE modern-4 columns + yaml extension | DONE | `docs/packets/dispatchers/cash_shop_operation.yaml` has 57 `- key:` entries, each with a `modes:` map; modern versions populated. Commit `794b61c41`. |
| 0.5 | RE legacy-4 columns + yaml legacy modes | DONE | yaml `gms_v48`/`gms_v61` mode counts (40/57, 44/57 arms present respectively) are non-trivial subsets consistent with D2's "structurally divergent, subset + n-a" expectation; `n-a` proven by enumeration citations in `arm-catalog.md` "Per-version wire divergences" section. Commit `794b61c41` (consolidated). |
| 0.6 | Catalog review checkpoint (hard gate) | DONE | `arm-catalog.md` "Count note" section (lines 68–77) documents a self-review reconciling the 56/47→57/48 discrepancy rather than silently proceeding; `operations --check` was run at this stage per commit sequence (yaml exists before Wave 1 codec commits begin). |
| 1.1 | Failure-arm archetype + repeat (`_failed.go`) | DONE | `shop_operation_result_failed.go` (899 lines) — one discrete struct per failure arm, `packet-audit:fname` comments, config-resolved mode+reason. Commit `35168437e`. |
| 1.2 | Counter-arm archetype + repeat (`_slots.go`) | DONE (with a real, documented deviation) | `shop_operation_result_slots.go` implements `IncTrunkCountSuccess`/`IncCharacterSlotCountSuccess`/`IncBuyCharacterCountSuccess`/`EnableEquipSlotExtSuccess` as `mode+uint16` (NOT the plan's worked example `mode+type+uint16`) — documented in a file-header comment (`shop_operation_result_slots.go:15-22`): "RE-proven shape: mode + uint16 absolute-counter update — NO separate inventory/slot-type byte (contra `InventoryCapacitySuccess`'s mode+type+uint16 shape)." This is exactly the kind of RE-driven correction the plan authorizes (arm-catalog.md is the wire-truth source; the plan's Step 3 worked code is explicitly a copy-template, not RE-verified). Commit `762ddbad7`; v72 shape variant added later in `ea62f1c3a`. |
| 1.3 | Item-blob archetype + 0x4D TODO resolution (`_gift.go`) | DONE (with a real, documented deviation) | `GiftDone` (GIFT_SUCCESS) is a pure scalar body (`mode+recipientName+itemId+quantity+[nxCashSpent]`), NOT an item-blob, contra the plan's shape-group hypothesis — documented in-code: "TRUE SHAPE per task-0.3e report: NO item-blob at all — pure scalar body. This resolves the legacy 0x4D gift TODO (the old CashShopGifts stub modeled the wrong shape entirely)" (`shop_operation_result_gift.go:94-101`). `0x4D` literal + `// TODO map codes for JMS` + `CashShopGifts` struct + `CashShopCashGiftsBody` func are all deleted (verified: `grep -n "type CashShopGifts\|func NewCashShopGifts\|func CashShopCashGiftsBody"` → no matches). Commit `a0d046077`. |
| 1.4 | Scalar/notice/transfer/gachapon archetype | DONE | `shop_operation_result_misc.go`, `_transfer.go`, `_gachapon.go` created with all remaining arms (`LIMIT_GOODS_COUNT_CHANGED`, `DESTROY_*`, `EXPIRE_DONE`, `PURCHASE_RECORD*`, `FREE_CASH_ITEM_DONE`, `NAME_CHANGE_BUY_DONE`, `TRANSFER_WORLD_*`, `GACHAPON_*`, `CHANGE_MAPLE_POINT_*`). Commit `0281f8137`. |
| 1.5 | Version-gate modern field divergences | PARTIAL (justified) | jms `nxCashSpent` region-gate added (`giftHasNxCashSpent`, commit `1d2734ff4`) but its `gates.yaml` registration was attempted and abandoned as structurally impossible for dispatcher-arm packets — see "Skipped / Deferred Tasks" below. `IncBuyCharacterCountSuccess` v72 one-version gate added (`ea62f1c3a`) and correctly NOT registered in `gates.yaml` (that file's schema is for two-adjacent-version boundary gates, not one-version exact-match divergences — confirmed by reading `gates.yaml`'s header doc). |
| 1.6 | dispatcher-lint + package tests green | DONE | `go run ./tools/packet-audit dispatcher-lint` → `dispatcher-lint: clean` (verified live, this audit). `go test ./cash/... -count=1` → PASS (verified live). |
| 1.7 | Regenerate modern template operations maps | DONE | `template_gms_{83,84,87,95}_1.json`, `template_jms_185_1.json` all modified (diffstat confirms), `operations --check` passes clean (verified live). Commit `f3f9514f2`. |
| 2.1 | Verify modern arms, batched per IDB | DONE | 464 `packet-audit:verify` marker lines across 16 test files in `cash/clientbound`; evidence records present under `docs/packets/evidence/*/cash.clientbound.*.yaml` (diffstat shows new evidence files for jms, and modified/new for other versions). Commits `f97c6972e` through `8f6f24f1f`. |
| 2.2/2.3 (matrix + gate-check) | DONE | `matrix --check` and `gate-check --check` both pass clean (verified live, see Build & Test Results). |
| 3.1 | Legacy codecs + version gating | DONE | Legacy arm tests present (`shop_operation_result_failed_test.go` contains `"GMS/v48": 0x44` etc. per-version mode maps); legacy-specific gates documented in-code (goodsSN non-modeling, v72 shape). Commits `8c06e3a4f`, `848919646`. |
| 3.2 | Regenerate legacy templates + verify legacy cells | DONE | `template_gms_{48,61,72,79}_1.json` modified; v48 `CashShopOperation` writer added at opcode 0x100 (commit `5ba6b2685`, confirmed live: writer present in `template_gms_48_1.json`). `operations --check` passes for all 9 versions (verified live). Legacy verification commits `bb49ee54e`, `b3bb1997d`. |
| 4.1 | Full verification suite | DONE | See Build & Test Results — all commands re-run live for this audit and pass. |
| 4.2 | Code review before PR | PARTIAL | Reviews were run (`audit-backend.md`/`.json`, `completeness-critic.md` exist and are substantive — backend audit is a clean PASS; completeness-critic found 1 real actionable issue). But: (a) the completeness-critic's actionable finding (stray disproven `CashCashItemMovedToInventory`/gms_v72 verify marker) was **not fixed on-branch**, contradicting the plan's own Step 2 ("Address findings on-branch — no deferral") and CLAUDE.md's "No Deferring Producible Work"; (b) the three review artifacts are uncommitted/untracked, so Step 3's "confirm the tree is clean" is currently false (`git status --short` shows 3 untracked files). See "Skipped / Deferred Tasks" below. |
| Op-row re-graduation (D3/D4 close-out) | DONE | `docs/packets/evidence/families.yaml` no longer lists `CCashShop::OnCashItemResult` as an active `dispatchers:` entry (comment-only historical record; confirmed via `grep -n "^  [A-Za-z]"` → zero live entries). `PROCESS.md:114` `family_cap_dispatchers: []` with a note "(CCashShop re-graduated task-183: all 57 arms verified)". Matches design §5.1/D3 expected end state (un-graduate mid-task for honest tracking, re-graduate at close-out) — confirmed via commits `a21a0834b` (un-graduate) → `698e4fca3` (re-graduate). |

**Completion Rate:** 21/23 tracked sub-tasks DONE, 2/23 PARTIAL (both justified/explained, one with a real unresolved action item), 0 SKIPPED, 0 DEFERRED-without-explanation.
**Skipped without approval:** 0
**Partial implementations:** 2 (Task 1.5 gate registration — architecturally blocked, well-documented; Task 4.2 — review ran but its finding wasn't actioned and artifacts uncommitted)

## Skipped / Deferred Tasks

### Task 1.5 — nxCashSpent gate registration (DEFERRED, architecturally justified)

`docs/packets/gates.yaml` carries a ~40-line `# TODO(task-183 Wave 2, re-diagnosed)` comment (lines 169–208) explaining that a gate row for `cash/clientbound/CashGiftDone`/`CashGiftPackageDone`'s GMS/jms `nxCashSpent` divergence was attempted twice and rejected by `gate-check --check` with "no matrix row for packet …". The root cause is structural, not a missing fixture: `CASHSHOP_OPERATION` is a single dispatcher op-row (per `tools/packet-audit/internal/matrix/build.go`'s `worstCandidateCell`/`usedWriters` logic), so no individual dispatcher-arm packet (verified or not, before or after Wave 4 re-graduation) can ever satisfy `gate-check`'s per-packet row lookup — confirmed true of all 20+ other verified arms as well, not specific to this pair. The comment lays out three concrete resolution options (a/b/c) and states the byte-fixture test's own `v.Region == "GMS"` branch assertion is the de-facto verification in the interim. This is a genuine "stop-and-ask"-class architectural limitation in `packet-audit` itself (out of this task's scope to fix the tool), documented thoroughly rather than silently dropped, and the wire behavior itself (the region gate) IS implemented and tested. **Impact:** low — coverage is real (byte-fixture-verified per region), only the `gate-check` cross-check mechanism can't currently represent it. Recommend a small follow-up task against `packet-audit` (option a or b in the comment) rather than blocking this PR.

### Task 4.2 — completeness-critic finding not actioned (GAP — should be fixed before merge)

`completeness-critic.md` (present in the worktree, uncommitted) found: `libs/atlas-packet/cash/clientbound/v72_test.go:36` carries `// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToInventory version=gms_v72 ida=0x470e13` plus a byte-equality-vs-v83 test case and a pinned evidence record (`docs/packets/evidence/gms_v72/cash.clientbound.CashCashItemMovedToInventory.yaml`), asserting this arm is client-validated on `gms_v72`. This task's own Wave-0 RE work directly disproves that: `arm-catalog.md:179-184` states "v72 does not have this arm at all — `CASH_ITEM_MOVED_TO_INVENTORY` and `MOVE_L_TO_S_FAILED` are both `n-a` in v72 (proven by full switch enumeration + `func_query name_regex="MoveLtoS"` returning zero hits...)" — and the pre-existing per-packet audit report (`docs/packets/audits/gms_v72/CashCashItemMovedToInventory.json`) already independently records `"Verdict": 2` (fail), `"IDAComment": "function not found in IDB"`. I confirmed `v72_test.go:36` still carries the stray marker as of HEAD (`698e4fca3`) — it was never touched by this branch's commits despite the completeness-critic's explicit recommendation to fix it "rather than deferring, per the no-deferring-producible-work rule." **Impact:** low-severity but real — a false "client-validated" claim sits in the test suite for an arm this task itself proved doesn't exist on gms_v72; leaving it contradicts the branch's own grounding work and CLAUDE.md's explicit anti-deferral rule. This predates task-183 (task-096/PR#971) but task-183 is the party that discovered and documented the contradiction, so per CLAUDE.md ("attempt the unblock before calling it terminal... the user should not have to prod you to finish bounded work") it should have been fixed here.

Separately, the three review artifacts (`audit-backend.md`, `audit-backend.json`, `completeness-critic.md`) are untracked (`git status --short`) — Task 4.2 Step 3 ("confirm the tree is clean") is not currently satisfied.

## Build & Test Results

| Module | Build | Vet | Tests | Notes |
|--------|-------|-----|-------|-------|
| libs/atlas-packet | PASS | PASS | PASS | `go build ./...`, `go vet ./...`, `go test ./... -count=1` all clean; `cash/clientbound` and `cash/serverbound` both `ok`. |
| tools/packet-audit | PASS | PASS | PASS | `go build ./...`, `go vet ./...`, `go test ./... -count=1` all clean across all 12 internal packages. |
| services/atlas-channel (atlas.com/channel) | PASS | — | — | Only a dead-comment removal; `go build ./...` clean. No go.mod change → no `docker buildx bake` required (confirmed: `git diff --name-only <base>...HEAD \| grep go.mod` empty). |

| Gate | Result | Notes |
|------|--------|-------|
| `packet-audit dispatcher-lint` | PASS | `dispatcher-lint: clean` |
| `packet-audit operations --check` | PASS | `operations check OK (0 absent-writer note(s))` |
| `packet-audit gate-check --check` | PASS | `gate-check: all 19 gate(s) have verified byte-fixtures on both straddling versions (0 partial-by-design)` — nxCashSpent intentionally not among the 19 (see above). |
| `packet-audit matrix --check` | PASS | Clean; one unrelated pre-existing `n-a evidence consumed` note (USE_TELEPORT_ROCK × gms_v48, not part of this task). |
| `packet-audit fname-doc --check` | PASS | `fname-doc check OK (243 structs without an audit report carry no fname)` |
| `tools/lint.sh --check --go libs/atlas-packet tools/packet-audit` | PASS | `lint.sh: OK`, 0 issues (scoped to the two touched modules per the tool's documented `path...` restriction, since a full tree-wide run exceeded the practical timeout in this audit session; formatting is enforced tree-wide regardless of scope). |
| `tools/redis-key-guard.sh` | PASS | Exit 0. |
| `tools/goroutine-guard.sh` | PASS | Exit 0. |
| `tools/template-opcode-order-guard.sh` | PASS | `OK: 22 template arrays are in ascending opcode order.` |

## Deviation Assessment (per audit-request items 1–4)

1. **57 arms / 48 new vs. plan's "56/47".** Confirmed real and honestly documented: `arm-catalog.md`'s "Count note" section states the design/plan prose was a miscount, the Appendix A table itself has 57 rows, and the Wave-1 task key-lists total 48 new keys. All 57 catalog operation keys are present in both `cash_shop_operation.yaml` (57 `- key:` entries) and `tools/packet-audit/cmd/run.go` (57 `OnCashItemResult#` case entries). This is a documented recount, not scope drift — **justified**.

2. **Counter/GIFT_SUCCESS/goodsSN shape deviations from the plan's worked examples.** All three confirmed real in the code:
   - Counter arms are `mode+uint16` (not `mode+type+uint16`), documented in a file-header comment citing the RE report and explicitly contrasting with the plan's `InventoryCapacitySuccess` reference shape.
   - `GiftDone` (GIFT_SUCCESS) is scalar, not item-blob, documented in-struct as "TRUE SHAPE per task-0.3e report."
   - The conditional `goodsSN` field on several failure arms (`BUY_FAILED`, `COUPLE_FAILED`, `BUY_PACKAGE_FAILED`, `GIFT_PACKAGE_FAILED`, `BUY_NORMAL_FAILED`, `FRIENDSHIP_FAILED`) is documented as intentionally unmodeled — "atlas emits only the base mode+reason (no producer sends goodsSN — non-goal per task-183 design decision)" — repeated consistently across `arm-catalog.md`, the struct doc comments, and the test file comments.
   All three are RE-driven corrections to the plan's pre-RE worked-example code, consistent with the plan's own framing (arm-catalog.md is "the sole source of wire truth," Task 1.1's worked archetype is explicitly a copy-template not a byte-exact spec) — **justified, not silent scope-narrowing**.

3. **nxCashSpent gate registration.** Confirmed real, deferred, and thoroughly documented (see "Skipped / Deferred Tasks" above) — a genuine tool-architecture limitation, not a silent drop. **Acceptable as documented**, though it represents unfinished tooling work that should be tracked.

4. **Op-row graduation state.** Confirmed matches the expected end state: un-graduated mid-task (`a21a0834b`) for honest tracking while 48 unverified arms were added, re-graduated at close-out (`698e4fca3`) once all 57 arms had verified-or-n-a cells across all applicable versions. `families.yaml` and `PROCESS.md` both reflect the final graduated state consistently. **Matches design D3/§8 exactly.**

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE
- **Recommendation:** NEEDS_FIXES

The core deliverable (57 discrete arm codecs, full RE grounding, full verification, template regeneration, matrix re-graduation) is complete, correct, and well-tested. The gap is process, not substance: a review step (Task 4.2) surfaced one concrete, previously-undetected defect (a false verify marker this task's own RE work disproves) and that finding was not closed out before this audit, and the review artifacts themselves were never committed.

## Action Items

1. **Fix the stray v72 verify marker** (completeness-critic finding, not yet actioned): remove `// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToInventory version=gms_v72 ida=0x470e13` from `libs/atlas-packet/cash/clientbound/v72_test.go:36`, remove the corresponding `"MovedToInventory"` case from `TestCashShopOperationArmsV72`'s arm table (`v72_test.go:58`), and delete/mark-n-a `docs/packets/evidence/gms_v72/cash.clientbound.CashCashItemMovedToInventory.yaml`, then re-run `matrix --check` to confirm the arm correctly falls to n-a rather than a false-verified state.
2. **Commit the review artifacts**: `docs/tasks/task-183-cashshop-result-family/audit-backend.md`, `audit-backend.json`, and `completeness-critic.md` are untracked; commit them (or fold their content into the PR description) so `git status --short` is clean per Task 4.2 Step 3, and so the review trail is preserved.
3. **Optional / follow-up (not a merge blocker):** file a small follow-up task against `tools/packet-audit` to teach `gate-check` to fall back to the evidence/marker ledger for packets with no matrix row (option (a) in the `gates.yaml` TODO comment), so the `nxCashSpent` region gate (and any future intra-dispatcher-arm gate) can be machine-checked rather than relying solely on the byte-fixture test's own region assertion.
4. **Optional housekeeping:** the plan's 81 checkboxes are all unchecked despite the work being done — not required for merge, but worth ticking (or noting as N/A with a pointer to this audit) so `plan.md` doesn't read as 0% complete to a future reader.
