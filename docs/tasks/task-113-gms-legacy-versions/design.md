# GMS Legacy Versions (48.1 / 61.1 / 72.1 / 79.1) — Design

Status: Approved-for-planning
Created: 2026-06-30
PRD: `docs/tasks/task-113-gms-legacy-versions/prd.md`

---

## 1. Purpose & scope of this document

This design fixes the **architecture, sequencing, and methodology** for bringing up four
new pre-v83 GMS tenant versions. It does not re-derive the requirements (the PRD owns
those); it answers *how* the four passes are structured, *where* artifacts live, *which*
strategy resolves the two hardest sub-problems (operation/mode-table extraction and the
cross-version code-gate audit), and *how* agent fan-out is organized for the verification
campaign. The plan (`/plan-task`) turns this into an ordered, checkpointed task list.

## 2. Decisions locked for this design

Two architectural forks were resolved with the owner before writing:

- **Execution shape: vertical per-version.** Each version completes its full
  protocol → code-audit slice → runtime bring-up before the next (lower) version starts.
  The just-completed higher version is the nearest-neighbor anchor for the next. The
  FR-7 code-gate audit *spans* all versions, so it is handled by an **accumulating audit
  table** (each pass adds its rows and fixes only the gates that affect that version) plus
  a **final reconciliation pass** at task close — not by front-loading a separate phase.
- **WZ data: assumed available for all four.** WZ ingestion + serve-verification
  (FR-8/FR-10) is a firm deliverable for every version, not a conditional one. OQ-4's
  "carve-out with owner sign-off" path is retained only as a contingency if a specific
  version's data turns out to be unobtainable during its pass.

Everything else follows the PRD as written.

## 3. The two-layer mental model

Each version is brought up across two stacked layers. The design keeps them conceptually
separate because they have different failure modes, different tools, and different
"done" signals — but within a vertical pass they execute back-to-back for one version.

```
  Protocol layer  (what the wire looks like)         Runtime layer (does it actually run)
  ─────────────────────────────────────────          ───────────────────────────────────
  registry YAML  ── discover-ops ──┐                  WZ ingest → atlas-data serve
  seed template  ── opcodes+ops ───┤                  k8s socket ports (2 synced places)
  IDA export     ── harvest ───────┼─→ matrix ──→     tenant provision (region/major/minor)
  static audit   ── SUMMARY ───────┤   verify         channel/login deploy + restart
  byte-fixtures  ── packet-audit ──┘   campaign       real-client playthrough
                                                       code-gate audit slice (FR-7)
```

The descending **anchor chain (79←83, 72←79, 61←72, 48←61)** lives entirely in the
protocol layer: each version's registry/template/export is *derived from* the
just-completed higher version's, then **verified against the target IDB** — never copied
and trusted. "Same as anchor" is a finding requiring switch-case evidence (FR-1.4), not a
default. The GMS 95 IDB/PDB is the tie-breaker when the descending anchor is ambiguous.

## 4. The repeatable pass (applied 79 → 72 → 61 → 48)

Every pass runs the same nine stages. This is the unit of work the plan will template
four times. Stages A–E are protocol; F is the code-gate slice; G–I are runtime.

