# Race-Index → Job Findings (task-283)

Derived from the client binaries listed below. Every row cites the function and address it was
read from. A row that could not be derived is `status: unverified` with a reason; no row is
populated from remembered MapleStory knowledge.

| version key | binary | md5 | IDB |
|---|---|---|---|
| gms_12 | — | — | — (no IDB, no export) |
| gms_v48 | GMS_v48_1_DEVM.exe | (export records none) | GMS_v48_1_DEVM.exe.i64 |
| gms_v61 | GMS_v61.1_U_DEVM.exe | (export records none) | GMS_v61.1_U_DEVM.exe.i64 |
| gms_v72 | GMS_v72.1_U_DEVM.exe | (export records none) | GMS_v72.1_U_DEVM.exe.i64 |
| gms_v79 | GMS_v79_1_DEVM.exe | (export records none) | GMS_v79_1_DEVM.exe.i64 |
| gms_v83 | MapleStory_dump.exe (v83 Me) | 80ff438ced539b831f0d2ed95099275d | MapleStory_dump.exe.i64 |
| gms_v84 | GMS_v84.1_U_DEVM | (export records none) | GMS_v84.1_U_DEVM.i64 |
| gms_v87 | GMSv87_4GB.exe | 2e692f3ab5078e04138d264f8ea1e668 | GMSv87_4GB.exe.i64 |
| gms_v92 | GMS_v92_1_DEVM.exe | bdef16653b92eefca2361fd5668cc509 | GMS_v92_1_DEVM.exe.i64 |
| gms_v95 | GMS_v95.0_U_DEVM.exe | 3c71fd8872d5efbe16183ae8c51f887d | GMS_v95.0_U_DEVM.exe.i64 |
| gms_jms_185 | MapleStory_dump_SCY.exe (JMS v185.1) | af6652ff9b7c549341f35e3569d7564a | MapleStory_dump_SCY.exe.i64 |

## Method common to every column

Three independent facts were read per binary, and only their agreement was accepted:

1. **Does the creation request even carry a race index?** — `CLogin::SendNewCharPacket`, read
   field by field. This is the wire contract the server sees.
2. **How many race ordinals does the client branch on?** — the race switch in `CLogin::Update`
   (or its unnamed equivalent), which constructs a different name-entry dialog per race.
3. **Which class is each ordinal?** — the dialog class constructed per case. `gms_v95` and
   `gms_v83` carry the class symbols (`CUINewCharNameSelectCygnus/Normal/Aran/Evan/Res`). The
   other IDBs are stripped in the UI region, so each unnamed constructor was fingerprinted by
   its `CWnd::CreateWnd(z, x, y, width, height, …)` immediates against the *named* v83/v95
   constructors. The fingerprints are distinct per class and stable across versions:

   | class (named in v83 / v95) | z | y | width | x formula |
   |---|---|---|---|---|
   | `CUINewCharNameSelectCygnus` | 113 (114 from v84) | 203 | 226 | fixed `-2613` |
   | `CUINewCharNameSelectNormal` (Explorer) | 109 | 201 | 224 | `-2613 - 600*uiRace` |
   | `CUINewCharNameSelectAran` | 104 (109 on v79) | 207 | 276 | `-2603/-3802/-3813 - 600*uiRace` |
   | `CUINewCharNameSelectEvan` | 109 | 201 | 224 | `-2613 - 600*uiRace` |
   | `CUINewCharNameSelectRes` | 98 | 212 | 245 | `-2616 - 600*uiRace` |

   Note the Explorer and Evan dialogs share a fingerprint (checked directly in v95:
   `CUINewCharNameSelectNormal` at `0x5f4080` and `CUINewCharNameSelectEvan` at `0x5f42e0` emit
   the identical `CreateWnd(109, -2613 - 600*v3, 201, 0xE0, 10, …)`). Geometry therefore
   **cannot** separate Explorer from Evan; where that separation was needed it came from the
   switch ordinal plus the fixed `-3213` literal that v83's named `Normal` uses (i.e. race 1).

## gms_v95

