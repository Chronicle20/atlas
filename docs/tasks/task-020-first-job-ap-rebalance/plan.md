# First-Job AP Rebalance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace unsatisfiable class-minimum stat gates on first-job advancement with a server-side `rebalance_ap` operation that zeros the four primary stats, raises the target stat(s) to the class floor, and returns the reclaimed AP as unallocated, for all five Explorer NPC scripts and all five Cygnus quest scripts.

**Architecture:** A new saga action `RebalanceAP` flows exactly like the existing `ChangeJob` / `ResetStats` actions: NPC/quest operation dispatcher → saga-orchestrator step handler → Kafka command (`REBALANCE_AP` on `COMMAND_TOPIC_CHARACTER`) → atlas-character consumer → new processor method `RebalanceAPAndEmit` that performs O(1) arithmetic in a DB transaction and emits a single multi-stat `STAT_CHANGED` event. Arithmetic is isolated in a pure helper `computeRebalance` for unit testing. The operation takes an array of `{stat, floor}` targets to support Thunder Breaker's double-gate; single-target is a one-element array.

**Tech Stack:** Go 1.21+, GORM, Kafka (segmentio), logrus, `libs/atlas-saga` shared library, JSON conversation scripts. Tests use `miniredis` and in-memory SQLite already established in the two services.

---

## Task 1: Add `RebalanceAP` action & payload to `libs/atlas-saga`

**Files:**
- Modify: `libs/atlas-saga/model.go`
- Modify: `libs/atlas-saga/payloads.go`
- Modify: `libs/atlas-saga/unmarshal.go`
- Test: `libs/atlas-saga/unmarshal_test.go` (create if missing; otherwise add a test function)

- [ ] **Step 1: Write the failing test**

Create or append to `libs/atlas-saga/unmarshal_test.go`:

```go
package saga

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestUnmarshalRebalanceAPStep(t *testing.T) {
	raw := []byte(`{
		"stepId": "rebalance_ap-42",
		"status": "pending",
		"action": "rebalance_ap",
		"payload": {
			"characterId": 42,
			"worldId": 0,
			"channelId": 1,
			"targets": [
				{"stat": "dexterity", "floor": 20}
			]
		},
		"createdAt": "2026-04-24T00:00:00Z",
		"updatedAt": "2026-04-24T00:00:00Z"
	}`)

	var step Step[any]
	if err := json.Unmarshal(raw, &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != RebalanceAP {
		t.Fatalf("expected action RebalanceAP, got %q", step.Action)
	}
	p, ok := step.Payload.(RebalanceAPPayload)
	if !ok {
		t.Fatalf("expected RebalanceAPPayload, got %T", step.Payload)
	}
	if p.CharacterId != 42 {
		t.Errorf("characterId: expected 42, got %d", p.CharacterId)
	}
	if len(p.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(p.Targets))
	}
	if p.Targets[0].Stat != RebalanceStatDexterity {
		t.Errorf("stat: expected dexterity, got %q", p.Targets[0].Stat)
	}
	if p.Targets[0].Floor != 20 {
		t.Errorf("floor: expected 20, got %d", p.Targets[0].Floor)
	}

	// Silence unused import warning if uuid isn't used elsewhere in the test.
	_ = uuid.Nil
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-saga && go test ./... -run TestUnmarshalRebalanceAPStep`
Expected: FAIL with undefined `RebalanceAP`, `RebalanceAPPayload`, `RebalanceStatDexterity`.

- [ ] **Step 3: Add the action constant to `libs/atlas-saga/model.go`**

In the `Character state actions` block (after `ResetStats` at line 69), add:

```go
	ResetStats             Action = "reset_stats"
	RebalanceAP            Action = "rebalance_ap"
	ValidateCharacterState Action = "validate_character_state"
```

- [ ] **Step 4: Add the payload struct and the rebalance-stat enum to `libs/atlas-saga/payloads.go`**

Immediately after the `ResetStatsPayload` block (which ends at line 194), insert:

```go
// RebalanceStat identifies which primary stat a RebalanceAP target operates on.
type RebalanceStat string

const (
	RebalanceStatStrength     RebalanceStat = "strength"
	RebalanceStatDexterity    RebalanceStat = "dexterity"
	RebalanceStatIntelligence RebalanceStat = "intelligence"
	RebalanceStatLuck         RebalanceStat = "luck"
)

// RebalanceTarget pairs a primary stat with the floor value it should be raised to.
type RebalanceTarget struct {
	Stat  RebalanceStat `json:"stat"`
	Floor uint16        `json:"floor"`
}

// RebalanceAPPayload represents the payload required to rebalance a character's
// primary stats during first-job advancement. The algorithm resets STR/DEX/INT/LUK
// to 4, raises each target stat to its floor, and returns the reclaimed surplus
// to unallocated AP. HP/MP are not touched.
type RebalanceAPPayload struct {
	CharacterId uint32            `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id          `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id        `json:"channelId"`   // ChannelId associated with the action
	Targets     []RebalanceTarget `json:"targets"`     // Target stats and floors to apply
}
```

- [ ] **Step 5: Add the unmarshal case to `libs/atlas-saga/unmarshal.go`**

Immediately after the `case ResetStats:` block (ending at line 149), insert:

```go
	case RebalanceAP:
		var payload RebalanceAPPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
```

- [ ] **Step 6: Run test to verify it passes**

Run: `cd libs/atlas-saga && go test ./... -run TestUnmarshalRebalanceAPStep`
Expected: PASS.

- [ ] **Step 7: Run full lib test suite and build**

Run: `cd libs/atlas-saga && go test ./... && go build ./...`
Expected: all tests pass, build succeeds.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-saga/
git commit -m "feat(atlas-saga): add RebalanceAP action, payload, and unmarshal case"
```

---

## Task 2: Mirror `RebalanceAP` into atlas-npc-conversations saga re-export

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/saga/model.go`

- [ ] **Step 1: Add the payload type re-export**

In `services/atlas-npc-conversations/atlas.com/npc/saga/model.go`, after the `ResetStatsPayload` entry (line 55), add:

```go
	ResetStatsPayload      = sharedsaga.ResetStatsPayload
	RebalanceAPPayload     = sharedsaga.RebalanceAPPayload
	RebalanceTarget        = sharedsaga.RebalanceTarget
```

- [ ] **Step 2: Add the action constant re-export**

In the same file, in the `Character stat actions` const block (after `ResetStats` at line 127), add:

```go
	SetHP         = sharedsaga.SetHP
	ResetStats    = sharedsaga.ResetStats
	RebalanceAP   = sharedsaga.RebalanceAP
```

- [ ] **Step 3: Also re-export the RebalanceStat enum values**

In the same const block (below `RebalanceAP`), add:

```go
	RebalanceStatStrength     = sharedsaga.RebalanceStatStrength
	RebalanceStatDexterity    = sharedsaga.RebalanceStatDexterity
	RebalanceStatIntelligence = sharedsaga.RebalanceStatIntelligence
	RebalanceStatLuck         = sharedsaga.RebalanceStatLuck
```

- [ ] **Step 4: Build to confirm**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go build ./saga/...`
Expected: build succeeds.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/saga/model.go
git commit -m "feat(atlas-npc-conversations): re-export RebalanceAP action and payload"
```

---

## Task 3: Add the `rebalance_ap` case to the NPC-conversations operation executor

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go`
- Test: `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go`:

```go
func TestCreateStepForOperation_RebalanceAP_SingleTarget(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(77)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}

	op, err := NewOperationBuilder().
		SetType("rebalance_ap").
		AddParamValue("targets", `[{"stat":"dexterity","floor":"20"}]`).
		Build()
	if err != nil {
		t.Fatalf("build op: %v", err)
	}

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	stepId, status, action, payload, err := executor.createStepForOperation(f, characterId, op)
	if err != nil {
		t.Fatalf("createStepForOperation returned error: %v", err)
	}
	if action != saga.RebalanceAP {
		t.Fatalf("expected action RebalanceAP, got %q", action)
	}
	if status != saga.Pending {
		t.Fatalf("expected status Pending, got %q", status)
	}
	if stepId == "" {
		t.Fatal("expected non-empty stepId")
	}
	p, ok := payload.(saga.RebalanceAPPayload)
	if !ok {
		t.Fatalf("expected RebalanceAPPayload, got %T", payload)
	}
	if p.CharacterId != characterId {
		t.Errorf("characterId: expected %d, got %d", characterId, p.CharacterId)
	}
	if len(p.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(p.Targets))
	}
	if p.Targets[0].Stat != sharedsaga.RebalanceStatDexterity || p.Targets[0].Floor != 20 {
		t.Errorf("unexpected target: %+v", p.Targets[0])
	}
}

func TestCreateStepForOperation_RebalanceAP_MultiTarget(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(78)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}

	op, err := NewOperationBuilder().
		SetType("rebalance_ap").
		AddParamValue("targets", `[{"stat":"strength","floor":"20"},{"stat":"dexterity","floor":"20"}]`).
		Build()
	if err != nil {
		t.Fatalf("build op: %v", err)
	}
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	_, _, action, payload, err := executor.createStepForOperation(f, characterId, op)
	if err != nil {
		t.Fatalf("createStepForOperation returned error: %v", err)
	}
	if action != saga.RebalanceAP {
		t.Fatalf("expected RebalanceAP action")
	}
	p := payload.(saga.RebalanceAPPayload)
	if len(p.Targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(p.Targets))
	}
	if p.Targets[0].Stat != sharedsaga.RebalanceStatStrength || p.Targets[0].Floor != 20 {
		t.Errorf("target[0] unexpected: %+v", p.Targets[0])
	}
	if p.Targets[1].Stat != sharedsaga.RebalanceStatDexterity || p.Targets[1].Floor != 20 {
		t.Errorf("target[1] unexpected: %+v", p.Targets[1])
	}
}

