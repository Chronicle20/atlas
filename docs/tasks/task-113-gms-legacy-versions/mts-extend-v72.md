# MTS / ITC (CITC) extension to GMS v72 — task-113

IDA: `GMS_v72.1_U_DEVM.exe`, port 13339. All opcodes/mode bytes/bodies verified
from the v72 send-sites (distrust-IDB-symbols; body-verified). Codecs are
version-agnostic (from `main`); only opcodes + the dispatcher mode table differ
per version. **v72's 17 ITC operation mode bytes are identical to v83/v79; only
the four container opcodes + the two clientbound opcodes differ.** Nearly all of
v72's CITC senders are **inlined/unnamed** (unlike v79's named `CITC::On*`
functions): 19 were renamed in the v72 IDB + `idb_save`d so the export harvests
them.

## Serverbound container opcodes (verified)

| Op | v72 opcode | v83 | v79 | Fname (v72 addr) | Body |
|----|-----------|-----|-----|------------------|------|
| ENTER_MTS | 154 / 0x9A | 156 | 153 | CWvsContext::SendMigrateToITCRequest @0x90c9bd (COutPacket(154) @0x90cad6) | bodiless |
| ITC_STATUS_CHARGE | 239 / 0xEF | 251 | 241 | CITC::OnStatusCharge @0x561585 (renamed sub_561585) | bodiless |
| ITC_QUERY_CASH_REQUEST | 240 / 0xF0 | 252 | 242 | CITC::TrySendQueryCashRequest @0x561879 (renamed sub_561879) | bodiless |
| ITC_OPERATION | 241 / 0xF1 | 253 | 243 | dispatcher (17 modes, below) | mode-byte + per-arm body |

ENTER_MTS: `COutPacket(154)` @0x90cad6 then SendPacket, RemoveAll, zero Encode —
bodiless. STATUS_CHARGE (@0x561585) sets the latch **before** the send (structural
twin of v79 OnStatusCharge@0x57a1b0); QUERY_CASH (@0x561879) sets it **after**
(twin of v79 TrySendQueryCashRequest@0x57a4a4) — this pairing disambiguates the
two consecutive bodiless opcodes 239/240.

## ITC_OPERATION mode table (v72 — dispatcher opcode 0xF1/241)

Re-derived from the v72 CITC send switch. **Every mode byte is identical to v83.**
`operations` table for v72 == v83 == v79.

| Mode | value | v72 sender (addr) |
|------|-------|-------------------|
| REGISTER_SALE | 2 | CITC::OnRegisterSaleEntry arg0==0 @0x5615e1 (send @0x56173d) |
| SALE_CURRENT_ITEM | 3 | CITC::OnSaleCurrentItem @0x5617ea |
| REGISTER_WISH_ENTRY | 4 | CITC::OnRegisterWishEntry @0x561eb4 |
| GET_ITC_LIST | 5 | OnChangedCategory @0x561c09 / OnChangedCategorySub @0x561ce8 / OnChangedPage @0x561dd7 |
| SEARCH_ITC_LIST | 6 | CITCWnd_Tab::OnButtonClicked @0x578c91 (btn 1004, renamed sub_578C91) |
| CANCEL_SALE | 7 | CITC::OnCancelSaleItem @0x5624ec (YesNo-gated) |
| TAKE_HOME | 8 | CITC::OnMoveITCPurchaseItemLtoS @0x562595 |
| SET_ZZIM | 9 | CITC::OnSetZzim @0x56220f |
| DELETE_ZZIM | 10 (0xA) | CITC::OnDeleteZzim @0x562320 |
| VIEW_WISH | 11 (0xB) | CITC::OnViewWish @0x562393 |
| BUY_WISH | 12 (0xC) | CITC::OnBuyWish @0x562406 |
| CANCEL_WISH | 13 (0xD) | CITC::OnCancelWish @0x562479 |
| BUY | 16 (0x10) | CITC::OnBuy @0x562009 |
| BUY_ZZIM | 17 (0x11) | CITC::OnBuyZzim @0x562282 (YesNo-gated) |
| REGISTER_AUCTION | 18 (0x12) | CITC::OnRegisterSaleEntry arg0==1 @0x5615e1 (send @0x561673) |
| PLACE_BID | 19 (0x13) | CITCBidAuctionDlg::OnButtonClicked @0x584a31 (already named) |
| BUY_AUCTION_IMM | 20 (0x14) | CITC::OnBuyAuctionImm @0x56207c (already named) |

