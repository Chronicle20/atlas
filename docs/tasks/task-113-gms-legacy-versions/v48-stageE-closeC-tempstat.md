# v48 Stage E — CLOSE batch C (CharacterTemporaryStat legacy 64-bit mask)

Anchor v61, IDB port 13337 (GMS_v48_1_DEVM.exe). Branch
`task-113-gms-legacy-versions`. The cohesive cluster whose shared shape is the
pre-v61 CharacterTemporaryStat (SecondaryStat) mask: **GIVE_BUFF (op28, local)**,
**CANCEL_BUFF (op29, local)**, **SPAWN_PLAYER (0x64, foreign)**.

## Result

**3 op cells promoted ❌ → ✅** (v48 verified 143 → **146**, +3). All other
versions UNCHANGED. `matrix --check` exit 0; problem-grep 0; v48 conflicts 0.
Every fixture byte traces to a live v48 decompile line.

## The derived v48 CharacterTemporaryStat mask — 64-bit / 8-byte

IDA-verified the mask WIDTH divergence (vs the shared codec's 16-byte/128-bit
UINT128):

| path | function | mask read | addr |
|---|---|---|---|
| local set (GIVE_BUFF) | `CWvsContext::OnTemporaryStatSet` @0x71af4b → `sub_5CA524` | `DecodeBuffer(&v150, 8)` | 0x5ca539 |
| local reset (CANCEL_BUFF) | `CWvsContext::OnTemporaryStatReset` @0x71b054 | `DecodeBuffer(&v8, 8)` | 0x71b06e |
| foreign (SPAWN_PLAYER) | `sub_5CBA1F` (called by `sub_6BBC17` @0x6bbcde) | `DecodeBuffer(&v17, 8)` | 0x5cba33 |

All three read a plain **8-byte little-endian int64** and test **bits 0-46**.

### Bit → stat mapping = the shared registry shift order (NOT guessed)

The mask bits map **identically** to the shared `CharacterTemporaryStat`
registry shift order for bits 0-46. This is proven — not assumed — by the
foreign decoder `sub_5CBA1F`, whose per-bit value shapes match the registry's
per-stat `foreignValueWriter` stat-for-stat:

| bit | v48 foreign read (sub_5CBA1F) | registry shift → stat | registry foreign writer | match |
|---|---|---|---|---|
| 7 (0x80) | Decode1 (byte) @0x5cba48 | 7 Speed | ValueAsByte | ✓ |
| 10 (0x400) | flag only @0x5cbc99 | 10 DarkSight | NoOp | ✓ |
| 16 (0x10000) | flag only @0x5cbcbb | 16 SoulArrow | NoOp | ✓ |
| 17 (0x20000) | Decode4 (int) @0x5cbaeb | 17 Stun | ValueAsInt | ✓ |
| 18 (0x40000) | Decode2+Decode4 @0x5cbc0d/43 | 18 Poison | ValueSourceLevel (6B) | ✓ size |
| 19 (0x80000) | Decode4 @0x5cbb63 | 19 Seal | ValueAsInt | ✓ |
| 20 (0x100000) | Decode4 @0x5cbb27 | 20 Darkness | ValueAsInt | ✓ |
| 21 (0x200000) | Decode1 (byte) @0x5cba71 | 21 Combo | ValueAsByte | ✓ |
| 22 (0x400000) | Decode4 @0x5cbaaf | 22 WhiteKnightCharge | ValueAsInt | ✓ |
| 26 (0x4000000) | flag only @0x5cbc77 | 26 ShadowPartner | NoOp (<87) | ✓ |
| 30 (0x40000000) | Decode4 @0x5cbb9f | 30 Weaken | ValueAsInt | ✓ |
| 31 (0x80000000) | Decode4 @0x5cbbdb | 31 Curse | ValueAsInt | ✓ |
| 33 (0x2_00000000) | Decode2 (short) @0x5cbcd6 | 33 Morph | ValueAsShort | ✓ |
| 39 (0x80_00000000) | Decode4 @0x5cbd0f | 39 Seduce | LevelSource (4B) | ✓ size |
| 40 (0x100_00000000) | Decode4 @0x5cbd38 | 40 ShadowClaw | ValueAsInt | ✓ |

Speed=byte@7, Combo=byte@21, Morph=short@33, the three int-valued charges, and
the flag-only stealth/soul stats all land on their registry shifts — an
unambiguous cross-check that v48's bit numbering equals the shared shift order.
So **the only mask divergence is the width**, gated `GMS && MajorVersion < 61`.

The two-state base stats (EnergyCharge..Undead) occupy shifts 81-87, i.e.
`mask.H`; pre-v61 clients never read them (all three decoders stop at bit 46),
so `WriteLong(mask.L)` drops them and an empty v48 mask is 8 zero bytes.

## Local value block (GIVE_BUFF) — sub_5CA524

Per set bit: `Decode2(value)+Decode4(reason)+Decode2(duration)` (short + int +
short; the client does `500 * Decode2` @0x5ca58d). **No** disease split (the
helper reads Decode4 for every stat), **no** nDefenseAtt/nDefenseState bytes,
**no** trailing base-stat blocks — OnTemporaryStatSet reads only a Decode2 delay
@0x71af82 then an optional Decode1 (movement byte, `sub_5C941C(mask)`-gated,
emitted unconditionally as harmless last-field, matching the v61/72/79 precedent).

