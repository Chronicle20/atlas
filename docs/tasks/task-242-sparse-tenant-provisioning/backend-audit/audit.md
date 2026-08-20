# Backend Audit — atlas-pr-bootstrap (task-242-sparse-tenant-provisioning)

- **Service Path:** services/atlas-pr-bootstrap (+ tools/, deploy/k8s/)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-19
- **Range:** 8dfb4f99a..1eeb1e223 (30 files, +4728/-50)
- **Build:** N/A — no Go changed (`go_changed=false`); Dockerfile/shell only
- **Tests:** 34 passed, 0 failed (`bats test/env_record_test.bats test/tenant_provisioning_test.bats test/dockerfile_test.bats`)
- **Overall:** PASS

## Build & Test Results

No Go packages are in scope (`go_changed=false` per `tools/task-facts.sh`), so
Phase 1's `go build`/`go test` gate does not apply — this service has no
`atlas.com/<module>` Go tree. In its place I ran the objective checks that
exist for this surface:

```
$ cd services/atlas-pr-bootstrap && bats test/env_record_test.bats test/tenant_provisioning_test.bats test/dockerfile_test.bats
1..34
ok 1..34  (all pass)

$ ./tools/shell-guard.sh --require-shellcheck
shell-guard: 68 script(s) OK (syntax + shellcheck -S error).   exit=0

$ bash tools/overlay-env-guard.sh
... 19 PASS, 1 SKIP (documented pr-sparse placeholder exclusion), 0 FAIL   exit=0

$ ./tools/pr-sparse-mirror-guard.sh
pr-sparse-mirror-guard: up to date   exit=0
```

All four are clean. Treating this as PASS/FAIL-equivalent to Phase 1: no
build/test failure blocks the audit.

## Applicability

Per `tools/task-facts.sh`: `changed_services=atlas-pr-bootstrap`,
`go_changed=false`, `db_surface=true`, `deploy_surface=true`,
`backend_audit_families=DEPLOY`.

