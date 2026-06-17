# task-096 — Codec inventory (Task 0.2)

What already exists in `libs/atlas-packet/field` and `libs/atlas-packet/chat`,
and which of the 75 CField work-list ops (from `cfield-ops.md`) a pre-existing
codec plausibly serves.

## Existing field/chat codecs

`find libs/atlas-packet/field libs/atlas-packet/chat -name '*.go' -not -name '*_test.go'`.

`marker?` = the sibling `_test.go` contains a `// packet-audit:verify` marker.
`evidence?` = a record exists under `docs/packets/evidence/*/{field,chat}.*`.

| File | Struct(s) | NAME constant value | dir | marker? | evidence? |
|------|-----------|---------------------|-----|---------|-----------|
| chat/clientbound/general.go | GeneralChat | `CharacterChatGeneral` (GeneralChatWriter) | CB | yes | no |
| chat/clientbound/multi.go | MultiChat | `CharacterMultiChat` (MultiChatWriter) | CB | no | no |
| chat/clientbound/whisper.go | WhisperSendResult, WhisperReceive, WhisperFindResultCashShop, WhisperFindResultMap, WhisperFindResultChannel, WhisperFindResultError, WhisperError, WhisperWeather | `CharacterChatWhisper` (WhisperWriter) | CB | no | no |
| chat/clientbound/world_message.go | WorldMessageSimple, WorldMessageTopScroll, WorldMessageSuperMegaphone, WorldMessageBlueText, WorldMessageItemMegaphone, WorldMessageYellowMegaphone, WorldMessageMultiMegaphone, WorldMessageGachapon | `WorldMessage` (WorldMessageWriter) | CB | no | no |
| chat/clientbound/world_message_extra.go | WorldMessageUnknown3, WorldMessageUnknown7, WorldMessageUnknown8, WorldMessageWeather | `WorldMessage` (WorldMessageWriter) | CB | no | no |
| chat/serverbound/general.go | General | `CharacterChatGeneralHandle` | SB | yes (5) | yes (gms_v84 chat.serverbound.ChatGeneral) |
| chat/serverbound/multi.go | Multi | `CharacterMultiChatHandle` (CharacterChatMultiHandle) | SB | no | no |
| chat/serverbound/whisper.go | Whisper | `CharacterChatWhisperHandle` | SB | no | no |
| field/clientbound/affected_area_created.go | AffectedAreaCreated | `AffectedAreaCreated` (AffectedAreaCreatedWriter) | CB | yes (affected_area_test.go) | no (only AffectedAreaRemoved pinned) |
| field/clientbound/affected_area_removed.go | AffectedAreaRemoved | `AffectedAreaRemoved` (AffectedAreaRemovedWriter) | CB | yes | yes (all 5 versions) |
| field/clientbound/clock.go | Clock | `Clock` (ClockWriter) | CB | yes | yes (all 5 versions) |
| field/clientbound/effect.go | EffectSummon, EffectTremble, EffectString, EffectBossHp, EffectRewardRullet | `FieldEffect` (FieldEffectWriter) | CB | yes (25) | yes (all 5 versions, per-struct) |
| field/clientbound/effect_weather.go | EffectWeather | `FieldEffectWeather` (FieldEffectWeatherWriter) | CB | yes | yes (all 5 versions) |
| field/clientbound/kite_destroy.go | KiteDestroy | `DestroyKite` (KiteDestroyWriter) | CB | yes | yes (all 5 versions) |
| field/clientbound/kite_error.go | KiteError | `SpawnKiteError` (KiteErrorWriter) | CB | yes | yes (all 5 versions) |
| field/clientbound/kite_spawn.go | KiteSpawn | `SpawnKite` (KiteSpawnWriter) | CB | yes | yes (all 5 versions) |
| field/clientbound/set_field.go | SetField | `SetField` (SetFieldWriter) | CB | yes | yes (all 5 versions) |
| field/clientbound/transport.go | Transport | `FieldTransportState` (FieldTransportStateWriter) | CB | yes | yes (all 5 versions) |
| field/clientbound/warp_to_map.go | WarpToMap | `SetField` (SetFieldWriter) | CB | yes (warp_to_map_test.go) | no (shares SetField op; FieldWarpToMap pinned all 5) |
| field/field_effect_body.go | (no struct/Operation — shared body helper) | — | — | n/a | n/a |
| field/serverbound/change.go | Change | `MapChangeHandle` | SB | yes (5) | yes (all 5 versions) |

