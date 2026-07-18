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
| USE_TELEPORT_ROCK | SB | unnamed | unnamed | named (0x?) | **0x53** | payload has NO inline destination |
| TROCK_ADD_MAP | SB | ? | **94** | ? | unnamed | v61 payload documented |

## Central open question (drives the use flow)

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

## Out of scope

- v12 (not a packet-matrix column).
- JMS legacy (not requested).
