# MTS / ITC (CITC) extension to GMS v79 — task-113

IDA: `GMS_v79_1_DEVM.exe`, port 13340. All opcodes/mode bytes/bodies verified from
the v79 send-sites (distrust-IDB-symbols; body-verified). Codecs are
version-agnostic (from `main`); only opcodes + the dispatcher mode table differ
per version. v79's ITC operation mode bytes are **identical to v83**; only the
container opcodes differ.

## Serverbound opcodes (verified)

| Op | v79 opcode | v83 opcode | Fname (v79 addr) | Body |
|----|-----------|-----------|------------------|------|
| ENTER_MTS | 153 / 0x99 | 156 / 0x9C | CWvsContext::SendMigrateToITCRequest @0x95dd85 (COutPacket(153) @0x95de9e) | bodiless |
| ITC_STATUS_CHARGE | 241 / 0xF1 | 251 / 0xFB | CITC::OnStatusCharge @0x57a1b0 (COutPacket(241) @0x57a1d2) | bodiless |
| ITC_QUERY_CASH_REQUEST | 242 / 0xF2 | 252 / 0xFC | CITC::TrySendQueryCashRequest @0x57a4a4 (COutPacket(242) @0x57a4c6) | bodiless |
| ITC_OPERATION | 243 / 0xF3 | 253 / 0xFD | dispatcher (19 arms, below) | mode-byte + per-arm body |

All three bodiless requests: COutPacket(opcode) immediately followed by
SendPacket with ZERO Encode calls (m_bITCRequestSent latch not on the wire).
Confirmed identical to the v83 codec (empty body).

## ITC_OPERATION mode table (v79 — dispatcher opcode 0xF3/243)

Re-derived from the v79 CITC send switch. **Every mode byte is identical to v83**,
so the tenant `operations` table for v79 == v83.

| Mode | value | v79 sender (addr) |
|------|-------|-------------------|
| REGISTER_SALE | 2 | CITC::OnRegisterSaleEntry arg0==0 @0x57a20c |
| SALE_CURRENT_ITEM | 3 | CITC::OnSaleCurrentItem @0x57a415 |
| REGISTER_WISH_ENTRY | 4 | CITC::OnRegisterWishEntry @0x57aadf |
| GET_ITC_LIST | 5 | OnChangedCategory @0x57a834 / OnChangedCategorySub @0x57a913 / OnChangedPage @0x57aa02 |
| SEARCH_ITC_LIST | 6 | CITCWnd_Tab::OnButtonClicked @0x5919d4 |
| CANCEL_SALE | 7 | CITC::OnCancelSaleItem @0x57b117 |
| TAKE_HOME | 8 | CITC::OnMoveITCPurchaseItemLtoS @0x57b1c0 |
| SET_ZZIM | 9 | CITC::OnSetZzim @0x57ae3a |
| DELETE_ZZIM | 10 (0xA) | CITC::OnDeleteZzim @0x57af4b |
| VIEW_WISH | 11 (0xB) | CITC::OnViewWish @0x57afbe |
| BUY_WISH | 12 (0xC) | CITC::OnBuyWish @0x57b031 |
| CANCEL_WISH | 13 (0xD) | CITC::OnCancelWish @0x57b0a4 |
| BUY | 16 (0x10) | CITC::OnBuy @0x57ac34 |
| BUY_ZZIM | 17 (0x11) | CITC::OnBuyZzim @0x57aead |
| REGISTER_AUCTION | 18 (0x12) | CITC::OnRegisterSaleEntry arg0==1 @0x57a20c |
| PLACE_BID | 19 (0x13) | CITCBidAuctionDlg::OnButtonClicked @0x59da55 |
| BUY_AUCTION_IMM | 20 (0x14) | CITC::OnBuyAuctionImm @0x57aca7 |

No v79-absent modes (all 17 present; 0 n-a).

## Clientbound (verified)

| Op | v79 opcode | v83 opcode | Fname (v79 addr) | Body |
|----|-----------|-----------|------------------|------|
| SET_ITC (SetItc writer) | 119 / 0x77 | 126 / 0x7E | CStage::OnSetITC @0x6f1c4a (reader sub_57925A) | CharacterData envelope + acct + 5 config int32 + 8-byte FILETIME |
| MTS_CHARGE_PARAM_RESULT (MtsChargeParamResult) | 322 / 0x142 | 346 / 0x15A | CITC::OnChargeParamResult @0x57f3d7 | bodiless (client reads nothing; opens web site) |

CITC::OnPacket dispatch decoded (`mov eax,[esp+4]; sub eax,0x142`):
0x142→OnChargeParamResult, 0x143→OnQueryCashResult, 0x144→OnNormalItemResult.

## item_use_point_reset / skill_macro

- `CharacterCashItemUseHandle` (item_use_point_reset / AP-reset cash item):
  ALREADY wired in the v79 template @0x4D and present in the registry as
  USE_CASH_ITEM (CWvsContext::SendConsumeCashItemUseRequest @0x95634a, op 77).
  No change needed.
- `CharacterSkillMacro` writer: ALREADY wired in the v79 template @0x75 and
  present in the registry as MACRO_SYS_DATA_INIT (CWvsContext::OnMacroSysDataInit,
  op 117). No change needed.

## Matrix promotability note

- Serverbound (ENTER_MTS, ITC_STATUS_CHARGE, ITC_QUERY_CASH_REQUEST,
  ITC_OPERATION + 19 FieldItcOperation* sub-structs) promote for v79.
- Clientbound SET_ITC / MTS_CHARGE_PARAM_RESULT are wired (registry + template)
  but remain ❌ like EVERY other version: `candidatesFromFName` has no
  CStage::OnSetITC / CITC::OnChargeParamResult report mapping, so `run` generates
  no report and the matrix cannot promote them anywhere (v83 SET_ITC = ❌,
  v83 IDA_0X15A = ❌). Wiring is added for runtime routing; verification parity
  with v83 is "wired, unreported" — not a v79 regression.
