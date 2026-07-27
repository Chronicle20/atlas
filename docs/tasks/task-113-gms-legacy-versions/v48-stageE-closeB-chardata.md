# v48 Stage E — CLOSE batch B (char-data legacy encoding)

Anchor v61, IDB port 13337 (GMS_v48_1_DEVM.exe). Branch
`task-113-gms-legacy-versions`. The heaviest v48 cluster: CHAR_INFO, CHARLIST,
ADD_NEW_CHAR_ENTRY, VIEW_ALL_CHAR cb (+4 sub-structs), VIEW_ALL_CHAR sb,
DISTRIBUTE_AP sb, SPAWN_PLAYER cb.

## Result summary

**7 op cells promoted ❌/🟡 → ✅** (v48 verified 136 → **143**, +7 net), **1 demux
arm dispositioned n-a**, **1 cell remaining** (SPAWN_PLAYER — client-crash-risk
codec divergence, stopped cleanly). All other versions UNCHANGED. `matrix --check`
exit 0; problem-grep 0; v48 conflicts 0.

Every fixture byte traces to a live v48 decompile line — no blind mirrors, no
fabricated bytes.

## Derived v48 legacy shared encoding (Stage A)

### GW_CharacterStat — v48 (sub_49B627 @0x49b627) vs v61 (@0x4b4081)

Field-by-field, the ONLY divergence is the pet-locker section:

| field | v48 | v61 | codec gate |
|---|---|---|---|
| id (int), name (13), gender, skin, face, hair | same | same | — |
| **pet locker SN** | **1× 8-byte** (DecodeBuffer 8 @0x49b6bc) | **3× long** (DecodeBuffer 24) | `>=61` multi, else single |
| level, job, str/dex/int/luk, hp/maxhp/mp/maxmp (short), ap, sp | same | same | — |
| exp, fame, mapId, spawnPoint | same | same | — |
| gachaExp / trailing int / nSubJob | absent | absent (gacha absent @v61) | already gated |

Gate: `character_statistics.go` pet block `(GMS && >28)` → `(GMS && >=61)`.
v48 (48<61) → single long; v28 and v61+ unchanged.

### AvatarLook — v48 (sub_49E1E0 @0x49e1e0) vs v61 (@0x4b76c6)

| field | v48 | v61 |
|---|---|---|
| gender, skin, face, !mega, hair | same | same |
| equip loop / masked loop (0xFF term) | same | same |
| cash weapon (int) | same @0x49e2b6 | same |
| **pet** | **1× 4-byte int** (Decode4 @0x49e2b9) | **3× int** (DecodeBuffer 12 @0x4b77b1) |

Gate: `avatar.go` pet block — added a v48 (GMS 29..60) branch writing a SINGLE
4-byte int; `>=61` keeps 3 ints; `<=28` keeps the 8-byte long.

## Per-cell outcomes

| Cell | Dir | v48 handler | Outcome |
|---|---|---|---|
| CHARLIST | cb | sub_5013ED @0x5013ed | ✅ v48-gated (drops trailing slot-count int for GMS<61; v61 sub_56688D @0x566b02 reads it) |
| ADD_NEW_CHAR_ENTRY | cb | sub_501973 @0x501973 | ✅ =legacy (`legacyAddEntry` [code][stat][avatar], no trailer) |
| VIEW_ALL_CHAR (Characters/Count/SearchFailed/Error) | cb | sub_50232D @0x50232d | ✅ (dispatcher mode 0=chars, 1=count, 2/4/5=bare-byte error) |
| CHAR_INFO | cb | CWvsContext::OnCharacterInfo @0x71caed | ✅ v48-gated legacy branch |
| DISTRIBUTE_AP | sb | sub_71CD00 @0x71cd00 | ✅ (registry aligned; demux AutoDistributeAp arm n-a) |
| VIEW_ALL_CHAR | sb | sub_502293 @0x502293 | ✅ (empty body; export stub spliced) |
| SPAWN_PLAYER | cb | sub_6BBC17 @0x6bbc17 | **REMAINING** — see below |

## Codec gates added (all leave v61/v72/v79/v83/v84/v87/v95/jms UNCHANGED)

1. `model/character_statistics.go` — pet gate `>28`→`>=61` (v48 single 8-byte pet).
2. `model/avatar.go` — added v48 (GMS 29..60) single-4-byte-int pet branch.
3. `character/clientbound/list.go` — drop trailing `characterSlots` int for GMS<61.
4. `character/clientbound/info.go` — GMS 29..60 legacy branch: NO marriage bool,
   NO alliance string, NO medalInfo byte, SINGLE flag-gated pet (no "more pets"
   terminator), NO monster-book. v48 @0x71caed vs v61 @0x8455ed which adds all four.
   (medal block `>61` and chair `>=87` already absent for v48.)

## Arms dispositioned n-a

