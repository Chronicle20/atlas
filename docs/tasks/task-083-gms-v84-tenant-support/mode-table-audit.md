# v84 config mode/sub-op table audit

## Why
The v84 template (`template_gms_84_1.json`) was authored by **copying all 43
`socket.*.options` code/mode tables verbatim from the v83 template** (42/43 were
byte-identical before this audit). These tables map symbolic names → a sub-op /
mode **byte** that the client dispatches on. They are version-sensitive client
enums; wherever the enum shifted v83→v84, the copy is wrong.

The v87/v95 templates can't be used as a cross-reference — their mode tables are
**empty/incomplete** (v95 population is in progress). The **v84 client binary is
the only ground truth**. Method: extract the client dispatcher's switch-case set
and confirm every config value is a member; a config value absent from the set is
mis-dispatched.

## Risk model (why cash shop booted but most don't)
- **Dangerous:** a wrong value collides with an *active* client handler that
  disconnects/transfers. Cash shop's wishlist mode `0x4F` hit `LoadLockerFailed →
  NoticeFailReason + SendTransferFieldPacket` = boot.
- **Benign:** a wrong value hits the switch **default** → generic message / no-op.
  Most social/dialog handlers behave this way (party/guild/storage/npc-say).

## Stable (safe — identical across v83/v87/v95, not version-sensitive)
AddCharacterEntry, AuthLoginFailed, CharacterNameResponse, CharacterViewAll,
DeleteCharacterResponse, PinOperation, PinUpdate, ServerIP[codes], ServerIP[modes].

## Results (verified against v84 client)
| table | client dispatcher (v84) | verdict |
|---|---|---|
| **CashShopOperation** [operations] | `CCashShop::OnCashItemResult` @0x47C291 | **WRONG (+3) → FIXED** |
| CharacterStatusMessage | `CWvsContext::OnMessage` @0xA6BDD9 | OK (0..13 ⊆ 0..14) |
| BuddyOperation | `CWvsContext::OnFriendResult` @0xA8ADA2 | OK (⊆ 0,7..22) |
| MessengerOperation | `CUIMessenger::OnPacket` @0x87CBD8 | OK (0..7 ⊆ 0..8) |
| HiredMerchantOperation | `CWvsContext::OnEntrustedShopCheckResult` @0xA73538 | OK (⊆ 6..18,253..255) |
| NPCShopOperation | `CShopDlg::OnPacket` @0x77905B | OK (exact) |
| StorageOperation | `CTrunkDlg::OnPacket` @0x7EEC1A | OK except `ERROR_MESSAGE=24` → default (benign) |
| NoteOperation [operations] | `CWvsContext::OnMemoResult` @0xA70785 | OK (3,4,5,7 ⊆ v95 0,3..7) |
| PartyOperation | `CWvsContext::OnPartyResult` @0xA89CF3 | core 4..40 handled; extras 19,22,23,27,28,29 → default (benign) |
| GuildOperation | `CWvsContext::OnGuildResult` @0xA82E2B | large dense range; core handled; not exhaustively value-checked |
| NPCConversation [messageType] | `CScriptMan::OnScriptMessage` @0x76850A | core 0..10,4=ASK_MENU,13,14 OK (v84 follows v83 enum, NOT v95); `ASK_YES_NO_QUEST=12` → v84 default (v84 has 11/15, not 12) — quest yes/no dialog won't render |

## Round 2 — additional outbound writers verified against v84 client
| table | client dispatcher (v84) | verdict |
|---|---|---|
| CharacterEffect | `CUserLocal::OnEffect` @0x96EA92 | OK (atlas 0..26 ⊆ client 0..27,29) |
| CharacterEffectForeign | (shares CharacterEffect effect-type enum) | OK (inferred) |
| FieldEffect | `CField::OnFieldEffect` @0x53F37D | OK (0..7 exact) |
| UiOpen | `CWvsContext::UI_Open` @0xA4FD3E | OK (exact: 0,1,2,3,5,6,9,10,11,18,20,22,23,26..33) |
| WorldMessage | `CWvsContext::OnBroadcastMsg` @0xA6DC97 | core 0..15 OK; config 16,17,18 → v84 default (benign; newer broadcast types) |
| FameResponse | `CWvsContext::OnFameResult` @0xA6D8EE | OK (0..5 specific, 6 → generic msg) |

**Net outbound-writer result: cash shop was the ONLY broken table (fixed).** Every
other outbound writer is either exact or degrades stray config values to a benign
default — no other boot/crash path found.

## Still NOT verified (lowest risk)
- **PetActivated** (0xAB) [0..4] — `CUserPool::OnPetActionPacket` @0x97015C; pet
  action sub-modes; mismatch → pet effect no-op (no boot).
- **CharacterInteraction** (0x141) [operations] + [enterError] — `CMiniRoomBaseDlg::
  OnPacketBase` @0x673DB5; base trade ops (2,3,6,9,10) + per-room vtable + enter-
  result error code; complex, needs per-room decompile.
- **Auth{Temporary,Permanent}Ban** [failedReasonCodes] (0x00) — login ban-reason
  strings; login-stage, rare.

Inbound request sub-ops (client→server; wrong value = atlas mis-reads request, **no
client crash**): CashShopOperationHandle, BuddyOperationHandle, GuildOperationHandle,
MessengerOperationHandle, CharacterInteractionHandle, NPCShopHandle,
StorageOperationHandle, NoteOperationHandle, PartyOperationHandle, GuildBBSHandle,
NPCContinueConversationHandle[messageType].

## v84 IDB names added (CWvsContext::OnPacket recv handlers + dialogs)
OnMessage 0x27, OnFriendResult 0x41, OnEntrustedShopCheckResult 0x32,
OnMemoResult 0x29, OnBroadcastMsg 0x46, OnPartyResult 0x3E, OnGuildResult 0x43,
OnFameResult 0x26; CScriptMan OnScriptMessage 0x137; (earlier) CShopDlg/CTrunkDlg/
CUIMessenger/CMiniRoomBaseDlg/CCashShop recv handlers.
