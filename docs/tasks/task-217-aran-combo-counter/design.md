# Aran Combo Counter — Design

Task: task-217-aran-combo-counter
Status: Draft for review
Created: 2026-08-12
Inputs: `prd.md` (approved)

---

## 1. Summary

The client owns the *trigger* (`CMob::OnHit` → `RequestIncCombo`, body-less);
the server owns the *count*. The server re-derives every gate, advances a
per-character counter, echoes it back with `SHOW_COMBO`, and applies /
cancels the Combo Ability buff so the icon and the `ARAN_COMBO` temporary
stat track the combo's lifetime.

The design phase closed all nine of the PRD's open questions against the six
IDBs and the v83 GMS WZ dump. Three of those answers change the shape of the
implementation materially versus the PRD's sketch:

1. **Every opcode in the PRD's table is already correct.** All twelve
   (2 ops × 6 versions) were IDA-verified; zero registry corrections are
   needed and the serverbound body is empty on every version (§2.1, §2.2).
   FR-6.1's "hard prerequisite" is discharged here, not during execution.
2. **The combo count is NOT the `ARAN_COMBO` stat value.** `DrawCombo` reads
   the count exclusively from `SHOW_COMBO`'s 4-byte body; the `ARAN_COMBO`
   secondary stat is a damage-calc input, decoded as a *signed short*
   (§2.3). Storing the count on that stat — the PRD's FR-3, inherited from
   task-142 — would conflate two values, truncate above 32767, and force a
   Kafka round trip per melee hit to read the count back. **The count lives
   in a process-local mirror in atlas-channel instead** (§3.2, §5.1).
3. **The client clears its own counter after an idle window and there is no
   clientbound way to force-clear it.** `CUserLocal::Update` calls
   `ClearCombo` after 3000 ms with no `SHOW_COMBO` (5000 ms on v95), and
   `ClearCombo` sends a *skill-cancel request* for the Combo Ability id.
   `DrawCombo` returns early on a count of 0 without releasing its layers, so
   `SHOW_COMBO 0` is a visual no-op. Server decay must therefore *match* the
   client's window rather than drive it, and the incoming cancel is a
   first-class reset input (§2.5, §3.4).

Everything else follows the PRD.

---

## 2. Verified findings

Every value below was read from the named IDB or WZ node. Nothing here is
inferred from general MapleStory knowledge.

### 2.1 Serverbound `ARAN_COMBO_COUNTER` — opcode and body

`CUserLocal::RequestIncCombo` is a guard (`m_bHoldCombo`) plus
`COutPacket(op)` plus `SendPacket`. **No body on any version.**

| Version | IDB | Function | `COutPacket(n)` | Registry | Match |
|---|---|---|---|---|---|
| gms v83 | `MapleStory_dump.exe` | `CUserLocal::RequestIncCombo` @ `0x9602f3` | `0xA3` (163) | 163 | ✅ |
| gms v84 | `GMS_v84.1_U_DEVM` | `CUserLocal__RequestIncCombo_send_0xA9` @ `0x99f346` | `169` | 169 | ✅ |
| gms v87 | `GMSv87_4GB` | `CUserLocal::RequestIncCombo` @ `0x9e37bc` | `0xAD` (173) | 173 | ✅ |
| gms v92 | `GMS_v92_1_DEVM` | `sub_8EF840` (unnamed) @ `0x8ef840` | `0xBA` (186) | 186 | ✅ |
| gms v95 | `GMS_v95.0_U_DEVM` | `CUserLocal::RequestIncCombo` @ `0x909070` | `189` | 189 | ✅ |
| jms v185 | `MapleStory_dump_SCY` | `CUserLocal::RequestIncCombo` @ `0xa2d435` | `0x9D` (157) | 157 | ✅ |

Closes PRD open questions 1 and 3. **No registry edits required.** The v92
function is unnamed in its IDB and should be renamed to
`CUserLocal__RequestIncCombo_send_0xBA` with `idb_save` as part of execution,
so the next reader does not have to re-derive it (per the project's
"name symbols while reversing" rule).

### 2.2 Clientbound `SHOW_COMBO` — opcode and body

