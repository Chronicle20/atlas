# MOB/MONSTER Packet Family — Byte-Plumbing Batch 1 — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-13
---

## 1. Overview

The task-085 packet-audit coverage matrix grades every (packet × direction × version) cell of the MapleStory protocol against Atlas's implementation. After task-085 landed, ~600 named client operations across the five supported versions (gms_v83/v84/v87/v95, jms_v185) have **no `libs/atlas-packet` encoder/decoder at all** — they show as `incomplete` not because they're mis-encoded, but because Atlas simply doesn't speak them yet. The program goal is to drive that `incomplete` count to 0 by **implementing the protocol**, version by version, using the matrix as the burndown gate. Reclassifying out-of-scope packets as `n/a` is explicitly *not* the strategy — if a packet exists in the client, Atlas should eventually speak it.

This task is **batch 1**: the **MOB/MONSTER operation family** — 42 named, currently-unimplemented operations (30 universal across all 5 versions, plus a 4v/3v/2v/1v tail). It is scoped to **byte-plumbing only**: implement byte-exact `Encode` (clientbound) / `Decode` (serverbound) in `libs/atlas-packet`, wire each into the owning service with seed-template routes + validators across every applicable version, and add byte-fixture tests + `packet-audit:verify` markers + evidence records so each coverage cell promotes to `verified`. Full gameplay *behavior* (the server actually acting on / triggering these packets) is deliberately out of scope and follows in later behavior-focused tasks.

Beyond closing ~190 matrix cells, this batch's second deliverable is a **documented, repeatable per-packet recipe** that templates the remaining ~420 named unimplemented operations across the other domains (character, party, guild, cashshop, npc, field, …).

## 2. Goals

Primary goals:
- Implement byte-exact encoders/decoders in `libs/atlas-packet/monster/` (and `libs/atlas-packet/character/` or `…/context/` where the client owner is `CWvsContext`/`CUserLocal`) for the 42 MOB/MONSTER operations enumerated in §4, with per-version structural deltas derived from the version IDBs.
- Wire each operation into its owning service: clientbound → a channel writer (+ producer trigger stub); serverbound → a channel handler with the correct validator (+ action stub). Add the opcode routes to all five seed templates (with validators — see the task-085 gotcha) and patch live tenants per the established operational procedure.
- For every operation, add a byte-fixture round-trip test + `packet-audit:verify` marker + evidence record so the matrix cell(s) promote to `verified`.
- Establish and document the **per-packet implementation recipe** (in `docs/packets/`) as the reusable template for subsequent batches.

Non-goals:
- **Gameplay behavior.** No server-side logic that *decides when* to emit a clientbound packet or *what to do* on a serverbound one beyond a clearly-marked stub. (e.g., monster-book card tracking, actual mob-catch removal, carnival match orchestration are later behavior tasks.)
- Non-MOB/MONSTER operation families (separate batches).
- Reclassifying any cell as `n/a` to "close" it.
- Re-implementing `SET_TAMING_MOB_INFO` if task-086 (Mount/Rider) already provides its encoder — dedupe instead (see §9).

## 3. User Stories

- As the Atlas channel, I can **decode** every serverbound mob packet a v83/v84/v87/v95/jms client sends (touch attack, mob-vs-mob damage, banish, CRC reply, drop-pickup, bomb, monster-book cover) into a typed model, so future behavior tasks have a clean entry point.
- As the Atlas channel, I can **encode** every clientbound mob packet (catch effect, affected, special-effect, reset-animation, CRC change, monster-book set-card/cover, carnival messages) byte-identically to what each client version expects.
- As a protocol maintainer, the coverage matrix shows the MOB/MONSTER family at **0 incomplete cells** (verified, or honestly n/a only where an op genuinely does not exist in a given version), and `matrix --check` stays green.
- As the next batch's implementer, I can follow a documented recipe that turns "registry op with an IDB fname" into "verified matrix cell" without re-deriving the process.

## 4. Functional Requirements

### 4.1 Operation inventory (42 ops)

Each row: operation, direction, #versions-applicable, client function anchor (the IDB source of truth for the byte layout). Grouped into sub-clusters that the plan may sequence independently.

**Cluster A — Mob combat / damage (serverbound-heavy, 5v):**
| Op | Dir | Anchor |
|---|---|---|
| FIELD_DAMAGE_MOB | serverbound | CMob::Update |
| MOB_DAMAGE_MOB | serverbound | CMob::SetDamagedByMob (`…_send_0xC7`) |
| MOB_DAMAGE_MOB_FRIENDLY | serverbound | CMob::Update (`…send_0xC5`) |
| TOUCH_MONSTER_ATTACK | serverbound | CUserLocal::TryDoingBodyAttack |
| MONSTER_BOMB | serverbound | CMob::TryFirstSelfDestruction |
| MOB_TIME_BOMB_END | serverbound | CMob::UpdateTimeBomb |
| MOB_SKILL_DELAY_END | serverbound | CMob::Update |
| MOB_AFFECTED | clientbound | CMob::OnAffected |
| MONSTER_SPECIAL_EFFECT_BY_SKILL | clientbound | CMob::OnSpecialEffectBySkill |
| RESET_MONSTER_ANIMATION | clientbound | CMob::OnSuspendReset (v84 = case 0xFA, named this session) |

