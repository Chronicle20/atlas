# ServerIP (← `CLogin::OnSelectCharacterResult`)

- **IDA:** 0x63319a
- **Atlas file:** `libs/atlas-packet/login/clientbound/server_ip.go`
- **Variant:** GMS/v87
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode` | ✅ |  |
| 1 | byte | byte `subMode` | ✅ |  |
| 2 | bytes | bytes `ipv4 (4 octets — stored as Decode4 in client; same wire bytes)` | ✅ |  |
| 3 | int16 | int16 `port` | ✅ |  |
| 4 | int32 | int32 `dwCharacterID` | ✅ |  |
| 5 | byte | byte `bAuthenCode ((Decode1>>1)&1)` | ✅ |  |
| 6 | int32 | int32 `ulPremiumArgument — present in v87 at LABEL_48 (same as v95)` | ✅ |  |

