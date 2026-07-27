# Skill Books and Mastery Books — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

MapleStory players raise a skill's *master level* (the cap its normal level can be trained to) by consuming Skill Books (item classification 228, 26 items in v83 `Consume/0228.img.xml`) and Mastery Books (classification 229, 139 items in `0229.img.xml`). Each book's WZ data declares which skills it applies to (`skills[]`), the master level it grants (`masterLevel`), the minimum current skill level required (`reqSkillLevel`), and a percent success chance (`success`). This is the core progression mechanic for 4th-job skills.

Atlas currently has no consumer for this data. The classification constants exist (`libs/atlas-constants/item/constants.go:41-42`), atlas-data already parses and serves `success`/`masterLevel`/`reqSkillLevel`/`skills[]` on the consumable resource (`services/atlas-data/atlas.com/data/consumable/reader.go:60,107-114`, `rest.go:56-101`), and atlas-skills already persists `masterLevel` per skill (`skill/entity.go:27`) — but the serverbound `USE_SKILL_BOOK` packet is unhandled and the clientbound `SKILL_LEARN_ITEM_RESULT` writer does not exist. STATUS.md rows 72 and 570 show ❌ for both packets across all versions. Using a book in-game is a silent no-op today.

This task implements the flow end-to-end: channel handler → consumables validation + success roll → saga-orchestrated item consumption and master-level update → result packet broadcast to the map.

## 2. Goals

Primary goals:
- A player can consume a Skill Book (228) or Mastery Book (229) and, on a successful roll, have the target skill's master level raised per the book's WZ data.
- The book is consumed on failure as well as success (owner decision, matches reference behavior).
- All eligibility rules are enforced server-side (see FR-2); the client-side gate is never trusted.
- The result is broadcast to the map: bystanders see the glow effect on the user; the user additionally gets the success/failure sound and chat message (client behavior, IDA-verified — see §5).
- Item consumption and skill update are atomic via atlas-saga-orchestrator.
- Both packets are wired for all supported tenant versions, and byte-fixture verified where an IDB exists (promoting the STATUS.md cells).

Non-goals:
- NPC-, quest-, or gachapon-granted mastery increases (only item-driven consumption).
- Big Bang-era book mechanics (e.g., ★ mastery book unification).
- atlas-ui changes.
- gms_12 tenant support (neither `USE_SKILL_BOOK` nor `SKILL_LEARN_ITEM_RESULT` exists in any v12 registry; the mechanic postdates that client).

## 3. User Stories

- As a player, I want to double-click a Mastery Book so that my skill's master level rises when the roll succeeds.
- As a player, I want a failed book to still show the failure effect and message so that I know the attempt happened and the book was used up.
- As a player who doesn't meet the book's requirements, I want a "You cannot use..." response so that my client isn't left in a wedged state.
- As a bystander on the map, I want to see the glow effect on the player using a book so that the world feels shared.
- As an operator, I want the flow to be atomic so that a crash between "book destroyed" and "master level raised" cannot eat a player's book.

## 4. Functional Requirements

### FR-1 — Serverbound handler (atlas-channel)

1. Handle `USE_SKILL_BOOK` (fname `CWvsContext::SendSkillLearnItemUseRequest`). Body, IDA-verified on v95 (`0x9d65e0`): `int32 updateTime`, `int16 slot`, `int32 itemId`. Per-version body assumed identical pending fixture verification.
2. Handler entry requires a validator (`LoggedInValidator`), per the silently-dropped-handler gotcha.
3. The handler forwards a consume request (field, characterId, itemId, slot, updateTime) to atlas-consumables via the existing command topic pattern; no game logic in the handler.

### FR-2 — Validation (atlas-consumables)

All checks run server-side against atlas-data consumable data, atlas-character, and atlas-skills state. A request failing any check MUST still produce a result packet to the requester with `canUse=0` (the client sets an exclusive-request lock when sending and only the result packet clears it — IDA-verified on v83).

