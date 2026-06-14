# task-096 — Matrix baseline (Task 0.1)

Captured before any task-096 codec work. This is the burndown reference: every
CField op below is `❌` (or `⬜` n/a) for the listed versions today; promotions to
`✅` during task-096 are measured against this snapshot.

## `matrix --check` result

Command (from worktree root):

```
go run ./tools/packet-audit matrix --check; echo "exit=$?"
```

Result: **exit=0** (clean). No pre-existing matrix failures to record.

> If a later `matrix --check` run fails, diff against this clean baseline — any
> failure introduced during task-096 is attributable to task-096.

## Per-op × per-version baseline state

States from `docs/packets/audits/STATUS.md` at baseline. Legend: ✅ verified ·
🟡 partial · ❌ incomplete · ⬜ n/a · 🟥 conflict.

The 75 ops are enumerated in `cfield-ops.md`. Where an Op name appears in
multiple STATUS.md rows, the row matching the cfield-ops.md FName is used; any
collision is noted.

### CField (45 ops)

| Op | FName | v83 | v84 | v87 | v95 | jms_v185 |
|----|-------|-----|-----|-----|-----|----------|
| ADMIN_CHAT | CField::SendChatMsgSlash | ❌ | ❌ | ❌ | ❌ | ❌ |
| ADMIN_COMMAND | CField::SendChatMsgSlash | ❌ | ❌ | ❌ | ❌ | ❌ |
| ADMIN_LOG | CField::SendChatMsgSlash | ❌ | ❌ | ❌ | ❌ | ❌ |
| ADMIN_RESULT | CField::OnAdminResult | ❌ | ❌ | ❌ | ❌ | ❌ |
| ARIANT_RESULT | CField::OnWarnMessage | ❌ | ❌ | ❌ | ❌ | ⬜ |
| BLOCKED_MAP | CField::OnTransferFieldReqIgnored | ❌ | ❌ | ❌ | ❌ | ❌ |
| BLOCKED_SERVER | CField::OnTransferChannelReqIgnored | ❌ | ❌ | ❌ | ❌ | ❌ |
| FIELD_OBSTACLE_ALL_RESET | CField::OnFieldObstacleAllRese(t) | ❌ | ❌ | ❌ | ❌ | ❌ |
| FIELD_OBSTACLE_ONOFF | CField::OnFieldObstacleOnOff | ❌ | ❌ | ❌ | ❌ | ❌ |
| FIELD_OBSTACLE_ONOFF_LIST | CField::OnFieldObstacleOnOffStatus | ❌ | ❌ | ❌ | ❌ | ❌ |
| FOOTHOLD_INFO | CField::OnRequestFootHoldInfo | ❌ | ❌ | ❌ | ❌ | ❌ |
| FORCED_MAP_EQUIP | CField::OnFieldSpecificData | ❌ | ❌ | ❌ | ❌ | ❌ |
| GENERAL_CHAT (PKT) | CField::SendChatMsg | ❌ | ✅ | ✅ | ✅ | ✅ |
| GMEVENT_INSTRUCTIONS | CField::OnDesc | ❌ | ❌ | ❌ | ❌ | ❌ |
| GUILD_OPERATION (PKT) | CField::InputGuildName (guild/serverbound/GuildOperation) | ❌ | ❌ | ❌ | ✅ | ✅ |
| HORNTAIL_CAVE | CField::OnHontailTimer | ❌ | ❌ | ❌ | ❌ | ❌ |
| IDA_0X098 | CField::OnStalkResult | ⬜ | ⬜ | ⬜ | ⬜ | ❌ |
| IDA_0X09C | CField::OnFootHoldInfo;OnStalkResult | ❌ | ⬜ | ⬜ | ⬜ | ❌ |
| IDA_0X09D | CField::OnRequestFootHoldInfo | ⬜ | ⬜ | ⬜ | ⬜ | ❌ |
| IDA_0X0A4 | CField::OnStalkResult | ⬜ | ⬜ | ❌ | ⬜ | ⬜ |
| IDA_0X0AA | CField::OnFootHoldInfo | ⬜ | ⬜ | ❌ | ⬜ | ❌ |
| IDA_0X0AC | CField::OnStalkResult | ⬜ | ⬜ | ⬜ | ❌ | ❌ |
| IDA_0X0B0 | CField::OnFootHoldInfo | ⬜ | ⬜ | ⬜ | ❌ | ⬜ |
| IDA_0X0B1 | CField::OnRequestFootHoldInfo | ⬜ | ⬜ | ⬜ | ❌ | ⬜ |
| IDA_0X169 | CField::OnHontailTimer | ⬜ | ⬜ | ⬜ | ❌ | ❌ |
| MATCH_TABLE | CField::SendChatMsgSlash | ❌ | ❌ | ❌ | ❌ | ❌ |
| MTS_OPERATION | CField::OnCharacterSale | ❌ | ❌ | ❌ | ❌ | ⬜ |
| MTS_OPERATION2 | CField::OnCharacterSale | ❌ | ❌ | ❌ | ❌ | ⬜ |
| MULTICHAT | CField::OnGroupMessage | ❌ | ❌ | ❌ | ❌ | ❌ |
| OX_QUIZ | CField::OnQuiz | ❌ | ❌ | ❌ | ❌ | ❌ |
| PLAY_JUKEBOX | CField::OnPlayJukeBox | ❌ | ❌ | ❌ | ❌ | ❌ |
| SET_OBJECT_STATE | CField::OnSetObjectState | ❌ | ❌ | ❌ | ❌ | ❌ |
| SET_QUEST_CLEAR | CField::OnSetQuestClear | ❌ | ❌ | ❌ | ❌ | ❌ |
| SET_QUEST_TIME | CField::OnSetQuestTime | ❌ | ❌ | ❌ | ❌ | ❌ |
| SLIDE_REQUEST | CField::SendChatMsgSlash | ⬜ | ⬜ | ⬜ | ❌ | ❌ |
| SPOUSE_CHAT | CField::OnCoupleMessage | ❌ | ❌ | ❌ | ❌ | ⬜ |
| STOP_CLOCK | CField::OnDestroyClock | ❌ | ❌ | ❌ | ❌ | ❌ |
| SUE_CHARACTER | CField::SendChatMsgSlash | ❌ | ❌ | ❌ | ❌ | ⬜ |
| SUMMON_ITEM_INAVAILABLE | CField::OnSummonItemInavailable | ❌ | ❌ | ❌ | ❌ | ❌ |
| USE_DOOR | CField::TryEnterTownPortal | ❌ | ❌ | ❌ | ❌ | ❌ |
| VICIOUS_HAMMER | CField::OnItemUpgrade | ❌ | ❌ | ❌ | ❌ | ⬜ |
| WHISPER (a) | CField::OnWhisper | ❌ | ❌ | ❌ | ❌ | ❌ |
| WHISPER (b) | CField::OnWhisper;SendChatMsgWhisper;SendLocationWhisper | ❌ | ❌ | ❌ | ❌ | ❌ |
| WITCH_TOWER_SCORE_UPDATE | CField::OnChaosZakumTimer | ❌ | ❌ | ❌ | ❌ | ❌ |
| ZAKUM_SHRINE | CField::OnZakumTimer | ❌ | ❌ | ❌ | ❌ | ❌ |

