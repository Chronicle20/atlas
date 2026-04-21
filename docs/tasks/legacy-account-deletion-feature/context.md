# Account Deletion Feature - Context Document

**Last Updated:** 2026-02-03
**Related Plan:** `plan.md`

---

## Key Files Reference

### atlas-account Service

| File | Purpose | Key Functions/Structures |
|------|---------|--------------------------|
| `services/atlas-account/atlas.com/account/account/entity.go` | Database entity | `Entity` struct with TenantId, ID, Name, Password |
| `services/atlas-account/atlas.com/account/account/model.go` | Domain model | `Model` struct with state tracking |
| `services/atlas-account/atlas.com/account/account/processor.go` | Core business logic | Create, Update, Login, Logout, GetById |
| `services/atlas-account/atlas.com/account/account/administrator.go` | DB write operations | create(), update() functions |
| `services/atlas-account/atlas.com/account/account/producer.go` | Event emission | Event provider functions |
| `services/atlas-account/atlas.com/account/account/provider.go` | DB read operations | Entity query providers |
| `services/atlas-account/atlas.com/account/account/resource.go` | REST handlers | HTTP endpoint handlers |
| `services/atlas-account/atlas.com/account/account/registry.go` | Session state | In-memory login state tracking |
| `services/atlas-account/atlas.com/account/kafka/message/account/kafka.go` | Kafka messages | Commands, events, topics |
| `services/atlas-account/atlas.com/account/kafka/consumer/account/consumer.go` | Kafka handlers | Command processing |
| `services/atlas-account/main.go` | Service entry | Consumer registration |

### atlas-character Service

| File | Purpose | Key Functions/Structures |
|------|---------|--------------------------|
| `services/atlas-character/atlas.com/character/character/entity.go` | Database entity | `Entity` with AccountId field |
| `services/atlas-character/atlas.com/character/character/processor.go` | Core logic | Delete operations |
| `services/atlas-character/atlas.com/character/character/administrator.go` | DB writes | delete() function |
| `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go` | Kafka messages | StatusEventDeleted |

### atlas-storage Service

| File | Purpose | Key Functions/Structures |
|------|---------|--------------------------|
| `services/atlas-storage/atlas.com/storage/storage/entity.go` | Database entity | `Entity` with AccountId field |
| `services/atlas-storage/atlas.com/storage/storage/processor.go` | Core logic | Storage operations |
| `services/atlas-storage/atlas.com/storage/storage/administrator.go` | DB writes | Write operations |

### atlas-cashshop Service (Reference - Already Has Handler)

| File | Purpose | Key Functions/Structures |
|------|---------|--------------------------|
| `services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/account/consumer.go` | Account event handler | Handles DELETED status |
| `services/atlas-cashshop/atlas.com/cashshop/wallet/processor.go` | Wallet operations | DeleteAndEmit() |
| `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/processor.go` | Inventory operations | DeleteByAccountId() |

---

## Existing Patterns to Follow

### Event Status Pattern (from kafka/message/account/kafka.go)

```go
const (
    EventStatusCreated   = "CREATED"
    EventStatusLoggedIn  = "LOGGED_IN"
    EventStatusLoggedOut = "LOGGED_OUT"
    // Add: EventStatusDeleted = "DELETED"
)

type StatusEvent struct {
    AccountId uint32 `json:"account_id"`
    Name      string `json:"name"`
    Status    string `json:"status"`
}
```

### Administrator Delete Pattern (from atlas-character)

```go
func delete(db *gorm.DB) IdOperator {
    return func(ctx context.Context, tenantId uuid.UUID, id uint32) error {
        return db.WithContext(ctx).
            Where(&Entity{TenantId: tenantId, ID: id}).
            Delete(&Entity{}).
            Error
    }
}
```

### Processor Method Pattern (from account/processor.go)

```go
func (p *Processor) Delete(ctx context.Context, accountId uint32) error {
    // 1. Get account to verify exists
    // 2. Check not logged in
    // 3. Delete from database
    return p.deleteFunc(ctx, tenant.MustFromContext(ctx), accountId)
}

func (p *Processor) DeleteAndEmit(ctx context.Context, accountId uint32) error {
    // 1. Get account details for event
    // 2. Delete account
    // 3. Emit deleted event
    return nil
}
```

### REST Handler Pattern (from account/resource.go)

```go
func handleDeleteAccount(p *Processor) rest.HandlerFunc {
    return func(d rest.HandlerData) rest.HandlerResult {
        accountId, err := strconv.ParseUint(d.PathParam("accountId"), 10, 32)
        if err != nil {
            return d.BadRequest(err)
        }

        err = p.DeleteAndEmit(d.Context(), uint32(accountId))
        if err != nil {
            // Handle not found, conflict, etc.
            return d.InternalServerError(err)
        }

        return d.NoContent()
    }
}
```

### Kafka Command Pattern (from kafka/consumer/account/consumer.go)

```go
type DeleteCommand struct {
    AccountId uint32 `json:"accountId"`
}

func handleDeleteAccountCommand(p *account.Processor) message.Handler[DeleteCommand] {
    return func(l logrus.FieldLogger, ctx context.Context, c DeleteCommand) {
        err := p.DeleteAndEmit(ctx, c.AccountId)
        if err != nil {
            l.WithError(err).Errorf("Unable to delete account [%d].", c.AccountId)
        }
    }
}
```

