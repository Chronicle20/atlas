# gms_v95 — `USE_INNER_PORTAL` / `CUserLocal::TryRegisterTeleport`

Session: `ecc757f4` (`GMS_v95.0_U_DEVM.exe.i64`)

## USE_INNER_PORTAL

- Export address of `CUserLocal::TryRegisterTeleport`: **`0x913690`**
  (`?TryRegisterTeleport@CUserLocal@@IAEHPBUSKILLENTRY@@JPBD1H@Z`, already
  named in the IDB — no rename required for this version).
- `COutPacket` constructor: `COutPacket::COutPacket(&rc, 113);` at
  **`0x91387e`** — opcode constant **`113`** (`0x071`).
- Registry cross-check: `docs/packets/registry/gms_v95.yaml:2825` —
  `opcode: 113`, `fname: CUserLocal::TryRegisterTeleport`. **Matches.**

### Ordered field table

| # | field | width | client expression |
|---|---|---|---|
| 1 | `fieldKey` | `byte` | `COutPacket::Encode1(&rc, field->m_bFieldKey)` — `field = get_field()` |
| 2 | `portalName` | ASCII string (u16 len + bytes) | `COutPacket::EncodeStr(&rc, v74[0])` — `v74` built via `ZXString<char>::ZXString<char>(v74, sPortalName, -1)` |
| 3 | `x` | `int16` | `COutPacket::Encode2(&rc, *v19)` — `v19 = this->GetPos()` |
| 4 | `y` | `int16` | `COutPacket::Encode2(&rc, *(v20 + 4))` — `v20 = this->GetPos()` |
| 5 | `targetX` | `int16` | `COutPacket::Encode2(&rc, *(pfh + 12))` — `pfh` is `PORTAL* PortalByName` (the target portal, via `CPortalList::FindPortalByName(ms_pInstance, sTargetPortalName)`), field offset `+12` = `ptPos.x` |
| 6 | `targetY` | `int16` | `COutPacket::Encode2(&rc, *(v21 + 16))` — same `PORTAL*`, offset `+16` = `ptPos.y` |

Send site is gated by `if (sPortalName)` inside the `sTargetPortalName != NULL`
branch of `TryRegisterTeleport`; the sibling branch (`sTargetPortalName ==
NULL`) is the Teleport-skill path and sends `SendSkillUseRequest`, not this
opcode.

### Caller

`CUserLocal::CheckPortal_Collision` (`0x919a10`) calls it at `0x919d92`,
pushing `sPortalName = portal->pn`, `sTargetPortalName = portal->tn`,
`bForced = 1`. `CUserLocal::HandleUpKeyDown` (`0x919e50`) is the second
caller, reaching the same send site.

### Per-version delta

No delta vs the other five versions (gms_v83, gms_v84, gms_v87, gms_v92,
jms_v185) — see the gate decision below.

## Gate decision

All six derived versions (gms_v83, gms_v84, gms_v87, gms_v92, gms_v95,
jms_v185) emit the identical six-field sequence with identical widths and
identical semantics (`fieldKey`, source `portalName`, pre-teleport `x`/`y`,
destination portal's `targetX`/`targetY`). **No `MajorAtLeast` gate is
required** for the field layout — only the opcode differs per version, which
is resolved through the existing per-tenant `operations` opcode table, not a
field-shape gate.
