# atlas-storage

Cross-character, account-level storage microservice for the Atlas platform. Manages shared storage that characters on the same account (within a world) can deposit and withdraw items from.

## Overview

The storage service provides:

- **Account-scoped storage**: Each storage is uniquely identified by `(tenantId, worldId, accountId)` with a default capacity of 4 slots
- **Lazy initialization**: Storage is automatically created on first access (GET request or deposit operation)
- **Unified asset model**: All item types (equipment, consumables, setup, etc, cash, pets) are stored as a single flat asset entity with all fields inline -- no separate tables for different item types
- **Mesos management**: Store and manage mesos in storage with SET, ADD, and SUBTRACT operations
- **Merge and sort**: Intelligent merging of stackable items respecting slotMax limits
- **Projections**: In-memory projections of storage state for active character sessions, organized by inventory type compartments
- **Saga participation**: Integrated with atlas-saga-orchestrator for transactional deposit/withdraw operations via compartment commands (ACCEPT/RELEASE)
- **Expiration**: Handles asset expiration with optional replacement item creation

## Dependencies

- **atlas-data**: Item data lookups (slotMax, rechargeable flag)
- **atlas-saga-orchestrator**: Transaction coordination
