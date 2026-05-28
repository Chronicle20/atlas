# OperationMemoryGameMoveStone (← `COmokDlg::PutStoneChecker`)

- **IDA:** 0x6801e0
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_memory_game_move_stone.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `point (tagPOINT x,y = 8 bytes)` | ❌ | width mismatch |
| 1 | byte | byte `color (m_nPlayerColor)` | ✅ |  |


> 🔍 tool limitation, NOT a wire bug. Atlas WriteInt64(point) == client
> EncodeBuffer(pt,8) (tagPOINT x,y); byte-identical wire. See
> `docs/packets/ida-exports/_pending.md` → "Interaction tool-limitation false positives".
