# Post-Phase-B Scoping Checkpoint — task-027

This document enumerates the remaining work after task-027 shipped on branch
`task-027-atlas-packet-v95-audit` (PR #438). Use it to scope follow-on
sibling tasks.

> **NOTE**: This doc was significantly expanded after initial creation as the
> task scope grew. The original 20-task plan focused on 6 spike packets +
> tooling; the shipped scope additionally covered the complete login-domain
> audit (28 packets), Phase 2 sub-struct analyzer descent, real balloon
> support, and template opcode/sub-op verification against v95 IDA.

## Login-domain audit — final state

**Coverage: 28 packets audited (27 ✅ / 1 ❌)** against `GMS_v95.0_U_DEVM.exe`.

### Clientbound (server → client) — 12

| Writer | Verdict | Notes |
|---|---|---|
| AuthSuccess | ✅ | field-7 width fix shipped (byte → int16 on v95+) |
| AuthLoginFailed | ✅ | failure-branch shape verified |
| AuthTemporaryBan | ✅ | |
| AuthPermanentBan | ✅ | trailing-bytes fix shipped (don't emit 9 unused bytes on v95) |
| SetAccountResult | ✅ | |
| PinOperation | ✅ | |
| PinUpdate | ✅ | |
| SelectWorld | ✅ | template opcode 0x1A → 0x18 fix shipped |
| ServerListEntry | ✅ | per-channel world-id bug fixed; real balloon support added |
| ServerListEnd | ✅ | synthetic FName for 0xFF terminator branch |
| ServerListRecommendations | ✅ | template opcode 0x1B → 0x19 fix shipped |
| ServerIP | ✅ | analyzer learned `WriteByteArray` |
| CharacterList | ❌ | **documented static-analyzer false positive** — runtime wire is correct; sub-struct trailer over-counts conditional branches with early returns |

### Serverbound (client → server) — 16

| Handler | Verdict | Notes |
|---|---|---|
| Request (LoginHandle) | ✅ | modified-v95 wire shape |
| ServerStatusRequest | ✅ | **width fix shipped** (byte → int16 on v95+; v95 client sends int16) |
| ServerListRequest | ✅ | empty body (ChangeStepImmediate opcode 4) |
| WorldCharacterListRequest | ✅ | matched against SendLoginPacket modified-client path |
| AcceptTos | ✅ | OnAcceptLicense outbound opcode 7 |
| AfterLogin | ✅ | byte+byte+string outbound opcode 9 |
| CharacterSelect | ✅ | no-PIC path; template handler opcode mapping verified |
| CharacterSelectWithPic | ✅ | template handler opcode 0x1E → 0x1D fix shipped |
| CharacterSelectRegisterPic | ✅ | template handler opcode 0x1D → 0x1C fix shipped |
| AllCharacterListRequest | ✅ | SendViewAllCharPacket |
| AllCharacterListSelect | ✅ | SendSelectCharPacketByVAC m_bLoginOpt==2/3 |
| AllCharacterListSelectWithPic | ✅ | template handler opcode 0x20 → 0x1F fix shipped |
| AllCharacterListSelectWithPicRegister | ✅ | template handler opcode 0x1F → 0x1E fix shipped |
| AllCharacterListPong | ✅ | MakeVACDlg / ResetVAC opcode 0x0F |
| DeleteCharacterHandle (registered) | — | template handler opcode 0x17 → 0x18 fix shipped (atlas-packet has no decoder type) |

### Verified out-of-scope (not exercised by v95 GMS client)

These atlas writers/handlers exist but don't apply to GMS v95:
- `LoginAuth` (JMS v1.85 only)
- `ServerLoad` (GMS v12 / legacy)
- `ServerSelect` (GMS v12 / legacy; v95 uses WorldCharacterListRequest)
- `PicResult` (opcode 0x1C routed outside `CLogin::OnPacket` login state)

See `docs/packets/ida-exports/_pending.md` for details.

## Real wire bugs fixed (4)

| Bug | Fix commit |
|---|---|
| `ServerStatusRequest` reads 1 byte but v95 client sends 2 (int16) | `d6593b257` |
| `AuthPermanentBan` emits 9 trailing bytes the v95 client never reads on resultCode 27 | `13a2891ce` |
| `GW_CharacterStat` HP/MaxHP/MP/MaxMP written as int16; v95 widened to int32 | `fe77a672a` |
| `DeleteCharacterResponse.NEXON_ID_DIFFERENT_THEN_REGISTERED` at value 16 (v95 uses 26; value 16 silent-succeeds) | `68d24f97c` |

## Template opcode/enum fixes (7 + 1 enum)

| Fix | Why | Commit |
|---|---|---|
| `SelectWorld` writer 0x1A → 0x18 | v95 `OnLatestConnectedWorld` is case 24 (was 26 in v83) | `01c8b7359` |
| `ServerListRecommendations` writer 0x1B → 0x19 | v95 `OnRecommendWorldMessage` is case 25 | `01c8b7359` |
| `DeleteCharacterHandle` handler 0x17 → 0x18 | v95 `SendDeleteCharPacket` emits opcode 24; 0x17 reserved for CharSale-variant CreateChar | `01c8b7359` |
| `RegisterPicHandle` 0x1D → 0x1C | v95 `SendSelectCharPacket` m_bLoginOpt==0 | `01c8b7359` |
| `CharacterSelectedPicHandle` 0x1E → 0x1D | m_bLoginOpt==1 | `01c8b7359` |
| `CharacterViewAllSelectedPicRegisterHandle` 0x1F → 0x1E | VAC m_bLoginOpt==0 | `01c8b7359` |
| `CharacterViewAllSelectedPicHandle` 0x20 → 0x1F | VAC m_bLoginOpt==1 | `01c8b7359` |
| `DeleteCharacterResponse.NEXON_ID_DIFFERENT_THEN_REGISTERED: 16` → `26` | v95 case 26 has Notice(0xFD4); value 16 falls through to success path | `68d24f97c` |

All other sub-op enums (AuthLoginFailed/PinOperation/PinUpdate/ServerIP/CharacterNameResponse/AddCharacterEntry/CharacterViewAll) verified value-by-value against v95 IDA switches — no other changes needed.

## Tooling improvements (general-purpose)

Reusable for any future GMS/JMS version audit:

- **Audit pipeline** (`tools/packet-audit/`): Go CLI with AST analyzer, IDA-export source, diff engine, markdown+JSON report writer, SUMMARY aggregator
- **TypeRegistry**: scans `libs/atlas-packet/**` for struct types with Encode/Write methods, pre-analyzes their bodies for sub-struct descent
- **Diff Flatten** inlines `KindRepeat` bodies and `KindRecurse` markers using the registry
- **Trailing-loop downgrade**: IDA loop-body calls (guard `"loop X"`) where atlas runs short get `⚠️` instead of `❌` (wire-correct for zero-iteration loops)
- **Analyzer recognizes**: `WriteByteArray(c.Encode(...)(opts))` (wrapped recurse), `WritePaddedString` / `ReadPaddedString` (free-function helpers), `WriteKeyValue` (compound byte+int32), `WriteInt8/16/32/64` (size-explicit aliases)
- **Guard parser** with conjoin that synthesizes a working `Eval` even when text-based reparse fails on `<unparsed:...>` markers (preserves outer-guard restrictions for nested branches)
- **Synthetic FName scheme** (`CLogin::OnX#AtlasWriterName`) lets one IDA function model multiple sub-branches that map to distinct atlas writers/handlers — covers PIC dispatch, OnCheckPasswordResult auth-fail variants, etc.
- **Real balloon support** added: `model.WorldBalloon` type threaded through `ServerListEntry` encoder/decoder/test

## Remaining work

### Highest-leverage follow-ups

1. **Channel-domain audit** — apply the same workflow to channel/clientbound and channel/serverbound packets. Each domain (character/monster/drop/inventory/field/pet/reactor/quest/party/guild/buddy/chat/messenger/note/merchant/interaction/fame/storage/cashshop/ui/socket) is roughly the size of the login-domain audit shipped here. One sibling task per domain pair (clientbound + serverbound).

2. **CharacterList ❌ false-positive resolution** — extend the analyzer to model `return` statements inside guarded blocks as exclusive (when one branch returns, sibling branches' calls shouldn't both contribute to the static flat list). Would flip CharacterList ❌ → ✅.

3. **Cosmetic: rename `SERVER_UNDER_INSPECTION` → `ALREADY_LOGGED_IN`** across all 6 version templates + the Go constant in `services/atlas-login/atlas.com/login/socket/writer/server_ip.go`. Wire-equivalent; pure clarity.

### Lower-leverage

4. **Real MCP `export` subcommand** — currently `packet-audit export ...` is a stub. Refreshes go through Claude-with-MCP. Building it out would let maintainers run one CLI command. Worthwhile if export refreshes become frequent.

5. **Bare-handler audit gaps** — some template handlers (SetGenderHandle, RegisterPinHandle, CharacterCheckNameHandle, CreateCharacterHandle, ClientStartHandle) don't have atlas-packet decoder types — they're handled inline by atlas-login service code. Audit would need to descend into the service code to verify wire shapes.

## Path forward

Recommended sequencing:
1. Land this PR (#438) — login domain is comprehensively verified
2. Spec the highest-impact channel-domain sub-task (character or monster — character spawn is shared with login's CharacterStat/AvatarLook so the registry is already warm)
3. Repeat per domain

The infrastructure built in this task makes each subsequent domain audit roughly the same effort as login: catalog atlas writers/handlers, populate the IDA export, run the pipeline, triage findings.
