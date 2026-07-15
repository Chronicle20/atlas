# v61 Stage E — close reconciliation

v61 Stage E advance gate MET. `matrix --check` exit 0 (0 orphan/dangling/stale/drift, 0 conflicts); existing baseline versions frozen (v83 367/v84 345/v87 379/v95 399/jms 361→362 coverage-bonus, none dropped); v61 verified = 208. All producible in-scope cells fixtured or version-gated.

## Residual in-scope matrix-❌ (5) — 0 producible gaps
| cell | reason (documented) |
|---|---|
| guild/serverbound/GuildJoin (GUILD_OPERATION) | all sub-arms verified; op-row can't flip due to the matrix `lookupAnyVersion` jms-fold tooling bug (needs a +98-row matrix restructure — separate tool task, flagged B2) |
| guild/serverbound/GuildBBSListThreads (BBS_OPERATION) | same — 6 BBS sub-op decoders verified; op-row stuck on the jms-fold tooling bug |
| cash/serverbound/CashShopOperationGetPurchaseRecord (CASHSHOP_OPERATION) | v61-absent (mode 0x27 is a name-entry dialog; exhaustive send-site scan). Documented in `_unimplemented.json`; the tooling models CASHSHOP_OPERATION sb as single-writer so a present-op/absent-arm can't render n-a |
| storage/clientbound/StorageErrorMessage (STORAGE) | WZ-blocked: v61 storage has no discrete inventory-full mode, only generic error notices (mode 6→SP3602, 7→3601); identifying which needs String.wz (unavailable this session — same deferral class as runtime WZ) |
| messenger/clientbound/MessengerAdd (MESSENGER) | MessengerAdd body fixtured; op-row grades worst-of-siblings and JOIN/INVITE_SENT fall to a runtime virtual-dispatch default (dword_975D08 shared active-dialog ptr) not statically pinnable |

## Campaign outcome
All 205 in-scope cells fixtured, version-gated, or documented. Notable v61 work: **systematic serverbound opcode scramble fixed** — Stage B's blind harvest mislabeled 9+ CWvsContext-region serverbound opcodes (DISTRIBUTE_AP=80, DISTRIBUTE_SP=82, MESO_DROP=86, GIVE_FAME=87, CHAR_INFO_REQUEST=89, TROCK_ADD_MAP=94, ANTI_MACRO=96, ITEM_SORT=64, ITEM_SORT2=65, ITEM_CANCEL=68, NPC_TALK=54, the 243↔244 MESSENGER/PLAYER_INTERACTION swap) — all re-derived by body-signature + registry+template corrected (production routing fixes). status-message shrinks to arms 0–9 (no SP). AUTO_DISTRIBUTE_AP + several minigame/PIC/blacklist ops IDA-confirmed v61-absent.

## Cross-version follow-ups surfaced (non-blocking, for Phase 5 / reviews)
- **NPC_TALK (NpcStartConversation) serverbound false-pass in v72 (and likely v79)**: the v61 pass found the client sends oid+x+y, but the shipped v72 fixture is oid-only (stale marker ida=0x70dd49). v72/v79 left untouched per scope; needs a cross-version re-verify (`startConversationHasXY`).
- guild BBS/GUILD_OPERATION op-row jms-fold: a matrix `lookupAnyVersion` restructure would let these render verified.
