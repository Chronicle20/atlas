# JMS 185 CharacterData Divergences — Master Level & Extended SP — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the client's `is_skill_need_master_level` and extended-SP job predicates into `libs/atlas-constants/job` as version-resolved functions, and rewire `CharacterData` encode/decode to them, so JMS 185 and GMS v92/v95 emit the byte shape their clients read.

**Architecture:** Both predicates move to `libs/atlas-constants/job` (which already imports `skill`, so no cycle) and take `(region string, major uint16)` — the decomposed-scalar idiom `constants.For` already uses, keeping `atlas-constants` free of an `atlas-tenant` module edge. `skill.NeedsMasterLevel` and `character.isEvanJob` are deleted, not forwarded. Version arms are a `switch` with an IDA address cited per arm, not a generated table.

**Tech Stack:** Go 1.27, modules `libs/atlas-constants`, `libs/atlas-packet`, `services/atlas-channel/atlas.com/channel`.

**Spec:** `docs/tasks/task-275-jms185-characterdata-divergences/design.md` (PRD: `prd.md`)

## Global Constraints

- Every version arm carries an IDA address in a code comment (design §2, PRD NFR).
- The predicates run per skill inside an encode loop: integer arithmetic and `switch` only, no allocation, no logging.
- `libs/atlas-constants/go.mod` must keep exactly one non-indirect dependency, `github.com/google/uuid`. Do **not** add `libs/atlas-tenant`.
- `region` is compared case-insensitively via `strings.ToUpper`, matching `libs/atlas-constants/constants/for.go:40`.
- No Kafka topic, REST route, migration, or UI change.
- Module roots (`go build`/`go test` cwd): `libs/atlas-constants`, `libs/atlas-packet`, `services/atlas-channel/atlas.com/channel`.
- Design D1 deliberately departs from PRD FR-1's "takes `tenant.Model`": the signature is `(region string, major uint16)`. Do not "fix" this back.
- Deliberate byte change on GMS v95 only (design §5, §2.3); every other version/job an Atlas tenant can create must be byte-identical.

---

## Task 1: Capture the FR-9 no-byte-movement goldens (before any code change)

These goldens must be captured from the **current, unmodified** encoder. They are the regression guard proving the fix moves no byte for a creatable non-Evan, non-Dual-Blade character on the already-verified GMS cells. Task 1 must land before Tasks 2-8.

The character is job 312 (Ranger, a real Atlas identity) with two skills chosen so that no arm added by this task can touch them:

- `3110000` — job 311, 3rd job, never carries a master level on any version.
- `3121002` — job 312, 4th job (`312%10 == 2`), carries one on every version, and is **not** one of the sixteen `is_ignore_master_level_for_common` ids (the job-312 members are `3120010` and `3120011`).

### Files

- `libs/atlas-packet/character/data_golden_test.go` — **new file**; the two goldens
- `libs/atlas-packet/character/data_evan_test.go` — read-only; copy the `mk(...)` + `pt.CreateContext` fixture shape from here
- `libs/atlas-packet/test/context.go` — read-only; `pt.CreateContext`, `pt.Encode`

Module root: `libs/atlas-packet`.

Patterns to copy: `libs/atlas-packet/character/data_evan_test.go:40-73` (fixture construction and `pt.Encode` use); `libs/atlas-packet/field/clientbound/set_field_test.go:66-78` (byte-literal golden compared with `bytes.Equal`).

- [ ] **Step 1: Write the golden test with an empty expected slice**

Create `libs/atlas-packet/character/data_golden_test.go`. Write the exact structure below, leaving `wantHex` as the empty string for both cases — Step 2 fills them in from the current encoder.

```go
package character

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestCharacterDataNoByteMovement is the FR-9 regression guard: for a job an
// Atlas tenant can create today that is neither an Evan (22xx / 2001) nor a
// Dual Blade (43x), and whose skills are outside GMS v95's
// is_ignore_master_level_for_common list (@0x47cc20), the CharacterData wire
// bytes must be identical before and after task-275.
//
// The goldens below were captured from the encoder as it stood at the parent
// commit of task-275's first code change. If a later change to a version arm
// moves one of these bytes, that arm is reaching a character it must not.
func TestCharacterDataNoByteMovement(t *testing.T) {
	mk := func() CharacterData {
		return CharacterData{
			Stats: CharacterStats{
				Id: 1000, Name: "Ranger", Gender: 0, SkinColor: 1,
				Face: 20000, Hair: 30000,
				Level: 120, JobId: 312, Str: 100, Dex: 250, Int: 30, Luk: 20,
				Hp: 5000, MaxHp: 5000, Mp: 3000, MaxMp: 3000,
				Ap: 5, Sp: 3, Exp: 50000, Fame: 10,
				GachaExp: 0, MapId: 100000000, SpawnPoint: 0,
			},
			BuddyCapacity: 20,
			Meso:          100000,
			Inventory: InventoryData{
				EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
				EtcCapacity: 24, CashCapacity: 24,
				EquipSlotExtExpire: 94354848000000000,
			},
			Skills: []SkillEntry{
				// job 311 (3rd job) — no master level on any version.
				{Id: 3110000, Level: 20, Expiration: -1},
				// job 312 (4th job) — master level on every version; NOT one of
				// the sixteen v95 ignore-list ids (312's are 3120010/3120011).
				{Id: 3121002, Level: 30, Expiration: -1, MasterLevel: 30},
			},
		}
	}

	for _, c := range []struct {
		name    string
		region  string
		major   uint16
		wantHex string
	}{
		{"GMS v84", "GMS", 84, ""},
		{"GMS v95", "GMS", 95, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			cd := mk()
			got := hex.EncodeToString(pt.Encode(t, pt.CreateContext(c.region, c.major, 1), cd.Encode, nil))
			if got != c.wantHex {
				t.Errorf("%s CharacterData golden mismatch:\n got %s\nwant %s", c.name, got, c.wantHex)
			}
		})
	}
}
```

- [ ] **Step 2: Run it against the CURRENT encoder and paste the captured hex in**

Run from `libs/atlas-packet`:

```
go test ./character/ -run TestCharacterDataNoByteMovement -v
```

Expected: two FAILs, each printing `got <hex>` with an empty `want`. Copy each `got` value verbatim into the matching `wantHex` field. Do **not** hand-derive these strings; they are only meaningful if they came out of the unmodified encoder.

- [ ] **Step 3: Re-run to verify both pass**

```
go test ./character/ -run TestCharacterDataNoByteMovement -v
```

