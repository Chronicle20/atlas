# task-167 (Homing Beacon / Bullseye) — Verification Results

Recorded at the final commit on `task-167-homing-beacon-bullseye`
(`d9dc16e2b` — `refactor(channel): make BeaconEntry immutable with getters`),
25 commits ahead of `main`. This document is the honest record for Task 14
of `plan.md`; it distinguishes measured / inferred / unverified exactly as
the evidence docs under `evidence/` do — it does not restate their content,
only summarizes and cites it.

## 1. Gates (all re-verified by the controller at this commit)

| Gate | Module(s) | Result |
|---|---|---|
| `go build ./...` | `libs/atlas-packet`, `services/atlas-buffs/atlas.com/buffs`, `services/atlas-channel/atlas.com/channel` | clean, all three |
| `go vet ./...` | same three | clean, all three |
| `go test -race ./...` | same three | clean, all three (0 failures; `libs/atlas-packet` 78 packages, `services/atlas-buffs` 12 test packages, `services/atlas-channel` 98 packages) |
| `docker buildx bake atlas-buffs atlas-channel` | — | exit 0, both images built |
| `tools/redis-key-guard.sh` | repo root | exit 0 (beacon mirror is a plain in-memory map; no Redis keys touched outside `libs/atlas-redis`) |
| `tools/goroutine-guard.sh` | repo root | exit 0 |
| `tools/skill-job-id-guard.sh` | repo root | exit 0 — "clean (14 divergent const(s) checked)"; `5211006`/`5220011` are not on the task-187 divergence list, so the raw `sid != skill3.OutlawHomingBeaconId` comparison in `character_attack_common.go` is permitted |
| `tools/lint.sh --check` | — | **NOT run.** Known false-fail in this environment (`ui:node-missing`, no Node 22) regardless of whether any changes touched atlas-ui — it stalled two earlier subagent dispatches (~10 min each) before being excluded. Must still pass in CI. |
| `TestCTS*` subtests | `libs/atlas-packet/model` | **73 pass** (`go test ./model/ -run TestCTS -v` → 73 `--- PASS` lines: 20 top-level + 53 table-driven subtests), spanning all nine in-scope versions |

Code review (Task 14 Step 5) was run via `superpowers:requesting-code-review`:
`audit-plan-adherence.md` (plan-adherence-reviewer) found all 82 checkbox
items map to real work, one deliberate documentation gap (checkboxes
unticked — this document and Job 1 of this closeout fix that) and no
substance gaps. `audit-backend-guidelines.md` (backend-guidelines-reviewer)
raised one Important finding (`BeaconEntry` used exported mutable fields,
violating the immutable-domain-model pattern) and one Minor (see §4 below);
the Important finding was fixed in the final commit `d9dc16e2b`
(`refactor(channel): make BeaconEntry immutable with getters`) — re-verified
here by re-reading `character/buff/beacon.go`, which now exposes
`SourceId()`/`Level()`/`MobId()` getters over private fields.

## 2. Where the plan's Task 14 Step 4a checklist is now stale

`plan.md`'s Step 4a checklist (written before Task 7's IDA pass and Task 8's
implementation) makes four assertions that no longer hold as literal facts.
All four are annotated in place in `plan.md` rather than silently ticked.

1. **Empty-CTS length assumption.** The checklist assumes every in-scope
   version's empty CTS is `16+2+110=128` bytes. **v61 is deliberately 106**
   (`16+2+88`) — its client two-state group has 6 members, not 7. This was
   IDA-measured twice independently (`evidence/two-state-group-per-version.md`
   §v61, and the Task 7b re-pass documented in `.superpowers/sdd/plan/progress.md`),
   gated in `libs/atlas-packet/model/character_temporary_stat.go` via
   `isGmsV61`, and pinned by `TestCTSHomingBeaconPre95AbsentStaysEmpty`'s
   v61 subtest expecting 106. Following the checklist literally — "every
   version is 128" — would mean reverting a real fix.