Race field: `CLogin+0x240` (`m_nCurSelectedRace`), sub-job `CLogin+0x244` (int16), both confirmed
by `CLogin::ResetRaceAndSubJob` at `0x5d1cb0` (`mov dword [this+240h],0` / `mov [this+244h],ax`).
Wire: `CLogin::SendNewCharPacket` `0x5d7bd0` — opcode 22 = `EncodeStr(name)`, `Encode4(race)`,
`Encode2(subJob)`, 8 x `Encode4(AL)`, `Encode1(gender)`.

| raceIndex | subJobIndex | class | job id | IDA function | address | status | notes |
|---|---|---|---|---|---|---|---|
| 0 | 0 | Resistance / Citizen | not encoded by client | `CLogin::Update` -> `CUINewCharNameSelectRes` ctor | switch `0x5df541`, case-0 arm `0x5df5fd`, ctor call `0x5df61f` | verified | Slot is drawn — see FR-12 below |
| 1 | 0 | Explorer | not encoded by client | `CLogin::Update` -> `CUINewCharNameSelectNormal` | case-1 arm `0x5df586`, ctor call `0x5df5a8` | verified | |
| 1 | 1 | Explorer, sub-job 1 (Dual Blade) | not encoded by client | `CUINewCharRaceSelect::SelectRaceButton` | `0x5f4d60` | verified | Button 0 sets `m_nSelectedRace = 1` and `m_nSelectedSubJob = 1`; every other button sets sub-job 0 |
| 2 | 0 | Cygnus Knight | not encoded by client | `CLogin::Update` -> `CUINewCharNameSelectCygnus` | case-2 arm `0x5df556`, ctor call `0x5df57c` | verified | |
| 3 | 0 | Aran | not encoded by client | `CLogin::Update` -> `CUINewCharNameSelectAran` | case-3 arm `0x5df5af`, ctor call `0x5df5cd` | verified | |
| 4 | 0 | Evan | not encoded by client | `CLogin::Update` -> `CUINewCharNameSelectEvan` | case-4 arm `0x5df5d4`, ctor call `0x5df5f6` | verified | |

### Method
`CLogin::Update` at `0x5dee90` loads `[esi+240h]`, bounds-checks it (`cmp eax, edi` / `ja` with
`edi == 5`) and dispatches through `jpt_5DF54F[eax*4]` — a five-case jump table. IDA labels each
arm with its case number in the listing (`jumptable 005DF54F case 0` … `case 4`), and each arm
allocates `0xA0` bytes and calls exactly one `CUINewCharNameSelect*` constructor. That labelling
is the ordinal evidence; the constructor symbol is the class evidence.

The carousel *entry* side agrees: `CUINewCharRaceSelect::SelectRaceButton` (`0x5f4d60`) maps
button index -> `(race, subJob)` as `0 -> (1,1)`, `1 -> (1,0)`, `2 -> (2,0)`, `3 -> (3,0)`,
`4 -> (4,0)`, `5 -> (0,0)`, then copies both into `m_pLogin->m_nCurSelectedRace` /
`m_nCurSelectedSubJob`. `CLogin::ConvertSelectedRaceToUIRace` (`0x5d1c30`) maps race -> screen
column `0->4, 1->1, 2->0, 3->3, 4->2`, i.e. the drawn left-to-right order is Cygnus, Explorer,
Evan, Aran, Resistance — the drawing order is *not* the ordinal order.

### Lead comparison (FR-7)
Independently derived ordering: `0 = Resistance, 1 = Explorer (subJob 1 = Dual Blade),
2 = Cygnus, 3 = Aran, 4 = Evan`. The lead claimed exactly that, at `CLogin::Update 0x5dee90`.
**Confirmed** — the derivation above was reached from `SelectRaceButton` and the `Update` jump
table, and the address the lead cites is the same function.

## gms_jms_185

Race field: `CLogin+0x238`, sub-job `CLogin+0x23C` (int16) — `CLogin::ResetRaceAndSubJob`
`0x672826` (`and dword [ecx+238h], 0` / `and word [ecx+23Ch], 0`).
Wire: `CLogin::SendNewCharPacket` `0x66e2ab` — opcode 0x0B = `EncodeStr(name)`, `Encode4(race)`,
`Encode2(subJob)`, **6** x `Encode4(item)`, **no gender byte**.

