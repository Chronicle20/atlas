# Packet-Audit Closeout — Four-Version Baseline — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-04
---

## 1. Overview

The Atlas packet-audit effort (tasks 027, 028, 065, 066, 067, 068, 069) audited every
wire-bound directory under `libs/atlas-packet/` against IDA-decompiled MapleStory clients
across four baselines — **GMS v83, GMS v87, GMS v95, and JMS v185**. Directory coverage is
complete: every `libs/atlas-packet/` directory maps to a contributing task or a documented
non-wire exclusion (`model/`, `test/`, `tool/`). The cross-task ledger lives at
`docs/packets/audits/gms_v95/TOTAL.md`; per-version results live in
`docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/SUMMARY.md`; the master deferral
ledger is `docs/packets/ida-exports/_pending.md` (~1844 lines, 44 sections).

While coverage is complete, a substantial body of work was **deferred** during the per-packet
loop: real wire bugs that exceeded a bucket's scope, op-byte/mode-enum verifications that were
never exhaustively enumerated, provisional version gates awaiting a specific binary, a
login-domain IDA-export backlog, a JMS cash-shop payment protocol divergence, and a class of
static-analyzer artifacts (loop-flatten, width-label, struct-name collisions) that the
`packet-audit` tool reports as ❌/🔍 even though the wire is provably correct. These deferrals
are scattered across the master ledger and the per-task `post-phase-b` closeouts (all
deferrals are reconciled — no orphan deferral exists outside `_pending.md`).

This task **closes out every actionable deferral** so the four-version audit becomes a
**finished, trusted baseline with zero open actionable deferrals**. After this task the
`_pending.md` ledger contains only explicitly-blessed permanent exclusions, the analyzer no
longer emits the known false-positive classes, every real wire bug is fixed with byte-level
tests and IDA-verified version gates, and the baseline is documented as a reusable starting
point for future client-version audit passes (e.g. a new GMS/KMS/SEA version).

## 2. Goals

Primary goals:

- **Zero open actionable deferrals.** After this task, `docs/packets/ida-exports/_pending.md`
  lists no item that requires future code or audit action. Everything is either resolved
  (✅), fixed, or recorded in a curated "accepted permanent exclusion" registry with IDA
  evidence.
- **Fix all real wire bugs** surfaced by the audit (bucket 1 + the real-bug subset of bucket 2),
  each gated correctly across the four versions and covered by byte-level wire-shape tests.
- **Resolve all op-byte / mode-enum / sub-op verification deferrals** (bucket 3) to a definite
  verdict using live IDA (all four IDBs available on demand). Any new real wire bug discovered
  during a spike is **fixed in-task**.
- **Confirm the two provisional v87 gates** (bucket 4) against the v87 binary and tighten/keep
  as the evidence dictates.
- **Clear the login-domain IDA-export backlog** (bucket 6): export the missing FNames to the
  json, audit them, and assign verdicts.
- **Close the JMS v185 cash-shop NX-payment divergence** (bucket 5) end-to-end: wire-correct
  the 5 JMS serverbound payment packets, remap the JMS template op-bytes, and route them into
  the **existing** atlas-cashshop wallet/purchase flow.
- **Enhance the `packet-audit` analyzer** to eliminate the known false-positive classes at the
  source (early-return/exclusive-branch modeling, sub-struct/loop descent, qualified
  struct-name tracking in `locateAtlasFile`, opaque-buffer/width-label equality), so a clean
  re-run shows no spurious ❌/🔍.
- **Document the baseline** so a future maintainer can start a new client-version pass from a
  known-good state, with a short "how to run a new version pass" guide and a regenerated set
  of SUMMARY/TOTAL artifacts.

Non-goals:

- Re-opening already-closed items: **storage `Show`** segmentation (resolved task-067 P2),
  **`MonsterControl`** (reclassified non-bug, `e32a3d809`), **SETFIELD/WarpToMap
  `m_dwOldDriverID` + `nHP`** gates (confirmed across v83/v87/v95).
- Auditing client versions beyond the four baselines (v92, v111, KMS, SEA, etc.). The output
  is a *baseline to build those from*, not those passes themselves.
- Building new cash-shop economy primitives — the NX wallet (Credit / Maple Points / Prepaid)
  and purchase flow already exist in `services/atlas-cashshop/atlas.com/cashshop/wallet/`.
