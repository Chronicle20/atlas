# RPS Reward Ladder — Sourcing & Verification (task-132, Task 26)

## Outcome

**The `rps-rewards` config ships an operator-tunable, meso-only ladder.** No item-reward
entries are seeded, because neither the authentic Cosmic reward set nor an item-id
verification path was available in this execution environment, and the project rule
**"Do not ship an unverified item id"** (CLAUDE.md → Grounding & Honesty) is binding.

Seeded default (`services/atlas-tenants/configurations/rps-rewards/default.json`):

| rung | itemId | quantity | meso |
|------|--------|----------|------|
| entry cost | — | — | 1000 |
| 1 | 0 (none) | 0 | 2000 |
| 2 | 0 (none) | 0 | 5000 |
| 3 | 0 (none) | 0 | 10000 |

`entryCostMeso: 1000` is consistent with the NPC 9000019 conversation seed
(`deploy/seed/*/npc-conversations/npc/npc-9000019.json`, `rpsAction.entryCostMeso: 1000`)
and the dialogue text, by convention (design decision D-context #2). The meso escalation
(2000 / 5000 / 10000 for a 1000 ante) is a **tunable default, explicitly not claimed
authentic** — operators tune it per tenant via the `rps-rewards` configuration resource
(PATCH; see `live-config-patch.md`).

## Why item rewards were not seeded (verification constraints)

Task 26's brief requires the ladder be **Cosmic-sourced and every item id WZ/atlas-data
verified**. Both prerequisites were unavailable here:

1. **No Cosmic source in-repo.** There is no Cosmic `9000019.js` (or equivalent RPS reward
   reference) checked into this repository (`docs/research/`, `docs/`, and the seed trees
   contain none). The design named Cosmic as the reference but did not vendor it.
2. **No item-id verification path in-environment.** Verifying an item id requires either
   local WZ item data or a reachable `atlas-data` item endpoint:
   - No WZ data files are present at a queryable path (the `libs/atlas-wz` *library* exists,
     but no bundled/mounted WZ item tables were found to resolve ids against).
   - No running `atlas-data` instance / `DATA_SERVICE_URL` was configured in this
     environment to resolve ids at runtime.
3. **Fetching Cosmic externally would not unblock the constraint.** Even if the Cosmic
   script were fetched from an external source, its item ids could not be verified against
   local WZ/atlas-data here — so seeding them would ship **unverified** item ids, which the
   grounding rule forbids. The blocker is *verification capability*, not merely *sourcing*.

Per the brief's own instruction — *"Any id that does not resolve is dropped or replaced with
a meso equivalent … Do not ship an unverified item id"* — the ladder is therefore meso-only.
This is the safe, correct terminal outcome given the environment, not a silently skipped step:
the config, the codec, the channel writer, the NPC entry saga, and the payout saga all support
item rewards (`AwardAsset`), so filling the ladder later is a **data-only change** (a config
PATCH per tenant, or an updated `default.json` re-seed) with no code changes required.

## Follow-up (to replace the meso-only default with verified item rewards)

When a Cosmic RPS reward reference **and** an item-id verification path (local WZ item data or
a reachable `atlas-data`) are both available:

1. Obtain the authentic Cosmic `9000019` reward set; record the raw ladder here.
2. For **every** item id, verify it resolves in local WZ / atlas-data. Drop or meso-substitute
   any id that does not resolve; record each decision.
3. Write the verified ladder into `default.json` (and PATCH live tenants). Keep `entryCostMeso`
   consistent with the NPC seed — **if the authentic entry cost differs from 1000, update BOTH
   the config and all five `npc-9000019.json` seeds** (the dialogue text quotes the cost).
4. Re-run the atlas-tenants module gate; re-verify the atlas-rps `GetLadder` round-trip.

This follow-up is recorded in `verification.md` (Task 27) alongside the v92 and Retry parks.

## What was verified

- The meso-only ladder contains **zero item ids** → nothing unverified is shipped.
- `entryCostMeso` (1000) matches the NPC conversation seed and the dialogue text.
- The ladder round-trips through `atlas-rps` `configuration.GetLadder` (Task 7/21 contract,
  array-shaped JSON:API) and resolves via `game.Ladder.PrizeAt` (Task 6) — a rung with
  `itemId: 0, quantity: 0` yields a meso-only prize, and the payout saga (Task 12) emits an
  `AwardMesos`-only step (no `AwardAsset`) for such a rung, which is exercised by the
  atlas-rps meso-only Collect test.
