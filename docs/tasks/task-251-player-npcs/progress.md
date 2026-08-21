# Task 2 — Cygnus level cap verification

PRD FR-1.2 asserts Cygnus Knights cap at 120; PRD Open Question 5 flags it unverified. This
records the evidence search and its result.

## What was searched

WZ trees (per design §1 C-1, the two stock checkouts on this machine):

- `ms_1172` checkout's `wz/String.wz`, `wz/Character.wz`
- `AtlasMS` checkout's `wz/String.wz`, `wz/Character.wz`

Commands run:

```
grep -rliE 'blockedJobs|maxLevel|levelLimit|LevelLimit|MaxLevel' \
  <tree>/String.wz <tree>/Character.wz
```

— zero matches in either tree. Neither `String.wz` nor `Character.wz` contains a field named
`blockedJobs`, `maxLevel`, or `levelLimit` anywhere in the extracted XML.

`Character.wz` in both trees contains only equip-category folders (`Cap`, `Coat`, `Weapon`, ...)
and a handful of numeric `000xxxxx.img.xml`/`00120xx.img.xml` skin/hair ids — no per-job base
stat or level-cap templates. Job-line level caps, if enforced at all in this era's data, are not
represented as WZ data in these trees; that is consistent with them being server-side logic
rather than client data.

A secondary sweep for `Cygnus` + `level`/`cap`/`120`/`200` across `String.wz/*.img.xml` in
`ms_1172` turned up only narrative/flavor text, not a cap declaration:

- `Consume.img.xml`: a mount item description — "can only be used by Cygnus Lv. 120 for 20
  days" (an item level *requirement*, not evidence of a job level *ceiling*).
- `Skill.img.xml` (Blessing of the Fairy / Cygnus's Blessing tooltips, several duplicate ids):
  "Skill Points increase by 1 when a Cygnus Knight character reaches Lv. 30, 50, 70, 100, and
  120" — a skill-point milestone list that stops enumerating at 120. Suggestive of 120 having
  been a notable threshold for Cygnus characters in this client version, but this is a skill
  tooltip's milestone list, not a max-level/level-cap field. It does not confirm a hard cap.

No `Npc.img.xml`/`Ins.img.xml` hits tying Cygnus to an explicit level ceiling were found.

## IDB (v95, session `ecc757f4`)

Checked by the controller after the fact (the implementing agent had no `mcp__ida-pro__*`
tools). The client has exactly one Cygnus-classification helper,
`is_cygnus_job(long)` at `0x47ca80` — a pure `nJob / 1000 == 1` test with no level term.
All 15 of its call sites were enumerated (`xrefs_to 0x47ca80`):

| caller | address |
|---|---|
| `CLogin::SendDeleteCharPacket` | `0x5d5517` |
| `CSkillInfo::GetShootSkillRange` | `0x7097e2`, `0x709843` |
| `get_weapon_mastery` | `0x709ba3`, `0x709cd5`, `0x709d7a` |
| `get_critical_skill_level` | `0x70a2d8`, `0x70a333` |
| `CUserLocal::TryDoingNormalAttack` | `0x912551` |
| `CUserLocal::TryDoingMeleeAttack` | `0x91ff79`, `0x920152` |
| `CUserLocal::HandleCtrlKeyDown` | `0x932a5d` |
| `CUserLocal::SetDamaged` | `0x935236` |
| `CUserLocal::DoActiveSkill` | `0x947009` |
| `CWvsContext::SendActivatePetRequest` | `0x9f6c1c` |

Every one is a combat, skill-range, mastery, pet or character-delete path. None is a
level-cap comparison. A listing-wide regex sweep for `cygnus|blockedJob|maxLevel|levelLimit`
returned only that helper and `SKILLENTRY::GetMaxLevel` (`0x50a020`), which is a **skill**
level ceiling read from `SKILLENTRY+0x70`, not a character level cap.

Result for this leg: **negative, and now actually checked** — no Cygnus character-level cap is
reachable from the client's only Cygnus-classification helper, and no level-cap symbol or string
surfaced in a listing-wide sweep.

## Result

**Not confirmed.** No local WZ evidence establishes an explicit 120-level cap for
`job.TypeCygnus`; the closest hits are flavor-text milestones stopping at 120, which is a lead,
not a finding. Per the brief, `MaxLevelFor` and its Cygnus test rows are left unchanged (still
200 for all job lines). Both legs of Step 1 have now been searched and both are negative, so this is settled:
no local evidence supports a Cygnus cap below 200.

# Task 8 — `atlas-player-npcs` service scaffold and registration

Service scaffold and registration checklist landed (see task-8-report.md for the full list
of files touched). `tools/service-registration-guard.sh` exits 0, both `deploy/k8s/overlays/pr`
and `deploy/k8s/overlays/main` render clean via `kubectl kustomize`, and `go build ./...` from
`services/atlas-player-npcs/atlas.com/player-npcs` exits 0.

## Operator hand-backs

Two checklist steps this task cannot perform and must be actioned by the operator before the
service can run end-to-end:

1. **§6.1 — create the database.** Create `atlas-player-npcs-main` on `postgres.home` (the same
   Postgres instance/role the other `atlas-*-main` service databases live on). This task only
   registered `atlas-player-npcs` in `tools/db-bootstrap.sh`'s `DBS` list and the `main`/`pr`
   overlay `DB_NAME` patches — none of that creates the database itself on the `main`
   environment's Postgres instance.
2. **§6b — flip the GHCR package public.** After the first `atlas-player-npcs` image is pushed
   to `ghcr.io/chronicle20/atlas-player-npcs/atlas-player-npcs`, the package will be private by
   default (GHCR default for a repo's first push of a new image name). Flip it to public in the
   GitHub Packages UI, matching every other `atlas-*` image, or the cluster's anonymous pull
   will fail.
