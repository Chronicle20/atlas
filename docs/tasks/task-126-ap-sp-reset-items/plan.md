# AP/SP Reset Cash Items Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement AP Reset (item 5050000) and SP Reset (items 5050001–5050004) end-to-end: serverbound packet decode, channel pre-validation, a `point_reset` saga (`destroy_asset` → `transfer_ap`/`transfer_sp`), authoritative transfer commands in atlas-character/atlas-skills, and pink-text/enable-actions feedback.

**Architecture:** Saga shape B from design.md §3 — destroy-first with reverse-walk compensation (destroy → re-award, modeled on PetEvolution), full channel pre-validation so validation failures never create a saga, and exactly-one-owner policy tables in atlas-character. New Kafka surface only; no REST endpoints, no schema migration (`hpmp_used` already exists).

**Tech Stack:** Go, Kafka (atlas-kafka), GORM, atlas-saga orchestration, atlas-packet codecs, IDA-verified byte fixtures via tools/packet-audit.

## Global Constraints

- Spec: `docs/tasks/task-126-ap-sp-reset-items/design.md` (authoritative); PRD at `prd.md` in the same folder.
- Item ids: AP Reset = 5050000; SP Reset tier N = 505000N (N = 1..4). Wire sub-body: two int32s read **To then From** (hypothesis until IDA-verified per version — Task 4).
- Server policy constants (design §2.2/§7, fixed, not tenant-configurable): primary-stat floor **4** (source must be ≥ 5), primary-stat cap **32767**, MaxHP/MaxMP cap **30000**.
- Policy tables verbatim from PRD §4.3 (Cosmic parity). Encode as data, never an if/else chain.
- Machine-readable error codes (exact strings, shared by services and channel): `STAT_AT_MINIMUM`, `STAT_AT_MAXIMUM`, `INSUFFICIENT_HPMP_AP_USED`, `POOL_BELOW_JOB_MINIMUM`, `SKILL_AT_ZERO`, `SKILL_AT_CAP`, `WRONG_TIER`, `INVALID_TARGET`.
- The item is destroyed **iff** the transfer step succeeded. Validation failures never consume the item.
- Evan job lines (2200–2218) are rejected for SP Reset (`WRONG_TIER` + warn). gms_v92 is parked (no IDB) — item stays inert there.
- Every impossible-from-a-legit-client rejection logs at warn with character id and offending values.
- `database.ExecuteTransaction` is a known no-op (never begins a transaction — verified `libs/atlas-database/transaction.go:9-18`). Multi-row atomicity in atlas-skills MUST use gorm-native `p.db.Transaction(...)`.
- Test setup uses the project Builder pattern; no `*_testhelpers.go` files.
- Verification gates (CLAUDE.md): `go test -race ./...`, `go vet ./...`, `go build ./...` per changed module; `docker buildx bake` per changed service; `tools/redis-key-guard.sh`; `go run ./tools/packet-audit matrix --check`.
- Commit after every task. Run commands from the worktree root unless a task says otherwise.

---

### Task 1: `job.Advancement` in libs/atlas-constants

**Files:**
- Create: `libs/atlas-constants/job/advancement.go`
- Create: `libs/atlas-constants/job/advancement_test.go`

**Interfaces:**
- Consumes: existing `job.Id` (`uint16`, `constants.go:7`), `job.IsBeginner` (`model.go:65-67`), stage-id constants (`constants.go:1145-1228`).
- Produces: `func Advancement(jobId Id) int` — job-advancement tier 0–4; **-1** for Evan stage lines (2200–2218) and any id that doesn't map to a tier. Tasks 10 and 13 call this.

- [ ] **Step 1: Write the failing test**

```go
package job_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

func TestAdvancement(t *testing.T) {
	cases := []struct {
		name  string
		jobId job.Id
		want  int
	}{
		{"Beginner", job.BeginnerId, 0},
		{"Noblesse", job.NoblesseId, 0},
		{"Legend (Aran beginner)", job.LegendId, 0},
		{"Evan beginner (2001)", job.EvanId, 0},
		{"Warrior", job.Id(100), 1},
		{"Fighter", job.Id(110), 2},
		{"Crusader", job.Id(111), 3},
		{"Hero", job.Id(112), 4},
		{"Page", job.Id(120), 2},
		{"Paladin", job.Id(122), 4},
		{"Spearman", job.Id(130), 2},
		{"Dark Knight", job.Id(132), 4},
		{"Magician", job.Id(200), 1},
		{"FP Wizard", job.Id(210), 2},
		{"IL Arch Mage", job.Id(222), 4},
		{"Bowman", job.Id(300), 1},
		{"Bowmaster", job.Id(312), 4},
		{"Thief", job.Id(400), 1},
		{"Night Lord", job.Id(412), 4},
		{"Pirate", job.Id(500), 1},
		{"Corsair", job.Id(522), 4},
		{"Dawn Warrior 1", job.DawnWarriorStage1Id, 1},
		{"Dawn Warrior 2", job.DawnWarriorStage2Id, 2},
		{"Dawn Warrior 3", job.DawnWarriorStage3Id, 3},
		{"Dawn Warrior 4", job.DawnWarriorStage4Id, 4},
		{"Aran 1", job.AranStage1Id, 1},
		{"Aran 2", job.AranStage2Id, 2},
		{"Aran 3", job.AranStage3Id, 3},
		{"Aran 4", job.AranStage4Id, 4},
		{"Evan stage 1 (excluded)", job.EvanStage1Id, -1},
		{"Evan stage 5 (excluded)", job.EvanStage5Id, -1},
		{"Evan stage 10 (excluded)", job.EvanStage10Id, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := job.Advancement(tc.jobId); got != tc.want {
				t.Errorf("Advancement(%d) = %d, want %d", tc.jobId, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./job/ -run TestAdvancement -v`
Expected: FAIL with `undefined: job.Advancement`

- [ ] **Step 3: Write minimal implementation**

`libs/atlas-constants/job/advancement.go`:

```go
package job

// Advancement returns the job-advancement tier (0-4) for a job id:
// 0 for beginners (Beginner/Noblesse/Legend/Evan-beginner), 1 for a branch
// root (jobId%100 == 0), else 2 + jobId%10. Evan stage lines (2200-2218) do
// not map onto the 4-tier scheme and return -1, as does any id whose derived
// tier falls outside 0-4.
func Advancement(jobId Id) int {
	if jobId >= EvanStage1Id && jobId <= EvanStage10Id {
		return -1
	}
	if IsBeginner(jobId) {
		return 0
	}
	if jobId%100 == 0 {
		return 1
	}
	tier := 2 + int(jobId%10)
	if tier > 4 {
		return -1
	}
	return tier
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-constants && go test ./job/ -run TestAdvancement -v`
Expected: PASS

- [ ] **Step 5: Full module check + commit**

```bash
cd libs/atlas-constants && go test -race ./... && go vet ./...
git add libs/atlas-constants/job/advancement.go libs/atlas-constants/job/advancement_test.go
git commit -m "feat(constants): job.Advancement tier helper (task-126)"
```

---

### Task 2: `skill.IsPointResetExcluded` in libs/atlas-constants

**Files:**
- Create: `libs/atlas-constants/skill/point_reset.go`
- Create: `libs/atlas-constants/skill/point_reset_test.go`

**Interfaces:**
- Consumes: existing `skill.Id` (`uint32`, `constants.go:3`).
- Produces: `func IsPointResetExcluded(skillId Id) bool` — true for skills that may not be an SP Reset source or target (design §4.1 exclusion set). Tasks 10 and 13 call this.

- [ ] **Step 1: Write the failing test**

```go
package skill_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestIsPointResetExcluded(t *testing.T) {
	excluded := []skill.Id{
		21110007, 21110008, 21120009, 21120010, // Aran hidden combo skills
		9001000, 9101008, 9050000, // GM skill range bounds + interior
		8001000, 8001001, // GM skills
		20000014, 20000018, // PQ skill range bounds
		10000013, 20001013, // PQ skills (fixed ids)
		1009, 1010, 1011, 10001009, 20001011, // id%10000000 in 1009-1011
		1020, 20001020, // id%10000000 == 1020
	}
	for _, id := range excluded {
		if !skill.IsPointResetExcluded(id) {
			t.Errorf("IsPointResetExcluded(%d) = false, want true", id)
		}
	}
	included := []skill.Id{
		1001003, // Iron Body (1st job warrior)
		3121004, // Hurricane (4th job bowman)
		2301002, // Heal
		21100000, // Aran non-hidden
		1012, 1008, // just outside the 1009-1011 band
	}
	for _, id := range included {
		if skill.IsPointResetExcluded(id) {
			t.Errorf("IsPointResetExcluded(%d) = true, want false", id)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./skill/ -run TestIsPointResetExcluded -v`
Expected: FAIL with `undefined: skill.IsPointResetExcluded`

- [ ] **Step 3: Write minimal implementation**

`libs/atlas-constants/skill/point_reset.go`:

```go
package skill

// IsPointResetExcluded reports whether skillId may not participate in an SP
// Reset transfer (as source or target): Aran hidden combo skills, GM skills,
// and PQ-granted skills, whose points are not pool-backed. Set per Cosmic's
// AssignSPProcessor.canSPAssign / GameConstants.isPqSkill / isGMSkills gates
// (see docs/tasks/task-126-ap-sp-reset-items/design.md §4.1).
func IsPointResetExcluded(skillId Id) bool {
	switch skillId {
	case Id(21110007), Id(21110008), Id(21120009), Id(21120010): // Aran hidden combo
		return true
	case Id(10000013), Id(20001013): // PQ skills (fixed ids)
		return true
	}
	if skillId >= Id(9001000) && skillId <= Id(9101008) { // GM skills
		return true
	}
	if skillId >= Id(8001000) && skillId <= Id(8001001) { // GM skills
		return true
	}
	if skillId >= Id(20000014) && skillId <= Id(20000018) { // PQ skills
		return true
	}
	rem := uint32(skillId) % 10000000
	if rem >= 1009 && rem <= 1011 { // PQ skills (per-class beginner band)
		return true
	}
	if rem == 1020 { // PQ skill
		return true
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-constants && go test ./skill/ -run TestIsPointResetExcluded -v`
Expected: PASS

- [ ] **Step 5: Full module check + commit**

```bash
cd libs/atlas-constants && go test -race ./... && go vet ./...
git add libs/atlas-constants/skill/point_reset.go libs/atlas-constants/skill/point_reset_test.go
git commit -m "feat(constants): skill.IsPointResetExcluded predicate (task-126)"
```

---

### Task 3: `ItemUsePointReset` codec + round-trip tests

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/item_use_point_reset.go`
- Create: `libs/atlas-packet/cash/serverbound/item_use_point_reset_test.go`

**Interfaces:**
- Consumes: `request.Reader` / `response.NewWriter` from atlas-socket (same as `item_use_field_effect.go:1-50`).
- Produces: `NewItemUsePointReset(updateTimeFirst bool) *ItemUsePointReset` with `To() uint32`, `From() uint32`, `UpdateTime() uint32`, `Operation()`, `String()`, `Encode`, `Decode`. Task 13 decodes with it; Task 4 verifies and may adjust the read order.

- [ ] **Step 1: Write the failing round-trip test**

`libs/atlas-packet/cash/serverbound/item_use_point_reset_test.go` (mirrors `shop_operation_buy_test.go` structure; byte fixtures with `packet-audit:verify` markers are added in Task 4 after IDA verification):

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUsePointResetRoundTrip(t *testing.T) {
	for _, utf := range []bool{true, false} {
		name := "trailingUpdateTime"
		if utf {
			name = "updateTimeFirst"
		}
		t.Run(name, func(t *testing.T) {
			ctx := pt.CreateContext("GMS", 83, 1)
			input := ItemUsePointReset{to: 2048, from: 64, updateTime: 12345, updateTimeFirst: utf}
			output := ItemUsePointReset{updateTimeFirst: utf}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.To() != input.To() {
				t.Errorf("To: got %d want %d", output.To(), input.To())
			}
			if output.From() != input.From() {
				t.Errorf("From: got %d want %d", output.From(), input.From())
			}
			if !utf && output.UpdateTime() != input.UpdateTime() {
				t.Errorf("UpdateTime: got %d want %d", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -run TestItemUsePointResetRoundTrip -v`
Expected: FAIL with `undefined: ItemUsePointReset`

- [ ] **Step 3: Write the codec**

