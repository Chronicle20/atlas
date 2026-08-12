# Monster Magnet Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make a Monster Magnet cast (Hero 1121001 / Paladin 1221001 / Dark Knight 1321001) decode correctly on all ten provisioned client versions and produce its four observable effects — grab animation, aggro wipe, forced controller handover to the caster, and a direction-oriented skill effect on remote clients.

**Architecture:** Five independently testable units. A new early-return branch in the shared `SkillUsageInfo` decoder consumes the magnet body (two wire shapes, split at the gms_48/gms_61 boundary). A new `skill/handler/monstermagnet` subpackage validates the client's claimed targets against a server-computed region derived from WZ `range`, then broadcasts the already-built-but-senderless `CATCH_MONSTER` writer to the other sessions in the field and emits two new orthogonal `COMMAND_TOPIC_MONSTER` commands per grabbed monster. atlas-monsters gains a full damage-aggro wipe and a forced-control path that reuses the existing `StartControl` sequencing. The direction byte threads into the existing per-cast SKILL_USE broadcast through a new pair of announce functions.

**Tech Stack:** Go 1.2x, three modules (`libs/atlas-packet`, `services/atlas-channel/atlas.com/channel`, `services/atlas-monsters/atlas.com/monsters`); Kafka via `atlas-kafka`; Redis-backed monster registry via `atlas-redis`; logrus structured logging.

**Paths and shells.** Every path in this plan is repo-relative. Shell blocks that must run from the worktree root open with `cd "$(git rev-parse --show-toplevel)"`; blocks that run inside a Go module `cd` there from the root. Do not write literal home paths into any committed file.

## Global Constraints

Copied verbatim from the PRD/design and from CLAUDE.md. **Every task's requirements implicitly include this section.**

- **No raw wire-id compares for the magnet.** Version gating and skill identification resolve through `constants.For(t.Region(), t.MajorVersion(), t.MinorVersion()).Skill.Resolve(...)` and `skill.IsIdentity(...)`, never `==`/`case`/`skill.Is(...)` against the wire-id constants. `tools/skill-job-id-guard.sh` enforces this.
- **Version gates are `IsRegion` + `MajorAtLeast`, never a raw `>` or `<` on the major.** The magnet gate is exactly `t.IsRegion("GMS") && !t.MajorAtLeast(61)`.
- **Every version gate carries an inline comment naming the decompiled address per version** that justifies it. The existing `isAntiRepeatBuffSkill` gate comment in `skill_usage_info.go:31-39` is the reference for style and depth.
- **No `// TODO`, no stubs, no 501s** in landed commits.
- **No bare `go` statements** — `tools/goroutine-guard.sh`. Nothing in this task needs a goroutine.
- **No keyed Redis commands on a raw go-redis client** outside `libs/atlas-redis` — `tools/redis-key-guard.sh`.
- **Test setup uses the project Builder pattern.** No `*_testhelpers.go` files, no test-only constructors.
- **No literal home/absolute paths** (`/home/<user>/…`) in any committed file.
- **Preserve line endings**; do not normalize CRLF→LF as a side effect.
- **The two monster-command contract copies must be edited in the same commit.** `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go` and `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go` are separate Go modules; a divergent `Type` string or json tag fails no build and silently decodes into a zero-valued body. There is no guard script for this pair.
- **New command bodies stay minimal.** Every handler registered on `COMMAND_TOPIC_MONSTER` json-unmarshals *every* message on the topic into its own body type. A new field name whose Go type disagrees with a sibling body logs one spurious unmarshal error per message.
- **No wire change may alter the decode of any other skill on any version.** The magnet branch is additive, is the first branch, and returns.
- **The packet coverage matrix must stay exactly as it is.** `go run ./tools/packet-audit matrix --check` must exit 0 and `status.json` / `STATUS.md` must be byte-unchanged by this branch. See Task 5 for why no `packet-audit:verify` marker is written.

---

## File Structure

**`libs/atlas-packet`**

| File | Responsibility |
|---|---|
| `model/skill_usage_info.go` (modify) | `MagnetGrab` value type; `magnetGrabs`/`direction` fields, getters, builder setters; `isMonsterMagnet`; `legacyMagnetLayout`; `decodeMagnet`; the early-return branch in `Decode` |
| `model/skill_usage_info_magnet_test.go` (create) | Behavioural decode tests for both shapes + the non-regression matrix |
| `model/skill_usage_info_magnet_versions_test.go` (create) | The ten per-version byte fixtures |

**`services/atlas-channel/atlas.com/channel`**

| File | Responsibility |
|---|---|
| `data/skill/effect/rest.go`, `model.go` (modify) | Decode + expose the WZ `range` attribute |
| `skill/handler/mob_select.go` (modify) | `MagnetRegion`; export `IntersectMobIds`; add `ExceedsMobCap` |
| `skill/handler/common.go` (modify) | `applyToMobs` consumes the two extracted helpers |
| `skill/handler/monstermagnet/monstermagnet.go` (create) | The registered handler: validate → broadcast → emit two commands |
| `skill/handler/registrations/registrations.go` (modify) | Blank import |
| `kafka/message/monster/kafka.go` (modify) | Two command-type consts + two bodies (producer copy) |
| `monster/producer.go` (modify) | Two message providers |
| `monster/processor.go` (modify) | Two emit methods on `Processor` + `ProcessorImpl` |
| `monster/mock/processor.go` (modify) | Two mock fields + methods |
| `socket/handler/effects.go` (modify) | `AnnounceDirectedSkillUse` / `AnnounceForeignDirectedSkillUse` |
| `socket/handler/character_skill_use.go` (modify) | Call the directed variants with `sui.Direction()` |

**`services/atlas-monsters/atlas.com/monsters`**

| File | Responsibility |
|---|---|
| `kafka/consumer/monster/kafka.go` (modify) | Two command-type consts + two bodies (consumer copy) |
| `kafka/consumer/monster/consumer.go` (modify) | Two handler funcs + two registrations |
| `monster/model.go` (modify) | `ControlWithAggro` |
| `monster/registry.go` (modify) | `ClearSummary`; `ClearDamageEntries`; `ControlMonsterWithAggro` |
| `monster/processor.go` (modify) | `ClearAggro`; `ForceControl`; `startControl(…, forceAggro bool)` split |

**No template edits, no new codec, no migrations, no REST changes, no `go.mod` changes.** If a task discovers otherwise, stop and report — it changes which verification gates apply.

---

### Task 1: Magnet wire decode

**Files:**
- Modify: `libs/atlas-packet/model/skill_usage_info.go`
- Test: `libs/atlas-packet/model/skill_usage_info_magnet_test.go` (create)

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces:
  - `type MagnetGrab struct` with `func (m MagnetGrab) ObjectId() uint32` and `func (m MagnetGrab) Grabbed() bool`
  - `func NewMagnetGrab(objectId uint32, grabbed bool) MagnetGrab`
  - `func (m *SkillUsageInfo) MagnetGrabs() []MagnetGrab`
  - `func (m *SkillUsageInfo) Direction() bool`
  - `func (b *SkillUsageInfoBuilder) SetMagnetGrabs(v []MagnetGrab) *SkillUsageInfoBuilder`
  - `func (b *SkillUsageInfoBuilder) SetDirection(v bool) *SkillUsageInfoBuilder`

Wire layouts (design §2, derived per version from the client binaries):

```
modern (gms_61…gms_95, jms_185)        legacy (gms_48 only)
  updateTime : uint32                     updateTime : uint32
  skillId    : uint32                     skillId    : uint32
  skillLevel : byte                       skillLevel : byte
  grabCount  : uint32                     entryCount : byte
  repeat grabCount:                       repeat entryCount:
    objectId : uint32                       objectId : uint32   (index 0 = CASTER)
    grabbed  : byte                       delay      : uint16
  direction  : byte
```

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-packet/model/skill_usage_info_magnet_test.go`:

```go
package model

import (
	"encoding/binary"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// decodeMagnetBody runs the shared decoder over buf under the given tenant
// version and returns the model plus the reader's unconsumed byte count. A
// non-zero remainder means the layout is wrong, which is the single most
// valuable assertion these tests make.
func decodeMagnetBody(t *testing.T, region string, major, minor uint16, buf []byte) (*SkillUsageInfo, int) {
	t.Helper()
	req := request.Request(buf)
	reader := request.NewRequestReader(&req, 0)
	m := &SkillUsageInfo{}
	m.Decode(nil, pt.CreateContext(region, major, minor))(&reader, nil)
	return m, reader.Available()
}

// modernMagnetBody builds a gms_61+/jms magnet body: uint32 grab count,
// (objectId uint32, grabbed byte) per entry, trailing direction byte, NO delay.
func modernMagnetBody(skillId skill.Id, level byte, grabs []MagnetGrab, left bool) []byte {
	buf := make([]byte, 0, 13+5*len(grabs))
	buf = binary.LittleEndian.AppendUint32(buf, 12345)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skillId))
	buf = append(buf, level)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(grabs)))
	for _, g := range grabs {
		buf = binary.LittleEndian.AppendUint32(buf, g.ObjectId())
		if g.Grabbed() {
			buf = append(buf, 1)
		} else {
			buf = append(buf, 0)
		}
	}
	if left {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}
	return buf
}

// legacyMagnetBody builds the gms_48 magnet body: byte entry count, uint32
// object ids with entry[0] = the CASTER's own object id, trailing delay short,
// NO per-entry result and NO direction byte.
func legacyMagnetBody(skillId skill.Id, level byte, casterObjectId uint32, mobIds []uint32, delay uint16) []byte {
	buf := make([]byte, 0, 12+4*(len(mobIds)+1))
	buf = binary.LittleEndian.AppendUint32(buf, 12345)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skillId))
	buf = append(buf, level)
	buf = append(buf, byte(len(mobIds)+1))
	buf = binary.LittleEndian.AppendUint32(buf, casterObjectId)
	for _, id := range mobIds {
		buf = binary.LittleEndian.AppendUint32(buf, id)
	}
	buf = binary.LittleEndian.AppendUint16(buf, delay)
	return buf
}

func TestDecodeMagnetModern_MixedGrabResults(t *testing.T) {
	want := []MagnetGrab{
		NewMagnetGrab(1001, true),
		NewMagnetGrab(1002, false),
		NewMagnetGrab(0, true), // released slot — id 0 is legitimate on the wire
	}
	m, left := decodeMagnetBody(t, "GMS", 83, 1,
		modernMagnetBody(skill.HeroMonsterMagnetId, 30, want, true))

	if left != 0 {
		t.Fatalf("reader has %d unconsumed bytes after decode", left)
	}
	got := m.MagnetGrabs()
	if len(got) != len(want) {
		t.Fatalf("MagnetGrabs len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ObjectId() != want[i].ObjectId() || got[i].Grabbed() != want[i].Grabbed() {
			t.Fatalf("MagnetGrabs[%d] = (%d,%v), want (%d,%v)",
				i, got[i].ObjectId(), got[i].Grabbed(), want[i].ObjectId(), want[i].Grabbed())
		}
	}
	if !m.Direction() {
		t.Fatal("Direction = false, want true (stance&1 == 1 means facing left)")
	}
	if m.Delay() != 0 {
		t.Fatalf("Delay = %d, want 0 — the modern magnet body carries no delay short", m.Delay())
	}
}

func TestDecodeMagnetModern_EmptyGrabTable(t *testing.T) {
	m, left := decodeMagnetBody(t, "GMS", 95, 1,
		modernMagnetBody(skill.PaladinMonsterMagnetId, 1, nil, false))

	if left != 0 {
		t.Fatalf("reader has %d unconsumed bytes after decode", left)
	}
	if len(m.MagnetGrabs()) != 0 {
		t.Fatalf("MagnetGrabs = %v, want empty", m.MagnetGrabs())
	}
	if m.Direction() {
		t.Fatal("Direction = true, want false")
	}
}

func TestDecodeMagnetModern_JmsTakesModernBranch(t *testing.T) {
	m, left := decodeMagnetBody(t, "JMS", 185, 1,
		modernMagnetBody(skill.DarkKnightMonsterMagnetId, 20,
			[]MagnetGrab{NewMagnetGrab(7, true)}, true))

	if left != 0 {
		t.Fatalf("jms_185 must take the modern branch; %d unconsumed bytes", left)
	}
	if len(m.MagnetGrabs()) != 1 || m.MagnetGrabs()[0].ObjectId() != 7 {
		t.Fatalf("MagnetGrabs = %v, want [(7,true)]", m.MagnetGrabs())
	}
}

func TestDecodeMagnetLegacy_DiscardsLeadingCasterEntry(t *testing.T) {
	m, left := decodeMagnetBody(t, "GMS", 48, 1,
		legacyMagnetBody(skill.HeroMonsterMagnetId, 15, 900001, []uint32{2001, 2002}, 750))

	if left != 0 {
		t.Fatalf("reader has %d unconsumed bytes after decode", left)
	}
	got := m.MagnetGrabs()
	if len(got) != 2 {
		t.Fatalf("MagnetGrabs len = %d, want 2 — entry[0] is the caster and must be discarded", len(got))
	}
	if got[0].ObjectId() != 2001 || got[1].ObjectId() != 2002 {
		t.Fatalf("MagnetGrabs ids = [%d %d], want [2001 2002]", got[0].ObjectId(), got[1].ObjectId())
	}
	for i, g := range got {
		if !g.Grabbed() {
			t.Fatalf("MagnetGrabs[%d].Grabbed = false; v48 sends no per-entry result, every surviving entry is an unconditional grab", i)
		}
	}
	if m.Delay() != 750 {
		t.Fatalf("Delay = %d, want 750", m.Delay())
	}
	if m.Direction() {
		t.Fatal("Direction = true; v48 sends no direction byte")
	}
}

func TestDecodeMagnetLegacy_CasterOnlyYieldsNoGrabs(t *testing.T) {
	m, left := decodeMagnetBody(t, "GMS", 48, 1,
		legacyMagnetBody(skill.PaladinMonsterMagnetId, 1, 900001, nil, 600))

	if left != 0 {
		t.Fatalf("reader has %d unconsumed bytes after decode", left)
	}
	if len(m.MagnetGrabs()) != 0 {
		t.Fatalf("MagnetGrabs = %v, want empty (only the caster entry was sent)", m.MagnetGrabs())
	}
	if m.Delay() != 600 {
		t.Fatalf("Delay = %d, want 600", m.Delay())
	}
}

// TestDecodeMagnetHostileCountDoesNotSpin pins the allocation/loop bound. The
// shared reader returns 0 WITHOUT advancing pos once exhausted
// (atlas-socket/request/reader.go), so an unbounded loop over a client-supplied
// uint32 count would spin ~4 billion times on the channel's packet goroutine.
func TestDecodeMagnetHostileCountDoesNotSpin(t *testing.T) {
	buf := make([]byte, 0, 17)
	buf = binary.LittleEndian.AppendUint32(buf, 1)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skill.HeroMonsterMagnetId))
	buf = append(buf, 30)
	buf = binary.LittleEndian.AppendUint32(buf, 0xFFFFFFFF) // hostile grabCount
	buf = binary.LittleEndian.AppendUint32(buf, 1001)       // one real entry
	buf = append(buf, 1)

	m, _ := decodeMagnetBody(t, "GMS", 83, 1, buf)
	if len(m.MagnetGrabs()) > 4 {
		t.Fatalf("MagnetGrabs len = %d; the loop must be bounded by the bytes actually available", len(m.MagnetGrabs()))
	}
}

