# Backend Audit — task-113 GMS v79 legacy-versions pass

## v79 backend-guidelines

- **Scope:** 28 changed Go source files (22 `libs/atlas-packet/*` codecs, 3 `services/atlas-channel` writer seams, 2 `tools/packet-audit` files) + their fixture tests. Diff base `main` (merge-base `b0e526a60`).
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/SEC-*)
- **Date:** 2026-07-01
- **Build:** PASS — `libs/atlas-packet`, `services/atlas-channel/.../channel`, `tools/packet-audit` all `go build ./...` clean; `go vet ./...` clean on atlas-packet.
- **Tests:** PASS — atlas-packet 67 pkg ok / 0 fail; 14 touched codec packages ok; channel `socket/writer` ok; packet-audit all internal pkgs ok.
- **Overall:** PASS (NEEDS-WORK: 0 blocking; 2 minor/non-blocking).

### Build & Test Results

```
libs/atlas-packet          go build ./... OK   go vet ./... OK   go test ./... 67 ok / 0 FAIL
services/atlas-channel     go build ./... OK   go test ./socket/writer OK
tools/packet-audit         go build ./... OK   go test ./... all ok (incl. TestEveryVersionKeyHasTemplateFile)
```

### Applicability note

The changed files are shared codec (`libs/atlas-packet`) and thin channel writer seams — not DDD domain packages. There is no `model.go`/`processor.go`/`resource.go`/`administrator.go` in scope, so DOM-01..DOM-20, SUB-*, EXT-*, and SCAFFOLD-* are N/A. The load-bearing checks for this change are: version-gate correctness (no off-by-one; existing versions unchanged), DOM-21 (shared predicates), immutability/getter conventions, and SEC (login-adjacent codec).

### Findings

**Critical:** none.

**Important:** none.

**Minor / non-blocking:**

- **M1 — Region-less `MajorVersion()`/`MajorAtLeast(83)` gates (style/intent).**
  `character/clientbound/attack.go:106,166`, `drop/serverbound/pick_up.go:26`,
  `summon/clientbound/attack.go:73,102`, `summon/clientbound/spawn.go:157` gate
  purely on major version (`>= 83` / `MajorAtLeast(83)`) with no `Region()` scope,
  whereas the sibling gates in this same change (`damage_info.go`, `change.go`,
  `list.go`, `status_message.go`, `heal_over_time.go`) scope with
  `Region()=="GMS"`. All are functionally correct because the only sub-83 tenant is
  GMS v79 and JMS is v185 (`>=83` true), so existing behavior is unchanged — but the
  Region-less form is marginally less intention-revealing for the off-by-one class
  this task is guarding against. Not a defect; cite for consistency only.

- **M2 — Mixed tenant-extraction style in `npc/clientbound/conversation.go`.**
  `AskMenuConversationDetail.Encode` (line ~196) uses the non-panicking
  `tenant.FromContext(ctx)()` with `err == nil` (safe default = v83 plain-string
  path if tenant missing), while `AskMemberShopAvatarConversationDetail.Encode`
  (line ~290) uses `tenant.MustFromContext(ctx)` (panics if absent). Both are
  acceptable; the inconsistency is cosmetic.

