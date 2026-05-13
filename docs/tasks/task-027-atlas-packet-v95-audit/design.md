# Atlas-Packet v95 Audit & Wire Correctness — Design

Version: v1
Status: Proposed
Created: 2026-05-13
PRD: `prd.md`

---

## 1. Design Goals

Translate the PRD into concrete architecture decisions for the audit pipeline, the encoder pattern, the template-config extension, and the phasing artifacts. The constraints driving most decisions:

- **The pipeline must run without IDA Pro** (CI, contributors without a Windows licence) yet still produce identical reports as the IDA-MCP path. Forces a stable intermediate JSON format.
- **Atlas-packet encoders are public API for two atlas services** (atlas-login, atlas-channel) and the public-facing shape is the `Encode`/`Decode` body, not a separate schema. Rules out any "generate encoders from a spec" rewrite — too disruptive for a correctness pass.
- **Version-conditional logic already works** (`auth_success.go` uses it). The bar for any new abstraction is "demonstrably better than inline `if t.MajorVersion() >= N`" — not "shinier".
- **The audit pass is the slow path; fixes are the fast path.** Optimize developer ergonomics for "read the report, edit the encoder, rerun the audit" — not for one-shot generation.
- **No regression on v83.** The library currently serves v83 cleanly; every change must preserve that and the round-trip suite is the proof.

---

## 2. Architecture Overview

```
┌──────────────────────────────┐    ┌──────────────────────────────┐
│ docs/packets/                │    │ libs/atlas-packet/**/*.go    │
│   *.csv (FName↔opcode)       │    │   (current Encode/Decode)    │
│   ida-exports/<region>_v*.json│   └──────────────┬───────────────┘
│   spike-login-v95.md          │                  │
└────────────┬─────────────────┘                   │
             │                                      │
             ▼                                      ▼
┌──────────────────────────────────────────────────────────────────┐
│ tools/packet-audit/                                              │
│                                                                  │
│  ┌────────────┐  ┌────────────┐  ┌──────────────┐  ┌──────────┐  │
│  │ csv        │→ │ template   │→ │ ida source   │→ │ atlas    │  │
│  │ parser     │  │ parser     │  │  resolver    │  │  packet  │  │
│  │            │  │            │  │ (mcp│export) │  │ analyzer │  │
│  └────────────┘  └────────────┘  └──────────────┘  └────┬─────┘  │
│                                                          │        │
│                                                          ▼        │
│                                                  ┌──────────────┐ │
│                                                  │ diff engine  │ │
│                                                  └──────┬───────┘ │
│                                                         │         │
│  ┌────────────────────────────────────┐                ▼          │
│  │   reports/                         │  ┌────────────────────┐   │
│  │     <writer>.md  + <writer>.json   │← │   report writer    │   │
│  │     _substruct/<Name>.md           │  └────────────────────┘   │
│  │     SUMMARY.md / SUMMARY.json      │                            │
│  └────────────────────────────────────┘                            │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
              ┌─────────────────────────────────────┐
              │ docs/packets/audits/<region>_v<n>/  │
              └─────────────────────────────────────┘
```

### Data flow

1. **CSV parser** loads the two CSVs once, building a bidirectional map `FName ↔ {version → opcode}` plus direction (clientbound / serverbound).
2. **Template parser** loads one `template_<region>_<major>_<minor>.json`, building per-opcode lookups for writers (clientbound) and handlers (serverbound). The template's `clientVariant` flag (default `"modified"`) propagates into the run config.
3. **IDA source** resolves `FName → ordered primitive field list`. Two implementations behind a `FieldSource` interface:
   - `MCPSource` — calls `mcp__ida-pro__*` tools (`get_function_by_name`, `decompile_function`) live.
   - `ExportSource` — reads `docs/packets/ida-exports/<region>_v<major>.json`, the canonical artifact for CI and offline contributors.
4. **Atlas-packet analyzer** does Go-AST static analysis of `libs/atlas-packet/**/*.go` to extract, per `Encode`/`Decode` method, a tree of (write-call, version-guard) tuples. This is the most novel part — see §3.
5. **Diff engine** runs the analyzer's branches for the *specific* tenant context the audit targets (e.g. GMS / 95 / modified), then aligns field-by-field against the IDA source list. Verdicts: `✅`, `⚠️`, `❌`, `🔍`.
6. **Report writer** emits markdown + JSON per packet and a SUMMARY.

