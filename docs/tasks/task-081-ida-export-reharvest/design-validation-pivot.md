# Task-081 — Validation pivot design (supersedes the "replace/re-audit" Phases 2–7)

> Status: design. Folds the per-mode validation approach into task-081. The
> original plan's Phases 2–7 ("re-export the four baselines + re-audit") are
> **abandoned** based on an empirical measurement (below). This document defines
> what replaces them.

## 1. Why the pivot (empirical evidence)

Phases 0/1/1.5 built and hardened a real automated exporter (MCP-HTTP client,
direction-aware alias-tracked decompile parser, struct descent, honest
`Unresolved`). A live v83 audit-delta measurement then settled the core question:

- Built a merge (hand-authored authoritative; overlay the exporter's resolved base
  entries + descended helpers), re-ran the audit, diffed verdicts vs the
  hand-authored baseline.
- **Result: overlaying the exporter REGRESSES the audit — 26 packets ✅→❌ vs only
  3 ❌→✅** (169✅→146✅).

Root cause is **structural, not a bug**: the exporter parses a whole function and
flattens every switch/variant branch into one sequence. The hand-authored baseline
decomposes a single switch handler into per-wire-shape entries via synthetic
`#`-mode FNames (e.g. `CWvsContext::OnFriendResult#Invite` = case-9 reads). The
audit keys on the `#`-mode names. A flattened whole-function read cannot match
Atlas's per-variant encoder, so overwriting good per-mode data with it manufactures
false blockers. The prior export also uses **zero `Delegate` refs** (all reads
inline), so the exporter's 82 auto-decomposed structs are unreferenced when compared
raw.

**Conclusion:** a fully-automated exporter cannot *replace* the carefully hand-traced
baseline. But it can **verify** it — which is what task-081's PRD actually wanted
("turn the audit from trusted-but-input-limited into genuinely verified"). The pivot:
**use the exporter to VALIDATE the hand-authored per-mode baseline against the live
IDB, never to replace it.** A validator can only *find* problems, never regress the
audit.

## 2. The key enabler is already built

