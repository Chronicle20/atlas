# gms_v79 — SET_BACK_EFFECT / CLEAR_BACK_EFFECT

IDB session: `5a1cd4f3` (`GMS_v79_1_DEVM.exe.i64`)
Export: none used — live IDB. Symbols in this IDB are mangled; look functions up by
their MSVC-mangled name (`?OnPacket@CMapLoadable@@UAEXJAAVCInPacket@@@Z`), the
demangled `CMapLoadable::OnPacket` does not resolve.

Step 0 (already implemented?): NO. `grep -rl BackEffect libs/atlas-packet/` returns
nothing, so this is a genuinely new codec, not a wrapper over a shared decoder.

## Router

`CMapLoadable::OnPacket` @ `0x614406` — an if/else chain, not a switch:

```c
void __thiscall CMapLoadable::OnPacket(CMapLoadable *this, int a2, struct CInPacket *a3)
{
  if ( a2 == 121 )                                                    /*0x61440d*/
    CMapLoadable::OnSetBackEffect((CMapLoadable *)((char *)this - 8), a3); /*0x614427*/
  else if ( a2 == 122 )                                               /*0x614410*/
    CMapLoadable::OnClearBackEffect(a3);                              /*0x614419*/
}
```

- case 121 (`0x79`) -> `CMapLoadable::OnSetBackEffect` @ `0x614572`
- case 122 (`0x7A`) -> `CMapLoadable::OnClearBackEffect` @ `0x614977`

Opcode cross-check: `docs/packets/registry/gms_v79.yaml:859` records
`SET_BACK_EFFECT opcode: 121` (`0x79`). MATCHES the router and the task opcode table
(v79 = `0x079`).

## SET_BACK_EFFECT read order

Decode callee: `sub_56424F` @ `0x56424f` (unnamed in this IDB; called from
`OnSetBackEffect` @ `0x6145b0`):

```c
int __thiscall sub_56424F(_DWORD *this, CInPacket *a2)
{
  this[4] = (unsigned __int8)CInPacket::Decode1(a2);  /*0x564263*/
  this[5] = CInPacket::Decode4(a2);                   /*0x56426d*/
  this[6] = (unsigned __int8)CInPacket::Decode1(a2);  /*0x56427a*/
  result  = CInPacket::Decode4(a2);                   /*0x56427d*/
  this[7] = result;                                   /*0x564283*/
  return result;
}
```

| # | Read | Width | Field |
|---|---|---|---|
| 1 | Decode1 | byte  | nEffect |
| 2 | Decode4 | int32 | nFieldID |
| 3 | Decode1 | byte  | nPageID |
| 4 | Decode4 | int32 | tDuration |

Total 10 bytes. Slot mapping is `this[4]=nEffect`, `this[5]=nFieldID`,
`this[6]=nPageID`, `this[7]=tDuration` — the same `+0x10..+0x1C` layout as the
symbolized `Field::BackEffect::Decode` on v83/v92/v95.

Branch shape, from `CMapLoadable::OnSetBackEffect` @ `0x614572`: the handler tests the
decoded `nEffect` slot (`v36`, the struct's `+0x10` word).
- `nEffect == 0`: takes the arm at `0x614733`, resolves the page from the back-layer
  map (`sub_61504C` @ `0x614748`), walks the `IWzGr2DLayer` list and calls the
  vtable `+144` tween with alpha target **255** (`0x614836`), then
  `IWzVector2D::RelMove` @ `0x61484f`.
- `nEffect == 1`: takes the arm at `0x6145c7`, same shape, alpha target **0**
  (`0x6146c7`), `RelMove` @ `0x6146e0`.
- any other value: neither arm is entered; the handler falls through to its epilogue
  (`0x6148a2`) without touching the field.

The tween end time is `v38 + <update time>` — `tDuration` added to the current clock,
i.e. a fade LENGTH, not an expiry timestamp.

Verdict vs the gms_v95 reference (design §1.1): **IDENTICAL**.

## CLEAR_BACK_EFFECT

Handler @ `0x614977`:

```c
// attributes: thunk
int __stdcall sub_614977(int a1)
{
  return CMapLoadable::ReloadBack();   /*0x61497c*/   // ReloadBack @ 0x61443e
}
```

Packet reads: **none**. The thunk discards its `CInPacket *` argument and tail-calls
`?ReloadBack@CMapLoadable@@QAEXXZ` @ `0x61443e`.

Verdict vs the gms_v95 reference (design §1.2): **IDENTICAL** (bare opcode, empty
body).

FINDING: `docs/packets/registry/gms_v79.yaml` carries **no** `CLEAR_BACK_EFFECT`
entry, but the client router does have a clear arm at opcode 122 (`0x7A`) reached
from `CMapLoadable::OnPacket` @ `0x614410`. The op is PRESENT on v79; the registry
gap is a registry gap, not an absence. Task 2 owns what to do about it.
