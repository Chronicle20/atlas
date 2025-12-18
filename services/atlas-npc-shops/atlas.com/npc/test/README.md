# Test Utilities

This package provides utilities for testing the shops and commodities processors.

## Database Utilities

### SetupTestDB

Creates a new SQLite in-memory database for testing.

```go
db := test.SetupTestDB(t, migrations...)
```

### CleanupTestDB

Cleans up the test database.

```go
test.CleanupTestDB(t, db)
```

## Tenant Utilities

### CreateTestContext

Creates a context with a test tenant for testing.

```go
ctx := test.CreateTestContext()
```

### CreateMockTenant

Creates a new mock tenant with the given ID.

```go
tenant := test.CreateMockTenant("00000000-0000-0000-0000-000000000001")
```

### CreateDefaultMockTenant

Creates a new mock tenant with a default ID.

```go
tenant := test.CreateDefaultMockTenant()
```

### ToContext

Adds the mock tenant to the context.

```go
ctx = test.ToContext(ctx, tenant)
```

### MustFromContext

Retrieves the mock tenant from the context.

```go
tenant := test.MustFromContext(ctx)
```

## Processor Utilities

### CreateCommoditiesProcessor

Creates a new commodities processor for testing.

```go
processor, db, cleanup := test.CreateCommoditiesProcessor(t)
defer cleanup()
```

### CreateCommoditiesProcessorWithDB

Creates a new commodities processor with an existing database.

```go
commoditiesProcessor := test.CreateCommoditiesProcessorWithDB(t, db)
```

### CreateShopsProcessor

Creates a new shops processor for testing.

```go
processor, db, cleanup := test.CreateShopsProcessor(t)
defer cleanup()
```

### WithMockTenant

Creates a new context with a mock tenant.

```go
ctx = test.WithMockTenant(ctx, "00000000-0000-0000-0000-000000000001")
```

## Example Usage

```go
func TestShopsProcessor(t *testing.T) {
    // Create processor, database, and cleanup function
    processor, db, cleanup := test.CreateShopsProcessor(t)
    defer cleanup()

    // Run tests
    t.Run("TestGetByNpcId", func(t *testing.T) {
        // Test data
        npcId := uint32(2001)
        templateId := uint32(3001)
        mesoPrice := uint32(5000)
        tokenPrice := uint32(2500)

        // Create test commodity for the shop
        commoditiesProcessor := test.CreateCommoditiesProcessorWithDB(t, db)
        _, err := commoditiesProcessor.CreateCommodity(npcId, templateId, mesoPrice, tokenPrice)
        if err != nil {
            t.Fatalf("Failed to create test commodity: %v", err)
        }

        // Get shop by NPC ID
        shop, err := processor.GetByNpcId(npcId)
        if err != nil {
            t.Fatalf("Failed to get shop by NPC ID: %v", err)
        }

        // Verify shop
        if shop.NpcId() != npcId {
            t.Errorf("Expected NPC ID %d, got %d", npcId, shop.NpcId())
        }
    })
}
```