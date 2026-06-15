# Keydown Skill Prepare/Cancel Broadcast — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** atlas-channel relays the keydown-skill *prepare* (keydown) and *cancel* (keyup) packets so observers in a map see the cast aura start and stop. Server is a validated relay: decode serverbound prepare/cancel → validate ownership + `skill.IsKeyDownSkill` → broadcast a clientbound *foreign* packet to other map sessions. No persistence/Kafka/cross-service. Versions: **v83, v84, v87, v95, JMS185** (v92 deferred — no IDB). Opcodes and read/write orders are **pinned from each version's client IDB**, never the registry CSV or Cosmic.

**Architecture:** Per design.md. Two serverbound handlers + two clientbound foreign writers + their codecs. The single highest risk is per-version wire correctness, so **Task 1 pins the wire spec from the IDBs before any code is written**, and every codec ships with a byte-fixture test pinned to that spec plus a version round-trip test.

**Tech stack:** Go (atlas-channel service + libs/atlas-packet + libs/atlas-constants). `pt` packet-test helpers (`pt.Variants`/`pt.CreateContext`/`pt.RoundTrip`). ida-pro MCP for verification. JSON seed templates. No `go.mod` changes → docker-bake gate N/A.

---

## File Structure

| File | Responsibility after change |
|---|---|
| `docs/tasks/task-099-keydown-skill-prepare-broadcast/wire-spec.md` | NEW. Per-version (×5) opcode + read/write-order table for all four ops, pinned from IDBs. Single source for all downstream tasks. |
| `libs/atlas-packet/model/skill_prepare_info.go` | NEW. `SkillPrepareInfo` (prepare) + `SkillCancelInfo` (cancel) serverbound codecs, version-conditional `Decode`/`Encode`. |
| `libs/atlas-packet/character/clientbound/skill_prepare_foreign.go` | NEW. `CharacterSkillPrepareForeign` codec + `CharacterSkillPrepareForeignWriter` const. |
| `libs/atlas-packet/character/clientbound/skill_cancel_foreign.go` | NEW. `CharacterSkillCancelForeign` codec + `CharacterSkillCancelForeignWriter` const. |
| `libs/atlas-packet/character/*.go` | Body constructors `CharacterSkillPrepareForeignBody` / `CharacterSkillCancelForeignBody` (mirror `CharacterSkillUseEffectForeignBody`). |
| `libs/atlas-packet/**/*_test.go` | Round-trip tests (`pt.Variants`) + byte-fixture tests (`packet-audit:verify`) for each codec. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare.go` | NEW. `CharacterSkillPrepareHandleFunc` + `CharacterSkillCancelHandleFunc`. |
| `services/atlas-channel/atlas.com/channel/socket/handler/effects.go` | Add `AnnounceForeignSkillPrepare` / `AnnounceForeignSkillCancel`. |
| `services/atlas-channel/atlas.com/channel/main.go` | Register 2 handlers (`produceHandlers` ~L788) + 2 writers (`produceWriters` ~L683). |
| `services/atlas-configurations/seed-data/templates/template_{gms_83_1,gms_84_1,gms_87_1,gms_95_1,jms_185_1}.json` | Add `socket.handlers[]` (validator `LoggedInValidator`) + `socket.writers[]` rows. |
| `docs/packets/audits/STATUS.md`, `docs/packets/registry/*.yaml`, per-version audit docs | Promote the four ops × five versions; pin evidence. |

## Conventions used in every task

- All paths are relative to the task worktree root (`<repo-root>/.worktrees/task-099-keydown-skill-prepare-broadcast`).
- Before each commit, confirm worktree + branch:
  ```bash
  git rev-parse --show-toplevel   # must end with /.worktrees/task-099-keydown-skill-prepare-broadcast
  git branch --show-current       # must be task-099-keydown-skill-prepare-broadcast
  ```
- Go gates per changed module: `go test -race ./...`, `go vet ./...`, `go build ./...` (run from the module dir: `libs/atlas-packet`, `services/atlas-channel/atlas.com/channel`). Plus `tools/redis-key-guard.sh` from repo root (`GOWORK=off`).
- Test setup uses the project Builder pattern; no `*_testhelpers.go`.
- IDB instances (read-only; `select_instance` per call): v83 `:13342`, v84 `:13337`, v87 `:13341`, v95 `:13340`, JMS185 `:13339`. Demangled `Class::Method` names often fail lookup — navigate via opcode dispatch / xrefs / addresses.

---

## Task 1: Pin the per-version wire spec from the IDBs

This is the dependency gate. Produce `wire-spec.md`: for each of v83/v84/v87/v95/jms185, the opcode + exact byte read/write order for all four ops. Every later task consumes this; do not guess opcodes/orders elsewhere.

**Files:** create `docs/tasks/task-099-keydown-skill-prepare-broadcast/wire-spec.md`.

- [ ] **Step 1: Per-version IDB extraction.** For each version (dispatch IDA work per-IDB; v95 baseline already known from design: serverbound prepare `0x069`=skillId u32/level u8/action u16[bit15=move|low15=action]/actionSpeed u8; serverbound cancel `0x068`=skillId u32; clientbound remote-prepare nType215 `0x0D7`=charId+above; clientbound remote-cancel nType217=charId+skillId), pin from the client:
  1. serverbound prepare opcode + read order (`CUserLocal::DoActiveSkill_Prepare`).
  2. serverbound cancel opcode + read order (keyup `SendSkillCancelRequest`) — **do NOT trust Cosmic's CANCEL_BUFF**.
  3. clientbound remote-prepare opcode + read order (`CUserRemote::OnSkillPrepare`).
  4. clientbound remote-cancel opcode + read order (`CUserRemote::OnSkillCancel`).
  Cross-check each opcode against `docs/packets/registry/<ver>.yaml` (SKILL_EFFECT etc.) — registry is a hint, IDB is truth; note any disagreement.
- [ ] **Step 2: Resolve OQ-5 (MovingShootAttackPrepare).** Determine, per version, whether any in-scope `skill.IsKeyDownSkill` skill dispatches through `OnMovingShootAttackPrepare` (nType 216 / `MOVING_SHOOT_ATTACK_PREPARE`) instead of `OnSkillPrepare` (nType 215). Record the finding. If an in-scope skill needs it, add a "moving-shoot prepare" row to wire-spec and a note that Tasks 2–6 gain a parallel packet; if not, mark MovingShoot **out of scope** with the evidence.
- [ ] **Step 3: Write `wire-spec.md`** — a table keyed by (op, version) → {opcode, ordered field list with widths, source fname+address}. Include a "read-order deltas across versions" note (per design D5/OQ-3). Mark any field uncertain rather than guessing.
- [ ] **Step 4: Commit.**
  ```bash
  git add docs/tasks/task-099-keydown-skill-prepare-broadcast/wire-spec.md
  git commit -m "spec(task-099): pin per-version prepare/cancel wire spec from IDBs"
  ```

---

## Task 2: Serverbound codecs — `SkillPrepareInfo` + `SkillCancelInfo` (TDD)

**Files:** create `libs/atlas-packet/model/skill_prepare_info.go` + `_test.go`. Anchors: mirror `libs/atlas-packet/model/attack_info.go` (struct + `Decode(l,ctx)(r,options)` + `tenant.MustFromContext(ctx)` version branches, L181-236) and the test convention in `attack_info_test.go` (L9-54: `pt.Variants`, `pt.CreateContext`, `pt.RoundTrip`).

- [ ] **Step 1 (RED): round-trip + byte-fixture tests.** Add `skill_prepare_info_test.go`: (a) a `pt.Variants` round-trip test for `SkillPrepareInfo` and `SkillCancelInfo` (`pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)`); (b) a byte-fixture test per version asserting the exact bytes from `wire-spec.md` (mark with a `packet-audit:verify` comment). Build the models via setters/Builder, no `_testhelpers.go`. Run `cd libs/atlas-packet && go test ./model/ -run SkillPrepare` → FAIL (types don't exist).
- [ ] **Step 2 (GREEN): implement the codecs.** `SkillPrepareInfo{skillId uint32; level byte; action uint16; actionSpeed byte}` with getters/setters + `Decode`/`Encode(l,ctx)` version-conditional per `wire-spec.md`. `SkillCancelInfo{skillId uint32}` likewise. Use `tenant.MustFromContext(ctx)` for any version deltas Task 1 found. Run the tests → PASS.
- [ ] **Step 3: gates + commit.** `cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./...`; then commit `feat(atlas-packet): serverbound skill prepare/cancel codecs`. Verify branch/worktree.

---

## Task 3: Clientbound foreign codecs + body constructors + writer consts (TDD)

**Files:** create `libs/atlas-packet/character/clientbound/skill_prepare_foreign.go`, `skill_cancel_foreign.go` (+ tests); add body constructors in `libs/atlas-packet/character/`. Anchors: `libs/atlas-packet/character/clientbound/effect_skill_use.go` (`EffectSkillUseForeign` struct/Encode/Decode/`Operation()`, L106-201), writer const in `clientbound/effect.go` (L12-13), and the `character` body-constructor pattern used by `effects.go` (`CharacterSkillUseEffectForeignBody`).

- [ ] **Step 1 (RED): tests.** `pt.Variants` round-trip + per-version byte-fixture (from `wire-spec.md`) for `CharacterSkillPrepareForeign` (fields: characterId + skillId + level + action + actionSpeed) and `CharacterSkillCancelForeign` (characterId + skillId). Run → FAIL.
- [ ] **Step 2 (GREEN): implement.** Two structs with `Encode(l,ctx)(options)` (version-conditional per wire-spec), `Decode` for round-trip symmetry, and `Operation()` returning the new writer-name const. Declare `const CharacterSkillPrepareForeignWriter = "CharacterSkillPrepareForeign"` and `const CharacterSkillCancelForeignWriter = "CharacterSkillCancelForeign"`. **Make `Operation()` return the matching foreign writer const** (note: existing `EffectSkillUseForeign.Operation()` returns the non-foreign const — do NOT copy that; registration/byte-fixtures rely on the correct name).
- [ ] **Step 3: body constructors.** Add `CharacterSkillPrepareForeignBody(characterId uint32, info SkillPrepareInfo) packet.Encode` and `CharacterSkillCancelForeignBody(characterId, skillId uint32) packet.Encode` in `libs/atlas-packet/character/`, mirroring `CharacterSkillUseEffectForeignBody`.
- [ ] **Step 4: gates + commit.** Module gates as Task 2; commit `feat(atlas-packet): clientbound foreign skill prepare/cancel writers`.

---

## Task 4: atlas-channel handlers, broadcast helpers, registration (TDD)

**Files:** create `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare.go` (+ test); edit `socket/handler/effects.go`; edit `main.go`. Anchors: `character_skill_use.go` (const + `HandleFunc` signature + ownership validation L51-70 + broadcast L108-110); `effects.go` (L1-39 `AnnounceForeignSkillUse`); `main.go` `produceHandlers` (~L788) and `produceWriters` (~L683).

- [ ] **Step 1: broadcast helpers.** In `effects.go`, add `AnnounceForeignSkillPrepare(l)(ctx)(wp)(characterId uint32, info SkillPrepareInfo) Operator[session.Model]` → `session.Announce(...)(charcb.CharacterSkillPrepareForeignWriter)(charpkt.CharacterSkillPrepareForeignBody(characterId, info))`, and `AnnounceForeignSkillCancel(...)(characterId, skillId)` similarly. (No self helper — design D1.)
- [ ] **Step 2 (RED): handler tests.** Add `character_skill_prepare_test.go` asserting: (a) a keydown skill the caster owns → broadcast invoked to other map sessions; (b) non-keydown skill → dropped, no broadcast; (c) unowned/level-0 skill → dropped, **session NOT destroyed** (design D3). Use Builder-pattern fakes for character/session/map processors. Run → FAIL (handlers absent).
- [ ] **Step 3 (GREEN): handlers.** Implement `CharacterSkillPrepareHandleFunc` (decode `SkillPrepareInfo`, look up character w/ skill decorator, gate on ownership-level>0 AND `skill.IsKeyDownSkill(id)` via `shouldBroadcastKeydown`, on miss `l.Debugf`+return (no Destroy), else `ForOtherSessionsInMap(... AnnounceForeignSkillPrepare ...)`). Define const `CharacterSkillPrepareHandle`. **Cancel is NOT a standalone handler (D10):** the keyup uses the same serverbound opcode as buff-cancel, so EXTEND `socket/handler/character_buff_cancel.go` — keep `buff.Cancel(...)` unconditional, then gate on `IsKeyDownSkill` first + ownership and broadcast `AnnounceForeignSkillCancel`. Run tests → PASS.
- [ ] **Step 4: register.** `main.go produceHandlers()`: add `handlerMap[handler.CharacterSkillPrepareHandle] = handler.CharacterSkillPrepareHandleFunc` (prepare only — cancel rides the existing `CharacterBuffCancel` registration). `produceWriters()`: append `charcb.CharacterSkillPrepareForeignWriter` and `charcb.CharacterSkillCancelForeignWriter`.
- [ ] **Step 5: gates + commit.** `cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...`; commit `feat(atlas-channel): skill prepare/cancel handlers + foreign broadcast`. Verify branch.

---

## Task 5: Seed template opcode wiring (5 versions)

**Files:** `services/atlas-configurations/seed-data/templates/template_{gms_83_1,gms_84_1,gms_87_1,gms_95_1,jms_185_1}.json`. Anchors: existing `socket.handlers[]` entry `{opCode, validator, handler}` and `socket.writers[]` entry `{opCode, writer}`.

- [ ] **Step 1: add rows per version** using the **opcodes from `wire-spec.md`** (per-version — they differ): ONE `socket.handlers` row (`CharacterSkillPrepareHandle`, serverbound prepare opcode) with `"validator": "LoggedInValidator"` (a missing validator silently drops the handler), and two `socket.writers` rows (`CharacterSkillPrepareForeign` clientbound-prepare opcode, `CharacterSkillCancelForeign` clientbound-cancel opcode). **Do NOT add a serverbound cancel handler row (D10):** the keyup opcode is already mapped to `CharacterBuffCancel` (which Task 4 extended) — adding a second handler at that opcode would collide. Verify in each template that the prepare opcode is free and the cancel opcode already maps to `CharacterBuffCancel`.
- [ ] **Step 2: validate JSON + opcode uniqueness.** Each template parses (`jq . <file>`), and the new opcodes don't collide with an existing handler/writer opcode in that template. Report any collision rather than overwriting.
- [ ] **Step 3: commit** `chore(atlas-configurations): seed skill prepare/cancel opcodes for v83/84/87/95/jms185`.

---

## Task 6: Promote the coverage matrix + audit docs

**Files:** `docs/packets/audits/STATUS.md`, `docs/packets/registry/*.yaml` (verification status), per-version audit docs under `docs/packets/audits/<ver>/`. Follow `docs/packets/audits/VERIFYING_A_PACKET.md`.

- [ ] **Step 1: pin evidence + promote** the four ops (serverbound prepare, serverbound cancel, clientbound remote-prepare, clientbound remote-cancel) × five versions in STATUS.md, each backed by the byte-fixture test added in Tasks 2/3 (the `packet-audit:verify` marker links the cell to its test). Update/author the per-version audit docs for these ops; reconcile registry `fname`/opcode where IDB disagreed (Task 1 Step 1).
- [ ] **Step 2: verify the matrix** regenerates/validates per the playbook; no cell promoted without a test. Commit `docs(packets): verify skill prepare/cancel for v83/84/87/95/jms185`.

---

## Task 7: Full verification sweep + handoff notes

**Files:** none (verification) except appending operational follow-ups to the task folder.

- [ ] **Step 1: module gates.** `libs/atlas-packet` and `services/atlas-channel/atlas.com/channel`: `go test -race ./...`, `go vet ./...`, `go build ./...` all clean.
- [ ] **Step 2: repo gates.** `tools/redis-key-guard.sh` (GOWORK=off) introduces no new findings; all five seed templates parse (`jq`).
- [ ] **Step 3: scope guards.** `grep -rn` confirms the attack writer `isKeydownSkill` (`socket/writer/character_attack_common.go`) is UNCHANGED (design D9); confirm no v92 template/opcode was added.
- [ ] **Step 4: diff review.** `git diff main...HEAD --stat` shows only the intended files (codecs, handler, effects, main.go, 5 templates, packet docs, task docs). No stray edits.
- [ ] **Step 5: document operational + manual-validation follow-ups** in the task folder (not code): (a) **live tenant config patch** for each existing tenant + channel restart (handlers/writers don't hot-reload; FR-7.2); (b) **in-map manual validation** (two chars, observer sees Hurricane aura start on keydown and stop on keyup; spot-check Monster Magnet / BigBang / Rapid Fire); (c) the **death/stun-while-in-map** check (D6) — confirm no stuck aura, add server synthesis only if observed; (d) **v92 parked** follow-up note. Commit `docs(task-099): operational + manual-validation follow-ups`.

---

## Self-Review

**Spec coverage (PRD FRs + design decisions):**

| Requirement | Task |
|---|---|
| FR-1.1/1.2 serverbound prepare handler + decode | Task 1 (wire), Task 2 (codec), Task 4 (handler) |
| FR-1.3 ownership validation | Task 4 Step 3 |
| FR-1.4 gate on `IsKeyDownSkill` | Task 4 Step 3 (D4) |
| FR-2.1/2.2/2.3 clientbound remote-prepare broadcast | Task 3 (codec), Task 4 (helper+broadcast) |
| FR-3.1/3.2/3.3 serverbound cancel handler + decode | Task 1, Task 2, Task 4 |
| FR-4.1/4.2 clientbound remote-cancel broadcast | Task 3, Task 4 |
| FR-4.3 prepare/cancel pairing (no stuck aura) | Task 4 (both handlers shipped together); D6 |
| FR-5.1 termination | D6: keyup relay (Task 4); disconnect/leave = avatar removal (no code); death/stun check Task 7 Step 5 |
| FR-6.1 five versions (v92 deferred) | Tasks 1–6 across v83/84/87/95/jms185 |
| FR-6.2 IDB-verified opcodes/read-orders | Task 1 |
| FR-6.3 matrix promotion + byte fixtures | Tasks 2/3 (fixtures), Task 6 (promotion) |
| FR-7.1 seed templates | Task 5 |
| FR-7.2 live config patch | Task 7 Step 5 (operational) |
| D1 foreign-only writers | Task 3/4 (no self helper) |
| D2 dedicated writers | Task 3 |
| D3 drop-not-destroy | Task 4 Step 2/3 |
| D5 version-conditional codec | Tasks 2/3 |
| D8 MovingShoot conditional | Task 1 Step 2 |
| D9 attack writer untouched | Task 7 Step 3 |
| Verification (test/vet/build/redis-guard) | Task 7 |

**Placeholder scan:** no TBD/TODO/"similar to" — every code step names exact anchors (file + symbol/line) and the data source (`wire-spec.md`). Opcodes/read-orders are intentionally not hardcoded in the plan because Task 1 pins them; this is a dependency, not a placeholder.

**Identifier consistency:** `SkillPrepareInfo`, `SkillCancelInfo`, `CharacterSkillPrepareForeign(Writer)`, `CharacterSkillCancelForeign(Writer)`, `CharacterSkillPrepareForeignBody`, `CharacterSkillCancelForeignBody`, `CharacterSkillPrepareHandle(Func)`, `CharacterSkillCancelHandle(Func)`, `AnnounceForeignSkillPrepare`, `AnnounceForeignSkillCancel` — used consistently across Tasks 2–6.

**Ordering note:** Task 1 is a hard gate — Tasks 2/3/5/6 consume `wire-spec.md` opcodes/orders and must not start until it's committed. Tasks 2 and 3 are independent (can run in either order); Task 4 depends on both; Task 5 depends on Task 1 (opcodes) + Task 3/4 (writer/handler names); Task 6 depends on the byte-fixtures from Tasks 2/3.
