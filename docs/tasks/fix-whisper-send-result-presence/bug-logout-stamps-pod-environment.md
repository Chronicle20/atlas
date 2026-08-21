# bug: session-timeout logout stamps the pod's own environment, so every logout is dropped

**Reproduced:** yes, live, in `atlas-pr-1412` (sparse environment; only
`atlas-channel` and `atlas-login` are overridden — every other service is the
`atlas-main` baseline). Tenant `08b47980-b83f-4aae-a8fd-d9f0f2d0d1c4`,
region GMS, version 83.1. Character 32 (`chronicle`), world 0 channel 0,
map 240000000.

## Observed

A character logs out. The account's session state flips to logged out, but the
character stays visible in the map to other players, and a whisper addressed to
them is still produced and delivered.

Evidence, in order, from the live pods:

1. `atlas-character` (atlas-main, `atlas-character-7bc8674fb-qs85w`), the
   session-timeout task's logout for character 32:

   ```
   22:20:31.833  Issuing [GET] request to [http://atlas-ingress.atlas-main.../api/characters/32/location].
   22:20:31.835  error="bad request"  Logout: atlas-maps lookup failed for [32]
                 (infrastructure error); emitting with zero map.
   ```

2. The resulting event therefore carries `mapId: 0`:

   ```
   22:20:31.888  {"transactionId":"2b09988f-...","worldId":0,"characterId":32,
                  "type":"LOGOUT","body":{"channelId":0,"mapId":0,"instance":"000...0"}}
   ```

3. Both consumers **drop** that event at the ownership gate:

   ```
   atlas-maps    22:20:31.891  Dropping message: environment is unresolvable.
                               reason="mismatched" topic=EVENT_TOPIC_CHARACTER_STATUS-main
   atlas-channel 22:20:31.888  Dropping message: environment is unresolvable.
                               reason="mismatched" topic=EVENT_TOPIC_CHARACTER_STATUS-main
   ```

   `atlas-maps` never logs `Character [32] has logged out` — it logged the
   LOGIN at 22:20:18 and nothing after. So `_map.ExitAndEmit` never ran and
   `location.SetState(OFFLINE)` never ran.

4. The location row consequently stays live. 35 seconds after the logout:

   ```
   22:21:07  GET /api/characters/32/location
             → {"worldId":0,"channelId":0,"mapId":240000000,"state":"IN_FIELD"}
   ```

5. Which is exactly what this PR's whisper gate reads, so it produces:

   ```
   22:20:28.226  branch="in-field" "whisper resolved" target_name=chronicle
   ```

Reproduced by hand against the live pods (curl from inside the cluster, same
tenant headers, varying only `ENVIRONMENT`):

| target | `ENVIRONMENT` | result |
|---|---|---|
| `atlas-maps.atlas-main:8080/api/characters/32/location` | *(absent)* | 200 |
| same | `pr-1412` | 200 |
| same | `main` | **400** |

and the matching server-side log:

```
atlas-maps 22:26:34.939  Environment header disagrees with the tenant's environment.
  error="environment header disagrees with the tenant's environment:
         header=\"main\" tenant=\"pr-1412\"(08b47980-b83f-4aae-a8fd-d9f0f2d0d1c4)"
  originator=get_character_location
```

## Expected

Logging out removes the character from the map's presence set and flips
`character_locations.state` to `OFFLINE`, so a whisper to them announces
`WhisperSendResult(false)` and is not delivered.

## Root cause

`services/atlas-character/atlas.com/character/session/task.go` — the
session-timeout sweep runs outside any request/message, so it has no
environment on the context and `main.go` threads one in. That closure
(`services/atlas-character/atlas.com/character/main.go:150-152`) stamps
**this pod's own** environment:

```go
session.NewTimeout(l, db, ..., func(ctx context.Context) context.Context {
    return env.WithContext(ctx, env.Self())
})
```

For a baseline pod serving a *sparse* environment's tenant, that is the wrong
environment. The pod is `main`; the tenant belongs to `pr-1412`. Every
downstream check reconciles the operation's environment against the tenant's
and rejects the disagreement outright (FR-7.7 — a hard error, deliberately not
a reconciliation):

- REST: `libs/atlas-rest/server/handler.go:151` → `env.Reconcile` →
  `ErrEnvironmentMismatch` → 400. That is the "bad request" in step 1, which
  is why `Logout` fell into its `mapId 0` fallback
  (`services/atlas-character/atlas.com/character/character/processor.go:559-567`).
