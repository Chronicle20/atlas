# Backend Audit — task-100 (WHISPER + SPOUSE_CHAT serverbound ops)

- **Scope:** Go changes on branch, diff 47452b3..46eaf6e
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-16
- **Build:** PASS (libs/atlas-packet, atlas-channel, tools/packet-audit)
- **Vet:** PASS (changed packages)
- **Tests:** PASS (chat/serverbound, field/serverbound, socket/handler)
- **Overall:** PASS

## Nature of the change

This is a packet-codec + channel socket/message-layer change, not a DDD domain
package. No package under the diff has `model.go` / `entity.go` / `resource.go`,
so the DOM-01..DOM-20 mechanical checks (builder/ToEntity/Transform/JSON:API/
administrator/provider) are not applicable. The relevant guidelines are the
functional/immutable patterns, the processor Interface+Impl pattern, the Kafka
producer pattern, and DOM-21 (no reinvented atlas-constants).

## Findings

### Critical
- None.

### Important
- None.

### Minor
- **No handler-level test for the new spouse-chat handler.**
  `services/atlas-channel/atlas.com/channel/socket/handler/character_spouse_chat.go:14`
  has no `*_test.go`. This is consistent with the sibling it mirrors
  (`character_chat_whisper.go` also has no handler test), so it is not a
  regression, but the emit path (`Processor.SpouseChat` →
  `SpouseChatCommandProvider`) is exercised only indirectly. The packet codec
  itself (`spouse_chat.go`) has full golden + round-trip coverage. Low severity.

## Check results (applicable items)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Build gate | go build all changed modules | PASS | libs/atlas-packet, atlas-channel, tools/packet-audit all build clean |
| Vet gate | go vet changed packages | PASS | chat/serverbound, field/serverbound clean |
| Test gate | go test changed packages | PASS | chat/serverbound 0.004s, field/serverbound 0.008s, socket/handler 0.007s |
| Immutability | private fields + getters | PASS | `field/serverbound/spouse_chat.go:33-35` CoupleMessage has private `spouseName`/`message`, getters `SpouseName()`/`Message()` at :30,:32 |
| Processor Interface+Impl | SpouseChat on both | PASS | `message/processor.go:21` interface method; `message/processor.go:84` ProcessorImpl method |
| Kafka producer pattern | curried ProviderImpl + SingleMessageProvider | PASS | `message/producer.go:57` SpouseChatCommandProvider returns `model.Provider[[]kafka.Message]` via `producer.SingleMessageProvider`; emit via `producer.ProviderImpl(p.l)(p.ctx)(...)` at processor.go:85 — byte-identical to WhisperChatCommandProvider |
| Topic reuse (DOM-23) | no new topic constant | PASS | reuses `message2.EnvCommandTopicChat` (processor.go:85); no new `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` added |
| DOM-21 reinvented constants | no shadow of atlas-constants | PASS | new types are `SpouseChatBody{SpouseName string}` (kafka.go:69) and `ChatTypeSpouse = "SPOUSE"` (kafka.go:20) — chat-DTO/enum, not item/inventory/world/job id reinventions |
| Version gate correctness | whisper updateTime gate matches sibling | PASS | `chat/serverbound/whisper.go:30` `whisperHasUpdateTime` = `(GMS && Major>=87) \|\| JMS`, identical to `interaction/serverbound/operation_chat.go:33` `chatHasUpdateTime`; replaces prior `>=95` bug |
| Error handling | handler logs, no swallow | PASS | `character_spouse_chat.go:20-22` checks err and logs `WithError`; processor returns the producer error |
| No silent stubs | no TODO/501/panic in new code | PASS | grep of all new files: no stub markers |
| packet-audit tool | candidatesFromFName cases added | PASS | `tools/packet-audit/cmd/run.go` adds `CField::SendLocationWhisper` + `CUIStatusBar::SendCoupleMessage` cases; tool builds |
| Test style | table-driven round-trip + golden fixtures | PASS | `spouse_chat_test.go:129` round-trip over `pt.Variants`; per-version golden tests with `packet-audit:verify` markers |

---

# Plan-Adherence Verification — task-100 (independent audit)

- **Auditor:** plan-adherence reviewer (read-only)
- **Date:** 2026-06-16
- **Diff range:** 47452b3..46eaf6e (2 commits)
- **Verdict:** PASS — all stated requirements genuinely implemented with byte-level / command evidence.

## Verification gates (exit codes confirmed)

| Gate | Result |
|------|--------|
| `go run ./tools/packet-audit matrix --check` | EXIT 0 |
| `go run ./tools/packet-audit fname-doc --check` | EXIT 0 (213 structs w/o report carry no fname) |
| `go run ./tools/packet-audit operations --check` | EXIT 0 (2 pre-existing jms absent-writer notes, unrelated) |
| `libs/atlas-packet` build / vet / test | 0 / 0 / 0 (all pkgs ok) |
| `tools/packet-audit` build / vet / test | 0 / 0 / 0 |
| `services/atlas-channel` build / vet / test | 0 / 0 / 0 (0 FAIL) |

