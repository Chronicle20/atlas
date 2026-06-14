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

## Stage 2 progress

- **Cluster D** (CRC/misc, 4 ops) — committed `95178fbdf`.
- **Cluster A** (combat/damage, 10 ops) — committed `2db28f14c` (clientbound trio) +
  `b4394460e` (serverbound damage). 9 of 10 ops landed; matrix --check exit 0.
  - Clientbound: MOB_AFFECTED, MONSTER_SPECIAL_EFFECT_BY_SKILL (v95-only 3-field
    layout, region+major gate), RESET_MONSTER_ANIMATION — ✅ all 5.
  - Serverbound: FIELD_DAMAGE_MOB ✅5, MOB_DAMAGE_MOB ✅5, MOB_DAMAGE_MOB_FRIENDLY
    (reconciled to pre-existing character/MonsterDamageFriendly) ✅5, MONSTER_BOMB
    ✅4 (v84 sender unnamed), MOB_SKILL_DELAY_END ✅4 (v83 version-absent),
    MOB_TIME_BOMB_END ✅2 (v95/jms; v83/v84/v87 inlined into CMob::Update).
  - **2.A7 TOUCH_MONSTER_ATTACK — DEFERRED (not landed).** CUserLocal::TryDoingBodyAttack
    is a large, branch-heavy, version-DIVERGENT attack packet, NOT byte-plumbing:
    v83 (opcode 0x30 @0x9593f7) has two distinct serialization branches (touch vs
    body attack) with a per-hit detail loop; v95 (opcode 0x32 @0x931a6d) is a wholly
    different shape (field-key, _DR_INFO crypto-masked fields, GetCrc32 checksum,
    SKILLLEVELDATA, ATTACKINFO[15] hit loop). The two are not byte-compatible. jms
    TryDoingBodyAttack Hex-Rays decompile FAILS (per applicability.md fn9). A faithful
    5-version codec requires modeling the full attack/hit-detail structure — out of
    scope for this batch. Left as a follow-up; no codec/route landed (the opcode stays
    "unhandled" rather than shipping a knowingly-wrong codec). Registry rows untouched.

### v84/registry opcode corrections made in Cluster A (IDB-verified, were csv-stale)
- clientbound: MOB_AFFECTED 245→251/0xFB; MONSTER_SPECIAL_EFFECT_BY_SKILL 247→253/0xFD
  (CMobPool::OnMobPacket @0x68fef7 dispatcher cases).
- serverbound: FIELD_DAMAGE_MOB 191→196/0xC4; MOB_SKILL_DELAY_END 195→200/0xC8
  (CMob::Update @0x67dd33 / @0x67d534 COutPacket sites).
- v83 serverbound MOB_SKILL_DELAY_END row removed (version-absent).

- **Cluster E** (Monster Carnival, 9 ops) — new pkg `monster/carnival/{clientbound,serverbound}`.
  All 9 ✅ across all 5 versions; matrix --check exit 0. Tier-1 preserved (cells show T1 —
  `monster/` prefix matches via the nested pkg). carnivalcb/carnivalsb aliases added in main.go.
  - clientbound (8): MONSTER_CARNIVAL_START (OnEnter: Decode1 team + 6×Decode2 CP + per-slot
    Decode1 loop), OBTAINED_CP (OnPersonalCP: 2×Decode2), PARTY_CP (OnTeamCP: Decode1+2×Decode2),
    SUMMON + MESSAGE (OnRequestResult demux — **confirmed two DISTINCT shapes**: SUMMON arg≠0 =
    Decode1,Decode1,DecodeStr; MESSAGE arg=0 = single Decode1, strings from StringPool),
    DIED (OnProcessForDeath: Decode1,DecodeStr,Decode1), LEAVE (OnShowMemberOutMsg:
    Decode1,Decode1,DecodeStr), RESULT (OnShowGameResult: Decode1).
  - serverbound (1): MONSTER_CARNIVAL (RequestSend: Encode1 tab + Encode4 idx-1), LoggedInValidator.
  - All `CField_MonsterCarnival::On*` decompiled per version (v83/v95/jms full bodies; v87/jms/v84
    addresses + dispatcher OnPacket verified). Layouts byte-identical across all 5 versions.
  - **v84 opcode corrections (csv-stale → IDB-verified, the +7 cb / +6 sb v84 table shift):**
    cb START..RESULT 0x121-0x128 → 0x128-0x12F (296-303; CField_MonsterCarnival::OnPacket
    @0x571FF5: SUMMON=case 299 arg=1, MESSAGE=case 300 arg=0); sb MONSTER_CARNIVAL 0xDA → 0xE0
    (RequestSend @0x89bdda COutPacket(224)). Registry gms_v84.yaml rows updated (ida-discovered +
    ida.address). v83/v87/v95/jms opcodes confirmed unchanged against their dispatchers.

