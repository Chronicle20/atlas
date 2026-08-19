# Review: Task 7 fix round (commit 29ec83a32)

**Verdict: APPROVED**

## Scope reviewed

Commit `29ec83a32223022ea7b2ee76e8e05493915899a65` (actual short sha `29ec83a32`),
three claims:

1. `docs/packets/audits/STATUS.md` / `status.json` regeneration (tool self-hash only).
2. `libs/atlas-packet/parcel/parcel.go`: `0x6F0DC3` renamed from "RemoveParcel" to
   "the unnamed sub_6F0DC3".
3. `libs/atlas-packet/parcel/parcel.go`: new v83 corroboration for `+21 uint64 sentAt`
   citing `CTabReceive::ReceiveParcel @0x6F0D11`.

All three derived independently against live IDA (v83 IDB, session `41f09cce`,
`E:\Programs\Nexon\IDBs_v9\GMS\v83_Me\MapleStory_dump.exe.i64`) before reading the
task's fix report or prior review, then reconciled.

## Item 1 — STATUS.md / status.json regeneration

Not re-verified per instructions (already independently confirmed by the
requester: `matrix --check` exits 0, one-line diff each). I did re-read the
commit diff as a cheap sanity check: `git show 29ec83a32 -- docs/packets/audits/STATUS.md docs/packets/audits/status.json`
shows exactly one changed line per file, both the `Tool:` / `toolSha` hash
(`319a7ad9…` → `9d9dd920…`), no coverage/orphan/drift/conflict line moved.
Consistent with the claim. **No IDA content to verify here — clean.**

## Item 2 — `0x6F0DC3` is genuinely unnamed

`mcp__ida-pro__lookup_funcs` on `0x6F0DC3` in the v83 IDB (session `41f09cce`)
returns:

```
{"addr":"0x6f0dc3","name":"sub_6F0DC3","size":"0xce"}
```

No demangled/named symbol — `sub_` prefix is IDA's auto-generated name for an
unnamed function, confirming there is no real name ("RemoveParcel" or otherwise)
attached to this address in the v83 binary. Disassembly of the function start
(`mcp__ida-pro__disasm @0x6F0DC3`) shows the label is literally `sub_6F0DC3` with
no alternate symbol.

Decompile of the function confirms the discard-flow shape the comment describes:
bounds-check against the parcel-list count/capacity, a `CUtilDlg::YesNo`
confirmation prompt (`SP_3889_THE_ITEMS_AND_OR_MESOS_INCLUDED_IN_THE_PACKAGE…`),
then on confirm:

```
COutPacket::COutPacket(&v10, 65);          // opcode 0x41 = 65
COutPacket::Encode1(&v10, 5u);             // DUEY_ACTION::DISCARD (5)
COutPacket::Encode4(&v10, **(*this + 8 * *(this + 1) + 4));  // parcel+0, parcelId
```

`**(*this + 8*idx + 4)` double-dereferences to the parcel struct pointer, then
reads its first 4 bytes — i.e. `*(parcel+0)`, matching the `+0 uint32 parcelId`
row this comment sits under. **Verdict: confirmed correct in both directions —
the name is genuinely absent, and the code is genuinely a DISCARD-with-parcelId
flow at offset +0.**

## Item 3 — v83 `0x6F0D11` corroboration for `+21 uint64 sentAt`

`mcp__ida-pro__lookup_funcs` on `0x6F0D11` resolves to the enclosing function:

```
{"addr":"0x6f0ca3","name":"?ReceiveParcel@CTabReceive@@QAEXXZ","size":"0x120"}
```

i.e. `CTabReceive::ReceiveParcel`, matching the citation exactly (function name
and the fact that `0x6F0D11` falls inside its body, not some other routine).

Decompile shows the relevant computation:

```c
v5 = *(*(*this + 8 * *(this + 1) + 4) + 21) - v18;      /*0x6f0d11*/
if ( sub_A62970(v5, HIDWORD(v5), 711573504, 201) < 30 )  /*0x6f0d2e*/
```

— a 64-bit subtraction of a FILETIME-like value read from `parcel+21` against
the current time, then compared against a 30-day threshold via `sub_A62970`,
i.e. exactly the "<30-day check" the comment and the prior v72 citation
describe.

Disassembly at `0x6F0CA3` pins down the offset and width precisely:

```
6f0d0a  mov eax, [ecx+eax*8+4]      ; eax = parcel struct pointer
6f0d0e  mov ecx, [eax+15h]          ; 0x15 = 21 decimal -> low dword of parcel+21
6f0d11  sub ecx, [ebp+var_18]       ; <-- cited address: low-dword subtract
6f0d14  mov eax, [eax+19h]          ; 0x19 = 25 decimal -> high dword (parcel+21+4)
6f0d17  sbb eax, [ebp+var_18+4]     ; high-dword subtract-with-borrow
```

This is a textbook 64-bit (QWORD) subtraction: low dword at `parcel+0x15` (=21),
high dword at `parcel+0x19` (=25, i.e. 21+4), combined via `sub`/`sbb`. That
confirms all three specifics the task required:

- **Offset**: +21 (0x15) exactly — not +17, not +25.
- **Width**: 8 bytes (a genuine QWORD split across two 32-bit halves), not 4.
- **Enclosing function**: `CTabReceive::ReceiveParcel`, not some other tab
  handler.
- **Cited address**: `0x6F0D11` is precisely the low-dword `sub` instruction of
  that 64-bit subtraction — not an approximate "somewhere in this region" address.

**Verdict: confirmed correct on every axis the task called out as a risk
(offset, width, function).**

## What I did NOT check

- Item 1's mechanical claim beyond the one-line diff re-read (no independent
  `matrix --check` rerun — the requester already did this and I was told not
  to repeat it).
- The `+29..233` message-span inference (explicitly out of scope, already
  closed).
- jms_v185 notice-arm values (explicitly out of scope).
- `packet-audit:verify` markers (assigned to Task 28, out of scope).
- Anything the concurrent Task 8 implementer is touching
  (`libs/atlas-packet/parcel/clientbound/*`, `tools/packet-audit/cmd/run.go`).
- The v72 half of the `+21` citation (`ReceiveParcel @0x65AF41` region) — that
  citation was pre-existing (not part of this commit's diff) and the task
  scoped my check to the new v83 corroboration only.
- I did not open/modify any file beyond reading via IDA MCP tools and `git
  show`/`git diff` — no codec, registry, template, evidence, or generated file
  was touched.

## Bottom line

Both provenance comments hold up against live IDA on the exact binary
(`MapleStory_dump.exe.i64`, GMS v83) the citations claim. No fabrication, no
wrong-binary or wrong-offset drift. Item 1's regeneration is a clean one-line
hash bump consistent with the stated cause (stale tool self-hash after Task 7's
`run.go` edit). Nothing here is BLOCKING.