## Requirement-by-requirement

### R1a — WHISPER serverbound → verified: PASS
- Gate fix: `whisper.go:28-30` `whisperHasUpdateTime` = `(GMS && Major>=87) || JMS`; replaces prior `>=95` bug at both Encode (`whisper.go:77`) and Decode (`whisper.go:92`).
- Fixtures prove the gate: v83/v84 goldens have NO updateTime; v87/v95/jms insert 4-byte `0x64 00 00 00` at index 1 (`whisper_test.go`).
- Routed in all 5 templates (`CharacterChatWhisperHandle`): gms_83/84 pre-existing, gms_87 0x7E, gms_95 0x8D, jms 0x7A added — opcodes match registry serverbound rows.
- candidatesFromFName linkage: `run.go:1248` adds `case "CField::SendLocationWhisper"` (the registry PRIMARY fname) → `chat/serverbound/Whisper`.
- Per-packet matrix cell `chat/serverbound/ChatWhisper` = verified across v83/v84/v87/v95/jms (status.json).

### R1b — SPOUSE_CHAT serverbound NEW codec → verified: PASS
- New codec `field/serverbound/spouse_chat.go` `CoupleMessage`: `EncodeStr(spouseName)+EncodeStr(message)`, no mode byte, no updateTime — matches fixtures (two strings only).
- Channel handler `character_spouse_chat.go` decodes + calls `message.Processor.SpouseChat`; Kafka command added (`ChatTypeSpouse`, `SpouseChatBody`, `SpouseChatCommandProvider`, processor method); wired in `main.go:855` via `fieldsb` alias.
- Routed in 4 GMS templates (gms_83 0x79, gms_84 0x7B, gms_87 0x7F, gms_95 0x8E) — match registry; jms correctly NOT routed (version-absent).
- candidatesFromFName: `run.go:1257` `case "CUIStatusBar::SendCoupleMessage"` → `field/serverbound/CoupleMessage`.
- Per-packet cell `field/serverbound/FieldCoupleMessage` = verified v83/v84/v87/v95, n-a jms (status.json).

### R2 — v84 serverbound reshift completion: PASS
- Registry gms_v84.yaml IDA-verified +2 values, all confirmed: MULTI_CHAT 121/0x79, SPOUSE_CHAT 123/0x7B, PLAYER_INTERACTION 125/0x7D, DENY_PARTY_REQUEST 127/0x7F — each `provenance: ida-discovered` with `ida.address` + send-site note.
- No duplicate serverbound opcodes in gms_v84 (python scan). (Pre-existing login/AP dups in v87/v95/jms are unrelated and untouched.)

### R3 — Registry hygiene: PASS
- `CField::OnWhisper` removed from WHISPER **serverbound** fname_alts in all 5 versions (now only `CField::SendChatMsgWhisper`). The remaining `OnWhisper` occurrences are the legitimate **clientbound** WHISPER `fname` rows (correct — OnWhisper is the receive decode).

## Expected-❌ confirmation (not a gap)
- WHISPER serverbound OP-ROW = `incomplete` ("no audit report") across all versions — consistent with the mode-prefix flat-diff limitation. The per-packet `chat/serverbound/ChatWhisper` row carries the ✅. This matches the stated expectation; not a missed requirement.

## Artifacts present
- Byte fixtures carry `packet-audit:verify` markers: ChatWhisper ×5 (incl jms), FieldCoupleMessage ×4 (no jms).
- Evidence YAMLs (`docs/packets/evidence/<ver>/...`) with IDA fn/address/decompile_sha256 + `verifies:` test-fn refs.
- Audit reports (`docs/packets/audits/<ver>/{ChatWhisper,FieldCoupleMessage}.{md,json}`) with ✅ verdicts.

## No silent gaps found
Every cell claimed ✅ is backed by a marker fixture + evidence + audit report; no stubs/TODOs in new code.

---

# Backend Audit — per-mode dispatcher body codecs (task-096 graduation)

- **Scope:** new/changed Go in `libs/atlas-packet/{messenger,cash,interaction,storage,npc,field}/clientbound/` (MESSENGER, CASHSHOP_OPERATION, PLAYER_INTERACTION, STORAGE, CONFIRM_SHOP_TRANSACTION, MTS_OPERATION dispatcher families) + `tools/packet-audit/cmd/run.go`.
- **Diff range:** BASE=7d3990a10 HEAD=d22f32686 (~12k lines / 385 files; the bulk is docs evidence YAMLs + ida-exports, out of Go scope — ~25 clientbound `.go` files + run.go in scope).
- **Date:** 2026-06-16
- **Reviewer:** backend-guidelines-reviewer (packet-codec mode; DOM-01..20 builder/entity/JSON:API N/A — these are immutable packet structs, not DDD domains)
- **Build:** PASS (`go build ./...` in libs/atlas-packet)
- **Vet:** PASS (`go vet ./...`)
- **Tests:** PASS (`go test` messenger/cash/interaction/storage/npc/field clientbound — all ok)

