# Task 7 review — PARCEL struct and the OPEN / OPEN_QUICK arms

Commit reviewed: `797baebef` (range `91a6e82aa..797baebef`), branch
`task-241-duey-parcel-delivery`. Verified via `git branch --show-current` /
`git log -1` at review time — both matched.

IDA sessions used (all pre-existing, `idb_list`-confirmed): v83 `41f09cce`
(GMS_v83), v72 `f2a2e7c1` (GMS v72.1), v79 `f36df4cd` (GMS v79.1),
jms_v185 `05eb9c27` (JMS v185). v84/v87/v92/v95 were **not** independently
decompiled this review (see "Not checked" below).

## Priority 1 — the +29..233 message-span inference

**Verdict: honest and correctly labelled; the derivable facts are all
correct.**

Decompiled `PARCEL::Decode` @0x4E4345 (v83, `41f09cce`):

```
LODWORD(v4) = 234;                          /*0x4e4357*/
CInPacket::DecodeBuffer(a2, this, v4);      /*0x4e435d*/  <- single raw 234-byte copy
if ( CInPacket::Decode1(a2) )               /*0x4e4365*/  <- +234 hasItem bool
  GW_ItemSlotBase::Decode(v5, a2);          /*0x4e4378*/  <- +235.. optional item
```

This is a **fixed-size opaque memcpy**, not a field-by-field decode — so there
is genuinely no reader in `PARCEL::Decode` itself to pin the internal layout.

Cross-checked all downstream consumers of the decoded object:
- `CTabReceive::SetParcel` @0x6EF69C: `ZXString<char>::Format(&v42, m_pStr,
  v11 + 4)` for SP_3878 (name, confirms **+4**); `*(v33+17)` gated read +
  `*(v16+17)` as the format arg for SP_3879 (confirms **+17** mesos).
  `*(v33+238)` is read for the item-name notice, but 238 is an offset on the
  **C++ wrapper object**, not the wire buffer (parcel.go's comment already
  makes this distinction explicitly, correctly).
- `CTabReceive::ReceiveParcel` @0x6F0CA3 (v83): `**(*this + 8*idx + 4)` as the
  outbound RECEIVE request id at 0x6F0D66 (`COutPacket::Encode4`) — confirms
  **+0** id. Also `*(*(*this + 8*idx + 4) + 21) - v18` at 0x6F0D11, fed into a
  `<30`-day check (`sub_A62970`) — this independently confirms **+21 uint64
  sentAt** directly in v83 itself, stronger than the parcel.go comment's own
  citation (which only names v72's `ReceiveParcel` for this offset — v83
  corroborates it too, so no issue, just an available strengthening).
- The discard/removal flow at 0x6F0DC3 (an **unnamed** `sub_6F0DC3`, not the
  differently-named `CTabReceive::RemoveParcel`/`CParcelDlg::RemoveParcel`)
  encodes opcode 5 with `**(*this + 8*idx + 4)` as the id — confirms **+0**
  again for DISCARD. Minor: parcel.go's comment attributes this to
  "RemoveParcel" by name, but the function at that exact address is unnamed
  (`sub_6F0DC3`); the address and the +0 semantics are correct, only the
  function-name label carried over from the brief is loose. Non-blocking.
- `search_structs("PARCEL")` in the v83 IDB returns **no declared struct
  type** — there is no IDA-side field layout to consult either. This directly
  substantiates the "no consumer, no type" claim rather than just a failed
  grep.
- No `PARCEL::Encode` (or any writer-side memcpy) exists anywhere in the v83
  binary (`func_query` for `*PARCEL*` lists only `Decode` plus unrelated UI
  methods) — so there is no fixed-width copy on an encode path either that
  could pin the span. The client never re-serializes a full `PARCEL`; it only
  sends discrete id/opcode requests. This closes off the second avenue the
  review brief asked me to check (writer-side memcpy width) — there genuinely
  is none to check.

