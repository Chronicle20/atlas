# GMS v84 Tenant Support — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-09
---

## 1. Overview

Atlas is a multi-tenant Go microservices game server. Today it operationally runs a
single GMS **v83.1** tenant. The game/client version is a first-class part of tenant
identity: `tenant.Model` carries `region`, `majorVersion`, and `minorVersion`
(`libs/atlas-tenant/tenant.go:9-14`). Three subsystems key off that version:

1. **Socket protocol** — opcodes, handler/writer bindings, and message-type tables are
   seeded per tenant from version-keyed JSON templates in
   `services/atlas-configurations/seed-data/templates/` (e.g. `template_gms_83_1.json`).
2. **WZ game data** — atlas-data resolves WZ assets from object storage by version path
   `<scope>/regions/<region>/versions/<major>.<minor>/<archive>`
   (`services/atlas-data/.../data/runwz.go:40`).
3. **Version-conditional behavior** — scattered Go branches gate behavior on
   `Region()`/`MajorVersion()` (e.g. `>= 95`, `== 83`, `> 83`, `<= 28`, `<= 12`).

This task adds **GMS v84.1** as a new supported version that runs **alongside** the
existing v83 tenant — not a migration. The end state is a real v84 client that connects
through the login server, selects a character, enters a channel, and plays basic flows
(login → world/channel select → character select → map load → movement/chat).

The principal unknown is the **packet/opcode delta between v83 and v84**. v84 is one
minor GMS revision above v83 (both pre-"Big Bang"), so deltas are expected to be small —
but "expected small" is a hypothesis to be *verified against source*, never assumed.
Opcode resolution will be done by cross-referencing a **partially-named v84 IDA database**
against **both** the v83 IDB (nearest-neighbor naming anchor) and the v95 IDB
(tie-breaker / secondary confirmation). WZ data for v84 is already available to ingest.

## 2. Goals

Primary goals:

- A GMS v84.1 tenant can be created in atlas-tenants and seeded with a complete,
  version-correct socket configuration (handlers, writers, message types) without
  disturbing the existing v83 tenant.
- v84 WZ game data is ingested and served by atlas-data at
  `regions/GMS/versions/84.1/`.
- The v83↔v84 opcode and packet-structure delta is **discovered and documented** from
  IDA source (v84 IDB cross-checked against v83 + v95), not inferred.
- Every existing version-conditional Go branch is audited and given a correct, explicit
  classification for v84 (close the `== 83` / `> 83` boundary gaps).
- A real v84 client completes a basic end-to-end playthrough: login → channel → map →
  movement/chat, with the v83 tenant still functioning.

Non-goals:

- Migrating, removing, or deprecating v83 support.
- Supporting any non-GMS region or any GMS version other than 84.1.
- 100% content/feature parity for v84-exclusive systems beyond what v83 already supports;
  the bar is "basic playthrough works," not "every v84 feature implemented."
- Building new IDA tooling. Use existing IDA-MCP harvest workflows; tooling gaps are
  flagged, not solved here.
- Frontend (atlas-ui) changes, unless tenant-version selection in the UI is found to be a
  hard blocker for creating/operating the v84 tenant (treated as an open question).

## 3. User Stories

- As an **operator**, I want to create a GMS v84.1 tenant and have it seeded with the
  correct socket config automatically, so I can stand up a v84 deployment without
  hand-editing opcode tables.
- As an **operator**, I want the v84 tenant to run alongside the v83 tenant on the same
  cluster, so adding v84 does not put the existing deployment at risk.
- As a **v84 player**, I want my client to connect, select a character, enter a channel,
  and move/chat on a map, so the deployment is actually playable.
- As a **maintainer**, I want a documented v83→v84 packet delta and an audit of every
  version-gated branch, so future version work has a source-of-truth reference and no
  silent boundary bugs.

## 4. Functional Requirements

Organized by capability area. Each requirement is testable.

### 4.1 Opcode & Packet Delta Discovery (verification deliverable)

