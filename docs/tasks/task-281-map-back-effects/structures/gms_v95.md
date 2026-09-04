# gms_v95 — SET_BACK_EFFECT / CLEAR_BACK_EFFECT

IDB session: `ecc757f4` (`GMS_v95.0_U_DEVM.exe.i64`)

**Source of this record:** transcription of `docs/tasks/task-281-map-back-effects/design.md`
§1.1 / §1.2, not a fresh decompile run in task 2. The design's derivation was run
against session `ecc757f4` during phase 2 and is the reference layout every other
version in this task is compared against. Addresses below are quoted from the
design document verbatim.

Step 0 (already implemented?): NO. `grep -rl BackEffect libs/atlas-packet/`
returns nothing, so this is a genuinely new codec, not a wrapper over a shared
decoder.

## Router

`CMapLoadable::OnPacket` @ `0x61fd80`:

- case `144` (`0x90`) -> `CMapLoadable::OnSetBackEffect` @ `0x612850`
- case `145` (`0x91`) -> `CMapLoadable::OnSetMapObjectVisible`
- case `146` (`0x92`) -> `CMapLoadable::OnClearBackEffect` @ `0x61f230`

Opcode cross-check: `docs/packets/registry/gms_v95.yaml:761,771` record
`SET_BACK_EFFECT opcode: 144` and `CLEAR_BACK_EFFECT opcode: 146`. Both MATCH the
router.

## SET_BACK_EFFECT read order

Decode callee: `Field::BackEffect::Decode` @ `0x565500` (symbolized in this IDB;
called from `OnSetBackEffect` @ `0x612850`, which touches the packet nowhere
else):

```c
void Field::BackEffect::Decode(Field::BackEffect *this, CInPacket *iPacket)
{
  this->nEffect   = CInPacket::Decode1(iPacket);
  this->nFieldID  = CInPacket::Decode4(iPacket);
  this->nPageID   = CInPacket::Decode1(iPacket);
  this->tDuration = CInPacket::Decode4(iPacket);
}
```

| # | Read | Width | Field |
|---|---|---|---|
| 1 | Decode1 | byte  | nEffect |
| 2 | Decode4 | int32 | nFieldID |
| 3 | Decode1 | byte  | nPageID |
| 4 | Decode4 | int32 | tDuration |

Total 10 bytes.

Branch shape, from `CMapLoadable::OnSetBackEffect` @ `0x612850`:
- `nEffect == 0`: page lookup
  `ZMap<long, ZRef<ZList<IWzGr2DLayer>>>::GetAt(&this->m_mlLayerBack, &key, &value)`
  keyed on `nPageID`, end time `tDuration + get_update_time()`, layer walk,
  `IWzVector2D::RelMove(alpha, 255, ...)` — fade the page **in**.
- `nEffect == 1`: identical shape, `RelMove(alpha, 0, ...)` — fade it **out**.
- any other value: the handler returns without touching the field.

`nFieldID` is decoded but never read by the v95 handler — it is
position-significant only.

## CLEAR_BACK_EFFECT

Handler @ `0x61f230`:

```c
// attributes: thunk
void CMapLoadable::OnClearBackEffect(CMapLoadable *this, CInPacket *iPacket)
{
  CMapLoadable::ReloadBack(this);   // ReloadBack @ 0x61f0c0
}
```

Packet reads: **none**. `iPacket` is untouched.

`ReloadBack` @ `0x61f0c0` is the reference clear shape used to recognise this
handler on other versions: `RemoveAll` on the back-layer
`ZMap<long, ZRef<ZList<IWzGr2DLayer>>>` member -> two `IWzGr2D::Getcenter` COM
blocks (vtbl `+100` with `vtEmpty`, then vtbl `+64` with `(0,0)`) ->
`CMapLoadable::RestoreBack(this)` -> `CUser::GetVecCtrl` ->
`CAnimationDisplayer::SetCenterOrigin`. The effect is field-wide: it clears every
page's effect at once, not just one page.

This is the **reference** record. Every other version's record in this directory
states its verdict relative to it.
