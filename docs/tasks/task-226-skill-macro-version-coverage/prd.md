# Skill Macro Version Coverage — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-13
---

## 1. Overview

Skill macros (the client's `CMacroSysMan` — up to N named rows binding three skill ids
plus a "shout" flag, edited in the client's Skill window) are fully implemented on the
server. `services/atlas-channel/atlas.com/channel/macro/` persists them, the serverbound
handler `CharacterSkillMacroHandleFunc` is registered in `main.go:943`, and the
clientbound writer is announced both at login
(`kafka/consumer/session/consumer.go:373`) and on every macro update
(`kafka/consumer/macro/consumer.go:85`). The codec lives at
`libs/atlas-packet/character/skill_macro.go`.

What is missing is **routing**. A handler or writer only reaches the socket layer if the
tenant's socket-config template binds it to an opcode. Auditing all eleven seed templates
under `services/atlas-configurations/seed-data/templates/` shows the serverbound handler
is bound only on gms_72/79/83/84, and the clientbound writer is bound everywhere except
gms_12/48/92. Consequently players on gms_87, gms_95 and jms_185 receive their macro list
at login but **cannot save changes** — every `CMacroSysMan::FlushToSvr` is dropped at
dispatch — and players on gms_92 can neither save nor load. This is the same class of
defect as `bug_new_opcodes_not_in_live_tenant_config`: green builds, green tests, silent
feature loss for a whole tenant.

Separately, neither direction of this packet has ever been byte-verified. The coverage
matrix carries `SKILL_MACRO | CMacroSysMan::FlushToSvr; sub_6022DB` (STATUS.md:650) and
`MACRO_SYS_DATA_INIT | CWvsContext::OnMacroSysDataInit` (STATUS.md:177) as ❌ in every
column. The only existing test, `libs/atlas-packet/character/skill_macro_test.go`, is a
round-trip encode→decode against the codec's own encoder — it proves self-consistency and
nothing about the wire (see `bug_matrix_roundtrip_fixture_false_verify`). The codec is
also entirely version-blind: one layout, no `MajorAtLeast` gates, for a version span from
GMS v61 to JMS v185. Registering the missing opcodes without deriving the real per-version
layout would ship a decoder that is wrong in a new and quieter way.

This task therefore does both: derive the per-version field layout from the IDBs,
version-gate the codec where the layout diverges, byte-verify every applicable
op × version cell to ✅, and bind the missing opcodes in the seed templates.

## 2. Goals

Primary goals:

- Every GMS/JMS version whose client implements skill macros can both **save** and
  **load** them: the serverbound handler and the clientbound writer are bound in that
  version's seed template.
- The per-version wire layout of both packets is **derived from the client**, not
  assumed, and encoded as explicit version gates in `skill_macro.go`.
- Both matrix rows (`SKILL_MACRO`, `MACRO_SYS_DATA_INIT`) reach ✅ in every applicable
  column, with pinned evidence records and byte fixtures.
- The `⬜ n-a` claims for gms_v48 (both rows) and gms_v61 (`SKILL_MACRO` only) are
  re-checked against the v48/v61 IDBs and either confirmed or corrected.

Non-goals:

- Changing macro persistence semantics, the `macro` REST/Kafka contract, or the
  `atlas-character`-side storage. This task is routing + codec only.
- The `ANTI_MACRO_*` family (`ANTI_MACRO_RESULT`, `ANTI_MACRO_ITEM_USE`,
  `ANTI_MACRO_TARGET`) — an unrelated anti-botting feature that merely shares a name
  fragment.
- `template_gms_12_1.json`. v12 is not a coverage-matrix column and is out of scope.
- PATCHing live tenant socket configurations. The user will apply those post-merge; this
  task delivers the seed templates and the reconciliation input for that step (FR-5.2).
- Any UI surface for macros in atlas-ui.

## 3. User Stories

- As a player on a gms_87, gms_92, gms_95 or jms_185 tenant, I want to edit my skill
  macros in the client and have them persist across logins, so the feature behaves the
  same as it does on gms_83.
- As a player on a gms_92 tenant, I want my saved macros to appear in the Skill window
  when I log in, rather than an empty macro list.
- As a server developer, I want the macro codec to carry explicit version gates so that
  adding a future version column forces an explicit layout decision instead of silently
  inheriting v83's.
- As a maintainer reading the coverage matrix, I want `SKILL_MACRO` and
  `MACRO_SYS_DATA_INIT` to reflect real byte-level verification, so a ✅ means the client
  accepted the bytes rather than that our encoder agrees with our decoder.

## 4. Functional Requirements

### FR-1 — Per-version layout derivation

- **FR-1.1** For each version column in scope (gms_v48, v61, v72, v79, v83, v84, v87,
  v92, v95, jms_185), decompile the client's `CMacroSysMan::FlushToSvr` (serverbound
  send site) and `CWvsContext::OnMacroSysDataInit` (clientbound read site) from that
  version's IDB, following `docs/packets/audits/VERIFYING_A_PACKET.md`. On versions where
  the symbol is unnamed, resolve it from the opcode dispatch table and **name the symbol
  in the IDB** (per `feedback_name_idb_symbols_while_reversing`); an unnamed function is
  not evidence of an absent feature (`feedback_unnamed_idb_function_still_exists`).
