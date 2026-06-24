# Summon Clientbound Packet-Fixture Campaign — Context

Companion to `plan.md`. Key files, decisions, gotchas, and the grounding gathered during planning. Read this once before starting; it is the "why" behind the plan's "what."

## What this task is (and is not)

- **Is:** a *verification* campaign. Promote 25 coverage-matrix cells (`docs/packets/audits/STATUS.md`) in the `summon` clientbound family from `incomplete`/`partial` to `verified` by adding `packet-audit:verify` byte-fixtures (and tier-1 evidence) derived field-by-field from the live client read order.
- **Is not:** a feature. No new summon gameplay; no serverbound handler changes (`SummonMoveHandle`/`SummonDamageHandle`/`SummonAttackHandle` are already verified on all five versions). A codec (`*.go`) changes **only** if a live decompile proves a byte error (§R3 in the plan) — expected not to happen, possible.

## The 25 cells

24 tier-1 (`v83/v84/v87/jms` × `{SummonSpawn, SummonRemove, SummonMove, SummonAttack, SummonDamage, SummonSkill}`) + 1 tier-0 (`v95 SummonMove`, the lone 🟡 in the whole matrix). v95 for the other five packets is already ✅ (out of scope).

## The single most important decision: two grading recipes

Confirmed against `tools/packet-audit/internal/matrix/grade.go` and `docs/packets/evidence/tiers.yaml` during planning:

- `summon/` is **NOT** listed in `tiers.yaml` (`packets: []`; `packet_prefixes` covers monster/pet/interaction/… but not summon). So the grader does not treat summon as a fixed tier.
- A summon cell's tier is therefore `tier1 = in.Tier1[pkt] || rep.FlatInvalid` (grade.go:117). `in.Tier1["summon/clientbound/*"]` is **false**, so the report's `FlatInvalid` flag decides.
- Confirmed report flags (dumped during planning):
  - **v95** (all six): `Verdict=0 (Match)`, `FlatInvalid=false` → **tier-0**. Verified rule `toolPass && marker.Found` (grade.go:215). v95 SummonMove just needs a marker — **NO evidence record** (`VERIFYING_A_PACKET.md` §7; a tier-0 evidence record is a standing freshness liability).
  - **v83/v84/v87/jms** (all six each): `Verdict=3 (🔍)`, `FlatInvalid=true` → **tier-1**. Verified rule `marker.Found && hasEvidence && evidence.Fresh` (grade.go:199). Each needs a marker **AND** a fresh pinned evidence record.

`FlatInvalid=true` means the static analyzer could not reduce a writer branch to a version predicate (`internal/report/report.go`), so the flat positional diff is capped to 🔍 — a *modeling limitation, not a wire bug*. A byte-fixture resolves it; tier-1 does not require `toolPass`.

This corrects the PRD's blanket "every promotion pins an evidence record": the lone v95 cell must **not**; the other 24 must.

## The landmine: active-vs-inactive dispatch (the v83 SummonSpawn re-point)

