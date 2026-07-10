# Echo of Hero — Map-Wide Buff Application — Design

Task: task-162-echo-of-hero-mapwide
Status: Approved PRD → Design
Created: 2026-07-10

---

## 1. Design-Phase Verification (FR-4 / OQ-1) — CLOSED

The PRD required IDA verification that the v83 client sends no affected-member
bitmap or extra payload for X005 casts before committing to "no decode change."
Verified against the v83 IDB (`MapleStory_dump.exe`, ida-mcp port 13342):

- **`CUserLocal::SendSkillUseRequest` (0x96d399)** is the single sender of the
  UseSkill packet (opcode 0x5B) for stat-change skills. Wire order:
  `Encode4(update_time)`, `Encode4(skillId)`, `Encode1(nSLV)`, then
  conditionals: antirepeat skills add `Encode2(x), Encode2(y)`; 4121006 adds
  `Encode4(javelinItemId)`; the affected-member bitmap byte is encoded **only
  when `dwAffectedMemberBitmap != 0`**; the mob section (`Encode1(count)` +
  `Encode4(id)*n`) **only when `nMobCount >= 0`**; then a trailing
  `Encode2(delay)`.
- **X005 is not in `is_antirepeat_buff_skill` (0x96d6ca)** — the id list spans
  1001003..5221000 plus Cygnus/Aran/Evan variants; none of 1005 / 10001005 /
  20001005 / 20011005 appear. No castX/castY bytes.
- **Every direct caller of `CUserLocal::DoActiveSkill_StatChange` (0x969e21)
  passes `dwTargetFlag = 1`** (two sites in `CUserLocal::DoActiveSkill` at
  0x967b79 / 0x968dde, one in `TryDoingPreparedSkill` at 0x96d06c — all
  `push 1`). Inside StatChange, `FindParty` (bitmap production) is gated by
  `test dwTargetFlag, 2` (0x96a117) and the mob-rect query by
  `test dwTargetFlag, 4` (0x96a12f). With flag=1 neither runs → bitmap arg is
  0 → **the bitmap byte is never encoded**, and nMobCount=-1 → **no mob
  section**.

**Conclusion:** an X005 cast arrives as exactly `updateTime(4) + skillId(4) +
skillLevel(1)` followed by a trailing `delay(2)` the current decoder correctly
leaves unread. FR-4 holds: `libs/atlas-packet` is untouched, X005 stays out of
`isPartyBuff` / `isMobAffectingBuff` / `isAntiRepeatBuffSkill`.

*Out-of-scope observation (recorded, not asserted):* the same analysis implies
v83 stat-change casts generally pass flag=1, which raises a question about how
non-beginner party buffs populate the bitmap Atlas decodes today (Heal
hand-rolls its own 0x5B packet with an unconditional `FindParty` +
`Encode1(bitmap)` at 0x96a69d, but Maple-Warrior-class buffs route through
StatChange). Whatever the resolution, it does not affect X005 (not in
`isPartyBuff`, so Atlas never reads a bitmap for it) and FR-1.1 freezes all
other skills' behavior. Flagged for a possible future audit; **unverified**
beyond what is stated here.

## 2. Decisions

| Question | Decision | Why |
|---|---|---|
| **D1** Routing mechanism | **Inline branch inside `UseSkill`'s generic buff step** (`common.go:107-111`), not a registry handler | The per-skill registry (`Lookup`) dispatches *after* the generic self+party apply, so a registered handler cannot replace recipient selection without double-applying to the caster or adding a suppression flag. The mount toggle precedent (short-circuit *before* the buff step, `common.go:100-105`) would force duplicating the effect-gating logic. The buff step already owns "who receives this effect" — branching there is the smallest honest change. |
| **D2** Recipient selector | **New `SelectMapWideRecipients(l, ctx, f, casterId)` in `recipients.go`**, returning `[]PartyRecipient` | Sits beside the two existing party selectors, reuses the `PartyRecipient` descriptor + Builder (id/hp fields already fit), and reuses the existing seams (`inMapCharacterIdsFunc`, `loadPartyMemberFunc`). The caster is enumerated by the session sweep like everyone else — applied exactly once (FR-1). |
| **D3** Hidden-GM check | **Per-recipient buff fetch via `buff.NewProcessor(l, ctx).GetByCharacterId(id)`; skip when any buff has `SourceId() == int32(skill.SuperGmHideId)`** | Task-156 (design OQ-2) establishes hide as a `DARK_SIGHT` buff with `SourceId == 9101004`, `duration = MaxInt32`. `buff.Model.SourceId()` exists today (`character/buff/model.go:32`), so the check is implementable now and becomes effective the moment task-156 lands (vacuously true until then — FR-2.2). New seam var `loadBuffsFunc` for tests. |
| **D4** Per-recipient failure policy | **Any fetch failure (character *or* buff list) skips that recipient with a debug log and a `fetch_failures` count; never aborts the cast** | FR-2.4. Uniform skip is simpler to reason about than fail-open-for-buffs/fail-closed-for-HP, and matches the existing `selectPartyMembers` behavior for member fetch errors (`recipients.go:169-173`). |
| **D5** Echo predicate | `isEchoOfHero(id skill.Id)` helper in the handler package using `skill.Is(id, BeginnerEchoOfHeroId, NoblesseEchoOfHeroId, LegendEchoOfHeroId, EvanEchoOfHeroId)` | All four ids already exist in `libs/atlas-constants/skill/constants.go` (DOM-21: no new constants). `skill.Is` is the established membership helper. |
| **D6** Enumeration/fetch shape | **Collect ids via the existing concurrent `ForSessionsInMap` sweep (mutex-guarded), then fetch/filter sequentially** | Mirrors `inMapCharacterIdsFunc` exactly (NFR concurrency note, `recipients.go:57-70`). Map population is bounded; 2 REST calls per candidate (character + buffs) is acceptable and stated in §5. Ids are sorted ascending before the apply loop for deterministic logs/tests. |
| **OQ-2** (PRD) exclusion phrasing | **Confirmed as specified**: exclude *hidden* GMs and dead characters; visible GMs are normal recipients | Matches the PRD interpretation; no counter-signal found during design. |

