# Fix: `GET /api/rings` missing from ingress routing table

**Bug:** `docs/tasks/task-269-ring-pair-behavior/bug-rings-route-missing-from-ingress.md`
**Commit:** `1721d8d25` — "fix(deploy): route GET /api/rings to atlas-cashshop via ingress"
**Branch:** `task-269-ring-pair-behavior` (worktree
`.worktrees/task-269-ring-pair-behavior`)

## Scope

Per the controller's rulings, this fix touches only the routes.conf block,
its regeneration, the `routes_nginxt.sh` assertion, and the
`atlas-channel` REST-dependency doc line. No Go file was changed;
`ring/processor.go` was read for context only, never edited.

## What changed

1. **`deploy/shared/routes.conf`** — added a new location block after the
   `^/api/coupon-redemptions(/.*)?$` block (same atlas-cashshop-owned
   surface group), with a comment following the file's existing style
   explaining why the block is needed (the resource is registered at the
   cashshop router root, not under `/cash-shop`):

   ```
   location ~ ^/api/rings(/.*)?$ {
     set $u "atlas-cashshop:8080";
     proxy_pass http://$u$request_uri;
   }
   ```

2. **`deploy/k8s/base/routes.conf.template.generated`** — regenerated via
   `tools/gen-routes.sh` (no hand-edit). `ns-vars.generated.yaml` was
   regenerated too but produced no diff, confirming `NS_ATLAS_CASHSHOP`
   already exists.

3. **`deploy/shared/test/routes_nginxt.sh`** — added a python block (same
   block-extraction style as the existing atlas-renders header check)
   that:
   - finds the `location ~ ^/api/rings(/.*)?$` block and asserts it
     proxies to `atlas-cashshop`
   - asserts `/api/rings?filter[characterId]=1` and
     `/api/rings/9c5b3b1a-1111-4a2b-9c3d-000000000001` both match that
     location's regex pattern

4. **`services/atlas-channel/docs/rest.md`** — the file does enumerate
   upstream REST dependencies (`## External Service Dependencies`,
   including a `### CASHSHOP` section). Added a `GET
   /rings?filter[characterId]={characterId}` entry there, following the
   section's existing per-endpoint format (Parameters / Request Model /
   Response Model / Error Conditions), noting the router-root registration
   and the ingress block that now routes it, and the fail-soft/cache
   semantics from `ring.ProcessorImpl.Populate`.

## Verification (both required commands, exit 0)

```
$ bash tools/gen-routes.sh --check; echo "GEN_CHECK_EXIT: $?"
gen-routes: up to date
GEN_CHECK_EXIT: 0

$ bash deploy/shared/test/routes_nginxt.sh; echo "NGINXT_EXIT: $?"
...
nginx: the configuration file /etc/nginx/nginx.conf syntax is ok
nginx: configuration file /etc/nginx/nginx.conf test is successful
routes.conf MinIO upstream cross-namespace check: OK
routes.conf atlas-renders tenant header check: OK
routes.conf character-render hash-length check: OK
routes.conf rings route check: OK
routes drift check (shared vs k8s-generated): OK
NGINXT_EXIT: 0
```

Note on sequencing: `routes_nginxt.sh`'s F18 drift check regenerates
`routes.conf.template.generated` and then `git diff --quiet`s it against
the committed copy, restoring the committed copy via `git checkout --` on
failure. Run against an uncommitted working tree this correctly fails
(nothing is committed yet to diff against), and reverts the regenerated
file. The fix: regenerate again, then commit `routes.conf` +
`routes.conf.template.generated` (+ the test and doc changes) together, then
re-run — which is what happened here; the second run (shown above, post
commit `1721d8d25`) is clean.

## Files changed

- `deploy/shared/routes.conf`
- `deploy/k8s/base/routes.conf.template.generated` (regenerated, not
  hand-edited)
- `deploy/shared/test/routes_nginxt.sh`
- `services/atlas-channel/docs/rest.md`

`deploy/k8s/base/ns-vars.generated.yaml` was regenerated as part of the
`tools/gen-routes.sh` run but produced no diff and was not staged.

## Self-review

- Route pattern `^/api/rings(/.*)?$` covers both `ring/resource.go` routes:
  the list `/rings?filter[characterId]=` (path is exactly `/api/rings`,
  matched by the bare-`$` empty-group alternative) and the item
  `/rings/{ringId}` (matched by the `(/.*)?` group). Confirmed by the new
  `routes_nginxt.sh` python assertion, which passed.
- No Go file touched; `ring/processor.go` was read-only for context.
- `routes.conf.template.generated` was produced solely by
  `tools/gen-routes.sh`, never hand-edited — confirmed by the `--check`
  pass and the `git diff --quiet` step inside `routes_nginxt.sh`.
- Did not touch the cash-id join logic in `selectPair`
  (`ring/processor.go:170-200`) per the brief's "Not yet answered" note —
  that is out of scope and re-tested live separately.

## Issues / concerns

None. Both required checks exit 0 (pasted above). The "Not yet answered"
cash-id-join question from the bug file is explicitly out of scope for this
fix and was not investigated further.
