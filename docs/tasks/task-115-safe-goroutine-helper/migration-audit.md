# task-115 Migration Audit

Pre-migration `tools/goroutine-guard.sh` findings: 167 (recorded 2026-07-02).
Every row must carry a disposition before the branch is done. Row count must equal 167.

**Completion (2026-07-02):** all 167 rows dispositioned — 166 `migrated` (4 carrying the design §6.1 accepted-consequence note for the atlas-model combinators) + 1 `allowlisted` (`libs/atlas-model/testutil/helpers.go`, the sole `//goroutine-guard:allow` marker in the repo). `tools/goroutine-guard.sh` exits 0 (clean, zero findings) and `tools/redis-key-guard.sh` exits 0, both from the repo root.

| # | file:line | form | classification | logger source | ctx source | disposition |
|---|---|---|---|---|---|---|
| 1 | services/atlas-buffs/atlas.com/buffs/character/processor.go:152 | anon | ticker | l | ctx | migrated |
| 2 | services/atlas-buffs/atlas.com/buffs/character/processor.go:197 | anon | ticker | l | ctx | migrated |
| 3 | services/atlas-buffs/atlas.com/buffs/tasks/task.go:12 | anon | ticker | l | ctx (plumbed: Register(t Task) → Register(l, ctx) func(t Task)) | migrated |
| 4 | services/atlas-buffs/atlas.com/buffs/main.go:65 | named-call | ticker | l | tdm.Context() | migrated |
| 5 | services/atlas-buffs/atlas.com/buffs/main.go:66 | named-call | ticker | l | tdm.Context() | migrated |
| 6 | services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/reservation/cache.go:31 | named-call | lifecycle | logrus.StandardLogger() | ctx (local cancellable; singleton has no logger/caller to plumb) | migrated |
| 7 | services/atlas-login/atlas.com/login/listener/registry.go:220 | anon | lifecycle | r.l | h.Ctx | migrated |
| 8 | services/atlas-login/atlas.com/login/socket/init.go:22 | anon | lifecycle | l | ctx | migrated |
| 9 | services/atlas-login/atlas.com/login/socket/init.go:38 | anon | lifecycle | l | ctx | migrated |
| 10 | services/atlas-login/atlas.com/login/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 11 | services/atlas-login/atlas.com/login/main.go:165 | named-call | lifecycle | l | tdm.Context() | migrated |
| 12 | services/atlas-login/atlas.com/login/main.go:178 | anon | ticker | l | tdm.Context() | migrated |
| 13 | services/atlas-login/atlas.com/login/main.go:205 | named-call | ticker | l | tdm.Context() | migrated |
| 14 | services/atlas-ban/atlas.com/ban/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 15 | services/atlas-ban/atlas.com/ban/main.go:81 | named-call | ticker | l | tdm.Context() | migrated |
| 16 | services/atlas-ban/atlas.com/ban/main.go:82 | named-call | ticker | l | tdm.Context() | migrated |
| 17 | services/atlas-asset-expiration/atlas.com/asset-expiration/task/periodic.go:43 | named-call | ticker | t.l | t.ctx (plumbed: NewPeriodicTask(l, interval) -> NewPeriodicTask(l, ctx, interval); main passes tdm.Context()) | migrated |
| 18 | services/atlas-channel/atlas.com/channel/listener/registry.go:208 | anon | lifecycle | r.l | h.Ctx | migrated |
| 19 | services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go:346 | anon | handler-spawned | l | ctx | migrated |
| 20 | services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go:357 | anon | handler-spawned | l | ctx | migrated |
| 21 | services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go:366 | anon | handler-spawned | l | ctx | migrated |
| 22 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:180 | anon | handler-spawned | l | ctx | migrated |
| 23 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:200 | anon | handler-spawned | l | ctx | migrated |
| 24 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:216 | anon | handler-spawned | l | ctx | migrated |
| 25 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:222 | anon | handler-spawned | l | ctx | migrated |
| 26 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:228 | anon | handler-spawned | l | ctx | migrated |
| 27 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:234 | anon | handler-spawned | l | ctx | migrated |
| 28 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:240 | anon | handler-spawned | l | ctx | migrated |
| 29 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:246 | anon | handler-spawned | l | ctx | migrated |
| 30 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:251 | anon | handler-spawned | l | ctx | migrated |
| 31 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:257 | anon | handler-spawned | l | ctx | migrated |
| 32 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:263 | anon | handler-spawned | l | ctx | migrated |
| 33 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:269 | anon | handler-spawned | l | ctx | migrated |
| 34 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:275 | anon | handler-spawned | l | ctx | migrated |
| 35 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:281 | anon | handler-spawned | l | ctx | migrated |
| 36 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:293 | anon | handler-spawned | l | ctx | migrated |
| 37 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:306 | anon | handler-spawned | l | ctx | migrated |
| 38 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:317 | anon | handler-spawned | l | ctx | migrated |
| 39 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:380 | anon | handler-spawned | l | ctx | migrated |
| 40 | services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:707 | named-call | handler-spawned | l | ctx | migrated |
| 41 | services/atlas-channel/atlas.com/channel/kafka/consumer/drop/consumer.go:163 | anon | handler-spawned | l | ctx | migrated |
| 42 | services/atlas-channel/atlas.com/channel/kafka/consumer/drop/consumer.go:195 | anon | handler-spawned | l | ctx | migrated |
| 43 | services/atlas-channel/atlas.com/channel/kafka/consumer/messenger/consumer.go:81 | anon | handler-spawned | l | ctx | migrated |
| 44 | services/atlas-channel/atlas.com/channel/kafka/consumer/messenger/consumer.go:89 | anon | handler-spawned | l | ctx | migrated |
| 45 | services/atlas-channel/atlas.com/channel/kafka/consumer/messenger/consumer.go:138 | anon | handler-spawned | l | ctx | migrated |
| 46 | services/atlas-channel/atlas.com/channel/kafka/consumer/messenger/consumer.go:151 | anon | handler-spawned | l | ctx | migrated |
| 47 | services/atlas-channel/atlas.com/channel/movement/processor.go:47 | anon | handler-spawned | p.l | p.ctx | migrated |
| 48 | services/atlas-channel/atlas.com/channel/movement/processor.go:54 | anon | handler-spawned | p.l | p.ctx | migrated |
| 49 | services/atlas-channel/atlas.com/channel/movement/processor.go:68 | anon | handler-spawned | p.l | p.ctx | migrated |
| 50 | services/atlas-channel/atlas.com/channel/movement/processor.go:85 | anon | handler-spawned | p.l | p.ctx | migrated |
| 51 | services/atlas-channel/atlas.com/channel/movement/processor.go:98 | anon | handler-spawned | p.l | p.ctx | migrated |
| 52 | services/atlas-channel/atlas.com/channel/movement/processor.go:135 | anon | handler-spawned | p.l | p.ctx | migrated |
| 53 | services/atlas-channel/atlas.com/channel/movement/processor.go:158 | anon | handler-spawned | p.l | p.ctx | migrated |
| 54 | services/atlas-channel/atlas.com/channel/movement/processor.go:165 | anon | handler-spawned | p.l | p.ctx | migrated |
| 55 | services/atlas-channel/atlas.com/channel/movement/processor.go:196 | anon | handler-spawned | p.l | p.ctx | migrated |
| 56 | services/atlas-channel/atlas.com/channel/movement/processor.go:205 | anon | handler-spawned | p.l | p.ctx | migrated |
| 57 | services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:218 | anon | handler-spawned | l | ctx | migrated |
| 58 | services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:244 | anon | handler-spawned | l | ctx | migrated |
| 59 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:208 | anon | handler-spawned | l | ctx | migrated |
| 60 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:216 | anon | handler-spawned | l | ctx | migrated |
| 61 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:259 | anon | handler-spawned | l | ctx | migrated |
| 62 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:267 | anon | handler-spawned | l | ctx | migrated |
| 63 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:304 | anon | handler-spawned | l | ctx | migrated |
| 64 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:312 | anon | handler-spawned | l | ctx | migrated |
| 65 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:406 | anon | handler-spawned | l | ctx | migrated |
| 66 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go:414 | anon | handler-spawned | l | ctx | migrated |
| 67 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/member/consumer.go:124 | anon | handler-spawned | l | ctx | migrated |
| 68 | services/atlas-channel/atlas.com/channel/kafka/consumer/party/member/consumer.go:157 | anon | handler-spawned | l | ctx | migrated |
| 69 | services/atlas-channel/atlas.com/channel/kafka/consumer/pet/consumer.go:210 | anon | handler-spawned | l | ctx | migrated |
| 70 | services/atlas-channel/atlas.com/channel/kafka/consumer/pet/consumer.go:216 | anon | handler-spawned | l | ctx | migrated |
| 71 | services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:204 | anon | handler-spawned | l | ctx | migrated |
| 72 | services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:214 | anon | handler-spawned | l | ctx | migrated |
| 73 | services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:244 | anon | handler-spawned | l | ctx | migrated |
| 74 | services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:283 | anon | handler-spawned | l | ctx | migrated |
| 75 | services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:309 | anon | handler-spawned | l | ctx | migrated |
| 76 | services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:320 | anon | handler-spawned | l | ctx | migrated |
| 77 | services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:340 | anon | handler-spawned | l | ctx | migrated |
| 78 | services/atlas-channel/atlas.com/channel/socket/init.go:22 | anon | lifecycle | l | ctx | migrated |
| 79 | services/atlas-channel/atlas.com/channel/socket/init.go:39 | anon | lifecycle | l | ctx | migrated |
| 80 | services/atlas-channel/atlas.com/channel/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 81 | services/atlas-channel/atlas.com/channel/main.go:318 | named-call | lifecycle | l | tdm.Context() | migrated |
| 82 | services/atlas-channel/atlas.com/channel/main.go:327 | named-call | ticker | l | tdm.Context() | migrated |
| 83 | services/atlas-guilds/atlas.com/guilds/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 84 | services/atlas-guilds/atlas.com/guilds/main.go:101 | named-call | ticker | l | tdm.Context() | migrated |
| 85 | services/atlas-pets/atlas.com/pets/pet/task.go:37 | anon | ticker | t.l | sctx | migrated |
| 86 | services/atlas-pets/atlas.com/pets/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 87 | services/atlas-pets/atlas.com/pets/main.go:91 | named-call | ticker | l | tdm.Context() | migrated |
| 88 | services/atlas-character-factory/atlas.com/character-factory/main.go:111 | named-call | lifecycle | l | tdm.Context() | migrated |
| 89 | services/atlas-skills/atlas.com/skills/tasks/task.go:12 | anon | ticker | l | ctx (plumbed: Register(t Task) -> Register(l, ctx) func(t Task)) | migrated |
| 90 | services/atlas-skills/atlas.com/skills/main.go:79 | named-call | ticker | l | tdm.Context() | migrated |
| 91 | services/atlas-invites/atlas.com/invites/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 92 | services/atlas-invites/atlas.com/invites/main.go:82 | named-call | ticker | l | tdm.Context() | migrated |
| 93 | services/atlas-reactors/atlas.com/reactors/tasks/task.go:12 | anon | ticker | l | ctx (plumbed: Register(t Task) -> Register(l, ctx) func(t Task)) | migrated |
| 94 | services/atlas-reactors/atlas.com/reactors/main.go:70 | named-call | ticker | l | tdm.Context() | migrated |
| 95 | services/atlas-summons/atlas.com/summons/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 96 | services/atlas-summons/atlas.com/summons/main.go:109 | anon | lifecycle | l | tdm.Context() | migrated |
| 97 | services/atlas-expressions/atlas.com/expressions/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 98 | services/atlas-expressions/atlas.com/expressions/main.go:51 | named-call | ticker | l | tdm.Context() | migrated |
| 99 | services/atlas-maps/atlas.com/maps/map/monster/processor.go:142 | named-call | handler-spawned | p.l | p.ctx | migrated |
| 100 | services/atlas-maps/atlas.com/maps/map/processor.go:84 | anon | handler-spawned | p.l | p.ctx | migrated |
| 101 | services/atlas-maps/atlas.com/maps/map/processor.go:87 | anon | handler-spawned | p.l | p.ctx | migrated |
| 102 | services/atlas-maps/atlas.com/maps/tasks/mist_tick.go:145 | anon | ticker | r.l | ctx | migrated |
| 103 | services/atlas-maps/atlas.com/maps/tasks/respawn.go:39 | named-call | ticker | r.l | tctx | migrated |
| 104 | services/atlas-maps/atlas.com/maps/tasks/respawn.go:42 | named-call | ticker | r.l | tctx | migrated |
| 105 | services/atlas-maps/atlas.com/maps/tasks/task.go:12 | anon | ticker | l | ctx (plumbed: Register(t Task) → Register(l, ctx) func(t Task)) | migrated |
| 106 | services/atlas-maps/atlas.com/maps/main.go:116 | named-call | ticker | l | tdm.Context() | migrated |
| 107 | services/atlas-maps/atlas.com/maps/main.go:117 | named-call | ticker | l | tdm.Context() | migrated |
| 108 | services/atlas-maps/atlas.com/maps/main.go:118 | named-call | ticker | l | tdm.Context() | migrated |
| 109 | services/atlas-drops/atlas.com/drops/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 110 | services/atlas-drops/atlas.com/drops/main.go:94 | named-call | ticker | l | tdm.Context() | migrated |
| 111 | services/atlas-configurations/atlas.com/configurations/main.go:61 | named-call | lifecycle | l | tdm.Context() | migrated |
| 112 | services/atlas-npc-conversations/atlas.com/npc/map/processor.go:56 | anon | handler-spawned | p.l | p.ctx (loop var hoisted: id := mapId) | migrated |
| 113 | services/atlas-renders/atlas.com/renders/character/handler.go:137 | anon | handler-spawned | l | r.Context() | migrated |
| 114 | services/atlas-renders/atlas.com/renders/mapr/handler.go:133 | anon | handler-spawned | l | r.Context() (arg hoisted: payload := body) | migrated |
| 115 | services/atlas-data/atlas.com/data/baseline/restore.go:234 | anon | handler-spawned | l | ctx (plumbed: runRestoreTables/restoreOneTable/copyInBinary gained an `l logrus.FieldLogger` param threaded from Restorer.L, no `l` was in scope 3 frames deep) | migrated |
| 116 | services/atlas-data/atlas.com/data/data/processor.go:224 | anon | handler-spawned | l | ctx | migrated |
| 117 | services/atlas-data/atlas.com/data/data/processor.go:236 | anon | handler-spawned | l | ctx | migrated |
| 118 | services/atlas-data/atlas.com/data/runtime/ingest/run.go:42 | named-call | ticker | l | ctx | migrated |
| 119 | services/atlas-data/atlas.com/data/main.go:111 | named-call | lifecycle | l | tdm.Context() | migrated |
| 120 | services/atlas-doors/atlas.com/doors/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 121 | services/atlas-doors/atlas.com/doors/main.go:110 | anon | lifecycle | l | tdm.Context() | migrated |
| 122 | services/atlas-monster-death/atlas.com/monster/kafka/consumer/monster/consumer.go:47 | anon | handler-spawned | l | ctx | migrated |
| 123 | services/atlas-monster-death/atlas.com/monster/kafka/consumer/monster/consumer.go:61 | anon | handler-spawned | l | ctx | migrated |
| 124 | services/atlas-transports/atlas.com/transports/main.go:125 | anon | ticker | l | tdm.Context() | migrated |
| 125 | services/atlas-character/atlas.com/character/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 126 | services/atlas-character/atlas.com/character/main.go:104 | named-call | ticker | l | tdm.Context() | migrated |
| 127 | services/atlas-marriages/atlas.com/marriages/scheduler/ceremony_timeout.go:48 | named-call | ticker | s.log | s.ctx | migrated |
| 128 | services/atlas-marriages/atlas.com/marriages/scheduler/proposal_expiry.go:48 | named-call | ticker | s.log | s.ctx | migrated |
| 129 | services/atlas-monsters/atlas.com/monsters/monster/processor.go:700 | anon | handler-spawned | p.l | p.ctx | migrated |
| 130 | services/atlas-monsters/atlas.com/monsters/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 131 | services/atlas-monsters/atlas.com/monsters/main.go:120 | anon | lifecycle | l | tdm.Context() | migrated |
| 132 | services/atlas-families/atlas.com/family/scheduler/reputation_reset.go:72 | named-call | ticker | j.log | ctx (Start(ctx) param) | migrated |
| 133 | services/atlas-account/atlas.com/account/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 134 | services/atlas-account/atlas.com/account/main.go:78 | named-call | ticker | l | tdm.Context() | migrated |
| 135 | services/atlas-mounts/atlas.com/mounts/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 136 | services/atlas-mounts/atlas.com/mounts/main.go:88 | named-call | ticker | l | tdm.Context() | migrated |
| 137 | services/atlas-merchant/atlas.com/merchant/service/teardown.go:41 | anon | lifecycle | logrus.StandardLogger() | m.context (Manager has no logger field; vendored copy of atlas-service) | migrated |
| 138 | services/atlas-merchant/atlas.com/merchant/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 139 | services/atlas-world/atlas.com/world/tasks/task.go:18 | anon | ticker | l | ctx | migrated |
| 140 | services/atlas-world/atlas.com/world/main.go:134 | named-call | lifecycle | l | tdm.Context() | migrated |
| 141 | services/atlas-world/atlas.com/world/main.go:147 | named-call | ticker | l | tdm.Context() | migrated |
| 142 | services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go:222 | anon | ticker | l | tdm.Context() | migrated |
| 143 | services/atlas-party-quests/atlas.com/party-quests/main.go:87 | anon | ticker | l | tdm.Context() | migrated |
| 144 | libs/atlas-service/teardown.go:41 | anon | lifecycle | logrus.StandardLogger() | m.context (lib Manager has no logger field) | migrated |
| 145 | libs/atlas-redis/coalesced.go:80 | named-call | lib-internal | logrus.StandardLogger() | context.Background() (lib: registry has no ctx; stopCh-based lifecycle) | migrated |
| 146 | libs/atlas-redis/tenant_coalesced.go:64 | named-call | lib-internal | logrus.StandardLogger() | context.Background() (lib: registry has no ctx; stopCh-based lifecycle) | migrated |
| 147 | libs/atlas-socket/server.go:125 | anon | lifecycle | l | ctx | migrated |
| 148 | libs/atlas-socket/server.go:152 | named-call | lib-internal | l | ctx | migrated |
| 149 | libs/atlas-socket/server.go:173 | anon | lifecycle | l | ctx | migrated |
| 150 | libs/atlas-socket/server.go:226 | named-call | lib-internal | fl | ctx | migrated |
| 151 | libs/atlas-outbox/drainer.go:148 | named-call | lib-internal | d.l | sweepCtx | migrated |
| 152 | libs/atlas-outbox/notify.go:28 | named-call | lib-internal | l | context.Background() (lib: newNotifier has no ctx; range-over-channel pump) | migrated |
| 153 | libs/atlas-rest/server/server.go:171 | anon | lifecycle | sb.l | sb.ctx | migrated |
| 154 | libs/atlas-rest/server/server.go:186 | anon | lifecycle | sb.l | ctx | migrated |
| 155 | libs/atlas-kafka/consumer/manager.go:145 | named-call | lib-internal | l | ctx | migrated |
| 156 | libs/atlas-kafka/consumer/manager.go:523 | anon | lib-internal | l | ctx | migrated |
| 157 | libs/atlas-kafka/consumer/manager.go:558 | anon | lib-internal | handlerLogger | wctx | migrated |
| 158 | libs/atlas-seeder/handlers.go:49 | anon | lib-internal | l | ctx | migrated |
| 159 | libs/atlas-model/model/processor.go:155 | anon | lib-internal | logrus.StandardLogger() | ctx | migrated (accepted: recovered worker panic never reaches errChannels — wg.Done() still fires, ExecuteForEachSlice returns nil for that item; Error log is the detection path, design §6.1) |
| 160 | libs/atlas-model/model/processor.go:167 | anon | lib-internal | logrus.StandardLogger() | ctx | migrated |
| 161 | libs/atlas-model/model/processor.go:208 | anon | lib-internal | logrus.StandardLogger() | ctx | migrated (accepted: recovered worker panic never reaches errChannels — wg.Done() still fires, ExecuteForEachMap returns nil for that item; Error log is the detection path, design §6.1) |
| 162 | libs/atlas-model/model/processor.go:220 | anon | lib-internal | logrus.StandardLogger() | ctx | migrated |
| 163 | libs/atlas-model/model/processor.go:441 | named-call | lib-internal | logrus.StandardLogger() | context.Background() (no ctx in SliceMap scope) | migrated (accepted: recovered panic leaves a zero-value element in results at res.index; Error log is the detection path, design §6.1) |
| 164 | libs/atlas-model/async/processor.go:72 | anon | lib-internal | logrus.StandardLogger() | ctx | migrated (accepted: recovered provider panic never reaches resultChannels/errChannels — AwaitSlice times out with ErrAwaitTimeout; Error log is the detection path, design §6.1) |
| 165 | libs/atlas-model/testutil/helpers.go:189 | anon | test-support | n/a | n/a | allowlisted — panic propagation is the point of a test harness |
| 166 | libs/atlas-lock/leader.go:155 | anon | lifecycle | le.cfg.log | leaderCtx | migrated |
| 167 | libs/atlas-lock/leader.go:167 | anon | lifecycle | le.cfg.log | leaderCtx | migrated |
