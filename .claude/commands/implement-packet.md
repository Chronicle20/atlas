---
description: Implement a new packet codec across all applicable versions — derive, model, wire all 9 templates, then verify each cell
argument-hint: <packet/feature description, e.g. monster/clientbound/MobCrcKeyChanged or "the note-delete serverbound op">
---

You are implementing a new packet codec that does not exist yet in
`libs/atlas-packet`.

Arguments: $ARGUMENTS (the packet id or a feature description of the op).

Dispatch the `packet-implementer` agent to do the work. It follows
`docs/packets/IMPLEMENTING_A_PACKET.md` §0–4 verbatim — read it FIRST, in full,
and execute it; do not paraphrase, shortcut, or work from a remembered version
of the rules. That playbook is the single source of truth for the four steps
(Derive → Model + codec → Wire → Verify), the Step-0 "already implemented /
shared-codec wrapper" decision, the GMS-v95.1-IDB-is-truth / distrust-symbols
rule, the `MajorAtLeast` version-gate idiom (never raw `> N`), the
route-all-nine-templates + validator-mandatory-handler rule, the DOM-25
config-resolved mode/message bytes rule, and the no-wire-change-to-an-existing-
version rule. Its Step 4 hands each cell to `packet-verifier`.

Pass the agent the worktree path and branch, and the packet/feature description.

Output: the implemented codec's cells (op × version, old state → new state) plus
the commit SHA(s) and the `matrix --check` / `operations --check` /
`fname-doc --check` / `dispatcher-lint` exit codes, or a precise blocker (which
playbook section, what failed — e.g. an unresolved fname to escalate).