- **FR-1.2** Record, per version, the observed field order and widths for at least:
  the leading macro-count type and its maximum value, the name string encoding
  (length-prefixed ASCII vs fixed-width), the shout flag's polarity and width, and the
  three skill-id widths. The current codec (`skill_macro.go:41-75`) assumes
  `byte count`, `WriteAsciiString name`, `WriteBool !shout`, `3 × uint32` — treat that as
  a hypothesis to confirm per version, not a baseline to diff against.
- **FR-1.3** Where a version's layout diverges, gate it in `skill_macro.go` using the
  `MajorAtLeast` idiom against the resolved version from `readerOptions`/`options` —
  never a raw `> N` comparison (see `bug_majorversion_gt83_is_off_by_one_v87`).
- **FR-1.4** Where a version's layout matches, make **no wire change** to that version's
  bytes. Versions already in production (gms_83, gms_84) must round-trip identically
  before and after this task.
- **FR-1.5** Document the derivation in
  `docs/tasks/task-226-skill-macro-version-coverage/layout-derivation.md`: one row per
  version × field, each citing the IDB function and decompiled line it came from. Any
  field that could not be resolved is recorded as **unknown**, never guessed.

### FR-2 — n-a re-check

- **FR-2.1** Re-check `SKILL_MACRO` for gms_v48 and gms_v61 (both currently `⬜`) and
  `MACRO_SYS_DATA_INIT` for gms_v48 (currently `⬜`) against those IDBs. Determine
  whether the client has a macro system at all at that version, and if it does, whether
  it sends a flush packet.
- **FR-2.2** If `⬜` is confirmed, record the confirming evidence (absent symbol *and*
  absent dispatch entry) so the `n-a` consistency gate
  (`reference_packet_na_consistency_gate`) has something to stand on.
- **FR-2.3** If `⬜` is wrong, the cell is corrected: the opcode is added to the registry,
  the template binding is added, and the cell is verified like any other in FR-3/FR-4.
  Correcting a wrong `n-a` is in scope, not a follow-up task.

### FR-3 — Codec verification (serverbound `SKILL_MACRO`)

- **FR-3.1** Write a byte-fixture test per applicable version, carrying a
  `packet-audit:verify` marker, decoding a fixture whose bytes are derived from the
  client read/write order established in FR-1 — not from our own `Encode`.
- **FR-3.2** Delete or rewrite `libs/atlas-packet/character/skill_macro_test.go`'s
  round-trip-only assertion so it can no longer be mistaken for verification. A
  round-trip test may remain alongside the fixtures, but it must not be the sole coverage
  for any cell.
- **FR-3.3** Pin the evidence record and regenerate the matrix. Every applicable
  `SKILL_MACRO` cell reads ✅.

### FR-4 — Codec verification (clientbound `MACRO_SYS_DATA_INIT`)