| # | Stage | Primary playbook | Key output | Gate to advance |
|---|-------|------------------|------------|-----------------|
| A | **Anchor & delta discovery** | PRD §4.1, IDA-MCP | `v<major>-packet-delta.md` (opcode map, operation/mode map, struct deltas, evidence) | Every in-scope opcode + mode cites IDB or anchor evidence |
| B | **Operation registry** | `STARTING_A_NEW_VERSION_PASS.md` §1.1 | `gms_v<major>.yaml` + `discover_gms_v<major>.md` | No "Missing at discovery" unresolved; every op has `provenance` |
| C | **Seed template** | §1.2 + FR-3.x | `template_gms_<major>_1.json` | Every handler has a validator; all names resolve to real symbols; operations tables populated |
| D | **IDA export & static audit** | §1.3–1.4 | `ida-exports/gms_v<major>.json` + `audits/gms_v<major>/` | Export smoke-tested; SUMMARY emitted; `_unimplemented.json` justified |
| E | **Matrix + verification campaign** | §2–§3, `VERIFYING_A_PACKET.md`, `DISPATCHER_FAMILY.md` | column in `STATUS.md`/`status.json`; byte-fixtures + evidence records | `matrix --check` zero `orphan\|dangling\|stale\|drift\|unresolv\|malformed` + zero 🟥; no in-scope ❌ |
| F | **Code-gate audit slice** | PRD §4.7 | rows appended to the cross-version audit table | Every gate this version touches is classified + (if wrong) fixed with an explicit predicate |
| G | **WZ ingest & serve** | PRD §4.8 | data under `regions/GMS/versions/<major>.1/` | Representative map/item/mob/reactor served for the tenant header; caches cleared |
| H | **k8s ports & tenant provision** | PRD §4.9–4.10 | ports in both synced places; tenant row + seeded config | Ports agree in `atlas-{login,channel}.yaml` *and* `services` config |
| I | **End-to-end playthrough** | PRD §4.10 | documented repeatable steps + a live run | login→world/channel→char select→enter channel→load map→move+chat against the running stack |

**IDB pre-flight (start of every pass).** Confirm the four IDB ports via
`mcp__ida-pro__list_instances` (ports follow launch order, never hardcoded — memory:
`reference_ida_instance_ports_shifted_idbs_v9`). Confirm the target by *binary name*, not
port number. Use `func_query` with `name_regex` only (CLAUDE.md / IDA-MCP rule). An
unresolved fname is a stop-and-ask, never a fabricated substitution
(`feedback_unresolved_fname_escalate`).

## 5. Hardest sub-problem #1 — operation/mode (sub-op) tables

This is the highest-risk surface and it is called out explicitly because three separate
shipped bugs live here:

- `bug_operations_mode_tables_missing_v87_v95_jms` — a missing operations table makes
  `ResolveCode` return a sentinel that crashes the client.
- `bug_v83_status_message_operations_off_by_one` — a one-off shift in the status-message
  table caused the "crash on spawn, fine on re-log" class.
- `bug_npc_msgtype_hardcoded_vs_config` — inbound handlers that hardcode enum bytes
  instead of reverse-resolving the table break when the enum shifts.

**Methodology.** For each dispatcher family the template routes (messenger, cashshop,
storage, npcshop, interaction, party, buddy, guild, worldmessage, status-message,
message-type, …), the per-mode byte table is extracted **from that version's own
dispatcher switch in the IDB** — not inherited from the anchor — and written into the
writer entry's `options.operations` map (`{KEY: byteValue}`, the structure confirmed in
`template_gms_84_1.json`'s `FameResponse` writer). The delta doc (FR-1.2) records each
mode value with its switch-case evidence. Pre-v83 tables are expected to shift
*non-uniformly* vs v83, exactly like the opcode table did for v84
(`bug_v84_opcode_table_shifted_vs_v83`). The number of families needing per-version
extraction (OQ-7) is enumerated during Stage A and recorded as that pass's campaign sizing
input.

**Verification rule.** A dispatcher-family cell counts toward FR-5.3 only with a discrete
struct per mode, a config-resolved mode byte, and a **per-mode body + per-mode byte
fixture** (`DISPATCHER_FAMILY.md`). Enumerating mode bytes without per-arm bodies is a
false pass (`feedback_dispatcher_mode_byte_is_false_pass`) and is graded ❌. The
`dispatcher-family-implementer` agent owns these, serialized one family at a time (shared
`run.go`/`families.yaml`/global IDA instance — never two in parallel).

## 6. Hardest sub-problem #2 — cross-version code-gate audit (FR-7)

The PRD-default vertical shape means each version sees the same `Region()` /
`MajorVersion()` gates, but evaluates them differently. The risk is the systematic
off-by-one class (`bug_majorversion_gt83_is_off_by_one_v87`): a `> 83` / `>= 95` boundary
that coincidentally "works" for one version while silently mis-gating another.

