---
title: Cross-Service Implementation Guide
description: Guidelines for implementing features that span multiple microservices
---

# Cross-Service Implementation Guide

## Overview
When implementing features that span multiple services (e.g., cosmetic changes, pet operations, inventory transactions), follow this systematic approach to ensure all dependencies are properly updated and tested.

---

## Critical Rules

### 1. Implementation Order Matters
Always implement changes in dependency order:
1. **Shared types/constants** (e.g., saga payload types, action constants)
2. **Core service implementations** (e.g., character cosmetic changes)
3. **Orchestrator/handler updates** (e.g., saga handler methods)
4. **Consumer service operations** (e.g., NPC conversation operations)
5. **Mock updates** (for all modified interfaces)
6. **Build verification** (all affected services)
7. **Test execution** (all affected test suites)

### 2. Never Leave Partial Implementations
A feature is NOT complete until:
- ✅ All services compile successfully
- ✅ All tests pass
- ✅ All mocks are updated
- ✅ No orphaned/duplicate code remains

---

## Pre-Implementation Checklist

Before starting a cross-service feature:

```bash
# 1. Identify all affected services
# Check go.work to understand dependencies
cat go.work

# 2. Map the data flow
# Example: NPC → Saga Orchestrator → Character Service
# Identify: What types? What messages? What handlers?

# 3. Verify existing implementations
# Check what's already implemented vs. what needs to be added
```

---

## Implementation Checklist

### Phase 1: Type Definitions & Constants

**Before implementing any logic**, add all required types:

- [ ] Add saga action constants (e.g., `ChangeHair`, `ChangeFace`)
- [ ] Add saga payload types (e.g., `ChangeHairPayload`)
- [ ] Add Kafka message types if needed
- [ ] Add validation condition types if needed

**Example:**
```go
// services/atlas-npc-conversations/atlas.com/npc/saga/model.go
const (
    ChangeHair Action = "change_hair"
    ChangeFace Action = "change_face"
    ChangeSkin Action = "change_skin"
)

type ChangeHairPayload struct {
    CharacterId uint32     `json:"characterId"`
    WorldId     world.Id   `json:"worldId"`
    ChannelId   channel.Id `json:"channelId"`
    StyleId     uint32     `json:"styleId"`
}
```

**Verification:**
```bash
go build ./services/atlas-npc-conversations/atlas.com/npc/saga/...
```

---

### Phase 2: Interface Changes

When adding methods to interfaces:

- [ ] Update the primary interface definition
- [ ] Find ALL implementations using workspace search
- [ ] Update ALL implementations
- [ ] Update ALL mocks in test packages

**Finding All Implementations:**
```bash
# Find all files that might implement the interface
grep -r "type.*Processor.*struct" services/

# Find all mocks
find . -path "*/mock/*.go" -name "*processor*.go"
```

**Common Interfaces to Check:**
- `character.Processor` → `character/mock/processor.go`
- `saga.Handler` → `saga/mock/handler.go`
- Any interface in a `/mock/` directory

---

### Phase 3: Event Producers/Handlers

When modifying event producers:

- [ ] Remove old/duplicate function declarations
- [ ] Update function signatures consistently
- [ ] Find ALL call sites using workspace search
- [ ] Update ALL callers to use new signatures
- [ ] Delete unused/orphaned functions

**Finding All Call Sites:**
```bash
# Search for function calls
grep -r "hairChangedEventProvider" services/

# Check both AndEmit and Buffer variants
grep -r "ChangeHairAndEmit\|ChangeHair(mb" services/
```

**Anti-Pattern - Duplicate Functions:**
```go
❌ // OLD - takes channel.Model
func hairChangedEventProvider(..., channel channel.Model, ...) {
    WorldId: channel.WorldId()
}

❌ // NEW - takes world.Id (but old one still exists!)
func hairChangedEventProvider(..., worldId world.Id, ...) {
    WorldId: worldId
}
```

**Correct Pattern - Single Function:**
```go
✅ // Only ONE version exists
func hairChangedEventProvider(..., worldId world.Id, ...) {
    WorldId: worldId
}
```

---

### Phase 4: Helper Methods & Dependencies

When operations reference helper methods:

