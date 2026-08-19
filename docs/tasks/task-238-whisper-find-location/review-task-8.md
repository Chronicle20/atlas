# Review: Task 8 — the core `/find` decision table

**Commit range:** `98ec7fd7f..8ca62121b` (single commit `8ca62121b`, "fix(atlas-channel): report the target's real location for /find")

**Brief:** `.superpowers/sdd/plan/task-8-brief.md`
**Implementer report:** `.superpowers/sdd/plan/task-8-report.md`

**Files touched:**
- `services/atlas-channel/atlas.com/channel/maps/location/requests.go` (+6)
- `services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper.go` (+160/-17)
- `services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper_test.go` (new, +492)

## 1. Decision table — every row, in order

Read `findDecision` (`character_chat_whisper.go:93-162`) top to bottom against the brief's table:

| # | Branch | Location in code | Test |
|---|---|---|---|
| FR-1 | `unresolved` | `:94-98`, `findCharacterByNameFunc` err | `TestFind_FR1_UnresolvableName` — PASS |
| FR-2 | `cross-world` | `:102-104`, `tc.WorldId() != s.Field().WorldId()` | `TestFind_FR2_CrossWorld` — PASS, also asserts `locCalls == 0` |
| FR-3 | `gm-concealed` | `:108-110`, `tc.Gm() && !s.Gm()` | `TestFind_FR3_GmConcealment` — PASS (5 subcases); mutation-killed below |
| FR-4a | `cash-shop-local` | `:115-120`, local session found and `CashScene() != CashSceneNone` | `TestFind_FR4a_LocalCashScene` — PASS (cash shop + mts scenes) |
| FR-5 | `map-local` | `:121-136`, local session found, scene none | `TestFind_FR5_LocalOnMap` — PASS, x/y-by-wire-length check for both arms |
| FR-4b | `cash-shop-remote` | `:145-147`, `loc.State() == PresenceStateInCashShop` | `TestFind_FR4b_RemoteCashShop` — PASS |
| FR-6 | `channel-remote` | `:148-156`, `loc.State() == PresenceStateInField`, `channelId: uint32(loc.ChannelId())` | `TestFind_FR6_RemoteChannel` — PASS; mutation-killed below (the actual bug fix) |
| FR-7a | `offline` | `:157-160` default case of the `loc.State()` switch | `TestFind_FR7_NotFindable/offline_*` — PASS |
| FR-7b | `never-logged-in` | `locationErrorOutcome:168-169`, `errors.Is(err, location.ErrNotFound)` | `TestFind_FR7_NotFindable/never_logged_in_*` — PASS |
| FR-7c | `lookup-failed` | `locationErrorOutcome:170-171` | `TestFind_FR7_NotFindable/lookup_failed_*` — PASS, and asserts `level=error` in the log |

**Local-session-before-location ordering** (the item called out explicitly in the task): confirmed at `character_chat_whisper.go:115` — `findLocalSessionFunc` is called and, on success, both FR-4a and FR-5 are resolved *without ever calling* `findCharacterLocationFunc` for FR-5's map id lookup exception (see next paragraph) or at all for FR-4a. Location lookup is only reached at `:139` after the local-session branch returns `lerr != nil` (no local session). This matches the brief's ordering exactly and both `TestFind_FR4a_LocalCashScene` and `TestFind_FR2_CrossWorld` assert `env.locCalls == 0` where the table says location must not be consulted.

One nuance not explicit in the summary table but present in the brief's code listing: FR-5 (map-local) still calls `findCharacterLocationFunc` once, to source `mapId` (the local session doesn't carry a map id). This is exactly what the brief's Step 4 code does (`:125` `loc, lerr2 := findCharacterLocationFunc(...)`), and is a different call from the remote-location-consulted-first-only-after-local-session-absent flow. Confirmed correct, not a table violation.

All ten table rows are present, correctly ordered, first-match-wins (each is an early `return`), and each has a passing, branch-specific test that additionally asserts `env.logs` contains the branch name from the FR-13 log line.

**Verdict: PASS.**

## 2. Channel id — no hidden ±1

