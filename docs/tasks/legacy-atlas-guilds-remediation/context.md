# Atlas-Guilds Remediation - Context

**Last Updated:** 2026-01-13 (Revision 2)

---

## Audit Artifacts

- `dev/audits/atlas-guilds/audit.md` - Detailed audit findings
- `dev/audits/atlas-guilds/audit.json` - Machine-readable audit data

---

## Issues to Remediate

### P1 Issues (Medium Impact)

| Check ID | Issue | Files Affected |
|----------|-------|----------------|
| ARCH-010 | Model.Builder() returns pointers to internal state | 5 builder.go files |

### P2 Issues (Low Impact)

| Check ID | Issue | Files Affected |
|----------|-------|----------------|
| ARCH-003 | Missing provider.go files | member, title, reply packages |
| REST-002 | Missing JSON:API interface methods | member, title, reply rest.go files |
| STRUCT-002 | Empty administrator.go file | guild/character/administrator.go |

---

## Reference Implementations

### Builder Pattern
- `services/atlas-guilds/atlas.com/guilds/guild/builder.go` - Guild builder
- `services/atlas-guilds/atlas.com/guilds/guild/builder_test.go` - Builder tests

### Provider Pattern
- `services/atlas-guilds/atlas.com/guilds/guild/provider.go` - Provider implementation

### Existing Mocks
- `services/atlas-guilds/atlas.com/guilds/character/mock/processor.go` - Character mock
- `services/atlas-guilds/atlas.com/guilds/party/mock/processor.go` - Party mock

---

## Key Source Files

### Builder Files to Modify (Phase 1)

| File | Model.Builder() Location | Field Count |
|------|-------------------------|-------------|
| `guild/builder.go` | ~line 180 | ~15 fields |
| `guild/member/builder.go` | ~line 122 | ~9 fields |
| `guild/title/builder.go` | TBD | ~4 fields |
| `thread/builder.go` | TBD | ~10 fields |
| `thread/reply/builder.go` | TBD | ~5 fields |

### Existing Test Files

| File | Status | Notes |
|------|--------|-------|
| `guild/builder_test.go` | Exists | Add immutability test |
| `guild/processor_test.go` | Exists | Comprehensive tests |
| `guild/member/builder_test.go` | Exists | Add immutability test |
| `guild/member/processor_test.go` | Exists | Comprehensive tests |
| `guild/title/builder_test.go` | Exists | Add immutability test |
| `guild/title/processor_test.go` | Exists | Comprehensive tests |
| `thread/builder_test.go` | Exists | Add immutability test |
| `thread/processor_test.go` | Exists | Comprehensive tests |
| `thread/reply/builder_test.go` | Exists | Add immutability test |
| `thread/reply/processor_test.go` | Exists | Comprehensive tests |

---

## Files to Create (Phase 2)

### Provider Files

| File | Status | Notes |
|------|--------|-------|
| `guild/member/provider.go` | Pending | getByGuildId, getById |
| `guild/title/provider.go` | Pending | getByGuildId |
| `thread/reply/provider.go` | Pending | getByThreadId |

---

## Files to Modify/Remove (Phase 3)

| File | Action | Notes |
|------|--------|-------|
| `guild/character/administrator.go` | Remove | Empty file (package declaration only) |
| `guild/member/rest.go` | Optional | Add JSON:API methods |
| `guild/title/rest.go` | Optional | Add JSON:API methods |
| `thread/reply/rest.go` | Optional | Add JSON:API methods |

## Key Decisions

### Decision 1: Model.Builder() Fix Approach
**Decision:** Create local value copies before assigning to Builder pointers
**Rationale:** Simple, non-breaking change that preserves existing behavior while fixing immutability

### Decision 2: Empty Administrator File
**Decision:** Remove `guild/character/administrator.go`
**Rationale:** File contains only package declaration; no actual functions

### Decision 3: JSON:API Methods for Nested Models
**Decision:** Optional - only implement if needed for standalone resource use
**Rationale:** `member.RestModel`, `title.RestModel`, and `reply.RestModel` are embedded-only

