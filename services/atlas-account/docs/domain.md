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
| birthDate | uint32 | Account birth date |
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

### CharacterSlotModel

Immutable domain representation of a per-(account, world) character slot count.

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| accountId | uint32 | Account identifier |
| worldId | byte | World identifier |
| slots | int16 | Character slot count for the account in the world |

## Invariants

- Password is stored as bcrypt hash
- Gender defaults to 0 (Male) or 10 (UI Choose) based on region and version
- An account cannot log in if already logged in via another session
- An account cannot be deleted if currently logged in
- Channel login requires an existing session in transition state
- Logout is blocked for sessions in transition state (State 2)
- PIN and PIC attempt counters reset to 0 on successful entry
- PIN and PIC attempt counters reset to 0 after ban is issued
- Character slot count for an (account, world) pair with no existing row defaults to 4 (DefaultCharacterSlotsPerWorld)
- Character slot count for an (account, world) pair cannot be incremented beyond 12 (MaxCharacterSlotsPerWorld)

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
| ByIdProvider | Provider for account by ID, decorated with session state |
| GetByName | Retrieve account by name |
| ByNameProvider | Provider for account by name, decorated with session state |
| AllProvider | Provider for one page of the tenant's accounts, decorated with session state |
| LoggedInTenantProvider | Retrieve logged-in accounts for tenant |
| GetOrCreate | Retrieve or create account if automatic registration enabled |
| Create | Create new account with hashed password |
| CreateAndEmit | Create account and emit status event |
| Update | Update account attributes (pin, pic, birthDate, tos, pinAttempts, picAttempts, gender) |
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
| GetCharacterSlots | Retrieve character slot count for an (account, world) pair, defaulting to 4 if no row exists |
| IncrementCharacterSlots | Increment character slot count for an (account, world) pair, creating the row on first use; errors if the count is already at the cap of 12 |

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
| ErrCharacterSlotCapReached | Character slot count for an (account, world) pair is already at the maximum (12) |

## Error Codes

| Code | Description |
|------|-------------|
| SYSTEM_ERROR | Internal system error |
| NOT_REGISTERED | Account not found and auto-register disabled |
| DELETED_OR_BLOCKED | Account is banned |
| ALREADY_LOGGED_IN | Account already has active session |
| INCORRECT_PASSWORD | Password validation failed |
| TOO_MANY_ATTEMPTS | Login attempt limit exceeded |
| INVALID_PIN | PIN validation failed |
| INVALID_PIC | PIC validation failed |
