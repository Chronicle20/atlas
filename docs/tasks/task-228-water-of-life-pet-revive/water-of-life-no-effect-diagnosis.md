# "The Water of Life has no effect." on atlas-pr-1360 — diagnosis

Reported 2026-08-16: pet reads as dried up in the inventory, double-clicking the
Water of Life produces *"The Water of Life has no effect."*

## Not a code defect on this branch. Stale ingested atlas-data document.

The packet path is healthy end to end. `atlas-channel` logged four
`[WaterOfLifeHandle] read []` invocations for character 1, so the opcode is
routed (`0x75` → `WaterOfLifeHandle` is present in the live tenant config) and
the empty-body codec decodes. Every one of the four bailed at the same guard:

```
Water of Life [5180000] has no info/life; refusing to consume it for character [1].
```

which is `water_of_life.go`'s `cd.Life == 0` pre-flight → `waterOfLifeNoEffectMessage`.

### Evidence chain

1. WZ has the node. `Item.wz/Cash/0518.img → 05180000/info/life = 90`.
2. The deployed reader parses it. Both `atlas-data` and `atlas-channel` run
   `pr-1360-668f115`, which carries `m.Life = i.GetIntegerWithDefault("life", 0)`
   (`services/atlas-data/.../cash/reader.go`).
3. The served REST payload has no `life`:
   `GET /api/data/cash/items/5180000` →
   `{"slotMax":1,"spec":{},"tradeBlock":false,"tradeAvailable":0,"karma":0}`.
4. `atlas-data` runs `MODE=rest` — it never re-parses WZ per request (no
   `Processing cash [...]` line in its log for these requests). It serves the
   pre-ingested `documents` row, which is dated **before** the reader change:

```
tenant_id                            | updated_at                    | content
14e6eaee-cbe2-4cb7-8d98-2983231de016 | 2026-08-08 17:25:50.279929+00 | {"data":{"type":"cash_items","id":"5180000","attributes":{"slotMax":1,"spec":{}}}}
```

This is the known ephemeral-env shape: the PR env's tenant atlas-data is a
verbatim restore of a pre-published canonical baseline, not a fresh WZ ingest,
so it predates any new reader field. Compounding it, the deployment's
`INGEST_IMAGE` is pinned to `:latest` (i.e. `main`), which does **not** contain
the `life` reader — so a normal re-ingest would reproduce the same empty row.

`atlas-pets` and `atlas-inventory` read `life` from the same document, so all
three consumers are affected by the one stale row.

## Unblocking the env for manual testing

Patch the stored document, then restart the REST pods (`ByIdProvider`'s
in-memory `RegStorage` cache has no TTL and no invalidation, and 5180000 was
already requested, so it is cached stale for the pod's lifetime):

```sh
# 1. add life:90 to the tenant's cash document
psql -h postgres.home -U atlas -d atlas-data-c322 -c \
  "update documents set content = jsonb_set(content::jsonb, '{data,attributes,life}', '90') \
   where type='CASH' and document_id='5180000' \
     and tenant_id='14e6eaee-cbe2-4cb7-8d98-2983231de016';"

# 2. drop the cached row
kubectl -n atlas-pr-1360 rollout restart deploy/atlas-data
```

The durable fix is a re-ingest with an image that contains the reader change
(`INGEST_IMAGE=ghcr.io/chronicle20/atlas-data/atlas-data:pr-1360-<sha>`), which
also republishes a canonical baseline carrying `life`.

## Carry-over for the PR

The feature has a hard runtime dependency on re-ingested cash data: any tenant
whose atlas-data baseline predates commit `39bf1890a` serves `life` absent, and
the `cd.Life == 0` guard turns every Water of Life use into a no-op with the
"no effect" message. Worth calling out in the PR description as a deploy
prerequisite — the guard behaves correctly, but "correct" here is
indistinguishable from "broken" to a player.