`CUserLocal::OnIncComboResponse` is three statements on every version — v95's
PDB-backed decompile names the fields:

```c
this->m_nCombo        = CInPacket::Decode4(iPacket);   // 4-byte count
this->m_tLastSetCombo = get_update_time();
CUserLocal::DrawCombo(this);
```

**Body = one 4-byte little-endian count. No version divergence.** Dispatch
case labels, read from each client's `OnPacket` jump table:

| Version | Dispatcher | Case | Registry | Match |
|---|---|---|---|---|
| gms v83 | `CUserLocal::OnPacket` @ `0x95086b` | 225 (`0x0E1`) | 225 | ✅ |
| gms v84 | `CUserLocal__OnPacket_dispatch_209_240` @ `0x988698` | 230 (`0x0E6`) | 230 | ✅ |
| gms v87 | `CUserLocal::OnPacket` @ `0x9cbcd4` | 239 (`0x0EF`) | 239 | ✅ |
| gms v92 | `CField::OnPacket` @ `0x913a68` | 259 (`0x103`) | 259 | ✅ |
| gms v95 | `CUserLocal::OnPacket` @ `0x934238` | 257 (`0x101`) | 257 | ✅ |
| jms v185 | `CUserLocal::OnPacket` @ `0xa1208d` | 235 (`0x0EB`) | 235 | ✅ |

Closes PRD open question 2. v79 keeps its `❌` for `SHOW_COMBO` (`0x0D2`
exists there, but `ARAN_COMBO_COUNTER` is `n-a`, so nothing can drive it).

### 2.3 What the `ARAN_COMBO` temporary stat actually is

- The v83 client declares **two distinct** combo secondary stats:
  `CTS_ComboCounter` (bit ~21 in declaration order, between `CTS_Darkness`
  and `CTS_WeaponCharge`) — the Warrior/Crusader orb stat task-142
  implemented — and `CTS_ComboAbilityBuff`, whose dynamic initializer
  (`0x7912d4`) resolves to **`UINT128(1) << 68`**, sitting between
  `CTS_EventRate` and `CTS_ComboDrain`. That is exactly where
  `TemporaryStatTypeAranCombo` sits in
  `libs/atlas-packet/model/character_temporary_stat.go:162`
  (…`EventRate`, `AranCombo`, `ComboDrain`, `ComboBarrier`…). **Bit order
  confirmed; FR-5.3 holds — no new stat type, no model change.**
- `SecondaryStat::DecodeForLocal` @ `0x7859f2` decodes it as
  `Decode2` (signed short value) + `Decode4` (skill id) + `Decode4`
  (duration) — the standard non-diseased shape the model already writes.
- Its only consumers in the v83 IDB are `SecondaryStat::Reset`,
  `DecodeForLocal`, `CheckByTime`, and **`IsCalcDamageStat`**. It is a
  damage-calculation input. `DrawCombo` never reads it.
- Combo Ability's WZ node (`Skill.wz/2100.img`, tenant dump
  `GMS/83.1`) carries per level only `hs`, `x = <level>`, `y = 1`, `z = 5`,
  plus `weapon = 44` and `invisible = 1`. Max level 20. **There is no field
  whose value is 100**, and no duration.

**Conclusion (closes open question 4):** the hardcoded `100` at
`services/atlas-data/atlas.com/data/skill/reader.go:470` has no provenance in
WZ or in the client. The value is not the combo count and not a tier index.
The design changes it to `int32(e.X())` — the same treatment every sibling
Aran statup already gets (`ComboBarrier`, `ComboDrain`, `SmartKnockBack`,
`BodyPressure` all pass `e.X()`), giving 1–20 for Aran and 10 for Legend,
comfortably inside the signed short. **Residual unknown, stated plainly:** the
exact arithmetic the client's `CalcDamage` performs with the value was not
traced. Nothing server-side reads `ARAN_COMBO` (verified: the only non-test
references in the tree are the constant, the packet model entry, and
`reader.go:470`), so the change is inert server-side and strictly
better-grounded than `100`.

### 2.4 The client's own gates

