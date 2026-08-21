# Self-Destructing Mobs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make a monster's WZ `selfDestruction` block do something — detonate on an HP threshold, on a timer, or on a client `MONSTER_BOMB` contact report — with the WZ animation on the wire and the same drops, EXP and bookkeeping as an ordinary kill.

**Architecture:** Four trigger paths (damage threshold, DoT tick, timer sweep, `SELF_DESTRUCT` command) converge on one atomic registry primitive (`Registry.SelfDestruct`) and one shared kill epilogue (`finalizeKill`), which is the same epilogue an ordinary kill runs. The animation byte rides `KILLED`/`DESTROYED` to atlas-channel, which writes it into `MonsterDestroy`; the codec's dead-type-4 trailing `int32` becomes version-gated.

**Tech Stack:** Go 1.x, Redis (`atlas-redis` `TenantRegistry`), Kafka (`atlas-kafka`), JSON:API REST (`atlas-rest`), miniredis + `producertest` for tests, `packet-audit` for the coverage matrix.

**Spec:** [`design.md`](./design.md) (PRD: [`prd.md`](./prd.md))

## Global Constraints

- Worktree: `.worktrees/task-253-self-destructing-mobs`. Branch: `task-253-self-destructing-mobs`. Never edit the main repo.
- Never invent a value, name, opcode, or behavior. Every dead-type meaning cited in a comment must carry its IDA address from design §2.1/§2.2.
- Version gates use the `MajorAtLeast(N)` idiom, never a raw `>` comparison (`gate-lint` is a CI gate).
- Immutable domain models: unexported fields, value receivers, builders for test setup. No `*_testhelpers.go`.
- Check `libs/atlas-constants/` before defining any new domain type or numeric constant.
- No stubs, no `// TODO`, no unimplemented status responses.
- Implementers run module-local `go build ./... && go test ./...` only. Repo-wide `tools/verify.sh` is the controller's gate.
- The absent-`selfDestruction` sentinel is `{action: 0, removeAfter: -1, hp: -1}` after Task 1. Presence is `Hp > -1 || RemoveAfter > -1`. Never `hp > -1` alone (design §2.6).
- Death-type byte is passed through from WZ verbatim; the server never remaps it per version (design §2.2).

---

## Task order & dependencies

```
1 (atlas-data sentinel)      independent
2 (packet gate)              independent
3 (matrix re-verify)         needs 2
4 (information type)         needs 1
5 (Registry.SelfDestruct)    independent
6 (kill epilogue + deathType) independent
7 (threshold + SelfDestruct)  needs 4, 5, 6
8 (DoT threshold)             needs 7
9 (timer registry + task)     needs 4, 7
10 (SELF_DESTRUCT consumer)   needs 7
11 (channel contract + wire)  needs 2, 6
12 (MONSTER_BOMB handler)     needs 11
13 (monster-death ActorId=0)  independent
```

---

### Task 1: `atlas-data` — absent-`selfDestruction` sentinel

**Goal:** A monster with no `selfDestruction` node reports `{0, -1, -1}` rather than `{0, 0, 0}`, so "absent" and "present with hp 0" stop colliding (design D2, §2.6; PRD FR-1.4).

### Files

- `services/atlas-data/atlas.com/data/monster/reader.go:206` — `getSelfDestruction`'s missing-node return
- `services/atlas-data/atlas.com/data/monster/reader_test.go:1267` — expectation for a mob with no block
- `services/atlas-data/atlas.com/data/monster/rest.go:43` — read-only; the `selfDestruction` REST shape (unchanged)

Module root for `go build`/`go test`: `services/atlas-data/atlas.com/data`.

- [x] **Step 1: Update the failing expectation**

In `reader_test.go`, the existing assertion at line 1267 reads:

```go
if rm.SelfDestruction != (selfDestruction{0, 0, 0}) {
	t.Errorf("SelfDestruction mismatch: got %+v, expected %+v", rm.SelfDestruction, selfDestruction{0, 0, 0})
}
```

Change both literals to `selfDestruction{0, -1, -1}`. Field order is `{Action, RemoveAfter, Hp}`.

- [x] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-data/atlas.com/data && go test ./monster/ -run TestRead -v`
Expected: FAIL, `SelfDestruction mismatch: got {Action:0 RemoveAfter:0 Hp:0}, expected {Action:0 RemoveAfter:-1 Hp:-1}`

- [x] **Step 3: Fix the sentinel**

In `reader.go`, `getSelfDestruction`:

```go
func getSelfDestruction(node *xml.Node) selfDestruction {
	c, err := node.ChildByName("selfDestruction")
	if err != nil {
		// Absent block. The -1 sentinels match the present-but-omitted-field
		// defaults below, so every consumer has ONE shape to read: hp == -1 and
		// removeAfter == -1 together mean "this monster does not self-destruct"
		// (task-253 design D2). Returning the zero value here would make hp == 0
		// — a legal threshold — for every ordinary monster in the game.
		return selfDestruction{Action: 0, RemoveAfter: -1, Hp: -1}
	}
	action := byte(c.GetIntegerWithDefault("action", 0))
	removeAfter := c.GetIntegerWithDefault("removeAfter", -1)
	hp := c.GetIntegerWithDefault("hp", -1)
	return selfDestruction{
		Action:      action,
		RemoveAfter: removeAfter,
		Hp:          hp,
	}
}
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-data/atlas.com/data && go build ./... && go test ./monster/`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/monster/reader.go services/atlas-data/atlas.com/data/monster/reader_test.go
git commit -m "fix(atlas-data): report -1 sentinels for an absent selfDestruction block"
```

---

### Task 2: `libs/atlas-packet` — dead-type constants and the version-gated trailing `int32`

**Goal:** `MonsterDestroy` carries the trailing `swallowCharacterId` only on the versions whose `CMobPool::OnMobLeaveField` reads it, and names every dead-type the WZ data uses (design D10, §2.1, §2.2; PRD FR-4.4, FR-4.5, FR-4.6).

**Interfaces**

- Produces: `hasSwallowCharacterId(t tenant.Model) bool`; exported constants `DestroyTypeDisappear`, `DestroyTypeFadeOut`, `DestroyTypeBomb`, `DestroyTypeDestructByMiss`, `DestroyTypeSwallow`, `DestroyTypeSelfDestruct` of type `DestroyType`.

### Files

- `libs/atlas-packet/monster/clientbound/destroy.go` — constants, gate, `Encode`/`Decode`
- `libs/atlas-packet/monster/clientbound/destroy_test.go` — new byte fixtures
- `libs/atlas-packet/reactor/clientbound/spawn.go:29` — read-only; the gate idiom to copy
- `libs/atlas-packet/test/context.go` — read-only; `test.Variants` and `test.CreateContext`

Module root: `libs/atlas-packet`.

Patterns to copy: `libs/atlas-packet/reactor/clientbound/spawn.go:16-31` (gate function + doc comment shape), `libs/atlas-packet/reactor/clientbound/spawn.go:60-70` (taking the tenant from `ctx` inside `Encode`).

- [x] **Step 1: Write the failing tests**

Append to `destroy_test.go`. `TestMonsterDestroySwallowGate` is table-driven over the version keys below; `TestMonsterDestroyNonSwallowTypesAreFiveBytes` covers the non-gated dead-types.

Setup to copy: `destroy_test.go:1-10` (imports) and `TestMonsterDestroyBytesV79` (`test.CreateContext(region, major, minor)`, `input.Encode(nil, ctx)(nil)`, `bytes.Equal`).

`uniqueId` is 5001 in every case → `0x89 0x13 0x00 0x00`. `swallowCharacterId` is 12345 → `0x39 0x30 0x00 0x00`.

`TestMonsterDestroySwallowGate` — input `NewMonsterDestroyBySwallow(5001, 12345)`:

| subtest | region | major | minor | expected bytes |
|---|---|---|---|---|
| `gms_v48` | GMS | 48 | 1 | `89 13 00 00 04` |
| `gms_v61` | GMS | 61 | 1 | `89 13 00 00 04` |
| `gms_v72` | GMS | 72 | 1 | `89 13 00 00 04` |
| `gms_v79` | GMS | 79 | 1 | `89 13 00 00 04` |
| `gms_v83` | GMS | 83 | 1 | `89 13 00 00 04` |
| `gms_v84` | GMS | 84 | 1 | `89 13 00 00 04` |
| `gms_v87` | GMS | 87 | 1 | `89 13 00 00 04` |
| `gms_v92` | GMS | 92 | 1 | `89 13 00 00 04 39 30 00 00` |
| `gms_v95` | GMS | 95 | 1 | `89 13 00 00 04 39 30 00 00` |
| `jms_v185` | JMS | 185 | 1 | `89 13 00 00 04 39 30 00 00` |

Each subtest must also assert decode symmetry: build a fresh `var out Destroy`, decode the encoded bytes under the same ctx via `test.RoundTrip(t, ctx, input.Encode, out.Decode, nil)`, which fails if any byte is left unconsumed.

`TestMonsterDestroyNonSwallowTypesAreFiveBytes` — for every `v` in `test.Variants`, and for each of `DestroyTypeFadeOut` (1), `DestroyTypeDestructByMiss` (3), `DestroyTypeSelfDestruct` (5), `NewMonsterDestroy(5001, dt).Encode(nil, ctx)(nil)` must be exactly 5 bytes: `89 13 00 00` followed by the single dead-type byte (`01`, `03`, `05`). Subtest name: `fmt.Sprintf("%s/type%d", v.Name, dt)`.

Add these `packet-audit:verify` markers above `TestMonsterDestroySwallowGate` (addresses from design §2.1):

```
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=gms_v61 ida=0x5d4b87
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=gms_v72 ida=0x6258a1
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=gms_v79 ida=0x646ff6
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=gms_v83 ida=0x67961d
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=gms_v84 ida=0x6901b3
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=gms_v87 ida=0x6b5169
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=gms_v92 ida=0x64bb90
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=gms_v95 ida=0x658b90
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=jms_v185 ida=0x6f8a1f
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./monster/clientbound/ -run 'TestMonsterDestroySwallowGate|TestMonsterDestroyNonSwallowTypes' -v`
Expected: FAIL — `gms_v48`…`gms_v87` encode 9 bytes, want 5; `DestroyTypeDestructByMiss` / `DestroyTypeSelfDestruct` are undefined identifiers (compile error).

- [x] **Step 3: Add the constants and the gate**

In `destroy.go`, replace the constant block and add the gate above the struct:

