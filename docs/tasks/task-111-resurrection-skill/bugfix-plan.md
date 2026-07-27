# Four-Bug Fix Implementation Plan (pr-869 playtest findings)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the four bugs found while playtesting the task-111 ephemeral env: (1) Characters-page refresh never refetches the per-row map/location cache, (2) GM demotion via PATCH is a silent no-op, (3) observer-spawn packets carry foothold 0 so dead/idle characters render at the wrong position for re-entering observers, (4) atlas-parties' Redis character registry never refreshes GM status (not even on relog).

**Architecture:** Four independent fixes on the `task-111-resurrection-skill` branch (investigation and fix stay on one worktree — no forks). Fix 2 gives the GM field presence semantics (`*int`) end-to-end in atlas-character's PATCH path. Fix 3 threads the already-on-the-wire `fh` value through atlas-character's temporal registry and REST projection into the CharacterSpawn packet. Fix 4 makes atlas-parties refresh GM/level/job on login and consume the (now actually emittable) `GM_CHANGED` event. Fixes 2+4 together restore a working demote flow: UI PATCH → DB update + `GM_CHANGED` → parties registry refresh.

**Tech Stack:** Go (atlas-character, atlas-channel, atlas-parties, libs/atlas-packet), TypeScript/React + TanStack React Query 5 + Vitest (atlas-ui).

## Global Constraints

- Work in the worktree `.worktrees/task-111-resurrection-skill/` on branch `task-111-resurrection-skill`. Verify `git branch --show-current` prints `task-111-resurrection-skill` before the first commit and after every commit.
- Root-cause investigation for all four bugs is already done and pinned with file:line evidence — do NOT re-investigate; implement exactly what each task states.
- Per repo CLAUDE.md: before calling the branch done — `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake atlas-<svc>` for every touched Go service; `tools/redis-key-guard.sh` clean from repo root.
- No `// TODO`, stubs, or deferred work in commits.
- Wire compatibility: all Kafka/REST JSON changes in this plan are additive fields; absent fields must decode to today's zero-value behavior.
- v83 client behavior cited in this plan (foothold-0 physics drop, OnPartyResult case 32) is already IDA-verified — do not re-derive it.

---

### Task 1: atlas-ui — Characters-page refresh must invalidate the per-row location cache

The Map column is rendered by `CharacterMapCell` from a separate `["character-location", tenantId, characterId]` query (`src/lib/hooks/api/useCharacterLocation.ts:22-27`, staleTime 60s). `useGridRefresh` only refetches the three page-level queries, so mounted location cells never refetch. Fix: let `useGridRefresh` accept an extra async action and have CharactersPage invalidate `characterLocationKeys.all` (invalidation refetches active/mounted queries by default).

**Files:**
- Modify: `services/atlas-ui/src/lib/hooks/useGridRefresh.ts`
- Modify: `services/atlas-ui/src/pages/CharactersPage.tsx`
- Test: `services/atlas-ui/src/lib/hooks/__tests__/useGridRefresh.test.ts`

**Interfaces:**
- Produces: `useGridRefresh(queries, options?: { successMessage?: string; alsoRefresh?: () => Promise<unknown> })` — `alsoRefresh` runs concurrently with the refetches on every `onRefresh()`.

- [ ] **Step 1: Write the failing test** — append to `useGridRefresh.test.ts` inside the existing `describe`:

```ts
  it("runs alsoRefresh alongside the refetches", async () => {
    const q1 = makeQuery();
    const alsoRefresh = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() => useGridRefresh([q1], { alsoRefresh }));

    await act(async () => {
      await result.current.onRefresh();
    });

    expect(alsoRefresh).toHaveBeenCalledTimes(1);
    expect(q1.refetch).toHaveBeenCalledTimes(1);
    expect(toast.success).toHaveBeenCalledTimes(1);
  });
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npx vitest run src/lib/hooks/__tests__/useGridRefresh.test.ts`
Expected: FAIL — new test fails (`alsoRefresh` never called; option not part of the signature yet).

- [ ] **Step 3: Implement** — in `useGridRefresh.ts`, change the options type and `onRefresh`:

