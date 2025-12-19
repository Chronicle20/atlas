# Renovate Monorepo Configuration Plan

**Last Updated: 2025-12-19**

---

## Executive Summary

This plan outlines the consolidation of 50 individual Renovate configuration files into a single, centralized `renovate.json` at the repository root. The new configuration will properly support the monorepo structure with Go services, Go libraries, and a Next.js UI service while maintaining the existing auto-merge behavior and dependency update strategies.

### Goals
1. **Centralized Configuration**: Single `renovate.json` at repository root replaces 50 individual configs
2. **Monorepo-Aware Updates**: Group related updates and handle cross-service dependencies
3. **Consistent Policies**: Uniform auto-merge rules across all services and libraries
4. **Reduced Noise**: Smart grouping to minimize PR volume while maintaining update frequency

### Key Benefits
- Single source of truth for dependency update configuration
- Easier maintenance (1 file vs 50)
- Better visibility into update patterns across the monorepo
- Reduced PR noise through intelligent grouping
- Consistent behavior across all services and libraries

---

## Current State Analysis

### Existing Configuration
- **50 individual `renovate.json` files** distributed across:
  - 43 Go services in `services/atlas-*/renovate.json`
  - 6 Go libraries in `libs/atlas-*/renovate.json`
  - 1 UI service in `services/atlas-ui/renovate.json`

### Current Configuration Patterns

#### Go Services/Libraries Configuration
```json
{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "extends": ["config:recommended"],
  "automerge": true,
  "automergeType": "pr",
  "platformAutomerge": true,
  "requiredStatusChecks": null,
  "allowedPostUpgradeCommands": ["go mod tidy"],
  "enabledManagers": ["gomod"],
  "packageRules": [
    { "managers": ["gomod"], "matchUpdateTypes": ["patch", "minor"], "automerge": true },
    { "managers": ["gomod"], "matchUpdateTypes": ["major"], "automerge": false },
    { "managers": ["gomod"], "matchPackageNames": ["go"], "groupName": "Go version updates", "matchUpdateTypes": ["minor", "patch"], "automerge": true },
    { "managers": ["gomod"], "matchPackageNames": ["go"], "groupName": "Go version updates", "matchUpdateTypes": ["major"], "automerge": false },
    { "managers": ["dockerfile"], "matchDepPatterns": ["golang"], "groupName": "Go version updates in Dockerfiles", "automerge": true }
  ],
  "postUpgradeTasks": {
    "commands": ["go mod tidy"],
    "fileFilters": ["go.mod", "go.sum"]
  }
}
```

#### UI Service Configuration
```json
{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "extends": ["config:recommended"],
  "automerge": true,
  "automergeType": "pr",
  "platformAutomerge": true,
  "requiredStatusChecks": null,
  "packageRules": [
    { "matchUpdateTypes": ["patch", "minor"], "automerge": true },
    { "matchUpdateTypes": ["major"], "automerge": false }
  ]
}
```

### Identified Issues
1. **Duplication**: 50 nearly identical configuration files
2. **No Root Config**: No centralized monorepo configuration
3. **Missing Grouping**: Each service creates separate PRs for the same dependency
4. **Inconsistent Managers**: Go configs enable specific managers, UI uses defaults
5. **No Go Workspace Awareness**: Individual configs don't account for `go.work`
6. **Scattered Updates**: Same dependency updated in 43 services = 43 PRs

### Dependency Summary
| Component Type | Count | Package Manager | Dependency Sources |
|----------------|-------|-----------------|-------------------|
| Go Services | 43 | gomod | External Go packages, Chronicle20 libs |
| Go Libraries | 6 | gomod | External Go packages |
| UI Service | 1 | npm | npm registry |
| Dockerfiles | 44 | dockerfile | golang base images |

---

## Proposed Future State

### New Configuration Architecture

```
atlas/
├── renovate.json              # Root monorepo configuration (NEW)
├── go.work                    # Go workspace (existing)
├── services/
│   ├── atlas-*/              # Go services (remove individual renovate.json)
│   └── atlas-ui/             # UI service (remove individual renovate.json)
└── libs/
    └── atlas-*/              # Go libraries (remove individual renovate.json)
```

### Configuration Strategy

#### 1. Root-Level Configuration
Single `renovate.json` at repository root with:
- Monorepo preset for handling multiple packages
- Path-based matching rules for services vs libraries vs UI
- Grouped updates to reduce PR volume
- Consistent auto-merge policies

