#!/usr/bin/env bash
# Asserts that deploy/k8s/overlays/pr-sparse addresses the BASELINE's shared
# substrates by the baseline's names — and that it keeps addressing the
# per-deployment ones by its own.
#
# WHY this guard exists: task-232 shipped the sparse overlay on the reading
# that "a sparse env shares the baseline's Kafka/Postgres/Redis" meant "let
# deploy/k8s/base's unsuffixed defaults stand". It does not. Every Atlas
# substrate in the cluster is named for the environment that owns it:
#
#   topics    EVENT_TOPIC_X-main         (overlays/main suffixes all 170)
#   databases atlas-characters-main      (there is no unsuffixed database)
#   redis     main:atlas:<ns>:...        (overlays/main sets ATLAS_ENV=main)
#
# so the unsuffixed name is not the baseline's name — it is a fourth,
# empty namespace nobody publishes to. In atlas-pr-1411 on 2026-08-20 that
# put atlas-login in CrashLoopBackOff (its configuration projection caught
# up instantly on a topic with end-offset 0, published a nil service
# snapshot, and fatally timed out looking for the `timeout` task) while
# atlas-channel sat Running and idle, consuming topics no producer wrote.
# See docs/tasks/task-232-sparse-ephemeral-environments/
# bug-sparse-baseline-scoping.md.
#
# The split this guard protects: ATLAS_ENV names the DEPLOYMENT and must
# stay per-env (KAFKA_CONSUMER_GROUP uniqueness, and libs/atlas-lock's
# `atlas:lock:<ATLAS_ENV>:` lease — a shared lease loses every
# leader-gated sweep to the baseline's pod, the task-200 defect).
# ATLAS_REDIS_ENV names the Redis KEYSPACE and must match the baseline.
#
#   sparse-baseline-scoping-guard.sh   render overlays/pr-sparse and check
#                                      all four invariants; exit 1 naming
#                                      any that fail.
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
BASELINE_TOKEN="PLACEHOLDER_BASELINE_ENVIRONMENT"
SELF_TOKEN="PLACEHOLDER_ATLAS_ENV"

if ! command -v kustomize >/dev/null 2>&1; then
    echo "sparse-baseline-scoping-guard: kustomize not found on PATH" >&2
    exit 1
fi
if ! command -v python3 >/dev/null 2>&1; then
    echo "sparse-baseline-scoping-guard: python3 not found on PATH" >&2
    exit 1
fi

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

kustomize build "$REPO_ROOT/deploy/k8s/overlays/pr-sparse" > "$WORKDIR/pr-sparse.yaml"

BASELINE_TOKEN="$BASELINE_TOKEN" SELF_TOKEN="$SELF_TOKEN" python3 - "$WORKDIR/pr-sparse.yaml" <<'PY'
import os
import sys

import yaml

baseline = "-" + os.environ["BASELINE_TOKEN"]
self_token = os.environ["SELF_TOKEN"]

docs = [d for d in yaml.safe_load_all(open(sys.argv[1])) if d]
status = 0


def fail(msg):
    global status
    print("sparse-baseline-scoping-guard: FAIL - " + msg, file=sys.stderr)
    status = 1


def ok(msg):
    print("sparse-baseline-scoping-guard: PASS - " + msg)


env_cm = next(
    (d for d in docs
     if d.get("kind") == "ConfigMap" and d["metadata"]["name"].startswith("atlas-env")
     and "BOOTSTRAP_SERVERS" in (d.get("data") or {})),
    None,
)
if env_cm is None:
    fail("no atlas-env ConfigMap in the rendered overlay")
    sys.exit(status)

data = env_cm["data"]

# 1. Kafka topics name the baseline's topics.
topics = {k: v for k, v in data.items()
          if k.startswith(("COMMAND_TOPIC_", "EVENT_TOPIC_", "STATUS_EVENT_TOPIC_"))}
if not topics:
    fail("atlas-env ConfigMap carries no topic variables at all")
else:
    bad = sorted(k for k, v in topics.items() if not v.endswith(baseline))
    if bad:
        fail("%d of %d topic vars are not baseline-suffixed, e.g. %s=%s"
             % (len(bad), len(topics), bad[0], topics[bad[0]]))
    else:
        ok("all %d topic vars end with %s" % (len(topics), baseline))

# 2. Redis keyspace is the baseline's.
if data.get("ATLAS_REDIS_ENV") != os.environ["BASELINE_TOKEN"]:
    fail("atlas-env ATLAS_REDIS_ENV is %r, want %r"
         % (data.get("ATLAS_REDIS_ENV"), os.environ["BASELINE_TOKEN"]))
else:
    ok("atlas-env ATLAS_REDIS_ENV is the baseline's environment")

# 3. Postgres databases are the baseline's.
db_names = []
for d in docs:
    if d.get("kind") != "Deployment":
        continue
    for c in d["spec"]["template"]["spec"]["containers"]:
        for e in c.get("env") or []:
            if e.get("name") == "DB_NAME":
                db_names.append((d["metadata"]["name"], e.get("value")))
if not db_names:
    fail("no Deployment carries a DB_NAME env var")
else:
    bad = [(n, v) for n, v in db_names if not str(v).endswith(baseline)]
    if bad:
        fail("%d of %d DB_NAME values are not baseline-suffixed, e.g. %s=%s"
             % (len(bad), len(db_names), bad[0][0], bad[0][1]))
    else:
        ok("all %d DB_NAME values end with %s" % (len(db_names), baseline))

# 4. ATLAS_ENV stays per-deployment. A sparse env that adopted the
#    baseline's ATLAS_ENV would share its leader-election leases.
atlas_envs = set()
for d in docs:
    if d.get("kind") == "ConfigMap" and "ATLAS_ENV" in (d.get("data") or {}):
        atlas_envs.add(d["data"]["ATLAS_ENV"])
    if d.get("kind") != "Deployment":
        continue
    for c in d["spec"]["template"]["spec"]["containers"]:
        for e in c.get("env") or []:
            if e.get("name") == "ATLAS_ENV":
                atlas_envs.add(e.get("value"))
if not atlas_envs:
    fail("nothing in the rendered overlay sets ATLAS_ENV")
elif atlas_envs != {self_token}:
    fail("ATLAS_ENV is set to %s, want only %r (per-deployment)"
         % (sorted(map(repr, atlas_envs)), self_token))
else:
    ok("ATLAS_ENV is %r everywhere it is set (per-deployment)" % self_token)

sys.exit(status)
PY
