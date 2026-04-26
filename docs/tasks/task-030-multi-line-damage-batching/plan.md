# Multi-line damage batching — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Batch all damage lines of a single attack on a single monster into one Kafka `DAMAGE` command, so atlas-monsters applies them atomically and always emits a `damaged` event before any `killed` event — fixing the HP bar visually skipping the killing line of a multi-line attack (e.g., L7).

**Architecture:** Hard-cut Kafka schema change on `COMMAND_TOPIC_MONSTER` `DAMAGE` body: rename `Damage uint32` → `Damages []uint32`. atlas-channel collapses its per-line producer loop into one Kafka message per `DamageInfo`. atlas-monsters consumes the slice, applies lines in order in a Go loop (registry script unchanged), early-breaks on kill, and emits exactly one `damaged` event per attack plus a `killed` event when applicable.

**Tech Stack:** Go 1.25, Kafka (segmentio/kafka-go), redis (go-redis), miniredis for tests. Two service modules: `atlas-channel` and `atlas-monsters`.

---

## Pre-flight

- [ ] **Step 1: Confirm branch**

Run: `git rev-parse --abbrev-ref HEAD`
Expected: `feature/task-030-multi-line-damage-batching`

If not on that branch, switch: `git checkout feature/task-030-multi-line-damage-batching`

- [ ] **Step 2: Confirm clean working tree on the design commit**

Run: `git status --short`
Expected: empty (or only untracked unrelated files).

Run: `git log --oneline -1`
Expected: a `docs(task-030):` design commit.

- [ ] **Step 3: Read the design and context**

Read `docs/tasks/task-030-multi-line-damage-batching/design.md` and `docs/tasks/task-030-multi-line-damage-batching/context.md`. Do not skip — they spell out the exact behaviour and the conventions to follow.

---

## Task 1 — atlas-monsters: write failing test for new damage command body schema

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka_test.go`

This test pins the JSON shape of `damageCommandBody`. It will fail to compile (the `Damages` field doesn't exist yet) and that's the whole point.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka_test.go`:

```go
package monster

import (
	"encoding/json"
	"testing"
)

func TestDamageCommandBody_DecodeNewShape(t *testing.T) {
	raw := []byte(`{"characterId":42,"damages":[100,200,300],"attackType":1}`)
	var body damageCommandBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if body.CharacterId != 42 {
		t.Fatalf("CharacterId = %d, want 42", body.CharacterId)
	}
	if len(body.Damages) != 3 || body.Damages[0] != 100 || body.Damages[1] != 200 || body.Damages[2] != 300 {
		t.Fatalf("Damages = %v, want [100 200 300]", body.Damages)
	}
	if body.AttackType != 1 {
		t.Fatalf("AttackType = %d, want 1", body.AttackType)
	}
}

func TestDamageCommandBody_MissingDamagesIsNil(t *testing.T) {
	raw := []byte(`{"characterId":42,"attackType":1}`)
	var body damageCommandBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if body.Damages != nil {
		t.Fatalf("Damages = %v, want nil for missing field", body.Damages)
	}
}

func TestDamageCommandBody_OldDamageFieldIgnored(t *testing.T) {
	// In-flight messages from the old shape have only "damage" (singular).
	// The new consumer must decode them with Damages == nil so the handler
	// no-ops them. Asserts the schema rename was a hard cut, not a coexist.
	raw := []byte(`{"characterId":42,"damage":500,"attackType":1}`)
	var body damageCommandBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if body.Damages != nil {
		t.Fatalf("Damages = %v, want nil when only legacy 'damage' field present", body.Damages)
	}
}
```

- [ ] **Step 2: Run test to verify it fails to compile**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./kafka/consumer/monster/...`
Expected: build failure with messages like `body.Damages undefined (type damageCommandBody has no field or method Damages)`.

---

## Task 2 — atlas-monsters: rename schema field

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`

- [ ] **Step 1: Rename the field**

In `kafka.go`, replace:

```go
type damageCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	Damage      uint32 `json:"damage"`
	AttackType  byte   `json:"attackType"`
}
```

with:

```go
type damageCommandBody struct {
	CharacterId uint32   `json:"characterId"`
	Damages     []uint32 `json:"damages"`
	AttackType  byte     `json:"attackType"`
}
```

- [ ] **Step 2: Run schema tests — they should pass; consumer.go should still fail to compile**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./kafka/consumer/monster/...`
Expected: compile failure in `consumer.go:72` referencing `c.Body.Damage` (the singular field that no longer exists). The schema tests themselves pass once the build clears.

That's the next task.

---

## Task 3 — atlas-monsters: update consumer handler

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go`

- [ ] **Step 1: Rewrite handleDamageCommand**

Replace the existing function (around line 66):

```go
func handleDamageCommand(l logrus.FieldLogger, ctx context.Context, c command[damageCommandBody]) {
	if c.Type != CommandTypeDamage {
		return
	}

	p := monster.NewProcessor(l, ctx)
	p.Damage(c.MonsterId, c.Body.CharacterId, c.Body.Damage, c.Body.AttackType)
}
```

with:

```go
func handleDamageCommand(l logrus.FieldLogger, ctx context.Context, c command[damageCommandBody]) {
	if c.Type != CommandTypeDamage {
		return
	}
	if len(c.Body.Damages) == 0 {
		l.Debugf("DAMAGE command for monster [%d] has no damage lines; ignoring.", c.MonsterId)
		return
	}

	p := monster.NewProcessor(l, ctx)
	p.Damage(c.MonsterId, c.Body.CharacterId, c.Body.Damages, c.Body.AttackType)
}
```

(Note: `p.Damage` now takes a slice. The processor signature change happens in Task 4 — until then this file will still compile-fail against the old signature. That's expected; we'll fix both before running the build.)

- [ ] **Step 2: Do not run build yet**

The build will fail because `monster.NewProcessor(...).Damage(...)` is still typed as `(uint32, ...)`. Move directly to Task 4.

---