- [ ] Verify helper methods exist BEFORE using them
- [ ] Implement missing methods if needed
- [ ] Check method signatures match usage

**Example - Context Methods:**
```go
// ❌ BAD - Assuming methods exist
err = e.setContextValue(characterId, key, value)  // Method doesn't exist!

// ✅ GOOD - Verify first, implement if missing
// 1. Check if OperationExecutorImpl has getContextValue/setContextValue
// 2. If not, implement them
// 3. Then use them
```

---

### Phase 5: Build Verification

**CRITICAL:** Build ALL affected services before proceeding:

```bash
# From workspace root (/Users/tumidanski/source/atlas)

# Build each affected service
go build ./services/atlas-character/atlas.com/character/...
go build ./services/atlas-npc-conversations/atlas.com/npc/...
go build ./services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/...

# Fix ALL compilation errors before continuing
# Common errors:
# - Missing methods on interfaces
# - Duplicate function declarations
# - Type mismatches in function calls
# - Undefined types or constants
```

---

### Phase 6: Test Execution

Run tests for ALL affected services:

```bash
# Test each service individually
cd services/atlas-character/atlas.com/character
go test ./...

cd ../../../atlas-npc-conversations/atlas.com/npc
go test ./...

cd ../../../atlas-saga-orchestrator/atlas.com/saga-orchestrator
go test ./...
```

**If Tests Fail:**
1. ❌ DO NOT ignore or skip
2. ✅ Report failures immediately
3. ✅ Fix the issue (usually missing mock methods)
4. ✅ Re-run tests until they pass

---

## Common Failure Patterns & Solutions

### Pattern 1: Missing Mock Methods

**Error:**
```
cannot use *ProcessorMock as Processor value:
  *ProcessorMock does not implement Processor (missing method ChangeFace)
```

**Solution:**
1. Open the mock file (e.g., `character/mock/processor.go`)
2. Add the missing method to the struct:
   ```go
   ChangeFaceFunc func(...)
   ```
3. Implement the method:
   ```go
   func (m *ProcessorMock) ChangeFace(...) error {
       if m.ChangeFaceFunc != nil {
           return m.ChangeFaceFunc(...)
       }
       return nil
   }
   ```

### Pattern 2: Duplicate Function Declarations

**Error:**
```
hairChangedEventProvider redeclared in this block
  other declaration of hairChangedEventProvider
```

**Solution:**
1. Find both declarations (check line numbers in error)
2. Determine which signature is correct (check caller usage)
3. Delete the incorrect/old version
4. Update all callers to use consistent signature

### Pattern 3: Incomplete Call Site Updates

**Error:**
```
cannot use channel (type channel.Model) as world.Id value in argument
```

**Solution:**
1. Function signature was changed but not all callers updated
2. Search for ALL calls: `grep -r "functionName" services/`
3. Update ALL call sites to match new signature
4. Example: Change `channel` → `channel.WorldId()`

### Pattern 4: Missing Type Definitions

**Error:**
```
undefined: saga.ChangeHairPayload
undefined: saga.ChangeHair
```

**Solution:**
1. Add the type definition FIRST in the saga model
2. Then implement operations that use it
3. Never reference types that don't exist

---

## Type-Safe Refactoring Pattern

When changing function signatures across services:

