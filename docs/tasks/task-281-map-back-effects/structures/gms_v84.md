# gms_v84 — SET_BACK_EFFECT / CLEAR_BACK_EFFECT

IDB session: `46c2a2eb` (`GMS_v84.1_U_DEVM.i64`)
Export: none used.

Step 0 (already implemented?): NO. `grep -rl BackEffect libs/atlas-packet/` returns
nothing, so this is a genuinely new codec, not a wrapper over a shared decoder.

Source: **transcribed, not re-decompiled.** The set path is `derived in design §1.1`;
the router and clear addresses come from the task-100 cluster-E/F registry notes at
`docs/packets/registry/gms_v84.yaml:889` (SET) and `:905` (CLEAR), which record a
direct read of the v84 router. No address in this file was invented.

## Router

`CField::OnPacket` @ `0x53d5a7` forwards headers 131-133 to
`CMapLoadable::OnPacket` @ `0x659cbd` (registry note `gms_v84.yaml:889`):

- case 131 (`0x83`) -> `CMapLoadable::OnSetBackEffect` @ `0x659e3c`
- case 132 (`0x84`) -> `CMapLoadable::OnSetMapObjectVisible` @ `0x65a249`
- case 133 (`0x85`) -> `CMapLoadable::OnClearBackEffect` @ `0x65a241`
  (thunk -> `0x659d08`)

Opcode cross-check: `docs/packets/registry/gms_v84.yaml:882` records
`SET_BACK_EFFECT opcode: 131` (`0x83`), `:905` records
`CLEAR_BACK_EFFECT opcode: 133` (`0x85`). Both MATCH the task opcode table
(v84 = `0x083` / `0x085`). Note the v84 shift: these are `+3` on the v83 values
(`0x80`/`0x82`) because of the v84 CWvsContext reshift — the stale v83 carryover
slots are NOT valid on v84.

## SET_BACK_EFFECT read order

Decode callee: `sub_597B59` @ `0x597b59` (the unnamed v84 `Field::BackEffect::Decode`;
derived in design §1.1) — `Decode1`, `Decode4`, `Decode1`, `Decode4` into
`this[4..7]`.

| # | Read | Width | Field |
|---|---|---|---|
| 1 | Decode1 | byte  | nEffect |
| 2 | Decode4 | int32 | nFieldID |
| 3 | Decode1 | byte  | nPageID |
| 4 | Decode4 | int32 | tDuration |

Total 10 bytes.

Branch shape (design §1.1 and the `gms_v84.yaml:889` note): `BackEffect::Decode`
followed by two near-identical branches on the decoded state (`0`/`1`), each doing
`ZMap` `GetAt` on the layer map + `get_update_time` + a walk of the `IWzGr2DLayer`
`ZList` with `GetAlpha`/`RelMove` — alpha **255** for state 0, alpha **0** for
state 1. Any other value: no arm taken. The registry note records the body as a
match for v95 `CMapLoadable::OnSetBackEffect` @ `0x612850`.

Verdict vs the gms_v95 reference (design §1.1): **IDENTICAL**.

## CLEAR_BACK_EFFECT

Handler @ `0x65a241` — a thunk to `0x659d08` (registry note `gms_v84.yaml:905`).
Packet reads: **none** (thunk shape, `iPacket` untouched; same
`OnClearBackEffect -> ReloadBack` form proven on v72/v79/v83/v87/v92/v95).

Verdict vs the gms_v95 reference (design §1.2): **IDENTICAL** (bare opcode, empty
body).
