# Renovate Monorepo Configuration - Context

**Last Updated: 2025-12-19**

---

## Key Files

### Configuration Files
| File | Purpose | Status |
|------|---------|--------|
| `renovate.json` (root) | Centralized monorepo Renovate config | To be created |
| `go.work` | Go workspace defining all modules | Existing |
| `services/*/renovate.json` | Individual service configs | To be removed (43 files) |
| `libs/*/renovate.json` | Individual library configs | To be removed (6 files) |
| `services/atlas-ui/renovate.json` | UI service config | To be removed (1 file) |

### CI/CD Integration
| File | Purpose | Relevance |
|------|---------|-----------|
| `.github/workflows/pr-validation.yml` | PR validation workflow | Status checks for auto-merge |
| `.github/workflows/main-publish.yml` | Main branch publish | Post-merge build verification |
| `.github/actions/detect-changes/` | Change detection action | Validates affected services |

### Go Module Files
| Location | Count | Notes |
|----------|-------|-------|
| `services/atlas-*/atlas.com/*/go.mod` | 43 | Nested under atlas.com subdirectory |
| `libs/atlas-*/go.mod` | 6 | Direct under library directory |

### Package Files
| File | Manager | Notes |
|------|---------|-------|
| `services/atlas-ui/package.json` | npm | Next.js 15.5.9, React 19 |
| `services/atlas-ui/package-lock.json` | npm | Locked dependencies |

### Dockerfiles
| Location | Count | Base Image |
|----------|-------|------------|
| `services/atlas-*/Dockerfile` | 44 | `golang:*-alpine` or `node:*-alpine` |

---

## Key Decisions

### Decision 1: Centralized vs Distributed Configuration
**Decision**: Centralized root-level configuration
**Rationale**:
- Single source of truth for dependency management
- Easier maintenance (1 file vs 50)
- Better grouping across services
- Consistent policies enforced at repo level
**Trade-offs**:
- All services share same config (less flexibility)
- Single point of failure for config issues

### Decision 2: Grouping Strategy
**Decision**: Group by dependency type and update type
**Rationale**:
- Go version updates affect all services - single PR
- Dockerfile base images - single PR
- Chronicle20 internal deps - grouped by library
- External deps - per-service grouping
**Trade-offs**:
- Large groups may be harder to review
- Single failure blocks entire group

### Decision 3: Auto-merge Policy
**Decision**: Auto-merge patch/minor, manual merge major
**Rationale**:
- Patch updates are low risk
- Minor updates follow semver (backwards compatible)
- Major updates require review for breaking changes
**Trade-offs**:
- Some minor updates may introduce issues
- Relies on CI catching problems

### Decision 4: Post-Upgrade Tasks
**Decision**: Run `go mod tidy` after Go dependency updates
**Rationale**:
- Ensures go.sum stays in sync
- Removes unused dependencies
- Matches existing workflow behavior
**Trade-offs**:
- Adds processing time
- May fail in edge cases

### Decision 5: Go Workspace Handling
**Decision**: Use gomod manager with workspace awareness
**Rationale**:
- go.work defines all modules
- Renovate supports go.work since v34
- Consistent Go version across workspace
**Trade-offs**:
- Workspace support still maturing in Renovate
- May need fallback to individual modules

---

## Dependencies

### External Dependencies
| Dependency | Type | Purpose |
|------------|------|---------|
| Renovate GitHub App | Service | Automated dependency updates |
| GitHub Actions | CI | Status checks for auto-merge |
| GHCR | Registry | Published Docker images |

### Internal Dependencies
| From | To | Type |
|------|-----|------|
| Go Services | atlas-kafka | Import |
| Go Services | atlas-model | Import |
| Go Services | atlas-rest | Import |
| Go Services | atlas-socket | Import |
| Go Services | atlas-tenant | Import |
| Go Services | atlas-constants | Import |

### Renovate Presets Used
| Preset | Purpose |
|--------|---------|
| `config:recommended` | Base recommended settings |
| `:separateMajorReleases` | Keep major updates separate |
| `:combinePatchMinorReleases` | Group patch and minor together |

---

## Technology Stack Reference

### Go Services
- **Go Version**: 1.25.5 (from go.work)
- **Module Path Pattern**: `atlas.com/{service-name}`
- **External Deps**: OpenTelemetry, Kafka, GORM, logrus, gorilla/mux
- **Build**: Multi-stage Dockerfile with alpine base

### UI Service (atlas-ui)
- **Framework**: Next.js 15.5.9
- **React**: 19.0.0
- **TypeScript**: 5.8.2
- **Styling**: Tailwind CSS 4.0.12
- **State**: React Query (TanStack) 5.83.0
- **Testing**: Jest

### Docker Images
- **Go Services**: `golang:1.24.4-alpine` base
- **UI Service**: `node:22-alpine` base
- **Registry**: `ghcr.io/chronicle20/{service}/{service}:latest`
- **Architectures**: AMD64, ARM64 (multi-arch manifest)

---

## Service Inventory

### Go Services (43)
```
atlas-account          atlas-buddies          atlas-buffs
atlas-cashshop         atlas-chairs           atlas-chalkboards
atlas-channel          atlas-character        atlas-character-factory
atlas-compartment-transfer  atlas-configurations  atlas-consumables
atlas-data             atlas-drop-information atlas-drops
atlas-equipables       atlas-expressions      atlas-fame
atlas-families         atlas-guilds           atlas-inventory
atlas-invites          atlas-keys             atlas-login
atlas-maps             atlas-marriages        atlas-messages
atlas-messengers       atlas-monster-death    atlas-monsters
atlas-notes            atlas-npc-conversations atlas-npc-shops
atlas-parties          atlas-pets             atlas-portals
atlas-query-aggregator atlas-reactors         atlas-saga-orchestrator
atlas-skills           atlas-tenants          atlas-transports
atlas-world
```

### Go Libraries (6)
```
atlas-constants   atlas-kafka   atlas-model
atlas-rest        atlas-socket  atlas-tenant
```

### UI Services (1)
```
atlas-ui
```

---

## Renovate Manager Reference

### gomod Manager
- **Files**: `**/go.mod`
- **Features**: Go version updates, dependency updates, go mod tidy
- **Workspace**: Supports go.work files

### npm Manager
- **Files**: `**/package.json`, `**/package-lock.json`
- **Features**: Dependency updates, lock file maintenance
- **Registry**: npm registry (default)

### dockerfile Manager
- **Files**: `**/Dockerfile`
- **Features**: Base image updates, version extraction
- **Patterns**: `FROM golang:*`, `FROM node:*`

---

## Related Plans
| Plan | Status | Relationship |
|------|--------|--------------|
| [monorepo-ci-cd](../monorepo-ci-cd/) | Completed | CI provides status checks for Renovate auto-merge |