## 3. Architecture

Only `services/atlas-channel` changes (PRD §7). Two files.

### 3.1 `skill/handler/common.go` — routing branch (FR-1)

Replace the body of the existing buff step:

```go
if e.Duration() > 0 && len(e.StatUps()) > 0 {
    applyBuffFunc := buff.NewProcessor(l, ctx).Apply(f, characterId, int32(info.SkillId()), info.SkillLevel(), e.Duration(), e.StatUps())
    if isEchoOfHero(skillId) {
        applyToMap(l)(ctx)(f, characterId)(applyBuffFunc)
    } else {
        _ = applyBuffFunc(characterId)
        _ = applyToParty(l)(ctx)(f, characterId, info.AffectedPartyMemberBitmap())(applyBuffFunc)
    }
}
```

- `skillId` (`skill2.Id`) is already in scope from the mount check at
  `common.go:99`.
- The Echo branch does **not** call `applyBuffFunc(characterId)` separately —
  the map-wide recipient set includes the caster once (FR-1: caster never
  buffed twice).
- Everything earlier in `UseSkill` (HP/MP/item consume, cooldown, mount
  short-circuit) is untouched (FR-1.2); `applyToMobs` and the registry
  dispatch after the buff step are untouched (X005 has no `AffectedMobIds`
  and no registered handler, so both no-op for it) (FR-1.1).
- `applyToMap` follows the `applyToParty` curried shape
  (`common.go:342-354`): select recipients, run the operator per id, emit the
  summary log. Per-recipient `Apply` errors are logged and do not stop the
  loop.

### 3.2 `skill/handler/recipients.go` — map-wide selector (FR-2)

```go
// loadBuffsFunc is the recipient-buff-list seam tests can replace. Used by
// the map-wide selector to detect hidden GMs (SuperGmHide-sourced buff).
var loadBuffsFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]buff.Model, error) {
    return buff.NewProcessor(l, ctx).GetByCharacterId(characterId)
}

// SelectMapWideRecipients returns every character with a live session in the
// caster's field — INCLUDING the caster — excluding dead characters (Hp 0)
// and hidden GMs (any active buff with SourceId == SuperGmHideId; the hide
// representation established by task-156). Party membership, the client
// bitmap, and LT/RB rectangles are all ignored (FR-2.3). A per-recipient
// fetch failure skips that recipient and continues (FR-2.4).
func SelectMapWideRecipients(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32) ([]PartyRecipient, MapWideSelectionStats)
```

Algorithm:

1. `inMapCharacterIdsFunc(l, ctx, f)` — existing seam; field-scoped
   (world+channel+map+instance), therefore tenant-scoped and excludes
   other-map/channel/instance characters by construction (FR-2, AC-4).
2. Sort ids ascending (deterministic iteration).
3. Per id: `loadPartyMemberFunc` (existing seam) → on error, count
   `fetch_failures`, debug-log, continue. `Hp() == 0` → count `skipped_dead`,
   continue (FR-2.1).
4. `loadBuffsFunc` → on error, count `fetch_failures`, debug-log, continue;
   any buff with `SourceId() == int32(skill2.SuperGmHideId)` → count
   `skipped_hidden`, continue (FR-2.2).
5. Append `PartyRecipient` (id/x/y/hp/maxHp via the existing Builder).

`MapWideSelectionStats` is a small struct (`applied`, `skippedDead`,
`skippedHidden`, `fetchFailures` ints) consumed by the summary log —
mirrors the `buildSummaryFields` pattern rather than threading five loose
ints.

### 3.3 Buff application & visibility (FR-3)