// TestDecodeMagnetDoesNotDisturbOtherSkills is the FR-1.3 / backward-compat
// guard: a magnet id must not be reachable through the mob-affecting, party or
// anti-repeat lists, and the representative non-magnet arms must decode
// byte-identically with the magnet branch present.
func TestDecodeMagnetDoesNotDisturbOtherSkills(t *testing.T) {
	for _, id := range []skill.Id{
		skill.HeroMonsterMagnetId,
		skill.PaladinMonsterMagnetId,
		skill.DarkKnightMonsterMagnetId,
	} {
		if isMobAffectingBuff(id) {
			t.Fatalf("skill [%d] must not be in isMobAffectingBuff", id)
		}
		if isPartyBuff(id) {
			t.Fatalf("skill [%d] must not be in isPartyBuff", id)
		}
		if isAntiRepeatBuffSkill(id) {
			t.Fatalf("skill [%d] must not be in isAntiRepeatBuffSkill", id)
		}
	}

	// Shadow Stars: updateTime(4) skillId(4) slv(1) javelinItemId(4).
	buf := make([]byte, 0, 13)
	buf = binary.LittleEndian.AppendUint32(buf, 999)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skill.NightLordShadowStarsId))
	buf = append(buf, 30)
	buf = binary.LittleEndian.AppendUint32(buf, 2070006)
	m, left := decodeMagnetBody(t, "GMS", 83, 1, buf)
	if left != 0 || m.SpiritJavelinItemId() != 2070006 {
		t.Fatalf("Shadow Stars decode regressed: javelin=%d, %d bytes left", m.SpiritJavelinItemId(), left)
	}
	if len(m.MagnetGrabs()) != 0 {
		t.Fatalf("non-magnet cast populated MagnetGrabs = %v", m.MagnetGrabs())
	}

	// Bishop Resurrection: updateTime(4) skillId(4) slv(1) bitmap(1) delay(2).
	buf = buf[:0]
	buf = binary.LittleEndian.AppendUint32(buf, 999)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skill.BishopResurrectionId))
	buf = append(buf, 30)
	buf = append(buf, 0b101)
	buf = binary.LittleEndian.AppendUint16(buf, 400)
	m, left = decodeMagnetBody(t, "GMS", 83, 1, buf)
	if left != 0 || m.AffectedPartyMemberBitmap() != 0b101 || m.Delay() != 400 {
		t.Fatalf("Resurrection decode regressed: bitmap=%#b delay=%d, %d bytes left",
			m.AffectedPartyMemberBitmap(), m.Delay(), left)
	}
}

