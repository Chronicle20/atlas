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

**Task 2 update:** this ruling holds only across the six versions Task 1
scoped. Task 2's re-derivation of v48/v61/v72/v79
(`../version-coverage.md`) found all four **present** — v61/v72/v79 match
this six-field shape, but **v48 omits the `fieldKey` byte** (5 fields, no
`Encode1` call at its send site). If the plan is amended to bring v48 into
scope, the "no gate is required" ruling above no longer holds and a real
field-shape gate is needed for v48. See `../version-coverage.md` for the
full finding.

## Threshold derivation

Collision rect half-extents (CUserLocal::CheckPortal_Collision @0x919a10):

`CheckPortal_Collision` itself does not inline the rect test — it calls
`CPortalList::FindPortal_Collision(v2, *v4, *(v5+4))` (`0x6ab310`), which
builds the actual collision rectangle:

```c
ZAPI.SetRect(
  &rc,
  p->ptPos.x - p->nHRange / 2,
  p->ptPos.y - p->nVRange / 2,
  p->ptPos.x + p->nHRange / 2,
  p->ptPos.y + p->nVRange / 2);
result = ZAPI.PtInRect(&rc, __PAIR64__(y, x));
```

`nHRange`/`nVRange` are per-portal fields loaded from WZ map data in
`CPortalList::RestorePortal` (`0x6ad3c0`), each read via a
`get_int32(propertyValue, defaultValue)` call keyed by property id. Both
reads pass a literal default when the map's portal node omits the property:

```c
v122.llVal = v33 | 0x6400000000LL;     // high dword = 0x64 = 100 (default)
...
v35 = IWzProperty::Getitem(v14, &v157, /* prop id 5122 = "hRange" */);
p_t->nHRange = get_int32(v35, v122.cyVal.Hi);   // default 100

v122.llVal = v36 | 0x6400000000LL;     // high dword = 0x64 = 100 (default)
...
v38 = IWzProperty::Getitem(v14, &v148, /* prop id 5215 = "vRange" */);
p_t->nVRange = get_int32(v38, v122.cyVal.Hi);   // default 100
```

So on the common case — a portal whose WZ node does not override
`hRange`/`vRange` — the client uses:

  halfWidth  = 100 / 2 = 50
  halfHeight = 100 / 2 = 50

Diagonal bound: ceil(sqrt(halfWidth^2 + halfHeight^2)) = ceil(sqrt(50^2 + 50^2)) = ceil(70.7107) = 71

Movement-latency margin: 10 map units, because `CUserLocal::TryRegisterTeleport`
(the send site this collision check leads into) itself uses a 10-unit
positional-tolerance literal at the only other place in this call path where
the client tests "close enough" to a portal: `v16 = PortalByName->ptPos.y - 10`
before probing `CWvsPhysicalSpace2D::GetFootholdUnderneath` for the
destination portal's ground. The client's actual walk-speed-per-tick
distance (`max_walk_speed`, `0x992b20`) is not a static constant — it scales
a runtime `CONSTANTS.dWalkSpeed` value loaded from WZ `Physics.img` at
runtime, which is not resolvable as a literal from this IDB — so rather than
invent a px/tick figure, the margin reuses the one concrete positional-slop
literal the client itself applies in this exact code path.

maxPortalEntryDistance = 81   # map coordinate units (71 diagonal bound + 10 margin)
