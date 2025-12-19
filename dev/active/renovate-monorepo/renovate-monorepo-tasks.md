# Renovate Monorepo Configuration - Task Checklist

**Last Updated: 2025-12-19**

---

## Overview
- **Total Tasks**: 19
- **Phases**: 5
- **Effort Estimate**: M (Medium - 1-2 hours)

---

## Phase 1: Create Root Configuration

### Task 1.1: Create Root renovate.json
**Effort**: M | **Status**: [ ] Pending

Create the root `renovate.json` with base configuration.

**Steps**:
- [ ] Create `renovate.json` at repository root
- [ ] Add JSON schema reference
- [ ] Add monorepo presets (config:recommended)
- [ ] Enable managers: gomod, npm, dockerfile
- [ ] Set base automerge settings

**Acceptance Criteria**:
- [ ] File created at repository root
- [ ] Valid JSON with schema reference
- [ ] Extends recommended presets
- [ ] Enables required managers

---

### Task 1.2: Configure Go Module Manager
**Effort**: S | **Status**: [ ] Pending
**Depends on**: 1.1

Configure gomod manager settings.

**Steps**:
- [ ] Add postUpgradeTasks for go mod tidy
- [ ] Set fileFilters for go.mod and go.sum
- [ ] Test go.work workspace detection

**Acceptance Criteria**:
- [ ] Detects all 49 go.mod files
- [ ] Uses Go 1.25.5 from go.work
- [ ] Runs go mod tidy after updates

---

### Task 1.3: Configure npm Manager
**Effort**: S | **Status**: [ ] Pending
**Depends on**: 1.1

Configure npm manager for atlas-ui.

**Steps**:
- [ ] Verify npm manager enabled
- [ ] Add path matching for services/atlas-ui
- [ ] Configure lock file handling

**Acceptance Criteria**:
- [ ] Detects atlas-ui package.json
- [ ] Respects existing lock file
- [ ] Proper path matching

---

### Task 1.4: Configure Dockerfile Manager
**Effort**: S | **Status**: [ ] Pending
**Depends on**: 1.1

Configure dockerfile manager.

**Steps**:
- [ ] Verify dockerfile manager enabled
- [ ] Add package patterns for golang/node images
- [ ] Test version extraction

**Acceptance Criteria**:
- [ ] Detects all service Dockerfiles
- [ ] Identifies golang base images
- [ ] Extracts correct versions

---

## Phase 2: Configure Grouping Rules

### Task 2.1: Go Version Grouping
**Effort**: S | **Status**: [ ] Pending
**Depends on**: Phase 1

Create grouping rule for Go version updates.

**Steps**:
- [ ] Add packageRule matching go package
- [ ] Set groupName: "go-version"
- [ ] Configure for minor/patch updates
- [ ] Enable automerge

**Acceptance Criteria**:
- [ ] Single PR for Go version across all services
- [ ] Clear group name in PR title
- [ ] Auto-merge enabled for minor/patch

---

### Task 2.2: Dockerfile Base Image Grouping
**Effort**: S | **Status**: [ ] Pending
**Depends on**: Phase 1

Create grouping rule for Dockerfile golang images.

**Steps**:
- [ ] Add packageRule for dockerfile manager
- [ ] Match golang package name
- [ ] Set groupName: "dockerfile-golang"
- [ ] Enable automerge

**Acceptance Criteria**:
- [ ] Single PR for golang image updates
- [ ] All 44 Dockerfiles included
- [ ] Auto-merge enabled

---

### Task 2.3: Chronicle20 Dependency Grouping
**Effort**: M | **Status**: [ ] Pending
**Depends on**: Phase 1

Create grouping for internal Chronicle20 dependencies.

**Steps**:
- [ ] Add packageRule with matchPackagePatterns
- [ ] Pattern: `^github.com/Chronicle20/`
- [ ] Set groupName: "chronicle20-libs"
- [ ] Configure automerge for minor/patch

**Acceptance Criteria**:
- [ ] Internal deps grouped together
- [ ] Cross-service updates in single PR
- [ ] Clear group name

---

### Task 2.4: External Dependency Grouping
**Effort**: M | **Status**: [ ] Pending
**Depends on**: Phase 1

Create grouping for external Go dependencies.

**Steps**:
- [ ] Add packageRule for patch updates
- [ ] Add packageRule for minor updates
- [ ] Exclude major updates from grouping

**Acceptance Criteria**:
- [ ] Patch updates grouped per service
- [ ] Minor updates grouped per service
- [ ] Major updates individual PRs

---

### Task 2.5: npm Dependency Grouping
**Effort**: S | **Status**: [ ] Pending
**Depends on**: Phase 1

Create grouping for atlas-ui npm dependencies.

**Steps**:
- [ ] Add packageRule for production deps
- [ ] Add packageRule for dev deps
- [ ] Set separate group names
- [ ] Exclude major from groups

**Acceptance Criteria**:
- [ ] Production deps grouped
- [ ] Dev deps grouped separately
- [ ] Major updates individual

---

## Phase 3: Configure Auto-merge Policies

### Task 3.1: Go Patch/Minor Auto-merge
**Effort**: S | **Status**: [ ] Pending
**Depends on**: Phase 2

Configure auto-merge for Go dependencies.

**Steps**:
- [ ] Add packageRule matching gomod manager
- [ ] Match patch and minor update types
- [ ] Enable automerge
- [ ] Set automergeType: pr

**Acceptance Criteria**:
- [ ] Patch updates auto-merge
- [ ] Minor updates auto-merge
- [ ] Uses platform auto-merge

---

### Task 3.2: npm Patch/Minor Auto-merge
**Effort**: S | **Status**: [ ] Pending
**Depends on**: Phase 2