- Kafka: `libs/atlas-kafka/consumer/header.go:79` `EnvHeaderParser` →
  `env.WithMismatch` → `libs/atlas-kafka/consumer/gate.go:68` →
  `gateDropUnresolvable`/`reasonMismatched`. That is step 3, and it is what
  actually leaves the character in the map: the LOGOUT event is acknowledged
  and discarded by every consumer, in every deployment.

The environment of a piece of tenant-scoped background work is a property of
**the tenant**, not of the pod that happens to run the sweep. `env.Self()` is
correct only where there is genuinely no tenant to derive from.

This is a baseline (`main`) defect, not a regression from this PR. It is filed
here because the PR's whisper gate is the thing it makes visibly wrong: the
gate's logic is correct, the presence data it reads is stale. Fixing it on this
branch also makes it live-testable — sparse overrides are derived from the
PR's changed services, so touching `atlas-character` puts an
`atlas-character` override into `atlas-pr-1412`.

## Fix

- `libs/atlas-env/tenants.go` — add a pure resolver beside `Reconcile`:

  ```go
  // ForTenant resolves the environment for tenant-scoped work that arrived
  // with no environment of its own (a background sweep, not a request).
  func ForTenant(r Registry, tenantId string, self Id) Id
  ```

  Rules: type-assert `r` to the existing `TenantResolver` exactly as
  `Reconcile` does; if the assert fails, `tenantId == ""`, the tenant is
  unknown, or its projected environment is `""` (legacy — see the existing
  comment in `Reconcile` on why empty is legacy and not "belongs to nothing"),
  return `self`. Otherwise return the tenant's environment. Never returns `""`
  when `self` is non-empty — the FR-1.8 guarantee `NewTimeout`'s comment
  depends on must survive.

- `libs/atlas-env/tenants_test.go` — table test, one case per rule above,
  including "tenant belongs to another environment → that environment, not
  self", which is the case this bug is.

- `services/atlas-character/atlas.com/character/session/task.go` — signature
  UNCHANGED. Every call site in the repo already applies `envContext` to a
  context that ALREADY carries the tenant
  (`t.envContext(tenant.WithContext(sctx, m.Tenant()))`), and so do all four
  sibling services, so the seam stays `func(context.Context) context.Context`
  everywhere and only the installed closure changes. Update the `NewTimeout`
  doc comment: the threaded value is no longer "this pod's own environment
  identity" but "the environment that owns the session's tenant, falling back
  to this pod's". Do not make `session` import `atlas-env` — the
  env-domain-guard import list is why this is a function value.

- `libs/atlas-service/` — NEW, the shared origination closure all five call
  sites use. That module already requires both `atlas-env` and `atlas-tenant`,
  and every service's `main.go` already imports it for `service.Bootstrap`:

  ```go
  // TenantEnvironment originates the environment for tenant-scoped background
  // work: the environment that owns the tenant already on ctx, falling back to
  // this pod's own (env.Self()) when ctx carries no tenant or the registry
  // does not know it.
  func TenantEnvironment(ctx context.Context) context.Context
  ```

- `services/atlas-character/atlas.com/character/main.go:150-152` — replace the
  inline closure with `service.TenantEnvironment`.

- `services/atlas-character/atlas.com/character/session/task_test.go` (extend)
  — assert that `sessionTenantContext` passes the session's tenant id through
  to the seam, and that the tenant survives on the returned context.

- `services/atlas-character/atlas.com/character/character/processor.go` — do
  **not** change `Logout`'s `mapId 0` fallback. It is a correct fallback for a
  genuinely absent location; the bug is that it was reached, not what it does.

## Not yet answered

- **The same defect class exists in four other background tasks.** The user
  ruled on 2026-08-20 that they are IN SCOPE for this branch, so they are no
  longer open questions — see
  `sweep-tenant-environment-origination.md` for the brief. Sites:
  - `services/atlas-guilds/atlas.com/guilds/main.go:122-124` (`guild/task.go`)
  - `services/atlas-pets/atlas.com/pets/main.go:115-117` (`pet/task.go`)
  - `services/atlas-channel/atlas.com/channel/main.go:349` (`character/combo/task.go`,
    via `socket.WithSelfEnvironment`)
  - `services/atlas-trades/atlas.com/trades/main.go:79` (`trade/settlement.go`,
    via `trade.SetEnvContext`)

- `env.Self()` in the socket paths (`atlas-channel`, `atlas-login`
  `socket/init.go`, `socket/handler/handle.go`) is **correct** and must not be
  touched — a game client's connection genuinely belongs to the pod that
  terminates it, and the value must never be read off the wire.

- Whether `atlas-maps` should also make its LOGOUT handling idempotent against
  a missed event (e.g. reconcile presence against session state on a timer).
  Out of scope — do not build it.

