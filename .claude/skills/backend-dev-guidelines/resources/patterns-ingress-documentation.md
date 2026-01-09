---
title: Ingress and Documentation Pattern
description: Guidelines for maintaining ingress configuration and service documentation when working with REST APIs
---

# Ingress and Documentation Pattern

## Overview

When adding or modifying REST endpoints in a microservice, two critical infrastructure pieces must be updated:
1. **Ingress Configuration** - Routes external traffic to your service
2. **Service README** - Documents the API contract for other developers

Failing to update these causes:
- Endpoints that can't be reached externally (missing ingress)
- Misleading documentation that doesn't match implementation
- Wasted time for developers trying to use your service

---

## Ingress Configuration

### When to Update Ingress

Update `atlas-ingress.yml` when:
- ✅ Adding new REST endpoints to a service
- ✅ Adding a new service with REST endpoints
- ✅ Changing the base path of existing endpoints

Do NOT update ingress for:
- ❌ Internal-only services accessed via Kafka only
- ❌ Changes to Kafka producers/consumers
- ❌ Backend logic changes without endpoint changes

---

### Ingress File Location

**File:** `atlas-ingress.yml` (at the root of the atlas directory)

This file contains:
- Nginx configuration (lines 8-240)
- Multiple `location` blocks routing traffic to different services
- Each service gets a regex pattern matching its API path

---

### Adding a New Service Route

#### Pattern to Follow

```nginx
location ~ ^/api/<service-name>(/.*)?$ {
  proxy_pass http://atlas-<service-name>.atlas.svc.cluster.local:8080;
}
```

#### Real Example

```nginx
location ~ ^/api/storage(/.*)?$ {
  proxy_pass http://atlas-storage.atlas.svc.cluster.local:8080;
}
```

