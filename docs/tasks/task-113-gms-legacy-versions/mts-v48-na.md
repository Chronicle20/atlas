# MTS/ITC (CITC) disposition — version-absent (n-a) for GMS v48

**Task:** task-113-gms-legacy-versions
**IDB:** `GMS_v48_1_DEVM.exe`, port **13337** (`select_instance(13337)`)
**Authoritative basis (owner-confirmed):** MTS was added in GMS **v53**. v48 (< 53) has NO MTS/ITC protocol — only a CITCWnd UI shell. v61/v72/v79 (all ≥ 53) received the feature (verified separately).

## 1. IDA absence evidence

`func_query name_regex` sweeps run against the v48 IDB:

| Query | Result |
|-------|--------|
| `CITC::\|MigrateToITC\|OnRegisterSaleEntry\|OnGetITCList\|TrySendQueryCash` | **EMPTY** — no `CITC::` packet class, no `SendMigrateToITCRequest` COutPacket send-site |
| `ITC\|Mts\|Auction\|Zzim\|WishList` (broad sweep) | **ONLY 4 UI shells** (below) |

UI shells present (rendering only — no packet I/O):

| Symbol | Address |
|--------|---------|
| `CITCWnd_Inventory::OnCreate` | `0x43c290` |
| `CITCWnd_Inventory::Draw` | `0x43c98c` |
| `CRegisterAuctionEntryDlg::Draw` | `0x448cad` |
| `CCashShop::FindWishList` | `0x44f059` |

**Packet class absent:** no `CITC::` dispatch/send/result functions of any kind
(`CITC::OnNormalItemResult`, `CITC::OnQueryCashResult`, `CITC::OnStatusCharge`,
`CITC::TrySendQueryCashRequest`, `CITC::OnChargeParamResult`, `CITC::OnBuy*`,
`CITCBidAuctionDlg`, `CITCWnd_Tab`, `CWvsContext::SendMigrateToITCRequest`,
`CStage::OnSetITC` — all contain `ITC`/`Mts`/`Auction` and would have surfaced in
the broad sweep; none did). Only `CITCWnd_Inventory` + the two dialog/wishlist
UI shells exist. **Absence confirmed: yes.**

## 2. Cells dispositioned n-a

The op-level MTS/ITC rows already render `n-a` (⬜) in the coverage matrix for v48
(op absent from the v48 registry, opcode -1); they are recorded in the ledger for
completeness. The clientbound `CITC::OnNormalItemResult`/`OnQueryCashResult`
export carried **36 unresolved "function not found in IDB" stub reports**
(`FieldMtsResult*` ×35 + `FieldMtsOperation2`) — these were **stripped** from the
gms_v48 export so grading excludes the absent arms. Stripping altered **no** matrix
cell (all were already consumed as n-a via the op row) — `STATUS.md`/`status.json`
are byte-identical.

**25 entries added** to `docs/packets/audits/gms_v48/_unimplemented.json`
(20 → 45 total), covering:

- `CWvsContext::SendMigrateToITCRequest` (ENTER_MTS / FieldEnterMts)
- `CStage::OnSetITC` (SET_ITC / SetItc)
- `CITC::OnStatusCharge` (ITC_STATUS_CHARGE / FieldItcStatusCharge)
- `CITC::TrySendQueryCashRequest` (ITC_QUERY_CASH_REQUEST / FieldItcQueryCashRequest)
- `CITC::OnChargeParamResult` (MTS_CHARGE_PARAM_RESULT / MtsChargeParamResult)
- `CITC::OnQueryCashResult` (MTS_OPERATION2 / FieldMtsOperation2 — stub stripped)
- `CITC::OnNormalItemResult` (MTS_OPERATION dispatcher / 34 FieldMtsResult* arms — stubs stripped)
- 18 ITC_OPERATION serverbound arms (`CITC::OnBuy`, `OnBuyAuctionImm`, `OnBuyWish`,
  `OnBuyZzim`, `OnCancelSaleItem`, `OnCancelWish`, `OnChangedCategory`,
  `OnChangedCategorySub`, `OnChangedPage`, `OnDeleteZzim`, `OnMoveITCPurchaseItemLtoS`,
  `OnRegisterSaleEntry`, `OnRegisterWishEntry`, `OnSaleCurrentItem`, `OnSetZzim`,
  `OnViewWish`, `CITCBidAuctionDlg::OnButtonClicked`, `CITCWnd_Tab::OnButtonClicked`)

No MTS/ITC ops were added to the v48 registry/template.

## 3. Verification

- `go run ./tools/packet-audit matrix --check` → **exit 0**
- v48 conflicts: **0**; MTS/ITC CITC report files remaining in gms_v48: **0**
- No MTS/ITC orphan/dangling/stale/drift lines in STATUS.md/status.json

**Verified-count regression check (all unchanged):**

| Version | Verified | Version | Verified |
|---------|----------|---------|----------|
| gms_v48 | 165 | gms_v84 | 366 |
| gms_v61 | 229 | gms_v87 | 400 |
| gms_v72 | 237 | gms_v95 | 420 |
| gms_v79 | 249 | jms_v185 | 383 |
| gms_v83 | 389 | | |

Only gms_v48 report files were touched; no other version's cells changed.

## Commit

`task-113(v48): disposition MTS/ITC n-a (feature added v53, absent in v48)`
