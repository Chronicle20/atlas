# Mob Spawn Move-Action (Stance) Byte — Client Grounding Re-Verification

Task: task-179-mob-spawn-stance-byte (Task 1 of the implementation plan)
Purpose: PRD NFR-1 grounding — re-verify the four encoding facts design.md §2
cites, against the live v83 client, before Task 2 trusts them in code.
Status: read-only, no Go code touched.

---

## 0. IDA session used

`idb_list` was run first and matched by binary **NAME** per project convention
(never trust "active" GUI focus alone). The v83 session:

```
session_id: e7e0e4fe
filename:   MapleStory_dump.exe.i64
input_path: E:\Programs\Nexon\IDBs_v9\GMS\v83_Me\MapleStory_dump.exe.i64
```

This matches the PRD/design's cited binary (`MapleStory_dump.exe`, v83). All
addresses below were decompiled from session `e7e0e4fe`.

---

## 1. The idle move-action encoding formula

### 1.1 `CMob::GetFineAction @0x671999`

Decompiled (quoted verbatim):

```c
int __thiscall CMob::GetFineAction(CMob *this, int a2)
{
  ...
  v10[0] = a2; /*0x671a2b*/
  if ( !sub_671929(this, a2) ) /*0x671a2e*/
  {
    v9 = v4;
    v5 = sub_6736A8(&dword_BEDB10, &v9, 0);
    if ( ZMap<long,long,long>::GetAt((v5 + 12), &a2, v10) )
    {
      sub_671929(this, v10[0]);
    }
    else
    {
      if ( sub_671929(this, 1) )
      {
        v10[0] = 1;
      }
      else
      {
        if ( !sub_671929(this, 3) )
        {
          ... _CxxThrowException ...
        }
        v10[0] = 3;
      }
      ...
      ZMap<long,long,long>::Insert((v7 + 12), &a2, v10);
    }
  }
  return v10[0];
}
```

`CMob::GetFineAction` is a **validate-and-cache** function: given a requested
category (`a2`, either `1`=ground or `3`=fly), it validates the mob template
actually has that animation set (`sub_671929`, a per-template animation
existence check), falling back `1 → 3` if not, and caches the resolved
category per-template in a `ZMap`. It is **not** itself the byte-formula
function; it returns the validated category (`1` or `3`, occasionally other
small ints from the cache).

### 1.2 `sub_671AFF` (the caller of `GetFineAction`, design calls it
`GetFineMoveDirAction` — no such symbol exists in the IDB; unnamed `sub_`)

Decompiled (quoted verbatim):

```c
int __thiscall sub_671AFF(CMob *this, int a2)
{
  v3 = sub_664D42(this, a2, &a2); /*0x671b0c*/
  FineAction = CMob::GetFineAction(this, v3); /*0x671b14*/
  if ( !FineAction )
  {
    v10 = 1;
    goto LABEL_12;
  }
  v5 = FineAction - 1;
  if ( !v5 ) { v10 = 2; goto LABEL_12; }
  v6 = v5 - 1;
  if ( !v6 ) { v10 = 3; goto LABEL_12; }
  v7 = v6 - 1;
  if ( !v7 ) { v10 = 6; goto LABEL_12; }
  if ( v7 == 35 )
  {
    v10 = 16;
LABEL_12:
    v8 = v10;
    return a2 & 1 | (2 * v8);
  }
  v8 = _ZtlSecureFuse<long>((this->m_pTemplate + 16), this->m_pTemplate[18]) != 3 ? 2 : 6;
  return a2 & 1 | (2 * v8); /*0x671b6c*/
}
```

Two facts here, both confirmed:

- **The `FineAction → v10` map**: `FineAction==1 → v10=2` (ground), `FineAction==3
  → v10=6` (fly). Since `sub_664D42`'s default branch (see §1.3) returns `1`
  for ground / `3` for fly, and `GetFineAction` passes that category through
  unchanged when valid, the **primary dispatch path** already produces
  `v10=2` (ground) / `v10=6` (fly) — i.e. `actionIndex = 2` ground, `6` fly —
  matching design's table.
- **The fallback branch** (taken only when `FineAction` is some other value
  the switch doesn't recognize) is the literal line design cited:
  `v8 = _ZtlSecureFuse<long>(m_pTemplate+16, m_pTemplate[18]) != 3 ? 2 : 6;
  return a2 & 1 | (2*v8);` — this is the exact `(moveAbility != 3) ? 2 : 6`
  formula from design §2, and it produces the **same** `actionIndex` values
  (`2` ground / `6` fly) as the primary path above.

**Correction to design's attribution:** design's citation reads "`sub_671AFF`
→ `sub_664D42`: `v8 = (moveAbility != 3) ? 2 : 6; return (a2 & 1) | (2*v8)`" —
implying the formula line lives inside `sub_664D42`. It does not: that exact
line is inline in `sub_671AFF`'s **fallback** branch (taken when
`GetFineAction`'s FineAction value isn't `0/1/2/3/38`), not inside
`sub_664D42`. `sub_664D42` is a separate helper (see §1.3) that resolves the
raw input into a *category* (`1`=ground/`3`=fly), which the *primary* path
of `sub_671AFF` maps to the same `actionIndex` values via the
`FineAction→v10` switch. **The numeric result (actionIndex 2 ground / 6
fly) is unaffected — corroborated by two independent branches of the same
function** — but the function-attribution in design §2 is imprecise and
should not be repeated verbatim in code comments.

