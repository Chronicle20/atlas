# RPS Reward Ladder — Sourcing & Verification (task-132, Task 26)

## Outcome (updated 2026-07-17)

**The `rps-rewards` config ships a 10-rung streak-certificate ladder plus a
consolation prize.** Rung N (N consecutive wins) grants the "Certificate of
N-straight Win(s)" item; a first-round loss pays a small meso consolation.

Seeded default (`services/atlas-tenants/configurations/rps-rewards/default.json`):

| rung | itemId | quantity | meso |
|------|--------|----------|------|
| entry cost | — | — | 1000 (deducted on entry; ShowEffect) |
| consolation (rung-0 loss only) | — | — | 500 |
| 1 | 4031332 | 1 | 0 |
| 2 | 4031333 | 1 | 0 |
| 3 | 4031334 | 1 | 0 |
| 4 | 4031335 | 1 | 0 |
| 5 | 4031336 | 1 | 0 |
| 6 | 4031337 | 1 | 0 |
| 7 | 4031338 | 1 | 0 |
| 8 | 4031339 | 1 | 0 |
| 9 | 4031340 | 1 | 0 |
| 10 | 4031341 | 1 | 0 |

`entryCostMeso: 1000` matches the NPC 9000019 conversation seed
(`rpsAction.entryCostMeso: 1000`) and the dialogue text. `consolationMeso: 500`
matches the v83 client string `SP_3681` ("here's 500 mesos as a consolation
prize"). Operators tune all of these per tenant via the `rps-rewards`
configuration resource (PATCH — see `live-config-patch.md`).

## Item-id verification (the "no unverified item id" rule is satisfied)

The certificate item ids were supplied by the maintainer (external authority)
and independently verified two ways before shipping — they are **not** invented
from memory, so CLAUDE.md's Grounding rule is honored:

1. **WZ-verified.** `Item.wz/Etc/0403.img` + `String.wz/Etc.img` resolve
   `4031332`–`4031341` to real items named **"Certificate of 1-straight Win"**
   … **"Certificate of 10-straight Wins"** — the ids and their 1:1 mapping to
   the streak count are correct (`4031331 + N` → N straight wins).
2. **Live-verified end-to-end.** In the `atlas-pr-933` env a win-streak Collect
   at rung 6 emitted the payout saga and the asset-status event confirmed the
   item was actually created in the character's inventory
   (`templateId:4031337, type:CREATED, quantity:1`) — i.e. the ids resolve in
   the tenant's atlas-data and `AwardAsset` grants them.

## Payout wiring

- **Win → climb.** Each win advances the rung (the streak). The player may
  **Collect** (bank the current rung's certificate via the payout saga's
  `AwardAsset` step) or **Continue** (risk it for the next rung).
- **Loss at rung 0** (never won this game) → **consolation** `consolationMeso`
  via an `AwardMesos` step, awarded when the player leaves the loss screen
  (Exit/Retry), deferred so the meso effect lands after the client renders the
  loss. A loss at rung ≥ 1 (after a win) pays nothing — the streak was gambled.
- **Retry** re-charges the full `entryCostMeso` (blocking — a saga-submit
  failure aborts the restart) and reopens a fresh round at rung 0.

The config, codec, channel writer, NPC entry saga, and payout saga all support
both meso and item rewards, so re-tuning the ladder is a **data-only change**
(a config PATCH per tenant, or an updated `default.json` re-seed).

## Historical note

An earlier revision of this doc shipped a meso-only placeholder ladder
(2000/5000/10000, `itemId: 0`) because no item-id verification path was
available in that execution environment. That constraint was lifted once the
maintainer supplied the certificate ids and both the WZ tables and a live
atlas-data tenant became reachable for verification (above).
