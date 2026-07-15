# v72 Stage E — close reconciliation

v72 Stage E advance gate MET. `matrix --check` exit 0 (0 orphan/dangling/stale/drift, 0 conflicts); every existing version frozen (v83 367/v84 345/v87 379/v95 399/jms 361, identical to pass-start); v72 verified = 216.

## Residual in-scope matrix-❌ (3) — all justified-n-a, zero producible gaps
The matrix cannot render a `sub-struct` cell as n-a (op-cells only; known tooling limitation, same as v79). These 3 are documented v72-absent in `docs/packets/audits/gms_v72/_unimplemented.json` (6 entries), have no parent op-row, and are IDA-confirmed absent:

| sub-struct cell | v72-absence basis |
|---|---|
| cash/serverbound/CashShopOperationIncreaseCharacterSlot | v72 lacks the mode-9 send; the 0x468e43 "IncCharSlot" symbol is a mislabel for EnableEquipSlot (size+body confirmed) |
| interaction/serverbound/InteractionOperationMerchantAddToBlackList | CEntrustedShopDlg blacklist is post-v72 (exhaustive COutPacket(121) scan; v79 has it at opcode-120) |
| interaction/serverbound/InteractionOperationMerchantRemoveFromBlackList | same — post-v72 |

## Campaign outcome
All 214 in-scope tier-1/login cells fixtured or justified-n-a. Notable v72 divergences version-gated for the legacy range (v79/v83/84/87/95/jms byte-unchanged): status-message OnMessage shrinks to arms 0–11; DropPickUpMeso/IncreaseExperience trailing fields (<79); attack action byte (<79); DamageInfo CRC (>=72); CHANGE_MAP chase-byte (>=72); char-list family byte + CreateCharacter jobIndex (>=73 intra-legacy discriminator); monster stat-mask + MOVE_MONSTER/MOVE_LIFE field drops; summon skillCRC (>=79). Registry self-corrections: DeleteCharacter opcode 24 (not 23), NPC-shop senders located (sub_6A8B15/6A8D8F/6A8FB2). char-mgmt opcodes body-verified (rotated symbols, = v79/v83 by behavior).

## Known follow-up (non-blocking)
v72 `CharacterInteractionHandle` serverbound `operations` table is empty in the template (Stage C gap; mode bytes Δ-shifted from v83) — Stage F to populate from the v72 switch.
