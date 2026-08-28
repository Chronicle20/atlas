# Bug fix report — task-251 Hall duplicate / click logout

Implements all four items in `bug-hall-duplicate-and-click-logout.md`'s `## Fix` inventory.

## What was implemented

### Defect 1 — `@playernpc` missing from `@help`
`services/atlas-messages/atlas.com/messages/command/help/commands.go` — added the two
`commandSyntaxList` entries verbatim as specified in the brief.

### Defect 2 — duplicate Player NPC render
- `libs/atlas-object-id/reserved.go` — added `IsPlayerNpcImitateTemplate(templateId uint32) bool`
  (plus unexported `playerNpcImitateTemplateMin`/`Max` = 9901000/9906599, design §4.2), named to
  match the file's existing vocabulary (`PlayerNpcObjectIdFor`, `PlayerNpcObjectIdBase`).
- `services/atlas-channel/atlas.com/channel/data/npc/processor.go` — added the unexported
  `notImitateTemplate(m Model) bool` predicate and passed it into
  `InMapModelProvider`'s `model.Filters[Model](...)` call. `InMapByObjectIdModelProvider` /
  `GetInMapByObjectId` are untouched, as the brief directed (defect 3's fix covers that path).
- `services/atlas-channel/atlas.com/channel/data/npc/processor_test.go` (new) —
  `TestInMapModelProvider_FiltersImitateTemplates` builds a mixed list (imitate-pool boundaries at
  9901000/9906599, plus one entry each just outside the pool on both sides, plus one ordinary WZ
  template) through `model.FilteredProvider` + `notImitateTemplate` and asserts only the
  non-imitate entries survive.

### Defect 3 — click-logout
- `services/atlas-channel/atlas.com/channel/socket/handler/npc_start_conversation.go` — added the
  guard exactly as specified: `oid >= objectid.PlayerNpcObjectIdBase && oid < objectid.MinId`
  returns early (debug log, no `Destroy`) before the `GetInMapByObjectId` call, citing PRD FR-7.4.
- `services/atlas-channel/atlas.com/channel/socket/handler/npc_start_conversation_test.go` (new) —
  `TestNPCStartConversation_PlayerNpcOidIsNoOp` drives the real `NPCStartConversationHandleFunc`
  with a StartConversation packet carrying `objectid.PlayerNpcObjectIdBase + 1000` and asserts the
  character's session is still present in the registry afterward (i.e. `Destroy` never ran).
- `services/atlas-channel/atlas.com/channel/socket/handler/npc_item_use.go` — **not changed**; see
  "Discrepancy found" below.

### Defect 4 — `rx0`/`rx1` swapped
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/builder.go:124-132` — swapped to
  `rx0 = x - 50`, `rx1 = x + 50`; doc comment now cites the WZ convention (rx0 < rx1) and the bug
  report instead of design §3.1.
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/model_test.go:188-193` — inverted the
  expectations (`RX0() == 50`, `RX1() == 150` for `x = 100`).
- `docs/tasks/task-251-player-npcs/design.md:241` — corrected the stated derivation
  (`rx0 = x - 50`, `rx1 = x + 50`) and noted it matches the WZ convention / cites the bug report.

## Discrepancy found — npc_item_use.go (defect 3)

The brief's fix inventory says `npc_item_use.go` "performs the same probe and needs the same
treatment," citing comments at `:30`/`:80`. I read the file (and its only history commit,
`543a88df6`) and confirmed: it never calls `GetInMapByObjectId`, never operates on a clicked map
object's oid at all, and never calls `session.Destroy`. Its "shared probe" (the comments at those
line numbers) is `npcShopProbeFunc` — `shops.NewProcessor(...).GetShop(npcTemplateId)`, used purely
to classify shop-vs-conversation for an NPC resolved from *consumable item data* (`cd.Npc()`), not
from a clicked object. That probe never fails toward `Destroy`; there is no click-logout code path
in this file to guard. I also checked the other three `GetInMapByObjectId` call sites in
atlas-channel (`npc_action.go`, `movement/processor.go`, `npc/controller/announce.go`) — none of
them call `Destroy` on a miss, only `npc_start_conversation.go` does.

I did not invent a guard for a probe that isn't the vulnerable one. Grounded on source reading, not
a guess — flagging per CLAUDE.md's "surface it and ask" rule for a brief-vs-source conflict rather
than silently deviating.

## Testing

Module-local build + test, one command block per module touched:

```
cd services/atlas-messages/atlas.com/messages && go build ./... && go test ./...
```
→ all packages `ok` (no failures; `command/help` has no test files, unchanged from before).

```
cd libs/atlas-object-id && go build ./... && go test ./...
```
→ `ok  	github.com/Chronicle20/atlas/libs/atlas-object-id	0.035s`

```
cd services/atlas-player-npcs/atlas.com/player-npcs && go build ./... && go test ./...
```
→ all packages `ok`, including `playernpc` (`ok  	atlas-player-npcs/playernpc	6.172s`).

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
→ all packages `ok`, including `atlas-channel/data/npc` (new test) and
`atlas-channel/socket/handler` (new test, `2.594s`).

No go.mod changes were needed for the new `atlas-object-id` import in `atlas-channel` — the repo's
`go.work` already `use`s that module, so `go build`/`go test` resolved it without a `require`/
`replace` edit (confirmed: `git status --short go.mod go.sum` was empty after the build). `go.work.sum`
picked up new transitive checksum lines from that resolution; diff reviewed, benign (indirect
lockfile hashes only).

## Files changed

- `services/atlas-messages/atlas.com/messages/command/help/commands.go`
- `libs/atlas-object-id/reserved.go`
- `services/atlas-channel/atlas.com/channel/data/npc/processor.go`
- `services/atlas-channel/atlas.com/channel/data/npc/processor_test.go` (new)
- `services/atlas-channel/atlas.com/channel/socket/handler/npc_start_conversation.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/npc_start_conversation_test.go` (new)
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/builder.go`
- `services/atlas-player-npcs/atlas.com/player-npcs/playernpc/model_test.go`
- `docs/tasks/task-251-player-npcs/design.md`
- `go.work.sum` (transitive checksum additions only)

Commits (in order): `187e49bab`, `ba201d4d4`, `6a8c51bdb`, `e249d96eb`.

## Self-review findings

- Considered adding a second `npc_start_conversation_test.go` case proving the *unguarded* side
  still destroys the session (a WZ-band oid miss). Wrote it, ran it, confirmed it does reproduce the
  pre-fix Destroy path — but it took ~37s and produced a wall of real Kafka dial-refused / HTTP
  retry log noise because `Destroy` unconditionally emits to two Kafka topics with real backoff.
  Removed it: not required by the brief, and a slow, noisy test asserting pre-existing (unchanged)
  behavior isn't worth that cost. The kept test is the actual regression assertion.
- Double-checked `notImitateTemplate`'s boundary values (9901000, 9906599) match the brief's stated
  pool exactly, and added off-by-one entries (9900999, 9906600) to the test to pin the boundary.
- Confirmed `GetInMapByObjectId`/`InMapByObjectIdModelProvider` were left untouched per the brief's
  explicit instruction.

## Issues or concerns

- The npc_item_use.go discrepancy above (status: `DONE_WITH_CONCERNS`, not blocking — the actual
  click-logout bug is fixed; the brief's second file target doesn't correspond to real vulnerable
  code).
- "Not yet answered" items from the brief (imitate-band template used outside Hall of Fame; whether
  the 8 suppressed placeholders were cosmetic) were not re-investigated — out of this task's scope
  per the brief itself, and nothing found during defect-2 work contradicts the brief's assumption.
