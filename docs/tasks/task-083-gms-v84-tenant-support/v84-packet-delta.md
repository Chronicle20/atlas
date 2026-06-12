# v83 → v84 Packet / Opcode / Version-Branch Delta

Source of truth for task-083 (FR-1.4). Every code/template change cites a row here.

## 0. IDB inventory & dispatch-table anchors

All five IDA-MCP instances reachable (confirmed `list_instances` 2026-06-09): v83 GMS (13337), v84 GMS (13341 — the hard requirement), v95 GMS (13339), v87 GMS (13338), JMS v185 (13340, out of scope). v84 IDB is loaded and analyzable.

Dispatch architecture (identical across all three GMS IDBs): the client recv path is a two-stage dispatch. `CClientSocket::ProcessPacket` reads the opcode via `CInPacket::Decode2`, handles socket-level opcodes (0x10–0x14, 0x19) inline, routes a low/high band to the active CStage vtable, and forwards the bulk band to `CWvsContext::OnPacket`, which is the large server→client handler switch (these map to Atlas WRITERS). The send path (client→server, Atlas HANDLERS) is NOT a switch: it is a distributed set of call sites that construct a `COutPacket` with the send-opcode (`COutPacket::COutPacket(long)`) and emit through the single sink `CClientSocket::SendPacket`. The "outbound dispatch table" therefore = the `SendPacket` sink; individual send opcodes are recovered by enumerating xrefs to `COutPacket::COutPacket` / `SendPacket` (deferred to later tasks). The addresses below give both the recv-switch anchor (`CWvsContext::OnPacket`) and the recv entry (`ProcessPacket`), plus the send sink and the `COutPacket(long)` ctor.

| IDB | port | dispatch table (inbound) addr | dispatch table (outbound) addr | naming density |
|---|---|---|---|---|
| v83 GMS | 13337 | `CWvsContext::OnPacket` @ `0xA07A08` (recv switch, 0x1D–0x7C; recv entry `CClientSocket::ProcessPacket` @ `0x4965F1`) | send sink `CClientSocket::SendPacket` @ `0x49637B` (per-site opcode via `COutPacket::COutPacket(long)` @ `0x6EC9CE`) | dense |
| v84 GMS | 13341 | `CWvsContext::OnPacket` @ `0xA51CD0` (recv switch, 0x1D–0x7F; recv entry `CClientSocket::ProcessPacket` @ `0x49B502`) | send sink `CClientSocket::SendPacket` @ `0x49B28C` (per-site opcode via `COutPacket::COutPacket(long)` @ `0x703CFA`) | partial |
| v95 GMS | 13339 | `CWvsContext::OnPacket` @ `0x9E5830` (recv switch, 0x1D–0x8C; recv entry `CClientSocket::ProcessPacket` @ `0x4B00F0`) | send sink `CClientSocket::SendPacket` @ `0x4AF9F0` (per-site opcode via `COutPacket::COutPacket(long)` @ `0x68D090`) | dense |

### Confirmation method
- **v83 (anchor):** `ProcessPacket` decompiled — confirmed `Decode2`→`switch`, default band forwards to `CWvsContext::OnPacket`. `OnPacket` decompiled — 80+ `case` switch, every target a named `CWvsContext::On*` handler. Send sink/ctor resolved by mangled symbol and verified against multiple `SendPacket` call-site comments.
- **v84 (mandatory):** structurally identical to v83. `ProcessPacket` @ `0x49B502` retains its mangled symbol; decompile confirms the same `Decode2`→`switch`→`CWvsContext::OnPacket` shape, with the forward band widened to **0x1D–0x7F** (v83 was 0x1D–0x7C). `CWvsContext::OnPacket` @ `0xA51CD0` decompiled — 90+ `case` switch spanning 0x1D–0x7F. Send sink and `COutPacket(long)` ctor resolved by mangled symbol and cross-checked against `SendPacket` call-site comments.
- **v95 (tie-breaker):** all four anchors resolved by mangled symbol; `OnPacket` @ `0x9E5830` decompiled — 140-case switch (0x1D–0x8C), all targets named `CWvsContext::On*`.

### OQ-7 evidence — low-confidence v84 anchors (unnamed `sub_XXXX`)
The two v84 dispatch *frames* (`CClientSocket::ProcessPacket`, `CWvsContext::OnPacket`) are reliably named, so the anchor addresses above are high-confidence. **The risk is one level down:** inside v84's `CWvsContext::OnPacket` switch, *every* per-opcode handler target is an unnamed `sub_XXXX` — none carry the `CWvsContext::On*` symbol that v83/v95 have. Examples: opcode 0x1D → `sub_A69D8F` (v83 `OnInventoryOperation`), 0x1F → `sub_A6AE08` (v83 `OnStatChanged`), 0x3D → `sub_A6EDA8` (v83 `OnCharacterInfo`), 0x44 → `sub_A8592D` (v83 `OnBroadcastMsg`). The socket-level handlers reached from `ProcessPacket` are likewise unnamed (`sub_49B616`/0x10, `sub_49B5D5`/0x11, `sub_49B70D`/0x12, `sub_49B865`/0x13, `sub_49B8BB`/0x19). Later tasks that need to confirm a *specific* v84 opcode's packet shape must decompile the corresponding `sub_XXXX` and align it positionally against the v83 (dense) named handler — they cannot trust a v84 symbol because there is none. Additionally, the opcode-band ceiling differs per version (v83 0x7C, v84 0x7F, v95 0x8C), so v95's richer naming is a *naming* Rosetta only; opcode numbering must not be assumed 1:1 between v84 and v95.

## 1. Inbound (handler) opcode map  (FR-1.1, FR-1.3)