### Decision 4: Provider Files
**Decision:** Add for consistency, but don't modify existing preloading behavior
**Rationale:** Provider files complement GORM preloading, don't replace it

---

## Processor Interface Reference

### guild.Processor (guild/processor.go)

```go
type Processor interface {
    // Transaction handling
    WithTransaction(tx *gorm.DB) Processor

    // Read operations - PRIORITY 1 for testing
    AllProvider() model.Provider[[]Model]
    ByIdProvider(guildId uint32) model.Provider[Model]
    ByNameProvider(worldId world.Id, name string) model.Provider[Model]
    GetSlice(filters ...model.Filter[Model]) ([]Model, error)
    GetById(guildId uint32) (Model, error)
    GetByName(worldId world.Id, name string) (Model, error)
    GetByMemberId(memberId uint32) (Model, error)

    // Write operations - PRIORITY 2 for testing (require mocks)
    RequestCreate(mb *message.Buffer) func(characterId uint32) func(field field.Model) func(name string) func(transactionId uuid.UUID) error
    RequestCreateAndEmit(characterId uint32, field field.Model, name string, transactionId uuid.UUID) error
    Create(mb *message.Buffer) func(worldId byte) func(leaderId uint32) func(name string) (Model, error)
    CreateAndEmit(worldId byte, leaderId uint32, name string) (Model, error)
    CreationAgreementResponse(mb *message.Buffer) func(characterId uint32) func(agreed bool) func(transactionId uuid.UUID) error
    CreationAgreementResponseAndEmit(characterId uint32, agreed bool, transactionId uuid.UUID) error
    ChangeEmblem(mb *message.Buffer) func(guildId uint32) func(characterId uint32) func(logo uint16) func(logoColor byte) func(logoBackground uint16) func(logoBackgroundColor byte) func(transactionId uuid.UUID) error
    ChangeEmblemAndEmit(...) error
    UpdateMemberOnline(mb *message.Buffer) func(characterId uint32) func(online bool) func(transactionId uuid.UUID) error
    UpdateMemberOnlineAndEmit(characterId uint32, online bool, transactionId uuid.UUID) error
    ChangeNotice(mb *message.Buffer) func(guildId uint32) func(characterId uint32) func(notice string) func(transactionId uuid.UUID) error
    ChangeNoticeAndEmit(...) error
    Leave(mb *message.Buffer) func(guildId uint32) func(characterId uint32) func(force bool) func(transactionId uuid.UUID) error
    LeaveAndEmit(...) error
    RequestInvite(mb *message.Buffer) func(guildId uint32) func(characterId uint32) func(targetId uint32) error
    RequestInviteAndEmit(...) error
    Join(mb *message.Buffer) func(guildId uint32) func(characterId uint32) func(transactionId uuid.UUID) error
    JoinAndEmit(...) error
    ChangeTitles(mb *message.Buffer) func(guildId uint32) func(characterId uint32) func(titles []string) func(transactionId uuid.UUID) error
    ChangeTitlesAndEmit(...) error
    ChangeMemberTitle(mb *message.Buffer) func(guildId uint32) func(characterId uint32) func(targetId uint32) func(title byte) func(transactionId uuid.UUID) error
    ChangeMemberTitleAndEmit(...) error
    RequestDisband(mb *message.Buffer) func(characterId uint32) func(transactionId uuid.UUID) error
    RequestDisbandAndEmit(characterId uint32, transactionId uuid.UUID) error
    RequestCapacityIncrease(mb *message.Buffer) func(characterId uint32) func(transactionId uuid.UUID) error
    RequestCapacityIncreaseAndEmit(characterId uint32, transactionId uuid.UUID) error
}
```

### member.Processor (guild/member/processor.go)

```go
type Processor interface {
    AddMember(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte) (Model, error)
    RemoveMember(guildId uint32, characterId uint32) error
    UpdateStatus(characterId uint32, online bool) error
    UpdateTitle(characterId uint32, title byte) error
}
```

### thread.Processor (thread/processor.go)

