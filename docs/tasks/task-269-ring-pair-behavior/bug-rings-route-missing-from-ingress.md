# bug: ring pairing invisible in game — `GET /api/rings` is not routed by the nginx ingress, so the channel's ring cache is always empty

**Reproduced:** tenant `935188e4-74a9-4161-b1b5-71660736ca0e` (GMS 83.1, environment
`pr-1524`, namespace `atlas-pr-1524`). Characters `Atlas` (id 38, account 21) and
`Chronicle` (id 39) hold a FRIENDSHIP pair; Atlas has ring template `1112802`
equipped (cash asset id 410, slot `-112`, `cashId` `375638604841764269`, from
`GET /api/characters/38/inventory`). In game: neither character shows a partner
name for the ring, and no proximity effect renders when both are on the same map
with the ring equipped.

**Observed:** every inter-service REST call resolves through the nginx ingress —
`atlas-env` sets `BASE_SERVICE_URL = http://atlas-ingress.atlas-pr-1524.svc.cluster.local:80/api/`
and `libs/atlas-rest/requests/url.go:34-64` rewrites only the namespace, never the
host. The nginx routing table has no location for `/api/rings`, so the request
falls through to the `location /` atlas-ui catch-all and returns the SPA:

```
$ curl -sD - 'http://1524.atlas.home/api/rings?filter[characterId]=38' -H 'TENANT_ID: 935188e4-...' ...
HTTP/1.1 200 OK
Content-Length: 2574
Content-Type: text/html
...
<!doctype html><html lang="en"><head>... <title>AtlasMS</title>
```

Confirmed against the deployed config, not just the repo:
`kubectl -n atlas-pr-1524 get cm atlas-ingress-routes-8cm4g2b9hh -o json` →
`routes.conf.template` contains `cash-shop` but contains no occurrence of
`rings`. In the repo the same hole is in `deploy/shared/routes.conf` (the
`^/api/cash-shop(/.*)?$` block is at line 155; `^/api/coupons(/.*)?$` at 164) and
in the generated `deploy/k8s/base/routes.conf.template.generated`. The comment at
`deploy/shared/routes.conf:159-163` already names this exact failure mode for the
coupon routes: "Without these three blocks the requests fall through to the
`location /` atlas-ui catch-all and the UI gets the SPA's index.html back instead
of JSON."

Downstream, the HTML body fails JSON:API decoding inside
`requests.DrainProvider`, so `ring.ProcessorImpl.Populate`
(`services/atlas-channel/atlas.com/channel/ring/processor.go:106-119`) takes its
fail-soft branch (`degrade.Observe`, returns nil) and caches nothing. With no
cached halves:

- `GetRingRecords` returns an empty `RingRecords`
  (`ring/processor.go:135-141`) → `CharacterData`'s record block
  (`socket/writer/character_data.go:58-60`) writes zero couple/friend records →
  **symptom 1**, no partner name anywhere the client renders one.
- `GetRingSet` returns an empty `RingSet` (`ring/processor.go:124-133`) →
  `CharacterSpawn` (`socket/writer/character_spawn.go:61-65`),
  `CharacterInfo` (`socket/writer/character_info.go:55-66`), and the
  appearance-update path (`kafka/consumer/asset/consumer.go:428`) all encode
  three zero flag bytes → the client never gets the SNs it needs and
  `CUser::SetCoupleItemEffect` never fires → **symptom 2**, no proximity effect.

Channel pod logs could not be read to show the `channel.ring.populate` degrade
line: `kubectl logs` times out cluster-wide from this workstation
(`Unable to connect to the server: net/http: request canceled (Client.Timeout
exceeded while awaiting headers)`) while `kubectl get` succeeds. The ingress
evidence above is direct and does not depend on the logs.

**Expected:** PRD FR-5 / §8 — the channel populates its ring cache once per
character load from atlas-cashshop's `GET /rings?filter[characterId]=`, and the
record block plus the spawn ring set carry the pair. The cashshop route itself
exists and is correct: `services/atlas-cashshop/atlas.com/cashshop/ring/resource.go:29`
registers `/rings` and `/rings/{ringId}` at the router root — **not** under the
`/cash-shop` prefix — so the existing `^/api/cash-shop(/.*)?$` block does not
cover it.

