# Review — Task 1: derive the GMS wire layout on v72/79/83/84/87/92

Range: `0e54b1b1a..b9bd002d1` (1 commit, `b9bd002d1`)
Scope: `docs/tasks/task-281-map-back-effects/structures/{gms_v72,gms_v79,gms_v83,gms_v84,gms_v87,gms_v92}.md` — new files only, no other paths touched. `git diff --stat` confirms 6 files changed, 522 insertions, 0 deletions, nothing outside `structures/`.

## Scope confirmation

Diff matches the brief exactly: six new structures records, documentation only, no Go, no registry writes (registry files are read via `cat`/`grep` provenance citations only, never modified in the diff). No scope violation.

## Checklist results

### 1. All six files exist and follow the skeleton

PASS. All six files (`gms_v72.md`, `gms_v79.md`, `gms_v83.md`, `gms_v84.md`, `gms_v87.md`, `gms_v92.md`) contain `## Router`, `## SET_BACK_EFFECT read order` (with the branch-shape prose and a `Verdict vs the gms_v95 reference` line), and `## CLEAR_BACK_EFFECT` (with its own `Verdict vs the gms_v95 reference` line). Confirmed via `grep -n "^## \|Verdict vs\|FINDING"` on each file.

### 2. Per-version verdict and read-order table

PASS. Every file's `SET_BACK_EFFECT read order` table is `Decode1 byte nEffect / Decode4 int32 nFieldID / Decode1 byte nPageID / Decode4 int32 tDuration`, matching the required 10-byte layout exactly, in every one of the six files (`gms_v72.md:39-44`, `gms_v79.md:44-49`, `gms_v83.md:36-41`, `gms_v84.md:35-40`, `gms_v87.md:37-42`, `gms_v92.md:57-62`). Every file states `Verdict vs the gms_v95 reference (design §1.1): IDENTICAL` for SET and `... (design §1.2): IDENTICAL` for CLEAR. No divergence reported for any of the six versions, consistent with the report.

### 3. Router case number matches the version's opcode table

PASS for all six, cross-checked against the constraint table (v72=117/0x075, v79=121/0x079, v83=128/0x080, v84=131/0x083, v87=136/0x088, v92=143/0x08F):
- v72: `gms_v72.md:23` case 117 (`0x75`) — matches.
- v79: `gms_v79.md:20` case 121 (`0x79`) — matches.
- v83: `gms_v83.md:16` case 128 (`0x80`) — matches.
- v84: `gms_v84.md:17` case 131 (`0x83`) — matches.
- v87: `gms_v87.md:18` case `0x88` (136) — matches.
- v92: `gms_v92.md:22` case 143 (`0x8F`) — matches.

Each file also independently cross-checks its opcode against the registry file at a cited line number; I re-verified these citations directly against the registry files (below).

### 4. Internal consistency (address citations, thunk targets)

PASS. Traced every non-trivial cross-reference:
- v72/v79/v83/v87/v92 CLEAR handlers are all explicitly shown as thunks with a named or positionally-inferred target (`sub_5F5A1B` for v72, `ReloadBack@0x61443e` for v79, `ReloadBack@0x644491` for v83, `ReloadBack@0x67dba7` for v87, `sub_612D80` for v92), and each file is explicit about whether the name is proven or inferred (v72 and v92 explicitly flag the target name as a positional/behavioral inference, not proven — `gms_v72.md:74-78`, `gms_v92.md:100-104`). The load-bearing claim (empty body / zero packet reads) is proven independently of the name in both cases.
- v84's router/handler addresses (`0x659e3c` SET, `0x65a241`→`0x659d08` CLEAR) trace exactly to `design.md:68-69` and to `docs/packets/registry/gms_v84.yaml:882-889` (SET) and `:898-905` (CLEAR) — verified by direct `sed` read of those registry lines; the note text at `gms_v84.yaml:889` and `:905` contains the same addresses and body-shape description the structures file transcribes.
- No address in any of the six files is contradicted between its Router section and its SET/CLEAR sections.

### 5. Step 0 outcome recorded

PASS. All six files carry the `Step 0 (already implemented?): NO. grep -rl BackEffect libs/atlas-packet/ returns nothing...` line (`gms_v72.md:6-7`, `gms_v79.md:6-7`, `gms_v83.md:6-7`, `gms_v84.md:6-7`, `gms_v87.md:6-7`, `gms_v92.md:6-7`).

### 6. Address provenance — decompile or explicit citation

