# task-115 Migration Audit

Pre-migration `tools/goroutine-guard.sh` findings: 167 (recorded 2026-07-02).
Every row must carry a disposition before the branch is done. Row count must equal 167.

| # | file:line | form | classification | logger source | ctx source | disposition |
|---|---|---|---|---|---|---|
| 1 | services/atlas-buffs/atlas.com/buffs/character/processor.go:152 | anon | | | | |
| 2 | services/atlas-buffs/atlas.com/buffs/character/processor.go:197 | anon | | | | |
| 3 | services/atlas-buffs/atlas.com/buffs/tasks/task.go:12 | anon | | | | |
| 4 | services/atlas-buffs/atlas.com/buffs/main.go:65 | named-call | | | | |
| 5 | services/atlas-buffs/atlas.com/buffs/main.go:66 | named-call | | | | |
| 6 | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/reservation/cache.go:31 | named-call | | | | |
| 7 | services/atlas-login/atlas.com/login/listener/registry.go:220 | anon | | | | |
| 8 | services/atlas-login/atlas.com/login/socket/init.go:22 | anon | | | | |
| 9 | services/atlas-login/atlas.com/login/socket/init.go:38 | anon | | | | |
| 10 | services/atlas-login/atlas.com/login/tasks/task.go:18 | anon | | | | |
| 11 | services/atlas-login/atlas.com/login/main.go:165 | named-call | | | | |
| 12 | services/atlas-login/atlas.com/login/main.go:178 | anon | | | | |
| 13 | services/atlas-login/atlas.com/login/main.go:205 | named-call | | | | |
| 14 | services/atlas-ban/atlas.com/ban/tasks/task.go:18 | anon | | | | |
| 15 | services/atlas-ban/atlas.com/ban/main.go:81 | named-call | | | | |
| 16 | services/atlas-ban/atlas.com/ban/main.go:82 | named-call | | | | |
| 17 | services/atlas-asset-expiration/atlas.com/asset-expiration/task/periodic.go:43 | named-call | | | | |
| 18 | services/atlas-channel/atlas.com/channel/listener/registry.go:208 | anon | | | | |
| 19 | services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go:346 | anon | | | | |
| 20 | services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go:357 | anon | | | | |
| 21 | services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go:366 | anon | | | | |
| 22 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:180 | anon | | | | |
| 23 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:200 | anon | | | | |
| 24 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:216 | anon | | | | |
| 25 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:222 | anon | | | | |
| 26 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:228 | anon | | | | |
| 27 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:234 | anon | | | | |
| 28 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:240 | anon | | | | |
| 29 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:246 | anon | | | | |
| 30 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:251 | anon | | | | |
| 31 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:257 | anon | | | | |
| 32 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:263 | anon | | | | |
| 33 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:269 | anon | | | | |
| 34 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:275 | anon | | | | |
| 35 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:281 | anon | | | | |
| 36 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:293 | anon | | | | |
| 37 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:306 | anon | | | | |
| 38 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:317 | anon | | | | |
| 39 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:380 | anon | | | | |
| 40 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:707 | named-call | | | | |
| 41 | services/atlas-channel/atlas.com/channel/kafka/consumer/drop/consumer.go:163 | anon | | | | |
| 42 | services/atlas-channel/atlas.com/channel/kafka/consumer/drop/consumer.go:195 | anon | | | | |
| 43 | services/atlas-channel/atlas.com/channel/kafka/consumer/messenger/consumer.go:81 | anon | | | | |
| 44 | services/atlas-channel/atlas.com/channel/kafka/consumer/messenger/consumer.go:89 | anon | | | | |
| 45 | services/atlas-channel/atlas.com/channel/kafka/consumer/messenger/consumer.go:138 | anon | | | | |
| 46 | services/atlas-channel/atlas.com/channel/kafka/consumer/messenger/consumer.go:151 | anon | | | | |
| 47 | services/atlas-channel/atlas.com/channel/movement/processor.go:47 | anon | | | | |
| 48 | services/atlas-channel/atlas.com/channel/movement/processor.go:54 | anon | | | | |
| 49 | services/atlas-channel/atlas.com/channel/movement/processor.go:68 | anon | | | | |
| 50 | services/atlas-channel/atlas.com/channel/movement/processor.go:85 | anon | | | | |
| 51 | services/atlas-channel/atlas.com/channel/movement/processor.go:98 | anon | | | | |
| 52 | services/atlas-channel/atlas.com/channel/movement/processor.go:135 | anon | | | | |
| 53 | services/atlas-channel/atlas.com/channel/movement/processor.go:158 | anon | | | | |
| 54 | services/atlas-channel/atlas.com/channel/movement/processor.go:165 | anon | | | | |
| 55 | services/atlas-channel/atlas.com/channel/movement/processor.go:196 | anon | | | | |
| 56 | services/atlas-channel/atlas.com/channel/movement/processor.go:205 | anon | | | | |
| 57 | services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:218 | anon | | | | |
| 58 | services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:244 | anon | | | | |
| 59 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:208 | anon | | | | |
| 60 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:216 | anon | | | | |
| 61 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:259 | anon | | | | |
| 62 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:267 | anon | | | | |
| 63 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:304 | anon | | | | |
| 64 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:312 | anon | | | | |
| 65 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:406 | anon | | | | |
| 66 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:414 | anon | | | | |
| 67 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/member/consumer.go:124 | anon | | | | |
| 68 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/member/consumer.go:157 | anon | | | | |
| 69 | services/atlas-channel/atlas.com/channel/kafka/consumer/pet/consumer.go:210 | anon | | | | |
| 70 | services/atlas-channel/atlas.com/channel/kafka/consumer/pet/consumer.go:216 | anon | | | | |
| 71 | services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:204 | anon | | | | |
| 72 | services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:214 | anon | | | | |
| 73 | services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:244 | anon | | | | |
| 74 | services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:283 | anon | | | | |
| 75 | services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:309 | anon | | | | |
| 76 | services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:320 | anon | | | | |
| 77 | services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:340 | anon | | | | |
| 78 | services/atlas-channel/atlas.com/channel/socket/init.go:22 | anon | | | | |
| 79 | services/atlas-channel/atlas.com/channel/socket/init.go:39 | anon | | | | |
| 80 | services/atlas-channel/atlas.com/channel/tasks/task.go:18 | anon | | | | |
| 81 | services/atlas-channel/atlas.com/channel/main.go:318 | named-call | | | | |
| 82 | services/atlas-channel/atlas.com/channel/main.go:327 | named-call | | | | |
| 83 | services/atlas-guilds/atlas.com/guilds/tasks/task.go:18 | anon | | | | |
| 84 | services/atlas-guilds/atlas.com/guilds/main.go:101 | named-call | | | | |
| 85 | services/atlas-pets/atlas.com/pets/pet/task.go:37 | anon | | | | |
| 86 | services/atlas-pets/atlas.com/pets/tasks/task.go:18 | anon | | | | |
| 87 | services/atlas-pets/atlas.com/pets/main.go:91 | named-call | | | | |
| 88 | services/atlas-character-factory/atlas.com/character-factory/main.go:111 | named-call | | | | |
| 89 | services/atlas-skills/atlas.com/skills/tasks/task.go:12 | anon | | | | |
| 90 | services/atlas-skills/atlas.com/skills/main.go:79 | named-call | | | | |
| 91 | services/atlas-invites/atlas.com/invites/tasks/task.go:18 | anon | | | | |
| 92 | services/atlas-invites/atlas.com/invites/main.go:82 | named-call | | | | |
| 93 | services/atlas-reactors/atlas.com/reactors/tasks/task.go:12 | anon | | | | |
| 94 | services/atlas-reactors/atlas.com/reactors/main.go:70 | named-call | | | | |
| 95 | services/atlas-summons/atlas.com/summons/tasks/task.go:18 | anon | | | | |
| 96 | services/atlas-summons/atlas.com/summons/main.go:109 | anon | | | | |
| 97 | services/atlas-expressions/atlas.com/expressions/tasks/task.go:18 | anon | | | | |
| 98 | services/atlas-expressions/atlas.com/expressions/main.go:51 | named-call | | | | |
| 99 | services/atlas-maps/atlas.com/maps/map/monster/processor.go:142 | anon | | | | |
| 100 | services/atlas-maps/atlas.com/maps/map/processor.go:84 | anon | | | | |
| 101 | services/atlas-maps/atlas.com/maps/map/processor.go:87 | anon | | | | |
| 102 | services/atlas-maps/atlas.com/maps/tasks/mist_tick.go:145 | anon | | | | |
| 103 | services/atlas-maps/atlas.com/maps/tasks/respawn.go:39 | anon | | | | |
| 104 | services/atlas-maps/atlas.com/maps/tasks/respawn.go:42 | anon | | | | |
| 105 | services/atlas-maps/atlas.com/maps/tasks/task.go:12 | anon | | | | |
| 106 | services/atlas-maps/atlas.com/maps/main.go:116 | named-call | | | | |
| 107 | services/atlas-maps/atlas.com/maps/main.go:117 | named-call | | | | |
| 108 | services/atlas-maps/atlas.com/maps/main.go:118 | named-call | | | | |
| 109 | services/atlas-drops/atlas.com/drops/tasks/task.go:18 | anon | | | | |
| 110 | services/atlas-drops/atlas.com/drops/main.go:94 | named-call | | | | |
| 111 | services/atlas-configurations/atlas.com/configurations/main.go:61 | named-call | | | | |
| 112 | services/atlas-npc-conversations/atlas.com/npc/map/processor.go:56 | anon | | | | |
| 113 | services/atlas-renders/atlas.com/renders/character/handler.go:137 | anon | | | | |
| 114 | services/atlas-renders/atlas.com/renders/mapr/handler.go:133 | anon | | | | |
| 115 | services/atlas-data/atlas.com/data/baseline/restore.go:234 | anon | | | | |
| 116 | services/atlas-data/atlas.com/data/data/processor.go:224 | anon | | | | |
| 117 | services/atlas-data/atlas.com/data/data/processor.go:236 | anon | | | | |
| 118 | services/atlas-data/atlas.com/data/runtime/ingest/run.go:42 | named-call | | | | |
| 119 | services/atlas-data/atlas.com/data/main.go:111 | named-call | | | | |
| 120 | services/atlas-doors/atlas.com/doors/tasks/task.go:18 | anon | | | | |
| 121 | services/atlas-doors/atlas.com/doors/main.go:110 | anon | | | | |
| 122 | services/atlas-monster-death/atlas.com/monster/kafka/consumer/monster/consumer.go:47 | anon | | | | |
| 123 | services/atlas-monster-death/atlas.com/monster/kafka/consumer/monster/consumer.go:61 | anon | | | | |
| 124 | services/atlas-transports/atlas.com/transports/main.go:125 | anon | | | | |
| 125 | services/atlas-character/atlas.com/character/tasks/task.go:18 | anon | | | | |
| 126 | services/atlas-character/atlas.com/character/main.go:104 | named-call | | | | |
| 127 | services/atlas-marriages/atlas.com/marriages/scheduler/ceremony_timeout.go:48 | named-call | | | | |
| 128 | services/atlas-marriages/atlas.com/marriages/scheduler/proposal_expiry.go:48 | named-call | | | | |
| 129 | services/atlas-monsters/atlas.com/monsters/monster/processor.go:700 | anon | | | | |
| 130 | services/atlas-monsters/atlas.com/monsters/tasks/task.go:18 | anon | | | | |
| 131 | services/atlas-monsters/atlas.com/monsters/main.go:120 | anon | | | | |
| 132 | services/atlas-families/atlas.com/family/scheduler/reputation_reset.go:72 | named-call | | | | |
| 133 | services/atlas-account/atlas.com/account/tasks/task.go:18 | anon | | | | |
| 134 | services/atlas-account/atlas.com/account/main.go:78 | named-call | | | | |
| 135 | services/atlas-mounts/atlas.com/mounts/tasks/task.go:18 | anon | | | | |
| 136 | services/atlas-mounts/atlas.com/mounts/main.go:88 | named-call | | | | |
| 137 | services/atlas-merchant/atlas.com/merchant/service/teardown.go:41 | anon | | | | |
| 138 | services/atlas-merchant/atlas.com/merchant/tasks/task.go:18 | anon | | | | |
| 139 | services/atlas-world/atlas.com/world/tasks/task.go:18 | anon | | | | |
| 140 | services/atlas-world/atlas.com/world/main.go:134 | named-call | | | | |
| 141 | services/atlas-world/atlas.com/world/main.go:147 | named-call | | | | |
| 142 | services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go:222 | anon | | | | |
| 143 | services/atlas-party-quests/atlas.com/party-quests/main.go:87 | anon | | | | |
| 144 | libs/atlas-service/teardown.go:41 | anon | | | | |
| 145 | libs/atlas-redis/coalesced.go:80 | named-call | | | | |
| 146 | libs/atlas-redis/tenant_coalesced.go:64 | named-call | | | | |
| 147 | libs/atlas-socket/server.go:125 | anon | | | | |
| 148 | libs/atlas-socket/server.go:152 | named-call | | | | |
| 149 | libs/atlas-socket/server.go:173 | anon | | | | |
| 150 | libs/atlas-socket/server.go:226 | named-call | | | | |
| 151 | libs/atlas-outbox/drainer.go:148 | named-call | | | | |
| 152 | libs/atlas-outbox/notify.go:28 | named-call | | | | |
| 153 | libs/atlas-rest/server/server.go:171 | anon | | | | |
| 154 | libs/atlas-rest/server/server.go:186 | anon | | | | |
| 155 | libs/atlas-kafka/consumer/manager.go:145 | named-call | lib-internal | l | ctx | migrated |
| 156 | libs/atlas-kafka/consumer/manager.go:523 | anon | lib-internal | l | ctx | migrated |
| 157 | libs/atlas-kafka/consumer/manager.go:558 | anon | lib-internal | handlerLogger | wctx | migrated |
| 158 | libs/atlas-seeder/handlers.go:49 | anon | | | | |
| 159 | libs/atlas-model/model/processor.go:155 | anon | lib-internal | logrus.StandardLogger() | ctx | migrated (accepted: recovered worker panic never reaches errChannels — wg.Done() still fires, ExecuteForEachSlice returns nil for that item; Error log is the detection path, design §6.1) |
| 160 | libs/atlas-model/model/processor.go:167 | anon | lib-internal | logrus.StandardLogger() | ctx | migrated |
| 161 | libs/atlas-model/model/processor.go:208 | anon | lib-internal | logrus.StandardLogger() | ctx | migrated (accepted: recovered worker panic never reaches errChannels — wg.Done() still fires, ExecuteForEachMap returns nil for that item; Error log is the detection path, design §6.1) |
| 162 | libs/atlas-model/model/processor.go:220 | anon | lib-internal | logrus.StandardLogger() | ctx | migrated |
| 163 | libs/atlas-model/model/processor.go:441 | named-call | lib-internal | logrus.StandardLogger() | context.Background() (no ctx in SliceMap scope) | migrated (accepted: recovered panic leaves a zero-value element in results at res.index; Error log is the detection path, design §6.1) |
| 164 | libs/atlas-model/async/processor.go:72 | anon | lib-internal | logrus.StandardLogger() | ctx | migrated (accepted: recovered provider panic never reaches resultChannels/errChannels — AwaitSlice times out with ErrAwaitTimeout; Error log is the detection path, design §6.1) |
| 165 | libs/atlas-model/testutil/helpers.go:189 | anon | test-support | n/a | n/a | allowlisted — panic propagation is the point of a test harness |
| 166 | libs/atlas-lock/leader.go:155 | anon | lifecycle | le.cfg.log | leaderCtx | migrated |
| 167 | libs/atlas-lock/leader.go:167 | anon | lifecycle | le.cfg.log | leaderCtx | migrated |
