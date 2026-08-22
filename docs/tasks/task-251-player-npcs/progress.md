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

## Task 16 fix round — `d908c3c95`

`rest.go:496` `conversationPath` `false` → `true`; the stale "eligibility endpoint" fixture
expectation corrected (now ineligible under `DefaultModel()`'s `autoDeployEnabled: true`
fallback); a new subtest mocks `TENANTS_SERVICE_URL` to vary `AutoDeployEnabled` across both
values and asserts the verdict tracks it; 7 `.bruno/*.bru` request files added, one per route,
on the `services/atlas-notes/.bruno/` convention.

## Task 15/16 combined fix review — APPROVED, 0 findings

`.superpowers/sdd/plan/task-15-16-fix-review.md`, over `92be6a7cc` + `f75b7e0e0` + `d908c3c95`.
The reviewer did not take the fix reports' claims — it re-derived them:

- `routing/routing.go:26-46,71-104` — each podium map really is 1:1 with one job category.
- `playernpc/builder.go:109-111` — the stored `job_id` column holds `routing.JobCategory`,
  which is what `nextWorldJobRank` queries by. That is the link that makes `world_job_rank - 1`
  correct.
- `processor_test.go:503-587` — **hand-verified all 5 podium slot assertions independently**;
  all match, so the new test is not back-filled from observed output.
- `resource_test.go:684-742` — traced the new subtest against *both* the old and new
  `conversationPath` values to confirm it is genuinely non-vacuous.
- `.bruno/*.bru` — mapped 1:1 against `rest.go:173-183`'s registered routes.

Both blocking findings are closed. The `worldId` gap was correctly left untouched.

## Task 16 lint follow-up — `1e4ac4f53`

Gate 24's one staticcheck QF1012 (`Write([]byte(fmt.Sprintf(...)))` in the new test mock),
closed. `TestPlayerNpcResource` now 18 subtests + 2 `AutoDeployEnabled` variants, all passing.

**Tooling note worth keeping:** bare `tools/lint.sh <path>` runs *both* ecosystems tree-wide
and does not scope to the path. Use `tools/lint.sh --go <module>` to actually lint just the Go
module. The first agent on this fix also stalled by backgrounding its lint run and ending its
turn waiting for a notification that only ever goes to the controller — run lint synchronously.

# Session 6 handoff (after Tasks 15 and 16, both fix rounds closed)

Tasks **1–16 are complete and reviewed**. Six tasks remain: **17–22**.

**Gate 25 was still in flight at handoff** — `tools/verify.sh --quick --base 3c5a3fd82`,
log at `.superpowers/sdd/plan/gates/gate-25.log`. It covers `3c5a3fd82..HEAD`, i.e. Task 16
(`a24feca93`), the Task 15 podium fix (`92be6a7cc`), both lint rounds (`f75b7e0e0`,
`1e4ac4f53`) and the Task 16 eligibility fix (`d908c3c95`). **Read that log first and ledger
its verdict before dispatching anything.** Last *confirmed* gated commit is `3c5a3fd82`
(gate 22 PASS); gates 23 and 24 both FAILED on lint only and were fixed.

**NEXT ACTION: Task 17 (Kafka — messages, producer, consumers, plan.md:1161).** Its brief is
already generated, fact-blocked, and annotated with the processor contract, the
partial-success constraint and the fixture gotcha: `.superpowers/sdd/plan/task-17-brief.md`.
Dispatch an `atlas-implementer` (sonnet) against it — no brief regeneration needed.

Standing operating rules on this branch, unchanged:

- Only ONE mutating agent may hold the worktree at a time; a gate may never run alongside a
  writer. A read-only reviewer MAY run alongside one writer, but only with an explicit
  no-mutation constraint block, and it must review the *commit range* rather than the moving
  working tree.
- Treat plan.md's TABLES as authoritative and its PROSE as suspect — six defects found and
  fixed so far. Verify every named symbol against source before use.
- The `skill/job id guard` has tripped twice on this module: resolve job ids through the
  version-aware `constants.SkillJobSet`, never a raw wire id against a `job.*Id` constant.
- **A lint block's quoted finding count is a floor, not a total** — golangci-lint's
  `max-same-issues: 3` capped one round at 7 when there were really 16. Always drive
  `tools/lint.sh --go <module>` to exit 0 rather than fixing only the quoted lines.

Two constraints later tasks must consume:

- **Task 21** — bulk `Remove` (`processor.go:567-581`) is N transactions and N emits, not one
  atomic operation. The GM handler must treat a mid-loop failure as a **partial success**.
- **Task 22** — the eligibility endpoint defaults `worldId` to `0`, which silently mis-scopes
  the duplicate check for any non-zero world. The endpoint already accepts `worldId`
  optionally, so the conversation-engine condition **must pass it explicitly**.

Two operator hand-backs still outstanding (unchanged, recorded under Task 8): create
`atlas-player-npcs-main` on `postgres.home`, and flip the GHCR package public after the first
image push.

