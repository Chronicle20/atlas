# FieldWarpToMap (← `CStage::OnSetField#WarpToMap`)

- **IDA:** 0x71a0a0
- **Atlas file:** `libs/atlas-packet/field/clientbound/warp_to_map.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `DecodeOpt — atlas WriteShort(0) gated (GMS>83 \|\| JMS)` | ✅ |  |
| 1 | int32 | int32 `m_nChannelID (channel id)` | ✅ |  |
| 2 | int32 | int32 `m_dwOldDriverID — v95 reads Decode4 unconditionally; atlas now emits it gated GMS>=95 (RESOLVED, v83 omits)` | ✅ |  |
| 3 | byte | byte `sNotifierMessage flag (atlas writes 0)` | ✅ |  |
| 4 | byte | byte `bCharacterData flag (atlas writes 0 — warp path)` | ✅ |  |
| 5 | int16 | int16 `nNotifierCheck (atlas writes 0)` | ✅ |  |
| 6 | byte | byte `revive flag (else branch, line 180) — atlas WriteByte(0) gated (GMS>28 \|\| JMS)` | ✅ |  |
| 7 | int32 | int32 `dwPosMap (target map id)` | ✅ |  |
| 8 | byte | byte `nPortal (target portal id)` | ✅ |  |
| 9 | int32 | int32 `nHP — v95 Decode4; atlas was WriteShort(2), FIXED to WriteInt(4)` | ✅ |  |
| 10 | byte | byte `m_bChaseEnable — atlas WriteBool(false) gated (GMS>28)` | ✅ |  |
| 11 | int64 | int64 `timestamp (DecodeBuffer p,8u); atlas WriteInt64` | ✅ |  |


## Audit notes

🔍 **envelope-only path:** WarpToMap reuses the `CStage::OnSetField` handler with
`bCharacterData=0` (else branch). No CharacterData blob is embedded; the audit is
full for the warp envelope and all 12 rows align 1:1.

**DEFERRED BUG RESOLVED — m_dwOldDriverID (task-068 Phase 3 v83):** the v95
deferral is closed. v83 IDA (`CStage::OnSetField` @0x776020) reads channelId then
`sNotifierMessage` immediately, with NO old-driver-id between them — so the field
was introduced after v83. Atlas now emits the 4-byte `m_dwOldDriverID` gated on
`GMS && MajorVersion>=95` (see `warp_to_map.go`), which is correct for v95 (row 2
✅) and omitted for v83/v87. `nHP` remains the earlier fix (Decode4 for GMS
v95+/JMS, Decode2 for v83/v87). With both gates the v95 warp envelope is fully
✅.

Ack: world-audit Phase 3 v95-refresh on 2026-05-28
