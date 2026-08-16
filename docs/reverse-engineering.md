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

## Function lookup

Use the `func_query` tool with `name_regex` — the documented method. Do not
improvise alternate lookup approaches. See the IDA-MCP notes in project
memory for the current API.

## Deriving a packet field order

See [`docs/packets/PROCESS.md`](packets/PROCESS.md), the dominant consumer of
these mechanics.
