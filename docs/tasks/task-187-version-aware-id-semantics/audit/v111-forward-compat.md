# v111 forward-compat review (task-187 Task 15) — NOT SHIPPED

**This document records findings only. No code changed. No
`gen/wzsnapshot/gms_111_1.json`, `gen/semantics/gms_111_1.yaml`,
`identities.yaml` entry, or `deploy/k8s/base/versions.json` row was added.
GMS v1.11 (atlas `majorVersion=111`) is explicitly out of the provisioned
set — this task's only deliverable is this assessment plus a bring-up
recipe for whoever does Task 16+ / a future task.**

## Scope and grounding

Two evidence sources, same discipline as `README.md` (Task 1):

1. **IDA `func_query` against the v111 reference IDB**, session
   `9174868f` — confirmed via `idb_list` to be
   `E:\...\GMS\v111\GMS_v111.1_U_DEVM.exe.i64`
   (filename `GMS_v111.1_U_DEVM.exe.i64`), matching the brief's target.
   Cross-referenced against the v95 session `e4abcb98`
   (`GMS_v95.0_U_DEVM.exe.i64`) — the shipped provisioned ceiling.
2. **meymink patch-log release anchors**
   (`https://raw.githubusercontent.com/meymink/Maplestory-Patch-Logs/master/README.md`,
   fetched fresh this session, HTTP 200) — same source Task 1 used, for
   dating which job classes exist by the v1.11 window.

Per project memory (`reference_v111_named_from_v95_bindiff`), the v111 IDB
was named via BinDiff-style alignment **from v95**. This has a direct,
verified consequence for this review (see "IDA naming-coverage limitation"
below): symbols for functions/structs that did not exist in v95 have no
v95 counterpart to align a name from, so post-v95 additions are
**structurally under-named** in this IDB. A `func_query` name-regex miss in
v111 is therefore not proof of absence — I distinguish "confirmed absent
(structural evidence)" from "name-unconfirmable (methodological gap)"
throughout.

## Version-number mapping

Per the README.md precedent: meymink's `0.XX`/`1.XX` client-version numbers
map directly to atlas `majorVersion` (`0.83` → `83`, `1.11` → `111`).
**The meymink log has no standalone `1.11` entry.** Its `<summary>` version
headers jump from `1.09 (Apr 12, 2012)` directly to a combined
`1.12-1.14 (Jul 23, 2012)` entry (verified by grepping every `<summary>`
block in the fetched log — no `1.10` or `1.11` header exists between
them). This is a genuine, reported gap in the anchor source, not an
inference: **v1.11 falls somewhere in the Apr 12 – Jul 23, 2012 window**,
bracketed by those two entries, and no meymink patch-note text is
independently attributable to `1.11` specifically. Everything below dates
jobs by whether their release entry falls before or after that bracket.

## Finding 1: new job classes/branches between v95 and the v111 window

**Confirmed new (released after v0.95, before the v1.11 bracket) per
meymink, entry `1.04 (Dec 08, 2011)`:**

> - New Job: Cannoneer
> - New Job: Mercedes
> - New Job: Demon Slayer

This is the only `New Job:` entry anywhere between the `0.95` anchor
(line ~3595 of the fetched log, per README.md) and the `1.11` bracket.
Every intervening entry (`0.96` through `1.09`, 12 versions) was checked
individually — none mentions a new job class. So relative to v95, **three
new job classes (Cannoneer, Mercedes, Demon Slayer) exist by the time
v1.11 ships**, all added well before the bracket (Dec 2011 vs. the
Apr–Jul 2012 bracket), so there is no ambiguity about whether they're in
scope for v111.

**IDA corroboration attempted, inconclusive by design (not a gap in this
review — the BinDiff-naming limitation above):**
`func_query` against v111 (`database=9174868f`) with
`name_regex` `(?i)(cannoneer|mercedes|demonslayer|demon_slayer)` and
broader single-token regexes (`(?i)cannon`, `(?i)demon`, `(?i)elf`)
returned **zero** matching class-identity symbols (the `demon`/`elf`
regexes only matched unrelated hits — `DecodeMoney`/`OnTradeMoneyLimit`,
`_ZtlSecureGet_bSelfDestruction`/`TryFirstSelfDestruction` mob-AI
functions, `is_self_destruct_summon_skill`, `OnSelfEnter[Result]`). The
same three-class regex against v95 (`database=e4abcb98`) also returned
zero — consistent with the BinDiff-from-v95 mechanism: since these classes
post-date v95, there is no v95 symbol for BinDiff to propagate a name
from into v111, so **name-based search cannot confirm or deny their
presence in either IDB**. The meymink dates carry this finding, not IDA
symbol evidence — recorded explicitly as **IDA: UNVERIFIED (methodological
limit), meymink: CONFIRMED**.

