#!/usr/bin/env bash
# Pins the per-overlay environment id and the ingress ATLAS_ENVIRONMENT
# default that task-242 (sparse tenant provisioning) wires into the
# rendered manifests:
#
#   - each overlay's atlas-env ConfigMap carries the right ATLAS_ENVIRONMENT
#     literal (main / "" / pr-PLACEHOLDER_PR_NUMBER) — FR-1.1, FR-1.2, FR-1.5,
#     FR-4.2
#   - overlays/main renders an atlas-environment-record Job that POSTs
#     directly to atlas-configurations (bypassing atlas-ingress) with
#     phase=ACTIVE — D1
#   - overlays/pr renders no such Job — FR-1.5
#   - the atlas-ingress container/ConfigMaps carry the D3 ingress-default
#     wiring (ATLAS_ENVIRONMENT_DEFAULT env var, NGINX_ENVSUBST_FILTER,
#     env-default.conf.template, and the resolved-environment
#     proxy_set_header) — FR-4.1, D3
#
# Renders each overlay once with `kustomize build` and asserts against the
# rendered YAML stream with plain `grep -F`/awk — no new toolchain
# dependency (see tools/pr-sparse-mirror-guard.sh, the closest existing
# pattern, for the same style).
#
# NOTE on overlays/pr-sparse and the ingress env-var assertions (9/10 in
# the task-242 Task 6 brief's assertion table): overlays/pr-sparse/
# ns-overrides.yaml:38-46 is a strategic-merge patch targeting the
# atlas-ingress Deployment's `nginx` container, checked in as
# `env: #PLACEHOLDER_NS_OVERRIDES` — which YAML-parses to `env: null`.
# Verified directly (in-vitro kustomize repro, task-242 Task 6 report): an
# explicit but empty `env:` in a strategic-merge patch clears the base
# container's entire env list. So a raw, placeholder-unfilled
# `kustomize build overlays/pr-sparse` genuinely has NO env vars at all on
# the ingress container (not just a missing ATLAS_ENVIRONMENT_DEFAULT —
# POD_NAMESPACE, NGINX_ENVSUBST_FILTER, every NS_* var are gone too).
# Controller-ruled (task-242 Task 6, Task 5 review) as by-design and
# pre-existing, not a defect: .github/workflows/pr-validation.yml:1027
# substitutes PLACEHOLDER_NS_OVERRIDES with the real `- name: NS_<SVC>`
# list at deploy time, on the bot/pr-<N>-resolved branch — the raw overlay
# is never applied as-is. Do NOT edit ns-overrides.yaml or the workflow to
# make this pass; the placeholder is the intended mechanism, and
# overlays/pr-sparse is read-only for Task 6 regardless. Assertions 9 and
# 10 are therefore scoped to base/overlays/main and overlays/pr only, and
# print an explicit SKIP line for pr-sparse rather than silently omitting
# it. The ConfigMap-side assertions (8, 11, env-default.conf.template key;
# 12, nginx.conf's proxy_set_header) are unaffected by that patch and are
# still checked against all three overlays, per the brief.
set -euo pipefail

usage() {
    echo "usage: overlay-env-guard.sh [--help]"
}

if [ "${1:-}" = "--help" ]; then
    usage
    exit 0
fi

if ! command -v kustomize >/dev/null 2>&1; then
    echo "overlay-env-guard: kustomize not found on PATH" >&2
    exit 1
fi

REPO_ROOT="$(git rev-parse --show-toplevel)"
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

status=0

fail() {
    echo "overlay-env-guard: FAIL - $1" >&2
    status=1
}

pass() {
    echo "overlay-env-guard: PASS - $1"
}

skip() {
    echo "overlay-env-guard: SKIP - $1"
}

render() {
    local overlay="$1"
    kustomize build "$REPO_ROOT/deploy/k8s/overlays/$overlay" > "$WORKDIR/$overlay.yaml"
}

render main
render pr
render pr-sparse

MAIN="$WORKDIR/main.yaml"
PR="$WORKDIR/pr.yaml"
SPARSE="$WORKDIR/pr-sparse.yaml"

# get_doc <file> <kind> <name-regex>
# Kustomize joins rendered documents with a line consisting solely of
# "---". Extracts the single document whose `kind:` line matches <kind>
# and whose top-level `metadata.name:` (2-space indented, not a nested
# field such as a container name) matches <name-regex>. Empty if no such
# document exists.
get_doc() {
    local file="$1" kind="$2" name="$3"
    awk -v RS='\n---\n' -v kind="$kind" -v name="$name" '
        $0 ~ ("(^|\n)kind: " kind "\n") && $0 ~ ("\n  name: " name "(\n|$)") { print; exit }
    ' "$file"
}

