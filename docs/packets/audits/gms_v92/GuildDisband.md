# GuildDisband (← `CWvsContext::OnGuildResult#Disband`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `dispatcher-family arm not harvested for gms_v92 (see notes)` | 🚫 | IDA read-order unresolved: dispatcher-family arm not harvested for gms_v92 (see notes) |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

