# Monorepo CI/CD GitHub Actions Plan

**Last Updated: 2025-12-19**

---

## Executive Summary

This plan outlines the migration from 100 individual service/library GitHub Actions workflows to a unified monorepo CI/CD pipeline. The new system will detect changed services/libraries and run only the necessary tests and builds, significantly improving CI efficiency and maintainability.

### Goals
1. **Pull Request Workflow**: Detect changes, run tests (Go/Node.js), build Docker images (without publishing) to validate PRs
2. **Main Branch Workflow**: Detect changes, build and publish Docker images with `latest` tag to GHCR

### Key Benefits
- Single source of truth for CI/CD configuration
- Reduced duplication (from 100 files to 2 workflows + reusable components)
- Faster CI runs through intelligent change detection
- Consistent build and test processes across all services
- Easier maintenance and updates to CI/CD pipeline

---

## Current State Analysis

### Existing Workflow Structure
- **Location**: Individual `.github/workflows/` directories in each service/library folder
- **Total Workflows**: 100 files (50 services/libraries x 2 workflows each)
- **Patterns**:
  - `pull-request.yml` - PR validation (build + test)
  - `main-snapshot.yml` - Main branch Docker publish

### Workflow Types by Service Type

| Type | Count | PR Workflow | Main Workflow |
|------|-------|-------------|---------------|
| Go Services | 43 | Go build/test | Multi-arch Docker build/publish |
| Go Libraries | 6 | Go build/test + coverage | Go test + codecov OR Docker publish |
| React UI | 1 | Node.js build + Docker build | Multi-arch Docker build/publish |

### Current Trigger Configuration
- **PR**: `pull_request: branches: ["main"]`
- **Main**: `push: branches: ["main"]`
- **No path filtering** - All workflows run on every change

### Current Build Process

**Go Services/Libraries**:
```
go mod download → go mod tidy → go build ./... → go test -v ./...
```

**React UI**:
```
npm install → npm run build → docker build
```

**Docker Publishing** (Multi-arch):
- Parallel AMD64 and ARM64 builds
- Manifest creation combining both architectures
- Published to `ghcr.io/chronicle20/{service}/{service}:latest`

### Identified Issues with Current Approach
1. **Duplication**: 100 nearly identical workflow files
2. **No change detection**: All services tested/built on every PR
3. **Scattered configuration**: Updates require changes to 50+ files
4. **Inconsistent Go versions**: Mix of 1.24.4, 1.25.5, and 'stable'
5. **No caching**: Go modules and Docker layers not cached

---

## Proposed Future State

### New Workflow Architecture

```
.github/
├── workflows/
│   ├── pr-validation.yml          # Main PR workflow orchestrator
│   └── main-publish.yml           # Main branch publish orchestrator
├── actions/
│   ├── detect-changes/
│   │   └── action.yml             # Reusable change detection
│   ├── go-test/
│   │   └── action.yml             # Reusable Go test action
│   ├── node-test/
│   │   └── action.yml             # Reusable Node.js test action
│   └── docker-build/
│       └── action.yml             # Reusable Docker build action
└── config/
    └── services.json              # Service configuration metadata
```

### Change Detection Strategy

Use `dorny/paths-filter` or custom script to detect changes in:
- `services/{service-name}/**` - Service code changes
- `libs/{library-name}/**` - Library code changes
- `go.work` - Workspace changes (trigger all Go builds)
- `.github/**` - CI/CD changes (trigger all builds)

### Service Dependency Graph

