# Context — task-113 GMS Legacy Versions (48.1 / 61.1 / 72.1 / 79.1)

Companion to `plan.md`. Captures the key files, decisions, and dependencies an
implementer needs but the bite-sized plan steps don't repeat.

## What this task is

Add four pre-v83 GMS client versions (48.1, 61.1, 72.1, 79.1) as fully supported,
runnable tenants **alongside** the existing GMS 83/84/87/95 + JMS 185 tenants —
not a migration. Each version gets the full new-version bring-up: protocol layer
(registry → template → IDA export → static audit → verified matrix column) +
runtime layer (WZ data → k8s ports → tenant → live playthrough). Four sequential
**vertical passes** in **descending** order (79 → 72 → 61 → 48); each completed
version is the anchor for the next-lower one.

## Locked decisions (design §2)

- **Vertical per-version execution.** Each version completes its full
  protocol → code-audit → runtime slice before the next starts. (Not horizontal /
  all-registries-then-all-templates.)
- **FR-7 code-gate audit spans all versions** → one accumulating table
  (`code-gate-audit.md`), enumerated once (Phase 0), filled per pass, reconciled
  at close. Not a front-loaded separate phase.
- **WZ data assumed available for all four** (owner-confirmed). Ingestion is a
  firm deliverable; OQ-4 carve-out retained only as contingency.
- **`template_gms_12_1.json` is rejected** as a baseline — left untouched/unused.

## Anchor chain & tie-breaker

| Pass | Version key | Anchor | Tie-breaker |
|---|---|---|---|
| 1 | `gms_v79` | `gms_v83` | GMS 95 IDB/PDB |
| 2 | `gms_v72` | `gms_v79` | GMS 95 IDB/PDB |
| 3 | `gms_v61` | `gms_v72` | GMS 95 IDB/PDB |
| 4 | `gms_v48` | `gms_v61` | GMS 95 IDB/PDB |

"Same as anchor" is a finding requiring switch-case evidence (FR-1.4), never a default.

## Key existing files & patterns

**Playbooks (read these — they are authoritative):**
- `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` — the orchestration: registry
  (§1.1) → template (§1.2) → export (§1.3) → static audit (§1.4) → matrix (§2) →
  promote cells (§3) → task-close gate (§4) → conflict/degradation remediation (§5).
- `docs/packets/audits/VERIFYING_A_PACKET.md` — single-cell verification procedure.
- `docs/packets/DISPATCHER_FAMILY.md` — discrete-struct-per-mode dispatcher rules.

**Reference artifacts to copy patterns from:**
- Templates: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
  is the cleanest structural reference (handlers `{opCode,validator,handler}`;
  writers `{opCode,writer,options?:{operations:{KEY:byte}}}`; `FameResponse` writer
  shows the `operations` mode-table shape).
- Registry/worklist: `docs/packets/registry/gms_v87.yaml` + `discover_gms_v87.md`
  are the reference completed examples (the v87 run found ~40 dispatchers).
- Existing exports/audits: `docs/packets/ida-exports/gms_v83.json`,
  `docs/packets/audits/gms_v83/`.

**packet-audit tooling edit sites (design §8) — wired incrementally per pass:**
- `tools/packet-audit/internal/matrix/model.go:14` — `var VersionKeys = []string{...}`.
  The matrix `--versions` default derives from this (`matrix.go:56` joins it), so
  this is the single place that controls both the column set and the CI default.
- `tools/packet-audit/internal/matrix/render.go:14` — short-label map
  (`"gms_v83":"v83"`, ...). Add `"gms_v79":"v79"` etc.
- `tools/packet-audit/cmd/fnamedoc.go:222` — `order := []string{...}` slice
  (note: includes a `gms_jms_185` alias alongside `jms_v185`).
- Phase-0 Task 0.2 adds guard tests so a half-wired key fails `go test`, not just
  `matrix --check`.

## Code-gate audit — known boundary classes (FR-7, design §6)

Verified anchors from a worktree grep (`MajorVersion()` predicates):
- **`<= 28`** (login state): `services/atlas-login/atlas.com/login/main.go:277`,
  `services/atlas-channel/atlas.com/channel/main.go:391`.
- **`<= 12`** (session model): `services/atlas-login/atlas.com/login/session/model.go:35`,
  `services/atlas-channel/atlas.com/channel/session/model.go:40`.
  These low gates **do** fire for the legacy range — resolve from IDA/behavior (OQ-6),
  never assume.
- **`>= 95`** (many): `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`
  (many lines), `.../socket/writer/character_attack_common.go:180`,
  `.../socket/model/damage_taken_info.go:66`, `libs/atlas-packet/party/member_data.go:87`,
  `libs/atlas-packet/chat/serverbound/multi.go:54`.
- **`== 83` exact** and **`> 83`**: character auto-AP, default gender (atlas-account) —
  enumerate the full set with the Phase-0 grep; do not trust this list as exhaustive.

