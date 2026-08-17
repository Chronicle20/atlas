#!/usr/bin/env python3
"""Helper for tools/service-name-guard.sh.

Reads a stream of rendered Kubernetes manifests (YAML multi-doc) on stdin and
prints one violation line per pod-template container missing a correctly
downward-API-sourced SERVICE_NAME. Prints nothing and exits 0 when clean.

Checks every workload kind whose spec carries a `spec.template.spec`
pod template (Deployment, StatefulSet, DaemonSet) — not just Deployment.
The fleet has no StatefulSet/DaemonSet today (confirmed by task-29A review),
but the pod-template shape is identical across all three, so covering them
costs nothing and closes the gap before it's ever hit rather than after.

Usage: kubectl kustomize <overlay> | service-name-guard-check.py <overlay-label>
"""
import sys

import yaml

WORKLOAD_KINDS = {"Deployment", "StatefulSet", "DaemonSet"}
SIDECAR_ALLOWLIST = {"git-sync"}
EXPECTED_FIELD_PATH = "metadata.labels['app']"


def main() -> int:
    overlay = sys.argv[1] if len(sys.argv) > 1 else "<overlay>"
    docs = yaml.safe_load_all(sys.stdin)
    violations = []
    for doc in docs:
        if not doc or doc.get("kind") not in WORKLOAD_KINDS:
            continue
        kind = doc.get("kind")
        name = doc.get("metadata", {}).get("name", "<unknown>")
        containers = (
            doc.get("spec", {})
            .get("template", {})
            .get("spec", {})
            .get("containers", [])
        )
        for c in containers:
            cname = c.get("name", "<unknown>")
            if cname in SIDECAR_ALLOWLIST:
                continue
            env = c.get("env") or []
            entry = next((e for e in env if e.get("name") == "SERVICE_NAME"), None)
            if entry is None:
                violations.append(
                    f"{overlay}: {kind} {name} container {cname}: missing SERVICE_NAME"
                )
                continue
            field_path = (entry.get("valueFrom") or {}).get("fieldRef", {}).get(
                "fieldPath", ""
            )
            if field_path != EXPECTED_FIELD_PATH:
                violations.append(
                    f"{overlay}: {kind} {name} container {cname}: SERVICE_NAME not "
                    f"sourced from {EXPECTED_FIELD_PATH} (entry={entry!r})"
                )
    for v in violations:
        print(v)
    return 1 if violations else 0


if __name__ == "__main__":
    sys.exit(main())
