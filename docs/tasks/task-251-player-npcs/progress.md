# Task 2 — Cygnus level cap verification

PRD FR-1.2 asserts Cygnus Knights cap at 120; PRD Open Question 5 flags it unverified. This
records the evidence search and its result.

## What was searched

WZ trees (per design §1 C-1, the two stock checkouts on this machine):

- `ms_1172` checkout's `wz/String.wz`, `wz/Character.wz`
- `AtlasMS` checkout's `wz/String.wz`, `wz/Character.wz`

Commands run:

```
grep -rliE 'blockedJobs|maxLevel|levelLimit|LevelLimit|MaxLevel' \
  <tree>/String.wz <tree>/Character.wz
```

— zero matches in either tree. Neither `String.wz` nor `Character.wz` contains a field named
`blockedJobs`, `maxLevel`, or `levelLimit` anywhere in the extracted XML.

`Character.wz` in both trees contains only equip-category folders (`Cap`, `Coat`, `Weapon`, ...)
and a handful of numeric `000xxxxx.img.xml`/`00120xx.img.xml` skin/hair ids — no per-job base
stat or level-cap templates. Job-line level caps, if enforced at all in this era's data, are not
represented as WZ data in these trees; that is consistent with them being server-side logic
rather than client data.

A secondary sweep for `Cygnus` + `level`/`cap`/`120`/`200` across `String.wz/*.img.xml` in
`ms_1172` turned up only narrative/flavor text, not a cap declaration:

- `Consume.img.xml`: a mount item description — "can only be used by Cygnus Lv. 120 for 20
  days" (an item level *requirement*, not evidence of a job level *ceiling*).
