# Review: bug-event-definition-seed-yields-zero.md (commit c5e8b8bbe, range f4e62a93d..c5e8b8bbe)

## Scope

`git diff --stat f4e62a93d..c5e8b8bbe`:

```
deploy/seed/shared/all/events/definitions/event-anniversary.json     |  2 +
deploy/seed/shared/all/events/definitions/event-crimson-balrog.json  |  2 +
docs/tasks/.../bug-event-definition-seed-yields-zero.md              | 174 ++
libs/atlas-seeder/handlers.go                                        | 49 +-
libs/atlas-seeder/handlers_test.go                                   | 59 ++
services/atlas-pr-bootstrap/scripts/bootstrap.sh                     |  2 +
services/atlas-ui/src/lib/hooks/api/__tests__/useSeed.test.tsx       | 21 +
services/atlas-ui/src/lib/hooks/api/useSeed.ts                       | 38 ++
services/atlas-ui/src/pages/SetupPage.tsx                            | 16 +
services/atlas-ui/src/pages/__tests__/SetupPage.test.tsx             | 19 +-
services/atlas-ui/src/services/api/seed.service.ts                   | 22 +
tools/catalog-lint/subdomains.go                                     |  1 +
```

This matches the bug file's five-part fix list exactly (envelope repair, linter
rule, bootstrap endpoints, `postSeed` logging, party-quests Setup row). No
scope mismatch.

## Findings

### 1. Seed envelope repair — PASS

`deploy/seed/shared/all/events/definitions/event-anniversary.json` now has
`"data.id": "anniversary"`, `"data.type": "event-definition"`; the crimson-balrog
file has `"data.id": "crimson-balrog"`, `"data.type": "event-definition"`.

Checked against `services/atlas-events/atlas.com/events/event/definition/subdomain.go:27-31`:
`Type()` returns `"event-definition"` (match), `EntityIDPattern()` is
`^event-(.+)\.json$` applied to the filenames `event-anniversary.json` /
`event-crimson-balrog.json` → captures `anniversary` / `crimson-balrog`, both
equal to the new `data.id` values (match).

### 2. catalog-lint rule — PASS

`tools/catalog-lint/subdomains.go` gained
`{path: "events/definitions", typ: "event-definition", pattern: regexp.MustCompile(`^event-(.+)\.json$`)}`,
identical path/type/pattern triple to the subdomain contract above. Ran
`go run ./tools/catalog-lint deploy/seed` — exits 0, no output (clean).

### 3. bootstrap.sh endpoints — PASS

`services/atlas-pr-bootstrap/scripts/bootstrap.sh:489-490` adds
`/api/events/definitions/seed` and `/api/party-quests/definitions/seed` inside
the existing `endpoints=( ... )` array (confirmed via `sed -n '478,494p'` — both
new lines sit between the existing `/api/maps/actions/seed` entry and the
closing `)`, syntactically valid bash array elements).

Path-prefix match confirmed against `deploy/shared/routes.conf`:
- `location ~ ^/api/events(/.*)?$` → `atlas-events:8080` (routes.conf, "events"
  block) — matches `/api/events/definitions/seed`.
- `location ~ ^/api/party-quests(/.*)?$` → `atlas-party-quests:8080` — matches
  `/api/party-quests/definitions/seed`.

Confirmed the target services actually serve those paths:
`services/atlas-events/.../definition/subdomain.go:108-109` registers
`URLPrefix: "/events/definitions"` under `seeder.RegisterRoutes`, which appends
`/seed`; `atlas-events/main.go:44` mounts the router at prefix `/api/`, giving
`/api/events/definitions/seed`. `services/atlas-party-quests/.../definition/groups.go:18`
registers `URLPrefix: "/party-quests/definitions"` the same way.

### 4. postSeed logging shape on clean run — PASS (verified by revert-and-test)

`libs/atlas-seeder/handlers.go:65-80`: the `else` branch (taken when
`classifyOutcome(res.Subdomains) == "success"`) is byte-identical to the
pre-fix unconditional log call — same `logrus.Fields` keys
(`tenant_id`, `group_name`, `catalog_revision`, `subdomains`), same message
`"Seed complete"`, same `.Info` level. The new `outcome`/`failed`/`errors`
fields and `.Error` level appear only in the `if outcome != "success"` branch.
Verified this by diffing the pre- and post-commit `postSeed` bodies directly
(`git diff f4e62a93d..c5e8b8bbe -- libs/atlas-seeder/handlers.go`) — the
`else` block contains zero net changes versus the old unconditional block.

`summarizeFailed`/`summarizeErrors` (new, `handlers.go:118-142`) only populate
entries for subdomains with `Failed > 0` / non-empty `Errors`, so a clean run's
`summarize()` map (unchanged, still `Created`-only) is the only subdomain data
in the info-level log — no added noise on success.

**Test-honesty check**: temporarily reverted `handlers.go` to the pre-fix
version and reran `TestRegisterRoutes_AllFilesRejectedLogsError` in isolation —
it fails (`level = info, want error`) against the old code, and passes against
the new code. The new test is not vacuous. File restored afterward;
`git status` on `libs/atlas-seeder/` is clean.

### 5. Party-quests UI wiring — PASS

`seed.service.ts` adds `PartyQuestDefinitionsSeedStatus`,
`seedPartyQuestDefinitions()` (POST `/api/party-quests/definitions/seed`), and
`getPartyQuestDefinitionsSeedStatus()` (GET
`/api/party-quests/definitions/seed/status`, `subdomainCount(s, "definition.partyquest")`).
Confirmed `DefinitionSubdomain.Name()` in
`services/atlas-party-quests/atlas.com/party-quests/definition/subdomain.go:21`
returns exactly `"definition.partyquest"` — the key matches.

`useSeed.ts` adds the mutation hook, status key, and query hook with
`staleTime: 0, refetchInterval: 5000`, matching the shape of
`useSeedEventDefinitions`/`useEventDefinitionsSeedStatus`.

`SetupPage.tsx` adds a `"Party Quest Definitions"` row wired to the new
mutation/status hooks with a `Users` icon — same shape as the existing rows.

Tests: `useSeed.test.tsx` covers polling/key-by-tenant via the shared
`describe.each` table and a dedicated mutation test asserting the service call
happens on `.mutate()`. `SetupPage.test.tsx` updates the row-count assertion
(twelve → thirteen) and adds a click-triggers-mutate test. Both are consistent
with the existing per-row test pattern for `03e870091`'s event-definitions row.

## Not evaluable

None — every item in the review brief was checked against source, and
`go run ./tools/catalog-lint deploy/seed` and the targeted `go test` were
actually executed rather than assumed from the commit message.

## Verdict

APPROVED. All five fix parts match the bug file's prescription; the clean-run
log shape is provably unchanged (diff-verified and revert-tested); the seed
envelopes, linter rule, and UI wiring all check out against the authoritative
subdomain contracts they claim to match.
