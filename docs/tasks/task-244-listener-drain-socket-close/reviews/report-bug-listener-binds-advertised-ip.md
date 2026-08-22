# Implementer report — bug-listener-binds-advertised-ip

- **Agent**: atlas-implementer (sonnet)
- **Status**: DONE
- **Commit**: `f66935f2b` — `fix(atlas-channel): bind the wildcard address instead of the advertised IP`

Recorded by the controller: the implementer's `Write` was blocked by harness
policy on report/summary files, so it returned this content in its final
message instead.

## Files changed

- `services/atlas-channel/atlas.com/channel/socket/init.go` —
  `socket.Bind(l, ipAddress, port)` → `socket.Bind(l, "0.0.0.0", port)`. Error
  wrap changed from `"bind %s:%d: %w"` (ipAddress, port) to `"bind port %d: %w"`
  (port only), since it no longer binds `ipAddress`. Doc comment expanded to
  state that `ipAddress` is advertisement-only and why the pod cannot bind its
  own advertised address. The
  `channel.NewProcessor(...).Register(sc.Channel(), ipAddress, port)`
  advertisement call is unchanged.
- `services/atlas-channel/atlas.com/channel/socket/init_test.go` — added
  `TestCreateSocketServiceBindsTheWildcardAddressNotTheAdvertisedAddress`:
  calls `CreateSocketService` with advertised `ipAddress="192.0.2.1"`
  (RFC 5737 TEST-NET-1, non-assignable) and port `0`; asserts a non-nil
  listener, a nil error, and that the bound address is the wildcard
  (`IP.IsUnspecified()`), not `192.0.2.1`. On the pre-fix code this fails with
  `EADDRNOTAVAIL`. The two pre-existing `CreateSocketService` tests were left
  unchanged, per the brief.

`libs/atlas-socket/server.go` was not touched, as the bug file directed.

## Test summary

`go build ./... && go test ./...` from
`services/atlas-channel/atlas.com/channel` — all packages pass, including
`ok atlas-channel/socket 0.063s`. A verbose run of the three
`CreateSocketService*` tests shows `Starting tcp server on [0.0.0.0:0]` even
when `ipAddress="192.0.2.1"` was passed in, so the wildcard bind is directly
observed rather than only asserted.

## Self-review

Diff confirms only the bind call, the error wrap, and the doc comment changed
in `init.go`; the advertisement path is untouched. The new test follows the
file's existing Builder pattern (`server.NewProcessor(...).Register(...)` with
a `server.GetRegistry().Deregister` cleanup) and adds no test-only
constructors.

## Concerns

None. The bug file's `## Not yet answered` item — unbounded retry with no
backoff ceiling in the projection apply loop — was explicitly out of scope and
was not acted on.
