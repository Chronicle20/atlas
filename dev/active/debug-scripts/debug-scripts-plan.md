# Debug Scripts Strategic Plan

Last Updated: 2025-12-21

## Executive Summary

Create two bash scripts in `tools/` to enable developers to debug individual microservices locally while the rest of the system runs in Kubernetes. The solution modifies the nginx ingress ConfigMap to redirect traffic for specific services to a developer's local machine, while scaling down the in-cluster deployment to avoid conflicts.

**Key Capabilities:**
- Debug one or more services simultaneously
- Restore individual services without affecting other debug sessions
- Idempotent operations (can be run multiple times safely)
- Clean state management via ConfigMap annotations or state tracking

## Current State Analysis

### Infrastructure Overview
- **45 microservices** deployed in the `atlas` namespace
- **Nginx ingress** handles all API routing via ConfigMap `atlas-ingress-configmap`
- **Service discovery** uses K8s DNS: `{service}.atlas.svc.cluster.local`
- **All Go services** listen on port 8080, UI on port 3000
- **Routing pattern**: `location ~ ^/api/{path}(/.*)?$ { proxy_pass http://{service}.atlas.svc.cluster.local:8080; }`

### Nginx ConfigMap Structure
```yaml
# atlas-ingress.yml - ConfigMap section
apiVersion: v1
kind: ConfigMap
metadata:
  name: atlas-ingress-configmap
  namespace: atlas
data:
  nginx.conf: |
    # ... nginx configuration with ~45 location blocks
```

### Service Deployment Pattern
Each service has:
- Kubernetes Deployment (replicas: 1)
- Kubernetes Service (port: 8080)
- ConfigMap reference (atlas-env)
- Secret references (db-credentials)

### Existing Tools
Located in `/tools/`:
- `build-services.sh` - Docker build orchestration
- `db-bootstrap.sh` - PostgreSQL initialization
- `test-all-go.sh` - Go test runner
- `tidy-all-go.sh` - Dependency management

## Proposed Future State

### New Scripts

#### 1. `tools/debug-start.sh`
Enables debugging for a specific service by:
1. Scaling the deployment to 0 replicas
2. Saving the original replica count for restoration
3. Patching the nginx ConfigMap to redirect traffic to developer's IP:port
4. Reloading nginx to apply changes
5. Recording debug state for tracking

#### 2. `tools/debug-stop.sh`
Restores a specific service by:
1. Restoring the nginx ConfigMap to original routing
2. Scaling the deployment back to original replica count
3. Reloading nginx to apply changes
4. Cleaning up debug state tracking

### State Management Approach

**Option A: ConfigMap Annotations (Recommended)**
- Store debug state as annotations on `atlas-ingress-configmap`
- Format: `debug.atlas.io/{service}={original_url}|{original_replicas}`
- Self-contained, no external files needed
- Survives pod restarts

**Option B: Kubernetes ConfigMap for State**
- Create a separate `atlas-debug-state` ConfigMap
- Store service states as data keys
- Clean separation of concerns

**Recommendation:** Option A for simplicity - single artifact to manage

### Service Name to Route Mapping

The scripts need to map service names to nginx location patterns. Since some services handle multiple routes (e.g., `atlas-cashshop` handles 4 different location blocks), the script must:
1. Find all `proxy_pass` directives containing the service name
2. Replace all occurrences when debugging starts
3. Restore all occurrences when debugging stops

### Edge Cases Handled

