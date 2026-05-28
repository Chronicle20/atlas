# Pending IDA function exports

This list tracks IDA functions referenced by the login-domain audit matrix
(task-027) but NOT yet in `gms_v95.json`. Each row needs a future maintainer
run of `packet-audit export ...` (live IDA-MCP) or hand-derivation from a
focused spike doc to add the function's wire-layout.

## Resolved (now in gms_v95.json)

| FName | Atlas writer/handler | Verdict |
|---|---|---|
| `CLogin::OnCheckPasswordResult` (success) | AuthSuccess | ✅ (v95 field-7 width fix shipped) |
| `CLogin::OnCheckPasswordResult#AuthLoginFailed` (synthetic) | AuthLoginFailed | ✅ |
| `CLogin::OnCheckPasswordResult#AuthTemporaryBan` (synthetic) | AuthTemporaryBan | ✅ |
| `CLogin::OnCheckPasswordResult#AuthPermanentBan` (synthetic) | AuthPermanentBan | ✅ (v95 trailing-bytes fix shipped) |
| `CLogin::OnSetAccountResult` | SetAccountResult | ✅ |
| `CLogin::OnCheckPinCodeResult` | PinOperation | ✅ |
| `CLogin::OnUpdatePinCodeResult` | PinUpdate | ✅ |
| `CLogin::OnLatestConnectedWorld` | SelectWorld | ✅ |
| `CLogin::OnRecommendWorldMessage` | ServerListRecommendations | 🔍 (sub-struct loop) |
| `CLogin::OnSelectWorldResult` | CharacterList | 🔍 (sub-struct CharacterListEntry) |
| `CLogin::OnWorldInformation` | ServerListEntry | 🔍 (sub-struct ChannelLoad loop) |
| `CLogin::OnSelectCharacterResult` | ServerIP | ✅ |
| `CLogin::SendCheckPasswordPacket` | Request (LoginHandle) | ✅ |
| `CLogin::SendSelectCharPacket` | CharacterSelect | ✅ |
| `CLogin::SendCheckUserLimitPacket` | ServerStatusRequest | ✅ (v95 width fix shipped) |
| `CLogin::SendViewAllCharPacket` | AllCharacterListRequest | ✅ |
| `CLogin::OnAcceptLicense` | AcceptTos (account/serverbound) | ✅ |

**17 packets audited, 14 ✅ / 3 🔍 / 0 ❌.**

## Still pending — login domain

| FName / Symbol | Atlas writer/handler | Notes |
|---|---|---|
| `CLogin::OnViewAllCharResult` (0x5de120, size 0x521) | AllCharacterListPong | Medium-complex; involves CharacterListEntry sub-struct. Phase 2 (analyzer descent) needed for high-fidelity audit. |
| `CLogin::SendSelectCharPacketByVAC` (0x5d7550, size 0x669) | CharacterSelectWithPic / *Register? | VAC-variant of char select. Large function; needs careful branch analysis. |
| `CLogin::OnSelectCharacterByVACResult` (0x5de670, size 0x375) | PicResult? | VAC result packet. |
| `CLogin::OnDenyLicense` (0x5d45d0) | — | Client-side function; constructs an outbound deny packet. |
| `CLicenseDlg::OnButtonClicked` (0x5ff870) | (UI callback) | Drives OnAcceptLicense / OnDenyLicense; not directly a wire format. |
| `LoginAuth` (atlas writer) | — | Orphan: atlas writes `WriteAsciiString(screen)`. No IDA function found by direct search. May be a legacy v83 packet that v95 client no longer reads. |

## Out of scope for GMS v95 audit (cross-region or cross-version)

These atlas writers/handlers exist in the codebase but the GMS v95 client
doesn't exercise them. The audit pipeline correctly produces no report
because there's no v95 IDA function to compare against:

- `LoginAuth` (clientbound, writes 1 string) — **JMS v1.85 only**. Whether
  GMS ever produces it is unconfirmed. Not in the gms_95 template.
- `ServerLoad` (clientbound, writes 1 byte) — **GMS v12 (or earlier) only**.
  Not in the gms_95 template.