## Task 4 — atlas-monsters: rewrite processor Damage to take a slice

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go`

- [ ] **Step 1: Update the Processor interface**

In `processor.go` (around line 40), change:

```go
Damage(id uint32, characterId uint32, damage uint32, attackType byte)
```

to:

```go
Damage(id uint32, characterId uint32, damages []uint32, attackType byte)
```

- [ ] **Step 2: Rewrite ProcessorImpl.Damage**

Replace the existing `Damage` method (lines 229-309) with:

```go
// Damage applies a sequence of damage lines from a single attack to a monster.
// Lines are applied in order; if any line kills the monster, later lines are
// dropped (overkill discarded). Always emits a `damaged` event reflecting the
// final state, plus a `killed` event when the attack lands a kill, so the
// channel writes the final HP-bar packet before the death animation.
func (p *ProcessorImpl) Damage(id uint32, characterId uint32, damages []uint32, attackType byte) {
	if len(damages) == 0 {
		return
	}

	m, err := GetMonsterRegistry().GetMonster(p.t, id)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get monster [%d].", id)
		return
	}
	if !m.Alive() {
		p.l.Debugf("Character [%d] trying to apply damage to an already dead monster [%d].", characterId, id)
		return
	}

	// Reflect runs once per attack, not once per line.
	p.checkReflect(m, characterId, attackType)

	// Fetch monster info for boss flag and revives
	var isBoss bool
	var revives []uint32
	if ma, infoErr := information.GetById(p.l)(p.ctx)(m.MonsterId()); infoErr == nil {
		isBoss = ma.Boss()
		revives = ma.Revives()
	}

	var last DamageSummary
	killed := false
	for _, d := range damages {
		s, err := GetMonsterRegistry().ApplyDamage(p.t, characterId, d, m.UniqueId())
		if err != nil {
			p.l.WithError(err).Errorf("Error applying damage to monster %d from character %d.", m.UniqueId(), characterId)
			return
		}
		last = s
		if s.Killed {
			killed = true
			break // discard overkill
		}
	}

	// Always emit damaged so the channel writes the final HP-bar packet,
	// even when the attack lands a kill.
	if err := producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(damagedStatusEventProvider(last.Monster, last.CharacterId, last.CharacterId, isBoss, DamageSourceCharacterAttack, last.Monster.DamageSummary())); err != nil {
		p.l.WithError(err).Errorf("Monster [%d] damaged, but unable to display that for the characters in the field.", last.Monster.UniqueId())
	}

	if killed {
		// Clear cooldowns and drop timer on death
		GetCooldownRegistry().ClearCooldowns(p.ctx, p.t, id)
		GetDropTimerRegistry().Unregister(p.ctx, p.t, id)

		// Emit cancellation events for any active status effects before death
		for _, se := range last.Monster.StatusEffects() {
			_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(statusEffectCancelledEventProvider(last.Monster, se))
		}

		if err := producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(killedStatusEventProvider(last.Monster, last.CharacterId, isBoss, last.Monster.DamageSummary())); err != nil {
			p.l.WithError(err).Errorf("Monster [%d] killed, but unable to display that for the characters in the field.", last.Monster.UniqueId())
		}
		if _, err := GetMonsterRegistry().RemoveMonster(p.ctx, p.t, last.Monster.UniqueId()); err != nil {
			p.l.WithError(err).Errorf("Monster [%d] killed, but not removed from registry.", last.Monster.UniqueId())
		}

		// Boss revive: spawn next phase monsters
		if len(revives) > 0 {
			p.spawnRevives(last.Monster, revives)
		}
		return
	}

	// Damage-leader re-control runs once after the full attack.
	if characterId != last.Monster.ControlCharacterId() {
		if last.Monster.DamageLeader() == characterId {
			p.l.Debugf("Character [%d] has become damage leader. They should now control the monster.", characterId)
			m2, err := p.GetById(last.Monster.UniqueId())
			if err != nil {
				return
			}
			if err := p.StopControl(m2); err != nil {
				p.l.WithError(err).Errorf("Unable to stop [%d] from controlling monster [%d].", last.Monster.ControlCharacterId(), last.Monster.UniqueId())
			}
			if _, err := p.StartControl(m2.UniqueId(), characterId); err != nil {
				p.l.WithError(err).Errorf("Unable to start [%d] controlling monster [%d].", characterId, m2.UniqueId())
			}
		}
	}
}
```

Important details preserved from the old code:
- Existing imports stay; no new imports.
- `s.CharacterId` was used for the event ids; we use `last.CharacterId` here (same value, just stored on the latest summary).
- Status-effect cancellation order before `killed` is preserved.
- Drop-timer/cooldown cleanup order is preserved.
- Boss revive logic is preserved.

- [ ] **Step 3: Build atlas-monsters**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./...`
Expected: build succeeds.

- [ ] **Step 4: Run all atlas-monsters tests**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./...`
Expected: all tests pass (including the new schema tests from Task 1 and the existing registry tests, which still cover cumulative damage application via `ApplyDamage`).

If `monster/registry_test.go` tests fail, do NOT modify them — investigate. The processor change must not have altered registry behaviour.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go \
        services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka_test.go \
        services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go \
        services/atlas-monsters/atlas.com/monsters/monster/processor.go
git commit -m "$(cat <<'EOF'
fix(atlas-monsters): batch multi-line damage in single Damage call

DAMAGE command body now carries Damages []uint32 instead of a
single Damage. Processor applies lines in order in a Go loop
(registry script unchanged), breaks on kill, and always emits a
damaged event reflecting the final state — followed by a killed
event when the attack lands a kill. This guarantees the channel
writes the final MonsterHealth packet before the death animation.

Reflect, drop-timer cleanup, status-effect cancellation, boss
revive, and damage-leader re-control all run once per attack
rather than once per line.

Refs task-030.
EOF
)"
```

---

## Task 5 — atlas-channel: write failing schema test for command body

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka_test.go`:

```go
package monster

import (
	"encoding/json"
	"testing"
)

func TestDamageCommandBody_EncodeNewShape(t *testing.T) {
	body := DamageCommandBody{
		CharacterId: 42,
		Damages:     []uint32{100, 200, 300},
		AttackType:  1,
	}
	out, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	want := `{"characterId":42,"damages":[100,200,300],"attackType":1}`
	if string(out) != want {
		t.Fatalf("got %s, want %s", out, want)
	}
}

