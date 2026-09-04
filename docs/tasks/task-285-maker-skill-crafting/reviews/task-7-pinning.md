# Review: MAKER_SKILL evidence-pinning campaign (task-7-pinning)

Commits reviewed: `640b8caa1` (verify(packets): promote MAKER_SKILL...), `f712ada30`
(docs(task-285): record the MAKER_SKILL pinning report).

Scope: the export splice, registry `packet:` additions, evidence yaml, marker
promotion in `maker_skill_test.go`, and STATUS.md/status.json regeneration.
No `ida-pro` MCP access — the decompile content of the eight spliced export
entries (opcode literal, `calls` sequence, addresses) is judged only for
internal consistency across the diff and against the registry/report; it is
**not** independently verified against the IDBs. That is out of my evaluable
surface and is called out below rather than silently accepted.

## 1. Splice is additions-only

Confirmed directly:

```
git diff --numstat 640b8caa1^ 640b8caa1 -- docs/packets/ida-exports/
62  0  docs/packets/ida-exports/gms_jms_185.json
62  0  docs/packets/ida-exports/gms_v72.json
62  0  docs/packets/ida-exports/gms_v79.json
62  0  docs/packets/ida-exports/gms_v83.json
62  0  docs/packets/ida-exports/gms_v84.json
62  0  docs/packets/ida-exports/gms_v87.json
62  0  docs/packets/ida-exports/gms_v92.json
62  0  docs/packets/ida-exports/gms_v95.json
```

Inspected the actual hunks for all eight files (not just the numstat): each
diff inserts a single new top-level key `"CUIItemMaker::RequestItemMake": {...}`
immediately after the `"functions": {` opening brace, with no `-` lines
anywhere in the file. No existing export entry is touched. **PASS.**

## 2. Opcode cross-checks against the registry

Checked `docs/packets/registry/<version>.yaml` directly for the `MAKER_SKILL`
op entry inserted alongside the splice:

| version | registry `opcode:` | claimed `COutPacket` | match |
|---|---|---|---|
| gms_v72 | 112 (`gms_v72.yaml:2649`) | 0x70=112 | yes |
| gms_v79 | 111 (`gms_v79.yaml:3145`) | 111 | yes |
| gms_v83 | 113 (`gms_v83.yaml:2541`) | 0x71=113 | yes |
| gms_v84 | 113 (`gms_v84.yaml:3243`) | 113 | yes |
| gms_v87 | 116 (`gms_v87.yaml:2656`) | 0x74=116 | yes |
| gms_v92 | 124 (`gms_v92.yaml:2875`) | 0x7C=124 | yes |
| gms_v95 | 125 (`gms_v95.yaml:2963`) | 125 | yes |
| jms_v185 | 108 (`jms_v185.yaml:2614`) | 0x6C=108 | yes |

All eight match exactly. **PASS.** (The address/opcode/notes text inside the
exported `functions` entry itself is agent-authored prose describing an IDA
decompile I cannot re-run; I can only confirm it is internally consistent with
the registry number, not that the decompile is faithful to the IDB.)

## 3. Report-less promotion path — documented, not a manufactured carve-out

Read `tools/packet-audit/internal/matrix/grade.go:198-222` (the `!a.hasReport`
branch) directly. It is a pre-existing, commented code path:

> "No IDA-export audit report for this op. A committed golden byte-test
> (packet-audit:verify marker) backed by fresh evidence still proves the exact
> wire ... This requires the registry entry to carry a `packet:` link."

And `tools/packet-audit/cmd/matrix.go:388-403` (`registryDeclaresPacket`) is
explicitly attributed to a prior commit (`6c202cb7`), predating this campaign,
with its own doc comment describing the same exception for the dangling-
evidence `--check` rule. This is a documented, previously-established
promotion mechanism that this campaign used as designed — not a carve-out
invented to force a `✅` here. **PASS.**

## 4. Markers

Diffed `maker_skill_test.go` for the commit: exactly the eight
`// EVIDENCE (pin pending ...)` blocks are replaced 1:1 by a single
`// packet-audit:verify packet=character/serverbound/MakerSkill version=<v>
ida=<addr>` line each. Verified per-version address in each new marker matches
the `ida=` in the corresponding `// IDA evidence (...)` comment immediately
above it (unchanged) — e.g. v72 marker `ida=0x760cc3` matches the IDA-evidence
comment's `CUIItemMaker::RequestItemMake@0x760cc3` two lines above. Confirmed
for all eight (v72/v79/v83/v84/v87/v92/v95/jms_v185).

`git diff --stat` for the commit shows no change to
`libs/atlas-packet/character/serverbound/maker_skill.go` — the codec is
untouched. **PASS.**

## 5. Forward constraint re: partial routing under Task 25 — NOT SUPPORTED BY THE CODE

The report/commit claims: "because the op is routed nowhere, `routedElsewhere`
is false everywhere and no wiring conflict is raised — but when Task 25 routes
it, it must route all eight versions at once, since a partial routing would
flip the unrouted versions to `🟥`."

