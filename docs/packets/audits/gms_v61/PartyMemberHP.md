# PartyMemberHP (← `CUserRemote::OnReceiveHP`)

- **IDA:** 0x7cc0e6
- **Atlas file:** `libs/atlas-packet/party/clientbound/member_hp.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

