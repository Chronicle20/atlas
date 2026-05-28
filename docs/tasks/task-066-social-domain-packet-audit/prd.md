# Social-Domain Packet Audit — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-15
---

## 1. Overview

task-027 built the packet-audit pipeline (analyzer, `tools/packet-audit/` CLI,
`EncodeForeign` registry, cycle guard, `candidatesFromFName` mappings) and
produced wire-correct login packets. task-028 applied the same pipeline to the
character domain (48 packets) and proved the workflow at scale: per-packet
audit reports, IDA-export JSON, 4-variant `pt.Variants` round-trip tests,
template opcode drift detection. Both tasks have shipped to main and the
pipeline is mature.

The social domain — guild, party, buddy, messenger, note, chat — is the largest
untouched grouping at ~147 packets. Social packets carry highly user-visible
state: guild ranks, party HP bars, buddy-list presence, messenger
conversations. Wire bugs here are subtle but user-impacting (wrong rank ID →
character sees wrong permissions; chat sub-type mismatch → megaphone text
appears as whisper or vice versa). The domain also contains the highest
density of sub-op dispatch in the whole library: guild BBS operations, party
leader/member operations, chat type-discriminators (whisper, megaphone, smega,
item-pop, avatar-megaphone). These sub-op families are the primary
`_pending.md` deferral risk.

GMS v83, v87, v95, and JMS v185 IDA exports are available for cross-version
gate validation. No prior tasks have directly targeted the social domain.

## 2. Goals

Primary goals:

- Produce per-packet audit reports under `docs/packets/audits/gms_v95/` for
  every packet in the guild, party, buddy, messenger, note, and chat domains
  (147 total).
- Land real wire-bug fixes with 4-variant `pt.Variants` round-trip tests
  (GMS v28 / v83 / v95 + JMS v185).
- Append social-domain IDA-export entries to
  `docs/packets/ida-exports/gms_v95.json` and matching `gms_v83.json`,
  `gms_v87.json`, `gms_jms_185.json` files.
- Verify template opcodes against IDA dispatcher case-statements for every new
  template entry touched.
- Defer sub-op enum drift (chat whisper/megaphone/smega/item-pop, guild BBS)
  to `_pending.md` per design §9.
- Defer bare handlers (atlas-side handlers without a `libs/atlas-packet`
  decoder) to `_pending.md` per task-028 design §1.

Non-goals:

- Business logic in `services/atlas-guild/`, `services/atlas-party/`,
  `services/atlas-buddies/`, or other social services.
- Re-auditing login (task-027) or character (task-028) domain packets.
- Sub-op enum expansion for chat type-discriminators beyond what the analyzer
  can statically resolve.
- Atlas-side bare handlers that have no `libs/atlas-packet` decoder.
- Guild BBS sub-op mode dispatch — flag as deferral-candidate on first
  encounter; confirm scope at audit time.

## 3. User Stories

- As a packet-correctness maintainer, I want a verdict (✅/⚠️/❌) for every
  social-domain packet so I know which ones are wire-verified, which are
  tool-limited, and which are bugs.
- As a backend engineer touching `libs/atlas-packet/{guild,party,buddy,
  messenger,note,chat}/`, I want a 4-variant test sweep so a wire-shape
  regression in any one version (GMS v28/v83/v95 or JMS v185) fails CI.
- As a future auditor extending the pipeline to the next domain, I want
  `tools/packet-audit/internal/atlaspacket/registry.go` to already cover the
  sub-structs used in social packets (guild member entry, party HP/MP bar
  entry, buddy list entry, messenger chat history entry).
- As an integrator running atlas against a live client at a specific version,
  I want template opcode values in `services/atlas-configurations/seed-data/
  templates/template_{gms_{12,28,83,87,92,95},jms_185}_1.json` to match IDA
  dispatcher case-statements at that version.

## 4. Functional Requirements

### 4.1 Coverage matrix

Produce a verdict row in `docs/packets/audits/gms_v95/SUMMARY.md` for each of
the 147 social-domain packets, enumerated from `libs/atlas-packet/<domain>/{
clientbound,serverbound}/`:

| Domain    | Clientbound | Serverbound | Total |
|-----------|-------------|-------------|-------|
| guild     | 10          | 37          | 47    |
| party     | 19          | 12          | 31    |
| buddy     | 12          | 6           | 18    |
| messenger | 15          | 10          | 25    |
| note      | 5           | 6           | 11    |
| chat      | 9           | 6           | 15    |
| **Total** | **70**      | **77**      | **147** |

Each row carries:

- FName
- IDA function address
- Verdict (✅ / ⚠️ / ❌)
- Notes with `file:line` citation or `_pending.md` reference

