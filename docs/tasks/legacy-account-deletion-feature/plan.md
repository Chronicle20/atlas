# Account Deletion Feature Plan

**Last Updated:** 2026-02-03
**Status:** PLANNING
**Service Path:** `services/atlas-account`
**Related Services:** `atlas-character`, `atlas-storage`, `atlas-cashshop`

---

## Executive Summary

This plan implements account deletion functionality for the `atlas-account` service. The feature enables account deletion through both REST API and Kafka command interfaces. When an account is deleted, the system will:

1. Clean up the account record from the database
2. Emit an `account.StatusEventDeleted` event
3. Trigger cascading cleanup in dependent services (characters, storage, cash shop inventory)

The architecture follows the existing event-driven patterns in Atlas, where dependent services subscribe to account status events and handle their own cleanup.

**Key Deliverables:**
- REST endpoint: `DELETE /api/accounts/{accountId}`
- Kafka command: `COMMAND_TOPIC_DELETE_ACCOUNT`
- Account status event: `EventStatusDeleted`
- Cascading cleanup via existing event consumers

---

## Current State Analysis

### Existing Account Operations
- **Create:** REST POST + Kafka command, emits `StatusEventCreated`
- **Update:** REST PATCH (PIN, PIC, TOS, Gender)
- **Login/Logout:** Kafka command, emits `StatusEventLoggedIn`/`StatusEventLoggedOut`
- **Delete:** NOT IMPLEMENTED

### Existing Event Infrastructure
The account service already emits status events for account lifecycle:
- `EventStatusCreated` = "CREATED"
- `EventStatusLoggedIn` = "LOGGED_IN"
- `EventStatusLoggedOut` = "LOGGED_OUT"

Missing: `EventStatusDeleted` = "DELETED"

### Dependent Services (Direct Account References)

| Service | Table | Field | Cleanup Strategy |
|---------|-------|-------|------------------|
| atlas-character | `characters` | `AccountId` | Delete all characters (triggers character deletion cascade) |
| atlas-storage | `storages` | `AccountId` | Delete storage records |
| atlas-cashshop | `accounts` (wallet) | `AccountId` | Delete wallet (already has handler) |
| atlas-cashshop | `cash_compartments` | `AccountId` | Delete compartments (already has handler) |

### Services with Existing Deletion Handlers
- **atlas-cashshop:** Already handles `account.StatusEventDeleted` for wallet and inventory deletion

### Services Needing Deletion Handlers
- **atlas-character:** Needs handler to delete all characters for account
- **atlas-storage:** Needs handler to delete storage records for account

---

## Proposed Future State

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Account Deletion Flow                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  REST: DELETE /api/accounts/{id}                                     │
│            │                                                         │
│            ▼                                                         │
│  ┌─────────────────┐    Kafka Command    ┌──────────────────┐       │
│  │ account/resource│───────────────────▶│ account/consumer │       │
│  └─────────────────┘                     └────────┬─────────┘       │
│                                                   │                  │
│                                                   ▼                  │
│                                          ┌───────────────┐           │
│                                          │   processor   │           │
│                                          │   Delete()    │           │
│                                          └───────┬───────┘           │
│                                                  │                   │
│                           ┌──────────────────────┼──────────────┐    │
│                           │                      │              │    │
│                           ▼                      ▼              ▼    │
│                    ┌────────────┐    ┌──────────────┐  ┌────────────┐│
│                    │ Delete DB  │    │ Emit Status  │  │ Logout if  ││
│                    │   Record   │    │    Event     │  │ Logged In  ││
│                    └────────────┘    └──────┬───────┘  └────────────┘│
│                                             │                        │
│                    EVENT_TOPIC_ACCOUNT_STATUS                        │
│                    { status: "DELETED" }                             │
│                                             │                        │
│              ┌──────────────┬───────────────┼───────────────┐        │
│              ▼              ▼               ▼               ▼        │
│       ┌────────────┐ ┌────────────┐ ┌─────────────┐ ┌────────────┐   │
│       │  character │ │  storage   │ │  cashshop   │ │   login    │   │
│       │ (delete    │ │ (delete    │ │ (delete     │ │ (cleanup   │   │
│       │  all chars)│ │  storages) │ │  wallet/inv)│ │  session)  │   │
│       └─────┬──────┘ └────────────┘ └─────────────┘ └────────────┘   │
│             │                                                        │
│             ▼                                                        │
│    CHARACTER STATUS EVENT { status: "DELETED" }                      │
│             │                                                        │
│    ┌────────┼────────┬────────┬────────┬────────┐                    │
│    ▼        ▼        ▼        ▼        ▼        ▼                    │
│ buddies inventory  skills   pets   notes    keys   (etc.)            │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### New Components