#### 2. Smart Grouping Strategy
- **Go Version Updates**: Single PR for all Go version updates across services
- **Dockerfile Base Images**: Single PR for all golang base image updates
- **Internal Libraries**: Group Chronicle20 library updates
- **External Dependencies**: Group by update type (patch/minor together)
- **Major Updates**: Individual PRs for visibility and control

#### 3. Package Rules Hierarchy
```
1. Go version updates (grouped across all services)
2. Dockerfile golang updates (grouped across all services)
3. Chronicle20 internal dependencies (grouped)
4. External Go dependencies (grouped by update type per service)
5. npm dependencies (grouped by update type for atlas-ui)
```

---

## Implementation Phases

### Phase 1: Create Root Configuration
Set up the centralized Renovate configuration.

**Tasks**:
1. Create root `renovate.json` with monorepo presets
2. Configure Go workspace manager settings
3. Configure npm manager for atlas-ui
4. Configure Dockerfile manager for all services
5. Set up path-based package matching

### Phase 2: Configure Grouping Rules
Implement smart dependency grouping.

**Tasks**:
1. Create Go version grouping rule (all services)
2. Create Dockerfile base image grouping rule
3. Create Chronicle20 internal dependency grouping
4. Create external dependency grouping by update type
5. Configure npm dependency grouping for atlas-ui

### Phase 3: Configure Auto-merge Policies
Set up consistent auto-merge behavior.

**Tasks**:
1. Configure patch/minor auto-merge for Go dependencies
2. Configure patch/minor auto-merge for npm dependencies
3. Configure auto-merge for Go version updates (minor/patch)
4. Configure manual merge for major updates
5. Set up required status checks integration

### Phase 4: Post-Upgrade Tasks
Configure post-upgrade commands.

**Tasks**:
1. Configure `go mod tidy` for Go modules
2. Set up file filters for Go workspace
3. Test post-upgrade task execution

### Phase 5: Cleanup and Validation
Remove old configs and validate new setup.

**Tasks**:
1. Test configuration with dry-run
2. Validate PR grouping behavior
3. Remove individual renovate.json files
4. Update repository documentation
5. Monitor initial PR wave

---

## Detailed Tasks

### Phase 1: Create Root Configuration

#### Task 1.1: Create Root renovate.json
**Effort**: M
**Dependencies**: None

Create the root `renovate.json` with base configuration:
- Schema reference
- Monorepo presets
- Base auto-merge settings
- Enabled managers list

**Acceptance Criteria**:
- [ ] File created at repository root
- [ ] Valid JSON with schema reference
- [ ] Extends monorepo-recommended presets
- [ ] Enables gomod, npm, and dockerfile managers

#### Task 1.2: Configure Go Module Manager
**Effort**: S
**Dependencies**: 1.1

Configure gomod manager settings:
- Go version detection from go.work
- Module discovery across services and libs
- Post-upgrade `go mod tidy` command

**Acceptance Criteria**:
- [ ] Detects all 49 go.mod files
- [ ] Uses Go 1.25.5 from go.work
- [ ] Runs go mod tidy after updates

#### Task 1.3: Configure npm Manager
**Effort**: S
**Dependencies**: 1.1

Configure npm manager for atlas-ui:
- Path filtering to services/atlas-ui
- Node version detection
- Lock file maintenance

**Acceptance Criteria**:
- [ ] Detects atlas-ui package.json
- [ ] Respects existing lock file
- [ ] Proper path matching

#### Task 1.4: Configure Dockerfile Manager
**Effort**: S
**Dependencies**: 1.1

Configure dockerfile manager:
- Detect golang base images
- Path filtering for service Dockerfiles
- Version extraction from images

**Acceptance Criteria**:
- [ ] Detects all service Dockerfiles
- [ ] Identifies golang base images
- [ ] Extracts correct versions

---

### Phase 2: Configure Grouping Rules

#### Task 2.1: Go Version Grouping
**Effort**: S
**Dependencies**: Phase 1

Create grouping rule for Go version updates:
- Match `go` package in gomod manager
- Group all services into single PR
- Apply to minor/patch updates

**Acceptance Criteria**:
- [ ] Single PR for Go version across all services
- [ ] Clear group name in PR title
- [ ] Auto-merge enabled for minor/patch

#### Task 2.2: Dockerfile Base Image Grouping
**Effort**: S
**Dependencies**: Phase 1

Create grouping rule for Dockerfile golang images:
- Match `golang` in dockerfile manager
- Group all Dockerfiles into single PR
- Sync with Go version updates where possible