**Root cause:** task-269 added a new top-level REST resource (`/rings`) on
atlas-cashshop without adding the matching nginx location to
`deploy/shared/routes.conf`. `tools/verify.sh:568` only runs
`gen-routes.sh --check` (drift between source and generated file); nothing checks
that a newly registered REST resource has a route, so the gate passed.

## Fix

- `deploy/shared/routes.conf` — add a location block for the rings resource,
  alongside the other atlas-cashshop-owned surfaces (after the
  `^/api/cash-shop(/.*)?$` block at line 155, or with the coupon blocks that
  carry the same explanatory comment). It must cover both routes registered by
  `ring/resource.go` — the list `/api/rings?filter[characterId]=` and the item
  `/api/rings/{ringId}`:

  ```
  location ~ ^/api/rings(/.*)?$ {
    set $u "atlas-cashshop:8080";
    proxy_pass http://$u$request_uri;
  }
  ```

  Keep the file's existing comment style: state why the block is needed (the
  resource is registered at the cashshop router root, not under `/cash-shop`).
- `deploy/k8s/base/routes.conf.template.generated` — regenerate with
  `tools/gen-routes.sh` (do not hand-edit). `tools/gen-routes.sh --check` must
  then exit 0; `deploy/k8s/base/ns-vars.generated.yaml` needs no new variable
  because `NS_ATLAS_CASHSHOP` already exists.
- `deploy/shared/test/routes_nginxt.sh` — add an assertion that
  `/api/rings?filter[characterId]=1` and `/api/rings/<uuid>` both resolve to
  `atlas-cashshop`, following the file's existing per-route assertion style.
  This is the test that fails before the routes.conf change and passes after.
- `services/atlas-channel/docs/` — if the service docs enumerate the upstream
  REST dependencies of the ring cache, note that `GET /rings` is served by
  atlas-cashshop through the ingress. Do not invent a new doc file for this.

## Not yet answered

- **Whether the cash-id join is also broken.** With the ingress hole closed,
  `selectPair` (`ring/processor.go:170-200`) still has to match
  `equipment slot.CashEquipable.CashId()` against the cashshop half's
  `CashId()`. Live data shows the equipped half carries `cashId`
  `375638604841764269` at slot `-112` (the cash-equip mirror of ring1 `-12`); the
  cashshop side could not be read because the very route this bug is about is
  unreachable. **The fix agent must not speculatively change the join.** Land the
  route fix only; the join is re-tested live afterwards and gets its own bug file
  if it is still wrong.
- **FR-12 remains in force**: a ring equipped while both characters already stand
  on the same map still needs a map change before the effect appears. When
  re-testing, both characters must re-enter the map (or relog) after the fix is
  deployed — a live re-test that skips this does not disprove the fix.

## Resolution

- **Fixed by:** `1721d8d25` — "fix(deploy): route GET /api/rings to atlas-cashshop
  via ingress". Files: `deploy/shared/routes.conf`,
  `deploy/k8s/base/routes.conf.template.generated` (regenerated via
  `tools/gen-routes.sh`), `deploy/shared/test/routes_nginxt.sh`,
  `services/atlas-channel/docs/rest.md`. No Go file changed.
- **Gate:** `tools/verify.sh --quick --base 8afae7249` → exit 0 (service
  registration, service name, LB port drift, routes drift, version coverage,
  overlay env drift, sparse baseline scoping all green).
- **Live re-test:** NOT YET CONFIRMED. The fix only takes effect in
  `atlas-pr-1524` once the PR redeploys the `atlas-ingress-routes` ConfigMap and
  the nginx pod picks it up. After that, per FR-12, both characters must relog or
  change maps before the partner name and the proximity effect appear. Update
  this section with the live result.
- **Still open:** the cash-id join question in `## Not yet answered` above is
  untouched and can only be answered after the live re-test.