- `ServerSelect` (serverbound, reads 1 byte worldId) — **GMS v12 (or earlier)
  only**. v95 uses `WorldCharacterListRequest` instead. Not in the gms_95
  template; the `WorldSelectHandle` symbol is dead in v95.
- `PicResult` (clientbound, opcode 0x1C, writes 1 byte) — semantically tied
  to `CLogin::SendSelectCharPacket` (the PIC-register branch's reply).
  Opcode 0x1C is not handled by `CLogin::OnPacket` directly in v95; receipt
  is routed through a different state machine, so the audit pipeline's
  CLogin-based dispatch model can't reach it. Wire shape (1 byte) is
  trivial enough that a manual cross-check confirms ✅.

## Still pending — handlers without an IDA mapping

Atlas writers/handlers under `libs/atlas-packet/login/` whose corresponding IDA
function hasn't been identified yet. Each likely corresponds to a
`CLogin::Send*` outbound packet constructor or a `CLogin::On*` inbound result:

- `AfterLoginHandle` (opcode 0x09) — atlas decodes `byte pinMode, optional (byte opt2, string pin)`
- `RegisterPinHandle` (opcode 0x0A)
- `CheckPicHandle`, `RegisterPicHandle`, `CharacterSelectedPicHandle`, `CharacterListSelectHandle`, `CharacterListSelectWithPicHandle` (PIC family, opcodes 0x15–0x1E)
- `SetGenderHandle` (opcode 0x08) — likely `CLogin::SendSetGenderPacket`
- `WorldCharacterListRequest` (opcode 0x05) — likely `CLogin::SendSelectWorldPacket` or similar
- `ServerStatus` (clientbound) — likely sent by `CLogin::OnCheckUserLimit`?
- `ServerLoad` (clientbound)
- `ServerListEnd` (clientbound, opcode 0x0A end-of-list sentinel inside ServerListEntry) — already audited as part of ServerListEntry's dispatch byte
- `PicResult` (clientbound)

## Known false positives in current audit output

`CharacterList.md` (verdict ❌): the per-entry trailer reports a 1-byte
over-count from row 45 onward. Static analysis collects all conditional
branches' calls (viewAll byte + gm byte + world-rank-enabled byte = 3
bytes), but at runtime only 2 fire: either {viewAll=0, gm=0} → 2 bytes
total (gm path returns early) or {viewAll=0, rank-enabled=1} → 1+16 = 17
+1 = 18 bytes total. v95 reads 2 bytes (onFamily + hasRank) + optional 16
bytes — matches both runtime paths. The pipeline doesn't model
early-return blocks as exclusive, so the audit over-counts. Resolution
would require an analyzer extension that flags `return` statements inside
guarded blocks; deferred to a follow-up.

## Cosmetic / cross-version concerns (not v95-specific bugs)

- `ServerIP.codes.SERVER_UNDER_INSPECTION: 7` (template_gms_95_1.json) — in
  v95 IDA, value 7 in `OnSelectCharacterResult`'s v3 switch triggers
  `GotoTitle + Error(17)` which is the "already logged in" path, not
  server-inspection. The wire value 7 still produces the right behavior
  (kick to title), but the constant name is misleading. Renaming would
  require updating the Go constant in `services/atlas-login/atlas.com/login/socket/writer/server_ip.go`
  AND all version templates (v83/v87/v92/v95/v111/JMS) that share this
  key. Left as-is for now to avoid cross-version breakage.

## Sub-op enum / sub-struct deferrals — commerce domain (task-067)

### Show clientbound — per-tab item segmentation + spurious padding (storage)

`CTrunkDlg::OnPacket#Show` (case 0x16 → `SetTrunkDlg`@0x76a940 →
`SetGetItems`@0x76a390, v95 GMS_v95.0_U_DEVM 3c71fd88...). The v95 client read
order for the open-storage ("Show") packet is:

1. `Decode1` mode (22)
2. `Decode4` npcTemplateId
3. `Decode1` slotCount
4. `DecodeBuffer(8)` tab-flag bitmask
5. **conditional** `Decode4` meso — read **only if** `flag & 2`
6. **per-tab loop** over tab bits 4/8/16/32/64 (Equip/Use/Setup/Etc/Cash): for
   each set bit, `Decode1` count then `count × GW_ItemSlotBase::Decode`.