`character_chat_whisper.go:155`: `channelId: uint32(loc.ChannelId())` — a direct, unadjusted conversion. No arithmetic anywhere else in the diff touches `channelId` before it reaches `NewWhisperFindResultChannel` (`:207` in the adapter, `o.channelId` passed straight through).

Confirmed against the codec: `libs/atlas-packet/field/clientbound/whisper.go` `WhisperFindResultChannel.Encode`:
```go
w.WriteByte(m.mode)
w.WriteAsciiString(m.targetName)
w.WriteByte(3)
w.WriteInt(m.channelId)
```
— `w.WriteInt(m.channelId)` with no adjustment, matching the brief's claim exactly.

Mutation test performed (see §6): reverting `channelId: uint32(loc.ChannelId())` to a hard-coded `channelId: 0` (i.e., reintroducing the actual historical bug) makes `TestFind_FR6_RemoteChannel` fail with `channelId = 0, want 7`. The test is not vacuous.

**Verdict: PASS.**

## 3. `location.ResolveMapId` not used

`grep -n "ResolveMapId" character_chat_whisper.go` returns exactly one hit, at line 58, inside a comment explaining *why* `location.Get` was chosen over `ResolveMapId` ("ResolveMapId collapses every failure to map id 0"). The diff confirms the only removed reference to `ResolveMapId` was in the deleted old code (`git diff` shows `tcMapId := location.ResolveMapId(...)` only on a `-` line). Production code now calls `findCharacterLocationFunc`, which wraps `location.Get`, exclusively.

**Verdict: PASS.**

## 4. Both placeholder comments removed outright

`git diff 98ec7fd7f..8ca62121b -- .../character_chat_whisper.go` shows, as `-` (deleted) lines only:
```
-					// TODO query cash shop.
-					cs := false
...
-					// TODO find a way to look up remote channel.
```
Neither placeholder survives anywhere in the new file (confirmed by grepping the post-image for `TODO` — zero hits). The `cs := false` dead-flag block is gone entirely, replaced by the real FR-4a/FR-4b/FR-6 logic. This satisfies the CLAUDE.md prohibition on landing placeholder/stubbed handlers.

**Verdict: PASS.**

## 5. The two disclosed deviations

Both are called out explicitly in `task-8-report.md` §"Deviations from the brief's literal code listing" and verified independently here against the real repo signatures (not by trusting the report):

- **`MustBuild()` vs `.Build()`**: `character/builder.go:153` — `func (b *modelBuilder) Build() (Model, error)`; `character/builder.go:200` — `func (b *modelBuilder) MustBuild() Model`. The brief's snippet assigns the builder chain's result directly to `env.target character.Model` (a single value), which `Build()` cannot satisfy (it returns two values). `MustBuild()` is the correct, and only, mechanical fix — it is not a behavior change; both panic-on-error and error-returning forms build the same `Model` on the success path exercised by every test.
- **`request.NewRequestReader(&req, 0)` vs `request.Request(b).Reader()`**: `libs/atlas-socket/request/request.go` defines `type Request []byte` with no `Reader()` method anywhere in the package; `libs/atlas-socket/request/reader.go:9` defines `func NewRequestReader(p *Request, time int64) Reader`. The brief's literal `.Reader()` call does not compile against the real type. The substituted pattern matches the decode idiom used elsewhere in the same test package (report cites `npc_item_use_test.go:78-79`) and is purely a construction mechanism for the reader used to decode wire bytes back into a struct for assertions — it has no effect on the packet bytes produced by the handler under test.

Both deviations are genuine, disclosed, and are mechanical API-signature corrections, not scope or behavior changes.

**Verdict: PASS.**

## 6. Test quality — mutation testing on the two highest-risk rows

Reverted the file to a temporary mutant, ran the targeted test, then restored (confirmed via `git status --porcelain` afterward that the tree is clean — no stray edits left for the Task 9 implementer working in `docs/packets/` / `libs/atlas-packet/`).

**FR-3 (gm-concealed):** replaced `if tc.Gm() && !s.Gm() {` with a mutant that can never conceal (`if tc.Gm() && tc.Gm() && false {`). Result:
```
--- FAIL: TestFind_FR3_GmConcealment (0.00s)
    --- FAIL: TestFind_FR3_GmConcealment/gm_1_target,_ordinary_requester (0.00s)
    --- FAIL: TestFind_FR3_GmConcealment/gm_2_target,_ordinary_requester (0.00s)
```
The test kills the mutant. Not vacuous.

