# Per-Branch Verification — Design

**Task:** task-081-ida-export-reharvest (extension)
**Date:** 2026-06-09
**Status:** Approved (brainstorming), pending implementation plan

## Goal

Convert the ~450 unverifiable `#Mode` (branching-handler) entries into positively
*verified-or-flagged* results, and stand up the **case↔mode dispatch model** that later
lets a fresh client version be audited and integrated automatically.

This is the **first** of four capability gaps on the road to comprehensive (≈100%)
machine-verification of packet handlers/writers. Today only ~27% of wire shapes
(293 / 1097 across the four hand-done versions) are positively verified; 450 of the 508
`unverifiable` entries are branching handlers the tool cannot isolate per-case.

## Background / current state (established by investigation)

- **Baseline** `docs/packets/ida-exports/gms_<v>.json`: a `functions` map of 256 entries
  (118 are `#Mode`), each `{address, direction, calls}`. Keys are the `FName`
  (e.g. `CLogin::OnCheckPasswordResult#AuthLoginFailed`). **No `dispatch` selector is
  stored on any entry; hand `calls` carry no guard.**
- **`ExtractShape(f Fields, dispatch []Selector)`** (`internal/idasrc/extract.go`)
  isolates one branch's reads from a resolved (flattened) function by matching the
  parser-emitted `Guard` on each `FieldCall` against `Selector{Discriminator, Case}`.
  It already includes the pre-branch common-header reads (empty-guard prefix).
- **`validate`** (`cmd/validate.go`): for a `#Mode` entry, if `len(e.Dispatch)==0` or the
  selector matches nothing, it reports `ShapeUnverifiable` —
  `"per-mode shape not extractable (no usable dispatch selector)"`. **This is the 450.**
- **`parse.go`** emits guards for `switch`/`case` (`var == N`) and loops (`loop N`) only —
  **not `if/else` dispatch chains.** Confirmed by reading `reSwitch`/`reCase` and the
  guard-tracking comment block.
- **`InferDispatch` / `InferDispatchJoint`** (`internal/idasrc/infer.go`) already propose
  selectors; the joint (one-to-one) variant reached ~85% precision on high-confidence
  picks in the Jun-5 run, but those picks were never persisted to the committed baseline.

### Root-cause split of the 450

- **Switch-dispatched** handlers (most of the `CWvsContext` family: `OnPartyResult`,
  `OnGuildResult`, `OnMemoResult`, `OnInventoryOperation`, …) — the parser *already*
  emits their `var == N` guards, so they verify as soon as a selector exists.
- **If/else-dispatched** handlers (`OnCheckPasswordResult#*`, `OnSetAccountResult#*`, …) —
  need the new parser capability *and* a selector. Exact split to be measured during
  planning (avoid burning IDA calls now); both are in scope this pass.

## Decisions (from brainstorming)

1. **Selector provenance:** inference-first with an agent-confirmation gate (not pure
   hand-authoring, not unpersisted live inference).
2. **Confirmation model:** agent-confirms-via-IDA — for each low-confidence pick, the agent
   reads the live decompile, confirms/corrects the selector against the actual
   `switch`/`if` discriminator + case label, and escalates only genuine client-ambiguity
   to the human. This *is* the new-version onboarding loop.
3. **Verification semantics:** bijection (correctness **and** completeness) — verify matched
   modes' wire shapes *and* report client cases with no Atlas writer (missing-mode) and
   Atlas modes with no client case (extra-mode), with a per-version allowlist to suppress
   intentionally-unimplemented client cases.
4. **Sequencing:** build `switch` + `if/else` support together in one pass (not switch-first).

## Components

### 1. If/else guard emission — `internal/idasrc/parse.go` (riskiest)

Extend the guard tracker to recognize `if / else if / else` dispatch chains and emit the
**same** `var == N` guard grammar `ExtractShape` already consumes:

- `if ( x == N ) { … }` → reads inside carry guard `x == N`.
- `else if ( x == M ) { … }` → `x == M`.
- trailing `else { … }` → a **synthetic default-case guard**, mirroring the existing
  `switch default:` handling (so `ExtractShape`/`Selector` need a default representation —
  see component 2a).
- Compound `&&` conditions already compose via the existing guard-composition logic.
- **Bail to no-guard** on conditions the equality-chain model cannot represent (ranges,
  negations other than the final `else`, non-discriminator predicates). Such reads stay
  honestly `unverifiable` rather than being silently mis-extracted.

Heaviest test coverage in the plan: table-driven against **real Hex-Rays fixtures** pulled
from all four IDBs (the Phase-1.5 lesson — synthetic fixtures hid real-decompile gaps).

#### 1a. Default/else selector representation — `extract.go`

`Selector`/`ExtractShape` must match a trailing-`else`/`default` arm. Add an explicit
`Default bool` field to `Selector` (a default selector matches reads whose guard is the
parser's default-arm marker, not any `== N` clause). The parser emits a distinct
default-arm guard token for both `switch default:` and trailing `else`; `ExtractShape`'s
matcher recognizes it when `Selector.Default` is set. Chosen over a sentinel `Case` value
to avoid colliding with a real case constant.

### 2. Case-label-set enumeration — `parse.go`

Emit the discriminator's **full** set of case labels (independent of whether a case body
reads anything), exposed on the resolved `Fields` (e.g. `Fields.CaseLabels map[discriminator][]int64`
plus a default flag). Without this, a client case that dispatches but reads nothing is
invisible to the bijection check (component 5). Covers both `switch` labels and `if/else`
equality arms.

