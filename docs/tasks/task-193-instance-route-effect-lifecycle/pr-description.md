## Operator step (required for live tenants)

**All three seed groups must be re-seeded, in this order.** They are owned by
three different services, so re-seeding only one leaves the tenant in a
split state where the old scripts and the new `atlas-transports` logic both
run — producing a double apply on boarding and a double cancel on exit.

| # | Endpoint | Service | Why |
|---|---|---|---|
| 1 | `POST /npcs/conversations/seed` | `atlas-npc-conversations` | Drops the `applyBuff` state from `npc-2082003` |
| 2 | `POST /portals/scripts/seed` | `atlas-portal-actions` | Drops the apply op from `outTemple` and the cancel ops from `templeenter` / `undodraco` |
| 3 | `POST /tenants/configurations/instance-routes/seed` | `atlas-tenants` | Adds `effectItemIds` / `forcedReturnMapId` to the two flight routes |

Then confirm via `GET` on atlas-transports' `instance-routes` resource that
`effectItemIds` and `forcedReturnMapId` are populated on
`temple-of-time-flight` and `temple-of-time-return-flight`.

A tenant that is not re-seeded keeps today's behaviour for those two routes.
The new fields are additive and optional, so existing stored configurations
continue to deserialize unchanged.

### Split-state symptoms (if only some groups are re-seeded)

- **Instance-routes only:** the morph is applied twice on boarding (old
  `outTemple` script + new `StartTransport`) and cancelled twice on exit
  (old `undodraco` / `templeenter` script + new `HandleMapEnter`).
- **Scripts only:** the routes declare no `effectItemIds`, so nothing applies
  or cancels the morph at all — the player boards unmorphed.

Known guard: `AfterSeed` refuses to emit when a run deletes rows but creates
none (the missing-catalog-mount signature). If the status event never arrives,
check that log line first — the seed will have reported success.
