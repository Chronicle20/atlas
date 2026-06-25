# Character Packet-Fixture Verification Campaign — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-23
---

## 1. Overview

The `character` family is the **largest implemented family** in the packet coverage
matrix (`docs/packets/audits/STATUS.md`): 71 implemented packet rows, ~83% verified,
with **47 `incomplete` cells** — the biggest absolute gap of any family. Unlike
summon/door (one verified reference version, port to the rest), the character gaps are
**mixed**: some are single-version holes that port from a verified sibling, but
several packets are `incomplete` on *every* version and need fresh per-version
decompilation from the client read order.

This is a long-tail campaign covering the character lifecycle (view-all/list/create/
delete), spawn/movement/appearance, buffs, key-map, and quest-effect packets. Because
it is large, the design phase should sequence the work — tier-1 packets and the
fully-unverified packets first, then the single-version holes.

## 2. Goals

Primary goals:
- Drive every `incomplete` cell in the `character` family to `verified` (✅) across all
  applicable versions.
- Land each promotion as the three coupled artifacts: byte-fixture (with
  `packet-audit:verify` marker), pinned evidence, regenerated matrix.

Non-goals:
- No new character features or gameplay behavior — verification only.
- No changes to already-verified character packets.
- No opcode reshifts unless a fixture proves the registry opcode wrong (then surface,
  don't silently patch).

## 3. User Stories

- As an Atlas maintainer, I want the full character lifecycle and in-field character
  packets proven byte-correct on every version so character select/create, spawn,
  movement, appearance, buffs, and key-map behave identically across tenants.
- As a packet-audit reviewer, I want the largest family's 47-cell gap closed to ✅.

## 4. Functional Requirements

Close every `character`-family `incomplete` cell. Grouped by remediation shape:

**A. Fully unverified — need fresh per-version decompilation (no sibling reference):**
- `character/clientbound/CharacterList` (T1) — ❌ all five versions
- `character/clientbound/CharacterAppearanceUpdate` (T1) — ❌ all five versions
- `character/clientbound/EffectQuest` (T1) — ❌ all five versions (two rows; verify each)
- `character/serverbound/KeyMapChange` (T1) — ❌ v83, v87, v95, jms (v84 verified — port)

**B. Single-/double-version holes — port from a verified sibling:**
- `character/clientbound/CharacterViewAllCharacters` (T1) — [v84, jms]
- `character/clientbound/AddCharacterEntry` (T1) — [jms]
- `character/clientbound/BuffGive` (T1) — [jms]
- `character/clientbound/CharacterInfo` (T1) — [jms]
- `character/clientbound/BuffGiveForeign` (T1) — [jms]
- `character/clientbound/CharacterSpawn` (T1) — [jms]
- `character/clientbound/CharacterMovement` (T1) — [v84, jms]
- `character/clientbound/CharacterExpression` (T1) — [v83, v84]
- `character/clientbound/CharacterChairShow` (T1) — [v83, v84, jms]
- `character/serverbound/CheckName` (T1) — [v83, v84, jms]
- `character/serverbound/CreateCharacter` (T1) — [jms]
- `character/serverbound/DeleteCharacter` (T1) — [jms]
- `character/serverbound/AutoDistributeAp` (T1) — [v84] (multiple rows; verify each)
- `character/serverbound/HealOverTime` (T1) — [jms]
- `character/serverbound/KeyMapChange` (T1) — [v84]

(Exact row multiplicity — duplicated packet paths with distinct fnames, e.g.
`AutoDistributeAp`, `EffectQuest` — is in `status.json`; design must enumerate each row
by fname so none is missed.)

For each cell, follow `docs/packets/audits/VERIFYING_A_PACKET.md`: decompile the client
read order (clientbound) or write order (serverbound, §9–10) at the per-version
opcode → byte-fixture with `packet-audit:verify` marker → pinned evidence (+ REPORT
for serverbound) → regenerate matrix → commit artifacts together.

## 5. API Surface

None. Wire-format verification of existing character handlers/writers.

## 6. Data Model

None.

## 7. Service Impact

- **atlas-login / atlas-channel** (character lifecycle vs in-field packets live in
  different services — confirm split in design) — test files and fixtures added;
  production code changed only if a fixture proves a writer/handler byte-incorrect.
- **docs/packets/** — evidence records and regenerated `STATUS.md` / `status.json`.

## 8. Non-Functional Requirements

- Byte-level verification — no enumeration shortcuts. Large packets (CharacterList,
  CharacterAppearanceUpdate, CharacterInfo) exercise the full body including nested
  avatar/look blocks.
- Serverbound cells follow the serverbound verification rules.
- IDA lookups via the documented MCP API; confirm instance/version per cell
  (v83=13341, v84=13337, v87=13340, v95=13339, jms=13338).

## 9. Open Questions

- Sequencing: the design should phase this (fully-unverified + T1 first). Is a single
  task the right unit, or should the fully-unverified group (CharacterList /
  AppearanceUpdate / EffectQuest / KeyMapChange) split into its own follow-up? Default:
  keep as one task per the campaign request; phase internally in plan.md.
- `CharacterList` is unverified on *all* versions yet `CharacterViewAllCharacters` is
  mostly verified — is CharacterList's writer actually wired/emitted, or latent? Confirm.
- Duplicated rows (`AutoDistributeAp`, `EffectQuest`): enumerate by fname before
  counting "done".
- Which service owns each packet (login-flow vs channel in-field)?

## 10. Acceptance Criteria

- [ ] Every `character`-family row shows `verified` (✅) or `n-a` for all five versions
      in the regenerated matrix — no `incomplete` remains.
- [ ] Each duplicated-path row is verified per distinct fname.
- [ ] Every promoted cell has a `packet-audit:verify` byte-fixture and a fresh pinned
      evidence record (serverbound: + REPORT) committed together.
- [ ] `packet-audit matrix --check` (and fname-doc/operations `--check`) exit 0.
- [ ] Affected Go module(s): `go test -race ./...`, `go vet ./...`, `go build ./...`
      clean; `docker buildx bake` for any service whose `go.mod` was touched.