I traced this through `grade.go` and `build.go` and the claim does **not**
hold as the grading code is written:

- The `!a.routed && a.routedElsewhere` conflict check lives at
  `grade.go:227`, strictly *after* the `if !a.hasReport { ... return }` block
  (`grade.go:198-222`). It is reachable only when `hasReport == true` for that
  version's cell.
- `hasReport` (`gradeOpCell`, `grade.go:121` `findReport`) and the
  `worstCandidateCell`/`fnameWriters` lookup in `build.go:500-509` are both
  keyed off `in.Reports[version]`, which in turn is built from committed
  per-version audit-report JSON files, which `tools/packet-audit/cmd/run.go`
  (`lookupFName`, `run.go:3956-3988`) only produces for a version whose own
  seed template (`services/atlas-configurations/seed-data/templates/...`)
  already binds a writer/handler at that op's opcode — the *same* seed
  template that also drives `in.Routed[version]` (per `build.go:281-313`'s own
  comment: "routed by that version's own opcode").
- Consequently `hasReport(v)` and `routed(v)` are coupled through the *same*
  per-version artifact. For a version Task 25 leaves unrouted, `hasReport`
  stays false there too (no report is generated for an op the version's own
  template doesn't bind), so the cell keeps taking the `!hasReport` branch —
  which never references `a.routed` or `a.routedElsewhere` at all
  (`grade.go:198-222`, confirmed by reading the full branch body). The
  existing marker + fresh evidence keep grading it `✅` regardless of what
  any other version's template does.

So a partial Task-25 routing would not visibly flip the still-unrouted
versions to `🟥` — under the current code they would silently **stay** `✅`
via the report-less fixture path, masking the fact that those versions still
don't actually dispatch the opcode. That is arguably a worse outcome than the
one described (a silent false-positive rather than a visible red flag), and
it means the "must route all eight at once" framing, while a reasonable
caution to keep, is not actually a safety net enforced by `grade.go` today —
Task 25 planning should not rely on the matrix to catch a partial rollout of
MAKER_SKILL.

**This is a finding against the report's claim, not against the committed
diff's correctness** — nothing in `640b8caa1`/`f712ada30` needs to change to
fix it, and it doesn't block this promotion. It is flagged as non-blocking
because it does not affect the two reviewed commits' contents, but it is a
documentation/reasoning defect that should not be carried into Task 25's plan
uncorrected.

## 6. IDB renames (`sub_8524B7` / `sub_7AFDC0` → `?RequestItemMake@CUIItemMaker@@IAEHXZ`)

Not a defect, confirmed sanctioned: `docs/packets/audits/VERIFYING_A_PACKET.md:176-178`
("Naming unnamed senders across versions: the byte signature ... uniquely
locates a send site; structure-match to a named twin in another version.")
explicitly documents this as an expected step in the verification playbook.
This mutates shared IDBs outside the git repo (as the agent notes) but is the
sanctioned workflow, not an out-of-scope escalation.

## Gates re-run against the committed tree (not the report's transcript)

Built `tools/packet-audit` from the committed `640b8caa1`/`f712ada30` tree
(clean working tree, `git status --short` showed no tracked-file changes) and
ran all four gates directly:

```
packet-audit matrix --check           EXIT=0
packet-audit fname-doc --check        EXIT=0  "fname-doc check OK (272 structs without an audit report carry no fname)"
packet-audit operations --check       EXIT=0  "operations check OK (0 absent-writer note(s))"
packet-audit dispatcher-lint          EXIT=0  "dispatcher-lint: clean"
```

All four match the report's claimed exit codes and output. Also ran
`go test ./character/serverbound/...` in `libs/atlas-packet` — both packages
pass.

## STATUS.md / status.json regeneration

`git diff` on `status.json` shows exactly the eight MAKER_SKILL cells
(gms_v72/v79/v83/v84/v87/v92/v95/jms_v185) flipping `incomplete` →
`verified` with the `"note": "no audit report"` line dropped, plus the eight
per-version export hash lines updating (expected, since the export files
changed). No other row/cell in the diff. **PASS.**

## Verdict rationale

Every one of the five scrutiny items the reviewer was asked to check came back
clean except item 5, and item 5 is a defect in the *claim*, not in the
committed artifacts — the pin itself (splice, opcodes, markers, gates,
STATUS.md) is correct and internally consistent. This is exactly what
`APPROVED_WITH_FINDINGS` is for: the commits are sound, but the forward-looking
claim in the commit message / report about Task 25's safety net is wrong and
should not be relied upon un-corrected.

## Not evaluable

- The content of the eight spliced IDA-export entries (address, `calls`
  sequence, per-arm Encode order) cannot be checked against the IDBs — no
  `ida-pro` MCP access in this session. Judged only for internal consistency
  (opcode literal vs. registry, marker address vs. evidence-comment address),
  both of which pass.