**Strategy: one accumulating table, incremental fixes, final reconciliation.**

1. **Enumerate once, lazily.** The *first* pass (v79) performs the full enumeration of
   every `Region()`/`MajorVersion()`-gated branch in `services/` and `libs/` into a single
   committed table: `code-gate-audit.md` (branch → file:line → result-per-version →
   correct? → action). Known boundary classes to seed from (PRD §4.7): `== 83` exact
   (e.g. character auto-AP in `atlas-character/.../processor.go`), `> 83` / `>= 95`
   (default gender, whisper, attack-common, damage-taken), and the low-version gates
   `<= 28` / `<= 12` (`login/main.go`, `channel/main.go`, session models) that *do* fire
   for these legacy versions and must be verified, not assumed (OQ-6).
2. **Each pass fills its column + fixes its breaks.** A pass classifies how *its* version
   evaluates each row and, where wrong, changes the predicate to an explicit,
   intention-revealing version range backed by Stage-A findings — **never** altering any
   existing version's evaluation (FR-7.2 + NFR isolation).
3. **Final reconciliation (task close).** After v48, a sweep confirms every row has all
   four columns filled, every fix encodes the *intended* range (not a coincidental
   boundary), and a v83 regression check proves no existing version moved.

This keeps the audit table as the single source of truth while honoring the vertical
shape — the table is built once and fixes land per-pass, rather than re-discovering gates
four times.

## 7. Verification campaign — agent fan-out architecture

The campaign (Stage E) is the bulk of the work. Organization:

- **One IDB at a time.** All fan-out for a given version targets that version's single
  loaded IDB. Cross-version parallelism is avoided because the IDA-MCP instance and the
  packet-audit baseline files are shared state.
- **`packet-verifier` per cell-family** for tier-1 byte-fixture cells: derives the client
  read order from the IDB, writes the fixture with a `packet-audit:verify` marker, pins
  the evidence record, regenerates the matrix, and commits test + evidence + STATUS.md
  together. The `discover_gms_v<major>.md` worklist is the coordination list.
- **`dispatcher-family-implementer` per family**, serialized (see §5).
- **Batching.** Tier-1 cells (per `docs/packets/evidence/tiers.yaml`) require fixtures;
  tier-0 cells can reach 🟡 from a tool ✅ + evidence record. The in-scope cell set per
  version (OQ-8) is the registry/matrix output of Stages B–D and becomes that pass's
  acceptance checklist.
- **No hand-editing `STATUS.md`** — it is always regenerated by `packet-audit matrix`.

A 🟥 conflict is never allowlisted: it is arbitrated three-way (registry / template /
Atlas code) against the IDB per `STARTING_A_NEW_VERSION_PASS.md` §5.1, and the wrong leg
is fixed.

## 8. Tooling change — making the four versions first-class

Minimal and surgical. The hardcoded version lists discovered in the worktree:

- `tools/packet-audit/internal/matrix/model.go:14` — `var VersionKeys = []string{…}` (add
  `gms_v48, gms_v61, gms_v72, gms_v79`).
- `tools/packet-audit/internal/matrix/render.go:14` — short-label map (`"gms_v79": "v79"`, …).
- `tools/packet-audit/cmd/fnamedoc.go:222` — `order` slice for fname-doc ordering.
- The `matrix` `--versions` default flag string in
  `STARTING_A_NEW_VERSION_PASS.md` §2 (`tools/packet-audit/cmd/run.go`, default
  `"gms_v83,gms_v84,gms_v87,gms_v95,jms_v185"`).

These are added incrementally (a version's key is wired when its column is first
generated) so `matrix --check` covers it in CI from that pass forward. No new tooling is
built (PRD non-goal); only the `--versions`/default-set wiring. `template_gms_12_1.json`
is left untouched and unused (explicitly rejected as a baseline).

## 9. Artifact inventory (per version `<major> ∈ {79,72,61,48}`)

