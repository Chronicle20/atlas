# Addendum — Hex of the Beholder: authentic per-pulse buff accumulation

Status: Proposed (follow-up to task-088)
Created: 2026-06-14
Author context: raised during live v83 play-testing of task-088 on `atlas-pr-746`.
Relates to: `design.md` Q3, §7 (Beholder snapshot), `processor.go`, `beholder_task.go`,
and the shared `atlas-buffs` service.

---

## 1. Motivation

In original (pre-Big-Bang) GMS, a Dark Knight's **Hex of the Beholder** (`1320009`)
visibly **accumulates** buff icons over time — they pop in one at a time, each with
its own countdown — rather than all appearing/refreshing together. Our task-088
implementation (and Cosmic, which it mirrors) instead applies the **entire** stat
set every pulse, so all icons reset in lockstep and read as a "recast." This
addendum specs the work to (optionally) reproduce the authentic accumulation, and
documents the structural blocker that makes it non-trivial.

**Ownership.** This is a **task-088 (summons) gap** — Beholder does not behave as a
player expects. The fix happens to live in `atlas-buffs` because that is the
mechanism the Beholder drives, but the missing capability is owned by, and tracked
under, the summons feature.

This is a **fidelity** improvement, not a crash/correctness fix. The Beholder heal,
hex stat values, cast animation, and crash fixes from the main task are all already
correct and live.

## 2. Verified findings (evidence)

1. **Original GMS applies one buff per pulse.** "The beholder grants **one** of those
   buffs every 10 seconds, with each buff lasting 80 seconds." (GameFAQs / Hidden-Street
   Before-Big-Bang / MapleLegends old-school forums — see §7.) Each buff carries its own
   independent duration, so icons accumulate one-at-a-time until the pool is full.

2. **The WZ stat pool grows with skill level** (`~/source/Cosmic/wz/Skill.wz/132.img.xml`,
   skill `1320009`):

   | level | interval `x` | duration `time` | stats in the single effect node |
   |---|---|---|---|
   | 1 | 48s | 20s | `pdd` |
   | 10 | 30s | 40s | `pdd, mdd` |
   | 20 | 10s | 80s | `pdd, mdd, acc, eva` |
   | 25 | 4s | 99s | `pdd, mdd, acc, eva, pad` |

3. **Cosmic (our reference) applies the whole set at once.** `Character.java:4474-4488`
   schedules `buffEffect.applyTo(this)` every interval — one combined buff carrying all
   WZ statups, all refreshed to the same duration. Cosmic does **not** reproduce the
   one-at-a-time accumulation; it only randomizes the *animation* stance
   (`summonSkill(..., (int)(Math.random()*3)+6)`).

4. **Atlas mirrors Cosmic.** `services/atlas-summons/.../summon/processor.go` snapshots
   the full `hex.Statups()` into `BuffChanges` at spawn; `beholder_task.go:sweepBuff`
   emits a single `APPLY` carrying all of `BuffChanges` each pulse. Result: identical
   lockstep behavior.

## 3. Current behavior (code references)

- Snapshot (spawn): `services/atlas-summons/atlas.com/summons/summon/processor.go:164-178`
  — `SetBuffChanges(changes)` with **all** statups; `SetBuffSourceId(int32(1320009))`
  (positive — see the task-088 crash fix; negative crashes the client icon lookup).
- Pulse (sweep): `services/atlas-summons/atlas.com/summons/summon/beholder_task.go:sweepBuff`
  — one `buffmsg.ApplyProvider(..., m.BuffSourceId(), ..., changes)` carrying the whole set.
- Downstream: `atlas-buffs` `Apply` → `APPLIED` event → `atlas-channel`
  `CharacterBuffGiveBody` → client give-buff (each stat written with `sourceId` as its
  per-stat `rSkillID`).

## 4. Desired behavior

Each Beholder buff pulse applies **one** statup chosen at random from the level's pool
(`pdd` / `mdd` / `acc` / `eva` / `pad` as unlocked by level), as its **own** buff with
its **own** duration. Over successive pulses the icons accumulate one-at-a-time and
refresh on independent timers — matching original GMS.

## 5. Structural blocker (must be resolved first)

**`atlas-buffs` keys active buffs by `sourceId` only.**
`services/atlas-buffs/atlas.com/buffs/character/registry.go:67` stores
`m.buffs[sourceId] = b` (a `map[int32]buff.Model`); `Cancel`/`expire` also key by
`sourceId`. Therefore:

- Emitting five separate `APPLY`s all sourced from `1320009` **collide** — each new
  pulse overwrites the prior buff under key `1320009`. You cannot hold five
  independently-timed hex stats under one source. This is the exact opposite of what
  accumulation needs.
- We **cannot** simply assign a distinct `sourceId` per stat: the client uses `sourceId`
  as the give-buff `rSkillID` and calls `GetSkillTemplate(sourceId)` for the icon — an
  invalid/synthetic id crashes the v83 client (this is precisely the bug the task-088
  positive-sourceId fix resolved). The only safe `sourceId` is the real skill `1320009`.

In real MapleStory the client keys temporary stats by **stat type**, not by skill, so
five stats can share `rSkillID = 1320009` yet keep independent durations. `atlas-buffs`
does not model this.

### 5.1 Does fixing this break normal "recast overwrites" buffs?

No — and this is the key insight that shapes the recommended design.

**Accumulate-vs-overwrite is decided by the Apply *payload*, not by the registry key.**