1. **Item integrity:** the asset at `slot` in the USE compartment exists, its template id equals `itemId`, and quantity ≥ 1.
2. **Classification:** `itemId` classification is 228 or 229 (`item.GetClassification`).
3. **Alive:** character HP > 0.
4. **Job match:** the book's `skills[]` contains at least one skill whose job prefix matches the character's job. The first matching skill is the target skill.
5. **Skill state — mastery book (229):** the target skill is already learned (a skill record exists with level ≥ 1).
6. **reqSkillLevel:** current skill level ≥ the book's `reqSkillLevel`.
7. **Master-level ceiling:** current master level < the book's `masterLevel`.

### FR-3 — Success roll and consumption (atlas-consumables → saga)

1. On passing validation, roll success: uniform roll passes with probability `success`% (`success` is 0–100; verify exact interpretation against reference in design).
2. Construct and submit a saga:
   - Step 1: `destroy_asset_from_slot` (or equivalent existing action) — consume exactly 1 book from the slot. This step runs on **both** success and failure (consume-on-fail).
   - Step 2 (success roll only): `update_skill` (or `create_skill` if no record exists and the flow permits it — design decision D-1) setting `masterLevel` to the book's `masterLevel`, preserving current skill level.
3. On saga completion (or validation rejection), emit a status event carrying characterId, isMasteryBook (from classification), skillId, the granted masterLevel, canUse, and success — the inputs the writer needs.

### FR-4 — Clientbound result writer (atlas-channel)

1. Implement `SKILL_LEARN_ITEM_RESULT` (fname `CWvsContext::OnSkillLearnItemResult`). Body, IDA-verified decode order on v83 (`0xa1e5af`): `int32 characterId`, `byte isMasteryBook`, `int32 skillId`, `int32 masterLevel`, `byte canUse`, `byte success`. (Third/fourth ints are decoded-then-discarded by the v83 client; field naming per reference semantics, to be pinned per-version during fixture work.)
2. **Broadcast to the map** (IDA-verified: the client resolves `characterId` via `CUserPool::GetUser` and renders the glow effect on that avatar for any observer; sound + chat message only fire behind an is-local-user check). Validation rejections (`canUse=0`) MAY be sent to the requester only — design decision D-2.
3. Opcode and any mode bytes resolve from tenant config; no hard-coded opcodes (per the msgType-hardcoding gotcha).

### FR-5 — Version wiring

1. Add the handler + writer entries to seed templates: `gms_83`, `gms_84`, `gms_87`, `gms_92`, `gms_95`, `jms_185` (opcodes from the corresponding registries; gms_92 from its template lineage since no v92 registry/IDB exists).
2. Patch live tenant configurations for existing tenants and note the channel restart requirement (new-opcodes-not-in-live-config gotcha).

### FR-6 — Byte fixtures (packet verification)

1. For each version with an available IDB: byte-fixture tests with `packet-audit:verify` markers for both packets, evidence pinned, matrix regenerated — promoting STATUS.md rows 72 and 570 cells to ✅ per `docs/packets/audits/VERIFYING_A_PACKET.md`.
2. Versions without an IDB (gms_92 today) follow the mount-food precedent: implement from template/registry, leave the cell unverified, document why.

## 5. API Surface

No new REST endpoints. All new surface is Kafka + packets:

- **atlas-consumables:** new command type on its existing consumable command topic (e.g., `REQUEST_SKILL_BOOK_USE`) carrying field, characterId, itemId, slot, updateTime.
- **atlas-consumables → saga-orchestrator:** existing saga submission path; reuses `destroy_asset_from_slot` and `update_skill`/`create_skill` actions (`libs/atlas-saga/model.go:51,78-79`).
- **atlas-skills:** existing `REQUEST_UPDATE` command already carries `MasterLevel` (`kafka/message/skill/kafka.go:33-38`); expected to need no schema change (verify the update path doesn't clobber level/expiration — design item).
- **status event → atlas-channel:** new event type (e.g., `SKILL_BOOK_RESULT`) consumed by atlas-channel to drive the writer.
- **Packets:** serverbound `USE_SKILL_BOOK` opcodes 0x52/0x52/0x55/0x58/0x4A (v83/v84/v87/v95/jms); clientbound `SKILL_LEARN_ITEM_RESULT` 0x33/0x33/0x33/0x32/0x30 (STATUS.md:72,570).

## 6. Data Model

No migrations expected:

- atlas-skills `skill` entity already has `MasterLevel byte` (`skill/entity.go:27`), tenant-scoped.
- atlas-data consumable resource already serves `success`, `masterLevel`, `reqSkillLevel`, `skills[]`.
- Saga state persistence is unchanged (existing actions).

## 7. Service Impact

| Service | Change |
|---|---|
| atlas-channel | New serverbound handler (`character_skill_book_use.go` or similar), new clientbound writer + map-broadcast consumer for the result event. |
| atlas-consumables | New processor entry point (pattern: dedicated `RequestFeed`-style method, since the client uses a distinct opcode), validation per FR-2, success roll, saga construction, result-event emission. |
| atlas-skills | Likely none beyond verifying `REQUEST_UPDATE` semantics for master-level-only updates. |
| atlas-saga-orchestrator | Likely none (existing actions); verify `update_skill` step wiring covers this shape. |
| atlas-configurations | Seed template updates for 6 version templates. |
| atlas-data | None expected (data already exposed); confirm no gaps for 228 vs 229 parsing. |
| libs/atlas-packet | Byte-fixture tests + evidence records for both packets per version. |

## 8. Non-Functional Requirements

- **Multi-tenancy:** all lookups and events tenant-scoped via `tenant.MustFromContext`; opcodes resolved from tenant config.
- **Atomicity:** book destruction and master-level update ride one saga; compensation restores the book if the skill update fails.
- **Idempotency/race:** duplicate `USE_SKILL_BOOK` packets (client excl-lock normally prevents this, but don't trust it) must not double-consume — the slot reserve/consume path must be safe under retry.
- **Observability:** validation rejections logged at warn with characterId/itemId/reason; success/failure rolls at info.
- **No trust in client gate:** the client only sends for classification 228/`is_masterybook_item` with HP>0 and a 200ms throttle (IDA v95); server re-validates everything.

## 9. Open Questions

- **D-1 — Skill-book (228) create-vs-update:** may a 228 book teach a skill the character has no record for (create at level 0 with the book's master level), or must a record already exist for both classifications? Reference behavior (Cosmic `UseSkillBookHandler`) to be read during design; do not assume from memory.
- **D-2 — Rejection routing:** `canUse=0` results — requester-only or broadcast like success/failure results? (Client renders the effect only when `canUse=1`, so broadcast is harmless but noisy; requester-only is cleaner. IDA shows the message is local-only either way.)
- **D-3 — `success` field semantics:** confirm percent-out-of-100 and whether any book has `success=0`/missing (reader defaults?) during design.
- **D-4 — Result packet third/fourth ints:** v83 discards both decoded ints; per-version fixture work must pin what each version actually reads (v95 `OnSkillLearnItemResult` is 0x456 bytes — decompile during fixture phase).
- **IDB availability:** current IDA instance set (2026-07-02) is v48/v61/v72/v79/v95/v83-dump — no v84/v87/jms loaded right now. Fixture cells for those versions depend on reloading those IDBs (they exist per prior tasks).

## 10. Acceptance Criteria

- [ ] Using a valid Mastery Book on a v83 tenant raises the skill's master level on success, consumes the book in both outcomes, and shows the map-broadcast effect + local message.
- [ ] Using a book while failing any FR-2 check consumes nothing, sends `canUse=0` to the requester, and the client is not wedged (excl lock cleared).
- [ ] Saga compensation verified: forced failure of the skill-update step restores the book.
- [ ] Handler + writer entries present in all six seed templates with validators; live tenant configs patched.
- [ ] Byte fixtures + evidence promote `USE_SKILL_BOOK` and `SKILL_LEARN_ITEM_RESULT` STATUS.md cells to ✅ for every version with an IDB; gms_92 exception documented.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake` for every touched service; `tools/redis-key-guard.sh` clean.
- [ ] Code review (plan-adherence + backend-guidelines) run before PR.
