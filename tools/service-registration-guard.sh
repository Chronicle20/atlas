#!/usr/bin/env bash
# service-registration-guard.sh — cross-checks every hand-maintained list a
# service must be registered in (see docs/adding-a-new-service.md).
#
# Source of truth: .github/config/services.json (go-services) plus each
# service's base manifest (container name, DB_NAME). Fails when a service is
# missing from docker-bake.hcl, go.work, the k8s base kustomization, either
# overlay's images: block, the main ATLAS_ENV patch, the DB-name suffix
# patches, ATLAS_DB_NAMES, or tools/db-bootstrap.sh — and when a base
# env-configmap key is dropped by the overlay's replace-mode atlas-env
# configMapGenerator or a patch document targets a container name that doesn't
# exist in the base manifest. These are exactly the silent-failure traps that
# let atlas-mts ship half-wired (crash-looping DB_NAME, unpinned :latest image,
# unsuffixed Kafka topics).
#
# Parsing is structured (PyYAML), not substring/regex over raw text, so
# commented-out entries do NOT count as present, env VALUES are checked (not
# just that a patch document exists), and the atlas-env key-parity check is
# scoped to the atlas-env generator alone (not pooled with atlas-db-names /
# atlas-pr-bootstrap-tenant literals). If PyYAML is unavailable or a core list
# parses empty, the guard fails closed rather than reporting a false clean.
#
# Usage: tools/service-registration-guard.sh   (from the repo root)
# Exit 0 = clean, 1 = violations, 2 = cannot verify (fail closed).

set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

python3 - <<'PYEOF'
import glob, json, os, re, sys

try:
    import yaml
except ImportError:
    print('service-registration-guard: FATAL — PyYAML not importable; cannot '
          'verify registration lists. Failing closed.', file=sys.stderr)
    sys.exit(2)

violations = []
def fail(msg):
    violations.append(msg)

# Go-services that intentionally have no k8s deployment yet. Every name here
# must be justified; remove the entry the moment the service gets a manifest.
ALLOW_NO_DEPLOYMENT = {
    'atlas-families',   # scaffolded service, not yet deployed
    'atlas-marriages',  # scaffolded service, not yet deployed
}

# ---- loaders --------------------------------------------------------------
def load(path):
    with open(path) as f:
        return yaml.safe_load(f)

def load_all(path):
    with open(path) as f:
        return [d for d in yaml.safe_load_all(f) if isinstance(d, dict)]

def strip_line_comments(text, markers=('#', '//')):
    """Drop whole-line comments so a commented-out entry is not seen as present
    by the plain-text membership checks over HCL / go.work."""
    keep = []
    for ln in text.splitlines():
        s = ln.lstrip()
        if any(s.startswith(m) for m in markers):
            continue
        keep.append(ln)
    return '\n'.join(keep)

def base_manifest(name):
    return f'deploy/k8s/base/{name}.yaml'

def base_info(name):
    """(container_names, db_name) from a base manifest's Deployment doc(s)."""
    containers, db = [], None
    for d in load_all(base_manifest(name)):
        if d.get('kind') != 'Deployment':
            continue
        spec = (((d.get('spec') or {}).get('template') or {}).get('spec') or {})
        for c in spec.get('containers') or []:
            containers.append(c.get('name'))
            for e in c.get('env') or []:
                if isinstance(e, dict) and e.get('name') == 'DB_NAME':
                    db = e.get('value')
    return containers, db

def patch_index(path):
    """deployment_name -> {container_name -> {env_key: env_value}} for a
    strategic-merge patch file (multi-doc)."""
    idx = {}
    for d in load_all(path):
        if d.get('kind') != 'Deployment':
            continue
        dep = (d.get('metadata') or {}).get('name')
        if not dep:
            continue
        spec = (((d.get('spec') or {}).get('template') or {}).get('spec') or {})
        cmap = idx.setdefault(dep, {})
        for c in spec.get('containers') or []:
            env = {e['name']: e.get('value')
                   for e in (c.get('env') or [])
                   if isinstance(e, dict) and 'name' in e}
            cmap.setdefault(c.get('name'), {}).update(env)
    return idx

def kust_images(k):
    return {i.get('name') for i in (k.get('images') or []) if isinstance(i, dict)}

def generator_literals(k, gen_name):
    for g in k.get('configMapGenerator') or []:
        if g.get('name') == gen_name:
            return g.get('literals') or []
    return None

def atlas_env_keys(k):
    return {l.split('=', 1)[0] for l in (generator_literals(k, 'atlas-env') or [])}

