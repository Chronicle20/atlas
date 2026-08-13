# Energy Charge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Pirate melee energy bar work end to end — it fills 102 per attacked monster, charges at 10000, grants weapon attack and unlocks Energy Blast for the skill's duration, then resets.

**Architecture:** One buff row in atlas-buffs carrying a single `ENERGY_CHARGE` stat, driven through the existing `UPDATE_STAT_VALUE`/`INCREMENT`+`Cap` command (widened with a `CreateIfMissing` upsert so the attack path needs no REST read) and the existing `EVENT_TOPIC_CHARACTER_BUFF_STATUS` fan-out. atlas-channel reacts to that event stream: it mirrors the bar value pod-locally for the cast gate, announces the skill-use effect, and promotes 10000 → the 15000 charged sentinel with a timed `APPLY`. atlas-effective-stats turns the charged sentinel into a `weapon_attack` bonus resolved from the skill effect's `pad`. libs/atlas-packet stops writing the bar as zeros.

**Tech Stack:** Go 1.x multi-module workspace (`go.work`), Kafka (segmentio), Redis-backed registries (miniredis in tests), JSON:API REST via api2go, `testify/assert` in atlas-buffs / plain `testing` elsewhere.

## Global Constraints

- **No `go.mod` changes.** Every module touched already depends on everything it needs, so `docker buildx bake` is NOT required (CLAUDE.md Build & Verification item 4). If a task turns out to need a new dependency, STOP and re-plan.
- **Durations are MILLISECONDS everywhere.** `effect.Model.Duration()` already returns ms; `ApplyCommandBody.Duration` is ms. Never write `* 1000` / `time.Second` scaling into a buff command body — `tools/buff-duration-guard.sh` fails CI on it.
- **Never compare raw job/skill wire ids.** Resolve through `constants.For(t.Region(), t.MajorVersion(), t.MinorVersion()).Skill` — `.Resolve(wireId) (Identity, bool)` and `.Wire(identity) (Id, bool)` — then compare Identities with `skill.IsIdentity`. `tools/skill-job-id-guard.sh` enforces this.
- **Bar constants:** gain `102` per attacked monster, accumulation cap `10000`, charged sentinel `15000`. 15000 is a sentinel, never "150% full".
- **Energy bookkeeping must never fail an attack.** Every error on the attack path is logged at Error and swallowed; no branch returns an error to `processAttack`.
- **Zero blocking REST on the attack path.** The accumulation emit is fire-and-forget Kafka; the cast gate is a pod-local mirror read. One REST call is permitted only on a *rejected* cast.
- **Test setup uses the project Builder/constructor pattern.** No `*_testhelpers.go` files.
- **Never write absolute home paths** (`/home/<user>/…`) into committed files; use repo-relative paths.
- **Every commit message** ends with the two trailer lines this repo uses (`Co-Authored-By:` / `Claude-Session:`), matching recent history.

## File Structure

| Module | File | Responsibility |
|---|---|---|
| libs/atlas-packet | `model/character_temporary_stat.go` | `getBaseTemporaryStats`: emit ENERGY_CHARGE's `nOption`/`rOption` instead of zeros |
| libs/atlas-packet | `model/character_temporary_stat_test.go` | byte fixtures pinning the populated ENERGY_CHARGE base block (v83 + v61) |
| atlas-buffs | `kafka/message/character/kafka.go` | `UpdateStatValueCommandBody` gains `CreateIfMissing` + `Level` |
| atlas-buffs | `character/registry.go` | `StatValueUpdate` value struct; `UpdateStatValue` upsert branch |
| atlas-buffs | `character/processor.go` | emit `APPLIED` on create, `STAT_UPDATED` on change |
| atlas-buffs | `kafka/consumer/character/consumer.go` | pass ChannelId + the two new body fields through |
| atlas-channel | `kafka/message/buff/kafka.go` | mirror the two new command-body fields |
| atlas-channel | `character/buff/producer.go` / `processor.go` | `StatValueUpdate` mirror struct; provider carries the new fields |
| atlas-channel | `character/buff/energy.go` (new) | `EnergyMirror` — pod-local bar value per tenant/character |
| atlas-channel | `socket/handler/character_attack_energy_charge.go` (new) | line resolution, gain amount, qualify predicate, blast predicate, deps, `energyChargeTryUpdate`, `energyBlastPermitted`, re-announce |
| atlas-channel | `socket/handler/character_attack_common.go` | blast gate + accumulation call site inside `processAttack` |
| atlas-channel | `kafka/consumer/buff/consumer.go` | mirror maintenance, skill-use effect announce, 10000 → 15000 promotion |
| atlas-effective-stats | `external/buffs/rest.go` | `Level byte` field |
| atlas-effective-stats | `character/initializer.go` | `energyChargeBonus` helper + wiring in `fetchBuffBonuses` |

---

### Task 1: atlas-packet — ENERGY_CHARGE base block carries the bar value

The `ENERGY_CHARGE` two-state base block is currently written with `nOption`/`rOption` at zero, so a `GIVE_BUFF` carrying `ENERGY_CHARGE = 4998` puts `0` on the wire. The v83 client reads that first int32 as the bar reading (design.md §1.1: `sub_7F9BAD` computes `this[364] / this[365] * flt_B38988`). Block shape and size are unchanged — only the two leading int32s stop being zero. `DASH_SPEED` / `DASH_JUMP` / `UNDEAD` are deliberately left alone: no evidence was gathered for what their clients read, so the change is keyed on the stat name, not on the shared `default:` arm.

**Files:**
- Modify: `libs/atlas-packet/model/character_temporary_stat.go` (`getBaseTemporaryStats`, around `:1362-1410`)
- Test: `libs/atlas-packet/model/character_temporary_stat_test.go` (append)

**Interfaces:**
- Consumes: nothing from other tasks.
- Produces: no Go API change. `CharacterTemporaryStat.Encode` now emits `nOption = stat value`, `rOption = stat sourceId` for `character.TemporaryStatTypeEnergyCharge`.

- [x] **Step 1: Write the failing tests**

Append to `libs/atlas-packet/model/character_temporary_stat_test.go`. Fixture values: bar `4998` = `0x1386` → LE `86 13 00 00`; skill `5110001` = `0x004DF8F1` → LE `F1 F8 4D 00`.

```go
// TestCTSEnergyChargePre95PopulatedBlock pins the ENERGY_CHARGE base block's
// two leading int32s. The client reads nOption as the energy-bar reading:
// GMS v83 IDB sub_7F9BAD computes fill = this[364] / this[365] * bar width,
// where this[364] is the first field of the received two-state entry
// (design.md §1.1). A zeroed nOption renders an empty bar no matter what the
// buff actually holds.
func TestCTSEnergyChargePre95PopulatedBlock(t *testing.T) {
	pre95 := []struct {
		name   string
		region string
		major  uint16
	}{
		{"GMS v72", "GMS", 72},
		{"GMS v79", "GMS", 79},
		{"GMS v83", "GMS", 83},
		{"GMS v84", "GMS", 84},
		{"GMS v87", "GMS", 87},
		{"GMS v92", "GMS", 92},
		{"GMS v95", "GMS", 95},
		{"JMS v185", "JMS", 185},
	}
	for _, v := range pre95 {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			tn, _ := tenant.Create([16]byte{}, v.region, v.major, 1)
			input := NewCharacterTemporaryStat()
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypeEnergyCharge), 5110001, 4998, 1, time.Time{})

			got := input.Encode(nil, ctx)(nil)

			// 16 mask + 2 leading defense bytes + one 15-byte dynamic base block.
			if len(got) != 16+2+15 {
				t.Fatalf("energy charge packet length: got %d want %d", len(got), 16+2+15)
			}
			// nOption=4998 then rOption=5110001 as consecutive LE int32s.
			head := []byte{0x86, 0x13, 0x00, 0x00, 0xF1, 0xF8, 0x4D, 0x00}
			if !bytes.Contains(got, head) {
				t.Fatalf("populated ENERGY_CHARGE head (nOption=4998,rOption=5110001) missing; got % x", got)
			}
		})
	}
}

// TestCTSEnergyChargeV61PopulatedBlock covers GMS v61's narrower base block
// (14 bytes: the third field is a bare Decode4, not the bool-prefixed
// 5-byte time pair). Only the block width differs; the two leading int32s
// are in the same place.
func TestCTSEnergyChargeV61PopulatedBlock(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 61, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeEnergyCharge), 5110001, 4998, 1, time.Time{})

	got := input.Encode(nil, ctx)(nil)

	if len(got) != 16+2+14 {
		t.Fatalf("v61 energy charge packet length: got %d want %d", len(got), 16+2+14)
	}
	head := []byte{0x86, 0x13, 0x00, 0x00, 0xF1, 0xF8, 0x4D, 0x00}
	if !bytes.Contains(got, head) {
		t.Fatalf("v61 populated ENERGY_CHARGE head missing; got % x", got)
	}
}

// TestCTSEnergyChargeRoundTrip guards encode/decode symmetry: the decoder is
// shape-only, so a populated block must still be consumed byte-for-byte.
func TestCTSEnergyChargeRoundTrip(t *testing.T) {
	for _, v := range []struct {
		region string
		major  uint16
	}{{"GMS", 61}, {"GMS", 83}, {"GMS", 95}, {"JMS", 185}} {
		ctx := pt.CreateContext(v.region, v.major, 1)
		tn, _ := tenant.Create([16]byte{}, v.region, v.major, 1)
		input := NewCharacterTemporaryStat()
		input.AddStat(nil)(tn)(string(character.TemporaryStatTypeEnergyCharge), 5110001, 4998, 1, time.Time{})
		output := NewCharacterTemporaryStat()
		pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	}
}

// TestCTSDashSpeedStaysZeroed is the negative half of the ENERGY_CHARGE fix:
// the other twoStateDynamic members (DASH_SPEED, DASH_JUMP, UNDEAD) keep the
// zeroed block, because no evidence was gathered for what their clients read
// and this task must not make an unverified wire change to a verified cell
// (design.md §1.1).
func TestCTSDashSpeedStaysZeroed(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeDashSpeed), 5110001, 4998, 1, time.Time{})

	got := input.Encode(nil, ctx)(nil)

	block := got[18:]
	for i := 0; i < 8; i++ {
		if block[i] != 0x00 {
			t.Fatalf("DASH_SPEED base block must stay zeroed; got % x", block)
		}
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./model/ -run 'TestCTSEnergyCharge|TestCTSDashSpeedStaysZeroed' -v`
Expected: the three `TestCTSEnergyCharge*` populated-block assertions FAIL with "populated ENERGY_CHARGE head … missing" (the round-trip and the DASH_SPEED test pass already — they are regression guards).

- [x] **Step 3: Implement**

In `libs/atlas-packet/model/character_temporary_stat.go`, replace the `default:` arm of the switch inside `getBaseTemporaryStats`:

```go
		default: // twoStateDynamic
			// ENERGY_CHARGE's nOption IS the client's energy-bar reading:
			// GMS v83 sub_7F9BAD computes the fill as this[364]/this[365],
			// where this[364] is the block's first int32 (task-216
			// design.md §1.1). rOption carries the source skill id, matching
			// every other populated two-state block.
			//
			// The group's other dynamic members (DASH_SPEED, DASH_JUMP,
			// UNDEAD) keep the zeroed block deliberately: no evidence was
			// gathered for what their clients read, and their matrix cells
			// are already verified against the zeros.
			if bs.name == character.TemporaryStatTypeEnergyCharge {
				list = append(list, NewCharacterTemporaryStatBaseWithOptions(true, s.Value(), s.SourceId(), narrow)) // 15 (14 on GMS v61)
				continue
			}
			list = append(list, NewCharacterTemporaryStatBase(true, narrow)) // dynamic, 15 (14 on GMS v61)
		}
```

Note `continue` targets the enclosing `for` loop over `twoStateBaseStats(t)` — that is intentional and correct.

