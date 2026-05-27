# NoteOperation (← `CWvsContext::OnMemoNotify_Receive`)

- **IDA:** 0x9f3830
- **Atlas file:** `libs/atlas-packet/note/serverbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op (=2, REQUEST) — byte sent as NOTE_ACTION sub-op by OnMemoNotify_Receive@0x9f3830` | ✅ |  |

ack: op-byte dispatcher — note/serverbound/operation.go Operation writes only the sub-op discriminator byte for NOTE_ACTION opcode 0x9A/154. Sub-ops audited individually via synthetic IDA entries; sub-op value space (SEND=0, DISCARD=1, REQUEST=2) deferred to Phase 2. See _pending.md OP-FAMILY-note.

