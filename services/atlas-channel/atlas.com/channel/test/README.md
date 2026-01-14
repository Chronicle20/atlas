# Test Utilities

This package provides reusable mock implementations and test utilities for testing the atlas-channel service.

## Context Utilities

### CreateTestContext

Creates a context with a mock tenant for testing:

```go
import "atlas-channel/test"

func TestSomething(t *testing.T) {
    ctx := test.CreateTestContext()
    // ctx now has a tenant with ID 00000000-0000-0000-0000-000000000001
}
```

### CreateTestContextWithTenant

Creates a context with a specific tenant ID:

```go
tenantId := uuid.New()
ctx := test.CreateTestContextWithTenant(tenantId)
```

## Mock Kafka Producer

The `MockProducer` captures Kafka messages instead of sending them to a real broker.

### Basic Usage

```go
import "atlas-channel/test"

func TestKafkaEmission(t *testing.T) {
    // Create mock producer
    mockProducer := test.NewMockProducer()

    // Get the provider to inject into your processor
    provider := mockProducer.Provider()

    // ... run your test code that emits Kafka messages ...

    // Verify messages were emitted
    if !mockProducer.HasMessage("TOPIC_SESSION_STATUS") {
        t.Error("Expected session status message to be emitted")
    }

    // Get all messages for a topic
    messages := mockProducer.MessagesForTopic("TOPIC_SESSION_STATUS")
    if len(messages) != 1 {
        t.Errorf("Expected 1 message, got %d", len(messages))
    }
}
```

### Simulating Errors

```go
func TestKafkaError(t *testing.T) {
    mockProducer := test.NewMockProducer()
    mockProducer.SetError(errors.New("kafka unavailable"))

    // ... test error handling ...

    // Reset for next test
    mockProducer.Reset()
}
```

### MockProducer Methods

| Method | Description |
|--------|-------------|
| `NewMockProducer()` | Creates a new mock producer |
| `Provider()` | Returns a `producer.Provider` for dependency injection |
| `Messages()` | Returns all captured messages |
| `MessageCount()` | Returns total number of captured messages |
| `MessagesForTopic(topic)` | Returns messages for a specific topic |
| `HasMessage(topic)` | Checks if any message was sent to a topic |
| `Reset()` | Clears all captured messages and errors |
| `SetError(err)` | Configures the producer to return an error |

## Mock HTTP Server

The mock HTTP server utilities help test REST client code.

### Basic Usage

```go
import "atlas-channel/test"

func TestHTTPClient(t *testing.T) {
    server := test.NewMockServer(test.MockServerConfig{
        Routes: []test.MockRoute{
            {
                Method:   "GET",
                Path:     "/api/characters/123",
                Status:   200,
                Response: test.CreateCharacterResponse(123, "TestChar", 10, 0, 100000000),
            },
        },
    })
    defer server.Close()

    // Use server.URL as the base URL for your HTTP client
}
```

### Wildcard Routes

Use `*` suffix for prefix matching:

```go
test.MockRoute{
    Method:   "GET",
    Path:     "/api/characters/*",  // Matches /api/characters/123, /api/characters/456, etc.
    Status:   200,
    Response: characterListResponse,
}
```

### JSON:API Helpers

```go
// Create a single resource
resource := test.NewJSONAPIResource("characters", "123", map[string]interface{}{
    "name":  "TestChar",
    "level": 10,
})
doc := test.NewJSONAPIDocument(resource)

// Create a list response
resources := []test.JSONAPIResource{resource1, resource2}
listDoc := test.NewJSONAPIListDocument(resources)

// Error responses
notFound := test.NewJSONAPINotFoundResponse()
customError := test.NewJSONAPIErrorResponse("400", "Bad Request", "Invalid parameter")
```

### Pre-built Response Helpers

| Function | Description |
|----------|-------------|
| `CreateCharacterResponse(id, name, level, jobId, mapId)` | Creates a complete character JSON:API response |
| `NewJSONAPINotFoundResponse()` | Standard 404 error response |
| `NewJSONAPIErrorResponse(status, title, detail)` | Custom error response |

## Example: Testing a Processor

```go
package mypackage_test

import (
    "atlas-channel/test"
    "testing"
)

func TestProcessorEmitsEvent(t *testing.T) {
    // Setup
    ctx := test.CreateTestContext()
    mockProducer := test.NewMockProducer()

    // Create processor with mock dependencies
    // processor := NewProcessor(logger, ctx, mockProducer.Provider())

    // Execute
    // err := processor.DoSomething()

    // Verify
    // if err != nil {
    //     t.Fatalf("unexpected error: %v", err)
    // }
    // if !mockProducer.HasMessage("EXPECTED_TOPIC") {
    //     t.Error("expected message not emitted")
    // }
}
```
