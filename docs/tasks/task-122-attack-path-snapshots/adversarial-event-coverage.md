# Adversarial event-coverage review — task-122 (attack-path-snapshots)

Scope: `git diff main...HEAD` for the branch above, treating
`docs/tasks/task-122-attack-path-snapshots/event-coverage.md` as a claim to
falsify. Independent producer-side enumeration of every event that can change
character core stats, skills, inventory/equips, or buffs, cross-checked
against every atlas-channel consumer arm (rich-apply, invalidate, or
demonstrably-unreachable).

## Method

1. Enumerated every `statChangedProvider(...)` call site in
   `services/atlas-character/atlas.com/character/character/processor.go`
   (26 sites) and manually checked, at each, that the `stats []stat.Type`
   list and the `values map[string]interface{}` argument agree key-for-key
   with atlas-channel's `statValueKeys` table.
2. Read every atlas-channel Kafka consumer touched by the diff
   (`character`, `skill`, `asset`, `compartment`, `buff`) end to end,
   matching each producer-side event to its snapshot-side handler (or its
   documented invalidate/no-op disposition).
3. Traced the attack path's actual snapshot reads
   (`character_attack_common.go`, `character_attack_combo.go`,
   `character_attack_combo_drain.go`, `character_attack_projectile.go`,
   `character_attack_energy_charge.go`) to determine which snapshot fields
   are load-bearing for damage/gating math, as opposed to fields the
   snapshot carries but the attack path never reads.
4. Checked the mixed-version-deploy degrade path directly in
   `registry.go`'s `ApplyStatChanged`.

## 1. STAT_CHANGED `Values` sweep (producer side) — PASS, no gap found

All 26 `statChangedProvider(...)` call sites in
`services/atlas-character/atlas.com/character/character/processor.go` were
read individually (job/hair/face/skin change: lines 611/645/679/713;
experience award/deduct: 784/830; level award: 871; meso change (3 sites):
917/979/1015; fame: 1048; AP-distribute reject/success: 1067/1131/1142/1150;
SP-distribute: 1194; HP/MP change/clamp: 1453/1505/1540/1575/1611;
level-up growth: 1730; job-change growth: 1991; reset-stats: 2282;
rebalance-AP: 2336; AP-transfer: 2583).

- Every site's `stats []stat.Type` argument has a 1:1 key match against
  `statValueKeys` (`services/atlas-channel/.../character/snapshot/registry.go:246-265`)
  for every type it emits, **except** `stat.TypeAvailableSP`
  (SP-distribute, line 1194) and the two AP-distribute reject-branch sites
  (1067, 1131, both emit `[]stat.Type{}, nil` — an empty `Updates` list, a
  legitimate no-op). `AvailableSP` is deliberately absent from
  `statValueKeys` (comment at `registry.go:267-270`: it "is a per-book
  string table on the model" the snapshot can't apply in place), so
  `ApplyStatChanged` correctly falls into the "unknown key → invalidate"
  branch (`registry.go:338-343`) rather than corrupting anything.
- `TestStatChanged_ValuesCompleteOnHotPaths`
  (`services/atlas-character/atlas.com/character/character/stat_values_test.go:233`)
  independently re-asserts this for the 10 hottest sites by decoding the
  actual emitted event and diffing `Values` against the re-read DB column.
  Ran it directly: `go test ./character/... -run
  TestStatChanged_ValuesCompleteOnHotPaths -v` → **PASS** (10/10 subtests).

**Verdict: no missing-Values gap.** event-coverage.md's own claim ("Values
nil on most paths pre-change") is now correctly resolved everywhere STAT_CHANGED
is emitted.

## 2. Consumer-side coverage sweep — matches claims, one real omission

Read `kafka/consumer/{character,skill,asset,compartment,buff}/consumer.go`
end to end (not just the handler names event-coverage.md lists).

- **character**: `handleSnapshotStatChanged` (`consumer.go:556-567`) applies
  in place via `ApplyStatChanged`, which safely invalidates on any
  unknown/missing/non-numeric key (`registry.go:324-367`) — this is also
  the answer to the mixed-version-deploy question (§3 below).
  `handleSnapshotLevelChanged`/`ExperienceChanged`/`MapChanged` all present
  and match the audit.
- **skill**: CREATED/UPDATED/DELETED all have snapshot handlers
  (`consumer.go:200-240`); DELETED is new (event-coverage.md correctly
  flagged this as a gap and the diff closes it).
- **asset**: CREATED/UPDATED/ACCEPTED (full replace by AssetId),
  QUANTITY_CHANGED (absolute), MOVED (absolute slot by AssetId), DELETED
  (remove), RELEASED/EXPIRED (invalidate) — all present
  (`consumer.go:618-724`), matches the audit exactly.
- **compartment**: CREATED/DELETED/CAPACITY_CHANGED/MERGE_COMPLETE/
  SORT_COMPLETE all invalidate inventory (`consumer.go:212-275`) — matches
  the audit's disposition.
- **buff**: APPLIED → `UpsertBuff`, EXPIRED → `RemoveBuff`
  (`consumer.go:636-668`). **event-coverage.md §5 never mentions
  `EVENT_STATUS_TYPE_STAT_UPDATED` at all** — its producer table for
  atlas-buffs lists only APPLY/CANCEL/CANCEL_ALL/CANCEL_BY_TYPES/expiration
  ticker, omitting `UpdateStatValue` (`services/atlas-buffs/atlas.com/buffs/character/processor.go:229-256`),
  which emits `STAT_UPDATED` for a buff whose stat *amount* changed in
  place (as opposed to APPLIED for a brand-new buff). atlas-channel's
  packet layer has a full handler for it
  (`handleStatusEventStatUpdated`, `consumer.go:219-240`), but **there is
  no `handleSnapshotBuffStatUpdated`** — the snapshot buff feed has no
  handler for this event type at all.

  Traced both current producers of `STAT_UPDATE` commands to determine
  blast radius: Aran Combo Attack orb count
  (`character_attack_combo.go:147-159`, `character_skill_use.go:187-194`)
  and Energy Charge bar amount
  (`character_attack_energy_charge.go:124`). Both are amount-sensitive
  reads, but **neither reads the amount through the snapshot's buff list**:
  Enrage's orb-cap gate reads via a live `buff.NewProcessor(...).GetByCharacterId`
  call (`character_skill_use.go:144`, not `sp`/snapshot), and Energy Charge's
  gate reads via the dedicated `buff.GetEnergyMirror()` singleton
  (`character_attack_energy_charge.go:182`), which is updated directly by
  the same event handler that also (correctly) skips the snapshot. The
  attack path's only snapshot-buff reads are presence-based (SOUL_ARROW /
  SHADOW_PARTNER gates in `character_attack_projectile.go`) or read a
  buff's static per-cast Amount that STAT_UPDATED never mutates (Combo
  Drain heal percent, `character_attack_combo_drain.go:114`). The shadow
  verifier (`shadow.go:139-142`) independently confirms the intended
  buff read-set is presence-only for SOUL_ARROW/SHADOW_PARTNER.

  **Disposition: non-blocking.** This is a real omission from
  event-coverage.md's producer enumeration — a full "sweep, don't
  spot-check" of `EVENT_TOPIC_CHARACTER_BUFF_STATUS` would have found
  `STAT_UPDATED` — and it is a latent landmine: any *future* buff whose
  damage-relevant amount is mutated via `UpdateStatValue` and read through
  `sp.GetBuffs()`/`c.Buffs()` on the attack path would silently freeze at
  its APPLIED-time value forever (the registry never creates entries and
  never receives an update for this event type, so it just goes stale with
  no invalidation and no error). It is not exploitable today because both
  current amount-mutating call sites were deliberately routed around the
  snapshot into dedicated mirrors. Flagging so the next buff feature that
  touches `UpdateStatValue` doesn't assume the snapshot is complete.

