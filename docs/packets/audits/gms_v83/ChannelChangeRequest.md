# ChannelChangeRequest (← `CField::SendTransferChannelRequest`)

- **IDA:** 0x5304af
- **Atlas file:** `../../libs/atlas-packet/channel/serverbound/channel_change.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nTargetChannel (target channel ID, 0-based byte @0x53055b)` | ✅ |  |
| 1 | int32 | int32 `get_update_time() (client tick / update time, uint32 @0x530569)` | ✅ |  |


## Manual analysis

**v83 IDA:** `CField::SendTransferChannelRequest` @ 0x5304af — Encode1(nTargetChannel), Encode4(get_update_time()). Matches v95 exactly.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
