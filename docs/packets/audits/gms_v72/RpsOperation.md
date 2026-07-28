# RpsOperation (← `CRPSGameDlg::OnBtStart`)

- **IDA:** 0x69c950
- **Atlas file:** `libs/atlas-packet/rps/serverbound/operation.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op byte (0=OnBtStart @0x69c92a, Encode1(0) @0x69c950, COutPacket(134); bodyless arm, no further fields -- byte-signature 68 86 00 00 00 8D locates all 6 RPS_ACTION senders; live IDA port 13339 2026-07-16)` | ✅ |  |