2. **"Any UNVERIFIED verdict blocks completion."** `evidence/two-state-group-per-version.md`
   §v79 and §v84 are titled "block sizes UNVERIFIED, not verified": both
   IDBs lack constructor/RTTI/vtable symbols, and two independent passes
   (documented in `progress.md` "Task 7 ROUND 2") exhausted every
   approach tried (name search, binary-wide immediate search with
   individually-checked false positives, operator-new search, reset-function
   tracing, sibling-helper tracing). What **is** confirmed for both versions:
   GuidedBullet's slot (class-identity-confirmed for v79 via a non-virtual
   `GetMobID` call; slot-index-confirmed for v84) and its mask bit
   (registry 87, fixture-pinned). The trailer length used, 110, is
   **INFERRED by bracketing**: v72 (measured 110), v87 (measured 110), and
   v92 (measured 110) all sit around v79/v84 in the version timeline, all
   three with identical member order — five versions total independently
   measured at 110. This is a known, documented, owner-adjudicated state,
   not a silent pass: `evidence/two-state-group-per-version.md` labels it
   `UNVERIFIED` + `INFERRED`, never `MATCHES`.
3. **Coverage command undercounts.** The checklist's audit command
   (`grep -oE 'CreateContext\("[A-Z]+", [0-9]+' model/character_temporary_stat_test.go`)
   finds only 3 literals because the beacon fixtures are table-driven
   (`TestCTSHomingBeaconPre95PopulatedBlock`, `...AbsentStaysEmpty`,
   `...LegacyVersionsHaveNoTrailer` each iterate a `[]struct{...}` table
   rather than calling `CreateContext` once per version inline). The real
   coverage evidence is the 73 subtest names themselves — `go test ./model/ -run TestCTS -v`
   shows `GMS v61` .. `JMS v185` for every populated/absent pair, plus the
   v95/legacy-specific tests.
4. **Predates three implementation events.** The checklist text was written
   before: the v61 layout fix (Task 7b, commit `be0927c38`), the 5-branch
   per-version movement filter (Task 8, commits `6b72feda4`, `dfbf51d34`,
   `28a046b00` — superseding the 2-branch v83/v95-only draft in the plan's
   Task 8 body), and the 4 re-baselined cancel fixtures (`TestBuffCancelV72/V79/V61/V48ByteFixture`,
   which pinned the pre-fix give-shape cancel mask that Task 8 exists to
   replace). None of these are mentioned in the checklist as originally
   written.

## 3. Scope beyond the original plan — two pre-existing bugs found and fixed

Both were discovered during Task 7/7b/8's IDA verification pass, not
scoped in the PRD, and fixed on this branch because they sit directly on
the beacon's own wire path:

- **v61's two-state trailer was malformed for ALL two-state stats**, not
  just the beacon — mount, dash (speed/jump), and energy charge were all
  affected. Pre-fix, the encoder wrote 7 blocks at v83's sizes
  (15/15/15/13/20/17/15); the v61 client reads 6 smaller blocks
  (14/14/14/12/18/16, no 7th/Undead member) — a field-alignment mismatch
  that would have corrupted every v61 two-state give, not merely produced
  a length error. Fixed by Task 7b's `isGmsV61`-gated block shapes.
- **Buff-cancel packets reused the give-shape mask** (`EncodeMask`'s
  unconditional two-state bits), so ANY cancel — of any single stat —
  cleared every active two-state stat client-side, because the client's
  `TemporaryStatReset` clears every masked bit (v83 `0xA2071F`, v95
  `0x9F2AB0`). This is design.md's "F1" defect: a Speed-potion expiry
  mid-beacon-lock would have silently dropped the lock and the mount/dash
  bits too. Fixed by Task 8's `CancelMask`/`MovementAffectingMask`
  (commit `6b72feda4`), which computes the cancel mask from only the
  stats actually present on the packet.

## 4. Known limitations (stated plainly)

- **Live acceptance not performed.** Task 14 Step 6's 9 scenarios (cast,
  re-cast, whiff, unrelated-buff survival, map-change clear, kill-no-cancel,
  death/logout clear, icon rendering) require a running v83 tenant with an
  Outlaw or Corsair character; no such tenant was available in this
  environment. All 9 remain outstanding — see `plan.md` Task 14 Step 6.
  At least one (icon renders with no duration bar) is only observable live
  and cannot be closed by a byte fixture regardless of environment.