| raceIndex | subJobIndex | class | job id | IDA function | address | status | notes |
|---|---|---|---|---|---|---|---|
| 0 | 0 | unverified | not encoded by client | `CLogin::Update` race switch | `0x66c17f`, arm `0x66c1ff` | ordinal verified, class unverified | Ordinal 0 exists and constructs a distinct name dialog; the class is not named in this IDB and was not fingerprinted within budget |
| 1 | 0 | unverified | not encoded by client | `CLogin::Update` -> `sub_68AEDE` | `0x66c17f`, arm `0x66c1df` | ordinal verified, class unverified | as above |
| 2 | 0 | unverified | not encoded by client | `CLogin::Update` -> `sub_68B1C6` | `0x66c17f`, arm `0x66c1bb` | ordinal verified, class unverified | as above |
| 3 | 0 | unverified | not encoded by client | `CLogin::Update` -> `sub_68B4B3` | `0x66c17f`, arm `0x66c197` | ordinal verified, class unverified | as above |
| 1 | 1 | — | — | — | — | unverified | The packet carries a live sub-job field, but the race-select button handler was not reached within this task's tool budget, so no (1,1) slot is confirmed or denied |

### Method
`CLogin::Update` at `0x66c17f` reads `[esi+238h]` and runs a four-arm `sub/dec/dec/dec` compare
chain (`sub eax,0; jz`, `dec; jz`, `dec; jz`, `dec; jnz default`) — **exactly four race ordinals,
0..3**. This is the same four-race shape as gms_84/87/92, and explicitly **not** the five-race
shape of gms_95. Each arm allocates `0x9C` and calls a distinct constructor.

Suggestive only, recorded as a note and used as no part of the derivation:
`template_jms_185_1.json` seeds the same four `(0..3, 0)` rows plus a `(1,1)` row, i.e. it was
copied from the gms_95 shape. The race count matches; the class identities have not been checked.
Continuation point: `CConfirmRaceDlg::SetOption` at `0x68ed14` and `CUINewCharRaceSelect` at
`0x68980a`.

## gms_v92

Race field: `CLogin+0x224`, sub-job `CLogin+0x228` (int16) — `CLogin::SendNewCharPacket`
`0x5ce1e0`, opcode 0x16 = `EncodeStr(name)`, `Encode4([this+0x224])`, `Encode2([this+0x228])`,
8 x `Encode4(AL)`, `Encode1(gender)`. The sub-job is read from the member, **not** hard-coded.

| raceIndex | subJobIndex | class | job id | IDA function | address | status | notes |
|---|---|---|---|---|---|---|---|
| 0 | 0 | Cygnus Knight | not encoded by client | `sub_5D5680` race switch -> `sub_5EA490` | switch `0x5d5cad` (jumptable `0x5d5cbc`), case 0 -> ctor `0x5ea490` | verified | `sub_5EA490` emits `CreateWnd(114, -2613, 203, 226, 10)` — the Cygnus fingerprint |
| 1 | 0 | Explorer | not encoded by client | `sub_5D5680` -> `sub_5EA5B0` | case 1 -> ctor `0x5ea5b0` | verified | `CreateWnd(109, -2613 - 600*race, 201, 224, 10)` — the Normal/Explorer fingerprint; corroborated by the `race - 1 == 0` guard at `0x5d5b65` constructing `sub_5EB010` with the same geometry |
| 2 | 0 | Aran or Evan (one of the two) | not encoded by client | `sub_5D5680` -> `sub_5EA6E0` | case 2 -> ctor `0x5ea6e0` | ordinal verified, class unverified | Class not fingerprinted within budget |
| 3 | 0 | Aran or Evan (the other) | not encoded by client | `sub_5D5680` -> `sub_5EA820` | case 3 -> ctor `0x5ea820` | ordinal verified, class unverified | as above |
| 1 | 1 | — | — | — | — | unverified | Sub-job is transmitted live, so a non-zero sub-job is *possible* on v92; the race-select button handler that would set it was not located within budget |

### Method
`sub_5D5680` (the v92 `CLogin::Update` equivalent; the symbol is stripped) contains two race
switches, both `cmp eax, 3 / ja default / jmp jpt[eax*4]` — **four cases, ordinals 0..3**
(`0x5d5983` and `0x5d5cad`). `sub_5C8150` (the `ConvertSelectedRaceToUIRace` equivalent) maps
race -> screen column with `2 -> 3` and `3 -> 2` swapped and 0/1 identity, i.e. the same UI
transposition v84/v87 use.