The tool is **read-only** against `libs/atlas-packet/` — never writes Go files.

---

## 3. The hard part: extracting field lists from Atlas Encode/Decode bodies

This is the section where most candidate designs fall apart. Three approaches considered, recommendation last.

### Option A — Source-level regex / text scan

Run `regexp` over each `Encode`/`Decode` body, count `w.WriteByte`/`w.WriteInt`/`r.ReadX` calls in order.

- Pros: trivial to implement.
- Cons: cannot reason about `if t.MajorVersion() > 87 { ... }` branches at all. The whole point of the audit is to enumerate per-tenant variants. Rejected.

### Option B — Run the encoder, sniff the bytes

Build a minimal stub `tenant.Model`, call `Encode(...)(opts)`, decode the byte stream by walking known type widths.

- Pros: exact wire shape, no static-analysis bugs.
- Cons: byte widths alone don't disambiguate semantic field types (a `WriteInt(0)` and a `WriteByte(0) × 4` look identical post-emission). Also requires *every* model to be constructible with zero-value fixtures — many models have non-trivial sub-structs (CharacterStat, Asset) that need their own fixtures. Triggers a fixture-explosion problem. Rejected as the primary path; **kept as a verification cross-check** — see §3.4.

### Option C — Go AST walker with branch enumeration  ✅

Parse each file with `go/parser`, walk the `*ast.FuncDecl` for `Encode`/`Decode`. Within the closure body, collect:

- **Write/Read calls**: `w.Write{Byte,Bool,Short,Int,Long,AsciiString,Bytes}` / `r.Read{Byte,Bool,Uint16,Uint32,Uint64,AsciiString,Bytes}`.
- **Guards**: `*ast.IfStmt` conditions involving `t.MajorVersion()`, `t.Region()`, and (new) `t.ClientVariant()`. Parsed into a small predicate language: `regionEq("GMS")`, `majorGE(95)`, `majorLT(12)`, `variantEq("stock")`, plus AND/OR/NOT.
- **Sub-struct calls**: invocations of `<expr>.Encode(l, ctx)(opts)` or a `for ... { x.Encode(...)... }` loop — captured as a "recurse here" marker with the static type of the receiver.

Output per method: a tree where leaves are field writes annotated with a guard predicate (the conjunction of all enclosing `if` conditions), recursion markers, and a stable source position.

To produce a *specific tenant's* wire shape, evaluate every guard against `{region, majorVersion, minorVersion, clientVariant}` and flatten the tree to a list. To produce the *audit*, enumerate the cross-product of variants present in `test/context.go` plus `clientVariant ∈ {modified, stock}`.

### 3.1 Why AST walking is tractable here

The encoder body shape is highly regular by convention. From a scan of `libs/atlas-packet/**/*.go`:

- 100% of write calls are method calls on a single receiver (`w.WriteX(...)`) — no wrappers, no aliases.
- Version guards live at `if`-statement level, not in expressions; ternary-style usage is absent.
- Closure bodies don't define helper functions inline.

The narrow encoder dialect — under 20 distinct write methods, no metaprogramming — makes AST walking tractable. The analyzer is a glorified visitor with ~30 cases, not a Go semantic interpreter.

### 3.2 What about non-guard control flow?

Some encoders use `for` (loops over slices) and `switch` on a method argument. Strategy:

- **`for` over a model slice (e.g. channel loads):** the loop body is recursed into; the resulting field list is wrapped with a `repeat(count_field)` marker. Output: `(count_field, repeat(body_fields))`. IDA-side IDA loops show the same `do { decode_inner() } while (i < n)` shape and are normalized identically.
- **`switch`:** out of scope for v1 — fail loud with `🔍 manual review` if encountered. A grep of the current library shows ≤3 occurrences; cheap to hand-audit.
- **Early return** (e.g. `if resultCode != 0 { return }`): tracked as a guarded suffix; the diff engine reports v95 success-branch alignment, with non-success branches deferred for manual review.

### 3.3 Recursion into sub-models