contains() {
    # $1 = haystack (may be multi-line), $2 = literal needle
    printf '%s\n' "$1" | grep -qF -- "$2"
}

# --- Assertions 1-2: overlays/main's atlas-env ConfigMap (FR-1.1, FR-1.2) ---

main_env_cm="$(get_doc "$MAIN" ConfigMap 'atlas-env')"
if [ -z "$main_env_cm" ]; then
    fail "overlays/main: no atlas-env ConfigMap rendered"
elif contains "$main_env_cm" "  ATLAS_ENVIRONMENT: main"; then
    pass "overlays/main atlas-env ConfigMap has ATLAS_ENVIRONMENT: main"
else
    fail "overlays/main atlas-env ConfigMap does not have ATLAS_ENVIRONMENT: main"
fi

main_namespace="$(grep -m1 '^namespace: ' "$REPO_ROOT/deploy/k8s/overlays/main/kustomization.yaml" | sed 's/^namespace: *//')"
main_derived_id="${main_namespace#atlas-}"
if [ -z "$main_namespace" ] || [ "$main_namespace" = "$main_derived_id" ]; then
    fail "overlays/main/kustomization.yaml: could not derive an environment id from namespace: '$main_namespace' (expected an atlas- prefix)"
elif contains "$main_env_cm" "  ATLAS_ENVIRONMENT: $main_derived_id"; then
    pass "overlays/main ATLAS_ENVIRONMENT ($main_derived_id) matches namespace ($main_namespace) with the atlas- prefix stripped"
else
    fail "overlays/main ATLAS_ENVIRONMENT does not match namespace ($main_namespace) with the atlas- prefix stripped (expected $main_derived_id)"
fi

# --- Assertions 3-5: overlays/main's atlas-environment-record Job (D1) ---

main_env_job="$(get_doc "$MAIN" Job 'atlas-environment-record')"
if [ -z "$main_env_job" ]; then
    fail "overlays/main: no atlas-environment-record Job rendered"
else
    if contains "$main_env_job" 'argocd.argoproj.io/sync-wave: "11"'; then
        pass "overlays/main atlas-environment-record Job carries sync-wave 11"
    else
        fail "overlays/main atlas-environment-record Job is missing argocd.argoproj.io/sync-wave: \"11\""
    fi

    if contains "$main_env_job" "atlas-configurations.atlas-main.svc.cluster.local"; then
        if contains "$main_env_job" "atlas-ingress"; then
            fail "overlays/main atlas-environment-record Job's script routes through atlas-ingress instead of bypassing it"
        else
            pass "overlays/main atlas-environment-record Job's script targets atlas-configurations directly, bypassing atlas-ingress"
        fi
    else
        fail "overlays/main atlas-environment-record Job's script does not target atlas-configurations.atlas-main.svc.cluster.local"
    fi

    if contains "$main_env_job" '"phase": "ACTIVE"' && contains "$main_env_job" '"name": "main"'; then
        pass "overlays/main atlas-environment-record Job's POST body has phase ACTIVE and name main"
    else
        fail "overlays/main atlas-environment-record Job's POST body is missing \"phase\": \"ACTIVE\" and/or \"name\": \"main\""
    fi
fi

# --- Assertions 6-7: overlays/pr's atlas-env ConfigMap and absent Job (FR-1.5) ---

pr_env_cm="$(get_doc "$PR" ConfigMap 'atlas-env')"
if [ -z "$pr_env_cm" ]; then
    fail "overlays/pr: no atlas-env ConfigMap rendered"
elif contains "$pr_env_cm" '  ATLAS_ENVIRONMENT: ""'; then
    pass "overlays/pr atlas-env ConfigMap has ATLAS_ENVIRONMENT present with an empty value"
else
    fail "overlays/pr atlas-env ConfigMap does not have ATLAS_ENVIRONMENT: \"\""
fi

pr_env_job="$(get_doc "$PR" Job 'atlas-environment-record')"
if [ -z "$pr_env_job" ]; then
    pass "overlays/pr renders no atlas-environment-record Job"
else
    fail "overlays/pr unexpectedly renders an atlas-environment-record Job"
fi

# --- Assertion 8: overlays/pr-sparse's atlas-env ConfigMap (FR-4.2) ---

sparse_env_cm="$(get_doc "$SPARSE" ConfigMap 'atlas-env')"
if [ -z "$sparse_env_cm" ]; then
    fail "overlays/pr-sparse: no atlas-env ConfigMap rendered"