## Note on conventions applied
These files follow the packet-codec contract (canonical reference `field/clientbound/effect.go`): immutable struct (private fields + value-receiver getters, no mutators), `New…` constructor returning a value, `Operation() string`, `String() string`, `Encode(l,ctx)` on value receiver, `Decode(l,ctx)` on pointer receiver. Tests are encode-only golden-byte fixtures pinned to IDA via `packet-audit:verify` markers — that is the established campaign convention, so absence of round-trip Decode in tests is NOT a defect by itself.

## Critical
- None. No silent stubs, no `// TODO`/`FIXME`/`panic`/501 in landed code (grep-clean across scope). Build/vet/test green. `run.go` candidatesFromFName additions are internally consistent and well-cited.

## Important
- **[gofmt] Two newly-introduced files are not gofmt-clean.** `libs/atlas-packet/field/clientbound/mts_operation_body.go` and `libs/atlas-packet/field/clientbound/mts_operation_list.go` (both NEW in this diff) fail `gofmt -l`. The violations are manual whitespace alignment of consecutive getter one-liners and struct-field trailing comments that gofmt re-flows (e.g. `mts_operation_body.go:89-91`, `:152`, `:304-318`). Non-functional, but a repo convention violation that should have been caught pre-merge. Fix: `gofmt -w` both files.
  - For context, 12 other changed files also fail `gofmt -l`, but those (`messenger/clientbound/{add,chat,invite_declined,invite_sent,request_invite}.go`, `interaction/clientbound/interaction.go`, `cash/clientbound/{shop_inventory,shop_item_moved,shop_operation_result}.go`, `storage/clientbound/{show,update_assets}.go`, `interaction_body.go`) were already gofmt-dirty at BASE — pre-existing debt, not introduced here. Worth a `gofmt -w` sweep while touching them anyway.

## Minor
- **[consistency] `InteractionUpdateMerchant` has `Encode` but no `Decode`.** `libs/atlas-packet/interaction/clientbound/interaction.go:263` (Encode at :278). It is the only struct in that file lacking a Decode — all 7 siblings (`InteractionInvite` :44, `InteractionInviteResult` :80, `InteractionEnter` :112, `InteractionEnterResultSuccess` :143, `InteractionChat` :179, `InteractionEnterResultError` :214, `InteractionLeave` :249) pair Encode with a pointer-receiver Decode, and every new MTS body struct in this same campaign defines Decode despite encode-only fixtures. The struct predates this diff but the diff rewrote its comment/fname (`CPersonalShopDlg`→`CEntrustedShopDlg`) and added an encode-only fixture, so it was in-hand. The body (mode + meso + count + per-item perBundle/quantity/price/`model.Asset`) is internally correct and the loop matches `interaction.RoomShopItem` widths; the gap is purely the missing mirror. Add the matching `Decode` for symmetry.
- **[consistency] Two MTS constructors hardcode `mode` instead of taking it as a parameter.** `NewMtsResultRegisterSaleEntryFailed` (`mts_operation_body.go:253`, hardcodes `mode: 0x1E`) and `NewMtsResultSuccessBidInfo` (`:310`, hardcodes `mode: 0x3E`); `NewMtsResultGet*`/`NewMtsResultLoadWishSaleListDone` in `mts_operation_list.go` likewise hardcode their mode bytes. Every other body constructor in scope (`NewMtsResultEmpty`, `NewMtsResultReason`, `NewMtsResultTwoInts`, and all the messenger/cash/storage/effect siblings) accepts `mode byte`. Defensible since each list/conditional arm has exactly one fixed mode, but it is an inconsistent constructor signature across the family. Either standardize on accepting `mode` everywhere or document why the single-arm structs pin it.

## Confirmed clean (adversarially verified, no defect)
- **DOM-21 (no reinvented shared types):** PASS. Item blobs reuse `libs/atlas-packet/model.Asset` (`mts_operation_list.go:7,49,116,140`; `storage/show.go`, `storage/update_assets.go`, `cash/shop_item_moved.go:18`, `interaction.go:288`); avatars reuse `model.Avatar` (`messenger/add.go`, `messenger/update.go`); `storage/show.go` reuses `inventory.Type`. No item/inventory/world/character id type redeclared. The cash GW_CashItemInfo 55-byte blob uses the cash lib's existing `CashInventoryItem` record (shared across CashShopInventory/PurchaseSuccess), not a per-packet reinvention.
- **Immutability:** PASS. No mutators/setters on any struct in scope.
- **Encode/Decode mirror symmetry (where Decode exists):** PASS. Verified conditional-tail symmetry on the trickiest arms: `MtsResultRegisterSaleEntryFailed` Decode2 tail gated `reason==0x48` on both sides (`mts_operation_body.go:270/281`); `MtsResultSuccessBidInfo` `itemId>0` tail (price + 8-byte FILETIME) gated identically (`:330/343`); `cash/shop_inventory.go` v95/jms extra-counts tail gated by the same predicate both sides; `storage/show.go` currency-flag + per-tab loops symmetric. `[8]byte` FILETIME buffers written via `WriteByteArray(x[:])` and read via `copy(x[:], r.ReadBytes(8))` — fixed 8, no length prefix, matched.
- **Receiver/return conventions:** PASS. All constructors return values; all Encode value-receiver, all Decode pointer-receiver.
- **`tools/packet-audit/cmd/run.go`:** PASS. New `#`-suffixed `candidatesFromFName` cases map cleanly to the new struct names, each carries per-version IDA address citations, no duplicate/conflicting fname cases.

