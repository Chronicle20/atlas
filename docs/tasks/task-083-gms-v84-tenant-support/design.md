# GMS v84 Tenant Support — Design

Status: Approved-pending-review
Created: 2026-06-09
PRD: `docs/tasks/task-083-gms-v84-tenant-support/prd.md`

---

## 1. Summary

Add **GMS v84.1** as a new supported tenant version running **alongside** the
existing v83.1 tenant. There is **no new database entity, no new service, and no
new REST endpoint** — v84 reuses the existing tenant + configuration + WZ model.
The work is therefore four cooperating efforts plus a final live milestone:

1. **Discovery** — full v84 opcode-table dump from the v84 IDB, diffed against
   v83 (primary anchor) and v95 (tie-breaker), producing the source-of-truth
   delta document.
2. **Code audit + version helpers** — introduce range/region predicate helpers
   on `tenant.Model`, then audit every version-gated branch and migrate/correct
   the touched sites so v84 is explicitly classified. v83 behavior must not change.
3. **Configuration** — a new `template_gms_84_1.json` seed file (socket
   handlers/writers/message types + version header), authored from v83 plus the
   discovered delta.
4. **WZ data** — ingest v84 WZ into object storage at `regions/GMS/versions/84.1/`
   via the existing operational ingest Job; verify atlas-data serves it.
5. **Provisioning + live E2E** — a repeatable runbook to stand up the v84 tenant,
   then a real v84 client completing login → channel → map → move/chat, with v83
   regression-verified. **Per the approved decision, the task is not "done" until
   this live playthrough passes.**

### Approved architecture decisions (the three forks)

| Decision | Choice | Consequence |
|---|---|---|
| Boundary predicates | **Add `tenant.Model` version helpers** and migrate touched sites | New helpers in `libs/atlas-tenant`; touched sites read as ranges, not raw inequalities. Scope of migration is bounded (§5). |
| Delta discovery depth | **Full v84 opcode-table dump + diff** | Both directions, all opcodes harvested from the IDB and diffed against v83 — not just in-scope flows. Packet *structure* analysis stays exhaustive for in-scope flows, spot-checked elsewhere (§3.3). |
| Live E2E | **Block done on live playthrough** | Task completion is coupled to v84-client + running-cluster availability. The plan ends on a verification milestone, not a merge. |

---

## 2. Existing Architecture (verified, not assumed)

- **`tenant.Model`** (`libs/atlas-tenant/tenant.go:10-15`) carries `region string`,
  `majorVersion uint16`, `minorVersion uint16` with getters only (immutable). No
  comparison helpers exist today; every call site hand-rolls `MajorVersion() op N`.
- **Socket config templates** live at
  `services/atlas-configurations/seed-data/templates/template_gms_<M>_<m>.json`.
  Existing GMS templates: 12, 83, 87, 92, 95 (plus JMS 185). **v83 is by far the
  most complete** (93 handlers / 112 writers); 87/92/95 are partial (~31–34
  handlers). Template shape:
  - `region`, `majorVersion`, `minorVersion`, `usesPin` (all GMS = `false` today)
  - `socket.handlers[] = {opCode, validator, handler}`
  - `socket.writers[]  = {opCode, writer}`
  - non-socket: `characters{templates,presets}`, `npcs[]`, `worlds[]`, `cashShop{}`
- **Seeder** (`seeder/seeder.go`) discovers every `*.json` under
  `<SEED_DATA_PATH>/templates`, and **skips any whose `(region, major, minor)`
  already exists** (`importTemplate` → `templateExists`, line 218-228). Adding a
  new file is therefore idempotent and physically cannot mutate the v83 row —
  **FR-2.3 holds by construction.** Templates are version-keyed records; tenant
  creation copies the matching template into the per-tenant config store.
- **WZ resolution** (`data/runwz.go:40`) keys off
  `<scope>/regions/<region>/versions/<major>.<minor>/<archive>`.