```go
type Processor interface {
    WithTransaction(tx *gorm.DB) Processor
    AllProvider(guildId uint32) model.Provider[[]Model]
    GetAll(guildId uint32) ([]Model, error)
    ByIdProvider(guildId uint32, threadId uint32) model.Provider[Model]
    GetById(guildId uint32, threadId uint32) (Model, error)

    Create(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(posterId uint32) func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error)
    CreateAndEmit(...) (Model, error)
    Update(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(threadId uint32) func(posterId uint32) func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error)
    UpdateAndEmit(...) (Model, error)
    Delete(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(threadId uint32) func(actorId uint32) error
    DeleteAndEmit(...) error
    Reply(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(threadId uint32) func(posterId uint32) func(message string) (Model, error)
    ReplyAndEmit(...) (Model, error)
    DeleteReply(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(threadId uint32) func(actorId uint32) func(replyId uint32) (Model, error)
    DeleteReplyAndEmit(...) (Model, error)
}
```

---

## Test Setup Pattern (from atlas-fame)

```go
func setupTestLogger(t *testing.T) logrus.FieldLogger {
    t.Helper()
    l, _ := test.NewNullLogger()
    return l
}

func setupTestTenant(t *testing.T) tenant.Model {
    t.Helper()
    ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
    if err != nil {
        t.Fatalf("Failed to create tenant: %v", err)
    }
    return ten
}

func setupTestContext(t *testing.T, ten tenant.Model) context.Context {
    t.Helper()
    return tenant.WithContext(context.Background(), ten)
}

func setupTestDatabase(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Silent),
    })
    if err != nil {
        t.Fatalf("Failed to connect to database: %v", err)
    }

    // Run migrations for all entities
    if err = guild.Migration(db); err != nil {
        t.Fatalf("Failed to migrate database: %v", err)
    }
    // ... additional migrations

    return db
}
```

---

## Entity Model Reference

### Guild Entity (guild/entity.go)

```go
type Entity struct {
    TenantId            uuid.UUID        `gorm:"not null"`
    Id                  uint32           `gorm:"primaryKey;autoIncrement:true"`
    WorldId             byte             `gorm:"not null"`
    Name                string           `gorm:"not null"`
    Notice              string
    Points              uint32
    Capacity            uint32           `gorm:"not null;default:30"`
    Logo                uint16
    LogoColor           byte
    LogoBackground      uint16
    LogoBackgroundColor byte
    LeaderId            uint32           `gorm:"not null"`
    Members             []member.Entity  `gorm:"foreignKey:TenantId,GuildId;references:TenantId,Id"`
    Titles              []title.Entity   `gorm:"foreignKey:TenantId,GuildId;references:TenantId,Id"`
}
```

### Thread Entity (thread/entity.go)

```go
type Entity struct {
    TenantId   uuid.UUID       `gorm:"not null"`
    GuildId    uint32          `gorm:"not null"`
    Id         uint32          `gorm:"primaryKey;autoIncrement:true"`
    PosterId   uint32          `gorm:"not null"`
    Title      string          `gorm:"not null"`
    Message    string
    EmoticonId uint32
    Notice     bool
    CreatedAt  time.Time
    Replies    []reply.Entity  `gorm:"foreignKey:TenantId,ThreadId;references:TenantId,Id"`
}
```

---

## Kafka Topics Reference

### Consumed Topics

| Topic Env Var | Handler |
|--------------|---------|
| `COMMAND_TOPIC_GUILD` | `kafka/consumer/guild/consumer.go` |
| `COMMAND_TOPIC_THREAD` | `kafka/consumer/thread/consumer.go` |
| `COMMAND_TOPIC_INVITE` | `kafka/consumer/invite/consumer.go` |
| `STATUS_EVENT_TOPIC_CHARACTER` | `kafka/consumer/character/consumer.go` |

### Produced Topics

| Topic Env Var | Producer |
|--------------|----------|
| `STATUS_EVENT_TOPIC_GUILD` | `guild/producer.go` |
| `STATUS_EVENT_TOPIC_THREAD` | `thread/producer.go` |
| `COMMAND_TOPIC_INVITE` | `invite/producer.go` |
