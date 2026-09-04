# Review — Task 7: atlas-maps back-effect REST collection

Commit range: `372af1f83..HEAD` (single commit `a84d158`, "feat(atlas-maps): serve active back-effects over REST")
Brief: `.superpowers/sdd/plan/task-7-brief.md`
Report: `.superpowers/sdd/plan/task-7-report.md`

## Scope

`git diff --stat 372af1f83..HEAD`:

```
services/atlas-maps/atlas.com/maps/main.go         |   2 +
.../atlas.com/maps/map/backeffect/resource.go      |  54 +++++++++
.../atlas.com/maps/map/backeffect/resource_test.go | 121 +++++++++++++++++++++
.../atlas.com/maps/map/backeffect/rest.go          |  34 ++++++
4 files changed, 211 insertions(+)
```

Matches the brief's file list exactly — no drift into Task 5 (registry.go/processor.go) or Task 6 (kafka handler/producer) files. Scope confirmed.

## Findings

### 1. `RestModel` matches the brief's "Interfaces produced" block

`services/atlas-maps/atlas.com/maps/map/backeffect/rest.go:5-11`:

```go
type RestModel struct {
	Id       string `json:"-"`
	Effect   uint8  `json:"effect"`
	FieldId  uint32 `json:"fieldId"`
	PageId   uint8  `json:"pageId"`
	Duration uint32 `json:"duration"`
}
```

Field names, types, and json tags are byte-for-byte identical to the brief (task-7-brief.md:59-66). `GetName()` returns `"backEffect"` (`rest.go:15-17`), matching the brief and the test assertion at `resource_test.go:83`. `Transform` (`rest.go:24-32`) sets `Id: strconv.Itoa(int(e.PageId))` as specified, and does an explicit `byte`→`uint8` cast for `Effect`/`PageId` (no-op, since `byte` is an alias for `uint8` in Go) — cosmetic only, does not change wire shape. PASS.

### 2. Route registration matches the jukebox pattern exactly

`services/atlas-maps/atlas.com/maps/map/backeffect/resource.go:20-23`:

```go
r := router.PathPrefix("/worlds").Subrouter()
r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/backEffects", rest.RegisterHandler(l)(si)(getBackEffectsInMap, handleGetBackEffectsInMap)).Methods(http.MethodGet)
```

Full effective path: `/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/backEffects` — exact match to the brief. Diffed byte-for-byte against `map/jukebox/resource.go:23-28`: identical `InitResource` shape, identical `rest.ParseWorldId` → `rest.ParseChannelId` → `rest.ParseMapId` → `rest.ParseInstanceId` chain (`resource.go:26-30` vs `jukebox/resource.go:31-35`), same `field.NewBuilder(...).SetInstance(instanceId).Build()` field construction. PASS.

### 3. Empty result is 200 with an empty JSON array, not 404/null

`resource.go:33`: `res := make([]RestModel, 0, len(entries))` — confirmed present, matches brief's Step 4 code verbatim. `TestGetBackEffectsInMap_EmptyIsTwoHundred` (`resource_test.go:99-121`) drives a request against a field with no entries set for a fresh tenant and asserts `http.StatusOK` (`resource_test.go:114`, not 404) and `assert.Len(t, doc.Data.DataArray, 0)` (`resource_test.go:120`). The collection marshal path (`server.MarshalResponse[[]RestModel]`, `resource.go:37`) is the same generic used by `report/resource.go` per the brief's cited precedent, and with a `make(..., 0, ...)` slice it serializes to `[]`, never `null`, consistent with Go's `encoding/json` behavior for non-nil empty slices. Confirmed both by static allocation and by a green test that explicitly names the PRD §5 deviation in its doc comment (`resource_test.go:96-98`). PASS.

### 4. `Id` derivation and `SetID` round-trip

`Id: strconv.Itoa(int(e.PageId))` (`rest.go:26`) — page id as string, matches brief. `SetID` exists (`rest.go:19-22`):

```go
func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}
```

`GetID` returns `m.Id` (`rest.go:15-17`). Round-trip is symmetric with `jukebox/rest.go:11-22`'s identical shape. Test confirms the wire-level id: `assert.Equal(t, "1", doc.Data.DataArray[0].ID)` and `"2"` for the second entry (`resource_test.go:82,90`), matching `PageId: 1` / `PageId: 2` set via `p.Set(...)` (`resource_test.go:61-62`). PASS.

### 5. Tests drive the real router with a fresh tenant per case

`setupBackEffectRouter()` (`resource_test.go:29-35`) builds a real `*mux.Router` via `InitResource(&backEffectTestServerInformation{})(r, l)`, and both tests wrap it in `httptest.NewServer(setupBackEffectRouter())` (`resource_test.go:64`, `resource_test.go:104`) and issue actual HTTP requests through `http.Client{}.Do(req)` — not a direct handler-function call. Each test calls `tenantId := uuid.New()` independently (`resource_test.go:54`, `resource_test.go:100`), so the two tests cannot collide in the package-level registry (`getRegistry()` keys by `FieldKey{Tenant: t, Field: f}` per `registry.go:46`). PASS.

### 6. Route initializer registered in `main.go`

`services/atlas-maps/atlas.com/maps/main.go:15` adds the import `"atlas-maps/map/backeffect"` beside `"atlas-maps/map/jukebox"`, and `main.go:151` adds `AddRouteInitializer(backeffect.InitResource(GetServer())).` immediately after `AddRouteInitializer(jukebox.InitResource(GetServer())).` and before `AddRouteInitializer(visit.InitResource(...))`. Confirmed via diff:

```
+	"atlas-maps/map/backeffect"
 	"atlas-maps/map/jukebox"
...
+		AddRouteInitializer(backeffect.InitResource(GetServer())).
 		AddRouteInitializer(visit.InitResource(GetServer())(db)).
```

PASS.

## Build/test verification (module-local only, no repo-wide gate run)

```
cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./map/backeffect/... -v
```

All 6 tests pass, including the two new resource tests (`TestGetBackEffectsInMap_ReturnsEntries`, `TestGetBackEffectsInMap_EmptyIsTwoHundred`). Consistent with the report's GREEN evidence.

## Not evaluable

- Task 9's atlas-channel client mirror of `RestModel` was not reviewed here (out of this unit's scope — this task only produces the server-side contract). Drift detection at that seam is Task 9's reviewer's job.
- The vendor `atlas-rest/server.MarshalResponse` / `jsonapi` library internals were not read line-by-line; the empty-vs-null behavior was inferred from Go's `encoding/json` semantics for non-nil empty slices plus the brief's stated precedent (`report/resource.go`) rather than by tracing the generic's implementation. Low risk given the passing test asserts the observed wire behavior end-to-end via `httptest`.

## Verdict

APPROVED. All six brief requirements verified against file:line evidence; the two new tests genuinely exercise the real router with tenant isolation, and the deliberate 200-with-empty-array deviation from PRD §5 is both implemented and explicitly pinned by a test.
