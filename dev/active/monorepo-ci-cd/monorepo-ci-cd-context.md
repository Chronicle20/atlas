# Monorepo CI/CD - Context Document

**Last Updated: 2025-12-19**

---

## Key Files Reference

### Current Workflow Files (to be replaced)
Each service/library has two workflow files in their `.github/workflows/` directory:

| File | Purpose | Example Location |
|------|---------|------------------|
| `pull-request.yml` | PR validation (build/test) | `services/atlas-account/.github/workflows/pull-request.yml` |
| `main-snapshot.yml` | Main branch Docker publish | `services/atlas-account/.github/workflows/main-snapshot.yml` |

### Key Configuration Files
| File | Purpose | Path |
|------|---------|------|
| `go.work` | Go workspace definition | `/go.work` |
| `go.work.sum` | Go workspace checksums | `/go.work.sum` |

### Build Tools
| Script | Purpose | Path |
|--------|---------|------|
| `build-services.sh` | Build all Docker images | `/tools/build-services.sh` |
| `test-all-go.sh` | Test all Go modules | `/tools/test-all-go.sh` |
| `tidy-all-go.sh` | Tidy all Go modules | `/tools/tidy-all-go.sh` |

### New Files to Create
| File | Purpose | Path |
|------|---------|------|
| `pr-validation.yml` | PR workflow | `/.github/workflows/pr-validation.yml` |
| `main-publish.yml` | Main workflow | `/.github/workflows/main-publish.yml` |
| `detect-changes/action.yml` | Change detection | `/.github/actions/detect-changes/action.yml` |
| `go-test/action.yml` | Go test action | `/.github/actions/go-test/action.yml` |
| `node-test/action.yml` | Node.js test action | `/.github/actions/node-test/action.yml` |
| `docker-build/action.yml` | Docker build action | `/.github/actions/docker-build/action.yml` |
| `services.json` | Service metadata | `/.github/config/services.json` |

---

## Technical Decisions

### Decision 1: Change Detection Approach
**Options Considered**:
1. `dorny/paths-filter` - Popular action for path-based filtering
2. Custom `git diff` script - Full control, no dependencies
3. GitHub's built-in path filters - Limited to triggering, not matrix generation

**Selected**: `dorny/paths-filter` with custom post-processing

**Rationale**:
- Well-maintained, widely used
- Handles edge cases (merge commits, force pushes)
- Outputs usable for matrix generation
- Can be supplemented with custom logic for dependency detection

### Decision 2: Go Version Strategy
**Options Considered**:
1. Single pinned version (1.25.5)
2. Per-service versions (current state)
3. Use `stable` everywhere

**Selected**: Single pinned version (1.25.5)

**Rationale**:
- Consistent builds across all services
- go.work already specifies 1.25.5
- Easier to maintain and upgrade
- Avoids compatibility issues between services

### Decision 3: Docker Build Strategy
**Options Considered**:
1. Native Docker build per architecture
2. QEMU emulation for cross-architecture
3. Buildx with multiple builders

**Selected**: Native Docker build per architecture (existing approach)

**Rationale**:
- Already working in current workflows
- ARM64 runner available (ubuntu-24.04-arm)
- Native builds are faster than emulation
- Multi-arch manifest creation well understood

### Decision 4: Matrix Job Organization
**Options Considered**:
1. Single matrix for all services
2. Separate matrices for libs, services, UI
3. Per-type workflows

**Selected**: Separate matrices for libs, services, UI

**Rationale**:
- Different test configurations per type
- Libraries need coverage checks
- UI needs Node.js setup
- Clearer job organization
- Easier to debug failures

### Decision 5: Cache Strategy
**Options Considered**:
1. No caching (simple but slow)
2. Go module cache only
3. Full caching (Go modules + Docker layers)

**Selected**: Full caching (Go modules + Docker layers)

**Rationale**:
- Significant time savings
- GitHub Actions provides 10GB cache per repo
- Go modules change infrequently
- Docker layer caching reduces build time

---

## Dependencies Between Components

### Service Directory Structure
```
services/atlas-{name}/
├── .github/workflows/          # OLD - to be removed
├── atlas.com/{module}/         # Go module location
│   ├── go.mod
│   ├── go.sum
│   └── *.go
└── Dockerfile
```

### Library Directory Structure
```
libs/atlas-{name}/
├── .github/workflows/          # OLD - to be removed
├── go.mod
├── go.sum
└── *.go
```

