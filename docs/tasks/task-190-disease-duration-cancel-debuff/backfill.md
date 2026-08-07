# CANCEL_DEBUFF live-tenant config backfill

Routing `CancelDebuffHandle` in the seed templates does not reach any tenant
that already exists. Seed templates apply at **tenant creation only**, and
atlas-channel does not hot-reload socket configuration — the pods must be
restarted after the PATCH.

Until a tenant is backfilled it degrades to today's behaviour: the opcode has
no handler, `libs/atlas-socket/server.go` logs the unhandled op at Info, and
nothing errors (NFR-5).

## Scope

All ten live tenants need the backfill: GMS v48, v61, v72, v79, v83, v84,
v87, v92, v95, and JMS v185. Every seed template already carries a
`CancelDebuffHandle` entry (verified directly against
`services/atlas-configurations/seed-data/templates/` — see the opcode table
below), so there is no version exclusion here (unlike task-153, which
excluded GMS v48 because the skill it wired didn't exist on that client).

## Per-tenant procedure

For each live tenant:

1. `GET /tenants/{tenantId}/configurations/socket` — capture the current
   document.
2. Insert this entry into the `handlers` array at its **sorted `opCode`
   position** (strictly ascending — see `docs/packets/TEMPLATE_CONVENTIONS.md`),
   using the opcode for that tenant's version:

   | Tenant version | opCode | Source (seed template : line) |
   |---|---|---|
   | GMS 48.1 | `0x4E` | `template_gms_48_1.json:428` |
   | GMS 61.1 | `0x5B` | `template_gms_61_1.json:520` |
   | GMS 72.1 | `0x62` | `template_gms_72_1.json:532` |
   | GMS 79.1 | `0x61` | `template_gms_79_1.json:532` |
   | GMS 83.1 | `0x63` | `template_gms_83_1.json:587` |
   | GMS 84.1 | `0x63` | `template_gms_84_1.json:591` |
   | GMS 87.1 | `0x66` | `template_gms_87_1.json:506` |
   | GMS 92.1 | `0x6E` | `template_gms_92_1.json:240` |
   | GMS 95.1 | `0x6F` | `template_gms_95_1.json:408` |
   | JMS 185.1 | `0x5E` | `template_jms_185_1.json:478` |

   ```json
   {
     "opCode": "<from the table>",
     "validator": "LoggedInValidator",
     "handler": "CancelDebuffHandle",
     "services": ["channel"]
   }
   ```

   The `validator` field is **not optional**. A handler entry without one
   fails the `validatorMap[hc.Validator]` lookup in
   `libs/atlas-opcodes/producer.go:65-69` and is dropped from the built
   handler map with only a `Warnf` — the opcode is silently unrouted, which
   from an operator's vantage point is indistinguishable from the packet
   never arriving.
3. `PATCH` the document back.
4. `GET` again and confirm the entry is present at the expected position —
   atlas-configurations does not echo the document on PATCH.
5. Restart the tenant's atlas-channel pods.

## Verification

With a client of that tenant connected, take a mob debuff and watch the
channel logs:

- Before: `Read a unhandled message with op 0x63` (or the tenant's opcode),
  repeating at frame rate.
- After: at most one Debug `Throttled CANCEL_DEBUFF` line per second per
  character, and zero unhandled-op lines for that opcode.

## Also required after this deploy: re-ingest Skill.wz

The duration fix in `atlas-data mobskill/reader.go:73` (`m.Duration =
uint32(node.GetIntegerWithDefault("time", 0)) * 1000`) does NOT retroactively
correct already-ingested rows — WZ data is ingested, not parsed per request.
See the Rollout section of `design.md` §10; the ordering is:

1. Deploy the code (atlas-data, atlas-monsters, atlas-maps, atlas-channel,
   atlas-buffs, atlas-configurations, atlas-ui).
2. Re-ingest `Skill.wz` per tenant. Mob-skill ingestion writes through
   `document.Storage.Add` (`services/atlas-data/atlas.com/data/document/storage.go:128-141`),
   whose DB leg (`document/db_storage.go:144-149`) issues the write inside an
   `ON CONFLICT (tenant_id, type, document_id) DO UPDATE` clause, so
   re-ingest is an upsert, not a duplicate error.
3. **Roll atlas-data.** `Storage.ByIdProvider`
   (`document/storage.go:28-36`) checks the per-pod in-memory
   `document.Registry` first and only falls through to the database on a
   cache miss, so a replica that already served a given mob-skill id keeps
   its stale (pre-fix) value cached indefinitely — re-ingest only refreshes
   the DB and the ingesting pod's own cache, not every replica's. Skipping
   this restart is the most likely way for the fix to appear not to work.
4. Verify the DATA, not the deploy: `GET /data/mob-skills/126` must show
   `duration: 15000` for level 2 (raw WZ `time` is `15` seconds — see
   `investigation.md` line 28). atlas-monsters holds no cache
   (`monster/mobskill/processor.go:29-31` calls `requests.Provider` per
   invocation), so it picks up the new value immediately once atlas-data
   serves it.
5. Backfill the socket configs per the procedure above.

Full sweep required — do not backfill one tenant and declare all ten done.
