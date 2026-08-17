# task-230 — play-test failure on `atlas-pr-1359` (2026-08-15)

## Symptom

Using Golden Compass (`2430008`, GMS 83.1, tenant `8c195db0…`) produced:

```
Saga transaction failed. Reason: [Cash item use (scripted_item_use) failed at
step [start_item_conversation] action [start_item_conversation]]
```

The channel handler, saga orchestrator, and `START_ITEM_CONVERSATION` command all
behaved exactly as designed. `atlas-npc-conversations` logged the real cause:

```
No conversation authored for scripted item; not consuming.
SELECT * FROM "item_conversations" WHERE item_id = 2430008 … [rows:0]
```

So the feature path is correct end to end; the **content never reached the
database**. Three independent defects, none of which `tools/verify.sh` can see.

## Defect 1 — the authored seed content fails domain validation

`POST /api/items/conversations/seed` on the pod returned 202 and logged
`Seed complete … {"item.conversation": 0}`. The `seed_state` row carries the
errors the log drops:

```json
"errors": ["item-2430008.json: item-conversations: build: sendOk requires exactly 2 choices",
           "item-2430013.json: item-conversations: build: sendOk requires exactly 2 choices"]
```

`DialogueBuilder.Build` (`conversation/model.go:641`) requires **exactly 2**
choices for `sendOk`. Both authored files gave one (`"OK"`). All 16 files
(2 items × 8 versions) were affected. Fixed to the convention used by every
shipped npc conversation — `("Ok", "Exit")`, both `nextState: null`
(679 of 680 `sendOk` states in `gms/83_1/npc-conversations/npc` use exactly that).

`tools/catalog-lint` only validates the JSON:API envelope (type/id), never the
domain build, which is why CI was green. Added
`conversation/item/seed_content_test.go`, which drives every version's item
seed files through the seeder's own `Decode` → `Build` path. Confirmed it fails
on the pre-fix content with the exact production error and passes after.

## Defect 2 — nothing ever invokes the item seed group

`services/atlas-pr-bootstrap/scripts/bootstrap.sh` enumerates the seed
endpoints an ephemeral env POSTs at creation. `/api/items/conversations/seed`
was not in the list, so the group was never seeded in `atlas-pr-1359` — the
service logs show only `npc-conversations:npc` and `npc-conversations:quests`
running at 18:09. Added.

## Defect 3 — the item-conversation routes are unreachable through the ingress

`deploy/shared/routes.conf` had no `location` for `/api/items/conversations`, so
every item-conversation route (the whole CRUD resource *and* the seed endpoint)
fell through to the `location /` SPA. Verified live:

```
GET http://atlas-ingress/api/items/conversations/seed/status
→ 200 text/html  (atlas-ui index.html)
```

Added the location block and regenerated
`deploy/k8s/base/routes.conf.template.generated` via `tools/gen-routes.sh`.

## Known gap left open

The `npc-conversations:items` group has no operator surface in atlas-ui —
`seed.service.ts`, `useSeed.ts`, and `SetupPage.tsx` carry entries for every
other seed group but not this one. Not required for the play-test (bootstrap
now seeds it), but it is the only group an operator cannot re-seed or inspect
from the UI.

## Observability note (not fixed)

`postSeed` logs `Seed complete` at info with only the **created** counts
(`libs/atlas-seeder/handlers.go:64`, `summarize`). A run where every file failed
is indistinguishable in the logs from a run with an empty catalog; the errors
only exist in `seed_state.result_summary`. This is what made the failure silent
for both this task and any future one.
