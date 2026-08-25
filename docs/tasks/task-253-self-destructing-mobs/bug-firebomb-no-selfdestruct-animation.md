# bug: Firebomb (5100002) self-destructs with an ordinary death, no self-destruct animation

Task: task-253-self-destructing-mobs
Branch: `task-253-self-destructing-mobs` @ `bbe808f7c`
Environment: namespace `atlas-pr-1462`, tenant `b3a10958-f792-4e02-bb62-89cd6cf11e9e`,
region **GMS**, version **83.1** (confirmed from
`/api/configurations/tenants/b3a10958-...`).

## Reproduced

Not reproduced interactively — reported from live testing on `atlas-pr-1462`. The
server-side path was traced end to end instead, against the live tenant's data and
config plus the GMS v83 IDB (session `754107bf`, `MapleStory_dump.exe.i64`).

`atlas-monsters` runs at info level in this namespace
(`kubectl -n atlas-pr-1462 logs deploy/atlas-monsters --since=6h | grep -c '"log.level":"debug"'`
→ `0`), so the `SELF_DESTRUCT ... deathType [...]` debug line was not available as
runtime evidence.

## Observed

Firebomb detonates at its HP threshold and vanishes with what looks like an ordinary
death — no distinct "self-destruct state", no explode animation.

## Expected (reporter)

At the threshold the mob should enter a self-destruct state, play an animation, and
then disappear.

## Root cause — NOT a defect in the branch; it is the data

The full chain, each link verified:

1. **WZ data for this tenant.** `atlas-data` on the live namespace returns for
   mob 5100002 (**Firebomb**):
   `"self_destruction": {"action": 1, "remove_after": -1, "hp": 1800}`,
   with `animation_times.die1 = 780`. There is no explode/bomb animation in its
   extracted animation set.
   (`kubectl -n atlas-pr-1462 exec deploy/atlas-data -- wget -qO- .../data/monsters/5100002`)

2. **atlas-monsters.** `damageCore` (`monster/processor.go:693-696`) crosses the
   threshold and calls `selfDestructFrom(..., deathTypeForAction(l, sd.Action()), TriggerThreshold)`.
   `deathTypeForAction` (`processor.go:1878-1897`) maps `action == 1` → `DeathTypeFadeOut`
   (`monster/kafka.go:68` = `"FADE_OUT"`), carried on the KILLED event.

3. **atlas-channel.** `handleStatusEventKilled` → `destroyCodeFor(e.Body.DeathType)`
   (`kafka/consumer/monster/consumer.go:313,216`) → `writer.DestroyMonsterBody(uniqueId, "FADE_OUT")`
   (`socket/writer/monster_destroy.go:28`), resolved through the tenant operations
   table. The live tenant's `DestroyMonster` writer (opcode `0xED`) carries
   `{"BOMB":2,"DESTRUCT_BY_MISS":3,"DISAPPEAR":0,"FADE_OUT":1,"SELF_DESTRUCT":5,"SWALLOW":4}`,
   so the wire byte is **1**.

4. **GMS v83 client.** `CMobPool::OnMobLeaveField` @ `0x67961d` stores a non-zero
   dead-type and queues the mob into the delayed-dead list. `CMobPool::Update`
   @ `0x679138` then branches:

   ```c
   if ( deadType <= 1 || deadType == 3 )  CMob::OnDie(mob);   // 0x663995
   else                                   CMob::OnBomb(mob);  // 0x663e5b
   ```

   `CMob::OnBomb` is the behaviour the reporter is describing: it plays the mob
   sound, sets `m_nOneTimeAction = 12`, calls `PrepareActionLayer`, and ends with
   `CMob::DoAttack(this, 12, IsLeft, 0)` — i.e. the mob visibly enters an explode
   action. It is reachable on v83 only from dead-types **2, 4, 5**.

   Dead-type **1** goes to `CMob::OnDie` — the ordinary death. That is exactly what
   the reporter is seeing.

Also confirmed: the v83 client has **no** `selfDestruction` / `removeAfter` string
(`find_regex` over session `754107bf` → 0 matches), so the client cannot derive the
self-destruct animation on its own. The server's dead-type byte is the only lever.

