# Saga Pattern Guidelines

## Overview

Sagas coordinate distributed transactions across multiple services using the saga orchestrator. Each saga consists of multiple steps that must either all succeed or all rollback to maintain data consistency.

**Critical Requirement:** Every saga step MUST have a completion mechanism, or the saga will remain stuck in "pending" state forever.

---

## Two Types of Saga Actions

### 1. Asynchronous Actions (Most Common)

Actions that emit commands to other services and wait for response events.

**Pattern:**
```go
// Orchestrator handler sends command
func (h *HandlerImpl) handleAwardMesos(s Saga, st Step[any]) error {
    payload, ok := st.Payload.(AwardMesosPayload)
    if !ok {
        return errors.New("invalid payload")
    }

    // Send command to character service
    err := h.charP.AwardMesosAndEmit(s.TransactionId, ...)
    if err != nil {
        h.logActionError(s, st, err, "Unable to award mesos.")
        return err
    }

    // DO NOT mark complete here - wait for event
    return nil
}

// Consumer marks step complete when event received
func handleMesosChangedEvent(l logrus.FieldLogger, ctx context.Context, e character.StatusEvent[...]) {
    if e.Type != character.StatusEventTypeMesosChanged {
        return
    }

    // Mark saga step as completed
    _ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}
```

**Requirements:**
- Service receives command → processes → emits status event
- Orchestrator has consumer listening for status events
- Consumer calls `saga.NewProcessor(l, ctx).StepCompleted(transactionId, success)`
- Must handle both success and error events

### 2. Synchronous Actions (Rare)

Fire-and-forget commands with no async response expected (e.g., UI commands).

**Pattern:**
```go
func (h *HandlerImpl) handleShowStorage(s Saga, st Step[any]) error {
    payload, ok := st.Payload.(ShowStoragePayload)
    if !ok {
        return errors.New("invalid payload")
    }

    // Send synchronous command
    err := h.storageP.ShowStorageAndEmit(s.TransactionId, ...)
    if err != nil {
        h.logActionError(s, st, err, "Unable to show storage.")
        return err
    }

    // MUST mark complete immediately - no event expected
    _ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId, true)

    return nil
}
```

**Requirements:**
- Handler marks step complete immediately after sending command
- No consumer needed
- Only use for true fire-and-forget operations

---

## Implementation Checklist

When adding a new saga action, verify ALL of these:

### For Async Actions (Command → Event → Complete)

- [ ] **Handler exists** in `saga/handler.go`
  - Action registered in `GetHandler()` switch statement
  - Handler function sends command to target service
  - Handler returns without marking complete

- [ ] **Target service emits event**
  - Service has `*AndEmit()` method that processes and emits event
  - Event includes `TransactionId` field
  - Event type constant defined (e.g., `StatusEventTypeMesosUpdated`)

- [ ] **Orchestrator consumer exists**
  - Consumer file in `kafka/consumer/<service>/consumer.go`
  - `InitConsumers()` subscribes to correct event topic
  - `InitHandlers()` registers event handlers

- [ ] **Event handler marks completion**
  - Handler checks event type
  - Calls `saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)` on success
  - Calls `saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, false)` on error

- [ ] **Error event handling**
  - Service emits error event on failures
  - Consumer has handler for error event type
  - Error handler marks saga step as failed

- [ ] **Consumer registered in main.go**
  - Import added
  - `InitConsumers()` called
  - `InitHandlers()` called

### For Sync Actions (Command → Immediate Complete)

- [ ] **Handler exists** in `saga/handler.go`
  - Action registered in `GetHandler()` switch statement
  - Handler sends command
  - Handler calls `NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId, true)` immediately
  - Returns nil on success

- [ ] **No consumer needed**
  - Verify action is truly synchronous (no response expected)
  - Document why action doesn't need async completion

---

## Verification Steps

### Before Deployment

1. **Check orchestrator consumers directory:**
   ```bash
   ls atlas-saga-orchestrator/kafka/consumer/
   ```
   - Verify consumer exists for your target service
   - If missing, saga steps will be stuck forever

2. **Search for StepCompleted calls:**
   ```bash
   grep -r "StepCompleted.*TransactionId" atlas-saga-orchestrator/
   ```
   - Verify your action's event is handled
   - Ensure both success and failure paths call StepCompleted

3. **Check consumer registration:**
   ```bash
   grep "InitConsumers\|InitHandlers" atlas-saga-orchestrator/main.go
   ```
   - Verify your consumer is initialized

4. **Verify event topic matches:**
   ```bash
   # In target service
   grep "EnvEventTopic\|EVENT_TOPIC" atlas-<service>/kafka/message/

   # In orchestrator consumer
   grep "EnvEventTopic\|EVENT_TOPIC" atlas-saga-orchestrator/kafka/consumer/<service>/
   ```
   - Topics must match exactly

### Testing

Test both success and failure scenarios:

```go
// Test saga completes successfully
func TestStorageDepositSaga_Success(t *testing.T) {
    // Create saga with deposit steps
    // Execute saga
    // Verify steps progress to "completed"
    // Verify saga status is "completed"
}

// Test saga rolls back on failure
func TestStorageDepositSaga_InsufficientFunds(t *testing.T) {
    // Create saga with character having insufficient mesos
    // Execute saga
    // Verify first step fails
    // Verify saga status is "failed"
    // Verify no side effects (character/storage unchanged)
}
```

---

## Common Issues and Solutions

### Issue: Saga Steps Stuck as "Pending"

**Symptoms:**
- Query saga endpoint shows steps remain "pending"
- Services process commands successfully but saga never completes

