# v95 two-state group — standalone evidence record (task-167, closes "Task 41b")

This is a standalone transcription of the v95 two-state member group,
addresses, block sizes, bits 122-127, and the mask-gated tail read —
verified during design (not by this campaign's per-version pass) and
recorded here as its own evidence file so the long-open "Task 41b" question
has a durable, citable record independent of `design.md`.

Source: `design.md` §2.4 ("IDA — v95 client (`GMS_v95.0_U_DEVM.exe`) —
closes the 'Task 41b' gap (FR-4.3)"), transcribed verbatim with no
re-derivation. Binary: `GMS_v95.0_U_DEVM.exe` (IDA port 13341).

---

## Two-state mask bits (from `CTS_*` dynamic initializers)

| Bit | Stat |
|---|---|
| 122 | EnergyCharged |
| 123 | Dash_Speed |
| 124 | Dash_Jump |
| 125 | RideVehicle |
| 126 | PartyBooster |
| **127** | **GuidedBullet** |

These are read directly from linker-visible `_dynamic_initializer_for__CTS_X__`
symbols — this is the one IDB in the whole task-167 campaign that carries
full per-bit symbol names for the two-state group (contrast every gms_v6x
through gms_v92 evidence file, all of which report `CTS_*` symbol search as
empty).

## Group membership and block sizes

From `SecondaryStat::SecondaryStat` @`0x72F190`, which builds
`aTemporaryStat[7]`, and each variant's `DecodeForClient`:

| Slot | Stat | Template variant | `DecodeForClient` | Block size |
|---|---|---|---|---|
| 0 | EnergyCharged | `greater_equal<10000>`, `Expire<BaseOnLastUpdatedTime,DynamicTermSet>`, `Decrease<200,10000>` | `0x72C740` | base 13 + expireTerm 2 = **15** |
| 1 | Dash_Speed | `not_equal<0>`, `Expire<BaseOnLastUpdatedTime,DynamicTermSet>` | `0x726BA0` | **15** |
| 2 | Dash_Jump | same as slot 1 | `0x726BA0` | **15** |
| 3 | RideVehicle | `not_equal<0>`, `NoExpire` | `0x726AB0` | base only = **13** |
| 4 | PartyBooster | `TemporaryStat_PartyBooster` over `not_equal<0>`, `Expire<BaseOnCurrentTime,DynamicTermSet>` | `0x72C600` | base 13 + tCurrentTime 5 + expireTerm 2 = **20** |
| 5 | **GuidedBullet** | `TemporaryStat_GuidedBullet` (`NoExpire` + `m_dwMobID`) | `0x727180` | base 13 + dwMobId 4 = **17** |
| 6 | (unnamed) | `not_equal<0>`, `Expire<BaseOnLastUpdatedTime,DynamicTermSet>` | — | 15, **no CTS mask constant exists → unreachable on the wire** — this is the "Undead overflows the mask" slot the lib's pre-existing comment predicted |

`DecodeTime` (@`0x725430`) = Decode1 + Decode4 = 5 bytes, same shared
primitive used across every pre-95 version in this campaign.

**6 of the 7 constructed slots are reachable on the wire.** Slot 6 exists as
a constructed object (same template shape as slots 1/2) but has no
corresponding `CTS_*` mask bit — the client can never be told to decode it,
because bit 128 does not exist in the 128-bit `UINT128` mask (the group's
7th bit would need to be bit 128, one past the mask's own width). This is a
genuine mask-overflow, not a missing feature.

**Reachable trailer sum**: 15+15+15+13+20+17 = **95 bytes** (slots 0-5
only; slot 6 is structurally unreachable and contributes nothing to any
wire-observed trailer).

## The v95 two-state trailer read is mask-gated per member

`DecodeForLocal` (@`0x7350E0`), tail loop at `0x73DBA0-0x73DBF2`, builds
`UINT128(1) << shift` and tests it against the decoded flag — **only on a
hit** does it virtual-call that member's `DecodeForClient`.

This is why the lib's existing (pre-task-167) truncated 4-block v95 encode
— which sets only bits 122-125 (slots 0-3) — was already byte-consistent:
the existing fixture total `16+2+58` matches slots 0-3 exactly
(15+15+15+13 = 58). The truncation was correct as far as it went; it simply
never extended to PartyBooster/GuidedBullet.

## Set path

`CWvsContext::OnTemporaryStatSet` @`0xA02FC0`: explicit
`flag & CTS_GuidedBullet && aTemporaryStat[5]->IsActivated()` →
`CMob::SetGuided(CMobPool::GetMob(GetMobID()), GetReason(), 0)`. Same
constraints as v83 (nValue nonzero, rValue passed as skill/reason, dwMobId
selects the mob). `IsActivated` for the `NoExpire` variant @`0x726A80` is
`m_value != 0`.

## Reset path

`CWvsContext::OnTemporaryStatReset` @`0x9F2AB0`: mask-based; on the
GuidedBullet bit with an active stat, calls
`CMobPool::ResetGuidedMob(m_reason, mobId)` (@`0x6572E0`) then
`SecondaryStat::Reset(mask)`. A mask-based cancel fully clears the lock on
v95, exactly as it does on v83 — no value-0 give is needed.

## Movement filter (v95)

`SecondaryStat::IsMovementAffectingStat` @`0x7208C0`: Speed, Jump, Stun,
Weakness, Slow, Morph, Ghost, BasicStatUp, Attract, RideVehicle,
Dash_Speed, Dash_Jump, Flying, Frozen, YellowAura (15 total). GuidedBullet
is **not** in it — consistent with every other version examined in this
campaign, where GuidedBullet is never movement-affecting. See
`movement-filter.md` for the full per-version comparison.

## Additional v95-only client behavior

v95 additionally uses the stat for client-side damage
(`ApplyGuidedBulletDamage` @`0x7265E0`) — this is purely client-side; no
server-side work is implied.

## What this closes

Prior to this record, the extension of the pre-95 GuidedBullet-block
pattern to v95 (whether the group's shape, addresses, and gating were
verified rather than assumed) was an open item referred to internally as
"Task 41b." This record — table of addresses, block sizes, the bit
122-127 assignment, and the explicit mask-gated tail-read mechanism, all
sourced from design.md §2.4's IDA verification — closes that question with
a citable, standalone evidence file independent of the design document.
