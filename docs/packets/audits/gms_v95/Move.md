# Move (← `CVecCtrlUser::EndUpdateActive`)

- **IDA:** 0x9a0d20
- **Atlas file:** `libs/atlas-packet/character/serverbound/move.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dr0 (~drInfo[0])` | ✅ |  |
| 1 | int32 | int32 `dr1 (~drInfo[1])` | ✅ |  |
| 2 | byte | byte `fieldKey` | ✅ |  |
| 3 | int32 | int32 `dr2 (~drInfo[2])` | ✅ |  |
| 4 | int32 | int32 `dr3 (~drInfo[3])` | ✅ |  |
| 5 | int32 | int32 `crc (field CRC for anti-cheat)` | ✅ |  |
| 6 | int32 | int32 `dwKey (random seed for CRC32)` | ✅ |  |
| 7 | int32 | int32 `crc32 (CRC32 of bDetect using dwKey)` | ✅ |  |
| 8 | byte | bytes `movement: CMovePath::Encode — Encode2(x)+Encode2(y)+Encode2(vx)+Encode2(vy)+Encode1(elemCount)+per-elem(nAttr+coords+bMoveAction+tElapse)+Encode1(keyPadStateCount)+keyPadStates+Encode2(rcMove.left/top/right/bottom); tool cannot linearize loop — ack:tool-limitation` | 🔍 | sub-struct: movement — see _substruct/ |

