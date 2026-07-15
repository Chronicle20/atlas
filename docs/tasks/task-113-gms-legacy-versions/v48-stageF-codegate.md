# Task-113 v48 — Stage F: Code-Gate Audit Column (FINAL PASS)

Doc-only pass. Fills the `v48` column of `code-gate-audit.md` for every row,
adds the v48-NEW gate section, confirms OQ-6 for v48, and verifies no
existing-version (v61/v72/v79/v83/84/87/95/JMS) evaluation changed as a side
effect (with the one documented exception below).

## 1. Rows filled

- **191** pre-existing gate cells filled programmatically = **`= v61`** anchor
  (mirroring v61's TRUE/FALSE/? state), since v48 shares the v61 codec for every
  gate whose boundary does not sit in (48,61]. Fast-path expected and confirmed:
  v48 and v61 are on the same side of every `<=28`/`>28`/`<=12`/`>12`/`>=72`/
  `>=73`/`>=79`/`>=83`/`>=84`/`>=87`/`>=90`/`>=95`/`==0`/`>61` boundary.
- All gate rows validated well-formed: **0 malformed, 0 empty v48 cells**
  (9-column table integrity checked by unescaped-pipe split).

### Main-table divergence overrides (v48 ≠ v61), hand-corrected in place
| Row | Change | v48 |
|---|---|---|
| `field/change` chase | predicate lowered `>=61`→**`>=48`** (`change.go:82`), all GMS incl. v48 emit the chase byte | **TRUE** |
| `character/spawn` inner block | predicate narrowed `<95`→**`>=61 && <95`** (`spawn.go:170`), v48 has no new-year-card flag | **FALSE** |
| `model/damage_info` per-mob CRC | boundary NOT lowered for v48; `>=61` correctly excludes v48 (attack sends body-verified B10, no CRC) | **FALSE** |
| `messenger/add` legacyAdd | **CORRECTION** — predicate `<72`→**`<=28`** (`add.go:51`); fixed a v61 false-pass (see §4) | **FALSE** |
| `npc/start_conversation` x/y | predicate `+ \|\| MajorVersion()==48` (`start_conversation.go:37`); v48 IDA-confirmed oid+x+y | **TRUE** |

## 2. v48-NEW gates added — "Campaign-added intra-legacy gates — Stage F (v48)"

25 rows added (fourth intra-legacy discriminator, all IDA-verified vs
`GMS_v48.1_U_DEVM.exe`, port 13340). file:line + representative commit each:

| Gate | file:line | boundary | v48 |
|---|---|---|---|
| GW_CharacterStat single pet-locker SN | `model/character_statistics.go:103` (Decode :190) | `>=61` | FALSE→single 8-byte pet |
| AvatarLook single pet (4-byte int) | `model/avatar.go:81` (:151) | `>=61` | FALSE→single int pet |
| CTS 8-byte int64 mask (local + foreign) | `model/character_temporary_stat.go:574` (`legacyGmsMask`) | `<61` | TRUE→8-byte LE int64 |
| SPAWN_PLAYER legacy (no jobId/single-pet/8-byte foreign mask/6-flag tail) | `character/clientbound/spawn.go:81` (`legacyV48`; :216) | `<61` | TRUE |
| ranged bulletX/Y omission | `model/attack_info.go:59` (`legacyGmsNoRangedBulletCoords`; :212,:365) | `<61` | TRUE→omit |
| IncreaseExperience shorter EXP body | `character/clientbound/status_message.go:530` (`legacyV48`; :588) | `<61` | TRUE |
| EffectWeather (BLOW_WEATHER) no bool | `field/clientbound/effect_weather.go:43` (:88) | `<61` | TRUE→omit |
| FIELD_OBSTACLE single-vs-list | `field/clientbound/field_obstacle_on_off_list.go:68` (:91) | `<61` | TRUE→single |
| GENERAL_CHAT drops bOnlyBalloon | `field/serverbound/general.go:57` (:73) | `<61` | FALSE→omit |
| SPOUSE_CHAT (cb+sb) | `field/clientbound/spouse_chat.go` / `field/serverbound/spouse_chat.go` | none | **no-gate** (flattened union) |
| cash buyOmitsCurrency | `cash/serverbound/shop_operation_buy.go:79` (`buyOmitsCurrency`) | `<61` | TRUE→omit currency int |
| guild alliance boundary (48,61] — GUILDMEMBER | `model/guild_member.go:20` | `<61` | TRUE→33-byte |
| guild alliance — MemberJoined | `guild/clientbound/operation.go:725` (`legacyNoAlliance`) | `<61` | TRUE→omit |
| guild alliance — GUILDDATA | `guild/clientbound/info.go:22` (`guildInfoLegacyNoAlliance`) | `<61` | TRUE→points only |
| CHAR_INFO legacy branch | `character/clientbound/info.go:86` (:214) | `>28 && <61` | TRUE→short body |
| CHARLIST trailing slots | `character/clientbound/list.go:72` (:112) | `>=61` | FALSE→omit slots |
| CHAR_INFO_REQUEST no petInfo | `character/serverbound/info_request.go:53` (:65) | `<61` | FALSE→omit petInfo |
| monster/serverbound/movement hackedCode | `monster/serverbound/movement.go:77` (:116) | `>=61` | FALSE→omit |
| auth_success nNumOfChar field | `login/clientbound/auth_success.go:78` (:152) | `>=61` | FALSE→omit |
| npc/shop_list legacy | `npc/clientbound/shop_list.go:76` (:115) | `<61` | TRUE |
| npc AskMenu count byte narrowed | `npc/clientbound/conversation.go:229` | `>=61 && <83` | FALSE (48<61) |
| summon/serverbound/attack updateTime | `summon/serverbound/attack.go:186` (`hasSummonAttackUpdateTime`) | `>=61` | FALSE→omit |
| pet/serverbound leading petId | `pet/serverbound/legacy.go:23` (`hasLeadingPetId`) | `>=61` | FALSE→omit |
| party/member_data leaderId | `party/member_data.go:49` (`legacyNoLeader`; :135) | `<61` | TRUE→omit |
| party/disband trailer | `party/clientbound/disband.go:42` (`legacyNoTrailer`; :59) | `<61` | TRUE→omit |
| party/invite autoJoin | `party/clientbound/invite.go:50` (`legacyNoAutoJoin`; :72) | `<61` | TRUE→omit |

(CHANGE_MAP chase `>=48`, spawn inner `>=61 && <95`, messenger `<=28`,
start_conversation `==48` are recorded in-place in their original sections, not
duplicated here — matching the v61 pass convention.)

## 3. OQ-6 confirmed for v48

Both the opcode-framing and the crypto legs of OQ-6 hold for v48 and are now
carried explicitly in the 4 framing/crypto rows (channel/main, channel/session,
login/main, login/session):

- **2-byte opcode framing (ShortReadWriter):** `CClientSocket::ProcessPacket
  @0x464fb6` reads the opcode via `CInPacket::Decode2` (uint16). v48 major = 48
  > 28 → NOT the `<=28` `ByteReadWriter` path → Short. (delta doc, top-level
  routing section.)
- **Standard AES-OFB (not zero-fill IV):** v48 uses an encrypted socket + the
  standard `COutPacket(1)` login handshake `CLogin::SendCheckPasswordPacket
  @0x4ffeb2`. v48 = 48 > 12 → NOT the pre-v13 zero-fill-IV path.

## 4. No existing-version evaluation changed (verified)

Every v48-new gate uses a `<61`/`>=61`/`==48`/`>=48` boundary → FALSE for v61+
(a `<61` gate) or TRUE for v61+ (a `>=61`/`>=48` gate) — so v61/v72/v79/v83/84/
87/95/JMS all stay on the branch they already took. Shared codecs spot-checked by
reading the gate boundary:

- **attack_info** (`legacyGmsNoRangedBulletCoords <61`, `legacyGmsNoSkillDataCrc
  <72`): v61+ keep the v72/v83 layout; only v48/(v28) omit bulletX/Y.
- **avatar / character_statistics** (single-pet `>=61`): v61+ read the multi-pet
  block unchanged.
- **character_temporary_stat** (`legacyGmsMask <61`): v61+ keep the 16-byte
  UINT128 mask; per-bit 0-46 identity proven so the 8 verified versions are byte-
  identical to before.
- **spawn** (`legacyV48 <61`, inner `>=61 && <95`): v61..v94 keep the new-year-card
  flag; v83+ unchanged.
- **damage_info** (`>=61`): v61+ keep the per-mob CRC (boundary not touched by v48).

**One deliberate exception (documented, not a regression):** the `messenger/add`
`legacyAdd` gate was **corrected** `<72`→`<=28`. The old `<72` was pinned on a
misidentified function; the real v61 Add sender (`sub_6D144E`) reads 6 fields
(channelId + pad present) like v48/v83. The correction flips **v61** from
"omit" to "present" — fixing a v61 **false-pass**, not introducing a regression;
v61 coverage held (208), evidence re-pinned. v72/v79/v83+ unchanged. Recorded
in-place with a `yes (CORRECTED)` verdict.

## 5. v28 boundary note (OWNER-REVIEW ITEM)

Every `<61` / `>=61` gate below the anchor also catches the **test-only v28**
variant (no v28 IDB exists). Per the v48 controller decision (close-I), v28 was
folded onto the pre-v61 legacy branch by inference — it rides the same 8-byte CTS
mask / single-pet / no-alliance / single-obstacle codec as v48. This is
**UNVERIFIED-BY-INFERENCE**; the in-code gates carry the `unverified-by-inference
(no v28 IDB)` comment and the v28 round-trip tests were made symmetric to the
legacy shape. Flagged for owner review in both the section preamble and the
file-header banner.

## Verification / bars

- Doc-only: `git status` shows only `code-gate-audit.md` (+ this report) modified;
  no `.go` file touched → `go build ./...` / `go vet ./...` remain clean (already
  were). Full export intentionally NOT run.
- Table integrity: 0 malformed rows, 0 empty v48 cells, 58 bold divergence rows.

## Commit

`task-113(v48): stage F — code-gate audit column` (explicit `git add` of
`code-gate-audit.md` + this report). Branch `task-113-gms-legacy-versions`.