**Scope & method.** "Inbound" = Atlas handlers = client→server send path. The
authoritative in-scope set is every handler in the v83 seed template
`services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
(`socket.handlers`): **93 opcode entries = 93 distinct opcodes / 92 distinct
handler-name strings** (opcode `0x04` and `0x0B` are two distinct opcodes that
both map to the one `ServerListRequestHandle`). Each handler-name string is the
Atlas logical name; every name resolves to a `const ...Handle = "..."` in
`libs/atlas-packet/*/serverbound/` (verified by grep — e.g. `LoginHandle` →
`login/serverbound/request.go:13`, `CharacterMoveHandle` →
`character/serverbound/move.go:14`, `MapChangeHandle` →
`field/serverbound/change.go:13`, `ChannelChangeHandle` →
`channel/serverbound/channel_change.go:13`).

**Send opcode = the `long` immediate to `COutPacket::COutPacket(long)` at each
client construction site.** The Atlas template opcode IS that immediate (proven:
v83 `CLogin::SendCheckPasswordPacket` @ `0x5F6952` builds `COutPacket(&v, 1)` —
matches LoginHandle `0x01`; the nearby `0xC9` is the NMCO/passport `LoginAuth`
arg, not a packet op).

**Key finding — the client send-opcode enum is STABLE v83→v84.** Five opcodes
spanning the in-scope flows (login, channel, map, chat, party) were byte-read
from the binaries; all five are identical v83↔v84:

| opcode | v83 fn / immediate | v84 fn / immediate | flow |
|---|---|---|---|
| `0x01` LoginHandle | `?SendCheckPasswordPacket@CLogin@@` @ `0x5F6952`, `COutPacket(…,1)` | same symbol @ `0x60B88B`, `COutPacket(…,1)` | login (named in BOTH) |
| `0x26` MapChangeHandle | `?SendTransferFieldRequest@CField@@` @ `0x53035D`, `COutPacket(…,0x26)` | CField send cluster (positional; size/addr aligned) | map load |
| `0x27` ChannelChangeHandle | `?SendTransferChannelRequest@CField@@` @ `0x5304AF`, `COutPacket(…,0x27)` | CField send cluster (positional) | channel change |
| `0x31` CharacterChatGeneralHandle | `?SendChatMsg@CField@@` @ `0x52C315` (size 0xF7) | `sub_5382D7` (size 0xF7, EncodeStr+Encode1), `COutPacket(…,49)` = `0x31` | chat |
| `0x7D` PartyInviteRejectHandle | party-invite send | `sub_53BC2A`, `COutPacket(…,125)` = `0x7D` | party |

This stability is fully consistent with Section 5: v84 is a *minor* GMS bump
(84.1). The recv-side handler switch (`CWvsContext::OnPacket`, clientbound) only
*widens at the top* (band v83 `0x1D–0x7C` → v84 `0x1D–0x7F`, i.e. +3 new
server→client cases) with no renumbering; the send-opcode enum is untouched. The
v84 delta is in packet *structure* (the `MajorVersion() > 83` predicates in
Section 5), **not opcode numbering.** The v84 image also retains many class
symbols at addresses aligned with v83 (`SendCheckPasswordPacket`,
`SendCreateNewPartyMsg`, `SendJoinPartyMsg`, `OnConnect`, `OnSelectWorldResult`),
and `xrefs_to` the v84 `SendPacket` sink @ `0x49B28C` reproduces the v83 send-site
topology one-for-one.

**Classification.** All 93 in-scope handler opcodes → **SAME** (v84 opcode == v83
opcode, same handler). No SHIFTED / ADDED / REMOVED found within the in-scope
serverbound flows (the recv-band +3 is clientbound and belongs to Section 2).
Confidence is graded per row:

- **high** = opcode byte-read in the v84 binary (or v84 retains the named
  function): `0x01`, `0x31`, `0x7D`.
- **med** = v83 byte-read + v84 positional/structural anchor (size+addr+encode
  shape aligned) and/or enum-stability inference: `0x26`, `0x27`.
- **med (enum-stable, OQ-7)** = not independently re-read in v84; classified SAME
  by the proven send-enum stability + the unnamed-`sub_XXXX` caveat. This covers
  the remaining rows. None is a guessed hex value — each carries the v83 template
  opcode, which IS the wire opcode, and the only claim being inferred is "v84 did
  not renumber it," which the five verified anchors and the architecture support.

| logical name | v83 opcode | v84 opcode | classification | evidence (v83 fn/addr; v84 fn/addr; confidence) |
|---|---|---|---|---|
| LoginHandle | 0x01 | 0x01 | SAME | v83 `?SendCheckPasswordPacket@CLogin@@`@0x5F6952 `COutPacket(1)`; v84 same symbol@0x60B88B `COutPacket(1)`; **high** |
| ServerListRequestHandle | 0x04 | 0x04 | SAME | template (also bound at 0x0B); v83 login-stage send; v84 enum-stable; **med (OQ-7)** |
| CharacterListWorldHandle | 0x05 | 0x05 | SAME | template; login-stage; enum-stable; **med (OQ-7)** |
| ServerStatusHandle | 0x06 | 0x06 | SAME | template; login-stage; enum-stable; **med (OQ-7)** |
| AcceptTosHandle | 0x07 | 0x07 | SAME | template; login-stage; enum-stable; **med (OQ-7)** |
| SetGenderHandle | 0x08 | 0x08 | SAME | template; login-stage; enum-stable; **med (OQ-7)** |
| AfterLoginHandle | 0x09 | 0x09 | SAME | template; login-stage; enum-stable; **med (OQ-7)** |
| RegisterPinHandle | 0x0A | 0x0A | SAME | template; PIN flow; enum-stable; **med (OQ-7)** |
| ServerListRequestHandle | 0x0B | 0x0B | SAME | template (dup of 0x04 handler); enum-stable; **med (OQ-7)** |
| CharacterViewAllHandle | 0x0D | 0x0D | SAME | template; v83 `?SendViewAllCharPacket@CLogin@@`@0x5FAC34 region; enum-stable; **med (OQ-7)** |
| CharacterViewAllSelectedHandle | 0x0E | 0x0E | SAME | template; view-all flow; enum-stable; **med (OQ-7)** |
| CharacterViewAllPongHandle | 0x0F | 0x0F | SAME | template; view-all flow; enum-stable; **med (OQ-7)** |
| CharacterSelectedHandle | 0x13 | 0x13 | SAME | template; v83 `?SendSelectCharPacket@CLogin@@`@0x5F726D region; enum-stable; **med (OQ-7)** |
| CharacterLoggedInHandle | 0x14 | 0x14 | SAME | template; migrate-in; enum-stable; **med (OQ-7)** |
| CharacterCheckNameHandle | 0x15 | 0x15 | SAME | template; char-create flow; enum-stable; **med (OQ-7)** |
| CreateCharacterHandle | 0x16 | 0x16 | SAME | template; v83 `?SendNewCharPacket@CLogin@@`@0x5F7E7A; enum-stable; **med (OQ-7)** |
| DeleteCharacterHandle | 0x17 | 0x17 | SAME | template; v83 `?SendDeleteCharPacket@CLogin@@`@0x5F7C4A; enum-stable; **med (OQ-7)** |
| PongHandle | 0x18 | 0x18 | SAME | template; keepalive; enum-stable; **med (OQ-7)** |
| StartErrorHandle | 0x19 | 0x19 | SAME | template; client-start error; enum-stable; **med (OQ-7)** |
| RegisterPicHandle | 0x1D | 0x1D | SAME | template; PIC flow; enum-stable; **med (OQ-7)** |
| CharacterSelectedPicHandle | 0x1E | 0x1E | SAME | template; PIC select; enum-stable; **med (OQ-7)** |
| CharacterViewAllSelectedPicRegisterHandle | 0x1F | 0x1F | SAME | template; PIC register; enum-stable; **med (OQ-7)** |
| CharacterViewAllSelectedPicHandle | 0x20 | 0x20 | SAME | template; PIC view-all; enum-stable; **med (OQ-7)** |
| ClientStartHandle | 0x23 | 0x23 | SAME | template; client start; enum-stable; **med (OQ-7)** |
| NoOpHandler | 0x24 | 0x24 | SAME | template (no-op slot); enum-stable; **med (OQ-7)** |
| MapChangeHandle | 0x26 | 0x26 | SAME | v83 `?SendTransferFieldRequest@CField@@`@0x53035D `COutPacket(0x26)`; v84 CField send cluster (positional/size-aligned); **med** |
| ChannelChangeHandle | 0x27 | 0x27 | SAME | v83 `?SendTransferChannelRequest@CField@@`@0x5304AF `COutPacket(0x27)`; v84 CField send cluster (positional); **med** |
| CashShopEntryHandle | 0x28 | 0x28 | SAME | template; cash-shop entry; enum-stable; **med (OQ-7)** |
| CharacterMoveHandle | 0x29 | 0x29 | SAME | template; `character/serverbound/move.go`; v83 CUserLocal move send; v84 not re-read (move opcode not a simple immediate); enum-stable; **med (OQ-7)** |
| CharacterChairInteractionHandle | 0x2A | 0x2A | SAME | template; chair; enum-stable; **med (OQ-7)** |
| CharacterChairPortableHandle | 0x2B | 0x2B | SAME | template; portable chair; enum-stable; **med (OQ-7)** |
| CharacterMeleeAttackHandle | 0x2C | 0x2C | SAME | template; melee attack; enum-stable; **med (OQ-7)** |
| CharacterRangedAttackHandle | 0x2D | 0x2D | SAME | template; ranged attack; enum-stable; **med (OQ-7)** |
| CharacterMagicAttackHandle | 0x2E | 0x2E | SAME | template; magic attack; enum-stable; **med (OQ-7)** |
| CharacterTouchAttackHandle | 0x2F | 0x2F | SAME | template; touch/body attack; enum-stable; **med (OQ-7)** |
| CharacterDamageHandle | 0x30 | 0x30 | SAME | template; take-damage; enum-stable; **med (OQ-7)** |
| CharacterChatGeneralHandle | 0x31 | 0x31 | SAME | v83 `?SendChatMsg@CField@@`@0x52C315 (size 0xF7); v84 `sub_5382D7` (size 0xF7) `COutPacket(49)`=0x31, EncodeStr+Encode1; **high** |
| ChalkboardCloseHandle | 0x32 | 0x32 | SAME | template; chalkboard close; enum-stable; **med (OQ-7)** |
| CharacterExpressionHandle | 0x33 | 0x33 | SAME | template; emote; enum-stable; **med (OQ-7)** |
| MonsterBookCover | 0x39 | 0x39 | SAME | template; monster-book cover; enum-stable; **med (OQ-7)** |
| NPCStartConversationHandle | 0x3A | 0x3A | SAME | template; NPC select/start; enum-stable; **med (OQ-7)** |
| NPCContinueConversationHandle | 0x3C | 0x3C | SAME | template; NPC continue; enum-stable; **med (OQ-7)** |
| NPCShopHandle | 0x3D | 0x3D | SAME | template; NPC shop op; enum-stable; **med (OQ-7)** |
| StorageOperationHandle | 0x3E | 0x3E | SAME | template; storage op; enum-stable; **med (OQ-7)** |
| HiredMerchantOperationHandle | 0x3F | 0x3F | SAME | template; hired-merchant op; enum-stable; **med (OQ-7)** |
| CompartmentMergeHandle | 0x45 | 0x45 | SAME | template; inventory gather/merge; enum-stable; **med (OQ-7)** |
| CompartmentSortHandle | 0x46 | 0x46 | SAME | template; inventory sort; enum-stable; **med (OQ-7)** |
| CharacterInventoryMoveHandle | 0x47 | 0x47 | SAME | template; inv-item move; enum-stable; **med (OQ-7)** |
| CharacterItemUseHandle | 0x48 | 0x48 | SAME | template; use item; enum-stable; **med (OQ-7)** |
| CharacterItemCancelHandle | 0x49 | 0x49 | SAME | template; cancel item; enum-stable; **med (OQ-7)** |
| CharacterItemUseSummonBagHandle | 0x4B | 0x4B | SAME | template; summon bag; enum-stable; **med (OQ-7)** |
| PetFoodHandle | 0x4C | 0x4C | SAME | template; pet food; enum-stable; **med (OQ-7)** |
| CharacterCashItemUseHandle | 0x4F | 0x4F | SAME | template; cash-item use; enum-stable; **med (OQ-7)** |
| CharacterItemUseTownScrollHandle | 0x55 | 0x55 | SAME | template; return/town scroll; enum-stable; **med (OQ-7)** |
| CharacterItemUseScrollHandle | 0x56 | 0x56 | SAME | template; upgrade scroll; enum-stable; **med (OQ-7)** |
| CharacterDistributeApHandle | 0x57 | 0x57 | SAME | template; AP distribute; enum-stable; **med (OQ-7)** |
| CharacterAutoDistributeApHandle | 0x58 | 0x58 | SAME | template; auto-AP; enum-stable; **med (OQ-7)** |
| CharacterHealOverTimeHandle | 0x59 | 0x59 | SAME | template; heal-over-time; enum-stable; **med (OQ-7)** |
| CharacterDistributeSpHandle | 0x5A | 0x5A | SAME | template; SP distribute; enum-stable; **med (OQ-7)** |
| CharacterUseSkillHandle | 0x5B | 0x5B | SAME | template; use skill; enum-stable; **med (OQ-7)** |
| CharacterBuffCancel | 0x5C | 0x5C | SAME | template; cancel buff; enum-stable; **med (OQ-7)** |
| CharacterDropMesoHandle | 0x5E | 0x5E | SAME | template; drop meso; enum-stable; **med (OQ-7)** |
| FameChangeHandle | 0x5F | 0x5F | SAME | template; give fame; enum-stable; **med (OQ-7)** |
| CharacterInfoRequestHandle | 0x61 | 0x61 | SAME | template; char-info request; enum-stable; **med (OQ-7)** |
| PetSpawnHandle | 0x62 | 0x62 | SAME | template; pet activate; enum-stable; **med (OQ-7)** |
| PortalScriptHandle | 0x64 | 0x64 | SAME | template; portal script; enum-stable; **med (OQ-7)** |
| QuestActionHandle | 0x6B | 0x6B | SAME | template; quest action; enum-stable; **med (OQ-7)** |
| CharacterSkillMacroHandle | 0x6E | 0x6E | SAME | template; skill macro; enum-stable; **med (OQ-7)** |
| CharacterMultiChatHandle | 0x77 | 0x77 | SAME | template; party/buddy/guild chat; enum-stable; **med (OQ-7)** |
| CharacterChatWhisperHandle | 0x78 | 0x78 | SAME | template; v83 `?SendChatMsgWhisper@CField@@`@0x52F185 (size 0x841); v84 `sub_53B2DB` (size 0x841, positional); enum-stable; **med (OQ-7)** |
| MessengerOperationHandle | 0x7A | 0x7A | SAME | template; messenger op; enum-stable; **med (OQ-7)** |
| CharacterInteractionHandle | 0x7B | 0x7B | SAME | template; trade/mini-room op; enum-stable; **med (OQ-7)** |
| PartyOperationHandle | 0x7C | 0x7C | SAME | template; v84 party send cluster around `sub_53BC2A`; enum-stable; **med (OQ-7)** |
| PartyInviteRejectHandle | 0x7D | 0x7D | SAME | v84 `sub_53BC2A` `COutPacket(125)`=0x7D (party-invite reject/decline shape); **high** |
| GuildOperationHandle | 0x7E | 0x7E | SAME | template; guild op; enum-stable; **med (OQ-7)** |
| GuildInviteRejectHandle | 0x7F | 0x7F | SAME | template; guild invite reject; enum-stable; **med (OQ-7)** |
| BuddyOperationHandle | 0x82 | 0x82 | SAME | template; buddy op; enum-stable; **med (OQ-7)** |
| NoteOperationHandle | 0x83 | 0x83 | SAME | template; note/memo op; enum-stable; **med (OQ-7)** |
| CharacterKeyMapChangeHandle | 0x87 | 0x87 | SAME | template; key-map change; enum-stable; **med (OQ-7)** |
| GuildBBSHandle | 0x9B | 0x9B | SAME | template; guild BBS; enum-stable; **med (OQ-7)** |
| PetMovementHandle | 0xA7 | 0xA7 | SAME | template; pet move; enum-stable; **med (OQ-7)** |
| PetChatHandle | 0xA8 | 0xA8 | SAME | template; pet chat; enum-stable; **med (OQ-7)** |
| PetCommandHandle | 0xA9 | 0xA9 | SAME | template; pet command; enum-stable; **med (OQ-7)** |
| PetDropPickUpHandle | 0xAA | 0xAA | SAME | template; pet loot; enum-stable; **med (OQ-7)** |
| PetItemUseHandle | 0xAB | 0xAB | SAME | template; pet item use; enum-stable; **med (OQ-7)** |
| PetItemExcludeHandle | 0xAC | 0xAC | SAME | template; pet item ignore-list; enum-stable; **med (OQ-7)** |
| MonsterMovementHandle | 0xBC | 0xBC | SAME | template; mob move; enum-stable; **med (OQ-7)** |
| MonsterDamageFriendlyHandle | 0xC0 | 0xC0 | SAME | template; friendly-mob damage; enum-stable; **med (OQ-7)** |
| NPCActionHandle | 0xC5 | 0xC5 | SAME | template; NPC move/action; enum-stable; **med (OQ-7)** |
| DropPickUpHandle | 0xCA | 0xCA | SAME | template; drop pickup; enum-stable; **med (OQ-7)** |
| ReactorHitHandle | 0xCD | 0xCD | SAME | template; reactor hit; enum-stable; **med (OQ-7)** |
| CashShopCheckWalletHandle | 0xE4 | **0xEA** | **SHIFTED (+6)** | CORRECTED post-audit: v83 `CCashShop::TrySendQueryCashRequest` `COutPacket(0xE4)` body-less; v84 `CCashShop__TrySendQueryCashRequest_send_0xEA` `COutPacket(0xEA)` body-less (this[6] guard). v83-value 0xE4 in v84 = `CWvsContext::SendPartyWanted`. **high (decompiled)** |
| CashShopOperationHandle | 0xE5 | **0xEB** | **SHIFTED (+6)** | CORRECTED post-audit: v83 `CCashShop::OnBuy` `COutPacket(0xE5)`+`Encode1(3=BUY)`+`bbiii`; v84 `CCashShop__OnBuy_send_0xEB_op3` `COutPacket(0xEB)`+`Encode1(3)`+identical `bbiii`. 0xEB carries the full op-type set {3,4,5,6,7,8,9,13,14,26,29,30,31,33,35,40,46,49}. v83-value 0xE5 in v84 = `CWvsContext::SendCancelPartyWanted` (body-less → atlas read op=0 → "Unhandled Cash Shop Operation [0]"). **high (decompiled)** |

> **AUDIT-TABLE CAVEAT (post-corrective-audit):** rows above reading `SAME … med (OQ-7)`
> are the *original A2 harvest*, which assumed CP-enum stability WITHOUT decompiling each
> sender — that assumption was wrong for the cash-shop pair (and others). The authoritative
> inbound opcodes live in `template_gms_84_1.json`, not this table. Decompile-confirmed
> corrections since A2: pet band (0xA7→0xAC…0xAC→0xB1), MonsterMovement 0xBC→0xC1,
> MonsterDamageFriendly 0xC0→0xC5, NPCAction 0xC5→0xCB, DropPickUp 0xCA→0xD0,
> ReactorHit 0xCD→0xD3, MultiChat 0x77→0x79, social band (Whisper 0x78→0x7A,
> Messenger 0x7A→0x7C, Interaction 0x7B→0x7D, Party 0x7C→0x7E, PartyReject 0x7D→0x7F,
> Guild 0x7E→0x82, GuildReject 0x7F→0x83, Buddy 0x82→0x86, Note 0x83→0x87,
> KeyMap 0x87→0x8B), and the cash-shop pair (0xE4→0xEA, 0xE5→0xEB). A re-sweep of every
> registered gameplay opcode (0x26–0xEB) against the v84 client send-opcode map found no
> further gross mismatch.

**Completeness vs v83 template:** 93 opcode entries = 93 distinct opcodes; the
table above has exactly 93 rows, one per opcode, all SAME. The only repeated
handler *name* is `ServerListRequestHandle` (bound at both 0x04 and 0x0B → two
distinct rows). `grep -o '"handler": "[^"]*"' … | sort -u | wc -l` = 92 = the 92
distinct handler-name strings (93 entries − 1 duplicate-bound name). **Unresolved /
explicitly low-confidence:** none at the guessed-value level — the byte of every
opcode is the v83 template value, which IS the wire send-opcode. The only inferred
claim is "v84 did not renumber," held at **med (OQ-7)** for the 88 rows not
independently re-read in the v84 binary, and **high/med** for the 5 verified
anchors. CharacterMoveHandle (0x29) is the one in-scope flow whose v84 byte was
not re-read (move opcode is not a flat COutPacket immediate); it is SAME by enum
stability + Section 5 (move.go deltas are all structural `> 83`, never opcode).

## 2. Outbound (writer) opcode map  (FR-1.1, FR-1.3)

**Scope & method.** "Outbound" = Atlas writers = server→client = the client
RECV/parse path. The authoritative in-scope set is every writer in the v83 seed
template `template_gms_83_1.json` (`socket.writers`): **112 entries across 108
distinct opcodes** (the 4-way `0x00` Auth* group + the dual-bound `0x0A`
ServerListEntry/End share opcodes). Each writer-name string is the Atlas logical
name and resolves to a `const ...Writer = "..."` in
`libs/atlas-packet/*/clientbound/` (verified by grep — e.g. `SetField` →
`field/clientbound/warp_to_map.go:16`, `CashShopOpen` →
`cash/clientbound/shop_open.go:14`, `FieldTransportState` →
`field/clientbound/transport.go:12`, `FieldEffectWeather` →
`field/clientbound/effect_weather.go:13`).

**Unlike the send path, the recv path IS a switch** — so the v84 opcode is read
*directly* off the case label the v83 parse-fn's structural analog sits under,
which makes SHIFTED detectable. The recv dispatch is a multi-stage cascade keyed
on the opcode `Decode2`'d in `CClientSocket::ProcessPacket` (v83 `@0x4965F1`, v84
`@0x49B502`):

- **Socket band 0x10–0x14, 0x19** — handled inline in `ProcessPacket` (migrate,
  alive, authen, crc). Not Atlas writers.
- **CWvsContext band** — v83 `0x1D–0x7C` → `CWvsContext::OnPacket@0xA07A08`; v84
  **`0x1D–0x7F`** → `@0xA51CD0`. The `ProcessPacket` band test literally widened:
  v83 `if (op<0x1D || op>0x7C)` → v84 `if (op<0x1D || op>0x7F)`. **+3 ceiling.**
- **Everything else** (`op<0x1D` or `op>band-ceiling`) → the active CStage vtable
  `(*(vtbl+8))(op,pkt)`, i.e. `CLogin::OnPacket` (login stage, band 0x00–0x1C) or
  `CField::OnPacket` (field stage, bands ≥ the CStage band up through 0x151), which
  themselves fan out by band to `CStage`/`CMapLoadable`/`CUserPool`/`CMobPool`/
  `CNpcPool`/`CDropPool`/`CReactorPool`/`CScriptMan`/dialog statics. Each pool's
  inner switch is also opcode-keyed, so the pool case label = the wire opcode.

**Key finding — the v84 recv map is a piecewise +Δ shift, not a clean rename.**
The +3 CWvsContext ceiling is the *seed* of a cascade of insertions that grows the
shift as you move up the opcode space. Three regimes:

1. **0x00–0x3E (login band + low CWvsContext): SHIFT = 0 (SAME).** Verified the v84
   `CLogin::OnPacket@0x60D075` case table is 1:1 with v83 `@0x5F80FF` (0x00–0x1C,
   same handlers), and v84 `CWvsContext::OnPacket` cases 0x1D–0x3E align 1:1 with
   v83 (`OnInventoryOperation`…`OnPartyResult`). The first insertion is at 0x3F.
2. **0x3F–0x52 (mid CWvsContext): SHIFT = +1 then +2** — non-Atlas handlers
   (Alliance, TownPortal-redirect, Marriage, Incubator, etc.) were inserted/merged,
   pushing the band. The shift collapses back to **0 at 0x53/0x54** (MonsterBook,
   re-anchored: v84 dispatches both via `CNpcPool::OnPacket(…,0x53/0x54)`, byte-equal
   to v83).
3. **≥ 0x7D (CStage + field/pool bands): SHIFT = +3 → +9**, growing per band because
   every band both *starts* +3 higher and *widens*. Measured per-band (below).

Per-band shift, measured by reading both versions' band-boundary tests + inner
pool switches:

| band (v83 → v84 stage fn) | v83 range | v84 range | Δ |
|---|---|---|---|
| CStage (SetField/ITC/CashShop) | 0x7D–0x7F | 0x80–0x82 | **+3** |
| CMapLoadable | 0x80–0x82 | 0x83–0x85 | +3 |
| CField inline (chat/effect/clock/transport) | 0x83–0x9F | 0x86–0xA2 | **+3** |
| CUserPool enter/leave/common (spawn/chat/pet/upgrade) | 0xA0–0xB8 | 0xA3–0xBC | **+3** |
| CUserPool remote (move/attack/foreign buffs/HP/guild mark) | 0xB9–0xCC | 0xBD–0xD0 | **+4** |
| CUserPool local (sit/effect/hint/UI/guide/cooldown) | 0xCD–0xEA | 0xD1–0xF0 | **+4** |
| CMobPool (spawn/move/stat/damage/health) | 0xEC–0x100 | 0xF2–0x107 | **+6** |
| CNpcPool (spawn/controller/action) | 0x101–0x107 | 0x108–0x10E | **+7** |
| CDropPool (drop spawn/destroy) | 0x10C–0x10D | 0x113–0x114 | **+7** |
| CReactorPool (hit/spawn/destroy) | 0x115–0x118 | 0x11C–0x11F | **+7** |
| CShopDlg (NPC shop) | 0x131–0x132 | 0x138–0x139 | **+7** |
| CScriptMan (NPC conversation) | 0x130 | 0x137 | **+7** |
| CFuncKeyMappedMan (keymap) | 0x14F–0x151 | 0x158–0x15A | **+9** |

Every v83 binary anchor is densely named; v84 per-opcode handlers are unnamed
`sub_XXXX` (positionally aligned against v83 by decompiled-body identity — never by
v84 symbol). Confidence grading:
- **high** = v84 case label read AND handler body byte/structure matched to the v83
  named fn (e.g. SetField, MonsterBook, TownPortal, BroadcastMsg, CashPetFood,
  Party, Buddy, the user/mob/npc/drop/reactor pool anchors).
- **med (band Δ)** = opcode derived by applying the measured per-band Δ to a row
  whose neighbour anchors were body-matched; v84 case label is in-band but the
  individual `sub_XXXX` not independently re-read.
- **low (OQ-7)** = upper dialog/cash-shop stages (Storage/Messenger/Interaction/
  CashShop) whose exact v84 case wasn't decompiled; Δ inferred from the surrounding
  +7→+9 upper-band trend. Flagged.

### Section 2 table (one row per v83 template writer entry)

| logical name | v83 opcode | v84 opcode | classification | evidence (v83 fn/addr; v84 fn/addr; confidence) |
|---|---|---|---|---|
| AuthSuccess | 0x00 | 0x00 | SAME | v83 `CLogin::OnCheckPasswordResult`@0x5F83EE (CLogin case 0); v84 CLogin case 0 `sub_60D368`@0x60D368; **high** |
| AuthTemporaryBan | 0x00 | 0x00 | SAME | same case-0 fn (result-code variant of OnCheckPasswordResult); **high** |
| AuthPermanentBan | 0x00 | 0x00 | SAME | same case-0 fn (result-code variant); **high** |
| AuthLoginFailed | 0x00 | 0x00 | SAME | same case-0 fn (result-code variant); **high** |
| ServerStatus | 0x03 | 0x03 | SAME | v83 `OnCheckUserLimitResult`@0x5F92AE (CLogin case 3); v84 CLogin case 3 `sub_60E275`; **high** |
| SetAccountResult | 0x04 | 0x04 | SAME | v83 `OnSetAccountResult`@0x5FC731 (case 4); v84 CLogin case 4 `sub_611809`; **high** |
| PinOperation | 0x06 | 0x06 | SAME | v83 `OnCheckPinCodeResult`@0x5FC89D (case 6); v84 CLogin case 6 `sub_611975`; **high** |
| PinUpdate | 0x07 | 0x07 | SAME | v83 `OnUpdatePinCodeResult`@0x5FCBC1 (case 7); v84 CLogin case 7 `sub_611C99`; **high** |
| CharacterViewAll | 0x08 | 0x08 | SAME | v83 `OnViewAllCharResult`@0x5FACCA (case 8); v84 CLogin case 8 `sub_60FFE8`; **high** |
| ServerListEntry | 0x0A | 0x0A | SAME | v83 `OnWorldInformation`@0x5F95B7 (case 0xA); v84 CLogin case 0xA `sub_60E5B3`; **high** |
| ServerListEnd | 0x0A | 0x0A | SAME | same case-0xA fn (world-list terminator variant); **high** |
| CharacterList | 0x0B | 0x0B | SAME | v83 `OnSelectWorldResult`@0x5F9891 (case 0xB); v84 `CLogin::OnSelectWorldResult`@0x60E8C6 (named in BOTH); **high** |
| ServerIP | 0x0C | 0x0C | SAME | v83 `OnSelectCharacterResult`@0x5FB541 (case 0xC); v84 CLogin case 0xC `sub_61085F`; **high** |
| CharacterNameResponse | 0x0D | 0x0D | SAME | v83 `OnCheckDuplicatedIDResult`@0x5F9C72 (case 0xD); v84 CLogin case 0xD `sub_60ECA7`; **high** |
| AddCharacterEntry | 0x0E | 0x0E | SAME | v83 `OnCreateNewCharacterResult`@0x5FA26C (case 0xE); v84 CLogin case 0xE `sub_60F268`; **high** |
| DeleteCharacterResponse | 0x0F | 0x0F | SAME | v83 `OnDeleteCharacterResult`@0x5F9D15 (case 0xF); v84 CLogin case 0xF `sub_60ED4A`; **high** |
| ChannelChange | 0x10 | 0x10 | SAME | socket-inline `OnMigrateCommand` (ProcessPacket case 0x10, both versions); **high** |
| Ping | 0x11 | 0x11 | SAME | socket-inline `OnAliveReq` (ProcessPacket case 0x11, both); **high** |
| SelectWorld | 0x1A | 0x1A | SAME | v83 `OnLatestConnectedWorld`@0x5F82F4 (CLogin case 0x1A); v84 CLogin case 0x1A `sub_60D26E`; **high** |
| ServerListRecommendations | 0x1B | 0x1B | SAME | v83 `OnRecommendWorldMessage`@0x5F8340 (case 0x1B); v84 CLogin case 0x1B `sub_60D2BA`; **high** |
| PicResult | 0x1C | 0x1C | SAME | v83 `OnCheckSPWResult`@0x5FBA49 (case 0x1C); v84 CLogin case 0x1C `sub_610D67`; **high** |
| CharacterInventoryChange | 0x1D | 0x1D | SAME | v83 `OnInventoryOperation`@0xA1EAD9 (CWvsContext case 0x1D); v84 case 0x1D `sub_A69D8F`; **high** |
| StatChanged | 0x1F | 0x1F | SAME | v83 `OnStatChanged`@0xA1FB52 (case 0x1F); v84 case 0x1F `sub_A6AE08`; **high** |
| CharacterBuffGive | 0x20 | 0x20 | SAME | v83 `OnTemporaryStatSet`@0xA202BE (case 0x20); v84 case 0x20 `sub_A6B6C3`; **high** |
| CharacterBuffCancel | 0x21 | 0x21 | SAME | v83 `OnTemporaryStatReset`@0xA2071F (case 0x21); v84 case 0x21 `sub_A6BB24`; **high** |
| CharacterSkillChange | 0x24 | 0x24 | SAME | v83 `OnChangeSkillRecordResult`@0xA1E48C (case 0x24); v84 case 0x24 `sub_A6972B`; **high** |
| FameResponse | 0x26 | 0x26 | SAME | v83 `OnGivePopularityResult`@0xA223DC (case 0x26); v84 case 0x26 `sub_A6D8EE`; **high** |
| CharacterStatusMessage | 0x27 | 0x27 | SAME | v83 `OnMessage`@0xA209D4 (case 0x27); v84 case 0x27 `sub_A6BDD9`; **high** |
| NoteOperation | 0x29 | 0x29 | SAME | v83 `OnMemoResult`@0xA2508B (case 0x29); v84 case 0x29 `sub_A70785`; **high** |
| HiredMerchantOperation | 0x32 | 0x32 | SAME | v83 `OnEntrustedShopCheckResult`@0xA27D75 (case 0x32); v84 case 0x32 `sub_A73538`; **high** |
| CompartmentMerge | 0x34 | 0x34 | SAME | v83 `OnGatherItemResult`@0xA1E943 (case 0x34); v84 case 0x34 `sub_A69BF9`; **high** |
| CompartmentSort | 0x35 | 0x35 | SAME | v83 `OnSortItemResult`@0xA1E96D (case 0x35); v84 case 0x35 `sub_A69C23`; **high** |
| GuildBBS | 0x3B | 0x3B | SAME | v83 `OnGuildBBSPacket`@0xA1233F (case 0x3B); v84 case 0x3B `sub_A5C77C`; **high** |
| CharacterInfo | 0x3D | 0x3D | SAME | v83 `OnCharacterInfo`@0xA2370B (case 0x3D); v84 case 0x3D `sub_A6EDA8`; **high** |
| PartyOperation | 0x3E | 0x3E | SAME | v83 `OnPartyResult`@0xA3E31C (case 0x3E); v84 case 0x3E `sub_A89CF3` (party-result body: COutPacket(126/127/129) invite responses, party-id Decode4); **high** |
| BuddyOperation | 0x3F | 0x40 | **SHIFTED** | v83 `OnFriendResult`@0xA3F2E8 (case 0x3F); v84 **case 0x40** `sub_A89CD4` (Decode1→`sub_528AD3`, friend-result shape); first insertion at 0x3F pushes Buddy +1; **high** |
| GuildOperation | 0x41 | 0x41 | SAME | v83 `OnGuildResult`@0xA37490 (case 0x41); v84 case 0x41 `sub_A8ADA2` (guild result/alliance-join switch); held SAME (insertion absorbed at 0x42); **med (band Δ)** |
| WorldMessage | 0x44 | 0x46 | **SHIFTED** | v83 `OnBroadcastMsg`@0xA22785 (case 0x44 — notice/megaphone/"#w"/slide-notice subtype switch 0–13); v84 **case 0x46** `sub_A6DC97` (byte-identical subtype switch incl. case-4 slide-notice, case-6 item-name, "#w"); +2; **high** |
| PetCashFoodResult | 0x4C | 0x4E | **SHIFTED** | v83 `OnCashPetFoodResult`@0xA29049 (case 0x4C — Decode1, pet-array index, `play_pet_sound(...,100)`, SP "EAT"/"cannot consume"); v84 **case 0x4E** `sub_A7480C` (Decode1, `*(petArr+8*Decode1+4)`, `sub_9CA9D5(...,100)` pet-sound); +2; **high** |
| MonsterBookSetCard | 0x53 | 0x53 | SAME | v83 `OnMonsterBookSetCard`@0xA081B8 (case 0x53); v84 case 0x53 `sub_A5263C`=`CNpcPool::OnPacket(…,0x53)` (re-anchor: shift collapses to 0); **high** |
| MonsterBookSetCover | 0x54 | 0x54 | SAME | v83 `OnMonsterBookSetCover`@0xA082D5 (case 0x54); v84 case 0x54 `sub_A52650`=`CNpcPool::OnPacket(…,0x54)`; **high** |
| ScriptProgress | 0x7A | 0x7A | SAME | v83 `OnScriptProgressMessage`@0xA13F20 (case 0x7A); v84 case 0x7A `sub_A76B5D` (script-progress msg body); within the un-shifted top of the CWvsContext band; **high** |
| CharacterSkillMacro | 0x7C | 0x7C | SAME | v83 `OnMacroSysDataInit`@0xA290F8 (case 0x7C); v84 case 0x7C `sub_A5E1CA` (Decode1→this[3635]); top of CWvsContext band, un-shifted; **high** |
| SetField | 0x7D | 0x80 | **SHIFTED** | v83 `CStage::OnSetField`@0x776020 (CStage case 125=0x7D); v84 `CStage::OnSetField`@0x798987 (CStage case 0x80); CStage band moved 0x7D–0x7F→0x80–0x82; **high** |
| CashShopOpen | 0x7F | 0x82 | **SHIFTED** | v83 `CStage::OnSetCashShop`@0x776A4F (CStage case 127=0x7F); v84 `CStage::OnSetCashShop` (CStage case 0x82); +3; **high** |
| CharacterMultiChat | 0x86 | 0x89 | **SHIFTED** | v83 `CField::OnGroupMessage`@0x531E00 (field inline 0x86); v84 field inline 0x89 `sub_53DAE2`; field-inline band +3; **high** |
| CharacterChatWhisper | 0x87 | 0x8A | **SHIFTED** | v83 `CField::OnWhisper`@0x53228E (0x87); v84 inline 0x8A `sub_53DC8E`; +3; **high** |
| FieldEffect | 0x8A | 0x8D | **SHIFTED** | v83 `CField::OnFieldEffect`@0x5330F7 (0x8A); v84 inline 0x8D `sub_53F37D`; +3; **high** |
| FieldEffectWeather | 0x8E | 0x91 | **SHIFTED** | v83 `CField::OnBlowWeather`@0x535179 (0x8E); v84 inline 0x91 `sub_53F2DD`; +3; **high** |
| Clock | 0x93 | 0x96 | **SHIFTED** | v83 field inline 0x93 = `(*(this+44))(…)` Clock vcall; v84 inline 0x96 = same `(*(this+44))(…)` vcall; +3 (re-anchored by the vtable+44 call, both versions); **high** |
| FieldTransportState | 0x95 | 0x98 | **SHIFTED** | v83 field inline 0x95 (transport state, in 0x83–0x9F band); v84 inline 0x98; +3 (band-Δ; neighbours Clock/effect body-matched); **med (band Δ)** |
| CharacterSpawn | 0xA0 | 0xA3 | **SHIFTED** | v83 `CUserPool::OnUserEnterField`@0x972100 (CUserPool case 0xA0); v84 `sub_9B20A0` (CUserPool case 0xA3, user-enter body: GetUser, field-add); +3; **high** |
| CharacterDespawn | 0xA1 | 0xA4 | **SHIFTED** | v83 `OnUserLeaveField`@0x9722F9 (0xA1); v84 `sub_9B2299` (0xA4, remove-user body); +3; **high** |
| CharacterChatGeneral | 0xA2 | 0xA5 | **SHIFTED** | v83 `CUser::OnChat` via common-anchor 162=0xA2; v84 common-anchor `sub_96E3ED` at 165=0xA5; +3; **high** |
| ChalkboardUse | 0xA4 | 0xA7 | **SHIFTED** | v83 `CUser::OnADBoard` (common 164=0xA4); v84 common 0xA7 `sub_96E8C0`-region; +3; **high** |
| CharacterItemUpgrade | 0xA7 | 0xAA | **SHIFTED** | v83 `CUser::ShowItemUpgradeEffect` (common 167=0xA7); v84 common 0xAA `sub_96E8C0` (ShowItemUpgrade body); +3; **high** |
| PetActivated | 0xA8 | 0xAB | **SHIFTED** | v83 `CUser::OnPetPacket` band 0xA8–0xAE; v84 pet band 170–178 = 0xAA–0xB2 (`sub_97015C`); 0xA8→0xAB; +3; **high** |
| PetMovement | 0xAA | 0xAD | **SHIFTED** | v83 pet band (0xAA); v84 pet band; +3; **med (band Δ)** |
| PetChat | 0xAB | 0xAE | **SHIFTED** | v83 pet band (0xAB); v84 pet band; +3; **med (band Δ)** |
| PetExcludeResponse | 0xAD | 0xB0 | **SHIFTED** | v83 pet band (0xAD); v84 pet band; +3; **med (band Δ)** |
| PetCommandResponse | 0xAE | 0xB1 | **SHIFTED** | v83 pet band (0xAE); v84 pet band; +3; **med (band Δ)** |
| CharacterMovement | 0xB9 | 0xBD | **SHIFTED** | v83 `CUserRemote::OnMove`@0x9726AE (remote case 0xB9); v84 `sub_9B26CD` (remote case 189=0xBD, OnMove body); remote band +4; **high** |
| CharacterAttackMelee | 0xBA | 0xBE | **SHIFTED** | v83 `CUserRemote::OnAttack` band 0xBA–0xBD; v84 `sub_9C0572` band 190–193=0xBE–0xC1; 0xBA→0xBE; +4; **high** |
| CharacterAttackRanged | 0xBB | 0xBF | **SHIFTED** | v83 OnAttack band; v84 attack band; +4; **high** |
| CharacterAttackMagic | 0xBC | 0xC0 | **SHIFTED** | v83 OnAttack band; v84 attack band; +4; **high** |
| CharacterAttackEnergy | 0xBD | 0xC1 | **SHIFTED** | v83 OnAttack band (last attack case 0xBD); v84 attack case 0xC1; +4; **high** |
| CharacterDamage | 0xC0 | 0xC4 | **SHIFTED** | v83 `CUserRemote::OnHit`@0x9832E3 (remote 0xC0); v84 `sub_9C3681` (remote case 196=0xC4, OnHit body); +4; **high** |
| CharacterExpression | 0xC1 | 0xC5 | **SHIFTED** | v83 remote 0xC1 = `CAvatar::SetEmotion(Decode4)`; v84 remote case 197=0xC5 = `Decode4`+`sub_4537A9` (SetEmotion); +4; **high** |
| CharacterShowChair | 0xC4 | 0xC8 | **SHIFTED** | v83 remote 0xC4 = `RemoteUser[3567]=Decode4` (show-chair field); v84 remote case 200=0xC8 = `*(v4+14660)=Decode4`; +4; **high** |
| CharacterAppearanceUpdate | 0xC5 | 0xC9 | **SHIFTED** | v83 `CUserRemote::OnAvatarModified`@0x98367E (0xC5); v84 `sub_9C3A1C` (remote case 201=0xC9); +4; **high** |
| CharacterEffectForeign | 0xC6 | 0xCA | **SHIFTED** | v83 `CUser::OnEffect`@0x9377D9 (remote 0xC6); v84 `sub_96EA92` (remote case 202=0xCA, OnEffect body); +4; **high** |
| CharacterBuffGiveForeign | 0xC7 | 0xCB | **SHIFTED** | v83 `CUserRemote::OnSetTemporaryStat`@0x98385D (0xC7); v84 `sub_9C3BFB` (remote case 203=0xCB); +4; **high** |
| CharacterBuffCancelForeign | 0xC8 | 0xCC | **SHIFTED** | v83 `CUserRemote::OnResetTemporaryStat`@0x983921 (0xC8); v84 `sub_9C3CBF` (remote case 204=0xCC); +4; **high** |
| PartyMemberHP | 0xC9 | 0xCD | **SHIFTED** | v83 `CUserRemote::OnReceiveHP`@0x9839EA (0xC9); v84 `sub_9C3D88` (remote case 205=0xCD); +4; **high** |
| GuildNameChanged | 0xCA | 0xCE | **SHIFTED** | v83 `CUserRemote::OnGuildNameChanged`@0x983A6A (0xCA); v84 `sub_9C3E08` (remote case 206=0xCE); +4; **high** |
| GuildEmblemChanged | 0xCB | 0xCF | **SHIFTED** | v83 `CUserRemote::OnGuildMarkChanged`@0x983AB5 (0xCB); v84 `sub_9C3E53` (remote case 207=0xCF); +4; **high** |
| CharacterSitResult | 0xCD | 0xD1 | **SHIFTED** | v83 `CUserLocal::OnSitResult`@0x959797 (local case 205=0xCD); v84 `sub_997968` (local case 209=0xD1, base `add eax,-209`); local band +4; **high** |
| CharacterEffect | 0xCE | 0xD2 | **SHIFTED** | v83 `CUser::OnEffect` (local case 206=0xCE); v84 `sub_96EA92` (local case 210=0xD2, OnEffect body); +4; **high** |
| CharacterHint | 0xD6 | 0xDA | **SHIFTED** | v83 `CUserLocal::OnBalloonMsg`@0x95D88B (local 0xD6); v84 local case 0xDA (balloon-msg); +4; **med (band Δ)** |
| UiOpen | 0xDC | 0xE0 | **SHIFTED** | v83 `CUserLocal::OnOpenUI`@0x9600F0 (local case 220=0xDC); v84 local case 0xE0; +4; **med (band Δ)** |
| UiLock | 0xDD | 0xE1 | **SHIFTED** | v83 `CUserLocal::SetDirectionMode` (local 0xDD); v84 local 0xE1; +4; **med (band Δ)** |
| UiDisable | 0xDE | 0xE2 | **SHIFTED** | v83 `CUserLocal::OnSetStandAloneMode` (local 0xDE); v84 local 0xE2; +4; **med (band Δ)** |
| GuideTalk | 0xE0 | 0xE4 | **SHIFTED** | v83 `CUserLocal::OnTutorMsg` (local 0xE0); v84 local 0xE4; +4; **med (band Δ)** |
| CharacterSkillCooldown | 0xEA | 0xEE | **SHIFTED** | v83 `CUserLocal::OnSkillCooltimeSet`@0x95BE66 (local case 234=0xEA); v84 local case 238=0xEE; +4; **high** |
| SpawnMonster | 0xEC | 0xF2 | **SHIFTED** | v83 `CMobPool::OnMobEnterField`@0x67945A (MobPool case 0xEC); v84 `CMobPool::OnMobEnterField`@0x68FFF0 (MobPool case 242=0xF2); MobPool band +6; **high** |
| DestroyMonster | 0xED | 0xF3 | **SHIFTED** | v83 `OnMobLeaveField` (0xED); v84 MobPool case 243=0xF3 (`OnMobLeaveField`); +6; **high** |
| ControlMonster | 0xEE | 0xF4 | **SHIFTED** | v83 `OnMobChangeController` (0xEE); v84 MobPool case 244=0xF4 (`OnMobChangeController`); +6; **high** |
| MoveMonster | 0xEF | 0xF5 | **SHIFTED** | v83 `OnMobPacket` band 0xEF–0xFF; v84 `OnMobPacket` band 245–262=0xF5–0x106; 0xEF→0xF5; +6; **high** |
| MoveMonsterAck | 0xF0 | 0xF6 | **SHIFTED** | v83 OnMobPacket band; v84 OnMobPacket band; +6; **med (band Δ)** |
| MonsterStatSet | 0xF2 | 0xF8 | **SHIFTED** | v83 OnMobPacket band; v84 OnMobPacket band; +6; **med (band Δ)** |
| MonsterStatReset | 0xF3 | 0xF9 | **SHIFTED** | v83 OnMobPacket band; v84 OnMobPacket band; +6; **med (band Δ)** |
| MonsterDamage | 0xF6 | 0xFC | **SHIFTED** | v83 OnMobPacket band; v84 OnMobPacket band; +6; **med (band Δ)** |
| MonsterHealth | 0xFA | 0x100 | **SHIFTED** | v83 OnMobPacket band; v84 OnMobPacket band; +6; **med (band Δ)** |
| SpawnNPC | 0x101 | 0x108 | **SHIFTED** | v83 `CNpcPool::OnNpcEnterField`@0x6D9993 (NpcPool case 0x101); v84 NpcPool case 0x108 `sub_6F0B33`; NpcPool band +7; **high** |
| SpawnNPCRequestController | 0x103 | 0x10A | **SHIFTED** | v83 `OnNpcChangeController` (0x103); v84 NpcPool case 0x10A `sub_6F0C26`; +7; **high** |
| NPCAction | 0x104 | 0x10B | **SHIFTED** | v83 `OnNpcPacket` band 0x104–0x106; v84 NpcPool `sub_6F0A7E` band 267–269=0x10B–0x10D; 0x104→0x10B; +7; **high** |
| DropSpawn | 0x10C | 0x113 | **SHIFTED** | v83 `CDropPool::OnDropEnterField`@0x505900 (DropPool case 0x10C); v84 DropPool case 0x113 `sub_50E789`; DropPool band +7; **high** |
| DropDestroy | 0x10D | 0x114 | **SHIFTED** | v83 `OnDropLeaveField`@0x506590 (0x10D); v84 DropPool case 0x114 `sub_50F409`; +7; **high** |
| ReactorHit | 0x115 | 0x11C | **SHIFTED** | v83 `CReactorPool::OnReactorChangeState`@0x73502D (ReactorPool case 0x115); v84 ReactorPool case 0x11C `sub_752622`; +7; **high** |
| ReactorSpawn | 0x117 | 0x11E | **SHIFTED** | v83 `OnReactorEnterField`@0x735127 (0x117); v84 ReactorPool case 0x11E `sub_75271C`; +7; **high** |
| ReactorDestroy | 0x118 | 0x11F | **SHIFTED** | v83 `OnReactorLeaveField`@0x73551F (0x118); v84 ReactorPool case 0x11F `sub_752B14`; +7; **high** |
| NPCConversation | 0x130 | 0x137 | **SHIFTED** | v83 `CScriptMan::OnPacket(…,0x130)` (CField routes 0x130→ScriptMan); v84 `CScriptMan::OnPacket@0x7684F4` case 0x137 (CField routes 0x137→ScriptMan); +7; **high** |
| NPCShop | 0x131 | 0x138 | **SHIFTED** | v83 `CShopDlg::OnPacket@0x756DA7` case 0x131 (set-shop); v84 CField routes 312–313=0x138–0x139→`CShopDlg`; 0x131→0x138; +7; **high** |
| NPCShopOperation | 0x132 | 0x139 | **SHIFTED** | v83 `CShopDlg::OnPacket` case 0x132 (shop result switch); v84 CShopDlg case 0x139; +7; **high** |
| StorageOperation | 0x135 | 0x13C | **SHIFTED** | v83 `CTrunkDlg::OnPacket` (CField case 0x135); v84 CField inline 0x13C `sub_7EEC1A`(=storage/trunk redirect); +7 (upper-band trend; exact stage case not re-decompiled); **low (OQ-7)** |
| MessengerOperation | 0x139 | 0x140 | **SHIFTED** | v83 `CUIMessenger::OnPacket` (CField case 0x139); v84 CField inline `sub_87CBD8`(case 0x140 region); +7 (upper-band trend); **low (OQ-7)** |
| CharacterInteraction | 0x13A | 0x141 | **SHIFTED** | v83 `CMiniRoomBaseDlg::OnPacketBase` (CField case 0x13A); v84 CField case 0x141 `sub_673DB5`; +7 (upper-band trend); **low (OQ-7)** |
| CashShopCashQueryResult | 0x144 | 0x14B | **SHIFTED (+7)** | VERIFIED: v83 `CCashShop::OnPacket`@0x478e2b case 0x144=`OnQueryCashResult`; v84 `CCashShop::OnPacket`@0x47BF59 (switch base 0x14A) case 0x14B=`OnQueryCashResult` (decodes 3 ints = wallet Credit/Points/Prepaid). **high (decompiled)** |
| CashShopOperation | 0x145 | 0x14C | **SHIFTED (+7)** | VERIFIED: v83 `CCashShop::OnPacket` case 0x145=`OnCashItemResult`; v84 `CCashShop::OnPacket`@0x47BF59 case 0x14C=`OnCashItemResult`. The whole CCashShop recv band shifted uniformly +7 (0x143→0x14A … 0x14D→0x154). **high (decompiled)** |
| CharacterKeyMap | 0x14F | 0x158 | **SHIFTED** | v83 `CFuncKeyMappedMan::OnInit`@0x58DDB4 (FuncKey case 0x14F); v84 CField routes 344–346=0x158–0x15A→`CFuncKeyMappedMan`; 0x14F→0x158; FuncKey band +9; **high** |
| CharacterKeyMapAutoHp | 0x150 | 0x159 | **SHIFTED** | v83 `OnPetConsumeItemInit` (FuncKey 0x150); v84 FuncKey case 0x159; +9; **high** |
| CharacterKeyMapAutoMp | 0x151 | 0x15A | **SHIFTED** | v83 `OnPetConsumeMPItemInit` (FuncKey 0x151); v84 FuncKey case 0x15A; +9; **high** |

### ADDED candidates 0x7D / 0x7E / 0x7F in v84 `CWvsContext::OnPacket`

The +3 CWvsContext ceiling created three new server→client cases at the top of the
bulk band. Decompiled:

- **v84 0x7D `sub_A5E47E`** — small Decode-only handler writing a `this[]` field
  (sibling of the v83 0x7B/0x7C MacroSysData/CRC cluster that shifted up). Not an
  in-scope flow; no Atlas writer maps to it.
- **v84 0x7E `sub_A5E4EB`** — likewise a small Decode-into-`this[]` setter; not
  in-scope.
- **v84 0x7F `sub_A748BB`** — Decode handler in the item/quest cluster
  (`sub_A748xx` neighbourhood); not an in-scope flow.

None of 0x7D–0x7F in the v84 CWvsContext band is required by the in-scope flows
(login / channel / map / spawn / move / chat) — those are all covered by the
SAME/SHIFTED rows above. **No ADDED row is needed for Component C's in-scope wiring.**
(The functional v84 SetField/CashShop at 0x80–0x82 are CStage writers and appear as
SHIFTED rows, not ADDED.)

### Completeness vs the v83 template

`grep -o '"writer": "[^"]*"' template_gms_83_1.json | sort -u | wc -l` = **112**
writer entries / **108 distinct opcodes**. The table above carries **all 112 rows**
(one per template entry, incl. the 4 `0x00` Auth* and the dual `0x0A`
ServerListEntry/End). Classification tally:

- **SAME: 47** — the entire login band (0x00–0x1C), the low CWvsContext band
  (0x1D–0x3E: inventory/stat/buff/skill/fame/status/note/merchant/compartment/
  guildBBS/charinfo/party), MonsterBook 0x53/0x54 (re-anchored), and ScriptProgress
  0x7A / SkillMacro 0x7C (un-shifted top of CWvsContext band).
- **SHIFTED: 65** — everything ≥ 0x7D (CStage/field/user/mob/npc/drop/reactor/
  dialog/keymap bands, +3→+9) plus the three mid-CWvsContext rows BuddyOperation
  (0x3F→0x40, +1), WorldMessage (0x44→0x46, +2), PetCashFoodResult (0x4C→0x4E, +2).
- **ADDED: 0** in-scope (v84 0x7D–0x7F exist but no in-scope flow needs them).
- **REMOVED: 0** — every v83 template writer has a live v84 analog.

**Cash-shop writers — VERIFIED (post-audit):** CashShopCashQueryResult 0x14B and
CashShopOperation 0x14C are decompile-confirmed against v84 `CCashShop::OnPacket`
@0x47BF59 (switch base 0x14A; whole CCashShop recv band shifted uniformly +7 from
v83 0x143). CashShopOpen 0x82 confirmed via `CStage::OnPacket`@0x79894b case 0x82.

**Upper-dialog writers — VERIFIED (post-audit):** all 3 decompile-confirmed via
`CField::OnPacket` (v83 @0x531325 → v84 @0x53d5a7), whose dialog band shifted
uniformly **+7** (OnHontaleTimer 0x12E→0x135, ZakumTimer, Trunk, RPSGame,
Messenger, MiniRoom, Parcel all +7):
- StorageOperation 0x135→**0x13C** (v84 case 0x13C → `CTrunkDlg::OnPacket` @0x7EEC1A)
- MessengerOperation 0x139→**0x140** (v84 case 0x140 → `CUIMessenger::OnPacket` @0x87CBD8)
- CharacterInteraction 0x13A→**0x141** (v84 case 0x141 → `CMiniRoomBaseDlg::OnPacketBase`
  @0x673DB5; `Decode1(op)` + miniroom factory/vtable routing, matching v83 OnPacketBase)

All three already held the correct values in the template (verification only, no
change). All in-scope writers (login handshake, world/channel
list, char list/select, enter-channel→SetField, map spawn, movement, chat) are
**high** confidence with v84 case labels read and bodies matched. No opcode is a
guessed hex value — every SHIFTED v84 opcode is a measured band-Δ applied to a
body-matched anchor or a directly-read case label.

## 3. Packet-structure delta (FR-1.2)

**Headline result (the payoff for B5/C1).** v84 GMS is a *minor* bump: for every
in-scope packet the v84 client (de)serializer is **byte-for-byte identical to
v83**. The structural fields Atlas currently adds for v84 via `MajorVersion() > 83`
predicates (decode-opt short, logout-gift block, `nCompletedSetItemID`, char-info
chair int, movement `XOffset/YOffset`, chat `updateTime`, char-create
`subJobIndex`, and the movement `dr*` header fields) are **NOT present in the v84
client** — they are all **v87+ (or later) additions**. The `> 83` boundary in
`libs/atlas-packet/**` is therefore a *systematic off-by-one*: the true GMS
boundary for these fields sits between v84 and v87 (so the predicate should be
`>= 87`, i.e. `> 84`), not between v83 and v84. **A v84 tenant driven by today's
`> 83` code would emit/consume extra bytes the v84 client never reads, desyncing
every one of these packets.** This is the single most important finding of A4 and
directly contradicts the Section-5 assumption that the `migrate+correct` rows are
"already correct for v84." They are correct for v87, wrong for v84.

Evidence base: each fact below cites the v84 IDA function actually read, with the
v83 anchor it was aligned against and (where the boundary mattered) the v87
function that proves the field is v87+.

### 3.1 In-scope flows (exhaustive): login handshake, auth, world/channel list, character list, character select / PIC-PIN, enter-channel, map load (spawn/field), movement, chat

Per-flow verdict legend: **identical** = v84 (de)serializer byte-equal to v83, no
Atlas branch involved; **Atlas-correct** = v84 differs from v83 and Atlas's
existing branch already matches v84; **MISMATCH** = Atlas's existing `>83`/`==83`
branch does NOT match what the v84 client expects (a real B5 bug).

#### 3.1.1 Login handshake / auth — verdict: **identical to v83** (no `>83` branch in this flow)
- v84 `CLogin::OnCheckPasswordResult` = CLogin case 0 `sub_60D368@0x60D368`
  (recv) decoded; the auth-success path reads, in order: `Decode4`(accountId),
  `Decode1`(gender), `Decode1`(grade), `Decode1`, `Decode1`, `DecodeStr`(name),
  `Decode1`, `Decode1`, two 8-byte buffers (`DecodeBuffer 8`), `Decode4`, then
  hands to `sub_60DD8D` (OnCommonLoginResult), then `Decode1`(v43)+`Decode1`(v44)
  trailing flags → sends 0x0B (world-list) or 0x09 (AfterLogin) + `DecodeBuffer 8`.
- v83 `CLogin::OnCheckPasswordResult@0x5F83EE` is **byte-identical** in the
  success path (`SetAccountInfo`, then `v60`/`v61` flags → 0xB / 0x9). No version
  predicate. **Atlas login/auth writers carry no `>83` structural branch — nothing
  to fix.** (See §4 for the usesPin consequence.)
- Client SEND `LoginHandle 0x01`: v84 `CLogin::SendCheckPasswordPacket@0x60B88B`
  retains the v83 symbol and shape (Section 1, high). Identical.

#### 3.1.2 World / channel list — verdict: **identical to v83**
- v84 world-list = CLogin case 0xA `sub_60E5B3` (`OnWorldInformation`);
  char-list = `CLogin::OnSelectWorldResult@0x60E8C6` (named in BOTH versions,
  Section 2 high). Both align 1:1 with v83 `OnWorldInformation@0x5F95B7` /
  `OnSelectWorldResult@0x5F9891`. No GMS `>83` predicate exists in
  `character/clientbound/list.go` (only `<=28` early-return and `>87`/JMS gates,
  all evaluating identically for v83 and v84 — Section 5 rows
  list.go:56/61/63/91/96/98 confirmed unchanged). **No structural delta; Atlas
  already correct for v84.**

#### 3.1.3 Character list / AllCharacterList extra fields — verdict: **identical to v83**
- `character/clientbound/list.go` per-char encode: the only version gates are
  `<=28` (early-return, false for both v83/v84) and `>87`/JMS (false for both).
  v84 char-list dispatch `OnSelectWorldResult@0x60E8C6` is the v83 function
  relocated; the per-character record layout is unchanged across v83→v84 (no
  `>83` field). **No AllCharacterList extra field appears at v84.** Atlas correct.

#### 3.1.4 Character select / PIC-PIN — verdict: **identical to v83**
- PIC result = CLogin case 0x1C `sub_610D67` (`OnCheckSPWResult`), char-select
  ServerIP = case 0xC `sub_61085F` (`OnSelectCharacterResult`) — both align to
  v83 (`OnCheckSPWResult@0x5FBA49`, `OnSelectCharacterResult@0x5FB541`). The PIC
  send path (`delete.go`, `create.go` SPW) gates are `>82` (true both) — no `>83`
  structural shift in select/PIC. **No delta; Atlas correct.** (PIN flow: §4.)

#### 3.1.5 Character create (`subJobIndex`) — verdict: **MISMATCH (likely; v83 proven, v84 by pattern + v83 send)**
- v83 client SEND `CLogin::SendNewCharPacket@0x5F7E7A`: `COutPacket(0x16)`,
  `EncodeStr(name)`, `Encode4(race)`, **8×`Encode4`** (face/hair/hairColor/skin/
  top/bottom/shoes/weapon), `Encode1(gender)`. **No `subJobIndex` short.** This
  matches Atlas `create.go` with `>83` = false for v83.
- v84: the basic-create send (`COutPacket(0x16)`) was not isolated in the v84
  CLogin send cluster within the A4 budget (v84 routes most creation through the
  SPW/view-all path `sub_60C624`, which builds 0x0E/0x1F/0x20, not 0x16; the bare
  0x16 builder was not positively located). Confidence **MED**: by the proven
  universal `>83`→`>=87` pattern (§3 headline) plus the v83 send having no
  subJobIndex, v84 almost certainly does **not** send a `subJobIndex` short.
- **Atlas branch:** `create.go:116` writes `WriteShort(SubJobIndex())` and
  `create.go:153` (`<=83`) zero-defaults it. For a v84 tenant `>83` = true →
  Atlas would **read/expect a 2-byte subJobIndex the v84 client does not send**,
  desyncing char-create. **B5 fix:** change `create.go:116` and `create.go:153`
  boundary from `> 83` / `<= 83` to `>= 87` / `< 87` (GMS). (Confirm against a
  v84 0x16 send capture before landing, given MED confidence.)

#### 3.1.6 Enter-channel → SetField (decode-opt header + logout-gift block) — verdict: **MISMATCH (confirmed)**
- v84 `CStage::OnSetField@0x798987` decoded and aligned field-for-field against
  v83 `CStage::OnSetField@0x776020`: **byte-identical.** Parse order (both):
  `Decode4`(channelId/seed) → `Decode1`(sNotifierMessage) → `Decode1`
  (bCharacterData) → `Decode2`(nNotifierCheck) → notifier strings → branch.
- **decode-opt short:** v84 reads **NO** leading 2-byte decode-opt before
  channelId. Proof it is a v87+ field: v87 `CStage::OnSetField@0x7c429c` calls
  **`CClientOptMan::DecodeOpt@0x4a513e`** (reads `Decode2` count + N×8-byte pairs)
  *before* `Decode4(channelId)`; v83 and v84 have no such call. Atlas
  `set_field.go:46/92` and `warp_to_map.go:56/96` write/read `WriteShort(0)`
  decode-opt under `(GMS && >83) || JMS`. **For v84 this emits a spurious 2-byte
  short the client never consumes → whole SetField/WarpToMap desyncs.**
- **logout-gift block:** v84 `OnSetField` bCharacterData branch reads 3 damage
  seeds → `sub_799FBF` (a *local* CharacterData allocation that reads **zero**
  packet bytes — same body as v83's mis-named `OnSetLogoutGiftConfig@0x777616`)
  → `CharacterData::Decode` (`sub_4EDDE5`, the full char-data parser, last). **No
  4-int logout-gift block is read in v83 OR v84.** Proof it is v87+: v87
  `OnSetLogoutGiftConfig@0xa990f2` reads **4×`Decode4`** and is invoked with the
  packet *after* `CharacterData::Decode`. Atlas `set_field.go:75` writes a 4-int
  logout-gift block under `(GMS && >83) || JMS`. **For v84 this emits 16 spurious
  bytes after char-data.**
- **m_dwOldDriverID (`>=95`)** and **nHP width / nNotifierCheck (`>28`)**: Atlas
  gates these `>=95` / `>28`, both evaluate identically for v83 and v84 (already
  verified in existing warp_to_map.go comments). **No delta there.**
- **B5 fix:** in `set_field.go` (lines 46, 75, 92, 121) and `warp_to_map.go`
  (lines 56, 96) change the decode-opt and logout-gift gates from
  `(GMS && MajorVersion() > 83) || JMS` to `(GMS && MajorVersion() >= 87) || JMS`
  (JMS clause unaffected — JMS still gets both, which existing tests assume).

#### 3.1.7 Map load / character spawn (`nCompletedSetItemID`) — verdict: **MISMATCH (confirmed)**
- v84 spawn parse = `CUserRemote::Init` (`sub_9BF6F0`, reached from CUserPool
  user-enter `sub_9B20A0@0x9B20A0`, the v84 case-0xA3 handler). After
  `AvatarLook::Decode` it reads **exactly 3×`Decode4`**: choco, item-effect,
  chair (`*(this+14660)` = SetActivePortableChair). **No `nCompletedSetItemID`.**
- v83 `CUserRemote::Init@0x97f55d` reads the same 3×`Decode4` (choco, item-effect,
  chair) — byte-identical to v84.
- Proof it is v87+: v87 `CUserRemote::Init@0xa04b9e` reads **4×`Decode4`** there
  (choco, item-effect, **`nCompletedSetItemID`** `*(this+11652)`, chair) and calls
  `CUser::SetSetItemEffect`. So the field enters at v87.
- **Atlas branch:** `spawn.go:85` (Encode) / `spawn.go:188` (Decode) write/read
  `nCompletedSetItemID` int under `GMS && MajorVersion() > 83`. **For a v84 tenant
  `>83` = true → Atlas emits a spurious 4-byte field mid-spawn → every foreign
  CharacterSpawn desyncs (mount/pets/position garbled).**
- **B5 fix:** `spawn.go:85` and `spawn.go:188` change `> 83` → `>= 87` (GMS).

#### 3.1.8 Movement (`XOffset`/`YOffset` on NORMAL elements) — verdict: **MISMATCH (confirmed) + Atlas internal inconsistency**
- v84 movement element parser `CMovePath::Decode` (`sub_6A0FD0`, reached via
  `CUserRemote::OnMove sub_9B26CD@0x9B26CD` → `sub_6A203F`): NORMAL element cases
  (0/5/0xF/0x11) read **exactly 5×`Decode2`** (X,Y,Vx,Vy,Fh). **No `XOffset`/
  `YOffset`.** (v84 adds an unrelated new move-type case 0x17 with 4 Decode2 — not
  XOffset.)
- v83 `CMovePath::Decode@0x68a33c` reads the same 5 fields — byte-identical.
- **Atlas branch:** `model/movement.go:128` (`NormalElement.Decode`) reads
  `XOffset/YOffset` under `Region() != "GMS" || MajorVersion() > 83`. For GMS v84
  `>83` = true → Atlas reads **2 extra Int16 the v84 client never sent**,
  desyncing every movement element after the first NORMAL. **Notably the matching
  Encode (`movement.go:217`) already gates `> 87`** — so Atlas is internally
  inconsistent (decode `>83`, encode `>87`); the *encode* boundary is the correct
  one. **B5 fix:** `movement.go:128` change `MajorVersion() > 83` → `> 87` to match
  the encode side (which the v83/v84 evidence confirms is right).

#### 3.1.9 Movement self-move header (`dr0/dr1/dr2/dr3/dwKey/crc32`) — verdict: **MISMATCH (likely; same off-by-one family, MED)**
- `character/serverbound/move.go:56-71/82-97` gates the `dr*`/`dwKey`/`crc32`
  header fields on `GMS && MajorVersion() > 83` (and `crc` on `>28`). v83 self-move
  carries only `fieldKey` + `crc(>28)` before the move path; the `dr*`/`dwKey`/
  `crc32` block is a later (v95-era position-validation) addition.
- v84: the bare 0x29 self-move builder was not isolated in v84 (move opcode is not
  a flat `COutPacket` immediate; the CUserLocal send is unnamed). Confidence
  **MED** by the same `>83`→`>=87`/later off-by-one family as 3.1.6–3.1.8: the v84
  client self-move almost certainly carries **no `dr*` block** (v83-style).
- **B5 fix (pending v84 self-move capture):** raise the `move.go` `> 83` gates to
  the correct GMS boundary (likely `>= 87` or `>= 95` — confirm against a v84 move
  capture; the `dr*` fields may even be `>= 95`). The `crc (>28)` gate is fine.

#### 3.1.10 Chat (general `updateTime`) — verdict: **MISMATCH (confirmed)**
- v84 client SEND general chat `sub_5382D7` (Section 1, size 0xF7): builds
  `COutPacket(0x31)`, `EncodeStr(msg)`, `Encode1(bOnlyBalloon)`. **No
  `updateTime`.**
- v83 `CField::SendChatMsg@0x52C315` is byte-identical (`COutPacket(0x31)`,
  `EncodeStr`, `Encode1`) — no updateTime.
- **Atlas branch:** `chat/serverbound/general.go:45/57` writes/reads `updateTime`
  (uint32) before the msg string under `(GMS && >83) || JMS`. **For a v84 tenant
  `>83` = true → Atlas reads 4 bytes of the message as updateTime → chat desyncs.**
  (The clientbound `chat/clientbound/general.go` gates the same way; both wrong for
  v84.)
- **B5 fix:** `chat/serverbound/general.go:45/57` (and the clientbound counterpart)
  change `> 83` → the correct GMS boundary. The exact upper boundary (87 vs 95) was
  **not pinned in A4** (v87/v95 chat send not isolated); `updateTime` is a later
  GMS chat addition, plausibly `>= 95`. **Confirm the boundary before landing**;
  what is certain is that **v84 has no updateTime**, so any boundary `> 84` fixes
  v84.

#### 3.1.11 Char-info (`>83` chair int) — verdict: **MISMATCH (confirmed)**
- (Char-info is a select/inspect flow, in-scope-adjacent; included because
  Section 5 flags info.go:116/183 as `migrate+correct`.) v84 char-info =
  CWvsContext case 0x3D `sub_A6EDA8@0xA6EDA8`; v83 `OnCharacterInfo@0xA2370B`.
  Both end the body with the medal block (`MedalAchievementInfo::Decode` in v83 /
  its v84 analog) and then cleanup — **neither reads a trailing chair `Decode4`**
  after the medal block.
- **Atlas branch:** `info.go:116/183` writes/reads a chair int at the end under
  `(GMS && >83) || JMS`. **For a v84 tenant `>83` = true → Atlas emits 4 spurious
  trailing bytes** (less catastrophic, being last, but still a length mismatch).
  **B5 fix:** `info.go:116/183` change `> 83` → `>= 87` (GMS).

### 3.2 Spot-checked elsewhere (what was checked, what was assumed)

- **Cash shop (out of scope):** the `cash/clientbound/shop_open.go` /
  `query_result.go` predicates (Section 5 rows) are all `>12`/`>=95`/`==GMS`/
  `==JMS` — every one evaluates identically for v83 and v84 (no `>83` boundary in
  the cash-shop writers). **Assumed unchanged for v84; not byte-verified** (cash
  shop is out of scope and not on the playthrough path). The CStage cash-shop
  *opcode* shift (CashShopOpen 0x7F→0x82) is opcode-numbering (Section 2), not
  structure.
- **Storage / messenger / interaction (out of scope):** opcode rows are the
  low-confidence upper-band `+7` SHIFTs (Section 2, flagged OQ-7). **Packet
  *structure* not examined**; these carry no `>83` Atlas branch in Section 5, so
  no structural off-by-one is suspected, but this is **assumed, not verified.**
- **Pet spawn (`model.Pet` inside spawn):** the v84 spawn pet loop
  (`while(Decode1){...CPet::Init}`) is positionally identical to v83 in
  `CUserRemote::Init` (both `sub_9BF6F0`/`0x97f55d`). The pet sub-structure was not
  field-diffed but sits inside an already byte-matched enclosing function;
  **assumed identical for v84.**
- **NPC / monster / drop / reactor spawn structure:** out of scope for the A4
  playthrough; only their opcodes were mapped (Section 2). Structure **not
  examined** — assumed v83-equivalent given v84's minor-bump character, but
  unverified.
- **Movement non-NORMAL element types** (Teleport/StartFallDown/Jump/etc.):
  v84 `CMovePath::Decode` (`sub_6A0FD0`) case bodies for these were read in passing
  and match v83 `0x68a33c` case-for-case (same Decode2 counts), except v84 adds a
  **new** case 0x17 (4×Decode2) absent in v83 — a new move-action type, not a
  change to existing types. Not an in-scope concern unless that action is emitted.

#### Other `> 83` rows in Section 5 NOT individually IDA-verified in A4
The same systematic `> 83`→`>= 87` off-by-one almost certainly applies to the
remaining `> 83` GMS-true rows that A4 did not have budget to decompile
individually, because **every** `> 83` field A4 *did* examine (8 distinct fields
across 6 packets) turned out to be a v87+ addition with no v84/v83 difference — a
100% hit rate. These un-decompiled rows are flagged so B5 treats them as
**probably-MISMATCH, confirm-then-fix** rather than trusting the `> 83` gate:
- `model/character_temporary_stat.go:105` (ShadowPartner buff `> 83`) — appears in
  the enter-channel/spawn CTS block; in-scope-adjacent. **Likely `>= 87`.**
- `model/monster.go:512` (monster spawn `> 83` field) — map-load path; not on the
  minimal playthrough but adjacent. **Likely `>= 87`.**
- `guild/clientbound/operation.go:430/447` (`> 83`) — out of scope (guild), but
  same family.
- `login/serverbound/all_character_list_request.go:56/70` (`> 83`) — the
  view-all-characters request; in-scope-adjacent. **Likely `>= 87`.**

None of these is on the minimal A4 playthrough (login→world→char-select→enter
field→move→chat), so they do not block C1's smoke-test, but B5 should re-gate them
to `>= 87` after a one-function v84 confirmation each, given the perfect pattern.

## 4. usesPin determination (OQ-1)

**`usesPin = false` for v84 GMS** — identical to v83, and consistent with every
current GMS tenant template default.

**Evidence (the v84 login post-auth flow):**
- v84 `CLogin::OnCheckPasswordResult` = CLogin dispatch case 0 `sub_60D368`
  (`@0x60D368`, the AuthSuccess handler). Decompiled in full. On auth success
  (`m_nRegStatID <= 1`) the client: decodes the account record, calls `sub_60DD8D`
  (OnCommonLoginResult / SetAccountInfo), then reads two trailing flag bytes
  (`v43 = Decode1`, `v44 = Decode1` → `*(this+392)` = the SPW/PIC-enabled flag) and
  **immediately** sends either `COutPacket(0x0B)` (ServerListRequest / world list)
  or `COutPacket(0x09)` (AfterLogin). **There is no `SetPin`/`CheckPin` dialog
  branch, no PIN prompt, and no PIN packet emitted in the success path.** The
  transition is auth → world-select, exactly as v83.
- v83 `CLogin::OnCheckPasswordResult@0x5F83EE` is **byte-identical** in the success
  path: reads `v60`/`v61` flags → sends 0xB (world list) or 0x9 (AfterLogin). Same
  flow, no PIN dialog.
- The `PinOperation`/`PinUpdate` handlers exist in the dispatch table of both
  versions (CLogin case 6 = `OnCheckPinCodeResult` v84 `sub_611975`; case 7 =
  `OnUpdatePinCodeResult` v84 `sub_611C99`) but they are **server-initiated** — the
  client only runs a PIN dialog if the server *sends* a pin-operation packet. The
  client's own login sequence does **not** require or initiate a PIN step. This is
  exactly the v83 situation, where Atlas already ships `usesPin: false`.

**Conclusion for C1/template:** the v84 GMS tenant template should keep
`usesPin: false` (no change from the v83 template). The proving functions are v84
`sub_60D368` (auth-success → straight to world list) cross-checked against v83
`OnCheckPasswordResult@0x5F83EE`.

## 5. Version-branch audit table (FR-3.1, FR-3.3)

Grep command: `grep -rn 'Region()\|MajorVersion()\|MinorVersion()' services/ libs/ --include='*.go' | grep -v '_test.go' | grep -E '==|!=|>=|<=|>|<'`

Total hits: **412**

Evaluation key (for GMS tenant, region="GMS"):
- `> 83` → v83: **false**, v84: **true** (boundary predicate — changes at v84)
- `>= 87` → v83: false, v84: false (unchanged)
- `>= 95` → v83: false, v84: false (unchanged)
- `>= 90` → v83: false, v84: false (unchanged)
- `>= 83` → v83: true, v84: true (unchanged)
- `>= 73` → v83: true, v84: true (unchanged)
- `> 87` → v83: false, v84: false (unchanged)
- `> 82` → v83: true, v84: true (unchanged)
- `> 28` → v83: true, v84: true (unchanged)
- `> 12` → v83: true, v84: true (unchanged)
- `<= 12` → v83: true, v84: true (unchanged)
- `<= 28` → v83: true, v84: true (unchanged)
- `<= 83` → v83: **true**, v84: **false** (boundary predicate — changes at v84)
- `<= 87` → v83: true, v84: true (unchanged)
- `<= 95` → v83: true, v84: true (unchanged)
- `== 83` → v83: **true**, v84: **false** (changes at v84)
- `!= "GMS"` or `!= "JMS"` → region comparisons, not version; evaluated as-is for GMS

In-scope flows: login handshake, auth, world/channel list, character list, character select/PIC-PIN, enter-channel, map load (spawn/field), movement, chat.

| branch site (file:line) | predicate | v83 result | v84 result | correct for v84? | action | delta evidence |
|---|---|---|---|---|---|---|
| libs/atlas-packet/buddy/clientbound/invite.go:50 | `Region() != "GMS" \|\| MajorVersion() >= 87` | false (GMS && 83<87) | false (GMS && 84<87) | yes — no change | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/buddy/clientbound/invite.go:76 | `Region() != "GMS" \|\| MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/query_result.go:41 | `Region() == "GMS" && MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/query_result.go:53 | `Region() == "GMS" && MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_inventory.go:132 | `(Region() == "GMS" && MajorVersion() >= 95) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:36 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:39 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:44 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:45 | `MajorVersion() <= 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:52 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:56 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:60 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:66 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:85 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:90 | `Region() == "GMS" && MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:94 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:97 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:111 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:114 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:119 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:120 | `MajorVersion() <= 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:127 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:131 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:135 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:141 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:162 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:170 | `Region() == "GMS" && MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:177 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/clientbound/shop_open.go:180 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/item_use.go:38 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/item_use.go:50 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy.go:44 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy.go:57 | `Region() == "GMS" && MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy.go:75 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy.go:87 | `Region() == "GMS" && MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go:46 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go:56 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go:80 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go:89 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go:46 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go:56 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go:80 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go:89 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_gift.go:49 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_gift.go:59 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_gift.go:65 | `Region() == "GMS" && MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_gift.go:83 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_gift.go:92 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_gift.go:98 | `Region() == "GMS" && MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go:42 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go:52 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go:72 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go:81 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/attack.go:107 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/attack.go:165 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/damage.go:55 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/damage.go:78 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/expression.go:62 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/expression.go:65 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/expression.go:80 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/expression.go:83 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/info.go:106 | `(Region() == "GMS" && MajorVersion() <= 87) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/info.go:116 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.11): v84 OnCharacterInfo sub_A6EDA8 has no trailing chair Decode4 (==v83). Fix: `> 83`→`>= 87` |
| libs/atlas-packet/character/clientbound/info.go:173 | `(Region() == "GMS" && MajorVersion() <= 87) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/info.go:183 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.11): v84 OnCharacterInfo sub_A6EDA8 has no trailing chair Decode4 (==v83). Fix: `> 83`→`>= 87` |
| libs/atlas-packet/character/clientbound/item_upgrade.go:91 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/item_upgrade.go:98 | `(Region() == "GMS" && MajorVersion() > 87) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/item_upgrade.go:114 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/item_upgrade.go:119 | `(Region() == "GMS" && MajorVersion() > 87) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:56 | `Region() == "GMS" && MajorVersion() <= 28` | true | true | yes — v84 still > 28, predicate is false; note: the `<= 28` check is the early-return path | unchanged (correct) | pending Phase A (character list) |
| libs/atlas-packet/character/clientbound/list.go:61 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:63 | `MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:91 | `Region() == "GMS" && MajorVersion() <= 28` | true | true | yes — same as :56 (early-return path) | unchanged (correct) | pending Phase A (character list) |
| libs/atlas-packet/character/clientbound/list.go:96 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:98 | `MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:47 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:81 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:66 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/list.go:101 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:79 | `(Region() == "GMS" && MajorVersion() > 87) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:85 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.7): v84 CUserRemote::Init sub_9BF6F0 reads 3 Decode4 (no nCompletedSetItemID); v87 reads 4. Fix: `> 83`→`>= 87` |
| libs/atlas-packet/character/clientbound/spawn.go:128 | `Region() == "GMS" && MajorVersion() < 95` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:134 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:135 | `MajorVersion() <= 87` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:138 | `MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:182 | `(Region() == "GMS" && MajorVersion() > 87) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:188 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.7): v84 CUserRemote::Init sub_9BF6F0 reads 3 Decode4 (no nCompletedSetItemID); v87 reads 4. Fix: `> 83`→`>= 87` |
| libs/atlas-packet/character/clientbound/spawn.go:219 | `Region() == "GMS" && MajorVersion() < 95` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:225 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:226 | `MajorVersion() <= 87` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:229 | `MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/status_message.go:528 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/status_message.go:561 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/view_all.go:83 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/view_all.go:103 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:114 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:125 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:148 | `(Region() == "GMS" && MajorVersion() > 28 && MajorVersion() <= 87) \|\| Region() == "JMS"` | true | true | yes — v84 is in (28,87] | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:152 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:170 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:181 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:207 | `(Region() == "GMS" && MajorVersion() > 28 && MajorVersion() <= 87) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:211 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:243 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:266 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:272 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:273 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:280 | `MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:310 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:333 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:339 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:340 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:347 | `MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:364 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:372 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:380 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:390 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:397 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:403 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:410 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:419 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:428 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:437 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:449 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:457 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:468 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:474 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:480 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:486 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:492 | `Region() == "GMS" && MajorVersion() < 28` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:502 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:526 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:575 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:598 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:621 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:642 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:664 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:672 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:682 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:693 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/serverbound/create.go:113 | `(Region() == "GMS" && MajorVersion() >= 73) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (character create) |
| libs/atlas-packet/character/serverbound/create.go:116 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.5): v83 SendNewChar@0x5F7E7A has no subJobIndex; v84 same (MED); field is v87+. Fix: `> 83`→`>= 87` |
| libs/atlas-packet/character/serverbound/create.go:129 | `(Region() == "GMS" && MajorVersion() > 28) && Region() != "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (character create) |
| libs/atlas-packet/character/serverbound/create.go:132 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character create) |
| libs/atlas-packet/character/serverbound/create.go:147 | `(Region() == "GMS" && MajorVersion() >= 73) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (character create) |
| libs/atlas-packet/character/serverbound/create.go:153 | `Region() == "GMS" && MajorVersion() <= 83` | true | **false** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.5): v83 SendNewChar@0x5F7E7A has no subJobIndex; v84 same (MED); field is v87+. Fix: `> 83`→`>= 87` |
| libs/atlas-packet/character/serverbound/create.go:162 | `Region() != "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/serverbound/create.go:172 | `(Region() == "GMS" && MajorVersion() <= 28) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (character create) |
| libs/atlas-packet/character/serverbound/create.go:179 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character create) |
| libs/atlas-packet/character/serverbound/delete.go:51 | `Region() == "GMS" && MajorVersion() > 82` | true | true | yes | unchanged (correct) | pending Phase A (character delete/PIC) |
| libs/atlas-packet/character/serverbound/delete.go:53 | `Region() == "GMS"` (else-if of `> 82`) | true | true | yes (never reached since `> 82` is true) | unchanged (correct) | pending Phase A (character delete/PIC) |
| libs/atlas-packet/character/serverbound/delete.go:64 | `Region() == "GMS" && MajorVersion() > 82` | true | true | yes | unchanged (correct) | pending Phase A (character delete/PIC) |
| libs/atlas-packet/character/serverbound/delete.go:67 | `Region() == "GMS"` (else-if of `> 82`) | true | true | yes (never reached) | unchanged (correct) | pending Phase A (character delete/PIC) |
| libs/atlas-packet/character/serverbound/expression.go:58 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/serverbound/expression.go:73 | `Region() == "GMS" && MajorVersion() > 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/serverbound/heal_over_time.go:60 | `Region() == "GMS" && MajorVersion() <= 95` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/serverbound/heal_over_time.go:74 | `Region() == "GMS" && MajorVersion() <= 95` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/serverbound/move.go:56 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4 MISMATCH likely (see §3.1.9, MED): v84 self-move not isolated; dr* are v87+/v95-era. Fix: raise `> 83` (confirm boundary) |
| libs/atlas-packet/character/serverbound/move.go:61 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4 MISMATCH likely (see §3.1.9, MED): same dr* family. Fix: raise `> 83` (confirm boundary) |
| libs/atlas-packet/character/serverbound/move.go:65 | `Region() == "GMS" && MajorVersion() > 28` | true | true | yes | unchanged (correct) | pending Phase A (movement) |
| libs/atlas-packet/character/serverbound/move.go:68 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4 MISMATCH likely (see §3.1.9, MED): same dr* family. Fix: raise `> 83` (confirm boundary) |
| libs/atlas-packet/character/serverbound/move.go:82 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4 MISMATCH likely (see §3.1.9, MED): v84 self-move dr* almost certainly absent. Fix: raise `> 83` (confirm boundary) |
| libs/atlas-packet/character/serverbound/move.go:87 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4 MISMATCH likely (see §3.1.9, MED): same dr* family. Fix: raise `> 83` (confirm boundary) |
| libs/atlas-packet/character/serverbound/move.go:91 | `Region() == "GMS" && MajorVersion() > 28` | true | true | yes | unchanged (correct) | pending Phase A (movement) |
| libs/atlas-packet/character/serverbound/move.go:94 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4 MISMATCH likely (see §3.1.9, MED): same dr* family. Fix: raise `> 83` (confirm boundary) |
| libs/atlas-packet/chat/serverbound/general.go:45 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.10): v84 chat send sub_5382D7 has no updateTime (==v83 SendChatMsg@0x52C315). Fix: raise `> 83` (boundary 87 vs 95 TBD) |
| libs/atlas-packet/chat/serverbound/general.go:57 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.10): v84 chat send sub_5382D7 has no updateTime (==v83 SendChatMsg@0x52C315). Fix: raise `> 83` (boundary 87 vs 95 TBD) |
| libs/atlas-packet/chat/serverbound/multi.go:54 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/chat/serverbound/multi.go:71 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/chat/serverbound/whisper.go:60 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/chat/serverbound/whisper.go:75 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/field/clientbound/affected_area_created.go:91 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/field/clientbound/effect_weather.go:40 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/field/clientbound/effect_weather.go:70 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/field/clientbound/set_field.go:46 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.6): v84 OnSetField@0x798987 has no decode-opt; v87@0x7c429c calls DecodeOpt. Fix: `> 83`→`>= 87` |
| libs/atlas-packet/field/clientbound/set_field.go:50 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/field/clientbound/set_field.go:60 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/field/clientbound/set_field.go:75 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.6): v84 OnSetField reads no logout-gift block; v87 OnSetLogoutGiftConfig@0xa990f2 reads 4 ints. Fix: `> 83`→`>= 87` |
| libs/atlas-packet/field/clientbound/set_field.go:92 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.6): v84 has no decode-opt short. Fix: `> 83`→`>= 87` |
| libs/atlas-packet/field/clientbound/set_field.go:96 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/field/clientbound/set_field.go:106 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/field/clientbound/set_field.go:121 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.6): v84 reads no logout-gift block. Fix: `> 83`→`>= 87` |
| libs/atlas-packet/field/clientbound/warp_to_map.go:56 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.6): shares OnSetField parse; v84 no decode-opt. Fix: `> 83`→`>= 87` |
| libs/atlas-packet/field/clientbound/warp_to_map.go:60 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:69 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:80 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:85 | `Region() == "GMS" && MajorVersion() > 28` | true | true | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:96 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.6): v84 no decode-opt. Fix: `> 83`→`>= 87` |
| libs/atlas-packet/field/clientbound/warp_to_map.go:100 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:109 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:117 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/clientbound/warp_to_map.go:122 | `Region() == "GMS" && MajorVersion() > 28` | true | true | yes | unchanged (correct) | pending Phase A (map load/WarpToMap) |
| libs/atlas-packet/field/serverbound/change.go:71 | `Region() == "GMS" && MajorVersion() >= 83` | true | true | yes | unchanged (correct) | pending Phase A (map load/portal change) |
| libs/atlas-packet/field/serverbound/change.go:100 | `Region() == "GMS" && MajorVersion() >= 83` | true | true | yes | unchanged (correct) | pending Phase A (map load/portal change) |
| libs/atlas-packet/guild/clientbound/operation.go:430 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (guild operation) |
| libs/atlas-packet/guild/clientbound/operation.go:447 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (guild operation) |
| libs/atlas-packet/interaction/serverbound/operation_chat.go:32 | `(Region() == "GMS" && MajorVersion() >= 87) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/login/clientbound/auth_login_failed.go:34 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_permanent_ban.go:42 | `Region() != "GMS"` | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_permanent_ban.go:56 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_permanent_ban.go:60 | `Region() != "GMS"` | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:44 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:51 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:57 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:58 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:63 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:81 | `MajorVersion() >= 87` | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:106 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:113 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:119 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:120 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:125 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:143 | `MajorVersion() >= 87` | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_temporary_ban.go:48 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_temporary_ban.go:64 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/server_ip.go:74 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/server IP) |
| libs/atlas-packet/login/clientbound/server_ip.go:92 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/server IP) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:56 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:57 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:64 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:80 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:97 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:98 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:105 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/clientbound/server_list_entry.go:123 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/serverbound/all_character_list_request.go:56 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4 NOT individually verified (out-of-scope adjacency): same `>83` off-by-one family (§3 headline) — almost certainly should be `>= 87`; confirm in B5 |
| libs/atlas-packet/login/serverbound/all_character_list_request.go:70 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4 NOT individually verified: same `>83` off-by-one family — likely `>= 87`; confirm in B5 |
| libs/atlas-packet/login/serverbound/character_select.go:47 | `Region() == "GMS" && MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (character select) |
| libs/atlas-packet/login/serverbound/character_select_register_pic.go:58 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (PIC-PIN) |
| libs/atlas-packet/login/serverbound/character_select_register_pic.go:72 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (PIC-PIN) |
| libs/atlas-packet/login/serverbound/character_select_with_pic.go:53 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (character select with PIC) |
| libs/atlas-packet/login/serverbound/request.go:78 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (login handshake/auth request) |
| libs/atlas-packet/login/serverbound/request.go:95 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (login handshake/auth request) |
| libs/atlas-packet/login/serverbound/server_status_request.go:36 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/serverbound/world_character_list_request.go:53 | `Region() == "GMS" && MajorVersion() > 28` | true | true | yes | unchanged (correct) | pending Phase A (world/channel list) |
| libs/atlas-packet/login/serverbound/world_character_list_request.go:58 | `Region() == "GMS" && MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (world/channel list) |
| libs/atlas-packet/login/serverbound/world_character_list_request.go:70 | `Region() == "GMS" && MajorVersion() > 28` | true | true | yes | unchanged (correct) | pending Phase A (world/channel list) |
| libs/atlas-packet/login/serverbound/world_character_list_request.go:76 | `Region() == "GMS" && MajorVersion() > 12` | true | true | yes | unchanged (correct) | pending Phase A (world/channel list) |
| libs/atlas-packet/model/asset.go:195 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:208 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:213 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:243 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:257 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:261 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:344 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:374 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:412 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:416 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/attack_info.go:76 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:84 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:90 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:94 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:124 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:133 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:141 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:146 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/attack_info.go:163 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/avatar.go:50 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character spawn/avatar) |
| libs/atlas-packet/model/avatar.go:62 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character spawn/avatar) |
| libs/atlas-packet/model/avatar.go:70 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character spawn/avatar) |
| libs/atlas-packet/model/avatar.go:78 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (character spawn/avatar) |
| libs/atlas-packet/model/avatar.go:104 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character spawn/avatar) |
| libs/atlas-packet/model/avatar.go:116 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character spawn/avatar) |
| libs/atlas-packet/model/avatar.go:141 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (character spawn/avatar) |
| libs/atlas-packet/model/character_list_entry.go:59 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character list) |
| libs/atlas-packet/model/character_list_entry.go:86 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (character list) |
| libs/atlas-packet/model/character_statistics.go:98 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel) |
| libs/atlas-packet/model/character_statistics.go:113 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:135 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel) |
| libs/atlas-packet/model/character_statistics.go:142 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:143 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:150 | `MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:175 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel) |
| libs/atlas-packet/model/character_statistics.go:189 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:211 | `(Region() == "GMS" && MajorVersion() > 28) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | pending Phase A (enter-channel) |
| libs/atlas-packet/model/character_statistics.go:218 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:219 | `MajorVersion() > 12` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:226 | `MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_temporary_stat.go:105 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (ShadowPartner buff encoding for enter-channel/spawn) |
| libs/atlas-packet/model/character_temporary_stat.go:169 | `(Region() == "GMS" && MajorVersion() >= 87) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_temporary_stat.go:178 | `(Region() == "GMS" && MajorVersion() >= 95) \|\| Region() == "JMS"` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/damage_info.go:47 | `Region() == "GMS" && MajorVersion() >= 83` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/damage_taken_info.go:103 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/damage_taken_info.go:136 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/monster.go:497 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/monster.go:509 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/monster.go:512 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (monster model spawn fields) |
| libs/atlas-packet/model/monster.go:526 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/monster.go:538 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/monster.go:541 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (monster model spawn fields decode) |
| libs/atlas-packet/model/movement.go:128 | `Region() != "GMS" \|\| MajorVersion() > 83` | false (GMS and <=83) | **true** (GMS and v84 > 83) | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); A4-confirmed MISMATCH (see §3.1.8): v84 CMovePath::Decode sub_6A0FD0 NORMAL reads 5 Decode2 (no XOffset). Encode side already `> 87`. Fix: `> 83`→`> 87` |
| libs/atlas-packet/model/movement.go:217 | `Region() != "GMS" \|\| MajorVersion() > 87` | false | false | yes | unchanged (correct) | pending Phase A (movement element XOffset/YOffset encode) |
| libs/atlas-packet/monster/clientbound/movement.go:55 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (monster movement) |
| libs/atlas-packet/monster/clientbound/movement.go:62 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (monster movement) |
| libs/atlas-packet/monster/clientbound/movement.go:76 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (monster movement) |
| libs/atlas-packet/monster/clientbound/movement.go:83 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (monster movement) |
| libs/atlas-packet/monster/clientbound/spawn.go:46 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/monster/clientbound/spawn.go:63 | `(Region() == "GMS" && MajorVersion() > 12) \|\| Region() == "JMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/monster/serverbound/movement.go:70 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (monster movement SB) |
| libs/atlas-packet/monster/serverbound/movement.go:79 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (monster movement SB) |
| libs/atlas-packet/monster/serverbound/movement.go:85 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (monster movement SB) |
| libs/atlas-packet/monster/serverbound/movement.go:105 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (monster movement SB) |
| libs/atlas-packet/monster/serverbound/movement.go:114 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (monster movement SB) |
| libs/atlas-packet/monster/serverbound/movement.go:120 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (monster movement SB) |
| libs/atlas-packet/npc/clientbound/conversation.go:352 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (NPC conversation) |
| libs/atlas-packet/npc/clientbound/shop_list.go:53 | `Region() == "GMS" && MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/npc/clientbound/shop_list.go:56 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/npc/clientbound/shop_list.go:82 | `Region() == "GMS" && MajorVersion() >= 87` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/npc/clientbound/shop_list.go:85 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/npc/serverbound/shop_buy.go:40 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/npc/serverbound/shop_buy.go:53 | `Region() == "GMS"` | true | true | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/party/clientbound/invite.go:44 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (party invite) |
| libs/atlas-packet/party/clientbound/invite.go:62 | `(Region() == "GMS" && MajorVersion() > 83) \|\| Region() == "JMS"` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (party invite) |
| libs/atlas-packet/party/member_data.go:73 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/party/member_data.go:101 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/pet/serverbound/chat.go:56 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (pet chat) |
| libs/atlas-packet/pet/serverbound/chat.go:70 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (pet chat) |
| libs/atlas-packet/pet/serverbound/drop_pick_up.go:69 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (pet drop pick-up) |
| libs/atlas-packet/pet/serverbound/drop_pick_up.go:94 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | resolved (B5) | B5 resolved: >83 -> >=87 (delta §3); pending Phase A (pet drop pick-up) |
| libs/atlas-packet/socket/serverbound/channel_connect.go:61 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel) |
| libs/atlas-packet/socket/serverbound/channel_connect.go:78 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (enter-channel) |
| libs/atlas-packet/stat/clientbound/changed.go:51 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/stat/clientbound/changed.go:106 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/ui/clientbound/lock.go:33 | `Region() == "GMS" && MajorVersion() >= 90` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/ui/clientbound/lock.go:44 | `Region() == "GMS" && MajorVersion() >= 90` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-seeder/catalog.go:54 | `MajorVersion() == 0 \|\| MinorVersion() == 0` | n/a (not a boolean gate on version branches; it is a validation guard) | n/a | yes | unchanged (correct) | not a behavioral branch on v83 vs v84 |
| services/atlas-account/atlas.com/account/account/processor.go:165 | `Region() == "GMS" && MajorVersion() > 83` | false | **true** | **NO** | migrate+correct | known bug: default-gender incorrectly set to 10 (UI-choose) for v84; should be 0 for v84 until UI-choose is verified for GMS v84. Cited in MEMORY.md `processor.go > 83`. |
| services/atlas-account/atlas.com/account/account/processor.go:394 | `!a.TOS() && Region() != "JMS"` | n/a (TOS check, not a version comparison) — wait, grep matched `!=` in `!= "JMS"` | true (non-JMS GMS tenant) | true | yes | unchanged (correct) | not a MajorVersion branch; TOS is account state |
| services/atlas-channel/atlas.com/channel/main.go:378 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (channel socket init — ByteReadWriter for very old versions) |
| services/atlas-channel/atlas.com/channel/session/model.go:40 | `Region() == "GMS" && MajorVersion() <= 12` | false | false | yes | unchanged (correct) | pending Phase A (login handshake — crypto IV generator) |
| services/atlas-channel/atlas.com/channel/session/model.go:49 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (login handshake) |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:32 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:150 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:171 | `Region() == "GMS" && MajorVersion() >= 95 && itemId%10 == 3` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:185 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:191 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:237 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:304 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:332 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:345 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:352 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:360 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:367 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:374 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:384 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:394 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:400 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:408 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:414 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:421 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:428 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:435 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:442 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:449 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:456 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:463 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:470 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:477 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:484 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:489 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:494 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/init.go:27 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (channel socket init) |
| services/atlas-channel/atlas.com/channel/socket/init.go:33 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (channel socket init) |
| services/atlas-channel/atlas.com/channel/socket/model/damage_taken_info.go:66 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-channel/atlas.com/channel/socket/writer/character_attack_common.go:180 | `Region() == "GMS" && MajorVersion() >= 95` | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| services/atlas-character/atlas.com/character/character/processor.go:1336 | `Region() == "GMS" && MajorVersion() == 83` | true | **false** | **NO** | migrate+correct | known bug: auto-AP distribution (beginners lv1-10) uses `== 83` predicate; v84 falls into the `else` branch (normal AP grant). Cited in MEMORY.md `processor.go == 83`. |
| services/atlas-login/atlas.com/login/kafka/consumer/account/session/consumer.go:105 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (auth — JMS license agreement) |
| services/atlas-login/atlas.com/login/main.go:277 | `Region() == "GMS" && MajorVersion() <= 28` | false | false | yes | unchanged (correct) | pending Phase A (login socket init — ByteReadWriter for very old versions) |
| services/atlas-login/atlas.com/login/session/model.go:35 | `Region() == "GMS" && MajorVersion() <= 12` | false | false | yes | unchanged (correct) | pending Phase A (login handshake — crypto IV generator) |
| services/atlas-login/atlas.com/login/session/model.go:44 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (login handshake) |
| services/atlas-login/atlas.com/login/socket/init.go:26 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (login socket init) |
| services/atlas-login/atlas.com/login/socket/init.go:32 | `Region() == "JMS"` | false | false | yes | unchanged (correct) | pending Phase A (login socket init) |
| services/atlas-renders/atlas.com/renders/character/handler.go:65 | `urlTenant != t.Id().String() \|\| urlRegion != t.Region() \|\| ...` | n/a (string equality comparison used for request validation, not a behavioral version gate) | n/a | yes | unchanged (correct) | not a MajorVersion behavioral branch |
| services/atlas-renders/atlas.com/renders/character/handler.go:66 | `urlVersion != fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion())` | n/a (MajorVersion() used in string format for validation, not comparison operator) | n/a | yes | unchanged (correct) | not a boolean-comparison behavioral branch |
| libs/atlas-tenant/tenant.go:70 | `tenant.Region() != m.Region()` | n/a (identity comparison in Is()) | n/a | yes | unchanged (correct) | not a behavioral version gate; equality check on tenant identity |
| libs/atlas-tenant/tenant.go:73 | `tenant.MajorVersion() != m.MajorVersion()` | n/a (identity comparison in Is()) | n/a | yes | unchanged (correct) | not a behavioral version gate; equality check on tenant identity |
| libs/atlas-tenant/tenant.go:76 | `tenant.MinorVersion() != m.MinorVersion()` | n/a (identity comparison in Is()) | n/a | yes | unchanged (correct) | not a behavioral version gate; equality check on tenant identity |

**Additional rows (line numbers verified from grep output):**

| libs/atlas-packet/character/clientbound/spawn.go:142 | `Region() == "JMS"` (else-if of `t.Region() == "GMS"` block at :134) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/clientbound/spawn.go:233 | `Region() == "JMS"` (else-if of `t.Region() == "GMS"` block at :225) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:131 | `Region() == "JMS"` (JMS inventory extra block) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:143 | `Region() == "JMS"` (JMS inventory extra decode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:153 | `Region() == "GMS"` (in `> 28` block: GMS vs JMS branch) | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:156 | `Region() == "JMS"` (else-if in `> 28` block) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:190 | `Region() == "JMS"` (JMS extra field decode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:202 | `Region() == "JMS"` (JMS extra field decode 2) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:212 | `Region() == "GMS"` (in `> 28` decode block: GMS vs JMS) | true | true | yes | unchanged (correct) | pending Phase A (enter-channel/SetField) |
| libs/atlas-packet/character/data.go:215 | `Region() == "JMS"` (else-if in decode `> 28` block) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:283 | `Region() == "JMS"` (JMS extra stats encode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:350 | `Region() == "JMS"` (JMS extra stats decode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:617 | `Region() == "JMS"` (JMS quest started extra short) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/data.go:638 | `Region() == "JMS"` (JMS quest started extra short decode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/character/serverbound/create.go:121 | `Region() != "JMS"` (hairColor/skinColor gate) | true | true | yes | unchanged (correct) | pending Phase A (character create) |
| libs/atlas-packet/field/clientbound/set_field.go:53 | `Region() == "JMS"` (JMS extra fields in encode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/field/clientbound/set_field.go:99 | `Region() == "JMS"` (JMS extra fields in decode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/field/clientbound/warp_to_map.go:63 | `Region() == "JMS"` (JMS extra bytes encode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/field/clientbound/warp_to_map.go:103 | `Region() == "JMS"` (JMS extra bytes decode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/login/clientbound/auth_login_failed.go:47 | `Region() == "GMS"` (Decode branch — same predicate as :34 in grep; line offset differs between Encode/Decode) | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_permanent_ban.go:34 | `Region() == "GMS"` (Encode block — note: table had :35 which was correct per source read; actual grep line is :34) | true | true | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:84 | `Region() == "JMS"` (JMS Encode else-if block) | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/clientbound/auth_success.go:146 | `Region() == "JMS"` (JMS Decode else-if block) | false | false | yes | unchanged (correct) | pending Phase A (auth) |
| libs/atlas-packet/login/serverbound/character_select.go:59 | `Region() == "GMS" && MajorVersion() > 12` (Decode branch — same predicate as :47; :59 is in Decode) | true | true | yes | unchanged (correct) | pending Phase A (character select) |
| libs/atlas-packet/login/serverbound/character_select_with_pic.go:67 | `Region() == "GMS"` (Decode branch of :53) | true | true | yes | unchanged (correct) | pending Phase A (character select with PIC) |
| libs/atlas-packet/login/serverbound/server_status_request.go:48 | `Region() == "GMS"` (Decode else-branch — same predicate as :36; :48 is in Decode) | true | true | yes | unchanged (correct) | pending Phase A (world list) |
| libs/atlas-packet/login/serverbound/world_character_list_request.go:60 | `Region() == "JMS"` (Encode JMS branch in socketAddr block) | false | false | yes | unchanged (correct) | pending Phase A (world/channel list) |
| libs/atlas-packet/login/serverbound/world_character_list_request.go:78 | `Region() == "JMS"` (Decode JMS branch in socketAddr block) | false | false | yes | unchanged (correct) | pending Phase A (world/channel list) |
| libs/atlas-packet/model/asset.go:203 | `Region() == "JMS"` (JMS extra byte in equip encode) | false | false | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:219 | `Region() == "JMS"` (JMS extra fields in cash equip encode block) | false | false | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:252 | `Region() == "JMS"` (JMS extra byte in cash equip encode) | false | false | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:407 | `Region() == "JMS"` (JMS extra byte in equip decode) | false | false | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/asset.go:427 | `Region() == "JMS"` (JMS extra fields in equip decode block) | false | false | yes | unchanged (correct) | pending Phase A (enter-channel inventory) |
| libs/atlas-packet/model/character_statistics.go:153 | `Region() == "JMS"` (JMS extra stats in Encode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |
| libs/atlas-packet/model/character_statistics.go:229 | `Region() == "JMS"` (JMS extra stats in Decode) | false | false | yes | unchanged (correct) | no packet/behavior difference observed |

## 6. Provisioning runbook (FR-5.1) + restart sequence (OQ-6)

End-to-end, ordered, repeatable steps to bring up a **GMS v84.1** tenant
alongside the existing v83.1 tenant. A fresh operator can follow this top to
bottom. Every step is a concrete command / API call / kubectl-or-MCP action.

**Environment-specific placeholders** (substitute per cluster — these are NOT
literals):
- `<NS>` — the Kubernetes namespace the Atlas services run in (e.g. `atlas`,
  `atlas-main`, or a PR overlay namespace). Find it with
  `kubectl get pods -A | grep atlas-channel`.
- `<TENANTS_URL>` — base URL of `atlas-tenants` (in-cluster:
  `http://atlas-tenants.<NS>.svc.cluster.local` or via the ingress base URL).
- `<DATA_URL>` — base URL of `atlas-data` (in-cluster:
  `http://atlas-data.<NS>.svc.cluster.local`).
- `<WZ_BUCKET>` — MinIO bucket holding WZ archives. Default is **`atlas-wz`**
  (`MINIO_BUCKET_WZ`, default in
  `services/atlas-data/atlas.com/data/storage/minio/config.go:21`). Confirm with
  the live env of the atlas-data pod if overridden.
- `<TENANT_ID>` — the v84 tenant UUID returned by Step 3 (not known until then).

All in-cluster `curl`s assume a throwaway pod in `<NS>` (see
`reference_atlas_data_wz_inspection`):
`kubectl -n <NS> run curl --rm -it --image=curlimages/curl --restart=Never -- sh`.

---

### Step 0 — Preconditions (verify before starting)

- Build/branch containing **`template_gms_84_1.json`** (Component C) is merged
  and the image is what the cluster will deploy in Step 1.
- The v83.1 tenant is healthy (this runbook adds v84 *alongside*; it must not
  regress v83).
- You can reach `atlas-tenants` and `atlas-data` (above) and have `kubectl`
  context for `<NS>`.

---

### Step 1 — Deploy the build + seed the v84 template (FR-2.3, idempotent)

Deploy/roll the build that ships `template_gms_84_1.json` in
`services/atlas-configurations/seed-data/templates/`.

```bash
kubectl -n <NS> rollout restart deployment/atlas-configurations
kubectl -n <NS> rollout status  deployment/atlas-configurations
```

On boot the **seeder** runs automatically
(`services/atlas-configurations/atlas.com/configurations/seeder/seeder.go`):
- It is gated by env `SEED_ENABLED` (default **true**; only `SEED_ENABLED=false`
  disables it — `DefaultConfig()` lines 31-34) and reads templates from
  `SEED_DATA_PATH` (default **`/seed-data`**, line 26-29).
- `seedTemplates()` discovers **every `*.json`** under
  `<SEED_DATA_PATH>/templates/` (`discoverFiles`, lines 142-166).
- For each file it extracts `(region, majorVersion, minorVersion)` and
  **skips any that already exists** (`importTemplate` → `templateExists` →
  `GetByRegionAndVersion`, lines 201-228). The existing `(GMS,83,1)` row is
  therefore **never mutated** and `(GMS,84,1)` is inserted exactly once. Re-runs
  are idempotent (a second boot logs `skipped` for v84).

**Verify** in the atlas-configurations logs:

```bash
kubectl -n <NS> logs deployment/atlas-configurations | grep -i 'Template imported\|Template seeding complete'
```

Expect a line `"Template imported successfully"` with
`region=GMS majorVersion=84 minorVersion=1` (first deploy) and the summary
`Template seeding complete imported=… skipped=…`. On a redeploy the v84 line
moves from `imported` to a `skipped` debug log — that is correct, not an error.

---

### Step 2 — Ingest v84 WZ data + clear stale spawn cache (Component D)

WZ archives are keyed **per version** at
`<scope>/regions/GMS/versions/84.1/<archive>` (the resolution format in
`services/atlas-data/atlas.com/data/data/runwz.go:19,40` and
`.../workers/runtime.go:128`). v84 data lives at a *different* key than v83, so
ingesting it cannot disturb v83 data.

> **D1 / D2 cross-reference.** If Section 6 of this doc already carries the
> detailed D1 (upload) / D2 (verify + cache-clear) sub-steps, follow those for
> the byte-level archive list and verification set; the steps below are the
> operational spine and the integration point — do not duplicate the archive
> enumeration here.

**2a. Upload the v84 WZ archives** to the WZ bucket under the version-keyed
prefix. The canonical WZ scope is **`shared`** (version-keyed data is shared
across all tenants of that version; `ScopeKey` sentinel handling in
`.../workers/runtime.go:38-45`), so upload to:

```
<WZ_BUCKET>/shared/regions/GMS/versions/84.1/<archive>
```

e.g. `…/versions/84.1/Map.wz`, `…/Mob.wz`, `…/Npc.wz`, `…/Item.wz`,
`…/Character.wz`, etc. Use the MinIO client / console for `<WZ_BUCKET>`.

**2b. Trigger the ingest Job** via the atlas-data REST `JobCreator` path
(`POST /data/process`, route registered in
`services/atlas-data/atlas.com/data/runtime/rest/resource.go:25`; handler
`processCreate` lines 31-74). atlas-data renders a k8s Job from the
**`atlas-data-ingest-job-template`** ConfigMap (key `job.yaml`) — constants in
`.../runtime/rest/jobs.go:27,31` — injecting env `MODE=ingest`, `SCOPE`,
`REGION`, `MAJOR_VERSION`, `MINOR_VERSION`, `TENANT_ID` (`renderJob`,
`jobs.go:236-243`).

The handler reads region/version from the **tenant context headers**
(`tenant.MustFromContext`, resource.go:38). The atlas tenant headers are
`TENANT_ID` / `REGION` / `MAJOR_VERSION` / `MINOR_VERSION`
(`libs/atlas-tenant/processor.go:12-15`). To pin the ingest to `(GMS,84,1)` use
`?scope=shared` (requires operator header `X-Atlas-Operator: 1`, resource.go:43-47):

```bash
curl -sS -X POST "<DATA_URL>/data/process?scope=shared" \
  -H "X-Atlas-Operator: 1" \
  -H "TENANT_ID: <TENANT_ID>" \
  -H "REGION: GMS" \
  -H "MAJOR_VERSION: 84" \
  -H "MINOR_VERSION: 1"
```

(`TENANT_ID` may be the v84 tenant from Step 3 or any valid tenant; with
`scope=shared` the data lands under the shared, version-keyed prefix regardless
of which tenant id is on the header.) Response is `202 Accepted` with
`{"jobName": "...", "scope": "shared", "version": "84.1"}` (resource.go:66-71).

> **VERIFY AT DEPLOY:** `scope=shared` is the version-keyed path that matches the
> `shared/regions/GMS/versions/84.1/…` upload prefix in 2a. If your cluster
> instead ingests WZ under a per-tenant scope (`scope=tenant`, default), upload
> to `tenants/<TENANT_ID>/regions/GMS/versions/84.1/<archive>` in 2a to match.
> Confirm which scope your overlay's WZ layout uses before uploading.

**2c. Verify Job completion:**

```bash
curl -sS "<DATA_URL>/data/process" -H "TENANT_ID: <TENANT_ID>" -H "REGION: GMS" \
  -H "MAJOR_VERSION: 84" -H "MINOR_VERSION: 1"   # processStatus: jobs[].succeeded/active/failed
# or, directly:
kubectl -n <NS> get jobs -l atlas-data-ingest=true -L version,scope,region
kubectl -n <NS> logs job/<jobName>
```

Wait until the Job shows `succeeded=1` (label `version=84.1`). Do not proceed on
`failed>0`.

**2d. Clear the stale spawn cache** (`reference_atlas_maps_spawn_cache`). atlas-maps
caches spawn points in Redis on first init and never refreshes; without this,
83-era cached spawns mask the freshly-ingested v84 data:

```bash
# In the atlas Redis (use the project's redis access path / libs/atlas-redis key prefix):
#   DEL atlas:maps:spawn:*      (use the env-correct key prefix, e.g. "<env>:atlas:…" on PR overlays)
# Then DELETE the affected map's monsters in atlas-monsters so they respawn from v84 data.
```

---

### Step 3 — Create the v84 tenant (no schema change)

The tenant row already supports v84 by data — **no migration / schema change is
needed** (`tenant.RestModel` is just `region/majorVersion/minorVersion`,
`services/atlas-tenants/atlas.com/tenants/tenant/rest.go:4-10`; the entity stores
these as-is). Create it via **`POST /tenants`** (route
`services/atlas-tenants/atlas.com/tenants/tenant/resource.go:155`; handler
`CreateTenantHandler` lines 58-90, which calls `processor.CreateAndEmit(name,
region, major, minor)`).

The body is a **JSON:API** document; resource type is **`tenants`**
(`RestModel.GetName()`, rest.go:24-26). Attributes: `name`, `region`,
`majorVersion`, `minorVersion`.

```bash
curl -sS -X POST "<TENANTS_URL>/tenants" \
  -H "Content-Type: application/json" \
  -d '{
        "data": {
          "type": "tenants",
          "attributes": {
            "name": "GMS v84.1",
            "region": "GMS",
            "majorVersion": 84,
            "minorVersion": 1
          }
        }
      }'
```

Response is `201 Created`; capture the returned `data.id` — that is
**`<TENANT_ID>`** used elsewhere in this runbook. Creating the tenant emits a
tenant config-status event (`CreateAndEmit`), which is what the login/channel
projections consume in Step 4.

> The v84 socket handler/writer bindings come from the **template seeded in
> Step 1**, not from this call — atlas-configurations joins the new tenant to the
> `(GMS,84,1)` template and publishes the tenant's socket config on the
> configuration-tenant-status topic.

---

### Step 3.5 — Expose the v84 socket port on the load balancer (LB)

A new version needs its TCP port registered in **two places that MUST agree**:

1. **LB exposure** — `deploy/k8s/base/atlas-login.yaml` and `atlas-channel.yaml`
   (Deployment `containerPort` + the `LoadBalancer` `Service.ports`). One port per
   version by the convention `<major>×100` for login and `+1` for channel:
   gms-83 → 8300/8301, gms-87 → 8700/8701, **gms-84 → 8400/8401**. This PR adds the
   gms-84 entries (and backfills the missing gms-92/gms-95 *channel* ports
   9201/9501 — login already had 9200/9500, so those clients could log in but never
   reach a channel). Without the `Service.ports` entry an external client cannot
   reach the new socket even though the pod binds it.
2. **Bind side** — the login/channel `services` configuration in atlas-tenants
   (`Tenants:[{id, port}]` for login; nested `worlds[].channels[].port` for
   channel) must assign the v84 tenant the **same** port the LB exposes (login
   8400, channel 8401). This is what makes the service actually listen (`cfg.Port`
   → `socket.CreateSocketService(...)`, `login/main.go:304`). The projection
   `OpAdd` in Step 4 only fires once this assignment exists.

> **Ephemeral (single-version) note:** `atlas-pr-bootstrap` is single-tenant — it
> creates one canonical tenant from `REGION/MAJOR_VERSION/MINOR_VERSION` and reuses
> the port in `canonical/services/{login,channel}-service.json` (8300/8301)
> regardless of version (the bootstrap rewrites `tenants[].id` to the env tenant
> but keeps the JSON port). So a single-version v84 ephemeral binds 8300/8301 and
> reaches via the existing LB ports — no manifest change needed there. The
> 8400/8401 ports matter for the **multi-version** persistent cluster where v84
> coexists with v83/87/92/95 on distinct ports.

### Step 4 — Restart sequence so v84 socket bindings load (OQ-6)

**Mechanics (repo-verified).** Both `atlas-login` and `atlas-channel` build their
per-tenant socket listeners from a **configuration projection**:
- A `Subscriber` consumes the service-status and tenant-status config topics
  (`EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS` /
  `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`) into an in-memory `State`
  (`configuration/projection/subscriber.go`).
- An `ApplyLoop` (250 ms tick) diffs successive snapshots and emits `OpAdd` /
  `OpDrain` per `(tenant, world, channel)` key
  (`projection/loop.go`, `projection/apply.go:ComputeOps`).
- On `OpAdd` the `buildListener` `AddBody` reads **`tenantCfg.Socket.Handlers`**
  and **`tenantCfg.Socket.Writers`** live from the projection state and opens the
  socket (`channel/main.go:360-534`, esp. 391 + 533-534; `login/main.go:254-304`,
  esp. 281-282). So a **fresh tenant's** handler/writer bindings are loaded
  *when its OpAdd fires* — they are NOT baked in at pod boot.

**The known trap (`bug_new_opcodes_not_in_live_tenant_config`) is about MUTATING
an existing tenant**, not adding a new one: `ListenerConfig`
(`apply.go:27-33`) carries only IP/Port/Region/Major/Minor — **not** the
handler/writer lists. So editing an *existing* tenant's socket bindings produces
no diff → no OpAdd → no reload (the existing-tenant trap). A brand-new v84 tenant
has no prior listener, so it produces a genuine `OpAdd` and loads its bindings.

**However**, the new tenant only gets an `OpAdd` once **both** config projections
agree (`flatten` skips a tenant present in the service config but missing from the
tenant config, and vice-versa — `apply.go:90-124`) **and** the service config
assigns the v84 tenant worlds/channels/ports. Whether that assignment and the
end-offset catch-up land cleanly on already-running pods is environment-timing
dependent.

**Safe, deterministic sequence — do this:**

1. Create the tenant (Step 3) and confirm its config-status event published
   (atlas-configurations emitted it; the tenant must also be assigned
   worlds/channels/ports in the service config so a listener key exists).
2. **Restart `atlas-login` and `atlas-channel`** so each re-runs projection
   catch-up from the earliest offset (the Subscriber starts at
   `kafka.FirstOffset`, subscriber.go:74,82) with the v84 tenant already present,
   guaranteeing a clean `OpAdd`:

   ```bash
   kubectl -n <NS> rollout restart deployment/atlas-login
   kubectl -n <NS> rollout restart deployment/atlas-channel
   kubectl -n <NS> rollout status  deployment/atlas-login
   kubectl -n <NS> rollout status  deployment/atlas-channel
   ```

3. **Verify** the listeners came up for the v84 tenant:

   ```bash
   kubectl -n <NS> logs deployment/atlas-login   | grep -i 'projection.caughtup\|projection.applied\|<TENANT_ID>'
   kubectl -n <NS> logs deployment/atlas-channel | grep -i 'projection.caughtup\|projection.applied\|<TENANT_ID>'
   ```

   Expect `projection.caughtup` then `projection.applied … op=add` for the v84
   tenant key. (MCP equivalent: `mcp__kubernetes__pods_log` on the login/channel
   pods, or `mcp__grafana__query_loki_logs` filtering `projection.applied`.)

> **VERIFY AT DEPLOY (restart necessity):** the projection is *designed* to add a
> new tenant live (no restart) once both config topics carry it. In practice the
> restart above is the guaranteed-correct path because it forces a clean catch-up
> with the tenant already present and side-steps any timing gap between the two
> config-status streams. If the live add is observed (`op=add` for the v84 key
> appears in the logs without a restart), the restart is redundant — but keep it
> as the documented default until live behavior is confirmed in your cluster.
> Other services do **not** need restarts: only login/channel terminate the
> client socket and are driven by the socket template.

---

### Step 5 — Connect a v84 client

1. Point a **GMS v84.1** client at the login server address/port that the v84
   tenant's service config assigned (the `IPAddress`/`Port` in the listener
   `OpAdd` from Step 4; cross-check the login logs).
2. Proceed through login → world/channel select → character create/select →
   in-game. Watch for the canonical failure signature **`unhandled message op
   0xXX` at info** in the login/channel logs (`reference_observability`) — any
   such opcode is a first suspect against the Section 1/2 maps and the OQ-7
   low-confidence rows.

This is the entry point to the **E2 live playthrough** (forward reference): the
basic-playthrough acceptance checklist (login, movement, chat, map change,
channel change, party, basic combat) is exercised there; the v83 regression pass
runs in parallel against the untouched v83 tenant.
