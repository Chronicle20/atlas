# Design — Legacy teleport-rock support (GMS v48/v61/v72/v79)

Date: 2026-07-18
Status: RE in progress (v79 characterized) — seeds the implementation plan

## Why this exists

task-124 scoped teleport rocks to v83/v84/v87/v92/v95/jms_v185 and left
v48/v61/v72/v79 as `n-a`/`incomplete` in the coverage matrix. **That was wrong.**
The legacy IDBs contain `CWvsContext::SendMapTransferItemUseRequest` and
`CWvsContext::OnMapTransferResult`, and v79's decompiled `OnMapTransferResult`
reads a full 5-regular / 10-VIP saved-map list — so these clients have the
**complete** teleport-rock feature (use + saved-map list), not a missing or
degraded one. The serverbound packets diverge from v83, so they need
per-version reverse-engineered codecs. (v12 is NOT a packet-matrix column — out
of scope here.)

## Confirmed RE

### v79 (port 13340, GMS_v79_1_DEVM.exe) — fully characterized

**USE_TELEPORT_ROCK — `CWvsContext::SendMapTransferItemUseRequest` @0x968c52**
(sig `(unsigned __int16 nPOS, unsigned int nItemID)`), called from
`CDraggableItem::OnDoubleClicked` @0x4e3663 (double-click the rock in inventory):
```
COutPacket(0x53)                 // opcode 83
Encode4(updateTime)              // leading (get_update_time())
Encode2(nPOS)                    // inventory slot
Encode4(nItemID)                 // item id
```
**No inline destination** — v83 appends the `RunMapTransferItem` target
(byName flag + name/mapId); v79 does not. So the pre-v83 use packet is a bare
"I used rock X in slot Y."

**MAP_TRANSFER_RESULT — `CWvsContext::OnMapTransferResult` @0x96f362** (opcode 39):
```
mode = Decode1
flag = Decode1                   // vip flag
switch(mode):
  case 2/3 (DELETE_LIST/REGISTER_LIST):
      n = flag ? 10 : 5          // VIP=10, regular=5
      repeat n: Decode4(mapId)   // full saved-map list, into ctx list buffer
                                 //   (+1303 regular / +1323 VIP dwords)
  case 5,6,7,8,9,10,11:          // notice/error modes (StringPool 2963/2928/2934/2931/2935)
```
Structure ≈ v83 (mode byte + flag + list refresh + error modes) — the existing
clientbound codec is reusable; only the opcode differs.

**TROCK_ADD_MAP — `SendMapTransferRequest`**: the list is registerable (result
mode 3 exists), so the sender exists but is **unnamed** in v79. Opcode + payload
still to be located (RE TODO).

### v61 (from the checked-in registry, already RE'd in a prior pass)

- **TROCK_ADD_MAP** op **94**: `COutPacket(94) + Encode1(mode) + Encode1(mapIndex) + [mode==0: Encode4(mapId)]` (send-site `sub_8478EA`; the saved-map-list dialog `sub_6CA4C9`/`sub_6CA6D8`). So v61 has the list-management send path documented.
- **MAP_TRANSFER_RESULT** op 39.

### Opcode/state summary (from registry + exports + v79 RE)

| Op | dir | v48 | v61 | v72 | v79 | note |
|---|---|---|---|---|---|---|
| MAP_TRANSFER_RESULT | CB | 35 | 39 | 39 | 39 | `OnMapTransferResult` named in all; structure ≈ v83 |
| USE_TELEPORT_ROCK | SB | unnamed | unnamed | **0x54** | **0x53** | uniform pre-v83 payload; opcode per version |
| TROCK_ADD_MAP | SB | ? | **94** | ? | unnamed | v61 payload documented |

**v72 confirmed** (`SendMapTransferItemUseRequest` @0x917221, byte-identical
function size to v79): `COutPacket(0x54) + Encode4(updateTime) + Encode2(nPOS) +
Encode4(nItemID)` — identical layout to v79, only the opcode differs (0x54 vs
0x53). **The pre-v83 USE payload is uniform across legacy versions; only the
opcode changes.** This means one pre-v83 `USE_TELEPORT_ROCK` codec, opcode-gated
per version — not four different codecs. The same uniformity very likely holds
for the RESULT (structure already ≈ v83) and ADD_MAP (v61 documented) sides,
pending confirmation of the unnamed functions in v48/v61.

### v61 (port 13338) — confirmed

