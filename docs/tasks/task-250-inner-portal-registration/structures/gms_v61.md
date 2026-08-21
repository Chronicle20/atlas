# gms_v61 — `USE_INNER_PORTAL` / `CUserLocal::TryRegisterTeleport`

Session: `921fdbb5` (`GMS_v61.1_U_DEVM.exe.i64`)

## USE_INNER_PORTAL

- Export address of `CUserLocal::TryRegisterTeleport`: **`0x7aa1e3`**
  (`?TryRegisterTeleport@CUserLocal@@IAEHPBUSKILLENTRY@@JPBD1H@Z`, already
  named in the IDB — `func_query "*TryRegisterTeleport*"` returns this
  address directly; no rename required for this version).
- `COutPacket` constructor: `COutPacket::COutPacket((COutPacket *)v49, 93);`
  at **`0x7aa2fb`** — opcode constant **`93`** (`0x05D`).
- Registry cross-check: `docs/packets/registry/gms_v61.yaml` — **grepped for
  `TryRegisterTeleport` / `USE_INNER_PORTAL`, no hit.** The registry does not
  yet declare this op; Task 11 adds it with opcode `93`.

### Ordered field table

Re-confirmed directly against the IDB (decompile of `0x7aa1e3`, session
`921fdbb5`):

| # | field | width | client expression |
|---|---|---|---|
| 1 | `fieldKey` | `byte` | `COutPacket::Encode1((COutPacket *)v49, *((_BYTE *)field + 248))` — `field = get_field()` |
| 2 | `portalName` | ASCII string (u16 len + bytes) | `COutPacket::EncodeStr(v47)` — preceded by `ZXString<char>::ReleaseBuffer(Src, 0xFFFFFFFF)`, `Src` = `sPortalName` |
| 3 | `x` | `int16` | `COutPacket::Encode2((COutPacket *)v49, *v15)` — `v15 = this->GetPos()` |
| 4 | `y` | `int16` | `COutPacket::Encode2((COutPacket *)v49, *(_WORD *)(v16 + 4))` — `v16 = this->GetPos()` |
| 5 | `targetX` | `int16` | `COutPacket::Encode2((COutPacket *)v49, *(_WORD *)(v11 + 12))` — `v11` is the target `PORTAL*` from `sub_61FEF2(a5)` (`a5` = `sTargetPortalName`), offset `+12` = `ptPos.x` |
| 6 | `targetY` | `int16` | `COutPacket::Encode2((COutPacket *)v49, *(_WORD *)(v11 + 16))` — same `PORTAL*`, offset `+16` = `ptPos.y` |

6-field layout, matching the gms_v95 reference shape. Send site is gated by
`if (Src)` (`sPortalName != NULL`) inside the branch that follows a
successful `CWvsPhysicalSpace2D::GetFootholdUnderneath` probe under the
target portal.

### Caller

Not re-walked in this pass — same rationale as `gms_v48.md`: the send site
itself was decompiled directly and is sufficient to establish the field
order; Task 2's `version-coverage.md` already confirmed the site is live and
reachable via the skill-teleport call path (`SendSkillUseRequest` sibling
branch present in the same function).

### Per-version delta

Matches gms_v95's six-field shape exactly (`fieldKey, portalName, x, y,
targetX, targetY`). Opcode is `93` (`0x05D`), unique to this version — opcode
is resolved through the per-tenant `operations` table, not part of the field
shape.

## Gate decision

This is the version that closes the `fieldKey` gate boundary. Decompiled
directly in this pass: `0x7aa1e3`'s send site opens with
`COutPacket::Encode1((COutPacket *)v49, *((_BYTE *)field + 248))` immediately
after the `COutPacket` constructor, followed by `EncodeStr` — i.e. `fieldKey`
**is present** on gms_v61. gms_v48's send site (`0x6a5462`, see
`gms_v48.md`), decompiled in the same pass, has **no** `Encode1` call at all.

The two versions adjacent to the boundary — gms_v48 (absent) and gms_v61
(present) — are both directly read in this pass, so the boundary is placed
with direct evidence on both sides, not inferred from one side alone.

**Gate constant: `MajorAtLeast(61)`.** `fieldKey` is emitted when
`tenant.MustFromContext(ctx).MajorAtLeast(61)` is true. gms_v48 is the only
in-scope version below this boundary; every other in-scope version (61, 72,
79, 83, 84, 87, 92, 95, jms_v185) is at or above it and carries `fieldKey`.
Codec `Encode`/`Decode` for `portal/serverbound/PortalInnerPortal` must gate
only this one field on this constant — a raw `> 48` or `> 60` comparison is
not acceptable per PRD FR-2.3; use the `MajorAtLeast` idiom.