Arithmetic: total 234, confirmed leading fields consume 4+13+4+8=29, so the
205-byte remainder is a hard constraint, not a guess. The chosen encoding
(single zero-padded ASCII buffer) is consistent with every constraint the
binary imposes (total size, neighbouring `senderName[13]`'s own fixed-raw-ASCII
style) and is explicitly flagged in-code as a design-level inference rather
than asserted as verified. The business rule (100-char message cap) is
consistent with a 205-byte buffer (100 ASCII chars + null easily fits; no
contradiction).

**Conclusion: this is the "honest, constraint-consistent, clearly-labelled
inference" the brief says is acceptable, not a silent guess.** Not blocking.

## Priority 2 — the deliberate deviation (no `packet-audit:verify` markers)

**Verdict: correct call, correctly recorded.**

Confirmed today, at HEAD:
- `grep -rn "packet-audit:verify" libs/atlas-packet/parcel/` → no matches.
- `grep -rn "packet-audit:fname" libs/atlas-packet/parcel/` → exactly the two
  markers the brief asked for (`Open`, `OpenQuick`), each a plain doc comment,
  not a `:verify` marker.
- `go run ./tools/packet-audit dispatcher-lint` → `dispatcher-lint: clean`.
- Both test functions carry an explicit "No `packet-audit:verify` markers
  yet" doc comment with the reason, and the report's "Follow-up needed"
  section names the exact 16 (struct×version) cells still owed
  (`Open`×8, `OpenQuick`×8) and the mode bytes already known for each,
  including jms_v185 for both keys — nothing is left half-marked or silently
  dropped for Task 28 to trip over. The `DISPATCHER_FAMILY.md` step split
  (struct authoring vs. per-mode verification as a separate playbook) is a
  real property of the docs, not a rationalization — resolving a
  `packet-audit:verify` marker requires either an audit report or an
  evidence yaml, and both require a real per-version IDA export/hash chain
  that does not exist yet for this family. Fabricating either would violate
  the repo's evidence rules. This was the right call, and the deviation was
  flagged rather than silently taken. Not blocking.

## Priority 3 — the ordinary contract

- Modes resolved via `WithResolvedCode("operations", ParcelOperationOpen /
  ParcelOperationOpenQuick, …)` in both body functions
  (`parcel_body.go:24,32`) — no hard-coded mode byte anywhere in
  `clientbound/parcel.go` or `parcel_body.go`
  (`grep 'mode:\s*0x'` / `grep 'func(_ byte)'` both empty, reconfirmed).
- `MajorAtLeast` idiom: not applicable — Task 7 introduces no version-gated
  behavior (all 8 versions share one wire shape per Task 6's header
  derivation); no raw `> N` version comparison appears anywhere in the diff.
- No existing verified version's wire changed — `parcel` is a brand-new
  family with zero prior verified cells (`dispatcher-lint-baseline.yaml` has
  no `parcel` entry, confirmed by grep).
- `tools/packet-audit/cmd/run.go`: diff confirmed **purely additive** — the
  two new `case` arms are inserted immediately ahead of the pre-existing
  `NOTE_ACTION` comment/case block; nothing else in the 1300+-line switch was
  touched, reordered, or renumbered (`git show 797baebef -- tools/packet-audit/cmd/run.go`
  shows a clean `+14` hunk, no `-` lines).
- `clientbound/parcel.go` / `parcel_body.go` are additive-shaped: one
  `Open`/`OpenQuick` struct pair per file plus a two-key const block with an
  explicit "Task 8 and Task 9 append the remaining 19 keys to this same const
  block" comment — matches the brief's contract for later tasks.
- Byte-fixture tests genuinely assert wire bytes, not round-trips: expected
  byte slices are hand-assembled (`0x07,0x00,0x00,0x00`, literal `name`/`msg`
  buffers, `filetime[:]`, literal `0x08,0x01,0x00,0x00` for the OPEN-empty
  case, etc.) and compared against `p.Encode(...)`/`Open.Encode(...)` output
  — not `Decode(Encode(x)) == x`. `go test ./parcel/...` passes.

