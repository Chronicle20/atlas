# client analysis: how v83 suppresses mob touch damage under Dark Sight

Read-only decompile of the GMS v83 client (`MapleStory_dump.exe.i64`, IDB
session `41f09cce`). No IDB mutations were made. Produced 2026-08-18 as part
of the second investigation pass on `bug-dark-sight-monster-damage.md`, after
live testing contradicted the first pass's conclusion.

## Why this pass exists

The first pass concluded "the v83 client's mob-touch/collision damage path
does not gate on Dark Sight at all — this is baseline client behavior, so the
fix must be server-side suppression." Live testing refutes that: monsters
spawned by the map's own life data, and monsters spawned by the GM `!spawn`
command, do NOT touch-damage a dark-sighted player, and no damage report even
reaches the server. Only monsters spawned by the CRIMSON_BALROG event do. The
client therefore does suppress touch damage under Dark Sight; the first pass
looked for the gate in the wrong function.

## The producer chain (confirmed end-to-end)

```
CMob::Update
  [whole arming block gated by *(m_pvcActive-12 + 0x18)]
  -> CMob::TryDoingSkill(...)
     || (*(m_pvcActive-12 + 0x250) && CMob::IsTargetInAttackRange(this, 3, ...))   0x66a517
  -> CMob::GenerateMovePath(this, attackIdx+12, ...)                               0x66b6fc
  -> CMob::DoAttack(this, attackIdx+12, ...)                                       0x66d9c0
     - requires CMobTemplate::GetAttackInfo(idx) != NULL   (0x66da34)
     - type-0 (plain contact) branch pushes a node onto CMob::m_lpLayerASAni
       (offset 0xE4): node[1] = AttackInfo type (0), node[7] = attack index
  -> sub_666362 (runs every Update, unconditionally)                               0x6685c8
     - drains that queue, re-tests each node's rect against the local player's
       CURRENT body rect every tick until the node's due-time
     - on overlap dispatches vtable slot 18 = CUserLocal::SetDamaged (0x9581a9)
       under the guard `if (!AttackInfo[+0x24] || localPlayerVecCtrl[+0x110])`
       (disasm 0x66724d / 0x667252)
```

`CMob::DoAttack` is the ONLY producer of `m_lpLayerASAni` nodes.

## Where Dark Sight actually suppresses touch damage

Not in `sub_666362` — the first pass was right that this function never calls
`CUser::IsDarkSight` or `CUser::IsSneak`. The suppression is upstream, in
`CMob::IsTargetInAttackRange`'s per-attack validity loop (`0x66a517`, condition
at `0x66a80x`-`0x66a854`):

```c
if ( (AttackInfo[36] || !v5->m_bDoFirstAttack) && !AttackInfo[4]
  && AttackInfo[5] <= <mob MP>
  && (a2 || !CAvatar::IsHideMorphed(...))
  && (!v19[40] || v5->m_bAttackReady)
  && (a2 || v19[7] || <special mobID> || !CUser::IsDarkSight(v93) && !CUser::IsSneak(v93)) )
    break;   // this attack index is usable
```

With the resolved target being the local player, `a2` resolves to 0, so the
last clause requires `AttackInfo[7]` (a per-attack "ignore stealth" override)
or `!IsDarkSight`. While the player is dark-sighted and the attack carries no
override, no attack index validates, `IsTargetInAttackRange` returns 0
(`0x66ae5e`), `GenerateMovePath`'s attack branch is never taken, `DoAttack`
never runs, and no node is ever queued. `sub_666362` still runs every tick,
but has nothing to dispatch.

**So the suppression is "the mob never arms an attack against an invisible
player," not a check at the moment of contact.** `CUser::IsDarkSight`
(`0x4f0d45`) has exactly one mob-related call site in the whole binary: this
one. Every other xref is player-side skill-use / UI code.

Corollary, resolving a loose end from the first pass: `GetAttackInfo` never
returns NULL for a queued node, so a client-internal `nAttackIdx == -1` never
reaches `sub_666362`. The `nAttackIdx == -1` seen on the wire is
`CUserLocal::SetDamaged` reporting a type-0 (plain contact) attack as -1, not
an absence of attack info.

## What this does NOT explain

The hypothesis under test was: a mob spawned with foothold 0 is ungrounded,
skips the Dark-Sight-gated step, and still reaches the touch dispatch.
**That specific mechanism is refuted by the control flow.** The entire arming
block in `CMob::Update` — `TryDoingSkill(...) || (*(v171+592) &&
IsTargetInAttackRange(this,3,...))` — sits inside

```c
if ( (... m_pTemplate[16]==3 ...) && *(v171 + 24) )   // v171 = m_pvcActive-12
```

`*(m_pvcActive-12 + 0x18)` is the same flag `CMob::TryFirstAttack` requires
just to attempt chasing (`if (!*(...+0x18) || *(...+0x250)) return;`,
`0x66e356`). If that flag is the mob's "landed" flag, an ungrounded mob does
not bypass the Dark Sight check while still reaching `DoAttack` — it never
reaches the block containing EITHER. If anything that predicts fewer touch
hits while ungrounded, not more.

`CMobPool::OnMobEnterField` -> `CMob::Init` (`0x662981`-`0x66299a`) does feed
an unresolved spawn foothold through as `a3 = 0` into `CVecCtrlMob::Init`, so
`fh = 0` on the wire genuinely does produce a differently-initialized mob
physics state. What that state then does is the open question.

## Open — what could not be determined

- The identity and setter of the `-12`-adjusted `+0x18` "landed" flag, and
  therefore whether an `fh = 0` mob ever regains it and how fast. Candidates
  not yet checked: `CVecCtrl::CalcFloat` (`0x9b2c3c`),
  `CVecCtrl::WorkUpdateActive` (`0x9b19d0`), `CVecCtrlMob::WorkUpdateActive`
  (`0x9bca2a`), `CVecCtrlMob::CtrlUpdateActiveMove` (`0x9bccaf`).
  (Note: the first pass mis-identified `CVecCtrl::FallDown` (`0x9b1c51`) as
  the setter; its `this+24`/`this+30` writes are at a different base-pointer
  adjustment and are a different field.)
- The semantics of `AttackInfo[+0x24]` and `localPlayerVecCtrl[+0x110]` — the
  guard actually inside `sub_666362`. Neither carries Dark Sight or grounding
  semantics as far as could be determined; both only matter for a node that
  already exists.
- Whether `AttackInfo[7]` — the per-attack "ignore stealth" override in the
  gate above — is set on Crimson Balrog (8150000)'s contact attack. If it is,
  that template ignores Dark Sight unconditionally and the spawn path is
  irrelevant.
- Whether any route exists to a queued `m_lpLayerASAni` node that does NOT
  pass through the local client's own `IsTargetInAttackRange` call — e.g. a
  server/control-packet-driven `m_nOneTimeAction` change putting the mob
  straight into an attack animation. NOT ruled out.