**atlas-account:**
- `DELETE /api/accounts/{accountId}` - REST endpoint
- `DeleteCommand` - Kafka command message
- `EventStatusDeleted` - Status event constant
- `Delete()` / `DeleteAndEmit()` - Processor methods
- `delete()` - Administrator function
- `deletedEventProvider()` - Event producer

**atlas-character:**
- Account deletion handler - Deletes all characters for account

**atlas-storage:**
- Account deletion handler - Deletes all storage records for account

---

## Implementation Phases

### Phase 1: Core Account Deletion (atlas-account)
**Objective:** Implement deletion logic in the account service

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 1.1 | Add `EventStatusDeleted` constant | S | `kafka/message/account/kafka.go` has "DELETED" constant |
| 1.2 | Add `delete()` administrator function | S | `administrator.go` has delete function using GORM |
| 1.3 | Add `deletedEventProvider()` producer | S | `producer.go` emits status event with "DELETED" |
| 1.4 | Add `Delete()` processor method | M | Validates account exists, not logged in, deletes record |
| 1.5 | Add `DeleteAndEmit()` processor method | S | Calls Delete() then emits event |
| 1.6 | Add processor unit tests | M | Test delete success, not found, logged in rejection |

### Phase 2: Kafka Command Handler (atlas-account)
**Objective:** Enable deletion via Kafka command

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 2.1 | Define `DeleteCommand` message | S | `kafka/message/account/kafka.go` has command struct |
| 2.2 | Add command topic constant | S | `COMMAND_TOPIC_DELETE_ACCOUNT` defined |
| 2.3 | Create command consumer | M | `kafka/consumer/account/delete_consumer.go` handles command |
| 2.4 | Register consumer in main.go | S | Consumer started on service initialization |
| 2.5 | Add consumer unit tests | M | Test command processing and error handling |

### Phase 3: REST Endpoint (atlas-account)
**Objective:** Enable deletion via REST API

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 3.1 | Add DELETE handler function | M | `handleDeleteAccount` in resource.go |
| 3.2 | Register DELETE route | S | Route registered in InitResource |
| 3.3 | Add REST handler tests | M | Test 204 success, 404 not found, 409 conflict (logged in) |

### Phase 4: Character Service Integration (atlas-character)
**Objective:** Handle account deletion by removing all characters

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 4.1 | Add account status event consumer | M | New consumer file or extend existing |
| 4.2 | Implement deletion handler | M | Deletes all characters for accountId |
| 4.3 | Emit character deleted events | M | Each character deletion emits status event |
| 4.4 | Add consumer tests | M | Test cascade deletion |

### Phase 5: Storage Service Integration (atlas-storage)
**Objective:** Handle account deletion by removing storage records

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 5.1 | Add account status event consumer | M | New consumer for account events |
| 5.2 | Implement deletion handler | M | Deletes all storage records for accountId |
| 5.3 | Add consumer tests | M | Test storage cleanup |

### Phase 6: Documentation and Verification
**Objective:** Document the feature and verify end-to-end flow

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 6.1 | Update atlas-account README | S | Document new endpoints and commands |
| 6.2 | Create integration test plan | M | Document manual verification steps |
| 6.3 | Verify cascade deletion | M | End-to-end test with account having characters |

---

## Risk Assessment and Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Deleting logged-in account causes inconsistent state | High | High | Check login state before deletion, reject if logged in |
| Orphaned data in services without handlers | Medium | Medium | Ensure all dependent services have handlers before deployment |
| Character deletion cascade fails mid-way | Low | High | Use transaction for account deletion; character cascade is eventually consistent |
| Performance impact on large accounts | Low | Medium | Deletion is async via events; services handle independently |
| Accidental deletion | Medium | High | REST endpoint should require confirmation (future: soft delete) |

### Edge Cases to Handle

1. **Account is currently logged in**
   - Reject deletion with 409 Conflict
   - Alternative: Force logout then delete (configurable)