The analyzer records `<typeName>.Encode` invocations rather than inlining them. The audit driver then schedules the sub-model as its own work item; recursion is iterative, not stack-bound. This keeps the per-packet report short and lets the recursive sub-struct audit (PRD 4.2) emerge naturally — every sub-model that is referenced ≥3× across the run gets its own report file, the rest are inlined into their (single) parent's report.

### 3.4 Round-trip cross-check

The Phase A exit criterion is "reproduces spike findings within tolerance". To get the *within tolerance* part formal, the tool's unit tests do both Option B and Option C on the spike's 6 packets and assert the field lists agree. This catches AST-analyzer bugs (e.g. misclassified guards) without us hand-eyeballing every report.

---

## 4. IDA source — live MCP and exported JSON

### 4.1 The interface

```go
// FieldSource resolves an IDA function (by name) to a primitive field list.
type FieldSource interface {
    Resolve(ctx context.Context, fname string) (Fields, error)
    Variants() []string  // e.g. ["GMS/v95"] — for reporting
}

type Fields struct {
    Function  string
    Address   string    // "0x5dc600"
    Direction Direction // clientbound | serverbound
    Calls     []FieldCall
}

type FieldCall struct {
    Op       Primitive    // ReadByte, ReadInt32, ReadString, ...
    Comment  string       // free-form label from IDA (e.g. "nNumOfChar")
    Guard    GuardExpr    // best-effort: "if (resultCode == 0)" → guard.eq("resultCode", 0)
}
```

### 4.2 MCPSource

Calls `mcp__ida-pro__get_function_by_name` to locate the entry, then `mcp__ida-pro__decompile_function` to fetch text. A small parser scans the decompile for `CInPacket::Decode*` / `COutPacket::Encode*` calls in source order. Guard expressions are best-effort — pulled from the enclosing `if` in the decompile text via lexical scan, not full C parsing. Anything ambiguous is recorded with `Guard: nil` and a `notes` field.

### 4.3 ExportSource

Persists the output of `MCPSource` as a checked-in JSON file:

```
docs/packets/ida-exports/gms_v95.json
{
  "binary": "GMS_v95.0_U_DEVM.exe",
  "md5":    "3c71fd8872d5efbe16183ae8c51f887d",
  "generated_at": "2026-05-13T...",
  "functions": {
    "?OnCheckPasswordResult@CLogin@@IAEXAAVCInPacket@@@Z": {
      "address": "0x5dc600",
      "direction": "clientbound",
      "calls": [
        {"op":"Decode1","comment":"resultCode"},
        {"op":"Decode1","comment":"post-auth flag"},
        ...
      ]
    }
  }
}
```

The MCP path *generates* this file via `packet-audit export --output docs/packets/ida-exports/gms_v95.json`. The CI / non-IDA path *consumes* it.

**Open question 2 resolution (proposed):** the export is checked into the repo, manually refreshed by whichever contributor runs the audit against a new IDA build. Refresh cadence is "whenever v95 IDA findings change" (rare). A `Makefile` target `make audit-export-v95` runs the export end-to-end given the MCP setup.

### 4.4 Trust model

The export file is the **canonical** IDA artifact for CI purposes. If MCP and the checked-in export disagree, the CI tool flags this as a verification failure — never silently trusts one over the other. Concretely: `packet-audit --verify-export` runs MCP + export side-by-side and diffs, intended as a maintainer pre-commit gate, not CI gate.

---

## 5. Encoder pattern — version conditionals

### 5.1 Decision: keep inline checks; add optional helper for *new* code

The PRD allows an optional `libs/atlas-packet/version/` helper. Recommendation:

- **Adopt the helper** for any encoder touched in this task. Three reasons:
  1. Mixed inline + helper styles already exist in nearby code — convergence is cheap.
  2. The helper makes `t.ClientVariant()` access uniform and prevents `if t.ClientVariant() == "stock"` literal-comparison errors (typos).
  3. `version.Between(t, 87, 95)` reads better than `if mv > 87 && mv <= 95` when both bounds matter (which is common in this audit).
- **Do not bulk-migrate untouched encoders.** Cost without payoff.

Proposed surface:

```go
// libs/atlas-packet/version/version.go
package version

import (
    tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Region string

const (
    GMS Region = "GMS"
    JMS Region = "JMS"
)

type ClientVariant string

const (
    Modified ClientVariant = "modified" // default
    Stock    ClientVariant = "stock"
)

func RegionOf(t tenant.Model) Region            { return Region(t.Region()) }
func AtLeast(t tenant.Model, n uint16) bool     { return t.MajorVersion() >= n }
func LessThan(t tenant.Model, n uint16) bool    { return t.MajorVersion() < n }
func Between(t tenant.Model, lo, hi uint16) bool {
    mv := t.MajorVersion()
    return mv >= lo && mv <= hi
}
func IsStock(t tenant.Model) bool { return VariantOf(t) == Stock }

// VariantOf reads the per-tenant clientVariant flag with a default of Modified
// when unset (back-compat for tenants that predate the flag).
func VariantOf(t tenant.Model) ClientVariant { ... }
```

`VariantOf` requires the tenant model to expose the flag — see §6. All comparisons are `uint16` to match `tenant.Model.MajorVersion()` directly.

### 5.2 Branch-nesting limit (PRD 4.3)

The PRD proposes 3 levels with a warning/error gradient. Recommendation:

- **Linter warning at 3, hard fail at 4.** The static analyzer in §3 already walks the if-tree; emitting a warning when depth ≥ 3 is one extra check and gives reviewers an explicit signal to consider a sibling-file split. Depth-4 is rejected outright.
- The audit pipeline output includes a `branch_depth: N` field per packet so we can monitor whether we're trending in the right direction.

### 5.3 When to split into a sibling per-version file

PRD 4.4 lists the "small (≤4 lines) → inline / larger → sibling file" rule. Reify with two concrete sub-rules:

- A whole-packet structural rewrite (e.g. stock-v95 `LoginHandle.Request`) lives in `request_stock.go` next to `request.go`, dispatched from a shared `Encode`/`Decode` entry point that reads `version.VariantOf(t)` once.
- A tail-only divergence (e.g. ≥95 widens one field) stays inline.

The sibling pattern keeps the wire-flat encoder body trivially diff-able against IDA per-variant, which is the workflow this whole task optimizes for.

---

## 6. Template & tenant integration

### 6.1 Template schema

Add `clientVariant` at the **root** of `template_<region>_<major>_<minor>.json`:

```json
{
  "region": "GMS",
  "majorVersion": 95,
  "minorVersion": 1,
  "usesPin": false,
  "clientVariant": "modified",
  "socket": { ... },
  ...
}
```

The flag is omitted (and treated as `"modified"`) on every existing template. No file is rewritten in this task except `template_gms_95_1.json`, which is set explicitly to `"modified"` to anchor the default. Stock-v95 ships either as a sibling template (`template_gms_95_2.json` with `clientVariant: "stock"`) or via a Phase F-specific template addition — deferred per PRD.

### 6.2 atlas-configurations changes

Touch points (mirroring `atlas-configurations/templates/rest.go:11`):

- `RestModel` gains `ClientVariant string \`json:"clientVariant"\`` (string, not enum — the validator below catches bad values).
- `entity.go` adds `ClientVariant string` and the GORM column. Migrate-on-startup is the existing pattern; the column is nullable / defaulted to `""` so existing rows don't break.
- `processor.go` `Extract`/`Transform` propagates the field; default `""` is normalized to `"modified"` at read time so wire format and DB are stable.
- `validation_error.go` adds a single rule: `ClientVariant ∈ {"", "modified", "stock"}`.
- Mock (`configuration/mock/processor.go` if it exists for templates — verify during planning) updates.
- `templates/rest_test.go` and `processor_test.go` get one fixture covering each variant.

### 6.3 tenant.Model surface

`tenant.Model` needs a `ClientVariant() string` accessor for the encoders to consume. Two implementations options:

- **(a)** Add a field to `tenant.Model` in `libs/atlas-tenant`, populated from the template/config when atlas-tenants seeds the tenant.
- **(b)** Embed in tenant config blob (a `map[string]string` already exists for arbitrary tenant settings — if it does; verify in planning).

Recommendation: **(a)**. The flag is wire-critical, not a generic config knob; making it first-class on `tenant.Model` makes the encoder code path explicit and the static analyzer's job easier (it scans for `t.ClientVariant()` directly).

A backing-store migration is required in atlas-tenants. The default-to-`"modified"` rule keeps that migration safe — old tenant rows continue to work.

