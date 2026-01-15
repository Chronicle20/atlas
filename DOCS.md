# Atlas Microservice Documentation Contract (DOCS.md)

## 1. Purpose

This document defines the mandatory documentation structure, scope, and constraints
for all Atlas Go microservices.

Documentation is a first-class artifact and must:
- Reflect code as implemented
- Follow strict file responsibilities
- Avoid inference, improvement, or speculation
- Remain consistent across services

This document is authoritative for what may and may not appear in documentation.

---

## 2. Core Principles

### 2.1 Documentation mirrors architecture
Documentation structure must directly reflect:
- Domain boundaries
- File semantics
- Transport mechanisms
- Storage models

### 2.2 Documentation is descriptive, not prescriptive
Documentation:
- Describes what exists
- Never describes what should exist
- Never proposes alternatives
- Never improves or rationalizes design choices

### 2.3 No cross-layer leakage
Each documentation artifact has a single concern.
Cross-references are allowed. Explanations are not.

---

## 3. Required Documentation Artifacts

Each service MUST contain the following files:

/README.md
/docs/domain.md
/docs/kafka.md
/docs/rest.md
/docs/storage.md

Optional artifacts MAY exist only if explicitly justified:

/docs/saga.md
/docs/state.md
/docs/migrations.md

---

## 4. README.md

### Purpose
High-level orientation for humans.

### Allowed Content
- Service responsibility (1–2 paragraphs)
- External dependencies (databases, Kafka, Redis, etc.)
- Runtime configuration overview
- Links to deeper documentation

### Forbidden Content
- Business rules
- Domain invariants
- Kafka message schemas
- REST request or response details
- Database schema definitions

---

## 5. docs/domain.md

### Purpose
Describe domain logic and invariants, independent of transport or storage.

### Required Structure

## <domain-name>

### Responsibility
### Core Models
### Invariants
### State Transitions (if applicable)
### Processors

### Allowed Content
- Domain model responsibilities
- Immutable model invariants
- Processor responsibilities
- High-level state transitions

### Forbidden Content
- REST endpoints
- Kafka topics or payloads
- Database tables or queries
- Infrastructure concerns

---

## 6. docs/kafka.md

### Purpose
Document Kafka integration surface only.

### Required Structure

## Topics Consumed
## Topics Produced
## Message Types
## Transaction Semantics

### Allowed Content
- Topic names
- Direction (command or event)
- Message struct names
- Required headers (for example, transactionId)
- Ordering or partitioning notes

### Forbidden Content
- Business logic explanations
- Processor behavior
- State transitions
- Retry or compensation logic

---

## 7. docs/rest.md

### Purpose
Document public HTTP interface only.

### Required Structure

## Endpoints

### <METHOD> <PATH>

Each endpoint MUST include:
- Parameters
- Request model
- Response model
- Error conditions

### Allowed Content
- HTTP methods and paths
- JSON:API resource types
- Validation rules
- Error codes and meanings

### Forbidden Content
- Processor logic
- Database queries
- Kafka emission details
- Domain invariants

---

## 8. docs/storage.md

### Purpose
Describe persistent storage representation, not access logic.

### Required Structure

## Tables
## Relationships
## Indexes
## Migration Rules

### Allowed Content
- Table names
- Columns and types
- Relationships
- Indexing strategy
- Migration guarantees

### Forbidden Content
- Query logic
- Caching strategies
- Business rules
- REST or Kafka references

---

## 9. File-to-Documentation Mapping Rules

| Code Artifact | Documentation |
|--------------|---------------|
| model.go | docs/domain.md |
| builder.go | docs/domain.md |
| processor.go | docs/domain.md |
| producer.go | docs/kafka.md |
| kafka/consumer/* | docs/kafka.md |
| resource.go | docs/rest.md |
| rest.go | docs/rest.md |
| entity.go | docs/storage.md |

If a file exists without a corresponding documentation entry,
documentation is incomplete.

---

## 10. Documentation Update Rules

### Required Updates
Documentation MUST be updated when:
- A new domain is added
- A new Kafka topic is produced or consumed
- A REST endpoint is added or modified
- A database schema changes

### Forbidden Updates
Documentation MUST NOT be updated:
- During design-only planning
- Based on intended or future behavior
- To explain implementation rationale

---

## 11. AI Usage Rules

When documentation is generated or updated by an AI agent:

### The agent MUST:
- Follow this document exactly
- Use code as the source of truth
- Ask before adding new sections
- Preserve existing structure and tone

### The agent MUST NOT:
- Infer missing behavior
- Improve clarity beyond restating facts
- Merge sections across concerns
- Reorganize files without instruction

---

## 12. Validation Criteria

Documentation is considered complete and valid if:
- All required files exist
- All required sections are present
- No forbidden content is included
- All code-to-documentation mappings are satisfied

---

## 13. Non-Goals

This documentation contract does not:
- Teach Go
- Explain Kafka or REST fundamentals
- Justify architectural decisions
- Serve as onboarding training material