### 1.3 `sub_664D42` — category resolution, and the sentinel branch

Decompiled (quoted verbatim):

```c
int __thiscall sub_664D42(_DWORD *this, int a2, int *a3)
{
  if ( a3 )
    *a3 = a2 & 1;                       /*0x664d54 — facing bit extraction*/
  switch ( a2 >> 1 )                    /*0x664d59*/
  {
    case 1: return 0;
    case 2: return 1;
    case 3: return 2;
    case 6: return 3;
  }
  if ( a2 >> 1 != 16 )
    return _ZtlSecureFuse<long>(this[98] + 64, *(this[98] + 72)) != 3 ? 1 : 3;
  return 38;
}
```

This is the **sentinel branch**, confirmed directly: when `a2` (the raw
move-action byte from the packet / `CMob::Init`) is `0` or `1`, `a2 >> 1 ==
0`, which matches **none** of the explicit `switch` cases (`1,2,3,6`) and is
not `16`, so control falls to the `_ZtlSecureFuse<long>(this[98]+64, ...) !=
3 ? 1 : 3` line — i.e. it **resolves the idle category from the mob's
move-ability field** (returning `1`=ground or `3`=fly), exactly the "please
compute the idle action" behavior design describes.

`a2 >> 1 == 0` is algebraically identical to `(a2 & ~1) == 0` for `a2 ∈
[0,255]` (both are true only for `a2 ∈ {0, 1}`), which **confirms design's
sentinel condition `(byte & ~1) == 0`** — expressed here as the `switch`
falling through to the default arm rather than as a literal masked
comparison, but functionally the same gate.

### 1.4 Truth table — CONFIRMED, unchanged from design

| isFly | fixedStance(facing) | actionIndex | byte |
|---|---|---|---|
| false (ground) | 0 (right) | 2 | **4** |
| false (ground) | 1 (left)  | 2 | **5** |
| true  (fly)    | 0 (right) | 6 | **12** |
| true  (fly)    | 1 (left)  | 6 | **13** |

`byte = (actionIndex << 1) | facingBit`, `actionIndex ∈ {2, 6}`. Confirmed via
two independent branches of `sub_671AFF` (the `FineAction`-mapped primary
path and the `moveAbility`-checked fallback), both landing on the same
`actionIndex` values. **No correction needed to design's numeric table** —
Task 2's `idleActionIndexGround = 2`, `idleActionIndexFly = 6` constants and
the `4/5` / `12/13` truth table are correct as designed.

---

## 2. Sentinel crash path

### 2.1 `CMob::OnResolveMoveAction @0x66b599`

Found via `func_query` (name regex `.*ResolveMoveAction.*`) — demangled as
`?OnResolveMoveAction@CMob@@UAEJJJJPBVCVecCtrl@@@Z` (a **virtual** method,
`U` calling-convention marker). Only one static xref exists, a **data** xref
at `0xaf8250` (a vtable slot) — consistent with it being invoked only through
indirect/virtual dispatch, not a direct `call` instruction anywhere in the
binary.

Decompiled (quoted verbatim, relevant excerpt):

```c
int __thiscall CMob::OnResolveMoveAction(MobStat *this, int nInputX, int nInputY, bool nCurMoveAction, int pvc)
{
  ...
  v9 = _ZtlSecureFuse<long>(nDazzle + 64, *(nDazzle + 72)); /*0x66b603*/
  if ( !v9 )
  {
    if ( !nInputX ) return nCurMoveAction | 4;
    return (nInputX < 0) | 4;
  }
  v10 = v9 - 1;
  if ( !v10 ) goto LABEL_20;
  v11 = v10 - 1;
  if ( !v11 )
  {
    if ( *(pvc + 272) )          /*0x66b620 — unchecked deref of `pvc`*/
    {
LABEL_20:
      if ( nInputX ) { v12 = v7 != 0 ? 16 : 1; goto LABEL_22; }
      return nCurMoveAction | 4;
    }
    ...
  }
  ...
}
```

**Confirmed:** `OnResolveMoveAction`'s `pvc` parameter (the design's `m_pvc`)
is dereferenced at `*(pvc + 272)` (`0x66b620`) with **no null check** before
the dereference. If `pvc` is `0` at that point, this is the access
violation design's summary describes.

Also confirmed by inspection: the same idle-actionIndex constants (`4`
ground, and — by the parallel `LABEL_20`/fly branches elsewhere in the
function, not fully quoted above — `6`-based `12`/`13` for fly) recur inside
`OnResolveMoveAction` itself (`return nCurMoveAction | 4;` / `... | 6`
pattern), corroborating the same `actionIndex ∈ {2,6}` encoding from a third,
independent code site.

### 2.2 `CMob::Init @0x662884` — partial confirmation, one caveat flagged

