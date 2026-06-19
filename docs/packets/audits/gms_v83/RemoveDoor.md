# RemoveDoor (← `CTownPortalPool::OnTownPortalRemoved`)

- **IDA:** 0x7be064
- **Atlas file:** `libs/atlas-packet/door/clientbound/remove.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `animate flag (Cosmic writes constant 0; gates removal animation)` | ✅ |  |
| 1 | int32 | int32 `ownerId (door owner character id; registry lookup key)` | ✅ |  |