```go
// DestroyType is CMob::m_nDeadType as CMobPool::Update switches on it. The
// same byte does NOT render the same way on every version — v95 (0x658610)
// has a full six-arm switch, v87 (0x6b4c78) collapses it to
// {0,1,3} -> CMob::OnDie and {2,4,5} -> the action-12 bomb. The server passes
// the WZ selfDestruction.action byte through verbatim and never remaps it;
// per-version rendering is the client's business (task-253 design §2.2).
const (
	// DestroyTypeDisappear removes the mob with no animation. OnMobLeaveField
	// removes a type-0 mob immediately rather than queueing it.
	DestroyTypeDisappear DestroyType = 0
	// DestroyTypeFadeOut is the ordinary death: v95 CMob::OnDie @0x64e4b0.
	DestroyTypeFadeOut DestroyType = 1
	// DestroyTypeBomb is v95 CMob::OnBomb @0x650ec0.
	DestroyTypeBomb DestroyType = 2
	// DestroyTypeDestructByMiss is v95 CMob::OnDestructByMiss @0x64ea30
	// (SE_MOB_DIE, m_nOneTimeAction = 40, m_nSuspended = 2).
	DestroyTypeDestructByMiss DestroyType = 3
	// DestroyTypeSwallow is v95 CMob::OnSwallowed @0x641810, which consumes
	// m_dwSwallowCharacterID. It is the ONLY dead-type that carries a trailing
	// wire field, and only on the versions hasSwallowCharacterId reports.
	DestroyTypeSwallow DestroyType = 4
	// DestroyTypeSelfDestruct renders as an ordinary die on v95 and as the
	// action-12 bomb on v87 (sub_69E44A).
	DestroyTypeSelfDestruct DestroyType = 5
)

// hasSwallowCharacterId reports whether CMobPool::OnMobLeaveField reads the
// trailing int32 that follows dead-type 4. Swept across all ten IDBs
// (task-253 design §2.1): absent on GMS v48 0x55957b, v61 0x5d4b87,
// v72 0x6258a1, v79 0x646ff6, v83 0x67961d, v84 0x6901b3, v87 0x6b5169;
// present on GMS v92 0x64bb90, v95 0x658b90 and JMS v185 0x6f8a1f. The JMS
// arm is derived independently, not left to fall out of MajorAtLeast(185 >= 92).
func hasSwallowCharacterId(t tenant.Model) bool {
	return (t.IsRegion("GMS") && t.MajorAtLeast(92)) || t.Region() == "JMS"
}
```

Add `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"` to the imports.

- [x] **Step 4: Gate `Encode` and `Decode`**

```go
func (m Destroy) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteByte(byte(m.destroyType))
		if m.destroyType == DestroyTypeSwallow && hasSwallowCharacterId(t) {
			w.WriteInt(m.swallowCharacterId)
		}
		return w.Bytes()
	}
}

func (m *Destroy) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.destroyType = DestroyType(r.ReadByte())
		if m.destroyType == DestroyTypeSwallow && hasSwallowCharacterId(t) {
			m.swallowCharacterId = r.ReadUint32()
		}
	}
}
```

Also update `NewMonsterDestroyBySwallow`'s doc comment to say the trailing field is version-gated to GMS ≥ 92 / JMS.

- [x] **Step 5: Run the tests to verify they pass**

Run: `cd libs/atlas-packet && go build ./... && go test ./monster/clientbound/`
Expected: PASS, including the pre-existing `TestMonsterDestroy`, `TestMonsterDestroyBytesV79`, `TestMonsterDestroyBytesV72` and `TestMonsterDestroyBySwallow`.

- [x] **Step 6: Confirm no raw version comparison was introduced**

Run: `go run ./tools/packet-audit gate-lint --check`
Expected: exit 0.

- [x] **Step 7: Commit**

```bash
git add libs/atlas-packet/monster/clientbound/destroy.go libs/atlas-packet/monster/clientbound/destroy_test.go
git commit -m "fix(atlas-packet): version-gate the dead-type-4 trailing int32 on MonsterDestroy"
```

---

### Task 3: Packet coverage matrix — re-verify every `MonsterDestroy` cell

**Goal:** Every cell whose encoder moved in Task 2 is re-verified with evidence, `gms_v92` promotes from `partial` to `verified`, and the gate is registered (design §10; PRD FR-4.7).

### Files

- `docs/packets/gates.yaml` — new gate row, `>=92` boundary
- `docs/packets/audits/STATUS.md` — regenerated, do not hand-edit
- `docs/packets/audits/status.json` — regenerated, do not hand-edit
- `docs/packets/evidence/gms_v92/monster.clientbound.MonsterDestroy.yaml` — new file, written by `evidence pin`
- `docs/packets/evidence/gms_v61/monster.clientbound.MonsterDestroy.yaml` — add `verifies:`
- `docs/packets/evidence/gms_v72/monster.clientbound.MonsterDestroy.yaml` — add `verifies:`
- `docs/packets/evidence/gms_v79/monster.clientbound.MonsterDestroy.yaml` — add `verifies:`
- `docs/packets/evidence/gms_v83/monster.clientbound.MonsterDestroy.yaml` — add `verifies:`
- `docs/packets/evidence/gms_v84/monster.clientbound.MonsterDestroy.yaml` — add `verifies:`
- `docs/packets/evidence/gms_v87/monster.clientbound.MonsterDestroy.yaml` — add `verifies:`
- `docs/packets/evidence/gms_v95/monster.clientbound.MonsterDestroy.yaml` — add `verifies:`
- `docs/packets/evidence/jms_v185/monster.clientbound.MonsterDestroy.yaml` — add `verifies:`
- `docs/packets/audits/VERIFYING_A_PACKET.md` — read-only; §7 and §8 are the playbook
- `libs/atlas-packet/monster/clientbound/destroy_test.go` — read-only here; Task 2 added the markers

`gms_v48` is `incomplete` and explicitly out of scope — do not attempt to promote it.

- [x] **Step 1: Pin the v92 evidence record**

```bash
go run ./tools/packet-audit evidence pin \
  --packet monster/clientbound/MonsterDestroy --version gms_v92 \
  --ida "CMobPool::OnMobLeaveField" --category TIER1-FIXTURE
```

Expected: writes `docs/packets/evidence/gms_v92/monster.clientbound.MonsterDestroy.yaml` with `address: "0x64bb90"` resolved from `docs/packets/ida-exports/gms_v92.json`.

- [x] **Step 2: Add `verifies:` to all nine evidence records**

Append to each of the nine YAMLs (the v92 one just written plus the eight existing):

```yaml
verifies:
    - libs/atlas-packet/monster/clientbound/destroy_test.go#TestMonsterDestroySwallowGate
    - libs/atlas-packet/monster/clientbound/destroy_test.go#TestMonsterDestroyNonSwallowTypesAreFiveBytes
```

Preserve each file's existing indentation style (four spaces under `ida:` in the current records).

- [x] **Step 3: Register the gate**

Append to `docs/packets/gates.yaml`, in a new `# ---- boundary v87 / v92 ----` group placed in version order:

```yaml
  # ---- boundary v87 / v92 ------------------------------------------------
  - packet: monster/clientbound/MonsterDestroy
    direction: clientbound
    field: dead-type-4 trailing swallowCharacterId (read v92+; `(GMS>=92)||JMS`)
    boundary: ">=92"
    lower_version_key: gms_v87
    upper_version_key: gms_v92
```

- [x] **Step 4: Regenerate the matrix**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Expected: `matrix --check` exits 0, and the `KILL_MONSTER` row's `gms_v92` cell is now `"state": "verified"` in `docs/packets/audits/status.json` (it was `"partial"`).

- [x] **Step 5: Run the remaining `--check` passes**

```bash
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit gate-lint --check
```

Expected: all exit 0.

- [x] **Step 6: Commit**

```bash
git add docs/packets/gates.yaml docs/packets/audits/STATUS.md docs/packets/audits/status.json docs/packets/evidence
git commit -m "docs(packets): re-verify MonsterDestroy cells behind the v92 swallow gate"
```

---

### Task 4: `atlas-monsters` — `information.SelfDestruction` value type

**Goal:** The `selfDestruction` block reaches the domain model with an unambiguous presence predicate (design D1; PRD FR-1.1, FR-1.2, FR-1.3, FR-1.5).

**Interfaces**

- Consumes: the `{0,-1,-1}` absent sentinel from Task 1.
- Produces:
  - `information.SelfDestruction` with `Present() bool`, `Action() byte`, `RemoveAfter() int32`, `Hp() int32`, `OnHpThreshold() bool`, `OnTimer() bool`
  - `information.NewSelfDestruction(present bool, action byte, removeAfter int32, hp int32) SelfDestruction`
  - `information.Model.SelfDestruction() SelfDestruction`
  - `(*information.ModelBuilder).SetSelfDestruction(SelfDestruction) *ModelBuilder`

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/information/model.go` — the value type, the `Model` field and accessor
- `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go` — `SetSelfDestruction` + `Build` wiring
- `services/atlas-monsters/atlas.com/monsters/monster/information/rest.go` — `Extract` mapping (the `selfDestruction` DTO already exists at line 56)
- `services/atlas-monsters/atlas.com/monsters/monster/information/self_destruction_test.go` — new file

Module root: `services/atlas-monsters/atlas.com/monsters`.

- [x] **Step 1: Write the failing tests**

New file `self_destruction_test.go`, package `information`. No fixtures or servers needed — `Extract` takes a `RestModel` literal.

`TestSelfDestructionPredicates` — table-driven over constructed `SelfDestruction` values:

| case | present | action | removeAfter | hp | want `Present` | want `OnHpThreshold` | want `OnTimer` |
|---|---|---|---|---|---|---|---|
| absent | false | 0 | -1 | -1 | false | false | false |
| hp threshold (Boomer 5100002) | true | 1 | -1 | 1800 | true | true | false |
| timer only (9300166) | true | 4 | 0 | -1 | true | false | true |
| both (9400566) | true | 3 | 5 | 1 | true | true | false |
| present, hp 0 | true | 1 | -1 | 0 | true | true | false |

`TestExtractMapsSelfDestruction` — table-driven over `RestModel{SelfDestruction: selfDestruction{...}}`:

| case | DTO `{Action, RemoveAfter, Hp}` | want `Present()` | want `Action()` | want `RemoveAfter()` | want `Hp()` |
|---|---|---|---|---|---|
| absent block | `{0, -1, -1}` | false | 0 | -1 | -1 |
| Boomer | `{1, -1, 1800}` | true | 1 | -1 | 1800 |
| timer mob | `{4, 0, -1}` | true | 4 | 0 | -1 |
| legacy absent (pre-Task-1 atlas-data) | `{0, 0, 0}` | false | 0 | 0 | 0 |

The last row pins design D2's rolling-deploy claim: under the OLD `{0,0,0}` sentinel the predicate is still false, so an old atlas-data reports "not self-destructing" for everything.

`TestBuilderSetsSelfDestruction` — `NewModelBuilder().SetSelfDestruction(NewSelfDestruction(true, 3, -1, 5000)).Build().SelfDestruction()` must report `Present() == true`, `Action() == 3`, `Hp() == 5000`.

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/information/ -run 'SelfDestruction' -v`
Expected: FAIL — `undefined: NewSelfDestruction`, `m.SelfDestruction undefined`.

- [x] **Step 3: Add the value type and the model field**

In `model.go`, add the field `selfDestruction SelfDestruction` to `Model` (after `attacks`), and:

