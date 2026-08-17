# Reverse Engineering / IDA

This document owns the mechanics of confirming the target version, resolving
an IDA session, and looking up functions via `func_query`. It is deliberately
narrow — not an RE tutorial.

## Confirm the target version first

Before investigating any bug, confirm the exact server/tenant version the IDB
or instance under investigation matches. The wrong version sends the whole
investigation down the wrong path.

## Session resolution

`select_instance(port)` and port-based selection are dead (since task-138).
Resolve the session from `idb_list` by binary **name** and pass it as the
`database` parameter to subsequent calls.

## Load the tool schemas in one turn

The IDA tool set is knowable from the task type, so load it once at the start
rather than one schema per unit of work:

```text
ToolSearch: select:mcp__ida-pro__idb_list,mcp__ida-pro__func_query,mcp__ida-pro__decompile,mcp__ida-pro__xrefs_to,mcp__ida-pro__insn_query
```

Measured: one session spent 4 `ToolSearch` turns for 13 MCP calls, another 5
turns for 5 calls — one schema-loading turn per unit of work, each costing a
full context re-read.

## Function lookup

Use the `func_query` tool with `name_regex` — the documented method. Do not
improvise alternate lookup approaches. See the IDA-MCP notes in project
memory for the current API.

## Bound the result, not just the query

When the question is a read order or a call site, prefer `func_query` and a
targeted `xrefs_to` over a full `decompile`, and give `insn_query` a narrow
address range. A decompiled function body and a raw instruction dump are both
*bounded-window* requests answered with an unbounded result.

Measured: one `decompile` (28.1 KB) plus one `insn_query` range (24.7 KB)
landed at turns 30–31 of a 148-turn session ≈ 1.5M tokens, 5.5% of that
session — when the useful portion was the read order of one packet. If you do
need a large body, `tools/doc-slice.sh --grep` over a spilled result beats
re-reading it (see [`docs/slice-first.md`](slice-first.md)).

## Deriving a packet field order

See [`docs/packets/PROCESS.md`](packets/PROCESS.md), the dominant consumer of
these mechanics.
