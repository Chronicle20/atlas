---
name: golang-microservice
description: Skill for creating and modifying Golang microservices using DDD, immutable models, functional composition, GORM entities, JSON:API transport, and Kafka messaging with context based multi tenancy.
---


# Golang Microservice Skill

## Purpose
Provide a composable entry point that activates when working on any Golang service. This skill aligns development and AI generation with architecture patterns and conventions.

## When to Use
Activate when working on:
- Any Go microservice
- Files: `model.go`, `entity.go`, `builder.go`, `processor.go`, `provider.go`, `producer.go`, `resource.go`, `rest.go`, `requests.go`, `state.go`, or `cache.go`
- Kafka producers/consumers
- Caching layers and singleton patterns
- REST JSON:API endpoints
- Cross-service REST client calls
- Multi-tenancy context logic
- Testing domain logic, providers, or emission paths

---

## Quick Start Checklist
- [ ] Immutable **Model** with accessors
- [ ] **Entity** with GORM tags and migrations
- [ ] Fluent **Builder** enforcing invariants
- [ ] Pure **Processor** functions and `AndEmit` variants
- [ ] Lazy **Provider** for data access
- [ ] Kafka **Producer** initialized with context decorators
- [ ] **Resource** file for route registration and handlers
- [ ] **Requests** + **rest.go** for cross-service REST calls (if needed)
- [ ] Context-based multi-tenancy (`tenant.MustFromContext`)
- [ ] Table-driven **tests** for all logic layers


---

## Key Principles
1. **Immutability** — Models never mutate; all state changes yield new instances.
2. **Functional Composition** — Use curried functions and providers for composition.
3. **Event-Driven Design** — Kafka coordinates inter-service communication.
4. **Context Isolation** — Tenant and trace always derived from context.
5. **Layer Separation** — Each file type has a clear single responsibility.
6. **Pure Logic First** — Business logic runs without side effects unless explicitly wrapped in `AndEmit`.


---

## File Responsibilities

| File | Primary Responsibility | Key Dependencies |
|------|-------------------------|------------------|
| `model.go` | Domain model definition | None |
| `entity.go` | Database schema and migrations | GORM |
| `builder.go` | Fluent construction of valid models | Model |
| `processor.go` | Core business logic | Model, Provider |
| `provider.go` | Lazy database access | GORM, Entity |
| `producer.go` | Kafka event creation | Kafka, Provider |
| `cache.go` | Singleton cache implementation | sync.Once, sync.RWMutex |
| `resource.go` | Route registration and handlers | REST, Processor |
| `rest.go` | JSON:API resource mappings | Model |
| `requests.go` | Cross-service REST client functions | atlas-rest, rest.go |
| `state.go` | Domain states or enums | Model |

---


## Navigation Guide

| Topic | Reference |
|-------|------------|
| Architecture Overview | [resources/architecture-overview.md](resources/architecture-overview.md) |
| File Responsibilities | [resources/file-responsibilities.md](resources/file-responsibilities.md) |
| Functional & Builder Patterns | [resources/patterns-functional.md](resources/patterns-functional.md) |
| Provider Pattern | [resources/patterns-provider.md](resources/patterns-provider.md) |
| **Cache Patterns** | **[resources/patterns-cache.md](resources/patterns-cache.md)** |
| Kafka Integration | [resources/patterns-kafka.md](resources/patterns-kafka.md) |
| REST JSON:API | [resources/patterns-rest-jsonapi.md](resources/patterns-rest-jsonapi.md) |
| Multi-Tenancy Context | [resources/patterns-multitenancy-context.md](resources/patterns-multitenancy-context.md) |
| Testing Conventions | [resources/testing-guide.md](resources/testing-guide.md) |
| **Cross-Service Implementation** | **[resources/cross-service-implementation.md](resources/cross-service-implementation.md)** |
| AI Code Guidance | [resources/ai-guidance.md](resources/ai-guidance.md) |
| Anti-Patterns | [resources/anti-patterns.md](resources/anti-patterns.md) |

---
