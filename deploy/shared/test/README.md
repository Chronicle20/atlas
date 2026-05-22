# atlas-ingress route tests

## routes_nginxt.sh

Syntax-validates `deploy/shared/routes.conf` by running it through `nginx -t`
inside a `nginx:alpine` container. Quick, no upstreams, no behavioural
assertions.

Requires: docker.

```sh
bash deploy/shared/test/routes_nginxt.sh
```

The script wraps `routes.conf` in a minimal `http{}` / `server{}` stanza
(because `routes.conf` is normally included inside the ingress's `server`
block) and asks nginx to parse it. A non-zero exit means the config has
syntax or directive-context errors that nginx will refuse to load.

## Deferred: upstream-stub regression harness

Task 071 plan §15.3 calls for a richer test that spins nginx + a small Go
upstream stub and asserts each route lands at the right upstream with the
right rewritten path. That work is deferred — track it as a follow-up.
The shape of the future harness:

1. `upstream-stub.go` — recording HTTP server that logs `Host` and request
   path on every hit.
2. `expectations.txt` — list of `(request URL, expected upstream, expected
   forwarded path)` tuples covering character render, map render hit/miss,
   per-tenant asset hit, shared-fallback asset miss, `/api/data/wz` upload,
   and the generic `/api/data` catch-all.
3. `routes_test.sh` — starts nginx + the stub, replays expectations, and
   asserts the stub's access log matches.