# ---- parse everything up front --------------------------------------------
services = json.load(open('.github/config/services.json'))['services']
go_services = [s for s in services if s['type'] == 'go-service']

bake_code = strip_line_comments(open('docker-bake.hcl').read())
gowork_code = strip_line_comments(open('go.work').read())
db_bootstrap = open('tools/db-bootstrap.sh').read()

base_kust = load('deploy/k8s/base/kustomization.yaml')
main_kust = load('deploy/k8s/overlays/main/kustomization.yaml')
pr_kust = load('deploy/k8s/overlays/pr/kustomization.yaml')

base_resources = set(base_kust.get('resources') or [])
main_images = kust_images(main_kust)
pr_images = kust_images(pr_kust)
main_env_keys = atlas_env_keys(main_kust)
pr_env_keys = atlas_env_keys(pr_kust)
base_cm_keys = set((load('deploy/k8s/base/env-configmap.yaml').get('data') or {}).keys())

atlas_db_names = set()
for l in generator_literals(pr_kust, 'atlas-db-names') or []:
    if l.startswith('ATLAS_DB_NAMES='):
        atlas_db_names = set(l.split('=', 1)[1].split())

PATCH_FILES = {
    'main atlas-env-env': 'deploy/k8s/overlays/main/patches/atlas-env-env.yaml',
    'main db-name-suffix': 'deploy/k8s/overlays/main/patches/db-name-suffix.yaml',
    'pr db-name-suffix': 'deploy/k8s/overlays/pr/patches/db-name-suffix.yaml',
    'pr consumer-group-env': 'deploy/k8s/overlays/pr/patches/consumer-group-env.yaml',
}
patch_idx = {k: patch_index(v) for k, v in PATCH_FILES.items()}
main_env_idx = patch_idx['main atlas-env-env']
main_db_idx = patch_idx['main db-name-suffix']
pr_db_idx = patch_idx['pr db-name-suffix']
pr_cg_idx = patch_idx['pr consumer-group-env']

# ---- fail-closed sanity: a broken parse must not look like a clean tree ----
sanity = []
if not go_services:                 sanity.append('no go-services in services.json')
if not base_cm_keys:                sanity.append('base env-configmap has no data keys')
if not main_env_keys:               sanity.append('main overlay atlas-env generator has no literals')
if not pr_env_keys:                 sanity.append('pr overlay atlas-env generator has no literals')
if not atlas_db_names:              sanity.append('ATLAS_DB_NAMES parsed empty')
if not (main_images and pr_images): sanity.append('an overlay images: list parsed empty')
if not base_resources:              sanity.append('base kustomization resources parsed empty')
if sanity:
    print('service-registration-guard: FATAL — parse sanity failed (failing closed):')
    for s in sanity:
        print(f'  {s}')
    sys.exit(2)

# Mirror gen-consumer-group-patch.sh's own detection: a service "declares a
# consumer group" iff its main.go matches either supported form. Keeping this
# identical to the generator means the guard's notion of "needs a doc" tracks
# the generator's notion of "emits a doc".
CG_DECL = re.compile(
    r'consumerGroupId.*=.*consumergroup\.Resolve\("[^"]+"\)'
    r'|consumerGroupId(?:Template)? *= *"')

