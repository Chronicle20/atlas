# Ping (← `CClientSocket::OnAliveReq#PingReceive`)

- **IDA:** 0x4a870a
- **Atlas file:** `libs/atlas-packet/socket/clientbound/ping.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `CClientSocket::OnAliveReq` @ 0x4a870a (clientbound PingReceive): no payload fields. Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03
