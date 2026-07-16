# GMS Legacy Versions (48.1 / 61.1 / 72.1 / 79.1) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-30
---

## 1. Overview

Atlas is a multi-tenant Go microservices game server. The client/game version is
first-class tenant identity: `tenant.Model` carries `region`, `majorVersion`, and
`minorVersion`. Today the supported version columns are GMS 83.1, 84.1, 87.1, 95.1 and
JMS 185.1 (see `docs/packets/audits/STATUS.md`). A skeletal `template_gms_12_1.json`
exists but is poorly constructed and is **not** a usable baseline.

This task adds **four new pre-v83 GMS client versions — 48.1, 61.1, 72.1, and 79.1 —** as
fully supported, runnable tenants that operate **alongside** the existing tenants (not a
migration). All four sit *below* the v83 baseline, so their opcode tables, operation/mode
(sub-op) tables, and packet structures are expected to diverge from v83 *more* than v84 did
— and "expected" is a hypothesis to be **verified against the IDB**, never assumed.

Each version is brought up by following the canonical new-version playbook
(`docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md`): build the operation registry,
author a version-correct seed template (opcodes **and** operation/mode tables), produce an
IDA export, run the static audit, add the matrix column, then promote every in-scope cell
through a **comprehensive byte-fixture verification campaign**
(`docs/packets/audits/VERIFYING_A_PACKET.md`). On top of the protocol layer, each version
gets the full task-083-style runtime bring-up: WZ game-data ingestion, version-gated Go
code audit, k8s socket-port wiring, tenant provisioning, and a real-client end-to-end
playthrough — with all existing tenants remaining functional.