- `Skill.img.xml` (Blessing of the Fairy / Cygnus's Blessing tooltips, several duplicate ids):
  "Skill Points increase by 1 when a Cygnus Knight character reaches Lv. 30, 50, 70, 100, and
  120" — a skill-point milestone list that stops enumerating at 120. Suggestive of 120 having
  been a notable threshold for Cygnus characters in this client version, but this is a skill
  tooltip's milestone list, not a max-level/level-cap field. It does not confirm a hard cap.

No `Npc.img.xml`/`Ins.img.xml` hits tying Cygnus to an explicit level ceiling were found.

## IDB (v95, session `ecc757f4`)

Checked by the controller after the fact (the implementing agent had no `mcp__ida-pro__*`
tools). The client has exactly one Cygnus-classification helper,
`is_cygnus_job(long)` at `0x47ca80` — a pure `nJob / 1000 == 1` test with no level term.
All 15 of its call sites were enumerated (`xrefs_to 0x47ca80`):

| caller | address |
|---|---|
| `CLogin::SendDeleteCharPacket` | `0x5d5517` |
| `CSkillInfo::GetShootSkillRange` | `0x7097e2`, `0x709843` |
| `get_weapon_mastery` | `0x709ba3`, `0x709cd5`, `0x709d7a` |
| `get_critical_skill_level` | `0x70a2d8`, `0x70a333` |
| `CUserLocal::TryDoingNormalAttack` | `0x912551` |
| `CUserLocal::TryDoingMeleeAttack` | `0x91ff79`, `0x920152` |
| `CUserLocal::HandleCtrlKeyDown` | `0x932a5d` |
| `CUserLocal::SetDamaged` | `0x935236` |
| `CUserLocal::DoActiveSkill` | `0x947009` |
| `CWvsContext::SendActivatePetRequest` | `0x9f6c1c` |

Every one is a combat, skill-range, mastery, pet or character-delete path. None is a
level-cap comparison. A listing-wide regex sweep for `cygnus|blockedJob|maxLevel|levelLimit`
returned only that helper and `SKILLENTRY::GetMaxLevel` (`0x50a020`), which is a **skill**
level ceiling read from `SKILLENTRY+0x70`, not a character level cap.

Result for this leg: **negative, and now actually checked** — no Cygnus character-level cap is
reachable from the client's only Cygnus-classification helper, and no level-cap symbol or string
surfaced in a listing-wide sweep.

## Result

**Not confirmed.** No local WZ evidence establishes an explicit 120-level cap for
`job.TypeCygnus`; the closest hits are flavor-text milestones stopping at 120, which is a lead,
not a finding. Per the brief, `MaxLevelFor` and its Cygnus test rows are left unchanged (still
200 for all job lines). Both legs of Step 1 have now been searched and both are negative, so this is settled:
no local evidence supports a Cygnus cap below 200.

# Task 8 — `atlas-player-npcs` service scaffold and registration

Service scaffold and registration checklist landed (see task-8-report.md for the full list
of files touched). `tools/service-registration-guard.sh` exits 0, both `deploy/k8s/overlays/pr`
and `deploy/k8s/overlays/main` render clean via `kubectl kustomize`, and `go build ./...` from
`services/atlas-player-npcs/atlas.com/player-npcs` exits 0.

## Operator hand-backs

Two checklist steps this task cannot perform and must be actioned by the operator before the
service can run end-to-end:

1. **§6.1 — create the database.** Create `atlas-player-npcs-main` on `postgres.home` (the same
   Postgres instance/role the other `atlas-*-main` service databases live on). This task only
   registered `atlas-player-npcs` in `tools/db-bootstrap.sh`'s `DBS` list and the `main`/`pr`
   overlay `DB_NAME` patches — none of that creates the database itself on the `main`
   environment's Postgres instance.
2. **§6b — flip the GHCR package public.** After the first `atlas-player-npcs` image is pushed
   to `ghcr.io/chronicle20/atlas-player-npcs/atlas-player-npcs`, the package will be private by
   default (GHCR default for a repo's first push of a new image name). Flip it to public in the
   GitHub Packages UI, matching every other `atlas-*` image, or the cluster's anonymous pull
   will fail.

# Task 10 — `allocation/` script-id pool

Landed `63cd38e46` (implementation) + `288586406` (version-aware Pirate branch). Gate 14 PASS
at `288586406`; Task 10 review APPROVED, 0 blocking, 0 non-blocking.

## Controller rulings carried forward

1. **GM branch formula is literal.** PRD FR-3.3 is `26 + 4*(mapId/100000000)` with 400-id
   continent blocks starting at 9902600, so continent 1 → branch **30** and continent 2 →
   branch **34**. The plan's Task 10 `expect` column read 27/30, which contradicted its own
   shown derivation; the implementer flagged it and the table was corrected in `6701fbe33`.
   Tasks 12/15 inherit 30/34.
2. **`Allocate`'s global fallback needs no second validation.** It scans the same validated
   `usable` set (design D-1), so a fallback-allocated id is exactly as safe as a
   branch-allocated one — no extra check, no distinct error path.

## Signature change Tasks 12/15 must consume

The `skill/job id guard` rejected the original `BranchFor`, exactly as it rejected Task 9's
routing. The fix mirrors Task 9's `routing.HallOfFameMapFor` (`66071a227`) byte for byte:
resolve through the version-aware Identity before the raw category switch.

```
func BranchFor(set constants.SkillJobSet, jobId job.Id, mapId _map.Id) uint32
```

This is now the real contract — the plan's older two-argument form is superseded. This module
has tripped that guard on two consecutive tasks, so any later task comparing a `job.*Id`
constant should resolve through `constants.SkillJobSet` rather than comparing raw.

# Task 11 — `position/` grid and podium positioners

Landed `14300ea56`. Gate 15 PASS at `14300ea56`; review APPROVED_WITH_FINDINGS — 0 blocking,
2 non-blocking, both Task 15-facing.

The implementer designed `Placement`'s field set and `RaisePodiumStep` itself: the brief's test
table required them but its Step-2 signature list never named them. The reviewer read plan
Task 15 and confirmed the invented shapes are functionally sufficient for Task 15's grid and
podium reorganization paths — `Reorganize` reads only `Placement.ScriptId`, never `.Rect` or
`.Point`, so Task 15 can build its input from a bare script-id list. Not a redesign trigger.

## Two rulings for Task 15

1. **`Placement.ScriptId` must become `uint32`.** It is currently `int32`
   (`position/types.go:37`), while `allocation.Allocate` (`allocation/allocation.go:116`) and
   the entire script-id pool are `uint32`. No overflow risk — the values fit — but as it stands
   Task 15 needs an unenforced cast at the boundary. Align the type rather than carrying the
   cast forward. **This fix is pending and should land before Task 15 consumes it.**
2. **`Placement.Step` is computed but not persisted.** Plan Task 12 and design §3.1 list only
   `x, cy, fh, rx0, rx1, dir`, so Task 15 computes `Step` and discards it, and has no persisted
   "current step" to read for a *new* deploy. Task 15 must therefore derive the step by retrying
   `NextGridPosition` from step 0 — a mechanism no artifact names yet. Task 15 must either
   implement that derivation explicitly or add the column; decide there, and write down which.

# Task 12 — `playernpc/` persistence layer

Landed `4afd98e44`. Gate 16 PASS at `4afd98e44` — 106 tests across 5 packages, output pristine.
Entity, model, builder, administrator and provider, copied from the
`services/atlas-notes/atlas.com/notes/note/` house shape.

## Flagged for verification, beyond this task

The implementer reports that the test DSN other Atlas services use, `_pragma=foreign_keys(1)`,
is glebarez/modernc syntax and **silently does not enable foreign-key enforcement** under
`mattn/go-sqlite3` — which is the driver this module actually resolves to via
`gorm.io/driver/sqlite`. It used `_foreign_keys=1` instead and documented why in
`testDatabase`'s comment. If the claim holds, other services' FK-constraint tests are not
asserting what they appear to, which is a finding wider than task-251. The Task 12 reviewer was
asked to confirm the driver and the claim rather than take it on trust — read
`.superpowers/sdd/plan/task-12-review.md` for its verdict before acting on this.

# Task 11 follow-up — `Placement.ScriptId` aligned to `uint32`

Landed `5dd60245a`, closing non-blocking finding 2 of the Task 11 review. `position` tests
22/22. Task 15 no longer needs an unenforced cast at the `allocation.Allocate` boundary.

Ruling 2 of the Task 11 block (`Placement.Step` computed but not persisted) is **still open**
and is Task 15's to decide.

## Task 12 review — APPROVED_WITH_FINDINGS, 0 blocking

Three non-blocking findings (`.superpowers/sdd/plan/task-12-review.md`):

1. `playernpc/model_test.go` — round-trip tests never assert `EquipmentModel.Id()` survives
   (only `Slot`/`ItemId`). Test-coverage gap, not a production defect.
2. `playernpc/administrator.go:34-35` — `createPlayerNpc` re-fetches the row in a second DB
   round trip after commit instead of building the Model from the in-memory entity/equipment it
   already holds with assigned ids. Correct, just extra I/O.
3. **The SQLite FK-pragma claim is confirmed, and it is repo-wide.** The reviewer verified it
   independently — `go list -m` to resolve the driver, plus an empirical `PRAGMA foreign_keys`
   probe against `mattn/go-sqlite3` — rather than taking the implementer's word. Six
   `services/atlas-data/atlas.com/data/{npc,reactor,searchindex,monster,map,item}` test files
   use `_pragma=foreign_keys(1)`, which silently does **not** enable FK enforcement under this
   repo's actual driver. Those tests are not asserting what they appear to. Out of scope for
   task-251 — **worth its own follow-up ticket**, and it should not be lost when this branch
   merges.

Reviewer note: it reported being unable to find `prd.md`/`design.md`, so PRD §6 and design §3.1
/§8 were checked only against the brief's verbatim quotes (which matched field-for-field). Both
files do exist at `docs/tasks/task-251-player-npcs/` — this was a reviewer lookup miss, not a
missing artifact. A later reviewer should read them directly.

# Task 13 — read clients

Landed `6c373b7d9`. Gate 18 PASS at `6c373b7d9` (last gated commit). Review
APPROVED_WITH_FINDINGS — 1 blocking, 2 non-blocking
(`.superpowers/sdd/plan/task-13-review.md`).

**Blocking, fix queued behind Task 14** (only one mutating agent may hold the worktree):
`data/map/{model.go:20,processor.go:15-16+29,requests.go:20+28}` types the map id as raw
`uint32` instead of `_map.Id` from `libs/atlas-constants/map`. Every other `data/map` client
in the repo (atlas-channel, atlas-doors, atlas-messages) and this service's own
`routing`/`allocation` packages use `_map.Id`. The Task 13 report's self-review wrongly
claimed `_map.Id` was already reused here.

Non-blocking: the report's sweep claim ("every `SetReferencedStructs` in the repo is a dead
stub") is false — `services/atlas-npc-shops/atlas.com/npc/inventory/rest.go:82-96` is a real
decoder. The narrower true claim (the brief's *cited* precedents are stubs) would have
sufficed and does not change the shipped code.

## Open cross-service question — equipment capture

`atlas-character` does not serve an equipment/inventory relationship today; the reviewer
confirmed the seam independently. PRD/design both assumed `?include=inventory` works. Task 13
shipped a grounded decode shape (modelled on `atlas-channel/asset`) that will decode correctly
once a server side exists, but **nothing populates it now**, so Task 14's snapshot will capture
zero equipment against a live `atlas-character`. Needs a ruling: extend `atlas-character` on
this branch, split it to its own task, or ship Player NPCs without equipment for now.

# Task 14 — `snapshot/` and `eligibility/`

Landed `0d8a8257b`. Gate 19 PASS at `0d8a8257b` (last gated commit). Review **APPROVED** —
0 blocking, 1 cosmetic non-blocking (`snapshot.go:44-51` builds `equipmentPositions` via an
IIFE var rather than an init function).

The reviewer traced every load-bearing claim to source rather than to the brief: the
cash/masked slot convention (`cash := s < -100; s += 100`) against
`atlas-channel/character/model.go:317-343` and `socket/model/avatar.go:10-31`; the FR-5.2
1–11 / 101–111 boundary against `libs/atlas-constants/inventory/slot`; `112 → 100` job-category
derivation; the FR-6.1 predicate gating; and `overall_rank == world_rank` as design D-3's
deliberate single-source assignment, not a fabricated cross-world value.

# Controller ruling — equipment comes from `atlas-inventory`

The Task 13 "atlas-character doesn't serve equipment" concern was a **wrong premise on both
sides**, and the earlier entry in this file overstated it. `atlas-character`'s
`?include=inventory` is not how anything fetches equipment. `atlas-channel` and `atlas-login`
both call the separate **`atlas-inventory`** service:

- `services/atlas-channel/atlas.com/channel/inventory/requests.go:11,16` — `characters/%d/inventory`, root url key `INVENTORY`
- `services/atlas-channel/atlas.com/channel/compartment/model.go` — compartments carry `[]asset.Model`
- `services/atlas-channel/atlas.com/channel/asset/rest.go:10-12` — assets carry `slot` (`int16`) and `templateId` (`uint32`)
- `services/atlas-login/atlas.com/login/inventory/requests.go:11,16` — the same client again

So there is **no missing server-side work and no follow-up ticket** — Task 13 simply copied the
wrong client. Fixed on this branch in the round below. Task 14's slot arithmetic is correct
regardless of the source and does not change.

Fix round brief: `.superpowers/sdd/plan/task-13-brief-fix.md` — the `_map.Id` blocker from the
Task 13 review plus the inventory rewire, in one commit range.

# Task 13/14 fix round — closed

`cc6851b6e` (Fix 1, `_map.Id` retype) + `7f26d416c` (Fix 2, equipment from `atlas-inventory`).
Gate 20 PASS at `7f26d416c` — **last gated commit**. Re-review **APPROVED**, 0 blocking,
0 non-blocking (`.superpowers/sdd/plan/task-13-fix-review.md`).

The reviewer diffed `snapshot.Capture`/`captureEquipment` byte-for-byte pre- vs post-fix and
confirmed the masking arithmetic and the 1–11 / 101–111 range gate did **not** move — only the
parameter type and call site changed, which was the regression risk in this round. It also
verified the fix report's registration claim rather than taking it: `characters/{id}/inventory`
is already routed to `atlas-inventory` at `deploy/k8s/base/routes.conf.template.generated:135-138`,
`atlas-player-npcs.yaml` carries no per-domain `_SERVICE_URL` override for any of its domains,
and `RootUrlFor` resolves purely by env lookup — so no deploy change was needed and none is
missing.

## New signature Task 15 consumes

```
snapshot.Capture(characterId uint32, worldId world.Id,
                 cp character.Processor, ip inventory.Processor, rp ranking.Processor) (Model, error)
```

Call site needs `inventory.NewProcessor(l, ctx)`.

# Session 5 handoff (after Tasks 13, 14 and the fix round)

**Last gated commit: `7f26d416c`** (gate 20 PASS). Tasks 1–14 are complete, reviewed and gated.
Nothing is in flight; the tree is clean.

**NEXT ACTION: Task 15 (the deploy transaction, plan.md:1037).** Its brief is already
generated, fact-blocked at base `7f26d416c`, and annotated with the three rulings it must
consume: `.superpowers/sdd/plan/task-15-brief.md`. Dispatch an `atlas-implementer` (sonnet)
against it — no brief regeneration needed.

Open decision Task 15 owns: **`Placement.Step` is computed but not persisted.** Design §3.1
persists only `x, cy, fh, rx0, rx1, dir`, so a new deploy has no stored "current step". Task 15
must either derive the step by retrying `NextGridPosition` from step 0, or add the column —
and write down which.

Standing operating rules on this branch, unchanged:
 - Only ONE mutating agent may hold the worktree at a time; a gate may never run alongside a
   writer. A read-only reviewer MAY run alongside one writer, but only with the explicit
   no-mutation constraint block.
 - Treat plan.md's TABLES as authoritative and its PROSE as suspect — six defects found and
   fixed so far. Verify every named symbol against source before use.
 - The `skill/job id guard` has tripped twice on this module: resolve job ids through the
   version-aware `constants.SkillJobSet`, never a raw wire id against a `job.*Id` constant.

Two operator hand-backs still outstanding (unchanged, recorded above under Task 8): create
`atlas-player-npcs-main` on `postgres.home`, and flip the GHCR package public after the first
image push.

# Task 15 — the deploy transaction

Landed `811c296af` (12/12 `TestDeploy` subtests). Gate 21 FAILED on one staticcheck QF1008
selector nit in `playernpc/administrator.go:110`; everything else in that gate passed.
Fixed in `3c5a3fd82`. **Gate 22 PASS at `3c5a3fd82` — last gated commit.**

`playernpc.Processor` composes Tasks 9–14 into the design §8.1 transaction and is the
contract Tasks 16/17/21 build against:

```
Deploy(characterId uint32, worldId world.Id, mapId _map.Id, enforceEligibility bool, explicit *Position) (Model, error)
Redeploy(id uuid.UUID) (Model, error)
RemoveById(id uuid.UUID) (Model, error)
Remove(characterId uint32, mapId *_map.Id) ([]Model, error)
GetById(id uuid.UUID) (Model, error)
GetByMap(worldId world.Id, mapId _map.Id, page model.Page) ([]Model, error)

NewProcessor(l, ctx, db, cp character.Processor, ip inventory.Processor,
             rp ranking.Processor, cfgp configuration.Processor,
             np npcdata.Processor, mp mapdata.Processor, emit EventEmitter) Processor
```

Two things the implementer flagged, carried into the Task 16 brief:

- `resolvePodiumPosition` reads design §5.2's "rank" as `world_job_rank - 1`. That is the
  implementer's own inference — §5.2 does not say which counter feeds it — grounded in podium
  maps routing 1:1 with a single job branch, but not table-verified. Worth re-checking when
  Task 16/19 exercise a live podium map.
- Task 16's REST body must be confirmed against `Position`'s two fields before wiring.

**Test-fixture gotcha worth keeping:** `102000004` is `_map.VictoriaRoadHallOfWarriors1Id`, a
real Hall of Fame *and* podium map. Task 12's `administrator_test.go` uses it harmlessly for
persistence-only fixtures, but reusing it by habit elsewhere produces confusing
podium-dispatch failures. `processor_test.go` uses `555000004` — outside both `hallOfFameMaps`
and `podiumMaps` — for grid-path tests.

Task 15's review was still in flight when Task 16 was dispatched; its verdict is recorded
below when it lands.

## Task 15 review — APPROVED_WITH_FINDINGS, 1 blocking

`.superpowers/sdd/plan/task-15-review.md`. The reviewer traced the deploy transaction,
event-after-commit discipline, `Redeploy`'s non-touching of position, the prior fix round's
signatures (`snapshot.Capture`, `_map.Id`, `Placement.ScriptId uint32`) and job-id typing
(via `constants.SkillJobSet`, never a raw wire comparison) to source — all check out. It also
confirmed the **`Step` column decision is complete via GORM `AutoMigrate`**, not a
hand-migration gap.

**Blocking:** `processor.go:414-469` — the podium deploy path (`resolvePodiumPosition` /
`podiumRank`) ships the implementer's admitted-unverified `world_job_rank - 1` reading of
design §5.2 with **zero test coverage**; every `TestDeploy` subtest uses the deliberately
non-podium `555000004`. Fix brief: `.superpowers/sdd/plan/task-15-brief-fix.md` — settle the
semantics against source *and* pin them with a podium subtest.

**Non-blocking, accepted not fixed — carried forward as a Task 21 constraint:**
`processor.go:567-581` bulk `Remove` runs N separate transactions/emits rather than one atomic
operation. That matches design's per-NPC REMOVED shape, so the code stands, but **Task 21's GM
handler must treat a mid-loop failure as a partial success, not all-or-nothing.**

# Task 16 — REST resource

Landed `a24feca93` (`playernpc/{rest.go,resource.go,requests.go,resource_test.go}` new,
`main.go` edited; `TestPlayerNpcResource` 17/17). Not yet gated — its commit joins the next
gate's range together with the Task 15 fix.

Two implementer concerns handed to the Task 16 reviewer:

- The eligibility endpoint defaults `worldId` to `0`. Design's literal signature omits it but
  the duplicate check needs it, so this is the implementer's inference — needs a ruling on
  whether it silently checks the wrong world.
- No `.bruno/` collection entry was added; the implementer argues that file belongs to Task 8's
  inventory, not Task 16's. Reviewer to check the convention.

## Task 15 fix round — closed

`92be6a7cc`. The implementer **verified** the `world_job_rank - 1` reading rather than
re-asserting it: every podium map routes 1:1 to a single job category (`routing.go`
`HallOfFameMapFor`) and `nextWorldJobRank` is a gapless per-`(world, job category)` `MAX+1`
counter, so it lines up exactly with `PodiumPosition`'s 0-based rank. **No arithmetic changed**
— only the doc comment, which now cites the evidence instead of admitting it was unverified. A
new `podium deploy` subtest deploys 3 NPCs to `VictoriaRoadHallOfWarriors1Id` and pins the
non-raise, raise+reposition and new-slot branches (`TestDeploy` 13/13).

## Task 16 review — CHANGES_REQUIRED, 1 blocking

`.superpowers/sdd/plan/task-16-review.md`. The Task 15 `Position` seam checks out
(`PositionRestModel{X,Y int16}` vs `Position{X,Y int16}`), as does `Deploy(..., true, explicit)`
pass-through, backed by a non-vacuous assertion.

**Blocking:** `rest.go:496` — `handleGetEligibility` hardcodes `conversationPath=false`, so the
tenant's `AutoDeployEnabled` setting is **never** consulted. Design §9.1 defines this endpoint's
predicate as including that clause, and it only applies when `conversationPath=true`
(`eligibility/eligibility.go:24-30`). The conversation engine's `canSpawnPlayerNpc` is that
flag's sole intended caller, so as shipped it contradicts the single-predicate guarantee §9.1
gives as the reason for the shared endpoint. The existing subtest is **vacuous on this axis** —
it never varies `AutoDeployEnabled` — so the fix needs a corrected fixture and genuinely varying
coverage, not a one-line flip. Fix brief: `.superpowers/sdd/plan/task-16-brief-fix.md`, which
also sweeps in the missing per-endpoint `.bruno/*.bru` files (non-blocking 3 — every sibling
REST service has them; player-npcs had only Task 8's scaffolding).

### Constraint for Task 22 — the eligibility endpoint's `worldId`

Review non-blocking 2 is a **real, live defect that Task 16 cannot fix**. Design §9.1's literal
signature omits `worldId`, but `countByName` (`administrator.go:119`) is scoped by
`(world_id, map_id, name)`, matching PRD §6's `(tenant, world, map, name)` unique index.
`character.Model` carries no world field, so the handler defaults to `0` — silently wrong for
any non-zero world (a false negative or a false positive, never an error).

The endpoint **already accepts `worldId` optionally**, so the resolution is caller-side:
**Task 22's conversation-engine condition MUST pass `worldId` explicitly.** Recorded here as a
Task 22 constraint rather than an external follow-up ticket, because it is producible on this
branch.

## Task 16 lint fix — `f75b7e0e0`

Gate 23 FAILED on 7 `errcheck` `Body.Close()` findings. The implementer found the gate log was
**under-reporting**: golangci-lint's default `max-same-issues: 3` cap hid 9 more identical hits
in the same file. It fixed all **16** occurrences rather than the 7 quoted, so the next
`--quick` run does not fail again on a newly un-capped batch. Worth remembering when reading any
future lint block on this branch: **the quoted count is a floor, not a total.**