No v72-absent modes (all 17 present; **0 n-a**). Each mode's body was decompiled
and confirmed byte-identical to the gms_v83 codec (item-slot blob via sub_4CF950,
serial-only arms read `nITCSN` at `*(item+4)+32`, browse/search EncodeStr order).

## Clientbound (wired, matrix-❌)

| Op | v72 opcode | v83 | v79 | Fname (v72 addr) |
|----|-----------|-----|-----|------------------|
| SET_ITC (SetItc writer) | 115 / 0x73 | 126 | 119 | CStage::OnSetITC @0x6c2145 (CStage::OnPacket @0x6c0c61 case 's') |
| MTS_CHARGE_PARAM_RESULT | 309 / 0x135 | 346 | 322 | CITC::OnChargeParamResult @0x566768 (renamed sub_566768) |

SET_ITC was already in the v72 registry (opcode 115, from the SetField/SetITC/
SetCashShop 'r'/'s'/'t' char-literal switch). CITC OnPacket dispatch @0x56672C:
`mov eax,[esp+4]; sub eax,0x135` → 0x135 OnChargeParamResult (opens charge
web-site), 0x136 OnQueryCashResult, 0x137 OnNormalItemResult. Both writers are
wired in registry + template for runtime routing but remain matrix-❌ like every
other version: `candidatesFromFName` has no CStage::OnSetITC / CITC::OnChargeParamResult
report mapping, so `run` generates no report and the cell cannot promote.

## item_use_point_reset / skill_macro

- `CharacterCashItemUseHandle` (item_use_point_reset): ALREADY wired in the v72
  template @0x4E and present in the registry (USE_CASH_ITEM). No change.
- `CharacterSkillMacro` writer + `CharacterSkillMacroHandle`: ALREADY wired in the
  v72 template and registry (MACRO_SYS_DATA_INIT). No change.

## Artifacts

- **Export**: 22 serverbound CITC entries spliced into `gms_v72.json` (501
  insertions, 0 deletions — additive). `calls` arrays copied from v79 (byte-shape
  is version-agnostic); top-level addresses/notes set to v72.
- **Registry** (`gms_v72.yaml`): +5 entries (ENTER_MTS, ITC_STATUS_CHARGE,
  ITC_QUERY_CASH_REQUEST, ITC_OPERATION serverbound + MTS_CHARGE_PARAM_RESULT
  clientbound). SET_ITC already present.
- **Template** (`template_gms_72_1.json`): +4 handlers (EnterMts 0x9A / LoggedIn,
  ItcStatusCharge 0xEF / NoOp, ItcQueryCash 0xF0 / LoggedIn, ItcOperation 0xF1 /
  LoggedIn + operations mode table) and +2 writers (SetItc 0x73,
  MtsChargeParamResult 0x135).
- **Audit reports**: 22 serverbound `Field{EnterMts,ItcStatusCharge,
  ItcQueryCashRequest,ItcOperation*}.{json,md}` in `docs/packets/audits/gms_v72/`.
- **Evidence**: 22 TIER1-FIXTURE records in `docs/packets/evidence/gms_v72/`.
- **Byte-fixtures**: `libs/atlas-packet/field/serverbound/itc_mts_v72_test.go`
  (22 tests, gms_v72 context, per-packet verify markers with v72 addresses).

## Verification

- `go test ./libs/atlas-packet/...` green; `go vet ./...` clean (atlas-packet +
  tools/packet-audit).
- `go run ./tools/packet-audit matrix --check` **exit 0** (run from worktree
  root); zero problem lines mentioning any Itc/Mts/Enter packet; v72 conflicts 0.
- The `run` full pipeline exits 1 only on the pre-existing malformed
  `CCashShop::TrySendQueryCashRequest` export entry (main-side); no unrelated
  reports were re-churned (only the 22 new serverbound reports copied in).

## Coverage delta

v72 verified cells **216 → 237 (+21)** — matches the v79 agent's +21 exactly
(3 standalone ops + 1 dispatcher op row + 17 mode sub-structs, ITC_OPERATION op
row grades worst-of its arms). No regression on any other version:
v48 165 / v61 208 / v79 249 / v83 389 / v84 366 / v87 400 / v95 420 / jms 383 —
all unchanged.
