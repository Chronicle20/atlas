# CWvsContext::OnMessage Dispatcher Family Migration — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-18
---

## 1. Overview

The `MESSAGE` opcode (`CWvsContext::OnPacket` case 0x26 → `CWvsContext::OnMessage`) is a
clientbound **dispatcher family**: the client reads a leading mode byte (0x0–0xF) and
delegates to one of ~16 sub-handlers (`OnDropPickUpMessage`, `OnQuestRecordMessage`,
`OnIncEXPMessage`, …), several of which fan out further into specific message variants.
Atlas implements this family with **24 `StatusMessage*` structs** in
`libs/atlas-packet/character/clientbound/status_message.go` (writer `CharacterStatusMessage`),
covering drop/pickup feedback, quest-record updates, exp/meso/fame/SP/guild-point gains,
buff grants, item-expiry notices, and system messages.

Today the packet-audit tooling treats the entire family as a **single representative**:
`tools/packet-audit/cmd/run.go` maps `CWvsContext::OnMessage` to exactly one candidate,
`StatusMessageDropPickUpInventoryFull`. The coverage matrix therefore has **one row**
(`SHOW_STATUS_INFO`, STATUS.md) for all 24 arms — graded ✅ on `gms_v83/v84/v87/v95` and
❌ on `jms_v185`. That ✅ only verifies the outermost mode byte plus one arm's body; the
other 23 arms are never individually byte-verified. This is the "passes on one byte"
false-pass pattern that the canonical discrete-per-mode dispatcher migration exists to
eliminate (most recently applied to the guild family in task-103).

This task migrates `CWvsContext::OnMessage` to the canonical discrete-per-mode pattern
(`docs/packets/DISPATCHER_FAMILY.md`): one `#`-entry per arm, config-resolved mode bytes,
per-version byte fixtures with IDA citations, and every supported arm driven to verified
across all five versions. It also resolves the jms ❌ — whose root cause is an
export-completeness gap (the jms export models the 16 per-mode delegates but none of the
delegated sub-functions are present in the export).

## 2. Goals

Primary goals:
- Decompose `CWvsContext::OnMessage` from its single-representative wiring into discrete
  per-mode (and per-sub-mode) `#`-entries — one per the 24 `StatusMessage*` arms.
- Config-drive every arm's mode byte via the tenant `operations` table for the
  `CharacterStatusMessage` writer (no literal mode bytes), across all supported versions.
- Drive every one of the 24 arms to verified (✅) across `gms_v83/v84/v87/v95/jms_v185`,
  with per-version byte fixtures + `// packet-audit:verify` markers + IDA citations.
  Version-absent arms are marked ⬜ (never ❌, never fabricated ✅).
- Promote jms from ❌ by splicing the 16 missing per-mode delegate sub-functions into the
  jms export and decomposing the GMS exports' flat `OnMessage` entry into the same
  per-mode delegate structure.
- Leave the family fully migrated and `dispatcher-lint`-clean (no baseline entry needed).
- Keep all four packet-audit gates (`dispatcher-lint`, `matrix --check`,
  `fname-doc --check`, `operations --check`) at exit 0.

Non-goals:
- Changing the business logic of the ~8 atlas-channel consumers that emit these messages
  (quest, drop, compartment, asset, system_message, character, conversation_reward_notice).
  Call sites are re-routed through new config-driven body functions only.
- Adding new message arms not already present in the client switch / Atlas struct set
  (no inventing). The arm set is the existing 24 `StatusMessage*` structs.
- Any DB schema or REST API change.
- Implementing the live tenant config patch as part of the merge (it is a post-deploy
  step; see §7 and §9).

## 3. User Stories

- As a **packet-audit maintainer**, I want each of the 24 OnMessage arms individually
  byte-verified per version so that a green matrix cell means the arm is actually correct,
  not just that the mode byte decodes.
- As a **channel developer**, I want OnMessage mode bytes resolved from tenant config
  (like every other migrated dispatcher family) so version drift is handled uniformly and
  a new tenant version doesn't silently send the wrong sub-op.
- As a **jms tenant operator**, I want the MESSAGE family to work on jms (currently ❌) so
  drop/quest/exp/system messages render correctly.
- As a **reviewer**, I want `dispatcher-lint` to scan OnMessage clean so the family can't
  regress to a caller-selectable or literal-mode footgun.

## 4. Functional Requirements

### Grounding & honesty (FR-1)
- FR-1.1: Every mode byte, sub-mode, field width, and per-version presence MUST trace to a
  decompile line (function + address) or a checked-in export entry, cited in the
  struct/test comment. No values from MapleStory general knowledge or memory.
- FR-1.2: Resolve each IDB via `select_instance(port)` and confirm the version before
  reading. v84 has a real IDB (port 13337) — verify v84 directly; do not fold from v83 by
  assumption (task-103 found a real v84≠v83 divergence in the guild Invite arm).
- FR-1.3: An unresolved packet-audit fname or a switch case needing a brand-new key is a
  stop-and-ask — never auto-substitute, fake a hash, or invent.
- FR-1.4: Gate version divergence as `>=87`, never `>83`, except where IDA proves a
  different boundary.