1. **Add** new function with new signature (don't modify existing yet)
2. **Find** all callers of old function: `grep -r "oldFunction" services/`
3. **Update** all callers to use new function
4. **Verify** builds: `go build ./...`
5. **Delete** old function
6. **Re-verify** builds to catch any missed callers

This prevents compilation errors during the transition.

---

## Red Flags That Require Extra Verification

| Red Flag | Action Required |
|----------|----------------|
| Adding methods to interfaces | ✅ Check ALL mocks |
| Changing function signatures | ✅ Search ALL call sites |
| Adding new saga actions | ✅ Add types to saga model FIRST |
| Referencing context methods | ✅ Implement them BEFORE using |
| Multiple services in git diff | ✅ Build ALL of them |
| Operations using new types | ✅ Verify types exist first |

---

## Pre-Commit Verification Script

Before committing cross-service changes:

```bash
#!/bin/bash
# Save as: scripts/verify-build.sh

set -e  # Exit on first error

echo "Building all services..."

services=(
    "atlas-character"
    "atlas-npc-conversations"
    "atlas-saga-orchestrator"
)

for service in "${services[@]}"; do
    echo "Building $service..."
    go build ./services/$service/atlas.com/*/...
done

echo "Running tests..."
for service in "${services[@]}"; do
    echo "Testing $service..."
    (cd services/$service/atlas.com/* && go test ./...)
done

echo "✅ All builds and tests passed!"
```

---

## Example: Adding Cosmetic Change Feature

### Step 1: Add Types (saga model)
```go
// services/atlas-npc-conversations/atlas.com/npc/saga/model.go
const (
    ChangeHair Action = "change_hair"
)

type ChangeHairPayload struct {
    CharacterId uint32
    StyleId     uint32
}
```

### Step 2: Add Interface Methods (character processor)
```go
// services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/processor.go
type Processor interface {
    ChangeHairAndEmit(...) error
    ChangeHair(mb *message.Buffer) func(...) error
}
```

### Step 3: Update Mocks
```go
// services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/mock/processor.go
type ProcessorMock struct {
    ChangeHairFunc func(...)
}

func (m *ProcessorMock) ChangeHair(...) error {
    if m.ChangeHairFunc != nil {
        return m.ChangeHairFunc(...)
    }
    return nil
}
```

### Step 4: Implement Logic (character service)
```go
// services/atlas-character/atlas.com/character/character/processor.go
func (p *ProcessorImpl) ChangeHair(...) error {
    // Implementation
}
```

### Step 5: Add Operations (NPC conversations)
```go
// services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go
case "change_hair":
    payload := saga.ChangeHairPayload{...}
    return stepId, saga.Pending, saga.ChangeHair, payload, nil
```

### Step 6: Verify Everything
```bash
go build ./services/atlas-character/atlas.com/character/...
go build ./services/atlas-npc-conversations/atlas.com/npc/...
go build ./services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/...

cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./...
```

---

## Summary

**Before considering cross-service work complete:**

1. ✅ All types/constants defined
2. ✅ All interfaces updated
3. ✅ All implementations updated
4. ✅ All mocks updated
5. ✅ No duplicate code
6. ✅ All services build
7. ✅ All tests pass
8. ✅ No orphaned call sites

**Remember:** A feature that doesn't build or test is not complete!

---

## REST Client Pattern for Cross-Service Calls

When one service needs to call another service's REST API (not via Kafka), use the `requests.go` / `rest.go` pattern.

### File Structure

Create a sub-package for the external service call:

```
myservice/
└── domain/
    └── external_service/      # Sub-package for REST client
        ├── rest.go            # RestModel for response deserialization
        └── requests.go        # Request functions
```

### Step 1: Create `rest.go`

Define a `RestModel` struct implementing JSON:API interface methods:

```go
package status

const Resource = "quests"

// RestModel represents the response from the external service
type RestModel struct {
    Id     string `json:"-"`          // ID set via SetID, not from JSON body
    Status uint8  `json:"status"`     // Actual response field
}

// JSON:API interface methods
func (r RestModel) GetName() string { return Resource }
func (r RestModel) GetID() string   { return r.Id }
```

**Key Points:**
- `Id` field uses `json:"-"` tag (populated via SetID from JSON:API wrapper)
- Other fields match the JSON response structure
- Implement `GetName()` and `GetID()` for JSON:API compliance

### Step 2: Create `requests.go`

Define request functions using `atlas-rest` library:

```go
package status

import (
    "myservice/rest"
    "fmt"
    "github.com/Chronicle20/atlas-rest/requests"
)

// getBaseRequest returns the base URL from environment variable
// SERVICE_NAME maps to SERVICE_NAME_BASE_URL environment variable
func getBaseRequest() string {
    return requests.RootUrl("QUEST")  // Uses QUEST_BASE_URL env var
}

// GET request example
func requestByCharacterAndQuest(characterId, questId uint32) requests.Request[RestModel] {
    return rest.MakeGetRequest[RestModel](
        fmt.Sprintf(getBaseRequest()+"/characters/%d/quests/%d", characterId, questId),
    )
}

// POST request example
func requestValidation(body RestModel) requests.Request[RestModel] {
    return rest.MakePostRequest[RestModel](
        getBaseRequest()+"/validations",
        body,
    )
}
```

**Key Points:**
- `getBaseRequest()` uses `requests.RootUrl("SERVICE")` which reads `SERVICE_BASE_URL` env var
- Request functions return `requests.Request[T]` (a curried function)
- Use `rest.MakeGetRequest[T]` for GET, `rest.MakePostRequest[T]` for POST
- Build URLs with `fmt.Sprintf` for path parameters

### Step 3: Call from Processor

Execute the request by calling with `(logger, context)`:

```go
func (p *Processor) doSomething(ctx context.Context, characterId, questId uint32) error {
    // Execute the request - note the (l, ctx) call pattern
    resp, err := status.requestByCharacterAndQuest(characterId, questId)(p.l, ctx)
    if err != nil {
        p.l.WithError(err).Error("Failed to get quest status")
        return err
    }

    // Use the response
    questStatus := resp.Status
    // ...
}
```

**Key Points:**
- Request functions are curried: `requestFn(params)(logger, context)`
- Context propagates tenant headers automatically
- Logger is used for request/error logging

### Complete Example: Quest Status Client

**Package structure:**
```
conversation/quest/status/
├── rest.go
└── requests.go
```

**rest.go:**
```go
package status

const Resource = "quests"

type QuestStatus int
const (
    QuestNotStarted QuestStatus = 1
    QuestStarted    QuestStatus = 2
    QuestCompleted  QuestStatus = 3
)

type RestModel struct {
    Id     string `json:"-"`
    Status uint8  `json:"status"`
}

func (r RestModel) GetName() string { return Resource }
func (r RestModel) GetID() string   { return r.Id }
```

**requests.go:**
```go
package status

import (
    "atlas-npc-conversations/rest"
    "fmt"
    "github.com/Chronicle20/atlas-rest/requests"
)

func getBaseRequest() string {
    return requests.RootUrl("QUEST")
}

func requestByCharacterAndQuest(characterId, questId uint32) requests.Request[RestModel] {
    return rest.MakeGetRequest[RestModel](
        fmt.Sprintf(getBaseRequest()+"/characters/%d/quests/%d", characterId, questId),
    )
}
```

**Usage in processor:**
```go
import "atlas-npc-conversations/atlas.com/npc/conversation/quest/status"

func (p *Processor) startQuestConversation(ctx context.Context, req StartRequest) {
    resp, err := status.requestByCharacterAndQuest(req.CharacterId, req.QuestId)(p.l, ctx)
    if err != nil {
        // Handle error
        return
    }
    questStatus := status.QuestStatus(resp.Status)
    // Route based on status...
}
```

### Anti-Patterns

❌ **Don't use provider pattern for REST calls:**
```go
// WRONG - this is for database access, not REST calls
func QuestStatusProvider(l logrus.FieldLogger) func(ctx context.Context) func(id uint32) model.Provider[Status] {
    // ...
}
```

❌ **Don't define client interfaces:**
```go
// WRONG - unnecessary abstraction
type QuestClient interface {
    GetStatus(characterId, questId uint32) (Status, error)
}
```

✅ **Do use requests.go pattern:**
```go
// CORRECT - simple request function
func requestByCharacterAndQuest(characterId, questId uint32) requests.Request[RestModel] {
    return rest.MakeGetRequest[RestModel](url)
}
```

### Environment Variables

The `requests.RootUrl("SERVICE")` function reads from environment:

| RootUrl Parameter | Environment Variable |
|-------------------|---------------------|
| `"QUEST"` | `QUEST_BASE_URL` |
| `"QUERY_AGGREGATOR"` | `QUERY_AGGREGATOR_BASE_URL` |
| `"MAPS"` | `MAPS_BASE_URL` |

### Existing Examples in Codebase

Reference these for additional patterns:

| Service | Package | Purpose |
|---------|---------|---------|
| atlas-npc-conversations | `validation/` | POST request to query-aggregator |
| atlas-npc-conversations | `map/` | GET characters in map |
| atlas-npc-conversations | `cosmetic/` | GET appearance data |
| atlas-channel | `character/` | GET/POST character data |
