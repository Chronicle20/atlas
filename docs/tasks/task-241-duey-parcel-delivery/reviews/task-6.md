# Review — Task 6: Packet record (dispatcher yamls, registry entries, template tables)

Commit range given: `2604430f3..91a6e82aa`.

## Scope note (read first)

`git log --oneline 2604430f3..91a6e82aa` shows **one** commit in this range —
`91a6e82aa` ("derive parcel/duey_action mode columns for all 8 versions"). The
brief's "two commits: `a590457c6` partial + `91a6e82aa` continuation" framing
does not match: `a590457c6` is an ancestor of `2604430f3` (`git merge-base
--is-ancestor a590457c6 2604430f3` succeeds), i.e. it landed before this
range starts. This review evaluates the diff actually in the given range
(`91a6e82aa`, the IDA continuation) and treats `a590457c6`'s content as
already-landed context, reading it where the continuation depends on it (the
v83 baseline columns, the header comments it wrote).

**Tooling limitation, disclosed up front:** this review session exposes only
`Read`/`Bash`/`Write` — no `ida-pro` MCP tool (`idb_list`/`func_query`/
`decompile`) is present, despite the task instructions asserting IDA access
with open sessions. I could **not** literally re-run a decompile against any
of the v72/v79/v84/v87/v92/v95/jms_v185 IDBs myself. Everything below that
depends on decompiled bytes is verified by the only means available without
that tool: internal cross-consistency of the artifact (do the cited hex/decimal
addresses convert correctly, do the claimed opcodes match the registry entries
they're claimed to match, does the claimed delta pattern hold against
independently-computed neighboring deltas, does a pre-existing in-repo partial
IDA export corroborate the claim), not independent decompilation. This is
reported under **Not evaluable** below and should not be read as a "the bytes
are correct" verdict — it is "the report's claims are internally consistent
and match all other repo evidence I could check by other means."

## What I checked and found

### 1. v79 `DUEY_ACTION` opcode (63 / 0x3F)

- `docs/packets/registry/gms_v79.yaml:2910-2921` — new entry, `opcode: 63`,
  `ida.address: 6821647`. `6821647 == 0x68170f` (confirmed by direct
  conversion), matching the `CTabSend::SendParcel @0x68170f` address cited in
  both the registry note and `duey_action.yaml`'s header comment — internally
  consistent, not a copy/paste mismatch.