- **WZ ingest is operational, not code** (`runtime/ingest/run.go`): a k8s Job pod
  (`MODE=ingest`) reads `SCOPE/REGION/MAJOR_VERSION/MINOR_VERSION` env and runs
  the worker fan-out. The REST pod's `JobCreator` renders that Job from the
  `atlas-data-ingest-job-template` ConfigMap. So v84 ingest = upload archives to
  the 84.1 MinIO path + trigger the ingest Job for 84.1. **No atlas-data code
  change.**
- **Version gates already anticipate a 83–95 version.** The audit (§5) found ~42
  unique predicates / ~410 sites. Many ranges *already* classify v84 correctly by
  accident of how they were written (`>28 && <=87` → monster book present in v84;
  `<=87` → buddy job-level present; `>83` → UI-choose gender). These must be
  **confirmed against the delta, not trusted**. The one unambiguous bug is the
  exact match `Region()=="GMS" && MajorVersion()==83` at
  `services/atlas-character/.../character/processor.go:1336` (auto-AP), which
  silently excludes v84 and carries an inline TODO admitting the range is undefined.

---

## 3. Component A — Opcode & Packet Delta Discovery

**Output:** `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md`
(FR-1.4) — the single source of truth that feeds Components B and C.

### 3.1 IDA harvest method (full dump)

Use the existing IDA-MCP workflow (`reference_ida_mcp_new_api`,
`reference_ida_harvest_subagents`). Only one IDB is loaded per instance;
`select_instance(port)` switches between the v83 / v84 / v95 IDBs. Heed the
real-decompile gaps (`project_packet_audit_exporter_real_decompile_gaps`): the
v84 IDB is **partially named** (OQ-7), so expect unnamed `sub_XXXX` dispatch and
alias sets.

Procedure:
1. **Locate the dispatch tables in each IDB.** Inbound = the client's
   recv/`ProcessPacket` opcode switch (server→client writers, from the client's
   POV these are the packets it *parses*). Outbound = the client's send sites
   (client→server handlers). Anchor on the v83 IDB where naming is densest.
2. **Harvest opcode → function** for both directions in the **v84** IDB. For each
   v84 opcode, name-anchor by structural/byte similarity against the v83 function
   at the same logical slot; use v95 as a tie-breaker when v83 is ambiguous or the
   slot shifted.
3. **Diff against v83.** Produce, for every opcode value, one of:
   `SAME` (v84 == v83, with evidence), `SHIFTED` (same handler, different opcode
   value), `ADDED` (v84-only), `REMOVED` (v83-only, absent in v84).

### 3.2 Opcode map deliverable (FR-1.1, FR-1.3)

Two tables — **inbound (handlers)** and **outbound (writers)** — each row:

| logical name | v83 opcode | v84 opcode | classification | evidence (IDB fn/addr or ref version) |

"Same as v83" is a row with cited evidence, never a default (FR-1.3).

### 3.3 Packet-structure delta (FR-1.2)

Full *opcode* dump is feasible; full *structure* diff of every packet is not, and
is not warranted by the "basic playthrough" bar. So:
- **Exhaustive** structure analysis for the in-scope flows: login handshake,
  auth, world/channel list, character list, character select / PIC-PIN,
  enter-channel, map load (spawn/field), movement, chat. Each field
  added/removed/reordered/resized/conditional is documented with IDA evidence.
- **Spot-checked** elsewhere: record what was checked and what was assumed
  (FR-1.2), so future work knows the confidence boundary.

### 3.4 Boundary cross-reference

The delta doc's findings are the *evidence column* for the Component B audit
table. Every predicate classification in §5 must point back to a delta-doc
finding (or to "no packet/behavior difference observed").

---

## 4. Component C — Socket Configuration Template