## Overall
**NEEDS-WORK (cosmetic only).** No correctness, immutability, DOM-21, or stub defects. Blocking-adjacent item is the gofmt failure on the two new files (`gofmt -w` fixes it). The two Minor consistency items (missing `InteractionUpdateMerchant.Decode`, hardcoded-mode constructors) are non-blocking polish.

---

# Adversarial Honesty Audit — Dispatcher per-mode-body campaign (BASE 7d3990a10 .. HEAD d22f32686)

**Date:** 2026-06-16  **Auditor mandate:** prove the 6 dispatcher families' ✅ are HONEST (no false pass on mode-byte-only codecs). Read-only; no code changes.

## Verdict per family

| Family | Dispatcher fname | Verdict | Strongest evidence |
|---|---|---|---|
| MESSENGER | CUIMessenger::OnPacket | **PASS** | All 8 arms write mode+full body; each has `#fname` marker + verify fixture asserting per-field values. |
| CASHSHOP_OPERATION | CCashShop::OnCashItemResult | **PASS** | 8 graded arms each mode+full body, `#fname` + verify fixtures. Gift arm correctly UN-marked (not aggregated). |
| PLAYER_INTERACTION | CMiniRoomBaseDlg::OnPacketBase | **PASS** | 6 arms full bodies; UPDATE_MERCHANT genuinely encodes meso+item-loop (interaction.go:278-292) with golden-byte fixture. |
| STORAGE | CTrunkDlg::OnPacket | **PASS** | Show/UpdateAssets/UpdateMeso/ErrorMessage write real bodies w/ golden-byte fixtures; ErrorSimple mode-only is IDA-justified. |
| CONFIRM_SHOP_TRANSACTION | CShopDlg::OnPacket | **PASS (weak fixture)** | LevelRequirement=mode+WriteInt(level), GenericError=mode+bool+optional string (shop_operation.go:100-114,62-72) — real bodies. BUT fixtures are round-trip-symmetry-only, no golden-byte assertion (weakest in campaign). |
| MTS_OPERATION | CITC::OnNormalItemResult | **PASS (IDA-verified)** | 35 modes → 11 body shapes, all covered; dispatcher + arms IDA-confirmed live (v95). |

## Item 1 — families.yaml de-capped: PASS
`docs/packets/evidence/families.yaml` parses `dispatchers: None` (Python yaml load: key present, value empty — all 6 fnames are now COMMENTS, no list items). `matrix.LoadFamilies().Set()` therefore returns an empty map → no op capped at 🧩. No family shows ✅ while still capped. Verified `families.go:39-46` keys off the parsed `Dispatchers []string` slice only.

## Item 2 — every supported arm maps to a FULL-body codec: PASS (one weak-fixture note)
The matrix mechanically enforces "all arms covered": `build.go:32-41` joins all reports sharing a base FName (strips `#suffix` via `baseFName`, build.go:22-26); `worstCandidateCell` (build.go:261-285) grades the op-row as the WORST of every `#`-arm report. So an op-row is ✅ only if EVERY `#`-arm sub-report is ✅, and a sub-report reaches ✅ only with `marker.Found && hasEvidence && evidence.Fresh` (grade.go:197-203). All 6 op-rows are ✅ in STATUS.md (MTS/MTS2 jms ⬜ = version-absent, correct).
- Mode-byte-only codecs found are all the legitimate "Empty/notice" arms (MtsResultEmpty, StorageErrorSimple, ShopOperationSimple) — each carries an IDA sub-handler address + "no Decode* after dispatcher Decode1" justification. **Spot-checked OnSetZzimDone @0x576140 (v95) live in IDA: body is StringPool::GetString + CUtilDlg::Notice + m_bITCRequestSent=0 store, ZERO CInPacket::Decode* — claim is honest.**

