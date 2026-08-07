# task-190 — Investigation Record

Captured 2026-08-04. This is the evidence behind the PRD; it is not a design.

## 1. How this surfaced

Reported during task-153 (Corsair Battleship) testing on the `atlas-pr-1138`
ephemeral namespace, tenant `cf1ebecd-732d-441c-ac91-c0dde8a3eee1`, GMS 83.1,
character 1, map 240011000:

> "the mount/dismount and use of skills seems to work. however, once i took magic
> damage, i think things got bugged. i couldn't use skills anymore. and cancelling
> the buff didn't dismount me."

Battleship itself was exonerated — ship HP initialised to 20 000 (matching the
client's `200 × (charLevel + 2×SLV − 120)` for a level-200 character at SLV 10) and
drained 20 000 → 16 710 across two mounts with zero errors or warnings from the
battleship package. The HP gauge was also working; it renders as the mount buff
icon's progression rather than a separate bar.

## 2. Live timeline (Loki, `{namespace="atlas-pr-1138"}`)

| Time (UTC) | Event |
|---|---|
| 22:52:27.547 | mount #1 APPLIED (`5221006`, `MONSTER_RIDING` 1932000, `duration` 2147483647) |
| 22:52:30 / :49 / 22:53:01 | three clean dismounts — `[CharacterBuffCancel] read [skillId [5221006]]` each followed by EXPIRED |
| 22:53:05.767 | mount #4 APPLIED |
| **22:53:10.242** | **mob 7130002 skill 126 → `APPLY {"sourceId":126,"level":2,"duration":15,"changes":[{"type":"SLOW","amount":80}]}`** |
| **22:53:10.365** | **first `Read a unhandled message with op 0x63`** — 123 ms after the APPLY |
| 22:53:13 – 22:53:28 | battleship drains continue normally (19 642 → 18 525) |
| 22:53:37.630 | last `CharacterBuffCancel` the client ever sends |
| 22:54:48 | mount #5 APPLIED (after a relogin; buff survived in atlas-buffs) |
| 22:55:02 – 22:57:26 | drains continue (19 743 → 16 710) |
| 22:56:45.364 | last SLOW APPLY |
| 22:56:54.259 | last SLOW expiry |
| **22:56:54.319** | **last `0x63`** — 60 ms after that expiry |

The mob re-applied SLOW nine times between 22:53:10 and 22:56:45 (roughly every
20 s), so the loop never got a chance to end. `count_over_time` per minute across
the window: 200 / 314 / 273 / 700. Roughly 1,500 packets in 3m44s.

The `0x63` window brackets the disease window on both ends to within ~100 ms. No
`0x63` appears anywhere outside it.

## 3. Mechanism

1. `atlas-monsters .../monster/processor.go:1242` — `duration := int32(sd.Duration())`,
   forwarding WZ `time` (seconds) verbatim.
2. `atlas-buffs .../buffs/buff/model.go:142` — `expiresAt: time.Now().Add(time.Duration(duration) * time.Millisecond)`.
   15 → 15 ms.
3. `libs/atlas-packet/model/character_temporary_stat.go:666` —
   `et := int32(v.ExpiresAt().Sub(time.Now()).Milliseconds())`, already ≤ 0 by encode time.
4. Client receives a temporary stat born expired; `CWvsContext::CheckTemporaryStatDuration`
   sends serverbound `CANCEL_DEBUFF` every tick.
5. `template_gms_83_1.json` has no handler at `opCode` 99, and `CANCEL_DEBUFF` has no
   codec or handler anywhere in the repo → the request is logged and discarded, forever.

**Blast radius.** `libs/atlas-constants/monster/skill.go` `SkillTypeToDiseaseName` maps 13
diseases: SEAL, DARKNESS, WEAKEN, STUN, CURSE, POISON, SLOW, SEDUCE, CONFUSE, UNDEAD,
STOP_PORTION, STOP_MOTION, FEAR. All are affected. SEAL is the one that blocks skill use
outright and is the most likely explanation for the reported symptom.

**Unproven.** The exact client-side mechanism by which the wedge blocks skill use and
buff-cancel is not established from logs alone — only that the client *stopped sending*
those packets (no server-side rejection appears; `battleship_attack_rejected_not_riding`
never fires). Confirming it needs a v83 IDB read of `CheckTemporaryStatDuration` and
`CUserLocal::TryDoingSkill`. Treat as strong-but-unverified until then.

## 4. Unit-contract chronology

Audited via `git log -L`/`-S` on 2026-08-04. **This contract has been flipped three times.**

| Date | Commit | Change |
|---|---|---|
| 2025-02-18 | `00b389a1f` | atlas-buffs `NewBuff` created with `* time.Second`. Seconds contract established. |
| 2026-04-29 | `6d8d253d2` | task-036 creates atlas-maps `MistTickTask`, publishing **ms** into the then-seconds contract. |
| 2026-05-02 | `11e07dfa7` | **Flip 1.** "fix(atlas-maps): mist tick publishes disease duration in seconds" — a 15 s mist had become ~4h10m. Fixed toward seconds, explicitly citing `executeDebuff` as the correct convention. |
| 2026-05-03 22:01 | `67ea45026` | task-054 fixes atlas-data `skill/reader.go` to emit skill-effect duration in ms. |
| 2026-05-03 22:08 | `197324e40` | **Flip 2.** task-054 flips atlas-buffs `time.Second` → `time.Millisecond` — the global consumer-side change, **one day after flip 1**. `git show --stat` confirms only atlas-data and atlas-buffs files were touched; atlas-monsters and atlas-maps were never propagated to. |
| 2026-05-04 | task-054 `audit.md` | Explicitly rules `processor.go:1068`/`:1105` "out of scope, correctly left alone" as "a different Duration field." Does not mention `executeDebuff` at all — the one path that does cross into atlas-buffs. |
| 2026-07-25 | `88d270bf1` | **Flip 3.** task-140 removes a leftover `/1000` in `atlas-consumables/consumable/processor.go`; every timed consumable buff had been expiring ~1000× early for ~3 months. |
| 2026-08-04 | — | atlas-monsters `:1242` and atlas-maps `:86` still on seconds. Flip 1 has been silently inverted since flip 2. |

**Why this matters for the fix.** Correcting `mobskill/reader.go` reverses `11e07dfa7`.
That reversal must be stated in the commit message and in the code comment, or the next
engineer reading the current `mist_tick.go` comment — which still asserts the seconds
contract — will flip it back a fourth time. This is the reasoning behind PRD §FR-3.

**Correction to an earlier claim:** an intermediate audit pass asserted the task-140
reference was a misattribution. It is not — `git show 88d270bf1 -- services/atlas-consumables`
confirms the `/1000` removal with a commit message explicitly citing task-054.

## 5. Double-conversion trap

Consumers of `mobskill.Model.Duration()` that already do their own ×1000 and would break
if the reader starts returning ms (PRD §FR-1.2):

- `atlas-monsters .../monster/processor.go:1068` — silently clamped to 60 s by
  `MistDurationCapMs`, so the regression would not be obvious.
- `atlas-monsters .../monster/processor.go:1105` — ~1000× too long.
- `atlas-maps .../tasks/mist_tick.go:86` — downstream, still divides back to seconds.

`processor.go:1242` needs no edit; the reader fix corrects it.

Tests pinning the old contract: `atlas-monsters .../monster/processor_test.go` ≈`:894`,
`:1179`, `:1236`; `atlas-maps .../tasks/mist_tick_test.go` ≈`:163`.

## 6. CANCEL_DEBUFF current state

`docs/packets/audits/STATUS.md:631`:

```
| CANCEL_DEBUFF | CWvsContext::CheckTemporaryStatDuration |  |  | ⬜ |  | ⬜ |  | ⬜ |  | ⬜ | 0x063 | ❌ | 0x063 | ❌ | 0x066 | ❌ | 0x06F | ❌ | 0x05E | ❌ |
```

Known: v83 `0x63`, v84 `0x63`, v87 `0x66`, v95 `0x6F`, jms185 `0x5E`. Unknown: v48, v61,
v72, v79 (`⬜`). v92 has a seed template but no registry file at all.

No codec, no handler, no template routing anywhere in the repo.

**The receiving plumbing already exists end-to-end** — the handler is a thin
decode-and-dispatch, not new infrastructure:

```
atlas-channel buff.Processor.CancelByTypes(f, characterId, types)   // character/buff/processor.go:73
  → COMMAND_TOPIC_CHARACTER_BUFF
    → atlas-buffs kafka/consumer/character/consumer.go:90
      → character.Processor.CancelByStatTypes                        // character/processor.go:131
        → Registry.CancelByStatTypes                                 // character/registry.go:233
```

## 7. Adjacent

`USER_CALC_DAMAGE_STAT_SET_REQUEST` (v83 `0x6C` / 108) is also unhandled. It appeared in the
same capture at 22:52:27.559 and 22:52:30.322 — immediately after each mount apply and
expire. §8.3 resolves what it is.

## 8. IDB findings (2026-08-04)

Read across all ten client IDBs: gms v48, v61, v72, v79, v83, v84, v87, v92, v95, and
jms v185. (v48 and v61 were loaded in a second pass.)

### 8.1 CANCEL_DEBUFF has an empty body

v83 `CWvsContext::CheckTemporaryStatDuration` @ `0xa20935`:

```c
void __thiscall CWvsContext::CheckTemporaryStatDuration(int *this)
{
  v2 = dword_BF060C();                                   // tick
  v3 = SecondaryStat::CheckByTime(this + 2125, v4, v2);  // locally-expired mask
  if ( UINT128::operator bool(v3) && v2 - this[3404] > 200 )
  {
    COutPacket::COutPacket(v5, 0x63);
    CClientSocket::SendPacket(TSingleton<CClientSocket>::ms_pInstance, v5);
    ZArray<unsigned char>::RemoveAll(v6);
  }
}
```

Nothing is encoded between construction and send. The client computes the expired mask and
discards it. Identical shape on every version examined.

### 8.2 Per-version opcodes and the throttle divergence

| Version | Opcode | Assigns throttle anchor? | Symbol |
|---|---|---|---|
| gms_v48 | `0x4E` (78) | **no** | `0x71b126` (already named) |
| gms_v61 | `0x5B` (91) | **no** | `0x84374e` (already named) |
| gms_v72 | `0x62` (98) | **no** | `sub_91914F` → renamed `CWvsContext::CheckTemporaryStatDuration` |
| gms_v79 | `0x61` (97) | **no** | `sub_96AD48` → renamed |
| gms_v83 | `0x63` (99) | **no** | `0xa20935` (already named) |
| gms_v84 | `0x63` (99) | **no** | `sub_A6BD3A` → renamed |
| gms_v87 | `0x66` (102) | **no** | `0xab7fd7` (already named) |
| gms_v92 | `0x6E` (110) | **yes** (`this[3796] = v2`) | `sub_9C7A70` → renamed |
| gms_v95 | `0x6F` (111) | **yes** (`m_tLastStatResetRequest`) | `0x9f2d30`, PDB-backed |
| jms_v185 | `0x5E` (94) | **yes** | `0xb0783e` |

Registry values for v83/84/87/95/jms185 (`provenance: csv-import`) are all confirmed
correct. v48, v61, v72, v79 and v92 are new. All ten resolved; none is `n-a`.

**`0x63` is version-unstable.** It is `CANCEL_DEBUFF` at v83/v84, but at v61 the same byte is
the calc-damage-stat request that `OnTemporaryStatReset` emits (§8.3). Resolve the opcode
from tenant config per version; a hard-coded `0x63` mis-routes on v61.

v48 uses a three-argument `COutPacket::COutPacket(v5, 78, 0)` constructor where later
versions use two. Body is still empty — no encode calls before `SendPacket`.

**The throttle divergence explains the observed spam rate.** Every version gates on
`tick - anchor > 200`, but v72–v87 never write the anchor in this function. It is written
elsewhere — on temporary-stat change. So once 200 ms elapse after the last stat change, the
guard latches open and the client sends once per frame forever, because the server never
sends a reset to advance it. That is the measured ~30 ms spacing. v92/v95/jms185 assign the
anchor before sending and therefore self-limit to 5/sec.

### 8.3 `0x6C` is the tail of the same handshake

v83 `CWvsContext::OnTemporaryStatReset` @ `0xa2071f` (clientbound, registry opcode 33) ends:

```c
UINT128::UINT128(&v24, v22, v31, 0x80u);
if ( IsCalcDamageStat(v24) )
{
  COutPacket::COutPacket(v29, 0x6C);
  CClientSocket::SendPacket(TSingleton<CClientSocket>::ms_pInstance, v29);
}
```

So `0x6C` is emitted *in reaction to* a stat reset touching a calc-damage stat — also
empty-bodied. The live `0x6C` right after each mount apply/expire fits: the mount buff
carries WEAPON_DEFENSE / MAGIC_DEFENSE. Unlike `0x63` it is one-shot per reset, not a loop,
so leaving it unhandled cannot wedge a client. PRD §9.3.

Its opcode is version-specific: **v48 `0x56` (86)**, **v61 `0x63` (99)**, **v83 `0x6C` (108)**.
The other seven are not yet read (PRD §9.7).

### 8.6 Clientbound reset mask width: v48 is 8 bytes, v61+ is 16

- v48 `OnTemporaryStatReset` @ `0x71b054`: `CInPacket::DecodeBuffer(a2, &v8, 8u)` — an
  8-byte mask, no `UINT128`, and a much smaller body (`0xd2` vs `0x1e0`).
- v61 `OnTemporaryStatReset` @ `0x84353a`: `CInPacket::DecodeBuffer(v28, 16)` — full
  `UINT128`, structurally identical to v83.

This is precisely the split `legacyGmsMask(t)` already implements in
`libs/atlas-packet/model/character_temporary_stat.go` (its comment cites "Pre-v61 GMS local
value block"). The existing clientbound writer therefore covers both arms unchanged.

### 8.7 `sub_77DC78` identified — and §9.6 downgraded

The v83 predicate gating the trailing `Decode1` in `OnTemporaryStatReset` is unnamed
(`sub_77DC78`), but the v61 IDB names the same call `SecondaryStat::IsMovementAffectingStat`.

That resolves the §9.6 observation: atlas writes `tSwallowBuffTime` unconditionally while the
client reads it only when the mask contains a movement-affecting stat. Because packets are
length-framed, the unread byte is simply ignored — this is a **benign over-send, not a
desync**. Worth tidying, not a live bug. (Correcting the initial characterization, which
called it a latent desync before the predicate was identified.)

### 8.4 The clientbound half already exists

The same function opens with `CInPacket::DecodeBuffer(a2, v31, 16)` — a 16-byte UINT128
mask — then `SecondaryStat::Reset`, `CTemporaryStatView::ResetTemporary`, and
`CWvsContext::ValidateStat`. atlas already encodes exactly this via
`libs/atlas-packet/character/clientbound/buff_cancel.go` (`cts.EncodeMask`). No new
clientbound packet is needed; the server's answer to `0x63` is a writer that already ships.

**Consequence for the design:** the PRD's original FR-2.6 (map wire stat names →
`CancelByTypes`) is impossible. The server must instead re-evaluate the character's buffs
and emit the existing `EXPIRED` path, which already reaches this writer. `atlas-buffs` has
only a fleet-wide `ExpireBuffs()` (`character/processor.go:190`), so a per-character variant
is the one genuinely new piece of plumbing.

**Consequence for security:** with no client-supplied parameters there is no buff-theft
vector. The real exposure is request amplification from the unthrottled v72–v87 clients.

### 8.5 Symbols named during this pass

`CWvsContext::CheckTemporaryStatDuration` renamed in the v72, v79, v84 and v92 IDBs
(previously `sub_91914F`, `sub_96AD48`, `sub_A6BD3A`, `sub_9C7A70`). v83, v87, v95 and
jms185 already carried the symbol.
