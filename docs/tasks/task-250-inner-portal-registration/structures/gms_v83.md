# gms_v83 — `USE_INNER_PORTAL` / `CUserLocal::TryRegisterTeleport`

Session: `754107bf` (`MapleStory_dump.exe.i64`, v83_Me)

## Derivation method — caller-walk

`func_query "*TryRegisterTeleport*"` returns empty in this IDB (no symbol).
Located via caller-walk:

1. `CUserLocal::CheckPortal_Collision` — `0x94dac6` (already named in the
   IDB).
2. In its decompilation, the `tm == currentFieldId` branch (guarded by
   `v6[7] == 999999999` → return, mirroring the `tm != 999999999` guard) calls
   a five-argument function whose last argument is the literal `1`:

   ```c
   if (!CUserLocal::TryRegisterTeleport(v43, 0, 0, v6[1], v6[8], 1))   /*0x94dd98*/
       return;
   ```

   `v6[1]` = source portal's `pn`, `v6[8]` = source portal's `tn` — matching
   the `sPortalName` / `sTargetPortalName` roles from the confirmed versions.
3. That callee resolved to address **`0x957b74`**.
4. `mcp__ida-pro__rename` applied: `0x957b74` → `CUserLocal::TryRegisterTeleport`,
   then `idb_save`. **Done — Task 12's export splice will now find this name.**

## USE_INNER_PORTAL

- Export address of `CUserLocal::TryRegisterTeleport`: **`0x957b74`**
  (renamed this session).
- `COutPacket` constructor: `COutPacket::COutPacket(v58, 0x65);` at
  **`0x957c8a`** — opcode constant **`0x65`** (`101` decimal).
- Registry cross-check: `docs/packets/registry/gms_v83.yaml:2454` —
  `opcode: 101`, `fname: CUserLocal::TryRegisterTeleport`. **Matches.**

### Ordered field table

| # | field | width | client expression |
|---|---|---|---|
| 1 | `fieldKey` | `byte` | `COutPacket::Encode1(v58, *(field + 308))` — `field = get_field()` |
| 2 | `portalName` | ASCII string (u16 len + bytes) | `COutPacket::EncodeStr(v58, v55[0])` — built via `ZXString<char>::GetBuffer(v55, this, a4, 0xFFFFFFFF)` where `a4` = `sPortalName` |
| 3 | `x` | `int16` | `COutPacket::Encode2(v58, *v15)` — `v15 = this->GetPos()` |
| 4 | `y` | `int16` | `COutPacket::Encode2(v58, *(v16 + 4))` — `v16 = this->GetPos()` |
| 5 | `targetX` | `int16` | `COutPacket::Encode2(v58, *(v11 + 12))` — `v11` is the target `PORTAL*` from `sub_712B24(dword_BED768, a5)` (`a5` = `sTargetPortalName`; `dword_BED768` is the unnamed `CPortalList::ms_pInstance`, `sub_712B24` is the unnamed `FindPortalByName`), offset `+12` = `ptPos.x` |
| 6 | `targetY` | `int16` | `COutPacket::Encode2(v58, *(v11 + 16))` — same `PORTAL*`, offset `+16` = `ptPos.y` |

Same gating shape as the confirmed versions: `if (a4)` (`sPortalName != NULL`)
inside the `a5 != NULL` (`sTargetPortalName`) branch, guarded by a foothold
check under the target portal.

### Per-version delta

No delta vs gms_v95 in field order, widths, or semantics. Opcode is `101`
(`0x65`) vs `113` for v95.

## Gate decision

Consistent with the other five versions: **no `MajorAtLeast` gate is
required** for the field layout — the layout is byte-identical across
gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, and jms_v185. Only the opcode
differs per version, resolved through the per-tenant `operations` opcode
table.
