# Task 8 review — MAKER_RESULT dispatcher family

Commit range: `f712ada30..8f596fd97` (single commit `8f596fd97`; `4efbdd497`
in the nominal range is a docs-only artifact commit, not part of Task 8, and
is excluded from this review).

Brief: `.superpowers/sdd/plan/task-8-brief.md`.
Implementer report: `.superpowers/sdd/plan/task-8-report.md` — **this file
does not exist** in the worktree (`find` for `task-8-report.md` under
`task-285` returns nothing). The only prose record of the implementer's
reasoning is the code comments themselves
(`libs/atlas-packet/character/maker_result_body.go:53-66`,
`libs/atlas-packet/character/clientbound/maker_result.go:13-37,361-368`) and
the commit message. Recorded under Not evaluable; not fatal, since the
central deviation the report would explain is in fact documented in the
committed code.

## Files changed

```
docs/packets/audits/STATUS.md                      |   2 +-
docs/packets/audits/status.json                    |   2 +-
libs/atlas-packet/character/clientbound/maker_result.go | 395 +++++++++
libs/atlas-packet/character/maker_result_body.go   |  73 +++
tools/packet-audit/cmd/run.go                      |  25 +++
```
(plus unrelated `docs/tasks/.../reviews/*.md` and `agent-ledger.tsv`, which
are prior review artifacts carried by the `4efbdd497` docs commit, not by
`8f596fd97`.)

## 1. Central question — `MakerResultFailedBody`'s deviation from `WithResolvedCode`

**Ruling: the deviation is correct, and it is not really a deviation — the
brief's own signature at line 127 requires it.**

1. **`ResolveCode` behavior confirmed** (`libs/atlas-packet/resolve.go:55-88`).
   When `codes[key]` (i.e. the "operations"→"FAILED" entry) is absent,
   `ResolveCode` logs `l.Errorf("Code [%s] not configured in property [%s].
   Defaulting to 99 which will likely cause a client crash.", ...)` and
   returns `99` (`resolve.go:68-72`). `WithResolvedCode`
   (`resolve.go:41-48`) calls `factory(mode)` unconditionally — there is no
   short-circuit on the sentinel. So routing `FAILED` through
   `WithResolvedCode` with no configured key would log an error-level "will
   likely cause a client crash" line on every single send, for a byte
   (`m.mode`) that `MakerResultFailed.Encode` never writes
   (`maker_result.go:383-389` writes only `m.result`). The implementer's
   claim is exactly what the code does — verified, not just narrated.

