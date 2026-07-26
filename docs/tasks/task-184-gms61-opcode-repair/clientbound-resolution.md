# task-184: gms_61 clientbound opcode repair — resolution

READ-ONLY IDA pass. IDB: `GMS_v61.1_U_DEVM.exe.i64` (session `965202bf`). No
template/code edits were made; this document only records findings.

## Method

`CClientSocket`'s receive path splits opcode ranges across two top-level
per-instance dispatchers in this binary:

- `CWvsContext::OnPacket(int a2, CInPacket&)` @ `0x8303eb` — handles the
  low opcode range (26–91 observed).
- `CField::OnPacket(int pExceptionObject, CInPacket&)` @ `0x4e9ea3` — a
  virtual dispatcher (overridden by e.g. `CField_Wedding::OnPacket`) that
  handles the higher range (92–272 observed), routing further to per-dialog
  `OnPacket` statics.

Each writer's dialog/pool `OnPacket` was located by name, then `xrefs_to`
was used to find its unique caller inside one of the two switches above, and
the switch was decompiled to read the verbatim `case N:` (or nested numeric
range comparison, which Hex-Rays renders as an if-chain instead of a
`switch` when the case values are sparse).

## Findings

### 1. StorageOperation — Trunk dialog result

- Client class/handler: `CTrunkDlg::OnPacket(CInPacket&)` @ `0x63bf0e`.
- Dispatch: `CField::OnPacket` @ `0x4e9ea3`, decompiled:
  ```
  case 242: /*0x4ea028*/
    CTrunkDlg::OnPacket(a3); /*0x4ea25c*/
    break;
  ```
- **TRUE gms_61 clientbound opcode: `242` = `0xF2`.**
- This is the SAME value currently in `template_gms_61_1.json` (`0xF2`).
  Confirmed correct — **no correction needed for StorageOperation itself.**
- Cross-check: every `packet-audit:fname` marker for the `StorageOperation`
  writer's sub-messages (`show.go`, `error.go`, `error_modes.go`,
  `store_retrieve_assets.go`) already cites `CTrunkDlg::OnPacket#<mode>`,
  consistent with this handler (not `CStoreBankDlg::OnPacket`, which is a
  distinct class dispatched separately at `case 240–241` in the same switch
  and is not the StorageOperation writer's target in this codebase).

### 2. RPSGame — Rock-Paper-Scissors minigame

- Client class/handler: `CRPSGameDlg::OnPacket(CInPacket&)` @ `0x607cf7`.
- Dispatch: same `CField::OnPacket` @ `0x4e9ea3`, decompiled:
  ```
  case 252: /*0x4ea028*/
    CRPSGameDlg::OnPacket(a3); /*0x4ea23e*/
    break;
  ```
- **TRUE gms_61 clientbound opcode: `252` = `0xFC`.**
- Current template value is `0xF2` (wrong — this is the corrupted copy that
  collided with StorageOperation). **Correction: `RPSGame` should be `0xFC`,
  not `0xF2`.**
- No existing gms_61 writer in `template_gms_61_1.json` currently occupies
  `0xFC` (252) — checked the full `socket.writers` list — so this
  correction introduces no new collision.

### 3. SpawnPortal / RemoveTownDoor — field portal indicator spawn/despawn

- Client class/handler: `CWvsContext::OnTownPortal(CInPacket&)` @ `0x844c71`
  (handles both the spawn-indicator and town-side removal paths internally,
  branching on whether the two map ids equal `MapId.NONE`/999999999 — this
  matches atlas's existing `SpawnPortal`/`RemoveTownDoor` split exactly).
- Dispatch: `CWvsContext::OnPacket` @ `0x8303eb`, decompiled:
  ```
  case 64: /*0x8303fd*/
    CWvsContext::OnTownPortal(this, a3); /*0x8304ca*/
    break;
  ```
- **TRUE gms_61 clientbound opcode: `64` = `0x40`.**
- This matches the current template value (`0x40`) exactly, and matches the
  copy-source gms_72 template's value. **Confirmed correct — no correction
  needed.** (Note: this is a genuinely different dispatcher path than
  StorageOperation/RPSGame — `CWvsContext::OnPacket`, not `CField::OnPacket`
  — which is why 0x40 does not collide with the CField-dispatched opcodes
  even though both switches are keyed by the same wire opcode space.)

## Summary table

| writer | current template opcode | TRUE opcode | evidence (recv case → handler) |
|---|---|---|---|
| StorageOperation | `0xF2` | `0xF2` (unchanged) | `CField::OnPacket` @0x4e9ea3: `case 242: CTrunkDlg::OnPacket(a3); /*0x4ea25c*/` → `CTrunkDlg::OnPacket` @0x63bf0e |
| RPSGame | `0xF2` (collision) | **`0xFC`** | `CField::OnPacket` @0x4e9ea3: `case 252: CRPSGameDlg::OnPacket(a3); /*0x4ea23e*/` → `CRPSGameDlg::OnPacket` @0x607cf7 |
| SpawnPortal | `0x40` | `0x40` (unchanged) | `CWvsContext::OnPacket` @0x8303eb: `case 64: CWvsContext::OnTownPortal(this, a3); /*0x8304ca*/` → `CWvsContext::OnTownPortal` @0x844c71 |
| RemoveTownDoor | `0x40` | `0x40` (unchanged) | same as SpawnPortal — `OnTownPortal` internally branches spawn vs. removal on map-id `NONE` guards, matching atlas's writer split |

## Collision check

- `0xF2` after the fix is held by StorageOperation only (RPSGame moves off
  it to `0xFC`) — collision resolved.
- `0xFC` (252) is not used by any other writer currently in
  `template_gms_61_1.json`'s `socket.writers` list (checked programmatically
  against all 153 entries).
- `0x40` continues to be legitimately shared by SpawnPortal/RemoveTownDoor
  only (both route through the same `CWvsContext::OnTownPortal` handler);
  no third writer occupies `0x40`.

Nothing here is UNRESOLVED — all four writers have direct verbatim
`case N:` evidence in the gms_61 IDB.
