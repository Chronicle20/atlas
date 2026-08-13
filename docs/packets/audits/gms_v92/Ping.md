# Ping (← `CClientSocket::OnAliveReq#PingReceive`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/socket/clientbound/ping.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `dispatcher-family arm not harvested for gms_v92 (see notes)` | ❌ | atlas: short — missing trailing field |