Opcode/struct resolution uses the four target IDBs (loaded, "reasonably but not
comprehensively" named), cross-referenced against the **GMS 83.1 IDB** and **GMS 95
IDB/PDB** as naming anchors. IDB functions are unmangled/named as new ones must be
identified to resolve a packet. The four versions are processed as **four sequential
passes within this single task**, anchored in **descending order**: 79→83, 72→79, 61→72,
48→61. Each completed pass becomes the nearest-neighbor anchor for the next-lower version.

## 2. Goals

Primary goals:

- For each of GMS 48.1, 61.1, 72.1, 79.1: a complete, version-correct socket configuration
  template (handlers, writers, message-type table, and operation/mode sub-op tables) seeded
  per tenant, derived from IDB findings — not copied blindly from an adjacent version.
- For each version: an operation registry, an IDA export, a static audit pass, and a new
  coverage-matrix column in `STATUS.md`/`status.json`, with **zero unresolved 🟥 conflict
  cells** in the declared scope.
- For each version: a **comprehensive** byte-fixture verification campaign promoting every
  in-scope matrix cell to ✅ (tier-1) or 🟡-with-evidence (tier-0), or ⬜ where the client
  genuinely lacks the op. No in-scope cell remains ❌ at close.
- For each version: WZ game data ingested and served by atlas-data, every version-gated Go
  branch audited and given an explicit/correct classification, k8s socket ports wired, a
  tenant provisioned, and a real client completing a basic end-to-end playthrough
  (login → world/channel select → character select → enter channel → load map → move + chat).
- The four target IDBs are updated with unmangled/named functions for every packet handler
  and struct identified during the passes; new names cite their anchor evidence.
- All existing tenants (GMS 83/84/87/95, JMS 185) remain fully functional throughout
  (regression-verified).

Non-goals:

- Migrating, removing, or deprecating any existing version.
- Supporting any non-GMS region or any GMS version other than the four named.
- Rehabilitating or using `template_gms_12_1.json` as a baseline (explicitly rejected as a
  poor baseline by the owner).
- Building new packet-audit / IDA tooling. Use the existing `tools/packet-audit` workflow
  and IDA-MCP harvest; tooling gaps are flagged, not solved here (except the minimal
  `--versions` / `matrix` default-set wiring needed to make the four columns first-class).
- 100% content/feature parity for version-exclusive systems beyond what the existing
  codebase already supports; the runtime bar is "basic playthrough works" + "every
  in-scope packet cell verified," not "every legacy feature implemented."
- Frontend (atlas-ui) changes, unless tenant-version selection is found to be a hard
  blocker for creating/operating a target tenant (treated as an open question, OQ-5).

## 3. User Stories

- As an **operator**, I want to create a GMS 48.1 / 61.1 / 72.1 / 79.1 tenant seeded with the
  correct socket config automatically, so I can stand up a legacy-version deployment without
  hand-editing opcode or operation tables.
- As an **operator**, I want each target tenant to run alongside the existing tenants on the
  same cluster, so adding legacy versions does not put existing deployments at risk.
- As a **legacy-version player**, I want my client to connect, select a character, enter a
  channel, and move/chat on a map, so the deployment is actually playable.
- As a **maintainer**, I want a documented per-version packet delta, a verified coverage
  matrix, and an audit of every version-gated branch, so future version work has a
  source-of-truth reference and there are no silent gating bugs.
- As a **maintainer**, I want the four IDBs left with unmangled, named functions for every
  identified handler/struct, so the next person reading them does not re-derive the same
  names.

## 4. Functional Requirements

Organized by capability area. Each requirement is testable. Requirements prefixed **(per
version)** are evaluated independently for each of the four versions; the task closes only
when all four satisfy them.

### 4.1 Opcode, Operation & Packet Delta Discovery (per version)

- **FR-1.1** Produce a documented mapping of each version's **inbound (handler)** opcodes
  and **outbound (writer)** opcodes, derived from the target IDB cross-referenced against
  its descending anchor (79←83, 72←79, 61←72, 48←61) and the GMS 95 IDB/PDB as tie-breaker.
  Every entry cites evidence (IDB function name/address or the anchor version it was
  confirmed against).
- **FR-1.2** Produce a documented mapping of each version's **operation / mode (sub-op)
  tables** — the per-dispatcher mode bytes resolved via `WithResolvedCode("operations", …)`
  (messenger, cashshop, storage, npcshop, interaction, party, buddy, guild, worldmessage,
  status-message, message-type, etc.). These are version-dependent and shift non-uniformly,
  exactly like opcodes (cf. `bug_operations_mode_tables_missing_v87_v95_jms`,
  `bug_v83_status_message_operations_off_by_one`). Each mode value cites its IDB switch-case
  evidence.
- **FR-1.3** Identify and document every packet whose **structure/encoding** differs from
  the anchor version (field added/removed/reordered, size change, conditional/gated fields).
  For the in-scope flows (login → channel → map → movement/chat plus every tier-1 packet)
  this must be exhaustive; for other flows, document what was checked and what was assumed.
- **FR-1.4** Where a version's opcodes / operation tables / packet structures are identical
  to its anchor, state so explicitly with evidence. "Same as anchor" is a finding that must
  be backed, not a default.
- **FR-1.5** Each version's delta document lives at
  `docs/tasks/task-113-gms-legacy-versions/v<major>-packet-delta.md` and is the source of
  truth for that version's registry, template, and any Go changes.

### 4.2 Operation Registry (per version)

- **FR-2.1** Create `docs/packets/registry/gms_v<major>.yaml` via `registry seed` (from the
  ops CSVs where columns exist) then `discover-ops` against the target IDB, following the
  dispatcher-curation checklist in `STARTING_A_NEW_VERSION_PASS.md` §1.1 (include top-level
  `*::OnPacket` opcode dispatchers; exclude body-mode demuxers). Apply with
  `provenance: ida-discovered`; resolve every "Missing at discovery" entry as either a CSV
  transcription fix or a `provenance: manual` entry with an IDA citation.
- **FR-2.2** Commit the per-version discover worklist at
  `docs/packets/registry/discover_gms_v<major>.md`.

### 4.3 Socket Configuration Template (per version)

- **FR-3.1** Add `services/atlas-configurations/seed-data/templates/template_gms_<major>_1.json`
  with `region: "GMS"`, `majorVersion: <major>`, `minorVersion: 1`, the correct `usesPin`
  value (verified from the client; expected `false` for all four pre-v83 versions — confirm),
  and complete `socket.handlers`, `socket.writers`, message-type, and **operations / mode**
  tables reflecting the FR-1.x findings.
- **FR-3.2** Every `socket.handlers` entry MUST carry a `validator` (Pong/StartError →
  `NoOpValidator`, else `LoggedInValidator`/`NoOpValidator` per the IDB-confirmed login
  state). A handler entry without a validator is silently dropped at registration
  (`bug_socket_handler_missing_validator_silently_dropped`) — treat a validator-less handler
  as a defect, not a default.
- **FR-3.3** Inbound handlers must reverse-resolve the same message-type / operation table
  the writers use (`ResolveName` reverse-lookup), never hardcode enum bytes — the legacy
  enums are shifted relative to v83 (cf. `bug_npc_msgtype_hardcoded_vs_config`,
  `bug_v83_status_message_operations_off_by_one`).
- **FR-3.4** Operation/mode tables must be populated per-version from each version's
  dispatcher switch; an absent table makes `ResolveCode` return a sentinel that crashes the
  client (`bug_operations_mode_tables_missing_v87_v95_jms`). Every dispatcher family the
  template routes must have a populated operations table.
- **FR-3.5** Seeding a target template must be idempotent and must NOT modify or override any
  existing version's template/config (seeder reads `SEED_DATA_PATH`, gated by
  `SEED_ENABLED`).
