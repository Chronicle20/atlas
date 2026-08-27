# Review — Task 1: derive serverbound wire (§1–§3)

Range reviewed: `6fc822ac4..98efc7f0d` (single commit `98efc7f0d`).
Diff: `docs/tasks/task-246-maple-life-character-creation/derivation.md`, +503/-0,
new file. No other files touched — matches the brief's file-write scope
exactly (`docs/tasks/task-246-maple-life-character-creation/derivation.md`
new; all others read-only).

## Scope confirmation

The unit is a pure documentation derivation with no code change. Reviewed:
the full `derivation.md` §1–§3, cross-checked every claim that can be
verified from repo state (registry YAMLs, referenced Go source, WZ data,
the evidence-hash tool, the ida-exports JSON files) against the document's
assertions. **I have no IDA MCP session access in this review context**, so
the raw decompile/disassembly content quoted in the document (the actual
instruction bytes, offsets, and control flow inside the IDBs) could not be
independently re-derived or re-verified against the live binaries — that
portion is recorded under "Not evaluable" below, not silently approved.

## §1 — `get_cashslot_item_type` 543 arm (v83 + v95)

- **Addresses given**: v83 `0x48645b` (size `0x2fc`), v95 `0x488c70` (size
  `0x3c9`), both with disassembly windows (`0x486720`–`0x486746` /
  `0x488ff0`–`0x489012`) and named registers/opcodes (`jl`/`jg` signed vs.
  `ja` unsigned). Meets the "trace to a per-version IDA address" bar at the
  instruction level, not just the function level. ✅
- Signedness explicitly stated for both versions (v83 signed two-sided,
  v95 unsigned one-sided idiom). ✅
- 543 → type value mapping given (65/57 on v83, 66/58 on v95). ✅
- OQ-2 answered in prose (`derivation.md:124-153`): 57/58 **is** reachable
  with shipped data via `05430000`, because that id's `itemId/1000` (5430)
  falls outside `[5431,5432]` on both signed and unsigned forms. This
  correctly diverges from the brief's simplified either/or framing and the
  divergence is called out honestly (§1.3, closing paragraph) rather than
  forced into the brief's two buckets. ✅
- Cross-checked the derived classification logic against Atlas's live
  handler at `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1456-1465`
  (`ClassificationCharacterCreation` arm: `if itemId/1000-5431 > 1 { 57/58 } else { 65/66 }`,
  gated by `t.Region() == "GMS" && t.MajorVersion() >= 95`) — confirms the
  document's claim that Atlas's existing Go re-implementation already
  matches the derived logic. The document correctly flags this as
  conditional on `item.Id`'s underlying type being unsigned rather than
  asserting it flatly. ✅
- §1.4 WZ spec check: read from "two independent local WZ corpora," XML
  quoted for all three ids, explicit finding of no `spec` node differences
  (icon dimensions only). OQ-3 answered. ✅

**§1 verdict: meets the brief.** No invented values found; every number
traces to a cited address/instruction.

## §2 — `CUICharacterSaleDlg::SendCreateNewCharacter` (all 5 versions)

- **Opcode literals verified against the registry** (I read
  `docs/packets/registry/gms_{v83,v87,v92,v95}.yaml` directly): v83=79,
  v87=82, v92=86, v95=85 — all four match `derivation.md`'s claimed
  `USE_CASH_ITEM` opcodes exactly, byte for byte against the registry file.
  ✅
