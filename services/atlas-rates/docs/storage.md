# Storage

## Tables

None. This service uses in-memory state only.

The following singleton structures hold all state:

- **Registry** (`character/registry.go`): Maps tenant -> characterId -> Model. Stores rate factors per character.
- **ItemTracker** (`character/item_tracker.go`): Maps tenant -> characterId -> templateId -> TrackedItem. Stores time-based rate items.
- **initializedCharacters** (`character/initializer.go`): Maps tenant -> characterId -> bool. Tracks which characters have been lazily initialized.

All state is lost on service restart and rebuilt lazily from external services on the next rate query or map change event.

## Relationships

Not applicable.

## Indexes

Not applicable.

## Migration Rules

Not applicable.
