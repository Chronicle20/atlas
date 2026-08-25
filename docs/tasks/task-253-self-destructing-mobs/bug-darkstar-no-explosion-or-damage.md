# bug: High Darkstar (8500003) detonates on spawn with no explosion and no damage

Task: task-253-self-destructing-mobs
Branch: `task-253-self-destructing-mobs` @ `bbe808f7c`
Environment: namespace `atlas-pr-1462`, tenant `b3a10958-f792-4e02-bb62-89cd6cf11e9e`,
region **GMS**, version **83.1**.

Companion to [`bug-firebomb-no-selfdestruct-animation.md`](bug-firebomb-no-selfdestruct-animation.md);
findings B and C below share that file's open decision.

## Reproduced

Yes — from live logs of the reporter's own session. `atlas-channel` runs at debug
level in this namespace, so the whole sequence is on record. Two spawns, `1000145`
at 19:33:17 and `1000146` at 19:34:23 (`@mob spawn 8500003`).

For `1000146`:

| t (19:34:23) | evidence |
|---|---|
| `.010` | `Spawned 1x monster 8500003 (High Darkstar) at (-254, 95).` |
| `.022` | `MONSTER_STATUS` `CREATED`, uniqueId 1000146 |
| `.025` | `GET /api/monsters/1000146` → `hp: 10000, maxHp: 10000, damageEntries: []` |
| `.049` | `[MonsterBomb] read [mobId [1000146]]` |
| `.052` | `Requesting self-destruct of monster [1000146] reported by character [1].` |
| `.077 .119 .136 .166` | four more `MonsterBomb` reports for the same mob |
| `.167` | `MONSTER_STATUS` `KILLED` — `damageEntries: []`, `boss: true`, `deathType: "DESTRUCT_BY_MISS"` |
| `.169 .199` | `MONSTER_BOMB: monster [1000146] is not in the live mirror; dropping report` |

So the mob died **145 ms after CREATED, at full HP, with zero damage entries**, via
the contact trigger.

## Observed

Mob spawns, is visible for an instant, disappears. No explosion animation, no damage
to the character.

## Root cause — three separate things, only one of which is arguably a defect

### A. The instant detonation is the contact trigger working as designed

`@mob spawn` places the mob on top of the spawning character, so the client's contact
check passes on the very first frame. The client function bound to opcode `0xC1` is
`CMob::TryFirstSelfDestruction` (v95 `ecc757f4` @ `0x640ee0` — the v83 IDB has no
symbol for it, but the v83 client demonstrably sends the same opcode). It fires when:

```c
if ( m_pTemplate->bSelfDestruction && m_pTemplate->bFirstSelfDestruction ) {
    CAvatar::GetBodyRect(&localUser, &rcLocalUser, 0);
    for each attack i with AttackInfo->nType == 0:
        rcRange = AttackInfo->rcRange offset by mob pos;
        if ( IntersectRect(&rcIntersect, &rcRange, &rcLocalUser) ) {
            COutPacket(232 /* v95 opcode */); Encode4(GetMobID(this)); SendPacket();
        }
}
```

i.e. the mob's type-0 attack rect intersecting the local user's body rect. Spawning it
under your own feet satisfies that immediately, and the client re-sends every ~30 ms
until the mob is gone. `Registry.SelfDestruct` makes the detonation exactly-once and
the live-mirror guard drops the stragglers, both visible in the log above. **Not a
defect. To test 8500003 meaningfully, spawn it away from the character.**

### B. No explosion animation — WZ `action: 3` cannot render as one on v83

`atlas-data` reports `8500003` ("High Darkstar"): `hp: 10000`,
`self_destruction: {action: 3, remove_after: -1, hp: 5000}`, `boss: true`,
animations `[attack1, die1, hit1, stand]`.

`deathTypeForAction(3)` → `DESTRUCT_BY_MISS` → wire byte **3**. The v83 client
(`754107bf`, `CMobPool::Update` @ `0x679138`) branches:

```c
if ( deadType <= 1 || deadType == 3 )  CMob::OnDie(mob);   // 0x663995
else                                   CMob::OnBomb(mob);  // 0x663e5b
```

`CMob::OnBomb` is the explosion: mob sound, `m_nOneTimeAction = 12`,
`PrepareActionLayer`, then `CMob::DoAttack(this, 12, IsLeft, 0)`. It is reachable on
v83 **only from dead-types 2, 4, 5**. Byte 3 is distinct only on v92+/v95, where it
maps to `CMob::OnDestructByMiss` (`0x64ea30`).