## STAGE 2 COMPLETE — all 6 clusters landed (D,A,B,C,F,E). matrix --check exit 0.
Commits: D `95178fbdf`; A `2db28f14c`+`b4394460e`; B/C `1107bbfde`; F `f1406b946`; E `593c8c0bb`+`ef695647b`.
41 of 42 ops have codec+wiring+test+routes; the vast majority of cells verified ✅.

### Residual cleanup for Stage 4 — STATUS (all IDA-verified this stage)
1. **TOUCH_MONSTER_ATTACK** — v83 wire layout DERIVED + documented
   (`structures/touch_monster_attack.md`); opcode off-by-one found (real v83 =
   `0x30`, not csv `0x2F`). Codec NOT landed: it is a heavily-conditional,
   version-divergent attack packet whose v95 variant carries crypto-masked
   `_DR_INFO` fields + GetCrc32 + an ATTACKINFO[15] hit loop — a faithful
   byte-exact 3-version codec is task-sized and cannot be verified to the
   no-knowingly-wrong-codec bar inside this stage. Recommended follow-up task,
   with v83 already derived as a head start.
2. **jms/v95 escort + SHOW_MAGNET dedupe — DONE** (commit, item 2): verified all
   5 OnMobPacket dispatchers; v83 off-by-one fixed; SHOW_MAGNET deleted; jms
   placeholders renamed to canonical. MOB_SPEAKING ✅×5; jms escort/skill-delay ✅.
3. **v87 escort VERSION-ABSENT — DONE** (item 3): dropped the v87 MOB_ESCORT_FULL_PATH
   + MOB_ESCORT_COLLISION rows (dispatcher ends at 0x110); cells now ⬜.
4. **Inlined/unnamed senders — MOSTLY DONE** (item 4): named the v84 senders in the
   IDB and pinned MONSTER_BOMB, MOB_DROP_PICKUP_REQUEST, MONSTER_BOOK_COVER (v84)
   and MOB_BANISH_PLAYER (v83+v84 — both DISCRETE, the "inlined" claim was wrong)
   → all ✅×5. Fixed the v84 DROP_PICKUP route opcode bug (0xBE→0xC3) and added the
   missing MonsterBomb route (0xC6). Only **MOB_TIME_BOMB_END v83/v84/v87** remains
   ❌ — no sender locatable in those clients (v84 exhaustive scan negative; no named
   TimeBomb fn in v83/v87); left honest rather than fabricated. v95/jms pinned.

### Remaining stages
- **Stage 3 — DONE.** `docs/packets/IMPLEMENTING_A_PACKET.md` (four-step recipe, worked MOB_CRC_KEY_CHANGED example, conventions: package-by-owner-class + tier-1 prefix, no-emitter seam, validator-mandatory/BuildHandlerMap silent-drop, `>83`→`MajorAtLeast(87)` gate, fname-mislabel guard, export-resolvability precondition; cross-links VERIFYING_A_PACKET/tiers.yaml/registry README) + `deploy-notes.md` (per-version writer+handler opcode tables from the template diff, PATCH shape, rollout checklist, post-deploy checks). Also removed a stray 40MB `atlas-channel` build binary left untracked in the worktree root.
- **Stage 4** — final gates (test/vet/build, redis-key-guard, matrix --check) + reconcile residual 1–4 above + `superpowers:requesting-code-review` (plan-adherence + backend-guidelines).
