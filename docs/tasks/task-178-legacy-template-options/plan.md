# task-178 — Legacy template `options` remediation

## Problem
PR #971 (GMS legacy versions 48.1/61.1/72.1/79.1) routed the socket handlers but
left many handlers WITHOUT their `options` block. atlas-channel handlers that
resolve a serverbound mode byte from `readerOptions["operations"]` (and the
movement decoder from `readerOptions["types"]`) have **no fallback**:
- operations handlers → every branch logs `"Code [X] not configured for use."` and
  silently drops the packet (see `buddy_operation.go` `isBuddyOperation`).
- movement handlers → `movementPathAttrFromOptions` defaults attr→99, "which will
  likely cause a client crash."

So buddy / guild / guild-BBS / messenger / note / storage / NPC-shop / cash-shop
and pet/monster/NPC movement are all dead (or crash-prone) on v48/61/72/79.

## Missing handlers per version (matched by handler name vs v83 which has options)
- v79/v72/v61 (11 each): BuddyOperationHandle, GuildBBSHandle, NPCActionHandle,
  CashShopOperationHandle, MessengerOperationHandle, GuildOperationHandle,
  NPCShopHandle, StorageOperationHandle, NoteOperationHandle, PetMovementHandle,
  MonsterMovementHandle.
- v48 (9): same minus GuildBBSHandle and NPCActionHandle (not routed in v48).

## Two option categories

### A. `types` movement tables — NO RE (copy within version)
`movementPathAttrFromOptions` (libs/atlas-packet/model/movement.go:284) reads one
shared `types` list for ALL movement handlers. In v83/v84 the Pet/Monster/NPCAction
`types` are byte-identical to that version's `CharacterMoveHandle.types`. Every
legacy template already has a verified `CharacterMoveHandle.types` (23 entries).
**Resolution: copy each version's own `CharacterMoveHandle.options.types` into its
NPCAction/Pet/Monster handlers.** (v48 has no NPCActionHandle → only Pet + Monster.)

### B. `operations` serverbound mode tables — RE per legacy IDB
8 families. Recovered from the client send functions (`COutPacket(<opcode>)` →
leading `Encode1(<mode>)`) in each legacy IDB. See `re-v79.md`, `re-v72.md`,
`re-v61.md`, `re-v48.md`. v83 ground truth in `re-reference.md`.

Key finding (v79): Buddy, GuildBBS, NPCShop, Storage, Messenger, Guild are
byte-identical to v83; Note DISCARD/REQUEST match (SEND=0 UI-gated, parity);
**CashShop is heavily renumbered — must NOT be copied from v83.**

## Live tenants to patch (region GMS, ns atlas-main, single atlas-channel replica)
- v48 `e1f06ae2-80c1-47f7-bb6f-38a9f50d23dd`
- v61 `0d250dc9-64c4-45ae-8bc2-fc0a9cdb5578`
- v72 `48d415ca-59de-4953-9aed-0c4156a09bc9`
- v79 `92adbe47-5ada-4f3b-8224-f58c80a4a2d5`

## Execution steps
1. RE all 4 legacy IDBs (parallel, session-pinned): v48=ef9c0dd8, v61=9a1bdd7a,
   v72=eb2a156e, v79=88dfa464. → re-vNN.md + JSON.
2. Populate the 4 seed templates (worktree) with the derived `operations` + copied
   `types`. Validate JSON round-trips; run configs service build/tests.
3. Live-patch: for each tenant, GET live config from atlas-configurations
   (`curl --resolve dev.atlas.home:80:192.168.23.230
   http://dev.atlas.home/api/configurations/tenants/{id}`), inject `options` into
   the matching socket handlers BY HANDLER NAME (opcode-independent; preserves any
   live opcode drift), PATCH back full config (JSON:API envelope,
   `Content-Type: application/vnd.api+json`).
4. `kubectl -n atlas-main rollout restart deployment/atlas-channel`; confirm
   `Configuring opcode ... handler ...` + no "not configured for use" at startup.
5. Code review → PR from task-178 branch.

## Outcome (live patch)
- **Templates**: all 4 populated + committed; correctness-reviewed (values vs RE
  evidence, no dup modes, movement types, clean diff) — PASS.
- **Live v72 `48d415ca` + v79 `92adbe47`**: PATCH 200, atlas-channel restarted
  clean (750 opcodes configured, zero "not configured for use"). DONE + verified.
- **Live v48 `e1f06ae2` + v61 `0d250dc9`: RESOLVED** (full-replace PATCH validates the
  whole tenant, so pre-existing invalid presets had to be fixed first). The legacy
  bring-up seeded modern-content presets into pre-modern clients; corrected per
  version and re-PATCHed (200):
  - Pirate presets (Buccaneer 512 / Corsair 522) removed from both (Pirates shipped v62).
  - v48 (4th job shipped v49; v48 predates it): 10 explorer 4th-job presets downgraded
    to 3rd job (jobId x12/x22/x32 → x11/x21/x31, skills/name/tags/description updated).
  - v61: over-level 4th-job skills clamped to atlas-data maxLevel (30→20, 5→1).
  - both: equipment templateIds absent from that version's atlas-data stripped.
  - **v48 is a parked tenant** atlas-channel does not currently serve (like v92), so
    its handlers aren't instantiated at runtime — but its config is now correct.
    v61 IS served and verified working.

## Final live state (atlas-channel restarted clean, 0 "not configured for use")
- Served + verified: **v61, v72, v79** (+ v83/84/87/95/jms unaffected).
- Config-correct but parked (not served): **v48** (pre-existing).

## Open decisions (resolve at population)
- v79 CashShop `ENABLE_EQUIP_SLOT` sends 6 OR 7 (7 for 9110xxx items) — one atlas
  key → two modes. Decide mapping vs handler semantics; a dropped mode-7 is a minor
  gap. Some CashShop keys (MOVE_*_CASH_INVENTORY, REBATE_LOCKER, APPLY_WISHLIST,
  BUY_NAME_CHANGE, INCREASE_INVENTORY/STORAGE) were UNRESOLVED/ABSENT on v79 —
  include only evidence-verified keys; omit truly-absent (feature can't be sent).
- Note.SEND=0 parity-inferred (DISCARD=1/REQUEST=2 verified) — flag in PR.
