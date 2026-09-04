# gms_v92 — SET_BACK_EFFECT / CLEAR_BACK_EFFECT

IDB session: `019cd393` (`GMS_v92_1_DEVM.exe.i64`)
Export: none used — live IDB.

Step 0 (already implemented?): NO. `grep -rl BackEffect libs/atlas-packet/` returns
nothing, so this is a genuinely new codec, not a wrapper over a shared decoder.

## Router

There is **no** `CMapLoadable::OnPacket` function in this IDB. The field-loadable
arms live in a tail chunk of the function IDA names
`?OnPacket@CLogin@@UAEXJAAVCInPacket@@@Z` @ `0x5d6070` — a merged router whose
default branch dispatches the `140-142` (CStage) and `143-145` (CMapLoadable)
ranges. The back-effect sub-switch is the chunk at `0x613ba9`:

```c
else if ( (unsigned int)(a2 - 143) <= 2 )                    /*0x5d6201*/
{
  switch ( a2 )                                              /*0x613ba9*/
  {
    case 143: CMapLoadable::OnSetBackEffect((this - 8), a3);       break; /*0x613bdd*/
    case 144: CMapLoadable::OnSetMapObjectVisible((this - 8), a3); break; /*0x613bcd*/
    case 145: CMapLoadable::OnClearBackEffect(a3);                 break; /*0x613bbd*/
  }
}
```

- case 143 (`0x8F`) -> `CMapLoadable::OnSetBackEffect` @ `0x606d80`
- case 145 (`0x91`) -> `CMapLoadable::OnClearBackEffect` @ `0x612ef0`

Opcode cross-check: `docs/packets/registry/gms_v92.yaml:750,762` record
`SET_BACK_EFFECT opcode: 143` (`0x8F`) and `CLEAR_BACK_EFFECT opcode: 145` (`0x91`).
Both MATCH the router and the task opcode table (v92 = `0x08F` / `0x091`).

This also **resolves the gms_v92 registry caveat** at `gms_v92.yaml:753,765`
("not IDA-confirmed by the discover-ops pass... routed via an if/else or nested-switch
arm that ParseDispatch cannot enumerate"): the arms are now read directly, and the
csv-imported opcodes are correct.

## SET_BACK_EFFECT read order

Decode callee: `Field::BackEffect::Decode` @ `0x55e120` (symbolized in this IDB;
called from `OnSetBackEffect` @ `0x606e00`):

```c
int __thiscall Field::BackEffect::Decode(Field::BackEffect *this, struct CInPacket *a2)
{
  *((_DWORD *)this + 4) = (unsigned __int8)CInPacket::Decode1((int)a2); /*0x55e134*/
  *((_DWORD *)this + 5) = CInPacket::Decode4((int)a2);                  /*0x55e13e*/
  *((_DWORD *)this + 6) = (unsigned __int8)CInPacket::Decode1((int)a2); /*0x55e149*/
  result                = CInPacket::Decode4((int)a2);                  /*0x55e14e*/
  *((_DWORD *)this + 7) = result;                                       /*0x55e154*/
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

Branch shape: Hex-Rays produces a degenerate body for `OnSetBackEffect` @ `0x606d80`
(`Field::BackEffect::Decode(a2); return -2;`) because of the chunked layout, so the
branch was read from the **disassembly**:

```
606e00  call ?Decode@BackEffect@Field@@UAEXAAVCInPacket@@@Z
606e05  mov  eax, [esp+0A0h+var_1C]      ; var_1C == struct +0x10 == nEffect
606e0c  sub  eax, edi                    ; edi == 0
606e0e  jz   loc_607001                  ; nEffect == 0 arm
606e14  sub  eax, 1
606e17  jnz  loc_606FE5                  ; any other value -> epilogue, no action
                                         ; falls through: nEffect == 1 arm
```

- `nEffect == 1` arm (falls through at `0x606e1d`): the tween call at `0x606f2e` is
  reached with `push 0` at `0x606f2c` — alpha target **0** (hide).
- `nEffect == 0` arm (`loc_607001`): the same tween site is reached with
  `push 0FFh` at `0x607108` — alpha target **255** (show).
- any other value: `loc_606FE5`, the epilogue; the field is not touched.

Verdict vs the gms_v95 reference (design §1.1): **IDENTICAL**.

## CLEAR_BACK_EFFECT

Handler @ `0x612ef0` (Hex-Rays refuses this 2-instruction thunk; read from
disassembly):

```
612ef0  call sub_612D80
612ef5  retn 4
```

`sub_612D80` @ `0x612d80` takes only `this` (`int __thiscall sub_612D80(char *this)`)
— no `CInPacket` parameter — and rebuilds the back layers (`sub_605BD0(this + 232)`,
two COM calls, `sub_612760`), which is the `ReloadBack` body. `?ReloadBack@CMapLoadable@@`
is not symbolized in this IDB, so treat the *name* as inferred; the load-bearing fact
— that the handler consumes **no packet bytes** — is proven by the thunk discarding
its argument (`retn 4`) and by `sub_612D80` having no packet parameter.

Packet reads: **none**.

Verdict vs the gms_v95 reference (design §1.2): **IDENTICAL** (bare opcode, empty
body).