Expected: PASS for both `GMS_v84` and `GMS_v95`.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-packet/character/data_golden_test.go
git commit -m "test(character): pin pre-change CharacterData goldens for GMS v84/v95 (task-275 FR-9)"
```

---

## Task 2: `job.ClientJobLevel`

A verbatim port of the client's `get_job_level`. It is pure integer arithmetic — no table, no WZ data, and explicitly **not** `job.Advancement`, which returns -1 for the whole Evan stage line and has no 43x branch.

### Files

- `libs/atlas-constants/job/master_level.go` — **new file**; `ClientJobLevel` only in this task
- `libs/atlas-constants/job/master_level_test.go` — **new file**; `TestClientJobLevel` only in this task
- `libs/atlas-constants/job/advancement.go` — read-only; the function this must NOT be confused with
- `libs/atlas-constants/job/constants.go` — read-only; `type Id uint16`, `EvanStage1Id`..`EvanStage10Id`

Module root: `libs/atlas-constants`.

Patterns to copy: `libs/atlas-constants/job/advancement.go:1-23` (package-level doc-comment-plus-function shape); `libs/atlas-constants/skill/model_test.go:57-102` (named-case table with `t.Run`).

**Interfaces produced:** `func ClientJobLevel(jobId Id) int`, package `job`.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-constants/job/master_level_test.go` with `TestClientJobLevel` — table-driven, one `t.Run` per named case, using the `{name, jobId, want}` struct shape of `skill/model_test.go:57-102`. Cases, in full:

| name | jobId | want |
|---|---|---|
| beginner 0 | 0 | 1 |
| warrior root 100 | 100 | 1 |
| magician root 200 | 200 | 1 |
| Evan beginner 2001 | 2001 | 1 |
| Evan stage 1 (2200) | 2200 | 1 |
| fighter 110 | 110 | 2 |
| crusader 111 | 111 | 3 |
| hero 112 | 112 | 4 |
| tier-5 non-Evan 113 | 113 | 0 |
| Aran 4th 2112 | 2112 | 4 |
| Aran tier-5 2113 | 2113 | 0 |
| Blade Recruit 430 | 430 | 2 |
| Blade Acolyte 431 | 431 | 2 |
| Blade Specialist 432 | 432 | 3 |
| Blade Lord 433 | 433 | 3 |
| Blade Master 434 | 434 | 4 |
| Evan stage 2 (2210) | 2210 | 2 |
| Evan stage 3 (2211) | 2211 | 3 |
| Evan stage 5 (2213) | 2213 | 5 |
| Evan stage 8 (2216) | 2216 | 8 |
| Evan stage 9 (2217) | 2217 | 9 |
| Evan stage 10 (2218) | 2218 | 10 |

Include this comment above the table, verbatim:

```go
// The GMS and JMS clients spell the 43x branch differently — GMS v95
// @0x47cb90 and GMS v92 @0x479260 compute (job-430)/2, JMS 185 @0x47d347
// computes (job%10)/2 — but they agree on every value in 430..434
// (0,0,1,1,2) and diverge only at job >= 435, which no client defines.
// This ports the GMS form; the cases below pin the agreed values.
```

- [ ] **Step 2: Run it to verify it fails**

From `libs/atlas-constants`:

```
go test ./job/ -run TestClientJobLevel -v
```

Expected: FAIL — `undefined: ClientJobLevel`.

- [ ] **Step 3: Write the implementation**

Create `libs/atlas-constants/job/master_level.go`:

```go
package job

// ClientJobLevel is a verbatim port of the client's get_job_level(nJob)
// (GMS v92 sub_479260 @0x479260, GMS v95 @0x47cb90, JMS 185 @0x47d347).
//
// It is NOT Advancement. Advancement returns -1 for the whole Evan stage
// line and has no 43x branch; the client's helper returns 2..10 for Evan
// growths and 2..4 for Dual Blades, and both of those are load-bearing for
// is_skill_need_master_level. Reusing Advancement here would be the same
// class of mistake as the job.GetSkillBook off-by-one that task-218 hit.
//
//	if job%100 == 0 || job == 2001: return 1
//	v := job%10;  if job/10 == 43 { v = (job-430)/2 }
//	lvl := v + 2
//	return lvl if lvl >= 2 && (lvl <= 4 || (lvl <= 10 && isEvanJob(job))) else 0
//
// The 43x expression differs between regions — GMS computes (job-430)/2,
// JMS 185 (@0x47d347) computes (job%10)/2 — but the two agree over the whole
// defined range 430..434 (0,0,1,1,2) and diverge only at job >= 435, which no
// client defines. Modelled as the GMS form; do not branch on region here.
func ClientJobLevel(jobId Id) int {
	if jobId%100 == 0 || jobId == 2001 {
		return 1
	}
	v := int(jobId % 10)
	if jobId/10 == 43 {
		v = (int(jobId) - 430) / 2
	}
	lvl := v + 2
	if lvl < 2 {
		return 0
	}
	if lvl <= 4 {
		return lvl
	}
	if lvl <= 10 && isEvanJob(jobId) {
		return lvl
	}
	return 0
}

// isEvanJob is the client's is_evan_job (GMS v95 @0x47cad0): the Evan growth
// band plus the Evan beginner. JMS 185 inlines only job/100 == 22, which is
// equivalent at its call sites because job == 2001 has already returned.
func isEvanJob(jobId Id) bool {
	return jobId/100 == 22 || jobId == 2001
}
```

- [ ] **Step 4: Run the test to verify it passes**

```
go test ./job/ -run TestClientJobLevel -v
```

Expected: PASS, all 22 subtests.

- [ ] **Step 5: Build and vet the module**