`OnMapTransferResult` @0x846a0f (named, opcode 39). `SendMapTransferItemUseRequest`
(USE) is **unnamed** (needs locating). ADD_MAP sender `sub_8478EA` decompiled:
```
COutPacket(94)
Encode1(nType)
Encode1(flag)
if (nType == 0) Encode4(mapId)     // mapId only on delete (nType 0); register (nType 1) sends none
```
**This is structurally identical to v83's `SendMapTransferRequest`** (Encode1 nType
+ Encode1 flag + conditional mapId) — only the opcode differs (94 vs v83's 102).

### Revised conclusion — 2 of 3 ops are reusable

- **MAP_TRANSFER_RESULT** (CB): structure ≈ v83 across legacy → existing codec, per-version opcode.
- **TROCK_ADD_MAP** (SB): structure = v83 (v61 confirmed) → existing codec, per-version opcode.
- **USE_TELEPORT_ROCK** (SB): the ONLY divergence — bare `updateTime+nPOS+nItemID`, no
  inline destination (v83 appends `RunMapTransferItem`). Needs a pre-v83 codec +
  a resolved use→warp flow.

## v61 saved-map dialog (RE'd) — list management resolved, warp-trigger not yet

The dialog is a `CDialog`-derived class (vtable @0x8ecdc8); button dispatch
`sub_6CA1CA(this, buttonId)`:
- button **2000** → `sub_6CA4C9` = **register current map** (`sub_8478EA(1,0,vip)`; full-list/dup/continent checks + YesNo).
- button **2001** → `sub_6CA6D8` = **delete selected map** (`sub_8478EA(0, selectedMapId, vip)`).
- buttons **1 / 2** → a vtable-dispatched action (the GO / close pair — the warp trigger, not yet cleanly resolved).
- The dialog create (`sub_6C974C`) builds buttons 2000/2001/1/2 + a scrollbar + a
  `CCtrlEdit` (name input, for warp-by-player as in v83), and copies the saved-map
  list from **CharacterData** (`+1187` regular / `+1207` VIP, 5/10 entries; empty
  slot = 999999999). So the list is client-resident character state.

Confirmed: register/delete ride `TROCK_ADD_MAP` (nType 1/0) exactly like v83. The
warp itself (select saved map or type a name → GO) is the one op still to pin
(dialog buttons 1/2 → vtable method). It is NOT `USE_TELEPORT_ROCK` (that packet
is bare and is sent from `CDraggableItem::OnDoubleClicked` to *open* the flow),
and NOT `TROCK_ADD_MAP` (register/delete only).

## RESOLVED — legacy USE is NOT bare; the feature = v83 with different opcodes

The v61 real USE sender is `sub_8327DB` (regular rock, `a3/10000 == 232`):
```
COutPacket(77)                      // opcode 0x4D
Encode2(nPOS)
Encode4(nItemID)
if (sub_8328C9(pkt, 0))             // RunMapTransferItem: opens the modal dialog,
                                    //   appends the target — Encode1(1)+EncodeStr(name)
                                    //   for warp-by-player, else Encode1(0)+Encode4(mapId)
                                    //   for a saved map; returns 0 if cancelled → not sent
    Encode4(updateTime)             // trailing
SendPacket()
```
This is **byte-for-byte the v83 `USE_TELEPORT_ROCK` layout** (`Decode2 + Decode4 +
RunMapTransferItem target + trailing updateTime`) — only the opcode differs (77 vs
v83's 0x54). `sub_8328C9` is the legacy `RunMapTransferItem`; the map-transfer
dialog (register 2000 / delete 2001 / GO / name-input) is modal and returns the
target to the USE sender. Cash rocks ride `SendConsumeCashItemUseRequest`
(@0x832a5d in v61) which also calls `RunMapTransferItem` — same as v83.

**Consequence:** the entire teleport-rock feature (USE-with-target, ADD_MAP
register/delete, MAP_TRANSFER_RESULT, RunMapTransferItem target payload) is
**structurally identical across v48/v61/v72/v79 and v83+**. The existing atlas
codecs apply unchanged — legacy support is **per-version opcodes + template wiring
+ byte-fixture verification**, NOT new codecs. The atlas-ui saved-map card also
applies. Much smaller than the "divergent pre-v83 codec" fear.

**v61 opcodes:** USE=77 (0x4D), TROCK_ADD_MAP=94 (0x5E), MAP_TRANSFER_RESULT=39.

**v79 cross-check (confirms the resolution):** v79's `SendConsumeCashItemUseRequest`
@0x95634a calls `CDialog::DoModal` + `EncodeStr` + `Encode1`/`Encode4` inline — it
opens the map-transfer dialog and encodes the byName/byMap target, exactly like
v61/v83. So the cash rock carries the target on v79 too. The teleport-rock feature
is confirmed v83-structured on a second legacy version.

**Discrepancy to settle during per-cell verification:** v72/v79 have a *named*
`SendMapTransferItemUseRequest` (0x917221/0x968c52) that looked bare
(Encode4+Encode2+Encode4, op 0x54/0x53) with NO RunMapTransferItem call — yet v61's
real sender does call it. Likely the named v72/v79 symbol is an older/other variant
and the real RunMapTransferItem-based USE sender in v72/v79 is unnamed (as v61's
was). Resolve per version when pinning opcodes.

## (superseded) Central open question — narrowed

v83's use packet carries the destination; the pre-v83 use packet does not. So in
legacy the flow must be: double-click rock → `SendMapTransferItemUseRequest`
(bare) → server opens the map-transfer UI (a `MAP_TRANSFER_RESULT`?) → user picks
→ destination sent via `SendMapTransferRequest`. The exact destination/warp path
(which op carries the chosen map, and how warp is triggered) must be nailed in RE
before the serverbound handlers can be written. This differs from the atlas
implementation's current model (use packet carries the target).

## Approach

- **Clientbound `MAP_TRANSFER_RESULT`**: reuse the existing codec; add per-version
  opcode routing (v48=35, v61/72/79=39) — structure matches v83, so likely a
  verify+wire, not a new codec.
- **Serverbound `USE_TELEPORT_ROCK` / `TROCK_ADD_MAP`**: per-version codecs,
  version-gated with the `MajorAtLeast` idiom (the pre-v83 layouts differ from the
  v83+ layouts already implemented). The channel-side use-flow/handlers must
  branch on the pre-v83 vs v83+ shape.
- **atlas-ui saved-map-list card**: applies unchanged to legacy tenants (the 5/10
  list exists); only the packet layer differs.
- **Templates**: wire USE/ADD_MAP/RESULT into `template_gms_{48,61,72,79}_1.json`.
- **Verification**: byte-fixture per op × version per `VERIFYING_A_PACKET.md`;
  promote the matrix cells.
- **Matrix**: the current `n-a` for USE_TELEPORT_ROCK on v48–v79 is wrong
  (function exists) — correct it as cells are verified.

## Remaining RE (Phase 0 of implementation)

Per version v48/v61/v72/v79 (name functions in-IDB as they're found):
1. Locate + name the unnamed `SendMapTransferItemUseRequest` (v48/v61) — opcode + payload.
2. Locate + name `SendMapTransferRequest` (TROCK_ADD_MAP) in v48/v72/v79 — opcode + payload (v61 done).
3. Confirm `OnMapTransferResult` opcode + read order per version (v79 done; v48/61/72 spot-check).
4. Resolve the use→destination→warp flow (the central open question).

## Implementation path (bounded — reuses existing codecs)

Because the feature is v83-structured, no new codecs are needed. Per legacy
version (v48/v61/v72/v79), via the packet-audit per-cell workflow
(`/verify-packet` + `packet-verifier`, `VERIFYING_A_PACKET.md`):

1. **Pin the real per-version opcodes** for `USE_TELEPORT_ROCK` /
   `TROCK_ADD_MAP` (naming the real senders in-IDB; the v72/v79 named
   `SendMapTransferItemUseRequest` is a mislabeled/older symbol — the real sender
   calls `RunMapTransferItem`, as v61's `sub_8327DB` and the cash path do). RESULT
   opcodes already known (v48=35, v61/72/79=39). v61 confirmed: USE=77, ADD_MAP=94.
2. **Route** USE/ADD_MAP/RESULT into `template_gms_{48,61,72,79}_1.json` with those
   opcodes + the operations table (same keys as v83+).
3. **Byte-fixture verify** each op×version, pin evidence, regenerate the matrix,
   and **correct the wrong `n-a` cells** (USE/ADD_MAP exist on these versions).
4. atlas-ui saved-map card + the atlas-channel handlers/use-flow already apply
   (structure = v83); only the version-gated opcode routing is new.

Do NOT re-derive layouts — they equal v83; the work is opcode confirmation +
wiring + per-cell byte fixtures.

## Out of scope

- v12 (not a packet-matrix column).
- JMS legacy (not requested).
