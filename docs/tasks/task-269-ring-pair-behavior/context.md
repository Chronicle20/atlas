# Ring Pair Field Behavior — Planning Context

Task: task-269-ring-pair-behavior
Artifacts: [`prd.md`](prd.md) → [`design.md`](design.md) → [`plan.md`](plan.md)
Written: 2026-08-26

---

## 1. Branch state — read this first

**This branch is rebased onto `origin/task-240-cash-shop-stub-operations`
(head `dd7e0bbb4`), not onto `main`.**

`design.md` §8 and `prd.md` §11 both state that planning must not begin until
PR #1426 lands on `main`. At planning time `gh pr view 1426` returned
`"state":"OPEN"`, and `git ls-tree origin/main -- services/atlas-cashshop/atlas.com/cashshop/ring`
returned nothing. The user chose to stack this branch on task-240 rather than
wait, so the `ring` package, `cash_rings`, `GET /rings` and `RING_PURCHASED`
resolve here.

Consequences to carry into execution:

- `origin/task-240-...` is itself **behind `main`**. Do not rebase onto `main`
  until #1426 merges; do it at PR time, per CLAUDE.md's "produce the clean PR
  branch by rebase at PR time."
- If #1426 changes in review, Task 8's assumptions about `ring.Model`,
  `ring.RestModel` and `cashshop/processor_ring.go` move with it. Task 8 is the
  only task exposed to that risk; Tasks 1-7 touch `libs/atlas-packet` only.

Rebase performed: `git rebase --onto origin/task-240-cash-shop-stub-operations 6c956fe8c task-269-ring-pair-behavior`, clean, 2 commits replayed.

## 2. Scope decision

The user confirmed `design.md` §1's scope correction: **all four encoder sites
plus the `appearance_update.go` frame fix**, not the PRD's literal two.