**No further new job classes appear before the v1.11 bracket closes.**
Everything meymink lists between `1.12-1.14 (Jul 23, 2012)` and later —
Phantom + Jett (`1.12-1.14`), Luminous (`1.24`, Dec 4 2012), Kaiser
(`1.25`, Dec 14 2012), Angelic Buster (`1.26`, Jan 8 2013), Mihile
(`1.17`, Aug 31 2012), Hayato (`1.31`, Mar 12 2013), Kanna (`1.29`,
Feb 26 2013), Xenon (`1.38`, Jul 1 2013) — releases **after** the v1.11
bracket, so none of these are in scope at v111. This was cross-checked
structurally for Xenon, Phantom, Luminous, Kaiser, Mihile, Jett: a broad
`func_query` `name_regex` sweep against v111 for all six found **zero**
matches (a real, structural absence in this case, not just a naming gap —
see "Finding 2" for why this is stronger evidence than the Cannoneer/
Mercedes/Demon Slayer case: these classes post-date v111 too, not just
v95, so even the BinDiff-limitation reasoning above doesn't apply the same
way — a class released *after* v111 genuinely won't exist as compiled
code in the v111 binary regardless of naming).

## Finding 2: Wild Hunter is present in BOTH v95 and v111 — and absent from the shipped namespace at either ceiling

This is the most concrete, doubly-IDA-confirmed finding of the review, and
it is **not** a v111-forward-compat gap in the sense the brief expects —
it's a pre-existing gap at the v95 ceiling that persists unchanged at
v111:

- **v95** (`database=e4abcb98`), `name_regex` `(?i)(wildhunter|wild_hunter)`
  returns a rich, fully-named function set: `is_wildhunter_job`,
  `is_wildhunter_jaguar_vehicle` (`0x4066f0`),
  `IsRidingWildHunterJaguar@CAvatar` (`0x45f7c0`), `GW_WildHunterInfo`
  constructor/decode/refcount machinery (`Decode@GW_WildHunterInfo`
  `0x4f2bc0`), `GetWildHunterInfo@CharacterData` (`0x4f9c70`),
  `IsWildhunterJaguarVehicle@SecondaryStat` (`0x7274e0`),
  `CheckOverlapMob`/`GetRidingItem`/`GetRandomCapturedMob@GW_WildHunterInfo`
  (capture-mechanic accessors), and the packet handler
  `OnWildHunterInfo@CWvsContext` (`0x9feda0`).
- **v111** (`database=9174868f`) has the same core set (smaller by count,
  but the identity-defining ones survive): `IsRidingWildHunterJaguar@CAvatar`
  (`0x47d280`), `Decode@GW_WildHunterInfo` (`0x525f10`), `_Alloc@?$ZRef@
  UGW_WildHunterInfo` (`0x52e8f0`), `GetWildHunterInfo@CharacterData`
  (`0x531140`), `OnWildHunterInfo@CWvsContext` (`0xc52ea0`).
- A broader `GW_\w+Info` sweep against v111 found exactly one
  class-specific info struct besides the unrelated `GW_CashItemInfo`:
  `GW_WildHunterInfo`. No `GW_XenonInfo`, `GW_KaiserInfo`,
  `GW_BattleMageInfo`, etc. — consistent with Finding 1's conclusion that
  those classes don't exist yet in the v111 binary.
- **`libs/atlas-constants/gen/identities.yaml` has no `WildHunter` entry at
  all** (grep confirmed zero matches; the full 82-row `domain: job` token
  list tops out at `2218 EvanStage10`, with the Cygnus branches occupying
  `1100-1512` and no Resistance-family branch token present anywhere).

