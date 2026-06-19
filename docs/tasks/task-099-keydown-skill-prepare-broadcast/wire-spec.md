# Task 099 — Keydown Skill Prepare/Cancel Wire Spec (IDB-pinned)

Pinned from the five client IDBs (truth source). Registry CSVs and Cosmic were
cross-checked but are NOT authoritative; every opcode below was confirmed against
the decompiled `COutPacket`/`CInPacket` calls and then compared to
`docs/packets/registry/<ver>.yaml`. No registry disagreements were found.

IDA instances: v83 `:13342`, v84 `:13337`, v87 `:13341`, v95 `:13340`, JMS185 `:13339`.

## Critical clarification on the clientbound (remote) packets

The four clientbound remote ops (prepare / cancel) are **not** distinct
top-level opcodes. They are sub-dispatched by `CUserPool::OnUserRemotePacket`,
which switches on an `nType` value. **The `charId` (u32) is read by the
dispatcher itself**, BEFORE it calls the per-user handler — so
`CUserRemote::OnSkillPrepare` / `OnSkillCancel` only read the
skill-specific tail. The full wire field list for the clientbound packet is
therefore: `charId(u32)` [read by dispatcher] + the handler's reads.

The registry's clientbound `SKILL_EFFECT` / `CANCEL_SKILL_EFFECT` opcodes are
the values that select the `nType` branch (they equal the dispatch case label in
every IDB checked), so they are the correct opcodes to emit for the remote
relay.

The plan's stated v95 baseline ("clientbound remote-prepare … charId u32 +
skillId u32 + …") is correct in field content, but note the charId is consumed
by the dispatcher, not by `OnSkillPrepare`'s body. Same for cancel.

---

## Op × Version opcode summary (hex)

| Op | v83 | v84 | v87 | v95 | jms185 |
|----|-----|-----|-----|-----|--------|
| serverbound prepare (`DoActiveSkill_Prepare`) | `0x5D` | `0x5D` | `0x60` | `0x69` | `0x58` |
| serverbound cancel (`SendSkillCancelRequest`) | `0x5C` | `0x5C` | `0x5F` | `0x68` | `0x57` |
| clientbound prepare (`OnSkillPrepare`, nType) | `0xBE` | `0xC2` | `0xCB` | `0xD7` | `0xC4` |
| clientbound cancel (`OnSkillCancel`, nType) | `0xBF` | `0xC3` | `0xCC` | `0xD9` | `0xC5` |

Decimal cross-check vs registry: v83 (93/92/190/191), v84 (93/92/194/195),
v87 (96/95/203/204), v95 (105/104/215/217), jms185 (88/87/196/197). All match.

> Note on v84 clientbound opcodes: the v84 clientbound opcode table is shifted
> vs v83 (attacks occupy 0xBE–0xC1); prepare/cancel are 0xC2/0xC3 —
> IDA-verified at :13337 (OnUserRemotePacket cases 194/195). Serverbound is
> unshifted (0x5D/0x5C, v83-identical).

---

## Detailed per-(op, version) records

Field widths: u32 = Encode4/Decode4 (4 bytes LE), u16 = Encode2/Decode2,
u8 = Encode1/Decode1.

### 1. Serverbound prepare — `CUserLocal::DoActiveSkill_Prepare`

Ordered writes (`COutPacket`):

| ver | opcode | fields | source fname @ addr |
|-----|--------|--------|---------------------|
| v83 | `0x5D` | skillId u32, level u8, action u16 `(oneTimeAction & 0x7FFF) \| (moveAction << 15)`, actionSpeed u8 | `CUserLocal::DoActiveSkill_Prepare` @ `0x96A86E` |
| v84 | `0x5D` | skillId u32, level u8, action u16, actionSpeed u8 | `DoActiveSkill_Prepare` (unnamed `sub_9A9761`) @ `0x9A9761` |
| v87 | `0x60` | skillId u32, level u8, action u16, actionSpeed u8 | `CUserLocal::DoActiveSkill_Prepare` @ `0x9EE1E6` |
| v95 | `0x69` | skillId u32, level u8, action u16, actionSpeed u8, **[if skillId == 33101005: swallowMobId u32]** | `CUserLocal::DoActiveSkill_Prepare` @ `0x941710` |
| jms185 | `0x58` | skillId u32, level u8, action u16, actionSpeed u8, **[if skillId == 33101005: swallowMobId u32]** | `CUserLocal::DoActiveSkill_Prepare` @ `0xA39CFD` |

