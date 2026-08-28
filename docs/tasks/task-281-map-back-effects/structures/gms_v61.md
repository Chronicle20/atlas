# gms_v61 — SET_BACK_EFFECT / CLEAR_BACK_EFFECT

IDB session: `921fdbb5` (`GMS_v61.1_U_DEVM.exe.i64`)
Export: none used — live IDB.

> **FINDING — both ops are PRESENT on gms_v61.** The plan expected this version to
> be a VERSION-ABSENT proof for both ops, on the strength of the checked-in export
> recording `CMapLoadable::OnSetBackEffect` as `unresolved: true`. That was a lead,
> not evidence: the handler exists, it is simply unnamed in this IDB (`sub_5A8316`).
> `docs/packets/registry/gms_v61.yaml` currently has **no** entry for either op.
> This is the same failure mode the controller ruling flagged for v72/v79.

Step 0 (already implemented?): NO. `grep -rl BackEffect libs/atlas-packet/`
returns nothing — genuinely new codec.

## Router

Reached from `CField::OnPacket` @ `0x4e9ea3`:

```c
if ( pExceptionObject < 95 || pExceptionObject > 96 ) { ... }
else CMapLoadable::OnPacket(pExceptionObject, (int)a3);   /*call site 0x4ea215*/
```

The CMapLoadable opcode window on this version is exactly **95..96 — two opcodes
wide**. Opcode 97 is already owned by `CField::OnTransferFieldReqIgnored`
(`0x4ea2de`) in the same switch, so nothing else can belong to CMapLoadable.

`CMapLoadable::OnPacket` @ `0x5a81b9` (symbol
`?OnPacket@CMapLoadable@@UAEXJAAVCInPacket@@@Z`; the decompiler renders it
`__stdcall sub_5A81B9` because the `this` pointer is elided):

```c
int __stdcall sub_5A81B9(int a1, int a2)
{
  if ( a1 == 95 )                       /*0x5a81c0*/
    return sub_5A8316(a2);              /*0x5a81da*/
  result = a1 - 96;                     /*0x5a81c2*/
  if ( a1 == 96 )                       /*0x5a81c3*/
    return sub_5A871B(a2);              /*0x5a81cc*/
  return result;                        /*0x5a81df*/
}
```

- case `95` (`0x5F`) -> `CMapLoadable::OnSetBackEffect` = `sub_5A8316` @ `0x5a8316`
- case `96` (`0x60`) -> `CMapLoadable::OnClearBackEffect` = `sub_5A871B` @ `0x5a871b`

**Divergence vs v72+:** gms_v61 has **no `OnSetMapObjectVisible` arm**. From v72
onward the trio is `set / set-map-object-visible / clear` at N / N+1 / N+2; on
v61 it is `set / clear` at N / N+1. `CLEAR_BACK_EFFECT` is therefore
`SET_BACK_EFFECT + 1` here, not `+ 2`. Any opcode assigned by analogy to a later
version would be off by one.

## SET_BACK_EFFECT read order

Decode callee: `sub_5163AE` @ `0x5163ae` (unnamed in this IDB; the
`Field::BackEffect::Decode` body — called from `sub_5A8316` as `sub_5163AE(a1)`;
the decompiler mis-attributes the `this` pointer):

```c
int __thiscall sub_5163AE(_DWORD *this, CInPacket *a2)
{
  this[4] = (unsigned __int8)CInPacket::Decode1(a2);   /*0x5163c2*/
  this[5] = CInPacket::Decode4(a2);                    /*0x5163cc*/
  this[6] = (unsigned __int8)CInPacket::Decode1(a2);   /*0x5163d9*/
  result  = CInPacket::Decode4(a2);                    /*0x5163dc*/
  this[7] = result;                                    /*0x5163e2*/
  return result;
}
```

| # | Read | Width | Field |
|---|---|---|---|
| 1 | Decode1 | byte  | nEffect |
| 2 | Decode4 | int32 | nFieldID |
| 3 | Decode1 | byte  | nPageID |
| 4 | Decode4 | int32 | tDuration |

Total 10 bytes.

Branch shape, from `sub_5A8316` @ `0x5a8316`: the handler tests the decoded
`nEffect` slot (`v29`, the `this[4]` written by the decode).
- `nEffect == 0`: page lookup `sub_5A8E05(&v30, v36)` (`0x5a84ec`) keyed on
  `nPageID` (`v30`), end time `v31 + <update time>`, layer walk, vtable `+144`
  tween (call through `*(*v20 + 144)`) with alpha target **255**, then
  `IWzVector2D::RelMove`.
- `nEffect == 1`: identical shape (`sub_5A8E05` @ `0x5a8380`), alpha target **0**.
- any other value: neither arm is entered; the handler falls to its epilogue
  without touching the field.

`sub_5A8E05` (the back-layer `ZMap ... GetAt`) has exactly **2 xrefs**, both
inside `sub_5A8316` — one per branch. Nothing else in the binary reads the
back-layer page map.

Verdict vs the gms_v95 reference (design §1.1): **IDENTICAL** read order, widths
and branch semantics. Only the opcode differs (and the missing
`SetMapObjectVisible` neighbour).

## CLEAR_BACK_EFFECT

Handler @ `0x5a871b` — an 8-byte thunk:

```
sub_5A871B:
  0x5a871b  call sub_5A81E2
  0x5a8720  retn 4
```

Packet reads: **none**. The `CInPacket&` argument is passed straight through and
never touched.

`sub_5A81E2` @ `0x5a81e2` is `CMapLoadable::ReloadBack` — size `0x134`, matching
the reference shape exactly:

```c
int __thiscall sub_5A81E2(char *this)
{
  sub_5A8F43(this + 176);                       // RemoveAll on the back-layer ZMap member
  ...  (*(*v3 + 100))(v3, Buf2 /*vtEmpty*/, ...) // IWzGr2D::Getcenter, vtbl+100
  ...  (*(*v6 + 64))(*v6, 0, 0)                  // IWzGr2D::Getcenter, vtbl+64, (0,0)
  sub_5A0B76(v16);                               // CMapLoadable::RestoreBack(this)
  Dragon = CUser::GetDragon();
  CUser::GetVecCtrl(Dragon, v12);                // 0x485a49
  CAnimationDisplayer::Effect_Quest();           // mislabelled CAnimationDisplayer::SetCenterOrigin
  return sub_429345(v12[0]);
}
```

`sub_5A81E2` has exactly **1 xref**: the thunk `sub_5A871B` at `0x5a871b`. That is
the router arm — i.e. the only caller of ReloadBack on this version is the
clear-back-effect packet handler.

Verdict vs the gms_v95 reference (design §1.2): **IDENTICAL** (bare opcode, empty
body, whole-field reload).

## Downstream consequences

- Both gms_v61 cells are **PRESENT**, not VERSION-ABSENT.
- `docs/packets/registry/gms_v61.yaml` needs entries added:
  `SET_BACK_EFFECT` opcode **95**, fname `CMapLoadable::OnSetBackEffect`;
  `CLEAR_BACK_EFFECT` opcode **96**, fname `CMapLoadable::OnClearBackEffect`.
  Both fnames are **unresolved in the checked-in gms_v61 export** (`sub_5A8316` /
  `sub_5A871B` are unnamed in the IDB), so `evidence pin` will fail until the
  functions are named in the IDB and the export regenerated. Flag to the
  controller before task 3/4 wires this version.
- `template_gms_61_1.json` needs both routes.
