# Echo of Hero — Map-Wide Buff Application — Design

Task: task-162-echo-of-hero-mapwide
Status: Approved PRD → Design (v2)
Created: 2026-07-10
Updated: 2026-08-07 — rebased onto main (254 commits). task-156, task-187 and the
per-skill handler registry all landed in the interim; decisions D1/D2/D3/D5 are
**revised** below, and §7 is rewritten for the full 11-version set.

---

## 0. What changed on rebase (read this first)

The v1 design was written against a `main` that no longer exists. Three landed
tasks invalidate its core decisions:

| Landed | Effect on this design |
|---|---|
| **task-187** (version-aware id semantics) | `UseSkill` now resolves the incoming wire id to a `skill.Identity` through the tenant's version set (`common.go:180-183`) and dispatches the registry on the **Identity** (`common.go:194-201`). Raw wire-id compares against version-divergent ids are banned by `tools/skill-job-id-guard.sh`. |
| **task-156** (GM hide / heal+dispel) | `character/buff.IsGmHidden(ctx, bs)` now exists as the canonical, **version-aware** hidden-GM check (`character/buff/hidden.go:21`). The v1 design's hand-rolled `SourceId() == 9101004` compare is version-**incorrect** (hide is `5101004` at v0.48). Task-156 also added `SelectAllCharactersInMap` and the `healdispel` handler package — a near-exact template for this task. |
| **per-skill handler registry** | `skill/handler/registry.go` is keyed on `skill.Identity`, with seven handler subpackages already registered. This is now the established routing mechanism for exactly this shape of skill. |

The net effect is that this task gets *smaller*: the recipient selector and the
hidden-GM predicate both already exist, and version correctness falls out of the
registry's Identity keying instead of needing per-version logic.

## 1. Design-Phase Verification (FR-4 / OQ-1) — CLOSED (v83)

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

**Scope of this verification:** v83 only. The other ten versions are covered by
the negative-assertion argument in §7, not by decompilation. Stated as a
residual risk (PRD OQ-1), not a verified cross-version claim.

*Out-of-scope observation (recorded, not asserted):* the same analysis implies
v83 stat-change casts generally pass flag=1, which raises a question about how
non-beginner party buffs populate the bitmap Atlas decodes today (Heal
hand-rolls its own 0x5B packet with an unconditional `FindParty` +
`Encode1(bitmap)` at 0x96a69d, but Maple-Warrior-class buffs route through
StatChange). Whatever the resolution, it does not affect X005 (not in
`isPartyBuff`, so Atlas never reads a bitmap for it) and FR-1.3 freezes all
other skills' behavior. Flagged for a possible future audit; **unverified**
beyond what is stated here.

## 2. Decisions

