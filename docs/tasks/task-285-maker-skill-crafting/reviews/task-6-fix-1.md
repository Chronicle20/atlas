# Task 6 — fix round 1 re-review

Scope: commit `e29058143`, a single-file change to
`docs/tasks/task-285-maker-skill-crafting/wire-derivation.md`. This is a
scoped re-review of the prior findings in `reviews/task-6.md`, not a fresh
review of Task 6. As in the original review, I have no `ida-pro` MCP access,
so I am judging the record's internal evidentiary sufficiency (are the
claims traced to quoted tool output, are the addresses internally
consistent, does the reasoning follow from what is quoted) — not ground
truth against the live IDBs. That is the correct scope for this pass.

## Diff shape

`git diff --stat e29058143^ e29058143` → one file,
`docs/tasks/task-285-maker-skill-crafting/wire-derivation.md`,
+266/-25. No codec, template, registry, `gates.yaml`, or `libs/atlas-packet`
file touched. Derivation-only, as the commit message claims.

## Blocking finding re-check: `MAKER_RESULT` section (`wire-derivation.md:281-535`)

Previously: `wire-derivation.md:229-293` (old numbering) had `as above` for
all seven non-reference versions with no quoted decompilation or address.
The new text (`wire-derivation.md:330-535`, "Evidence — quoted
decompilation") replaces this with four subsections. Checked each of the
three claims Tasks 7-8 will act on:

**1. Field-for-field IDENTICAL across all eight (underwrites C-2).**
`wire-derivation.md:469-499` gives two address tables — the mode-1/2 arm
(11 columns × 8 versions, `:477-486`) and the mode-3/mode-4 arms (7 columns
× 8 versions, `:490-499`) — with a concrete instruction address for every
`Decode*` call site, per version. This is a real per-version trace, not a
restatement. The verdict table at `:509-520` explicitly says "'same' here is
not shorthand for 'assumed': each cell is the row of addresses in §3" —
correctly self-aware about the failure mode of the original round. PASS.

**2. `bNoItemGain`/`bUsedCatalyst` as real wire bytes, not guard
expressions.** `wire-derivation.md:402-448` quotes disassembly for
`gms_v95` and `gms_v72` for both fields, showing a single `Decode1` call
whose return value feeds the branch test, with `Decode4` calls only inside
the taken branch — the exact distinction the flattened export list could
not make. `wire-derivation.md:454-467` gives one representative quoted `if`
line per remaining version (v79/v83/v84/v87/v92/jms185), each with its own
address, tying the general claim to per-version evidence. PASS.

**3. The unsigned-`nResult` comparison, revised basis.** `wire-derivation.md:339-352`
quotes `gms_v95` disassembly at the function's first `Decode4`:
```
0x910337  call ?Decode4@CInPacket@@QAEKXZ
0x910340  cmp  eax, esi              ; esi == 0
0x910342  jz   short loc_91034D      ; nResult == 0 -> body
0x910344  cmp  eax, 1
0x910347  jnz  loc_910B1E            ; nResult != 1 -> skip body entirely
```
and the same shape for `gms_v72` at `:356-365`. This is genuinely a pair of
equality tests (`cmp`/`jz`, `cmp`/`jnz`), not a magnitude compare — the
quoted asm supports exactly the claimed reading: body executes iff
`nResult` is exactly 0 or 1, and every other value (including negative
values, since these are equality tests and not a signed/unsigned magnitude
compare that could mis-handle sign) takes the bodyless path. That is
functionally identical to unsigned `<= 1` even though it is not literally a
compiled `<=`. The reasoning at `:367-376` (why 4 of 8 IDBs render the local
as `unsigned int` and why `v95`'s `int` typing is a decompiler artefact, not
a machine-code difference) is consistent with the quoted asm. The
per-version address table at `:381-390` gives four addresses per version
(the `Decode4` for `nResult`, the two `cmp`/jump instructions, and the
`nMode` `Decode4`) for all eight — this is the mechanical trace, not an
assumption. PASS — this is a materially stronger and more accurate basis
than the original "Hex-Rays prints `<= 1`" framing, and the revision is
disclosed rather than silently substituted.

**Blocking finding: ADDRESSED.**

## Non-blocking items re-check

1. **PRD anchor.** `wire-derivation.md:34-37` now cites `prd.md:117-136`
   (§4.3 "Craft operations") for the double-encode snippet and explicitly
   disclaims `FR-4.3` at `prd.md:184`. Verified against `prd.md`: lines
   117-136 are indeed the unlabeled illustrative wire-layout block under
   "### 4.3 Craft operations", and `prd.md:184` is indeed `FR-4.3`, which
   reads "`MAKER_RESULT` is a **mode-prefix dispatcher family**..." — an
   unrelated requirement about `MAKER_RESULT`, not the double-encode
   artefact. The correction is accurate. FIXED.

2. **Mode-encode addresses for `gms_v84`/`gms_v87`/`jms_v185`.**
   `wire-derivation.md:210-212` (MAKER_SKILL per-version confirmation table)
   now carries in-arm addresses for all three (`0x8525da`/`0x8525c0`/
   `0x85257f` for v84; `0x88b0f4`/`0x88b0da`/`0x88b099` for v87;
   `0x8b1163`/`0x8b1149`/`0x8b1108` for jms185), matching the format already
   used for v72/v79/v83. FIXED.

3. **`[^unsym]` footnote.** `wire-derivation.md:16-22` adds a `†` marker on
   the summary table for `gms_v84`/`gms_v92` on both ops, with inline text
   pointing at "The two unsymbolized versions" section. `wire-derivation.md:63-88`
   is that section, unchanged in substance from the original review (the
   identification chain for both functions was already present pre-fix and
   was not flagged before). The footnote is additionally attached to every
   downstream table cell that depends on the structural (not name-based)
   identification: MAKER_SKILL table row (`:212`), MAKER_RESULT guard
   address table (`:386`), the two mode-arm address tables (`:482`, `:495`),
   and the verdict table (`:516`, `:518`). FIXED — the footnote is
   consistently propagated, not attached once and forgotten.

## New claim scrutinized: `char`-narrowed `Decode4` locals (`wire-derivation.md:501-507`)

Text: several IDBs render `char v4; v4 = CInPacket::Decode4(v2);` (naming
`gms_v72 0x86a1ce`, `v79 0x8b5b71`, `v83 0x95db4f`, `v84 0x99be38`,
`v87 0x9e022e`, `jms185 0xa295a4`), characterized as a Hex-Rays
register-width inference on an unused high byte, not a wire-width signal —
callee is `?Decode4@CInPacket@@QAEKXZ` in every case, so 4 bytes are
consumed and must not be narrowed to a byte in the codec.

Findings:
- Unlike claims 1-3 above, this paragraph is **prose, not a quoted code
  block** — there is no fenced `c` or `asm` excerpt reproducing the literal
  `char v4 = CInPacket::Decode4(v2);` line the paragraph describes, in
  contrast to the doc's own established bar elsewhere (§1 and §2 both quote
  literal decompiler/disassembly text for their claims).
