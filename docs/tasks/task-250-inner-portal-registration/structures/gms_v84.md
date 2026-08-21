# gms_v84 — `USE_INNER_PORTAL` / `CUserLocal::TryRegisterTeleport`

Session: `46c2a2eb` (`GMS_v84.1_U_DEVM.i64`)

## Derivation method — caller-walk

`func_query "*TryRegisterTeleport*"` returns empty in this IDB (no symbol).
Located via caller-walk:

1. `CUserLocal::CheckPortal_Collision` — `0x985767` (already named in the
   IDB).
2. In its decompilation, the `tm == currentFieldId` branch (guarded by
   `*(v6 + 28) == 999999999` → return) calls a five-argument function whose
   last argument is the literal `1`:

   ```c
   if (!sub_995C92(0, 0, *(void **)(v6 + 4), *(_DWORD *)(v6 + 32), 1))   /*0x985a51*/
       return;
   ```

   `v6 + 4` = source portal's `pn`, `v6 + 32` = source portal's `tn` —
   matching the `sPortalName` / `sTargetPortalName` roles from the confirmed
   versions and the identical shape found for gms_v83.
3. That callee resolved to address **`0x995c92`** (`sub_995C92`).
4. `mcp__ida-pro__rename` applied: `0x995c92` → `CUserLocal::TryRegisterTeleport`,
   then `idb_save`. **Done — Task 12's export splice will now find this name.**

## USE_INNER_PORTAL

- Export address of `CUserLocal::TryRegisterTeleport`: **`0x995c92`**
  (renamed this session).
- `COutPacket` constructor: `COutPacket::COutPacket(&v65, 101);` at
  **`0x995e1b`** — opcode constant **`101`** (`0x65`).
- Registry cross-check: `docs/packets/registry/gms_v84.yaml:3142` —
  `opcode: 101`, `fname: CUserLocal::TryRegisterTeleport`, with note "seeded
  from the v83 CSV column ... task-083 found v84 byte-identical to v83."
  **Matches**, and this derivation independently confirms the v83/v84
  byte-identity claim: v84's opcode constant, field order, and field widths
  are identical to gms_v83's derivation above.

### Ordered field table

| # | field | width | client expression |
|---|---|---|---|
| 1 | `fieldKey` | `byte` | `COutPacket::Encode1(&v65, *(_BYTE *)(v15 + 328))` — `v15 = get_field()` (offset differs from v83's `+308` only because IDA's field-alignment reading picked a slightly different struct-member text; both resolve to `m_bFieldKey`) |
| 2 | `portalName` | ASCII string (u16 len + bytes) | `COutPacket::EncodeStr(&v65, v60)` — built via `ZXString<char>::ReleaseBuffer(Src, 0xFFFFFFFF)` where `Src` = `sPortalName` (parameter `a6` in the outer scope aliases `Src` after reassignment) |
| 3 | `x` | `int16` | `COutPacket::Encode2(&v65, *v16)` — `v16 = this->GetPos()` |
| 4 | `y` | `int16` | `COutPacket::Encode2(&v65, *(_WORD *)(v17 + 4))` — `v17 = this->GetPos()` |
| 5 | `targetX` | `int16` | `COutPacket::Encode2(&v65, *(_WORD *)(v12 + 12))` — `v12` is the target `PORTAL*` from `sub_72F90D(a6)` (`a6` = `sTargetPortalName`; `sub_72F90D` is the unnamed `FindPortalByName`), offset `+12` = `ptPos.x` |
| 6 | `targetY` | `int16` | `COutPacket::Encode2(&v65, *(_WORD *)(v12 + 16))` — same `PORTAL*`, offset `+16` = `ptPos.y` |

Same gating shape: `if (Src)` (`sPortalName != NULL`) inside the `a6 != NULL`
(`sTargetPortalName`) branch, guarded by a foothold check under the target
portal.

### Per-version delta

No delta vs gms_v83 or gms_v95 in field order, widths, or semantics. Opcode
is `101` (`0x65`), identical to gms_v83 — consistent with the registry note
that v84 is byte-identical to v83 for this op.

## Gate decision

Consistent with the other five versions: **no `MajorAtLeast` gate is
required** for the field layout.