```go
// SelfDestruction is a monster's WZ `selfDestruction` block. Present is
// carried explicitly because hp == -1 is BOTH "no HP threshold" and the
// absent-block default (task-253 design §2.6): without it, a timer-driven
// mob and an ordinary monster are indistinguishable.
type SelfDestruction struct {
	present     bool
	action      byte
	removeAfter int32
	hp          int32
}

func NewSelfDestruction(present bool, action byte, removeAfter int32, hp int32) SelfDestruction {
	return SelfDestruction{present: present, action: action, removeAfter: removeAfter, hp: hp}
}

func (s SelfDestruction) Present() bool      { return s.present }
func (s SelfDestruction) Action() byte       { return s.action }
func (s SelfDestruction) RemoveAfter() int32 { return s.removeAfter }
func (s SelfDestruction) Hp() int32          { return s.hp }

// OnHpThreshold reports the HP-driven mechanic: the mob detonates once its
// post-damage HP falls to or below Hp().
func (s SelfDestruction) OnHpThreshold() bool { return s.present && s.hp > -1 }

// OnTimer reports the timer-driven mechanic: the mob detonates RemoveAfter
// seconds after spawn, with no HP predicate.
func (s SelfDestruction) OnTimer() bool { return s.present && s.hp <= -1 }

func (m Model) SelfDestruction() SelfDestruction {
	return m.selfDestruction
}
```

- [x] **Step 4: Add the builder setter**

In `builder.go`, add `selfDestruction SelfDestruction` to `ModelBuilder`, then:

```go
// SetSelfDestruction sets the WZ selfDestruction block on the builder. Used by
// tests driving the HP-threshold and timer detonation paths.
func (b *ModelBuilder) SetSelfDestruction(sd SelfDestruction) *ModelBuilder {
	b.selfDestruction = sd
	return b
}
```

and add `selfDestruction: b.selfDestruction,` to the `Model{...}` literal in `Build`.

- [x] **Step 5: Map it in `Extract`**

In `rest.go`, inside `Extract`, before the `return`:

```go
	// Presence: the absent block and the present-but-omitted-field defaults are
	// both -1 (atlas-data monster/reader.go getSelfDestruction, task-253 D2), so
	// a block is present iff at least one of the two predicates was written.
	sdPresent := rm.SelfDestruction.Hp > -1 || rm.SelfDestruction.RemoveAfter > -1
```

and add to the returned `Model{...}` literal:

```go
		selfDestruction: NewSelfDestruction(sdPresent, rm.SelfDestruction.Action, rm.SelfDestruction.RemoveAfter, rm.SelfDestruction.Hp),
```

- [x] **Step 6: Run the tests to verify they pass**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./monster/information/`
Expected: PASS

- [x] **Step 7: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/information/
git commit -m "feat(atlas-monsters): carry selfDestruction on information.Model"
```

---

### Task 5: `atlas-monsters` — `Registry.SelfDestruct` atomic transition

**Goal:** One atomic compare-and-set drives a monster to 0 HP and reports `Killed` to exactly one caller, so concurrent triggers emit exactly one `KILLED` (design D3; PRD FR-2.2, FR-5.5).

**Interfaces**

- Produces: `func (r *Registry) SelfDestruct(t tenant.Model, uniqueId uint32) (DamageSummary, error)`. `Killed` is true only for the caller that performed the HP > 0 → 0 transition. `Monster` is the post-transition model. `CharacterId` is 0 and `VisibleDamage` is 0 — a detonation is not damage.

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/registry.go` — new method, next to `ApplyDamage` (line 436)
- `services/atlas-monsters/atlas.com/monsters/monster/self_destruct_registry_test.go` — new file
- `services/atlas-monsters/atlas.com/monsters/monster/model.go:36` — read-only; `DamageSummary`

Module root: `services/atlas-monsters/atlas.com/monsters`.

Patterns to copy: `registry.go:436-492` (`ApplyDamage`'s `r.reg.Update` + `fromStored` shape), `registry_test.go:26-51` (`TestMain` with miniredis, `testContext`, `r.Clear(ctx)`, `r.CreateMonster(...)`).

- [x] **Step 1: Write the failing tests**

New file `self_destruct_registry_test.go`, package `monster`. It reuses the package's existing `TestMain` (miniredis + `producertest.InstallNoop()`) and the `testContext` helper — do not add another `TestMain`.

`TestRegistrySelfDestructTransitionsOnce`:
- tenant `tenant.Create(uuid.New(), "GMS", 83, 1)`; `field.NewBuilder(0, 0, 40000).Build()`
- `m := r.CreateMonster(ctx, ten, f, 5100002, 0, 0, 0, 5, 0, 4000, 0, "", "")`
- first `r.SelfDestruct(ten, m.UniqueId())` → `err == nil`, `s.Killed == true`, `s.Monster.Hp() == 0`, `s.Monster.Alive() == false`
- second call on the same uniqueId → `err == nil`, `s.Killed == false`, `s.Monster.Hp() == 0`

`TestRegistrySelfDestructLeavesDamageEntries`:
- create the mob, then `r.ApplyDamage(ten, 777, 100, m.UniqueId(), time.Now().UnixMilli())`
- `r.SelfDestruct(ten, m.UniqueId())` → the returned `s.Monster.DamageSummary()` must still hold exactly one entry with `CharacterId == 777` and `Damage == 100`, and `s.Monster.DamageLeader()` must be `777`. A detonation must not rewrite the damage leader.

`TestRegistrySelfDestructUnknownMonster`:
- `r.SelfDestruct(ten, 999999)` → a non-nil error; assert the call does not panic and `s.Killed == false`.

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestRegistrySelfDestruct' -v`
Expected: FAIL — `r.SelfDestruct undefined`.

- [x] **Step 3: Implement `SelfDestruct`**

Add to `registry.go` immediately after `ApplyDamage`:

```go
// SelfDestruct atomically drives the monster to 0 HP. Killed is true for the
// caller that performed the transition and false for every caller that finds
// it already at 0 — which is what makes concurrent triggers (two damage lines,
// a timer racing a bomb report) emit exactly one KILLED. Damage entries are
// left untouched: a detonation is not damage and must not rewrite the damage
// leader (task-253 design D3).
//
// This deliberately does NOT reuse ApplyDamage, whose Killed is `m.Hp() == 0`
// and is therefore true for every concurrent caller once HP has reached zero.
func (r *Registry) SelfDestruct(t tenant.Model, uniqueId uint32) (DamageSummary, error) {
	ctx := context.Background()

	var transitioned bool
	sm, err := r.reg.Update(ctx, t, uniqueId, func(cur storedMonster) storedMonster {
		transitioned = cur.Hp > 0
		cur.Hp = 0
		return cur
	})
	if err != nil {
		return DamageSummary{}, errMonsterNotFound
	}

	_, m, err := fromStored(sm)
	if err != nil {
		return DamageSummary{}, err
	}

	return DamageSummary{
		Monster: m,
		Killed:  transitioned,
	}, nil
}
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./monster/ -run 'TestRegistrySelfDestruct'`
Expected: PASS

- [x] **Step 5: Run the full package suite**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/`
Expected: PASS

- [x] **Step 6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/registry.go services/atlas-monsters/atlas.com/monsters/monster/self_destruct_registry_test.go
git commit -m "feat(atlas-monsters): add Registry.SelfDestruct exactly-once HP transition"
```

---

### Task 6: `atlas-monsters` — extract the kill epilogue and put `deathType` on the wire

**Goal:** One shared post-kill epilogue, and `KILLED`/`DESTROYED` carry the animation byte (design D5, D9; PRD FR-4.1, FR-4.2, FR-6.5).

**Interfaces**

- Produces:
  - `monster.DeathTypeUnset byte = 0`, `monster.DeathTypeFadeOut byte = 1`
  - `func (p *ProcessorImpl) finalizeKill(m Model, killerId uint32, isBoss bool, revives []uint32, deathType byte)`
  - `killedStatusEventProvider(m Model, killerId uint32, boss bool, damageSummary []entry, deathType byte)`
  - `destroyedStatusEventProvider(m Model, deathType byte)`
  - `statusEventKilledBody.DeathType byte \`json:"deathType"\`` and the same on `statusEventDestroyedBody`

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/kafka.go:94,132` — `DeathType` on both bodies, `DeathType*` constants
- `services/atlas-monsters/atlas.com/monsters/monster/producer.go:31,149` — both providers take `deathType`
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go:553,1340` — `finalizeKill` extraction, `Destroy` call site
- `services/atlas-monsters/atlas.com/monsters/monster/producer_test.go` — new assertions

Module root: `services/atlas-monsters/atlas.com/monsters`.

Patterns to copy: `producer_test.go:14-44` (`TestStartControlBodyEncodesControllerHasAggro` — build a `Model` via `Clone(NewMonster(...))`, call the provider, `json.Unmarshal` into an anonymous envelope struct).

- [x] **Step 1: Write the failing tests**

Append to `producer_test.go`.

`TestKilledBodyCarriesDeathType` — table-driven; monster built as
`Clone(NewMonster(field.NewBuilder(0,0,40000).Build(), 1, 5100002, 0, 0, 0, 5, 0, 100, 50, "", "")).Build()`,
provider `killedStatusEventProvider(m, 42, false, nil, deathType)`, envelope
`struct{ Type string \`json:"type"\`; Body statusEventKilledBody \`json:"body"\` }`:

| case | `deathType` arg | want `env.Type` | want `env.Body.DeathType` | want `env.Body.ActorId` |
|---|---|---|---|---|
| ordinary kill | `DeathTypeFadeOut` | `EventMonsterStatusKilled` | 1 | 42 |
| self-destruct action 3 | `3` | `EventMonsterStatusKilled` | 3 | 42 |
| no killer | `5` | `EventMonsterStatusKilled` | 5 | 0 (pass `killerId` 0) |

`TestDestroyedBodyCarriesDeathType` — `destroyedStatusEventProvider(m, DeathTypeUnset)` → `env.Body.DeathType == 0` and `env.Type == EventMonsterStatusDestroyed`.

`TestKilledBodyDeathTypeIsOmittedShapeCompatible` — unmarshal the raw JSON
`{"x":0,"y":0,"actorId":9,"boss":false,"damageEntries":null}` into
`statusEventKilledBody`; `DeathType` must be `0`. This pins design D9's
rolling-deploy claim: an omitting producer's event decodes to 0.

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'DeathType' -v`
Expected: FAIL — `undefined: DeathTypeFadeOut`, `too many arguments in call to killedStatusEventProvider`.

- [x] **Step 3: Add the constants and body fields**

In `kafka.go`, add to the existing `const (...)` block:

```go
	// DeathType* mirror libs/atlas-packet monster/clientbound DestroyType, which
	// is CMob::m_nDeadType. DeathTypeUnset is what an omitting producer sends
	// (rolling deploy); every consumer maps it to fade-out, so the wire stays
	// byte-identical to pre-task-253 behaviour. Wire dead-type 0 ("remove with
	// no animation") is therefore inexpressible through this event — nothing
	// emits it and no WZ selfDestruction.action uses it (task-253 design D9).
	DeathTypeUnset   byte = 0
	DeathTypeFadeOut byte = 1
```

Add `DeathType byte \`json:"deathType"\`` as the last field of both `statusEventKilledBody` and `statusEventDestroyedBody`.

- [x] **Step 4: Thread `deathType` through the providers**

In `producer.go`:

```go
func destroyedStatusEventProvider(m Model, deathType byte) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusDestroyed, statusEventDestroyedBody{ActorId: 0, DeathType: deathType}, m.SpawnSourceType(), m.SpawnSourceId())
}
```

and add `deathType byte` as the last parameter of `killedStatusEventProvider`, setting `DeathType: deathType` in `statusEventKilledBody{...}`.

- [x] **Step 5: Extract `finalizeKill` and update the call sites**

In `processor.go`, add above `damageCore`:

```go
// finalizeKill is the single kill epilogue. Every death — ordinary damage,
// Mortal Blow, and all three self-destruct triggers — runs exactly this
// sequence, so a self-destruct cannot drift from an ordinary kill (task-253
// design D5, FR-6.5). deathType is the wire dead-type the channel renders;
// ordinary deaths pass DeathTypeFadeOut.
func (p *ProcessorImpl) finalizeKill(m Model, killerId uint32, isBoss bool, revives []uint32, deathType byte) {
	GetCooldownRegistry().ClearCooldowns(p.ctx, p.t, m.UniqueId())
	GetAttackCooldownRegistry().ClearCooldowns(p.ctx, p.t, m.UniqueId())
	GetDropTimerRegistry().Unregister(p.ctx, p.t, m.UniqueId())

	// Emit cancellation events for any active status effects before death
	for _, se := range m.StatusEffects() {
		_ = p.emit(EnvEventTopicMonsterStatus, statusEffectCancelledEventProvider(m, se))
	}

	if err := p.emit(EnvEventTopicMonsterStatus, killedStatusEventProvider(m, killerId, isBoss, m.DamageSummary(), deathType)); err != nil {
		p.l.WithError(err).Errorf("Monster [%d] killed, but unable to display that for the characters in the field.", m.UniqueId())
	}
	if _, err := GetMonsterRegistry().RemoveMonster(p.ctx, p.t, m.UniqueId()); err != nil {
		p.l.WithError(err).Errorf("Monster [%d] killed, but not removed from registry.", m.UniqueId())
	}

	// Boss revive: spawn next phase monsters
	if len(revives) > 0 {
		p.spawnRevives(m, revives)
	}
}
```

Replace the whole body of `damageCore`'s `if killed { ... return }` block with:

```go
	if killed {
		p.finalizeKill(last.Monster, last.CharacterId, isBoss, revives, DeathTypeFadeOut)
		return
	}
```

In `Destroy` (line 1340), change the emit to `destroyedStatusEventProvider(m, DeathTypeUnset)`.

- [x] **Step 6: Run the tests to verify they pass**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./monster/`
Expected: PASS. `processor_test.go`'s existing kill assertions must still pass unchanged — the epilogue is a pure extraction.

- [x] **Step 7: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/kafka.go services/atlas-monsters/atlas.com/monsters/monster/producer.go services/atlas-monsters/atlas.com/monsters/monster/processor.go services/atlas-monsters/atlas.com/monsters/monster/producer_test.go
git commit -m "refactor(atlas-monsters): extract the kill epilogue and carry deathType on KILLED/DESTROYED"
```

---

### Task 7: `atlas-monsters` — HP-threshold detonation and the `SelfDestruct` processor API

**Goal:** Crossing `selfDestruction.hp` on the damage path detonates the mob exactly once with its WZ animation, and every other trigger gets a single authoritative entry point (design D4, D5, D7; PRD FR-2.1, FR-2.2, FR-2.3, FR-2.5, FR-2.6, FR-6.2, FR-6.3).

**Interfaces**

- Consumes: `information.SelfDestruction` (Task 4), `Registry.SelfDestruct` (Task 5), `finalizeKill` + `DeathTypeFadeOut` (Task 6).
- Produces:
  - `type SelfDestructTrigger string` with `TriggerThreshold`, `TriggerTimer`, `TriggerContact`
  - `Processor.SelfDestruct(uniqueId uint32, characterId uint32, trigger SelfDestructTrigger)` on the interface and `*ProcessorImpl`
  - `func (p *ProcessorImpl) selfDestructFrom(m Model, characterId uint32, action byte, trigger SelfDestructTrigger)`
  - `func (p *ProcessorImpl) monsterInformation(monsterId uint32) (information.Model, error)`

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/processor.go:32,553,1751` — interface, `damageCore`, `Kill`
- `services/atlas-monsters/atlas.com/monsters/monster/self_destruct_test.go` — new file
- `services/atlas-monsters/atlas.com/monsters/monster/information/model.go` — read-only; the predicates

Module root: `services/atlas-monsters/atlas.com/monsters`.

Patterns to copy: `processor_test.go:1594-1600` (swapping `testInformationLookup` and restoring it with `defer`), `registry_test.go:51-70` (miniredis-backed monster creation).

- [x] **Step 1: Write the failing tests**

New file `self_destruct_test.go`, package `monster`. Every test swaps `testInformationLookup` and restores it with `defer`, and asserts emitted events by installing a recording `emit` on a directly-constructed `&ProcessorImpl{l: l, ctx: ctx, t: ten, emit: rec}` — the same pattern `processor_test.go` uses for kill assertions.

Boomer template: monsterId `5100002`, max HP `4000`, `selfDestruction` `{action: 1, removeAfter: -1, hp: 1800}`.

`TestDamageCoreCrossingThresholdDetonates` — table-driven:

| case | mob HP at start | `selfDestruction` | damage lines | want KILLED count | want `DeathType` | want `ActorId` |
|---|---|---|---|---|---|---|
| crosses threshold | 4000 | `{1, -1, 1800}` | `[]uint32{2300}` | 1 | 1 | 55 |
| lands exactly on threshold | 4000 | `{1, -1, 1800}` | `[]uint32{2200}` | 1 | 1 | 55 |
| stays above threshold | 4000 | `{1, -1, 1800}` | `[]uint32{100}` | 0 | — | — |
| multi-line, crosses on line 2 | 4000 | `{1, -1, 1800}` | `[]uint32{1500, 1500}` | 1 | 1 | 55 |
| already below threshold, next hit | 1000 | `{1, -1, 1800}` | `[]uint32{1}` | 1 | 1 | 55 |
| ordinary kill wins (damage reaches 0) | 4000 | `{1, -1, 1800}` | `[]uint32{4000}` | 1 | 1 (fade-out, ordinary path) | 55 |
| no block — regression | 4000 | `{0, -1, -1}` | `[]uint32{4000}` | 1 | 1 | 55 |
| no block, partial damage — regression | 4000 | `{0, -1, -1}` | `[]uint32{3999}` | 0 | — | — |

For each case: create the monster with the given HP, call
`p.damageCore(m, 55, damages)`, then count `EventMonsterStatusKilled` messages
on the recorded emits and unmarshal the body. Every case must also assert the
monster is absent from the registry afterwards when a KILLED was emitted
(`GetMonsterRegistry().GetMonster(ten, uid)` returns an error), and present
when none was.

`TestSelfDestructRejects` — table-driven over `p.SelfDestruct(uniqueId, 0, TriggerContact)`:

| case | setup | want KILLED count |
|---|---|---|
| unknown monster | uniqueId `999999`, never created | 0 |
| already dead | created then `GetMonsterRegistry().SelfDestruct` called first | 0 |
| no `selfDestruction` block | created; info returns `{0, -1, -1}` | 0 |
| information lookup fails | created; `testInformationLookup` returns an error | 0 |
| valid target | created, alive, info `{3, -1, 5000}` | 1, with `DeathType == 3` |

`TestSelfDestructAttributesToDamageLeader` — create the mob, apply
`GetMonsterRegistry().ApplyDamage(ten, 777, 100, uid, now)`, then
`p.SelfDestruct(uid, 0, TriggerTimer)`. The KILLED body's `ActorId` must be
`777` (FR-6.3 damage-leader fallback).

`TestSelfDestructNoDamageEntriesReportsNoKiller` — create the mob, take no
damage, `p.SelfDestruct(uid, 0, TriggerTimer)`. The KILLED body's `ActorId`
must be `0`.

`TestSelfDestructIsIdempotent` — call `p.SelfDestruct(uid, 0, TriggerContact)`
twice on a valid target. Exactly one KILLED across both calls.

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'SelfDestruct|Threshold' -v`
Expected: FAIL — `undefined: TriggerContact`, `p.SelfDestruct undefined`.

- [x] **Step 3: Add the trigger type and the information seam**

In `processor.go`, add near the top-level declarations:

```go
// SelfDestructTrigger names which of the four detonation paths fired. It is a
// log/observability discriminator only — every trigger runs the identical
// kill (task-253 design D5).
type SelfDestructTrigger string

const (
	TriggerThreshold SelfDestructTrigger = "threshold"
	TriggerTimer     SelfDestructTrigger = "timer"
	TriggerContact   SelfDestructTrigger = "contact"
)

// monsterInformation resolves a monster template's information, honouring the
// test-only lookup override. Kill already routed through the override; the
// self-destruct paths and damageCore need the same seam because the threshold
// and the animation both come from this data.
func (p *ProcessorImpl) monsterInformation(monsterId uint32) (information.Model, error) {
	if testInformationLookup != nil {
		return testInformationLookup(monsterId)
	}
	return information.NewProcessor(p.l, p.ctx).GetById(monsterId)
}
```

Replace the `testInformationLookup` branch inside `Kill` (processor.go:1763-1769) with a single `info, infoErr := p.monsterInformation(m.MonsterId())`.

- [x] **Step 4: Add `selfDestructFrom` and the exported `SelfDestruct`**

Add `SelfDestruct(uniqueId uint32, characterId uint32, trigger SelfDestructTrigger)` to the `Processor` interface, in the `// Commands` block after `Kill`. Then:

```go
// SelfDestruct detonates a monster with the animation its WZ selfDestruction
// block specifies. It is the authoritative entry point for every externally
// requested detonation (the SELF_DESTRUCT command, the timer sweep): it
// re-derives every predicate rather than trusting the caller, because the
// contact path originates in a client-controlled packet naming an arbitrary
// id (task-253 design D7/D8). Every rejection is a silent debug-level drop.
func (p *ProcessorImpl) SelfDestruct(uniqueId uint32, characterId uint32, trigger SelfDestructTrigger) {
	m, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)
	if err != nil {
		p.l.Debugf("SELF_DESTRUCT: monster [%d] not found.", uniqueId)
		return
	}
	if !m.Alive() {
		p.l.Debugf("SELF_DESTRUCT: monster [%d] already dead.", uniqueId)
		return
	}
	ma, infoErr := p.monsterInformation(m.MonsterId())
	if infoErr != nil {
		p.l.WithError(infoErr).Debugf("SELF_DESTRUCT: information lookup failed for monster [%d]; dropping.", uniqueId)
		return
	}
	sd := ma.SelfDestruction()
	if !sd.Present() {
		p.l.Debugf("SELF_DESTRUCT: monster [%d] (template [%d]) carries no selfDestruction block; dropping.", uniqueId, m.MonsterId())
		return
	}
	p.selfDestructFrom(m, characterId, sd.Action(), trigger)
}

// selfDestructFrom is the shared detonation epilogue. Callers have already
// established that the mob self-destructs; this owns the exactly-once
// transition and the kill bookkeeping.
func (p *ProcessorImpl) selfDestructFrom(m Model, characterId uint32, action byte, trigger SelfDestructTrigger) {
	s, err := GetMonsterRegistry().SelfDestruct(p.t, m.UniqueId())
	if err != nil {
		p.l.WithError(err).Debugf("Self-destruct of monster [%d] failed; it is likely already gone.", m.UniqueId())
		return
	}
	if !s.Killed {
		p.l.Debugf("Self-destruct of monster [%d] lost the race; another trigger already detonated it.", m.UniqueId())
		return
	}

	// FR-6.3: credit the triggering character, else the damage leader, else
	// nobody. atlas-monster-death tolerates ActorId 0 and spawns unowned drops.
	killerId := characterId
	if killerId == 0 {
		killerId = s.Monster.DamageLeader()
	}

	var isBoss bool
	var revives []uint32
	if ma, infoErr := p.monsterInformation(m.MonsterId()); infoErr == nil {
		isBoss = ma.Boss()
		revives = ma.Revives()
	}

	p.l.Debugf("Monster [%d] (template [%d]) self-destructed via [%s] with action [%d]; credited to character [%d].", m.UniqueId(), m.MonsterId(), trigger, action, killerId)
	p.finalizeKill(s.Monster, killerId, isBoss, revives, action)
}
```