```
go build ./... && go vet ./...
```

Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-constants/job/master_level.go libs/atlas-constants/job/master_level_test.go
git commit -m "feat(job): port the client get_job_level as ClientJobLevel"
```

---

## Task 3: `job.NeedsMasterLevel` with its three version arms

The version-resolved port of `is_skill_need_master_level`. It supersedes `skill.NeedsMasterLevel`, which Task 6 deletes; leave the old function alone in this task.

### Files

- `libs/atlas-constants/job/master_level.go` — modify; append the three arm helpers, the sixteen-id ignore list, and `NeedsMasterLevel`
- `libs/atlas-constants/job/master_level_test.go` — new file created in Task 2; append four test functions here
- `libs/atlas-constants/skill/model.go` — read-only in this task; `skill.Id`, `skill.Is` (`:148`), and the comment block being superseded (`:78-114`)
- `libs/atlas-constants/job/model.go` — read-only; confirms `job` already imports `skill`, so this direction is cycle-free

Module root: `libs/atlas-constants`.

Patterns to copy: `libs/atlas-constants/skill/model.go:78-135` (IDA-cited doc comment above a ported predicate); `libs/atlas-constants/skill/model_test.go:57-102` (named-case table).

**Interfaces consumed:** `ClientJobLevel(Id) int`, `isEvanJob(Id) bool` (Task 2).
**Interfaces produced:** `func NeedsMasterLevel(skillId skill.Id, region string, major uint16) bool`, package `job`.

- [ ] **Step 1: Write the failing tests**

Append to `libs/atlas-constants/job/master_level_test.go`. Four functions; every one iterates the six version columns `{"GMS",83}, {"GMS",84}, {"GMS",87}, {"GMS",92}, {"GMS",95}, {"JMS",185}` and uses a `t.Run(fmt.Sprintf("%s_v%d/%s", region, major, caseName), ...)` subtest name.

**`TestNeedsMasterLevel_DualBladePerVersion`** — the design §2.1 grid. Generic per-job skill ids (`job*10000`), then the four named ids:

| skill | job | GMS 83 | GMS 84 | GMS 87 | GMS 92 | GMS 95 | JMS 185 |
|---|---|---|---|---|---|---|---|
| 4300000 | 430 | false | false | false | false | false | false |
| 4310000 | 431 | false | false | false | false | false | false |
| 4320000 | 432 | **true** | **true** | false | false | false | false |
| 4330000 | 433 | false | false | false | false | false | false |
| 4340000 | 434 | false | false | false | **true** | **true** | **true** |
| 4311003 | 431 | false | false | false | **true** | **true** | **true** |
| 4321000 | 432 | **true** | **true** | false | **true** | **true** | **true** |
| 4331002 | 433 | false | false | false | **true** | **true** | **true** |
| 4331005 | 433 | false | false | false | **true** | **true** | **true** |

With this comment above the table, verbatim:

```go
// v83 (@0x4e8f04) and v84 (@0x4f0ad2) have no 43x arm at all, so these fall
// through to the common rule job%100 != 0 && job%10 == 2 — which is TRUE for
// job 432 and false for 430/431/433/434. That is the client's observable
// answer, not a modelling choice. v87 (@0x508fa4) returns false for all five.
// v92 (@0x479371), v95 (@0x47ccb0) and JMS 185 (@0x47d2f9) return
// ClientJobLevel(job) == 4 || skillId in {4311003,4321000,4331002,4331005}.
```

**`TestNeedsMasterLevel_EvanArm`** — the migrated cases from `skill/model_test.go:57-102`, given version columns:

| skill | GMS 83 | GMS 84 | GMS 87 | GMS 92 | GMS 95 | JMS 185 |
|---|---|---|---|---|---|---|
| 22171000 (9th growth) | true | true | true | true | true | true |
| 22181000 (10th growth) | true | true | true | true | true | true |
| 22001001 (1st growth) | false | false | false | false | false | false |
| 22131000 (5th growth) | false | false | false | false | false | false |
| 22141001 (6th growth) | false | false | false | false | false | false |
| 22151001 (7th growth) | false | false | false | false | false | false |
| 22161003 (Recovery Aura) | false | false | false | false | false | false |
| 22161000 (8th growth, skill-book guard) | false | false | false | false | false | false |
| 20010000 (Evan beginner 2001) | false | false | false | false | false | false |
| 22111001 (exception) | false | **true** | **true** | **true** | **true** | false |
| 22141002 (exception) | false | **true** | **true** | **true** | **true** | false |
| 22140000 (exception) | false | **true** | **true** | **true** | **true** | false |

Carry across the two comments this replaces: the exception list is present GMS v84+ (@0x4f0ad2, @0x508f33, @0x4792f0, @0x47ccb0) and absent on GMS v83 (@0x4e8f04) and JMS 185 (@0x47d2a8); and the 22161000/22171000 pair is the `job.GetSkillBook` off-by-one guard from `skill/model_test.go:103-113` (GetSkillBook indexes 2210→1 … 2218→9, the client's level is 2210→2 … 2218→10, so a book-indexed "9 or 10" would wrongly select 2218/2219).

**`TestNeedsMasterLevel_CommonBranch`** — the version-invariant fallthrough, asserted true on all six columns unless noted:

| skill | job | expect (all six columns) |
|---|---|---|
| 1120003 | 112 | true |
| 2321000 | 232 | true |
| 4120002 | 412 | true |
| 21120001 | 2112 | true (Aran 4th goes through the common branch, not the Evan one) |
| 1100000 | 110 | false |
| 1110000 | 111 | false |
| 21110000 | 2111 | false |
| 10000 | 0 | false |
| 1000000 | 100 | false |
| 2000000 | 200 | false |

**`TestNeedsMasterLevel_IgnoreCommonV95Only`** — all sixteen `is_ignore_master_level_for_common` ids (@0x47cc20):

```
1120012 1220013 1320011 2120009 2220009 2320010 3120010 3120011
3220009 3220010 4120010 4220009 5120011 5220012 32120009 33120010
```

Every one of the sixteen is `false` on `{"GMS",95}` and `true` on each of `{"GMS",83}`, `{"GMS",84}`, `{"GMS",87}`, `{"GMS",92}`, `{"JMS",185}` (each satisfies the common `job%10 == 2` rule). Comment above the list, verbatim:

```go
// Fourteen of these sixteen belong to jobs an Atlas tenant can create
// (112, 122, 132, 212, 222, 232, 312, 322, 412, 422, 512, 522), so before
// task-275 Atlas wrote a master-level int for them on GMS v95 where the
// client reads none — a live 4-byte-per-skill shift. The remaining two
// (jobs 3212, 3312) match no Atlas identity and are modelled only because
// the client's list is a flat id set and a partial port is not a port.
```

Also add a lower-case region case: `NeedsMasterLevel(skill.Id(4340000), "jms", 185)` is `true`, guarding the `strings.ToUpper` normalisation.

- [ ] **Step 2: Run the tests to verify they fail**

From `libs/atlas-constants`:

```
go test ./job/ -run 'TestNeedsMasterLevel' -v
```

Expected: FAIL — `undefined: NeedsMasterLevel`.

- [ ] **Step 3: Write the implementation**

Append to `libs/atlas-constants/job/master_level.go` (add `"strings"` and the `skill` import at the top of the file):

```go
// dualBladeShape is which of the three forms a client's 430-434 arm takes.
type dualBladeShape int

