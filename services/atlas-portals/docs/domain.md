# Domain

## Portal

### Responsibility

Represents a portal within a map. Portals transport characters between maps or trigger scripts.

### Core Models

**Model**

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Portal identifier |
| name | string | Portal name |
| target | string | Target portal name |
| portalType | uint8 | Portal type |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| targetMapId | uint32 | Target map identifier |
| scriptName | string | Script to execute |

### Invariants

- A portal with `targetMapId` of `999999999` has no target map.
- A portal with a non-empty `scriptName` has a script.

### Processors

**InMapProvider**

Provides all portals in a map from the DATA service.

**InMapByNameProvider**

Provides portals in a map by name from the DATA service.

**InMapByIdProvider**

Provides a portal in a map by id from the DATA service.

**GetInMapByName**

Returns the first portal matching a name in a map.

**GetInMapById**

Returns a portal by id in a map.

**Enter**

Processes a character entering a portal:
1. Checks if the portal is blocked for the character.
2. If blocked, enables character actions and returns.
3. Fetches portal data from DATA service.
4. If portal has a script, emits a portal actions command.
5. If portal has a target map, resolves the target portal and warps the character.
6. If target portal name not found, falls back to portal 0.
7. Otherwise, enables character actions.

**Warp**

Warps a character to a target map:
1. Retrieves all portals in the target map.
2. If no portals found, defaults to portal 0.
3. Otherwise, selects a random portal from the target map.

**WarpById**

Warps a character to a specific portal in a map.

**WarpToPortal**

Emits a character change map command.

---

## Blocked

### Responsibility

Tracks portals that are temporarily blocked for specific characters.

### Core Models

**Model**

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| mapId | uint32 | Map identifier |
| portalId | uint32 | Portal identifier |

### Invariants

- Blocked state is scoped to a tenant.
- Blocked state is cleared when a character logs out.
- Blocking the same portal twice is idempotent.

### Processors

**Registry**

Manages blocked portal state per character:
- `IsBlocked`: Checks if a portal is blocked for a character.
- `Block`: Marks a portal as blocked for a character.
- `Unblock`: Removes a portal from the blocked list for a character.
- `ClearForCharacter`: Removes all blocked portals for a character.
- `GetForCharacter`: Returns all blocked portals for a character.
