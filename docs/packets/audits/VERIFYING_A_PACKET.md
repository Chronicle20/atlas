# Verifying a packet (single packet × single version)

The canonical procedure for promoting a coverage-matrix cell
(`docs/packets/audits/STATUS.md`) to ✅. Written for a human or an agent.
Hard rule (CLAUDE.md "Verification Over Memory"): every byte in a fixture must
trace to a decompile line — never to MapleStory knowledge from memory.

Cell states: `✅` verified · `🧩` family (mode-prefix dispatcher; sub-arms
unverified) · `🟡` partial · `❌` incomplete · `⬜` n-a · `🟥` conflict.

## 0. Prerequisites
- The nine registry files in `docs/packets/registry/` (one per version key:
  gms_v48/v61/v72/v79/v83/v84/v87/v95, jms_v185).
- The version's IDA export in `docs/packets/ida-exports/` (jms_v185 uses
  `gms_jms_185.json`).
- For fresh decompiles: a live ida-pro-mcp instance with the version's IDB.

## 1. Resolve scope
Look the op up in `docs/packets/registry/<version>.yaml`. If absent there:
your job is confirming `n-a` (verify the template doesn't route the opcode in
`services/atlas-configurations/seed-data/templates/` and no Atlas struct
claims it) or filing a 🟥 conflict — then stop.

## 2. Check current state
- The cell in STATUS.md and `status.json`.
- Any evidence record: `docs/packets/evidence/<version>/<packet dots>.yaml`.
- The latest audit report: `docs/packets/audits/<version>/<Writer>.{json,md}`.

Dispatcher-family trap: an op row whose FName fans out to several per-case
writers (PARTY_OPERATION, MESSENGER, STORAGE, …) grades **worst-of all
siblings**. Promoting one mode improves that packet's cell in `status.json`
but the STATUS.md op row stays at the worst sibling until the whole family is
done. If you need a visible STATUS.md flip, pick a single-writer op; for
campaign work on a family, plan to verify every sibling.

## 3. Decompile the client side
- Enumerate live instances (`mcp__ida-pro__list_instances`) and
  `select_instance` the one whose loaded IDB matches the target version —
  ports vary by IDA launch order, NEVER hardcode them.
- Decompile the registry entry's `fname` (batch `decompile`); descend into
  helper reads (address-based descent, same rule as the exporter).
- Write down the full ordered read/write list including guards and loop bounds.

## 4. Compare against Atlas
The encoder/decoder in `libs/atlas-packet/<pkg>/`, including version gates
(`MajorVersion()` comparisons — beware the v84 off-by-one class: `>83` must be
`>=87` when v84 matches v83). Divergence ⇒ wire fix FIRST (own commit, own
review), then continue.

## 5. Derive expected bytes
Concrete model fixture; hand-compute the byte sequence from the client read
order. One fixture per mode for mode-driven packets. Cite the decompile line
for every field in a comment.

Opaque-family caveat: for OPAQUE_LEDGER families the export's `calls` stop at
the opaque buffer boundary (`DecodeBuf`). Bytes inside the blob cannot cite a
per-field decompile line; derive them from the Atlas encoder source plus the
audit report's "absorbed by trailing opaque buffer" rows, and say so in the
test comment (the OPAQUE_LEDGER VERIFIED-EXCEPTION discipline).

## 6. Write the byte-test
With the marker above the function:

    // packet-audit:verify packet=<pkg/dir/Struct> version=<key> ida=<0xaddr>

Use the existing `pt.Variants` table pattern
(`libs/atlas-packet/party/clientbound/invite_test.go` is the reference).
The table is a `[]struct{ variant pt.TenantVariant; ... }` slice that
accesses `pt.Variants` by index; see `TestInviteByteOutput` for a complete
example with per-variant byte counts and IDA evidence comments.

## 7. Pin evidence (tier-1 only)
Tier-0 cells promote on tool-✅ + marker alone — do NOT pin a record for
them: an evidence record is a standing freshness liability (export drift
would degrade the cell to ❌), so only carry one where the grading rules
need it (tier-1, or a deferral that evidence justifies).

    go run ./tools/packet-audit evidence pin --packet <id> --version <key> \
      --ida "<FName>" --category TIER1-FIXTURE

This writes `docs/packets/evidence/<version>/<packet dots>.yaml`. After the
command succeeds, open the YAML and add the `verifies:` field manually:

    verifies:
      - <test file>#<TestName>

The `--ida` argument is the function name exactly as it appears as a key in
the IDA export's `functions` map (e.g. `CWvsContext::OnPartyResult` — fully
qualified, including any `#case` suffix the export uses), not a hex address.
The tool resolves the address from the export and embeds it in the record
automatically.

## 8. Regenerate and verify promotion
    go run ./tools/packet-audit matrix
    go run ./tools/packet-audit matrix --check