elif contains "$sparse_env_cm" "  ATLAS_ENVIRONMENT: pr-PLACEHOLDER_PR_NUMBER"; then
    pass "overlays/pr-sparse atlas-env ConfigMap has ATLAS_ENVIRONMENT: pr-PLACEHOLDER_PR_NUMBER"
else
    fail "overlays/pr-sparse atlas-env ConfigMap does not have ATLAS_ENVIRONMENT: pr-PLACEHOLDER_PR_NUMBER"
fi

# --- Assertions 9-10: atlas-ingress's ATLAS_ENVIRONMENT_DEFAULT wiring
# (FR-4.1, D3) — main and pr only; see the file header for why pr-sparse's
# raw, placeholder-unfilled render is excluded here. ---

check_ingress_default_env() {
    local overlay="$1" file="$2"
    local doc
    doc="$(get_doc "$file" Deployment 'atlas-ingress')"
    if [ -z "$doc" ]; then
        fail "overlays/$overlay: no atlas-ingress Deployment rendered"
        return
    fi

    if contains "$doc" "- name: ATLAS_ENVIRONMENT_DEFAULT" \
        && contains "$doc" "key: ATLAS_ENVIRONMENT" \
        && contains "$doc" "name: atlas-env"; then
        pass "overlays/$overlay atlas-ingress carries ATLAS_ENVIRONMENT_DEFAULT sourced from atlas-env/ATLAS_ENVIRONMENT"
    else
        fail "overlays/$overlay atlas-ingress is missing the ATLAS_ENVIRONMENT_DEFAULT configMapKeyRef (atlas-env/ATLAS_ENVIRONMENT)"
    fi

    if contains "$doc" "value: POD_NAMESPACE|NS_|ATLAS_ENVIRONMENT_DEFAULT"; then
        pass "overlays/$overlay atlas-ingress NGINX_ENVSUBST_FILTER includes ATLAS_ENVIRONMENT_DEFAULT"
    else
        fail "overlays/$overlay atlas-ingress NGINX_ENVSUBST_FILTER does not include ATLAS_ENVIRONMENT_DEFAULT"
    fi
}

check_ingress_default_env main "$MAIN"
check_ingress_default_env pr "$PR"
skip "overlays/pr-sparse atlas-ingress ATLAS_ENVIRONMENT_DEFAULT/NGINX_ENVSUBST_FILTER (ns-overrides.yaml:38-46's env: #PLACEHOLDER_NS_OVERRIDES YAML-parses to env: null, wiping the container's env list until .github/workflows/pr-validation.yml:1027 substitutes it at PR-apply time; by design, not this guard's concern)"

# --- Assertions 11-12: the ingress ConfigMaps, all three overlays (D3, FR-4.1) ---

check_ingress_configmaps() {
    local overlay="$1" file="$2"

    local routes_cm
    routes_cm="$(get_doc "$file" ConfigMap 'atlas-ingress-routes-[^ ]*')"
    if [ -z "$routes_cm" ]; then
        fail "overlays/$overlay: no atlas-ingress-routes-* ConfigMap rendered"
    elif contains "$routes_cm" "env-default.conf.template:"; then
        pass "overlays/$overlay atlas-ingress-routes-* ConfigMap carries env-default.conf.template"
    else
        fail "overlays/$overlay atlas-ingress-routes-* ConfigMap is missing the env-default.conf.template key"
    fi

    local nginx_cm
    nginx_cm="$(get_doc "$file" ConfigMap 'atlas-ingress-configmap')"
    if [ -z "$nginx_cm" ]; then
        fail "overlays/$overlay: no atlas-ingress-configmap ConfigMap rendered"
        return
    fi

    # shellcheck disable=SC2016 # literal nginx $-variable text, not shell expansion
    if contains "$nginx_cm" 'proxy_set_header ENVIRONMENT $atlas_environment;'; then
        # shellcheck disable=SC2016 # literal nginx $-variable text, not shell expansion
        if contains "$nginx_cm" 'proxy_set_header ENVIRONMENT $http_environment;'; then
            fail "overlays/$overlay atlas-ingress-configmap nginx.conf still has the old proxy_set_header ENVIRONMENT \$http_environment; line"
        else
            pass "overlays/$overlay atlas-ingress-configmap nginx.conf resolves ENVIRONMENT via \$atlas_environment"
        fi
    else
        fail "overlays/$overlay atlas-ingress-configmap nginx.conf is missing proxy_set_header ENVIRONMENT \$atlas_environment;"
    fi
}

check_ingress_configmaps main "$MAIN"
check_ingress_configmaps pr "$PR"
check_ingress_configmaps pr-sparse "$SPARSE"

exit "$status"
