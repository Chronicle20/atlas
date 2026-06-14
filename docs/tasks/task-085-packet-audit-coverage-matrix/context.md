# Task-085 Context — key files, locked decisions, dependencies

Companion to `plan.md`. Everything below was verified against the worktree on
2026-06-12 (commit 142283a27 base).

## Inputs

- Spec: `design.md` (this folder). No `prd.md` — this task was designed
  directly via brainstorming (the allowed skip-PRD path); `retrospective.md`
  is the predecessor analysis the design builds on.
- Out of scope: design Phase 6 (tier-1 fixture campaign) → family-sized
  follow-up tasks. Live-IDB tasks (plan 5.4/5.5) are operator-gated and may
  split into a follow-up if no IDA is available during execution.

## Verified code geography

| What | Where |
|---|---|
| Subcommand dispatch | `tools/packet-audit/cmd/root.go:26-46` — plain `flag` string dispatch, NOT cobra. New: `registry`, `matrix`, `evidence`, `discover-ops`. |
| Existing subcommands | export, validate, infer, decompose, triage, resolve-dispatch, diff-shape |
| Audit report struct | `internal/report/report.go:13-30` (`report.Packet`; JSON per packet at `docs/packets/audits/<version>/<Writer>.json`) |
| Verdict enum | `internal/diff/diff.go:10-45` — Match=0 ✅, Minor=1 ⚠️, Blocker=2 ❌, Deferred=3 🔍, Unresolved=4 🚫 (JSON-encodes as int) |
| FlatInvalid | `report.Packet.FlatInvalid` — caps verdict at 🔍; matrix treats it as tier-1 unconditionally |
| Old CSV parser | `internal/csv/csv.go:61-123` — keeps only FName→opcodes; NOT sufficient for seeding (drops Op names + index cells) → new `internal/seedcsv` |
| Template loader | `internal/template/template.go:45-94` — `Load(path)`, `Writers() map[int]string`, `Handlers() map[int]string` |
| IDA export schema | `internal/idasrc/export.go:11-74`; files `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json` (no v84 yet) |
| MCP client | `internal/idasrc/mcp.go:32-37` (GetFunctionByName/DecompileFunction/GetCallees/StructInfo); HTTP impl `mcphttp.go` incl. `SelectInstance(port)` |
| Atlas analyzer | `internal/atlaspacket/registry.go:88+` `NewTypeRegistry(root)`; `Calls(typeName)`, `IsOpaque`, Call.Guard holds version predicates |
| Tenant templates | `services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_92,gms_95,jms_185}_1.json` (+gms_12) |
| Skills format | `.claude/commands/<name>.md` (frontmatter: description, argument-hint) — convert-npc is the reference; agents in `.claude/agents/<name>.md` |
| CI model | `.github/workflows/catalog-lint.yml` — path-filtered single job, GO_VERSION 1.25.5 |
| go.work | `tools/packet-audit` is a workspace member (line 81) |

## CSV facts (verified by reading the real files)

- Header: `Op,FName,Index,GMS v12,,GMS v48,...,GMS v111,,JMS v185`. A
  version's (index, opcode) pair = (column **before** the labeled column,
  labeled column). First version's index column is literally named `Index`.
- Presence: index cell non-empty OR opcode != 0. Proof cases:
  `LOGIN_STATUS` v83 = (`0`, `0x000`) → present with opcode 0;
  `ACCOUNT_INFO` JMS = (``, `0x000`) → absent.
- ClientBound has versions v12..v111+JMS185; ServerBound only
  v12,83,87,92,95,111,JMS185. **Neither has a v84 column** → `gms_v84.yaml`
  seeds as a v83 copy + note (task-083: v84 ≡ v83), corrected by discover-ops.
- Multiline FName cells exist (`SERVERLIST_REREQUEST` row 5: two FNames in one
  quoted cell) → `fname` + `fname_alts`. Rows with empty FName exist
  (`GUEST_LOGIN`) → keep with `fname: ""`.
- Row counts: ~585 ClientBound, ~703 ServerBound (vs 253 audited v83 rows —
  most present ops have no Atlas implementation; this is why D3 below exists).

## Locked design-ambiguity decisions

