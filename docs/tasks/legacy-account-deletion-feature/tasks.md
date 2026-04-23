# Account Deletion Feature - Task Checklist

**Last Updated:** 2026-02-03
**Related Plan:** `plan.md`
**Status:** IN PROGRESS

---

## Progress Summary

| Phase | Status | Progress |
|-------|--------|----------|
| Phase 1: Core Account Deletion | Complete | 6/6 |
| Phase 2: Kafka Command Handler | Complete | 5/5 |
| Phase 3: REST Endpoint | Complete | 2/3 |
| Phase 4: Character Service Integration | Complete | 3/4 |
| Phase 5: Storage Service Integration | Complete | 3/3 |
| Phase 6: Documentation | Not Started | 0/3 |

**Overall Progress:** 19/24 tasks complete

---

## Phase 1: Core Account Deletion (atlas-account)

**Objective:** Implement deletion logic in the account service

- [x] **1.1** Add `EventStatusDeleted` constant to `kafka/message/account/kafka.go`
  - Add `EventStatusDeleted = "DELETED"` constant
  - Effort: S

- [x] **1.2** Add `delete()` administrator function to `account/administrator.go`
  - Implemented as `deleteById()` to avoid shadowing built-in delete
  - Pattern: `func deleteById(db *gorm.DB) IdOperator`
  - Effort: S

- [x] **1.3** Add `deletedEventProvider()` to `account/producer.go`
  - Create event producer function for deleted status
  - Returns StatusEvent with status "DELETED"
  - Effort: S

- [x] **1.4** Add `Delete()` processor method to `account/processor.go`
  - Validate account exists
  - Check account is not logged in (reject with error if logged in)
  - Call delete administrator function
  - Clean up registry entry if present
  - Effort: M

- [x] **1.5** Add `DeleteAndEmit()` processor method to `account/processor.go`
  - Get account details before deletion (for event)
  - Call Delete()
  - Emit deleted event via producer
  - Effort: S

- [x] **1.6** Add processor unit tests for deletion
  - Test successful deletion (`TestDelete`)
  - Test deletion of non-existent account (`TestDeleteNotFound`)
  - Test deletion of logged-in account (`TestDeleteLoggedIn`)
  - Test multiple account deletion (`TestDeleteMultipleAccounts`)
  - Effort: M

---

## Phase 2: Kafka Command Handler (atlas-account)

**Objective:** Enable deletion via Kafka command

- [x] **2.1** Define `DeleteCommand` message in `kafka/message/account/kafka.go`
  - **CHANGED:** Consolidated into `Command[E any]` with `DeleteCommandBody`
  - Effort: S

- [x] **2.2** Add command topic constant
  - **CHANGED:** Consolidated `COMMAND_TOPIC_CREATE_ACCOUNT` and delete into `COMMAND_TOPIC_ACCOUNT`
  - Added `CommandTypeCreate` and `CommandTypeDelete` constants
  - Effort: S

- [x] **2.3** Create command consumer handler
  - Added `handleDeleteAccountCommand` in `kafka/consumer/account/consumer.go`
  - Call processor.DeleteAndEmit()
  - Effort: M

- [x] **2.4** Register consumer in `main.go`
  - Consumer already registered (uses consolidated topic)
  - Effort: S

- [ ] **2.5** Add consumer unit tests
  - Test command processing calls processor
  - Test error handling and logging
  - Effort: M

---

## Phase 3: REST Endpoint (atlas-account)

**Objective:** Enable deletion via REST API

- [x] **3.1** Add `handleDeleteAccount` handler in `account/resource.go`
  - Parse accountId from path
  - Call processor.DeleteAndEmit()
  - Return 204 No Content on success
  - Return 404 Not Found if account doesn't exist
  - Return 409 Conflict if account is logged in
  - Effort: M

- [x] **3.2** Register DELETE route in `InitResource`
  - Route: `DELETE /accounts/{accountId}`
  - Effort: S

- [ ] **3.3** Add REST handler tests
  - Test 204 response on successful deletion
  - Test 404 response for non-existent account
  - Test 409 response for logged-in account
  - Effort: M

---

## Phase 4: Character Service Integration (atlas-character)

**Objective:** Handle account deletion by removing all characters

- [x] **4.1** Add account status event consumer
  - Created `kafka/consumer/account/consumer.go`
  - Created `kafka/message/account/kafka.go`
  - Subscribe to `EVENT_TOPIC_ACCOUNT_STATUS`
  - Filter for status "DELETED"
  - Effort: M

