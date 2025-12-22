# Debug Scripts - Task Checklist

Last Updated: 2025-12-21

## Overview
Create `debug-start.sh` and `debug-stop.sh` scripts in `tools/` for local service debugging.

**STATUS: IMPLEMENTED**

---

## Phase 1: Core Script Development

### debug-start.sh Foundation
- [x] **1.1** Create script skeleton with shebang, set options, and configuration
- [x] **1.2** Implement argument parsing (--service, --target, --help, --list, --status)
- [x] **1.3** Add precondition checks (kubectl, namespace access)
- [x] **1.4** Implement service validation (check deployment exists)

### Deployment Scaling
- [x] **1.5** Get current replica count from deployment
- [x] **1.6** Scale deployment to 0 replicas
- [x] **1.7** Wait for scale-down confirmation

### Nginx Modification
- [x] **1.8** Extract nginx.conf from ConfigMap
- [x] **1.9** Find all proxy_pass directives for target service
- [x] **1.10** Replace service URLs with debug target
- [x] **1.11** Validate modified nginx config
- [x] **1.12** Apply patched ConfigMap
- [x] **1.13** Reload nginx in ingress pod

---

## Phase 2: State Management

### State Storage
- [x] **2.1** Implement state save function (add annotation to ConfigMap)
- [x] **2.2** Implement state read function (get annotation from ConfigMap)
- [x] **2.3** Store original replica count in annotation

---

## Phase 3: debug-stop.sh Development

### Script Foundation
- [x] **3.1** Create script skeleton with shebang, set options, and configuration
- [x] **3.2** Implement argument parsing (--service, --help, --all)
- [x] **3.3** Add precondition checks

### Restoration Logic
- [x] **3.4** Read debug state from annotation
- [x] **3.5** Validate service is in debug mode
- [x] **3.6** Extract nginx.conf from ConfigMap
- [x] **3.7** Restore proxy_pass directives to original URLs
- [x] **3.8** Apply restored ConfigMap
- [x] **3.9** Reload nginx in ingress pod
- [x] **3.10** Scale deployment back to original replicas
- [x] **3.11** Remove debug state annotation

---

## Phase 4: Polish & Error Handling

### UX Improvements
- [x] **4.1** Implement --list flag (show available services)
- [x] **4.2** Implement --status flag (show debugged services)
- [x] **4.3** Add colored output for status messages
- [x] **4.4** Add comprehensive --help text with examples

### Error Handling
- [x] **4.5** Add rollback on ConfigMap patch failure
- [x] **4.6** Add rollback on nginx reload failure
- [x] **4.7** Handle partial state (deployment scaled but nginx not updated)
- [x] **4.8** Add idempotency checks (skip if already in desired state)

### Validation
- [ ] **4.9** Test with single-route service (atlas-account) - Requires cluster access
- [ ] **4.10** Test with multi-route service (atlas-cashshop) - Requires cluster access
- [ ] **4.11** Test multiple simultaneous debug sessions - Requires cluster access
- [ ] **4.12** Test restoration of one service while others still debug - Requires cluster access

---

## Completion Criteria

- [x] Both scripts are executable and in `tools/` directory
- [x] Scripts follow existing `tools/` code style conventions
- [x] All error cases handled gracefully
- [x] Scripts are idempotent
- [x] Documentation in --help is comprehensive
- [x] Works with all 44 Go services (designed for all, requires cluster testing)

---

## Quick Reference

### Start Debug Session
```bash
./tools/debug-start.sh --service atlas-account --target 192.168.1.100:8080
```

### Stop Debug Session
```bash
./tools/debug-stop.sh --service atlas-account
```

### Check Status
```bash
./tools/debug-start.sh --status
```

### List Services
```bash
./tools/debug-start.sh --list
```

### Restore All Services
```bash
./tools/debug-stop.sh --all
```