## Resolution

Fixed in two commits on `fix-whisper-send-result-presence`:

- **`c96a0f5e7`** `fix(atlas-character): resolve tenant-owning environment for
  the session-timeout logout sweep` — adds `env.ForTenant` (pure resolver,
  beside `Reconcile`) and `service.TenantEnvironment` (the shared origination
  closure, in `libs/atlas-service` because it is the only module that already
  requires both `atlas-env` and `atlas-tenant`), and reduces
  `atlas-character`'s `main.go` wiring to passing the helper directly. The
  `session/task.go` seam signature is unchanged — the tenant was already on
  the context at every call site, which is what made the widened-seam design
  in the `## Fix` section above unnecessary. Tests: `libs/atlas-env` 42/42,
  `libs/atlas-service` 51/51, `atlas-character` 368/368.

- **`9f67a6332`** `fix(services): derive background sweep environment from
  tenant, not pod, in four services` — the sibling sweep (`atlas-guilds`,
  `atlas-pets`, `atlas-channel`'s combo decay tick, `atlas-trades`), per
  `sweep-tenant-environment-origination.md`. `socket.WithSelfEnvironment` and
  the socket/listener paths are untouched; `atlas-trades`' now-dead
  `withSelfEnvironment` was deleted rather than left behind. Those four sites
  are the same code shape as the defect reproduced here but were NOT
  separately reproduced live — recorded in that commit's message.

**Live re-test: not yet performed.** The reproduction above is what must be
re-run once `atlas-pr-1412` rolls the new images. The sparse override set is
derived from the PR's changed services, so `atlas-character` (and now
`atlas-guilds`/`atlas-pets`/`atlas-channel`/`atlas-trades`) will be overridden
in that namespace and the fix becomes observable there. The check is: log
character 32 out, then confirm

- `atlas-character` no longer logs `Logout: atlas-maps lookup failed … emitting
  with zero map`, and the LOGOUT event carries the real `mapId`, not `0`;
- `atlas-maps` logs `Character [32] has logged out` and NO
  `reason="mismatched"` drop appears on `EVENT_TOPIC_CHARACTER_STATUS-main`;
- `GET /api/characters/32/location` returns `state: OFFLINE`;
- a whisper to `chronicle` announces `WhisperSendResult(false)` and is not
  delivered.

Until that runs, this bug is fixed-but-unconfirmed.

### Review remediation

`atlas-reviewer` returned CHANGES_REQUIRED against `89aea18c9..9f67a6332`
(artifact: `review-tenant-environment.md`) with two blocking findings, both
confirmed by hand before acting and both fixed in **`98ef8b11a`**
`fix(atlas-env): restore Reconcile doc comment, assert tenant on envContext
input`:

1. `ForTenant` had been inserted between `Reconcile`'s doc comment and
   `func Reconcile`, so Go's adjacency rule merged Reconcile's whole doc block
   — including its CAUTION paragraph on parser registration order — onto
   `ForTenant` and left `Reconcile` undocumented (`go doc -all .` printed
   nothing for it). Relocated; both now print their own text.

2. `TestProcessExpiredCoordinationsAppliesEnvContextToAct` (atlas-guilds) and
   `TestProcessExpiriesAppliesEnvContextToCancel` (atlas-channel combo) only
   asserted that a spy marker round-tripped downstream, so both passed
   identically before and after the fix. They now capture the context
   `envContext` RECEIVES and assert the tenant is already on it — the property
   the whole change rests on, since a tenant-less ctx would make
   `service.TenantEnvironment` fall back to `env.Self()` and silently keep the
   bug. The atlas-pets and atlas-trades equivalents were re-checked
   independently and were already adequate.

The reviewer confirmed as sound: `ForTenant`'s rule set, `TenantEnvironment`'s
inability to produce a context disagreeing with its own tenant, the tenant's
presence on ctx at all five wiring sites (traced by hand), and the untouched
socket/listener paths.

### Gate

`tools/verify.sh --quick --base 89aea18c9` exits **0** at `98ef8b11a` — zero
FAIL lines, `env domain guard ✓`, `lint & format guard (89 modules) ✓`.

An earlier run of the same command reported `undefined: env.ForTenant` across
21 service modules via `libs/atlas-service`. That was NOT a defect: the gate
was running while the review-remediation agent was mid-edit in the same
worktree, relocating `ForTenant` in `libs/atlas-env/tenants.go`, so `go vet`
compiled the file in the window between the cut and the paste. Do not run a
gate and an implementer against one worktree concurrently.

`--quick` skips the docker bake and `-race`. The flagless `tools/verify.sh`
still has to pass before this branch opens a PR.