- However, the six addresses cited (`0x86a1ce`, `0x8b5b71`, `0x95db4f`,
  `0x99be38`, `0x9e022e`, `0xa295a4`) are not free-floating: they are
  exactly the `m3 nSourceItemID` column values already published in the §3
  mode-3/mode-4 address table (`wire-derivation.md:492-499`, cross-checked:
  `gms_v72` row column 2 = `0x86a1ce`, `gms_v79` = `0x8b5b71`, `gms_v83` =
  `0x95db4f`, `gms_v84` = `0x99be38`, `gms_v87` = `0x9e022e`, `jms_v185` =
  `0xa295a4` — all match). That table's header states each column is "the
  address of the `CInPacket::Decode*` call that reads that field," so the
  claim that these are `Decode4` calls is not newly asserted here — it
  inherits from the already-published per-field address table, which in
  turn is scoped by the doc's stated methodology ("each of the eight
  functions was decompiled in full," `:332-333`) rather than instruction-
  level `insn_query` (which the doc explicitly reserves for the three
  higher-risk sites at `:333-335`: the `nResult` guard, `bNoItemGain`,
  `bUsedCatalyst`).
- The width argument itself is sound *given* the callee identity: the
  mangled name `?Decode4@CInPacket@@QAEKXZ` return type is `K` (`unsigned
  long`, 4 bytes) regardless of what local variable type Hex-Rays chooses
  to assign the return value to — a `char`-typed local capturing a 4-byte
  return is exactly the kind of decompiler rendering artefact the paragraph
  describes, and this reasoning pattern is consistent with how the rest of
  the document treats Hex-Rays type annotations as non-authoritative
  relative to the callee identity (see the identical argument made for
  `nResult`'s `int` vs. `unsigned int` typing at `:371-376`).

Disposition: **non-blocking, but flag for tightening.** The claim traces
correctly to already-published, table-scoped addresses and a sound
"callee name is dispositive over local type" argument that the document
uses elsewhere — it is not a bare assertion invented for this one section.
It falls short of the doc's own strongest evidence bar only in that it does
not additionally reproduce the literal Hex-Rays line for at least one
version (a single quoted line, as done for `nResult` and the guard bytes,
would remove all doubt). Given that Task 7 needs "do not narrow to byte"
to be right, and the underlying addresses are independently anchored in an
existing address table rather than freestanding, I am not treating this as
a blocking gap in this round — but a reviewer of Task 7 should re-confirm
the actual field width lands as 4 bytes in the codec/tests, since this is
the one claim in the fix that stops one level short of a literal quote.

## Weakening / dropped-claim check

Diffed the removed lines (`git diff e29058143^ e29058143` minus-side): every
removed line is either (a) an `as above` placeholder row being replaced by
concrete addresses, (b) the old two-column-narrower summary table rows
being replaced by footnoted equivalents, or (c) the FR-4.3 mis-anchor being
corrected. No previously-evidenced claim (address, quoted asm/C, or
verdict) was deleted or softened. The "IDENTICAL" / "REFERENCE" verdicts
for both ops are unchanged from before the fix; only the depth of
supporting evidence changed.

## Not evaluable

Ground truth against the live IDBs (addresses, mangled names, and quoted
asm/C actually appearing in the sessions cited) — no `ida-pro` MCP access
in this environment, consistent with the original review's stated
limitation. This is the review's declared scope, not an absorbed gap.

## Verdict

ADDRESSED. The MAKER_RESULT blocking finding is resolved: all three claims
Task 7-8 will implement now trace to quoted decompilation/disassembly with
per-version instruction addresses, and the revised unsigned-comparison
basis (equality-pair, not magnitude compare) is correctly supported by the
quoted asm. All three non-blocking items are confirmed fixed. One new claim
(the `Decode4`-into-`char`-local narrowing note) is evidenced by inherited,
table-scoped addresses and sound reasoning but stops short of a literal
quoted decompiler line — flagged as non-blocking, worth a follow-up check
when Task 7 lands.