`CMob::OnHit` (v84 `sub_67EC73` @ `0x67ed01`) calls `RequestIncCombo` under:

```c
v17 = CSkillInfo::GetSkillLevel(cd, job != 2000 ? 21000000 : 20000017, &lv);
v18 = GetWeaponType(equippedWeapon);           // sub_462CCA(pUser[315])
if (attacker == localUserId && v17 > 0 && v18 == 44 && damage > 0)
    RequestIncCombo();
```

There is **no job check beyond the skill-level check** — owning Combo Ability
at level > 0 *is* the job gate. The weapon gate is `type == 44`, and the
damage gate is `damage > 0` (one request per damaging hit landed on a mob).

`ClearCombo` (v83 `0x960f21`, v92 `sub_8EBFF0`, v95 `0x905660`) uses the same
`job != 2000 ? 21000000 : 20000017` selector, verified at the instruction
level in v92 (`call [vtbl+0x44]` → `sub eax, 7D0h` → `sbb`/`and 0F422Fh` →
`add 1312D11h`).

### 2.5 Decay: window, shape, and who drives it

`CUserLocal::Update` runs, every frame:

```c
if (GetSkillLevel(comboAbilityId) > 0
    && m_nCombo > 0
    && m_tLastSetCombo + W < tCur)
    ClearCombo();
```

| Version | Site | `W` |
|---|---|---|
| gms v83 | `Update` @ `0x94bdc8` | `0xBB8` = **3000 ms** |
| gms v84 | `Update` @ `0x983953` | `0xBB8` = **3000 ms** |
| gms v87 | `Update` @ `0x9c6dab` | `0xBB8` = **3000 ms** |
| gms v92 | `sub_916860` @ `0x919037` | `0xBB8` = **3000 ms** |
| gms v95 | `Update` @ `0x939c93` | **`0x1388` = 5000 ms** |
| jms v185 | `Update` @ `0xa0c0ae` | `0xBB8` = **3000 ms** |

`ClearCombo` zeroes `m_nCombo`, releases the combo layers, and calls
`CUserLocal::SendSkillCancelRequest(comboAbilityId)` — i.e. the client emits
the ordinary `CANCEL_BUFF` serverbound packet that
`services/atlas-channel/.../socket/handler/character_buff_cancel.go` already
handles.

`DrawCombo` opens with `if (m_nCombo <= 0) return;` **without releasing the
digit layers.** Sending `SHOW_COMBO 0` therefore leaves the stale digits on
screen; it cannot clear the HUD.

**Closes open question 6.** The shape is a hard reset, not a step-down. The
window is version-divergent (v95 = 5000 ms, everything else 3000 ms). Cosmic's
3 s matches retail on five of six versions.

### 2.6 Combo cap

Nothing in WZ governs it (§2.3). Client-side limits observed in `DrawCombo`:

- Tier selection is hardcoded: `< 30` → 0, `30–99` → 1, `100–199` → 2,
  `>= 200` → 3.
- At exactly 30 / 100 / 200 the client shows a "skill ready" cue for
  `21100004` / `21110004` / `21120006` respectively, each gated on
  `GetSkillLevel(id) > 0`. **These thresholds are client-hardcoded; the
  server neither sends nor configures them.**
- The digit renderer decomposes the count into a **5-slot** array
  (`m_apLayerComboDigit[5]` in the v95 PDB names; a 20-byte stack array in
  v83/v84). A 6-digit count would overrun it.

**Closes open question 5.** The server cap is a policy value with a hard
client-derived ceiling of **99999**. The design uses 99999 (five digits) as
the cap constant, documented against the digit-array width.

### 2.7 Legend `20000017`

`Skill.wz/2000.img` in the GMS 83.1 tenant dump **contains `20000017`**:
one level, `x = 10`, `y = 1`, `z = 5`, `weapon = 44`, `invisible = 1`. This
directly contradicts the note quoted in PRD §FR-5.1 (which listed job 2000's
skill set as `20001000`–`20001004` only); the full node list for `2000.img`
is `20000012, 20000014–20000018, 20000024, 20001000–20001031, 20009000–2`.