The 37-count for `guild/serverbound/` is unusually high — many are likely
sub-op variants of a single dispatcher packet (e.g., `guild_operation.go`
variants). Audit determines how many are analyzer-addressable vs. deferrable
sub-op drift.

### 4.2 IDA exports

Populate social-domain FName entries in
`docs/packets/ida-exports/gms_v95.json` for all 147 packets. During the
cross-version pass, add matching entries to `gms_v83.json`, `gms_v87.json`,
and `gms_jms_185.json`. Use the existing entry schema established by
task-027/028.

### 4.3 Wire bug fixes

Every ❌ verdict gets a fix in `libs/atlas-packet/<domain>/{clientbound,
serverbound}/*.go`, gated on affected versions only. Each fix lands with:

- A 4-variant test sweep using `pt.Variants` (GMS v28/v83/v95 + JMS v185).
- A 1–3 sentence comment citing the IDA function name and finding.
- A row in `post-phase-b.md`'s "Real wire bugs fixed" table.

### 4.4 Template fixes

Template opcode/enum drift gets fixed in
`services/atlas-configurations/seed-data/templates/template_gms_*.json` and
`template_jms_185_1.json`. Each fix lands with:

- A row in `post-phase-b.md`'s "Template opcode/enum fixes" table.
- A commit message citing the IDA case-statement value.

### 4.5 TypeRegistry extensions

When the analyzer surfaces an unregistered sub-struct type, extend
`tools/packet-audit/internal/atlaspacket/registry.go`. Likely new types for
the social domain:

- Guild member entry sub-struct (used in `guild/clientbound/` member-list
  packets).
- Party member HP/MP bar sub-struct (used in `party/clientbound/`).
- Buddy list entry sub-struct (used in `buddy/clientbound/`).
- Messenger chat history entry.

Each registry addition commits with a tagged test in `registry_test.go`.

### 4.6 Cross-version re-verification

After the v95 primary pass, the user loads v83 IDA and verifies each fixed
file's `Region/MajorVersion` gate. Repeat for v87, then JMS v185. Per-version
notes recorded in `post-phase-b.md`.

### 4.7 Deferral handling

Sub-op enum drift (chat type-discriminators, guild BBS operations, party
operation codes) and atlas-side bare handlers are recorded in
`docs/packets/audits/gms_v95/_pending.md`. Each entry: packet FName, IDA
address (if known), reason for deferral, and pointer to the dispatch family
it belongs to.

## 5. API Surface

No new HTTP/REST endpoints. The audit produces:

- Updated `tools/packet-audit/` TypeRegistry entries.
- New per-packet audit reports under
  `docs/packets/audits/gms_v95/{guild,party,buddy,messenger,note,chat}/`.
- Updated `docs/packets/ida-exports/gms_{v83,v87,v95}.json` and
  `gms_jms_185.json`.
- Updated `template_*.json` opcode/enum values for any drift found.

## 6. Data Model

No persistent-data changes. The configuration JSON files
(`services/atlas-configurations/seed-data/templates/`) receive opcode
corrections only — schema is unchanged.

## 7. Service Impact

| Service / Lib                                          | Impact                                                              |
|--------------------------------------------------------|---------------------------------------------------------------------|
| `libs/atlas-packet/guild/`                             | Wire-bug fixes in clientbound/serverbound encoders. Tests per variant. |
| `libs/atlas-packet/party/`                             | Wire-bug fixes. Tests per variant.                                  |
| `libs/atlas-packet/buddy/`                             | Wire-bug fixes. Tests per variant.                                  |
| `libs/atlas-packet/messenger/`                         | Wire-bug fixes. Tests per variant.                                  |
| `libs/atlas-packet/note/`                              | Wire-bug fixes. Tests per variant.                                  |
| `libs/atlas-packet/chat/`                              | Wire-bug fixes. Tests per variant.                                  |
| `tools/packet-audit/`                                  | TypeRegistry additions for social sub-structs.                      |
| `services/atlas-configurations/seed-data/templates/`   | Opcode/enum corrections.                                            |
| `docs/packets/ida-exports/`                            | New/refreshed export JSON per version.                              |
| `services/atlas-channel`, `atlas-guild`, `atlas-party`, `atlas-buddies` | Constructor-signature ripple verification only; no logic changes expected. |

## 8. Non-Functional Requirements

- All fixes ship with 4-variant test sweeps using `pt.Variants` (GMS
  v28/v83/v95 + JMS v185).
- Hard cap: no encoder/decoder grows beyond 2 nested region/version guards
  (per task-028 design §7). 3+ nested → STOP, log to `_pending.md`.