So the branch is faithful to the data and to design §2.2 ("the server passes the WZ
`action` byte through verbatim and does not remap per version"). Firebomb's data asks
for an ordinary death, and it gets one.

The tension is with the PRD, which states as a user story (prd.md:66) "As a player
killing a Boomer, I want it to play its explosion animation rather than the generic
fade-out" — unreachable for `action == 1` without deviating from the WZ data. Of the
12 mobs in the PRD's §6.3 reference table, only `9300166`/`9300329` (action 4) and
`9400547`/`9400550` (action 5) reach `CMob::OnBomb` on v83.

**Separate PRD defect (naming, not behaviour):** prd.md §6.3 annotates `5100002` as
"Boomer", and the prd.md:66 user story is written about "a Boomer". `atlas-data`
reports `5100002`'s name as **Firebomb**; `5100000`–`5100004` are Jr. Yeti,
Transforming Jr. Yeti, Firebomb, Hodori, Samiho, and none of the others carries a
`selfDestruction` block. Whichever mob the PRD meant by "Boomer", it is not
`5100002`, and it does not appear in the §6.3 table. Fix the label regardless of
which option below is chosen.

## Fix

**Blocked on a product decision — do not implement until it is made.** Any fix here
means deliberately not sending the WZ `action` byte, which design §2.2 explicitly
rejected. Options, none of them yet chosen:

- **A — accept as correct.** No code change. Update prd.md:66 so the Firebomb user
  story matches the data, and record the v83 `OnDie`/`OnBomb` split in design §2.2
  (currently only v87 and v95 are documented there).
- **B — remap self-destruct triggers to a bomb dead-type.** Send `BOMB` (or the
  per-version equivalent) for any `TriggerThreshold`/`TriggerTimer`/`TriggerContact`
  death regardless of `action`. Reverses design D2/§2.2 and invents an animation the
  data did not ask for; Firebomb has no extracted explode animation, so the client
  result is unverified.
- **C — per-mob data override.** Keep pass-through, add an explicit override table
  for mobs whose `action` disagrees with intended behaviour.

Files that would change under B or C:

- `services/atlas-monsters/atlas.com/monsters/monster/processor.go`
  (`deathTypeForAction`, `SelfDestruct`, `selfDestructFrom`, `damageCore` threshold arm)
- `services/atlas-monsters/atlas.com/monsters/monster/self_destruct_timer_task.go`
- tests: `monster/self_destruct_test.go`, `monster/self_destruct_timer_test.go`
- docs: `docs/tasks/task-253-self-destructing-mobs/design.md` §2.2, `prd.md` §3/§6.3

Under A, only prd.md and design.md change.

## Not yet answered

- Which option the reporter wants. This is the blocking question.
- What the reference GMS v83 server actually sent for Firebomb. Cosmic passes
  `selfDestruction.action` through verbatim (matching option A), but that has not
  been confirmed against retail behaviour.
- Whether Firebomb's Mob.wz actually contains an action-12 animation. `atlas-data`'s
  extracted `animation_times` does not list one, but that extraction is not
  exhaustive — the raw WZ node was not inspected.
- Whether the reporter's "instantaneously" means "no animation at all" (which would
  contradict the `CMob::OnDie` + `die1 = 780ms` trace above and reopen the
  investigation) or "an ordinary death instead of a self-destruct sequence" (which
  the trace fully explains).

## Update 2026-08-25 — likely NOT A BUG, superseded

The reporter confirms Cosmic passes `selfDestruction.action` through on the HP-threshold
path and only hardcodes a bomb dead-type in `MONSTER_BOMB` (contact) handling. Firebomb's
threshold detonation is therefore *supposed* to render as an ordinary death on v83, and
this branch already does that. Option A (accept, fix the "Boomer" mislabel in prd.md:66
and §6.3, document the v83 `OnDie`/`OnBomb` split in design §2.2) is the resolution.

The contact-path deviation is tracked in
[`bug-darkstar-no-explosion-or-damage.md`](bug-darkstar-no-explosion-or-damage.md).

## Resolution

Behaviour: not a defect. Docs fix outstanding (prd.md naming, design.md §2.2 v83 arm).
No code change. No commit yet.