```ts
export function useGridRefresh(
  queries: RefreshableQuery[],
  options?: { successMessage?: string; alsoRefresh?: () => Promise<unknown> },
): UseGridRefreshResult {
  const isRefreshing = queries.some((q) => q.isFetching);

  const onRefresh = async (): Promise<void> => {
    const [results] = await Promise.all([
      Promise.all(queries.map((q) => q.refetch())),
      options?.alsoRefresh?.(),
    ]);
    const failed = results.find((r) => r.isError);
    if (failed) {
      toast.error(failed.error, { context: { action: "refresh" } });
      return;
    }
    toast.success(options?.successMessage ?? "Data refreshed");
  };

  return { isRefreshing, onRefresh };
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npx vitest run src/lib/hooks/__tests__/useGridRefresh.test.ts`
Expected: PASS (all 6 tests).

- [ ] **Step 5: Wire CharactersPage** — in `CharactersPage.tsx`, add the two imports and pass `alsoRefresh`:

```tsx
import { useQueryClient } from "@tanstack/react-query";
import { characterLocationKeys } from "@/lib/hooks/api/useCharacterLocation";
```

```tsx
  const queryClient = useQueryClient();
  const { isRefreshing, onRefresh } = useGridRefresh(
    [charactersQuery, accountsQuery, tenantConfigQuery],
    {
      alsoRefresh: () =>
        queryClient.invalidateQueries({ queryKey: characterLocationKeys.all }),
    },
  );
```

(The existing `onRefresh` is already threaded to both `getColumns` and `DataTableWrapper`; no other page change.)

- [ ] **Step 6: Full UI verification**

Run: `cd services/atlas-ui && npm run test && npm run lint && npm run build`
Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/lib/hooks/useGridRefresh.ts services/atlas-ui/src/pages/CharactersPage.tsx services/atlas-ui/src/lib/hooks/__tests__/useGridRefresh.test.ts
git commit -m "fix(ui): characters refresh invalidates per-row location cache"
```

---

### Task 2: atlas-character — presence-aware `Gm` so PATCH demote works

`RestModel.Gm` is a value-typed `int` (`character/rest.go:44`), so an explicit `gm:0` is indistinguishable from an absent field; the processor guard (`character/processor.go:1736`) resolves that by making demotion unreachable. Fix: `Gm *int` + nil-guard.

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/rest.go`
- Modify: `services/atlas-character/atlas.com/character/character/processor.go:1732-1750`
- Test: `services/atlas-character/atlas.com/character/character/patch_integration_test.go`

**Interfaces:**
- Produces: `RestModel.Gm *int` (`json:"gm"`). GET responses still always emit a number (Transform always sets it). PATCH treats nil = "no change", non-nil = explicit set (0 = demote). Task 6's parties consumer relies on the `GM_CHANGED` event this path emits with `newGm:false`.

- [ ] **Step 1: Write the failing tests** — append to `patch_integration_test.go` (mirror the style of `TestGmChangedEventEmission` at :1317; reuse its imports). Also add the pointer helper:

