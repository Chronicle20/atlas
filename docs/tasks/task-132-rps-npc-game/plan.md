# Rock-Paper-Scissors NPC Game — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the Henesys game-park Rock-Paper-Scissors minigame (NPC `9000019`): a meso-bet entry, server-authoritative RPS rounds, and a collect-or-continue reward ladder — spanning a new `atlas-rps` microservice, two dispatcher-family packet codecs, channel wiring, an NPC-conversation entry point, a saga action, and a tenant-config reward ladder.

**Architecture:** `atlas-rps` owns ephemeral session state (Redis TTL registry), the server RNG/adjudication, and the ladder logic. The NPC-conversation `9000019` state machine builds an **entry saga** `[AwardMesos(−entryCost), StartRPSGame]`; `StartRPSGame` is dispatched by `atlas-saga-orchestrator` **synchronously over REST** to `atlas-rps` (the gachapon precedent — self-completing step). The round loop (select→result / continue / collect / quit) runs directly channel↔atlas-rps↔channel over Kafka. On **collect**, `atlas-rps` submits a **payout saga** `[AwardMesos(+meso)?, AwardAsset(item,qty)?]`. `atlas-channel` decodes serverbound `RPS_ACTION` into commands and encodes `atlas-rps` events into clientbound `RPS_GAME` frames. All economy mutations stay inside their owning services via the shared saga vocabulary.

**Tech Stack:** Go 1.25.x microservices; `libs/atlas-redis` (TTL registry + Set), `libs/atlas-kafka`, `libs/atlas-saga`, `libs/atlas-packet` (dispatcher families), `libs/atlas-constants`, `libs/atlas-rest` (JSON:API via api2go/jsonapi); tenant seed templates in `atlas-configurations`; NPC-conversation JSON seeds.

## Global Constraints

- **Version set (design D1):** `gms_v83, gms_v84, gms_v87, gms_v95, jms_v185`. **v92 is parked** (no v92 IDB, `gms_92_1` template is a stub with no `operations` table) — document the parked follow-up, do not implement or seed v92. Never invent v92 bytes.
- **RPS_GAME (clientbound, `CRPSGameDlg::OnPacket`) opcodes:** v83 `0x138`, v84 `0x13F`, v87 `0x149`, v95 `0x173`, jms185 `0x151` (verified against `docs/packets/audits/STATUS.md:418`).
- **RPS_ACTION (serverbound, `CRPSGameDlg::OnBt*`/`SendSelection`/`Update`) opcodes:** v83 `0x088`, v84 `0x08C`, v87 `0x090`, v95 `0x0A0`, jms185 `0x08B` (verified against `docs/packets/audits/STATUS.md:641`).
- **Verify, don't invent (CLAUDE.md):** per-version mode bytes, packet frame field layouts, and concrete reward-ladder item ids are IDA/WZ-derived during execution (§ IDA-gated tasks). Do NOT write a byte fixture, a mode-byte table, or an item-id ladder from memory. A cell/behavior is claimed done only when its IDA/WZ evidence is pinned.
- **Dispatcher-family invariants (`docs/packets/DISPATCHER_FAMILY.md`):** one discrete struct per mode; `Encode` writes the mode byte + full arm body; every constructor takes `mode byte`; every body func resolves via `WithResolvedCode("operations", FIXED_KEY, func(mode byte)…)` — zero `mode: 0x` literals, zero `func(_ byte)`, zero caller-supplied op/code/mode/key selector. RPS is **not** in `dispatcher-lint-baseline.yaml`, so `dispatcher-lint` enforces it from day one.
- **Economy authority:** meso only via `atlas-character` (through `AwardMesos` saga action); items only via `atlas-inventory` (through `AwardAsset`). `atlas-rps`/`atlas-channel` never mutate economy directly.
- **Multi-tenancy:** all state/config tenant-scoped via `tenant.MustFromContext(ctx)`; consumers use `consumer.SpanHeaderParser, consumer.TenantHeaderParser`; Redis registries key per-tenant. Tenant-safe: no bare-`characterId` SQL PK (we use Redis, so N/A, but the session registry keys on `(tenant, characterId)`).
- **Redis discipline:** all Redis access via `libs/atlas-redis` lib types (`atlas.TTLRegistry`, `atlas.Set`) — `tools/redis-key-guard.sh` bans raw keyed go-redis outside `libs/atlas-redis`.
- **Shared types (DOM-21):** use `libs/atlas-constants` types (`world.Id`, `channel.Id`, `character` id `uint32`, `item.Id`) — no new numeric aliases.
- **No stubs (CLAUDE.md):** no `// TODO`, no 501s, no bodyless-but-declared handlers in landed commits. (The chalkboards template you clone has two stray `// TODO`s at `chalkboard/processor.go:53-54` — do NOT copy them.)
- **Build gate for every changed module (CLAUDE.md §Build & Verification):** `go test -race ./...`, `go vet ./...`, `go build ./...` clean; **`docker buildx bake atlas-<svc>` from the worktree root for every service whose `go.mod` was touched**; `tools/redis-key-guard.sh` clean from repo root. All green before "done".
- **Worktree:** all work in `.worktrees/task-132-rps-npc-game` on branch `task-132-rps-npc-game`. Every implementer subagent `cd`s into the worktree first and verifies branch after each commit.

---

## Milestone A — libs/atlas-saga: `StartRPSGame` action

### Task 1: Add `StartRPSGame` action + payload + unmarshal

**Files:**
- Modify: `libs/atlas-saga/model.go` (action consts block, near `SelectGachaponReward` ~line 143)
- Modify: `libs/atlas-saga/payloads.go` (after `SelectGachaponRewardPayload` ~line 667)
- Modify: `libs/atlas-saga/unmarshal.go` (switch in `Step[T].UnmarshalJSON`, near the `SelectGachaponReward` case ~line 432)
- Test: `libs/atlas-saga/unmarshal_test.go`

**Interfaces:**
- Produces: `saga.StartRPSGame Action = "start_rps_game"`; `saga.StartRPSGamePayload{ CharacterId uint32; WorldId world.Id; ChannelId channel.Id; NpcId uint32 }`. Consumed by atlas-npc-conversations (Task 22), atlas-saga-orchestrator (Task 13).

- [ ] **Step 1: Write the failing unmarshal test** in `libs/atlas-saga/unmarshal_test.go` (mirror `TestUnmarshalEvolvePetStep` at ~line 250):

```go
func TestUnmarshalStartRPSGameStep(t *testing.T) {
	raw := `{
		"stepId": "start_rps_game-1",
		"status": "pending",
		"action": "start_rps_game",
		"payload": { "characterId": 100, "worldId": 0, "channelId": 1, "npcId": 9000019 },
		"createdAt": "2026-07-04T00:00:00Z",
		"updatedAt": "2026-07-04T00:00:00Z"
	}`
	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != StartRPSGame {
		t.Fatalf("expected action StartRPSGame, got %s", step.Action)
	}
	p, ok := step.Payload.(StartRPSGamePayload)
	if !ok {
		t.Fatalf("expected StartRPSGamePayload, got %T", step.Payload)
	}
	if p.CharacterId != 100 || p.NpcId != 9000019 {
		t.Errorf("payload mismatch: %+v", p)
	}
}
```

- [ ] **Step 2: Run it, verify it fails** — `cd libs/atlas-saga && go test ./... -run TestUnmarshalStartRPSGameStep`. Expected: FAIL (undefined `StartRPSGame`, `StartRPSGamePayload`).

- [ ] **Step 3: Add the action const** in `libs/atlas-saga/model.go` near the gachapon actions:

```go
	// RPS actions
	StartRPSGame Action = "start_rps_game"
```

- [ ] **Step 4: Add the payload struct** in `libs/atlas-saga/payloads.go` (import `world`, `channel` from atlas-constants are already present in this file — confirm the import aliases match the existing `AwardMesosPayload`):

```go
// StartRPSGamePayload represents the payload required to open an RPS game session for a character.
type StartRPSGamePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId the game opens for
	WorldId     world.Id   `json:"worldId"`     // WorldId of the session
	ChannelId   channel.Id `json:"channelId"`   // ChannelId for the client dialog routing
	NpcId       uint32     `json:"npcId"`       // Entry NPC (9000019)
}
```

- [ ] **Step 5: Add the unmarshal case** in `libs/atlas-saga/unmarshal.go` next to the `SelectGachaponReward` case:

```go
	case StartRPSGame:
		var payload StartRPSGamePayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
```

- [ ] **Step 6: Run the test, verify it passes** — `go test ./... -run TestUnmarshalStartRPSGameStep`. Expected: PASS.

