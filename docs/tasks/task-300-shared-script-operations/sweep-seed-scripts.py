#!/usr/bin/env python3
"""FR-20 data sweep for task-300.

Walks every seeded script JSON under deploy/seed/ and reports the parameter
shapes of the operations whose contract task-300 converges. Parses each
document and visits every {"type": ..., "params": {...}} node, so a rule
nested anywhere in the tree is still counted.

Run from the repository root:

    python3 docs/tasks/task-300-shared-script-operations/sweep-seed-scripts.py
"""

import collections
import glob
import json
import sys

OPS = {
    "spawn_monster",
    "drop_message",
    "send_message",
    "warp",
    "warp_to_map",
}


def classify(value):
    s = str(value)
    if s.lstrip("-").isdigit():
        return "integer"
    if "{" in s:
        return "context-ref"
    return "text"


def walk(node, path, sink):
    if isinstance(node, dict):
        if node.get("type") in OPS and isinstance(node.get("params"), dict):
            sink(node, path)
        for key, value in node.items():
            walk(value, path, sink)
    elif isinstance(node, list):
        for value in node:
            walk(value, path, sink)


def main():
    shapes = collections.Counter()
    examples = {}
    missing_map_id = []
    message_type_keys = collections.Counter()
    cross_map_spawns = []

    def record(node, path):
        op = node["type"]
        params = node["params"]

        for name, value in params.items():
            key = (op, name, classify(value))
            shapes[key] += 1
            examples.setdefault(key, (path, value))

        if op in ("warp", "warp_to_map") and "mapId" not in params:
            missing_map_id.append((path, params))

        if op in ("drop_message", "send_message"):
            message_type_keys[
                (op, params.get("messageType"), params.get("type"))
            ] += 1

        if op == "spawn_monster" and "mapId" in params:
            cross_map_spawns.append((path, params))

    files = sorted(glob.glob("deploy/seed/**/*.json", recursive=True))
    if not files:
        print("no seed files found — run from the repository root", file=sys.stderr)
        return 1

    for path in files:
        try:
            with open(path) as handle:
                document = json.load(handle)
        except (OSError, ValueError) as err:
            print(f"skipping {path}: {err}", file=sys.stderr)
            continue
        walk(document, path, record)

    print(f"scanned {len(files)} seed documents\n")

    print("== parameter shapes (op, param, kind) -> count")
    for key in sorted(shapes):
        sample = "" if key[2] == "integer" else f"  e.g. {examples[key][1]!r}"
        print(f"  {key} -> {shapes[key]}{sample}")

    print("\n== warp / warp_to_map missing mapId")
    if missing_map_id:
        for path, params in missing_map_id:
            print(f"  {path}: {params}")
    else:
        print("  none")

    print("\n== message-type keys (op, messageType, type) -> count")
    for key in sorted(message_type_keys, key=str):
        print(f"  {key} -> {message_type_keys[key]}")

    print("\n== spawn_monster with an explicit mapId (OQ-3 instance rule)")
    if cross_map_spawns:
        for path, params in cross_map_spawns:
            print(f"  {path}: {params}")
    else:
        print("  none")

    return 0


if __name__ == "__main__":
    sys.exit(main())