- [x] **Step 4: Run the tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./model/ -run 'TestCTS' -v`
Expected: PASS, including every pre-existing `TestCTS*` fixture (they must not move — the block shape is unchanged).

- [x] **Step 5: Full module verification**

Run: `cd libs/atlas-packet && go test -race ./... && go vet ./...`
Expected: PASS / no output.

- [x] **Step 6: Commit**

```bash
git add libs/atlas-packet/model/character_temporary_stat.go libs/atlas-packet/model/character_temporary_stat_test.go
git commit -m "fix(packet): ENERGY_CHARGE base block carries the energy-bar value"
```

---

### Task 2: atlas-buffs — `StatValueUpdate` struct and the create-if-missing upsert

`Registry.UpdateStatValue` returns `(Model{}, false, nil)` when the buff is missing — by design, and Combo relies on it. Energy Charge needs the *first* hit to create the buff, but the channel cannot know whether it exists without a REST read on the attack path (forbidden by NFR-1). So `INCREMENT` gains an opt-in upsert: when the buff is missing and `CreateIfMissing` is set, atlas-buffs creates a `NoExpiry` buff holding one stat change at `min(Amount, Cap)`.

The semantics stay skill-agnostic — "accumulator upsert" is the same family as the `INCREMENT`+`Cap` the command already owns. Because the parameter list would otherwise reach nine positional arguments, the request collapses into a `StatValueUpdate` value struct.

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go` (`UpdateStatValueCommandBody`, `:94-100`)
- Modify: `services/atlas-buffs/atlas.com/buffs/character/registry.go` (`UpdateStatValue`, `:360-420`)
- Test: `services/atlas-buffs/atlas.com/buffs/character/registry_test.go` (existing `TestRegistry_UpdateStatValue_*` call sites at `:583-680` must be updated to the new signature; new tests appended)

**Interfaces:**
- Produces (used by Tasks 3 and 4):
  - `character.StatValueUpdate{SourceId int32; StatType string; Operation string; Amount int32; Cap int32; CreateIfMissing bool; Level byte}` in package `atlas-buffs/character`.
  - `(*Registry).UpdateStatValue(ctx context.Context, worldId world.Id, channelId channel.Id, characterId uint32, u StatValueUpdate) (buff.Model, bool /*changed*/, bool /*created*/, error)`
  - `character.UpdateStatValueCommandBody` gains `CreateIfMissing bool \`json:"createIfMissing"\`` and `Level byte \`json:"level"\``.

- [x] **Step 1: Write the failing tests**

First, mechanically update the four existing `TestRegistry_UpdateStatValue_*` tests and `TestRegistry_UpdateStatValue_NoOps` to the new signature. Example of the shape (apply the same transformation to each call site):

```go
	updated, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 1, Cap: 6})
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.False(t, created)
```

Then append the new coverage:

```go
// TestRegistry_UpdateStatValue_CreateIfMissingCreatesNoExpiryBuff is the
// Energy Charge accumulation entry point: the very first qualifying hit
// arrives with no buff at all, and the channel cannot check for one without
// a REST read on the attack path (task-216 design.md §4.2).
func TestRegistry_UpdateStatValue_CreateIfMissingCreatesNoExpiryBuff(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	updated, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{
			SourceId: 5110001, StatType: "ENERGY_CHARGE",
			Operation: character2.StatOperationIncrement,
			Amount:    102, Cap: 10000, CreateIfMissing: true, Level: 20,
		})

	assert.NoError(t, err)
	assert.True(t, changed)
	assert.True(t, created)
	assert.True(t, updated.NoExpiry())
	assert.Equal(t, byte(20), updated.Level())
	assert.Len(t, updated.Changes(), 1)
	assert.Equal(t, "ENERGY_CHARGE", updated.Changes()[0].Type())
	assert.Equal(t, int32(102), updated.Changes()[0].Amount())
}

// A create whose Amount already exceeds Cap stores the clamped value, so the
// created buff can never start out above the accumulation ceiling.
func TestRegistry_UpdateStatValue_CreateIfMissingClampsToCap(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	updated, _, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{
			SourceId: 5110001, StatType: "ENERGY_CHARGE",
			Operation: character2.StatOperationIncrement,
			Amount:    99999, Cap: 10000, CreateIfMissing: true, Level: 20,
		})

	assert.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, int32(10000), updated.Changes()[0].Amount())
}

// The second hit increments the buff the first hit created, and reports
// created=false so the processor emits STAT_UPDATED rather than APPLIED.
func TestRegistry_UpdateStatValue_CreateIfMissingThenIncrements(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	u := StatValueUpdate{
		SourceId: 5110001, StatType: "ENERGY_CHARGE",
		Operation: character2.StatOperationIncrement,
		Amount:    102, Cap: 10000, CreateIfMissing: true, Level: 20,
	}
	_, _, _, _ = GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000, u)
	updated, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000, u)

	assert.NoError(t, err)
	assert.True(t, changed)
	assert.False(t, created)
	assert.Equal(t, int32(204), updated.Changes()[0].Amount())
}

// CreateIfMissing is opt-in: without it a missing buff stays the no-op Combo
// depends on. This is the Combo regression guard.
func TestRegistry_UpdateStatValue_MissingBuffWithoutCreateIsNoOp(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	_, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 1, Cap: 6})

	assert.NoError(t, err)
	assert.False(t, changed)
	assert.False(t, created)
}

// CreateIfMissing only makes sense for INCREMENT; SET must not conjure a buff.
func TestRegistry_UpdateStatValue_CreateIfMissingIgnoredForSet(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	_, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 5110001, StatType: "ENERGY_CHARGE", Operation: character2.StatOperationSet, Amount: 15000, CreateIfMissing: true, Level: 20})

	assert.NoError(t, err)
	assert.False(t, changed)
	assert.False(t, created)
}

// At the 15000 charged sentinel every further gain is a no-op, because the
// value is already at/above the 10000 accumulation cap. FR-2.5 is structural,
// not a guard.
func TestRegistry_UpdateStatValue_ChargedSentinelBlocksGain(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), 1000, 5110001, 20, 31000,
		[]stat.Model{stat.NewStat("ENERGY_CHARGE", 15000)}, false, false)
	assert.NoError(t, err)

	_, changed, created, err := GetRegistry().UpdateStatValue(ctx, world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 5110001, StatType: "ENERGY_CHARGE", Operation: character2.StatOperationIncrement, Amount: 102, Cap: 10000, CreateIfMissing: true, Level: 20})

	assert.NoError(t, err)
	assert.False(t, changed)
	assert.False(t, created)
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -run TestRegistry_UpdateStatValue`
Expected: FAIL to compile — "undefined: StatValueUpdate" and "too many arguments".

- [x] **Step 3: Implement**

In `services/atlas-buffs/atlas.com/buffs/character/registry.go`, add the struct above `UpdateStatValue` and rewrite the method:

```go
// StatValueUpdate is one UPDATE_STAT_VALUE request. Collected into a struct
// rather than passed positionally because the accumulator-upsert fields
// (task-216) push the argument list past readability.
type StatValueUpdate struct {
	SourceId  int32
	StatType  string
	Operation string
	Amount    int32
	Cap       int32
	// CreateIfMissing turns INCREMENT into an upsert: when the character has
	// no buff for SourceId, one is created with NoExpiry carrying a single
	// StatType change of min(Amount, Cap). Opt-in, so every pre-existing
	// caller (Combo orbs, Enrage consume) keeps the missing-buff no-op.
	// Ignored for SET — SET replaces a value, it does not accumulate one.
	CreateIfMissing bool
	// Level is the source skill level stamped on a buff created by
	// CreateIfMissing. Ignored otherwise.
	Level byte
}

// UpdateStatValue changes the amount of one stat on the character's active
// buff for u.SourceId. INCREMENT adds u.Amount clamped to u.Cap (no-op when
// already at/above cap); SET replaces the amount outright. With
// u.CreateIfMissing an INCREMENT against a missing buff creates a NoExpiry
// buff instead of no-opping.
//
// Returns (buff, changed, created, error). changed is true whenever a
// mutation was stored; created is true only for the CreateIfMissing path, so
// the processor can emit APPLIED rather than STAT_UPDATED. Returns
// (Model{}, false, false, nil) when the buff is missing/expired without
// CreateIfMissing, lacks the stat, the operation is unknown, or the value
// would not change. Only whole-source (non-accumulate) buffs are addressed
// via srcKey. Same get-modify-put shape as Cancel, serialized per character
// by the command topic's characterId partition key.
func (r *Registry) UpdateStatValue(ctx context.Context, worldId world.Id, channelId channel.Id, characterId uint32, u StatValueUpdate) (buff.Model, bool, bool, error) {
	t := tenant.MustFromContext(ctx)

	canCreate := u.CreateIfMissing && u.Operation == character2.StatOperationIncrement && u.Amount > 0

	m, err := r.characters.Get(ctx, t, characterId)
	if errors.Is(err, atlas.ErrNotFound) {
		if !canCreate {
			return buff.Model{}, false, false, nil
		}
		m = Model{
			worldId:     worldId,
			channelId:   channelId,
			characterId: characterId,
			buffs:       make(map[string]buff.Model),
		}
	} else if err != nil {
		return buff.Model{}, false, false, err
	}

	b, ok := m.buffs[srcKey(u.SourceId)]
	if !ok || b.Expired() {
		if !canCreate {
			return buff.Model{}, false, false, nil
		}
		initial := u.Amount
		if u.Cap > 0 && initial > u.Cap {
			initial = u.Cap
		}
		created, cerr := buff.NewNoExpiryBuff(u.SourceId, u.Level, []stat.Model{stat.NewStat(u.StatType, initial)})
		if cerr != nil {
			return buff.Model{}, false, false, cerr
		}
		m.buffs[srcKey(u.SourceId)] = created
		if perr := r.characters.Put(ctx, t, characterId, m); perr != nil {
			return buff.Model{}, false, false, perr
		}
		return created, true, true, nil
	}

	var current int32
	found := false
	for _, c := range b.Changes() {
		if c.Type() == u.StatType {
			current = c.Amount()
			found = true
			break
		}
	}
	if !found {
		return buff.Model{}, false, false, nil
	}

	var next int32
	switch u.Operation {
	case character2.StatOperationIncrement:
		if u.Amount <= 0 || current >= u.Cap {
			return buff.Model{}, false, false, nil
		}
		next = current + u.Amount
		if next > u.Cap {
			next = u.Cap
		}
	case character2.StatOperationSet:
		if u.Amount < 1 {
			return buff.Model{}, false, false, nil
		}
		next = u.Amount
	default:
		return buff.Model{}, false, false, nil
	}
	if next == current {
		return buff.Model{}, false, false, nil
	}

	updated, ok := b.WithStatAmount(u.StatType, next)
	if !ok {
		return buff.Model{}, false, false, nil
	}
	m.buffs[srcKey(u.SourceId)] = updated
	if err := r.characters.Put(ctx, t, characterId, m); err != nil {
		return buff.Model{}, false, false, err
	}
	return updated, true, false, nil
}
```

Ensure `channel` and `stat` are imported in `registry.go` (both already are — `channel` via `Apply`, `stat` via the `Apply` signature).

Also widen the command body in `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`:

```go
// UpdateStatValueCommandBody changes the amount of one stat on a character's
// existing buff (identified by SourceId). The body is stat-generic; task-142
// uses it for COMBO orb bookkeeping. Cap applies to INCREMENT only.
type UpdateStatValueCommandBody struct {
	SourceId  int32  `json:"sourceId"`
	StatType  string `json:"statType"`
	Operation string `json:"operation"`
	Amount    int32  `json:"amount"`
	Cap       int32  `json:"cap"`
	// CreateIfMissing turns INCREMENT into an accumulator upsert: with no
	// buff for SourceId, one is created with NoExpiry carrying a single
	// StatType change of min(Amount, Cap), and APPLIED (not STAT_UPDATED) is
	// emitted. Opt-in — omitted/false leaves every existing producer's
	// behaviour byte-identical. (task-216 design.md §4.2)
	CreateIfMissing bool `json:"createIfMissing,omitempty"`
	// Level is the source skill level stamped on a buff created by
	// CreateIfMissing. Ignored otherwise.
	Level byte `json:"level,omitempty"`
}
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-buffs/atlas.com/buffs && go build ./... && go test ./character/ -run TestRegistry_UpdateStatValue -v`
Expected: PASS for every subtest. (The processor still calls the old signature, so `go build ./...` will fail at `processor.go` — that is Task 3. If you want a green build at this checkpoint, do Steps 3 of Task 2 and Task 3 back-to-back before committing.)