- [ ] **Step 7: Full module gate** — `go test -race ./... && go vet ./...` in `libs/atlas-saga`. Expected: clean.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-saga/model.go libs/atlas-saga/payloads.go libs/atlas-saga/unmarshal.go libs/atlas-saga/unmarshal_test.go
git commit -m "feat(saga): add StartRPSGame action and payload (task-132)"
```

---

## Milestone B — `atlas-rps` service

New Go module at `services/atlas-rps/atlas.com/rps/`, module name `atlas-rps`. Clone the spine from `atlas-expressions` (TTL registry + Kafka + sweeper) and the REST layer from `atlas-chalkboards`. Import shared libs as `github.com/Chronicle20/atlas/libs/atlas-<name>/...` with `replace` directives at relative depth `../../../../libs/atlas-<name>`.

### Task 2: Service skeleton + registration (compiles, boots, bakes)

**Files:**
- Create: `services/atlas-rps/atlas.com/rps/go.mod` (module `atlas-rps`, `go 1.25.x` matching expressions; require + replace the libs actually imported: `atlas-constants, atlas-kafka, atlas-model, atlas-redis, atlas-rest, atlas-service, atlas-tenant, atlas-tracing, atlas-saga`)
- Create: `services/atlas-rps/atlas.com/rps/main.go`
- Create: `services/atlas-rps/atlas.com/rps/logger/init.go` (copy verbatim from `services/atlas-expressions/atlas.com/expressions/logger/init.go`)
- Create: `services/atlas-rps/atlas.com/rps/tasks/task.go` (copy verbatim from `services/atlas-expressions/atlas.com/expressions/tasks/task.go`)
- Create: `services/atlas-rps/atlas.com/rps/rest/server.go` (the `jsonapi.ServerInformation` impl — copy from `services/atlas-chalkboards/atlas.com/chalkboards/` `GetServer()` provider)
- Modify: `go.work` (repo root) — add `./services/atlas-rps/atlas.com/rps` under `use (`, alphabetically after `./services/atlas-renders/...` line
- Modify: `.github/config/services.json` — add the go-service entry
- Modify: `docker-bake.hcl` — add `"atlas-rps"` to the hand-synced `go_services` list (~line 35)
- Create: `deploy/k8s/base/atlas-rps.yaml`
- Modify: `deploy/k8s/base/kustomization.yaml` — add `- atlas-rps.yaml` to `resources`
- Modify: `deploy/k8s/overlays/pr/kustomization.yaml` — add the image entry (`newTag: latest`)
- Modify: `deploy/k8s/overlays/main/kustomization.yaml` — add the image entry (`newTag: latest`)

**Interfaces:**
- Produces: a bootable REST+Kafka service. `serviceName = "atlas-rps"`, `consumerGroupId = consumergroup.Resolve("RPS Service")`, REST base path `/api/`, `REST_PORT` env, port `8080`.

- [ ] **Step 1:** Create `go.mod` cloning `services/atlas-expressions/atlas.com/expressions/go.mod`; module `atlas-rps`; trim replace lines to only imported libs plus add `atlas-saga` (require + replace `=> ../../../../libs/atlas-saga`).

- [ ] **Step 2:** Create `main.go` (adapt the expressions main; REST resource + sweeper both present — the sweeper's concrete task and the game consumers are wired in later tasks, so at this step register only the REST server and the debug handler; leave a clearly-named `// consumers registered in Task 10` is NOT allowed — instead land this task with the consumer/sweeper lines already calling the real (empty-for-now) InitConsumers created in Task 8/10). To keep this task independently compilable, land it as REST-only and wire consumers/sweeper in their own tasks by editing main.go then. Minimal bootable main:

```go
package main

import (
	"atlas-rps/logger"
	"os"

	"github.com/Chronicle20/atlas/libs/atlas-service/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"github.com/Chronicle20/atlas/libs/atlas-service/server"
	"github.com/Chronicle20/atlas/libs/atlas-tracing"
)

const serviceName = "atlas-rps"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}
	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath("/api/").
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		Run()

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
```

(Import paths above are illustrative — resolve the exact package paths against the expressions/chalkboards `main.go` you clone; do not guess a path that fails to compile.)

- [ ] **Step 3:** Create `logger/init.go`, `tasks/task.go`, `rest/server.go` by copying the named source files with the package path renamed to `atlas-rps/...`.

- [ ] **Step 4:** Add the `go.work` line, the `services.json` entry, and the `docker-bake.hcl` `go_services` entry:

```json
{
  "name": "atlas-rps",
  "type": "go-service",
  "path": "services/atlas-rps",
  "module_path": "services/atlas-rps/atlas.com/rps",
  "docker_image": "ghcr.io/chronicle20/atlas-rps/atlas-rps",
  "docker_context": "."
}
```

- [ ] **Step 5:** Create `deploy/k8s/base/atlas-rps.yaml` (REST+Kafka service — Deployment **with** a Service, mirroring `atlas-chalkboards.yaml`; readiness probe on `/api/readyz` per memory `bug_readiness_probe_path_under_api_basepath` if the server exposes readiness — check whether chalkboards/expressions declare one; they do not, so match them and omit unless the base server auto-mounts `/api/readyz`):

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atlas-rps
spec:
  replicas: 2
  selector:
    matchLabels:
      app: atlas-rps
  template:
    metadata:
      labels:
        app: atlas-rps
    spec:
      containers:
      - name: rps
        image: ghcr.io/chronicle20/atlas-rps/atlas-rps:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: atlas-env
        env:
        - name: LOG_LEVEL
          value: "debug"
---
apiVersion: v1
kind: Service
metadata:
  name: atlas-rps
spec:
  selector:
    app: atlas-rps
  ports:
  - protocol: TCP
    port: 8080
```

- [ ] **Step 6:** Add `- atlas-rps.yaml` to `deploy/k8s/base/kustomization.yaml` resources (alphabetical, after `atlas-renders`/before `atlas-saga-orchestrator`), and add the image entry to both `overlays/pr/kustomization.yaml` (`newTag: latest`) and `overlays/main/kustomization.yaml` (`newTag: latest`):

```yaml
  - name: ghcr.io/chronicle20/atlas-rps/atlas-rps
    newTag: latest
```

- [ ] **Step 7: Verify build + bake** — from the worktree root:

```bash
cd services/atlas-rps/atlas.com/rps && go build ./... && go vet ./...
cd - && docker buildx bake atlas-rps
```

Expected: build clean; bake succeeds (proves services.json + docker-bake + Dockerfile SERVICE param resolve). Also run `kustomize build deploy/k8s/overlays/pr >/dev/null` to prove the manifests parse.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-rps go.work .github/config/services.json docker-bake.hcl deploy/k8s/base/atlas-rps.yaml deploy/k8s/base/kustomization.yaml deploy/k8s/overlays/pr/kustomization.yaml deploy/k8s/overlays/main/kustomization.yaml
git commit -m "feat(rps): scaffold atlas-rps service + registration (task-132)"
```

### Task 3: Session model + Builder

**Files:**
- Create: `services/atlas-rps/atlas.com/rps/game/model.go`
- Create: `services/atlas-rps/atlas.com/rps/game/builder.go`
- Test: `services/atlas-rps/atlas.com/rps/game/model_test.go`

**Interfaces:**
- Produces: `game.Model` (immutable, private fields + getters + custom `MarshalJSON`/`UnmarshalJSON` since it is Redis-serialized); `game.Status` enum (`StatusOpen`, `StatusAwaitingSelect`, `StatusAwaitingDecision`, `StatusEnded`); `game.Throw` enum (`ThrowRock`, `ThrowPaper`, `ThrowScissors`); `game.NewModelBuilder(t tenant.Model) *ModelBuilder` with `SetCharacterId/SetWorldId/SetChannelId/SetNpcId/SetRung/SetStatus/SetLastThrow/SetCreatedAt/SetUpdatedAt`, `Build() (Model, error)`, `MustBuild() Model`, and `CloneModelBuilder(m Model) *ModelBuilder`. Getters: `Tenant() tenant.Model`, `CharacterId() uint32`, `WorldId() world.Id`, `ChannelId() channel.Id`, `NpcId() uint32`, `Rung() int`, `Status() Status`, `LastThrow() Throw`, `CreatedAt()/UpdatedAt() time.Time`.

- [ ] **Step 1: Write failing test** `model_test.go`:

```go
func TestModelBuilderRoundTripsThroughJSON(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	m := game.NewModelBuilder(ten).
		SetCharacterId(100).SetWorldId(0).SetChannelId(1).SetNpcId(9000019).
		SetRung(2).SetStatus(game.StatusAwaitingDecision).MustBuild()

	b, err := json.Marshal(m)
	if err != nil { t.Fatalf("marshal: %v", err) }
	var out game.Model
	if err := json.Unmarshal(b, &out); err != nil { t.Fatalf("unmarshal: %v", err) }

	if out.CharacterId() != 100 || out.Rung() != 2 || out.Status() != game.StatusAwaitingDecision {
		t.Errorf("round-trip mismatch: %+v", out)
	}
}

func TestModelBuilderRejectsZeroCharacter(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	if _, err := game.NewModelBuilder(ten).SetCharacterId(0).Build(); err == nil {
		t.Fatal("expected error for characterId 0")
	}
}
```

- [ ] **Step 2: Run, verify fail** — `go test ./game/ -run TestModelBuilder`. Expected: FAIL (package/type undefined).

- [ ] **Step 3: Implement `model.go`** — private-field struct with getters and `MarshalJSON`/`UnmarshalJSON` (anon struct with exported fields — mirror `services/atlas-expressions/atlas.com/expressions/expression/model.go`). Define `Status` as a string enum and `Throw` as a `byte` enum. Use `world.Id`, `channel.Id` from atlas-constants.

- [ ] **Step 4: Implement `builder.go`** — `NewModelBuilder(t)` seeds `createdAt = time.Now()`, chained setters, `Build()` validates `tenant` non-nil + `characterId != 0` and stamps `updatedAt`, `MustBuild()` panics on error, `CloneModelBuilder(m)` copies fields.

- [ ] **Step 5: Run, verify pass** — `go test ./game/ -run TestModelBuilder`. Expected: PASS.

- [ ] **Step 6: Commit** — `git add services/atlas-rps/atlas.com/rps/game && git commit -m "feat(rps): session model + builder (task-132)"`

### Task 4: Redis TTL session registry + tenant tracking

**Files:**
- Create: `services/atlas-rps/atlas.com/rps/game/registry.go`
- Test: `services/atlas-rps/atlas.com/rps/game/registry_test.go` (use `alicebob/miniredis/v2`, already a test dep in expressions)

**Interfaces:**
- Produces: `game.InitRegistry(client *goredis.Client)`, `game.GetRegistry() *Registry`, and methods `Put(ctx, m Model)`, `Get(ctx, characterId uint32) (Model, bool)`, `Remove(ctx, characterId uint32)`, `PopExpired(ctx) []Model` (fans out over tracked tenants). Namespace `"rps"`, tenant Set `"rps:_tenants"`, default TTL e.g. `5 * time.Minute` (RPS games abandon after inactivity). Every access via `atlas.TTLRegistry`/`atlas.Set` — no raw go-redis keyed calls.

- [ ] **Step 1: Write failing test** — put a model, get it back; put one, `SetNowFunc` to advance past TTL, assert `PopExpired` returns it and removes it. Mirror the expressions registry test structure with miniredis.

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement `registry.go`** cloning `services/atlas-expressions/atlas.com/expressions/expression/registry.go`: `atlas.NewTTLRegistry[uint32, Model](client, "rps", keyFn, defaultTTL)` + `atlas.NewSet(client, "rps:_tenants")`; `trackTenant`/`getTrackedTenants` via JSON-marshal of `tenant.Model` into the Set; `PopExpired` loops tracked tenants calling `reg.PopExpired(ctx, t)`.

- [ ] **Step 4: Run, verify pass.**

- [ ] **Step 5: Redis guard** — from repo root `GOWORK=off tools/redis-key-guard.sh` (per memory `reference_rediskeyguard_invariant`) — expect clean (all access through lib types).

- [ ] **Step 6: Commit** — `git commit -m "feat(rps): redis TTL session registry (task-132)"`

### Task 5: Adjudication (server RNG + RPS rules)

**Files:**
- Create: `services/atlas-rps/atlas.com/rps/game/adjudicate.go`
- Test: `services/atlas-rps/atlas.com/rps/game/adjudicate_test.go`

**Interfaces:**
- Produces: `type Outcome int` (`OutcomeLose`, `OutcomeTie`, `OutcomeWin` — from the *player's* perspective); `func Adjudicate(playerThrow, opponentThrow Throw) Outcome` (pure); `type ThrowSource func() Throw`; `func DefaultThrowSource() Throw` (uses `math/rand`); a way to inject the source for tests. Rules: rock>scissors, scissors>paper, paper>rock. Server authority lives here (FR-2.2).

- [ ] **Step 1: Write failing table test** covering all 9 `(player, opponent)` combinations → expected Outcome, plus a determinism test that injects a fixed `ThrowSource` and asserts the opponent throw.

```go
func TestAdjudicateAllCombinations(t *testing.T) {
	cases := []struct{ p, o game.Throw; want game.Outcome }{
		{game.ThrowRock, game.ThrowScissors, game.OutcomeWin},
		{game.ThrowRock, game.ThrowPaper, game.OutcomeLose},
		{game.ThrowRock, game.ThrowRock, game.OutcomeTie},
		{game.ThrowPaper, game.ThrowRock, game.OutcomeWin},
		{game.ThrowPaper, game.ThrowScissors, game.OutcomeLose},
		{game.ThrowPaper, game.ThrowPaper, game.OutcomeTie},
		{game.ThrowScissors, game.ThrowPaper, game.OutcomeWin},
		{game.ThrowScissors, game.ThrowRock, game.OutcomeLose},
		{game.ThrowScissors, game.ThrowScissors, game.OutcomeTie},
	}
	for _, c := range cases {
		if got := game.Adjudicate(c.p, c.o); got != c.want {
			t.Errorf("Adjudicate(%v,%v)=%v want %v", c.p, c.o, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement `adjudicate.go`** — pure `Adjudicate`, `DefaultThrowSource` over `math/rand` (seed at package init; math/rand is fine in Go service code — the Workflow-script RNG ban does not apply here).

- [ ] **Step 4: Run, verify pass.**

- [ ] **Step 5: Commit** — `git commit -m "feat(rps): RPS adjudication + server RNG (task-132)"`

### Task 6: Reward-ladder resolution

**Files:**
- Create: `services/atlas-rps/atlas.com/rps/game/ladder.go`
- Test: `services/atlas-rps/atlas.com/rps/game/ladder_test.go`

**Interfaces:**
- Produces: `type Rung struct { Rung int; ItemId item.Id; Quantity uint32; Meso uint32 }`; `type Ladder struct { EntryCostMeso uint32; Rungs []Rung }`; methods `PrizeAt(rung int) (Rung, bool)`, `MaxRung() int`, `IsMax(rung int) bool`. Rung indices are 1-based (rung 0 = fresh, no prize). Consumed by the processor (Task 9) and configuration loader (Task 7).

- [ ] **Step 1: Write failing test** — build a `Ladder` with rungs 1,2,3; assert `PrizeAt(2)` returns rung 2, `PrizeAt(0)` false, `MaxRung()==3`, `IsMax(3)` true.

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement `ladder.go`** — plain resolution over the ordered `Rungs` slice. Use `item.Id` from `libs/atlas-constants`.

- [ ] **Step 4: Run, verify pass.**

- [ ] **Step 5: Commit** — `git commit -m "feat(rps): reward-ladder resolution (task-132)"`

### Task 7: Configuration loader (read `rps-rewards` from atlas-tenants)

**Files:**
- Create: `services/atlas-rps/atlas.com/rps/configuration/rest.go` (service-local `RpsRewardRestModel` + `Extract` into `game.Ladder`)
- Create: `services/atlas-rps/atlas.com/rps/configuration/requests.go` (`GET {TENANTS}tenants/{tenantId}/configurations/rps-rewards`)
- Create: `services/atlas-rps/atlas.com/rps/configuration/processor.go` (`GetLadder(tenantId) (game.Ladder, error)`)
- Test: `services/atlas-rps/atlas.com/rps/configuration/processor_test.go` (httptest server returning a JSON:API `rps-rewards` doc)

**Interfaces:**
- Consumes: the atlas-tenants config REST surface from Task 21 (resource `rps-rewards`).
- Produces: `configuration.NewProcessor(l, ctx).GetLadder(tenantId uuid.UUID) (game.Ladder, error)`. Mirror `services/atlas-transports/atlas.com/transports/transport/config/{requests.go,rest.go,processor.go}` (the downstream config-read precedent): `requests.RootUrl("TENANTS")`, `requests.GetRequest[[]RpsRewardRestModel](url)`, `requests.SliceProvider[...]`.

- [ ] **Step 1: Write failing test** — spin an `httptest` server returning `{"data":{"id":"rps-rewards","type":"rps-rewards","attributes":{"entryCostMeso":1000,"ladder":[{"rung":1,...}]}}}`; point `TENANTS` env at it; assert `GetLadder` returns `EntryCostMeso==1000` and the parsed rungs.

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement** the three files mirroring the transports config precedent; `Extract` maps `RpsRewardRestModel` → `game.Ladder`.

- [ ] **Step 4: Run, verify pass.**

- [ ] **Step 5: Commit** — `git commit -m "feat(rps): rps-rewards config loader (task-132)"`

### Task 8: Kafka topics, message structs, producer, event providers

**Files:**
- Create: `services/atlas-rps/atlas.com/rps/kafka/message/message.go` (copy `Buffer`/`Emit`/`EmitWithResult` verbatim from expressions)
- Create: `services/atlas-rps/atlas.com/rps/kafka/message/rps/kafka.go` (topic env consts + Command + Event structs)
- Create: `services/atlas-rps/atlas.com/rps/kafka/producer/producer.go` (copy verbatim from expressions, rename package)
- Create: `services/atlas-rps/atlas.com/rps/game/producer.go` (event `model.Provider[[]kafka.Message]` builders)
- Create: `services/atlas-rps/atlas.com/rps/kafka/consumer/consumer.go` (copy `NewConfig`/`LookupBrokers` verbatim from expressions)
- Test: `services/atlas-rps/atlas.com/rps/kafka/message/rps/kafka_test.go` (marshal/unmarshal of Command/Event by `Type` discriminator)

**Interfaces:**
- Produces:
  - `EnvCommandTopic = "COMMAND_TOPIC_RPS"`, `EnvEventTopic = "EVENT_TOPIC_RPS"`.
  - `Command` with a `Type` discriminator: `CommandTypeSelect`, `CommandTypeContinue`, `CommandTypeCollect`, `CommandTypeQuit` (StartGame arrives via REST, not this topic — see Task 11). Body carries `CharacterId uint32`, `WorldId world.Id`, `ChannelId channel.Id`, and (for Select) `Throw byte`.
  - `Event` with `Type`: `EventTypeGameOpened`, `EventTypeRoundResult`, `EventTypeGameEnded`. `RoundResult` body: `OpponentThrow byte`, `Outcome int`, `Rung int`, `Prize {ItemId uint32; Quantity uint32; Meso uint32}`. `GameEnded` body: `Reason string` (`collected|lost|quit|disconnected`), `GrantedPrize` (optional). Partition key: `producer.CreateKey(int(characterId))`.

- [ ] **Step 1: Write failing test** — marshal a `Select` command and each event type, unmarshal, assert `Type` + body fields survive.

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement** the message/producer/consumer boilerplate (clone) + the RPS command/event structs + `game/producer.go` providers (`gameOpenedEventProvider`, `roundResultEventProvider`, `gameEndedEventProvider`).

- [ ] **Step 4: Run, verify pass.**

- [ ] **Step 5: Commit** — `git commit -m "feat(rps): kafka topics, messages, producer (task-132)"`

### Task 9: Processor (Start/Select/Continue/Collect/Quit + AndEmit)

**Files:**
- Create: `services/atlas-rps/atlas.com/rps/game/processor.go`
- Create: `services/atlas-rps/atlas.com/rps/game/mock/processor.go` (interface mock; `var _ game.Processor = (*ProcessorMock)(nil)`)
- Test: `services/atlas-rps/atlas.com/rps/game/processor_test.go`

**Interfaces:**
- Consumes: registry (Task 4), adjudicate (Task 5), ladder + configuration (Tasks 6/7), kafka message/producer (Task 8).
- Produces: `game.Processor` interface + `NewProcessor(l, ctx) Processor` with `tenant.MustFromContext(ctx)`. Methods (each buffered `Method(mb, …)` + emitting `MethodAndEmit(…)`):
  - `Start(mb, characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) (Model, error)` — dispose any stale session, create rung-0 `StatusOpen`, buffer `GameOpened`.
  - `Select(mb, characterId uint32, throw Throw) (Model, error)` — load session (must be `StatusOpen`/`StatusAwaitingSelect`); RNG opponent throw; `Adjudicate`; **win** → rung+1, `StatusAwaitingDecision`, prize=`ladder.PrizeAt(rung)`, buffer `RoundResult{win}`; **tie** → rung unchanged, `StatusAwaitingSelect`, buffer `RoundResult{tie}`; **loss** → remove session, buffer `RoundResult{lose}` + `GameEnded{lost}`.
  - `Continue(mb, characterId uint32) (Model, error)` — require `StatusAwaitingDecision`; if `ladder.IsMax(rung)` force a Collect instead; else `StatusAwaitingSelect`.
  - `Collect(mb, characterId uint32) (Model, error)` — resolve `ladder.PrizeAt(rung)`; **submit payout saga** (Task 12); remove session; buffer `GameEnded{collected, prize}`.
  - `Quit(mb, characterId uint32) (Model, error)` / `Dispose(...)` — remove session; buffer `GameEnded{quit}` (Dispose: `disconnected`, no event needed if session already gone).
- Injectable `ThrowSource` for deterministic tests (constructor variant or a settable field).

- [ ] **Step 1: Write failing test** — with an injected fixed throw source and an in-memory (miniredis) registry + a stub ladder: drive `Start → Select(win) → Continue → Select(tie) → Select(win) → Collect`; assert rung transitions, statuses, and that the emitted buffer contains the expected event sequence. Add a `Select`-loss test (session removed, `GameEnded{lost}`), a max-rung `Continue`-forces-collect test, and a `Quit`-no-payout test.

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement `processor.go`** (Interface+Impl; `…AndEmit` wraps via `message.EmitWithResult`), `mock/processor.go`.

- [ ] **Step 4: Run, verify pass** with `-race`.

- [ ] **Step 5: Commit** — `git commit -m "feat(rps): game processor state machine (task-132)"`

### Task 10: Kafka command consumer (round loop) + main.go wiring

**Files:**
- Create: `services/atlas-rps/atlas.com/rps/kafka/consumer/rps/consumer.go` (`InitConsumers` + `InitHandlers` for `COMMAND_TOPIC_RPS`, one handler per command `Type`)
- Modify: `services/atlas-rps/atlas.com/rps/main.go` (add Redis connect + `game.InitRegistry(rc)`, `cmf := consumer.GetManager()...`, `rps.InitConsumers(l)(cmf)(consumerGroupId)`, `rps.InitHandlers(...)`, and the sweeper `go tasks.Register(...)(game.NewSweepTask(...))` from Task 12)

**Interfaces:**
- Consumes: `Command` (Task 8), processor (Task 9). Header parsers `SpanHeaderParser, TenantHeaderParser` inject tenant into ctx.
- Produces: live round-loop handling — `Select/Continue/Collect/Quit` commands drive the processor's `…AndEmit` methods.

- [ ] **Step 1: Write failing test** — a handler-level test: feed a `Select` command through `handleCommand`, assert the processor is invoked and an event is emitted (use the mock producer/registry). (Mirror `services/atlas-expressions/atlas.com/expressions/kafka/consumer/expression/consumer.go` handler style.)

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement** `consumer/rps/consumer.go` — `switch c.Type` → `processor.SelectAndEmit(...)` etc. Wire into `main.go` (Redis connect, registry init, consumer registration).

- [ ] **Step 4: Run, verify pass;** then `go build ./... && docker buildx bake atlas-rps`.

- [ ] **Step 5: Commit** — `git commit -m "feat(rps): command consumer + main wiring (task-132)"`

### Task 11: REST — `POST /rps/games` (StartRPSGame entry) + `GET /rps/games/{characterId}`

**Files:**
- Create: `services/atlas-rps/atlas.com/rps/rest/handler.go` (copy `ParseCharacterId` etc. from chalkboards)
- Create: `services/atlas-rps/atlas.com/rps/game/rest.go` (`RestModel` + `GetName()="rps-games"` + `Transform`)
- Create: `services/atlas-rps/atlas.com/rps/game/resource.go` (`InitResource`: `POST /rps/games` + `GET /rps/games/{characterId}`)
- Modify: `services/atlas-rps/atlas.com/rps/main.go` (`AddRouteInitializer(game.InitResource(GetServer()))`)
- Test: `services/atlas-rps/atlas.com/rps/game/resource_test.go`

**Interfaces:**
- Produces:
  - `POST /rps/games` — JSON:API body `{data:{type:"rps-games",attributes:{characterId,worldId,channelId,npcId}}}`; handler calls `game.NewProcessor(...).StartAndEmit(...)` (creating the session + emitting `GameOpened`) and returns the created `rps-games` resource. **This is the endpoint the saga-orchestrator calls for `StartRPSGame`** (gachapon REST precedent). Uses `rest.RegisterInputHandler[RestModel]`.
  - `GET /rps/games/{characterId}` — returns the current session (`rung`, `status`, current prize) or `404` when none (PRD §5.1). Read-only.

- [ ] **Step 1: Write failing test** — `POST /rps/games` with a JSON:API envelope creates a session (assert 200 + a `rps-games` body with `status:"open"`); a second `POST` for the same character disposes-and-recreates (FR-1.4); `GET` on an unknown character returns 404.

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement** the REST model/resource/handler; POST → `StartAndEmit`, GET → `Get`. Remember the JSON:API envelope requirement (memory `bug_ui_jsonapi_envelope_required_for_input_handlers`).

- [ ] **Step 4: Run, verify pass.**

- [ ] **Step 5: Commit** — `git commit -m "feat(rps): REST start + get endpoints (task-132)"`

### Task 12: Payout saga submission + abandoned-session sweeper

**Files:**
- Create: `services/atlas-rps/atlas.com/rps/kafka/message/saga/kafka.go` (local saga command envelope for submitting a saga)
- Create: `services/atlas-rps/atlas.com/rps/saga/processor.go` (`Create(saga)` producer to the orchestrator command topic — mirror how npc-conversations submits a saga via `saga.NewProcessor(l,ctx).Create(s)`)
- Create: `services/atlas-rps/atlas.com/rps/game/task.go` (`NewSweepTask` — pops expired sessions, disposes, no payout)
- Modify: `services/atlas-rps/atlas.com/rps/game/processor.go` (`Collect` builds + submits the payout saga)
- Test: `services/atlas-rps/atlas.com/rps/saga/processor_test.go`, `services/atlas-rps/atlas.com/rps/game/task_test.go`

**Interfaces:**
- Consumes: `libs/atlas-saga` builder + `AwardMesos`/`AwardAsset` actions.
- Produces: on `Collect`, a payout saga `[AwardMesos(+meso)?, AwardAsset(item,qty)?]` (only non-zero components) submitted to the orchestrator. Sweeper: `game.NewSweepTask(l, interval)` implementing `tasks.Task` (`Run()` re-injects tenant via `tenant.WithContext` per expired model and disposes it — mirror `expression/task.go`).

- [ ] **Step 1: Write failing test** — for the sweeper: put a session, advance `SetNowFunc` past TTL, run the task, assert the session is gone and no payout saga was produced. For payout: call `Collect` at a rung whose prize has meso+item, assert a saga with the two expected steps is produced (mock the saga producer).

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement** the saga submit processor, the sweeper task, and the `Collect` payout wiring. Register the sweeper in `main.go` (`go tasks.Register(l, tdm.Context())(game.NewSweepTask(l, ...))`).

- [ ] **Step 4: Run, verify pass** with `-race`; then `go vet ./... && go build ./... && docker buildx bake atlas-rps`; from repo root `GOWORK=off tools/redis-key-guard.sh`.

- [ ] **Step 5: Commit** — `git commit -m "feat(rps): collect payout saga + session sweeper (task-132)"`

---

## Milestone C — atlas-saga-orchestrator: `StartRPSGame` dispatch

### Task 13: RPS REST client + `handleStartRPSGame` + acceptance table

**Files:**
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/rps/{processor.go,requests.go,rest.go}` (mirror the `gachapon/` REST client package)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` — add `handleStartRPSGame` method, its `Handler` interface entry, and the `GetHandler` `case StartRPSGame`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` — add `sharedsaga.StartRPSGame: {},` to `acceptanceTable` (self-completing REST action → empty event set)
- Modify: the `HandlerImpl` construction site to inject the new `rps` processor (mirror `h.gachaponP`)
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go` (+ the acceptance-table coverage test will now require the entry)

**Interfaces:**
- Consumes: `saga.StartRPSGame` + `StartRPSGamePayload` (Task 1); the atlas-rps `POST /rps/games` endpoint (Task 11).
- Produces: on a `StartRPSGame` step, the orchestrator POSTs to atlas-rps (`RPS_URL` env → `{RPS_URL}rps/games`) with the payload, and self-completes the step (`StepCompleted(..., true)`) — no async event. Mirror `handleSelectGachaponReward` minus the follow-on-step injection.

- [ ] **Step 1: Write failing test** — a handler test: given a saga with a `StartRPSGame` step and an httptest atlas-rps returning 200, `handleStartRPSGame` completes the step; assert the POST body carried the payload. Also assert `event_acceptance_test` (the coverage test) passes once the table entry exists.

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement** the `rps/` REST client (`BaseUrl = "RPS_URL"`, `PostRequest[...]`), `handleStartRPSGame`, the interface method, the `GetHandler` case, and the acceptance-table entry.

- [ ] **Step 4: Run, verify pass;** then `go test -race ./... && go vet ./... && go build ./... && docker buildx bake atlas-saga-orchestrator`.

- [ ] **Step 5: Commit** — `git commit -m "feat(saga-orchestrator): dispatch StartRPSGame to atlas-rps (task-132)"`

---

## Milestone D — libs/atlas-packet: `RPS_GAME` clientbound dispatcher family

> **IDA-gated.** The frame vocabulary, per-mode field layouts, and per-version mode bytes come from decompiling `CRPSGameDlg::OnPacket`. Do NOT invent them.

### Task 14: IDA-verify `CRPSGameDlg::OnPacket` (clientbound read order + modes)

**Files:**
- Create: `docs/tasks/task-132-rps-npc-game/ida-rps-clientbound.md` (verification note: the `OnPacket` switch, one section per mode, each field with its decompile line/address, per version)

**Interfaces:**
- Produces: the authoritative clientbound frame list (OPEN / RESULT / END / any tie-redraw or timeout arm the switch reveals) with exact field order per frame, and the per-version mode-byte table for the `operations` map. This document is the source the codec + fixtures cite.

- [ ] **Step 1:** Confirm the loaded IDA instances with `list_instances` and match binary NAME per memory `reference_ida_instance_ports_shifted_idbs_v9` (v83 dump is present; v95 present; v84/v87/jms may be absent — if a version's IDB is not loaded, record it as **blocked-pending-IDB** for that version's cells, exactly like the v92 park, and proceed with the versions that are loaded).

- [ ] **Step 2:** For each available version, `func_query`/`decompile` `CRPSGameDlg::OnPacket`; transcribe the mode switch and, for each mode, the exact read sequence (types/order) into the note, citing addresses. Cross-check the opcode against the Global-Constraints table.

- [ ] **Step 3:** Resolve the open items from design §16: OnBtRetry-vs-tie-redraw and whether a tie is a distinct frame or an outcome code in RESULT; the `Update` sub-action's server relevance. Record the answers with evidence.

- [ ] **Step 4: Commit** the note — `git add docs/tasks/task-132-rps-npc-game/ida-rps-clientbound.md && git commit -m "docs(rps): IDA verification of RPS_GAME clientbound (task-132)"`

### Task 15: `RPS_GAME` clientbound codec + byte fixtures

**Files:**
- Create: `libs/atlas-packet/rps/clientbound/operation.go` (one discrete struct per mode from Task 14; `const RPSGameWriter = "RPSGame"`; each struct `Operation() string { return RPSGameWriter }`, a `// packet-audit:fname CRPSGameDlg::OnPacket#<Mode>` comment, `Encode` writing mode byte + arm body, matching `Decode`)
- Create: `libs/atlas-packet/rps/operation_body.go` (root body funcs `RPSGame<Mode>Body(...)` each `WithResolvedCode("operations", RPSGameMode<Mode>, func(mode byte) packet.Encoder {...})`; mode keys as `type RPSGameMode = string` consts)
- Modify: `tools/packet-audit/cmd/run.go` (`candidatesFromFName` — add one `case "CRPSGameDlg::OnPacket#<Mode>"` per mode → clientbound candidate in pkg `rps`)
- Create: `docs/packets/dispatchers/rps_game.yaml` (mode→key documentation, mirror `docs/packets/dispatchers/storage_operation.yaml`)
- Test: `libs/atlas-packet/rps/clientbound/operation_test.go` (per-mode byte fixtures with `// packet-audit:verify packet=rps/clientbound/<Struct> version=<v> ida=<addr>` markers, iterating `pt.Variants`)

**Interfaces:**
- Consumes: the frame list + modes from Task 14; opcodes from Global Constraints.
- Produces: `rps/clientbound` structs (per mode), `rps.RPSGame<Mode>Body(...)` body funcs (the writer API used by atlas-channel Task 19), and the `RPSGameWriter` const. Follows all INV-1..5.

- [ ] **Step 1: Write the failing fixture test** for the first mode (OPEN), asserting exact byte offsets from the Task-14 note and round-tripping via `pt.RoundTrip`. (Model on `libs/atlas-packet/storage/clientbound/show_test.go`.) Use a per-version mode-shift helper if the note shows the mode byte shifts across versions.

- [ ] **Step 2: Run, verify fail** — `cd libs/atlas-packet && go test ./rps/... -run TestRPSGame`.

- [ ] **Step 3: Implement** the clientbound structs + the body funcs + the `candidatesFromFName` cases + the dispatcher YAML, one mode at a time (OPEN, RESULT, END, plus any tie/timeout arm from Task 14). Every field cited to the Task-14 note; zero invented bytes.

- [ ] **Step 4: Run, verify pass** for each mode/version fixture.

- [ ] **Step 5: Run the dispatcher gate** — from repo root: `go run ./tools/packet-audit dispatcher-lint`, `go run ./tools/packet-audit matrix --check`, `go run ./tools/packet-audit fname-doc --check`, `go run ./tools/packet-audit operations --check`. All must exit 0. Then regenerate the matrix so the `RPS_GAME` cells promote for the verified versions (blocked-pending-IDB versions stay `⬜`/`❌` with a documented reason).

- [ ] **Step 6: Commit** — `git add libs/atlas-packet/rps tools/packet-audit/cmd/run.go docs/packets/dispatchers/rps_game.yaml docs/packets/audits && git commit -m "feat(packet): RPS_GAME clientbound dispatcher family (task-132)"`

---

## Milestone E — libs/atlas-packet: `RPS_ACTION` serverbound dispatcher family

> **IDA-gated.** The STATUS row names six senders (`OnBtStart`, `SendSelection`, `OnBtContinue`, `OnBtRetry`, `OnBtExit`, `Update`) → the serverbound side is itself a mode dispatcher. Derive its sub-op modes + body layouts from IDA.

### Task 16: IDA-verify the RPS_ACTION serverbound senders

**Files:**
- Create: `docs/tasks/task-132-rps-npc-game/ida-rps-serverbound.md`

**Interfaces:**
- Produces: the serverbound sub-op mode table (start/select/continue/retry/exit/update) and each arm's body field layout per version, with addresses. Confirms which arm carries the throw byte (SendSelection) and which are bodyless.

- [ ] **Step 1:** Per available version, decompile each `CRPSGameDlg::` sender (`OnBtStart`/`SendSelection`/`OnBtContinue`/`OnBtRetry`/`OnBtExit`/`Update`) to derive the leading sub-op byte each writes and the trailing body. Record per version; mark absent-IDB versions blocked-pending-IDB.

- [ ] **Step 2:** Resolve design §16 items 2/3/5 with evidence (retry vs tie-redraw, `Update` server relevance, exit-vs-continue semantics).

- [ ] **Step 3: Commit** — `git commit -m "docs(rps): IDA verification of RPS_ACTION serverbound (task-132)"`

### Task 17: `RPS_ACTION` serverbound codec + byte fixtures

**Files:**
- Create: `libs/atlas-packet/rps/serverbound/operation.go` (top-level `Operation{ mode byte }` decoding only the leading byte; `const RPSActionHandle = "RPSActionHandle"`; `Operation() string { return RPSActionHandle }`; `Mode() byte`)
- Create: `libs/atlas-packet/rps/serverbound/operation_select.go` (+ per-arm body structs for start/continue/retry/exit/update as the note requires; select carries the throw)
- Modify: `tools/packet-audit/cmd/run.go` (`candidatesFromFName` — serverbound cases per sender fname → `rps` serverbound candidates)
- Test: `libs/atlas-packet/rps/serverbound/operation_test.go` (per-arm byte fixtures + `// packet-audit:verify` markers)

**Interfaces:**
- Consumes: the sub-op table from Task 16.
- Produces: `rps/serverbound.Operation` + per-arm structs used by the channel handler (Task 18); `RPSActionHandle` const.

- [ ] **Step 1: Write failing fixtures** — decode a `select` frame (mode byte + throw) and each bodyless arm; assert `Mode()` + body. Model on `libs/atlas-packet/storage/serverbound/*`.

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement** `Operation` + arm structs + `candidatesFromFName` cases from the Task-16 note.

- [ ] **Step 4: Run, verify pass.**

- [ ] **Step 5: Dispatcher gate** — `dispatcher-lint` + `matrix --check` + `fname-doc --check` + `operations --check` all exit 0; regenerate matrix (RPS_ACTION cells promote for verified versions).

- [ ] **Step 6: Commit** — `git commit -m "feat(packet): RPS_ACTION serverbound dispatcher family (task-132)"`

---

## Milestone F — atlas-channel wiring

### Task 18: Serverbound `RPS_ACTION` handler (+validator registration)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/rps_action.go` (`RPSActionHandle` alias to `rpssb.RPSActionHandle`; `RPSActionHandleFunc`; local mode-name consts matching the serverbound `operations` keys; an `isRPSAction(l)(options, mode, key)` check mirroring `isStorageOperation`)
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (add `handlerMap[rpssb.RPSActionHandle] = handler.RPSActionHandleFunc` in `produceHandlers`)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/rps_action_test.go`

**Interfaces:**
- Consumes: `rps/serverbound` (Task 17); emits `Command`s to `atlas-rps` via a channel-local producer for `COMMAND_TOPIC_RPS` (add a small `kafka/producer` provider + `kafka/message/rps` mirror of the command struct, or reuse an existing channel producer pattern).
- Produces: decoded mode → the matching `atlas-rps` command (select→`CommandTypeSelect{throw}`, continue→`Continue`, exit→`Quit`, retry→per Task-16 semantics, update→per Task-16 (likely no-op)). Registered **with `LoggedInValidator`** — the validator is bound in the seed template (Task 20), and `BuildHandlerMap` silently drops a validator-less entry (memory `bug_socket_handler_missing_validator_silently_dropped`), so Task 20's seed row is mandatory.

- [ ] **Step 1: Write failing test** — decode a `select` operation via the handler, assert it emits a `Select` command with the throw (mock the producer); assert an unknown mode logs a warning and drops.

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement** the handler (clone `storage_operation.go` structure) + main.go registration + the channel-side rps command producer/message.

- [ ] **Step 4: Run, verify pass;** `go build ./...`.

- [ ] **Step 5: Commit** — `git commit -m "feat(channel): RPS_ACTION serverbound handler (task-132)"`

### Task 19: Clientbound `RPS_GAME` writer registration + event consumer

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (import `rpscb "…/libs/atlas-packet/rps/clientbound"`; add `rpscb.RPSGameWriter,` to `produceWriters()`)
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/rps/consumer.go` (subscribe `EVENT_TOPIC_RPS`; on `GameOpened`/`RoundResult`/`GameEnded`, resolve the session and `session.Announce(l)(ctx)(wp)(rpscb.RPSGameWriter)(<body>)(s)` using the `rps.RPSGame<Mode>Body(...)` funcs from Task 15)
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (register the new consumer alongside the others — `InitConsumers`/`InitHandlers`)
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/rps/consumer_test.go`

**Interfaces:**
- Consumes: `atlas-rps` `Event`s (Task 8); the `RPSGame<Mode>Body` writer funcs (Task 15).
- Produces: clientbound frames written to the player's session on each RPS event (open dialog / animate result / end).

- [ ] **Step 1: Write failing test** — feed a `RoundResult` event, assert the correct writer body func is selected and `Announce` targets the character's session (mirror the storage consumer error-demux test).

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement** the consumer (event `Type` → body func) + writer registration + main.go consumer wiring.

- [ ] **Step 4: Run, verify pass;** `go build ./... && docker buildx bake atlas-channel`.

- [ ] **Step 5: Commit** — `git commit -m "feat(channel): RPS_GAME writer + event consumer (task-132)"`

### Task 20: Tenant seed templates (5 versions) + live-config patch note

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`, `_gms_84_1.json`, `_gms_87_1.json`, `_gms_95_1.json`, `template_jms_185_1.json` — each gets: a `socket.handlers` entry (RPS_ACTION opcode → `LoggedInValidator` → `RPSActionHandle` → `operations` sub-op table) and a `socket.writers` entry (RPS_GAME opcode → `RPSGame` → `operations` mode-byte table). Opcodes from Global Constraints; mode tables from Tasks 14/16.
- Create: `docs/tasks/task-132-rps-npc-game/live-config-patch.md` (the PATCH steps to add the RPS opcodes/handlers/writers/operations to already-provisioned live tenant configs + channel restart — memory `bug_new_opcodes_not_in_live_tenant_config`)

**Interfaces:**
- Consumes: the verified mode tables (Tasks 14/16) and the handler/writer names (Tasks 15/17/18/19).
- Produces: tenant-config wiring so the opcodes route. `operations` table is the single mode-byte source read by both `WithResolvedCode` (writer) and `isRPSAction` (handler).

- [ ] **Step 1:** Add the handler entry to `template_gms_83_1.json` (opcode `0x088`), e.g.:

```json
{
  "opCode": "0x088",
  "validator": "LoggedInValidator",
  "handler": "RPSActionHandle",
  "options": { "operations": { "START": <n>, "SELECT": <n>, "CONTINUE": <n>, "RETRY": <n>, "EXIT": <n>, "UPDATE": <n> } }
}
```

(the `<n>` sub-op values come from the Task-16 note — do not invent).

- [ ] **Step 2:** Add the writer entry to `template_gms_83_1.json` (opcode `0x138`):

```json
{
  "opCode": "0x138",
  "writer": "RPSGame",
  "options": { "operations": { "OPEN": <n>, "RESULT": <n>, "END": <n> } }
}
```

(mode values from the Task-14 note).

- [ ] **Step 3:** Repeat Steps 1–2 for `_gms_84_1` (`0x08C`/`0x13F`), `_gms_87_1` (`0x090`/`0x149`), `_gms_95_1` (`0x0A0`/`0x173`), `template_jms_185_1` (`0x08B`/`0x151`), using each version's mode tables. Skip any version marked blocked-pending-IDB (record the omission).

- [ ] **Step 4:** Write `live-config-patch.md`.

- [ ] **Step 5:** Validate JSON — `for f in template_gms_83_1 template_gms_84_1 template_gms_87_1 template_gms_95_1 template_jms_185_1; do python3 -m json.tool services/atlas-configurations/seed-data/templates/$f.json >/dev/null && echo "$f ok"; done`. Then run `go run ./tools/packet-audit operations --check` (seed ↔ dispatcher-YAML consistency).

- [ ] **Step 6: Commit** — `git commit -m "feat(configurations): seed RPS opcodes + operations tables (task-132)"`

---

## Milestone G — atlas-tenants: `rps-rewards` config resource

### Task 21: `rps-rewards` configuration resource + seed data

**Files (all under `services/atlas-tenants/atlas.com/tenants/` unless noted):**
- Modify: `configuration/rest.go` — `RpsRewardRestModel` (`GetName()="rps-rewards"`, `GetID`/`SetID`), `TransformRpsReward`, `ExtractRpsReward` (`type:"rps-rewards"`; note nested `ladder` array + numeric `entryCostMeso` arriving as `float64`), `CreateRpsRewardJsonData`/`CreateSingleRpsRewardJsonData`
- Modify: `configuration/resource.go` — 5 CRUD handlers + `SeedRpsRewardsHandler` + 6 route registrations inside `RegisterRoutes` under `/tenants/{tenantId}/configurations/rps-rewards`
- Modify: `configuration/processor.go` — 10 interface entries + `SeedRpsRewards` + impls (clone the vessels block; swap the `"rps-rewards"` resource-name literal)
- Modify: `configuration/provider.go` — `GetRpsRewardByIdProvider` + `GetAllRpsRewardsProvider` on `GetByTenantIdAndResourceNameProvider(tenantID, "rps-rewards")`
- Modify: `configuration/kafka.go` — `EventTypeRpsRewardCreated/Updated/Deleted` + `CreateRpsRewardStatusEventProvider` (`ResourceType: "rps-reward"`)
- Modify: `configuration/seed.go` — `defaultRpsRewardsPath = "/configurations/rps-rewards"`, `RPS_REWARDS_SEED_PATH` override, `LoadRpsRewardFiles()`
- Modify: `configuration/mock/processor.go` — mock fields + methods (compile-time `var _ Processor` check enforces completeness)
- Modify: `rest/handler.go` — `ParseRpsRewardId(l, next)` on route segment `{rpsRewardId}`
- Create: `services/atlas-tenants/configurations/rps-rewards/default.json` (a valid **default** ladder — entry cost 1000, a minimal meso-only ladder as operator-tunable defaults; authoritative content filled in Task 26)
- Test: `services/atlas-tenants/atlas.com/tenants/configuration/rest_test.go` (transform/extract round-trip) + mock-parity is enforced at compile time

**Interfaces:**
- Produces: the REST surface `atlas-rps` reads in Task 7, and the seed data. `entryCostMeso` default 1000; `ladder` ordered, top element = max rung.

- [ ] **Step 1: Write failing test** — `TransformRpsReward(ExtractRpsReward(m))` round-trips `entryCostMeso` + a 2-rung ladder.

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement** all files by cloning the vessels blocks (agent-mapped line ranges) and swapping identifiers/resource name. Do NOT touch `entity.go`/`model.go`/`builder.go`/`administrator.go` (generic) or `main.go` (RegisterRoutes already wired).

- [ ] **Step 4:** Create `configurations/rps-rewards/default.json` (flat single record, no `data` envelope — matches the vessels seed-file shape):

```json
{
  "id": "rps-rewards",
  "type": "rps-rewards",
  "attributes": {
    "entryCostMeso": 1000,
    "ladder": [
      { "rung": 1, "itemId": 0, "quantity": 0, "meso": 2000 },
      { "rung": 2, "itemId": 0, "quantity": 0, "meso": 5000 },
      { "rung": 3, "itemId": 0, "quantity": 0, "meso": 10000 }
    ]
  }
}
```

(These meso values are **operator-tunable defaults**, explicitly not claimed-authentic — Task 26 replaces the ladder with Cosmic-sourced, WZ-verified content.)

- [ ] **Step 5: Run tests + gate** — `go test -race ./... && go vet ./... && go build ./... && docker buildx bake atlas-tenants`.

- [ ] **Step 6: Commit** — `git commit -m "feat(tenants): rps-rewards configuration resource + default seed (task-132)"`

---

## Milestone H — atlas-npc-conversations: NPC 9000019

### Task 22: Saga re-export shim

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/saga/model.go` — add `StartRPSGamePayload = sharedsaga.StartRPSGamePayload` (payload-alias block ~line 66) and `StartRPSGame = sharedsaga.StartRPSGame` (action-const block ~line 154)

**Interfaces:**
- Produces: local aliases `saga.StartRPSGame`, `saga.StartRPSGamePayload` for use in Task 23.

- [ ] **Step 1:** Add the two alias lines.
- [ ] **Step 2: Verify** — `cd services/atlas-npc-conversations/atlas.com/npc && go build ./...`. Expected: clean.
- [ ] **Step 3: Commit** — `git commit -m "feat(npc): re-export StartRPSGame saga action (task-132)"`

### Task 23: `rpsAction` state type + `processRPSActionState`

**Files (under `services/atlas-npc-conversations/atlas.com/npc/conversation/`):**
- Modify: `model.go` — `RPSActionType StateType = "rpsAction"` (~line 40); accessor; builder `SetRPSAction`; `Build()` validation; `RPSActionModel{ npcId uint32; entryCostMeso uint32; failureState string }` + getters + `NewRPSActionBuilder()`/`Build()` (mirror `GachaponActionModel` ~line 1358)
- Modify: `model_json.go` — `RPSActionModel` `MarshalJSON`/`UnmarshalJSON` (~line 279 pattern) + add `RPSAction *RPSActionModel json:"rpsAction,omitempty"` to the state envelope (~line 427)
- Modify: `processor.go` — `case RPSActionType:` in `processState` (~line 480) → `processRPSActionState`, which builds the entry saga `[AwardMesos(−entryCostMeso), StartRPSGame]` with a fresh `uuid.New()` transaction id, `Create`s it, sets `pendingSagaId`, stores `Context()["rpsAction_failureState"]`, and returns the same state id to park (mirror `processGachaponActionState` ~line 934)
- Test: `services/atlas-npc-conversations/atlas.com/npc/conversation/processor_test.go` (+ `model_json_test.go` round-trip)

**Interfaces:**
- Consumes: `saga.StartRPSGame`/`StartRPSGamePayload` (Task 22), `saga.AwardMesos`/`AwardMesosPayload` (existing).
- Produces: the parked-on-saga entry flow. The entry saga (design D3): step 1 `AwardMesos{ Amount: -int32(entryCostMeso), CharacterId, WorldId, ChannelId, ActorId, ActorType, ShowEffect:false }` (a NOT_ENOUGH_MESO failure routes to `rpsAction_failureState` via the saga-failed consumer, Task 24 — FR-1.3); step 2 `StartRPSGame{ CharacterId, WorldId, ChannelId, NpcId }`.

**Decision (refines design §8):** the entry cost is carried on the `rpsAction` state JSON field `entryCostMeso` (mirrors `gachaponAction.ticketItemId`), keeping npc-conversations free of a new config client. It stays tenant-tunable via the per-version seed and matches the `rps-rewards` config default (1000) by convention.

- [ ] **Step 1: Write failing tests** — `model_json` round-trip of an `rpsAction` state; a processor test that `processRPSActionState` builds a 2-step saga (`AwardMesos` negative amount then `StartRPSGame`), sets `pendingSagaId`, and parks on the same state (mock the saga processor).

- [ ] **Step 2: Run, verify fail.**

- [ ] **Step 3: Implement** the state type (model + json + envelope), the processor case + `processRPSActionState`.

- [ ] **Step 4: Run, verify pass.**

- [ ] **Step 5: Commit** — `git commit -m "feat(npc): rpsAction state + entry saga (task-132)"`

### Task 24: Resume/failure routing in the saga consumer

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/kafka/consumer/saga/consumer.go` — in `handleStatusEventCompleted` (~line 41) add an `rpsAction_failureState` branch (clear pending, End + Dispose — mirror the gachapon arm ~line 99); in `handleStatusEventFailed`/`resolveFailureState` (~line 221) add the `rpsAction` fallback routing to the stored failure state (FR-1.3 not-enough-meso path)
- Test: `services/atlas-npc-conversations/atlas.com/npc/kafka/consumer/saga/consumer_test.go`

**Interfaces:**
- Consumes: the `rpsAction_failureState` context flag set in Task 23.
- Produces: on entry-saga success → conversation ends + NPC disposed (the client dialog takes over via `GameOpened`); on failure → route to the failure dialogue state.

- [ ] **Step 1: Write failing test** — a completed status event for an rpsAction conversation clears pending + disposes; a failed event routes to the failure state.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** both branches.
- [ ] **Step 4: Run, verify pass;** `go build ./... && docker buildx bake atlas-npc-conversations`.
- [ ] **Step 5: Commit** — `git commit -m "feat(npc): resume/failure routing for rpsAction (task-132)"`

### Task 25: NPC 9000019 conversation seeds (5 versions)

**Files:**
- Create: `deploy/seed/gms/83_1/npc-conversations/npc/npc-9000019.json`, and the `84_1`, `87_1`, `95_1` siblings, plus `deploy/seed/jms/185_1/npc-conversations/npc/npc-9000019.json`

**Interfaces:**
- Consumes: the `rpsAction` state type (Task 23). Structure: JSON:API envelope (`data.type:"npc-conversation"`, `data.id:"9000019"`), `startState` → a `dialogue` `sendYesNo` offer (entry cost surfaced in text) → on Yes an `rpsAction` state (`entryCostMeso`, `npcId:9000019`, `failureState`) → a `sendOk` "not enough meso" failure dialogue. Mirror `deploy/seed/gms/83_1/npc-conversations/npc/npc-9100100.json`.

- [ ] **Step 1:** Create the v83 seed:

```json
{
  "data": {
    "attributes": {
      "npcId": 9000019,
      "startState": "offer",
      "states": [
        {
          "id": "offer",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendYesNo",
            "text": "Care for a game of Rock Paper Scissors? It costs #b1000 meso#k to play. Win and climb the prize ladder!",
            "choices": [
              { "nextState": "playRPS", "text": "Yes" },
              { "nextState": null, "text": "No" }
            ]
          }
        },
        {
          "id": "playRPS",
          "type": "rpsAction",
          "rpsAction": { "npcId": 9000019, "entryCostMeso": 1000, "failureState": "noMeso" }
        },
        {
          "id": "noMeso",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendOk",
            "text": "You don't have enough meso to play.",
            "choices": [ { "nextState": null, "text": "Ok" } ]
          }
        }
      ]
    },
    "id": "9000019",
    "type": "npc-conversation"
  }
}
```

- [ ] **Step 2:** Copy to the `84_1`, `87_1`, `95_1`, and `jms/185_1` seed dirs (identical content — the conversation is version-agnostic).

- [ ] **Step 3: Validate JSON** — `for d in gms/83_1 gms/84_1 gms/87_1 gms/95_1 jms/185_1; do python3 -m json.tool deploy/seed/$d/npc-conversations/npc/npc-9000019.json >/dev/null && echo "$d ok"; done`.

- [ ] **Step 4: Commit** — `git commit -m "feat(seed): NPC 9000019 RPS conversation (task-132)"`

---

## Milestone I — Reward-ladder content + end-to-end verification

### Task 26: Cosmic-sourced, WZ-verified reward ladder

**Files:**
- Modify: `services/atlas-tenants/configurations/rps-rewards/default.json` (replace the default ladder with verified content)
- Create/Update: `docs/tasks/task-132-rps-npc-game/reward-ladder.md` (the Cosmic `9000019.js` reward set + the WZ/atlas-data verification of every item id + quantity)

**Interfaces:**
- Produces: the authoritative entry cost + rung prizes. Every `itemId` verified to exist in local WZ / atlas-data (memory: verify, don't invent — no MapleStory-memory item ids).

- [ ] **Step 1:** Obtain the Cosmic `9000019.js` reward set (the reference the PRD/design name). Record the raw ladder in `reward-ladder.md`.

- [ ] **Step 2:** For every item id in the ladder, verify it resolves in local WZ / atlas-data (query atlas-data or the WZ item tables). Any id that does not resolve is dropped or replaced with a meso equivalent — record the decision. Do not ship an unverified item id.

- [ ] **Step 3:** Write the verified ladder into `default.json`; keep `entryCostMeso` consistent with the NPC seed (Task 25) — if the Cosmic entry cost differs from 1000, update BOTH the config and all five NPC seeds.

- [ ] **Step 4: Validate JSON** + re-run the atlas-tenants module gate.

- [ ] **Step 5: Commit** — `git commit -m "feat(rps): Cosmic-sourced WZ-verified reward ladder (task-132)"`

### Task 27: Full verification gate

**Files:** none (verification only) — record results in `docs/tasks/task-132-rps-npc-game/verification.md`.

**Interfaces:** consumes every prior task.

- [ ] **Step 1:** For **every changed module** (`libs/atlas-saga`, `libs/atlas-packet`, `services/atlas-rps`, `services/atlas-saga-orchestrator`, `services/atlas-channel`, `services/atlas-configurations`, `services/atlas-tenants`, `services/atlas-npc-conversations`, `tools/packet-audit`): run `go test -race ./...`, `go vet ./...`, `go build ./...`. All clean.

- [ ] **Step 2:** From the worktree root, **`docker buildx bake`** each service whose `go.mod` was touched: `atlas-rps`, `atlas-saga-orchestrator`, `atlas-channel`, `atlas-configurations`, `atlas-tenants`, `atlas-npc-conversations` (and confirm `libs/atlas-packet`/`libs/atlas-saga` consumers still bake). Use `docker buildx bake all-go-services` if faster.

- [ ] **Step 3:** From repo root: `GOWORK=off tools/redis-key-guard.sh` (clean), and `go run ./tools/packet-audit dispatcher-lint && … matrix --check && … fname-doc --check && … operations --check` (all exit 0).

- [ ] **Step 4:** `kustomize build deploy/k8s/overlays/pr >/dev/null && kustomize build deploy/k8s/overlays/main >/dev/null` (manifests parse).

- [ ] **Step 5:** Run the **code-review** step (CLAUDE.md "Code Review Before PR") via `superpowers:requesting-code-review` — it dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer` (Go changed). Address findings in `docs/tasks/task-132-rps-npc-game/audit.md`.

- [ ] **Step 6:** Record the parked follow-ups in `verification.md`: **v92 support** (needs a v92 IDB — mirrors task-086 mount-food), and any version marked **blocked-pending-IDB** in Tasks 14/16.

- [ ] **Step 7: Commit** — `git add docs/tasks/task-132-rps-npc-game && git commit -m "docs(rps): final verification + parked follow-ups (task-132)"`

---

## Self-Review notes (spec coverage)

- FR-1.1..1.4 → Tasks 23, 25, 11 (re-entry dispose). FR-2.1..2.6 → Tasks 9, 15, 17, 18, 19. FR-3.1..3.7 → Tasks 6, 9 (ladder + collect/continue/max-rung). FR-4.1..4.3 → Tasks 9 (Quit/Dispose), 12 (sweeper). FR-5.1..5.3 → Tasks 14–17, 20. FR-6.1..6.3 → Tasks 1, 12, 13 (saga-mediated economy).
- PRD §5 API surface → Tasks 8 (Kafka), 11 (REST). §6 data model → Tasks 3 (session), 21 (rps-rewards). §7 service impact → all milestones. §9 open questions → Tasks 14/16 (IDA), 26 (ladder). Acceptance criteria → Task 27.
- **v92** deliberately out of scope (design D1) — recorded as a parked follow-up in Task 27, not silently dropped.
- **IDA/WZ-gated values** (mode bytes, frame layouts, item ids) are derived in Tasks 14, 16, 26 — never invented; every downstream task cites those notes.
