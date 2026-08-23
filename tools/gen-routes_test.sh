#!/usr/bin/env sh
set -eu
REPO_ROOT="$(git rev-parse --show-toplevel)"
fail() { echo "FAIL: $*" >&2; exit 1; }

# 1. Every upstream in the generated routes uses a per-service NS_ variable.
if grep -q '\${POD_NAMESPACE}' "$REPO_ROOT/deploy/k8s/base/routes.conf.template.generated"; then
    fail "routes still reference \${POD_NAMESPACE}; expected per-service NS_ variables"
fi

# 2. Every NS_ variable referenced in the routes is defined in the env block.
for v in $(grep -o 'NS_[A-Z0-9_]*' "$REPO_ROOT/deploy/k8s/base/routes.conf.template.generated" | sort -u); do
    grep -q "name: $v" "$REPO_ROOT/deploy/k8s/base/ns-vars.generated.yaml" \
        || fail "$v referenced in routes but not defined in ns-vars.generated.yaml"
done

# 3. Every defined variable defaults to POD_NAMESPACE (NFR-7).
grep -c 'value: \$(POD_NAMESPACE)' "$REPO_ROOT/deploy/k8s/base/ns-vars.generated.yaml" >/dev/null \
    || fail "ns-vars defaults are not \$(POD_NAMESPACE)"

# 4. --check is clean against the checked-in output.
"$REPO_ROOT/tools/gen-routes.sh" --check || fail "gen-routes drift"

# 5. The map-instance jukebox subresource has a route to atlas-maps (bug-jukebox-replay-route-404).
awk '/^location ~ \^\/api\/worlds\/\[\^\/\]\+\/channels\/\[\^\/\]\+\/maps\/\[\^\/\]\+\/instances\/\[\^\/\]\+\/jukebox/{f=1} f{print; if (/}/) exit}' \
    "$REPO_ROOT/deploy/k8s/base/routes.conf.template.generated" | grep -q 'atlas-maps' \
    || fail "no map-instance jukebox location routed to atlas-maps in routes.conf.template.generated"

echo "PASS"