`action` is always packed identically: low 15 bits = oneTimeAction/prepare-action,
bit 15 = move-action LSB. The trailing conditional `swallowMobId u32` is the
Dragon Knight "Dragon's Roar / swallow" skill (33101005) and only exists on the
versions that ship that skill (v95, jms185). v83/v84/v87 have no such branch.

### 2. Serverbound cancel — `CUserLocal::SendSkillCancelRequest`

| ver | opcode | fields | source fname @ addr |
|-----|--------|--------|---------------------|
| v83 | `0x5C` | skillId u32 | `CUserLocal::SendSkillCancelRequest` @ `0x96D873` |
| v84 | `0x5C` | skillId u32 | `SendSkillCancelRequest` (`sub_9AD694`) @ `0x9AD694` |
| v87 | `0x5F` | skillId u32 | `CUserLocal::SendSkillCancelRequest` @ `0x9F22B8` |
| v95 | `0x68` | skillId u32 | `CUserLocal::SendSkillCancelRequest` @ `0x93D730` |
| jms185 | `0x57` | skillId u32 | `CUserLocal::SendSkillCancelRequest` @ `0xA3E3EC` |

Every version writes only `Encode4(skillId)` after the opcode. Some versions
remap a couple of skill ids before encoding (v95 remaps 32120000→32001003 etc.),
but the wire payload is a single u32. **The v83 cancel opcode (previously listed
UNKNOWN in the plan) is `0x5C`** — confirmed both in the IDB and the registry
(`CANCEL_BUFF` serverbound = 92).

### 3. Clientbound remote prepare — `CUserRemote::OnSkillPrepare`

Wire = `charId u32` (read by `OnUserRemotePacket`) then the handler reads:

| ver | nType/opcode | handler reads | source fname @ addr |
|-----|--------------|---------------|---------------------|
| v83 | `0xBE` | skillId u32, level u8, action u16, actionSpeed u8 | `CUserRemote::OnSkillPrepare` @ `0x980A81` |
| v84 | `0xC2` (dispatch case 194) | skillId u32, level u8, action u16, actionSpeed u8 | `sub_9C0C5F` @ `0x9C0C5F` |
| v87 | `0xCB` | skillId u32, level u8, action u16, actionSpeed u8 | `CUserRemote::OnSkillPrepare` @ `0xA06135` |
| v95 | `0xD7` | skillId u32, level u8, action u16, actionSpeed u8 | `CUserRemote::OnSkillPrepare` @ `0x953A30` |
| jms185 | `0xC4` | skillId u32, level u8, action u16, actionSpeed u8 | `CUserRemote::OnSkillPrepare` @ `0xA53F49` |

Dispatcher `CUserPool::OnUserRemotePacket` addresses: v83 `0x9724F9`,
v84 `0x9B2518`, v87 `0x9F7492`, v95 `0x94B390`, jms185 `0xA44246`. Each opens
with `CInPacket::Decode4` (charId) then `GetRemoteUser`. The handler decodes
the same four fields in the same order on every version.

### 4. Clientbound remote cancel — `CUserRemote::OnSkillCancel`

Wire = `charId u32` (dispatcher) then handler reads:

| ver | nType/opcode | handler reads | source fname @ addr |
|-----|--------------|---------------|---------------------|
| v83 | `0xBF` | skillId u32 | unnamed `sub_980BF5` @ `0x980BF5` (IDB left handler unnamed; registry records it as `OnSkillCancel` w/ alt `sub_980BF5`) |
| v84 | `0xC3` (dispatch case 195) | skillId u32 | unnamed `sub_9C0DD3` @ `0x9C0DD3` |
| v87 | `0xCC` | skillId u32 | `CUserRemote::OnSkillCancel` @ `0xA062B1` |
| v95 | `0xD9` | skillId u32 | `CUserRemote::OnSkillCancel` @ `0x954600` |
| jms185 | `0xC5` | skillId u32 | `CUserRemote::OnSkillCancel` @ `0xA540C4` |

The cancel handler reads exactly one u32 (`Decode4` → skillId) on every version;
the remaining body is local animation cleanup (no further packet reads).

---

## Read-order deltas across versions

The **field read/write order is identical across all five versions** for every
op. There are no width changes and no field reordering. The only differences are:

1. **Opcode values shift per version** (see summary table). The serverbound and
   clientbound opcodes are version-specific and must be looked up per tenant;
   they are NOT derivable from one another.
   - v83 serverbound matches v84 (`0x5D`/`0x5C`). v84 clientbound is shifted
     vs v83: prepare `0xC2` / cancel `0xC3` (v83 has `0xBE`/`0xBF`); the v84
     opcode table is shifted by the attack-writer insertions at 0xBE–0xC1.
   - v87 sits at `0x60`/`0x5F`/`0xCB`/`0xCC`.
   - v95 sits at `0x69`/`0x68`/`0xD7`/`0xD9`.
   - jms185 sits at `0x58`/`0x57`/`0xC4`/`0xC5` (lower than the GMS line).

2. **Serverbound prepare has a trailing conditional `swallowMobId u32`** on
   v95 and jms185 only (skillId == 33101005, Dragon Knight swallow). v83/v84/v87
   omit it entirely. This is the ONLY structural payload difference, and it is
   gated on a skill id that is out of scope for this task's keydown set — but a
   correct generic encoder/decoder for the serverbound prepare on v95/jms185
   must account for it.

3. **Clientbound charId is consumed by the dispatcher, not the handler.** This
   is uniform across all versions; the relay packet the server emits must lead
   with `charId u32`.

4. **v84 clientbound opcode shift.** v84's `OnUserRemotePacket` uses case
   labels 194/195 for prepare/cancel (vs v83's 190/191), and the emitted wire
   opcodes are `0xC2`/`0xC3` — not `0xBE`/`0xBF` as in v83. The shift is due
   to attack-writer insertions at 0xBE–0xC1 in the v84 clientbound table
   (IDA-verified at :13337). Emit `0xC2`/`0xC3` for v84 prepare/cancel.

No version diverges in a way that breaks the plan's "single relay shape +
per-version opcode table" assumption. A shared codec parameterized by
(opcode, optional-swallow-field) covers all five versions.

---

## OQ-5 — MovingShootAttackPrepare determination

**Finding: MovingShoot is OUT OF SCOPE for the keydown prepare/cancel broadcast.**

Evidence per version:

- **v95** (`:13340`): `CUserPool::OnUserRemotePacket` @ `0x94B390` has a distinct
  `nType 216` → `CUserRemote::OnMovingShootAttackPrepare` @ `0x953BC0`, fed by
  serverbound `CUserLocal::TryDoingSmoothingMovingShootAttackPrepare` @
  `0x912F90` (opcode 51 / `0x33`). That serverbound function is the
  **fire-while-moving** path: it is gated on `(m_nMoveAction & 0xFFFFFFFE) == 0x12`
  (a moving/jumping move-action state) and weapon types 45/46 (bow/crossbow),
  47/49 (gun/knuckle). It is the continuous-shoot mechanic, NOT the keydown
  prepare. The in-scope keydown skills (Hurricane 3121004, Rapid Fire 5221004,
  Piercing Arrow, etc.) are handled by `DoActiveSkill_Prepare` via
  `is_keydown_skill()`, which routes to the regular prepare opcode — not through
  moving-shoot. Hurricane 3121004 appears explicitly in `DoActiveSkill_Prepare`'s
  keydown branch and in both `OnSkillPrepare`/`OnSkillCancel` keydown id lists,
  confirming it relays via SKILL_EFFECT/CANCEL_SKILL_EFFECT.

- **v83 / v84 / v87 / jms185**: There is **no `OnMovingShootAttackPrepare` nType
  case at all** in their `OnUserRemotePacket` dispatch tables. The dispatch goes
  directly attack → prepare → cancel → hit with no moving-shoot slot:
  - v83 `0x9724F9`: BA–BD attack, BE prepare, BF cancel, C0 hit.
  - v84 `0x9B2518`: 190–193 attack, 194 prepare, 195 cancel, 196 hit.
  - v87 `0x9F7492`: C7–CA attack, CB prepare, CC cancel, CD hit.
  - jms185 `0xA44246`: C0–C3 attack, C4 prepare, C5 cancel, C6 hit.
  So moving-shoot-prepare as a *remote-relayed* packet does not even exist on
  these versions; the in-scope keydown skills relay only through prepare/cancel.

Conclusion: no in-scope keydown skill dispatches through
`OnMovingShootAttackPrepare`. No parallel "moving-shoot prepare" relay packet is
required downstream. MovingShoot stays out of scope.

---

## UNRESOLVED cells

**None.** All four ops were pinned with an exact opcode, ordered field list, and
source fname + address in all five IDBs, and all opcodes agree with the
checked-in registry. The only IDB naming quirk is that v83/v84 leave the
clientbound cancel handler unnamed (`sub_980BF5` / `sub_9C0DD3`); both were
verified by following the dispatcher case label, and v83's is already documented
as such in `docs/packets/registry/gms_v83.yaml`.