| Question | Decision | Why |
|---|---|---|
| **D1** *(revised)* Routing mechanism | **Per-skill registry handler** — a new `skill/handler/echoofhero/` package calling `Register` for all four identities. `common.go` is **not modified**. | v1 rejected this on the grounds that "registry dispatch runs *after* the generic self+party apply, so the handler would double-apply the caster." That premise was half-right and led to the wrong conclusion. The generic step applies to **the caster only** — X005 is not an `isPartyBuff`, so its bitmap is always `0`, and `selectPartyMembers` early-returns `nil` on a zero bitmap (`recipients.go:236-238`), making the `applyToParty` call a no-op. So the handler does not need to *replace* recipient selection, only to *extend* it: apply to everyone **except** the caster. Caster gets exactly one copy (FR-1.1), and the diff shrinks to one new package plus one blank import. |
| **D2** *(revised)* Recipient selector | **Reuse the existing `SelectAllCharactersInMap(l, ctx, f)`** (`recipients.go:204`); filter caster / dead / hidden inside the handler. No new selector. | task-156 added this exact map-wide selector for GM Heal + Dispel — same field-scoped `ForSessionsInMap` sweep, same `PartyRecipient` return, same `inMapCharacterIdsFunc` / `loadPartyMemberFunc` seams the v1 design planned to hand-roll. It deliberately applies no HP and no hidden filter, leaving those to the caller, which is precisely what this task needs. Adding a second near-identical selector would be duplication. |
| **D3** *(revised)* Hidden-GM check | **`character/buff.IsGmHidden(ctx, bs)`** (`character/buff/hidden.go:21`), fed by `buff.NewProcessor(l, ctx).GetByCharacterId(id)`. | v1 specified a raw `SourceId() == int32(skill.SuperGmHideId)` compare. That is **version-incorrect**: SuperGM Hide is wire `5101004` at v0.48 and `9101004` at v0.62+, so the literal compare silently never matches a v0.48 hide buff — the exact bug class `tools/skill-job-id-guard.sh` bans. `IsGmHidden` resolves `SourceId` through the tenant's version set before comparing to the `SuperGmHide` **Identity**, and additionally skips expired buffs and correctly distinguishes GM hide from Rogue Dark Sight (which shares the `DARK_SIGHT` stat). This is the single largest correctness fix in the rebase. |
| **D4** *(unchanged)* Per-recipient failure policy | **Any fetch failure (character *or* buff list) skips that recipient with a debug log and a counter; never aborts the cast** | FR-2.5. Uniform skip is simpler to reason about and matches `SelectAllCharactersInMap`'s own per-id skip-on-fetch-failure behavior (`recipients.go:209-212`). |
| **D5** *(revised)* Echo predicate | **None — deleted.** Registration is `Register(skill.BeginnerEchoOfHero, Apply)` ×4 on the Identity constants (`libs/atlas-constants/skill/identities_gen.go:11` et al). | v1 planned an `isEchoOfHero(id skill.Id)` raw-wire-id membership helper. Under task-187 the dispatch already resolves wire→Identity before `Lookup`, so an id predicate is both redundant and the banned idiom. The `resurrection` package is the multi-identity registration precedent (`resurrection.go:27-29`). |
| **D6** *(unchanged)* Enumeration/fetch shape | **Concurrent `ForSessionsInMap` id sweep (inside the reused selector), then sequential fetch/filter**; recipients sorted ascending for deterministic logs/tests | Mirrors `inMapCharacterIdsFunc` (`recipients.go:64-78`) and the `healdispel` loop. Map population is bounded; the per-head buff fetch is the same cost `healdispel` already pays map-wide. |
| **D7** *(new)* Deps seam shape | **A `echoDeps` struct of function seams** (`selectInMap`, `isGmHidden`, `applyBuff`), built in `Apply` and consumed by a pure `applyEchoOfHero(l, f, casterId, info, e, d)` core | Directly mirrors `healDispelDeps` (`healdispel/healdispel.go:174-206`) and `hideDeps`. Keeps the fan-out loop unit-testable offline with no Kafka/REST, satisfying the no-`*_testhelpers.go` rule. |

## 3. Architecture

Only `services/atlas-channel` changes. Two files touched, one of them a single line.

### 3.1 `skill/handler/echoofhero/echoofhero.go` — new package (FR-1)

```go
func init() {
    channelhandler.Register(skill2.BeginnerEchoOfHero, Apply)
    channelhandler.Register(skill2.NoblesseEchoOfHero, Apply)
    channelhandler.Register(skill2.LegendEchoOfHero, Apply)
    channelhandler.Register(skill2.EvanEchoOfHero, Apply)
}

// echoDeps holds the fan-out collaborators as function seams so the core loop
// is unit-testable offline (no Kafka/REST/session).
type echoDeps struct {
    selectInMap func(f field.Model) []channelhandler.PartyRecipient
    isGmHidden  func(characterId uint32) (bool, error)
    applyBuff   model.Operator[uint32] // = func(uint32) error; what buff.Processor.Apply returns
}

// applyEchoOfHero is the tested core: fan the already-constructed buff operator
// out to every live-session character in the field except the caster, skipping
// dead characters and hidden GMs.
func applyEchoOfHero(
    l logrus.FieldLogger, f field.Model, casterId uint32,
    info packetmodel.SkillUsageInfo, e effect.Model, d echoDeps,
) error
```

