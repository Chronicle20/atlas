# Storage

This service does not use persistent storage. All state is either:

- Fetched on demand from external services via REST (character data, inventory data, pet data, reference data)
- Maintained in-memory in the character location registry (populated from character status events, not persisted)
- Communicated via Kafka commands to services that own the persistent state

The character location registry is a volatile in-memory map that is rebuilt from character status events on service restart.
