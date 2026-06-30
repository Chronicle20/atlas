# task-113 Pre-flight — IDB ports, CSV columns, WZ availability

> Single source of truth for `<PORT>` (every Stage A/B/D/E task) and the Stage B / Stage G inputs across all four passes. Produced by Phase 0, Task 0.1.

## 1. IDB instance → port map (confirmed by binary name)

Enumerated via `mcp__ida-pro__list_instances`. **Ports follow launch order — confirmed by binary name, never assumed** (`reference_ida_instance_ports_shifted_idbs_v9`). All instances `reachable: true` at pre-flight.

| Version key | Binary name | Port | Role |
|---|---|---|---|
| `gms_v48` | `GMS_v48_1_DEVM.exe` | 13337 | Pass 4 target |
| `gms_v61` | `GMS_v61.1_U_DEVM.exe` | 13338 | Pass 3 target |
| `gms_v72` | `GMS_v72.1_U_DEVM.exe` | 13339 | Pass 2 target |
| `gms_v79` | `GMS_v79_1_DEVM.exe` | 13340 | Pass 1 target |
| `gms_v95` | `GMS_v95.0_U_DEVM.exe` | 13341 | Tie-breaker (all passes) |
| `gms_v83` | `MapleStory_dump.exe` | 13342 | Anchor for Pass 1 (v79) |

All four in-scope target IDBs (48/61/72/79) are loaded and reachable → **Task 0.1 STOP gate cleared**. The GMS 95 tie-breaker and the v83 anchor are also loaded.

> These ports differ from the values recorded in project memory; that is expected (launch-order dependent). Re-confirm with `list_instances` at the start of every pass before reading — do not hardcode these numbers into later sessions.

## 2. Ops CSV column inventory

Headers (`head -1`) of `docs/packets/MapleStory Ops - ClientBound.csv` and `… - ServerBound.csv`:

- **ClientBound** columns: GMS v12, **GMS v48**, **GMS v61**, **GMS v72**, **GMS v79**, GMS v83, GMS v87, GMS v92, GMS v95, GMS v111, JMS v185.
  → All four targets **have a clientbound column** → Stage B clientbound seeds directly from the CSV via `registry seed`.
- **ServerBound** columns: GMS v12, GMS v83, GMS v87, GMS v92, GMS v95, GMS v111, JMS v185.
  → **None** of the four targets has a serverbound column → Stage B serverbound must **copy the descending anchor's YAML** (`gms_v79`←`gms_v83`, `gms_v72`←`gms_v79`, `gms_v61`←`gms_v72`, `gms_v48`←`gms_v61`) and annotate `provenance` (manual entries carry an IDA citation), per Stage B step 1.

Per-version Stage B seeding consequence:

| Version | Clientbound seed | Serverbound seed |
|---|---|---|
| `gms_v79` | CSV `GMS v79` column | copy `gms_v83.yaml`, re-derive from v79 IDB, annotate provenance |
| `gms_v72` | CSV `GMS v72` column | copy `gms_v79.yaml`, re-derive from v72 IDB, annotate provenance |
| `gms_v61` | CSV `GMS v61` column | copy `gms_v72.yaml`, re-derive from v61 IDB, annotate provenance |
| `gms_v48` | CSV `GMS v48` column | copy `gms_v61.yaml`, re-derive from v48 IDB, annotate provenance |

## 3. WZ data availability (OQ-4)

Design closed OQ-4 as **WZ data available for all four versions (owner-confirmed)**; ingestion is a firm Stage G deliverable. atlas-data ingests WZ into **object storage** under `regions/GMS/versions/<major>.1/` (not committed in-repo; no `versions/` dir exists in the worktree, as expected for object-storage-backed data).

**To confirm at each pass's Stage G (not a protocol-layer blocker):** the concrete WZ source/path for that version. If a version's data turns out genuinely unobtainable at Stage G, that is the OQ-4 stop-and-ask carve-out (protocol bar still holds); otherwise ingest under `regions/GMS/versions/<major>.1/` and clear the spawn cache (`reference_atlas_maps_spawn_cache`).

**Stage I (live playthrough)** additionally needs a real client per version — a human-in-the-loop step the controller cannot perform alone; surface at Stage H/I time.
