# Storage

This service does not use persistent database storage.

## In-Memory Registries

### Session Registry

Stores active sessions in memory, keyed by tenant ID and session ID.

| Key | Type | Description |
|-----|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| sessionId | uuid.UUID | Session identifier |

**Value: Session Model**

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Session identifier |
| accountId | uint32 | Associated account identifier |
| worldId | byte | Selected world identifier |
| channelId | byte | Selected channel identifier |
| con | net.Conn | TCP connection |
| send | crypto.AESOFB | Send encryption cipher |
| sendLock | *sync.Mutex | Mutex for send operations |
| recv | crypto.AESOFB | Receive encryption cipher |
| encryptFunc | crypto.EncryptFunc | Encryption function |
| lastPacket | time.Time | Last packet timestamp |
| locale | byte | Client locale |

**Operations**

| Operation | Description |
|-----------|-------------|
| Add | Adds a session to the registry |
| Remove | Removes a session from the registry |
| Get | Retrieves a session by tenant and session ID |
| Update | Updates a session in the registry |
| GetInTenant | Retrieves all sessions for a tenant |

### Account Registry

Stores account login status in memory, keyed by tenant and account ID.

| Key | Type | Description |
|-----|------|-------------|
| Tenant | tenant.Model | Tenant model |
| Id | uint32 | Account identifier |

**Value**

| Type | Description |
|------|-------------|
| bool | Login status (true = logged in) |

**Operations**

| Operation | Description |
|-----------|-------------|
| Init | Initializes registry with account login states |
| Login | Marks an account as logged in |
| Logout | Marks an account as logged out |
| LoggedIn | Checks if an account is logged in |

## Tables

None.

## Relationships

None.

## Indexes

None.

## Migration Rules

None.
