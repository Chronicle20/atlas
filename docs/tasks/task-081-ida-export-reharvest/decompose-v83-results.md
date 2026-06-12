# Task-081 — `decompose` targeted mode: v83 results

Built the `decompose` subcommand (commits `11f63a7a7` + `bc2d904f4`) and ran it live on
v83 to answer "how many of the non-✅ packets can we auto-resolve?".

## What `decompose` does
For each NON-✅ packet in a version's committed audit, look up its function, run the
live exporter (`ResolveLive`, decompile-by-address + struct descent) to get the
faithful read-order, and:
- **`upgraded`** — ONLY when the faithful order is a strict PREFIX-EXTENSION of the
  hand-authored order (unambiguous truncation: the export stopped short). Replace that
  entry's `calls` with the faithful order in an extended baseline.
- **`divergence`** — any mid-stream difference (`fieldEquivalent` false within the
  common prefix, or hand longer than faithful). FLAGGED for human triage, NEVER
  overwritten (so a real Atlas wire bug can't be silently hidden by the client shape).
- **`unchanged`** / **`needs-dispatch`** (`#` per-mode) / **`error`** (decompile fail).
The input baseline is read-only; only `upgraded` entries change in the output.

## v83 run (port 13337)
Classification of the ~83 non-✅ packets:
`upgraded 11 / unchanged 4 / divergence 28 / needs-dispatch 34 / error 6`.

Re-audit of the extended baseline vs the committed baseline:
- baseline: `80 ❌ / 169 ✅ / 4 🔍 / 1 ⚠️`
- decomposed: `83 ❌ / 170 ✅ / 1 ⚠️`
- **CLEARED (non-✅ → ✅): 1** (`Request`). **REGRESSED (✅ → non-✅): 0.** The 4 🔍 shifted to ❌.

## Why the yield is low (the bottleneck)
**8 of the 11 `upgraded` faithful orders contain `Unresolved` spans.** The new
ida-pro-mcp server resolves addresses / `sub_XXXX` / mangled names but returns
"Not found" for demangled `Class::Method` helper names — so named-helper descent
bottoms out in `Unresolved`, and the re-audit renders those reads as ❌/🚫 instead of
✅. The 3 clean upgrades mostly still don't match Atlas exactly (the truncation
extension surfaced reads that genuinely differ, i.e. candidate findings).

## Takeaways
- The mechanism is **correct and safe** (0 regressions; truncations upgraded only on
  provably-safe prefix-extension; real divergences flagged, never overwritten).
- Its **effectiveness is gated on the same recall lever** as validation:
  **resolve demangled `Class::Method` helper names** (mangle-and-retry or a name
  search) → clean faithful orders without `Unresolved` → far more `upgraded` entries
  would actually clear. This is the registered follow-up.
- The immediately-actionable value is the **28 flagged `divergence` candidates** —
  these are concrete hand-vs-live read-order disagreements to triage in IDA (real Atlas
  bug → byte-test fix; hand-tracing error → corrected baseline). The auto-clear count
  (1) understates the tool's value; the divergence list is the real output.

## Honest answer to "how many does it clear?"
On v83, **1 packet auto-clears today** — but the IMPORTANT finding is WHY the rest don't.

## Re-run after the demangled-name fix (callee-based descent) — hypothesis revised
After fixing `ResolveLive` to descend demangled helpers via `callees`+demangle (commit
`1cca19a32`), the faithful read-orders are clean (few `Unresolved`), yet decompose STILL
clears only 1 packet (`upgraded 12 / divergence 26 / needs-dispatch 34 / error 6`;
re-audit 1 cleared, 0 regressed). Extending the export to the FULL faithful client
read-order does NOT clear the "atlas extra" rows.

**Why: most non-✅ packets are GENUINE Atlas-vs-client divergences, not export
truncation.** Worked example — `DropDestroy` (← `CDropPool::OnDropLeaveField`):
- Exporter's faithful client read-order: `[Decode1, Decode4, Decode4, Decode2, Decode1]`
  — **verified byte-for-byte against the live IDA decompile** (the client really reads
  exactly these 5 fields; the exporter is correct, not mis-parsing).
- Atlas's `DropDestroy` encoder writes `[Encode1, Encode4, Encode4, Encode2, Encode4,
  Encode4]` (6 fields). Re-audit row detail: position 4 = atlas `Encode4` vs client
  `Decode1` (**width mismatch**); position 5 = atlas `Encode4` the client never reads.
- So Atlas writes more/wider than the v83 client reads — a **real candidate wire bug**
  (or a per-mode/branch the flat audit can't model), previously MASKED by the truncated
  hand-export as a vague "atlas extra" with no detail.

**The decompose's real value is not auto-clearing — it is SURFACING these divergences
with row-level precision and a verified client read-order.** It turns task-080's vague
"blessed truncation exclusions" into specific, investigable findings:
`<packet>: client reads [..] / atlas writes [..], mismatch at position N`.

## So: how do we resolve the discrepancies?
For each surfaced `divergence`/non-clearing-`upgraded` packet, TRIAGE in IDA (the
exporter already gives the verified client read-order):
1. **Real Atlas bug** (Atlas writes wrong width / extra field the client doesn't read)
   → fix `libs/atlas-packet/...` so the encoder matches the client read-order, + a
   per-version byte test → re-audit → ✅.
2. **Per-mode / branch** (the divergence is one arm of a switch/if the flat audit
   flattened) → handle via the dispatch-selector path (validate/`#`-mode).
3. **Representation-equivalent** (same bytes, different decomposition) → extend the
   analyzer's `widthEquivalent` tolerance.
4. **Version difference** (Atlas intentionally over-writes for another version) →
   documented exclusion.

The clear count (1) is low precisely BECAUSE most flagged packets are real divergences
or modes — which is the audit doing its job. The actionable output is the surfaced,
verified divergence list, not a bulk auto-resolve. **This also re-opens a question about
task-080's completeness claim**: a number of "blessed truncation" exclusions, compared
against the full verified client read-order, are genuine Atlas-vs-client divergences
that warrant triage rather than a blanket bless.