**Cluster B — Catch / taming (5v):**
| Op | Dir | Anchor |
|---|---|---|
| CATCH_MONSTER | clientbound | CMob::OnCatchEffect |
| CATCH_MONSTER_WITH_ITEM | clientbound | CMob::OnEffectByItem |
| BRIDLE_MOB_CATCH_FAIL | clientbound | CWvsContext::OnBridleMobCatchFail |
| SET_TAMING_MOB_INFO | clientbound | CWvsContext::OnSetTamingMobInfo — **dedupe vs task-086** |

**Cluster C — Monster book (5v):**
| Op | Dir | Anchor |
|---|---|---|
| MONSTER_BOOK_SET_CARD | clientbound | CWvsContext::OnMonsterBookSetCard |
| MONSTER_BOOK_SET_COVER | clientbound | CWvsContext::OnMonsterBookSetCover |
| MONSTER_BOOK_COVER | serverbound | (fname missing — derive send-site from IDB) |

**Cluster D — CRC / misc plumbing (5v):**
| Op | Dir | Anchor |
|---|---|---|
| MOB_CRC_KEY_CHANGED | clientbound | CMobPool::OnMobCrcKeyChanged |
| MOB_CRC_KEY_CHANGED_REPLY | serverbound | CMobPool::OnMobCrcKeyChanged |
| MOB_BANISH_PLAYER | serverbound | CUserLocal::SendBanMapByMobRequest |
| MOB_DROP_PICKUP_REQUEST | serverbound | CMob::SendDropPickUpRequest |

**Cluster E — Monster Carnival (9 ops, 5v):** MONSTER_CARNIVAL (serverbound, CUIMonsterCarnival::RequestSend) + clientbound MONSTER_CARNIVAL_START / OBTAINED_CP / PARTY_CP / SUMMON / MESSAGE / DIED / LEAVE / RESULT (CField_MonsterCarnival::On*). Coherent minigame sub-feature; may be split to its own task if it bloats the plan.

**Cluster F — Version-tail (non-universal):** INC_MOB_CHARGE_COUNT (4v, clientbound), MOB_SKILL_DELAY (4v), MOB_SPEAKING (4v), MOB_ESCORT_COLLISION (3v, sb), MOB_ESCORT_FULL_PATH (2v), MOB_ESCORT_STOP_END_REQUEST (2v, sb), MOB_REQUEST_ESCORT_INFO (2v, sb), MOB_ATTACKED_BY_MOB (1v), MOB_ESCORT_RETURN_BEFORE/STOP/STOP_SAY (1v), MOB_NEXT_ATTACK (1v). Implement only for the versions where each is applicable. **Note:** several Cluster-F registry fnames look mislabeled (e.g. MOB_SPEAKING→OnIncMobChargeCount) — verify each against the IDB before implementing (same registry-staleness class as task-085 v84).

### 4.2 Per-packet recipe (the unit of work — must be documented)

For each (op, applicable versions):
1. **Derive structure** — decompile the anchor function (and its per-version siblings) from each applicable IDB; record the field order/types (Decode1/2/4/Str/Buffer) and any version deltas. Use the multi-instance IDA setup (v83=13337, v87=13338, v95=13339, jms=13340, v84=13341).
2. **Model + codec** — add an immutable model with `Encode(l, ctx)` (clientbound) / `Decode` (serverbound) in the appropriate `libs/atlas-packet/...` package, following the existing `monster/clientbound/spawn.go` pattern (version-branch on `ctx` where structure differs).
3. **Wire** — register a channel writer (clientbound) or handler+validator (serverbound); add the opcode route to all five seed templates **with a validator** (LoggedInValidator default; NoOp for connection-level); add a producer/handler **stub** marked `// behavior: task-NNN` where real logic is deferred.
4. **Verify** — byte-fixture round-trip test (per version) + `packet-audit:verify packet=<path> version=<ver> ida=<addr>` marker + evidence record; run `matrix` to confirm the cell flips to `verified`.

### 4.3 Coverage outcome

After this batch, every MOB/MONSTER cell in the matrix is `verified` for each version where the op is applicable. Any cell that cannot be verified must be justified in writing (genuine version-absence → `n/a` with evidence, not a silent skip).

## 5. API Surface

No REST/JSON:API surface. The "API" is the packet codec contract:
- New `libs/atlas-packet` models implementing the project's `Encode`/`Decode` signatures (match existing `monster/clientbound` files).
- New channel writer registrations (clientbound) and handler+validator registrations (serverbound).
- New `socket.handlers` / `socket.writers` entries in the five seed templates (`services/atlas-configurations/seed-data/templates/template_{gms_83,84,87,95}_1.json`, `template_jms_185_1.json`), each handler carrying a `validator`.