```go
func gmPtr(v int) *int { return &v }

func TestGmDemotionEventEmission(t *testing.T) {
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("DemoteMe").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetGm(1). // starts as GM
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter, 0)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	buf := message.NewBuffer()
	updatePayload := character.RestModel{
		Id: createdCharacter.Id(),
		Gm: gmPtr(0), // explicit demote
	}

	transactionId := uuid.New()
	err = processor.Update(buf)(transactionId, createdCharacter.Id(), updatePayload)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}

	updatedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get updated character: %v", err)
	}
	if updatedCharacter.GM() != 0 {
		t.Errorf("Expected GM status 0 after demotion, got %d", updatedCharacter.GM())
	}

	statusMessages, exists := buf.GetAll()[character2.EnvEventTopicCharacterStatus]
	if !exists || len(statusMessages) != 1 {
		t.Fatalf("Expected exactly 1 GM_CHANGED event, got %d", len(statusMessages))
	}
	var eventValue character2.StatusEvent[character2.StatusEventGmChangedBody]
	if err := json.Unmarshal(statusMessages[0].Value, &eventValue); err != nil {
		t.Fatalf("Failed to unmarshal event value: %v", err)
	}
	if eventValue.Type != character2.StatusEventTypeGmChanged {
		t.Errorf("Expected event type '%s', got '%s'", character2.StatusEventTypeGmChanged, eventValue.Type)
	}
	if eventValue.Body.OldGm != true || eventValue.Body.NewGm != false {
		t.Errorf("Expected oldGm=true newGm=false, got oldGm=%t newGm=%t", eventValue.Body.OldGm, eventValue.Body.NewGm)
	}
}

func TestGmAbsentMeansNoChange(t *testing.T) {
	db := testDatabase(t)
	tenantModel := testTenant()
	tctx := tenant.WithContext(context.Background(), tenantModel)
	logger := testLogger()

	originalCharacter := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(world.Id(0)).
		SetName("StayGm").
		SetLevel(1).
		SetStrength(4).
		SetDexterity(4).
		SetIntelligence(4).
		SetLuck(4).
		SetMaxHp(50).SetHp(50).
		SetMaxMp(50).SetMp(50).
		SetJobId(job.Id(0)).
		SetGender(0).
		SetHair(30000).
		SetFace(20000).
		SetSkinColor(0).
		SetGm(2).
		Build()

	processor := character.NewProcessor(logger, tctx, db)
	createdCharacter, err := processor.Create(message.NewBuffer())(uuid.New(), originalCharacter, 0)
	if err != nil {
		t.Fatalf("Failed to create character for testing: %v", err)
	}

	buf := message.NewBuffer()
	// Gm nil = field absent from the PATCH: must not change GM, must not emit
	// GM_CHANGED. With no other field set, Update takes the empty-changes early
	// return and succeeds as a no-op.
	updatePayload := character.RestModel{
		Id: createdCharacter.Id(),
	}
	err = processor.Update(buf)(uuid.New(), createdCharacter.Id(), updatePayload)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}

	updatedCharacter, err := processor.GetById()(createdCharacter.Id())
	if err != nil {
		t.Fatalf("Failed to get updated character: %v", err)
	}
	if updatedCharacter.GM() != 2 {
		t.Errorf("Expected GM status to remain 2, got %d", updatedCharacter.GM())
	}
	for _, msg := range buf.GetAll()[character2.EnvEventTopicCharacterStatus] {
		var probe character2.StatusEvent[character2.StatusEventGmChangedBody]
		if err := json.Unmarshal(msg.Value, &probe); err == nil && probe.Type == character2.StatusEventTypeGmChanged {
			t.Error("GM_CHANGED must not be emitted when gm is absent from the payload")
		}
	}
}
```

Note: if `Hair` 30030 fails `isValidHair` in the test DB, use any hair id the existing tests use for valid-hair updates (grep `Hair:` in this test file and reuse).

- [ ] **Step 2: Run tests to verify they fail to compile/pass**

Run: `cd services/atlas-character/atlas.com/character && go test -run 'TestGm' ./character/...`
Expected: compile FAIL (`gmPtr(0)` is `*int`, field is `int`) — that's the point.

- [ ] **Step 3: Change `RestModel.Gm` to a pointer** — `character/rest.go:44`:

```go
	// Gm is a pointer so PATCH can distinguish an explicit gm:0 (demote) from
	// an absent field (no change). GET responses always set it.
	Gm                 *int    `json:"gm"`
```

In `transformWithTemporal` (rest.go:82-116), replace `Gm: m.GM(),` — pointers can't reference the getter inline, so hoist:

```go
func transformWithTemporal(m Model, td temporalData) RestModel {
	gm := m.GM()
	rm := RestModel{
		// ... all existing fields unchanged ...
		Gm:         &gm,
		// ...
	}
	return rm
}
```

In `Extract` (rest.go, the builder chain around :149), replace `SetGm(m.Gm).` with `SetGm(derefOrZero(m.Gm)).` and add:

```go
func derefOrZero(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
```

- [ ] **Step 4: Fix the processor guard** — replace `character/processor.go:1732-1750` with:

```go
			// GM validation and update
			// Gm is a pointer: nil = field absent (no change requested);
			// non-nil = explicit set, including 0 (demotion).
			if input.Gm != nil && *input.Gm != c.GM() {
				newGmVal := *input.Gm
				if !p.isValidGm(newGmVal) {
					return errors.New("invalid GM value")
				}
				changes = append(changes, fieldChange{
					updateFunc:  SetGm(newGmVal),
					shouldApply: true,
					eventFunc: func() error {
						oldGm := c.GM() != 0
						newGm := newGmVal != 0
						return mb.Put(character2.EnvEventTopicCharacterStatus, gmChangedEventProvider(transactionId, characterId, c.WorldId(), oldGm, newGm))
					},
				})
			}
```