- Cosmetic constant renames flagged as "left as-is to avoid cross-version breakage" (e.g.
  `ServerIP.SERVER_UNDER_INSPECTION`), unless a fix in this task touches the same file anyway.

## 3. User Stories

- As a **maintainer starting a new client-version audit**, I want the four existing baselines
  to have zero open deferrals and a documented "start here" procedure, so I can fork a new
  version pass from a trusted state instead of first reconciling old loose ends.
- As an **atlas-channel developer**, I want serverbound packets (NPC continue-conversation,
  hired-merchant, quest actions, group chat) parsed with the correct discriminators and field
  widths, so clients don't desync on those hot paths.
- As a **JMS v185 player**, I want cash-shop buy/gift/couple/friendship/rebate to work, so
  purchases settle against my NX wallet instead of desyncing the cash shop.
- As a **reviewer**, I want every ❌/🔍 in the audit output to be a *real* finding, so the
  audit verdict tables are trustworthy and a green run means green.
- As a **future auditor**, I want the analyzer to model early-returns, sub-structs, and
  qualified names, so I'm not re-triaging the same false-positive classes on every pass.

## 4. Functional Requirements

Work is organized into the six buckets from the deferral inventory plus an analyzer-enhancement
and a baseline-documentation workstream. Each item cites its file(s), verdict, and IDA evidence.
Every wire change MUST be accompanied by a byte-level wire-shape test and, where version
behavior differs, a version gate verified against the relevant IDB.

### 4.1 Real wire bugs (bucket 1 + real-bug subset of bucket 2)

| ID | Item | File(s) | IDA evidence | Required change |
|---|---|---|---|---|
| B1.1 | **AffectedAreaCreated / SPAWN_MIST** structural rewrite | `libs/atlas-packet/field/clientbound/affected_area_created.go` (+ atlas-maps mist event plumbing) | `CAffectedAreaPool::OnAffectedAreaCreated` v83@0x431a63, v87@0x432f3f, v95@0x437ec0, JMS185@0x436572 | Replace bespoke shape with client layout: `dwId, nType, dwOwnerId, nSkillID(int32), nSLV(byte), phase(int16), rcArea(16-byte RECT buffer), tEnd(int32)`; add leading `tStart(int32)` gated `GMS && >=95`. Drop invented `originX/originY`; plumb `nType` (mist type) + `nSkillID` + RECT coords from atlas-maps. v83==v87==JMS185 (8 fields); v95 adds `tStart`. |
| B1.2 | **chat `Multi` serverbound** missing leading `updateTime` | `libs/atlas-packet/chat/serverbound/multi.go` + group-chat-send callers | `CUIStatusBar::SendGroupMessage`@0x87f7f0 prepends `Encode4(update_time)` before chat-type byte | Add leading `updateTime uint32` gated `GMS>83`; update all callers. Hot path. |
| B1.3 | **quest `ActionStart`/`ActionComplete`** missing `nItemPos` | `libs/atlas-packet/quest/serverbound/*` + `services/atlas-channel/.../quest_action.go` | `CQuest::StartQuest`@0x6b40a0 (actions 1/2) | Insert `Encode4(nItemPos)` (delivery-item slot, 0 normal) between `npcId` and conditional `x,y`; verify atlas `autoStart` gate ↔ IDA `!CQuestMan::IsAutoAlertQuest(questId)`. Packet + handler together. |
| B1.4 | **quest `ActionRestoreLostItem`** redesign | `libs/atlas-packet/quest/serverbound/*` + `services/atlas-channel/.../quest_action.go` | `CQuest::OnCompleteQuestFailed`@0x6b1fc0 (action 0) | Redesign to count-prefixed id array: `Encode1(0)+Encode2(questId)+Encode4(count)+EncodeBuffer(4*count)`. Replace single `unk1+itemId` model with slice + count. |
| B1.5 | **EffectWeather JMS185** shape (BLOW_WEATHER) | `libs/atlas-packet/field/clientbound/effect_weather.go` | JMS185 `CField::OnPacket`@0x56e721 case 0x8B → `sub_5723E6` | Region branch: GMS keeps leading `!active`/`m_nBlowType` byte; JMS drops it, reads `Decode4 itemId` first, optional `Decode4 extra` when `get_consume_cash_item_type(itemId)==51`, optional `DecodeStr message` when `itemId!=0`. GMS/v83/v87 already correct. |

