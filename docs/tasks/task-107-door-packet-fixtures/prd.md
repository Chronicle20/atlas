# Door Clientbound Packet-Fixture Verification Campaign — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-23
---

## 1. Overview

The `door` family in the packet coverage matrix
(`docs/packets/audits/STATUS.md`) is the **lowest-percentage implemented family** at
~20% verified. All three clientbound door packets are verified for `gms_v83` only and
`incomplete` for `gms_v84`, `gms_v87`, `gms_v95`, and `jms_v185`.

`gms_v83` is a **complete, verified reference** for all three packets, so this is a
*port-the-verified-read-order-across-versions* campaign: the wire shapes are known
and pinned for v83; only per-version opcodes and any version-shifted offsets differ.
It is the smallest of the family campaigns (3 packets × 4 versions ≈ 12 fixtures),
making it a good fast-moving unit.

## 2. Goals

Primary goals:
- Drive every `incomplete` cell in the `door` family to `verified` (✅) across all
  applicable versions.
- Land each promotion as the three coupled artifacts: byte-fixture (with
  `packet-audit:verify` marker), pinned evidence, regenerated matrix.

Non-goals:
- No new door/town-portal features — verification only.
- No opcode/registry reshifts unless a fixture proves the registry opcode wrong (then
  surface, don't silently patch).

## 3. User Stories

- As an Atlas maintainer, I want mystic-door spawn/remove proven byte-correct on
  v84/v87/v95/jms so doors render and despawn correctly for those tenants.
- As a packet-audit reviewer, I want the door family to read ✅ instead of mostly ❌.

## 4. Functional Requirements

Each clientbound packet must reach `verified` on every applicable version:

| Packet | Verified today | To verify |
|---|---|---|
| `door/clientbound/SpawnDoor` (T1) | v83 | v84, v87, v95, jms |
| `door/clientbound/RemoveDoor` | v83 | v84, v87, v95, jms |
| `door/clientbound/RemoveTownDoor` | v83 | v84, v87, v95, jms |

For each cell, follow `docs/packets/audits/VERIFYING_A_PACKET.md`:
1. Decompile the client read order via ida-pro-mcp at the per-version opcode, using the
   verified `gms_v83` read order as the structural template.
2. Write/extend the byte-fixture test with a `packet-audit:verify` marker.
3. Pin the evidence record.
4. Regenerate the matrix and confirm the cell promotes to ✅.
5. Commit the three artifacts together.

## 5. API Surface

None. Wire-format verification of existing clientbound writers only.

## 6. Data Model

None.

## 7. Service Impact

- **atlas-channel** (or wherever door clientbound writers live — confirm in design):
  test files and fixtures added; production code changed only if a fixture proves a
  writer byte-incorrect for a version.
- **docs/packets/** — evidence records and regenerated `STATUS.md` / `status.json`.

## 8. Non-Functional Requirements

- Byte-level verification — no enumeration shortcuts.
- IDA lookups via the documented MCP API; confirm instance/version per cell
  (v83=13341, v84=13337, v87=13340, v95=13339, jms=13338).
- Multi-tenancy unaffected.

## 9. Open Questions

- Which Atlas service/package owns the door clientbound writers? (Confirm in design.)
- Is `RemoveTownDoor`/`RemoveDoor` distinction (town-portal vs party mystic door) the
  same opcode with a flag, or two opcodes, on each version? Confirm against IDA.
- Any version where a door packet is genuinely absent (→ `n-a`, not a gap)?

## 10. Acceptance Criteria

- [ ] All three door clientbound packets show `verified` (✅) for v83, v84, v87, v95,
      jms (or `n-a` where genuinely version-absent).
- [ ] Every promoted cell has a `packet-audit:verify` byte-fixture and a fresh pinned
      evidence record committed together.
- [ ] `packet-audit matrix --check` (and fname-doc/operations `--check`) exit 0.
- [ ] Affected Go module: `go test -race ./...`, `go vet ./...`, `go build ./...` clean;
      `docker buildx bake` for any service whose `go.mod` was touched.
