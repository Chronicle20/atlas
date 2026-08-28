# Ruling — the `MAKER_RESULT` `FAILED` arm (Task 8 / Task 9)

Durable because `.superpowers/` is gitignored (open item 2) and this binds Task 9,
Task 26, and plan-adherence review.

## Question 1 — is `MakerResultFailedBody` allowed to skip `WithResolvedCode`?

**RULED: yes. Keep the implementation as committed in `8f596fd97`.**

The `FAILED` arm writes no mode field: the client's `nResult > 1` guard returns before the
`nMode` `Decode4`. Two independent agents confirmed against source:

- `resolve.go:66-70` — an absent operations key logs at Error ("will likely cause a client
  crash") and returns the byte `99`. The sentinel is real, not a rumor. Resolving `FAILED`
  would emit that error on every send for a byte the client never reads.
- `dispatcher_lint.go:649-718` — INV-5 is a pure struct-construction reachability check. It
  matches `New…`/composite-literal forms textually anywhere outside the struct's own file. It
  never inspects whether construction passes through `WithResolvedCode`, and never consults the
  YAML operations table. `maker_result_body.go:70` calls `NewMakerResultFailed(result)`
  directly and is matched. INV-5 genuinely passes — it is not being evaded.
- The brief's own interface block (line 127) declares `NewMakerResultFailed(result uint32)`
  with **no `mode` parameter**. The deviation is therefore *required* by the brief, which
  contradicts itself elsewhere by saying every body resolves a const.
- The letter-conforming alternative (`func(m byte)` with `m` unused) is `func(_ byte)`
  (AP-3/INV-2) wearing a name to pass the regex. The committed form is the honest one.

**Precedent:** `guild/clientbound/info.go:41` (`#Info`) is enrolled as an arm of the 39-arm
`CWvsContext::OnGuildResult` family, has **no** YAML operations key, and its body function
`GuildInfoBody` (`guild/operation_body.go:277-284`) constructs it without `WithResolvedCode`.
That exact shape passes `dispatcher-lint` and `operations --check` in the tree today.

## Question 2 — should Task 9's YAML carry a `FAILED` key?

The implementer told me "Task 9 should not emit a `FAILED` key." I did **not** take that at
face value — its stated reasoning was wrong. It argued `ResolveCode` would return the `99`
sentinel, but Task 9 Step 2 generates the tenant operations tables *from that same YAML block*
(`operations.go:139-256`); the dispatcher YAML and the operations table are one artifact. Once
a `FAILED` key exists, `ResolveCode` would find it and return `0`, never `99`.

The conclusion happens to survive its broken reasoning, on different grounds:

- **No gate couples the two directions.** `operations --check` never opens a `.go` file
  (`operations.go:250` parses only the YAMLs). `dispatcher-lint` reads YAML operations tables
  only in `FAM-CAP` (`dispatcher_lint.go:399-478`), which checks the `fname:` has a `run.go`
  case — never individual `key:` rows. `matrix --check` / `fname-doc --check` never reference
  operations keys at all. So a key with no Go consumer, **and** an arm with no key, both pass
  all four gates silently.
- Therefore this is a **cleanliness** decision, not a correctness one. An unconsumed `FAILED: 0`
  would sit in every seed template as dead, misleading config that nothing validates or reads
  back — and `0` is an invented placeholder (plan.md itself calls it one).

**RULED BY THE USER: OMIT the `FAILED` key**, matching the `guild.Info` precedent, with a
comment in `maker_result.yaml` explaining why.

This is a **recorded, sanctioned deviation from plan.md Task 9 Step 1** (brief line 58, which
lists `FAILED` with mode `0` on all eight versions). `plan-adherence-reviewer` must not flag the
absent key as an unimplemented step — point it at this ruling. The four other keys (`CREATE`,
`CREATE_WITH_UPGRADE`, `MONSTER_CRYSTAL`, `DISASSEMBLE`) are unchanged and still required.

## Task 8 review finding — DISMISSED as a scope error

`task-reviewer` returned `CHANGES_REQUIRED`, blocking on "no Encode/Decode round-trip or
byte-fixture tests for the five arms." That obligation is **Task 9's**, not Task 8's:

- Task 8's brief has five steps (structs, body functions, `run.go`, build-and-lint, commit) and
  contains **no** mention of tests, fixtures, or round-trips anywhere in 215 lines.
- Task 9's brief Step 3 is literally "Write the byte fixtures," with expected bytes per arm
  (line 84) including `MakerResultFailed` (line 137, `nResult = 2`, nothing else).

The reviewer compared against Task 7, which did ship its codec and fixtures in one commit — but
the plan deliberately splits this op across two tasks. No fix round for Task 8.

**Obligation transferred:** Task 9 must produce fixtures for **all five** arms plus round-trip
coverage. If Task 9 ships without covering all five, that becomes a genuine blocking finding
there, and it will not be dismissible.
