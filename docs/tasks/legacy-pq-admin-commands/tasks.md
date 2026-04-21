# Tasks: Party Quest Admin Commands

Last Updated: 2026-02-16

## Phase 1: atlas-party-quests — FORCE_STAGE_COMPLETE

- [ ] **1.1** Add `CommandTypeForceStageComplete` and `ForceStageCompleteCommandBody` to `kafka/message/party_quest/kafka.go`
- [ ] **1.2** Add `ForceStageComplete` and `ForceStageCompleteAndEmit` to processor interface and implement
- [ ] **1.3** Add `handleForceStageCompleteCommand` handler and register in `InitHandlers`
- [ ] **1.4** Build and test: `go build && go test ./... -count=1`

## Phase 2: atlas-messages — Kafka message layer

- [ ] **2.1** Create `kafka/message/party_quest/kafka.go` with command types and message providers

## Phase 3: atlas-messages — Command handlers

- [ ] **3.1** Create `command/party_quest/commands.go` with `PQRegisterCommandProducer` and `PQStageCommandProducer`
- [ ] **3.2** Register commands in `main.go`
- [ ] **3.3** Update help text in `command/help/commands.go`
- [ ] **3.4** Build and test: `go build && go test ./... -count=1`
