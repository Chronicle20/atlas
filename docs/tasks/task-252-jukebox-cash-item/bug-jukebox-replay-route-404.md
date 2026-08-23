# bug: jukebox song does not replay on map re-entry — the replay REST call 404s at the ingress

**Reproduced:** tenant GMS v83.1, PR environment `atlas-pr-1457`
(namespace `atlas-pr-1457`), 2026-08-23 22:49–22:53 UTC, map 240000000,
instance `00000000-0000-0000-0000-000000000000`, item 5100000, duration 67 s.
Steps: character 1 uses the jukebox in map 240000000; while the song is still
playing, character 1 (caster) and character 2 (observer) each leave and
re-enter the map.

**Observed:** on re-entry the jukebox track stops and the map's own BGM plays,
for the caster and the observer alike.

The start/end broadcasts are fine — atlas-maps registers and expires the entry,
atlas-channel broadcasts both edges:

```
atlas-maps    22:51:51 Jukebox started in map [240000000] instance [000...000] with item [5100000] by [Atlas] for [1m7.03s].
atlas-channel 22:51:51 Jukebox started in map [240000000] instance [000...000] with item [5100000] by [Atlas].
atlas-channel 22:52:58 Jukebox ended in map [240000000] instance [000...000].
```

The **replay** path is the one that fails. atlas-channel issues the GET on every
map entry, as designed:

```
atlas-channel 22:52:31.090 Issuing [GET] request to
  [http://atlas-ingress.atlas-pr-1457.svc.cluster.local:80/api/worlds/0/channels/0/maps/240000000/instances/00000000-0000-0000-0000-000000000000/jukebox].
```

and the ingress answers **404** — every time, including 22:52:31, which is
inside the 22:51:51 → 22:52:58 song window:

```
atlas-ingress 22:49:23 "GET /api/worlds/0/channels/0/maps/240000000/instances/000.../jukebox HTTP/1.1" 404 19 "-" "Go-http-client/1.1"
atlas-ingress 22:51:45 ... 404 19
atlas-ingress 22:51:58 ... 404 19   (observer, char 2, song active)
atlas-ingress 22:52:09 ... 404 19   (observer, char 2, song active)
atlas-ingress 22:52:31 ... 404 19   (caster,   char 1, song active)
```

`announceActiveJukebox` fails open on any error
(`services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:1153-1163`),
so the 404 is swallowed silently and no `PlayJukebox` packet is ever sent to the
entering session. The client therefore builds a fresh `CField` whose
`m_nJukeBoxItemID` is 0, and `CField::LoadMap` → `CMapLoadable::PlayBGMFromMapInfo`
(v95 `0x5469f0` → `0x61a330`) plays the map's own BGM. That is the observed
behaviour, and it is entirely explained by the missing packet — no client-side
race is involved.

**Expected:** a character entering a map with an in-progress jukebox song
receives `PlayJukebox` for the active item and hears the song for the remainder
of its duration (PRD FR: map-enter replay; design §2 "Map-enter replay",
plan task "broadcast and replay jukebox songs on map status events",
commit `c5d9bdfcc`).

**Root cause:** the ingress routing table has no `location` block for the
map-instance `jukebox` subresource. `deploy/shared/routes.conf` enumerates one
regex `location` per map-instance subresource — `characters` (line 480),
`weather` (line 486), `monsters` (line 490), `summons`, `dragons`, … — and
task-252 added the atlas-maps resource
(`services/atlas-maps/atlas.com/maps/map/jukebox/resource.go:22-26`,
`/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/jukebox`)
**without** adding its route. With no specific block, the request falls through
to the catch-all `location ~ ^/api/worlds(/.*)?$` at
`deploy/shared/routes.conf:672`, which proxies to **atlas-world:8080** — a
service that has no such route — producing the 404 seen above.

The generated deployment routing tables are derived from that file
(`tools/gen-routes.sh` → `deploy/k8s/base/routes.conf.template.generated`,
`deploy/k8s/base/ns-vars.generated.yaml`), so they are missing it too, and
`tools/gen-routes.sh --check` will fail if the source is edited without
regenerating.

## Fix

- `deploy/shared/routes.conf` — add, immediately after the `weather` block
  (line 486-489) so it precedes the `^/api/worlds(/.*)?$` catch-all:

  ```
  location ~ ^/api/worlds/[^/]+/channels/[^/]+/maps/[^/]+/instances/[^/]+/jukebox(/.*)?$ {
    set $u "atlas-maps:8080";
    proxy_pass http://$u$request_uri;
  }
  ```

  Match the surrounding blocks exactly — two-space indent, `set $u "<svc>:8080";`,
  `proxy_pass http://$u$request_uri;`.
- `deploy/k8s/base/routes.conf.template.generated`,
  `deploy/k8s/base/ns-vars.generated.yaml` — regenerate by running
  `tools/gen-routes.sh` from the worktree root; do not hand-edit. Confirm
  `tools/gen-routes.sh --check` exits 0 afterwards and both generated files are
  committed with the source change.
- `tools/gen-routes_test.sh` — add a fourth-style assertion (the file is a
  plain `fail()`-based shell script with numbered checks) that the generated
  routes contain a map-instance `jukebox` location whose upstream resolves to
  `atlas-maps`. This is the regression guard: it fails before the routes.conf
  edit and passes after. Run `bash tools/gen-routes_test.sh` and show it green.

No Go code changes. `announceActiveJukebox`, the atlas-maps resource, the
registry, the Kafka edges, and the packet codec are all correct as implemented
and must not be touched.

## Not yet answered

- Whether other task-252 REST surfaces are similarly unrouted: the jukebox
  resource is the only new REST route the task added
  (`services/atlas-maps/atlas.com/maps/map/jukebox/resource.go`), so one block
  is expected to be sufficient. If a grep of the task's changed
  `resource.go`/`requests.go` files turns up another path with no matching
  `location` in `deploy/shared/routes.conf`, add that block too and note it in
  the report rather than leaving it for a second round.
- Deploying the fix to `atlas-pr-1457` and re-testing in-game is the
  controller's job after the gate, not the fix agent's.

## Outcome

- Fix commit: `35a842331` — "fix: route map-instance jukebox subresource to
  atlas-maps". Adds the `location` block to `deploy/shared/routes.conf`,
  regenerates `deploy/k8s/base/routes.conf.template.generated`
  (`ns-vars.generated.yaml` unchanged — `NS_ATLAS_MAPS` already existed), and
  adds the coverage assertion to `tools/gen-routes_test.sh`, verified red on
  the pre-fix files and green after. No Go changes.
- Gate: PASS — `tools/verify.sh --quick --base 1f25b0316` exit 0
  (`gen-routes_test.sh`, `routes drift`, `shell tooling guard`, `overlay env
  drift` all green; Go/UI layers skipped, no Go or TS file changed). Docker
  bake skipped, so this is not a pre-PR pass.
- Live re-test on `atlas-pr-1457`: **pending** — requires redeploying the PR
  environment so the ingress ConfigMap picks up the new routing table, then
  repeating the reproduction (use jukebox, leave and re-enter the map inside
  the song window, for the caster and an observer). Confirm via
  `kubectl -n atlas-pr-1457 logs <atlas-ingress-pod> | grep jukebox` that the
  GET now returns 200 instead of 404.
