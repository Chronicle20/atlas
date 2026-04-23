# Kafka At-Least-Once Delivery — Task Checklist

Last Updated: 2026-02-19

## Phase 1: Extend KafkaReader Interface (libs/atlas-kafka)

- [x] 1. Add `FetchMessage` and `CommitMessages` to `KafkaReader`/`MessageReader` interfaces (`consumer/manager.go`)
- [x] 2. Replace `ReadMessage` with `FetchMessage` in consumer loop (`consumer/manager.go:177`)
- [x] 3. Make handler dispatch synchronous — wait for all handlers before committing (`consumer/manager.go:215-256`)
- [x] 4. Add `CommitMessages` call after successful handler completion (`consumer/manager.go:198`)
- [x] 5. Add commit error handling (log and continue) (`consumer/manager.go:199`)
- [x] 6. Wrap handler execution in `recover()` to survive panics (`consumer/manager.go:258-267`)

## Phase 2: Update Tests (libs/atlas-kafka)

- [x] 7. Update `MockReader` and `ChannelMockReader` to implement new interface (`consumer/manager_test.go`)
- [x] 8. Add test: `TestCommitAfterHandlerCompletes`
- [x] 9. Add test: `TestHandlerErrorPreventsCommit`
- [x] 10. Add test: `TestHandlerPanicPreventsCommit`
- [x] 11. Add test: `TestMultipleHandlersAllCompleteBeforeCommit`
- [x] 12. Verify existing tests pass (`TestGracefulShutdown`, `TestSpanPropagation`, `TestTenantPropagation`)

## Phase 3: Build Validation

- [x] 13. Build all 60 modules — all pass
- [x] 14. No service test mocks implement `KafkaReader` — no fixes needed
- [x] 15. Run service tests — all failures pre-existing (atlas-drops, atlas-npc-conversations, timeouts)

## Phase 4: Idempotency Audit

- [ ] 16. Audit all ~531 handler registrations for duplicate-message safety
- [ ] 17. Categorize handlers (naturally idempotent / tolerates duplicates / needs attention)
- [ ] 18. Add idempotency guards where needed