Atlas `storage/clientbound/show.go` `Show.Encode` instead writes:
`mode, npcId, slots, Long(flags), Int(meso) [UNCONDITIONAL], Short(0), byte(count),
[flat assets], Short(0), byte(0)`.

Two real divergences (verified ❌ in `Show.md` rows 5-9, v95):

- **Spurious padding** the v95 client never reads: a `WriteShort(0)` *before*
  the item count, and a trailing `WriteShort(0)` + `WriteByte(0)`.
- **Single flat count vs per-tab segmented counts.** Atlas is called with
  `StorageOperationShowBody` which hardcodes `flags = StorageFlagAll` (bits
  2|4|8|16|32|64 = 0x7E) and `toAllPacketAssets` (a flat, unsegmented list of
  every inventory type). With all 5 tab bits set the client expects **5
  separate count+items blocks**; atlas emits one. → wire desync that wedges the
  storage panel until logout (the durable hot-path failure mode).

**Deferred (not fixed here)** because: (a) the correct fix is a structural
rewrite of `Show.Encode` — bucket assets by `Asset.inventoryType()`, emit a
count byte per set tab bit, gate meso on `flag&2`, drop the three padding bytes
— not a width/order tweak; (b) `Show.Encode` is version-agnostic (no region
guards) and only the v95 client `SetGetItems` was readable in this session, so a
change risks the v83/v87/v92/v111/JMS185 clients which may read the trailing
padding differently; (c) the conservative-rule bar (a speculative hot-path
rewrite is worse than an honest deferral). Fix warrants its own task with a
live storage-panel round-trip test across all target versions.

Files: `libs/atlas-packet/storage/clientbound/show.go`.

### UpdateAssets clientbound — loop/sub-struct tool limitation (storage), correct in practice

`CTrunkDlg::OnPacket#UpdateAssets` (cases 9/13/15/19 → `SetGetItems`). Verdict
🔍 in `UpdateAssets.md` is a packet-audit **tool limitation** (the per-tab item
loop + `GW_ItemSlotBase`/`Asset` sub-struct can't be flattened row-for-row), NOT
a wire bug. Runtime callers (`services/atlas-channel/.../storage/consumer.go`)
always pass `flags = inventoryTypeToFlag(t)` (exactly **one** tab bit, currency
bit never set) and `toPacketAssets(t, …)` (assets filtered to that one type), so
the client's per-tab loop runs exactly once → reads one count + items, which
matches atlas's single-count emission byte-for-byte. No fix needed. (Contrast
with Show, which sets all bits and sends a flat list — that one is the real bug.)

### OP-FAMILY-storage — serverbound dispatcher op-byte (storage)

`storage/serverbound/operation.go` `Operation.Encode` writes only the op-byte
(`WriteByte(mode)`); the field payloads live in the sibling sub-bodies. The IDA
oracle is the family of `CTrunkDlg::Send*Request` senders, each of which
`COutPacket(…,67=0x43)` then `Encode1(op)` + fields:

| Atlas struct | op-byte | IDA sender | fields after op |
|---|---|---|---|
| (Operation dispatcher) | — | — | op byte only |
| OperationRetrieveAsset | 4 | `CTrunkDlg::SendGetItemRequest`@0x769e00 | `Encode1` invType, `Encode1` slot |
| OperationStoreAsset | 5 | `CTrunkDlg::SendPutItemRequest`@0x768570 | `Encode2` slot, `Encode4` itemId, `Encode2` qty |
| OperationMeso | 7 | `CTrunkDlg::SendGetMoneyRequest`@0x7688e0 / `SendPutMoneyRequest`@0x7689e0 | `Encode4` signed amount (+withdraw / −deposit) |

The dispatcher's op-byte is supplied by the caller (the resolved `operations`
code), so its lone-byte ✅ in `Operation.md` is expected. Sub-bodies audited
independently — all ✅ in v95. No fix needed; recorded here for traceability.

### Add clientbound — asset sub-struct tool limitation (inventory), correct in practice

