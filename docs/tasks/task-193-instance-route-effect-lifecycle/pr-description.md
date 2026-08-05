## Operator step (required for live tenants)

1. `POST /tenants/configurations/instance-routes/seed` for each live tenant.
2. Confirm via `GET` on atlas-transports' `instance-routes` resource that
   `effectItemIds` and `forcedReturnMapId` are populated on
   `temple-of-time-flight` and `temple-of-time-return-flight`.

A tenant that is not re-seeded keeps today's behaviour for those two routes.
The new fields are additive and optional, so existing stored configurations
continue to deserialize unchanged.

Known guard: `AfterSeed` refuses to emit when a run deletes rows but creates
none (the missing-catalog-mount signature). If the status event never arrives,
check that log line first — the seed will have reported success.