PASS for all six. v79/v83/v87/v92 addresses are attributed to live-IDB decompiles shown inline as decompiled C or disassembly with named session IDs. v72 and v84 are explicitly marked as transcriptions ("Source: ... transcribed from design §1.1", "Source: transcribed, not re-decompiled") with the design.md line numbers or registry line numbers cited, and I confirmed those citations resolve to real content:
- `design.md:68` (v72 row: `0x5f5b4f`, `sub_54C265`@`0x54c265`) and `design.md:69` (v84 row: `0x659e3c`, `sub_597B59`@`0x597b59`) match `gms_v72.md:37` and `gms_v84.md:33` exactly.
- `docs/packets/registry/gms_v84.yaml:882` (opcode 131) and `:889` (note with router/handler addresses) and `:898`/`:905` (CLEAR opcode 133 and thunk target `0x659d08`) match what `gms_v84.md` cites.
- Registry opcode-crosscheck line citations for v83 (`gms_v83.yaml:659,669`), v87 (`gms_v87.yaml:716,726`), v92 (`gms_v92.yaml:750,762`) all resolve to the exact `opcode:` lines claimed, verified by direct grep/sed.
- No bare, un-provenanced address was found in any file.

### 7. Diff stays inside `structures/`

PASS. Confirmed via `git diff --stat` — only the six `structures/*.md` files are touched; nothing under `libs/atlas-packet/`, `docs/packets/registry/`, `docs/packets/evidence/`, `docs/packets/audits/`, seed templates, or `feature-*.yaml`.

## v72/v79 CLEAR_BACK_EFFECT documentation quality (per controller's framing)

The controller has already accepted the underlying finding (CLEAR_BACK_EFFECT is present on v72 opcode 118 and v79 opcode 122, contrary to the plan's opcode table, which only lists SET opcodes and doesn't cover CLEAR at all for v72/v79). Reviewing only the *rigor of the documentation* of that finding:

- **v72** (`gms_v72.md:55-81`): router arm shown in situ (`else if (a2 == 118)` @ `0x5f59ed` → `sub_5F5F54` @ `0x5f5f54`), the handler's decompiled body is shown as a thunk tail-calling `sub_5F5A1B` with zero arguments, the zero-read claim is stated explicitly ("Packet reads: none. The thunk discards its `CInPacket *` argument..."), and the `ReloadBack` identification for `sub_5F5A1B` is explicitly flagged as an inference, not a proven symbol, with the reasoning (positional adjacency to `OnPacket`) given. A closing `FINDING:` paragraph names the registry gap and defers the resolution to Task 2. This meets the bar.
- **v79** (`gms_v79.md:76-97`): same structure — router arm `else if (a2 == 122)` @ `0x614410` → `CMapLoadable::OnClearBackEffect` @ `0x614977`, decompiled thunk body shown, zero-read claim explicit, target already carries the real symbol `?ReloadBack@CMapLoadable@@QAEXXZ` @ `0x61443e` (no inference needed here), and the same closing `FINDING:` paragraph. This meets the bar.

Both records are internally consistent with their own Router sections (case 118/122 addresses match between the Router C snippet and the CLEAR_BACK_EFFECT section) and with the registry files as they stand today (confirmed above that `gms_v72.yaml` and `gms_v79.yaml` carry no `CLEAR_BACK_EFFECT` op entry, so the "registry gap" claim is accurate, not invented).

## Findings

No blocking findings.

Non-blocking observations (not defects, recorded for completeness):
- `gms_v84.md`'s registry opcode-crosscheck line (`:882`) points at the `opcode:` field of the SET_BACK_EFFECT entry; the file also cites `:889` for the note containing the handler addresses — both resolve correctly, but a reader following only the top-level citation `gms_v84.yaml:889` in the "Files" list of the brief would land one line off from the exact `opcode:` field cited inline. This is a trivial precision-of-citation nit, already resolvable by reading the surrounding block, not a provenance gap.
- The report's "Concerns for the controller" section (three items) is not reproduced inside the structures files themselves beyond the v72/v79 `FINDING:` paragraphs — the v92 router-symbol naming concern and the two-inferred-names concern are documented in-file (`gms_v92.md:9-25`, `gms_v72.md:74-78`, `gms_v92.md:100-104`), so all three concerns are in fact traceable to the files, just not collected in one place. No action needed.

## Not evaluable

None. All six files, the diff, and every cited registry/design line were directly inspected.

## Verdict

APPROVED.
