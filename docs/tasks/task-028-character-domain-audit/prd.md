# Character-Domain Packet Audit — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-14
---

## 1. Overview

Apply the audit workflow built in task-027 to the character domain in `libs/atlas-packet`. The shipped login-domain audit caught four real wire bugs (`ServerStatusRequest` width, `AuthPermanentBan` trailing bytes, `GW_CharacterStat` HP/MP widening on v95+, `NEXON_ID_DIFFERENT_THEN_REGISTERED` silent-success) and seven template opcode shifts. The character domain is ~1.7× the size — 30 clientbound writers + 18 serverbound handlers, 48 packets total — and includes the hottest paths in the protocol (spawn, movement, attack, damage, buffs, skill changes). Any silent-success or width-mismatch bug here corrupts gameplay for every character every session, not just login flow.

This task ships:

1. Wire-shape verification of all 48 `libs/atlas-packet/character/**` packets against GMS v95 IDA, with file:line evidence per packet.
2. Cross-version re-verification of every change against GMS v83, GMS v87, and JMS v185 IDA so atlas-packet's `Region/MajorVersion` gates remain correct across the supported version matrix.
3. An analyzer fix for the documented `CharacterList ❌` false positive (early-return modeling in guarded branches). Character domain almost certainly has the same pattern in attack/damage/spawn packets, so fixing the analyzer here pays off immediately.
4. Real wire bug fixes in `libs/atlas-packet/character/**` and corresponding `services/atlas-configurations/seed-data/templates/template_gms_*_1.json` template adjustments for any opcode or enum drift.

The user has IDA Pro with binaries for v83, v87, v95, and JMS v185 wired to the `mcp__ida-pro__*` MCP tools and will swap binaries one at a time as each verification phase begins.

## 2. Goals

Primary goals:

- Audit all 48 character/clientbound and character/serverbound packets against v95 IDA — each gets a verdict (✅ / ⚠️ / ❌) backed by file:line citations in atlas-packet and IDA addresses for the matching `CWvsContext::OnXxxPacket` or `CClientSocket::Send*` function.
- Fix every real wire bug found. "Real wire bug" = clients decode different bytes than atlas writes, or write different bytes than atlas reads, on at least one supported version.
- Fix every template opcode/enum drift found, across all six version templates that ship character handlers (`template_gms_28_1.json`, `template_gms_83_1.json`, `template_gms_87_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json`).
- Cross-verify v95 changes against v83, v87, and JMS v185 IDA — same one-binary-at-a-time workflow used in the task-027 post-merge pass.
- Resolve the `CharacterList ❌` false-positive in the analyzer (early-return modeling).
- Extend `tools/packet-audit/`'s `TypeRegistry` incrementally for character-specific sub-structs (item snapshots, buff payloads, attack info, secondary stat blocks) as the audit surfaces them.

Non-goals:

- Performance benchmarks or optimization of character-domain packets. This is a correctness audit only.
- Changes to `services/atlas-character/`'s domain logic beyond the minimum needed to wire a wire-fix through.
- Audit of `atlas-channel`-side bare handlers (handlers whose decoders live in service code rather than atlas-packet types). Document gaps in `_pending.md`; don't descend.
- Audit of NPC, monster, drop, field, inventory, or any other non-character domain — those are sibling tasks.
- Migration to a new wire format or protocol change. This audit assumes the current atlas-packet shapes are the intended target; any divergence from IDA is a bug.

## 3. User Stories

- As a server operator, I want character-domain wire shapes to match v95 IDA byte-for-byte so that the client renders correct HP/damage/buff durations, doesn't desync on movement, and doesn't silently succeed on operations the server intended to fail.
- As an Atlas maintainer, I want a verdict + IDA address for every character packet so that future client-version bumps can be re-audited mechanically rather than re-discovered through gameplay bugs.
- As a contributor adding a new character packet, I want the analyzer to handle early-return guards correctly so that I can trust the audit pipeline's verdict without manually verifying every false positive.
- As a multi-version operator (GMS v83/v87/v95 or JMS v185), I want each `Region/MajorVersion` gate in atlas-packet to be backed by IDA evidence on every version it claims to support.

## 4. Functional Requirements

### 4.1 Coverage matrix

For each of the 48 `libs/atlas-packet/character/{clientbound,serverbound}/*.go` files, produce a row in `docs/packets/audits/gms_v95/SUMMARY.md` with:

- Atlas writer/handler name (FName).
- IDA function address and decompile reference.
- Verdict: ✅ pass, ⚠️ tolerable mismatch (loop-tagged trailing calls, etc.), ❌ wire bug.
- Notes — file:line citation when atlas writer was fixed; or `_pending.md` reference when out of scope.

Required attempt-coverage:

- 30 clientbound packets: `add_entry`, `add_entry_error`, `appearance_update`, `attack`, `buff_cancel`, `buff_give`, `chair_show`, `chalkboard`, `damage`, `delete_response`, `despawn`, `effect`, `effect_quest`, `effect_skill_use`, `expression`, `hint`, `info`, `item_upgrade`, `keymap`, `keymap_auto_hp`, `keymap_auto_mp`, `list`, `movement`, `name_response`, `sit_result`, `skill_change`, `skill_cooldown`, `spawn`, `status_message`, `view_all`.
- 18 serverbound packets: `auto_distribute_ap`, `buff_cancel`, `chair_fixed`, `chair_portable`, `chalkboard_close`, `check_name`, `create`, `delete`, `distribute_ap`, `distribute_sp`, `drop_meso`, `expression`, `heal_over_time`, `info_request`, `item_cancel`, `key_map_change`, `monster_damage_friendly`, `move`.

A packet may legitimately end up "documented out-of-scope" in `_pending.md` (e.g., bare handler with no atlas-packet type, or JMS-only writer with no GMS equivalent) — that counts as audited.

### 4.2 IDA export

Populate `docs/packets/ida-exports/gms_v95.json` (existing) and create the matching `gms_v83.json`, `gms_v87.json`, `gms_jms_185.json` files with character-domain FName entries during the cross-version pass. Each entry uses the existing schema (op list with `Decode1/2/4/Str/Buffer/Loop` operations + guard expressions).

### 4.3 TypeRegistry extensions

When the analyzer encounters a sub-struct that's not in the registry, add a `<Type>::Encode` or `<Type>::Write` analysis to `tools/packet-audit/internal/atlaspacket/registry.go`. Expected new types include (verified during audit, not exhaustive):

- `Inventory::Encode` or similar item-snapshot sub-structs in `spawn` and `view_all`.
- `BuffStatPayload` or per-buff payloads in `buff_give`.
- `AttackInfo` block in `attack` / `damage`.
- Secondary stat block in `appearance_update`.

Each addition is committed with a tagged test in `tools/packet-audit/internal/atlaspacket/registry_test.go`.

### 4.4 Analyzer fix — `CharacterList ❌` false positive

In `tools/packet-audit/internal/atlaspacket/analyzer.go` (or the appropriate file in that package), extend the static analyzer to model `return` statements inside guarded blocks as exclusive — when one branch returns, sibling branches should not both contribute to the static flat list of `Write*` calls. After the fix, re-run the login audit and confirm `CharacterList` flips ❌ → ✅ in `docs/packets/audits/gms_v95/SUMMARY.md`.

### 4.5 Cross-version re-verification

After v95 audit lands with all fixes shipped:

1. User loads v83 IDA. Verify each fixed file's `Region/MajorVersion` gate by decompiling the matching v83 function. Widen / narrow gates as needed.
2. Repeat for v87.
3. Repeat for JMS v185.

Each version's IDA export gets its character-domain entries during its phase. Findings get recorded in `post-phase-b.md` once all four versions are done.

### 4.6 Wire bug fixes

Real wire bugs found get fixed in atlas-packet, with the fix gated on the affected versions only. Every fix lands with:

- A test in the same `_test.go` file sweeping all four version variants (v28/v83/v95 + JMS v185).
- A 1-3 sentence comment in the encoder/decoder citing the specific IDA finding (function name + behavior).
- A row in `post-phase-b.md`'s "Real wire bugs fixed" table.

### 4.7 Template opcode/enum fixes

Template drift gets fixed in `services/atlas-configurations/seed-data/templates/template_gms_*.json` and `template_jms_185_1.json`. Every template fix lands with:

- A row in `post-phase-b.md`'s "Template opcode/enum fixes" table.
- A commit message citing the IDA case-statement value or opcode constant that justifies the fix.

## 5. API Surface

No new HTTP/REST endpoints. The audit produces:

- New / refreshed `tools/packet-audit/` analyzer behavior (internal CLI; no public API).
- New entries in `docs/packets/ida-exports/gms_{v83,v87,v95}.json` and `gms_jms_185.json`.
- New per-packet audit reports under `docs/packets/audits/gms_v95/character/*.md` and the SUMMARY index.
- Updated `template_*.json` opcode/enum values (config-data; consumed by atlas-configurations at runtime).

## 6. Data Model

No new persistent data. Template JSON files in `services/atlas-configurations/seed-data/templates/` get small key/value updates for any opcode or enum drift found. No migration needed — atlas-configurations re-reads templates on startup and the changes are wire-format adjustments, not schema changes.

## 7. Service Impact