**Mode-byte cross-check against the client** (beyond trusting `parcel.yaml`):
decompiled `CParcelDlg::OnPacket` at the cited address in four of the eight
IDBs and confirmed the mode-8/mode-26(0x1A) shapes match `parcel.yaml` and
the Go `Open`/`OpenQuick` encode order in every case:
- **v83** `@0x6F56EA`: `case 8`: `Decode1` bool (quickEnabled) →
  `CParcelDlg::CParcelDlg(v25, v24?2:0)` → `SetParcelDlg`→`SetParcel`
  (mailbox count+N×PARCEL, arrived count+M×PARCEL). `case 26`:
  `CParcelDlg::CParcelDlg(v9, 1)`, no further packet reads. Matches
  `Open.Encode`/`OpenQuick.Encode` byte-for-byte in field order.
- **v72** `@0x65F93A`: identical shape at case 8 / case 26 (only the ZAlloc
  size constant and callee names differ, both consistent with mode 8=OPEN,
  26=OPEN_QUICK).
- **v79** `@0x683709`: identical shape at case 8 / case 26 — this is one of
  the four columns the task flagged as not independently re-derived by
  Task 6's audit; no mismatch found, so no re-derivation was triggered.
- **jms_v185** `@0x755A21`: `case 10` (not 8) does the same
  bool-then-`SetParcelDlg` shape — matches `parcel.yaml`'s jms_v185 OPEN=10.
  `case 27` (not 26) does the same bare-`CParcelDlg::CParcelDlg(v9,1)`
  shape — matches jms_v185 OPEN_QUICK=27 (0x1B is the packet header, 27 is
  the in-dispatcher mode byte after the leading `Decode1`).

No mode-byte mismatch was found anywhere checked, so per the review's
instruction, v84/v87/v92/v95 (already independently covered by Task 6's own
audit) were not re-derived.

## Blocking finding: `matrix --check` does not exit 0 at HEAD

`go run ./tools/packet-audit matrix --check` at `797baebef` (verified via
`git branch --show-current`/`git log -1`, clean working tree apart from this
review's own new files):

```
matrix --check: docs/packets/audits/STATUS.md is stale — regenerate and commit
matrix --check: docs/packets/audits/status.json is stale
exit status 1
```

Root cause, isolated: `STATUS.md`/`status.json` embed a self-hash of the
`packet-audit` tool ("Tool: `<hash>`"). `git diff --stat 91a6e82aa..797baebef -- tools/packet-audit/`
shows the **only** tool-source change in this commit range is Task 7's own
14-line `run.go` addition (the two new `case` arms) — i.e. Task 7's own edit
changed the tool's hash, and the commit did not regenerate+commit
`STATUS.md`/`status.json` alongside it. Running `go run ./tools/packet-audit
matrix` (no `-check`) regenerates both files with **only the `Tool:` hash
line changing** — confirming this is a mechanical staleness from Task 7's own
`run.go` edit, not a deeper coverage/orphan/drift problem (no parcel rows,
counts, or conflict numbers changed).

This directly contradicts the report's Step 6 transcript, which shows
`matrix --check` exiting 0 — the report's gate run must have preceded the
final `run.go` edit, or the files were reverted after. Either way, the
committed state fails the task's explicit hard gate today. Per the review
brief: *"the `matrix --check` hard gate (must exit 0 ...)"* — this is
blocking. Fix is mechanical: `go run ./tools/packet-audit matrix` then commit
the two regenerated files (no code change needed).

## Not checked

- v84, v87, v92, v95 `CParcelDlg::OnPacket` mode bytes were not independently
  decompiled this review (Task 6's own IDA audit already covers these four;
  the review brief scoped independent re-derivation to v72/v79/v84/v95 only
  on a mismatch, and none was found in the versions sampled).
- `jms_v185`'s `OPEN`/`OPEN_QUICK` FILETIME/id/mesos field offsets inside
  `PARCEL::Decode` were not independently re-decompiled (the struct itself is
  asserted byte-identical across versions per Task 6's header derivation;
  only the dispatcher's mode-byte shape was spot-checked for jms_v185 here).
- `go vet`/`go test -race` were run only for `libs/atlas-packet/parcel/...`,
  not the full module or `tools/packet-audit`, since the report's own
  transcript already showed both clean and nothing in this review touched
  either package's other code.