## 3. Mixed-version deploy seam — PASS

`ApplyStatChanged` (`registry.go:324-367`) treats a nil `values` map (old
atlas-character talking to new atlas-channel) identically to a present-but-
incomplete map: for every `u` in `updates`, `values[key]` is looked up with
the two-value form; a nil map simply always misses, so `present == false`
→ `coreValid = false` (invalidate), never a zero-value apply. The reverse
direction (new atlas-character, old-shape atlas-channel that doesn't exist
here) is moot — this branch only lands on this repo's own worktree. No
partial/zero corruption path exists; the degrade is invalidation only, both
directions, confirmed by reading the code (not run against a live rollout).

## 4. Cross-check against event-coverage.md

- §5 (buffs): misses `STAT_UPDATED` entirely — see finding above.
- All other sections (§1 character core, §2 position, §3 skills, §4
  inventory, §6 effective stats, §7 skill data, §8 monster) were
  independently re-derived from the producer/consumer source in this
  review and matched the audit's claims and file:line citations, including
  the "why it's safe" reasoning for accepted gaps (session-scoped eviction
  for skill bulk-delete and LOGOUT map changes; reservation-registry parity
  with REST; RequestReserve loop already fixed upstream). No other
  falsifiable claim found.
- Session eviction: `snapshot.GetRegistry().Evict` fires from
  `session.Processor.Destroy` (`session/processor.go:416`), which is the
  single funnel for logout, character deletion (client disconnect), and
  channel transfer — confirmed this covers the "character deletion" and
  "cross-channel transfer" mutation classes named in the review brief.
- GM/admin commands: atlas-character exposes no GM-specific stat-mutation
  command outside the 26 sites already swept in §1 (grepped for a
  dedicated admin/GM stat-set path; none exists — GM tooling routes
  through the same RESET_STATS/CHANGE_* commands already covered).
- Cash-shop / pending-change resolution
  (`services/atlas-character/atlas.com/character/pending_change/*`,
  applied on logout via `handleLogoutApplyPendingChanges`) is pre-existing
  code untouched by this diff; its model carries `changeType`,
  `requestedName`, `destinationWorldId`, `assetId` — name-change and
  world-transfer requests, not stat/inventory/buff grants — so it does not
  touch any field the attack path's snapshot reads. Confirmed by reading
  `pending_change/model.go`; not read further (out of the diff's blast
  radius and out of the attack read-set).

## Not evaluable

- Pet/mount stat effects (buffs granted by pet skills, mount speed/jump)
  were not traced end-to-end; the review brief calls them out but they did
  not surface as attack-damage-relevant in the files the diff touches, and
  a full trace would require reading atlas-pets producers, which the diff
  does not touch.

## Verdict rationale

No mutation path was found that leaves the snapshot stale for a field the
attack path actually reads with no invalidation — the one real coverage
gap found (buff `STAT_UPDATED` → snapshot) has no live blast radius on the
current attack path because both amount-mutating producers were
deliberately routed around the snapshot into their own mirrors. That gap is
real, was missed by the branch's own coverage audit, and should be closed
or explicitly documented as an architectural invariant ("STAT_UPDATED
never touches the snapshot buff feed — new buff amount reads on the attack
path MUST NOT use `sp.GetBuffs()`") before it can silently regress.