**And `CMob::OnDie` does not play a die animation for dead-type 3 either.** This is the
answer to "but die1 *is* the explosion, why don't I see it?" — `CMob::OnDie` picks the
one-time action like this (v83 `0x663995`, with the v95 `0x64e4b0` symbolised twin
below it):

```c
// v83 @0x663a1b                        // v95 @0x64e6bc (symbolised)
v6 = m_pTemplate[137];                  nDieCount = m_pTemplate->nDieCount;
if ( v6 > 0 ) {                         if ( nDieCount > 0 ) {
  if ( m_nDeadType == 3 )  v8 = 21;       if ( m_nDeadType == 3 )  v13 = 22;
  else  v8 = rand() % v6 + 9;             else  v13 = rand() % nDieCount + 10;
  m_nOneTimeAction = v8;                  m_nOneTimeAction = v13;
  CMob::PrepareActionLayer(this);         CMob::PrepareActionLayer(this);
```

The v95 symbols confirm `m_pTemplate[137]` is `nDieCount`; the two versions differ only
by a one-slot shift in the action enum (base 9 vs 10, special 21 vs 22).

So dead-type 3 **explicitly diverts away from `die1..dieN`** to a dedicated one-time
action. `8500003`'s animations are `[attack1, die1, hit1, stand]` — it has no art for
that action, so `PrepareActionLayer` renders nothing and the mob silently vanishes.
The `die1` explosion sprite is never reached.

This also corrects design §2.2, which describes v87 as collapsing `{0,1,3}` into "an
ordinary death". The `CMobPool::Update` dispatch does route 3 to `OnDie`, but `OnDie`
itself branches on dead-type 3 — so 3 is *not* an ordinary death on v83/v87, it is a
distinct action with no fallback to `die1`.

Under the "pass the WZ action byte through verbatim" rule, a v83 client therefore shows
these bombs neither exploding nor dying.

### C. Nothing deals detonation damage — and the PRD never asked for it

`selfDestructFrom` (`monster/processor.go:1902-1927`) does registry transition →
credit → `finalizeKill`. There is no emission of damage to characters anywhere on the
self-destruct path, and no `atlas-channel` path that would produce it.

The PRD lists this neither as a goal nor as a non-goal (§2 Non-goals covers
`MOB_TIME_BOMB_END`, Papulatus scripting, Monster Carnival, `info/removeAfter`, and
ordinary deaths — not explosion damage). So "it should blow up and deal damage" is an
unimplemented requirement that was never written down, not a regression.

On v83 the natural mechanism for that damage is the client's own
`CMob::OnBomb` → `DoAttack(12)`, which is unreachable at dead-type 3 — so B and C
likely have a single fix, not two.

## Reference-server evidence (reporter, 2026-08-25)

Cosmic **hardcodes dead-type `4`** for all `MONSTER_BOMB` handling — it does *not*
pass `selfDestruction.action` through on the contact path — and the reporter confirms
the resulting Darkstar behaviour matches retail gameplay.

On v83 that lands in the `else` arm of `CMobPool::Update`, i.e. `CMob::OnBomb`. So the
explosion the reporter expects comes from **the contact path using a bomb dead-type
instead of the WZ action byte**, and the HP-threshold path keeping pass-through
(Cosmic does pass `action` through there). That also explains why Firebomb's threshold
detonation looking like an ordinary death is correct: it is Cosmic parity.

## Fix — DECIDED 2026-08-25 (reporter approved)

Scope the deviation to **`TriggerContact` only**: in
`ProcessorImpl.SelfDestruct` (`monster/processor.go:1847`), use `DeathTypeBomb`
instead of `deathTypeForAction(sd.Action())` when `trigger == TriggerContact`. Leave
`TriggerThreshold` and `TriggerTimer` on WZ pass-through.

Use `BOMB` (byte **2**), not Cosmic's literal `4`:

| version | dead-type 2 | dead-type 4 |
|---|---|---|
| GMS v83 (`754107bf` `0x679138`) | `CMob::OnBomb` | `CMob::OnBomb` |
| GMS v87 (design §2.2) | bomb arm | bomb arm |
| GMS v95 (design §2.2) | `CMob::OnBomb` `0x650ec0` | `CMob::OnSwallowed` `0x641810` |
| JMS v185 (`a977912e` `0x6f850a`) | `CMob::OnBomb` | `CMob::OnSwallowed` |
| GMS v92 | **unverified** — see below | `CMob::OnSwallowed` (trailing int32) |