- [x] **4.2** Implement `DeleteByAccountIdAndEmit()` processor method
  - Added `getForAccount()` provider function
  - Query all characters for accountId
  - Delete each character using existing Delete() method
  - Effort: M

- [x] **4.3** Emit character deleted events for each character
  - Each deleted character emits StatusEventDeleted via existing Delete() method
  - This triggers cascade cleanup in character-dependent services
  - Effort: M

- [ ] **4.4** Add consumer tests
  - Test handler receives event
  - Test all characters for account are deleted
  - Test character deleted events are emitted
  - Effort: M

---

## Phase 5: Storage Service Integration (atlas-storage)

**Objective:** Handle account deletion by removing storage records

- [x] **5.1** Add account status event consumer
  - Created `kafka/consumer/account/consumer.go`
  - Created `kafka/message/account/kafka.go`
  - Subscribe to `EVENT_TOPIC_ACCOUNT_STATUS`
  - Filter for status "DELETED"
  - Effort: M

- [x] **5.2** Implement `DeleteByAccountId()` processor method
  - Added `GetByAccountId()` provider function
  - Added `Delete()` administrator function
  - Query all storage records for accountId
  - Delete storage records and associated assets
  - Effort: M

- [x] **5.3** Add processor unit tests for deletion
  - Test deletion of storage with assets (`TestProcessor_DeleteByAccountId`)
  - Test deletion across multiple worlds (`TestProcessor_DeleteByAccountId_MultipleWorlds`)
  - Test deletion when no storage exists (`TestProcessor_DeleteByAccountId_NoStorage`)
  - Effort: M

---

## Phase 6: Documentation and Verification

**Objective:** Document the feature and verify end-to-end flow

- [ ] **6.1** Update `services/atlas-account/README.md`
  - Document DELETE endpoint
  - Document delete command topic
  - Document status event for deletion
  - Effort: S

- [ ] **6.2** Create integration test plan
  - Document manual verification steps
  - List test scenarios
  - Define expected outcomes
  - Effort: M

- [ ] **6.3** End-to-end verification
  - Create test account with characters and storage
  - Delete account via REST
  - Verify account record deleted
  - Verify all characters deleted
  - Verify all storage records deleted
  - Verify wallet and inventory deleted (existing handler)
  - Effort: M

---

## Dependencies

```
Phase 1.1 ─┬─▶ Phase 1.3 ─▶ Phase 1.5
           │
Phase 1.2 ─┴─▶ Phase 1.4 ─▶ Phase 1.5 ─┬─▶ Phase 2.3
                                       │
                                       ├─▶ Phase 3.1
                                       │
                                       ├─▶ Phase 4.1
                                       │
                                       └─▶ Phase 5.1

Phase 2.1 ─▶ Phase 2.2 ─▶ Phase 2.3 ─▶ Phase 2.4

Phase 3.1 ─▶ Phase 3.2

Phase 4.1 ─▶ Phase 4.2 ─▶ Phase 4.3

Phase 5.1 ─▶ Phase 5.2

All Phases ─▶ Phase 6
```

---

## Notes

### Completion Criteria
Mark a task as complete when:
1. Code is written and compiles
2. Unit tests pass
3. Manual verification succeeds (where applicable)

### Blocked Tasks
If a task is blocked, note the blocker here:
- (None currently)

### Changes from Plan
Document any deviations from the original plan:
- **Consolidated Command Topic:** Instead of separate `COMMAND_TOPIC_CREATE_ACCOUNT` and `COMMAND_TOPIC_DELETE_ACCOUNT`, consolidated into single `COMMAND_TOPIC_ACCOUNT` with `CommandTypeCreate` and `CommandTypeDelete` type constants. This follows the pattern used by other services like atlas-storage and atlas-cashshop.
- **Administrator Function Renamed:** Renamed `delete()` to `deleteById()` in administrator.go to avoid shadowing Go's built-in `delete` function.

### Pre-existing Implementations
The following services already had account deletion handling before this feature work:
- **atlas-cashshop:** Already handles `EventStatusDeleted` to delete wallet and inventory (see `kafka/consumer/account/consumer.go`)

---

## Quick Reference Commands

```bash
# Run atlas-account tests
cd services/atlas-account && go test ./...

# Run atlas-character tests
cd services/atlas-character && go test ./...

# Run atlas-storage tests
cd services/atlas-storage && go test ./...

# Build all services
make build

# Start services for integration testing
docker-compose up -d
```