**Closes open question 8:** the Legend branch is real and reachable. Add
`LegendComboAbilityId = Id(20000017)` to
`libs/atlas-constants/skill/constants.go` with its `Skill` value and
per-version identity-table entries, following the `AranStage1ComboAbilityId`
pattern (`constants.go:3391`, `constants.go:2044`, `constants.go:2832`).
Neither id is on `tools/skill-job-id-guard.sh`'s divergent list, so direct
comparison is permitted.

### 2.8 Polearm

`item.GetWeaponType` computes `cat = (itemId / 10000) % 100` and returns
`WeaponTypePolearm` for `cat - 30 == 14`, i.e. `cat == 44`
(`libs/atlas-constants/item/constants.go:136-162`). The client's
`GetWeaponType(...) == 44` and Combo Ability's own WZ `weapon = 44` are the
same number on the same basis. **Closes open question 9:**
`item.WeaponTypePolearm` is the correct member; no new constant.

### 2.9 Milestones

**Closes open question 7.** No server work. The 30/100/200 cues are drawn by
`DrawCombo` from the count it already has, gated on locally-known skill
levels. Nothing is re-applied at milestones and nothing is sent. A single
Combo Ability buff for the lifetime of the combo is sufficient.

---

## 3. Architecture

```
client                atlas-channel                         atlas-buffs
  |                        |                                     |
  |-- melee attack ------->| attack pipeline (already fetches    |
  |                        |  character w/ Inventory+Skill        |
  |                        |  decorators, line ~745)             |
  |                        |  -> refresh ComboEligibility cache  |
  |                        |                                     |
  |-- ARAN_COMBO_COUNTER ->| AranComboCounterHandle              |
  |   (no body)            |  gates from cache (0 REST)          |
  |                        |  count = min(count+1, cap)          |
  |                        |  if 0->1: APPLY_NO_EXPIRY --------->| buff stored
  |<-- SHOW_COMBO(count) --|  (written immediately, same tick)   |
  |                        |                                     |
  |   ...3s/5s idle...     | ComboDecayTick (1 Hz)               |
  |   client ClearCombo    |  idle > window -> count = 0         |
  |-- CANCEL_BUFF -------->|  CharacterBuffCancelHandle          |
  |   (skill 21000000)     |   -> Cancel ---------------------->| buff removed
  |                        |   -> combo mirror Clear             |
```

### 3.1 Component inventory

| Component | Location | New? |
|---|---|---|
| `AranComboCounter` serverbound model | `libs/atlas-packet/character/serverbound/` | new |
| `ShowCombo` clientbound model | `libs/atlas-packet/character/clientbound/` | new |
| `LegendComboAbilityId` + identity rows | `libs/atlas-constants/skill/` | new |
| `ComboMirror` (count + last-hit + eligibility) | `services/atlas-channel/.../character/combo/` | new |
| `AranComboCounterHandleFunc` | `services/atlas-channel/.../socket/handler/character_aran_combo.go` | new |
| `ShowCombo` writer | `services/atlas-channel/.../socket/writer/` | new |
| Combo decay tick task | `services/atlas-channel/.../tasks/` | new |
| Eligibility refresh hook | `character_attack_common.go` (~line 981, beside `comboOrbTryUpdate`) | edit |
| Combo reset on Combo Ability cancel | `character_buff_cancel.go` | edit |
| `ARAN_COMBO` statup value | `services/atlas-data/.../skill/reader.go:470` | edit |
| Handler + writer entries × 6 templates | `services/atlas-configurations/seed-data/templates/` | edit |

`atlas-buffs` needs **no change at all** — a deviation from PRD §8, justified
in §5.1.

### 3.2 `ComboMirror` — the count's home

Process-local, tenant-keyed singleton with a `sync.RWMutex`, modelled exactly
on `services/atlas-channel/.../character/buff/beacon.go`'s `BeaconMirror`
(`map[uuid.UUID]map[uint32]entry`, `sync.Once` accessor).

