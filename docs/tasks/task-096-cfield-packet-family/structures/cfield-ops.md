# task-096 CField op work-list (75 ops with ❌, from STATUS.md)

dir: CB=clientbound (On*), SB=serverbound (Send*), ?=unresolved. PKT=already has a codec file.


## CField  (45 ops)
- [SB] ADMIN_CHAT                       CField::SendChatMsgSlash                 - ❌5
- [SB] ADMIN_COMMAND                    CField::SendChatMsgSlash                 - ❌5
- [SB] ADMIN_LOG                        CField::SendChatMsgSlash                 - ❌5
- [CB] ADMIN_RESULT                     CField::OnAdminResult                    - ❌5
- [CB] ARIANT_RESULT                    CField::OnWarnMessage                    - ❌4
- [CB] BLOCKED_MAP                      CField::OnTransferFieldReqIgnored        - ❌5
- [CB] BLOCKED_SERVER                   CField::OnTransferChannelReqIgnored      - ❌5
- [CB] FIELD_OBSTACLE_ALL_RESET         CField::OnFieldObstacleAllRese           - ❌5
- [CB] FIELD_OBSTACLE_ONOFF             CField::OnFieldObstacleOnOff             - ❌5
- [CB] FIELD_OBSTACLE_ONOFF_LIST        CField::OnFieldObstacleOnOffStatus       - ❌5
- [CB] FOOTHOLD_INFO                    CField::OnRequestFootHoldInfo            - ❌5
- [CB] FORCED_MAP_EQUIP                 CField::OnFieldSpecificData              - ❌5
- [SB] GENERAL_CHAT                     CField::SendChatMsg                      PKT ❌1
- [CB] GMEVENT_INSTRUCTIONS             CField::OnDesc                           - ❌5
- [?] GUILD_OPERATION                  CField::InputGuildName                   PKT ❌3
- [CB] HORNTAIL_CAVE                    CField::OnHontailTimer                   - ❌5
- [CB] IDA_0X098                        CField::OnStalkResult                    - ❌1
- [CB] IDA_0X09C                        CField::OnFootHoldInfo                   - ❌2
- [CB] IDA_0X09D                        CField::OnRequestFootHoldInfo            - ❌1
- [CB] IDA_0X0A4                        CField::OnStalkResult                    - ❌1
- [CB] IDA_0X0AA                        CField::OnFootHoldInfo                   - ❌2
- [CB] IDA_0X0AC                        CField::OnStalkResult                    - ❌2
- [CB] IDA_0X0B0                        CField::OnFootHoldInfo                   - ❌1
- [CB] IDA_0X0B1                        CField::OnRequestFootHoldInfo            - ❌1
- [CB] IDA_0X169                        CField::OnHontailTimer                   - ❌2
- [SB] MATCH_TABLE                      CField::SendChatMsgSlash                 - ❌5
- [CB] MTS_OPERATION                    CField::OnCharacterSale                  - ❌4
- [CB] MTS_OPERATION2                   CField::OnCharacterSale                  - ❌4
- [CB] MULTICHAT                        CField::OnGroupMessage                   - ❌5
- [CB] OX_QUIZ                          CField::OnQuiz                           - ❌5
- [CB] PLAY_JUKEBOX                     CField::OnPlayJukeBox                    - ❌5
- [CB] SET_OBJECT_STATE                 CField::OnSetObjectState                 - ❌5
- [CB] SET_QUEST_CLEAR                  CField::OnSetQuestClear                  - ❌5
- [CB] SET_QUEST_TIME                   CField::OnSetQuestTime                   - ❌5
- [SB] SLIDE_REQUEST                    CField::SendChatMsgSlash                 - ❌2
- [CB] SPOUSE_CHAT                      CField::OnCoupleMessage                  - ❌4
- [CB] STOP_CLOCK                       CField::OnDestroyClock                   - ❌5
- [SB] SUE_CHARACTER                    CField::SendChatMsgSlash                 - ❌4
- [CB] SUMMON_ITEM_INAVAILABLE          CField::OnSummonItemInavailable          - ❌5
- [?] USE_DOOR                         CField::TryEnterTownPortal               - ❌5
- [CB] VICIOUS_HAMMER                   CField::OnItemUpgrade                    - ❌4
- [CB] WHISPER                          CField::OnWhisper                        - ❌5
- [CB] WHISPER                          CField::OnWhisper                        - ❌5
- [CB] WITCH_TOWER_SCORE_UPDATE         CField::OnChaosZakumTimer                - ❌5
- [CB] ZAKUM_SHRINE                     CField::OnZakumTimer                     - ❌5