func TestCreateStepForOperation_RebalanceAP_RejectsEmpty(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(79)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}

	op, _ := NewOperationBuilder().
		SetType("rebalance_ap").
		AddParamValue("targets", `[]`).
		Build()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	if _, _, _, _, err := executor.createStepForOperation(f, characterId, op); err == nil {
		t.Fatal("expected error on empty targets")
	}
}

func TestCreateStepForOperation_RebalanceAP_RejectsDuplicateStat(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(80)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}

	op, _ := NewOperationBuilder().
		SetType("rebalance_ap").
		AddParamValue("targets", `[{"stat":"dexterity","floor":"20"},{"stat":"dexterity","floor":"25"}]`).
		Build()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	if _, _, _, _, err := executor.createStepForOperation(f, characterId, op); err == nil {
		t.Fatal("expected error on duplicate stat")
	}
}

func TestCreateStepForOperation_RebalanceAP_RejectsInvalidStat(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	l, _ := test.NewNullLogger()
	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)
	characterId := uint32(81)
	GetRegistry().SetContext(tctx, characterId, NewConversationContextBuilder().SetCharacterId(characterId).Build())
	defer GetRegistry().ClearContext(tctx, characterId)

	executor := &OperationExecutorImpl{l: l, ctx: tctx, t: tm}

	op, _ := NewOperationBuilder().
		SetType("rebalance_ap").
		AddParamValue("targets", `[{"stat":"banana","floor":"20"}]`).
		Build()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	if _, _, _, _, err := executor.createStepForOperation(f, characterId, op); err == nil {
		t.Fatal("expected error on invalid stat")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run TestCreateStepForOperation_RebalanceAP`
Expected: FAIL with "unsupported operation" or missing `rebalance_ap` case.

- [ ] **Step 3: Implement the `rebalance_ap` case in `operation_executor.go`**

Immediately after the `reset_stats` case (ending at line 2199), insert:

```go
	case "rebalance_ap":
		// Format: rebalance_ap
		// Params: targets (JSON array of {stat:<name>, floor:<int>})
		// Applied during first-job advancement: zeroes STR/DEX/INT/LUK, raises each
		// target stat to its floor, returns reclaimed surplus to unallocated AP.
		targetsRaw, exists := operation.Params()["targets"]
		if !exists {
			return "", "", "", nil, errors.New("missing targets parameter for rebalance_ap operation")
		}
		targets, err := parseRebalanceTargets(targetsRaw)
		if err != nil {
			return "", "", "", nil, fmt.Errorf("rebalance_ap: %w", err)
		}
		if len(targets) == 0 {
			return "", "", "", nil, errors.New("rebalance_ap requires at least one target")
		}
		seen := make(map[saga.RebalanceStat]struct{}, len(targets))
		for _, tt := range targets {
			if _, dup := seen[tt.Stat]; dup {
				return "", "", "", nil, fmt.Errorf("rebalance_ap: duplicate target stat %q", tt.Stat)
			}
			seen[tt.Stat] = struct{}{}
		}

		payload := saga.RebalanceAPPayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			Targets:     targets,
		}

		return stepId, saga.Pending, saga.RebalanceAP, payload, nil
