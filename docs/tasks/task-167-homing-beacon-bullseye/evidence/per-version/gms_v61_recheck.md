# GMS v61 — independent recheck (task-167)

Binary: `GMS_v61.1_U_DEVM.exe.i64` (input path `E:\Programs\Nexon\IDBs_v9\GMS\v61\GMS_v61.1_U_DEVM.exe.i64`) · session: 415bf585 · identity confirmed: `mcp__ida-pro__idb_list` returned session `415bf585` with `filename: "GMS_v61.1_U_DEVM.exe.i64"`, an exact match to the expected target.

## Method

Anchors given: `CWvsContext::OnTemporaryStatReset` @0x84353A, `CWvsContext::OnTemporaryStatSet` @0x84311F. Both decompile cleanly and both read/gate an embedded `SecondaryStat` sub-object (`this+8468` inside `CWvsContext`), but neither function *itself* contains the trailer loop — they call into `SecondaryStat::DecodeForLocal` (@0x663665, local player) and, for other players, `SecondaryStat::DecodeForRemote` (@0x667C5F). `DecodeForRemote` is small enough (0x68A bytes) to decompile whole and it ends in an explicit bounded loop over a pointer array — that loop is the trailer group. I then found the *same* loop pattern inline inside `DecodeForLocal` (via a shared mask-constructor helper, `sub_66EDE6`, appearing at two call sites, 0x666D82 and 0x666E39) and a *third*, independent instance of the identical `i<6` bound in the `SecondaryStat` constructor (`sub_65F66F`), which is where I resolved concrete member types/sizes by reading the literal vtable pointers written at construction time (MSVC writes the vtable address as an immediate `mov [obj], offset off_XXXXXX`, so no runtime/virtual ambiguity — these calls looked virtual at the call site but were staticaly resolvable from the constructor).

## Findings

### Member count: 6 — three independent bounded loops, all hard-coded `6`

1. `SecondaryStat::DecodeForRemote` (0x667C5F), tail:
   ```
   v51 = 0;
   v52 = this + 598;
   do {
     v53 = sub_66EDE6(v57, v51);
     v54 = (UINT128 *)UINT128::operator&(v56, v53);
     if ( (unsigned __int8)UINT128::compareTo(v54) )
       (*(void (__thiscall **)(_DWORD, CInPacket *))(*(_DWORD *)*v52 + 24))(*v52, a3);
     ++v51;
     ++v52;
   } while ( v51 < 6 );
   ```
2. `SecondaryStat::DecodeForLocal` (0x663665), disasm 0x666D66–0x666E6A: `and [ebp+var_14],0` (i=0) ... `cmp [ebp+var_14], 6` / `jl loc_666D73` — identical 6-iteration bound, base pointer `lea eax,[esi+958h]` (958h = 2392 = 598×4, same slot array).
3. `SecondaryStat`'s constructor `sub_65F66F` (0x65F66F):
   ```
   for ( i = 0; i < 6; ++i ) {
     ... this[i + 598] = <alloc + placement-construct> ...
   }
   sub_65FCA2(this);
   ```
   Six `operator new`-style allocations (`ZAllocEx<ZAllocAnonSelector>::Alloc`) of sizes 0x28, 0x28, 0x28, 0x1C, 0x2C, 0x20 land in `this[598..603]`.

All three loop bounds agree: **6 members**, stored as pointers at `this[598]`..`this[603]` (byte offsets 0x958–0x96C) inside `SecondaryStat`.

### Members in order (index, offset, DecodeForClient addr, block size, name/confidence)

Each `DecodeForClient` address below was resolved by reading the literal vtable-pointer store in the constructor, then reading the vtable's slot-6 (offset +0x18) function pointer directly out of the binary — not runtime-inferred:

| i | slot | ctor | alloc | vtable (`*this=`) | DecodeForClient | wire bytes | name (confidence) |
|---|---|---|---|---|---|---|---|
| 0 | `this[598]` | `sub_65F94E` | 0x28 (40B) | `off_8EA5B0` | `0x66EC19` | base(12)+Decode2(2)=**14** | EnergyCharge (positional) |
| 1 | `this[599]` | `sub_65FA09` | 0x28 (40B) | `off_8EA650` | `0x66ED94` | base(12)+Decode2(2)=**14** | DashSpeed (positional, Dash pair) |
| 2 | `this[600]` | `sub_65FA09` (same fn as i=1) | 0x28 (40B) | `off_8EA650` (same) | `0x66ED94` (same) | **14** | DashJump (positional, Dash pair — byte-identical to i=1, order not distinguishable from code alone) |
| 3 | `this[601]` | `sub_65F8F6` | 0x1C (28B) | `off_8EA574` | `0x66EB3D` → `sub_66E9B6` (base only, no extra fields) | **12** | **RideVehicle** — independently confirmed, see below |
| 4 | `this[602]` | `sub_65F9E2` | 0x2C (44B) | `off_8EA614` | `0x66E8EF` | base(12)+Decode4(4)+Decode2(2)=**18** | SpeedInfusion (positional) |
| 5 | `this[603]` | inline in `sub_65F66F` (`*v3=&off_8EA504; v3[6]=&off_8EA4E8;`) | 0x20 (32B) | `off_8EA504` | `0x65F840` → calls `sub_66EB3D`(=base 12) + `CInPacket::Decode4` | base(12)+dwMobId(4)=**16** | **GuidedBullet** — independently confirmed, see below |

Shared base decode, `sub_66E9B6` (called by every one of the 6, directly or via wrapper `sub_66EB3D`):
```
CInPacket::DecodeBuffer((int)a2, this + 1, 4u);   /*0x66e9df*/
CInPacket::DecodeBuffer((int)a2, this + 2, 4u);   /*0x66e9ed*/
result = CInPacket::Decode4(a2);                  /*0x66e9f5*/
this[3] = result;                                 /*0x66e9fc*/
```
= 4+4+4 = 12 bytes, always present. Individual members append 0–6 more bytes on top (table above).

**Independent (non-positional) confirmation of i=3=RideVehicle and i=5=GuidedBullet.** `CWvsContext::OnTemporaryStatReset`/`OnTemporaryStatSet` read the embedded `SecondaryStat` at `this+8468` and directly index the *same* slots:
```
v3 = (char *)this + 8468;                                                 /*0x843571, OnTemporaryStatReset*/
v6 = (_DWORD *)sub_4C12B4(*((_DWORD *)v3 + 601));                         /*0x843593*/
CUser::ShowRideVehicleEffect(v5, *v6);                                    /*0x84359c*/
...
v8 = (TemporaryStat_GuidedBullet *)*((_DWORD *)v3 + 603);                 /*0x8435d2*/
MobID = TemporaryStat_GuidedBullet::GetMobID(v8);                        /*0x8435e5*/
Reason = (_DWORD *)TemporaryStatBase<long>::GetReason(v8);               /*0x8435e8*/
CMobPool::ResetGuidedMob(v9, *Reason, MobID);                            /*0x8435f1*/
```
Index 601 (=598+3) drives `ShowRideVehicleEffect`; index 603 (=598+5) is typed `TemporaryStat_GuidedBullet` and calls its named `GetMobID`/`GetReason` accessors — an exact match to slots 3 and 5 of the constructor loop.

### Summed trailer length

14 + 14 + 14 + 12 + 18 + 16 = **88 bytes** (if all 6 members are active/present simultaneously).

### GuidedBullet mask-bit shift

**Raw client basis: shift 64.** Three independent derivations agree:
1. Loop position: GuidedBullet is `i=5`, the last of the 6-iteration loop.
2. Mask-constructor formula: `sub_66EDE6(a2)` = `UINT128(1) << (a2+59)` (verified by decompiling `sub_66EDE6` and its callee `sub_6F3940`, a UINT128 left-shift). The dedicated global `unk_97C2E0`, used directly in `OnTemporaryStatReset`/`OnTemporaryStatSet` to gate the GuidedBullet-specific logic, is built by its own static-initializer `sub_840E0B`:
   ```
   v0 = sub_66EDE6((UINT128 *)v2, 5);
   return UINT128::UINT128((UINT128 *)&unk_97C2E0, v0, 0x80u);
   ```
   → shift = 5+59 = **64**.
3. Cross-check: `unk_97C300` (used to gate `ShowRideVehicleEffect`, i.e. the RideVehicle/i=3 slot) is built the same way, `sub_840DE1` → `sub_66EDE6(3)` → shift 62 = 59+3, matching i=3 exactly. Same formula, consistent across both cross-checked members.

### Raw-vs-registry offset on this version

Determined, with a caveat on scope. Reading `libs/atlas-packet/model/character_temporary_stat.go`'s `buildCharacterTemporaryStatRegistry`: for GMS < 87 (v61 included — none of the `post87`/`gmsV95Plus`/`extended`/`jms` gates fire differently for v61 vs v83), the two-state group is registered starting at shift 82: EnergyCharge=82, DashSpeed=83, DashJump=84, MonsterRiding/RideVehicle=85, SpeedInfusion=86, HomingBeacon/GuidedBullet=**87** (and `TemporaryStatTypeUndead` is registered next, unconditionally, at 88 — see "conflict" note below).

