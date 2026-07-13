# Fleet-wide 503 Adoption — task-168

Follow-on to the atlas-inventory reference implementation (Task 11): **every
DB-backed service now maps transient DB connection errors to `503 Service
Unavailable` + `Retry-After: 1`** instead of a bare `500`, per the pattern in
`.claude/skills/backend-dev-guidelines/resources/patterns-resilience.md` and
enforced by DOM-27 (503) and DOM-28 (no silent degradation).

## What each service got

1. `main.go` — registers the transient classifier immediately after
   `database.Connect`:
   ```go
   server.RegisterTransientErrorClassifier(func(err error) bool {
       if database.IsTransientConnectionError(err) {
           database.CountTransient(err)
           return true
       }
       return false
   })
   ```
2. Every REST handler `500`-write — `w.WriteHeader(http.StatusInternalServerError)`,
   local `writeErrorResponse(w, http.StatusInternalServerError, …)`, and
   `http.Error(w, …, 500)` — replaced with
   `server.WriteErrorResponse(d.Logger())(w)(err)`. 400/404/409/other-status
   branches were left untouched; local `writeErrorResponse` helpers were kept
   where still used for non-500 codes.

No `go.mod`/`go.sum` changes: every service already depended on
`atlas-database` (via `database.Connect`) and `atlas-rest` (via the REST
server); only new symbols in already-required modules are used.

## Services adopted (32 DB-backed services with 500 handlers)

atlas-account, atlas-ban, atlas-buddies, atlas-cashshop, atlas-character,
atlas-configurations, atlas-data, atlas-drop-information, atlas-families,
atlas-gachapons, atlas-guilds, atlas-inventory (Task 11 reference), atlas-keys,
atlas-map-actions, atlas-maps, atlas-marriages, atlas-merchant,
atlas-monster-book, atlas-mounts, atlas-mts, atlas-notes,
atlas-npc-conversations, atlas-npc-shops, atlas-party-quests, atlas-pets,
atlas-portal-actions, atlas-quest, atlas-reactor-actions,
atlas-saga-orchestrator, atlas-skills, atlas-storage, atlas-tenants.

**atlas-fame**: DB-backed but has zero `500`-writing REST handlers — nothing to
adopt; no classifier registered (a classifier with no `WriteErrorResponse`
consumer would be dead code).

## Non-mechanical sites handled explicitly

- **atlas-maps** `character/location/resource.go` — `changeCharacterLocation`
  was a pure `int`-status-returning (unit-tested) helper; refactored to return
  `(int, error)` so the caller routes true 500s through `WriteErrorResponse`
  while preserving the 404/400/204 paths. Its 5 unit-test call sites were
  updated with `err` assertions.
- **atlas-data** — also used the `http.Error(w, …, 500)` idiom and a
  `baseline/handler.go` shared `code` variable that could resolve to 422;
  split into an explicit 422 branch + `WriteErrorResponse` for the true 500.
- **atlas-tenants** — four handlers had a manual `json.NewEncoder(w).Encode(…)`
  after the old `WriteHeader`; removed to avoid double-writing the body (which
  `WriteErrorResponse` now emits in full JSON:API form).
- **atlas-npc-conversations** `ReindexRecipesHandler` — a failed type-assertion
  invariant guard (not a DB error, no `err` in scope) was routed through
  `WriteErrorResponse` with a synthesized error so its 500 still yields a
  JSON:API body; the classifier returns false for it, so it stays 500.

## Verification

- Full fleet grep sweep: zero bare `500` writes remain in any DB-backed
  service; 32 classifiers registered (all DB services except atlas-fame).
- `go build` / `go vet` / `go test` clean per changed service module.
- `tools/redis-key-guard.sh` clean.
- `docker buildx bake all-go-services` — all images build.