```

- [ ] **Step 4: Add the `parseRebalanceTargets` helper near the top of `operation_executor.go`**

Place this new function immediately after `evaluateContextValueAsInt` (which ends near line 197):

```go
// parseRebalanceTargets parses a JSON-encoded array of {stat, floor} entries into
// typed RebalanceTarget slice. Accepts either integer or string-encoded floor values
// to match the rest of the operation_executor's lenient param decoding.
func parseRebalanceTargets(raw string) ([]saga.RebalanceTarget, error) {
	var items []struct {
		Stat  string      `json:"stat"`
		Floor interface{} `json:"floor"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, fmt.Errorf("invalid targets JSON: %w", err)
	}
	out := make([]saga.RebalanceTarget, 0, len(items))
	for i, it := range items {
		stat := saga.RebalanceStat(it.Stat)
		switch stat {
		case saga.RebalanceStatStrength, saga.RebalanceStatDexterity,
			saga.RebalanceStatIntelligence, saga.RebalanceStatLuck:
		default:
			return nil, fmt.Errorf("target[%d]: invalid stat %q", i, it.Stat)
		}
		var floorInt int
		switch v := it.Floor.(type) {
		case float64:
			floorInt = int(v)
		case string:
			parsed, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("target[%d]: invalid floor %q: %w", i, v, err)
			}
			floorInt = parsed
		default:
			return nil, fmt.Errorf("target[%d]: floor must be a number or numeric string", i)
		}
		if floorInt < 4 {
			return nil, fmt.Errorf("target[%d]: floor must be >= 4, got %d", i, floorInt)
		}
		out = append(out, saga.RebalanceTarget{Stat: stat, Floor: uint16(floorInt)})
	}
	return out, nil
}
```

If `encoding/json` is not already imported in this file, add it to the import block; verify first with a quick grep before editing.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run TestCreateStepForOperation_RebalanceAP`
Expected: PASS for all five subtests.

- [ ] **Step 6: Run the full conversation package tests**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/...`
Expected: no regressions.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go
git commit -m "feat(atlas-npc-conversations): dispatch rebalance_ap operation to saga step"
```

---

## Task 4: Add `computeRebalance` pure helper in atlas-character

**Files:**
- Create: `services/atlas-character/atlas.com/character/character/rebalance.go`
- Test: `services/atlas-character/atlas.com/character/character/rebalance_test.go`

- [ ] **Step 1: Write the failing table-driven test**

Create `services/atlas-character/atlas.com/character/character/rebalance_test.go`:

```go
package character

import (
	"testing"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

func TestComputeRebalance(t *testing.T) {
	tests := []struct {
		name          string
		str, dex, in_, luk, unalloc uint16
		targets       []sharedsaga.RebalanceTarget
		wantStr       uint16
		wantDex       uint16
		wantInt       uint16
		wantLuk       uint16
		wantUnalloc   uint16
		wantErr       bool
	}{
		{
			name:  "pirate reference video — DEX 20",
			str: 53, dex: 9, in_: 4, luk: 4, unalloc: 0,
			targets:     []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatDexterity, Floor: 20}},
			wantStr: 4, wantDex: 20, wantInt: 4, wantLuk: 4, wantUnalloc: 38,
		},
		{
			name:  "bowman/thief/wind-archer — DEX 25",
			str: 53, dex: 9, in_: 4, luk: 4, unalloc: 0,
			targets:     []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatDexterity, Floor: 25}},
			wantStr: 4, wantDex: 25, wantInt: 4, wantLuk: 4, wantUnalloc: 33,
		},
		{
			name:  "warrior/dawn-warrior — STR 35 (surplus boundary)",
			str: 53, dex: 9, in_: 4, luk: 4, unalloc: 0,
			targets:     []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatStrength, Floor: 35}},
			wantStr: 35, wantDex: 4, wantInt: 4, wantLuk: 4, wantUnalloc: 23,
		},
		{
			name:  "magician/blaze-wizard — INT 20",
			str: 53, dex: 9, in_: 4, luk: 4, unalloc: 0,
			targets:     []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatIntelligence, Floor: 20}},
			wantStr: 4, wantDex: 4, wantInt: 20, wantLuk: 4, wantUnalloc: 38,
		},
		{
			name:  "night-walker — LUK 25",
			str: 53, dex: 9, in_: 4, luk: 4, unalloc: 0,
			targets:     []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatLuck, Floor: 25}},
			wantStr: 4, wantDex: 4, wantInt: 4, wantLuk: 25, wantUnalloc: 33,
		},
		{
			name:  "thunder-breaker — STR 20 + DEX 20 (multi-target)",
			str: 53, dex: 9, in_: 4, luk: 4, unalloc: 0,
			targets: []sharedsaga.RebalanceTarget{
				{Stat: sharedsaga.RebalanceStatStrength, Floor: 20},
				{Stat: sharedsaga.RebalanceStatDexterity, Floor: 20},
			},
			wantStr: 20, wantDex: 20, wantInt: 4, wantLuk: 4, wantUnalloc: 22,
		},
		{
			name:  "existing unallocated AP carries through",
			str: 53, dex: 9, in_: 4, luk: 4, unalloc: 5,
			targets:     []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatDexterity, Floor: 20}},
			wantStr: 4, wantDex: 20, wantInt: 4, wantLuk: 4, wantUnalloc: 43,
		},
		{
			name:  "insufficient AP returns error",
			str: 4, dex: 4, in_: 4, luk: 4, unalloc: 0,
			targets: []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatDexterity, Floor: 20}},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := computeRebalance(tc.str, tc.dex, tc.in_, tc.luk, tc.unalloc, tc.targets)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result=%+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Str != tc.wantStr || got.Dex != tc.wantDex || got.Int != tc.wantInt || got.Luk != tc.wantLuk || got.Unallocated != tc.wantUnalloc {
				t.Errorf("mismatch: got=%+v want STR=%d DEX=%d INT=%d LUK=%d AP=%d",
					got, tc.wantStr, tc.wantDex, tc.wantInt, tc.wantLuk, tc.wantUnalloc)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run TestComputeRebalance`
Expected: FAIL — `computeRebalance` is undefined.

- [ ] **Step 3: Implement `computeRebalance` and supporting types**

Create `services/atlas-character/atlas.com/character/character/rebalance.go`:

```go
package character

import (
	"fmt"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// rebalanceResult is the output of computeRebalance: the new primary stat values
// and the new unallocated AP total. All values are post-rebalance.
type rebalanceResult struct {
	Str, Dex, Int, Luk uint16
	Unallocated        uint16
}

// computeRebalance implements the first-job AP rebalance algorithm.
//   1. reclaimed = Σ max(0, stat - 4) over the four primary stats.
//   2. All four primaries reset to 4.
//   3. For each target, the corresponding stat is raised to target.Floor.
//   4. cost = Σ (target.Floor - 4) across targets.
//   5. newUnallocated = unallocated + reclaimed - cost.
// Returns an error if newUnallocated would be negative. Callers must ensure targets
// contain no duplicate stats — the helper trusts that invariant.
func computeRebalance(str, dex, in_, luk, unallocated uint16, targets []sharedsaga.RebalanceTarget) (rebalanceResult, error) {
	const base uint16 = 4

	reclaimed := uint32(0)
	if str > base {
		reclaimed += uint32(str - base)
	}
	if dex > base {
		reclaimed += uint32(dex - base)
	}
	if in_ > base {
		reclaimed += uint32(in_ - base)
	}
	if luk > base {
		reclaimed += uint32(luk - base)
	}

	result := rebalanceResult{Str: base, Dex: base, Int: base, Luk: base}

	cost := uint32(0)
	for _, t := range targets {
		if t.Floor < base {
			return rebalanceResult{}, fmt.Errorf("rebalance target floor %d is below base %d", t.Floor, base)
		}
		cost += uint32(t.Floor - base)
		switch t.Stat {
		case sharedsaga.RebalanceStatStrength:
			result.Str = t.Floor
		case sharedsaga.RebalanceStatDexterity:
			result.Dex = t.Floor
		case sharedsaga.RebalanceStatIntelligence:
			result.Int = t.Floor
		case sharedsaga.RebalanceStatLuck:
			result.Luk = t.Floor
		default:
			return rebalanceResult{}, fmt.Errorf("unknown rebalance stat %q", t.Stat)
		}
	}

	available := uint32(unallocated) + reclaimed
	if cost > available {
		return rebalanceResult{}, fmt.Errorf("insufficient AP for rebalance: need %d, have %d", cost, available)
	}
	result.Unallocated = uint16(available - cost)
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run TestComputeRebalance -v`
Expected: PASS, all 8 subtests.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-character/atlas.com/character/character/rebalance.go services/atlas-character/atlas.com/character/character/rebalance_test.go
git commit -m "feat(atlas-character): add computeRebalance pure helper for first-job AP redistribution"
```

---

## Task 5: Add `RebalanceAP(mb)` + `RebalanceAPAndEmit` processor methods

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/processor.go`
- Test: `services/atlas-character/atlas.com/character/character/processor_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-character/atlas.com/character/character/processor_test.go`:

```go
func TestRebalanceAP_PersistsAndEmits(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	proc := character.NewProcessor(testLogger(), tctx, db)

	// Seed a level-10 beginner with vanilla v83 auto-allocated stats.
	input := character.NewModelBuilder().
		SetAccountId(1000).SetWorldId(0).SetName("PirateRef").
		SetLevel(10).SetStrength(53).SetDexterity(9).SetIntelligence(4).SetLuck(4).SetAP(0).
		Build()
	c, err := proc.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	ch := channel.NewModel(0, 1)
	buf := message.NewBuffer()
	targets := []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatDexterity, Floor: 20}}
	if err := proc.RebalanceAP(buf)(uuid.New(), c.Id(), ch, targets); err != nil {
		t.Fatalf("RebalanceAP: %v", err)
	}

	refreshed, err := proc.GetById()(c.Id())
	if err != nil {
		t.Fatalf("reread: %v", err)
	}
	if refreshed.Strength() != 4 || refreshed.Dexterity() != 20 || refreshed.Intelligence() != 4 || refreshed.Luck() != 4 {
		t.Errorf("stats: STR=%d DEX=%d INT=%d LUK=%d; want 4/20/4/4",
			refreshed.Strength(), refreshed.Dexterity(), refreshed.Intelligence(), refreshed.Luck())
	}
	if refreshed.AP() != 38 {
		t.Errorf("AP: got %d, want 38", refreshed.AP())
	}
	if refreshed.Hp() != c.Hp() || refreshed.Mp() != c.Mp() {
		t.Errorf("HP/MP changed: before HP=%d MP=%d, after HP=%d MP=%d", c.Hp(), c.Mp(), refreshed.Hp(), refreshed.Mp())
	}
	if refreshed.MaxHp() != c.MaxHp() || refreshed.MaxMp() != c.MaxMp() {
		t.Errorf("MaxHP/MaxMP changed")
	}
}

func TestRebalanceAP_ErrorDoesNotMutate(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)
	proc := character.NewProcessor(testLogger(), tctx, db)

	input := character.NewModelBuilder().
		SetAccountId(1000).SetWorldId(0).SetName("LowAP").
		SetLevel(1).SetStrength(4).SetDexterity(4).SetIntelligence(4).SetLuck(4).SetAP(0).
		Build()
	c, err := proc.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	ch := channel.NewModel(0, 1)
	buf := message.NewBuffer()
	targets := []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatDexterity, Floor: 20}}
	if err := proc.RebalanceAP(buf)(uuid.New(), c.Id(), ch, targets); err == nil {
		t.Fatal("expected error on insufficient AP, got nil")
	}

	refreshed, err := proc.GetById()(c.Id())
	if err != nil {
		t.Fatalf("reread: %v", err)
	}
	if refreshed.Strength() != 4 || refreshed.Dexterity() != 4 || refreshed.Intelligence() != 4 || refreshed.Luck() != 4 || refreshed.AP() != 0 {
		t.Errorf("entity mutated on error path: %+v", refreshed)
	}
}
```

Also add the `sharedsaga` import to the test file's import block if not present:

```go
sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run TestRebalanceAP`
Expected: FAIL — `RebalanceAP` is undefined.

- [ ] **Step 3: Add the interface entry**

In `services/atlas-character/atlas.com/character/character/processor.go`, find the `Processor` interface and add these two lines next to the existing `ResetStatsAndEmit` / `ResetStats` declarations (search for `ResetStatsAndEmit`, currently around line 109):

```go
	ResetStatsAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model) error
	ResetStats(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model) error
	RebalanceAPAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, targets []sharedsaga.RebalanceTarget) error
	RebalanceAP(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, targets []sharedsaga.RebalanceTarget) error
```

Ensure `sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"` is present in the imports.

- [ ] **Step 4: Implement the two methods**

Immediately after the existing `ResetStats` method (currently ending around line 1844), append:

```go
func (p *ProcessorImpl) RebalanceAPAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, targets []sharedsaga.RebalanceTarget) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.RebalanceAP(buf)(transactionId, characterId, channel, targets)
	})
}

func (p *ProcessorImpl) RebalanceAP(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, targets []sharedsaga.RebalanceTarget) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, targets []sharedsaga.RebalanceTarget) error {
		var beforeStr, beforeDex, beforeInt, beforeLuk, beforeAP uint16
		var result rebalanceResult

		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			beforeStr, beforeDex, beforeInt, beforeLuk, beforeAP = c.Strength(), c.Dexterity(), c.Intelligence(), c.Luck(), c.AP()
			result, err = computeRebalance(beforeStr, beforeDex, beforeInt, beforeLuk, beforeAP, targets)
			if err != nil {
				return err
			}
			return dynamicUpdate(tx)(
				SetStrength(result.Str),
				SetDexterity(result.Dex),
				SetIntelligence(result.Int),
				SetLuck(result.Luk),
				SetAP(result.Unallocated),
			)(c)
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not rebalance AP for character [%d].", characterId)
			return txErr
		}

		p.l.Infof("Rebalanced character [%d] AP. Before STR=%d DEX=%d INT=%d LUK=%d AP=%d → after STR=%d DEX=%d INT=%d LUK=%d AP=%d, targets=%+v",
			characterId,
			beforeStr, beforeDex, beforeInt, beforeLuk, beforeAP,
			result.Str, result.Dex, result.Int, result.Luk, result.Unallocated,
			targets)

		values := map[string]interface{}{
			"strength":     result.Str,
			"dexterity":    result.Dex,
			"intelligence": result.Int,
			"luck":         result.Luk,
		}
		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, []stat.Type{stat.TypeAvailableAP, stat.TypeStrength, stat.TypeDexterity, stat.TypeIntelligence, stat.TypeLuck}, values))
		return nil
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run TestRebalanceAP -v`
Expected: both `TestRebalanceAP_PersistsAndEmits` and `TestRebalanceAP_ErrorDoesNotMutate` PASS.

- [ ] **Step 6: Run full character package tests**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/...`
Expected: no regressions.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-character/atlas.com/character/character/processor.go services/atlas-character/atlas.com/character/character/processor_test.go
git commit -m "feat(atlas-character): add RebalanceAPAndEmit processor method"
```

---

## Task 6: Add `REBALANCE_AP` Kafka command type in atlas-character

**Files:**
- Modify: `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go`

- [ ] **Step 1: Add the command constant**

In the top `const` block (currently ending at line 34 with `CommandDeleteCharacter`), insert after `CommandResetStats` (line 31):

```go
	CommandResetStats          = "RESET_STATS"
	CommandRebalanceAP         = "REBALANCE_AP"
	CommandClampHP             = "CLAMP_HP"
```