- A normal multi-stat buff (Rage, a multi-stat potion, Maple Warrior, …) re-sends
  **all** of its stats on every cast: `Apply(sourceId=R, [A, B, C], duration)`. Whether
  the registry keys by `sourceId` or by `(sourceId, statType)`, those stats land on the
  exact slots the previous cast used, so a recast **overwrites/refreshes** — identical
  to today.
- Hex differs only because each *pulse* carries **one** stat: pulse→`(1320009, pdd)`,
  next pulse→`(1320009, mdd)` (a different slot, so it **adds**); re-rolling an
  already-active stat hits the same slot and refreshes just that one timer.

So finer keying alone would *not* convert existing buffs into accumulators. It would,
however, introduce one genuine behavior change: under naive `(sourceId, statType)`
keying a recast that carries a **smaller** stat set than before would leave the dropped
stat lingering until its own expiry, whereas today's `sourceId`-only model replaces the
whole buff and drops it cleanly. Real skills have stable stat sets so this is largely
theoretical — but it is a behavior change to a shared service, and it is the reason we
prefer an **opt-in mode** over a blanket re-key.

### 5.2 Resolution options

- **Option A — explicit `accumulate` mode on `Apply` (recommended; see §6).** Default
  `Apply` keeps **today's exact** replace-by-`sourceId` semantics (every existing buff
  byte-for-byte untouched, zero blast radius). A new opt-in `accumulate` flag makes
  `atlas-buffs` track that source's stats with **independent per-stat timers** and emit
  incremental give-buff/expire deltas. Only the Beholder hex sets the flag.
- **Option B — globally re-key by `(sourceId, statType)`.** Cleaner long-term model but
  changes the internal contract for *every* buff (Apply/Cancel/expiry/
  CancelByStatTypes/REST/projection all assume one entry per source) and carries the
  §5.1 reduced-stat-recast edge case. Larger regression surface for no extra player-
  visible benefit over A.
- **Option C — accept the divergence (do nothing).** Keep the lockstep full-set
  behavior. At the levels that matter (≥20, interval ≤10s ≪ duration ≥80s) all stats are
  effectively always-on regardless, so the only observable difference is the ~20–40s
  cosmetic "fill-in" right after summoning. Functional buff outcome is identical.

## 6. Recommendation

**Pursue Option A (explicit `accumulate` mode).** It is the smallest change that makes
Beholder behave authentically while guaranteeing existing buffs are untouched —
normal multi-stat buffs keep overwriting on recast precisely *because* they never set
the accumulate flag (§5.1). Treat it as two coordinated pieces under the summons
feature umbrella:

1. **`atlas-buffs`**: add the opt-in `accumulate` capability (per-stat timers +
   incremental APPLIED/EXPIRED deltas) without altering default behavior.
2. **`atlas-summons`** (`beholder_task.go:sweepBuff`): emit one randomly-selected
   statup from the snapshot pool per pulse with the `accumulate` flag set, instead of
   the full set.

If the appetite for the `atlas-buffs` work is low, **Option C** remains an acceptable
fallback (the gap is cosmetic at the relevant levels) — but the gap stays open and
owned by summons, not closed.

## 7. Scope, non-goals, acceptance

**In scope (if pursued — Option A):**
- `atlas-buffs`: add an opt-in `accumulate` mode to `Apply` (per-stat independent
  timers under one `sourceId`; incremental APPLIED/EXPIRED deltas). Default `Apply`
  semantics unchanged. Covered by its own tests, including a regression that a
  default-mode multi-stat recast still overwrites (does not accumulate).
- `atlas-summons` `beholder_task.go:sweepBuff`: emit one statup chosen at random from
  the snapshot pool per pulse with `accumulate` set (randomness varied per pulse on the
  leader ticker, not a fixed seed). The full pool is already snapshotted in
  `BuffChanges`.
- Per-stat independent duration carried end-to-end (summons → buffs → channel
  give-buff). `sourceId` stays the real `1320009` throughout.

**Non-goals:**
- Changing the heal (`AURA_OF_BEHOLDER`) — single-stat already.
- Changing hex stat *values* or interval/duration (those are correct per WZ).
- Any client/packet change — the give-buff wire is unchanged (still `sourceId = 1320009`).

**Acceptance criteria (if pursued):**
- On a fresh Beholder with Hex ≥ L20, buff icons appear **one at a time** over
  successive pulses (not all at once), each with an independent countdown.
- No client crash (sourceId stays the real `1320009`).
- `atlas-buffs` existing buff behavior (potions, other skills) unchanged — full
  regression of `Apply`/`Cancel`/expiry.
- Verified on live v83 (`atlas-pr-XXX`) by observation + Loki `APPLIED`/`EXPIRED` trace
  showing per-stat independent timers.

## 8. Sources

- GameFAQs — Dark Knight, Hex of the Beholder:
  https://gamefaqs.gamespot.com/boards/924697-maplestory/57979560
- MapleLegends (Old School) — "A reasonable Hex of the Beholder buff":
  https://forum.maplelegends.com/index.php?threads/a-reasonable-hex-of-the-beholder-buff.23359/
- Hidden-Street (Before Big Bang) — Hex of the Beholder:
  https://bbb.hidden-street.net/character/skill/hex-of-the-beholder
- Local: `~/source/Cosmic/src/main/java/client/Character.java:4447-4490`;
  `~/source/Cosmic/wz/Skill.wz/132.img.xml` (skill `1320009`).