The parser already tags every read with its switch/case guard (`switch == 9`,
`mode == 1 && loop count`, …). So the exporter *already* produces the per-case
decomposition internally; flattening just concatenates the cases. The only missing
piece is the mapping from a synthetic `#Mode` suffix to its case/dispatch path
(today that mapping lives implicitly in the hand-author's head + the per-mode reads).

Everything else reuses Phases 0/1/1.5 unchanged: the MCP-HTTP client (session-id,
soft-fail), the direction-aware alias-tracked parser, descent + `Unresolved`, the
`resolveWithVisited` Delegate splicer, and `diff.widthEquivalent` tolerance.

## 3. Design

### 3.1 Dispatch-selector schema (the small, durable human annotation)

Extend the export/baseline entry schema with an optional ordered `dispatch` path:

```json
"CWvsContext::OnFriendResult#Invite": {
  "address": "0xa3f2e8",
  "direction": "clientbound",
  "dispatch": [{ "discriminator": "switch", "case": 9 }],
  "calls": [ /* hand-authored reads, unchanged */ ]
}
```

- `dispatch` is an ordered list of selectors (one per nesting level) — supports
  nested switch → sub-switch (mode → sub-mode). Each selector names the discriminator
  kind and the case value.
- Clientbound switch handlers select by `case N` (the parser's `switch == N` guard).
- Serverbound multi-send (a `Send*` function that builds different `COutPacket`s on
  different branches) needs a selector keyed on the send branch — modeled as
  `{ "discriminator": "send", "op": "0x2A" }` (the leading op byte the branch
  encodes) or `{ "branch": "<guard-label>" }`. Exact form confirmed against real
  serverbound shapes in V1/V4.
- The selector is the only new human input; the *reads* are derived from the live
  IDB and verified. The `calls` array stays as the trusted reference.

### 3.2 Per-shape extraction

`ExtractShape(baseFn parsed, dispatch []selector) -> []FieldCall`:
- Resolve the base function (decompile → direction-aware parse → resolve Delegates).
- Walk the resolved reads, keeping only those whose composed guard satisfies every
  selector in the dispatch path (e.g. guard contains `switch == 9`); drop the
  discriminator read and reads of other cases.
- Compose with dispatcher prefixes (`per-mob`/`per-pet`) already handled by the
  resolver.
Result: the per-wire-shape read sequence the audit would compare — derived from the
live IDB.

### 3.3 Validate-diff (audit-grade tolerance)

`ValidateShape(handAuthored []call, extracted []FieldCall) -> verdict`, reusing
`diff.widthEquivalent` + loop-guard handling so:
- `DecodeBuf ≡ Decode4` (same bytes), composite-run equivalence, opaque-buffer
  equivalence → **MATCH** (not a finding).
- Loop-body reads with count==0 → not a phantom.
- Only genuine order/width divergence, missing trailing field, or extra field →
  **DIVERGE** (a finding).
Verdicts: `verified` | `divergent(detail)` | `unverifiable(reason)` (undecompilable
→ honest `Unresolved`, indirect dispatch, or no usable selector).

### 3.4 Auto-inference bootstrap (avoid hand-annotating 118×4 entries)

For each existing `#`-entry, propose its dispatch selector automatically:
- Decompile the base function; enumerate each case's resolved read sub-sequence.
- Score each case against the hand-authored reads using audit-grade equivalence
  (exact-with-tolerance; tie-break by case label / address proximity).
- High-confidence single match → propose `dispatch: [{case: N}]`.
- Ambiguous / no match / multi-case → flag for human review (a small reviewable list).
Output: a proposed selector map per version → human confirms → baked into the schema.
After this one-time bootstrap, re-validation is fully automatic.

### 3.5 Report

Per version (`docs/packets/validation/<version>.md` or similar):
- `verified`: count + list (the bulk — confirms the baseline against the live IDB).
- `divergent`: each with hand vs live read-orders + IDA address, triaged into
  hand-tracing-error / real-Atlas-relevant / representation (auto-resolved).
- `unverifiable`: with reason (undecompilable / indirect / no selector).
Roll-up: "X/Y wire shapes verified against live IDB; Z divergences; W unverifiable."

### 3.6 What gets committed (and what does NOT)

- **Committed:** dispatch-selector annotations on baseline entries (small, durable,
  enables future re-validation); the validation reports.
- **NOT committed:** any replacement of hand-authored `calls` (they stay
  system-of-record). The exporter never overwrites trusted reads.
- **Acted on:** a `divergent` row that is a *real Atlas bug* → fixed per the original
  plan's Phase-4 discipline (per-version byte test as oracle). A `divergent` row that
  is a *hand-tracing error* → correct that one baseline entry, citing the live IDB
  address as evidence. Representation diffs → recorded, no action.

## 4. Why this is net-positive by construction

- **Branch-flattening disappears** — extraction is per-case, so the sequence matches
  Atlas's per-variant encoder. The 26 ✅→❌ regressions were purely an artifact of
  overwriting per-mode data with a flattened read; validation never overwrites.
- **Struct decompositions get consumed** — validation compares *resolved* reads, so
  `Delegate→GW_CharacterStat::Decode` splices to the same inline fields the
  hand-author wrote and they match. (They looked "unreferenced" only in the raw,
  unspliced comparison.)
- **A diff can only find problems, never regress the audit.**
- **Delivers the PRD goal** — every hand-authored per-mode entry becomes
  *checked against the live IDB*, and the selector annotations shrink future manual
  tracing (new modes/versions) to "name the case," not "trace every byte."

## 5. Implementation tasks (the new sub-plan — to be detailed in plan.md)

- **V1 — Per-shape extraction.** `ExtractShape(parsed, dispatch)` filtering resolved
  reads by dispatch selector. TDD on `testdata/real_onfriendresult_v83.c`: extract
  `case 9` → `[Decode4, DecodeStr, Delegate→sub_A40028]` (then resolved chain).
- **V2 — Dispatch-selector schema.** Add the optional `dispatch` field; resolver/
  extractor honor it; serverbound `send`-selector shape confirmed against a real
  `Send*` fixture.
- **V3 — Auto-inference matcher.** Best-match case for a hand-authored read list with
  audit-grade tolerance + confidence/ambiguity handling. TDD.
- **V4 — `packet-audit validate` command.** Per-entry extract + diff + report; honors
  soft-fail/`Unresolved`; deterministic report output.
- **V5 — Bootstrap selectors (live, maintainer-cycled IDBs).** Auto-infer over the
  four baselines → human-confirm the ambiguous ones → commit selector annotations.
- **V6 — Validate the four versions (live).** Produce reports; triage every
  `divergent`: real-bug → Phase-4 byte-test fix; hand-error → correct baseline entry
  (IDA evidence); representation → noted.
- **V7 — Ledger + docs.** Record that the `#`-mode entries are now live-verified;
  update `STARTING_A_NEW_VERSION_PASS.md` with the validate workflow + selector
  annotation procedure.

## 6. Open questions (resolve during planning / against real shapes)

- Serverbound multi-send selector form (op-byte vs guard-label vs send-index).
- Nested dispatch (mode → sub-mode) extraction against a real nested case.
- Auto-inference ambiguity when two cases share an identical read shape (fall back to
  human + case-label/address hints).
- Composition with existing dispatcher-prefix entries (`per-mob`/`per-pet`).
- Whether validation runs all four versions in V6 or starts with v83 as a proof, then
  scales (recommend: v83 proof → confirm net-positive report → scale).

## 7. Relationship to existing artifacts

- Reuses Phases 0/1/1.5 (committed) wholesale; adds extraction + selector schema +
  inference + validate command on top.
- Supersedes plan.md Phases 2–7 (replace/re-audit). The verdict snapshot
  (`verdict-snapshot-080.md`) and the Phase-1.5 hardening spec remain valid context.
- The exporter's existing `export` subcommand stays (useful for bootstrapping a
  brand-new version that has no hand-authored baseline); `validate` is the new
  primary mode for the existing four baselines.