- `gitleaks` scrub clean before PR (no `/home/<user>/` absolute paths in audit
  reports).
- `go test -race ./libs/atlas-packet/... ./tools/packet-audit/...` clean.
- `go vet ./libs/atlas-packet/...` clean.
- `go build ./...` clean across the workspace.
- Login (task-027) and character (task-028) verdicts unchanged.

### Working assumptions

- IDA v95 is the primary target; v83/v87/JMS v185 IDA available for
  cross-version pass when user loads them.
- Templates `template_gms_{12,28,83,87,92,95}_1.json` and
  `template_jms_185_1.json` exist; new opcodes added per audit.
- `guild/serverbound/` 37-file count is likely sub-op-driven: many files
  represent individual guild operation variants routed by a leading-byte
  discriminator. The analyzer surfaces these as distinct static sequences but
  the common dispatcher packet is flagged for sub-op deferral per design §9.
- Chat sub-types (whisper, megaphone, smega, item-pop, avatar megaphone) are
  the classic sub-op enum drift pattern — each lands in `_pending.md` unless
  the atlas-packet files already encode the discriminator byte as a literal
  in each individual file (in which case, audit each individually).
- Party leader-transfer, kick, and invite-decline operations may carry
  version-sensitive field additions in v95. Cross-version pass surfaces gate
  boundaries.

### Key risks / patterns to watch for

(carried forward from task-027/028 lessons)

- **Sub-op enum drift (primary risk)**: Chat type-discriminators, guild BBS
  operations, and party operation codes are all leading-byte dispatch
  families. Analyzer sees each file's static `Write*` sequence but cannot
  model the discriminator byte pattern. Heavy `_pending.md` deferrals
  expected.
- **Dispatcher-layer offset**: CUserPool dispatchers prepend `characterId`
  before routing — atlas wire includes it at offset 0. Guild/party membership
  packets encoding member IDs are susceptible.
- **`EncodeMask` / sub-struct method calls**: appear as one analyzer call but
  emit multiple bytes — ack as tool-limitation.
- **Loop linearization**: Guild member lists, buddy lists, party member bars
  are fixed-count loops. Analyzer flattens them — ack as tool-limitation;
  verify against IDA loop bounds.
- **Dispatcher case-statement validation**: every new template opcode MUST be
  confirmed against IDA dispatcher decompile before commit. Do not guess.
- **Cross-version gate boundaries**: party/guild feature sets differ
  significantly between v83 and v95 (alliance system, PQ group system). Don't
  assume gate boundaries until cross-version IDA confirms.
- **Hidden constructor-signature ripples**: adding fields to encoder structs
  ripples to `atlas-channel`, `atlas-guild`, `atlas-party`, `atlas-buddies`
  handlers — verify build clean across services.
- **Audit-report ack footers**: add ack footer AFTER the final audit run; if
  re-running, `git checkout HEAD -- <report.md>` to revert.

## 9. Open Questions

- Are there v95-only social features (alliance, cross-server party, etc.)
  that add new fields to guild or party packets with no v83 equivalent?
  Cross-version pass will reveal.
- `guild/serverbound/` 37-file count: confirm at design time whether these
  are truly independent packet types or sub-op variants. If sub-op variants,
  how many are deferrable vs. analyzer-addressable?
- JMS v185 social domain: JMS guild and party features diverge noticeably
  from GMS (JMS has fewer alliance tiers, different BBS structure). If JMS
  packet shapes are wildly different, how much sharing via `Region() == "JMS"`
  branches is feasible vs. deferred?

## 10. Acceptance Criteria

- [ ] All 147 listed packet files have audit reports under
  `docs/packets/audits/gms_v95/`.
- [ ] Every ❌ verdict has either a fix commit OR a `_pending.md` row.
- [ ] All 4 verification commands pass cleanly:
  - `go build ./...`
  - `go vet ./libs/atlas-packet/...`
  - `go test -race ./libs/atlas-packet/...`
  - `go test -race ./tools/packet-audit/...`
- [ ] `gitleaks` scrub clean (no absolute home-directory paths in audit
  reports).
- [ ] `post-phase-b.md` ledger written, with "Real wire bugs fixed" and
  "Template opcode/enum fixes" tables populated.
- [ ] `plan-adherence-reviewer` and `backend-guidelines-reviewer` dispatched
  and findings addressed before PR.
- [ ] Login (task-027) and character (task-028) audit verdicts unchanged.

## 11. Dependencies

- task-027 (login audit, analyzer baseline) — merged.
- task-028 (character audit, `EncodeForeign` registry, cycle guard, ack
  pattern) — merged.