- [ ] **Step 2: Add the body type**

Immediately after `ResetStatsCommandBody` (line 187-189), insert:

```go
type ResetStatsCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
}

type RebalanceAPCommandBody struct {
	ChannelId channel.Id                   `json:"channelId"`
	Targets   []RebalanceAPTarget          `json:"targets"`
}

// RebalanceAPTarget mirrors sharedsaga.RebalanceTarget on the Kafka wire.
// Kept local to avoid pulling sharedsaga into the kafka/message package.
type RebalanceAPTarget struct {
	Stat  string `json:"stat"`
	Floor uint16 `json:"floor"`
}
```

- [ ] **Step 3: Build to confirm**

Run: `cd services/atlas-character/atlas.com/character && go build ./...`
Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-character/atlas.com/character/kafka/message/character/kafka.go
git commit -m "feat(atlas-character): add REBALANCE_AP Kafka command type"
```

---

## Task 7: Wire up `handleRebalanceAP` consumer in atlas-character

**Files:**
- Modify: `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go`

- [ ] **Step 1: Add the handler function**

Immediately after `handleResetStats` (currently ending around line 391), insert:

```go
func handleRebalanceAP(db *gorm.DB) message.Handler[character2.Command[character2.RebalanceAPCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.RebalanceAPCommandBody]) {
		if c.Type != character2.CommandRebalanceAP {
			return
		}

		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		targets := make([]sharedsaga.RebalanceTarget, 0, len(c.Body.Targets))
		for _, t := range c.Body.Targets {
			targets = append(targets, sharedsaga.RebalanceTarget{
				Stat:  sharedsaga.RebalanceStat(t.Stat),
				Floor: t.Floor,
			})
		}
		if err := character.NewProcessor(l, ctx, db).RebalanceAPAndEmit(c.TransactionId, c.CharacterId, cha, targets); err != nil {
			l.WithError(err).Errorf("Unable to rebalance AP for character [%d].", c.CharacterId)
		}
	}
}
```

Add the `sharedsaga` import at the top of the file if not already present:

```go
sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
```

- [ ] **Step 2: Register the handler in `InitHandlers`**

In `InitHandlers`, immediately after the `handleResetStats` registration (line 86-88), insert:

```go
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleResetStats(db)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRebalanceAP(db)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleClampHP(db)))); err != nil {
					return err
				}
```

- [ ] **Step 3: Build to confirm**

Run: `cd services/atlas-character/atlas.com/character && go build ./...`
Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go
git commit -m "feat(atlas-character): register REBALANCE_AP command handler"
```

---

## Task 8: Add `REBALANCE_AP` Kafka command in saga-orchestrator

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/character/kafka.go`

- [ ] **Step 1: Add constant and body type**

Insert into the `const` block (after `CommandResetStats` at line 31):

```go
	CommandResetStats          = "RESET_STATS"
	CommandRebalanceAP         = "REBALANCE_AP"
	CommandDeleteCharacter     = "DELETE_CHARACTER"
```

Insert after `ResetStatsCommandBody` (line 157-159):

```go
type ResetStatsCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
}

type RebalanceAPCommandBody struct {
	ChannelId channel.Id          `json:"channelId"`
	Targets   []RebalanceAPTarget `json:"targets"`
}

type RebalanceAPTarget struct {
	Stat  string `json:"stat"`
	Floor uint16 `json:"floor"`
}
```

- [ ] **Step 2: Build**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./kafka/...`
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/character/kafka.go
git commit -m "feat(saga-orchestrator): add REBALANCE_AP Kafka command type"
```

---

## Task 9: Add `RebalanceAPProvider` Kafka producer in saga-orchestrator

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/producer.go`

- [ ] **Step 1: Append the new provider**

Immediately after `ResetStatsProvider` (currently ending at line 200), append:

```go
func RebalanceAPProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, targets []character2.RebalanceAPTarget) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.RebalanceAPCommandBody]{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandRebalanceAP,
		Body: character2.RebalanceAPCommandBody{
			ChannelId: ch.Id(),
			Targets:   targets,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 2: Build**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./character/...`
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/producer.go
git commit -m "feat(saga-orchestrator): add RebalanceAPProvider Kafka producer"
```

---

## Task 10: Extend saga-orchestrator character Processor with `RebalanceAPAndEmit`

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/processor.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/mock/processor.go`

- [ ] **Step 1: Extend the Processor interface**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/processor.go`, immediately after the `ResetStats` interface entry (line 49), add:

```go
	ResetStatsAndEmit(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	ResetStats(mb *message.Buffer) func(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	RebalanceAPAndEmit(transactionId uuid.UUID, ch channel.Model, characterId uint32, targets []character2.RebalanceAPTarget) error
	RebalanceAP(mb *message.Buffer) func(transactionId uuid.UUID, ch channel.Model, characterId uint32, targets []character2.RebalanceAPTarget) error
```

- [ ] **Step 2: Append the two method implementations**

At the end of the file (after `ResetStats` at line 242), append:

```go
func (p *ProcessorImpl) RebalanceAPAndEmit(transactionId uuid.UUID, ch channel.Model, characterId uint32, targets []character2.RebalanceAPTarget) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.RebalanceAP(mb)(transactionId, ch, characterId, targets)
	})
}

func (p *ProcessorImpl) RebalanceAP(mb *message.Buffer) func(transactionId uuid.UUID, ch channel.Model, characterId uint32, targets []character2.RebalanceAPTarget) error {
	return func(transactionId uuid.UUID, ch channel.Model, characterId uint32, targets []character2.RebalanceAPTarget) error {
		return mb.Put(character2.EnvCommandTopic, RebalanceAPProvider(transactionId, ch, characterId, targets))
	}
}
```

- [ ] **Step 3: Extend the mock**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/mock/processor.go`, add a new function field next to `ResetStatsAndEmitFunc` (line 44):

```go
	ResetStatsAndEmitFunc       func(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	RebalanceAPAndEmitFunc      func(transactionId uuid.UUID, ch channel.Model, characterId uint32, targets []character2.RebalanceAPTarget) error
```

After the existing `ResetStats` mock methods (around line 289), append:

```go
// RebalanceAPAndEmit is a mock implementation of Processor.RebalanceAPAndEmit.
func (m *ProcessorMock) RebalanceAPAndEmit(transactionId uuid.UUID, ch channel.Model, characterId uint32, targets []character2.RebalanceAPTarget) error {
	if m.RebalanceAPAndEmitFunc != nil {
		return m.RebalanceAPAndEmitFunc(transactionId, ch, characterId, targets)
	}
	return nil
}

func (m *ProcessorMock) RebalanceAP(mb *message.Buffer) func(transactionId uuid.UUID, ch channel.Model, characterId uint32, targets []character2.RebalanceAPTarget) error {
	return func(transactionId uuid.UUID, ch channel.Model, characterId uint32, targets []character2.RebalanceAPTarget) error {
		return m.RebalanceAPAndEmit(transactionId, ch, characterId, targets)
	}
}
```

Verify `character2` is imported in both files; if not, add `character2 "atlas-saga-orchestrator/kafka/message/character"` to the import block.

- [ ] **Step 4: Build**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./...`
Expected: build succeeds; no interface-unimplemented errors.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/
git commit -m "feat(saga-orchestrator): add RebalanceAPAndEmit to character Processor"
```

---

## Task 11: Add `RebalanceAP` saga action + payload + unmarshal + handler in saga-orchestrator

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`

- [ ] **Step 1: Re-export the action constant**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go`, in the `Character state actions` block after `ResetStats` (line 77), add:

```go
	ResetStats             = sharedsaga.ResetStats
	RebalanceAP            = sharedsaga.RebalanceAP
	ValidateCharacterState = sharedsaga.ValidateCharacterState
```

- [ ] **Step 2: Re-export the payload and related types**

In the payload type-re-export block, after `ResetStatsPayload` (line 189), add:

```go
	ResetStatsPayload                    = sharedsaga.ResetStatsPayload
	RebalanceAPPayload                   = sharedsaga.RebalanceAPPayload
	RebalanceTarget                      = sharedsaga.RebalanceTarget
	RebalanceStat                        = sharedsaga.RebalanceStat
```

And re-export the stat-name constants near the bottom of the const block:

```go
	RebalanceStatStrength     = sharedsaga.RebalanceStatStrength
	RebalanceStatDexterity    = sharedsaga.RebalanceStatDexterity
	RebalanceStatIntelligence = sharedsaga.RebalanceStatIntelligence
	RebalanceStatLuck         = sharedsaga.RebalanceStatLuck
```

- [ ] **Step 3: Add the unmarshal case**

In `model.go`, in the unmarshal switch statement, after `case ResetStats:` (line 1110-1115), insert:

```go
	case ResetStats:
		var payload ResetStatsPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case RebalanceAP:
		var payload RebalanceAPPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
```

- [ ] **Step 4: Register the handler dispatch**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`, in the `GetHandler` switch (line ~824), immediately after `case ResetStats:` block, insert:

```go
	case ResetStats:
		return h.handleResetStats, true
	case RebalanceAP:
		return h.handleRebalanceAP, true
