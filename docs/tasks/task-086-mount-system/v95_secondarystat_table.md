# GMS v95 SecondaryStat (CTS) bit table — IDA-verified 2026-06-13

Source of truth: the v95 client (`GMS_v95.0_U_DEVM.exe`, IDA port 13339). Each bit
was read from the stat's `_dynamic_initializer_for__CTS_X__` function, which does
`UINT128(1) << N` (the shift `N` is the bit; shifts ≥10 print as hex, e.g. `0xAu`).
Cross-checked against `SecondaryStat::DecodeForLocal` @0x7350E0 decode order and the
trailing two-state loop literal (`bits = 122`). **Zero bit collisions.**

## Key results

- **Bits 0–81 are identical to the registry's shared base** (v83 order). `CTS_ReverseInput`
  @51 is v95's name for `CONFUSE`. So `buildCharacterTemporaryStatRegistry` bits 0–81 are
  already correct for v95; only the post-SoulStone region diverges.
- **RideVehicle (MONSTER_RIDING) = bit 125** on v95 (registry currently puts it at 113).
- The mask is read/written identically by client and Atlas (big-endian UINT128 dword
  array, AND'd in wire order) — so a registry shift of 125 lands MonsterRiding exactly
  where the v95 client reads it. No per-version mask placement; the version-gated
  enumeration does the work (same principle as the v83 fix).
- `CTS_Undead` = bit **128** and `CTS_SummonBomb` = bit **129** — both **overflow** the
  128-bit mask, so the v95 client cannot receive them via GIVE_BUFF. Undead is the 7th
  two-state slot and is effectively disabled on v95.

## Full table (bit → CTS name → atlas character.TemporaryStatType)

Bits 0–81 (shared base; abbreviated — same as registry order):
`0 WeaponAttack … 7 Speed, 8 Jump, 9 MagicGuard, 10 DarkSight … 51 Confuse(ReverseInput) … 81 SoulStone`

Bits 82–121 (v95-specific post-SoulStone block — **differs from registry**):

| bit | CTS | atlas type | exists? |
|----|-----|-----------|---------|
| 82 | Flying | Flying | yes |
| 83 | Frozen | Frozen | yes |
| 84 | AssistCharge | AssistCharge | yes |
| 85 | Enrage | **Enrage** | NEW |
| 86 | SuddenDeath | SuddenDeath | yes |
| 87 | NotDamaged | NotDamaged | yes |
| 88 | FinalCut | FinalCut | yes |
| 89 | ThornsEffect | ThornsEffect | yes |
| 90 | SwallowAttackDamage | SwallowAttackDamage | yes |
| 91 | MorewildDamageUp | WildDamageUp | yes |
| 92 | Mine | Mine | yes |
| 93 | EMHP | EMHP | yes |
| 94 | EMMP | EMMP | yes |
| 95 | EPAD | EPAD | yes |
| 96 | EPDD | EPPD | yes |
| 97 | EMDD | EMDD | yes |
| 98 | Guard | Guard | yes |
| 99 | SafetyDamage | SafetyDamage | yes |
| 100 | SafetyAbsorb | SafetyAbsorb | yes |
| 101 | Cyclone | Cyclone | yes |
| 102 | SwallowCritical | SwallowCritical | yes |
| 103 | SwallowMaxMP | SwallowMaxMP | yes |
| 104 | SwallowDefence | SwallowDefense | yes |
| 105 | SwallowEvasion | SwallowEvasion | yes |
| 106 | Conversion | Conversion | yes |
| 107 | Revive | Revive | yes |
| 108 | Sneak | Sneak | yes |
| 109 | Mechanic | **Mechanic** | NEW |
| 110 | Aura | **Aura** | NEW |
| 111 | DarkAura | **DarkAura** | NEW |
| 112 | BlueAura | **BlueAura** | NEW |
| 113 | YellowAura | **YellowAura** | NEW |
| 114 | SuperBody | **SuperBody** | NEW |
| 115 | MorewildMaxHP | **WildMaxHpUp** | NEW |
| 116 | Dice | **Dice** | NEW (special: 22×int32 in client) |
| 117 | BlessingArmor | **BlessingArmor** | NEW |
| 118 | DamR | **DamageReduce** | NEW |
| 119 | TeleportMasteryOn | **TeleportMastery** | NEW |
| 120 | CombatOrders | **CombatOrders** | NEW |
| 121 | Beholder | **Beholder** | NEW |

Two-state group bits 122–128 (trailing base-stat loop, `bits=122`, 7 iterations):

| bit | CTS | atlas type | note |
|----|-----|-----------|------|
| 122 | EnergyCharged | EnergyCharge | |
| 123 | Dash_Speed | DashSpeed | |
| 124 | Dash_Jump | DashJump | |
| 125 | RideVehicle | **MonsterRiding** | the mount bit |
| 126 | PartyBooster | **PartyBooster** | NEW — replaces SpeedInfusion vs v83 |
| 127 | GuidedBullet | HomingBeacon | |
| 128 | Undead | Undead | overflow — not receivable |

## Implementation notes

- Add a dedicated `GMS && major>=95` enumeration branch; leave v83/v84, v87, and JMS paths
  untouched (v87's 4-stat block and JMS remain as-is/unverified).
- Two-state base-stat blocks for v95 differ from v83 (PartyBooster, not SpeedInfusion) — the
  v95 `getBaseTemporaryStats` variant must emit the v95 order, and each block's wire size must
  match the client's per-stat `DecodeForClient` (verify before claiming render-correct).
- v87 and v111 still need their own initializer-derived tables (same method, ports 13338/13342).
