---
description: Walk the VERIFYING_A_PACKET playbook for one packet × version — promote a coverage-matrix cell to verified
argument-hint: <packet id, e.g. buddy/clientbound/Invite> <version key, e.g. gms_v83>
---

You are verifying one packet × version cell of the packet coverage matrix.
Follow `docs/packets/audits/VERIFYING_A_PACKET.md` step by step. Read it FIRST.

Arguments: $ARGUMENTS (packet id + version key).

Non-negotiable rules:
1. **Verification Over Memory** — every byte in the fixture must trace to a
   decompile line you obtained in this session (or a checked-in export entry).
   If you cannot resolve a read order, STOP and report the cell as blocked;
   never fabricate from MapleStory knowledge.
2. Resolve the IDA instance by its loaded IDB, never by hardcoded port
   (list_instances → select_instance).
3. A wire divergence found in step 4 is its own commit (fix + test) BEFORE the
   verification commit.
4. The three artifacts land together in one commit: byte-test (with
   `packet-audit:verify` marker), evidence YAML, regenerated
   STATUS.md/status.json. `packet-audit matrix --check` must introduce no
   new problems: zero orphan/dangling/stale/drift lines for your packet and
   no conflict-count increase (exit 0 once the pre-existing seed-conflict
   backlog is gone — see the playbook's note on `--check` exit codes).
5. If the packet is tier-1 mode-driven, one fixture per mode (the registry
   entry and audit report's dispatch selectors enumerate the modes).

Output: the promoted cell (op × version, old state → new state) plus the
commit SHA, or a precise blocker (which playbook step, what failed).
