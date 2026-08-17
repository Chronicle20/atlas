# Slice First; Escalate When Necessary

How to read a large artifact inside an agent or a controller.

**This is a default, not a prohibition.** If semantic correctness requires the
whole document, read the whole document. Under-reading and getting it wrong
costs a fix round, which is far more than any read it saved. The rule is only
that a whole-file read should be a *decision*, not the reflex opening move.

---

## The measurement

Cost is not bytes; it is bytes × the turns that re-read them. A result entering
at turn 5 of a 200-turn agent is re-billed ~195 times; the same result at turn
190 is re-billed 10 times.

That multiplier lands where it hurts most:

> **The median tool result over 12 KB arrives at position 0.10 of its stream,
> and 75% of them arrive in the first quarter.**

So the heavy tail is not merely large — it is placed exactly where it is most
expensive. Measured over one task: 268 results >12 KB carried 26.8% of all tool
bytes from 2.4% of calls, and total carry came to 0.92 GB-turns ≈ 18% of the
task's entire billed input.

The two costliest objects were documents, not code:

| Carry | Reads | Streams | Document |
|---|---|---|---|
| 37.5 MB-turns | 26 | 25 | `service-wiring-recipe.md` (23 KB) |
| 31.8 MB-turns | 48 | 16 | `query-scope-audit.md` (126 KB) |
| 21.5 MB-turns | 155 | — | `processor.go` (for comparison) |

Two files, 74 of 11,102 tool calls, **7.5% of all carry**. Read 26 and 48 times,
nearly always whole, when each agent needed one Pattern section or a handful of
service rows. `prd.md` was read *three* times and outranks `requests.go`, read
105 times, because those three reads landed at turns 4–6 of long sessions.

**The documents are not the problem.** They are the reason 40+ services were
wired consistently, and a thinner recipe would be re-derived by 25 agents at far
greater cost. Change the access pattern; leave the content complete.

---

## The rule

**For any file you expect to exceed ~20 KB, lead with a slice.**

Escalate to a full read when the slice is insufficient — and when you do, say so
in your report, because a document that is repeatedly escalated is a document
that needs restructuring.

`tools/doc-slice.sh` is the accessor:

```sh
# What is in here, and where? Cheap first move on an unfamiliar document.
tools/doc-slice.sh docs/tasks/<task>/service-wiring-recipe.md --outline

# The one section this batch needs.
tools/doc-slice.sh docs/tasks/<task>/service-wiring-recipe.md --section 'Pattern C'

# The rows for the services in this brief — table header preserved.
tools/doc-slice.sh docs/tasks/<task>/query-scope-audit.md \
  --rows atlas-account --rows atlas-buddies

# A needle in an offloaded tool result, rather than re-reading it whole.
tools/doc-slice.sh <session>/tool-results/bd3sc8ctl.txt --grep 'PANIC' --context 3
```

It prints the source path and size with every slice, so escalation is always one
deliberate step away.

## Where each case lands

| Situation | Slice-first move | Escalate when |
|---|---|---|
| One plan task out of a 291 KB `plan.md` | `tools/task-brief.sh <plan> <N>` — the brief IS the extract | the task references a decision recorded elsewhere in the plan |
| Auditing many plan tasks (`plan-adherence-reviewer`) | one `task-brief.sh` extract per task under audit | never read the whole plan 13 times — that was one measured agent |
| A reference recipe | `--outline`, then `--section '<pattern named in the brief>'` | the section cross-references another pattern |
| An audit / result matrix | `--rows <service>` for the services in scope | a row's meaning depends on the document's preamble — the invariants preamble travels with every slice for exactly this reason |
| A review diff | `git diff --stat <range>` first, then `git diff <range> -- <file>` per flagged file | a change's correctness genuinely spans files |
| An offloaded `tool-results/*.txt` | `--grep` / `--lines` from disk | you need the whole log, which is rare |
| A config or routing table | `grep -n` for the entries you need | you are changing its structure, not one entry |
| Source code | targeted `grep -n` / `sed -n` for the symbol, then read the file that owns it | reading a file you are about to edit — do read it |

That last row matters: **targeted read slices are not the problem.** They were
the single largest Bash family in the audited task (3,158 calls) and they are
agents choosing what to look at — semantic work, done well. This document is
about whole-document reads as a discovery reflex, not about reading less.

## Front-load the cheap thing, not the expensive one

If you need both an inventory and a detail, take the inventory first:
`git diff --stat` before hunks, `--outline` before `--section`, `ls` before
`Read`. The inventory is small and tells you which expensive read is actually
required — and it arrives at the position where a large result would have been
worst.

## What already works — do not "improve" it

Build and test output is consistently bounded (`go build ./... 2>&1 | head -60`
and friends): 1,251 invocations produced 0.81 MB in-context, ~650 bytes each.
The tool-result spill mechanism works too — large payloads land as ~1.3 KB stubs
and get sliced from disk. Both are the model this document asks documentation
reads to copy, not costs to trim.