2. **Brief-internal consistency.** Brief line 127:
   `func NewMakerResultFailed(result uint32) MakerResultFailed` — no `mode`
   parameter, in contrast to all four other constructors which lead with
   `mode byte` (lines 119, 121, 123, 125; INV-2 requires this). Line 130
   confirms: *"`MakerResultFailed` takes no `mode` — the client stops
   reading at `nResult > 1`, so that arm writes no mode field."* A body
   function that calls `WithResolvedCode`'s `factory func(byte)
   packet.Encoder` **must** produce a constructor whose first argument is
   that resolved byte (INV-2/INV-3's own contract, `resolve.go:41`). Since
   the brief's own `NewMakerResultFailed` signature has no such parameter,
   `MakerResultFailedBody` calling `WithResolvedCode(...)` and discarding the
   resolved byte would itself be the `func(_ byte)` shape the brief bans at
   line 169 ("`func(_ byte)` is banned"). The brief is internally
   inconsistent if read as "every body function must call
   `WithResolvedCode`," and the implementer resolved that inconsistency the
   only way the brief's own signature permits: construct directly.
   Verdict: **required by the brief, not a departure from it.**

3. **INV-5 reachability for `FAILED` — genuinely satisfied, not superficially.**
   Traced `checkINV5Orphans` (`tools/packet-audit/cmd/dispatcher_lint.go:649-718`):
   it greps every non-def, non-test `.go` file under the usage roots for a
   `New<Struct>(` call or a `<Struct>{` composite literal. It has no
   awareness of `WithResolvedCode` at all — it only cares whether the
   constructor is invoked from outside its own file.
   `maker_result_body.go:70` calls `clientbound.NewMakerResultFailed(result)`
   directly, which matches the regex `\bNewMakerResultFailed\s*\(`
   unconditionally. Ran the gate against the committed tree (see §"Gates
   re-run" below): `dispatcher-lint: clean`. INV-5 holds for real, not by
   linter blind spot.

   Also checked that neither INV-2(b) (`func(_ byte) packet.Encoder`
   pattern, `dispatcher_lint.go:851-860`) nor INV-3 (selector-param-by-name
   or selector-flows-into-`WithResolvedCode`-key,
   `dispatcher_lint.go:862-900`) fire on `MakerResultFailedBody`: it has
   exactly one parameter (`result uint32`), matching neither pattern, and it
   never calls `WithResolvedCode` at all, so there is no key-flow site to
   flag. The gate is not being gamed by the omission — it simply doesn't
   apply to a function that structurally can't produce a discarded-mode
   violation.

## 2. Also verified

- **INV-1 (shared code, not a shared struct with two `#`-entries).**
  `MakerResultCreate` (`maker_result.go:125-184`) and
  `MakerResultCreateWithUpgrade` (`maker_result.go:192-251`) are two distinct
  struct declarations, each with its own constructor, accessors,
  `Operation()`, `String()`, `Encode`, `Decode`, and its own
  `packet-audit:fname` marker (`#Create` / `#CreateWithUpgrade`). Both call
  the shared unexported `writeMakerCreateBody`/`readMakerCreateBody`
  (`maker_result.go:72-119`, called at `173/182` and `240/249`). This is
  shared *code*, and the two `#`-entries map to two distinct `resolvedStruct`
  records for `dispatcher-lint`'s purposes — INV-1 holds.

- **Encode/Decode symmetry, field for field, all five arms** — read
  `maker_result.go` in full and compared each `Encode`/`Decode` pair:
  - `Create`/`CreateWithUpgrade` (`:168-184`, `:235-251`): `nResult`,
    `nMode`, then `writeMakerCreateBody`/`readMakerCreateBody`
    (`:72-119`) — identical read/write order on both sides, including the
    **inverted** `noItemAwarded` (`w.WriteBool(noItemAwarded); if
    !noItemAwarded { ... }` mirrored by `noItemAwarded = r.ReadBool(); if
    !noItemAwarded { ... }`, `:73-77` / `:96-100`) and the
    **`catalystUsed`-gated** trailing id (`:87-90` / `:113-116`) — both
    gates are on the same boolean, in the same position, on both sides.
  - `MonsterCrystal` (`:279-297`): two unconditional `Decode4`s, matches.
  - `Disassemble` (`:329-359`): id, count-prefixed loop, meso — matches,
    including the loop writing/reading `{itemId; count}` in the same order
    (`:337-338` / `:353-354`).
  - `Failed` (`:383-395`): `nResult` only on both sides.
  - Cross-checked every cited `gms_v95` address in the code comments
    (`0x910717`, `0x910732`, `0x910746`, `0x91080e`, `0x91082a`, `0x91083d`,
    `0x9108ef`, `0x910904`, `0x9109c6`, `0x9109d8`, `0x910aa2` for the create
    body; `0x91037a`/`0x91038d` for monster-crystal; `0x910516`, `0x91058b`,
    `0x9105a9`, `0x9105bc`, `0x91068f` for disassemble) against
    `wire-derivation.md`'s §3 per-version address table
    (`docs/tasks/task-285-maker-skill-crafting/wire-derivation.md:475-499`).
    Every address matches exactly. Per the task's declared scope limitation,
    I did not (cannot, no `ida-pro`) verify the addresses against the client
    binary itself — only that the derivation record and the shipped code
    are mutually consistent, which they are.

- **No version gate.** Grepped `maker_result.go` and `maker_result_body.go`
  for `MajorAtLeast`/version conditionals — none present. This matches
  `wire-derivation.md`'s verdict table (`:509-521`, all eight rows
  "IDENTICAL"/"REFERENCE") and the file's own header comment
  (`maker_result.go:26-37`) citing all eight per-version addresses with no
  conditional split. Correct: a gate here would be a defect, and there is
  none.

- **Loop counts from `len()`, never caller-supplied.** `maker_result.go:78,
  83, 335` — all three counted-loop writes use `uint32(len(...))` on the
  slice actually being iterated, not a separate parameter.

- **`packet-audit:fname` markers ↔ `run.go` cases.** Five markers
  (`maker_result.go:124, 191, 258, 306, 368`, exactly `#Create`,
  `#CreateWithUpgrade`, `#MonsterCrystal`, `#Disassemble`, `#Failed`) match
  the five new `run.go` cases verbatim (`run.go:733, 735, 737, 739, 741`).

- **`toolSha` discipline.** `git diff f712ada30..8f596fd97 --
  docs/packets/audits/STATUS.md docs/packets/audits/status.json` shows
  exactly one changed line in each file: the `Tool:`/`toolSha` hash
  (`00ce3601... → 07ab1979...`). No matrix cell, export hash, or verdict
  moved. Confirmed these files are genuinely inside commit `8f596fd97`
  (`git show --stat 8f596fd97` lists both paths) rather than merely present
  in the working tree.

- **Family disposition (neither baseline nor `families.yaml`).** Grepped
  both `docs/packets/dispatcher-lint-baseline.yaml` and
  `docs/packets/families.yaml` for `MakerResult`/`MAKER_RESULT`/
  `OnMakerResult` — no hits in either, matching the `run.go` comment's claim
  (`run.go:718-731`).

## 3. Gates re-run against the committed tree (not taken from the report/transcript)

Ran all four from the worktree root, against the tree as committed at
`8f596fd97` (no working-tree changes present):

```
$ go run ./tools/packet-audit dispatcher-lint && echo GATE_PASS
dispatcher-lint: clean
GATE_PASS

$ go run ./tools/packet-audit matrix --check && echo GATE_PASS
note  n-a evidence consumed: CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79 (docs/packets/feature-na-evidence.yaml)
note  n-a evidence consumed: USE_TELEPORT_ROCK × gms_v48 (docs/packets/feature-na-evidence.yaml)
GATE_PASS

$ go run ./tools/packet-audit operations --check && echo GATE_PASS
operations check OK (0 absent-writer note(s))
GATE_PASS

$ go run ./tools/packet-audit fname-doc --check && echo GATE_PASS
fname-doc check OK (277 structs without an audit report carry no fname)
GATE_PASS
```

All four exit 0. The two `matrix --check` notes are pre-existing n-a
evidence consumption unrelated to `MAKER_RESULT`.

Also ran, from `libs/atlas-packet`:

```
$ go build ./... && echo BUILD_OK
BUILD_OK
$ go vet ./... && echo VET_OK
VET_OK
```

## 4. Blocking finding — no tests for any of the five new arms

`grep -rln "MakerResult" --include="*_test.go" .` returns **no matches** —
there is no test file at all for
`libs/atlas-packet/character/clientbound/maker_result.go` or
`libs/atlas-packet/character/maker_result_body.go`. No Encode/Decode
round-trip test, no byte fixture, nothing exercising the `noItemAwarded`
inversion (`w.WriteBool(noItemAwarded)` truthy-suppresses the following
pair) or the `catalystUsed` gating, and nothing confirming
`MakerResultFailedBody` actually delegates to `NewMakerResultFailed(result)`
without a mode.

This is not a hypothetical convention concern: the brief's own two
"patterns to copy" both ship this kind of test as a matter of course —
`libs/atlas-packet/cash/clientbound/shop_operation_result_failed_test.go`
and siblings, and `libs/atlas-packet/field/clientbound/mts_operation_test.go`
— and the **immediately preceding sibling commit in this same task**,
`ed190de3a` (Task 7, `MAKER_SKILL` serverbound, structurally the same
"discrete per-mode arm derived from the same wire-derivation.md" shape)
shipped a 321-line `maker_skill_test.go` with per-version byte fixtures and
an explicit round-trip test
(`TestMakerSkillEncodeDecodeRoundTripPerMode`), which Task 7's own review
(`docs/tasks/task-285-maker-skill-crafting/reviews/task-7.md:81-101`) relied
on as load-bearing evidence that the derived layout actually works, not
just that it compiles. Task 8 has none of that. `go build`/`go vet`/the four
audit gates all being green does not substitute — none of those exercises
the actual byte-level Encode/Decode contract this commit introduces
(inverted booleans, conditional tails, shared-vs-discrete struct pairing).

This is squarely inside the reviewable surface (the diff introduces the
untested code) and is a "does it do what the brief said" + "test honesty"
concern in the sense that there is no test to be honest or dishonest about
— the derived wire layout, however carefully cross-checked against
`wire-derivation.md` on paper, has never been exercised as running Go code.

## Not evaluable

- **Ground truth against the client binary.** No `ida-pro` access; judged
  the derivation record's internal consistency only, per the declared scope
  limitation. Not counted as a finding.
- **`.superpowers/sdd/plan/task-8-report.md`** does not exist in the
  worktree. Its intended content (the implementer's own narrative of the
  `MakerResultFailedBody` deviation) is present instead as code comments
  (`maker_result_body.go:53-66`), which this review verified directly
  against `resolve.go`, so the missing report is not a blocking gap — but it
  is worth flagging that the expected artifact is absent.

## Summary

The one deliberate, flagged deviation (`MakerResultFailedBody` skipping
`WithResolvedCode`) is correct and in fact required by the brief's own
`NewMakerResultFailed(result uint32)` signature — ruled in the implementer's
favor, independently verified against `resolve.go` and
`dispatcher_lint.go`. INV-1, INV-2, INV-3, INV-5, the no-version-gate
requirement, the fname markers, and the `toolSha`-only diff on
`STATUS.md`/`status.json` all hold under direct inspection and gate re-runs.
The blocking gap is the complete absence of tests for the five new arms,
which is a material regression in rigor relative to both the cited
"patterns to copy" and the immediately preceding sibling commit in the same
task.