- **FR-3.6** All handler `validator`/`handler` names and writer names in each template must
  resolve to symbols that exist in atlas-channel / atlas-login (no dangling references).

### 4.4 IDA Export & Static Audit (per version)

- **FR-4.1** Produce `docs/packets/ida-exports/gms_v<major>.json` via `packet-audit export`,
  bootstrapping the roster from the descending anchor's export and purging cross-IDB
  coincidentals (functions present only because of a shared binary segment, not actually in
  the target).
- **FR-4.2** Run the static audit (`packet-audit` with the version's template + export)
  producing `docs/packets/audits/gms_v<major>/SUMMARY.md` + per-packet detail files. Run the
  live `validate` / `decompose` / `triage` passes where they add confidence; allowlist
  genuine `missing-mode` cases into `docs/packets/audits/gms_v<major>/_unimplemented.json`
  with justification.

### 4.5 Coverage Matrix & Comprehensive Verification (per version)

- **FR-5.1** Regenerate the matrix (`packet-audit matrix`) so each version appears as a
  first-class column. Wire the four version keys into the `matrix` default `--versions` set
  (and any other hardcoded version lists) so `matrix --check` covers them in CI.
- **FR-5.2** `packet-audit matrix --check` must report **zero** `orphan | dangling | stale |
  drift | unresolv | malformed` lines, and **zero unresolved 🟥 conflict cells** in the
  four new columns, before each pass is considered done. Any 🟥 is resolved via the §5.1
  three-way arbiter (IDB is neutral arbiter; fix registry, template, or Atlas code as the
  IDB dictates) — conflicts may not be allowlisted or deferred.
- **FR-5.3 (comprehensive campaign)** Every in-scope matrix cell for each version is
  promoted to ✅ (tier-1 packets — byte-fixture test with a `packet-audit:verify` marker +
  evidence record), 🟡-with-current-evidence (tier-0), or ⬜ (client genuinely lacks the
  op). **No in-scope cell may remain ❌ at task close.** Fan out with the `packet-verifier`
  / `dispatcher-family-implementer` agents per cell-family, one IDB at a time. Each promoted
  cell commits test + evidence record + regenerated `STATUS.md`/`status.json` together;
  `STATUS.md` is never hand-edited.
- **FR-5.4** Dispatcher-family packets (party/buddy/guild/messenger/cashshop/storage/
  npcshop/interaction/worldmessage/…) follow `docs/packets/DISPATCHER_FAMILY.md`: discrete
  struct per mode, config-resolved mode byte, per-mode body + per-mode byte fixture. A
  mode-byte enumeration without per-arm bodies is a false pass
  (`feedback_dispatcher_mode_byte_is_false_pass`) and does not count toward FR-5.3.

### 4.6 IDB Naming (per version)

- **FR-6.1** Every packet handler/writer and every struct identified while resolving an
  in-scope cell is unmangled and named in the target IDB (demangled `Class::Method` form,
  matching the v83/v95 naming convention). Names cite the anchor evidence used to identify
  them. An unresolved fname is a stop-and-ask, never a fabricated/auto-substituted name
  (`feedback_unresolved_fname_escalate`).
- **FR-6.2** Confirm the correct IDB instance/port before reading via
  `mcp__ida-pro__list_instances` (ports depend on launch order; never hardcode). Use
  `func_query` with `name_regex` per the documented IDA-MCP API.

### 4.7 Version-Conditional Code Audit (spans all versions)

- **FR-7.1** Enumerate every `Region()` / `MajorVersion()`-gated branch in `services/` and
  `libs/` and record, for each, what each of the four versions evaluates to today and whether
  that is correct. Known boundary classes to start from (non-exhaustive): `== 83` exact
  matches (e.g. character auto-AP at `atlas-character/.../processor.go`), `> 83` / `>= 95`
  branches (default gender, whisper, attack-common, damage-taken), and the low-version gates
  `<= 28` / `<= 12` (`login/main.go`, `channel/main.go`, session models) which DO fire for
  some/all of the four legacy versions and must be verified, not assumed.
- **FR-7.2** For every branch where a target version's current evaluation is wrong, change
  the predicate to an explicit, intention-revealing form (explicit version range) backed by
  FR-1.x findings. Do NOT change any existing version's behavior. Beware the systematic
  off-by-one class (`bug_majorversion_gt83_is_off_by_one_v87`): predicates must encode the
  *intended* version range, not a coincidental boundary.
- **FR-7.3** Produce a written audit table (branch → file:line → result-per-version →
  correct? → action) in the task docs.

### 4.8 WZ Game Data (per version)

- **FR-8.1** Ingest each version's WZ data into object storage under
  `regions/GMS/versions/<major>.1/` so atlas-data serves it for that tenant.
- **FR-8.2** Verify atlas-data resolves and serves each version's assets end-to-end (a known
  map/item/mob/reactor loads for the tenant header) without affecting other versions'
  resolution.