`design.md` §1 justifies it from `xrefs_to` on the v83 registrars, which returns
exactly `CUserLocal::SetPairCharacterID` (site A), `CUserRemote::Init` (site B)
and `CUserRemote::OnAvatarModified` (site C). PRD §10's own acceptance criteria
("see who my ring is paired with when I inspect my own character", "the partner
name on the ring item in my equip inventory") are unsatisfiable without site A.

## 3. Three corrections this plan makes to `design.md`

The design was derived before this survey. Each of these was verified against
the repo or the checked-in exports during planning, and the plan encodes the
corrected version. They are recorded here so a reader who trusts `design.md`
over `plan.md` is not misled.

### 3.1 The trailing 4-byte ring field is unresolved (Task 1)

`design.md` §2 and §3.1 name it `ItemId`. The checked-in exports disagree:

| Source | Names it |
|---|---|
| `docs/packets/ida-exports/gms_v83.json` → `CUserRemote::OnAvatarModified` | `dwPairCharacterId (4 bytes)` |
| `gms_v87.json`, `gms_v95.json` → same function | `dwPairCharacterId (4 bytes)` |
| `gms_jms_185.json` → `CUserRemote::OnAvatarModified` | `friendship pair characterId (per entry)` |
| `gms_jms_185.json` → `CUserRemote::Init` @0xa52876 | `couple-ring itemId (per entry)` |

The JMS export contradicts *itself* across two functions. One of these export
comments is wrong. This decides whether the channel sends a template id or a
character id — it cannot be papered over, so **Task 1 is a blocking derivation
task that produces no code**, and every later task consumes its verdict.

The same class affects the marriage block: exports name the triple
`dwMarriageCharacterID / dwMarriagePairCharacterID / nWeddingRingID`, while
`design.md` §3.1 names the first field `MarriageId`.

Per CLAUDE.md ("an unresolved packet-audit fname … surface it and ask"), this
was surfaced to the user before the plan was written.

### 3.2 `appearance_update.go` is one line wrong, not three

`design.md` §3.2 says the encoder is "two bytes short and one int long."

`gms_v83.json` `CUserRemote::OnAvatarModified` @0x98367e shows the second read
is a **flags byte**: `bit0=avatarLook, bit1=speed, bit2=carryItem`. The
`nSpeed` and `nCarryItemEffect` reads are guarded on bits 1 and 2.
`appearance_update.go:34` writes `WriteByte(1)` — bit0 only — so the client
correctly skips both. **The encoder is not two bytes short.**

The genuine defect is the trailing `w.WriteInt(0) // completed set item id` at
line 39, which no export shows the client reading on any version. Task 6 removes
that one line and its `Decode` mirror.

This still changes bytes on the empty path, which is why Task 6 is separate from
Task 5 and why Task 14 re-pins eight evidence records rather than extending them.

### 3.3 `CharacterAppearanceUpdate` and `CharacterData` are nine columns, not ten

`design.md` §7 claims "all 10" for every op. The matrix disagrees:

- `STATUS.md:264` — `UPDATE_CHAR_LOOK` is `⬜` (n-a) on `gms_v48`, no opcode.
- `STATUS.md:180` — `SET_FIELD` is likewise `⬜` on `gms_v48`.

`CharacterSpawn` and `CharacterInfo` really are ten columns
(`STATUS.md:197`, `:86` — ✅ on nine, ❌ on `gms_v92`). Task 13 promotes v92 for
those two; Task 14 confirms the v48 `n-a` disposition for the other two rather
than trying to verify a cell that does not exist.

## 4. Two design-internal inconsistencies resolved at plan time

### 4.1 One `RingSet` cannot serve both wire shapes

`design.md` §3.1 defines `RingSet{Couple *PairRing; Friendship *PairRing;
Marriage *MarriageRing}` — singular pointers, correct for sites B/C/D. §3.3 then
writes `m.Rings.CoupleRecords` — a **list**, because site A is count-prefixed
(`Decode2` count ×3, v83 @0x4e6333/0x4e6361/0x4e638f) and its records carry a
13-byte `sPairCharacterName` the field blocks do not.

The plan splits them: **Task 2** builds `RingSet` + `EncodeField`/`DecodeField`
for sites B/C; **Task 3** builds `RingRecords` + `EncodeRecords`/`DecodeRecords`
for site A. Two tasks, two reviewer gates, one file.

### 4.2 `GetRingSet`'s equipment parameter

`design.md` §4.1 declares `GetRingSet(characterId, equipped []asset.Model)`.
The channel has no such list. `character.Model.Equipment()`
(`character/model.go:244`) returns `equipment.Model`, whose `Slots()` gives
`map[slot.Type]slot.Model`, and each `slot.Model`
(`equipment/slot/model.go:9-13`) carries **`Equipable *asset.Model` and
`CashEquipable *asset.Model`**. A cash-shop ring is a cash equip, so the join
reads `CashEquipable`. Task 10 takes `equipment.Model`.

This also gives the FR-15 selection rule real coordinates: ring slot positions
are `ring1` -12, `ring2` -13, `ring3` -15, `ring4` -16
(`libs/atlas-constants/inventory/slot/constants.go:21-35`). The `petRing*`,
`pet2Ring*`, `pet3Ring*` entries at -21/-29/-31/-37/-39/-45 are pet equipment
and are excluded.

## 5. Key files and why they matter

| File | Role |
|---|---|
| `libs/atlas-packet/model/ring.go` | New. The single codec all four sites share — §3.1's answer to "three hand-written copies is how four literals became eleven" |
| `libs/atlas-packet/model/avatar.go:47-110` | The version-gated small-model precedent Task 2 copies |
| `libs/atlas-packet/character/data.go:765-776` | Site A. Uses `t.Region() == "GMS" && t.MajorVersion() > N` exclusively |
| `libs/atlas-packet/character/clientbound/spawn.go:174-176` | Site B |
| `libs/atlas-packet/character/clientbound/appearance_update.go` | Site C. **No tenant import today** — Task 6 adds it |
| `libs/atlas-packet/character/clientbound/info.go:131` | Site D. One line |
| `services/atlas-channel/atlas.com/channel/monster/information/cache.go:93-149` | The cache precedent. **`door/` has no cache** — it is the REST precedent only |
| `services/atlas-channel/atlas.com/channel/listener/evict.go:24` | `RegisterEvictor`, safe from `init()` |
| `services/atlas-channel/atlas.com/channel/asset/model.go:110` | `CashId() int64` — the join key and the wire payload |
| `services/atlas-cashshop/atlas.com/cashshop/ring/` | The upstream read model Task 8 widens |
| `docs/packets/audits/VERIFYING_A_PACKET.md` | The playbook Tasks 13-14 follow verbatim |

## 6. Decisions made without asking

- **`ring.Type` / `ring.State` are re-declared channel-side, not promoted to
  `libs/atlas-constants`** (PRD §6 left this open). `libs/atlas-constants/item/constants.go:24`
  has `ClassificationRing = Classification(111)`, which is an item
  classification, not a pairing type — the cashshop package documents that
  distinction at `ring/model.go:9-11`, and the channel package inherits it.
- **The three new `ring.RestModel` fields land on this branch, not as an
  amendment to #1426** (`design.md` §9 item 2 left this open). Amending a PR
  after its review to serve a downstream consumer is worse than carrying the
  change in the consumer's own branch, where it is reviewed alongside its use.
- **`rings` / `hasMarriageRing` are appended last** in both constructor
  signatures, so existing positional calls read unchanged up to the new
  argument.
- **`RingSet` uses exported struct fields**, not the unexported+getter shape.
  It is constructed by callers in three packages and protects no invariant;
  `model.Avatar` and `model.Pet` are the same.

## 7. Task sizing

Fifteen tasks. `plan-lint.sh` reports **clean — 0 errors, 0 warnings**, so no
task trips the >6-files or >1-service F4 threshold. Nothing was deliberately
left oversized.

The two places the plan splits more finely than file count alone would demand,
and why:

- **Task 5 vs Task 6.** Sites B and D keep their bytes on the empty path; site C
  does not. A reviewer must be able to accept the first and reject the second.
- **Task 13 vs Task 14.** Task 13 *adds* matrix columns. Task 14 *re-pins* eight
  existing evidence records whose expected bytes changed. Those are different
  risks; the second can silently degrade a cell that was already ✅.

Task 1 produces no code at all. That is deliberate — see §3.1.

## 8. What execution must not assume

- **`RING_PURCHASED` cannot patch the cache in place.** `RingPurchasedBody`
  (`services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go:394-402`)
  carries no `cashId` for either half and no partner character id. Invalidate
  and let the next population re-fetch.
- **The `RING_PURCHASED` handler is at `consumer.go:496`**, not the `:461` that
  `design.md` §4.3 cites; registration is at `:120`. Re-grep rather than trusting
  either number.
- **`GetRingSet` never issues a REST call.** PRD §8: spawn is hot; population
  happens on character load. A cache miss returns the zero `RingSet`.
- **Resolve the `RingSet` once per broadcasting character** in
  `updateAppearance` (`kafka/consumer/asset/consumer.go:419`) — it runs inside a
  per-recipient closure driven by `ForSessionsInMap` (`:381`). Task 11 has a
  call-count test for exactly this.
- **`cash_rings.RingType` only admits `COUPLE` and `FRIENDSHIP`**
  (`ring/model.go:15-18`), and marriage-ring acquisition is a PRD §2 non-goal.
  `RingSet.Marriage` will be nil in practice; the field exists because the wire
  block does.
- **Do not hand-edit an IDA export.** They are regenerated by
  `packet-audit export` and a hand-edit is silently overwritten. If Task 1 finds
  an export comment wrong, record it for a follow-up refresh.
