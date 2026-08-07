# RpsOperation (← `CRPSGameDlg::OnBtStart`)

- **IDA:** 0x6c2160
- **Atlas file:** `libs/atlas-packet/rps/serverbound/operation.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op byte (0=OnBtStart @0x6c213a, Encode1(0) @0x6c2160, COutPacket(133); bodyless arm, no further fields -- byte-signature 68 85 00 00 00 8D locates all 6 RPS_ACTION senders; live IDA port 13340 2026-07-16)` | ✅ |  |

