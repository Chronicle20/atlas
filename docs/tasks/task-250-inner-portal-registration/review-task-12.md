# Review — Task 12 (final): promote USE_INNER_PORTAL across ten versions

Commit under review: `25a72cdbc` (range `bb7ec8dbd..25a72cdbc`)
Brief: `.superpowers/sdd/plan/task-12-brief.md` (CONTROLLER AMENDMENTS section, 10-wide scope)
Report: `.superpowers/sdd/plan/reports/task-12-report.md`

## Scope confirmed

Reviewed the full diff of `25a72cdbc` against `bb7ec8dbd`: ten IDA export
JSONs, ten evidence YAMLs, the test-file markers, regenerated
`STATUS.md`/`status.json`, and the `design.md:88` correction. Independently
re-ran `matrix --check`, `fname-doc --check`, `operations --check`, and
diffed every export file's `functions` map (before vs after, keyed by
fname) with Python/`json`, plus stood up a throwaway detached worktree at
the parent commit (`bb7ec8dbd`) to get `matrix --check`'s baseline output —
removed after use, per the "no `git stash`" hazard (used `git worktree add
--detach` instead).

## Findings

### BLOCKING — the "splice" stripped `note`/`notes`/`region` metadata from hundreds of unrelated entries across all ten export files, and this is the actual cause of the ~1120-line `matrix --check` decompile-drift failure the report calls "pre-existing"

Diffing each export's `functions` map by key (not by `git diff` line count)
shows exactly one function added per file
(`CUserLocal::TryRegisterTeleport`, address matching the amendment table)
— but also a large number of **other, pre-existing entries silently
losing their `note`/`notes`/`region`/`_note`/`discriminator` fields**:

| version | entries with a field dropped |
|---|---|
| gms_v48 | 19 |
| gms_v61 | 28 |
| gms_v72 | 26 |
| gms_v79 | 65 |
| gms_v83 | 205 |
| gms_v84 | 229 |
| gms_v87 | 203 |
| gms_v92 | 16 |
| gms_v95 | 163 |
| gms_jms_185 | 256 |

Example (`docs/packets/ida-exports/gms_v92.json`,
`CUserLocal::OnIncComboResponse`): the before entry carries a hand-traced
`note` explaining a Hex-Rays decompile failure at `0x8fe8e0` and the manual
disassembly walk used instead (task-217 provenance); the after entry has
the identical `calls`/`address`/`direction` but the `note` field is simply
gone. Same pattern for `CTradingRoomDlg::OnTrade#TradeConfirm` and 14
others in that one file alone. Across the other nine files the same thing
happens to `region: "GMS"` and `notes:` fields on ITC/MTS/guild/messenger
entries that had nothing to do with this task.

This directly violates the brief's own warning: "Never overwrite a
committed export — the harvest is not idempotent and a full re-export
drifts ~150 unrelated entries. Use the surgical `--splice` path... Confirm
each run prints `export: spliced ... (one entry merged, others
preserved)` and that `git diff --stat` on the export shows a small delta,
not a rewrite." `git diff --stat` on these ten files shows deltas of
2,750–31,399 changed lines each (`docs/packets/ida-exports/gms_v84.json`:
+15,558/-15,841). The report's Step 1 write-up asserts the opposite: "No
export file's other ~150+ pre-existing entries changed (`git diff` on each
file is a small, isolated addition)." That claim is false and is
contradicted by the diff the report itself was working from.