| Artifact | Path | Owner stage |
|---|---|---|
| Packet delta | `docs/tasks/task-113-gms-legacy-versions/v<major>-packet-delta.md` | A |
| Registry | `docs/packets/registry/gms_v<major>.yaml` | B |
| Discover worklist | `docs/packets/registry/discover_gms_v<major>.md` | B |
| Seed template | `services/atlas-configurations/seed-data/templates/template_gms_<major>_1.json` | C |
| IDA export | `docs/packets/ida-exports/gms_v<major>.json` | D |
| Audit dir | `docs/packets/audits/gms_v<major>/` (SUMMARY + per-packet + `_unimplemented.json`) | D |
| Matrix column | `docs/packets/audits/STATUS.md` + `status.json` (regenerated) | E |
| Evidence records | `docs/packets/evidence/gms_v<major>/<packet>.yaml` | E |
| Byte fixtures | `libs/atlas-packet/...` (with `packet-audit:verify` markers) | E |

Shared/cross-version: `code-gate-audit.md` (one file, all four columns) in the task folder,
and the `tools/packet-audit` version-list edits.

The IDA export splice is surgical and **non-idempotent** — bootstrap from the descending
anchor's export, purge cross-IDB coincidentals, never overwrite a committed export
wholesale (`reference_packet_audit_serverbound_verification`).

## 10. Template construction rules (Stage C, codified)

The template structure is `{region, majorVersion, minorVersion, usesPin, socket:{handlers[],
writers[]}, characters, npcs, worlds, cashShop}`. Handlers are `{opCode, validator,
handler}`; writers are `{opCode, writer, options?:{operations:{KEY:byte}}}`. The
non-negotiable rules:

- **`usesPin`** verified from the client per version (OQ-2 — expected `false` for all four
  pre-v83; confirmed, not assumed).
- **Every handler carries a `validator`** (Pong/StartError → `NoOpValidator`, else
  `LoggedInValidator`/`NoOpValidator` per IDB-confirmed login state). A validator-less
  handler is silently dropped at registration
  (`bug_socket_handler_missing_validator_silently_dropped`) — treated as a defect.
- **Inbound handlers reverse-resolve** the same message-type/operation table the writers
  use (`ResolveName`), never hardcode bytes (`bug_npc_msgtype_hardcoded_vs_config`).
- **Every routed dispatcher family has a populated `operations` table** (§5).
- **All handler/validator/writer names resolve** to real symbols in atlas-channel /
  atlas-login (no dangling references — FR-3.6).
- Seeding is idempotent, reads `SEED_DATA_PATH`, gated by `SEED_ENABLED`, and **must not
  touch any existing version's template/config** (FR-3.5).

## 11. Runtime bring-up notes (Stages G–I)

- **WZ ingest (G):** data under `regions/GMS/versions/<major>.1/`; clear/seed the spawn
  cache so new data is observed (`reference_atlas_maps_spawn_cache`); verify a
  *representative set* (map/item/mob/reactor), not a single asset, watching for legacy WZ
  structural differences the reader may not handle (prior gaps: `consumeOnPickup`,
  snap-to-ground).
- **Ports (H):** wired in **both** `deploy/k8s/base/atlas-{login,channel}.yaml`
  (containerPort + Service.ports) **and** the login/channel `services` config, convention
  `<major>×100` (channel +1). A port in only one place silently fails to route
  (`bug_new_version_lb_socket_ports`).
- **Provision + restart (H/I):** a freshly seeded tenant avoids the live-config-reload
  trap, but the restart sequence is documented (handler/writer projections don't
  hot-reload — `bug_new_opcodes_not_in_live_tenant_config`). Watch the
  config-load-once fatal-on-miss class for factory/world
  (`bug_factory_world_config_load_once_fatalf`) when a tenant is provisioned after pod
  start.
- **Login-flow divergence (OQ-3)** is the single biggest "does it even connect" risk and
  is front-loaded inside Stage A for each pass (handshake, character-list encoding,
  world/channel list, character-select), because the legacy gap from v83 is larger than
  v84's was.

