# Review — fix(monsters): contact self-destructs detonate as bomb, not WZ action byte

Range: `60ac906d9` (single commit)
Requirement: `docs/tasks/task-253-self-destructing-mobs/bug-darkstar-no-explosion-or-damage.md`,
"## Fix — DECIDED 2026-08-25 (reporter approved)"
Implementer report: `docs/tasks/task-253-self-destructing-mobs/fix-contact-bomb-report.md`

## Scope

`git diff --stat 60ac906d9` (single commit, parent implicit):

```
docs/tasks/task-253-self-destructing-mobs/design.md               |  48 +++++-
docs/tasks/task-253-self-destructing-mobs/fix-contact-bomb-report.md | 175 ++++++++
docs/tasks/task-253-self-destructing-mobs/prd.md                  |   6 +-
services/atlas-monsters/.../monster/processor.go                  |  20 +++--
services/atlas-monsters/.../monster/self_destruct_test.go         |  60 +++-
```

No `atlas-channel` files, no seed-template files, and no `self_destruct_timer_task.go`
are in this diff — the fix is confined to `atlas-monsters`' `processor.go` and its test,
plus the three doc files. This matches the "decided fix" scope note that the deviation
touches only `SelfDestruct`, and is narrower than the bug file's original "files that
would change" list (which speculatively included the timer task file and
`self_destruct_timer_test.go` — neither needed a change, and none was made, which is
correct: `TriggerTimer` calls `SelfDestruct` and inherits the untouched
`deathTypeForAction` behavior for anything other than `TriggerContact`).

## Findings

### 1. Only `TriggerContact` deviates — PASS

`services/atlas-monsters/atlas.com/monsters/monster/processor.go:1876-1880` (post-diff):

```go
deathType := deathTypeForAction(p.l, sd.Action())
if trigger == TriggerContact {
    deathType = DeathTypeBomb
}
p.selfDestructFrom(m, characterId, deathType, trigger)
```

- Threshold path: `processor.go:695` — `deathTypeForAction(p.l, sd.Action())` passed
  directly to `selfDestructFrom(..., TriggerThreshold)`, unchanged by this commit and not
  routed through the new branch (it never calls `SelfDestruct`). Confirmed via `grep`
  that line 695 is untouched in the diff.
- Timer path: `self_destruct_timer_task.go:46` — calls
  `NewProcessor(t.l, tctx).SelfDestruct(uniqueId, 0, TriggerTimer)`, which does route
  through the new `if`, but the guard is `trigger == TriggerContact`, so a `TriggerTimer`
  call falls through unchanged to `deathTypeForAction(sd.Action())`. Confirmed by the new
  test `TestSelfDestructContactAlwaysBomb/timer_keeps_WZ-derived_deathType`, which passes.
- Firebomb's threshold behaviour (WZ pass-through) is not touched: no lines in
  `damageCore`'s threshold arm are part of this diff.

### 2. Consumer seam (`destroyCodeFor`) and seed templates — PASS, verified not assumed

`services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:217-222`
(not modified by this commit, read to verify pre-existing behavior):

```go
func destroyCodeFor(deathType string) writer.DestroyMonsterCode {
	if deathType == "" {
		return writer.DestroyMonsterFadeOut
	}
	return writer.DestroyMonsterCode(deathType)
}
```

This is a direct string-cast, not a switch — so the seam works for `"BOMB"` as long as
`writer.DestroyMonsterBomb == "BOMB"`. Confirmed at
`services/atlas-channel/atlas.com/channel/socket/writer/monster_destroy.go:20`:
`DestroyMonsterBomb DestroyMonsterCode = "BOMB"`, and `DestroyMonsterBody` resolves that
key through `atlas_packet.WithResolvedCode("operations", string(key), ...)` — i.e.
through the tenant operations table (DOM-25), not a literal byte.

Checked all 11 seed templates under
`services/atlas-configurations/seed-data/templates/` (`gms_{12,48,61,72,79,83,84,87,92,95}_1`,
`jms_185_1`) for the `DestroyMonster` writer's `operations` block. Every one already has
exactly one `"BOMB"` entry, e.g. `template_gms_83_1.json`:

```
"writer": "DestroyMonster",
"fname": "CMobPool::OnMobLeaveField",
"options": { "operations": { "DISAPPEAR": 0, "FADE_OUT": 1, "BOMB": 2, ... } }
```