- Complete field-by-field encode order given for v83/v87/v92/v95, each with
  a function address and (for v83/v87) a size. `update_time` lead/trail
  called out explicitly and cross-checked against `UpdateTimeFirst`
  (`libs/atlas-packet/cash/serverbound/item_use.go:22-24`, read directly —
  doc comment matches: "GMS v87+ ... lead" / "two oldest versions ...
  trailing"). ✅
- `MajorAtLeast` boundary for the *new* doubled-update_time finding is
  correctly expressed as `t.IsRegion("GMS") && t.MajorAtLeast(87)`
  (`derivation.md:397`) — grepped the whole file for `MajorVersion(` and
  found zero raw literal-comparison occurrences. The constraint against a
  bare `MajorVersion() >= N` is honored. ✅
- §2.6 explicit derived statement that the layout does **not** match
  `charsb.CreateCharacter` — verified directly against
  `libs/atlas-packet/character/serverbound/create.go`: its `CreateCharacter`
  struct (`name, jobIndex, subJobIndex, face, hair, hairColor, skinColor,
  topTemplateId, bottomTemplateId, shoesTemplateId, weaponTemplateId,
  gender, strength, dexterity, intelligence, luck`) genuinely has no
  `nPOS`/`nItemID`/`update_time`/`nCurrentClass`/`nSP` fields, matching the
  document's claim field for field. ✅

### v84 VERSION-ABSENT finding — evidenced, but one contradicting repo fact not addressed

`derivation.md:203-243` gives three independently-corroborating searches
with addresses (`func_query` zero-hit, `find_regex` for the RTTI string
zero-hit, and `insn_query` for literals 543/5431/5432 inside
`0xa54a2f`/size `0x4499`, zero-hit). This meets the "addresses/searches
shown" evidentiary bar the review brief asks me to hold this claim to.

However, I independently checked
`docs/packets/registry/gms_v84.yaml:2972-2986` (`USE_CASH_ITEM`, opcode
79) and its `fname_alts` list **does include**
`CUICharacterSaleDlg::SendCreateNewCharacter`. On inspection this is
explained by the entry's own `note`: *"seeded from the v83 CSV column —
the CSVs have no v84 column ... Corrected by discover-ops against the v84
IDB"* — i.e. the opcode number was IDA-corrected but the `fname_alts` text
is inherited, unverified CSV boilerplate identical to v83's list (I
diffed v83's and v84's `fname_alts` lists — they are byte-identical). So
this does not actually contradict the derivation's finding once traced
through, but **the derivation never mentions or reconciles this
registry entry**, even though it directly names the exact class/method the
task concludes is version-absent. A reader auditing only `derivation.md`
would have no way to know this apparent contradiction exists and had
already been checked. This is a completeness gap, not a correctness
defect — recorded as non-blocking below.

### decompile_sha256 — all four PENDING

I independently grepped `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v92,gms_v95}.json`
for `SendCreateNewCharacter` and confirmed zero matches in all four files,
and confirmed `tools/packet-audit/internal/evidence/hash.go` exists at the
path the report cites. The PENDING claim is factually accurate, not a
fabricated excuse. The brief's global constraint says the hash "must be
recorded verbatim." It is not. But the brief's own Step scoping restricts
this task's only repo write to `derivation.md`, and computing the hash
requires re-harvesting and committing `docs/packets/ida-exports/*.json` —
a change outside that scope. The implementer flagged this prominently in
§2.1 as a blocking dependency for Tasks 4–6 rather than fabricating a
value or silently omitting the row. This is the correct response to a
genuine brief/constraint conflict (per CLAUDE.md's "surface it and ask"
carve-out) — I judge this an **acceptable recorded gap for Task 1
specifically**, but it is a real, unresolved requirement failure against
the brief's own text that must be closed (by re-harvesting the exports)
before Task 4–6 can proceed. This should not be silently inherited as
"already handled."

### Doubled update_time — recorded, but with lower evidentiary rigor than §1

`derivation.md:290-384` records `update_time` written twice (leading +
trailing) on v87/v92/v95, a finding beyond the brief's explicit ask. It is
recorded with function addresses and opcode evidence, and is structurally
consistent across three independent versions plus the v92 `sub_936E80`
naming reconciliation. However, unlike §1 (which quotes raw disassembly
with per-instruction addresses for its signedness claim), §2's per-field
encode order — including the specific second `update_time` call — is given
as a paraphrased decompiled-C listing tied only to the function's entry
address, not to an instruction address for that specific encode call. For
a finding that is explicitly flagged as changing Task 3's codec shape
(the existing single-field `ItemUse.updateTime` cannot model it as-is),
this is a materially weaker evidentiary trail than the brief's own
address-tracing standard implies elsewhere in the document. Recommend
Task 3 independently re-confirm the second `update_time` call site with a
raw disassembly excerpt before committing to a two-field codec change.

## §3 — `USE_MAPLELIFE` / OQ-1

