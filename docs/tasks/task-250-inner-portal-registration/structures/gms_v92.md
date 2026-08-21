# gms_v92 — `USE_INNER_PORTAL` / `CUserLocal::TryRegisterTeleport`

Session: `019cd393` (`GMS_v92_1_DEVM.exe.i64`)

## Derivation method — caller-walk (initially via `SendSkillUseRequest` xrefs; caller-side confirmation closed in a follow-up pass, below)

`func_query "*TryRegisterTeleport*"` returns empty in this IDB (no symbol).
Unlike gms_v83/gms_v84, this IDB also carries **no** `CheckPortal_Collision`,
`FindPortalByName`, or `FindPortal_Collision` symbol at all — `func_query`
for each pattern returns empty. Fell back to the alternate anchor the brief
allows: the family's other data structures / call graph.

1. `CUserLocal::SendSkillUseRequest` resolves (partially demangled) as
   `CUserLocal__SendSkillUseRequest` at **`0x91d310`**.
2. `xrefs_to` on `0x91d310` returns 8 callers. `TryRegisterTeleport` is
   expected among them because its Teleport-skill branch (the sibling of the
   portal branch) calls `SendSkillUseRequest`. The candidates were
   size-filtered against the confirmed versions' sizes (v87 `0x655`, v95
   `0x95c` — several hundred bytes), which ruled out most of the 8 (sizes
   `0x145`, `0x1ea`, `0x195`, `0x139`, `0x134` are too small for a function
   with this much branching).
3. Two size-plausible candidates remained: `sub_91EC80` (`0x61e`) and
   `sub_8F85C0` (`0x822`). Decompiled both:
   - `sub_91EC80` — takes only 3 args, no `COutPacket` constructor, no portal
     parameter. **Ruled out** (it is a different skill-use function, likely
     `TryHeal`-shaped).
   - `sub_8F85C0` — takes 6 args matching `(this, pSkill, nSLV, sPortalName,
     sTargetPortalName, bForced)`; contains `CPortalList::FindPortalByName`
     equivalent (`sub_6A0830`), a `COutPacket::COutPacket(&v74, 0x70u)`
     constructor, and the exact `Encode1, EncodeStr, Encode2 ×4` sequence.
     **Confirmed as `TryRegisterTeleport`.**
4. `mcp__ida-pro__rename` applied: `0x8f85c0` → `CUserLocal::TryRegisterTeleport`,
   then `idb_save`. **Done — Task 12's export splice will now find this name.**

### Caller-side confirmation — CLOSED

The first pass could not find a `CheckPortal_Collision` symbol and recorded
the argument-role confirmation as missing. A follow-up pass closed it: the
function exists in this IDB, it was simply **unnamed**.

`xrefs_to 0x8f85c0` returns four callers:

| caller (v92) | size | gms_v95 counterpart | size |
|---|---|---|---|
| `?OnTeleport@CUserLocal@@QAEXAAVCInPacket@@@Z` @ `0x8f8df0` | `0x82` | `?OnTeleport@CUserLocal@@…` @ `0x913ff0` | `0x82` |
| `sub_8FDBE0` | `0x43c` | `?CheckPortal_Collision@CUserLocal@@IAEXXZ` @ `0x919a10` | `0x43b` |
| `sub_8FE020` | `0x47e` | `?HandleUpKeyDown@CUserLocal@@IAEXXZ` @ `0x919e50` | `0x485` |
| `sub_921B10` | `0x30e9` | `?DoActiveSkill@CUserLocal@@QAEHJIPAJ@Z` @ `0x9445b0` | `0x3b54` |

Four of v95's five callers are present at near-identical sizes, one of them a
byte-identical **named** `CUserLocal::OnTeleport`. (v95's fifth caller,
`MoveToPortal` @ `0x52a1f0`, is a 21-byte thunk — inlined or absent here.)

