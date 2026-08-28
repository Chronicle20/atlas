# gms_v72 — SET_BACK_EFFECT / CLEAR_BACK_EFFECT

IDB session: `99e435d8` (`GMS_v72.1_U_DEVM.exe.i64`)
Export: none used — live IDB.

Step 0 (already implemented?): NO. `grep -rl BackEffect libs/atlas-packet/` returns
nothing, so this is a genuinely new codec, not a wrapper over a shared decoder.

Source: the `OnSetBackEffect` / decode facts are transcribed from **design §1.1**
(`docs/tasks/task-281-map-back-effects/design.md`), which derived them on this same
session. The router and the clear arm below were read directly from the IDB during
this task (the design did not cover them for v72).

## Router

`CMapLoadable::OnPacket` @ `0x5f59e3` — an if/else chain, not a switch:

```c
void __thiscall CMapLoadable::OnPacket(CMapLoadable *this, int a2, struct CInPacket *a3)
{
  if ( a2 == 117 )                                                    /*0x5f59ea*/
    CMapLoadable::OnSetBackEffect((CMapLoadable *)((char *)this - 8), a3); /*0x5f5a04*/
  else if ( a2 == 118 )                                               /*0x5f59ed*/
    sub_5F5F54(a3);                                                   /*0x5f59f6*/
}
```

- case 117 (`0x75`) -> `CMapLoadable::OnSetBackEffect` @ `0x5f5b4f`
- case 118 (`0x76`) -> `sub_5F5F54` @ `0x5f5f54` (unnamed; the clear arm — see below)

Opcode cross-check: `docs/packets/registry/gms_v72.yaml:838` records
`SET_BACK_EFFECT opcode: 117` (`0x75`). MATCHES the router and the task opcode
table (v72 = `0x075`).

## SET_BACK_EFFECT read order

Decode callee: `sub_54C265` @ `0x54c265` (derived in design §1.1 — `Decode1`,
`Decode4`, `Decode1`, `Decode4` into `this[4..7]`).

| # | Read | Width | Field |
|---|---|---|---|
| 1 | Decode1 | byte  | nEffect |
| 2 | Decode4 | int32 | nFieldID |
| 3 | Decode1 | byte  | nPageID |
| 4 | Decode4 | int32 | tDuration |

Total 10 bytes.

Branch shape (design §1.1): `nEffect==0` -> `RelMove(alpha, 255, ...)`;
`nEffect==1` -> `RelMove(alpha, 0, ...)`; any other value -> the handler returns
without touching the field.

Verdict vs the gms_v95 reference (design §1.1): **IDENTICAL**.

## CLEAR_BACK_EFFECT

Handler @ `0x5f5f54` (`sub_5F5F54`, unnamed in this IDB):

```c
// attributes: thunk
int __stdcall sub_5F5F54(int a1)
{
  return sub_5F5A1B();   /*0x5f5f59*/
}
```

Packet reads: **none**. The thunk discards its `CInPacket *` argument and tail-calls
`sub_5F5A1B` @ `0x5f5a1b` with no arguments.

`sub_5F5A1B` is unnamed in this IDB. It sits immediately after
`CMapLoadable::OnPacket` (`0x5f59e3` + `0x29` = `0x5f5a0c`), the exact position that
`?ReloadBack@CMapLoadable@@QAEXXZ` occupies in gms_v79 (`OnPacket` @ `0x614406`,
`ReloadBack` @ `0x61443e`). Treat "this is `ReloadBack`" as a strong positional
inference, NOT a proven name; the load-bearing fact for the codec — that the body is
empty — is proven by the thunk itself.

Verdict vs the gms_v95 reference (design §1.2): **IDENTICAL** (bare opcode, empty
body).

FINDING: `docs/packets/registry/gms_v72.yaml` carries **no** `CLEAR_BACK_EFFECT`
entry, but the client router does have a clear arm at opcode 118 (`0x76`). The op is
PRESENT on v72; the registry gap is a registry gap, not an absence. Task 2 owns what
to do about it.
