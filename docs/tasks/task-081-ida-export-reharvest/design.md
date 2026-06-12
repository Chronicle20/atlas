# IDA Export Re-Harvest — Automated Decompiler-Parser Exporter — Design

Status: Approved
Created: 2026-06-04
PRD: `docs/tasks/task-081-ida-export-reharvest/prd.md`
Builds on: task-080 (packet-audit closeout, PR #678)

---

## 1. Context and the central reframing

The PRD repeatedly instructs us to "fix the **exporter** so future exports descend
into struct-reading helpers and never truncate." Exploration of the current code
shows the framing rests on a component that does not yet exist:

- `tools/packet-audit/cmd/export.go` `runExport` is a stub — it prints
  `"requires --ida-source mcp with a configured MCP client"` and returns exit
  code 3.
- `tools/packet-audit/internal/idasrc/mcp.go` `ParseDecompile` is a stub —
  `return nil, errors.New("idasrc: ParseDecompile not yet implemented")`.
- `MCPClient` is an interface with **no real implementation**.
- The checked-in `docs/packets/ida-exports/*.json` files were **hand-authored**
  from IDA-MCP harvest (subagents reading decompiles), per the project memory
  `reference_ida_harvest_subagents`.

Therefore the v83/v87/JMS BuddyInvite mistraces task-080 found were **not a tool
bug** — they were a *harvest-methodology* miss: whoever traced `OnFriendResult`
case-9 did not descend into `CFriend::Insert` (the `GW_Friend` 39-byte struct
read) and mislabeled it as a `count + buddy[i]` loop; the JMS trace truncated
after `level`.

Two further facts shape the design:

1. **Struct-helper descent already has schema support.** The export JSON schema
   supports `Op:"Delegate"` with `Ref:"<siblingFName>"`; the resolver
   (`internal/idasrc/export.go` `resolveWithVisited`) splices the referenced
   FName's resolved calls inline, AND-ing guards and cycle-guarding the descent
   (task-065 item 8). What was missing was a *producer* that emits these
   descents, not a *consumer* that honors them.

2. **IDA-MCP is reachable over HTTP.** The MCP server is configured as
   `{"type":"http","url":"http://192.168.20.3:13337/mcp"}`. A standalone Go
   binary can speak MCP-over-HTTP to it directly — a real `MCPClient` is
   feasible. The endpoint is a LAN address, so the export step is
   **maintainer-local** (matching the README's "export is a maintainer step; CI
   consumes the checked-in JSON" model); CI never needs IDA.

**Decision (approved):** build the real automated exporter — a decompiler-parser
that drives IDA-MCP over HTTP, parses Hex-Rays output **with IDA structured-query
assistance**, descends into packet-reading helpers, and emits deterministic
per-version JSON. Then re-export all four IDBs, re-audit, triage every verdict
flip against IDA ground truth, and fix what is genuinely wrong — all in one
phased task.

### 1.1 The defining correctness property

> **The parser MUST NOT emit a confident-but-wrong trace. When it cannot prove a
> read-order element (loop vs. fixed struct, an indirect dispatch target, a
> data-driven read), it emits `unresolved:true` rather than a guess.**

This is the **anti-BuddyInvite invariant**. A wrong-but-plausible trace
(`count + loop` where the truth is a fixed 39-byte struct) is strictly worse
than an honest `unresolved`, because the audit then renders a confident-but-false
verdict that hides a real bug. Precision over recall is the design's guiding
trade-off: the tool's value is full-coverage truncation-catching and
reproducibility, **not** 100% automation. The hard packets that fall to
`unresolved` are hand-filled (using the existing `Delegate`/`Ref` schema), and
§4.7 verdict-delta triage hand-verifies every flip against the IDA decompile
regardless.

## 2. Architecture overview

```
                    maintainer machine (one IDB loaded at a time)
                    ┌─────────────────────────────────────────────┐
   packet-audit     │  MCP-HTTP                                    │
   export  ────────▶│  ┌──────────────┐   tools/call   ┌────────┐ │
   --ida-url ...     │  │ mcphttp.go   │───────────────▶│ IDA-MCP│ │
   --version gms_v95 │  │ (MCPClient)  │◀───────────────│  :13337│ │
                    │  └──────┬───────┘                 └────────┘ │
                    │         │ decompile text + callees + struct  │
                    │         ▼                                    │
                    │  ┌──────────────┐                            │
                    │  │  parse.go    │  read-order extraction,    │
                    │  │ (ParseDecomp │  descent (cycle/bound),    │
                    │  │  + descent)  │  loop-vs-struct, sub-cases,│
                    │  └──────┬───────┘  unresolved fallback       │
                    │         ▼                                    │
                    │  ┌──────────────┐                            │
                    │  │ runExport    │  roster, determinism,      │
                    │  │ (cmd)        │  provenance, JSON write     │
                    │  └──────┬───────┘                            │
                    └─────────┼───────────────────────────────────┘
                              ▼
        docs/packets/ida-exports/<region>_v<major>.json  (replaces existing)
                              │
                              ▼
        packet-audit (run)  ──▶  docs/packets/audits/<region>_v<major>/...
        (CI-safe: consumes checked-in JSON, no IDA)
```

The audit-consumption path (`run.go`, FName × `candidatesFromFName`, the
resolver, the analyzer) is **unchanged** — the export JSON schema is preserved
(plus the additive `unresolved` marker) so the audit consumes corrected exports
without modification.

## 3. Components

All new/changed code lives under `tools/packet-audit` unless noted.

### 3.1 `internal/idasrc/mcphttp.go` — real `MCPClient` over MCP-HTTP

Implements the existing `MCPClient` interface plus the extra queries the
API-assisted parser needs. Surface:

```go
type MCPClient interface {
    GetFunctionByName(ctx, name) (addr string, ok bool, err error)
    DecompileFunction(ctx, addr) (text string, err error)
    // new, for API-assisted parsing:
    GetCallees(ctx, addr) ([]Callee, error)            // mcp__ida-pro__get_callees
    StructInfo(ctx, name|addr) (StructLayout, error)   // analyze_struct_detailed / get_struct_at_address
}
```

- MCP-HTTP lifecycle: `initialize` → `notifications/initialized` → `tools/call`
  per request; reuse one session per export run.
- Tool mapping: `get_function_by_name`, `decompile_function`, `get_callees`,
  `analyze_struct_detailed`/`get_struct_at_address` (struct layout for fixed-vs-loop).
- Config: `--ida-url` flag (default the configured LAN endpoint), `--ida-timeout`.
- Resilience: explicit error if the endpoint is unreachable (maintainer ran it
  with no IDB / wrong network) — never silently produce an empty export.

This file is **not unit-tested against a live server**; its logic is exercised by
an in-package fake transport. Live behavior is validated during P2 (the actual
re-export) by a maintainer.

### 3.2 `internal/idasrc/parse.go` — `ParseDecompile` + descent driver

Replaces the stub. Responsibilities, in priority order:

1. **Ordered primitive extraction.** Scan decompile text for
   `CInPacket::Decode{1,2,4,8}`, `DecodeStr`, `DecodeBuffer` (and the
   `COutPacket::Encode*` duals) in source order, capturing op + best-effort
   semantic label (label reliability is explicitly *not* gated on — §9 of the
   PRD; only op width + order are load-bearing).
2. **Descent target identification.** Use `get_callees(addr)` to enumerate calls,
   then determine which callees receive the packet pointer as an argument
   (the `CInPacket*`/`a1`-equivalent). Those are packet-reading helpers
   (`CFriend::Insert`, `GW_*::Decode`/`Insert`, `DecodeBuffer`-into-struct) and
   are recursed into; their inlined reads splice at the call position. Non-packet
   callees (UI/dialog/alloc: `CUIFadeYesNo::*`, `StringPool::*`, `operator new`,
   etc.) are skipped via a denylist + the "does not take the packet arg" test.
3. **Recursion safety.** Cycle-guard (visited set, mirroring the resolver's
   existing pattern) + a configurable depth bound (`--descent-depth`, default
   small). Exceeding the bound on a still-descending path → `unresolved`, never a
   truncated guess.
4. **Loop-vs-fixed-struct disambiguation.** The BuddyInvite anti-case. A run of
   reads is a genuine count-prefixed loop only when there is a `Decode*` count
   immediately preceding a loop construct whose body reads `count` times. A
   fixed-size struct read is identified by: a struct typedef on the destination,
   a `get_callees` descent into a `GW_*::Decode`/`Insert` helper, or a fixed
   `N*Index` stride in surrounding code. When the evidence is contradictory or
   absent → `unresolved`.
5. **Mode-switch sub-cases.** A `switch`/`case` on a mode/discriminator byte
   yields one read-order per case, each captured under its guard (consumed by
   the audit's existing per-`#Suffix` FName mechanism / guard field).
6. **`unresolved` fallback.** Any element the parser cannot prove emits
   `{op:"Unresolved", comment:"<why>"}` or sets `unresolved:true` on the function
   (see §3.4). This is logged explicitly (observability NFR) so gaps are visible.

The parser is a structured line/brace scanner (not a full C-AST parse — Hex-Rays
output is not clean C: `__int64`, `LOBYTE()`, gotos, casts). It tracks brace
depth and the `for`/`while`/`do`/`switch` keywords for control-flow shape, and
leans on the IDA structured queries (callees, struct layout) wherever text
analysis is ambiguous — which is exactly where the BuddyInvite mistrace happened.

### 3.3 `cmd/export.go` `runExport` — driver

- **Roster** (which FNames to harvest): the union of
  (a) existing keys in the target version's `*.json`,
  (b) the `candidatesFromFName` FName set (`run.go`), and
  (c) FNames listed in `docs/packets/ida-exports/_pending.md`.
  This re-exports everything currently audited plus the known gaps — not every
  function in the binary (most are not packets).
- **Per-version invocation:**
  `packet-audit export --ida-url <url> --version gms_v95 --output docs/packets/ida-exports/gms_v95.json`.
  The maintainer loads the matching IDB, runs, then cycles to the next version.
- **Determinism (FR-1.5):** stable key ordering (sorted FNames), stable call
  ordering (source order), normalized whitespace — so the large diff is
  reviewable per-function.
- **Provenance:** `binary`, `md5`, `generated_at`, per-function `address`.
- **Unresolved reporting:** end-of-run summary to stderr — count resolved /
  descended / unresolved, and the unresolved FName list — so the maintainer
  knows exactly what needs hand-fill.

### 3.4 Schema additions (additive, backward-compatible)

```jsonc
// exportFn gains:
"unresolved": true,        // function the parser could not faithfully trace
// rawCall gains the op:
{ "op": "Unresolved", "comment": "indirect dispatch via vtable; hand-trace" }
```

- The audit resolver treats an `unresolved` function / call as a **known gap**:
  it produces no false verdict and the SUMMARY shows it as an explicit
  unresolved marker (distinct from `✅`/`❌`/`🔍`), satisfying FR-2.3.
- `Delegate`/`Ref` is retained unchanged — it is the hand-fill mechanism for
  helpers the maintainer traces manually (author the helper as its own FName
  entry, reference it via a `Delegate` call).

## 4. Phased execution plan

"Everything, phased" (per approval). Each phase ends with an explicit gate; later
phases consume earlier artifacts only.

### Phase 0 — Rebase + verdict snapshot (hard prerequisite)
- Rebase `task-081` onto task-080 so the corrected baseline, A1–A5 analyzer
  enhancements, `STARTING_A_NEW_VERSION_PASS.md`, and curated `_pending.md` are
  present. (Preferred: once PR #678 merges to `main`, rebase onto `main`; if not
  yet merged at execution time, rebase onto the `task-080-packet-audit-closeout`
  branch and re-baseline onto `main` after merge.) **This worktree currently
  forks from pre-080 `main` (`40af0c80f`); the corrected exports and snapshot do
  not exist here yet.**
- Snapshot task-080's per-packet verdict set per version (the §4.7 / FR-7.1
  comparison baseline) into the task folder
  (`docs/tasks/.../verdict-snapshot-080.md`).
- **Gate:** rebase clean; snapshot committed; existing task-080 audit re-run
  reproduces its SUMMARYs (sanity check the base is intact).

### Phase 1 — Build the exporter (TDD)
- Implement `mcphttp.go`, `parse.go`, `runExport`, schema additions.
- TDD against **checked-in decompile fixtures** under
  `internal/idasrc/testdata/` (raw Hex-Rays text + expected `Fields`), so CI
  needs no live IDA. Fixtures cover: simple linear reads, a count-loop, a
  fixed-struct descent, a mode-switch, a cycle, a non-packet-helper skip, and an
  unresolvable case.
- **Canonical regression test:** the four-version `OnFriendResult#Invite` traces
  resolve to `…name, [jobId, level if ≥v87/JMS], GW_Friend(39), inShop` — *not* a
  count-loop and *not* truncated. This is the explicit anti-case that proves the
  descent + loop-disambiguation works.
- **Gate:** `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
  `tools/packet-audit`; all fixtures pass; the BuddyInvite regression passes for
  all four versions.

### Phase 2 — Full re-export (live IDA, maintainer-cycled)
- For each of GMS v83 / v87 / v95 + JMS v185: maintainer loads the IDB, runs
  `packet-audit export`, parser descends + emits JSON. Subagents may drive the
  per-IDB batch (memory `reference_ida_harvest_subagents`); IDA is serial so runs
  are one-at-a-time.
- Replace the four `docs/packets/ida-exports/*.json`.
- Hand-fill `unresolved` functions where feasible using `Delegate`/`Ref` +
  focused decompile reads; leave a genuine `unresolved` marker where not.
- **Gate:** all four re-exported, deterministic re-run is byte-identical;
  committed with a **structural-change summary** (descents resolved / truncations
  recovered / loops corrected / unresolved count) so a reviewer can audit the
  *exporter's* correctness, not just trust it (FR-2.2, reviewable-diffs NFR).

### Phase 3 — Re-audit + verdict-delta triage (§4.7)
- Re-run `packet-audit` over corrected exports for all four versions, correct
  invocation: `--output docs/packets/audits` (the **parent** — the tool appends
  `<region>_v<major>`; per memory `reference_packet_audit_tool_mechanics`).
- Compute the **per-packet verdict delta** vs the Phase-0 snapshot (FR-7.2) —
  the exact set of packets whose verdict changed, not aggregate counts.
- Classify and handle every delta entry (FR-7.3):
  - **❌→✅:** confirm a representative sample flipped *because* of the corrected
    read-order (not coincidence); accept the remainder.
  - **✅→❌ (the dangerous class):** hand-decompile the function in IDA and
    compare to Atlas. Outcome is exactly one of: (a) real Atlas wire bug → fix
    (P4); (b) new export over-corrected → fix the **exporter** (back to P1),
    not Atlas; (c) verified representation-equivalence → recorded exception with
    IDA evidence. Never acted on by trusting the new export alone.
  - **new non-`✅` on previously-unaudited/unresolved:** same hand-investigation.
- **Gate:** every flip carries a written disposition (fixed-Atlas / fixed-exporter
  / verified-equiv); **zero rubber-stamped** in either direction (FR-7.5).

### Phase 4 — Fix surfaced wire bugs (task-080 discipline)
- For each genuine divergence: confirm read-order in IDA → ship a **per-version
  byte-level test as the oracle** → apply version/region gates symmetrically in
  Encode/Decode → use the region-dispatch idiom for >2-version divergences
  (≤2 nested guards, analyzer-visible via task-080 A5). No wire change ships on
  analyzer verdict alone.
- Scope: `libs/atlas-packet/**` + downstream `services/**` handlers/producers as
  surfaced.
- Genuinely-too-large (multi-service protocol) bug → register a **dedicated
  follow-up task** (never parked into `_pending.md` as accepted — memory
  `feedback_no_todos_in_deliverables`).
- **Gate:** each fix has a per-version byte test; changed modules pass
  test/vet/build; `docker buildx bake` per touched `go.mod`.

### Phase 5 — Opaque register-boundary decomposition (FR-4)
- For each type in task-080's opaque set (`model.Asset`/`GW_ItemSlotBase`,
  `GW_CharacterStat`, monster stat blobs, `BuddyEntry`, pet bodies, the ~31
  A3-flagged types): determine via IDA whether the client read is decomposable
  into known primitives.
- Decomposable → extend the analyzer/registry (or the corrected export's
  `Delegate` entries) so fields verify inline.
- Genuinely undecomposable (mask/mode-driven variable layout) → confirm Atlas's
  encoder against the client with a dedicated byte test + a **recorded verified
  exception** ("verified correct, analyzer can't model it"), replacing the
  "analyzer skipped it" status.
- **Gate:** no type remains in an unexamined "analyzer skipped" state.

### Phase 6 — Per-version template completeness (FR-5)
- Enumerate packet families with Atlas code but no per-version routing — notably
  JMS NPC-shop (`NPCShopHandle`/`NPCShopOperation`) and the mini-room
  player-interaction family beyond the two ops task-080 wired.
- Per family: wire the per-version op-byte map (IDA-confirmed, like task-080
  B5.1f) in `services/atlas-configurations` seed templates so the family routes
  per version, OR record a verified client-absent verdict.
- **Threshold (PRD open Q):** if a family's full routing is itself a large effort,
  split it into a follow-up task (FR-3.4 discipline) rather than bloat this task.
- Validate every edited template parses (`python3 -m json.tool`).
- **Gate:** edited templates parse; audit reflects newly-routed families; no
  unrouted family left without a verdict.

### Phase 7 — Ledger + guide (FR-6)
- Re-curate `docs/packets/ida-exports/_pending.md` and
  `docs/packets/audits/gms_v95/_pending.md`: the "export read-order
  truncation/mistrace" category is **eliminated** (rows are now fixed,
  verified-exclusion, or explicit `unresolved` markers). Only *verified*
  exclusions remain; zero unexamined opaque skips.
- Update `docs/packets/audits/gms_v95/TOTAL.md` (verdict roll-up + completeness
  statement) and `STARTING_A_NEW_VERSION_PASS.md` (document the fixed exporter's
  descent behavior + the new `packet-audit export` workflow + the
  `unresolved`-over-guess invariant).
- **Gate:** zero "export was wrong" entries; both `_pending.md` copies contain
  only verified exclusions.

### Phase 8 — Verify + code review
- All CLAUDE.md gates: `go test -race ./...`, `go vet ./...`, `go build ./...`
  clean in every changed module; `docker buildx bake atlas-<svc>` for every
  service whose `go.mod` was touched (at minimum `tools/packet-audit`'s module;
  plus `libs/atlas-packet` consumers if wire fixes landed); `tools/redis-key-guard.sh`
  clean; backend nesting-cap clean.
- Code review before PR: `superpowers:requesting-code-review` →
  `plan-adherence-reviewer` + `backend-guidelines-reviewer` (Go files changed).
- **Gate:** all green; audit written to `docs/tasks/.../audit.md`.

## 5. Testing strategy

- **Parser (P1):** TDD against committed Hex-Rays fixtures — CI runs without IDA.
  The four-version BuddyInvite fixed-struct trace is the canonical regression
  guarding the anti-BuddyInvite invariant.
- **MCP-HTTP client:** in-package fake transport for request/response shaping;
  live behavior validated by the maintainer during P2.
- **Wire fixes (P4):** every fix ships a per-version byte-level test as the
  oracle. A round-trip test alone cannot catch a wrong-but-symmetric
  encode/decode bug (memory `reference_packet_audit_tool_mechanics`).
- **Opaque types (P5):** byte-level tests are the oracle for verified exceptions.
- **Templates (P6):** `python3 -m json.tool` parse validation + audit re-run.

## 6. Data model / artifacts

No database entities. Artifacts touched:

- `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json` — fully
  re-exported (replaced), `unresolved` markers added where applicable.
- `docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/SUMMARY.md` +
  per-packet `.md`/`.json` — regenerated.
- `docs/packets/ida-exports/_pending.md` + `docs/packets/audits/gms_v95/_pending.md`
  — re-curated (truncation/mistrace category removed).
- `docs/packets/audits/gms_v95/TOTAL.md`,
  `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md` — updated.
- `tools/packet-audit/**` — new exporter (mcphttp, parse, runExport, schema,
  fixtures, tests).
- `libs/atlas-packet/**` — wire fixes + byte tests (P4/P5, as surfaced).
- `services/atlas-configurations/seed-data/templates/**` — per-version op-byte
  maps (P6, as surfaced).
- `docs/tasks/task-081-ida-export-reharvest/verdict-snapshot-080.md` — §4.7
  baseline.

All `docs/packets/**` artifacts are tenant-agnostic build/audit data.

## 7. Service impact

- **`tools/packet-audit`** — primary change (the exporter: MCP-HTTP client,
  parser, descent, unresolved markers; possibly analyzer/registry support for
  opaque decomposition in P5).
- **`docs/packets/`** — exports, audits, registries regenerated/re-curated.
- **`libs/atlas-packet`** — wire fixes for genuine divergences (P4/P5).
- **`services/atlas-channel`, `atlas-maps`, `atlas-cashshop`, etc.** — only if a
  surfaced wire fix requires a handler/producer/event change.
- **`services/atlas-configurations`** — per-version template op-byte maps (P6).

## 8. Key risks and mitigations

| Risk | Mitigation |
|---|---|
| **Hex-Rays parser recall** — mask-driven / indirect-dispatch packets resist static parsing. | Precision-over-recall: emit `unresolved`, hand-fill via `Delegate`/`Ref`. §4.7 hand-verifies every flip regardless. Tool value = truncation-catching + reproducibility, not 100% automation. |
| **task-080 not merged** — corrected baseline/snapshot absent in this worktree. | Phase 0 rebase is a hard, explicit prerequisite; gated before any exporter work. |
| **Phase 4 bug count unbounded.** | FR-3.4 escape valve: fix in-task if feasible, else dedicated follow-up task. |
| **MCP-HTTP endpoint LAN-only / no IDB loaded.** | By design export is maintainer-local; client errors loudly on unreachable endpoint; CI consumes checked-in JSON. |
| **Confident-but-wrong trace (the core failure mode).** | The §1.1 invariant: `unresolved` over guess; the BuddyInvite four-version regression test enforces it in P1. |
| **Large re-export diff hard to review.** | Deterministic ordering + per-function stability + the P2 structural-change summary so the *exporter* is auditable, not just trusted. |

## 9. Open questions resolved during design

- **Exporter implementation surface (PRD Q1):** resolved — a standalone Go binary
  (`packet-audit export`) speaking MCP-over-HTTP to IDA-MCP. No separate harvest
  script; no Claude-as-transport indirection (the endpoint is directly dialable).
- **Decompiler-label reliability (PRD Q2):** the parser captures labels
  best-effort but **does not gate correctness on them** — only op width + order
  are load-bearing, which the API-assisted approach reads reliably.
- **Opaque decomposition depth (PRD Q3):** decided per-type in P5 — extend the
  analyzer where statically modelable, else byte-test-backed verified exception.
- **Template completeness scope creep (PRD Q4):** P6 threshold — wire if the
  op-byte map is IDA-confirmable in-task; otherwise split into a follow-up.
- **Re-export blast radius (PRD Q5):** governed by the §4.7 triage gate; count is
  unknown until P3 runs, but every flip is individually dispositioned.

## 10. Acceptance (maps to PRD §10)

- Exporter descends into struct helpers + inlines reads (FR-1.1); four-version
  `OnFriendResult#Invite` re-export matches the hand-decompiled truth.
- Full read-orders, no truncation (FR-1.2); loop-vs-struct disambiguated
  (FR-1.3); unresolved functions emit explicit markers (FR-2.3).
- All four exports re-exported + committed with a reviewable structural-change
  summary (FR-2.1/2.2).
- Audit re-run; truncation/mistrace residual category eliminated (FR-3.1/3.2).
- Verdict-delta triage applied; every `✅`→`❌`/new non-`✅` flip dispositioned,
  zero rubber-stamped (FR-7).
- Every genuine divergence fixed with per-version byte tests + gates, or
  registered as a dedicated follow-up (FR-3.3/3.4).
- Opaque types decomposed-and-verified or byte-test-backed exception (FR-4).
- Templates completed or verified client-absent; all parse (FR-5).
- Both `_pending.md` contain only verified exclusions; `TOTAL.md` +
  `STARTING_A_NEW_VERSION_PASS.md` updated (FR-6).
- All CLAUDE.md verify gates pass; code review run before PR.