- [x] **Step 5: Wire the threshold check into `damageCore`**

In `damageCore`, extend the information fetch at the top to capture the block:

```go
	// Fetch monster info for boss flag, revives, and the self-destruct block.
	// One lookup per damage event, already cache-served — the threshold check
	// costs the hot path nothing (task-253 design D4).
	var isBoss bool
	var revives []uint32
	var sd information.SelfDestruction
	if ma, infoErr := p.monsterInformation(m.MonsterId()); infoErr == nil {
		isBoss = ma.Boss()
		revives = ma.Revives()
		sd = ma.SelfDestruction()
	}
```

Then, immediately before the `if killed {` block:

```go
	// FR-2.1/2.2/2.5: evaluated once per attack, after every line, and only
	// when the attack did not already kill — so a mob that crosses its
	// threshold produces exactly one death, never two.
	if !killed && sd.OnHpThreshold() && int64(last.Monster.Hp()) <= int64(sd.Hp()) {
		p.selfDestructFrom(last.Monster, last.CharacterId, sd.Action(), TriggerThreshold)
		return
	}
```

- [x] **Step 6: Run the tests to verify they pass**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./monster/`
Expected: PASS

- [x] **Step 7: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go services/atlas-monsters/atlas.com/monsters/monster/self_destruct_test.go
git commit -m "feat(atlas-monsters): detonate on the selfDestruction HP threshold"
```

---

### Task 8: `atlas-monsters` — DoT ticks trip the threshold

**Goal:** A poisoned Boomer that crosses its threshold explodes, while the kill-prevention cap stays exactly as it is (design §2.5; PRD FR-2.4).

**Interfaces**

- Consumes: `Processor.SelfDestruct` and `TriggerThreshold` (Task 7), `information.SelfDestruction` (Task 4).

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/status_task.go:64` — `processDoTTick`
- `services/atlas-monsters/atlas.com/monsters/monster/self_destruct_dot_test.go` — new file
- `services/atlas-monsters/atlas.com/monsters/monster/poison_damage_test.go` — read-only; existing DoT test setup

Module root: `services/atlas-monsters/atlas.com/monsters`.

Patterns to copy: `poison_damage_test.go` (constructing a `StatusExpirationTask` and a poisoned monster), `processor_test.go:1594-1600` (`testInformationLookup` swap).

- [x] **Step 1: Write the failing tests**

New file `self_destruct_dot_test.go`, package `monster`.

`TestDoTTickCrossingThresholdDetonates`:
- Boomer at HP `1830`, `selfDestruction` `{action: 1, removeAfter: -1, hp: 1800}`, a poison effect whose per-tick magnitude is `50` and whose `SourceCharacterId` is `88`.
- Run one `processDoTTick`.
- Expect: the monster is absent from the registry afterwards, and the monster registry's HP before removal reached `1780`. Assert via the emitted events that exactly one `EventMonsterStatusKilled` was produced with `DeathType == 1` and `ActorId == 88`.

`TestDoTTickNotCrossingThresholdDoesNotDetonate`:
- Same mob at HP `4000`, poison magnitude `50`.
- One tick → HP `3950`, monster still present, zero `EventMonsterStatusKilled`.

`TestDoTTickCannotReachZeroHp` (regression on the existing cap):
- Mob with **no** `selfDestruction` block (`{0, -1, -1}`) at HP `30`, poison magnitude `500`.
- One tick → HP is `1`, monster still alive, zero `EventMonsterStatusKilled`. Poison must still be unable to kill.

`TestDoTTickThresholdMobStillCannotBeReducedToZero`:
- Boomer at HP `1810`, `selfDestruction` `{1, -1, 1800}`, poison magnitude `5000`.
- The cap clamps the tick to `1809`, landing on HP `1`, which is `<= 1800` — so the mob detonates. Exactly one `EventMonsterStatusKilled`, `DeathType == 1`.

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'TestDoTTick' -v`
Expected: FAIL — no KILLED is emitted on the crossing cases.

- [x] **Step 3: Add the threshold check to `processDoTTick`**

In `status_task.go`, after the "Emit damaged event" line at the end of `processDoTTick`, append:

```go
	// FR-2.4: DoT never enters damageCore (it caps at currentHp-1 and cannot
	// kill), but the self-destruct threshold is above zero, so a poison tick
	// can still cross it. The kill-prevention cap above is untouched — poison
	// still cannot reduce a mob to 0 HP; it can only trip a detonation
	// (task-253 design §2.5).
	if ma, ierr := information.NewProcessor(t.l, ctx).GetById(m.MonsterId()); ierr == nil {
		sd := ma.SelfDestruction()
		if sd.OnHpThreshold() && int64(ds.Monster.Hp()) <= int64(sd.Hp()) {
			NewProcessor(t.l, ctx).SelfDestruct(m.UniqueId(), se.SourceCharacterId(), TriggerThreshold)
		}
	}
```

Add `"atlas-monsters/monster/information"` to the file's imports.

- [x] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./monster/`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/status_task.go services/atlas-monsters/atlas.com/monsters/monster/self_destruct_dot_test.go
git commit -m "feat(atlas-monsters): trip the self-destruct threshold on DoT ticks"
```

---

### Task 9: `atlas-monsters` — timer-driven self-destruction

**Goal:** A mob with `selfDestruction` but no HP threshold detonates `removeAfter` seconds after spawn, and the timer is cancelled whenever the mob dies or is destroyed first (design D6; PRD FR-3.1 … FR-3.6).

**Interfaces**

- Consumes: `information.SelfDestruction.OnTimer()`/`Action()`/`RemoveAfter()` (Task 4), `Processor.SelfDestruct` + `TriggerTimer` (Task 7), `finalizeKill` (Task 6).
- Produces:
  - `monster.SelfDestructTimerEntry` with `MonsterId() uint32`, `Field() field.Model`, `Action() byte`, `FireAt() time.Time`
  - `monster.InitSelfDestructTimerRegistry(rc *goredis.Client)`, `monster.GetSelfDestructTimerRegistry() *SelfDestructTimerRegistry`
  - `Register(ctx, t, uniqueId, e)`, `Unregister(ctx, t, uniqueId)`, `GetAll(ctx) map[MonsterKey]SelfDestructTimerEntry`
  - `monster.NewSelfDestructTimerTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *SelfDestructTimerTask`

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/self_destruct_timer_registry.go` — new file
- `services/atlas-monsters/atlas.com/monsters/monster/self_destruct_timer_task.go` — new file
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go:217,1340` — arm in `Create`, unregister in `Destroy` and `finalizeKill`
- `services/atlas-monsters/atlas.com/monsters/monster/self_destruct_timer_test.go` — new file
- `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go:26` — add the registry to `TestMain`
- `services/atlas-monsters/atlas.com/monsters/main.go:65,104` — init the registry, register the sweep task

Module root: `services/atlas-monsters/atlas.com/monsters`.

Patterns to copy: `monster/drop_timer_registry.go` in full (entry type, `stored*` struct, `sync.Once` init, `Register`/`Unregister`/`GetAll`, `fromStored*`), `monster/drop_timer_task.go` in full (`Run`, `processEntry`, `SleepTime`).

- [x] **Step 1: Write the failing tests**

New file `self_destruct_timer_test.go`, package `monster`. Timer mob template: monsterId `9300166`, `selfDestruction` `{action: 4, removeAfter: 0, hp: -1}`. `now` is captured once per test and passed to `processEntry` explicitly — never `time.Sleep`.

`TestCreateArmsTimerForTimerMob` — `p.Create(f, RestModel{MonsterId: 9300166, ...})` with `testInformationLookup` returning the timer template. `GetSelfDestructTimerRegistry().GetAll(ctx)` must hold one entry keyed by the new uniqueId, with `Action() == 4`, `MonsterId() == 9300166`, and `FireAt()` not after `time.Now()` (because `removeAfter` is 0).

`TestCreateDoesNotArmTimerForOtherMobs` — table-driven:

| case | `selfDestruction` | want entries |
|---|---|---|
| no block | `{0, -1, -1}` | 0 |
| HP threshold only (Boomer) | `{1, -1, 1800}` | 0 |
| HP threshold and removeAfter | `{3, 5, 1}` | 0 |
| timer only, removeAfter 30 | `{4, 30, -1}` | 1, `FireAt()` ≈ now+30s (±2s) |

`TestSelfDestructTimerTaskFiresOnElapsedEntry` — create the timer mob, arm the entry with `fireAt` in the past, run `processEntry(now, ten, uid, entry)`. Expect exactly one `EventMonsterStatusKilled` with `DeathType == 4` and `ActorId == 0`, the monster absent from the monster registry, and `GetAll` empty afterwards.

`TestSelfDestructTimerTaskSkipsUnelapsedEntry` — `fireAt` one minute in the future. `processEntry` emits nothing and leaves both registries untouched.

`TestSelfDestructTimerTaskUnregistersDeadMob` — arm an elapsed entry, then drive the monster to 0 HP via `GetMonsterRegistry().SelfDestruct`. `processEntry` must emit no KILLED and must leave `GetAll` empty.

`TestSelfDestructTimerTaskUnregistersMissingMob` — arm an elapsed entry for a uniqueId that was never created. `processEntry` emits nothing, `GetAll` is empty.

`TestKillUnregistersTimer` — arm a timer entry for a live mob, then kill it through `p.damageCore(m, 55, []uint32{maxHp})`. `GetAll` must be empty (FR-3.3).

`TestDestroyUnregistersTimer` — arm a timer entry, call `p.Destroy(uid)`. `GetAll` must be empty.