### Account Event Consumer Pattern (from atlas-cashshop)

```go
func handleAccountStatusEvent(wp *wallet.Processor, ip *inventory.Processor) message.Handler[account.StatusEvent] {
    return func(l logrus.FieldLogger, ctx context.Context, e account.StatusEvent) {
        switch e.Status {
        case account.EventStatusCreated:
            // Create resources
        case account.EventStatusDeleted:
            // Delete resources
            err := wp.DeleteAndEmit(ctx, e.AccountId)
            err = ip.DeleteByAccountIdAndEmit(ctx, e.AccountId)
        }
    }
}
```

---

## Database Schema References

### accounts table (atlas-account)

```sql
CREATE TABLE accounts (
    tenant_id UUID NOT NULL,
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    pin VARCHAR(255),
    pic VARCHAR(255),
    gender SMALLINT DEFAULT 10,
    tos BOOLEAN DEFAULT FALSE,
    last_login BIGINT DEFAULT 0,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    UNIQUE(tenant_id, name)
);
```

### characters table (atlas-character)

```sql
CREATE TABLE characters (
    tenant_id UUID NOT NULL,
    id SERIAL PRIMARY KEY,
    account_id INT NOT NULL,  -- Foreign key to accounts
    world SMALLINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    -- ... other fields
    UNIQUE(tenant_id, name)
);
```

### storages table (atlas-storage)

```sql
CREATE TABLE storages (
    tenant_id UUID NOT NULL,
    id UUID PRIMARY KEY,
    account_id INT NOT NULL,  -- Foreign key to accounts
    world_id SMALLINT NOT NULL,
    capacity INT DEFAULT 4,
    mesos BIGINT DEFAULT 0,
    UNIQUE(tenant_id, account_id, world_id)
);
```

---

## Kafka Topic Configuration

### Existing Topics

| Topic | Purpose | Message Type |
|-------|---------|--------------|
| `COMMAND_TOPIC_CREATE_ACCOUNT` | Create account command | CreateCommand |
| `COMMAND_TOPIC_ACCOUNT_SESSION` | Session operations | SessionCommand |
| `EVENT_TOPIC_ACCOUNT_STATUS` | Account lifecycle events | StatusEvent |
| `EVENT_TOPIC_ACCOUNT_SESSION_STATUS` | Session state events | SessionStatusEvent |

### New Topics

| Topic | Purpose | Message Type |
|-------|---------|--------------|
| `COMMAND_TOPIC_DELETE_ACCOUNT` | Delete account command | DeleteCommand |

---

## Service Dependencies Graph

```
Account Deletion Event Flow:

atlas-account
     │
     │ StatusEvent{status: "DELETED"}
     │
     ├──────────────────┬───────────────────┬──────────────────┐
     ▼                  ▼                   ▼                  ▼
atlas-character    atlas-storage     atlas-cashshop      atlas-login
(delete chars)     (delete storage)  (delete wallet)     (cleanup)
     │
     │ CharacterStatusEvent{status: "DELETED"} (per character)
     │
     ├─────┬─────┬─────┬─────┬─────┬─────┬─────┐
     ▼     ▼     ▼     ▼     ▼     ▼     ▼     ▼
buddies  inv  skills pets notes keys  guilds  etc.
```

---

## Important Architectural Decisions

### 1. Event-Driven Cascade
Deletion in dependent services happens asynchronously via Kafka events. This provides:
- Loose coupling between services
- Resilience (services can be temporarily unavailable)
- Scalability (parallel processing)

### 2. No Foreign Key Constraints
Atlas uses soft references (account_id stored as integer) rather than database foreign keys. This:
- Allows services to run on separate databases
- Requires explicit cascade cleanup via events
- May leave orphaned data if events are lost

### 3. Multi-Tenant Isolation
All operations include tenant_id filtering. Deletion events must include tenant context for proper isolation.

### 4. Registry State Management
The account registry tracks login state in memory. Deletion should:
- Check registry state before deletion
- Clean up registry entry on deletion

---

## Error Handling Strategy

### Account Not Found
- Return 404 from REST
- Log warning from Kafka handler
- Skip processing (idempotent)

### Account Logged In
- Return 409 Conflict from REST
- Reject from Kafka handler with error log
- Alternative: Force logout first (configurable)

### Database Error
- Return 500 from REST
- Retry logic in Kafka consumer
- Log error with context

### Event Emission Failure
- Kafka producer handles retries
- Consider outbox pattern for reliability (future)

---

## Testing Strategy

### Unit Tests
- `processor_test.go`: Delete(), DeleteAndEmit()
- `administrator_test.go`: delete() function
- `consumer_test.go`: Command handler

### Integration Tests
- REST endpoint returns correct status codes
- Kafka command triggers deletion
- Event emitted to topic

### End-to-End Tests
- Create account with characters
- Delete account
- Verify cascade cleanup in all services
- Verify no orphaned data

---

## Rollback Plan

If issues are discovered post-deployment:

1. **Disable REST endpoint** - Remove route registration
2. **Disable Kafka consumer** - Comment out consumer start
3. **Revert event emission** - Stop emitting DELETED events
4. **Data recovery** - Use database backups if needed

Note: Event-driven cascade means dependent service handlers can be disabled independently.