- [ ] **Step 5: Fix remaining compile sites** — `grep -rn 'Gm:' services/atlas-character/atlas.com/character --include='*.go'` and update every `character.RestModel{... Gm: N ...}` literal to `Gm: gmPtr(N)` (tests) or `&v` (non-test, if any). `TestGmChangedEventEmission` at patch_integration_test.go:1356 becomes `Gm: gmPtr(1)`. The Kafka create path (`kafka/consumer/character/consumer.go:342 SetGm(c.Body.Gm)`) uses a different struct — untouched.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-character/atlas.com/character && go test -race -run 'TestGm' ./character/... && go test -race ./... && go vet ./...`
Expected: PASS, vet clean. (Consumers of this REST API — atlas-channel/atlas-parties `RestModel.Gm int` decoders — are unaffected: GET always emits a number.)

- [ ] **Step 7: Commit**

```bash
git add services/atlas-character/atlas.com/character/character/rest.go services/atlas-character/atlas.com/character/character/processor.go services/atlas-character/atlas.com/character/character/patch_integration_test.go
git commit -m "fix(character): presence-aware gm PATCH so demotion works"
```

---

### Task 3: atlas-character — persist foothold in the temporal registry and REST projection

atlas-channel already emits `fh` in the movement command JSON (`services/atlas-channel/.../kafka/message/movement/kafka.go:25` — `Fh int16 \`json:"fh"\``); atlas-character's `MovementCommand` silently drops it. Thread it through to the REST x/y/stance decoration.

**Files:**
- Modify: `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:393-404`
- Modify: `services/atlas-character/atlas.com/character/character/temporal_data.go`
- Modify: `services/atlas-character/atlas.com/character/character/processor.go:728-731` (+ `Move` in the `Processor` interface, + any mock)
- Modify: `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go:364-371`
- Modify: `services/atlas-character/atlas.com/character/character/rest.go` (RestModel + transformWithTemporal)
- Test: `services/atlas-character/atlas.com/character/character/temporal_data_test.go` (new)

**Interfaces:**
- Consumes: inbound movement JSON already contains `"fh"` (int16) — no channel-side producer change needed.
- Produces: `RestModel.Fh int16` (`json:"fh"`) on character GET responses; `temporalData.Fh() int16`; `Move(characterId uint32, x int16, y int16, fh int16, stance byte) error`. Task 4 consumes the REST `fh`.

- [ ] **Step 1: Write the failing test** — create `temporal_data_test.go` in package `character`:

```go
package character

import (
	"encoding/json"
	"testing"
)

func TestTemporalDataJSONRoundTripIncludesFh(t *testing.T) {
	in := temporalData{x: -12, y: 250, fh: 37, stance: 4}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out temporalData
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.X() != -12 || out.Y() != 250 || out.Fh() != 37 || out.Stance() != 4 {
		t.Errorf("round trip mismatch: got x=%d y=%d fh=%d stance=%d", out.X(), out.Y(), out.Fh(), out.Stance())
	}
}

func TestTemporalDataUnmarshalWithoutFhDefaultsZero(t *testing.T) {
	// Entries written before this change have no "fh" key — must decode as 0.
	var out temporalData
	if err := json.Unmarshal([]byte(`{"x":5,"y":6,"stance":2}`), &out); err != nil {
		t.Fatalf("unmarshal legacy payload: %v", err)
	}
	if out.Fh() != 0 || out.X() != 5 {
		t.Errorf("legacy decode mismatch: fh=%d x=%d", out.Fh(), out.X())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-character/atlas.com/character && go test -run TestTemporalData ./character/...`
Expected: compile FAIL (`fh` field / `Fh()` undefined).

- [ ] **Step 3: Implement temporal data** — `temporal_data.go`:

```go
type temporalData struct {
	x      int16
	y      int16
	fh     int16
	stance byte
}

func (d temporalData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		X      int16 `json:"x"`
		Y      int16 `json:"y"`
		Fh     int16 `json:"fh"`
		Stance byte  `json:"stance"`
	}{X: d.x, Y: d.y, Fh: d.fh, Stance: d.stance})
}

func (d *temporalData) UnmarshalJSON(data []byte) error {
	var raw struct {
		X      int16 `json:"x"`
		Y      int16 `json:"y"`
		Fh     int16 `json:"fh"`
		Stance byte  `json:"stance"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.x = raw.X
	d.y = raw.Y
	d.fh = raw.Fh
	d.stance = raw.Stance
	return nil
}