`TestDestroyAllLeavesNoTimers` — arm timers for two mobs in the tenant, call `DestroyAll(l, ctx)`, assert `GetAll(ctx)` is empty (FR-3.6).

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run 'Timer' -v`
Expected: FAIL — `undefined: GetSelfDestructTimerRegistry`.

- [x] **Step 3: Write the registry**

New file `self_destruct_timer_registry.go`, package `monster`. Mirror `drop_timer_registry.go` exactly:

```go
type SelfDestructTimerEntry struct {
	monsterId uint32
	field     field.Model
	action    byte
	fireAt    time.Time
}

func NewSelfDestructTimerEntry(monsterId uint32, f field.Model, action byte, fireAt time.Time) SelfDestructTimerEntry {
	return SelfDestructTimerEntry{monsterId: monsterId, field: f, action: action, fireAt: fireAt}
}

func (e SelfDestructTimerEntry) MonsterId() uint32  { return e.monsterId }
func (e SelfDestructTimerEntry) Field() field.Model { return e.field }
func (e SelfDestructTimerEntry) Action() byte       { return e.action }
func (e SelfDestructTimerEntry) FireAt() time.Time  { return e.fireAt }

type storedSelfDestructTimer struct {
	TenantId           string      `json:"tenantId"`
	TenantRegion       string      `json:"tenantRegion"`
	TenantMajorVersion uint16      `json:"tenantMajorVersion"`
	TenantMinorVersion uint16      `json:"tenantMinorVersion"`
	UniqueId           uint32      `json:"uniqueId"`
	MonsterId          uint32      `json:"monsterId"`
	Field              field.Model `json:"field"`
	Action             byte        `json:"action"`
	FireAtMs           int64       `json:"fireAtMs"`
}

// SelfDestructTimerRegistry is tenant-scoped: the stored key is
// atlas:self-destruct-timer:<tenantId>:<region>:<major>.<minor>:<uniqueId>.
// GetAll is the one genuine cross-tenant operation — the periodic sweep has no
// tenant to loop over — and uses the explicit GetAllAcrossTenants sibling, the
// same shape DropTimerRegistry uses.
type SelfDestructTimerRegistry struct {
	reg *atlasredis.TenantRegistry[uint32, storedSelfDestructTimer]
}

var (
	selfDestructTimerRegistry *SelfDestructTimerRegistry
	selfDestructTimerOnce     sync.Once
)

func InitSelfDestructTimerRegistry(rc *goredis.Client) {
	selfDestructTimerOnce.Do(func() {
		reg := atlasredis.NewTenantRegistry[uint32, storedSelfDestructTimer](rc, "self-destruct-timer", func(id uint32) string { return strconv.FormatUint(uint64(id), 10) })
		selfDestructTimerRegistry = &SelfDestructTimerRegistry{reg: reg}
	})
}

func GetSelfDestructTimerRegistry() *SelfDestructTimerRegistry {
	return selfDestructTimerRegistry
}

func (r *SelfDestructTimerRegistry) Register(ctx context.Context, t tenant.Model, uniqueId uint32, e SelfDestructTimerEntry) {
	_ = r.reg.Put(ctx, t, uniqueId, storedSelfDestructTimer{
		TenantId:           t.Id().String(),
		TenantRegion:       t.Region(),
		TenantMajorVersion: t.MajorVersion(),
		TenantMinorVersion: t.MinorVersion(),
		UniqueId:           uniqueId,
		MonsterId:          e.monsterId,
		Field:              e.field,
		Action:             e.action,
		FireAtMs:           e.fireAt.UnixMilli(),
	})
}

func (r *SelfDestructTimerRegistry) Unregister(ctx context.Context, t tenant.Model, uniqueId uint32) {
	_ = r.reg.Remove(ctx, t, uniqueId)
}

func (r *SelfDestructTimerRegistry) GetAll(ctx context.Context) map[MonsterKey]SelfDestructTimerEntry {
	result := make(map[MonsterKey]SelfDestructTimerEntry)
	items, err := r.reg.GetAllAcrossTenants(ctx)
	if err != nil {
		return result
	}
	for _, sd := range items {
		t, entry := fromStoredSelfDestructTimer(sd)
		result[MonsterKey{Tenant: t, MonsterId: sd.UniqueId}] = entry
	}
	return result
}

func fromStoredSelfDestructTimer(sd storedSelfDestructTimer) (tenant.Model, SelfDestructTimerEntry) {
	tid, _ := uuid.Parse(sd.TenantId)
	t, _ := tenant.Create(tid, sd.TenantRegion, sd.TenantMajorVersion, sd.TenantMinorVersion)
	return t, SelfDestructTimerEntry{
		monsterId: sd.MonsterId,
		field:     sd.Field,
		action:    sd.Action,
		fireAt:    time.UnixMilli(sd.FireAtMs),
	}
}
```

Imports match `drop_timer_registry.go`'s.

- [x] **Step 4: Write the sweep task**

New file `self_destruct_timer_task.go`, package `monster`:

```go
// SelfDestructTimerTask sweeps armed self-destruct timers. It is registered
// alongside NewDropTimerTask in registerSweepTasks, so it is leader-gated when
// leader election is on; when it is not, a double fire is a no-op because
// Registry.SelfDestruct decides the transition exactly once (design D3/D6).
type SelfDestructTimerTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
}

func NewSelfDestructTimerTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *SelfDestructTimerTask {
	l.Infof("Initializing self-destruct timer task to run every %dms.", interval.Milliseconds())
	return &SelfDestructTimerTask{l: l, ctx: ctx, interval: interval}
}

func (t *SelfDestructTimerTask) Run() {
	now := time.Now()
	for key, entry := range GetSelfDestructTimerRegistry().GetAll(t.ctx) {
		t.processEntry(now, key.Tenant, key.MonsterId, entry)
	}
}

func (t *SelfDestructTimerTask) processEntry(now time.Time, ten tenant.Model, uniqueId uint32, e SelfDestructTimerEntry) {
	if now.Before(e.FireAt()) {
		return
	}

	m, err := GetMonsterRegistry().GetMonster(ten, uniqueId)
	if err != nil || !m.Alive() {
		GetSelfDestructTimerRegistry().Unregister(t.ctx, ten, uniqueId)
		return
	}

	tctx := tenant.WithContext(t.ctx, ten)
	NewProcessor(t.l, tctx).SelfDestruct(uniqueId, 0, TriggerTimer)
}

func (t *SelfDestructTimerTask) SleepTime() time.Duration {
	return t.interval
}
```

- [x] **Step 5: Arm and cancel the timer in the processor**

In `Create` (`processor.go:217`), after the existing friendly/drop-period block:

```go
	// FR-3.1/3.5: arm the detonation timer for a mob whose selfDestruction
	// block carries no HP predicate. removeAfter is read as SECONDS, matching
	// Cosmic MapleMap.java:1868 (task-253 design §2.4); a value <= 0 yields a
	// fireAt of now, which the next sweep tick fires.
	if sd := ma.SelfDestruction(); sd.OnTimer() {
		delay := sd.RemoveAfter()
		if delay < 0 {
			delay = 0
		}
		p.l.Debugf("Arming self-destruct timer for monster [%d] (template [%d]) in [%d]s with action [%d].", m.UniqueId(), m.MonsterId(), delay, sd.Action())
		GetSelfDestructTimerRegistry().Register(p.ctx, p.t, m.UniqueId(), NewSelfDestructTimerEntry(m.MonsterId(), f, sd.Action(), time.Now().Add(time.Duration(delay)*time.Second)))
	}
```

In `Destroy` (`processor.go:1340`), add as the first statement:

```go
	GetSelfDestructTimerRegistry().Unregister(p.ctx, p.t, uniqueId)
```

In `finalizeKill` (Task 6), add after the `GetDropTimerRegistry().Unregister(...)` line:

```go
	GetSelfDestructTimerRegistry().Unregister(p.ctx, p.t, m.UniqueId())
```

- [x] **Step 6: Wire the registry and task into `main.go` and `TestMain`**

In `main.go`, add `monster.InitSelfDestructTimerRegistry(rc)` immediately after `monster.InitDropTimerRegistry(rc)` (line 65), and inside `registerSweepTasks` add
`tasks.Register(l, ctx)(monster.NewSelfDestructTimerTask(l, ctx, time.Second))` immediately after the `NewDropTimerTask` line (line 104).

In `registry_test.go`'s `TestMain`, add `InitSelfDestructTimerRegistry(rc)` after `InitDropTimerRegistry(rc)`.

- [x] **Step 7: Run the tests to verify they pass**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...`
Expected: PASS

- [x] **Step 8: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/self_destruct_timer_registry.go services/atlas-monsters/atlas.com/monsters/monster/self_destruct_timer_task.go services/atlas-monsters/atlas.com/monsters/monster/self_destruct_timer_test.go services/atlas-monsters/atlas.com/monsters/monster/processor.go services/atlas-monsters/atlas.com/monsters/monster/registry_test.go services/atlas-monsters/atlas.com/monsters/main.go
git commit -m "feat(atlas-monsters): add the timer-driven self-destruct sweep"
```

---

### Task 10: `atlas-monsters` — the `SELF_DESTRUCT` command arm

**Goal:** atlas-channel can request a detonation over the existing monster command topic (design D7; PRD FR-5.4, FR-5.6).

**Interfaces**

- Consumes: `Processor.SelfDestruct` + `TriggerContact` (Task 7).
- Produces: `CommandTypeSelfDestruct = "SELF_DESTRUCT"` and `selfDestructCommandBody{CharacterId uint32}` in the consumer contract; `handleSelfDestructCommand` registered on `EnvCommandTopic`.

### Files

- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go:31` — command type + body
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go:57,189` — handler + registration
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka_test.go` — new decode assertion

Module root: `services/atlas-monsters/atlas.com/monsters`.

Patterns to copy: `consumer.go:189-196` (`handleKillCommand`), `consumer.go:54-56` (its `rf(t, message.AdaptHandler(message.PersistentConfig(...)))` registration), `kafka.go:128-139` (`killCommandBody` and its "why no extra fields" comment).

- [x] **Step 1: Write the failing test**

Append to `kafka_test.go`:

`TestSelfDestructCommandUnmarshal` — unmarshal

```json
{"worldId":0,"channelId":1,"mapId":100000000,"monsterId":7001,"type":"SELF_DESTRUCT","body":{"characterId":4242}}
```

into `command[selfDestructCommandBody]` and assert `Type == CommandTypeSelfDestruct`, `MonsterId == 7001`, `Body.CharacterId == 4242`.

`TestSelfDestructCommandTypeValue` — assert `CommandTypeSelfDestruct == "SELF_DESTRUCT"`, so the channel's mirror constant cannot drift silently.