**Output:** `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
(FR-2.1).

- **Author from v83.** Copy `template_gms_83_1.json` (the complete one), set
  `majorVersion: 84`. Apply only the socket deltas from Component A:
  - `SAME` opcodes → unchanged.
  - `SHIFTED` → change the `opCode` value, keep the handler/validator/writer name.
  - `ADDED` → new entry referencing an existing symbol (see below).
  - `REMOVED` → drop the entry.
- **`usesPin`** (OQ-1): verify from the v84 client login flow during Component A.
  Default expectation `false` (every GMS template is false), but this is a finding
  to confirm, not assume.
- **Symbol-resolution gate (FR-2.4):** every `validator`, `handler`, and `writer`
  name in the template must resolve to a registered symbol in atlas-channel /
  atlas-login. A validation step (script or test) cross-checks template names
  against the registries; **dangling names fail the gate**. If Component A surfaces
  a v84 packet with no existing handler/writer symbol, that is escalated (it means
  a Go change, not just config) rather than silently referencing a non-existent name.
- **Message-type table (FR-2.2):** the v84 message-type table must be v84-correct,
  and inbound handlers must **reverse-resolve the same table the writers use**
  (the v83 list-selection regression, `bug_npc_msgtype_hardcoded_vs_config` /
  `bug_new_opcodes_not_in_live_tenant_config`). If v84's message-type enum differs
  from v83, the delta doc captures it and the template encodes it; no enum bytes
  are hardcoded in Go.
- **Non-socket sections** (`characters`, `npcs`, `worlds`, `cashShop`): copy from
  v83 (v84 content parity is close enough for basic playthrough). `worlds` is
  operator-tunable deployment config; copying v83 values is the safe default.

---

## 5. Component B — Version Helpers + Code Audit

### 5.1 New predicate helpers (`libs/atlas-tenant/tenant.go`)

Add pure, behavior-preserving methods on `*Model`:

```go
func (m *Model) IsRegion(region string) bool   { return m.region == region }
func (m *Model) MajorAtLeast(v uint16) bool     { return m.majorVersion >= v }
func (m *Model) MajorAtMost(v uint16) bool      { return m.majorVersion <= v }
func (m *Model) MajorInRange(lo, hi uint16) bool { return m.majorVersion >= lo && m.majorVersion <= hi } // inclusive
```

Rationale for *range/comparison* helpers (the approved option) rather than named
capability flags: they are intention-revealing at the call site
(`m.IsRegion("GMS") && m.MajorInRange(28, 87)`), require no central capability
registry, and each migration is a provable 1:1 behavior-preserving rewrite.
Region stays a separate predicate so it composes with any version range and the
lib stays region-agnostic. Where a range encodes a known capability, an inline
comment names it (e.g. `// monster book: GMS 28..87`) — we do **not** bake
capability names into method identities (that was the rejected option).

Each helper gets unit tests (TDD) proving the boundary values (83, 84, and the
relevant edge like 28/87/95) evaluate identically to the inequality it replaces.

### 5.2 Migration scope (bounded — not all 410 sites)

Migrate to helpers, and correct for v84, **only**:
- (a) every predicate the audit finds **wrong for v84** (must change anyway), and
- (b) every predicate on an **in-scope flow** (login/channel/map/movement/chat)
  that we verify against the delta.

Out-of-scope-flow predicates that already evaluate correctly for v84 are recorded
in the audit table as "correct, unchanged" and **left as-is** — migrating all 410
would be unrelated refactoring and needless v83-regression surface. (If a future
task wants a blanket migration, the helpers will already exist.)

### 5.3 Known correction sites (from the enumeration; confirm each vs. delta)

| Site | Current predicate | v84 today | Action |
|---|---|---|---|
| `atlas-character/.../character/processor.go:1336` | `Region()=="GMS" && MajorVersion()==83` | **excluded (bug)** | Confirm auto-AP is a pre-Big-Bang behavior; widen to the verified range (e.g. `IsRegion("GMS") && MajorInRange(83,94)` or `MajorAtMost(94)`), backed by IDA/behavior (OQ-3). Must not change v83 result. |
| `atlas-account/.../account/processor.go:165` | `Region()=="GMS" && MajorVersion()>83` | gender=10 (UI-choose) fires | Confirm v84 wants UI-choose; if yes, migrate to `MajorAtLeast(84)` (identical), keep v83=Male. |
| `libs/atlas-packet/character/data.go` monster-book (`>28 && <=87`) | present for v84 | confirm v84 encodes monster book → keep range as `MajorInRange(28,87)` |
| `libs/atlas-packet/buddy/clientbound/invite.go` (`<=87`) | job-level present for v84 | confirm vs delta → `MajorAtMost(87)` |
| `>=95` family (cash-item-use, whisper, damage-taken, spawn) | not fired for v84 | confirm v84 wants pre-95 path → `MajorAtLeast(95)` |
| `<=28` / `<=12` families (login/channel init, crypto) | not fired for v84 | confirm correct → `MajorAtMost(28)` / `MajorAtMost(12)` |