`libs/atlas-packet/cash/serverbound/item_use_point_reset.go` (pattern: `item_use_field_effect.go`):

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUsePointReset is the USE_CASH_ITEM sub-body for AP Reset (5050000) and
// SP Reset (5050001-5050004): two int32s read as To then From. For AP resets
// they are client stat flags; for SP resets they are skill ids. Layouts
// without an updateTime-first prefix carry a trailing updateTime. Read order
// is IDA-verified per version (see byte fixtures in the test file).
type ItemUsePointReset struct {
	to              uint32
	from            uint32
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUsePointReset(updateTimeFirst bool) *ItemUsePointReset {
	return &ItemUsePointReset{updateTimeFirst: updateTimeFirst}
}

func (m ItemUsePointReset) To() uint32         { return m.to }
func (m ItemUsePointReset) From() uint32       { return m.from }
func (m ItemUsePointReset) UpdateTime() uint32 { return m.updateTime }

func (m ItemUsePointReset) Operation() string { return "ItemUsePointReset" }

func (m ItemUsePointReset) String() string {
	return fmt.Sprintf("to [%d] from [%d] updateTime [%d]", m.to, m.from, m.updateTime)
}

func (m ItemUsePointReset) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.to)
		w.WriteInt(m.from)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUsePointReset) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.to = r.ReadUint32()
		m.from = r.ReadUint32()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -run TestItemUsePointResetRoundTrip -v`
Expected: PASS

- [ ] **Step 5: Full module check + commit**

```bash
cd libs/atlas-packet && go test -race ./... && go vet ./...
git add libs/atlas-packet/cash/serverbound/item_use_point_reset.go libs/atlas-packet/cash/serverbound/item_use_point_reset_test.go
git commit -m "feat(packet): ItemUsePointReset serverbound sub-body codec (task-126)"
```

---

### Task 4: IDA verification, byte fixtures, audit reports (per version)

**Files:**
- Modify: `tools/packet-audit/cmd/run.go` (new `candidatesFromFName` case near line 1831)
- Modify: `libs/atlas-packet/cash/serverbound/item_use_point_reset.go` (only if IDA contradicts the read-order hypothesis)
- Modify: `libs/atlas-packet/cash/serverbound/item_use_point_reset_test.go` (exact-bytes fixtures + markers)
- Create: `docs/packets/audits/gms_v83/ItemUsePointReset.{md,json}`, same for `gms_v84`, `gms_v87`, `gms_v95`, `jms_v185`
- Modify: `docs/packets/audits/STATUS.md` / `status.json` (regenerated by the tool)

**Interfaces:**
- Consumes: the Task 3 codec; `docs/packets/audits/VERIFYING_A_PACKET.md` (the governing playbook — follow it exactly); ida-pro-mcp instances or the checked-in IDA exports.
- Produces: verified per-version fixtures; the finalized wire order that Task 13's handler relies on.

**Rules for this task (non-negotiable):**
- The To-then-From order and the trailing-updateTime placement are **hypotheses from Cosmic** until each version's `CWvsContext::SendConsumeCashItemUseRequest` point-reset branch is decompiled and read. If a version contradicts the hypothesis, fix the codec and re-run Task 3's tests before writing that version's fixture.
- Before any IDA read: `mcp__ida-pro__list_instances` and match the **binary name** to the target version (the loaded set rotates). For versions with no live IDB, use the checked-in IDA export per the playbook. If neither resolves the fname: **STOP and report BLOCKED** — never substitute an fname or fake evidence.
- gms_v92: no IDB, no export — **parked**, documented in Task 16's deployment doc. Do not guess.
- Registry opcodes (already confirmed): v83 0x4F, v84 0x4F, v87 0x52, v95 0x55, jms_v185 0x47. Primary fname in the registry is `CItemSpeakerDlg::_SendConsumeCashItemUseRequest`; `CWvsContext::SendConsumeCashItemUseRequest` is an fname_alt.

- [ ] **Step 1: Add the `candidatesFromFName` case**

In `tools/packet-audit/cmd/run.go`, in the "Serverbound CWvsContext senders" block (after the `CWvsContext::SendUpgradeItemUseRequest` case at ~line 1840):

```go
	case "CWvsContext::SendConsumeCashItemUseRequest", "CItemSpeakerDlg::_SendConsumeCashItemUseRequest":
		return []candidate{{name: "ItemUsePointReset", dir: csvpkg.DirServerbound, pkg: "cash"}}
```

Run: `cd tools/packet-audit && go build ./...`
Expected: clean build.

- [ ] **Step 2: Verify gms_v83** — follow `docs/packets/audits/VERIFYING_A_PACKET.md` end-to-end for `ItemUsePointReset` × gms_v83: decompile the point-reset branch of `CWvsContext::SendConsumeCashItemUseRequest` (v83 IDB or export), confirm/adjust the codec read order, add an exact-bytes fixture to `item_use_point_reset_test.go` with a `// packet-audit:verify packet=cash/serverbound/ItemUsePointReset version=gms_v83 ida=0x<addr>` marker (fixture style: `TestShopOperationBuyBytes` in `shop_operation_buy_test.go:60-101` — `testlog.NewNullLogger()`, `hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 83, 1))(nil))` against concatenated hex literals), and produce the audit REPORT pair under `docs/packets/audits/gms_v83/`.

- [ ] **Step 3: Verify gms_v84** — same procedure, version gms_v84, context `pt.CreateContext("GMS", 84, 1)`.

- [ ] **Step 4: Verify gms_v87** — same procedure, version gms_v87, context `pt.CreateContext("GMS", 87, 1)`.

- [ ] **Step 5: Verify gms_v95** — same procedure, version gms_v95, context `pt.CreateContext("GMS", 95, 1)`. Expected from the existing prefix convention: updateTime-first, so no trailing int in the sub-body — confirm in IDA, don't assume.

- [ ] **Step 6: Verify jms_v185** — same procedure, version jms_v185, context per `pt.Variants`' JMS entry (see `shop_operation_buy_test.go:87-101` for the JMS raw-byte fixture style). Note the jms audit dir is `docs/packets/audits/jms_v185` — pass `--audit-dir` explicitly to triage/decompose subcommands (default dir name mismatch silently reports 0/0/0/0).

- [ ] **Step 7: Regenerate matrix + gate**

```bash
cd libs/atlas-packet && go test -race ./cash/... && cd ../..
go run ./tools/packet-audit matrix --check
```
Expected: tests PASS; matrix --check exit 0 with the USE_CASH_ITEM/ItemUsePointReset cells promoted for the five verified versions.

- [ ] **Step 8: Commit**

```bash
git add tools/packet-audit/cmd/run.go libs/atlas-packet/cash/serverbound/ docs/packets/audits/
git commit -m "feat(packet): IDA-verify ItemUsePointReset across gms_v83/84/87/95 + jms_v185 (task-126)"
```

---

### Task 5: libs/atlas-saga — type, actions, payloads, unmarshal

**Files:**
- Modify: `libs/atlas-saga/model.go` (Type block ~lines 9-27; Action block ~lines 62-75)
- Modify: `libs/atlas-saga/payloads.go` (after the RebalanceAP family ~line 223)
- Modify: `libs/atlas-saga/unmarshal.go` (two new cases before the `default` at ~line 498)

**Interfaces:**
- Consumes: existing `world.Id`, `channel.Id` imports in payloads.go; add `job` and `skill` imports from atlas-constants.
- Produces (used by Tasks 11, 13, 14, 15):
  - `PointReset Type = "point_reset"`
  - `TransferAP Action = "transfer_ap"`, `TransferSP Action = "transfer_sp"`
  - `TransferAPPayload{CharacterId uint32; WorldId world.Id; ChannelId channel.Id; From string; To string}`
  - `TransferSPPayload{CharacterId uint32; WorldId world.Id; ChannelId channel.Id; JobId job.Id; FromSkillId skill.Id; ToSkillId skill.Id; ItemTier byte; TargetMaxLevel byte}`

- [ ] **Step 1: Add the Type and Action constants**

In `model.go`, append to the Type const block:

```go
	PointReset           Type = "point_reset"
```

In the Action const block, in the character-state group after `EvolvePet`:

```go
	TransferAP             Action = "transfer_ap"
	TransferSP             Action = "transfer_sp"
```

- [ ] **Step 2: Add the payload structs**

In `payloads.go` after the `RebalanceAPPayload` family (add `job`/`skill` imports from `github.com/Chronicle20/atlas/libs/atlas-constants/...`):

```go
// TransferAPPayload represents the payload for transfer_ap (AP Reset,
// item 5050000): move one already-spent ability point From -> To. From/To
// are validated ability enums (STRENGTH/DEXTERITY/INTELLIGENCE/LUCK/HP/MP),
// never raw client stat flags.
type TransferAPPayload struct {
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	From        string     `json:"from"`
	To          string     `json:"to"`
}

// TransferSPPayload represents the payload for transfer_sp (SP Reset,
// items 5050001-5050004): move one skill point FromSkillId -> ToSkillId.
// JobId, ItemTier, and TargetMaxLevel ride along for authoritative
// re-validation in atlas-skills (trusted server-side caller — atlas-channel).
type TransferSPPayload struct {
	CharacterId    uint32     `json:"characterId"`
	WorldId        world.Id   `json:"worldId"`
	ChannelId      channel.Id `json:"channelId"`
	JobId          job.Id     `json:"jobId"`
	FromSkillId    skill.Id   `json:"fromSkillId"`
	ToSkillId      skill.Id   `json:"toSkillId"`
	ItemTier       byte       `json:"itemTier"`
	TargetMaxLevel byte       `json:"targetMaxLevel"`
}
```

- [ ] **Step 3: Add the unmarshal cases**

In `unmarshal.go`, before the `default` case (copy the exact shape of the `EvolvePet` case at lines 186-191):

```go
	case TransferAP:
		var payload TransferAPPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case TransferSP:
		var payload TransferSPPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
```

- [ ] **Step 4: Run the module tests**

Run: `cd libs/atlas-saga && go test -race ./... && go vet ./...`
Expected: PASS (if the lib has an unmarshal round-trip test pattern, add the two new actions to it in the same style as existing entries).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-saga/
git commit -m "feat(saga): point_reset type, transfer_ap/transfer_sp actions + payloads (task-126)"
```

---

### Task 6: atlas-character — `Build()` hpMpUsed regression fix

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/model.go` (Build() at lines 400-430)
- Create: `services/atlas-character/atlas.com/character/character/model_test.go`

**Interfaces:**
- Consumes: existing `NewModelBuilder()`, `SetHpMpUsed(int)` (model.go:432-435), `CloneModel` (model.go:237-268), `HpMpUsed()` (model.go:180-182).
- Produces: `Build()` that preserves `hpMpUsed`. Task 8's FR-6 gate depends on this.

- [ ] **Step 1: Write the failing regression test**

`model_test.go` (package `character_test`):

```go
package character_test

import (
	"testing"

	"atlas-character/character"
)

func TestBuildPreservesHpMpUsed(t *testing.T) {
	m := character.NewModelBuilder().SetName("Atlas").SetHpMpUsed(7).Build()
	if m.HpMpUsed() != 7 {
		t.Fatalf("Build() dropped hpMpUsed: got %d, want 7", m.HpMpUsed())
	}
}

func TestCloneBuildRoundTripPreservesHpMpUsed(t *testing.T) {
	orig := character.NewModelBuilder().SetName("Atlas").SetHpMpUsed(3).Build()
	clone := character.CloneModel(orig).Build()
	if clone.HpMpUsed() != 3 {
		t.Fatalf("CloneModel().Build() dropped hpMpUsed: got %d, want 3", clone.HpMpUsed())
	}
}
```

Note: adjust the builder call chain to satisfy any required-field validation `Build()` enforces (mirror the minimal chain used in `processor_test.go:53`). The import path for the character package must match the module name used by the existing test files (check the imports of `character/processor_test.go` and copy them).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run TestBuildPreservesHpMpUsed -v`
Expected: FAIL with `got 0, want 7`

- [ ] **Step 3: Fix Build()**

In `model.go` `Build()` (lines 400-430), add one line to the returned `Model` literal, next to `meso`:

```go
		meso:               c.meso,
		hpMpUsed:           c.hpMpUsed,
		skills:             c.skills,
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run "TestBuildPreservesHpMpUsed|TestCloneBuildRoundTripPreservesHpMpUsed" -v`
Expected: PASS

- [ ] **Step 5: Full module check + commit**

```bash
cd services/atlas-character/atlas.com/character && go test -race ./... && go vet ./...
git add services/atlas-character/
git commit -m "fix(character): Build() preserves hpMpUsed (task-126)"
```

---

### Task 7: atlas-character — point-reset policy tables

**Files:**
- Create: `services/atlas-character/atlas.com/character/character/point_reset.go`
- Create: `services/atlas-character/atlas.com/character/character/point_reset_test.go`

**Interfaces:**
- Consumes: `job.Id`, `job.Is` from `libs/atlas-constants/job`.
- Produces (Task 8 calls all of these):
  - `pointResetPrimaryFloor uint16 = 4`, `pointResetPrimaryCap uint16 = 32767`, `pointResetPoolCap uint16 = 30000`
  - `pointResetPolicy{takeHp, takeMp, gainHp, gainMp uint16}` and `pointResetPolicyFor(jobId job.Id) pointResetPolicy`
  - `pointResetMinHp(jobId job.Id, level byte) int`, `pointResetMinMp(jobId job.Id, level byte) int` (int because offsets can be negative)

- [ ] **Step 1: Write the failing table-driven tests**

`point_reset_test.go` (package `character` — internal test, these are unexported symbols):

```go
package character

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