### 6.4 atlas-login adapter

Per PRD 7, atlas-login may need an adapter when stock-v95 field semantics differ. Concretely: stock-v95 `LoginHandle.Request` carries a `passport` token; the existing service contract expects `password`. Approach:

- `request.go` (or the sibling `request_stock.go`) exposes both `Password() string` and `Passport() string` methods on the model; one is `""` depending on variant.
- atlas-login's request adapter (`atlas-login/atlas.com/login/.../adapter.go` — exact path to confirm in planning) reads `version.VariantOf(t)` and dispatches to the correct downstream call.
- The downstream Nexon-passport validation flow is **stubbed in this task** (`return errors.New("nexon passport validation not implemented")`) and split to a sibling task — see §9.

---

## 7. Reports — what gets emitted, where, how it's used

### 7.1 Per-packet markdown

Modeled on `docs/packets/spike-login-v95.md`'s per-packet sections. Skeleton:

```
# <WriterName> (← <IDA function FName>)

- **IDA:** <addr>, function symbol
- **Template:** <writer>, opcode <op>
- **Atlas file(s):** path:line
- **Branch depth:** N
- **Verdict:** ✅ | ⚠️ | ❌ | 🔍

## v95 wire layout
| # | Type | Field | Notes |

## Atlas diff (variant: GMS/v95/modified)
| # | Atlas writes | v95 reads | Verdict |

## Drift summary
| Severity | Issue | Recommendation |
```

### 7.2 JSON sidecar

Each packet's report has an `.json` sibling containing the structured field lists for both sides, the diff records, and the verdict. Schema is intentionally rough-but-stable; consumed by a `SUMMARY.json` aggregator.

### 7.3 SUMMARY

`docs/packets/audits/<region>_v<major>/SUMMARY.md`:

- Sorted lists by verdict.
- Linked per-packet reports.
- Counts by domain (login, character, …).
- Drift trend (compared to previous SUMMARY.json if present — i.e. PR check).

### 7.4 CI gate

Phase A delivers the tool; CI integration is its own item:

- The tool runs on every PR touching `libs/atlas-packet/**` or `docs/packets/ida-exports/**`.
- Exit codes: per PRD §5.1.
- **Diff-mode**: compare current SUMMARY.json against `main`'s. A PR that introduces a NEW ❌ entry fails CI; one that flips a ❌ to ✅ succeeds. Pure ⚠️ count changes are non-blocking but surfaced in the PR comment.
- The CI run uses `ExportSource` (no IDA dependency).

---

## 8. Phasing — concrete artifacts

The PRD's Phase A–F is preserved. Adding *concrete artifacts* per phase so plan-task has a starting point:

### Phase A — Tooling foundation
- `tools/packet-audit/` skeleton with `main.go`, `cmd/`, `internal/` packages.
- `internal/csv`, `internal/template`, `internal/atlaspacket` (Go-AST analyzer), `internal/idasrc` (interface + MCP + Export impls), `internal/diff`, `internal/report`.
- `tools/packet-audit/README.md`.
- Fixtures: `internal/<pkg>/testdata/` per package.
- `docs/packets/ida-exports/gms_v95.json` (initial export).
- Phase A exit: `packet-audit --template template_gms_95_1.json --ida-source docs/packets/ida-exports/gms_v95.json --output /tmp/v95` produces reports for the spike's 6 packets matching the spike report's verdict column.

### Phase B — Login domain
- `libs/atlas-packet/version/` helper package (§5.1) + tests.
- Fixes for `auth_success.go` field-7 width and field-3 region gate.
- Fix for `server_list_entry.go` per-channel world-id hardcoded `1`.
- Renames per PRD 4.5 (cosmetic field labels). Per **open question 4**: ship renames in the same PR as the audit-report it appears in — never a bare rename PR. This pins the reason-for-change to the audit evidence.
- Full audit report set under `docs/packets/audits/gms_v95/` for every login packet.
- `libs/atlas-packet/login/**` round-trip tests pass for all `Variants` in `test/context.go`.

### Phase C — Sub-structs
- Sub-struct audit for `CharacterStat`, `AvatarLook`, `Asset`, `ChannelLoad`, `AttackInfo`, plus pipeline-flagged additions.
- Reports under `docs/packets/audits/gms_v95/_substruct/`.
- Fixes where applicable; flag any cross-domain ripple in the report.
- Round-trip tests for each sub-struct.