## gms_v87

Race field: `CLogin+0x214`. Wire: `CLogin::SendNewCharPacket` `0x62f603`, opcode 0x16 =
`EncodeStr(name)`, `Encode4([this+0x214])`, **`Encode2(0)` — the sub-job is a hard-coded zero**,
8 x `Encode4(AL)`, `Encode1(gender)`.

| raceIndex | subJobIndex | class | job id | IDA function | address | status | notes |
|---|---|---|---|---|---|---|---|
| 0 | 0 | Cygnus Knight | not encoded by client | `CLogin::Update` -> `sub_64E969` | switch `0x62c5c8`, ctor `0x64e969` | verified | `CreateWnd(114, -2613, 0xCB, 226, 10)` = the v83 `CUINewCharNameSelectCygnus` fingerprint |
| 1 | 0 | Explorer | not encoded by client | `CLogin::Update` -> `sub_64ED01` | switch `0x62c5c8`, ctor `0x64ed01` | verified | `CreateWnd(109, -2613 - 600*race, 0xC9, 224, 10)`; at race 1 the x literal is `-3213`, byte-identical to v83's named `CUINewCharNameSelectNormal` |
| 2 | 0 | Aran | not encoded by client | `CLogin::Update` -> `sub_64EFDD` | switch `0x62c5c8`, ctor `0x64efdd` | verified | `CreateWnd(104, -2603 - 600*v4, 0xCF, 276, 10)` — z/y/width identical to v83's named `CUINewCharNameSelectAran` |
| 3 | 0 | fourth race (Evan slot) | not encoded by client | `CLogin::Update` -> `sub_64F2CA` | switch `0x62c5c8`, ctor `0x64f2ca` | ordinal verified, class unverified | Distinct class (own vftables `off_B9B334/off_B9B2E8/off_B9B2E4`, own singleton `dword_CA0588`). Its geometry equals the Explorer/Evan shared fingerprint, so geometry cannot name it, and there is no symbol or RTTI string in this IDB. It is the ordinal that first appears at v84 and the only one left once 0/1/2 are pinned |

### Method
`CLogin::Update` reads `[esi+214h]` at `0x62c5c8` and runs a four-arm `sub/dec/dec/dec` chain
(arms at `0x62c650` = case 0, `0x62c629` = case 1, `0x62c606` = case 2, `0x62c5df` = case 3 —
descending case order by ascending address, the layout confirmed against v83 below). A second,
structurally identical switch at `0x62c3f7` selects the avatar-step dialogs (alloc `0x148`).

## gms_v84

Race field: `CLogin+0x214`. Wire: `CLogin::SendNewCharPacket` `0x60cdf0`, opcode 22 =
`EncodeStr(name)`, `Encode4([this+0x214])`, 8 x `Encode4(AL)`, `Encode1(gender)` — **no sub-job
field at all**.

| raceIndex | subJobIndex | class | job id | IDA function | address | status | notes |
|---|---|---|---|---|---|---|---|
| 0 | 0 | Cygnus Knight | not encoded by client | `CLogin__Update` -> `sub_62B89E` | switch `0x609e9f`, arm `0x609f27`, ctor `0x62b89e` | verified | `CreateWnd(114, -2613, 0xCB, 226, 10)` = Cygnus fingerprint |
| 1 | 0 | Explorer | not encoded by client | `CLogin__Update` -> `sub_62BC1E` | arm `0x609f00`, ctor `0x62bc1e` | verified | `CreateWnd(109, -2613 - 600*race, 0xC9, 224, 10)` |
| 2 | 0 | Aran | not encoded by client | `CLogin__Update` -> `sub_62BEEE` | arm `0x609edd`, ctor `0x62beee` | verified | `CreateWnd(104, -2603 - 600*v4, 0xCF, 276, 10)` = Aran fingerprint |
| 3 | 0 | fourth race (Evan slot) | not encoded by client | `CLogin__Update` -> `sub_62C1CF` | arm `0x609eb6`, ctor `0x62c1cf` | ordinal verified, class unverified | Same situation as gms_v87 race 3 — distinct class, geometry shared with Explorer, no symbol |

