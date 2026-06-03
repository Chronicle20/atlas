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

## Tool domain — utility-only (task-069)

`libs/atlas-packet/tool/` contains only `uint128.go` — a 128-bit unsigned
integer utility type (ShiftLeft/ShiftRight/And/Or/Xor/Add/Mult/IsZero) consumed
by socket/channel handshake encoders for hash fields. It is NOT a packet domain:
zero `Operation()`/`Encode()`/`Decode()` methods, zero audit rows. Confirmed at
audit time via `find libs/atlas-packet/tool -name '*.go' ! -name '*_test.go'`
(single file) and method enumeration. Listed in TOTAL.md §2 under "no packets;
utility-only".

## locateAtlasFile struct-name collisions (task-069)

The audit's `locateAtlasFile` (tools/packet-audit/cmd/run.go) resolves an atlas
struct by the FIRST `type <Name> struct` match in alphabetical `WalkDir` order
within the matching direction folder. When two domains define the same struct
name in the same direction, the wrong file is audited:

| Struct | Audited (wrong) | Intended | Effect |
|---|---|---|---|
| `ChannelChange` (clientbound) | `buddy/clientbound/channel_change.go` | `channel/clientbound/change.go` | spurious ❌ on the ChannelChange row; the channel packet is verified correct manually + by wire-shape test (see ChannelChange.md Manual analysis) |

Not fixed here (tool change out of scope per design §1's spirit). Future misc
buckets must check for same-name collisions and verify the audited file path in
SUMMARY points at the intended domain; if not, verify the packet manually and
annotate the report.

## Bare handlers — misc domain (task-069)

| Handler constant | Location | Reason deferred |
|---|---|---|
| `HiredMerchantOperationHandle` | `libs/atlas-packet/merchant/serverbound/operation.go` | Bare constant only — no atlas-packet decoder struct. The serverbound parse is handled in `services/atlas-channel` socket handler. Out of scope for the libs/atlas-packet audit. |

## Missing / unverified modes — merchant (task-069, sub-phase 2f)

| Mode | Constant | Reason deferred |
|---|---|---|
| 8 (0x08) | `HiredMerchantOperationModeErrorUnknown` | In IDA switch (`OnEntrustedShopCheckResult`): `Decode4(shopId)+Decode1(channelId)` — channel-name notice. No atlas body emits it; missing implementation, not a struct wire bug. |
| 1 (0x01) | `HiredMerchantOperationModeErrorUnableToOpenTheStore` | Absent from the v95 `OnEntrustedShopCheckResult` switch. Possibly hire-merchant (task-067), KMS-only, or client-side-only. Cross-reference task-067 before implementing. |
| 11 (0x0B) | *(no atlas constant)* | Present in IDA switch (string-pool 3508 notice, no extra decode) but no atlas constant/struct. Add when the mode is exercised. |

## Still pending — quest (task-069, sub-phase 2g)

These three serverbound quest packets have a confirmed v95 wire mismatch but the fix
requires updating BOTH the atlas-packet struct AND the `services/atlas-channel`
`quest_action.go` handler together — broader than the libs-only audit. Recommend a
dedicated follow-up task.

- **`ActionStart` / `ActionComplete`** (`CQuest::StartQuest` @0x6b40a0, actions 1/2):
  v95 writes `Encode4(nItemPos)` (delivery-item slot, 0 for normal quests) BETWEEN `npcId`
  and the conditional `x,y` coords; atlas omits it. Also: atlas's `autoStart` gate on `x,y`
  corresponds to IDA's `!CQuestMan::IsAutoAlertQuest(questId)` — equivalent but verify naming.
- **`ActionRestoreLostItem`** (`CQuest::OnCompleteQuestFailed` @0x6b1fc0, action 0):
  IDA sends a count-prefixed variable-length array of lost-item ids
  (`Encode1(0)+Encode2(questId)+Encode4(count)+EncodeBuffer(4*count)`); atlas models a single
  `unk1+itemId`. Needs struct redesign (slice + count) + handler update. Rarely exercised.

## Phase 3 cross-version verification TODOs (task-069)

Gates applied during the v95 (Phase 2) pass that were conservatively scoped to v95+ and
MUST be re-checked against v83 / v87 / JMS185 IDA in Phase 3 (widen or narrow as evidence
dictates):

- **`stat/clientbound/changed.go`** — second trailing flag byte (battle-recovery-info) gated
  to `GMS && MajorVersion>=95`. Confirm whether v83/v87/JMS clients also read two trailing
  flags. (HP/MaxHP/MP/MaxMP int32 widening is already version-evidenced via
  `model/character_statistics.go` and need not be re-derived.)
- **`ui/clientbound/lock.go`** — the int32 `tAfterLeaveDirectionMode` field is gated
  `GMS && MajorVersion>=90` (pre-existing). v83/non-GMS clients may send only the 1-byte flag;
  confirm whether those versions also read the int32, else the gate is correct as-is.
