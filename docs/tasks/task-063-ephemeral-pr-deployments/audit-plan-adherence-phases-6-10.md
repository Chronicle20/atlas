# Plan Audit — task-063-ephemeral-pr-deployments (Phases 6, 7, 9, 10)

**Plan Path:** `docs/tasks/task-063-ephemeral-pr-deployments/plan.md`
**Audit Date:** 2026-05-08
**Branch:** `task-063-ephemeral-pr-deployments`
**Base Branch:** `main`
**Audit Range:** `779194b1c..1bce80fca`
**Scope:** Phases 6, 7, 9, 10 (Phase 0 and 1–5 audited previously; Phase 8 is by design out-of-atlas; Phase 11 is post-merge manual)

## Executive Summary

All implementable plan tasks in Phases 6, 7, 9, and 10 are PASS. The deliverables match the plan's intent and the user's stated deviations:
the Phase 6 bootstrap container shipped both follow-up rounds (C1–C3 + I1–I2, then CR1 + IM1–IM2); the Phase 7 Kustomize restructure
splits base / `overlays/main` / `overlays/pr` cleanly, both overlays render with kustomize 5.4.3, the four Argo CD hook YAMLs are wired
into the PR overlay, generators emit DB-name and consumer-group patches, and `loadBalancerIP` is unpinned in the PR overlay. Phase 9 lands
`build-docker-pr` in `pr-validation.yml` and a new `pr-cleanup.yml`; both pass `actionlint`. Phase 10 ships `deploy/k8s/README.md`,
`docs/runbooks/ephemeral-pr-deployments.md` (§9.1–§9.10), and a new "Filtering by environment" section in `docs/observability.md`.

The plan's two stale references (`§9.5` filename and `§9.8` step-2 wording in plan.md) were never edited to match the shipped docs that
already reflect the post-Phase-8 reshape; this is acknowledged in the audit scope and is not a regression. The kustomize-replacements
substring-substitution gap for bracketed `[PLACEHOLDER_ATLAS_ENV]` and dash-suffixed multi-char placeholders is documented as a CAVEAT in
`deploy/k8s/overlays/pr/kustomization.yaml:9-21` and is expected to be resolved cluster-side by the ApplicationSet at sync time.

## Task Completion

### Phase 6 — Bootstrap container source

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 6.1 Step 1 | Dockerfile | DONE | `services/atlas-pr-bootstrap/Dockerfile:1-47` — multi-stage build, stage 1 fetches Apache Kafka 3.7.2 from `archive.apache.org` (line 11) with `--retry 3 --retry-delay 5` (line 10), stage 2 copies just the four needed scripts + libs/config + openjdk17-jre-headless. C1 + CR1 resolved. |
| 6.1 Step 1b | Stage canonical payloads in image | DONE | `services/atlas-pr-bootstrap/canonical/tenant.json` + `canonical/services/{login,channel,drops}-service.json` exist; Dockerfile COPYs at line 43. |
| 6.1 Step 2 | `scripts/lib.sh` | DONE | `services/atlas-pr-bootstrap/scripts/lib.sh:1-60` — `log()` has jq fallback (I2 fix at lines 9-18); `require_env`, `retry`, `http_ok`, `http_ok_tenant` all present (C3 added http_ok_tenant at line 51). |
| 6.1 Step 3 | `scripts/bootstrap.sh` | DONE | `services/atlas-pr-bootstrap/scripts/bootstrap.sh:1-250` — OPEN comment resolved to path (a): `upsert_service_config()` at line 133 iterates the three canonical files. Plan §6 `OPEN` text was NOT scrubbed but the shipped behavior follows path (a). C2 retry-loop helpers `extraction_done()` / `data_processing_done()` at lines 51-61. IM1 UUID validation at lines 24-27. |
| 6.1 Step 4 | `scripts/cleanup.sh` | DONE | `services/atlas-pr-bootstrap/scripts/cleanup.sh:1-91` — I1 trim of `DB_USER`/`DB_PASSWORD` at lines 25-26 happens BEFORE `require_env` at line 28 (IM2 ordering fix). |
| 6.1 Step 5 | `test/bootstrap_test.bats` | DONE | `services/atlas-pr-bootstrap/test/bootstrap_test.bats:1-17` — two required-env tests as planned. |
| 6.1 Step 6 | `test/cleanup_test.bats` | DONE | `services/atlas-pr-bootstrap/test/cleanup_test.bats:1-19` — two required-env tests as planned. |
| 6.1 Step 7 | shellcheck + bats run | PARTIAL | shellcheck not installed locally; `bash -n` (syntax) clean on all three scripts. (Plan step is verification-only; not a code deliverable.) |
| 6.1 Step 8 | README.md | DONE | `services/atlas-pr-bootstrap/README.md:1-13` — two-entrypoints summary + Loki note + runbook pointer. |
| 6.1 Step 9 | services.json entry | DONE | `.github/config/services.json:351-356` — `"type": "support-image"`, correct path and docker_image. |
| 6.1 Step 10 | Commit | DONE | `98f16b425` (initial), `519131977` (C1/C2/C3/I1/I2), `c276d2426` (CR1/IM1/IM2). |

