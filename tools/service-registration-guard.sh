#!/usr/bin/env bash
# service-registration-guard.sh — cross-checks every hand-maintained list a
# service must be registered in (see docs/adding-a-new-service.md).
#
# Source of truth: .github/config/services.json (go-services) plus each
# service's base manifest (container name, DB_NAME). Fails when a service is
# missing from docker-bake.hcl, go.work, the k8s base kustomization, either
# overlay's images: block, the main ATLAS_ENV patch, the DB-name suffix
# patches, ATLAS_DB_NAMES, or tools/db-bootstrap.sh — and when a base
# env-configmap key is dropped by an overlay's replace-mode configMapGenerator
# or a patch document targets a container name that doesn't exist in the base
# manifest. These are exactly the silent-failure traps that let atlas-mts ship
# half-wired (crash-looping DB_NAME, unpinned :latest image, unsuffixed Kafka
# topics).
#
# Usage: tools/service-registration-guard.sh   (from the repo root)
# Exit 0 = clean, exit 1 = violations listed on stdout.

set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

python3 - <<'PYEOF'
import glob, json, os, re, sys

violations = []

def fail(msg):
    violations.append(msg)

# Go-services that intentionally have no k8s deployment yet. Every name here
# must be justified; remove the entry the moment the service gets a manifest.
ALLOW_NO_DEPLOYMENT = {
    'atlas-families',   # scaffolded service, not yet deployed
    'atlas-marriages',  # scaffolded service, not yet deployed
}

services = json.load(open('.github/config/services.json'))['services']
go_services = [s for s in services if s['type'] == 'go-service']

bake = open('docker-bake.hcl').read()
gowork = open('go.work').read()
base_kust = open('deploy/k8s/base/kustomization.yaml').read()
main_kust = open('deploy/k8s/overlays/main/kustomization.yaml').read()
pr_kust = open('deploy/k8s/overlays/pr/kustomization.yaml').read()
main_env_patch = open('deploy/k8s/overlays/main/patches/atlas-env-env.yaml').read()
main_db_patch = open('deploy/k8s/overlays/main/patches/db-name-suffix.yaml').read()
pr_db_patch = open('deploy/k8s/overlays/pr/patches/db-name-suffix.yaml').read()
pr_cg_patch = open('deploy/k8s/overlays/pr/patches/consumer-group-env.yaml').read()
db_bootstrap = open('tools/db-bootstrap.sh').read()

m = re.search(r'ATLAS_DB_NAMES=([^\n]+)', pr_kust)
atlas_db_names = set(m.group(1).split()) if m else set()

def base_manifest(name):
    return f'deploy/k8s/base/{name}.yaml'

def parse_base(name):
    """Return (container_names, db_name) from a base manifest.

    Base manifests are single-app-container (sidecars come from kustomize
    components), but indentation styles vary — take the first `- name:`
    after each `containers:` line, whatever its indent.
    """
    lines = open(base_manifest(name)).read().split('\n')
    containers = []
    for i, ln in enumerate(lines):
        if re.match(r'^\s+containers:\s*$', ln):
            for follower in lines[i + 1:]:
                m = re.match(r'^\s+- name: (\S+)$', follower)
                if m:
                    containers.append(m.group(1))
                    break
    db = re.search(r'name: DB_NAME\s*\n\s*value: "([^"]+)"', '\n'.join(lines))
    return containers, (db.group(1) if db else None)

def patch_docs(txt):
    """Yield (deployment_name, [container_names]) per document in a patch file."""
    for doc in txt.split('\n---'):
        dep = re.search(r'^\s*name: (atlas-\S+)$', doc, re.M)
        if not dep:
            continue
        yield dep.group(1), re.findall(r'^        - name: (\S+)$', doc, re.M)

def has_doc_for(txt, dep, container):
    return any(d == dep and container in cs for d, cs in patch_docs(txt))