The cell must now be ✅. Commit test + evidence + STATUS.md/status.json
together.

`matrix --check` is a **hard, blocking CI gate** (`.github/workflows/packet-matrix.yml`):
the registry-seed conflict backlog was burned to zero (task-085), so a clean tree
exits 0 — there is no grandfathering or `continue-on-error`. Your bar is a clean
**exit 0**: any 🟥 conflict, any orphan/dangling/stale/drift finding, or a stale
committed STATUS.md/status.json fails the gate. Regenerate and commit
STATUS.md/status.json in the same PR.

## 9. Serverbound & shared-codec packets (task-092)

A serverbound cell needs THREE artifacts that all agree AND the op must be
**routed** in that version's seed template:

1. The `// packet-audit:verify` marker (§6).
2. The pinned evidence record (§7).
3. An **audit report** `docs/packets/audits/<version>/<Writer>.json`. Evidence
   with no report → `matrix --check` "dangling evidence" failure. Reports are
   generated by the ROOT command against the export (deterministic, no live IDA):

       go run ./tools/packet-audit \
         -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
         -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
         -template services/atlas-configurations/seed-data/templates/template_<v>.json \
         -ida-source docs/packets/ida-exports/<export>.json \
         -output /tmp/rpt   # writes /tmp/rpt/<version>/<Writer>.{json,md}

   Run to a temp `-output`, copy the specific report(s) you need into
   `docs/packets/audits/<version>/`. (`triage`/`decompose` only *upgrade* existing
   reports — they do not generate new ones.)

**Linkage.** A registry op links to a codec by its **primary `fname`** (not
`fname_alts`) via the `candidatesFromFName` switch in `cmd/run.go` →
`{name, pkg, dir: serverbound}`. `locateAtlasFile` matches `type <name> struct` in
`<pkg>/serverbound/`; `qualifiedWriterName(pkg, name)` = TitleCase(pkg)+name → both
the marker `packet=<pkg>/serverbound/<Qualified>` path and the report filename. So
the struct name carries NO pkg prefix (struct `MobBanishPlayer` + pkg `character` →
`CharacterMobBanishPlayer`). To verify a NEW serverbound op you must add its primary
fname as a `candidatesFromFName` case. Example: `CLOSE_RANGE_ATTACK` keys to
`CUserLocal::TryDoingNormalAttack`, not the `TryDoingMeleeAttack` alt.

**Shared-model ops.** When several ops share one decoder (the four attack ops →
`model.AttackInfo`), create a thin per-op wrapper struct in `<pkg>/serverbound/` that
embeds the shared model and delegates Encode/Decode (the analyzer recurses into the
embedded field). One wrapper per op = one packet/evidence per op. This mirrors the
clientbound `character/clientbound/Attack`. The wrapper may be an uncalled audit codec;
the production handler keeps decoding the shared model directly.

> **Before treating a serverbound ❌ as "missing", check it isn't already implemented.**
> An ❌ often means an *unverified shared codec*, not a missing one — grep the channel
> handlers + `libs/atlas-packet/model/` for a decoder of that opcode. If one exists,
> the work is verification (wrap + link), NOT a new (duplicate, possibly-wrong) codec.

## 10. Export hygiene, report-gen, and IDB naming (task-092)

- **The export is NOT idempotent — never overwrite a committed export.** Re-running
  `packet-audit export` drifts ~150 existing function keys (Hex-Rays variance) and
  degrades unrelated cells. To add/fix ONE function: harvest with
  `-prior-export "" -pending <roster.md> -descent-depth 12` to a temp file, then
  surgically splice ONLY the needed entries into the committed export (absent-only for
  helpers; overwrite the one sender). A committed entry can be a stub (`calls: null`) —
  overwrite it with the harvested real-calls version.
- **Per-instance export.** The tool connects to ONE IDA instance:
  `-ida-url http://<host>:<port>/mcp -ida-port <port>` must point at the TARGET version.