- FR-1.5: No `// TODO`, stub, or 501 in any landed commit; finish bounded work.

### IDA enumeration (FR-2)
- FR-2.1: Decompile `CWvsContext::OnMessage` per version (v83/v84/v87/v95/jms) and record
  the complete outer switch (mode 0x0–0xF) with the delegated sub-handler per arm.
- FR-2.2: For each delegated sub-handler that fans out (notably `OnDropPickUpMessage` and
  `OnQuestRecordMessage`), decompile it and record the inner sub-mode → specific
  `StatusMessage*` arm mapping and read order, per version.
- FR-2.3: Produce a grounded table mapping each of the 24 `StatusMessage*` structs to its
  (outer mode, inner sub-mode, read order, per-version mode byte, present?✅/⬜).
- FR-2.4: Capture per-version function addresses for every outer sub-handler and every
  inner arm for citation.

### Discrete-per-mode structs (FR-3)
- FR-3.1: Each of the 24 arms is its own discrete struct (they already exist) with the
  mode byte(s) injected via constructor — no hard-coded `mode: 0x` struct literals.
- FR-3.2: The full body of each arm is encoded and byte-verified (not mode-byte-only).
- FR-3.3: No single struct maps to more than one mode/arm.

### Config-driven modes (FR-4)
- FR-4.1: Every arm resolves its mode byte(s) via `WithResolvedCode("operations", KEY, …)`
  against the `CharacterStatusMessage` writer's `operations` table — for the outer mode and
  for any inner sub-mode that is config-relevant. No literal mode bytes in the body funcs.
- FR-4.2: The `CharacterStatusMessage` writer is registered with an `operations` map
  populated for every supported version's mode bytes across all five seed templates
  (register the writer where missing; populate via `packet-audit operations`).
- FR-4.3: Body functions take only data parameters, never a caller-selectable op/key
  selector (no `*ErrorBody(code)`-style footgun).

### run.go rewire (FR-5)
- FR-5.1: Replace the single `CWvsContext::OnMessage` candidate with one `#`-entry per arm
  (per-mode / per-sub-mode), each returning its discrete struct candidate.
- FR-5.2: Remove the single-representative phantom mapping; the bare root must not return a
  representative (match the exemplar's handling, e.g. guild `OnGuildResult` /
  `OnFieldEffect`).