const (
	// dualBladeNone: no 43x arm at all; the id falls through to the common
	// rule, which is TRUE for job 432. GMS v83 @0x4e8f04, v84 @0x4f0ad2.
	dualBladeNone dualBladeShape = iota
	// dualBladeAlwaysFalse: the arm exists and returns 0 for all of 430-434.
	// GMS v87 @0x508fa4 (v1/10 == 43 || !(v1%100) -> return 0).
	dualBladeAlwaysFalse
	// dualBladeJobLevel: ClientJobLevel(job) == 4, or one of four named ids.
	// GMS v92 @0x479371, GMS v95 @0x47ccb0, JMS 185 @0x47d2f9.
	dualBladeJobLevel
)

// dualBladeArm reports which shape this client's 430-434 arm takes. The arms
// are monotone in major within a region, so an unprovisioned major takes the
// nearest lower provisioned arm rather than a baseline fallback.
func dualBladeArm(region string, major uint16) dualBladeShape {
	if strings.ToUpper(region) != "GMS" {
		return dualBladeJobLevel
	}
	switch {
	case major < 87:
		return dualBladeNone
	case major < 92:
		return dualBladeAlwaysFalse
	default:
		return dualBladeJobLevel
	}
}

// hasEvanExceptions reports whether this client's Evan arm carries the
// three-skill exception list {22111001, 22141002, 22140000}. Present on
// GMS v84 (@0x4f0ad2), v87 (@0x508f33), v92 (@0x4792f0) and v95 (@0x47ccb0);
// absent on GMS v83 (@0x4e8f04) and on JMS 185 (@0x47d2a8).
func hasEvanExceptions(region string, major uint16) bool {
	return strings.ToUpper(region) == "GMS" && major >= 84
}

// ignoresCommonMasterLevel reports whether this client early-outs on
// is_ignore_master_level_for_common. GMS v95 @0x47cc20 is the only client
// Atlas has read that carries it; >= 95 rather than == 95 because the arms
// are monotone in major within a region.
func ignoresCommonMasterLevel(region string, major uint16) bool {
	return strings.ToUpper(region) == "GMS" && major >= 95
}

// ignoredCommonMasterLevelSkills is the flat membership of GMS v95's
// is_ignore_master_level_for_common (@0x47cc20). Its callers see a false
// is_skill_need_master_level for every member.
//
// Fourteen of the sixteen belong to jobs an Atlas tenant can create
// (112, 122, 132, 212, 222, 232, 312, 322, 412, 422, 512, 522). The other
// two (jobs 3212, 3312) match no Atlas identity and are modelled only
// because the client's list is a flat id set and a partial port is not one.
var ignoredCommonMasterLevelSkills = map[skill.Id]struct{}{
	1120012: {}, 1220013: {}, 1320011: {},
	2120009: {}, 2220009: {}, 2320010: {},
	3120010: {}, 3120011: {}, 3220009: {}, 3220010: {},
	4120010: {}, 4220009: {},
	5120011: {}, 5220012: {},
	32120009: {}, 33120010: {},
}