**Acceptance Criteria**:
- [ ] Single PR for golang image updates
- [ ] All 44 Dockerfiles included
- [ ] Auto-merge enabled

#### Task 2.3: Chronicle20 Dependency Grouping
**Effort**: M
**Dependencies**: Phase 1

Create grouping for internal Chronicle20 dependencies:
- Match `github.com/Chronicle20/*` packages
- Group by library name across services
- Separate groups for atlas-kafka, atlas-model, etc.

**Acceptance Criteria**:
- [ ] Internal deps grouped by library
- [ ] Cross-service updates in single PR
- [ ] Clear group names for each library

#### Task 2.4: External Dependency Grouping
**Effort**: M
**Dependencies**: Phase 1

Create grouping for external Go dependencies:
- Separate groups for patch vs minor updates
- Per-service grouping for non-shared deps
- Exclude major updates from grouping

**Acceptance Criteria**:
- [ ] Patch updates grouped per service
- [ ] Minor updates grouped per service
- [ ] Major updates individual PRs

#### Task 2.5: npm Dependency Grouping
**Effort**: S
**Dependencies**: Phase 1

Create grouping for atlas-ui npm dependencies:
- Group production dependencies
- Group dev dependencies separately
- Exclude major updates from grouping

**Acceptance Criteria**:
- [ ] Production deps grouped
- [ ] Dev deps grouped separately
- [ ] Major updates individual

---

### Phase 3: Configure Auto-merge Policies

#### Task 3.1: Go Patch/Minor Auto-merge
**Effort**: S
**Dependencies**: Phase 2

Configure auto-merge for Go dependencies:
- Auto-merge patch and minor updates
- Platform auto-merge enabled
- PR-based auto-merge type

**Acceptance Criteria**:
- [ ] Patch updates auto-merge
- [ ] Minor updates auto-merge
- [ ] Uses platform auto-merge feature

#### Task 3.2: npm Patch/Minor Auto-merge
**Effort**: S
**Dependencies**: Phase 2

Configure auto-merge for npm dependencies:
- Auto-merge patch and minor updates
- Respect package.json semver ranges
- Handle lock file updates

**Acceptance Criteria**:
- [ ] Patch updates auto-merge
- [ ] Minor updates auto-merge
- [ ] Lock file properly updated

#### Task 3.3: Major Update Manual Merge
**Effort**: S
**Dependencies**: Phase 2

Configure manual merge for major updates:
- Disable auto-merge for major updates
- Add labels for visibility
- Require manual review

**Acceptance Criteria**:
- [ ] Major updates not auto-merged
- [ ] Labels applied to major PRs
- [ ] Clear indication of breaking changes

#### Task 3.4: Status Checks Integration
**Effort**: S
**Dependencies**: Phase 2

Configure status checks requirements:
- Integration with pr-validation workflow
- Wait for CI before auto-merge
- Handle check failures appropriately

**Acceptance Criteria**:
- [ ] Waits for CI to pass
- [ ] Respects branch protection rules
- [ ] Auto-merge only after checks pass

---

### Phase 4: Post-Upgrade Tasks

#### Task 4.1: Configure go mod tidy
**Effort**: S
**Dependencies**: Phase 1

Configure post-upgrade task for Go modules:
- Run `go mod tidy` after updates
- Filter for go.mod and go.sum files
- Handle workspace context

**Acceptance Criteria**:
- [ ] go mod tidy runs after updates
- [ ] Works with go.work workspace
- [ ] go.sum properly updated

#### Task 4.2: Test Post-Upgrade Execution
**Effort**: S
**Dependencies**: 4.1

Test post-upgrade task execution:
- Verify command runs correctly
- Check file modifications
- Validate commit includes tidy changes

**Acceptance Criteria**:
- [ ] Commands execute successfully
- [ ] Changes committed properly
- [ ] No manual intervention needed

---

### Phase 5: Cleanup and Validation

#### Task 5.1: Dry-Run Validation
**Effort**: M
**Dependencies**: Phases 1-4

Perform dry-run to validate configuration:
- Use Renovate debug mode
- Check discovered dependencies
- Verify grouping behavior
- Confirm path matching

**Acceptance Criteria**:
- [ ] All go.mod files discovered
- [ ] package.json discovered
- [ ] Dockerfiles discovered
- [ ] Grouping works as expected

#### Task 5.2: Verify PR Grouping
**Effort**: S
**Dependencies**: 5.1

Verify PR grouping behavior:
- Check group assignments
- Verify PR titles and descriptions
- Confirm auto-merge settings

