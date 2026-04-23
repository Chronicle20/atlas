# Nginx Ingress SPOF — Context & Key Files

Last Updated: 2026-02-19

## Key Files

### Infrastructure
| File | Purpose |
|------|---------|
| `atlas-ingress.yml` | Nginx deployment, service, configmap, and ingress — the central file to modify |
| `services/atlas-env.yaml` | Shared ConfigMap with `BASE_SERVICE_URL` and all topic/infra env vars |
| `tools/debug-start.sh` | Debug tooling that rewrites nginx ConfigMap — must be updated |
| `tools/debug-stop.sh` | Restores nginx ConfigMap after debugging |

### Library Code
| File | Purpose |
|------|---------|
| `libs/atlas-rest/requests/url.go` | `RootUrl()` function — resolves `DOMAIN_SERVICE_URL` or falls back to `BASE_SERVICE_URL` |
| `libs/atlas-rest/requests/header.go` | `TenantHeaderDecorator` — injects tenant headers on outbound requests |
| `libs/atlas-rest/requests/client.go` | Shared HTTP client with connection pooling and timeouts |

### Per-Service Request Files
Every service that makes outbound REST calls has one or more `requests.go` files defining URL construction. Pattern:
```
services/<name>/atlas.com/<module>/<domain>/requests.go
```

## Domain-to-Service Mapping (Derived from nginx config + RootUrl calls)

| RootUrl Domain | Nginx Path Pattern | Backend Service | K8s DNS |
|----------------|-------------------|-----------------|---------|
| `CHARACTERS` | `/api/characters` | atlas-character | `atlas-character.atlas.svc.cluster.local:8080` |
| `DATA` | `/api/data` | atlas-data | `atlas-data.atlas.svc.cluster.local:8080` |
| `SKILLS` | `/api/characters/{id}/skills` | atlas-skills | `atlas-skills.atlas.svc.cluster.local:8080` |
| `INVENTORY` | `/api/characters/{id}/inventory` | atlas-inventory | `atlas-inventory.atlas.svc.cluster.local:8080` |
| `PETS` | `/api/pets`, `/api/characters/{id}/pets` | atlas-pets | `atlas-pets.atlas.svc.cluster.local:8080` |
| `MAPS` | `/api/worlds/.../maps/.../characters` | atlas-maps | `atlas-maps.atlas.svc.cluster.local:8080` |
| `MONSTERS` | `/api/monsters`, `/api/worlds/.../monsters` | atlas-monsters | `atlas-monsters.atlas.svc.cluster.local:8080` |
| `PARTIES` | `/api/parties` | atlas-parties | `atlas-parties.atlas.svc.cluster.local:8080` |
| `PARTY_QUESTS` | `/api/party-quests` | atlas-party-quests | `atlas-party-quests.atlas.svc.cluster.local:8080` |
| `CASHSHOP` | `/api/cash-shop`, `/api/accounts/{id}/cash-shop` | atlas-cashshop | `atlas-cashshop.atlas.svc.cluster.local:8080` |
| `BANS` | `/api/bans` | atlas-ban | `atlas-ban.atlas.svc.cluster.local:8080` |
| `BUFFS` | `/api/characters/{id}/buffs` | atlas-buffs | `atlas-buffs.atlas.svc.cluster.local:8080` |
| `RATES` | `/api/worlds/.../rates` | atlas-rates | `atlas-rates.atlas.svc.cluster.local:8080` |
| `STORAGE` | `/api/storage` | atlas-storage | `atlas-storage.atlas.svc.cluster.local:8080` |
| `DROPS_INFORMATION` | `/api/drops/seed`, `/api/continents/drops`, `/{id}/drops` | atlas-drop-information | `atlas-drop-information.atlas.svc.cluster.local:8080` |
| `QUESTS` | `/api/characters/{id}/quests` | atlas-quest | `atlas-quest.atlas.svc.cluster.local:8080` |
| `CONFIGURATIONS` | `/api/configurations` | atlas-configurations | `atlas-configurations.atlas.svc.cluster.local:8080` |
| `TENANTS` | `/api/tenants` | atlas-tenants | `atlas-tenants.atlas.svc.cluster.local:8080` |
| `BUDDIES` | `/api/characters/{id}/buddy-list` | atlas-buddies | `atlas-buddies.atlas.svc.cluster.local:8080` |
| `GUILDS` | `/api/guilds` | atlas-guilds | `atlas-guilds.atlas.svc.cluster.local:8080` |
| `TRANSPORTS` | `/api/transports` | atlas-transports | `atlas-transports.atlas.svc.cluster.local:8080` |
| `EFFECTIVE_STATS` | `/api/worlds/.../stats` | atlas-effective-stats | `atlas-effective-stats.atlas.svc.cluster.local:8080` |
| `MARRIAGE` | (not in nginx — likely new) | atlas-marriage | TBD |

## Critical Path Concern

The URL construction pattern is:
```go
func getBaseRequest() string {
    return requests.RootUrl("CHARACTERS")  // currently returns "http://ingress:80/api/"
}

func requestById(id uint32) requests.Request[RestModel] {
    return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+"characters/%d", id))
}
```

With direct URLs, `CHARACTERS_SERVICE_URL` would be `http://atlas-character:8080/`. The concatenated URL becomes `http://atlas-character:8080/characters/123`. This must match the route the target service actually listens on.

**Each service must be audited** to confirm it handles the path without the `/api/` prefix. The nginx config strips `/api/` implicitly because the location regex captures it.

## Key Decisions Made

1. **Hybrid approach**: Keep nginx for external traffic, use direct DNS for internal
2. **Phase 1 first**: Resilience improvements are quick wins with zero code changes
3. **Incremental rollout**: Migrate one service pair at a time in Phase 2
4. **Preserve fallback**: `BASE_SERVICE_URL` stays as a safety net for unmapped domains

## Dependencies

- No library changes needed — `_SERVICE_URL` mechanism already exists
- Phase 2 depends on Phase 1 being complete (so nginx is resilient during transition)
- Phase 3 can proceed in parallel with Phase 2
- Phase 4 depends on Phase 2 being substantially complete
