# Packet Completeness Critique — task-139-pet-auto-pot-validation

Branch: `task-139-pet-auto-pot-validation`, merge-base `68b26af0a` → HEAD `35fcaf10b`.

**Verdict: 2 findings (1 minor CHANGED-BUT-UNCLAIMED, 1 CLAIMED-BUT-UNVERIFIED /
n-a mismatch). No severe scope holes. The manifest substantively matches the
branch's actual libs/atlas-packet diff and matrix delta.**

## Method

- `claimedPackets` resolved from `coverage-manifest.yaml` `ops`: `PET_AUTO_POT`
  → `pet/serverbound/PetItemUse` (dir `pet/serverbound`); `model/asset` (dir
  `model`); `cash/serverbound/CashItemUsePetSkill` (dir `cash/serverbound`).
- `out_of_scope`: `pet/serverbound/PetCommand|PetChat|PetDropPickUp|PetMovement`.
- Touched `.go` non-test files under `libs/atlas-packet` (`git diff --name-only
  68b26af0a...HEAD -- 'libs/atlas-packet'`):
  `cash/serverbound/item_use_pet_skill.go`, `model/asset.go`,
  `pet/serverbound/item_use.go`, `resolve.go`.
- Version-gate diff (`MajorVersion|MajorAtLeast|IsRegion|Region\(\)`) and the
  `status.json` row-level diff were both walked in full (see below).
- Confirmed at HEAD: `packet-audit matrix --check` exit 0, `fname-doc --check`
  exit 0 (244 fname-less structs, none new/unexplained), `operations --check`
  exit 0 (0 absent-writer notes).

## CHANGED-BUT-UNCLAIMED

| kind | file/packet | evidence | recommendation |
|---|---|---|---|
| codec (minor) | `libs/atlas-packet/resolve.go` (root package `atlas_packet`, no subdir) | `git diff 68b26af0a...HEAD -- libs/atlas-packet/resolve.go` adds `func ResolveCode16(...)` (+44 lines). This file's dir (`libs/atlas-packet` root) is not literally any `claimedPackets` entry (`pet/serverbound`, `model`, `cash/serverbound`) nor `out_of_scope`. | Not a real scope hole — the manifest's `model/asset` `fields` note explicitly names and justifies "the new `atlas_packet.ResolveCode16` helper in `libs/atlas-packet/resolve.go`", and the function carries no version gate itself (the version-dependent behavior lives entirely in `model/asset.go`'s call sites, which ARE claimed). Recommend only a cosmetic fix: add `resolve.go` (or `libs/atlas-packet` root) to `out_of_scope` or its own one-line `ops` note so the critic's directory-match rule doesn't need to fall back to prose next time. |

No other CHANGED-BUT-UNCLAIMED codec or gate hits. Every `MajorVersion|MajorAtLeast|IsRegion|Region()`
diff line attributes cleanly to a claimed file:
- `libs/atlas-packet/model/asset.go`: `(t.IsRegion("GMS") && t.MajorAtLeast(72)) || t.Region() == "JMS"` (remainLife)
  and `(t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS"` (trailing attribute) — both exactly match
  the manifest's `fields` entry for `model/asset` (GMS>=72||JMS, GMS>=79||JMS). Claimed.
- `libs/atlas-packet/pet/serverbound/item_use.go`: calls the pre-existing `hasLeadingPetId(t)` helper (defined in
  `pet/serverbound/legacy.go`, which is untouched this branch — confirmed via
  `git diff --name-only 68b26af0a...HEAD -- libs/atlas-packet/pet/serverbound/` → only `item_use.go`(+test) changed).
  Matches the manifest's claimed gate exactly. `legacy.go`'s own out_of_scope siblings (PetCommand/PetChat/
  PetDropPickUp/PetMovement) were not touched.
- `libs/atlas-packet/pet/serverbound/item_use_test.go` line `(v.Region != "GMS" || v.MajorVersion >= 61)` is
  test-fixture logic, not a codec gate; not in scope for this check.

`status.json` matrix delta (`git diff 68b26af0a...HEAD -- docs/packets/audits/status.json`, row-indexed):
- `PET_AUTO_POT` / `pet/serverbound/PetItemUse` (serverbound): `gms_v48` n-a→verified, `gms_v61/v72/v79`
  partial→verified. Claimed (`PET_AUTO_POT` is in `ops`).
- New row `cash/serverbound/CashItemUsePetSkill` (jms_v185: verified; 8 GMS cells: incomplete/"no audit report").
  Claimed (`cash/serverbound/CashItemUsePetSkill` is in `ops`) — see CLAIMED-BUT-UNVERIFIED below for the nuance.

No other row in `status.json` changed state (`gms_v83/v84/v87/v95`/`jms_v185` PET_AUTO_POT cells were already
`verified` pre-branch — confirmed by reading the base-commit `status.json`, unchanged in the diff — matching the
manifest's claim that these five were "already verified pre-branch and are asserted byte-identical").

## CLAIMED-BUT-UNVERIFIED

| op | version | actual state | recommendation |
|---|---|---|---|
| `cash/serverbound/CashItemUsePetSkill` | gms_v48, gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v95 | `incomplete`, note `"no audit report"`, `opcode: -1` (HEAD `status.json`) | The manifest's `fields` note says this struct is "jms_v185 only — sender does not exist pre-v87 (task-8 finding)," i.e. every GMS version is claimed as inapplicable by the task's own narrative. But the manifest's `versions` list is task-wide (all 9) with no per-op scoping in the schema, so it literally claims all 9 versions for this op, and the matrix records the 8 GMS cells as `incomplete` (unaudited default), not `n-a`. Per PROCESS.md's n-a-consistency expectation, a cell the task itself says is inapplicable should read `n-a`, not sit at the `incomplete` default. Recommend either (a) run `packet-audit na-consistency`-style promotion to mark the 8 GMS cells `n-a` with the "sender doesn't exist pre-v87" justification wired into `docs/packets/feature-na-evidence.yaml` (the same mechanism already used for `USE_TELEPORT_ROCK × gms_v48`, seen in this repo's `matrix --check` output), or (b) narrow this op's applicability in the manifest text/schema so it isn't read as an unmet claim across 8 versions it was never meant to cover. |

Also flagged for narrative accuracy, not machine-checkable: the manifest's comment on `cash/serverbound/CashItemUsePetSkill` says "no separate matrix row (rolls into the shared ... dispatcher)" — but `status.json` HEAD shows this struct DOES have its own row (`packet: cash/serverbound/CashItemUsePetSkill`, no `op`), separate from any dispatcher row. The row exists and is correctly the vehicle carrying the jms_v185 `verified` cell; the manifest prose is simply stale on this one detail and should say "own row, jms_v185-only in scope" instead of "no separate matrix row."

`PET_AUTO_POT` × all 9 versions: all `verified` at HEAD — no unverified claim.

`model/asset` is a shared sub-encoder with no independent `status.json` row (per manifest, correctly) — its
coverage rides on `PET_AUTO_POT`'s serverbound verification plus the byte-fixture assertions in
`model/asset_test.go`; no separate claim to check here.

## Confirmed checks (re-run at HEAD, not assumed)

```
packet-audit matrix --check      → exit 0 (note: USE_TELEPORT_ROCK × gms_v48 n-a evidence consumed)
packet-audit fname-doc --check   → exit 0 (244 structs without audit report carry no fname)
packet-audit operations --check  → exit 0 (0 absent-writer notes)
```