func TestPointResetPolicyFor(t *testing.T) {
	cases := []struct {
		name  string
		jobId job.Id
		want  pointResetPolicy
	}{
		{"Hero (warrior line)", job.Id(112), pointResetPolicy{takeHp: 54, takeMp: 4, gainHp: 20, gainMp: 2}},
		{"Dawn Warrior 3", job.DawnWarriorStage3Id, pointResetPolicy{takeHp: 54, takeMp: 4, gainHp: 20, gainMp: 2}},
		{"Aran 4", job.AranStage4Id, pointResetPolicy{takeHp: 54, takeMp: 4, gainHp: 20, gainMp: 2}},
		{"FP Arch Mage", job.Id(212), pointResetPolicy{takeHp: 10, takeMp: 31, gainHp: 6, gainMp: 18}},
		{"Blaze Wizard 2", job.BlazeWizardStage2Id, pointResetPolicy{takeHp: 10, takeMp: 31, gainHp: 6, gainMp: 18}},
		{"Bowmaster", job.Id(312), pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
		{"Wind Archer 1", job.WindArcherStage1Id, pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
		{"Night Lord", job.Id(412), pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
		{"Night Walker 2", job.NightWalkerStage2Id, pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
		{"Corsair", job.Id(522), pointResetPolicy{takeHp: 42, takeMp: 16, gainHp: 18, gainMp: 14}},
		{"Thunder Breaker 1", job.ThunderBreakerStage1Id, pointResetPolicy{takeHp: 42, takeMp: 16, gainHp: 18, gainMp: 14}},
		{"Beginner", job.BeginnerId, pointResetPolicy{takeHp: 12, takeMp: 8, gainHp: 8, gainMp: 6}},
		{"Noblesse", job.NoblesseId, pointResetPolicy{takeHp: 12, takeMp: 8, gainHp: 8, gainMp: 6}},
		{"Legend (Aran beginner)", job.LegendId, pointResetPolicy{takeHp: 12, takeMp: 8, gainHp: 8, gainMp: 6}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pointResetPolicyFor(tc.jobId); got != tc.want {
				t.Errorf("pointResetPolicyFor(%d) = %+v, want %+v", tc.jobId, got, tc.want)
			}
		})
	}
}

func TestPointResetMinPools(t *testing.T) {
	const lvl = byte(30) // representative level; expectations are mult*30+off
	cases := []struct {
		name           string
		jobId          job.Id
		wantHp, wantMp int
	}{
		{"Warrior base", job.Id(100), 24*30 + 118, 4*30 + 55},
		{"Fighter line", job.Id(111), 24*30 + 418, 4*30 + 55},
		{"Page line", job.Id(121), 24*30 + 118, 4*30 + 155},
		{"Spearman line", job.Id(131), 24*30 + 118, 4*30 + 155},
		{"Dawn Warrior 1", job.DawnWarriorStage1Id, 24*30 + 118, 4*30 + 55},
		{"Dawn Warrior 2", job.DawnWarriorStage2Id, 24*30 + 418, 4*30 + 55},
		{"Aran 1", job.AranStage1Id, 24*30 + 118, 4*30 + 55},
		{"Aran 3", job.AranStage3Id, 24*30 + 418, 4*30 + 55},
		{"Magician base", job.Id(200), 10*30 + 54, 22*30 - 1},
		{"FP Wizard (2nd job)", job.Id(210), 10*30 + 54, 22*30 + 449},
		{"Blaze Wizard 1", job.BlazeWizardStage1Id, 10*30 + 54, 22*30 - 1},
		{"Blaze Wizard 2", job.BlazeWizardStage2Id, 10*30 + 54, 22*30 + 449},
		{"Bowman base", job.Id(300), 20*30 + 58, 14*30 - 15},
		{"Hunter line", job.Id(311), 20*30 + 358, 14*30 + 135},
		{"Thief base", job.Id(400), 20*30 + 58, 14*30 - 15},
		{"Bandit line", job.Id(422), 20*30 + 358, 14*30 + 135},
		{"Wind Archer 1", job.WindArcherStage1Id, 20*30 + 58, 14*30 - 15},
		{"Night Walker 2", job.NightWalkerStage2Id, 20*30 + 358, 14*30 + 135},
		{"Pirate base", job.Id(500), 22*30 + 38, 18*30 - 55},
		{"Brawler line", job.Id(512), 22*30 + 338, 18*30 + 95},
		{"Gunslinger line", job.Id(520), 22*30 + 338, 18*30 + 95},
		{"Thunder Breaker 1", job.ThunderBreakerStage1Id, 22*30 + 38, 18*30 - 55},
		{"Thunder Breaker 2", job.ThunderBreakerStage2Id, 22*30 + 338, 18*30 + 95},
		{"Beginner", job.BeginnerId, 12*30 + 38, 10*30 - 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pointResetMinHp(tc.jobId, lvl); got != tc.wantHp {
				t.Errorf("pointResetMinHp(%d, %d) = %d, want %d", tc.jobId, lvl, got, tc.wantHp)
			}
			if got := pointResetMinMp(tc.jobId, lvl); got != tc.wantMp {
				t.Errorf("pointResetMinMp(%d, %d) = %d, want %d", tc.jobId, lvl, got, tc.wantMp)
			}
		})
	}
}
```

Note: if constant names like `job.WindArcherStage1Id`, `job.BlazeWizardStage2Id`, `job.NightWalkerStage2Id`, `job.ThunderBreakerStage1Id/2Id` differ in `libs/atlas-constants/job/constants.go` (lines 1145-1228), use the actual names — the numeric ids are 1300, 1210, 1410, 1500, 1510.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run "TestPointResetPolicyFor|TestPointResetMinPools" -v`
Expected: FAIL with undefined symbols.

- [ ] **Step 3: Write the tables**

`character/point_reset.go`:

```go
package character

import "github.com/Chronicle20/atlas/libs/atlas-constants/job"

// AP Reset (item 5050000) server policy. Values verbatim from PRD §4.3
// (Cosmic AssignAPProcessor under default config). Fixed reference-config
// parity — not tenant-configurable (design §10).
const (
	pointResetPrimaryFloor = uint16(4)     // post-swap floor; source must be >= 5
	pointResetPrimaryCap   = uint16(32767) // Cosmic MAX_AP
	pointResetPoolCap      = uint16(30000) // Cosmic assignHP/assignMP reject bound
)

type pointResetPolicy struct {
	takeHp uint16 // MaxHP loss when resetting OUT of HP
	takeMp uint16 // MaxMP loss when resetting OUT of MP
	gainHp uint16 // MaxHP gain when resetting INTO HP (deterministic AP-reset path)
	gainMp uint16 // MaxMP gain when resetting INTO MP
}

// Branch rows use job.Is semantics against branch-root reference ids; first
// match wins, default (Beginner/Noblesse/Legend) last. Explorer roots are the
// raw branch ids: 100 warrior, 200 magician, 300 bowman, 400 thief, 500 pirate.
var pointResetPolicyRows = []struct {
	refs   []job.Id
	policy pointResetPolicy
}{
	{refs: []job.Id{job.Id(100), job.DawnWarriorStage1Id, job.AranStage1Id}, policy: pointResetPolicy{takeHp: 54, takeMp: 4, gainHp: 20, gainMp: 2}},
	{refs: []job.Id{job.Id(200), job.BlazeWizardStage1Id}, policy: pointResetPolicy{takeHp: 10, takeMp: 31, gainHp: 6, gainMp: 18}},
	{refs: []job.Id{job.Id(300), job.WindArcherStage1Id}, policy: pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
	{refs: []job.Id{job.Id(400), job.NightWalkerStage1Id}, policy: pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
	{refs: []job.Id{job.Id(500), job.ThunderBreakerStage1Id}, policy: pointResetPolicy{takeHp: 42, takeMp: 16, gainHp: 18, gainMp: 14}},
}

var pointResetDefaultPolicy = pointResetPolicy{takeHp: 12, takeMp: 8, gainHp: 8, gainMp: 6}

func pointResetPolicyFor(jobId job.Id) pointResetPolicy {
	for _, row := range pointResetPolicyRows {
		for _, ref := range row.refs {
			if job.Is(jobId, ref) {
				return row.policy
			}
		}
	}
	return pointResetDefaultPolicy
}

// Minimum pool after a reset-out: mult*level + off (PRD §4.3 min table).
// Rows are ordered narrowest-first because job.Is on a branch root also
// matches its sub-lines. Offsets can be negative; callers compare as int.
type poolMinRow struct {
	refs []job.Id
	mult int
	off  int
}

var pointResetMinHpRows = []poolMinRow{
	{refs: []job.Id{job.Id(110), job.DawnWarriorStage2Id, job.AranStage2Id}, mult: 24, off: 418},               // Fighter-line, DW2+, Aran2+
	{refs: []job.Id{job.Id(100), job.DawnWarriorStage1Id, job.AranStage1Id}, mult: 24, off: 118},               // rest of the warrior branch (incl. Page/Spearman lines)
	{refs: []job.Id{job.Id(200), job.BlazeWizardStage1Id}, mult: 10, off: 54},                                  // Magician-line, Blaze Wizard
	{refs: []job.Id{job.Id(310), job.Id(320), job.Id(410), job.Id(420), job.WindArcherStage2Id, job.NightWalkerStage2Id}, mult: 20, off: 358}, // 2nd-job+ bowman/thief lines
	{refs: []job.Id{job.Id(300), job.Id(400), job.WindArcherStage1Id, job.NightWalkerStage1Id}, mult: 20, off: 58},                            // bowman/thief base
	{refs: []job.Id{job.Id(510), job.Id(520), job.ThunderBreakerStage2Id}, mult: 22, off: 338},                 // Brawler/Gunslinger lines, TB2+
	{refs: []job.Id{job.Id(500), job.ThunderBreakerStage1Id}, mult: 22, off: 38},                               // Pirate base, TB1
}

var pointResetMinMpRows = []poolMinRow{
	{refs: []job.Id{job.Id(120), job.Id(130)}, mult: 4, off: 155},                                              // Page-/Spearman-line
	{refs: []job.Id{job.Id(100), job.DawnWarriorStage1Id, job.AranStage1Id}, mult: 4, off: 55},                 // Warrior, Fighter-line, DW, Aran
	{refs: []job.Id{job.Id(210), job.Id(220), job.Id(230), job.BlazeWizardStage2Id}, mult: 22, off: 449},       // Magician 2nd job+
	{refs: []job.Id{job.Id(200), job.BlazeWizardStage1Id}, mult: 22, off: -1},                                  // Magician base, BW1
	{refs: []job.Id{job.Id(310), job.Id(320), job.Id(410), job.Id(420), job.WindArcherStage2Id, job.NightWalkerStage2Id}, mult: 14, off: 135}, // bowman/thief 2nd job+
	{refs: []job.Id{job.Id(300), job.Id(400), job.WindArcherStage1Id, job.NightWalkerStage1Id}, mult: 14, off: -15},                           // bowman/thief base
	{refs: []job.Id{job.Id(510), job.Id(520), job.ThunderBreakerStage2Id}, mult: 18, off: 95},                  // Brawler/Gunslinger lines, TB2+
	{refs: []job.Id{job.Id(500), job.ThunderBreakerStage1Id}, mult: 18, off: -55},                              // Pirate base, TB1
}

func resolvePoolMin(rows []poolMinRow, defaultMult int, defaultOff int, jobId job.Id, level byte) int {
	for _, row := range rows {
		for _, ref := range row.refs {
			if job.Is(jobId, ref) {
				return row.mult*int(level) + row.off
			}
		}
	}
	return defaultMult*int(level) + defaultOff
}

func pointResetMinHp(jobId job.Id, level byte) int {
	return resolvePoolMin(pointResetMinHpRows, 12, 38, jobId, level) // default: Beginner/Noblesse
}

func pointResetMinMp(jobId job.Id, level byte) int {
	return resolvePoolMin(pointResetMinMpRows, 10, -5, jobId, level) // default: Beginner/Noblesse
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run "TestPointResetPolicyFor|TestPointResetMinPools" -v`
Expected: PASS

- [ ] **Step 5: Full module check + commit**

```bash
cd services/atlas-character/atlas.com/character && go test -race ./... && go vet ./...
git add services/atlas-character/
git commit -m "feat(character): point-reset job policy tables (task-126)"
```

---

### Task 8: atlas-character — TRANSFER_AP command

**Files:**
- Modify: `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go` (command const + body; error consts + body)
- Modify: `services/atlas-character/atlas.com/character/character/producer.go` (error provider)
- Modify: `services/atlas-character/atlas.com/character/character/processor.go` (interface + `TransferAP`/`TransferAPAndEmit`)
- Modify: `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go` (handler + registration)
- Create: `services/atlas-character/atlas.com/character/character/transfer_ap_test.go`

**Interfaces:**
- Consumes: Task 6's fixed `Build()`; Task 7's `pointResetPolicyFor`/`pointResetMinHp`/`pointResetMinMp`/constants; existing `CommandDistributeApAbility*` consts (processor.go:44-51), `dynamicUpdate` + `SetStrength/SetDexterity/SetIntelligence/SetLuck/SetMaxHp/SetMaxMp/SetHealth/SetMana/SetHpMpUsed` (administrator.go), `statChangedProvider` (producer.go:249-264), `StatusEventTypeError` (kafka.go:236).
- Produces (Task 11's orchestrator emits this command; Task 11's error handler consumes the error event):
  - `CommandTransferAP = "TRANSFER_AP"`, `TransferAPCommandBody{ChannelId channel.Id; From string; To string}`
  - Error type consts: `StatusEventErrorTypeStatAtMinimum = "STAT_AT_MINIMUM"`, `StatusEventErrorTypeStatAtMaximum = "STAT_AT_MAXIMUM"`, `StatusEventErrorTypeInsufficientHpMpApUsed = "INSUFFICIENT_HPMP_AP_USED"`, `StatusEventErrorTypePoolBelowJobMinimum = "POOL_BELOW_JOB_MINIMUM"`, `StatusEventErrorTypeApTransferInvalidTarget = "INVALID_TARGET"`
  - `StatusEventApTransferErrorBody{Error string; Detail string}` on the character status topic with `Type: ERROR`
  - Processor: `TransferAP(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, from string, to string) error` + `TransferAPAndEmit(...)` same params

- [ ] **Step 1: Add message types**

In `kafka/message/character/kafka.go`, next to `CommandRebalanceAP` (line 32):

```go
	CommandTransferAP = "TRANSFER_AP"
```

Next to `RebalanceAPCommandBody` (line 194):

```go
// TransferAPCommandBody moves one already-spent AP From -> To (AP Reset item
// 5050000). From/To are CommandDistributeApAbility* enum strings.
type TransferAPCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	From      string     `json:"from"`
	To        string     `json:"to"`
}
```

Next to `StatusEventErrorTypeNotEnoughMeso` (line 237):

```go
	StatusEventErrorTypeStatAtMinimum           = "STAT_AT_MINIMUM"
	StatusEventErrorTypeStatAtMaximum           = "STAT_AT_MAXIMUM"
	StatusEventErrorTypeInsufficientHpMpApUsed  = "INSUFFICIENT_HPMP_AP_USED"
	StatusEventErrorTypePoolBelowJobMinimum     = "POOL_BELOW_JOB_MINIMUM"
	StatusEventErrorTypeApTransferInvalidTarget = "INVALID_TARGET"
```

Next to `StatusEventMesoErrorBody` (line 330):

```go
// StatusEventApTransferErrorBody reports a rejected TRANSFER_AP. Error is one
// of the StatusEventErrorType* point-reset constants; Detail names the
// offending stat (STR/DEX/INT/LUK/HP/MP) where applicable.
type StatusEventApTransferErrorBody struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}
```

- [ ] **Step 2: Add the error provider**

In `character/producer.go`, next to `notEnoughMesoErrorStatusEventProvider` (line 203):

```go
func apTransferErrorStatusEventProvider(transactionId uuid.UUID, characterId uint32, worldId world.Id, errorType string, detail string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.StatusEventApTransferErrorBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       worldId,
		Type:          character2.StatusEventTypeError,
		Body: character2.StatusEventApTransferErrorBody{
			Error:  errorType,
			Detail: detail,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 3: Write the failing processor tests**

`character/transfer_ap_test.go` (package `character_test`, using the shared helpers from `processor_test.go:20-48` — `testDatabase(t)`, `testTenant()` context, `testLogger()`, Builder-pattern character creation, and `message.NewBuffer()` to invoke the buffered method without Kafka). Cover this matrix (one subtest each):

1. STR→DEX success: STR 10 → 9, DEX 4 → 5; `STAT_CHANGED` message buffered with `stat.TypeStrength` and `stat.TypeDexterity`.
2. STR at 4 → rejected `STAT_AT_MINIMUM` detail `STRENGTH`; nothing mutated; error event buffered; returns nil.
3. Target DEX at 32767 → rejected `STAT_AT_MAXIMUM` detail `DEXTERITY`; nothing mutated.
4. HP→STR with `hpMpUsed` 0 → rejected `INSUFFICIENT_HPMP_AP_USED`; nothing mutated.
5. HP→STR success (warrior job 100, level 30, MaxHp 2000, Hp 2000, hpMpUsed 2): MaxHp 2000 → 1946 (−54), Hp 2000 → 1946, hpMpUsed 2 → 1, STR +1.
6. HP→STR where MaxHp−54 < `pointResetMinHp(100, level)` → rejected `POOL_BELOW_JOB_MINIMUM` detail `HP`; nothing mutated.
7. STR→HP success (warrior): MaxHp +20 (gain table), hpMpUsed +1, STR −1; MaxHp at 30000 → rejected `STAT_AT_MAXIMUM` detail `HP`.
8. HP→MP success: MaxHp −54, Hp −54 (floored at 1), MaxMp +2, hpMpUsed net 0 (−1 +1).
9. From==To STR→STR with STR 10: processed, net value unchanged (validations still ran); STAT_CHANGED emitted.
10. From==To HP→HP (warrior, hpMpUsed ≥ 1, pool comfortably above minimum): MaxHp net −34 (−54 +20), hpMpUsed net 0.
11. Invalid ability string (e.g. `"FAME"`) → rejected `INVALID_TARGET`; nothing mutated.
12. Current HP floor: Hp 30, take 54 → Hp floored at 1 (not underflowed).

Write the test bodies concretely — create the character with the needed stats via the existing Builder + `Create` + `dynamicUpdate`-style setters or the processor's update paths (mirror how `rebalance_test.go` arranges stats), call `processor.TransferAP(mb)(uuid.New(), id, channel.NewModel(0, 1), from, to)`, then re-read via `GetById` and assert. Inspect the buffer's messages for topic + event type where the case requires it (mirror how existing tests assert buffered messages, e.g. in `producer_test.go` / `rebalance_test.go`).

- [ ] **Step 4: Run tests to verify they fail**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run TestTransferAP -v`
Expected: FAIL with `undefined` / missing method errors.

- [ ] **Step 5: Implement the processor method**

In `character/processor.go` interface block (next to the RebalanceAP declarations, lines 120-121):

```go
	TransferAPAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, from string, to string) error
	TransferAP(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, from string, to string) error
```

Implementation (mirrors `RebalanceAP` at processor.go:1872-1921 — same transaction/load idiom, validate-then-apply so a target-side failure never leaks the source decrement):

```go
func (p *ProcessorImpl) TransferAPAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, from string, to string) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.TransferAP(buf)(transactionId, characterId, channel, from, to)
	})
}

// transferApRejection is a non-nil sentinel carrying the machine-readable
// error code + detail for a rejected transfer; it is emitted as an ERROR
// status event, not returned as a Go error.
type transferApRejection struct {
	code   string
	detail string
}

func (p *ProcessorImpl) TransferAP(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, channel channel.Model, from string, to string) error {
	return func(transactionId uuid.UUID, characterId uint32, channel channel.Model, from string, to string) error {
		var rejection *transferApRejection
		var stats []stat.Type
		values := map[string]interface{}{}

		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				return err
			}
			policy := pointResetPolicyFor(c.JobId())

			// Running values: source applied first, then target validated
			// against the post-source state (handles From==To naturally).
			newStr, newDex, newInt, newLuk := c.Strength(), c.Dexterity(), c.Intelligence(), c.Luck()
			newMaxHp, newMaxMp := c.MaxHp(), c.MaxMp()
			newHp, newMp := c.Hp(), c.Mp()
			newHpMpUsed := c.HpMpUsed()

			primary := func(ability string) *uint16 {
				switch ability {
				case CommandDistributeApAbilityStrength:
					return &newStr
				case CommandDistributeApAbilityDexterity:
					return &newDex
				case CommandDistributeApAbilityIntelligence:
					return &newInt
				case CommandDistributeApAbilityLuck:
					return &newLuk
				}
				return nil
			}

			// Source arm.
			switch from {
			case CommandDistributeApAbilityStrength, CommandDistributeApAbilityDexterity,
				CommandDistributeApAbilityIntelligence, CommandDistributeApAbilityLuck:
				src := primary(from)
				if *src < pointResetPrimaryFloor+1 {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypeStatAtMinimum, detail: from}
					return nil
				}
				*src = *src - 1
			case CommandDistributeApAbilityHp:
				if newHpMpUsed < 1 {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypeInsufficientHpMpApUsed, detail: from}
					return nil
				}
				if int(newMaxHp)-int(policy.takeHp) < pointResetMinHp(c.JobId(), c.Level()) {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypePoolBelowJobMinimum, detail: from}
					return nil
				}
				newMaxHp -= policy.takeHp
				if int(newHp)-int(policy.takeHp) < 1 {
					newHp = 1
				} else {
					newHp -= policy.takeHp
				}
				newHpMpUsed--
			case CommandDistributeApAbilityMp:
				if newHpMpUsed < 1 {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypeInsufficientHpMpApUsed, detail: from}
					return nil
				}
				if int(newMaxMp)-int(policy.takeMp) < pointResetMinMp(c.JobId(), c.Level()) {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypePoolBelowJobMinimum, detail: from}
					return nil
				}
				newMaxMp -= policy.takeMp
				if int(newMp)-int(policy.takeMp) < 0 {
					newMp = 0
				} else {
					newMp -= policy.takeMp
				}
				newHpMpUsed--
			default:
				rejection = &transferApRejection{code: character2.StatusEventErrorTypeApTransferInvalidTarget, detail: from}
				return nil
			}

			// Target arm (validated against post-source running values).
			switch to {
			case CommandDistributeApAbilityStrength, CommandDistributeApAbilityDexterity,
				CommandDistributeApAbilityIntelligence, CommandDistributeApAbilityLuck:
				dst := primary(to)
				if *dst+1 > pointResetPrimaryCap {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypeStatAtMaximum, detail: to}
					return nil
				}
				*dst = *dst + 1
			case CommandDistributeApAbilityHp:
				if newMaxHp >= pointResetPoolCap {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypeStatAtMaximum, detail: to}
					return nil
				}
				newMaxHp += policy.gainHp
				if newMaxHp > pointResetPoolCap {
					newMaxHp = pointResetPoolCap
				}
				newHpMpUsed++
			case CommandDistributeApAbilityMp:
				if newMaxMp >= pointResetPoolCap {
					rejection = &transferApRejection{code: character2.StatusEventErrorTypeStatAtMaximum, detail: to}
					return nil
				}
				newMaxMp += policy.gainMp
				if newMaxMp > pointResetPoolCap {
					newMaxMp = pointResetPoolCap
				}
				newHpMpUsed++
			default:
				rejection = &transferApRejection{code: character2.StatusEventErrorTypeApTransferInvalidTarget, detail: to}
				return nil
			}

			// Apply everything in one dynamicUpdate. remainingAp untouched (FR-11).
			mods := make([]EntityUpdateFunction, 0, 8)
			if newStr != c.Strength() {
				mods = append(mods, SetStrength(newStr))
				stats = append(stats, stat.TypeStrength)
				values["strength"] = newStr
			}
			if newDex != c.Dexterity() {
				mods = append(mods, SetDexterity(newDex))
				stats = append(stats, stat.TypeDexterity)
				values["dexterity"] = newDex
			}
			if newInt != c.Intelligence() {
				mods = append(mods, SetIntelligence(newInt))
				stats = append(stats, stat.TypeIntelligence)
				values["intelligence"] = newInt
			}
			if newLuk != c.Luck() {
				mods = append(mods, SetLuck(newLuk))
				stats = append(stats, stat.TypeLuck)
				values["luck"] = newLuk
			}
			if newMaxHp != c.MaxHp() {
				mods = append(mods, SetMaxHp(newMaxHp))
				stats = append(stats, stat.TypeMaxHp)
				values["max_hp"] = newMaxHp
			}
			if newMaxMp != c.MaxMp() {
				mods = append(mods, SetMaxMp(newMaxMp))
				stats = append(stats, stat.TypeMaxMp)
				values["max_mp"] = newMaxMp
			}
			if newHp != c.Hp() {
				mods = append(mods, SetHealth(newHp))
				stats = append(stats, stat.TypeHp)
				values["hp"] = newHp
			}
			if newMp != c.Mp() {
				mods = append(mods, SetMana(newMp))
				stats = append(stats, stat.TypeMp)
				values["mp"] = newMp
			}
			if newHpMpUsed != c.HpMpUsed() {
				mods = append(mods, SetHpMpUsed(newHpMpUsed))
			}
			if len(mods) == 0 {
				// From==To primary transfer nets to zero; still answer the
				// client so it unlocks (empty STAT_CHANGED, exclRequestSent).
				return nil
			}
			return dynamicUpdate(tx)(mods...)(c)
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not transfer AP for character [%d].", characterId)
			return txErr
		}

		if rejection != nil {
			p.l.WithFields(logrus.Fields{"character_id": characterId, "from": from, "to": to}).
				Warnf("Rejected AP transfer: [%s] detail [%s].", rejection.code, rejection.detail)
			_ = mb.Put(character2.EnvEventTopicCharacterStatus, apTransferErrorStatusEventProvider(transactionId, characterId, channel.WorldId(), rejection.code, rejection.detail))
			return nil
		}

		_ = mb.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel, characterId, stats, values))
		return nil
	}
}
```

Adjust identifier details to the file's actual idioms (e.g. `stat` import alias, `logrus` import). For the From==To zero-mod case, still emit `statChangedProvider(transactionId, channel, characterId, []stat.Type{}, map[string]interface{}{})` so the saga step completes and the client unlocks — move that `mb.Put` outside the `len(mods)==0` early return accordingly (i.e. after the transaction, treat it as success with empty stats).

- [ ] **Step 6: Add the consumer handler**

In `kafka/consumer/character/consumer.go`, register next to the RebalanceAP registration (~line 86):

```go
		rf(t, message.AdaptHandler(message.PersistentConfig(handleTransferAP(db))))
```

Handler (next to `handleRebalanceAP` at line 384):

```go
func handleTransferAP(db *gorm.DB) message.Handler[character2.Command[character2.TransferAPCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c character2.Command[character2.TransferAPCommandBody]) {
		if c.Type != character2.CommandTransferAP {
			return
		}
		cha := channel.NewModel(c.WorldId, c.Body.ChannelId)
		if err := character.NewProcessor(l, ctx, db).TransferAPAndEmit(c.TransactionId, c.CharacterId, cha, c.Body.From, c.Body.To); err != nil {
			l.WithError(err).Errorf("Unable to transfer AP for character [%d].", c.CharacterId)
		}
	}
}
```

- [ ] **Step 7: Run the full test suite**

Run: `cd services/atlas-character/atlas.com/character && go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS/clean.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-character/
git commit -m "feat(character): TRANSFER_AP command with point-reset validation (task-126)"
```

---

### Task 9: atlas-skills — transaction plumbing (macro WithTransaction)

**Files:**
- Modify: `services/atlas-skills/atlas.com/skills/macro/processor.go` (add `WithTransaction`)
- Modify: `services/atlas-skills/atlas.com/skills/skill/processor.go` (add `WithTransaction` to the `Processor` interface — the method already exists on `*ProcessorImpl` at line 86)
- Test: covered by Task 10's atomicity tests; this task just needs the module to build.

**Interfaces:**
- Produces: `skill.Processor.WithTransaction(tx *gorm.DB) Processor` (interface method) and `macro.Processor.WithTransaction(tx *gorm.DB) Processor` (new). Task 10 threads one gorm tx through both.

- [ ] **Step 1: Add `WithTransaction` to the skill `Processor` interface**

In `skill/processor.go`, add to the interface (lines 21-66):

```go
	WithTransaction(tx *gorm.DB) Processor
```

(The implementation at line 86 already satisfies it; add the `gorm.io/gorm` import to the interface file section if not present.)

- [ ] **Step 2: Add `WithTransaction` to the macro processor**

In `macro/processor.go`, add to the interface (lines 19-31) and implement, mirroring `skill/processor.go:86-93` field-for-field against macro's `ProcessorImpl` struct:

```go
	WithTransaction(tx *gorm.DB) Processor
```

```go
func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
		t:   p.t,
	}
}
```

(Match macro's actual `ProcessorImpl` fields — copy them from its `NewProcessor`.)

- [ ] **Step 3: Build + existing tests**

Run: `cd services/atlas-skills/atlas.com/skills && go build ./... && go test -race ./... && go vet ./...`
Expected: clean. (Mock processors, if any exist for these interfaces, gain the new method — grep for implementations of `skill.Processor`/`macro.Processor` and add `WithTransaction` passthroughs.)

- [ ] **Step 4: Commit**

```bash
git add services/atlas-skills/
git commit -m "feat(skills): expose WithTransaction on skill + macro processors (task-126)"
```

---

### Task 10: atlas-skills — TRANSFER_SP command

**Files:**
- Modify: `services/atlas-skills/atlas.com/skills/kafka/message/skill/kafka.go` (command + status consts/bodies)
- Modify: `services/atlas-skills/atlas.com/skills/skill/processor.go` (+ `TransferSp`/`TransferSpAndEmit`)
- Modify: `services/atlas-skills/atlas.com/skills/skill/producer.go` (SP_TRANSFERRED + ERROR providers)
- Modify: `services/atlas-skills/atlas.com/skills/kafka/consumer/skill/consumer.go` (handler + registration)
- Create: `services/atlas-skills/atlas.com/skills/skill/transfer_sp_test.go`

**Interfaces:**
- Consumes: Task 1 `job.Advancement`, Task 2 `skill.IsPointResetExcluded`, existing `job.Is`, `job.IdFromSkillId`, `job.IsFourthJob` (all in `libs/atlas-constants/job/model.go`); Task 9's `WithTransaction` on both processors; existing `ByIdProvider(characterId, id)`, `dynamicUpdate`/`SetLevel`, `create`, macro `ByCharacterIdProvider`/`Update`.
- Produces (Task 11 emits the command and consumes both events):
  - `CommandTypeTransferSp = "TRANSFER_SP"`, `TransferSpBody{JobId job.Id; FromSkillId uint32; ToSkillId uint32; ItemTier byte; TargetMaxLevel byte}`
  - `StatusEventTypeSpTransferred = "SP_TRANSFERRED"` with `StatusEventSpTransferredBody{FromSkillId uint32; FromLevel byte; ToLevel byte}` (envelope `SkillId` = target skill)
  - `StatusEventTypeError = "ERROR"` with `StatusEventErrorBody{Error string; Detail string}`; error consts `StatusEventErrorTypeSkillAtZero = "SKILL_AT_ZERO"`, `StatusEventErrorTypeSkillAtCap = "SKILL_AT_CAP"`, `StatusEventErrorTypeWrongTier = "WRONG_TIER"`, `StatusEventErrorTypeInvalidTarget = "INVALID_TARGET"`
  - Processor: `TransferSp(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) error` + `TransferSpAndEmit(...)`

- [ ] **Step 1: Add message types**

In `kafka/message/skill/kafka.go` (command consts, line ~11):

```go
	CommandTypeTransferSp = "TRANSFER_SP"
```

```go
// TransferSpBody moves one skill point FromSkillId -> ToSkillId (SP Reset
// item 505000<ItemTier>). JobId and TargetMaxLevel are supplied by the
// trusted server-side caller (atlas-channel) because atlas-skills stores
// neither job nor game data; everything state-derived is re-validated here.
type TransferSpBody struct {
	JobId          job.Id `json:"jobId"`
	FromSkillId    uint32 `json:"fromSkillId"`
	ToSkillId      uint32 `json:"toSkillId"`
	ItemTier       byte   `json:"itemTier"`
	TargetMaxLevel byte   `json:"targetMaxLevel"`
}
```

Status consts (next to `StatusEventTypeCooldownExpired`, line ~57):

```go
	StatusEventTypeSpTransferred = "SP_TRANSFERRED"
	StatusEventTypeError         = "ERROR"

	StatusEventErrorTypeSkillAtZero   = "SKILL_AT_ZERO"
	StatusEventErrorTypeSkillAtCap    = "SKILL_AT_CAP"
	StatusEventErrorTypeWrongTier     = "WRONG_TIER"
	StatusEventErrorTypeInvalidTarget = "INVALID_TARGET"
```

Bodies (next to `StatusEventUpdatedBody`, line ~76):

```go
// StatusEventSpTransferredBody signals a completed SP transfer; the envelope
// SkillId carries the target skill. This is the saga-completion event.
type StatusEventSpTransferredBody struct {
	FromSkillId uint32 `json:"fromSkillId"`
	FromLevel   byte   `json:"fromLevel"`
	ToLevel     byte   `json:"toLevel"`
}

// StatusEventErrorBody reports a rejected TRANSFER_SP; Error is one of the
// StatusEventErrorType* constants, Detail names the offending skill id.
type StatusEventErrorBody struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}
```

- [ ] **Step 2: Write the failing processor tests**

`skill/transfer_sp_test.go` (package `skill_test`, using `setupProcessor` from `skill/processor_test.go:25-39` and `test.SetupTestDB` — which already migrates both `skill.Entity` and `macro.Entity`). Use warrior job 100-line skill ids: 1st-tier skill 1001003 (Iron Body), 2nd-tier Fighter 1101006, 4th-tier Hero 1120003 — and macro rows created via `macro.NewProcessor(...).Update(...)`. Matrix (one subtest each):

1. Tier-1 transfer success (item tier 1, job 100): from 1000000-tier skill level 3 → 2, to level 1 → 2; buffer contains SP_TRANSFERRED + two UPDATED events.
2. Target row absent (never created): target treated as level 0 and **created** at level 1 (masterLevel 0, zero expiration); CREATED event buffered instead of the second UPDATED.
3. Source level 0 (or row absent) → `SKILL_AT_ZERO` error event; nothing mutated.
4. Target at `TargetMaxLevel` → `SKILL_AT_CAP`; nothing mutated.
5. 4th-job target: cap = its own `masterLevel` row, not `TargetMaxLevel` — target level == masterLevel → `SKILL_AT_CAP`; target level < masterLevel → success.
6. 4th-job target with no row / masterLevel 0 → `SKILL_AT_CAP` (master level must be earned first).
7. Target tier ≠ ItemTier → `WRONG_TIER`; source tier > ItemTier → `WRONG_TIER`; tier-0 (beginner prefix) source or target → `WRONG_TIER`; Evan JobId → `WRONG_TIER`.
8. Skill outside job tree (`job.Is` false) → `INVALID_TARGET`; excluded skill (`skill.IsPointResetExcluded`) → `INVALID_TARGET`.
9. Macro cleanup: source drops to 0 and a macro references it in SkillId2 → that slot zeroed, other slots intact, macro UPDATED event buffered; source drops to 1 (not 0) → macros untouched, no macro event.
10. Validation failure leaves everything untouched (assert both skill rows AND macros unchanged after a `SKILL_AT_CAP` rejection).
11. Master level of source unchanged after transfer out (FR-15).

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd services/atlas-skills/atlas.com/skills && go test ./skill/ -run TestTransferSp -v`
Expected: FAIL with undefined method.

- [ ] **Step 4: Implement providers + processor**

`skill/producer.go` — add two providers mirroring `statusEventUpdatedProvider`'s shape:

```go
func statusEventSpTransferredProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, toSkillId uint32, fromSkillId uint32, fromLevel byte, toLevel byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.StatusEvent[skill2.StatusEventSpTransferredBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		SkillId:       toSkillId,
		Type:          skill2.StatusEventTypeSpTransferred,
		Body: skill2.StatusEventSpTransferredBody{
			FromSkillId: fromSkillId,
			FromLevel:   fromLevel,
			ToLevel:     toLevel,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventErrorProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, errorType string, detail string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.StatusEvent[skill2.StatusEventErrorBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		SkillId:       skillId,
		Type:          skill2.StatusEventTypeError,
		Body: skill2.StatusEventErrorBody{
			Error:  errorType,
			Detail: detail,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

(Match the exact envelope construction of the file's existing providers.)

`skill/processor.go` — interface additions:

```go
	TransferSp(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) error
	TransferSpAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) error
```

Implementation. **Use gorm-native `p.db.Transaction(...)` — NOT `database.ExecuteTransaction`, which never begins a transaction (libs/atlas-database/transaction.go:9-18).** All reads and writes go through tx-bound processors so the two skill rows + macros commit or roll back together:

```go
func (p *ProcessorImpl) TransferSpAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.TransferSp(buf)(transactionId, worldId, characterId, jobId, fromSkillId, toSkillId, itemTier, targetMaxLevel)
	})
}

func (p *ProcessorImpl) TransferSp(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) error {
		reject := func(errType string, detailSkillId uint32) error {
			p.l.WithFields(logrus.Fields{"character_id": characterId, "from": fromSkillId, "to": toSkillId, "tier": itemTier}).
				Warnf("Rejected SP transfer: [%s].", errType)
			_ = mb.Put(skill2.EnvStatusEventTopic, statusEventErrorProvider(transactionId, worldId, characterId, detailSkillId, errType, strconv.FormatUint(uint64(detailSkillId), 10)))
			return nil
		}

		// Structural validation (no DB needed).
		fromJob := job.IdFromSkillId(constskill.Id(fromSkillId))
		toJob := job.IdFromSkillId(constskill.Id(toSkillId))
		if !job.Is(jobId, fromJob) || !job.Is(jobId, toJob) {
			return reject(skill2.StatusEventErrorTypeInvalidTarget, toSkillId)
		}
		if constskill.IsPointResetExcluded(constskill.Id(fromSkillId)) || constskill.IsPointResetExcluded(constskill.Id(toSkillId)) {
			return reject(skill2.StatusEventErrorTypeInvalidTarget, toSkillId)
		}
		fromTier := job.Advancement(fromJob)
		toTier := job.Advancement(toJob)
		if toTier != int(itemTier) || fromTier < 1 || fromTier > int(itemTier) {
			return reject(skill2.StatusEventErrorTypeWrongTier, toSkillId)
		}

		return p.db.Transaction(func(tx *gorm.DB) error {
			sp := p.WithTransaction(tx)

			from, err := sp.ByIdProvider(characterId, fromSkillId)()
			if err != nil || from.Level() == 0 {
				return reject(skill2.StatusEventErrorTypeSkillAtZero, fromSkillId)
			}

			// Target row may not exist yet: treat as level 0 / masterLevel 0.
			var toLevel, toMaster byte
			var toExists bool
			var toExpiration time.Time
			if to, err := sp.ByIdProvider(characterId, toSkillId)(); err == nil {
				toLevel, toMaster, toExpiration, toExists = to.Level(), to.MasterLevel(), to.Expiration(), true
			}

			cap := targetMaxLevel
			if job.IsFourthJob(toJob) {
				cap = toMaster // 4th-job cap is the earned master level (design §9.2)
			}
			if toLevel >= cap {
				return reject(skill2.StatusEventErrorTypeSkillAtCap, toSkillId)
			}

			// Apply: source -1, target +1 (master levels untouched, FR-15/16).
			if _, err := sp.Update(mb)(transactionId, worldId, characterId, fromSkillId, from.Level()-1, from.MasterLevel(), from.Expiration()); err != nil {
				return err
			}
			if toExists {
				if _, err := sp.Update(mb)(transactionId, worldId, characterId, toSkillId, toLevel+1, toMaster, toExpiration); err != nil {
					return err
				}
			} else {
				if _, err := sp.Create(mb)(transactionId, worldId, characterId, toSkillId, 1, 0, time.Time{}); err != nil {
					return err
				}
			}

			// Macro cleanup (FR-18) inside the same tx.
			if from.Level()-1 == 0 {
				mp := macro.NewProcessor(p.l, p.ctx, /* db per macro.NewProcessor signature */).WithTransaction(tx)
				macros, err := mp.ByCharacterIdProvider(characterId)()
				if err != nil {
					return err
				}
				changed := false
				updated := make([]macro.Model, 0, len(macros))
				for _, m := range macros {
					b := macro.CloneModel(m)
					if uint32(m.SkillId1()) == fromSkillId {
						b = b.SetSkillId1(0)
						changed = true
					}
					if uint32(m.SkillId2()) == fromSkillId {
						b = b.SetSkillId2(0)
						changed = true
					}
					if uint32(m.SkillId3()) == fromSkillId {
						b = b.SetSkillId3(0)
						changed = true
					}
					updated = append(updated, b.Build())
				}
				if changed {
					if _, err := mp.Update(mb)(transactionId, worldId, characterId, updated); err != nil {
						return err
					}
				}
			}

			_ = mb.Put(skill2.EnvStatusEventTopic, statusEventSpTransferredProvider(transactionId, worldId, characterId, toSkillId, fromSkillId, from.Level()-1, toLevel+1))
			return nil
		})
	}
}
```

Adjust to actual signatures: `macro.NewProcessor`'s parameters, macro's builder methods (`CloneModel`/setter names — check `macro/model.go`/`builder.go`; if macro models lack a Clone/builder, construct replacement Models via its `NewModelBuilder` equivalents), and the `constskill` import alias for `github.com/Chronicle20/atlas/libs/atlas-constants/skill` (the local package is also named `skill`). Skill-row `Update` requires the row to exist — the existing `Update` re-reads before writing; keep its semantics.

- [ ] **Step 5: Add the consumer handler**

In `kafka/consumer/skill/consumer.go`, one more `rf(...)` in `InitHandlers` (mirror line 40) plus:

```go
func handleCommandTransferSp(db *gorm.DB) message.Handler[skill2.Command[skill2.TransferSpBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c skill2.Command[skill2.TransferSpBody]) {
		if c.Type != skill2.CommandTypeTransferSp {
			return
		}
		if err := skill.NewProcessor(l, ctx, db).TransferSpAndEmit(c.TransactionId, c.WorldId, c.CharacterId, c.Body.JobId, c.Body.FromSkillId, c.Body.ToSkillId, c.Body.ItemTier, c.Body.TargetMaxLevel); err != nil {
			l.WithError(err).Errorf("Unable to transfer SP for character [%d].", c.CharacterId)
		}
	}
}
```

- [ ] **Step 6: Run the full test suite**

Run: `cd services/atlas-skills/atlas.com/skills && go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS/clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-skills/
git commit -m "feat(skills): TRANSFER_SP command with tier validation and macro cleanup (task-126)"
```

---

### Task 11: atlas-saga-orchestrator — actions, events, compensation

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` (alias/const re-exports if present; second `Step.UnmarshalJSON` switch — two new cases)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` (3 new EventKinds + 2 table entries)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance_test.go` (`allActions` + 2)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` (GetHandler cases + 2 handlers)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/processor.go` (+ `TransferAPAndEmit`)
- Modify: orchestrator's skill processor (find via `grep -rn "RequestCreateAndEmit\|EnvCommandTopic" services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/skill/`) (+ `TransferSPAndEmit`)
- Modify: orchestrator's `kafka/message/character/kafka.go` and `kafka/message/skill/kafka.go` copies (mirror the Task 8/10 command consts + bodies and the Task 8/10 status consts + bodies)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/character/consumer.go` (AP-error handler)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/skill/consumer.go` (SP_TRANSFERRED + skill-error handlers)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go` (point_reset reverse-walk)
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/point_reset_compensation_test.go` (or extend the existing compensator test file — check for a PetEvolution compensation test first and follow its structure)

**Interfaces:**
- Consumes: Task 5's shared-lib symbols; Task 8/10 message shapes; existing `AcceptEvent`, `StepCompleted`/`StepCompletedWithResult` (processor.go:337-358), `Step.Result()` (model.go:707), `EmitSagaFailed` (producer.go:102), `compP.RequestCreateItem` (compensator.go:42-56).
- Produces: a runnable `point_reset` saga end-to-end. Error-code threading contract for Task 14: the failed step's result map carries `errorCode` + `errorDetail`; `compensatePointReset` emits `EmitSagaFailed` with that `errorCode` and with **reason = errorDetail** (the channel branch reads `Body.Reason` as the detail carrier).

- [ ] **Step 1: Unmarshal + acceptance + allActions (tests enforce completeness)**

1. Add `TransferAP`/`TransferSP` cases to the orchestrator's `Step.UnmarshalJSON` in `saga/model.go` (same shape as the lib's — before the default branch).
2. In `event_acceptance.go` add EventKinds:

```go
	EventKindCharacterApTransferError EventKind = "character.ap_transfer_error"
	EventKindSkillSpTransferred       EventKind = "skill.sp_transferred"
	EventKindSkillSpTransferError     EventKind = "skill.sp_transfer_error"
```

and table entries:

```go
	sharedsaga.TransferAP: {EventKindCharacterStatChanged, EventKindCharacterApTransferError},
	sharedsaga.TransferSP: {EventKindSkillSpTransferred, EventKindSkillSpTransferError},
```

3. Append both actions to `allActions` in `event_acceptance_test.go`.
4. If `saga/model.go` has an alias/const re-export block (like atlas-channel's), add `PointReset`, `TransferAP`, `TransferSP`, `TransferAPPayload`, `TransferSPPayload`.

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run "TestAcceptanceTable_EveryActionRepresented|TestStepUnmarshal_EveryActionRepresented" -v`
Expected: PASS (these tests FAIL until both the table entry and the unmarshal case exist — use them as the TDD loop).

- [ ] **Step 2: Command emit methods + message mirrors**

Mirror `CommandTransferAP`/`TransferAPCommandBody` into the orchestrator's `kafka/message/character/kafka.go` and `CommandTypeTransferSp`/`TransferSpBody` + the new status consts/bodies (`StatusEventTypeSpTransferred`, `StatusEventTypeError`, `StatusEventSpTransferredBody`, `StatusEventErrorBody`, and character `StatusEventApTransferErrorBody` + error type consts) into the respective message copies, exactly as defined in Tasks 8 and 10.

Add to `character/processor.go` (mirror the file's existing `ResetStatsAndEmit`-style method shape and its producer helpers):

```go
func (p *ProcessorImpl) TransferAPAndEmit(transactionId uuid.UUID, ch channel.Model, characterId uint32, from string, to string) error
```

emitting `character2.Command[character2.TransferAPCommandBody]{TransactionId: transactionId, WorldId: ch.WorldId(), CharacterId: characterId, Type: character2.CommandTransferAP, Body: {ChannelId: ch.Id(), From: from, To: to}}` on the character command topic.

Add to the orchestrator's skill processor:

```go
func (p *ProcessorImpl) TransferSPAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) error
```

emitting the `TRANSFER_SP` command on the skill command topic.

- [ ] **Step 3: GetHandler + handlers**

In `handler.go` `GetHandler` switch (line 701-864):

```go
	case TransferAP:
		return h.handleTransferAP, true
	case TransferSP:
		return h.handleTransferSP, true
```

Handlers (mirror `handleResetStats` at handler.go:2220-2234):

```go
func (h *HandlerImpl) handleTransferAP(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(TransferAPPayload)
	if !ok {
		return errors.New("invalid payload")
	}
	ch := channel.NewModel(payload.WorldId, payload.ChannelId)
	err := h.charP.TransferAPAndEmit(s.TransactionId(), ch, payload.CharacterId, payload.From, payload.To)
	if err != nil {
		h.logActionError(s, st, err, "Unable to transfer AP.")
		return err
	}
	return nil
}

func (h *HandlerImpl) handleTransferSP(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(TransferSPPayload)
	if !ok {
		return errors.New("invalid payload")
	}
	err := h.skillP.TransferSPAndEmit(s.TransactionId(), payload.WorldId, payload.CharacterId, payload.JobId, uint32(payload.FromSkillId), uint32(payload.ToSkillId), payload.ItemTier, payload.TargetMaxLevel)
	if err != nil {
		h.logActionError(s, st, err, "Unable to transfer SP.")
		return err
	}
	return nil
}
```

(`h.skillP` — use the handler struct's actual skill-processor field name; find it via the `CreateSkill` handler.)

- [ ] **Step 4: Status-event consumers**

`kafka/consumer/character/consumer.go` — new handler registered in `InitHandlers` (mirror the meso-error handler at lines 157-175):

```go
func handleCharacterApTransferErrorEvent(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventApTransferErrorBody]) {
	if e.Type != character2.StatusEventTypeError {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindCharacterApTransferError); !ok {
		return
	}
	l.WithFields(logrus.Fields{"transaction_id": e.TransactionId, "error": e.Body.Error}).Debug("AP transfer rejected; marking saga step failed.")
	_ = p.StepCompletedWithResult(e.TransactionId, false, map[string]any{"errorCode": e.Body.Error, "errorDetail": e.Body.Detail})
}
```

`kafka/consumer/skill/consumer.go` — two new handlers registered in `InitHandlers` (mirror `handleSkillUpdatedEvent` at lines 43-76):

```go
func handleSkillSpTransferredEvent(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventSpTransferredBody]) {
	if e.Type != skill2.StatusEventTypeSpTransferred {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindSkillSpTransferred); !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}

func handleSkillErrorEvent(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventErrorBody]) {
	if e.Type != skill2.StatusEventTypeError {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindSkillSpTransferError); !ok {
		return
	}
	l.WithFields(logrus.Fields{"transaction_id": e.TransactionId, "error": e.Body.Error}).Debug("SP transfer rejected; marking saga step failed.")
	_ = p.StepCompletedWithResult(e.TransactionId, false, map[string]any{"errorCode": e.Body.Error, "errorDetail": e.Body.Detail})
}
```

Note: the acceptance-table gate (`AcceptEvent` default-deny) is what keeps these ERROR events from confusing other saga types, and keeps meso-error/AP-error handlers from cross-firing — both decode `Type == "ERROR"` but only the step's action determines which EventKind is accepted.

- [ ] **Step 5: point_reset compensation**

In `compensator.go`, in `CompensateFailedStep` after the PetEvolution dispatch (line ~192):

```go
	// Point-reset reverse-walk: destroy-first saga; invert the completed
	// destroy via re-award, then emit the saga-failed event carrying the
	// service's machine-readable error code (threaded via the failed step's
	// result map) so atlas-channel can render specific pink text.
	if s.SagaType() == PointReset {
		return c.compensatePointReset(s, failedStep)
	}
```

And (mirror `compensatePetEvolution`/`DispatchPetEvolutionRollbacks` at compensator.go:1053-1140):

```go
func (c *CompensatorImpl) compensatePointReset(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{"transaction_id": s.TransactionId(), "failed_step": failedStep.StepId()}).
		Info("Compensating point_reset saga.")
	c.DispatchPointResetRollbacks(s)

	if !GetCache().TryTransition(c.ctx, s.TransactionId(), SagaLifecycleCompensating, SagaLifecycleFailed) {
		SagaTimers().Cancel(s.TransactionId())
		GetCache().Remove(c.ctx, s.TransactionId())
		return nil
	}
	SagaTimers().Cancel(s.TransactionId())
	GetCache().Remove(c.ctx, s.TransactionId())

	errorCode := sagaMsg.ErrorCodeUnknown
	reason := fmt.Sprintf("Point reset failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if res := failedStep.Result(); res != nil {
		if v, ok := res["errorCode"].(string); ok && v != "" {
			errorCode = v
		}
		if v, ok := res["errorDetail"].(string); ok && v != "" {
			reason = v // channel reads Reason as the detail carrier (stat name)
		}
	}
	if err := EmitSagaFailed(c.l, c.ctx, s, errorCode, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).Error("Unable to emit saga failed event for point_reset.")
		return err
	}
	return nil
}

func (c *CompensatorImpl) DispatchPointResetRollbacks(s Saga) {
	steps := s.Steps()
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Status() != Completed {
			continue
		}
		if step.Action() != DestroyAsset {
			continue
		}
		if payload, ok := step.Payload().(DestroyAssetPayload); ok {
			qty := payload.Quantity
			if qty == 0 {
				qty = 1
			}
			if err := c.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, time.Time{}); err != nil {
				c.l.WithError(err).Error("Reverse-walk: DestroyAsset -> CreateItem dispatch failed; continuing chain.")
			}
		}
	}
}
```

(Match the file's exact cache/timer/log idioms from `compensatePetEvolution` — copy that function and adapt; the structural skeleton above shows intent, the existing function is the source of truth for lifecycle calls.)

- [ ] **Step 6: Compensation test**

Follow the existing PetEvolution compensation test (search `compensator_test.go` / `grep -rn "compensatePetEvolution\|PetEvolution" services/atlas-saga-orchestrator/ --include="*_test.go"`). Write the analogous `point_reset` test: a 2-step saga with `destroy_asset` Completed and `transfer_ap` Failed (result map `{"errorCode": "POOL_BELOW_JOB_MINIMUM", "errorDetail": "HP"}`) → assert `RequestCreateItem` dispatched with the destroyed template id and the saga-failed event carries `ErrorCode == "POOL_BELOW_JOB_MINIMUM"` and `Reason == "HP"`. If no PetEvolution test exists to pattern-match, test `DispatchPointResetRollbacks` + the errorCode extraction as separable units with whatever fake/capture seams the package already uses.

- [ ] **Step 7: Full module check + commit**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... && go vet ./... && go build ./...
git add services/atlas-saga-orchestrator/ libs/atlas-saga/
git commit -m "feat(saga-orchestrator): point_reset saga with transfer_ap/transfer_sp steps (task-126)"
```

---

### Task 12: atlas-channel — HpMpUsed accessor + macro status consumer

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/character/model.go` (accessor)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/macro/kafka.go` (status topic + event types)
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/macro/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (register the new consumer)

**Interfaces:**
- Consumes: atlas-skills' macro status event shape (`services/atlas-skills/atlas.com/skills/kafka/message/macro/kafka.go` — `EnvStatusEventTopic = "STATUS_EVENT_TOPIC_SKILL_MACRO"`, `StatusEvent[StatusEventUpdatedBody]`, `MacroBody`); the login-path macro packet code (`kafka/consumer/session/consumer.go:320-339`).
- Produces: `character.Model.HpMpUsed() int` (Task 13 uses it); live macro-packet push on macro UPDATED (FR-18 client visibility).

- [ ] **Step 1: Add the accessor**

In `character/model.go` (field `hpMpUsed int` exists at line 42):

```go
func (m Model) HpMpUsed() int {
	return m.hpMpUsed
}
```

- [ ] **Step 2: Mirror the macro status event types**

In `kafka/message/macro/kafka.go` (which today only has `EnvCommandTopic` + `CommandTypeUpdate`), add the status constants and structs copied field-for-field from `services/atlas-skills/atlas.com/skills/kafka/message/macro/kafka.go` (`EnvStatusEventTopic`, `StatusEventTypeUpdated`, `StatusEvent[E]`, `StatusEventUpdatedBody`, `MacroBody`).

- [ ] **Step 3: Write the consumer**

`kafka/consumer/macro/consumer.go`, modeled exactly on `kafka/consumer/skill/consumer.go:1-63` (same InitConsumers/InitHandlers shape, consumer name `"skill_macro_status_event"`, topic `macro2.EnvStatusEventTopic`), with one handler:

```go
func handleUpdated(sc server.Model, wp writer.Producer) message.Handler[macro2.StatusEvent[macro2.StatusEventUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e macro2.StatusEvent[macro2.StatusEventUpdatedBody]) {
		if e.Type != macro2.StatusEventTypeUpdated {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, announceMacros(l)(ctx)(wp)(e.Body.Macros))
		if err != nil {
			l.WithError(err).Errorf("Unable to update skill macros for character [%d].", e.CharacterId)
		}
	}
}

func announceMacros(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(bodies []macro2.MacroBody) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(bodies []macro2.MacroBody) model.Operator[session.Model] {
		return func(wp writer.Producer) func(bodies []macro2.MacroBody) model.Operator[session.Model] {
			return func(bodies []macro2.MacroBody) model.Operator[session.Model] {
				sorted := make([]macro2.MacroBody, len(bodies))
				copy(sorted, bodies)
				sort.Slice(sorted, func(i, j int) bool { return sorted[i].Id < sorted[j].Id })
				mms := make([]packetmodel.Macro, 0, len(sorted))
				for _, sm := range sorted {
					mms = append(mms, packetmodel.NewMacro(sm.Name, sm.Shout, skill2.Id(sm.SkillId1), skill2.Id(sm.SkillId2), skill2.Id(sm.SkillId3)))
				}
				macros := packetmodel.NewMacros(mms...)
				return session.Announce(l)(ctx)(wp)(charpkt.CharacterSkillMacroWriter)(macros.Encode)
			}
		}
	}
}
```

Match `MacroBody`'s actual field names/types (from the atlas-skills source of truth) and `packetmodel.NewMacro`'s parameter types (the login path at `session/consumer.go:331-334` shows the working call — copy its conversions; `charpkt` is `github.com/Chronicle20/atlas/libs/atlas-packet/character` per the session consumer's import).

- [ ] **Step 4: Register in main.go**

Find where `skill.InitConsumers`/`skill.InitHandlers` (channel's own consumer package) are wired in `main.go` and add the macro consumer package's `InitConsumers`/`InitHandlers` adjacent, same argument shapes.

- [ ] **Step 5: Build + commit**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./... && go vet ./...
git add services/atlas-channel/
git commit -m "feat(channel): HpMpUsed accessor + live skill-macro status consumer (task-126)"
```

---

### Task 13: atlas-channel — `pointreset` validation package

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/pointreset/model.go`
- Create: `services/atlas-channel/atlas.com/channel/pointreset/model_test.go`

**Interfaces:**
- Consumes: Task 1 `job.Advancement`, Task 2 `skill.IsPointResetExcluded`, `job.Is`, `job.IdFromSkillId`, `job.IsFourthJob`; Task 12's `character.Model.HpMpUsed()`; channel `character.Model` accessors (`Strength/Dexterity/Intelligence/Luck/MaxHp/MaxMp/Hp/JobId/Level/SkillById`) and channel `character/skill.Model` (`Level()/MasterLevel()`).
- Produces (Tasks 14 and 15 consume all of this):
  - Ability consts: `AbilityStrength = "STRENGTH"`, `AbilityDexterity = "DEXTERITY"`, `AbilityIntelligence = "INTELLIGENCE"`, `AbilityLuck = "LUCK"`, `AbilityHp = "HP"`, `AbilityMp = "MP"` (must equal atlas-character's `CommandDistributeApAbility*` strings)
  - Error-code consts mirroring the Global Constraints list: `ErrorCodeStatAtMinimum` … `ErrorCodeInvalidTarget`
  - `ApResetItemId = item.Id(5050000)`; `SpResetTier(itemId item.Id) (byte, bool)` (true for 5050001–5050004, tier = id%10)
  - `AbilityFromWireFlag(flag uint32) (string, bool)` (64→STR, 128→DEX, 256→INT, 512→LUK, 2048→HP, 8192→MP)
  - `ValidationError{Code string; Detail string}`
  - `ValidateApTransfer(c character.Model, from string, to string) *ValidationError` (nil = pass)
  - `ValidateSpTransfer(c character.Model, fromId skill.Id, toId skill.Id, tier byte, gameDataMaxLevel byte) *ValidationError`
  - `ErrorMessage(code string, detail string) string`

- [ ] **Step 1: Write the failing tests**

`pointreset/model_test.go` covering:

1. `AbilityFromWireFlag` — all six mappings + unknown flag false.
2. `SpResetTier` — 5050001→(1,true) … 5050004→(4,true); 5050000/5050005→false.
3. `ValidateApTransfer` matrix using channel `character.Model` values built via its builder (`character/builder.go` — `SetStrength`, `SetHpMpUsed`, etc.; check the actual setter names): source primary at 4 → `{STAT_AT_MINIMUM, "STRENGTH"}`; source ≥ 5 pass; HP source with HpMpUsed 0 → `{INSUFFICIENT_HPMP_AP_USED, "HP"}`; target primary at 32767 → `{STAT_AT_MAXIMUM, "DEXTERITY"}`; target MaxHp ≥ 30000 → `{STAT_AT_MAXIMUM, "HP"}`; bad ability string → `{INVALID_TARGET, ...}`; From==To STR→STR with STR 10 → nil (pool-minimum is deliberately NOT checked here — atlas-character owns it, design §7).
4. `ValidateSpTransfer` matrix (job 100-line character with a skills slice): out-of-job-tree skill → `INVALID_TARGET`; excluded skill → `INVALID_TARGET`; target tier ≠ item tier → `WRONG_TIER`; source tier > item tier → `WRONG_TIER`; beginner-prefix skill → `WRONG_TIER`; Evan job → `WRONG_TIER`; source level 0 / absent → `SKILL_AT_ZERO`; target at gameDataMaxLevel → `SKILL_AT_CAP`; 4th-job target capped by MasterLevel not gameDataMaxLevel; happy path nil.
5. `ErrorMessage`: `STAT_AT_MINIMUM`+`"STR"` → `"You don't have the minimum STR required to swap."`; `INSUFFICIENT_HPMP_AP_USED` → `"You don't have enough HPMP stat points to spend on AP Reset."`; `POOL_BELOW_JOB_MINIMUM`+`"HP"` → `"You don't have the minimum HP pool required to swap."`; `SKILL_AT_ZERO` → `"There are no points in that skill to move."`; `SKILL_AT_CAP` → `"That skill cannot be raised any further."`; `WRONG_TIER` → `"That SP Reset cannot move points into that skill."`; `INVALID_TARGET` → `"That skill's points cannot be moved."`; unknown code / unrecognized detail → `"Couldn't execute AP reset operation."`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./pointreset/ -v`
Expected: FAIL (package doesn't exist yet — create the test alongside the implementation file skeleton so it compiles, then watch assertions fail).

- [ ] **Step 3: Implement**

`pointreset/model.go`:

```go
// Package pointreset holds the channel-side pre-validation and player-facing
// messages for AP Reset (5050000) and SP Reset (5050001-5050004) cash items.
// The numeric job policy tables (take/gain/min-pool) deliberately live in
// atlas-character (design §7); this package checks only the structural rules
// and the floors/caps/gates visible on the channel character model.
package pointreset

import (
	"fmt"

	"atlas-channel/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// Ability enum strings — must match atlas-character's
// CommandDistributeApAbility* constants (its processor.go:44-51).
const (
	AbilityStrength     = "STRENGTH"
	AbilityDexterity    = "DEXTERITY"
	AbilityIntelligence = "INTELLIGENCE"
	AbilityLuck         = "LUCK"
	AbilityHp           = "HP"
	AbilityMp           = "MP"
)

// Machine-readable rejection codes — shared strings with the services' ERROR
// status events and the saga-failed ErrorCode.
const (
	ErrorCodeStatAtMinimum          = "STAT_AT_MINIMUM"
	ErrorCodeStatAtMaximum          = "STAT_AT_MAXIMUM"
	ErrorCodeInsufficientHpMpApUsed = "INSUFFICIENT_HPMP_AP_USED"
	ErrorCodePoolBelowJobMinimum    = "POOL_BELOW_JOB_MINIMUM"
	ErrorCodeSkillAtZero            = "SKILL_AT_ZERO"
	ErrorCodeSkillAtCap             = "SKILL_AT_CAP"
	ErrorCodeWrongTier              = "WRONG_TIER"
	ErrorCodeInvalidTarget          = "INVALID_TARGET"
)

// Fixed server policy (design §2.2): source floor 4 (must be >= 5 to move
// out), primary cap 32767, pool cap 30000.
const (
	primaryFloor = uint16(4)
	primaryCap   = uint16(32767)
	poolCap      = uint16(30000)
)

var ApResetItemId = item.Id(5050000)

// SpResetTier returns the SP Reset job-advancement tier for items
// 5050001-5050004 and false for anything else.
func SpResetTier(itemId item.Id) (byte, bool) {
	if itemId >= item.Id(5050001) && itemId <= item.Id(5050004) {
		return byte(itemId % 10), true
	}
	return 0, false
}

// AbilityFromWireFlag maps the client stat-flag encoding of the AP Reset body
// (Cosmic AssignAPProcessor.APResetAction) to an ability enum string.
func AbilityFromWireFlag(flag uint32) (string, bool) {
	switch flag {
	case 64:
		return AbilityStrength, true
	case 128:
		return AbilityDexterity, true
	case 256:
		return AbilityIntelligence, true
	case 512:
		return AbilityLuck, true
	case 2048:
		return AbilityHp, true
	case 8192:
		return AbilityMp, true
	}
	return "", false
}

type ValidationError struct {
	Code   string
	Detail string
}

func primaryValue(c character.Model, ability string) (uint16, bool) {
	switch ability {
	case AbilityStrength:
		return c.Strength(), true
	case AbilityDexterity:
		return c.Dexterity(), true
	case AbilityIntelligence:
		return c.Intelligence(), true
	case AbilityLuck:
		return c.Luck(), true
	}
	return 0, false
}

// ValidateApTransfer checks the structural AP-reset rules the channel can see
// cheaply. The job pool-minimum check (minHp/minMp tables) is atlas-character's
// alone and is NOT mirrored here.
func ValidateApTransfer(c character.Model, from string, to string) *ValidationError {
	// Source.
	if v, ok := primaryValue(c, from); ok {
		if v < primaryFloor+1 {
			return &ValidationError{Code: ErrorCodeStatAtMinimum, Detail: from}
		}
	} else if from == AbilityHp || from == AbilityMp {
		if c.HpMpUsed() < 1 {
			return &ValidationError{Code: ErrorCodeInsufficientHpMpApUsed, Detail: from}
		}
	} else {
		return &ValidationError{Code: ErrorCodeInvalidTarget, Detail: from}
	}
	// Target.
	if v, ok := primaryValue(c, to); ok {
		if v >= primaryCap {
			return &ValidationError{Code: ErrorCodeStatAtMaximum, Detail: to}
		}
	} else if to == AbilityHp {
		if c.MaxHp() >= poolCap {
			return &ValidationError{Code: ErrorCodeStatAtMaximum, Detail: to}
		}
	} else if to == AbilityMp {
		if c.MaxMp() >= poolCap {
			return &ValidationError{Code: ErrorCodeStatAtMaximum, Detail: to}
		}
	} else {
		return &ValidationError{Code: ErrorCodeInvalidTarget, Detail: to}
	}
	return nil
}

// ValidateSpTransfer checks the full SP-reset rule set (design §4.3 arm 5).
// gameDataMaxLevel is len(Effects()) from atlas-data for the target skill;
// the 4th-job cap is the character's own master level.
func ValidateSpTransfer(c character.Model, fromId skill.Id, toId skill.Id, tier byte, gameDataMaxLevel byte) *ValidationError {
	fromJob := job.IdFromSkillId(fromId)
	toJob := job.IdFromSkillId(toId)
	if !job.Is(c.JobId(), fromJob) || !job.Is(c.JobId(), toJob) {
		return &ValidationError{Code: ErrorCodeInvalidTarget}
	}
	if skill.IsPointResetExcluded(fromId) || skill.IsPointResetExcluded(toId) {
		return &ValidationError{Code: ErrorCodeInvalidTarget}
	}
	fromTier := job.Advancement(fromJob)
	toTier := job.Advancement(toJob)
	if toTier != int(tier) || fromTier < 1 || fromTier > int(tier) {
		return &ValidationError{Code: ErrorCodeWrongTier}
	}
	fromSkill, ok := c.SkillById(fromId)
	if !ok || fromSkill.Level() == 0 {
		return &ValidationError{Code: ErrorCodeSkillAtZero}
	}
	var toLevel, toMaster byte
	if toSkill, ok := c.SkillById(toId); ok {
		toLevel, toMaster = toSkill.Level(), toSkill.MasterLevel()
	}
	cap := gameDataMaxLevel
	if job.IsFourthJob(toJob) {
		cap = toMaster
	}
	if toLevel >= cap {
		return &ValidationError{Code: ErrorCodeSkillAtCap}
	}
	return nil
}

// abilityDisplay maps ability enum strings to the short names used in the
// Cosmic-parity messages.
var abilityDisplay = map[string]string{
	AbilityStrength:     "STR",
	AbilityDexterity:    "DEX",
	AbilityIntelligence: "INT",
	AbilityLuck:         "LUK",
	AbilityHp:           "HP",
	AbilityMp:           "MP",
}

// ErrorMessage renders the player-facing pink-text message for a rejection
// code. detail is the ability enum (or, on the saga path, the failed event's
// Reason field, which the compensator sets to the service's errorDetail).
func ErrorMessage(code string, detail string) string {
	disp, known := abilityDisplay[detail]
	if !known {
		disp = detail
	}
	switch code {
	case ErrorCodeStatAtMinimum:
		if disp != "" && (known || disp == "STR" || disp == "DEX" || disp == "INT" || disp == "LUK" || disp == "HP" || disp == "MP") {
			return fmt.Sprintf("You don't have the minimum %s required to swap.", disp)
		}
	case ErrorCodeInsufficientHpMpApUsed:
		return "You don't have enough HPMP stat points to spend on AP Reset."
	case ErrorCodePoolBelowJobMinimum:
		if disp == "HP" || disp == "MP" {
			return fmt.Sprintf("You don't have the minimum %s pool required to swap.", disp)
		}
	case ErrorCodeSkillAtZero:
		return "There are no points in that skill to move."
	case ErrorCodeSkillAtCap:
		return "That skill cannot be raised any further."
	case ErrorCodeWrongTier:
		return "That SP Reset cannot move points into that skill."
	case ErrorCodeInvalidTarget:
		return "That skill's points cannot be moved."
	}
	return "Couldn't execute AP reset operation."
}
```

Adjust `c.SkillById(...)`'s return shape to the actual signature at `character/model.go:259-266` (if it returns `(skill.Model, error)` adapt the two call sites). Verify the builder setter names used in tests against `character/builder.go`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./pointreset/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/pointreset/
git commit -m "feat(channel): pointreset pre-validation + player messages (task-126)"
```

---

### Task 14: atlas-channel — saga aliases, message constants, failed-event branch

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/saga/model.go` (alias block, lines 9-71)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/saga/kafka.go` (SagaTypePointReset)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer.go` (point_reset branch in `handleFailedEvent`)

**Interfaces:**
- Consumes: Task 5 shared-lib symbols; Task 13 `pointreset.ErrorMessage`; existing `handleFailedEvent` (consumer.go:78-130), pink-text idiom (`chatpkt.WorldMessageWriter` + `writer.WorldMessagePinkTextBody("", "", msg)` — see `kafka/consumer/party_quest/consumer.go:117`), enable-actions idiom (`statpkt.NewStatChanged(make([]statpkt.Update, 0), true)` — see `kafka/consumer/consumable/consumer.go:76`).
- Produces: saga aliases `PointReset`, `TransferAP`, `TransferSP`, `TransferAPPayload`, `TransferSPPayload` for Task 15; player feedback on mid-saga failures.

- [ ] **Step 1: Extend the alias block**

In `saga/model.go` type aliases:

```go
	TransferAPPayload = sharedsaga.TransferAPPayload
	TransferSPPayload = sharedsaga.TransferSPPayload
```

In the const block (saga types group):

```go
	PointReset = sharedsaga.PointReset
```

(action group):

```go
	TransferAP = sharedsaga.TransferAP
	TransferSP = sharedsaga.TransferSP
```

- [ ] **Step 2: Add the saga-type constant to the channel message copy**

In `kafka/message/saga/kafka.go` next to `SagaTypeStorageOperation` (line 23):

```go
	SagaTypePointReset = "point_reset"
```

- [ ] **Step 3: Add the failed-event branch**

In `handleFailedEvent` (consumer.go), after the storage-operation block, add:

```go
		// Point-reset failures: specific pink text where the service supplied
		// a machine-readable code (threaded through the compensator; Reason
		// carries the stat detail), then re-enable client actions.
		if e.Body.SagaType == saga.SagaTypePointReset {
			msg := pointreset.ErrorMessage(e.Body.ErrorCode, e.Body.Reason)
			err = session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePinkTextBody("", "", msg))(s)
			if err != nil {
				l.WithError(err).WithField("character_id", e.Body.CharacterId).Error("Failed to send point-reset pink text.")
			}
			err = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
			if err != nil {
				l.WithError(err).WithField("character_id", e.Body.CharacterId).Error("Failed to send enable-actions after point-reset failure.")
			}
			return
		}
```

Add the imports the file lacks (`atlas-channel/pointreset`, `atlas-channel/socket/writer`, `chatpkt`/`statpkt` packet aliases — copy exact paths from `party_quest/consumer.go` and `consumable/consumer.go`).

- [ ] **Step 4: Build + commit**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./... && go vet ./...
git add services/atlas-channel/
git commit -m "feat(channel): point_reset saga failure feedback (task-126)"
```

---

### Task 15: atlas-channel — handler arm + saga assembly

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` (new arm; use the `writer.Producer` param)
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_point_reset.go`

**Interfaces:**
- Consumes: Task 3 codec (`cashsb.NewItemUsePointReset`), Task 12 (`HpMpUsed()`, character `GetById` + `SkillModelDecorator` — processor.go:65-70/149-155), Task 13 (`pointreset.*`), Task 14 aliases, existing `saga.NewProcessor(l, ctx).Create` (saga/processor.go:31-33), `data/skill` client (`GetById(...).Effects()` — data/skill/processor.go), pink-text/enable-actions idioms.
- Produces: the complete feature entry point (design §4.3 flow).

- [ ] **Step 1: Name the cash-slot types and use the writer producer**

In `character_cash_item_use.go`:
1. Change the handler signature's `_ writer.Producer` to `wp writer.Producer` (line 25).
2. Add named constants next to the existing ones (lines 116-120):

```go
	CashSlotItemTypeSpReset       = CashSlotItemType(23) // GetCashSlotItemType quirk: 5050002-4 also land here
	CashSlotItemTypeApReset       = CashSlotItemType(24)
```

3. Add the arm ABOVE the fall-through warn (line ~109), after the field-effect arm:

```go
		if it == CashSlotItemTypeApReset || it == CashSlotItemTypeSpReset {
			sp := cashsb.NewItemUsePointReset(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			handlePointResetItemUse(l, ctx, wp)(s, itemId, *sp)
			return
		}
```

- [ ] **Step 2: Write the arm implementation**

`character_cash_item_use_point_reset.go`:

```go
package handler

import (
	character2 "atlas-channel/character"
	dataskill "atlas-channel/data/skill"
	"atlas-channel/pointreset"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// handlePointResetItemUse implements the CashSlotItemType 23/24 arm: AP Reset
// (5050000) and SP Reset (5050001-5050004). AP-vs-SP is decided by item id —
// the 23/24 type distinction is never used for dispatch (design §2.4).
func handlePointResetItemUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id, p cashsb.ItemUsePointReset) {
	return func(s session.Model, itemId item.Id, p cashsb.ItemUsePointReset) {
		enableActions := func() {
			_ = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
		}
		rejectWithMessage := func(msg string) {
			_ = session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePinkTextBody("", "", msg))(s)
			enableActions()
		}

		cp := character2.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.SkillModelDecorator)(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to load character [%d] for point reset.", s.CharacterId())
			enableActions()
			return
		}

		// FR-4: a dead character cannot use either item (enable-actions only,
		// no pink text — Cosmic parity).
		if c.Hp() == 0 {
			l.Warnf("Character [%d] attempted point reset [%d] while dead.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		f := s.Field()
		now := time.Now()

		buildSaga := func(transferStep saga.Step) saga.Saga {
			return saga.Saga{
				TransactionId: uuid.New(),
				SagaType:      saga.PointReset,
				InitiatedBy:   "CASH_ITEM_USE",
				Steps: []saga.Step{
					{
						StepId: "consume_point_reset_item",
						Status: saga.Pending,
						Action: saga.DestroyAsset,
						Payload: saga.DestroyAssetPayload{
							CharacterId: s.CharacterId(),
							TemplateId:  uint32(itemId),
							Quantity:    1,
							RemoveAll:   false,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					transferStep,
				},
			}
		}

		if itemId == pointreset.ApResetItemId {
			to, okTo := pointreset.AbilityFromWireFlag(p.To())
			from, okFrom := pointreset.AbilityFromWireFlag(p.From())
			if !okTo || !okFrom {
				l.Warnf("Character [%d] sent AP reset with invalid stat flags to [%d] from [%d].", s.CharacterId(), p.To(), p.From())
				enableActions()
				return
			}
			if ve := pointreset.ValidateApTransfer(c, from, to); ve != nil {
				l.Warnf("Character [%d] AP reset pre-validation rejected: [%s] detail [%s].", s.CharacterId(), ve.Code, ve.Detail)
				rejectWithMessage(pointreset.ErrorMessage(ve.Code, ve.Detail))
				return
			}
			_ = saga.NewProcessor(l, ctx).Create(buildSaga(saga.Step{
				StepId: "transfer_point",
				Status: saga.Pending,
				Action: saga.TransferAP,
				Payload: saga.TransferAPPayload{
					CharacterId: s.CharacterId(),
					WorldId:     f.WorldId(),
					ChannelId:   f.ChannelId(),
					From:        from,
					To:          to,
				},
				CreatedAt: now,
				UpdatedAt: now,
			}))
			return
		}

		if tier, ok := pointreset.SpResetTier(itemId); ok {
			toId := skill2.Id(p.To())
			fromId := skill2.Id(p.From())

			// Game-data max level for the target (non-4th-job cap); also
			// confirms the skill exists in game data.
			ds, err := dataskill.NewProcessor(l, ctx).GetById(uint32(toId))
			if err != nil {
				l.WithError(err).Warnf("Character [%d] SP reset target [%d] not found in game data.", s.CharacterId(), toId)
				rejectWithMessage(pointreset.ErrorMessage(pointreset.ErrorCodeInvalidTarget, ""))
				return
			}
			targetMaxLevel := byte(len(ds.Effects()))

			if ve := pointreset.ValidateSpTransfer(c, fromId, toId, tier, targetMaxLevel); ve != nil {
				l.Warnf("Character [%d] SP reset pre-validation rejected: [%s] from [%d] to [%d] tier [%d].", s.CharacterId(), ve.Code, fromId, toId, tier)
				rejectWithMessage(pointreset.ErrorMessage(ve.Code, ve.Detail))
				return
			}
			_ = saga.NewProcessor(l, ctx).Create(buildSaga(saga.Step{
				StepId: "transfer_point",
				Status: saga.Pending,
				Action: saga.TransferSP,
				Payload: saga.TransferSPPayload{
					CharacterId:    s.CharacterId(),
					WorldId:        f.WorldId(),
					ChannelId:      f.ChannelId(),
					JobId:          c.JobId(),
					FromSkillId:    fromId,
					ToSkillId:      toId,
					ItemTier:       tier,
					TargetMaxLevel: targetMaxLevel,
				},
				CreatedAt: now,
				UpdatedAt: now,
			}))
			return
		}

		// A 505x classification id that is neither 5050000 nor 5050001-4 is
		// impossible from a legit client.
		l.Warnf("Character [%d] attempted point reset with unexpected item [%d].", s.CharacterId(), itemId)
		enableActions()
	}
}
```

Adjust to actual signatures where the file disagrees: `character2.Model.JobId()` accessor name, `SkillModelDecorator` invocation shape (it is a method value — `cp.GetById(cp.SkillModelDecorator)` per processor.go:65-70 requires the concrete processor; if `NewProcessor` returns an interface exposing both, this works as written), and the packet import aliases already used elsewhere in the handler package.

- [ ] **Step 3: Full channel verification**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./... && go vet ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/
git commit -m "feat(channel): AP/SP reset cash-item handler arm (task-126)"
```

---

### Task 16: Seed templates, live-config patch doc, v92 park

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`
- Create: `docs/tasks/task-126-ap-sp-reset-items/deployment.md`

**Interfaces:**
- Consumes: registry opcodes (v87 0x52, v95 0x55, jms 0x47 — `docs/packets/registry/*.yaml`); the gms_83 handler row shape (`template_gms_83_1.json:409-413`).
- Produces: seed wiring for new tenants + a documented PATCH procedure for existing tenants.

- [ ] **Step 1: Add the handler rows**

In each template's top-level `"handlers"` array (starts line ~7), insert (keep the array's existing ordering convention — adjacent to other handler entries):

`template_gms_87_1.json`:
```json
      {
        "opCode": "0x52",
        "validator": "LoggedInValidator",
        "handler": "CharacterCashItemUseHandle"
      },
```

`template_gms_95_1.json`: same row with `"opCode": "0x55"`.
`template_jms_185_1.json`: same row with `"opCode": "0x47"`.

Every entry MUST carry the validator — a validator-less handler entry is silently dropped by `BuildHandlerMap`.

- [ ] **Step 2: Verify the writers this feature answers with are wired on those versions**

```bash
for tmpl in template_gms_87_1 template_gms_95_1 template_jms_185_1; do
  for w in StatChanged WorldMessage CharacterSkillChange CharacterSkillMacro; do
    echo "$tmpl $w: $(grep -c "\"$w\"" services/atlas-configurations/seed-data/templates/$tmpl.json)"
  done
done
```
Expected: every count ≥ 1. If a writer is missing from a template, STOP and report BLOCKED — wiring a new clientbound writer requires its own IDA-verified opcode; do not guess one.

- [ ] **Step 3: Write the deployment doc**

`docs/tasks/task-126-ap-sp-reset-items/deployment.md` containing:
1. **Live tenant patch** — existing tenants do not re-seed: for each live tenant on gms_87/gms_95/jms_185, PATCH the tenant's socket configuration to append the handler row from Step 1 (same JSON shape), then **restart atlas-channel** (handlers do not hot-reload). Reference the atlas-tenants configurations REST surface (`GET/PATCH /tenants/{tenantId}/configurations/...`) and use repo-relative paths only.
2. **gms_v92 parked** — the `CharacterCashItemUseHandle` opcode and the point-reset body layout cannot be verified for v92 (no IDB, registry has no USE_CASH_ITEM row). The items stay inert on v92 exactly as today (they hit no handler). Unblocks when a v92 IDB exists (same precedent as the v92 mount-food park).
3. **New-tenant behavior** — seeded automatically from the updated templates.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/ docs/tasks/task-126-ap-sp-reset-items/deployment.md
git commit -m "feat(config): wire CharacterCashItemUseHandle on gms_87/gms_95/jms_185 seeds (task-126)"
```

---

### Task 17: Final verification gates

**Files:** none (verification only; fix-and-rebuild as needed).

- [ ] **Step 1: Per-module test/vet/build**

```bash
for m in libs/atlas-constants libs/atlas-packet libs/atlas-saga \
         services/atlas-character/atlas.com/character \
         services/atlas-skills/atlas.com/skills \
         services/atlas-channel/atlas.com/channel \
         services/atlas-saga-orchestrator/atlas.com/saga-orchestrator \
         tools/packet-audit; do
  echo "=== $m ==="
  (cd "$m" && go test -race ./... && go vet ./... && go build ./...) || exit 1
done
```
Expected: all clean.

- [ ] **Step 2: Packet + redis gates**

```bash
go run ./tools/packet-audit matrix --check
tools/redis-key-guard.sh
```
Expected: both exit 0.

- [ ] **Step 3: Docker bakes (mandatory for every changed service)**

```bash
docker buildx bake atlas-character atlas-skills atlas-channel atlas-saga-orchestrator atlas-configurations
```
Expected: all images build. (The shared libs changed, so any missing `COPY libs/...` line in the shared Dockerfile surfaces here, not in `go build`.)

- [ ] **Step 4: Commit any fixes, then final status**

Report per-gate results with actual command output. Do not claim done on a spot-check.

---

## Self-Review Notes (performed at plan time)

- **Spec coverage:** design §2.1 (reuse hpMpUsed + Build() bug) → Task 6; §2.2 (caps/floors) → Tasks 7/8/13; §2.3 (template wiring + v92 park) → Task 16; §2.4 (23/24 quirk) → Task 15; §2.5 (macro consumer) → Task 12; §2.6 (TargetMaxLevel in payload) → Tasks 10/13/15; §3 (saga shape B) → Tasks 11/15; §4.1 (constants) → Tasks 1/2; §4.2 (codec + verification) → Tasks 3/4; §4.3 (handler/pre-validation/feedback) → Tasks 13/14/15; §4.4 (saga lib + orchestrator) → Tasks 5/11; §4.5 (TRANSFER_AP) → Tasks 7/8; §4.6 (TRANSFER_SP) → Tasks 9/10; §4.7 (seeds) → Task 16; §6 (messages) → Task 13; §7 (tables) → Task 7; §8 (testing) → embedded per task; §9/§10 (resolutions/limitations) → Tasks 4/10/13/16.
- **Known intentional deviations from design naming:** `TransferAP`/`TransferSP` (matching the lib's `RebalanceAP` capitalization) instead of design's `TransferAp`/`TransferSp` — cosmetic; JSON action strings are exactly `transfer_ap`/`transfer_sp`.
- **Error-detail threading decision (plan-level):** the service ERROR events carry `{Error, Detail}`; orchestrator consumers store both in the failed step's result map; `compensatePointReset` emits `EmitSagaFailed(errorCode, reason=errorDetail)`; the channel failed-event branch passes `Body.Reason` as the detail to `pointreset.ErrorMessage`. `ErrorMessage` guards against non-ability Reason strings by falling back to the generic message.
- **Type consistency spot-checks:** `job.Id`=uint16, `skill.Id`=uint32 (constants.go); payload uses typed ids, service command bodies use uint32 skill ids matching existing skill message bodies; `TransferAPCommandBody.From/To` strings equal `CommandDistributeApAbility*` equal `pointreset.Ability*`.