| Service / Lib | Impact |
|---|---|
| `libs/atlas-packet/character/` | Wire-bug fixes in clientbound and serverbound encoders/decoders. Tests added per variant. |
| `libs/atlas-packet/model/` | Possibly new sub-struct types (e.g., `AttackInfo`, `BuffStatPayload`) if the existing `model/` doesn't already cover them. |
| `tools/packet-audit/` | TypeRegistry additions for character sub-structs. Analyzer fix for early-return guard modeling. |
| `services/atlas-configurations/seed-data/templates/` | Opcode/enum corrections across `template_gms_{28,83,87,95}_1.json` and `template_jms_185_1.json` as drift is found. |
| `services/atlas-character/` | Only touched if a wire bug surfaces a semantic field-name mismatch that propagates from packet to service. Audit-driven changes only. |
| `docs/packets/ida-exports/` | New / refreshed export JSON per version. |
| `docs/packets/audits/gms_v95/` | Per-packet audit reports + SUMMARY updates. |

## 8. Non-Functional Requirements

### Performance

No performance work in scope. Hot packets (movement, attack, damage, spawn) are audited for correctness only — atlas-packet's existing encoder hot paths stay byte-for-byte where possible. Any fix that changes the encoded byte sequence for an existing version is a real wire bug fix, not a performance regression.

### Security

Wire-shape mismatches that allow clients to corrupt server state (e.g., silent-success on delete, malformed length-prefixed strings) are treated as security-relevant findings and prioritized. No new attack surface introduced.

### Observability

No observability changes required. The audit pipeline is offline tooling; runtime telemetry is unchanged.

### Multi-tenancy

All `Region/MajorVersion` gates in atlas-packet must remain tenant-scoped via `tenant.MustFromContext(ctx)`. Fixes must not introduce global state or per-tenant configuration outside the existing template system.

### Test discipline

- No `*_testhelpers.go` files. Use the project's Builder pattern.
- Each fixed encoder/decoder gets a table-driven test sweeping every supported variant (v28/v83/v87/v95 + JMS v185).
- Tests verify byte output explicitly against expected hex sequences pulled from IDA decompiles, not just round-trip.

## 9. Open Questions

- **Bare handlers**: Several character serverbound handlers (`create`, `check_name`, possibly `delete`) may have inline decoders in `atlas-channel` or `atlas-character` rather than atlas-packet types. The login audit deferred these to `_pending.md`. Same treatment here, or descend into the service for the most-critical ones (e.g., `create`)? **Working assumption:** defer to `_pending.md`, mirror login behavior.
- **v28 coverage**: `template_gms_28_1.json` exists in the templates dir. The login audit didn't deeply verify v28 against IDA (no v28 binary in user's collection at the time). Same assumption here, or seek a v28 binary?
- **JMS character-domain divergence**: JMS v185 login dispatch was a separate opcode space from GMS. Character-domain may show similar or worse divergence. If JMS character is wildly different, do we still try to share atlas-packet code via `Region() == "JMS"` branches, or accept that the `jms` template is the integration point and JMS bugs become out-of-scope unless atlas-packet writes wrong bytes?
- **Inventory sub-struct ownership**: Character spawn includes an inventory snapshot. atlas-packet has `libs/atlas-packet/inventory/` — should spawn's inventory block be modeled by importing an existing inventory encoder, or via a character-domain-specific sub-type? **Working assumption:** import existing inventory encoders where they exist; only introduce new types when no fit.

## 10. Acceptance Criteria

- [ ] All 48 character/* packets have a row in `docs/packets/audits/gms_v95/SUMMARY.md` with ✅, ⚠️, or ❌ verdict + IDA address evidence.
- [ ] Every ❌ verdict produces a fix commit on the task branch (atlas-packet code or template JSON or both), with a citation in the commit message linking to the IDA function/address.
- [ ] `docs/packets/ida-exports/gms_v95.json` contains character-domain FNames for every packet attempted.
- [ ] `gms_v83.json`, `gms_v87.json`, `gms_jms_185.json` each get the matching character-domain entries during their phase.
- [ ] `CharacterList` flips from ❌ to ✅ in the login SUMMARY after the analyzer early-return fix lands.
- [ ] `post-phase-b.md` (this task's, not task-027's) summarizes: total packets audited, real wire bugs fixed, template fixes shipped, cross-version notes, deferred items in `_pending.md`.
- [ ] `go build ./...`, `go vet ./...`, `go test -race ./...` clean in `libs/atlas-packet/` and `tools/packet-audit/`.
- [ ] `docker build -f services/atlas-configurations/Dockerfile .` clean if the template seed-data structure changed (key additions don't break the build but new sub-key types might).
- [ ] plan-adherence-reviewer + backend-guidelines-reviewer both PASS before opening PR.
- [ ] gitleaks clean (no `/home/<user>/` absolute paths leaked into audit reports — `task-027-followup` PR caught this once; reviewers must scrub their own output now).
