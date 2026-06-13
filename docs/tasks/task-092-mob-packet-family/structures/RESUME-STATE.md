# task-092 — Resume state (after Stage 0 / 0.5 / 1 / 1.5)

Last updated 2026-06-13. Foundation is built and committed; **Stage 2 (codecs) not started.**

## Done & committed (on `task-092-mob-packet-family`)
- **Stage 0** recon: baseline matrix clean; export-gap + registry-gap audits.
- **Stage 0.5 tooling** (`ca54ef850`): `packet-audit export` gained `--prior-export`/`--pending` (targeted harvest); `idasrc.GetFunctionByName` now bridges demangled→mangled via `func_query` fallback. Tested + live-verified.
- **Stage 1** (`551b00199`→`69b66a35c`): harvested byte layouts into `structures/<vk>.md`, refreshed all 5 IDA exports (absent-keys-only merge, no drift), fixed registry fnames.
- **Stage 1.5** IDB naming (`f7a2b4a8d`,`a2fe44636`,`7b8d7f0fe`,`e8def21fe`,`205f6974c`,`842103bff`): named ~24 unnamed mob functions (v84 22 + v83/jms TryDoingBodyAttack) by layout-match to v83 twins; re-harvested.
- `matrix --check` exit 0 throughout. The IDBs themselves were modified (renames persist in the user's .i64 files).

## Coverage now
- **v95**: full. **v87/v83/jms**: most ops pinnable. **v84**: 22 handlers named + pinnable.

## Residual blockers (small set — serverbound senders, inlined or undecompilable)
Decide per cell at Stage 2 (codec+test+route still land; only the evidence pin is affected):
- **MOB_BANISH_PLAYER** (`CUserLocal::SendBanMapByMobRequest`) — INLINED into `CUserLocal::Update`; no standalone function to pin (v83; check other versions).
- **MOB_TIME_BOMB_END** (`CMob::UpdateTimeBomb`) — INLINED into `CMob::Update` (v83, v87, v84).
- **v84 unnamed sender cluster**: MONSTER_BOMB (`TryFirstSelfDestruction`), MOB_DROP_PICKUP_REQUEST (`SendDropPickUpRequest`), TOUCH_MONSTER_ATTACK (`TryDoingBodyAttack`), MONSTER_BOOK_COVER (`SetMonsterBookCover`) — unnamed, no anchor symbols; not layout-matchable without building the namespace.
- **jms TOUCH_MONSTER_ATTACK** — symbol named at `0xA2AB71` but jms Hex-Rays FAILS to decompile it; resolves by address for manual evidence; layout inherits v83/v95.
- Options for these: (a) pin the parent `Update` function for inlined sends with a call-site note; (b) hand-author the byte-test from the sibling layout + a VERSION-equivalent evidence note; (c) leave those specific cells `partial` with justification.

## Confirmed VERSION-ABSENT (legitimate n/a, no pinning needed)
- v83 MOB_SKILL_DELAY (cb) — v95 first adds it (case 303). v84 HAS it (case 261, Decode4×4) — a real v84≠v83 delta; the codec must gate it `MajorAtLeast(84)` for that op (NOT the usual ≥87).
- v87 MOB_ESCORT_COLLISION + escort family — absent in v87.
- jms MOB_SPEAKING/INC_MOB_CHARGE_COUNT/MOB_SKILL_DELAY — absent.
- (See `structures/applicability.md` for the authoritative grid.)

## Stage 2 entry notes (from harvest)
- Serverbound send-side functions captured ~0 Decode calls (harvester is Decode-focused) → hand-trace Encode order from the decompile / COutPacket build sites for serverbound codecs.
- `CMob::Update` backs FIELD_DAMAGE_MOB / MOB_DAMAGE_MOB_FRIENDLY / MOB_SKILL_DELAY_END — derive per-op payloads from the COutPacket build sites, not the shared read-side.
- `CField_MonsterCarnival::OnRequestResult` demuxes SUMMON (arg≠0: Decode1,Decode1,DecodeStr) vs MESSAGE (arg=0: single Decode1; strings from StringPool).
- Carnival ops live under new pkg `monster/carnival/{clientbound,serverbound}` (keeps `monster/` tier-1 prefix).
- Gate rule: `MajorAtLeast(87)`, except the v84-only MOB_SKILL_DELAY delta noted above.

## Next: Stage 2 (plan.md Cluster order D→A→B→C→F→E), then Stage 3 docs, Stage 4 gates+review.
