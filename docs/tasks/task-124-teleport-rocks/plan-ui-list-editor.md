# Teleport-Rock List Editor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add admin CRUD over a character's Regular/VIP teleport-rock lists to the atlas-ui Character Detail page, backed by synchronous REST endpoints on atlas-character.

**Architecture:** atlas-character gains synchronous `Add`/`Remove` processor methods and POST/DELETE REST handlers that share the existing validate→mutate→emit logic with the game path (validation returns typed errors → HTTP status; success emits `LIST_UPDATED` so online characters refresh). atlas-ui gets a JSON:API service, React Query hooks, and two card components rendered at the bottom of the character detail page.

**Tech Stack:** Go (atlas-character, DDD immutable models + processors + JSON:API), Vite + React + React Router + React Query + shadcn/ui + sonner (atlas-ui).

## Global Constraints

- Work only in the worktree `.worktrees/task-124-teleport-rocks`; branch `task-124-teleport-rocks`.
- Backend verification (run from worktree root): `go test -race ./...`, `go vet ./...`, `go build ./...` clean in `services/atlas-character`; `tools/lint.sh --check` clean.
- No new Go module/lib is added → no `Dockerfile`/`go.work`/docker-bake change; no new REST route prefix → no ingress change (POST/DELETE extend the existing `/characters/{characterId}/teleport-rock-maps` path).
- atlas-ui is **Vite + React Router**, NOT Next.js: no `next/*`, no `"use client"`; use the `@/` import alias. Write tests with `vitest` (`vi.*`), not `jest`.
- atlas-ui verification: `cd services/atlas-ui && npm run build && npm test` (source nvm 22 first). Gate on build + tests passing and no new lint errors.
- Regular list capacity = 5, VIP = 10 (from `teleport_rock/model.go`: `RegularCapacity`, `VipCapacity`, `Capacity(vip)`).
- Header rock icons: Regular = item `5040000`, VIP = item `5041000`.
- Follow the codebase norm for mapped REST errors: bare `w.WriteHeader(status)` (per `character/character/resource.go`); the UI maps status → message.

---

### Task 1: atlas-character — typed-error processor refactor + synchronous Add/Remove

Refactor validation to return typed errors; keep the async game path emitting the ERROR status event; add synchronous `Add`/`Remove` returning `(Model, error)`.

**Files:**
- Modify: `services/atlas-character/atlas.com/character/teleport_rock/processor.go`
- Test: `services/atlas-character/atlas.com/character/teleport_rock/processor_test.go`

**Interfaces:**
- Consumes: existing `getByCharacterId`, `modelFromEntities`, `replaceList`, `EligibleForRegistration`, `Capacity`, `ListType`, `Contains`; event providers `listUpdatedEventProvider(txId, worldId, characterId, vip, registered, maps)`, `errorEventProvider(txId, worldId, characterId, vip, reason)`; constants `teleportrock2.EnvEventTopicStatus`, `ErrorReasonMapNotAllowed|ListFull|Duplicate|NotFound`.
- Produces: sentinel errors `ErrMapNotAllowed`, `ErrListFull`, `ErrDuplicate`, `ErrNotFound`; processor methods `Add(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) (Model, error)` and `Remove(...) (Model, error)` on the `Processor` interface + `ProcessorImpl`.

- [ ] **Step 1: Write the failing tests**

Append to `processor_test.go` (mirror the existing `testDatabase(t)` / `testContext(t)` helpers already in this file):

```go
func TestAddReturnsUpdatedModelOnSuccess(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.Add(uuid.New(), 0, 42, 100000000, false)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(m.Regular()) != 1 || m.Regular()[0] != 100000000 {
		t.Fatalf("regular list = %v, want [100000000]", m.Regular())
	}
}

func TestAddReturnsTypedErrors(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	// ineligible map (0 fails EligibleForRegistration)
	if _, err := p.Add(uuid.New(), 0, 42, 0, false); !errors.Is(err, ErrMapNotAllowed) {
		t.Fatalf("ineligible: got %v, want ErrMapNotAllowed", err)
	}
	// duplicate
	if _, err := p.Add(uuid.New(), 0, 42, 100000000, false); err != nil {
		t.Fatalf("seed add: %v", err)
	}
	if _, err := p.Add(uuid.New(), 0, 42, 100000000, false); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate: got %v, want ErrDuplicate", err)
	}
	// remove-not-present
	if _, err := p.Remove(uuid.New(), 0, 42, 200000000, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("remove absent: got %v, want ErrNotFound", err)
	}
}

func TestAddReturnsListFull(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	// Fill the regular list to capacity (5) with eligible maps.
	for _, mid := range []_map.Id{100000000, 101000000, 102000000, 103000000, 104000000} {
		if _, err := p.Add(uuid.New(), 0, 42, mid, false); err != nil {
			t.Fatalf("seed add %d: %v", mid, err)
		}
	}
	if _, err := p.Add(uuid.New(), 0, 42, 105000000, false); !errors.Is(err, ErrListFull) {
		t.Fatalf("full: got %v, want ErrListFull", err)
	}
}
```

