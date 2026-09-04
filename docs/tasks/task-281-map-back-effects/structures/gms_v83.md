# gms_v83 — SET_BACK_EFFECT / CLEAR_BACK_EFFECT

IDB session: `754107bf` (`MapleStory_dump.exe.i64`, v83_Me)
Export: none used — live IDB. Symbols are mangled; look up
`?OnPacket@CMapLoadable@@UAEXJAAVCInPacket@@@Z`.

Step 0 (already implemented?): NO. `grep -rl BackEffect libs/atlas-packet/` returns
nothing, so this is a genuinely new codec, not a wrapper over a shared decoder.

## Router

`CMapLoadable::OnPacket` @ `0x644446`:

```c
switch ( nType )                                                /*0x64444f*/
{
  case 128: CMapLoadable::OnSetBackEffect((this - 8), iPacket);      break; /*0x64447a*/
  case 129: CMapLoadable::OnSetMapObjectVisible((this - 8), iPacket); break; /*0x64446c*/
  case 130: CMapLoadable::OnClearBackEffect((this - 8), iPacket);    break; /*0x64445e*/
}
```

- case 128 (`0x80`) -> `CMapLoadable::OnSetBackEffect` @ `0x6445c5`
- case 130 (`0x82`) -> `CMapLoadable::OnClearBackEffect` @ `0x6449ca`

Opcode cross-check: `docs/packets/registry/gms_v83.yaml:659,669` record
`SET_BACK_EFFECT opcode: 128` (`0x80`) and `CLEAR_BACK_EFFECT opcode: 130` (`0x82`).
Both MATCH the router and the task opcode table (v83 = `0x080` / `0x082`).

## SET_BACK_EFFECT read order

Decode callee: `Field::BackEffect::Decode` @ `0x587d24` (symbolized in this IDB;
called from `OnSetBackEffect` @ `0x644603`):

```c
int __thiscall Field::BackEffect::Decode(Field::BackEffect *this, struct CInPacket *a2)
{
  *(this + 4) = CInPacket::Decode1(a2);   /*0x587d38*/
  *(this + 5) = CInPacket::Decode4(a2);   /*0x587d42*/
  *(this + 6) = CInPacket::Decode1(a2);   /*0x587d4f*/
  result      = CInPacket::Decode4(a2);   /*0x587d52*/
  *(this + 7) = result;                   /*0x587d58*/
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

Branch shape, from `CMapLoadable::OnSetBackEffect` @ `0x6445c5` (this version keeps
the readable symbols, so the semantics are explicit):
- `nEffect == 0` (`v25`): arm at `0x644786` — `sub_6456D2(this->m_mlLayerBack, &v26, v32)`
  (`0x64479b`) looks the page up by `nPageID` (`v26`), the tween end time is
  `v27 + get_update_time()` (`0x6447b3`, i.e. `tDuration` + now), and each layer's
  `IWzGr2DLayer::GetAlpha` (`0x64484f`) result is tweened to **255** via the vtable
  `+144` call at `0x644889`.
- `nEffect == 1`: arm at `0x64461a`, identical shape, alpha target **0**
  (`0x64471a`).
- any other value: neither arm is entered; the handler falls to its epilogue
  (`0x6448f5`) without touching the field.

`nFieldID` (`this+5`) is decoded and never read by the handler — position-significant
only, exactly as on v95.

Verdict vs the gms_v95 reference (design §1.1): **IDENTICAL**.

## CLEAR_BACK_EFFECT

Handler @ `0x6449ca`:

```c
// attributes: thunk
void __thiscall CMapLoadable::OnClearBackEffect(CMapLoadable *this, CInPacket *a2)
{
  CMapLoadable::ReloadBack(this);   /*0x6449ca*/   // ReloadBack @ 0x644491
}
```

Packet reads: **none**. `a2` is untouched.

Verdict vs the gms_v95 reference (design §1.2): **IDENTICAL** (bare opcode, empty
body).