### Checklist results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| BUILD | go build/vet clean, all modules | PASS | atlas-packet/channel/packet-audit build+vet clean |
| TEST | go test clean, existing fixtures pass | PASS | 67/0 atlas-packet; 14 touched codec pkgs ok |
| GATE-01 | No existing-version (v83/84/87/95/jms) behavior changed | PASS | Every gate narrows to `<83` (legacy) or lowers a `>=83` bound to `>=79`; v83+ paths byte-identical. Pinned `packet-audit:verify` v83/v95/jms fixtures still present and passing (`spawn_test.go` 3 markers, `list_test.go`, `attack_test.go`). |
| GATE-02 | admin_result v83 arm preserved | PASS | `field/clientbound/admin_result.go:163` old `MajorVersion()<84` narrowed to `>=83 && <84`; v83 output identical, new `<83` arm added separately (`:179`). |
| GATE-03 | spawn dragon-effect + team bytes unchanged for v83+ | PASS | `character/clientbound/spawn.go:153` `<=87` → `>=83 && <=87` (v83/84/87 still true, v95 still false); team byte `Region!="JMS" && !legacy` keeps GMS>=83 writing. |
| GATE-04 | change.go / damage_info.go lowered bound safe | PASS | `field/serverbound/change.go:77` and `model/damage_info.go:48,79` `>=83`→`>=79`; v83+ still true, only adds v79. |
| DOM-21 | Uses shared tenant predicates, no reinvented version logic | PASS | All gates via `tenant.Model` `Region()`/`MajorVersion()`/`MajorAtLeast()`/`IsRegion()` (`libs/atlas-tenant/tenant.go:21,25,93`). |
| IMMUT | Private fields + getters + constructors | PASS | `login/serverbound/after_login.go:31` adds private `accountId` + getter `AccountId()` (`:31`); monster codecs add private `uniqueId` + getter, constructors updated; no test-only constructors. |
| SEC-CLIENT-TRUST | Client-supplied AfterLogin.AccountId() not trusted for authz | PASS | Login handler `services/atlas-login/.../socket/handler/after_login.go:40` uses `s.AccountId()` (session, set at auth), never `p.AccountId()`; the new packet getter is used only for wire round-trip in `after_login.go:61`. Handler file unchanged by this task. |
| SEC-04 | No hardcoded secrets | PASS | None introduced; codec/version-gate only. |
| CHANNEL | Writer seams thread new param, gated in codec | PASS | `catch_monster.go`/`inc_mob_charge_count.go`/`monster_special_effect_by_skill.go` add `uniqueId` param; codec `legacyMobPoolPrefix` (GMS `<83`) is the only emit gate. |

### Scope-decision observations (not defects)

