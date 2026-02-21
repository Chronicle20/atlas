# Key Domain

## Responsibility

The key domain manages keyboard binding configurations for characters. Each key binding maps a physical key to a type and action.

## Core Models

### Model

Represents a key binding for a character.

| Field       | Type   | Description                        |
|-------------|--------|------------------------------------|
| characterId | uint32 | Character that owns the binding    |
| key         | int32  | Key identifier                     |
| theType     | int8   | Type of key binding                |
| action      | int32  | Action associated with the binding |

## Invariants

- A key binding is uniquely identified by tenant, character, and key.
- Default key bindings are created from predefined arrays when a character is created.

## Processors

### Processor Interface

| Method              | Description                                      |
|---------------------|--------------------------------------------------|
| ByCharacterIdProvider | Returns a provider for keys by character ID    |
| GetByCharacterId    | Retrieves all key bindings for a character       |
| Reset               | Deletes existing bindings and creates defaults   |
| CreateDefault       | Creates default key bindings for a character     |
| Delete              | Deletes all key bindings for a character         |
| ChangeKey           | Creates or updates a single key binding          |