`CWvsContext::OnInventoryOperation#Add` (dispatcher @0xa08a70, case 0 line 158 →
`GW_ItemSlotBase::Decode`, v95 GMS_v95.0_U_DEVM 3c71fd88...). Verdict 🔍 in
`Add.md` row 5 is a packet-audit **tool limitation**: the analyzer flattens
`Add.Encode`'s `WriteByteArray(m.asset.Encode(...))` into one opaque row and
cannot descend into the `model.Asset` / `GW_ItemSlotBase` sub-struct. NOT a wire
bug — the dispatcher case-0 read order (Decode1 mode, Decode1 invType, Decode2
slot, then the item via `GW_ItemSlotBase::Decode`) matches `Add.Encode`
(`mode 0, type, slot, asset.Encode`) field-for-field. The asset body is the
shared `model.Asset` encoder, audited independently of this packet. No fix
needed.

### ChangeBatch clientbound — loop + conditional trailing addMov tool limitation (inventory), correct in practice

`CWvsContext::OnInventoryOperation#ChangeBatch` (dispatcher @0xa08a70). Verdict ❌
in `ChangeBatch.md` row 3 is a packet-audit **false positive**, NOT a wire bug.
The dispatcher's per-entry body is a variable-length switch loop
(`case 0/1/2/3`, lines 148-411) that the analyzer collapses to a single opaque
buffer op (the Phase 0 `EncodeEntry` extension resolves the entry recursion, so
row 2 is ✅). After the loop the dispatcher reads **one** trailing addMov byte
(`Decode1` → `SetSecondaryStatChangedPoint`, line 315) iff `nCurItemPos` was set
— which happens when any entry is an equip-slot move (case 2:
`invType==1 && (oldSlot<0||newSlot<0)`, line 225) or remove (case 3:
`invType==1 && slot<0`, line 374). `ChangeBatch.Encode` writes its single
trailing `WriteInt8(addMov)` under exactly that condition (`addMov = max over
entries of EntryAddMov()`, emitted iff `> -1`). The ❌ arises only because the
conditional trailing byte (atlas row 3) has no flat IDA op to align against once
the loop is buffer-collapsed. Wire-correct in v95; no fix needed.

### move.go ↔ change.go:ChangeMove addMov symmetry — verified NOT shared (inventory)

The serverbound move request `CWvsContext::SendChangeSlotPositionRequest`@0x9d9c10
encodes exactly `Encode4 update_time, Encode1 nType, Encode2 nOldPos,
Encode2 nNewPos, Encode2 nCount` — **no addMov byte**. The clientbound echo
`change.go:ChangeMove` (dispatcher case 2) DOES carry the trailing addMov byte.
These are distinct packets (ITEM_MOVE opcode serverbound vs INVENTORY_OPERATION
clientbound) and correctly do NOT share the addMov field — the addMov byte is
exclusively a clientbound `OnInventoryOperation` artifact. `move.go` ✅ matches
its IDA sender byte-for-byte; `change.go:ChangeMove` ✅ matches the dispatcher
(including the conditional addMov). No mismatch, no fix needed; recorded for
traceability.

## Workflow notes

Refresh procedure:
1. `mcp__ida-pro__list_functions_filter` with a partial name to find the IDA FName (mangled symbols are common; use plain prefix like "SelectChar")
2. `mcp__ida-pro__get_function_by_name` (resolve address)
3. `mcp__ida-pro__decompile_function` (extract C source)
4. Parse the `CInPacket::DecodeN` / `COutPacket::EncodeN` call sequence in lexical order (success path only; multi-branch functions need manual filtering)
5. Add the entry to `gms_v95.json` and the `candidatesFromFName` map in `tools/packet-audit/cmd/run.go`
6. Regenerate audit: `cd tools/packet-audit && go run . --csv-clientbound ... --csv-serverbound ... --template ... --atlas-packet ../../libs/atlas-packet --ida-source ../../docs/packets/ida-exports/gms_v95.json --output ../../docs/packets/audits`

The synthetic-FName scheme (e.g., `CLogin::OnCheckPasswordResult#AuthLoginFailed`)
lets one IDA function model multiple sub-branches when atlas has separate
writers for different result codes.
