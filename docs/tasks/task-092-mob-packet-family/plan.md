# MOB/MONSTER Packet Family — Byte-Plumbing Batch 1 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship byte-exact `Encode`/`Decode` codecs for the 42 unimplemented MOB/MONSTER operations across gms_v83/v84/v87/v95 + jms_v185, wire each into `atlas-channel` + the five seed templates, and promote every applicable packet-audit coverage cell to `verified`.

**Architecture:** Two decoupled stages. **Stage 1 (IDA-bound)** harvests the byte layout of every in-scope op from each version IDB — one IDB loaded at a time — into per-version `structures/*.md` notes; it also fixes registry fname mislabels/gaps. **Stage 2 (pure-Go, no IDA)** transcribes each harvested layout into an immutable codec + round-trip/golden test + verify marker + pinned evidence, registers the writer/handler in `atlas-channel`, and adds the per-version opcode route to all five seed templates. The matrix grader (`tools/packet-audit`) is the burndown gate.

**Tech Stack:** Go 1.2x, `libs/atlas-packet` (codec), `libs/atlas-socket` (Reader/Writer), `libs/atlas-tenant` (version from ctx), `libs/atlas-opcodes` (name↔opcode), `tools/packet-audit` (matrix/evidence), atlas-configurations seed-template JSON, IDA-Pro MCP (Stage 1 only).

**Read first:** `docs/tasks/task-092-mob-packet-family/context.md` — it holds the per-version opcode table, the test-harness API, the wiring sites, the registry gaps, and the packet-audit command surface. Task steps below reference it by section (e.g. `context.md §2`) instead of repeating data.

**Why codec bodies are not pre-written here:** CLAUDE.md forbids citing packet bytes from memory — the field order of an unimplemented op is unknown until its IDB function is decompiled. Stage 1 produces that field order as a concrete artifact (`structures/<version>.md#<OP>`). Stage-2 codec steps transcribe from that artifact. This is a deliberate two-stage structure, not a placeholder: every non-byte detail (paths, names, opcodes, markers, commands, template JSON) is fully specified below.

---

## Conventions used by every Stage-2 op

Three reusable recipes. Each op task names which recipe(s) it uses and supplies its own data; the full code lives here so no task is "similar to" another.

### Recipe R-CB — new clientbound codec (server→client)

**Files:** Create `libs/atlas-packet/<pkg>/<Op>.go`; create `libs/atlas-packet/<pkg>/<Op>_test.go`; modify `services/atlas-channel/atlas.com/channel/main.go` (`produceWriters`); create `services/atlas-channel/atlas.com/channel/socket/writer/<op>.go`; modify the 5 templates.

- [ ] **R-CB.1 — Write the failing test.** In `<Op>_test.go`, golden-byte for v83 + round-trip across all variants. Field values are illustrative; byte expectations come from `structures/gms_v83.md#<OP>`.

```go
package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// (markers added in R-CB.5, not yet)
func Test<Op>(t *testing.T) {
	input := New<Op>(/* concrete fixture args */)
	// golden byte check (v83 baseline) — bytes transcribed from structures/gms_v83.md#<OP>
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{ /* per-field bytes with // comments citing the decompile line */ }
	if !bytes.Equal(got, want) {
		t.Fatalf("<OP> layout mismatch\n got % x\nwant % x", got, want)
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
```

- [ ] **R-CB.2 — Run, expect FAIL** (`New<Op>` undefined): `go test ./libs/atlas-packet/<pkg>/ -run Test<Op>`.
- [ ] **R-CB.3 — Write the codec** `<Op>.go`, modeled on `monster/clientbound/spawn.go` / `character/clientbound/set_taming_mob_info.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const <Op>Writer = "<WriterName>"   // see op data table

type <Op> struct { /* private fields, types per structures/<version>.md */ }
func New<Op>(/* args */) <Op> { return <Op>{ /* … */ } }
/* getters … */
func (m <Op>) Operation() string { return <Op>Writer }
func (m <Op>) String() string { return fmt.Sprintf("…", /* fields */) }

func (m <Op>) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		_ = t // drop if no version branch
		// fields in client read-order from structures/<version>.md#<OP>;
		// version-branch with t.MajorAtLeast(87) / t.Region()=="JMS" — NEVER >83
		return w.Bytes()
	}
}

func (m *<Op>) Decode(l logrus.FieldLogger, ctx context.Context) func(*request.Reader, map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		_ = t
		// exact mirror of Encode read-order
	}
}
```

