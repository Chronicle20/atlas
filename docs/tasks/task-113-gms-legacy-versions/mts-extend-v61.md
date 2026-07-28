# MTS / ITC (CITC) extension to GMS v61 — task-113

IDA: `GMS_v61.1_U_DEVM.exe`, port 13338. All opcodes/mode bytes/bodies verified
from the v61 send-sites (distrust-IDB-symbols; body-verified). Codecs are
version-agnostic (from `main`); only the container opcodes + the two clientbound
opcodes differ. **v61's 17 ITC operation mode bytes are identical to v83/v72/v79;
only the four container opcodes + the two clientbound opcodes differ.** Nearly all
of v61's CITC senders are **inlined/unnamed** (like v72): 19 were renamed in the
v61 IDB + `idb_save`d so the export harvests them.

## Serverbound container opcodes (verified)

| Op | v61 opcode | v83 | v72 | Fname (v61 addr) | Body |
|----|-----------|-----|-----|------------------|------|
| ENTER_MTS | 135 / 0x87 | 156 | 154 | CWvsContext::SendMigrateToITCRequest @0x839b94 (COutPacket(135) @0x839ca3) | bodiless |
| ITC_STATUS_CHARGE | 213 / 0xD5 | 251 | 239 | CITC::OnStatusCharge @0x528ed7 (renamed sub_528ED7, nId 1000) | bodiless |
| ITC_QUERY_CASH_REQUEST | 214 / 0xD6 | 252 | 240 | CITC::TrySendQueryCashRequest @0x5291cc (renamed sub_5291CC, nId 1001) | bodiless |
| ITC_OPERATION | 215 / 0xD7 | 253 | 241 | dispatcher (17 modes, below) | mode-byte + per-arm body |

ENTER_MTS: `COutPacket(135)` @0x839ca3 then SendPacket, zero Encode — bodiless.
The two bodiless singletons are disambiguated by the CITC-window button
dispatcher `sub_53FBD1`: nId 1000 → OnStatusCharge (latch set **before** the send,
opcode 213), nId 1001 → TrySendQueryCashRequest (latch set **after** the send,
opcode 214) — structural twins of the v72 pairing. ENTER_MTS is a CWvsContext op,
not adjacent to the CITC block.

## ITC_OPERATION mode table (v61 — dispatcher opcode 0xD7/215)

Re-derived from the v61 CITC send switch. **Every mode byte is identical to v83.**
`operations` table for v61 == v83 == v72 == v79.

| Mode | value | v61 sender (addr) |
|------|-------|-------------------|
| REGISTER_SALE | 2 | CITC::OnRegisterSaleEntry a2==0 @0x528f35 (send @0x528f5b) |
| SALE_CURRENT_ITEM | 3 | CITC::OnSaleCurrentItem @0x52913b |
| REGISTER_WISH_ENTRY | 4 | CITC::OnRegisterWishEntry @0x52980f |
| GET_ITC_LIST | 5 | OnChangedCategory @0x529560 / OnChangedCategorySub @0x529640 / OnChangedPage @0x529730 |
| SEARCH_ITC_LIST | 6 | CITCWnd_Tab::OnButtonClicked @0x53e6a9 (nId 1004, send @0x53e741) |
| CANCEL_SALE | 7 | CITC::OnCancelSaleItem @0x529e54 (YesNo-gated) |
| TAKE_HOME | 8 | CITC::OnMoveITCPurchaseItemLtoS @0x529efc |
| SET_ZZIM | 9 | CITC::OnSetZzim @0x529b6e |
| DELETE_ZZIM | 10 (0xA) | CITC::OnDeleteZzim @0x529c80 |
| VIEW_WISH | 11 (0xB) | CITC::OnViewWish @0x529cf5 |
| BUY_WISH | 12 (0xC) | CITC::OnBuyWish @0x529d6a |
| CANCEL_WISH | 13 (0xD) | CITC::OnCancelWish @0x529ddf |
| BUY | 16 (0x10) | CITC::OnBuy @0x529964 |
| BUY_ZZIM | 17 (0x11) | CITC::OnBuyZzim @0x529be3 (YesNo-gated) |
| REGISTER_AUCTION | 18 (0x12) | CITC::OnRegisterSaleEntry a2==1 @0x528f35 |
| PLACE_BID | 19 (0x13) | CITCBidAuctionDlg::OnButtonClicked @0x549672 (nId==1, send @0x549809) |
| BUY_AUCTION_IMM | 20 (0x14) | CITC::OnBuyAuctionImm @0x5299d9 (already named) |