**Breakdown:**
- `location ~ ^/api/storage(/.*)?$` - Regex pattern matching `/api/storage` and all subpaths
- `proxy_pass http://atlas-storage.atlas.svc.cluster.local:8080` - Kubernetes service URL
- Service name format: `atlas-<service>.atlas.svc.cluster.local:8080`
- Port is typically `8080` (verify in service's Kubernetes deployment if different)

---

### Placement in File

Place new routes **alphabetically** among existing services for maintainability.

**Example:**
```nginx
location ~ ^/api/shops(/.*)?$ {
  proxy_pass http://atlas-npc-shops.atlas.svc.cluster.local:8080;
}

location ~ ^/api/storage(/.*)?$ {
  proxy_pass http://atlas-storage.atlas.svc.cluster.local:8080;
}

location ~ ^/api/tenants(/.*)?$ {
  proxy_pass http://atlas-tenants.atlas.svc.cluster.local:8080;
}
```

---

### Verification Steps

After updating ingress configuration:

1. **Apply the configuration** (if testing locally):
   ```bash
   kubectl apply -f atlas-ingress.yml
   ```

2. **Wait for nginx reload**:
   ```bash
   kubectl rollout status deployment/atlas-ingress -n atlas
   ```

3. **Test the endpoint**:
   ```bash
   curl -H "TENANT_ID: test-tenant" http://dev.atlas.home/api/storage/accounts/123?worldId=0
   ```

---

## Service Documentation

### When to Update README

Update the service's README.md when:
- ✅ Adding new REST endpoints
- ✅ Changing endpoint paths or parameters
- ✅ Adding/removing query parameters
- ✅ Changing request/response formats
- ✅ Adding new Kafka commands or events
- ✅ Changing Kafka message structures

---

### README File Location

**File:** `services/atlas-<service>/atlas.com/<service>/README.md`

**Example:** `services/atlas-storage/atlas.com/storage/README.md`

---

### REST Endpoints Section

#### Standard Format

```markdown
### REST Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/storage/accounts/{accountId}?worldId={worldId}` | Get storage metadata (capacity, mesos) |
| POST | `/api/storage/accounts/{accountId}?worldId={worldId}` | Create storage (default capacity: 4, mesos: 0) |
| GET | `/api/storage/accounts/{accountId}/assets?worldId={worldId}` | Get all assets in storage |
| GET | `/api/storage/accounts/{accountId}/assets/{assetId}?worldId={worldId}` | Get single asset by ID |
```

#### Critical Requirements

1. **Exact Path Match**: Path must match the actual `router.HandleFunc()` definition in `resource.go`
2. **Parameter Style**: Clearly distinguish path parameters `{accountId}` from query parameters `?worldId={worldId}`
3. **Description**: Include key details like default values, return types, or side effects

---

### Common Documentation Errors

#### ❌ Wrong: Path Parameters vs Query Parameters

**Documentation says:**
```markdown
| GET | `/api/gis/{worldId}/accounts/{accountId}/storage` | Get storage |
```

**Implementation actually uses:**
```go
router.HandleFunc("/storage/accounts/{accountId}", handler).Methods(http.MethodGet)
// with worldId as query parameter: ?worldId={worldId}
```

**Correct documentation:**
```markdown
| GET | `/api/storage/accounts/{accountId}?worldId={worldId}` | Get storage |
```

---

#### ❌ Wrong: Incorrect Base Path

**Documentation says:**
```markdown
| GET | `/api/gis/storage/accounts/{accountId}` | Get storage |
```

**Service registers routes with:**
```go
// resource.go - routes start with /storage
router.HandleFunc("/storage/accounts/{accountId}", ...)
```

**Correct documentation:**
```markdown
| GET | `/api/storage/accounts/{accountId}?worldId={worldId}` | Get storage |
```

**Note:** The `/api` prefix comes from ingress, not the service itself.

---

### Kafka Commands Section

#### Standard Format

```markdown
### Kafka Commands

**Topic**: `COMMAND_TOPIC_STORAGE`

| Command | Description |
|---------|-------------|
| `DEPOSIT` | Deposit item into storage |
| `WITHDRAW` | Withdraw item from storage |
| `UPDATE_MESOS` | Update storage mesos |
| `DEPOSIT_ROLLBACK` | Rollback a deposit (saga compensation) |
| `ARRANGE` | Merge and sort stackable items in storage |
```

#### Requirements

1. **Topic Name**: Document the actual Kafka topic constant name
2. **All Commands**: Include ALL commands the service consumes, not just common ones
3. **Clear Descriptions**: Explain what each command does and when it's used

---

### Kafka Events Section

#### Standard Format

```markdown
### Kafka Events

**Topic**: `EVENT_TOPIC_STORAGE_STATUS`

| Event | Description |
|-------|-------------|
| `DEPOSITED` | Item was deposited |
| `WITHDRAWN` | Item was withdrawn |
| `MESOS_UPDATED` | Mesos amount changed |
| `ARRANGED` | Items were merged and sorted |
| `ERROR` | Operation failed |
```

#### Requirements

1. **Topic Name**: Document the actual event topic constant name
2. **All Events**: Include ALL events the service produces
3. **Include Error Events**: Don't forget error/failure events

---

## Verification Checklist

Before claiming documentation is complete:

### REST Endpoints
- [ ] Every endpoint in `resource.go` is documented
- [ ] Path parameters use `{paramName}` syntax
- [ ] Query parameters use `?paramName={paramName}` syntax
- [ ] HTTP methods (GET, POST, PUT, DELETE) are correct
- [ ] Descriptions mention key details (defaults, side effects, etc.)

### Kafka Integration
- [ ] All consumed commands are documented with correct topic name
- [ ] All produced events are documented with correct topic name
- [ ] Command/event descriptions are clear and accurate

### Ingress Configuration
- [ ] Service has a location block in `atlas-ingress.yml`
- [ ] Location pattern matches service's base path
- [ ] Kubernetes service name is correct
- [ ] Route is placed alphabetically

---

## Example: Complete Documentation Update

### Scenario
Adding a new endpoint to atlas-storage: `POST /api/storage/accounts/{accountId}/assets`

### Step 1: Implement the Endpoint

**File:** `services/atlas-storage/atlas.com/storage/asset/resource.go`
```go
func InitializeRoutes(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
    return func(db *gorm.DB) server.RouteInitializer {
        return func(router *mux.Router, l logrus.FieldLogger) {
            // ... existing routes ...

            // NEW ENDPOINT
            router.HandleFunc("/storage/accounts/{accountId}/assets",
                server.RegisterInputHandler[CreateAssetRequest](l)(si)("create-asset", createAssetHandler(db)),
            ).Methods(http.MethodPost)
        }
    }
}
```

### Step 2: Verify Ingress Configuration

**File:** `atlas-ingress.yml`

Check if route exists:
```nginx
location ~ ^/api/storage(/.*)?$ {
  proxy_pass http://atlas-storage.atlas.svc.cluster.local:8080;
}
```

✅ Route exists - the new endpoint will be accessible at `/api/storage/accounts/{accountId}/assets`

### Step 3: Update README

**File:** `services/atlas-storage/atlas.com/storage/README.md`

**Before:**
```markdown
### REST Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/storage/accounts/{accountId}?worldId={worldId}` | Get storage metadata |
| GET | `/api/storage/accounts/{accountId}/assets?worldId={worldId}` | Get all assets |
```

**After:**
```markdown
### REST Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/storage/accounts/{accountId}?worldId={worldId}` | Get storage metadata |
| POST | `/api/storage/accounts/{accountId}?worldId={worldId}` | Create storage |
| GET | `/api/storage/accounts/{accountId}/assets?worldId={worldId}` | Get all assets |
| POST | `/api/storage/accounts/{accountId}/assets?worldId={worldId}` | Create asset in storage |
```

---

## Anti-Patterns

### ❌ Don't Assume Ingress Exists

**Wrong Assumption:**
"The service defines REST endpoints, so they must be accessible through ingress"

**Reality:**
Services can have REST endpoints that aren't exposed externally. Always verify ingress configuration.

---

### ❌ Don't Copy Documentation from Another Service

**Wrong Approach:**
Copy documentation structure from another service without verifying paths match your implementation

**Correct Approach:**
1. Read your `resource.go` to see actual routes
2. Check ingress for actual external paths
3. Write documentation matching your implementation

---

### ❌ Don't Skip "Minor" Changes

**Wrong:**
"I only changed a query parameter from required to optional, no need to update docs"

**Correct:**
Update documentation for ANY API contract change, no matter how small.

---

## Integration with Workflow

This pattern integrates into the Standard Implementation Workflow:

1. **Implement changes** to primary files
2. **→ Update ingress** if REST endpoints added/modified ← YOU ARE HERE
3. **→ Update README** if API contracts changed ← YOU ARE HERE
4. Update mocks if interfaces changed
5. Run tests
6. Fix failures
7. Verify build
8. Report results

---

## Summary

### Key Takeaways

1. **Ingress is not automatic** - REST endpoints won't be accessible without ingress configuration
2. **Documentation must match implementation** - Verify paths and parameters match actual code
3. **Update both together** - Don't update ingress without documentation or vice versa
4. **Query vs Path parameters** - Distinguish clearly in documentation

### Quick Reference

| Task | File | What to Update |
|------|------|---------------|
| Add REST endpoint | `atlas-ingress.yml` | Add nginx location block |
| Add REST endpoint | `services/.../README.md` | Add row to REST Endpoints table |
| Add Kafka command | `services/.../README.md` | Add row to Kafka Commands table |
| Add Kafka event | `services/.../README.md` | Add row to Kafka Events table |

---