## CField_SnowBall  (6 ops)
- [CB] HIT_SNOWBALL                     CField_SnowBall::OnSnowBallHit           - ❌5
- [?] LEFT_KNOCKBACK                   CField_SnowBall::Update                  - ❌5
- [CB] LEFT_KNOCK_BACK                  CField_SnowBall::OnSnowBallTouch         - ❌5
- [?] SNOWBALL                         CField_SnowBall::BasicActionAttack       - ❌5
- [CB] SNOWBALL_MESSAGE                 CField_SnowBall::OnSnowBallMsg           - ❌5
- [CB] SNOWBALL_STATE                   CField_SnowBall::OnSnowBallState         - ❌5

## CField_Tournament  (5 ops)
- [CB] TOURNAMENT                       CField_Tournament::OnTournament          - ❌5
- [CB] TOURNAMENT_CHARACTERS            CField_Tournament::OnPacket              - ❌5
- [CB] TOURNAMENT_MATCH_TABLE           CField_Tournament::OnTournamentMatchTable - ❌5
- [CB] TOURNAMENT_SET_PRIZE             CField_Tournament::OnTournamentSetPrize  - ❌5
- [CB] TOURNAMENT_UEW                   CField_Tournament::OnTournamentUEW       - ❌5

## CField_Wedding  (4 ops)
- [CB] WEDDING_ACTION                   CField_Wedding::OnWeddingProgress        - ❌4
- [CB] WEDDING_CEREMONY_END             CField_Wedding::OnWeddingCeremonyEnd     - ❌5
- [CB] WEDDING_PROGRESS                 CField_Wedding::OnWeddingProgress        - ❌5
- [CB] WEDDING_TALK                     CField_Wedding::OnWeddingProgress        - ❌4

## CField_Coconut  (3 ops)
- [?] COCONUT                          CField_Coconut::BasicActionAttack        - ❌5
- [CB] COCONUT_HIT                      CField_Coconut::OnCoconutHit             - ❌5
- [CB] COCONUT_SCORE                    CField_Coconut::OnCoconutScore           - ❌5

## CField_GuildBoss  (3 ops)
- [?] GUILD_BOSS                       CField_GuildBoss::BasicActionAttack      - ❌5
- [CB] GUILD_BOSS_HEALER_MOVE           CField_GuildBoss::OnHealerMove           - ❌5
- [CB] GUILD_BOSS_PULLEY_STATE_CHANGE   CField_GuildBoss::OnPulleyStateChange    - ❌5

## CField_ContiMove  (2 ops)
- [?] CONTI_MOVE                       CField_ContiMove::Init                   - ❌1
- [CB] CONTI_MOVE                       CField_ContiMove::OnContiMove            - ❌5

## CField_AriantArena  (2 ops)
- [CB] ARIANT_ARENA_SHOW_RESULT         CField_AriantArena::OnShowResult         - ❌5
- [CB] ARIANT_ARENA_USER_SCORE          CField_AriantArena::OnUserScore          - ❌5

## CField_Battlefield  (2 ops)
- [CB] SHEEP_RANCH_CLOTHES              CField_Battlefield::OnTeamChanged        - ❌5
- [CB] SHEEP_RANCH_INFO                 CField_Battlefield::OnScoreUpdate        - ❌5

## CField_Massacre  (1 ops)
- [CB] PYRAMID_GAUGE                    CField_Massacre::OnMassacreIncGauge      - ❌5

## CField_MassacreResult  (1 ops)
- [CB] PYRAMID_SCORE                    CField_MassacreResult::OnMassacreResult  - ❌5

## CField_Witchtower  (1 ops)
- [CB] ARIANT_SCORE                     CField_Witchtower::OnPacket              - ❌1