Decompiling `sub_8FDBE0` confirms it **is** `CheckPortal_Collision`: it
switches on the portal type (`*(_DWORD *)(v6 + 8)`), calls
`CField::SendTransferFieldRequest` in its default arm, and in that same arm
calls the candidate with exactly the shape the caller-walk procedure
specifies — five arguments, last a literal `1` (`bForced`), 3rd and 4th the
portal's `pn` / `tn` string pointers:

```c
if ( !CUserLocal::TryRegisterTeleport(0, 0, *(void **)(v6 + 4), *(_DWORD *)(v6 + 32), 1) )
  return;
```

`v6 + 4` is the portal's `pn` (the same field row 2 encodes as `portalName`)
and `v6 + 32` its `tn`. **The argument-role confirmation gms_v83/gms_v84 have
is therefore established for gms_v92 as well**; this cell is *derived*, not
inferred, and is at the same confidence tier as the other five versions.

`sub_8FDBE0` was renamed to `CUserLocal::CheckPortal_Collision` and the IDB
saved.

#### Adjacent opcode — do not confuse

`CheckPortal_Collision`'s **`case 9:`** arm carries its own inline send with
the adjacent opcode `0x6F` (`111`) and a **four**-field body — `Encode1`
(fieldKey), `EncodeStr` (portalName), `Encode2` (x), `Encode2` (y) — with no
`targetX`/`targetY`. That is a different operation, not a v92 variant of
`USE_INNER_PORTAL`. `USE_INNER_PORTAL` on gms_v92 is `0x70` (`112`), six
fields, inside `TryRegisterTeleport`.

## USE_INNER_PORTAL

- Export address of `CUserLocal::TryRegisterTeleport`: **`0x8f85c0`**
  (renamed this session).
- `COutPacket` constructor: `COutPacket::COutPacket(&v74, 0x70u);` at
  **`0x8f8782`** — opcode constant **`0x70`** (`112` decimal).
- Registry cross-check: `docs/packets/registry/gms_v92.yaml:2790` —
  `opcode: 112`, `fname: CUserLocal::TryRegisterTeleport`. **Matches.**

### Ordered field table

| # | field | width | client expression |
|---|---|---|---|
| 1 | `fieldKey` | `byte` | `COutPacket::Encode1(&v74, *(_BYTE *)(v19 + 332))` — `v19 = get_field()`-equivalent (`sub_43A6F0`) |
| 2 | `portalName` | ASCII string (u16 len + bytes) | `COutPacket::EncodeStr(&v74, v64)` — built via `sub_416270(Src, 0xFFFFFFFF)` where `Src` = `sPortalName` |
| 3 | `x` | `int16` | `COutPacket::Encode2(&v74, *v21)` — `v21 = this->GetPos()` |
| 4 | `y` | `int16` | `COutPacket::Encode2(&v74, *(_WORD *)(v22 + 4))` — `v22 = this->GetPos()` |
| 5 | `targetX` | `int16` | `COutPacket::Encode2(&v74, *(_WORD *)(v66 + 12))` — `v66` is the target `PORTAL*` from `sub_6A0830(a5)` (`a5` = `sTargetPortalName`; `sub_6A0830` is the unnamed `FindPortalByName`), offset `+12` = `ptPos.x` |
| 6 | `targetY` | `int16` | `COutPacket::Encode2(&v74, *(_WORD *)(v23 + 16))` — same `PORTAL*` (`v23 = v66`), offset `+16` = `ptPos.y` |

Same gating shape: `if (Src)` (`sPortalName != NULL`) inside the `a5 != NULL`
(`sTargetPortalName`) branch, guarded by a foothold check
(`sub_9E9FB0` = `GetFootholdUnderneath`) under the target portal.

### Per-version delta

No delta vs gms_v95 in field order, widths, or semantics. Opcode is `112`
(`0x70`).

## Gate decision

Consistent with the other five versions: **no `MajorAtLeast` gate is
required** for the field layout — all six versions (gms_v83, gms_v84,
gms_v87, gms_v92, gms_v95, jms_v185) emit the identical six-field sequence.
