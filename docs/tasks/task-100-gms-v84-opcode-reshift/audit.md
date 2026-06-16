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