- [x] **Step 5: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/character/registry.go \
        services/atlas-buffs/atlas.com/buffs/character/registry_test.go \
        services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go
git commit -m "feat(buffs): create-if-missing upsert for UPDATE_STAT_VALUE increments"
```

---

### Task 3: atlas-buffs — emit APPLIED on create, STAT_UPDATED on change

The processor is the only place that knows which status event to publish. A created buff must announce `APPLIED` (it is a new buff, with its own createdAt/expiresAt and the `noExpiry` flag the channel needs); a mutated buff keeps announcing `STAT_UPDATED`.

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/character/processor.go` (`Processor` interface `:25`, `UpdateStatValue` `:168-189`)
- Modify: `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go` (`handleUpdateStatValue` `:103-110`)
- Test: `services/atlas-buffs/atlas.com/buffs/character/processor_test.go` (existing `TestProcessor_UpdateStatValue_*` at `:255-289` updated; new tests appended)

**Interfaces:**
- Consumes: `character.StatValueUpdate` and the 4-value `Registry.UpdateStatValue` return from Task 2.
- Produces: `Processor.UpdateStatValue(worldId world.Id, channelId channel.Id, characterId uint32, u StatValueUpdate) error`.

- [x] **Step 1: Write the failing tests**

Update the three existing tests to the new signature, e.g.:

```go
	_ = processor.UpdateStatValue(world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 2, Cap: 6})
```

Then append:

```go
// A CreateIfMissing increment against a character with no buffs at all
// stores a NoExpiry buff — the first Energy Charge hit of the cycle.
func TestProcessor_UpdateStatValue_CreateIfMissingStoresBuff(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	err := processor.UpdateStatValue(world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{
			SourceId: 5110001, StatType: "ENERGY_CHARGE",
			Operation: character2.StatOperationIncrement,
			Amount:    102, Cap: 10000, CreateIfMissing: true, Level: 20,
		})
	assert.NoError(t, err)

	m, err := GetRegistry().Get(ctx, 1000)
	assert.NoError(t, err)
	b := m.Buffs()[srcKey(5110001)]
	assert.True(t, b.NoExpiry())
	assert.Equal(t, int32(102), b.Changes()[0].Amount())
}

// Without CreateIfMissing a missing buff stays a logged no-op (Combo).
func TestProcessor_UpdateStatValue_MissingBuffWithoutCreateStoresNothing(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	err := processor.UpdateStatValue(world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 1, Cap: 6})
	assert.NoError(t, err)

	_, err = GetRegistry().Get(ctx, 1000)
	assert.ErrorIs(t, err, ErrNotFound)
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -run TestProcessor_UpdateStatValue`
Expected: FAIL to compile — the processor still takes the old positional arguments.

- [x] **Step 3: Implement**

In `services/atlas-buffs/atlas.com/buffs/character/processor.go`, change the interface line and the method:

```go
	UpdateStatValue(worldId world.Id, channelId channel.Id, characterId uint32, u StatValueUpdate) error
```

```go
// UpdateStatValue applies a stat-value mutation to an existing buff — or, with
// u.CreateIfMissing, creates the buff — and emits the matching status event: a
// created buff announces APPLIED (it is a new buff, carrying its own
// createdAt/expiresAt and noExpiry flag), a mutated one announces STAT_UPDATED
// with the buff's ORIGINAL timestamps (so the channel re-broadcasts the
// remaining duration and never extends the buff). Missing/expired buff without
// CreateIfMissing, and at-cap increments, stay Debug no-ops — the buff can
// lapse between the channel's attack and this command.
func (p *ProcessorImpl) UpdateStatValue(worldId world.Id, channelId channel.Id, characterId uint32, u StatValueUpdate) error {
	if u.Operation != character2.StatOperationIncrement && u.Operation != character2.StatOperationSet {
		p.l.Warnf("Unknown stat value operation [%s] for character [%d] buff [%d]; ignoring.", u.Operation, characterId, u.SourceId)
		return nil
	}
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		updated, changed, created, err := GetRegistry().UpdateStatValue(p.ctx, worldId, channelId, characterId, u)
		if err != nil {
			return err
		}
		if !changed {
			p.l.Debugf("No stat value change for character [%d] buff [%d] stat [%s].", characterId, u.SourceId, u.StatType)
			return nil
		}
		if created {
			return buf.Put(character2.EnvEventStatusTopic, appliedStatusEventProvider(worldId, characterId, characterId, updated.SourceId(), updated.Level(), updated.Duration(), updated.Changes(), updated.CreatedAt(), updated.ExpiresAt(), updated.NoExpiry()))
		}
		return buf.Put(character2.EnvEventStatusTopic, statUpdatedStatusEventProvider(worldId, characterId, updated.SourceId(), updated.Level(), updated.Duration(), updated.Changes(), updated.CreatedAt(), updated.ExpiresAt()))
	})
}
```

`fromId` is the character themself — the buff is self-inflicted, exactly as a self-cast is.

Then update the consumer in `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go`:

```go
func handleUpdateStatValue(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.UpdateStatValueCommandBody]) {
	if c.Type != character2.CommandTypeUpdateStatValue {
		return
	}

	u := character.StatValueUpdate{
		SourceId:        c.Body.SourceId,
		StatType:        c.Body.StatType,
		Operation:       c.Body.Operation,
		Amount:          c.Body.Amount,
		Cap:             c.Body.Cap,
		CreateIfMissing: c.Body.CreateIfMissing,
		Level:           c.Body.Level,
	}
	if err := character.NewProcessor(l, ctx).UpdateStatValue(c.WorldId, c.ChannelId, c.CharacterId, u); err != nil {
		l.WithError(err).Errorf("Unable to update stat value on buff [%d] for character [%d].", c.Body.SourceId, c.CharacterId)
	}
}
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-buffs/atlas.com/buffs && go build ./... && go test -race ./... && go vet ./...`
Expected: PASS everywhere.

- [x] **Step 5: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/character/processor.go \
        services/atlas-buffs/atlas.com/buffs/character/processor_test.go \
        services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go
git commit -m "feat(buffs): emit APPLIED for upsert-created stat-value buffs"
```

---

### Task 4: atlas-channel — mirror the widened UPDATE_STAT_VALUE contract

atlas-buffs owns the command body; the channel carries a mirror. Both fields must appear on the channel side or the new flag never reaches the wire. The channel's `buff.Processor.UpdateStatValue` collapses into the same struct shape so the two existing call sites (Combo, Enrage) stay readable.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go` (`UpdateStatValueCommandBody`, `:74-83`)
- Modify: `services/atlas-channel/atlas.com/channel/character/buff/producer.go` (`UpdateStatValueCommandProvider`, `:123-141`)
- Modify: `services/atlas-channel/atlas.com/channel/character/buff/processor.go` (interface `:25`, method `:93-96`)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo.go` (`comboOrbProductionDeps`, `:152`)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go` (Enrage consume, `:171`)
- Test: `services/atlas-channel/atlas.com/channel/character/buff/producer_test.go` (append)

**Interfaces:**
- Produces (used by Task 6):
  - `buff.StatValueUpdate{SourceId int32; StatType string; Operation string; Amount int32; Cap int32; CreateIfMissing bool; Level byte}` in package `atlas-channel/character/buff`.
  - `buff.Processor.UpdateStatValue(f field.Model, characterId uint32, u StatValueUpdate) error`
  - `buff.UpdateStatValueCommandProvider(f field.Model, characterId uint32, u StatValueUpdate) model.Provider[[]kafka.Message]`

- [x] **Step 1: Write the failing test**

Append to `services/atlas-channel/atlas.com/channel/character/buff/producer_test.go`:

```go
// TestUpdateStatValueCommandProviderCarriesUpsertFields pins the two fields
// task-216 added to the mirrored command body. The struct is duplicated across
// two Go modules, so a field that exists on one side and not the other fails no
// build — it decodes into a zero value at runtime, silently.
func TestUpdateStatValueCommandProviderCarriesUpsertFields(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()

	msgs, err := UpdateStatValueCommandProvider(f, 1000, StatValueUpdate{
		SourceId: 5110001, StatType: "ENERGY_CHARGE", Operation: buff2.StatOperationIncrement,
		Amount: 204, Cap: 10000, CreateIfMissing: true, Level: 20,
	})()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var cmd buff2.Command[buff2.UpdateStatValueCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != buff2.CommandTypeUpdateStatValue {
		t.Fatalf("type: got %q want %q", cmd.Type, buff2.CommandTypeUpdateStatValue)
	}
	if !cmd.Body.CreateIfMissing {
		t.Fatal("createIfMissing must survive the round trip")
	}
	if cmd.Body.Level != 20 {
		t.Fatalf("level: got %d want 20", cmd.Body.Level)
	}
	if cmd.Body.Amount != 204 || cmd.Body.Cap != 10000 {
		t.Fatalf("amount/cap: got %d/%d want 204/10000", cmd.Body.Amount, cmd.Body.Cap)
	}
}
```

Match the import aliases and the `field.NewBuilder(...)` invocation already used in `producer_test.go`; add `encoding/json` if absent.

- [x] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/buff/ -run TestUpdateStatValueCommandProviderCarriesUpsertFields`
Expected: FAIL to compile — "undefined: StatValueUpdate".

- [x] **Step 3: Implement**

`kafka/message/buff/kafka.go` — mirror the body exactly as atlas-buffs declares it:

```go
// UpdateStatValueCommandBody changes the amount of one stat on a character's
// existing buff (identified by SourceId). Owned by atlas-buffs; this is the
// channel-side mirror. Cap applies to INCREMENT only.
type UpdateStatValueCommandBody struct {
	SourceId  int32  `json:"sourceId"`
	StatType  string `json:"statType"`
	Operation string `json:"operation"`
	Amount    int32  `json:"amount"`
	Cap       int32  `json:"cap"`
	// CreateIfMissing turns INCREMENT into an accumulator upsert: with no buff
	// for SourceId, atlas-buffs creates one with NoExpiry carrying a single
	// StatType change of min(Amount, Cap) and emits APPLIED. Opt-in.
	// (task-216 design.md §4.2)
	CreateIfMissing bool `json:"createIfMissing,omitempty"`
	// Level is the source skill level stamped on a buff created by
	// CreateIfMissing. Ignored otherwise.
	Level byte `json:"level,omitempty"`
}
```

`character/buff/producer.go` — add the struct and rewrite the provider:

```go
// StatValueUpdate is one UPDATE_STAT_VALUE request. Mirrors
// atlas-buffs/character.StatValueUpdate; collected into a struct rather than
// passed positionally because the accumulator-upsert fields (task-216) push
// the argument list past readability.
type StatValueUpdate struct {
	SourceId        int32
	StatType        string
	Operation       string
	Amount          int32
	Cap             int32
	CreateIfMissing bool
	Level           byte
}