- **`COutPacket`-delegate harvest artifact.** On some send functions the harvester
  records the `COutPacket` ctor as `{op: Delegate, ref: COutPacket}`; report-gen descent
  then fails ("delegate to COutPacket: not in export") and the report isn't written.
  Fix = strip that one call from the export entry (it is the packet ctor, not a wire
  read; other versions' entries omit it). A missing deep helper (`sub_XXXX not in
  export`) is fixed by a deeper harvest that pulls it in.
- **Distrust IDB function names.** Ground truth for a send op is the integer in
  `COutPacket::COutPacket(&pkt, OPCODE)`, not the symbol (a function once read as
  `TryDoingBodyAttack` was actually `SetDamaged`); the csv-seeded registry opcode can
  also be off-by-one. Cross-check the COutPacket opcode against the registry.
- **jms has two IDBs.** The retail dump is SMC / control-flow-virtualized (attack sends
  encrypted at rest; Hex-Rays fails, opcode hidden). Use the clean `*_U_DEVM` build.
- **Naming unnamed senders** across versions: the byte signature
  `6A <op> 8D 8D ?? ?? ?? ?? E8` (push opcode; lea ecx; call COutPacket ctor) uniquely
  locates a send site; structure-match to a named twin in another version. Batch via
  IDA-harvest subagents, ONE IDB at a time (`select_instance` is shared global state).

## Is this cell `n-a`? (proving absence)

Marking a cell `n-a` ("op absent from this version") is a claim, not a default —
it is held to the **same evidentiary bar as a positive verification**, not a lesser
one. Absence needs positive proof, not a failed search.

1. **A failed name/region search is absence-of-evidence, not evidence-of-absence.**
   Searching for a symbol name, or scoping a search to a region the decompiled twin
   happens to occupy in another version, and finding nothing, proves only that the
   search was too narrow — see the teleport-rock v48 case (task-124): the first pass
   scoped its search to `0x700000-0x726000` and concluded `TROCK_ADD_MAP` was
   absent; a second, wider pass found it at `0x71e7f3`. The op existed; the search
   didn't.
2. **Anchor on invariants, never on IDB names or an assumed address range.** The
   things that must be present if the feature is present: the opcode-construction
   site (`COutPacket::COutPacket(&pkt, OPCODE)` for a send; the dispatcher's
   opcode-to-handler jump/switch for a receive), the `itemId`/category gate that
   selects the feature, the `StringPool`/notice ids the feature's UI reads, and the
   shared data structure (e.g. a `CharacterData` field) the family reads or writes.
   An unnamed function is not an absent function (§10's "distrust IDB names").
3. **MANDATORY sibling cross-check.** Before marking a serverbound op `n-a`, decompile
   its same-family clientbound receive handler (and vice versa) on that same version.
   If the receive side decodes/populates the feature's state, the send side exists
   somewhere — keep looking. This is "the receive side proves the send side": a
   feature's ops are correlated, not independent coin flips.
4. **Record a family-inconsistent `n-a`, or `matrix --check` fails.** When a cell is
   `n-a` for an op while a same-family sibling op (declared in
   `docs/packets/feature-families.yaml`) is `verified` on the same version, the gate in
   `packet-audit matrix --check` (task-124) requires a matching entry in
   `docs/packets/feature-na-evidence.yaml` carrying the positive proof from steps 1-3
   (non-empty `evidence:` text — an entry with blank/whitespace evidence does not
   count). Declare new feature families in `feature-families.yaml` as you group
   related ops; the evidence file only shrinks (an entry is removed only when the
   cell is later verified, never to silence the gate).

## Producible prerequisite vs genuine blocker (don't defer what you can produce)

Most reasons a cell "can't" verify are **producible steps, not terminal blockers** —
close them in-session rather than recording a "documented gap / follow-up":
- sender **unnamed** in the IDB → name it (§10 byte-signature + twin match). Unnamed ≠
  unnameable.
- op **not routed** in a version's seed template → wire the route (with a validator).
  Not-routed ≠ shouldn't-be-routed.
- fname **missing/stale in the export**, or **no audit report** → splice the export /
  generate the report (§9–10).

A cell is genuinely blocked only when, after *attempting* the unblock, you hit: an
SMC/control-flow-virtualized or otherwise undecompilable binary (e.g. the jms retail
dump); wire content gated on runtime/server config not on the wire; an op IDA-confirmed
version-absent; a missing IDB; or a decision that is the user's (scope/cost). Verify the
blocker by trying — don't infer it from an absence.

## Failure modes (design §13)
- `evidence pin` fails "not in export" → the citation is unresolvable; harvest +
  surgically splice the fname into the export (§10), or re-harvest (task-081 playbook).
- `matrix --check` "dangling evidence: … has no audit report" → generate + copy the
  report (§9 step 3); a tier-1 cell needs all three artifacts.
- `matrix --check` conflict "implements this op … but this version's template does not
  route its opcode, though another version's does" (`routedElsewhere && !routed`) → a
  real template-wiring gap. Either route it in that version's seed template (if the
  tenant should support it) or don't claim the cell.
- `matrix --check` reports an orphan marker → the marker's ida= address matches neither
  the evidence record nor the audit report; fix the address, never delete the check.
- Hash drift on an existing record → cosmetic decompile churn (incl. editing the export
  entry after pinning) → re-pin; material change → re-verify from step 3.
- `matrix --check` "n-a consistency: … is n-a but sibling … is verified" (task-124) →
  a same-family sibling proves the feature exists on that version; either verify the
  cell, or — only if genuinely absent per "Is this cell `n-a`?" above — record the
  positive proof in `docs/packets/feature-na-evidence.yaml`. Never weaken the gate.