## Item 3 — MTS 35 modes all covered, list arms real ITCITEM: PASS (IDA-verified)
- Cross-checked all 35 yaml modes (mts_operation.yaml, decimal 21-62) against the 11 shape groups (Empty 19, Reason 7, TwoInts 2, RegisterSaleEntryFailed 1, SuccessBidInfo 1, + 5 list arms). **Set equality holds exactly: no mode uncovered, no extra mode invented.**
- Live IDA v95 decompile of `CITC::OnNormalItemResult @0x5771d0`: confirmed it is `switch(CInPacket::Decode1())` with case labels 0x15-0x18,0x1D-0x38,0x3C-0x3E — exactly the 35 modes. Sub-handler addresses match the codec comments 1:1 (OnGetITCListDone@0x576500, OnSetZzimDone@0x576140, OnSuccessBidInfoResult@0x577000, OnNotifyCancelWishResult@0x576f00).
- List arms (mts_operation_list.go) use `MtsItem` = `model.Asset` (GW_ItemSlotBase) + a 16-field ITCITEM trailer cited 1:1 to ITCITEM::Decode (v95 0x575710). Real body, not stubbed.
- **Live IDA verify of the conditional SuccessBidInfo body @0x577000:** Decode1(soldFlag), Decode4(itemId), `if(itemId>0){ Decode4(price); DecodeBuffer(8) }`, else body ends. EXACTLY matches MtsResultSuccessBidInfo.Encode (mts_operation_body.go:324-336). Conditional-body claim honest.

## Item 4 — byte-fixtures real, not tautological: MOSTLY PASS
- MTS, STORAGE, INTERACTION fixtures include explicit golden-byte assertions (byte offsets + sentinel values, e.g. mts_operation_list_test.go:91-124). Not tautological.
- MESSENGER, CASHSHOP fixtures use round-trip + per-FIELD getter assertions (distinct values per field) — catches field-order/width bugs, acceptable.
- **CONFIRM_SHOP_TRANSACTION fixtures (shop_operation_test.go:24-66) are the lone weakness: pure `RoundTrip(...nil)` encode/decode SYMMETRY with NO field or byte assertion.** A symmetric Encode/Decode error would pass. Marker scan (matrix.go:217-232) only validates marker↔evidence address linkage, not test content, so the grader cannot detect this. The codec body is provably real and matches the documented decompile; the fixture just doesn't independently prove byte-exactness. Below the `IMPLEMENTING_A_PACKET.md` golden-byte bar.

## Item 5 — gates (run live): ALL PASS
- `packet-audit matrix --check` → exit 0
- `packet-audit fname-doc --check` → exit 0 ("212 structs without an audit report carry no fname")
- `packet-audit operations --check` → exit 0 (2 absent-writer notes, unrelated jms CharacterStatusMessage/NoteOperation)
- libs/atlas-packet: `go build ./...` exit 0, `go vet ./...` exit 0, `go test -race ./...` exit 0
- tools/packet-audit: `go test ./...` exit 0
- STATUS.md: all 6 op-rows ✅ (MTS/MTS2 jms ⬜ version-absent).

## Item 6 — TODO/stub/faked-body in landed code: ONE PRE-EXISTING (not a campaign false-pass)
- `cash/clientbound/shop_operation_body.go:80` — `// TODO map codes for JMS — currently hardcoded to 0x4D` in `CashShopCashGiftsBody`, calling `NewCashShopGifts(0x4D)` (a `// stub`, shop_inventory.go:171 writing mode+short(0)). **PRE-EXISTING** (git -S: introduced 6b6a74c9d 2026-03-18, before campaign BASE). The gift arm has NO `#fname` marker → NOT aggregated into the CASHSHOP op-row, and its only caller in atlas-channel is COMMENTED OUT. Does not inflate any ✅. Noted as residual tech-debt, not a campaign honesty defect.
- No `// TODO`/stub introduced by this campaign in any graded arm.

## Arms I could NOT independently confirm as honestly byte-exact
- **CONFIRM_SHOP_TRANSACTION: Simple, LevelRequirement, GenericError on-wire byte layout** — codec source is correct and matches the pinned decompile, but the fixture proves only encode/decode symmetry, not byte-exactness. Recommend (non-blocking) adding v83 golden-byte assertions (mode|level int; mode|bool|string) citing the already-pinned decompile offsets. This does not retract the ✅ (mechanically legitimate) but it is the one place the "FULL per-mode body" claim is attested rather than fixture-proven.

## Bottom line
All 6 families graduate HONESTLY. families.yaml is de-capped; the worst-of-arms aggregation means each ✅ requires every `#`-arm to carry marker+fresh-evidence; the 35 MTS modes are fully and exactly covered and IDA-confirmed live (dispatcher, an Empty arm, and the conditional SuccessBidInfo body). The single material weakness is CONFIRM_SHOP_TRANSACTION's symmetry-only fixtures — a verification-strength gap, NOT a false pass (the bodies are real). No mode-byte-only false pass found in any graded arm.

---

# Backend Guidelines Audit — discrete-per-mode dispatcher codecs (task-096 split)