func TestDamageCommandBody_DecodeRoundTrip(t *testing.T) {
	in := DamageCommandBody{
		CharacterId: 7,
		Damages:     []uint32{1, 2},
		AttackType:  0,
	}
	raw, _ := json.Marshal(in)
	var got DamageCommandBody
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.CharacterId != 7 || len(got.Damages) != 2 || got.Damages[0] != 1 || got.Damages[1] != 2 || got.AttackType != 0 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails to compile**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/message/monster/...`
Expected: build failure with `unknown field Damages in struct literal of type DamageCommandBody`.

---

## Task 6 — atlas-channel: rename schema field

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`

- [ ] **Step 1: Rename the field**

In `kafka.go`, replace:

```go
type DamageCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	Damage      uint32 `json:"damage"`
	AttackType  byte   `json:"attackType"`
}
```

with:

```go
type DamageCommandBody struct {
	CharacterId uint32   `json:"characterId"`
	Damages     []uint32 `json:"damages"`
	AttackType  byte     `json:"attackType"`
}
```

- [ ] **Step 2: Verify schema tests pass; producer/processor still break**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/message/monster/...`
Expected: schema tests pass.

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`
Expected: build failure in `monster/producer.go` (uses `Damage:` field that no longer exists).

That's the next task.

---

## Task 7 — atlas-channel: update DamageCommandProvider

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/monster/producer.go`
- Create: `services/atlas-channel/atlas.com/channel/monster/producer_test.go`

- [ ] **Step 1: Write failing producer test first**

Create `services/atlas-channel/atlas.com/channel/monster/producer_test.go`:

```go
package monster

import (
	"encoding/json"
	"testing"

	monster2 "atlas-channel/kafka/message/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

func TestDamageCommandProvider_EncodesDamagesSlice(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	provider := DamageCommandProvider(f, 12345, 67, []uint32{40, 80, 120}, 1)

	msgs, err := provider()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}

	var cmd monster2.Command[monster2.DamageCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal command: %v", err)
	}
	if cmd.Type != monster2.CommandTypeDamage {
		t.Fatalf("Type = %s, want %s", cmd.Type, monster2.CommandTypeDamage)
	}
	if cmd.MonsterId != 12345 {
		t.Fatalf("MonsterId = %d, want 12345", cmd.MonsterId)
	}
	if cmd.Body.CharacterId != 67 {
		t.Fatalf("Body.CharacterId = %d, want 67", cmd.Body.CharacterId)
	}
	if len(cmd.Body.Damages) != 3 || cmd.Body.Damages[0] != 40 || cmd.Body.Damages[1] != 80 || cmd.Body.Damages[2] != 120 {
		t.Fatalf("Body.Damages = %v, want [40 80 120]", cmd.Body.Damages)
	}
	if cmd.Body.AttackType != 1 {
		t.Fatalf("Body.AttackType = %d, want 1", cmd.Body.AttackType)
	}
}
```

- [ ] **Step 2: Run test — expect compile failure**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./monster/...`
Expected: build failure — the test calls `DamageCommandProvider` with `[]uint32` but the function still takes `uint32`.

- [ ] **Step 3: Update DamageCommandProvider**

In `services/atlas-channel/atlas.com/channel/monster/producer.go` (around line 84), replace:

```go
func DamageCommandProvider(f field.Model, monsterId uint32, characterId uint32, damage uint32, attackType byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &monster2.Command[monster2.DamageCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      monster2.CommandTypeDamage,
		Body: monster2.DamageCommandBody{
			CharacterId: characterId,
			Damage:      damage,
			AttackType:  attackType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

with:

```go
func DamageCommandProvider(f field.Model, monsterId uint32, characterId uint32, damages []uint32, attackType byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &monster2.Command[monster2.DamageCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      monster2.CommandTypeDamage,
		Body: monster2.DamageCommandBody{
			CharacterId: characterId,
			Damages:     damages,
			AttackType:  attackType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 4: Verify producer test passes; processor still breaks**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./monster/...`
Expected: producer test passes.

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`
Expected: build failure in `monster/processor.go` (`Damage` method still has old signature) and `socket/handler/character_attack_common.go`.

---

## Task 8 — atlas-channel: update Processor.Damage signature

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/monster/processor.go`

- [ ] **Step 1: Change the Damage method**

In `processor.go` (around line 43), replace:

```go
func (p *Processor) Damage(f field.Model, monsterId uint32, characterId uint32, damage uint32, attackType byte) error {
	p.l.Debugf("Applying damage to monster [%d]. Character [%d]. Damage [%d].", monsterId, characterId, damage)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(DamageCommandProvider(f, monsterId, characterId, damage, attackType))
}
```

with:

```go
func (p *Processor) Damage(f field.Model, monsterId uint32, characterId uint32, damages []uint32, attackType byte) error {
	p.l.Debugf("Applying damage to monster [%d]. Character [%d]. Lines [%d].", monsterId, characterId, len(damages))
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(DamageCommandProvider(f, monsterId, characterId, damages, attackType))
}
```

- [ ] **Step 2: Do not run build yet**

`character_attack_common.go` still calls the old signature. Move directly to Task 9.

---

## Task 9 — atlas-channel: collapse the per-line loop in the attack handler

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`

- [ ] **Step 1: Replace the inner loop**

In `character_attack_common.go` (lines 67-73), replace:

```go
mp := monster.NewProcessor(l, ctx)
for _, di := range ai.DamageInfo() {
	for _, d := range di.Damages() {
		err := mp.Damage(s.Field(), di.MonsterId(), s.CharacterId(), d, byte(ai.AttackType()))
		if err != nil {
			l.WithError(err).Errorf("Unable to apply damage [%d] to monster [%d] from character [%d].", d, di.MonsterId(), s.CharacterId())
		}
	}

	// Apply monster status effects from skill (e.g., freeze, poison, stun)
	if len(se.MonsterStatus()) > 0 {
```

with:

```go
mp := monster.NewProcessor(l, ctx)
for _, di := range ai.DamageInfo() {
	if damages := di.Damages(); len(damages) > 0 {
		if err := mp.Damage(s.Field(), di.MonsterId(), s.CharacterId(), damages, byte(ai.AttackType())); err != nil {
			l.WithError(err).Errorf("Unable to apply damage to monster [%d] from character [%d].", di.MonsterId(), s.CharacterId())
		}
	}

	// Apply monster status effects from skill (e.g., freeze, poison, stun)
	if len(se.MonsterStatus()) > 0 {
```

(Only the inner loop changes. The outer `for _, di := range ai.DamageInfo()` stays. The status-effect block after it is unchanged.)

- [ ] **Step 2: Build atlas-channel**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`
Expected: build succeeds.

- [ ] **Step 3: Run all atlas-channel tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./...`
Expected: all tests pass (including the new schema and producer tests).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go \
        services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka_test.go \
        services/atlas-channel/atlas.com/channel/monster/producer.go \
        services/atlas-channel/atlas.com/channel/monster/producer_test.go \
        services/atlas-channel/atlas.com/channel/monster/processor.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
git commit -m "$(cat <<'EOF'
feat(atlas-channel): batch attack damage lines into single DAMAGE command

DamageCommandBody.Damage uint32 → Damages []uint32. The attack
handler collapses the per-line loop into one Damage call per
DamageInfo (one per targeted monster). Empty damage slices are
skipped — no Kafka message is emitted.

Pairs with the matching atlas-monsters change so multi-line
attacks (e.g., L7) drain the HP bar through every line, including
the killing one.

Refs task-030.
EOF
)"
```

---

## Task 10 — Final verification across both services

- [ ] **Step 1: atlas-monsters full build + test**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...`
Expected: green.

- [ ] **Step 2: atlas-channel full build + test**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...`
Expected: green.

- [ ] **Step 3: Confirm no other call sites of the old Damage signatures remain**

Run from repo root:
```
grep -rn "DamageCommandBody{" services/atlas-channel services/atlas-monsters
grep -rn "\.Damage(.*uint32.*byte)" services/atlas-channel services/atlas-monsters
```

Expected: every match either constructs the body with `Damages:` (slice) or calls a `.Damage(...)` method with a `[]uint32` argument. Any reference to a singular `Damage:` field or `damage uint32` parameter on these specific functions is a bug — fix it.

- [ ] **Step 4: Confirm git log shows the two implementation commits on top of the design commit**

Run: `git log --oneline -3`
Expected: 3 commits — design, atlas-monsters fix, atlas-channel feat.

---

## Task 11 — Manual smoke test (required before merge)

This is not a `git commit`-style step; it produces a paragraph for the PR description.

- [ ] **Step 1: Bring up the local stack**

Start atlas-channel, atlas-monsters, kafka, and redis (project-standard `docker compose` or equivalent). If you do not have a working local stack, skip to Step 4.

- [ ] **Step 2: Reproduce the original bug pre-revert (optional)**

To convince yourself the fix works, you can `git stash` the implementation commits, run the stack, observe the HP bar skipping the killing line, then restore the commits. Skip if you trust the design.

- [ ] **Step 3: Verify the fix**

In-game:
- Pick a low-HP monster you can two-shot with L7. Cast L7. The HP bar must drain twice (once per damage line) before death.
- Pick a monster you can one-shot with L7. The HP bar must drain to 0% before the death animation (a behaviour change vs. today, intentional per the design).

- [ ] **Step 4: Record findings for PR**

Either:
- "Smoke tested locally: 2-line L7 drains HP bar twice before death, 1-line L7 drains HP bar to 0% before death animation."
- Or: "Did not run local stack. Recommend reviewer or QA repro before merge."

Be honest — do not claim the smoke test passed if you didn't run it.

---

## Task 12 — Open the pull request

- [ ] **Step 1: Push the branch**

Run: `git push -u origin feature/task-030-multi-line-damage-batching`

- [ ] **Step 2: Open the PR**

Run:
```bash
gh pr create --title "fix: batch multi-line attack damage so HP bar drains through every line" --body "$(cat <<'EOF'
## Summary
- Renames the `COMMAND_TOPIC_MONSTER` `DAMAGE` body field from `damage uint32` to `damages []uint32` (hard cut). atlas-channel now emits one Kafka command per `DamageInfo` instead of one per damage line.
- atlas-monsters applies the lines in order in a Go loop, breaks on kill, and always emits a `damaged` event followed by a `killed` event when applicable — guaranteeing the channel writes the final HP-bar packet before the death animation.
- Fixes the user-visible bug where multi-line player attacks (e.g., L7) showed only N-1 of N HP-bar drains before death.

See `docs/tasks/task-030-multi-line-damage-batching/design.md` for the design and rationale.

## Test plan
- [ ] `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...` green
- [ ] `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...` green
- [ ] Manual: 2-line L7 drains HP bar twice before death (or note skipped)
- [ ] Manual: 1-line killing blow drains HP bar to 0% before death animation (or note skipped)
EOF
)"
```

- [ ] **Step 3: Confirm the PR opens cleanly**

Expected: PR URL printed; CI begins running.

---

## Self-review (already done by author)

Spec coverage: every section of `design.md` maps to a task.
- Schema rename → Tasks 1–2 (monsters), 5–6 (channel)
- Producer change → Task 7
- Channel attack handler loop change → Task 9
- Consumer handler empty-slice guard → Task 3
- Processor rewrite (loop, always-damaged, kill ordering, reflect once, leader once) → Task 4
- Edge cases (empty, dead-on-arrival, kill on middle line, one-line) → covered by code in Task 4 + Task 3 empty-slice guard; not separately unit-tested due to lack of Kafka harness, called out in `context.md`
- Channel handler unchanged → no task needed; verified by Task 10 Step 2 (existing tests pass) and Step 3 (grep)
- Rollout (hard cut) → Task 1 schema test pins old-shape decoding to nil

Placeholder scan: none.

Type consistency:
- Both services use `Damages []uint32` with JSON tag `"damages"`.
- atlas-monsters `Processor.Damage` and `ProcessorImpl.Damage` agree on `(id uint32, characterId uint32, damages []uint32, attackType byte)`.
- atlas-channel `Processor.Damage` and `DamageCommandProvider` agree on `(... characterId uint32, damages []uint32, attackType byte)`.
- atlas-channel handler call: `mp.Damage(s.Field(), di.MonsterId(), s.CharacterId(), damages, byte(ai.AttackType()))` — argument order matches.