- **FR-8.3** Honor the spawn-cache caveat (`reference_atlas_maps_spawn_cache`): clear/seed
  caches so new data is observed and not masked by stale entries. Watch for legacy WZ
  structural differences the reader may not handle (cf. prior reader gaps:
  `consumeOnPickup`, snap-to-ground) — verify a representative asset set, not one asset.
  WZ-data **availability** for each version is OQ-4 (must be sourced before that pass can
  complete its runtime bar).

### 4.9 k8s / Networking (per version)

- **FR-9.1** Wire each version's login/channel socket ports in BOTH places they must agree:
  `deploy/k8s/base/atlas-{login,channel}.yaml` (containerPort + Service.ports) AND the
  login/channel `services` config — per the convention `<major>×100` (channel +1)
  (`bug_new_version_lb_socket_ports`). A port wired in only one place silently fails to
  route.

### 4.10 Tenant Provisioning & End-to-End (per version)

- **FR-10.1** Document the exact provisioning steps for each tenant: create tenant
  (region GMS, major, minor 1) via atlas-tenants, seed config, ingest WZ, wire ports,
  deploy/restart channel + login. Note the live-config caveat
  (`bug_new_opcodes_not_in_live_tenant_config`): a freshly seeded tenant avoids it, but
  document any restart sequence (handler/writer projections do not hot-reload).
- **FR-10.2** A real client for each version must complete: connect to login → authenticate
  → world/channel select → character list → enter channel → load starting map → move and
  chat. Each step verified against a running stack (not only unit tests). Watch for the
  config-projection fatal-on-miss class for factory/world
  (`bug_factory_world_config_load_once_fatalf`).
- **FR-10.3** Throughout, every existing tenant (GMS 83/84/87/95, JMS 185) must remain fully
  functional (regression check on at least the v83 login→channel→map path).

## 5. API Surface

No new public REST endpoints are anticipated. Surfaces touched:

- **atlas-tenants** — existing `CreateTenantHandler` already accepts
  `region`/`majorVersion`/`minorVersion`; each tenant is created via the existing API. No
  schema change expected.
- **atlas-configurations** — version-keyed templates consumed by the existing seeder; the
  addition is four new template files, not a new endpoint.
- **atlas-data** — existing version-path resolution and `MAJOR_VERSION`/`MINOR_VERSION`
  selection; no new endpoint, only new data under the four version paths.
- **tools/packet-audit** — internal CLI: extend the default `--versions` set (and any
  hardcoded version lists) to include the four new keys so `matrix`/`matrix --check` cover
  them. Not a public API.

If the code audit (FR-7), WZ work (FR-8), or provisioning (FR-10) surfaces a required API
change, it is captured in Open Questions and escalated rather than silently added.

## 6. Data Model

- **No new database entities.** Each version reuses the existing tenant + configuration
  model.
- **Tenant records:** four new rows at `(region=GMS, major∈{48,61,72,79}, minor=1)`.
  `tenant.Model` already supports arbitrary major/minor (`uint16`); no schema migration.
- **Configuration storage:** atlas-configurations seeds each version's socket config from its
  template into the generic JSONB configuration store (per-tenant, `tenant_id`-scoped).
- **Object storage (WZ):** new keys under `<scope>/regions/GMS/versions/<major>.1/...`;
  additive, no migration of existing keys.
