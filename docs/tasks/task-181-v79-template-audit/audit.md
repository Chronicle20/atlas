# task-181 — GMS v79 handler/writer template audit vs v83

**Scope:** review `template_gms_79_1.json` socket `handlers`/`writers` against
`template_gms_83_1.json`; verify every implemented opcode against the v79 IDB;
verify modes/operations; sort both arrays opcode-ascending; report what is
missing vs v83.

**Grounding sources (all IDB-derived):**
- Opcode truth: `docs/packets/registry/gms_v79.yaml` — every entry
  `provenance: ida-discovered` with a v79 IDB address.
- Feature presence: `docs/tasks/task-113-gms-legacy-versions/v79-packet-delta.md`.
- New IDB evidence (this task): v79 IDB `GMS_v79_1_DEVM.exe` (ida session), the
  `CField_*` subclass `OnPacket` overrides that the Stage-B registry walk
  explicitly deferred (delta doc lines 194–198, 448–449).
- Operation-table values: ported from task-178's v79 RE
  (`docs/tasks/task-178-legacy-template-options/re-v79.md`,
  `re-cashshop-v79.md`); the three `types` tables were independently
  re-validated against v79 `CharacterMoveHandle.types`.

---

## 1. Opcode verification — PASS

| | Count | IDB-mapped | Result |
|---|---|---|---|
| handlers | 123 | 123/123 join a serverbound registry op (IDB address) | all verified |
| writers  | 163 | 163/163 join a clientbound registry op | all verified |

Serverbound registry count (123) equals the handler count; every join is
semantically consistent. Opcode collisions (`0x00`×4 Auth*, `0x0A`×2 serverlist,
`0x40`×2 portal/door) are legitimate multi-entry-per-opcode patterns that mirror
v83.

## 2. Modes / operations — FIXED

11 handlers had **no `options`** on the audited baseline. This task populates
them (values carry task-178 IDB provenance; `types` tables independently
validated):

| Handler | added `options` |
|---|---|
| BuddyOperationHandle | operations (4) |
| CashShopOperationHandle | operations (17) |
| GuildBBSHandle | operations (6) |
| GuildOperationHandle | operations (10) |
| MessengerOperationHandle | operations (5) |
| NoteOperationHandle | operations (3) |
| NPCShopHandle | operations (4) |
| StorageOperationHandle | operations (5) |
| MonsterMovementHandle | types (23) — == CharacterMoveHandle.types |
| NPCActionHandle | types (23) — == CharacterMoveHandle.types |
| PetMovementHandle | types (23) — == CharacterMoveHandle.types |

Writer `options` were already present and version-consistent (version-appropriate
count deltas: `NPCConversation` messageType 11 vs v83 14; `NPCShopOperation`
operations 11 vs 13; `CharacterInteraction` operations 21 vs 23).

> Overlap note: these 11 tables duplicate the unmerged task-178 branch. Reconcile
> at merge (take one or the other), do not land both.

## 3. Sort — APPLIED

`handlers` and `writers` are now stable-sorted by ascending numeric opcode. The
change is provably reorder-only + the 11 option additions (nothing else in the
file mutated; verified by set-diff).

---

## 4. Missing HANDLERS (v83 not v79) — 8, none a real gap

| v83 handler | v83 op | v79 status | Evidence |
|---|---|---|---|
| RegisterPicHandle | 0x1D | Absent (correct) | `usesPin=false`; no SPW/PIC cases in `CLogin::OnPacket` |
| CharacterSelectedPicHandle | 0x1E | Absent (correct) | same |
| CharacterViewAllSelectedPicRegisterHandle | 0x1F | Absent (correct) | same |
| CharacterViewAllSelectedPicHandle | 0x20 | Absent (correct) | same |
| ClientStartHandle | 0x23 | Absent | no `CLIENT_START` serverbound op in v79 registry |
| NoOpHandler | 0x24 | Absent (no-op) | `NEXON_PASSPORT`; v83 routes to a no-op |
| EnterDoorHandle | 0x085 | Absent / v83 artifact | v79 has `USE_DOOR` only; the v83 entry duplicates opcode 133 with `UseDoor` |
| OwlActionHandle | 0x42 | Likely absent | v79 serverbound registry has `OWL_WARP` only; verify if owl-search routing is wanted |

## 5. Missing WRITERS (v83 not v79) — 39

### 5a. Present in v79 client but UNROUTED → real gaps (30)

Each has a named v79 IDB handler at the opcode shown (subclass `OnPacket`
overrides the registry Stage-B walk skipped):

| Writer(s) | v79 op(s) | v79 handler (IDB) |
|---|---|---|
| ContiMove, FieldTransportState | 0x8C, 0x8D | `CField_ContiMove::OnContiMove/OnContiState` |
| AriantArenaShowResult | 0x93 | `CField_AriantArena::OnShowResult` |
| SnowballState, SnowballHit, SnowballMessage, SnowballTouch | 0x103–0x106 | `CField_SnowBall::On*` |
| CoconutHit, CoconutScore | 0x107, 0x108 | `CField_Coconut::On*` |
| GuildBossHealerMove, GuildBossPulleyStateChange | 0x109, 0x10A | `CField_GuildBoss::On*` |
| MonsterCarnivalStart..Result (8) | 0x10B–0x112 | `CField_MonsterCarnival::On*` |
| AriantArenaUserScore | 0x113 | `CField_AriantArena::OnUserScore` |
| WitchTowerScoreUpdate | 0x117 | `CField_Witchtower::OnScoreUpdate` |
| Tournament, TournamentMatchTable, TournamentSetPrize, TournamentUew | 0x125–0x128 | `CField_Tournament::On*` |
| TournamentCharacters | 0x129 | present but `nullsub` (client ignores) |
| WeddingProgress, WeddingCeremonyEnd | 0x12A, 0x12B | `CField_Wedding::On*` |
| MtsOperation2, MtsOperation | 0x143, 0x144 | `CITC::OnQueryCashResult / OnNormalItemResult` |

> Routing these requires the atlas writers to encode correctly for v79 (opcode +
> body). Out of scope for this task (sort + options); tracked here as follow-up.

### 5b. Confirmed absent in v79 (correctly not routed) — 9

| Writer | Evidence |
|---|---|
| SelectWorld, ServerListRecommendations, PicResult | no case in `CLogin::OnPacket` (delta §f) |
| PyramidGauge, PyramidScore | no `CField_Pyramid` subclass; absent from base `CField` switch |
| SheepRanchInfo, SheepRanchClothes | no `CField_SheepRanch` subclass |
| VegaScroll | no handler (later feature) |

### 5c. Unresolved — 1

| Writer | Note |
|---|---|
| UiOpen (OPEN_UI 0xDC) | no named handler found in v79 CWvsContext/CField/CCashShop/CITC dispatchers; likely absent |

### v79-only writer (not in v83)
`ChatGeneralChat` (0x98) — present in v79 template, no v83 counterpart.

---

## v79 clientbound subclass opcode map (new IDB evidence)

Closes the delta doc's deferred item ("Stage B should check `CField_ContiMove` /
`CField_Massacre` subclass `OnPacket` overrides"). Full `CField_*` subclass set
in v79: AriantArena, Battlefield, Coconut, ContiMove, GuildBoss,
MonsterCarnival(+Revive), SnowBall, Tournament, Wedding, Witchtower. **No**
`CField_Pyramid`, `CField_SheepRanch`, or `CField_Massacre` exist → Pyramid /
SheepRanch are absent in v79.