Libraries are dependencies of services. When a library changes:
1. Test the library itself
2. Build (but don't test) all dependent services

Service to library dependencies will be auto-detected via `go.mod` imports.

### Workflow Design

#### PR Validation Workflow (`pr-validation.yml`)

```yaml
Triggers: pull_request to main
Jobs:
  1. detect-changes
     - Identify modified services/libraries
     - Output matrix of affected components

  2. test-go-libs (if libs changed)
     - Matrix build for each changed library
     - Run: go build, go test -race, coverage check

  3. test-go-services (if services changed)
     - Matrix build for each changed service
     - Run: go build, go test

  4. test-ui (if atlas-ui changed)
     - Run: npm install, npm test, npm run build

  5. build-docker-images (for all changed services)
     - Matrix build for each changed service
     - Build AMD64 only (no push) for validation
```

#### Main Publish Workflow (`main-publish.yml`)

```yaml
Triggers: push to main
Jobs:
  1. detect-changes
     - Identify modified services/libraries
     - Output matrix of affected components

  2. build-and-publish (for each changed service)
     - Matrix build with parallel AMD64/ARM64
     - Push to GHCR with :latest tag
     - Create and push multi-arch manifest
```

---

## Implementation Phases

### Phase 1: Foundation
Set up the core infrastructure for the new CI/CD system.

**Tasks**:
1. Create `.github/actions/detect-changes/action.yml` - Change detection composite action
2. Create service configuration file listing all services and their types
3. Test change detection with a draft PR

### Phase 2: Reusable Actions
Create reusable composite actions for common operations.

**Tasks**:
1. Create `.github/actions/go-test/action.yml` - Go test composite action
2. Create `.github/actions/node-test/action.yml` - Node.js test composite action
3. Create `.github/actions/docker-build/action.yml` - Docker build composite action

### Phase 3: PR Validation Workflow
Implement the pull request validation workflow.

**Tasks**:
1. Create `.github/workflows/pr-validation.yml`
2. Implement Go library testing with coverage
3. Implement Go service testing
4. Implement UI testing
5. Implement Docker image build validation
6. Test with various change scenarios

### Phase 4: Main Branch Publishing
Implement the main branch publishing workflow.

**Tasks**:
1. Create `.github/workflows/main-publish.yml`
2. Implement multi-arch Docker builds
3. Implement GHCR authentication and publishing
4. Implement manifest creation
5. Test with merge scenarios

### Phase 5: Optimization & Cleanup
Optimize and clean up the implementation.

**Tasks**:
1. Add Go module caching
2. Add Docker layer caching
3. Add workflow concurrency controls
4. Update repository documentation
5. Remove old workflow files from service directories
6. Add workflow status badges to README

---

## Detailed Tasks

### Phase 1: Foundation

#### Task 1.1: Create Change Detection Action
**Effort**: M
**Dependencies**: None

Create a composite action that:
- Uses `git diff` to detect changed files
- Maps file paths to service/library names
- Outputs JSON matrix of affected components
- Handles special cases (go.work, .github changes)

**Acceptance Criteria**:
- [ ] Correctly identifies changed services from file paths
- [ ] Correctly identifies changed libraries from file paths
- [ ] Detects when all services need rebuilding (go.work changes)
- [ ] Outputs valid JSON for GitHub Actions matrix
- [ ] Handles edge case of no changes

#### Task 1.2: Create Service Configuration
**Effort**: S
**Dependencies**: None

Create a JSON configuration file mapping services to their:
- Type (go-service, go-library, node-service)
- Path in repository
- Docker image name
- Go module path (for Go services)

**Acceptance Criteria**:
- [ ] Lists all 44 services with correct metadata
- [ ] Lists all 6 libraries with correct metadata
- [ ] Valid JSON schema
- [ ] Documented format

#### Task 1.3: Test Change Detection
**Effort**: S
**Dependencies**: 1.1, 1.2

Create a test workflow that:
- Runs on PR
- Outputs detected changes
- Validates change detection logic

**Acceptance Criteria**:
- [ ] Test workflow runs successfully
- [ ] Changes are correctly detected
- [ ] Output is readable and debuggable

---

### Phase 2: Reusable Actions

#### Task 2.1: Create Go Test Action
**Effort**: M
**Dependencies**: None

Create a composite action that:
- Sets up Go environment (version 1.25.5)
- Caches Go modules
- Runs go build
- Runs go test with coverage
- Supports configurable test flags

**Inputs**:
- `working-directory`: Path to Go module
- `coverage-threshold`: Minimum coverage % (optional)
- `race-detection`: Enable race detection (optional)

**Acceptance Criteria**:
- [ ] Successfully builds Go code
- [ ] Runs tests and reports results
- [ ] Respects coverage threshold
- [ ] Caches modules between runs
- [ ] Supports race detection flag

#### Task 2.2: Create Node.js Test Action
**Effort**: S
**Dependencies**: None

Create a composite action that:
- Sets up Node.js environment (version 22)
- Caches npm dependencies
- Runs npm install
- Runs npm test
- Runs npm run build

**Inputs**:
- `working-directory`: Path to Node.js project

**Acceptance Criteria**:
- [ ] Successfully installs dependencies
- [ ] Runs tests and reports results
- [ ] Builds successfully
- [ ] Caches dependencies between runs

#### Task 2.3: Create Docker Build Action
**Effort**: M
**Dependencies**: None

Create a composite action that:
- Sets up Docker Buildx
- Builds Docker image
- Optionally pushes to registry
- Supports multi-architecture builds

**Inputs**:
- `context`: Docker build context path
- `image-name`: Full image name with registry
- `push`: Whether to push (default: false)
- `platform`: Target platform(s)
- `tags`: Image tags

**Acceptance Criteria**:
- [ ] Builds Docker images successfully
- [ ] Supports AMD64 and ARM64 platforms
- [ ] Pushes when requested
- [ ] Uses Buildx for multi-arch support
- [ ] Proper error handling

---

### Phase 3: PR Validation Workflow

#### Task 3.1: Create PR Validation Workflow Structure
**Effort**: M
**Dependencies**: 1.1, 1.2

Create the main workflow file with:
- PR trigger configuration
- Change detection job
- Matrix strategy for parallel builds
- Job dependencies

**Acceptance Criteria**:
- [ ] Triggers on PR to main
- [ ] Runs change detection first
- [ ] Sets up matrix for parallel testing
- [ ] Proper job dependencies

#### Task 3.2: Implement Go Library Testing
**Effort**: M
**Dependencies**: 2.1, 3.1

Add job for testing Go libraries:
- Matrix over changed libraries
- Use Go test action
- Include coverage and race detection
- 75% coverage threshold (matching atlas-model)

**Acceptance Criteria**:
- [ ] Tests all changed libraries
- [ ] Enforces coverage threshold
- [ ] Includes race detection
- [ ] Reports test results

#### Task 3.3: Implement Go Service Testing
**Effort**: M
**Dependencies**: 2.1, 3.1

Add job for testing Go services:
- Matrix over changed services
- Use Go test action
- Handle nested working directories

**Acceptance Criteria**:
- [ ] Tests all changed services
- [ ] Handles atlas.com/* nested paths
- [ ] Reports test results

#### Task 3.4: Implement UI Testing
**Effort**: S
**Dependencies**: 2.2, 3.1

Add job for testing UI:
- Conditional on atlas-ui changes
- Use Node.js test action
- Run build verification

**Acceptance Criteria**:
- [ ] Only runs when atlas-ui changes
- [ ] Runs all tests
- [ ] Verifies build succeeds

#### Task 3.5: Implement Docker Build Validation
**Effort**: M
**Dependencies**: 2.3, 3.2, 3.3, 3.4

Add job for validating Docker builds:
- Matrix over all changed services
- Build without push
- Validate Dockerfile integrity

**Acceptance Criteria**:
- [ ] Builds Docker images for changed services
- [ ] Does not push to registry
- [ ] Reports build failures clearly

---

### Phase 4: Main Branch Publishing

#### Task 4.1: Create Main Publish Workflow Structure
**Effort**: M
**Dependencies**: 1.1, 1.2

Create the main workflow file with:
- Push to main trigger
- Change detection job
- Matrix strategy for builds
- GHCR authentication

**Acceptance Criteria**:
- [ ] Triggers on push to main
- [ ] Runs change detection
- [ ] Authenticates with GHCR
- [ ] Sets up parallel build matrix

#### Task 4.2: Implement Multi-Arch Docker Builds
**Effort**: L
**Dependencies**: 2.3, 4.1

Add jobs for multi-architecture builds:
- Parallel AMD64 and ARM64 builds
- Push individual architecture images
- Create and push manifest

**Acceptance Criteria**:
- [ ] Builds for both architectures
- [ ] Pushes to GHCR with arch-specific tags
- [ ] Creates multi-arch manifest
- [ ] Pushes manifest with :latest tag
- [ ] Maintains existing image naming convention

#### Task 4.3: Test Publishing Pipeline
**Effort**: M
**Dependencies**: 4.2

Test the complete publishing pipeline:
- Verify image availability
- Verify manifest correctness
- Test rollback scenarios

**Acceptance Criteria**:
- [ ] Images accessible from GHCR
- [ ] Multi-arch manifest works correctly
- [ ] Both architectures pullable

---

### Phase 5: Optimization & Cleanup

#### Task 5.1: Implement Go Module Caching
**Effort**: S
**Dependencies**: Phase 3 complete

Add caching for Go modules:
- Cache go mod download results
- Use hash of go.sum files as cache key
- Restore cache at start of Go jobs

**Acceptance Criteria**:
- [ ] Cache hits reduce build time
- [ ] Cache invalidates on dependency changes
- [ ] Works across matrix jobs

#### Task 5.2: Implement Docker Layer Caching
**Effort**: M
**Dependencies**: Phase 4 complete

Add Docker layer caching:
- Use GitHub Actions cache
- Cache Buildx layers
- Share cache across builds

**Acceptance Criteria**:
- [ ] Cache hits reduce build time
- [ ] Cache works across PR builds
- [ ] Cache works for both architectures

#### Task 5.3: Add Workflow Concurrency Controls
**Effort**: S
**Dependencies**: Phase 3, 4 complete

Add concurrency controls:
- Cancel redundant PR builds
- Prevent parallel publishes for same service
- Queue main builds appropriately

**Acceptance Criteria**:
- [ ] Superseded PR builds are cancelled
- [ ] No race conditions in publishing
- [ ] Appropriate concurrency grouping

#### Task 5.4: Update Documentation
**Effort**: S
**Dependencies**: All phases complete

Update repository documentation:
- Add CI/CD documentation to README
- Document workflow configuration
- Add troubleshooting guide

**Acceptance Criteria**:
- [ ] README explains CI/CD setup
- [ ] Service configuration format documented
- [ ] Common issues and solutions documented

#### Task 5.5: Remove Old Workflow Files
**Effort**: M
**Dependencies**: All phases complete, verified working

Remove deprecated workflow files:
- Delete `.github/workflows/` from all services
- Delete `.github/workflows/` from all libraries
- Verify no orphaned configurations

**Acceptance Criteria**:
- [ ] All old workflow files removed
- [ ] Repository cleaner
- [ ] No duplicate CI runs

#### Task 5.6: Add Status Badges
**Effort**: S
**Dependencies**: Phase 3, 4 complete

Add workflow status badges:
- PR validation status badge
- Main publish status badge
- Add to repository README

**Acceptance Criteria**:
- [ ] Badges display current status
- [ ] Link to workflow runs
- [ ] Visible in README

---

## Risk Assessment and Mitigation

### Risk 1: Change Detection Accuracy
**Risk**: False negatives (missing changes) or false positives (unnecessary builds)
**Probability**: Medium
**Impact**: High (security/stability if false negative, cost/time if false positive)
**Mitigation**:
- Comprehensive testing of change detection
- Include escape hatch to force full builds
- Conservative approach: when in doubt, build

### Risk 2: Docker Build Failures
**Risk**: Existing Dockerfiles may not build correctly with new workflow
**Probability**: Low
**Impact**: Medium
**Mitigation**:
- Test all Dockerfiles before migration
- Maintain same build context and arguments
- Gradual rollout with fallback

### Risk 3: GHCR Authentication Issues
**Risk**: Token permissions or authentication changes break publishing
**Probability**: Low
**Impact**: High (blocking deployments)
**Mitigation**:
- Use repository secrets (already configured)
- Test authentication before full migration
- Document authentication requirements

### Risk 4: Matrix Job Limits
**Risk**: GitHub Actions matrix limits (256 jobs) may be hit
**Probability**: Low (50 services < 256)
**Impact**: Medium
**Mitigation**:
- Current service count well below limit
- Can batch services if needed
- Monitor job counts

### Risk 5: Concurrent PR Builds
**Risk**: Resource contention or race conditions with many PRs
**Probability**: Medium
**Impact**: Low
**Mitigation**:
- Implement concurrency controls
- Cancel superseded builds
- Use appropriate concurrency groups

---

## Success Metrics

### Efficiency Metrics
- **Baseline**: All 50 workflows run on every PR (~50 jobs)
- **Target**: Only affected services test/build (average 2-5 jobs per PR)
- **Measurement**: Compare job counts before/after migration

### Time Metrics
- **Baseline PR Time**: Measure current average PR validation time
- **Target**: 50% reduction for typical single-service PRs
- **Measurement**: GitHub Actions workflow timing

### Maintenance Metrics
- **Baseline**: 100 workflow files to maintain
- **Target**: 6 workflow/action files total
- **Measurement**: File count

### Reliability Metrics
- **Target**: 99% workflow success rate (excluding legitimate test failures)
- **Measurement**: GitHub Actions success/failure rate

---

## Required Resources and Dependencies

### GitHub Actions Requirements
- **Runner Access**: ubuntu-latest, ubuntu-24.04-arm
- **Secrets Required**: GHCR_TOKEN (existing)
- **Features**: Matrix builds, composite actions, caching

### External Dependencies
- `actions/checkout@v4`
- `actions/setup-go@v5`
- `actions/setup-node@v4`
- `actions/cache@v4`
- `docker/setup-buildx-action@v3`
- `docker/login-action@v3`
- `dorny/paths-filter@v3` (for change detection)

### Repository Requirements
- Root `.github/` directory for workflows
- Go workspace file (existing: `go.work`)
- Service directory structure (existing)

---

## Appendix: Service List

### Go Services (43)
```
atlas-account, atlas-buddies, atlas-buffs, atlas-cashshop, atlas-chairs,
atlas-chalkboards, atlas-channel, atlas-character, atlas-character-factory,
atlas-compartment-transfer, atlas-configurations, atlas-consumables, atlas-data,
atlas-drop-information, atlas-drops, atlas-equipables, atlas-expressions,
atlas-fame, atlas-families, atlas-guilds, atlas-inventory, atlas-invites,
atlas-keys, atlas-login, atlas-maps, atlas-marriages, atlas-messages,
atlas-messengers, atlas-monster-death, atlas-monsters, atlas-notes,
atlas-npc-conversations, atlas-npc-shops, atlas-parties, atlas-pets,
atlas-portals, atlas-query-aggregator, atlas-reactors, atlas-saga-orchestrator,
atlas-skills, atlas-tenants, atlas-transports, atlas-world
```

### Go Libraries (6)
```
atlas-constants, atlas-kafka, atlas-model, atlas-rest, atlas-socket, atlas-tenant
```

### Node.js Services (1)
```
atlas-ui
```

### Docker Image Registry
- Registry: `ghcr.io/chronicle20`
- Image pattern: `ghcr.io/chronicle20/{service-name}/{service-name}:latest`
