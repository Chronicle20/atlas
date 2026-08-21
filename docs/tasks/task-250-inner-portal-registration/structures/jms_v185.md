# jms_v185 — `USE_INNER_PORTAL` / `CUserLocal::TryRegisterTeleport`

Session: `a977912e` (`MapleStory_dump_SCY.exe.i64`)

## USE_INNER_PORTAL

- Export address of `CUserLocal::TryRegisterTeleport`: **`0xa2218f`**
  (`CUserLocal::TryRegisterTeleport`, already named in the IDB — no rename
  required for this version).
- `COutPacket` constructor: `COutPacket::COutPacket(v67, 0x60);` at
  **`0xa22313`** — opcode constant **`0x60`** (`96` decimal).

  **Finding:** the plan's fact block (carried from design.md §1.3) states the
  ctor address as `0xa2230e`. The actual decompiled call-site marker for
  `COutPacket::COutPacket` in this session is `0xa22313`, 5 bytes later
  (consistent with the constructor call following the `push 0x60` /
  argument-setup instructions that begin a few bytes earlier). The opcode
  constant itself (`0x60` / `96`) matches the design doc and the registry
  exactly — only the exact instruction address cited differs by a few bytes,
  most likely because the design doc cited the `push` instruction address and
  this derivation cites the `call` (constructor) address the decompiler
  markers point to. Recorded here per instruction not to silently reconcile.
- Registry cross-check: `docs/packets/registry/jms_v185.yaml:2547` —
  `opcode: 96`, `fname: CUserLocal::TryRegisterTeleport`. **Matches.**

### Ordered field table

| # | field | width | client expression |
|---|---|---|---|
| 1 | `fieldKey` | `byte` | `COutPacket::Encode1(v67, *(field + 328))` — `field = get_field()` |
| 2 | `portalName` | ASCII string (u16 len + bytes) | `COutPacket::EncodeStr(v67, v61)` — built via `ZXString<char>::Assign(&v61, sPortalName, 0xFFFFFFFF)` |
| 3 | `x` | `int16` | `COutPacket::Encode2(v67, *v17)` — `v17 = this->GetPos()` |
| 4 | `y` | `int16` | `COutPacket::Encode2(v67, *(v18 + 4))` — `v18 = this->GetPos()` |
| 5 | `targetX` | `int16` | `COutPacket::Encode2(v67, *(nSLV + 12))` — `nSLV` here aliases the target `PORTAL*` (`sub_778767(sTargetPortalName)` result, IDA reused the `nSLV` stack slot), offset `+12` = `ptPos.x` |
| 6 | `targetY` | `int16` | `COutPacket::Encode2(v67, *(nSLV + 16))` — same `PORTAL*`, offset `+16` = `ptPos.y` |

Identical gating structure: `if (sPortalName)` inside the
`sTargetPortalName != NULL` branch, guarded by
`CWvsPhysicalSpace2D::GetFootholdUnderneath` on the target portal's position.

### Per-version delta

No delta vs gms_v95 in field order, widths, or semantics. Only the opcode
differs (`96` vs `113`).

## Gate decision

Consistent with gms_v95: **no `MajorAtLeast` gate is required** for the field
layout.