The committed v83 `SummonSpawn` report points at `CSummonedPool::OnCreated @0x938f61` — the **inactive** twin whose dispatcher does NOT pre-read `cid`. The live-confirmed **active** field-path target is `OnCreated @0x95ADEC` (task-088 x32dbg: at OnCreated's first `Decode4` the read offset is already past `cid`, so the int after `cid` is the `oid`). The codec now writes `oid` on all versions accordingly.

Consequence: where a cell's committed report/export address ≠ the live active read function, the export entry must be re-pointed (surgical splice, plan §R5) and the report regenerated so **marker + report + evidence all agree on the active address** — otherwise `matrix --check` raises an orphan-marker failure, OR (worse) you ratify the wrong byte layout = a false ✅. The dispatch chain is `CUserPool::OnUserCommonPacket` (reads `cid`) → `CSummonedPool::OnPacket @0x938dd7` (reads `oid`) → per-op leaf. Markers/evidence are keyed to the **leaf** fname (matching the v95 convention: `SummonSpawn version=gms_v95 ida=0x75a9a0` = the OnCreated leaf).

The existing `*_test.go` comments explicitly flag v84/v87/jms as inheriting this correction "by inference … but NOT re-confirmed live — the old `ida=` markers point at the wrong path." Hence Approach A (strict per-cell live re-verification), not a blind port.

## Naming-vs-address swap on Damage/Skill (don't get tripped)

The registry maps `DAMAGE_SUMMON → CSummonedPool::OnHit` and `SUMMON_SKILL → CSummonedPool::OnSkill`, and the reports follow (`SummonDamage` idaName=`OnHit`, `SummonSkill` idaName=`OnSkill`). The codec *comments* describe the bodies the other way around ("damage is read by OnSkill@0x7a6ebe", "skill by OnHit@0x7a6e5a") — the "swapped skill/damage opcodes" note. This is a human-naming quirk: the **address↔FName↔report linkage is internally consistent** (`OnHit`→`0x7a6ebe` in the export resolves the evidence address; marker uses the same). So you do **not** need to re-point for the naming — just confirm the body at the committed address matches the codec, and key the marker/evidence to the FName the report already uses. Only re-point where the live active address genuinely differs (the Spawn case).

## Key files

| Purpose | Path |
|---|---|
| Codecs (read; fix only on proven byte error) | `libs/atlas-packet/summon/clientbound/{spawn,remove,move,attack,damage,skill}.go` |
| Fixtures (add `…Bytes<VER>` + markers) | `libs/atlas-packet/summon/clientbound/*_test.go` |
| Test helpers | `libs/atlas-packet/test/context.go` (`Variants`, `CreateContext`), `roundtrip.go` (`Encode`, `RoundTrip`) |
| Reference fixture pattern | `libs/atlas-packet/party/clientbound/invite_test.go` |
| Production writer (wraps the codecs) | `services/atlas-channel/atlas.com/channel/socket/writer/summon.go` |
| Playbook | `docs/packets/audits/VERIFYING_A_PACKET.md` |
| Grader | `tools/packet-audit/internal/matrix/grade.go` |
| Tier config | `docs/packets/evidence/tiers.yaml` |
| Registries | `docs/packets/registry/{gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}.yaml` |
| Audit reports | `docs/packets/audits/{gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}/Summon*.json` |
| IDA exports | `docs/packets/ida-exports/{gms_v83,gms_v84,gms_v87,gms_v95,gms_jms_185}.json` |
| Evidence ledger | `docs/packets/evidence/<version>/summon.clientbound.*.yaml` |
| Templates (report re-gen) | `services/atlas-configurations/seed-data/templates/template_{gms_83_1,gms_84_1,gms_87_1,gms_95_1,jms_185_1}.json` |

## Per-version naming quirks (confirmed)

- **Version keys** (markers, `evidence pin --version`, evidence/audit dir names): `gms_v83`, `gms_v84`, `gms_v87`, `gms_v95`, `jms_v185`.
- **jms export file** keeps the historical name `gms_jms_185.json` even though its version key is `jms_v185` (`tools/packet-audit/internal/matrix/model.go:17-20`; `matrix.ExportPath("jms_v185")` resolves it). The jms **audit dir** is `jms_v185/` (NOT `gms_jms_185/`) — project memory "jms audit-dir name mismatch."
- **v84 IDB** is `GMS_v84.1_U_DEVM` (the clean DEVM build, per registry notes from the task-100 reshift).

## Committed report addresses (starting points — confirm/correct live)

From `docs/packets/audits/<version>/Summon*.json` during planning. These are where to start the decompile; the live active address is authoritative.

| Packet | v83 | v84 | v87 | v95 (tier-0 ref) | jms |
|---|---|---|---|---|---|
| Spawn (OnCreated) | 0x938f61 ⚠️active 0x95ADEC | 0x97038b | 0x9b3749 | 0x75a9a0 | 0x9f80f8 |
| Remove (OnRemoved) | 0x7a64eb | 0x7cbfa1 | 0x7f8cb0 | 0x75a470 | 0x828502 |
| Move (OnMove) | 0x7a6861 | 0x7cc317 | 0x7f902b | **0x759830** | 0x8286e4 |
| Attack (OnAttack) | 0x7a6882 | 0x7cc338 | 0x7f904c | 0x759860 | 0x828707 |
| Damage (OnHit) | 0x7a6ebe | 0x7cc984 | 0x7f969f | 0x7598c0 | 0x828d16 |
| Skill (OnSkill) | 0x7a6e5a | 0x7cc920 | 0x7f963b | 0x759890 | 0x828cb2 |

⚠️ = active-vs-inactive re-point candidate (plan §4 / §R5). The v95 SummonMove `0x759830` is the only tier-0 marker target.

## Wire layouts (from the codecs — confirm each against the live read)

- **SummonSpawn:** `int ownerId(cid), int oid, int skillId, byte charLevel(0x0A visual), byte level, short x, short y, byte stance, short foothold(0 visual), byte movementType, bool !puppet, bool !animated`, then **iff `spawnHasAvatarLook`** a trailing `byte bAvatarLook=0`. `spawnHasAvatarLook` = true for GMS≥95 and JMS≥185; false for GMS v83/v84/v87. (No roster summon carries an avatar look; Tesla Coil out of roster.)
- **SummonRemove:** `int ownerId, int oid, byte animated?4:1`.
- **SummonMove:** `int cid, int oid, raw CMovePath blob` — the blob already begins with start x,y (`CMovePath::Encode`); do NOT write the position separately (mis-aligns the observer's `CMovePath::Decode` by 4 bytes → client error 38).
- **SummonAttack:** `int cid, int oid, byte charLevel(0), byte direction, byte count, per target {int monsterOid, byte 6, int damage}`, then **GMS≥95 only** a trailing `byte 0` flag. v83/v84/v87/jms have NO trailing byte.
- **SummonDamage:** `int cid, int oid, byte attackIdx(12), int damage, int monsterIdFrom, byte bLeft(0)`. Stops at bLeft on ALL versions (the dir<0 byte belongs to the serverbound `SetDamaged`, not this broadcast).
- **SummonSkill:** `int cid, int oid, byte (stance&0x7F)`. No summonSkillId int on any version.

The `oid` is present on **all** versions (the dispatcher pre-reads `cid`, so the per-op `Decode4` is the `oid`). This is the universal correction from the v83 x32dbg finding.

## IDB instances

Confirm by binary **name**, never hardcode the port (launch order shifts them). Memory (IDBs_v9): v83=13341, v84=13337, v87=13340, v95=13339, jms=13338. Enumerate with `mcp__ida-pro__list_instances`, then `select_instance(<port>)`. **`select_instance` is shared global state** — one IDB at a time, never two versions in parallel; that is why the plan batches one version per task.

## Sequencing & dependencies

1. **Task 1 (v95 SummonMove, tier-0)** first — simplest, validates the marker→matrix loop and confirms the tier-0 no-evidence path before heavier work.
2. **Task 2 (v83)** — active dispatch best understood (task-088); resolves the SummonSpawn re-point that later versions may inherit.
3. **Task 3 (v84)** — watch the off-by-one gate class (clientbound stays v83-shaped).
4. **Task 4 (v87)**.
5. **Task 5 (jms)** last — SMC risk isolated so a jms blocker doesn't stall GMS cells.
6. **Task 6** — acceptance gate.

No task depends on another's code (each cell is independent), but they share `select_instance` global state and the STATUS.md/status.json regeneration, so they run **serially**. The PR branch is produced by rebase at PR time — one worktree, no mid-task forks.

## Acceptance bar (confirmed baseline)

`go run ./tools/packet-audit matrix --check` exits **0** at baseline in this worktree (confirmed during planning — no pre-existing summon conflicts). So the bar is strict: exit 0 after every commit, no new orphan/dangling/stale/drift lines, conflict count stays 0. Final gates: `go test -race ./... && go vet ./... && go build ./...` clean in `libs/atlas-packet`; `GOWORK=off tools/redis-key-guard.sh` clean; `docker buildx bake atlas-channel` **only if** a §R3 codec fix touched a service `go.mod` (test-only fixture changes do not).

## Gotchas

- Do not rename existing test funcs — tests reference internals (project memory). Add new `…Bytes<VER>` funcs; reuse shared body vars (`summonSpawnV83Body`, `summonAttackV83Body`, `summonDamageV83Body`) after live confirmation.
- `damage_test.go` already has `TestSummonDamageBytesV87`; `spawn_test.go` already has `TestSummonSpawnBytesV83` and `TestSummonSpawnBytesJMS185`; `move_test.go` has `TestSummonMoveBytes` (v83) and `TestSummonMoveBytesV95`. Confirm-and-mark these rather than duplicating.
- `evidence pin` resolves the address from the export by the `--ida` FName (`cmd/evidence.go:45`). So the marker `ida=` must equal what that FName resolves to in the export — re-pointing the export changes the resolved address, keeping marker/report/evidence in sync.
- After editing an export entry post-pin, a hash-drift on the record is cosmetic → re-pin (do not re-verify from scratch unless the byte content changed).
- `redis-key-guard.sh` runs from the repo **root** with `GOWORK=off` (project memory).