### Phase D — Channel clientbound
- Per-domain sweeps (character, inventory, monster, drop, field, pet, reactor, quest, party, guild, buddy, chat, messenger, note, merchant, interaction, fame, storage, cash, ui, socket).
- One audit-report sub-PR per domain to keep review tractable.

### Phase E — Channel serverbound
- Same sweep, handler-side.

### Phase F — Stock-Nexon variant (deferred unless small)
- `version.IsStock` already in tree from Phase B; Phase F implements the encoder variants and the atlas-login adapter dispatch.
- **Recommendation:** split into a sibling task (`task-NNN-atlas-packet-stock-nexon-v95`) unless Phase A–E ship with budget remaining. The reason: stock-Nexon requires Nexon-passport validation, which has no prior integration in Atlas, and the threat-modelling burden (PRD 8.2) is non-trivial. Keep this task focused on the audit pipeline + modified-v95 wire correctness — that's already a large surface.

---

## 9. Resolution of PRD open questions

The PRD's §9 open questions get the following design-time answers (planning may revise):

1. **Stock-Nexon passport validation** — split to a sibling task. This task delivers (a) the `clientVariant` flag, (b) the encoder slot in `request.go` / `request_stock.go`, (c) a stub validator that rejects every passport. The sibling task delivers the Nexon backend integration + real validator.
2. **IDA-MCP availability for CI** — checked-in export under `docs/packets/ida-exports/<region>_v<major>.json`, refreshed by the contributor running an audit cycle against a new IDB. A `Makefile` target documents the refresh command.
3. **Overlap with `legacy-atlas-packet-improvements` Phase 4** — coordinate during planning; the audit pipeline's `🔍 manual review` verdict naturally captures the no-op-decode packets without forcing a fix in this task. If `legacy-atlas-packet-improvements` lands first, those packets will report ✅; if it lands second, this task's reports document the gap.
4. **Field rename strategy** — bundle renames with the audit-pass PR that surfaces them. Never a bare rename PR. Reason: keeps the rename anchored to a citable audit evidence chain.
5. **Test-context expansion** — non-v95 findings are committed as informational-only (verdict `🔍 informational`). No follow-up issues filed by default.
6. **Audit-pipeline placement** — `tools/packet-audit/` per existing convention.
7. **Branch-nesting threshold** — warning at depth 3, hard fail at depth 4 (§5.2).

---

## 10. Tradeoffs and rejected alternatives

### 10.1 Encoder pattern alternatives

- **One file per (packet × version)** — e.g. `auth_success_v83.go`, `auth_success_v95.go`. Rejected: explodes file count for what are usually 1–2 line widenings, splits the encoder logic in ways that make whole-packet review impossible.
- **A spec file per packet (YAML/JSON) compiled to Go** — rejected: too invasive for a correctness pass, requires a new build step, and the encoder bodies are short enough that a spec file is no more compact than the current Go.
- **Reflect-on-tagged struct (`pkt:"byte"`)** — rejected: every encoder would need rewriting, runtime reflection penalty on hot paths, version conditionals would need a DSL anyway, and the inevitable struct tag for "conditional on v ≥ 95" reinvents the current `if` statements.

### 10.2 Audit pipeline alternatives

- **Hand-roll one report per known packet, never re-audit** — rejected because that's the current state. The PRD's reusability goal demands a tool.
- **Defer pipeline to Phase B+** and hand-audit Phase B login fixes — rejected because the spike already proved the manual approach for 6 packets; investing in the tool earlier pays off as soon as Phase C sub-structs land (≥3-referenced types need cross-domain visibility).
- **Use a third-party diff tool** — there isn't one for this domain. Rejected as a starting point.

### 10.3 `clientVariant` location

- **Per-tenant attribute (chosen)** — variant is wire-critical, must propagate to every encoder via context, easiest with first-class field.
- **Per-socket override** — e.g. one Atlas instance serves both modified and stock on different ports. Rejected as YAGNI; nothing in the PRD requires it and a single tenant per socket is the standing assumption.
- **Build-time flag** — rejected; precludes a single Atlas binary serving both variants concurrently for different tenants, which is an explicit non-functional requirement (PRD 8.4).