### Method
`CLogin__Update` reads `[esi+214h]` at `0x609e9f`; `sub eax,ebx(0)/jz`, `dec/jz`, `dec/jz`,
`dec/jnz default` — four ordinals. Every constructor is the same shape as the gms_v87 pair,
including the `2 <-> 3` UI transposition inside `sub_62BEEE` and `sub_62C1CF`.

## gms_v83

Race field: `CLogin+0x214`. Wire: `CLogin::SendNewCharPacket` `0x5f7e7a`, opcode 0x16 =
`EncodeStr(name)`, `Encode4(m_nCurSelectedRace)`, 8 x `Encode4(AL)`, `Encode1(gender)` — **no
sub-job field**.

| raceIndex | subJobIndex | class | job id | IDA function | address | status | notes |
|---|---|---|---|---|---|---|---|
| 0 | 0 | Cygnus Knight | not encoded by client | `CLogin::Update` -> `CUINewCharNameSelectCygnus` | switch `0x5f4f26`, arm `0x5f505e`, ctor call `0x5f50de` | verified | Class symbol present in this IDB |
| 1 | 0 | Explorer | not encoded by client | `CLogin::Update` -> `CUINewCharNameSelectNormal` | arm `0x5f4fd0`, ctor call `0x5f5054` | verified | Class symbol present |
| 2 | 0 | Aran | not encoded by client | `CLogin::Update` -> `CUINewCharNameSelectAran` | arm `0x5f4f42`, ctor call `0x5f4fc6` | verified | Class symbol present |
| 3 | 0 | — | — | — | — | absent | The switch has only three arms; ordinal 3 falls to the default case at `0x5f50e7` |

### Method
This is the anchor column: `CLogin::Update` (`0x5f4c16`) reads `[esi+214h]` at `0x5f4f26` and
branches `jz 0x5F505E` (case 0), `dec/jz 0x5F4FD0` (case 1), `dec/jnz 0x5F50E7` (default) with
case 2 falling through at `0x5f4f42`. Each arm calls a **named** `CUINewCharNameSelect*` ctor,
which both fixes the pre-Big-Bang ordering and establishes the descending-case/ascending-address
arm layout and the `CreateWnd` fingerprints reused for the stripped v79/v84/v87/v92 IDBs.

## gms_v79

Race field: `CLogin+0x1E0`. Wire: `CLogin::SendNewCharPacket` `0x5ccfa4`, opcode 22 =
`EncodeStr(name)`, `Encode4([this+0x1E0])`, 8 x `Encode4(AL)`, `Encode1(gender)` — **no sub-job**.

| raceIndex | subJobIndex | class | job id | IDA function | address | status | notes |
|---|---|---|---|---|---|---|---|
| 0 | 0 | Cygnus Knight | not encoded by client | `CLogin__Update` -> `sub_5E9E5A` | switch `0x5ca641`, arm `0x5ca779`, ctor `0x5e9e5a` | verified | `CreateWnd(113, -2613, 203, 226, 10)` — identical to v83's named Cygnus ctor |
| 1 | 0 | Explorer | not encoded by client | `CLogin__Update` -> `sub_5EA1CB` | arm `0x5ca6eb`, ctor `0x5ea1cb` | verified | `CreateWnd(109, -3213, 201, 224, 10)` — identical to v83's named Normal ctor |
| 2 | 0 | Aran-family dialog | not encoded by client | `CLogin__Update` -> `sub_5EA47D` | arm `0x5ca65d`, ctor `0x5ea47d` | ordinal verified, class by geometry only | `CreateWnd(109, -3813, 207, 276, 10)`; y = 207 and width = 276 are the Aran fingerprint (z differs: 109 here vs 104 from v83 on) |
| 3 | 0 | — | — | — | — | absent | Three arms only; ordinal 3 falls to the default case at `0x5ca802` |

### Method
`CLogin__Update` reads `[esi+1E0h]` at `0x5ca641`, three-arm `sub/dec/dec` chain. There is no
`ConvertSelectedRaceToUIRace`-style transposition on v79: the per-race x literals are the plain
`-2613 - 600 * raceIndex` sequence (`-2613`, `-3213`, `-3813`), so drawing order equals ordinal
order.

## gms_v72