### 3. Persisted dispatch model — `dispatch` field per `#Mode` entry

Round-trip a confirmed `Dispatch []Selector` plus a provenance note on each `#Mode` entry in
`gms_<v>.json`. `export.go` already declares `Dispatch []Selector json:"dispatch,omitempty"`;
wire the baseline **loader and writer** so the field persists. The selector *is* the
case↔mode binding (e.g. `OnPartyResult#Invite → {Discriminator:"operation", Case:9}`).
Provenance note records `inferred-high-confidence` | `agent-confirmed` | `human-confirmed`
and (for confirmed) the IDA evidence (address + discriminator + case label).

### 4. Selector resolution + agent-confirmation — new `resolve-dispatch` subcommand

Keep `infer` a **pure proposer**; add a stateful `resolve-dispatch` that:

1. Loads the baseline; groups `#Mode` entries by base address.
2. `ResolveLive(base addr)` once per base handler; enumerates the client case-set
   (component 2).
3. Runs `InferDispatchJoint` → one-to-one case→mode assignments with confidence.
4. **Auto-accepts** high-confidence (≥ threshold) picks.
5. **Low-confidence** picks → an **agent-confirmation worklist** (markdown + structured
   JSON: FName, base address, candidate cases, live discriminator, proposed case). The
   agent reads the IDA decompile at that address, confirms/corrects, and only escalates
   genuine client-ambiguity to the human.
6. Writes confirmed selectors + provenance into the baseline (component 3).

This command, pointed at a fresh IDB, **is** the new-version onboarding primitive.

### 5. Bijection / completeness — extend `validate`

Per base handler:

- client case-set `C` (component 2) vs Atlas `#Mode` set `M` (baseline selectors).
- matched `(case, mode)` → verify wire shape (existing `ExtractShape` + `ValidateShape`).
- `C \ M` → **missing-mode** (client case with no Atlas writer — "build this").
- `M \ C` → **extra-mode** (Atlas mode with no client case — latent bug / dead mode).
- subtract the per-version **allowlist** (component 6).
- new report buckets: **verified / divergent / missing-mode / extra-mode / unverifiable**.

The roll-up line and markdown sections gain the two new buckets.

### 6. Allowlist — `docs/packets/audits/<version>/_unimplemented.json`

Committed, per-version `{FName, Case, reason}` records for intentionally-absent client
cases, so they don't resurface every run. Hand-maintained; the bijection check reads it and
moves matching `missing-mode` entries into an `allowlisted` (suppressed) tally that is
counted but not flagged.

## Data flow

```
baseline #Mode entries (+ confirmed dispatch)
  → validate: ResolveLive(base addr) once per address
  → enumerate client case-set C (switch + if/else case labels)
  → ExtractShape per selector → ValidateShape vs hand calls        [correctness]
  → diff case-set C vs mode-set M, minus allowlist                 [completeness]
  → report: verified / divergent / missing-mode / extra-mode / unverifiable
```

## Onboarding tie-in

`resolve-dispatch` (component 4) + the agent-confirmation IDA loop + bijection (component 5)
together form the auto-onboarding path: point them at a fresh IDB, they enumerate handlers,
propose case→mode bindings, the agent confirms against IDA, and bijection emits the
integrate worklist (what's missing). Built as a byproduct of fixing the four known versions.

## Testing & verification

- **Parser if/else + case-label enumeration:** table-driven unit tests with real Hex-Rays
  fixtures (switch, if/else-if, trailing else/default, nested, compound, unrepresentable →
  no-guard). Fixtures sourced from the live IDBs, not synthesized.
- **ExtractShape:** default/else-arm selector matching.
- **Bijection:** synthetic case-set vs mode-set with allowlist (missing/extra/allowlisted).
- **`resolve-dispatch`:** proposal → auto-accept vs worklist routing; baseline round-trip of
  the `dispatch` field + provenance.
- **End-to-end:** re-run `validate` on all four IDBs (ports 13337–13340); assert the
  `per-mode shape not extractable` bucket collapses and the missing/extra buckets populate.
  Remember the jms audit-dir naming quirk (`jms_v185`, not `gms_jms_185`).
- **Gates (CLAUDE.md):** `go test -race ./...`, `go vet ./...`, `go build ./...` on
  `tools/packet-audit`. Not a service → **no** `docker buildx bake`. No redis → redis-key-guard N/A.

## Out of scope (this pass — sequenced after)

The other three capability gaps from the comprehensive-verification roadmap:

- **Unverifiable long-tail:** delegate cycle (`CUIMessenger::OnEnter`, 15), demangled
  `Class::Method` helper-name resolution (~14 Unresolved spans), `ABSENT` → N/A
  classification (11), base-decompile failures (19), address hygiene (2).
- **Divergent loop/opaque-block/mask modeling (296):** making live extraction model the
  loops, opaque blocks, and stat masks the hand baseline already models.

These are tracked as follow-on phases within task-081, not this design.