```go
type entry struct {
    count      int32       // authoritative combo count, 0 == no combo
    lastHit    time.Time   // refreshed on every accepted increment
    eligible   bool        // gate result from the last attack/lazy fetch
    comboId    skill.Id    // 21000000 or 20000017
    comboLevel byte
    statAmount int32       // effect X() for the ARAN_COMBO statup
    checkedAt  time.Time   // eligibility freshness
}
```

Rationale for process-local rather than Redis:

- Combo lives 3–5 seconds and dies with the session; a channel restart
  losing it is indistinguishable from an idle reset.
- A session is pinned to one channel process, so there is no cross-process
  reader.
- Keeps the hot path free of both Redis and Kafka, and keeps
  `tools/redis-key-guard.sh` out of scope.

The same accepted degradation the beacon mirror documents applies and is
recorded in the type's doc comment.

### 3.3 Increment path (the hot path)

`AranComboCounterHandleFunc`:

1. Decode nothing — the model's `Decode` consumes zero bytes (FR-1.4).
2. Read the mirror entry for `(tenant, characterId)`.
   - Entry present and `checkedAt` within the eligibility TTL (60 s):
     use it. **Zero REST.**
   - Otherwise perform the one-time gate fetch (§3.5) and cache it.
3. If `!eligible`: debug-log the failing gate and return. No packet, no
   Kafka (NFR-2).
4. `next = min(count+1, comboCap)`; if `next == count`, refresh `lastHit`
   and return without a write (at cap — the client will keep asking).
5. If `count == 0 && next == 1`: emit `ApplyNoExpiry(f, characterId,
   characterId, int32(comboId), comboLevel, [{ARAN_COMBO, statAmount}])`.
   Failure is logged and swallowed (NFR-4) — the counter still advances.
6. Store `count = next`, `lastHit = now`.
7. Write `SHOW_COMBO{Count: next}` to the acting session only (FR-6.3).

Per-packet cost in steady state: one mutex-guarded map read/write and one
socket write. Kafka is touched once per combo *chain*, not once per hit.

### 3.4 Reset paths

There are three, and they all converge on "clear the mirror entry + cancel
the buff":

1. **Decay tick** (authoritative). A 1 Hz task registered in
   `atlas-channel`'s `main.go` alongside its existing tick registrations,
   spawned through `routine.Go` (NFR-7). It walks the mirror — never a
   session or tenant scan; an empty mirror is an empty walk (FR-4.5) — and
   for every entry with `count > 0 && now.Sub(lastHit) > window`, sets
   `count = 0` and emits `buff.Cancel(field, characterId, int32(comboId))`.
   **It does not send `SHOW_COMBO`** (§2.5: a 0 cannot clear the HUD; the
   client clears itself on the same schedule).
2. **Client cancel** (fast path). `CharacterBuffCancelHandle` already
   receives the `ClearCombo`-driven cancel for `21000000` / `20000017` and
   already calls `buff.Cancel`. Add one branch: clear the mirror's count for
   that character. This also self-heals the out-of-scope combo-consuming
   skills — `DoActiveSkill` calls `ClearCombo` when Combo Smash / Fenrir /
   Tempest fire, so the client's cancel resets the server count for free and
   the two never drift.
3. **Session end / field change.** Clear the entry wherever the beacon
   mirror is cleared, so a logout or map change cannot resurrect a count.

### 3.5 Eligibility

Gate set, re-derived server-side, in the order the client applies them:

| Gate | Source | PRD |
|---|---|---|
| Combo Ability learned at level > 0 (`21000000`, or `20000017` when `JobId() == job.LegendId`) | `character.Model.Skills()` via `SkillModelDecorator` | FR-2.1 + FR-2.2 |
| Equipped weapon is `item.WeaponTypePolearm` | `character.Model.Equipment()` weapon slot via `InventoryDecorator` | FR-2.3 |

`JobId()` is on the base model, so FR-2.1's explicit Aran/Legend range check
is redundant with the skill check (§2.4) — the design keeps only the skill
gate plus the Legend-id selection, matching the client exactly. Adding a
job-range gate the client does not apply would reject legitimate states
(e.g. a Legend who has the skill).