func TestMagnetBuilderSetters(t *testing.T) {
	grabs := []MagnetGrab{NewMagnetGrab(42, true)}
	m := NewSkillUsageInfoBuilder().
		SetSkillId(uint32(skill.HeroMonsterMagnetId)).
		SetMagnetGrabs(grabs).
		SetDirection(true).
		Build()

	if len(m.MagnetGrabs()) != 1 || m.MagnetGrabs()[0].ObjectId() != 42 {
		t.Fatalf("MagnetGrabs = %v, want [(42,true)]", m.MagnetGrabs())
	}
	if !m.Direction() {
		t.Fatal("Direction = false, want true")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd "$(git rev-parse --show-toplevel)/libs/atlas-packet" && go test ./model/ -run Magnet -v
```

Expected: FAIL to compile — `undefined: MagnetGrab`, `undefined: NewMagnetGrab`, `m.MagnetGrabs undefined`, `m.Direction undefined`, `SetMagnetGrabs`/`SetDirection` undefined.

- [ ] **Step 3: Add the `MagnetGrab` value type and the new fields**

In `libs/atlas-packet/model/skill_usage_info.go`, extend the struct (add the two fields at the end of the existing field list):

```go
type SkillUsageInfo struct {
	updateTime                uint32
	skillId                   uint32
	skillLevel                byte
	castX                     int16
	castY                     int16
	spiritJavelinItemId       uint32
	affectedPartyMemberBitmap uint8
	affectedMobIds            []uint32
	delay                     uint16
	magnetGrabs               []MagnetGrab
	direction                 bool
}

// MagnetGrab is one entry of the Monster Magnet grab table: the CMob object
// id the client picked up and whether it reports the grab as successful.
//
// Immutable value type (FR-1.6). objectId 0 is a LEGITIMATE wire value, not a
// sentinel: the client's encode loop walks its whole candidate array, and
// slots whose CanGoThrough/CanWalkThrough probe failed were released earlier in
// the same function, leaving a null ZRef whose id reads 0 (gms_83
// CUserLocal::TryDoingMonsterMagnet @0x96C215). Dropping such entries is the
// server-side validator's job, not the decoder's.
type MagnetGrab struct {
	objectId uint32
	grabbed  bool
}

func NewMagnetGrab(objectId uint32, grabbed bool) MagnetGrab {
	return MagnetGrab{objectId: objectId, grabbed: grabbed}
}

func (m MagnetGrab) ObjectId() uint32 { return m.objectId }

func (m MagnetGrab) Grabbed() bool { return m.grabbed }
```

Add the getters next to the existing ones (after `Delay()`):

```go
// MagnetGrabs returns the Monster Magnet grab table. Empty for every other
// skill. On gms_48 the client's leading caster entry has already been
// discarded and every remaining entry is marked grabbed (that version sends no
// per-entry result).
func (m *SkillUsageInfo) MagnetGrabs() []MagnetGrab {
	return m.magnetGrabs
}

// Direction is the caster's facing bit (CUserLocal.stance & 1; true = facing
// left) as sent on the Monster Magnet body. Always false on gms_48, which
// sends no direction byte, and for every non-magnet skill.
func (m *SkillUsageInfo) Direction() bool {
	return m.direction
}
```

Add the builder setters next to the existing ones (after `SetDelay`):

```go
func (b *SkillUsageInfoBuilder) SetMagnetGrabs(v []MagnetGrab) *SkillUsageInfoBuilder {
	b.info.magnetGrabs = v
	return b
}

func (b *SkillUsageInfoBuilder) SetDirection(v bool) *SkillUsageInfoBuilder {
	b.info.direction = v
	return b
}
```

- [ ] **Step 4: Add the identity predicate, the version gate, and the decode branch**

Add the import `"github.com/Chronicle20/atlas/libs/atlas-constants/constants"` to the file's import block. (`constants` imports only `skill` and `job`; no cycle.)

Append to the bottom of `skill_usage_info.go`:

```go
// magnetEntrySizeModern is the on-wire size of one gms_61+/jms grab entry:
// objectId uint32 + grabbed byte. magnetEntrySizeLegacy is the gms_48 entry:
// objectId uint32, no result byte.
const (
	magnetEntrySizeModern = 5
	magnetEntrySizeLegacy = 4
)

// isMonsterMagnet resolves the incoming wire id through the tenant's version
// set and reports whether it is one of the three Monster Magnet identities
// (FR-1.1). Identity-keyed rather than a raw skill.Is compare against
// 1121001/1221001/1321001 (task-187): the wire id happens to be stable across
// every provisioned version for these three, but the resolver is the contract
// this file's remaining raw-id lists exist to be migrated onto, and a raw
// compare here would be the wrong precedent for the next branch added.
func isMonsterMagnet(t tenant.Model, skillId uint32) bool {
	set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
	id, ok := set.Skill.Resolve(skill.Id(skillId))
	if !ok {
		return false
	}
	return skill.IsIdentity(id,
		skill.HeroMonsterMagnet,
		skill.PaladinMonsterMagnet,
		skill.DarkKnightMonsterMagnet,
	)
}

// legacyMagnetLayout reports whether this tenant's client sends the gms_48
// magnet body (byte count, no per-entry result, trailing delay short, no
// direction byte, leading caster-id entry) rather than the modern one.
//
// The split is gms_48 vs EVERYTHING else — deliberately NARROWER than the
// isAntiRepeatBuffSkill gate above, which splits at gms_72. Do not harmonise
// the two. Verified by decompiling CUserLocal::TryDoingMonsterMagnet per
// version: gms_48 @0x6AD842 (COutPacket ctor `push 46h` @0x6ADABC; entryCount
// Encode1 @0x6ADB02; per-entry Encode4 @0x6ADB1B; delay Encode2 @0x6ADB29; the
// caster's own object id is inserted at index 0 by ZArray<ulong>::InsertBefore
// @0x6AD977-0x6AD987 BEFORE the mob loop @0x6ADA89-0x6ADA99, both reading
// offset +0x654), versus the modern shape at gms_61 @0x7B9684, gms_72
// @0x876605, gms_79 @0x8C3117, gms_83 @0x96C215, gms_84 @0x9ABDB7, gms_87
// @0x9F086F, gms_92 @0x91F2A0, gms_95 @0x940570 and jms_185 @0xA3C61C — all of
// which Encode4 the grab count, Encode4/Encode1 per entry, and Encode1 the
// direction with NO trailing delay. jms takes the modern branch.
func legacyMagnetLayout(t tenant.Model) bool {
	return t.IsRegion("GMS") && !t.MajorAtLeast(61)
}

// decodeMagnet consumes the Monster Magnet body. It is a REPLACEMENT body, not
// an additive suffix of the common prefix the other arms share, which is why
// Decode returns immediately after calling it.
//
// The per-entry loops are bounded by the bytes actually available as well as by
// the client-supplied count: the shared reader returns zero WITHOUT advancing
// pos once exhausted (atlas-socket/request/reader.go), so an unbounded loop
// over a hostile 0xFFFFFFFF count would spin ~4 billion times on the channel's
// packet-handling goroutine producing nothing.
func (m *SkillUsageInfo) decodeMagnet(r *request.Reader, legacy bool) {
	if legacy {
		entryCount := int(r.ReadByte())
		if maxEntries := r.Available() / magnetEntrySizeLegacy; entryCount > maxEntries {
			entryCount = maxEntries
		}
		m.magnetGrabs = make([]MagnetGrab, 0, entryCount)
		for i := range entryCount {
			objectId := r.ReadUint32()
			// entry[0] is the CASTER's own object id, not a monster.
			if i == 0 {
				continue
			}
			m.magnetGrabs = append(m.magnetGrabs, NewMagnetGrab(objectId, true))
		}
		m.delay = r.ReadUint16()
		return
	}

	grabCount := int(r.ReadUint32())
	if maxEntries := r.Available() / magnetEntrySizeModern; grabCount > maxEntries {
		grabCount = maxEntries
	}
	m.magnetGrabs = make([]MagnetGrab, 0, grabCount)
	for range grabCount {
		objectId := r.ReadUint32()
		// gms_83 @0x96C215 encodes `COutPacket::Encode1(v65, *v40 == 3)` — a
		// BOOL, not an enum. The 3 is the client's own prop-roll sentinel.
		grabbed := r.ReadByte() != 0
		m.magnetGrabs = append(m.magnetGrabs, NewMagnetGrab(objectId, grabbed))
	}
	m.direction = r.ReadByte() != 0
}
```

Insert the branch in `Decode`, immediately after `m.skillLevel = r.ReadByte()` and **before** the `isAntiRepeatBuffSkill` gate:

```go
		m.skillLevel = r.ReadByte()
		// Monster Magnet is delivered on this same opcode but its body is
		// written by a DIFFERENT client function (CUserLocal::TryDoingMonsterMagnet)
		// than every other skill here (CUserLocal::SendSkillUseRequest), and it
		// diverges immediately after skillLevel. Consume it and return: the
		// magnet shares no suffix with the arms below, and an early return makes
		// the mutual exclusion structural rather than list-maintained (FR-1.3).
		if isMonsterMagnet(t, m.skillId) {
			m.decodeMagnet(r, legacyMagnetLayout(t))
			return
		}
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd "$(git rev-parse --show-toplevel)/libs/atlas-packet" && go test ./model/ -run Magnet -v
```

Expected: PASS, all eight tests.

- [ ] **Step 6: Run the full module suite for regressions**

```bash
cd "$(git rev-parse --show-toplevel)/libs/atlas-packet" && go test -race ./... && go vet ./...
```

Expected: PASS. The pre-existing `skill_usage_info_test.go` cases (Dispel on gms_61, etc.) must be unaffected.

- [ ] **Step 7: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add libs/atlas-packet/model/skill_usage_info.go libs/atlas-packet/model/skill_usage_info_magnet_test.go
git commit -m "feat(task-215): decode the Monster Magnet arm of the use-skill opcode

Two wire shapes derived per version from the client binaries: gms_48 sends a
byte count with a leading caster entry and a trailing delay short; gms_61+ and
jms send a uint32 count, a per-entry grab bool, and a trailing direction byte.
The branch is first and returns, so no other skill's decode changes."
```

---

### Task 2: Expose the WZ `range` attribute on the channel effect model

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/effect/rest.go`
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go`
- Test: `services/atlas-channel/atlas.com/channel/data/skill/effect/rest_range_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: `func (m Model) Range() int32` on `atlas-channel/data/skill/effect`.

Monster Magnet carries no `lt`/`rb` in WZ (design §3 — verified against the local extracted `Skill.wz/{112,122,132}.img.xml`), so `hasEffectBbox` returns false for it and the existing rect path can never fire. It carries `range` (200 at level 1 → 450 at level 30), which atlas-data already serves (`services/atlas-data/atlas.com/data/skill/effect/rest.go:78`) and atlas-channel currently discards.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/data/skill/effect/rest_range_test.go`:

```go
package effect

import "testing"

// TestExtractRange pins that the channel decodes the WZ `range` attribute
// atlas-data already serves. Monster Magnet has no lt/rb, so `range` is the
// only WZ input to its server-side target region (task-215 design §3).
func TestExtractRange(t *testing.T) {
	m, err := Extract(RestModel{Range: 450, MobCount: 7})
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if m.Range() != 450 {
		t.Fatalf("Range = %d, want 450", m.Range())
	}
	if m.MobCount() != 7 {
		t.Fatalf("MobCount = %d, want 7", m.MobCount())
	}
}

func TestExtractRangeAbsentIsZero(t *testing.T) {
	m, err := Extract(RestModel{})
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if m.Range() != 0 {
		t.Fatalf("Range = %d, want 0 when the attribute is absent", m.Range())
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-channel/atlas.com/channel" && go test ./data/skill/effect/ -run Range -v
```

Expected: FAIL to compile — `unknown field Range in struct literal of type RestModel`.

- [ ] **Step 3: Add the field, the mapping, and the getter**

In `rest.go`, add the field alongside `MobCount` (keep the existing json-tag style and alignment):

```go
	Range             int32   `json:"range"`
```

In the same file's `Extract` (the struct literal that already assigns `mobCount: rm.MobCount`), add:

```go
		rangeValue:           rm.Range,
```

In `model.go`, add the private field next to `mobCount`:

```go
	rangeValue           int32
```

and the getter next to `MobCount()`:

```go
// Range returns the skill's WZ `range` attribute in map pixels. For Monster
// Magnet (which carries no lt/rb) this is the only WZ input to the server-side
// target region: the client selects candidates through
// CMobPool::CheckMobInTrapezoid out to this distance. Zero means the attribute
// is absent, in which case range-based selection must fall back to cap-only.
func (m Model) Range() int32 {
	return m.rangeValue
}
```

> The private field is `rangeValue`, not `range` — `range` is a Go keyword.

- [ ] **Step 4: Run the test to verify it passes**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-channel/atlas.com/channel" && go test ./data/skill/effect/ -run Range -v
```

Expected: PASS, both tests.

- [ ] **Step 5: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add services/atlas-channel/atlas.com/channel/data/skill/effect/
git commit -m "feat(task-215): decode the WZ range attribute on the channel effect model

atlas-data already serves it; the channel discarded it. Monster Magnet carries
no lt/rb, so range is the only WZ input to its server-side target region."
```

---

### Task 3: Monster command contract, providers, and channel emitters

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/processor.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/mock/processor.go`
- Test: `services/atlas-channel/atlas.com/channel/monster/producer_magnet_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces:
  - Contract (both copies): type strings `"CLEAR_AGGRO"` and `"FORCE_CONTROL"`; body json shapes `{}` and `{"characterId": <uint32>}`.
  - atlas-channel: `monster.ClearAggroCommandProvider(f field.Model, monsterId uint32) model.Provider[[]kafka.Message]`, `monster.ForceControlCommandProvider(f field.Model, monsterId, characterId uint32) model.Provider[[]kafka.Message]`, and on `monster.Processor`: `ClearAggro(f field.Model, monsterId uint32) error`, `ForceControl(f field.Model, monsterId, characterId uint32) error`.

**Both contract copies are edited in this one task and land in one commit.** They are separate Go modules with no mirror guard; a divergent `Type` string or json tag fails no build and silently decodes into a zero-valued body.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/monster/producer_magnet_test.go`:

```go
package monster

import (
	monster2 "atlas-channel/kafka/message/monster"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func magnetTestField() field.Model {
	return field.NewBuilder(1, 2, 100000000).SetInstance(uuid.Nil).Build()
}

// TestClearAggroCommandProviderShape pins the envelope and the deliberately
// empty body (FR-4.3). Every handler on COMMAND_TOPIC_MONSTER unmarshals every
// message into its own body type, so an empty body cannot collide with a
// sibling's field types.
func TestClearAggroCommandProviderShape(t *testing.T) {
	msgs, err := ClearAggroCommandProvider(magnetTestField(), 4242)()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("provider produced %d messages, want 1", len(msgs))
	}

	var c monster2.Command[monster2.ClearAggroCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Type != monster2.CommandTypeClearAggro {
		t.Fatalf("Type = %q, want %q", c.Type, monster2.CommandTypeClearAggro)
	}
	if c.MonsterId != 4242 {
		t.Fatalf("MonsterId = %d, want 4242", c.MonsterId)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(msgs[0].Value, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if string(raw["body"]) != "{}" {
		t.Fatalf("body = %s, want {} — the clear-aggro body must carry no fields", raw["body"])
	}
}

func TestForceControlCommandProviderShape(t *testing.T) {
	msgs, err := ForceControlCommandProvider(magnetTestField(), 4242, 777)()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("provider produced %d messages, want 1", len(msgs))
	}

	var c monster2.Command[monster2.ForceControlCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Type != monster2.CommandTypeForceControl {
		t.Fatalf("Type = %q, want %q", c.Type, monster2.CommandTypeForceControl)
	}
	if c.MonsterId != 4242 {
		t.Fatalf("MonsterId = %d, want 4242", c.MonsterId)
	}
	if c.Body.CharacterId != 777 {
		t.Fatalf("Body.CharacterId = %d, want 777", c.Body.CharacterId)
	}
}

// TestMonsterCommandsShareMonsterKey pins the ordering contract: both commands
// key on the monster id, so CLEAR_AGGRO then FORCE_CONTROL for the same monster
// land on the same partition in emit order. Reversing them would have the wipe
// immediately clear the aggro flag the handover just set.
func TestMonsterCommandsShareMonsterKey(t *testing.T) {
	clear, err := ClearAggroCommandProvider(magnetTestField(), 4242)()
	if err != nil {
		t.Fatalf("clear provider returned error: %v", err)
	}
	force, err := ForceControlCommandProvider(magnetTestField(), 4242, 777)()
	if err != nil {
		t.Fatalf("force provider returned error: %v", err)
	}
	if string(clear[0].Key) != string(force[0].Key) {
		t.Fatalf("keys differ (%x vs %x); both commands must key on the monster id",
			clear[0].Key, force[0].Key)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-channel/atlas.com/channel" && \
  go test ./monster/ -run 'ClearAggro|ForceControl|MonsterCommandsShare' -v
```

Expected: FAIL to compile — `undefined: ClearAggroCommandProvider`, `undefined: monster2.CommandTypeClearAggro`, etc.

- [ ] **Step 3: Add the producer-side contract**

In `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`, extend the const block:

```go
	CommandTypeKill           = "KILL"
	CommandTypeClearAggro     = "CLEAR_AGGRO"
	CommandTypeForceControl   = "FORCE_CONTROL"
```

and add the two bodies after `KillCommandBody`:

```go
// ClearAggroCommandBody asks atlas-monsters to fully wipe a monster's
// accumulated damage-aggro table — every character's entry, not a decay toward
// the aggro floor. Deliberately EMPTY (FR-4.3): the command is orthogonal and
// carries nothing magnet-specific, and an empty body cannot collide with a
// sibling body's field types on this shared, fan-to-every-handler topic.
type ClearAggroCommandBody struct{}

// ForceControlCommandBody asks atlas-monsters to hand a monster's controller to
// a named character, bypassing the normal picker election, and to set the
// controller-has-aggro flag so the resulting START_CONTROL drives
// writer.StartControlMonsterBody(m, true).
//
// characterId is the only field, and `characterId uint32` already appears with
// that exact name and type in DamageCommandBody, KillCommandBody and the
// catch body, so it introduces no unmarshal collision on the shared topic.
type ForceControlCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}
```

- [ ] **Step 4: Add the matching consumer-side contract (same commit)**

In `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`, extend the const block (note this file's Go names are unexported by convention, but the **string values must match the producer's exactly**):

```go
	CommandTypeCatch             = "CATCH"
	CommandTypeClearAggro        = "CLEAR_AGGRO"
	CommandTypeForceControl      = "FORCE_CONTROL"
```

and add the two bodies after `catchCommandBody`:

```go
// clearAggroCommandBody asks the processor to fully wipe a monster's
// damage-aggro table. Deliberately empty: the command carries nothing
// caller-specific, and an empty body cannot collide with a sibling body's field
// types on this shared, fan-to-every-handler topic. Mirrors
// atlas-channel's monster2.ClearAggroCommandBody — edit both together.
type clearAggroCommandBody struct{}

// forceControlCommandBody asks the processor to hand control of a monster to a
// named character with the aggro flag set, bypassing the picker. Mirrors
// atlas-channel's monster2.ForceControlCommandBody — edit both together.
type forceControlCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}
```

- [ ] **Step 5: Add the two providers**

In `services/atlas-channel/atlas.com/channel/monster/producer.go`, append:

```go
// ClearAggroCommandProvider asks atlas-monsters to fully wipe the monster's
// damage-aggro table. Keyed on the monster id like every other monster command,
// so it is ordered against ForceControlCommandProvider for the same monster.
func ClearAggroCommandProvider(f field.Model, monsterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &monster2.Command[monster2.ClearAggroCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      monster2.CommandTypeClearAggro,
		Body:      monster2.ClearAggroCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

// ForceControlCommandProvider asks atlas-monsters to hand the monster's
// controller to characterId with the aggro flag set.
func ForceControlCommandProvider(f field.Model, monsterId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &monster2.Command[monster2.ForceControlCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      monster2.CommandTypeForceControl,
		Body: monster2.ForceControlCommandBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 6: Add the two processor methods**

In `services/atlas-channel/atlas.com/channel/monster/processor.go`, add to the `Processor` interface after `Kill`:

```go
	ClearAggro(f field.Model, monsterId uint32) error
	ForceControl(f field.Model, monsterId uint32, characterId uint32) error
```

and the implementations at the end of the file:

```go
// ClearAggro asks atlas-monsters to fully wipe the monster's damage-aggro
// table. Orthogonal to ForceControl — either may be issued without the other.
func (p *ProcessorImpl) ClearAggro(f field.Model, monsterId uint32) error {
	p.l.Debugf("Clearing aggro for monster [%d].", monsterId)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(ClearAggroCommandProvider(f, monsterId))
}

// ForceControl asks atlas-monsters to hand control of the monster to
// characterId with the aggro flag set, bypassing the picker election.
func (p *ProcessorImpl) ForceControl(f field.Model, monsterId uint32, characterId uint32) error {
	p.l.Debugf("Forcing control of monster [%d] to character [%d].", monsterId, characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(ForceControlCommandProvider(f, monsterId, characterId))
}
```

- [ ] **Step 7: Extend the mock processor**

In `services/atlas-channel/atlas.com/channel/monster/mock/processor.go`, add two fields to the `ProcessorMock` struct after `KillFunc` (`:25`), keeping the file's column alignment:

```go
	ClearAggroFunc             func(f field.Model, monsterId uint32) error
	ForceControlFunc           func(f field.Model, monsterId uint32, characterId uint32) error
```

and two methods after `Kill` (`:128`), matching its shape exactly:

```go
func (m *ProcessorMock) ClearAggro(f field.Model, monsterId uint32) error {
	if m.ClearAggroFunc != nil {
		return m.ClearAggroFunc(f, monsterId)
	}
	return nil
}

func (m *ProcessorMock) ForceControl(f field.Model, monsterId uint32, characterId uint32) error {
	if m.ForceControlFunc != nil {
		return m.ForceControlFunc(f, monsterId, characterId)
	}
	return nil
}
```

- [ ] **Step 8: Run the tests to verify they pass**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-channel/atlas.com/channel" && go test ./monster/... -v && go build ./...
cd "$(git rev-parse --show-toplevel)/services/atlas-monsters/atlas.com/monsters" && go build ./...
```

Expected: PASS and both builds clean.

- [ ] **Step 9: Verify the two contract copies agree**

```bash
cd "$(git rev-parse --show-toplevel)"
grep -n 'CLEAR_AGGRO\|FORCE_CONTROL' \
  services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go \
  services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go
grep -n 'json:"characterId"' \
  services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go \
  services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go
```

Expected: `"CLEAR_AGGRO"` and `"FORCE_CONTROL"` appear once in each file with identical string literals, and the two force-control bodies both carry `json:"characterId"`.

- [ ] **Step 10: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go \
        services/atlas-channel/atlas.com/channel/monster/ \
        services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go
git commit -m "feat(task-215): add CLEAR_AGGRO and FORCE_CONTROL monster commands

Two orthogonal command types on COMMAND_TOPIC_MONSTER with the producer and
consumer contract copies edited together. Bodies are minimal (empty, and a lone
characterId that already exists with that name and type in sibling bodies)
because every handler on this topic unmarshals every message."
```

---

### Task 4: atlas-monsters aggro wipe and forced control

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/model.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/registry.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go`
- Test: `services/atlas-monsters/atlas.com/monsters/monster/clear_aggro_test.go` (create)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/force_control_test.go` (create)

**Interfaces:**
- Consumes: `CommandTypeClearAggro`, `CommandTypeForceControl`, `clearAggroCommandBody`, `forceControlCommandBody` from Task 3.
- Produces:
  - `func (m Model) ControlWithAggro(characterId uint32) Model`
  - `type ClearSummary struct { Monster Model; ControllerCharacterId uint32; AggroFlippedOff bool }`
  - `func (r *Registry) ClearDamageEntries(t tenant.Model, uniqueId uint32) (ClearSummary, error)`
  - `func (r *Registry) ControlMonsterWithAggro(tenant tenant.Model, uniqueId uint32, characterId uint32) (Model, error)`
  - On `Processor`: `ClearAggro(uniqueId uint32) error`, `ForceControl(uniqueId uint32, characterId uint32) error`

- [ ] **Step 1: Write the failing tests**

The harness below is the one this package already uses: `newTestTenant(t)` (`cooldown_test.go:28`), `testField()` (`model_test.go:15`), and `recordingProcessor(ctx, tm, &emitted)` (`control_assignment_test.go:17`), which builds a `ProcessorImpl` whose `emit` hook counts `EnvEventTopicMonsterStatus` emissions. Note that `recordingProcessor` leaves `inFieldFn` and `hiddenFn` nil, so the force-control tests set them explicitly — `ForceControl` calls both.

Create `services/atlas-monsters/atlas.com/monsters/monster/clear_aggro_test.go`:

```go
package monster

import (
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// newAggroedMonster stands up a monster in the registry, applies damage from
// each of the given characters (which flips controllerHasAggro true once the
// monster is controlled), and returns its unique id.
func newAggroedMonster(t *testing.T, ctx context.Context, tm tenant.Model, controllerId uint32, attackers []uint32) uint32 {
	t.Helper()
	r := GetMonsterRegistry()
	r.CreateMonster(ctx, tm, testField(), 9000000, 0, 0, 0, 0, 0, 100000, 50)
	mons := r.GetMonstersInMap(tm, testField())
	if len(mons) != 1 {
		t.Fatalf("expected 1 monster; got %d", len(mons))
	}
	uid := mons[0].UniqueId()

	if controllerId != 0 {
		if _, err := r.ControlMonster(tm, uid, controllerId); err != nil {
			t.Fatalf("ControlMonster: %v", err)
		}
	}
	for i, a := range attackers {
		if _, err := r.ApplyDamage(tm, a, uint32(100*(i+1)), uid, int64(1_000+i)); err != nil {
			t.Fatalf("ApplyDamage(%d): %v", a, err)
		}
	}
	return uid
}

func TestClearAggro_WipesEveryEntryAndFlipsFlag(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	const controller = uint32(7)
	uid := newAggroedMonster(t, ctx, tm, controller, []uint32{7, 8, 9})

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)

	before, err := p.GetById(uid)
	if err != nil {
		t.Fatalf("GetById before: %v", err)
	}
	if len(before.DamageEntries()) != 3 {
		t.Fatalf("setup: expected 3 damage entries, got %d", len(before.DamageEntries()))
	}
	if !before.ControllerHasAggro() {
		t.Fatal("setup: expected controllerHasAggro true after damage from the controller")
	}

	if err := p.ClearAggro(uid); err != nil {
		t.Fatalf("ClearAggro: %v", err)
	}

	after, err := p.GetById(uid)
	if err != nil {
		t.Fatalf("GetById after: %v", err)
	}
	if len(after.DamageEntries()) != 0 {
		t.Fatalf("DamageEntries = %d, want 0 — the wipe must remove EVERY character's entry, not just the caster's", len(after.DamageEntries()))
	}
	if after.ControllerHasAggro() {
		t.Fatal("ControllerHasAggro = true, want false after a wipe")
	}
	// Losing aggro is not losing control — DecayDamageEntries behaves the same.
	if after.ControlCharacterId() != controller {
		t.Fatalf("ControlCharacterId = %d, want %d; the wipe must not clear the controller",
			after.ControlCharacterId(), controller)
	}
	if emitted != 1 {
		t.Fatalf("emitted %d MONSTER_STATUS events, want exactly 1 (the AGGRO_CHANGED flip)", emitted)
	}
}

func TestClearAggro_EmptyTableIsANoOp(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	uid := newAggroedMonster(t, ctx, tm, 0, nil)

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)

	if err := p.ClearAggro(uid); err != nil {
		t.Fatalf("first ClearAggro: %v", err)
	}
	if err := p.ClearAggro(uid); err != nil {
		t.Fatalf("second ClearAggro must be idempotent: %v", err)
	}
	if emitted != 0 {
		t.Fatalf("emitted %d events, want 0 — wiping an already-empty table must emit nothing", emitted)
	}
}

func TestClearAggro_MissingMonsterIsDropped(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)

	if err := p.ClearAggro(4242); err != nil {
		t.Fatalf("ClearAggro on a nonexistent monster must be dropped, not an error: %v", err)
	}
	if emitted != 0 {
		t.Fatalf("emitted %d events, want 0", emitted)
	}
}

// TestClearAggro_DecaySweepSeesNothingToDo is the FR-4.4 interaction check: the
// wipe converges on the same state DecayDamageEntries reaches when its list
// empties, so the next sweep tick over the same monster is a no-op.
func TestClearAggro_DecaySweepSeesNothingToDo(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	uid := newAggroedMonster(t, ctx, tm, 7, []uint32{7, 8})

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)
	if err := p.ClearAggro(uid); err != nil {
		t.Fatalf("ClearAggro: %v", err)
	}

	summary, err := GetMonsterRegistry().DecayDamageEntries(tm, uid, 1_000_000)
	if err != nil {
		t.Fatalf("DecayDamageEntries after wipe: %v", err)
	}
	if summary.AggroFlippedOff {
		t.Fatal("the decay sweep flipped the aggro flag again after a wipe; the wipe must already have converged")
	}
	if len(summary.Monster.DamageEntries()) != 0 {
		t.Fatalf("decay found %d entries after a wipe, want 0", len(summary.Monster.DamageEntries()))
	}
}
```

Create `services/atlas-monsters/atlas.com/monsters/monster/force_control_test.go`:

```go
package monster

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// forceControlProcessor is recordingProcessor plus the two field/hidden seams
// ForceControl consults. recordingProcessor leaves them nil.
func forceControlProcessor(ctx context.Context, tm tenant.Model, emitted *int, inField []uint32, hidden map[uint32]struct{}) *ProcessorImpl {
	p := recordingProcessor(ctx, tm, emitted)
	p.inFieldFn = func(_ field.Model) ([]uint32, error) { return inField, nil }
	p.hiddenFn = func() (map[uint32]struct{}, error) { return hidden, nil }
	return p
}

func TestForceControl_HandsOverWithAggroFlagSet(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	const (
		previous = uint32(7)
		caster   = uint32(9)
	)
	uid := newAggroedMonster(t, ctx, tm, previous, nil)

	emitted := 0
	p := forceControlProcessor(ctx, tm, &emitted, []uint32{previous, caster}, map[uint32]struct{}{})

	if err := p.ForceControl(uid, caster); err != nil {
		t.Fatalf("ForceControl: %v", err)
	}

	got, err := p.GetById(uid)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.ControlCharacterId() != caster {
		t.Fatalf("ControlCharacterId = %d, want %d", got.ControlCharacterId(), caster)
	}
	if !got.ControllerHasAggro() {
		t.Fatal("ControllerHasAggro = false; the handover must set the flag so START_CONTROL writes StartControlMonsterBody(m, true)")
	}
	// STOP_CONTROL (previous controller) + START_CONTROL (new controller).
	if emitted < 2 {
		t.Fatalf("emitted %d events, want at least 2 (stop then start)", emitted)
	}
}

func TestForceControl_UncontrolledMonsterEmitsStartOnly(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	const caster = uint32(9)
	uid := newAggroedMonster(t, ctx, tm, 0, nil)

	emitted := 0
	p := forceControlProcessor(ctx, tm, &emitted, []uint32{caster}, map[uint32]struct{}{})

	if err := p.ForceControl(uid, caster); err != nil {
		t.Fatalf("ForceControl: %v", err)
	}
	got, err := p.GetById(uid)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.ControlCharacterId() != caster || !got.ControllerHasAggro() {
		t.Fatalf("controller = %d hasAggro = %v, want %d/true", got.ControlCharacterId(), got.ControllerHasAggro(), caster)
	}
}

func TestForceControl_SameControllerIsANoOp(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	const caster = uint32(9)
	uid := newAggroedMonster(t, ctx, tm, caster, nil)

	emitted := 0
	p := forceControlProcessor(ctx, tm, &emitted, []uint32{caster}, map[uint32]struct{}{})

	if err := p.ForceControl(uid, caster); err != nil {
		t.Fatalf("ForceControl: %v", err)
	}
	if emitted != 0 {
		t.Fatalf("emitted %d events, want 0 — forcing control to the current controller must not emit a redundant control packet (FR-5.4)", emitted)
	}
}

func TestForceControl_CharacterNotInFieldIsDropped(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	uid := newAggroedMonster(t, ctx, tm, 7, nil)

	emitted := 0
	p := forceControlProcessor(ctx, tm, &emitted, []uint32{7}, map[uint32]struct{}{})

	if err := p.ForceControl(uid, 9); err != nil {
		t.Fatalf("ForceControl for an absent character must be dropped, not an error: %v", err)
	}
	if emitted != 0 {
		t.Fatalf("emitted %d events, want 0", emitted)
	}
}

func TestForceControl_HiddenCharacterIsDropped(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	uid := newAggroedMonster(t, ctx, tm, 7, nil)

	emitted := 0
	p := forceControlProcessor(ctx, tm, &emitted, []uint32{7, 9}, map[uint32]struct{}{9: {}})

	if err := p.ForceControl(uid, 9); err != nil {
		t.Fatalf("ForceControl for a GM-hidden character must be dropped: %v", err)
	}
	if emitted != 0 {
		t.Fatalf("emitted %d events, want 0 — RelinquishControlOnHide would immediately strip it back, producing a flap", emitted)
	}
}

func TestForceControl_MissingMonsterIsDropped(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	emitted := 0
	p := forceControlProcessor(ctx, tm, &emitted, []uint32{9}, map[uint32]struct{}{})

	if err := p.ForceControl(4242, 9); err != nil {
		t.Fatalf("ForceControl on a nonexistent monster must be dropped: %v", err)
	}
	if emitted != 0 {
		t.Fatalf("emitted %d events, want 0", emitted)
	}
}

// TestStartControlStillDefaultsAggroOff guards the forceAggro split: the
// existing StartControl path must be byte-for-byte unchanged in behaviour.
func TestStartControlStillDefaultsAggroOff(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	uid := newAggroedMonster(t, ctx, tm, 0, nil)

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)

	m, err := p.StartControl(uid, 9)
	if err != nil {
		t.Fatalf("StartControl: %v", err)
	}
	if m.ControllerHasAggro() {
		t.Fatal("StartControl set the aggro flag; only ForceControl may do that")
	}
	if m.ControlCharacterId() != 9 {
		t.Fatalf("ControlCharacterId = %d, want 9", m.ControlCharacterId())
	}
}
```

> If any helper signature above does not match the current source (`recordingProcessor`, `newTestTenant`, `testField`, `ApplyDamage`, `Clear`), use the real one — do not add a `*_testhelpers.go` file and do not change the production signature to fit the test.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-monsters/atlas.com/monsters" && \
  go test ./monster/ -run 'ClearAggro|ForceControl' -v
```

Expected: FAIL to compile — `ClearAggro`, `ForceControl`, `ClearDamageEntries`, `ControlMonsterWithAggro`, `ControlWithAggro` all undefined.

- [ ] **Step 3: Add `Model.ControlWithAggro`**

In `monster/model.go`, next to `Control`:

```go
// ControlWithAggro assigns the controller AND marks the controller as holding
// aggro in one value transition. Control() sets only controlCharacterId, which
// would leave the flag false after a CLEAR_AGGRO wipe and make the resulting
// START_CONTROL write StartControlMonsterBody(m, false).
func (m Model) ControlWithAggro(characterId uint32) Model {
	return Clone(m).
		SetControlCharacterId(characterId).
		SetControllerHasAggro(true).
		Build()
}
```

- [ ] **Step 4: Add the registry operations**

In `monster/registry.go`, next to `DecaySummary` / `DecayDamageEntries`:

```go
// ClearSummary is returned by ClearDamageEntries. It mirrors DecaySummary
// field-for-field so callers of both converge on the same emit decision.
type ClearSummary struct {
	Monster               Model
	ControllerCharacterId uint32
	AggroFlippedOff       bool
}

// ClearDamageEntries atomically wipes EVERY damage entry on the monster and
// flips controllerHasAggro false when the monster had aggro. This is a full
// wipe, not a decay toward AggroDecayFloor (FR-4.2).
//
// It deliberately converges on the same state DecayDamageEntries reaches when
// its entry list empties, which is what makes the interaction with the decay
// sweep and the controller picker correct (FR-4.4): the sweep's next tick sees
// an empty list and does nothing, and the picker's ControllerHasAggro gate
// behaves identically to a naturally-decayed monster. The controller itself is
// NOT cleared — losing aggro is not losing control.
//
// Written against storedMonster via reg.Update rather than atomicUpdate for the
// same reason DecayDamageEntries is: Model exposes no builder path that clears
// the damage-entry slice. aggroFlippedOff and controllerCharacterId derive
// purely from cur, so the captured values reflect the final successful
// invocation under optimistic-lock retry.
func (r *Registry) ClearDamageEntries(t tenant.Model, uniqueId uint32) (ClearSummary, error) {
	ctx := context.Background()

	var aggroFlippedOff bool
	var controllerCharacterId uint32
	sm, err := r.reg.Update(ctx, monsterSuffix(t, uniqueId), func(cur storedMonster) storedMonster {
		aggroFlippedOff = false

		cur.DamageEntries = nil
		if cur.ControllerHasAggro {
			cur.ControllerHasAggro = false
			aggroFlippedOff = true
		}

		controllerCharacterId = cur.ControlCharacterId
		return cur
	})
	if errors.Is(err, atlasredis.ErrNotFound) {
		return ClearSummary{}, errMonsterNotFound
	}
	if err != nil {
		return ClearSummary{}, err
	}
	_, m, err := fromStored(sm)
	if err != nil {
		return ClearSummary{}, err
	}
	return ClearSummary{
		Monster:               m,
		ControllerCharacterId: controllerCharacterId,
		AggroFlippedOff:       aggroFlippedOff,
	}, nil
}
```

and next to `ControlMonster`:

```go
// ControlMonsterWithAggro assigns the controller and sets the aggro flag in one
// atomic transition. Used by the forced-control path so the resulting
// START_CONTROL event carries controllerHasAggro = true.
func (r *Registry) ControlMonsterWithAggro(tenant tenant.Model, uniqueId uint32, characterId uint32) (Model, error) {
	return r.atomicUpdate(context.Background(), tenant, uniqueId, func(m Model) Model {
		return m.ControlWithAggro(characterId)
	})
}
```

> Before settling on `cur.DamageEntries = nil`, check how `fromStored`/`toStored` round-trip that field. If a `nil` slice decodes differently from an empty one in the stored form, use the empty-slice form instead. The "full wipe" test from Step 1 catches either mistake.

- [ ] **Step 5: Split `startControl` and add the two processor methods**

In `monster/processor.go`, add to the `Processor` interface after `Catch`:

```go
	ClearAggro(uniqueId uint32) error
	ForceControl(uniqueId uint32, characterId uint32) error
```

Refactor `StartControl` into a shared core. Replace the existing `StartControl` (`:387-417`) with:

```go
// StartControl starts a character controlling a monster.
func (p *ProcessorImpl) StartControl(uniqueId uint32, controllerId uint32) (Model, error) {
	return p.startControl(uniqueId, controllerId, false)
}

// startControl is the shared control-transfer core. forceAggro additionally
// marks the new controller as holding aggro in the same atomic transition, so
// the emitted START_CONTROL drives StartControlMonsterBody(m, true) on the
// channel side. The stop-then-start sequencing, the START_CONTROL emission and
// the RepickReasonControlChange semantics below are unchanged from before the
// split — no caller writes controller state directly.
func (p *ProcessorImpl) startControl(uniqueId uint32, controllerId uint32, forceAggro bool) (Model, error) {
	m, err := p.GetById(uniqueId)
	if err != nil {
		return Model{}, err
	}

	if m.ControlCharacterId() != 0 {
		err = p.StopControl(m)
		if err != nil {
			return Model{}, err
		}
	}

	if forceAggro {
		m, err = GetMonsterRegistry().ControlMonsterWithAggro(p.t, uniqueId, controllerId)
	} else {
		m, err = GetMonsterRegistry().ControlMonster(p.t, uniqueId, controllerId)
	}
	if err == nil {
		_ = p.emit(EnvEventTopicMonsterStatus, startControlStatusEventProvider(m))
		// FR-2.3 parity: a controller-change must not start a fresh skill
		// decision when the new controller has no aggro. Without this guard
		// every mob in a map picks a skill the moment a player walks in (e.g.
		// 12 freshly-spawned Wyverns all decide skill 126 on entry, then the
		// channel inbox serves the prediction into MoveMonsterAck and the
		// client animates 12 simultaneous casts). Mirrors postExecute's
		// ControllerHasAggro gate in UseSkill.
		//
		// A forced handover deliberately satisfies this gate: the mobs were
		// just aggroed onto the caster, and the fan-out is bounded by the
		// skill's WZ mobCount (<= 7), unlike the map-entry storm above.
		if !m.ControllerHasAggro() {
			p.l.Debugf("Controller-change picker: monster [%d] new controller [%d] has no aggro; skipping re-pick.", uniqueId, controllerId)
		} else if rerr := p.RepickAndEmit(uniqueId, RepickReasonControlChange); rerr != nil {
			p.l.WithError(rerr).Warnf("Controller-change picker: monster [%d] re-pick failed.", uniqueId)
		}
	}
	return m, err
}
```

Add the two new methods at the end of the file:

```go
// ClearAggro fully wipes the monster's damage-aggro table. Idempotent: wiping
// an already-empty table emits nothing and returns nil (FR-4.5). A command
// naming a monster that no longer exists is logged and dropped, not retried
// into an error loop (FR-4.6).
func (p *ProcessorImpl) ClearAggro(uniqueId uint32) error {
	summary, err := GetMonsterRegistry().ClearDamageEntries(p.t, uniqueId)
	if err != nil {
		if errors.Is(err, errMonsterNotFound) {
			p.l.Debugf("CLEAR_AGGRO for monster [%d]: monster no longer exists; dropping.", uniqueId)
			return nil
		}
		return err
	}
	if summary.AggroFlippedOff {
		_ = p.emit(EnvEventTopicMonsterStatus,
			aggroChangedStatusEventProvider(summary.Monster, summary.ControllerCharacterId, false))
	}
	return nil
}

// ForceControl hands the monster's controller to characterId with the aggro
// flag set, bypassing the picker election. Every rejection path is a logged
// drop returning nil, never an error: these commands arrive from a client-driven
// cast and a stale target must not wedge the consumer.
func (p *ProcessorImpl) ForceControl(uniqueId uint32, characterId uint32) error {
	m, err := p.GetById(uniqueId)
	if err != nil {
		p.l.Debugf("FORCE_CONTROL for monster [%d]: monster no longer exists; dropping.", uniqueId)
		return nil
	}

	// FR-5.4 — forcing control to the current controller must not emit a
	// redundant stop/start pair on every client in the field.
	if m.ControlCharacterId() == characterId {
		p.l.Debugf("FORCE_CONTROL for monster [%d]: character [%d] already controls it; no-op.", uniqueId, characterId)
		return nil
	}

	ids, err := p.inFieldFn(m.Field())
	if err != nil {
		p.l.WithError(err).Debugf("FORCE_CONTROL for monster [%d]: unable to list characters in field [%s]; dropping.", uniqueId, m.Field().Id())
		return nil
	}
	present := false
	for _, id := range ids {
		if id == characterId {
			present = true
			break
		}
	}
	if !present {
		p.l.Debugf("FORCE_CONTROL for monster [%d]: character [%d] is not in field [%s]; dropping.", uniqueId, characterId, m.Field().Id())
		return nil
	}

	// A GM-hidden character must not be granted control: RelinquishControlOnHide
	// actively strips control from hidden characters, so granting it here would
	// produce an immediate flap.
	if hiddenIds, herr := p.hiddenFn(); herr == nil {
		if _, isHidden := hiddenIds[characterId]; isHidden {
			p.l.Debugf("FORCE_CONTROL for monster [%d]: character [%d] is GM-hidden; dropping.", uniqueId, characterId)
			return nil
		}
	}

	if _, serr := p.startControl(uniqueId, characterId, true); serr != nil {
		p.l.WithError(serr).Warnf("FORCE_CONTROL for monster [%d] to character [%d] failed.", uniqueId, characterId)
	}
	return nil
}
```

- [ ] **Step 6: Register the two consumer handlers**

In `kafka/consumer/monster/consumer.go`, add two registrations inside `InitHandlers` after the `handleCatchCommand` block:

```go
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleClearAggroCommand))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleForceControlCommand))); err != nil {
				return err
			}
```

and the two handlers after `handleCatchCommand`:

```go
func handleClearAggroCommand(l logrus.FieldLogger, ctx context.Context, c command[clearAggroCommandBody]) {
	if c.Type != CommandTypeClearAggro {
		return
	}

	p := monster.NewProcessor(l, ctx)
	if err := p.ClearAggro(c.MonsterId); err != nil {
		l.WithError(err).Errorf("CLEAR_AGGRO failed for monster [%d].", c.MonsterId)
	}
}

func handleForceControlCommand(l logrus.FieldLogger, ctx context.Context, c command[forceControlCommandBody]) {
	if c.Type != CommandTypeForceControl {
		return
	}

	p := monster.NewProcessor(l, ctx)
	if err := p.ForceControl(c.MonsterId, c.Body.CharacterId); err != nil {
		l.WithError(err).Errorf("FORCE_CONTROL failed for monster [%d] character [%d].", c.MonsterId, c.Body.CharacterId)
	}
}
```

- [ ] **Step 7: Run the tests to verify they pass**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-monsters/atlas.com/monsters" && \
  go test ./monster/ -run 'ClearAggro|ForceControl' -v
```

Expected: PASS, all eleven cases (four in `clear_aggro_test.go`, seven in `force_control_test.go`).

- [ ] **Step 8: Run the full module suite**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-monsters/atlas.com/monsters" && \
  go test -race ./... && go vet ./... && go build ./...
```

Expected: PASS. The `startControl` split must leave every existing control/picker test green.

- [ ] **Step 9: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add services/atlas-monsters/atlas.com/monsters/
git commit -m "feat(task-215): monster aggro wipe and forced controller handover

ClearDamageEntries converges on the same state DecayDamageEntries reaches when
its list empties, so the decay sweep and picker behave identically after a wipe.
ForceControl routes through the existing StartControl sequencing via a
forceAggro split rather than writing controller state directly."
```

---

### Task 5: Per-version byte fixtures

**Files:**
- Test: `libs/atlas-packet/model/skill_usage_info_magnet_versions_test.go` (create)

**Interfaces:**
- Consumes: `MagnetGrab`, `NewMagnetGrab`, `MagnetGrabs()`, `Direction()`, and the three test builders from Task 1.
- Produces: nothing consumed by later tasks.

> **Read `context.md` §2 before starting this task.** PRD FR-8's `packet-audit:verify` marker plus pinned evidence record is **not achievable as written**, and this task deliberately omits both. In short: the matrix's unit of promotion is the whole `SPECIAL_MOVE` op × version cell, which carries **sixteen** fnames of which `CUserLocal::TryDoingMonsterMagnet` is one; a marker that matches no evidence record or audit report is reported as `orphan marker …` and hard-fails `matrix --check` (`tools/packet-audit/cmd/matrix.go:232-249`, `matrix_markers_test.go:100-109`), while a marker that *does* match promotes the whole cell to ✅ with fifteen fnames still unverified — the false ✅ FR-8.3 forbids.
>
> **Do NOT write a `packet-audit:verify` marker in this file. Do NOT pin an evidence record. Do NOT edit `status.json` or `STATUS.md`.** If the user has since decided otherwise, that decision changes this task's scope substantially — stop and re-plan rather than improvising.

- [ ] **Step 1: Write the ten fixtures**

Create `libs/atlas-packet/model/skill_usage_info_magnet_versions_test.go`. It reuses `decodeMagnetBody`, `modernMagnetBody` and `legacyMagnetBody` from Task 1's test file (same package).

```go
package model

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// TestMagnetByteLayoutPerVersion is the per-version regression fixture for the
// Monster Magnet arm of the use-skill opcode. Each row's expected byte count and
// field order was derived by reading that version's
// CUserLocal::TryDoingMonsterMagnet packet-build tail instruction-by-instruction
// (addresses below; four of the ten were unnamed in the IDB and were renamed to
// CUserLocal__TryDoingMonsterMagnet there during derivation).
//
// This deliberately carries NO `packet-audit:verify` marker and NO pinned
// evidence record: the coverage matrix's unit of promotion is the whole
// SPECIAL_MOVE op cell, which carries sixteen fnames of which this is one, and a
// marker would either orphan (failing `matrix --check`) or promote a cell whose
// other fifteen fnames are unverified. See
// docs/tasks/task-215-monster-magnet/context.md section 2.
//
// The load-bearing assertion is `left == 0`: a layout that is wrong in any
// width, order, or presence/absence of a trailing field leaves the reader short
// of or past the end of the body.
func TestMagnetByteLayoutPerVersion(t *testing.T) {
	grabs := []MagnetGrab{
		NewMagnetGrab(1001, true),
		NewMagnetGrab(1002, false),
	}

	tests := []struct {
		name        string
		region      string
		major       uint16
		minor       uint16
		skillId     skill.Id
		legacy      bool
		ida         string
		wantGrabs   int
		wantDelay   uint16
		wantLeft    bool
		wantByteLen int
	}{
		{
			name: "gms_v48_legacy", region: "GMS", major: 48, minor: 1,
			skillId: skill.HeroMonsterMagnetId, legacy: true,
			// CUserLocal::TryDoingMonsterMagnet @0x6AD842; COutPacket ctor
			// `push 46h` @0x6ADABC (opcode 0x46, matching template_gms_48_1.json's
			// CharacterUseSkillHandle). updateTime Encode4 @0x6ADAD3, skillId
			// Encode4 @0x6ADAE0, skillLevel Encode1 @0x6ADAEB, entryCount Encode1
			// @0x6ADB02 (ONE byte), per-entry Encode4 @0x6ADB1B (NO result byte),
			// delay Encode2 @0x6ADB29 (NO direction byte). entry[0] is the caster:
			// ZArray<ulong>::InsertBefore @0x6AD977-0x6AD987 pushes [esi+0x654]
			// (esi = CUserLocal) before the mob loop @0x6ADA89-0x6ADA99 appends
			// [mob+0x654].
			ida: "0x6AD842", wantGrabs: 2, wantDelay: 750, wantLeft: false,
			// 4+4+1 + 1 + 4*(1 caster + 2 mobs) + 2 = 24
			wantByteLen: 24,
		},
		{
			name: "gms_v61_modern", region: "GMS", major: 61, minor: 1,
			skillId: skill.HeroMonsterMagnetId, legacy: false,
			// CUserLocal::TryDoingMonsterMagnet @0x7B9684; COutPacket(83) = 0x53,
			// matching template_gms_61_1.json's CharacterUseSkillHandle.
			ida: "0x7B9684", wantGrabs: 2, wantDelay: 0, wantLeft: true,
			// 4+4+1 + 4 + 5*2 + 1 = 24
			wantByteLen: 24,
		},
		{
			name: "gms_v72_modern", region: "GMS", major: 72, minor: 1,
			skillId: skill.PaladinMonsterMagnetId, legacy: false,
			// @0x876605; encodes @0x876A2B/38/43/4E, loop @0x876A86,0x876A95,
			// tail (direction) @0x876AB2. Opcode 0x5A.
			ida: "0x876605", wantGrabs: 2, wantDelay: 0, wantLeft: true,
			wantByteLen: 24,
		},
		{
			name: "gms_v79_modern", region: "GMS", major: 79, minor: 1,
			skillId: skill.PaladinMonsterMagnetId, legacy: false,
			// @0x8C3117; encodes @0x8C3540/4D/58/63, loop @0x8C359B,0x8C35AA,
			// tail @0x8C35C7. Opcode 0x59.
			ida: "0x8C3117", wantGrabs: 2, wantDelay: 0, wantLeft: false,
			wantByteLen: 24,
		},
		{
			name: "gms_v83_modern", region: "GMS", major: 83, minor: 1,
			skillId: skill.DarkKnightMonsterMagnetId, legacy: false,
			// @0x96C215; `COutPacket::Encode1(v65, *v40 == 3)` is the per-entry
			// grab bool. Opcode 0x5B.
			ida: "0x96C215", wantGrabs: 2, wantDelay: 0, wantLeft: true,
			wantByteLen: 24,
		},
		{
			name: "gms_v84_modern", region: "GMS", major: 84, minor: 1,
			skillId: skill.DarkKnightMonsterMagnetId, legacy: false,
			// @0x9ABDB7; encodes @0x9AC1F3/200/20B/216, loop @0x9AC24E,0x9AC25D,
			// tail @0x9AC27D. Opcode 0x5B.
			ida: "0x9ABDB7", wantGrabs: 2, wantDelay: 0, wantLeft: false,
			wantByteLen: 24,
		},
		{
			name: "gms_v87_modern", region: "GMS", major: 87, minor: 1,
			skillId: skill.HeroMonsterMagnetId, legacy: false,
			// @0x9F086F; encodes @0x9F0CAB/CB8/CC3/CCE, loop @0x9F0D06,0x9F0D15,
			// tail @0x9F0D35. Opcode 0x5E.
			ida: "0x9F086F", wantGrabs: 2, wantDelay: 0, wantLeft: true,
			wantByteLen: 24,
		},
		{
			name: "gms_v92_modern", region: "GMS", major: 92, minor: 1,
			skillId: skill.HeroMonsterMagnetId, legacy: false,
			// @0x91F2A0; encodes @0x91FA54/60/71/7F, loop @0x91FABD,0x91FAE1,
			// tail @0x91FAFB. Opcode 0x66.
			ida: "0x91F2A0", wantGrabs: 2, wantDelay: 0, wantLeft: false,
			wantByteLen: 24,
		},
		{
			name: "gms_v95_modern", region: "GMS", major: 95, minor: 1,
			skillId: skill.PaladinMonsterMagnetId, legacy: false,
			// @0x940570; encodes @0x940D25/31/42/50, loop @0x940D8D,0x940DB1,
			// tail @0x940DCB. Opcode 0x67.
			ida: "0x940570", wantGrabs: 2, wantDelay: 0, wantLeft: true,
			wantByteLen: 24,
		},
		{
			name: "jms_v185_modern", region: "JMS", major: 185, minor: 1,
			skillId: skill.DarkKnightMonsterMagnetId, legacy: false,
			// @0xA3C61C; encodes @0xA3CC52/5C/67/72, loop @0xA3CCAA,0xA3CCC6,
			// tail @0xA3CCE3. Opcode 0x56. jms takes the MODERN branch.
			ida: "0xA3C61C", wantGrabs: 2, wantDelay: 0, wantLeft: false,
			wantByteLen: 24,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf []byte
			if tc.legacy {
				buf = legacyMagnetBody(tc.skillId, 30, 900001, []uint32{1001, 1002}, tc.wantDelay)
			} else {
				buf = modernMagnetBody(tc.skillId, 30, grabs, tc.wantLeft)
			}
			if len(buf) != tc.wantByteLen {
				t.Fatalf("fixture is %d bytes, want %d (%s)", len(buf), tc.wantByteLen, tc.ida)
			}

			m, left := decodeMagnetBody(t, tc.region, tc.major, tc.minor, buf)
			if left != 0 {
				t.Fatalf("%s: reader has %d unconsumed bytes — layout wrong (%s)", tc.name, left, tc.ida)
			}
			if len(m.MagnetGrabs()) != tc.wantGrabs {
				t.Fatalf("%s: MagnetGrabs len = %d, want %d (%s)",
					tc.name, len(m.MagnetGrabs()), tc.wantGrabs, tc.ida)
			}
			if m.MagnetGrabs()[0].ObjectId() != 1001 || m.MagnetGrabs()[1].ObjectId() != 1002 {
				t.Fatalf("%s: object ids = [%d %d], want [1001 1002] (%s)",
					tc.name, m.MagnetGrabs()[0].ObjectId(), m.MagnetGrabs()[1].ObjectId(), tc.ida)
			}
			if m.Delay() != tc.wantDelay {
				t.Fatalf("%s: Delay = %d, want %d (%s)", tc.name, m.Delay(), tc.wantDelay, tc.ida)
			}
			if m.Direction() != tc.wantLeft {
				t.Fatalf("%s: Direction = %v, want %v (%s)", tc.name, m.Direction(), tc.wantLeft, tc.ida)
			}

			// Per-entry grab results: the modern shape carries a bool per entry;
			// the legacy shape has none and every surviving entry is a grab.
			if tc.legacy {
				for i, g := range m.MagnetGrabs() {
					if !g.Grabbed() {
						t.Fatalf("%s: grab[%d] not grabbed; v48 sends no per-entry result (%s)", tc.name, i, tc.ida)
					}
				}
			} else {
				if !m.MagnetGrabs()[0].Grabbed() || m.MagnetGrabs()[1].Grabbed() {
					t.Fatalf("%s: grab results = [%v %v], want [true false] (%s)",
						tc.name, m.MagnetGrabs()[0].Grabbed(), m.MagnetGrabs()[1].Grabbed(), tc.ida)
				}
			}
		})
	}
}
```

> The `wantLeft` values alternate across rows purely to exercise both direction states on the modern branch; they are test inputs, not per-version facts. The `legacy` column is the per-version fact.

- [ ] **Step 2: Run the fixtures**

```bash
cd "$(git rev-parse --show-toplevel)/libs/atlas-packet" && go test ./model/ -run TestMagnetByteLayoutPerVersion -v
```

Expected: PASS, ten subtests.

- [ ] **Step 3: Verify no marker was written and the matrix is untouched**

```bash
cd "$(git rev-parse --show-toplevel)"
grep -rn "packet-audit:verify" libs/atlas-packet/model/ || echo "OK: no marker in model/"
go run ./tools/packet-audit matrix --check && echo "OK: matrix clean"
git status --short docs/packets/
```

Expected: no marker under `libs/atlas-packet/model/`; `matrix --check` exits 0; `git status` reports nothing under `docs/packets/`.

- [ ] **Step 4: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add libs/atlas-packet/model/skill_usage_info_magnet_versions_test.go
git commit -m "test(task-215): per-version byte fixtures for the Monster Magnet arm

Ten fixtures, one per provisioned version, each citing the decompiled encode
addresses it was derived from. No packet-audit:verify marker and no pinned
evidence: the matrix promotes whole op cells and SPECIAL_MOVE carries fifteen
other unverified fnames. See docs/tasks/task-215-monster-magnet/context.md."
```

---

### Task 6: Target-validation helpers and the Monster Magnet handler

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/common.go:226-237,290`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/mob_select_test.go`
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/monstermagnet/monstermagnet.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/monstermagnet/monstermagnet_test.go` (create)

**Interfaces:**
- Consumes: `packetmodel.MagnetGrab` / `MagnetGrabs()` (Task 1); `effect.Model.Range()` (Task 2); `monster.Processor.ClearAggro` / `ForceControl` (Task 3).
- Produces:
  - `func MagnetRegion(casterX, casterY int16, facingLeft bool, skillRange int32) (x1, y1, x2, y2 int16)` (exported from `skill/handler`)
  - `func IntersectMobIds(client, server []uint32) (applied, anomaly []uint32)` (renamed from `intersectMobIds`)
  - `func ExceedsMobCap(l logrus.FieldLogger, event string, characterId uint32, sid skill2.Id, slvl uint32, mobCap uint32, mobIds []uint32) bool`
  - `monstermagnet.Apply` registered for all three magnet Identities.

Geometry (design §4.2/§3.1). The client selects candidates through `CMobPool::CheckMobInTrapezoid` (gms_83 `0x679084`) with `xStart = casterX ± 50`, `xEnd = casterX ± range`, `y = casterY - 28`, slope 4 — a wedge whose half-height grows as `|dx|/4`. The server computes the **axis-aligned bounding box** of that wedge, not the wedge itself, because the client tests each mob's **body rect** while atlas-monsters exposes only the mob's anchor point.

- [ ] **Step 1: Write the failing tests**

First, extend `mob_select_test.go` with the geometry and cap cases, and update its five existing `intersectMobIds(...)` call sites to `IntersectMobIds(...)`. Add `"github.com/sirupsen/logrus"` to that file's imports.

```go
// TestMagnetRegionFacingRight pins the AABB of the client's trapezoid for a
// right-facing caster. The client walks x from casterX+50 out to casterX+range
// with half-height |dx|/4 about casterY-28 (CMobPool::CheckMobInTrapezoid,
// gms_83 @0x679084).
func TestMagnetRegionFacingRight(t *testing.T) {
	x1, y1, x2, y2 := MagnetRegion(1000, 500, false, 450)
	if x1 >= x2 {
		t.Fatalf("x bounds not ordered: %d..%d", x1, x2)
	}
	if x1 > 1000+50 {
		t.Fatalf("x1 = %d; the near edge must not exclude mobs at casterX+50", x1)
	}
	if x2 < 1000+450 {
		t.Fatalf("x2 = %d; the far edge must reach casterX+range (1450)", x2)
	}
	if y1 >= y2 {
		t.Fatalf("y bounds not ordered: %d..%d", y1, y2)
	}
	if y2 < 500-28+450/4 {
		t.Fatalf("y2 = %d; the box must cover the wedge's max half-height (%d)", y2, 500-28+450/4)
	}
}

func TestMagnetRegionFacingLeftMirrors(t *testing.T) {
	rx1, _, rx2, _ := MagnetRegion(1000, 500, false, 450)
	lx1, _, lx2, _ := MagnetRegion(1000, 500, true, 450)
	if lx1 >= lx2 {
		t.Fatalf("facing left: x bounds not ordered: %d..%d", lx1, lx2)
	}
	if (rx2 - rx1) != (lx2 - lx1) {
		t.Fatalf("mirrored widths differ: right=%d left=%d", rx2-rx1, lx2-lx1)
	}
	if lx1 > 1000-450 {
		t.Fatalf("facing left: x1 = %d must reach casterX-range (550)", lx1)
	}
}

func TestExceedsMobCapRejectsWholeCast(t *testing.T) {
	l := logrus.New()
	if !ExceedsMobCap(l, "test_over_cap", 1, 1121001, 30, 3, []uint32{1, 2, 3, 4}) {
		t.Fatal("4 claimed targets against a cap of 3 must exceed the cap")
	}
	if ExceedsMobCap(l, "test_over_cap", 1, 1121001, 30, 3, []uint32{1, 2, 3}) {
		t.Fatal("3 claimed targets against a cap of 3 must not exceed the cap")
	}
}
```

Then create `monstermagnet/monstermagnet_test.go`, replacing the package's seams with `t.Cleanup`-restored overrides — the pattern `dispel/dispel_test.go` uses. Effect models are built through `effect.Extract(effect.RestModel{...})`, matching `common_apply_to_mobs_test.go:134-162`.

```go
package monstermagnet

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/monster"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"io"
	"testing"

	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

const casterId = uint32(1)

func tl() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func testField() field.Model {
	return field.NewBuilder(0, 0, 1).Build()
}

// magnetEffect builds the WZ effect for a level-30 magnet: mobCount 7,
// range 450 (design section 3). rangeValue 0 exercises the cap-only fallback.
func magnetEffect(t *testing.T, mobCount uint32, skillRange int32) effect.Model {
	t.Helper()
	e, err := effect.Extract(effect.RestModel{MobCount: mobCount, Range: skillRange})
	if err != nil {
		t.Fatalf("effect.Extract: %v", err)
	}
	return e
}

func magnetInfo(grabs ...packetmodel.MagnetGrab) packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(uint32(skill2.HeroMonsterMagnetId)).
		SetSkillLevel(30).
		SetMagnetGrabs(grabs).
		Build()
}

// call records one seam invocation so tests can assert both counts and order.
type call struct {
	kind      string // "announce" | "clear" | "force"
	monsterId uint32
	result    byte
	success   byte
}

// stubs installs all five seams and returns the recorded call log. rectIds is
// what the rect query reports as present server-side; a nil rectErr/casterErr
// means the corresponding seam succeeds.
func stubs(t *testing.T, rectIds []uint32, casterErr, rectErr error, rectCalls *int) *[]call {
	t.Helper()
	origCaster, origRect := loadCasterFunc, rectQueryFunc
	origAnnounce, origClear, origForce := announceCatchFunc, clearAggroFunc, forceControlFunc
	t.Cleanup(func() {
		loadCasterFunc, rectQueryFunc = origCaster, origRect
		announceCatchFunc, clearAggroFunc, forceControlFunc = origAnnounce, origClear, origForce
	})

	var calls []call

	loadCasterFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (character.Model, error) {
		if casterErr != nil {
			return character.Model{}, casterErr
		}
		// stance defaults to 0 (facing right); the character builder has no
		// SetStance, and MagnetRegion's left-facing branch is covered by
		// TestMagnetRegionFacingLeftMirrors in skill/handler.
		return character.NewModelBuilder().SetId(characterId).SetX(1000).SetY(500).MustBuild(), nil
	}
	rectQueryFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]monster.Model, error) {
		if rectCalls != nil {
			*rectCalls++
		}
		if rectErr != nil {
			return nil, rectErr
		}
		mobs := make([]monster.Model, 0, len(rectIds))
		for _, id := range rectIds {
			// monster.NewModelBuilder takes (uniqueId, field, monsterId).
			m, berr := monster.NewModelBuilder(id, testField(), 9000000).Build()
			if berr != nil {
				t.Fatalf("monster.NewModelBuilder(%d): %v", id, berr)
			}
			mobs = append(mobs, m)
		}
		return mobs, nil
	}
	announceCatchFunc = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, cid uint32, monsterId uint32) error {
		calls = append(calls, call{kind: "announce", monsterId: monsterId, result: grabResultSuccess, success: grabSuccessFlag})
		return nil
	}
	clearAggroFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32) error {
		calls = append(calls, call{kind: "clear", monsterId: monsterId})
		return nil
	}
	forceControlFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32, characterId uint32) error {
		calls = append(calls, call{kind: "force", monsterId: monsterId})
		return nil
	}
	return &calls
}

func countKind(calls []call, kind string) int {
	n := 0
	for _, c := range calls {
		if c.kind == kind {
			n++
		}
	}
	return n
}

func TestMagnetHappyPathEmitsInOrder(t *testing.T) {
	calls := stubs(t, []uint32{1001, 1002}, nil, nil, nil)

	err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(packetmodel.NewMagnetGrab(1001, true), packetmodel.NewMagnetGrab(1002, true)),
		magnetEffect(t, 7, 450))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	got := *calls
	if len(got) != 6 {
		t.Fatalf("recorded %d calls, want 6 (announce+clear+force per monster): %+v", len(got), got)
	}
	want := []call{
		{kind: "announce", monsterId: 1001, result: 1, success: 1},
		{kind: "clear", monsterId: 1001},
		{kind: "force", monsterId: 1001},
		{kind: "announce", monsterId: 1002, result: 1, success: 1},
		{kind: "clear", monsterId: 1002},
		{kind: "force", monsterId: 1002},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("call %d = %+v, want %+v — CLEAR_AGGRO must precede FORCE_CONTROL per monster", i, got[i], want[i])
		}
	}
}

func TestMagnetSkipsFailedGrabsAndReleasedSlots(t *testing.T) {
	calls := stubs(t, []uint32{1001, 1002, 1003}, nil, nil, nil)

	err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(
			packetmodel.NewMagnetGrab(1001, true),
			packetmodel.NewMagnetGrab(1002, false), // client reports a failed grab
			packetmodel.NewMagnetGrab(0, true),     // released slot
		),
		magnetEffect(t, 7, 450))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	for _, c := range *calls {
		if c.monsterId != 1001 {
			t.Fatalf("acted on monster [%d]; only the successful, non-zero grab may be acted on: %+v", c.monsterId, *calls)
		}
	}
	if countKind(*calls, "announce") != 1 || countKind(*calls, "clear") != 1 || countKind(*calls, "force") != 1 {
		t.Fatalf("expected exactly one of each call kind, got %+v", *calls)
	}
}

func TestMagnetOverCapRejectsWholeCast(t *testing.T) {
	calls := stubs(t, []uint32{1001, 1002, 1003, 1004}, nil, nil, nil)

	err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(
			packetmodel.NewMagnetGrab(1001, true),
			packetmodel.NewMagnetGrab(1002, true),
			packetmodel.NewMagnetGrab(1003, true),
			packetmodel.NewMagnetGrab(1004, true),
		),
		magnetEffect(t, 3, 450)) // cap 3, four claimed
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("recorded %d calls, want 0 — an over-cap cast grabs NOTHING (FR-2.2): %+v", len(*calls), *calls)
	}
}

func TestMagnetOutOfRegionDropsIndividually(t *testing.T) {
	// The server sees only 1001; the client also claimed 1002.
	calls := stubs(t, []uint32{1001}, nil, nil, nil)

	err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(packetmodel.NewMagnetGrab(1001, true), packetmodel.NewMagnetGrab(1002, true)),
		magnetEffect(t, 7, 450))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(*calls) != 3 {
		t.Fatalf("recorded %d calls, want 3 — the in-region monster proceeds, the other is dropped: %+v", len(*calls), *calls)
	}
	for _, c := range *calls {
		if c.monsterId != 1001 {
			t.Fatalf("acted on out-of-region monster [%d]", c.monsterId)
		}
	}
}

func TestMagnetCasterLoadFailureDropsWholeCast(t *testing.T) {
	calls := stubs(t, []uint32{1001}, errors.New("boom"), nil, nil)

	err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(packetmodel.NewMagnetGrab(1001, true)),
		magnetEffect(t, 7, 450))
	if err != nil {
		t.Fatalf("Apply must return nil even on a dropped cast: %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("recorded %d calls, want 0 (FR-2.7): %+v", len(*calls), *calls)
	}
}

func TestMagnetRectQueryFailureDropsWholeCast(t *testing.T) {
	calls := stubs(t, nil, nil, errors.New("boom"), nil)

	err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(packetmodel.NewMagnetGrab(1001, true)),
		magnetEffect(t, 7, 450))
	if err != nil {
		t.Fatalf("Apply must return nil even on a dropped cast: %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("recorded %d calls, want 0 (FR-2.7): %+v", len(*calls), *calls)
	}
}

// TestMagnetIssuesExactlyOneRectQuery pins the NFR performance contract: one
// rect query per cast, not one lookup per claimed monster.
func TestMagnetIssuesExactlyOneRectQuery(t *testing.T) {
	rectCalls := 0
	ids := []uint32{1001, 1002, 1003, 1004, 1005}
	stubs(t, ids, nil, nil, &rectCalls)

	grabs := make([]packetmodel.MagnetGrab, 0, len(ids))
	for _, id := range ids {
		grabs = append(grabs, packetmodel.NewMagnetGrab(id, true))
	}

	if err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(grabs...), magnetEffect(t, 7, 450)); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if rectCalls != 1 {
		t.Fatalf("issued %d rect queries, want exactly 1 for a 5-monster cast", rectCalls)
	}
}

func TestMagnetNoRangeFallsBackToCapOnly(t *testing.T) {
	rectCalls := 0
	calls := stubs(t, nil, nil, nil, &rectCalls)

	err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(packetmodel.NewMagnetGrab(1001, true), packetmodel.NewMagnetGrab(1002, true)),
		magnetEffect(t, 7, 0)) // no range in this tenant's WZ data
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if rectCalls != 0 {
		t.Fatalf("issued %d rect queries with no range, want 0", rectCalls)
	}
	if countKind(*calls, "force") != 2 {
		t.Fatalf("expected both capped grabs to proceed on the cap-only path, got %+v", *calls)
	}
}

func TestMagnetReturnsNilWhenCommandsFail(t *testing.T) {
	stubs(t, []uint32{1001}, nil, nil, nil)
	origClear, origForce := clearAggroFunc, forceControlFunc
	t.Cleanup(func() { clearAggroFunc, forceControlFunc = origClear, origForce })
	clearAggroFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32) error {
		return errors.New("kafka down")
	}
	forceControlFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32, characterId uint32) error {
		return errors.New("kafka down")
	}

	// Apply must still return nil: a non-nil return makes UseSkill log a second
	// error, and the caller's EnableActions unlock must never be aborted.
	if err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(packetmodel.NewMagnetGrab(1001, true)), magnetEffect(t, 7, 450)); err != nil {
		t.Fatalf("Apply returned %v, want nil", err)
	}
}

func TestMagnetRegisteredOnAllThreeIdentities(t *testing.T) {
	for _, id := range []skill2.Identity{
		skill2.HeroMonsterMagnet,
		skill2.PaladinMonsterMagnet,
		skill2.DarkKnightMonsterMagnet,
	} {
		if h, ok := channelhandler.Lookup(id); !ok || h == nil {
			t.Fatalf("no handler registered for identity [%v]", id)
		}
	}
}
```

Two assertions are structural rather than behavioural — Task 9 Steps 5 and 6 check them with grep rather than contorting a unit test: that `CATCH_MONSTER_WITH_ITEM` is never referenced from the magnet path (FR-3.3), and that the fan-out uses `ForOtherSessionsInMap` so the caster is excluded.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-channel/atlas.com/channel" && \
  go test ./skill/handler/ -run 'MagnetRegion|ExceedsMobCap' -v; \
  go test ./skill/handler/monstermagnet/ -v
```

Expected: FAIL to compile — `MagnetRegion`, `ExceedsMobCap` undefined; the `monstermagnet` package does not exist.

- [ ] **Step 3: Add the geometry and the two extracted helpers**

In `skill/handler/mob_select.go`, rename `intersectMobIds` → `IntersectMobIds` (keeping its existing doc comment verbatim), update the single production call site at `common.go:290`, add `"github.com/sirupsen/logrus"` to the imports, and append:

```go
// magnetNearOffset, magnetYAnchor and magnetSlopeDivisor reproduce the
// arguments the client passes to CMobPool::CheckMobInTrapezoid from
// CUserLocal::TryDoingMonsterMagnet (gms_83 @0x679084): the wedge starts at
// casterX ± 50, is centred on casterY - 28, and opens with half-height |dx|/4.
//
// magnetBodyMargin pads the resulting box. The client intersects each mob's
// BODY RECT against the wedge; atlas-monsters' GetInMapRect only exposes the
// mob's anchor point, so reproducing the wedge exactly against a point would
// reject legitimate grabs of tall mobs near the edge. The margin stands in for
// the unmodelled body rect. This makes the server region a strict superset of
// the client's, which is the correct posture for an anti-cheat gate: it still
// rejects a target on the other side of the map or beyond `range`, and it never
// fights sub-pixel geometry.
const (
	magnetNearOffset   = 50
	magnetYAnchor      = 28
	magnetSlopeDivisor = 4
	magnetBodyMargin   = 60
)

// MagnetRegion returns the axis-aligned bounding box of the client's Monster
// Magnet target trapezoid, as (x1, y1, x2, y2). skillRange is the effect's WZ
// `range` attribute. The tuple is normalized (x1 <= x2, y1 <= y2).
//
// Monster Magnet carries no lt/rb in WZ, so calculateBoundingBox is not
// applicable to it — see docs/tasks/task-215-monster-magnet/design.md section 3.
func MagnetRegion(casterX, casterY int16, facingLeft bool, skillRange int32) (x1, y1, x2, y2 int16) {
	sign := int32(1)
	if facingLeft {
		sign = -1
	}
	cx := int32(casterX)
	near := cx + sign*magnetNearOffset
	far := cx + sign*skillRange
	if near > far {
		near, far = far, near
	}
	x1 = int16(near - magnetBodyMargin)
	x2 = int16(far + magnetBodyMargin)

	halfHeight := skillRange/magnetSlopeDivisor + magnetBodyMargin
	yc := int32(casterY) - magnetYAnchor
	y1 = int16(yc - halfHeight)
	y2 = int16(yc + halfHeight)
	return
}

// ExceedsMobCap reports whether the client claimed more targets than the
// skill's WZ mobCount permits, logging the over-cap anomaly under the caller's
// `event` discriminator. Extracted from applyToMobs (FR-2.6) so every
// client-target-set consumer enforces the identical reject-the-whole-cast
// policy with the identical log field vocabulary, which is what lets the
// existing monster_buff_anomaly_* dashboards pick both up.
func ExceedsMobCap(l logrus.FieldLogger, event string, characterId uint32, sid skill2.Id, slvl uint32, mobCap uint32, mobIds []uint32) bool {
	if uint32(len(mobIds)) <= mobCap {
		return false
	}
	l.WithFields(logrus.Fields{
		"event":            event,
		"character_id":     characterId,
		"skill_id":         uint32(sid),
		"skill_level":      slvl,
		"mob_count_cap":    mobCap,
		"client_mob_count": len(mobIds),
		"client_mob_ids":   mobIds,
	}).Warn("client_target_count_exceeds_skill_cap")
	return true
}
```

In `common.go`, replace the inline cap block at `:226-237` with the helper (preserving the existing `monster_buff_anomaly_over_cap` event string so no dashboard breaks):

```go
	// FR-4.3 — mobCount cap. Reject the entire cast if the client claims more
	// targets than the skill's WZ definition permits. This runs before any
	// atlas-monsters round-trip; an over-cap cast produces zero emit calls.
	if ExceedsMobCap(l, "monster_buff_anomaly_over_cap", characterId, sid, slvl, mobCap, mobIds) {
		return
	}
```

- [ ] **Step 4: Write the handler**

Create `skill/handler/monstermagnet/monstermagnet.go`:

```go
// Package monstermagnet implements the Hero / Paladin / Dark Knight Monster
// Magnet cast (task-215). The client picks up to the skill's WZ mobCount nearby
// monsters, rolls a per-monster grab result and sends the table on the use-skill
// opcode; the server validates that table, plays the grab effect on every OTHER
// client in the field, wipes each grabbed monster's damage aggro, and hands the
// monster's controller to the caster.
package monstermagnet

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	monstercb "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
)

func init() {
	// Identity, not wire id: registry.go is keyed on skill2.Identity and
	// UseSkill resolves the incoming wire id through
	// constants.For(...).Skill.Resolve before Lookup (task-187). One
	// registration per identity covers every provisioned version.
	//
	// The Handler registry, NOT AttackCastHandler: the magnet arrives on the
	// use-skill opcode rather than an attack packet, and it deals no damage.
	// Registration here also means UseSkill has already charged the WZ mpCon
	// before Apply runs — Apply must not charge it again.
	channelhandler.Register(skill2.HeroMonsterMagnet, Apply)
	channelhandler.Register(skill2.PaladinMonsterMagnet, Apply)
	channelhandler.Register(skill2.DarkKnightMonsterMagnet, Apply)
}

// grabResultSuccess / grabSuccessFlag are the CATCH_MONSTER writer arguments
// for a successful grab. The wire grab result is a BOOLEAN, not an enum: the
// local client computes its ShowCatchEffect selector as `(grabResult == 3)`
// (gms_83 CMob::OnHit @0x668B83 — the three magnet ids at @0x668DB7/DC3/DCA all
// jump to @0x668E14, which does setz al on (arg == 3) then calls
// ShowCatchEffect @0x668E22). So a successful grab maps to 1.
const (
	grabResultSuccess = byte(1)
	grabSuccessFlag   = byte(1)
)

// loadCasterFunc is the caster-load seam tests replace.
var loadCasterFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (character.Model, error) {
	cp := character.NewProcessor(l, ctx)
	return cp.GetById()(characterId)
}

// rectQueryFunc is the mob-selection seam tests replace.
var rectQueryFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]monster.Model, error) {
	return monster.NewProcessor(l, ctx).GetInMapRect(f, x1, y1, x2, y2, limit)
}

// announceCatchFunc is the grab-effect broadcast seam tests replace.
//
// OTHER sessions only, deliberately. The caster's own client already renders
// the effect locally: TryDoingMonsterMagnet calls CMob::AddDamageInfo per
// grabbed mob, which drives CMob::OnHit -> ShowCatchEffect on that client.
// Sending CATCH_MONSTER to the caster too would play the animation twice —
// exactly the double-render task-212 removed from the catch-item path. Remote
// clients never run AddDamageInfo for the magnet, so they DO need the packet.
var announceCatchFunc = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, casterId uint32, monsterId uint32) error {
	return _map.NewProcessor(l, ctx).ForOtherSessionsInMap(
		f, casterId,
		session.Announce(l)(ctx)(wp)(monstercb.CatchMonsterWriter)(
			writer.CatchMonsterBody(monsterId, grabResultSuccess, grabSuccessFlag)),
	)
}