## Foreign (SPAWN_PLAYER) — sub_5CBA1F + sub_6BBC17

`sub_5CBA1F` ends after the per-bit value blocks: no defense bytes, no base
blocks. `sub_6BBC17` (CUserRemote::Init) read order confirms the closeB spawn
divergences: CTS-foreign @0x6bbcde → **AvatarLook::Decode @0x6bbcea with no
Decode2(jobId) between**; single-pet `if(Decode1){sub_58C7CC}` @0x6bbe19
(sub_58C7CC reads Int templateId + name + 8-byte SN + x/y shorts + stance + fh
short + 2 flag bytes = byte-identical to `model.Pet.Encode`); six tail flags
(miniroom @0x6bbed5 / adboard @0x6bc045 / couple @0x6bc174 / friend @0x6bc1bf /
marriage @0x6bc20a / final-effect @0x6bc25c), **no new-year-card, no team byte**.

## Codec gates added (all leave v61/72/79/83/84/87/95/JMS UNCHANGED)

`libs/atlas-packet/model/character_temporary_stat.go` — `legacyGmsMask(t)` =
`GMS && MajorVersion < 61`:
- `EncodeMask`: legacy → `WriteLong(mask.L)` (8 bytes) else 16-byte UINT128.
- `DecodeMask(r, t)`: legacy → `ReadUint64` (signature gained `t`; 3 callers updated).
- local `Encode`: legacy per-stat short+int+short, no defense bytes, no base blocks.
- `EncodeForeign`: legacy stops after value blocks (no defense bytes, no base blocks).
- `Decode`/`DecodeForeign`: mirror the legacy reads.

`libs/atlas-packet/character/clientbound/spawn.go` — `legacyV48 = GMS < 61`:
skip `WriteShort(jobId)`; single-pet flag instead of the 3-slot loop; drop the
new-year-card byte (`>=61 && <95`). Decode side mirrors all three.

`libs/atlas-packet/character/clientbound/buff_cancel.go` — `Decode` gained the
tenant to route `DecodeMask(r, t)`. (BuffGive/BuffCancel encoders needed no edit —
their Short/Byte trailers already satisfy v48's delay-short + optional-byte.)

### v28 note
GMS v28 (also `< 61`, no IDB, round-trip-only, no byte fixture) inherits the
legacy path: single-pet, no jobId, 8-byte mask. Previously it took the
equally-unverified 128-bit/16-byte path; an 8-byte mask is more plausible for a
pre-v61 client. The two shared spawn round-trip tests were updated to expect the
`<61` single-pet / no-jobId shape.

## Local vs foreign gate summary

| aspect | local (Encode / GIVE_BUFF, CANCEL_BUFF) | foreign (EncodeForeign / SPAWN_PLAYER) |
|---|---|---|
| mask | 8-byte `WriteLong(mask.L)` | 8-byte `WriteLong(mask.L)` |
| value block | short+int+short (all stats) | registry foreignValueWriter (byte counts match sub_5CBA1F) |
| defense bytes | none | none |
| base-stat blocks | none | none |

## Per-cell outcome

| Cell | Op | Outcome | Fixture |
|---|---|---|---|
| GIVE_BUFF cb | 28 | ✅ v48-gated | TestBuffGiveV48Mask (empty, 11B) + TestBuffGiveV48SingleStat (Combo bit-order + value block) |
| CANCEL_BUFF cb | 29 | ✅ v48-gated | TestBuffCancelV48ByteFixture (8-byte mask + trailer, 9B) |
| SPAWN_PLAYER cb | 0x64 | ✅ v48-gated | TestCharacterSpawnV48Golden (deterministic 99B: header+8-byte mask, avatar-follows-mask, 6-flag tail) |

## Anchor-regression — verified counts held (all 8 versions)

| ver | count | ver | count |
|---|---|---|---|
| gms_v61 | 208 ✓ | gms_v84 | 345 ✓ |
| gms_v72 | 216 ✓ | gms_v87 | 379 ✓ |
| gms_v79 | 228 ✓ | gms_v95 | 399 ✓ |
| gms_v83 | 367 ✓ | jms_v185 | 362 ✓ |

gms_v48 143 → **146**. `go test -race` green for model + character/*
(all 8 versions of the BuffGive/BuffCancel/CharacterSpawn round-trip + mask
fixtures re-run and pass); `go vet ./libs/atlas-packet/...` clean; packet-audit
tool `go build`+`go test` green. `matrix --check` exit 0; problem-grep 0; v48
conflicts 0.

## Tooling / report

- `tools/packet-audit/cmd/run.go`: added `candidatesFromFName("sub_6B277B") →
  CharacterSpawn` (the v48 unnamed spawn dispatch target, matching the sub_5013ED
  CharacterList precedent).
- `docs/packets/audits/gms_v48/CharacterSpawn.json`: surgically anchored the
  report IDAName to the resolved `sub_6B277B` (@0x6b277b) — the v48 export lists
  `CUserPool::OnUserEnterField` as an address-less alias — so the op links
  (mirrors the batch10 sub_XXXX-fname precedent). Diff verdict is advisory for
  tier-1; the byte-fixture is the verification.
- Evidence pinned (TIER1-FIXTURE): BuffGive (OnTemporaryStatSet), BuffCancel
  (OnTemporaryStatReset), CharacterSpawn (sub_6BBC17).

## EXACT remaining

None — all 3 target cells verified.