# Gate 25 — PASS at `1e4ac4f53`

Reconciled at the start of session 7. `tools/verify.sh --quick --base 3c5a3fd82` covered
`3c5a3fd82..1e4ac4f53` — Task 16 (`a24feca93`), the Task 15 podium fix (`92be6a7cc`), both lint
rounds (`f75b7e0e0`, `1e4ac4f53`) and the Task 16 eligibility fix (`d908c3c95`). All 8 selected
gates green (go build/vet, analyzer guards, skill/job id, scope, producer seam, env domain, env
bootstrap, lint & format). **Last gated commit: `1e4ac4f53`.**

# Task 17 — Kafka messages, consumers, producer

Landed `4fc4a4ec7` (10 files, 1355 insertions): `kafka/message/{playernpc,character}/kafka.go`,
`kafka/consumer/{playernpc,character}/consumer.go` + tests, `playernpc/producer.go` + test,
`kafka/consumer/consumer.go`, `main.go` wiring. `TestPlayerNpcCommandConsumer` 7/7,
`TestLevelChangedConsumer` 7/7, `TestNewEmitter` 4/4; `go vet ./...` clean.

**Gate 26 FAILED** on one staticcheck ST1005 (capitalized error string,
`playernpc/producer.go:79`); every other gate in that run passed. Fix round dispatched back to
the same implementer with the "quoted count is a floor" instruction. Last *confirmed* gated
commit remains `1e4ac4f53`.

Three implementer deviations handed to the Task 17 reviewer rather than accepted:

1. REDEPLOY's command body gained a `WorldId` field beyond plan.md's literal `(characterId,
   mapId)` table — justified as forced by `Processor.Redeploy` resolving via
   `GetByMap(worldId, mapId, page)` with no narrower lookup exposed. Reviewer to verify the
   claim against `processor.go`/`administrator.go`.
2. LEVEL_CHANGED's "fetch fails" log line omits the target map (design §8.2 implies it) because
   the map derives from the job id in the very fetch that failed. Reviewer to rule forced vs
   convenient.
3. `kafka/consumer/consumer.go` (local `NewConfig` wrapper) added outside the brief's file list
   as required plumbing. Reviewer to confirm it matches the sibling-service shape.

**Brief-vs-source discrepancy for Task 18:** the Task 18 brief says to copy
`services/atlas-channel/atlas.com/channel/kite/`'s "neighbouring `rest.go` test". That package
has **no `rest_test.go`** — its only test is `processor_drain_test.go`. The client *shape*
(`model/builder/rest/requests/processor.go`) is there to copy; the test shape must come from
another channel read client.

## Task 17 lint fix — PARTIAL hand-back

The Task 17 implementer reported **PARTIAL** on the lint round: it diagnosed the ST1005 finding
but hit the 120 tool-call cap before committing the fix or finishing the sweep for sibling
capitalized error strings. Task 17's implementation commit `4fc4a4ec7` is unaffected and stands.

Continuation brief: `.superpowers/sdd/plan/task-17-brief-cont.md`; a fresh `atlas-implementer`
was dispatched against it with the same report file as persistent memory. This is the **first**
PARTIAL on Task 17 — a second would mean the plan under-decomposed it, but a lint round handed
back at the cap is the designed outcome, not a sizing signal.

## Task 17 lint fix — closed

`78b6026fc`. Lowercased the ST1005 error string at `playernpc/producer.go:79`.
`tools/lint.sh --go services/atlas-player-npcs/atlas.com/player-npcs` → `0 issues. lint.sh: OK`;
module `go build ./... && go test ./...` green. The sweep for sibling capitalized error strings
found none — this round the quoted count really was the total, unlike the earlier 7-vs-16 round.
Gate 27 launched over `1e4ac4f53..78b6026fc`.

## Task 17 review — APPROVED, 0 blocking

`.superpowers/sdd/plan/task-17-review.md`, over `1e4ac4f53..4fc4a4ec7`. The reviewer traced all
three implementer deviations to source rather than accepting them:

1. **REDEPLOY's added `WorldId` stands.** `Processor.Redeploy` takes only a `uuid.UUID`
   (`processor.go:514`), and the one lookup resolving `(characterId, mapId)` without a world —
   `entitiesByCharacter` (`administrator.go:151`) — is package-private and reachable only from
   `Remove`/`RemoveById`, not from the consumer package. Exported `GetByMap` requires `worldId`.
   Given the plan's constraint against changing exported `Processor` signatures, this was the
   only viable path; it is documented on the wire type.
2. **The LEVEL_CHANGED log omission is forced, not convenient.** `targetMapId` derives from
   `c.JobId()`, which comes from the very fetch that failed. Every *other* failure branch in
   that handler does log the target map.
3. **`kafka/consumer/consumer.go` is the standard wrapper** — diffed byte-for-byte against
   `services/atlas-notes/.../kafka/consumer/consumer.go`, identical but for a doc comment, and
   present in 58 other Atlas services.

