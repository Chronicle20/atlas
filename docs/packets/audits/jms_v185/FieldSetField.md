# FieldSetField (← `CStage::OnSetField`)

- **IDA:** 0x7eea69
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/set_field.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `DecodeOpt (CClientOptMan::DecodeOpt @line119) — atlas WriteShort(0) gated (GMS>83 \|\| JMS)` | ✅ |  |
| 1 | int32 | int32 `m_nChannelID (channel id @line121)` | ✅ |  |
| 2 | byte | byte `JMS-only byte (@line122 → szCookie[76]) — atlas WriteByte(0) gated JMS` | ✅ |  |
| 3 | int32 | int32 `JMS-only int (@line123 → szReserved[1976]) — atlas WriteInt(0) gated JMS` | ✅ |  |
| 4 | byte | byte `sNotifierMessage flag (@line126)` | ✅ |  |
| 5 | byte | byte `bCharacterData flag (@line127; ==1 here)` | ✅ |  |
| 6 | int16 | int16 `nNotifierCheck / notifier-string count (@line128)` | ✅ |  |
| 7 | int32 | int32 `damage seed 1 (@line159)` | ✅ |  |
| 8 | int64 | int32 `damage seed 2 (@line160)` | ❌ | width mismatch |
| 9 | byte | int32 `damage seed 3 (@line161)` | ❌ | width mismatch |
| 10 | byte | bytes `CharacterData::Decode (@line165) — ENVELOPE BOUNDARY; inner shape under character domain` | ❌ | width mismatch |
| 11 | byte | int32 `m_bPredictQuit (OnSetLogoutGiftConfig@0xae81c0) — logout gift int #1, gated (GMS>83 \|\| JMS)` | ❌ | width mismatch |
| 12 | int32 | int32 `logout gift commodity SN #1` | ✅ |  |
| 13 | int32 | int32 `logout gift commodity SN #2` | ✅ |  |
| 14 | int32 | int32 `logout gift commodity SN #3` | ✅ |  |
| 15 | int32 | int64 `timestamp DecodeBuffer(p,8 @line209; FILETIME) — atlas WriteInt64` | ❌ | width mismatch |
| 16 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual envelope verdict (JMS v185, `CStage::OnSetField` @0x7eea69)

The flat ❌ is the documented seed-loop + CharacterData-boundary analyzer artifact
(same class as GMS v95 — the `for i:=0;i<3 { WriteInt }` seed loop collapses to one
representative op and the cross-package `WriteByteArray(characterData.Encode)` recurse
is dropped, so positional alignment past the seed loop is unreliable). Every envelope
field matches JMS185 IDA:

| Field | JMS185 IDA | Atlas | Match |
|---|---|---|---|
| DecodeOpt | Decode2 (@line119) | WriteShort(0) gated (GMS>83 \|\| JMS) | ✅ |
| channelId | Decode4 (@line121) | WriteInt | ✅ |
| **JMS byte** | **Decode1 (@line122, szCookie[76])** | **WriteByte(0) gated JMS** | ✅ |
| **JMS int** | **Decode4 (@line123, szReserved[1976])** | **WriteInt(0) gated JMS** | ✅ |
| sNotifierMessage | Decode1 (@line126) | WriteByte(1) | ✅ |
| bCharacterData | Decode1 (@line127) | WriteByte(1) | ✅ |
| nNotifierCheck | Decode2 (@line128) | WriteShort(0) gated (GMS>28 \|\| JMS) | ✅ |
| damage seeds | 3× Decode4 (@lines159-161) | 3× WriteInt gated (GMS>28 \|\| JMS) | ✅ |
| CharacterData | CharacterData::Decode (@line165) | WriteByteArray(characterData.Encode) | 🔍 boundary |
| logout gifts | 4× Decode4 (OnSetLogoutGiftConfig@0xae81c0) | 4× WriteInt(0) gated (GMS>83 \|\| JMS) | ✅ |
| timestamp | DecodeBuffer(p,8 @line209; FILETIME) | WriteInt64 | ✅ |

**JMS byte+int block VERIFIED:** JMS185 reads Decode1 then Decode4 immediately after
channelId (lines 122-123) — atlas's `WriteByte(0); WriteInt(0)` JMS-only block matches.
**4 logout-gift ints VERIFIED:** OnSetLogoutGiftConfig@0xae81c0 reads Decode4 + 3× Decode4
loop = 4 ints. No JMS set_field sub-divide — the JMS block is a single shape (no early-JMS
vs 185+ split inside the Region=="JMS" branch); the 3-deep set_field nesting cap was NOT hit.

Ack: world-audit Phase 3 JMS185 field+portal on 2026-05-28
