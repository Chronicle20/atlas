# Task 26 blocker — diagnosis

Task 26 (final verification + coverage promotion) came back **BLOCKED** with no commit.
Two findings; both confirmed independently by the controller, and one is narrower than
the implementer reported.

## Finding 1 — `packet-audit template --check` is not a real subcommand (plan error)

Confirmed against `tools/packet-audit/cmd/root.go:25-88`, the complete dispatch table.
The real subcommands are: export, validate, infer, decompose, triage, registry,
seed-fname, matrix, dispatcher-lint, doc-freshness, gate-lint, gate-check, fname-doc,
operations, evidence, resolve-dispatch, diff-shape, discover-ops, verify-serverbound,
support-summary, status. There is no `template` subcommand — `-template` is a *flag* on
the bare-invocation flag set (`root.go:94`), not a command.

**Ruling: drop `template --check` from Task 26 Step 1.** Template opcode ordering is
already covered by `tools/verify.sh`'s own "template opcode order guard" (it appears in
`tools/task-facts.sh`'s `applicable_guards` for this branch), which Step 6 runs anyway.
The other four Step 1 commands (`matrix`, `operations --check`, `fname-doc --check`,
`gate-check`) all exit 0 clean.

## Finding 2 — only ONE of the three ops is unpromoted, not all of Step 2

The implementer reported Step 2 as broadly failing. Actual cell states, read out of
`docs/packets/audits/status.json` (non-`n-a` cells only):

```
MAPLELIFE_CHECK_NAME  v83 verified  v87 verified  v92 verified  v95 verified
MAPLELIFE_RESULT      v83 verified  v87 verified  v92 verified  v95 verified   (v84 incomplete)
MAPLELIFE_ERROR       v83 verified  v87 verified  v92 verified  v95 verified   (v84 incomplete)
USE_MAPLELIFE                                                   v95 incomplete
```

So **Step 2's first expectation is already met** — `MapleLifeResult` and `MapleLifeError`
are ✅ on all four in-scope versions. The gap is entirely `USE_MAPLELIFE` (serverbound,
`ItemUseMapleLife`):

- **v83, v87, v92** — no registry row at all.
- **v95** — registered but `incomplete`; no audit report backing it.

`derivation.md` §2 gives an address for all four versions, so the codec is derived and
cited. Only the audit/registry layer is missing.

## What remains, and who does it

This is producible in-repo — not an external blocker, so per CLAUDE.md it gets finished
on this branch rather than deferred to a new task. It is a **packet-verification
fan-out**, which is a self-contained detour from the rest of this plan:

1. Register `USE_MAPLELIFE` for gms_v83, gms_v87, gms_v92 (`packet-audit registry`).
2. Dispatch `packet-verifier` per cell — `USE_MAPLELIFE` × {v83, v87, v92, v95} — each
   following `docs/packets/audits/VERIFYING_A_PACKET.md`: derive the client read order,
   write the byte-fixture test with a `packet-audit:verify` marker, pin the evidence
   record, regenerate the matrix. Four cells, batched per IDB.
3. Re-run Task 26 Steps 1-5 (minus the invalid `template --check`) and Step 7's commit.
4. Then the controller's branch-end flagless `tools/verify.sh`.

## One open question for the user

**`MAPLELIFE_RESULT` / `MAPLELIFE_ERROR` are `incomplete` on gms_v84.** Task 26's Step 4
expects v84 to carry writer routing from Task 7 but *no* `mapleLife` config block
(`grep -c mapleLife template_gms_84_1.json` → 0, which the implementer confirmed). If
v84 is deliberately out of scope, `incomplete` there is expected and should be left
alone. If not, it is two more cells to verify. **Not resolved — do not guess.**

## Steps that passed cleanly (no action needed)

- Step 4 (blast radius) and Step 5 (AP/SP additivity) match the brief exactly.
- Step 3's diff is wider than "only additions": a legitimate Task 3 registry rename plus
  an unanticipated but strictly-improving downstream correction to
  `MONSTER_CARNIVAL_SUMMON`. Trace is in `.superpowers/sdd/plan/task-26-report.md`.