Configure auto-merge for npm dependencies.

**Steps**:
- [ ] Add packageRule matching npm manager
- [ ] Match patch and minor update types
- [ ] Enable automerge
- [ ] Set automergeType: pr

**Acceptance Criteria**:
- [ ] Patch updates auto-merge
- [ ] Minor updates auto-merge
- [ ] Lock file properly updated

---

### Task 3.3: Major Update Manual Merge
**Effort**: S | **Status**: [ ] Pending
**Depends on**: Phase 2

Configure manual merge for major updates.

**Steps**:
- [ ] Add packageRule matching major updates
- [ ] Disable automerge
- [ ] Consider adding labels for visibility

**Acceptance Criteria**:
- [ ] Major updates not auto-merged
- [ ] Clear in PR that manual review required
- [ ] Applies to all managers

---

### Task 3.4: Status Checks Integration
**Effort**: S | **Status**: [ ] Pending
**Depends on**: Phase 2

Configure status checks requirements.

**Steps**:
- [ ] Review requiredStatusChecks setting
- [ ] Ensure pr-validation workflow detected
- [ ] Configure platformAutomerge: true

**Acceptance Criteria**:
- [ ] Waits for CI to pass
- [ ] Respects branch protection rules
- [ ] Auto-merge only after checks pass

---

## Phase 4: Post-Upgrade Tasks

### Task 4.1: Configure go mod tidy
**Effort**: S | **Status**: [ ] Pending
**Depends on**: Phase 1

Configure post-upgrade task for Go modules.

**Steps**:
- [ ] Add postUpgradeTasks configuration
- [ ] Set commands: ["go mod tidy"]
- [ ] Set fileFilters: ["**/go.mod", "**/go.sum"]
- [ ] Set executionMode: branch

**Acceptance Criteria**:
- [ ] go mod tidy runs after updates
- [ ] Works with go.work workspace
- [ ] go.sum properly updated

---

### Task 4.2: Test Post-Upgrade Execution
**Effort**: S | **Status**: [ ] Pending
**Depends on**: 4.1

Test post-upgrade task execution.

**Steps**:
- [ ] Wait for first Renovate PR
- [ ] Verify go mod tidy executed
- [ ] Check commit includes tidy changes

**Acceptance Criteria**:
- [ ] Commands execute successfully
- [ ] Changes committed properly
- [ ] No manual intervention needed

---

## Phase 5: Cleanup and Validation

### Task 5.1: Dry-Run Validation
**Effort**: M | **Status**: [ ] Pending
**Depends on**: Phases 1-4

Perform dry-run to validate configuration.

**Steps**:
- [ ] Commit renovate.json to branch
- [ ] Check Renovate logs/dashboard
- [ ] Verify discovered dependencies
- [ ] Confirm path matching works

**Acceptance Criteria**:
- [ ] All go.mod files discovered
- [ ] package.json discovered
- [ ] Dockerfiles discovered
- [ ] Grouping works as expected

---

### Task 5.2: Verify PR Grouping
**Effort**: S | **Status**: [ ] Pending
**Depends on**: 5.1

Verify PR grouping behavior.

**Steps**:
- [ ] Review generated PRs
- [ ] Check group assignments correct
- [ ] Verify PR titles descriptive

**Acceptance Criteria**:
- [ ] PRs grouped as configured
- [ ] Titles are descriptive
- [ ] Auto-merge applied correctly

---

### Task 5.3: Remove Individual Configs
**Effort**: S | **Status**: [ ] Pending
**Depends on**: 5.2

Remove old renovate.json files.

**Steps**:
- [ ] Delete services/atlas-*/renovate.json (43 files)
- [ ] Delete libs/atlas-*/renovate.json (6 files)
- [ ] Delete services/atlas-ui/renovate.json (1 file)
- [ ] Commit removal

**Acceptance Criteria**:
- [ ] All 50 files removed
- [ ] No orphaned configurations
- [ ] Single renovate.json remains

---

### Task 5.4: Update Documentation
**Effort**: S | **Status**: [ ] Pending
**Depends on**: 5.3

Update repository documentation.

**Steps**:
- [ ] Add Renovate section to README
- [ ] Document configuration approach
- [ ] Add any troubleshooting notes

**Acceptance Criteria**:
- [ ] README updated with Renovate info
- [ ] Configuration documented
- [ ] Common issues addressed

---

### Task 5.5: Monitor Initial Wave
**Effort**: S | **Status**: [ ] Pending
**Depends on**: All complete

Monitor initial PR wave.

**Steps**:
- [ ] Track number of PRs created
- [ ] Verify auto-merge working
- [ ] Address any issues found

**Acceptance Criteria**:
- [ ] PR count reduced vs old approach
- [ ] Auto-merge working correctly
- [ ] No unexpected behavior

---

## Progress Summary

| Phase | Tasks | Completed | Progress |
|-------|-------|-----------|----------|
| Phase 1: Create Root Configuration | 4 | 0 | 0% |
| Phase 2: Configure Grouping Rules | 5 | 0 | 0% |
| Phase 3: Configure Auto-merge Policies | 4 | 0 | 0% |
| Phase 4: Post-Upgrade Tasks | 2 | 0 | 0% |
| Phase 5: Cleanup and Validation | 5 | 0 | 0% |
| **Total** | **19** | **0** | **0%** |

---

## Quick Reference: Commands

```bash
# Validate JSON syntax
cat renovate.json | jq .

# Find all existing renovate.json files
find . -name "renovate.json" -type f

# Count renovate.json files
find . -name "renovate.json" -type f | wc -l

# Remove all individual renovate.json files (after validation)
find services libs -name "renovate.json" -type f -delete

# View Renovate logs (if self-hosted)
docker logs renovate-bot
```