**Diagnosis:**
```bash
# Check if consumer exists
ls atlas-saga-orchestrator/kafka/consumer/<service>/

# Check if consumer is registered
grep "<service>.Init" atlas-saga-orchestrator/main.go

# Check logs for event consumption
docker logs atlas-saga-orchestrator | grep "transaction_id"
```

**Solutions:**

1. **Missing consumer entirely:**
   - Create consumer file following pattern in `kafka/consumer/compartment/consumer.go`
   - Register in `main.go`

2. **Consumer exists but not registered:**
   - Add import: `"atlas-saga-orchestrator/kafka/consumer/<service>"`
   - Add init calls: `<service>.InitConsumers()` and `<service>.InitHandlers()`

3. **Wrong event topic:**
   - Verify service emits to same topic consumer listens on
   - Check for typos in topic constants

4. **Handler doesn't call StepCompleted:**
   - Add `saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)` to handler

5. **Synchronous action missing immediate completion:**
   - Add `NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId, true)` to handler

### Issue: Saga Doesn't Rollback on Failure

**Symptoms:**
- First step succeeds but second step fails
- First step's changes are not rolled back

**Solutions:**

1. **Service doesn't emit error events:**
   - Add error event emission in service on failures
   - Define error event type constant

2. **Consumer doesn't handle error events:**
   - Add error event handler
   - Call `StepCompleted(e.TransactionId, false)` in error handler

3. **Missing compensation logic:**
   - Verify orchestrator has compensation handlers for your action
   - Check rollback step definitions

---

## Event-Driven Completion Flow

```
┌─────────────┐
│  Initiator  │
│  (Channel)  │
└──────┬──────┘
       │ 1. Create Saga
       ↓
┌─────────────────────┐
│ Saga Orchestrator   │
│                     │
│  Step 1: Pending    │
└──────┬──────────────┘
       │ 2. Send Command
       ↓
┌─────────────────────┐
│  Target Service     │
│  (Character)        │
│                     │
│  Process command    │
└──────┬──────────────┘
       │ 3. Emit Event
       ↓
┌─────────────────────┐
│ Orchestrator        │
│ Consumer            │
│                     │
│ handleEvent()       │
└──────┬──────────────┘
       │ 4. StepCompleted(transactionId, true)
       ↓
┌─────────────────────┐
│ Saga Orchestrator   │
│                     │
│  Step 1: Completed ✓│
│  Step 2: Pending    │
└─────────────────────┘
```

---

## Reference Implementations

### Complete Async Action Example

**Orchestrator Handler:** `atlas-saga-orchestrator/saga/handler.go`
```go
case AwardMesos:
    return h.handleAwardMesos, true
```

**Orchestrator Consumer:** `atlas-saga-orchestrator/kafka/consumer/character/consumer.go`
```go
func handleMesosChangedEvent(...) {
    if e.Type != character.StatusEventTypeMesosChanged {
        return
    }
    _ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}
```

**Service Processor:** `atlas-character/character/processor.go`
```go
func (p *Processor) RequestChangeMeso(...) error {
    // Business logic
    // ...

    // Emit success event
    return p.emitMesosChangedEvent(transactionId, ...)
}
```

### Complete Sync Action Example

**Orchestrator Handler:** `atlas-saga-orchestrator/saga/handler.go`
```go
func (h *HandlerImpl) handleShowStorage(s Saga, st Step[any]) error {
    err := h.storageP.ShowStorageAndEmit(...)
    if err != nil {
        return err
    }

    // Mark complete immediately - no async response
    _ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId, true)
    return nil
}
```

---

## Best Practices

1. **Prefer async actions** - Easier to test and debug
2. **Always emit events** - Even on errors
3. **Include TransactionId** - In all commands and events
4. **Test failure paths** - Not just happy path
5. **Document sync actions** - Why no async completion needed
6. **Check pending sagas** - Query endpoint during development
7. **Add timeouts** - For stuck saga detection (future enhancement)
8. **Use consistent naming** - Event types should be descriptive

---

## Anti-Patterns

❌ **Handler marks step complete without waiting for event**
```go
// WRONG - Async action marked complete immediately
func (h *HandlerImpl) handleAwardMesos(...) error {
    err := h.charP.AwardMesosAndEmit(...)
    _ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId, true) // WRONG!
    return err
}
```

❌ **Sync action doesn't mark complete**
```go
// WRONG - Sync action never marks complete
func (h *HandlerImpl) handleShowStorage(...) error {
    err := h.storageP.ShowStorageAndEmit(...)
    return err // Step stuck forever!
}
```

❌ **Consumer missing for service**
```go
// WRONG - Handler exists but no consumer
case UpdateStorageMesos:
    return h.handleUpdateStorageMesos, true

// Missing: atlas-saga-orchestrator/kafka/consumer/storage/consumer.go
```

❌ **Event doesn't include TransactionId**
```go
// WRONG - Can't correlate event to saga
type StatusEvent struct {
    // Missing: TransactionId uuid.UUID
    Type string
    Body any
}
```

---

## Quick Diagnostic Commands

```bash
# Find all saga actions
grep -r "case.*:" atlas-saga-orchestrator/saga/handler.go | grep "return h.handle"

# Find all orchestrator consumers
ls -1 atlas-saga-orchestrator/kafka/consumer/

# Find all StepCompleted calls
grep -r "StepCompleted" atlas-saga-orchestrator/kafka/consumer/

# Check consumer registration
grep -A 20 "cmf :=" atlas-saga-orchestrator/main.go

# Query pending sagas (when orchestrator running)
curl http://localhost:8080/api/saga-orchestrator/sagas?status=pending
```