### 5.4 Audit table deliverable (FR-3.3)

In `v84-packet-delta.md`: **branch → file:line → predicate → v83 result → v84
result → correct? → action → delta evidence**, covering every
`Region()/MajorVersion()/MinorVersion()`-gated site found. This is the
acceptance artifact for FR-3.1.

---

## 6. Component D — WZ Game Data

- **Ingest (FR-4.1):** place the available v84 WZ archives in the WZ bucket at
  `<scope>/regions/GMS/versions/84.1/<archive>`, then trigger the existing ingest
  Job for `(GMS, 84, 1)` (REST `JobCreator` path; sets `MAJOR_VERSION=84
  MINOR_VERSION=1`). Additive — no 83.1 key is touched (FR-4.2 isolation).
- **Verify (FR-4.2):** with an 84.1 tenant header, `GET` a representative
  map/item/mob/reactor from atlas-data and confirm it resolves from the 84.1 path;
  confirm an 83.1 header still resolves 83.1.
- **Structural-reader risk (OQ-4):** v84 WZ may have structural shapes the reader
  doesn't handle (cf. prior gaps: `consumeOnPickup`, snap-to-ground). Verify a
  *representative set* (a map, an item, a mob, a reactor), not one asset. Any
  reader gap is a finding; small fixes land here, large ones escalate.
- **Spawn cache (FR-4.3):** honor `reference_atlas_maps_spawn_cache` — after v84
  ingest, `DEL atlas:maps:spawn:*` (and clear affected monsters) so the v84 data
  is observed and not masked by stale 83-era cache. Document this in the runbook.

---

## 7. Component E — Provisioning Runbook + Live E2E

**Output:** a `provisioning.md` runbook section (or appended to the delta doc) and
the live verification result.

### 7.1 Provisioning steps (FR-5.1)

1. Ensure `template_gms_84_1.json` is deployed and seeded (seeder idempotent;
   skips if `(GMS,84,1)` template already present).
2. Upload v84 WZ to `regions/GMS/versions/84.1/`; run the ingest Job; clear spawn
   cache.
3. Create the tenant `(region=GMS, major=84, minor=1)` via the existing
   atlas-tenants `CreateTenantHandler` (no schema change).
4. **Restart sequence (OQ-6):** handler/writer binding is **not** hot-reloaded
   (`bug_new_opcodes_not_in_live_tenant_config`). A *freshly created* v84 tenant
   seeds its config at creation, but document and verify whether channel/login
   pods must restart to load the v84 tenant's socket bindings; capture the exact
   sequence that works.

### 7.2 Live playthrough (FR-5.2 — the blocking gate)

A real GMS v84 client completes, against a running stack:
connect → authenticate → world/channel select → character list → enter channel →
load starting map → move + chat. Diagnose failures live via the k8s/Grafana MCP
tooling (`reference_observability`); the canonical failure signature is
`unhandled message op 0xXX` at info, meaning a missing/wrong opcode in the
template — fix in the delta/template loop, not by guessing.

### 7.3 v83 regression (FR-5.3)

Re-run the same login→channel→map path on the existing v83 tenant after all
changes. v83 must be unchanged. The version-helper migration's unit tests (§5.1)
are the first line of defense; the live v83 path is the confirmation.

---

## 8. Data Flow

