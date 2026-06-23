# Summon Clientbound Packet-Fixture Verification Campaign — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-23
---

## 1. Overview

The packet coverage matrix (`docs/packets/audits/STATUS.md`) tracks, per packet ×
per client version, whether the Atlas writer/handler has been proven byte-correct
against the real client read order (a `packet-audit:verify` byte-fixture + a pinned
evidence record). The `summon` family is the **lowest-coverage implemented family**
in the matrix at ~44% verified — every clientbound summon packet is verified for
`gms_v95` but `incomplete` for `gms_v83`, `gms_v84`, `gms_v87`, and `jms_v185`.

Critically, `gms_v95` is a **complete, verified reference**: the client read order
is already decompiled and pinned for all six clientbound summon packets. This task
is therefore a *port-the-verified-read-order-across-versions* campaign, not a
from-scratch reverse-engineering effort. The shape of each packet is known; only
per-version opcodes and any version-shifted field offsets differ.

This campaign also clears the **single remaining `partial` cell in the entire
matrix** (`summon/clientbound/SummonMove` on `gms_v95`).

## 2. Goals

Primary goals:
- Drive every `incomplete` cell in the `summon` family to `verified` (✅) across all
  applicable versions.
- Resolve the lone `partial` cell: `summon/clientbound/SummonMove` × `gms_v95`.
- Each promotion lands the three coupled artifacts together: the byte-fixture test
  (with a `packet-audit:verify` marker), the pinned evidence record, and the
  regenerated matrix.

Non-goals:
- No new summon *features* or gameplay behavior — verification only.
- No changes to the serverbound summon handlers (`SummonMoveHandle`,
  `SummonDamageHandle`) — already verified across all five versions.
- No opcode/registry reshifts unless a fixture proves the current registry opcode is
  wrong (in which case it is surfaced, not silently patched).

## 3. User Stories

- As an Atlas maintainer, I want every summon clientbound packet proven byte-correct
  on v83/v84/v87/jms so that summon spawn/move/attack/skill/damage render correctly
  for those tenants, not just v95.
- As a packet-audit reviewer, I want the matrix to show ✅ (not ❌/🟡) for the whole
  summon family so coverage reporting reflects reality.

## 4. Functional Requirements

Each of the following clientbound packets must reach `verified` on every applicable
version (current gaps in brackets):

| Packet | Verified today | To verify |
|---|---|---|
| `summon/clientbound/SummonSpawn` (T1) | v95 | v83, v84, v87, jms |
| `summon/clientbound/SummonRemove` (T1) | v95 | v83, v84, v87, jms |
| `summon/clientbound/SummonMove` (T1) | v95 *(partial)* | v83, v84, v87, jms, **+ fix v95 partial** |
| `summon/clientbound/SummonAttack` (T1) | v95 | v83, v84, v87, jms |
| `summon/clientbound/SummonDamage` (T1) | v95 | v83, v84, v87, jms |
| `summon/clientbound/SummonSkill` (T1) | v95 | v83, v84, v87, jms |

Already fully verified (no work): `SummonMoveHandle`, `SummonDamageHandle`
(serverbound, all five versions); `SummonMove` serverbound.

For each packet × version cell, follow `docs/packets/audits/VERIFYING_A_PACKET.md`:
1. Decompile the client read order via ida-pro-mcp at the per-version opcode (use the
   verified `gms_v95` read order as the structural template; confirm field-by-field).
2. Write/extend the byte-fixture test with a `packet-audit:verify` marker.
3. Pin the evidence record.
4. Regenerate the matrix (`packet-audit matrix`); confirm the cell promotes to ✅.
5. Commit the three artifacts together.

## 5. API Surface

None. No REST or Kafka surface changes. This is wire-format verification of existing
clientbound writers.

## 6. Data Model

None. No schema or migration changes.

## 7. Service Impact

- **atlas-channel** (or wherever summon clientbound writers live — confirm in design):
  test files and fixtures added; production writer code changed only if a fixture
  proves a current writer is byte-incorrect for a version.
- **docs/packets/** — evidence records and regenerated `STATUS.md` / `status.json`.

## 8. Non-Functional Requirements

- Verification is byte-level — no "enumeration = ✅" shortcuts (see project memory:
  dispatcher mode-byte enumeration is a false pass). Each fixture exercises the full
  packet body.
- IDA lookups use `func_query`/the documented MCP API; confirm the instance/version
  (v83=13341, v84=13337, v87=13340, v95=13339, jms=13338) matches the cell under test.
- Multi-tenancy: unaffected (wire format is tenant-config-driven; no tenant data).

## 9. Open Questions

- Which Atlas service/package owns the summon clientbound writers? (Confirm in design.)
- Does any version's summon opcode differ from the registry value, or are all gaps
  purely missing fixtures? (Expected: missing fixtures only, since v95 is verified.)
- `SummonMove` v95 `partial` — what exactly is unverified (a trailing field / movement
  fragment)? Diagnose before porting to other versions.

## 10. Acceptance Criteria

- [ ] All six summon clientbound packets show `verified` (✅) for v83, v84, v87, v95, jms
      in the regenerated matrix (or `n-a` where the packet is genuinely version-absent).
- [ ] No `partial` (🟡) cell remains anywhere in the `summon` family.
- [ ] Every promoted cell has a `packet-audit:verify` byte-fixture and a fresh pinned
      evidence record committed together.
- [ ] `packet-audit matrix --check` (and any fname-doc/operations `--check`) exit 0.
- [ ] Affected Go module: `go test -race ./...`, `go vet ./...`, `go build ./...` clean;
      `docker buildx bake` for any service whose `go.mod` was touched.