- **FR-1.1** Produce a documented mapping of v84 **inbound (handler)** opcodes and
  **outbound (writer)** opcodes, derived from the v84 IDB cross-referenced against the
  v83 IDB (primary anchor) and v95 IDB (tie-breaker). Every entry must cite its evidence
  (IDB function name/address or the reference version it was confirmed against).
- **FR-1.2** Identify and document every packet whose **structure/encoding** differs
  between v83 and v84 (field added/removed/reordered, size change, conditional fields).
  For login → channel → map → movement/chat flows this must be exhaustive; for other
  flows, document what was checked and what was assumed.
- **FR-1.3** Where v84 opcodes/packet structures are identical to v83, state so explicitly
  with evidence. "Same as v83" is a finding that must be backed, not a default.
- **FR-1.4** The delta document lives at
  `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` and is the source of
  truth for the template and any Go changes.

### 4.2 Socket Configuration Template

- **FR-2.1** Add `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
  with `region: "GMS"`, `majorVersion: 84`, `minorVersion: 1`, the correct `usesPin`
  value for v84, and complete `socket.handlers` and `socket.writers` arrays reflecting the
  FR-1.x delta.
- **FR-2.2** The template's message-type table (and any other version-keyed socket config)
  must be v84-correct. Inbound handlers must reverse-resolve the same message-type table
  the writers use — never hardcode enum bytes (see the v83 list-selection regression,
  `bug_npc_msgtype_hardcoded_vs_config`).
- **FR-2.3** Seeding the v84 template must be idempotent and must not modify or override the
  existing v83 template/config (the seeder reads `SEED_DATA_PATH`, default `/seed-data`,
  gated by `SEED_ENABLED`; `seeder.go:26-32`).
- **FR-2.4** Handler `validator` and `handler` names, and writer names, in the template must
  resolve to symbols that exist in atlas-channel / atlas-login (no dangling references).

### 4.3 Version-Conditional Code Audit

- **FR-3.1** Enumerate every `Region()/MajorVersion()`-gated branch in `services/` and
  `libs/` and record, for each, what v84 evaluates to today and whether that is correct.
  Known boundary sites to start from (non-exhaustive):
  - `services/atlas-character/.../character/processor.go:1336` —
    `MajorVersion() == 83` gates Beginner/Noblesse/Legend auto-AP assignment. **Exact match
    excludes v84**; the inline TODO says the range is undefined. Decide whether v84 belongs
    in this branch (likely yes) and widen the predicate accordingly.
  - `services/atlas-account/.../account/processor.go:165` —
    `MajorVersion() > 83` sets default gender to `10` (UI-choose). **This fires for v84.**
    Confirm v84 should use UI-choose gender.
  - `character_cash_item_use.go`, `character_attack_common.go:180`,
    `model/damage_taken_info.go:66`, `libs/atlas-packet/chat/serverbound/whisper.go:60,75`
    — `MajorVersion() >= 95` branches (do **not** fire for v84; confirm v84 wants the
    pre-95 path).
  - `login/main.go:277`, `channel/main.go:378` — `MajorVersion() <= 28`;
    `login/session/model.go:35`, `channel/session/model.go:40` — `MajorVersion() <= 12`
    (do not fire for v84; confirm correct).
- **FR-3.2** For every branch where v84's current evaluation is **wrong**, change the
  predicate to an explicit, intention-revealing form (e.g. `>= 83`, `<= 94`, or an explicit
  version range) backed by the FR-1.x findings. Do not change v83 behavior.
- **FR-3.3** Produce a written audit table (branch → file:line → v83 result → v84 result →
  correct? → action) in the delta/audit doc.

### 4.4 WZ Game Data

- **FR-4.1** Ingest the available v84 WZ data into object storage under
  `regions/GMS/versions/84.1/` so atlas-data serves it for a 84.1 tenant.
- **FR-4.2** Verify atlas-data resolves and serves v84 assets end-to-end (e.g. a known
  map/item/mob loads for an 84.1 tenant header) without affecting 83.1 resolution.
- **FR-4.3** Honor the documented atlas-maps spawn-cache caveat
  (`reference_atlas_maps_spawn_cache`): clearing/seeding caches as needed so v84 data is
  actually observed and not masked by stale 83-era cache entries.

### 4.5 Tenant Provisioning & End-to-End

- **FR-5.1** Document the exact provisioning steps for a v84.1 tenant: create tenant
  (region GMS, major 84, minor 1) via atlas-tenants, seed config, ingest WZ, deploy/restart
  the channel + login as needed. (Note the live-config caveat:
  `bug_new_opcodes_not_in_live_tenant_config` — existing tenants don't hot-pick new
  handler/writer opcodes; a fresh v84 tenant seeded from template avoids that, but document
  any restart requirement.)
- **FR-5.2** A real v84 client must complete: connect to login → authenticate →
  world/channel select → character list → enter channel → load starting map → move and
  chat. Each step verified against a running stack (not only unit tests).
- **FR-5.3** Throughout, the existing v83 tenant must remain fully functional (regression
  check on the v83 login→channel→map path).

## 5. API Surface

No new public REST endpoints are anticipated. Surfaces touched:

- **atlas-tenants** — existing `CreateTenantHandler`
  (`services/atlas-tenants/.../tenant/resource.go:59-90`) already accepts
  `region`/`majorVersion`/`minorVersion`; a v84 tenant is created via the existing API with
  `majorVersion=84, minorVersion=1`. No schema change expected.
- **atlas-configurations** — version-keyed template consumed by the existing seeder
  (`seeder.go`); the addition is a new template file, not a new endpoint.
- **atlas-data** — existing version-path resolution (`runwz.go:40`) and the
  `MAJOR_VERSION`/`MINOR_VERSION` (header/env) selection; no new endpoint, only new data
  under the 84.1 path.

If the audit (FR-3) or provisioning (FR-5) surfaces a required API change (e.g. a config
patch endpoint or UI version field), it is captured in Open Questions and escalated rather
than silently added.

## 6. Data Model

- **No new database entities.** v84 reuses the existing tenant + configuration model.
- **Tenant record:** one new row at `(region=GMS, major=84, minor=1)`. `tenant.Model`
  already supports arbitrary major/minor (`uint16`), so no schema migration.
- **Configuration storage:** atlas-configurations seeds the v84 socket config from the new
  template into its generic JSONB configuration store (per-tenant, `tenant_id`-scoped).
- **Object storage (WZ):** new keys under `<scope>/regions/GMS/versions/84.1/...`; additive,
  no migration of existing 83.1 keys.
- **Multi-tenancy:** all new data is tenant/version-scoped; nothing is shared mutable state
  between the v83 and v84 tenants.

## 7. Service Impact

| Service | Change |
|---|---|
| **atlas-configurations** | New `template_gms_84_1.json` seed file (handlers/writers/message types); verify seeder picks it up idempotently. |
| **atlas-data** | Ingest + serve v84 WZ assets at `regions/GMS/versions/84.1/`; verify resolution; manage spawn cache. |
| **atlas-channel** | Audit/adjust `MajorVersion()` branches; apply any v84-specific packet encode/decode deltas in handlers/writers found by FR-1. |
| **atlas-login** | Audit/adjust `MajorVersion()` branches; apply any v84 login-flow packet deltas (login + char-select are in scope). |
| **atlas-character** | `processor.go:1336` `== 83` auto-AP boundary — decide v84 classification and widen predicate. |
| **atlas-account** | `processor.go:165` `> 83` default-gender boundary — confirm v84 behavior. |
| **libs/atlas-packet, libs/atlas-opcodes** | Any version-gated packet structs (e.g. whisper) and opcode registry entries touched by the delta. |
| **atlas-tenants** | Operational only: create the v84 tenant via existing API (no code change expected). |
| **atlas-ui** | Out of scope unless tenant-version selection is a hard blocker (open question). |

## 8. Non-Functional Requirements

- **Multi-tenancy / isolation:** Adding v84 must not alter v83 behavior. All version logic
  must be explicit and version-scoped; no global flags. Regression-verify v83.
- **Correctness over assumption:** Per project rules, packet/opcode/game-data values must be
  verified against IDA source and WZ data, never cited from general MapleStory memory.
  Boundary predicates must be intention-revealing (avoid silent `== 83` exact matches that
  exclude adjacent versions).
- **Observability:** v84 connection/handler failures must surface in logs (watch for
  "unhandled message op 0x.." at info — the symptom of a missing/wrong opcode). Use the
  k8s/Grafana MCP tooling for live diagnosis during the playthrough.
- **Idempotency / safety:** Seeding and WZ ingest must be re-runnable without corrupting
  existing config or data.
- **Build/verification gates:** All changed Go modules pass `go test -race ./...`,
  `go vet ./...`, `go build ./...`; `docker buildx bake atlas-<svc>` for every service whose
  `go.mod` is touched; `tools/redis-key-guard.sh` clean.

## 9. Open Questions

- **OQ-1 (usesPin):** Does GMS v84 use the PIN flow (`usesPin`)? v83 template has
  `usesPin: false`; confirm for v84 from the client.
- **OQ-2 (login parity):** How much does the v84 login-server protocol differ from v83
  (handshake, character-list encoding, world/channel list, character-select/PIC)? Scope says
  verify; this is the biggest correctness risk for "client actually connects."
- **OQ-3 (boundary semantics):** For each `== 83` / `> 83` branch, what is the *intended*
  version range? (e.g. is auto-AP a "≤ pre-Big-Bang" behavior, hence `<= 94`/`<= 83`?)
  Resolve from IDA/behavior, not guesswork.
- **OQ-4 (WZ structural diffs):** Does v84 WZ data have structural differences atlas-data's
  reader doesn't yet handle (cf. prior reader gaps like `consumeOnPickup`, snap-to-ground)?
  Verify a representative map/item/mob/reactor set, not just one asset.
- **OQ-5 (UI blocker):** Does standing up/operating a v84 tenant require any atlas-ui change
  (version field in tenant creation/selection), or is the REST/seed path sufficient?
- **OQ-6 (live-config restart):** What restart/redeploy sequence is required for a freshly
  seeded v84 tenant to be served (channel/login handler+writer load is not hot-reloaded)?
- **OQ-7 (IDB availability):** The v84 IDB is "partially named." Which functions/opcode
  tables are unnamed, and does that block any in-scope flow (login/channel/map/movement/chat)?

## 10. Acceptance Criteria

- [ ] `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` exists and documents
      (a) v84 handler+writer opcode map with evidence, (b) every v83↔v84 packet-structure
      delta for the in-scope flows, (c) the version-branch audit table.
- [ ] `template_gms_84_1.json` exists with v84-correct opcodes, writers, message types, and
      `usesPin`; all handler/validator/writer names resolve to real symbols.
- [ ] Seeding the v84 template is idempotent and leaves the v83 template/config unchanged.
- [ ] v84 WZ data is ingested at `regions/GMS/versions/84.1/` and atlas-data serves a
      representative map/item/mob/reactor for an 84.1 tenant without affecting 83.1.
- [ ] Every `MajorVersion()`-gated branch is audited; boundary predicates touching the 83/84
      edge are corrected with explicit ranges and evidence; v83 behavior is unchanged.
- [ ] A v84.1 tenant is provisioned (tenant + config + WZ) with documented, repeatable steps.
- [ ] A real GMS v84 client completes login → world/channel select → character select →
      enter channel → load map → move + chat, verified against a running stack.
- [ ] The existing v83 tenant passes the same login→channel→map regression check.
- [ ] All changed Go modules pass `go test -race`, `go vet`, `go build`; `docker buildx bake`
      passes for every service with a touched `go.mod`; `tools/redis-key-guard.sh` is clean.