// clearAggroFunc and forceControlFunc are the two monster-command emit seams.
var clearAggroFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32) error {
	return monster.NewProcessor(l, ctx).ClearAggro(f, monsterId)
}

var forceControlFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32, characterId uint32) error {
	return monster.NewProcessor(l, ctx).ForceControl(f, monsterId, characterId)
}

// Apply is the registered Monster Magnet handler. It always returns nil: a
// partial failure must never abort the caller's EnableActions unlock.
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer, f field.Model, characterId uint32,
	info packetmodel.SkillUsageInfo, e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer, f field.Model, characterId uint32,
		info packetmodel.SkillUsageInfo, e effect.Model,
	) error {
		return func(
			wp writer.Producer, f field.Model, characterId uint32,
			info packetmodel.SkillUsageInfo, e effect.Model,
		) error {
			sid := skill2.Id(info.SkillId())
			slvl := uint32(info.SkillLevel())

			// 1. Drop failed grabs (FR-2.5) and released slots. objectId 0 is a
			//    legitimate wire value for a slot the client released mid-cast.
			claimed := make([]uint32, 0, len(info.MagnetGrabs()))
			for _, g := range info.MagnetGrabs() {
				if !g.Grabbed() || g.ObjectId() == 0 {
					continue
				}
				claimed = append(claimed, g.ObjectId())
			}
			if len(claimed) == 0 {
				l.WithFields(logrus.Fields{
					"caster":           characterId,
					"skill_id":         uint32(sid),
					"skill_level":      slvl,
					"client_mob_count": len(info.MagnetGrabs()),
					"grabbed":          0,
				}).Debug("monster_magnet_summary")
				return nil
			}

			// 2. FR-2.2 — over-cap rejects the WHOLE cast.
			if channelhandler.ExceedsMobCap(l, "monster_magnet_anomaly_over_cap", characterId, sid, slvl, e.MobCount(), claimed) {
				return nil
			}

			// 3. FR-2.7 — caster load failure drops the whole cast.
			c, cErr := loadCasterFunc(l, ctx, characterId)
			if cErr != nil {
				l.WithError(cErr).WithFields(logrus.Fields{
					"event":        "monster_magnet_caster_load_failed",
					"character_id": characterId,
					"skill_id":     uint32(sid),
				}).Error("monster_magnet_caster_load_failed")
				return nil
			}

			// 4. Region check. Exactly ONE rect query per cast regardless of how
			//    many monsters were claimed.
			applied := claimed
			var anomaly []uint32
			rect := [4]int16{}
			if e.Range() > 0 {
				facingLeft := (c.Stance() & 1) == 1
				x1, y1, x2, y2 := channelhandler.MagnetRegion(c.X(), c.Y(), facingLeft, e.Range())
				rect = [4]int16{x1, y1, x2, y2}

				mobs, qErr := rectQueryFunc(l, ctx, f, x1, y1, x2, y2, e.MobCount())
				if qErr != nil {
					l.WithError(qErr).WithFields(logrus.Fields{
						"event":        "monster_magnet_rect_query_failed",
						"character_id": characterId,
						"skill_id":     uint32(sid),
						"rect":         rect,
					}).Error("monster_magnet_rect_query_failed")
					return nil
				}
				serverMobIds := make([]uint32, 0, len(mobs))
				for _, m := range mobs {
					serverMobIds = append(serverMobIds, m.UniqueId())
				}

				applied, anomaly = channelhandler.IntersectMobIds(claimed, serverMobIds)

				// FR-2.3 — an out-of-region target is dropped INDIVIDUALLY.
				if len(anomaly) > 0 {
					l.WithFields(logrus.Fields{
						"event":           "monster_magnet_anomaly_out_of_rect",
						"character_id":    characterId,
						"skill_id":        uint32(sid),
						"skill_level":     slvl,
						"rect":            map[string]int16{"x1": x1, "y1": y1, "x2": x2, "y2": y2},
						"mob_count_cap":   e.MobCount(),
						"client_mob_ids":  claimed,
						"server_mob_ids":  serverMobIds,
						"anomaly_mob_ids": anomaly,
					}).Warn("client_targeted_mob_outside_server_rect")
				}
			} else {
				// FR-2.4's fallback, relocated from lt/rb to `range`: no region
				// contract in this tenant's WZ data, so accept the client's list
				// subject to the cap only. Defensive — `range` is present for all
				// three magnet skills at every level in the data read for design
				// section 3.
				l.WithFields(logrus.Fields{
					"skill_id":         uint32(sid),
					"skill_level":      slvl,
					"client_mob_count": len(claimed),
				}).Debug("monster_magnet_no_range_cap_only")
			}

			// 5. Per surviving monster: grab effect, then aggro wipe, then
			//    controller handover. CLEAR_AGGRO must precede FORCE_CONTROL —
			//    both key on the monster id so they land on the same partition in
			//    this order, and reversing them would have the wipe immediately
			//    clear the aggro flag the handover just set.
			grabbed := 0
			for _, monsterId := range applied {
				if aErr := announceCatchFunc(l, ctx, wp, f, characterId, monsterId); aErr != nil {
					l.WithError(aErr).Warnf("Monster Magnet: unable to broadcast the grab effect for monster [%d].", monsterId)
				}
				if cmErr := clearAggroFunc(l, ctx, f, monsterId); cmErr != nil {
					l.WithError(cmErr).Warnf("Monster Magnet: unable to clear aggro for monster [%d].", monsterId)
				}
				if fErr := forceControlFunc(l, ctx, f, monsterId, characterId); fErr != nil {
					l.WithError(fErr).Warnf("Monster Magnet: unable to force control of monster [%d] to character [%d].", monsterId, characterId)
				}
				grabbed++
			}

			l.WithFields(logrus.Fields{
				"caster":              characterId,
				"skill_id":            uint32(sid),
				"skill_level":         slvl,
				"client_mob_count":    len(info.MagnetGrabs()),
				"claimed":             len(claimed),
				"grabbed":             grabbed,
				"out_of_rect_dropped": len(anomaly),
				"rect":                rect,
			}).Debug("monster_magnet_summary")

			return nil
		}
	}
}
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-channel/atlas.com/channel" && go test ./skill/handler/... -v
```

Expected: PASS — the three new geometry/cap tests, all ten `monstermagnet` cases, and every pre-existing `skill/handler` test (the `IntersectMobIds` rename and the `ExceedsMobCap` extraction must leave `common_apply_to_mobs_test.go` green).

- [ ] **Step 6: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add services/atlas-channel/atlas.com/channel/skill/handler/
git commit -m "feat(task-215): Monster Magnet target validation and grab handler

Extracts the mobCount cap check and exports the client/server id intersection
from applyToMobs so both consumers share one policy and one log vocabulary. The
magnet gets its own geometry: Monster Magnet has no lt/rb, so the region is the
AABB of the client's range-derived trapezoid, padded to stand in for the mob
body rect GetInMapRect does not expose."
```