**Acceptance Criteria**:
- [ ] PRs grouped as configured
- [ ] Titles are descriptive
- [ ] Auto-merge applied correctly

#### Task 5.3: Remove Individual Configs
**Effort**: S
**Dependencies**: 5.2

Remove old renovate.json files:
- Delete from all 43 Go services
- Delete from all 6 Go libraries
- Delete from atlas-ui
- Total: 50 files removed

**Acceptance Criteria**:
- [ ] All 50 files removed
- [ ] No orphaned configurations
- [ ] Single renovate.json remains

#### Task 5.4: Update Documentation
**Effort**: S
**Dependencies**: 5.3

Update repository documentation:
- Document Renovate configuration
- Explain grouping strategy
- Add troubleshooting guide

**Acceptance Criteria**:
- [ ] README updated with Renovate info
- [ ] Configuration documented
- [ ] Common issues addressed

#### Task 5.5: Monitor Initial Wave
**Effort**: S
**Dependencies**: All complete

Monitor initial PR wave:
- Track number of PRs created
- Verify auto-merge behavior
- Address any configuration issues

**Acceptance Criteria**:
- [ ] PR count reduced vs old approach
- [ ] Auto-merge working correctly
- [ ] No unexpected behavior

---

## Risk Assessment and Mitigation

### Risk 1: Configuration Migration Disruption
**Risk**: Temporary disruption during migration with duplicate or missed PRs
**Probability**: Medium
**Impact**: Low (cosmetic, no security impact)
**Mitigation**:
- Perform migration during low-activity period
- Close existing Renovate PRs before migration
- Enable new config before removing old

### Risk 2: Over-Grouping Dependencies
**Risk**: Grouping unrelated updates may obscure breaking changes
**Probability**: Low
**Impact**: Medium
**Mitigation**:
- Keep major updates ungrouped
- Maintain per-service grouping for external deps
- Clear PR descriptions with change details

### Risk 3: go.work Compatibility
**Risk**: Renovate may not fully support go.work workspace files
**Probability**: Low
**Impact**: Medium
**Mitigation**:
- Test go.work handling in dry-run
- Fall back to individual go.mod handling if needed
- Monitor Renovate updates for workspace support

### Risk 4: Post-Upgrade Task Failures
**Risk**: `go mod tidy` may fail in certain scenarios
**Probability**: Low
**Impact**: Low (PR still created, needs manual tidy)
**Mitigation**:
- Test with various dependency combinations
- CI will catch untidy go.mod files
- Can disable post-upgrade if problematic

### Risk 5: Auto-merge with Failing CI
**Risk**: Dependencies auto-merged despite breaking tests
**Probability**: Low
**Impact**: High
**Mitigation**:
- Configure required status checks
- Integrate with existing pr-validation workflow
- Enable branch protection rules

---

## Success Metrics

### Configuration Metrics
- **Baseline**: 50 renovate.json files
- **Target**: 1 renovate.json file
- **Measurement**: File count in repository

### PR Volume Metrics
- **Baseline**: Up to 50 PRs for shared dependency update
- **Target**: 1-5 PRs for same update (grouped by service type)
- **Measurement**: PR count per dependency update

### Maintenance Metrics
- **Baseline**: Updates require editing 50 files
- **Target**: Updates require editing 1 file
- **Measurement**: Files changed for config update

### Auto-merge Success Rate
- **Target**: >95% auto-merge success for patch/minor
- **Measurement**: Renovate dashboard metrics

---

## Required Resources and Dependencies

### Renovate Requirements
- **Renovate App**: GitHub App installed on repository
- **Permissions**: Write access for PRs and commits
- **Platform**: GitHub (supports platform auto-merge)

### Configuration Dependencies
- `config:recommended` preset (Renovate built-in)
- `monorepo:default` preset (Renovate built-in)
- Go workspace support (Renovate gomod manager)
- npm support (Renovate npm manager)
- Dockerfile support (Renovate dockerfile manager)

### Repository Requirements
- GitHub Actions for CI (existing pr-validation.yml)
- Branch protection rules (recommended)
- CODEOWNERS file (optional)

---

## Final Configuration Preview