func (d *temporalData) Fh() int16 {
	return d.fh
}
```

Update the three mutators (preserve `fh` where not provided):

```go
func (r *temporalRegistry) UpdatePosition(ctx context.Context, t tenant.Model, characterId uint32, x int16, y int16) {
	existing, _ := r.reg.Get(ctx, t, characterId)
	_ = r.reg.Put(ctx, t, characterId, temporalData{x: x, y: y, fh: existing.fh, stance: existing.stance})
}

func (r *temporalRegistry) Update(ctx context.Context, t tenant.Model, characterId uint32, x int16, y int16, fh int16, stance byte) {
	_ = r.reg.Put(ctx, t, characterId, temporalData{x: x, y: y, fh: fh, stance: stance})
}

func (r *temporalRegistry) UpdateStance(ctx context.Context, t tenant.Model, characterId uint32, stance byte) {
	existing, _ := r.reg.Get(ctx, t, characterId)
	_ = r.reg.Put(ctx, t, characterId, temporalData{x: existing.x, y: existing.y, fh: existing.fh, stance: stance})
}
```

- [ ] **Step 4: Thread fh through command → Move → REST**

`kafka/message/character/kafka.go` `MovementCommand` — add after `Y`:

```go
	Fh            int16      `json:"fh"`
```

`character/processor.go:728`:

```go
func (p *ProcessorImpl) Move(characterId uint32, x int16, y int16, fh int16, stance byte) error {
	GetTemporalRegistry().Update(p.ctx, tenant.MustFromContext(p.ctx), characterId, x, y, fh, stance)
	return nil
}
```

Update the `Move` signature in the `Processor` interface declaration in the same file, and run `grep -rn '\.Move(' services/atlas-character/atlas.com/character --include='*.go'` — update every caller/mock (known: `kafka/consumer/character/consumer.go:366`):

```go
		err := character.NewProcessor(l, ctx, db).Move(uint32(c.ObjectId), c.X, c.Y, c.Fh, c.Stance)
```

`character/rest.go` — add to `RestModel` after `Y`:

```go
	Fh                 int16    `json:"fh"`
```

and in `transformWithTemporal`, after `Y: td.Y(),`:

```go
		Fh:         td.Fh(),
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-character/atlas.com/character && go test -race ./... && go vet ./...`
Expected: PASS, vet clean. (Legacy Redis entries without `fh` decode to 0 — today's behavior; verified by `TestTemporalDataUnmarshalWithoutFhDefaultsZero`.)

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character/atlas.com/character
git commit -m "fix(character): persist movement foothold in temporal registry and REST projection"
```

---

### Task 4: atlas-packet + atlas-channel — encode the real foothold in CharacterSpawn

`CharacterSpawn.Encode` hardcodes `w.WriteShort(0) // fh` (`libs/atlas-packet/character/clientbound/spawn.go:101`). IDA-verified (v83 `CUserRemote::Init` → `CWvsPhysicalSpace2D::GetFoothold`): fh=0 yields no foothold anchor, so client physics drops the remote character — for a character at exactly surface y (any standing/dead character), through the platform. Encode the real fh for the already-in-map enumeration path; keep 0 for the `enteringField` jump-in spawn (that path intentionally spawns airborne at y−42 and must not change).

**Files:**
- Modify: `libs/atlas-packet/character/clientbound/spawn.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/rest.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/model.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/character_spawn.go:60-63`
- Test: `libs/atlas-packet/character/clientbound/spawn_test.go`, `libs/atlas-packet/character/clientbound/version_bounds_test.go` (signature updates + new assertion)

**Interfaces:**
- Consumes: `RestModel.Fh` from Task 3 (absent → 0, same as today).
- Produces: `NewCharacterSpawn(characterId uint32, level byte, name string, guild GuildEmblem, cts *model.CharacterTemporaryStat, jobId uint16, avatar model.Avatar, pets []SpawnPet, enteringField bool, x int16, y int16, stance byte, fh int16) CharacterSpawn` — one new trailing `fh int16` parameter; `(CharacterSpawn) Fh() int16` getter; channel `character.Model.Fh() int16`.

- [ ] **Step 1: Update the packet struct and codec** — `spawn.go`:

Add `fh int16` to the `CharacterSpawn` struct after `stance`; add the trailing parameter to `NewCharacterSpawn` and set `fh: fh` in the literal. In `Encode`, replace line 101 (`w.WriteShort(0) // fh`) with:

```go
		if m.enteringField {
			// jump-in spawn is intentionally airborne (y-42, stance 6): no anchor
			w.WriteInt16(0) // fh
		} else {
			w.WriteInt16(m.fh)
		}
```