| raceIndex | subJobIndex | class | job id | IDA function | address | status | notes |
|---|---|---|---|---|---|---|---|
| — | — | Explorer only; **no race index exists on this version** | not encoded by client | `CLogin::SendNewCharPacket` | `0x5b219a` | verified | See method |

### Method
`CLogin::SendNewCharPacket` at `0x5b219a` builds opcode 22 as `EncodeStr(name)`,
8 x `Encode4(AL)`, `Encode1([this+484] = gender)` — and nothing else. There is **no** `Encode4`
of a race member and **no** `Encode2` of a sub-job. The v72 create-character request carries no
race selector at all, so no `(raceIndex, subJobIndex)` carousel is meaningful on this version.

## gms_v61

| raceIndex | subJobIndex | class | job id | IDA function | address | status | notes |
|---|---|---|---|---|---|---|---|
| — | — | Explorer only; **no race index exists on this version** | not encoded by client | `CLogin::SendNewCharPacket` | `0x5653e9` | verified | See method |

### Method
`CLogin::SendNewCharPacket` at `0x5653e9` builds opcode 22 as `EncodeStr(name)`,
8 x `Encode4(AL)`, `Encode1(gender)`, then four `Encode1` bytes from the caller's array. No race
field, no sub-job field.

## gms_v48

| raceIndex | subJobIndex | class | job id | IDA function | address | status | notes |
|---|---|---|---|---|---|---|---|
| — | — | Explorer only; **no race index exists on this version** | not encoded by client | `CLogin::SendNewCharPacket` | `0x500545` | verified | See method |

### Method
`CLogin::SendNewCharPacket` at `0x500545` builds **opcode 21** (not 22) as `EncodeStr(name)`,
8 x `Encode4(AL)`, `Encode1(gender)`, then four `Encode1` bytes. No race field, no sub-job field.

## gms_12

| raceIndex | subJobIndex | class | job id | IDA function | address | status | notes |
|---|---|---|---|---|---|---|---|
| 1 | 0 | Explorer | 0 (BeginnerId) | — | — | unverified | No IDA export and no IDB. Its lone (1,0) slot is present in every candidate mapping, so it is insensitive to the ambiguity. |

## Open questions resolved

- **Resistance selectable on v95.0?** **Yes.** Evidence, in four independent places:
  (a) `CUINewCharRaceSelect::SelectRaceButton` (`0x5f4d60`) has a `case 5u` arm that sets
  `m_nSelectedRace = 0` — the Resistance ordinal — with no availability guard;
  (b) `CUINewCharRaceSelect::OnKey` (`0x5f94d0`) cycles the selection over button indices 0..5
  unconditionally in both directions, so index 5 is reachable by keyboard with no feature flag;
  (c) `CUINewCharRaceSelect::OnCreate` (`0x5f81c0`) contains **six** identical button-creation
  blocks (at `0x5f8240`, `0x5f835e`, `0x5f843a`, `0x5f8514`, `0x5f85f0`, `0x5f86cc`) and a
  `cmp`-instruction sweep of the whole function shows no comparison against any configuration
  global that could suppress one of them — every `cmp` in that region is a null check
  (`cmp eax, ebp` with `ebp == 0`) or a COM `HRESULT` check;
  (d) `CLogin::Update` `0x5df5fd` constructs `CUINewCharNameSelectRes` for race 0.
  **The beginner job id the client expects for it could not be read: the client never carries a
  job id on this path** (see the next bullet). -> this applies **FR-13** (Resistance is offered);
  the job id itself must come from a non-IDA source.
- **Dual Blade creation job id?** **Outcome 2, with a correction to its premise.** On gms_v95 the
  client offers a `(race 1, subJob 1)` slot — `SelectRaceButton` `0x5f4d60` button 0 sets
  `m_nSelectedRace = 1, m_nSelectedSubJob = 1` — and `CLogin::SendNewCharPacket` `0x5d7bd0`
  transmits that pair and **no job id**. `CConfirmRaceDlg::SetOption` `0x5f75b0` also treats it as
  `nRace == 1` with `nSubJob != 0`, loading a different canvas (`StringPool` id `0x15CC`) than
  plain Explorer (`0x15C8`). So the creation identity is "Explorer race with sub-job marker 1",
  not a distinct creation job id. **No distinct creation job id exists in the client on any
  version examined.** On gms_v87 the wire field is a hard-coded `Encode2(0)` (`0x62f603`), so a
  Dual Blade slot is impossible there; on gms_v84 and earlier there is no sub-job field at all.
  On gms_v92 and gms_jms_185 the field is live but the offering UI was not reached within budget.