Core loop:

1. **Gate (FR-1.2):** `if e.Duration() <= 0 || len(e.StatUps()) == 0 { return nil }`
   — mirrors the generic buff step so a statup-less effect fans out to nobody.
2. `d.selectInMap(f)` → `[]PartyRecipient`, field-scoped
   (world+channel+map+instance), therefore tenant-scoped and excluding
   other-map/channel/instance characters by construction (AC-4).
3. Sort recipients by id ascending (deterministic iteration).
4. Per recipient:
   - `r.Id() == casterId` → skip, count `skipped_caster` (FR-1.1: already buffed
     by the generic step).
   - `r.Hp() == 0` → skip, count `skipped_dead` (FR-2.2).
   - `d.isGmHidden(r.Id())` → on error count `fetch_failures` + debug-log +
     continue (FR-2.5); on true count `skipped_hidden` + continue (FR-2.3).
   - `d.applyBuff(r.Id())` → on error count `apply_failures` + log + continue.
5. Emit the summary log (§3.3).

`Apply` builds the production deps exactly as `healdispel.Apply` does — including
the `isGmHidden` closure over `bp.GetByCharacterId` + `buff.IsGmHidden(ctx, bs)`,
and `applyBuff` from
`buff.NewProcessor(l, ctx).Apply(f, casterId, int32(info.SkillId()), info.SkillLevel(), e.Duration(), e.StatUps())`
— the same operator construction the generic step uses for the caster (FR-3).

### 3.2 `skill/handler/registrations/registrations.go` — one line

```go
_ "atlas-channel/skill/handler/echoofhero"   // Echo of Hero map-wide — task-162
```

### 3.3 Observability

One debug summary per cast, following the `mob_buff_apply_summary` style:

```
echo_of_hero_apply_summary  caster, skill_id, skill_level, in_map,
                            applied, skipped_caster, skipped_dead,
                            skipped_hidden, fetch_failures, apply_failures
```

### 3.4 Buff application & visibility (FR-3)