Byte 2 reproduces Cosmic's v83 gameplay exactly while staying correct on v92+/JMS,
where Cosmic's 4 would hit the swallow arm and drag in the trailing
`swallowCharacterId` int32 that `hasSwallowCharacterId` already gates. It also stays
inside the operations table, so DOM-25 is preserved and no byte is hardcoded.

Files that would change:

- `services/atlas-monsters/atlas.com/monsters/monster/processor.go`
  (`deathTypeForAction`, `SelfDestruct`, `selfDestructFrom`, `damageCore` threshold arm)
- `services/atlas-monsters/atlas.com/monsters/monster/self_destruct_timer_task.go`
- tests: `monster/self_destruct_test.go`, `monster/self_destruct_timer_test.go`
- docs: `design.md` §2.2, `prd.md` §3/§4/§6.3

If explosion damage must be server-authoritative rather than client-driven, that is a
larger change (a new damage emission from `atlas-monsters` into the character damage
path) and should be its own task.

## Not yet answered

- **v92's dead-type switch is partly unverified — but the wire hazard is closed.**
  `CMobPool::Update` is unsymbolised in session `019cd393` (only the `OnMobPacket`
  family carries symbols), so which handler byte 2 dispatches to on v92 was not
  confirmed. What *is* confirmed, straight from v92 `CMobPool::OnMobLeaveField`
  `0x64bb90`: `if ( v4 == 4 ) v5 = CInPacket::Decode4(...)` — only dead-type **4**
  reads the trailing int32, so byte 2 is wire-safe on v92. The residual unknown is
  cosmetic (which animation plays), not a desync, and both neighbours (v87, v95) plus
  JMS v185 route 2 to the bomb arm.
- Whether the v83 client renders anything for `CMob::OnBomb` on a mob whose
  animation set has no action-12 entry. `8500003`'s extracted animations are
  `[attack1, die1, hit1, stand]`. **Untested — a BOMB dead-type could produce a
  silent removal, which is no better than today.** This must be tried live before any
  remap is accepted.
- Whether `CMob::DoAttack(this, 12, ...)` actually produces player damage on v83, or
  only an animation. Not traced.
- **Separate lead, out of this task's scope:** the client gates
  `TryFirstSelfDestruction` on `m_pTemplate->bFirstSelfDestruction`, yet `atlas-data`
  reports `first_attack: false` for `8500003`. The client evidently disagrees with our
  extraction. `getFirstAttack` (`services/atlas-data/atlas.com/data/monster/reader.go:196-204`)
  may be reading the wrong node. Worth a look under the auto-aggro/first-attack work
  (#1460), not here.

## Resolution

Fixed in **`60ac906d9`** — "fix(monsters): contact self-destructs detonate as bomb, not
WZ action byte". `TriggerContact` now emits `DeathTypeBomb`; threshold and timer keep WZ
pass-through. Covered by `TestSelfDestructContactAlwaysBomb`.

Review: **APPROVED**, zero findings —
[`reviews/review-bug-contact-bomb.md`](reviews/review-bug-contact-bomb.md). The reviewer
independently confirmed the atlas-channel seam (`destroyCodeFor` → `DestroyMonsterBomb`)
and that all 11 seed templates already carry `"BOMB": 2`.

Gate: `tools/verify.sh --quick --base bbe808f7c` **FAILED at the lint guard only**, on
pre-existing toolchain drift unrelated to this change — the pinned
`golangci-lint-v2.12.2` is built with go1.26.1 and cannot type-check the go1.27 stdlib
now on PATH (`panic: file requires newer Go version go1.27`). Confirmed environmental by
running the same binary against `atlas-buffs`, a module this branch never touches: same
panic. `go build`, `go vet`, and all tests passed. No `go.mod` in the repo declares
go 1.27. **A gate fix is coming on a separate branch (reporter, 2026-08-25); the flagless
gate must be re-run on this branch afterwards before task-253 is called done.**

**Live re-test still outstanding.** Spawn `8500003` *away from* the character —
`@mob spawn` places it underfoot and the contact trigger detonates it in ~27 ms.