// NeedsMasterLevel reports whether a skill entry in GW_CharacterData carries
// the trailing 4-byte master level. It is a direct port of the client's
// is_skill_need_master_level(nSkillID) (GMS v83 @0x4e8f04, v84 @0x4f0ad2,
// v87 @0x508f33, v92 @0x4792f0, v95 @0x47ccb0, JMS 185 @0x47d2a8), which is
// the ONLY authority: the field is per-SKILL, and approximating it with a
// per-JOB test is what produced the task-218 field report (a preset Evan
// closed the client with error 38 while a level-1 Evan logged in fine,
// because the length only diverges once the character owns skills).
//
// Because the field is not length-prefixed, answering it differently than
// the client does not produce a wrong value — it shifts every subsequent
// field of GW_CharacterData by four bytes.
//
// region/major select the arms; region is matched case-insensitively, the
// same normalisation constants.For applies (constants/for.go:40). The arms
// are monotone in major within a region, so an unprovisioned major takes the
// nearest lower provisioned arm; there is no baseline fallback and no
// logging, because this runs once per skill inside an encode loop.
func NeedsMasterLevel(skillId skill.Id, region string, major uint16) bool {
	if ignoresCommonMasterLevel(region, major) {
		if _, ok := ignoredCommonMasterLevelSkills[skillId]; ok {
			return false
		}
	}

	jobId := Id(uint32(skillId) / 10000)

	if isEvanJob(jobId) {
		// Growths 9 and 10 (jobs 2217, 2218) on every client read.
		if lvl := ClientJobLevel(jobId); lvl == 9 || lvl == 10 {
			return true
		}
		if !hasEvanExceptions(region, major) {
			return false
		}
		return skill.Is(skillId, skill.Id(22111001), skill.Id(22141002), skill.Id(22140000))
	}

	if jobId/10 == 43 {
		switch dualBladeArm(region, major) {
		case dualBladeAlwaysFalse:
			return false
		case dualBladeJobLevel:
			return ClientJobLevel(jobId) == 4 ||
				skill.Is(skillId, skill.Id(4311003), skill.Id(4321000), skill.Id(4331002), skill.Id(4331005))
		case dualBladeNone:
			// No arm: fall through to the common rule below, which is what
			// GMS v83/v84 do — and which is TRUE for job 432.
		}
	}

	if jobId%100 == 0 {
		return false
	}
	return jobId%10 == 2
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```
go test ./job/ -run 'TestNeedsMasterLevel|TestClientJobLevel' -v
```

Expected: PASS.

- [ ] **Step 5: Build and vet the module**

```
go build ./... && go vet ./...
```

Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-constants/job/master_level.go libs/atlas-constants/job/master_level_test.go
git commit -m "feat(job): version-resolved NeedsMasterLevel with Dual Blade and v95 ignore-list arms"
```

---

## Task 4: `job.UsesExtendedSP`

### Files

- `libs/atlas-constants/job/extended_sp.go` — **new file**
- `libs/atlas-constants/job/extended_sp_test.go` — **new file**
- `libs/atlas-packet/character/data.go:268-272` — read-only; the `isEvanJob` this supersedes (deleted in Task 5)

Module root: `libs/atlas-constants`.

Patterns to copy: `libs/atlas-constants/job/master_level.go` as written in Task 3 (same arm-comment style); `libs/atlas-packet/character/data_evan_test.go:13-34` (the case list being absorbed).

**Interfaces produced:** `func UsesExtendedSP(jobId Id, region string, major uint16) bool`, package `job`.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-constants/job/extended_sp_test.go` with `TestUsesExtendedSP`. Same six version columns as Task 3, `t.Run` per (column, job). Full grid:

| jobId | GMS 83 | GMS 84 | GMS 87 | GMS 92 | GMS 95 | JMS 185 |
|---|---|---|---|---|---|---|
| 2001 (Evan beginner) | false | true | true | true | true | **true** |
| 2200 | false | true | true | true | true | **true** |
| 2210 | false | true | true | true | true | **true** |
| 2218 | false | true | true | true | true | **true** |
| 2299 | false | true | true | true | true | **true** |
| 3000 (Resistance band) | false | false | false | **true** | **true** | **true** |
| 3999 (Resistance band) | false | false | false | **true** | **true** | **true** |
| 0 | false | false | false | false | false | false |
| 100 | false | false | false | false | false | false |
| 312 | false | false | false | false | false | false |
| 322 | false | false | false | false | false | false |
| 430 (Dual Blade — FR-10) | false | false | false | false | false | false |
| 434 (Dual Blade — FR-10) | false | false | false | false | false | false |
| 2000 | false | false | false | false | false | false |
| 2100 | false | false | false | false | false | false |
| 2300 | false | false | false | false | false | false |

Plus a lower-case-region case: `UsesExtendedSP(2218, "jms", 185)` is `true`.

Comments to carry, verbatim:

```go
// The 3000-3999 arm (job/1000 == 3) is pinned per FR-7 even though no Atlas
// job identity lands in that band today — the only 3xx identities are Bowman
// 300 .. Marksman 322, all of which are < 1000 after the divide. Pinning it
// means a future Resistance bring-up inherits the client's rule instead of
// rediscovering it.

// FR-10: a Dual Blade is NOT an extended-SP job (430/1000 == 0). It takes the
// plain SP short while still taking the per-skill master-level int, so the two
// task-275 fixes must not be conflated.
```

- [ ] **Step 2: Run it to verify it fails**

```
go test ./job/ -run TestUsesExtendedSP -v
```

Expected: FAIL — `undefined: UsesExtendedSP`.

- [ ] **Step 3: Write the implementation**

Create `libs/atlas-constants/job/extended_sp.go`:

```go
package job

import "strings"

// UsesExtendedSP reports whether GW_CharacterStat carries the variable-length
// extended-SP block (a Decode1 count followed by count x (Decode1
// masterLevelIdx, Decode1 sp)) in place of the single SP Decode2.
//
// Because the divergence is not length-prefixed at the point of the branch,
// answering it differently than the client shifts every field after SP.
//
// Client reads, per version:
//
//	GMS <= 83                    no extended-SP path at all (Evan launched at v84)
//	GMS v84  inline @0x4e9da4    job/100 == 22 || job == 2001
//	GMS v87  inline @0x501e9c    job/100 == 22 || job == 2001
//	GMS v92  inline @0x4f50f4 / @0x4f5100 / @0x4f510f
//	                             job/1000 == 3 || job/100 == 22 || job == 2001
//	GMS v95  is_extendsp_job @0x4f1e30   same as v92
//	JMS 185  sub_5163A2 @0x5163a2, called @0x50eda2   same as v92
//
// region is matched case-insensitively, the same normalisation constants.For
// applies (constants/for.go:40). The arms are monotone in major within a
// region, so an unprovisioned major takes the nearest lower provisioned arm.
//
// The job/1000 == 3 arm is modelled although no Atlas job identity reaches it
// today (the only 3xx identities are Bowman 300 .. Marksman 322, all < 1000
// after the divide); a future Resistance bring-up must inherit the client's
// rule, not rediscover it. A Dual Blade (43x) is deliberately NOT an
// extended-SP job: 430/1000 == 0, so it keeps the plain SP short.
func UsesExtendedSP(jobId Id, region string, major uint16) bool {
	if strings.ToUpper(region) == "GMS" {
		if major < 84 {
			return false
		}
		if major < 92 {
			return jobId/100 == 22 || jobId == 2001
		}
	}
	return jobId/1000 == 3 || jobId/100 == 22 || jobId == 2001
}
```

- [ ] **Step 4: Run the test to verify it passes**

```
go test ./job/ -run TestUsesExtendedSP -v
```

Expected: PASS.

- [ ] **Step 5: Run the whole module's tests**

```
go build ./... && go vet ./... && go test ./...
```

Expected: all PASS (`skill.NeedsMasterLevel` and its tests are still present and untouched).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-constants/job/extended_sp.go libs/atlas-constants/job/extended_sp_test.go
git commit -m "feat(job): add version-resolved UsesExtendedSP predicate"
```

---

## Task 5: Rewire `libs/atlas-packet/character` onto the new predicates

Both gates and both call sites move in one commit, because encode and decode must never be on different authorities (FR-8). `skill.NeedsMasterLevel` stays in the tree until Task 6 so both modules keep compiling at every commit.

### Files

- `libs/atlas-packet/character/data.go` — modify: delete `isEvanJob` (`:268-272`), retarget the extended-SP gate at `:316` (encode) and `:397` (decode), retarget the master-level call at `:670` (encode) and `:696` (decode), add the `job` import
- `libs/atlas-packet/character/data_evan_test.go` — modify: delete `TestIsEvanJob` (`:13-34`) and the now-unused nothing-else; keep `TestEvanExtendedSPv84` and `TestEvanSkillMasterLevelLength` unchanged
- `libs/atlas-packet/character/data_test.go:275-276` — modify: the two `skill.NeedsMasterLevel(..., true)` assertions inside `TestCharacterDataWithSkillsRoundTrip`
- `libs/atlas-packet/character/data_golden_test.go` — new file created in Task 1; read-only here, must still pass unchanged (the FR-9 guard)

Module root: `libs/atlas-packet`.

Patterns to copy: `libs/atlas-packet/character/data.go:294-300` (the existing `t.IsRegion("GMS") && t.MajorAtLeast(61)` gate-with-IDA-comment shape being replaced).

**Interfaces consumed:** `job.NeedsMasterLevel(skill.Id, string, uint16) bool` (Task 3), `job.UsesExtendedSP(job.Id, string, uint16) bool` (Task 4).

- [ ] **Step 1: Delete `isEvanJob` and `TestIsEvanJob`**

Remove `libs/atlas-packet/character/data.go:268-272` (the `isEvanJob` doc comment and function) and `libs/atlas-packet/character/data_evan_test.go:13-34` (`TestIsEvanJob`). The cases from that test now live in `job/extended_sp_test.go` (Task 4). `data_evan_test.go` keeps its remaining imports — `testlog`, `pt`, `response`, `tenant` are all still used by the two surviving tests.

- [ ] **Step 2: Retarget the four call sites**

Add to the import block of `data.go` (alphabetically, next to the existing `item` / `_map` / `skill` atlas-constants imports):

```go
"github.com/Chronicle20/atlas/libs/atlas-constants/job"
```

Encode gate, replacing `data.go:316`:

```go
if job.UsesExtendedSP(job.Id(m.Stats.JobId), t.Region(), t.MajorVersion()) {
	// Extended SP: byte count + count x (masterLevelIdx, sp) byte-pairs
	// (GW_CharacterStat::DecodeExtendSP; JMS 185 sub_50E8B0 @0x50e8b0,
	// count @0x50e8c6, pair @0x50e8de/@0x50e8e0). Atlas has no
	// per-master-level SP allocation model, so the count is always 0.
	// See job.UsesExtendedSP for the per-version predicate.
	w.WriteByte(0)
} else {
	w.WriteShort(m.Stats.Sp)
}
```

Decode gate, replacing `data.go:397` (body otherwise unchanged — the discard loop must stay, it has to consume a nonzero count from a client-authored packet):

```go
if job.UsesExtendedSP(job.Id(m.Stats.JobId), t.Region(), t.MajorVersion()) {
	// Mirror of encodeStats — one authority, so the two cannot diverge.
	count := r.ReadByte()
	for i := byte(0); i < count; i++ {
		_ = r.ReadByte() // master-level index
		_ = r.ReadByte() // sp
	}
} else {
	m.Stats.Sp = r.ReadUint16()
}
```

Encode master level, replacing `data.go:670`:

```go
if job.NeedsMasterLevel(skill.Id(s.Id), t.Region(), t.MajorVersion()) {
	w.WriteInt(s.MasterLevel)
}
```

Decode master level, replacing `data.go:696`:

```go
m.Skills[i].NeedsMasterLevel = job.NeedsMasterLevel(skill.Id(m.Skills[i].Id), t.Region(), t.MajorVersion())
```

Update the two surrounding comments (`data.go:667-669` and `data.go:694-695`) so they name `job.NeedsMasterLevel` rather than `skill.NeedsMasterLevel`. Also update the `SkillEntry.NeedsMasterLevel` field comment at `data.go:69-74`, which names `skill.NeedsMasterLevel`.

- [ ] **Step 3: Retarget the round-trip assertion in `data_test.go`**

At `libs/atlas-packet/character/data_test.go:275-276`, inside `TestCharacterDataWithSkillsRoundTrip`'s `for _, v := range pt.Variants` loop, replace both `skill.NeedsMasterLevel(skill.Id(input.Skills[i].Id), true)` expressions with:

```go
job.NeedsMasterLevel(skill.Id(input.Skills[i].Id), v.Region, v.MajorVersion)
```

Add the `job` import to `data_test.go`. `skill` stays imported — `skill.Id(...)` is still used. The two fixture skills are `2101001` (job 210, no master level) and `2121006` (job 212, 4th job, master level on every version; **not** one of the sixteen v95 ignore-list ids, whose job-212 member is `2120009`), so this loop's expected answers are unchanged on every variant.

- [ ] **Step 4: Run the package's tests**

From `libs/atlas-packet`:

```
go build ./... && go vet ./... && go test ./character/ -v
```

Expected: PASS. In particular `TestCharacterDataNoByteMovement` (Task 1's FR-9 guard) must still pass for both `GMS v84` and `GMS v95` — if it fails, an arm is reaching job 312 and the arm is wrong, not the golden.

- [ ] **Step 5: Run the whole module**

```
go test ./...
```

Expected: PASS, including `libs/atlas-packet/field/clientbound` (SET_FIELD's goldens re-derive the CharacterData span by calling `cd.Encode`, and their fixtures are job 312 with no skills, so they are unaffected).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/character/data.go libs/atlas-packet/character/data_test.go libs/atlas-packet/character/data_evan_test.go
git commit -m "fix(character): resolve master-level and extended-SP gates per client version"
```

---

## Task 6: Delete `skill.NeedsMasterLevel`

A straightforward move, not a re-export: per CLAUDE.md, no forwarding wrapper is left behind. Task 5 removed the last non-test caller; re-run the grep rather than trusting that line.

### Files

- `libs/atlas-constants/skill/model.go` — modify: delete the `NeedsMasterLevel` function (`:115-129`) and its 37-line doc comment (`:78-114`)
- `libs/atlas-constants/skill/model_test.go` — modify: delete `TestNeedsMasterLevelMatchesClientRule` (`:47-102`) and `TestNeedsMasterLevelNotSkillBookIndexed` (`:103-113`); both are superseded by `job/master_level_test.go` (Task 3)

Module root: `libs/atlas-constants`.

- [ ] **Step 1: Confirm there is no remaining caller**

```
grep -rn 'skill\.NeedsMasterLevel\|[^.]\bNeedsMasterLevel(' --include='*.go' libs services
```

Expected: only `libs/atlas-constants/skill/model.go`, `libs/atlas-constants/skill/model_test.go`, and the comment references at `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go:76,80` (Task 8 fixes those). `libs/atlas-packet/character` must show only `job.NeedsMasterLevel` and the `SkillEntry.NeedsMasterLevel` field. If anything else appears, stop and report — Task 5 missed a call site.

- [ ] **Step 2: Delete the function, its comment, and its two tests**

Remove `libs/atlas-constants/skill/model.go:78-129` in full. Remove `libs/atlas-constants/skill/model_test.go:47-113` in full. `model_test.go` keeps `TestIsKeyDownSkill` and its `import "testing"`; `model.go` keeps every other function (`IsBuff`, `NeedsCharging`, `IsShootSkillNotUsingShootingWeapon`, `IsShootSkillNotConsumingBullet`, `IsKeyDownSkill`, `IsGrenadeSkill`, `Is`).

- [ ] **Step 3: Build, vet and test both affected modules**

From `libs/atlas-constants`:

```
go build ./... && go vet ./... && go test ./...
```

From `libs/atlas-packet`:

```
go build ./... && go test ./...
```

Expected: PASS in both.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-constants/skill/model.go libs/atlas-constants/skill/model_test.go
git commit -m "refactor(skill): drop NeedsMasterLevel, superseded by job.NeedsMasterLevel"
```

---

## Task 7: Byte-exact `CharacterData` fixtures for the corrected shapes

Layers 2 and 3 of design §6 — the fixtures that prove the fix in bytes rather than in predicate booleans.

### Files

- `libs/atlas-packet/character/data_master_level_test.go` — **new file**
- `libs/atlas-packet/character/data_evan_test.go` — read-only; `TestEvanSkillMasterLevelLength` (`:91-122`) is the `encodeSkills`-length pattern these copy
- `libs/atlas-packet/character/data_golden_test.go` — new file created in Task 1; read-only here, must still pass

Module root: `libs/atlas-packet`.

Patterns to copy: `libs/atlas-packet/character/data_evan_test.go:91-122` (direct `encodeSkills` byte-length assertion with `response.NewWriter` and `tenant.MustFromContext(pt.CreateContext(...))`); `libs/atlas-packet/character/data_evan_test.go:41-73` (full-`CharacterData` `mk(...)` + length-delta + `pt.RoundTrip`).

- [ ] **Step 1: Write the fixtures**

Create `libs/atlas-packet/character/data_master_level_test.go` with five test functions.

**`TestSkillsDualBladeMasterLevelJMS`** — direct `encodeSkills`, mirroring `data_evan_test.go:91-122`. Skills, in order:

```go
cd := CharacterData{Skills: []SkillEntry{
	{Id: 4300000, Level: 20}, // job 430 — no master level on any version
	{Id: 4340000, Level: 30}, // job 434 — ClientJobLevel 4: master level on v92/v95/JMS
	{Id: 4311003, Level: 20}, // job 431 — named exception id, master level on v92/v95/JMS
}}
```

Base length for a 3-skill list on a version that writes the 8-byte expiration:
`const base = 2 + 3*(4+4+8) + 2` = 52 (count 2 + three x (id 4 + level 4 + expiration 8) + cooldownCount 2).

| tenant | expected `len(w.Bytes())` | why |
|---|---|---|
| `("JMS", 185, 1)` | `base + 2*4` = 60 | 434 and 4311003 carry it (@0x47d2f9); 430 does not |
| `("GMS", 95, 1)` | `base + 2*4` = 60 | same arm (@0x47ccb0); none of the three is in the ignore list |
| `("GMS", 87, 1)` | `base + 0` = 52 | the v87 arm returns false for all of 430-434 (@0x508fa4) |
| `("GMS", 83, 1)` | `base + 0` = 52 | no arm; common rule — 430 %10==0, 434 %10==4, 431 %10==1, all false |

Each failure message must name the version and the arm, in the style of `data_evan_test.go:112`.

**`TestSkillsIgnoreCommonV95`** — direct `encodeSkills`:

```go
cd := CharacterData{Skills: []SkillEntry{
	{Id: 1120012, Level: 30}, // job 112 — in is_ignore_master_level_for_common @0x47cc20
	{Id: 1120003, Level: 30}, // job 112 — NOT in the list; ordinary 4th-job rule
}}
```

`const base = 2 + 2*(4+4+8) + 2` = 38.

| tenant | expected | why |
|---|---|---|
| `("GMS", 95, 1)` | `base + 1*4` = 42 | 1120012 is ignored; only 1120003 carries it |
| `("GMS", 92, 1)` | `base + 2*4` = 46 | no ignore list; both are job-112 4th-job skills |
| `("GMS", 87, 1)` | `base + 2*4` = 46 | same |
| `("JMS", 185, 1)` | `base + 2*4` = 46 | same |

Comment above it: this is the live GMS v95 bug of design §2.3 — before task-275 Atlas wrote a master-level int for all fourteen reachable ignore-list ids on v95, a 4-byte-per-skill shift of everything after the skill block.

**`TestCharacterDataJMSDualBladePlainSP`** — full-`CharacterData` encode, FR-10. Use a `mk(jobId uint16, skills []SkillEntry) CharacterData` helper of the shape at `data_evan_test.go:41-54` (`Id: 1, Name: "Blade", Level: 120`, `Hp/MaxHp/Mp/MaxMp: 100`, `Sp: 3`, `MapId: 100030102`, `BuddyCapacity: 20`, and the same `InventoryData` block including `EquipSlotExtExpire: 94354848000000000`). With `ctx := pt.CreateContext("JMS", 185, 1)`:

- `blade := mk(434, []SkillEntry{{Id: 4340000, Level: 30, MasterLevel: 30}})`
- `ranger := mk(312, []SkillEntry{{Id: 3121002, Level: 30, MasterLevel: 30}})`

Assert `len(pt.Encode(t, ctx, blade.Encode, nil)) == len(pt.Encode(t, ctx, ranger.Encode, nil))`. Both take the plain 2-byte SP short (`430/1000 == 0`, `312/1000 == 0`) and both carry exactly one master-level int, so the two encodings are the same length. A failure means the Dual Blade wrongly took the extended-SP path, which is exactly the conflation FR-10 forbids.

**`TestCharacterDataJMSEvanExtendedSP`** — the JMS half of the extended-SP fix, mirroring `TestEvanExtendedSPv84` but on JMS. With `ctx := pt.CreateContext("JMS", 185, 1)` and the same `mk` helper, no skills:

- `evan := mk(2218, nil)`, `normal := mk(312, nil)`
- Assert `len(evanBytes) == len(normalBytes)-1` — the 1-byte extended-SP count replaces the 2-byte SP short (`sub_5163A2` @0x5163a2, called @0x50eda2).
- Round-trip: `out := CharacterData{}; pt.RoundTrip(t, ctx, evan.Encode, out.Decode, nil)`, then assert `out.Stats.JobId == 2218`. This is D4's mirror property — decode must read the count byte, not a short.
- Add a second round-trip case for `mk(434, ...)` on JMS asserting `out.Stats.Sp == 3`, proving the Dual Blade still round-trips the plain SP short.

**`TestDecodeExtendedSPNonZeroCount`** — the D5 decode-only case, proving the reader consumes `1 + 2*count` bytes for a client-authored packet. Atlas's encoder only ever writes count 0, so this shape cannot be produced by a round-trip:

Build the byte stream by hand: encode a JMS 185 Evan (`mk(2218, nil)`) with `pt.Encode`, then splice — replace the single `0x00` count byte with `0x02, 0x0A, 0x03, 0x14, 0x05` (count 2, then two `(masterLevelIdx, sp)` pairs). Locate the count byte by encoding a plain-SP twin (`mk(312, nil)`) and using the known prefix length up to SP, or more simply: assert on the decoded tail rather than the offset — decode the spliced stream with `CharacterData{}.Decode` via a `request.Reader` and assert that `Stats.Exp == <the encoded Exp>` and `Stats.MapId == <the encoded MapId>`, i.e. every field AFTER the extended-SP block landed at the right offset. If the reader consumed the wrong number of bytes, those two fields are garbage.

Use `mk` values that make this unambiguous: set `Exp: 50000` and `MapId: 100030102` in the fixture and assert both after decode.

- [ ] **Step 2: Run the new tests**

From `libs/atlas-packet`:

```
go test ./character/ -run 'TestSkillsDualBladeMasterLevelJMS|TestSkillsIgnoreCommonV95|TestCharacterDataJMSDualBladePlainSP|TestCharacterDataJMSEvanExtendedSP|TestDecodeExtendedSPNonZeroCount' -v
```

Expected: PASS. These test already-implemented behaviour (Task 5 landed the fix), so a failure here means either the fixture arithmetic or the Task 3/4 arms are wrong — diagnose which before adjusting a number.

- [ ] **Step 3: Run the whole module**

```
go build ./... && go vet ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-packet/character/data_master_level_test.go
git commit -m "test(character): byte fixtures for JMS Dual Blade, JMS Evan extended SP, and the v95 ignore list"
```

---

## Task 8: atlas-channel comments, packet-audit gates, and full verification

### Files

- `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go` — modify lines 72-80; comment text only, no code change
- `docs/packets/evidence/gms_v95/field.clientbound.FieldSetField.yaml` — read-only; check whether a re-pin is owed (see Step 2)
- `libs/atlas-packet/field/clientbound/set_field_test.go` — read-only; the `packet-audit:verify` markers for the SET_FIELD cells
- `tools/verify.sh` — read-only; the final gate

Module roots: `services/atlas-channel/atlas.com/channel`, then repo root for `verify.sh`.

- [ ] **Step 1: Update the atlas-channel comment**

`services/atlas-channel/atlas.com/channel/socket/writer/character_data.go:72-80` holds only prose referring to `skill.NeedsMasterLevel`, which no longer exists. Replace the comment body inside the `for _, s := range c.Skills()` loop with:

```go
		// MasterLevel is always populated; whether it reaches the wire is the
		// codec's call (charpkt derives it per SKILL from the id via
		// job.NeedsMasterLevel, which resolves the arm from the tenant's
		// region and major version). Gating it here on the per-JOB IsFourthJob
		// is what closed the client with error 38 on a preset Evan: the client
		// asks is_skill_need_master_level(nSkillID), which for Evan selects
		// only the 9th/10th growths plus three named skills — not the whole
		// 2214-2218 band. See job.NeedsMasterLevel (task-218, task-275).
```

No code in this file changes; the tenant is already in scope at the writer and the codec owns the decision.

- [ ] **Step 2: Determine whether the v95 SET_FIELD evidence needs a re-pin**

Design §5 flags this as a required step, not a contingency. Establish the answer from the artifacts rather than assuming:

```
cat docs/packets/evidence/gms_v95/field.clientbound.FieldSetField.yaml
grep -n 'cdBytes\|Skills:' libs/atlas-packet/field/clientbound/set_field_test.go
```

The expected finding, which the evidence in hand already supports, is that **no re-pin is owed**: the evidence record pins only `ida.function`, `ida.address` and `decompile_sha256` — a client-side hash this change cannot move — and the SET_FIELD golden tests assert the CharacterData middle as an opaque span re-derived by calling `cd.Encode` (`set_field_test.go:76`), over fixtures that are job 312 with **no skills**. Neither the ignore list nor any 43x arm can reach them.

If either premise turns out false — a hard-coded CharacterData byte literal, or a fixture carrying one of the sixteen ignore-list ids — stop and report before editing an evidence record; a re-pin is a `packet-verifier` job, not a comment edit.

Record the finding (either way) in `docs/tasks/task-275-jms185-characterdata-divergences/context.md` under "Evidence re-pin".

- [ ] **Step 3: Run the packet-audit checks**

From the repo root:

```
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
```

Expected: exit 0 for all three. If `matrix --check` reports the v95 SET_FIELD cell as degraded, that is the design §8 risk — report it with the exact output rather than editing status.json by hand.

- [ ] **Step 4: Build and test atlas-channel**

From `services/atlas-channel/atlas.com/channel`:

```
go build ./... && go vet ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 5: Run the full verification gate**

From the repo root:

```
tools/verify.sh
```

Flagless — `--quick` and `--no-docker` skip the bake and `-race` and do not count. Expected: exit 0.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/writer/character_data.go docs/tasks/task-275-jms185-characterdata-divergences/context.md
git commit -m "docs(channel): point the master-level comment at job.NeedsMasterLevel"
```

---

## Acceptance cross-check

| PRD acceptance criterion | Task |
|---|---|
| Predicate resolves every arm from the tenant's facts; no hand-derived `region == "GMS"` boolean at any caller | 3, 5 |
| Dual Blade arm pinned per version | 3 (`TestNeedsMasterLevel_DualBladePerVersion`) |
| Evan arm unchanged, exception list GMS ≥ 84 only | 3 (`TestNeedsMasterLevel_EvanArm`) |
| Extended-SP gate fires for JMS; full and narrow predicate bodies pinned | 4 (`TestUsesExtendedSP`) |
| Byte-exact JMS Dual Blade fixture, plain SP short present | 7 (`TestSkillsDualBladeMasterLevelJMS`, `TestCharacterDataJMSDualBladePlainSP`) |
| Byte-exact JMS Evan extended-SP fixture | 7 (`TestCharacterDataJMSEvanExtendedSP`) |
| No byte movement on an already-verified GMS cell (v84 and v95 at minimum) | 1 (`TestCharacterDataNoByteMovement`) |
| Encode/decode mirror images; round-trip covers both new shapes | 5 (single call site each), 7 (round-trips + `TestDecodeExtendedSPNonZeroCount`) |
| Every new arm carries an IDA address | 2, 3, 4 |
| Stale "no 430-434 job" claim corrected | 6 (comment deleted with the function), 3 (replacement comment) |
| Evidence re-pinned where read order changed; `packet-audit` exits 0 | 8 |
| Flagless `tools/verify.sh` exits 0 | 8 |
| Design §2.3: v95 `is_ignore_master_level_for_common` modelled | 3, 7 (`TestSkillsIgnoreCommonV95`) |
| Design §5: `ClientJobLevel` ported, not `Advancement` | 2 |