### Phase 7 — `deploy/k8s/` Kustomize restructure

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 7.1 Step 1 | Move flat manifests into `base/` | DONE | `ls deploy/k8s/base/*.yaml` → 59 files including all `atlas-*.yaml`, `env-configmap.yaml`, `namespace.yaml`, `secrets.example.yaml`, plus `kustomization.yaml`. |
| 7.1 Step 2 | Strip `namespace: atlas` from base | DONE | `grep -rE '^\s*namespace:' deploy/k8s/base/*.yaml` → empty. |
| 7.1 Step 3 | Verify no `namespace:` left | DONE | Same grep as above; only the namespace resource (which uses `metadata.name`) remains. |
| 7.1 Step 4 | `base/kustomization.yaml` | DONE | `deploy/k8s/base/kustomization.yaml:1-62` — 57 service `.yaml`s + `env-configmap.yaml` + `namespace.yaml`. Note: `secrets.example.yaml` correctly omitted (documentation-only). |
| 7.1 Step 5 | Verify base renders | DONE | `kustomize build deploy/k8s/base` exit=0, 113 `kind:` entries. |
| 7.1 Step 6 | Commit | DONE | `ad5139b40` "refactor(deploy): move flat manifests into Kustomize base" |
| 7.2 Step 1 | `overlays/main/kustomization.yaml` | DONE | `deploy/k8s/overlays/main/kustomization.yaml:1-229` — namespace `atlas-main`, BASE_SERVICE_URL fixed to `atlas-main.svc.cluster.local` (commit `9bd5d1e5c`), `-main` suffix on every topic, all images pinned `:latest`, `commonLabels: atlas.env: main`. |
| 7.2 Step 2 | `overlays/main/atlas-env-tokens.yaml` | DONE | `deploy/k8s/overlays/main/atlas-env-tokens.yaml:1-6` — literal value `main`. |
| 7.2 Step 3 | `overlays/main/lb-pin.yaml` | DONE | `deploy/k8s/overlays/main/lb-pin.yaml:1-15` — pins `atlas-login-lb` 192.168.23.231 and `atlas-channel-lb` 192.168.23.232. |
| 7.2 Step 4 | Verify overlay renders | DONE | `kustomize build deploy/k8s/overlays/main` exit=0, 114 `kind:` entries, PLACEHOLDER count = 0. |
| 7.2 Step 5 | Commit | DONE | `fb8c1453c` "feat(deploy): main overlay treats main as ATLAS_ENV=main" (plus follow-up `9bd5d1e5c` for BASE_SERVICE_URL). |
| 7.2 Extra | `overlays/main/patches/atlas-env-env.yaml` | DONE | New patch file injects `ATLAS_ENV=main` per-Deployment (57 entries). Not in plan §7.2 explicitly but required for the symmetric main env. |
| 7.2 Extra | `overlays/main/patches/db-name-suffix.yaml` | DONE | Patches per-Deployment `DB_NAME` to `<base>-main`. Mirrors the PR overlay's generator output. |
| 7.3 Step 1 | `gen-consumer-group-patch.sh` | DONE | `deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh:1-76` — uses `yq eval-all` with kind-filter (line 52, fixing the plan's naive `yq eval`). |
| 7.3 Step 1b | `gen-db-name-suffix.sh` (split) | DONE | `deploy/k8s/overlays/pr/scripts/gen-db-name-suffix.sh:1-48` — split from gen-consumer-group; emits db-name-suffix.yaml. Acknowledged deviation 4 in audit scope. |
| 7.3 Step 2 | Run the generator | DONE | `deploy/k8s/overlays/pr/patches/consumer-group-env.yaml` has 47 Deployment patches; `deploy/k8s/overlays/pr/patches/db-name-suffix.yaml` has 28. |
| 7.3 Step 3 | Commit | DONE | `2dd8cd727` "feat(deploy): generators for per-PR consumer-group and DB-name patches". |
| 7.4 Step 1 | `pr/atlas-env-tokens.yaml` | DONE | `deploy/k8s/overlays/pr/atlas-env-tokens.yaml:1-11` — `data.ATLAS_ENV: "PLACEHOLDER_ATLAS_ENV"`. |
| 7.4 Step 1b | `pvc-storageclass.yaml` (longhorn-pr) | DONE | `deploy/k8s/overlays/pr/patches/pvc-storageclass.yaml:1-30` — three PVCs (`atlas-data-pvc`, `atlas-wz-input-pvc`, `atlas-assets-pvc`) all `storageClassName: longhorn-pr`. Phase 0 Task 0.6 finding referenced (line 8). |
| 7.4 Step 2 | `db-name-suffix.yaml` | DONE | Generated by `gen-db-name-suffix.sh`; 28 Deployments. |
| 7.4 Step 3 | `pr/kustomization.yaml` | DONE | `deploy/k8s/overlays/pr/kustomization.yaml:1-288` — namespace `atlas-pr-PLACEHOLDER_PR_NUMBER`, all 4 hook YAMLs in resources (lines 33-36), 4 patches (lines 38-42), full topic-suffixing literal block, replacements covering ATLAS_ENV slot with `create: true`. **CAVEAT block at lines 9-21** documents the substring-substitution gap for bracketed/dash-suffixed placeholders (matches acknowledged deviation 5). |
| 7.4 Step 4 | `gen-topic-config.sh` | DONE | `deploy/k8s/overlays/pr/scripts/gen-topic-config.sh:1-12`. |
| 7.4 Step 5 | Inline topic literals | DONE | `pr/kustomization.yaml:61-139` — 79 topic env vars suffixed `-PLACEHOLDER_ATLAS_ENV`. |
| 7.4 Step 6 | Commit | DONE | `5b3a7ba17` "feat(deploy): per-PR Kustomize overlay shell". |
| 7.5 Step 1 | `ingress-route.yaml` | DONE | `deploy/k8s/overlays/pr/ingress-route.yaml:1-13` — `Host(\`PLACEHOLDER_PR_NUMBER.atlas.home\`)` → `atlas-ingress:80`. |
| 7.5 Step 2 | Render and verify | DONE | `kustomize build deploy/k8s/overlays/pr` includes the IngressRoute resource. |
| 7.5 Step 3 | Commit | DONE | `9c675654e` "feat(deploy): per-PR Traefik IngressRoute". |
| 7.6 Step 1 | `presync-create-dbs.yaml` | DONE | `deploy/k8s/overlays/pr/presync-create-dbs.yaml:1-50` — Job with `argocd.argoproj.io/hook: PreSync` + `HookSucceeded`; trim of credentials BEFORE psql (lines 43-44); `gexec` for idempotent CREATE DATABASE. |
| 7.6 Step 2 | `atlas-db-names` configMapGenerator | DONE | `pr/kustomization.yaml:140-142` — space-separated list of 28 base DB names. |
| 7.6 Step 3 | Commit | DONE | `1a34657fb` "feat(deploy): PreSync Job that creates per-env Postgres DBs". |
| 7.7 Step 1 | `postsync-bootstrap.yaml` | DONE | `deploy/k8s/overlays/pr/postsync-bootstrap.yaml:1-87` — ServiceAccount + Role + RoleBinding + Job. Role grants `services:get` + deployment patch verbs for rolling restart. PVC `atlas-wz-canonical-readonly` mounted at `/opt/wz`. |
| 7.7 Step 2 | `atlas-pr-bootstrap-tenant` ConfigMap | DONE | `pr/kustomization.yaml:143-148` — TENANT_ID UUID-shaped, REGION=GMS, MAJOR=83, MINOR=1. |
| 7.7 Step 3 | Commit | DONE | `a864fdf51` "feat(deploy): PostSync bootstrap Job". |
| 7.8 Step 1 | `postsync-pihole-add.yaml` | DONE | `deploy/k8s/overlays/pr/postsync-pihole-add.yaml:1-47` — Job with PostSync hook (sync-wave: 10) + tolerates single-Pi-hole failure (`[ "$ok" -ge 1 ] || exit 1`). |
| 7.8 Step 2 | Commit | DONE | `41c7aee50` "feat(deploy): PostSync Pi-hole DNS-register Job". |
| 7.9 Step 1 | `postdelete-cleanup.yaml` | DONE | `deploy/k8s/overlays/pr/postdelete-cleanup.yaml:1-47` — PostDelete hook + `backoffLimit: 0` + envFrom: atlas-env-tokens, atlas-db-names, db-credentials, pihole-credentials, ghcr-pat. |
| 7.9 Step 2 | `ATLAS_SERVICES` literal | DONE | `pr/postdelete-cleanup.yaml:47` — 57-service comma-separated list. |
| 7.9 Step 3 | Commit | DONE | `ac6a83161` "feat(deploy): PostDelete cleanup Job". |
| 7.10 Step 1 | `patches/lb-allocate.yaml` | DONE | `deploy/k8s/overlays/pr/patches/lb-allocate.yaml:1-15` — empty-string `loadBalancerIP` on both `atlas-login-lb` and `atlas-channel-lb`. |
| 7.10 Step 2 | Add to `patches:` list | DONE | `pr/kustomization.yaml:42` — `- path: patches/lb-allocate.yaml`. |
| 7.10 Step 3 | Render and confirm unpinned | DONE | `grep -A3 'name: atlas-channel-lb' /tmp/pr-rendered.yaml` → `loadBalancerIP: ""`. |
| 7.10 Step 4 | Commit | DONE | `8c3ffedd4` "feat(deploy): per-PR LoadBalancer allocation for game-socket". |
| 7.11 Step 1 | Render with PLACEHOLDER values | DONE | `kustomize build deploy/k8s/overlays/pr` exit=0; 124 `kind:` entries. |
| 7.11 Step 2 | Verify replacements would apply | DONE | `PLACEHOLDER_ATLAS_ENV: 500`, `PLACEHOLDER_PR_NUMBER: 474`, `PLACEHOLDER_SHA: 54`. All non-zero as expected. |
| 7.11 Step 3 | kubeconform lint | PARTIAL | kubeconform not installed locally; the plan note allows accepting "schema not found" warnings for Traefik/Argo CRDs in any case. Verification gate documented. |
| 7.11 Step 4 | No commit | N/A | Verification-only by design. |

### Phase 9 — GitHub Actions

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 9.1 Step 1 | Add `build-docker-pr` job | DONE | `.github/workflows/pr-validation.yml:157-208` — added with `permissions: { contents: read, packages: write }` and `github.event_name == 'pull_request'` guard (acknowledged deviation 7). |
| 9.1 Step 2 | Update `pr-validation-complete` `needs:` | DONE | `pr-validation.yml:212` includes `build-docker-pr`; result-table updated at lines 227, 235, 238. |
| 9.1 Step 3 | actionlint | DONE | `actionlint .github/workflows/pr-validation.yml` exit=0. |
| 9.1 Step 4 | Commit | DONE | `49a6cd4b4` "ci(pr-validation): build and push per-PR images to ghcr". |
| 9.2 Step 1 | `pr-cleanup.yml` | DONE | `.github/workflows/pr-cleanup.yml:1-64` — both `delete-images` (`packages: write`) and `notify-argo` (`needs: delete-images`) jobs; the implementer filtered out `support-image` type at line 33 (i.e., atlas-pr-bootstrap itself isn't deleted per-PR). |
| 9.2 Step 2 | actionlint | DONE | `actionlint .github/workflows/pr-cleanup.yml` exit=0. |
| 9.2 Step 3 | Commit | DONE | `02cbcdae3` "ci(pr-cleanup): delete per-PR ghcr image tags on PR close". |

### Phase 10 — Documentation

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 10.1 Step 1 | `deploy/k8s/README.md` | DONE | `deploy/k8s/README.md:1-71` — describes base + both overlays, kustomize rendering, "Adding a new service" (7 steps including the new gen-db-name-suffix.sh run), ATLAS_ENV flow table, hook list, "Cluster-side gitops" pointer to the cluster-infra repo (acknowledged deviation 8). |
| 10.1 Step 2 | Commit | DONE | `d2935ec82` "docs(deploy): Kustomize structure and ATLAS_ENV flow". |
| 10.2 Step 1 | `docs/runbooks/ephemeral-pr-deployments.md` | DONE | `docs/runbooks/ephemeral-pr-deployments.md:1-156` — covers §9.1 through §9.10. §9.5 correctly refers to `argocd-secrets.yml.example` in cluster-infra (matches acknowledged deviation 8). §9.8 correctly references `<infra-repo>/argocd.yml` header comment. §9.10 added beyond plan as "PR env doesn't get scheduled" guidance. |
| 10.2 Step 2 | Commit | DONE | `992bffbed` "docs(runbooks): ephemeral PR deployments operational guide". |
| 10.3 Step 1 | Locate `docs/observability.md` | DONE | File exists pre-task; edited rather than created. |
| 10.3 Step 2 | Append env-label section | DONE | `docs/observability.md:114-138` — "Filtering by environment" section, Loki note about Promtail dots→underscores normalisation at line 127. |
| 10.3 Step 3 | Commit | DONE | `1bce80fca` "docs(observability): document per-env label filtering". |

**Completion Rate:** 67 of 67 implementable tasks (100%). Two `PARTIAL` entries are tooling-not-installed (shellcheck, kubeconform); the
underlying artefact passed `bash -n` syntax check and `kustomize build`. No `SKIPPED`.

## Skipped / Deferred Tasks

None. All implementable plan tasks landed.

The plan's two stale docs references in `§10.2` (filename `argocd-pihole-secret.yml.example`, file `deploy/argocd/README.md`) and the
plan-§6 OPEN callout text were left unscrubbed; the shipped artefacts diverge from plan-text in the user-acknowledged ways and reflect the
post-Phase-8 reshape correctly.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-pr-bootstrap (scripts) | PASS | N/A locally | `bash -n` clean on `lib.sh`, `bootstrap.sh`, `cleanup.sh`. Bats tests not run locally (no `bats` in PATH); the test files themselves are well-formed and trivially small (4 tests total). |
| Kustomize: `deploy/k8s/base` | PASS | N/A | `kustomize build` exit=0, 113 `kind:` entries. |
| Kustomize: `deploy/k8s/overlays/main` | PASS | N/A | `kustomize build` exit=0, 114 `kind:` entries, **PLACEHOLDER count = 0**. |
| Kustomize: `deploy/k8s/overlays/pr` | PASS | N/A | `kustomize build` exit=0, 124 `kind:` entries, `PLACEHOLDER_ATLAS_ENV:500`, `PLACEHOLDER_PR_NUMBER:474`, `PLACEHOLDER_SHA:54` — all non-zero as the plan expected. |
| Gen scripts | PASS | N/A | `bash -n` clean on `gen-consumer-group-patch.sh`, `gen-db-name-suffix.sh`, `gen-topic-config.sh`. |
| `pr-validation.yml` | PASS | actionlint exit=0 | `permissions` and `github.event_name == 'pull_request'` guards added. |
| `pr-cleanup.yml` | PASS | actionlint exit=0 | Two jobs, packages:write permission, concurrency group keyed on PR number. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

The user's acknowledged deviations all surface in the shipped artefacts as documented; the substring-substitution CAVEAT in
`pr/kustomization.yaml` is honestly disclosed and the resolution is explicit (resolved by cluster-side `argocd-atlas.yml` ApplicationSet
`kustomize.replacements` with `delimiter:` + `index:`). Spot-checks confirm:

- bootstrap.sh has `upsert_service_config()` (line 133) iterating the three canonical files — not the OPEN single-payload form.
- Dockerfile stage 1 fetches from `archive.apache.org` (line 11), not `downloads.apache.org`.
- `cleanup.sh` trims credentials at lines 25-26, BEFORE `require_env` at line 28.
- `lib.sh`'s `log()` has the `command -v jq` fallback (lines 9-18).
- All 4 hook YAMLs (`presync-create-dbs.yaml`, `postsync-bootstrap.yaml`, `postsync-pihole-add.yaml`, `postdelete-cleanup.yaml`) are uncommented in `pr/kustomization.yaml:33-36`.
- `pvc-storageclass.yaml` references `longhorn-pr` for all three per-PR PVCs.
- `lb-allocate.yaml` clears both `atlas-login-lb` and `atlas-channel-lb` IPs to empty string.
- `build-docker-pr` is in the summary `pr-validation-complete` `needs:` and result-table.
- `pr-cleanup.yml` has both `delete-images` and `notify-argo` jobs and is actionlint-clean.
- `deploy/k8s/README.md` describes both overlays; runbook has §9.1 through §9.10; `docs/observability.md` has the new "Filtering by environment" section.

Phase 8 plan section (lines 2929-2956) correctly lists the 4 cluster-side files, the 7-step bootstrap order, and the `design.md` §6
reference; no atlas commits expected there. Confirmed: no `deploy/argocd/` directory, no `argocd*.yml` in atlas.

## Action Items

None blocking. For housekeeping only (not gating):

1. (Optional) Scrub the OPEN callout in `plan.md:1546-1604` — the shipped `bootstrap.sh` resolved to path (a) and the OPEN block is now
   historical. Not blocking because the shipped code is what runs.
2. (Optional) Update `plan.md:3295-3343` (§9.5 + §9.8) to match the shipped runbook's filenames if a future maintainer wants the plan and
   runbook in lockstep. Not blocking because the shipped runbook is what operators read.