- **Scope:** ~93 discrete packet codec structs from splitting shared-by-shape codecs in `libs/atlas-packet/{field,npc,storage,cash}/clientbound/` + body-function layers (`{field,storage,cash}/operation_body.go`, `npc/clientbound/shop_operation_body.go`) + `tools/packet-audit/cmd/run.go` #-entries + the npc/storage channel consumers.
- **Diff range:** BASE=52ec9f2fa HEAD=982085708
- **Guidelines source:** backend-dev-guidelines skill. These are immutable packet codec structs (private fields + getters + Encode/Decode/Operation/String), NOT DDD domains — DOM-01..20 (builder/entity/JSON:API/processor) are N/A. Audit focused on: immutability, constructor+Encode+Decode+Operation()+String() consistency vs the reference `npc/clientbound/conversation.go`, DOM-21 (constant/model reuse), copy-paste DRIFT across near-identical structs, and no silent stubs.
- **Build:** PASS (`go build ./...` clean in `libs/atlas-packet` and `services/atlas-channel/atlas.com/channel`)
- **Vet:** PASS (`go vet ./...` clean in `libs/atlas-packet`)
- **Tests:** PASS (`go test ./field/... ./npc/... ./storage/... ./cash/...` all `ok`)
- **Overall:** PASS

## Verification performed

1. **Per-struct DRIFT sweep** (delegated, exhaustive read of all 7 new codec files, 50 structs): checked wrong mode byte (doc-comment hex vs struct literal vs Encode inline comment), copied-but-unedited doc/fname/String, Encode↔Decode field-order/width symmetry, wrong `Operation()` writer const, mutability leaks, stubs, and duplicate mode bytes within a family. **Zero drift found.**
2. **Immutability:** all structs have private fields, value-receiver getters, no exported setters; only `Decode` (pointer receiver) mutates — matches the `NpcConversation` reference. PASS.
3. **DOM-21 (shared-type reuse):** storage assets use `libs/atlas-packet/model.Asset`; storage consumer maps via `libs/atlas-constants/inventory.Type` (`inventory.TypeValueEquip/Use/Setup/ETC/Cash`) — no redeclared inventory enums. PASS.
4. **No dead code after retirement:** retired shared structs (`MtsResultEmpty/Reason/TwoInts`, storage `ErrorSimple`/`UpdateAssets`, cash `WishList`/`CashShopWishListBody`) have zero remaining references in production code (only retirement-note comments). The unrelated `merchant` family still uses its own `ErrorSimple` — out of scope, untouched. PASS.
5. **Callers rewired:** cash handlers (`cash_shop_entry.go:104`, `cash_shop_operation.go:70`) updated from `CashShopWishListBody(bool,…)` to discrete `CashShopWishListLoadBody`/`CashShopWishListUpdateBody`. npc + storage consumers map every error code through the discrete body funcs with a `default` warn arm (no silent drop). PASS.
6. **Test coverage:** per-struct golden-byte + round-trip tests with `packet-audit:verify` IDA markers (19 empty-mode, 7 reason-mode, 2 two-ints, 5 storage, 5 cash, 13 npc). The deleted `mts_operation_body_test.go` (196 lines, tested retired shared structs) is replaced by the new per-struct files. PASS.
7. **DOM-24 (Kafka producer stub):** N/A — the new/changed `*_test.go` are pure codec encode/decode tests; no `AndEmit`/`message.Emit`/`producer.Produce` and no consumer-handler invocation.

## Findings

### Critical
None.

### Important
None.

### Minor
- **MINOR-1 — `String()` naming inconsistency in the cash family.** Every other family prefixes `String()` with an operation-identifying phrase (e.g. `storage error inventory full mode [%d]` in `storage/clientbound/error_modes.go:44`; `mts buy wish done mode [%d]` in `field/clientbound/mts_result_empty_modes.go:44`; `shop operation OK mode [%d]` in `npc/clientbound/shop_operation.go:32`). The cash structs omit any identifier: `cash/clientbound/shop_operation_result.go:33` (`LoadInventoryFailure.String()` → `"mode [%d] errorCode [%d]"`), `:70` (`InventoryCapacitySuccess`), `:107` (`InventoryCapacityFailed`), `:145` (`WishListLoad`), `:192` (`WishListUpdate`). Log lines from these encoders will be ambiguous (no way to tell which cash arm produced them). Non-blocking; cosmetic. Recommend prefixing each with the arm name to match siblings.

## Notes (verified, not findings)
- **Mode-byte sourcing differs by family — intentional, not drift.** The field/MTS structs hardcode their mode byte in the constructor (`{mode: 0x1D}` etc.) because the MTS case labels are IDA-verified version-stable across gms_v83/v84/v87/v95 (documented at `field/clientbound/mts_result_empty_modes.go:20-25`). The npc/storage/cash structs take the mode as a constructor arg resolved per-tenant via `WithResolvedCode` in their body funcs. Both patterns are internally consistent and each struct's golden test asserts the exact mode byte.
- **Field MTS body funcs have no production caller yet** (`field/operation_body.go`). This is documented intentional pre-wiring (`operation_body.go:14-19`: "Atlas has no MTS feature emitting these yet; the body functions … are wired config-driven so a future MTS implementation sends the version-correct mode"). They are fully-implemented, tested, exported library API — not silent stubs. Acceptable for a shared packet-codec library that exposes codecs ahead of service use.

---

# Adversarial Re-Audit — dispatcher discrete-per-mode split (task-096 CField family)