Unchanged mechanics: each recipient goes through the same
`buff.Processor.Apply` operator (Kafka command to atlas-buffs, one command
per recipient — identical to today's party-buff fan-out). Foreign visibility
is whatever the existing emission already produces; note
`libs/atlas-packet/model/character_temporary_stat.go:131` registers
`EchoOfHero` with `NoOpForeignValueWriter`, i.e. the foreign temp-stat
encoding for this stat is already defined and requires no packet work
(FR-3.1, OQ-3 expected-yes).

### 3.4 Observability

One debug summary per cast, following the `mob_buff_apply_summary` style
(`common.go:296-308`):

```
echo_of_hero_apply_summary  caster, skill_id, skill_level, in_map,
                            applied, skipped_dead, skipped_hidden,
                            fetch_failures
```

## 4. Alternatives Considered

| Alternative | Verdict | Why rejected |
|---|---|---|
| **A. Add X005 to `isPartyBuff` and ride the bitmap path** | Rejected | The client sends no bitmap for X005 (§1 — verified); would desync the decoder by one byte, violates FR-4, and beginners are usually partyless so the bitmap path is semantically wrong anyway. |
| **B. Per-skill registry handler (like `heal/`)** | Rejected | Registry dispatch runs *after* the generic buff step (`common.go:117-121`), so the caster would already be buffed — the handler would need to suppress the generic step (new mechanism) or tolerate double-apply (violates FR-1). |
| **C. Mount-style pre-buff short-circuit returning early** | Rejected | Would duplicate the `Duration/StatUps` gating and the `Apply` construction inside a second code path; the chosen branch keeps one buff step with two recipient strategies. |
| **D. New atlas-buffs "apply to field" Kafka command (server-side fan-out)** | Rejected | New API surface (PRD §5 says none); atlas-buffs has no session/field enumeration today — that knowledge lives in atlas-channel's map processor. Per-recipient commands are the established pattern. |
| **E. Exclude hidden GMs via a character-service flag** | Rejected | No such flag exists; task-156 deliberately modeled hide as a buff (its OQ-2), so the buff query is the canonical representation. |

## 5. Performance & Concurrency Notes (PRD §8)

- Recipient count is bounded by map population. Cost per cast: 1 session
  sweep + per-candidate 1 character fetch + 1 buff fetch (2 REST calls). The
  party path today already does the character fetch per member; the buff
  fetch is the only new per-head cost. No caching layer is warranted for a
  level-200-gated skill cast; if profiling ever disagrees, batching belongs
  in a follow-up.
- `ForSessionsInMap` runs callbacks concurrently — id collection reuses the
  existing mutex-guarded `inMapCharacterIdsFunc` verbatim; the fetch/filter
  loop is sequential, so no new shared-state hazards.
- All lookups flow through the tenant-scoped `ctx`; the field-scoped
  enumeration is tenant-scoped already.

## 6. Testing (PRD §8, AC-8)

Seam-variable + Builder patterns only (no `*_testhelpers.go`):

- **Routing:** X005 cast (each of the four ids) → map-wide selector invoked,
  legacy self+party path not; non-X005 buff → legacy path only. Via stubbed
  `inMapCharacterIdsFunc` / `loadPartyMemberFunc` / `loadBuffsFunc` and a
  recorded apply operator (`SkillUsageInfoBuilder` to build casts).
- **Caster-once:** caster id appears in the in-map set; assert exactly one
  apply call for it.
- **Exclusions:** dead (Hp 0) skipped; hidden (buff with
  `SourceId == 9101004`, stubbed via `loadBuffsFunc`) skipped; other-map
  characters absent because the stubbed in-map set omits them.
- **FR-2.4:** character-fetch error and buff-fetch error each skip only the
  offending recipient; remaining recipients still applied.
- **Bit-order regression:** existing party selector tests unchanged
  (`recipients_test.go`) — guards FR-1.1.

Verification gate: `go test -race ./...`, `go vet ./...`, `go build ./...`
clean in atlas-channel; `tools/redis-key-guard.sh` clean;
`docker buildx bake atlas-channel` if `go.mod` changes (not expected — no new
module deps).

## 7. Version Scope (FR-5)

Server-side routing only; the four ids are version-stable constants and the
recipient logic is packet-free. The IDA verification in §1 is v83 (the PRD's
required target); no per-version gating exists to get wrong. Risk that a
later client (v87–jms) adds payload to X005 casts is nil in practice — the
skills predate v83 and the beginner cast path is stable — and would surface
as a decode desync in existing tooling, not silently.

## 8. Resolved Open Questions

- **OQ-1 (wire format):** closed — §1, no extra bytes.
- **OQ-2 (exclusion phrasing):** hidden GMs + dead excluded; visible GMs
  receive. Confirmed as PRD-specified.
- **OQ-3 (foreign visual):** expected-yes with evidence
  (`NoOpForeignValueWriter` registration already covers EchoOfHero foreign
  encoding); final confirmation remains an implementation-testing item as the
  PRD states.