### Go Module Paths
Services use nested module paths under `atlas.com/`:
- `atlas-account` → `services/atlas-account/atlas.com/account/`
- `atlas-login` → `services/atlas-login/atlas.com/login/`

Libraries use root paths:
- `atlas-model` → `libs/atlas-model/`
- `atlas-kafka` → `libs/atlas-kafka/`

### Docker Image Naming Convention
```
ghcr.io/chronicle20/{service-name}/{service-name}:latest
```

Examples:
- `ghcr.io/chronicle20/atlas-account/atlas-account:latest`
- `ghcr.io/chronicle20/atlas-ui/atlas-ui:latest`

---

## External Dependencies

### GitHub Actions Used
| Action | Version | Purpose |
|--------|---------|---------|
| `actions/checkout` | v4/v5 | Clone repository |
| `actions/setup-go` | v5 | Install Go |
| `actions/setup-node` | v4 | Install Node.js |
| `actions/cache` | v4 | Caching |
| `docker/setup-buildx-action` | v3 | Docker Buildx |
| `docker/login-action` | v3 | GHCR auth |
| `dorny/paths-filter` | v3 | Change detection |
| `codecov/codecov-action` | v4 | Coverage upload |

### Required Secrets
| Secret | Purpose | Scope |
|--------|---------|-------|
| `GHCR_TOKEN` | GitHub Container Registry auth | Repository |

### Runner Requirements
| Runner | Purpose |
|--------|---------|
| `ubuntu-latest` | AMD64 builds, tests |
| `ubuntu-24.04-arm` | ARM64 builds |

---

## Path Mapping

### Change Detection Path Patterns
```yaml
# Services
services/atlas-account/**  → atlas-account (go-service)
services/atlas-ui/**       → atlas-ui (node-service)

# Libraries
libs/atlas-model/**        → atlas-model (go-library)
libs/atlas-kafka/**        → atlas-kafka (go-library)

# Global triggers (rebuild all)
go.work                    → ALL go services/libraries
.github/**                 → ALL services
```

### Go Module to Service Mapping
| Service | Module Path | Working Directory |
|---------|-------------|-------------------|
| atlas-account | `atlas-account` | `services/atlas-account/atlas.com/account` |
| atlas-login | `atlas-login` | `services/atlas-login/atlas.com/login` |
| atlas-character | `atlas-character` | `services/atlas-character/atlas.com/character` |
| ... | ... | ... |

---

## Testing Configurations

### Go Services Test Command
```bash
cd services/atlas-{name}/atlas.com/{module}
go test -v ./...
```

### Go Libraries Test Command (with coverage)
```bash
cd libs/atlas-{name}
go test -race -v ./...
go test -cover -coverprofile=coverage.out ./...
# Check 75% coverage threshold
```

### Node.js UI Test Command
```bash
cd services/atlas-ui
npm install
npm test
npm run build
```

---

## Migration Notes

### Files to Remove After Migration
All workflow files in service/library subdirectories:
```bash
# Pattern
services/*/.github/workflows/*.yml
libs/*/.github/workflows/*.yml

# Count: 100 files (50 services/libs x 2 workflows)
```

### Backward Compatibility
- Same GHCR_TOKEN secret used
- Same Docker image names maintained
- Same multi-arch support (AMD64 + ARM64)
- Same `:latest` tag strategy

### Breaking Changes
- None expected for consumers of Docker images
- CI behavior change: only changed services are built/tested
- Workflow names will change in GitHub UI

---

## Troubleshooting Guide

### Common Issues

#### 1. Change Detection Not Triggering
**Symptom**: PR doesn't trigger expected service builds
**Check**:
- Verify paths-filter configuration
- Check git diff output
- Ensure base branch is correct

#### 2. Go Module Cache Miss
**Symptom**: go mod download runs every time
**Check**:
- Verify go.sum hash in cache key
- Check cache size limits
- Verify cache restore step

#### 3. Docker Build Fails
**Symptom**: Docker build step fails
**Check**:
- Verify context path is correct
- Check Dockerfile location
- Verify all build arguments

#### 4. GHCR Push Fails
**Symptom**: docker push returns 403
**Check**:
- Verify GHCR_TOKEN is set
- Check token permissions
- Verify image name format

#### 5. ARM64 Build Fails
**Symptom**: arm64 job fails, amd64 succeeds
**Check**:
- Verify runner availability
- Check architecture-specific code
- Review Dockerfile compatibility
