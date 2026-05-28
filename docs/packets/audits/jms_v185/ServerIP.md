# ServerIP (← `CLogin::OnSelectCharacterResult`)

- **IDA:** 0x6712b1
- **Atlas file:** `../../libs/atlas-packet/login/clientbound/server_ip.go`
- **Variant:** JMS/v185
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode (v3)` | ✅ |  |
| 1 | byte | byte `subMode (v14)` | ✅ |  |
| 2 | bytes | bytes `ipv4 (4 bytes)` | ✅ |  |
| 3 | int16 | int16 `port` | ✅ |  |
| 4 | int32 | int32 `dwCharacterID` | ✅ |  |
| 5 | byte | byte `bAuthenCode` | ✅ |  |
| 6 | int32 | int32 `ulPremiumArgument` | ✅ |  |