- [ ] **R-CB.4 — Run, expect PASS:** `go test ./libs/atlas-packet/<pkg>/ -run Test<Op>`.
- [ ] **R-CB.5 — Add verify markers** above `Test<Op>`, one per applicable version, `ida=` = the export address for the op's fname (from `structures/<version>.md#<OP>`, which records it from the export):

```go
// packet-audit:verify packet=<pkg>/<Op> version=gms_v83 ida=0x<addr>
// packet-audit:verify packet=<pkg>/<Op> version=gms_v84 ida=0x<addr>
// packet-audit:verify packet=<pkg>/<Op> version=gms_v87 ida=0x<addr>
// packet-audit:verify packet=<pkg>/<Op> version=gms_v95 ida=0x<addr>
// packet-audit:verify packet=<pkg>/<Op> version=jms_v185 ida=0x<addr>
```

- [ ] **R-CB.6 — Pin evidence**, once per applicable version, then add the `verifies:` list to each generated YAML:

```bash
for V in gms_v83 gms_v84 gms_v87 gms_v95 jms_v185; do \
  go run ./tools/packet-audit evidence pin --packet <pkg>/<Op> --version $V \
    --ida "<fname>" --category TIER1-FIXTURE ; done
```

(If `pin` reports "function … not in export", STOP and ESCALATE to the user — that version's export lacks the fname. Do not fabricate the hash, auto-re-export, or silently substitute a fname; present it to the user and wait for a decision per context.md §3.4.)

- [ ] **R-CB.7 — Register the writer.** Add `<pkg-alias>.<Op>Writer` to `produceWriters()` (main.go:592-693, sorted near sibling writers). Create `socket/writer/<op>.go`:

```go
package writer

import (
	"context"

	<alias> "github.com/Chronicle20/atlas/libs/atlas-packet/<pkg>"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

func <Op>Body(/* domain args matching New<Op> */) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return <alias>.New<Op>(/* args */).Encode(l, ctx)(options)
		}
	}
}
```

- [ ] **R-CB.8 — Route in all 5 templates.** Insert a `socket.writers[]` entry `{ "opCode": "0x<hex>", "writer": "<WriterName>" }` in sorted opcode position in each `template_{gms_83,gms_84,gms_87,gms_95}_1.json` + `template_jms_185_1.json`, using the per-version opcode from context.md §2.
- [ ] **R-CB.9 — Regenerate + check:** `go run ./tools/packet-audit matrix && go run ./tools/packet-audit matrix --check`; confirm the op's row flipped to ✅ for every applicable version, 0 conflicts.
- [ ] **R-CB.10 — Commit** (`go test ./... -race` + `go vet ./...` for libs/atlas-packet + atlas-channel first):

```bash
git add libs/atlas-packet/<pkg>/ services/atlas-channel/ services/atlas-configurations/ docs/packets/
git commit -m "feat(task-092): <OP> codec + wiring + verified (5v)"
```

### Recipe R-SB — new serverbound codec (client→server)

Identical to R-CB except:
- Codec lives under `…/serverbound/`; const is `<Op>Handle = "<HandlerName>"`.
- No writer Body. Instead, register the handler: add `hm[<alias-sb>.<Op>Handle] = handler.<Op>HandleFunc` to `produceHandlers()` (main.go:695-770) and create `socket/handler/<op>.go`:

```go
package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-packet/<pkg>/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func <Op>HandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, ro map[string]interface{}) {
	return func(s session.Model, r *request.Reader, ro map[string]interface{}) {
		p := serverbound.<Op>{}
		p.Decode(l, ctx)(r, ro)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		// behavior: deferred (decode-and-log only — see design D2)
	}
}
```

- Template entry goes in `socket.handlers[]` as `{ "opCode": "0x<hex>", "validator": "LoggedInValidator", "handler": "<HandlerName>" }` (NoOpValidator only for connection-level ops — none expected here).
- The round-trip test still drives both `Encode` and `Decode` (serverbound models implement both for testability; `Encode` mirrors the client write so RoundTrip closes).

### Recipe R-MARK — verify-only (already-implemented codec)

For `SET_TAMING_MOB_INFO` (encoder + golden test already exist; see context.md §4): skip codec/wiring; do only R-CB.5 (markers), R-CB.6 (evidence pin), R-CB.9 (regenerate/check), R-CB.10 (commit). Confirm in Stage 1 that v95/jms layouts equal v83; if any differs, that wire-fix is its own commit before marking.

---

## Stage 0 — Recon, tooling, baseline (no IDA)

### Task 0.1: Capture baseline matrix state

**Files:** none modified (read-only + scratch note).

- [ ] **Step 1:** From the worktree root, run `go run ./tools/packet-audit matrix --check; echo "exit=$?"`. Expected: exit 0 (clean baseline) — if non-zero, record the pre-existing failures in `docs/tasks/task-092-mob-packet-family/structures/baseline.md` so they're not attributed to this task.
- [ ] **Step 2:** Record the current ❌ state of all 42 ops by grepping `docs/packets/audits/STATUS.md` into `structures/baseline.md`.
- [ ] **Step 3:** Commit: `git add docs/tasks/task-092-mob-packet-family/structures/baseline.md && git commit -m "chore(task-092): record matrix baseline"`.

### Task 0.2: Export-resolvability audit

**Files:** Create `structures/export-gaps.md`.

- [ ] **Step 1:** List the export files: `ls docs/packets/ida-exports/`.
- [ ] **Step 2:** For each of the 42 ops × applicable versions, check the op's `fname` (context.md §2) resolves in the matching export JSON's `functions` map. Use a throwaway Go/jq script: for each version key, `jq '.functions | keys[]' <export>` and grep for each fname.
- [ ] **Step 3:** Write `structures/export-gaps.md`: a table of (op, version, fname, resolves? Y/N). Every "N" is a pin blocker — flag whether re-export (task-081 playbook) is needed or the fname string differs from the export's key (demangled vs mangled — see memory `reference_ida_mcp_new_api`).
- [ ] **Step 4: ESCALATE any "N" to the user.** Do NOT auto-trigger a re-export or pick a substitute fname. Stop and present the unresolved (op, version, fname) list to the user for a decision (re-export vs fname correction vs descope). Resume only after the user responds. Commit `structures/export-gaps.md`.

### Task 0.3: Registry gap inventory

**Files:** Create `structures/registry-gaps.md`.

- [ ] **Step 1:** Read all 5 `docs/packets/registry/*.yaml`. For each of the 42 ops, record present/absent per version and the `direction` + `fname` recorded.
- [ ] **Step 2:** Enumerate the known issues (context.md §3): MOB_SPEAKING / INC_MOB_CHARGE_COUNT / MOB_SKILL_DELAY fname mislabels; MONSTER_BOOK_COVER missing fname; MOB_ESCORT_RETURN_STOP / _STOP_SAY absent everywhere. Add any newly-found gaps.
- [ ] **Step 3:** Write `structures/registry-gaps.md` as the Phase-1 work list. Commit.

---

## Stage 1 — IDA derivation harvest (one IDB at a time)

> **Operator note:** Each task below requires the matching IDB loaded in IDA. The user switches the active IDB between tasks (memory: `reference_ida_harvest_subagents`). Do all of a version's derivations before asking the user to switch. Ports: v83=13337, v84=13341, v87=13338, v95=13339, jms=13340.

Each Stage-1 task produces `structures/<version>.md` with, per applicable op: the demangled fname, the export address, and the ordered field list (`name : width : note`) in client read-order, including version guards and loop bounds. It also patches `docs/packets/registry/<version>.yaml` for that version's gaps.

### Task 1.A: Harvest gms_v83 (port 13337)

**Files:** Create `structures/gms_v83.md`; modify `docs/packets/registry/gms_v83.yaml`.

- [ ] **Step 1:** `select_instance(13337)`; confirm with `list_instances` the IDB is v83.
- [ ] **Step 2:** For each op present in `gms_v83.yaml` (clientbound + serverbound from the 42-list), `decompile` its `fname`, descending into helper read/write subs. Record the ordered field list + export address into `structures/gms_v83.md#<OP>`. For serverbound, record the opcode from the yaml too (context.md §2 leaves serverbound opcodes to be read here).
- [ ] **Step 3:** Resolve the v83 fname mislabels (MOB_SPEAKING, INC_MOB_CHARGE_COUNT): confirm the correct fname against the decompile; fix the yaml row (`provenance: manual`, IDA citation in `note`).
- [ ] **Step 4:** Derive MONSTER_BOOK_COVER (serverbound) send-site; set its `fname`/`ida.address` in the yaml (`provenance: ida-discovered`).
- [ ] **Step 5:** Note the v84≡v83 expectation for confirmation in Task 1.B. Commit `structures/gms_v83.md` + the yaml edits.

### Task 1.B: Harvest gms_v84 (port 13341)

**Files:** Create `structures/gms_v84.md`; modify `docs/packets/registry/gms_v84.yaml`.

- [ ] **Step 1:** `select_instance(13341)`; confirm v84.
- [ ] **Step 2:** For each op, confirm the layout matches v83 (expected: byte-identical below the shifted opcode table — memory `bug_v84_opcode_table_shifted_vs_v83`). Record only the **deltas vs v83** in `structures/gms_v84.md` (plus per-op export addresses, which differ). Flag any op whose body genuinely diverges from v83.
- [ ] **Step 3:** Fix v84 fname mislabels in the yaml as in 1.A.3. Commit.

### Task 1.C: Harvest gms_v87 (port 13338)

**Files:** Create `structures/gms_v87.md`; modify `docs/packets/registry/gms_v87.yaml`.

- [ ] **Step 1:** `select_instance(13338)`; confirm v87.
- [ ] **Step 2:** Derive each op; record full field list + addresses; capture the v87+ structural additions (the `MajorAtLeast(87)` branch fields). Include the Cluster-F v87 ops (MOB_ESCORT_FULL_PATH, MOB_ESCORT_COLLISION).
- [ ] **Step 3:** Fix the v87 MOB_SKILL_DELAY fname mislabel (`OnMobAttackedByMob`→`OnMobSkillDelay`) in the yaml. Commit.

### Task 1.D: Harvest gms_v95 (port 13339)

**Files:** Create `structures/gms_v95.md`; modify `docs/packets/registry/gms_v95.yaml`.

- [ ] **Step 1:** `select_instance(13339)`; confirm v95.
- [ ] **Step 2:** Derive each op incl. the v95-only Cluster-F tail (MOB_ATTACKED_BY_MOB, MOB_ESCORT_RETURN_BEFORE/STOP/STOP_SAY, MOB_NEXT_ATTACK, MOB_ESCORT_STOP_END_REQUEST, MOB_REQUEST_ESCORT_INFO). Record field lists + addresses.
- [ ] **Step 3:** Add registry rows for MOB_ESCORT_RETURN_STOP / _STOP_SAY if present in the IDB (`provenance: ida-discovered`); if absent from the IDB, document non-existence in `structures/gms_v95.md` and drop them from scope. Commit.

### Task 1.E: Harvest jms_v185 (port 13340)

**Files:** Create `structures/jms_v185.md`; modify `docs/packets/registry/jms_v185.yaml`.

- [ ] **Step 1:** `select_instance(13340)`; confirm jms.
- [ ] **Step 2:** Derive each applicable op (note JMS lacks MOB_SPEAKING/INC_MOB_CHARGE_COUNT/MOB_SKILL_DELAY per context.md §2 — confirm and mark VERSION-ABSENT). Record field lists + addresses + region deltas.
- [ ] **Step 3:** Fix any jms fname gaps. Commit.

### Task 1.F: Reconcile applicability matrix

**Files:** Create `structures/applicability.md`.

- [ ] **Step 1:** From the 5 structures docs + registries, build the authoritative (op × version) applicability grid: implement / n-a(VERSION-ABSENT). This drives which marker/evidence lines each Stage-2 op needs.
- [ ] **Step 2:** For every n-a cell, note the `VERSION-ABSENT` justification (IDB evidence). Commit.

---

## Stage 2 — Codec + wiring + verification (pure Go)

Cluster order per design §5: D → A → B → C → F → E. Each op = one task using R-CB / R-SB / R-MARK with the data below. After each cluster, run the full module gates (`go test -race`, `go vet`) and `matrix --check`.

### Cluster D — CRC / misc plumbing (proves the recipe)

| Task | Op | Recipe | pkg / Struct | Name const | fname |
|---|---|---|---|---|---|
| 2.D1 | MOB_CRC_KEY_CHANGED | R-CB | `monster/clientbound/MobCrcKeyChanged` | `MobCrcKeyChangedWriter="MobCrcKeyChanged"` | CMobPool::OnMobCrcKeyChanged |
| 2.D2 | MOB_CRC_KEY_CHANGED_REPLY | R-SB | `monster/serverbound/MobCrcKeyChangedReply` | `MobCrcKeyChangedReplyHandle="MobCrcKeyChangedReply"` | CMobPool::OnMobCrcKeyChanged |
| 2.D3 | MOB_DROP_PICKUP_REQUEST | R-SB | `monster/serverbound/MobDropPickupRequest` | `MobDropPickupRequestHandle="MobDropPickupRequest"` | CMob::SendDropPickUpRequest |
| 2.D4 | MOB_BANISH_PLAYER | R-SB | `character/serverbound/MobBanishPlayer` | `MobBanishPlayerHandle="MobBanishPlayer"` | CUserLocal::SendBanMapByMobRequest |

Each task: opcodes per version from context.md §2 (serverbound opcodes from `structures/<version>.md`); codec body from `structures/<version>.md#<OP>`; markers/evidence for the versions marked "implement" in `structures/applicability.md`. **2.D1 is the worked exemplar** — execute all R-CB steps fully and use the resulting files as the copy-reference for later ops; capture the recipe verbatim into `IMPLEMENTING_A_PACKET.md` draft (Task 9.1) as you go.

### Cluster A — Mob combat / damage

| Task | Op | Recipe | pkg / Struct | Name const | fname |
|---|---|---|---|---|---|
| 2.A1 | FIELD_DAMAGE_MOB | R-SB | `monster/serverbound/FieldDamageMob` | `FieldDamageMobHandle="FieldDamageMob"` | CMob::Update |
| 2.A2 | MOB_DAMAGE_MOB | R-SB | `monster/serverbound/MobDamageMob` | `MobDamageMobHandle="MobDamageMob"` | CMob::SetDamagedByMob |
| 2.A3 | MOB_DAMAGE_MOB_FRIENDLY | R-SB | `monster/serverbound/MobDamageMobFriendly` | `MobDamageMobFriendlyHandle="MobDamageMobFriendly"` | CMob::Update |
| 2.A4 | MONSTER_BOMB | R-SB | `monster/serverbound/MonsterBomb` | `MonsterBombHandle="MonsterBomb"` | CMob::TryFirstSelfDestruction |
| 2.A5 | MOB_TIME_BOMB_END | R-SB | `monster/serverbound/MobTimeBombEnd` | `MobTimeBombEndHandle="MobTimeBombEnd"` | CMob::UpdateTimeBomb |
| 2.A6 | MOB_SKILL_DELAY_END | R-SB | `monster/serverbound/MobSkillDelayEnd` | `MobSkillDelayEndHandle="MobSkillDelayEnd"` | CMob::Update |
| 2.A7 | TOUCH_MONSTER_ATTACK | R-SB | `character/serverbound/TouchMonsterAttack` | `TouchMonsterAttackHandle="TouchMonsterAttack"` | CUserLocal::TryDoingBodyAttack |
| 2.A8 | MOB_AFFECTED | R-CB | `monster/clientbound/MobAffected` | `MobAffectedWriter="MobAffected"` | CMob::OnAffected |
| 2.A9 | MONSTER_SPECIAL_EFFECT_BY_SKILL | R-CB | `monster/clientbound/MonsterSpecialEffectBySkill` | `MonsterSpecialEffectBySkillWriter="MonsterSpecialEffectBySkill"` | CMob::OnSpecialEffectBySkill |
| 2.A10 | RESET_MONSTER_ANIMATION | R-CB | `monster/clientbound/ResetMonsterAnimation` | `ResetMonsterAnimationWriter="ResetMonsterAnimation"` | CMob::OnSuspendReset |

> Damage ops reuse shared sub-structs from `libs/atlas-packet/model/` where the IDB shows them; check `structures/*.md` for which fields are opaque sub-structs vs scalars before declaring new types.

### Cluster B — Catch / taming

| Task | Op | Recipe | pkg / Struct | Name const | fname |
|---|---|---|---|---|---|
| 2.B1 | CATCH_MONSTER | R-CB | `monster/clientbound/CatchMonster` | `CatchMonsterWriter="CatchMonster"` | CMob::OnCatchEffect |
| 2.B2 | CATCH_MONSTER_WITH_ITEM | R-CB | `monster/clientbound/CatchMonsterWithItem` | `CatchMonsterWithItemWriter="CatchMonsterWithItem"` | CMob::OnEffectByItem |
| 2.B3 | BRIDLE_MOB_CATCH_FAIL | R-CB | `character/clientbound/BridleMobCatchFail` | `BridleMobCatchFailWriter="BridleMobCatchFail"` | CWvsContext::OnBridleMobCatchFail |
| 2.B4 | SET_TAMING_MOB_INFO | R-MARK | `character/clientbound/SetTamingMobInfo` (exists) | `SetTamingMobInfoWriter` (exists) | CWvsContext::OnSetTamingMobInfo |

### Cluster C — Monster book

| Task | Op | Recipe | pkg / Struct | Name const | fname |
|---|---|---|---|---|---|
| 2.C1 | MONSTER_BOOK_SET_CARD | R-CB | `character/clientbound/MonsterBookSetCard` | `MonsterBookSetCardWriter="MonsterBookSetCard"` | CWvsContext::OnMonsterBookSetCard |
| 2.C2 | MONSTER_BOOK_SET_COVER | R-CB | `character/clientbound/MonsterBookSetCover` | `MonsterBookSetCoverWriter="MonsterBookSetCover"` | CWvsContext::OnMonsterBookSetCover |
| 2.C3 | MONSTER_BOOK_COVER | R-SB | `character/serverbound/MonsterBookCover` | `MonsterBookCoverHandle="MonsterBookCover"` | (derived in Task 1.A.4) |

> **Note:** a handler name `MonsterBookCover` already appears in the templates (session-context exploration). Before 2.C3, grep the templates + `produceHandlers` for an existing `MonsterBookCover` handler; if one exists, reconcile (reuse the name, add the codec/decode) rather than duplicating.

### Cluster F — Version-tail (implement only where applicable per `structures/applicability.md`)

| Task | Op | Recipe | pkg / Struct | Name const | fname | versions |
|---|---|---|---|---|---|---|
| 2.F1 | INC_MOB_CHARGE_COUNT | R-CB | `monster/clientbound/IncMobChargeCount` | `IncMobChargeCountWriter="IncMobChargeCount"` | CMob::OnIncMobChargeCount | v83,v84,v87,v95 |
| 2.F2 | MOB_SKILL_DELAY | R-CB | `monster/clientbound/MobSkillDelay` | `MobSkillDelayWriter="MobSkillDelay"` | CMob::OnMobSkillDelay | v83,v84,v87,v95 |
| 2.F3 | MOB_SPEAKING | R-CB | `monster/clientbound/MobSpeaking` | `MobSpeakingWriter="MobSpeaking"` | CMob::OnMobSpeaking | v83,v84,v87,v95 |
| 2.F4 | MOB_ESCORT_COLLISION | R-SB | `monster/serverbound/MobEscortCollision` | `MobEscortCollisionHandle="MobEscortCollision"` | CMob::OnEscortCollision | per applicability (≈v87,v95,jms) |
| 2.F5 | MOB_ESCORT_FULL_PATH | R-CB | `monster/clientbound/MobEscortFullPath` | `MobEscortFullPathWriter="MobEscortFullPath"` | CMob::OnEscortFullPath | v87,v95 |
| 2.F6 | MOB_ESCORT_STOP_END_REQUEST | R-SB | `monster/serverbound/MobEscortStopEndRequest` | `MobEscortStopEndRequestHandle="MobEscortStopEndRequest"` | (structures) | per applicability |
| 2.F7 | MOB_REQUEST_ESCORT_INFO | R-SB | `monster/serverbound/MobRequestEscortInfo` | `MobRequestEscortInfoHandle="MobRequestEscortInfo"` | (structures) | per applicability |
| 2.F8 | MOB_ATTACKED_BY_MOB | R-CB | `monster/clientbound/MobAttackedByMob` | `MobAttackedByMobWriter="MobAttackedByMob"` | CMob::OnMobAttackedByMob | v95 |
| 2.F9 | MOB_ESCORT_RETURN_BEFORE | R-CB | `monster/clientbound/MobEscortReturnBefore` | `MobEscortReturnBeforeWriter="MobEscortReturnBefore"` | CMob::OnEscortReturnBefore | v95 |
| 2.F10 | MOB_ESCORT_RETURN_STOP | R-CB | `monster/clientbound/MobEscortReturnStop` | `MobEscortReturnStopWriter="MobEscortReturnStop"` | (Task 1.D.3) | per Task 1.D.3 |
| 2.F11 | MOB_ESCORT_RETURN_STOP_SAY | R-CB | `monster/clientbound/MobEscortReturnStopSay` | `MobEscortReturnStopSayWriter="MobEscortReturnStopSay"` | (Task 1.D.3) | per Task 1.D.3 |
| 2.F12 | MOB_NEXT_ATTACK | R-CB | `monster/clientbound/MobNextAttack` | `MobNextAttackWriter="MobNextAttack"` | CMob::OnNextAttack | v95 |

For each F op: emit markers/evidence ONLY for the applicable versions; for the inapplicable versions, pin a `VERSION-ABSENT` evidence record (no test) so the cell grades `n/a`, citing the `structures/applicability.md` justification. Confirm with `matrix --check` that inapplicable cells are `⬜ n/a`, not `🟥 conflict`.

### Cluster E — Monster Carnival (new package `monster/carnival/`)

First task creates the package dirs + the `carnivalcb`/`carnivalsb` import aliases in main.go.

| Task | Op | Recipe | pkg / Struct | Name const | fname |
|---|---|---|---|---|---|
| 2.E1 | MONSTER_CARNIVAL | R-SB | `monster/carnival/serverbound/MonsterCarnival` | `MonsterCarnivalHandle="MonsterCarnival"` | CUIMonsterCarnival::RequestSend |
| 2.E2 | MONSTER_CARNIVAL_START | R-CB | `monster/carnival/clientbound/MonsterCarnivalStart` | `MonsterCarnivalStartWriter="MonsterCarnivalStart"` | CField_MonsterCarnival::OnEnter |
| 2.E3 | MONSTER_CARNIVAL_OBTAINED_CP | R-CB | `monster/carnival/clientbound/MonsterCarnivalObtainedCP` | `MonsterCarnivalObtainedCPWriter="MonsterCarnivalObtainedCP"` | CField_MonsterCarnival::OnPersonalCP |
| 2.E4 | MONSTER_CARNIVAL_PARTY_CP | R-CB | `monster/carnival/clientbound/MonsterCarnivalPartyCP` | `MonsterCarnivalPartyCPWriter="MonsterCarnivalPartyCP"` | CField_MonsterCarnival::OnTeamCP |
| 2.E5 | MONSTER_CARNIVAL_SUMMON | R-CB | `monster/carnival/clientbound/MonsterCarnivalSummon` | `MonsterCarnivalSummonWriter="MonsterCarnivalSummon"` | CField_MonsterCarnival::OnRequestResult |
| 2.E6 | MONSTER_CARNIVAL_MESSAGE | R-CB | `monster/carnival/clientbound/MonsterCarnivalMessage` | `MonsterCarnivalMessageWriter="MonsterCarnivalMessage"` | CField_MonsterCarnival::OnRequestResult |
| 2.E7 | MONSTER_CARNIVAL_DIED | R-CB | `monster/carnival/clientbound/MonsterCarnivalDied` | `MonsterCarnivalDiedWriter="MonsterCarnivalDied"` | CField_MonsterCarnival::OnProcessForDeath |
| 2.E8 | MONSTER_CARNIVAL_LEAVE | R-CB | `monster/carnival/clientbound/MonsterCarnivalLeave` | `MonsterCarnivalLeaveWriter="MonsterCarnivalLeave"` | CField_MonsterCarnival::OnShowMemberOutMsg |
| 2.E9 | MONSTER_CARNIVAL_RESULT | R-CB | `monster/carnival/clientbound/MonsterCarnivalResult` | `MonsterCarnivalResultWriter="MonsterCarnivalResult"` | CField_MonsterCarnival::OnShowGameResult |

> **SUMMON vs MESSAGE caveat:** both registry rows point at `OnRequestResult` (context.md §2). They are distinct opcodes — confirm in Stage 1 whether they are two modes of one writer or two separate structures; model accordingly (two structs if the byte layouts differ).
> **Tier-1 preservation:** `monster/carnival/...` keeps the `monster/` prefix so `tiers.yaml` still grades these tier-1 (design §4). Do not create a top-level `carnival/` package.

---

## Stage 3 — Documentation

### Task 9.1: `docs/packets/IMPLEMENTING_A_PACKET.md`

**Files:** Create `docs/packets/IMPLEMENTING_A_PACKET.md`.

- [ ] **Step 1:** Write the four-step recipe (derive → model+codec → wire → verify) transcribing the R-CB / R-SB / R-MARK recipes and the worked Task-2.D1 files as the canonical example.
- [ ] **Step 2:** Document: the package-by-owner-class rule + the tier-1 `monster/`/`character/` prefix caveat; the "clientbound writer with no emitter is an intentional seam, not dead code" convention (design D2); the validator-mandatory rule + `BuildHandlerMap` silent-drop failure mode; the `>83`→`MajorAtLeast(87)` gate rule; the registry-fname-mislabel guard; the export-resolvability precondition for `evidence pin`.
- [ ] **Step 3:** Cross-link `VERIFYING_A_PACKET.md`, `tiers.yaml`, `registry/README.md`. Commit.

### Task 9.2: `deploy-notes.md`

**Files:** Create `docs/tasks/task-092-mob-packet-family/deploy-notes.md`.

- [ ] **Step 1:** Per-version opcode table for every new handler + writer (from context.md §2 + `structures/*.md`), in the live-tenant PATCH shape.
- [ ] **Step 2:** The rollout checklist: PATCH each live v83/v84/v87/v95/jms tenant `socket.handlers`/`socket.writers`; restart `atlas-channel`; post-deploy checks (`grep "Unable to locate validator"`==0; no "unhandled message op" for the new serverbound ops). Commit.

---

## Stage 4 — Final verification & handoff

### Task 10.1: Full gates

- [ ] **Step 1:** `go test -race ./...` clean in `libs/atlas-packet` and `services/atlas-channel/atlas.com/channel`.
- [ ] **Step 2:** `go vet ./...` clean in the same; `GOWORK=off ./tools/redis-key-guard.sh` clean from repo root.
- [ ] **Step 3:** `git diff --name-only -- '**/go.mod'` — if `services/atlas-channel/.../go.mod` changed, `docker buildx bake atlas-channel` from the worktree root; expect success. (atlas-configurations: JSON-only edits → no bake.)
- [ ] **Step 4:** `go run ./tools/packet-audit matrix && go run ./tools/packet-audit matrix --check; echo exit=$?` — expect exit 0; spot-check STATUS.md shows every targeted op ✅ for applicable versions and ⬜ (n/a) with VERSION-ABSENT evidence elsewhere; 0 🟥.
- [ ] **Step 5:** Final commit of regenerated STATUS.md/status.json if not already committed per-cluster.

### Task 10.2: Code review

- [ ] **Step 1:** Invoke `superpowers:requesting-code-review` (dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer`). Brief the backend reviewer that uncalled clientbound writer Body helpers are an intentional seam (design D2 / IMPLEMENTING_A_PACKET.md), not dead code.
- [ ] **Step 2:** Address findings per `superpowers:receiving-code-review`. Re-run Task 10.1 gates after fixes.

---

## Self-Review notes (coverage check vs design)

- Design §1 "verified requires codec+test+marker+evidence" → R-CB/R-SB steps 1-9 + R-MARK. ✓
- Design §3 four-step recipe → Stage 1 (derive) + R-CB/R-SB (model/wire/verify). ✓
- Design §4 placement (monster/, character/, monster/carnival/) → Stage-2 pkg columns. ✓
- Design §5 sequencing D→A→B→C→F→E → Cluster order. ✓
- Design §6 SET_TAMING dedup → Task 2.B4 / R-MARK + context.md §4. ✓
- Design §7 rollout → Task 9.2. ✓
- Design §8 gates → Task 10.1. ✓
- Design §9 IMPLEMENTING_A_PACKET.md → Task 9.1. ✓
- Design §10 risks (fname mislabel, carnival tier-1, dead-code, v84≡v83, book-cover fname) → Stage 1 tasks + cluster notes + Task 10.2 brief. ✓
- Open Qs #1-#6 → resolved in design §12; mechanics in Stage 1 + Task 2.B4. ✓