This lines up exactly with a gap Task 5's own generator already documents
(`libs/atlas-constants/gen/availability.go`, `classOf` doc-comment):
**DualBlade, Mechanic, and Resistance have NO identity in the namespace at
all** — `classOf` never returns those labels because no job/skill
canonicalToken maps to them; the whole Resistance job family (Citizen root
→ Demolition Expert → Mechanic/BattleMage/WildHunter/Xenon branches) is
simply not modeled as an `Identity`, even though Mechanic's *release
gating* is tracked via `availability.csv`'s class-label ledger (a level
above `Identity`). Wild Hunter's IDA-confirmed presence at v95 shows this
gap is not new at v111 — it already exists at the shipped ceiling. **v111
doesn't widen this particular gap, it just makes it visible again** when
someone eventually wires v111.

## Finding 3: engine-level job-hierarchy algorithm is unchanged at v111

Cross-referencing the structural (non-identity) finding from `v048-v062.md`
(`is_correct_job_for_skill_root` identical between v48/v72): the same
function still exists, still named, at v111 —
`is_correct_job_for_skill_root` at `0x7cfd10`
(`YAHJJ@Z`, tier-check signature `(job, skillRoot) -> bool`), alongside the
supporting `get_skill_root_from_job` (`0x4e75a0`),
`GetSkillRoot@CSkillInfo` (`0x7cd0b0`), and `get_job_category`/
`get_job_level`/`get_job_change_level` (`0x49eeb0`/`0x49f0a0`/`0xa16500`).
I did not diff the decompiled body against v48/v72/v95 (out of scope for
this pass — this is a corroborating existence check, not a byte-level
diff), so **"unchanged logic" is UNVERIFIED at the instruction level**;
what is confirmed is that the same named entry points still exist,
consistent with (not proof of) an unchanged job/skill-tier matching
algorithm at v111. No existing shipped identity's *meaning* was found to
change — this is a structural continuity signal, not a semantic audit of
every shipped wireId at v111 (see "What this review does NOT cover"
below).

One data-driven observation worth recording for a future bring-up:
`get_job_name` (`0x4e5110`, ~0x248f bytes — the largest function found in
this review) resolves job display names via `StringPool::GetInstance` /
`StringPool::GetString` at runtime, **not** via hardcoded string literals
in the binary. This means IDA alone (without the client's string
resource — `Eng.nx`/`String.wz`) cannot enumerate job display-name text
for a version; it corroborates why this review leans on meymink dates and
struct/function existence rather than trying to read job names directly
out of the binary.

## Finding 4: no skill-id reorg evidence beyond the v92→v95 Big Bang window