- FR-5.3: Comments reflect the current per-version verdict (no stale "deferred to
  _pending.md" banners once arms are enumerated).

### Export completeness (FR-6)
- FR-6.1: Splice the 16 missing per-mode delegate sub-functions into the jms export
  (`OnDropPickUpMessage`, `OnQuestRecordMessage`, `OnCashItemExpireMessage`,
  `OnIncEXPMessage`, `OnIncSPMessage`, `OnIncPOPMessage`, `OnIncMoneyMessage`,
  `OnIncGPMessage`, `OnGiveBuffMessage`, `OnGeneralItemExpireMessage`, `OnSystemMessage`,
  `OnQuestRecordExMessage`, `OnItemProtectExpireMessage`, `OnItemExpireReplaceMessage`,
  `OnSkillExpireMessage`, and the mode 0xF handler currently exported as `sub_B0931C` —
  resolve/name it from the jms IDB), plus any inner fan-out sub-functions, each with real
  IDA addresses (surgical splice, never overwrite the file).
- FR-6.2: The GMS exports (v83/v87/v95) currently have a flat `OnMessage` entry (0
  delegates). Add the same per-mode delegate structure (Decode1 + guarded Delegate refs)
  and splice any referenced sub-functions, so the audit decomposes the family on GMS too.
- FR-6.3: Export `address` fields and `// packet-audit:verify` marker `ida=` fields carry
  real decompile addresses (no `ida=0x0` placeholders); evidence records pin a real
  `decompile_sha256`.

### Per-version verification (FR-7)
- FR-7.1: Each of the 24 arms has a per-version byte fixture with a `// packet-audit:verify`
  marker and IDA citation for every supported version; version-absent arms are ⬜.
- FR-7.2: A row reaches ✅ only when its codec is genuinely byte-verified against that
  version's IDA. Re-pin a stale cell only after confirming the existing codec matches the
  decompiled read order; otherwise fix the codec.
- FR-7.3: Drive jms from ❌ to verified for every jms-present arm.

### Call-site migration (FR-8)
- FR-8.1: Re-route the ~8 atlas-channel consumers that emit `StatusMessage*` through the
  new config-driven body functions; no business-logic change.
- FR-8.2: Every serverbound/clientbound handler/writer entry the family touches retains a
  non-empty validator / opcode in the seed templates (avoid the silently-dropped trap).

### Seed templates & matrix (FR-9)
- FR-9.1: Reconcile every version's `CharacterStatusMessage` `operations` table against the
  enumerated mode table; populate across all five templates.
- FR-9.2: Regenerate `STATUS.md` / `status.json` via `packet-audit matrix`; `matrix --check`
  exits 0 (regenerate after committing any merge so the `toolSha` stamp matches HEAD).

### Gates (FR-10)
- FR-10.1: `dispatcher-lint`, `matrix --check`, `fname-doc --check`, `operations --check`
  all exit 0; OnMessage is scanned clean and needs no `dispatcher-lint-baseline.yaml` entry.
- FR-10.2: `go build`/`go vet`/`go test -race` clean in every changed module
  (`libs/atlas-packet`, `tools/packet-audit`, `services/atlas-channel`); `docker buildx
  bake atlas-channel` clean.

## 5. API Surface

No REST/JSON:API changes. The affected interface is the packet wire format and the tenant
`operations` configuration table for the `CharacterStatusMessage` writer:
- New `operations` keys (one per arm / sub-mode) → per-version mode bytes, added to the
  `CharacterStatusMessage` writer in all five seed templates and resolved at emit time via
  `WithResolvedCode("operations", KEY, …)`.

## 6. Data Model

No database entities. The only "data" is tenant configuration:
- `CharacterStatusMessage` writer `options.operations` map (key → mode byte) per tenant
  template, mirroring the `GuildOperation` pattern established in task-103.

## 7. Service Impact

- **libs/atlas-packet** (primary): the 24 `StatusMessage*` structs
  (`character/clientbound/status_message.go` and siblings) take config-injected modes;
  add per-mode body functions + key consts; per-version byte fixtures.
- **tools/packet-audit**: `cmd/run.go` per-arm `#`-entries (remove single representative);
  regenerate audit reports + matrix.
- **docs/packets**: `ida-exports/*` (jms 16-delegate splice + GMS delegate structure +
  inner fan-out functions), `registry/*`, `evidence/*`, `audits/*` (STATUS regen), and a
  new `dispatchers/message.yaml` (per-version mode table, source of truth).
- **services/atlas-channel**: ~8 consumers re-routed through the new body functions
  (no logic change).
- **services/atlas-configurations**: `CharacterStatusMessage` `operations` maps across all
  five seed templates.
- **Live tenant config** (post-deploy): a runbook to PATCH existing tenants' operations
  tables + restart channels, executed after merge/deploy (mirrors task-103 Task 11).

## 8. Non-Functional Requirements

- **Honesty / no false-pass**: a green cell must mean the arm is byte-verified, not that
  the mode byte decodes. Enumeration of mode bytes is not verification.
- **Multi-tenancy**: mode bytes resolve per-tenant from config; version drift handled
  uniformly across all supported versions.
- **Observability**: post-deploy, confirm via channel logs that no `unhandled message op`
  appears for the MESSAGE family and that a representative message renders per version.
- **Path robustness**: run all packet-audit commands from the worktree root; resolve
  generated STATUS.md/status.json conflicts by regeneration, never hand-merge; resolve
  export conflicts by JSON-semantic union.

## 9. Open Questions

These are resolved at execution (design/implementation), not now:
1. Exact per-version outer mode bytes (0x0–0xF) and the inner sub-mode → arm mapping for
   the fan-out handlers (`OnDropPickUpMessage`, `OnQuestRecordMessage`) — IDA enumeration
   (FR-2). Expect version drift like the guild v95 shift.
2. The mode 0xF handler (jms `sub_B0931C`) — resolve its real name/arm from the jms IDB; if
   it does not correspond to an existing Atlas arm, stop-and-ask (do not invent).
3. Which of the 24 arms are version-absent (⬜) on any given version vs must reach ✅.
4. Whether the inner sub-mode (e.g. drop pickup variant byte) should also be config-resolved
   or is a structurally-fixed inner enum — decide per the DISPATCHER_FAMILY uniformity rule
   in design.
5. Live tenant/version set for the post-deploy config patch — determined at execution via
   k8s/Grafana MCP.

## 10. Acceptance Criteria

- [ ] `docs/packets/dispatchers/message.yaml` enumerates all 24 arms with per-version mode
      bytes, IDA-grounded (function + address citations).
- [ ] All 24 `StatusMessage*` arms are discrete, config-driven (no literal mode bytes), each
      with per-version byte fixtures + `// packet-audit:verify` markers + real IDA addresses.
- [ ] `run.go` has one `#`-entry per arm; the single-representative mapping is removed; no
      phantom root representative.
- [ ] jms export carries all 16 per-mode delegate sub-functions (+ inner fan-out) with real
      addresses; GMS exports carry the per-mode delegate structure; no `ida=0x0` / no
      `address: "0x0"` placeholders in this family.
- [ ] Every arm is ✅ across `gms_v83/v84/v87/v95/jms_v185` in STATUS.md (version-absent →
      ⬜); the jms MESSAGE family is no longer ❌.
- [ ] `CharacterStatusMessage` `operations` tables populated/reconciled across all five seed
      templates; the ~8 atlas-channel consumers route through the new config-driven bodies.
- [ ] All four packet-audit gates exit 0; OnMessage scanned clean by `dispatcher-lint` with
      no baseline entry.
- [ ] `go build`/`go vet`/`go test -race` clean in `libs/atlas-packet`, `tools/packet-audit`,
      `services/atlas-channel`; `docker buildx bake atlas-channel` clean.
- [ ] Post-deploy live-config runbook authored (execution gated on merge/deploy + operator
      authorization).