**Date:** 2026-06-17
**Branch:** task-096-cfield-packet-family
**HEAD:** 982085708  **BASE:** 52ec9f2fa (9 commits)
**Auditor mindset:** FAIL until file:line / command evidence proves PASS.

## Verdict per family

| Family | Dispatcher | Verdict |
|--------|-----------|---------|
| MTS_OPERATION | CITC::OnNormalItemResult | **PASS** |
| CONFIRM_SHOP_TRANSACTION | CShopDlg::OnPacket | **PASS** |
| STORAGE | CTrunkDlg::OnPacket | **PASS** |
| CASHSHOP_OPERATION | CCashShop::OnCashItemResult | **PASS** |

All four families: the discrete-per-mode split is honest and wire-correct. One
non-blocking stale-artifact finding (LATENT-1) and the pre-existing items the
prior audit already logged.

## Checklist results

### 1. No struct serves >1 mode; old shared structs deleted — PASS
- `grep "^type (MtsResultEmpty|MtsResultReason|MtsResultTwoInts|ShopOperationSimple|ShopOperationLevelRequirement|StorageErrorSimple|OperationError|CashWishList) "` over `libs/` → **NONE** (all deleted).
- No non-test, non-comment Go reference to any deleted shared struct. Surviving mentions are doc comments in `tools/packet-audit/cmd/run.go` (lines 1691/1705/1901/1907/1972) and the unrelated `CharacterInfo.WishList()` getter (`character/clientbound/info.go:153`).
- `storage/clientbound/update_assets.go` deleted; replaced by discrete `StoreAssets`/`RetrieveAssets` in `store_retrieve_assets.go`.
- npc Over/Under level requirement = two discrete structs (`shop_operation.go:365`, `:397`).
- cash WishList → `WishListLoad` (`shop_operation_result.go:131`) + `WishListUpdate` (`:178`).
- cash OperationError → discrete `LoadInventoryFailure` (`:19`) (+ pre-existing `InventoryCapacityFailed`).

### 2. No body func takes an op/code/mode/key parameter — PASS
- `grep "func.*Body(" … | grep -iE "\b(op|code|mode|key)\b"` across all four body files → **no match**.
- Every body func fixes its operation KEY via `WithResolvedCode("operations", <FIXED_CONST>, …)` (field/storage/npc) or `ResolveCode(…, "operations", <FIXED_CONST>)` (cash failure arms). The only string params present are `reason`/`message` routed to the `errors` table — not an operation key/mode.
- MTS body funcs use `func(_ byte)` (discard the resolved byte; the struct fixes its own mode internally).

### 3. Each Encode writes the real per-mode wire — PASS (spot-checked)
- MTS Reason arm `MtsResultGetItcListFailed.Encode` (`mts_result_reason_modes.go:51`): mode 0x16 + reason byte. ✔
- MTS TwoInts `MtsResultMoveItcPurchaseItemLtoSDone.Encode` (`mts_result_two_ints_modes.go:51`): mode 0x27 + Decode4 tab + Decode4 selectedNo. ✔
- MTS conditional-tail `MtsResultRegisterSaleEntryFailed.Encode` (`mts_operation_body.go:78`): mode 0x1E + reason, +Decode2 short only when reason==0x48. ✔
- MTS conditional-tail `MtsResultSuccessBidInfo.Encode` (`:137`): mode 0x3E + soldFlag + itemId, +price+8-byte FILETIME only when itemId>0. ✔
- Storage `StoreAssets`/`RetrieveAssets.Encode` (`store_retrieve_assets.go:57`,`:110`): mode + slots + flags(8B) + count + asset blobs; distinct fixed mode keys (STORE_ASSETS=13, RETRIEVE_ASSETS=9). ✔
- cash `WishListLoad`/`WishListUpdate.Encode` (`shop_operation_result.go:148`,`:195`): mode + 10×int32 SN buffer; wire-identical shape, distinct keys (LOAD_WISHLIST vs UPDATE_WISHLIST). ✔
- All empty-shape arms write exactly the one mode byte (`mts_result_empty_modes.go`, `storage/clientbound/error_modes.go`).

### 4. Coverage: every family op-row ✅ across applicable versions — PASS
- STATUS.md: CONFIRM_SHOP_TRANSACTION, STORAGE, CASHSHOP_OPERATION dispatcher rows are ✅ for gms_v83/v84/v87/v95/jms_v185. MTS_OPERATION row (line 452) ✅ for gms_v83/v84/v87/v95, ⬜ jms (CITC registry-absent — correct).
- Per-mode evidence completeness:
  - MTS: 35 discrete MtsResult* modes; each has a `.md`+`.json` report (35/35 in gms_v83) and verify markers for **all 4** applicable versions (35×4=140 markers; jms correctly absent). The only Mts struct without a report is `MtsItem` (the embedded ITCITEM list element — not a dispatcher mode; correct).
  - NPC: 13 discrete modes; each has a report; markers for 5 versions (GenericErrorWithReason has no jms marker — jms-absent, registered).
  - STORAGE: StoreAssets/RetrieveAssets/ErrorInventoryFull/ErrorNotEnoughMesos/ErrorOneOfAKind each have report + 5-version markers.
  - CASH: WishListLoad/WishListUpdate/LoadInventoryFailure (+ pre-existing mode structs) each have report + 5-version markers.