Add the imports `"errors"` and `test "github.com/sirupsen/logrus/hooks/test"` if not already present (the file already uses `_map`, `uuid`). If existing tests use eligible map ids other than the `10x000000` range, reuse whatever `EligibleForRegistration` accepts (check `model.go` `EligibleForRegistration`).

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-character/atlas.com/character && go test ./teleport_rock/ -run 'TestAdd|TestRemove' -count=1`
Expected: FAIL — `p.Add`/`p.Remove` undefined, `ErrMapNotAllowed` etc. undefined.

- [ ] **Step 3: Implement the refactor**

In `processor.go`, add the sentinels and extract validation to return them; layer the async ERROR emission in the `AndEmit` wrappers; add synchronous `Add`/`Remove`. Add both methods to the `Processor` interface.

```go
// Typed validation errors: the REST path maps these to HTTP status; the async
// game path (AddMapAndEmit/RemoveMapAndEmit) maps them to ERROR status events.
var (
	ErrMapNotAllowed = errors.New("map not allowed")
	ErrListFull      = errors.New("list full")
	ErrDuplicate     = errors.New("duplicate map")
	ErrNotFound      = errors.New("map not found")
)

// reasonForError maps a validation sentinel to its wire ERROR reason (game path).
func reasonForError(err error) (string, bool) {
	switch {
	case errors.Is(err, ErrMapNotAllowed):
		return teleportrock2.ErrorReasonMapNotAllowed, true
	case errors.Is(err, ErrListFull):
		return teleportrock2.ErrorReasonListFull, true
	case errors.Is(err, ErrDuplicate):
		return teleportrock2.ErrorReasonDuplicate, true
	case errors.Is(err, ErrNotFound):
		return teleportrock2.ErrorReasonNotFound, true
	default:
		return "", false
	}
}
```

Rewrite `AddMap` so validation returns a typed error and success buffers `LIST_UPDATED` (the returned Model is via a captured pointer so the sync path can read it):

```go
// addMap validates then mutates. On validation failure it returns a typed error
// and buffers nothing. On success it buffers LIST_UPDATED and reports the new model.
func (p *ProcessorImpl) addMap(mb *message.Buffer, transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) (Model, error) {
	var updated Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		es, err := getByCharacterId(tx, p.t.Id(), characterId)
		if err != nil {
			return err
		}
		m := modelFromEntities(characterId, es)
		list := m.List(vip)

		if !EligibleForRegistration(mapId) {
			return ErrMapNotAllowed
		}
		if len(list) >= Capacity(vip) {
			return ErrListFull
		}
		if m.Contains(vip, mapId) {
			return ErrDuplicate
		}

		newList := append(append([]_map.Id{}, list...), mapId)
		if err := replaceList(tx, p.t.Id(), characterId, ListType(vip), newList); err != nil {
			return err
		}
		b := NewBuilder().SetCharacterId(characterId)
		if vip {
			updated = b.SetRegular(m.Regular()).SetVip(newList).Build()
		} else {
			updated = b.SetRegular(newList).SetVip(m.Vip()).Build()
		}
		return mb.Put(teleportrock2.EnvEventTopicStatus, listUpdatedEventProvider(transactionId, worldId, characterId, vip, true, newList))
	})
	return updated, txErr
}
```

The `Model` is built via the existing fluent `NewBuilder()` in `builder.go`
(`SetCharacterId`/`SetRegular`/`SetVip`/`Build`) — the same path `modelFromEntities`
uses. The key invariant: `updated` reflects the post-mutation lists.

Async game wrapper — catch the typed error and emit the ERROR event (preserves current behavior):

```go
func (p *ProcessorImpl) AddMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		_, err := p.addMap(buf, transactionId, worldId, characterId, mapId, vip)
		if err == nil {
			return nil
		}
		if reason, ok := reasonForError(err); ok {
			p.l.Warnf("Character [%d] add-map [%d] rejected: %s.", characterId, mapId, reason)
			return buf.Put(teleportrock2.EnvEventTopicStatus, errorEventProvider(transactionId, worldId, characterId, vip, reason))
		}
		return err // infrastructure error (db) — propagate
	})
}
```

Sync REST path — propagate the typed error, flush `LIST_UPDATED` on success:

```go
func (p *ProcessorImpl) Add(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) (Model, error) {
	var updated Model
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		m, e := p.addMap(buf, transactionId, worldId, characterId, mapId, vip)
		if e != nil {
			return e // Emit discards the buffer; no event on failure
		}
		updated = m
		return nil
	})
	return updated, err
}
```

Repeat the identical structure for `removeMap`/`RemoveMapAndEmit`/`Remove` (validation: `!m.Contains(vip, mapId)` → `ErrNotFound`; success compacts the list and buffers `listUpdatedEventProvider(..., false, newList)` — copy the compaction from the existing `RemoveMap`). Add `Add` and `Remove` to the `Processor` interface. Delete the now-unused old `AddMap`/`RemoveMap` buffer methods **only if** nothing else references them; otherwise keep them delegating to `addMap`/`removeMap` for the command consumer. (Check the consumer: `kafka/consumer/teleportrock/consumer.go` calls `AddMapAndEmit`/`RemoveMapAndEmit`, so the exported `AddMap`/`RemoveMap` may be removable — grep first.)

Update `teleport_rock/mock/processor.go` to add `Add`/`Remove` function fields with nil-safe defaults.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-character/atlas.com/character && go test ./teleport_rock/ -count=1`
Expected: PASS (new tests + existing `TestAddMap*`/`TestRemoveMap*`).

