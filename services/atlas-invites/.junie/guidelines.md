# Atlas Microservice Architecture and Style Guide

## Table of Contents
1. [Language Conventions and Dependencies](#language-conventions-and-dependencies)
2. [Project Layout and Structure](#project-layout-and-structure)
3. [Domain Modeling](#domain-modeling)
4. [Design Patterns](#design-patterns)
5. [API Conventions](#api-conventions)
6. [Kafka and Messaging](#kafka-and-messaging)
7. [Logging and Observability](#logging-and-observability)
8. [Testing Conventions](#testing-conventions)

## Language Conventions and Dependencies

### Go Version
- Go 1.24.2 (latest)

### Key Dependencies
- **Database**: gorm.io/gorm with PostgreSQL and SQLite drivers
- **Messaging**: segmentio/kafka-go for Kafka integration
- **API**: gorilla/mux for routing, api2go/jsonapi for JSON:API implementation
- **Observability**: opentracing/opentracing-go, uber/jaeger-client-go, sirupsen/logrus
- **Utilities**: google/uuid for unique identifiers
- **Internal Libraries**: Custom Atlas libraries (atlas-constants, atlas-kafka, atlas-model, atlas-rest, atlas-tenant)

### Coding Style
- Functional programming style with extensive use of curried functions
- Clear separation of interfaces and implementations
- Descriptive naming conventions for functions and variables
- Comprehensive error handling with custom error types
- Extensive use of Go's type system for domain modeling

## Project Layout and Structure

### Directory Organization
- Domain-driven organization with top-level packages representing business domains
- Infrastructure concerns separated into dedicated packages
- Clear separation between domain logic and technical infrastructure

### Package Structure
- **Domain Packages**: shops, character, inventory, commodities, etc.
- **Infrastructure Packages**: kafka, database, rest, logger, tracing
- **Utility Packages**: retry, test
- **Data Packages**: data/consumable, data/equipable, data/etc, data/setup

### File Organization
- **model.go**: Domain models and builders
- **entity.go**: Database entities
- **processor.go**: Business logic and service implementations
- **rest.go**: JSON:API models
- **producer.go/consumer.go**: Kafka message producers and consumers
- **administrator.go**: Database modification functions
- **provider.go**: Database accessor functions
- **resource.go**: API endpoints

## Domain Modeling

### Model Structure
- Domain models are defined as structs named `Model` with private fields
- Public getter methods provide access to the fields
- Builder pattern for object construction using a struct named `Builder` (created with `NewBuilder()`)
- Models reference other domain models through composition
- Clear separation between domain models and database entities

### Entity Mapping
- Entities map directly to database tables and are named "Entity" (not domain-specific names like "NoteEntity")
- Conversion functions between entities and domain models. The transformation function from an entity to model is called "Make"
- Entities focus on persistence concerns
- Models focus on business logic and behavior

### Value Objects
- Immutable objects representing domain concepts
- Use of Go's type system to enforce constraints
- Validation at construction time

## Design Patterns

### Dependency Injection
- Dependencies passed via constructor parameters
- Clear interfaces for all services
- Testable components with mockable dependencies

### Repository Pattern
- Data access abstracted behind repository interfaces
- CRUD operations encapsulated in repository implementations
- Transaction management handled at the repository level

### Builder Pattern
- Used for constructing complex domain objects
- Fluent interface with method chaining
- Ensures object validity at construction time

### Decorator Pattern
- Used to extend functionality of domain models
- Applied through functional composition
- Allows for separation of cross-cutting concerns

### Provider Pattern
- Lazy evaluation of resources
- Functional approach to resource provisioning
- Error handling integrated into the provider chain

#### Database Entity Provider Pattern
- Uses the `database.EntityProvider[E any]` interface to retrieve domain entities from the database
- Follows a curried function pattern for parameter application
- Returns a `model.Provider[E]` that lazily evaluates to an entity or error
- Example: `func getByIdProvider(tenantId uuid.UUID) func(id uint32) database.EntityProvider[Entity]`
- Database queries are executed only when the provider is invoked
- Entity-to-model transformation is performed using:
  - `model.Map[Entity, Model](Make)` for single entity transformation
  - `model.SliceMap[Entity, Model](Make)` for slice of entities transformation
- Transformation functions (e.g., `Make`) convert database entities to domain models
- Example of entity-to-model mapping:
```
// ByIdProvider retrieves a note by ID
func (p *ProcessorImpl) ByIdProvider(id uint32) model.Provider[Model] {
  return model.Map[Entity, Model](Make)(getByIdProvider(p.t.Id())(id)(p.db))
}

// ByCharacterProvider retrieves all notes for a character
func (p *ProcessorImpl) ByCharacterProvider(characterId uint32) model.Provider[[]Model] {
  return model.SliceMap[Entity, Model](Make)(getByCharacterIdProvider(p.t.Id())(characterId)(p.db))(model.ParallelMap())
}
```
- Promotes clean separation between database access and domain logic
- Enables composition of data access operations with transformation functions
- Supports parallel processing of entity collections with `model.ParallelMap()`

### Factory Pattern
- Creation of complex objects encapsulated in factory functions
- Ensures proper initialization and validation

### Registry Pattern
- Used for tracking runtime state (e.g., shop registry)
- Thread-safe access to shared resources
- Clear ownership of state management

### Processor Pattern
- Core component for implementing domain business logic
- Exposes a well-defined interface with dual method signatures for each operation:
  - Nested functional methods that return curried functions for deferred execution
  - Flat `AndEmit` methods for immediate business logic execution and Kafka event emission
- Interface includes support for create, update, delete, and retrieval operations, all using Go's functional style
- Dependencies (logger, context, database, tenant, producer) are injected via constructor
- Business logic methods:
  - Use a message buffer to accumulate Kafka messages
  - Are curried to support partial application
  - Return domain models and errors
- Emission methods:
  - Compose nested functions using `model.Flip` to produce flat function signatures
  - Call Kafka producers using `Emit` or `EmitWithResult` utilities
- Promotes separation of concerns:
  - Domain logic is independent of message emission
  - Kafka emission can be disabled during testing
- Providers are used for lazy evaluation of database queries
  - Methods such as `ByIdProvider`, `ByCharacterProvider`, and `InTenantProvider` return `model.Provider`
  - `model.Map` and `model.SliceMap` are used to transform database entities into domain models
  - Pure business logic version with nested functional approach (e.g., `Create`)
  - Message-emitting version that integrates with Kafka (e.g., `CreateAndEmit`)
- Pure business logic methods:
  - Accept a message buffer as first parameter to collect messages during processing
  - Use curried functions (nested functions) to allow partial application
  - Return domain models and errors
  - Example: `Create(mb *message.Buffer) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (Model, error)`
- Message-emitting methods:
  - Provide a flattened interface for direct use
  - Internally use the pure business logic methods combined with message emission
  - Use functional composition with `model.Flip` to transform function signatures
  - Example: `CreateAndEmit(characterId uint32, senderId uint32, msg string, flag byte) (Model, error)`
- Provider methods:
  - Return lazy-evaluated functions that retrieve domain models
  - Follow the Provider pattern for resource access
  - Example: `ByIdProvider(id uint32) model.Provider[Model]`
- Promotes separation of concerns:
  - Business logic is isolated from message emission
  - Database operations are separated from domain logic
  - Testability is enhanced by allowing business logic to be tested without message emission

## API Conventions

### JSON:API Specification
- Follows JSON:API specification for REST endpoints
- Consistent resource naming and URL structure
- Proper handling of relationships and included resources

### REST Models
- Dedicated REST models named `RestModel` separate from domain models
- Implements JSON:API interfaces for serialization
- Transform/Extract functions for conversion between domain models and REST models

#### REST Model Structure
- REST models are defined as structs with appropriate data types (not just strings):
  - Numeric IDs use `uint32` for database compatibility
  - Timestamps use `time.Time` for proper date handling
  - Flags use `byte` for compact storage
  - Text fields use `string` for variable-length content
- JSON tags control field visibility and naming in API responses:
  - `json:"-"` hides fields from JSON output (e.g., internal IDs)
  - `json:"fieldName"` specifies the JSON property name
- Example REST model structure:
```
// RestModel is the JSON:API resource for notes
type RestModel struct {
    Id          uint32    `json:"-"`           // Hidden from JSON output
    CharacterId uint32    `json:"characterId"` // Exposed as "characterId" in JSON
    SenderId    uint32    `json:"senderId"`    // Exposed as "senderId" in JSON
    Message     string    `json:"message"`     // Exposed as "message" in JSON
    Flag        byte      `json:"flag"`        // Exposed as "flag" in JSON
    Timestamp   time.Time `json:"timestamp"`   // Exposed as "timestamp" in JSON
}
```

#### JSON:API Interface Implementation
- REST models implement the JSON:API resource interface with three required methods:
  - `GetID() string`: Returns the resource ID as a string
  - `SetID(id string) error`: Sets the resource ID from a string
  - `GetName() string`: Returns the resource type name (collection name)
- Example implementation:
```
// GetID returns the resource ID
func (n RestModel) GetID() string {
    return strconv.Itoa(int(n.Id))  // Convert uint32 to string
}

// SetID sets the resource ID
func (n *RestModel) SetID(strId string) error {
    id, err := strconv.Atoi(strId)  // Convert string to int
    if err != nil {
        return err
    }
    n.Id = uint32(id)  // Store as uint32
    return nil
}

// GetName returns the resource name
func (n RestModel) GetName() string {
    return "notes"  // Collection name in plural form
}
```

#### Transform/Extract Functions
- Transform and Extract functions must satisfy the model.Transformer interface:
  - `func Transform(domainModel) (restModel, error)`: Converts domain model to REST model
  - `func Extract(restModel) (domainModel, error)`: Converts REST model to domain model
- Both functions return an error to handle validation or conversion failures
- These functions enable bidirectional conversion between domain and REST models
- Example implementation:
```
// Transform converts a Model domain model to a RestModel
func Transform(n Model) (RestModel, error) {
    return RestModel{
        Id:          n.Id(),
        CharacterId: n.CharacterId(),
        SenderId:    n.SenderId(),
        Message:     n.Message(),
        Flag:        n.Flag(),
        Timestamp:   n.Timestamp(),
    }, nil
}

// Extract converts a RestModel to parameters for creating or updating a Model
func Extract(r RestModel) (Model, error) {
    return NewBuilder().
        SetId(r.Id).
        SetCharacterId(r.CharacterId).
        SetSenderId(r.SenderId).
        SetMessage(r.Message).
        SetFlag(r.Flag).
        SetTimestamp(r.Timestamp).
        Build(), nil
}
```
- These functions are used with model.Map and model.SliceMap for transforming single models and collections:
```
// Transform a single model
rm, err := model.Map(Transform)(modelProvider)()

// Transform a collection of models
rms, err := model.SliceMap(Transform)(modelCollectionProvider)(model.ParallelMap())()
```

### Error Handling
- Consistent error responses following JSON:API format
- Descriptive error messages and codes
- Proper HTTP status codes for different error conditions

### RESTful Resource Implementation
- Resource endpoints defined in `resource.go` files
- Common helper functions for parameter extraction and response handling
- Consistent pattern for all endpoint handlers

#### Parameter Extraction
- URL path parameters extracted using helper functions:
  - `rest.ParseCharacterId`: Extracts and validates character IDs from URL paths
  - `rest.ParseNoteId`: Extracts and validates note IDs from URL paths
- Query parameters extracted using:
  - `r.URL.Query()`: Gets all query parameters
  - `jsonapi.ParseQueryFields(&query)`: Parses JSON:API specific query parameters (fields, includes, etc.)
- Request body parsing handled by:
  - `rest.ParseInput`: Deserializes JSON:API request bodies into model structs
  - `Extract`: Converts REST models to domain models with proper type conversion

#### Handler Registration
- Handlers registered using helper functions:
  - `rest.RegisterHandler`: For handlers that don't require request body parsing
  - `rest.RegisterInputHandler`: For handlers that require request body parsing
- These functions provide consistent dependency injection and error handling
- Example:
```
registerHandler := rest.RegisterHandler(l)(db)(si)
registerInputHandler := rest.RegisterInputHandler[RestModel](l)(db)(si)

router.HandleFunc("/notes", registerHandler("get_all_notes", GetAllNotesHandler)).Methods(http.MethodGet)
router.HandleFunc("/notes", registerInputHandler("create_note", CreateNoteHandler)).Methods(http.MethodPost)
```

#### Domain Processing
- Processors used to manipulate domain data:
  - Created with `NewProcessor(logger, context, db)`
  - Provider methods (e.g., `ByIdProvider`, `ByCharacterProvider`) retrieve domain models
  - Action methods (e.g., `CreateAndEmit`, `UpdateAndEmit`) modify domain data and emit events
- Example:
```
// Retrieve data
mp := NewProcessor(d.Logger(), d.Context(), d.DB()).ByIdProvider(noteId)

// Modify data
m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).CreateAndEmit(
    im.CharacterId(), im.SenderId(), im.Message(), im.Flag())
```

#### Model Transformation
- Domain models transformed to REST models using:
  - `model.Map(Transform)`: For single model transformation
  - `model.SliceMap(Transform)(model.ParallelMap())`: For transforming collections with parallel processing
- `Transform` function converts domain models to REST models with proper type conversion
- `Extract` function converts REST models back to domain models
- Example:
```
// Transform a single model
rm, err := model.Map(Transform)(mp)()

// Transform a collection of models
rm, err := model.SliceMap(Transform)(mp)(model.ParallelMap())()
```

#### Response Marshaling
- Responses marshaled using common function:
  - `server.MarshalResponse[T](logger)(writer)(serverInfo)(queryParams)(model)`
- Works with both single models and collections
- Handles JSON:API formatting, including sparse fieldsets and includes
- Example:
```
query := r.URL.Query()
queryParams := jsonapi.ParseQueryFields(&query)
server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
```

### Resource Naming
- Resources named using plural nouns (e.g., "shops", "commodities")
- Consistent URL structure for all resources
- Clear relationship naming in JSON:API responses

## Kafka and Messaging

### File Organization
- **Kafka Directory Structure**: Located at `kafka` with three main subdirectories:
  - `consumer`: Contains Kafka consumer implementations
  - `message`: Contains message definitions
  - `producer`: Contains Kafka producer interfaces and common functionality

#### Consumer Files
- Base consumer interface and functionality in `kafka/consumer/consumer.go`
- Domain-specific consumers in subdirectories (e.g., `kafka/consumer/character/consumer.go`)
- Consumers handle incoming Kafka messages and trigger appropriate business logic

#### Message Files
- Common message types and utilities in `kafka/message/message.go`
- Domain-specific message definitions in `kafka/message/{domain}/kafka.go` (e.g., `kafka/message/character/kafka.go`, `kafka/message/note/kafka.go`)
- Message files define command and event structures, constants for topic names, and helper functions for message creation

#### Producer Files
#### Event Producer Provider Pattern
- Kafka event producers must follow the `Provider` pattern and return `model.Provider[[]kafka.Message]`
- Each producer function should end in `Provider` to indicate that message creation is deferred until invoked
- The return value is a provider that yields a slice of Kafka messages, promoting testability and composability
- Event keys are generated using helper functions like `producer.CreateKey`, ensuring partitioning consistency
- Event values are composed using structured domain-specific bodies wrapped in generic message containers (e.g., `StatusEvent[T]`)
- Kafka messages are constructed via the `producer.SingleMessageProvider(key, value)` utility, abstracting serialization and formatting
- This approach ensures clear separation between event construction and emission, supporting both batch and single-message workflows
- Example:
```
// CreateNoteStatusEventProvider creates a provider for note status events
func CreateNoteStatusEventProvider(characterId uint32, noteId uint32, senderId uint32, msg string, flag byte, timestamp time.Time) model.Provider[[]kafka.Message] {
    key := producer.CreateKey(int(characterId))
    body := note.StatusEventCreatedBody{
        NoteId:    noteId,
        SenderId:  senderId,
        Message:   msg,
        Flag:      flag,
        Timestamp: timestamp,
    }
    value := note.StatusEvent[note.StatusEventCreatedBody]{
        CharacterId: characterId,
        Type:        "CREATED",
        Body:        body,
    }
    return producer.SingleMessageProvider(key, value)
}
```

- Common producer interface and functionality in `kafka/producer/producer.go`
- Domain-specific producer implementations in domain packages (e.g., `note/producer.go`)
- Producers handle message serialization and sending to Kafka topics

### Message Structure
- Consistent message format across the application
- Clear separation between message creation and emission
- Topic-based routing for different message types

### Event and Command Naming Conventions
- **Event Naming**:
  - Events are named using past tense verbs to indicate something that has happened (e.g., `Created`, `Updated`, `Deleted`)
  - Event struct names follow the pattern `[Domain]Event` or `[Domain][Action]Event` (e.g., `RouteStateEvent`, `NoteCreatedEvent`)
  - Event types (string identifiers) use uppercase constants (e.g., `CREATED`, `UPDATED`, `ERROR`)
  - Status events use the pattern `StatusEvent[E any]` with a generic type parameter for the body

- **Command Naming**:
  - Commands are named using imperative verbs to indicate an action to be performed (e.g., `Create`, `Update`, `Delete`)
  - Command struct names follow the pattern `Command[E any]` with a generic type parameter for the body
  - Command types (string identifiers) use uppercase constants (e.g., `ENTER`, `EXIT`, `BUY`)
  - Command bodies are named with the pattern `Command[Action]Body` (e.g., `CommandShopEnterBody`, `CommandShopBuyBody`)

### Topic Naming Conventions
- Topic names use lowercase words separated by dots (`.`)
- Topics follow the pattern `[domain].[entity].[action]` or `[domain].[entity].[event-type]`
- Command topics use the suffix `.commands` (e.g., `shop.item.commands`, `character.inventory.commands`)
- Event topics use the suffix `.events` (e.g., `shop.item.events`, `character.inventory.events`)
- Status event topics use the suffix `.status` (e.g., `shop.status`, `note.status`)
- Error topics use the suffix `.errors` (e.g., `shop.errors`, `note.errors`)
- Topics should be domain-specific and clearly indicate the type of messages they contain
- Examples:
  - `route.state.transitions` - For route state transition events
  - `character.shop.commands` - For shop-related commands for characters
  - `inventory.item.events` - For inventory item-related events
  - `note.status` - For note status events

#### Command Messages
- Generic structure with type parameter for the body: `Command[E any]`
- Common members across all commands:
  - `CharacterId`: Identifies the character the command applies to
  - `Type`: String identifier for the command type (e.g., "ENTER", "EXIT", "BUY")
  - `Body`: Generic field containing command-specific data
- Command bodies are strongly typed structs specific to each command type
- Examples:
  - `CommandShopEnterBody`: Contains `NpcTemplateId`
  - `CommandShopBuyBody`: Contains `Slot`, `ItemTemplateId`, `Quantity`, `DiscountPrice`
  - `RequestChangeMesoBody`: Contains `ActorId`, `ActorType`, `Amount`

#### StatusEvent Messages
- Generic structure with type parameter for the body: `StatusEvent[E any]`
- Common members across all status events:
  - `CharacterId`: Identifies the character the event applies to
  - `Type`: String identifier for the event type (e.g., "ENTERED", "EXITED", "ERROR")
  - `Body`: Generic field containing event-specific data
- Status event bodies are strongly typed structs specific to each event type
- Examples:
  - `StatusEventEnteredBody`: Contains `NpcTemplateId`
  - `StatusEventErrorBody`: Contains `Error`, `LevelLimit`, `Reason`
  - `StatusEventMapChangedBody`: Contains `ChannelId`, `OldMapId`, `TargetMapId`, `TargetPortalId`

### Producer/Consumer Pattern
- Dedicated producers and consumers for each domain
- Clear separation of concerns between message production and consumption
- Error handling integrated into the messaging flow

### Buffer Pattern
- Messages collected in a buffer before sending
- Allows for atomic message operations
- Ensures consistency in message emission

### Functional Approach
- Higher-order functions for message handling
- Composition of message handlers
- Generic programming for flexible message types

### xxxAndEmit Pattern
- Separation of business logic from message emission
- Each operation has two versions: one with direct business logic (xxx) and one that emits messages (xxxAndEmit)
- The xxxAndEmit methods use the message.Emit function to wrap the business logic and emit Kafka messages
- Business logic methods accept a message.Buffer parameter to collect messages during processing
- Functional composition using model.Flip to transform function signatures
- Ensures consistent message handling across the application
- Examples include EnterAndEmit/Enter, ExitAndEmit/Exit, BuyAndEmit/Buy, etc.
- Promotes testability by allowing business logic to be tested without message emission

## Logging and Observability

### Structured Logging
- Uses logrus for structured logging
- ECS (Elastic Common Schema) formatting for log entries
- Consistent field naming across log entries
- Service name included in all log entries

### Tracing
- OpenTracing API with Jaeger implementation
- Span creation and management
- Correlation between logs and traces
- Proper context propagation

### Configuration
- Environment variable-based configuration
- Sensible defaults with override capability
- Clear separation of configuration concerns

### Error Reporting
- Comprehensive error logging
- Context-rich error messages
- Proper error propagation through the call stack

## Documentation Practices

- All services must maintain a clearly written and up-to-date `README.md` file at the root of the service directory.
- The `README.md` should include:
  - A brief description of the service and its purpose
  - Setup instructions including environment variables and dependencies
  - Usage examples for key operations or endpoints
  - Any configuration flags or feature toggles relevant to deployment or development
- Any time a significant change is made to API contracts, behavior, environment configuration, or service outputs, the `README.md` must be updated accordingly.
- This ensures that both human developers and automation agents referencing the service have reliable and actionable documentation.

## Testing Conventions

### Test Organization
- Table-driven tests for comprehensive coverage
- Subtests for better organization and reporting
- Clear test naming conventions

### Test Helpers
- Dedicated test package with helper functions
- Mock implementations for external dependencies
- Database setup and teardown utilities

### Test Patterns
- Follows AAA (Arrange-Act-Assert) pattern
- Tests both happy paths and edge cases
- Verifies database state after operations
- Comprehensive CRUD operation testing

### Mocking
### Mock Processor Requirements
- Every concrete processor must have a corresponding `ProcessorMock` implementation to support automated testing.
- The mock should reside in a `mock` subpackage within the same domain package as the real processor.
- The mock struct must implement the `Processor` interface and expose overrideable function fields for each method:
  - e.g., `CreateFunc`, `UpdateAndEmitFunc`, `ByIdProviderFunc`, etc.
- Each method should check whether its corresponding function field is nil, and fallback to a default no-op or zero-value response when unset.
- This structure allows for precise and minimal overrides during testing without affecting unrelated behavior.
- Mocks should be used to simulate business logic behavior, Kafka emissions, and provider returns without invoking real infrastructure or state.


- Interface-based design enables easy mocking
- Mock implementations for external services
- Dependency injection facilitates testing with mocks

### Resource Cleanup
- Proper cleanup of test resources
- Use of defer for guaranteed cleanup
- Isolation between test cases