- **D1 — hash basis.** Exports store parsed function records, not decompile
  text. `decompile_sha256` = sha256 over canonical re-marshaled JSON
  (`map[string]any` round-trip; Go sorts keys) of the function's export entry.
  One helper (`evidence.FunctionHash`) serves both `pin` and `--check`.
- **D2 — no date in the stamp.** Design §9 asks for a date; it breaks
  determinism (design §14 requires byte-identical re-runs) and the
  `--check` committed-output comparison. Stamp = tool tree SHA
  (`git rev-parse HEAD:tools/packet-audit`) + per-version export sha256 only.
- **D3 — conflict rule refinement.** "Registry present but template unrouted"
  is a 🟥 conflict **only when another version's template routes the same
  (opcode, direction)** (the task-067/068 gap class). Routed-nowhere = ❌
  incomplete (unimplemented surface, ~300+ ops) — otherwise the matrix is born
  with hundreds of junk conflicts. Registry-absent + (template routes OR Atlas
  report exists) is always 🟥.
- **D4 — tier expansion.** `tiers.yaml` stays the explicit reviewed artifact
  but at three granularities: `packets`, `packet_prefixes` (dispatcher
  families), `opaque_types` (the 8 ledger families; expansion to packets via
  TypeRegistry recursion is deterministic tool code). FlatInvalid ⇒ tier-1
  computed, not listed.
- **D5 — op↔report join.** Registry FName ↔ report `IDAName` with the `#case`
  suffix stripped; an FName mapping to multiple per-case writers grades as the
  **worst** candidate cell. Reports never consumed by an op row render in a
  separate "Sub-structs & shared types" section (no applicability/n-a logic),
  which implements design §10 rule 4.
- **D6 — "latest-tool" enforcement** is at matrix level (stamp + `--check`
  staleness), not per-report: reports don't record tool SHA and regenerating
  them all is out of scope. A tool change ⇒ `--check` fails until matrix is
  regenerated and committed in the same PR.
- **D7 — serverbound discovery** does not extend MCPClient speculatively:
  phase-5 code does dispatch-walk (clientbound) + registry-FName verification
  (serverbound); xref-based send-site enumeration is decided at the live-run
  checkpoint.
- **D8 — first external dep.** `gopkg.in/yaml.v3` added to the tool module
  (design mandates YAML schemas; hand-rolling YAML is worse).
- **D9 — discovered-op naming.** discover-ops appends placeholder op names
  (`IDA_0X<opcode>`); canonical renames are human edits with
  `provenance: manual`.

## Known traps for the executor

- jms_v185's export file is `gms_jms_185.json` (audit dir IS `jms_v185`);
  `matrix.ExportPath` owns the mapping. The old `--version gms_jms_185`
  triage/decompose default-dir mismatch is documented in memory + root.go:163.
- Committed audit reports still carry `AtlasFile: "../../libs/atlas-packet/..."`
  (PR #729 normalized the writer, not the committed JSONs) → `PacketID()`
  normalizes defensively.
- `_pending.md` exists in TWO places: `docs/packets/ida-exports/_pending.md`
  (master, 26KB) and `docs/packets/audits/gms_v95/_pending.md` (v95 subset).
  Both get frozen banners in Task 2.5.
- Case labels in Hex-Rays carry `u` suffixes (`case 200u:`) — discover parser
  and any export tooling must strip them.
- The seeded registry WILL produce conflict noise until Task 5.4 discovery
  runs (design §15 risk 3) — don't "fix" it by hand-editing the registry in
  Phase 1, and don't land the CI gate red (Task 6.1 has the escape hatch).
- gms_v84 has no export and no audit dir until Task 5.5; every loader must
  tolerate that absence (LoadReports returns empty for missing dir; evidence
  for v84 would be a dangling-citation check failure — there is none yet).
- Don't create `*_testhelpers.go` files (CLAUDE.md); test fixtures live in
  `testdata/` like the rest of the tool.

## Verification gates

Per phase: `cd tools/packet-audit && go test -race ./... && go vet ./...`.
Final: + `libs/atlas-packet` tests, `tools/redis-key-guard.sh`,
`go run ./tools/packet-audit matrix --check` exit 0, code review
(`superpowers:requesting-code-review`) before PR.