**Population (NFR-1).** The melee-attack pipeline already fetches the
character with *both* decorators at
`character_attack_common.go:745` and already calls `comboOrbTryUpdate` at
line 981. The eligibility refresh hangs there: same model, no extra fetch,
and it is inherently fresh because the only legitimate way to gain combo is
to land a melee hit. Entry TTL is 60 s; a lazy first fetch covers the cold
case (fresh login whose first combo packet beats the attack hook). Worst
case is one triple-decorator fetch per character per 60 s, and zero in steady
melee combat.

Staleness is benign: an unequip stops the *client* sending the packet, so a
stale-eligible entry only matters to a modified client, and it self-corrects
within 60 s.

### 3.6 Packet models

Both are immutable structs with `Encode` and `Decode`, in
`libs/atlas-packet`, following the existing convention.

- `serverbound.AranComboCounterRequest` — zero fields. `Decode` reads
  nothing; `Encode` writes nothing (needed for the byte fixture).
- `clientbound.ShowCombo` — one field, `count uint32`, written as a 4-byte
  little-endian value (§2.2).

**No version gates.** Both are byte-identical across all six versions, so
there is no `MajorAtLeast` call site and NFR-6 is satisfied vacuously. The
only version-divergent value in the whole feature is the idle window, and
that is config-resolved (§4).

---

## 4. Version handling

| Version | serverbound | clientbound | idle window |
|---|---|---|---|
| gms v83 | `0x0A3` | `0x0E1` | 3000 ms |
| gms v84 | `0x0A9` | `0x0E6` | 3000 ms |
| gms v87 | `0x0AD` | `0x0EF` | 3000 ms |
| gms v92 | `0x0BA` | `0x103` | 3000 ms |
| gms v95 | `0x0BD` | `0x101` | **5000 ms** |
| jms v185 | `0x09D` | `0x0EB` | 3000 ms |

Template entries, one handler and one writer per template, each inserted at
its sorted `opCode` position (`tools/template-opcode-order-guard.sh`):

```jsonc
// socket.handlers
{ "opCode": "0x0A3", "validator": "LoggedInValidator",
  "handler": "AranComboCounterHandle",
  "fname": "CUserLocal::RequestIncCombo",
  "options": { "idleResetMs": 3000 },
  "services": ["channel"] }

// socket.writers
{ "opCode": "0x0E1", "writer": "ShowCombo",
  "fname": "CUserLocal::OnIncComboResponse",
  "services": ["channel"] }
```

`idleResetMs` carries the version divergence as tenant configuration rather
than a compiled-in major-version branch, consistent with the project's
"client wire values are config-resolved" rule and with the existing
writer-`options` mechanism. `template_gms_95_1.json` gets `5000`; the other
five get `3000`. The decay task resolves it per tenant from the handler
options, falling back to 3000 ms if absent.

Both entries carry a non-empty validator / `fname` — the two documented
silent-drop traps for seed templates.

Live tenants must be reconciled to the updated templates after merge, or the
opcodes will be missing from the running socket config and the packets
silently dropped.

---

## 5. Alternatives considered

### 5.1 Count on the `ARAN_COMBO` buff stat (the PRD's FR-3) — **rejected**

The task-142 shape (seed via `APPLY_NO_EXPIRY`, advance via
`UPDATE_STAT_VALUE` `INCREMENT` with a cap) fits Warrior combo orbs, where
the orb count *is* the `COMBO` stat the client renders. It does not fit here:

- **Wrong field.** The count is delivered by `SHOW_COMBO`, not by the
  temporary stat (§2.3). The stat is a damage-calc input whose correct value
  is the skill's `x`. One field cannot be both.
- **Truncation.** The stat encodes as a signed short (`Decode2`), capping the
  representable count at 32767 against a client that renders five digits.
- **Latency and cost on a hit-frequency path.** Reading the count back
  requires channel → Kafka → atlas-buffs → `STAT_UPDATED` → Kafka → channel
  before `SHOW_COMBO` can be written. That is a Kafka round trip per melee
  hit per Aran, to answer a request/response HUD packet the channel could
  answer from memory in microseconds. `handleStatusEventStatUpdated`
  (`kafka/consumer/buff/consumer.go:197`) also re-broadcasts a buff-give on
  every update, so each hit would additionally push a temporary-stat packet
  the client does not need.

