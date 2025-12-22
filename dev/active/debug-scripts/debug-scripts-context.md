# Debug Scripts Context

Last Updated: 2025-12-21

## Key Files

### Primary Files to Create
| File | Purpose |
|------|---------|
| `tools/debug-start.sh` | Start debug session for a service |
| `tools/debug-stop.sh` | Stop debug session and restore service |

### Reference Files
| File | Purpose | Why Important |
|------|---------|---------------|
| `atlas-ingress.yml` | Nginx ConfigMap + Ingress | Contains all routing rules to modify |
| `tools/build-services.sh` | Existing script pattern | Template for script structure/style |
| `services/atlas-account/atlas-account.yml` | Service deployment example | Pattern for scaling operations |
| `services/atlas-env.yaml` | Environment config | Understand service configuration |

## Architecture Decisions

### Decision 1: State Storage Location
**Choice:** ConfigMap annotations on `atlas-ingress-configmap`
**Rationale:**
- Self-contained with the resource being modified
- Survives pod restarts and script interruptions
- No external state files to manage
- Easy to inspect with `kubectl describe configmap`

**Alternative considered:** Separate `atlas-debug-state` ConfigMap
**Rejected because:** Added complexity for minimal benefit

### Decision 2: Nginx Reload Method
**Choice:** `kubectl exec` into nginx pod and send HUP signal or run `nginx -s reload`
**Rationale:**
- No pod restart required
- Fast and non-disruptive
- Standard nginx reload pattern

**Implementation:**
```bash
kubectl exec -n atlas deploy/atlas-ingress -- nginx -s reload
```

### Decision 3: Service-to-Route Mapping
**Choice:** Dynamic discovery via grep on nginx.conf
**Rationale:**
- No hardcoded mapping to maintain
- Works with any new services automatically
- Handles services with multiple routes

**Pattern:** Search for `proxy_pass http://{service}.atlas.svc.cluster.local`

### Decision 4: Replica Count Handling
**Choice:** Store original replica count, not assume 1
**Rationale:**
- Future-proofs for scaled deployments
- Accurate restoration

## Service Mapping

### Multi-Route Services
These services have multiple nginx location blocks:

| Service | Routes |
|---------|--------|
| `atlas-cashshop` | `/api/accounts/[^/]+/cash-shop`, `/api/accounts/[^/]+/wallet`, `/api/characters/[^/]+/cash-shop`, `/api/cash-shop` |
| `atlas-skills` | `/api/characters/[^/]+/skills`, `/api/characters/[^/]+/macros` |
| `atlas-drops` | `/api/worlds/.../drops`, `/api/drops` |
| `atlas-pets` | `/api/pets`, `/api/characters/[^/]+/pets` |
| `atlas-monsters` | `/api/worlds/.../monsters`, `/api/monsters` |
| `atlas-reactors` | `/api/worlds/.../reactors`, `/api/reactors` |
| `atlas-npc-shops` | `/api/npcs`, `/api/shops` |
| `atlas-npc-conversations` | `/api/npcs/conversations`, `/api/npcs/[^/]+/conversations` |
| `atlas-notes` | `/api/characters/[^/]+/notes`, `/api/notes` |

All routes for a service MUST be updated together.

### Single-Route Services
Most services have exactly one location block. Examples:
- `atlas-account` → `/api/accounts`
- `atlas-character` → `/api/characters`
- `atlas-world` → `/api/worlds`
- `atlas-guilds` → `/api/guilds`

## State Format

### Annotation Key
```
debug.atlas.io/{service-name}
```

### Annotation Value
```
{original_replicas}
```

Example:
```
debug.atlas.io/atlas-account=1
```

The original URL pattern is reconstructed as:
```
http://{service-name}.atlas.svc.cluster.local:8080
```

This is deterministic, so we only need to store the replica count.

## Dependencies

### Script Dependencies
| Tool | Purpose | Check Command |
|------|---------|---------------|
| `kubectl` | K8s operations | `kubectl version --client` |
| `bash` 4.0+ | Script execution | `bash --version` |
| `sed` | Text replacement | `sed --version` |
| `grep` | Pattern matching | `grep --version` |

### Kubernetes Resources
| Resource | Namespace | Operations |
|----------|-----------|------------|
| ConfigMap `atlas-ingress-configmap` | atlas | get, patch |
| Deployment `atlas-ingress` | atlas | exec (nginx reload) |
| Deployment `{service}` | atlas | get, scale |

### Network Requirements
- Developer machine must be reachable from K8s cluster nodes
- If using WSL2, ensure port forwarding is configured
- Firewall must allow inbound connections on debug port

## Common Scenarios

### Scenario 1: Debug Single Service
```bash
# Start debugging atlas-account, redirect to local machine
./tools/debug-start.sh --service atlas-account --target 192.168.1.100:8080

# Stop debugging and restore
./tools/debug-stop.sh --service atlas-account
```

### Scenario 2: Debug Multiple Services
```bash
# Debug account and character services simultaneously
./tools/debug-start.sh --service atlas-account --target 192.168.1.100:8080
./tools/debug-start.sh --service atlas-character --target 192.168.1.100:8081

# Restore just account
./tools/debug-stop.sh --service atlas-account

# Later, restore character
./tools/debug-stop.sh --service atlas-character
```

### Scenario 3: Update Debug Target
```bash
# If IP changes, just run start again
./tools/debug-start.sh --service atlas-account --target 192.168.1.101:8080
```

### Scenario 4: Check Debug Status
```bash
# List what's currently being debugged
./tools/debug-start.sh --status

# List available services
./tools/debug-start.sh --list
```

## Testing Approach

### Manual Testing Steps
1. Start debug for `atlas-account`
2. Verify deployment scaled to 0
3. Verify nginx ConfigMap updated
4. Verify nginx reloaded
5. Curl request to see traffic goes to local
6. Stop debug
7. Verify deployment scaled back
8. Verify nginx ConfigMap restored
9. Verify nginx reloaded

### Edge Case Testing
- Run debug-start twice for same service
- Run debug-stop for non-debugged service
- Run debug-start for non-existent service
- Stop during multi-service debug session
- Restart scripts after interruption

## Recovery Procedures

### Nginx ConfigMap Corrupted
```bash
# Re-apply original ingress configuration
kubectl apply -f atlas-ingress.yml
```

### Debug State Lost (Annotations Missing)
```bash
# Check for scaled-down deployments
kubectl get deployments -n atlas -o json | \
  jq '.items[] | select(.spec.replicas == 0) | .metadata.name'

# Manually scale up
kubectl scale deployment {service} -n atlas --replicas=1
```

### Nginx Not Reloading
```bash
# Restart the nginx pod entirely
kubectl rollout restart deployment/atlas-ingress -n atlas
```

## Code Style Guidelines

Following existing `tools/` conventions:
- Use `#!/usr/bin/env bash`
- Use `set -euo pipefail`
- Define configuration variables at top
- Create helper functions (log, fail)
- Include precondition checks
- Provide clear status output
- Use consistent naming patterns
