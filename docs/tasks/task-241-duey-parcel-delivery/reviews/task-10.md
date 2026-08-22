# Review — Task 10: `DUEY_ACTION` serverbound codecs

Range: `707e97efc..586ee9ebb` (commits `6bb465f86`, `586ee9ebb`). Read-only
review; no repo edits made. `matrix --check` was not re-run (already
confirmed green at HEAD per the dispatch brief); `--check`-form
packet-audit subcommands only were used where run at all (none needed to be
run — all evidence gathered by reading files/diffs and decompiling in IDA).

## Priority 1(a) — no `WithResolvedCode` layer

- `docs/packets/DISPATCHER_FAMILY.md:138-175` — "Serverbound dispatcher
  files are out of scope for FAM-CAP" section read in full. It states
  FAM-CAP models a CLIENTBOUND mode-prefix demultiplexer, that a
  serverbound `dispatchers/*.yaml` is a HANDLER routing table populated by
  N independent client call sites each carrying its own discrete struct,
  and that `direction: serverbound` is what opts a file out. Matches the
  implementer's citation. `duey_action.yaml` does carry
  `direction: serverbound` (confirmed in file).
- `libs/atlas-packet/storage/serverbound/operation.go` and
  `operation_store_asset.go` — `grep -rn WithResolvedCode` on both:
  no matches. The named reference pattern genuinely has no
  `WithResolvedCode`/body-function layer, as claimed.
- Mode-byte origin check (the question that actually matters, independent
  of the FAM-CAP argument): `grep -rn 'mode:\s*0x'` and
  `grep -rn 'func(_ byte)'` in `libs/atlas-packet/parcel/serverbound/` —
  both empty. Read `action.go` directly: `Action.Decode` sets
  `m.mode = r.ReadByte()` — the mode byte is read off the wire, never
  hard-coded or literal-assigned. `Action` mirrors
  `storage/serverbound.Operation` field-for-field (single `mode byte`,
  identical `Encode`/`Decode`/`String`/`Operation()` shape). No hard-coded
  mode anywhere in the four new files. This satisfies the no-hard-coding
  rule on its own terms, independent of whether the FAM-CAP scoping
  argument is accepted.
- Byte values: `docs/packets/dispatchers/duey_action.yaml` operations table
  — SEND gms_v83=2/jms_v185=3, RECEIVE=4/5, DISCARD=5/6, CLOSE=7/8. Matches
  exactly.
- **Verdict on 1(a): the deviation is correct.** Both legs hold — the
  FAM-CAP scoping citation is accurate, and independently, no mode byte is
  hard-coded; every mode originates from `r.ReadByte()` and is resolved
  against the YAML by the (out-of-scope, Task 17/18) atlas-channel handler,
  exactly as `storage/serverbound.Operation`'s mode is.

## Priority 1(b) — matrix regeneration needs a second commit

- `tools/packet-audit/cmd/matrix.go:491-492` —
  `toolTreeSHA()` runs `exec.Command("git", "ls-tree", "-r", "HEAD",
  "tools/packet-audit").Output()`. This reads the **committed tree at
  HEAD**, not the working tree. Mechanically confirms the claim: hashing
  `run.go`'s content in the same commit that changes it computes the hash
  against the state at time of the git invocation, but `STATUS.md`
  written in that same commit is then checked against a *later* HEAD (the
  commit that includes it) whose tree-hash of `run.go` differs only if
  `run.go` itself changed within that same commit — the implementer's
  repro (green pre-commit, red immediately post-commit, byte-identical
  file, only HEAD differing) is consistent with this mechanism, and I
  found no alternate hash source (only one `ls-tree` call site exists in
  `matrix.go`).
- `git diff 6bb465f86..586ee9ebb --stat`: `STATUS.md | 2 +-`,
  `status.json | 2 +-`, 2 insertions/2 deletions total. Full diff shows
  only the `Tool:`/`toolSha` line changed in each file — no coverage,
  orphan, or drift entry moved.
- **Verdict on 1(b): confirmed, not merely plausible.** This is a genuine
  structural property of `matrix`'s tool-hash computation, not an
  implementer-error pattern; landing a `run.go` edit and its matrix
  regeneration in the same commit will always go stale the instant that
  commit lands. Splitting into a second commit is the correct fix. (Whether
  earlier "implementer error" diagnoses on this branch should be revisited
  in light of this is outside this task's scope — flagged here for the
  controller, not adjudicated.)

## Priority 2 — codecs

- `Action.Decode` reads one byte, `Mode()` returns it — dispatches
  correctly (`action.go`).
- `ActionReceive` and `ActionDiscard` (`action_parcel_id.go`) are separate
  structs, each with its own `ParcelId() uint32`, `Operation()` string,
  `Encode`/`Decode` — not collapsed, despite the identical
  `uint32 parcelId` wire shape. Matches the brief's constraint.
