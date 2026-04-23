# Plan: Party Quest Admin Commands

Last Updated: 2026-02-16

## Executive Summary

Add two GM text commands to `atlas-messages` for testing party quests:
- **`@pq register <questId>`** — Register the character for a party quest by quest ID.
- **`@pq stage`** — Force-advance the current stage of the party quest the character is in.

No changes to `atlas-party-quests` — existing `REGISTER` and `STAGE_ADVANCE` commands are sufficient. `STAGE_ADVANCE` already skips condition checks (those live in `STAGE_CLEAR_ATTEMPT`). The `@pq stage` command does a REST lookup to resolve the instance ID from the character, then sends `STAGE_ADVANCE`.

---

## Implementation

### 1. Kafka message layer
**File:** `kafka/message/party_quest/kafka.go` (CREATE)
- `EnvCommandTopic = "COMMAND_TOPIC_PARTY_QUEST"`
- Command envelope matching party-quests service format
- `RegisterCommandProvider(...)` and `StageAdvanceCommandProvider(...)` message providers

### 2. REST client for party-quest instances
**Files:** `party_quest/model.go`, `party_quest/rest.go`, `party_quest/requests.go`, `party_quest/processor.go` (CREATE)
- Minimal model — only need instance `id` (uuid.UUID)
- REST lookup via `GET /party-quests/instances/character/{characterId}` using `PARTY_QUESTS` env var
- Processor with `GetByCharacter(characterId)` method

### 3. Command handlers
**File:** `command/party_quest/commands.go` (CREATE)
- `PQRegisterCommandProducer` — regex `^@pq\s+register\s+(\S+)$`, GM check, sends REGISTER Kafka command
- `PQStageCommandProducer` — regex `^@pq\s+stage$`, GM check, REST lookup instance by character, sends STAGE_ADVANCE

### 4. Wiring
- `main.go` — Register both command producers
- `command/help/commands.go` — Add help entries

All paths relative to `services/atlas-messages/atlas.com/messages/`.