### 10.4 Sub-struct recursion stop condition

The analyzer recurses into any `<Type>.Encode` / `<Type>.Decode` call where `Type` is defined inside `libs/atlas-packet/**`. Calls into `atlas-socket/response`/`request` (the `w`/`r` receiver) are leaves. This is the simplest stop rule that captures every sub-model without dragging in unrelated dependencies.

---

## 11. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| AST analyzer misclassifies a complex guard (e.g. `&&` of three conditions) and reports wrong variants | Medium | High — false ✅ would mask a v95 bug | Phase A cross-check via Option B byte-sniffing (§3.4). Hard fail in CI if Option B and Option C disagree for the same fixture. |
| IDA export drifts from MCP because contributors forget to refresh | Medium | Medium — stale CI gate | Maintainer-only `make audit-export-v95` + a `--verify-export` mode; export staleness becomes a maintainer responsibility, surfaced as a SUMMARY.md banner if the export's `generated_at` is older than 30 days. |
| Tenant-flag schema change in atlas-configurations / atlas-tenants requires migrations that block deploys | Low | Medium | Nullable column + default-to-`"modified"` at read time. No data migration required; new column lands empty. |
| Phase B touches enough files that round-trip tests can't keep up | Medium | Medium | Round-trip iteration over `test/context.go` becomes a single `t.Run` per variant — keeps test time bounded; planning should split Phase B into per-packet sub-tasks if a single PR exceeds ~10 encoder edits. |
| Nexon passport validation surface gets pushed in despite §9 recommendation | Low | High — pulls in legal/threat-modelling without budget | Hold the line on splitting Phase F unless backlog allocates for it. |
| The encoder for a packet has both v95 widening AND a stock-v95 rewrite, forcing both `clientVariant` and `MajorVersion` guards in one body | Medium | Low | Existing `if` chain handles this; static analyzer treats them as independent guard axes. If branch depth hits 3, the linter forces a sibling-file split per §5.2. |

---

## 12. Out of scope (explicit)

These are noted to anchor scope and prevent slippage:

- Atlas business-logic refactors beyond the v95 wire bridge.
- WZ data updates.
- Client binary distribution / patching.
- A general-purpose packet definition DSL.
- Audit-driven fixes for v87/v92/v111/JMS v185 wire shapes (reports only, per PRD).
- The Nexon-passport backend integration (split to sibling task).
- Removing the `Variants` array in `test/context.go` (we add to it, don't refactor it).

---

## 13. Reference points in the existing tree

- `libs/atlas-packet/login/clientbound/auth_success.go:37-94` — canonical version-conditional encoder body.
- `libs/atlas-packet/login/clientbound/server_list_entry.go:46-87` — pattern for sub-model loop (`for _, x := range m.channelLoads`).
- `libs/atlas-packet/test/context.go:18-23` — `Variants` array; round-trip iteration target.
- `libs/atlas-packet/test/roundtrip.go:12-24` — leftover-byte assertion; the "wire complete" definition.
- `services/atlas-configurations/atlas.com/configurations/templates/rest.go:11-22` — template REST model to extend.
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — first template to anchor `clientVariant: "modified"`.
- `docs/packets/spike-login-v95.md` — single source of truth for the spike findings the audit pipeline must reproduce.

---

## 14. What plan-task should do next

The plan should break Phase A into ≤12 sequential implementation tasks (each a small PR) and define Phase B as a per-packet matrix (Atlas file × IDA function × audit-report path). Phases C–E are scoped at task time, not now — the Phase A tool determines the actual workload. Phase F deferral is a planning checkpoint, not an execution item here.

Specifically the plan should answer:

- The exact AST-walker API in `tools/packet-audit/internal/atlaspacket/` (it's the load-bearing component — needs design-level detail).
- The minimal change set on `tenant.Model` / atlas-tenants schema to land `ClientVariant()` without breaking existing services.
- Round-trip test extension shape: a `for _, v := range test.Variants { t.Run(v.Name, ...) }` wrapper, or an explicit per-variant test function. Lean toward the loop for compactness.
- Whether to land the `version/` helper as Phase A's last sub-task or Phase B's first — it gates almost all Phase B PRs.
