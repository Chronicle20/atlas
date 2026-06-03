# Change (← `CWvsContext::SendGivePopularityRequest`)

- **IDA:** 0x9f67e0
- **Atlas file:** `libs/atlas-packet/fame/serverbound/change.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwCharacterId (target character ID as uint32)` | ✅ |  |
| 1 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |

## Manual analysis

**IDA function:** `CWvsContext::SendGivePopularityRequest` @ 0x9f67e0

The client encodes the give-fame request packet with two fields:

```
Encode4 → m_dwCharacterId (RemoteUserByName->m_dwCharacterId, uint32 LE)
Encode1 → bInc (1=fame / 0=defame, passed as ZRef<GW_ItemSlotBase>* in decompilation
                but used as a boolean byte)
```

Total: 5 bytes after opcode.

### Atlas decoder (`fame/serverbound/change.go`)

```
ReadUint32() → targetId  (4 bytes LE)
ReadInt8()   → mode      (1 byte; 1=fame, 0=defame)
```

### Wire comparison

| Field | IDA width | Atlas width | Match? |
|---|---|---|---|
| dwCharacterId | 4 bytes (Encode4) | 4 bytes (ReadUint32) | ✅ |
| bInc | 1 byte (Encode1) | 1 byte (ReadInt8) | ✅ |

**SUMMARY row collision check:** There is a collision: both
`libs/atlas-packet/fame/serverbound/change.go` and
`libs/atlas-packet/field/serverbound/change.go` define `type Change struct`.
`locateAtlasFile` performs an alphabetical `WalkDir`; `fame` sorts before `field`,
so the tool resolves to `fame/serverbound/change.go` — the correct file.
The SUMMARY row confirms: `libs/atlas-packet/fame/serverbound/change.go` ✅.

### No bug — already correct

`Change.Decode` matches v95 exactly. The ✅ static-diff verdict is accurate.

Wire shape verified by `TestChangeFameWireShape` in
`libs/atlas-packet/fame/serverbound/change_test.go`:
all four variants produce exactly 5 bytes, with bytes 0–3 being `uint32(99999)` in
LE and byte 4 being `int8(1)`.

Ack: misc-audit Phase 2e on 2026-06-03