2. **Account has active characters in-game**
   - Account login state check covers this (characters require logged-in account)

3. **Deletion event lost before processing**
   - Use Kafka persistence and at-least-once delivery
   - Services should be idempotent on deletion

4. **Concurrent delete requests**
   - First request succeeds, subsequent return 404

---

## Success Metrics

| Metric | Target |
|--------|--------|
| REST endpoint responds correctly | DELETE returns 204/404/409 appropriately |
| Kafka command processed | Command results in deletion and event emission |
| Account record removed | Database query returns no record after deletion |
| Status event emitted | Kafka topic contains DELETED event |
| Characters cascade deleted | All account characters deleted via events |
| Storage cascade deleted | All account storage records deleted via events |
| Cash shop cascade deleted | Wallet and inventory deleted (existing handler) |

---

## Required Resources and Dependencies

### Files to Modify (atlas-account)

| File | Changes |
|------|---------|
| `kafka/message/account/kafka.go` | Add EventStatusDeleted, DeleteCommand, topic constant |
| `account/administrator.go` | Add delete() function |
| `account/producer.go` | Add deletedEventProvider() |
| `account/processor.go` | Add Delete(), DeleteAndEmit() methods |
| `account/resource.go` | Add handleDeleteAccount, register route |
| `main.go` | Register delete command consumer |

### Files to Create (atlas-account)

| File | Purpose |
|------|---------|
| `kafka/consumer/account/delete_consumer.go` | Command handler (or extend existing consumer.go) |
| `account/processor_test.go` | Additional test cases |

### Files to Modify (atlas-character)

| File | Changes |
|------|---------|
| `kafka/consumer/account/consumer.go` | Add/create consumer for account events |
| `character/processor.go` | Add DeleteByAccountId() method |

### Files to Modify (atlas-storage)

| File | Changes |
|------|---------|
| `kafka/consumer/account/consumer.go` | Create consumer for account events |
| `storage/processor.go` | Add DeleteByAccountId() method |

### External Dependencies
- None (all changes internal to Atlas services)

### Reference Materials
- Existing deletion patterns in atlas-cashshop account consumer
- Character deletion event handling patterns
- REST handler patterns in resource.go

---

## API Specification

### REST Endpoint

```
DELETE /api/accounts/{accountId}

Path Parameters:
  accountId (uint32) - The account ID to delete

Responses:
  204 No Content - Account successfully deleted
  404 Not Found - Account does not exist
  409 Conflict - Account is currently logged in

Headers:
  X-Tenant-ID: <uuid> - Required tenant identifier
```

### Kafka Command

```json
Topic: COMMAND_TOPIC_DELETE_ACCOUNT

Message:
{
  "accountId": 12345
}
```

### Status Event

```json
Topic: EVENT_TOPIC_ACCOUNT_STATUS

Message:
{
  "account_id": 12345,
  "name": "account_name",
  "status": "DELETED"
}
```

---

## Execution Order

1. **Phase 1** - Core deletion logic (no external dependencies)
2. **Phase 2** - Kafka command (depends on Phase 1)
3. **Phase 3** - REST endpoint (depends on Phase 1)
4. **Phase 4** - Character integration (depends on Phase 1 event emission)
5. **Phase 5** - Storage integration (depends on Phase 1 event emission)
6. **Phase 6** - Documentation (after all implementation complete)

Phases 2-3 can be done in parallel.
Phases 4-5 can be done in parallel, but after Phase 1.

---

## Notes and Decisions

### Hard Delete vs Soft Delete
This implementation performs a **hard delete** (permanent removal). A soft delete pattern (setting a `deleted_at` timestamp) could be considered for:
- Account recovery requests
- Audit trail requirements
- GDPR compliance (delayed deletion)

**Decision:** Start with hard delete for simplicity. Soft delete can be added as a future enhancement if needed.

### Forced Logout Before Delete
Current design rejects deletion if account is logged in. An alternative is to force logout first, then delete.

**Decision:** Reject if logged in. Users must explicitly logout before deletion. This prevents accidental data loss during active sessions.

### Synchronous vs Asynchronous Deletion
- Account deletion in database: Synchronous
- Cascade cleanup in other services: Asynchronous (event-driven)

**Decision:** This matches existing Atlas patterns and provides loose coupling between services.