# ---- per-service checks --------------------------------------------------
for svc in go_services:
    name, module, image = svc['name'], svc['module_path'], svc['docker_image']

    if f'"{name}"' not in bake:
        fail(f'{name}: missing from docker-bake.hcl go_services list')
    if f'./{module}' not in gowork:
        fail(f'{name}: module ./{module} missing from go.work')

    if not os.path.exists(base_manifest(name)):
        if name not in ALLOW_NO_DEPLOYMENT:
            fail(f'{name}: no base manifest deploy/k8s/base/{name}.yaml '
                 f'(add it, or allowlist in {sys.argv[0] if sys.argv else "this guard"})')
        continue
    if name in ALLOW_NO_DEPLOYMENT:
        fail(f'{name}: has a base manifest but is still in ALLOW_NO_DEPLOYMENT — remove the allowlist entry')

    containers, db_name = parse_base(name)
    if not containers:
        fail(f'{name}: could not parse container name from base manifest')
        continue
    container = containers[0]

    if f'- {name}.yaml' not in base_kust:
        fail(f'{name}: missing from deploy/k8s/base/kustomization.yaml resources')

    for label, kust in (('main', main_kust), ('pr', pr_kust)):
        if f'name: {image}\n' not in kust:
            fail(f'{name}: missing images: entry for {image} in {label} overlay '
                 f'(the bump workflow silently skips absent entries -> runs :latest forever)')

    if not has_doc_for(main_env_patch, name, container):
        fail(f'{name}: no ATLAS_ENV patch document for container "{container}" in '
             f'deploy/k8s/overlays/main/patches/atlas-env-env.yaml')

    if db_name:
        if f'value: "{db_name}-main"' not in main_db_patch or not has_doc_for(main_db_patch, name, container):
            fail(f'{name}: main overlay db-name-suffix.yaml missing '
                 f'DB_NAME="{db_name}-main" for container "{container}"')
        if f'value: "{db_name}-PLACEHOLDER_ATLAS_ENV"' not in pr_db_patch or not has_doc_for(pr_db_patch, name, container):
            fail(f'{name}: pr overlay db-name-suffix.yaml missing '
                 f'DB_NAME="{db_name}-PLACEHOLDER_ATLAS_ENV" for container "{container}"')
        if db_name not in atlas_db_names:
            fail(f'{name}: DB base name "{db_name}" missing from ATLAS_DB_NAMES in the pr overlay '
                 f'(drives ephemeral-env DB create AND drop)')
        if not re.search(rf'^\s+{re.escape(db_name)}$', db_bootstrap, re.M):
            fail(f'{name}: DB base name "{db_name}" missing from the DBS list in tools/db-bootstrap.sh')

    # Services that declare a Kafka consumer group must be in the generated
    # PR consumer-group patch (re-run gen-consumer-group-patch.sh).
    mains = glob.glob(f'services/{name}/atlas.com/*/main.go')
    if mains:
        main_go = open(mains[0]).read()
        if re.search(r'consumergroup\.Resolve\("|consumerGroupIdTemplate *= *"', main_go):
            if not has_doc_for(pr_cg_patch, name, container):
                fail(f'{name}: declares a consumer group in main.go but has no document in '
                     f'deploy/k8s/overlays/pr/patches/consumer-group-env.yaml '
                     f'(re-run deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh)')

# ---- reverse check: stale bake entries ------------------------------------
known = {s['name'] for s in go_services}
for bake_name in re.findall(r'^\s*"(atlas-[a-z0-9-]+)",?$', bake, re.M):
    if bake_name not in known:
        fail(f'docker-bake.hcl lists "{bake_name}" which is not a go-service in services.json')

# ---- patch docs must target real base containers ---------------------------
for patch_path, txt in (
    ('deploy/k8s/overlays/main/patches/atlas-env-env.yaml', main_env_patch),
    ('deploy/k8s/overlays/main/patches/db-name-suffix.yaml', main_db_patch),
    ('deploy/k8s/overlays/pr/patches/db-name-suffix.yaml', pr_db_patch),
    ('deploy/k8s/overlays/pr/patches/consumer-group-env.yaml', pr_cg_patch),
):
    for dep, cs in patch_docs(txt):
        if not os.path.exists(base_manifest(dep)):
            fail(f'{patch_path}: document targets "{dep}" which has no base manifest')
            continue
        base_containers, _ = parse_base(dep)
        for c in cs:
            if c not in base_containers:
                fail(f'{patch_path}: document for "{dep}" targets container "{c}" '
                     f'but base manifest containers are {base_containers} '
                     f'(strategic-merge would ADD a broken container instead of patching)')

# ---- configmap key parity ---------------------------------------------------
base_keys = set(re.findall(r'^  ([A-Z][A-Z_0-9]+):', open('deploy/k8s/base/env-configmap.yaml').read(), re.M))
for label, kust in (('main', main_kust), ('pr', pr_kust)):
    overlay_keys = set(re.findall(r'^      - ([A-Z][A-Z_0-9]+)=', kust, re.M))
    for key in sorted(base_keys - overlay_keys):
        fail(f'{label} overlay: base env-configmap key {key} not re-listed in the '
             f'replace-mode configMapGenerator (the key will be ABSENT in that env; '
             f'Kafka topic vars then silently fall back to unsuffixed names)')

# ----------------------------------------------------------------------------
if violations:
    print(f'service-registration-guard: {len(violations)} violation(s)\n')
    for v in violations:
        print(f'  FAIL {v}')
    print('\nSee docs/adding-a-new-service.md for what each list is and how to fix.')
    sys.exit(1)
print('service-registration-guard: clean')
PYEOF