- [ ] **Step 5: Verify the async ERROR behavior is preserved**

Run: `go test ./kafka/consumer/teleportrock/... -count=1` (and any test asserting ERROR events).
Expected: PASS. If an existing test asserted an ERROR event was buffered by `AddMap` directly, re-point it at `AddMapAndEmit`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character/atlas.com/character/teleport_rock/processor.go \
        services/atlas-character/atlas.com/character/teleport_rock/processor_test.go \
        services/atlas-character/atlas.com/character/teleport_rock/mock/processor.go
git commit -m "feat(task-124): synchronous teleport-rock Add/Remove with typed errors"
```

---

### Task 2: atlas-character — capacity in RestModel + POST/DELETE handlers

**Files:**
- Modify: `services/atlas-character/atlas.com/character/teleport_rock/rest.go`
- Modify: `services/atlas-character/atlas.com/character/teleport_rock/resource.go`
- Modify: `services/atlas-character/atlas.com/character/README.md`
- Test: `services/atlas-character/atlas.com/character/teleport_rock/rest_test.go`

**Interfaces:**
- Consumes: `Processor.Add`/`Remove` (Task 1); `ErrMapNotAllowed|ErrListFull|ErrDuplicate|ErrNotFound`; `character.NewProcessor(l, ctx, db).GetById()(characterId) (character.Model, error)` with `character.Model.WorldId() world.Id`; `rest.RegisterInputHandler[M]`, `rest.RegisterHandler`, `rest.ParseCharacterId`, `server.WriteErrorResponse`, `server.MarshalResponse`.
- Produces: `RestModel` fields `RegularCapacity int json:"regularCapacity"`, `VipCapacity int json:"vipCapacity"`; input model `AddMapInputRestModel{ List string json:"list"; MapId uint32 json:"mapId" }`; routes `POST` and `DELETE` on `/characters/{characterId}/teleport-rock-maps`.

- [ ] **Step 1: Write the failing tests**

Extend `rest_test.go`. Test (a) `Transform` sets capacities and (b) the POST/DELETE handlers return the right status for each typed error. Mirror the existing rest_test setup (it already builds a `RestModel`/`Transform`). Example for the capacity + a handler status case (use `httptest` + the mock processor if the existing rest_test does; otherwise test `Transform` + a thin `statusForError` helper):

```go
func TestTransformIncludesCapacities(t *testing.T) {
	m := NewBuilder().SetCharacterId(42).SetRegular([]_map.Id{100000000}).Build()
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if rm.RegularCapacity != RegularCapacity || rm.VipCapacity != VipCapacity {
		t.Fatalf("capacities = %d/%d, want %d/%d", rm.RegularCapacity, rm.VipCapacity, RegularCapacity, VipCapacity)
	}
}