In `Decode`, replace `_ = r.ReadUint16() // fh` (line 209) with:

```go
		m.fh = r.ReadInt16()
```

Add the getter next to the others:

```go
func (m CharacterSpawn) Fh() int16 { return m.fh }
```

- [ ] **Step 2: Fix lib tests and add coverage** — `grep -n 'NewCharacterSpawn(' libs/atlas-packet/character/clientbound/*_test.go` and append `, 0` as the final argument to every existing call (byte fixtures are unchanged: they encoded fh=0 before; the JMS golden's 238-byte length is unaffected). Also extend `TestCharacterSpawnRoundTrip` (spawn_test.go:89): change its constructor call to `NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 312, avatar, nil, false, 100, 200, 3, 37)` and append after the stance assertion:

```go
			if output.Fh() != 37 {
				t.Errorf("fh: got %v, want %v", output.Fh(), 37)
			}
```

Then add a new test pinning the entering-field branch (pt.RoundTrip = encode → decode → assert-fully-consumed; no re-encode, so entering-field's y−42/stance-6 rewrite doesn't matter here):

```go
func TestCharacterSpawnEnteringFieldEncodesFhZero(t *testing.T) {
	// entering-field spawns are intentionally airborne (y-42, stance 6):
	// the wire fh must stay 0 even when the model carries a real foothold.
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			avatar := testSpawnAvatar()
			cts := model.NewCharacterTemporaryStat()
			guild := GuildEmblem{Name: "TestGuild"}
			input := NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 312, avatar, nil, true, 100, 200, 6, 37)
			output := CharacterSpawn{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Fh() != 0 {
				t.Errorf("entering-field fh on the wire: got %v, want 0", output.Fh())
			}
		})
	}
}
```

- [ ] **Step 3: Run lib tests**

Run: `cd libs/atlas-packet && go test -race ./character/... && go vet ./character/...`
Expected: PASS. Any `packet-audit:verify` fixture in these files must pass unmodified except for the added `, 0` argument — if a fixture's expected bytes need changing, STOP: that means behavior changed for an existing case, which this task must not do.

- [ ] **Step 4: Expose Fh on the channel character model** — `services/atlas-channel/atlas.com/channel/character/rest.go`: add to `RestModel` after `Y`:

```go
	Fh                 int16    `json:"fh"`
```

and in `Extract`, after `y: m.Y,`:

```go
		fh:                 m.Fh,
```

`model.go`: add field `fh int16` next to `x`/`y` (line ~50) and getter next to `Stance()`:

```go
func (m Model) Fh() int16 {
	return m.fh
}
```

Run `grep -n 'stance:' services/atlas-channel/atlas.com/channel/character/*.go` — if any other constructor/builder copies `stance`, copy `fh` alongside it the same way.

- [ ] **Step 5: Pass it in the spawn writer** — `socket/writer/character_spawn.go:60-63`:

```go
			return charpkt.NewCharacterSpawn(
				c.Id(), c.Level(), c.Name(), ge, cts, uint16(c.JobId()), ava,
				pets, enteringField, c.X(), c.Y(), c.Stance(), c.Fh(),
			).Encode(l, ctx)(options)
```

(Verified: this writer and the lib tests are the only `NewCharacterSpawn` callers in the repo.)

- [ ] **Step 6: Channel module verification**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./... && go vet ./...`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/character/clientbound services/atlas-channel/atlas.com/channel/character services/atlas-channel/atlas.com/channel/socket/writer/character_spawn.go
git commit -m "fix(channel): encode real foothold in CharacterSpawn for in-map enumeration"
```

---

### Task 5: atlas-parties — refresh GM on login and consume GM_CHANGED

The `party-character` Redis registry writes `gm` only at entry creation (`character/processor.go:72,:275`); `Login` skips the re-fetch when the entry exists, so a GM change (via Task 2's now-working PATCH, or a direct DB edit) is never seen — relog and pod restart included. Fix both halves: refresh on login, and react to `GM_CHANGED` live.

**Files:**
- Modify: `services/atlas-parties/atlas.com/parties/character/model.go`
- Modify: `services/atlas-parties/atlas.com/parties/character/processor.go`
- Modify: `services/atlas-parties/atlas.com/parties/kafka/consumer/character/kafka.go`
- Modify: `services/atlas-parties/atlas.com/parties/kafka/consumer/character/consumer.go`
- Test: `services/atlas-parties/atlas.com/parties/character/model_test.go` (new)

**Interfaces:**
- Consumes: `GM_CHANGED` StatusEvent from atlas-character (`Type: "GM_CHANGED"`, body `{oldGm bool, newGm bool}` — see `services/atlas-character/.../kafka/message/character/kafka.go:233,377-380`); fresh gm via existing `GetForeignCharacterInfo` (REST → Postgres, no cache).
- Produces: `Model.ChangeGm(gm int) Model`; `Processor.GmChange(characterId uint32) error`.

- [ ] **Step 1: Write the failing model test** — create `character/model_test.go` in package `character`:

```go
package character

import "testing"

func TestChangeGmPreservesOtherFields(t *testing.T) {
	m := Model{id: 42, name: "Atlas", level: 200, partyId: 7, online: true, gm: 1}
	out := m.ChangeGm(0)
	if out.GM() != 0 {
		t.Errorf("expected gm 0, got %d", out.GM())
	}
	if out.Id() != 42 || out.Name() != "Atlas" || out.Level() != 200 || out.PartyId() != 7 || !out.Online() {
		t.Error("ChangeGm must not mutate unrelated fields")
	}
}
```

(If `Model`'s unexported literal construction or any getter name differs, mirror exactly how `registry_test.go` in the same package builds models — same-package tests may use unexported fields.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-parties/atlas.com/parties && go test -run TestChangeGm ./character/...`
Expected: compile FAIL (`ChangeGm` undefined).

- [ ] **Step 3: Add `Model.ChangeGm`** — `character/model.go`, next to `ChangeJob` (line ~122), same copy-all style:

```go
func (m Model) ChangeGm(gm int) Model {
	return Model{
		tenantId: m.tenantId,
		id:       m.id,
		name:     m.name,
		level:    m.level,
		jobId:    m.jobId,
		field:    m.field,
		partyId:  m.partyId,
		online:   m.online,
		gm:       gm,
	}
}
```

(Match the exact field list of the neighboring Change* funcs — if `Model` has fields beyond these, copy them all.)

- [ ] **Step 4: Refresh on login** — `character/processor.go` `Login` (:62-89): when the registry entry already exists, re-fetch and fold the fresh gm/level/job into the same `Update` call:

```go
func (p *ProcessorImpl) Login(mb *message.Buffer) func(f field.Model, characterId uint32) error {
	return func(f field.Model, characterId uint32) error {
		c, err := p.GetById(characterId)
		refreshers := make([]func(Model) Model, 0, 3)
		if err != nil {
			p.l.Debugf("Adding character [%d] from world [%d] to registry.", characterId, f.WorldId())
			fm, err := p.GetForeignCharacterInfo(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to retrieve needed character information from foreign service.")
				return err
			}
			c = GetRegistry().Create(p.ctx, f, characterId, fm.Name(), fm.Level(), fm.JobId(), fm.GM())
		} else {
			// The registry entry survives relogs (Redis). GM/level/job can have
			// changed while offline (or via admin edit) with no event this
			// service saw — refresh them from the authoritative service.
			fm, ferr := p.GetForeignCharacterInfo(characterId)
			if ferr != nil {
				p.l.WithError(ferr).Warnf("Unable to refresh character [%d] info on login; using cached registry values.", characterId)
			} else {
				refreshers = append(refreshers,
					func(m Model) Model { return m.ChangeGm(fm.GM()) },
					func(m Model) Model { return m.ChangeLevel(fm.Level()) },
					func(m Model) Model { return m.ChangeJob(fm.JobId()) },
				)
			}
		}

		p.l.Debugf("Setting character [%d] to online in registry.", characterId)
		updaters := append(refreshers, Model.Login, func(m Model) Model { return Model.ChangeChannel(m, f.ChannelId()) })
		c = GetRegistry().Update(p.ctx, c.Id(), updaters...)

		if c.PartyId() != 0 {
			err = mb.Put(EnvEventMemberStatusTopic, loginEventProvider(c.PartyId(), c.WorldId(), characterId))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to announce the party [%d] member [%d] logged in.", c.PartyId(), c.Id())
				return err
			}
		}

		return nil
	}
}
```

- [ ] **Step 5: Add `GmChange` to the processor** — add `GmChange(characterId uint32) error` to the `Processor` interface (processor.go:20-37) and implement (re-fetch rather than trusting the event's bool — the registry stores an int GM level):

```go
func (p *ProcessorImpl) GmChange(characterId uint32) error {
	c, err := p.GetById(characterId)
	if err != nil {
		// Not in the registry: nothing to refresh; next login populates fresh.
		return nil
	}
	fm, err := p.GetForeignCharacterInfo(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to refresh GM status for character [%d].", characterId)
		return err
	}
	p.l.Debugf("Setting character [%d] GM status to [%d] in registry.", characterId, fm.GM())
	_ = GetRegistry().Update(p.ctx, c.Id(), func(m Model) Model { return m.ChangeGm(fm.GM()) })
	return nil
}
```

- [ ] **Step 6: Consume GM_CHANGED** — `kafka/consumer/character/kafka.go`: add to the status-event type const block (:122-133):

```go
	StatusEventTypeGmChanged         = "GM_CHANGED"
```

and next to the other body structs:

```go
type GmChangedStatusEventBody struct {
	OldGm bool `json:"oldGm"`
	NewGm bool `json:"newGm"`
}
```

`kafka/consumer/character/consumer.go`: register in `InitHandlers` (after the job-changed registration at :48-50):

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventGmChanged))); err != nil {
			return err
		}
```

and add the handler:

```go
func handleStatusEventGmChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[GmChangedStatusEventBody]) {
	if e.Type != StatusEventTypeGmChanged {
		return
	}

	l.WithField("characterId", e.CharacterId).
		WithField("worldId", e.WorldId).
		WithField("transactionId", e.TransactionId).
		WithField("oldGm", e.Body.OldGm).
		WithField("newGm", e.Body.NewGm).
		Debugf("Processing GM change event for character [%d].", e.CharacterId)

	err := character.NewProcessor(l, ctx).GmChange(e.CharacterId)
	if err != nil {
		l.WithError(err).
			WithField("characterId", e.CharacterId).
			WithField("transactionId", e.TransactionId).
			Errorf("Unable to process GM change for character [%d].", e.CharacterId)
		return
	}

	l.WithField("characterId", e.CharacterId).
		WithField("transactionId", e.TransactionId).
		Debugf("Successfully processed GM change for character [%d].", e.CharacterId)
}
```

- [ ] **Step 7: Module verification**

Run: `cd services/atlas-parties/atlas.com/parties && go build ./... && go test -race ./... && go vet ./...`
Expected: clean. If a character-processor mock exists (`grep -rn 'GmChange\|Processor interface' services/atlas-parties --include='*.go'` → check `character/mock` or similar), add the new method to it.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-parties/atlas.com/parties
git commit -m "fix(parties): refresh gm on login and consume GM_CHANGED"
```