- SEND asymmetry verified against the v83 decompile directly (not
  accepted from design.md): decompiled `sub_6F36A8` (NPC arm, quick=0) and
  `sub_6F1DF5` (quick arm, quick=1) in IDA session `41f09cce`
  (`v83_Me/MapleStory_dump.exe.i64`, the correct v83 IDB).
  - `sub_6F36A8`: `Encode1(2)` [mode], `Encode1(*(this+8))` [invType],
    `Encode2(*(this+12))` [slot], `Encode2(*(this+16))` [quantity],
    `Encode4(*(this+20))` [mesos], `EncodeStr(recipientName)`,
    `Encode1(0)` [quick=false] — **stops there**, no further encode calls.
  - `sub_6F1DF5`: identical prefix through `EncodeStr(recipientName)` (same
    field types, different member offsets since the quick-arm object has
    extra fields), then `Encode1(1)` [quick=true], `EncodeStr(message)`,
    `Encode4(ticketRef)`.
  - `Action_send.go`'s `Decode` reads exactly this order: invType(byte),
    slot(u16), quantity(u16), mesos(u32), recipientName(string),
    quick(bool), and only when quick: message(string), ticketRef(u32).
    Matches the decompile field-for-field. Opcode confirmed
    `COutPacket::COutPacket(&v15/&v22, 65)` = 0x41, matching v83's
    registered opcode.
  - Also decompiled `0x6F0CA3`, IDA's own resolved name
    `CTabReceive::ReceiveParcel` (not a `sub_` stub): `Encode1(4)` [mode],
    `Encode4(parcelId)` — matches `ActionReceive` and confirms the
    provenance-comment fix (see 1(b)/yaml diff below) points at a real,
    correctly-named RECEIVE call site.
- `action_test.go`: read in full. All 7 subtests (`mode send`, `send npc`,
  `send quick`, `send npc trailing garbage`, `receive`, `discard`, `close`)
  assert concrete decoded field values and byte-exact `Reader.Available()`
  counts against hand-built wire byte slices (`sendNpcBytes`,
  `sendQuickBytes`, literal `[]byte{...}`) — not constructor-only checks.
  The "trailing garbage" subtest specifically proves the quick-only fields
  aren't read on the NPC path (`Available()==4` after `Decode`, i.e. the
  four appended bytes are provably unconsumed).
- `run.go` diff (`707e97efc..6bb465f86`): `git diff --stat` shows the four
  new `case` arms as a pure append (24 insertions, 0 deletions) — visually
  confirmed in the unified diff, every added line is a `+` line, no `-`
  lines present.
- `duey_action.yaml` diff (`707e97efc..6bb465f86`): comment-only — the
  RECEIVE citation changed from `CTabReceive::ReceiveParcel @0x65AF41`
  (a v72 address, confirmed by the file's own "gms_v72 ... RECEIVE=4
  @0x65af41" per-version line) to `@0x6F0CA3`. The `operations:` block
  (mode-value table) is byte-identical before/after — no mode value
  changed.

## Priority 3 — writer-absent notes

- `packet-audit operations --check` output (from the report, reproduced):
  8 "writer absent" notes for `DueyAction` across gms_v72/79/83/84/87/92/95
  and jms_v185, all reading `writer "DueyAction" not in template`.
- `tools/packet-audit/cmd/operations.go` — confirmed the `--templates-dir`
  default points at `services/atlas-configurations/seed-data/templates`,
  a path this task's file inventory (`libs/atlas-packet/parcel/**`,
  `tools/packet-audit/cmd/run.go`) never touches.
- `git diff --stat 707e97efc..586ee9ebb -- services/atlas-configurations`
  not separately re-run here, but the task's own commits (`git diff
  --stat` on both commits, read above) touch only `libs/atlas-packet`,
  `tools/packet-audit/cmd/run.go`, `docs/packets/dispatchers/duey_action.yaml`,
  and `docs/packets/audits/{STATUS.md,status.json}` — no seed-template file
  appears in either commit, so the note count is structurally unchanged by
  this task (nothing it touched could add or remove a template writer
  entry).
- **Verdict: explanation holds.** The notes are genuinely template-side
  (atlas-configurations seed data), unrelated to the codec files this task
  owns, and unchanged in count/cause by these two commits.

## Not checked

- Did not re-run `go run ./tools/packet-audit matrix --check`,
  `dispatcher-lint`, `fname-doc --check`, or `operations --check` myself —
  relied on the report's pasted transcripts plus my own independent
  mechanism verification (toolTreeSHA source read, diff-shape checks,
  decompile) per the task instruction not to re-run the regenerating form
  and that `matrix --check` green at HEAD was already confirmed by the
  dispatching agent.
- Did not decompile `sub_6F0DC3` (DISCARD) or `CParcelDlg::CloseParcelDlg`
  (`0x6F5691`) myself — both are simple `uint32 parcelId`-only and
  no-body arms respectively, structurally identical in shape to the
  RECEIVE arm I did verify byte-for-byte, and the brief's Priority 2 list
  singled out only the SEND asymmetry as needing decompile-level
  verification.
- Did not check gms_v72/v79/v84/v87/v92/v95 per-version decompiles behind
  `duey_action.yaml`'s per-version comment block — those addresses were
  established in Task 6 (already reviewed and closed per the controller's
  framing); only the RECEIVE citation Task 10 itself changed was
  re-verified.
- Did not run `tools/verify.sh` (explicitly out of scope, running
  separately).
- Did not check Ruling 5 / jms_v185 `parcel.yaml` notice keys or the
  clientbound `PARCEL` arms (explicitly out of scope, already closed).
