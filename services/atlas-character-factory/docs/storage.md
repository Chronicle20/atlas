# Character Factory Storage

## Tables

None. This service does not use persistent database storage.

## In-Memory Storage

### FollowUpSagaTemplateStore

Thread-safe singleton store for follow-up saga templates.

| Key Format              | Value Type            |
|-------------------------|-----------------------|
| {tenantId}:{characterName} | FollowUpSagaTemplate |

### SagaCompletionTrackerStore

Thread-safe singleton store for saga completion tracking.

| Key Format      | Value Type              |
|-----------------|-------------------------|
| {transactionId} | *SagaCompletionTracker  |

Both character creation and follow-up saga transaction IDs point to the same tracker instance.

## Relationships

None.

## Indexes

None.

## Migration Rules

Not applicable. This service uses in-memory storage only.