---

### Task 7: Register the handler

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations_magnet_test.go` (create)

**Interfaces:**
- Consumes: `monstermagnet.Apply` and its `init()` (Task 6).
- Produces: all three magnet Identities resolvable via `channelhandler.Lookup`.

The `monstermagnet` package registers itself in `init()`, but nothing imports it — so without this task the handler is compiled and never installed, and a magnet cast falls through `UseSkill`'s dispatcher doing nothing, which is exactly the pre-task-215 behaviour. That silent-failure mode is why this task carries its own test.

- [ ] **Step 1: Write the failing test**

Create `registrations_magnet_test.go`:

```go
package registrations

import (
	channelhandler "atlas-channel/skill/handler"
	"testing"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// TestMonsterMagnetHandlersRegistered guards the blank-import wiring. The
// monstermagnet package registers itself from init(); if nothing imports it the
// handler compiles fine and is simply never installed.
func TestMonsterMagnetHandlersRegistered(t *testing.T) {
	for _, id := range []skill2.Identity{
		skill2.HeroMonsterMagnet,
		skill2.PaladinMonsterMagnet,
		skill2.DarkKnightMonsterMagnet,
	} {
		if _, ok := channelhandler.Lookup(id); !ok {
			t.Fatalf("no Handler registered for identity [%v]", id)
		}
		if _, ok := channelhandler.LookupAttackCast(id); ok {
			t.Fatalf("identity [%v] must NOT be in the AttackCastHandler registry: the magnet arrives on the use-skill opcode and deals no damage", id)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-channel/atlas.com/channel" && go test ./skill/handler/registrations/ -v
```

Expected: FAIL — `no Handler registered for identity [...]` for all three.

- [ ] **Step 3: Add the blank import**

In `registrations.go`, insert in the existing alphabetically-sorted block (between `mprecovery` and `mysticdoor`):

```go
	_ "atlas-channel/skill/handler/monstermagnet" // Hero/Paladin/DarkKnight Monster Magnet — task-215
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-channel/atlas.com/channel" && go test ./skill/handler/registrations/ -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add services/atlas-channel/atlas.com/channel/skill/handler/registrations/
git commit -m "feat(task-215): register the Monster Magnet handler for all three identities"
```

---

### Task 8: Thread the direction byte into the skill-effect broadcast

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/effects.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go:176,178`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/effects_direction_test.go` (create)

**Interfaces:**
- Consumes: `info.Direction()` (Task 1).
- Produces: `AnnounceDirectedSkillUse`, `AnnounceForeignDirectedSkillUse`.

The per-cast self + foreign SKILL_USE broadcast already fires once per cast for every skill (`character_skill_use.go:176,178`), so FR-6.1 and FR-6.3 are already satisfied. The codec already gates a trailing `monsterMagnetLeft` bool on the three magnet ids (`libs/atlas-packet/character/effect_body.go:69,81`). The only gap is that `effects.go` hard-codes `false` and no caller can pass anything else.

`AnnounceBerserkEffect` / `AnnounceForeignBerserkEffect` (`effects.go:47-67`) are the precedent: same problem, same shape.

> **FR-6 is inert on gms_48.** `template_gms_48_1.json` binds **no** `CharacterEffect` writer at all (its only effect writers are `FieldEffect`, `FieldEffectWeather`, `MonsterSpecialEffectBySkill`), so both SKILL_USE broadcasts are already dropped on that version for every skill. That is a pre-existing gap, not one this task creates, and v48 sends no direction byte anyway. Do **not** try to fix it here — adding the route needs the v48 clientbound opcode derived plus a v48 `operations` mode table, which is a separate packet-audit pass.

- [ ] **Step 1: Write the test that pins the codec contract**

Create `effects_direction_test.go`:

```go
package handler

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestSkillUseEffectCarriesMagnetDirection pins that the `left` argument reaches
// the encoder's monsterMagnetLeft field. effect_body.go derives isMonsterMagnet
// from the skill id, so the byte is present only for a magnet cast — a
// non-magnet skill encodes nothing extra regardless of what `left` says.
func TestSkillUseEffectCarriesMagnetDirection(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	opts := map[string]interface{}{}

	magnetLeft := charpkt.CharacterSkillUseEffectBody(
		uint32(skill.HeroMonsterMagnetId), 120, 30, false, false, true)(nil, ctx)(opts)
	magnetRight := charpkt.CharacterSkillUseEffectBody(
		uint32(skill.HeroMonsterMagnetId), 120, 30, false, false, false)(nil, ctx)(opts)

	if len(magnetLeft) != len(magnetRight) {
		t.Fatalf("magnet bodies differ in length (%d vs %d); the direction is one byte, not a length change",
			len(magnetLeft), len(magnetRight))
	}
	if string(magnetLeft) == string(magnetRight) {
		t.Fatal("left=true and left=false encoded identically; the direction byte is not reaching the encoder")
	}

	nonMagnetLeft := charpkt.CharacterSkillUseEffectBody(
		uint32(skill.FighterRageId), 120, 20, false, false, true)(nil, ctx)(opts)
	nonMagnetRight := charpkt.CharacterSkillUseEffectBody(
		uint32(skill.FighterRageId), 120, 20, false, false, false)(nil, ctx)(opts)
	if string(nonMagnetLeft) != string(nonMagnetRight) {
		t.Fatal("a non-magnet skill encoded differently for left=true/false; the gate must be skill-id derived")
	}
}
```

> If `CharacterSkillUseEffectBody` needs an `operations` mode table in `opts` to resolve the effect mode, build one the way an existing `character/clientbound` effect test in this repo already does, rather than inventing a new fixture shape.

- [ ] **Step 2: Run the test**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-channel/atlas.com/channel" && \
  go test ./socket/handler/ -run SkillUseEffectCarriesMagnetDirection -v
```

Expected: **PASS** — this exercises the already-complete codec, and it is here to pin that contract before the announce functions start depending on it. If it FAILS, the codec is not doing what design §1 claims; stop and re-derive before continuing.

- [ ] **Step 3: Add the directed announce functions**

Append to `socket/handler/effects.go`:

```go
// AnnounceDirectedSkillUse is AnnounceSkillUse with the caster's facing bit
// threaded through. The packet encoder writes it as a trailing byte only for
// the three Monster Magnet skill ids (effect_body.go derives that gate from the
// skill id), so passing it for any other skill encodes nothing.
//
// Same shape and rationale as AnnounceBerserkEffect above: the plain
// AnnounceSkillUse keeps its signature so its other call sites are untouched.
func AnnounceDirectedSkillUse(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte, left bool) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte, left bool) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte, left bool) model2.Operator[session.Model] {
			return func(skillId uint32, characterLevel byte, skillLevel byte, left bool) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectWriter)(charpkt.CharacterSkillUseEffectBody(skillId, characterLevel, skillLevel, false, false, left))
			}
		}
	}
}

// AnnounceForeignDirectedSkillUse is the same broadcast targeted at the other
// sessions on the caster's map — the ones that actually need the direction,
// since they do not render the cast locally.
func AnnounceForeignDirectedSkillUse(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, left bool) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, left bool) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, left bool) model2.Operator[session.Model] {
			return func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, left bool) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectForeignWriter)(charpkt.CharacterSkillUseEffectForeignBody(characterId, skillId, characterLevel, skillLevel, false, false, left))
			}
		}
	}
}
```

- [ ] **Step 4: Call the directed variants**

In `character_skill_use.go`, replace lines 176 and 178:

```go
			session.NewProcessor(l, ctx).IfPresentByCharacterId(s.Field().Channel())(s.CharacterId(), AnnounceDirectedSkillUse(l)(ctx)(wp)(sui.SkillId(), c.Level(), sui.SkillLevel(), sui.Direction()))

			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), AnnounceForeignDirectedSkillUse(l)(ctx)(wp)(s.CharacterId(), sui.SkillId(), c.Level(), sui.SkillLevel(), sui.Direction()))
```

- [ ] **Step 5: Verify the plain variants still have callers**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-channel/atlas.com/channel" && \
  grep -rn "AnnounceSkillUse\|AnnounceForeignSkillUse" --include="*.go" . | grep -v "func Announce"
```

Expected: the four pre-existing call sites (`heal`, `healdispel`, `resurrection`, the monster consumer) still reference them — they must be untouched by this task.

- [ ] **Step 6: Run the tests**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-channel/atlas.com/channel" && go test -race ./socket/... && go build ./...
```

Expected: PASS and a clean build.

- [ ] **Step 7: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add services/atlas-channel/atlas.com/channel/socket/handler/
git commit -m "feat(task-215): carry the magnet direction byte into the skill-effect broadcast

The per-cast broadcast and the codec's monsterMagnetLeft gate already existed;
only the argument was missing. Follows the AnnounceBerserkEffect precedent so
the plain announce variants' four other call sites are untouched."
```

---

### Task 9: Full-branch verification

**Files:** none modified (verification only, plus any fixes it surfaces).

- [ ] **Step 1: Run every module's tests, vet, and build**

```bash
cd "$(git rev-parse --show-toplevel)"
for m in libs/atlas-packet \
         services/atlas-channel/atlas.com/channel \
         services/atlas-monsters/atlas.com/monsters; do
  echo "=== $m ==="
  ( cd "$m" && go test -race ./... && go vet ./... && go build ./... ) || echo "FAILED: $m"
done
```

Expected: all three clean. Investigate and fix anything that is not — do not proceed with a known failure.

- [ ] **Step 2: Run the repo-root guards**

```bash
cd "$(git rev-parse --show-toplevel)"
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/skill-job-id-guard.sh
tools/buff-duration-guard.sh
tools/lint.sh              # fix mode first — rewrites files in place
tools/lint.sh --check
```

Expected: every guard exits 0. If `tools/lint.sh` rewrote anything, commit the formatting separately. If `--check` false-fails on the atlas-ui half, run `nvm use 22` and retry — that is a known environment issue, not a code problem.

- [ ] **Step 3: Confirm the packet matrix is untouched**

```bash
cd "$(git rev-parse --show-toplevel)"
go run ./tools/packet-audit matrix --check
git status --short docs/packets/
git diff --stat main -- docs/packets/
```

Expected: `matrix --check` exits 0; both `git` commands print nothing.

- [ ] **Step 4: Confirm the assumed-unchanged surfaces really are unchanged**

```bash
cd "$(git rev-parse --show-toplevel)"
git diff --stat main -- '**/go.mod' '**/go.sum' services/atlas-configurations/seed-data/templates/
```

Expected: empty. A non-empty result means the plan's scope assumption broke — a `go.mod` change requires `docker buildx bake atlas-channel` / `atlas-monsters`, and a template change requires `tools/template-opcode-order-guard.sh`, `tools/template-duplicate-binding-guard.sh` and `tools/template-movement-types-guard.sh`. Run whichever applies and report the deviation.

- [ ] **Step 5: Confirm `CATCH_MONSTER_WITH_ITEM` is not on the magnet path (FR-3.3)**

```bash
cd "$(git rev-parse --show-toplevel)"
grep -rn "CatchMonsterWithItem" services/atlas-channel/atlas.com/channel/skill/ && \
  echo "FAIL: the magnet path must never send CATCH_MONSTER_WITH_ITEM" || \
  echo "OK: no CATCH_MONSTER_WITH_ITEM reference under skill/"
```

Expected: `OK`.

- [ ] **Step 6: Confirm the grab-effect fan-out excludes the caster**

```bash
cd "$(git rev-parse --show-toplevel)"
grep -n "ForOtherSessionsInMap\|ForEachSessionInMap\|ForSessionsInMap" \
  services/atlas-channel/atlas.com/channel/skill/handler/monstermagnet/monstermagnet.go
```

Expected: exactly one match, `ForOtherSessionsInMap`. Any all-sessions variant would double-render the animation on the caster's own screen.

- [ ] **Step 7: Confirm the keydown relay regression path is untouched (design §7.4)**

```bash
cd "$(git rev-parse --show-toplevel)"
git diff --stat main -- \
  services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare.go \
  services/atlas-channel/atlas.com/channel/socket/handler/character_buff_cancel.go \
  libs/atlas-constants/skill/identity.go
```

Expected: empty. The prepare/keyup relay already covers all three magnet identities and must not have been touched.

- [ ] **Step 8: Run the code review**

Invoke `superpowers:requesting-code-review`. Go files changed in three modules and no atlas-ui TypeScript changed, so it should dispatch `plan-adherence-reviewer` and `backend-guidelines-reviewer` (not the frontend reviewer). Pin the reviewer subagents to Sonnet or Haiku, not an expensive model. Findings land in `docs/tasks/task-215-monster-magnet/audit.md`.

Verify the reviewers ran inside this worktree and left no stray edits in the main repo — run `git status --short` in both the worktree and the main checkout.

- [ ] **Step 9: Address the review findings, then re-run Steps 1–3**

- [ ] **Step 10: Commit the audit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add docs/tasks/task-215-monster-magnet/audit.md
git commit -m "docs(task-215): code review findings"
```

---

## Acceptance criteria not covered by automated tests

These are live-client checks. They cannot be automated in this branch; run them before merge on at least the primary test version, and record the results in the PR description.

- [ ] Casting Monster Magnet as Hero (1121001), Paladin (1221001) and Dark Knight (1321001) visibly grabs nearby monsters.
- [ ] Each grabbed monster plays the blue grab animation on every **other** client in the field, exactly once.
- [ ] Each grabbed monster's damage-aggro table is fully wiped.
- [ ] The caster becomes the controller of each grabbed monster and the monsters respond to the caster's client immediately.
- [ ] Remote players see the caster's magnet skill effect, oriented by the direction byte.
- [ ] Monsters the client reports as failed grabs are untouched.
- [ ] The keydown prepare and keyup still relay to remote players for all three magnet skills (regression check on the already-working path).
- [ ] The task-212 catch-item flow still plays exactly one animation and is otherwise unchanged.
- [ ] Casts of other use-skill-opcode skills — a mob-affecting buff, a party buff, Shadow Stars, Resurrection — behave unchanged.