| Family | Fired? | Evidence |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | N/A | No changed Go package — diff has zero `.go` files (`git diff --stat` shows only `.sh`, `.bats`, `.yaml`, `.md`, `Dockerfile`). |
| FILE placement (FILE-01..06) | N/A | Same — no changed Go package. |
| SUB (SUB-01..04) | N/A | Same. |
| REST (DOM-06..09,12..15,17..19,32) | N/A | No `resource.go`/`rest.go`/`processor.go`, no HTTP route registration in Go. |
| Constants reuse (DOM-21) | N/A | No new Go type/const/numeric-literal classification. |
| Testing (DOM-10,20,24,33) | N/A (Go-specific) | Diff does add/change tests, but they are `.bats`, not `_test.go`; the rule's own mechanism (table-driven Go tests, `producertest`, mock-interface parity) has no Go surface to apply to. |
| Cache (DOM-29) | N/A | No `cache.go`. |
| Messaging (DOM-30) | N/A | No `producer.go`, no `AndEmit`/`message.Emit`/`producer.ProviderImpl` call added. |
| Multi-tenancy (DOM-31) | N/A (Go-specific), but evaluated in spirit — see below | No `rest.go` in scope; the rule's own trigger (Go code reading `tenant.MustFromContext`/`db.WithContext`) doesn't fire on shell. The reviewer brief specifically asked me to verify environment-scoping correctness by reading code — done as a targeted lookup (see Checklist Results). |
| Migration hygiene (DOM-34/35) | N/A | No symbol moved into/out of `libs/atlas-*`. |
| **Deploy & topics (DOM-22, DOM-23)** | **Fired for DOM-22? No. DOM-23? No.** | DOM-22 triggers on a new `libs/atlas-*` module — none added. DOM-23 triggers on adding/renaming a `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` env var — the diff adds `ATLAS_ENVIRONMENT` (not a topic var) to `env-configmap.yaml`/overlay generators; `deploy/k8s/overlays/pr-sparse/patches/consumer-group-env.yaml` adds a `KAFKA_CONSUMER_GROUP`/`ATLAS_ENV` block for `atlas-events`, mirroring the existing pattern for other pr-sparse consumer-group patches — not a new topic name. Both DOM-22 and DOM-23 are N/A. `patterns-deploy.md` opened and read in full; neither rule's own trigger fires despite `deploy_surface=true`. |
| Runtime safety (DOM-26) | N/A | Rule is Go-goroutine-specific (`routine.Go`); no non-test `.go` file changed. |
| Channel wire values (DOM-25) | N/A | No `services/atlas-channel`/`libs/atlas-packet` touched, no client-interpreted byte added. |
| Resilience (DOM-27/28) | N/A | Go-specific (`server.WriteErrorResponse`, `model.Decorator`); no Go changed. |
| External clients (EXT-01..04) | N/A | Go-specific (`requests.RootUrl`/`requests.*Request[T]`); the shell scripts call other services via raw `curl`, which is a different surface with no EXT-* rule written for it. |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new `services/atlas-<svc>/` directory, no new channel Writer/Handler, `deploy/shared/routes.conf` untouched. |
| **Security (SEC-01..04)** | **SEC-04 evaluated; SEC-01/02/03 N/A** | atlas-pr-bootstrap is not an auth/token/redirect service (SEC-01/02/03's triggers — token parsing, revocation, redirect handlers — do not exist here). It does handle secrets pre-existing to this diff (`DB_PASSWORD`, `GHCR_TOKEN`, `PIHOLE_TOKEN`) so SEC-04 is in scope for the diff's own hunks: no new hardcoded secret found (see Checklist Results). |

## Checklist Results

### atlas-pr-bootstrap/scripts (support — shell, not a Go package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-22 | New `libs/atlas-*` wired into Dockerfile/go.work | N/A | No `libs/` module added in this diff (`git diff --stat` shows no `libs/` path). |
| DOM-23 | Kafka topic env var naming/configmap/overlay parity | N/A | No `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` var added or renamed; `ATLAS_ENVIRONMENT` is not a topic var and `deploy/k8s/overlays/pr-sparse/patches/consumer-group-env.yaml`'s new block only adds `KAFKA_CONSUMER_GROUP`/`ATLAS_ENV` to a consumer's env, mirroring the sibling patches already in that file for other services. |
| SEC-04 | No hardcoded secrets | PASS | `git diff ... | grep -niE "password|secret|token|apikey"` on the full diff returns only a pre-existing comment referencing `db-credentials secret` (cleanup.sh:29, unchanged text moved) and an unrelated `atlas-env-tokens.yaml` filename in the runbook doc (docs/runbooks/ephemeral-pr-deployments.md:234) — no literal credential value. `env-record.sh`, `bootstrap.sh`'s new `env_header_init`/`find_environment_tenant`/`create_environment_tenant`/`record_environment_tenant` functions carry no secret material — they pass `ATLAS_ENVIRONMENT` (an env-id string, not a credential) as a header. |
| (shell safety, foundational — no rule ID; graded against the reviewer's explicit ask) | Quoting / `set -uo pipefail` interaction / injection | PASS | `services/atlas-pr-bootstrap/scripts/bootstrap.sh:14` sets `set -euo pipefail`; line 21 explicitly restores `set -e` after `lib.sh` (sourced line 17) intentionally relaxes to `-uo pipefail` for its own try-all semantics — documented at bootstrap.sh:19-20. `ENV_HEADER` is built as a bash array specifically to avoid word-splitting on `"ENVIRONMENT: $ATLAS_ENVIRONMENT"` (bootstrap.sh:41-44, verified: `"${ENV_HEADER[@]}"` used at bootstrap.sh:76, 92). `TENANT_ID` is validated against a UUID regex before any header use (bootstrap.sh:135-138) — the one identifier in this flow that both crosses the network and lacks a fixed CI-controlled shape gets an explicit format gate; `ATLAS_ENVIRONMENT`/`PR_NUMBER`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION` do not get an equivalent gate, but all originate from CI-controlled values (`PR_NUMBER` from the GitHub Actions event, `ATLAS_ENVIRONMENT` derived as `pr-${PR_NUMBER}` at cleanup.sh:46 or injected by the Helm chart for bootstrap), not from external/attacker-controlled input — noted, not a finding. `env_record_patch` (env-record.sh:38-54) builds its JSON body entirely through `jq -nc --arg`/`--argjson`, never string interpolation into the JSON literal, so a `tenant`/`namespace`/`baseline` value containing `"` or `\` cannot break the payload shape. |
| (multi-tenancy correctness, foundational — reviewer's explicit ask) | Tenant lookup does not cross environment boundaries | PASS | `find_environment_tenant` (bootstrap.sh:74-81) sends `"${ENV_HEADER[@]}"` (the `ENVIRONMENT: $ATLAS_ENVIRONMENT` header, sparse-mode only) on the `GET /api/tenants` call. Targeted lookup: `services/atlas-tenants/atlas.com/tenants/tenant/provider.go:19-33` — `GetByIdProvider`/`getAll` both call `env.MustFromContext(ctx)` and wrap every query in `scope.Strict(db.WithContext(ctx), caller)`, i.e. the server itself enforces the boundary from the request's `ENVIRONMENT` header, not merely a client-side filter. `create_environment_tenant` (bootstrap.sh:87-101) also sends `"${ENV_HEADER[@]}"` on the POST, and the comment at bootstrap.sh:84-86 — confirmed against `services/atlas-tenants/atlas.com/tenants/tenant/entity.go:16` (`Environment string \`gorm:"not null;default:''"\`` is set server-side from context, not the request body) — is accurate: the header is the only way to stamp environment on the new row. `record_environment_tenant` (bootstrap.sh:114-126) only runs when `ATLAS_MODE=sparse` (bootstrap.sh:361) and requires `env_record_get`/`env_record_patch`, both of which require `ATLAS_ENVIRONMENT` to be set (env-record.sh:16-17) — an isolated bootstrap never calls it. In isolated mode `ENV_HEADER` is empty (bootstrap.sh:56-61) so every curl argv is byte-identical to pre-existing behavior, verified by bats test "env_header_init leaves ENV_HEADER empty in isolated mode" (tenant_provisioning_test.bats) and "find_environment_tenant sends no ENVIRONMENT header in isolated mode" — both pass (see Build & Test Results). |
| (idempotent create/PATCH) | `env_record_patch` sends all five attributes; `record_environment_tenant` preserves current phase | PASS | env-record.sh:38-54 — `env_record_patch` builds the payload with `phase`, `baseline`, `namespace`, `tenant`, `overrides` all present, per the doc comment at env-record.sh:23-37 explaining why a partial body would zero the others. `record_environment_tenant` (bootstrap.sh:114-126) reads the record's current `phase`/`baseline`/`namespace`/`overrides` via `env_record_get` before PATCHing, so the tenant-only intent doesn't clobber the other four — confirmed by passing bats tests "record_environment_tenant carries baseline, namespace and overrides through unchanged" and "carries the record's current phase, never an empty one". |

### tools/overlay-env-guard.sh (support — deploy-verification script)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| (script-verified assertion, per Phase 3's "settled by running the script" rule) | Guard renders and asserts on all three overlays; wired into `verify.sh` | PASS | `bash tools/overlay-env-guard.sh` exit 0, 19 PASS + 1 documented SKIP, 0 FAIL (ran above). `tools/verify.sh:551-559` adds `overlay-env-guard.sh` to the `touched '^(deploy/...)'` gate and a new `step "overlay env drift" ./tools/overlay-env-guard.sh` — confirmed present in the diff (`git diff tools/verify.sh`). |
| — assertion 9/10 substring match (non-blocking, confirmed) | `check_ingress_default_env`'s `contains()` uses `grep -qF` substring matching, not an exact-line match | CONFIRMED, non-blocking | `tools/overlay-env-guard.sh:107-110` (`contains()`) is `printf '%s\n' "$1" | grep -qF -- "$2"` — a substring test. Assertion 9 (lines 204-209) checks for the substrings `"- name: ATLAS_ENVIRONMENT_DEFAULT"`, `"key: ATLAS_ENVIRONMENT"`, `"name: atlas-env"` independently rather than a single anchored block, so a `key: ATLAS_ENVIRONMENT` belonging to an unrelated env var earlier in the same container spec would also satisfy it. Not exploitable today (verified: `deploy/k8s/base/atlas-ingress.yaml`'s only `key: ATLAS_ENVIRONMENT` is the one added at lines 122-125 of that file's diff), but the guard would not catch a future regression that moved the three lines apart. Downgraded to non-blocking because the guard still correctly fails when the block is fully absent, which is the regression it was written to catch. |
| — assertion 10 pins the full literal | `check_ingress_default_env`'s second check pins the entire `NGINX_ENVSUBST_FILTER` value as one literal string | CONFIRMED, non-blocking | `tools/overlay-env-guard.sh:212` — `contains "$doc" "value: POD_NAMESPACE|NS_|ATLAS_ENVIRONMENT_DEFAULT"`. This is brittle (a future `NS_` var addition changes nothing here since the literal is a fixed prefix substring within the larger generated block, but a reordering of the three terms would false-fail even though the effective envsubst filter set is unchanged) but not a correctness defect — the assertion still exercises the real rendered value, it is just over-pinned. Confirmed via source read; not independently exploitable as a false PASS. |

### services/atlas-pr-bootstrap/scripts/bootstrap.sh (line-level findings raised by prior per-task reviewers)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — `-d @"$CANONICAL_TENANT_JSON"` | Claimed: POST uses a variable path rather than a hardcoded path | REFUTED as a defect | bootstrap.sh:93 does read `-d @"$CANONICAL_TENANT_JSON"`, but `CANONICAL_TENANT_JSON` is declared at bootstrap.sh:35 as `"${CANONICAL_TENANT_JSON:-/atlas/canonical/tenant.json}"` — a default with an explicit, documented test seam ("Overridable so bats can point at a fixture," bootstrap.sh:34). This is the intended, correct pattern (matches `CANONICAL_TENANT_JSON` usage throughout the rest of the file, e.g. lines 281-284, 321-323) — not a deviation from any guideline. Non-blocking / not a finding at all. |
| — `env_record_patch \|\| exit 1` | Claimed: exit code of `env_record_patch` is flattened by the trailing `|| exit 1` | CONFIRMED, non-blocking | bootstrap.sh:362 — `ATLAS_STEP=record-tenant record_environment_tenant "$TENANT_ID" || exit 1` does discard `record_environment_tenant`'s actual nonzero code and always exits 1 on any failure. The comment at bootstrap.sh:356-360 states this is deliberate belt-and-braces against `set -e` being relaxed again (it was, once, by `lib.sh`) — a documented, intentional tradeoff, not an oversight. No guideline rule requires exit-code fidelity here; downgraded to non-blocking, matching the prior reviewers' disposition. |
| — `CURL_ARGS` append vs. truncate | Claimed: bats CURL_ARGS write changed from `>` (truncate) to `>>` (append) | CONFIRMED, non-blocking, and correct | `services/atlas-pr-bootstrap/test/env_record_test.bats:51` — `printf '%s\n' "$@" >>"$CURL_ARGS"`, with the shim's own comment (lines 46-49) explaining that `record_environment_tenant` makes a GET then a PATCH and both argvs must be visible in the same file. Each bats `@test` gets a fresh `$BATS_TEST_TMPDIR` (env_record_test.bats:20, 43) so there is no cross-test leakage — the append is required for the two-call assertions and does not weaken any single-call assertion (`patch_payload()` at lines 62-72 scans for the one line that parses as JSON, unaffected by extra non-JSON argv lines). Not a defect. |

## Not evaluable from the diff

- Whether `atlas-configurations`' `UpdateByName`/`ParseInput` truly zero omitted PATCH attributes and backfill from the existing record the way `env-record.sh:23-32`'s comment describes, beyond the one file I read (`tenant/entity.go`, `tenant/provider.go`) — I did not read `services/atlas-configurations/.../environments/administrator.go` or `processor.go` (not in this diff's changed-file set; the comment's claim is plausible and consistent with the tenant-scoping pattern I did verify, but I did not open those specific files to confirm processor.go:224-226/243-255's line numbers).
- Whether `libs/atlas-env/registry.go`'s `MapRegistry.EnvironmentsOwnedBy`/`IsOwner` behave exactly as `deploy/k8s/overlays/main/kustomization.yaml`'s new comment and `deploy/k8s/overlays/main/environment-record.yaml`'s header comment describe (lines 5-19 of that Job's script) — `libs/atlas-env` is not in this diff's changed-file set and reading it would exceed the scoped-review surface.
- Runtime behavior of the new `atlas-environment-record` Job and the `env-default.conf.template` nginx `map` block under a live cluster — both were verified by static read and by `overlay-env-guard.sh`'s rendered-YAML assertions, not by an actual `kubectl apply`/live nginx reload (out of reach for this review).

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- SUB-guard-assertion-9 (`tools/overlay-env-guard.sh:204-209`): the three `contains()` substring checks for `ATLAS_ENVIRONMENT_DEFAULT` wiring are independent, not anchored to a single contiguous block — could false-PASS if a future edit scattered similarly-named keys elsewhere in the same container spec.
- guard-assertion-10 (`tools/overlay-env-guard.sh:212`): pins the entire `NGINX_ENVSUBST_FILTER` value as one literal string, which is more brittle than necessary but not currently a false-pass risk.
- bootstrap.sh:362: `env_record_patch`'s real exit code is flattened by `|| exit 1` — intentional per the inline comment, but worth a future revisit if `record_environment_tenant`'s failure modes ever need to be distinguished at the call site.