- **Packet-audit artifacts (docs, not DB):** per-version registry YAML, IDA export JSON,
  audit dir, evidence ledger entries, and matrix columns in `STATUS.md`/`status.json`.
- **Multi-tenancy:** all new data is tenant/version-scoped; nothing is shared mutable state
  between tenants.

## 7. Service Impact

| Service | Change |
|---|---|
| **atlas-configurations** | Four new `template_gms_<major>_1.json` seed files (handlers/writers/message types/operations); verify idempotent pickup. |
| **atlas-data** | Ingest + serve each version's WZ assets at `regions/GMS/versions/<major>.1/`; verify resolution; manage spawn cache. |
| **atlas-channel** | Audit/adjust `MajorVersion()` branches; apply per-version packet encode/decode + operation-table deltas in handlers/writers; new socket port wiring. |
| **atlas-login** | Audit/adjust `MajorVersion()` branches; apply per-version login-flow packet deltas; new socket port wiring; verify low-version gates (`<=28`, `<=12`). |
| **atlas-character** | `== 83` auto-AP boundary and any other exact-match gates — classify all four versions and widen predicates. |
| **atlas-account** | `> 83` default-gender boundary and similar — confirm per-version behavior. |
| **libs/atlas-packet, libs/atlas-opcodes** | Version-gated packet structs (whisper, attack, damage-taken, etc.) and opcode/operation registry entries touched by the deltas. |
| **tools/packet-audit** | Extend default `--versions` set / hardcoded version lists to include the four keys. |
| **deploy/k8s/base** | `atlas-{login,channel}.yaml` containerPort + Service.ports for each version. |
| **atlas-tenants** | Operational only: create the four tenants via existing API (no code change expected). |
| **atlas-ui** | Out of scope unless tenant-version selection is a hard blocker (OQ-5). |
| **The four target IDBs** | Unmangle/name handler + struct functions identified during each pass (external to repo; tracked in the delta docs). |

## 8. Non-Functional Requirements

- **Multi-tenancy / isolation:** Adding the four versions must not alter any existing
  version's behavior. All version logic must be explicit and version-scoped; no global flags.
  Regression-verify at least v83.
- **Correctness over assumption:** Per project rules, packet/opcode/operation/game-data
  values must be verified against IDA source and WZ data, never cited from general
  MapleStory memory. Boundary predicates must be intention-revealing (avoid silent `== 83`
  exact matches and coincidental boundaries).
- **No false passes:** A cell is verified only with byte-level evidence per
  `VERIFYING_A_PACKET.md`; dispatcher families need per-mode bodies + per-mode fixtures, not
  mode-byte enumeration. Spot-checks presented as full sweeps are rejected.
- **Observability:** connection/handler failures must surface in logs (watch for "unhandled
  message op 0x.." at info — the symptom of a missing/wrong opcode, and
  `ResolveCode→sentinel` crashes from a missing operations table). Use the k8s/Grafana MCP
  tooling for live diagnosis during the playthrough.
- **Idempotency / safety:** Seeding, WZ ingest, and registry/export generation must be
  re-runnable without corrupting existing config, data, or committed artifacts. The IDA
  export splice is surgical/non-idempotent — never overwrite an existing export wholesale
  (`reference_packet_audit_serverbound_verification`).
- **Build/verification gates:** All changed Go modules pass `go test -race ./...`,
  `go vet ./...`, `go build ./...`; `docker buildx bake atlas-<svc>` for every service whose
  `go.mod` is touched; `tools/redis-key-guard.sh` clean (run with `GOWORK=off`).

## 9. Open Questions

- **OQ-1 (IDB ports & naming completeness):** What are the loaded ports for the four target
  IDBs (confirm via `list_instances`), and which handler/opcode/operation functions are
  currently unnamed? Does any unnamed region block an in-scope flow? (Naming the blockers is
  in scope per FR-6, not a deferral.)
- **OQ-2 (usesPin):** Do any of GMS 48/61/72/79 use the PIN flow? Expected `false` for all
  (pre-v83); confirm per version from the client.
- **OQ-3 (login-flow divergence):** How much does each version's login-server protocol differ
  from its anchor (handshake, character-list encoding, world/channel list, character-select)?
  This is the biggest correctness risk for "client actually connects," and the legacy gap
  from v83 is larger than v84's was.
- **OQ-4 (WZ data availability):** Is WZ data available for all four versions, and where is it
  sourced? Runtime bring-up (FR-8/FR-10) cannot complete for a version whose WZ data cannot
  be obtained — flag any unavailable version as a blocker for *that pass's* runtime bar (its
  protocol/packet bar can still complete).
