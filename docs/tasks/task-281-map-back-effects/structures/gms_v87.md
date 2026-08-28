# gms_v87 — SET_BACK_EFFECT / CLEAR_BACK_EFFECT

IDB session: `c0829805` (`GMSv87_4GB.exe.i64`)
Export: none used — live IDB. Symbols are mangled; look up
`?OnPacket@CMapLoadable@@UAEXJAAVCInPacket@@@Z`.

Step 0 (already implemented?): NO. `grep -rl BackEffect libs/atlas-packet/` returns
nothing, so this is a genuinely new codec, not a wrapper over a shared decoder.

## Router

`CMapLoadable::OnPacket` @ `0x67db5c`:

```c
switch ( nType )                                                     /*0x67db65*/
{
  case 0x88: CMapLoadable::OnSetBackEffect((this - 8), iPacket); break;       /*0x67db90*/
  case 0x89: CMapLoadable::OnSetMapObjectVisible((this - 8), iPacket); break; /*0x67db82*/
  case 0x8A: CMapLoadable::OnClearBackEffect(&this[-1].m_tRestoreBgmVolume, iPacket); break; /*0x67db74*/
}
```

- case `0x88` (136) -> `CMapLoadable::OnSetBackEffect` @ `0x67dcdb`
- case `0x8A` (138) -> `CMapLoadable::OnClearBackEffect` @ `0x67e0e0`

Opcode cross-check: `docs/packets/registry/gms_v87.yaml:716,726` record
`SET_BACK_EFFECT opcode: 136` (`0x88`) and `CLEAR_BACK_EFFECT opcode: 138` (`0x8A`).
Both MATCH the router and the task opcode table (v87 = `0x088` / `0x08A`).

## SET_BACK_EFFECT read order

Decode callee: `sub_5B6E0B` @ `0x5b6e0b` (unnamed in this IDB; called from
`OnSetBackEffect` as `sub_5B6E0B(&a2->m_bLoopback)` — the decompiler mis-attributes
the `this` pointer, but the callee is the `Field::BackEffect::Decode` body):

```c
int __thiscall sub_5B6E0B(_DWORD *this, CInPacket *a2)
{
  this[4] = CInPacket::Decode1(a2);   /*0x5b6e1f*/
  this[5] = CInPacket::Decode4(a2);   /*0x5b6e29*/
  this[6] = CInPacket::Decode1(a2);   /*0x5b6e36*/
  result  = CInPacket::Decode4(a2);   /*0x5b6e39*/
  this[7] = result;                   /*0x5b6e3f*/
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

Branch shape, from `CMapLoadable::OnSetBackEffect` @ `0x67dcdb`: the handler tests the
decoded `nEffect` slot (`v23`, struct `+0x10`).
- `nEffect == 0`: page lookup `sub_67EF22(&v24, v30)` keyed on `nPageID` (`v24`),
  end time `v25 + get_update_time()` (i.e. `tDuration` + now), layer walk, vtable
  `+144` tween with alpha target **255**.
- `nEffect == 1`: identical shape, alpha target **0**.
- any other value: neither arm is entered; the handler falls to its epilogue without
  touching the field.

Verdict vs the gms_v95 reference (design §1.1): **IDENTICAL**.

## CLEAR_BACK_EFFECT

Handler @ `0x67e0e0`:

```c
// attributes: thunk
void __thiscall CMapLoadable::OnClearBackEffect(_DWORD *this, CInPacket *iPacket)
{
  CMapLoadable::ReloadBack(this);   /*0x67e0e0*/   // ReloadBack @ 0x67dba7
}
```

Packet reads: **none**. `iPacket` is untouched.

Verdict vs the gms_v95 reference (design §1.2): **IDENTICAL** (bare opcode, empty
body).
