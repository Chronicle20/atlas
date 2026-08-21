# Review — Task 1: Derive `USE_INNER_PORTAL` structures for all six versions

**Commit reviewed:** `b0cfe59e4` (range `6f0799565..b0cfe59e4`)
**Scope:** six new files under `docs/tasks/task-250-inner-portal-registration/structures/` (docs-only, 341 insertions, 0 deletions — confirmed via `git show --stat b0cfe59e4`).

This is an evidence-quality review. I do not have IDA tool access in this
session, so I cannot independently re-run `func_query`/`decompile` against the
IDBs to verify a quoted line is the actual decompiler output rather than a
plausible fabrication. What I *can* and did check: internal consistency of
each doc (hex/decimal opcode pairs, field-order agreement across all six
files), the registry opcode cross-check against the actual registry files,
consistency against `design.md`'s existing §1 claims, explicit presence of
the required sections (gate decision, per-version delta, derivation method),
and honesty of the two self-reported findings.

## 1. Field rows quote decompiled expressions

All six docs' "Ordered field table" sections quote concrete client
expressions with variable names, stack offsets, and hex addresses per row
(e.g. `gms_v83.md:40-45`, `gms_v84.md:44-50`, `gms_v87.md:19-24`,
`gms_v92.md:56-61`, `gms_v95.md:19-24`, `jms_v185.md:30-35`), and every row
matches the brief's expected layout: `Encode1(fieldKey)`, `EncodeStr
(portalName)`, `Encode2×4` (`x`, `y`, `targetX`, `targetY`). Field order and
widths are identical across all six files — no version silently varies the
shape.

**PASS with a scope caveat.** I cannot independently confirm these are the
actual `decompile` outputs verbatim (no IDA access this session — see "Not
evaluable" below). What I can confirm: the quotes are internally consistent
(each version cites *different*, version-appropriate stack offsets — e.g.
`field + 308` for v83 vs `field + 328` for v87/jms_v185 vs `v15 + 328` for
v84 vs `v19 + 332` for v92 — which is the pattern expected of genuine
per-binary decompiler variance rather than a copy-pasted template), the
register/variable naming conventions differ appropriately per session (`v58`
for v83, `&v65` for v84, `&a3` for v87, `&rc` for v95, `v67` for jms_v185,
`&v74` for v92 — plausible per-function IDA variable numbering, not reused
across files), and the gms_v84.md author explicitly flags and explains a
naming quirk (`gms_v84.md:45`, offset `+328` vs v83's `+308` attributed to
IDA struct-member text variance) rather than silently smoothing it over —
that kind of self-correcting detail is evidence of genuine derivation rather
than template-filling.

## 2. Opcode cross-check per version, verified against the registry directly

Verified by reading `docs/packets/registry/<version>.yaml` myself (not
trusting the doc's citation):

| version | doc opcode | registry opcode (`grep` verified) | match |
|---|---|---|---|
| gms_v83 | 101 (`gms_v83.md:32-34`) | `docs/packets/registry/gms_v83.yaml:2460` `opcode: 101` | yes |
| gms_v84 | 101 (`gms_v84.md:32-39`) | `docs/packets/registry/gms_v84.yaml:3148` `opcode: 101` | yes |
| gms_v87 | 104 (`gms_v87.md:10-13`) | `docs/packets/registry/gms_v87.yaml:2576` `opcode: 104` | yes |
| gms_v92 | 112 (`gms_v92.md:47-50`) | `docs/packets/registry/gms_v92.yaml:2796` `opcode: 112` | yes |
| gms_v95 | 113 (`gms_v95.md:10-13`) | `docs/packets/registry/gms_v95.yaml:2831` `opcode: 113` | yes |
| jms_v185 | 96 (`jms_v185.md:10-11`) | `docs/packets/registry/jms_v185.yaml:2553` `opcode: 96` | yes |

All six match both the registry and the brief's own expected table exactly.
Hex/decimal pairs quoted in each doc are also internally self-consistent
(`0x65`=101, `0x68`=104, `0x70`=112, `0x60`=96 — checked arithmetically).
**PASS.**

## 3. No-`MajorAtLeast`-gate ruling stated per doc

Every one of the six docs carries an explicit `## Gate decision` section
stating "no `MajorAtLeast` gate is required" with a rationale tied to the
byte-identical field layout across all six versions, and a separate
`### Per-version delta` line stating either "no delta" or the specific
divergence (all six say "no delta," only the opcode differs). Confirmed at:
`gms_v83.md:51-62`, `gms_v84.md:56-65`, `gms_v87.md:29-38`,
`gms_v92.md:67-76`, `gms_v95.md:38-51`, `jms_v185.md:41-49`. **PASS** — this
satisfies Step 5 and gives Task 3 an explicit, non-implied ruling to copy.

## 4. Finding B — gms_v92 evidentiary chain

The report (`task-1-report.md:63-77`) and the doc itself
(`gms_v92.md:5-41`) both honestly disclose that gms_v92 has no
`CheckPortal_Collision`/`FindPortalByName`/`FindPortal_Collision` symbol, so
the send site was located by: (a) `xrefs_to(SendSkillUseRequest)` → 8
callers, (b) size-filtering against the confirmed versions' function sizes
to reduce to 2 candidates, (c) decompiling both and eliminating one on
argument-count/shape grounds, (d) confirming the survivor by 6-argument
signature match, an internal `FindPortalByName`-equivalent call, the exact
`COutPacket` ctor + `Encode1, EncodeStr, Encode2×4` sequence, and (e) an
exact opcode match (`112`) against the registry.