---

### Task 6: Full verification sweep + deploy verification

- [ ] **Step 1: Workspace-wide gates (repo CLAUDE.md §Build & Verification)** — from the worktree root:

Run, for each of `services/atlas-character/atlas.com/character`, `services/atlas-channel/atlas.com/channel`, `services/atlas-parties/atlas.com/parties`, `libs/atlas-packet`:
`go test -race ./... && go vet ./... && go build ./...`
Expected: all clean.

Run: `tools/redis-key-guard.sh` (from the worktree root, no global GOWORK prefix)
Expected: clean.

- [ ] **Step 2: Docker bake every touched Go service**

Run: `docker buildx bake atlas-character atlas-channel atlas-parties`
Expected: all three images build. (atlas-ui has no Go bake target; `npm run build` in Task 1 covered it.)

- [ ] **Step 3: Push and let pr-869 redeploy**

```bash
git push origin task-111-resurrection-skill
```

The PR has the `deploy-env` label, so the ephemeral env rebuilds automatically.

- [ ] **Step 4: In-env acceptance checks (manual, in the pr-869 env)**

1. Characters page: move a character in-game, click refresh → Map column updates without a page reload.
2. UI demote: set Atlas's GM to 0 via the character dialog → character REST returns gm 0, and (without deleting any Redis key) Atlas can create/invite a party after the GM_CHANGED event lands. Re-promote afterwards if desired.
3. Dead-position: die on a platform with character A, have character B leave and re-enter the map → B sees A's body on the platform, not dropped to the ground below.
4. Regression: normal map entry (portal jump-in animation) still looks right — the enteringField spawn path was intentionally left at fh 0.

- [ ] **Step 5: Code review before PR** — per repo rules, run `superpowers:requesting-code-review` over these commits before the branch's PR is finalized (the task-111 PR #869 already exists; these fixes ride on it).
