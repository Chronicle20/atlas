
---
title: Anti-Patterns
description: Common pitfalls to avoid when implementing Golang microservices.
---

# Anti-Patterns


| Anti-Pattern | Why It's Wrong |
|---------------|----------------|
| Business logic in handlers | Breaks separation of concerns |
| Mutable public fields | Violates immutability |
| Database logic in processors | Violates functional purity |

| Hardcoded topics | Breaks environment portability |
| Missing validation | Allows invalid domain states |
| Passing TenantId as param | Must use context extraction |
| Skipping header decorators | Breaks tracing and tenancy propagation |
| Global context usage | Breaks request isolation |
| Manual JSON:API envelope handling | Breaks JSON:API integration, adds boilerplate |
| Nested Data/Type/Attributes in requests | Use flat structures, let api2go handle envelope |
| Manual tenant parsing in handlers | Use `server.RegisterHandler` for automatic parsing |
| Custom error response helpers | Just write status codes directly |
| jsonapi struct tags on REST models | Use interface methods (`GetName`, `GetID`, `SetID`) |
| Plain http.HandlerFunc for routes | Use `server.RegisterHandler` for automatic tenant/tracing |

**Always** prefer pure, context-aware, curried, and testable functions.

**For REST:** Use `server.RegisterHandler` and `server.RegisterInputHandler` with flat JSON:API-compliant models.