```

- [ ] **Step 5: Implement `handleRebalanceAP`**

Immediately after `handleResetStats` (currently ending at line 2209), append:

```go
// handleRebalanceAP handles the RebalanceAP action.
// Used during first-job advancement to redistribute primary stats to the class floor.
func (h *HandlerImpl) handleRebalanceAP(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(RebalanceAPPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	ch := channel.NewModel(payload.WorldId, payload.ChannelId)
	kafkaTargets := make([]character2.RebalanceAPTarget, 0, len(payload.Targets))
	for _, t := range payload.Targets {
		kafkaTargets = append(kafkaTargets, character2.RebalanceAPTarget{
			Stat:  string(t.Stat),
			Floor: t.Floor,
		})
	}
	err := h.charP.RebalanceAPAndEmit(s.TransactionId(), ch, payload.CharacterId, kafkaTargets)
	if err != nil {
		h.logActionError(s, st, err, "Unable to rebalance AP.")
		return err
	}
	return nil
}
```

If `character2 "atlas-saga-orchestrator/kafka/message/character"` isn't already imported in `handler.go`, add it (check with a grep first).

- [ ] **Step 6: Build**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./...`
Expected: build succeeds.

- [ ] **Step 7: Run saga-orchestrator tests**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/...`
Expected: no regressions.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/
git commit -m "feat(saga-orchestrator): dispatch RebalanceAP saga step to character processor"
```

---

## Task 12: Edit NPC JSON — Bowman (`npc_1012100.json`, DEX 25)

**Files:**
- Modify: `services/atlas-npc-conversations/conversations/npc/npc_1012100.json`

- [ ] **Step 1: Remove the stat-check condition**

Line 142 currently reads:
```json
          {"conditions": [{"type": "dexterity", "operator": "<", "value": "25"}], "nextState": "firstJobInsufficientLevel"},
```

Delete the entire line. The remaining `checkFirstJobRequirements` state (lines 136-146 before the delete) keeps only the `level < 10` check plus the default fall-through.

- [ ] **Step 2: Replace `reset_stats` with `rebalance_ap` in `firstJobAdvance`**

The `firstJobAdvance` operation block (lines 210-215) currently reads:

```json
        "operations": [
          {"type": "change_job", "params": {"jobId": "300"}},
          {"type": "award_item", "params": {"itemId": "1452051", "quantity": "1"}},
          {"type": "award_item", "params": {"itemId": "2060000", "quantity": "1000"}},
          {"type": "reset_stats", "params": {}}
        ],
```

Replace with:

```json
        "operations": [
          {"type": "rebalance_ap", "params": {"targets": "[{\"stat\":\"dexterity\",\"floor\":25}]"}},
          {"type": "change_job", "params": {"jobId": "300"}},
          {"type": "award_item", "params": {"itemId": "1452051", "quantity": "1"}},
          {"type": "award_item", "params": {"itemId": "2060000", "quantity": "1000"}}
        ],
```

Note: `rebalance_ap` is placed first (before `change_job`) per design §4.3 so stat broadcasts precede the job-change broadcast. `reset_stats` is dropped.

- [ ] **Step 3: Validate JSON syntax**

Run: `python3 -c 'import json; json.load(open("services/atlas-npc-conversations/conversations/npc/npc_1012100.json"))'`
Expected: no output (valid JSON).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-npc-conversations/conversations/npc/npc_1012100.json
git commit -m "fix(atlas-npc-conversations): npc_1012100 (Bowman) use rebalance_ap DEX 25"
```

---

## Task 13: Edit NPC JSON — Warrior (`npc_1022000.json`, STR 35)

> **Note:** The design file table labels this file as Magician. That is wrong — this NPC advances to jobId 100 (Warrior). This task uses the correct STR 35 target.

**Files:**
- Modify: `services/atlas-npc-conversations/conversations/npc/npc_1022000.json`

- [ ] **Step 1: Remove only the strength condition from the combined check**

Line 143 currently reads:
```json
          {"conditions": [{"type": "level", "operator": ">=", "value": "10"}, {"type": "strength", "operator": ">=", "value": "35"}], "nextState": "firstJobOffer"},
```

Change to:
```json
          {"conditions": [{"type": "level", "operator": ">=", "value": "10"}], "nextState": "firstJobOffer"},
```

The level check survives; the strength check is removed.

- [ ] **Step 2: Replace `reset_stats` with `rebalance_ap` in `firstJobAdvance`**

Lines 200-204 currently read:

```json
        "operations": [
          {"type": "change_job", "params": {"jobId": "100"}},
          {"type": "award_item", "params": {"itemId": "1302077", "quantity": "1"}},
          {"type": "reset_stats", "params": {}}
        ],
```

Replace with:

```json
        "operations": [
          {"type": "rebalance_ap", "params": {"targets": "[{\"stat\":\"strength\",\"floor\":35}]"}},
          {"type": "change_job", "params": {"jobId": "100"}},
          {"type": "award_item", "params": {"itemId": "1302077", "quantity": "1"}}
        ],
```

- [ ] **Step 3: Validate JSON**

Run: `python3 -c 'import json; json.load(open("services/atlas-npc-conversations/conversations/npc/npc_1022000.json"))'`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-npc-conversations/conversations/npc/npc_1022000.json
git commit -m "fix(atlas-npc-conversations): npc_1022000 (Warrior) use rebalance_ap STR 35"
```

---

## Task 14: Edit NPC JSON — Magician (`npc_1032001.json`, INT 20)

> **Note:** The design file table labels this file as Warrior. That is wrong — this NPC advances to jobId 200 (Magician). This task uses the correct INT 20 target. This file has no existing stat-check gate (only `level < 8`); the plan only inserts `rebalance_ap` and removes `reset_stats`.

**Files:**
- Modify: `services/atlas-npc-conversations/conversations/npc/npc_1032001.json`

- [ ] **Step 1: Replace `reset_stats` with `rebalance_ap` in `performFirstJobAdvancement`**

Lines 286-303 currently contain:

```json
        "operations": [
          {
            "type": "change_job",
            "params": {
              "jobId": "200"
            }
          },
          {
            "type": "award_item",
            "params": {
              "itemId": "1372043",
              "quantity": "1"
            }
          },
          {
            "type": "reset_stats"
          }
        ],
```

Replace with:

```json
        "operations": [
          {
            "type": "rebalance_ap",
            "params": {
              "targets": "[{\"stat\":\"intelligence\",\"floor\":20}]"
            }
          },
          {
            "type": "change_job",
            "params": {
              "jobId": "200"
            }
          },
          {
            "type": "award_item",
            "params": {
              "itemId": "1372043",
              "quantity": "1"
            }
          }
        ],
```

- [ ] **Step 2: Validate JSON**

Run: `python3 -c 'import json; json.load(open("services/atlas-npc-conversations/conversations/npc/npc_1032001.json"))'`
Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-npc-conversations/conversations/npc/npc_1032001.json
git commit -m "fix(atlas-npc-conversations): npc_1032001 (Magician) use rebalance_ap INT 20"
```

---

## Task 15: Edit NPC JSON — Thief (`npc_1052001.json`, DEX 25)

**Files:**
- Modify: `services/atlas-npc-conversations/conversations/npc/npc_1052001.json`

- [ ] **Step 1: Remove the DEX check branch**

Lines 167-174 currently read:

```json
    {
      "id": "firstJobCheckDex",
      "type": "genericAction",
      "genericAction": {
        "operations": [],
        "outcomes": [
          {"conditions": [{"type": "dexterity", "operator": ">=", "value": "25"}], "nextState": "firstJobOffer"},
          {"conditions": [], "nextState": "firstJobRequirementsNotMet"}
```

Because this separate DEX state exists only as a gate, redirect the upstream level check to skip it. At line 161 the previous state reads:

```json
          {"conditions": [{"type": "level", "operator": ">=", "value": "10"}], "nextState": "firstJobCheckDex"},
```

Change `"firstJobCheckDex"` to `"firstJobOffer"` and delete the entire `firstJobCheckDex` state block (lines 167-176). Verify the `firstJobRequirementsNotMet` dialogue state is still referenced by the level-check fall-through (line 162); if so, keep it.

- [ ] **Step 2: Replace `reset_stats` with `rebalance_ap` in `firstJobAdvance`**

Lines 239-245 currently read:

```json
        "operations": [
          {"type": "change_job", "params": {"jobId": "400"}},
          {"type": "award_item", "params": {"itemId": "2070015", "quantity": "500"}},
          {"type": "award_item", "params": {"itemId": "1472061", "quantity": "1"}},
          {"type": "award_item", "params": {"itemId": "1332063", "quantity": "1"}},
          {"type": "reset_stats", "params": {}}
        ],
```

Replace with:

```json
        "operations": [
          {"type": "rebalance_ap", "params": {"targets": "[{\"stat\":\"dexterity\",\"floor\":25}]"}},
          {"type": "change_job", "params": {"jobId": "400"}},
          {"type": "award_item", "params": {"itemId": "2070015", "quantity": "500"}},
          {"type": "award_item", "params": {"itemId": "1472061", "quantity": "1"}},
          {"type": "award_item", "params": {"itemId": "1332063", "quantity": "1"}}
        ],
```

- [ ] **Step 3: Validate JSON**

Run: `python3 -c 'import json; json.load(open("services/atlas-npc-conversations/conversations/npc/npc_1052001.json"))'`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-npc-conversations/conversations/npc/npc_1052001.json
git commit -m "fix(atlas-npc-conversations): npc_1052001 (Thief) use rebalance_ap DEX 25"
```

---

## Task 16: Edit NPC JSON — Pirate (`npc_1090000.json`, DEX 20)

**Files:**
- Modify: `services/atlas-npc-conversations/conversations/npc/npc_1090000.json`

- [ ] **Step 1: Remove only the dexterity condition from the combined check**

Line 286 currently reads:
```json
          {"conditions": [{"type": "level", "operator": ">=", "value": "10"}, {"type": "dexterity", "operator": ">=", "value": "20"}], "nextState": "firstJobOffer"},
```

Change to:
```json
          {"conditions": [{"type": "level", "operator": ">=", "value": "10"}], "nextState": "firstJobOffer"},
```

- [ ] **Step 2: Replace `reset_stats` with `rebalance_ap` in the Pirate advance block**

Lines 343-348 currently read:

```json
        "operations": [
          {"type": "change_job", "params": {"jobId": "500"}},
          {"type": "award_item", "params": {"itemId": "1492000", "quantity": "1"}},
          {"type": "award_item", "params": {"itemId": "1482000", "quantity": "1"}},
          {"type": "award_item", "params": {"itemId": "2330000", "quantity": "1000"}},
          {"type": "reset_stats"}
        ],
```

Replace with:

```json
        "operations": [
          {"type": "rebalance_ap", "params": {"targets": "[{\"stat\":\"dexterity\",\"floor\":20}]"}},
          {"type": "change_job", "params": {"jobId": "500"}},
          {"type": "award_item", "params": {"itemId": "1492000", "quantity": "1"}},
          {"type": "award_item", "params": {"itemId": "1482000", "quantity": "1"}},
          {"type": "award_item", "params": {"itemId": "2330000", "quantity": "1000"}}
        ],
```

- [ ] **Step 3: Validate JSON**

Run: `python3 -c 'import json; json.load(open("services/atlas-npc-conversations/conversations/npc/npc_1090000.json"))'`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-npc-conversations/conversations/npc/npc_1090000.json
git commit -m "fix(atlas-npc-conversations): npc_1090000 (Pirate) use rebalance_ap DEX 20"
```

---

## Task 17: Edit Quest JSON — Dawn Warrior (`quest_20101.json`, STR 35)

**Files:**
- Modify: `services/atlas-npc-conversations/conversations/quests/quest_20101.json`

- [ ] **Step 1: Remove the STR check prerequisite**

Lines 65-70 currently contain:

```json
            {
              "conditions": [
                {"type": "stat", "operator": "<", "value": "35", "referenceId": "str"}
              ],
              "nextState": "requirementsNotMet"
            },
```

Delete the entire object (all six lines, including any trailing comma as needed to keep the parent array valid).

- [ ] **Step 2: Replace `reset_stats` with `rebalance_ap` in `performJobChange`**

Lines 130-135 currently read:

```json
          "operations": [
            {"type": "award_item", "params": {"itemId": "1302077", "quantity": "1"}},
            {"type": "award_item", "params": {"itemId": "1142066", "quantity": "1"}},
            {"type": "change_job", "params": {"jobId": "1100"}},
            {"type": "reset_stats", "params": {}},
            {"type": "complete_quest", "params": {}}
          ],
```

Replace with:

```json
          "operations": [
            {"type": "rebalance_ap", "params": {"targets": "[{\"stat\":\"strength\",\"floor\":35}]"}},
            {"type": "award_item", "params": {"itemId": "1302077", "quantity": "1"}},
            {"type": "award_item", "params": {"itemId": "1142066", "quantity": "1"}},
            {"type": "change_job", "params": {"jobId": "1100"}},
            {"type": "complete_quest", "params": {}}
          ],
```

- [ ] **Step 3: Validate JSON**

Run: `python3 -c 'import json; json.load(open("services/atlas-npc-conversations/conversations/quests/quest_20101.json"))'`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-npc-conversations/conversations/quests/quest_20101.json
git commit -m "fix(atlas-npc-conversations): quest_20101 (Dawn Warrior) use rebalance_ap STR 35"
```

---

## Task 18: Edit Quest JSON — Blaze Wizard (`quest_20102.json`, INT 20)

**Files:**
- Modify: `services/atlas-npc-conversations/conversations/quests/quest_20102.json`

- [ ] **Step 1: Remove the INT check prerequisite**

In the `checkRequirements` state, delete the entry `{"type": "stat", "operator": "<", "value": "20", "referenceId": "int"}` (located around line 67) and the containing condition object (including its `conditions`, `nextState: requirementsNotMet`, and trailing comma).

- [ ] **Step 2: Replace `reset_stats` with `rebalance_ap` in `performJobChange`**

Lines 130-136 currently read:

```json
          "operations": [
            {"type": "award_item", "params": {"itemId": "1372043", "quantity": "1"}},
            {"type": "award_item", "params": {"itemId": "1142066", "quantity": "1"}},
            {"type": "change_job", "params": {"jobId": "1200"}},
            {"type": "reset_stats", "params": {}},
            {"type": "complete_quest", "params": {}}
          ],
```

Replace with:

```json
          "operations": [
            {"type": "rebalance_ap", "params": {"targets": "[{\"stat\":\"intelligence\",\"floor\":20}]"}},
            {"type": "award_item", "params": {"itemId": "1372043", "quantity": "1"}},
            {"type": "award_item", "params": {"itemId": "1142066", "quantity": "1"}},
            {"type": "change_job", "params": {"jobId": "1200"}},
            {"type": "complete_quest", "params": {}}
          ],
```

- [ ] **Step 3: Validate JSON**

Run: `python3 -c 'import json; json.load(open("services/atlas-npc-conversations/conversations/quests/quest_20102.json"))'`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-npc-conversations/conversations/quests/quest_20102.json
git commit -m "fix(atlas-npc-conversations): quest_20102 (Blaze Wizard) use rebalance_ap INT 20"
```

---

## Task 19: Edit Quest JSON — Wind Archer (`quest_20103.json`, DEX 25)

**Files:**
- Modify: `services/atlas-npc-conversations/conversations/quests/quest_20103.json`

- [ ] **Step 1: Remove the DEX check prerequisite**

Delete the condition object containing `{"type": "stat", "operator": "<", "value": "25", "referenceId": "dex"}` (around line 67).

- [ ] **Step 2: Replace `reset_stats` with `rebalance_ap` in `performJobChange`**

Lines 137-142 currently read:

```json
          "operations": [
            {"type": "award_item", "params": {"itemId": "2060000", "quantity": "2000"}},
            {"type": "award_item", "params": {"itemId": "1452051", "quantity": "1"}},
            {"type": "award_item", "params": {"itemId": "1142066", "quantity": "1"}},
            {"type": "change_job", "params": {"jobId": "1300"}},
            {"type": "reset_stats", "params": {}},
            {"type": "complete_quest", "params": {}}
          ],
```

Replace with:

```json
          "operations": [
            {"type": "rebalance_ap", "params": {"targets": "[{\"stat\":\"dexterity\",\"floor\":25}]"}},
            {"type": "award_item", "params": {"itemId": "2060000", "quantity": "2000"}},
            {"type": "award_item", "params": {"itemId": "1452051", "quantity": "1"}},
            {"type": "award_item", "params": {"itemId": "1142066", "quantity": "1"}},
            {"type": "change_job", "params": {"jobId": "1300"}},
            {"type": "complete_quest", "params": {}}
          ],
```

- [ ] **Step 3: Validate JSON**

Run: `python3 -c 'import json; json.load(open("services/atlas-npc-conversations/conversations/quests/quest_20103.json"))'`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-npc-conversations/conversations/quests/quest_20103.json
git commit -m "fix(atlas-npc-conversations): quest_20103 (Wind Archer) use rebalance_ap DEX 25"
```

---

## Task 20: Edit Quest JSON — Night Walker (`quest_20104.json`, LUK 25)

**Files:**
- Modify: `services/atlas-npc-conversations/conversations/quests/quest_20104.json`

- [ ] **Step 1: Remove the LUK check prerequisite**

Delete the condition object containing `{"type": "stat", "operator": "<", "value": "25", "referenceId": "luk"}` (around line 67).

- [ ] **Step 2: Replace `reset_stats` with `rebalance_ap` in `performJobChange`**

Lines 137-142 currently read:

```json
          "operations": [
            {"type": "award_item", "params": {"itemId": "1472061", "quantity": "1"}},
            {"type": "award_item", "params": {"itemId": "2070000", "quantity": "800"}},
            {"type": "award_item", "params": {"itemId": "1142066", "quantity": "1"}},
            {"type": "change_job", "params": {"jobId": "1400"}},
            {"type": "reset_stats", "params": {}},
            {"type": "complete_quest", "params": {}}
          ],
```

Replace with:

```json
          "operations": [
            {"type": "rebalance_ap", "params": {"targets": "[{\"stat\":\"luck\",\"floor\":25}]"}},
            {"type": "award_item", "params": {"itemId": "1472061", "quantity": "1"}},
            {"type": "award_item", "params": {"itemId": "2070000", "quantity": "800"}},
            {"type": "award_item", "params": {"itemId": "1142066", "quantity": "1"}},
            {"type": "change_job", "params": {"jobId": "1400"}},
            {"type": "complete_quest", "params": {}}
          ],
```

- [ ] **Step 3: Validate JSON**

Run: `python3 -c 'import json; json.load(open("services/atlas-npc-conversations/conversations/quests/quest_20104.json"))'`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-npc-conversations/conversations/quests/quest_20104.json
git commit -m "fix(atlas-npc-conversations): quest_20104 (Night Walker) use rebalance_ap LUK 25"
```

---

## Task 21: Edit Quest JSON — Thunder Breaker (`quest_20105.json`, STR 20 + DEX 20)

**Files:**
- Modify: `services/atlas-npc-conversations/conversations/quests/quest_20105.json`

- [ ] **Step 1: Remove both STR and DEX check prerequisites**

Delete both condition objects — the one with `referenceId: "str"` (around line 67) and the one with `referenceId: "dex"` (around line 73).

- [ ] **Step 2: Replace `reset_stats` with multi-target `rebalance_ap` in `performJobChange`**

Lines 137-141 currently read:

```json
          "operations": [
            {"type": "award_item", "params": {"itemId": "1482014", "quantity": "1"}},
            {"type": "award_item", "params": {"itemId": "1142066", "quantity": "1"}},
            {"type": "change_job", "params": {"jobId": "1500"}},
            {"type": "reset_stats", "params": {}},
            {"type": "complete_quest", "params": {}}
          ],
```

Replace with:

```json
          "operations": [
            {"type": "rebalance_ap", "params": {"targets": "[{\"stat\":\"strength\",\"floor\":20},{\"stat\":\"dexterity\",\"floor\":20}]"}},
            {"type": "award_item", "params": {"itemId": "1482014", "quantity": "1"}},
            {"type": "award_item", "params": {"itemId": "1142066", "quantity": "1"}},
            {"type": "change_job", "params": {"jobId": "1500"}},
            {"type": "complete_quest", "params": {}}
          ],
```

- [ ] **Step 3: Validate JSON**

Run: `python3 -c 'import json; json.load(open("services/atlas-npc-conversations/conversations/quests/quest_20105.json"))'`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-npc-conversations/conversations/quests/quest_20105.json
git commit -m "fix(atlas-npc-conversations): quest_20105 (Thunder Breaker) use rebalance_ap STR 20 + DEX 20"
```

---

## Task 22: Add JSON-script validation test

**Files:**
- Create: `services/atlas-npc-conversations/atlas.com/npc/conversation/firstjob_scripts_test.go`

- [ ] **Step 1: Write the table-driven validation test**

Create `services/atlas-npc-conversations/atlas.com/npc/conversation/firstjob_scripts_test.go`:

```go
package conversation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type expectedTarget struct {
	Stat  string
	Floor int
}

type scriptCase struct {
	name         string
	relPath      string
	advanceState string // the state whose operations array must contain rebalance_ap + change_job
	targets      []expectedTarget
	bannedStats  []string // stats that must NOT appear in any condition "type" or referenceId
}

// TestFirstJobScriptsUseRebalanceAP asserts that every first-job advancement
// script has rebalance_ap present immediately before change_job in the
// advancement state, that reset_stats is not used in the advancement state,
// and that no stat-minimum gate condition remains anywhere in the script.
func TestFirstJobScriptsUseRebalanceAP(t *testing.T) {
	// Path from service test dir up to conversations/
	root := filepath.Join("..", "..", "..", "conversations")

	cases := []scriptCase{
		{
			name:         "Bowman",
			relPath:      filepath.Join("npc", "npc_1012100.json"),
			advanceState: "firstJobAdvance",
			targets:      []expectedTarget{{"dexterity", 25}},
			bannedStats:  []string{"dexterity"},
		},
		{
			name:         "Warrior",
			relPath:      filepath.Join("npc", "npc_1022000.json"),
			advanceState: "firstJobAdvance",
			targets:      []expectedTarget{{"strength", 35}},
			bannedStats:  []string{"strength"},
		},
		{
			name:         "Magician",
			relPath:      filepath.Join("npc", "npc_1032001.json"),
			advanceState: "performFirstJobAdvancement",
			targets:      []expectedTarget{{"intelligence", 20}},
			bannedStats:  []string{"intelligence"},
		},
		{
			name:         "Thief",
			relPath:      filepath.Join("npc", "npc_1052001.json"),
			advanceState: "firstJobAdvance",
			targets:      []expectedTarget{{"dexterity", 25}},
			bannedStats:  []string{"dexterity"},
		},
		{
			name:         "Pirate",
			relPath:      filepath.Join("npc", "npc_1090000.json"),
			advanceState: "firstJobAdvance",
			targets:      []expectedTarget{{"dexterity", 20}},
			bannedStats:  []string{"dexterity"},
		},
		{
			name:         "Dawn Warrior",
			relPath:      filepath.Join("quests", "quest_20101.json"),
			advanceState: "performJobChange",
			targets:      []expectedTarget{{"strength", 35}},
			bannedStats:  []string{"str"},
		},
		{
			name:         "Blaze Wizard",
			relPath:      filepath.Join("quests", "quest_20102.json"),
			advanceState: "performJobChange",
			targets:      []expectedTarget{{"intelligence", 20}},
			bannedStats:  []string{"int"},
		},
		{
			name:         "Wind Archer",
			relPath:      filepath.Join("quests", "quest_20103.json"),
			advanceState: "performJobChange",
			targets:      []expectedTarget{{"dexterity", 25}},
			bannedStats:  []string{"dex"},
		},
		{
			name:         "Night Walker",
			relPath:      filepath.Join("quests", "quest_20104.json"),
			advanceState: "performJobChange",
			targets:      []expectedTarget{{"luck", 25}},
			bannedStats:  []string{"luk"},
		},
		{
			name:         "Thunder Breaker",
			relPath:      filepath.Join("quests", "quest_20105.json"),
			advanceState: "performJobChange",
			targets:      []expectedTarget{{"strength", 20}, {"dexterity", 20}},
			bannedStats:  []string{"str", "dex"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, tc.relPath))
			if err != nil {
				t.Fatalf("read %s: %v", tc.relPath, err)
			}
			var doc map[string]any
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatalf("parse %s: %v", tc.relPath, err)
			}

			asString := string(raw)
			for _, banned := range tc.bannedStats {
				banPatterns := []string{
					`"type": "` + banned + `"`,
					`"referenceId": "` + banned + `"`,
				}
				for _, pat := range banPatterns {
					if strings.Contains(asString, pat) {
						t.Errorf("%s: forbidden stat gate pattern remains: %q", tc.relPath, pat)
					}
				}
			}

			if strings.Contains(asString, `"reset_stats"`) && opsOfStateContain(t, doc, tc.advanceState, "reset_stats") {
				t.Errorf("%s: reset_stats must not appear in %q advancement operations", tc.relPath, tc.advanceState)
			}

			ops := collectOps(t, doc, tc.advanceState)
			rebalanceIdx := -1
			changeJobIdx := -1
			for i, op := range ops {
				t := stringField(op, "type")
				if t == "rebalance_ap" && rebalanceIdx < 0 {
					rebalanceIdx = i
				}
				if t == "change_job" && changeJobIdx < 0 {
					changeJobIdx = i
				}
			}
			if rebalanceIdx < 0 {
				t.Fatalf("%s: no rebalance_ap in %q operations", tc.relPath, tc.advanceState)
			}
			if changeJobIdx < 0 {
				t.Fatalf("%s: no change_job in %q operations", tc.relPath, tc.advanceState)
			}
			if rebalanceIdx >= changeJobIdx {
				t.Errorf("%s: rebalance_ap (at %d) must precede change_job (at %d)", tc.relPath, rebalanceIdx, changeJobIdx)
			}

			rebalance := ops[rebalanceIdx]
			params, _ := rebalance["params"].(map[string]any)
			targetsStr, _ := params["targets"].(string)
			var gotTargets []map[string]any
			if err := json.Unmarshal([]byte(targetsStr), &gotTargets); err != nil {
				t.Fatalf("%s: cannot parse targets JSON %q: %v", tc.relPath, targetsStr, err)
			}
			if len(gotTargets) != len(tc.targets) {
				t.Fatalf("%s: expected %d targets, got %d", tc.relPath, len(tc.targets), len(gotTargets))
			}
			for i, want := range tc.targets {
				gotStat, _ := gotTargets[i]["stat"].(string)
				gotFloor := toInt(gotTargets[i]["floor"])
				if gotStat != want.Stat || gotFloor != want.Floor {
					t.Errorf("%s: target[%d]: got {%s,%d}, want {%s,%d}",
						tc.relPath, i, gotStat, gotFloor, want.Stat, want.Floor)
				}
			}
		})
	}
}

// helpers
func collectOps(t *testing.T, doc map[string]any, stateId string) []map[string]any {
	t.Helper()
	// Supports both NPC shape ({"states":[…]}) and quest shape
	// ({"startStateMachine":{…},"endStateMachine":{"states":[…]}}).
	if states, ok := doc["states"].([]any); ok {
		if ops := opsFromStates(states, stateId); ops != nil {
			return ops
		}
	}
	for _, key := range []string{"startStateMachine", "endStateMachine"} {
		if sm, ok := doc[key].(map[string]any); ok {
			if states, ok := sm["states"].([]any); ok {
				if ops := opsFromStates(states, stateId); ops != nil {
					return ops
				}
			}
		}
	}
	t.Fatalf("no state %q found", stateId)
	return nil
}

func opsFromStates(states []any, stateId string) []map[string]any {
	for _, s := range states {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		id, _ := sm["id"].(string)
		if id != stateId {
			continue
		}
		ga, _ := sm["genericAction"].(map[string]any)
		if ga == nil {
			return nil
		}
		opsRaw, _ := ga["operations"].([]any)
		out := make([]map[string]any, 0, len(opsRaw))
		for _, op := range opsRaw {
			if m, ok := op.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	}
	return nil
}

func opsOfStateContain(t *testing.T, doc map[string]any, stateId, opType string) bool {
	t.Helper()
	for _, op := range collectOps(t, doc, stateId) {
		if stringField(op, "type") == opType {
			return true
		}
	}
	return false
}

func stringField(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case string:
		// unlikely but cheap to support
		var i int
		_, _ = json.Unmarshal([]byte(x), &i)
		return i
	}
	return 0
}
```

- [ ] **Step 2: Run the test**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run TestFirstJobScriptsUseRebalanceAP -v`
Expected: PASS — all ten subtests. Any failure indicates one of the JSON edits in Tasks 12-21 was incorrect or drifted.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/firstjob_scripts_test.go
git commit -m "test(atlas-npc-conversations): guard first-job scripts use rebalance_ap"
```

---

## Task 23: Update the NPC conversation conversion spec

**Files:**
- Modify: `docs/npc_conversation_conversion_spec.md`

- [ ] **Step 1: Document `rebalance_ap` in the common operations list**

In `docs/npc_conversation_conversion_spec.md`, find the `Common Operation Types` list (around line 439). Insert these two bullets, in a logical spot alongside other character-state operations (right after `change_skin` at line 456):

```markdown
- `rebalance_ap` - Redistribute primary stats during first-job advancement (params: `targets` — JSON-encoded array of `{"stat":<name>,"floor":<int>}`). Zeroes STR/DEX/INT/LUK to 4, raises each target stat to its floor, returns the reclaimed surplus to unallocated AP. HP/MP untouched. Must precede `change_job` in the operation sequence so stat broadcasts land before the job-change broadcast. Valid `stat` values: `"strength"`, `"dexterity"`, `"intelligence"`, `"luck"`. Duplicate stats are rejected. Single target example: `{"operation": "rebalance_ap", "params": {"targets": "[{\"stat\":\"dexterity\",\"floor\":20}]"}}` (Explorer Pirate). Multi-target example: `"targets": "[{\"stat\":\"strength\",\"floor\":20},{\"stat\":\"dexterity\",\"floor\":20}]"` (Thunder Breaker).
- `reset_stats` - Reset STR/DEX/INT/LUK to 4 and return the reclaimed surplus to unallocated AP (params: none). Retained for GM tools and non-advancement flows; **for first-job advancement scripts, use `rebalance_ap` instead**.
```

- [ ] **Step 2: Add a first-job advancement guidance section**

Immediately after the operations list (after the `Local Operations:` block that ends around line 479), insert a new section:

```markdown
#### First-Job Advancement Guidance

First-job advancement (both NPC and quest scripts) must:

1. Use `rebalance_ap` to set the class-floor stats and return reclaimed AP to the unallocated pool. Place it **before** `change_job` in the operations sequence.
2. **Not** encode stat-minimum conditions (`strength >= X`, `dexterity >= Y`, `{"type":"stat","referenceId":"str"}`, etc.) as advancement gates. On a vanilla v83 client, beginner auto-allocation puts everything in STR and these gates are unsatisfiable. The server rebalances AP at advancement time; the gate is unnecessary.
3. Retain the level check (level 10 for most classes; level 8 for Magician). That check is legitimate and unchanged.
4. Not use `reset_stats` — `rebalance_ap` supersedes it for this flow.

Reference scripts already updated to this pattern:

- Explorer: `npc_1012100.json`, `npc_1022000.json`, `npc_1032001.json`, `npc_1052001.json`, `npc_1090000.json`
- Cygnus: `quest_20101.json` through `quest_20105.json`
```

- [ ] **Step 3: Verify rendering**

Run: `grep -n "rebalance_ap\|First-Job Advancement Guidance" docs/npc_conversation_conversion_spec.md`
Expected: new entries appear in the list and the guidance section exists.

- [ ] **Step 4: Commit**

```bash
git add docs/npc_conversation_conversion_spec.md
git commit -m "docs(npc-conversion): document rebalance_ap and reset_stats + first-job guidance"
```

---

## Task 24: Full-suite build & test pass

**Files:** (no edits; verification only)

- [ ] **Step 1: Build and test atlas-character**

Run: `cd services/atlas-character/atlas.com/character && go build ./... && go test ./...`
Expected: both succeed with no regressions.

- [ ] **Step 2: Build and test atlas-npc-conversations**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go build ./... && go test ./...`
Expected: both succeed with no regressions.

- [ ] **Step 3: Build and test atlas-saga-orchestrator**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./...`
Expected: both succeed with no regressions.

- [ ] **Step 4: Build and test libs/atlas-saga**

Run: `cd libs/atlas-saga && go build ./... && go test ./...`
Expected: both succeed with no regressions.

- [ ] **Step 5: If any service has a Dockerfile that copies shared libs, build it to confirm**

Run: `docker build -f services/atlas-character/Dockerfile . 2>&1 | tail -5` (from repo root). Repeat for atlas-npc-conversations and atlas-saga-orchestrator.
Expected: build completes. If there is no Dockerfile for a service, skip it.

- [ ] **Step 6: Commit a build-verification marker only if anything needed fixing**

Only commit if the task resulted in an additional edit; otherwise skip.

---

## Task 25: Documented manual end-to-end test plan

**Files:**
- Create: `docs/tasks/task-020-first-job-ap-rebalance/manual-test-plan.md`

- [ ] **Step 1: Write the test plan document**

Create `docs/tasks/task-020-first-job-ap-rebalance/manual-test-plan.md`:

```markdown
# First-Job AP Rebalance — Manual Test Plan

Exercise each row against a running dev deployment after implementation lands. Each row creates a Level-10 beginner on a vanilla v83 client, lets auto-allocation run to completion (expected starting stats STR 53 / DEX 9 / INT 4 / LUK 4, unallocated AP 0), then attempts first-job advancement.

| # | Class | NPC / Quest | Expected post-advancement | Acceptance criterion |
|---|---|---|---|---|
| 1 | Pirate | Kyrin (NPC 1090000) | STR 4, DEX 20, INT 4, LUK 4, unallocated 38 | PRD §10.7 video match |
| 2 | Warrior | Dances with Balrog (NPC 1022000) | STR 35, DEX 4, INT 4, LUK 4, unallocated 23 | PRD §10.2 surplus-return boundary |
| 3 | Bowman | Athena Pierce (NPC 1012100) | STR 4, DEX 25, INT 4, LUK 4, unallocated 33 | Representative DEX-25 class |
| 4 | Magician | Grendel (NPC 1032001) | STR 4, DEX 4, INT 20, LUK 4, unallocated 38 | INT path exercised |
| 5 | Thunder Breaker | quest_20105 (Cygnus) | STR 20, DEX 20, INT 4, LUK 4, unallocated 22 | Multi-target `rebalance_ap` |

For each row, also verify:

- atlas-character logs contain an info line of the form `Rebalanced character [N] AP. Before STR=... → after STR=... targets=[...]`.
- atlas-character emits exactly one `STAT_CHANGED` event containing `updates` with all five stat types (`AVAILABLE_AP`, `STRENGTH`, `DEXTERITY`, `INTELLIGENCE`, `LUCK`).
- atlas-character emits exactly one `JOB_CHANGED` event *after* the `STAT_CHANGED` (ordering observable in Kafka topic offsets).
- Character row in DB has the expected STR/DEX/INT/LUK/AP values.
- No regression: pre-existing 2nd/3rd/4th job advancements are unaffected.

Open Question 3 (client auto-opens stat window): observe client behavior on each run. If the stat window does not auto-open, file a follow-up task per PRD §9 Q3.
```

- [ ] **Step 2: Commit**

```bash
git add docs/tasks/task-020-first-job-ap-rebalance/manual-test-plan.md
git commit -m "docs(task-020): document manual end-to-end test plan for first-job rebalance"
```

---

## Spec Coverage Summary

| PRD / design requirement | Task(s) |
|---|---|
| PRD §4.1 algorithm, Warrior surplus case | Task 4 (computeRebalance tests) |
| PRD §4.2 HP/MP untouched | Task 5 (TestRebalanceAP_PersistsAndEmits assertion) |
| PRD §4.3 `rebalance_ap` operation schema | Task 3 (dispatcher) + Task 23 (docs) |
| PRD §4.4 script edits (ordering, gate removal) | Tasks 12-21 + Task 22 (regression guard) |
| PRD §4.5 affected NPCs/quests | Tasks 12-21 |
| PRD §4.6 conversion spec updates | Task 23 |
| PRD §4.7 stat-change event flow | Task 5 (multi-stat emission) + Task 25 (observational) |
| PRD §5 API surface (saga-only, no REST) | Tasks 1, 8-11 |
| PRD §6 no schema changes | Honored across all tasks (no migrations) |
| PRD §8 observability (info log) | Task 5 (info log in RebalanceAP) |
| PRD §10.1 Pirate reference test | Task 4 (table row) + Task 5 (integration) + Task 25 manual |
| PRD §10.2 Warrior boundary | Task 4 |
| PRD §10.3 HP/MP bit-identical | Task 5 |
| PRD §10.4 all scripts updated | Tasks 12-21, Task 22 enforces |
| PRD §10.5 spec updated | Task 23 |
| PRD §10.6 end-to-end test | Task 25 |
| PRD §10.7 video match | Task 4 table + Task 25 row 1 |
| PRD §10.8 no regressions | Task 24 (full-suite builds/tests) |
| Design §2 saga-orchestrated transport | Tasks 1, 8, 9, 10, 11 |
| Design §3.1 computeRebalance | Task 4 |
| Design §3.2/3.3 RebalanceAP / RebalanceAPAndEmit | Task 5 |
| Design §4.1 multi-target shape | Tasks 1 (payload), 3 (dispatcher), 4 (helper), 21 (Thunder Breaker) |
| Design §4.2 dispatcher validation | Task 3 tests (empty / duplicate / invalid stat) |
| Design §4.3 script edit pattern | Tasks 12-21 |
| Design §5 spec updates | Task 23 |
| Design §7.4 regression guard test | Task 22 |
