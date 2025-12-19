# Monorepo CI/CD - Task Checklist

**Last Updated: 2025-12-19**

---

## Phase 1: Foundation

### Task 1.1: Create Change Detection Action
**Effort**: M | **Status**: [x] Complete

- [ ] Create `.github/actions/detect-changes/action.yml`
- [ ] Implement `dorny/paths-filter` integration
- [ ] Define path patterns for all services
- [ ] Define path patterns for all libraries
- [ ] Output JSON matrix format
- [ ] Handle go.work changes (trigger all Go)
- [ ] Handle .github changes (trigger all)
- [ ] Test edge case: no changes
- [ ] Test edge case: only documentation changes

**Files to create**:
- `/.github/actions/detect-changes/action.yml`

---

### Task 1.2: Create Service Configuration
**Effort**: S | **Status**: [x] Complete

- [ ] Create `services.json` with all 44 services
- [ ] Include service type (go-service, go-library, node-service)
- [ ] Include repository path
- [ ] Include Docker image name
- [ ] Include Go module path (for Go services)
- [ ] Document JSON schema

**Files to create**:
- `/.github/config/services.json`

---

### Task 1.3: Test Change Detection
**Effort**: S | **Status**: [ ] Not Started
**Dependencies**: 1.1, 1.2

- [ ] Create test workflow file
- [ ] Test single service change detection
- [ ] Test library change detection
- [ ] Test multiple service changes
- [ ] Test go.work change detection
- [ ] Verify JSON output format

**Files to create**:
- `/.github/workflows/test-change-detection.yml` (temporary)

---

## Phase 2: Reusable Actions

### Task 2.1: Create Go Test Action
**Effort**: M | **Status**: [x] Complete

- [ ] Create composite action file
- [ ] Implement Go version setup (1.25.5)
- [ ] Implement module caching
- [ ] Implement go build step
- [ ] Implement go test step
- [ ] Add coverage flag support
- [ ] Add race detection flag support
- [ ] Add coverage threshold checking
- [ ] Add configurable working directory
- [ ] Document inputs and outputs

**Files to create**:
- `/.github/actions/go-test/action.yml`

---

### Task 2.2: Create Node.js Test Action
**Effort**: S | **Status**: [x] Complete

- [ ] Create composite action file
- [ ] Implement Node.js version setup (22)
- [ ] Implement npm cache
- [ ] Implement npm install step
- [ ] Implement npm test step
- [ ] Implement npm build step
- [ ] Add configurable working directory
- [ ] Document inputs and outputs

**Files to create**:
- `/.github/actions/node-test/action.yml`

---

### Task 2.3: Create Docker Build Action
**Effort**: M | **Status**: [x] Complete

- [ ] Create composite action file
- [ ] Implement Docker Buildx setup
- [ ] Implement registry login
- [ ] Implement build step
- [ ] Implement optional push step
- [ ] Add platform selection input
- [ ] Add tag configuration
- [ ] Add build args support
- [ ] Document inputs and outputs

**Files to create**:
- `/.github/actions/docker-build/action.yml`

---

## Phase 3: PR Validation Workflow

### Task 3.1: Create PR Validation Workflow Structure
**Effort**: M | **Status**: [x] Complete
**Dependencies**: 1.1, 1.2

- [ ] Create workflow file with PR trigger
- [ ] Add change detection job
- [ ] Configure matrix strategy from detection output
- [ ] Set up job dependencies
- [ ] Configure concurrency controls
- [ ] Add workflow_dispatch for manual triggers

**Files to create**:
- `/.github/workflows/pr-validation.yml`

---

### Task 3.2: Implement Go Library Testing
**Effort**: M | **Status**: [x] Complete
**Dependencies**: 2.1, 3.1

- [ ] Add test-libs job to workflow
- [ ] Configure matrix for changed libraries
- [ ] Use go-test action
- [ ] Enable race detection
- [ ] Enable coverage checking (75% threshold)
- [ ] Test with atlas-model library
- [ ] Test with atlas-kafka library
- [ ] Verify coverage output

---

### Task 3.3: Implement Go Service Testing
**Effort**: M | **Status**: [x] Complete
**Dependencies**: 2.1, 3.1

- [ ] Add test-services job to workflow
- [ ] Configure matrix for changed services
- [ ] Use go-test action
- [ ] Handle nested working directories
- [ ] Test with atlas-account service
- [ ] Test with multiple services
- [ ] Verify test output

---

### Task 3.4: Implement UI Testing
**Effort**: S | **Status**: [x] Complete
**Dependencies**: 2.2, 3.1

- [ ] Add test-ui job to workflow
- [ ] Configure conditional execution
- [ ] Use node-test action
- [ ] Test with atlas-ui service
- [ ] Verify build output

---

### Task 3.5: Implement Docker Build Validation
**Effort**: M | **Status**: [x] Complete
**Dependencies**: 2.3, 3.2, 3.3, 3.4

