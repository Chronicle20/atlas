# Review — bug-hall-duplicate-and-click-logout fix (b224ee255..e249d96eb)

Reviewed against `docs/tasks/task-251-player-npcs/bug-hall-duplicate-and-click-logout.md`'s
`## Fix` inventory and `docs/tasks/task-251-player-npcs/bug-fix-report.md`.

Commits: `187e49bab` (defect 1 @help), `ba201d4d4` (defect 2 imitate filter),
`6a8c51bdb` (defect 3 click-logout guard), `e249d96eb` (defect 4 rx0/rx1 swap).

## Defect 1 — `@playernpc` missing from `@help`

PASS. `services/atlas-messages/atlas.com/messages/command/help/commands.go:38-39` adds the two
exact lines specified in the brief, matching the existing phrasing style of `commandSyntaxList`.
Cross-checked against the registered regexes in
`services/atlas-messages/atlas.com/messages/command/playernpc/commands.go:29-30`
(`deployRe`/`removeRe`) — the described `add <target>` / `remove <target> [here]` shapes match.

## Defect 2 — duplicate Player NPC render (imitate-band filter)

PASS, with the choke point and both inheriting call sites confirmed by hand:

- `libs/atlas-object-id/reserved.go:33-52` adds `IsPlayerNpcImitateTemplate` bounded by
  `playerNpcImitateTemplateMin`/`Max` = 9901000/9906599, matching design §4.2 and the brief.
- `services/atlas-channel/atlas.com/channel/data/npc/processor.go:23` defines
  `notImitateTemplate`, wired into `InMapModelProvider` at line 61 via
  `model.Filters[Model](notImitateTemplate)` — this is the single choke point the brief names.
- `ForEachInMap` (processor.go:47-49) calls `InMapModelProvider`, so it inherits the filter.
  Traced both named consumers:
  - `kafka/consumer/map/consumer.go:240` — `npc2.NewProcessor(...).ForEachInMap(...)` for
    `spawnNPCForSession`. Inherits the filter — placeholders no longer double-spawn.
  - `npc/controller/processor.go` documents (line 26) that `data/npc.ForEachInMap` fans the
    controller sweep; it too inherits the filter, so no controller is elected for a placeholder.
- `InMapByObjectIdModelProvider`/`GetInMapByObjectId` (processor.go:63-69) are untouched — verified
  by reading the full file; no filter was added to that provider, exactly as the brief directed
  (defect 3 covers that path by guarding the caller instead).
- Swept every other consumer of `InMapModelProvider`/`ForEachInMap` on `data/npc` specifically
  (as opposed to the unrelated `door`/`chair`/`monster`/etc. packages that share the method names
  but are different types): `kafka/consumer/buff/consumer.go:394` also calls
  `npc2.NewProcessor(...).InMapModelProvider(...)` for the GM-reveal "uncontrolled NPCs" check.
  This consumer is not named in the bug file, but the effect is consistent with intent (a
  placeholder was never controllable and excluding it from `UncontrolledIn`'s candidate set is
  correct, not a regression) — flagged here as a non-blocking observation since the brief's
  inventory didn't enumerate it, not because it is wrong.
- New test `data/npc/processor_test.go::TestInMapModelProvider_FiltersImitateTemplates` exercises
  `notImitateTemplate` directly through `model.FilteredProvider`, with boundary-pinning entries at
  9901000/9906599 (inside) and 9900999/9906600 (just outside). `go test ./data/npc/...` passes.

## Defect 3 — click-logout guard

PASS. `services/atlas-channel/atlas.com/channel/socket/handler/npc_start_conversation.go:32`:
```go
if oid >= objectid.PlayerNpcObjectIdBase && oid < objectid.MinId {
```
`PlayerNpcObjectIdBase = 100000` (`libs/atlas-object-id/reserved.go:13`) and
`MinId = 1000000` (`libs/atlas-object-id/allocator.go:31`) — the bound is exactly
`[PlayerNpcObjectIdBase, MinId)` as specified. The guard returns (line 34) before the
`GetInMapByObjectId` call at line 39 and the `session.Destroy` branch at line 42, so it never
reaches the anti-cheat disconnect.

Verified the regression test is honest, not merely coverage: I temporarily short-circuited the
guard (`if false && oid >= ...`) and reran
`TestNPCStartConversation_PlayerNpcOidIsNoOp` — it failed
(`session for character [701] was destroyed on a Player NPC click: empty slice`), confirming the
test fails without the fix. Restored the file afterward; `git diff --stat` on it is clean.

Per the task instructions, `npc_action.go:31` and `movement/processor.go:128` were independently
confirmed to return without `Destroy` on their `GetInMapByObjectId` misses, and `npc_item_use.go`
has no oid-based lookup at all — the implementer's decision to leave both unchanged is correct,
and the brief's naming of `npc_item_use.go` was wrong (documented candidly in the implementer's
report's "Discrepancy found" section).

## Defect 4 — rx0/rx1 swap

PASS. `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/builder.go:128-130` now derives
`rx0 = x - 50`, `rx1 = x + 50` (was reversed), matching the WZ convention (`rx0 < rx1`) with
direct evidence in the bug file (`x -1, rx0 -51, rx1 49`).
`playernpc/model_test.go:188-192` inverts the expectations for `x = 100` to `RX0() == 50`,
`RX1() == 150`, consistent with the new derivation.
`docs/tasks/task-251-player-npcs/design.md:241` was corrected to state
`` `rx0 = x - 50`, `rx1 = x + 50` `` and now cites the WZ convention and the bug report instead of
the (wrong) original §3.1 text.

Swept other `rx0`/`rx1` references repo-wide (atlas-channel's `playernpc/model.go`, `rest.go`,
`kafka/*`; atlas-player-npcs' `entity.go`, `producer.go`, `rest.go`, `administrator.go`); all of
them pass the fields through opaquely (REST/Kafka marshalling) and none independently recompute
the derivation, so none needed updating.

## Build/test verification

Ran directly (not just trusting the implementer's report):
- `atlas-channel`: `go build ./...` clean; `go test ./data/npc/... ./socket/handler/...` → `ok`.
- `libs/atlas-object-id`: `go test ./...` → `ok`.
- `atlas-player-npcs`: `go test ./playernpc/...` → `ok`.
- `atlas-messages`: `go build ./...` clean.
- `go.work.sum`/`go.mod`/`go.sum` diff confirmed to contain no service `go.mod`/`go.sum` changes
  (`git diff --stat ... -- '**/go.mod' '**/go.sum'` empty) — the `go.work.sum` diff is transitive
  checksum lines only, as the implementer's report states.

## Not evaluable

- Live re-test in `atlas-pr-1475` (Hall of Fame map, double-click, `@help` listing) was not
  re-run by this review; the bug file's "Outcome" section is still unfilled. This review is
  source/test-level only.
- The brief's "Not yet answered" item (whether any non-Hall map has a real, interactive
  9901000-9906599 life entry) remains genuinely unanswered — neither the brief nor this fix
  resolves it, and nothing in the diff contradicts the brief's stated assumption.

## Verdict rationale

All four defects are fixed exactly as specified, at the exact locations the brief named, with
tests that were verified (not merely trusted) to be honest regressions. The one item outside the
brief's explicit inventory (the `buff` consumer's incidental inheritance of the filter) is a
correct side effect, not a defect, so it is noted but does not block.