func TestStatusForError(t *testing.T) {
	cases := map[error]int{
		ErrMapNotAllowed: http.StatusBadRequest,
		ErrListFull:      http.StatusConflict,
		ErrDuplicate:     http.StatusConflict,
		ErrNotFound:      http.StatusNotFound,
	}
	for err, want := range cases {
		if got := statusForError(err); got != want {
			t.Errorf("statusForError(%v) = %d, want %d", err, got, want)
		}
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd services/atlas-character/atlas.com/character && go test ./teleport_rock/ -run 'TestTransformIncludesCapacities|TestStatusForError' -count=1`
Expected: FAIL — capacities absent, `statusForError` undefined.

- [ ] **Step 3: Implement rest.go**

Add capacity fields + the input model + the status helper:

```go
type RestModel struct {
	Id              string    `json:"-"`
	Regular         []_map.Id `json:"regular"`
	Vip             []_map.Id `json:"vip"`
	RegularCapacity int       `json:"regularCapacity"`
	VipCapacity     int       `json:"vipCapacity"`
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:              strconv.Itoa(int(m.CharacterId())), // match existing GetID convention
		Regular:         m.Regular(),
		Vip:             m.Vip(),
		RegularCapacity: RegularCapacity,
		VipCapacity:     VipCapacity,
	}, nil
}

// AddMapInputRestModel is the POST body: {data:{type:"teleport-rock-maps", attributes:{list,mapId}}}.
type AddMapInputRestModel struct {
	Id    string `json:"-"`
	List  string `json:"list"`
	MapId uint32 `json:"mapId"`
}

func (r AddMapInputRestModel) GetName() string        { return "teleport-rock-maps" }
func (r AddMapInputRestModel) GetID() string          { return r.Id }
func (r *AddMapInputRestModel) SetID(id string) error { r.Id = id; return nil }

func statusForError(err error) int {
	switch {
	case errors.Is(err, ErrMapNotAllowed):
		return http.StatusBadRequest
	case errors.Is(err, ErrListFull), errors.Is(err, ErrDuplicate):
		return http.StatusConflict
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	default:
		return 0 // caller falls back to WriteErrorResponse (500)
	}
}
```

Keep the existing `GetName`/`GetID`/`SetID` on `RestModel` unchanged. Preserve whatever the current `Transform` used for `Id`.

- [ ] **Step 4: Implement resource.go routes + handlers**

Register POST/DELETE alongside the existing GET:

```go
r := router.PathPrefix("/characters/{characterId}/teleport-rock-maps").Subrouter()
r.HandleFunc("", registerGet("get_teleport_rock_maps", handleGetTeleportRockMaps)).Methods(http.MethodGet)
r.HandleFunc("", rest.RegisterInputHandler[AddMapInputRestModel](l)(db)(si)("add_teleport_rock_map", handleAddTeleportRockMap)).Methods(http.MethodPost)
r.HandleFunc("/{list}/{mapId}", registerGet("remove_teleport_rock_map", handleRemoveTeleportRockMap)).Methods(http.MethodDelete)
```

```go
func handleAddTeleportRockMap(d *rest.HandlerDependency, c *rest.HandlerContext, input AddMapInputRestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			vip, ok := listVip(input.List)
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			worldId, err := characterWorldId(d, characterId)
			if err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).
				Add(uuid.New(), worldId, characterId, _map.Id(input.MapId), vip)
			if err != nil {
				if s := statusForError(err); s != 0 {
					w.WriteHeader(s)
					return
				}
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			writeModel(d, c, w, r, m)
		}
	})
}

func handleRemoveTeleportRockMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			vip, ok := listVip(mux.Vars(r)["list"])
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			mapId, err := strconv.ParseUint(mux.Vars(r)["mapId"], 10, 32)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			worldId, err := characterWorldId(d, characterId)
			if err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).
				Remove(uuid.New(), worldId, characterId, _map.Id(mapId), vip)
			if err != nil {
				if s := statusForError(err); s != 0 {
					w.WriteHeader(s)
					return
				}
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			writeModel(d, c, w, r, m)
		}
	})
}

func listVip(list string) (bool, bool) {
	switch list {
	case ListTypeRegular:
		return false, true
	case ListTypeVip:
		return true, true
	default:
		return false, false
	}
}

func characterWorldId(d *rest.HandlerDependency, characterId uint32) (world.Id, error) {
	cm, err := character.NewProcessor(d.Logger(), d.Context(), d.DB()).GetById()(characterId)
	if err != nil {
		return 0, err
	}
	return cm.WorldId(), nil
}

func writeModel(d *rest.HandlerDependency, c *rest.HandlerContext, w http.ResponseWriter, r *http.Request, m Model) {
	res, err := model.Map(Transform)(model.FixedProvider(m))()
	if err != nil {
		d.Logger().WithError(err).Errorf("Creating REST model.")
		server.WriteErrorResponse(d.Logger())(w)(err)
		return
	}
	query := r.URL.Query()
	queryParams := jsonapi.ParseQueryFields(&query)
	server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
}
```

Add imports: `"errors"`, `"strconv"`, `"github.com/gorilla/mux"`, `"github.com/google/uuid"`, the `character` package, and `world`. Confirm the `character` import path doesn't create a cycle (teleport_rock → character). If it does, resolve worldId via a thin function passed in at resource-registration time, or read it from the `character` GORM row directly in a provider within teleport_rock. Check with `go build` and adjust.

- [ ] **Step 5: Run tests + build**

Run: `cd services/atlas-character/atlas.com/character && go test ./teleport_rock/ -count=1 && go build ./...`
Expected: PASS + clean build.

- [ ] **Step 6: Update README**

In `services/atlas-character/atlas.com/character/README.md`, add to the REST endpoints table:
`POST /characters/{characterId}/teleport-rock-maps` (body `{data:{type:"teleport-rock-maps",attributes:{list,mapId}}}` → 200 updated lists / 400 ineligible / 409 full|duplicate) and `DELETE /characters/{characterId}/teleport-rock-maps/{list}/{mapId}` (→ 200 / 404 not present). Note the GET now returns `regularCapacity`/`vipCapacity`.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-character/atlas.com/character/teleport_rock/rest.go \
        services/atlas-character/atlas.com/character/teleport_rock/resource.go \
        services/atlas-character/atlas.com/character/teleport_rock/rest_test.go \
        services/atlas-character/atlas.com/character/README.md
git commit -m "feat(task-124): teleport-rock POST/DELETE REST + capacity in read model"
```

