# Context — Pet Evolution (Quest-Driven)

Companion to `plan-quest-driven.md` / `design-quest-driven.md`. Key files, decisions, and
dependencies an implementer needs.

## The pivot in one paragraph
Pet evolution was built as a synthetic free NPC conversation on the wrong NPC (1032102 = Mar the
Fairy; the real one is 9102001 = Garnox). Canonically it is **quest-driven**: quests 8184/8185/8189/4659
at NPC 9102001, whose completion runs an `endscript` that evolves the pet. Atlas already has the
quest-conversation (startscript/endscript) mechanism. The work is: enforce the quest's pet/tameness
gate, author the Garnox quest conversations, and back out the synthetic layer. The evolution **engine**
is retained.

## The four quests (NPC 9102001, all pettamenessmin=1642)
| Quest | Name | Kind | Pet | Complete extras | Evolves to |
|---|---|---|---|---|---|
| 8184 | Pet's Evolution1 | item turn-in (non-script) | 5000029 baby dragon | 50×4000029, 50×4000023 → 10×2120000 | (unlocks 8185) |
| 8185 | Pet's Evolution2 | script (`q8185e`) | 5000029 | Rock 5380000 | adult 5000030–33 |
| 8189 | Pet's Re-Evolution | script | adult {5000030-33,5000049-52} | Rock | different adult |
| 4659 | Robo Upgrade! | script (`q4659e`) | 5000048 baby robo | Rock + 50×4000111 | adult 5000049–53 |

## Key code locations
- **Quest-conversation engine** (reused): `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/` (keyed by questId; `startStateMachine`/`endStateMachine`); dispatch `quest/processor.go:131` `GetStateMachineForCharacter` (NOT_STARTED→start, STARTED→end). Trigger: channel `socket/handler/quest_action.go` (`QuestActionScriptStart`=4 / `ScriptEnd`=5) → `StartQuestConversation` Kafka cmd → consumer `kafka/consumer/quest/consumer.go`.
- **Operations** (reused): `conversation/operation_executor.go` — `evolve_pet` (~2528), `enumerate_evolvable_pets` (~802), `destroy_item` (~1512), `complete_quest` (~1801), `start_quest` (~1864).
- **petTameness gate (NEW):**
  - shared const `libs/atlas-saga/validation.go:32` (next to `PetCountCondition`).
  - query-aggregator eval `validation/model.go` (`Evaluate` ~385; PetCount case ~719; `Condition.values` field :94); context `validation/context.go:37` (`petCount` → add `spawnedPets`); fetch site `validation/processor.go` (`reqs.Pets` block).
  - query-aggregator already fetches per-pet closeness: `pet/processor.go:32` `GetPets`, `pet/rest.go:12-19` (`Closeness`, `TemplateId`, `Slot`).
  - atlas-quest emission `data/validation/processor.go` `buildStartConditions`/`buildEndConditions`; const `data/validation/model.go:16-25`; requirements parsed `data/quest/rest.go:62-63` (`Pet []uint32`, `PetTamenessMin int16`).
- **GM test command (NEW):** GM commands live in **atlas-messages** (`messages/command/<domain>/commands.go`) — `@`-phrase regexp, GM-gated (`c.Gm()`), `me`/`map`/`<name>` targeting, direct producer emit, registered in `messages/main.go`. Add `command/pet/commands.go` `AwardTamenessCommandProducer` for `@award <target> tameness <amount>`, reusing atlas-pets' existing `AWARD_CLOSENESS` (additive — `kafka/message/pet/kafka.go:CommandAwardCloseness`, body `{Amount uint16}`). Needs a new atlas-messages `pet` lookup (spawned pet id, slot≥0) + `kafka/message/pet` producer. Reference cmds: `command/character/commands.go` `AwardMesoCommandProducer` (target resolution), `command/monster/commands.go` `MobSpawnCommandProducer` (direct emit). **No atlas-pets/atlas-channel change.**
- **Backout surface:** seeds `deploy/seed/gms/*/npc-conversations/npc/npc-9102001.json` (6); backend `conversation/{model,rest,processor}.go` + 3 `pickfromcontext_*_test.go`; UI `conversation.ts`, `stateMeta.ts`, `transitions.ts` (revert commit `81f57e4e8`).

## Decisions (locked)
- **Single composite `petTameness` condition** (pet-id set in `Values`, min closeness in `Value`), not separate pet + tameness — tameness must bind to the same summoned pet; 8189's id requirement is an OR over 8 adults. Eval = max closeness among spawned pets matching the id set, compared `>=`.
- **Gate is tameness (closeness) ≥ 1642**, not pet level 15 (the old conversation's mistake).
- **Dialogue source = Cosmic `q{id}s/e.js`**, not Say.img (incomplete for these script quests). Use `/convert-quest`. Inherited atlas conversions are structure-only references, NOT a correctness oracle (per user).
- **No multi-pet chooser** — the quest Check constrains the pet; the end machine evolves the qualifying summoned pet via `enumerate_evolvable_pets`. `pickFromContext` removed.
- **8184 needs no conversation** — plain item turn-in; only the Phase B gate.
- Plan/design use the `-quest-driven` suffix to avoid clobbering the superseded `plan.md`/`design.md`.

## Dependencies / blockers
- **Phase C blocked** on obtaining the Cosmic `.js` scripts (not in repo). Phases A, B unblocked.
- Live test needs: tenant re-ingest + `atlas-data` REST restart (stale in-memory cache — confirmed this session); the `@award me tameness <amount>` GM command (Task B5).

## Verification gotchas (from this session)
- Worktree `go.work` is unreliable for `./...`; run Go checks from the module's go.mod dir with `GOWORK=off`.
- atlas-data REST caches pet data in-memory; after re-ingest, `rollout restart deploy/atlas-data` or values stay stale.
- query-aggregator pet container/condition names must match real accessors (`Slot()/TemplateId()/Closeness()`) — verify against `pet/model.go`.
- `docker buildx bake` is mandatory for any go.mod-touched service (atlas-saga is a new dep for query-aggregator/atlas-quest → confirms the shared Dockerfile `COPY libs/...`).