Also independently confirmed: event-after-commit discipline intact (emit only from the existing
post-`ExecuteTransaction` call sites), bulk `Remove` really emits N REMOVED events, job ids route
through typed `job.Id`/`constants.SkillJobSet` with no raw wire comparison, and both new topic
env vars are already registered in `deploy/k8s/base/env-configmap.yaml`.

Non-blocking (1, accepted): `kafka/consumer/character/consumer_test.go:283-289` — the "fetch
fails" subtest asserts the log message is non-empty rather than checking a structured field. The
subtest's primary assertions (0 emitted commands, WarnLevel) are real, and the weakness is
self-documented in the test comment.

# Gate 27 — PASS at `78b6026fc`

`tools/verify.sh --quick --base 1e4ac4f53` over `1e4ac4f53..78b6026fc` (Task 17 `4fc4a4ec7` plus
the lint fix). All 8 selected gates green. **Last gated commit: `78b6026fc`.** Task 17 is
complete, reviewed (APPROVED) and gated.

# Task 18 — `atlas-channel` `playernpc/` read client

Landed `897c828c2` (6 new files under `services/atlas-channel/atlas.com/channel/playernpc/`).
`TestRestModel_Unmarshal`, `TestExtract_EquipmentOrder`, `TestForEachInMap_RequestsByMapAndWorld`
3/3; module `go build ./... && go test ./...` clean. Gate 28 launched over `78b6026fc..897c828c2`.

Reported **DONE_WITH_CONCERNS**: the implementer did not follow test-first ordering — the tests
and implementation were written together with no captured RED — and said so rather than
fabricating a transcript. The risk that creates is a back-filled test, so the Task 18 reviewer
was directed to check all three tests against the **producing** side
(`services/atlas-player-npcs/.../playernpc/{rest.go,resource.go}`) rather than against the client
that produced them. Its verdict is recorded below when it lands.

It also reports no deploy/routing change was needed — `/api/player-npcs` already present in
`deploy/shared/routes.conf` and `deploy/compose/routes.conf` from an earlier task. Handed to the
reviewer to verify rather than accepted.

# Gate 28 — PASS at `897c828c2`

`tools/verify.sh --quick --base 78b6026fc` over `78b6026fc..897c828c2` (Task 18). All selected
gates green. **Last gated commit: `897c828c2`.**

## Task 18 review — APPROVED_WITH_FINDINGS, 0 blocking

`.superpowers/sdd/plan/task-18-review.md`. The no-RED concern is **resolved**: the reviewer
checked the tests against the producing side rather than against the client that generated them.

- `playernpc/rest.go`'s `RestModel`/`EquipmentRestModel` compared field-for-field against
  `services/atlas-player-npcs/.../playernpc/resource.go` — identical JSON tags in identical
  order, `overallJobRank` included. `world.Id`/`_map.Id`/`job.Id` are plain `byte`/`uint32`/
  `uint16` definitions with **no** `MarshalJSON`/`UnmarshalJSON` override, so they encode
  byte-identically to the producer's raw wire types. The tests assert the real server contract,
  not a self-consistent round-trip.
- The filter string (`requests.go:15`) matches the producer's `parseListFilters` param names and
  integer widths exactly.
- The no-deploy-change claim **verified**: `deploy/shared/routes.conf:561` and
  `deploy/compose/routes.conf:561` already route `/api/player-npcs`, and `git diff --stat --
  deploy/` is empty for the range. `RootUrlFor(ctx, "PLAYER_NPCS")` needs no per-domain
  registration — same as `kite/`'s `RootUrlFor(ctx, "KITES")`.
- `tools/skill-job-id-guard.sh` on the new package: 0 findings. The client only forwards job ids,
  so it correctly needs no `constants.SkillJobSet` resolution.

**Non-blocking (1) + not-evaluable (1) — one issue, worth a follow-up beyond this task:**
`rest_test.go:157`'s `TestExtract_EquipmentOrder` builds `RestModel` directly instead of going
through `jsonapi.Unmarshal`, so the nested `equipment` attribute's real JSON:API decode with
non-empty data is never exercised. **This is a repo-wide gap, not a Task 18 defect** — the
producer's own `resource_test.go:183` has it identically. Consequence: api2go/jsonapi's decode
behaviour for non-empty nested struct-slice attributes is unverified on both sides of this seam.
Task 20 exercises the seam end-to-end and is the natural place to close it; if it does not, this
should not be lost at merge.

# Task 19 — `atlas-channel` spawn, broadcast, controller exclusion

Landed `7a3d648dd`. Module `go build ./... && go test ./...` from
`services/atlas-channel/atlas.com/channel` clean; `go vet` on touched packages clean. Files:
`kafka/consumer/map/{player_npc.go,player_npc_test.go}`, `kafka/consumer/map/consumer.go`,
`kafka/consumer/playernpc/{kafka.go,consumer.go,consumer_test.go}`, `main.go`. Gate 29 launched
over `897c828c2..7a3d648dd`.