---

### Task 3: atlas-ui — teleport-rocks service + hooks

**Files:**
- Create: `services/atlas-ui/src/services/api/teleport-rocks.service.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/useTeleportRocks.ts`
- Test: `services/atlas-ui/src/services/api/__tests__/teleport-rocks.service.test.ts`

**Interfaces:**
- Consumes: `api` from `@/lib/api/client` (`api.getOne`, `api.post`, `api.delete`); `useTenant` from `@/context/tenant-context`; React Query.
- Produces: `teleportRocksService.getByCharacterId(characterId): Promise<TeleportRockLists>`, `.addMap(characterId, list, mapId): Promise<TeleportRockLists>`, `.removeMap(characterId, list, mapId): Promise<TeleportRockLists>` where `type ListType = "regular" | "vip"` and `interface TeleportRockLists { regular: number[]; vip: number[]; regularCapacity: number; vipCapacity: number }`; hooks `useTeleportRockMaps(tenant, characterId)`, `useAddTeleportRockMap()`, `useRemoveTeleportRockMap()`, and `teleportRockKeys`.

- [ ] **Step 1: Write the failing service test**

```ts
import { describe, it, expect, vi, beforeEach } from "vitest";
import { teleportRocksService } from "@/services/api/teleport-rocks.service";
import { api } from "@/lib/api/client";

vi.mock("@/lib/api/client", () => ({
  api: { getOne: vi.fn(), post: vi.fn(), delete: vi.fn() },
}));

const resource = {
  id: "42", type: "teleport-rock-maps",
  attributes: { regular: [100000000], vip: [], regularCapacity: 5, vipCapacity: 10 },
};

describe("teleportRocksService", () => {
  beforeEach(() => vi.clearAllMocks());

  it("getByCharacterId flattens the resource", async () => {
    (api.getOne as ReturnType<typeof vi.fn>).mockResolvedValue(resource);
    const r = await teleportRocksService.getByCharacterId("42");
    expect(api.getOne).toHaveBeenCalledWith("/api/characters/42/teleport-rock-maps");
    expect(r).toEqual({ regular: [100000000], vip: [], regularCapacity: 5, vipCapacity: 10 });
  });

  it("addMap posts the JSON:API envelope", async () => {
    (api.post as ReturnType<typeof vi.fn>).mockResolvedValue(resource);
    await teleportRocksService.addMap("42", "regular", 100000000);
    expect(api.post).toHaveBeenCalledWith(
      "/api/characters/42/teleport-rock-maps",
      { data: { type: "teleport-rock-maps", attributes: { list: "regular", mapId: 100000000 } } },
    );
  });

  it("removeMap deletes the nested path", async () => {
    (api.delete as ReturnType<typeof vi.fn>).mockResolvedValue(resource);
    await teleportRocksService.removeMap("42", "vip", 200000000);
    expect(api.delete).toHaveBeenCalledWith("/api/characters/42/teleport-rock-maps/vip/200000000");
  });
});
```

- [ ] **Step 2: Run to verify failure**

Run: `cd services/atlas-ui && npm test -- teleport-rocks.service`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement the service**

```ts
import { api } from "@/lib/api/client";

const BASE_PATH = "/api/characters";

export type TeleportRockListType = "regular" | "vip";

export interface TeleportRockLists {
  regular: number[];
  vip: number[];
  regularCapacity: number;
  vipCapacity: number;
}

interface TeleportRockResource {
  id: string;
  type: "teleport-rock-maps";
  attributes: TeleportRockLists;
}

function flatten(r: TeleportRockResource): TeleportRockLists {
  return {
    regular: r.attributes.regular ?? [],
    vip: r.attributes.vip ?? [],
    regularCapacity: r.attributes.regularCapacity,
    vipCapacity: r.attributes.vipCapacity,
  };
}

export const teleportRocksService = {
  async getByCharacterId(characterId: string): Promise<TeleportRockLists> {
    const r = await api.getOne<TeleportRockResource>(`${BASE_PATH}/${characterId}/teleport-rock-maps`);
    return flatten(r);
  },
  async addMap(characterId: string, list: TeleportRockListType, mapId: number): Promise<TeleportRockLists> {
    const r = await api.post<TeleportRockResource>(
      `${BASE_PATH}/${characterId}/teleport-rock-maps`,
      { data: { type: "teleport-rock-maps", attributes: { list, mapId } } },
    );
    return flatten(r);
  },
  async removeMap(characterId: string, list: TeleportRockListType, mapId: number): Promise<TeleportRockLists> {
    const r = await api.delete<TeleportRockResource>(
      `${BASE_PATH}/${characterId}/teleport-rock-maps/${list}/${mapId}`,
    );
    return flatten(r);
  },
};
```