## 12. Risk register (mapped to known failure modes)

| Risk | Mitigation in this design |
|---|---|
| Missing/shifted operations table → client crash | §5: per-version switch extraction + per-mode fixtures |
| Off-by-one version gate | §6: explicit intention-revealing ranges + final reconciliation + v83 regression |
| Validator-less handler dropped | §10: every handler validated, enforced at template review |
| Hardcoded enum bytes | §10: reverse-resolve via `ResolveName` |
| Dispatcher false pass (mode-byte only) | §5/§7: per-mode body + fixture, `dispatcher-family-implementer` |
| Export overwrite / coincidental fns | §9: surgical splice, purge cross-IDB coincidentals |
| Stale spawn cache masks new WZ | §11: clear caches, representative-set verification |
| Port wired in one place only | §11: both synced places, `<major>×100` convention |
| Fabricated fname to "unblock" | §4 pre-flight: unresolved fname = stop-and-ask |
| Existing tenant regression | §13 |

## 13. Regression & isolation guarantees

Adding the four versions must not change any existing version's behavior (NFR isolation).
Concretely: template seeding is additive and version-keyed; WZ keys are additive; code-gate
fixes encode explicit ranges that preserve every existing version's evaluation; and each
pass ends with at least a v83 login→channel→map regression check (FR-10.3). The
cross-version reconciliation (§6) re-confirms this after the final (v48) pass.

## 14. Build & verification gates

Per CLAUDE.md, before any pass is called done for its Go-touching changes:
`go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module;
`docker buildx bake atlas-<svc>` for every service whose `go.mod` was touched;
`tools/redis-key-guard.sh` clean (run with `GOWORK=off`). Most passes touch
atlas-configurations (template — no Go), packet-audit (Go), and possibly
atlas-channel/login/character/account (Go gate fixes); each of those triggers the bake step.

## 15. Open questions — disposition

- **OQ-1 (IDB ports/naming):** resolved per-pass at the §4 pre-flight via
  `list_instances`; naming blockers are *in scope* (FR-6), not deferred.
- **OQ-2 (usesPin):** confirmed per version in Stage C (§10); expected `false`.
- **OQ-3 (login divergence):** front-loaded in Stage A (§11) — highest connect-risk.
- **OQ-4 (WZ availability):** **closed for design** — owner confirmed all four available;
  ingestion is a firm deliverable. Carve-out path retained only as contingency.
- **OQ-5 (UI blocker):** treated as discovered-at-provisioning; only addressed if standing
  up a tenant proves impossible via REST/seed (PRD non-goal otherwise).
- **OQ-6 (`<=28`/`<=12` gates):** resolved inside the §6 audit from IDA/behavior.
- **OQ-7 (operation-table count):** enumerated in each pass's Stage A; drives campaign size.
- **OQ-8 (scope-cell list):** produced by Stages B–D per pass; becomes that pass's
  acceptance checklist.

## 16. Out of scope (restated from PRD)

Migrating/removing any existing version; non-GMS regions or other GMS versions; using
`template_gms_12_1.json`; building new packet-audit/IDA tooling (beyond `--versions`
wiring); full content parity for version-exclusive systems; atlas-ui changes (unless OQ-5
becomes a hard blocker).

## 17. Sequencing summary

```
Pass 1: v79  (anchor 83)  →  Stages A–I  +  full code-gate-audit.md enumeration (v79 column)
Pass 2: v72  (anchor 79)  →  Stages A–I  +  v72 column + fixes
Pass 3: v61  (anchor 72)  →  Stages A–I  +  v61 column + fixes
Pass 4: v48  (anchor 61)  →  Stages A–I  +  v48 column + fixes
Close:  cross-version reconciliation + v83 regression + full build/bake/redis-guard gate
```

Each pass is internally ordered A→I with the gates in §4; the verification campaign (E)
and dispatcher families fan out under the constraints in §5/§7. The plan will template
this nine-stage structure four times plus the shared first-pass enumeration and the
closing reconciliation.