- `legacyMobPoolPrefix` (`monster/clientbound/*.go`) emits the per-mob `uniqueId`
  prefix only for GMS `<83`. The code comment flags that sibling per-mob packets
  (MonsterHealth op228, MonsterMovement op217) carry this prefix unconditionally,
  so v83+ likely need it here too — deliberately frozen per campaign scope. No
  regression: all three writers are unwired seams ("No emitter wires this writer
  yet"), so nothing emits these on any version today.

---

## v79 plan-adherence

- **Reviewer:** plan-adherence-reviewer
- **Date:** 2026-07-01
- **Branch:** task-113-gms-legacy-versions (HEAD `800938260`)
- **Scope:** Phase 1 (Pass v79), Stages A–F + J. Runtime stages G/H/I are DEFERRED per the recorded owner decision (`.superpowers/sdd/progress.md`) and are **not** flagged. Serverbound expanded to full v83 parity (~112 handlers) per owner decision (Task 1.B2).
- **Method:** live commands (`matrix --check`, `status.json` count diff, `go build/vet/test`, guard tests) — not prose assertion.

### Per-stage verdicts

| Stage | Deliverable | Verdict | Evidence |
|---|---|---|---|
| A — anchor & delta | `v79-packet-delta.md` (+char-mgmt CORRECTION) | **PASS** | File present (27,961 B). Char-mgmt rotated-symbol CORRECTION present: `v79-packet-delta.md:100-116` (NAME=13/ADD=14/DEL=15 == v83 by decompiled body @0x5ce875/0x5ceb55/0x5ce90a); registry carries the corrected opcodes + provenance notes (`gms_v79.yaml:96,104,112`). |
| B — registry | `gms_v79.yaml` + `discover_gms_v79.md` | **PASS** | Both present. Direction counts: `grep -c direction: clientbound`=**263**, `serverbound`=**112** — matches expected 263 cb / 112 sb. Worklist `discover_gms_v79.md` (13,627 B) present. |
| C — template | `template_gms_79_1.json` | **PASS** | `usesPin=false`; 112 handlers, **0 missing validators**; 151 writers, **18** with populated `options.operations` tables. |
| D — export & audit | `gms_v79.json` + `audits/gms_v79/SUMMARY.md` | **PASS** | Export present (361,059 B). `SUMMARY.md` present (68,299 B) and declares "Zero open actionable deferrals"; ❌/🔍 rows dispositioned via `_pending.md`. No `_unimplemented.json` (none claimed necessary). |
| E — matrix + campaign | `gms_v79` matrix column + fixtures/evidence | **PARTIAL — advance gate NOT fully met** | See detail below. |
| F — code-gate slice | `code-gate-audit.md` v79 column | **PASS** | 167 gate rows (file:line), **0 with an empty v79 column**. Header confirms v79/v72/v61/v48 columns (`:46`). (152 base + campaign-gate rows; ledger `1.F`.) |
| J — build gate | build/vet/test -race clean | **PASS** | `libs/atlas-packet`: build=0, vet=0, test=0 (0 FAIL). `tools/packet-audit` guard tests `TestEveryVersionKeyHasShortLabel|TestFnamedocOrderCoversVersionKeys` exit=0. `redis-key-guard` is pre-existing-broken on `main` (minio go.sum baseline) — no new violations (ledger `1.J`). |

### Existing-version isolation — PASS

`verified` counts are **identical** at pass-start `911fd93a1` vs current HEAD:

```
version    pass-start  current
gms_v83       367        367
gms_v84       345        345
gms_v87       379        379
gms_v95       399        399
jms_v185      361        361
gms_v79         0        165   (new column populated this pass)
```

`go run ./tools/packet-audit matrix` regeneration produces **no diff** to committed `STATUS.md`/`status.json` (no drift); working tree clean apart from this audit file.

### Stage E — the one gate not fully met

**Integrity half: PASS.** `go run ./tools/packet-audit matrix --check` → **exit 0**; `grep -ciE 'orphan|dangling|stale|drift|unresolv|malformed'` = **0**; zero conflict/🟥 lines in the v79 column.

**"No in-scope tier-1 ❌" half: GAP.** The current `status.json` still carries **23 tier-1 cells** in state `incomplete` with `verdict ❌` for `gms_v79`:

- cash/serverbound: `CashShopOperationBuyNameChange`, `BuyWorldTransfer`, `MoveFromCashInventory`, `RebateLockerItem`
- interaction/serverbound: `InteractionOperationMerchantBuy`, `MerchantPutItem`, `PersonalStoreAddToBlackList`, `PersonalStoreBuy`, `PersonalStorePutItem`, `PersonalStoreSetBlackList`, `TradeConfirm`, `TradePutItem`, `Transaction`
- npc/clientbound: `NpcAskAvatarConversationDetail`, `NpcAskBoxTextConversationDetail`, `NpcAskNumberConversationDetail`, `NpcAskPetAllConversationDetail`, `NpcAskPetConversationDetail`, `NpcAskTextConversationDetail`, `NpcSayConversationDetail`, `NpcSayImageConversationDetail`
- npc/serverbound: `NpcShopBuy`, `NpcShopSell`

All 23 share `opcode=-1` (no top-level opcode) — they are per-mode leaf cells of the body-mode demuxer families that Stage B deliberately excludes as routed opcodes (NpcShop→CShopDlg, NpcAsk/Say→CScriptMan, InteractionOperation→CMiniRoomBaseDlg, CashShopOperation→CCashShop). Their **parent** dispatcher cell IS verified for v79 (e.g. `CASHSHOP_OPERATION`→`cash/serverbound/CashShopOperationGetPurchaseRecord` verified; `PLAYER_INTERACTION`→`interaction/.../InteractionInteractionChat` verified; `NPC_TALK_MORE`→`NpcAskMemberShopAvatarConversationDetail` verified).

Why this is a GAP and not a false alarm: **every one of these 23 leaf cells is `verified` in all sibling versions** (v83/v84/v87/v95/jms — spot v84 has 2 additionally ❌), so the project treats each mode as an independently byte-fixtured tier-1 cell everywhere else. For v79 only the representative mode of each family got a fixture; the remaining per-mode arms were left ❌. There is **no `_unimplemented.json` allowlist** and **no ledger note** dispositioning these 23 as out-of-scope. This is precisely the "dispatcher per-mode body must be byte-fixtured, mode enumeration is a false pass" constraint the plan's Global Constraints and `feedback_dispatcher_mode_byte_is_false_pass` call out.

The ledger (`.superpowers/sdd/progress.md`) declares "v79 TIER-1 in-scope todo = 0 (advance gate MET)" — that claim rests on `matrix --check` exit 0, which is the **integrity** check only and does not count `❌` leaf cells. By the plan's own literal Stage E gate ("no in-scope cell remains ❌ — every in-scope cell is ✅ / 🟡-with-evidence / ⬜"), 23 tier-1 cells remain ❌.

**Recommendation:** either write the per-mode byte fixtures for the 23 dispatcher-arm cells (bringing v79 to sibling-version parity), or explicitly disposition them in a `docs/packets/audits/gms_v79/_unimplemented.json` with per-mode justification so the gate is honestly closed. Do not close Stage E on `matrix --check` alone.

### Summary

- **PASS:** A, B, C, D, F, J, existing-version isolation, matrix integrity (`--check` exit 0, no drift, no conflicts).
- **GAP (advance gate not fully met):** Stage E — 23 tier-1 dispatcher per-mode leaf cells remain `❌` for v79 (verified in all sibling versions, no allowlist). Ledger's "advance gate MET" conflates matrix-integrity with the ❌-count criterion.
- Deferred G/H/I correctly excluded per owner decision.

---

## v79 Stage E re-review (2026-07-01)

Independent re-verification of the Stage E advance gate **only** (A/B/C/D/F/J and existing-version isolation were confirmed in the prior section and are not re-audited here), after the campaign was closed via 5 closing batches and documented in `v79-stageE-close.md`. The prior review's finding (23+ in-scope ❌ leaf cells, no disposition) has been remediated. Re-run of every command below independently confirms the gate is now met.

### Gate definition applied
Every in-scope cell (kind `op` or `sub-struct`; `tier1==true` OR packet under `login/`; v79 state `incomplete`/`conflict`; **and** `verified` in all of v83/v84/v87/v95/jms) must be either (1) an **op-row-verified artifact** — the same packet's `kind==op` gms_v79 row is `verified` and the ❌ is only the redundant `sub-struct` scoring row — or (2) a **justified `_unimplemented.json` disposition**. Any ❌ that is neither = a real gap.

### Enumeration (from `docs/packets/audits/status.json`)
- **28** in-scope v79 `incomplete` cells total; **12** are strict (also `verified` in all five existing versions). The other 16 are excluded because the packet is not yet `verified` in every existing version (shared cross-version incompleteness, not a v79-specific Stage E gap) — matches the exact enumeration criterion.
- All 12 strict residual cells are `kind==sub-struct` (matches `v79-stageE-close.md`'s residual-12 claim).

### Disposition of the 12 strict cells
**(1) op-row-verified artifacts — 5** (confirmed each packet's `kind==op` gms_v79 row is `verified`):

| packet | op | gms_v79 op-row |
|---|---|---|
| character/serverbound/Move | MOVE_PLAYER | verified |
| summon/serverbound/SummonAttackHandle | SUMMON_ATTACK | verified |
| npc/serverbound/NpcStartConversation | NPC_TALK | verified |
| character/serverbound/ExpressionRequest | FACE_EXPRESSION | verified |
| field/serverbound/FieldChange | CHANGE_MAP | verified |

**(2) `_unimplemented.json` dispositions — 7** (each has NO op row; each maps to an fname entry whose `reason` explicitly names the packet/mode-arm with a concrete v79 IDA basis):

| packet | `_unimplemented.json` fname key |
|---|---|
| cash/serverbound/CashShopOperationBuyNormal | CCashShop::OnBuyNormal |
| cash/serverbound/CashShopOperationBuyWorldTransfer | CCashShop::SendBuyTransferWorldItemPacket |
| cash/serverbound/CashShopOperationIncreaseStorage | CCashShop::OnIncTrunkCount |
| cash/serverbound/CashShopOperationMoveFromCashInventory | CCashShop::OnMoveCashItemLtoS |
| cash/serverbound/CashShopOperationMoveToCashInventory | CCashShop::OnMoveCashItemStoL |
| cash/serverbound/CashShopOperationRebateLockerItem | CCashShop::OnRebateLockerItem |
| npc/clientbound/NpcAskPetConversationDetail | CScriptMan::OnAskPet#AskPetConversationDetail |

`_unimplemented.json` holds 15 entries (the 7 above + 4 PIC char-select variants + AskSlideMenu + OnBuyNameChange + OnBuySlotInc). Matrix scoring skips `_`-prefixed files (`internal/matrix/load.go:25`); the allowlist is consumed by the `validate` command — so the sub-struct rows still render ❌, which is the documented, non-blocking tooling limitation, not an undisposed gap.

**Real gaps found: 0.**

### IDA spot-checks against the v79 IDB (GMS_v79_1_DEVM.exe, port 13340 — confirmed active)
- **CCashShop::OnBuyNormal @0x46a3b0** — decompile confirms `COutPacket(221)`@0x46a5dc, `Encode1(0x23)`@0x46a5ea, then `Encode4(v30)`@0x46a5f5 + `Encode4(v36 buyFlags)`@0x46a600 + `Encode4(a2 nBuyType)`@0x46a60b + `EncodeStr`@0x46a624 + `EncodeStr`@0x46a63d = mode 0x23, 3 ints + 2 strings. Structurally ≠ the Atlas ShopOperationBuyNormal codec (mode 0x20, single `Encode4(serial)`). Every address in the reason matches exactly. Genuine v79-mismatch.
- **MoveCashItem** — `func_query name_regex="MoveCashItem|LtoS|StoL"` over the v79 IDB returns ONLY `CITC::OnMoveITCPurchaseItemLtoSDone@0x57fefe` and `...Failed@0x57ff65` (clientbound CITC receive); no `CCashShop` LtoS/StoL sender exists. Genuine v79-absent. Reason confirmed.

### Machine checks
- `packet-audit matrix --check` → **exit 0**, zero problem lines, zero drift.
- v79 `conflict` cells in status.json → **0**.
- **Existing-version isolation:** vs `git show 911fd93a1:docs/packets/audits/status.json` (pass start), `verified` and `n-a` counts are **identical** for all five existing versions (v83 367/202, v84 345/223, v87 379/165, v95 399/81, jms 361/172). The only `incomplete`-count deltas are row-set changes: two packet **renames** (NpcAskMenuConversationDetail→NpcAskMemberShopAvatarConversationDetail; InteractionInteractionEnter→InteractionInteractionChat, both same op, `verified` preserved in all 5 versions) plus fname-list updates. Zero existing-version `verified` cell regressed or was newly promoted.

### Verdict
**Stage E gate: MET.** 12 strict in-scope residual ❌ = 5 op-row-verified artifacts + 7 `_unimplemented.json`-documented dispositions + **0 real gaps**. The prior review's 23-cell gap is closed (representative-mode-only fixtures replaced by full per-mode coverage + a documented allowlist). Matrix integrity clean, existing versions frozen, and the two IDA spot-checks confirm the allowlist reasons are accurate against the actual v79 binary.

---

# v72 backend-guidelines review (DOM/SUB/SEC)

- **Date:** 2026-07-01
- **Scope:** 14 changed Go source codec files under `libs/atlas-packet/*` + 4 `tools/packet-audit/*` files (v72 legacy tenant gates). Range `d7b2f69c0..HEAD`.
- **Verdict:** PASS (no blocking findings). Existing-version behavior provably unchanged.

## Build & Test (worktree)
- `libs/atlas-packet`: `go build ./...` OK, `go vet ./...` OK, `go test ./... -count=1` = 67 packages ok, 0 failures.
- `tools/packet-audit`: `go build ./...` OK, `go test ./... -count=1` 0 failures.

## Version-gate correctness (every new/changed gate)
Confirmed `MajorAtLeast(v) ≡ MajorVersion() >= v` (`libs/atlas-tenant/tenant.go:25,93`), so `MajorVersion() < 79 ≡ !MajorAtLeast(79)`. All gate equivalences hold.

| Codec (file:line) | Gate | v72 | v79/83/84/87/95/jms | Verdict |
|---|---|---|---|---|
| character/clientbound/attack.go:128,199 | `GMS && <79` → 1-byte action | 1-byte | else = original 2-byte short | PASS — v79+ else-branch byte-identical |
| character/clientbound/skill_prepare_foreign.go:19 | `GMS && <79` → 1-byte action | 1-byte | short | PASS |
| character/clientbound/status_message.go:299,309 (DropPickUpMeso) | `!(GMS && <79)` → partial flag | omit partial | write partial (unchanged) | PASS |
| character/clientbound/status_message.go:546,586 (IncEXP) | outer `!(GMS && <79)`, inner pre-existing `!(<83)` / `>=95` | 1 trailing int | inner block IDENTICAL to prior code | PASS — no-op for v79+ |
| field/serverbound/change.go:78,108 | chase `>=79`→`>=72` | write chase | v79+ still writes (unchanged) | PASS |
| model/attack_info.go:31,37 (byteAction/singleCrc `<79`; dragon `!<79`) | 1-byte action + single CRC, no Evan dragon | 2-byte + crc2 + dragon (unchanged) | PASS — Evan launched v84, naturally absent pre-79 |
| model/character_list_entry.go:57,83 | family byte `!(GMS && <73)` | omit (72<73) | write (79>=73) | PASS — correctly EXCLUDES v72, INCLUDES v79 |
| model/damage_info.go:53,81 | mob CRC `>=79`→`>=72` | write | v79+ unchanged | PASS |
| model/monster.go:238,326 | mob stat-mask `IsRegion(GMS)&&<79` | single 32-bit | 128-bit UINT128 (unchanged) | PASS — v79 byte-pinned (see below) |
| model/skill_prepare_info.go:31,96,116 | action `<79` | 1-byte | short | PASS |
| monster/clientbound/movement.go:59,82 | bNextAttackPossible `(GMS&&>=79)||JMS` | omit | write (unchanged) | PASS |
| monster/serverbound/movement.go:78,115 | flyCtxTargetX/Y `(GMS&&>=79)||JMS` | omit | write (unchanged) | PASS |
| npc/clientbound/conversation.go:82,97 | param+secondary `!(GMS&&!>=79)` | omit | write (unchanged) | PASS |
| npc/serverbound/start_conversation.go:52,64 | x/y `!(GMS&&!>=79)` | omit | write (unchanged) | PASS |
| summon/serverbound/attack.go:163,240,290 | skillCRC trailer `MajorAtLeast(79)` | omit | write; jms185>=79 writes (unchanged) | PASS — version-only gate; all non-v72 supported versions >=79 |

Off-by-one class checked: `>=73` gates use `!(GMS && <73)` — v72 (72<73)=omit, v79 (79>=73)=write. Correct on both sides.

## DOM-21 (no reinvented version logic / constants)
PASS. Every gate routes through `tenant.Model` predicates (`Region()`, `MajorVersion()`, `MajorAtLeast()`, `IsRegion()`). No new numeric version enums, no redeclared atlas-constants types.

## Standard DOM / SUB / SEC
N/A. All 14 source files are `libs/atlas-packet` wire codecs (no `model.go`/`resource.go`/`processor.go` domain packages, no HTTP handlers, no auth/token/redirect surface). DOM-01..20, SUB-*, EXT-*, SCAFFOLD-*, SEC-* have no applicable trigger.

## Regression proof (existing-version byte fixtures)
- Restructured codecs where v79 shares the else-path with an in-`test.Variants` version (v83/84/87/95/jms) are covered by the shared `RoundTrip` harness (`libs/atlas-packet/test/context.go:18`, includes GMS v28 to exercise the legacy branch + v83+ for the modern branch).
- monster stat-mask restructure has BOTH byte pins: `TestMonsterStatSetByteOutputV72` and `...V79` (`monster/clientbound/stat_test.go:87,134`).
- Dedicated v72 byte fixtures exist for the `>=72`/`>=73` boundary paths v28 cannot cover: `TestChangeByteOutputV72` (chase), `character/serverbound/v72_test.go` (CreateCharacter jobIndex `<73`), `character/clientbound/v72_test.go` + `v79_test.go` (char-list family byte), `TestStatusMessageIncreaseExperienceByteOutputV72`.

## Minor / informational (non-blocking, pre-existing)
- `TestStatusMessageIncreaseExperience` (`character/clientbound/status_message_test.go:217`) round-trips over `test.Variants`, which contains no `gms_v79` entry, yet carries a `packet-audit:verify version=gms_v79 ida=0x96bd0d` marker (line 216). IncEXP is the one restructured codec whose v79 wire path (`item+premium`, NO rainbow) is unique vs every `Variants` member. The marker is therefore not backed by an executed v79 byte assertion. This predates the v72 pass (`test/context.go` untouched in `902dc6fe1..HEAD`), and the v72 diff wraps the pre-existing block in an always-true-for-v79+ gate — provably a no-op for v79+. Not a v72 regression; noted only because the scope asked to confirm v79 pins. Optional fix: add a `gms_v79` byte fixture for IncEXP, or add v79 to `test.Variants`.

**Real blocking findings: 0.**

---

# v72 plan-adherence (Phase 2 — Pass v72, anchor gms_v79)

**Audit date:** 2026-07-01 · **Branch:** `task-113-gms-legacy-versions` · **HEAD:** `dc614a741`
Scope: Stages A–F + advance gates (serverbound full parity per owner decision). Runtime G/H/I are DEFERRED — not audited, not flagged. All commands run live against HEAD.

## Per-stage verdict

| Stage | Deliverable | Verdict | Evidence |
|---|---|---|---|
| A | `v72-packet-delta.md` (char-mgmt body-verified; usesPin=false) | **PASS** | file present (30767 B); usesPin=false at line 56 ("**false** for v72"); char-mgmt body mapping at lines 127/401/428/467 (cb 13/14/15 = v79 by body, rotated symbols, IDB addr 0x5b3983/0x5b3c65/0x5b3a18) |
| B | `gms_v72.yaml` (254 cb + 112 sb parity) + `discover_gms_v72.md` | **PASS** | `direction: clientbound`=254, `direction: serverbound`=112 (=366 total); every op has provenance (311 ida-discovered + 55 manual); "Missing at discovery" entries all resolved as `provenance: manual` w/ `ida.address`; 1 discovery-internal collision (op 0x04E) adjudicated |
| C | `template_gms_72_1.json` (usesPin=false, 112 handlers 0 missing validators, 18 op tables, status-msg 0–11 shrink) | **PASS** | usesPin=false; handlers=112, 0 missing validator; 18 writer operations tables; `CharacterStatusMessage` = 12 entries keyed 0–11 (shrunk from v83's 0–13); validators ∈ {LoggedIn,NoOp}; handler/writer symbols resolve in atlas-channel |
| D | `gms_v72.json` export + `audits/gms_v72/SUMMARY.md` + `_unimplemented.json` | **PASS** | export 339 KB; SUMMARY.md present (565 per-packet `.md`); `_unimplemented.json` = 6 justified IDA-confirmed version-absent entries |
| E | matrix column, `matrix --check` exit 0, no undocumented in-scope ❌ | **PASS** | `matrix --check` exit=0, forbidden-keyword count=0; `matrix` regen produced no git diff; `gms_v72` wired into model.go:14 / render.go:13 / fnamedoc.go:221; guard tests pass; residual analysis below |
| F | `code-gate-audit.md` v72 column + CharacterInteraction ops populated | **PASS** | v72 column filled across ~152 gate rows (commit dc614a741 diff shows `\| \|`→`\| … = v79 \|`); `CharacterInteractionHandle` serverbound handler ops table = 17 modes |

## Stage E residual ❌ reconciliation (the requested count)

In-scope = `tier1 OR packet startswith login/`. Gate-relevant = in-scope AND v72=❌(incomplete) AND **verified in gms_v79**.

- **Gate-relevant residual ❌ = 3**, all `sub-struct` kind, all documented-n-a — **CONFIRMS** `v72-stageE-close.md`:
  1. `cash/serverbound/CashShopOperationIncreaseCharacterSlot` → `_unimplemented.json` `CCashShop::OnIncCharacterSlotCount` (mislabel for EnableEquipSlot; no mode-9 send in v72)
  2. `interaction/serverbound/InteractionOperationMerchantAddToBlackList` → `CEntrustedShopDlg::AddBlackList` (feature post-v72)
  3. `interaction/serverbound/InteractionOperationMerchantRemoveFromBlackList` → `CEntrustedShopDlg::DeleteBlackList` (post-v72)
  These are `sub-struct` rows the matrix cannot render as n-a (op-cell-only limitation, same as v79); each is in `_unimplemented.json` with IDA-confirmed absence.

- **Residual ❌ tally:** artifact/documented-n-a = **3** · real producible gap = **0**.

- **Out-of-gate note (honest caveat):** 31 further in-scope v72 cells are ❌, but each is **also** ❌/partial in the anchor v79 (29 incomplete, 2 partial) — pre-existing anchor coverage gaps, not v72-pass regressions. The plan's literal Stage-E wording ("no in-scope cell remains ❌") is therefore met only **relative to the anchor** (the owner-adopted "verified in v79" scoping), not in absolute terms. This mirrors the v79 pass and is not a v72-introduced defect.

## Existing-version isolation

Verified counts at HEAD vs `git show 911fd93a1:docs/packets/audits/status.json` (pass-start): v83 **367=367**, v84 **345=345**, v87 **379=379**, v95 **399=399**, jms **361=361** — all identical. v79 verified = **228** at HEAD (not dropped). No conflict (🟥) cells in the v72 column.

## Overall

- **Plan Adherence (Stages A–F):** FULL. Every stage deliverable exists, every advance gate is met, and the `v72-stageE-close.md` "3 residual sub-struct = documented-n-a" claim is confirmed byte-for-byte against `status.json` + `_unimplemented.json`.
- **Recommendation:** READY (Phase 2 protocol/audit slice). Runtime G/H/I deferred per owner decision.
- **Action items:** none blocking. (Optional, for Phase 5 reconciliation: the 31 in-scope ❌ cells shared with the v79 anchor remain a cross-pass coverage backlog; they are anchor-inherited, not v72-specific.)