The chosen design keeps atlas-buffs' role to what it is good at — owning the
buff icon and the stat — and keeps the counter where its consumer lives.

### 5.2 Decay ticker in atlas-buffs (the PRD's FR-4) — **rejected**

Follows 5.1: with the count in atlas-channel, a ticker in atlas-buffs would
have nothing to decay, and the last-hit timestamp registry the PRD sketched
(`TenantRegistry[uint32, time.Time]` beside `poisonTicks`) would be a Redis
round trip per melee hit for state that lives 3 seconds. The channel-side
mirror carries `lastHit` inline at no cost.

### 5.3 Server-driven HUD clear via `SHOW_COMBO 0` — **rejected (impossible)**

`DrawCombo` early-returns on a non-positive count without releasing its
layers (§2.5). There is no clientbound packet that clears the counter. The
server's job is to agree with the client's timer, not to drive it. This is
why the decay task deliberately sends nothing, and why the idle window is
configured per version instead of chosen freely.

### 5.4 Gate by fetching per packet — **rejected**

Three REST reads (character, inventory, skills) at melee-hit frequency
violates NFR-1 outright. §3.5's attack-pipeline population reuses a fetch the
server already pays for.

### 5.5 Implementing combo consumption here — **out of scope, unchanged**

The PRD's non-goal stands, and §3.4's client-cancel path means the omission
is not observable as drift: `DoActiveSkill` clears the client's combo and
sends the cancel, which resets the server. Combo Smash / Fenrir / Tempest
simply do not yet *require* or *scale with* the count.

---

## 6. Testing

Unit (table-driven, Builder pattern for setup — no `*_testhelpers.go`):

- `ComboMirror`: increment, clamp at 99999, reset, per-tenant isolation,
  concurrent access under `-race`.
- Gate evaluation: Aran with polearm + skill → eligible; Legend with
  `20000017` → eligible; Aran without the skill → not; Aran with a
  non-polearm → not; non-Aran → not. Each asserts zero emissions.
- Decay: entry idle past the window resets and emits exactly one `Cancel`;
  entry inside the window is untouched; `count == 0` entries emit nothing.
- `idleResetMs` resolution: 5000 for a v95 tenant, 3000 default when the
  option is absent.
- `reader.go` statup: Combo Ability level *n* yields `ARAN_COMBO = n`;
  Legend `20000017` yields 10.

Packet fixtures (the twelve matrix cells, FR-6.4): each cell verified through
`/verify-packet` / `packet-verifier` per
`docs/packets/audits/VERIFYING_A_PACKET.md`, with a byte fixture carrying a
`packet-audit:verify` marker and a pinned evidence record. The serverbound
fixture is a zero-length body; the clientbound fixture is the 4-byte count.
Evidence for each cell is the decompilation cited in §2.1 / §2.2.

Guards, all from the repo root: `redis-key-guard`, `goroutine-guard`,
`lint.sh --check`, `template-opcode-order-guard`,
`template-duplicate-binding-guard`, `template-movement-types-guard`,
`buff-duration-guard`, `skill-job-id-guard`, plus `go test -race`,
`go vet`, `go build`, and `docker buildx bake` for every service whose
`go.mod` moves.

---

## 7. Risks

| Risk | Mitigation |
|---|---|
| Live tenants not reconciled to the new templates → packets silently dropped | Called out in §4; reconcile step is part of rollout, and the symptom (no counter at all, no server log) is documented |
| A modified client spams the op while ineligible | Gate is cache-resident; rejection costs one map read, debug-logged only (NFR-2, NFR-5) |
| Mirror lost on channel restart mid-combo | Indistinguishable from an idle reset; documented on the type, mirrors `BeaconMirror`'s accepted degradation |
| `reader.go` statup change alters behaviour somewhere unseen | Verified: no server-side reader of `ARAN_COMBO` exists; only the constant, the packet-model entry, and `reader.go:470` reference it |
| v95's 5000 ms window mis-copied to another template | Covered by the `idleResetMs` unit test and visible in one line per template |

---

