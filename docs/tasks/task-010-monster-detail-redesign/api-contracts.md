# API Contracts — Monster Detail Redesign

All contracts are JSON:API over `application/vnd.api+json` with the existing atlas-data tenant header contract (`TENANT_ID` / `REGION` / `MAJOR_VERSION` / `MINOR_VERSION`). Examples below elide the `jsonapi` + `links` envelope for brevity.

## 1. `GET /data/monsters/{monsterId}/maps` (new)

Returns the list of maps where the given monster template spawns on the active tenant, sorted by spawn density descending.

### Request

```
GET /api/data/monsters/100100/maps HTTP/1.1
Accept: application/vnd.api+json
TENANT_ID: <uuid>
REGION: GMS
MAJOR_VERSION: 83
MINOR_VERSION: 1
```

### Path params

| Name | Type | Notes |
|---|---|---|
| `monsterId` | `uint32` | Monster template id. Validated via existing `rest.ParseMonsterId`. |

### Responses

**200 OK** — empty or non-empty list. Always returns `data: []` for a monster with no spawns, never `404`.

```json
{
  "data": [
    {
      "type": "monster-spawn-maps",
      "id": "104000000",
      "attributes": {
        "name": "Lith Harbor",
        "streetName": "Victoria Road",
        "spawnCount": 6
      }
    },
    {
      "type": "monster-spawn-maps",
      "id": "100000000",
      "attributes": {
        "name": "Henesys",
        "streetName": "Victoria Road",
        "spawnCount": 3
      }
    }
  ]
}
```

Ordering: `spawn_count DESC, name ASC`. The UI re-sorts defensively but should receive the rows in this order.

**400 Bad Request** — `monsterId` is not a valid `uint32`.

**500 Internal Server Error** — DB failure. No partial payload; the handler writes the status and returns.

### Notes

- `id` in each resource is the map id (so `<Link to={`/maps/${id}`}>` works without transformation).
- `type` is the fixed string `"monster-spawn-maps"` (returned by `MonsterSpawnMapRestModel.GetName()`). It's a read-only resource type — no POST/PATCH/DELETE.
- `spawnCount` counts raw spawn entries in the map document, not unique spawn points. A map that re-uses the same coordinate for two spawn entries counts as 2.

## 2. `GET /data/mob-skills/{skillId}` (modified)

Unchanged route, handler, and shape — but the response payload gains a `name` attribute. Existing consumers that ignore unknown fields are unaffected.

### Before

```json
{
  "data": [
    {
      "type": "mob-skills",
      "id": "1000001",
      "attributes": {
        "mp_con": 0,
        "duration": 30000,
        "hp": 100,
        "x": 0,
        "y": 0,
        "prop": 100,
        "interval": 5000,
        "count": 1,
        "limit": 0,
        "lt_x": 0,
        "lt_y": 0,
        "rb_x": 0,
        "rb_y": 0,
        "summon_effect": 0,
        "summons": []
      }
    }
  ]
}
```

### After

```json
{
  "data": [
    {
      "type": "mob-skills",
      "id": "1000001",
      "attributes": {
        "name": "Power Up",
        "mp_con": 0,
        "duration": 30000,
        "hp": 100,
        "x": 0,
        "y": 0,
        "prop": 100,
        "interval": 5000,
        "count": 1,
        "limit": 0,
        "lt_x": 0,
        "lt_y": 0,
        "rb_x": 0,
        "rb_y": 0,
        "summon_effect": 0,
        "summons": []
      }
    }
  ]
}
```

- `name` is identical for every level row returned by this endpoint (one name per skill id).
- When `String.wz/MobSkill.img.xml` is missing for the tenant, `name` returns an empty string. The UI treats empty as "no name" and renders the numeric id as a fallback.

## 3. UI consumption

```ts
// services/atlas-ui/src/services/api/monsters.service.ts
async getMonsterMaps(monsterId: string): Promise<MonsterSpawnMapData[]> {
  return api.getList<MonsterSpawnMapData>(`/api/data/monsters/${monsterId}/maps`);
}

// services/atlas-ui/src/services/api/mob-skills.service.ts
async getMobSkillName(skillId: number): Promise<string> {
  const rows = await api.getList<MobSkillData>(`/api/data/mob-skills/${skillId}`);
  return rows[0]?.attributes.name ?? "";
}

// services/atlas-ui/src/types/models/monster.ts (additions)
export interface MonsterSpawnMapAttributes {
  name: string;
  streetName: string;
  spawnCount: number;
}

export interface MonsterSpawnMapData {
  id: string;
  type: string;
  attributes: MonsterSpawnMapAttributes;
}
```

Hooks follow the existing 5-minute stale / 10-minute gc pattern from `useMonsterDrops`:

```ts
export function useMonsterMaps(monsterId: string) {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: ['monsters', monsterId, 'maps'],
    queryFn: () => monstersService.getMonsterMaps(monsterId),
    enabled: !!monsterId && !!activeTenant,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}
```