**FR-6 (channel-remote, the actual bug):** replaced `channelId: uint32(loc.ChannelId())` with `channelId: 0` (the historical bug). Result:
```
--- FAIL: TestFind_FR6_RemoteChannel (0.00s)
        character_chat_whisper_test.go:388: channelId = 0, want 7 (a hard-coded 0 must not pass)
```
The test kills the mutant. Not vacuous. `findRemoteChannel = channel.Id(7)` is deliberately neither `0` nor `1` (per the test file's own comment), which is exactly right — a fixture of 0 or 1 would pass against the broken code by accident.

After each mutation, the file was restored from a pre-mutation backup and `go build ./...` + `git status --porcelain` were used to confirm the tree returned to the exact committed state (no stray diffs left behind).

**Verdict: PASS — both rows are genuinely tested, not vacuous coverage.**

## 7. `NewModelForTest`

`maps/location/requests.go:135-138`:
```go
// NewModelForTest constructs a Model directly. Only call from a test;
// production code builds one through Get.
func NewModelForTest(characterId uint32, w world.Id, ch channel.Id, m _map.Id, instance uuid.UUID, state characterconst.PresenceState) Model {
	return Model{characterId: characterId, worldId: w, channelId: ch, mapId: m, instance: instance, state: state}
}
```
A pure struct literal constructor with no side effects, exported purely to let tests build a `location.Model` directly — matches the existing `SetBaseURLForTest` pattern immediately below it in the same file. No production code path (`Get`) is altered; confirmed by reading the surrounding diff context (the `Get` function body is untouched, `NewModelForTest` is inserted as a new standalone function).

**Verdict: PASS — test-support only, no behavioral impact.**

## Other checks

- **Build & tests**: `go build ./...` clean; `go test ./socket/handler/ -run TestFind_ -v` — all 9 top-level tests / ~30 subtests PASS; full module suite `go test ./...` — PASS across all packages, no regressions.
- **`go vet ./socket/handler/... ./maps/location/...`** — clean.
- **Arm symmetry** (`TestFind_ArmSymmetry`) confirms the 0x09/0x48 arms are byte-identical past the mode byte for cash-shop, channel, and error outcomes — a reasonable sanity check that the two whisper modes share one code path faithfully.
- **Types**: `character.Model.Gm()` (`character/model.go:69`, `bool`, `gm > 0` — Task 7's work, consumed correctly here) and `session.Model.Gm()` (`session/model.go:105`, `bool`) are both boolean, so `tc.Gm() && !s.Gm()` is a correct boolean predicate, not a residual int/tier comparison.
- **`findLocalSessionFunc`'s `ch channel.Model` parameter** matches `session.NewProcessor(...).GetByCharacterId(ch channel.Model)` (`session/processor.go:235`) and is fed `s.Field().Channel()`, whose return type is `channel.Model` (`libs/atlas-constants/field/model.go:33`). Correctly typed, no coercion games.
- Working tree left clean; no interference with the Task 9 implementer's in-progress work under `docs/packets/` / `libs/atlas-packet/` (untouched, not inspected beyond the one `whisper.go:227` `Encode` line required to verify the channel-id claim).

## Not evaluable

None. Everything the brief asked to be checked was checkable within this unit's diff plus the one called-out cross-file contract (`whisper.go`'s `Encode`).

## Scope

The diff matches the brief precisely — no extra files touched, no scope creep, no unrelated refactor. The implementer's two disclosed deviations are the only departures from the brief's literal listing, and both are correctly characterized as mechanical, not behavioral.

---

## Verdict

verdict: APPROVED
artifact: docs/tasks/task-238-whisper-find-location/review-task-8.md
scope_confirmed: reviewed the full diff of 8ca62121b (character_chat_whisper.go rewrite, its new test file, and the NewModelForTest addition in maps/location/requests.go), cross-checked the channel-id claim against libs/atlas-packet/field/clientbound/whisper.go's Encode, and mutation-tested the FR-3 and FR-6 rows against the real committed code (tree restored and confirmed clean afterward).
blocking: 0
non_blocking: 0
not_evaluable: 0
