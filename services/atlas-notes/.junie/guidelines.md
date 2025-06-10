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
- **rest.go**: API endpoints and JSON:API models
- **producer.go/consumer.go**: Kafka message producers and consumers
- **administrator.go**: Database modification functions
- **provider.go**: Database accessor functions

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

### Factory Pattern
- Creation of complex objects encapsulated in factory functions
- Ensures proper initialization and validation

### Registry Pattern
- Used for tracking runtime state (e.g., shop registry)
- Thread-safe access to shared resources
- Clear ownership of state management

## API Conventions

### JSON:API Specification
- Follows JSON:API specification for REST endpoints
- Consistent resource naming and URL structure
- Proper handling of relationships and included resources

### REST Models
- Dedicated REST models named `RestModel` separate from domain models
- Implements JSON:API interfaces for serialization
- Transform/Extract functions for conversion between domain models and REST models

### Error Handling
- Consistent error responses following JSON:API format
- Descriptive error messages and codes
- Proper HTTP status codes for different error conditions

### Resource Naming
- Resources named using plural nouns (e.g., "shops", "commodities")
- Consistent URL structure for all resources
- Clear relationship naming in JSON:API responses

## Kafka and Messaging

### Message Structure
- Consistent message format across the application
- Clear separation between message creation and emission
- Topic-based routing for different message types

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
- Interface-based design enables easy mocking
- Mock implementations for external services
- Dependency injection facilitates testing with mocks

### Resource Cleanup
- Proper cleanup of test resources
- Use of defer for guaranteed cleanup
- Isolation between test cases
