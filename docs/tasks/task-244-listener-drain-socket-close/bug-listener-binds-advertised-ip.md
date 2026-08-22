# Bug — channel listener binds the advertised IP instead of the wildcard address

- **Task**: task-244-listener-drain-socket-close
- **Branch**: `task-244-listener-drain-socket-close` (head at diagnosis: `3ee4b6442`)
- **Found in**: live testing of PR-1441 ephemeral environment (`atlas-pr-1441`)
- **Tenant**: `d9af99a3-dd83-47da-8182-7a070b59390e` — GMS, ms.version 83.1
- **Severity**: blocker — atlas-channel never accepts a connection on this branch.

## Reproduced

Log into the PR-1441 environment with the GMS 83.1 client. Login, world select,
character select, and character creation all succeed. Selecting a character
hands the client a channel address and the client then fails to join the
channel.

## Observed

`atlas-channel` has **zero** listeners. It is in a bind-retry hot loop —
`retries: 1050` and climbing at diagnosis time, roughly one retry per 250ms per
(world, channel):

```
{"log.level":"info","message":"Starting tcp server on [192.168.23.191:8301]","world.id":0,...}
{"log.level":"error","error":{"message":"listen tcp 192.168.23.191:8301: bind: cannot assign requested address"},
 "message":"Error listening on [192.168.23.191:8301].","world.id":0,...}
{"log.level":"info","message":"Starting tcp server on [192.168.23.191:7901]","world.id":1,...}
{"log.level":"error","error":{"message":"listen tcp 192.168.23.191:7901: bind: cannot assign requested address"},
 "message":"Error listening on [192.168.23.191:7901].","world.id":1,...}
{"log.level":"debug","message":"projection.applied add_failed","retries":1050,
 "key":{"TenantId":"d9af99a3-...","WorldId":1,"ChannelId":0},
 "error":{"message":"bind 192.168.23.191:7901: listen tcp 192.168.23.191:7901: bind: cannot assign requested address"}}
```

atlas-login is healthy and completes the whole flow. It resolves the channel
endpoint and hands it to the client:

```
GET /api/worlds/0/channels/0
 -> {"worldId":0,"channelId":0,"ipAddress":"192.168.23.191","port":8301,...}
```

`192.168.23.191` is the host LAN address the channel *advertises* to clients. It
is not assigned to any interface inside the pod, so `net.Listen` on it returns
`EADDRNOTAVAIL`.

The user's "add a world" action did not cause this; it only added a second
failing bind (world 1 → port 7901) alongside the pre-existing world 0 → 8301.

## Expected

The channel binds the wildcard address and advertises the configured
`ipAddress` to clients, so the listener accepts connections arriving at the
pod regardless of which address the client used to reach it.

## Root cause

`services/atlas-channel/atlas.com/channel/socket/init.go:79` — introduced by
this branch's `dbb7fcb1e` ("bind the socket synchronously and return the
listener from CreateSocketService") — passes the **advertised** address as the
**bind** address:

```go
lis, err := socket.Bind(l, ipAddress, port)
if err != nil {
    return nil, fmt.Errorf("bind %s:%d: %w", ipAddress, port, err)
}
```

Before this branch, `ipAddress` was advertisement-only. The listener came from

```go
err := socket.Run(l, ctx, wg, socket.SetHandlers(hp), socket.SetPort(port), ...)
```

with **no** `socket.SetIpAddress(...)` call, so the bind used the `config`
default `ipAddress: "0.0.0.0"` (`libs/atlas-socket/server.go:113`). Nothing in
the repository calls `socket.SetIpAddress` — verified by grep across all Go
sources; the only `SetIpAddress`/`SetIPAddress` hits are unrelated builder
methods in atlas-login's `channel` package and atlas-ban's `history` package.
`ipAddress` reached the network layer for the first time on this branch.