The brief also asks whether v111 has "any further skill-id reorg beyond
the v92→v95 Big Bang." **I have no live atlas-data tenant for v111**
(v111 is unprovisioned — no `deploy/k8s/base/versions.json` row, so no
`GET /api/data/jobs` drain is possible the way `bigbang-v092-v095.md` did
it for v92/v95), so a skill-id-set diff the way Task 1 did for Big Bang is
**not attempted here** — doing it would require either (a) a live v111
tenant in the baseline (doesn't exist), or (b) parsing v111's Skill.wz/
Job.wz directly from client asset files, which is out of scope for an IDA
`func_query`-based review. **Marked UNVERIFIED, not attempted** — this is
exactly the kind of drain a future v111 bring-up's Task-3-equivalent step
would need to do (see recipe below).

## Whether the identity namespace design extends to v111

**Yes, structurally — the two-axis design (`Identity` name/token +
per-version `semantics` binding + `availability` release gate) has no
architectural reason it couldn't cover v111. It would need new content,
not new mechanism:**

1. **New `identities.yaml` entries** for the confirmed-new job branches:
   `Cannoneer`, `Mercedes`, `DemonSlayer` (Finding 1) — new
   `canonicalToken` values plus their per-tier skill identities (the same
   pattern `bigbang-v092-v095.md` used for e.g. `EvanStage10`). The exact
   wire job-id/skill-id values for these three are **not derived in this
   review** (no WZ/IDA numeric evidence was pulled for them — only their
   existence-by-date is established); a bring-up task would need a live
   v111 (or nearest-available) WZ drain to assign real tokens, the same
   way Task 1/3 drained gms_92/gms_95 to get Big Bang's numbers.
2. **The pre-existing Resistance-family gap** (Mechanic/BattleMage/
   WildHunter/Xenon/Resistance-root — Finding 2) would also need filling,
   since it's already missing at the v95 ceiling and stays missing at
   v111. This is not new work created by v111 specifically, but a v111
   bring-up is a natural place to finally close it, since WildHunter's
   IDA evidence is now double-confirmed present in both endpoints of the
   range.
3. **No existing shipped identity was found to change meaning at v111.**
   Finding 3's structural check is reassuring but not exhaustive — a real
   bring-up should still re-run the divergence methodology
   (`v048-v062.md`/`bigbang-v092-v095.md`'s approach) against v111, not
   assume continuity from this review alone.

## Explicit non-goals confirmed

- No `gen/wzsnapshot/gms_111_1.json` was created.
- No `gen/semantics/gms_111_1.yaml` was created.
- No availability manifest content for v111 was added (see recipe below —
  the actual mechanism differs slightly from the brief's phrasing).
- No `identities.yaml` entries were added for Cannoneer/Mercedes/
  DemonSlayer/WildHunter/etc.
- No `deploy/k8s/base/versions.json` row was added.
- `libs/atlas-constants/constants/registry_gen.go` was not regenerated.

## What an un-brought-up v111 tenant does today

Confirmed by reading `libs/atlas-constants/constants/for.go`: `For(region,
major, minor)` looks up the tuple in the generated `registry` map; on a
miss (which `(gms, 111, 1)` would be — it is not, and after this task
still is not, a key in `registry_gen.go`) it logs a **once-per-tuple**
warning (`"constants.For: unprovisioned version; using GMS 83.1
baseline"`, deduplicated via the `loggedMisses sync.Map`) and returns the
canonical GMS 83.1 `baseline` `SkillJobSet` instead of panicking or
erroring. So a v111 tenant, if one were ever provisioned on the socket/LB
layer without the identity work below, **degrades gracefully to v83.1
skill/job semantics** rather than crashing — wrong for anything Cannoneer/
Mercedes/Demon-Slayer/Big-Bang-specific, but not a hard failure.

## Recipe for a future v111 bring-up (concrete, sourced from how the shipped versions actually did each step)

This mirrors the real Task 1/3/4/5/6 pipeline, not the brief's slightly
idealized phrasing (per-version YAML files don't exist for availability —
it's a single ledger CSV; noted below where the actual mechanism differs).

1. **Provision a live v111 tenant** in the `atlas-main` (or successor)
   baseline via `atlas-tenants`, the same way gms_92/gms_95 already are —
   needed both for the WZ drain in step 2 and for `deploy/k8s/base/
   versions.json` (step 6) to mean anything operationally.
2. **`gen/wzsnapshot/gms_111_1.json`**: drain `GET /api/data/jobs?
   page[size]=200` from the v111 tenant per `libs/atlas-constants/gen/
   wzsnapshot/PROVENANCE.md`'s documented method (jobs-union: job `id`
   field = job id-set, union of every row's `attributes.skills` = skill
   id-set; note the skill-list endpoint was unavailable for every shipped
   snapshot too, so the jobs-union fallback is the expected path, not an
   exception). Compute the sha256 `hash` field the same way the other 11
   snapshot files do (`snapshots.go`'s `LoadSnapshot` recomputes and
   fails loudly on drift — don't hand-edit the arrays without
   recomputing it).
3. **Divergence audit**: diff the v111 wzsnapshot's job/skill id-sets
   against the nearest shipped neighbor (gms_95) using
   `bigbang-v092-v095.md`'s method (full jobs-union diff, then pull
   `GET /api/data/skills/{id}` name/effects evidence for each affected
   job to classify rename/merge/split/no-counterpart/unverified). Add the
   resulting rows to `docs/tasks/task-187-version-aware-id-semantics/
   audit/divergences.csv` (the single audit ledger every version's
   semantics ultimately derives from — there is no separate per-version
   divergence file).
4. **`gen/semantics/gms_111_1.yaml`**: run `go run . -author-semantics`
   from `libs/atlas-constants/gen` (see that package's `main.go` doc
   comment) — it re-derives every `gen/semantics/<r>_<maj>_<min>.yaml`
   file, including a new one for `gms_111_1`, from the updated
   `divergences.csv` joined against the new wzsnapshot. Do not hand-write
   this file.
5. **New `identities.yaml` entries**: for identities with literally no
   existing token (Cannoneer/Mercedes/DemonSlayer at minimum per Finding
   1; WildHunter/BattleMage/Xenon/Resistance-root if closing Finding 2's
   pre-existing gap in the same pass), hand-add `domain: job` / `domain:
   skill` rows following the existing naming convention (`{Job}Stage{N}`
   for multi-tier branches, flat `{Job}` for single-tier — see the
   Cygnus/Aran/Evan rows already in the file). **Not** via
   `-bootstrap-identities` — that flag overwrites the whole file from the
   const-block scan and drops hand-added entries (per `main.go`'s own
   warning), so any hand-added v111-only identity would need to be
   re-applied after a bootstrap re-run, exactly like the doc already
   warns for the existing Big-Bang-introduced identities.
6. **Availability**: the brief's phrasing ("an availability manifest")
   doesn't match the actual mechanism — there is **no per-version
   `gen/availability/<r>_<maj>_<min>.yaml` file**. Availability is a
   single ledger, `docs/tasks/task-187-version-aware-id-semantics/audit/
   availability.csv`, consumed by `libs/atlas-constants/gen/
   availability.go`'s `classOf`/`computeAvailable` pipeline. A v111
   bring-up adds new rows to that CSV (release-class label × version →
   released bool) for Cannoneer/Mercedes/DemonSlayer (and WildHunter/
   BattleMage/Xenon/Resistance if step 5 added their identities), citing
   the same meymink anchors this review already found (`1.04` for
   Cannoneer/Mercedes/DemonSlayer). No code change is needed for this
   step beyond the CSV row — `availability.go` already reads the ledger
   generically.
7. **`deploy/k8s/base/versions.json`**: append `{ "region": "gms",
   "majorVersion": 111, "minorVersion": 1 }` to the `versions` array, then
   run `tools/gen-lb-ports.sh` per that file's own `$schema`/description
   comment, so the login/channel LoadBalancers expose v111's socket ports
   (same step `bug_new_version_lb_ports` in project memory documents as
   easy to forget).
8. **Registry regen**: run `go run .` (no flags) from
   `libs/atlas-constants/gen` to regenerate `skill/identities_gen.go`,
   `job/identities_gen.go`, the new `skill/version_gms_111_1_gen.go` /
   `job/version_gms_111_1_gen.go` pair, and
   `constants/registry_gen.go` (which is what actually makes
   `constants.For("gms", 111, 1)` stop falling back to baseline). Then
   `go run . -check` to confirm no drift, exactly as CI would.
9. **Re-run the full verification chain** from CLAUDE.md (`go test -race
   ./...`, `go vet ./...`, `go build ./...`, `docker buildx bake
   atlas-<affected-services>`, plus `tools/lint.sh --check`) for
   `libs/atlas-constants` and any service consuming `constants.For`.

## What this review does NOT cover (explicit, not silently dropped)

- No numeric wire job-id/skill-id values were derived for Cannoneer/
  Mercedes/DemonSlayer — only their existence-by-date.
- No skill-id-set diff was run for v111 against v95 (Finding 4) — no live
  v111 tenant exists to drain.
- WildHunter/BattleMage/Xenon's *release date* relative to the v95→v111
  window is not independently pinned by meymink (Wild Hunter has no
  dedicated "released"/"added" patch-note line anywhere in the fetched
  log — only later "revamped" mentions, which presuppose an earlier,
  undated release); its *presence* at both v95 and v111 is IDA-confirmed
  (Finding 2), which is the fact this review actually needed.
- Battle Mage's presence at either v95 or v111 is **UNVERIFIED** — no
  named symbols found in either IDB, and (unlike WildHunter) no
  `GW_BattleMageInfo`-shaped struct either, so unlike the Cannoneer/
  Mercedes/DemonSlayer case this isn't confidently attributable to the
  BinDiff-naming gap; it may simply not exist yet, or may exist under
  unnamed/differently-structured code. Left explicitly UNVERIFIED rather
  than guessed.
- `is_correct_job_for_skill_root`'s decompiled body was not diffed
  instruction-for-instruction against earlier versions at v111 — only
  its continued existence/signature was confirmed.