- **JMS v185 evidence provenance.** The only JMS IDB present in this
  environment is `MapleStory_dump_SCY.exe`, not the plan-sanctioned
  `*_U_DEVM` build (none exists here). Owner-accepted with this caveat
  recorded; mitigating evidence: the SCY build is unstripped with full
  mangled C++ symbols, and its shift math independently matches a
  pre-existing IDA-sourced repo comment predating this task. See
  `evidence/two-state-group-per-version.md` §JMS v185 and
  `progress.md` "Task 7: JMS v185".
- **gms_92 has no packet-matrix column** — no `docs/packets/registry/gms_v92.yaml`,
  absent from `STATUS.md` — so it gets fixtures and an evidence record
  (`evidence/two-state-group-per-version.md` §v92,
  `evidence/movement-filter.md` §v92) but no matrix cell to promote. This
  is a pre-existing gap (PRD §2.1/gap 7), not something introduced here.
- **gms_61 is byte-verified only** — no live-acceptance claim is made for
  it (PRD §2.1 caveat: its Pirate WZ data looks pre-release, task-187
  recorded untranslated Korean skill names).
- **Two deferred Minors** (both recorded during code review, neither
  blocking):
  - `isBeaconOnly` (`kafka/consumer/buff/consumer.go`, atlas-channel) is
    unit-tested against a `nil` changes slice but not an explicit empty
    slice literal (`[]buff2.StatChange{}`); the implementation handles
    both identically via the shared `len(changes) == 0` branch, so this is
    a test-coverage gap, not a behavior gap.
  - `BeaconMirror`'s singleton accessor (`character/buff/beacon.go`)
    returns the concrete type `*BeaconMirror` rather than an interface,
    and its tests reset singleton state by writing the package-private
    `beaconMirrorOnce`/`beaconMirror` vars directly rather than through an
    exported test helper — a deviation from `patterns-cache.md`'s
    documented singleton template. The load-bearing guarantees (singleton
    scope, `RWMutex` thread safety) are correctly implemented; only the
    template's specific shape is not followed.

## 5. Version coverage summary (byte-verified vs. live-accepted)

| Version | Two-state trailer | Movement filter | Beacon fixtures | Live acceptance |
|---|---|---|---|---|
| v61 | DIFFERS — 6 members/88B, measured 2x | base-12 (v83 set), registry-verified | pass (106B empty CTS) | not claimed (PRD caveat) |
| v72 | MATCHES v83 — 110B, measured | base-12 | pass | not performed (no live tenant) |
| v79 | 7 members confirmed; **sizes UNVERIFIED**, 110 INFERRED by bracketing | base-12 | pass (uses inferred 110) | not performed |
| v83 | MATCHES — 110B (design-phase verified) | base-12 (reference list) | pass | not performed |
| v84 | 7 members confirmed; **sizes UNVERIFIED**, 110 INFERRED by bracketing | base-12 + 2 unnamed@raw82/83 (strongly inferred Flying/Frozen by cross-version corroboration with v87) | pass (uses inferred 110) | not performed |
| v87 | MATCHES — 110B, measured (all 7 sizes) | base-12 + Flying + Frozen | pass | not performed |
| v92 | MATCHES — 110B, measured (all 7 sizes) | base-12 + Flying + Frozen (owner-ruled extension of the v84 inference); no matrix cell to promote | pass | not performed |
| v95 | MATCHES — conditional 6-member group, 122–127 (design-phase verified) | base-12 + Flying + Frozen + YellowAura | pass (58 unconditional + conditional PartyBooster/GuidedBullet) | not performed |
| JMS v185 | MATCHES v83 — 110B, measured; SCY provenance caveat | own 13-bit list, only 3 stats overlap v83 (Stun, GhostMorph, MonsterRiding); raw bit 126 unmapped in atlas registry, omitted + reported | pass | not performed |
| gms_12 / gms_48 | n/a — `legacyGmsMask` path, 8-byte mask only, no base-stat trailer | n/a | `TestCTSHomingBeaconLegacyVersionsHaveNoTrailer` pass (8 bytes) | n/a (feature explicitly not applicable) |

All nine in-scope versions (PRD §2.1) are byte-verified; none has a
live-acceptance claim. Full detail and IDA addresses are in
`evidence/movement-filter.md`, `evidence/two-state-group-per-version.md`,
and `evidence/v95-two-state-group.md`.