`BOMB: 2` on every version checked (83/87/92/95/JMS185), matching the bug file's
decided byte-2 rationale. None of these template files are part of this commit's diff —
correctly so, since the key already existed; the implementer's "verify, don't add"
claim holds up.

### 3. No literal dead-type byte in the diff — PASS

Swept every added line (`+` lines) in `processor.go` and `self_destruct_test.go` for a
bare `2`/`4` outside `DeathTypeBomb`/`deathType` identifiers. The only match is the
`SelfDestruct` doc comment ("dead-types 2, 4, 5" — prose, not code) and an unrelated
`testInformationLookup` line. `DeathTypeBomb` is the sole production value used, resolved
through the operations table per finding 2.

### 4. New test asserts the NEW contract — PASS

`self_destruct_test.go` (post-diff):
- `TestSelfDestructRejects`'s "valid_target" subtest now asserts `DeathType ==
  DeathTypeBomb` for a `TriggerContact` call with WZ action 3 (was
  `DeathTypeDestructByMiss` pre-fix) — this assertion fails against the pre-fix code,
  since the old code path always returned `deathTypeForAction(3) == DESTRUCT_BY_MISS`
  regardless of trigger.
- New `TestSelfDestructContactAlwaysBomb` table-tests all three triggers: contact
  (actions 1 and 3) → `DeathTypeBomb`; threshold (action 3) → `DeathTypeDestructByMiss`;
  timer (action 3) → `DeathTypeDestructByMiss`. This is exactly the discriminating
  matrix the bug file's fix section implies (contact deviates, threshold/timer don't) —
  a regression that widened the deviation to threshold/timer, or one that failed to
  apply it to contact, would both be caught.
- Ran the suite locally: `go build ./...` clean, `go test ./monster/ -run
  'TestSelfDestruct' -v` — all subtests pass, including the two new/changed ones. Output
  matches the implementer's report.

### 5. design.md §2.2 amendment — PASS

The v83 `CMobPool::Update 0x679138` two-arm switch, `CMob::OnDie 0x663995`, and the
dead-type-3 → one-time-action-21 branch (`0x663a1b`) are all present in the design.md
diff and match the bug file's citations verbatim (same addresses, same `v6 =
m_pTemplate[137]` / `m_nDeadType == 3` code). `CMob::OnBomb 0x663e5b` appears in the
D2 amendment paragraph, not §2.2's correction paragraph itself — but it is present in
the design.md diff exactly once, correctly attributed. The §2.2 correction is appended
as a new paragraph after the original v87/v95 text, not a rewrite of it (the original
"So on v87: `{0,1,3}` → `OnDie`..." sentence survives, now qualified with "/v83"
inline rather than deleted). The D2 amendment is likewise appended after D3's original
text begins, not interleaved into it. Both read as genuine amendments, not silent
rewrites.

### 6. prd.md 5100002 mislabel — PASS (scoped correctly)

`prd.md` §6.3 row (`5100002 | ... | Firebomb`) and the user story at what is now
`prd.md:66` ("As a player killing a Firebomb...") are both corrected. Three other
"Boomer" mentions remain (`prd.md:21`, `:110`, `:438`) — left untouched, which the task
brief explicitly scoped as out-of-bounds ("§6.3 row and the prd.md:66 user story" only).
Not a defect.

## Out-of-scope items confirmed absent (correctly)

- No new damage-emission code added anywhere in `selfDestructFrom` or its callees.
- No change to `TriggerThreshold`/`TriggerTimer` behavior (see finding 1).
- No touch to `atlas-data`'s `getFirstAttack`.

## Not evaluable

None. All six checklist items were verifiable within the diff plus the two read-only
seam files (`consumer.go`, `monster_destroy.go`) and the seed templates.

## Verdict

APPROVED — the fix is scoped exactly to `TriggerContact`, the cross-service seam
(`destroyCodeFor` → `writer.DestroyMonsterBomb` → `"BOMB"` operations-table key) already
exists end-to-end in every seed template and was not modified (correctly, per "verify,
don't add"), no literal dead-type byte was introduced, the new test discriminates the
new contact contract from the unchanged threshold/timer contract and fails without the
fix, and both doc amendments (design.md §2.2/D2, prd.md) are additive and accurate
against the bug file's citations.