### 4.2 atlas-channel handler-logic fixes (bucket 2)

| ID | Item | File(s) | IDA evidence | Required change |
|---|---|---|---|---|
| B2.1 | **NPC continue-conversation discriminator** | `services/atlas-channel/.../socket/handler/npc_continue_conversation.go` (structs in `libs/atlas-packet/npc/serverbound/continue_conversation*.go` are already correct) | `OnSay`@0x6dc110 (msgType 0), `OnAskYesNo`@0x6dc5a0 (2/13), `OnAskText`@0x6dc790 (3), `OnAskMenu`@0x6dce00 (5), `OnAskAvatar`@0x6dcff0 (8) | Fix discriminator: text reply is msgType **3** (AskText)/**14** (AskBoxText), not 2. Map 3/14 → `ContinueConversationText`; 5/8/9 → `ContinueConversationSelection`; 0/1/2/13 → no trailing body. |
| B2.2 | **Hired-merchant serverbound handler** bare | `libs/atlas-packet/merchant/serverbound/operation.go` + `services/atlas-channel/.../socket/handler` | (verify in IDA) | Implement/verify the serverbound decode + channel handler for the hired-merchant op family (currently a bare constant). |
| B2.3 | **Merchant modes 1 / 8 / 11** disposition | `services/atlas-channel` merchant handler + `libs/atlas-packet/merchant/` | `OnEntrustedShopCheckResult` (mode 8 = `Decode4 shopId + Decode1 channelId`; mode 1 absent in v95; mode 11 present, StringPool 3508) | Implement mode 8 (ErrorUnknown channel notice); determine if mode 1 is client/KMS-only; add mode 11 constant + emitter when exercised. |

### 4.3 Op-byte / mode-enum / sub-op verification (bucket 3) — resolve via IDA to a verdict; fix new bugs in-task

| ID | Item | File(s) | What to enumerate |
|---|---|---|---|
| B3.1 | messenger serverbound `Operation` full enum | `libs/atlas-packet/messenger/serverbound/operation.go` | Confirm no modes beyond 0/2/3/5/6; verify atlas-messengers routing matches. |
| B3.2 | messenger `declineMode` sub-enum | `libs/atlas-packet/messenger/clientbound/invite_declined.go` | `OnBlocked` mode=5, `if v3` → StringPool 0x31A vs 0x31B; confirm 0/1 only vs more. |
| B3.3 | npc shop-operation clientbound mode enum | `libs/atlas-packet/npc/clientbound/shop_operation.go`, `shop_operation_body.go` | Cross-check `operations` resolver vs every `CShopDlg::OnPacket`@0x6eb7d0 case (`nType==365`); confirm modes 4/6/7/0xB/0xC carry no emitter. |
| B3.4 | npc shop serverbound op-byte values (esp. LEAVE) | `libs/atlas-packet/npc/serverbound/{shop,shop_buy,shop_sell,shop_recharge}.go` + `services/atlas-channel/.../npc_shop.go` | Confirm channel `operations` config BUY=0/SELL=1/RECHARGE=2 against IDA (`SendBuyRequest`@0x6e9bb0 etc.); locate/confirm LEAVE op value and that no body trails it. |
| B3.5 | 7 interaction serverbound sub-ops, no located IDA sender (spikes) | `libs/atlas-packet/interaction/serverbound/operation_{create,open,cash_trade_open,invite_decline,visit,merchant_name_change,personal_store_set_visitor}.go` | Focused IDA spike per sub-op; assign verdict; fix any real divergence in-task. |
| B3.6 | social-domain sub-op enum-drift cross-version pass | buddy/chat/guild/party/note dispatchers + templates; incl. `BuddyError` conditional-string arms (modes 0x10/0x11/0x13/0x16) | Verify template-configured sub-op VALUE spaces (mode/op numbers) match client across v83/v87/v95/JMS185; per-struct wire shapes already ✅. |

### 4.4 v87 provisional-gate confirmation (bucket 4)

| ID | Item | File(s) | What to confirm |
|---|---|---|---|
| B4.1 | stat `Changed` HP/MP width gate; ui `Lock` int32 gate | `libs/atlas-packet/stat/clientbound/changed.go`, `libs/atlas-packet/ui/clientbound/lock.go` | Confirm both gates against the v87 IDB (currently confirmed v83 + JMS185 only); tighten/keep boundary as evidence dictates. |

### 4.5 JMS v185 cash-shop NX-payment protocol (bucket 5) — full closeout

| ID | Item | File(s) | IDA evidence | Required change |
|---|---|---|---|---|
| B5.1 | JMS cash-shop buy/gift/couple/friendship/rebate | `libs/atlas-packet/cash/serverbound/shop_operation_{buy,gift,buy_couple,buy_friendship,rebate_locker_item}.go`; `template_jms_185_1.json`; `services/atlas-channel` cash routing → `services/atlas-cashshop` wallet/purchase flow | `CCashShop::OnBuy`@0x47eaa7 (op 3), `SendGiftsPacket`@0x47bced (0x2E), `OnBuyCouple`@0x48085a (0x1E), `OnBuyFriendship`@0x481184 (0x24), `OnRebateLockerItem`@0x47c059 (0x1B) | Add JMS-correct serverbound bodies (SPW string + serial-number shapes) using a **region-dispatched body strategy** (NOT a 3rd nested guard — respect the 2-nested-guard hard cap). Remap JMS op-bytes in `template_jms_185_1.json` citing each `Encode1(...)`. Route into the existing atlas-cashshop wallet (Credit/Maple Points/Prepaid) purchase flow. Also remap (template-only, bodies already match): JMS PersonalStore `BuyItem` op 0x14/0x1F (GMS 0x17/0x22), `DeliverBlackList` op 0x1B (GMS 0x1E). |

### 4.6 Login-domain IDA-export backlog (bucket 6)

Export the missing FNames to the version json(s) via the `packet-audit export` (live IDA-MCP)
path, then audit and assign verdicts. Atlas writers/handlers under `libs/atlas-packet/login/`.

- **Addressed FNames:** `CLogin::OnViewAllCharResult`@0x5de120 (→ `AllCharacterListPong`,
  CharacterListEntry sub-struct), `CLogin::SendSelectCharPacketByVAC`@0x5d7550
  (→ `CharacterSelectWithPic`/`*Register?`), `CLogin::OnSelectCharacterByVACResult`@0x5de670
  (→ `PicResult?`), `CLogin::OnDenyLicense`@0x5d45d0, `CLicenseDlg::OnButtonClicked`@0x5ff870
  (UI callback), `LoginAuth` (orphan — confirm legacy/JMS-only or remove).
- **Bare login handlers without IDA mapping:** `AfterLoginHandle` (0x09),
  `RegisterPinHandle` (0x0A), PIC family (0x15–0x1E: `CheckPicHandle`, `RegisterPicHandle`,
  `CharacterSelectedPicHandle`, `CharacterListSelectHandle`, `CharacterListSelectWithPicHandle`),
  `SetGenderHandle` (0x08), `WorldCharacterListRequest` (0x05), `ServerStatus` (clientbound),
  `PicResult` (clientbound).
- **v87 login quirks:** `CLogin::SendCheckPasswordPacket`@0x62dfb4 v87 appends
  `Encode4(PartnerCode)` (zero functional impact — decide read-or-document);
  `CLogin::SendSelectCharPacket` 0x1D/0x1E v87 PIC opcode layout differs from the v87 template
  handler-opcode mapping (needs v87-specific handler variants or opcode-keyed dispatch).

### 4.7 Analyzer enhancements (decision: eliminate false positives at source)

Enhance `tools/packet-audit/` so the following known false-positive classes no longer produce
spurious ❌/🔍 (verify by re-running the audit and diffing against the curated registry):

- **Early-return / exclusive-branch modeling** — flag `return` inside guarded blocks so
  conditional bytes aren't over-counted (login `CharacterList`, character `CharacterSitResult`,
  monster/drop/reactor `Spawn`/`ReactorHitRequest`, cash `IncreaseInventory/Storage`).
- **Sub-struct / loop descent** — descend into `model.Asset`/`GW_ItemSlotBase`, per-tab loops,
  Visitor/Room sub-structs, party `WritePartyData`, guild BBS, npc `Action`/`ShopList`, pet
  bodies (inventory `Add`/`ChangeBatch`, storage `UpdateAssets`, character
  `CharacterInfo`/`CharacterSkillChange`/`AddCharacterEntry`/`CharacterViewAllCharacters`).
- **Opaque-buffer / width-label equality** — treat `WriteByteArray(N)` ≡ `DecodeBuf(N)`,
  `WriteLong` ≡ `EncodeBuffer(8)`, `WriteInt16+WriteShort(0)` ≡ `Decode4`,
  `WriteInt64 point` ≡ `EncodeBuffer(&pt,8)` (messenger AvatarLook, note `Display`, guild BBS
  FILETIME, fame `GiveResponse`, socket `Hello`/`ChannelConnect`, stat `Changed`, omok
  `MoveStone`).
- **Qualified struct-name tracking in `locateAtlasFile`** — eliminate same-name collisions
  (`ChannelChange` buddy-vs-channel; monster/drop/reactor/pet `Spawn`/`Destroy`/`Movement`).

Any artifact that remains genuinely outside analyzer reach after enhancement is moved to the
accepted-exception registry (§4.8) with a one-line justification — but the bar is to fix in the
tool first.

### 4.8 Baseline documentation + ledger curation

- Regenerate per-version `SUMMARY.md` after fixes + analyzer enhancement; the four files should
  show no spurious ❌/🔍.
- Reduce `docs/packets/ida-exports/_pending.md` and `docs/packets/audits/gms_v95/_pending.md`
  to zero actionable items. Replace the deferral content with a curated **accepted permanent
  exclusions** registry (the genuinely-unanalyzable residue, each with IDA evidence) plus a
  pointer to this task as the closeout of record.
- Update `docs/packets/audits/gms_v95/TOTAL.md`: flip task statuses to shipped, recompute the
  verdict roll-up, and replace §3/§5 with a "**baseline complete — zero open actionable
  deferrals**" statement.
- Add a short **"starting a new client-version pass from this baseline"** guide (where IDBs go,
  how to run `packet-audit export`/audit, how SUMMARY/TOTAL/_pending relate, the gate-naming
  convention `Region()=="GMS" && MajorVersion()>=N`, and the region-dispatched body strategy
  for >2-version divergences).

## 5. API Surface

No new REST/JSON:API endpoints. Wire-protocol (packet) changes only, plus internal
atlas-channel → atlas-cashshop messaging for the JMS cash-shop routing (reuse existing
cash purchase Kafka/command surface — do not invent a new external API).

`template_jms_185_1.json` op-byte remaps (cash serverbound + two interaction ops) are
configuration changes, each justified by a cited IDA `Encode1(...)`.

## 6. Data Model

No new persisted entities. The AffectedAreaCreated rewrite (B1.1) requires the atlas-maps mist
event / model to carry **`nType` (mist type)**, **`nSkillID`**, and the **RECT coordinates**
(LT/RB) so the packet can emit the client-correct shape; this is in-memory event/model
plumbing, not a schema migration. All multi-tenant context flows via the existing
`tenant.MustFromContext(ctx)` path unchanged.

## 7. Service Impact

- **`libs/atlas-packet/`** — primary surface: field, chat, quest, cash (serverbound),
  messenger, npc, interaction, stat, ui, login, merchant directories. New/changed encoders +
  byte-level tests + version gates.
- **`services/atlas-channel`** — handler-logic fixes (NPC continue-conversation, hired-merchant,
  quest actions, group chat callers, npc shop op config) and JMS cash-shop serverbound routing.
- **`services/atlas-cashshop`** — JMS cash-shop purchases route into the existing wallet
  (Credit / Maple Points / Prepaid) + purchase flow under `.../cashshop/wallet/`.
- **`services/atlas-messengers`** — confirm messenger op/declineMode enums against server-side
  emissions (verification; code change only if a divergence is found).
- **`tools/packet-audit/`** — analyzer enhancements (§4.7).
- **`docs/packets/`** — SUMMARY/TOTAL/_pending regeneration + baseline guide + accepted-exception
  registry; `template_jms_185_1.json` op-byte remaps.
- **atlas-maps** (mist event source) — plumb `nType`/`nSkillID`/RECT for B1.1.

## 8. Non-Functional Requirements

- **Correctness over coverage.** Every wire change is proven by a byte-level wire-shape test
  asserting the exact emitted bytes per targeted version; no change ships on analyzer verdict
  alone (the analyzer is an aid, the byte test + IDA is the oracle).
- **Version-gate discipline.** Gates use `Region()=="GMS" && MajorVersion()>=N`; divergences
  spanning >2 versions use a region-dispatched body strategy, never a 3rd nested `if`
  (respect the documented 2-nested-guard hard cap; the repo nesting `awk` must stay clean).
- **Multi-tenancy / observability** unchanged — existing context propagation and logging
  patterns apply; no packet path may drop tenant context.
- **Build/verify gates (per CLAUDE.md):** `go test -race ./...`, `go vet ./...`,
  `go build ./...` clean in every changed module; `docker buildx bake atlas-<svc>` for every
  service whose `go.mod` is touched; `tools/redis-key-guard.sh` clean.
- **IDA evidence captured.** Each resolved verification item and each new fix records the
  IDA function@address and the read-order it was verified against, in the report/registry.
- **No regressions to closed items** — storage Show, MonsterControl, SETFIELD/WarpToMap gates
  must remain untouched and green.

## 9. Open Questions

- **AffectedArea atlas-maps plumbing (B1.1):** does the atlas-maps mist/affected-area event
  already carry skill id + skill (mist) type + RECT, or must those be added to the event
  contract? (Resolve during design by reading the atlas-maps mist emission path.)
- **`LoginAuth` orphan:** confirm whether it is dead/legacy (remove) or JMS-only (gate). To be
  settled by the bucket-6 IDA spike.
- **Merchant mode 1:** confirm client/KMS-only vs a real v95 gap before deciding to implement
  vs document.
- **Analyzer enhancement depth:** how much of the sub-struct descent is worth generalizing vs
  special-casing — design should weigh tool-generality against closeout cost (the goal is a
  clean re-run, not a perfect decompiler).

## 10. Acceptance Criteria

- [ ] **B1.1–B1.5** real wire bugs fixed, each with a byte-level test per affected version and
      IDA-verified gates; AffectedArea emits the client-correct RECT-buffer shape with `tStart`
      gated `GMS>=95`; EffectWeather JMS branch correct.
- [ ] **B2.1–B2.3** atlas-channel handler fixes landed; NPC continue-conversation routes
      text(3/14)/selection(5/8/9)/none(0/1/2/13) correctly; hired-merchant serverbound handled;
      merchant mode 8 implemented (1/11 dispositioned).
- [ ] **B3.1–B3.6** every verification deferral resolved to a definite verdict against IDA;
      any real bug found is fixed in-task with tests; the social cross-version enum-drift pass
      confirms template op/mode values across all four versions.
- [ ] **B4.1** v87 stat-Changed + ui-Lock gates confirmed against the v87 IDB.
- [ ] **B5.1** JMS cash-shop buy/gift/couple/friendship/rebate wire-correct with tests, template
      op-bytes remapped (with IDA citations), and routed into the existing atlas-cashshop wallet
      purchase flow; the two interaction template op-bytes remapped; no 3rd nested guard
      introduced (nesting `awk` clean).
- [ ] **B6** login IDA-export backlog exported to json, audited, verdicts assigned; bare login
      handlers mapped or documented; v87 login quirks resolved/documented.
- [ ] **§4.7** analyzer enhanced; a fresh `packet-audit` run across all four versions emits no
      spurious ❌/🔍 from the named false-positive classes; remaining residue (if any) is in the
      accepted-exception registry with justification.
- [ ] **§4.8** all four `SUMMARY.md` regenerated; `_pending.md` (both copies) reduced to zero
      actionable items; `TOTAL.md` states "baseline complete — zero open actionable deferrals";
      a "new version pass from baseline" guide exists.
- [ ] Closed items (storage Show, MonsterControl, SETFIELD/WarpToMap) untouched and green.
- [ ] All CLAUDE.md build/verify gates pass on every changed module/service (test -race, vet,
      build, `docker buildx bake` per touched go.mod, redis-key-guard).
- [ ] Code review run (plan-adherence + backend-guidelines reviewers) before PR.
