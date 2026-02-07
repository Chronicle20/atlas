# Account Domain

## Responsibility

The account domain manages user account lifecycle including creation, authentication, deletion, session state tracking, account attribute updates, and PIN/PIC attempt tracking with ban enforcement.

## Core Models

### Model

Immutable domain representation of an account.

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uint32 | Account identifier |
| name | string | Account name |
| password | string | Hashed password |
| pin | string | Account PIN |
| pic | string | Account PIC |
| pinAttempts | int | Failed PIN attempt counter |
| picAttempts | int | Failed PIC attempt counter |
| state | State | Current session state |
| gender | byte | Gender value |
| tos | bool | Terms of service acceptance |
| updatedAt | time.Time | Last update timestamp |

### State

Account session state enumeration.

| Value | Name | Description |
|-------|------|-------------|
| 0 | StateNotLoggedIn | Account is not logged in |
| 1 | StateLoggedIn | Account is logged in |
| 2 | StateTransition | Account is transitioning between services |

### AccountKey

Composite key for account identification in the registry.

| Field | Type |
|-------|------|
| Tenant | tenant.Model |
| AccountId | uint32 |

### ServiceKey

Composite key for service session identification.

| Field | Type |
|-------|------|
| SessionId | uuid.UUID |
| Service | Service |

### Service

Service type enumeration.

| Value | Description |
|-------|-------------|
| LOGIN | Login service |
| CHANNEL | Channel service |

## Invariants

- Password is stored as bcrypt hash
- Gender defaults to 0 (Male) or 10 (UI Choose) based on region and version
- An account cannot log in if already logged in via another session
- An account cannot be deleted if currently logged in
- Channel login requires an existing session in transition state
- Logout is blocked for sessions in transition state (State 2)
- PIN and PIC attempt counters reset to 0 on successful entry
- PIN and PIC attempt counters reset to 0 after ban is issued

## State Transitions

```
StateNotLoggedIn -> StateLoggedIn (via Login)
StateLoggedIn -> StateTransition (via Transition)
StateTransition -> StateLoggedIn (via Channel Login)
StateLoggedIn -> StateNotLoggedIn (via Logout)
StateTransition -> StateNotLoggedIn (via Terminate or Expiration)
```

## Processors

### Processor

Primary domain processor providing account operations.

| Method | Description |
|--------|-------------|
| GetById | Retrieve account by ID |
| GetByName | Retrieve account by name |
| GetByTenant | Retrieve all accounts for tenant |
| LoggedInTenantProvider | Retrieve logged-in accounts for tenant |
| GetOrCreate | Retrieve or create account if automatic registration enabled |
| Create | Create new account with hashed password |
| CreateAndEmit | Create account and emit status event |
| Update | Update account attributes (pin, pic, tos, pinAttempts, picAttempts, gender) |
| Delete | Delete account and emit status event |
| DeleteAndEmit | Delete account and emit status event |
| Login | Record login for account and session |
| Logout | Record logout for account and session |
| LogoutAndEmit | Logout and emit status event |
| AttemptLogin | Validate credentials, check ban status, and process login attempt |
| AttemptLoginAndEmit | Attempt login and emit session status event |
| ProgressState | Transition account to specified state |
| ProgressStateAndEmit | Progress state and emit session status event |
| RecordPinAttempt | Record PIN attempt result and enforce limit |
| RecordPinAttemptAndEmit | Record PIN attempt and emit ban command if limit reached |
| RecordPicAttempt | Record PIC attempt result and enforce limit |
| RecordPicAttemptAndEmit | Record PIC attempt and emit ban command if limit reached |

### Registry

In-memory session state registry (singleton).

| Method | Description |
|--------|-------------|
| GetStates | Get all session states for an account |
| MaximalState | Get the minimal state value across sessions |
| IsLoggedIn | Check if account has any active session |
| Login | Record login for service key |
| Transition | Set session to transition state |
| ExpireTransition | Remove expired transition sessions |
| Logout | Remove session for service key |
| Terminate | Remove all sessions for account |
| GetExpiredInTransition | Get accounts with expired transition sessions |
| Tenants | Get all tenants with active sessions |

## Error Types

| Error | Description |
|-------|-------------|
| ErrAccountNotFound | Account does not exist |
| ErrAccountLoggedIn | Account is currently logged in and cannot be deleted |

## Error Codes

| Code | Description |
|------|-------------|
| SYSTEM_ERROR | Internal system error |
| NOT_REGISTERED | Account not found and auto-register disabled |
| DELETED_OR_BLOCKED | Account is banned |
| ALREADY_LOGGED_IN | Account already has active session |
| INCORRECT_PASSWORD | Password validation failed |
| TOO_MANY_ATTEMPTS | Login attempt limit exceeded |
