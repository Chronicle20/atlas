# Request (← `CLogin::SendCheckPasswordPacket`)

- **IDA:** 0x5b1170
- **Atlas file:** `libs/atlas-packet/login/serverbound/request.go`
- **Variant:** GMS/v72
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `` | ✅ |  |
| 1 | string | string `` | ✅ |  |
| 2 | bytes | bytes `` | ✅ |  |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | byte | byte `` | ✅ |  |
| 5 | byte | byte `` | ✅ |  |
| 6 | byte | byte `` | ✅ |  |
| 7 | int32 | int32 `` | ✅ |  |

