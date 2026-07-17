# Portable Chair Recovery — Design

Task: task-141-portable-chair-recovery
Status: Proposed
Inputs: `docs/tasks/task-141-portable-chair-recovery/prd.md` (approved)

---

## 1. Summary

The PRD asked for an implementation of `STATE_CHANGE_BY_PORTABLE_CHAIR_REQUEST` (STATUS row 562, `CWvsContext::TryRecovery`) on the assumption that this packet is the recurring recovery tick that makes recovery-stat chairs heal. IDA decompilation of all five target clients disproves that premise:

1. **Chair recovery amounts ride the already-implemented `HEAL_OVER_TIME` packet** (STATUS row 577, ✅ ×5). While seated on a portable chair with `recoveryHP`/`recoveryMP`, the client substitutes the chair's flat item values for its natural-regen amounts and sends them through the same `SendStatChangeRequest` opcode on the same cadence.
2. **Row 562 is a one-shot, empty-body notification.** The client sends it exactly once per sit, ≥20 s after sitting on a portable chair whose item data has no `spec` node, and expects no response. It carries no recovery data in any of the five versions.

The design therefore splits the task into its true parts:

- **A. Row 562 implementation** — empty-body codec, thin logging handler, byte fixtures, evidence records, seed-template wiring at the per-version opcodes, STATUS ❌→✅ ×5. Faithful to verified client semantics: fire-and-forget, no gameplay effect.
- **B. Server-authoritative chair recovery (the PRD's real gameplay goal, FR-4)** — harden the existing `HEAL_OVER_TIME` path: route recovery ticks through **atlas-chairs**, which validates sit state, sources amounts from item data (never from the client), and rate-limits. This is where "recovery chairs heal, server decides" actually lives.
- **C. `recoveryMP` in atlas-data** (FR-3) — unchanged from the PRD.
- **D. A latent bug fix found during grounding:** the gms_95 seed template's `CharacterHealOverTimeHandle` entry is missing its `validator`, so the handler is silently dropped on v95 (known `BuildHandlerMap` gotcha) — **all HP/MP regen application, including chair recovery, is dead on v95 today**. On v83/v84/v87/jms the handler is live, so recovery chairs likely already heal there (client-trusted amounts); the v95 symptom is a genuine total failure.

## 2. Ground truth (IDA, all five versions)

All decompilation was performed against the project IDB instances; per CLAUDE.md nothing below is taken from other server implementations or memory.

### 2.1 Row 562: the packet is empty and one-shot

| Version | IDB (port) | TryRecovery | Send site | Opcode | Body |
|---|---|---|---|---|---|
| gms_v83 | v83_Me `MapleStory_dump.exe` (13342) | `CWvsContext::TryRecovery` @ `0xa02e34` | `COutPacket(_, 0x4A)` @ `0xa032ad` | 0x4A | **empty** |
| gms_v84 | `GMS_v84.1_U_DEVM` (13345) | `sub_A4D05A` @ `0xa4d05a` (unnamed in IDB, structurally identical) | `COutPacket(_, 74)` | 0x4A | **empty** |
| gms_v87 | `GMSv87_4GB` (13343) | `CWvsContext::TryRecovery` @ `0xa97e50` | `COutPacket(_, 0x4D)` | 0x4D | **empty** |
| gms_v95 | `GMS_v95.0_U_DEVM` (13341) | `CWvsContext::TryRecovery` @ `0x9d4020` | `COutPacket(_, 80)` | 0x50 | **empty** |
| jms_v185 | JMS v185 `MapleStory_dump_SCY` (13344) | `CWvsContext::TryRecovery` @ `0xae6f5a` | `COutPacket(_, 0x42)` | 0x42 | **empty** |

In every version the construction is `COutPacket(ctor with opcode)` → `CClientSocket::SendPacket` with **zero Encode calls in between**. The registry opcodes for row 562 are hereby confirmed correct in all five registries (the v83/v84 rows are `csv-import` provenance; this verifies them).

Send gate (identical semantics in all five; v83 `sub_959941`, v84 `sub_997B16`, v87 `sub_9DBF17`, v95 `CUserLocal::SetPortableChairStatSetSent` @ `0x904fc0`, jms `sub_A246D8`):

- `CWvsContext::CanSendExclRequest(500, 0)` passes (500 ms exclusive-request spacing), AND
- an active portable chair id is set, AND
- time since sitting ≥ **20000 ms**, AND
- a per-sit latch is unset; the gate then sets the latch → **sent at most once per sit**.

Additionally the packet is only sent when the chair item has **no `spec` node** (v83 `sub_5D5B62` reads StringPool `SP_2306_SPEC`; v95 `CItemInfo::IsTherePortableChairStatUp` @ `0x5ac8e0`; jms `sub_641171`). No clientbound response exists: the client latches locally and nothing awaits a reply (no `OnStateChangeByPortableChair`-style handler exists in any of the five IDBs; the v95 function-name search for `PortableChair*` returns only the senders and the item-info reader).

### 2.2 Chair recovery amounts ride HEAL_OVER_TIME

In every version, `TryRecovery` (called once per frame from `CWvsContext::Update`; v83 caller @ `0xa03390`) computes HP and MP recovery and sends them through the **already-implemented** stat-change request (v83 `0xa1e997`, v95 `SendStatChangeRequest` @ `0x9f2a00`, jms `0xb054d6` = HEAL_OVER_TIME, STATUS row 577 ✅ ×5):

- Per stat, the client keeps an accumulator that adds **+30 per frame** and fires when it reaches a threshold:
  - HP natural: threshold 10000 (5000 with Warrior `Endure` 1000000; an Endure-family per-skill interval applies in prone/sit stances via e.g. v95 `sub_7B1A22`). Amount = `(base + 10) × fieldRecoveryRate` (×1.5 while sitting).
  - HP on a recovery chair: v95 `CItemInfo::GetPortableChairRecoveryRate(itemId, 1)` @ `0x5ac750` (v83 `sub_5D5A76`, reads `info/recoveryHP` / `info/recoveryMP` via StringPool `SP_1739_RECOVERYHP`/`SP_1740_RECOVERYMP`) — when non-zero the client sends the **flat item value** (no field-rate or sit multiplier) on the same threshold cadence.
  - MP: same structure, fixed threshold 10000; chair `recoveryMP` substitutes when present.
- The cadence is **frame-paced, not wall-clock-paced**: 10000 / 30 ≈ 334 update frames per tick — ≈11 s at a 30 fps update loop, ≈5.6 s at 60 fps. Any server rate limit must tolerate this spread (see §7).
- Per-stat exclusivity: when the chair has `recoveryHP > 0`, the chair amount **replaces** natural HP regen; a stat the chair doesn't cover (e.g. MP on an HP-only chair) continues to use natural regen. So seated characters legitimately send a mix of chair-valued and natural-valued ticks in the same session.

Consequence: on v83/v84/v87/jms — where `CharacterHealOverTimeHandle` is registered — sitting on The Relaxer already heals, because the handler applies the client-claimed amounts blindly (`services/atlas-channel/atlas.com/channel/socket/handler/character_heal_over_time.go`). The PRD's "chairs never heal" is accurate only for v95 (dead handler, §2.4) — and the blind-trust application is exactly the anti-cheat hole FR-4 wants closed.

### 2.3 WZ ground truth (v83-era dump, `Item.wz/Install/0301.img.xml`)

- 238 entries with `info/recoveryHP`, 178 with `info/recoveryMP` (e.g. 03010000 recoveryHP=50; 03010136 both at 60 per the PRD).
- **Zero** entries with a `spec` node → in v83-era data every portable chair passes the no-`spec` gate, so row 562 fires for every chair (once per sit, after 20 s). The `spec`-chair branch (client suppresses the packet) only matters for later data and needs no server work.

### 2.4 Latent v95 seed bug (in scope)

`services/atlas-configurations/seed-data/templates/template_gms_95_1.json` registers `{'opCode': '0x64', 'handler': 'CharacterHealOverTimeHandle'}` **without a `validator`**. Per the known gotcha, `BuildHandlerMap` skips validator-less entries with only a warning — the handler never registers, and every HEAL_OVER_TIME packet on v95 is dropped. Natural regen and chair recovery are both dead on v95. Fix: add `"validator": "LoggedInValidator"` to the seed template and patch live v95 tenants. This is squarely inside the PRD's goal ("recovery chairs heal on all five versions") and gets fixed in this task.

## 3. Scope re-alignment (PRD → design)

| PRD requirement | Where it lands |
|---|---|
| FR-1.* codec + fixtures for row 562 | Unchanged in intent; body is **empty** (§5.1) |
| FR-2.1 handler | Thin decode+log handler (§5.2); no recovery orchestration here — that was based on the disproved premise |
| FR-2.2 / FR-2.3 seed + live wiring | Unchanged; plus the v95 HEAL_OVER_TIME validator fix (§5.5, §8) |
| FR-3.* recoveryMP in atlas-data | Unchanged (§5.4) |
| FR-4.* server-authoritative recovery, rate limit | Applied to the **HEAL_OVER_TIME** path, the packet that actually carries recovery (§5.3, §7) |
| FR-4.3 "IDA-verify the client interval" | Verified: frame-paced, 10000 units at +30/frame; wall-clock 5.6–11 s across 60–30 fps (§2.2, §7) |
| FR-5.1 single validation owner | atlas-chairs owns it via a new `RECOVERY` command (§4) |
| §5 "New Kafka command only if design places validation in atlas-chairs" | It does: `RECOVERY` on the existing chair command topic |
| §2 non-goal "dedicated response packet if IDA reveals one" | IDA reveals none — no clientbound work |

## 4. Architecture decision: who owns recovery validation (FR-5.1)

**Decision: atlas-chairs owns it.** atlas-channel's `CharacterHealOverTimeHandleFunc` stops applying stats directly and instead emits a `RECOVERY` command on the existing chair command topic (`COMMAND_TOPIC_CHAIR`). atlas-chairs validates and emits `CHANGE_HP`/`CHANGE_MP` on `COMMAND_TOPIC_CHARACTER`.

### Options considered

**Option 1 — atlas-channel decides (REST reads to atlas-chairs + atlas-data).**
The handler queries `GET /chairs/{characterId}`, fetches setup data, applies validated amounts via the existing `character.Processor.ChangeHP/MP`.
- \+ No new command type; natural-regen path untouched.
- − The validation decision and rate-limit state land in atlas-channel; the sit-state check is effectively re-performed in channel per tick — the thing FR-5.1 forbids.
- − Rate-limit state in channel memory is lost on restart and invisible to any other consumer.

**Option 2 (chosen) — atlas-chairs orchestrates via Kafka command.**
- \+ Sit-state truth, recovery validation, and rate-limit state are co-located in one service; the rate-limit timestamps live inside the existing chair registry `Model`, so standing up (registry `Clear`) automatically ends recovery and discards the state — FR-4.5 and PRD §6 for free.
- \+ Matches the established pattern: `USE`/`CANCEL` chair commands already flow channel → chairs on this topic; same-key ordering means a `RECOVERY` following a `USE`/`CANCEL` is processed in order.
- \+ The PRD anticipated exactly this shape (§5).
- − atlas-chairs gains a `COMMAND_TOPIC_CHARACTER` producer (standard cross-service command emission, same as atlas-channel does today) and a `data/setup` REST client (mirrors its existing `data/map` client).
- − Natural-regen ticks (characters not on a recovery chair) also flow through atlas-chairs, which passes the claimed amounts through unchanged (§5.3). This keeps today's natural-regen behavior byte-for-byte while giving chair ticks a single authoritative owner. The semantic stretch (chairs forwarding non-chair regen) is accepted and documented; the alternative — demuxing in channel — requires sit-state knowledge in channel, which is the FR-5.1 violation.

**Option 3 — hybrid (channel applies natural, chairs applies chair ticks) was rejected:** channel cannot distinguish a chair tick from a natural tick without sit-state, so both services would apply the same packet (double heal) or channel would need the forbidden sit-state check.

### Row 562 handler placement

The packet has no effect to apply (verified §2.1) and validating "was actually seated ≥20 s" would only produce a log line. A Kafka round-trip for a once-per-sit log is not worth a new command type (YAGNI). **Decision: decode + debug-log in atlas-channel, no emission.** This is not a stub: it faithfully implements the packet's verified semantics (a fire-and-forget notification), removes the "unhandled message op" noise, and completes the coverage-matrix cell honestly.

## 5. Component design

### 5.1 libs/atlas-packet — `StateChangeByPortableChair` codec

New file `libs/atlas-packet/character/serverbound/state_change_by_portable_chair.go`, pattern-identical to `chair_portable.go`:

- `const CharacterStateChangeByPortableChairHandle = "CharacterStateChangeByPortableChairHandle"`
- `type StateChangeByPortableChair struct{}` — no fields.
- `Decode` reads nothing; `Encode` writes nothing; `String()` returns a fixed literal. No version branching — the body is empty in all five versions, so no reader options are needed.
- Godoc comment records the ground truth (fname `CWvsContext::TryRecovery`, per-version send sites/addresses, empty body, once-per-sit ≥20 s gate, no-`spec` condition) the way `heal_over_time.go` documents its wire layout.
- Byte-fixture tests ×5 with `packet-audit:verify` markers asserting a zero-length body round-trips (decode consumes nothing, encode emits nothing) under each version's tenant context. Evidence records pinned and matrix regenerated per `docs/packets/audits/VERIFYING_A_PACKET.md` (serverbound cell: marker + evidence + REPORT; surgical export splice only — the export is non-idempotent).

### 5.2 atlas-channel — two handler changes

**New** `socket/handler/character_state_change_by_portable_chair.go`:
decode `StateChangeByPortableChair`, `l.Debugf` with operation + characterId, return. Registered in `main.go`'s handler map like `CharacterChairPortableHandle`.

**Modified** `socket/handler/character_heal_over_time.go`:
replace the direct `character.NewProcessor(...).ChangeHP/MP` calls with a single
`chair.NewProcessor(l, ctx).Recover(s.Field(), s.CharacterId(), p.HP(), p.MP())`
which emits the new `RECOVERY` command (`kafka/message/chair` + provider in `chair/producer.go`, mirroring `Use`/`Cancel`). Ticks where both HP and MP are zero are dropped at the handler; non-zero claims — including negative ones, which the jms client sends as its own clamp-to-max corrections — are forwarded, preserving today's `!= 0` behavior.

### 5.3 atlas-chairs — recovery orchestration

**Kafka contract** (`kafka/message/chair/kafka.go`, mirrored into atlas-channel's copy):

```go
CommandRecovery = "RECOVERY"

type RecoveryCommandBody struct {
    CharacterId uint32 `json:"characterId"`
    Hp          int16  `json:"hp"` // client-claimed; trusted only for the natural-regen pass-through
    Mp          int16  `json:"mp"`
}
```

**Consumer arm** → `Processor.RecoverAndEmit(field, characterId, claimedHp, claimedMp)` (pure `Recover(mb)` + emitting wrapper per the processor pattern):

Per stat, independently (HP shown; MP identical with `recoveryMP`):

1. `GetById(characterId)` on the chair registry.
   - **Not seated, or seated on a `FIXED` seat, or seated portable chair with `recoveryHP == 0`:** natural-regen pass-through — emit `CHANGE_HP` with the claimed amount if > 0. (Fixed seats and non-recovery chairs still produce legitimate natural ticks with the client's 1.5× sit multiplier; verified §2.2.)
   - **Seated on a portable chair with `recoveryHP > 0`:** server-authoritative path:
     2. Rate limit: if `now − lastHpRecoveryAt < minTickInterval` → drop, debug/warn log with reason, no emission (FR-4.4; never disconnect).
     3. Emit `CHANGE_HP` with **the item's `recoveryHP`** — the claimed value is ignored for application (FR-4.2); if the claim differs from the item value, log at warn (anti-cheat signal).
     4. Update `lastHpRecoveryAt` in the registry model.

Clamping to max HP/MP remains atlas-character's job (existing `CHANGE_HP` semantics), per the PRD.

**Item data access:** new `data/setup` package mirroring the existing `data/map` (REST client to atlas-data `/data/setups/{id}`, `RestModel` with `recoveryHP`/`recoveryMP`, provider + processor). One GET per honored tick per seated character (≥4 s apart) is well within the PRD's performance budget; no new cache is introduced (none exists for `data/map` either).

**Registry model** (`chair/model.go`): add `lastHpRecoveryAt`, `lastMpRecoveryAt` (unix-milli `int64`, zero = never) to `Model`, its custom JSON marshal/unmarshal, and builder-style `WithHpRecoveryAt`/`WithMpRecoveryAt` copies. Stored in the existing Redis `TenantRegistry` under the same key, so:

- tenant-scoped by construction (NFR),
- `Clear` (stand up, `CANCEL_CHAIR`) removes the timestamps with the registration — subsequent ticks fall to the not-seated branch (FR-4.5),
- Kafka same-key ordering makes the read-modify-write single-writer per character.

### 5.4 atlas-data — `recoveryMP`

- `setup/reader.go`: parse `info/recoveryMP` with default 0 next to `recoveryHP`.
- `setup/rest.go`: `RecoveryMP uint32 \`json:"recoveryMP"\`` on `RestModel` (setups are stored as marshaled `RestModel` documents in the `SETUP` document storage, so the REST field and the persisted field are the same change).
- Tests: reader cases for a both-stats chair (03010136-style), an HP-only chair, and a neither chair; `resource_test` attribute assertion.
- **Backfill (FR-3.3):** documents ingested before this change unmarshal `recoveryMP` as 0. End state per environment: re-ingest the canonical tenant's `Item.wz` with the new reader, then re-publish the canonical baseline so baseline-bootstrap environments pick it up; tenants with their **own** (non-canonical) setup rows additionally need a per-tenant re-ingest, because per-id reads prefer tenant rows over canonical fallback. The plan phase writes the exact runbook; the design commits to "re-ingest canonical + baseline re-publish" as the mechanism.

### 5.5 Seed templates + live tenant configs

`services/atlas-configurations/seed-data/templates/`:

| Template | Add handler entry | Also |
|---|---|---|
| `template_gms_83_1.json` | `{"opCode": "0x4A", "validator": "LoggedInValidator", "handler": "CharacterStateChangeByPortableChairHandle"}` | — |
| `template_gms_84_1.json` | same, `0x4A` | — |
| `template_gms_87_1.json` | same, `0x4D` | — |
| `template_gms_95_1.json` | same, `0x50` | **add missing `"validator": "LoggedInValidator"` to the `0x64` `CharacterHealOverTimeHandle` entry** |
| `template_jms_185_1.json` | same, `0x42` | — |

Every entry carries an explicit validator (missing validator = silently dropped handler). Live tenants: seed templates apply only at creation, so existing tenant socket-handler configs must be PATCHed with the new entry (and the v95 validator fix) and atlas-channel restarted — the established rollout runbook; the plan documents the exact PATCH steps per environment (FR-2.3).

## 6. Data flow

```
Client (seated on The Relaxer, ~every 334 update frames)
  └─ HEAL_OVER_TIME (0x59 v83) {updateTime, val, hp=50, mp=0, opt}
       └─ atlas-channel CharacterHealOverTimeHandleFunc
            └─ Kafka COMMAND_TOPIC_CHAIR RECOVERY {characterId, hp:50, mp:0}
                 └─ atlas-chairs RecoverAndEmit
                      ├─ registry: seated PORTABLE 3010000
                      ├─ atlas-data GET /data/setups/3010000 → recoveryHP=50, recoveryMP=0
                      ├─ rate limit ok → CHANGE_HP {characterId, amount:50}  (item value)
                      └─ MP claim 0 → nothing
Client (same sit, once, ≥20 s after sitting)
  └─ STATE_CHANGE_BY_PORTABLE_CHAIR_REQUEST (0x4A v83, empty)
       └─ atlas-channel CharacterStateChangeByPortableChairHandleFunc → debug log, done
Client (not seated, natural tick)
  └─ HEAL_OVER_TIME {hp≈17, mp≈3}
       └─ channel → RECOVERY → chairs: no registration → pass-through CHANGE_HP/CHANGE_MP (claimed)
```

## 7. Rate limit (FR-4.3)

- **Verified client cadence:** per-stat accumulator +30 per `CWvsContext::Update` frame, threshold 10000 (HP threshold 5000 with Endure-family skills) → one tick per ≈334 frames. Wall-clock therefore depends on the client's frame rate: ≈11.1 s at 30 fps, ≈5.6 s at 60 fps. There is no wall-clock timer in the client to pin a single "true" interval to (this is the precise answer to PRD open question 2).
- **Server floor:** `minRecoveryTickInterval = 4000 ms` per stat per character. Rationale: comfortably below the fastest legitimate cadence observed at plausible frame rates (≥5.6 s at 60 fps; even a 75 fps client ticks at ≈4.5 s), while capping a spamming client at ≤15 chair-valued heals/min — versus ≤11/min legitimate — a negligible advantage since amounts are item-fixed anyway. The Endure-threshold-5000 case only applies to natural (pass-through) HP ticks, which are not rate-limited in this task.
- The constant lives in atlas-chairs. It is server-internal policy, not a client-wire value, so DOM-25 (config-resolved wire bytes) does not apply; a tenant-config knob is deliberately omitted (YAGNI — add one if a future client version measurably changes cadence).
- Rejections log at debug with a structured reason (`rate`, per FR-4.4) and never disconnect.

Anti-cheat properties after this design: while seated on a recovery chair, amounts are item-data-fixed and paced server-side; a forged claim value is ignored and logged. Row 562 grants nothing, so forging it is harmless. Natural-regen claims remain client-trusted pass-through — explicitly out of scope (§10), unchanged from today.

## 8. Rollout order

1. atlas-data (recoveryMP parse/REST) — deploy, re-ingest canonical, re-publish baseline, per-tenant re-ingest where needed (§5.4).
2. atlas-chairs (RECOVERY consumer, registry model, setup client) — deploy. Safe while channel still applies directly: the consumer simply receives nothing yet.
3. atlas-channel (both handler changes) + lib — deploy.
4. Seed templates merge with the services; live tenant PATCH (new 562 handler entry ×5 opcodes + v95 heal-over-time validator) + channel restart.

Step 1 before step 3 avoids the transient where MP chairs heal HP only (chairs would read `recoveryMP=0` from stale documents). If ordering can't be honored operationally, the transient is benign (HP still heals; MP falls back to natural pass-through) but the plan should keep the order.

## 9. Error handling & edge cases

- **atlas-data lookup fails** during a seated tick: log warn, drop the tick (fail-closed for the authoritative path; the next tick retries). Never fall back to the claimed value.
- **Stand-up race:** a `RECOVERY` in flight behind a `CANCEL` on the same key is processed after it (same topic, same key ordering) → not-seated branch → pass-through of one chair-valued claim (≤ the chair's own recovery value, once). Accepted; bounded and logged.
- **Claim/item mismatch while seated:** apply item value, warn log. Covers both hacked clients and version data drift.
- **Negative claims (jms clamp-to-max corrections):** forwarded and pass-through-applied like today; atlas-character's own clamping remains the final authority. On the seated-recovery path the item value replaces the claim regardless of its sign.
- **Character on a `spec` chair (future data):** client sends neither 562 nor chair-valued ticks it can't justify; nothing to handle server-side now (zero such chairs in v83-era data).
- **562 while not seated:** debug log only — by design the handler validates nothing (no effect to protect).

## 10. Out of scope (explicit, with rationale)

- **Natural-regen claim validation** (bounding non-seated `HEAL_OVER_TIME` claims): requires job/skill data (Improving HP Recovery, Endure), field recovery rates, and sit multipliers — a feature of its own. Today's blind-trust behavior is preserved for the natural path; the new `RECOVERY` command boundary is the natural place to add it later.
- **Fixed-seat 1.5× validation** — same bucket.
- **`spec`-node chair buffs** — no such chairs exist in the supported data (§2.3).
- **Clientbound work** — none required (verified §2.1).

## 11. Testing & verification

- **libs/atlas-packet:** empty-body round-trip fixtures ×5 with `packet-audit:verify` markers; `version_bounds_test.go` entry if the suite enumerates codecs. Evidence records + REPORT per `VERIFYING_A_PACKET.md`; regenerate matrix; row 562 ✅ ×5 with `packet-audit matrix --check`, `fname-doc --check`, `operations --check` clean.
- **atlas-chairs:** processor tests via the Builder pattern (no `*_testhelpers.go`): seated+recovery-chair happy path (HP-only chair; HP+MP chair), item-value-overrides-claim, rate-limit rejection, not-seated pass-through, fixed-seat pass-through, non-recovery-portable pass-through, data-lookup-failure drop, `Clear` resets timestamps. Registry marshal round-trip with timestamps.
- **atlas-channel:** handler emits `RECOVERY` with decoded values; 562 handler decodes empty body without error.
- **atlas-data:** reader/resource tests per §5.4.
- **Gates:** `go test -race`, `go vet`, `go build` in every changed module; `docker buildx bake` for atlas-channel, atlas-chairs, atlas-data (and any service whose `go.mod` changes via the lib bump); `tools/redis-key-guard.sh` (registry change stays inside `atlas-redis` types).
- **Live acceptance (v83 tenant):** The Relaxer restores HP at the client cadence; an HP+MP chair (03010136) restores both; standing stops it; v95 tenant regains regen after the validator patch; packet 0x4A produces the debug log once per sit, no disconnect.

## 12. PRD open questions — resolved

1. **Packet body:** empty, all five versions (§2.1).
2. **Client send interval:** frame-paced (10000 units at +30/frame) for HEAL_OVER_TIME ticks; 562 itself is once-per-sit ≥20 s. Server floor 4000 ms per stat (§7); no per-version config needed — the mechanism is identical in all five clients.
3. **Acknowledgment:** none expected or possible; no clientbound counterpart exists (§2.1).
4. **FR-5.1 placement:** atlas-chairs orchestrates via `RECOVERY` command (§4).
5. **recoveryMP backfill:** canonical re-ingest + baseline re-publish; per-tenant re-ingest where tenant-local rows exist (§5.4); exact runbook in the plan.

## 13. Risks

- **Regen availability now depends on atlas-chairs + Kafka** (previously channel → character direct). Ticks buffer in Kafka during a chairs outage and apply on recovery; regen is not latency-critical. Accepted.
- **Behavior change on the natural path is nil by construction** (pass-through), but the routing change touches every character's regen — the live acceptance step explicitly re-verifies natural regen on v83.
- **Baseline re-publish** has a history of column/order fragility; the setups change is JSONB-document-shaped (no DDL), which avoids the known binary-COPY pitfalls, but the plan should still verify a restore round-trip on an ephemeral env.
- **v95 validator fix enables a formerly-dead handler** — v95 clients will begin applying regen requests that were previously dropped; this is the intended fix, called out so the change in live behavior isn't mistaken for a regression.
