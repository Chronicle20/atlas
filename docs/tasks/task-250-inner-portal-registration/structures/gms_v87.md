# gms_v87 — `USE_INNER_PORTAL` / `CUserLocal::TryRegisterTeleport`

Session: `c0829805` (`GMSv87_4GB.exe.i64`)

## USE_INNER_PORTAL

- Export address of `CUserLocal::TryRegisterTeleport`: **`0x9da037`**
  (`?TryRegisterTeleport@CUserLocal@@IAEHPBUSKILLENTRY@@JPBD1H@Z`, already
  named in the IDB — no rename required for this version).
- `COutPacket` constructor: `COutPacket::COutPacket(&a3, 0x68);` at
  **`0x9da1c0`** — opcode constant **`0x68`** (`104` decimal).
- Registry cross-check: `docs/packets/registry/gms_v87.yaml:2570` —
  `opcode: 104`, `fname: CUserLocal::TryRegisterTeleport`. **Matches.**

### Ordered field table

| # | field | width | client expression |
|---|---|---|---|
| 1 | `fieldKey` | `byte` | `COutPacket::Encode1(&a3, *(field + 328))` — `field = get_field()` |
| 2 | `portalName` | ASCII string (u16 len + bytes) | `COutPacket::EncodeStr(&a3, p_a7)` — built via `ZXString<char>::Assign(Src, 0xFFFFFFFF)` on `sPortalName` (parameter `Src`) |
| 3 | `x` | `int16` | `COutPacket::Encode2(&a3, *v17)` — `v17 = this->GetPos()` |
| 4 | `y` | `int16` | `COutPacket::Encode2(&a3, *(v18 + 4))` — `v18 = this->GetPos()` |
| 5 | `targetX` | `int16` | `COutPacket::Encode2(&a3, *(v13 + 12))` — `v13` is the target `PORTAL*` from `CPortalList::FindPortalByName(ms_pInstance, a6)` (`a6` = `sTargetPortalName`), offset `+12` = `ptPos.x` |
| 6 | `targetY` | `int16` | `COutPacket::Encode2(&a3, *(v13 + 16))` — same `PORTAL*`, offset `+16` = `ptPos.y` |

Identical structure and gating to gms_v95: `if (Src)` (i.e. `sPortalName !=
NULL`) inside the `a6 != NULL` (`sTargetPortalName`) branch.

### Per-version delta

No delta vs gms_v95 — same field order, same widths, same semantics. Only the
opcode differs (`104` vs `113`), which is a per-tenant opcode-table concern,
not a field-shape gate.

## Gate decision

Consistent with gms_v95: **no `MajorAtLeast` gate is required** for the field
layout.