- **FR-4.1** Same procedure as FR-3, for the `Encode` direction, against
  `CWvsContext::OnMacroSysDataInit`'s read order.
- **FR-4.2** Every applicable `MACRO_SYS_DATA_INIT` cell reads ✅.

### FR-5 — Template routing

- **FR-5.1** Bind the missing entries in
  `services/atlas-configurations/seed-data/templates/`:

  | Template | Add to `handlers` | Add to `writers` |
  |---|---|---|
  | `template_gms_87_1.json` | `CharacterSkillMacroHandle` @ `0x071` | — (already `0x084`) |
  | `template_gms_92_1.json` | `CharacterSkillMacroHandle` @ `0x079` | `CharacterSkillMacro` @ `0x08B` |
  | `template_gms_95_1.json` | `CharacterSkillMacroHandle` @ `0x07A` | — (already `0x08C`) |
  | `template_jms_185_1.json` | `CharacterSkillMacroHandle` @ `0x069` | — (already `0x07A`) |

  Opcodes are taken from the registry / coverage matrix rows, which are the authority
  (`feedback_verify_packets_not_cross_version_opcodes`), and re-confirmed against the
  IDB dispatch table during FR-1. Any binding added under FR-2.3 extends this table.
- **FR-5.2** Every new handler entry carries `"validator": "LoggedInValidator"` and a
  `fname`, matching the existing gms_83/84 entries. A handler with a missing or unknown
  validator is silently dropped at load
  (`bug_socket_handler_missing_validator_silently_dropped`); a writer without `fname`
  fails seeding (`bug_seed_template_writers_require_fname`).
- **FR-5.3** New entries are inserted at their **sorted** position within the `handlers`
  and `writers` arrays, never appended next to a semantically-related entry
  (`docs/packets/TEMPLATE_CONVENTIONS.md`).
- **FR-5.4** Produce
  `docs/tasks/task-226-skill-macro-version-coverage/live-tenant-reconciliation.md`
  listing, per affected live tenant version, the exact entries to add — the input for the
  post-merge PATCH the user will run.

### FR-6 — Guards

All of the following must be clean from the repo root before the branch is claimed done:

- **FR-6.1** `tools/template-opcode-order-guard.sh`
- **FR-6.2** `tools/template-duplicate-binding-guard.sh`
- **FR-6.3** `tools/lint.sh --check`
- **FR-6.4** `go test -race ./...`, `go vet ./...`, `go build ./...` in every changed
  module (`libs/atlas-packet`, `services/atlas-channel`, `services/atlas-configurations`)
- **FR-6.5** `docker buildx bake atlas-channel` and `docker buildx bake
  atlas-configurations` if either service's `go.mod` was touched
- **FR-6.6** `packet-audit matrix --check` (or the equivalent from
  `docs/packets/PROCESS.md`) and the fname-doc / n-a consistency gates

## 5. API Surface

No REST or Kafka contract changes. The existing surfaces are unchanged and listed here
for reference only:

- `services/atlas-channel/atlas.com/channel/macro/rest.go` — macro fetch/update against
  atlas-character.
- `kafka/message/macro` — `skill_macro_status_event` topic, `StatusEventTypeUpdated`.
- Socket: serverbound `CharacterSkillMacroHandle`, clientbound `CharacterSkillMacro`.
  Only their **opcode bindings** change, per version, in the seed templates.

## 6. Data Model

No schema changes. `macro.Model` (`services/atlas-channel/atlas.com/channel/macro/model.go`)
and its persisted counterpart are untouched.

The only structural change is inside the codec: `SkillMacroEntry` may gain fields, or
existing fields may gain per-version widths, if FR-1 finds divergence. Any such change is
internal to `libs/atlas-packet/character` and must not alter the `macro.Model` mapping in
`socket/handler/character_skill_macro.go` unless FR-1 proves a field is version-dependent
in a way the model cannot represent — in which case the mapping change is documented in
`layout-derivation.md`.

## 7. Service Impact

