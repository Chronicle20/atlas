# PartyMemberHP (← `CUserRemote::OnReceiveHP`)

- **IDA:** 0x953f50
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/member_hp.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `hp — current HP of the remote party member` | ✅ |  |
| 1 | int32 | int32 `maxHp — max HP of the remote party member` | ✅ |  |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