```
v84 IDB ─┐
v83 IDB ─┼─(IDA-MCP harvest+diff)─▶ v84-packet-delta.md ──┬─▶ template_gms_84_1.json ─▶ seeder ─▶ per-tenant config
v95 IDB ─┘                          (opcode map +        │     (symbol-resolution gate)
                                     structure delta +    │
                                     audit table)         └─▶ Go: tenant helpers + corrected predicates
v84 WZ archives ─▶ MinIO 84.1/ ─▶ ingest Job ─▶ atlas-data serves 84.1 (spawn cache cleared)
                                                                  │
   tenant (GMS,84,1) ──────────────────────────────────────────▶ running stack ─▶ live v84 client E2E + v83 regression
```

Hard dependency order: **A (delta) precedes B (code) and C (template)**; B+C+D
are independent of each other; **E requires A+B+C+D all landed**.

---

## 9. Error Handling, Isolation & Risk

- **Isolation (NFR):** every artifact is version/tenant-scoped. The seeder cannot
  overwrite v83; WZ keys are additive; new helpers are pure and don't alter v83
  call sites unless a site is explicitly migrated with a behavior-preserving test.
  No global flags.
- **Dangling template symbols:** caught by the FR-2.4 resolution gate before any
  live attempt.
- **Partially-named v84 IDB (OQ-7):** if an in-scope opcode's v84 function is
  unnamed and can't be disambiguated against v83/v95, that opcode is flagged as
  low-confidence in the delta doc and is the first suspect during live E2E.
- **WZ reader gaps (OQ-4):** representative-set verification surfaces them before
  E2E rather than as a mid-playthrough crash.
- **Live-config / restart (OQ-6):** documented restart sequence prevents the
  "client action no-ops" silent-drop class.
- **Build gates:** all changed Go modules pass `go test -race`, `go vet`,
  `go build`; `docker buildx bake atlas-<svc>` for every touched `go.mod`
  (`libs/atlas-tenant` is consumed by many services — expect multiple bake
  targets); `tools/redis-key-guard.sh` clean.

## 10. Testing Strategy

- **Unit (TDD):** the four `tenant.Model` helpers — boundary values 12/28/83/84/87/94/95.
- **Behavior-preservation:** each migrated predicate site keeps/gets a test that
  the v83 evaluation is unchanged and the v84 evaluation is the audited-correct value.
- **Template validation:** automated symbol-resolution check (names → registries);
  JSON parses into `templates.RestModel`; `(GMS,84,1)` distinct from `(GMS,83,1)`.
- **Seeder idempotency:** re-run leaves v83 untouched and skips an already-seeded
  v84 (existing seeder behavior; assert via test if not already covered).
- **Data resolution:** atlas-data serves a representative 84.1 asset set; 83.1 unaffected.
- **Live E2E (blocking):** the §7.2 playthrough + §7.3 v83 regression on a running stack.

## 11. Open-Question Resolution Plan

| OQ | Resolved by |
|---|---|
| OQ-1 usesPin | Component A login-flow analysis; encoded in template (§4). |
| OQ-2 login parity | Component A exhaustive structure delta for login flow (§3.3). |
| OQ-3 boundary semantics | Component A behavior evidence drives each §5.3 range. |
| OQ-4 WZ structural diffs | Component D representative-set verification (§6). |
| OQ-5 UI blocker | Confirm REST/seed provisioning path is sufficient (§7.1); escalate only if a UI version field is a hard blocker (PRD non-goal otherwise). |
| OQ-6 live-config restart | Component E documents the working restart sequence (§7.1). |
| OQ-7 IDB naming gaps | Flagged per-opcode in the delta doc; low-confidence opcodes are first E2E suspects (§9). |

## 12. Scope Boundaries (YAGNI)

- No new entity, service, or REST endpoint.
- No blanket migration of all 410 version sites — only audit-touched sites (§5.2).
- No new IDA tooling — use existing harvest workflows; tooling gaps are flagged.
- No atlas-ui change unless a hard provisioning blocker (OQ-5).
- No v83 migration/deprecation; no non-GMS region; no GMS version other than 84.1.