- [x] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./kafka/consumer/monster/ -run SelfDestruct -v`
Expected: FAIL — `undefined: CommandTypeSelfDestruct`.

- [x] **Step 3: Add the contract**

In `kafka/consumer/monster/kafka.go`, add `CommandTypeSelfDestruct = "SELF_DESTRUCT"` to the command-type const block, and:

```go
// selfDestructCommandBody asks the processor to detonate a self-destructing
// monster with the animation its WZ selfDestruction block specifies.
// CharacterId is the character who reported the contact (MONSTER_BOMB), or 0
// when there is none; the processor re-derives every predicate itself, so this
// carries no animation and no reason discriminator — the animation must never
// come from a client-influenceable field (task-253 design D7).
//
// CharacterId matches the field name and type used by damageCommandBody,
// killCommandBody and forceControlCommandBody, so it introduces no unmarshal
// collision on the shared command topic.
type selfDestructCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}
```

- [x] **Step 4: Add the handler and register it**

In `consumer.go`, after `handleKillCommand`:

```go
func handleSelfDestructCommand(l logrus.FieldLogger, ctx context.Context, c command[selfDestructCommandBody]) {
	if c.Type != CommandTypeSelfDestruct {
		return
	}

	monster.NewProcessor(l, ctx).SelfDestruct(c.MonsterId, c.Body.CharacterId, monster.TriggerContact)
}
```

and register it in `InitHandlers` immediately after the `handleKillCommand` registration:

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSelfDestructCommand))); err != nil {
			return err
		}
```

- [x] **Step 5: Run the tests to verify they pass**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...`
Expected: PASS

- [x] **Step 6: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/
git commit -m "feat(atlas-monsters): consume the SELF_DESTRUCT monster command"
```

---

### Task 11: `atlas-channel` — command contract, producer, and the `deathType` passthrough

**Goal:** The channel can emit `SELF_DESTRUCT` and writes the event's animation into `MonsterDestroy` instead of a hardcoded fade-out (design D7, D9; PRD FR-4.3, FR-5.4, FR-5.6).

**Interfaces**

- Consumes: `DestroyType` constants and the version gate (Task 2); the `deathType` field on `KILLED`/`DESTROYED` (Task 6); `CommandTypeSelfDestruct` / `{"characterId": …}` (Task 10).
- Produces:
  - `monster2.CommandTypeSelfDestruct = "SELF_DESTRUCT"`, `monster2.SelfDestructCommandBody{CharacterId uint32}`
  - `monster2.StatusEventKilledBody.DeathType byte` and `monster2.StatusEventDestroyedBody.DeathType byte`
  - `monster.SelfDestructCommandProvider(f field.Model, monsterId uint32, characterId uint32) model.Provider[[]kafka.Message]`
  - `monster.Processor.SelfDestruct(f field.Model, monsterId uint32, characterId uint32) error`
  - `destroyTypeFor(deathType byte) monsterpkt.DestroyType` (unexported, consumer package)

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go:22,181,214` — command type, body, `DeathType` on both status bodies
- `services/atlas-channel/atlas.com/channel/monster/producer.go:175` — `SelfDestructCommandProvider`
- `services/atlas-channel/atlas.com/channel/monster/processor.go:31,155` — interface method + impl
- `services/atlas-channel/atlas.com/channel/monster/mock/processor.go:25,130` — `SelfDestructFunc`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:191,291` — `destroyTypeFor`, both `*ForSession` operators
- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go` — new tests

Module root: `services/atlas-channel/atlas.com/channel`.

Patterns to copy: `monster/producer.go:175-189` (`KillCommandProvider`), `monster/processor.go:155-158` (`Kill`), `monster/mock/processor.go:25,130-133` (mock field + method).

- [x] **Step 1: Write the failing tests**

Append to `kafka/consumer/monster/consumer_test.go`.

`TestDestroyTypeFor` — table-driven over `destroyTypeFor(deathType)`:

| case | input `deathType` | want |
|---|---|---|
| producer omitted the field | 0 | `monsterpkt.DestroyTypeFadeOut` (1) |
| ordinary fade-out | 1 | `monsterpkt.DestroyTypeFadeOut` (1) |
| bomb | 2 | `monsterpkt.DestroyTypeBomb` (2) |
| destruct-by-miss | 3 | `monsterpkt.DestroyTypeDestructByMiss` (3) |
| swallow | 4 | `monsterpkt.DestroyTypeSwallow` (4) |
| self-destruct | 5 | `monsterpkt.DestroyTypeSelfDestruct` (5) |

`TestStatusEventKilledBodyDecodesDeathType` — unmarshal
`{"x":0,"y":0,"actorId":9,"boss":false,"damageEntries":null,"deathType":3}`
into `monster2.StatusEventKilledBody` and assert `DeathType == 3`; then
unmarshal the same JSON with `"deathType"` removed and assert `DeathType == 0`
(rolling-deploy compatibility, design D9).

`TestSelfDestructCommandProviderShape` — call
`monster.SelfDestructCommandProvider(field.NewBuilder(0, 1, 100000000).Build(), 7001, 4242)()`,
expect one message, and unmarshal its value into
`monster2.Command[monster2.SelfDestructCommandBody]`. Assert `Type == monster2.CommandTypeSelfDestruct`,
`MonsterId == 7001`, `Body.CharacterId == 4242`, `WorldId == 0`, `ChannelId == 1`,
`MapId == 100000000`, and that the message key equals `producer.CreateKey(7001)`
— the same key `DamageCommandProvider` uses, so a contact report is ordered
behind the damage that preceded it (design §7). Put this test in
`services/atlas-channel/atlas.com/channel/monster/producer_test.go` if that
file exists; otherwise create it in package `monster`.

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/monster/ ./monster/ -run 'DestroyTypeFor|DeathType|SelfDestruct' -v`
Expected: FAIL — `undefined: destroyTypeFor`, `undefined: SelfDestructCommandProvider`.

- [x] **Step 3: Extend the message contract**

In `kafka/message/monster/kafka.go`, add `CommandTypeSelfDestruct = "SELF_DESTRUCT"` to the command-type const block, add `DeathType byte \`json:"deathType"\`` as the last field of both `StatusEventKilledBody` and `StatusEventDestroyedBody`, and:

```go
// SelfDestructCommandBody asks atlas-monsters to detonate a self-destructing
// monster. The channel supplies only the reporting character; atlas-monsters
// owns the animation (from WZ) and every predicate, because the trigger
// originates in a client-controlled packet (task-253 design D7/D8).
type SelfDestructCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}
```

- [x] **Step 4: Add the producer and processor method**

In `monster/producer.go`, after `KillCommandProvider`:

```go
// SelfDestructCommandProvider asks atlas-monsters to detonate a monster.
// Keyed on the monster id like every other monster command, so a contact
// report lands on the same partition as the damage that preceded it.
func SelfDestructCommandProvider(f field.Model, monsterId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &monster2.Command[monster2.SelfDestructCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      monster2.CommandTypeSelfDestruct,
		Body: monster2.SelfDestructCommandBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

In `monster/processor.go`, add `SelfDestruct(f field.Model, monsterId uint32, characterId uint32) error` to the `Processor` interface after `Kill`, and:

```go
func (p *ProcessorImpl) SelfDestruct(f field.Model, monsterId uint32, characterId uint32) error {
	p.l.Debugf("Requesting self-destruct of monster [%d] reported by character [%d].", monsterId, characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(SelfDestructCommandProvider(f, monsterId, characterId))
}
```

In `monster/mock/processor.go`, add the field
`SelfDestructFunc func(f field.Model, monsterId uint32, characterId uint32) error`
and the method returning `nil` when the func is nil, matching `Kill`'s shape.

- [x] **Step 5: Pass the death type through the consumer**

In `kafka/consumer/monster/consumer.go`, add above `destroyForSession`:

```go
// destroyTypeFor maps a KILLED/DESTROYED event's deathType byte onto the wire
// dead-type. 0 means "the producer did not set it" — an old atlas-monsters
// mid-rolling-deploy — and renders as fade-out, byte-identical to the
// pre-task-253 hardcode (task-253 design D9).
func destroyTypeFor(deathType byte) monsterpkt.DestroyType {
	if deathType == 0 {
		return monsterpkt.DestroyTypeFadeOut
	}
	return monsterpkt.DestroyType(deathType)
}
```

Change `killForSession` and `destroyForSession` to take a
`dt monsterpkt.DestroyType` parameter alongside `uniqueId` and pass it to
`monsterpkt.NewMonsterDestroy(uniqueId, dt)`. Update both call sites to pass
`destroyTypeFor(e.Body.DeathType)`.

- [x] **Step 6: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...`
Expected: PASS

- [x] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go services/atlas-channel/atlas.com/channel/monster/ services/atlas-channel/atlas.com/channel/kafka/consumer/monster/
git commit -m "feat(atlas-channel): emit SELF_DESTRUCT and render the event's death type"
```

---

### Task 12: `atlas-channel` — the `MONSTER_BOMB` handler

**Goal:** `MONSTER_BOMB` stops being decode-and-log and detonates a live, in-field mob for a living character (design D8; PRD FR-5.1, FR-5.2, FR-5.3, FR-5.5).

**Interfaces**

- Consumes: `monster.Processor.SelfDestruct` (Task 11), `monster.GetLiveMirror()` (existing), `serverbound.MonsterBomb.MobId()` (existing).
- Produces: package-level seams `monsterBombCharacterFunc` and `monsterBombSelfDestructFunc` for tests.

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/monster_bomb.go` — the handler
- `services/atlas-channel/atlas.com/channel/socket/handler/monster_bomb_test.go` — new file
- `services/atlas-channel/atlas.com/channel/monster/live_mirror.go` — read-only; `Lookup`/`Put`/`LiveEntry`
- `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go:59` — read-only; the dead-character idiom
- `libs/atlas-packet/monster/serverbound/monster_bomb.go` — read-only; codec unchanged

Module root: `services/atlas-channel/atlas.com/channel`.

Patterns to copy: `socket/handler/cash_shop_check_name_change.go:26` (package-level `var …Func = func(...)` seam), `socket/handler/character_chat_whisper_test.go:62-85` (building a session with `session.NewSession` + `session.NewProcessor(...).SetCharacterId/SetField`), `socket/handler/monster_catch_item_use_test.go:14-30` (encoding a serverbound packet into a `request.Reader`).

- [x] **Step 1: Write the failing tests**

New file `monster_bomb_test.go`, package `handler`. Each test swaps both package seams and restores them with `defer`, records whether `monsterBombSelfDestructFunc` was called, and drives the handler with a real encoded packet:

```go
ctx := tenant.WithContext(context.Background(), ten)
req := request.Request([]byte{0x59, 0x1B, 0x00, 0x00}) // mobId 7001
reader := request.NewRequestReader(&req, 0)
MonsterBombHandleFunc(l, ctx, nil)(s, &reader, nil)
```

`serverbound.MonsterBomb` has no exported constructor and its `mobId` field is
unexported, so build the payload bytes directly: the codec is a single
`Encode4` (`libs/atlas-packet/monster/serverbound/monster_bomb.go:47`), so
mob id `7001` is the four little-endian bytes `0x59 0x1B 0x00 0x00`.

`TestMonsterBombDetonates` — session character `4242` in field `(0, 1, 100000000)`,
`monsterBombCharacterFunc` returns a character with `Hp() == 500`, mirror seeded
via `monster.GetLiveMirror().Put(ten, 7001, monster.LiveEntry{Field: sameField, MonsterId: 8500003})`.
Expect the seam called exactly once with `f == sameField`, `monsterId == 7001`,
`characterId == 4242`.

`TestMonsterBombRejects` — table-driven; in every case the seam must **not** be called:

| case | setup | expected log substring |
|---|---|---|
| character lookup fails | `monsterBombCharacterFunc` returns an error | `unable to resolve` |
| dead character | character `Hp() == 0` | `while dead` |
| mirror miss | mirror never seeded for `7001` | `not in the live mirror` |
| mob in another field | mirror entry's `Field` is `(0, 1, 200000000)` | `not in the reporter's field` |

Assert on a `logrus` logger writing into a `bytes.Buffer` at `DebugLevel`, the
same way `character_chat_whisper_test.go` captures logs.

`TestMonsterBombDuplicateReportIsHarmless` — call the handler twice with the
same valid setup. The seam is called twice and neither call errors; idempotence
is enforced by atlas-monsters (`Registry.SelfDestruct`), not by the channel, so
the handler must not try to dedupe.

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run MonsterBomb -v`
Expected: FAIL — `undefined: monsterBombSelfDestructFunc`.

- [x] **Step 3: Implement the handler**

Replace `socket/handler/monster_bomb.go` in full:

```go
package handler

import (
	"atlas-channel/character"
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-packet/monster/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var monsterBombCharacterFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (character.Model, error) {
	cp := character.NewProcessor(l, ctx)
	return cp.GetById()(characterId)
}

var monsterBombSelfDestructFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32, characterId uint32) error {
	return monster.NewProcessor(l, ctx).SelfDestruct(f, monsterId, characterId)
}

// MonsterBombHandleFunc handles CMob::TryFirstSelfDestruction: the controller
// reports that a self-destructing mob's first-attack body rect intersected the
// local user. The decoded id is the mob OBJECT id (v95 @0x640ee0 encodes
// GetMobID(this)), the same identifier MonsterDestroy carries as uniqueId —
// never a template id (task-253 design §2.3).
//
// The channel keeps only the two guards it can answer from state it already
// holds: the reporter must be alive, and the target must be in the reporter's
// field. Whether the target actually carries a selfDestruction block is
// atlas-monsters' call — it is the authority on monster lifecycle, already
// holds the data behind a cache, and must re-check anyway (design D8).
//
// There is no failure packet and no enableActions: TryFirstSelfDestruction is
// fire-and-forget with no client-side response state, so a rejection is a log
// line and nothing else. Duplicate reports from several clients in the field
// are expected and harmless — Registry.SelfDestruct makes the detonation
// exactly-once server-side.
func MonsterBombHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := serverbound.MonsterBomb{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		c, err := monsterBombCharacterFunc(l, ctx, s.CharacterId())
		if err != nil {
			l.WithError(err).Debugf("MONSTER_BOMB: unable to resolve character [%d]; dropping report for monster [%d].", s.CharacterId(), p.MobId())
			return
		}
		if c.Hp() == 0 {
			l.Debugf("MONSTER_BOMB: character [%d] reported monster [%d] while dead; dropping.", s.CharacterId(), p.MobId())
			return
		}

		entry, ok := monster.GetLiveMirror().Lookup(tenant.MustFromContext(ctx), p.MobId())
		if !ok {
			l.Debugf("MONSTER_BOMB: monster [%d] is not in the live mirror; dropping report from character [%d].", p.MobId(), s.CharacterId())
			return
		}
		if entry.Field.Id() != s.Field().Id() {
			l.Debugf("MONSTER_BOMB: monster [%d] is not in the reporter's field; dropping report from character [%d].", p.MobId(), s.CharacterId())
			return
		}

		if err := monsterBombSelfDestructFunc(l, ctx, s.Field(), p.MobId(), s.CharacterId()); err != nil {
			l.WithError(err).Errorf("MONSTER_BOMB: unable to request self-destruct of monster [%d].", p.MobId())
		}
	}
}
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/ ./monster/`
Expected: PASS

- [x] **Step 5: Confirm the deferred-behavior marker is gone**

Run: `grep -rn "behavior: deferred" services/atlas-channel/atlas.com/channel/socket/handler/monster_bomb.go`
Expected: no output (PRD acceptance: "the handler no longer contains a `// behavior: deferred` comment").

- [x] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/monster_bomb.go services/atlas-channel/atlas.com/channel/socket/handler/monster_bomb_test.go
git commit -m "feat(atlas-channel): act on MONSTER_BOMB contact reports"
```

---

### Task 13: `atlas-monster-death` — pin the `ActorId = 0` behaviour

**Goal:** Prove, rather than assume, that a detonation with no killer produces unowned drops without error, excludes quest-specific drops, and distributes no EXP (design §5, §8.2; PRD FR-6.4).

No production change is expected. If any assertion below cannot be made to pass without changing production code, that is a real defect — fix it and say so in the commit message rather than weakening the test.

### Files

- `services/atlas-monster-death/atlas.com/monster/monster/actor_zero_test.go` — new file
- `services/atlas-monster-death/atlas.com/monster/monster/processor.go:41,104,126,207` — read-only; `CreateDrops`, `filterByQuestState`, `DistributeExperience`, `calculateExperienceStandardDeviationThreshold`
- `services/atlas-monster-death/atlas.com/monster/quest/provider_drain_test.go` — read-only; the httptest + `t.Setenv` pattern
- `services/atlas-monster-death/atlas.com/monster/monster/drop/builder.go` — read-only; `drop.NewBuilder()`

Module root: `services/atlas-monster-death/atlas.com/monster`.

Service URL env keys (from each package's `getBaseRequest`): quests `QUESTS_SERVICE_URL`, parties `PARTIES_SERVICE_URL`, rates `RATES_SERVICE_URL`, drop tables `DROPS_INFORMATION_SERVICE_URL`, monster info `DATA_SERVICE_URL`, maps `MAPS_SERVICE_URL`, characters `CHARACTERS_SERVICE_URL`. Every one takes a trailing `/`.

Patterns to copy: `quest/provider_drain_test.go:48-62` (`httptest.NewServer` + `t.Setenv("<KEY>_SERVICE_URL", srv.URL+"/")` + `tenant.Create` + `tenant.WithContext`), `monster/characterization_test.go` (pure-function assertions).

- [x] **Step 1: Write the tests**

New file `actor_zero_test.go`, package `monster`.

`TestFilterByQuestStateExcludesQuestDropsForNoKiller` — table-driven over
`(&ProcessorImpl{l: l, ctx: ctx}).filterByQuestState(0, drops)` where `drops`
holds one drop with `QuestId() == 0` (item `1000`) and one with
`QuestId() == 4000` (item `2000`):

| case | quest server behaviour | want item ids |
|---|---|---|
| quest lookup errors (500) | responds `500` | `[1000]` |
| character has no started quests | responds an empty JSON:API page | `[1000]` |
| no quest drops at all | only the `QuestId() == 0` drop is passed in; server must never be hit | `[1000]` |

For the empty-page case, serve
`{"data":[],"meta":{"total":0,"page":{"number":1,"size":250,"last":1}}}`
with `Content-Type: application/vnd.api+json`.

`TestCreateDropsWithNoKillerSpawnsUnownedDrop` — the drop spawn is a Kafka
command, not a REST call (`monster/drop/processor.go:87-92` emits
`spawnDropCommandProvider` to `drop.EnvCommandTopic`), so capture it.

Add a package `TestMain` to this new file (no other test file in
`services/atlas-monster-death/atlas.com/monster/monster` declares one):

```go
var emitted *producertest.Capture

func TestMain(m *testing.M) {
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
}
```

Then stand up three stub servers and set:
`DROPS_INFORMATION_SERVICE_URL` — one non-quest drop, `chance` `999999`,
`itemId` `1000`, `minimumQuantity` `1`, `maximumQuantity` `1`, `questId` `0`;
`RATES_SERVICE_URL` — `404`, so `rates.GetForCharacter` falls back to
`Default()` (`rates/provider.go:27-30`);
`PARTIES_SERVICE_URL` — `404`, so `party.GetByMemberId(0)` errors and
`ownerPartyId` stays `0` (`monster/processor.go:61-64`).
No quest server is needed: the only drop has `questId == 0`, so
`filterByQuestState` short-circuits before any quest lookup.

Call `emitted.Reset()`, then
`NewProcessor(l, ctx).CreateDrops(f, 500, 8500003, 10, 20, 0)`.

Assert: the returned error is `nil`; `emitted.Messages(drop.EnvCommandTopic)`
holds exactly one message; and unmarshalling that message's `Value` into

```go
var out struct {
	Body struct {
		OwnerId      uint32 `json:"ownerId"`
		OwnerPartyId uint32 `json:"ownerPartyId"`
		ItemId       uint32 `json:"itemId"`
	} `json:"body"`
}
```

yields `OwnerId == 0`, `OwnerPartyId == 0`, `ItemId == 1000` — the drop spawns
unowned (`spawnCommandBody` is unexported in package `drop`, hence the
anonymous struct).

`TestCalculateExperienceStandardDeviationThresholdEmptyIsNaN` — pin the known
adjacent defect (design §8.2):
`math.IsNaN(calculateExperienceStandardDeviationThreshold([]float64{}, 0))`
must be `true`. Comment the assertion as: harmless today because the only
caller then iterates an empty distribution map, and harmless on the
`ActorId = 0` path for the same reason; pinned, not changed.

`TestDistributeExperienceWithEmptyEntriesIsNoOp` — set `DATA_SERVICE_URL` to a
stub serving a monster with `hp` `4000` and `experience` `150`, and
`MAPS_SERVICE_URL` to a stub serving an empty character list. Point
`CHARACTERS_SERVICE_URL` at a handler that fails the test if it is ever
called. `DistributeExperience(f, 8500003, nil)` must return `nil` and the
character server must record zero requests.

- [x] **Step 2: Run the tests**

Run: `cd services/atlas-monster-death/atlas.com/monster && go test ./monster/ -run 'NoKiller|EmptyEntries|EmptyIsNaN|FilterByQuestState' -v`
Expected: PASS. If any fails, the behaviour FR-6.4 assumes is not actually
present — fix the production code in this task rather than adjusting the
assertion.

- [x] **Step 3: Run the module suite**

Run: `cd services/atlas-monster-death/atlas.com/monster && go build ./... && go test ./...`
Expected: PASS

- [x] **Step 4: Commit**

```bash
git add services/atlas-monster-death/atlas.com/monster/monster/actor_zero_test.go
git commit -m "test(atlas-monster-death): pin ActorId=0 drop and experience behaviour"
```

---

## Final gates (controller, not an implementer)

- [x] **Flagless verification**

Run: `tools/verify.sh`
Expected: exit 0. `--quick`/`--no-docker` do not count — they skip the bake and `-race`.

- [x] **Packet audit checks**

```bash
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit gate-lint --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit dispatcher-lint
```

Expected: all exit 0.

- [x] **Code review**

Dispatch `backend-guidelines-reviewer` (atlas-monsters, atlas-channel, atlas-data, atlas-monster-death) and `plan-adherence-reviewer` before opening the PR. Neither is optional.

- [x] **Live observation**

On a running channel, damage a Boomer (`5100002`) from full HP past `1800` and
confirm it plays its death animation and drops. Then attempt the Papulatus bomb
(`8500003`) case from PRD §10. If the bombs do not spawn (design §11, PRD OQ7),
report that explicitly — do not record the acceptance item as passed and do not
silently skip it.
