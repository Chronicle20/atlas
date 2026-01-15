# Saga Orchestration

## Transfer Saga

The compartment transfer saga coordinates asset movement between compartments using a two-phase approach.

## Saga Steps

### Phase 1: Accept

1. Receive TransferCommand
2. Determine destination inventory type (CHARACTER, CASH_SHOP, or STORAGE)
3. Send ACCEPT command to destination compartment
4. Store transfer info in TransactionCache with release step as next action
5. Wait for ACCEPTED event from destination

### Phase 2: Release

1. Receive ACCEPTED event from destination compartment
2. Retrieve transfer info from cache
3. Execute stored release step (send RELEASE command to source compartment)
4. Wait for RELEASED event from source

### Completion

1. Receive RELEASED event from source compartment
2. Emit COMPLETED status event
3. Remove transaction from cache

## Error Handling

On ERROR event from any compartment:
1. Remove transaction from cache
2. (Compensation not yet implemented)

## Inventory Types

| Type | Description |
|------|-------------|
| CHARACTER | Character inventory compartments |
| CASH_SHOP | Cash shop inventory compartments |
| STORAGE | Account storage compartments |