Comparing raw (IDA) to registry (source) shift for every member of the group:

| member | raw (IDA) | registry (source) | offset |
|---|---|---|---|
| EnergyCharge | 59 | 82 | +23 |
| DashSpeed | 60 | 83 | +23 |
| DashJump | 61 | 84 | +23 |
| RideVehicle | 62 | 85 | +23 |
| SpeedInfusion | 63 | 86 | +23 |
| GuidedBullet | 64 | 87 | **+23** |

The offset is a constant **+23** (registry = raw + 23) across the whole cluster — internally consistent. Caveat: this is read from the current registry *source code*, not independently re-derived from the binary for the ~82 non-two-state stats that precede the group (WeaponAttack..SoulStone) — I did not decompile those ~82 individual mask-test call sites in `OnTemporaryStatSet`/`OnTemporaryStatReset`/`DecodeForLocal` to confirm v61's client actually places 82 (not 59) ordinary flag bits before its own two-state group. The two-state-internal offset (+23, all 6 members self-consistent) is solid; whether "+23" also holds for bit 0–58 is UNVERIFIED from this session — flagging as a real open question, not asserting it either way.

### Trailer read style: per-member mask-gated (not unconditional)

`SecondaryStat::DecodeForLocal` (0x663665), disasm 0x666D66–0x666E6A:
```
666d66  and [ebp+var_14], 0            ; i = 0
666d6a  lea eax, [esi+958h]            ; &this[598]
666d73  mov eax, [eax]                 ; this[598+i] (member pointer)   <- loop head
666d82  call sub_66EDE6                ; UINT128(1) << (i+59)
666d91  call UINT128::operator&        ; decoded_mask & (1<<(i+59))
666d98  call UINT128::compareTo        ; != 0 ?
666d9f  jz   loc_666E5F                ; SKIP member entirely if bit clear
666dab  call dword ptr [eax+18h]       ; DecodeForClient — only reached if bit set
...
666e66  cmp [ebp+var_14], 6
666e6a  jl   loc_666D73
```
Identical structure in `DecodeForRemote` (0x667C5F, quoted in Method) and in `SecondaryStat::Reset` (0x662704, same test-then-branch pattern before each `_ZtlSecureTear<long>` reset call, though Reset's own trailer section wasn't independently re-walked this pass). When the mask bit is clear, `DecodeForClient` is never invoked and zero bytes are consumed for that member — the trailer is **not** a fixed-length blob; its wire length varies with which members are flagged active.

## Confidence

- **Member count (6):** triple cross-checked — `DecodeForRemote`'s loop, `DecodeForLocal`'s loop, and the constructor's allocation loop all independently hard-code `6`. High confidence.
- **GuidedBullet raw shift (64):** triple cross-checked — loop position (i=5), direct mask-constant formula (`sub_66EDE6(5)`), and the CWvsContext-offset correlation (slot 603 ⇒ named `TemporaryStat_GuidedBullet` accessors). High confidence.
- **RideVehicle raw shift (62):** same triple-cross-check pattern (i=3, `sub_66EDE6(3)`, slot 601 ⇒ `ShowRideVehicleEffect`). Used as a second control point to validate the shift formula, not just trust it for GuidedBullet alone.
- **Block sizes (14/14/14/12/18/16, sum 88):** each `DecodeForClient` address was reached via a *literal* vtable-pointer store in the constructor (not a runtime-only virtual dispatch) and then decompiled directly — not estimated, not left `UNVERIFIED`, since the static vtable assignment made every one of the 6 statically resolvable in this binary.
- **Read style (per-member mask-gated):** directly evidenced by the `compareTo`/`jz` branch guarding the vtable call in two independent functions (`DecodeForLocal`, `DecodeForRemote`).
- **Registry-basis shift / +23 offset:** single-sourced from reading `character_temporary_stat.go` (not IDA-cross-checked against the ~82 preceding ordinary flag bits in the binary) — reported with that caveat explicitly, per the task's basis-labeling requirement.
- After completing the above independently via direct tool calls, I found a pre-existing file at `docs/tasks/task-167-homing-beacon-bullseye/evidence/per-version/gms_v61.md` reaching the same conclusions (6 members, sizes 14/14/14/12/18/16, GuidedBullet shift 64, per-member mask-gated, offsets 601/603 for RideVehicle/GuidedBullet). This is reported as an external cross-check found *after* independent derivation, not as an input to it.