Risk: the off-by-one class (`bug_majorversion_gt83_is_off_by_one_v87`). Fixes must
encode the **intended** version range, never a coincidental boundary, and must not
change any existing version's evaluation.

## Operation/mode tables — highest-risk surface (design §5)

Three shipped bugs live here:
- `bug_operations_mode_tables_missing_v87_v95_jms` — a missing operations table →
  `ResolveCode` returns a sentinel → client crash.
- `bug_v83_status_message_operations_off_by_one` — one-off shift → "crash on spawn".
- `bug_npc_msgtype_hardcoded_vs_config` — inbound handlers hardcoding enum bytes
  break when the enum shifts.

Per dispatcher family (messenger, cashshop, storage, npcshop, interaction, party,
buddy, guild, worldmessage, status-message, message-type, …): extract the per-mode
byte table **from this version's own dispatcher switch in the IDB** (not inherited),
write it into the writer's `options.operations` map. Tables shift non-uniformly
vs v83 (like the opcode table did for v84). Dispatcher cells count only with a
discrete struct per mode + config-resolved byte + **per-mode body + per-mode fixture**
(`dispatcher-family-implementer` owns these, serialized — never two in parallel).

## Verification campaign — agent fan-out (design §7)

- **One IDB at a time** (IDA-MCP instance + packet-audit baseline are shared state).
- `packet-verifier` agent per tier-1 cell-family; `dispatcher-family-implementer`
  per family, serialized.
- Tier membership: `docs/packets/evidence/tiers.yaml`. Tier-1 needs a byte fixture;
  tier-0 reaches 🟡 from a tool ✅ + evidence record (hash must match the export).
- Coordinate via the `discover_gms_v<major>.md` worklist.
- Never hand-edit `STATUS.md` — `packet-audit matrix` regenerates it.
- A 🟥 is arbitrated three-way (registry/template/Atlas code) against the IDB
  (`STARTING_A_NEW_VERSION_PASS.md` §5.1) — never allowlisted/deferred.

## IDA / reverse-engineering rules

- Confirm instances via `mcp__ida-pro__list_instances`; **ports follow launch order**,
  never hardcode (`reference_ida_instance_ports_shifted_idbs_v9`). Confirm the target
  by **binary name**.
- `func_query` with `name_regex` only (CLAUDE.md / IDA-MCP rule).
- An unresolved fname is **stop-and-ask**, never a fabricated/auto-substituted name
  (`feedback_unresolved_fname_escalate`).
- The IDA export splice is surgical and **non-idempotent** — bootstrap from the
  anchor's export, purge cross-IDB coincidentals, never overwrite a committed export.

## Runtime bring-up gotchas (Stages G–I)

- WZ: data under `regions/GMS/versions/<major>.1/`; clear spawn cache
  (`reference_atlas_maps_spawn_cache`); verify a representative set (map/item/mob/
  reactor), watching for legacy reader gaps (`consumeOnPickup`, snap-to-ground).
- Ports: wired in **both** `deploy/k8s/base/atlas-{login,channel}.yaml`
  (containerPort + Service.ports) **and** the login/channel `services` config,
  convention `<major>×100` (channel +1) (`bug_new_version_lb_socket_ports`).
- A freshly seeded tenant avoids the live-config-reload trap, but handler/writer
  projections don't hot-reload (`bug_new_opcodes_not_in_live_tenant_config`); watch
  the config-load-once fatal-on-miss class for factory/world
  (`bug_factory_world_config_load_once_fatalf`).
- Login-flow divergence (OQ-3) is the biggest "does it even connect" risk —
  front-loaded into Stage A (handshake, character-list encoding, world/channel list,
  character-select).

## Dependencies & ordering

- **Strict pass ordering:** v72 depends on v79 (its anchor), v61 on v72, v48 on v61.
  Within a pass, stages are A→I in order; the verification campaign (E) and dispatcher
  families fan out under the §5/§7 constraints.
- **Phase 0 is a prerequisite for all passes** (IDB ports, CSV inventory, WZ
  availability, the guard tests, the gate-table skeleton).
- **Phase 5 depends on all four passes** (reconciliation + final regression + gate).

## Build / verification gates (CLAUDE.md)

For any pass touching Go: `go test -race ./...`, `go vet ./...`, `go build ./...`
clean in every changed module; `docker buildx bake atlas-<svc>` from the worktree
root for every service whose `go.mod` was touched; `tools/redis-key-guard.sh` clean
(`GOWORK=off`). Template-only/doc-only passes skip the Go gate but still run
`matrix --check`. `packet-audit` is a tool, not a baked service — don't bake it.

## Acceptance (PRD §10)

Per version: packet-delta doc, registry + worklist, seed template (validators +
operations tables + usesPin), IDA export + audit dir, matrix column with zero
unresolved 🟥 and `matrix --check` clean, comprehensive campaign with no in-scope ❌,
IDB named, WZ served, ports wired, tenant provisioned + live playthrough. Spanning:
code-gate audit table complete + reconciled, `--versions` covers all four, v83
regression passes, full build/bake/redis-guard clean.