func UpdateStatValueCommandProvider(f field.Model, characterId uint32, u StatValueUpdate) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buff.Command[buff.UpdateStatValueCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        buff.CommandTypeUpdateStatValue,
		Body: buff.UpdateStatValueCommandBody{
			SourceId:        u.SourceId,
			StatType:        u.StatType,
			Operation:       u.Operation,
			Amount:          u.Amount,
			Cap:             u.Cap,
			CreateIfMissing: u.CreateIfMissing,
			Level:           u.Level,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

`character/buff/processor.go` — interface line and method:

```go
	UpdateStatValue(f field.Model, characterId uint32, u StatValueUpdate) error
```

```go
func (p *ProcessorImpl) UpdateStatValue(f field.Model, characterId uint32, u StatValueUpdate) error {
	p.l.Debugf("Character [%d] updating stat [%s] on buff [%d]: %s %d (cap %d, createIfMissing %t).", characterId, u.StatType, u.SourceId, u.Operation, u.Amount, u.Cap, u.CreateIfMissing)
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(UpdateStatValueCommandProvider(f, characterId, u))
}
```

`socket/handler/character_attack_combo.go:152` — keep `comboOrbDeps.emitUpdate`'s positional signature untouched (its tests depend on it); only the production wiring changes:

```go
		emitUpdate: func(sourceId int32, operation string, amount int32, capValue int32) error {
			return bp.UpdateStatValue(f, characterId, buff.StatValueUpdate{
				SourceId:  sourceId,
				StatType:  string(constants.TemporaryStatTypeCombo),
				Operation: operation,
				Amount:    amount,
				Cap:       capValue,
			})
		},
```

`socket/handler/character_skill_use.go:171` — the Enrage consume:

```go
			if cerr := buff.NewProcessor(l, ctx).UpdateStatValue(s.Field(), s.CharacterId(), buff.StatValueUpdate{
				SourceId:  enrageComboSource,
				StatType:  string(charconst.TemporaryStatTypeCombo),
				Operation: buff2.StatOperationSet,
				Amount:    1,
			}); cerr != nil {
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./character/buff/... ./socket/handler/... `
Expected: PASS, including every pre-existing Combo test (the deps signature did not move).

- [x] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go \
        services/atlas-channel/atlas.com/channel/character/buff/producer.go \
        services/atlas-channel/atlas.com/channel/character/buff/producer_test.go \
        services/atlas-channel/atlas.com/channel/character/buff/processor.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go
git commit -m "feat(channel): mirror create-if-missing fields on UPDATE_STAT_VALUE"
```

---

### Task 5: atlas-channel — the pod-local energy mirror

The Energy Blast cast gate must not do REST on the attack path. `BeaconMirror` (`character/buff/beacon.go`) is the established shape for "channel-local projection of a buff stat, fed by buff-status events"; `EnergyMirror` is the same thing holding an `int32` bar value.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/character/buff/energy.go`
- Test: `services/atlas-channel/atlas.com/channel/character/buff/energy_test.go`

**Interfaces:**
- Produces (used by Tasks 6, 8, 9):
  - `buff.GetEnergyMirror() *EnergyMirror`
  - `(*EnergyMirror).Set(t tenant.Model, characterId uint32, value int32)`
  - `(*EnergyMirror).Get(t tenant.Model, characterId uint32) (int32, bool)`
  - `(*EnergyMirror).Clear(t tenant.Model, characterId uint32)`
  - package-level `energyMirror` / `energyMirrorOnce` so tests can reset the singleton (same idiom as `beaconMirror` / `beaconMirrorOnce`).

- [x] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/character/buff/energy_test.go`:

```go
package buff

import (
	"sync"
	"testing"
)

func TestEnergyMirrorSetGetClear(t *testing.T) {
	energyMirrorOnce = sync.Once{}
	energyMirror = nil
	m := GetEnergyMirror()
	tn := newTestTenant(t)

	if _, ok := m.Get(tn, 100); ok {
		t.Fatal("empty mirror must miss")
	}

	m.Set(tn, 100, 4998)
	v, ok := m.Get(tn, 100)
	if !ok || v != 4998 {
		t.Fatalf("get after set: got (%d,%v) want (4998,true)", v, ok)
	}

	// Re-set replaces (each gain overwrites the bar reading).
	m.Set(tn, 100, 15000)
	v, _ = m.Get(tn, 100)
	if v != 15000 {
		t.Fatalf("re-set must replace: got %d want 15000", v)
	}

	m.Clear(tn, 100)
	if _, ok := m.Get(tn, 100); ok {
		t.Fatal("get after clear must miss")
	}
}

// A zero bar is a real reading, not an absence: Get must report ok=true so the
// cast gate rejects rather than failing open.
func TestEnergyMirrorZeroIsPresent(t *testing.T) {
	energyMirrorOnce = sync.Once{}
	energyMirror = nil
	m := GetEnergyMirror()
	tn := newTestTenant(t)

	m.Set(tn, 100, 0)
	v, ok := m.Get(tn, 100)
	if !ok || v != 0 {
		t.Fatalf("zero must be a present reading: got (%d,%v)", v, ok)
	}
}

func TestEnergyMirrorTenantIsolation(t *testing.T) {
	energyMirrorOnce = sync.Once{}
	energyMirror = nil
	m := GetEnergyMirror()
	t1 := newTestTenant(t)
	t2 := newTestTenant(t)

	m.Set(t1, 100, 4998)
	if _, ok := m.Get(t2, 100); ok {
		t.Fatal("tenant 2 must not see tenant 1's bar")
	}
}
```

`newTestTenant` already exists in `beacon_test.go` in the same package — reuse it, do not redeclare it.

- [x] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/buff/ -run TestEnergyMirror`
Expected: FAIL to compile — "undefined: GetEnergyMirror".

- [x] **Step 3: Implement**

Create `services/atlas-channel/atlas.com/channel/character/buff/energy.go`:

```go
package buff

import (
	"sync"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// EnergyMirror tracks each character's ENERGY_CHARGE bar reading from buff
// APPLIED / STAT_UPDATED / EXPIRED events, so the Energy Blast cast gate can
// read it with zero I/O on the attack hot path (task-216 design.md §4.4).
//
// Values are the raw stat amounts: 0..10000 while accumulating, and the 15000
// sentinel while charged. A missing entry means "unknown", NOT "empty bar" —
// callers must treat the two differently, because this mirror is
// process-local and repopulates only from subsequent events after a channel
// restart or a channel change.
type EnergyMirror struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]int32
}

var (
	energyMirror     *EnergyMirror
	energyMirrorOnce sync.Once
)

// GetEnergyMirror returns the process-wide singleton mirror, lazily
// initialising it on first call.
func GetEnergyMirror() *EnergyMirror {
	energyMirrorOnce.Do(func() {
		energyMirror = &EnergyMirror{perTenant: make(map[uuid.UUID]map[uint32]int32)}
	})
	return energyMirror
}

// Set records or replaces the tenant/character's bar reading.
func (m *EnergyMirror) Set(t tenant.Model, characterId uint32, value int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.perTenant[t.Id()]
	if !ok {
		c = make(map[uint32]int32)
		m.perTenant[t.Id()] = c
	}
	c[characterId] = value
}

// Clear removes the tenant/character's bar reading, returning the gate to its
// fail-open "unknown" state.
func (m *EnergyMirror) Clear(t tenant.Model, characterId uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.perTenant[t.Id()]; ok {
		delete(c, characterId)
	}
}

// Get returns the tenant/character's bar reading, if one is known.
func (m *EnergyMirror) Get(t tenant.Model, characterId uint32) (int32, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.perTenant[t.Id()]
	if !ok {
		return 0, false
	}
	v, ok := c[characterId]
	return v, ok
}
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./character/buff/ -run TestEnergyMirror -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/buff/energy.go \
        services/atlas-channel/atlas.com/channel/character/buff/energy_test.go
git commit -m "feat(channel): pod-local ENERGY_CHARGE bar mirror"
```

---

### Task 6: atlas-channel — Energy Charge line resolution, gain, and predicates

All the decision logic lives in pure functions plus one deps struct, mirroring `character_attack_combo.go`. Eligibility resolves from the *owned skill through the version-aware identity set*, not from a job check: owning `5110001` at level > 0 already implies the Marauder line, and `set.Wire` returning false on gms_v61 for the Cygnus identity is exactly the "no-op rather than a bogus id" AC-10 asks for.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_energy_charge.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_energy_charge_test.go`

**Interfaces:**
- Consumes: `buff.StatValueUpdate` + `buff.Processor.UpdateStatValue` (Task 4).
- Produces (used by Tasks 7 and 8):
  - `energyLine{skillId skill3.Id; level byte}`
  - `energyChargeLine(set skill3.Set, skills []skill.Model) (energyLine, bool)`
  - `energyChargeGainAmount(mobsHit int) int32`
  - `energyChargeQualifies(at packetmodel.AttackType, attackId skill3.Identity, attackIdOk bool) bool`
  - `isEnergyBlast(attackId skill3.Identity, attackIdOk bool) bool`
  - `energyChargeDeps{emitUpsert func(sourceId int32, level byte, amount int32, capValue int32) error}`
  - `energyChargeProductionDeps(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32) energyChargeDeps`
  - `energyChargeTryUpdate(l logrus.FieldLogger, set skill3.Set, c character.Model, ai packetmodel.AttackInfo, deps energyChargeDeps)`
  - constants `energyChargeGainPerMob = int32(102)`, `energyChargeCap = int32(10000)`, `energyChargedValue = int32(15000)`

- [x] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_energy_charge_test.go`:

```go
package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/skill"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// energyTestSet returns the version-aware constant set for one GMS major.
func energyTestSet(major uint16) skill3.Set {
	return constants.For("GMS", major, 1).Skill
}

// energyTestSkill / energyTestCharacter / energyTestAttack mirror the
// constructors character_attack_combo_test.go already uses (:113-134), so the
// two attack-helper test files stay consistent.
func energyTestSkill(t *testing.T, id skill3.Id, level byte) skill.Model {
	t.Helper()
	m, err := skill.Extract(skill.RestModel{Id: uint32(id), Level: level})
	if err != nil {
		t.Fatalf("skill.Extract: %v", err)
	}
	return m
}

func energyTestCharacter(t *testing.T, skills ...skill.Model) character.Model {
	t.Helper()
	return character.NewModelBuilder().SetId(1000).SetSkills(skills).MustBuild()
}

func energyTestAttack(at packetmodel.AttackType, skillId uint32, hits int) packetmodel.AttackInfo {
	ai := packetmodel.NewAttackInfo(at)
	ai.SetSkillId(skillId)
	for i := 0; i < hits; i++ {
		ai.AddDamageInfo(*packetmodel.NewDamageInfo(1))
	}
	return *ai
}

func TestEnergyChargeLine(t *testing.T) {
	v83 := energyTestSet(83)

	t.Run("adventurer line owned", func(t *testing.T) {
		line, ok := energyChargeLine(v83, []skill.Model{energyTestSkill(t, skill3.MarauderEnergyChargeId, 20)})
		if !ok || line.skillId != skill3.MarauderEnergyChargeId || line.level != 20 {
			t.Fatalf("got (%+v,%v)", line, ok)
		}
	})
	t.Run("cygnus line owned", func(t *testing.T) {
		line, ok := energyChargeLine(v83, []skill.Model{energyTestSkill(t, skill3.ThunderBreakerStage2EnergyChargeId, 10)})
		if !ok || line.skillId != skill3.ThunderBreakerStage2EnergyChargeId || line.level != 10 {
			t.Fatalf("got (%+v,%v)", line, ok)
		}
	})
	t.Run("both owned prefers adventurer", func(t *testing.T) {
		line, ok := energyChargeLine(v83, []skill.Model{
			energyTestSkill(t, skill3.ThunderBreakerStage2EnergyChargeId, 10),
			energyTestSkill(t, skill3.MarauderEnergyChargeId, 20),
		})
		if !ok || line.skillId != skill3.MarauderEnergyChargeId {
			t.Fatalf("got (%+v,%v)", line, ok)
		}
	})
	t.Run("level 0 is not owned", func(t *testing.T) {
		if _, ok := energyChargeLine(v83, []skill.Model{energyTestSkill(t, skill3.MarauderEnergyChargeId, 0)}); ok {
			t.Fatal("level 0 must not resolve a line")
		}
	})
	t.Run("neither owned", func(t *testing.T) {
		if _, ok := energyChargeLine(v83, []skill.Model{energyTestSkill(t, skill3.CrusaderComboAttackId, 30)}); ok {
			t.Fatal("a non-pirate must not resolve a line")
		}
	})
	// AC-10: gms_v61 has the adventurer line only. The Cygnus identity has no
	// wire binding there, so the branch must be a no-op, never a bogus id.
	t.Run("gms_v61 cygnus identity is unavailable", func(t *testing.T) {
		v61 := energyTestSet(61)
		if _, ok := v61.Wire(skill3.ThunderBreakerStage2EnergyCharge); ok {
			t.Fatal("precondition: gms_v61 must not bind the Cygnus Energy Charge identity")
		}
		if _, ok := energyChargeLine(v61, []skill.Model{energyTestSkill(t, skill3.ThunderBreakerStage2EnergyChargeId, 10)}); ok {
			t.Fatal("gms_v61 must not resolve a Cygnus energy line")
		}
		line, ok := energyChargeLine(v61, []skill.Model{energyTestSkill(t, skill3.MarauderEnergyChargeId, 20)})
		if !ok || line.skillId != skill3.MarauderEnergyChargeId {
			t.Fatalf("gms_v61 adventurer line: got (%+v,%v)", line, ok)
		}
	})
}

func TestEnergyChargeGainAmount(t *testing.T) {
	for _, tc := range []struct {
		mobs int
		want int32
	}{{0, 0}, {-1, 0}, {1, 102}, {6, 612}} {
		if got := energyChargeGainAmount(tc.mobs); got != tc.want {
			t.Fatalf("mobs=%d: got %d want %d", tc.mobs, got, tc.want)
		}
	}
}

func TestEnergyChargeQualifies(t *testing.T) {
	v83 := energyTestSet(83)
	sharkWave, _ := v83.Resolve(skill3.ThunderBreakerStage3SharkWaveId)
	spark, _ := v83.Resolve(skill3.ThunderBreakerStage3SparkId)

	for _, tc := range []struct {
		name string
		at   packetmodel.AttackType
		id   skill3.Identity
		ok   bool
		want bool
	}{
		{"melee always", packetmodel.AttackTypeMelee, 0, false, true},
		{"energy/touch always", packetmodel.AttackTypeEnergy, 0, false, true},
		{"ranged shark wave", packetmodel.AttackTypeRanged, sharkWave, true, true},
		{"ranged other skill", packetmodel.AttackTypeRanged, spark, true, false},
		{"ranged unresolved", packetmodel.AttackTypeRanged, 0, false, false},
		{"magic never", packetmodel.AttackTypeMagic, 0, false, false},
	} {
		if got := energyChargeQualifies(tc.at, tc.id, tc.ok); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestIsEnergyBlast(t *testing.T) {
	v83 := energyTestSet(83)
	marauder, _ := v83.Resolve(skill3.MarauderEnergyBlastId)
	tb, _ := v83.Resolve(skill3.ThunderBreakerStage2EnergyBlastId)
	shockwave, _ := v83.Resolve(skill3.MarauderShockwaveId)

	if !isEnergyBlast(marauder, true) {
		t.Fatal("marauder energy blast must be recognised")
	}
	if !isEnergyBlast(tb, true) {
		t.Fatal("thunder breaker energy blast must be recognised")
	}
	if isEnergyBlast(shockwave, true) {
		t.Fatal("shockwave is MP-costed and must not be gated")
	}
	if isEnergyBlast(marauder, false) {
		t.Fatal("an unresolved id must never be treated as energy blast")
	}
}

func TestEnergyChargeTryUpdate(t *testing.T) {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	v83 := energyTestSet(83)

	marauder := energyTestCharacter(t, energyTestSkill(t, skill3.MarauderEnergyChargeId, 20))

	t.Run("emits one upsert of 102 x mobs", func(t *testing.T) {
		var gotSource, gotAmount, gotCap int32
		var gotLevel byte
		calls := 0
		deps := energyChargeDeps{emitUpsert: func(sourceId int32, level byte, amount int32, capValue int32) error {
			calls++
			gotSource, gotLevel, gotAmount, gotCap = sourceId, level, amount, capValue
			return nil
		}}

		energyChargeTryUpdate(l, v83, marauder, energyTestAttack(packetmodel.AttackTypeMelee, 0, 3), deps)

		if calls != 1 {
			t.Fatalf("expected exactly 1 emit, got %d", calls)
		}
		if gotSource != int32(skill3.MarauderEnergyChargeId) || gotLevel != 20 || gotAmount != 306 || gotCap != 10000 {
			t.Fatalf("got source=%d level=%d amount=%d cap=%d", gotSource, gotLevel, gotAmount, gotCap)
		}
	})

	t.Run("no line means no emit", func(t *testing.T) {
		nonPirate := energyTestCharacter(t, energyTestSkill(t, skill3.CrusaderComboAttackId, 30))
		called := false
		deps := energyChargeDeps{emitUpsert: func(int32, byte, int32, int32) error { called = true; return nil }}

		energyChargeTryUpdate(l, v83, nonPirate, energyTestAttack(packetmodel.AttackTypeMelee, 0, 3), deps)

		if called {
			t.Fatal("a character without the skill must produce no emit")
		}
	})

	t.Run("zero monsters means no emit", func(t *testing.T) {
		called := false
		deps := energyChargeDeps{emitUpsert: func(int32, byte, int32, int32) error { called = true; return nil }}

		energyChargeTryUpdate(l, v83, marauder, energyTestAttack(packetmodel.AttackTypeMelee, 0, 0), deps)

		if called {
			t.Fatal("an attack that hit nothing must produce no emit")
		}
	})

	// FR-7.2: the Energy Charge aura's own touch damage still grants energy.
	t.Run("energy/touch attack by the charge skill itself still gains", func(t *testing.T) {
		var gotAmount int32
		deps := energyChargeDeps{emitUpsert: func(_ int32, _ byte, amount int32, _ int32) error {
			gotAmount = amount
			return nil
		}}

		energyChargeTryUpdate(l, v83, marauder,
			energyTestAttack(packetmodel.AttackTypeEnergy, uint32(skill3.MarauderEnergyChargeId), 1), deps)

		if gotAmount != 102 {
			t.Fatalf("touch damage must still grant energy: got %d want 102", gotAmount)
		}
	})

	// AC-13: a failing dep must never escape as an error.
	t.Run("emit failure is swallowed", func(t *testing.T) {
		deps := energyChargeDeps{emitUpsert: func(int32, byte, int32, int32) error { return errors.New("kafka down") }}
		energyChargeTryUpdate(l, v83, marauder, energyTestAttack(packetmodel.AttackTypeMelee, 0, 3), deps)
	})
}
```

The three helpers use the same constructors `character_attack_combo_test.go:113-134` already uses (`skill.Extract` / `character.NewModelBuilder()` / `packetmodel.NewAttackInfo` + `AddDamageInfo`), so no new test scaffolding is introduced. `constants.For(region, major, minor)` returns the version-aware constant set whose `.Skill` field is the `skill3.Set` — the same expression `character_attack_common.go:757` uses.

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'TestEnergyCharge|TestIsEnergyBlast'`
Expected: FAIL to compile — "undefined: energyChargeLine".

- [x] **Step 3: Implement**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_energy_charge.go`:

```go
package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	buff2 "atlas-channel/kafka/message/buff"
	"context"

	"github.com/sirupsen/logrus"

	constants "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

const (
	// energyChargeGainPerMob is the bar gain per attacked monster. Cosmic
	// calls handleEnergyChargeGain once per attacked mob
	// (CloseRangeDamageHandler.java:136-140); Atlas collapses the loop into
	// one emit of 102 x mobs so the attack path costs at most one Kafka
	// message (NFR-1).
	energyChargeGainPerMob = int32(102)
	// energyChargeCap is the accumulation ceiling. Reaching it promotes the
	// character to the charged state.
	energyChargeCap = int32(10000)
	// energyChargedValue is the charged-state SENTINEL, not a bar reading.
	// Nothing may treat it as "150% full" (FR-3.1).
	energyChargedValue = int32(15000)
)

// energyLine is the Energy Charge skill line a character owns: the tenant's
// wire id for that line's Energy Charge skill, plus the character's level in
// it. The level selects the WZ effect row that supplies the charged window's
// duration and its weapon-attack payoff.
type energyLine struct {
	skillId skill3.Id
	level   byte
}

// energyChargeLine resolves the character's Energy Charge line from owned
// skills, adventurer branch first. ok == false when the character owns
// neither variant at level > 0.
//
// Identities, not raw ids: set.Wire returns false for the Cygnus identity on
// gms_v61 (Thunder Breaker postdates it), so that branch degrades to a no-op
// rather than resolving a bogus id (AC-10). No job check is needed —
// owning the skill at level > 0 already implies the line, which is all
// Cosmic's isCygnus() split was ever deciding.
func energyChargeLine(set skill3.Set, skills []skill.Model) (energyLine, bool) {
	find := func(id skill3.Id) byte {
		for _, s := range skills {
			if s.Id() == id {
				return s.Level()
			}
		}
		return 0
	}
	for _, identity := range []skill3.Identity{skill3.MarauderEnergyCharge, skill3.ThunderBreakerStage2EnergyCharge} {
		wire, ok := set.Wire(identity)
		if !ok {
			continue
		}
		if lvl := find(wire); lvl > 0 {
			return energyLine{skillId: wire, level: lvl}, true
		}
	}
	return energyLine{}, false
}

// energyChargeGainAmount is the bar gain for one attack: 102 per attacked
// monster, or nothing when the attack hit nothing (FR-2.4).
func energyChargeGainAmount(mobsHit int) int32 {
	if mobsHit <= 0 {
		return 0
	}
	return energyChargeGainPerMob * int32(mobsHit)
}

// energyChargeQualifies reports whether an attack feeds the energy bar. Melee
// covers every close-range attack including the basic attack (skillId 0);
// AttackTypeEnergy is the Energy Charge aura's own touch damage, which Cosmic
// routes through the same close-range handler; on the ranged path ONLY
// Thunder Breaker Shark Wave qualifies (RangedAttackHandler.java:90-99).
func energyChargeQualifies(at packetmodel.AttackType, attackId skill3.Identity, attackIdOk bool) bool {
	switch at {
	case packetmodel.AttackTypeMelee, packetmodel.AttackTypeEnergy:
		return true
	case packetmodel.AttackTypeRanged:
		return attackIdOk && skill3.IsIdentity(attackId, skill3.ThunderBreakerStage3SharkWave)
	}
	return false
}

// isEnergyBlast reports whether a cast is the charge-gated Energy Blast.
// Energy Blast is the only skill in the family that carries no mpCon at any
// level — the sole WZ evidence that it is energy-costed rather than
// MP-costed. Shockwave (mpCon 18) and Shark Wave (mpCon 15) are NOT gated
// (design.md OQ-3).
func isEnergyBlast(attackId skill3.Identity, attackIdOk bool) bool {
	return attackIdOk && skill3.IsIdentity(attackId, skill3.MarauderEnergyBlast, skill3.ThunderBreakerStage2EnergyBlast)
}

// energyChargeDeps groups the side-effecting call energyChargeTryUpdate makes
// so tests can drive every branch without a real processor or Kafka producer.
type energyChargeDeps struct {
	emitUpsert func(sourceId int32, level byte, amount int32, capValue int32) error
}

// energyChargeProductionDeps wires energyChargeDeps to the buff
// UPDATE_STAT_VALUE emitter for one attack. CreateIfMissing is what keeps the
// attack path free of a REST read: the channel never has to ask whether the
// bar buff exists (design.md §4.2).
func energyChargeProductionDeps(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32) energyChargeDeps {
	bp := buff.NewProcessor(l, ctx)
	return energyChargeDeps{
		emitUpsert: func(sourceId int32, level byte, amount int32, capValue int32) error {
			return bp.UpdateStatValue(f, characterId, buff.StatValueUpdate{
				SourceId:        sourceId,
				StatType:        string(constants.TemporaryStatTypeEnergyCharge),
				Operation:       buff2.StatOperationIncrement,
				Amount:          amount,
				Cap:             capValue,
				CreateIfMissing: true,
				Level:           level,
			})
		},
	}
}

// energyChargeTryUpdate applies Energy Charge bar bookkeeping for one
// qualifying attack: at most ONE emit, carrying 102 x mobs clamped to 10000.
//
// "No gain while charged" (FR-2.5) is structural rather than a guard here:
// at the 15000 sentinel atlas-buffs' current >= cap test makes the increment
// a no-op and emits no status event, so the channel broadcasts nothing.
//
// All failures are logged and swallowed — the attack pipeline never fails on
// energy bookkeeping (FR-2.6 / NFR-2).
func energyChargeTryUpdate(l logrus.FieldLogger, set skill3.Set, c character.Model, ai packetmodel.AttackInfo, deps energyChargeDeps) {
	line, ok := energyChargeLine(set, c.Skills())
	if !ok {
		return
	}
	amount := energyChargeGainAmount(len(ai.DamageInfo()))
	if amount == 0 {
		return
	}
	if err := deps.emitUpsert(int32(line.skillId), line.level, amount, energyChargeCap); err != nil {
		l.WithError(err).Errorf("Energy Charge: gain emit failed for character [%d] energy line [%d].", c.Id(), line.skillId)
	}
}
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'TestEnergyCharge|TestIsEnergyBlast' -v`
Expected: PASS for every subtest.

- [x] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_energy_charge.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_attack_energy_charge_test.go
git commit -m "feat(channel): Energy Charge line resolution and bar gain"
```

---

### Task 7: atlas-channel — wire accumulation into the attack pipeline

The call site sits beside `comboOrbTryUpdate`, with a wider gate: melee **or** energy/touch **or** ranged-Shark-Wave. FR-7 (Energy Charge must not re-apply its own effect on touch damage) needs **no code** — Atlas's attack path applies no skill statups; effect application lives behind `handler.UseSkill`, reached only from the `USE_SKILL` packet, and the attack path's only per-skill hook is the `LookupAttackCast` registry, which Energy Charge is not and must not be registered in. What FR-7 needs is a test pinning that invariant and a comment saying why.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (accumulation call beside `comboOrbTryUpdate`, `:980-982`)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_energy_charge_test.go` (append)

**Interfaces:**
- Consumes: `energyChargeQualifies`, `energyChargeTryUpdate`, `energyChargeProductionDeps` (Task 6); `attackId`/`attackIdOk` and `set`, already resolved at `character_attack_common.go:756`.
- Produces: nothing new.

- [x] **Step 1: Write the failing test**

Append to `character_attack_energy_charge_test.go`:

```go
// AC-12 / FR-7.1: Energy Charge must never be registered as an attack-cast
// handler. Atlas's attack path applies no skill statups of its own — the only
// per-skill hook it consults is the LookupAttackCast registry — so registering
// Energy Charge there would reintroduce exactly the bug Cosmic patched: the
// aura's own touch damage perpetually refreshing the charged window
// (AbstractDealDamageHandler.java:183-184).
func TestEnergyChargeIsNotAnAttackCastHandler(t *testing.T) {
	for _, id := range []skill3.Id{skill3.MarauderEnergyChargeId, skill3.ThunderBreakerStage2EnergyChargeId} {
		if _, ok := handler.LookupAttackCast(id); ok {
			t.Fatalf("skill [%d] must not be registered as an attack-cast handler; its own touch damage would refresh the charged window", id)
		}
	}
}
```

Import the same `handler` package alias `character_attack_common.go` uses for `LookupAttackCast` (see its imports; the registry lives in the channel's `skill/handler` package). Match the exact `LookupAttackCast` signature there — `character_attack_common.go:730` calls `handler.LookupAttackCast(castId)` and the id argument type must match.

- [x] **Step 2: Run the test to verify it passes immediately**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestEnergyChargeIsNotAnAttackCastHandler -v`
Expected: PASS. This is a regression guard, not a red-then-green cycle — it pins an invariant the design proved already holds. If it FAILS, stop: something registers Energy Charge as an attack cast and FR-7 needs real code after all.

- [x] **Step 3: Implement the call site**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`, immediately after the existing combo block at `:980-982`:

```go
					if ai.AttackType() == packetmodel.AttackTypeMelee {
						comboOrbTryUpdate(l, c, ai, comboOrbProductionDeps(l, ctx, s.Field(), s.CharacterId()))
					}

					// Energy Charge bar gain (task-216). Wider gate than Combo:
					// every close-range attack, the Energy Charge aura's own
					// touch damage (AttackTypeEnergy), and — on the ranged path
					// only — Thunder Breaker Shark Wave. Fire-and-forget beside
					// the combo emit: at most one Kafka message, zero REST, and
					// no branch can fail the attack (NFR-1 / NFR-2). The
					// character was fetched with SkillModelDecorator, so the
					// energy skill level is already in hand.
					//
					// Note there is deliberately NO "don't refresh Energy Charge
					// on its own touch damage" guard here (Cosmic's
					// AbstractDealDamageHandler.java:183-184). Atlas's attack
					// path applies no skill statups at all, so the aura cannot
					// refresh itself — see TestEnergyChargeIsNotAnAttackCastHandler.
					if energyChargeQualifies(ai.AttackType(), attackId, attackIdOk) {
						energyChargeTryUpdate(l, set.Skill, c, ai, energyChargeProductionDeps(l, ctx, s.Field(), s.CharacterId()))
					}
```

Confirm the field name for the skill set on the value returned by `constants.For(...)` (the surrounding code already uses `set.Skill.Resolve` at `:757`), and use the same expression.

- [x] **Step 4: Verify**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./socket/handler/`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_attack_energy_charge_test.go
git commit -m "feat(channel): feed the energy bar from melee, touch, and Shark Wave"
```

---

### Task 8: atlas-channel — the Energy Blast cast gate

Energy Blast is an *attack* skill (WZ shows `damage`/`mobCount`/`lt`/`rb` and no `time`), so it arrives on a melee ATTACK packet and lands in `processAttack` — a gate in `character_skill_use.go` would never fire. The correct precedent is `battleshipAttackPermitted`: a soft rejection before any cost, damage, or broadcast, returning `nil` rather than destroying the session.

The gate **fails open** on an unknown bar (no mirror entry — fresh channel, pod restart): a missing entry means "unknown", and an unknown must never eat a legitimate cast. On a genuine rejection it re-announces the authoritative bar so a desynced client resynchronises instead of losing the skill silently (OQ-1 resolution (b)).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_energy_charge.go` (append)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (`processAttack`, beside the battleship gate at `:785-791`)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_energy_charge_test.go` (append)

**Interfaces:**
- Consumes: `buff.GetEnergyMirror()` (Task 5), `isEnergyBlast` / `energyChargedValue` (Task 6).
- Produces: `energyBlastPermitted(t tenant.Model, characterId uint32, attackId skill3.Identity, attackIdOk bool) (bool, int32)` and `energyReannounceAuthoritative(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model)`.

- [x] **Step 1: Write the failing test**

Append to `character_attack_energy_charge_test.go`:

```go
func TestEnergyBlastPermitted(t *testing.T) {
	energyTestResetMirror()
	tn := energyTestTenant(t)
	v83 := energyTestSet(83)
	blast, _ := v83.Resolve(skill3.MarauderEnergyBlastId)
	shockwave, _ := v83.Resolve(skill3.MarauderShockwaveId)

	t.Run("non-blast always permitted", func(t *testing.T) {
		ok, _ := energyBlastPermitted(tn, 100, shockwave, true)
		if !ok {
			t.Fatal("a skill outside the blast pair must never be gated")
		}
	})

	t.Run("charged permitted", func(t *testing.T) {
		buff.GetEnergyMirror().Set(tn, 100, 15000)
		ok, _ := energyBlastPermitted(tn, 100, blast, true)
		if !ok {
			t.Fatal("a charged caster must be permitted")
		}
	})

	t.Run("below full rejected and reports the bar", func(t *testing.T) {
		buff.GetEnergyMirror().Set(tn, 100, 4998)
		ok, v := energyBlastPermitted(tn, 100, blast, true)
		if ok {
			t.Fatal("a partial bar must be rejected")
		}
		if v != 4998 {
			t.Fatalf("rejection must report the mirrored bar: got %d want 4998", v)
		}
	})

	t.Run("empty bar rejected", func(t *testing.T) {
		buff.GetEnergyMirror().Set(tn, 100, 0)
		if ok, _ := energyBlastPermitted(tn, 100, blast, true); ok {
			t.Fatal("a known-empty bar must be rejected")
		}
	})

	// Fail open: an unknown bar (fresh channel, pod restart) must never eat a
	// legitimate cast (design.md §4.4).
	t.Run("unknown bar permitted", func(t *testing.T) {
		buff.GetEnergyMirror().Clear(tn, 100)
		if ok, _ := energyBlastPermitted(tn, 100, blast, true); !ok {
			t.Fatal("an unknown bar must fail open")
		}
	})
}
```

Add two helpers in the test file: `energyTestResetMirror()` resetting the `buff` package singleton is not possible from `handler` (the reset vars are package-private to `buff`), so instead have `energyTestResetMirror()` clear the entries it sets via `buff.GetEnergyMirror().Clear(...)` at the start of each subtest, and `energyTestTenant(t)` build a tenant with `tenant.Create(uuid.New(), "GMS", 83, 1)`. Keep character ids distinct per subtest if that reads more cleanly.

- [x] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestEnergyBlastPermitted`
Expected: FAIL to compile — "undefined: energyBlastPermitted".

- [x] **Step 3: Implement**

Append to `character_attack_energy_charge.go` (add `atlas-channel/session`, `atlas-channel/socket/writer`, `charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"` and `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"` to the imports):

```go
// energyBlastPermitted gates Energy Blast on the caster being charged. Energy
// Blast is an ATTACK skill (WZ: damage/mobCount/lt/rb, no time), so it never
// reaches CharacterUseSkillHandleFunc — the gate belongs beside
// battleshipAttackPermitted in processAttack, and the rejection stays soft
// (return false, never destroy the session).
//
// Reads the pod-local mirror: zero I/O on the permitted path. Returns the
// mirrored bar alongside the verdict so the caller can log it and re-announce.
//
// Fails OPEN on a missing mirror entry. A miss means "unknown" — a fresh
// channel or a restarted pod, not an empty bar — and an unknown must never eat
// a legitimate cast. A KNOWN zero, by contrast, is a real reading and is
// rejected.
//
// This is a deliberate divergence from Cosmic, which performs no server-side
// charge check at all; no client-side gate was found in the v83 IDB either
// (design.md OQ-3). The fail-open plus the re-announce below bound the damage
// to "one cast allowed that Cosmic would also have allowed".
func energyBlastPermitted(t tenant.Model, characterId uint32, attackId skill3.Identity, attackIdOk bool) (bool, int32) {
	if !isEnergyBlast(attackId, attackIdOk) {
		return true, 0
	}
	v, ok := buff.GetEnergyMirror().Get(t, characterId)
	if !ok {
		return true, 0
	}
	return v == energyChargedValue, v
}

// energyReannounceAuthoritative re-sends the caster's true ENERGY_CHARGE bar
// after a rejected Energy Blast, so a client whose bar drifted (a dropped
// STAT_UPDATED, a reconnect before the buff replayed) resynchronises instead
// of losing the skill with no feedback (design.md OQ-1 resolution (b)).
//
// This is the ONE REST call the gate is allowed, and only on a rejection —
// the permitted path stays I/O-free. Failures are logged and swallowed: the
// rejection itself already happened.
func energyReannounceAuthoritative(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model) {
	bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
	if err != nil {
		l.WithError(err).Errorf("Energy Charge: unable to read authoritative bar for character [%d] after a rejected cast.", s.CharacterId())
		return
	}
	t := tenant.MustFromContext(ctx)
	for _, b := range bs {
		for _, c := range b.Changes() {
			if c.Type() != string(constants.TemporaryStatTypeEnergyCharge) {
				continue
			}
			buff.GetEnergyMirror().Set(t, s.CharacterId(), c.Amount())
			if aerr := session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffGiveWriter)(writer.CharacterBuffGiveBody([]buff.Model{b}))(s); aerr != nil {
				l.WithError(aerr).Errorf("Energy Charge: bar re-announce failed for character [%d].", s.CharacterId())
			}
			return
		}
	}
	// No ENERGY_CHARGE buff upstream at all: the mirror was stale. Clear it so
	// the next cast fails open rather than being rejected forever.
	buff.GetEnergyMirror().Clear(t, s.CharacterId())
}
```

Then wire the gate into `processAttack`, immediately after the battleship gate in `character_attack_common.go` (inside the same `if ai.SkillId() > 0 {` block, after the `battleshipAttackPermitted` rejection):

```go
						// Energy Blast requires a full energy bar (task-216
						// FR-6). Same soft-rejection posture as the battleship
						// gate — before any cost, damage, or broadcast, and
						// returning nil rather than destroying the session.
						// The bar is NOT consumed by a successful cast; only
						// the charged window's own timer resets it.
						if permitted, bar := energyBlastPermitted(t, s.CharacterId(), attackId, attackIdOk); !permitted {
							l.WithFields(logrus.Fields{
								"character_id": s.CharacterId(),
								"skill_id":     ai.SkillId(),
								"energy_bar":   bar,
							}).Debug("energy_blast_rejected_not_charged")
							energyReannounceAuthoritative(l, ctx, wp, s)
							return nil
						}
```

`t`, `attackId`, and `attackIdOk` are all already in scope from `:756-758`.

- [x] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./socket/handler/ -run 'TestEnergy' -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_energy_charge.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_attack_energy_charge_test.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
git commit -m "feat(channel): gate Energy Blast on a full energy bar"
```

---

### Task 9: atlas-channel — mirror maintenance, effect announce, and the charged promotion

Three reactions hang off the existing buff-status consumer:

1. **Mirror maintenance** — `APPLIED`/`STAT_UPDATED` set the bar, `EXPIRED` clears it.
2. **Skill-use effect** (FR-4.2) — emitted here rather than at the attack site, so it fires once per *actual value change*: a hit against a full bar produces no packet.
3. **The 10000 → 15000 promotion** (FR-3.1) — exactly-once by construction, because atlas-buffs emits `STAT_UPDATED` only when the value actually changed and clamps at the cap, so exactly one event in the bar's life carries 10000.

The owner/foreign `GIVE_BUFF` broadcast and the `EXPIRED` cancel pair already exist and need no change (OQ-4): `handleStatusEventExpired` already sends `CharacterBuffCancel` to the owner and `CharacterBuffCancelForeign` to the map, and `EncodeMask` claims exactly the stats the CTS holds, so the reset names `ENERGY_CHARGE` and nothing else.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go`
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/energy_test.go` (new)

**Interfaces:**
- Consumes: `buff.GetEnergyMirror()` (Task 5); `energyChargedValue`/`energyChargeCap` — **re-declare these as consumer-local constants rather than importing them**, because `socket/handler` already imports this package's siblings and the channel avoids that direction of dependency. Name them `energyChargeCapValue` and `energyChargedValue` in the consumer package and comment that they mirror `socket/handler`'s.
- Produces: `energyChargeChange(changes []buff2.StatChange) (buff2.StatChange, bool)`, `energyChargeShouldPromote(amount int32) bool`, and `energyChargeReact(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId uint32, sourceId int32, level byte, c buff2.StatChange)`.

- [x] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/energy_test.go`:

```go
package buff

import (
	buff2 "atlas-channel/kafka/message/buff"
	"testing"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

func TestEnergyChargeChange(t *testing.T) {
	t.Run("finds the energy change", func(t *testing.T) {
		c, ok := energyChargeChange([]buff2.StatChange{
			{Type: "COMBO", Amount: 3},
			{Type: string(charconst.TemporaryStatTypeEnergyCharge), Amount: 4998},
		})
		if !ok || c.Amount != 4998 {
			t.Fatalf("got (%+v,%v)", c, ok)
		}
	})
	t.Run("absent", func(t *testing.T) {
		if _, ok := energyChargeChange([]buff2.StatChange{{Type: "COMBO", Amount: 3}}); ok {
			t.Fatal("must not match an unrelated stat")
		}
	})
	t.Run("empty", func(t *testing.T) {
		if _, ok := energyChargeChange(nil); ok {
			t.Fatal("must not match an empty change set")
		}
	})
}

// The promotion fires on EXACTLY the value that tops the accumulation cap.
// atlas-buffs emits STAT_UPDATED only on a real change and clamps at the cap,
// so exactly one event in the bar's life carries 10000.
func TestEnergyChargeShouldPromote(t *testing.T) {
	for _, tc := range []struct {
		amount int32
		want   bool
	}{
		{0, false},
		{102, false},
		{9999, false},
		{10000, true},
		{15000, false},
	} {
		if got := energyChargeShouldPromote(tc.amount); got != tc.want {
			t.Fatalf("amount=%d: got %v want %v", tc.amount, got, tc.want)
		}
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/buff/ -run TestEnergyCharge`
Expected: FAIL to compile — "undefined: energyChargeChange".

- [x] **Step 3: Implement**

Add to `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go`, near `beaconChange`/`isBattleshipRide`:

```go
const (
	// energyChargeCapValue and energyChargedValue mirror the constants in
	// socket/handler/character_attack_energy_charge.go. Re-declared rather
	// than imported: the consumer must not depend on the socket handler
	// package. 15000 is a charged-state SENTINEL, not a bar reading.
	energyChargeCapValue = int32(10000)
	energyChargedValue   = int32(15000)
)

// energyChargeChange returns the event's ENERGY_CHARGE stat change, if any.
func energyChargeChange(changes []buff2.StatChange) (buff2.StatChange, bool) {
	for _, c := range changes {
		if c.Type == string(charconst.TemporaryStatTypeEnergyCharge) {
			return c, true
		}
	}
	return buff2.StatChange{}, false
}

// energyChargeShouldPromote reports whether a bar reading is the one that
// tops the accumulation cap and therefore triggers the charged state.
func energyChargeShouldPromote(amount int32) bool {
	return amount == energyChargeCapValue
}

// energyChargeReact is the whole Energy Charge reaction to one buff-status
// event carrying an ENERGY_CHARGE change: refresh the pod-local mirror the
// cast gate reads, announce the skill-use effect to the owner and the map,
// and — when the bar just topped out — promote to the charged state.
//
// Announcing the effect HERE rather than at the attack site is what keeps it
// honest: atlas-buffs emits a status event only when the value actually
// changed, so a hit against a full bar produces no packet.
func energyChargeReact(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId uint32, sourceId int32, level byte, c buff2.StatChange) {
	t := tenant.MustFromContext(ctx)
	buff.GetEnergyMirror().Set(t, characterId, c.Amount)

	_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId, func(s session.Model) error {
		cp := character.NewProcessor(l, ctx)
		ch, cerr := cp.GetById()(characterId)
		if cerr != nil {
			l.WithError(cerr).Errorf("Energy Charge: unable to read character [%d] for the skill-use effect.", characterId)
		} else {
			if aerr := socketHandler.AnnounceSkillUse(l)(ctx)(wp)(uint32(sourceId), ch.Level(), level)(s); aerr != nil {
				l.WithError(aerr).Errorf("Energy Charge: skill-use effect write failed for character [%d].", characterId)
			}
			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), characterId,
				socketHandler.AnnounceForeignSkillUse(l)(ctx)(wp)(characterId, uint32(sourceId), ch.Level(), level))
		}

		if !energyChargeShouldPromote(c.Amount) {
			return nil
		}

		// The charged window's length is the Energy Charge effect's `time` at
		// the character's skill level (31s at L1, 40s at L20 for 5110001; the
		// Cygnus table differs, which is why the level travels with the event
		// rather than being assumed). Duration() ALREADY returns milliseconds
		// and ApplyCommandBody.Duration is milliseconds — no scaling here.
		// tools/buff-duration-guard.sh fails CI on a seconds-valued emitter.
		se, eerr := dataskill.NewProcessor(l, ctx).GetEffect(uint32(sourceId), level)
		if eerr != nil {
			l.WithError(eerr).Errorf("Energy Charge: effect lookup failed for character [%d] skill [%d] level [%d]; the bar stays full but uncharged.", characterId, sourceId, level)
			return nil
		}

		// The charged APPLY REPLACES the accumulating buff in place: both
		// phases share srcKey(sourceId) in atlas-buffs, so there is never a
		// moment with two Energy Charge buffs.
		if perr := buff.NewProcessor(l, ctx).Apply(s.Field(), characterId, sourceId, level, se.Duration(),
			[]statup.Model{statup.NewModel(string(charconst.TemporaryStatTypeEnergyCharge), energyChargedValue)})(characterId); perr != nil {
			l.WithError(perr).Errorf("Energy Charge: charged APPLY emit failed for character [%d].", characterId)
		}
		return nil
	})
}
```

Add imports for `atlas-channel/character`, `atlas-channel/data/skill/effect/statup`, and confirm `dataskill` (already imported as `dataskill "atlas-channel/data/skill"`), `socketHandler`, `_map`, `session`, `charconst` are present — all six already are except `character` and `statup`. Verify `cp.GetById()` is the right call shape (`processor.go:71` takes variadic decorators, so `GetById()(characterId)` is correct).

Then call it from the three handlers:

- In `handleStatusEventApplied`, immediately after the beacon mirror block:

```go
			if ec, ok := energyChargeChange(e.Body.Changes); ok {
				energyChargeReact(l, ctx, sc, wp, e.CharacterId, e.Body.SourceId, e.Body.Level, ec)
			}
```

- In `handleStatusEventStatUpdated`, before the `announceBuffGive` call:

```go
		if ec, ok := energyChargeChange(e.Body.Changes); ok {
			energyChargeReact(l, ctx, sc, wp, e.CharacterId, e.Body.SourceId, e.Body.Level, ec)
		}
```

- In `handleStatusEventExpired`, beside the beacon `Clear`:

```go
			if _, ok := energyChargeChange(e.Body.Changes); ok {
				buff.GetEnergyMirror().Clear(t, e.CharacterId)
			}
```

Note the charged `APPLY` carries `ENERGY_CHARGE = 15000` and no other statup: the weapon-attack payoff is resolved server-side in atlas-effective-stats (Task 10), deliberately NOT sent as a `PAD` statup, because a `PAD` statup is a CTS bit and would light up a weapon-attack buff icon that neither Cosmic nor the real client shows for Energy Charge.

- [x] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./kafka/consumer/buff/ -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go \
        services/atlas-channel/atlas.com/channel/kafka/consumer/buff/energy_test.go
git commit -m "feat(channel): announce the energy bar and promote it to charged"
```

---

### Task 10: atlas-effective-stats — the charged weapon-attack bonus

While charged, the character gains the Energy Charge effect's `pad` as weapon attack (`Character.java:7676-7680`). `ENERGY_CHARGE` must **not** go through `BonusesForBuffChange`: the stat's amount is the bar reading (0–15000), not an attack value, so feeding it in directly would grant a five-digit stat. `BonusesForBuffChange` has no `ENERGY_CHARGE` arm today, so the hazard is already avoided by default — this special case is additive, not corrective.

Verified `pad` values for `5110001` (design.md §1.3): 0 at levels 1–3, 11 at levels 4–5, 15 at level 20. The Cygnus table differs (`15100004` has `pad 11` from level 2 and 20 levels total), which is exactly why the value is read from the buff's own `sourceId` + `level` and never hard-coded.

**Files:**
- Modify: `services/atlas-effective-stats/atlas.com/effective-stats/external/buffs/rest.go` (`BuffRestModel`)
- Modify: `services/atlas-effective-stats/atlas.com/effective-stats/character/initializer.go` (`fetchBuffBonuses`, `:174-198`)
- Test: `services/atlas-effective-stats/atlas.com/effective-stats/character/energy_charge_test.go` (new)

**Interfaces:**
- Consumes: `buffs.BuffRestModel` (widened here), `skilldata.RestModel.GetEffectForLevel`, `stat.NewBonus`, `stat.TypeWeaponAttack`.
- Produces: `energyChargeBonus(source string, sourceId int32, level byte, amount int32, effectFor func(skillId uint32, level byte) (*skilldata.EffectModel, error)) []stat.Bonus`.

- [x] **Step 1: Write the failing test**

Create `services/atlas-effective-stats/atlas.com/effective-stats/character/energy_charge_test.go`:

```go
package character

import (
	skilldata "atlas-effective-stats/external/data/skill"
	"atlas-effective-stats/stat"
	"errors"
	"testing"
)

func energyTestEffect(pad int16) func(uint32, byte) (*skilldata.EffectModel, error) {
	return func(uint32, byte) (*skilldata.EffectModel, error) {
		return &skilldata.EffectModel{WeaponAttack: pad}, nil
	}
}

func TestEnergyChargeBonus(t *testing.T) {
	// Charged: pad 15 at level 20 for 5110001, per WZ Skill.wz/511.img.xml.
	t.Run("charged grants pad as weapon attack", func(t *testing.T) {
		bs := energyChargeBonus("buff:5110001", 5110001, 20, 15000, energyTestEffect(15))
		if len(bs) != 1 {
			t.Fatalf("expected 1 bonus, got %d", len(bs))
		}
		if bs[0].Type() != stat.TypeWeaponAttack || bs[0].Amount() != 15 {
			t.Fatalf("got %s=%d want weapon_attack=15", bs[0].Type(), bs[0].Amount())
		}
	})

	// FR-5.3: the bar reading is NOT a stat value. A partial bar grants
	// nothing at all — never 4998 weapon attack.
	t.Run("partial bar grants nothing", func(t *testing.T) {
		if bs := energyChargeBonus("buff:5110001", 5110001, 20, 4998, energyTestEffect(15)); len(bs) != 0 {
			t.Fatalf("expected no bonus below the charged sentinel, got %+v", bs)
		}
	})

	// pad is 0 at levels 1-3; a zero bonus is omitted rather than emitted.
	t.Run("charged with pad 0 grants nothing", func(t *testing.T) {
		if bs := energyChargeBonus("buff:5110001", 5110001, 1, 15000, energyTestEffect(0)); len(bs) != 0 {
			t.Fatalf("expected no bonus for pad 0, got %+v", bs)
		}
	})

	t.Run("effect lookup failure grants nothing", func(t *testing.T) {
		fail := func(uint32, byte) (*skilldata.EffectModel, error) { return nil, errors.New("data down") }
		if bs := energyChargeBonus("buff:5110001", 5110001, 20, 15000, fail); len(bs) != 0 {
			t.Fatalf("expected no bonus on lookup failure, got %+v", bs)
		}
	})

	t.Run("nil effect grants nothing", func(t *testing.T) {
		nilEffect := func(uint32, byte) (*skilldata.EffectModel, error) { return nil, nil }
		if bs := energyChargeBonus("buff:5110001", 5110001, 99, 15000, nilEffect); len(bs) != 0 {
			t.Fatalf("expected no bonus for a missing effect row, got %+v", bs)
		}
	})
}
```

Check `stat.Bonus`'s accessor names in `stat/model.go` and adjust `bs[0].Type()`/`bs[0].Amount()` to match.

- [x] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-effective-stats/atlas.com/effective-stats && go test ./character/ -run TestEnergyChargeBonus`
Expected: FAIL to compile — "undefined: energyChargeBonus".

- [x] **Step 3: Implement**

First widen `external/buffs/rest.go` — a pure widening of an existing payload; atlas-buffs already serves `level` (`services/atlas-buffs/atlas.com/buffs/buff/rest.go:13`):

```go
// BuffRestModel represents a buff from atlas-buffs service
type BuffRestModel struct {
	Id string `json:"-"`
	SourceId int32 `json:"sourceId"`
	// Level is the source skill level. Needed to resolve level-dependent
	// payoffs from skill effect data (task-216: Energy Charge's `pad`).
	Level     byte            `json:"level"`
	Duration  int32           `json:"duration"`
	Changes   []StatRestModel `json:"changes"`
	CreatedAt time.Time       `json:"createdAt"`
	ExpiresAt time.Time       `json:"expiresAt"`
}
```

Then add the helper and wire it in `character/initializer.go`:

```go
// energyChargeBonus turns a charged ENERGY_CHARGE buff into its weapon-attack
// payoff (Cosmic Character.java:7676-7680, localwatk += ceffect.getWatk()).
//
// ENERGY_CHARGE is deliberately NOT routed through BonusesForBuffChange: the
// stat's amount is the ENERGY BAR READING (0..15000), not an attack value, so
// feeding it in directly would grant a five-digit weapon attack. The bonus is
// resolved from the skill effect's `pad` at the buff's own level instead, and
// only while the bar holds the 15000 charged sentinel.
//
// Level matters: 5110001 has pad 0 at L1-3, 11 at L4-5, 15 at L20, and the
// Cygnus table (15100004) differs again — hence the per-buff lookup rather
// than a constant.
func energyChargeBonus(source string, sourceId int32, level byte, amount int32, effectFor func(skillId uint32, level byte) (*skilldata.EffectModel, error)) []stat.Bonus {
	if amount != energyChargedValue {
		return nil
	}
	effect, err := effectFor(uint32(sourceId), level)
	if err != nil || effect == nil {
		return nil
	}
	if effect.WeaponAttack <= 0 {
		return nil
	}
	return []stat.Bonus{stat.NewBonus(source, stat.TypeWeaponAttack, int32(effect.WeaponAttack))}
}
```

Declare the sentinel next to it:

```go
// energyChargedValue is the ENERGY_CHARGE charged-state sentinel emitted by
// atlas-channel. It is a state marker, not a bar reading, and nothing may
// treat it as a magnitude. (task-216 FR-3.1)
const energyChargedValue = int32(15000)
```

And in `fetchBuffBonuses`, ahead of the generic dispatch:

```go
	bonuses := make([]stat.Bonus, 0)
	effectFor := func(skillId uint32, level byte) (*skilldata.EffectModel, error) {
		si, err := skilldata.RequestById(skillId)(l, ctx)
		if err != nil {
			return nil, err
		}
		return si.GetEffectForLevel(level), nil
	}
	for _, buff := range buffList {
		source := fmt.Sprintf("buff:%d", buff.SourceId)
		for _, change := range buff.Changes {
			if change.Type == string(charconst.TemporaryStatTypeEnergyCharge) {
				bonuses = append(bonuses, energyChargeBonus(source, buff.SourceId, buff.Level, change.Amount, effectFor)...)
				continue
			}
			bs := stat.BonusesForBuffChange(source, change.Type, change.Amount)
			if len(bs) == 0 {
				l.Debugf("Unknown buff stat type: %s", change.Type)
				continue
			}
			bonuses = append(bonuses, bs...)
		}
	}
```

Add the `charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"` import (or, if that library is not already a dependency of this module, use the literal `"ENERGY_CHARGE"` with a comment naming `libs/atlas-constants/character/temporary_stat.go:116` as its source — do NOT add a new module dependency, per Global Constraints).

- [x] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-effective-stats/atlas.com/effective-stats && go build ./... && go test -race ./... && go vet ./...`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-effective-stats/atlas.com/effective-stats/external/buffs/rest.go \
        services/atlas-effective-stats/atlas.com/effective-stats/character/initializer.go \
        services/atlas-effective-stats/atlas.com/effective-stats/character/energy_charge_test.go
git commit -m "feat(effective-stats): charged Energy Charge grants its effect's weapon attack"
```

---

### Task 11: Coverage-matrix reconciliation and full verification sweep

AC-9 requires the `ENERGY_CHARGE` encoding to be **verified against the coverage matrix, not asserted**. Task 1 changed how `GIVE_BUFF` / `GIVE_FOREIGN_BUFF` / `CANCEL_BUFF` serialize one base block, so those rows must be re-checked rather than assumed still-good. As of the current `docs/packets/audits/STATUS.md`:

- `GIVE_BUFF` (`:63`) is ✅ on v48/v61/v72/v79/v83/v84/v87/v95/JMS185 and **❌ on v92**.
- `CANCEL_BUFF` (`:65`) is ✅ on the same set and **❌ on v92**.
- `GIVE_FOREIGN_BUFF` (`:266`) is ⬜ on v48, ✅ elsewhere, and **❌ on v92**.

No existing fixture pins an `ENERGY_CHARGE` base block, so no verified cell asserts the zeros Task 1 replaced — but that claim must be *confirmed by running the fixtures*, not repeated.

**Files:**
- Modify: `docs/tasks/task-216-energy-charge/plan.md` (check off completed steps)
- Possibly modify: `docs/packets/audits/` artifacts — only if a cell actually moves.

**Interfaces:** none.

- [x] **Step 1: Confirm no verified cell asserted the zeros**

Run:
```bash
grep -rn "ENERGY_CHARGE" libs/atlas-packet/ | grep -i test
cd libs/atlas-packet && go test ./... 2>&1 | tail -20
```
Expected: the grep shows only the new task-216 fixtures plus pre-existing comments about the two-state group's mask shifts; the full atlas-packet suite is green. If any pre-existing fixture fails, STOP — a verified cell DID assert the zeros, and the change needs per-version re-derivation before it can land.

- [x] **Step 2: Record the AC-9 outcome**

Append a short "AC-9 — ENERGY_CHARGE encoding coverage" section to `docs/tasks/task-216-energy-charge/context.md` stating, per version in the §7.4 supported set (gms_v61 through jms_v185), whether the `GIVE_BUFF` / `GIVE_FOREIGN_BUFF` / `CANCEL_BUFF` cell is verified today and, where it is not (v92 on all three rows), that it is **out of scope with the reason "the row was already ❌ before this task; task-216 makes no wire-shape change, only a value change"**. Do not claim a promotion that did not happen.

- [x] **Step 3: Run every guard from the repo root**

Run:
```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/buff-duration-guard.sh
tools/skill-job-id-guard.sh
tools/lint.sh --check
```
Expected: all exit 0. `tools/lint.sh` (no flags) rewrites files in place — run the fix mode first if `--check` complains about formatting, then re-run `--check`. If `lint.sh --check` fails with an nvm/node error rather than a real finding, source nvm (`nvm use 22`) and re-run; a cross-worktree golangci-lint lock can also cause a spurious failure — retry once before treating it as real.

No template, opcode, docker-bake, k8s, or `services.json` change was made, so `tools/template-*-guard.sh`, `tools/service-registration-guard.sh`, and `tools/trade-contract-mirror-guard.sh` are not applicable.

- [x] **Step 4: Run the full per-module verification**

Run:
```bash
for m in libs/atlas-packet \
         services/atlas-buffs/atlas.com/buffs \
         services/atlas-channel/atlas.com/channel \
         services/atlas-effective-stats/atlas.com/effective-stats; do
  echo "== $m"
  ( cd "$m" && go build ./... && go vet ./... && go test -race ./... ) || echo "FAILED: $m"
done
```
Expected: no `FAILED:` lines. `libs/atlas-constants` was not modified, so it needs no pass.

- [x] **Step 5: Confirm no `go.mod` moved**

Run: `git diff --name-only main... | grep -E 'go\.(mod|sum)$' || echo "no module changes"`
Expected: `no module changes`. If any `go.mod` DID change, `docker buildx bake atlas-<svc>` becomes mandatory for that service (CLAUDE.md item 4) — run it before claiming the branch is done.

- [x] **Step 6: Commit**

```bash
git add docs/tasks/task-216-energy-charge/context.md
git commit -m "docs(task-216): record AC-9 ENERGY_CHARGE encoding coverage"
```

---

## Post-implementation

Before opening a PR:

1. Run the code-review step (`superpowers:requesting-code-review`, which dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer`). CLAUDE.md forbids skipping it even when the plan looks complete.
2. Run the live pass on a v83 tenant with a Marauder, per design.md §7: bar fills 102/mob, charges at 10000, the aura is visible to a second client in the map, it expires and the bar visibly zeroes (AC-6 / OQ-4's open confirmation), `GET /characters/{id}/effective-stats` weapon attack rises and falls with the window, and Energy Blast is refused below full and accepted at full **without consuming the bar**.

## Known accepted risks (from design.md §8)

- The bar-reads-`nOption` finding is verified on the **GMS v83 client only**. v61's narrower base block shares the field order (task-167), so the same fix applies, but the reader was not re-derived per version. Task 1's per-version fixtures and Task 11's AC-9 record are the mitigation; nothing is claimed that was not run.
- Kafka redelivery of the single 10000-carrying `STAT_UPDATED` would refresh the charged window. Accepted — the same at-least-once posture every other buff emit in this codebase carries.
- FR-6's cast gate is a deliberate divergence from Cosmic, which performs no server-side charge check; no client-side gate was found in the v83 IDB either. The fail-open + re-announce design bounds the damage to "one cast allowed that Cosmic would also have allowed".
