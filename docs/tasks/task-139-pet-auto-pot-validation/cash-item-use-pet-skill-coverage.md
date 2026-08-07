# cash/serverbound/CashItemUsePetSkill — GMS coverage sweep (task-139 final-review follow-up)

Resolves the open question left in `coverage-manifest.yaml`: does any GMS version's
`CWvsContext::SendConsumeCashItemUseRequest` type-28 arm carry the pet-skill-pouch
payload? All eight GMS versions were IDA-swept read-only this pass (no `rename`,
`set_comments`, or `idb_save`); `jms_v185` was left untouched (already `verified`).

## Per-version evidence table

| Version | Function @ addr | Case-28 dispatch | What it does | Decision |
|---|---|---|---|---|
| gms_v48 | `SendConsumeCashItemUseRequest` @0x70e495 (session 0bb5f11a) | Unique arm @0x7103aa (not in default list `26,29,32,33,35,44-46`; `xrefs_to` confirms its sole code xref is the switch dispatch @0x70e53c) | **Corrected this pass** (see note below) — case 28's own body is: `sub_71370B()` (bare `dword_80D398 != 0` global-flag check) → either a resource-0x91 string + jump to the shared 13-way cleanup @0x711c47 (no encode), or `get_field()`+`sub_711EEB` (bare `this[65]` `CField` member read, not pet-related) → a resource-0x10E notice + shared exit (no encode) OR a `CUtilDlgEx` confirm-dialog build that, on exactly one path, calls `COutPacket::EncodeStr` **once** @0x710571 (a formatted confirm value, most likely a price/description string) before jumping to the case-34-shared tail. Neither guard helper references a pet object. | **n-a** — no petId/locker-SN encode, not pet-related |
| gms_v61 | `SendConsumeCashItemUseRequest` @0x832a5d (session 965202bf) | Unique arm @0x834a4e (not in default list `27,34,35,37,38,40,42-44`) | 5 instructions total: calls `sub_832700` (checks a global flag then `field+252` against `[910000000,910000022]`, an event-map-id band; unrelated helper `sub_715A69` on match, chat-log otherwise) then jumps straight to the shared default cleanup. **Zero** `Encode*` calls. | **n-a** — zero payload, unrelated (map-gate) guard logic |
| gms_v72 | `SendConsumeCashItemUseRequest` @0x904fe2 (session 90e36cb0) | **No arm** — compiler comment at 0x905083 (`ja def_905089`) and 13 more jump-into-default sites lists `default case, cases 28,35,36,38,39,41,43-45,54-59,62` | n/a | **n-a** — case absent |
| gms_v79 | `SendConsumeCashItemUseRequest` @0x95634a (session 9a7d3642) | **No arm** — compiler comment at 0x9563eb and 13 more sites lists `default case, cases 28,35,36,38,39,41,43-45,54-59,62,67` | n/a | **n-a** — case absent |
| gms_v83 | `SendConsumeCashItemUseRequest` @0xa0a63f (session ce4ff298) | **No arm** — compiler comment at 0xa0a6e0 (re-derives final-review's original finding, same address) lists `default case, cases 28,35,36,38,39,41,43-45,54-59,62,67` | n/a | **n-a** — case absent |
| gms_v84 | `SendConsumeCashItemUseRequest` @0xa54a2f (session 79511a2a) | **No arm** — compiler comment at 0xa54ad0 lists `default case, cases 28,35,36,38,39,41,43-45,54-59,62,67-69` | n/a | **n-a** — case absent |
| gms_v87 | `SendConsumeCashItemUseRequest` @0xa9fef9 (session 81f32170) | **No arm** — compiler comment at 0xa9ffa8 (re-derives final-review's original finding, same address) lists `default case, cases 28,35,36,38,39,41,43-45,54-59,62,67-69` | n/a | **n-a** — case absent |
| gms_v95 | `SendConsumeCashItemUseRequest` @0x9eb3e0 (session e4abcb98) | Unique arm `$LN2090_1` @0x9f06be, dispatched via a byte-indirected jump table (range check @0x9eb4fd) | `lea this,[var_208]` (a `ZArray<unsigned char>` immediately after the `oPacket` slot — the packet's own byte buffer) → `ZArray<unsigned char>::RemoveAll()` @0x9f06c9 (discards the already-encoded common header) → `ZXString<char>::~ZXString<char>()` @0x9f06e0 → `jmp loc_9EB68E` @0x9f06e5, which is the function's own epilogue (SEH restore + pops + `retn 0x10`). No `Encode*` call anywhere on this path; no `SendPacket` reachable. | **n-a** (recommended; see below) — real arm, but sends nothing at all |
| jms_v185 | `SendConsumeCashItemUseRequest` @0xaef2f5 (session 3c4bb8b1) | Unique arm @0xaf16df | After pet lookup/selection-dialog prep, `EncodeBuffer(pet+0x18, 8)` @0xaf1a42 — the pet locker SN | **verified** (unchanged, pre-existing) |

Every GMS-version claim above was independently re-derived from IDA this pass,
including v83/v87/v95 (previously only noted in prose by the final review) — none
were taken on faith.

## Correction: gms_v48's original evidence attributed the wrong case's code

The first pass of this sweep committed (`75acfbfcc`) misattributed case-37's payload
to case 28 on gms_v48. The `Encode1(flag)`@0x710852 + six-`EncodeStr` block
(@0x710869/0x710880/0x710897/0x7108ae/0x7108c5/0x7108dc) is **case 37's** body (label
`loc_71059E` @0x71059e, its own independent switch-dispatch target — `xrefs_to`
confirms its sole code xref is the same switch jump @0x70e53c that dispatches case 28,
and the two are non-overlapping ranges: case 28 ends at the unconditional
`jmp loc_711D60` @0x710599, five bytes before case 37's entry at 0x71059e). Re-tracing
case 28 from `0x7103aa` end-to-end (both exit paths: the resource-0x91 early-out via the
shared cleanup @0x711c47 confirmed by `xrefs_to` to have 13 distinct callers across the
function, and the `sub_711EEB`-gated notice/confirm-dialog path) found its real, exclusive
body contains **exactly one** `COutPacket::EncodeStr` call @0x710571 and no other `Encode*`
call — not the megaphone-tier shape originally claimed. The two guard-helper
characterizations (`sub_71370B` = bare `dword_80D398 != 0`; `sub_711EEB` = bare `this[65]`
read) were correct in the original entry and are unchanged. **The disposition does not
change** — case 28 still contains no petId/locker-SN payload and no pet reference, so
`n-a` remains the correct grading — but the evidence text in both this report and
`docs/packets/audits/gms_v48/_unimplemented.json` has been rewritten to describe only
what case 28 actually contains. No other version's entry was touched.

## Recommendation on the v95 zero-payload arm

Recorded **n-a**, not left `incomplete`. Two facts drove this, both confirmed by
re-decompiling the arm and its immediate exit target:

1. The arm's `ZArray<unsigned char>::RemoveAll()` call operates on the stack slot
   sitting directly against the packet object — the client discards whatever was
   already encoded (the common header: opcode + `Encode2(nPOS)` + `Encode4(nItemID)`)
   before falling into the function epilogue. This is not "empty payload, header still
   sent" — it is "nothing is sent on the wire at all" for item-type 28 on this build.
2. There is no petId, locker-SN, or any other field this arm could ever be pinned to a
   byte fixture for — a future `packet-verifier` pass would have nothing to verify.
   Leaving it `incomplete` implies there's a codec still to be written; there isn't
   one, the client doesn't emit one.

Counter-consideration for a future reader to weigh: unlike v72/v79/v83/v84/v87, v95
*does* have a case value allocated in the jump table for type 28 (reached via a real,
uniquely-targeted label, not folded into `default`). If a later client build wired a
real send onto that same slot, this would be the first version where the compiler had
already reserved room for it. That's a structural signal worth remembering if a v92
correlate is ever checked, but it doesn't change what v95 itself sends today: nothing.

### The v92 correlate — checked, and it resolves the question

gms_v92 was outside this sweep's original eight versions (it was a declared exclusion
until merging main routed a `PetItemUseHandle` into its template). It has since been
checked directly, and it answers the paragraph above.

**v92 patterns with v95 structurally, and with everyone behaviourally.** In
`CWvsContext::SendConsumeCashItemUseRequest` @`0x9bfe10`, type 28 IS uniquely
dispatched: it is absent from the compiler's default-case list
(`35,36,38,39,41,43,45,54-59,62,67-69`), and resolving it through both tables —
index `28-12=16` → `byte_9C4ECC[16]=0x0e` → `jpt_9BFF39[14]=0x9c4dfe` — lands on its own
label. (Table arithmetic confirmed rather than assumed: `jpt_9BFF39[0x27]` = `0x9c00b5`,
exactly the `def_9BFF39` label the disassembly names.) So v92 *also* has room reserved.

But the arm emits nothing. Its full body is seven instructions: the `COutPacket`
destructor (`sub_42B500`, 0x1f bytes) on the same stack slot the constructor used, the
`ZXString` destructor (`sub_403900`, 0x11 bytes), then `jmp loc_9C00E7` — the function's
own epilogue (`mov large fs:0, ecx`; register pops; `add esp, 220h`; `retn 10h`). No
`Encode*` call, and neither `SendPacket` (`0x42a7a0`) nor `CClientSocket::SendPacket`
(`0x4ac120`) is reachable from it. It is a behavioural clone of the default case at
`0x9c00b5`, which runs the same destructor into the same epilogue.

So the "reserved slot" signal spans **two** consecutive versions (v92 and v95) without
either ever wiring a send. That weakens rather than strengthens the reading that the slot
was being held for this feature: two builds allocated a case value for type 28 and both
discard the packet. Recorded n-a in `docs/packets/audits/gms_v92/_unimplemented.json`;
the matrix row is now n-a across all nine GMS versions with jms_v185 the sole `✅`.

## Mechanism note (diverges from the task's literal instruction)

The task instruction pointed at `docs/packets/feature-na-evidence.yaml` (the
`USE_TELEPORT_ROCK × gms_v48` mechanism) as "the existing mechanism" for recording
absence. Verified directly from source
(`tools/packet-audit/internal/matrix/build.go` `gradeSubStructCell`,
`tools/packet-audit/cmd/na_consistency.go` `naConsistencyCheck`) that
`feature-na-evidence.yaml` only participates in the **RowOp** family-consistency gate
(`rowsByOp` filters `r.Kind != matrix.RowOp`) — `cash/serverbound/CashItemUsePetSkill`
is a `kind: sub-struct` row (`RowSubStruct`), which is graded exclusively from
`in.Unimplemented[vk][pkt]`, populated from each version's
`docs/packets/audits/<version>/_unimplemented.json`. Adding entries to
`feature-na-evidence.yaml` for this packet would have been silently inert — the
matrix cells would have stayed `incomplete`. All eight n-a dispositions were recorded
in the correct file (`_unimplemented.json`, `packet: cash/serverbound/CashItemUsePetSkill`
+ `fname: CWvsContext::SendConsumeCashItemUseRequest` + a `reason` citing the evidence
above), which is what actually flips `StateNA` in the regenerated matrix. `gms_v84` had
no `_unimplemented.json` file before this pass; one was created.

## Verification

```
packet-audit matrix          # exit 0 — regenerated STATUS.md/status.json, all 8 GMS cells n-a, jms_v185 unchanged (verified)
packet-audit matrix --check  # exit 0
packet-audit fname-doc --check   # exit 0
packet-audit operations --check  # exit 0
```

No file under `tools/packet-audit` was modified this pass, so the matrix's
`toolSha`-at-HEAD ordering trap does not apply — the matrix can be (and was)
regenerated before or after committing the `_unimplemented.json`/manifest changes
without needing a second regeneration pass.