## 8. Requirement traceability

| PRD | Disposition |
|---|---|
| FR-1.1 – FR-1.4 | §3.6, §4. Body confirmed empty on all six versions; no version gate needed |
| FR-2.1 – FR-2.3 | §3.5. Job-range check folded into the skill gate to match the client exactly |
| FR-2.4 / NFR-1 | §3.5. Zero REST per packet in steady state |
| FR-3.1 – FR-3.4 | **Revised.** Count moves to atlas-channel (§3.2, §5.1); the buff is still applied/cancelled and still carries `ARAN_COMBO`. Cap = 99999 from the client's 5-digit renderer (§2.6) |
| FR-4.1 – FR-4.5 | **Revised.** Decay task in atlas-channel (§3.4, §5.2); hard reset; window config-resolved per version; FR-4.4's "tell the client" is impossible and deliberately omitted (§5.3) |
| FR-5.1 | §2.7. `20000017` exists in WZ — add the constant; the PRD's doubt is resolved in favour of adding it |
| FR-5.2 | §2.3. `100` → `int32(e.X())`, with the residual unknown stated |
| FR-5.3 | §2.3. Confirmed at bit 68; no change |
| FR-6.1 | **Discharged in design.** All twelve opcodes verified; registry unchanged. v92's sender to be renamed in-IDB during execution |
| FR-6.2 – FR-6.4 | §3.6, §6 |
| NFR-2 – NFR-7 | §3.3, §3.4, §4, §7 |
| Open questions 1–9 | Closed in §2.1, §2.2, §2.1, §2.3, §2.6, §2.5, §2.9, §2.7, §2.8 |

---

## §9. Post-landing: the hidden combo variant disconnect

Found while testing the landed feature on the PR environment. The combo counter
itself was correct — it reached 7 and the client rendered every step — but the
session was destroyed on the next attack.

Channel log, session `4a43d7f1`, 113 ms:

```
13:26:51.834  Aran combo: character [1] count [7].
13:26:52.003  Character [1] is attempting a melee attack.
13:26:52.015  ERROR Character [1] attempting to attack with skill [21120009] which they do not own.
13:26:52.015  Destroying session.
13:26:52.116  Connection ended.
```

`21120009` is `AranStage4OverswingDoubleSwingId` — one of the four Aran hidden
combo variants the codebase already enumerates in
`libs/atlas-constants/skill/point_reset.go`:

| variant | parent |
| --- | --- |
| 21110007 / 21110008 (Full Swing Double/Triple Swing) | 21110002 Full Swing |
| 21120009 / 21120010 (Over Swing Double/Triple Swing) | 21120002 Over Swing |

They are never in the skill book: no SP is spent on them, they are excluded from
SP reset, and the Aran preset seeded by this task deliberately ships only the 26
non-hidden skills. The client sends the variant's id in the attack packet once
the combo count escalates the swing, so `processAttack`'s ownership gate — which
destroys the session for any unowned attack id — killed a legitimate Aran.

The gate is pre-existing on `main`. This task made it reachable: before the Aran
preset there was no way to hold Over Swing on any environment.

**Fix.** `resolveAttackSkill` (character_attack_common.go) resolves a hidden
variant through `skill.AranHiddenComboParent` and backs the attack with the
parent's `Model`, so the variant runs at the parent's level. An unowned parent
is still a destroy: the client can only produce the variant by escalating a
swing it already has, so that case is the same forged-attack signal the gate
exists to catch.

The variant's own id stays on the wire for the effect fetch. Verified against
live atlas-data (gms 83.1) — each variant carries its own per-level effect table
at the same maxLevel as its parent, so the parent's level always indexes a valid
row:

| skill | maxLevel | effect rows |
| --- | --- | --- |
| 21110002 Full Swing | 20 | 20 |
| 21110007 / 21110008 | 20 | 20 |
| 21120002 Over Swing | 30 | 30 |
| 21120009 / 21120010 | 30 | 30 |

Not verified: the exact combo count at which the client swaps in the two- and
three-swing variants. The fix does not depend on it — any combo count that
produces the variant id is handled — but it is unconfirmed in the IDB.
