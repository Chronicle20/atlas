# ServerIP (← `CLogin::OnSelectCharacterResult`)

- **IDA:** 0x5cf6ee
- **Atlas file:** `libs/atlas-packet/login/clientbound/server_ip.go`
- **Variant:** GMS/v79
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | bytes | int32 `` | ✅ |  |
| 3 | int16 | int16 `` | ✅ |  |
| 4 | int32 | int32 `` | ✅ |  |
| 5 | byte | byte `` | ✅ |  |
| 6 | int32 | int32 `` | ✅ |  |