- **Does jms_185 share the v95 carousel?** **No.** Derived result: `CLogin::Update` `0x66c17f`
  branches on **four** race ordinals (0..3), where gms_v95 branches on five. jms_185 therefore
  cannot carry the v95 arrangement, in which Resistance occupies ordinal 0 and Cygnus/Aran/Evan
  are shifted to 2/3/4. Its four-ordinal shape matches gms_84/87/92; the per-ordinal class
  identities remain unverified.

## Consequences for later tasks

### Carousels required (Task 5)

Five distinct carousels, plus one unverified column:

1. **`no-race`** — the client sends no race index and no sub-job. Version keys: `gms_48`,
   `gms_61`, `gms_72`. Only one creatable identity exists (Explorer beginner).
   Evidence: `CLogin::SendNewCharPacket` at `0x500545` (v48), `0x5653e9` (v61), `0x5b219a` (v72).
2. **`race3`** — `0 = Cygnus, 1 = Explorer, 2 = Aran`; no sub-job field. Version keys: `gms_79`,
   `gms_83`. Evidence: `CLogin::Update` `0x5ca641` (v79), `0x5f4f26` (v83).
3. **`race4`** — `0 = Cygnus, 1 = Explorer, 2 = Aran, 3 = fourth race (Evan slot)`. Version keys:
   `gms_84` (no sub-job field on the wire), `gms_87` (sub-job present but hard-coded `0`),
   `gms_92` (sub-job live). Evidence: `CLogin__Update` `0x609e9f` (v84), `CLogin::Update`
   `0x62c5c8` (v87), `sub_5D5680` `0x5d5cad` (v92).
4. **`race4-jms`** — four ordinals `0..3`, class identities unverified. Version key: `jms_185`.
   Evidence: `CLogin::Update` `0x66c17f`. Do **not** reuse the `race4` class labels for it
   without checking; only the ordinal count is established.
5. **`race5`** — `0 = Resistance, 1 = Explorer (subJob 1 = Dual Blade), 2 = Cygnus, 3 = Aran,
   4 = Evan`. Version key: `gms_95`. Evidence: `CLogin::Update` `0x5dee90` / jump table
   `0x5df54f`, `CUINewCharRaceSelect::SelectRaceButton` `0x5f4d60`.
6. **`unverified`** — `gms_12`. No binary. Its single `(1,0)` slot is common to every carousel
   above that has a race index, so it is insensitive to this whole question.

### New job constants required (Task 4)

**None derivable from IDA — Task 4 degenerates to a no-op unless a non-IDA source is authorised.**

The v95 client never carries a job id on the character-creation path. `CLogin::SendNewCharPacket`
(`0x5d7bd0`) encodes name, race, sub-job, 8 avatar-look ids and gender; the job is derived
server-side from `(race, subJob)`. To make this a sweep rather than a spot check, the whole login
and login-UI address range `0x5d0000 - 0x5fa000` (51,643 instructions) was scanned for the
immediates `2001` and `3000` — **zero matches for either**. No Citizen id and no Dual Blade id
can be read out of this binary.

Do not invent a value. If Task 4 needs a Citizen/Resistance beginner id, it must come from WZ
data or an explicit decision, and that is an open question for the human, not something this
document supplies.

### Seed rows to correct (Task 7, FR-20) — positive contradictions

These are statements that the currently seeded row **is wrong**, with the evidence:

1. **`template_gms_95_1.json` is shifted by one and is missing a row.** The file seeds
   `jobIndex` 0/1/2/3 with `mapId` 130010220 / 10000 / 140090000 / 100030100 plus a `(1,1)` row.
   The v95 client's ordinals are `0 = Resistance, 1 = Explorer, 2 = Cygnus, 3 = Aran,
   4 = Evan` (`CLogin::Update` `0x5dee90`). Therefore:
   - the seeded `(0,0)` row **is wrong**: index 0 is Resistance on v95, and 130010220 is the map
     this same file gives to index 0 on every pre-v95 column, where index 0 is Cygnus;
   - the seeded `(2,0)` row **is wrong**: index 2 is Cygnus on v95, and 140090000 is the map this
     same file gives to index 2 on the pre-v95 columns, where index 2 is Aran;
   - the seeded `(3,0)` row **is wrong**: index 3 is Aran on v95, not the class that gets
     100030100;
   - a `(4,0)` row **is missing entirely** — v95 has a fifth ordinal (Evan) and the file has no
     row for it;
   - the `(1,1)` row is **correct** as a slot and should stay (`SelectRaceButton` `0x5f4d60`
     button 0).
   This document does **not** supply the replacement `mapId` values: the client carries no map
   ids, and Resistance's start map in particular has no client-side source here.
2. **`template_gms_48_1.json`, `template_gms_61_1.json`, `template_gms_72_1.json` seed race rows
   for versions whose create-character request has no race field.** Each of those files seeds
   `(0,0)`, `(1,0)` and `(2,0)`. On all three binaries `CLogin::SendNewCharPacket` encodes no
   race member (`0x500545`, `0x5653e9`, `0x5b219a`). The `(0,0)` and `(2,0)` rows **are wrong**:
   a client on these versions cannot select or transmit those ordinals. This is the FR-8
   "(0,0) is seeded on versions that predate Cygnus Knights" check, and it comes back positive.
   The `(1,0)` row is the only one the client can produce.

### Seed rows to add (Task 7, FR-19)

- `template_gms_95_1.json` needs a `(4,0)` Evan row (see above). **`mapId` unknown from this
  work** — the client holds no map ids; source it from WZ, or from the existing 100030100 value
  currently mis-filed under `(3,0)`.

No other column is missing a row: v79/v83 seed exactly the three ordinals the client branches on,
and v84/v87/v92/jms_185 seed exactly four.

### Seed rows to remove/annotate (Task 7)

- **`template_gms_87_1.json`** has no `(1,1)` row and must not gain one: v87 transmits a
  hard-coded `Encode2(0)` sub-job (`CLogin::SendNewCharPacket` `0x62f603`), so a non-zero sub-job
  is unreachable on that version. Worth an annotation so a later pass does not copy the v92/v95
  shape onto it.
- **`template_gms_92_1.json` and `template_jms_185_1.json` `(1,1)` rows: neither contradicted nor
  confirmed.** Both versions transmit a live sub-job field, so the row is *possible*; the
  offering UI was not reached within this task's budget. Leave them as they are — "the findings
  did not mention it" is not a contradiction, and this is explicitly not one.

### The v84 vs v87 `(3,0)` mapId divergence (Step 7)

`template_gms_84_1.json` gives `(3,0)` `mapId 100030102` and `template_gms_87_1.json` gives
`100030100`. **Checked against the binaries: the client provides no basis for the divergence, and
neither row is contradicted.** The two clients are structurally identical on this slot — same
four-ordinal switch (`0x609e9f` vs `0x62c5c8`), same per-ordinal dialog classes with matching
`CreateWnd` fingerprints, same `2 <-> 3` UI transposition — and neither client ever transmits or
reads a map id (`CLogin::SendNewCharPacket` `0x60cdf0` / `0x62f603` encode name, race,
avatar-look ids and gender only). The start map is a purely server-side content choice, so the
difference is a data question for the human and **not** a typo this document can adjudicate.
Task 7 must not "fix" either value on the strength of these findings.

### Other things a later task must know

- The race ordinal is a **4-byte** field on every version that has one; the sub-job, where
  present, is a **2-byte** field immediately after it.
- The member offsets differ per version and must never be shared: `0x1E0` (v79), `0x214`
  (v83/84/87), `0x224` (v92), `0x238` (jms_185), `0x240` (v95).
- Ordinal order is **not** drawing order from v84 onward: v84/87/92 transpose 2 and 3 for layout,
  and v95 uses `CLogin::ConvertSelectedRaceToUIRace` (`0x5d1c30`) for a full permutation. Any UI
  work that reads a "position" must not treat it as the wire ordinal.
- `gms_v48` uses **opcode 21** for create-character where every later version uses 22 (v95 also
  uses 23 for the char-sale variant and jms_185 uses 0x0B / 0x0C). Outside this task's scope, but
  observed while reading the packet builders.