> If `api.post`/`api.delete` return the raw JSON:API envelope `{data:{...}}` rather than the unwrapped resource (confirm against `characterSkills.service.ts` / `bans.service.ts` — `getOne` unwraps, but `post`/`delete` may not), unwrap with `("data" in r ? r.data : r)` before `flatten`. Verify against `lib/api/client.ts` and match the existing services' handling.

- [ ] **Step 4: Implement the hooks**

```ts
import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from "@tanstack/react-query";
import { teleportRocksService, type TeleportRockListType, type TeleportRockLists } from "@/services/api/teleport-rocks.service";
import type { Tenant } from "@/types/models/tenant";

export const teleportRockKeys = {
  all: ["teleport-rocks"] as const,
  detail: (tenantId: string | undefined, characterId: string) =>
    [...teleportRockKeys.all, tenantId, characterId] as const,
};

export function useTeleportRockMaps(
  tenant: Tenant | null | undefined,
  characterId: string,
): UseQueryResult<TeleportRockLists, Error> {
  return useQuery({
    queryKey: teleportRockKeys.detail(tenant?.id, characterId),
    queryFn: () => teleportRocksService.getByCharacterId(characterId),
    enabled: !!tenant?.id && !!characterId,
    staleTime: 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

interface AddVars { characterId: string; list: TeleportRockListType; mapId: number; tenantId?: string }
interface RemoveVars extends AddVars {}

export function useAddTeleportRockMap(): UseMutationResult<TeleportRockLists, Error, AddVars> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: AddVars) => teleportRocksService.addMap(v.characterId, v.list, v.mapId),
    onSuccess: (data, v) => qc.setQueryData(teleportRockKeys.detail(v.tenantId, v.characterId), data),
  });
}

export function useRemoveTeleportRockMap(): UseMutationResult<TeleportRockLists, Error, RemoveVars> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: RemoveVars) => teleportRocksService.removeMap(v.characterId, v.list, v.mapId),
    onSuccess: (data, v) => qc.setQueryData(teleportRockKeys.detail(v.tenantId, v.characterId), data),
  });
}
```

- [ ] **Step 5: Run tests + typecheck**

Run: `cd services/atlas-ui && npm test -- teleport-rocks.service && npm run build`
Expected: PASS + clean build (the build type-checks `.test.ts` too).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/services/api/teleport-rocks.service.ts \
        services/atlas-ui/src/lib/hooks/api/useTeleportRocks.ts \
        services/atlas-ui/src/services/api/__tests__/teleport-rocks.service.test.ts
git commit -m "feat(task-124): atlas-ui teleport-rocks service + hooks"
```

---

### Task 4: atlas-ui — AddTeleportRockMapDialog (searchable picker)

**Files:**
- Create: `services/atlas-ui/src/components/features/characters/AddTeleportRockMapDialog.tsx`
- Test: `services/atlas-ui/src/components/features/characters/__tests__/AddTeleportRockMapDialog.test.tsx`

**Interfaces:**
- Consumes: `Dialog*` from `@/components/ui/dialog`, `Input`, `ScrollArea` from `@/components/ui/scroll-area`, `Button`; `useDebounce` from `@/lib/utils/debounce`; `useMapsByName` from `@/lib/hooks/api/useMaps`; `toast` from `sonner`; `useAddTeleportRockMap` (Task 3).
- Produces: `AddTeleportRockMapDialog({ characterId, list, tenantId, existingMapIds, open, onOpenChange }: { characterId: string; list: TeleportRockListType; tenantId?: string; existingMapIds: number[]; open: boolean; onOpenChange: (o: boolean) => void })`.

- [ ] **Step 1: Write the failing test**

```tsx
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AddTeleportRockMapDialog } from "@/components/features/characters/AddTeleportRockMapDialog";

