# Task-125 Skill/Mastery Books — Execution Context

Companion to `plan.md`. Key files, decisions, and dependencies an implementer needs without re-deriving the design.

## Flow (one line)

channel handler → `REQUEST_SKILL_BOOK_USE` (COMMAND_TOPIC_CONSUMABLE) → consumables validate + roll → `skill_book_use` saga (destroy book → conditional create/update skill) → one-time handler on EVENT_TOPIC_SAGA_STATUS → `SKILL_BOOK_RESULT` (EVENT_TOPIC_CONSUMABLE_STATUS) → channel writer `SKILL_LEARN_ITEM_RESULT` (map broadcast when canUse, requester-only otherwise).

## Key files (patterns to copy)

| Concern | Copy from |
|---|---|
| Serverbound codec shape | `libs/atlas-packet/inventory/serverbound/item_use.go` (same 4+2+4 body) |
| Clientbound codec + golden test | `libs/atlas-packet/character/clientbound/item_upgrade.go` + `_test.go` |
| Codec test harness | `pt.Variants` / `pt.CreateContext` / `pt.RoundTrip` in `libs/atlas-packet/test` |
| Channel handler → command | `services/atlas-channel/atlas.com/channel/socket/handler/character_item_use.go` (scroll variant) |
| Channel consumer split (map vs requester) | `kafka/consumer/consumable/consumer.go` `handleScrollConsumableEvent` (map) / `handleErrorConsumableEvent` (requester) |
| Consumables command demux | `kafka/consumer/consumable/consumer.go` `handleRequestScroll` |
| One-time handler + validator | `consumable/processor.go:568-577` (scroll) + `kafka/once/compartment/once.go` |
| Saga client (message mirror + producer + processor) | `services/atlas-map-actions/atlas.com/map-actions/saga/` + `kafka/message/saga/kafka.go` |
| Conditional create/update skill steps | `services/atlas-messages/atlas.com/messages/command/character/skill/commands.go:56-92` |
| Skills REST client | `services/atlas-messages/atlas.com/messages/skill/` (requests/rest/model/processor) |
| Orchestrator reverse-walk compensation + test | `compensator.go` `compensatePetEvolution`/`DispatchPetEvolutionRollbacks` (~:1044-1140) + `TestPetEvolutionCompensationRefundsResources` |
| Seed template entry shape | `template_gms_83_1.json` handlers ~:419, writers ~:1822 |

## Load-bearing decisions (verified at plan time — do not re-litigate)

1. **gms_92 EXCLUDED** (deviation from PRD FR-5.1). Its template has NO item-use family (37 handlers, login/field only); no v92 registry/IDB; known opcodes shift contradictorily vs v95 (ITEM_MOVE = v95+1, CHAR_INFO = v95−1) so interpolation is invention. Mount-food precedent. Banner in PR description.
2. **Roll is `roll < int32(rate)`** — NOT the scroll path's `<=` (that off-by-one gives success=0 a 1% pass). success=0 never, success=100 always.
3. **`update_skill` clobbers** (atlas-skills `.Select(columns)` writes zero values): the update payload MUST carry current Level + Expiration from the skills fetch. `create_skill` only when no record exists (else atlas-skills Create errors) — level 0, zero-time expiration (= permanent).
4. **Shared saga payloads, no WorldId**: all existing skill-step producers emit `sharedsaga.UpdateSkillPayload`/`CreateSkillPayload`; orchestrator unmarshals worldId=0. Follow precedent.
5. **One one-time saga-status handler per request**, body type `json.RawMessage`, validator matches txId AND Type ∈ {COMPLETED, FAILED}. Two typed registrations would leak the one that never fires.
6. **Destroy-first ordering** makes duplicate requests safe: second saga's destroy fails on the emptied slot → FAILED → canUse=0; no double-grant.
7. **Compensation** routes by SAGA TYPE (before the action switch, like CharacterCreation/PetEvolution) — a failed skill step re-awards via the new `DestroyAssetFromSlotPayload.TemplateId` (additive field); a failed destroy step compensates nothing. TemplateId==0 (legacy) must skip, never re-award item 0.
8. **Result routing:** canUse=1 → `ForSessionsInMap`; canUse=0 → requester only. Client demuxes glow (everyone) vs sound/message (local). The result packet is what clears the client's exclusive-request lock — every request must produce exactly one.
9. **Skill-window refresh is free**: atlas-skills emits CREATED/UPDATED on EVENT_TOPIC_SKILL_STATUS; channel's existing skill consumer announces `CharacterSkillChange`. Do NOT duplicate it in the new consumer.
10. **No channel `socket/writer/` file**: lib codecs are announced directly by writer-name const (ItemUpgrade precedent); the design's "thin writer file" would be dead code.
11. **No deploy changes**: `COMMAND_TOPIC_SAGA` + `EVENT_TOPIC_SAGA_STATUS` are in the shared `atlas-env` configmap; `RootUrl("SKILLS")` falls back to `BASE_SERVICE_URL`. Do NOT add `SKILLS_SERVICE_URL` overrides.

## Pre-verified data

- **WZ sweep clean** (v83 dump `tmp/<tenant-uuid>/GMS/83.1/Item.wz/Consume/022{8,9}.img.xml`): 26+139 books, all with explicit success (>0), masterLevel, non-empty skill[]. 228s have no reqSkillLevel node (→0, teachable when unlearned).
- **Fixture-friendly samples:** 2280000 {success 100, ML 10, req 0, skills [2121003]}; 2290000 {success 70, ML 20, req 5, skills [1121001,1221001,1321001]}.
- **Opcodes (hex):** sb 0x52/0x52/0x55/0x58/0x4A, cb 0x33/0x33/0x33/0x32/0x30 (v83/v84/v87/v95/jms). All 10 template slots collision-checked FREE.
- **IDA addresses from design:** cb v83 `OnSkillLearnItemResult`@0xa1e5af; sb v95 body @0x9d65e0. Neither fname exists in ANY audit export dir yet — both need surgical splices during the fixture phase.

## Dependencies / sequencing

- Tasks 1-2 (packet lib) and 3 (saga lib) are independent; 4 needs 3; 5-8 (consumables) need 3+5+6+7 in order; 9-10 (channel) need 1+2; 11 needs 1+2 names only; 12 needs all code tasks; 13 needs 1+2+12; 14 before PR; 15 post-merge.
- **IDB availability gates Task 13**: only v83 + v95 loaded (2026-07-02). v84/v87/jms cells are recorded as blocked, not faked. `list_instances` and match binary NAME first — ports rotate.
- Consumables gains its first saga submission — `kafka/consumer/saga/consumer.go` (topic registration, no persistent handlers) MUST be registered in main.go or one-time handlers never fire.

## Known sharp edges

- gms_95 template anchor block (`CharacterItemUseScrollHandle`) has no validator line — match the anchor exactly; the NEW entry still gets `LoggedInValidator`.
- Pre-existing, out of scope: `template_gms_95_1.json` has 35 validator-less handler entries (silently dropped at runtime) — surfaced to owner, do not fix here.
- Accepted limitation (design §4-C): consumables pod death mid-saga loses the result packet (client wedged until relog); matches existing scroll-flow behavior.
- `consumable/processor.go` import aliases are crowded (`character` = service pkg, `ts` = constants character, `character2` = map/character) — check the import block before adding `skill2`/`saga2`/`sagamsg`/`sagaonce`.