## 6. Data Model

No database entities or migrations. The "models" are immutable in-memory packet structs (private fields + getters + Builder, per project convention). Per-version structural variants are handled inside `Encode`/`Decode` via the tenant/version from `ctx`, not via separate types, unless a variant diverges enough to warrant its own model (decide in design).

## 7. Service Impact

- **`libs/atlas-packet`** — the bulk: new encoder/decoder files + byte tests under `monster/{clientbound,serverbound}` (and `character/`/`context/` for `CWvsContext`/`CUserLocal`-owned ops like SET_TAMING_MOB_INFO, MONSTER_BOOK_*, MOB_BANISH_PLAYER, TOUCH_MONSTER_ATTACK). `go.mod` not touched → no bake, but `go test -race`/`go vet` gates apply.
- **`atlas-channel`** — writer + handler registrations, validator wiring, and behavior stubs. Seed-template route additions live here conceptually but the files are in atlas-configurations.
- **`atlas-configurations`** — five seed templates gain the new handler/writer routes (with validators). Live-tenant patch + channel restart per the task-085 operational procedure.
- **`atlas-monsters`** — likely owner of clientbound producer triggers and serverbound action stubs for core mob ops (damage/affected/special-effect/CRC).
- **`atlas-monster-book`** — owner of MONSTER_BOOK_* producer/consumer stubs.

## 8. Non-Functional Requirements

- **Byte-exactness across versions** — each codec round-trips byte-identically to the client for every applicable version; verified by fixtures, not by static diff alone (the matrix's static analyzer is unreliable on mask/mode/sub-struct packets — byte-tests are the source of truth).
- **Validators mandatory** — every new `socket.handlers` entry carries a validator, or the channel silently drops it (`BuildHandlerMap` `continue`). Post-deploy check: `kubectl logs <atlas-channel> | grep "Unable to locate validator"` == 0.
- **Multi-tenancy** — codecs read version/region from `ctx`; no global state.
- **No behavior regressions** — stubs must be inert (no side effects beyond logging) so wiring a route can't change existing gameplay.
- **Gate** — `go test -race ./...` + `go vet ./...` clean in changed modules; `docker buildx bake` for any service whose `go.mod` changed; `tools/redis-key-guard.sh` clean; `matrix --check` exit 0 with the new cells verified.

## 9. Open Questions

1. **SET_TAMING_MOB_INFO vs task-086** — does task-086 (Mount/Rider) already ship a `libs/atlas-packet` encoder for this, or only a seed-template writer route? If the encoder exists, this task only adds the byte-test + marker + evidence to flip the cell; if not, implement it here. Resolve before planning.
2. **MONSTER_BOOK_COVER (serverbound)** has no registry fname — derive its send-site from the IDBs during structure-derivation.
3. **Cluster-F fname mislabels** — verify each non-universal op's anchor against the IDB before implementing (registry staleness, same class as task-085 v84).
4. **Producer/handler ownership** — confirm per packet whether the clientbound producer trigger belongs in atlas-monsters vs atlas-channel vs atlas-monster-book; serverbound action stub likewise. (Stubs only — but they must land in the right service.)
5. **Carnival split** — keep Cluster E (9 ops) in this task or spin it into task-093? Decide at plan time based on total size.
6. **Behavior-stub convention** — agree the exact stub shape (log + TODO marker referencing the future behavior task) so reviewers don't flag them as incomplete deliverables.

## 10. Acceptance Criteria

- [ ] Every MOB/MONSTER operation in §4.1 has a byte-exact codec in `libs/atlas-packet` for each version where it is applicable (or a written n/a justification with IDB evidence for genuine version-absence).
- [ ] Each is wired: clientbound writer / serverbound handler+validator registered; opcode route present in all five seed templates with a validator.
- [ ] Each has a per-version byte-fixture round-trip test + `packet-audit:verify` marker + evidence record.
- [ ] `matrix` regenerated; all targeted MOB/MONSTER cells show `verified`; `matrix --check` exits 0; 0 conflicts.
- [ ] `go test -race ./...` and `go vet ./...` clean in every changed module; `docker buildx bake` clean for any service whose `go.mod` changed; `tools/redis-key-guard.sh` clean.
- [ ] Live v83/v84/v87/v95/jms tenants patched with the new routes + channel restarted; `Unable to locate validator` count == 0; no error/fatal logs.
- [ ] Behavior stubs are inert and clearly marked with a reference to the future behavior task; no existing gameplay changes.
- [ ] The per-packet recipe (§4.2) is written up in `docs/packets/` as the reusable template for subsequent batches.
- [ ] Code review (plan-adherence + backend-guidelines) passes.