vi.mock("@/context/tenant-context", () => ({ useTenant: () => ({ activeTenant: { id: "t" } }) }));
vi.mock("@/lib/hooks/api/useMaps", () => ({
  useMapsByName: () => ({ data: [{ id: "100000000", attributes: { name: "Henesys" } }], isLoading: false }),
}));
const addMap = vi.fn().mockResolvedValue({ regular: [100000000], vip: [], regularCapacity: 5, vipCapacity: 10 });
vi.mock("@/lib/hooks/api/useTeleportRocks", () => ({
  useAddTeleportRockMap: () => ({ mutateAsync: addMap, isPending: false }),
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

function renderDialog() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <AddTeleportRockMapDialog characterId="42" list="regular" existingMapIds={[]} open onOpenChange={vi.fn()} />
    </QueryClientProvider>,
  );
}

describe("AddTeleportRockMapDialog", () => {
  beforeEach(() => vi.clearAllMocks());

  it("adds the selected map", async () => {
    renderDialog();
    fireEvent.change(screen.getByPlaceholderText(/search maps/i), { target: { value: "hen" } });
    fireEvent.click(await screen.findByText("Henesys"));
    await waitFor(() =>
      expect(addMap).toHaveBeenCalledWith(expect.objectContaining({ characterId: "42", list: "regular", mapId: 100000000 })));
  });
});
```

- [ ] **Step 2: Run to verify failure**

Run: `cd services/atlas-ui && npm test -- AddTeleportRockMapDialog`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement the dialog**

Build from the `ChangeMapDialog` structure: a `Dialog` with a debounced search `Input` (placeholder "Search maps…"), results in a `ScrollArea`, each result a button. Filter out `existingMapIds`. On click call `mutateAsync({ characterId, list, mapId, tenantId })`; `toast.success` on resolve, `toast.error(err.message)` on reject, then `onOpenChange(false)`. Show `isLoading` state. Use `useMapsByName(useDebounce(query, 300))`. (Full component ~80 lines; follow `ChangeMapDialog.tsx` verbatim for the Dialog/Input scaffolding and the sonner + error-message-from-`error.message` handling.)

- [ ] **Step 4: Run to verify pass**

Run: `cd services/atlas-ui && npm test -- AddTeleportRockMapDialog`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/AddTeleportRockMapDialog.tsx \
        services/atlas-ui/src/components/features/characters/__tests__/AddTeleportRockMapDialog.test.tsx
git commit -m "feat(task-124): AddTeleportRockMapDialog searchable map picker"
```

---

### Task 5: atlas-ui — TeleportRockListCard

**Files:**
- Create: `services/atlas-ui/src/components/features/characters/TeleportRockListCard.tsx`
- Test: `services/atlas-ui/src/components/features/characters/__tests__/TeleportRockListCard.test.tsx`

**Interfaces:**
- Consumes: `Card`, `CardContent`, `CardHeader` from `@/components/ui/card`; `Badge`, `Button`, `ScrollArea`; `useItemData` from `@/lib/hooks/useItemData`; `useMap` from `@/lib/hooks/api/useMaps`; `useRemoveTeleportRockMap` (Task 3); `AddTeleportRockMapDialog` (Task 4); `toast` from `sonner`.
- Produces: `TeleportRockListCard({ characterId, list, maps, capacity, tenantId }: { characterId: string; list: TeleportRockListType; maps: number[]; capacity: number; tenantId?: string })`.

- [ ] **Step 1: Write the failing test**

```tsx
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TeleportRockListCard } from "@/components/features/characters/TeleportRockListCard";

vi.mock("@/context/tenant-context", () => ({ useTenant: () => ({ activeTenant: { id: "t" } }) }));
vi.mock("@/lib/hooks/useItemData", () => ({ useItemData: () => ({ iconUrl: "icon.png", name: "Teleport Rock" }) }));
vi.mock("@/lib/hooks/api/useMaps", () => ({ useMap: (id: string) => ({ data: { attributes: { name: `Map ${id}` } } }) }));
const removeMap = vi.fn().mockResolvedValue({ regular: [], vip: [], regularCapacity: 5, vipCapacity: 10 });
vi.mock("@/lib/hooks/api/useTeleportRocks", () => ({ useRemoveTeleportRockMap: () => ({ mutateAsync: removeMap, isPending: false }) }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

function renderCard(maps = [100000000]) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <TeleportRockListCard characterId="42" list="regular" maps={maps} capacity={5} />
    </QueryClientProvider>,
  );
}

describe("TeleportRockListCard", () => {
  beforeEach(() => vi.clearAllMocks());

  it("shows used-of-capacity", () => {
    renderCard();
    expect(screen.getByText("1 of 5")).toBeInTheDocument();
  });

  it("disables Add at capacity", () => {
    renderCard([1, 2, 3, 4, 5].map((n) => n * 100000000));
    expect(screen.getByRole("button", { name: /add/i })).toBeDisabled();
  });

  it("removes a map", async () => {
    renderCard();
    fireEvent.click(screen.getByRole("button", { name: /remove map 100000000/i }));
    await waitFor(() =>
      expect(removeMap).toHaveBeenCalledWith(expect.objectContaining({ characterId: "42", list: "regular", mapId: 100000000 })));
  });
});
```

- [ ] **Step 2: Run to verify failure**

Run: `cd services/atlas-ui && npm test -- TeleportRockListCard`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement the card**

`Card` with `CardHeader` = `<img src={iconUrl}>` (from `useItemData(list === "vip" ? 5041000 : 5040000)`) + title (`Regular Teleport Rocks` / `VIP Teleport Rocks`) + a `Badge` reading `{maps.length} of {capacity}`. `CardContent` = a `ScrollArea` mapping `maps` to a row: a `MapRow` subcomponent calling `useMap(String(mapId))` for the name (`Map {id}` fallback while loading) + a trash `Button` with `aria-label={\`Remove map ${mapId}\`}` that calls `mutateAsync({ characterId, list, mapId, tenantId })`, `toast.success`/`toast.error`. Footer `Add` `Button` (`aria-label="Add"`) disabled when `maps.length >= capacity`, opening `AddTeleportRockMapDialog` (local `open` state, pass `existingMapIds={maps}`). Since hooks can't be called in a `.map` callback, implement `MapRow` as its own component so `useMap` is called at the top level of each row.

- [ ] **Step 4: Run to verify pass**

Run: `cd services/atlas-ui && npm test -- TeleportRockListCard`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/TeleportRockListCard.tsx \
        services/atlas-ui/src/components/features/characters/__tests__/TeleportRockListCard.test.tsx
git commit -m "feat(task-124): TeleportRockListCard with capacity + delete"
```

---

### Task 6: atlas-ui — wire the cards into CharacterDetailPage

**Files:**
- Modify: `services/atlas-ui/src/pages/CharacterDetailPage.tsx`

**Interfaces:**
- Consumes: `useTeleportRockMaps` (Task 3), `TeleportRockListCard` (Task 5), `useTenant`.

- [ ] **Step 1: Add the query + render the two cards**

Near the other per-character queries in `CharacterDetailPage.tsx`:

```tsx
const teleportRocks = useTeleportRockMaps(activeTenant, id ?? "");
```

At the bottom of the page body (after the existing sections), gated on data:

```tsx
{teleportRocks.data && (
  <div className="grid gap-4 md:grid-cols-2">
    <TeleportRockListCard
      characterId={String(id)} list="regular"
      maps={teleportRocks.data.regular} capacity={teleportRocks.data.regularCapacity}
      tenantId={activeTenant?.id}
    />
    <TeleportRockListCard
      characterId={String(id)} list="vip"
      maps={teleportRocks.data.vip} capacity={teleportRocks.data.vipCapacity}
      tenantId={activeTenant?.id}
    />
  </div>
)}
```

Add the imports. Match the existing `activeTenant`/`id` names already in the file (from `useTenant()` and `useParams()`).

- [ ] **Step 2: Verify build + full UI test suite**

Run: `cd services/atlas-ui && npm run build && npm test`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-ui/src/pages/CharacterDetailPage.tsx
git commit -m "feat(task-124): render teleport-rock list cards on character detail"
```

---

### Task 7: Full verification

- [ ] **Step 1: Backend gates**

Run (from worktree root):
```bash
( cd services/atlas-character/atlas.com/character && go test -race ./... -count=1 && go vet ./... && go build ./... )
tools/lint.sh --check --go services/atlas-character
```
Expected: all clean.

- [ ] **Step 2: Frontend gates**

Run:
```bash
cd services/atlas-ui && npm run build && npm test
```
Expected: build + tests pass, no new lint errors.

- [ ] **Step 3: Code review**

Run `superpowers:requesting-code-review` (backend-guidelines-reviewer + frontend-guidelines-reviewer + plan-adherence-reviewer). Address findings before opening the PR.

---

## Self-Review Notes

- **Spec coverage:** processor typed errors + sync Add/Remove (Task 1) ✓; REST POST/DELETE + capacity + error mapping + README (Task 2) ✓; live refresh via shared emit (Task 1 `Add`/`Remove` flush `LIST_UPDATED`) ✓; service + hooks (Task 3) ✓; searchable picker (Task 4) ✓; card with icon/capacity/delete (Task 5) ✓; page wiring (Task 6) ✓; testing at every layer ✓.
- **Assumptions to verify during implementation (flagged inline):** whether `api.post`/`api.delete` unwrap the JSON:API envelope like `api.getOne` does (Task 3 — inline `("data" in r ? r.data : r)` fallback); no import cycle teleport_rock→character for the worldId lookup (Task 2 — inline fallback to a provider/injected function). The `Model` Builder API (`NewBuilder().SetCharacterId/SetRegular/SetVip/Build`) is confirmed against `builder.go`.
- **Deliberate plan tradeoff:** the two presentational components (Tasks 4–5) give exact behavior via their failing tests (placeholders, `aria-label`s, mutation-call shapes) plus a named verbatim template (`ChangeMapDialog.tsx`) rather than a full inline paste, since they are deterministic from the test + template. All non-trivial logic (processor refactor, REST handlers, service) has full inline code.