Unchanged mechanics: each recipient goes through the same
`buff.Processor.Apply` operator (one Kafka command per recipient — identical to
today's party-buff fan-out). Foreign visibility is whatever the existing
emission already produces; note
`libs/atlas-packet/model/character_temporary_stat.go:133` registers
`EchoOfHero` with `NoOpForeignValueWriter`, i.e. the foreign temp-stat
encoding for this stat is already defined and requires no packet work
(FR-3.1, OQ-3 expected-yes).

### 3.5 What is explicitly NOT touched

`common.go`, `recipients.go`, `libs/atlas-packet`, `libs/atlas-constants`,
`services/atlas-data`, `services/atlas-buffs`, and every seed template. A
non-empty diff in any of these is a design violation, not a refinement.

## 4. Alternatives Considered

| Alternative | Verdict | Why rejected |
|---|---|---|
| **A. Add X005 to `isPartyBuff` and ride the bitmap path** | Rejected | The client sends no bitmap for X005 (§1 — verified); would desync the decoder by one byte, violates FR-4, and beginners are usually partyless so the bitmap path is semantically wrong anyway. |
| **B. Inline branch in `UseSkill`'s generic buff step** *(the v1 decision)* | Rejected | Now that seven handler subpackages exist, an inline `if isEchoOfHero(...)` in `common.go` is the odd one out — and it requires the banned raw-wire-id predicate (D5). The registry handler achieves the same behavior with a zero-line `common.go` diff. |
| **C. Mount-style pre-buff short-circuit returning early** | Rejected | Would duplicate the `Duration/StatUps` gating and the `Apply` construction in a second code path, and would have to re-apply the caster's copy itself. |
| **D. New `SelectMapWideRecipients` selector** *(the v1 decision)* | Rejected | `SelectAllCharactersInMap` already does the enumeration; the caster/dead/hidden filtering is caller-specific policy that `healdispel` also keeps in the handler. A second selector would duplicate the sweep. |
| **E. New atlas-buffs "apply to field" Kafka command (server-side fan-out)** | Rejected | New API surface (PRD §5 says none); atlas-buffs has no session/field enumeration — that knowledge lives in atlas-channel's map processor. Per-recipient commands are the established pattern. |
| **F. Exclude hidden GMs via a character-service flag** | Rejected | No such flag exists; task-156 deliberately modeled hide as a buff, so the buff query via `IsGmHidden` is the canonical representation. |
| **G. Per-version registration table (register only the identities a version ships)** | Rejected | Unnecessary and fragile. Registration is global and Identity-keyed; dispatch resolves wire→Identity per tenant, so an identity a version does not bind is simply unreachable. A hand-maintained per-version table would be a second source of truth that can drift from the generated sets. See §7. |

## 5. Performance & Concurrency Notes (PRD §8)

- Recipient count is bounded by map population. Cost per cast: 1 session sweep +
  1 character fetch per head (inside `SelectAllCharactersInMap`) + 1 buff fetch
  per head (hidden check). This is the **same profile `healdispel` already runs
  map-wide** on every SuperGM Heal + Dispel cast, so it is established rather
  than novel. No caching layer is warranted for a level-200-gated cast; if
  profiling ever disagrees, batching belongs in a follow-up and would benefit
  `healdispel` equally.
- `ForSessionsInMap` runs callbacks concurrently — the id collection reuses the
  existing mutex-guarded `inMapCharacterIdsFunc` verbatim; the fetch/filter loop
  is sequential, so no new shared-state hazards and no new goroutines
  (`tools/goroutine-guard.sh` stays clean).
- All lookups flow through the tenant-scoped `ctx`; the field-scoped
  enumeration is tenant-scoped already.

## 6. Testing (PRD §8, AC)

Seam-struct + Builder patterns only (no `*_testhelpers.go`), mirroring
`healdispel_test.go` and `mprecovery_test.go`:

- **Registration:** `channelhandler.Lookup(id)` returns a non-nil handler for
  each of the four identities (the `timeleap_test.go:186` / `resurrection_test.go:104`
  precedent). This is the test that proves the version story — see §7.
- **Caster-once:** caster id present in the selector's return; assert `applyBuff`
  is *not* called for it (its copy comes from the generic step).
- **Exclusions:** dead (`Hp()==0`) skipped; hidden (`isGmHidden` returns true)
  skipped; other-map characters absent because the stubbed selector omits them.
- **FR-2.5:** an `isGmHidden` error skips only that recipient; remaining
  recipients still applied.
- **FR-1.2 gate:** `Duration()==0` and empty `StatUps()` each fan out to nobody.
- **Regression guard:** `common.go` and `recipients.go` are untouched, so the
  existing `recipients_test.go` / `common_*_test.go` suites are the FR-1.3
  guard unchanged.

Verification gate: `go test -race ./...`, `go vet ./...`, `go build ./...`
clean in atlas-channel; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
`tools/skill-job-id-guard.sh`, `tools/lint.sh --check` clean;
`docker buildx bake atlas-channel` only if `go.mod` changes (not expected — no
new module deps).

## 7. Version Scope (FR-5) — all 11 provisioned versions

### 7.1 The version set

This task targets the **tenant version set of 11** — every version with a
generated identity set under `libs/atlas-constants/skill/version_<key>_gen.go`
and a seed template under `services/atlas-configurations/seed-data/templates/`:

`gms_v12`, `gms_v48`, `gms_v61`, `gms_v72`, `gms_v79`, `gms_v83`, `gms_v84`,
`gms_v87`, `gms_v92`, `gms_v95`, `jms_v185`.

This is deliberately **not** the packet coverage matrix's 9 columns
(`docs/packets/PROCESS.md` §Version set), which omit `gms_v12` and `gms_v92`.
The matrix axis tracks packet-codec verification; FR-4 makes this a zero-packet
change, so the relevant axis is which versions a tenant can be provisioned on.

### 7.2 Per-version X005 availability

Derived from the `Id → Identity` maps in each `version_<key>_gen.go`:

| Version key | Beginner `1005` | Noblesse `10001005` | Legend `20001005` | Evan `20011005` |
|---|:---:|:---:|:---:|:---:|
| `gms_v12`  | — | — | — | — |
| `gms_v48`  | — | — | — | — |
| `gms_v61`  | ✓ | — | — | — |
| `gms_v72`  | ✓ | ✓ | — | — |
| `gms_v79`  | ✓ | ✓ | ✓ | — |
| `gms_v83`  | ✓ | ✓ | ✓ | — |
| `gms_v84`  | ✓ | ✓ | ✓ | ✓ |
| `gms_v87`  | ✓ | ✓ | ✓ | ✓ |
| `gms_v92`  | ✓ | ✓ | ✓ | ✓ |
| `gms_v95`  | ✓ | ✓ | ✓ | ✓ |
| `jms_v185` | ✓ | ✓ | ✓ | ✓ |

The v1 design's claim that "the four ids are version-stable constants" was
**wrong** — availability varies across seven of the eleven versions. It happened
not to matter for the v1 six-version list (v83+), which is why it went unnoticed.

### 7.3 Why no per-version code is needed (FR-5.1)

Version correctness is structural, from two mechanisms that already exist:

1. **Registry dispatch is Identity-keyed after a version-set resolve.**
   `UseSkill` computes `set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())`
   then `castId, ok := set.Skill.Resolve(skill.Id(info.SkillId()))`, and only
   calls `Lookup(castId)` when `ok` (`common.go:180-201`). On `gms_v61` the only
   wire id that resolves to an EchoOfHero identity is `1005`; on `gms_v48` none
   do, so the handler is unreachable and the task is inert there. Registering all
   four identities unconditionally is therefore correct on all 11 versions — the
   version set itself is the gate.
2. **The hidden-GM check resolves through the same version set.**
   `buff.IsGmHidden` maps the buff's stored wire `SourceId` back to an Identity
   per tenant version, so it matches `5101004` on v0.48 and `9101004` on v0.62+
   without the handler knowing either literal (D3).

Consequently there is **no `MajorVersion()` comparison, no gates.yaml entry, and
no version table in the handler** — and `tools/skill-job-id-guard.sh` stays clean
because no wire-id literal appears in the diff at all.

### 7.4 What is verified vs. assumed

- **Verified (repo source):** the availability table in §7.2, the Identity
  constants, the resolve-then-Lookup dispatch order, and `IsGmHidden`'s
  version-aware resolve. The registration test (§6) mechanically proves all four
  identities are reachable.
- **Verified (IDA, v83 only):** the wire format in §1.
- **Assumed (OQ-1):** that no other version adds payload to an X005 cast. The
  argument is that FR-4 asserts a negative and a violation would surface as a
  decode desync on the shared generic skill-use path — loud, not silent. Not
  re-decompiled per version; recorded as residual risk.

## 8. Resolved Open Questions

- **OQ-1 (wire format):** closed for v83 (§1); cross-version treated as residual
  risk with the argument in §7.4. Re-open on any observed X005 decode desync.
- **OQ-2 (exclusion phrasing):** closed — hidden GMs + dead excluded; visible GMs
  receive.
- **OQ-3 (foreign visual):** expected-yes with evidence
  (`NoOpForeignValueWriter` registration already covers EchoOfHero foreign
  encoding); final confirmation remains an implementation-testing item.
