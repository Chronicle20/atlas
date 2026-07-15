# task-128 deploy runbook — live tenant config patch

Seed templates only apply at tenant creation. For every EXISTING tenant:

1. PATCH the tenant's socket configuration:
   - Append to `socket.writers`: `{"opCode": "<version's opcode>", "writer": "IncubatorResult"}`
     (gms_83: 0x45, gms_84: 0x47, gms_87: 0x47, gms_95: 0x48, jms_185: 0x3F;
     **legacy line (added when v48–v79 client columns were merged in): gms_61: 0x42, gms_72: 0x42, gms_79: 0x42** — IDA-verified
     `int itemId + short count`, byte-identical to v83 (`CWvsContext::OnIncubatorResult` @0x8490d7/0x9203de/0x9722d8);
     SKIP gms_92 — no verified opcode, see context.md; SKIP gms_48 — see caveat 3 below).
   - Where missing, append to `socket.handlers`:
     `{"opCode": "<version's opcode>", "validator": "LoggedInValidator", "handler": "CharacterCashItemUseHandle"}`
     (gms_87: 0x52, gms_95: 0x55, jms_185: 0x47; gms_83/84 already have it at 0x4F;
     gms_48/61/72/79 already carry `CharacterCashItemUseHandle` from their own bring-up — no handler patch needed there).
2. Seed the reward pool per tenant: `POST /api/tenants/{tenantId}/configurations/incubator-rewards/seed`.
3. Restart atlas-channel (writers/handlers do not hot-reload).
4. Smoke-test on a v83 tenant: tag an equip (name appears in tooltip, survives relog),
   seal an equip (lock icon + timer for timed variants), incubate an item (hatch dialog).

## Known issues / caveats

1. **INCUBATOR_RESULT body shape is version-specific (handled in code, no config
   impact).** The writer emits a SHORT body (int itemId + short count) for
   gms_61/72/79/83/84/87 and jms_185, and an EXTENDED body (+3 trailing zero
   ints) ONLY for gms_95. This is writer-code behavior selected internally per
   tenant version — it requires no tenant-config change and no action from this
   runbook. Documented here for maintainers debugging a byte-level mismatch on
   the client side. The legacy gms_61/72/79 cells are matrix-verified
   (`docs/packets/audits/status.json`) with pinned evidence + byte fixtures.

2. **FIXED in this branch (no config impact) — cash-item-use `updateTime` is a
   leading header field from GMS v87 onward.** The outer `cash/serverbound/ItemUse`
   codec previously gated the `updateTime` field to `GMS && MajorVersion >= 95`,
   but live IDA disassembly shows the real v87/v95/jms clients write `updateTime`
   unconditionally at the FRONT of the cash-item-use request
   (`CWvsContext::SendConsumeCashItemUseRequest`: gms_v87 @0xa9fef9, gms_v95
   @0x9eb3e0, jms_v185 @0xaef2f5 all `Encode4(update_time)` before the sub-body
   switch; only the two oldest builds gms_v83 @0xa0a63f / gms_v84 carry it as a
   trailing int32). The gate is now `MajorVersion() >= 87` in both
   `libs/atlas-packet/cash/serverbound/item_use.go` (the `ItemUse` header codec)
   and `character_cash_item_use.go`'s `updateTimeFirst` dispatch flag — the two
   MUST stay in lockstep. This is version-family-wide: it governs ALL cash-item
   sub-bodies (PetConsumable, Chalkboard, FieldEffect, ItemTag, Seal, Incubator),
   not just the tag/seal/incubator arms added here. `main` landed the identical
   fix independently, so the merge converged on it. If any v87/v95/jms cash-item
   sub-body ever misparses in the field, this header gate is the first place to
   check.

3. **gms_48 incubator is NOT supported by the flat writer — do NOT wire
   `IncubatorResult` into a gms_48 tenant.** Unlike every other supported version
   (whose `CWvsContext::OnIncubatorResult` reads a flat `int itemId + short
   count` body), the gms_48 client's `OnIncubatorResult` (@0x71f72a, opcode
   0x2A) is a **mode-prefix dispatcher**: it switches on `Decode1() - 6` and each
   mode reads a distinct body (channel-name notices, chatlog strings, a trailing
   `DecodeStr`), and the reward-granted arm reads NOTHING from the packet.
   Sending the flat `IncubatorResult` writer to a gms_48 client would be
   misparsed (the itemId's first byte is read as the mode selector). Proper gms_48
   support requires a dedicated mode-dispatched writer (a `dispatcher-family`
   task per `docs/packets/DISPATCHER_FAMILY.md`), not this codec. Until then the
   gms_48 INCUBATOR_RESULT matrix cell is intentionally left `incomplete` and the
   gms_48 seed template carries no `IncubatorResult` writer. The item-tag and
   sealing-lock arms are unaffected — they surface through the ordinary inventory
   update writers (already present in the gms_48 template), not through
   `IncubatorResult`.
