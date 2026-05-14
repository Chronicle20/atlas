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
| Same enum bug confirmed in v83 & v87 IDA — value 16 silent-succeeds on those versions too | `2771f3bd7` |

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
| `DeleteCharacterResponse.NEXON_ID_DIFFERENT_THEN_REGISTERED: 16` → `26` (v95) | v95 case 26 has Notice(0xFD4); value 16 falls through to success path | `68d24f97c` |
| Same fix applied to v83 & v87 templates | v83 IDA (md5 80ff438c…) and v87 IDA (md5 2e692f3a…) both confirm case 26 → Notice(SP_4011); value 16 silent-succeeds | `2771f3bd7` |

All other sub-op enums (AuthLoginFailed/PinOperation/PinUpdate/ServerIP/CharacterNameResponse/AddCharacterEntry/CharacterViewAll) verified value-by-value against v95 IDA switches — no other changes needed.

## Cross-version login-domain audit (v83 / v87 / v95 / JMS v185)

After the primary v95 audit shipped, the same workflow was repeated against three additional client binaries the user has access to. Findings:

### GMS v83 (`MapleStory_dump.exe` md5 `80ff438ced539b831f0d2ed95099275d`)

**Verified via IDA:** dispatch table identical to v87 (`0x00-0x0F` main results, `0x16` sub_633496, `0x17` OnEnableSPWResult, `0x1A` OnLatestConnectedWorld, `0x1B` OnRecommendWorldMessage). HP/MP int16. PIC opcodes 0x1D/0x1E. VAC PIC opcodes 0x1F/0x20. SendDeleteCharPacket opcode 0x17. ServerStatusRequest reads int16 (same as v95 — atlas was widened to apply to all GMS in commit `d6593b257`). AuthPermanentBan trailing bytes also wasted on v83 (atlas widened in `13a2891ce`). AllCharacterListRequest opcode 0xD body is empty (export corrected).

**Outcome:** no atlas-packet code changes needed beyond the all-GMS widenings already shipped. `template_gms_83_1.json` corrected for the NEXON_ID_DIFFERENT_THEN_REGISTERED 16→26 issue (`2771f3bd7`).

### GMS v87 (`GMSv87_4GB.exe` md5 `2e692f3ab5078e04138d264f8ea1e668`)

**Verified via IDA:** dispatch table **identical to v83**. Every wire-shape comparison matches v83: HP/MP int16, AuthResult-27 routes to dialog without trailing-byte decode, PIC opcodes 0x1D/0x1E, VAC PIC opcodes 0x1F/0x20, SendDeleteCharPacket opcode 0x17, balloon (x:int16, y:int16, msg:str) widths preserved, OnDeleteCharacterResult case 26 routes to Notice. atlas-packet's `MajorVersion() >= 95` gates already serve v87 correctly via the pre-v95 branches.

**Outcome:** no atlas-packet code changes needed. `template_gms_87_1.json` opcodes verified correct (SelectWorld 0x1A, ServerListRecommendations 0x1B, DeleteCharacterHandle 0x17, RegisterPic 0x1D, CharacterSelectedPic 0x1E, VAC PIC variants 0x1F/0x20). Same `NEXON_ID_DIFFERENT_THEN_REGISTERED 16→26` fix applied in `2771f3bd7`.

### JMS v185 (`MapleStory_dump_SCY.exe` md5 `af6652ff9b7c549341f35e3569d7564a`)

**Verified via IDA:** dispatch is entirely separate from GMS — sequential opcodes 0x00-0x07 for the main response results, 0x14 OnViewAllCharResult, 0x16 OnLatestConnectedWorld, 0x17 OnRecommendWorldMessage, 0x18 LoginAuth (JMS-only), 0x19 PIC follow-up. `template_jms_185_1.json` opcodes already aligned: `LoginHandle 0x01` (SendCheckPasswordPacket), `CharacterListWorldHandle 0x04` (SendLoginPacket/select-world), `CharacterSelectedHandle 0x06` (plain LoginOpt 2-3), `DeleteCharacterHandle 0x0D` (SendDeleteCharPacket), `RegisterPicHandle 0x13` (LoginOpt 0), `CharacterSelectedPicHandle 0x14` (LoginOpt 1), `SelectWorld 0x16`, `ServerListRecommendations 0x17`, `LoginAuth 0x18`.

`AuthSuccess` JMS branch in `libs/atlas-packet/login/clientbound/auth_success.go` produces byte-correct wire output: atlas's "GM-bool" and "admin-byte" positions map to v185's `nGradeCode` and `packedFlags` fields, but atlas always writes zeros so the wire is identical regardless of name. `GW_CharacterStat` and `AvatarLook` structures match GMS shape (24-byte pet locker, int16 HP/MP, equip-loop terminated by 0xFF).

**Outcome:** no atlas-packet code or template changes needed for JMS v185 wire compatibility.

**Out-of-scope concerns documented but not fixed in this task:**

- JMS uses 18 character slots per world (vs 15 in GMS); per-char struct stride is 586 bytes (vs 750 in GMS). atlas-login service-side encoder may need verification.
- JMS `LoginHandle` wire shape (2 strings + 16-byte buffer + int32 + 2 bytes) and `DeleteCharacterHandle` (int32-only, no PIC) live in atlas-login service code, not atlas-packet. Audit would need to descend into the service.
- JMS-specific result codes 32/64 → `Notice(StringPool[5099])` are not emitted by atlas; they're unused codes, not a wire bug.

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