- Positive finding that opcode 303 is never constructed: exhaustive
  enumeration of `CUICharacterSaleDlg::` methods (50 results, only two
  construct `COutPacket` with a literal opcode — `SendCreateNewCharacter`
  @85 and `SendCheckDuplicateIDPacket` @311, neither 303), plus a decisive
  cross-check that `CWvsContext::SendConsumeCashItemUseRequest` constructs
  exactly one `COutPacket` in its whole body, at `0x9eb4aa`, opcode `0x55`
  (85) — same literal as `SendCreateNewCharacter`'s call site.
- OQ-1 answered as exactly one of the three brief-specified alternatives
  ("v95 sends only the `USE_CASH_ITEM` 543 sub-body"), with address
  evidence (`0x9eb4aa`/`0x9eb4a4`, cross-confirmed against v83/v87/v92's own
  opcodes via the §2.5 table). ✅
- Registry-mislabel finding recorded as a flag, not auto-fixed (correctly
  deferred per `IMPLEMENTING_A_PACKET.md`'s escalate-don't-auto-fix rule,
  and per the constraint that this task only writes `derivation.md`).

**§3 verdict: meets the brief**, including a load-bearing recommendation
for Task 6's package layout, correctly flagged as a recommendation rather
than asserted as settled fact.

## Spec-compliance summary (brief items 1–4)

1. §1 both versions + OQ-2: ✅
2. §2 all five versions, `decompile_sha256`, encode order, lead/trail,
   `MajorAtLeast` boundaries, layout-vs-`charsb.CreateCharacter`: ✅ except
   `decompile_sha256` (❌, but with an acceptable, well-evidenced escalation
   — see above)
3. §3 sender/no-sender finding + OQ-1 with address evidence: ✅
4. §1.4 WZ spec differences / OQ-3: ✅

## Findings

### Non-blocking

1. `derivation.md` §2 never cross-references
   `docs/packets/registry/gms_v84.yaml`'s `USE_CASH_ITEM` `fname_alts`
   entry, which names `CUICharacterSaleDlg::SendCreateNewCharacter` for
   v84. I traced this myself and found it to be inherited, unverified
   CSV-import boilerplate (identical to v83's list, per the entry's own
   `note`), so it does not actually contradict the VERSION-ABSENT finding —
   but the derivation should have surfaced and dismissed this apparent
   conflict explicitly rather than leaving a future auditor to discover it.
2. `decompile_sha256` for all four `SendCreateNewCharacter` addresses is
   recorded as PENDING rather than a value, correctly root-caused (function
   absent from the checked-in `docs/packets/ida-exports/*.json`, confirmed
   independently by grep) and escalated as a blocking dependency for
   Tasks 4–6. This is the right call given the brief's own file-write
   scope restriction, but it is a genuine unmet requirement (the brief's
   global constraint calls the hash "needed verbatim") — the plan/task
   sequence must close this (re-harvest the exports, task-081 playbook)
   before any Task 4–6 evidence-pin step is attempted; do not treat it as
   already resolved.
3. The doubled-`update_time` finding (§2.2–§2.4) — a load-bearing
   discovery beyond the brief's ask that changes Task 3's codec shape — is
   recorded with function-level addresses but not an instruction-level
   address for the specific second encode call, a lower rigor bar than §1
   applies to its own contestable claim. Recommend Task 3 re-confirm with
   a raw disassembly excerpt before committing to the two-field codec
   change.

### Not evaluable

- The raw decompile/disassembly content quoted throughout the document
  (instruction bytes, control flow, struct offsets inside the five IDBs)
  could not be independently re-verified against the live IDA sessions —
  this review has no MCP IDA tool access. I verified everything that is
  cross-checkable from repo state (registry opcodes, referenced Go struct
  shapes, WZ XML, the evidence-hash tool path, the ida-exports JSON
  absence) and found no contradiction, but the IDA-internal claims
  themselves are taken on the document's word for the portions I cannot
  independently query.

## Verdict rationale

No fabricated or unsupported values were found — every claim I could
verify against repo state matched exactly (opcodes, struct shapes, WZ
data, tool paths, export absence). The one apparent contradiction (v84
registry `fname_alts`) resolves in the derivation's favor on inspection,
but should have been addressed in the document itself. The
`decompile_sha256` gap is real but correctly escalated rather than
fabricated or silently dropped, consistent with the "unverified is
unknown/unverified" rule. These are findings for the plan/next-task
sequence to act on, not defects in this unit's own execution against its
scoped brief.