One implementer concern handed to the reviewer rather than accepted: **REPOSITIONED handling
adds an extra `atlas-player-npcs` read-back** for appearance data not carried in the event body —
the implementer's own inference from the brief's test table. The reviewer was asked to rule both
halves: is the read-back *necessary* (check Task 17's envelope in
`kafka/message/playernpc/kafka.go` — if the data is already there, the read-back is waste), and
is it *correct* (scope, no N+1 per character or per channel, failure handling).

# Gate 29 — PASS at `7a3d648dd`

`tools/verify.sh --quick --base 897c828c2` over `897c828c2..7a3d648dd` (Task 19). All selected
gates green. **Last gated commit: `7a3d648dd`.**

# Session 7 handoff (after Tasks 17, 18 and 19)

Tasks **1–19 are complete and gated**. Three remain: **20, 21, 22**.

**Last gated commit: `7a3d648dd`** (gate 29 PASS). Tree is clean; no writer in flight.

**The Task 19 review was still running at handoff.** Dispatched `atlas-reviewer` over
`897c828c2..7a3d648dd`; its artifact will be at `.superpowers/sdd/plan/task-19-review.md`.
**Read that artifact first and ledger its verdict before dispatching anything.** Its load-bearing
question is the implementer's flagged REPOSITIONED read-back (see the Task 19 entry above); the
reviewer was also asked to rule on design D-4 controller exclusion, `SpawnNPC`-before-
`ImitatedNPCData` ordering under the `routine.Go` block, and N-spawn/one-`ImitatedNPCData`
batching.

**NEXT ACTION: Task 20 (`atlas-tenants` `player-npcs` configuration resource, plan.md).** Its
brief is already generated, fact-blocked at base `897c828c2`, and annotated:
`.superpowers/sdd/plan/task-20-brief.md`. Dispatch an `atlas-implementer` (sonnet) against it —
no brief regeneration needed.

Standing operating rules on this branch, unchanged:

- Only ONE mutating agent may hold the worktree at a time; a gate may never run alongside a
  writer. A read-only reviewer MAY run alongside one writer, but only with an explicit
  no-mutation constraint block, and it must review the *commit range* rather than the moving
  working tree.
- Treat plan.md's TABLES as authoritative and its PROSE as suspect — six defects found and fixed
  so far. Verify every named symbol and cited line number against source.
- The `skill/job id guard` has tripped twice on this module: resolve job ids through the
  version-aware `constants.SkillJobSet`, never a raw wire id against a `job.*Id` constant.
- A lint block's quoted finding count is a floor, not a total (`max-same-issues: 3` hid 9 of 16
  in one round). Drive `tools/lint.sh --go <module>` to exit 0; the bare form runs tree-wide.

Constraints later tasks must consume:

- **Task 20** — attribute names must match Task 13's already-shipped consuming client at
  `services/atlas-player-npcs/.../configuration/` field for field.
- **Task 21** — bulk `Remove` (`playernpc/processor.go`) is N transactions and N emits, not one
  atomic operation. The GM handler must treat a mid-loop failure as a **partial success**.
- **Task 22** — the eligibility endpoint defaults `worldId` to `0`, silently mis-scoping the
  duplicate check for any non-zero world. The endpoint already accepts `worldId` optionally, so
  the conversation-engine condition **must pass it explicitly**.

Open finding not owned by any remaining task: **the nested-`equipment` JSON:API decode gap**
(Task 18 review). Neither `services/atlas-channel/.../playernpc/rest_test.go:157` nor
`services/atlas-player-npcs/.../playernpc/resource_test.go:183` exercises api2go/jsonapi decoding
a non-empty nested struct-slice attribute — both build the model directly. Repo-wide pattern, not
a Task 18 defect, but it means that seam's decode is unverified on both sides. Close it before
the PR or record it as a deliberate carry.

Two operator hand-backs still outstanding (unchanged, recorded under Task 8): create
`atlas-player-npcs-main` on `postgres.home`, and flip the GHCR package public after the first
image push.

## Task 19 review — APPROVED, 0 blocking, 0 non-blocking

