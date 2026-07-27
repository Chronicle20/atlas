---
description: Walk the VERIFYING_A_PACKET playbook for one packet × version — promote a coverage-matrix cell to verified
argument-hint: <packet id, e.g. buddy/clientbound/Invite> <version key, e.g. gms_v83>
---

You are verifying one packet × version cell of the packet coverage matrix.

Arguments: $ARGUMENTS (packet id + version key).

**Follow `docs/packets/audits/VERIFYING_A_PACKET.md` §0–10 verbatim.** Read it
FIRST, in full, and execute it — do not paraphrase, shortcut, or work from a
remembered version of the rules. That playbook is the single source of truth for
the procedure, the Verification-Over-Memory rule, IDA-instance resolution, the
wire-divergence-is-its-own-commit rule, the commit grouping, the
`matrix --check` hard gate, and mode-driven fixture requirements.

Output: the promoted cell (op × version, old state → new state) plus the
commit SHA, or a precise blocker (which playbook section, what failed).
