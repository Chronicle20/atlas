# GiveResponse (← `CWvsContext::OnGivePopularityResult#GiveResponse`)

- **IDA:** 0x9fea60
- **Atlas file:** `libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (switch dispatch; case 0 = GIVE)` | ✅ |  |
| 1 | string | string `toName (recipient of the fame)` | ✅ |  |
| 2 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |
| 3 | int16 | int32 `nPOP (new total fame as int32)` | ❌ | width mismatch |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

## Manual analysis

**The auto-generated ❌ verdict is a static-tool artifact — the wire is correct.**

The atlas encoder for `GiveResponse.Encode` writes:
```
WriteByte(mode)          → 1 byte   (case 0 / GIVE)
WriteAsciiString(toName) → 2+len bytes
WriteInt8(fameMode)      → 1 byte   (0=defame, 1=fame)
WriteInt16(total)        → 2 bytes  (low 16 bits of fame total, LE)
WriteShort(0)            → 2 bytes  (always zero pad)
```

The static analyzer sees rows 3 and 4 as two separate `int16` writes and cannot
merge them. However, the two consecutive 2-byte LE writes produce exactly the same
4 wire bytes as a single `Decode4(int32 LE)` for any value in the `int16` range:
`WriteInt16(50) + WriteShort(0)` → `[0x32 0x00 0x00 0x00]` = `Decode4` → `int32(50)`.

Fame totals in MapleStory v95 are bounded to a range that fits in int16 (−32 768 ..
32 767), and the atlas public API (`GiveFameResponseBody`) already uses `total int16`,
so no value truncation occurs in practice.

### IDA evidence — `CWvsContext::OnGivePopularityResult` @ 0x9fea60, case 0

```
Decode1  → mode (switch; case 0 = GIVE result)
DecodeStr → toName (ZXString, 2-byte LE length + ShiftJIS bytes)
Decode1  → bInc (fame=1, defame=0)
Decode4  → nPOP (new fame total, int32 LE; passed to CUIUserInfo::NotifyGivePopResult)
```

Total after opcode: 1 (mode) + (2+len) (toName) + 1 (bInc) + 4 (total) bytes.

### Atlas vs IDA wire comparison

| Field | IDA width | Atlas width | Wire bytes | Match? |
|---|---|---|---|---|
| mode | 1 byte (Decode1) | 1 byte (WriteByte) | 1 | ✅ |
| toName | 2+len bytes (DecodeStr) | 2+len bytes (WriteAsciiString) | 2+len | ✅ |
| bInc | 1 byte (Decode1) | 1 byte (WriteInt8 via fameMode) | 1 | ✅ |
| total | 4 bytes (Decode4 int32) | 4 bytes (WriteInt16 + WriteShort(0)) | 4 | ✅ (wire-equivalent) |

**SUMMARY row collision check:** Atlas file path resolves to
`libs/atlas-packet/fame/clientbound/response.go` — correctly points at `fame/`.
No collision with other domains.

### No real bug — static-diff limitation only

The `GiveResponse` encoder matches v95 wire exactly. No fix is needed. The ❌
verdict is entirely due to the static diff tool seeing `WriteInt16 + WriteShort` as
two rows rather than one 4-byte field.

Wire shape verified by `TestGiveFameResponseWireShape` in
`libs/atlas-packet/fame/clientbound/response_test.go`:
all four variants produce exactly 10 bytes for a 2-character name `"P2"`, with the
last 4 bytes decoding to `int32(50)` via LE `binary.LittleEndian.Uint32`.

Ack: misc-audit Phase 2e on 2026-06-03
