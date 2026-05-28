# FieldWarpToMap (← `CStage::OnSetField#WarpToMap`)

- **IDA:** 0x71a0a0
- **Atlas file:** `libs/atlas-packet/field/clientbound/warp_to_map.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `DecodeOpt — atlas WriteShort(0) gated (GMS>83 \|\| JMS)` | ✅ |  |
| 1 | int32 | int32 `m_nChannelID (channel id)` | ✅ |  |
| 2 | byte | int32 `m_dwOldDriverID — v95 reads unconditionally; atlas omits for GMS (DIVERGENCE)` | ❌ | width mismatch |
| 3 | byte | byte `sNotifierMessage flag (atlas writes 0)` | ✅ |  |
| 4 | int16 | byte `bCharacterData flag (atlas writes 0 — warp path)` | ❌ | width mismatch |
| 5 | byte | int16 `nNotifierCheck (atlas writes 0)` | ❌ | width mismatch |
| 6 | int32 | byte `revive flag (else branch, line 180) — atlas WriteByte(0) gated (GMS>28 \|\| JMS)` | ❌ | width mismatch |
| 7 | byte | int32 `dwPosMap (target map id)` | ❌ | width mismatch |
| 8 | int32 | byte `nPortal (target portal id)` | ❌ | width mismatch |
| 9 | byte | int32 `nHP — v95 Decode4; atlas was WriteShort(2), FIXED to WriteInt(4)` | ❌ | width mismatch |
| 10 | int64 | byte `m_bChaseEnable — atlas WriteBool(false) gated (GMS>28)` | ❌ | width mismatch |
| 11 | byte | int64 `timestamp (DecodeBuffer p,8u); atlas WriteInt64` | ❌ | atlas: short — missing trailing field |


## Audit notes

🔍 **envelope-only path:** WarpToMap reuses the `CStage::OnSetField` handler with `bCharacterData=0` (the else branch, lines 178-215) — an in-game warp without the full CharacterData re-init. No CharacterData blob is embedded; the audit is full for the warp envelope.

⚠️ **Analyzer limitation:** the wire-diff cascade above is an alignment artifact of the single `m_dwOldDriverID` divergence at position 2 — every row after it is shifted by one, not an independent bug. Read the manual findings below.

### Manual verdict (GMS v95, `CStage::OnSetField` else branch @0x71a0a0)

| Field | IDA (v95) | Atlas (post-fix) | Match |
|---|---|---|---|
| DecodeOpt | Decode2 | WriteShort(0) gated (GMS>83 \|\| JMS) | ✅ |
| channelId | Decode4 (line 128) | WriteInt | ✅ |
| **m_dwOldDriverID** | **Decode4 (line 129, unconditional)** | **omitted for GMS** | **❌ DIVERGENCE (shared with SetField; deferred to _pending.md)** |
| sNotifierMessage | Decode1 (line 132) | WriteByte(0) | ✅ |
| bCharacterData | Decode1 (line 133) | WriteByte(0) | ✅ |
| nNotifierCheck | Decode2 (line 134) | WriteShort(0) gated (GMS>28 \|\| JMS) | ✅ |
| revive flag | Decode1 (line 180, else branch) | WriteByte(0) gated (GMS>28 \|\| JMS) | ✅ |
| dwPosMap (mapId) | Decode4 (line 198) | WriteInt | ✅ |
| nPortal | Decode1 (line 203) | WriteByte | ✅ |
| **nHP** | **Decode4 (line 204 / asm 0x71a320)** | **WriteInt (FIXED — was WriteShort)** | ✅ **(fix applied)** |
| m_bChaseEnable | Decode1 (line 207) | WriteBool(false) gated (GMS>28) | ✅ |
| timestamp | DecodeBuffer(p, 8) (line 235) | WriteInt64 | ✅ |

**Fix applied (this task):** `nHP` is read as `Decode4` (4 bytes) in v95 — confirmed at the assembly level (`0x71a320 call Decode4@CInPacket` storing into `_ZtlSecureTear_nHP`). Atlas previously wrote `WriteShort(m.hp)` (2 bytes); changed to `WriteInt(uint32(m.hp))` (Encode + Decode symmetric). With this fix and `m_dwOldDriverID` removed, the atlas warp envelope aligns 1:1 with v95 (11 ops each).

**Remaining divergence:** `m_dwOldDriverID` — same root cause and same deferral as FieldSetField (see `_pending.md`).

Ack: world-audit Phase 2c on 2026-05-28