- **AutoDistributeAp** (`CWvsContext::SendAbilityUpRequest#AutoDistributeAp`) —
  v48 SendAbilityUpRequest (sub_71CD00) builds only the single-stat COutPacket(67)+
  Enc4(updateTime)+Enc4(statFlag); the ZArray<StatPair> auto-distribute overload
  does not exist in this 2009-era client. Stripped the unresolved export stub +
  `_unimplemented.json` entry so the DISTRIBUTE_AP demux worst-of grades only the
  resolved single-stat arm (mirrors v61, which has no AutoDistributeAp report).

## Interaction false-pass corrected

`interaction/clientbound/v48_test.go` `TestInteractionArmsV48` asserted v48
avatar == v83 — a false premise, since the shared `AvatarLook::Decode` (sub_49E1E0)
reads a single 4-byte pet where v83 reads 3 ints. Corrected the two avatar-bearing
arms (Enter, EnterResultSuccess) to assert the exact 8-byte-per-avatar delta;
avatar content is byte-verified by the model + char-list v48 fixtures.

## Anchor-regression — verified counts held (all 8 versions)

| ver | count | ver | count |
|---|---|---|---|
| gms_v61 | 208 ✓ | gms_v84 | 345 ✓ |
| gms_v72 | 216 ✓ | gms_v87 | 379 ✓ |
| gms_v79 | 228 ✓ | gms_v95 | 399 ✓ |
| gms_v83 | 367 ✓ | jms_v185 | 362 ✓ |

gms_v48 136 → **143**. `go test -race` green for every changed package
(model, character/*, login/serverbound, interaction/*). `go vet ./libs/atlas-packet/...`
clean. Tool tests green. `matrix --check` exit 0; problem-grep 0; v48 conflicts 0.

## EXACT remaining — SPAWN_PLAYER cb (v48 op 100 / 0x64)

Client read = `sub_6B277B` (OnUserEnterField, Decode4 charId) → `sub_6BBC17`
(CUserRemote::Init) — **IDA-verified read order captured**. Diverges from the
current `spawn.go` `<83` legacy path (tuned for v79 sub_8D589E) in FOUR ways —
each a shared-codec change touching the v61/v72/v79 anchors, and a wrong spawn
wire crashes the client, so stopped cleanly per the brief's crash-risk rule:

1. **No jobId short.** v48 goes CTS-foreign (sub_5CBA1F @0x6bbcde) straight to
   `AvatarLook::Decode` (@0x6bbcea) — no Decode2(jobId) between. The current legacy
   path unconditionally writes `WriteShort(m.jobId)`.
2. **Different CTS-foreign decoder — sub_5CBA1F @0x5cba1f.** Must be byte-verified
   against `model.CharacterTemporaryStat.EncodeForeign` for v48 (mask layout). This
   is the load-bearing unknown: if the v48 foreign mask differs, it needs its own
   v48 CTS-foreign codec branch (a second complex shared codec) before any spawn
   fixture can be trusted.
3. **Pet section shape unconfirmed.** v48 reads `if (Decode1) { …sub_58C7CC(this,v4) }`
   (@0x6bbe19) — a single flag then a helper decode, NOT visibly the current 3-slot
   bool-terminated loop. sub_58C7CC needs decompiling to confirm single vs loop.
4. **6-flag tail, not the codec's 7.** After the mount triple (@0x6bbe92..0x6bbeac)
   v48 reads exactly six Decode1-gated blocks: miniroom (@0x6bbed5), adboard
   (@0x6bc045), couple (@0x6bc174), friend (@0x6bc1bf), marriage (@0x6bc20a),
   final-effect (@0x6bc25c). The current legacy path emits miniroom/adboard/couple/
   friend/marriage + newyearcard(GMS<95) + berserk = 7 bytes → one too many, and
   no single final-effect flag. (v79 has an extra count+loop block between marriage
   and final-effect that v48 lacks — batch 11's "6 vs 7".)

Fix = a dedicated `<61` spawn gate (drop jobId, v48 CTS-foreign mask, 6-flag tail)
on a codec shared with the verified v61/v72/v79 anchors + sub_5CBA1F/sub_58C7CC
verification. Dedicated batch-sized; deferred to avoid a client-crash false pass.

## Commits (6, one logical unit split into 2 for DISTRIBUTE_AP)

1. `245e1a235d` — derive v48 GW_CharacterStat/AvatarLook + verify CHARLIST/ADD_ENTRY/
   VIEW_ALL cb (shared codec gates, 6 clientbound fixtures, interaction false-pass fix,
   run.go sub_5013ED candidatesFromFName case, CharacterList report).
2. `1521a60708` + `ca40e67af1` — verify DISTRIBUTE_AP sb, disposition AutoDistributeAp
   n-a (export stub strip + _unimplemented; first commit carried only the report
   deletions after a failed `git add` on the already-`git rm`'d paths).
3. `54db2f812e` — verify VIEW_ALL_CHAR sb op (export splice + registry fname align).
4. `0a9b9fbd2b` — verify CHAR_INFO cb (v48 legacy codec branch + golden).

Each staged explicitly (never `git add -A`); branch verified
`task-113-gms-legacy-versions` after each; no out-of-scope report-regen drift
(AuthSuccess/ChatMulti/ReactorHitRequest/SUMMARY/MonsterCarnival untouched).