### CField_SnowBall (6 ops)

| Op | FName | v83 | v84 | v87 | v95 | jms_v185 |
|----|-------|-----|-----|-----|-----|----------|
| HIT_SNOWBALL | CField_SnowBall::OnSnowBallHit | ❌ | ❌ | ❌ | ❌ | ❌ |
| LEFT_KNOCKBACK | CField_SnowBall::Update | ❌ | ❌ | ❌ | ❌ | ❌ |
| LEFT_KNOCK_BACK | CField_SnowBall::OnSnowBallTouch | ❌ | ❌ | ❌ | ❌ | ❌ |
| SNOWBALL | CField_SnowBall::BasicActionAttack | ❌ | ❌ | ❌ | ❌ | ❌ |
| SNOWBALL_MESSAGE | CField_SnowBall::OnSnowBallMsg | ❌ | ❌ | ❌ | ❌ | ❌ |
| SNOWBALL_STATE | CField_SnowBall::OnSnowBallState | ❌ | ❌ | ❌ | ❌ | ❌ |

### CField_Tournament (5 ops)

| Op | FName | v83 | v84 | v87 | v95 | jms_v185 |
|----|-------|-----|-----|-----|-----|----------|
| TOURNAMENT | CField_Tournament::OnTournament | ❌ | ❌ | ❌ | ❌ | ❌ |
| TOURNAMENT_CHARACTERS | CField_Tournament::OnPacket | ❌ | ❌ | ❌ | ❌ | ❌ |
| TOURNAMENT_MATCH_TABLE | CField_Tournament::OnTournamentMatchTable | ❌ | ❌ | ❌ | ❌ | ❌ |
| TOURNAMENT_SET_PRIZE | CField_Tournament::OnTournamentSetPrize | ❌ | ❌ | ❌ | ❌ | ❌ |
| TOURNAMENT_UEW | CField_Tournament::OnTournamentUEW | ❌ | ❌ | ❌ | ❌ | ❌ |

### CField_Wedding (4 ops)

| Op | FName | v83 | v84 | v87 | v95 | jms_v185 |
|----|-------|-----|-----|-----|-----|----------|
| WEDDING_ACTION | CField_Wedding::OnWeddingProgress | ❌ | ❌ | ❌ | ❌ | ⬜ |
| WEDDING_CEREMONY_END | CField_Wedding::OnWeddingCeremonyEnd | ❌ | ❌ | ❌ | ❌ | ❌ |
| WEDDING_PROGRESS | CField_Wedding::OnWeddingProgress | ❌ | ❌ | ❌ | ❌ | ❌ |
| WEDDING_TALK | CField_Wedding::OnWeddingProgress | ❌ | ❌ | ❌ | ❌ | ⬜ |

### CField_Coconut (3 ops)

