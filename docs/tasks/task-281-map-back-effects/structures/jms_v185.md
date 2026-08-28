# jms_v185 — SET_BACK_EFFECT / CLEAR_BACK_EFFECT

IDB session: `a977912e` (`MapleStory_dump_SCY.exe.i64`)
Export: none used — live IDB. Symbols are mangled and fully resolved; look up
`?OnPacket@CMapLoadable@@UAEXJAAVCInPacket@@@Z`.

Step 0 (already implemented?): NO. Same as gms_v87 — `grep -rl BackEffect
libs/atlas-packet/` returns nothing, so this is a genuinely new codec, not a
wrapper over a shared decoder.

## Router

`CMapLoadable::OnPacket` @ `0x6ba102`:

```c
switch ( nType )                                                     /*0x6ba109*/
{
  case 0x7E: CMapLoadable::OnSetBackEffect((this - 8), iPacket); break;       /*0x6ba134*/
  case 0x7F: CMapLoadable::OnSetMapObjectVisible((this - 8), iPacket); break; /*0x6ba126*/
  case 0x80: CMapLoadable::OnClearBackEffect((this - 8), iPacket); break;     /*0x6ba118*/
}
```

- case `0x7E` (126) -> `CMapLoadable::OnSetBackEffect` @ `0x6ba27f`
- case `0x80` (128) -> `CMapLoadable::OnClearBackEffect` @ `0x6ba684`

Opcode cross-check: `docs/packets/registry/jms_v185.yaml:623,633` record
`SET_BACK_EFFECT opcode: 126` (`0x7E`) and `CLEAR_BACK_EFFECT opcode: 128`
(`0x80`). Both MATCH the router and the task opcode table (jms_185 = `0x07E` /
`0x080`).

## SET_BACK_EFFECT read order

Decode callee: `Field::BackEffect::Decode` @ `0x5dcc48` (symbolized in this IDB;
called from `OnSetBackEffect` at `0x6ba2bd`):

```c
void __thiscall Field::BackEffect::Decode(_DWORD *this, CInPacket *a2)
{
  this[4] = CInPacket::Decode1(a2);   /*0x5dcc5c*/
  this[5] = CInPacket::Decode4(a2);   /*0x5dcc66*/
  this[6] = CInPacket::Decode1(a2);   /*0x5dcc73*/
  this[7] = CInPacket::Decode4(a2);   /*0x5dcc7c*/
}
```

| # | Read | Width | Field |
|---|---|---|---|
| 1 | Decode1 | byte  | nEffect |
| 2 | Decode4 | int32 | nFieldID |
| 3 | Decode1 | byte  | nPageID |
| 4 | Decode4 | int32 | tDuration |

Total 10 bytes.

Branch shape, from `CMapLoadable::OnSetBackEffect` @ `0x6ba27f`: the handler
tests the decoded `nEffect` slot (`v23`, struct `+0x10`).
- `nEffect == 0`: page lookup `sub_6BB4DA(this->m_mlLayerBack, &v24, v30)`
  (`0x6ba455`) keyed on `nPageID` (`v24`), end time `v25 + get_update_time()`
  (`0x6ba46d`, i.e. `tDuration` + now), layer walk, vtable `+144` tween
  (`v18->lpVtbl[9]`, `0x6ba543`) with alpha target **255**.
- `nEffect == 1`: identical shape (`0x6ba2e9` / `0x6ba301` / `0x6ba3d4`), alpha
  target **0**.
- any other value: neither arm is entered; the handler falls to its epilogue
  without touching the field.

Verdict vs the gms_v95 reference (design §1.1): **IDENTICAL**. No JMS read-order
divergence — same four reads, same order, same widths, same two-value enum.

## CLEAR_BACK_EFFECT

Handler @ `0x6ba684`:

```c
// attributes: thunk
void __thiscall CMapLoadable::OnClearBackEffect(CMapLoadable *this, CInPacket *iPacket)
{
  CMapLoadable::ReloadBack(this);   /*0x6ba684*/   // ReloadBack @ 0x6ba14b
}
```

Packet reads: **none**. `iPacket` is untouched.

Verdict vs the gms_v95 reference (design §1.2): **IDENTICAL** (bare opcode, empty
body).
