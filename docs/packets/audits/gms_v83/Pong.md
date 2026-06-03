# Pong (← `CClientSocket::OnAliveReq#PongSend`)

- **IDA:** 0x4966c0
- **Atlas file:** `../../libs/atlas-packet/socket/serverbound/pong.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|


## Manual analysis

**v83 IDA:** `CClientSocket::OnAliveReq` @ 0x4966c0 — serverbound Pong sends opcode 24 with empty body (0 bytes). Matches v95 exactly.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
