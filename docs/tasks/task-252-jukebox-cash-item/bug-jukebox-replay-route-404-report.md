# bug-jukebox-replay-route-404 — fix report

## What I implemented

Added the missing ingress `location` block for the map-instance `jukebox`
subresource, exactly as specified in the brief's `## Fix` section.

- `deploy/shared/routes.conf` — inserted a new `location` block immediately
  after the `weather` block (before `monsters`), matching the surrounding
  blocks' style (two-space indent, `set $u "atlas-maps:8080";`,
  `proxy_pass http://$u$request_uri;`):

  ```
  location ~ ^/api/worlds/[^/]+/channels/[^/]+/maps/[^/]+/instances/[^/]+/jukebox(/.*)?$ {
    set $u "atlas-maps:8080";
    proxy_pass http://$u$request_uri;
  }
  ```

- `deploy/k8s/base/routes.conf.template.generated`,
  `deploy/k8s/base/ns-vars.generated.yaml` — regenerated via
  `tools/gen-routes.sh` from the worktree root. Only
  `routes.conf.template.generated` changed (the new location block, resolving
  to `atlas-maps.${NS_ATLAS_MAPS}.svc.cluster.local:8080`); `ns-vars.generated.yaml`
  has no diff because `NS_ATLAS_MAPS` was already defined (atlas-maps already
  owns other map-instance subresources). Confirmed `tools/gen-routes.sh --check`
  exits 0 after regeneration.

- `tools/gen-routes_test.sh` — added check 5, an awk-scoped assertion that
  extracts the generated jukebox `location` block from
  `routes.conf.template.generated` and asserts it contains `atlas-maps`. This
  is the regression guard.

## Other task-252 REST surfaces check

Read `services/atlas-maps/atlas.com/maps/map/jukebox/resource.go` — it
registers exactly one route,
`/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/jukebox`
(GET only). No other new REST route was added by task-252, so one `location`
block is sufficient. No further routes.conf changes were needed.

## TDD Evidence

**RED** — stashed the three route files (source + both generated), ran the
guard against the pre-fix state:

```
$ git stash push -- deploy/shared/routes.conf deploy/k8s/base/routes.conf.template.generated deploy/k8s/base/ns-vars.generated.yaml
$ bash tools/gen-routes_test.sh
gen-routes: up to date
FAIL: no map-instance jukebox location routed to atlas-maps in routes.conf.template.generated
exit=1
```

This is expected: pre-fix, the generated file has no jukebox location block,
so the awk scan over "lines from the jukebox `location` header to its closing
brace" finds nothing to grep, and the `grep -q 'atlas-maps'` fails.

**GREEN** — restored the fix (`git stash pop`) and reran:

```
$ bash tools/gen-routes_test.sh
gen-routes: up to date
PASS
```

## Files changed

- `deploy/shared/routes.conf` — added jukebox location block after `weather`.
- `deploy/k8s/base/routes.conf.template.generated` — regenerated (jukebox
  block added, resolving to `atlas-maps.${NS_ATLAS_MAPS}...`).
- `deploy/k8s/base/ns-vars.generated.yaml` — regenerated, no diff (NS var
  already present).
- `tools/gen-routes_test.sh` — added check 5 regression guard.

No Go code was touched, per the brief's constraint. Since no Go module
changed, module-local `go build`/`go test` do not apply to this fix; the
relevant verification is `bash tools/gen-routes_test.sh` (shown green above)
and `tools/gen-routes.sh --check` (exits 0).

## Self-review

- Block placement matches the brief exactly (after `weather`, before
  `monsters`, precedes the `^/api/worlds(/.*)?$` catch-all).
- Style matches surrounding blocks exactly (indentation, `set $u`,
  `proxy_pass`).
- Generated files committed alongside the source edit and the guard, in one
  commit, per the controller's ruling.
- Guard fails before the fix and passes after — verified directly by
  stash/pop rather than just asserted.
- Did not touch `announceActiveJukebox`, the atlas-maps resource, the
  registry, Kafka edges, or the packet codec.
- Confirmed no other task-252 resource.go/requests.go introduces an
  unrouted path.

## Issues or concerns

None. The fix is a single, narrowly-scoped nginx-routing addition with a
regression guard; no code changes were required or made.

## Outcome

- Fix commit: `35a842331` — "fix: route map-instance jukebox subresource to
  atlas-maps"
- Gate: pending (controller's job)
- Live re-test on `atlas-pr-1457`: pending (controller's job)