- Delta-pattern cross-check (computed independently, not taken from the
  report's prose): in the same registry block, `HIRED_MERCHANT_REQUEST` is
  v83=63/v79=61 (delta -2) and `ITEM_SORT` is v83=69/v79=67 (delta -2)
  (`docs/packets/registry/gms_v83.yaml:2257-2261`, `gms_v79.yaml` neighboring
  entries). `DUEY_ACTION` v83=65 → v79=63 is delta -2, consistent with both
  neighbors. This corroborates the claimed value but is pattern-consistency,
  not proof — the report itself makes the same caveat and says the delta was
  "confirmed directly" against 4 send-sites, which I could not independently
  re-run.
- No opcode collision: v79 opcode 63 is also used by `ALLIANCE_OPERATION`
  (clientbound) — different direction, different namespace, not a collision.
- `docs/packets/ida-exports/gms_v79.json:7395-7446` — a pre-existing (not
  authored by this commit) partial IDA export of `CParcelDlg::OnPacket
  @0x683709` — matches the `@0x683709` address cited in `parcel.yaml`'s
  header comment for the v79 column, and independently corroborates arms
  8/23/25/27 as explicit switch cases on v79 (see §2).

**Verdict on this point:** internally consistent, corroborated by
independent-of-the-report artifacts (opcode-delta arithmetic, pre-existing
export). Byte value itself is **not evaluable** by this reviewer without IDA
access — flagged, not silently approved.

### 2. Claim that v72/v79/v84/v87/v92/v95 are byte-identical to v83 across all 21 `parcel.yaml` keys

- `docs/packets/dispatchers/parcel.yaml:67-87` — confirmed by direct read:
  every key's `modes:` map has identical integer values for
  gms_v72/v79/v83/v84/v87/v92/v95 (8 through 28), only `jms_v185` diverges.
- Cross-checked against the **pre-existing** (not authored by this task)
  `docs/packets/ida-exports/gms_v72.json` and `gms_v79.json` partial exports:
  both independently show explicit switch cases `a1==8`, `a1==23`, `a1==24`
  (guarded), `a1==25`, `a1==27` for `CParcelDlg::OnPacket` — the same 5
  body-bearing arm numbers the report claims match v83. This is real,
  in-repo, pre-existing evidence (not something the implementer could have
  fabricated to fit the claim, since it predates this task) and it
  corroborates the body-bearing-arm claim for 2 of the 6 GMS versions.
- The other 4 versions (v84/v87/v92/v95) and the 13 bodyless notice-only arms
  on all 6 versions have **no independent artifact** in this repo for me to
  cross-check against — those rest entirely on the report's decompile
  citations (function addresses per version, e.g. `gms_v84 @0x70db9f`,
  `gms_v87 @0x7346db`). **Not evaluable** without IDA access.

### 3. `jms_v185` partial column — honestly represented, not silently defaulted

- `docs/packets/dispatchers/parcel.yaml:34-52` header comment explicitly
  documents the shift-not-uniform-delta finding and names exactly which 7
  keys are populated and why the remaining 14 are left unset.
- Verified by direct count: `grep -c jms_v185: parcel.yaml` → 8 occurrences
  (1 in the `opcodes:` block + 7 in `operations:`), matching "7 of 21"
  exactly — `OPEN(10)`, `SUCCESSFULLY_SENT(19)`, `PARCEL_REMOVED(24)`,
  `PARCEL_ARRIVED(25)`, `ALARM_NAMED(26)`, `OPEN_QUICK(27)`,
  `ALARM_GENERIC(28)`.
- Confirmed the unset keys are genuinely **absent** from the YAML map (no key
  at all), not present-with-a-v83-fallback-value — checked both the yaml
  source and the regenerated `template_jms_185_1.json` diff, which shows only
  those same 7 keys in `options.operations` for the `Parcel` entry.
- Confirmed the runtime consumer (`libs/atlas-packet/resolve.go`,
  `ResolveCode`) treats a missing key as a **loud** failure — logs an error
  and returns sentinel `99` ("will likely cause a client crash") — not a
  silent value. So an absent jms_v185 key cannot silently resolve to the v83
  byte at runtime either. This closes the "silent default" concern raised in
  the task: the gap is honest at both the data layer and the resolution
  layer.
- `duey_action.yaml`'s jms_v185 column, by contrast, is **fully** populated
  (SEND=3, RECEIVE=5, DISCARD=6, CLOSE=8, uniform +1 shift) — consistent with
  the report's claim that DUEY_ACTION (unlike PARCEL) had an unambiguous
  1:1 mapping across all 4 keys. No internal contradiction between the two
  files' jms_v185 completeness.

**Verdict:** PASS — the gap is honestly represented in both the yaml and its
header comment, and the resolution mechanism does not paper over it.

### 4. Generator-owned files — reproducibility confirmed by rerun

Ran the generator fresh against the worktree as it stood after `91a6e82aa`:

```
$ go run ./tools/packet-audit operations
... (8 "writer absent: DueyAction" notes, expected — no opcodes: map in duey_action.yaml)
operations: wrote 0 template(s)
$ git status --porcelain   → no diff
$ go run ./tools/packet-audit matrix
wrote docs/packets/audits/STATUS.md and docs/packets/audits/status.json
$ git status --porcelain   → no diff
$ go run ./tools/packet-audit fname-doc --check
fname-doc check OK (268 structs without an audit report carry no fname)
```

Both `operations` and `matrix` produced **zero** diff on rerun — the seed
templates and `STATUS.md`/`status.json` are genuinely generator-reproduced,
not hand-edited. PASS.

### 5. Coverage promotion — confirmed NOT done

Diffed `docs/packets/audits/status.json` line-by-line
(`git diff 2604430f3..91a6e82aa -- .../status.json`): the only content
change is the `DUEY_ACTION` entry's `gms_v72`/`gms_v79` cells moving from
`"state": "n-a", "opcode": -1` to `"state": "incomplete", "opcode": <real>`
(64/63) — plus the block's position shifting (regenerated sort order). Every
cell state present after the change is either `"n-a"` or `"incomplete"` —
grep confirms 0 occurrences of `"verified"`/`"covered"`/any promoted state
anywhere in the diff. `STATUS.md`'s ✅/❌ glyphs for the `DUEY_ACTION` row are
unchanged (still `❌` throughout) except the v72/v79 opcode cells filling in
from blank. This is Task 6's job (record); nothing here does Task 28's job
(promote). PASS.

### 6. Module-local verification — matches report exactly

```
$ cd tools/packet-audit && go build ./... && go test ./...
FAIL github.com/.../cmd  (TestFamilyCapRealTreeClean only)
ok   (all other 11 packages)
$ go run ./tools/packet-audit dispatcher-lint
docs/packets/dispatchers/parcel.yaml  FAM-CAP  ... (1 violation, as expected)
```

Isolated `-run TestFamilyCapRealTreeClean -v` confirms it's the only `FAIL`
line in `cmd`. This is the pre-declared, expected failure (Task 7's job to
clear) — not reported as a defect here, per the task's explicit instruction.

### 7. `duey_action.yaml` internal consistency

Every version's `duey_action.yaml` header-comment opcode claim
(v72=64, v79=63, v83=65, v84=65, v87=68, v92=71, v95=70, jms_v185=57) matches
that version's actual registry `DUEY_ACTION.opcode` entry exactly (checked by
extracting each registry file's `DUEY_ACTION` block directly) — no
transcription drift between the dispatcher-doc comment and the registry.

### Minor, non-blocking observations

- `duey_action.yaml` has no `opcodes:` map, while 2 of the other 3
  serverbound dispatcher docs in the repo (`cash_shop_coupon_code.yaml`,
  `character_interaction_handle.yaml`) do carry one and 1
  (`cash_shop_operation_handle.yaml`) doesn't — the convention is genuinely
  mixed in this repo, so omitting it here is not a deviation from an
  established norm, but it is worth flagging that the choice is inconsistent
  across the family and a future consolidation pass might want to reconcile
  it. Confirmed non-fatal either way: `operations.go:30` documents "a writer
  absent from a template is reported, not failed," and the rerun above
  confirms this produces only stderr notes, not an `operations --check`
  failure.

## Not evaluable

1. **The actual decompiled byte values** for the `parcel.yaml` columns of
   gms_v72/v79/v84/v87/v92/v95 (26 keys × 6 versions) and the
   `duey_action.yaml` columns for all 7 non-v83 versions, and the v79
   `DUEY_ACTION` opcode itself — I have no `ida-pro` MCP tool in this
   session (only `Read`/`Bash`/`Write`), so I could not run `idb_list`,
   `func_query`, or `decompile` against any of the named IDBs to
   independently re-derive a single byte. This is exactly the check the
   task asked a reviewer to perform ("Independently re-derive at least the
   v79 DUEY_ACTION opcode and at least two of the per-version parcel.yaml
   columns"), and I could not do it. What I verified instead (address
   round-trip math, opcode-delta consistency with neighboring registry
   entries, corroboration against 2 pre-existing partial IDA-export files
   that predate this task) is real but is not a substitute for decompiling
   the cited functions myself. **This gap should be closed by a reviewer (or
   the controller) with actual IDA/MCP tool access before treating the
   mode-byte table as fully audited**, given the stated blast radius (Tasks
   7-10 resolve against it).
2. The 4 non-v72/v79 versions' `duey_action.yaml` addresses
   (v84/v87/v92/v95's SEND/RECEIVE/DISCARD/CLOSE site addresses) — same
   limitation, no independent artifact in-repo to cross-check them against
   (unlike v72/v79 which have partial `ida-exports/*.json` files).

## Verdict rationale

Everything checkable without a decompiler — internal consistency, generator
reproducibility, coverage-non-promotion, the jms_v185 gap's honesty, the
expected-and-only FAM-CAP failure, cross-file agreement between the registry
and the dispatcher-doc comments — passes cleanly with no defects found. The
one substantive gap is that the byte-level claims (the actual payload of this
task) could not be independently re-derived in this review session due to a
missing tool, which is the review's own limitation, not a defect found in the
work. Given the stated stakes (four downstream tasks resolve modes against
this table), that gap is significant enough to flag as blocking-pending-tool,
not wave through as approved.
