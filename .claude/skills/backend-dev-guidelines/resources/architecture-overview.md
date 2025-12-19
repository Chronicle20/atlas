
---
title: Architecture Overview
description: Core architectural principles of Golang microservices.
---

# Architecture Overview

Services follow a strict layered design with functional composition and immutability.

## Layers
1. **Domain Layer** — Core logic, models, and validation.
2. **Infrastructure Layer** — Database, Kafka, and external systems.
3. **Transport Layer** — REST and Kafka message endpoints.
4. **Application Layer** — Orchestration and configuration.

## Core Technologies
- Go 1.24+
- GORM ORM (PostgreSQL / SQLite)

- JSON:API standard
- Kafka for messaging
- Functional programming with curried functions

## Principles

- **Immutability:** Domain models never mutate.
- **Separation of Concerns:** Domain logic isolated from persistence and transport.
- **Functional Composition:** Use `Provider`, `Map`, and `ParallelMap` for chaining.
- **Event-Driven:** All inter-service communication via Kafka.

## Startup Example

```go

db := database.Connect(logger, database.SetMigrations(domain.Migration))
server.New(logger).
  AddRouteInitializer(domain.InitializeRoutes(db)(GetServer())).
  Run()
```