1. **Multiple debug sessions** - Each service tracked independently
2. **Script re-runs** - Idempotent (no-op if already in desired state)
3. **Service not found** - Clear error message with available services
4. **Nginx reload failure** - Rollback changes
5. **Developer IP unreachable** - Warning only (developer's responsibility)

## Implementation Phases

### Phase 1: Core Script Development (S)

Create the two main scripts with basic functionality.

### Phase 2: State Management (S)

Implement reliable state tracking for debug sessions.

### Phase 3: Multi-Route Handling (S)

Handle services with multiple nginx location blocks.

### Phase 4: Error Handling & UX (S)

Add comprehensive error handling and user feedback.

## Detailed Tasks

### Phase 1: Core Script Development

#### Task 1.1: Create debug-start.sh skeleton
**Effort:** S
**Dependencies:** None
**Acceptance Criteria:**
- Script accepts `--service` and `--target` (ip:port) parameters
- Script validates required parameters
- Script validates service exists in cluster
- Help text available via `-h` or `--help`

#### Task 1.2: Implement deployment scaling
**Effort:** S
**Dependencies:** 1.1
**Acceptance Criteria:**
- Script captures current replica count before scaling
- Script scales deployment to 0
- Script stores original replica count for restoration

#### Task 1.3: Implement nginx ConfigMap patching
**Effort:** M
**Dependencies:** 1.1
**Acceptance Criteria:**
- Script extracts current nginx.conf from ConfigMap
- Script identifies all proxy_pass directives for target service
- Script replaces K8s DNS URLs with developer IP:port
- Script applies patched ConfigMap

#### Task 1.4: Implement nginx reload
**Effort:** S
**Dependencies:** 1.3
**Acceptance Criteria:**
- Script triggers nginx reload in ingress pod
- Script verifies nginx config is valid before reload
- Script handles reload failure gracefully

### Phase 2: State Management

#### Task 2.1: Design state storage structure
**Effort:** S
**Dependencies:** 1.3
**Acceptance Criteria:**
- State format defined and documented
- Storage mechanism selected (ConfigMap annotations)
- State includes: service name, original URL, original replicas, debug target

#### Task 2.2: Implement state save function
**Effort:** S
**Dependencies:** 2.1
**Acceptance Criteria:**
- Function saves debug state to ConfigMap annotation
- State is retrievable for restoration
- Multiple services can have state stored simultaneously

#### Task 2.3: Implement state read function
**Effort:** S
**Dependencies:** 2.1
**Acceptance Criteria:**
- Function reads debug state from ConfigMap annotation
- Returns empty/null if service not in debug mode
- Handles malformed state gracefully

### Phase 3: Create debug-stop.sh

#### Task 3.1: Create debug-stop.sh skeleton
**Effort:** S
**Dependencies:** 2.3
**Acceptance Criteria:**
- Script accepts `--service` parameter
- Script validates service is currently in debug mode
- Help text available

#### Task 3.2: Implement ConfigMap restoration
**Effort:** S
**Dependencies:** 3.1, 2.3
**Acceptance Criteria:**
- Script reads original URL from state
- Script restores all proxy_pass directives for service
- Script applies restored ConfigMap

#### Task 3.3: Implement deployment scaling restoration
**Effort:** S
**Dependencies:** 3.1, 2.3
**Acceptance Criteria:**
- Script reads original replica count from state
- Script scales deployment back to original count
- Script waits for pods to be ready (optional)

#### Task 3.4: Implement state cleanup
**Effort:** S
**Dependencies:** 3.2, 3.3
**Acceptance Criteria:**
- Script removes debug state annotation after successful restore
- Script handles partial failures (logs warning)

### Phase 4: Error Handling & UX

#### Task 4.1: Add service discovery helper
**Effort:** S
**Dependencies:** 1.1
**Acceptance Criteria:**
- `--list` flag shows all available services
- `--status` flag shows currently debugged services
- Output is clear and actionable

#### Task 4.2: Add comprehensive error messages
**Effort:** S
**Dependencies:** All previous
**Acceptance Criteria:**
- All kubectl failures have descriptive messages
- Suggested fixes provided where applicable
- Exit codes are meaningful (0=success, 1=error, 2=usage)

#### Task 4.3: Add idempotency checks
**Effort:** S
**Dependencies:** 2.3
**Acceptance Criteria:**
- Running debug-start twice for same service is no-op (or updates target)
- Running debug-stop for non-debugged service is clear error
- No data corruption from repeated runs

#### Task 4.4: Documentation and help text
**Effort:** S
**Dependencies:** All previous
**Acceptance Criteria:**
- Both scripts have comprehensive `--help` output
- Usage examples included
- Common scenarios documented

## Risk Assessment and Mitigation

### Risk 1: Nginx Configuration Corruption
**Severity:** High
**Probability:** Low
**Mitigation:**
- Validate nginx config before applying (`nginx -t`)
- Keep backup in state annotation
- Rollback mechanism on failure

### Risk 2: Lost Debug State
**Severity:** Medium
**Probability:** Low
**Mitigation:**
- Store state in persistent ConfigMap annotation
- Include recovery helper to scan for scaled-down deployments
- Document manual recovery steps

### Risk 3: Service Name Ambiguity
**Severity:** Low
**Probability:** Low
**Mitigation:**
- Exact match on service name
- List available services on error
- Validate against actual K8s deployments

### Risk 4: Developer Firewall/Network Issues
**Severity:** Medium
**Probability:** Medium
**Mitigation:**
- Document network requirements
- Optional connectivity check before starting
- Clear warning in output

## Success Metrics

1. **Functionality:** Both scripts work for all 44 Go services
2. **Reliability:** Zero data loss from script operations
3. **Usability:** Developer can start debugging in <30 seconds
4. **Maintainability:** Scripts follow existing tools/ conventions

## Required Resources and Dependencies

### Tools Required on Developer Machine
- `kubectl` - configured for cluster access
- `bash` 4.0+ - for associative arrays
- `jq` - for JSON parsing (optional, for cleaner output)
- `sed` - for text manipulation

### Kubernetes Access Requirements
- Permission to read/write ConfigMaps in `atlas` namespace
- Permission to scale Deployments in `atlas` namespace
- Permission to exec into nginx pod (for reload)

### Network Requirements
- Developer machine reachable from K8s cluster
- Port 8080 (or custom) open on developer machine
- Same network segment or VPN/tunnel configured

## Alternative Approaches Considered

### Approach A: Kubernetes Service Patch
Modify the K8s Service to point to an ExternalName or Endpoints object.
**Rejected:** More complex, affects service discovery for other services.

### Approach B: kubectl port-forward
Use port-forward from cluster to developer machine.
**Rejected:** Reverse direction - doesn't route cluster traffic to developer.

### Approach C: Telepresence-style Intercept
Install a proxy pod that redirects traffic.
**Rejected:** External dependency, more complex setup.

### Approach D: Nginx ConfigMap Modification (Selected)
Direct modification of routing rules.
**Selected:** Simple, no external deps, uses existing infrastructure.