No v61-absent modes (all 17 present; **0 n-a**). Each mode's body was decompiled
and confirmed byte-identical to the gms_v83 codec (item-slot blob via the v61
GW_ItemSlotBase encoder sub_4B4712; serial-only arms read `nITCSN` at
`*(*(item+4)+32)`; browse/search EncodeStr order).

## Clientbound (wired, matrix-❌)

| Op | v61 opcode | v83 | v72 | Fname (v61 addr) |
|----|-----------|-----|-----|------------------|
| SET_ITC (SetItc writer) | 93 / 0x5D | 126 | 115 | CStage::OnSetITC @0x65b3b4 (CStage::OnPacket case ']') |
| MTS_CHARGE_PARAM_RESULT | 273 / 0x111 | 346 | 309 | CITC::OnChargeParamResult @0x52d691 (renamed sub_52D691) |

SET_ITC was already in the v61 registry (opcode 93, CStage OnPacket char-literal
switch). CITC clientbound OnPacket dispatch @0x52D655: `mov eax,[esp+4];
sub eax,0x111` → 0x111 OnChargeParamResult (clears latch, `open_web_site`),
0x112 OnQueryCashResult, 0x113 OnNormalItemResult. Both writers are wired in
registry + template for runtime routing but remain matrix-❌ like every other
version: `candidatesFromFName` has no CStage::OnSetITC / CITC::OnChargeParamResult
report mapping, so `run` generates no report and the cell cannot promote.

## item_use_point_reset / skill_macro

- `CharacterCashItemUseHandle` (item_use_point_reset): ALREADY wired in the v61
  template and present in the registry. No change.
- `CharacterSkillMacro` writer: ALREADY wired in the v61 template. No change.

## Artifacts

- **Export**: 22 serverbound CITC entries spliced into `gms_v61.json` (502
  insertions, 1 deletion — additive). `calls` arrays copied from gms_v72
  (byte-shape is version-agnostic); top-level addresses/notes set to v61.
- **Registry** (`gms_v61.yaml`): +5 entries (ENTER_MTS, ITC_STATUS_CHARGE,
  ITC_QUERY_CASH_REQUEST, ITC_OPERATION serverbound + MTS_CHARGE_PARAM_RESULT
  clientbound). SET_ITC already present.
- **Template** (`template_gms_61_1.json`): +4 handlers (EnterMts 0x87 / LoggedIn,
  ItcStatusCharge 0xD5 / NoOp, ItcQueryCash 0xD6 / LoggedIn, ItcOperation 0xD7 /
  LoggedIn + operations mode table) and +2 writers (SetItc 0x5D,
  MtsChargeParamResult 0x111).
- **Audit reports**: 22 serverbound `Field{EnterMts,ItcStatusCharge,
  ItcQueryCashRequest,ItcOperation*}.{json,md}` in `docs/packets/audits/gms_v61/`
  (19 ✅; 3 🔍 for the item-blob arms RegisterSale/RegisterAuction/SaleCurrentItem,
  same verdict as v72 — byte-fixture markers promote them regardless).
- **Evidence**: 22 TIER1-FIXTURE records in `docs/packets/evidence/gms_v61/`.
- **Byte-fixtures**: `libs/atlas-packet/field/serverbound/itc_mts_v61_test.go`
  (22 tests, gms_v61 context, per-packet verify markers with v61 addresses).
- **run.go**: NO change — `candidatesFromFName` is version-agnostic and already
  maps every CITC fname (from the v79/v72/v83 passes).

## Verification

- `go test ./libs/atlas-packet/...` green (67 packages ok); `go vet ./...` clean
  (atlas-packet + tools/packet-audit).
- `go run ./tools/packet-audit matrix --check` **exit 0** (run from worktree
  root); zero problem lines mentioning any Itc/Mts/Enter packet; v61 conflicts 0.
- The `run` full pipeline exits 1 only on the pre-existing malformed
  `CCashShop::TrySendQueryCashRequest` export entry (main-side); the 182 unrelated
  re-churned reports (tool drift) and 6 unrelated new reports were reverted, so
  only the 22 new serverbound reports were kept.

## Coverage delta

v61 verified cells **208 → 229 (+21)** — matches the v79/v72 agents' +21 exactly
(3 standalone ops + 1 dispatcher op row + 17 mode sub-structs, ITC_OPERATION op
row grades worst-of its arms). No regression on any other version:
v48 165 / v72 237 / v79 249 / v83 389 / v84 366 / v87 400 / v95 420 / JMS185 383 —
all unchanged.