- **OQ-5 (UI blocker):** Does standing up/operating these tenants require any atlas-ui change
  (version field in tenant creation/selection), or is the REST/seed path sufficient?
- **OQ-6 (low-version gate semantics):** What is the intended behavior of the `<= 28` and
  `<= 12` gates for each of 48/61/72/79? These fire (or nearly fire) for legacy versions and
  must be resolved from IDA/behavior, not guesswork.
- **OQ-7 (operation-table extraction effort):** How many dispatcher families have
  version-shifted operation/mode tables across these four versions, and is per-version
  extraction from each dispatcher switch tractable within the campaign? (Drives campaign
  sizing for FR-1.2 / FR-3.4 / FR-5.4.)
- **OQ-8 (scope-cell enumeration):** The comprehensive campaign requires a concrete per-cell
  scope list (operation × direction × version). The exact tier-1/tier-0 cell set per version
  is produced during the registry/matrix step of each pass and recorded as that pass's
  acceptance checklist (see §10).

## 10. Acceptance Criteria

The task closes only when **all four versions** satisfy the criteria below. Each pass
(79 → 72 → 61 → 48) is gated by its own copy of this checklist; a later (lower) version's
pass uses the just-completed higher version as its anchor.

Per version `<major> ∈ {79, 72, 61, 48}`:

- [ ] `docs/tasks/task-113-gms-legacy-versions/v<major>-packet-delta.md` exists documenting
      (a) handler+writer opcode map with evidence, (b) operation/mode-table map with
      switch-case evidence, (c) every packet-structure delta vs. the anchor for in-scope
      flows, (d) the version-branch audit rows relevant to this version.
- [ ] `docs/packets/registry/gms_v<major>.yaml` + `discover_gms_v<major>.md` committed; every
      registry op has `provenance` and (for manual entries) an IDA citation.
- [ ] `services/atlas-configurations/seed-data/templates/template_gms_<major>_1.json` exists
      with correct opcodes, writers, message-type + operations tables, and `usesPin`; every
      handler has a validator; all handler/validator/writer names resolve to real symbols.
- [ ] `docs/packets/ida-exports/gms_v<major>.json` + `docs/packets/audits/gms_v<major>/`
      (SUMMARY + per-packet) committed; `_unimplemented.json` (if any) justified.
- [ ] `gms_v<major>` appears as a matrix column; `packet-audit matrix --check` reports zero
      `orphan|dangling|stale|drift|unresolv|malformed` lines and zero unresolved 🟥 in the
      column.
- [ ] **Comprehensive campaign:** every in-scope cell for `gms_v<major>` is ✅ (tier-1,
      byte-fixture + `packet-audit:verify` marker + evidence) or 🟡-with-current-evidence
      (tier-0) or ⬜ (op absent). No in-scope cell is ❌. Dispatcher families verified
      per-mode (body + fixture), not by mode-byte enumeration.
- [ ] The target IDB has every identified handler/struct unmangled + named, citing anchor
      evidence; no fabricated/auto-substituted fnames.
- [ ] WZ data ingested at `regions/GMS/versions/<major>.1/`; atlas-data serves a
      representative map/item/mob/reactor for the tenant without affecting other versions.
      (If WZ data is unavailable — OQ-4 — this and the playthrough criterion are explicitly
      carved out for that version with owner sign-off, and the protocol criteria still hold.)
- [ ] Login/channel socket ports wired in both `deploy/k8s/base/atlas-{login,channel}.yaml`
      and the login/channel `services` config (`<major>×100`, channel +1).
- [ ] A tenant is provisioned (tenant + config + WZ + ports) with documented, repeatable
      steps; a real client completes login → world/channel select → character select → enter
      channel → load map → move + chat against a running stack.

Spanning all versions:

- [ ] Every `Region()/MajorVersion()`-gated branch is audited with an explicit per-version
      classification; boundary predicates touching the legacy range are corrected with
      explicit version ranges + evidence; no existing version's behavior changes (audit table
      committed in task docs).
- [ ] `tools/packet-audit` default `--versions` set includes all four keys so CI covers them.
- [ ] Existing tenants (at least v83) pass the login→channel→map regression check.
- [ ] All changed Go modules pass `go test -race`, `go vet`, `go build`; `docker buildx bake`
      passes for every service with a touched `go.mod`; `tools/redis-key-guard.sh` clean.