**Judgment: this is *inferred*, not *derived*, and the doc/report are right
to say so explicitly rather than presenting it at the same evidentiary tier
as v83/v84.** The v83/v84 cells have a caller-side confirmation that the
5-argument call's 3rd/4th arguments are literally sourced from the caller's
own `pn`/`tn` struct fields (visible at the call site, independent of the
callee's internal variable naming) — i.e. two independent readings agree.
The v92 cell has only one reading: the callee's own internal structure
(argument shape + Encode sequence + opcode). Structural self-consistency and
an exact opcode match are strong corroboration and rule out "wrong
function" for all practical purposes, but they do not independently confirm
argument *role* assignment (e.g. that IDA's decompiled `a5` truly carries
`sTargetPortalName` and not some renamed/reordered parameter) the way a
caller-side read does. The report's own wording ("weaker corroboration,"
"different evidentiary path") matches this judgment — it is not overclaimed.

**Additional query that would close the gap** (naming it, since I have no
IDA access this session, per the task's own constraint): `func_query` a
broader wildcard on gms_v92 for `*Portal*` or `*Collision*` (not just the
three exact names already tried) to check whether the collision handler
exists unnamed at some address reachable from `CUserLocal::HandleUpKeyDown`
or a portal-enter dispatch table entry; failing that, `xrefs_to` on the
confirmed `0x8f85c0` callee itself to find *its* caller and inspect whether
that caller's decompilation shows a `portal->pn` / `portal->tn`-shaped
struct-member read feeding the 3rd/4th arguments, mirroring the v83/v84
caller-walk exactly. Either query would upgrade this cell from inferred to
derived. Not blocking for this task — the finding is honestly disclosed per
the brief's "record, don't reconcile" instruction — but Task 3 or a
follow-up reviewer should not treat gms_v92's field order as having the same
confidence as the other five without running one of these.

## 5. Finding A — jms_v185 ctor address discrepancy

Confirmed: `jms_v185.md:13-22` records this explicitly as a `**Finding:**`
callout, not a silent reconciliation — good adherence to the brief's Step 2
instruction ("A disagreement ... is a finding: record it ... and stop; do
not reconcile it silently").

I checked `design.md` directly (`docs/tasks/task-250-inner-portal-registration/design.md:88`):

```
| jms_v185 | `0xa2218f` (named) | `push 60h` at `0xa2230e` | `Encode1, EncodeStr, Encode2 ×4` |
```

Note design.md's own column header is "Opcode at the ctor" and its cell text
is `push 60h` — i.e. design.md is citing the *push* instruction that sets up
the opcode argument, not the `COutPacket::COutPacket` *call* instruction
itself. The new doc's ctor-call address (`0xa22313`) is 5 bytes later, which
is exactly the expected gap between a `push imm8` (2 bytes) plus a couple of
setup instructions and the subsequent `call`. The new doc's own explanation
(`jms_v185.md:19-22`) — "most likely because the design doc cited the push
instruction address and this derivation cites the call (constructor)
address" — is plausible and not actually a contradiction of design.md's
literal claim, only of a looser paraphrase ("the ctor address") of what
design.md said.

**Disposition:** the doc correctly records this as a Finding rather than
silently reconciling — satisfies the brief. Whether this needs a **correction
commit to design.md**: not blocking. design.md's literal text (`push 60h`
at `0xa2230e`) is not demonstrated to be wrong — it may be describing a
different (earlier) instruction in the same short sequence, which is
consistent with the two addresses differing by only 5 bytes. I did not find
evidence design.md's address is incorrect, only that the two docs describe
two different (nearby) instructions using loose, potentially conflatable
language ("ctor" / "push ... at"). A future doc-hygiene pass could tighten
design.md's column header to distinguish "push" vs "call" addresses
explicitly to prevent this ambiguity from recurring for other versions, but
that is a non-blocking polish item, not a defect in this task's output.

## 6. No literal home/absolute paths in committed docs

```
grep -n "/home/\|E:\\\\" docs/tasks/task-250-inner-portal-registration/structures/*.md
```
returned no matches. **PASS.** (Note: the task report at
`.superpowers/sdd/plan/reports/task-1-report.md:41` does contain a Windows
path fragment `E:\...IDBs_v9`, but `.superpowers/` is listed in
`.git/info/exclude` and is not part of this commit or of `docs/`, so it is
outside the enforced scope and outside this review's diff surface.)

## Not evaluable

- **Verbatim correctness of every quoted decompile line.** I have no IDA
  tool access this session. I verified internal consistency (per-version
  variable/offset variance, arithmetic consistency of hex/decimal opcode
  pairs, agreement with the brief's expected layout and with the registry)
  but cannot confirm any individual quoted C expression is byte-for-byte
  what IDA actually emitted. This is the one check the brief's own framing
  ("evidence-quality review") most wants, and it is the one I structurally
  cannot close without tool access — flagging rather than silently treating
  "internally consistent" as "verified."
- **Whether the two size-plausible gms_v92 candidates (`sub_91EC80`,
  `sub_8F85C0`) were the *only* two remaining after the size filter, or
  whether the filter's threshold was chosen to make the answer come out
  right.** The report states the filter ruled out functions "too small for a
  function with this much branching" using a qualitative, not a hard-coded,
  threshold; I cannot re-run the filter to check for confirmation bias in
  where the line was drawn.

## Verdict rationale

No blocking defects found. Field order, widths, opcodes, and the no-gate
ruling are all present, internally consistent, and cross-checked correctly
against the registry. Both self-reported findings (A: ctor address, B: v92
evidentiary weakness) are recorded as findings rather than silently
reconciled, per the brief's explicit instruction — this is the behavior the
brief was testing for, and the doc set delivers it. The one thing worth a
non-blocking flag for Task 3/12 consumers: treat gms_v92's field-order cell
as inferred-with-strong-corroboration, not derived-to-the-same-standard as
the other five, until one of the two named follow-up queries is run.
