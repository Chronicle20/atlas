# Damage-Taken Mitigation/Reaction Skills — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace atlas-channel's unconditional `ChangeHP(-damage)` on damage-taken packets with a server-authoritative mitigation pipeline covering Magic Guard, Power Guard, Meso Guard, Mana Reflection, Achilles, High Defense, Combo Barrier, Magic Shield, and the GUARD/Divine-Shield suppression rule, plus the mandatory packet-decoder fix.

**Architecture:** Pure mitigation math (`computeMitigation`) over an input struct with all version gates pre-resolved, orchestrated by a deps-injected `processDamageTaken` in the handler package (mirror of the proven `damageInfoEntryDeps`/`processDamageInfoEntry` pattern in `character_attack_common.go`). Wire fixes land in `libs/atlas-packet`; a new `fixed_damage` field lands in atlas-data; atlas-channel gains a monster-template data client, a tenant-scoped skill-data cache, and a `REQUEST_CHANGE_MESO` producer.

**Tech Stack:** Go, atlas-kafka producers, atlas-rest requests, api2go JSON:API, testify-free stdlib tests with the project Builder pattern.

## Global Constraints

- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module (libs/atlas-packet, services/atlas-data, services/atlas-channel).
- `docker buildx bake atlas-channel` and `docker buildx bake atlas-data` from the worktree root (both services' `go.mod` trees are touched).
- `tools/redis-key-guard.sh` clean from the repo root (no `GOWORK=off` prefix).
- `tools/goroutine-guard.sh` clean from the repo root (merged from `main`; bans bare `go` outside `libs/atlas-routine` — this task adds no goroutines, so it should pass untouched).
- **`tools/template-opcode-order-guard.sh` clean from the repo root — MANDATORY, this task edits socket-config templates** (Task 7 wires `CharacterDamageHandle` into gms_87/92/95/jms). New handler entries go at their sorted `opCode` position, never appended.
- `tools/lint.sh --check` clean from the repo root (merged from `main`; shared gofumpt/goimports + golangci-lint v2 guard across every Go module + atlas-ui). Run `tools/lint.sh` (fix mode) before committing.
- Builder pattern for test setup; NO `*_testhelpers.go` files (inline `t.Helper()` helpers in `_test.go` files are the established alternative — see `buildSkillModel` in `character_skill_prepare_test.go`).
- No `// TODO` markers in landed code. The battleship TODO (`// TODO decrease battleship hp`) is the ONLY TODO that survives in `character_damage.go` (owned by task-153).
- Never hard-code client wire values that the design derives from config/effect data; all mitigation amounts come from buff statups, skill effect data, or IDA-verified formulas below.
- All code comments/documents use repo-relative paths.
- Commit after each task with the message given in the task's final step.

**Version scope (post-`main`-merge).** `main`'s legacy bring-up wired `CharacterDamageHandle` into the gms_48/61/72/79/84 templates (it was gms_83/84-only when this plan was first written), so the decoder + pipeline now run on all pre-BB legacy columns. The full column set is **v48, v61, v72, v79, v83, v84, v87, v92, v95, jms_185** — all IDA-verified (design §2a, §3; v92 verified independently, not inherited from v87). The verification bottom line, load-bearing for this plan: (1) the mob-hit wire layout is v83-identical on **every** version except **v48**, which needs one decoder branch (Task 1); (2) every mitigation-formula version gate already in Task 6 (`pgCapDivisor`, `pgFixedDamageOverride`, `magicShieldOnReducedDamage`, the v95 GUARD rule) is verified correct across all ten columns — **no formula code change**; (3) per-version skill *availability* deltas need no gates because the pipeline is data-driven and server-authoritative; (4) gms_87/92/95/jms never routed this handler at all (pre-existing gap) — Task 7 wires them.

## Plan-phase verification findings (corrections & blockers)

These were verified during planning against the v83 (port 13342), v87 (13343), v95 (13341), and jms185 (13344) IDBs, and they refine design.md per FR-3.4 (client binary wins):

1. **Reflect-extension presence is length-derived, not flag-derived.** The client writes the 14-byte extension iff `bKnockback || nX != 0` (v83 `CUserLocal::SetDamaged` pseudocode line `if ( v171 || *v185 )`; v95 `if ( bKnockback || nX > 0 )` at decompile line 1162). The block byte is `bBlocked ? (bKnockback ? 2 : 1) : 0` and **bBlocked can be set without bKnockback** (`bKnockback = pInfo == 0` — knockback only for touch attacks; a Guardian-blocked mob *skill* attack yields blockByte=1 with NO extension). Conversely a vehicle-riding block yields bBlocked=0/bKnockback=1 → blockByte=0 WITH extension. So design.md §2's stated condition (`nX != 0 || blockByte != 0`) over-reads on blockByte==1 hits and under-reads on the vehicle edge. The only correct observable: after the block byte, exactly 1 byte (stance) remains without the extension, 15 with it → **read the extension iff `r.Available() > 1`**.
2. **Mob branch is `attackIdx >= -1`, and positive indices exist.** The client encodes the mob-shaped body whenever a mob pointer is present; `nAttackIdx` for mob skill attacks is the attack slot index (0+), touch is −1. The non-mob branch emits `(flag == 0) - 3` = **−2 or −3** (v83 line 1003, v95 line 634) — never −4 on any verified version. The current decoder's `== Physical || == Magic` branch misparses mob skill attacks with index ≥ 1 and misparses −2 events as mob-shaped never (they already fall through) — fix the branch to `>= DamageTypePhysical` and classify ≤ −2 as non-mob.
3. **Power Guard cap divisor differs on v95.** v83 (line 423), v87 (line 414), jms185 (line 476): cap = templateMaxHP/**10**. v95 GMS (line 1109): cap = templateMaxHP/**2**. Boss halving applies after the cap on all versions. Template `fixedDamage > 0` **replaces** the reflect on v95/jms (direct assignment) but is **min()**'d on v83/v87 (`if v50 <= v192` guard). Gates: `pgCapDivisor = 2 if (GMS && major ≥ 95) else 10`; `pgFixedDamageOverride = (GMS && major ≥ 95) || JMS`.
4. **Divine Shield skill-id constant is blocked on data availability — implement the GUARD rule without it.** The v95 client does not hard-code the skill id (design §1); the only per-hit obligation is GUARD-suppresses-PG-reflect, keyed off the GUARD temporary stat. Verification of the id from v95 WZ was attempted and is impossible in this environment: local WZ dumps are v83-era (Cosmic, ms_1172), the live atlas-data v95 tenant has **zero** skill documents (even 2001002 404s), and the MinIO `atlas-wz` bucket contains only `shared/regions/GMS/versions/83.1/Skill.wz` and `84.1`. Per design §5 the constant may only be added after WZ verification, so **no `libs/atlas-constants` change ships in this task**; the GUARD-based behavior is fully implemented (it needs no id). Record stays in `context.md`.
5. **`services/atlas-channel/atlas.com/channel/socket/model/damage_taken_info.go` is dead code** (only `packetmodel.NewDamageTakenInfo` is used; verified by grep). Delete it in Task 7 so a stale copy of the buggy decoder cannot linger.
6. **`fixedDamage` is not ingested by atlas-data and monster templates are not exposed to atlas-channel at all.** atlas-data's reader has no `fixedDamage` parse (WZ node confirmed as `<int name="fixedDamage" .../>` under `info` in real Mob.wz); atlas-channel has no `data/monster` client package. Tasks 2–3 create both. Note: tenants ingested before Task 2 deploys serve `fixed_damage: 0` until re-ingested — the cap then simply doesn't bind, which is graceful; re-ingestion activates it.

7. **Legacy columns (v48/61/72/79/84) + v92 verified post-merge; only v48 forces a decoder change.** After merging `main`, `CharacterDamageHandle` is wired for gms_48/61/72/79/84, so the decoder now runs pre-BB. Each column's `CUserLocal::SetDamaged` was decompiled against the v83 reference via the live per-version IDB sessions (v48 `0bb5f11a`, v61 `965202bf`, v72 `90e36cb0`, v79 `9a7d3642`, v84 `79511a2a`, v92 `c377e02e`; v83 ref `ce4ff298` — the packet-audit IDA MCP is session-based now, the old `13342`-style ports are dead). Results (design §2a/§3):
   - **Mob-hit wire layout = v83 byte-identical on v61/v72/v79/v84/v92** (14-byte reflect extension, same `reflect!=0 || block!=0` gate; v92 has **no** trailing `bGuard` byte — verified independently, **NOT** inherited from v87). **Only v48 diverges** (op 0x27): no `nMagicElemAttr` byte, 10-byte extension (no `charX`/`charY`), `stanceFlags` in the mob branch only. Legacy non-mob/obstacle branch (v48/61/72/79) has no trailing stance byte. → Task 1 adds one `GMS && MajorVersion() < 61` decode branch + v48 fixtures; nothing else in the codec changes.
   - **All mitigation-formula version gates already in Task 6 are verified correct across all ten columns** — Power-Guard cap `/10` (pre-BB) vs `/2` (GMS v95), fixedDamage `min` (pre-BB) vs replace (v95/jms), Magic-Shield `>=87` form, v95-only GUARD-suppress-PG + Mechanic Perfect Armor (confirmed absent on v92). **No Task 6 formula code changes.** v48 additionally *omits* the fixedDamage clamp, the Power-Guard invincibility-zero, and the Mana-Reflection MaxHP/20 cap; the server applies all three universally (they only bound reflect downward — safe) so no v48 formula gate is added.
   - **Per-version skill availability is auto-handled** by the data-driven, server-authoritative pipeline: absent skills (Magic Guard pre-≈v72, Aran pre-≈v76, Evan pre-v84) yield no buff/level → no-op; Achilles is client-dead-code on the v48/v61 **DEVM builds** but the server applies it from skill data and its authoritative `CHANGE_HP` is what renders. No per-version gates.
   - Verification limitation recorded honestly: the legacy IDBs are DEVM builds (as are the v83/v87/v95 IDBs the original design used — wire format is structural and reliable); and the Magic-Shield v83-vs-v87 form is stat-cookie-driven and could not be re-corroborated by the legacy immediate-search pass, so the design-phase `>=87` finding is retained (one Evan-only gate, narrow band).

8. **gms_87/92/95/jms never routed `CharacterDamageHandle`** (pre-existing gap — absent at branch base too), so the verified v87/v92/v95/jms behavior is currently unreachable. Task 7 wires all four at their serverbound `TAKE_DAMAGE` opcodes — v87=0x32, v92=0x35 (from the v92 IDB; v92 is not in the packet registry), v95=0x34, jms=0x27 — each verified **free of handler-opcode collision**, with `LoggedInValidator` (a handler entry with no validator is silently dropped) at its sorted `opCode` position (template-opcode-order-guard).

Verified formula/value grounding for tests (v83 Skill.wz, Cosmic dump): Achilles 1120004 `x` per-mille 995 (lvl 1) → 850 (lvl 30); High Defense 21120004 identical scale; Magic Guard 2001002 `x` percent 11 → 80; Meso Guard 4211005 `x` (cost rate) 90 → 81; Mana Reflection 2121002 `x` 55 → 140 with `prop` 31+; Combo Barrier 21120007 `x` per-mille 916 → 864.

## File Structure

```
libs/atlas-packet/model/damage_taken_info.go          Task 1  rewrite decode/encode + renames
libs/atlas-packet/model/damage_taken_info_test.go     Task 1  round-trips + raw byte fixtures
services/atlas-data/atlas.com/data/monster/rest.go    Task 2  + FixedDamage field
services/atlas-data/atlas.com/data/monster/reader.go  Task 2  + fixedDamage parse
services/atlas-data/atlas.com/data/monster/reader_test.go Task 2  + fixture test
services/atlas-channel/atlas.com/channel/data/monster/{requests,rest,model,processor}.go
                                                      Task 3  NEW template client (Boss/FixedDamage)
services/atlas-channel/atlas.com/channel/data/monster/rest_test.go Task 3
services/atlas-channel/atlas.com/channel/data/skill/registry.go     Task 4  NEW tenant-scoped cache
services/atlas-channel/atlas.com/channel/data/skill/processor.go    Task 4  read-through in GetById
services/atlas-channel/atlas.com/channel/data/skill/registry_test.go Task 4
services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go Task 5  + REQUEST_CHANGE_MESO
services/atlas-channel/atlas.com/channel/character/producer.go      Task 5  + provider
services/atlas-channel/atlas.com/channel/character/processor.go     Task 5  + interface method + impl
services/atlas-channel/atlas.com/channel/character/mock/processor.go Task 5  + mock method
services/atlas-channel/atlas.com/channel/socket/handler/character_damage_mitigation.go
                                                      Task 6  NEW pure chain + types
services/atlas-channel/atlas.com/channel/socket/handler/character_damage_mitigation_test.go
                                                      Task 6  unit tests
services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go
                                                      Task 7  orchestrator rework
services/atlas-channel/atlas.com/channel/socket/handler/character_damage_test.go
                                                      Task 7  deps-fake emission tests
services/atlas-channel/atlas.com/channel/socket/model/damage_taken_info.go
                                                      Task 7  DELETE (dead code)
services/atlas-configurations/seed-data/templates/template_gms_87_1.json   Task 7  + CharacterDamageHandle @0x32
services/atlas-configurations/seed-data/templates/template_gms_92_1.json   Task 7  + CharacterDamageHandle @0x35
services/atlas-configurations/seed-data/templates/template_gms_95_1.json   Task 7  + CharacterDamageHandle @0x34
services/atlas-configurations/seed-data/templates/template_jms_185_1.json  Task 7  + CharacterDamageHandle @0x27
```

---

### Task 1: Packet codec — conditional reflect extension, mob-branch fix, field renames

**Files:**
- Modify: `libs/atlas-packet/model/damage_taken_info.go` (full struct/codec rewrite below)
- Modify: `libs/atlas-packet/model/damage_taken_info_test.go`

**Interfaces:**
- Consumes: `request.Reader` (`ReadUint32/ReadInt8/ReadInt32/ReadBool/ReadByte/ReadInt16/Available`), `response.Writer`, `tenant.MustFromContext`.
- Produces (getters later tasks rely on): `AttackIdx() DamageType`, `Damage() int32`, `MonsterTemplateId() uint32`, `MonsterId() uint32`, `Left() bool`, `Reflect() byte`, `Guard() bool`, `BlockByte() byte`, `HasReflectExtension() bool`, `IsPowerGuard() bool`, `ReflectTargetMobId() uint32`, `HitAction() byte`, `HitX()/HitY()/CharacterX()/CharacterY() int16`, `StanceFlags() byte`, `ObstacleData() int16`, `UpdateTime() uint32`, `MagicElemAttr() DamageElementType`, `CharacterId() uint32`, `Operation() string`, `String() string`. Constants `DamageTypeMagic/Physical/Counter/Obstacle/Stat` unchanged.

Rename map (verified meanings, design §2): `nX`→`reflect` (`Reflect()`), `relativeDir`→`blockByte` (`BlockByte()`), `bPowerGuard`→`isPowerGuard` (`IsPowerGuard()`), `monsterId2`→`reflectTargetMobId` (`ReflectTargetMobId()`), `powerGuard bool`→`hitAction byte` (`HitAction()` — the client encodes `CMob::GetRandomHitAction`, a byte, not a bool), `expression`→`stanceFlags` (`StanceFlags()`), `bGuard`→`guard` (getter stays `Guard()`). New field `hasReflectExtension bool` records extension presence so Encode mirrors Decode byte-exactly.

- [ ] **Step 1: Write the failing tests**

Replace the body of `libs/atlas-packet/model/damage_taken_info_test.go` with (keep the package/imports header; full file):

```go
package model

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/sirupsen/logrus/hooks/test"
)

// mobHit builds a mob-sourced DamageTakenInfo. withExt controls the optional
// 14-byte reflect extension (present on the wire iff the client set
// bKnockback or nX — modeled by hasReflectExtension).
func mobHit(withExt bool, blockByte byte, reflect byte) DamageTakenInfo {
	m := DamageTakenInfo{
		characterId:         100,
		updateTime:          12345,
		nAttackIdx:          DamageTypePhysical,
		nMagicElemAttr:      DamageElementTypeFire,
		damage:              500,
		monsterTemplateId:   200100,
		monsterId:           42,
		left:                true,
		reflect:             reflect,
		guard:               true,
		blockByte:           blockByte,
		hasReflectExtension: withExt,
		stanceFlags:         1,
	}
	if withExt {
		m.isPowerGuard = true
		m.reflectTargetMobId = 42
		m.hitAction = 3
		m.hitX = 100
		m.hitY = 200
		m.characterX = 110
		m.characterY = 210
	}
	return m
}

func assertCommon(t *testing.T, in, out DamageTakenInfo) {
	t.Helper()
	if out.UpdateTime() != in.UpdateTime() {
		t.Errorf("updateTime: got %v, want %v", out.UpdateTime(), in.UpdateTime())
	}
	if out.AttackIdx() != in.AttackIdx() {
		t.Errorf("nAttackIdx: got %v, want %v", out.AttackIdx(), in.AttackIdx())
	}
	if out.MagicElemAttr() != in.MagicElemAttr() {
		t.Errorf("nMagicElemAttr: got %v, want %v", out.MagicElemAttr(), in.MagicElemAttr())
	}
	if out.Damage() != in.Damage() {
		t.Errorf("damage: got %v, want %v", out.Damage(), in.Damage())
	}
}

func assertMob(t *testing.T, in, out DamageTakenInfo) {
	t.Helper()
	assertCommon(t, in, out)
	if out.MonsterTemplateId() != in.MonsterTemplateId() {
		t.Errorf("monsterTemplateId: got %v, want %v", out.MonsterTemplateId(), in.MonsterTemplateId())
	}
	if out.MonsterId() != in.MonsterId() {
		t.Errorf("monsterId: got %v, want %v", out.MonsterId(), in.MonsterId())
	}
	if out.Left() != in.Left() {
		t.Errorf("left: got %v, want %v", out.Left(), in.Left())
	}
	if out.Reflect() != in.Reflect() {
		t.Errorf("reflect: got %v, want %v", out.Reflect(), in.Reflect())
	}
	if out.BlockByte() != in.BlockByte() {
		t.Errorf("blockByte: got %v, want %v", out.BlockByte(), in.BlockByte())
	}
	if out.HasReflectExtension() != in.HasReflectExtension() {
		t.Errorf("hasReflectExtension: got %v, want %v", out.HasReflectExtension(), in.HasReflectExtension())
	}
	if out.StanceFlags() != in.StanceFlags() {
		t.Errorf("stanceFlags: got %v, want %v", out.StanceFlags(), in.StanceFlags())
	}
	if in.HasReflectExtension() {
		if out.IsPowerGuard() != in.IsPowerGuard() {
			t.Errorf("isPowerGuard: got %v, want %v", out.IsPowerGuard(), in.IsPowerGuard())
		}
		if out.ReflectTargetMobId() != in.ReflectTargetMobId() {
			t.Errorf("reflectTargetMobId: got %v, want %v", out.ReflectTargetMobId(), in.ReflectTargetMobId())
		}
		if out.HitAction() != in.HitAction() {
			t.Errorf("hitAction: got %v, want %v", out.HitAction(), in.HitAction())
		}
		if out.HitX() != in.HitX() || out.HitY() != in.HitY() {
			t.Errorf("hit point: got (%d,%d), want (%d,%d)", out.HitX(), out.HitY(), in.HitX(), in.HitY())
		}
		if out.CharacterX() != in.CharacterX() || out.CharacterY() != in.CharacterY() {
			t.Errorf("character point: got (%d,%d), want (%d,%d)", out.CharacterX(), out.CharacterY(), in.CharacterX(), in.CharacterY())
		}
	}
}

// Plain hit: no reflect, no block. The old decoder read the 14-byte
// extension unconditionally here and over-ran the packet; RoundTrip's
// Available()==0 assertion is the regression guard.
func TestDamageTakenInfoPlainHitRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := mobHit(false, 0, 0)
			output := DamageTakenInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			assertMob(t, input, output)
		})
	}
}

// Guardian block of a mob SKILL attack: blockByte==1 (blocked, no
// knockback) with NO extension. A decoder keyed on blockByte!=0 would
// over-read 14 bytes here.
func TestDamageTakenInfoBlockWithoutKnockbackRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := mobHit(false, 1, 0)
			output := DamageTakenInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			assertMob(t, input, output)
		})
	}
}

func TestDamageTakenInfoReflectExtensionRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := mobHit(true, 2, 30)
			output := DamageTakenInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			assertMob(t, input, output)
			if v.Region == "GMS" && v.MajorVersion >= 95 {
				if output.Guard() != input.Guard() {
					t.Errorf("guard: got %v, want %v", output.Guard(), input.Guard())
				}
			}
		})
	}
}

// Mob skill attacks carry the attack slot index (>= 1) — the old decoder
// misrouted these into the obstacle branch.
func TestDamageTakenInfoMobSkillAttackIdxRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := mobHit(false, 0, 0)
			input.nAttackIdx = DamageType(1)
			output := DamageTakenInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			assertMob(t, input, output)
		})
	}
}

func TestDamageTakenInfoNonMobRoundTrip(t *testing.T) {
	for _, attackIdx := range []DamageType{DamageTypeCounter, DamageTypeObstacle, DamageTypeStat} {
		for _, v := range pt.Variants {
			t.Run(v.Name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				input := DamageTakenInfo{
					characterId:    100,
					updateTime:     999,
					nAttackIdx:     attackIdx,
					nMagicElemAttr: DamageElementTypeNone,
					damage:         120,
					obstacleData:   7,
					stanceFlags:    0,
				}
				output := DamageTakenInfo{}
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				assertCommon(t, input, output)
				if output.ObstacleData() != input.ObstacleData() {
					t.Errorf("obstacleData: got %v, want %v", output.ObstacleData(), input.ObstacleData())
				}
			})
		}
	}
}

// Sentinel −1 damage (Guardian/Fake block) must decode unchanged.
func TestDamageTakenInfoSentinelDamageRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := mobHit(false, 1, 0)
			input.damage = -1
			output := DamageTakenInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Damage() != -1 {
				t.Errorf("damage: got %v, want -1", output.Damage())
			}
		})
	}
}

// Raw byte fixtures pin the exact v83 wire layout (no bGuard byte,
// little-endian): a 22-byte plain hit and a 36-byte extension hit.
func TestDamageTakenInfoV83ByteFixtures(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	l, _ := test.NewNullLogger()

	plain := []byte{
		0x39, 0x30, 0x00, 0x00, // updateTime 12345
		0xFF,                   // nAttackIdx -1 (touch)
		0x02,                   // nMagicElemAttr fire
		0xF4, 0x01, 0x00, 0x00, // damage 500
		0x04, 0x0D, 0x03, 0x00, // mobTemplateId 200004
		0x2A, 0x00, 0x00, 0x00, // mobId 42
		0x01,                   // left
		0x00,                   // reflect
		0x01,                   // blockByte 1 (block, no knockback -> NO extension)
		0x05,                   // stanceFlags
	}
	req := request.Request(plain)
	reader := request.NewRequestReader(&req, 0)
	m := DamageTakenInfo{}
	m.Decode(l, ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Fatalf("plain fixture: %d unconsumed bytes", reader.Available())
	}
	if m.Damage() != 500 || m.MonsterTemplateId() != 200004 || m.MonsterId() != 42 ||
		m.BlockByte() != 1 || m.HasReflectExtension() || m.StanceFlags() != 5 {
		t.Fatalf("plain fixture decoded wrong: %s", m.String())
	}

	ext := []byte{
		0x39, 0x30, 0x00, 0x00,
		0xFF,
		0x00,
		0xF4, 0x01, 0x00, 0x00,
		0x04, 0x0D, 0x03, 0x00,
		0x2A, 0x00, 0x00, 0x00,
		0x01,
		0x1E,                   // reflect 30 (Power Guard percent echo)
		0x00,                   // blockByte 0
		0x01,                   // isPowerGuard
		0x2A, 0x00, 0x00, 0x00, // reflectTargetMobId 42
		0x03,                   // hitAction
		0x64, 0x00,             // hitX 100
		0xC8, 0x00,             // hitY 200
		0x6E, 0x00,             // characterX 110
		0xD2, 0x00,             // characterY 210
		0x00,                   // stanceFlags
	}
	req2 := request.Request(ext)
	reader2 := request.NewRequestReader(&req2, 0)
	m2 := DamageTakenInfo{}
	m2.Decode(l, ctx)(&reader2, nil)
	if reader2.Available() != 0 {
		t.Fatalf("ext fixture: %d unconsumed bytes", reader2.Available())
	}
	if !m2.HasReflectExtension() || !m2.IsPowerGuard() || m2.ReflectTargetMobId() != 42 ||
		m2.Reflect() != 30 || m2.HitAction() != 3 || m2.HitX() != 100 || m2.CharacterY() != 210 {
		t.Fatalf("ext fixture decoded wrong: %s", m2.String())
	}
}

// Raw byte fixtures pin the DIVERGENT v48 wire layout (design §2a): no
// nMagicElemAttr byte, a 10-byte reflect extension (no charX/charY), and a
// non-mob branch with no trailing stanceFlags byte. These are the fixtures
// that would silently pass on a v83-only decoder and fail on the real wire.
func TestDamageTakenInfoV48ByteFixtures(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	l, _ := test.NewNullLogger()

	// Mob-hit with reflect extension: 31 bytes (v83's equivalent is 34 —
	// v48 drops the 1-byte magicElemAttr and the 4-byte charX/charY pair).
	ext := []byte{
		0x39, 0x30, 0x00, 0x00, // updateTime 12345
		0xFF,                   // nAttackIdx -1 (touch)
		// (no nMagicElemAttr byte on v48)
		0xF4, 0x01, 0x00, 0x00, // damage 500
		0x04, 0x0D, 0x03, 0x00, // mobTemplateId 200004
		0x2A, 0x00, 0x00, 0x00, // mobId 42
		0x01,                   // left
		0x1E,                   // reflect 30
		0x00,                   // blockByte 0
		0x01,                   // isPowerGuard
		0x2A, 0x00, 0x00, 0x00, // reflectTargetMobId 42
		0x03,                   // hitAction
		0x64, 0x00,             // hitX 100
		0xC8, 0x00,             // hitY 200
		// (no charX/charY on v48)
		0x05,                   // stanceFlags (mob branch)
	}
	req := request.Request(ext)
	reader := request.NewRequestReader(&req, 0)
	m := DamageTakenInfo{}
	m.Decode(l, ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Fatalf("v48 ext fixture: %d unconsumed bytes", reader.Available())
	}
	if m.Damage() != 500 || m.MonsterTemplateId() != 200004 || !m.HasReflectExtension() ||
		!m.IsPowerGuard() || m.HitY() != 200 || m.CharacterX() != 0 || m.StanceFlags() != 5 {
		t.Fatalf("v48 ext fixture decoded wrong: %s", m.String())
	}

	// Obstacle/non-mob: 11 bytes, no trailing stance (design §2a).
	obstacle := []byte{
		0x39, 0x30, 0x00, 0x00, // updateTime
		0xFE,                   // nAttackIdx -2 (non-mob sentinel)
		// (no nMagicElemAttr byte on v48)
		0xF4, 0x01, 0x00, 0x00, // damage 500
		0x07, 0x00,             // obstacleData 7
		// (no trailing stanceFlags on pre-v83 non-mob)
	}
	req2 := request.Request(obstacle)
	reader2 := request.NewRequestReader(&req2, 0)
	m2 := DamageTakenInfo{}
	m2.Decode(l, ctx)(&reader2, nil)
	if reader2.Available() != 0 {
		t.Fatalf("v48 obstacle fixture: %d unconsumed bytes", reader2.Available())
	}
	if m2.Damage() != 500 || m2.ObstacleData() != 7 || m2.HasReflectExtension() {
		t.Fatalf("v48 obstacle fixture decoded wrong: %s", m2.String())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./model/ -run TestDamageTakenInfo -v`
Expected: FAIL — compile errors (`m.reflect`, `m.blockByte`, `HasReflectExtension` undefined). That is the correct failure mode for a rename+rewrite.

- [ ] **Step 3: Rewrite the model**

Replace the struct, getters, `String()`, `Decode`, and `Encode` in `libs/atlas-packet/model/damage_taken_info.go` (constants, package header, `NewDamageTakenInfo`, and `Operation()` stay as-is):

```go
type DamageTakenInfo struct {
	characterId       uint32
	updateTime        uint32
	nAttackIdx        DamageType
	nMagicElemAttr    DamageElementType
	damage            int32
	obstacleData      int16
	monsterTemplateId uint32
	monsterId         uint32
	left              bool
	reflect           byte
	guard             bool
	blockByte         byte
	// hasReflectExtension mirrors the client's variable-length tail: the
	// 14-byte reflect extension is written iff the client set bKnockback
	// or a non-zero reflect echo (CUserLocal::SetDamaged, verified v83/
	// v87/v95/jms185).
	hasReflectExtension bool
	isPowerGuard        bool
	reflectTargetMobId  uint32
	hitAction           byte
	hitX                int16
	hitY                int16
	characterX          int16
	characterY          int16
	stanceFlags         byte
}

func (m DamageTakenInfo) CharacterId() uint32                { return m.characterId }
func (m DamageTakenInfo) UpdateTime() uint32                 { return m.updateTime }
func (m DamageTakenInfo) AttackIdx() DamageType              { return m.nAttackIdx }
func (m DamageTakenInfo) MagicElemAttr() DamageElementType   { return m.nMagicElemAttr }
func (m DamageTakenInfo) Damage() int32                      { return m.damage }
func (m DamageTakenInfo) ObstacleData() int16                { return m.obstacleData }
func (m DamageTakenInfo) MonsterTemplateId() uint32          { return m.monsterTemplateId }
func (m DamageTakenInfo) MonsterId() uint32                  { return m.monsterId }
func (m DamageTakenInfo) Left() bool                         { return m.left }
func (m DamageTakenInfo) Reflect() byte                      { return m.reflect }
func (m DamageTakenInfo) Guard() bool                        { return m.guard }
func (m DamageTakenInfo) BlockByte() byte                    { return m.blockByte }
func (m DamageTakenInfo) HasReflectExtension() bool          { return m.hasReflectExtension }
func (m DamageTakenInfo) IsPowerGuard() bool                 { return m.isPowerGuard }
func (m DamageTakenInfo) ReflectTargetMobId() uint32         { return m.reflectTargetMobId }
func (m DamageTakenInfo) HitAction() byte                    { return m.hitAction }
func (m DamageTakenInfo) HitX() int16                        { return m.hitX }
func (m DamageTakenInfo) HitY() int16                        { return m.hitY }
func (m DamageTakenInfo) CharacterX() int16                  { return m.characterX }
func (m DamageTakenInfo) CharacterY() int16                  { return m.characterY }
func (m DamageTakenInfo) StanceFlags() byte                  { return m.stanceFlags }

func (m DamageTakenInfo) String() string {
	return fmt.Sprintf("characterId [%d], updateTime [%d], nAttackIdx [%d], nMagicElemAttr [%d], damage [%d], obstacleData [%d], monsterTemplate [%d], monsterId [%d], left [%t], reflect [%d], guard [%t], blockByte [%d], hasReflectExtension [%t], isPowerGuard [%t], reflectTargetMobId [%d], hitAction [%d], hit [%d,%d], character [%d,%d], stanceFlags [%d]",
		m.characterId, m.updateTime, m.nAttackIdx, m.nMagicElemAttr, m.damage, m.obstacleData, m.monsterTemplateId, m.monsterId, m.left, m.reflect, m.guard, m.blockByte, m.hasReflectExtension, m.isPowerGuard, m.reflectTargetMobId, m.hitAction, m.hitX, m.hitY, m.characterX, m.characterY, m.stanceFlags)
}

// Legacy layout gates (design §2a, verified per-version IDBs):
//   preV61Layout  — gms_v48 only: NO nMagicElemAttr byte; the reflect
//                   extension is 10 bytes (no charX/charY).
//   preV83NonMob  — gms_v48/v61/v72/v79: the non-mob (obstacle/stat)
//                   branch has NO trailing stanceFlags byte.
// v61 through v92 decode as v83 (mob-hit byte-identical). The mob branch's
// trailing stanceFlags is present on every version. v95-GMS adds the bGuard
// byte (gated >=95); jms takes the no-bGuard branch.
func gmsBelow(t tenant.Model, major uint16) bool {
	return t.Region() == "GMS" && t.MajorVersion() < major
}

func (m *DamageTakenInfo) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	preV61Layout := gmsBelow(t, 61)
	preV83NonMob := gmsBelow(t, 83)
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.nAttackIdx = DamageType(r.ReadInt8())
		if !preV61Layout {
			m.nMagicElemAttr = DamageElementType(r.ReadInt8())
		}
		m.damage = r.ReadInt32()

		if m.nAttackIdx >= DamageTypePhysical {
			m.monsterTemplateId = r.ReadUint32()
			m.monsterId = r.ReadUint32()
			m.left = r.ReadBool()

			m.reflect = r.ReadByte()
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				m.guard = r.ReadBool()
			}
			m.blockByte = r.ReadByte()
			// The client writes the reflect extension iff it set bKnockback
			// or a reflect echo. Neither flag is fully recoverable from
			// earlier bytes, so presence is derived from the remaining
			// length: without the extension exactly the 1-byte stance
			// remains; with it, 11 bytes (v48, 10-byte ext) or 15 bytes
			// (v61+, 14-byte ext) remain — so Available() > 1 detects it on
			// every version.
			if r.Available() > 1 {
				m.hasReflectExtension = true
				m.isPowerGuard = r.ReadBool()
				m.reflectTargetMobId = r.ReadUint32()
				m.hitAction = r.ReadByte()
				m.hitX = r.ReadInt16()
				m.hitY = r.ReadInt16()
				if !preV61Layout {
					m.characterX = r.ReadInt16()
					m.characterY = r.ReadInt16()
				}
			}
			m.stanceFlags = r.ReadByte()
		} else {
			m.obstacleData = r.ReadInt16()
			if !preV83NonMob {
				m.stanceFlags = r.ReadByte()
			}
		}
	}
}

func (m DamageTakenInfo) Encode(_ logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(logrus.WithFields(logrus.Fields{}))
	t := tenant.MustFromContext(ctx)
	preV61Layout := gmsBelow(t, 61)
	preV83NonMob := gmsBelow(t, 83)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt8(int8(m.nAttackIdx))
		if !preV61Layout {
			w.WriteInt8(int8(m.nMagicElemAttr))
		}
		w.WriteInt32(m.damage)

		if m.nAttackIdx >= DamageTypePhysical {
			w.WriteInt(m.monsterTemplateId)
			w.WriteInt(m.monsterId)
			w.WriteBool(m.left)

			w.WriteByte(m.reflect)
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				w.WriteBool(m.guard)
			}
			w.WriteByte(m.blockByte)
			if m.hasReflectExtension {
				w.WriteBool(m.isPowerGuard)
				w.WriteInt(m.reflectTargetMobId)
				w.WriteByte(m.hitAction)
				w.WriteInt16(m.hitX)
				w.WriteInt16(m.hitY)
				if !preV61Layout {
					w.WriteInt16(m.characterX)
					w.WriteInt16(m.characterY)
				}
			}
			w.WriteByte(m.stanceFlags)
		} else {
			w.WriteInt16(m.obstacleData)
			if !preV83NonMob {
				w.WriteByte(m.stanceFlags)
			}
		}
		return w.Bytes()
	}
}
```

**Legacy round-trip note (affects Step 1 test).** `pt.Variants` includes v28/v48/v61/v72/v79, and any subtest that loops the variant set now iterates them. Under the gates above the GMS `< 61` variants (v28, v48) take the 10-byte-extension / no-`nMagicElemAttr` path, so the shared `mobHit(...)` fixture's `nMagicElemAttr`, `characterX`, and `characterY` fields are dropped on encode and stay zero on decode. The round-trip is internally self-consistent (encode→decode under the same gate) and `pt.RoundTrip`'s `Available()==0` guard still holds, but the field-equality helpers do not: `assertCommon` compares `MagicElemAttr()` (line ~150) and `assertMob`'s extension block compares `CharacterX()/CharacterY()` (line ~195). Add a `legacyLayout(v TenantVariant) bool { return v.Region=="GMS" && v.MajorVersion < 61 }` predicate and, for legacy variants, either (a) build the `mobHit` input with `nMagicElemAttr`/`characterX`/`characterY` zeroed so the equality holds, or (b) pass a flag into `assertCommon`/`assertMob` that skips exactly those three comparisons. Do NOT weaken the assertions for non-legacy variants. Also add `{Name:"GMS v92", Region:"GMS", MajorVersion:92, MinorVersion:1}` to `pt.Variants` so every wired column is exercised (v92 = v83 layout — the round-trip covers it, no separate raw fixture needed).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test -race ./model/ -run TestDamageTakenInfo -v`
Expected: PASS (all subtests, all `pt.Variants`).

- [ ] **Step 5: Check for other consumers of the renamed getters**

Run: `grep -rn "\.NX()\|\.RelativeDir()\|\.PowerGuard()\|\.PowerGuard2()\|\.MonsterId2()\|\.Expression()" --include="*.go" services libs | grep -i damage`
Expected: no output (the only consumer, `character_damage.go`, uses `AttackIdx/Damage/MonsterTemplateId/Left`, which did not change; `atlas-channel/socket/model/damage_taken_info.go` is a self-contained dead copy deleted in Task 7). If anything else surfaces, update it in this task.

Then run the module-wide checks: `cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/model/damage_taken_info.go libs/atlas-packet/model/damage_taken_info_test.go
git commit -m "fix(atlas-packet): conditional reflect extension + mob-branch fix in damage-taken decode

The reflect extension was read unconditionally, over-running every plain
hit by 14 bytes; the mob branch dropped mob skill attacks with index >= 1.
Extension presence is length-derived (IDA-verified v83/v87/v95/jms185).
Fields renamed to their verified meanings."
```

---

### Task 2: atlas-data — ingest monster `fixedDamage`

**Files:**
- Modify: `services/atlas-data/atlas.com/data/monster/rest.go` (add field after `Boss`)
- Modify: `services/atlas-data/atlas.com/data/monster/reader.go` (parse after the `m.Boss` line)
- Modify: `services/atlas-data/atlas.com/data/monster/reader_test.go`

**Interfaces:**
- Produces: `RestModel.FixedDamage uint32` serialized as `"fixed_damage"` — consumed by Task 3's channel client. WZ source node: `<int name="fixedDamage" value="..."/>` under `info` (verified in real Mob.wz XML, e.g. 9300314).

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-data/atlas.com/data/monster/reader_test.go` (same file conventions as `TestReaderMobilityFlags`):

```go
func TestReaderFixedDamage(t *testing.T) {
	tt := testTenant()
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), tt)

	_, _ = GetMonsterStringRegistry().Add(tt, MonsterString{id: strconv.Itoa(9300314), name: "FakeFixed"})

	body := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="9300314.img">
  <imgdir name="info">
    <int name="maxHP" value="100"/>
    <int name="fixedDamage" value="5"/>
  </imgdir>
</imgdir>`

	rm, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(body)))()
	if err != nil {
		t.Fatal(err)
	}
	if rm.FixedDamage != 5 {
		t.Fatalf("FixedDamage=%d, want 5", rm.FixedDamage)
	}

	// Absent node defaults to zero.
	_, _ = GetMonsterStringRegistry().Add(tt, MonsterString{id: strconv.Itoa(9300315), name: "FakeNoFixed"})
	body2 := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="9300315.img">
  <imgdir name="info"><int name="maxHP" value="100"/></imgdir>
</imgdir>`
	rm2, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(body2)))()
	if err != nil {
		t.Fatal(err)
	}
	if rm2.FixedDamage != 0 {
		t.Fatalf("FixedDamage=%d, want 0", rm2.FixedDamage)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-data/atlas.com/data && go test ./monster/ -run TestReaderFixedDamage -v`
Expected: FAIL — `rm.FixedDamage undefined`.

- [ ] **Step 3: Implement**

In `rest.go`, after the `Boss` field (line 20):

```go
	FixedDamage        uint32            `json:"fixed_damage"`
```

In `reader.go`, after the `m.Boss = ...` line (line 64):

```go
			m.FixedDamage = uint32(node.GetIntegerWithDefault("fixedDamage", 0))
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-data/atlas.com/data && go test -race ./monster/ && go vet ./... && go build ./...`
Expected: PASS/clean.

Note (no action, record only): already-ingested tenants have stored documents without the field; they decode as `fixed_damage: 0` (cap simply doesn't bind) until the tenant is re-ingested.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/monster/rest.go services/atlas-data/atlas.com/data/monster/reader.go services/atlas-data/atlas.com/data/monster/reader_test.go
git commit -m "feat(atlas-data): ingest monster fixedDamage for Power Guard reflect cap"
```

---

### Task 3: atlas-channel — monster template data client (`data/monster`)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/data/monster/requests.go`
- Create: `services/atlas-channel/atlas.com/channel/data/monster/rest.go`
- Create: `services/atlas-channel/atlas.com/channel/data/monster/model.go`
- Create: `services/atlas-channel/atlas.com/channel/data/monster/processor.go`
- Create: `services/atlas-channel/atlas.com/channel/data/monster/rest_test.go`

**Interfaces:**
- Consumes: atlas-data route `GET {DATA}/data/monsters/{monsterId}` (JSON:API type `"monsters"`, attributes include `boss`, `fixed_damage` from Task 2).
- Produces: `monster.NewProcessor(l, ctx).GetById(templateId uint32) (Model, error)` with `Model.Boss() bool`, `Model.FixedDamage() uint32`, `Model.Id() uint32`. Handler imports it as `monsterdata "atlas-channel/data/monster"`.

- [ ] **Step 1: Write the failing test**

`rest_test.go`:

```go
package monster

import "testing"

func TestExtract(t *testing.T) {
	rm := RestModel{Boss: true, FixedDamage: 5}
	if err := rm.SetID("8510000"); err != nil {
		t.Fatal(err)
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatal(err)
	}
	if m.Id() != 8510000 {
		t.Errorf("Id=%d, want 8510000", m.Id())
	}
	if !m.Boss() {
		t.Error("Boss=false, want true")
	}
	if m.FixedDamage() != 5 {
		t.Errorf("FixedDamage=%d, want 5", m.FixedDamage())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./data/monster/ -v`
Expected: FAIL — package does not exist / types undefined.

- [ ] **Step 3: Implement the package**

`rest.go`:

```go
package monster

import "strconv"

// RestModel is a projection of atlas-data's monster resource; only the
// fields the damage pipeline needs are declared, the rest of the
// attributes payload is ignored on unmarshal.
type RestModel struct {
	Id          uint32 `json:"-"`
	Boss        bool   `json:"boss"`
	FixedDamage uint32 `json:"fixed_damage"`
}

func (r RestModel) GetName() string {
	return "monsters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:          rm.Id,
		boss:        rm.Boss,
		fixedDamage: rm.FixedDamage,
	}, nil
}
```

`model.go`:

```go
package monster

type Model struct {
	id          uint32
	boss        bool
	fixedDamage uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Boss() bool {
	return m.boss
}

func (m Model) FixedDamage() uint32 {
	return m.fixedDamage
}
```

`requests.go`:

```go
package monster

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	monstersResource = "data/monsters/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(monsterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+monstersResource, monsterId))
}
```

`processor.go`:

```go
package monster

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(monsterId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) GetById(monsterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(monsterId), Extract)()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./data/monster/ -v && go build ./...`
Expected: PASS/clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/data/monster/
git commit -m "feat(atlas-channel): monster template data client (boss flag, fixedDamage)"
```

---

### Task 4: atlas-channel — tenant-scoped skill-data cache

Skill effect data is immutable per tenant and `data/skill.GetById` is an uncached REST call on what becomes the per-hit path (Mana Reflection x/prop, warrior/Aran passive x). Add the project-standard registry (`sync.Once` + `RWMutex`, keyed by `tenant.Model` — same shape as `account/registry.go`) as a read-through inside `GetById`.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/data/skill/registry.go`
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/processor.go:30-32` (`GetById`)
- Create: `services/atlas-channel/atlas.com/channel/data/skill/registry_test.go`

**Interfaces:**
- Produces: `GetCache() *Cache` with `Get(t tenant.Model, skillId uint32) (Model, bool)` and `Put(t tenant.Model, skillId uint32, m Model)`. `Processor.GetById`/`GetEffect` signatures unchanged — callers are unaffected.

- [ ] **Step 1: Write the failing test**

`registry_test.go`:

```go
package skill

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestCachePutGet(t *testing.T) {
	t1 := testTenant(t)
	t2 := testTenant(t)

	if _, ok := GetCache().Get(t1, 2001002); ok {
		t.Fatal("unexpected cache hit before Put")
	}
	m, err := Extract(RestModel{Id: 2001002})
	if err != nil {
		t.Fatal(err)
	}
	GetCache().Put(t1, 2001002, m)

	got, ok := GetCache().Get(t1, 2001002)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Id() != 2001002 {
		t.Errorf("Id=%d, want 2001002", got.Id())
	}
	if _, ok := GetCache().Get(t2, 2001002); ok {
		t.Fatal("cache leaked across tenants")
	}
}

// GetById must serve a cached model without issuing any REST call: no
// DATA base URL is configured in tests, so a cache miss would error.
func TestGetByIdReadsThrough(t *testing.T) {
	tt := testTenant(t)
	ctx := tenant.WithContext(context.Background(), tt)
	l, _ := test.NewNullLogger()

	m, err := Extract(RestModel{Id: 4211005})
	if err != nil {
		t.Fatal(err)
	}
	GetCache().Put(tt, 4211005, m)

	got, err := NewProcessor(l, ctx).GetById(4211005)
	if err != nil {
		t.Fatalf("GetById should hit the cache, got err: %v", err)
	}
	if got.Id() != 4211005 {
		t.Errorf("Id=%d, want 4211005", got.Id())
	}
}
```

(`RestModel.Id uint32` and `Model.Id() uint32` exist — verified in `data/skill/rest.go:11` and `data/skill/model.go:19`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./data/skill/ -run 'TestCache|TestGetByIdReadsThrough' -v`
Expected: FAIL — `GetCache` undefined.

- [ ] **Step 3: Implement**

`registry.go`:

```go
package skill

import (
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Cache holds skill data models per tenant. Skill data is immutable for
// the lifetime of a tenant, so entries are never invalidated.
type Key struct {
	Tenant  tenant.Model
	SkillId uint32
}

type Cache struct {
	mutex  sync.RWMutex
	skills map[Key]Model
}

var cache *Cache
var once sync.Once

func GetCache() *Cache {
	once.Do(func() {
		cache = &Cache{skills: make(map[Key]Model)}
	})
	return cache
}

func (c *Cache) Get(t tenant.Model, skillId uint32) (Model, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	m, ok := c.skills[Key{Tenant: t, SkillId: skillId}]
	return m, ok
}

func (c *Cache) Put(t tenant.Model, skillId uint32, m Model) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.skills[Key{Tenant: t, SkillId: skillId}] = m
}
```

In `processor.go`, replace `GetById` (add the `tenant` import):

```go
func (p *ProcessorImpl) GetById(uniqueId uint32) (Model, error) {
	t := tenant.MustFromContext(p.ctx)
	if m, ok := GetCache().Get(t, uniqueId); ok {
		return m, nil
	}
	m, err := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(uniqueId), Extract)()
	if err != nil {
		return Model{}, err
	}
	GetCache().Put(t, uniqueId, m)
	return m, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./data/skill/... && go build ./...`
Expected: PASS/clean (existing `data/skill` consumers are unaffected — same signature, errors still propagate on miss).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/data/skill/registry.go services/atlas-channel/atlas.com/channel/data/skill/registry_test.go services/atlas-channel/atlas.com/channel/data/skill/processor.go
git commit -m "perf(atlas-channel): tenant-scoped read-through cache for skill data"
```

---

### Task 5: atlas-channel — `REQUEST_CHANGE_MESO` producer

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go` (constant + body next to `CommandRequestDropMeso`/`RequestDropMesoCommandBody`)
- Modify: `services/atlas-channel/atlas.com/channel/character/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/processor.go` (interface at line ~40 + impl next to `RequestDropMeso` at line 267)
- Modify: `services/atlas-channel/atlas.com/channel/character/mock/processor.go`
- Test: `services/atlas-channel/atlas.com/channel/character/producer_test.go` (create if absent)

**Interfaces:**
- Consumes: atlas-character consumer contract, verified at `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:22,127-132` — `Type: "REQUEST_CHANGE_MESO"`, body `{actorId uint32, actorType string, amount int32, showEffect bool}`. The channel `Command` envelope has no `TransactionId`; the consumer decodes it as zero-uuid, which is the already-shipping behavior for `REQUEST_DROP_MESO` from this same file.
- Produces: `Processor.RequestChangeMeso(f field.Model, characterId uint32, actorId uint32, actorType string, amount int32) error` (negative amount = deduction). Task 7 calls it with `actorType="SKILL"`, `actorId=characterId`.

- [ ] **Step 1: Write the failing test**

Create/extend `producer_test.go` in package `character`:

```go
package character

import (
	"encoding/json"
	"testing"

	character2 "atlas-channel/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func TestRequestChangeMesoCommandProvider(t *testing.T) {
	f := field.NewBuilder(2, 1, 100000000).Build()
	msgs, err := RequestChangeMesoCommandProvider(f, 42, 42, "SKILL", -1500)()
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages=%d, want 1", len(msgs))
	}
	var cmd character2.Command[character2.RequestChangeMesoBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatal(err)
	}
	if cmd.Type != character2.CommandRequestChangeMeso {
		t.Errorf("type=%s, want %s", cmd.Type, character2.CommandRequestChangeMeso)
	}
	if cmd.CharacterId != 42 || cmd.WorldId != 2 {
		t.Errorf("envelope=%+v", cmd)
	}
	if cmd.Body.Amount != -1500 || cmd.Body.ActorId != 42 || cmd.Body.ActorType != "SKILL" || cmd.Body.ShowEffect {
		t.Errorf("body=%+v", cmd.Body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/ -run TestRequestChangeMesoCommandProvider -v`
Expected: FAIL — `RequestChangeMesoCommandProvider`/`CommandRequestChangeMeso` undefined.

- [ ] **Step 3: Implement**

`kafka/message/character/kafka.go` — in the const block after `CommandRequestDropMeso`:

```go
	CommandRequestChangeMeso   = "REQUEST_CHANGE_MESO"
```

and after `RequestDropMesoCommandBody`:

```go
type RequestChangeMesoBody struct {
	ActorId    uint32 `json:"actorId"`
	ActorType  string `json:"actorType"`
	Amount     int32  `json:"amount"`
	ShowEffect bool   `json:"showEffect"`
}
```

`character/producer.go` — after `RequestDropMesoCommandProvider`:

```go
func RequestChangeMesoCommandProvider(f field.Model, characterId uint32, actorId uint32, actorType string, amount int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character.Command[character.RequestChangeMesoBody]{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		Type:        character.CommandRequestChangeMeso,
		Body: character.RequestChangeMesoBody{
			ActorId:   actorId,
			ActorType: actorType,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

`character/processor.go` — interface method (next to `RequestDropMeso` at line ~40):

```go
	RequestChangeMeso(f field.Model, characterId uint32, actorId uint32, actorType string, amount int32) error
```

impl (next to the `RequestDropMeso` impl at line ~267):

```go
func (p *ProcessorImpl) RequestChangeMeso(f field.Model, characterId uint32, actorId uint32, actorType string, amount int32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(RequestChangeMesoCommandProvider(f, characterId, actorId, actorType, amount))
}
```

`character/mock/processor.go` — next to the `RequestDropMeso` mock (line ~115):

```go
func (m *MockProcessor) RequestChangeMeso(_ field.Model, _ uint32, _ uint32, _ string, _ int32) error {
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./character/... && go build ./...`
Expected: PASS/clean (the mock compile-checks the interface).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go services/atlas-channel/atlas.com/channel/character/producer.go services/atlas-channel/atlas.com/channel/character/processor.go services/atlas-channel/atlas.com/channel/character/mock/processor.go services/atlas-channel/atlas.com/channel/character/producer_test.go
git commit -m "feat(atlas-channel): REQUEST_CHANGE_MESO producer for Meso Guard"
```

---

### Task 6: Mitigation chain — pure functions

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_mitigation.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_mitigation_test.go`

**Interfaces:**
- Produces (Task 7 consumes): `mitigationInput`, `mobInfo`, `mitigationResult`, `reflectIntent`, `computeMitigation(in mitigationInput, mob mobInfo) mitigationResult`, `clampDamage(raw int32) (clamped int32, adjusted bool)`, `clampInt16(v int32) int16`, constants `reflectAttackTypePhysical byte = 0` / `reflectAttackTypeMagic byte = 2` (values match `packetmodel.AttackTypeMelee`/`AttackTypeMagic`, which is what atlas-monsters' `checkReflect` keys its mob-counter logic on), `maxLegitimateDamage int32 = 999999` (the client's own CalcDamage clamp, design §8).

- [ ] **Step 1: Write the implementation file** (pure math first — the tests in Step 2 are the interesting artifact; both land before any run)

`character_damage_mitigation.go`:

```go
package handler

import (
	"math"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

const (
	// maxLegitimateDamage is the client's own hard damage cap in every
	// verified version's CalcDamage (design task-157 §8).
	maxLegitimateDamage = int32(999999)

	// reflect attack types feed atlas-monsters' Damage command; the values
	// mirror packetmodel.AttackTypeMelee / AttackTypeMagic, which is what
	// the mob-side counter-buff check distinguishes on.
	reflectAttackTypePhysical = byte(packetmodel.AttackTypeMelee)
	reflectAttackTypeMagic    = byte(packetmodel.AttackTypeMagic)
)

// mitigationInput carries everything computeMitigation needs, with wire
// cross-checks already validated against server-side buff state and all
// tenant-version gates pre-resolved, so the math is a pure function.
type mitigationInput struct {
	attackIdx  packetmodel.DamageType
	rawDamage  int32 // already clamped, >= 0
	mobSourced bool  // attackIdx >= DamageTypePhysical

	// powerGuardSignal: wire isPowerGuard AND server-side POWER_GUARD buff
	// AND mob-sourced. manaReflectSignal: wire reflect echo without
	// isPowerGuard AND server-side MANA_REFLECTION buff AND mob skill
	// attack (attackIdx >= 0). The client rolls Mana Reflection's prop;
	// the validated signal is honored, amounts are always recomputed.
	powerGuardSignal  bool
	manaReflectSignal bool

	currentMP uint16
	meso      uint32

	// Buff statup amounts (0 = buff absent).
	magicGuardPct        int32
	infinity             bool
	powerGuardPct        int32
	mesoGuardPct         int32
	comboBarrierPermille int32
	magicShieldPct       int32

	// Passive/effect-derived values (0 = absent).
	achillesPermille int32 // Achilles or Aran High Defense x, job-selected
	manaReflectPct   int32

	// Version gates, resolved from the tenant by the orchestrator.
	// Post-merge legacy verification (design §3) confirmed all three gates
	// hold across every column v48..jms with NO code change: the pre-BB
	// legacy versions (v48/61/72/79/84/92) all use pgCapDivisor 10,
	// fixedDamage min, and fall below the >=95 GUARD/Mechanic rule; v92 was
	// verified against its own IDB (NOT inherited from v87) and is
	// byte-identical to v83 here. (v48 additionally OMITS the fixedDamage
	// clamp / PG invincibility-zero / MR MaxHP/20 cap client-side; the
	// server applies all three universally since they only bound a reflect
	// downward — safe, so no v48-specific gate is added.)
	magicShieldOnReducedDamage bool  // MajorVersion >= 87: base is damage minus the Magic Guard portion
	pgCapDivisor               int32 // 2 on GMS >= 95, else 10 (IDA-verified across all 10 columns)
	pgFixedDamageOverride      bool  // GMS >= 95 or JMS: template fixedDamage replaces the reflect instead of min()
}

type mobInfo struct {
	present     bool
	alive       bool
	maxHp       uint32
	boss        bool
	fixedDamage uint32
}

type reflectIntent struct {
	amount     uint32
	attackType byte
}

type mitigationBreakdown struct {
	achillesReduce     int32
	comboBarrierReduce int32
	magicShieldReduce  int32
	magicGuardAbsorbed int32
	mesoGuarded        int32
	powerGuardReflect  int32
}

type mitigationResult struct {
	hpLoss    int32
	mpLoss    int32
	mesoCost  int32
	reflect   reflectIntent // amount 0 = none
	breakdown mitigationBreakdown
}

// clampDamage bounds the client-supplied damage per FR-10.1. The -1 block
// sentinel is handled by the caller before clamping.
func clampDamage(raw int32) (int32, bool) {
	if raw < 0 {
		return 0, true
	}
	if raw > maxLegitimateDamage {
		return maxLegitimateDamage, true
	}
	return raw, false
}

// clampInt16 bounds an int32 delta to the CHANGE_HP/CHANGE_MP int16
// contract (FR-10.2 — replaces the silent int16 truncation).
func clampInt16(v int32) int16 {
	if v > math.MaxInt16 {
		return math.MaxInt16
	}
	if v < math.MinInt16 {
		return math.MinInt16
	}
	return int16(v)
}

// computeMitigation is the server mirror of the client's damage-taken
// math (design task-157 §6, IDA-verified v83/v87/v95/jms185). Integer
// arithmetic follows the decompiled formulas exactly.
func computeMitigation(in mitigationInput, mob mobInfo) mitigationResult {
	var r mitigationResult
	raw := in.rawDamage
	if raw <= 0 {
		return r
	}

	var achillesReduce int32
	if in.achillesPermille > 0 {
		achillesReduce = raw * (1000 - in.achillesPermille) / 1000
	}

	var comboBarrierReduce int32
	if in.comboBarrierPermille > 0 {
		comboBarrierReduce = (raw - achillesReduce) * (1000 - in.comboBarrierPermille) / 1000
	}

	var magicGuardPortion, mpLoss, absorbed int32
	if in.magicGuardPct > 0 {
		magicGuardPortion = raw * in.magicGuardPct / 100
		mpLoss = magicGuardPortion
		if mpLoss > int32(in.currentMP) {
			mpLoss = int32(in.currentMP)
		}
		absorbed = mpLoss
		if in.infinity {
			absorbed = magicGuardPortion
			mpLoss = 0
		}
	}

	var magicShieldReduce int32
	if in.magicShieldPct > 0 {
		base := raw
		if in.magicShieldOnReducedDamage {
			base = raw - magicGuardPortion
		}
		magicShieldReduce = base * in.magicShieldPct / 100
	}

	var mesoGuarded, mesoCost int32
	if in.mesoGuardPct > 0 && in.mobSourced {
		mesoGuarded = raw / 2
		cost := int64(in.mesoGuardPct) * int64(mesoGuarded) / 100
		if cost > int64(in.meso) {
			// Partial guard: scale the guarded share down to what the
			// meso balance affords (CalcDamage::GetMesoGuardReduce).
			mesoGuarded = int32(int64(100) * int64(in.meso) / int64(in.mesoGuardPct))
			cost = int64(in.mesoGuardPct) * int64(mesoGuarded) / 100
		}
		mesoCost = int32(cost)
	}

	var pgReflect int32
	if in.powerGuardSignal && in.powerGuardPct > 0 && in.mobSourced {
		if mob.present && mob.alive {
			pgReflect = in.powerGuardPct * raw / 100
			divisor := in.pgCapDivisor
			if divisor <= 0 {
				divisor = 10
			}
			reflectCap := int32(mob.maxHp / uint32(divisor))
			if pgReflect > reflectCap {
				pgReflect = reflectCap
			}
			if mob.boss {
				pgReflect /= 2
			}
			if pgReflect > 0 && mob.fixedDamage > 0 {
				fixed := int32(mob.fixedDamage)
				if in.pgFixedDamageOverride || fixed < pgReflect {
					pgReflect = fixed
				}
			}
		}
	}

	hpLoss := raw - achillesReduce - comboBarrierReduce - magicShieldReduce - absorbed - mesoGuarded - pgReflect
	if hpLoss < 0 {
		hpLoss = 0
	}

	r.hpLoss = hpLoss
	r.mpLoss = mpLoss
	r.mesoCost = mesoCost
	r.breakdown = mitigationBreakdown{
		achillesReduce:     achillesReduce,
		comboBarrierReduce: comboBarrierReduce,
		magicShieldReduce:  magicShieldReduce,
		magicGuardAbsorbed: absorbed,
		mesoGuarded:        mesoGuarded,
		powerGuardReflect:  pgReflect,
	}
	if pgReflect > 0 {
		r.reflect = reflectIntent{amount: uint32(pgReflect), attackType: reflectAttackTypePhysical}
	}

	if in.manaReflectSignal && in.manaReflectPct > 0 && in.mobSourced && mob.present && mob.alive {
		mr := raw * in.manaReflectPct / 100
		mrCap := int32(mob.maxHp / 20)
		if mr > mrCap {
			mr = mrCap
		}
		if mr > 0 {
			// Mana Reflection does not reduce the caster's own damage.
			r.reflect = reflectIntent{amount: uint32(mr), attackType: reflectAttackTypeMagic}
		}
	}

	return r
}
```

- [ ] **Step 2: Write the tests**

`character_damage_mitigation_test.go` (table-driven; values grounded in v83 Skill.wz — see plan header):

```go
package handler

import (
	"math"
	"testing"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func mobUp(maxHp uint32) mobInfo {
	return mobInfo{present: true, alive: true, maxHp: maxHp}
}

func TestComputeMitigationNoBuffPassthrough(t *testing.T) {
	in := mitigationInput{attackIdx: packetmodel.DamageTypePhysical, rawDamage: 500, mobSourced: true, pgCapDivisor: 10}
	r := computeMitigation(in, mobUp(1000))
	if r.hpLoss != 500 || r.mpLoss != 0 || r.mesoCost != 0 || r.reflect.amount != 0 {
		t.Fatalf("passthrough broken: %+v", r)
	}
}

func TestComputeMitigationMagicGuard(t *testing.T) {
	// Magic Guard lvl 20: x=80 (v83 Skill.wz 2001002).
	base := mitigationInput{attackIdx: packetmodel.DamageTypePhysical, rawDamage: 1000, mobSourced: true, magicGuardPct: 80, pgCapDivisor: 10}

	t.Run("standard split", func(t *testing.T) {
		in := base
		in.currentMP = 5000
		r := computeMitigation(in, mobUp(1000))
		if r.mpLoss != 800 || r.hpLoss != 200 {
			t.Fatalf("mp=%d hp=%d, want 800/200", r.mpLoss, r.hpLoss)
		}
	})
	t.Run("MP shortfall spills to HP", func(t *testing.T) {
		in := base
		in.currentMP = 300
		r := computeMitigation(in, mobUp(1000))
		if r.mpLoss != 300 || r.hpLoss != 700 {
			t.Fatalf("mp=%d hp=%d, want 300/700", r.mpLoss, r.hpLoss)
		}
	})
	t.Run("Infinity absorbs fully with no MP cost", func(t *testing.T) {
		in := base
		in.currentMP = 0
		in.infinity = true
		r := computeMitigation(in, mobUp(1000))
		if r.mpLoss != 0 || r.hpLoss != 200 {
			t.Fatalf("mp=%d hp=%d, want 0/200", r.mpLoss, r.hpLoss)
		}
	})
}

func TestComputeMitigationMesoGuard(t *testing.T) {
	// Meso Guard: x is the meso cost rate (v83 4211005: 81-90).
	base := mitigationInput{attackIdx: packetmodel.DamageTypePhysical, rawDamage: 1000, mobSourced: true, mesoGuardPct: 81, pgCapDivisor: 10}

	t.Run("standard half guard", func(t *testing.T) {
		in := base
		in.meso = 1000000
		r := computeMitigation(in, mobUp(1000))
		// guarded = 500, cost = 81*500/100 = 405
		if r.hpLoss != 500 || r.mesoCost != 405 {
			t.Fatalf("hp=%d cost=%d, want 500/405", r.hpLoss, r.mesoCost)
		}
	})
	t.Run("partial guard when meso short", func(t *testing.T) {
		in := base
		in.meso = 100
		r := computeMitigation(in, mobUp(1000))
		// guarded = 100*100/81 = 123, cost = 81*123/100 = 99
		if r.breakdown.mesoGuarded != 123 || r.mesoCost != 99 || r.hpLoss != 877 {
			t.Fatalf("guarded=%d cost=%d hp=%d, want 123/99/877", r.breakdown.mesoGuarded, r.mesoCost, r.hpLoss)
		}
	})
	t.Run("zero meso guards nothing", func(t *testing.T) {
		in := base
		in.meso = 0
		r := computeMitigation(in, mobUp(1000))
		if r.mesoCost != 0 || r.hpLoss != 1000 {
			t.Fatalf("cost=%d hp=%d, want 0/1000", r.mesoCost, r.hpLoss)
		}
	})
	t.Run("not applied to obstacle damage", func(t *testing.T) {
		in := base
		in.meso = 1000000
		in.mobSourced = false
		in.attackIdx = packetmodel.DamageTypeObstacle
		r := computeMitigation(in, mobInfo{})
		if r.mesoCost != 0 || r.hpLoss != 1000 {
			t.Fatalf("cost=%d hp=%d, want 0/1000", r.mesoCost, r.hpLoss)
		}
	})
}

func TestComputeMitigationPowerGuard(t *testing.T) {
	base := mitigationInput{
		attackIdx: packetmodel.DamageTypePhysical, rawDamage: 1000, mobSourced: true,
		powerGuardSignal: true, powerGuardPct: 30, pgCapDivisor: 10,
	}

	t.Run("reflect reduces own HP loss", func(t *testing.T) {
		r := computeMitigation(base, mobUp(100000))
		// reflect = 30*1000/100 = 300, cap = 100000/10 = 10000
		if r.reflect.amount != 300 || r.reflect.attackType != reflectAttackTypePhysical || r.hpLoss != 700 {
			t.Fatalf("reflect=%+v hp=%d, want 300/physical/700", r.reflect, r.hpLoss)
		}
	})
	t.Run("cap binds at maxHp/divisor", func(t *testing.T) {
		r := computeMitigation(base, mobUp(1000))
		// cap = 1000/10 = 100
		if r.reflect.amount != 100 || r.hpLoss != 900 {
			t.Fatalf("reflect=%d hp=%d, want 100/900", r.reflect.amount, r.hpLoss)
		}
	})
	t.Run("v95 divisor 2", func(t *testing.T) {
		in := base
		in.pgCapDivisor = 2
		r := computeMitigation(in, mobUp(1000))
		// cap = 1000/2 = 500, reflect = 300 uncapped
		if r.reflect.amount != 300 {
			t.Fatalf("reflect=%d, want 300", r.reflect.amount)
		}
	})
	t.Run("boss halves after cap", func(t *testing.T) {
		mob := mobUp(1000)
		mob.boss = true
		r := computeMitigation(base, mob)
		if r.reflect.amount != 50 {
			t.Fatalf("reflect=%d, want 50", r.reflect.amount)
		}
	})
	t.Run("fixedDamage min pre-BB", func(t *testing.T) {
		mob := mobUp(100000)
		mob.fixedDamage = 5
		r := computeMitigation(base, mob)
		if r.reflect.amount != 5 {
			t.Fatalf("reflect=%d, want 5", r.reflect.amount)
		}
	})
	t.Run("fixedDamage override post-BB", func(t *testing.T) {
		in := base
		in.pgFixedDamageOverride = true
		mob := mobUp(100000)
		mob.fixedDamage = 100000
		r := computeMitigation(in, mob)
		if r.reflect.amount != 100000 {
			t.Fatalf("reflect=%d, want 100000 (override, not min)", r.reflect.amount)
		}
	})
	t.Run("dead mob drops reflect but keeps HP application", func(t *testing.T) {
		mob := mobUp(1000)
		mob.alive = false
		r := computeMitigation(base, mob)
		if r.reflect.amount != 0 || r.hpLoss != 1000 {
			t.Fatalf("reflect=%d hp=%d, want 0/1000", r.reflect.amount, r.hpLoss)
		}
	})
	t.Run("no signal means no reflect regardless of buff", func(t *testing.T) {
		in := base
		in.powerGuardSignal = false
		r := computeMitigation(in, mobUp(100000))
		if r.reflect.amount != 0 || r.hpLoss != 1000 {
			t.Fatalf("reflect=%d hp=%d, want 0/1000", r.reflect.amount, r.hpLoss)
		}
	})
}

func TestComputeMitigationManaReflection(t *testing.T) {
	in := mitigationInput{
		attackIdx: packetmodel.DamageTypeMagic, rawDamage: 1000, mobSourced: true,
		manaReflectSignal: true, manaReflectPct: 140, pgCapDivisor: 10,
	}
	t.Run("reflects without reducing own damage", func(t *testing.T) {
		r := computeMitigation(in, mobUp(100000))
		// reflect = 140*1000/100 = 1400, cap = 100000/20 = 5000
		if r.reflect.amount != 1400 || r.reflect.attackType != reflectAttackTypeMagic {
			t.Fatalf("reflect=%+v, want 1400/magic", r.reflect)
		}
		if r.hpLoss != 1000 {
			t.Fatalf("hp=%d, want 1000 (MR must not self-reduce)", r.hpLoss)
		}
	})
	t.Run("cap at maxHp/20", func(t *testing.T) {
		r := computeMitigation(in, mobUp(10000))
		if r.reflect.amount != 500 {
			t.Fatalf("reflect=%d, want 500", r.reflect.amount)
		}
	})
}

func TestComputeMitigationPassivesAndBarriers(t *testing.T) {
	t.Run("Achilles level 30", func(t *testing.T) {
		// x=850 per-mille -> reduce = 1000*(1000-850)/1000 = 150
		in := mitigationInput{attackIdx: packetmodel.DamageTypePhysical, rawDamage: 1000, mobSourced: true, achillesPermille: 850, pgCapDivisor: 10}
		r := computeMitigation(in, mobUp(1000))
		if r.breakdown.achillesReduce != 150 || r.hpLoss != 850 {
			t.Fatalf("reduce=%d hp=%d, want 150/850", r.breakdown.achillesReduce, r.hpLoss)
		}
	})
	t.Run("Achilles applies to obstacle damage", func(t *testing.T) {
		in := mitigationInput{attackIdx: packetmodel.DamageTypeObstacle, rawDamage: 1000, mobSourced: false, achillesPermille: 850}
		r := computeMitigation(in, mobInfo{})
		if r.hpLoss != 850 {
			t.Fatalf("hp=%d, want 850", r.hpLoss)
		}
	})
	t.Run("Combo Barrier stacks after Achilles", func(t *testing.T) {
		// cb x=864: reduce2 = (1000-150)*(1000-864)/1000 = 115
		in := mitigationInput{attackIdx: packetmodel.DamageTypePhysical, rawDamage: 1000, mobSourced: true, achillesPermille: 850, comboBarrierPermille: 864, pgCapDivisor: 10}
		r := computeMitigation(in, mobUp(1000))
		if r.breakdown.comboBarrierReduce != 115 || r.hpLoss != 735 {
			t.Fatalf("cb=%d hp=%d, want 115/735", r.breakdown.comboBarrierReduce, r.hpLoss)
		}
	})
	t.Run("Magic Shield v83 form uses raw damage", func(t *testing.T) {
		in := mitigationInput{attackIdx: packetmodel.DamageTypePhysical, rawDamage: 1000, mobSourced: true, magicGuardPct: 80, currentMP: 5000, magicShieldPct: 20, pgCapDivisor: 10}
		r := computeMitigation(in, mobUp(1000))
		// ms = 1000*20/100 = 200; hp = 1000 - 200 - 800 = 0
		if r.breakdown.magicShieldReduce != 200 || r.hpLoss != 0 {
			t.Fatalf("ms=%d hp=%d, want 200/0", r.breakdown.magicShieldReduce, r.hpLoss)
		}
	})
	t.Run("Magic Shield >=87 form uses damage minus MG portion", func(t *testing.T) {
		in := mitigationInput{attackIdx: packetmodel.DamageTypePhysical, rawDamage: 1000, mobSourced: true, magicGuardPct: 80, currentMP: 5000, magicShieldPct: 20, magicShieldOnReducedDamage: true, pgCapDivisor: 10}
		r := computeMitigation(in, mobUp(1000))
		// ms = (1000-800)*20/100 = 40; hp = 1000 - 40 - 800 = 160
		if r.breakdown.magicShieldReduce != 40 || r.hpLoss != 160 {
			t.Fatalf("ms=%d hp=%d, want 40/160", r.breakdown.magicShieldReduce, r.hpLoss)
		}
	})
}

func TestClampDamage(t *testing.T) {
	if v, adj := clampDamage(500); v != 500 || adj {
		t.Fatalf("got %d/%t", v, adj)
	}
	if v, adj := clampDamage(-5); v != 0 || !adj {
		t.Fatalf("forged negative: got %d/%t, want 0/true", v, adj)
	}
	if v, adj := clampDamage(50000000); v != maxLegitimateDamage || !adj {
		t.Fatalf("forged oversized: got %d/%t, want %d/true", v, adj, maxLegitimateDamage)
	}
}

func TestClampInt16(t *testing.T) {
	if clampInt16(40000) != math.MaxInt16 {
		t.Fatal("high clamp failed")
	}
	if clampInt16(-40000) != math.MinInt16 {
		t.Fatal("low clamp failed")
	}
	if clampInt16(-500) != -500 {
		t.Fatal("identity failed")
	}
}
```

- [ ] **Step 3: Run tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/ -run 'TestComputeMitigation|TestClamp' -v`
Expected: PASS. If any arithmetic assertion fails, re-derive the expected value by hand from the formula in the implementation — the formulas are IDA-verified, the test expectations must match them, not vice versa.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_damage_mitigation.go services/atlas-channel/atlas.com/channel/socket/handler/character_damage_mitigation_test.go
git commit -m "feat(atlas-channel): pure damage-taken mitigation chain (IDA-verified formulas)"
```

---

### Task 7: Handler orchestrator — wire the pipeline

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go` (full rewrite below)
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_test.go`
- Delete: `services/atlas-channel/atlas.com/channel/socket/model/damage_taken_info.go` (dead code — verified: only `packetmodel.NewDamageTakenInfo` is referenced)

**Interfaces:**
- Consumes: Task 1 getters, Task 3 `monsterdata.NewProcessor(l, ctx).GetById`, Task 5 `RequestChangeMeso`, Task 6 `computeMitigation`/`clampDamage`/`clampInt16`, `buff.NewProcessor(l, ctx).GetByCharacterId`, `skill2.GetLevel` (`atlas-channel/character/skill`), `dataskill.NewProcessor(l, ctx).GetEffect`, `monster.NewProcessor(l, ctx).GetById` (live monster), `charconst.TemporaryStatType*` constants (`libs/atlas-constants/character/temporary_stat.go`), `job.HeroId/PaladinId/DarkKnightId/AranStage4Id`, `skillconst.HeroAchillesId/PaladinAchillesId/DarkKnightAchillesId/AranStage4HighDefenseId` (`libs/atlas-constants/skill`).
- Produces: `processDamageTaken(l, t, f, p, c, deps)` — package-internal, exercised by tests via fake deps.

- [ ] **Step 1: Rewrite the handler**

`character_damage.go` (complete file):

```go
package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	skill2 "atlas-channel/character/skill"
	dataskill "atlas-channel/data/skill"
	"atlas-channel/data/skill/effect"
	monsterdata "atlas-channel/data/monster"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// damageMitigationDeps injects every lookup and side effect the
// damage-taken pipeline performs, mirroring damageInfoEntryDeps on the
// attack path, so tests drive the pipeline with fakes.
type damageMitigationDeps struct {
	getBuffs           func(characterId uint32) ([]buff.Model, error)
	getSkills          func(characterId uint32) ([]skill2.Model, error)
	getEffect          func(skillId uint32, level byte) (effect.Model, error)
	getMonster         func(monsterId uint32) (monster.Model, error)
	getMonsterTemplate func(templateId uint32) (monsterdata.Model, error)
	changeHP           func(f field.Model, characterId uint32, amount int16) error
	changeMP           func(f field.Model, characterId uint32, amount int16) error
	requestChangeMeso  func(f field.Model, characterId uint32, actorId uint32, actorType string, amount int32) error
	damageMonster      func(f field.Model, monsterId uint32, characterId uint32, damages []uint32, attackType byte) error
}

func CharacterDamageHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := packetmodel.NewDamageTakenInfo(s.CharacterId())
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		// TODO decrease battleship hp

		c, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			return
		}

		// Foreign-session announce always fires with the client-reported
		// event and is never blocked on mitigation (FR-2.5).
		err = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), session.Announce(l)(ctx)(wp)(charpkt.CharacterDamageWriter)(charpkt.NewCharacterDamage(c.Id(), p.AttackIdx(), p.Damage(), p.MonsterTemplateId(), p.Left()).Encode))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce character [%d] has been damaged to foreign characters in map [%d].", s.CharacterId(), s.MapId())
		}

		t := tenant.MustFromContext(ctx)
		cp := character.NewProcessor(l, ctx)
		mp := monster.NewProcessor(l, ctx)
		deps := damageMitigationDeps{
			getBuffs:           buff.NewProcessor(l, ctx).GetByCharacterId,
			getSkills:          skill2.NewProcessor(l, ctx).GetByCharacterId,
			getEffect:          dataskill.NewProcessor(l, ctx).GetEffect,
			getMonster:         mp.GetById,
			getMonsterTemplate: monsterdata.NewProcessor(l, ctx).GetById,
			changeHP:           cp.ChangeHP,
			changeMP:           cp.ChangeMP,
			requestChangeMeso:  cp.RequestChangeMeso,
			damageMonster:      mp.Damage,
		}
		processDamageTaken(l, t, s.Field(), *p, c, deps)
	}
}

// achillesSkillIdForJob selects the flat-reduction passive by job: the
// client's GetAchillesReduce picks Achilles for jobs 112/122/132 and High
// Defense for Aran (2112), same formula (IDA-verified, all versions).
func achillesSkillIdForJob(jobId job.Id) (skillconst.Id, bool) {
	switch jobId {
	case job.HeroId:
		return skillconst.HeroAchillesId, true
	case job.PaladinId:
		return skillconst.PaladinAchillesId, true
	case job.DarkKnightId:
		return skillconst.DarkKnightAchillesId, true
	case job.AranStage4Id:
		return skillconst.AranStage4HighDefenseId, true
	}
	return 0, false
}

// buffAmounts extracts the roster statup values from active buffs.
type buffAmounts struct {
	magicGuard   int32
	infinity     bool
	powerGuard   int32
	mesoGuard    int32
	manaReflect  bool
	manaReflectSourceId int32
	manaReflectLevel    byte
	comboBarrier int32
	magicShield  int32
	guard        bool
}

func extractBuffAmounts(buffs []buff.Model) buffAmounts {
	var a buffAmounts
	for _, b := range buffs {
		if b.Expired() {
			continue
		}
		for _, ch := range b.Changes() {
			switch ch.Type() {
			case string(charconst.TemporaryStatTypeMagicGuard):
				a.magicGuard = ch.Amount()
			case string(charconst.TemporaryStatTypeInfinity):
				a.infinity = true
			case string(charconst.TemporaryStatTypePowerGuard):
				a.powerGuard = ch.Amount()
			case string(charconst.TemporaryStatTypeMesoGuard):
				a.mesoGuard = ch.Amount()
			case string(charconst.TemporaryStatTypeManaReflection):
				a.manaReflect = true
				a.manaReflectSourceId = b.SourceId()
				a.manaReflectLevel = b.Level()
			case string(charconst.TemporaryStatTypeComboBarrier):
				a.comboBarrier = ch.Amount()
			case string(charconst.TemporaryStatTypeMagicShield):
				a.magicShield = ch.Amount()
			case string(charconst.TemporaryStatTypeGuard):
				a.guard = true
			}
		}
	}
	return a
}

// processDamageTaken applies the server-authoritative mitigation chain to
// one damage-taken event and emits the resulting deltas. The client's
// damage value is raw pre-mitigation input (IDA-verified); reflect
// amounts are always server-computed (FR-10.3).
func processDamageTaken(
	l logrus.FieldLogger,
	t tenant.Model,
	f field.Model,
	p packetmodel.DamageTakenInfo,
	c character.Model,
	deps damageMitigationDeps,
) {
	characterId := c.Id()

	// Block sentinel: the client sends damage == -1 for a fully blocked
	// hit (Guardian, Fake/Shadow Shifter, GUARD, v95 Mechanic Perfect
	// Armor) and applies zero HP loss. The old handler applied +1 HP.
	if p.Damage() == -1 {
		plausible := p.Guard() ||
			c.JobId() == job.HeroId || c.JobId() == job.PaladinId ||
			c.JobId() == job.Id(412) || c.JobId() == job.Id(422)
		if !plausible {
			if buffs, err := deps.getBuffs(characterId); err == nil {
				plausible = extractBuffAmounts(buffs).guard
			}
		}
		if !plausible {
			l.Warnf("Character [%d] in map [%d] sent a block sentinel with no plausible block source (job [%d], mob template [%d]). Ignoring damage.", characterId, f.MapId(), c.JobId(), p.MonsterTemplateId())
		}
		return
	}

	raw, adjusted := clampDamage(p.Damage())
	if adjusted {
		l.Warnf("Character [%d] in map [%d] sent out-of-bounds damage [%d] (mob template [%d], attackIdx [%d]). Clamped to [%d].", characterId, f.MapId(), p.Damage(), p.MonsterTemplateId(), p.AttackIdx(), raw)
	}

	mobSourced := p.AttackIdx() >= packetmodel.DamageTypePhysical

	var a buffAmounts
	buffs, err := deps.getBuffs(characterId)
	if err != nil {
		// Buff lookup failure must not leave the hit unapplied: fall back
		// to the unmitigated path (FR-2.4 behavior).
		l.WithError(err).Warnf("Unable to look up buffs for character [%d]; applying unmitigated damage.", characterId)
	} else {
		a = extractBuffAmounts(buffs)
	}

	// Cross-check the client's Power Guard claim against server state
	// (FR-5.4): amounts are never taken from the wire.
	powerGuardSignal := false
	if p.HasReflectExtension() && p.IsPowerGuard() && mobSourced {
		if a.powerGuard <= 0 {
			l.Warnf("Character [%d] claimed Power Guard without an active POWER_GUARD buff (mob [%d], map [%d]). Ignoring claim.", characterId, p.MonsterId(), f.MapId())
		} else if p.ReflectTargetMobId() != p.MonsterId() {
			l.Warnf("Character [%d] Power Guard reflect target [%d] is not the attacking mob [%d] (map [%d]). Ignoring claim.", characterId, p.ReflectTargetMobId(), p.MonsterId(), f.MapId())
		} else {
			powerGuardSignal = true
		}
	}

	// Mana Reflection: the client rolls prop and signals the outcome via
	// a reflect echo without isPowerGuard on a mob skill attack
	// (attackIdx >= 0). Honor the validated signal, recompute the amount.
	manaReflectSignal := false
	var manaReflectPct int32
	if p.HasReflectExtension() && !p.IsPowerGuard() && p.Reflect() > 0 && p.AttackIdx() >= packetmodel.DamageTypeMagic {
		if !a.manaReflect {
			l.Warnf("Character [%d] signaled Mana Reflection without an active MANA_REFLECTION buff (mob [%d], map [%d]). Ignoring claim.", characterId, p.MonsterId(), f.MapId())
		} else {
			eff, effErr := deps.getEffect(uint32(a.manaReflectSourceId), a.manaReflectLevel)
			if effErr != nil {
				l.WithError(effErr).Warnf("Unable to load Mana Reflection effect [%d] level [%d] for character [%d]. Dropping reflect.", a.manaReflectSourceId, a.manaReflectLevel, characterId)
			} else {
				manaReflectSignal = true
				manaReflectPct = int32(eff.X())
			}
		}
	}

	// Warrior/Aran flat-reduction passive: only fetch skills for the jobs
	// that have one (design §5 step 3).
	var achillesPermille int32
	if skillId, ok := achillesSkillIdForJob(c.JobId()); ok {
		skills, sErr := deps.getSkills(characterId)
		if sErr != nil {
			l.WithError(sErr).Warnf("Unable to look up skills for character [%d]; skipping passive reduction.", characterId)
		} else if level := skill2.GetLevel(skills, skillId); level > 0 {
			eff, effErr := deps.getEffect(uint32(skillId), level)
			if effErr != nil {
				l.WithError(effErr).Warnf("Unable to load passive effect [%d] level [%d] for character [%d]; skipping passive reduction.", skillId, level, characterId)
			} else {
				achillesPermille = int32(eff.X())
			}
		}
	}

	// Mob data is only needed when a reflect will actually be computed.
	var mob mobInfo
	if (powerGuardSignal && a.powerGuard > 0) || manaReflectSignal {
		live, mErr := deps.getMonster(p.MonsterId())
		if mErr != nil {
			l.WithError(mErr).Debugf("Reflect target mob [%d] not found for character [%d]; dropping reflect, keeping mitigation.", p.MonsterId(), characterId)
		} else {
			mob.present = true
			mob.alive = live.Hp() > 0
			mob.maxHp = live.MaxHp()
			tmpl, tErr := deps.getMonsterTemplate(p.MonsterTemplateId())
			if tErr != nil {
				l.WithError(tErr).Debugf("Monster template [%d] not found; boss/fixedDamage caps default to non-boss/none.", p.MonsterTemplateId())
			} else {
				mob.boss = tmpl.Boss()
				mob.fixedDamage = tmpl.FixedDamage()
			}
		}
	}

	pgCapDivisor := int32(10)
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		pgCapDivisor = 2
	}
	in := mitigationInput{
		attackIdx:                  p.AttackIdx(),
		rawDamage:                  raw,
		mobSourced:                 mobSourced,
		powerGuardSignal:           powerGuardSignal,
		manaReflectSignal:          manaReflectSignal,
		currentMP:                  c.Mp(),
		meso:                       c.Meso(),
		magicGuardPct:              a.magicGuard,
		infinity:                   a.infinity,
		powerGuardPct:              a.powerGuard,
		mesoGuardPct:               a.mesoGuard,
		comboBarrierPermille:       a.comboBarrier,
		magicShieldPct:             a.magicShield,
		achillesPermille:           achillesPermille,
		manaReflectPct:             manaReflectPct,
		magicShieldOnReducedDamage: t.MajorVersion() >= 87,
		pgCapDivisor:               pgCapDivisor,
		pgFixedDamageOverride:      (t.Region() == "GMS" && t.MajorVersion() >= 95) || t.Region() == "JMS",
	}

	result := computeMitigation(in, mob)
	l.Debugf("Character [%d] damage [%d] mitigated to hp [%d] mp [%d] meso [%d] reflect [%d] (achilles [%d], comboBarrier [%d], magicShield [%d], magicGuard [%d], mesoGuard [%d], powerGuard [%d]).",
		characterId, raw, result.hpLoss, result.mpLoss, result.mesoCost, result.reflect.amount,
		result.breakdown.achillesReduce, result.breakdown.comboBarrierReduce, result.breakdown.magicShieldReduce,
		result.breakdown.magicGuardAbsorbed, result.breakdown.mesoGuarded, result.breakdown.powerGuardReflect)

	_ = deps.changeHP(f, characterId, -clampInt16(result.hpLoss))
	if result.mpLoss > 0 {
		_ = deps.changeMP(f, characterId, -clampInt16(result.mpLoss))
	}
	if result.mesoCost > 0 {
		_ = deps.requestChangeMeso(f, characterId, characterId, "SKILL", -result.mesoCost)
	}
	if result.reflect.amount > 0 {
		_ = deps.damageMonster(f, p.MonsterId(), characterId, []uint32{result.reflect.amount}, result.reflect.attackType)
	}
}
```

Then delete the dead file:

```bash
git rm services/atlas-channel/atlas.com/channel/socket/model/damage_taken_info.go
```

Note on `message.Buffer` (design §5): the Meso Guard invariant (FR-6.3) is satisfied by deciding `mesoCost` and `hpLoss` together in `computeMitigation` from the server-side balance, then emitting both in the same invocation — matching the attack-path precedent, which also emits via processor methods rather than a buffer.

- [ ] **Step 2: Write the handler tests**

`character_damage_test.go` (package `handler`; reuses `buildSkillModel` from `character_skill_prepare_test.go` — same package):

```go
package handler

import (
	"testing"
	"time"

	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	skill2 "atlas-channel/character/skill"
	monsterdata "atlas-channel/data/monster"
	"atlas-channel/data/skill/effect"
	"atlas-channel/monster"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

type emissions struct {
	hp       []int16
	mp       []int16
	meso     []int32
	reflects []uint32
}

func fakeDeps(em *emissions, buffs []buff.Model, skills []skill2.Model, eff effect.Model, mob monster.Model, tmpl monsterdata.Model) damageMitigationDeps {
	return damageMitigationDeps{
		getBuffs:  func(uint32) ([]buff.Model, error) { return buffs, nil },
		getSkills: func(uint32) ([]skill2.Model, error) { return skills, nil },
		getEffect: func(uint32, byte) (effect.Model, error) { return eff, nil },
		getMonster: func(uint32) (monster.Model, error) { return mob, nil },
		getMonsterTemplate: func(uint32) (monsterdata.Model, error) { return tmpl, nil },
		changeHP: func(_ field.Model, _ uint32, amount int16) error {
			em.hp = append(em.hp, amount)
			return nil
		},
		changeMP: func(_ field.Model, _ uint32, amount int16) error {
			em.mp = append(em.mp, amount)
			return nil
		},
		requestChangeMeso: func(_ field.Model, _ uint32, _ uint32, _ string, amount int32) error {
			em.meso = append(em.meso, amount)
			return nil
		},
		damageMonster: func(_ field.Model, _ uint32, _ uint32, damages []uint32, _ byte) error {
			em.reflects = append(em.reflects, damages...)
			return nil
		},
	}
}

func activeBuff(statType charconst.TemporaryStatType, amount int32) buff.Model {
	future := time.Now().Add(time.Hour)
	return buff.NewBuff(2001002, 20, 3600, []stat.Model{stat.NewStat(string(statType), amount)}, time.Now(), future)
}

func testTenantModel(t *testing.T, region string, major uint16) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), region, major, 1)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func testCharacter(t *testing.T, jobId job.Id, mp uint16, meso uint32, skills []skill2.Model) character.Model {
	t.Helper()
	c, err := character.NewModelBuilder().
		SetId(42).
		SetJobId(jobId).
		SetHp(1000).SetMaxHp(2000).
		SetMp(mp).SetMaxMp(2000).
		SetMeso(meso).
		SetSkills(skills).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// damagePacket builds a decoded-equivalent DamageTakenInfo via the codec:
// struct fields are package-private to packetmodel, so encode+decode through
// the real codec is the construction path.
func damagePacket(t *testing.T, tm tenant.Model, attackIdx packetmodel.DamageType, damage int32, withPGExt bool) packetmodel.DamageTakenInfo {
	t.Helper()
	// Encode with a source model configured via the test hook below.
	return decodeDamagePacket(t, tm, attackIdx, damage, withPGExt)
}

func testField() field.Model {
	return field.NewBuilder(2, 1, 100000000).Build()
}

func TestProcessDamageTakenNoBuffAppliesFullDamage(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 500, false)
	c := testCharacter(t, job.Id(100), 100, 0, nil)
	processDamageTaken(l, tm, testField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -500 {
		t.Fatalf("hp emissions=%v, want [-500]", em.hp)
	}
	if len(em.mp)+len(em.meso)+len(em.reflects) != 0 {
		t.Fatalf("unexpected side effects: %+v", em)
	}
}

func TestProcessDamageTakenMagicGuardEmitsHPAndMP(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	buffs := []buff.Model{activeBuff(charconst.TemporaryStatTypeMagicGuard, 80)}
	deps := fakeDeps(em, buffs, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, false)
	c := testCharacter(t, job.Id(200), 2000, 0, nil)
	processDamageTaken(l, tm, testField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -200 {
		t.Fatalf("hp=%v, want [-200]", em.hp)
	}
	if len(em.mp) != 1 || em.mp[0] != -800 {
		t.Fatalf("mp=%v, want [-800]", em.mp)
	}
}

func TestProcessDamageTakenMesoGuardEmitsMesoDeduction(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	buffs := []buff.Model{activeBuff(charconst.TemporaryStatTypeMesoGuard, 81)}
	deps := fakeDeps(em, buffs, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, false)
	c := testCharacter(t, job.Id(422), 100, 1000000, nil)
	processDamageTaken(l, tm, testField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -500 {
		t.Fatalf("hp=%v, want [-500]", em.hp)
	}
	if len(em.meso) != 1 || em.meso[0] != -405 {
		t.Fatalf("meso=%v, want [-405]", em.meso)
	}
}

func TestProcessDamageTakenPowerGuardReflects(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	buffs := []buff.Model{activeBuff(charconst.TemporaryStatTypePowerGuard, 30)}
	mob, err := monster.NewModelBuilder(42, testField(), 200100).SetHp(50000).SetMaxHp(100000).Build()
	if err != nil {
		t.Fatal(err)
	}
	deps := fakeDeps(em, buffs, nil, effect.Model{}, mob, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, true)
	c := testCharacter(t, job.Id(110), 100, 0, nil)
	processDamageTaken(l, tm, testField(), p, c, deps)

	if len(em.reflects) != 1 || em.reflects[0] != 300 {
		t.Fatalf("reflects=%v, want [300]", em.reflects)
	}
	if len(em.hp) != 1 || em.hp[0] != -700 {
		t.Fatalf("hp=%v, want [-700]", em.hp)
	}
}

// Forged claim: isPowerGuard set on the wire but no POWER_GUARD buff —
// the claim is ignored, full damage applies, nothing reflects.
func TestProcessDamageTakenForgedPowerGuardIgnored(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, true)
	c := testCharacter(t, job.Id(110), 100, 0, nil)
	processDamageTaken(l, tm, testField(), p, c, deps)

	if len(em.reflects) != 0 {
		t.Fatalf("reflects=%v, want none", em.reflects)
	}
	if len(em.hp) != 1 || em.hp[0] != -1000 {
		t.Fatalf("hp=%v, want [-1000]", em.hp)
	}
}

// Forged oversized damage is clamped, and the int16 conversion is bounded.
func TestProcessDamageTakenForgedDamageClamped(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 50000000, false)
	c := testCharacter(t, job.Id(100), 100, 0, nil)
	processDamageTaken(l, tm, testField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -32767 {
		t.Fatalf("hp=%v, want [-32767] (clamped int16)", em.hp)
	}
}

// Block sentinel (-1) must not touch HP; the old handler healed +1.
func TestProcessDamageTakenSentinelNoHPChange(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, -1, false)
	c := testCharacter(t, job.HeroId, 100, 0, nil)
	processDamageTaken(l, tm, testField(), p, c, deps)

	if len(em.hp) != 0 {
		t.Fatalf("hp=%v, want none for block sentinel", em.hp)
	}
}

func TestProcessDamageTakenAchillesPassive(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	skills := []skill2.Model{buildSkillModel(t, skillconst.HeroAchillesId, 30)}
	eff, err := effect.Extract(effect.RestModel{X: 850})
	if err != nil {
		t.Fatal(err)
	}
	deps := fakeDeps(em, nil, skills, eff, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, false)
	c := testCharacter(t, job.HeroId, 100, 0, skills)
	processDamageTaken(l, tm, testField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -850 {
		t.Fatalf("hp=%v, want [-850] (Achilles x=850)", em.hp)
	}
}
```

Support details for this test file: import `skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"` for `HeroAchillesId`. `decodeDamagePacket` — the packet struct is package-private to `packetmodel`, so tests construct it through the real codec (also re-exercising Task 1). Add this helper to `character_damage_test.go`:

```go
// decodeDamagePacket produces a DamageTakenInfo as the handler would see
// it, by round-tripping raw bytes through the real decoder.
func decodeDamagePacket(t *testing.T, tm tenant.Model, attackIdx packetmodel.DamageType, damage int32, withPGExt bool) packetmodel.DamageTakenInfo {
	t.Helper()
	ctx := tenant.WithContext(context.Background(), tm)
	l, _ := test.NewNullLogger()

	w := response.NewWriter(l)
	w.WriteInt(uint32(12345))
	w.WriteInt8(int8(attackIdx))
	w.WriteInt8(0)
	w.WriteInt32(damage)
	if attackIdx >= packetmodel.DamageTypePhysical {
		w.WriteInt(uint32(200100)) // mobTemplateId
		w.WriteInt(uint32(42))     // mobId
		w.WriteBool(true)          // left
		if withPGExt {
			w.WriteByte(30) // reflect echo
		} else {
			w.WriteByte(0)
		}
		if tm.Region() == "GMS" && tm.MajorVersion() >= 95 {
			w.WriteBool(false)
		}
		w.WriteByte(0) // blockByte
		if withPGExt {
			w.WriteBool(true)      // isPowerGuard
			w.WriteInt(uint32(42)) // reflectTargetMobId == attacking mob
			w.WriteByte(3)         // hitAction
			w.WriteInt16(100)
			w.WriteInt16(200)
			w.WriteInt16(110)
			w.WriteInt16(210)
		}
	} else {
		w.WriteInt16(0) // obstacleData
	}
	w.WriteByte(0) // stanceFlags

	req := request.Request(w.Bytes())
	reader := request.NewRequestReader(&req, 0)
	m := packetmodel.NewDamageTakenInfo(42)
	m.Decode(l, ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Fatalf("test packet under-consumed: %d bytes left", reader.Available())
	}
	return *m
}
```

(Add `"context"`, `"github.com/Chronicle20/atlas/libs/atlas-socket/response"`, and `"github.com/Chronicle20/atlas/libs/atlas-socket/request"` to the test imports. If `response.NewWriter` needs a `*logrus.Entry` rather than a test logger, follow the construction used in `libs/atlas-packet/test/roundtrip.go`.)

- [ ] **Step 3: Run the tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/ -run TestProcessDamageTaken -v`
Expected: PASS (8 tests).

- [ ] **Step 4: Full service check**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...`
Expected: clean. Verify TODO removal: `grep -n "TODO" services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go` must print exactly one line — `// TODO decrease battleship hp`.

- [ ] **Step 5: Wire `CharacterDamageHandle` into the gms_87/92/95/jms templates**

`main` wired this handler into gms_48/61/72/79/84, but gms_87/92/95/jms never routed it (pre-existing gap — verified absent at branch base too; the packet is decoded correctly by Task 1 but no template dispatches it there). The design verified v87/v95/jms behavior and Task 1 verified v92, so the codec is ready — these four just need the handler entry. Add to each template's `handlers` array, at the **sorted `opCode` position** (never appended — `tools/template-opcode-order-guard.sh` enforces strictly-ascending order), exactly this shape (matching the existing gms_83 entry):

```json
{
  "opCode": "<op>",
  "validator": "LoggedInValidator",
  "handler": "CharacterDamageHandle",
  "services": ["channel"]
}
```

Serverbound `TAKE_DAMAGE` opcodes (verified free of handler-opcode collision — they exist only as clientbound *writers*, a separate namespace):

| Template | opCode |
|---|---|
| `template_gms_87_1.json` | `0x32` |
| `template_gms_92_1.json` | `0x35` (from the v92 IDB — v92 is not in the packet registry) |
| `template_gms_95_1.json` | `0x34` |
| `template_jms_185_1.json` | `0x27` |

A missing `validator` key would cause the handler entry to be silently dropped, so `LoggedInValidator` is mandatory. Then run `tools/template-opcode-order-guard.sh` from the repo root — expected clean. (Live-tenant socket-config reconciliation for already-provisioned tenants is a deploy-time step, out of scope for this branch; the seed templates are the source of truth this task changes.)

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go services/atlas-channel/atlas.com/channel/socket/handler/character_damage_test.go
git add services/atlas-configurations/seed-data/templates/template_gms_87_1.json services/atlas-configurations/seed-data/templates/template_gms_92_1.json services/atlas-configurations/seed-data/templates/template_gms_95_1.json services/atlas-configurations/seed-data/templates/template_jms_185_1.json
git rm services/atlas-channel/atlas.com/channel/socket/model/damage_taken_info.go
git commit -m "feat(atlas-channel): server-authoritative damage-taken mitigation pipeline

Magic Guard, Power Guard, Meso Guard, Mana Reflection, Achilles, High
Defense, Combo Barrier, Magic Shield, block sentinel, anti-cheat clamps.
Removes 9 of the 10 TODOs (battleship = task-153). Wires CharacterDamageHandle
into the gms_87/92/95/jms templates (v48-v84 already wired on main)."
```

---

### Task 8: Verification sweep

**Files:** none new — verification only.

- [ ] **Step 1: Module test/vet/build sweep**

From the worktree root:

```bash
(cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-data/atlas.com/data && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...)
```

Expected: all clean. (atlas-data's full test suite is slow; run it once here even if Task 2 only ran the monster package.)

- [ ] **Step 2: Docker bakes (mandatory — catches Dockerfile COPY gaps that go.work hides)**

```bash
docker buildx bake atlas-channel
docker buildx bake atlas-data
```

Expected: both succeed. No new shared lib was added, so no Dockerfile edits should be needed; if a bake fails on a missing `COPY libs/...`, fix the root `Dockerfile` and re-bake.

- [ ] **Step 3: Repo-root guards (merged from `main`)**

Run each from the worktree root:
- `tools/redis-key-guard.sh` — clean (no raw keyed go-redis calls added).
- `tools/goroutine-guard.sh` — clean (no bare `go` statements added).
- `tools/template-opcode-order-guard.sh` — clean (**required** — Task 7 edited gms_87/92/95/jms templates; verifies the new `CharacterDamageHandle` entries sit at their strictly-ascending `opCode` position).
- `tools/lint.sh --check` — clean (shared gofumpt/goimports + golangci-lint v2 across every Go module + atlas-ui). If it flags formatting, run `tools/lint.sh` (fix mode) and re-commit the affected task's files.

- [ ] **Step 4: Acceptance checklist against the PRD**

Confirm each and record in the commit message if all pass:
- Verification matrix: design.md §3 (incl. the post-merge legacy/v92 extension) + this plan's findings §7–8 (committed).
- 9 mitigation TODOs removed; battleship TODO intact (`grep -c TODO .../character_damage.go` == 1).
- Anti-cheat tests: `TestProcessDamageTakenForgedPowerGuardIgnored`, `TestProcessDamageTakenForgedDamageClamped`, `TestClampDamage`, `TestClampInt16` green.
- Version gating tests: `TestComputeMitigationPowerGuard/v95 divisor 2`, `fixedDamage override post-BB`, `TestComputeMitigationPassivesAndBarriers/Magic Shield >=87 form` green; Divine Shield has no code path to gate (GUARD is data-driven and never granted pre-BB).
- Packet fixtures: all `TestDamageTakenInfo*` subtests green across `pt.Variants` (incl. the legacy v28/v48/v61/v72/v79 columns and the added v92), plus `TestDamageTakenInfoV48ByteFixtures` (the divergent v48 layout — no `nMagicElemAttr`, 10-byte extension, no non-mob stance byte).
- Handler wiring: `CharacterDamageHandle` present in all of gms_48/61/72/79/83/84/87/92/95/jms templates (grep each); template-opcode-order-guard clean.

- [ ] **Step 5: Commit (only if anything changed during verification) and hand off**

Implementation complete. Per CLAUDE.md, run `superpowers:requesting-code-review` (dispatches plan-adherence + backend reviewers) before opening a PR — this is invoked by the /execute-task phase wrap-up, not skipped.

---

## Self-review notes

- Spec coverage: FR-2.x (pipeline/classification/announce) → Tasks 6–7; FR-3.x (verification matrix) → design §3 + plan findings; FR-4/5/6/7/8/9 (per-skill) → Tasks 6–7 with per-skill tests; FR-10 (anti-cheat) → `clampDamage`/`clampInt16`/forged-claim tests; FR-11 (version gates) → resolved gates in `mitigationInput` + codec gate tests; §5 API (REQUEST_CHANGE_MESO) → Task 5; §7 service impact (atlas-data fixedDamage, data clients) → Tasks 2–4. Body Pressure ships no damage-handler code by design (§1 correction 1 — TODO removed with the others in Task 7's rewrite). Divine Shield constant intentionally not shipped (plan findings §4).
- Type consistency: `computeMitigation(mitigationInput, mobInfo) mitigationResult` used identically in Tasks 6/7; `RequestChangeMeso(f, characterId, actorId, actorType, amount)` identical in Tasks 5/7; packet getters listed in Task 1 match every Task 7 call site.
- No placeholders: every step has complete code or an exact command with expected output.
- Post-merge version-scope reconciliation (findings §7–8, design §2a/§3): full column set v48…jms verified against live per-version IDBs. Net code impact is bounded — one v48 decoder branch (Task 1) + four template handler entries (Task 7 Step 5); the mitigation-formula gates (Task 6) are verified unchanged, and per-version skill availability is auto-handled by the data-driven, server-authoritative pipeline. v92 verified independently (not inherited from v87). New merged guards (template-opcode-order, goroutine, lint.sh) folded into Global Constraints + Task 8 Step 3.
