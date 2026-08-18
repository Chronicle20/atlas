# bug: potions cannot be used — every consumable/cash item 404s in atlas-data

**Reproduced:** tenant `15d000e1-260e-414b-bdf0-4c2b68b8995c`, GMS 83.1,
namespace `atlas-pr-1375` (PR #1375 = `task-231-generalized-events-service`).
Character 1 uses a White Potion (2000005) or Orange Potion (2000004) from the
USE inventory. Reproduced 25 times across three sessions and one relog
(13:32–13:54 UTC on 2026-08-18).

**Observed:** the item is never consumed, no HP is restored, and the client's
item-use action never completes.

The packet arrives and decodes correctly — `atlas-channel`:

```
"[CharacterItemUseHandle] read [updateTime [270443501], source [1], itemId [2000004]]"
```

`atlas-consumables` then answers on `EVENT_TOPIC_CONSUMABLE_STATUS` with an
ERROR carrying an **empty** error string:

```
Message received {"characterId":1,"type":"ERROR","body":{"error":""}}.
```

and logs the cause:

```
"Character [1] unable to consume item due to error: [not found]"
  originator: EVENT_TOPIC_COMPARTMENT_STATUS-3178
  trace.id:  d79d041112cd305127a78f15a675b26c
```

The `not found` originates in `atlas-data`, same trace id:

```
"Handling [GET] request on [/api/data/consumables/2000004]"
SELECT * FROM "documents" WHERE (type = 'CONSUMABLE' AND document_id = '2000004')
  AND "documents"."tenant_id" = '144ba144-0b45-5635-a37b-28ffacb55285'  [rows:0]
{"error":{"message":"record not found"},"message":"Unable to locate consumable 2000004."}
```

`atlas-data` is queried under both the raw tenant (`15d000e1…`) and the
canonical GMS-83.1 tenant (`144ba144…`, `canonical/canonical.go:33`, frozen in
`canonical_test.go:12`). **Both return zero rows.** Across the environment's
lifetime every `documents` query was one of these two types and every one
returned `rows:0`:

```
 30 SELECT * FROM "documents" WHERE (type = 'CASH'        …  → all rows:0
 25 SELECT * FROM "documents" WHERE (type = 'CONSUMABLE'  …  → all rows:0
```

Direct probe of `atlas-data` (in-cluster, GMS 83.1 headers):

| path | status |
|---|---|
| `consumables/2000000 .. 2070000` (10 ids probed) | **404** (all) |
| `cash/5040000` | **404** |
| `etcs/4000000` | 200 |
| `setups/3010000` | 200 |
| `pets/5000000` | 200 |
| `equipment/1302000` | 200 |
| `monsters/100100`, `maps/100000000`, `skills/4001003` | 200 |

**The item still appears in the web UI — that is not a contradiction.** The UI's
name and icon come from a *different* document type, `ITEM_STRING` (ingested
from `String.wz`, served by `/data/item-strings`,
`services/atlas-data/atlas.com/data/item/string_resource.go:27-29`), which
ingested fine:

```
GET /api/data/item-strings/2000004 → 200 {"attributes":{"name":"Elixir"}}
GET /api/data/consumables/2000004  → 404
```

Only the `CONSUMABLE` document — the one from `Item.wz/Consume` carrying the
HP/MP spec that `ConsumeStandard` needs — is absent.

`GET /api/data/status` reports `documentCount: 49049, updatedAt: 2026-08-08T17:28:58Z`
— the corpus is populated, but the CONSUMABLE and CASH document types are
entirely absent from it.

**Not PR-specific:** the same probe against namespace `atlas-main` also returns
404 for `consumables/2000004`. This is not a regression introduced by this
branch.

**Not task-229:** task-229 (`c2097964d`) changed template opcode bindings on
`gms_48`, `gms_72`, `gms_87`, `gms_92`, `gms_95` and `jms_185` only. It did not
touch `template_gms_83_1.json`, which is the template this tenant uses, and
whose item-use bindings are intact (`0x48 CharacterItemUseHandle`,
`0x49 CharacterItemCancelHandle`, `0x4B` summon bag, `0x55` town scroll,
`0x56` upgrade scroll). The packet reaching `CharacterItemUseHandle` with the
correct itemId proves the binding is correct.

**Expected:** `GET /api/data/consumables/2000004` returns 200 with the item's
spec; `ConsumeStandard` proceeds to `ConsumeItem` and applies the HP effect.

**Root cause:** *partially* established.

Established with evidence: the failure is entirely explained by
`atlas-data` having no `CONSUMABLE` (and no `CASH`) documents for GMS 83.1.
`ConsumeStandard`
(`services/atlas-consumables/atlas.com/consumables/consumable/processor.go:473`)
submits `cdp.GetById(itemId)` in a parallel group; its 404 fails `pg.Wait()` at
line 474 and routes to `ConsumeError` at line 475 — before `ConsumeItem` is ever
called.

**Not** established: *why* those two categories are absent while every other
category ingested fine. `Item.wz` ingest registers five categories in order —
`services/atlas-data/atlas.com/data/data/workers/item.go:41-46`:
`Consume`, `Cash`, `Etc`, `Install`, `Pet`. The first two are missing and the
last three are present. `registerAllInDirectory` only *warns* on failure
(`item.go:52`), so a failed directory walk would leave exactly this shape and
not fail the ingest. Ruled out:

- Not a per-file parse abort — `RegisterConsumable` opens one transaction per
  file (`consumable/processor.go:55`), so a single bad file cannot empty the
  category.
- Not a reader regression from `bb2ac767a` (Scissors of Karma) or `543a88df6`
  (scripted items): `go test ./consumable/... ./cash/...` in
  `services/atlas-data/atlas.com/data` passes, and the corpus `updatedAt` is
  2026-08-08 — it predates both commits (2026-08-15), so neither commit's code
  ever ran against this corpus.
- Not a tenant-keying mismatch: both the raw and canonical tenant ids were
  queried and both returned zero rows.

## Fix

Two separable pieces. **The first is operational and must be settled before any
code is written** — it decides whether there is a code fix at all.

1. **Re-run the `Item.wz` ingest for GMS 83.1 and capture the Job's logs.**
   `POST /api/data/process` on `atlas-data` launches the ingest Job
   (`services/atlas-data/atlas.com/data/runtime/rest/resource.go:34`). Watch for
   a `walk …/Item.wz/Consume` / `…/Item.wz/Cash` warning from
   `workers/item.go:52`. That warning names the real defect. If the categories
   then populate and potions work, there is no code defect in the consume path
   and only item 2 below remains. **This is a mutating operation on a live
   environment — get operator sign-off before running it.**

2. **`services/atlas-consumables/atlas.com/consumables/consumable/processor.go:379-397`
   (`ConsumeError`)** — independent robustness defect, real regardless of the
   ingest outcome. Every failure that is not `ErrPetCannotConsume` /
   `ErrPetCannotLearn` emits `ErrorEventProvider(…, "")` — the empty
   `errorType` seen on the wire above. The channel receives
   `{"type":"ERROR","body":{"error":""}}` and has nothing to map to a client
   message or an unstick action. Give the generic failure a distinct error type
   and make the channel act on it (at minimum, re-enable client actions so the
   user is not left with a dead item-use).
   - `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/`
     — add the error-type constant alongside `ErrorTypePetCannotConsume`.
   - `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go`
     — assert `ConsumeError` emits the new type for a generic error; must fail
     before, pass after.
   - the channel's `EVENT_TOPIC_CONSUMABLE_STATUS` consumer — handle the new
     type.

## Resolution

**Confirmed by re-ingest.** The operator triggered an `Item.wz` ingest on
`atlas-main` (Job `ingest-shared-gms-83-1-gn9ep4`, started 2026-08-18T14:19:16Z).
Mid-run, `GET /api/data/consumables/2000004` on `atlas-main` flipped from 404 to
**200**. The diagnosis holds: the failure was a missing `CONSUMABLE` document
corpus, not a code defect in the consume path.

`atlas-pr-1375` still returns **404** — it has its own database
(`atlas-data-3178`) and needs its own ingest before potions work there.

The ingest Job's own log shows only six warnings, all mobskill-related
(`mobskill.InitString failed`, `register BFSkill.img.xml`, `MobSkill.img.xml`,
etc.) — no `walk …/Item.wz/Consume` warning. So this run's Item worker did not
fail. Why the 2026-08-08 corpus lacked the category is not recoverable from
here: that run's Job logs are gone. Left unanswered rather than guessed.

Fix item 2 below (the empty `errorType`) is independent of all this and landed
in commit `f1d8bb567`.

## Not yet answered

- Whether the `Item.wz` `Consume` and `Cash` directories are absent from the
  serialized archive in MinIO, or present and failing to walk. Answer comes from
  the ingest Job log in fix step 1. Do not guess — do not "fix" the worker
  before that log exists.
- `atlas-main` shows the same 404, so this affects the mainline environment too.
  Whoever runs the ingest should run it for both.
- If the ingest restores the data, fix item 2 still lands: an empty `errorType`
  on the wire is a defect on its own.
