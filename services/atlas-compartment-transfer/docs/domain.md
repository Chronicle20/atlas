# Compartment Transfer Domain

## Responsibility

Orchestrates the transfer of assets between compartments across different inventory systems (character, cash shop, storage).

## Core Models

### TransferInfo

Holds information about an in-progress transfer.

| Field | Type | Description |
|-------|------|-------------|
| Step | TransactionStep | Function to execute the next saga step |
| WorldId | byte | World identifier |
| CharacterId | uint32 | Character identifier |
| AccountId | uint32 | Account identifier |
| AssetId | uint32 | Asset being transferred |
| ToCompartmentId | uuid.UUID | Destination compartment identifier |
| ToCompartmentType | byte | Destination compartment type |
| ToInventoryType | string | Destination inventory type |

### TransactionCache

Singleton cache that stores in-progress transfer information keyed by transaction ID.

| Method | Description |
|--------|-------------|
| Store | Stores transfer information for a transaction |
| Get | Retrieves transfer information by transaction ID |
| Delete | Removes a transaction from the cache |

## Invariants

- Each transfer is identified by a unique transaction ID
- A transfer must complete the accept phase before the release phase
- Transfer information is cached in memory until the transfer completes or fails

## Processors

### Processor

Orchestrates the compartment transfer saga.

| Method | Description |
|--------|-------------|
| Process | Initiates a transfer by sending an accept command to the destination compartment |
| HandleAccepted | Handles destination acceptance and triggers release from source |
| HandleReleased | Handles source release and emits transfer completed event |
| HandleError | Handles transfer failures and cleans up transaction cache |
