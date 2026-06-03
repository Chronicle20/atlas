# StartError (← `CClientSocket::OnConnect#StartError`)

- **IDA:** 0x4a6e5a
- **Atlas file:** `libs/atlas-packet/socket/serverbound/start_error.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `length uint16 (Encode2 exception log byte count @0x4a7371)` | ✅ |  |
| 1 | bytes | bytes `bytes variable-length exception log data (EncodeBuffer @0x4a7389)` | ✅ |  |


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `CClientSocket::OnConnect` @ 0x4a6e5a (StartError path, opcode 0x19): Encode2(length) + EncodeBuffer(exception log data). Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03