Total existing field/chat codec source files (non-test): **22**.

> None of these existing codecs (other than GENERAL_CHAT and GUILD_OPERATION,
> see below) correspond to a CField work-list op. They cover field-effect /
> clock / kite / affected-area / set-field / transport / warp and chat
> general/multi/whisper/world-message ops, which are NOT in the 75-op work-list
> (the work-list ops all have an empty Packet column in STATUS.md except the two
> `PKT`-flagged rows).

## 75 work-list ops → candidate existing codec

`dir`: CB=clientbound, SB=serverbound, ?=unresolved (per cfield-ops.md).
`marker?`/`evidence?` apply to the candidate codec where one exists.

| op | dir | candidate existing codec | marker? | evidence? | notes |
|----|-----|--------------------------|---------|-----------|-------|
| ADMIN_CHAT | SB | none | — | — | |
| ADMIN_COMMAND | SB | none | — | — | |
| ADMIN_LOG | SB | none | — | — | |
| ADMIN_RESULT | CB | none | — | — | |
| ARIANT_RESULT | CB | none | — | — | jms n/a |
| BLOCKED_MAP | CB | none | — | — | |
| BLOCKED_SERVER | CB | none | — | — | |
| FIELD_OBSTACLE_ALL_RESET | CB | none | — | — | |
| FIELD_OBSTACLE_ONOFF | CB | none | — | — | |
| FIELD_OBSTACLE_ONOFF_LIST | CB | none | — | — | |
| FOOTHOLD_INFO | CB | none | — | — | |
| FORCED_MAP_EQUIP | CB | none | — | — | |
| GENERAL_CHAT | SB | chat/serverbound/general.go (General) | yes | yes | **PKT**; STATUS maps to chat/serverbound/ChatGeneral (T1); v84/87/95/jms ✅, only v83 ❌ |
| GMEVENT_INSTRUCTIONS | CB | none | — | — | |
| GUILD_OPERATION | ? | guild/serverbound/GuildOperation (T1) — OUTSIDE field/chat | n/a | n/a | **PKT**; lives in guild family, not atlas-packet/field; v95/jms ✅, v83/84/87 ❌. Verify, don't duplicate |
| HORNTAIL_CAVE | CB | none | — | — | |
| IDA_0X098 | CB | none | — | — | OnStalkResult |
| IDA_0X09C | CB | none | — | — | OnFootHoldInfo/OnStalkResult |
| IDA_0X09D | CB | none | — | — | OnRequestFootHoldInfo |
| IDA_0X0A4 | CB | none | — | — | |
| IDA_0X0AA | CB | none | — | — | |
| IDA_0X0AC | CB | none | — | — | |
| IDA_0X0B0 | CB | none | — | — | |
| IDA_0X0B1 | CB | none | — | — | |
| IDA_0X169 | CB | none | — | — | OnHontailTimer |
| MATCH_TABLE | SB | none | — | — | |
| MTS_OPERATION | CB | none | — | — | jms n/a |
| MTS_OPERATION2 | CB | none | — | — | jms n/a |
| MULTICHAT | CB | none | — | — | NOTE: this is CField::OnGroupMessage (a field group-message clientbound), distinct from chat/clientbound/multi.go (CharacterMultiChat). multi.go is NOT a match |
| OX_QUIZ | CB | none | — | — | |
| PLAY_JUKEBOX | CB | none | — | — | |
| SET_OBJECT_STATE | CB | none | — | — | |
| SET_QUEST_CLEAR | CB | none | — | — | |
| SET_QUEST_TIME | CB | none | — | — | |
| SLIDE_REQUEST | SB | none | — | — | v95/jms only |
| SPOUSE_CHAT | CB | none | — | — | jms n/a |
| STOP_CLOCK | CB | none | — | — | CField::OnDestroyClock — distinct from field/clientbound/clock.go (Clock, CB destroy is a separate op) |
| SUE_CHARACTER | SB | none | — | — | jms n/a |
| SUMMON_ITEM_INAVAILABLE | CB | none | — | — | |
| USE_DOOR | ? | none | — | — | TryEnterTownPortal |
| VICIOUS_HAMMER | CB | none | — | — | jms n/a |
| WHISPER (a) | CB | none | — | — | CField::OnWhisper — distinct from chat/clientbound/whisper.go (CharacterChatWhisper). Whisper codecs are chat-family, not this CField op |
| WHISPER (b) | CB | none | — | — | same FName cluster |
| WITCH_TOWER_SCORE_UPDATE | CB | none | — | — | |
| ZAKUM_SHRINE | CB | none | — | — | |
| HIT_SNOWBALL | CB | none | — | — | |
| LEFT_KNOCKBACK | ? | none | — | — | |
| LEFT_KNOCK_BACK | CB | none | — | — | |
| SNOWBALL | ? | none | — | — | |
| SNOWBALL_MESSAGE | CB | none | — | — | |
| SNOWBALL_STATE | CB | none | — | — | |
| TOURNAMENT | CB | none | — | — | |
| TOURNAMENT_CHARACTERS | CB | none | — | — | |
| TOURNAMENT_MATCH_TABLE | CB | none | — | — | |
| TOURNAMENT_SET_PRIZE | CB | none | — | — | |
| TOURNAMENT_UEW | CB | none | — | — | |
| WEDDING_ACTION | CB | none | — | — | jms n/a |
| WEDDING_CEREMONY_END | CB | none | — | — | |
| WEDDING_PROGRESS | CB | none | — | — | |
| WEDDING_TALK | CB | none | — | — | jms n/a |
| COCONUT | ? | none | — | — | |
| COCONUT_HIT | CB | none | — | — | |
| COCONUT_SCORE | CB | none | — | — | |
| GUILD_BOSS | ? | none | — | — | |
| GUILD_BOSS_HEALER_MOVE | CB | none | — | — | |
| GUILD_BOSS_PULLEY_STATE_CHANGE | CB | none | — | — | |
| CONTI_MOVE (Init) | ? | none | — | — | v95 only |
| CONTI_MOVE (OnContiMove) | CB | none | — | — | |
| ARIANT_ARENA_SHOW_RESULT | CB | none | — | — | |
| ARIANT_ARENA_USER_SCORE | CB | none | — | — | |
| SHEEP_RANCH_CLOTHES | CB | none | — | — | |
| SHEEP_RANCH_INFO | CB | none | — | — | |
| PYRAMID_GAUGE | CB | none | — | — | |
| PYRAMID_SCORE | CB | none | — | — | |
| ARIANT_SCORE | CB | none | — | — | v95 only |

## Summary

- **A-row candidates** (op has a plausible existing codec): **2** — GENERAL_CHAT
  (chat/serverbound/general.go, already verified v84–jms; v83 to do) and
  GUILD_OPERATION (guild/serverbound/GuildOperation, T1, in the guild family —
  not atlas-packet/field; already verified v95/jms).
- **B-row candidates** (no existing codec — net-new codec work): **73**.
- Both A-row ops are the two `PKT`-flagged entries in cfield-ops.md, confirming
  that flag.
- Near-miss codecs explicitly ruled OUT (similar name, different op): chat
  whisper/multi/world-message clientbound (chat family, not the CField
  OnWhisper/OnGroupMessage ops); field clock.go (not the CField OnDestroyClock /
  STOP_CLOCK op). Per the matrix-redx memory note, before treating any near-miss
  as a duplicate, confirm by COutPacket opcode rather than IDB name.