| Op | FName | v83 | v84 | v87 | v95 | jms_v185 |
|----|-------|-----|-----|-----|-----|----------|
| COCONUT | CField_Coconut::BasicActionAttack | ❌ | ❌ | ❌ | ❌ | ❌ |
| COCONUT_HIT | CField_Coconut::OnCoconutHit | ❌ | ❌ | ❌ | ❌ | ❌ |
| COCONUT_SCORE | CField_Coconut::OnCoconutScore | ❌ | ❌ | ❌ | ❌ | ❌ |

### CField_GuildBoss (3 ops)

| Op | FName | v83 | v84 | v87 | v95 | jms_v185 |
|----|-------|-----|-----|-----|-----|----------|
| GUILD_BOSS | CField_GuildBoss::BasicActionAttack | ❌ | ❌ | ❌ | ❌ | ❌ |
| GUILD_BOSS_HEALER_MOVE | CField_GuildBoss::OnHealerMove | ❌ | ❌ | ❌ | ❌ | ❌ |
| GUILD_BOSS_PULLEY_STATE_CHANGE | CField_GuildBoss::OnPulleyStateChange | ❌ | ❌ | ❌ | ❌ | ❌ |

### CField_ContiMove (2 ops)

| Op | FName | v83 | v84 | v87 | v95 | jms_v185 |
|----|-------|-----|-----|-----|-----|----------|
| CONTI_MOVE (Init) | CField_ContiMove::Init | ⬜ | ⬜ | ⬜ | ❌ | ⬜ |
| CONTI_MOVE (OnContiMove) | CField_ContiMove::OnContiMove | ❌ | ❌ | ❌ | ❌ | ❌ |

### CField_AriantArena (2 ops)

| Op | FName | v83 | v84 | v87 | v95 | jms_v185 |
|----|-------|-----|-----|-----|-----|----------|
| ARIANT_ARENA_SHOW_RESULT | CField_AriantArena::OnShowResult | ❌ | ❌ | ❌ | ❌ | ❌ |
| ARIANT_ARENA_USER_SCORE | CField_AriantArena::OnUserScore | ❌ | ❌ | ❌ | ❌ | ❌ |

### CField_Battlefield (2 ops)

| Op | FName | v83 | v84 | v87 | v95 | jms_v185 |
|----|-------|-----|-----|-----|-----|----------|
| SHEEP_RANCH_CLOTHES | CField_Battlefield::OnTeamChanged | ❌ | ❌ | ❌ | ❌ | ❌ |
| SHEEP_RANCH_INFO | CField_Battlefield::OnScoreUpdate | ❌ | ❌ | ❌ | ❌ | ❌ |

### CField_Massacre (1 op)

| Op | FName | v83 | v84 | v87 | v95 | jms_v185 |
|----|-------|-----|-----|-----|-----|----------|
| PYRAMID_GAUGE | CField_Massacre::OnMassacreIncGauge | ❌ | ❌ | ❌ | ❌ | ❌ |

### CField_MassacreResult (1 op)

| Op | FName | v83 | v84 | v87 | v95 | jms_v185 |
|----|-------|-----|-----|-----|-----|----------|
| PYRAMID_SCORE | CField_MassacreResult::OnMassacreResult | ❌ | ❌ | ❌ | ❌ | ❌ |

### CField_Witchtower (1 op)

| Op | FName | v83 | v84 | v87 | v95 | jms_v185 |
|----|-------|-----|-----|-----|-----|----------|
| ARIANT_SCORE | CField_Witchtower::OnPacket | ⬜ | ⬜ | ⬜ | ❌ | ⬜ |

## Notes & collisions

- **GENERAL_CHAT** is the only already-verified work-list cell set (v84/v87/v95/jms
  ✅ via `chat/serverbound/ChatGeneral` (T1)); only v83 is ❌. It is the `PKT` row
  flagged in cfield-ops.md.
- **GUILD_OPERATION** has two STATUS.md rows. The work-list (CField) row maps the
  FName `CField::InputGuildName ...` to `guild/serverbound/GuildOperation` (T1):
  v95/jms ✅, v83/v84/v87 ❌. The *other* GUILD_OPERATION row
  (`CWvsContext::OnGuildResult` → `guild/clientbound/GuildCapacityChange`) is a
  separate clientbound op, not the CField work-list op, and is fully ✅. Both are
  `PKT`-flagged in cfield-ops.md.
- **WHISPER** has two STATUS.md rows, both `CField::OnWhisper` (clientbound); both
  ❌ across v83–jms. cfield-ops.md lists WHISPER twice — they correspond to these
  two rows.
- **SPOUSE_CHAT** also appears twice in STATUS.md (`OnCoupleMessage` and
  `CUIStatusBar::SendCoupleMessage`); the work-list FName is `OnCoupleMessage`.
  Both ❌/⬜.
- **CONTI_MOVE** appears twice (Init / OnContiMove); cfield-ops.md lists both.
- `⬜` cells mean the op does not exist at that version's opcode table (n/a) — not
  in scope for that version.
