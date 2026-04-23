---
description: Generate or update documentation for one Atlas service — dispatches the service-documentation agent
argument-hint: Service name or path (e.g., "atlas-account" or "services/atlas-account")
---

Dispatch the `service-documentation` agent against: **$ARGUMENTS**.

The agent treats code as the single source of truth, follows `DOCS.md`, and operates only within the target service directory. It outputs only updated doc files — no commentary, no analysis.
