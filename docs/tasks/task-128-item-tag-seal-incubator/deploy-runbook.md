# task-128 deploy runbook — live tenant config patch

Seed templates only apply at tenant creation. For every EXISTING tenant:

1. PATCH the tenant's socket configuration:
   - Append to `socket.writers`: `{"opCode": "<version's opcode>", "writer": "IncubatorResult"}`
     (gms_83: 0x45, gms_84: 0x47, gms_87: 0x47, gms_95: 0x48, jms_185: 0x3F; SKIP gms_92 — no verified opcode, see context.md).
   - Where missing, append to `socket.handlers`:
     `{"opCode": "<version's opcode>", "validator": "LoggedInValidator", "handler": "CharacterCashItemUseHandle"}`
     (gms_87: 0x52, gms_95: 0x55, jms_185: 0x47; gms_83/84 already have it at 0x4F).
2. Seed the reward pool per tenant: `POST /api/tenants/{tenantId}/configurations/incubator-rewards/seed`.
3. Restart atlas-channel (writers/handlers do not hot-reload).
4. Smoke-test on a v83 tenant: tag an equip (name appears in tooltip, survives relog),
   seal an equip (lock icon + timer for timed variants), incubate an item (hatch dialog).

## Known issues / caveats

1. **INCUBATOR_RESULT body shape is version-specific (handled in code, no config
   impact).** The writer emits a SHORT body (int itemId + short count) for
   gms_83/84/87 and jms_185, and an EXTENDED body (+3 trailing zero ints) ONLY
   for gms_95. This is writer-code behavior selected internally per tenant
   version — it requires no tenant-config change and no action from this
   runbook. Documented here for maintainers debugging a byte-level mismatch on
   the client side.

2. **PRE-EXISTING, out of scope for task-128 — needs its own follow-up task.**
   The outer `cash/serverbound/ItemUse` codec gates the `updateTime` field to
   `GMS && MajorVersion >= 95`, but live IDA disassembly shows the real
   v87/v95/jms clients write `updateTime` unconditionally on the cash-item-use
   request. This `updateTimeFirst` gating predates task-128 and affects ALL
   cash-item sub-bodies (PetConsumable, Chalkboard, FieldEffect), not just the
   tag/seal/incubator arms added here. If v87 or jms cash-item-use sub-body
   parsing misbehaves in the field (including this feature's arms), this
   gating is the most likely root cause and should be investigated and fixed
   as a separate task rather than patched inline here.
