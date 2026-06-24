# Login Packet-Fixture Verification Campaign — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-23
---

## 1. Overview

The `login` family in the packet coverage matrix
(`docs/packets/audits/STATUS.md`) sits at ~74% verified with **20 `incomplete`
cells** spread across both clientbound and serverbound login-flow packets. Unlike
summon/door, the gaps are **scattered single- or double-version holes** rather than a
clean "one version verified, port to the rest" shape — most packets are verified on
the majority of versions with one or two version-specific holes (commonly `gms_v84`
and `jms_v185`).

This campaign closes those holes so the entire login/world-select/character-select
handshake is byte-verified across all applicable versions. Because each hole usually
has at least one verified sibling version, most cells can port a known read order;
a few may need fresh per-version decompilation.

## 2. Goals

Primary goals:
- Drive every `incomplete` cell in the `login` family to `verified` (✅) across all
  applicable versions.
- Land each promotion as the three coupled artifacts: byte-fixture (with
  `packet-audit:verify` marker), pinned evidence, regenerated matrix.

Non-goals:
- No new login/auth/world-select features — verification only.
- No PIN/SPW/auth-flow behavior changes.
- No opcode reshifts unless a fixture proves the registry opcode wrong (then surface,
  don't silently patch).

## 3. User Stories

- As an Atlas maintainer, I want the login → world-list → character-select handshake
  proven byte-correct on every version so login works identically across tenants.
- As a packet-audit reviewer, I want the login family's scattered ❌ holes closed to ✅.

## 4. Functional Requirements

Close the following `incomplete` cells (current gaps in brackets; `n-a` noted where the
packet is version-absent):

Clientbound:
- `login/clientbound/AuthLoginFailed` — [v83, v84]
- `login/clientbound/ServerStatus` — [v84] (jms n-a)
- `login/clientbound/ServerListEnd` — [jms]

Serverbound:
- `login/serverbound/ServerStatusRequest` — [jms]
- `login/serverbound/AllCharacterListRequest` (T1) — [v83]
- `login/serverbound/AllCharacterListSelect` — [v84, v87] (jms n-a) — appears as
  multiple matrix rows; verify each
- `login/serverbound/CharacterSelect` — [v84, jms] — appears as multiple rows; verify each
- `login/serverbound/ServerListRequest` — [v83, v84]

(The exact row multiplicity is in `status.json`; design should enumerate each row by
fname so none is missed — multiple `CharacterSelect`/`AllCharacterListSelect` rows
exist with the same packet path but distinct fnames.)

For each cell, follow `docs/packets/audits/VERIFYING_A_PACKET.md`:
1. Decompile the client read order (clientbound) or the client write order
   (serverbound, per `VERIFYING_A_PACKET.md` §9–10 serverbound rules) at the
   per-version opcode, using a verified sibling version as the structural template
   where one exists.
2. Write/extend the byte-fixture test with a `packet-audit:verify` marker.
3. Pin the evidence record (serverbound needs marker + evidence + REPORT).
4. Regenerate the matrix and confirm the cell promotes to ✅.
5. Commit the artifacts together.

## 5. API Surface

None. Wire-format verification of existing login handlers/writers.

## 6. Data Model

None.

## 7. Service Impact

- **atlas-login** (and any shared character-list codec) — test files and fixtures
  added; production code changed only if a fixture proves a writer/handler
  byte-incorrect for a version.
- **docs/packets/** — evidence records and regenerated `STATUS.md` / `status.json`.

## 8. Non-Functional Requirements

- Byte-level verification — no enumeration shortcuts.
- Serverbound cells follow the serverbound verification rules (shared-model ops use
  thin wrapper codecs; distrust IDB names — the COutPacket opcode is truth; export is
  non-idempotent surgical splice).
- IDA lookups via the documented MCP API; confirm instance/version per cell
  (v83=13341, v84=13337, v87=13340, v95=13339, jms=13338).

## 9. Open Questions

- The duplicated `CharacterSelect` / `AllCharacterListSelect` rows: are these distinct
  fnames sharing a packet path (each needing its own fixture), or matrix artifacts?
  Enumerate by fname in design before counting "done".
- `AllCharacterListRequest` v83 hole (T1) and `ServerListRequest` v83/v84 holes —
  serverbound; confirm a verified sibling exists to port from, else fresh decompile.
- Any cell that is genuinely `n-a` for jms (login opcode space differs) vs a real gap.

## 10. Acceptance Criteria

- [ ] Every `login`-family row shows `verified` (✅) or `n-a` for all five versions in
      the regenerated matrix — no `incomplete` remains.
- [ ] Each duplicated-path row is verified per distinct fname.
- [ ] Every promoted cell has a `packet-audit:verify` byte-fixture and a fresh pinned
      evidence record (serverbound: + REPORT) committed together.
- [ ] `packet-audit matrix --check` (and fname-doc/operations `--check`) exit 0.
- [ ] Affected Go module(s): `go test -race ./...`, `go vet ./...`, `go build ./...`
      clean; `docker buildx bake` for any service whose `go.mod` was touched.