Decompiled excerpt (quoted verbatim) of the relevant span:

```c
moveAction = CInPacket::Decode1(v4); /*0x6628fe*/
this->_ZtlSecureTear_m_nMoveAction_CS = _ZtlSecureTear<long>(moveAction, this->_ZtlSecureTear_m_nMoveAction); /*0x662914*/
...
v13 = _ZtlSecureFuse<long>(this->_ZtlSecureTear_m_nMoveAction, this->_ZtlSecureTear_m_nMoveAction_CS); /*0x662a10*/
(*(*v123 + 4))(v123, 0, this->m_ptPosX, this->m_ptPosY, 0, 0, v13, a2); /*0x662a2f*/
v14 = sub_671AFF(this, *(v123 + 82)); /*0x662a3d*/
this->_ZtlSecureTear_m_nMoveAction_CS = _ZtlSecureTear<long>(v14, this->_ZtlSecureTear_m_nMoveAction); /*0x662a4f*/
```

What this confirms directly:

- `CMob::Init` decodes the raw move-action byte from the spawn/control packet
  (`CInPacket::Decode1`) into `v13`, the **same raw sentinel-or-resolved
  byte** the server emits.
- `Init` then calls `sub_671AFF(this, *(v123+82))` — i.e. the formula function
  from §1 — to compute the final `m_nMoveAction`. `sub_671AFF` internally
  calls `sub_664D42`, whose sentinel branch (§1.3) is what fires when the raw
  byte is `0`/`1`.
- Immediately before that, `Init` makes an **indirect virtual call**
  `(*(*v123 + 4))(v123, 0, m_ptPosX, m_ptPosY, 0, 0, v13, a2)` where `a2` at
  this point in `Init` has been reused to hold a **`CStaticFoothold*`**
  pointer (`a2 = CWvsPhysicalSpace2D::GetFoothold(...)`, possibly `0` if the
  mob spawns without a resolvable foothold under it — exactly the "bulk
  re-spawn" scenario design's summary names). The argument shapes (`nInputX,
  nInputY, nCurMoveAction, pvc`-like tail) are consistent with
  `OnResolveMoveAction`'s signature, and a null foothold pointer landing in
  `OnResolveMoveAction`'s `pvc` parameter would produce exactly the
  `*(pvc+272)` null-deref confirmed in §2.1.

**What is NOT fully traced (caveat, per CLAUDE.md — stating this rather than
asserting it as confirmed):** the vtable slot `(*v123 + 4)` was resolved to a
concrete function (`CVecCtrlMob::UpdateActiveInterrupted @0x9BC13D`, a
physics-tick update routine) whose decompiled parameter list does not
line up 1:1 with the 7 explicit arguments at the `Init` call site — almost
certainly a decompiler ABI-inference artifact for a register-heavy
`__userpurge` virtual call, not proof the wrong function was found, but not
conclusive either. I was not able to fully confirm, at the disassembly
level, that this specific indirect call is the one that invokes
`CMob::OnResolveMoveAction` as its callback with the foothold pointer as
`pvc`. **This link (Init's indirect call → OnResolveMoveAction) is therefore
"cited, not fully re-traced"** — everything else in this document (the byte
formula, the truth table, the sentinel arithmetic `(byte & ~1)==0`, and
`OnResolveMoveAction`'s unchecked `pvc` deref) is independently confirmed by
direct decompile.

This caveat does not affect Task 2: Task 2's helper only needs the byte
**encoding formula** (§1, fully confirmed) to be correct; the crash
mechanism (§2) is motivating context for *why* the server must never emit
`0`/`1`, not a formula Task 2's code needs to reproduce.

---

## 3. Constants for Task 2 — confirmed

Per design.md §3.1 (`libs/atlas-constants/monster/stance.go`):

```go
const (
    idleActionIndexGround = 2   // confirmed §1.4
    idleActionIndexFly    = 6   // confirmed §1.4

    FacingRight byte = 0
    FacingLeft  byte = 1
)
```

- `IdleMoveAction(isFly, fixedStance)` truth table: ground `4`/`5`, fly
  `12`/`13` — **confirmed, no correction**.
- Sentinel definition `(byte & ~1) == 0` (i.e. byte `0` or `1`) —
  **confirmed** (§1.3, §2.1), algebraically equivalent to the `a2>>1==0`
  fallthrough observed in `sub_664D42`.

No constant in design.md §3.1 requires correction. Task 2 may proceed with
the helper as designed.

---

## 4. Source addresses (v83, `MapleStory_dump.exe`, IDA session `e7e0e4fe`)

| Symbol | Address |
|---|---|
| `CMob::GetFineAction` | `0x671999` |
| `sub_671AFF` (formula caller; unnamed in IDB) | `0x671AFF` |
| `sub_664D42` (category resolver / sentinel switch; unnamed in IDB) | `0x664D42` |
| `CMob::OnResolveMoveAction` | `0x66b599` |
| `CMob::Init` | `0x662884` |
| `CVecCtrlMob::UpdateActiveInterrupted` (vtable+4 candidate, unconfirmed link) | `0x9BC13D` |
