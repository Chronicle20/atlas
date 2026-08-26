# gms_v72 — `USE_INNER_PORTAL` / `CUserLocal::TryRegisterTeleport`

Session: `99e435d8` (`GMS_v72.1_U_DEVM.exe.i64`)

## USE_INNER_PORTAL

- Export address of `CUserLocal::TryRegisterTeleport`: **`0x864562`**
  (`?TryRegisterTeleport@CUserLocal@@IAEHPBUSKILLENTRY@@JPBD1H@Z`, already
  named in the IDB — `func_query "*TryRegisterTeleport*"` returns this
  address directly; no rename required for this version).
- `COutPacket` constructor: `COutPacket::COutPacket((COutPacket *)v59, 100);`
  at **`0x864676`** — opcode constant **`100`** (`0x064`).
- Registry cross-check: `docs/packets/registry/gms_v72.yaml` — **grepped for
  `TryRegisterTeleport` / `USE_INNER_PORTAL`, no hit.** The registry does not
  yet declare this op; Task 11 adds it with opcode `100`.

### Ordered field table

Re-confirmed directly against the IDB (decompile of `0x864562`, session
`99e435d8`):

| # | field | width | client expression |
|---|---|---|---|
| 1 | `fieldKey` | `byte` | `COutPacket::Encode1((COutPacket *)v59, *((_BYTE *)field + 276))` — `field = get_field()` |
| 2 | `portalName` | ASCII string (u16 len + bytes) | `COutPacket::EncodeStr(v55)` — preceded by `ZXString<char>::ReleaseBuffer(Src, 0xFFFFFFFF)`, `Src` = `sPortalName` |
| 3 | `x` | `int16` | `COutPacket::Encode2((COutPacket *)v59, *v15)` — `v15 = this->GetPos()` |
| 4 | `y` | `int16` | `COutPacket::Encode2((COutPacket *)v59, *(_WORD *)(v16 + 4))` — `v16 = this->GetPos()` |
| 5 | `targetX` | `int16` | `COutPacket::Encode2((COutPacket *)v59, *(_WORD *)(v11 + 12))` — `v11` is the target `PORTAL*` from `sub_6799A8(a5)` (`a5` = `sTargetPortalName`), offset `+12` = `ptPos.x` |
| 6 | `targetY` | `int16` | `COutPacket::Encode2((COutPacket *)v59, *(_WORD *)(v11 + 16))` — same `PORTAL*`, offset `+16` = `ptPos.y` |

6-field layout, matching the gms_v95 reference shape. Send site is gated by
`if (Src)` (`sPortalName != NULL`) inside the branch that follows a
successful `CWvsPhysicalSpace2D::GetFootholdUnderneath` probe under the
target portal.

### Caller

Not re-walked in this pass — same rationale as `gms_v48.md`: the send site
itself was decompiled directly, and its `SendSkillUseRequest`-sibling branch
and `CUserLocal::SendSkillUseRequest` call at the end of the function confirm
it is the live `TryRegisterTeleport` body, consistent with the other nine
versions' derivation.

### Per-version delta

Matches gms_v95's six-field shape exactly (`fieldKey, portalName, x, y,
targetX, targetY`). Opcode is `100` (`0x064`), unique to this version.

## Gate decision

Consistent with the boundary established in `gms_v61.md`: gms_v72's send
site carries the `Encode1` (`fieldKey`) call ahead of `EncodeStr`, same as
every version from gms_v61 upward. **No new gate boundary here** —
`fieldKey` presence is governed by the single `MajorAtLeast(61)` constant
recorded in `gms_v61.md` and `gms_v48.md`. The other five fields
(`portalName`, `x`, `y`, `targetX`, `targetY`) are ungated across all ten
in-scope versions.
