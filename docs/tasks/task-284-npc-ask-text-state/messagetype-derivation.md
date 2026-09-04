# `messageType` table derivation — gms_87_1, gms_92_1, jms_185_1

Derivation performed by the controller via `ida-pro-mcp` on 2026-08-28, decompiling
`CScriptMan::OnScriptMessage` in each of the three client binaries below. This
implementer (task-12) did not run IDA and transcribed these tables verbatim from the
controller's decompilation, as instructed in the task-12 dispatch.

`OnScriptMessage` is the switch on the wire `lastMessageType` byte; each `case` calls
the corresponding `CScriptMan::OnAsk*` (or `OnSay*`) handler. The byte→name mapping
below is what `NPCContinueConversationHandle`'s `options.messageType` table encodes
per tenant template, keyed by the `NpcConversationMessageType` constants at
`libs/atlas-packet/npc/clientbound/conversation.go:19-34`.

## gms_87_1 — `GMSv87_4GB.exe`

- md5: `2e692f3ab5078e04138d264f8ea1e668`
- `CScriptMan::OnScriptMessage` @ `0x791666`

| byte | handler | addr |
|---|---|---|
| 0 | OnSay | 0x791828 |
| 1 | OnSayImage | 0x7919a9 |
| 2 | OnAskYesNo (bQuest=0) | 0x791b70 |
| 3 | OnAskText | 0x791cd0 |
| 4 | OnAskNumber | 0x792020 |
| 5 | OnAskMenu | 0x7921a8 |
| 6 | OnAskQuiz | 0x792b90 |
| 7 | OnAskSpeedQuiz | 0x792ba2 |
| 8 | OnAskAvatar | 0x792330 |
| 9 | OnAskMembershopAvatar | 0x7924cc |
| 0xA (10) | OnAskPet | 0x792663 |
| 0xB (11) | OnAskPetAll | 0x7928f1 |
| 0xD (13) | OnAskYesNo (bQuest=1) | 0x791b70 |
| 0xE (14) | OnAskBoxText | 0x791e79 |
| 0xF (15) | OnAskSlideMenu | 0x792bb4 |

No case 12. Applied to `template_gms_87_1.json` handler entry at
`/socket/handlers/44` (opCode `0x3F`, `NPCContinueConversationHandle`).

## gms_92_1 — `GMS_v92_1_DEVM.exe`

- md5: `bdef16653b92eefca2361fd5668cc509`
- `CScriptMan::OnScriptMessage` @ `0x6d1650`

| byte | handler | addr |
|---|---|---|
| 0 | OnSay | 0x6cf670 |
| 1 | OnSayImage | 0x6cf870 |
| 2 | OnAskYesNo (bQuest=0) | 0x6cfb00 |
| 3 | OnAskText | 0x6cfcf0 |
| 4 | OnAskNumber | 0x6d0160 |
| 5 | OnAskMenu | 0x6d0360 |
| 6 | OnAskQuiz | 0x6cf280 |
| 7 | OnAskSpeedQuiz | 0x6cf2a0 |
| 8 | OnAskAvatar | 0x6d0550 |
| 9 | OnAskMembershopAvatar | 0x6d08a0 |
| 10 | OnAskPet | 0x6d0c40 |
| 11 | OnAskPetAll | 0x6d1140 |
| 13 | OnAskYesNo (bQuest=1) | 0x6cfb00 |
| 14 | OnAskBoxText | 0x6cff20 |
| 15 | OnAskSlideMenu | 0x6cf480 |

No case 12. This is the same byte numbering as gms_87_1/gms_95_1 — a derived result
from independently decompiling the v92 switch, not an assumption carried over from
another version.

Applied to a new `template_gms_92_1.json` handler entry (this template had no
`NPCContinueConversationHandle` entry before task-12), inserted at its ordered
position between opCode `0x3B` (`CharacterUseDeathItemHandle`) and `0x43`
(`NPCShopHandle`).

opCode `0x42` for `NPC_TALK_MORE` (serverbound) is confirmed against
`docs/packets/registry/gms_v92.yaml:2495-2498` (`opcode: 66` = `0x42`), matching the
`docs/packets/audits/STATUS.md:571` opcode-table row (that row's v92 cell was
flagged `❌` for codec verification, not for the opcode value itself — the opcode is
independently confirmed here against the packet registry).

## jms_185_1 — `MapleStory_dump_SCY.exe` (JMS v185.1)

- md5: `af6652ff9b7c549341f35e3569d7564a`
- `CScriptMan::OnScriptMessage` @ `0x7b7160`

**Different numbering from GMS** — there is no `OnAskMembershopAvatar` arm, and
every arm from byte 9 up is shifted down by one relative to gms_87_1/gms_92_1.

| byte | handler | addr |
|---|---|---|
| 0 | OnSay | 0x7b7315 |
| 1 | OnSayImage | 0x7b7496 |
| 2 | OnAskYesNo (bQuest=0) | 0x7b765d |
| 3 | OnAskText | 0x7b77bd |
| 4 | OnAskNumber | 0x7b7b0d |
| 5 | OnAskMenu | 0x7b7c95 |
| 6 | OnAskQuiz | 0x7b84ef |
| 7 | OnAskSpeedQuiz | 0x7b8501 |
| 8 **and 16** | OnAskAvatar | 0x7b7e1d |
| 9 | OnAskPet | 0x7b7fc2 |
| 10 | OnAskPetAll | 0x7b8250 |
| 12 | OnAskYesNo (bQuest=1) | 0x7b765d |
| 13 | OnAskBoxText | 0x7b7966 |
| 14 | OnAskSlideMenu | 0x7b8513 |

No case 11, no case 15, no `OnAskMembershopAvatar`. Case 16 is a second switch arm
that also falls through to `OnAskAvatar` (an alias of byte 8, not a distinct message
type); since the config table is name→byte, only the canonical `ASK_AVATAR: 8` is
emitted and the byte-16 alias is recorded here for audit purposes only.
`ASK_MEMBER_SHOP_AVATAR` is omitted entirely — 14 entries, not 15.

Applied to `template_jms_185_1.json` handler entry at `/socket/handlers/32`
(opCode `0x34`, `NPCContinueConversationHandle`).