`ipAddress` is still used correctly for advertisement at
`services/atlas-channel/atlas.com/channel/socket/init.go:141`
(`channel.NewProcessor(l, ctx).Register(sc.Channel(), ipAddress, port)`) — that
call must not change.

The bind failure is not fatal because `listener.Registry.Add`'s rollback path
returns the error to the projection apply loop, which retries forever. The pod
stays `Running` and `Ready`; the only symptom is the log loop and a channel
that never accepts.

## Fix

Bind the wildcard address; keep `ipAddress` as the advertised address only,
restoring pre-task-244 network behavior without giving up the synchronous-bind
contract this task added.

**Files:**

- `services/atlas-channel/atlas.com/channel/socket/init.go:79-82` — bind the
  wildcard address instead of `ipAddress`. Keep the `fmt.Errorf` wrap
  informative: it should still name the port and should not claim a bind
  address it did not use. Update the `CreateSocketService` doc comment
  (lines 65-74) to state that `ipAddress` is advertisement-only and the
  listener binds the wildcard address, and say why (a pod cannot bind the
  advertised host address).
- `services/atlas-channel/atlas.com/channel/socket/init.go:141` — unchanged.
  `Register(sc.Channel(), ipAddress, port)` must keep using `ipAddress`.
- `services/atlas-channel/atlas.com/channel/socket/init_test.go` — add a
  regression test that pins the separation: call `CreateSocketService` with an
  advertised `ipAddress` that is **not** assignable on the test host (use
  `192.0.2.1`, TEST-NET-1 per RFC 5737) and a port of `0`, and assert the call
  returns a non-nil listener and a nil error. On the buggy code this fails with
  `EADDRNOTAVAIL`. Assert the returned listener's `Addr()` is a wildcard bind,
  not `192.0.2.1`.
  Note the two existing tests (`TestCreateSocketServiceReturnsErrorWhenPortIsAlreadyBound`
  at :47, `TestCreateSocketServiceReturnsTheBoundListener` at :84) pass
  `"127.0.0.1"` as `ipAddress` and will still pass after the change — the
  already-bound test still conflicts because a wildcard bind collides with a
  `127.0.0.1` bind on the same port. Do not delete them, but they no longer pin
  the bind address; the new test is what does.

Do **not** change `libs/atlas-socket/server.go`. `Bind(l, ipAddress, port)`
taking an explicit address is correct as a library primitive, and `Run` still
correctly resolves `c.ipAddress` from the configurators (defaulting to
`0.0.0.0`). The defect is only in what atlas-channel passes.

Scope is atlas-channel's `socket` package only. Module-local verification:
`go build ./... && go test ./...` from
`services/atlas-channel/atlas.com/channel`.

## Not yet answered

- Whether the projection apply loop's unbounded retry on a permanently
  unsatisfiable bind (1050+ retries, no backoff ceiling observed in the logs)
  deserves a separate fix. It is not required to resolve this bug and is out of
  scope for this fix — record it, do not act on it.

## Resolution

Fixed by `f66935f2b` — `fix(atlas-channel): bind the wildcard address instead
of the advertised IP`. `CreateSocketService` now calls
`socket.Bind(l, "0.0.0.0", port)`; `ipAddress` is used only by the `Register`
advertisement call. `TestCreateSocketServiceBindsTheWildcardAddressNotTheAdvertisedAddress`
pins the separation by passing a non-assignable advertised address
(`192.0.2.1`) and asserting the bound address is unspecified.

Implementer report: `reviews/report-bug-listener-binds-advertised-ip.md`.

**Gate**: `tools/verify.sh --quick --base 3ee4b6442` — exit 0. Docker bake
skipped (`--quick`), so this is not a pre-PR pass; the flagless run is still
required before the branch is called done.

**Live re-test**: NOT yet performed. The branch has not been pushed since the
fix, so PR-1441 is still running the broken image. Confirming this bug closed
requires a push, a rollout, and a channel join in `atlas-pr-1441` — expect
`Starting tcp server on [0.0.0.0:8301]` with no `add_failed` retries.