# ---- per-service checks ----------------------------------------------------
for svc in go_services:
    name, module, image = svc['name'], svc['module_path'], svc['docker_image']

    if f'"{name}"' not in bake_code:
        fail(f'{name}: missing from docker-bake.hcl go_services list')
    if f'./{module}' not in gowork_code:
        fail(f'{name}: module ./{module} missing from go.work')

    if not os.path.exists(base_manifest(name)):
        if name not in ALLOW_NO_DEPLOYMENT:
            fail(f'{name}: no base manifest {base_manifest(name)} '
                 f'(add it, or add to ALLOW_NO_DEPLOYMENT with justification)')
        continue
    if name in ALLOW_NO_DEPLOYMENT:
        fail(f'{name}: has a base manifest but is still in ALLOW_NO_DEPLOYMENT — '
             f'remove the allowlist entry')

    containers, db_name = base_info(name)
    if not containers or not containers[0]:
        fail(f'{name}: could not parse a container name from {base_manifest(name)}')
        continue
    container = containers[0]

    if f'{name}.yaml' not in base_resources:
        fail(f'{name}: missing from deploy/k8s/base/kustomization.yaml resources')

    for label, images in (('main', main_images), ('pr', pr_images)):
        if image not in images:
            fail(f'{name}: missing images: entry for {image} in {label} overlay '
                 f'(the bump workflow silently skips absent entries -> :latest forever)')

    # ATLAS_ENV: document must exist AND set ATLAS_ENV=main for this container.
    env = main_env_idx.get(name, {}).get(container)
    if env is None:
        fail(f'{name}: no ATLAS_ENV patch document for container "{container}" in '
             f'{PATCH_FILES["main atlas-env-env"]}')
    elif env.get('ATLAS_ENV') != 'main':
        fail(f'{name}: main atlas-env-env.yaml sets ATLAS_ENV={env.get("ATLAS_ENV")!r} '
             f'for container "{container}", expected "main"')

    if db_name:
        want_main = f'{db_name}-main'
        de = main_db_idx.get(name, {}).get(container)
        if de is None:
            fail(f'{name}: no db-name-suffix document for container "{container}" in '
                 f'{PATCH_FILES["main db-name-suffix"]}')
        elif de.get('DB_NAME') != want_main:
            fail(f'{name}: main db-name-suffix.yaml sets DB_NAME={de.get("DB_NAME")!r} '
                 f'for container "{container}", expected "{want_main}"')

        want_pr = f'{db_name}-PLACEHOLDER_ATLAS_ENV'
        pe = pr_db_idx.get(name, {}).get(container)
        if pe is None:
            fail(f'{name}: no db-name-suffix document for container "{container}" in '
                 f'{PATCH_FILES["pr db-name-suffix"]}')
        elif pe.get('DB_NAME') != want_pr:
            fail(f'{name}: pr db-name-suffix.yaml sets DB_NAME={pe.get("DB_NAME")!r} '
                 f'for container "{container}", expected "{want_pr}"')

        if db_name not in atlas_db_names:
            fail(f'{name}: DB base name "{db_name}" missing from ATLAS_DB_NAMES in the pr '
                 f'overlay (drives ephemeral-env DB create AND drop)')
        if not re.search(rf'^\s+{re.escape(db_name)}$', db_bootstrap, re.M):
            fail(f'{name}: DB base name "{db_name}" missing from the DBS list in '
                 f'tools/db-bootstrap.sh')

    # Services that declare a Kafka consumer group must have a generated PR
    # consumer-group document (re-run gen-consumer-group-patch.sh).
    mains = glob.glob(f'services/{name}/atlas.com/*/main.go')
    if mains and CG_DECL.search(open(mains[0]).read()):
        if pr_cg_idx.get(name, {}).get(container) is None:
            fail(f'{name}: declares a consumer group in main.go but has no document in '
                 f'{PATCH_FILES["pr consumer-group-env"]} '
                 f'(re-run deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh)')

# ---- reverse check: stale docker-bake entries ------------------------------
known = {s['name'] for s in go_services}
for bake_name in re.findall(r'^\s*"(atlas-[a-z0-9-]+)",?\s*$', bake_code, re.M):
    if bake_name not in known:
        fail(f'docker-bake.hcl lists "{bake_name}" which is not a go-service in services.json')

# ---- patch documents must target real base containers ----------------------
for label, idx in patch_idx.items():
    path = PATCH_FILES[label]
    for dep, cmap in idx.items():
        if not os.path.exists(base_manifest(dep)):
            fail(f'{path}: document targets "{dep}" which has no base manifest')
            continue
        base_containers, _ = base_info(dep)
        for c in cmap:
            if c not in base_containers:
                fail(f'{path}: document for "{dep}" targets container "{c}" but base '
                     f'manifest containers are {base_containers} (strategic-merge would '
                     f'ADD a broken container instead of patching)')

# ---- atlas-env configmap key parity (scoped to the atlas-env generator) -----
for label, overlay_keys in (('main', main_env_keys), ('pr', pr_env_keys)):
    for key in sorted(base_cm_keys - overlay_keys):
        fail(f'{label} overlay: base env-configmap key {key} not re-listed in the '
             f'atlas-env configMapGenerator (behavior: replace drops it -> the key is '
             f'ABSENT in that env; Kafka topic vars then fall back to unsuffixed names)')

# ----------------------------------------------------------------------------
if violations:
    print(f'service-registration-guard: {len(violations)} violation(s)\n')
    for v in violations:
        print(f'  FAIL {v}')
    print('\nSee docs/adding-a-new-service.md for what each list is and how to fix.')
    sys.exit(1)
print('service-registration-guard: clean')
PYEOF