- No export `.json` lacks a matching `.md` (only `_unimplemented.json` tooling sidecars, expected).

### 5. Gates + builds — PASS (exit codes captured)
- `go run ./tools/packet-audit matrix --check` → **exit 0**
- `go run ./tools/packet-audit fname-doc --check` → **exit 0** ("212 structs without an audit report carry no fname")
- `go run ./tools/packet-audit operations --check` → **exit 0** (2 pre-existing jms absent-writer notes: CharacterStatusMessage, NoteOperation — unrelated to the 4 families)
- `cd libs/atlas-packet && go build ./...` → **exit 0**; `go vet ./...` → **exit 0**
- `go test -race -count=1 ./field/clientbound ./cash/clientbound ./npc/clientbound ./storage/clientbound` → **all exit 0**
- `cd services/atlas-channel/atlas.com/channel && go build ./...` → **exit 0**
- Consumers updated to per-mode body funcs and build clean:
  - npc shop consumer (`kafka/consumer/npc/shop/consumer.go:107-131`) → all 13 discrete body funcs incl. separate Over/Under.
  - storage consumer (`kafka/consumer/storage/consumer.go:202-208`, `:255/:267` Store, `:310/:322` Retrieve) → discrete Store/Retrieve + 3 error bodies.
  - cash handlers (`socket/handler/cash_shop_entry.go:104` Load, `cash_shop_operation.go:70` Update) → discrete WishListLoad / WishListUpdate.
  - No reference anywhere to any old combined body func (grep for `UpdateAssetsBody|CashShopWishListBody|…LevelRequirementBody|MtsOperation(Empty|Reason|TwoInts)Body` → NONE).

### 6. TODO/stub/faked-byte/orphaned-mode/leftover-shared-struct — 1 LATENT finding
- No NEW TODO/stub/faked-byte lines added by this branch (diff scan of added `+` lines).
- One pre-existing TODO unrelated to the split: `cash/clientbound/shop_operation_body.go:80` `// TODO map codes for JMS — currently hardcoded to 0x4D` in `CashShopCashGiftsBody` (introduced 2026-03-18, commit 6b6a74c9d, predates BASE; CashShopGifts is a separate op, not a shared-shape struct being split). Not a regression.

## LATENT-1 — dangling `#Mode` candidate + stale report cite a DELETED struct/file (non-blocking)

The `MtsOperation` mode-only struct was retired by this campaign, but two
artifacts still point at it:

1. `tools/packet-audit/cmd/run.go:1887-1888` — `case "CITC::OnNormalItemResult#Mode": return …{name: "MtsOperation", …}`. `locateAtlasFile` searches for `type MtsOperation struct`, which **no longer exists** (`grep "^type MtsOperation struct"` → none; file `field/clientbound/mts_operation.go` deleted — only `mts_operation2.go`/`_body.go`/`_list.go` remain).
2. `docs/packets/audits/gms_v83/FieldMtsOperation.md:4` (and v84/v87/v95) — **Atlas file:** `libs/atlas-packet/field/clientbound/mts_operation.go` — a **deleted file**.

Impact: NOT a wire-correctness bug and does NOT fail any gate. The `FieldMtsOperation`
matrix cell stays ✅ via the byte-fixture markers in `mts_operation_test.go:22-25`,
whose golden test now exercises a discrete arm (`NewMtsResultRegisterSaleEntryDone`,
mode 0x1D) rather than the deleted struct. The danger is latent: on a future
report-gen run, `locateAtlasFile("MtsOperation")` returns `found=false` →
report-gen **silently `return`s** (`run.go:67-69`) and skips the cell, freezing the
stale `mts_operation.go`-citing `FieldMtsOperation.md` in place. Recommend either
repointing the `#Mode` candidate to the representative discrete struct
(`MtsResultRegisterSaleEntryDone`) or regenerating `FieldMtsOperation.md` so its
"Atlas file" cite is a file that exists.

## Items I could NOT fully confirm
- **jms version-absence of MTS and of npc GenericErrorWithReason** is taken from the in-repo comments/registry (CITC registry-absent in jms; GenericErrorWithReason has no jms marker) — `operations --check` and `matrix --check` accept these as registered absences (exit 0), but I did not independently decompile the jms_v185 IDB to prove no jms CITC dispatcher / no jms shop-reason arm exists. Treated as PASS on the strength of the green gates + registry notes, flagged here for transparency.
- **Per-version IDA sub-handler addresses** cited in each struct's doc comment (e.g. v95 0x576270) were not re-derived against the IDBs in this audit; they were verified to be internally consistent and the byte-fixtures pin them, but address correctness is inherited from the implementers' decompile, not re-proven here.