| Service / module | Change |
|---|---|
| `libs/atlas-packet` | `character/skill_macro.go` gains per-version gates; `character/skill_macro_test.go` replaced/extended with byte fixtures. |
| `services/atlas-configurations` | Four seed templates gain handler bindings; `template_gms_92_1.json` also gains a writer binding. Possibly gms_48/gms_61 under FR-2.3. |
| `services/atlas-channel` | No source change expected — handler and writer are already wired. If FR-1 forces a `SkillMacroEntry` field change, `socket/handler/character_skill_macro.go` is updated to match. |
| `docs/packets/audits` | `STATUS.md` / `status.json` regenerate; new evidence records pinned for both rows. |
| Live tenants | Out of scope for this branch; FR-5.4 produces the reconciliation input. |

## 8. Non-Functional Requirements

- **Multi-tenancy:** All routing is per-tenant socket config. No global opcode constants
  are introduced; the mode/opcode values stay config-resolved
  (`feedback_client_wire_values_config_resolved`, DOM-25).
- **Backward compatibility:** gms_83 and gms_84 byte output must be unchanged. This is
  verified by fixtures, not by inspection.
- **Observability:** No new logging is required. The existing handler already logs decode
  failures and update failures. If FR-1 reveals a version where the count byte can exceed
  the client's macro capacity, bound the decode loop rather than trusting the wire.
- **Grounding:** Every opcode, field width, and `n-a` claim in the delivered artifacts
  cites an IDB function or a repo file:line. Anything unresolved is written as
  **unknown**, never filled with a plausible value.

## 9. Open Questions

- Does the macro-count field stay one byte across the whole v48→jms185 span, and does the
  client cap it (commonly at 5)? Resolved by FR-1.2 — not assumed here.
- Is the shout flag's polarity really inverted (`WriteBool(!e.Shout)` at
  `skill_macro.go:47`) on every version, or is that an artifact of a single version's
  reversing? Resolved by FR-1.2.
- Does jms_185 diverge from the GMS layout (JMS string encoding differs elsewhere)?
  Resolved by FR-1.1.
- gms_v72 and gms_v79 carry `fname: sub_6022DB` rather than a named symbol. FR-1.1
  requires naming it; if the same `sub_` address is reused across two different IDBs by
  coincidence, that is a data error in the registry to be corrected here.

## 10. Acceptance Criteria

- [ ] `layout-derivation.md` exists, with a per-version × per-field table, every row
      citing an IDB function; unresolved fields marked **unknown**.
- [ ] The v48/v61 `⬜` cells are re-checked; the outcome (confirmed n-a, or corrected and
      implemented) is recorded with evidence.
- [ ] `skill_macro.go` carries explicit `MajorAtLeast`-style gates for every divergence
      found, and no raw `> N` version comparisons.
- [ ] Byte-fixture tests with `packet-audit:verify` markers exist for both rows, one per
      applicable version, derived from the client read order.
- [ ] `docs/packets/audits/STATUS.md` shows ✅ for `SKILL_MACRO` and
      `MACRO_SYS_DATA_INIT` in every applicable column; no cell remains ❌ without a
      recorded reason.
- [ ] gms_83 and gms_84 encoded bytes are byte-identical to `main` for a fixed input
      (regression fixture).
- [ ] `template_gms_87_1.json`, `template_gms_92_1.json`, `template_gms_95_1.json`,
      `template_jms_185_1.json` each bind `CharacterSkillMacroHandle` with
      `LoggedInValidator` + `fname`, at the sorted position.
- [ ] `template_gms_92_1.json` binds `CharacterSkillMacro` in `writers` with an `fname`,
      at the sorted position.
- [ ] `tools/template-opcode-order-guard.sh`, `tools/template-duplicate-binding-guard.sh`,
      and `tools/lint.sh --check` are all clean from the repo root.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed
      module; `docker buildx bake` clean for any service whose `go.mod` changed.
- [ ] `live-tenant-reconciliation.md` lists the exact per-version entries for the
      post-merge live PATCH.
- [ ] Code review (`superpowers:requesting-code-review`) run before the PR is opened.