- [ ] Add build-docker job to workflow
- [ ] Configure matrix for changed services
- [ ] Use docker-build action (no push)
- [ ] Build all service types
- [ ] Test with Go service
- [ ] Test with UI service
- [ ] Verify no images pushed

---

## Phase 4: Main Branch Publishing

### Task 4.1: Create Main Publish Workflow Structure
**Effort**: M | **Status**: [x] Complete
**Dependencies**: 1.1, 1.2

- [ ] Create workflow file with push trigger
- [ ] Add change detection job
- [ ] Configure GHCR authentication
- [ ] Set up matrix strategy
- [ ] Add concurrency controls

**Files to create**:
- `/.github/workflows/main-publish.yml`

---

### Task 4.2: Implement Multi-Arch Docker Builds
**Effort**: L | **Status**: [x] Complete
**Dependencies**: 2.3, 4.1

- [ ] Add AMD64 build job
- [ ] Add ARM64 build job
- [ ] Configure parallel execution
- [ ] Implement architecture-specific tags
- [ ] Add manifest creation job
- [ ] Configure manifest dependencies
- [ ] Push multi-arch manifest
- [ ] Test with single service
- [ ] Verify image accessibility

---

### Task 4.3: Test Publishing Pipeline
**Effort**: M | **Status**: [ ] Not Started
**Dependencies**: 4.2

- [ ] Merge test PR to main
- [ ] Verify image published to GHCR
- [ ] Verify AMD64 image
- [ ] Verify ARM64 image
- [ ] Verify manifest
- [ ] Pull and test image
- [ ] Document verification steps

---

## Phase 5: Optimization & Cleanup

### Task 5.1: Implement Go Module Caching
**Effort**: S | **Status**: [ ] Not Started
**Dependencies**: Phase 3 complete

- [ ] Add cache step to go-test action
- [ ] Configure cache key with go.sum hash
- [ ] Add cache restore step
- [ ] Test cache hit scenario
- [ ] Test cache miss scenario
- [ ] Measure time improvement

---

### Task 5.2: Implement Docker Layer Caching
**Effort**: M | **Status**: [ ] Not Started
**Dependencies**: Phase 4 complete

- [ ] Add cache configuration to docker-build action
- [ ] Configure cache backend (gha)
- [ ] Set up cache scope
- [ ] Test cache effectiveness
- [ ] Measure time improvement

---

### Task 5.3: Add Workflow Concurrency Controls
**Effort**: S | **Status**: [ ] Not Started
**Dependencies**: Phase 3, 4 complete

- [ ] Add concurrency to PR workflow
- [ ] Add concurrency to main workflow
- [ ] Configure cancel-in-progress for PRs
- [ ] Test concurrent PR behavior
- [ ] Verify no race conditions

---

### Task 5.4: Update Documentation
**Effort**: S | **Status**: [ ] Not Started
**Dependencies**: All phases complete

- [ ] Add CI/CD section to README
- [ ] Document workflow triggers
- [ ] Document service configuration format
- [ ] Add troubleshooting section
- [ ] Document manual trigger usage

---

### Task 5.5: Remove Old Workflow Files
**Effort**: M | **Status**: [ ] Not Started
**Dependencies**: All phases complete, verified working

- [ ] Verify new workflows are working
- [ ] List all old workflow files
- [ ] Create removal script
- [ ] Remove service workflow files (44 services x 2)
- [ ] Remove library workflow files (6 libraries x 2)
- [ ] Verify no orphaned references
- [ ] Commit removal as separate PR

**Files to delete**:
- All `services/*/.github/workflows/*.yml`
- All `libs/*/.github/workflows/*.yml`

---

### Task 5.6: Add Status Badges
**Effort**: S | **Status**: [ ] Not Started
**Dependencies**: Phase 3, 4 complete

- [ ] Generate PR validation badge
- [ ] Generate main publish badge
- [ ] Add badges to README.md
- [ ] Verify badge display
- [ ] Test badge links

---

## Summary

| Phase | Tasks | Effort |
|-------|-------|--------|
| Phase 1: Foundation | 3 | S + S + M = M |
| Phase 2: Reusable Actions | 3 | M + S + M = L |
| Phase 3: PR Validation | 5 | M + M + M + S + M = XL |
| Phase 4: Main Publishing | 3 | M + L + M = XL |
| Phase 5: Optimization | 6 | S + M + S + S + M + S = L |
| **Total** | **20** | **XL** |

---

## Progress Tracking

```
Phase 1: [x] [x] [ ]           2/3 complete (Task 1.3 test deferred)
Phase 2: [x] [x] [x]           3/3 complete
Phase 3: [x] [x] [x] [x] [x]   5/5 complete
Phase 4: [x] [x] [ ]           2/3 complete (Task 4.3 test deferred)
Phase 5: [ ] [ ] [ ] [ ] [ ] [ ] 0/6 complete (optimization phase)

Overall: 12/20 core tasks complete (60%)
Note: Phase 5 optimization tasks are deferred until core workflows are validated in production.
```