`.superpowers/sdd/plan/task-19-review.md`, over `897c828c2..7a3d648dd`. Scope confirmed across
all seven named files plus the read-only contracts they depend on (`npc/controller/processor.go`
and Task 17's `kafka/message/playernpc/kafka.go` envelope). No scope drift. The implementer's
flagged REPOSITIONED read-back produced no finding.

Two **not-evaluable** items the reviewer would not claim either way — both are honest scope
limits, not passes:

1. **Multi-pod "every channel of the world" broadcast.** The per-pod `IsWorld` gating is
   architecturally sound, but nothing in this unit's surface tests the behaviour across multiple
   channel pods. Live verification territory.
2. **Player-NPC vs ordinary-NPC object-id collision.** No path in this diff or its call sites
   feeds a raw id into `TryClaim`/`ElectFor` outside `ReleaseFor`'s registry filter, so the
   question is out of scope for this diff — but it was not affirmatively cleared either.

Neither blocks Task 19. Item 1 belongs to end-to-end validation once the service is deployed;
item 2 is worth a deliberate look during the pre-PR review rather than being treated as closed.

# Task 20 — `atlas-tenants` `player-npcs` configuration resource

**Commit `c08704867`**, "feat(atlas-tenants): add player-npcs configuration resource". Implementer
reported **DONE**, no concerns. Module-local `go build ./... && go test ./...` green from
`services/atlas-tenants/atlas.com/tenants`; `TestPlayerNpcConfigHandlerWireRoundTrip`
(create/get/update/delete/unconfigured/type-fidelity) passes; `gofmt -l` clean.

One deliberate deviation from the brief's file table: **`AreaSteps` is `int`, not `byte`**, to
match Task 13's already-shipped consumer field for field. Documented in
`.superpowers/sdd/plan/task-20-report.md`; handed to the Task 20 review as its load-bearing
question.

# Gate 30 — PASS at `c08704867`

`tools/verify.sh --quick --base 7a3d648dd`, exit 0. Selected gates green: go build/vet
(`services/atlas-tenants/atlas.com/tenants`), go analyzer guards, skill/job id guard, scope
guard, producer seam guard, env domain guard, lint & format guard (1 module). Docker bake
skipped as expected for `--quick`. **Last gated commit: `c08704867`.**

## Task 20 review — APPROVED, 0 blocking, 1 non-blocking

`.superpowers/sdd/plan/task-20-review.md`, over `c08704867`. The reviewer cross-checked the new
`services/atlas-tenants/atlas.com/tenants/configuration/` surface field for field against Task
13's shipped consumer (`services/atlas-player-npcs/.../configuration/{rest,model,requests}.go`)
and Task 16's `AutoDeployEnabled` usage. **The `AreaSteps` `int` deviation is confirmed correct** —
it matches the shipped consumer. No scope drift.

Non-blocking (report accuracy only, not code): `task-20-report.md`'s rationale for picking the
Rankings nested-attributes storage shape over KiteConfig's flat shape overstates the wire-contract
stakes — the wire shape is storage-shape-agnostic, produced uniformly by
`server.MarshalResponse`/`RegisterInputHandler` via struct-tag reflection. The code is correct.

Not-evaluable (an honest scope limit, not a pass): no cross-service integration test spins up
`atlas-tenants` and `atlas-player-npcs` together, and none is expected under this repo's
per-service conventions. The seam was instead verified by direct source comparison of both
sides' encode/decode contracts.

# Task 21 — `atlas-messages` `@playernpc` GM commands

**Commit `0142ff469`**, "feat(atlas-messages): add @playernpc GM commands (Task 21)". Implementer
reported **DONE_WITH_CONCERNS**. Module-local `go build ./... && go test ./...` green from
`services/atlas-messages/atlas.com/messages`; `command/playernpc` 12/12 subtests pass.

Two things handed to the Task 21 review:

1. **The failure-reporting path may not exist.** The implementer reports the shipped Task 17
   consumer in `atlas-player-npcs` has no synchronous reply or failure-event path back to
   `atlas-messages` — commands are fire-and-forget and downstream errors are logged and
   swallowed. The brief's "failure reported back — `pool_exhausted`" row could therefore only be
   satisfied by surfacing the *Kafka-publish-layer* error, not a real domain failure code. If
   confirmed, **FR-8.3 is not implementable without a status-event contract no task on this
   branch owns** — a branch-level finding, not a Task 21 defect. Task 22's FR-6.3 row depends on
   the same seam; its brief carries a note to verify before implementing.
2. **New test seams** — `characterLookup`, `commandPublisher`, `pinkTextSender` interfaces, added
   because no existing `command/*` package in this service mocks the Kafka/HTTP layer. Handed to
   review to rule on production-shaped seam vs. test-only scaffolding in production code.

# Gate 31 — PASS at `0142ff469`

`tools/verify.sh --quick --base c08704867`, exit 0. Selected gates green: go build/vet
(`services/atlas-messages/atlas.com/messages`), go analyzer guards, skill/job id guard, scope
guard, producer seam guard, env domain guard, env bootstrap guard, lint & format guard (1
module). Docker bake skipped as expected for `--quick`. **Last gated commit: `0142ff469`.**

## Task 21 review — APPROVED_WITH_FINDINGS, 1 blocking (branch-level, not a Task 21 defect)

`.superpowers/sdd/plan/task-21-review.md`, over `c08704867..0142ff469`. Diff matches the brief's
file list exactly; no scope drift. Task 21's own code is a faithful best-effort.

**BLOCKING — FR-8.3 is genuinely unimplementable as specified.** Task 17's shipped consumer
(`services/atlas-player-npcs/atlas.com/player-npcs/kafka/consumer/playernpc/consumer.go:66-82,
114-122`) only logs and swallows domain failures — there is no reply or status-event path, so no
domain failure code (`pool_exhausted` et al.) can ever reach `atlas-messages`. The implementer's
concern is **confirmed against source**. This cannot be fixed inside Task 21's four files.

`design.md:637-666` (§8.3/§9.1) has **Task 22 about to build a saga step on the identical false
assumption** (FR-6.3, "failure code surfaces"). Task 22's brief already carries a controller note
to verify this seam against shipped source and report the gap rather than fabricate a passing
test around a path that cannot fire.

Resolution is one of two, and it is a design decision, not a defect fix:

- **(A) Extend Task 17's contract** — add a status/failure event from `atlas-player-npcs` back to
  its command producers, and consume it in `atlas-messages` (FR-8.3) and
  `atlas-saga-orchestrator` (FR-6.3). Real work across three services; makes both requirements
  true as written.
- **(B) Formally narrow FR-8.3 and FR-6.3** in `prd.md`, `design.md` §8.3/§9.1 and the affected
  briefs to what fire-and-forget can actually deliver, and record the deferral.

**Escalated to the operator; the branch cannot be called done until one is chosen.**

Non-blocking (visibility only): the new `characterLookup`/`commandPublisher`/`pinkTextSender`
seams are new to `atlas-messages/command/*` but mirror Task 17's own `ProcessorProvider` seam
pattern (`consumer.go:33-37`) — not a `*_testhelpers.go` file, not a test-only exported
constructor, production wires real implementations inline. Acceptable.

# Operator ruling — FR-8.3 / FR-6.3: extend the Kafka contract

Presented with the Task 21 blocking finding, the operator chose **option (A): extend Task 17's
Kafka contract.** A status/failure event will be emitted by `atlas-player-npcs` back to its
command producers and consumed by `atlas-messages` (FR-8.3) and `atlas-saga-orchestrator`
(FR-6.3), making both requirements true as written. Option (B) — narrowing the requirements in
the PRD and design — was declined.

This is **a new plan task on this branch (Task 23), dispatched after Task 22 closes.** It is
explicitly NOT Task 22's work: Task 22 asserts the contract that exists today and leaves the
FR-6.3 gap documented in `saga/event_acceptance.go` so Task 23 can find it. Per CLAUDE.md, triage
and fix stay on the same branch; the clean PR branch comes from a rebase at PR time.

Scope Task 23 must cover:

- `atlas-player-npcs`: emit a status/failure event carrying the design §8.3 reason codes
  (`pool_exhausted` et al.) instead of logging and swallowing
  (`kafka/consumer/playernpc/consumer.go:66-82,114-122`).
- `atlas-messages`: consume it and surface the reason via `IssuePinkText` to the invoking GM
  (FR-8.3) — replacing Task 21's publish-layer-error-only best effort.
- `atlas-saga-orchestrator`: consume it so a step's failure status carries the code and a script
  can branch (FR-6.3) — which also makes the step no longer purely self-completing, so
  `saga/event_acceptance.go` changes with it.

# Task 22 — PARTIAL (commit `fcbe53c04`, cap reached)

"feat(task-22): canSpawnPlayerNpc condition and deploy_player_npc scaffolding". 134 tool calls;
committed what works and handed back per contract. **Not a failure.**

**Complete:** `libs/atlas-saga` (condition + action constants, `DeployPlayerNpcPayload`,
`unmarshal.go` case) and **FR-6.1 in full** — `atlas-query-aggregator`'s condition, the lazy
`GetPlayerNpcEligibility` accessor, the `playernpc/` eligibility client, and
`TestCanSpawnPlayerNpcCondition` (4/4 subtests). `atlas-saga-orchestrator` has scaffolding only
(re-exports, `kafka/message/playernpc`, the `playernpc/` location client).

**Remains:** FR-6.2's handler — `saga/handler.go` wiring and `handleDeployPlayerNpc`,
`saga/producer.go`'s `DeployPlayerNpcCommandProvider`, `TestDeployPlayerNpcAction`, and the three
documentation edits. Detail in `.superpowers/sdd/plan/task-22-report.md` §"What remains".

**Controller ruling — the `WorldId` source (option (a)).** `DeployPlayerNpcPayload` carries no
`WorldId` field and will not gain one. World and map both resolve through
`GetCurrentLocation(characterId)`; an explicit payload `MapId` still wins for the map, the lookup
supplies the world. This avoids touching `libs/atlas-saga` and both consuming services a second
time. Recorded in `.superpowers/sdd/plan/task-22-brief-cont.md`.

The implementer independently verified FR-6.3 unsatisfiable against the shipped consumer —
matching the Task 21 reviewer's finding from a separate context.

Continuation brief written: `.superpowers/sdd/plan/task-22-brief-cont.md`. The task review and
gate run once over the whole Task 22 range (`0142ff469`..final HEAD), not per segment.

# Gate 32 — PASS at `fcbe53c04`

`tools/verify.sh --quick --base 0142ff469`, exit 0. Docker bake skipped as expected for
`--quick`. **Last gated commit: `fcbe53c04`.** Note this gates the PARTIAL segment only; the
Task 22 review still runs once over the whole range once the continuation lands.

# Task 22 continuation — DONE (commit `c0b78bb9d`)

"feat(saga-orchestrator): wire deploy_player_npc handler (FR-6.2)". A fresh implementer took the
continuation brief; **DONE, no concerns**. Module-local `go build ./... && go test ./...` green
from `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`; `TestDeployPlayerNpcAction`
4/4 subtests pass; `gofmt -l` clean.

Landed: `saga/handler.go` (the handler + `HandlerImpl` wiring + dispatch case),
`saga/producer.go` (`DeployPlayerNpcCommandProvider`), `saga/handler_test.go`,
`playernpc/mock/mock.go` (new), and the three documentation edits
(`docs/npc_conversation_conversion_spec.md`, `services/atlas-query-aggregator/docs/rest.md`,
`services/atlas-query-aggregator/docs/domain.md`).

FR-6.3's failure-code row is **intentionally untested** per the confirmed ruling; the test file
documents the gap explicitly so Task 23 can find it.

Noted, not acted on: existing `With*` copy-constructors in `saga/handler.go` were already
inconsistent about which fields they preserve. Pre-existing, not introduced here; the implementer
followed local convention rather than sweeping it. Handed to the Task 22 review to confirm the
*new* field is preserved where it matters.

**Task 22 complete — all 22 plan tasks are now implemented.** The review runs once over the whole
range `0142ff469..c0b78bb9d` (both segments as one unit).

# Gate 33 — PASS at `c0b78bb9d`

`tools/verify.sh --quick --base fcbe53c04`, exit 0. Docker bake skipped as expected for
`--quick`. **Last gated commit: `c0b78bb9d`.**

## Task 22 review — APPROVED_WITH_FINDINGS, 0 blocking, 2 non-blocking

`.superpowers/sdd/plan/task-22-review.md`, over `0142ff469..c0b78bb9d` — both segments reviewed as
one unit, cross-checked against the shipped `atlas-player-npcs` REST endpoint and Kafka consumer.
All five load-bearing questions came back clean: `worldId` is passed explicitly (not defaulted to
`0`), an explicit payload `MapId` is not overwritten by the lookup, the saga path cannot emit
`EnforceEligibility: false`, the condition's six wiring points are all present with graceful
degradation, and FR-6.3 asserts the contract that exists rather than faking one.

Non-blocking, both small and both in files Task 23 will touch anyway — **fold them into Task 23**:

1. `services/atlas-saga-orchestrator/.../playernpc/mock/mock.go` — named `mock.go` where sibling
   packages use `processor.go`, and missing the
   `var _ playernpc.Processor = (*ProcessorMock)(nil)` compile-time assertion / `processor_test.go`
   most sibling `mock/` packages carry. Convention, not a defect.
2. `services/atlas-saga-orchestrator/.../kafka/message/playernpc/kafka.go:1-7` — the doc comment
   claims the type mirrors the shipped consumer "field for field," but the shipped
   `CommandDeployBody` has an extra optional `Position` field this mirror omits. Harmless under
   `omitempty`; the comment overstates what was mirrored. Fix the comment or add the field.

# Session 8 handoff (after Tasks 20, 21, 22)

**All 22 plan tasks are complete, reviewed and gated.** Last gated commit: **`c0b78bb9d`**
(gate 33 PASS). Tree is clean; no writer in flight; no review outstanding.

**The branch is NOT ready for PR.** One task remains, and it is new work the operator ruled in
during this session, not a plan.md task:

**NEXT ACTION: Task 23 — extend the Kafka contract so command failures reach their producers.**
Scoped in full under "Operator ruling — FR-8.3 / FR-6.3" above. In short: `atlas-player-npcs`
currently logs and swallows domain failures (`kafka/consumer/playernpc/consumer.go:66-82,114-122`)
with no reply path, which makes FR-8.3 and FR-6.3 unimplementable as written. Both the Task 21
reviewer and the Task 22 implementer confirmed this independently against source. The operator
chose to **extend the contract**, not narrow the requirements. Three services change:
`atlas-player-npcs` (emit the status/failure event with the design §8.3 reason codes),
`atlas-messages` (consume it, surface via `IssuePinkText`, replacing Task 21's publish-layer-only
best effort), and `atlas-saga-orchestrator` (consume it so a step's failure status carries the
code — which also makes the step no longer purely self-completing, so `saga/event_acceptance.go`
changes with it).

Task 23 has **no brief yet** — it needs one authored from the design §8.3 reason codes and the
three shipped consumer/producer surfaces. Fold in the two Task 22 non-blocking findings above
while there. Findable markers left for it: the doc comment in `saga/event_acceptance.go` and the
note in `saga/handler_test.go`.

After Task 23: the Final gate from plan.md — flagless `tools/verify.sh` (only the flagless run
counts), `go run ./tools/packet-audit matrix --check`, `tools/service-registration-guard.sh`,
`tools/plan-lint.sh` — then the code-review roster (`backend-guidelines-reviewer`,
`plan-adherence-reviewer`, `packet-completeness-critic` for Tasks 5-7), then
`superpowers:finishing-a-development-branch`.

Still open at handoff, unchanged:

- **The nested-`equipment` JSON:API decode gap** (Task 18 review). Neither
  `services/atlas-channel/.../playernpc/rest_test.go:157` nor
  `services/atlas-player-npcs/.../playernpc/resource_test.go:183` exercises api2go/jsonapi
  decoding a non-empty nested struct-slice attribute — both build the model directly. Repo-wide
  pattern, not a task defect, but that seam's decode is unverified on both sides. Close it before
  the PR or record it as a deliberate carry.
- **Two Task 19 not-evaluable items** worth a deliberate look in the pre-PR review: the multi-pod
  "every channel of the world" broadcast (live-verification territory), and player-NPC vs
  ordinary-NPC object-id collision (out of scope for that diff, never affirmatively cleared).
- **Two operator hand-backs** (recorded under Task 8): create `atlas-player-npcs-main` on
  `postgres.home`, and flip the GHCR package public after the first image push.

Standing operating rules on this branch are unchanged — see the Session 7 handoff above.

# Session 9 — Task 23 contract design and split

**Ruling: the correlation shape.** Task 23 extends `COMMAND_TOPIC_PLAYER_NPC` with two additive,
optional envelope fields — `transactionId` (uuid, `uuid.Nil` when not saga-driven) and an opaque
`requester` block (`characterId`, `worldId`, `channelId`, `mapId`) — and adds two new event types
on the **existing** `EVENT_TOPIC_PLAYER_NPC_STATUS` topic: `COMMAND_SUCCEEDED` / `COMMAND_FAILED`,
carrying `StatusCommandOutcomeBody` (`transactionId`, `characterId`, `commandType`, `code`,
`message`, `requester`). One event serves both consumers: the saga correlates on `transactionId`,
`atlas-messages` routes on `requester`, each ignoring the other's field.

Rejected: an in-memory correlation map in `atlas-messages`. Verified against source —
`services/atlas-messages/atlas.com/messages/main.go:33` resolves a **shared** consumer group
(`consumergroup.Resolve("Messages Service")`), so the pod that handled the GM command is not
necessarily the pod that consumes the outcome. Local correlation state is unsound across pods; only
a self-describing event routes correctly.

Rejected: threading `transactionId`/`requester` through the domain and stamping the existing
`DEPLOYED`/`REMOVED` events. The command consumer is the only place holding both the command's
correlation fields and the `Processor` error, so it emits the outcome itself. The four domain
events are untouched.

**No new topic and no deploy change.** Verified: `COMMAND_TOPIC_PLAYER_NPC` (line 78) and
`EVENT_TOPIC_PLAYER_NPC_STATUS` (line 173) are already in `deploy/k8s/base/env-configmap.yaml`, and
`atlas-messages` / `atlas-saga-orchestrator` take them via `envFrom: atlas-env`
(`deploy/k8s/base/atlas-messages.yaml:21-23`).

**FR-6.3's carrier is the existing `errorCode` result convention** —
`StepCompletedWithResult(txId, false, map[string]any{"errorCode": code})`, the same shape
`kafka/consumer/skill/consumer.go:115` and `kafka/consumer/character/consumer.go:202,226` already
use. No new saga plumbing needed; `DeployPlayerNpc` simply moves out of the fire-and-forget set in
`saga/event_acceptance.go`.

**Split into three dispatches** (23a is authoritative and lands first; 23b and 23c mirror it and
are independent of each other):

- **Task 23a** — `atlas-player-npcs`: the wire contract, `playernpc.CodeFor` shared by REST and
  Kafka, outcome emission on all three consumer arms. Brief:
  `.superpowers/sdd/plan/task-23a-brief.md`.
- **Task 23b** — `atlas-saga-orchestrator`: mirror, thread `s.TransactionId()`, new status
  consumer, `DeployPlayerNpc` becomes event-driven. Folds in Task 22 non-blocking finding #1
  (`playernpc/mock/mock.go` → `processor.go` + `var _` assertion) and #2 (the overstated
  "field for field" doc comment). Brief: `.superpowers/sdd/plan/task-23b-brief.md`.
- **Task 23c** — `atlas-messages`: mirror, set `requester` on both GM commands, first status
  consumer in the service, per-code pink text. Brief: `.superpowers/sdd/plan/task-23c-brief.md`.