This is not cosmetic. `matrix --check`'s "decompile hash drift" check
hashes the exported function body (`calls`) plus its surrounding record
against the evidence-pinned `decompile_sha256`. I confirmed independently,
by running `matrix --check` against the parent commit in a throwaway
detached worktree (`git worktree add --detach <scratch> bb7ec8dbd`, removed
after), that the parent commit's `matrix --check` output is **5 lines**
(two `n-a evidence consumed` notes and the two expected "STATUS.md/
status.json stale" lines — no drift entries at all), while this commit's
`matrix --check` output is **1124 lines**, entirely `decompile hash drift`
across ~30 unrelated packets (buddy, character, messenger, monster, pet,
guild, field, party, teleportrock, login, note, summon, incubator, fame,
…) × multiple versions each. None of the drifted lines mention `portal` —
that part of the report's claim is correct — but "pre-existing, not
introduced here" is not: the drift did not exist in the parent commit and
is a direct downstream consequence of this commit's own export edits
(`docs/packets/ida-exports/*.json`).

The report's "Concerns for the controller" section flags the drift as a
scope question ("recommend the controller treat `matrix --check`'s
portal-scoped result as the gate... track the pre-existing drift
separately") without ever verifying the "pre-existing" premise against the
parent commit — the one check the task brief and the review directive
both call for. That premise is false, and the fix belongs in this task
(scoped to the splice mechanism, not a 30-packet re-audit): the splice
needs to preserve every field on every entry it doesn't touch, not merely
its own new key.

Blocking because: (a) it is real data loss of curated, hand-traced IDA
provenance (task-217/task-096 notes) checked into the repo, silently, in a
commit whose stated purpose is a single-op promotion; (b) it produces a
concrete, independently-reproducible regression (~1120 new `matrix --check`
failures) that did not exist before this commit and is misattributed in
the report as unrelated pre-existing drift; (c) the final-verification
checklist item "`matrix --check` exit 0" cannot be waived on the strength
of a diagnosis that turns out to be wrong.

### Everything else checked — no other blocking findings

1. **Ten markers, ten addresses (PASS).**
   `libs/atlas-packet/portal/serverbound/inner_portal_test.go:13-22` carries
   exactly the ten markers from the amendment table, byte-for-byte:
   `gms_v48=0x6a5462`, `gms_v61=0x7aa1e3`, `gms_v72=0x864562`,
   `gms_v79=0x8afc42`, `gms_v83=0x957b74`, `gms_v84=0x995c92`,
   `gms_v87=0x9da037`, `gms_v92=0x8f85c0`, `gms_v95=0x913690`,
   `jms_v185=0xa2218f`. No `gms_v12` marker, no `gms_v12` evidence file, no
   `gms_v12` export splice (`git diff --stat` confirms no `gms_v12.json`
   touched). Cross-checked `gms_v84`'s address against
   `structures/gms_v84.md` (caller-walk derivation, independently landing
   on `0x995c92`) — matches.

2. **Exports spliced, not rewritten — content-wise for the target entry;
   line-diff-wise, NOT surgical (see blocking finding above).** The one
   new key per file (`CUserLocal::TryRegisterTeleport`) is correctly
   added and its `calls`/`address` shape is sane per version (e.g.
   `gms_v48` is the documented 5-field/no-`Decode1` shape; `gms_v92`/
   `gms_v95` each carry one extra leading `Delegate`/`Unresolved`
   call, consistent with the report's note). But the surrounding ~150+
   unrelated entries per file are not byte-preserved — see the blocking
   finding.

3. **Evidence records tool-written, correct `--verifies` target (PASS).**
   All ten YAMLs under `docs/packets/evidence/<version>/portal.serverbound.PortalInnerPortal.yaml`
   carry `ida.address` matching the amendment table, a `decompile_sha256`,
   and `verifies: [...#TestInnerPortalGoldenBytes]`. Confirmed
   `TestInnerPortalGoldenBytes` is a real function at
   `libs/atlas-packet/portal/serverbound/inner_portal_test.go:23`. No sign
   of hand-editing (all fields present and internally consistent with the
   export addresses).

4. **All ten cells promoted (PASS, read the regenerated file directly).**
   `docs/packets/audits/STATUS.md:615` —
   `USE_INNER_PORTAL | CUserLocal::TryRegisterTeleport |  | 0x050 ✅ | 0x05D ✅ | 0x064 ✅ | 0x063 ✅ | 0x065 ✅ | 0x065 ✅ | 0x068 ✅ | 0x070 ✅ | 0x071 ✅ | 0x060 ✅` —
   all ten ✅. `gms_v83`/`gms_v84` both show `0x065`, which matches the
   documented byte-identity between v83 and v84 for this op
   (`structures/gms_v84.md:33-36`), not a duplication bug.

5. **`matrix --check` — the drift claim, verified independently, and found
   wrong.** See the blocking finding above: no `portal` entries in the
   1124-line failure output (that part is true), but the drift itself is
   newly introduced by this commit, not pre-existing.
   `git diff bb7ec8dbd..25a72cdbc -- docs/packets/audits/status.json` does
   in fact touch the drifted packets' entries (the whole file is a 2610-line
   regenerated diff and the drift is real, reproducible downstream state,
   not stale leftover data) — the report's assertion that "their JSON
   entries are byte-identical before and after" was not independently
   re-checked by the report and does not hold up against the parent-commit
   `matrix --check` baseline.

6. **`--ida-url` substitution (PASS on the substance).** All three
   spot-checked addresses (`gms_v48=0x6a5462`, `gms_v92=0x8f85c0`,
   `gms_v95=0x913690`) match the amendment table and, for `gms_v84`,
   independently cross-check against the caller-walk derivation recorded in
   `structures/gms_v84.md`. The routing substitution itself (per-port
   `--ida-url` instead of `--ida-database <hash>`) is adequately justified
   and its output is verifiably correct for the target entry; it is a
   reasonable, disclosed deviation from the brief's literal invocation
   form given the live MCP server rejected the documented flag.

7. **Scope hygiene (PASS).** `git diff --stat bb7ec8dbd..25a72cdbc` touches
   only the ten export JSONs, ten evidence YAMLs, the test-marker block,
   `STATUS.md`/`status.json`, and `design.md`. No `services/atlas-ui/src/pages/*.tsx`
   file appears in the commit diff (those remain uncommitted working-tree
   edits per `git status --short`, correctly left alone). No codec,
   handler, registry, or template file is touched.
   `libs/atlas-packet/portal/serverbound/inner_portal_test.go`'s diff is
   exactly the ten added marker comment lines — no other code change.

8. **`design.md:88` correction (PASS).** Now reads: `jms_v185 |
   0xa2218f (named) | push 60h at 0xa22313 (corrected during Task 12
   promotion — this row originally read 0xa2230e; the opcode, 96 / 0x060,
   is unaffected) | ...`. Matches the amendment exactly.

9. **`fname-doc --check` / `operations --check` (PASS, re-run
   independently):** both exit 0. `fname-doc --check` reports 269 structs
   without an audit report carry no fname (expected, per
   `tools/packet-audit/cmd/fnamedoc.go:126-138`); `operations --check`
   reports 0 absent-writer notes.

## Not evaluable

- Whether the `--ida-url` per-port routing genuinely targeted the correct
  binary for all ten versions (vs. just the three spot-checked) could not
  be independently re-verified without live IDA-MCP access from this
  review session; relied on the amendment-table cross-check for all ten
  and the independent structures-doc cross-check for `gms_v84`.
- Whether the note/region stripping is a `packet-audit export --splice`
  tool defect (re-serializes every entry, dropping unknown/optional
  fields) versus an artifact of how this task's invocations were run could
  not be root-caused without reading `tools/packet-audit`'s export/splice
  implementation, which is out of this task's touched-file scope; flagged
  here as a fact (verified) rather than diagnosed to a specific tool line.

## Verdict rationale

The op-level work — ten markers, ten evidence pins, ten promoted matrix
cells, the docs correction, scope hygiene — is all correct and verified
independently. But the commit has a real, verified side effect the report
misdiagnoses: it strips curated annotation fields from hundreds of
unrelated IDA export entries across all ten files, and that stripping is
the demonstrated cause of ~1120 new `matrix --check` failures that did not
exist in the parent commit. Landing this as-is either loses that
documentation permanently or ships a broken `matrix --check` baseline
under a report that tells the next person it's pre-existing and safe to
ignore.