```json
{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "extends": [
    "config:recommended",
    ":separateMajorReleases",
    ":combinePatchMinorReleases"
  ],
  "automerge": true,
  "automergeType": "pr",
  "platformAutomerge": true,
  "enabledManagers": ["gomod", "npm", "dockerfile"],
  "ignorePaths": [],
  "postUpgradeTasks": {
    "commands": ["go mod tidy"],
    "fileFilters": ["**/go.mod", "**/go.sum"],
    "executionMode": "branch"
  },
  "packageRules": [
    {
      "description": "Group Go version updates across all services",
      "matchManagers": ["gomod"],
      "matchPackageNames": ["go"],
      "groupName": "go-version",
      "automerge": true,
      "matchUpdateTypes": ["minor", "patch"]
    },
    {
      "description": "Manual merge for major Go version updates",
      "matchManagers": ["gomod"],
      "matchPackageNames": ["go"],
      "matchUpdateTypes": ["major"],
      "automerge": false
    },
    {
      "description": "Group Dockerfile golang base image updates",
      "matchManagers": ["dockerfile"],
      "matchPackageNames": ["golang"],
      "groupName": "dockerfile-golang",
      "automerge": true
    },
    {
      "description": "Group Chronicle20 library updates",
      "matchManagers": ["gomod"],
      "matchPackagePatterns": ["^github.com/Chronicle20/"],
      "groupName": "chronicle20-libs",
      "automerge": true,
      "matchUpdateTypes": ["minor", "patch"]
    },
    {
      "description": "Auto-merge patch/minor Go dependencies",
      "matchManagers": ["gomod"],
      "matchUpdateTypes": ["patch", "minor"],
      "automerge": true
    },
    {
      "description": "Manual merge for major Go dependency updates",
      "matchManagers": ["gomod"],
      "matchUpdateTypes": ["major"],
      "automerge": false
    },
    {
      "description": "Group atlas-ui production dependencies",
      "matchManagers": ["npm"],
      "matchPaths": ["services/atlas-ui/**"],
      "matchDepTypes": ["dependencies"],
      "groupName": "atlas-ui-prod-deps",
      "automerge": true,
      "matchUpdateTypes": ["patch", "minor"]
    },
    {
      "description": "Group atlas-ui dev dependencies",
      "matchManagers": ["npm"],
      "matchPaths": ["services/atlas-ui/**"],
      "matchDepTypes": ["devDependencies"],
      "groupName": "atlas-ui-dev-deps",
      "automerge": true,
      "matchUpdateTypes": ["patch", "minor"]
    },
    {
      "description": "Manual merge for major npm updates",
      "matchManagers": ["npm"],
      "matchUpdateTypes": ["major"],
      "automerge": false
    }
  ]
}
```

---

## Appendix: Files to Remove

### Go Services (43 files)
```
services/atlas-account/renovate.json
services/atlas-buddies/renovate.json
services/atlas-buffs/renovate.json
services/atlas-cashshop/renovate.json
services/atlas-chairs/renovate.json
services/atlas-chalkboards/renovate.json
services/atlas-channel/renovate.json
services/atlas-character/renovate.json
services/atlas-character-factory/renovate.json
services/atlas-compartment-transfer/renovate.json
services/atlas-configurations/renovate.json
services/atlas-consumables/renovate.json
services/atlas-data/renovate.json
services/atlas-drop-information/renovate.json
services/atlas-drops/renovate.json
services/atlas-equipables/renovate.json
services/atlas-expressions/renovate.json
services/atlas-fame/renovate.json
services/atlas-families/renovate.json
services/atlas-guilds/renovate.json
services/atlas-inventory/renovate.json
services/atlas-invites/renovate.json
services/atlas-keys/renovate.json
services/atlas-login/renovate.json
services/atlas-maps/renovate.json
services/atlas-marriages/renovate.json
services/atlas-messages/renovate.json
services/atlas-messengers/renovate.json
services/atlas-monster-death/renovate.json
services/atlas-monsters/renovate.json
services/atlas-notes/renovate.json
services/atlas-npc-conversations/renovate.json
services/atlas-npc-shops/renovate.json
services/atlas-parties/renovate.json
services/atlas-pets/renovate.json
services/atlas-portals/renovate.json
services/atlas-query-aggregator/renovate.json
services/atlas-reactors/renovate.json
services/atlas-saga-orchestrator/renovate.json
services/atlas-skills/renovate.json
services/atlas-tenants/renovate.json
services/atlas-transports/renovate.json
services/atlas-world/renovate.json
```

### Go Libraries (6 files)
```
libs/atlas-constants/renovate.json
libs/atlas-kafka/renovate.json
libs/atlas-model/renovate.json
libs/atlas-rest/renovate.json
libs/atlas-socket/renovate.json
libs/atlas-tenant/renovate.json
```

### UI Service (1 file)
```
services/atlas-ui/renovate.json
```

**Total: 50 files to remove**
